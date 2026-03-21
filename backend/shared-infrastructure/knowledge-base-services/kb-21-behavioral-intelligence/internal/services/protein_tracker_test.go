package services

import (
	"math"
	"testing"

	"go.uber.org/zap"
)

func TestProteinTracker_TargetMid(t *testing.T) {
	pt := NewProteinTracker(zap.NewNop())
	mid := pt.TargetMid()
	if math.Abs(mid-1.0) > 0.001 {
		t.Errorf("TargetMid() = %f, want 1.0", mid)
	}
}

func TestProteinTracker_ComputeAdequacy(t *testing.T) {
	pt := NewProteinTracker(zap.NewNop())

	// 70kg patient, eating 70g protein/day = exactly target (1.0 g/kg * 70kg = 70g)
	adequacy := pt.ComputeAdequacy([]float64{70, 70, 70, 70, 70, 70, 70}, 70)
	if math.Abs(adequacy-1.0) > 0.001 {
		t.Errorf("Adequate protein = %f, want 1.0", adequacy)
	}

	// 70kg patient eating 35g/day = 0.5 adequacy
	adequacy = pt.ComputeAdequacy([]float64{35, 35, 35}, 70)
	if math.Abs(adequacy-0.5) > 0.001 {
		t.Errorf("Half protein = %f, want 0.5", adequacy)
	}

	// Zero weight → 0
	adequacy = pt.ComputeAdequacy([]float64{70}, 0)
	if adequacy != 0 {
		t.Errorf("Zero weight = %f, want 0", adequacy)
	}

	// Empty slice → 0
	adequacy = pt.ComputeAdequacy([]float64{}, 70)
	if adequacy != 0 {
		t.Errorf("Empty intake = %f, want 0", adequacy)
	}

	// Over-target → clamped to 1.0
	adequacy = pt.ComputeAdequacy([]float64{200}, 70)
	if adequacy != 1.0 {
		t.Errorf("Over target = %f, want 1.0", adequacy)
	}
}

func TestProteinTracker_CustomTargets(t *testing.T) {
	pt := NewProteinTrackerWithTargets(0.6, 0.8, zap.NewNop())
	mid := pt.TargetMid()
	if math.Abs(mid-0.7) > 0.001 {
		t.Errorf("Custom TargetMid() = %f, want 0.7", mid)
	}
}
