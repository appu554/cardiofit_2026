package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

// CardLifecycle manages card state transitions (Phase 4).
type CardLifecycle struct {
	db        *database.Database
	gateCache *MCUGateCache
	kb19      *KB19Publisher
	log       *zap.Logger
}

func NewCardLifecycle(db *database.Database, gc *MCUGateCache, kb19 *KB19Publisher, log *zap.Logger) *CardLifecycle {
	return &CardLifecycle{db: db, gateCache: gc, kb19: kb19, log: log}
}

// SupersedeExistingCards marks all ACTIVE cards for the given patient+node as
// SUPERSEDED, except for the newly created card identified by newCardID.
func (s *CardLifecycle) SupersedeExistingCards(ctx context.Context, patientID uuid.UUID, nodeID string, newCardID uuid.UUID) error {
	var existing []models.DecisionCard
	result := s.db.DB.WithContext(ctx).
		Where("patient_id = ? AND node_id = ? AND status = ? AND card_id != ?",
			patientID, nodeID, models.StatusActive, newCardID).
		Find(&existing)
	if result.Error != nil {
		return fmt.Errorf("query existing active cards: %w", result.Error)
	}

	now := time.Now()
	for i := range existing {
		card := &existing[i]
		card.Status = models.StatusSuperseded
		card.SupersededAt = &now
		card.SupersededBy = &newCardID

		if err := s.db.DB.WithContext(ctx).Save(card).Error; err != nil {
			return fmt.Errorf("supersede card %s: %w", card.CardID.String(), err)
		}

		s.log.Info("card superseded",
			zap.String("superseded_card_id", card.CardID.String()),
			zap.String("superseded_by", newCardID.String()),
			zap.String("patient_id", patientID.String()),
			zap.String("node_id", nodeID),
		)
	}

	return nil
}

// ResumeGate allows a clinician to resume a card whose MCU gate is PAUSE or
// HALT. The gate transitions to MODIFY (never directly to SAFE). The change is
// persisted, written to the MCU gate cache, and published to KB-19.
func (s *CardLifecycle) ResumeGate(ctx context.Context, cardID uuid.UUID, clinicianID string) error {
	var card models.DecisionCard
	if err := s.db.DB.WithContext(ctx).Where("card_id = ?", cardID).First(&card).Error; err != nil {
		return fmt.Errorf("card not found: %w", err)
	}

	if card.Status != models.StatusActive {
		return fmt.Errorf("card %s is not ACTIVE (current status: %s)", cardID.String(), card.Status)
	}

	if card.MCUGate != models.GatePause && card.MCUGate != models.GateHalt {
		return fmt.Errorf("card %s gate is %s; only PAUSE or HALT can be resumed", cardID.String(), card.MCUGate)
	}

	card.MCUGate = models.GateModify
	card.MCUGateRationale = fmt.Sprintf("clinician resume by %s", clinicianID)

	if err := s.db.DB.WithContext(ctx).Save(&card).Error; err != nil {
		return fmt.Errorf("update card gate: %w", err)
	}

	if err := s.gateCache.WriteGate(&card); err != nil {
		s.log.Error("gate cache write failed after resume",
			zap.String("card_id", cardID.String()),
			zap.Error(err),
		)
	}

	s.kb19.PublishGateChanged(&card)

	s.log.Info("gate resumed to MODIFY",
		zap.String("card_id", cardID.String()),
		zap.String("clinician_id", clinicianID),
	)

	return nil
}

// ArchiveCard sets the card status to ARCHIVED.
func (s *CardLifecycle) ArchiveCard(ctx context.Context, cardID uuid.UUID) error {
	result := s.db.DB.WithContext(ctx).
		Model(&models.DecisionCard{}).
		Where("card_id = ?", cardID).
		Update("status", models.StatusArchived)
	if result.Error != nil {
		return fmt.Errorf("archive card %s: %w", cardID.String(), result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("card %s not found", cardID.String())
	}
	return nil
}

// MarkPendingReaffirmation transitions an ACTIVE card into
// PENDING_REAFFIRMATION status, signalling that the hysteresis window suggests
// the clinical decision should be re-checked.
func (s *CardLifecycle) MarkPendingReaffirmation(ctx context.Context, cardID uuid.UUID) error {
	result := s.db.DB.WithContext(ctx).
		Model(&models.DecisionCard{}).
		Where("card_id = ?", cardID).
		Updates(map[string]interface{}{
			"status":                 models.StatusPendingReaffirmation,
			"pending_reaffirmation":  true,
		})
	if result.Error != nil {
		return fmt.Errorf("mark pending reaffirmation for card %s: %w", cardID.String(), result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("card %s not found", cardID.String())
	}
	return nil
}

// Reaffirm transitions a PENDING_REAFFIRMATION card back to ACTIVE,
// indicating the clinician or a new session has confirmed the original decision.
func (s *CardLifecycle) Reaffirm(ctx context.Context, cardID uuid.UUID) error {
	result := s.db.DB.WithContext(ctx).
		Model(&models.DecisionCard{}).
		Where("card_id = ?", cardID).
		Updates(map[string]interface{}{
			"status":                models.StatusActive,
			"pending_reaffirmation": false,
		})
	if result.Error != nil {
		return fmt.Errorf("reaffirm card %s: %w", cardID.String(), result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("card %s not found", cardID.String())
	}
	return nil
}

// ---------------------------------------------------------------------------
// AD-04: Dose-Halving State Machine
// ---------------------------------------------------------------------------
//
// State transitions:
//   DOSE_REDUCTION_PROPOSED → (physician approves) →
//   DOSE_REDUCTION_APPROVED → (dose halved) →
//   MONITORING → (monitoring window expires) →
//     if BP AT_TARGET → REMOVAL_PROPOSED → (physician approves) → REMOVED
//     if BP ABOVE_TARGET → STEP_DOWN_FAILED → re-escalation card
//
// ACEi/ARB is special: Step 2 is dose-reduce only, never full removal.
// ---------------------------------------------------------------------------

// TransitionToMonitoring moves a deprescribing card from DOSE_REDUCTION to
// MONITORING after the physician-approved dose halving has been applied.
func (s *CardLifecycle) TransitionToMonitoring(ctx context.Context, cardID uuid.UUID) error {
	var card models.DecisionCard
	if err := s.db.DB.WithContext(ctx).Where("card_id = ?", cardID).First(&card).Error; err != nil {
		return fmt.Errorf("card not found: %w", err)
	}

	if card.DeprescribingPhase != string(models.DeprescribingDoseReduction) {
		return fmt.Errorf("card %s is not in DOSE_REDUCTION phase (current: %s)",
			cardID.String(), card.DeprescribingPhase)
	}

	now := time.Now()
	monitoringWeeks := MonitoringWindowConfig(card.DeprescribingDrugClass)

	card.DeprescribingPhase = string(models.DeprescribingMonitoring)
	card.MonitoringStartDate = &now
	card.MonitoringWindowWeeks = monitoringWeeks

	if err := s.db.DB.WithContext(ctx).Save(&card).Error; err != nil {
		return fmt.Errorf("transition card %s to MONITORING: %w", cardID.String(), err)
	}

	s.log.Info("AD-04: deprescribing card transitioned to MONITORING",
		zap.String("card_id", cardID.String()),
		zap.String("drug_class", card.DeprescribingDrugClass),
		zap.Int("monitoring_weeks", monitoringWeeks),
	)

	return nil
}

// ---------------------------------------------------------------------------
// AD-05: Per-Class Monitoring Windows + Failure Thresholds
// ---------------------------------------------------------------------------

// MonitoringWindowConfig returns the monitoring duration (weeks) for a given
// antihypertensive drug class after dose halving.
//
// Per-class step-down sequences:
//   - Thiazide (HCTZ 25mg → 12.5mg): 4 weeks monitoring, then remove
//   - CCB (amlodipine 10mg → 5mg): 4 weeks monitoring, then remove
//   - Beta-blocker (half dose): 6 weeks monitoring, then remove
//   - ACEi/ARB (one dose step down): 6 weeks + ACR recheck, dose-reduce only
func MonitoringWindowConfig(drugClass string) int {
	switch drugClass {
	case "THIAZIDE", "CCB":
		return 4
	case "BETA_BLOCKER", "ACE_INHIBITOR", "ARB":
		return 6
	default:
		return 6 // conservative default
	}
}

// IsACEiARBClass returns true if the drug class is an ACEi or ARB, which
// have special deprescribing rules (never full removal, ACR monitoring).
func IsACEiARBClass(drugClass string) bool {
	return drugClass == "ACE_INHIBITOR" || drugClass == "ARB"
}

// EvaluateMonitoringOutcome checks BP status at monitoring window expiry.
//
// Decision logic:
//   - bpStatus "AT_TARGET" → generate REMOVAL_PROPOSED card (Step 2).
//     Exception: ACEi/ARB cards do not proceed to full removal; they
//     remain at the reduced dose and the card is archived as successful.
//   - bpStatus "ABOVE_TARGET" → close deprescribing card as STEP_DOWN_FAILED,
//     and the caller should generate a re-escalation card via GetReEscalationSpec.
func (s *CardLifecycle) EvaluateMonitoringOutcome(
	ctx context.Context,
	card *models.DecisionCard,
	bpStatus string,
) error {
	if card.DeprescribingPhase != string(models.DeprescribingMonitoring) {
		return fmt.Errorf("card %s is not in MONITORING phase (current: %s)",
			card.CardID.String(), card.DeprescribingPhase)
	}

	switch bpStatus {
	case "AT_TARGET":
		if IsACEiARBClass(card.DeprescribingDrugClass) {
			// ACEi/ARB: dose-reduce only, never full removal. Archive as success.
			card.DeprescribingPhase = string(models.DeprescribingRemoval)
			card.Status = models.StatusArchived
			s.log.Info("AD-05: ACEi/ARB step-down successful — dose remains reduced (no full removal)",
				zap.String("card_id", card.CardID.String()),
				zap.String("drug_class", card.DeprescribingDrugClass))
		} else {
			// Standard classes: propose full removal (Step 2).
			card.DeprescribingPhase = string(models.DeprescribingRemoval)
			s.log.Info("AD-05: monitoring outcome AT_TARGET — REMOVAL_PROPOSED",
				zap.String("card_id", card.CardID.String()),
				zap.String("drug_class", card.DeprescribingDrugClass))
		}

	case "ABOVE_TARGET":
		card.DeprescribingPhase = string(models.DeprescribingFailed)
		card.Status = models.StatusArchived
		s.log.Warn("AD-05: monitoring outcome ABOVE_TARGET — STEP_DOWN_FAILED",
			zap.String("card_id", card.CardID.String()),
			zap.String("drug_class", card.DeprescribingDrugClass))

	default:
		return fmt.Errorf("unrecognised bpStatus %q; expected AT_TARGET or ABOVE_TARGET", bpStatus)
	}

	if err := s.db.DB.WithContext(ctx).Save(card).Error; err != nil {
		return fmt.Errorf("save monitoring outcome for card %s: %w", card.CardID.String(), err)
	}

	return nil
}

// IsMonitoringWindowExpired returns true if the monitoring window has elapsed.
func IsMonitoringWindowExpired(card *models.DecisionCard) bool {
	if card.MonitoringStartDate == nil || card.MonitoringWindowWeeks == 0 {
		return false
	}
	expiry := card.MonitoringStartDate.Add(time.Duration(card.MonitoringWindowWeeks) * 7 * 24 * time.Hour)
	return time.Now().After(expiry)
}

// CompositeCardService handles 72-hour card synthesis (Phase 4).
type CompositeCardService struct {
	db      *database.Database
	metrics *metrics.Collector
	log     *zap.Logger
}

func NewCompositeCardService(db *database.Database, m *metrics.Collector, log *zap.Logger) *CompositeCardService {
	return &CompositeCardService{db: db, metrics: m, log: log}
}
