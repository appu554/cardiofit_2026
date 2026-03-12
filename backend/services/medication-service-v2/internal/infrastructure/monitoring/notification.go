package monitoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// NotificationChannel represents a notification channel type
type NotificationChannel string

const (
	ChannelEmail     NotificationChannel = "email"
	ChannelSlack     NotificationChannel = "slack"
	ChannelPagerDuty NotificationChannel = "pagerduty"
	ChannelWebhook   NotificationChannel = "webhook"
	ChannelSMS       NotificationChannel = "sms"
)

// NotificationMessage represents a notification message
type NotificationMessage struct {
	Channel     NotificationChannel    `json:"channel"`
	Recipients  []string               `json:"recipients"`
	Subject     string                 `json:"subject,omitempty"`
	Message     string                 `json:"message"`
	Priority    string                 `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	AlertID     string                 `json:"alert_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NotificationManager manages notification delivery
type NotificationManager struct {
	channels map[string]NotificationChannelConfig
	logger   *Logger
	
	// Channel handlers
	handlers map[NotificationChannel]NotificationHandler
}

// NotificationHandler interface for notification channels
type NotificationHandler interface {
	Send(ctx context.Context, message *NotificationMessage) error
	GetChannelType() NotificationChannel
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(channels map[string]NotificationChannelConfig, logger *Logger) *NotificationManager {
	nm := &NotificationManager{
		channels: channels,
		logger:   logger,
		handlers: make(map[NotificationChannel]NotificationHandler),
	}

	// Initialize handlers for enabled channels
	for name, config := range channels {
		if !config.Enabled {
			continue
		}

		var handler NotificationHandler
		switch NotificationChannel(config.Type) {
		case ChannelEmail:
			handler = NewEmailHandler(config, logger)
		case ChannelSlack:
			handler = NewSlackHandler(config, logger)
		case ChannelPagerDuty:
			handler = NewPagerDutyHandler(config, logger)
		case ChannelWebhook:
			handler = NewWebhookHandler(config, logger)
		case ChannelSMS:
			handler = NewSMSHandler(config, logger)
		default:
			logger.Warn("Unknown notification channel type", zap.String("type", config.Type))
			continue
		}

		nm.handlers[NotificationChannel(config.Type)] = handler
		logger.Info("Notification handler initialized", zap.String("channel", name), zap.String("type", config.Type))
	}

	return nm
}

// SendAlert sends an alert notification
func (nm *NotificationManager) SendAlert(ctx context.Context, alert *Alert) {
	message := nm.createAlertMessage(alert)
	nm.sendNotification(ctx, message)
}

// SendResolution sends an alert resolution notification
func (nm *NotificationManager) SendResolution(ctx context.Context, alert *Alert) {
	message := nm.createResolutionMessage(alert)
	nm.sendNotification(ctx, message)
}

// SendEscalation sends an alert escalation notification
func (nm *NotificationManager) SendEscalation(ctx context.Context, alert *Alert, escalation *EscalationRule) {
	message := nm.createEscalationMessage(alert, escalation)
	nm.sendNotification(ctx, message)
}

// createAlertMessage creates a notification message for a new alert
func (nm *NotificationManager) createAlertMessage(alert *Alert) *NotificationMessage {
	priority := nm.getPriorityFromSeverity(alert.Severity)
	
	subject := fmt.Sprintf("[%s] %s - %s", alert.Severity, alert.Category, alert.Name)
	
	message := fmt.Sprintf(`🚨 ALERT FIRED 🚨

Alert: %s
Category: %s
Severity: %s
Service: %s

Description: %s

Current Value: %v
Threshold: %v

Started: %s
Status: %s

Clinical Impact: %s
Safety Risk: %s

Action Required:
%s

Alert ID: %s
Correlation ID: %s`,
		alert.Name,
		alert.Category,
		alert.Severity,
		alert.ServiceName,
		alert.Description,
		alert.CurrentValue,
		alert.ThresholdValue,
		alert.StartTime.Format(time.RFC3339),
		alert.Status,
		alert.ClinicalImpact,
		alert.SafetyRisk,
		nm.formatActionItems(alert.ActionRequired),
		alert.ID,
		alert.CorrelationID,
	)

	return &NotificationMessage{
		Channel:    nm.getChannelForSeverity(alert.Severity),
		Recipients: nm.getRecipientsForAlert(alert),
		Subject:    subject,
		Message:    message,
		Priority:   priority,
		AlertID:    alert.ID,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"alert_category": alert.Category,
			"alert_severity": alert.Severity,
			"service_name":   alert.ServiceName,
			"patient_safety": alert.Category == CategoryPatientSafety,
		},
	}
}

// createResolutionMessage creates a notification message for alert resolution
func (nm *NotificationManager) createResolutionMessage(alert *Alert) *NotificationMessage {
	subject := fmt.Sprintf("[RESOLVED] %s - %s", alert.Category, alert.Name)
	
	duration := "N/A"
	if alert.EndTime != nil {
		duration = alert.EndTime.Sub(alert.StartTime).String()
	}
	
	message := fmt.Sprintf(`✅ ALERT RESOLVED ✅

Alert: %s
Category: %s
Severity: %s
Service: %s

Duration: %s
Resolved: %s

Alert ID: %s`,
		alert.Name,
		alert.Category,
		alert.Severity,
		alert.ServiceName,
		duration,
		alert.LastUpdate.Format(time.RFC3339),
		alert.ID,
	)

	return &NotificationMessage{
		Channel:    nm.getChannelForSeverity(alert.Severity),
		Recipients: nm.getRecipientsForAlert(alert),
		Subject:    subject,
		Message:    message,
		Priority:   "normal",
		AlertID:    alert.ID,
		Timestamp:  time.Now(),
	}
}

// createEscalationMessage creates a notification message for alert escalation
func (nm *NotificationManager) createEscalationMessage(alert *Alert, escalation *EscalationRule) *NotificationMessage {
	subject := fmt.Sprintf("[ESCALATED L%d] %s - %s", escalation.Level, alert.Category, alert.Name)
	
	duration := time.Since(alert.StartTime).String()
	
	message := fmt.Sprintf(`⚠️ ALERT ESCALATED ⚠️

Alert: %s
Escalation Level: %d
Category: %s
Severity: %s
Service: %s

Duration: %s (unresolved)
Current Value: %v
Threshold: %v

Clinical Impact: %s
Safety Risk: %s

%s

Alert ID: %s`,
		alert.Name,
		escalation.Level,
		alert.Category,
		alert.Severity,
		alert.ServiceName,
		duration,
		alert.CurrentValue,
		alert.ThresholdValue,
		alert.ClinicalImpact,
		alert.SafetyRisk,
		escalation.Message,
		alert.ID,
	)

	return &NotificationMessage{
		Channel:    nm.getChannelForSeverity(alert.Severity),
		Recipients: escalation.Recipients,
		Subject:    subject,
		Message:    message,
		Priority:   "urgent",
		AlertID:    alert.ID,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"escalation_level": escalation.Level,
			"alert_duration":   duration,
		},
	}
}

// sendNotification sends a notification through appropriate channels
func (nm *NotificationManager) sendNotification(ctx context.Context, message *NotificationMessage) {
	handler, exists := nm.handlers[message.Channel]
	if !exists {
		nm.logger.Warn("No handler found for notification channel", 
			zap.String("channel", string(message.Channel)))
		return
	}

	go func() {
		if err := handler.Send(ctx, message); err != nil {
			nm.logger.Error("Failed to send notification",
				zap.String("channel", string(message.Channel)),
				zap.String("alert_id", message.AlertID),
				zap.Error(err),
			)
		} else {
			nm.logger.Info("Notification sent successfully",
				zap.String("channel", string(message.Channel)),
				zap.String("alert_id", message.AlertID),
				zap.Strings("recipients", message.Recipients),
			)
		}
	}()
}

// Helper methods

func (nm *NotificationManager) getPriorityFromSeverity(severity AlertSeverity) string {
	switch severity {
	case SeverityCritical:
		return "critical"
	case SeverityHigh:
		return "urgent"
	case SeverityMedium:
		return "normal"
	case SeverityLow, SeverityInfo:
		return "low"
	default:
		return "normal"
	}
}

func (nm *NotificationManager) getChannelForSeverity(severity AlertSeverity) NotificationChannel {
	switch severity {
	case SeverityCritical:
		return ChannelPagerDuty // Critical alerts go to PagerDuty
	case SeverityHigh:
		return ChannelSlack     // High alerts go to Slack
	default:
		return ChannelEmail     // Other alerts go to email
	}
}

func (nm *NotificationManager) getRecipientsForAlert(alert *Alert) []string {
	// Default recipients based on alert category
	switch alert.Category {
	case CategoryPatientSafety:
		return []string{"safety-team@hospital.com", "medical-director@hospital.com"}
	case CategorySystemHealth:
		return []string{"devops@hospital.com", "platform-team@hospital.com"}
	case CategorySecurity:
		return []string{"security@hospital.com", "compliance@hospital.com"}
	case CategoryCompliance:
		return []string{"compliance@hospital.com", "legal@hospital.com"}
	default:
		return []string{"alerts@hospital.com"}
	}
}

func (nm *NotificationManager) formatActionItems(actions []string) string {
	if len(actions) == 0 {
		return "No specific actions defined"
	}
	
	result := ""
	for i, action := range actions {
		result += fmt.Sprintf("%d. %s\n", i+1, action)
	}
	return result
}

// Notification Channel Handlers

// EmailHandler handles email notifications
type EmailHandler struct {
	config NotificationChannelConfig
	logger *Logger
}

func NewEmailHandler(config NotificationChannelConfig, logger *Logger) *EmailHandler {
	return &EmailHandler{config: config, logger: logger}
}

func (h *EmailHandler) GetChannelType() NotificationChannel {
	return ChannelEmail
}

func (h *EmailHandler) Send(ctx context.Context, message *NotificationMessage) error {
	// Mock email sending - in production, integrate with actual email service
	h.logger.Info("Email notification sent (mock)",
		zap.String("subject", message.Subject),
		zap.Strings("recipients", message.Recipients),
		zap.String("priority", message.Priority),
	)
	return nil
}

// SlackHandler handles Slack notifications
type SlackHandler struct {
	config  NotificationChannelConfig
	logger  *Logger
	webhook string
}

func NewSlackHandler(config NotificationChannelConfig, logger *Logger) *SlackHandler {
	webhook, _ := config.Settings["webhook_url"].(string)
	return &SlackHandler{
		config:  config,
		logger:  logger,
		webhook: webhook,
	}
}

func (h *SlackHandler) GetChannelType() NotificationChannel {
	return ChannelSlack
}

func (h *SlackHandler) Send(ctx context.Context, message *NotificationMessage) error {
	if h.webhook == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	// Create Slack message payload
	payload := map[string]interface{}{
		"text": message.Subject,
		"attachments": []map[string]interface{}{
			{
				"color":      h.getSlackColor(message.Priority),
				"title":      message.Subject,
				"text":       message.Message,
				"timestamp":  message.Timestamp.Unix(),
				"footer":     "Medication Service V2",
				"footer_icon": "🏥",
				"fields": []map[string]interface{}{
					{
						"title": "Alert ID",
						"value": message.AlertID,
						"short": true,
					},
					{
						"title": "Priority",
						"value": message.Priority,
						"short": true,
					},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	// Send HTTP POST to Slack webhook
	resp, err := http.Post(h.webhook, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (h *SlackHandler) getSlackColor(priority string) string {
	switch priority {
	case "critical":
		return "danger"
	case "urgent":
		return "warning"
	case "normal":
		return "good"
	default:
		return "#36C5F0" // Slack blue
	}
}

// PagerDutyHandler handles PagerDuty notifications
type PagerDutyHandler struct {
	config NotificationChannelConfig
	logger *Logger
	apiKey string
}

func NewPagerDutyHandler(config NotificationChannelConfig, logger *Logger) *PagerDutyHandler {
	apiKey, _ := config.Settings["api_key"].(string)
	return &PagerDutyHandler{
		config: config,
		logger: logger,
		apiKey: apiKey,
	}
}

func (h *PagerDutyHandler) GetChannelType() NotificationChannel {
	return ChannelPagerDuty
}

func (h *PagerDutyHandler) Send(ctx context.Context, message *NotificationMessage) error {
	if h.apiKey == "" {
		return fmt.Errorf("pagerduty API key not configured")
	}

	// Mock PagerDuty integration - in production, use PagerDuty Events API
	h.logger.Info("PagerDuty notification sent (mock)",
		zap.String("subject", message.Subject),
		zap.String("priority", message.Priority),
		zap.String("alert_id", message.AlertID),
	)
	
	return nil
}

// WebhookHandler handles generic webhook notifications
type WebhookHandler struct {
	config NotificationChannelConfig
	logger *Logger
	url    string
}

func NewWebhookHandler(config NotificationChannelConfig, logger *Logger) *WebhookHandler {
	url, _ := config.Settings["url"].(string)
	return &WebhookHandler{
		config: config,
		logger: logger,
		url:    url,
	}
}

func (h *WebhookHandler) GetChannelType() NotificationChannel {
	return ChannelWebhook
}

func (h *WebhookHandler) Send(ctx context.Context, message *NotificationMessage) error {
	if h.url == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	// Create webhook payload
	payload := map[string]interface{}{
		"timestamp":   message.Timestamp,
		"alert_id":    message.AlertID,
		"subject":     message.Subject,
		"message":     message.Message,
		"priority":    message.Priority,
		"recipients":  message.Recipients,
		"metadata":    message.Metadata,
		"service":     "medication-service-v2",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Send HTTP POST to webhook
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(h.url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send webhook notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SMSHandler handles SMS notifications
type SMSHandler struct {
	config NotificationChannelConfig
	logger *Logger
}

func NewSMSHandler(config NotificationChannelConfig, logger *Logger) *SMSHandler {
	return &SMSHandler{config: config, logger: logger}
}

func (h *SMSHandler) GetChannelType() NotificationChannel {
	return ChannelSMS
}

func (h *SMSHandler) Send(ctx context.Context, message *NotificationMessage) error {
	// Mock SMS sending - in production, integrate with SMS service (Twilio, AWS SNS, etc.)
	h.logger.Info("SMS notification sent (mock)",
		zap.String("subject", message.Subject),
		zap.Strings("recipients", message.Recipients),
		zap.String("priority", message.Priority),
	)
	return nil
}