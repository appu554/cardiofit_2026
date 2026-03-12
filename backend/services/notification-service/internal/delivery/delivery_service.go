package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// DeliveryConfig holds delivery service configuration
type DeliveryConfig struct {
	Workers             int
	RetryMaxAttempts    int
	RetryBackoffSeconds int
	TimeoutSeconds      int
}

// RetryPolicy defines retry behavior with exponential backoff
type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
}

// DefaultRetryPolicy returns the default retry configuration
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
	}
}

// NotificationDeliveryService manages multi-channel notification delivery
type NotificationDeliveryService struct {
	twilioClient   *TwilioClient
	sendgridClient *SendGridClient
	firebaseClient *FirebaseClient
	db             *pgxpool.Pool
	logger         *zap.Logger
	config         DeliveryConfig
	workers        int
	retryPolicy    RetryPolicy
	workerPool     chan struct{}
	metricsCollector *MetricsCollector
	mu             sync.RWMutex
}

// MetricsCollector tracks delivery metrics
type MetricsCollector struct {
	mu sync.Mutex
	channelMetrics map[models.NotificationChannel]*ChannelMetrics
}

// ChannelMetrics tracks metrics per channel
type ChannelMetrics struct {
	TotalAttempts int64
	Successful    int64
	Failed        int64
	TotalLatency  int64 // milliseconds
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		channelMetrics: make(map[models.NotificationChannel]*ChannelMetrics),
	}
}

// RecordAttempt records a delivery attempt with result and latency
func (mc *MetricsCollector) RecordAttempt(channel models.NotificationChannel, success bool, latencyMs int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.channelMetrics[channel]; !exists {
		mc.channelMetrics[channel] = &ChannelMetrics{}
	}

	metrics := mc.channelMetrics[channel]
	metrics.TotalAttempts++
	metrics.TotalLatency += latencyMs

	if success {
		metrics.Successful++
	} else {
		metrics.Failed++
	}
}

// GetMetrics returns a copy of current metrics
func (mc *MetricsCollector) GetMetrics(channel models.NotificationChannel) *ChannelMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if metrics, exists := mc.channelMetrics[channel]; exists {
		// Return copy to avoid race conditions
		return &ChannelMetrics{
			TotalAttempts: metrics.TotalAttempts,
			Successful:    metrics.Successful,
			Failed:        metrics.Failed,
			TotalLatency:  metrics.TotalLatency,
		}
	}
	return &ChannelMetrics{}
}

// NewNotificationDeliveryService creates a new delivery service
func NewNotificationDeliveryService(
	cfg config.Config,
	db *pgxpool.Pool,
	logger *zap.Logger,
) (*NotificationDeliveryService, error) {
	deliveryConfig := DeliveryConfig{
		Workers:             10,
		RetryMaxAttempts:    3,
		RetryBackoffSeconds: 1,
		TimeoutSeconds:      30,
	}

	// Initialize Twilio client
	twilioClient := NewTwilioClient(
		cfg.Delivery.SMS.TwilioSID,
		cfg.Delivery.SMS.TwilioToken,
		cfg.Delivery.SMS.TwilioFromNumber,
		logger,
	)

	// Initialize SendGrid client
	sendgridClient := NewSendGridClient(
		cfg.Delivery.Email.SendGridAPIKey,
		cfg.Delivery.Email.FromEmail,
		logger,
	)

	// Initialize Firebase client
	firebaseClient, err := NewFirebaseClient(
		cfg.Delivery.Push.FirebaseCredentials,
		logger,
	)
	if err != nil {
		logger.Warn("Failed to initialize Firebase client, push notifications will be disabled", zap.Error(err))
		// Continue without Firebase - it's optional
	}

	service := &NotificationDeliveryService{
		twilioClient:     twilioClient,
		sendgridClient:   sendgridClient,
		firebaseClient:   firebaseClient,
		db:               db,
		logger:           logger,
		config:           deliveryConfig,
		workers:          deliveryConfig.Workers,
		retryPolicy:      DefaultRetryPolicy(),
		workerPool:       make(chan struct{}, deliveryConfig.Workers),
		metricsCollector: NewMetricsCollector(),
	}

	return service, nil
}

// Send sends a notification through the appropriate channel
func (d *NotificationDeliveryService) Send(ctx context.Context, notification *models.Notification) error {
	if notification == nil {
		return fmt.Errorf("notification cannot be nil")
	}

	if notification.User == nil || notification.Alert == nil {
		return fmt.Errorf("notification must have user and alert data")
	}

	// Acquire worker from pool
	select {
	case d.workerPool <- struct{}{}:
		defer func() { <-d.workerPool }()
	case <-ctx.Done():
		return ctx.Err()
	}

	// Send with retry logic
	return d.sendWithRetry(ctx, notification)
}

// sendWithRetry implements exponential backoff retry logic
func (d *NotificationDeliveryService) sendWithRetry(ctx context.Context, notification *models.Notification) error {
	var lastErr error

	for attempt := 0; attempt < d.retryPolicy.MaxAttempts; attempt++ {
		// Update retry count
		notification.RetryCount = attempt

		// Calculate backoff duration
		backoff := d.calculateBackoff(attempt)

		// Wait for backoff (except on first attempt)
		if attempt > 0 {
			d.logger.Info("Retrying notification delivery",
				zap.String("notification_id", notification.ID),
				zap.String("channel", string(notification.Channel)),
				zap.Int("attempt", attempt+1),
				zap.Duration("backoff", backoff),
			)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Update status to SENDING
		if err := d.updateDeliveryStatus(ctx, notification.ID, string(models.StatusSending), "", ""); err != nil {
			d.logger.Warn("Failed to update status to SENDING", zap.Error(err))
		}

		// Attempt delivery
		startTime := time.Now()
		err := d.deliverToChannel(ctx, notification)
		latencyMs := time.Since(startTime).Milliseconds()

		// Record metrics
		d.recordMetrics(notification.Channel, err == nil, latencyMs)

		if err == nil {
			// Success - update status and return
			now := time.Now()
			notification.SentAt = &now
			notification.Status = models.StatusSent

			updateErr := d.updateDeliveryStatus(
				ctx,
				notification.ID,
				string(models.StatusSent),
				notification.ExternalID,
				"",
			)
			if updateErr != nil {
				d.logger.Error("Failed to update delivery status",
					zap.String("notification_id", notification.ID),
					zap.Error(updateErr),
				)
			}

			d.logger.Info("Notification delivered successfully",
				zap.String("notification_id", notification.ID),
				zap.String("channel", string(notification.Channel)),
				zap.String("user_id", notification.UserID),
				zap.Int64("latency_ms", latencyMs),
			)

			return nil
		}

		lastErr = err
		d.logger.Warn("Notification delivery attempt failed",
			zap.String("notification_id", notification.ID),
			zap.String("channel", string(notification.Channel)),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}

	// All retries exhausted - mark as failed
	notification.Status = models.StatusFailed
	notification.ErrorMessage = lastErr.Error()

	updateErr := d.updateDeliveryStatus(
		ctx,
		notification.ID,
		string(models.StatusFailed),
		"",
		lastErr.Error(),
	)
	if updateErr != nil {
		d.logger.Error("Failed to update failure status",
			zap.String("notification_id", notification.ID),
			zap.Error(updateErr),
		)
	}

	return fmt.Errorf("delivery failed after %d attempts: %w", d.retryPolicy.MaxAttempts, lastErr)
}

// deliverToChannel routes to the appropriate channel handler
func (d *NotificationDeliveryService) deliverToChannel(ctx context.Context, notification *models.Notification) error {
	switch notification.Channel {
	case models.ChannelSMS:
		return d.sendSMS(ctx, notification)
	case models.ChannelEmail:
		return d.sendEmail(ctx, notification)
	case models.ChannelPush:
		return d.sendPush(ctx, notification)
	case models.ChannelVoice:
		return d.sendVoice(ctx, notification)
	case models.ChannelInApp:
		return d.sendInApp(ctx, notification)
	case models.ChannelPager:
		return d.sendPager(ctx, notification)
	default:
		return fmt.Errorf("unsupported channel: %s", notification.Channel)
	}
}

// sendSMS sends SMS notification via Twilio
func (d *NotificationDeliveryService) sendSMS(ctx context.Context, notification *models.Notification) error {
	if d.twilioClient == nil {
		return fmt.Errorf("twilio client not initialized")
	}

	phoneNumber := notification.User.PhoneNumber
	if phoneNumber == "" {
		return fmt.Errorf("user has no phone number configured")
	}

	messageID, err := d.twilioClient.SendSMS(ctx, phoneNumber, notification.Message)
	if err != nil {
		return fmt.Errorf("twilio SMS failed: %w", err)
	}

	notification.ExternalID = messageID
	return nil
}

// sendEmail sends email notification via SendGrid
func (d *NotificationDeliveryService) sendEmail(ctx context.Context, notification *models.Notification) error {
	if d.sendgridClient == nil {
		return fmt.Errorf("sendgrid client not initialized")
	}

	email := notification.User.Email
	if email == "" {
		return fmt.Errorf("user has no email configured")
	}

	// Build subject and HTML body
	subject := fmt.Sprintf("Clinical Alert: %s", notification.Alert.AlertType)
	htmlBody := d.sendgridClient.BuildAlertEmailHTML(notification.Alert, notification.User)

	messageID, err := d.sendgridClient.SendEmail(ctx, email, subject, htmlBody)
	if err != nil {
		return fmt.Errorf("sendgrid email failed: %w", err)
	}

	notification.ExternalID = messageID
	return nil
}

// sendPush sends push notification via Firebase
func (d *NotificationDeliveryService) sendPush(ctx context.Context, notification *models.Notification) error {
	if d.firebaseClient == nil {
		return fmt.Errorf("firebase client not initialized")
	}

	fcmToken := notification.User.FCMToken
	if fcmToken == "" {
		return fmt.Errorf("user has no FCM token configured")
	}

	title := fmt.Sprintf("%s Alert", notification.Alert.Severity)

	// Build data payload with deep link
	data := map[string]string{
		"alert_id":   notification.Alert.AlertID,
		"patient_id": notification.Alert.PatientID,
		"severity":   string(notification.Alert.Severity),
		"deep_link":  fmt.Sprintf("cardiofit://patient/%s/alert/%s", notification.Alert.PatientID, notification.Alert.AlertID),
	}

	messageID, err := d.firebaseClient.SendPush(ctx, fcmToken, title, notification.Message, data)
	if err != nil {
		return fmt.Errorf("firebase push failed: %w", err)
	}

	notification.ExternalID = messageID
	return nil
}

// sendVoice initiates voice call via Twilio
func (d *NotificationDeliveryService) sendVoice(ctx context.Context, notification *models.Notification) error {
	if d.twilioClient == nil {
		return fmt.Errorf("twilio client not initialized")
	}

	phoneNumber := notification.User.PhoneNumber
	if phoneNumber == "" {
		return fmt.Errorf("user has no phone number configured")
	}

	// Voice message for critical alerts
	voiceMessage := fmt.Sprintf(
		"Critical clinical alert. Patient %s. %s. Alert type: %s. Please acknowledge immediately.",
		notification.Alert.PatientID,
		notification.Alert.Message,
		notification.Alert.AlertType,
	)

	callID, err := d.twilioClient.InitiateCall(ctx, phoneNumber, voiceMessage)
	if err != nil {
		return fmt.Errorf("twilio voice call failed: %w", err)
	}

	notification.ExternalID = callID
	return nil
}

// sendInApp stores in-app notification in database
func (d *NotificationDeliveryService) sendInApp(ctx context.Context, notification *models.Notification) error {
	// In-app notifications are already in the database, just mark as delivered
	now := time.Now()
	notification.SentAt = &now
	notification.DeliveredAt = &now

	return d.updateDeliveryStatus(
		ctx,
		notification.ID,
		string(models.StatusDelivered),
		"",
		"",
	)
}

// sendPager sends pager notification (stub for now)
func (d *NotificationDeliveryService) sendPager(ctx context.Context, notification *models.Notification) error {
	// PagerDuty integration can be implemented here
	// For now, we'll stub it and log
	d.logger.Info("Pager notification requested (not yet implemented)",
		zap.String("notification_id", notification.ID),
		zap.String("user_id", notification.UserID),
		zap.String("pager_number", notification.User.PagerNumber),
	)

	// Stub: mark as sent
	return nil
}

// updateDeliveryStatus updates notification status in database
func (d *NotificationDeliveryService) updateDeliveryStatus(
	ctx context.Context,
	notificationID string,
	status string,
	externalID string,
	errorMessage string,
) error {
	query := `
		UPDATE notification_service.notifications
		SET status = $1,
		    external_id = CASE WHEN $2 != '' THEN $2 ELSE external_id END,
		    error_message = CASE WHEN $3 != '' THEN $3 ELSE error_message END,
		    sent_at = CASE WHEN $1 = 'SENT' THEN NOW() ELSE sent_at END,
		    delivered_at = CASE WHEN $1 = 'DELIVERED' THEN NOW() ELSE delivered_at END
		WHERE id = $4
	`

	_, err := d.db.Exec(ctx, query, status, externalID, errorMessage, notificationID)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	return nil
}

// recordMetrics records delivery metrics for monitoring
func (d *NotificationDeliveryService) recordMetrics(
	channel models.NotificationChannel,
	success bool,
	latencyMs int64,
) {
	d.metricsCollector.RecordAttempt(channel, success, latencyMs)

	// Log metrics periodically (can be extended to push to Prometheus/Grafana)
	if success {
		d.logger.Debug("Delivery metrics recorded",
			zap.String("channel", string(channel)),
			zap.Bool("success", success),
			zap.Int64("latency_ms", latencyMs),
		)
	}
}

// calculateBackoff calculates exponential backoff duration
func (d *NotificationDeliveryService) calculateBackoff(attempt int) time.Duration {
	backoff := float64(d.retryPolicy.InitialBackoff) * math.Pow(d.retryPolicy.Multiplier, float64(attempt))
	backoffDuration := time.Duration(backoff)

	if backoffDuration > d.retryPolicy.MaxBackoff {
		return d.retryPolicy.MaxBackoff
	}

	return backoffDuration
}

// SendBatch sends multiple notifications concurrently
func (d *NotificationDeliveryService) SendBatch(ctx context.Context, notifications []*models.Notification) []error {
	var wg sync.WaitGroup
	errors := make([]error, len(notifications))

	for i, notification := range notifications {
		wg.Add(1)
		go func(idx int, notif *models.Notification) {
			defer wg.Done()
			errors[idx] = d.Send(ctx, notif)
		}(i, notification)
	}

	wg.Wait()
	return errors
}

// GetDeliveryStatus retrieves current delivery status from database
func (d *NotificationDeliveryService) GetDeliveryStatus(ctx context.Context, notificationID string) (*models.Notification, error) {
	query := `
		SELECT id, alert_id, user_id, channel, priority, message, status,
		       retry_count, external_id, created_at, sent_at, delivered_at,
		       acknowledged_at, error_message, metadata
		FROM notification_service.notifications
		WHERE id = $1
	`

	notification := &models.Notification{}
	var metadataJSON []byte

	err := d.db.QueryRow(ctx, query, notificationID).Scan(
		&notification.ID,
		&notification.AlertID,
		&notification.UserID,
		&notification.Channel,
		&notification.Priority,
		&notification.Message,
		&notification.Status,
		&notification.RetryCount,
		&notification.ExternalID,
		&notification.CreatedAt,
		&notification.SentAt,
		&notification.DeliveredAt,
		&notification.AcknowledgedAt,
		&notification.ErrorMessage,
		&metadataJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to query notification: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &notification.Metadata); err != nil {
			d.logger.Warn("Failed to unmarshal metadata", zap.Error(err))
		}
	}

	return notification, nil
}

// GetChannelMetrics returns metrics for a specific channel
func (d *NotificationDeliveryService) GetChannelMetrics(channel models.NotificationChannel) *ChannelMetrics {
	return d.metricsCollector.GetMetrics(channel)
}

// Shutdown gracefully shuts down the delivery service
func (d *NotificationDeliveryService) Shutdown(ctx context.Context) error {
	d.logger.Info("Shutting down notification delivery service")

	// Wait for all workers to finish (with timeout)
	done := make(chan struct{})
	go func() {
		for i := 0; i < d.workers; i++ {
			d.workerPool <- struct{}{}
		}
		close(done)
	}()

	select {
	case <-done:
		d.logger.Info("All workers completed")
	case <-ctx.Done():
		d.logger.Warn("Shutdown timeout reached, forcing shutdown")
	}

	return nil
}
