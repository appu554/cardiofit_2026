package services

import (
	"errors"
	"math"
	"time"
)

// ---------------------------------------------------------------------------
// EGFRReading — a single timestamped eGFR observation
// ---------------------------------------------------------------------------

// EGFRReading represents one eGFR measurement with its timestamp.
type EGFRReading struct {
	Value      float64   `json:"value"`
	MeasuredAt time.Time `json:"measured_at"`
}

// ---------------------------------------------------------------------------
// EGFRTrajectoryResult — output of the OLS regression
// ---------------------------------------------------------------------------

// EGFRTrajectoryResult holds the computed trajectory from a series of eGFR readings.
type EGFRTrajectoryResult struct {
	Slope            float64 `json:"slope"`             // mL/min/1.73m²/year
	Classification   string  `json:"classification"`    // RAPID_DECLINE | MODERATE_DECLINE | STABLE | IMPROVING
	IsRapidDecliner  bool    `json:"is_rapid_decliner"`
	DataPoints       int     `json:"data_points"`
	SpanDays         int     `json:"span_days"`
	LatestEGFR       float64 `json:"latest_egfr"`
	RSquared         float64 `json:"r_squared"`
}

const rapidDeclineThreshold = -5.0 // mL/min/1.73m²/year

// ---------------------------------------------------------------------------
// ComputeEGFRTrajectory — OLS linear regression on timestamped readings
// ---------------------------------------------------------------------------

// ComputeEGFRTrajectory computes a linear regression (OLS) over eGFR readings.
// Slope is expressed in mL/min/1.73m²/year. Requires at least 2 readings.
func ComputeEGFRTrajectory(readings []EGFRReading) (*EGFRTrajectoryResult, error) {
	n := len(readings)
	if n < 2 {
		return nil, errors.New("insufficient eGFR data: need at least 2 readings")
	}

	// Find earliest reading as time origin and latest for LatestEGFR.
	earliest := readings[0].MeasuredAt
	latest := readings[0].MeasuredAt
	latestEGFR := readings[0].Value
	for _, r := range readings[1:] {
		if r.MeasuredAt.Before(earliest) {
			earliest = r.MeasuredAt
		}
		if r.MeasuredAt.After(latest) {
			latest = r.MeasuredAt
			latestEGFR = r.Value
		}
	}

	spanDays := int(latest.Sub(earliest).Hours() / 24)

	// Convert timestamps to years from origin for regression.
	const hoursPerYear = 365.25 * 24

	var sumX, sumY, sumXX, sumXY float64
	for _, r := range readings {
		x := r.MeasuredAt.Sub(earliest).Hours() / hoursPerYear
		y := r.Value
		sumX += x
		sumY += y
		sumXX += x * x
		sumXY += x * y
	}

	nf := float64(n)
	meanX := sumX / nf
	meanY := sumY / nf

	// OLS slope = Σ(xi - x̄)(yi - ȳ) / Σ(xi - x̄)²
	// Using computational form: (sumXY - n*meanX*meanY) / (sumXX - n*meanX*meanX)
	denom := sumXX - nf*meanX*meanX
	var slope float64
	if math.Abs(denom) < 1e-12 {
		slope = 0.0 // all readings at same time (or effectively so)
	} else {
		slope = (sumXY - nf*meanX*meanY) / denom
	}

	// R² calculation
	var ssTot, ssRes float64
	intercept := meanY - slope*meanX
	for _, r := range readings {
		x := r.MeasuredAt.Sub(earliest).Hours() / hoursPerYear
		predicted := intercept + slope*x
		ssRes += (r.Value - predicted) * (r.Value - predicted)
		ssTot += (r.Value - meanY) * (r.Value - meanY)
	}
	var rSquared float64
	if ssTot > 1e-12 {
		rSquared = 1.0 - ssRes/ssTot
	}

	classification := classifyEGFRSlope(slope)

	return &EGFRTrajectoryResult{
		Slope:           slope,
		Classification:  classification,
		IsRapidDecliner: slope <= rapidDeclineThreshold,
		DataPoints:      n,
		SpanDays:        spanDays,
		LatestEGFR:      latestEGFR,
		RSquared:        rSquared,
	}, nil
}

// ---------------------------------------------------------------------------
// classifyEGFRSlope — classify rate of change
// ---------------------------------------------------------------------------

func classifyEGFRSlope(slope float64) string {
	switch {
	case slope <= -5.0:
		return "RAPID_DECLINE"
	case slope <= -1.0:
		return "MODERATE_DECLINE"
	case slope < 1.0:
		return "STABLE"
	default:
		return "IMPROVING"
	}
}

// ---------------------------------------------------------------------------
// ProjectTimeToThreshold — months until eGFR crosses a threshold
// ---------------------------------------------------------------------------

// ProjectTimeToThreshold returns the number of months until currentEGFR
// reaches threshold given slopePerYear (mL/min/1.73m²/year).
// Returns nil if slope is non-negative (improving/stable) or already below threshold.
func ProjectTimeToThreshold(currentEGFR, slopePerYear, threshold float64) *float64 {
	if slopePerYear >= 0 {
		return nil // improving or stable — won't cross downward
	}
	if currentEGFR <= threshold {
		return nil // already below threshold
	}
	gap := currentEGFR - threshold
	years := gap / math.Abs(slopePerYear)
	months := years * 12.0
	return &months
}
