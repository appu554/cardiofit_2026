package services

import (
	"math"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// TestBaseline_SufficientData_7DayMedian
// ---------------------------------------------------------------------------

func TestBaseline_SufficientData_7DayMedian(t *testing.T) {
	// 10 readings within 7 days → median computed, confidence HIGH, lookback 7
	base := time.Now().UTC()
	readings := []float64{120, 118, 122, 119, 121, 117, 123, 120, 118, 122}
	timestamps := make([]time.Time, len(readings))
	for i := range readings {
		timestamps[i] = base.Add(-time.Duration(len(readings)-1-i) * 12 * time.Hour) // spread over ~5 days
	}

	snap := ComputeBaseline(readings, timestamps, 7)

	// sorted: [117, 118, 118, 119, 120, 120, 121, 122, 122, 123]
	// median of 10 even = avg(120, 120) = 120
	if snap.BaselineMedian != 120.0 {
		t.Errorf("expected median 120.0, got %.2f", snap.BaselineMedian)
	}
	if snap.Confidence != "HIGH" {
		t.Errorf("expected confidence HIGH, got %s", snap.Confidence)
	}
	if snap.LookbackDays != 7 {
		t.Errorf("expected lookback 7, got %d", snap.LookbackDays)
	}
	if snap.ReadingCount != 10 {
		t.Errorf("expected reading count 10, got %d", snap.ReadingCount)
	}
}

// ---------------------------------------------------------------------------
// TestBaseline_SparseData_14DayFallback
// ---------------------------------------------------------------------------

func TestBaseline_SparseData_14DayFallback(t *testing.T) {
	// 4 readings in last 7 days but 8 readings spanning 14 days
	// With lookback=14, all 8 should be captured → confidence MODERATE (3-6) or HIGH (7+)
	base := time.Now().UTC()

	// 4 older readings (8-12 days ago)
	readings := []float64{100, 102, 104, 106, 108, 110, 112, 114}
	timestamps := []time.Time{
		base.Add(-12 * 24 * time.Hour),
		base.Add(-11 * 24 * time.Hour),
		base.Add(-10 * 24 * time.Hour),
		base.Add(-9 * 24 * time.Hour),
		base.Add(-5 * 24 * time.Hour),
		base.Add(-3 * 24 * time.Hour),
		base.Add(-2 * 24 * time.Hour),
		base.Add(-1 * 24 * time.Hour),
	}

	snap := ComputeBaseline(readings, timestamps, 14)

	// sorted: [100, 102, 104, 106, 108, 110, 112, 114]
	// median of 8 even = avg(106, 108) = 107
	if snap.BaselineMedian != 107.0 {
		t.Errorf("expected median 107.0, got %.2f", snap.BaselineMedian)
	}
	if snap.Confidence != "HIGH" {
		t.Errorf("expected confidence HIGH (8 readings >= 7), got %s", snap.Confidence)
	}
	if snap.LookbackDays != 14 {
		t.Errorf("expected lookback 14, got %d", snap.LookbackDays)
	}
}

// ---------------------------------------------------------------------------
// TestBaseline_InsufficientData_LowConfidence
// ---------------------------------------------------------------------------

func TestBaseline_InsufficientData_LowConfidence(t *testing.T) {
	// Only 2 readings → confidence LOW
	base := time.Now().UTC()
	readings := []float64{130, 140}
	timestamps := []time.Time{
		base.Add(-2 * 24 * time.Hour),
		base.Add(-1 * 24 * time.Hour),
	}

	snap := ComputeBaseline(readings, timestamps, 7)

	// median of 2 = avg(130, 140) = 135
	if snap.BaselineMedian != 135.0 {
		t.Errorf("expected median 135.0, got %.2f", snap.BaselineMedian)
	}
	if snap.Confidence != "LOW" {
		t.Errorf("expected confidence LOW, got %s", snap.Confidence)
	}
	if snap.ReadingCount != 2 {
		t.Errorf("expected reading count 2, got %d", snap.ReadingCount)
	}
}

// ---------------------------------------------------------------------------
// TestBaseline_MAD_Computation
// ---------------------------------------------------------------------------

func TestBaseline_MAD_Computation(t *testing.T) {
	// Known values: [100, 102, 98, 105, 97]
	// sorted: [97, 98, 100, 102, 105] → median = 100
	// deviations from median: |97-100|=3, |98-100|=2, |100-100|=0, |102-100|=2, |105-100|=5
	// sorted deviations: [0, 2, 2, 3, 5] → MAD = median = 2.0
	base := time.Now().UTC()
	readings := []float64{100, 102, 98, 105, 97}
	timestamps := make([]time.Time, len(readings))
	for i := range readings {
		timestamps[i] = base.Add(-time.Duration(len(readings)-1-i) * 24 * time.Hour)
	}

	snap := ComputeBaseline(readings, timestamps, 7)

	if snap.BaselineMedian != 100.0 {
		t.Errorf("expected median 100.0, got %.2f", snap.BaselineMedian)
	}
	if math.Abs(snap.BaselineMAD-2.0) > 0.001 {
		t.Errorf("expected MAD 2.0, got %.4f", snap.BaselineMAD)
	}
	if snap.Confidence != "MODERATE" {
		t.Errorf("expected confidence MODERATE (5 readings), got %s", snap.Confidence)
	}
}

// ---------------------------------------------------------------------------
// TestBaseline_Refresh_NewReading
// ---------------------------------------------------------------------------

func TestBaseline_Refresh_NewReading(t *testing.T) {
	// Existing readings [100, 102, 98] → median 100
	// Add reading 110 → [98, 100, 102, 110] → median = avg(100, 102) = 101
	base := time.Now().UTC()
	readings := []float64{100, 102, 98, 110}
	timestamps := []time.Time{
		base.Add(-3 * 24 * time.Hour),
		base.Add(-2 * 24 * time.Hour),
		base.Add(-1 * 24 * time.Hour),
		base,
	}

	snap := ComputeBaseline(readings, timestamps, 7)

	if snap.BaselineMedian != 101.0 {
		t.Errorf("expected median 101.0 after adding new reading, got %.2f", snap.BaselineMedian)
	}
	if snap.ReadingCount != 4 {
		t.Errorf("expected reading count 4, got %d", snap.ReadingCount)
	}
}

// ---------------------------------------------------------------------------
// TestBaseline_IdenticalReadings_ZeroMAD
// ---------------------------------------------------------------------------

func TestBaseline_IdenticalReadings_ZeroMAD(t *testing.T) {
	// All readings 120 → MAD = 0, no panic
	base := time.Now().UTC()
	readings := []float64{120, 120, 120, 120, 120}
	timestamps := make([]time.Time, len(readings))
	for i := range readings {
		timestamps[i] = base.Add(-time.Duration(len(readings)-1-i) * 24 * time.Hour)
	}

	snap := ComputeBaseline(readings, timestamps, 7)

	if snap.BaselineMedian != 120.0 {
		t.Errorf("expected median 120.0, got %.2f", snap.BaselineMedian)
	}
	if snap.BaselineMAD != 0.0 {
		t.Errorf("expected MAD 0.0, got %.4f", snap.BaselineMAD)
	}
	if snap.ReadingCount != 5 {
		t.Errorf("expected reading count 5, got %d", snap.ReadingCount)
	}
	if snap.Confidence != "MODERATE" {
		t.Errorf("expected confidence MODERATE (5 readings), got %s", snap.Confidence)
	}
}
