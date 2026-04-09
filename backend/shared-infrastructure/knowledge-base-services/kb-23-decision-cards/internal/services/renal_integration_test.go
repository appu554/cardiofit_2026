package services

import (
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// TestFourPillar_RenalContraindication_OverridesDualDomain
// ---------------------------------------------------------------------------

// Verifies that a renal contraindication (eGFR 25, METFORMIN contra < 30)
// forces medication pillar to URGENT_GAP and urgency to IMMEDIATE, even
// when dual-domain state is "GC-HC" (both controlled).
func TestFourPillar_RenalContraindication_OverridesDualDomain(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{
		EGFR:           25,
		EGFRSlope:      -3,
		EGFRMeasuredAt: time.Now().Add(-24 * time.Hour), // fresh
		EGFRDataPoints: 4,
		CKDStage:       "G4",
	}

	meds := []ActiveMedication{
		{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 500},
	}

	gatingReport := gate.EvaluatePatient("patient-renal-01", renal, meds)
	if !gatingReport.HasContraindicated {
		t.Fatal("expected HasContraindicated=true for eGFR 25 with METFORMIN (contra < 30)")
	}

	// Four-pillar evaluation with renal gating
	input := FourPillarInput{
		PatientID:       "patient-renal-01",
		DualDomainState: "GC-HC", // both controlled
		Medication: MedicationPillarInput{
			OnGuidelineMeds: true,
			AdherencePct:    95,
		},
		Monitoring: MonitoringPillarInput{},
		Lifestyle:  LifestylePillarInput{AdherencePct: 80},
		Education:  EducationPillarInput{Complete: true},
		RenalGating: &gatingReport,
	}

	pillarResult := EvaluateFourPillars(input)

	// Medication pillar must be URGENT_GAP due to renal contraindication
	var medPillar *PillarResult
	for i, p := range pillarResult.Pillars {
		if p.Pillar == "MEDICATION" {
			medPillar = &pillarResult.Pillars[i]
			break
		}
	}
	if medPillar == nil {
		t.Fatal("MEDICATION pillar not found in result")
	}
	if medPillar.Status != PillarUrgentGap {
		t.Errorf("MEDICATION pillar status: want URGENT_GAP, got %s", medPillar.Status)
	}

	// Urgency must be IMMEDIATE despite "GC-HC" dual-domain
	urgency := CalculateDualDomainUrgency("GC-HC", pillarResult, &gatingReport)
	if urgency != UrgencyImmediate {
		t.Errorf("urgency: want IMMEDIATE, got %s", urgency)
	}
}

// ---------------------------------------------------------------------------
// TestBlocksUnsafeRecommendation
// ---------------------------------------------------------------------------

// Verifies BlockRecommendation prevents SGLT2i (contra < 20) and METFORMIN
// (contra < 30) at eGFR 18.
func TestBlocksUnsafeRecommendation(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{
		EGFR:           18,
		EGFRSlope:      -5,
		EGFRMeasuredAt: time.Now().Add(-24 * time.Hour),
		EGFRDataPoints: 5,
		CKDStage:       "G4",
	}

	// SGLT2i should be blocked (18 < 20)
	blocked, reason := gate.BlockRecommendation("SGLT2i", renal)
	if !blocked {
		t.Error("SGLT2i should be blocked at eGFR 18")
	}
	if reason == "" {
		t.Error("block reason should not be empty for SGLT2i")
	}
	t.Logf("SGLT2i block reason: %s", reason)

	// METFORMIN should be blocked (18 < 30)
	blocked, reason = gate.BlockRecommendation("METFORMIN", renal)
	if !blocked {
		t.Error("METFORMIN should be blocked at eGFR 18")
	}
	if reason == "" {
		t.Error("block reason should not be empty for METFORMIN")
	}
	t.Logf("METFORMIN block reason: %s", reason)
}

// ---------------------------------------------------------------------------
// TestEnrichedConflict_CombinesAllSafety
// ---------------------------------------------------------------------------

// Verifies DetectAllConflicts at eGFR 28 with 3 meds and steep slope -20:
//   - HasSafetyBlock true (METFORMIN contra at 28 < 30)
//   - METFORMIN in blocked classes
//   - Anticipatory alerts not empty (steep decline crosses SGLT2i contra at 20)
func TestEnrichedConflict_CombinesAllSafety(t *testing.T) {
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	gate := NewRenalDoseGate(formulary)

	renal := models.RenalStatus{
		EGFR:           28,
		EGFRSlope:      -20,
		EGFRMeasuredAt: time.Now().Add(-24 * time.Hour),
		EGFRDataPoints: 6,
		CKDStage:       "G4",
	}

	meds := []ActiveMedication{
		{DrugClass: "METFORMIN", DrugName: "Metformin 1000mg", CurrentDoseMg: 1000},
		{DrugClass: "SGLT2i", DrugName: "Empagliflozin 10mg", CurrentDoseMg: 10},
		{DrugClass: "ACEi", DrugName: "Ramipril 5mg", CurrentDoseMg: 5},
	}

	report := DetectAllConflicts(gate, formulary, "patient-renal-02", renal, meds, -20)

	// HasSafetyBlock must be true (METFORMIN contraindicated at eGFR 28 < 30)
	if !report.HasSafetyBlock {
		t.Error("HasSafetyBlock: want true")
	}

	// METFORMIN must be in blocked classes
	found := false
	for _, dc := range report.BlockedDrugClasses {
		if dc == "METFORMIN" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("BlockedDrugClasses should contain METFORMIN, got %v", report.BlockedDrugClasses)
	}

	// Anticipatory alerts should not be empty (slope -6 crosses thresholds)
	if len(report.AnticipatoryAlerts) == 0 {
		t.Error("AnticipatoryAlerts: want non-empty (eGFR 28 with slope -6 should trigger alerts)")
	}
	t.Logf("anticipatory alerts: %d", len(report.AnticipatoryAlerts))
	for _, a := range report.AnticipatoryAlerts {
		t.Logf("  %s → %s in %.1f months", a.DrugClass, a.ThresholdType, a.MonthsToThreshold)
	}

	// Renal gating report should be populated
	if report.RenalGating == nil {
		t.Fatal("RenalGating should not be nil")
	}
	if report.RenalGating.PatientID != "patient-renal-02" {
		t.Errorf("RenalGating.PatientID: want patient-renal-02, got %s", report.RenalGating.PatientID)
	}

	// StaleEGFR should be populated (not stale since measured yesterday)
	if report.StaleEGFR == nil {
		t.Fatal("StaleEGFR should not be nil")
	}
	if report.StaleEGFR.IsStale {
		t.Error("StaleEGFR.IsStale: want false (measured 1 day ago)")
	}
}
