package services

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

// HysteresisEngine implements N-01 asymmetric hysteresis for MCU_GATE transitions.
//
// Gate UPGRADES (more restrictive) are immediate — safety-first.
// Gate DOWNGRADES (less restrictive) require 2+ distinct sessions over 72 hours
// confirming the lower gate before the downgrade takes effect. This prevents
// rapid oscillation between gate states on borderline posterior values.
type HysteresisEngine struct {
	db          *database.Database
	metrics     *metrics.Collector
	log         *zap.Logger
	window      time.Duration // 72h lookback window
	minSessions int           // minimum confirming sessions for downgrade
}

// NewHysteresisEngine creates a HysteresisEngine with N-01 defaults:
// 72-hour lookback window and 2-session minimum for downgrades.
func NewHysteresisEngine(db *database.Database, m *metrics.Collector, log *zap.Logger) *HysteresisEngine {
	return &HysteresisEngine{
		db:          db,
		metrics:     m,
		log:         log,
		window:      72 * time.Hour,
		minSessions: 2,
	}
}

// Apply compares the proposed gate against the current cached gate and returns
// the effective gate after N-01 hysteresis filtering.
//
//   - Upgrade (more restrictive): always immediate.
//   - Same gate: returned unchanged.
//   - Downgrade (less restrictive): allowed only when 2+ distinct sessions in
//     the past 72h computed the same or lower gate level.
//
// Returns (effectiveGate, rationale).
func (h *HysteresisEngine) Apply(
	patientID uuid.UUID,
	currentGate, proposedGate models.MCUGate,
) (models.MCUGate, string) {
	// No change
	if currentGate == proposedGate {
		return proposedGate, ""
	}

	// Upgrade (more restrictive) — immediate, safety-first
	if proposedGate.Level() > currentGate.Level() {
		h.log.Debug("N-01: gate upgrade immediate",
			zap.String("patient_id", patientID.String()),
			zap.String("from", string(currentGate)),
			zap.String("to", string(proposedGate)),
		)
		return proposedGate, "N-01: gate upgrade immediate"
	}

	// Downgrade attempt — need 2+ sessions over 72h confirming the lower gate
	confirmingSessions, err := h.countConfirmingSessions(patientID, proposedGate)
	if err != nil {
		// On error, safety-first: keep the more restrictive gate
		h.log.Error("N-01: hysteresis history query failed, holding current gate",
			zap.String("patient_id", patientID.String()),
			zap.Error(err),
		)
		h.metrics.HysteresisBlocked.Inc()
		return currentGate, "N-01: downgrade blocked — history query error"
	}

	if confirmingSessions >= h.minSessions {
		h.log.Info("N-01: gate downgrade confirmed",
			zap.String("patient_id", patientID.String()),
			zap.String("from", string(currentGate)),
			zap.String("to", string(proposedGate)),
			zap.Int("confirming_sessions", confirmingSessions),
		)
		h.metrics.HysteresisAllowed.Inc()
		return proposedGate, "N-01: downgrade confirmed — 2+ sessions over 72h"
	}

	h.log.Info("N-01: gate downgrade blocked — insufficient confirming sessions",
		zap.String("patient_id", patientID.String()),
		zap.String("current", string(currentGate)),
		zap.String("proposed", string(proposedGate)),
		zap.Int("confirming_sessions", confirmingSessions),
		zap.Int("required", h.minSessions),
	)
	h.metrics.HysteresisBlocked.Inc()
	return currentGate, "N-01: downgrade blocked — insufficient confirming sessions (need 2+ over 72h)"
}

// countConfirmingSessions queries mcu_gate_history for distinct sessions within
// the 72h window where the computed gate was at or below the proposed level.
// A session that computed SAFE also confirms MODIFY (it agreed the patient was
// at or below MODIFY severity).
func (h *HysteresisEngine) countConfirmingSessions(patientID uuid.UUID, proposedGate models.MCUGate) (int, error) {
	// Build list of gate values at or below the proposed level
	gatesAtOrBelow := h.gatesAtOrBelow(proposedGate)
	cutoff := time.Now().Add(-h.window)

	var count int64
	result := h.db.DB.Model(&models.MCUGateHistory{}).
		Where("patient_id = ? AND created_at > ? AND gate_value IN ? AND session_id IS NOT NULL",
			patientID, cutoff, gatesAtOrBelow).
		Distinct("session_id").
		Count(&count)

	if result.Error != nil {
		return 0, result.Error
	}
	return int(count), nil
}

// gatesAtOrBelow returns all MCU gate values with level ≤ the given gate's level.
func (h *HysteresisEngine) gatesAtOrBelow(gate models.MCUGate) []string {
	level := gate.Level()
	allGates := []models.MCUGate{models.GateSafe, models.GateModify, models.GatePause, models.GateHalt}
	var result []string
	for _, g := range allGates {
		if g.Level() <= level {
			result = append(result, string(g))
		}
	}
	return result
}
