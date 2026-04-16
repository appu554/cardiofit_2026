package services

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
)

// TestComputeSecondDerivatives_DeceleratingDecline verifies that a
// domain whose slope is negative but becoming less negative over
// successive snapshots produces DECELERATING_DECLINE — the
// intervention is working, the patient is still declining but more
// slowly. V4-5 Phase 3.
func TestComputeSecondDerivatives_DeceleratingDecline(t *testing.T) {
	engine := NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, zap.NewNop())

	// 5 snapshots where glucose slope starts at -2.0 and improves toward -0.5
	// (decline is decelerating: slope_of_slope is positive)
	snapshots := []models.DomainTrajectoryHistory{
		{SnapshotDate: time.Now().AddDate(0, 0, -28), GlucoseSlope: -2.0, CardioSlope: -0.1},
		{SnapshotDate: time.Now().AddDate(0, 0, -21), GlucoseSlope: -1.6, CardioSlope: -0.1},
		{SnapshotDate: time.Now().AddDate(0, 0, -14), GlucoseSlope: -1.2, CardioSlope: -0.1},
		{SnapshotDate: time.Now().AddDate(0, 0, -7), GlucoseSlope: -0.8, CardioSlope: -0.1},
		{SnapshotDate: time.Now(), GlucoseSlope: -0.5, CardioSlope: -0.1},
	}

	results := engine.ComputeSecondDerivatives(snapshots)
	if results == nil {
		t.Fatal("expected non-nil results")
	}

	glucose, ok := results[models.DomainGlucose]
	if !ok {
		t.Fatal("expected GLUCOSE in results")
	}
	if glucose.SlopeOfSlope <= 0 {
		t.Errorf("SlopeOfSlope = %f, expected positive (decelerating decline)", glucose.SlopeOfSlope)
	}
	if glucose.Interpretation != "DECELERATING_DECLINE" {
		t.Errorf("Interpretation = %q, want DECELERATING_DECLINE", glucose.Interpretation)
	}
	if glucose.SnapshotsUsed != 5 {
		t.Errorf("SnapshotsUsed = %d, want 5", glucose.SnapshotsUsed)
	}
	if glucose.Confidence != models.ConfidenceHigh {
		t.Errorf("Confidence = %q, want HIGH (5 snapshots)", glucose.Confidence)
	}
}

// TestComputeSecondDerivatives_AcceleratingDecline verifies that a
// domain whose slope is becoming MORE negative over time produces
// ACCELERATING_DECLINE — the intervention is failing.
func TestComputeSecondDerivatives_AcceleratingDecline(t *testing.T) {
	engine := NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, zap.NewNop())

	// Glucose slope getting worse: -0.5 → -1.0 → -1.5 → -2.0
	snapshots := []models.DomainTrajectoryHistory{
		{SnapshotDate: time.Now().AddDate(0, 0, -21), GlucoseSlope: -0.5},
		{SnapshotDate: time.Now().AddDate(0, 0, -14), GlucoseSlope: -1.0},
		{SnapshotDate: time.Now().AddDate(0, 0, -7), GlucoseSlope: -1.5},
		{SnapshotDate: time.Now(), GlucoseSlope: -2.0},
	}

	results := engine.ComputeSecondDerivatives(snapshots)
	glucose := results[models.DomainGlucose]
	if glucose.SlopeOfSlope >= 0 {
		t.Errorf("SlopeOfSlope = %f, expected negative (accelerating decline)", glucose.SlopeOfSlope)
	}
	if glucose.Interpretation != "ACCELERATING_DECLINE" {
		t.Errorf("Interpretation = %q, want ACCELERATING_DECLINE", glucose.Interpretation)
	}
}

// TestComputeSecondDerivatives_StableDomain verifies that a domain
// with a near-zero second derivative produces STABLE.
func TestComputeSecondDerivatives_StableDomain(t *testing.T) {
	engine := NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, zap.NewNop())

	snapshots := []models.DomainTrajectoryHistory{
		{SnapshotDate: time.Now().AddDate(0, 0, -14), CardioSlope: 0.1},
		{SnapshotDate: time.Now().AddDate(0, 0, -7), CardioSlope: 0.1},
		{SnapshotDate: time.Now(), CardioSlope: 0.1},
	}

	results := engine.ComputeSecondDerivatives(snapshots)
	cardio := results[models.DomainCardio]
	if cardio.Interpretation != "STABLE" {
		t.Errorf("Interpretation = %q, want STABLE for flat slope-of-slope", cardio.Interpretation)
	}
}

// TestComputeSecondDerivatives_InsufficientSnapshots verifies that
// fewer than 3 snapshots returns nil (insufficient data for a
// meaningful second derivative).
func TestComputeSecondDerivatives_InsufficientSnapshots(t *testing.T) {
	engine := NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, zap.NewNop())

	snapshots := []models.DomainTrajectoryHistory{
		{SnapshotDate: time.Now().AddDate(0, 0, -7), GlucoseSlope: -1.0},
		{SnapshotDate: time.Now(), GlucoseSlope: -0.5},
	}

	results := engine.ComputeSecondDerivatives(snapshots)
	if results != nil {
		t.Errorf("expected nil for 2 snapshots, got %+v", results)
	}
}

// TestComputeSecondDerivatives_AcceleratingImprovement verifies
// that a domain whose positive slope is getting MORE positive
// produces ACCELERATING_IMPROVEMENT.
func TestComputeSecondDerivatives_AcceleratingImprovement(t *testing.T) {
	engine := NewTrajectoryEngine(config.DefaultTrajectoryThresholds(), nil, nil, zap.NewNop())

	snapshots := []models.DomainTrajectoryHistory{
		{SnapshotDate: time.Now().AddDate(0, 0, -14), GlucoseSlope: 0.5},
		{SnapshotDate: time.Now().AddDate(0, 0, -7), GlucoseSlope: 1.0},
		{SnapshotDate: time.Now(), GlucoseSlope: 1.5},
	}

	results := engine.ComputeSecondDerivatives(snapshots)
	glucose := results[models.DomainGlucose]
	if glucose.Interpretation != "ACCELERATING_IMPROVEMENT" {
		t.Errorf("Interpretation = %q, want ACCELERATING_IMPROVEMENT", glucose.Interpretation)
	}
}
