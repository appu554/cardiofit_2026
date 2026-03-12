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

type BehavioralGapHandler struct {
	db        *database.Database
	gateCache *MCUGateCache
	kb19      *KB19Publisher
	metrics   *metrics.Collector
	log       *zap.Logger
}

func NewBehavioralGapHandler(
	db *database.Database,
	gc *MCUGateCache,
	kb19 *KB19Publisher,
	m *metrics.Collector,
	log *zap.Logger,
) *BehavioralGapHandler {
	return &BehavioralGapHandler{db: db, gateCache: gc, kb19: kb19, metrics: m, log: log}
}

// HandleAlert processes a behavioral gap alert from KB-21 (G-01).
// Phase 2 minimal handler:
//
//	BEHAVIORAL_GAP -> MODIFY gate (do not intensify medication)
//	DISCORDANT     -> SAFE gate (medication review needed)
func (h *BehavioralGapHandler) HandleAlert(ctx context.Context, req *models.SafetyAlertRequest) (*models.DecisionCard, error) {
	var gate models.MCUGate
	var notes string
	var safetyTier models.SafetyTier

	switch req.TreatmentResponseClass {
	case "BEHAVIORAL_GAP":
		gate = models.GateModify
		notes = fmt.Sprintf("BEHAVIORAL_GAP: Do not intensify medication. Adherence=%.2f, HbA1c delta=%.2f. Adherence is the primary problem.",
			req.MeanAdherenceScore, req.HbA1cDelta)
		safetyTier = models.SafetyUrgent
	case "DISCORDANT":
		gate = models.GateSafe
		notes = fmt.Sprintf("MEDICATION_REVIEW: High adherence (%.2f) with no clinical improvement (HbA1c delta=%.2f). Consider medication class change.",
			req.MeanAdherenceScore, req.HbA1cDelta)
		safetyTier = models.SafetyRoutine
	default:
		return nil, fmt.Errorf("unsupported treatment_response_class: %s", req.TreatmentResponseClass)
	}

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                req.PatientID,
		TemplateID:               "CT_BEHAVIORAL_GAP",
		NodeID:                   "BEHAVIORAL",
		PrimaryDifferentialID:    req.TreatmentResponseClass,
		DiagnosticConfidenceTier: models.TierFirm, // KB-21 classification is definitive
		MCUGate:                  gate,
		MCUGateRationale:         notes,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityModerate,
		SafetyTier:               safetyTier,
		CardSource:               models.SourceBehavioralGap,
		Status:                   models.StatusActive,
		ClinicianSummary:         notes,
		PatientSummaryEn:         "Your treatment team is reviewing your medication based on recent health data.",
		PatientSummaryHi:         "\u0906\u092a\u0915\u0940 \u0909\u092a\u091a\u093e\u0930 \u091f\u0940\u092e \u0939\u093e\u0932 \u0915\u0947 \u0938\u094d\u0935\u093e\u0938\u094d\u0925\u094d\u092f \u0921\u0947\u091f\u093e \u0915\u0947 \u0906\u0927\u093e\u0930 \u092a\u0930 \u0906\u092a\u0915\u0940 \u0926\u0935\u093e \u0915\u0940 \u0938\u092e\u0940\u0915\u094d\u0937\u093e \u0915\u0930 \u0930\u0939\u0940 \u0939\u0948\u0964",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	if err := h.db.DB.Create(card).Error; err != nil {
		return nil, fmt.Errorf("save behavioral gap card: %w", err)
	}

	// Write gate to cache
	if err := h.gateCache.WriteGate(card); err != nil {
		h.log.Error("gate cache write failed for behavioral gap", zap.Error(err))
	}

	// Publish to KB-19
	go h.kb19.PublishGateChanged(card)

	h.log.Info("behavioral gap card generated",
		zap.String("card_id", card.CardID.String()),
		zap.String("response_class", req.TreatmentResponseClass),
		zap.String("gate", string(gate)),
	)

	return card, nil
}
