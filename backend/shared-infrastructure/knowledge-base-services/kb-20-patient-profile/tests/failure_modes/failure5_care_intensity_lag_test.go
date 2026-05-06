// Wave 6.1 — Failure Mode 5: care-intensity transition lag.
//
// Layer 2 doc Part 6 Failure 5: "a CFS≥7 score may sit unactioned in the
// system for days, delaying conversation about palliative reframing.
// Defence: a CFS score of 7 or higher writes a worklist hint within 60s
// of recording."
//
// Pure-logic test against the CFSScoreShouldHintCareIntensityReview
// helper from Wave 2.6. The 60-second SLA itself is a kb-20 outbox
// processing target verified by the integration test pack; this file
// asserts the predicate that drives the hint emission.
package failure_modes

import (
	"testing"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestFailure5_CFSGE7_ProducesWorklistHint(t *testing.T) {
	for _, score := range []int{7, 8, 9} {
		if !models.CFSScoreShouldHintCareIntensityReview(score) {
			t.Fatalf("CFS=%d MUST produce a hint (Failure 5 defence)", score)
		}
	}
}

func TestFailure5_CFSLT7_DoesNotProduceHint(t *testing.T) {
	for _, score := range []int{1, 2, 3, 4, 5, 6} {
		if models.CFSScoreShouldHintCareIntensityReview(score) {
			t.Fatalf("CFS=%d should NOT produce a hint (would create alert fatigue)", score)
		}
	}
}

func TestFailure5_CFSOutOfRange_DoesNotPanic(t *testing.T) {
	// Out-of-range scores are validator territory; the hint predicate is
	// expected to handle them gracefully (false rather than panic).
	for _, score := range []int{0, -1, 10, 99} {
		_ = models.CFSScoreShouldHintCareIntensityReview(score) // smoke
	}
}

func TestFailure5_ThresholdConstantStable(t *testing.T) {
	// Layer 2 §2.4 specifies CFS≥7 as the trigger. If this constant
	// changes, downstream hint workflows must coordinate — assert the
	// canonical value here so accidental edits flag in PR review.
	if models.CFSCareIntensityReviewThreshold != 7 {
		t.Fatalf("CFSCareIntensityReviewThreshold drift: want 7, got %d", models.CFSCareIntensityReviewThreshold)
	}
}
