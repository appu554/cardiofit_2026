package pattern_detection

// SuppressionInputs carries the deferral statistics for a single clinical rule
// over a rolling observation window.
type SuppressionInputs struct {
	// RuleID identifies the clinical rule being evaluated.
	RuleID string

	// TotalRecommendations is the number of times this rule generated a
	// recommendation during the window.
	TotalRecommendations int

	// DeferredCount is the number of recommendations that were deferred (not
	// accepted or rejected with contemporaneous documentation).
	DeferredCount int

	// DeferredWithReasoningCount is the subset of deferred recommendations for
	// which the deferring clinician recorded an explicit clinical rationale.
	DeferredWithReasoningCount int
}

// DetectSuppression flags systematic recommendation suppression per
// Guidelines §10.
//
// A rule is considered suppressed when two conditions are simultaneously met:
//
//  1. Deferral rate ≥ deferralThreshold — a large fraction of the rule's
//     recommendations are being deferred.
//  2. Undocumented deferral rate ≥ (1 − undocumentedThreshold) — most of
//     those deferrals lack recorded clinical reasoning.
//
// Division-by-zero guards: returns false immediately when
// TotalRecommendations == 0 (deferral rate undefined) or when DeferredCount
// == 0 (undocumented rate undefined — zero deferrals cannot be undocumented).
//
// The undocumented fraction is computed as:
//
//	undocumented = DeferredCount − DeferredWithReasoningCount
//	undocumentedRate = undocumented / DeferredCount
//
// and compared against (1 − undocumentedThreshold). For example, with
// undocumentedThreshold = 0.20, the flag fires when ≥ 80 % of deferrals lack
// reasoning.
func DetectSuppression(in SuppressionInputs, deferralThreshold, undocumentedThreshold float64) bool {
	if in.TotalRecommendations == 0 {
		return false
	}
	deferralRate := float64(in.DeferredCount) / float64(in.TotalRecommendations)
	if deferralRate < deferralThreshold {
		return false
	}
	if in.DeferredCount == 0 {
		return false
	}
	undocumented := in.DeferredCount - in.DeferredWithReasoningCount
	undocumentedRate := float64(undocumented) / float64(in.DeferredCount)
	return undocumentedRate >= 1-undocumentedThreshold
}
