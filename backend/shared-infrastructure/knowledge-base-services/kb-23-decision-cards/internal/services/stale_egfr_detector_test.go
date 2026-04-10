package services

import (
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// TestStaleEGFR_RecentMeasurement
// ---------------------------------------------------------------------------

func TestStaleEGFR_RecentMeasurement(t *testing.T) {
	// 30 days old, eGFR 55 → within 180-day window → OK
	rs := models.RenalStatus{
		EGFR:           55,
		EGFRMeasuredAt: time.Now().AddDate(0, 0, -30),
	}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(rs, cfg, false)

	if result.IsStale {
		t.Error("expected not stale for 30-day-old measurement at eGFR 55")
	}
	if result.Severity != "OK" {
		t.Errorf("expected severity OK, got %s", result.Severity)
	}
}

// ---------------------------------------------------------------------------
// TestStaleEGFR_OverdueForCKDStage
// ---------------------------------------------------------------------------

func TestStaleEGFR_OverdueForCKDStage(t *testing.T) {
	// 120 days old, eGFR 42 → CKD 3b (30-45 range) → expected 90 days → WARNING
	rs := models.RenalStatus{
		EGFR:           42,
		EGFRMeasuredAt: time.Now().AddDate(0, 0, -120),
	}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(rs, cfg, false)

	if !result.IsStale {
		t.Error("expected stale for 120-day-old measurement at eGFR 42")
	}
	if result.ExpectedMaxDays != 90 {
		t.Errorf("expected max days 90 for eGFR 42, got %d", result.ExpectedMaxDays)
	}
	if result.Severity != "WARNING" {
		t.Errorf("expected WARNING severity, got %s", result.Severity)
	}
}

// ---------------------------------------------------------------------------
// TestStaleEGFR_CriticallyStale
// ---------------------------------------------------------------------------

func TestStaleEGFR_CriticallyStale(t *testing.T) {
	// 200 days old → exceeds critical threshold of 180 → CRITICAL
	rs := models.RenalStatus{
		EGFR:           42,
		EGFRMeasuredAt: time.Now().AddDate(0, 0, -200),
	}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(rs, cfg, false)

	if !result.IsStale {
		t.Error("expected stale for 200-day-old measurement")
	}
	if result.Severity != "CRITICAL" {
		t.Errorf("expected CRITICAL severity, got %s", result.Severity)
	}
}

// ---------------------------------------------------------------------------
// TestStaleEGFR_RenalSensitiveMedTightensMonitoring
// ---------------------------------------------------------------------------

func TestStaleEGFR_RenalSensitiveMedTightensMonitoring(t *testing.T) {
	// eGFR 65, 100 days old, onMed=true → tightened to 90 days → stale
	rs := models.RenalStatus{
		EGFR:           65,
		EGFRMeasuredAt: time.Now().AddDate(0, 0, -100),
	}
	cfg := StaleEGFRConfig{WarningDays: 90, CriticalDays: 180}

	result := DetectStaleEGFR(rs, cfg, true)

	if !result.IsStale {
		t.Error("expected stale when on renal-sensitive med (90d max) and 100 days old")
	}
	if result.ExpectedMaxDays != 90 {
		t.Errorf("expected max days tightened to 90, got %d", result.ExpectedMaxDays)
	}
	if result.Severity != "WARNING" {
		t.Errorf("expected WARNING severity, got %s", result.Severity)
	}
}

// ===========================================================================
// Stale Potassium Tests
// ===========================================================================

func TestStalePotassium_NotOnKDrug_NoMonitoringNeeded(t *testing.T) {
	rs := models.RenalStatus{EGFR: 50}
	result := DetectStalePotassium(rs, false, false)
	if result.IsStale {
		t.Error("should not flag stale K+ when not on K+-affecting drug")
	}
}

func TestStalePotassium_NeverMeasured_Critical(t *testing.T) {
	rs := models.RenalStatus{EGFR: 40, PotassiumMeasuredAt: nil}
	result := DetectStalePotassium(rs, true, false)
	if !result.IsStale {
		t.Error("expected stale when K+ never measured on K+-affecting drug")
	}
	if result.Severity != "CRITICAL" {
		t.Errorf("expected CRITICAL, got %s", result.Severity)
	}
}

func TestStalePotassium_CombinationTherapy_Monthly(t *testing.T) {
	// ACEi + MRA combo: 45 days since K+ → stale (30-day max for combo)
	kDate := time.Now().AddDate(0, 0, -45)
	k := 4.5
	rs := models.RenalStatus{EGFR: 50, Potassium: &k, PotassiumMeasuredAt: &kDate}
	result := DetectStalePotassium(rs, true, true)
	if !result.IsStale {
		t.Error("expected stale: 45 days on combo therapy (30-day max)")
	}
	if result.ExpectedMaxDays != 30 {
		t.Errorf("expected 30-day max for combination, got %d", result.ExpectedMaxDays)
	}
}

func TestStalePotassium_SingleDrug_LowEGFR_Monthly(t *testing.T) {
	// Single ACEi at eGFR 38: 35 days → stale (30-day max when eGFR <45)
	kDate := time.Now().AddDate(0, 0, -35)
	k := 4.8
	rs := models.RenalStatus{EGFR: 38, Potassium: &k, PotassiumMeasuredAt: &kDate}
	result := DetectStalePotassium(rs, true, false)
	if !result.IsStale {
		t.Error("expected stale: 35 days on single K+-drug at eGFR <45")
	}
	if result.ExpectedMaxDays != 30 {
		t.Errorf("expected 30-day max at eGFR <45, got %d", result.ExpectedMaxDays)
	}
}

func TestStalePotassium_SingleDrug_NormalEGFR_Quarterly(t *testing.T) {
	// Single ACEi at eGFR 65: 60 days → not stale (90-day max)
	kDate := time.Now().AddDate(0, 0, -60)
	k := 4.2
	rs := models.RenalStatus{EGFR: 65, Potassium: &k, PotassiumMeasuredAt: &kDate}
	result := DetectStalePotassium(rs, true, false)
	if result.IsStale {
		t.Error("should not flag stale: 60 days at eGFR 65 (90-day max)")
	}
}
