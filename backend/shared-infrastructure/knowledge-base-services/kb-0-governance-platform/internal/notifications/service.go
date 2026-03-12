// Package notifications provides notification delivery for KB-0 governance events.
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// NOTIFICATION SERVICE
// =============================================================================

// Service handles notification delivery for governance events.
type Service struct {
	db           *sql.DB
	emailConfig  *EmailConfig
	slackConfig  *SlackConfig
	webhookURLs  []string
	templates    map[NotificationType]*NotificationTemplate
}

// Config holds notification service configuration.
type Config struct {
	Email    *EmailConfig
	Slack    *SlackConfig
	Webhooks []string
}

// EmailConfig holds email notification settings.
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	FromAddress  string
	FromName     string
	Enabled      bool
}

// SlackConfig holds Slack notification settings.
type SlackConfig struct {
	WebhookURL string
	Channel    string
	Enabled    bool
}

// NewService creates a new notification service.
func NewService(db *sql.DB, cfg *Config) *Service {
	svc := &Service{
		db:          db,
		templates:   make(map[NotificationType]*NotificationTemplate),
		webhookURLs: cfg.Webhooks,
	}

	if cfg.Email != nil {
		svc.emailConfig = cfg.Email
	}
	if cfg.Slack != nil {
		svc.slackConfig = cfg.Slack
	}

	// Initialize default templates
	svc.initializeTemplates()

	return svc
}

// =============================================================================
// NOTIFICATION TYPES
// =============================================================================

// NotificationType represents the type of notification.
type NotificationType string

const (
	// Workflow Notifications
	NotifyItemCreated         NotificationType = "ITEM_CREATED"
	NotifyItemSubmitted       NotificationType = "ITEM_SUBMITTED"
	NotifyReviewRequired      NotificationType = "REVIEW_REQUIRED"
	NotifyReviewCompleted     NotificationType = "REVIEW_COMPLETED"
	NotifyApprovalRequired    NotificationType = "APPROVAL_REQUIRED"
	NotifyApprovalCompleted   NotificationType = "APPROVAL_COMPLETED"
	NotifyItemActivated       NotificationType = "ITEM_ACTIVATED"
	NotifyItemRejected        NotificationType = "ITEM_REJECTED"
	NotifyItemRetired         NotificationType = "ITEM_RETIRED"

	// Urgent Notifications
	NotifyEmergencyOverride   NotificationType = "EMERGENCY_OVERRIDE"
	NotifySLABreach           NotificationType = "SLA_BREACH"
	NotifyHighRiskPending     NotificationType = "HIGH_RISK_PENDING"

	// Digest Notifications
	NotifyDailyDigest         NotificationType = "DAILY_DIGEST"
	NotifyWeeklyReport        NotificationType = "WEEKLY_REPORT"
	NotifyMonthlyCompliance   NotificationType = "MONTHLY_COMPLIANCE"

	// System Notifications
	NotifyIngestionComplete   NotificationType = "INGESTION_COMPLETE"
	NotifyIngestionFailed     NotificationType = "INGESTION_FAILED"
	NotifySystemAlert         NotificationType = "SYSTEM_ALERT"
)

// NotificationPriority represents notification urgency.
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "LOW"
	PriorityNormal   NotificationPriority = "NORMAL"
	PriorityHigh     NotificationPriority = "HIGH"
	PriorityCritical NotificationPriority = "CRITICAL"
)

// =============================================================================
// NOTIFICATION STRUCTURE
// =============================================================================

// Notification represents a notification to be sent.
type Notification struct {
	ID           string               `json:"id"`
	Type         NotificationType     `json:"type"`
	Priority     NotificationPriority `json:"priority"`
	Recipients   []Recipient          `json:"recipients"`
	Subject      string               `json:"subject"`
	Body         string               `json:"body"`
	HTMLBody     string               `json:"html_body,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	ItemID       string               `json:"item_id,omitempty"`
	KB           models.KB            `json:"kb,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	SentAt       *time.Time           `json:"sent_at,omitempty"`
	DeliveryStatus DeliveryStatus     `json:"delivery_status"`
	Channels     []Channel            `json:"channels"`
}

// Recipient represents a notification recipient.
type Recipient struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email,omitempty"`
	SlackID    string `json:"slack_id,omitempty"`
	Role       string `json:"role,omitempty"`
	Preference NotificationPreference `json:"preference,omitempty"`
}

// NotificationPreference holds user notification preferences.
type NotificationPreference struct {
	Email       bool     `json:"email"`
	Slack       bool     `json:"slack"`
	InApp       bool     `json:"in_app"`
	DigestOnly  bool     `json:"digest_only"`
	Priorities  []NotificationPriority `json:"priorities,omitempty"`
}

// DeliveryStatus represents notification delivery status.
type DeliveryStatus string

const (
	StatusPending   DeliveryStatus = "PENDING"
	StatusSent      DeliveryStatus = "SENT"
	StatusDelivered DeliveryStatus = "DELIVERED"
	StatusFailed    DeliveryStatus = "FAILED"
	StatusBounced   DeliveryStatus = "BOUNCED"
)

// Channel represents a notification delivery channel.
type Channel string

const (
	ChannelEmail   Channel = "EMAIL"
	ChannelSlack   Channel = "SLACK"
	ChannelInApp   Channel = "IN_APP"
	ChannelWebhook Channel = "WEBHOOK"
	ChannelSMS     Channel = "SMS"
)

// =============================================================================
// NOTIFICATION TEMPLATES
// =============================================================================

// NotificationTemplate represents a notification template.
type NotificationTemplate struct {
	Type        NotificationType
	Subject     string
	BodyText    string
	BodyHTML    string
	Priority    NotificationPriority
	Channels    []Channel
}

// initializeTemplates sets up default notification templates.
func (s *Service) initializeTemplates() {
	s.templates[NotifyReviewRequired] = &NotificationTemplate{
		Type:     NotifyReviewRequired,
		Subject:  "[KB-0] Review Required: {{.ItemName}}",
		BodyText: "A new knowledge item requires your review.\n\nItem: {{.ItemName}}\nKB: {{.KB}}\nRisk Level: {{.RiskLevel}}\nSubmitted By: {{.SubmittedBy}}\n\nPlease review at: {{.ReviewURL}}",
		BodyHTML: `<h2>Review Required</h2>
<p>A new knowledge item requires your review.</p>
<table>
<tr><td><strong>Item:</strong></td><td>{{.ItemName}}</td></tr>
<tr><td><strong>KB:</strong></td><td>{{.KB}}</td></tr>
<tr><td><strong>Risk Level:</strong></td><td>{{.RiskLevel}}</td></tr>
<tr><td><strong>Submitted By:</strong></td><td>{{.SubmittedBy}}</td></tr>
</table>
<p><a href="{{.ReviewURL}}">Click here to review</a></p>`,
		Priority: PriorityNormal,
		Channels: []Channel{ChannelEmail, ChannelSlack, ChannelInApp},
	}

	s.templates[NotifyApprovalRequired] = &NotificationTemplate{
		Type:     NotifyApprovalRequired,
		Subject:  "[KB-0] Approval Required: {{.ItemName}}",
		BodyText: "A knowledge item requires CMO approval.\n\nItem: {{.ItemName}}\nKB: {{.KB}}\nRisk Level: {{.RiskLevel}}\nReviewed By: {{.ReviewedBy}}\n\nPlease approve at: {{.ApprovalURL}}",
		BodyHTML: `<h2>CMO Approval Required</h2>
<p>A knowledge item has completed review and requires CMO approval.</p>
<table>
<tr><td><strong>Item:</strong></td><td>{{.ItemName}}</td></tr>
<tr><td><strong>KB:</strong></td><td>{{.KB}}</td></tr>
<tr><td><strong>Risk Level:</strong></td><td>{{.RiskLevel}}</td></tr>
<tr><td><strong>Reviewed By:</strong></td><td>{{.ReviewedBy}}</td></tr>
</table>
<p><a href="{{.ApprovalURL}}">Click here to approve</a></p>`,
		Priority: PriorityHigh,
		Channels: []Channel{ChannelEmail, ChannelSlack, ChannelInApp},
	}

	s.templates[NotifyEmergencyOverride] = &NotificationTemplate{
		Type:     NotifyEmergencyOverride,
		Subject:  "[KB-0] URGENT: Emergency Override Activated",
		BodyText: "EMERGENCY OVERRIDE ACTIVATED\n\nItem: {{.ItemName}}\nKB: {{.KB}}\nActivated By: {{.ActivatedBy}}\nReason: {{.Reason}}\nTime: {{.Timestamp}}\n\nImmediate CMO review required.",
		BodyHTML: `<h2 style="color: red;">⚠️ EMERGENCY OVERRIDE ACTIVATED</h2>
<p style="font-weight: bold; color: red;">An emergency override has been activated. Immediate CMO review required.</p>
<table>
<tr><td><strong>Item:</strong></td><td>{{.ItemName}}</td></tr>
<tr><td><strong>KB:</strong></td><td>{{.KB}}</td></tr>
<tr><td><strong>Activated By:</strong></td><td>{{.ActivatedBy}}</td></tr>
<tr><td><strong>Reason:</strong></td><td>{{.Reason}}</td></tr>
<tr><td><strong>Time:</strong></td><td>{{.Timestamp}}</td></tr>
</table>`,
		Priority: PriorityCritical,
		Channels: []Channel{ChannelEmail, ChannelSlack, ChannelInApp, ChannelWebhook},
	}

	s.templates[NotifySLABreach] = &NotificationTemplate{
		Type:     NotifySLABreach,
		Subject:  "[KB-0] SLA Breach Alert: {{.ItemName}}",
		BodyText: "SLA BREACH ALERT\n\nItem: {{.ItemName}}\nKB: {{.KB}}\nState: {{.State}}\nDays Pending: {{.DaysPending}}\nSLA Days: {{.SLADays}}\n\nImmediate action required.",
		BodyHTML: `<h2 style="color: orange;">⚠️ SLA Breach Alert</h2>
<p>A knowledge item has exceeded its SLA threshold.</p>
<table>
<tr><td><strong>Item:</strong></td><td>{{.ItemName}}</td></tr>
<tr><td><strong>KB:</strong></td><td>{{.KB}}</td></tr>
<tr><td><strong>State:</strong></td><td>{{.State}}</td></tr>
<tr><td><strong>Days Pending:</strong></td><td style="color: red;">{{.DaysPending}}</td></tr>
<tr><td><strong>SLA Days:</strong></td><td>{{.SLADays}}</td></tr>
</table>`,
		Priority: PriorityHigh,
		Channels: []Channel{ChannelEmail, ChannelSlack, ChannelInApp},
	}

	s.templates[NotifyItemActivated] = &NotificationTemplate{
		Type:     NotifyItemActivated,
		Subject:  "[KB-0] Item Activated: {{.ItemName}}",
		BodyText: "Knowledge item has been activated and is now live.\n\nItem: {{.ItemName}}\nKB: {{.KB}}\nVersion: {{.Version}}\nApproved By: {{.ApprovedBy}}",
		BodyHTML: `<h2 style="color: green;">✓ Item Activated</h2>
<p>A knowledge item has been approved and activated.</p>
<table>
<tr><td><strong>Item:</strong></td><td>{{.ItemName}}</td></tr>
<tr><td><strong>KB:</strong></td><td>{{.KB}}</td></tr>
<tr><td><strong>Version:</strong></td><td>{{.Version}}</td></tr>
<tr><td><strong>Approved By:</strong></td><td>{{.ApprovedBy}}</td></tr>
</table>`,
		Priority: PriorityNormal,
		Channels: []Channel{ChannelEmail, ChannelInApp},
	}

	s.templates[NotifyDailyDigest] = &NotificationTemplate{
		Type:     NotifyDailyDigest,
		Subject:  "[KB-0] Daily Governance Digest - {{.Date}}",
		BodyText: "KB-0 DAILY GOVERNANCE DIGEST\n\nDate: {{.Date}}\n\nPending Reviews: {{.PendingReviews}}\nPending Approvals: {{.PendingApprovals}}\nActivated Today: {{.ActivatedToday}}\nSLA Breaches: {{.SLABreaches}}\n\nView dashboard: {{.DashboardURL}}",
		Priority: PriorityLow,
		Channels: []Channel{ChannelEmail},
	}
}

// =============================================================================
// NOTIFICATION SENDING
// =============================================================================

// Send sends a notification through configured channels.
func (s *Service) Send(ctx context.Context, notification *Notification) error {
	if notification.ID == "" {
		notification.ID = generateNotificationID()
	}
	notification.CreatedAt = time.Now()
	notification.DeliveryStatus = StatusPending

	// Save notification to database
	if err := s.saveNotification(ctx, notification); err != nil {
		return fmt.Errorf("failed to save notification: %w", err)
	}

	// Send through each channel
	var errors []string
	for _, channel := range notification.Channels {
		var err error
		switch channel {
		case ChannelEmail:
			err = s.sendEmail(ctx, notification)
		case ChannelSlack:
			err = s.sendSlack(ctx, notification)
		case ChannelWebhook:
			err = s.sendWebhook(ctx, notification)
		case ChannelInApp:
			err = s.saveInAppNotification(ctx, notification)
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", channel, err))
		}
	}

	// Update delivery status
	if len(errors) == 0 {
		notification.DeliveryStatus = StatusSent
		now := time.Now()
		notification.SentAt = &now
	} else if len(errors) < len(notification.Channels) {
		notification.DeliveryStatus = StatusSent // Partial success
	} else {
		notification.DeliveryStatus = StatusFailed
	}

	// Update notification in database
	if err := s.updateNotificationStatus(ctx, notification); err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("delivery errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// sendEmail sends an email notification.
func (s *Service) sendEmail(ctx context.Context, notification *Notification) error {
	if s.emailConfig == nil || !s.emailConfig.Enabled {
		return nil
	}

	auth := smtp.PlainAuth("",
		s.emailConfig.Username,
		s.emailConfig.Password,
		s.emailConfig.SMTPHost,
	)

	for _, recipient := range notification.Recipients {
		if recipient.Email == "" {
			continue
		}
		if recipient.Preference.DigestOnly && notification.Priority != PriorityCritical {
			continue
		}

		msg := fmt.Sprintf("From: %s <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n"+
			"%s",
			s.emailConfig.FromName,
			s.emailConfig.FromAddress,
			recipient.Email,
			notification.Subject,
			notification.HTMLBody,
		)

		addr := fmt.Sprintf("%s:%d", s.emailConfig.SMTPHost, s.emailConfig.SMTPPort)
		err := smtp.SendMail(addr, auth, s.emailConfig.FromAddress, []string{recipient.Email}, []byte(msg))
		if err != nil {
			return fmt.Errorf("failed to send email to %s: %w", recipient.Email, err)
		}
	}

	return nil
}

// sendSlack sends a Slack notification.
func (s *Service) sendSlack(ctx context.Context, notification *Notification) error {
	if s.slackConfig == nil || !s.slackConfig.Enabled {
		return nil
	}

	// Build Slack message payload
	payload := map[string]interface{}{
		"channel": s.slackConfig.Channel,
		"text":    notification.Subject,
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]interface{}{
					"type": "plain_text",
					"text": notification.Subject,
				},
			},
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": notification.Body,
				},
			},
		},
	}

	// Add color based on priority
	switch notification.Priority {
	case PriorityCritical:
		payload["attachments"] = []map[string]interface{}{
			{"color": "danger"},
		}
	case PriorityHigh:
		payload["attachments"] = []map[string]interface{}{
			{"color": "warning"},
		}
	}

	// In production: send HTTP request to Slack webhook
	_ = payload
	return nil
}

// sendWebhook sends notifications to configured webhooks.
func (s *Service) sendWebhook(ctx context.Context, notification *Notification) error {
	if len(s.webhookURLs) == 0 {
		return nil
	}

	payload := map[string]interface{}{
		"id":        notification.ID,
		"type":      notification.Type,
		"priority":  notification.Priority,
		"subject":   notification.Subject,
		"body":      notification.Body,
		"item_id":   notification.ItemID,
		"kb":        notification.KB,
		"data":      notification.Data,
		"timestamp": notification.CreatedAt,
	}

	// In production: send HTTP POST to each webhook URL
	_ = payload
	return nil
}

// saveInAppNotification saves notification for in-app display.
func (s *Service) saveInAppNotification(ctx context.Context, notification *Notification) error {
	query := `
		INSERT INTO in_app_notifications (
			id, type, priority, subject, body, item_id, kb,
			data, created_at, read
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, false)
	`

	dataJSON, _ := json.Marshal(notification.Data)

	for _, recipient := range notification.Recipients {
		_, err := s.db.ExecContext(ctx, query,
			fmt.Sprintf("%s_%s", notification.ID, recipient.ID),
			notification.Type,
			notification.Priority,
			notification.Subject,
			notification.Body,
			notification.ItemID,
			notification.KB,
			dataJSON,
			notification.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to save in-app notification: %w", err)
		}
	}

	return nil
}

// =============================================================================
// NOTIFICATION EVENTS
// =============================================================================

// NotifyWorkflowTransition sends notifications for workflow state changes.
func (s *Service) NotifyWorkflowTransition(ctx context.Context, item *models.KnowledgeItem, fromState, toState models.ItemState, actor string) error {
	var notifType NotificationType
	var recipients []Recipient

	switch toState {
	case models.StatePrimaryReview:
		notifType = NotifyReviewRequired
		recipients = s.getReviewers(ctx, item.KB, "PRIMARY")
	case models.StateSecondaryReview:
		notifType = NotifyReviewRequired
		recipients = s.getReviewers(ctx, item.KB, "SECONDARY")
	case models.StateCMOApproval:
		notifType = NotifyApprovalRequired
		recipients = s.getApprovers(ctx, item.KB)
	case models.StateActive:
		notifType = NotifyItemActivated
		recipients = s.getSubscribers(ctx, item.KB)
	case models.StateRejected:
		notifType = NotifyItemRejected
		recipients = []Recipient{{ID: item.ID, Email: ""}} // Notify submitter
	case models.StateRetired:
		notifType = NotifyItemRetired
		recipients = s.getSubscribers(ctx, item.KB)
	default:
		return nil
	}

	template := s.templates[notifType]
	if template == nil {
		return nil
	}

	notification := &Notification{
		Type:       notifType,
		Priority:   template.Priority,
		Recipients: recipients,
		Subject:    s.renderTemplate(template.Subject, item),
		Body:       s.renderTemplate(template.BodyText, item),
		HTMLBody:   s.renderTemplate(template.BodyHTML, item),
		ItemID:     item.ID,
		KB:         item.KB,
		Data: map[string]interface{}{
			"ItemName":   item.Name,
			"KB":         item.KB,
			"RiskLevel":  item.RiskLevel,
			"FromState":  fromState,
			"ToState":    toState,
			"Actor":      actor,
			"Timestamp":  time.Now(),
		},
		Channels: template.Channels,
	}

	return s.Send(ctx, notification)
}

// NotifyEmergency sends emergency override notifications.
func (s *Service) NotifyEmergency(ctx context.Context, item *models.KnowledgeItem, actor, reason string) error {
	template := s.templates[NotifyEmergencyOverride]
	if template == nil {
		return fmt.Errorf("emergency template not found")
	}

	// Get all CMOs and administrators
	recipients := s.getEmergencyContacts(ctx)

	notification := &Notification{
		Type:       NotifyEmergencyOverride,
		Priority:   PriorityCritical,
		Recipients: recipients,
		Subject:    s.renderTemplate(template.Subject, item),
		Body:       s.renderTemplate(template.BodyText, item),
		HTMLBody:   s.renderTemplate(template.BodyHTML, item),
		ItemID:     item.ID,
		KB:         item.KB,
		Data: map[string]interface{}{
			"ItemName":    item.Name,
			"KB":          item.KB,
			"ActivatedBy": actor,
			"Reason":      reason,
			"Timestamp":   time.Now().Format(time.RFC3339),
		},
		Channels: template.Channels,
	}

	return s.Send(ctx, notification)
}

// NotifySLABreach sends SLA breach notifications.
func (s *Service) NotifySLABreach(ctx context.Context, item *models.KnowledgeItem, daysPending, slaLimit int) error {
	template := s.templates[NotifySLABreach]
	if template == nil {
		return fmt.Errorf("SLA breach template not found")
	}

	recipients := s.getReviewers(ctx, item.KB, "")
	recipients = append(recipients, s.getApprovers(ctx, item.KB)...)

	notification := &Notification{
		Type:       NotifySLABreach,
		Priority:   PriorityHigh,
		Recipients: recipients,
		Subject:    s.renderTemplate(template.Subject, item),
		Body:       s.renderTemplate(template.BodyText, item),
		HTMLBody:   s.renderTemplate(template.BodyHTML, item),
		ItemID:     item.ID,
		KB:         item.KB,
		Data: map[string]interface{}{
			"ItemName":    item.Name,
			"KB":          item.KB,
			"State":       item.State,
			"DaysPending": daysPending,
			"SLADays":     slaLimit,
		},
		Channels: template.Channels,
	}

	return s.Send(ctx, notification)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (s *Service) saveNotification(ctx context.Context, notification *Notification) error {
	query := `
		INSERT INTO notifications (
			id, type, priority, subject, body, item_id, kb,
			data, created_at, delivery_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	dataJSON, _ := json.Marshal(notification.Data)

	_, err := s.db.ExecContext(ctx, query,
		notification.ID,
		notification.Type,
		notification.Priority,
		notification.Subject,
		notification.Body,
		notification.ItemID,
		notification.KB,
		dataJSON,
		notification.CreatedAt,
		notification.DeliveryStatus,
	)

	return err
}

func (s *Service) updateNotificationStatus(ctx context.Context, notification *Notification) error {
	query := `
		UPDATE notifications
		SET delivery_status = $1, sent_at = $2
		WHERE id = $3
	`

	_, err := s.db.ExecContext(ctx, query,
		notification.DeliveryStatus,
		notification.SentAt,
		notification.ID,
	)

	return err
}

func (s *Service) getReviewers(ctx context.Context, kb models.KB, level string) []Recipient {
	// In production: query user database for KB-specific reviewers
	return []Recipient{}
}

func (s *Service) getApprovers(ctx context.Context, kb models.KB) []Recipient {
	// In production: query user database for CMOs
	return []Recipient{}
}

func (s *Service) getSubscribers(ctx context.Context, kb models.KB) []Recipient {
	// In production: query user database for KB subscribers
	return []Recipient{}
}

func (s *Service) getEmergencyContacts(ctx context.Context) []Recipient {
	// In production: query user database for all CMOs and admins
	return []Recipient{}
}

func (s *Service) renderTemplate(template string, item *models.KnowledgeItem) string {
	result := template
	result = strings.ReplaceAll(result, "{{.ItemName}}", item.Name)
	result = strings.ReplaceAll(result, "{{.KB}}", string(item.KB))
	result = strings.ReplaceAll(result, "{{.RiskLevel}}", string(item.RiskLevel))
	result = strings.ReplaceAll(result, "{{.Version}}", item.Version)
	return result
}

func generateNotificationID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}
