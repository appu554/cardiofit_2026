package services

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func timePtr(t time.Time) *time.Time { return &t }

func float64Ptr(v float64) *float64 { return &v }

func setupTestGate(t *testing.T) *RenalDoseGate {
	t.Helper()
	f, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary failed: %v", err)
	}
	return NewRenalDoseGate(f)
}

func renal(egfr float64, daysAgo int, potassium *float64) models.RenalStatus {
	now := time.Now()
	rs := models.RenalStatus{
		EGFR:           egfr,
		EGFRMeasuredAt: now.AddDate(0, 0, -daysAgo),
		EGFRDataPoints: 3,
	}
	if potassium != nil {
		rs.Potassium = potassium
		t := now.AddDate(0, 0, -daysAgo)
		rs.PotassiumMeasuredAt = &t
	}
	return rs
}

// ---------------------------------------------------------------------------
// TestGate_Metformin_EGFR25_Contraindicated
// ---------------------------------------------------------------------------

func TestGate_Metformin_EGFR25_Contraindicated(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 2000}
	rs := renal(25.0, 7, nil)

	result := gate.Evaluate(med, rs)

	if result.Verdict != models.VerdictContraindicated {
		t.Errorf("verdict: want CONTRAINDICATED, got %s", result.Verdict)
	}
	want := "eGFR 25.0 below 30.0"
	if !strings.Contains(result.Reason, want) {
		t.Errorf("reason should contain %q, got %q", want, result.Reason)
	}
}

// ---------------------------------------------------------------------------
// TestGate_Metformin_EGFR40_DoseReduce
// ---------------------------------------------------------------------------

func TestGate_Metformin_EGFR40_DoseReduce(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 2000}
	rs := renal(40.0, 7, nil)

	result := gate.Evaluate(med, rs)

	if result.Verdict != models.VerdictDoseReduce {
		t.Errorf("verdict: want DOSE_REDUCE, got %s", result.Verdict)
	}
	if result.MaxSafeDoseMg == nil || *result.MaxSafeDoseMg != 1000.0 {
		t.Errorf("MaxSafeDoseMg: want 1000.0, got %v", result.MaxSafeDoseMg)
	}
}

// ---------------------------------------------------------------------------
// TestGate_Metformin_EGFR55_MonitorEscalate
// ---------------------------------------------------------------------------

func TestGate_Metformin_EGFR55_MonitorEscalate(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 2000}
	rs := renal(55.0, 7, nil)

	result := gate.Evaluate(med, rs)

	if result.Verdict != models.VerdictMonitorEscalate {
		t.Errorf("verdict: want MONITOR_ESCALATE, got %s", result.Verdict)
	}
}

// ---------------------------------------------------------------------------
// TestGate_Metformin_EGFR75_Cleared
// ---------------------------------------------------------------------------

func TestGate_Metformin_EGFR75_Cleared(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 2000}
	rs := renal(75.0, 7, nil)

	result := gate.Evaluate(med, rs)

	if result.Verdict != models.VerdictCleared {
		t.Errorf("verdict: want CLEARED, got %s", result.Verdict)
	}
}

// ---------------------------------------------------------------------------
// TestGate_SGLT2i_EGFR18_Contraindicated
// ---------------------------------------------------------------------------

func TestGate_SGLT2i_EGFR18_Contraindicated(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "SGLT2i", DrugName: "Empagliflozin 10mg", CurrentDoseMg: 10}
	rs := renal(18.0, 7, nil)

	result := gate.Evaluate(med, rs)

	if result.Verdict != models.VerdictContraindicated {
		t.Errorf("verdict: want CONTRAINDICATED, got %s", result.Verdict)
	}
}

// ---------------------------------------------------------------------------
// TestGate_SGLT2i_EGFR35_Cleared
// ---------------------------------------------------------------------------

func TestGate_SGLT2i_EGFR35_Cleared(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "SGLT2i", DrugName: "Empagliflozin 10mg", CurrentDoseMg: 10}
	rs := renal(35.0, 7, nil)

	result := gate.Evaluate(med, rs)

	// SGLT2i: contra <20, no dose_reduce (0), efficacy_cliff <30 but 35 > 30,
	// monitor_escalate <45 so 35 < 45 → MONITOR_ESCALATE
	// Actually 35 is above efficacy cliff (30) and above contra (20), but below monitor_escalate (45)
	// With no dose_reduce zone, it should be MONITOR_ESCALATE
	if result.Verdict != models.VerdictMonitorEscalate {
		t.Errorf("verdict: want MONITOR_ESCALATE, got %s", result.Verdict)
	}
}

// ---------------------------------------------------------------------------
// TestGate_MRA_HighPotassium_Contraindicated
// ---------------------------------------------------------------------------

func TestGate_MRA_HighPotassium_Contraindicated(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "MRA", DrugName: "Spironolactone 25mg", CurrentDoseMg: 25}
	k := 5.3
	rs := renal(50.0, 7, &k)

	result := gate.Evaluate(med, rs)

	if result.Verdict != models.VerdictContraindicated {
		t.Errorf("verdict: want CONTRAINDICATED, got %s", result.Verdict)
	}
	if !strings.Contains(result.Reason, "potassium") && !strings.Contains(result.Reason, "K+") {
		t.Errorf("reason should mention potassium, got %q", result.Reason)
	}
}

// ---------------------------------------------------------------------------
// TestGate_ACEi_NoPotassiumData_MonitorEscalate
// ---------------------------------------------------------------------------

func TestGate_ACEi_NoPotassiumData_MonitorEscalate(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "ACEi", DrugName: "Ramipril 5mg", CurrentDoseMg: 5}
	rs := renal(40.0, 7, nil) // nil potassium

	result := gate.Evaluate(med, rs)

	// ACEi requires potassium check, eGFR 40 < 45 (monitor zone), nil K+ → MONITOR_ESCALATE
	if result.Verdict != models.VerdictMonitorEscalate {
		t.Errorf("verdict: want MONITOR_ESCALATE, got %s", result.Verdict)
	}
}

// ---------------------------------------------------------------------------
// TestGate_Thiazide_EGFR25_EfficacyCliff
// ---------------------------------------------------------------------------

func TestGate_Thiazide_EGFR25_EfficacyCliff(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "THIAZIDE", DrugName: "Hydrochlorothiazide 25mg", CurrentDoseMg: 25}
	rs := renal(25.0, 7, nil)

	result := gate.Evaluate(med, rs)

	// THIAZIDE: contra <15, efficacy_cliff <30, eGFR 25 < 30 → DOSE_REDUCE with substitute
	if result.Verdict != models.VerdictDoseReduce {
		t.Errorf("verdict: want DOSE_REDUCE (efficacy cliff), got %s", result.Verdict)
	}
	if result.SubstituteClass != "LOOP_DIURETIC" {
		t.Errorf("substitute: want LOOP_DIURETIC, got %q", result.SubstituteClass)
	}
}

// ---------------------------------------------------------------------------
// TestGate_StaleEGFR_InsufficientData
// ---------------------------------------------------------------------------

func TestGate_StaleEGFR_InsufficientData(t *testing.T) {
	gate := setupTestGate(t)
	med := ActiveMedication{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 2000}
	rs := renal(50.0, 200, nil) // 200 days old > 180 critical

	result := gate.Evaluate(med, rs)

	if result.Verdict != models.VerdictInsufficientData {
		t.Errorf("verdict: want INSUFFICIENT_DATA, got %s", result.Verdict)
	}
}

// ---------------------------------------------------------------------------
// TestEvaluatePatient_MultiMed
// ---------------------------------------------------------------------------

func TestEvaluatePatient_MultiMed(t *testing.T) {
	gate := setupTestGate(t)
	rs := renal(28.0, 7, nil)

	meds := []ActiveMedication{
		{DrugClass: "METFORMIN", DrugName: "Metformin 500mg", CurrentDoseMg: 2000},
		{DrugClass: "SGLT2i", DrugName: "Empagliflozin 10mg", CurrentDoseMg: 10},
		{DrugClass: "ACEi", DrugName: "Ramipril 5mg", CurrentDoseMg: 5},
		{DrugClass: "THIAZIDE", DrugName: "HCTZ 25mg", CurrentDoseMg: 25},
		{DrugClass: "SULFONYLUREA", DrugName: "Gliclazide MR 60mg", CurrentDoseMg: 60},
	}

	report := gate.EvaluatePatient("patient-001", rs, meds)

	if len(report.MedicationResults) != 5 {
		t.Fatalf("expected 5 medication results, got %d", len(report.MedicationResults))
	}

	// Build verdict map for easier assertion
	verdicts := make(map[string]models.GatingVerdict)
	for _, r := range report.MedicationResults {
		verdicts[r.DrugClass] = r.Verdict
	}

	expected := map[string]models.GatingVerdict{
		"METFORMIN":    models.VerdictContraindicated, // 28 < 30
		"SGLT2i":       models.VerdictMonitorEscalate, // 28 > 20 (no contra), but < 45 monitor zone; efficacy cliff 30 > 28 → DOSE_REDUCE actually
		"ACEi":         models.VerdictMonitorEscalate, // 28 < 30 (dose_reduce zone) but needs K+ check, nil K+ → MONITOR_ESCALATE
		"THIAZIDE":     models.VerdictDoseReduce,      // 28 < 30 efficacy cliff → DOSE_REDUCE
		"SULFONYLUREA": models.VerdictContraindicated,  // 28 < 30
	}

	// SGLT2i at eGFR 28: contra <20 (no), efficacy_cliff <30 and 28 < 30 → DOSE_REDUCE with substitute
	expected["SGLT2i"] = models.VerdictDoseReduce

	for drug, wantVerdict := range expected {
		got, ok := verdicts[drug]
		if !ok {
			t.Errorf("%s: no result found", drug)
			continue
		}
		if got != wantVerdict {
			t.Errorf("%s: want %s, got %s", drug, wantVerdict, got)
		}
	}

	if !report.HasContraindicated {
		t.Error("HasContraindicated should be true")
	}
	if !report.HasDoseReduce {
		t.Error("HasDoseReduce should be true")
	}
	if report.OverallUrgency != "IMMEDIATE" {
		t.Errorf("OverallUrgency: want IMMEDIATE, got %s", report.OverallUrgency)
	}

	_ = fmt.Sprintf("report: %+v", report) // ensure no unused import
}
