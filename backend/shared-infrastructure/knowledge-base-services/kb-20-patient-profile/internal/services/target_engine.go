package services

import (
	"time"

	"kb-patient-profile/internal/models"
)

// ────────────────────────────────────────────────────────────────────────────
// Personalised Clinical Target Engine (A1)
//
// Computes per-patient FBG, HbA1c, SBP, and eGFR targets from demographics,
// CKD stage, diabetes duration, comorbidities, and PREVENT risk tier. These
// targets feed Module 13 CKM velocity and state-change detectors, replacing
// hardcoded population defaults with clinically stratified goals.
//
// Clinical references:
//   ADA Standards of Care 2024 — Chapter 6 (Glycemic Targets)
//   KDIGO 2024 — Blood Pressure in CKD, eGFR evaluation
//   AHA PREVENT 2024 — SBP target stratification
// ────────────────────────────────────────────────────────────────────────────

// Population-level defaults (used when no personalisation factors apply).
const (
	DefaultFBGTarget           = 110.0 // mg/dL
	DefaultHbA1cTarget         = 7.0   // %
	DefaultSBPTarget           = 130.0 // mmHg
	DefaultSBPKidneyThreshold  = 140.0 // mmHg
	DefaultEGFRThreshold       = 45.0  // mL/min/1.73m² (G3b boundary)
)

// ComputePersonalizedTargets derives per-patient clinical targets from the
// patient profile, latest eGFR, UACR, and PREVENT risk tier. All inputs
// are nullable — missing data falls back to population defaults.
func ComputePersonalizedTargets(
	profile models.PatientProfile,
	latestEGFR *float64,
	latestUACR *float64,
	preventSBPTarget *float64,
) models.PersonalizedTargets {

	targets := models.PersonalizedTargets{
		PatientID:  profile.PatientID,
		ComputedAt: time.Now().UTC(),
	}

	targets.FBGTarget, targets.Rationale.FBGReason = computeFBGTarget(profile)
	targets.HbA1cTarget, targets.Rationale.HbA1cReason = computeHbA1cTarget(profile)
	targets.SBPTarget, targets.Rationale.SBPReason = computeSBPTarget(profile, latestEGFR, latestUACR, preventSBPTarget)
	targets.SBPKidneyThreshold = computeSBPKidneyThreshold(profile, latestUACR)
	targets.EGFRThreshold, targets.Rationale.EGFRReason = computeEGFRThreshold(profile, latestEGFR)

	return targets
}

// ── FBG Target ──────────────────────────────────────────────────────────────
// ADA 2024: 80–130 mg/dL preprandial for most adults with diabetes.
// Relaxed to <130 for elderly, long-duration DM, CKD G4+, or HF.
// Tightened to <100 for newly diagnosed, young, low hypoglycemia risk.

func computeFBGTarget(p models.PatientProfile) (float64, string) {
	hasDM := p.DMType == "T1DM" || p.DMType == "T2DM"
	hasHF := containsAny(p.Comorbidities, "HF", "HEART_FAILURE", "HFrEF", "HFpEF", "HFmrEF")

	// Relaxation criteria (ADA 2024 §6.5: less stringent goals)
	if p.Age >= 75 {
		return 130.0, "ADA 2024: relaxed FBG for age ≥75"
	}
	if hasDM && p.DMDurationYears > 20 {
		return 130.0, "ADA 2024: relaxed FBG for DM duration >20y"
	}
	if p.CKDStage == models.CKDG4 || p.CKDStage == models.CKDG5 {
		return 130.0, "KDIGO 2024: relaxed FBG for CKD G4/G5"
	}
	if hasHF {
		return 130.0, "ADA 2024: relaxed FBG with heart failure"
	}

	// Tightening criteria (ADA 2024 §6.3: more stringent goals)
	if hasDM && p.DMDurationYears < 5 && p.Age < 50 {
		return 100.0, "ADA 2024: tightened FBG for young, short-duration DM"
	}

	return DefaultFBGTarget, "ADA 2024: standard FBG target"
}

// ── HbA1c Target ────────────────────────────────────────────────────────────
// ADA 2024: <7.0% for most adults. <8.0% for elderly/complex comorbidities.
// <6.5% for newly diagnosed without significant CVD.

func computeHbA1cTarget(p models.PatientProfile) (float64, string) {
	hasDM := p.DMType == "T1DM" || p.DMType == "T2DM"
	hasHF := containsAny(p.Comorbidities, "HF", "HEART_FAILURE", "HFrEF", "HFpEF", "HFmrEF")

	// Relaxation: elderly, long DM, advanced CKD, or HF
	if p.Age >= 75 {
		return 8.0, "ADA 2024: relaxed HbA1c for age ≥75"
	}
	if hasDM && p.DMDurationYears > 15 {
		return 8.0, "ADA 2024: relaxed HbA1c for DM duration >15y"
	}
	if p.CKDStage == models.CKDG4 || p.CKDStage == models.CKDG5 {
		return 8.0, "KDIGO 2024: relaxed HbA1c for CKD G4/G5"
	}
	if hasHF {
		return 8.0, "ADA 2024: relaxed HbA1c with heart failure"
	}

	// Tightening: newly diagnosed, young, no CVD
	if hasDM && p.DMDurationYears < 5 && p.Age < 50 && !p.HasClinicalCVD {
		return 6.5, "ADA 2024: tightened HbA1c for young, short-duration DM, no CVD"
	}

	return DefaultHbA1cTarget, "ADA 2024: standard HbA1c target"
}

// ── SBP Target ──────────────────────────────────────────────────────────────
// KDIGO 2024: <120 mmHg for high PREVENT risk or proteinuria (UACR >30).
// AHA PREVENT: stratified 120 vs 130 based on 10-year CVD risk.
// Relaxed to 140 for elderly ≥80 without proteinuria.

func computeSBPTarget(
	p models.PatientProfile,
	egfr *float64,
	uacr *float64,
	preventTarget *float64,
) (float64, string) {
	// Elderly relaxation (KDIGO 2024 + HYVET trial)
	if p.Age >= 80 && (uacr == nil || *uacr <= 30) {
		return 140.0, "KDIGO 2024: relaxed SBP for age ≥80 without proteinuria"
	}

	// Proteinuria tightening (KDIGO 2024: SBP <120 if UACR >30 mg/g)
	if uacr != nil && *uacr > 30 {
		return 120.0, "KDIGO 2024: tightened SBP for proteinuria (UACR >30)"
	}

	// Use PREVENT-computed target when available (already stratified by risk tier)
	if preventTarget != nil && *preventTarget > 0 {
		return *preventTarget, "AHA PREVENT: risk-stratified SBP target"
	}

	// CKD G3b+ default tightening
	if egfr != nil && *egfr < 45 {
		return 120.0, "KDIGO 2024: tightened SBP for CKD G3b+"
	}

	return DefaultSBPTarget, "Standard SBP target"
}

// ── SBP Kidney Threshold ────────────────────────────────────────────────────
// Used by Module 13 CKMRiskComputer for renal velocity BP-kidney factor.
// Lower threshold = more sensitive to BP-kidney impact.

func computeSBPKidneyThreshold(p models.PatientProfile, uacr *float64) float64 {
	// Proteinuria present: more aggressive kidney-protective threshold
	if uacr != nil && *uacr > 30 {
		return 130.0
	}
	// Elderly: relaxed threshold
	if p.Age >= 80 {
		return 150.0
	}
	return DefaultSBPKidneyThreshold
}

// ── eGFR Threshold ──────────────────────────────────────────────────────────
// Determines the eGFR boundary that triggers RENAL_RAPID_DECLINE alerts.
// For patients already in CKD G3a, shift the alert boundary to the G4 cutoff
// (eGFR <30) to avoid constant false-positive alerts.

func computeEGFRThreshold(p models.PatientProfile, egfr *float64) (float64, string) {
	// Already in G3a (eGFR 45-59): alert on progression to G3b
	if p.CKDStage == models.CKDG3a {
		return 45.0, "KDIGO 2024: G3a baseline, alert on G3b progression"
	}
	// Already in G3b (eGFR 30-44): alert on progression to G4
	if p.CKDStage == models.CKDG3b {
		return 30.0, "KDIGO 2024: G3b baseline, alert on G4 progression"
	}
	// Already in G4 (eGFR 15-29): alert on progression to G5
	if p.CKDStage == models.CKDG4 {
		return 15.0, "KDIGO 2024: G4 baseline, alert on G5 progression"
	}
	// G5 or dialysis: no further eGFR threshold alerts meaningful
	if p.CKDStage == models.CKDG5 {
		return 0.0, "KDIGO 2024: G5 baseline, no further eGFR threshold"
	}

	return DefaultEGFRThreshold, "Standard eGFR rapid-decline threshold (G3b boundary)"
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func containsAny(arr []string, targets ...string) bool {
	for _, a := range arr {
		for _, t := range targets {
			if a == t {
				return true
			}
		}
	}
	return false
}
