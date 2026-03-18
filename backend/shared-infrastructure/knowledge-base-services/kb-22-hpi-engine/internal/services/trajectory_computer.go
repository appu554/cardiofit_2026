package services

import (
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// TrajectoryComputer performs linear regression on time-series data and projects
// when a state variable will cross a clinical threshold.
//
// Used by:
//   - MonitoringNodeEngine (PM-06: FBG slope detection)
//   - DeteriorationNodeEngine (MD-01 through MD-06: multi-variable deterioration signals)
//
// Math:
//
//	OLS slope = Σ(xi-x̄)(yi-ȳ) / Σ(xi-x̄)²
//	R²        = 1 - SS_res/SS_tot
//	Projection = current + slope*t = threshold → t = (threshold-current)/slope
type TrajectoryComputer struct {
	log *zap.Logger
}

// TrajectoryResult holds the output of a linear regression over a time series.
type TrajectoryResult struct {
	// Slope is the rate of change converted to the configured RateUnit
	// (e.g., per_month or per_year).
	Slope float64

	// Intercept is the OLS intercept at x=0 (first point in the series), in
	// original value units (not scaled).
	Intercept float64

	// RSquared is the coefficient of determination (0-1).  For perfectly flat
	// series (zero variance in y) the convention used here is 1.0.
	RSquared float64

	// DataPoints is the number of observations used in the regression.
	DataPoints int
}

// NewTrajectoryComputer creates a TrajectoryComputer with the given logger.
func NewTrajectoryComputer(log *zap.Logger) *TrajectoryComputer {
	return &TrajectoryComputer{log: log}
}

// Compute runs a linear regression on the supplied time series, returning slope
// scaled to cfg.RateUnit ("per_month" or "per_year") and R² as confidence.
//
// Steps:
//  1. Validate len(series) >= cfg.MinDataPoints.
//  2. Convert timestamps to numeric x (fractional days since first point).
//  3. OLS regression for slope (per day) and intercept.
//  4. Scale slope to RateUnit.
//  5. Compute R².
func (tc *TrajectoryComputer) Compute(series []models.TimeSeriesPoint, cfg models.TrajectoryConfig) (*TrajectoryResult, error) {
	if len(series) < cfg.MinDataPoints {
		return nil, fmt.Errorf(
			"trajectory: need at least %d data points, got %d",
			cfg.MinDataPoints, len(series),
		)
	}

	n := float64(len(series))
	origin := series[0].Timestamp

	// Build numeric (x, y) pairs where x = days from origin.
	xs := make([]float64, len(series))
	ys := make([]float64, len(series))
	for i, pt := range series {
		xs[i] = pt.Timestamp.Sub(origin).Hours() / 24.0
		ys[i] = pt.Value
	}

	// Means.
	var sumX, sumY float64
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
	}
	xBar := sumX / n
	yBar := sumY / n

	// OLS numerator and denominator.
	var sxy, sxx float64
	for i := range xs {
		dx := xs[i] - xBar
		dy := ys[i] - yBar
		sxy += dx * dy
		sxx += dx * dx
	}

	var slopePerDay, intercept float64
	if sxx == 0 {
		// All x values identical (degenerate: only one unique timestamp).
		slopePerDay = 0
		intercept = yBar
	} else {
		slopePerDay = sxy / sxx
		intercept = yBar - slopePerDay*xBar
	}

	// Scale slope to requested rate unit.
	scaledSlope := slopePerDay * rateUnitMultiplier(cfg.RateUnit)

	// R² = 1 - SS_res / SS_tot.
	var ssTot, ssRes float64
	for i := range xs {
		yHat := intercept + slopePerDay*xs[i]
		ssRes += math.Pow(ys[i]-yHat, 2)
		ssTot += math.Pow(ys[i]-yBar, 2)
	}

	var rSquared float64
	if ssTot == 0 {
		// All y values identical: perfect flat line, convention R²=1.
		rSquared = 1.0
	} else {
		rSquared = 1.0 - ssRes/ssTot
		if rSquared < 0 {
			rSquared = 0
		}
	}

	tc.log.Info("trajectory: regression complete",
		zap.Int("n", len(series)),
		zap.Float64("slope_per_day", slopePerDay),
		zap.Float64("scaled_slope", scaledSlope),
		zap.String("rate_unit", cfg.RateUnit),
		zap.Float64("r_squared", rSquared),
	)

	return &TrajectoryResult{
		Slope:      scaledSlope,
		Intercept:  intercept,
		RSquared:   rSquared,
		DataPoints: len(series),
	}, nil
}

// Project computes the future date at which a linearly-trending variable is
// projected to cross `threshold`, given its `current` value and `slope`
// expressed in per-month units.
//
// Returns an error when the slope is not moving toward the threshold (i.e.,
// projection is not physically meaningful — the variable would never reach the
// threshold at the current rate of change).
//
// Linear extrapolation:
//
//	days_to_threshold = (threshold - current) / (slope / 30)
func (tc *TrajectoryComputer) Project(current, slope, threshold float64) (*time.Time, error) {
	if slope == 0 {
		return nil, fmt.Errorf("trajectory: slope is zero, cannot project crossing of threshold %.4f", threshold)
	}

	// Convert per-month slope to per-day.
	slopePerDay := slope / 30.0

	daysToThreshold := (threshold - current) / slopePerDay

	if daysToThreshold <= 0 {
		return nil, fmt.Errorf(
			"trajectory: slope (%.4f/month) is not moving toward threshold %.4f from current %.4f",
			slope, threshold, current,
		)
	}

	projected := time.Now().Add(time.Duration(daysToThreshold*24) * time.Hour)

	tc.log.Info("trajectory: projection computed",
		zap.Float64("current", current),
		zap.Float64("slope_per_month", slope),
		zap.Float64("threshold", threshold),
		zap.Float64("days_to_threshold", daysToThreshold),
		zap.Time("projected_date", projected),
	)

	return &projected, nil
}

// rateUnitMultiplier returns the factor to convert a per-day slope to the
// requested unit.  Defaults to per_month (×30) for unrecognised units.
func rateUnitMultiplier(unit string) float64 {
	switch unit {
	case "per_year":
		return 365.0
	case "per_week":
		return 7.0
	case "per_month":
		return 30.0
	default:
		return 30.0
	}
}
