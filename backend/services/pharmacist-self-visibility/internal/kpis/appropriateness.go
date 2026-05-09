package kpis

// MeanAppropriateness computes the mean appropriateness score across a rolling
// 90-day window of scored reviews.
//
// VisibilityClass: PDP — never exposed per-pharmacist to employer; pharmacist
// sees their own aggregate only.
//
// Returns 0 for empty input (sentinel for "no data", not NaN).
func MeanAppropriateness(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	return sum / float64(len(scores))
}
