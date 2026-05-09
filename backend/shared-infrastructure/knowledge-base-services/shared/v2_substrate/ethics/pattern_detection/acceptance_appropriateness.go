// Package pattern_detection provides pure-function statistical and heuristic
// detectors for the continuous ethics-based auditing layer (Guidelines §10).
// Each detector is a side-effect-free mathematical primitive consumed by the
// ethics-monitoring service's daily/weekly detection workers.
//
// All functions in this package are safe for concurrent use; they carry no
// mutable state and perform no I/O.
//
// VisibilityClass: AD
package pattern_detection

// RuleSnapshot captures the acceptance and appropriateness metrics for a
// single clinical rule at a specific point in time (typically the start or end
// of a 30-day rolling window).
type RuleSnapshot struct {
	// RuleID identifies the clinical rule these metrics belong to.
	RuleID string

	// AcceptanceRate is the fraction of recommendations that were accepted by
	// the clinician, in the range [0, 1].
	AcceptanceRate float64

	// AppropriatenessMean is the mean appropriateness score assigned by
	// clinicians for this rule's recommendations. The scale is defined by the
	// craft engine (typically 1.0–5.0).
	AppropriatenessMean float64
}

// DetectDivergence flags acceptance-appropriateness divergence per Guidelines
// §1 Principle 2 and §10 daily detection.
//
// It returns true when acceptance rises by ≥ thresholdPP percentage points
// between prior and current without a parallel rise in appropriateness mean.
// A "parallel rise" is defined as ΔAppropriatenessMean ≥ 0.3; smaller deltas
// (including flat or declining appropriateness) indicate that acceptance
// growth is decoupled from perceived clinical quality.
//
// The detector is deliberately conservative on the negative-delta case:
// when acceptance decreases (Δ < 0) or the change is below thresholdPP it
// returns false unconditionally — divergence can only fire on rising
// acceptance.
//
// Boundary behaviour: the acceptance-delta comparison uses strict less-than
// (< thresholdPP), so a delta that equals thresholdPP exactly DOES trigger
// divergence detection (if appropriateness is flat). This is by design:
// equality at the threshold is treated as "at or above the concern level".
// See TestDivergence_BoundaryAtThreshold for the documented assertion.
func DetectDivergence(prior, current RuleSnapshot, thresholdPP float64) bool {
	deltaAcceptance := current.AcceptanceRate - prior.AcceptanceRate
	deltaAppropriateness := current.AppropriatenessMean - prior.AppropriatenessMean
	if deltaAcceptance < thresholdPP {
		return false
	}
	return deltaAppropriateness < 0.3
}
