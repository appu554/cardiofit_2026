package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationChannel is the interface for sending escalation notifications.
type NotificationChannel interface {
	Send(notification EscalationNotification) (DeliveryResult, error)
	Name() string
}

// EscalationNotification carries the payload for a single notification delivery.
type EscalationNotification struct {
	EscalationID    string
	PatientID       string
	PatientName     string
	ClinicianID     string
	ClinicianPhone  string
	Tier            string
	PrimaryReason   string
	SuggestedAction string
	Timeframe       string
	CardID          string
}

// DeliveryResult captures the outcome of a notification delivery attempt.
type DeliveryResult struct {
	Status    string // SENT, FAILED, PENDING
	MessageID string
	Channel   string
	SentAt    time.Time
}

// ---------------------------------------------------------------------------
// NoopChannel — logs via zap, returns SENT immediately (testing/dev)
// ---------------------------------------------------------------------------

// NoopChannel is a no-op notification channel that logs and returns success.
type NoopChannel struct {
	logger *zap.Logger
}

// NewNoopChannel creates a NoopChannel with the given logger.
func NewNoopChannel(logger *zap.Logger) *NoopChannel {
	return &NoopChannel{logger: logger}
}

func (c *NoopChannel) Name() string { return "noop" }

func (c *NoopChannel) Send(n EscalationNotification) (DeliveryResult, error) {
	c.logger.Info("noop channel: notification logged",
		zap.String("escalation_id", n.EscalationID),
		zap.String("patient_id", n.PatientID),
		zap.String("tier", n.Tier),
		zap.String("reason", n.PrimaryReason),
	)
	return DeliveryResult{
		Status:    "SENT",
		MessageID: uuid.New().String(),
		Channel:   c.Name(),
		SentAt:    time.Now(),
	}, nil
}

// ---------------------------------------------------------------------------
// SMSChannelStub — logs SMS body (max 160 chars), returns SENT
// ---------------------------------------------------------------------------

// SMSChannelStub simulates SMS delivery by logging the truncated message body.
type SMSChannelStub struct {
	logger *zap.Logger
}

// NewSMSChannelStub creates an SMSChannelStub with the given logger.
func NewSMSChannelStub(logger *zap.Logger) *SMSChannelStub {
	return &SMSChannelStub{logger: logger}
}

func (c *SMSChannelStub) Name() string { return "sms" }

func (c *SMSChannelStub) Send(n EscalationNotification) (DeliveryResult, error) {
	body := fmt.Sprintf("[%s] %s — %s. Action: %s", n.Tier, n.PatientName, n.PrimaryReason, n.SuggestedAction)
	if len(body) > 160 {
		body = body[:157] + "..."
	}
	c.logger.Info("sms stub: sending",
		zap.String("to", n.ClinicianPhone),
		zap.String("body", body),
		zap.Int("body_len", len(body)),
	)
	return DeliveryResult{
		Status:    "SENT",
		MessageID: uuid.New().String(),
		Channel:   c.Name(),
		SentAt:    time.Now(),
	}, nil
}

// ---------------------------------------------------------------------------
// WhatsAppChannelStub — logs WhatsApp template params, returns SENT
// ---------------------------------------------------------------------------

// WhatsAppChannelStub simulates WhatsApp delivery by logging template parameters.
type WhatsAppChannelStub struct {
	logger *zap.Logger
}

// NewWhatsAppChannelStub creates a WhatsAppChannelStub with the given logger.
func NewWhatsAppChannelStub(logger *zap.Logger) *WhatsAppChannelStub {
	return &WhatsAppChannelStub{logger: logger}
}

func (c *WhatsAppChannelStub) Name() string { return "whatsapp" }

func (c *WhatsAppChannelStub) Send(n EscalationNotification) (DeliveryResult, error) {
	c.logger.Info("whatsapp stub: sending template",
		zap.String("to", n.ClinicianPhone),
		zap.String("template", "escalation_alert_v1"),
		zap.String("param_patient", n.PatientName),
		zap.String("param_tier", n.Tier),
		zap.String("param_reason", n.PrimaryReason),
		zap.String("param_action", n.SuggestedAction),
		zap.String("param_timeframe", n.Timeframe),
	)
	return DeliveryResult{
		Status:    "SENT",
		MessageID: uuid.New().String(),
		Channel:   c.Name(),
		SentAt:    time.Now(),
	}, nil
}

// ---------------------------------------------------------------------------
// PushChannelStub — logs push payload, returns SENT
// ---------------------------------------------------------------------------

// PushChannelStub simulates push notification delivery by logging the payload.
type PushChannelStub struct {
	logger *zap.Logger
}

// NewPushChannelStub creates a PushChannelStub with the given logger.
func NewPushChannelStub(logger *zap.Logger) *PushChannelStub {
	return &PushChannelStub{logger: logger}
}

func (c *PushChannelStub) Name() string { return "push" }

func (c *PushChannelStub) Send(n EscalationNotification) (DeliveryResult, error) {
	c.logger.Info("push stub: sending notification",
		zap.String("clinician_id", n.ClinicianID),
		zap.String("title", fmt.Sprintf("[%s] Escalation for %s", n.Tier, n.PatientName)),
		zap.String("body", n.PrimaryReason),
		zap.String("action", n.SuggestedAction),
		zap.String("card_id", n.CardID),
	)
	return DeliveryResult{
		Status:    "SENT",
		MessageID: uuid.New().String(),
		Channel:   c.Name(),
		SentAt:    time.Now(),
	}, nil
}
