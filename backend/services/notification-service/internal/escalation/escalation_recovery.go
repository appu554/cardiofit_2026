package escalation

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/cardiofit/notification-service/internal/models"
)

// PendingEscalation represents an active escalation that needs recovery
type PendingEscalation struct {
	AlertID         string
	EscalationLevel int
	EscalatedAt     time.Time
	TimeoutMinutes  int
	Severity        string
	PatientID       string
	DepartmentID    string
}

// RecoverPendingEscalations recovers active escalations after service restart
// This ensures that escalations continue even if the service crashes and restarts
func (e *EscalationManager) RecoverPendingEscalations(ctx context.Context) error {
	e.logger.Info("Starting escalation recovery process")

	// Query for active escalations (not acknowledged, created within timeout window)
	pending, err := e.queryPendingEscalations(ctx)
	if err != nil {
		return fmt.Errorf("failed to query pending escalations: %w", err)
	}

	if len(pending) == 0 {
		e.logger.Info("No pending escalations to recover")
		return nil
	}

	e.logger.Info("Found pending escalations to recover",
		zap.Int("count", len(pending)),
	)

	recovered := 0
	executed := 0
	expired := 0

	for _, p := range pending {
		// Calculate elapsed time since escalation
		elapsed := time.Since(p.EscalatedAt)
		timeout := time.Duration(p.TimeoutMinutes) * time.Minute

		if elapsed >= timeout {
			// Timeout already passed - execute escalation immediately
			e.logger.Info("Executing overdue escalation",
				zap.String("alert_id", p.AlertID),
				zap.Int("level", p.EscalationLevel),
				zap.Duration("elapsed", elapsed),
				zap.Duration("timeout", timeout),
			)

			// Check if this is the max level
			if p.EscalationLevel >= e.config.MaxLevel {
				e.logger.Info("Max escalation level already reached",
					zap.String("alert_id", p.AlertID),
					zap.Int("level", p.EscalationLevel),
				)
				expired++
				continue
			}

			// Execute next level escalation
			alert := e.reconstructAlertFromPending(p)
			nextLevel := p.EscalationLevel + 1
			if err := e.escalateToNextLevel(ctx, alert, nextLevel); err != nil {
				e.logger.Error("Failed to execute overdue escalation",
					zap.String("alert_id", p.AlertID),
					zap.Int("next_level", nextLevel),
					zap.Error(err),
				)
				continue
			}
			executed++

			// Schedule next level if not at max
			if nextLevel < e.config.MaxLevel {
				nextTimeout := e.getTimeoutForLevel(nextLevel, e.parseSeverity(p.Severity))
				if err := e.ScheduleEscalation(ctx, alert, nextTimeout); err != nil {
					e.logger.Error("Failed to schedule recovered escalation",
						zap.String("alert_id", p.AlertID),
						zap.Error(err),
					)
				} else {
					recovered++
				}
			}
		} else {
			// Timeout not yet passed - reschedule timer
			remaining := timeout - elapsed
			e.logger.Info("Rescheduling escalation timer",
				zap.String("alert_id", p.AlertID),
				zap.Int("level", p.EscalationLevel),
				zap.Duration("remaining", remaining),
			)

			// Reconstruct alert and reschedule
			alert := e.reconstructAlertFromPending(p)
			if err := e.ScheduleEscalation(ctx, alert, remaining); err != nil {
				e.logger.Error("Failed to reschedule escalation",
					zap.String("alert_id", p.AlertID),
					zap.Error(err),
				)
				continue
			}
			recovered++

			// Update chain state to current level
			e.mu.Lock()
			if chain, ok := e.chains[p.AlertID]; ok {
				chain.CurrentLevel = p.EscalationLevel
			}
			e.mu.Unlock()
		}
	}

	e.logger.Info("Escalation recovery complete",
		zap.Int("total", len(pending)),
		zap.Int("recovered", recovered),
		zap.Int("executed", executed),
		zap.Int("expired", expired),
	)

	return nil
}

// queryPendingEscalations queries the database for active escalations
func (e *EscalationManager) queryPendingEscalations(ctx context.Context) ([]*PendingEscalation, error) {
	// Query escalation_log for active escalations
	// Consider active if: not acknowledged AND escalated within max possible timeout window
	maxWindow := e.config.MaxLevel * e.config.CriticalTimeoutMinutes * 2 // Buffer

	query := `
		SELECT DISTINCT ON (el.alert_id)
			el.alert_id,
			el.escalation_level,
			el.escalated_at,
			EXTRACT(EPOCH FROM (el.metadata->>'timeout_minutes')::interval)/60 AS timeout_minutes,
			n.metadata->>'severity' AS severity,
			n.metadata->>'patient_id' AS patient_id,
			n.metadata->>'department_id' AS department_id
		FROM notification_service.escalation_log el
		JOIN notification_service.notifications n ON n.alert_id = el.alert_id
		WHERE el.acknowledged_at IS NULL
		  AND el.escalated_at >= NOW() - INTERVAL '1 minutes' * $1
		  AND el.outcome IS NULL
		ORDER BY el.alert_id, el.escalation_level DESC, el.escalated_at DESC
	`

	rows, err := e.db.Query(ctx, query, maxWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending escalations: %w", err)
	}
	defer rows.Close()

	var pending []*PendingEscalation
	for rows.Next() {
		p := &PendingEscalation{}
		var timeoutMinutes *int

		err := rows.Scan(
			&p.AlertID,
			&p.EscalationLevel,
			&p.EscalatedAt,
			&timeoutMinutes,
			&p.Severity,
			&p.PatientID,
			&p.DepartmentID,
		)
		if err != nil {
			e.logger.Error("Failed to scan pending escalation",
				zap.Error(err),
			)
			continue
		}

		// Set timeout based on severity if not in metadata
		if timeoutMinutes != nil {
			p.TimeoutMinutes = *timeoutMinutes
		} else {
			severity := e.parseSeverity(p.Severity)
			if severity == models.SeverityCritical {
				p.TimeoutMinutes = e.config.CriticalTimeoutMinutes
			} else {
				p.TimeoutMinutes = e.config.HighTimeoutMinutes
			}
		}

		pending = append(pending, p)
	}

	return pending, nil
}

// reconstructAlertFromPending creates a minimal Alert object from pending escalation data
func (e *EscalationManager) reconstructAlertFromPending(p *PendingEscalation) *models.Alert {
	severity := e.parseSeverity(p.Severity)

	return &models.Alert{
		AlertID:      p.AlertID,
		PatientID:    p.PatientID,
		DepartmentID: p.DepartmentID,
		Severity:     severity,
		AlertType:    models.AlertTypeDeterioration, // Default
		Message:      fmt.Sprintf("Escalation for alert %s", p.AlertID),
		PatientLocation: models.PatientLocation{
			Room: "Unknown", // Would need to fetch from database if needed
		},
		Metadata: models.AlertMetadata{
			RequiresEscalation: true,
		},
		Timestamp: time.Now().Unix(),
	}
}

// parseSeverity parses a severity string to AlertSeverity enum
func (e *EscalationManager) parseSeverity(severityStr string) models.AlertSeverity {
	switch severityStr {
	case "CRITICAL":
		return models.SeverityCritical
	case "HIGH":
		return models.SeverityHigh
	case "MODERATE":
		return models.SeverityModerate
	case "LOW":
		return models.SeverityLow
	default:
		return models.SeverityHigh // Default to HIGH for safety
	}
}

// RecoverFromCheckpoint is an alternative recovery method using checkpoint data
// This is useful if you want to persist escalation state to Redis or a file
func (e *EscalationManager) RecoverFromCheckpoint(ctx context.Context, checkpoint map[string]*EscalationChain) error {
	e.logger.Info("Recovering from checkpoint",
		zap.Int("chains", len(checkpoint)),
	)

	recovered := 0
	for alertID, chain := range checkpoint {
		// Skip if already acknowledged
		if chain.AcknowledgedAt != nil {
			continue
		}

		// Check if we should still escalate
		if chain.CurrentLevel >= e.config.MaxLevel {
			continue
		}

		// Reconstruct alert (minimal version)
		alert := &models.Alert{
			AlertID:      alertID,
			PatientID:    "", // Would need to be stored in checkpoint
			DepartmentID: "", // Would need to be stored in checkpoint
			Severity:     models.SeverityCritical, // Assume critical for recovery
			AlertType:    models.AlertTypeDeterioration,
			Message:      fmt.Sprintf("Recovered escalation for alert %s", alertID),
			Metadata: models.AlertMetadata{
				RequiresEscalation: true,
			},
		}

		// Schedule next escalation
		nextLevel := chain.CurrentLevel + 1
		timeout := e.getTimeoutForLevel(nextLevel, alert.Severity)

		if err := e.ScheduleEscalation(ctx, alert, timeout); err != nil {
			e.logger.Error("Failed to recover escalation from checkpoint",
				zap.String("alert_id", alertID),
				zap.Error(err),
			)
			continue
		}

		// Restore chain state
		e.mu.Lock()
		e.chains[alertID] = chain
		e.mu.Unlock()

		recovered++
	}

	e.logger.Info("Checkpoint recovery complete",
		zap.Int("recovered", recovered),
	)

	return nil
}

// CreateCheckpoint creates a checkpoint of current escalation state
// This can be persisted to Redis or disk for crash recovery
func (e *EscalationManager) CreateCheckpoint() map[string]*EscalationChain {
	e.mu.RLock()
	defer e.mu.RUnlock()

	checkpoint := make(map[string]*EscalationChain, len(e.chains))
	for alertID, chain := range e.chains {
		// Only checkpoint active (unacknowledged) chains
		if chain.AcknowledgedAt == nil {
			// Create a copy to avoid mutation issues
			chainCopy := &EscalationChain{
				AlertID:        chain.AlertID,
				CurrentLevel:   chain.CurrentLevel,
				EscalatedTo:    append([]*models.User{}, chain.EscalatedTo...),
				AcknowledgedBy: chain.AcknowledgedBy,
				AcknowledgedAt: chain.AcknowledgedAt,
				CreatedAt:      chain.CreatedAt,
			}
			checkpoint[alertID] = chainCopy
		}
	}

	e.logger.Debug("Created escalation checkpoint",
		zap.Int("active_chains", len(checkpoint)),
	)

	return checkpoint
}

// GetActiveEscalations returns all currently active escalations
func (e *EscalationManager) GetActiveEscalations() map[string]*EscalationChain {
	e.mu.RLock()
	defer e.mu.RUnlock()

	active := make(map[string]*EscalationChain, len(e.chains))
	for alertID, chain := range e.chains {
		if chain.AcknowledgedAt == nil {
			active[alertID] = chain
		}
	}

	return active
}

// ClearCompletedEscalations removes all acknowledged escalations from memory
func (e *EscalationManager) ClearCompletedEscalations() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	cleared := 0
	for alertID, chain := range e.chains {
		if chain.AcknowledgedAt != nil {
			delete(e.chains, alertID)
			delete(e.timers, alertID)
			cleared++
		}
	}

	e.logger.Info("Cleared completed escalations",
		zap.Int("count", cleared),
	)

	return cleared
}
