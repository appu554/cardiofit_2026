package pattern_detection

import "testing"

// ---------------------------------------------------------------------------
// Plan-verbatim tests (Task 8)
// ---------------------------------------------------------------------------

func TestDivergence_FlagsRisingAcceptanceFlatAppropriateness(t *testing.T) {
	prior := RuleSnapshot{AcceptanceRate: 0.55, AppropriatenessMean: 3.8}
	current := RuleSnapshot{AcceptanceRate: 0.70, AppropriatenessMean: 3.85}
	if !DetectDivergence(prior, current, 0.10) {
		t.Errorf("expected divergence for +15pp acceptance with flat appropriateness")
	}
}

func TestDivergence_DoesNotFlagCorrelatedRise(t *testing.T) {
	prior := RuleSnapshot{AcceptanceRate: 0.55, AppropriatenessMean: 3.8}
	current := RuleSnapshot{AcceptanceRate: 0.70, AppropriatenessMean: 4.4}
	if DetectDivergence(prior, current, 0.10) {
		t.Errorf("correlated acceptance + appropriateness rise should not divergence-flag")
	}
}

// ---------------------------------------------------------------------------
// Augmentations
// ---------------------------------------------------------------------------

// TestDivergence_NegativeAcceptanceDeltaDoesNotFlag asserts that a declining
// acceptance rate never triggers divergence. Per spec, divergence only fires
// on rising acceptance; a falling rate poses no acceptance-quality decoupling
// risk.
func TestDivergence_NegativeAcceptanceDeltaDoesNotFlag(t *testing.T) {
	prior := RuleSnapshot{AcceptanceRate: 0.70, AppropriatenessMean: 4.0}
	current := RuleSnapshot{AcceptanceRate: 0.50, AppropriatenessMean: 3.5}
	if DetectDivergence(prior, current, 0.10) {
		t.Errorf("declining acceptance should never trigger divergence")
	}
}

// TestDivergence_BoundaryAtThreshold documents the exact-equality boundary
// behaviour of DetectDivergence.
//
// When ΔAcceptanceRate == thresholdPP (i.e. the delta exactly equals the
// threshold) and appropriateness is flat (Δ < 0.3), the detector DOES flag.
// The comparison uses strict less-than (deltaAcceptance < thresholdPP), so
// equality is treated as "at or above the concern level" — the caller set a
// threshold and the observation reached it precisely.
//
// Values are chosen so that the subtraction is exact in IEEE 754 double
// precision: 0.25 and 0.375 are exact binary fractions, so
// 0.375 − 0.25 == 0.125 without rounding error.
func TestDivergence_BoundaryAtThreshold(t *testing.T) {
	threshold := 0.125 // exact in float64 (2^-3)
	prior := RuleSnapshot{AcceptanceRate: 0.25, AppropriatenessMean: 3.8}
	// delta == threshold exactly (no floating-point rounding), appropriateness flat
	current := RuleSnapshot{
		AcceptanceRate:      0.375, // 0.25 + 0.125 is exact
		AppropriatenessMean: prior.AppropriatenessMean + 0.05, // Δ < 0.3
	}
	if !DetectDivergence(prior, current, threshold) {
		t.Errorf("acceptance delta exactly at threshold with flat appropriateness should flag")
	}
}
