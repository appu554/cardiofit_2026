package kpis

import "sort"

// MedianContextTime computes the median context-assembly time (in minutes) across
// a rolling window of up to 30 reviews.
//
// VisibilityClass: PFA — aggregate visible to employer; never per-pharmacist to employer.
//
// Returns 0 for empty input (sentinel for "no data").
// For even-length slices the median is the average of the two middle values.
func MedianContextTime(minutes []float64) float64 {
	if len(minutes) == 0 {
		return 0
	}

	// Copy to avoid mutating the caller's slice.
	cp := make([]float64, len(minutes))
	copy(cp, minutes)
	sort.Float64s(cp)

	n := len(cp)
	mid := n / 2
	if n%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2.0
}
