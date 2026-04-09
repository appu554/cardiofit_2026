package services

import (
	"fmt"
	"math"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// StaleEGFRResult — output of the stale-eGFR detection
// ---------------------------------------------------------------------------

// StaleEGFRResult describes whether a patient's eGFR measurement is overdue
// relative to their CKD stage and medication profile.
type StaleEGFRResult struct {
	IsStale        bool   `json:"is_stale"`
	DaysSince      int    `json:"days_since"`
	ExpectedMaxDays int   `json:"expected_max_days"`
	Severity       string `json:"severity"` // OK | WARNING | CRITICAL
	Action         string `json:"action"`
}

// ---------------------------------------------------------------------------
// DetectStaleEGFR — CKD-stage-aware staleness detection
// ---------------------------------------------------------------------------

// DetectStaleEGFR evaluates whether a patient's most recent eGFR measurement
// is overdue based on their renal function level and medication profile.
//
// CKD-stage-aware intervals:
//   - eGFR <30  → 30 days  (monthly)
//   - eGFR 30-45 → 90 days  (quarterly)
//   - eGFR 45-60 → 180 days (biannual)
//   - eGFR >60  → 365 days (annual)
//
// If onRenalSensitiveMed is true, the maximum interval is tightened to 90 days.
// Severity is CRITICAL if days since measurement exceeds CriticalDays from config.
func DetectStaleEGFR(renal models.RenalStatus, cfg StaleEGFRConfig, onRenalSensitiveMed bool) StaleEGFRResult {
	now := time.Now()
	daysSince := int(math.Round(now.Sub(renal.EGFRMeasuredAt).Hours() / 24))

	// Determine expected max days based on eGFR level.
	var expectedMax int
	switch {
	case renal.EGFR < 30:
		expectedMax = 30
	case renal.EGFR < 45:
		expectedMax = 90
	case renal.EGFR < 60:
		expectedMax = 180
	default:
		expectedMax = 365
	}

	// Tighten to 90 days if on renal-sensitive medication.
	if onRenalSensitiveMed && expectedMax > 90 {
		expectedMax = 90
	}

	result := StaleEGFRResult{
		DaysSince:      daysSince,
		ExpectedMaxDays: expectedMax,
	}

	if daysSince <= expectedMax {
		result.Severity = "OK"
		result.Action = "no action required"
		return result
	}

	result.IsStale = true

	if daysSince > cfg.CriticalDays {
		result.Severity = "CRITICAL"
		result.Action = fmt.Sprintf("eGFR is %d days old (critical threshold: %d days); order urgent renal function panel",
			daysSince, cfg.CriticalDays)
	} else {
		result.Severity = "WARNING"
		result.Action = fmt.Sprintf("eGFR is %d days old (expected within %d days); schedule renal function panel",
			daysSince, expectedMax)
	}

	return result
}
