package services

import (
	"math"
	"sort"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// ComputeBaseline computes the rolling median and MAD from readings.
// readings and timestamps must be same length and pre-sorted by time ascending.
// lookbackDays: 7 for primary, 14 for fallback.
func ComputeBaseline(readings []float64, timestamps []time.Time, lookbackDays int) models.PatientBaselineSnapshot {
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -lookbackDays)

	// Filter readings within the lookback window.
	var filtered []float64
	for i, ts := range timestamps {
		if !ts.Before(cutoff) {
			filtered = append(filtered, readings[i])
		}
	}

	// Sort filtered values for median/MAD computation.
	sorted := make([]float64, len(filtered))
	copy(sorted, filtered)
	sort.Float64s(sorted)

	median := Median(sorted)
	mad := MAD(sorted, median)
	confidence := BaselineConfidence(len(sorted))

	return models.PatientBaselineSnapshot{
		BaselineMedian: median,
		BaselineMAD:    mad,
		ReadingCount:   len(sorted),
		Confidence:     confidence,
		LookbackDays:   lookbackDays,
		ComputedAt:     now,
	}
}

// Median computes the median of a sorted float64 slice.
// Returns 0 if the slice is empty.
func Median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2.0
}

// MAD computes the Median Absolute Deviation.
// Given values and their median, it computes |value - median| for each value,
// then returns the median of those absolute deviations.
func MAD(values []float64, median float64) float64 {
	if len(values) == 0 {
		return 0
	}

	deviations := make([]float64, len(values))
	for i, v := range values {
		deviations[i] = math.Abs(v - median)
	}
	sort.Float64s(deviations)
	return Median(deviations)
}

// BaselineConfidence classifies confidence based on the number of readings.
//
//	>= 7 → "HIGH"   (full week of daily data)
//	3-6  → "MODERATE"
//	< 3  → "LOW"
func BaselineConfidence(readingCount int) string {
	switch {
	case readingCount >= 7:
		return "HIGH"
	case readingCount >= 3:
		return "MODERATE"
	default:
		return "LOW"
	}
}
