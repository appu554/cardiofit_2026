// reentry.go implements Commitment 3: 3-phase re-entry protocol (Phase 8.3).
//
// After a HALT or extended PAUSE, titration resumes through three phases:
//   Phase 1 — MONITORING:    Observe only, no dose changes (2 cycles min)
//   Phase 2 — CONSERVATIVE:  50% of normal max delta (rate-limited)
//   Phase 3 — NORMAL:        Full titration authority restored
//
// The protocol ensures safe re-engagement after safety events.
package titration

// ReentryPhase represents the current phase of the re-entry protocol.
type ReentryPhase string

const (
	ReentryNone         ReentryPhase = "NONE"         // normal operation
	ReentryMonitoring   ReentryPhase = "MONITORING"   // Phase 1: observe only
	ReentryConservative ReentryPhase = "CONSERVATIVE" // Phase 2: reduced deltas
	ReentryNormal       ReentryPhase = "NORMAL"       // Phase 3: full authority
)

// ReentryProtocol manages the 3-phase re-entry after a safety pause.
type ReentryProtocol struct {
	phase             ReentryPhase
	monitoringCycles  int // cycles remaining in monitoring phase
	conservativeCycles int // cycles remaining in conservative phase

	// Configurable thresholds
	MinMonitoringCycles  int     // default: 2
	MinConservCycles     int     // default: 3
	ConservMaxDeltaPct   float64 // default: 50% of normal
}

// NewReentryProtocol creates a protocol with production-safe defaults.
func NewReentryProtocol() *ReentryProtocol {
	return &ReentryProtocol{
		phase:               ReentryNone,
		MinMonitoringCycles: 2,
		MinConservCycles:    3,
		ConservMaxDeltaPct:  0.50,
	}
}

// Activate begins the re-entry protocol after a freeze/resume.
func (rp *ReentryProtocol) Activate() {
	rp.phase = ReentryMonitoring
	rp.monitoringCycles = rp.MinMonitoringCycles
	rp.conservativeCycles = rp.MinConservCycles
}

// Phase returns the current re-entry phase.
func (rp *ReentryProtocol) Phase() ReentryPhase {
	return rp.phase
}

// IsActive returns true if the re-entry protocol is engaged.
func (rp *ReentryProtocol) IsActive() bool {
	return rp.phase != ReentryNone
}

// AllowsDoseChange returns true if the current phase permits dose changes.
func (rp *ReentryProtocol) AllowsDoseChange() bool {
	return rp.phase != ReentryMonitoring
}

// MaxDeltaMultiplier returns the dose delta multiplier for the current phase.
//   MONITORING   → 0.0 (no changes)
//   CONSERVATIVE → ConservMaxDeltaPct (default 0.50)
//   NORMAL/NONE  → 1.0 (full authority)
func (rp *ReentryProtocol) MaxDeltaMultiplier() float64 {
	switch rp.phase {
	case ReentryMonitoring:
		return 0.0
	case ReentryConservative:
		return rp.ConservMaxDeltaPct
	default:
		return 1.0
	}
}

// AdvanceCycle is called after each titration cycle to progress the protocol.
func (rp *ReentryProtocol) AdvanceCycle() {
	switch rp.phase {
	case ReentryMonitoring:
		rp.monitoringCycles--
		if rp.monitoringCycles <= 0 {
			rp.phase = ReentryConservative
		}
	case ReentryConservative:
		rp.conservativeCycles--
		if rp.conservativeCycles <= 0 {
			rp.phase = ReentryNormal
		}
	case ReentryNormal:
		rp.phase = ReentryNone // protocol complete
	}
}

// Reset clears the re-entry protocol (e.g., on a new freeze event).
func (rp *ReentryProtocol) Reset() {
	rp.phase = ReentryNone
	rp.monitoringCycles = 0
	rp.conservativeCycles = 0
}
