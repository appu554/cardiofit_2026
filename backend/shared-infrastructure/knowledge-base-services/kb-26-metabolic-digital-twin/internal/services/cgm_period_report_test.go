package services

import (
	"math"
	"testing"
	"time"
)

func TestComputePeriodReport_EmptyReadings_InsufficientData(t *testing.T) {
	start := time.Now().Add(-14 * 24 * time.Hour)
	end := time.Now()
	report := ComputePeriodReport(nil, start, end)
	if report.SufficientData {
		t.Error("expected SufficientData=false for empty readings")
	}
	if report.ConfidenceLevel != "LOW" {
		t.Errorf("expected LOW confidence, got %s", report.ConfidenceLevel)
	}
}

func TestComputePeriodReport_AllInRange_TIR100(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(14 * 24 * time.Hour)

	// Dense in-range readings: every 15 min for 14 days = 1344 readings at 140 mg/dL
	readings := make([]GlucoseReading, 0, 1344)
	for ts := start; ts.Before(end); ts = ts.Add(15 * time.Minute) {
		readings = append(readings, GlucoseReading{Timestamp: ts, ValueMgDL: 140.0})
	}

	report := ComputePeriodReport(readings, start, end)

	if !report.SufficientData {
		t.Errorf("expected SufficientData=true for dense readings, coverage=%.1f", report.CoveragePct)
	}
	if math.Abs(report.TIRPct-100.0) > 0.1 {
		t.Errorf("expected TIR=100, got %.2f", report.TIRPct)
	}
	if report.TBRL1Pct != 0 || report.TBRL2Pct != 0 || report.TARL1Pct != 0 || report.TARL2Pct != 0 {
		t.Errorf("expected all out-of-range = 0, got TBR1=%.1f TBR2=%.1f TAR1=%.1f TAR2=%.1f",
			report.TBRL1Pct, report.TBRL2Pct, report.TARL1Pct, report.TARL2Pct)
	}
	if math.Abs(report.MeanGlucose-140.0) > 0.1 {
		t.Errorf("expected mean=140, got %.2f", report.MeanGlucose)
	}
	// GMI at mean=140: 3.31 + 0.02392*140 = 6.6588
	expectedGMI := 3.31 + 0.02392*140.0
	if math.Abs(report.GMI-expectedGMI) > 0.01 {
		t.Errorf("expected GMI=%.4f, got %.4f", expectedGMI, report.GMI)
	}
	if report.GRIZone != "A" {
		t.Errorf("expected GRI zone A for ideal control, got %s", report.GRIZone)
	}
}

func TestComputePeriodReport_MixedRange_CorrectPercentages(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(14 * 24 * time.Hour)

	// Build 1344 readings with a precise distribution:
	// 60% in range (140), 10% TBR L1 (65), 5% TBR L2 (50),
	// 20% TAR L1 (210), 5% TAR L2 (280).
	// 1344 = 806 + 134 + 67 + 269 + 67 (sums to 1343 — close enough for percentage test)
	readings := make([]GlucoseReading, 0, 1344)
	ts := start
	add := func(count int, val float64) {
		for i := 0; i < count; i++ {
			readings = append(readings, GlucoseReading{Timestamp: ts, ValueMgDL: val})
			ts = ts.Add(15 * time.Minute)
		}
	}
	add(806, 140) // TIR ~60%
	add(134, 65)  // TBR L1 ~10%
	add(67, 50)   // TBR L2 ~5%
	add(269, 210) // TAR L1 ~20%
	add(67, 280)  // TAR L2 ~5%

	report := ComputePeriodReport(readings, start, end)

	// Allow 1% tolerance on each bucket.
	checks := []struct {
		name     string
		got      float64
		expected float64
	}{
		{"TIR", report.TIRPct, 60.0},
		{"TBR_L1", report.TBRL1Pct, 10.0},
		{"TBR_L2", report.TBRL2Pct, 5.0},
		{"TAR_L1", report.TARL1Pct, 20.0},
		{"TAR_L2", report.TARL2Pct, 5.0},
	}
	for _, c := range checks {
		if math.Abs(c.got-c.expected) > 1.0 {
			t.Errorf("%s: expected ~%.1f, got %.2f", c.name, c.expected, c.got)
		}
	}
}

func TestComputePeriodReport_OutsideWindow_Excluded(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(14 * 24 * time.Hour)

	readings := []GlucoseReading{
		{Timestamp: start.Add(-48 * time.Hour), ValueMgDL: 500}, // outside, ignored
		{Timestamp: end.Add(48 * time.Hour), ValueMgDL: 500},    // outside, ignored
		{Timestamp: start.Add(24 * time.Hour), ValueMgDL: 140},  // in window
	}

	report := ComputePeriodReport(readings, start, end)
	if math.Abs(report.MeanGlucose-140.0) > 0.01 {
		t.Errorf("expected mean=140 (only in-window reading counted), got %.2f",
			report.MeanGlucose)
	}
}

func TestComputePeriodReport_CVStability(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(14 * 24 * time.Hour)

	// Alternating 120 and 160 at mean=140, stdev=20 → CV = (20/140)*100 ≈ 14.3%
	// That's below 36% so GlucoseStable should be true.
	readings := make([]GlucoseReading, 0, 1344)
	ts := start
	for i := 0; i < 1344; i++ {
		v := 120.0
		if i%2 == 0 {
			v = 160.0
		}
		readings = append(readings, GlucoseReading{Timestamp: ts, ValueMgDL: v})
		ts = ts.Add(15 * time.Minute)
	}

	report := ComputePeriodReport(readings, start, end)

	if math.Abs(report.CVPct-14.286) > 0.5 {
		t.Errorf("expected CV ~14.3%%, got %.2f", report.CVPct)
	}
	if !report.GlucoseStable {
		t.Error("expected GlucoseStable=true at CV~14.3%")
	}
}
