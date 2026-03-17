package services

import (
	"math"

	"kb-patient-profile/internal/models"
)

// Trajectory classifications
const (
	TrajectoryStable      = "STABLE"
	TrajectoryRising      = "RISING"
	TrajectoryRapidRising = "RAPID_RISING"
	TrajectoryDeclining   = "DECLINING"
	TrajectoryImproving   = "IMPROVING"
)

// Thresholds (mg/dL per week)
const (
	RisingThreshold      = 5.0   // >5 mg/dL/week = RISING
	RapidRisingThreshold = 15.0  // >15 mg/dL/week = RAPID_RISING
	DecliningThreshold   = -3.0  // <-3 mg/dL/week = DECLINING
	ImprovingThreshold   = -10.0 // <-10 mg/dL/week = IMPROVING
	HighCVThreshold      = 36.0  // CV >36% = high glycaemic variability (B-20)
	MinReadings          = 3     // minimum readings for trajectory classification
)

// TrajectoryResult holds the output of FBG trajectory classification.
type TrajectoryResult struct {
	Classification  string  // STABLE | RISING | RAPID_RISING | DECLINING | IMPROVING
	FBGSlope        float64 // mg/dL per week (linear regression)
	GlucoseCV       float64 // coefficient of variation %
	HighVariability bool    // true if CV > 36%
	ReadingsUsed    int
}

// ClassifyGlucoseTrajectory computes FBG slope via OLS regression and classifies
// the trajectory into one of 5 states. Also computes glucose CV%.
func ClassifyGlucoseTrajectory(readings []models.TimestampedLabValue) TrajectoryResult {
	result := TrajectoryResult{
		Classification: TrajectoryStable,
		ReadingsUsed:   len(readings),
	}

	if len(readings) < MinReadings {
		return result
	}

	// Compute glucose CV% (coefficient of variation)
	result.GlucoseCV = computeGlucoseCV(readings)
	result.HighVariability = result.GlucoseCV > HighCVThreshold

	// Compute FBG slope via linear regression (mg/dL per week)
	result.FBGSlope = computeFBGSlope(readings)

	// Classify based on slope
	switch {
	case result.FBGSlope >= RapidRisingThreshold:
		result.Classification = TrajectoryRapidRising
	case result.FBGSlope >= RisingThreshold:
		result.Classification = TrajectoryRising
	case result.FBGSlope <= ImprovingThreshold:
		result.Classification = TrajectoryImproving
	case result.FBGSlope <= DecliningThreshold:
		result.Classification = TrajectoryDeclining
	default:
		result.Classification = TrajectoryStable
	}

	return result
}

// computeFBGSlope returns the FBG slope in mg/dL per week via OLS regression.
func computeFBGSlope(readings []models.TimestampedLabValue) float64 {
	if len(readings) < 2 {
		return 0
	}

	earliest := readings[0].Timestamp
	n := float64(len(readings))
	var sumX, sumY, sumXY, sumX2 float64

	for _, r := range readings {
		x := r.Timestamp.Sub(earliest).Hours() / (24 * 7) // weeks
		y := r.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denom := n*sumX2 - sumX*sumX
	if math.Abs(denom) < 1e-10 {
		return 0
	}

	return (n*sumXY - sumX*sumY) / denom
}

// computeGlucoseCV returns the coefficient of variation (%) of glucose readings.
// CV = (SD / mean) × 100
func computeGlucoseCV(readings []models.TimestampedLabValue) float64 {
	if len(readings) < 2 {
		return 0
	}

	n := float64(len(readings))
	var sum float64
	for _, r := range readings {
		sum += r.Value
	}
	mean := sum / n

	if mean == 0 {
		return 0
	}

	var sumSqDiff float64
	for _, r := range readings {
		diff := r.Value - mean
		sumSqDiff += diff * diff
	}
	sd := math.Sqrt(sumSqDiff / (n - 1))

	return (sd / mean) * 100
}
