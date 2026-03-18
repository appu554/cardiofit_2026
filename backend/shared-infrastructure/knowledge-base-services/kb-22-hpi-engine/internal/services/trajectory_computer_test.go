package services

import (
	"math"
	"testing"
	"time"

	"kb-22-hpi-engine/internal/models"
)

// makeEquallySpacedSeries generates n points starting at origin, spaced intervalDays apart,
// with values following: y_i = intercept + slope * (i * intervalDays).
func makeEquallySpacedSeries(n int, intervalDays int, slope, intercept float64) []models.TimeSeriesPoint {
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	pts := make([]models.TimeSeriesPoint, n)
	for i := 0; i < n; i++ {
		t := origin.Add(time.Duration(i*intervalDays) * 24 * time.Hour)
		x := float64(i * intervalDays) // days from origin
		pts[i] = models.TimeSeriesPoint{
			Timestamp: t,
			Value:     intercept + slope*x,
		}
	}
	return pts
}

// TestTrajectoryComputer_LinearRegression: 10 equally spaced points (every 3 days),
// slope=0.1 mmol/L per day → per_month slope should be 0.1*30 = 3.0.
func TestTrajectoryComputer_LinearRegression(t *testing.T) {
	tc := NewTrajectoryComputer(testLogger())

	// slope = 0.1 per day, intercept = 5.0
	// With per_month rate_unit: expected slope = 0.1 * 30 = 3.0
	slopePerDay := 0.1
	series := makeEquallySpacedSeries(10, 3, slopePerDay, 5.0)

	cfg := models.TrajectoryConfig{
		Method:        "LINEAR_REGRESSION",
		WindowDays:    90,
		MinDataPoints: 5,
		RateUnit:      "per_month",
		DataSource:    "OBSERVATION",
	}

	result, err := tc.Compute(series, cfg)
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}

	expectedSlope := slopePerDay * 30.0 // converted to per_month
	if math.Abs(result.Slope-expectedSlope) > 0.001 {
		t.Errorf("Slope mismatch: got %.6f, want %.6f (tolerance 0.001)", result.Slope, expectedSlope)
	}
	if result.DataPoints != 10 {
		t.Errorf("DataPoints: got %d, want 10", result.DataPoints)
	}
}

// TestTrajectoryComputer_MinDataPoints: 3 points but min_data_points=5 → must return error.
func TestTrajectoryComputer_MinDataPoints(t *testing.T) {
	tc := NewTrajectoryComputer(testLogger())

	series := makeEquallySpacedSeries(3, 7, 0.05, 6.0)

	cfg := models.TrajectoryConfig{
		Method:        "LINEAR_REGRESSION",
		WindowDays:    90,
		MinDataPoints: 5,
		RateUnit:      "per_month",
		DataSource:    "OBSERVATION",
	}

	_, err := tc.Compute(series, cfg)
	if err == nil {
		t.Fatal("Expected error for insufficient data points, got nil")
	}
}

// TestTrajectoryComputer_Projection: slope=-0.05/month, current=0.35, threshold=0.20
// → days_to_threshold = (0.20 - 0.35) / (-0.05/30) = -0.15 / -0.001667 ≈ 90 days ≈ 3 months.
func TestTrajectoryComputer_Projection(t *testing.T) {
	tc := NewTrajectoryComputer(testLogger())

	// slope is in per_month units; Project uses slope in per_day internally.
	// slope per_month = -0.05 → slope per_day = -0.05/30
	slopePerMonth := -0.05
	current := 0.35
	threshold := 0.20

	// days_to_threshold = (threshold - current) / (slopePerMonth / 30)
	// = (0.20 - 0.35) / (-0.05 / 30)
	// = -0.15 / -0.001667
	// ≈ 90 days
	projectedDate, err := tc.Project(current, slopePerMonth, threshold)
	if err != nil {
		t.Fatalf("Project returned unexpected error: %v", err)
	}
	if projectedDate == nil {
		t.Fatal("Project returned nil date when slope is moving toward threshold")
	}

	now := time.Now()
	daysUntil := projectedDate.Sub(now).Hours() / 24.0

	// Allow ±5 days tolerance around 90 days
	if daysUntil < 85 || daysUntil > 95 {
		t.Errorf("Projected date is %.1f days from now, expected ~90 days (85-95 range)", daysUntil)
	}
}

// TestTrajectoryComputer_StableTrajectory: flat data → slope near 0.
func TestTrajectoryComputer_StableTrajectory(t *testing.T) {
	tc := NewTrajectoryComputer(testLogger())

	// All values identical → slope should be exactly 0
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	series := make([]models.TimeSeriesPoint, 8)
	for i := 0; i < 8; i++ {
		series[i] = models.TimeSeriesPoint{
			Timestamp: origin.Add(time.Duration(i*7) * 24 * time.Hour),
			Value:     7.2,
		}
	}

	cfg := models.TrajectoryConfig{
		Method:        "LINEAR_REGRESSION",
		WindowDays:    90,
		MinDataPoints: 5,
		RateUnit:      "per_month",
		DataSource:    "OBSERVATION",
	}

	result, err := tc.Compute(series, cfg)
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}

	if math.Abs(result.Slope) > 1e-9 {
		t.Errorf("Expected slope near 0 for flat data, got %.9f", result.Slope)
	}
}

// TestTrajectoryComputer_Confidence: R² is returned as a goodness-of-fit measure.
// A perfect linear series should produce R²=1.0; flat data should also give R²=1.0
// (degenerate case — zero variance, convention = 1.0).
// A noisy series should give R² < 1.0.
func TestTrajectoryComputer_Confidence(t *testing.T) {
	tc := NewTrajectoryComputer(testLogger())

	// Perfect linear series → R² = 1.0
	perfectSeries := makeEquallySpacedSeries(10, 3, 0.05, 5.0)
	cfg := models.TrajectoryConfig{
		Method:        "LINEAR_REGRESSION",
		WindowDays:    90,
		MinDataPoints: 5,
		RateUnit:      "per_month",
		DataSource:    "OBSERVATION",
	}

	result, err := tc.Compute(perfectSeries, cfg)
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}

	if math.Abs(result.RSquared-1.0) > 1e-6 {
		t.Errorf("Perfect linear series: expected R²=1.0, got %.6f", result.RSquared)
	}

	// Noisy series: add alternating ±noise on top of a trend
	origin := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	noisySeries := make([]models.TimeSeriesPoint, 10)
	noise := []float64{0.5, -0.5, 0.3, -0.3, 0.4, -0.4, 0.2, -0.2, 0.1, -0.1}
	for i := 0; i < 10; i++ {
		noisySeries[i] = models.TimeSeriesPoint{
			Timestamp: origin.Add(time.Duration(i*3) * 24 * time.Hour),
			Value:     5.0 + float64(i)*0.1 + noise[i],
		}
	}

	noisyResult, err := tc.Compute(noisySeries, cfg)
	if err != nil {
		t.Fatalf("Noisy compute returned unexpected error: %v", err)
	}
	if noisyResult.RSquared >= 1.0 {
		t.Errorf("Noisy series: expected R² < 1.0, got %.6f", noisyResult.RSquared)
	}
	if noisyResult.RSquared < 0.0 {
		t.Errorf("R² must be >= 0, got %.6f", noisyResult.RSquared)
	}
}

// TestTrajectoryComputer_ProjectionWrongDirection: if slope is moving away from threshold,
// Project should return nil (not a valid projection).
func TestTrajectoryComputer_ProjectionWrongDirection(t *testing.T) {
	tc := NewTrajectoryComputer(testLogger())

	// current=0.35, threshold=0.20, but slope is positive (moving away from 0.20)
	_, err := tc.Project(0.35, 0.05, 0.20)
	if err == nil {
		t.Fatal("Expected error when slope is moving away from threshold, got nil")
	}
}

// TestTrajectoryComputer_PerYear: verify per_year conversion.
func TestTrajectoryComputer_PerYear(t *testing.T) {
	tc := NewTrajectoryComputer(testLogger())

	// slope = 0.01 per day → per_year = 0.01 * 365 = 3.65
	series := makeEquallySpacedSeries(10, 3, 0.01, 60.0)

	cfg := models.TrajectoryConfig{
		Method:        "LINEAR_REGRESSION",
		WindowDays:    90,
		MinDataPoints: 5,
		RateUnit:      "per_year",
		DataSource:    "OBSERVATION",
	}

	result, err := tc.Compute(series, cfg)
	if err != nil {
		t.Fatalf("Compute returned unexpected error: %v", err)
	}

	expectedSlope := 0.01 * 365.0
	if math.Abs(result.Slope-expectedSlope) > 0.01 {
		t.Errorf("per_year slope: got %.4f, want %.4f (tolerance 0.01)", result.Slope, expectedSlope)
	}
}
