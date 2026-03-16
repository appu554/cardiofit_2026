package services

import (
	"math"
	"time"

	"kb-patient-profile/internal/models"
)

const (
	fbgSlopeWorseningThreshold = 0.5  // >+0.5 mmol/L per quarter = WORSENING
	fbgSlopeImprovingThreshold = -0.5 // <-0.5 mmol/L per quarter = IMPROVING
	glucoseCVHighThreshold     = 36.0 // CV% > 36% = high variability (ADA 2024)
	glucoseCVResolvedThreshold = 30.0 // CV% drops below 30% = resolved
	minReadingsForSlope        = 4    // minimum readings for linear regression
	minReadingsForCV           = 5    // minimum readings for CV computation
)

// computeFBGSlope returns the FBG slope in mmol/L per quarter (90 days)
// using ordinary least-squares linear regression on timestamped readings.
func computeFBGSlope(readings []models.TimestampedReading) float64 {
	n := len(readings)
	if n < minReadingsForSlope {
		return 0
	}

	t0 := readings[0].Timestamp
	var sumX, sumY, sumXY, sumX2 float64
	for _, r := range readings {
		x := r.Timestamp.Sub(t0).Hours() / 24.0
		y := r.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	fn := float64(n)
	denom := fn*sumX2 - sumX*sumX
	if denom == 0 {
		return 0
	}

	slopePerDay := (fn*sumXY - sumX*sumY) / denom
	return slopePerDay * 90.0
}

// classifyFBGTrend maps slope to trend category.
func classifyFBGTrend(slopePerQ float64) string {
	switch {
	case slopePerQ > fbgSlopeWorseningThreshold:
		return "WORSENING"
	case slopePerQ < fbgSlopeImprovingThreshold:
		return "IMPROVING"
	default:
		return "STABLE"
	}
}

// computeGlucoseCV computes coefficient of variation (SD/Mean * 100) for glucose readings.
func computeGlucoseCV(readings []models.TimestampedReading) float64 {
	n := len(readings)
	if n < minReadingsForCV {
		return 0
	}

	var sum float64
	for _, r := range readings {
		sum += r.Value
	}
	mean := sum / float64(n)
	if mean == 0 {
		return 0
	}

	var sumSqDiff float64
	for _, r := range readings {
		diff := r.Value - mean
		sumSqDiff += diff * diff
	}
	sd := math.Sqrt(sumSqDiff / float64(n))
	return (sd / mean) * 100.0
}

// filterReadingsInWindow returns readings within the last `days` days.
func filterReadingsInWindow(readings []models.TimestampedReading, days int) []models.TimestampedReading {
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	var filtered []models.TimestampedReading
	for _, r := range readings {
		if r.Timestamp.After(cutoff) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
