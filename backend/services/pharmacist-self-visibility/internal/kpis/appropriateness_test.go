package kpis

import (
	"math"
	"testing"
)

// TestAppropriateness_MeanRolling90 is the verbatim plan test:
// mean of [3.5, 4.0, 4.5, 4.2] ≈ 4.05.
func TestAppropriateness_MeanRolling90(t *testing.T) {
	got := MeanAppropriateness([]float64{3.5, 4.0, 4.5, 4.2})
	want := 4.05
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("expected ~4.05, got %v", got)
	}
}

// TestAppropriateness_Empty verifies the zero sentinel (not NaN) for empty input.
func TestAppropriateness_Empty(t *testing.T) {
	got := MeanAppropriateness([]float64{})
	if got != 0 {
		t.Errorf("expected 0 sentinel, got %v", got)
	}
	if math.IsNaN(got) {
		t.Errorf("expected 0, not NaN, for empty input")
	}
}

// TestAppropriateness_SingleValue verifies mean of one element.
func TestAppropriateness_SingleValue(t *testing.T) {
	got := MeanAppropriateness([]float64{4.2})
	if math.Abs(got-4.2) > 1e-9 {
		t.Errorf("expected 4.2, got %v", got)
	}
}
