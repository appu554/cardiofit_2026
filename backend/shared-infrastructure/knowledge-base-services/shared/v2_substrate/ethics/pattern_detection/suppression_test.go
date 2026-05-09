package pattern_detection

import "testing"

// ---------------------------------------------------------------------------
// Plan-verbatim tests (Task 9 — suppression)
// ---------------------------------------------------------------------------

func TestSuppression_FlagsHighDeferralWithoutReasoning(t *testing.T) {
	if !DetectSuppression(SuppressionInputs{
		TotalRecommendations:       100,
		DeferredCount:              40,
		DeferredWithReasoningCount: 5,
	}, 0.30, 0.20) {
		t.Errorf("expected suppression flag")
	}
}

func TestSuppression_DoesNotFlagBalanced(t *testing.T) {
	if DetectSuppression(SuppressionInputs{
		TotalRecommendations:       100,
		DeferredCount:              20,
		DeferredWithReasoningCount: 18,
	}, 0.30, 0.20) {
		t.Errorf("balanced deferral should not flag")
	}
}

// ---------------------------------------------------------------------------
// Augmentations
// ---------------------------------------------------------------------------

// TestSuppression_ZeroTotalReturnsFalse guards against division by zero when
// no recommendations were generated during the observation window (e.g. the
// rule was temporarily inactive).
func TestSuppression_ZeroTotalReturnsFalse(t *testing.T) {
	if DetectSuppression(SuppressionInputs{
		TotalRecommendations:       0,
		DeferredCount:              0,
		DeferredWithReasoningCount: 0,
	}, 0.30, 0.20) {
		t.Errorf("zero TotalRecommendations should return false (undefined deferral rate)")
	}
}

// TestSuppression_ZeroDeferredReturnsFalse guards against division by zero
// when the deferral rate is at or above the threshold but DeferredCount is
// zero — an impossible real-world case, but defensible guard for adversarial
// or test inputs where TotalRecommendations is also zero but is checked first
// (covered above). This test exercises the DeferredCount == 0 branch with a
// non-zero TotalRecommendations to reach the inner guard.
//
// If DeferredCount == 0 and TotalRecommendations > 0, the deferral rate is
// 0 / N = 0, which is below any positive deferralThreshold, so the function
// returns false from the deferral-rate check before reaching the undocumented
// rate. The test confirms this invariant even when thresholds are relaxed.
func TestSuppression_ZeroDeferredReturnsFalse(t *testing.T) {
	// Use threshold of 0.0 so deferral-rate check passes (0.0 >= 0.0),
	// exposing the DeferredCount == 0 guard in the undocumented-rate branch.
	if DetectSuppression(SuppressionInputs{
		TotalRecommendations:       100,
		DeferredCount:              0,
		DeferredWithReasoningCount: 0,
	}, 0.0, 0.20) {
		t.Errorf("zero DeferredCount should return false (undefined undocumented rate)")
	}
}
