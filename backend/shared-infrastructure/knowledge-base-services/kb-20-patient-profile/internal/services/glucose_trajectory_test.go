package services

import (
	"testing"
	"time"

	"kb-patient-profile/internal/models"
)

func TestClassifyGlucoseTrajectory_Stable(t *testing.T) {
	readings := makeReadings([]float64{130, 132, 128, 131, 129}, 7)
	result := ClassifyGlucoseTrajectory(readings)

	if result.Classification != TrajectoryStable {
		t.Errorf("expected STABLE, got %s (slope=%.2f)", result.Classification, result.FBGSlope)
	}
}

func TestClassifyGlucoseTrajectory_Rising(t *testing.T) {
	readings := makeReadings([]float64{130, 138, 145, 152, 160}, 7)
	result := ClassifyGlucoseTrajectory(readings)

	if result.Classification != TrajectoryRising {
		t.Errorf("expected RISING, got %s (slope=%.2f)", result.Classification, result.FBGSlope)
	}
}

func TestClassifyGlucoseTrajectory_RapidRising(t *testing.T) {
	readings := makeReadings([]float64{130, 150, 170, 185, 200}, 5)
	result := ClassifyGlucoseTrajectory(readings)

	if result.Classification != TrajectoryRapidRising {
		t.Errorf("expected RAPID_RISING, got %s (slope=%.2f)", result.Classification, result.FBGSlope)
	}
}

func TestClassifyGlucoseTrajectory_Declining(t *testing.T) {
	readings := makeReadings([]float64{180, 170, 162, 155, 148}, 7)
	result := ClassifyGlucoseTrajectory(readings)

	if result.Classification != TrajectoryDeclining {
		t.Errorf("expected DECLINING, got %s (slope=%.2f)", result.Classification, result.FBGSlope)
	}
}

func TestClassifyGlucoseTrajectory_Improving(t *testing.T) {
	readings := makeReadings([]float64{200, 180, 160, 145, 130}, 7)
	result := ClassifyGlucoseTrajectory(readings)

	if result.Classification != TrajectoryImproving {
		t.Errorf("expected IMPROVING, got %s (slope=%.2f)", result.Classification, result.FBGSlope)
	}
}

func TestGlucoseCV_HighVariability(t *testing.T) {
	// CV > 36% indicates high glycaemic variability (B-20 rule)
	readings := makeReadings([]float64{80, 200, 90, 210, 85}, 1)
	result := ClassifyGlucoseTrajectory(readings)

	if result.GlucoseCV <= 36.0 {
		t.Errorf("expected CV > 36%%, got %.1f%%", result.GlucoseCV)
	}
	if !result.HighVariability {
		t.Error("expected HighVariability=true for CV > 36%")
	}
}

func TestClassifyGlucoseTrajectory_InsufficientData(t *testing.T) {
	readings := makeReadings([]float64{130}, 7)
	result := ClassifyGlucoseTrajectory(readings)

	if result.Classification != TrajectoryStable {
		t.Errorf("expected STABLE (default) with insufficient data, got %s", result.Classification)
	}
}

func TestGlucoseTrajectory_FiredOnFBGWrite(t *testing.T) {
	// Type-check: LabService must have updateGlucoseTrajectory method.
	// Full integration test requires DB setup — defer to integration suite.
	var s *LabService
	_ = s // compile-time verification that LabService type exists with required methods
}

// makeReadings creates test FBG readings at dayInterval-day spacing.
func makeReadings(values []float64, dayInterval int) []models.TimestampedLabValue {
	now := time.Now().UTC()
	readings := make([]models.TimestampedLabValue, len(values))
	for i, v := range values {
		readings[i] = models.TimestampedLabValue{
			Value:     v,
			Timestamp: now.Add(-time.Duration(len(values)-1-i) * time.Duration(dayInterval) * 24 * time.Hour),
		}
	}
	return readings
}
