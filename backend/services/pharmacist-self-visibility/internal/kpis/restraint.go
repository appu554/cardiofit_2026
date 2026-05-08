package kpis

// RestraintOverride captures a single instance where the pharmacist overrode a
// system recommendation.
//
// VisibilityClass: POA — pharmacist alone sees this data; never exposed to employer.
type RestraintOverride struct {
	// RecommendationType is the category of recommendation that was overridden
	// (e.g. "deprescribe", "dose_reduce", "switch").
	RecommendationType string

	// Reasoning is the free-text rationale provided by the pharmacist.
	// Treated as reflective content; POA only.
	Reasoning string
}

// RestraintOverridePattern counts overrides by RecommendationType across the
// provided slice.
//
// Returns an empty (non-nil) map for empty input.
func RestraintOverridePattern(overrides []RestraintOverride) map[string]int {
	counts := make(map[string]int)
	for _, o := range overrides {
		counts[o.RecommendationType]++
	}
	return counts
}
