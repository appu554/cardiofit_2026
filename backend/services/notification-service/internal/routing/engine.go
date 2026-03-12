package routing

import (
	"context"
	"fmt"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/fatigue"
	"github.com/cardiofit/notification-service/internal/models"
	"go.uber.org/zap"
)

// Engine handles notification routing logic
type Engine struct {
	config         config.RoutingConfig
	fatigueManager *fatigue.Manager
	logger         *zap.Logger
}

// NewEngine creates a new routing engine
func NewEngine(cfg config.RoutingConfig, fatigueManager *fatigue.Manager, logger *zap.Logger) *Engine {
	return &Engine{
		config:         cfg,
		fatigueManager: fatigueManager,
		logger:         logger,
	}
}

// Route determines the appropriate channel and recipients for an alert
func (e *Engine) Route(ctx context.Context, alert *models.ClinicalAlert) (*models.RoutingDecision, error) {
	// Check fatigue management
	if e.fatigueManager != nil {
		for _, recipient := range alert.Recipients {
			if shouldSuppress, err := e.fatigueManager.ShouldSuppress(ctx, recipient, alert.Priority); err != nil {
				e.logger.Error("Error checking fatigue", zap.Error(err))
			} else if shouldSuppress {
				e.logger.Info("Suppressing notification due to fatigue",
					zap.String("recipient", recipient),
					zap.String("alert_id", alert.ID),
				)
				return nil, fmt.Errorf("notification suppressed due to fatigue")
			}
		}
	}

	// Select channel based on priority and preferences
	channel := e.selectChannel(alert)

	decision := &models.RoutingDecision{
		AlertID:       alert.ID,
		Channel:       channel,
		Recipients:    alert.Recipients,
		Content:       alert.Message,
		Priority:      alert.Priority,
		Metadata:      alert.Metadata,
		RetryAttempts: e.config.RetryAttempts,
	}

	return decision, nil
}

func (e *Engine) selectChannel(alert *models.ClinicalAlert) string {
	// Priority-based channel selection
	switch alert.Priority {
	case "critical":
		return "sms"
	case "high":
		return "push"
	case "medium":
		return "email"
	default:
		return e.config.DefaultChannel
	}
}
