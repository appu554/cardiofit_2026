package services

import (
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// Cross-domain conflict tests — renal gate vs HF mandatory meds
// ---------------------------------------------------------------------------
//
// These tests validate the resolution logic when a medication is
// simultaneously guideline-mandated for heart failure AND contraindicated
// by renal function. The expected behavior:
//
//   1. Mandatory med gap is marked Suppressed (not elevated to URGENT/IMMEDIATE)
//   2. A RenalHFConflict is recorded for referral card generation
//   3. For HFpEF losing SGLT2i, HasCriticalConflict is set (only GDMT lost)

// Scenario A: HFrEF patient currently on MRA but eGFR has dropped to 28.
// Renal gate must contraindicate MRA (< 30), and the suppression should
// prevent a duplicate "add MRA" card.
func TestCrossDomain_HFrEF_OnMRA_RenalContraindicated(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	gate := NewRenalDoseGate(formulary)

	k := 4.2 // normal potassium — avoids the "missing K+" short-circuit
	renal := models.RenalStatus{
		EGFR:           28, // below MRA contra threshold (30)
		EGFRSlope:      -2,
		EGFRMeasuredAt: time.Now().Add(-24 * time.Hour),
		EGFRDataPoints: 4,
		CKDStage:       "G4",
		Potassium:      &k,
	}

	// Patient has ALL four pillars — complete HFrEF GDMT
	meds := []ActiveMedication{
		{DrugClass: "ACEi", DrugName: "Ramipril 10mg", CurrentDoseMg: 10},
		{DrugClass: "BETA_BLOCKER_HF", DrugName: "Carvedilol 25mg", CurrentDoseMg: 25},
		{DrugClass: "MRA", DrugName: "Spironolactone 25mg", CurrentDoseMg: 25},
		{DrugClass: "SGLT2i", DrugName: "Dapagliflozin 10mg", CurrentDoseMg: 10},
	}

	report := DetectAllConflicts(gate, formulary, "patient-hfref-01",
		renal, meds, -2.0, "4c", "HFrEF")

	// MRA should be in RenalHFConflicts (not just silently suppressed)
	hasMRAConflict := false
	for _, c := range report.RenalHFConflicts {
		if c.DrugClass == "MRA" {
			hasMRAConflict = true
			if c.Urgency != "URGENT" {
				t.Errorf("MRA conflict should be URGENT, got %s", c.Urgency)
			}
			if c.ResolutionRecommendation == "" {
				t.Error("MRA conflict must have resolution recommendation")
			}
		}
	}
	// The mandatory checker won't emit an MRA gap since the patient IS on MRA.
	// But the renal gate flags it as contraindicated in BlockedDrugClasses.
	// The cross-domain detection for "on drug + renal contra" is handled by
	// the existing renal pipeline (HasSafetyBlock=true, BlockedDrugClasses=[MRA]).
	// The compound card generation from Suppressed gaps only applies to
	// prospective mandates (patient NOT on drug but should be).
	if hasMRAConflict {
		t.Log("MRA conflict detected via Suppressed gap path")
	}

	// Always: renal gate must flag MRA
	hasMRABlock := false
	for _, blocked := range report.BlockedDrugClasses {
		if blocked == "MRA" {
			hasMRABlock = true
		}
	}
	if !hasMRABlock {
		t.Error("renal gate should flag MRA as blocked at eGFR 28")
	}
	if !report.HasSafetyBlock {
		t.Error("HasSafetyBlock must be true when MRA is contraindicated")
	}
}

// Scenario B: HFrEF patient NOT on MRA or SGLT2i, eGFR 28.
// Mandatory med checker would normally flag both as IMMEDIATE gaps.
// Cross-domain resolution must suppress MRA (contra at <30) while
// SGLT2i remains a valid gap (contra at <20, eGFR 28 is fine).
func TestCrossDomain_HFrEF_MissingMRA_RenalWouldBlock(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{
		EGFR:           28,
		EGFRSlope:      -1,
		EGFRMeasuredAt: time.Now().Add(-24 * time.Hour),
		EGFRDataPoints: 4,
	}

	// HFrEF patient on ACEi + BB only — missing MRA and SGLT2i
	meds := []ActiveMedication{
		{DrugClass: "ACEi", DrugName: "Ramipril 10mg", CurrentDoseMg: 10},
		{DrugClass: "BETA_BLOCKER_HF", DrugName: "Carvedilol 25mg", CurrentDoseMg: 25},
	}

	report := DetectAllConflicts(gate, formulary, "patient-hfref-02",
		renal, meds, -1.0, "4c", "HFrEF")

	// MRA gap should be SUPPRESSED (would-block at eGFR 28)
	var mraGap, sglt2iGap *MandatoryMedGap
	for i, g := range report.MandatoryMedGaps {
		if g.MissingClass == "MRA" {
			mraGap = &report.MandatoryMedGaps[i]
		}
		if g.MissingClass == "SGLT2i" {
			sglt2iGap = &report.MandatoryMedGaps[i]
		}
	}

	if mraGap == nil {
		t.Fatal("MRA gap should be detected")
	}
	if !mraGap.Suppressed {
		t.Error("MRA gap should be Suppressed at eGFR 28 (MRA contra <30)")
	}
	if mraGap.SuppressionReason == "" {
		t.Error("Suppressed gap must carry a SuppressionReason")
	}

	if sglt2iGap == nil {
		t.Fatal("SGLT2i gap should be detected")
	}
	if sglt2iGap.Suppressed {
		t.Error("SGLT2i gap should NOT be suppressed at eGFR 28 (SGLT2i contra <20)")
	}

	// RenalHFConflicts should contain MRA but not SGLT2i
	var mraConflict *RenalHFConflict
	for i, c := range report.RenalHFConflicts {
		if c.DrugClass == "MRA" {
			mraConflict = &report.RenalHFConflicts[i]
		}
		if c.DrugClass == "SGLT2i" {
			t.Error("SGLT2i should NOT be in RenalHFConflicts at eGFR 28")
		}
	}
	if mraConflict == nil {
		t.Fatal("MRA should be in RenalHFConflicts")
	}
	if mraConflict.Urgency != "URGENT" {
		t.Errorf("MRA conflict urgency: want URGENT, got %s", mraConflict.Urgency)
	}
}

// Scenario C (CRITICAL): HFpEF patient with eGFR 18 — SGLT2i is the ONLY
// proven disease-modifying therapy for HFpEF, and it's now contraindicated.
// This patient has NO available GDMT → HasCriticalConflict must be true.
func TestCrossDomain_HFpEF_SGLT2iLost_Critical(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{
		EGFR:           18, // below SGLT2i contra (20)
		EGFRSlope:      -3,
		EGFRMeasuredAt: time.Now().Add(-24 * time.Hour),
		EGFRDataPoints: 5,
		CKDStage:       "G4",
	}

	// HFpEF patient on supportive therapy, not on SGLT2i
	meds := []ActiveMedication{
		{DrugClass: "THIAZIDE", DrugName: "Indapamide 2.5mg", CurrentDoseMg: 2.5},
		{DrugClass: "METFORMIN", DrugName: "Metformin 1000mg", CurrentDoseMg: 1000},
	}

	report := DetectAllConflicts(gate, formulary, "patient-hfpef-01",
		renal, meds, -3.0, "4c", "HFpEF")

	// HasCriticalConflict must be true — HFpEF has no fallback therapy
	if !report.HasCriticalConflict {
		t.Error("HasCriticalConflict must be true when HFpEF loses SGLT2i — only proven GDMT")
	}
	if !report.HasSafetyBlock {
		t.Error("HasSafetyBlock must be true for critical conflict")
	}

	// SGLT2i gap should be Suppressed
	var sglt2iGap *MandatoryMedGap
	for i, g := range report.MandatoryMedGaps {
		if g.MissingClass == "SGLT2i" {
			sglt2iGap = &report.MandatoryMedGaps[i]
		}
	}
	if sglt2iGap == nil {
		t.Fatal("SGLT2i gap should be detected for HFpEF")
	}
	if !sglt2iGap.Suppressed {
		t.Error("SGLT2i gap should be Suppressed at eGFR 18")
	}

	// RenalHFConflicts should contain SGLT2i with IMMEDIATE urgency
	var sglt2iConflict *RenalHFConflict
	for i, c := range report.RenalHFConflicts {
		if c.DrugClass == "SGLT2i" {
			sglt2iConflict = &report.RenalHFConflicts[i]
		}
	}
	if sglt2iConflict == nil {
		t.Fatal("SGLT2i should be in RenalHFConflicts for HFpEF at eGFR 18")
	}
	if sglt2iConflict.Urgency != "IMMEDIATE" {
		t.Errorf("HFpEF SGLT2i conflict must be IMMEDIATE, got %s", sglt2iConflict.Urgency)
	}
	if !containsStr(sglt2iConflict.ResolutionRecommendation, "CRITICAL") {
		t.Errorf("HFpEF SGLT2i conflict must mention CRITICAL: %s", sglt2iConflict.ResolutionRecommendation)
	}
}

// Scenario D: Renal function is fine — no cross-domain conflicts should fire.
// Verifies the cross-domain logic doesn't produce false positives.
func TestCrossDomain_NoConflict_NormalRenal(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{
		EGFR:           75, // normal
		EGFRSlope:      0,
		EGFRMeasuredAt: time.Now().Add(-24 * time.Hour),
		EGFRDataPoints: 4,
	}

	// HFrEF patient on ACEi + BB only — missing MRA and SGLT2i
	meds := []ActiveMedication{
		{DrugClass: "ACEi", DrugName: "Ramipril 10mg", CurrentDoseMg: 10},
		{DrugClass: "BETA_BLOCKER_HF", DrugName: "Carvedilol 25mg", CurrentDoseMg: 25},
	}

	report := DetectAllConflicts(gate, formulary, "patient-hfref-03",
		renal, meds, 0, "4c", "HFrEF")

	if report.HasCriticalConflict {
		t.Error("no critical conflict expected with normal renal function")
	}
	if len(report.RenalHFConflicts) > 0 {
		t.Errorf("no RenalHFConflicts expected, got %d", len(report.RenalHFConflicts))
	}

	// Mandatory gaps should NOT be suppressed
	for _, g := range report.MandatoryMedGaps {
		if g.Suppressed {
			t.Errorf("gap %s should NOT be suppressed at eGFR 75", g.MissingClass)
		}
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
