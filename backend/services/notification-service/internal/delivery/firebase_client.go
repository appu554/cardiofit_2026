package delivery

import (
	"context"
	"fmt"
	"time"

	"github.com/cardiofit/notification-service/internal/models"
	"go.uber.org/zap"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FirebaseClient manages Firebase Cloud Messaging for push notifications
type FirebaseClient struct {
	app     *firebase.App
	client  *messaging.Client
	logger  *zap.Logger
	context context.Context
}

// NewFirebaseClient creates a new Firebase client
func NewFirebaseClient(credentialsPath string, logger *zap.Logger) (*FirebaseClient, error) {
	if credentialsPath == "" {
		return nil, fmt.Errorf("firebase credentials path not configured")
	}

	ctx := context.Background()

	// Initialize Firebase app with credentials
	opt := option.WithCredentialsFile(credentialsPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	// Get messaging client
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get firebase messaging client: %w", err)
	}

	logger.Info("Firebase client initialized successfully",
		zap.String("credentials_path", credentialsPath),
	)

	return &FirebaseClient{
		app:     app,
		client:  client,
		logger:  logger,
		context: ctx,
	}, nil
}

// SendPush sends a push notification via Firebase Cloud Messaging
func (f *FirebaseClient) SendPush(ctx context.Context, fcmToken, title, body string, data map[string]string) (messageID string, err error) {
	if f.client == nil {
		return "", fmt.Errorf("firebase client not initialized")
	}

	if fcmToken == "" {
		return "", fmt.Errorf("FCM token is required")
	}

	if title == "" {
		return "", fmt.Errorf("notification title is required")
	}

	if body == "" {
		return "", fmt.Errorf("notification body is required")
	}

	// Build notification message
	message := &messaging.Message{
		Token: fcmToken,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "clinical_alerts",
				Priority:  messaging.PriorityHigh,
				Sound:     "alert_sound",
				Color:     "#dc3545", // Red for critical alerts
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority": "10", // Immediate delivery
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: title,
						Body:  body,
					},
					Badge:            getBadgePointer(1),
					Sound:            "alert.aiff",
					ContentAvailable: true,
					Category:         "CLINICAL_ALERT",
				},
			},
		},
	}

	// Send message
	f.logger.Debug("Sending push notification via Firebase",
		zap.String("title", title),
		zap.Int("body_length", len(body)),
		zap.Int("data_fields", len(data)),
	)

	startTime := time.Now()
	response, err := f.client.Send(ctx, message)
	latency := time.Since(startTime)

	if err != nil {
		f.logger.Error("Firebase push notification failed",
			zap.Error(err),
			zap.Duration("latency", latency),
		)
		return "", fmt.Errorf("firebase send failed: %w", err)
	}

	f.logger.Info("Push notification sent successfully via Firebase",
		zap.String("message_id", response),
		zap.Duration("latency", latency),
	)

	return response, nil
}

// SendMulticast sends push notifications to multiple devices
func (f *FirebaseClient) SendMulticast(ctx context.Context, fcmTokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error) {
	if f.client == nil {
		return nil, fmt.Errorf("firebase client not initialized")
	}

	if len(fcmTokens) == 0 {
		return nil, fmt.Errorf("at least one FCM token is required")
	}

	if len(fcmTokens) > 500 {
		return nil, fmt.Errorf("maximum 500 tokens allowed per batch")
	}

	// Build multicast message
	message := &messaging.MulticastMessage{
		Tokens: fcmTokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "clinical_alerts",
				Priority:  messaging.PriorityHigh,
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "alert.aiff",
					Badge: getBadgePointer(1),
				},
			},
		},
	}

	f.logger.Debug("Sending multicast push notification",
		zap.Int("recipient_count", len(fcmTokens)),
		zap.String("title", title),
	)

	startTime := time.Now()
	response, err := f.client.SendMulticast(ctx, message)
	latency := time.Since(startTime)

	if err != nil {
		f.logger.Error("Firebase multicast failed",
			zap.Error(err),
			zap.Duration("latency", latency),
		)
		return nil, fmt.Errorf("firebase multicast failed: %w", err)
	}

	f.logger.Info("Multicast push notification sent",
		zap.Int("success_count", response.SuccessCount),
		zap.Int("failure_count", response.FailureCount),
		zap.Duration("latency", latency),
	)

	return response, nil
}

// BuildNotificationPayload builds a notification payload for clinical alerts
func (f *FirebaseClient) BuildNotificationPayload(alert *models.Alert) *messaging.Message {
	title := fmt.Sprintf("%s Alert", alert.Severity)
	body := fmt.Sprintf("Patient %s: %s", alert.PatientID, alert.Message)

	// Build data payload with deep link
	data := map[string]string{
		"alert_id":    alert.AlertID,
		"patient_id":  alert.PatientID,
		"alert_type":  string(alert.AlertType),
		"severity":    string(alert.Severity),
		"risk_score":  fmt.Sprintf("%.2f", alert.RiskScore),
		"confidence":  fmt.Sprintf("%.2f", alert.Confidence),
		"timestamp":   fmt.Sprintf("%d", alert.Timestamp),
		"deep_link":   fmt.Sprintf("cardiofit://patient/%s/alert/%s", alert.PatientID, alert.AlertID),
		"room":        alert.PatientLocation.Room,
		"bed":         alert.PatientLocation.Bed,
		"hospital_id": alert.HospitalID,
	}

	// Add vital signs if available
	if alert.VitalSigns != nil {
		data["heart_rate"] = fmt.Sprintf("%d", alert.VitalSigns.HeartRate)
		data["blood_pressure"] = fmt.Sprintf("%d/%d",
			alert.VitalSigns.BloodPressureSystolic,
			alert.VitalSigns.BloodPressureDiastolic)
		data["temperature"] = fmt.Sprintf("%.1f", alert.VitalSigns.Temperature)
		if alert.VitalSigns.OxygenSaturation > 0 {
			data["spo2"] = fmt.Sprintf("%d", alert.VitalSigns.OxygenSaturation)
		}
	}

	// Determine priority based on severity
	priority := f.getPriorityForSeverity(alert.Severity)

	return &messaging.Message{
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
			ImageURL: f.getAlertIconURL(alert.Severity),
		},
		Data: data,
		Android: &messaging.AndroidConfig{
			Priority: priority,
			Notification: &messaging.AndroidNotification{
				ChannelID:    "clinical_alerts",
				Sound:        f.getSoundForSeverity(alert.Severity),
				Color:        f.getColorForSeverity(alert.Severity),
				Tag:          fmt.Sprintf("alert_%s", alert.AlertID),
				ClickAction:  "OPEN_ALERT",
			},
		},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-priority":   f.getAPNSPriority(alert.Severity),
				"apns-expiration": fmt.Sprintf("%d", time.Now().Add(1*time.Hour).Unix()),
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: title,
						Body:  body,
					},
					Badge:            getBadgePointer(1),
					Sound:            f.getSoundForSeverity(alert.Severity),
					ContentAvailable: true,
					Category:         "CLINICAL_ALERT",
					ThreadID:         alert.PatientID,
				},
			},
		},
	}
}

// getPriorityForSeverity returns Firebase priority for alert severity
func (f *FirebaseClient) getPriorityForSeverity(severity models.AlertSeverity) string {
	switch severity {
	case models.SeverityCritical, models.SeverityHigh:
		return "high"
	default:
		return "normal"
	}
}

// getAndroidPriority returns Android notification priority
func (f *FirebaseClient) getAndroidPriority(severity models.AlertSeverity) string {
	switch severity {
	case models.SeverityCritical:
		return "max"
	case models.SeverityHigh:
		return "high"
	case models.SeverityModerate:
		return "default"
	default:
		return "low"
	}
}

// getAPNSPriority returns APNS priority string
func (f *FirebaseClient) getAPNSPriority(severity models.AlertSeverity) string {
	switch severity {
	case models.SeverityCritical, models.SeverityHigh:
		return "10" // Immediate delivery
	default:
		return "5" // Power-efficient delivery
	}
}

// getSoundForSeverity returns appropriate notification sound
func (f *FirebaseClient) getSoundForSeverity(severity models.AlertSeverity) string {
	switch severity {
	case models.SeverityCritical:
		return "critical_alert.wav"
	case models.SeverityHigh:
		return "high_alert.wav"
	case models.SeverityModerate:
		return "moderate_alert.wav"
	default:
		return "default.wav"
	}
}

// getColorForSeverity returns color code for severity
func (f *FirebaseClient) getColorForSeverity(severity models.AlertSeverity) string {
	switch severity {
	case models.SeverityCritical:
		return "#dc3545" // Red
	case models.SeverityHigh:
		return "#fd7e14" // Orange
	case models.SeverityModerate:
		return "#ffc107" // Yellow
	case models.SeverityLow:
		return "#28a745" // Green
	default:
		return "#6c757d" // Gray
	}
}

// getAlertIconURL returns icon URL for alert severity
func (f *FirebaseClient) getAlertIconURL(severity models.AlertSeverity) string {
	baseURL := "https://cardiofit.app/assets/icons"
	switch severity {
	case models.SeverityCritical:
		return fmt.Sprintf("%s/critical_alert.png", baseURL)
	case models.SeverityHigh:
		return fmt.Sprintf("%s/high_alert.png", baseURL)
	case models.SeverityModerate:
		return fmt.Sprintf("%s/moderate_alert.png", baseURL)
	default:
		return fmt.Sprintf("%s/info_alert.png", baseURL)
	}
}

// ValidateToken validates if an FCM token is still valid
func (f *FirebaseClient) ValidateToken(ctx context.Context, fcmToken string) (bool, error) {
	if f.client == nil {
		return false, fmt.Errorf("firebase client not initialized")
	}

	// Try sending a dry-run message
	message := &messaging.Message{
		Token: fcmToken,
		Data: map[string]string{
			"test": "validation",
		},
	}

	// Use dry-run mode (doesn't actually send)
	_, err := f.client.SendDryRun(ctx, message)
	if err != nil {
		// Token is invalid or expired
		f.logger.Debug("FCM token validation failed",
			zap.Error(err),
		)
		return false, nil
	}

	return true, nil
}

// SubscribeToTopic subscribes tokens to a topic for group messaging
func (f *FirebaseClient) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	if f.client == nil {
		return fmt.Errorf("firebase client not initialized")
	}

	if len(tokens) == 0 {
		return fmt.Errorf("at least one token is required")
	}

	response, err := f.client.SubscribeToTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	f.logger.Info("Subscribed tokens to topic",
		zap.String("topic", topic),
		zap.Int("success_count", response.SuccessCount),
		zap.Int("failure_count", response.FailureCount),
	)

	return nil
}

// UnsubscribeFromTopic unsubscribes tokens from a topic
func (f *FirebaseClient) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	if f.client == nil {
		return fmt.Errorf("firebase client not initialized")
	}

	response, err := f.client.UnsubscribeFromTopic(ctx, tokens, topic)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe from topic: %w", err)
	}

	f.logger.Info("Unsubscribed tokens from topic",
		zap.String("topic", topic),
		zap.Int("success_count", response.SuccessCount),
		zap.Int("failure_count", response.FailureCount),
	)

	return nil
}

// SendToTopic sends a notification to all devices subscribed to a topic
func (f *FirebaseClient) SendToTopic(ctx context.Context, topic, title, body string, data map[string]string) (string, error) {
	if f.client == nil {
		return "", fmt.Errorf("firebase client not initialized")
	}

	message := &messaging.Message{
		Topic: topic,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	response, err := f.client.Send(ctx, message)
	if err != nil {
		return "", fmt.Errorf("failed to send to topic: %w", err)
	}

	f.logger.Info("Sent notification to topic",
		zap.String("topic", topic),
		zap.String("message_id", response),
	)

	return response, nil
}

// Close cleans up Firebase client resources
func (f *FirebaseClient) Close() error {
	f.logger.Info("Closing Firebase client")
	// Firebase client doesn't require explicit cleanup
	return nil
}

// Helper functions for pointer types

func getBadgePointer(badge int) *int {
	return &badge
}

func getDurationPointer(seconds int64) *time.Duration {
	duration := time.Duration(seconds) * time.Second
	return &duration
}
