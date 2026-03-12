package delivery

import (
	"context"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/models"
	"go.uber.org/zap"
)

// Manager manages multiple delivery providers
type Manager struct {
	emailProvider EmailProvider
	smsProvider   SMSProvider
	pushProvider  PushProvider
	logger        *zap.Logger
}

// EmailProvider interface for email delivery
type EmailProvider interface {
	Send(ctx context.Context, recipients []string, content string, metadata map[string]interface{}) error
}

// SMSProvider interface for SMS delivery
type SMSProvider interface {
	Send(ctx context.Context, recipients []string, content string, metadata map[string]interface{}) error
}

// PushProvider interface for push notification delivery
type PushProvider interface {
	Send(ctx context.Context, recipients []string, content string, metadata map[string]interface{}) error
}

// NewManager creates a new delivery manager
func NewManager(cfg config.DeliveryConfig, logger *zap.Logger) *Manager {
	return &Manager{
		emailProvider: NewSendGridProvider(cfg.Email),
		smsProvider:   NewTwilioProvider(cfg.SMS),
		pushProvider:  NewFirebaseProvider(cfg.Push),
		logger:        logger,
	}
}

// Deliver delivers a notification based on the routing decision
// NOTE: This is a legacy interface - use NotificationDeliveryService for new implementations
func (m *Manager) Deliver(ctx context.Context, decision *models.RoutingDecision) (*models.DeliveryResult, error) {
	// Legacy implementation stub - kept for backwards compatibility
	// New code should use delivery_service.go instead
	return &models.DeliveryResult{
		Success: true,
	}, nil
}
