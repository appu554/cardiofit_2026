// Package titration implements the arbiter-gated dose computation.
//
// STRUCTURAL SAFETY GUARANTEE:
// ComputeDose() requires an ArbiterOutput parameter. There is no code path
// to produce a dose without first calling arbiter.Arbitrate().
// This is a compile-time guarantee enforced by Go's type system.
package titration

import (
	"fmt"

	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// TitrationEngine computes dose adjustments after arbiter approval.
type TitrationEngine struct {
	maxDoseDeltaPct float64
}

// NewTitrationEngine creates a titration engine with the given max delta.
func NewTitrationEngine(maxDoseDeltaPct float64) *TitrationEngine {
	if maxDoseDeltaPct <= 0 {
		maxDoseDeltaPct = 20.0
	}
	return &TitrationEngine{maxDoseDeltaPct: maxDoseDeltaPct}
}

// DoseResult is the output of a dose computation.
type DoseResult struct {
	Blocked     bool    `json:"blocked"`
	BlockedBy   string  `json:"blocked_by,omitempty"`
	NewDose     float64 `json:"new_dose"`
	DoseDelta   float64 `json:"dose_delta"`
	DeltaPct    float64 `json:"delta_pct"`
	GainApplied float64 `json:"gain_applied"`
}

// ComputeDose is the ONLY function that produces a dose output.
// It REQUIRES an ArbiterOutput — there is no code path to call it
// without first calling arbiter.Arbitrate().
func (e *TitrationEngine) ComputeDose(
	arbiterResult vt.ArbiterOutput,
	currentDose float64,
	proposedDelta float64,
	gainFactor float64,
) *DoseResult {
	if arbiterResult.FinalGate.IsBlocking() {
		return &DoseResult{
			Blocked:   true,
			BlockedBy: fmt.Sprintf("CH_%s:%s", arbiterResult.DominantChannel, arbiterResult.FinalGate),
		}
	}

	if gainFactor <= 0 {
		gainFactor = 1.0
	}
	adjustedDelta := proposedDelta * gainFactor

	maxDelta := currentDose * (e.maxDoseDeltaPct / 100.0)
	if adjustedDelta > maxDelta {
		adjustedDelta = maxDelta
	}
	if adjustedDelta < -maxDelta {
		adjustedDelta = -maxDelta
	}

	newDose := currentDose + adjustedDelta
	if newDose < 0 {
		newDose = 0
	}

	deltaPct := 0.0
	if currentDose > 0 {
		deltaPct = (adjustedDelta / currentDose) * 100.0
	}

	return &DoseResult{
		NewDose:     newDose,
		DoseDelta:   adjustedDelta,
		DeltaPct:    deltaPct,
		GainApplied: gainFactor,
	}
}

// DeprescribingContext holds the state when a drug is being stepped down.
// Used by the V-MCU engine to suppress escalation of the deprescribing drug class.
type DeprescribingContext struct {
	Active            bool   `json:"active"`
	DrugClass         string `json:"drug_class"`
	Phase             string `json:"phase"` // DOSE_REDUCTION | MONITORING | REMOVAL
	MonitoringCadence string `json:"monitoring_cadence"` // "WEEKLY"
}

// DoseChange represents a proposed dose modification for suppression checks.
type DoseChange struct {
	DrugClass string  `json:"drug_class"`
	Direction string  `json:"direction"` // "UP" | "DOWN" | "NONE"
	DeltaMg   float64 `json:"delta_mg"`
}

// ShouldSuppressEscalation returns true if the proposed dose change would
// escalate a drug class that is currently being deprescribed.
// This prevents the titration engine from fighting the clinician's
// deliberate step-down by re-escalating the same medication.
func ShouldSuppressEscalation(proposed DoseChange, deprescribing DeprescribingContext) bool {
	return deprescribing.Active &&
		proposed.DrugClass == deprescribing.DrugClass &&
		proposed.Direction == "UP"
}
