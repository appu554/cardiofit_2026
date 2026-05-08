package kpis

import (
	"math"
	"testing"
)

// TestCPDActivityHours_SumByCategory is the verbatim plan test.
func TestCPDActivityHours_SumByCategory(t *testing.T) {
	acts := []ConfirmedActivity{
		{Category: "clinical", Hours: 3.0},
		{Category: "communication", Hours: 1.5},
		{Category: "clinical", Hours: 2.0},
	}
	got := CPDHoursByCategory(acts)

	if math.Abs(got["clinical"]-5.0) > 1e-9 {
		t.Errorf("expected clinical=5.0, got %v", got["clinical"])
	}
	if math.Abs(got["communication"]-1.5) > 1e-9 {
		t.Errorf("expected communication=1.5, got %v", got["communication"])
	}
}

// TestCPDActivityHours_Empty verifies an empty (non-nil) map is returned for
// empty input.
func TestCPDActivityHours_Empty(t *testing.T) {
	got := CPDHoursByCategory([]ConfirmedActivity{})
	if got == nil {
		t.Error("expected non-nil map for empty input, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

// TestCPDActivityHours_SingleActivity verifies a single activity is handled.
func TestCPDActivityHours_SingleActivity(t *testing.T) {
	acts := []ConfirmedActivity{
		{Category: "management", Hours: 4.5},
	}
	got := CPDHoursByCategory(acts)
	if math.Abs(got["management"]-4.5) > 1e-9 {
		t.Errorf("expected management=4.5, got %v", got["management"])
	}
}

// TestCPDActivityHours_FractionalHours verifies fractional hour accumulation.
func TestCPDActivityHours_FractionalHours(t *testing.T) {
	acts := []ConfirmedActivity{
		{Category: "clinical", Hours: 0.5},
		{Category: "clinical", Hours: 0.5},
		{Category: "clinical", Hours: 0.5},
	}
	got := CPDHoursByCategory(acts)
	if math.Abs(got["clinical"]-1.5) > 1e-9 {
		t.Errorf("expected clinical=1.5, got %v", got["clinical"])
	}
}
