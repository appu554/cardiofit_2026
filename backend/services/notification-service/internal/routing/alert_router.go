package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"

	"github.com/cardiofit/notification-service/internal/models"
)

// AlertRouter is responsible for routing alerts to appropriate users and channels
type AlertRouter struct {
	fatigueTracker  AlertFatigueTracker
	userService     UserPreferenceService
	deliveryService NotificationDeliveryService
	escalationMgr   EscalationManager
	logger          *zap.Logger
	metrics         *RouterMetrics
}

// AlertFatigueTracker interface for checking alert fatigue
type AlertFatigueTracker interface {
	ShouldSuppress(alert *models.Alert, user *models.User) (bool, string)
	RecordNotification(userID string, alert *models.Alert)
}

// UserPreferenceService interface for user management
type UserPreferenceService interface {
	GetAttendingPhysician(departmentID string) ([]*models.User, error)
	GetChargeNurse(departmentID string) ([]*models.User, error)
	GetPrimaryNurse(patientID string) ([]*models.User, error)
	GetResident(departmentID string) ([]*models.User, error)
	GetClinicalInformaticsTeam() ([]*models.User, error)
	GetPreferredChannels(user *models.User, severity models.AlertSeverity) []models.NotificationChannel
}

// NotificationDeliveryService interface for sending notifications
type NotificationDeliveryService interface {
	Send(ctx context.Context, notification *models.Notification) error
}

// EscalationManager interface for managing escalations
type EscalationManager interface {
	ScheduleEscalation(ctx context.Context, alert *models.Alert, timeout time.Duration) error
}

// RouterMetrics contains Prometheus metrics for the router
type RouterMetrics struct {
	alertsRoutedTotal    *prometheus.CounterVec
	routingDuration      prometheus.Histogram
	usersTargetedTotal   prometheus.Counter
	alertsSuppressedTotal *prometheus.CounterVec
	escalationsScheduled prometheus.Counter
}

// NewAlertRouter creates a new AlertRouter instance
func NewAlertRouter(
	fatigueTracker AlertFatigueTracker,
	userService UserPreferenceService,
	deliveryService NotificationDeliveryService,
	escalationMgr EscalationManager,
	logger *zap.Logger,
) *AlertRouter {
	metrics := &RouterMetrics{
		alertsRoutedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerts_routed_total",
				Help: "Total number of alerts routed by severity",
			},
			[]string{"severity", "alert_type"},
		),
		routingDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "routing_duration_seconds",
				Help:    "Duration of alert routing in seconds",
				Buckets: prometheus.DefBuckets,
			},
		),
		usersTargetedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "users_targeted_total",
				Help: "Total number of users targeted for notifications",
			},
		),
		alertsSuppressedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alerts_suppressed_total",
				Help: "Total number of alerts suppressed by reason",
			},
			[]string{"reason"},
		),
		escalationsScheduled: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "escalations_scheduled_total",
				Help: "Total number of escalations scheduled",
			},
		),
	}

	return &AlertRouter{
		fatigueTracker:  fatigueTracker,
		userService:     userService,
		deliveryService: deliveryService,
		escalationMgr:   escalationMgr,
		logger:          logger,
		metrics:         metrics,
	}
}

// RouteAlert is the main entry point for routing an alert to appropriate users
func (r *AlertRouter) RouteAlert(ctx context.Context, alert *models.Alert) error {
	startTime := time.Now()
	defer func() {
		r.metrics.routingDuration.Observe(time.Since(startTime).Seconds())
	}()

	r.logger.Info("Routing alert",
		zap.String("alert_id", alert.AlertID),
		zap.String("patient_id", alert.PatientID),
		zap.String("severity", string(alert.Severity)),
		zap.String("alert_type", string(alert.AlertType)),
		zap.String("department_id", alert.DepartmentID),
	)

	// Record metric
	r.metrics.alertsRoutedTotal.WithLabelValues(
		string(alert.Severity),
		string(alert.AlertType),
	).Inc()

	// Step 1: Determine target users based on severity and alert type
	users, err := r.determineTargetUsers(alert)
	if err != nil {
		r.logger.Error("Failed to determine target users",
			zap.String("alert_id", alert.AlertID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to determine target users: %w", err)
	}

	if len(users) == 0 {
		r.logger.Warn("No target users found for alert",
			zap.String("alert_id", alert.AlertID),
			zap.String("department_id", alert.DepartmentID),
		)
		return fmt.Errorf("no target users found for alert %s", alert.AlertID)
	}

	r.logger.Info("Target users determined",
		zap.String("alert_id", alert.AlertID),
		zap.Int("user_count", len(users)),
	)

	// Step 2: Check alert fatigue and send notifications for each user
	notificationsSent := 0
	for _, user := range users {
		r.metrics.usersTargetedTotal.Inc()

		// Check if alert should be suppressed due to fatigue
		shouldSuppress, reason := r.fatigueTracker.ShouldSuppress(alert, user)
		if shouldSuppress {
			r.logger.Info("Alert suppressed due to fatigue",
				zap.String("alert_id", alert.AlertID),
				zap.String("user_id", user.ID),
				zap.String("user_name", user.Name),
				zap.String("reason", reason),
			)
			r.metrics.alertsSuppressedTotal.WithLabelValues(reason).Inc()
			continue
		}

		// Step 3: Get user's preferred channels for this severity
		channels := r.userService.GetPreferredChannels(user, alert.Severity)
		if len(channels) == 0 {
			r.logger.Warn("No preferred channels for user",
				zap.String("user_id", user.ID),
				zap.String("severity", string(alert.Severity)),
			)
			// Fall back to default channels for severity
			channels = models.DefaultSeverityChannels[alert.Severity]
		}

		r.logger.Debug("Channels determined for user",
			zap.String("user_id", user.ID),
			zap.Int("channel_count", len(channels)),
		)

		// Step 4: Build and send notifications for each channel
		for _, channel := range channels {
			notification := r.buildNotification(alert, user, channel)

			// Send notification asynchronously
			go func(notif *models.Notification, ch models.NotificationChannel) {
				if err := r.deliveryService.Send(ctx, notif); err != nil {
					r.logger.Error("Failed to send notification",
						zap.String("notification_id", notif.ID),
						zap.String("alert_id", alert.AlertID),
						zap.String("user_id", user.ID),
						zap.String("channel", string(ch)),
						zap.Error(err),
					)
				} else {
					r.logger.Info("Notification sent successfully",
						zap.String("notification_id", notif.ID),
						zap.String("alert_id", alert.AlertID),
						zap.String("user_id", user.ID),
						zap.String("channel", string(ch)),
					)
				}
			}(notification, channel)

			notificationsSent++
		}

		// Step 5: Record notification sent for fatigue tracking
		r.fatigueTracker.RecordNotification(user.ID, alert)
	}

	r.logger.Info("Alert routing completed",
		zap.String("alert_id", alert.AlertID),
		zap.Int("users_notified", notificationsSent),
	)

	// Step 6: Schedule escalation if required
	if r.shouldScheduleEscalation(alert) {
		timeout := r.getEscalationTimeout(alert)
		if err := r.escalationMgr.ScheduleEscalation(ctx, alert, timeout); err != nil {
			r.logger.Error("Failed to schedule escalation",
				zap.String("alert_id", alert.AlertID),
				zap.Error(err),
			)
			// Don't fail the entire routing if escalation scheduling fails
		} else {
			r.metrics.escalationsScheduled.Inc()
			r.logger.Info("Escalation scheduled",
				zap.String("alert_id", alert.AlertID),
				zap.Duration("timeout", timeout),
			)
		}
	}

	return nil
}

// determineTargetUsers selects users to notify based on alert severity and type
func (r *AlertRouter) determineTargetUsers(alert *models.Alert) ([]*models.User, error) {
	var users []*models.User
	var err error

	// Base routing on severity
	switch alert.Severity {
	case models.SeverityCritical:
		// CRITICAL: Attending Physician + Charge Nurse
		attending, err := r.userService.GetAttendingPhysician(alert.DepartmentID)
		if err != nil {
			r.logger.Error("Failed to get attending physician",
				zap.String("department_id", alert.DepartmentID),
				zap.Error(err),
			)
		} else {
			users = append(users, attending...)
		}

		chargeNurse, err := r.userService.GetChargeNurse(alert.DepartmentID)
		if err != nil {
			r.logger.Error("Failed to get charge nurse",
				zap.String("department_id", alert.DepartmentID),
				zap.Error(err),
			)
		} else {
			users = append(users, chargeNurse...)
		}

	case models.SeverityHigh:
		// HIGH: Primary Nurse + Resident
		primaryNurse, err := r.userService.GetPrimaryNurse(alert.PatientID)
		if err != nil {
			r.logger.Error("Failed to get primary nurse",
				zap.String("patient_id", alert.PatientID),
				zap.Error(err),
			)
		} else {
			users = append(users, primaryNurse...)
		}

		resident, err := r.userService.GetResident(alert.DepartmentID)
		if err != nil {
			r.logger.Error("Failed to get resident",
				zap.String("department_id", alert.DepartmentID),
				zap.Error(err),
			)
		} else {
			users = append(users, resident...)
		}

	case models.SeverityModerate, models.SeverityLow:
		// MODERATE/LOW: Primary Nurse only
		primaryNurse, err := r.userService.GetPrimaryNurse(alert.PatientID)
		if err != nil {
			r.logger.Error("Failed to get primary nurse",
				zap.String("patient_id", alert.PatientID),
				zap.Error(err),
			)
		} else {
			users = append(users, primaryNurse...)
		}

	case models.SeverityMLAlert:
		// ML_ALERT: Clinical Informatics Team
		informaticsTeam, err := r.userService.GetClinicalInformaticsTeam()
		if err != nil {
			r.logger.Error("Failed to get clinical informatics team",
				zap.Error(err),
			)
		} else {
			users = append(users, informaticsTeam...)
		}
	}

	// Special routing for ML-sourced alerts - always include informatics team
	if alert.Metadata.SourceModule == "MODULE5_ML_INFERENCE" {
		informaticsTeam, err := r.userService.GetClinicalInformaticsTeam()
		if err != nil {
			r.logger.Warn("Failed to get clinical informatics team for ML alert",
				zap.Error(err),
			)
		} else {
			// Add informatics team users if not already included
			users = r.mergeUsers(users, informaticsTeam)
		}
	}

	return users, err
}

// buildNotification creates a notification object from alert, user, and channel
func (r *AlertRouter) buildNotification(
	alert *models.Alert,
	user *models.User,
	channel models.NotificationChannel,
) *models.Notification {
	now := time.Now()

	// Determine priority based on severity
	priority := r.severityToPriority(alert.Severity)

	// Build message based on channel constraints
	message := r.formatMessageForChannel(alert, channel)

	notification := &models.Notification{
		ID:         uuid.New().String(),
		AlertID:    alert.AlertID,
		UserID:     user.ID,
		User:       user,
		Alert:      alert,
		Channel:    channel,
		Priority:   priority,
		Message:    message,
		Status:     models.StatusPending,
		RetryCount: 0,
		CreatedAt:  now,
		Metadata: map[string]interface{}{
			"severity":       string(alert.Severity),
			"alert_type":     string(alert.AlertType),
			"patient_id":     alert.PatientID,
			"department_id":  alert.DepartmentID,
			"source_module":  alert.Metadata.SourceModule,
		},
	}

	return notification
}

// formatMessageForChannel formats the alert message based on channel constraints
func (r *AlertRouter) formatMessageForChannel(alert *models.Alert, channel models.NotificationChannel) string {
	switch channel {
	case models.ChannelSMS:
		// SMS: Max 160 characters
		// Format: "CRITICAL: PAT-001 Sepsis Alert (92%) - ICU Bed 5"
		return fmt.Sprintf(
			"%s: %s %s (%.0f%%) - %s",
			alert.Severity,
			alert.PatientID,
			alert.AlertType,
			alert.Confidence*100,
			alert.PatientLocation.Room,
		)

	case models.ChannelPager:
		// Pager: Ultra-short alphanumeric
		// Format: "CRIT PAT-001 SEPSIS ICU-5"
		severityShort := string(alert.Severity)[:4]
		alertTypeShort := string(alert.AlertType)
		if len(alertTypeShort) > 10 {
			alertTypeShort = alertTypeShort[:10]
		}
		return fmt.Sprintf(
			"%s %s %s %s",
			severityShort,
			alert.PatientID,
			alertTypeShort,
			alert.PatientLocation.Room,
		)

	case models.ChannelPush:
		// Push: Title + body format
		return fmt.Sprintf(
			"%s Alert: %s for patient %s in %s. Confidence: %.0f%%",
			alert.Severity,
			alert.AlertType,
			alert.PatientID,
			alert.PatientLocation.Room,
			alert.Confidence*100,
		)

	case models.ChannelEmail, models.ChannelInApp:
		// Email/In-App: Full details with recommendations
		return alert.Message

	case models.ChannelVoice:
		// Voice: Clear spoken message
		return fmt.Sprintf(
			"Critical alert. %s. Patient %s in %s has %s with %.0f percent confidence.",
			alert.Severity,
			alert.PatientID,
			alert.PatientLocation.Room,
			alert.AlertType,
			alert.Confidence*100,
		)

	default:
		return alert.Message
	}
}

// severityToPriority converts alert severity to notification priority
func (r *AlertRouter) severityToPriority(severity models.AlertSeverity) int {
	switch severity {
	case models.SeverityCritical:
		return 1 // Highest
	case models.SeverityHigh:
		return 2
	case models.SeverityModerate:
		return 3
	case models.SeverityLow:
		return 4
	case models.SeverityMLAlert:
		return 3
	default:
		return 5 // Lowest
	}
}

// shouldScheduleEscalation determines if escalation should be scheduled
func (r *AlertRouter) shouldScheduleEscalation(alert *models.Alert) bool {
	// Schedule escalation for CRITICAL and HIGH alerts
	if alert.Severity == models.SeverityCritical || alert.Severity == models.SeverityHigh {
		return true
	}

	// Also schedule if explicitly required in metadata
	if alert.Metadata.RequiresEscalation {
		return true
	}

	return false
}

// getEscalationTimeout returns the escalation timeout for the alert
func (r *AlertRouter) getEscalationTimeout(alert *models.Alert) time.Duration {
	if timeout, ok := models.DefaultEscalationTimeouts[alert.Severity]; ok {
		return timeout
	}
	return 15 * time.Minute // Default
}

// mergeUsers combines two user slices, avoiding duplicates
func (r *AlertRouter) mergeUsers(existing, additional []*models.User) []*models.User {
	userMap := make(map[string]*models.User)

	// Add existing users
	for _, user := range existing {
		userMap[user.ID] = user
	}

	// Add additional users (skipping duplicates)
	for _, user := range additional {
		if _, exists := userMap[user.ID]; !exists {
			userMap[user.ID] = user
		}
	}

	// Convert back to slice
	result := make([]*models.User, 0, len(userMap))
	for _, user := range userMap {
		result = append(result, user)
	}

	return result
}

// GetRoutingDecision returns a routing decision without sending notifications (for testing/preview)
func (r *AlertRouter) GetRoutingDecision(ctx context.Context, alert *models.Alert) (*models.RoutingDecision, error) {
	users, err := r.determineTargetUsers(alert)
	if err != nil {
		return nil, fmt.Errorf("failed to determine target users: %w", err)
	}

	decision := &models.RoutingDecision{
		Alert:              alert,
		TargetUsers:        users,
		UserChannels:       make(map[string][]models.NotificationChannel),
		SuppressedUsers:    make(map[string]string),
		RequiresEscalation: r.shouldScheduleEscalation(alert),
		EscalationTimeout:  r.getEscalationTimeout(alert),
	}

	// Check fatigue and get channels for each user
	for _, user := range users {
		shouldSuppress, reason := r.fatigueTracker.ShouldSuppress(alert, user)
		if shouldSuppress {
			decision.SuppressedUsers[user.ID] = reason
		} else {
			channels := r.userService.GetPreferredChannels(user, alert.Severity)
			if len(channels) == 0 {
				channels = models.DefaultSeverityChannels[alert.Severity]
			}
			decision.UserChannels[user.ID] = channels
		}
	}

	return decision, nil
}
