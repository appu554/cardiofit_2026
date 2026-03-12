// Package icu implements the ICU Intelligence Dominance Engine.
//
// This is NOT a Knowledge Base - it is a state-dominance engine that can
// veto everything except reality.
//
// ARCHITECTURE CRITICAL (CTO/CMO Directive):
//   - ICU can override KB-19, KB-18, KB-14, and CQL outputs
//   - ICU cannot override reality
//   - "CQL explains. KB-19 recommends. ICU decides."
//
// Authority Hierarchy:
//   1. Reality (immutable)
//   2. ICU Intelligence (this package)
//   3. KB-18 Governance
//   4. KB-19 Protocol Orchestrator
//   5. CQL Truth Evaluation
package icu

// DominanceState represents the 6 possible ICU dominance states.
// Using string type for better logging, debugging, and serialization.
type DominanceState string

const (
	// StateNone - No ICU dominance, normal workflow proceeds
	StateNone DominanceState = "NONE"

	// StateShock - Hemodynamic instability dominates all decisions
	// Triggers: MAP <65, Lactate >4, Vasopressor requirement
	StateShock DominanceState = "SHOCK"

	// StateHypoxia - Respiratory failure dominates all decisions
	// Triggers: SpO2 <88%, P/F ratio <100, FiO2 >0.6
	StateHypoxia DominanceState = "HYPOXIA"

	// StateActiveBleed - Active hemorrhage dominates all decisions
	// Triggers: Hgb drop >2g/dL/6h, Active transfusion, Surgical bleeding
	StateActiveBleed DominanceState = "ACTIVE_BLEED"

	// StateLowOutputFailure - Cardiogenic/distributive failure dominates
	// Triggers: CI <2.0, ScvO2 <60%, Inotrope escalation
	StateLowOutputFailure DominanceState = "LOW_OUTPUT_FAILURE"

	// StateNeurologicCollapse - CNS crisis dominates all decisions
	// Triggers: GCS <8, Active seizure, ICP >20, Herniation signs
	StateNeurologicCollapse DominanceState = "NEUROLOGIC_COLLAPSE"
)

// Priority returns the clinical priority of this state.
// Higher number = higher priority (evaluated first).
//
// CLINICAL RATIONALE:
//   - Neurologic collapse can cause ALL other states (code blue, herniation)
//   - Shock kills faster than hypoxia (minutes vs hours)
//   - Hypoxia compounds all other states rapidly
//   - Active bleeding must be addressed before optimizing cardiac output
func (s DominanceState) Priority() int {
	switch s {
	case StateNeurologicCollapse:
		return 6 // Highest - brain death/herniation trumps everything
	case StateShock:
		return 5 // Hemodynamic instability kills in minutes
	case StateHypoxia:
		return 4 // Respiratory failure compounds rapidly
	case StateActiveBleed:
		return 3 // Hemorrhage control before optimization
	case StateLowOutputFailure:
		return 2 // Cardiac output failure
	case StateNone:
		return 1 // Normal state - lowest priority
	default:
		return 0 // Unknown state
	}
}

// IsActive returns true if this represents an active dominance state.
func (s DominanceState) IsActive() bool {
	return s != StateNone && s != ""
}

// String returns the string representation of the state.
func (s DominanceState) String() string {
	return string(s)
}

// CanVetoKB19 returns true if this state has authority to veto KB-19 recommendations.
// Per CTO/CMO directive: ICU can always override KB-19.
func (s DominanceState) CanVetoKB19() bool {
	return s.IsActive()
}

// CanVetoKB18 returns true if this state has authority to veto KB-18 governance.
// Per CTO/CMO directive: ICU can override governance in crisis states.
func (s DominanceState) CanVetoKB18() bool {
	return s.IsActive()
}

// RequiresImmediateAttention returns true if this state requires immediate
// clinical intervention and should not wait for workflow approval.
func (s DominanceState) RequiresImmediateAttention() bool {
	switch s {
	case StateNeurologicCollapse, StateShock:
		return true
	default:
		return false
	}
}

// AllStates returns all valid dominance states in priority order (highest first).
func AllStates() []DominanceState {
	return []DominanceState{
		StateNeurologicCollapse, // Priority 6
		StateShock,              // Priority 5
		StateHypoxia,            // Priority 4
		StateActiveBleed,        // Priority 3
		StateLowOutputFailure,   // Priority 2
		StateNone,               // Priority 1
	}
}

// ParseDominanceState converts a string to DominanceState.
// Returns StateNone for unrecognized values.
func ParseDominanceState(s string) DominanceState {
	switch DominanceState(s) {
	case StateNone, StateShock, StateHypoxia, StateActiveBleed,
		StateLowOutputFailure, StateNeurologicCollapse:
		return DominanceState(s)
	default:
		return StateNone
	}
}
