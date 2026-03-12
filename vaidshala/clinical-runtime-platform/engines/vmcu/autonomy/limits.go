// Package autonomy implements V-MCU dose change autonomy limits (Task 3.4).
//
// V-MCU must never autonomously exceed certain dose limits without physician
// confirmation. This package enforces:
//   - MaxSingleStepPct:  maximum dose change per cycle (default 20%)
//   - MaxCumulativePct:  maximum cumulative change without confirmation (default 50%)
//   - MaxAbsoluteDoseMg: per-drug-class absolute ceilings
//
// When a limit is breached, the dose is frozen and a
// PHYSICIAN_CONFIRMATION_REQUIRED event should be published by the caller.
package autonomy

import (
	"fmt"
	"math"
	"sync"
)

// LimitResult describes the outcome of an autonomy limit check.
type LimitResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
	LimitID string `json:"limit_id,omitempty"`
}

// AutonomyLimits holds configurable dose change constraints.
type AutonomyLimits struct {
	MaxSingleStepPct  float64            `json:"max_single_step_pct"`  // default: 20.0
	MaxCumulativePct  float64            `json:"max_cumulative_pct"`   // default: 50.0
	MaxAbsoluteDoseMg map[string]float64 `json:"max_absolute_dose_mg"` // drug class → ceiling

	mu              sync.RWMutex
	cumulativeDeltas map[string]float64 // patientID:drugClass → cumulative % change since last confirmation
}

// DefaultAutonomyLimits returns production-safe defaults.
func DefaultAutonomyLimits() *AutonomyLimits {
	return &AutonomyLimits{
		MaxSingleStepPct: 20.0,
		MaxCumulativePct: 50.0,
		MaxAbsoluteDoseMg: map[string]float64{
			"BASAL_INSULIN":  100.0, // units
			"RAPID_INSULIN":  50.0,  // units
			"METFORMIN":      2000.0,
			"SGLT2I":         25.0,
			"DPP4I":          100.0,
			"SULFONYLUREA":   20.0,
		},
		cumulativeDeltas: make(map[string]float64),
	}
}

// CheckLimit validates a proposed dose change against autonomy constraints.
func (l *AutonomyLimits) CheckLimit(
	patientID string,
	currentDose, proposedDose float64,
	drugClass string,
) LimitResult {
	if currentDose <= 0 {
		// Can't compute percentage change from zero/negative base
		return LimitResult{Allowed: true}
	}

	doseDelta := proposedDose - currentDose
	pctChange := math.Abs(doseDelta) / currentDose * 100

	// Check 1: Single-step percentage limit
	if pctChange > l.MaxSingleStepPct {
		return LimitResult{
			Allowed: false,
			Reason: fmt.Sprintf(
				"single-step change %.1f%% exceeds limit %.1f%% (current: %.1f, proposed: %.1f)",
				pctChange, l.MaxSingleStepPct, currentDose, proposedDose,
			),
			LimitID: "AUTONOMY_SINGLE_STEP",
		}
	}

	// Check 2: Absolute dose ceiling
	if ceiling, ok := l.MaxAbsoluteDoseMg[drugClass]; ok {
		if proposedDose > ceiling {
			return LimitResult{
				Allowed: false,
				Reason: fmt.Sprintf(
					"proposed dose %.1f exceeds absolute ceiling %.1f for %s",
					proposedDose, ceiling, drugClass,
				),
				LimitID: "AUTONOMY_ABSOLUTE_CEILING",
			}
		}
	}

	// Check 3: Cumulative percentage limit
	l.mu.RLock()
	key := patientID + ":" + drugClass
	cumulative := l.cumulativeDeltas[key]
	l.mu.RUnlock()

	newCumulative := cumulative + pctChange
	if newCumulative > l.MaxCumulativePct {
		return LimitResult{
			Allowed: false,
			Reason: fmt.Sprintf(
				"cumulative change %.1f%% (adding %.1f%%) exceeds limit %.1f%% — physician confirmation required",
				newCumulative, pctChange, l.MaxCumulativePct,
			),
			LimitID: "AUTONOMY_CUMULATIVE",
		}
	}

	return LimitResult{Allowed: true}
}

// RecordDoseChange records a dose change for cumulative tracking.
// Call this after a dose change is actually applied.
func (l *AutonomyLimits) RecordDoseChange(patientID, drugClass string, currentDose, newDose float64) {
	if currentDose <= 0 {
		return
	}

	pctChange := math.Abs(newDose-currentDose) / currentDose * 100

	l.mu.Lock()
	defer l.mu.Unlock()

	key := patientID + ":" + drugClass
	l.cumulativeDeltas[key] += pctChange
}

// ConfirmByPhysician resets the cumulative tracker for a patient+drug,
// indicating a physician has reviewed and approved the accumulated changes.
func (l *AutonomyLimits) ConfirmByPhysician(patientID, drugClass string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := patientID + ":" + drugClass
	delete(l.cumulativeDeltas, key)
}

// GetCumulativeChange returns the current cumulative change percentage
// for a patient+drug since last physician confirmation.
func (l *AutonomyLimits) GetCumulativeChange(patientID, drugClass string) float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return l.cumulativeDeltas[patientID+":"+drugClass]
}
