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
