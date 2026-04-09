package services

import (
	"math"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestComputeEGFRTrajectory_RapidDecline
// ---------------------------------------------------------------------------

func TestComputeEGFRTrajectory_RapidDecline(t *testing.T) {
	// 5 readings over 1 year declining from 55 → 35 (slope ≈ -20/year)
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	readings := []EGFRReading{
		{Value: 55, MeasuredAt: base},
		{Value: 50, MeasuredAt: base.AddDate(0, 3, 0)},
		{Value: 45, MeasuredAt: base.AddDate(0, 6, 0)},
		{Value: 40, MeasuredAt: base.AddDate(0, 9, 0)},
		{Value: 35, MeasuredAt: base.AddDate(1, 0, 0)},
	}

	result, err := ComputeEGFRTrajectory(readings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Slope >= -5.0 {
		t.Errorf("expected slope < -5.0 (rapid decline), got %.2f", result.Slope)
	}
	if result.Classification != "RAPID_DECLINE" {
		t.Errorf("expected RAPID_DECLINE, got %s", result.Classification)
	}
	if !result.IsRapidDecliner {
		t.Error("expected IsRapidDecliner = true")
	}
	if result.DataPoints != 5 {
		t.Errorf("expected 5 data points, got %d", result.DataPoints)
	}
	if result.LatestEGFR != 35 {
		t.Errorf("expected latest eGFR 35, got %.1f", result.LatestEGFR)
	}
}

// ---------------------------------------------------------------------------
// TestComputeEGFRTrajectory_Stable
// ---------------------------------------------------------------------------

func TestComputeEGFRTrajectory_Stable(t *testing.T) {
	// 5 readings ~60 ± 0.5 over 1 year
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	readings := []EGFRReading{
		{Value: 60.0, MeasuredAt: base},
		{Value: 60.5, MeasuredAt: base.AddDate(0, 3, 0)},
		{Value: 59.5, MeasuredAt: base.AddDate(0, 6, 0)},
		{Value: 60.2, MeasuredAt: base.AddDate(0, 9, 0)},
		{Value: 60.0, MeasuredAt: base.AddDate(1, 0, 0)},
	}

	result, err := ComputeEGFRTrajectory(readings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Classification != "STABLE" {
		t.Errorf("expected STABLE, got %s", result.Classification)
	}
	if result.IsRapidDecliner {
		t.Error("expected IsRapidDecliner = false for stable readings")
	}
	if math.Abs(result.Slope) > 1.0 {
		t.Errorf("expected slope magnitude < 1.0 for stable, got %.2f", result.Slope)
	}
}

// ---------------------------------------------------------------------------
// TestComputeEGFRTrajectory_InsufficientData
// ---------------------------------------------------------------------------

func TestComputeEGFRTrajectory_InsufficientData(t *testing.T) {
	readings := []EGFRReading{
		{Value: 50, MeasuredAt: time.Now()},
	}

	_, err := ComputeEGFRTrajectory(readings)
	if err == nil {
		t.Fatal("expected error for insufficient data, got nil")
	}
	if got := err.Error(); !contains(got, "insufficient") {
		t.Errorf("expected error containing 'insufficient', got %q", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// TestProjectTimeToThreshold — table-driven
// ---------------------------------------------------------------------------

func TestProjectTimeToThreshold(t *testing.T) {
	tests := []struct {
		name      string
		current   float64
		slope     float64
		threshold float64
		wantNil   bool
		wantMonth float64 // approximate, ±1.0 tolerance
	}{
		{
			name:      "metformin_contraindication",
			current:   48,
			slope:     -8,
			threshold: 30,
			wantMonth: 27.0, // (48-30)/8 * 12 = 27
		},
		{
			name:      "sglt2i_contraindication",
			current:   48,
			slope:     -8,
			threshold: 20,
			wantMonth: 42.0, // (48-20)/8 * 12 = 42
		},
		{
			name:      "stable_slow_decline",
			current:   48,
			slope:     -0.5,
			threshold: 30,
			wantMonth: 432.0, // (48-30)/0.5 * 12 = 432
		},
		{
			name:    "improving_no_threshold",
			current: 48,
			slope:   2.0,
			threshold: 30,
			wantNil: true,
		},
		{
			name:    "already_below",
			current: 25,
			slope:   -5,
			threshold: 30,
			wantNil: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ProjectTimeToThreshold(tc.current, tc.slope, tc.threshold)
			if tc.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %.2f", *got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected ~%.1f months, got nil", tc.wantMonth)
			}
			if math.Abs(*got-tc.wantMonth) > 1.0 {
				t.Errorf("expected ~%.1f months, got %.2f", tc.wantMonth, *got)
			}
		})
	}
}
