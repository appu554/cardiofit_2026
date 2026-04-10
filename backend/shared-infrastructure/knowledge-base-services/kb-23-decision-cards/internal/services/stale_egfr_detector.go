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

// ---------------------------------------------------------------------------
// StalePotassiumResult — output of the stale-potassium detection
// ---------------------------------------------------------------------------

// StalePotassiumResult describes whether a patient's potassium measurement
// is overdue given their K+-affecting medication profile.
type StalePotassiumResult struct {
	IsStale        bool   `json:"is_stale"`
	DaysSince      int    `json:"days_since"`
	ExpectedMaxDays int   `json:"expected_max_days"`
	Severity       string `json:"severity"` // OK | WARNING | CRITICAL
	Action         string `json:"action"`
}

// DetectStalePotassium evaluates potassium measurement freshness for patients
// on K+-affecting medications. Per KDIGO 2024:
//   - ACEi/ARB + MRA combination → monthly (30 days)
//   - Any single K+-affecting drug at eGFR <45 → monthly (30 days)
//   - Any single K+-affecting drug at eGFR ≥45 → quarterly (90 days)
//   - No K+-affecting drugs → no potassium monitoring required
//
// Returns a zero-value result (IsStale=false, Severity="") if potassium
// monitoring is not clinically indicated.
func DetectStalePotassium(
	renal models.RenalStatus,
	onKAffectingDrug bool,
	onMultipleKAffectingDrugs bool,
) StalePotassiumResult {
	// If not on K+-affecting drugs, potassium monitoring not required
	if !onKAffectingDrug {
		return StalePotassiumResult{Severity: "OK"}
	}

	// If potassium was never measured, flag immediately
	if renal.PotassiumMeasuredAt == nil {
		return StalePotassiumResult{
			IsStale:  true,
			Severity: "CRITICAL",
			Action:   "Potassium never measured — check urgently given K+-affecting medication",
		}
	}

	now := time.Now()
	daysSince := int(math.Round(now.Sub(*renal.PotassiumMeasuredAt).Hours() / 24))

	// Determine expected interval
	expectedMax := 90 // quarterly default for single K+-drug at eGFR ≥45

	if onMultipleKAffectingDrugs {
		expectedMax = 30 // monthly for combination (ACEi/ARB + MRA/FINERENONE)
	} else if renal.EGFR < 45 {
		expectedMax = 30 // monthly when eGFR <45 on any K+-drug
	}

	result := StalePotassiumResult{
		DaysSince:       daysSince,
		ExpectedMaxDays: expectedMax,
	}

	if daysSince <= expectedMax {
		result.Severity = "OK"
		return result
	}

	result.IsStale = true
	if daysSince > expectedMax*2 {
		result.Severity = "CRITICAL"
		result.Action = fmt.Sprintf("Potassium is %d days old on K+-affecting medication (expected every %d days); check urgently before continuing",
			daysSince, expectedMax)
	} else {
		result.Severity = "WARNING"
		result.Action = fmt.Sprintf("Potassium is %d days old (expected every %d days per KDIGO); schedule serum potassium",
			daysSince, expectedMax)
	}

	return result
}
