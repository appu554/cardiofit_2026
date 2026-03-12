package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/models"
)

// SLAConfig defines SLA deadlines by MCU gate severity.
type SLAConfig struct {
	HaltSLA   time.Duration // default: 15 minutes
	PauseSLA  time.Duration // default: 1 hour
	ModifySLA time.Duration // default: 4 hours
}

// DefaultSLAConfig returns production-safe SLA defaults.
func DefaultSLAConfig() SLAConfig {
	return SLAConfig{
		HaltSLA:   15 * time.Minute,
		PauseSLA:  1 * time.Hour,
		ModifySLA: 4 * time.Hour,
	}
}

// SLAScanner periodically scans for ACTIVE decision cards that have exceeded
// their SLA deadline without physician acknowledgement.
//
// When an SLA breach is detected:
//  1. Card is marked as SLA-breached in the database
//  2. Card priority is auto-elevated
//  3. SLA_BREACH event is published to KB-19 for escalation
type SLAScanner struct {
	db       *database.Database
	kb19     *KB19Publisher
	log      *zap.Logger
	cfg      SLAConfig
	interval time.Duration
}

// NewSLAScanner creates a new SLA scanner.
func NewSLAScanner(db *database.Database, kb19 *KB19Publisher, log *zap.Logger, cfg SLAConfig) *SLAScanner {
	return &SLAScanner{
		db:       db,
		kb19:     kb19,
		log:      log,
		cfg:      cfg,
		interval: 1 * time.Minute,
	}
}

// Start begins the background SLA scanning loop.
// Blocks until ctx is cancelled.
func (s *SLAScanner) Start(ctx context.Context) {
	s.log.Info("SLA scanner started",
		zap.Duration("halt_sla", s.cfg.HaltSLA),
		zap.Duration("pause_sla", s.cfg.PauseSLA),
		zap.Duration("modify_sla", s.cfg.ModifySLA),
		zap.Duration("scan_interval", s.interval),
	)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.log.Info("SLA scanner stopped")
			return
		case <-ticker.C:
			if err := s.scanOverdueCards(ctx); err != nil {
				s.log.Error("SLA scan failed", zap.Error(err))
			}
		}
	}
}

// scanOverdueCards finds ACTIVE cards past their SLA deadline and processes them.
func (s *SLAScanner) scanOverdueCards(ctx context.Context) error {
	now := time.Now()

	// Query ACTIVE cards that haven't been breached yet and have exceeded SLA
	var overdueCards []models.DecisionCard
	result := s.db.DB.WithContext(ctx).
		Where("status = ? AND sla_breached = ? AND sla_deadline IS NOT NULL AND sla_deadline < ?",
			models.StatusActive, false, now).
		Find(&overdueCards)

	if result.Error != nil {
		return fmt.Errorf("query overdue cards: %w", result.Error)
	}

	if len(overdueCards) == 0 {
		return nil
	}

	s.log.Info("SLA breaches detected",
		zap.Int("count", len(overdueCards)),
	)

	for i := range overdueCards {
		card := &overdueCards[i]
		if err := s.processBreachedCard(ctx, card, now); err != nil {
			s.log.Error("failed to process SLA breach",
				zap.String("card_id", card.CardID.String()),
				zap.Error(err),
			)
		}
	}

	return nil
}

// processBreachedCard handles a single SLA-breached card.
func (s *SLAScanner) processBreachedCard(ctx context.Context, card *models.DecisionCard, now time.Time) error {
	// Mark as breached
	card.SLABreached = true
	card.SLABreachedAt = &now

	// Auto-elevate: if MODIFY gate, elevate to PAUSE for visibility
	if card.MCUGate == models.GateModify {
		card.MCUGate = models.GatePause
		card.MCUGateRationale = fmt.Sprintf("auto-elevated from MODIFY due to SLA breach at %s", now.Format(time.RFC3339))
	}

	if err := s.db.DB.WithContext(ctx).Save(card).Error; err != nil {
		return fmt.Errorf("update breached card: %w", err)
	}

	// Publish SLA_BREACH event to KB-19
	s.publishSLABreach(card, now)

	s.log.Warn("SLA breach processed",
		zap.String("card_id", card.CardID.String()),
		zap.String("patient_id", card.PatientID.String()),
		zap.String("gate", string(card.MCUGate)),
		zap.Time("sla_deadline", *card.SLADeadline),
		zap.Duration("overdue_by", now.Sub(*card.SLADeadline)),
	)

	return nil
}

// publishSLABreach sends an SLA_BREACH event to KB-19 for escalation.
func (s *SLAScanner) publishSLABreach(card *models.DecisionCard, breachedAt time.Time) {
	event := models.KB19Event{
		EventType: models.EventSLABreach,
		PatientID: card.PatientID,
		SessionID: card.SessionID,
		CardID:    card.CardID,
		Gate:      card.MCUGate,
		Timestamp: breachedAt,
	}

	if err := s.kb19.publishEvent(event); err != nil {
		s.log.Error("SLA_BREACH publish to KB-19 failed",
			zap.String("card_id", card.CardID.String()),
			zap.Error(err),
		)
	}
}

// ComputeSLADeadline calculates the SLA deadline for a new card based on
// its MCU gate severity.
func (s *SLAScanner) ComputeSLADeadline(gate models.MCUGate, createdAt time.Time) time.Time {
	switch gate {
	case models.GateHalt:
		return createdAt.Add(s.cfg.HaltSLA)
	case models.GatePause:
		return createdAt.Add(s.cfg.PauseSLA)
	case models.GateModify:
		return createdAt.Add(s.cfg.ModifySLA)
	default:
		// SAFE cards have no SLA (no action required)
		return createdAt.Add(24 * time.Hour)
	}
}
