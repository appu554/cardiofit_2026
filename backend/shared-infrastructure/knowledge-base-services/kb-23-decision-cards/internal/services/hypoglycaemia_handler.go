package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

type HypoglycaemiaHandler struct {
	cfg       *config.Config
	db        *database.Database
	gateCache *MCUGateCache
	kb19      *KB19Publisher
	metrics   *metrics.Collector
	log       *zap.Logger
}

func NewHypoglycaemiaHandler(
	cfg *config.Config,
	db *database.Database,
	gc *MCUGateCache,
	kb19 *KB19Publisher,
	m *metrics.Collector,
	log *zap.Logger,
) *HypoglycaemiaHandler {
	return &HypoglycaemiaHandler{
		cfg: cfg, db: db, gateCache: gc, kb19: kb19, metrics: m, log: log,
	}
}

// HandleAlert processes a hypoglycaemia alert (V-08 fast-path).
// Phase 2 minimal: immediately writes gate based on severity.
//
//	SEVERE   -> HALT
//	MODERATE -> PAUSE
//	MILD     -> MODIFY
func (h *HypoglycaemiaHandler) HandleAlert(ctx context.Context, req *models.SafetyAlertRequest) (*models.DecisionCard, error) {
	// Determine severity and gate
	severity := h.classifySeverity(req.GlucoseMmolL)
	gate := h.severityToGate(severity)

	haltSource := models.HaltMeasured
	if req.Source == string(models.HypoSourceVMCUPredicted) {
		haltSource = models.HaltPredicted
	}

	// Record the alert
	alert := models.HypoglycaemiaAlert{
		AlertID:        uuid.New(),
		PatientID:      req.PatientID,
		Source:         models.HypoglycaemiaSource(req.Source),
		GlucoseMmolL:  req.GlucoseMmolL,
		Severity:       severity,
		HaltSource:     haltSource,
		EventTimestamp: req.Timestamp,
		ProcessedAt:    time.Now(),
	}

	if req.DurationMinutes > 0 {
		dur := req.DurationMinutes
		alert.DurationMinutes = &dur
	}
	if req.PredictedAtHours > 0 {
		alert.PredictedAtHours = &req.PredictedAtHours
	}

	// Create fast-path decision card
	notes := fmt.Sprintf("HYPOGLYCAEMIA: glucose %.1f mmol/L, source=%s, severity=%s",
		req.GlucoseMmolL, req.Source, severity)

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                req.PatientID,
		TemplateID:               "CT_HYPOGLYCAEMIA_DEVICE_DETECTED",
		NodeID:                   "CROSS_NODE",
		PrimaryDifferentialID:    "HYPOGLYCAEMIA",
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  gate,
		MCUGateRationale:         notes,
		DoseAdjustmentNotes:      &notes,
		ObservationReliability:   models.ReliabilityHigh,
		SafetyTier:               models.SafetyImmediate,
		CardSource:               models.SourceHypoglycaemiaFast,
		Status:                   models.StatusActive,
		ClinicianSummary:         notes,
		PatientSummaryEn:         "Low blood sugar detected. Your medication dosing has been paused for safety.",
		PatientSummaryHi:         "\u0915\u092e \u0930\u0915\u094d\u0924 \u0936\u0930\u094d\u0915\u0930\u093e \u0915\u093e \u092a\u0924\u093e \u091a\u0932\u093e\u0964 \u0906\u092a\u0915\u0940 \u0926\u0935\u093e \u0915\u0940 \u0916\u0941\u0930\u093e\u0915 \u0938\u0941\u0930\u0915\u094d\u0937\u093e \u0915\u0947 \u0932\u093f\u090f \u0930\u094b\u0915 \u0926\u0940 \u0917\u0908 \u0939\u0948\u0964",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}

	// Save alert
	if err := h.db.DB.Create(&alert).Error; err != nil {
		return nil, fmt.Errorf("save hypoglycaemia alert: %w", err)
	}

	// Save card
	alert.GeneratedCardID = &card.CardID
	if err := h.db.DB.Create(card).Error; err != nil {
		return nil, fmt.Errorf("save fast-path card: %w", err)
	}

	// Update alert with card ID
	h.db.DB.Model(&alert).Update("generated_card_id", card.CardID)

	// Write gate to cache
	if err := h.gateCache.WriteGate(card); err != nil {
		h.log.Error("gate cache write failed for hypo alert", zap.Error(err))
	}

	// Publish to KB-19 (async)
	go h.kb19.PublishGateChanged(card)

	h.log.Warn("hypoglycaemia fast-path card generated",
		zap.String("card_id", card.CardID.String()),
		zap.String("gate", string(gate)),
		zap.String("severity", string(severity)),
		zap.String("source", req.Source),
	)

	return card, nil
}

func (h *HypoglycaemiaHandler) classifySeverity(glucoseMmolL float64) models.HypoglycaemiaSeverity {
	if glucoseMmolL <= h.cfg.HypoglycaemiaSevereThreshold {
		return models.HypoSevere
	}
	if glucoseMmolL <= h.cfg.HypoglycaemiaModerateThreshold {
		return models.HypoModerate
	}
	return models.HypoMild
}

func (h *HypoglycaemiaHandler) severityToGate(severity models.HypoglycaemiaSeverity) models.MCUGate {
	switch severity {
	case models.HypoSevere:
		return models.GateHalt
	case models.HypoModerate:
		return models.GatePause
	default:
		return models.GateModify
	}
}
