package fatigue

import (
	"context"
	"fmt"
	"time"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/database"
	"go.uber.org/zap"
)

// Manager manages notification fatigue prevention
type Manager struct {
	redis  *database.RedisClient
	config config.FatigueConfig
	logger *zap.Logger
}

// NewManager creates a new fatigue manager
func NewManager(redis *database.RedisClient, cfg config.FatigueConfig, logger *zap.Logger) *Manager {
	return &Manager{
		redis:  redis,
		config: cfg,
		logger: logger,
	}
}

// ShouldSuppress determines if a notification should be suppressed due to fatigue
func (m *Manager) ShouldSuppress(ctx context.Context, recipient string, priority string) (bool, error) {
	if !m.config.Enabled {
		return false, nil
	}

	// Check if we're in quiet hours
	if m.isQuietHours() && priority != "critical" {
		m.logger.Info("Suppressing notification during quiet hours",
			zap.String("recipient", recipient),
			zap.String("priority", priority),
		)
		return true, nil
	}

	// Check notification count in window
	key := fmt.Sprintf("fatigue:%s", recipient)
	count, err := m.redis.Incr(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to increment fatigue counter: %w", err)
	}

	// Set expiry on first increment
	if count == 1 {
		if err := m.redis.Expire(ctx, key, m.config.WindowDuration); err != nil {
			m.logger.Error("Failed to set expiry on fatigue key", zap.Error(err))
		}
	}

	// Check if limit exceeded
	if count > int64(m.config.MaxNotifications) {
		m.logger.Info("Notification limit exceeded",
			zap.String("recipient", recipient),
			zap.Int64("count", count),
			zap.Int("max", m.config.MaxNotifications),
		)
		return true, nil
	}

	return false, nil
}

func (m *Manager) isQuietHours() bool {
	now := time.Now()
	currentTime := now.Format("15:04")

	start := m.config.QuietHoursStart
	end := m.config.QuietHoursEnd

	if start > end {
		// Quiet hours span midnight
		return currentTime >= start || currentTime <= end
	}

	return currentTime >= start && currentTime <= end
}
