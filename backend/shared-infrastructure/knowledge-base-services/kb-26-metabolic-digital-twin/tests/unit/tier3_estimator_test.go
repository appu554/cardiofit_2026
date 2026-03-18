package unit

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/services"
)

func TestEstimateInsulinSensitivity_HOMAIR(t *testing.T) {
	ev := services.EstimateInsulinSensitivity(100, 15, true)
	if ev.Method != "HOMA_IR" {
		t.Errorf("expected HOMA_IR, got %s", ev.Method)
	}
	if ev.Confidence < 0.60 {
		t.Errorf("HOMA-IR confidence should be >= 0.60, got %f", ev.Confidence)
	}
}

func TestEstimateInsulinSensitivity_Fallback(t *testing.T) {
	ev := services.EstimateInsulinSensitivity(130, 0, false)
	if ev.Method != "TRAJECTORY_FALLBACK" {
		t.Errorf("expected TRAJECTORY_FALLBACK, got %s", ev.Method)
	}
	if ev.Confidence > 0.50 {
		t.Errorf("fallback confidence should be <= 0.50, got %f", ev.Confidence)
	}
}

func TestEstimateHepaticGlucoseOutput(t *testing.T) {
	ev := services.EstimateHepaticGlucoseOutput(true, 140, 120)
	if ev.Classification != "HIGH" {
		t.Errorf("dawn + high FBG/PPBG ratio → expected HIGH, got %s", ev.Classification)
	}
}

func TestEstimateMuscleMassProxy(t *testing.T) {
	ev := services.EstimateMuscleMassProxy(80, 1.2, 5000, true)
	if ev.Value < 0 || ev.Value > 1 {
		t.Errorf("muscle mass proxy should be 0-1, got %f", ev.Value)
	}
}
