package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestEvaluateOverlap_InsideBandPasses(t *testing.T) {
	got := EvaluateOverlap(0.50, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if got != models.OverlapPass {
		t.Fatalf("want PASS, got %s", got)
	}
}

func TestEvaluateOverlap_BelowFloor(t *testing.T) {
	got := EvaluateOverlap(0.02, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if got != models.OverlapBelowFloor {
		t.Fatalf("want BELOW_FLOOR, got %s", got)
	}
}

func TestEvaluateOverlap_AboveCeiling(t *testing.T) {
	got := EvaluateOverlap(0.99, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if got != models.OverlapAboveCeiling {
		t.Fatalf("want ABOVE_CEILING, got %s", got)
	}
}
