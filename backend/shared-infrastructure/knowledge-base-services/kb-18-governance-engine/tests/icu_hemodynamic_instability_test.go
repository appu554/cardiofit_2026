// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ICU HEMODYNAMIC INSTABILITY scenarios.
//
// Clinical Truth: In hemodynamic crisis, perfusion trumps everything except bleed.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ICU HEMODYNAMIC INSTABILITY SCENARIOS
// These tests prove that life-preserving hemodynamic support is prioritized
// while maintaining essential safety guardrails.
// =============================================================================

// TestICU_Hemodynamic_VasopressorAllowedDespiteChronic tests that vasopressors
// are allowed for hemodynamic support even with chronic conditions.
//
// Scenario: Cardiogenic shock + Chronic renal disease
// Expected: Vasopressor ALLOWED - organ perfusion is priority
func TestICU_Hemodynamic_VasopressorAllowedDespiteChronic(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Cardiogenic shock with CKD
	patient := &types.PatientContext{
		PatientID:  "PT-ICU-HEMO-001",
		Age:        68,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "R57.0", CodeSystem: "ICD10", Description: "Cardiogenic shock"},
			{Code: "N18.4", CodeSystem: "ICD10", Description: "CKD Stage 4"},
			{Code: "I50.9", CodeSystem: "ICD10", Description: "Heart failure"},
		},
		RenalFunction: &types.RenalFunction{
			EGFR:       22.0,
			Creatinine: 3.8,
			CKDStage:   "CKD_4",
			OnDialysis: false,
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "HEMODYNAMIC_SUPPORT", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  75,
			DiastolicBP: 45,
			HeartRate:   110,
			SpO2:        91.0,
		},
		RecentLabs: []types.LabResult{
			{Code: "2160-0", CodeSystem: "LOINC", Name: "Creatinine", Value: 3.8, Unit: "mg/dL"},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 4.8, Unit: "mmol/L"},
		},
	}

	// KB-19 recommends: Norepinephrine for hemodynamic support
	vasopressorRec := SimulatedRecommendation{
		Target:             "Norepinephrine",
		TargetRxNorm:       "7512",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.15,
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICU_HEMODYNAMIC_SUPPORT",
		Rationale:          "Cardiogenic shock - maintain organ perfusion (MAP goal ≥65 mmHg)",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vasopressorRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Vasopressor must be allowed - organ perfusion is critical
	if result.IsBlocked() {
		t.Errorf("❌ ICU HEMODYNAMIC FAILURE: Vasopressor BLOCKED in cardiogenic shock")
		t.Errorf("   MAP: 55 mmHg, Lactate: 4.8 mmol/L")
		t.Errorf("   Expected: APPROVED (organ perfusion priority)")
		t.Errorf("   Got: %s", result.FinalOutcome)
	}

	t.Logf("✅ ICU HEMODYNAMIC: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Context: Cardiogenic shock + CKD4")
	t.Logf("   Clinical Truth: Perfusion support trumps chronic disease concerns")
}

// TestICU_Hemodynamic_InotropeAllowed tests that inotropic support is allowed
// for cardiac output optimization.
//
// Scenario: Low cardiac output syndrome post-cardiac surgery
// Expected: Dobutamine ALLOWED for cardiac output support
func TestICU_Hemodynamic_InotropeAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-HEMO-002",
		Age:        72,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I97.110", CodeSystem: "ICD10", Description: "Postprocedural cardiac functional disturbance"},
			{Code: "I50.20", CodeSystem: "ICD10", Description: "Unspecified systolic heart failure"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "POST_CARDIAC_SURGERY", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  88,
			DiastolicBP: 58,
			HeartRate:   95,
			SpO2:        94.0,
		},
	}

	// KB-19 recommends: Dobutamine for cardiac output support
	inotropeRec := SimulatedRecommendation{
		Target:             "Dobutamine",
		TargetRxNorm:       "3616",
		DrugClass:          "INOTROPE",
		RecommendedDose:    5.0,
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "ICU_CARDIAC_OUTPUT",
		Rationale:          "Low cardiac output syndrome - optimize cardiac performance",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, inotropeRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Inotrope must be allowed for cardiac support
	if result.IsBlocked() {
		t.Errorf("❌ ICU CARDIAC FAILURE: Inotrope BLOCKED in low output syndrome")
	}

	t.Logf("✅ ICU INOTROPE: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Hemodynamic_FluidResuscitationDespiteHF tests that fluid resuscitation
// is allowed in hypovolemic shock even with heart failure history.
//
// Scenario: Hemorrhagic shock + chronic HF
// Expected: Fluids ALLOWED (life over chronic concerns)
func TestICU_Hemodynamic_FluidResuscitationDespiteHF(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-HEMO-003",
		Age:        65,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "R57.1", CodeSystem: "ICD10", Description: "Hypovolemic shock"},
			{Code: "I50.22", CodeSystem: "ICD10", Description: "Chronic systolic HF"},
			{Code: "K92.2", CodeSystem: "ICD10", Description: "GI hemorrhage"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "ACTIVE_HEMORRHAGE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  78,
			DiastolicBP: 50,
			HeartRate:   125,
			SpO2:        95.0,
		},
		RecentLabs: []types.LabResult{
			{Code: "718-7", CodeSystem: "LOINC", Name: "Hemoglobin", Value: 6.8, Unit: "g/dL"},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 5.5, Unit: "mmol/L"},
		},
	}

	// KB-19 recommends: Crystalloid resuscitation
	fluidRec := FluidRecommendation(2000) // 2L crystalloid

	result, err := ctx.ExecuteE2EFlow(patient, fluidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Fluids must be allowed despite HF history
	if result.IsBlocked() {
		t.Errorf("❌ ICU HEMORRHAGE FAILURE: Fluids BLOCKED in hypovolemic shock")
		t.Errorf("   MAP: 59 mmHg, Hgb: 6.8 g/dL, Lactate: 5.5 mmol/L")
		t.Errorf("   HF history does NOT override active hemorrhagic shock")
	}

	t.Logf("✅ ICU HEMORRHAGE RESUSCITATION: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Hemodynamic_VasopressinSecondLine tests that vasopressin is allowed
// as second-line vasopressor in refractory shock.
//
// Scenario: Septic shock not responding to norepinephrine
// Expected: Vasopressin ALLOWED as adjunct
func TestICU_Hemodynamic_VasopressinSecondLine(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := SepticShockPatient()
	patient.RecentLabs = append(patient.RecentLabs, types.LabResult{
		Code:       "14627-4",
		CodeSystem: "LOINC",
		Name:       "Lactate",
		Value:      6.2, // Rising despite treatment
		Unit:       "mmol/L",
	})

	// KB-19 recommends: Vasopressin as second-line
	vasopressinRec := SimulatedRecommendation{
		Target:             "Vasopressin",
		TargetRxNorm:       "11149",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.03,
		DoseUnit:           "units/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "ICU_REFRACTORY_SHOCK",
		Rationale:          "SSC 2021: Add vasopressin if MAP not at target with norepinephrine",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vasopressinRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Second-line vasopressor must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ ICU REFRACTORY SHOCK FAILURE: Second-line vasopressor BLOCKED")
	}

	t.Logf("✅ ICU VASOPRESSIN: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Hemodynamic_AnticoagBlocked_ActiveBleeding tests that anticoagulation
// is blocked even in hemodynamically unstable patients with active bleeding.
//
// Scenario: Shock + Active GI bleed + AFib (wants anticoag)
// Expected: Anticoagulation BLOCKED - bleeding risk overrides
func TestICU_Hemodynamic_AnticoagBlocked_ActiveBleeding(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-HEMO-004",
		Age:        70,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "R57.1", CodeSystem: "ICD10", Description: "Hypovolemic shock"},
			{Code: "I48.0", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
			{Code: "K92.0", CodeSystem: "ICD10", Description: "Hematemesis"}, // Active bleeding
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "ACTIVE_HEMORRHAGE", Status: "ACTIVE"},
			{RegistryCode: "ANTICOAGULATION", Status: "SUSPENDED"},
		},
		RecentLabs: []types.LabResult{
			{Code: "718-7", CodeSystem: "LOINC", Name: "Hemoglobin", Value: 7.2, Unit: "g/dL"},
		},
	}

	// Inappropriately recommended anticoagulation
	anticoagRec := AnticoagulationRecommendation("Heparin", "5224", 80.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Anticoag must be BLOCKED during active bleeding
	if result.FinalAllowed {
		t.Errorf("❌ ICU BLEEDING SAFETY FAILURE: Anticoag ALLOWED during active hemorrhage")
		t.Errorf("   Active hematemesis with Hgb 7.2 g/dL")
		t.Errorf("   AFib stroke risk does NOT override catastrophic bleed risk")
	}

	t.Logf("✅ ICU BLEEDING SAFETY: outcome=%s, blocked=%v", result.FinalOutcome, !result.FinalAllowed)
}

// =============================================================================
// ICU HEMODYNAMIC INVARIANT TESTS
// =============================================================================

// TestICU_Hemodynamic_Invariant_PerfusionPriority tests that hemodynamic support
// is consistently prioritized in shock states.
func TestICU_Hemodynamic_Invariant_PerfusionPriority(t *testing.T) {
	ctx := NewE2ETestContext()

	shockScenarios := []struct {
		name    string
		patient *types.PatientContext
		rec     SimulatedRecommendation
	}{
		{
			name:    "Vasopressor in septic shock",
			patient: SepticShockPatient(),
			rec: SimulatedRecommendation{
				Target:             "Norepinephrine",
				TargetRxNorm:       "7512",
				DrugClass:          "VASOPRESSOR",
				RecommendedDose:    0.1,
				DoseUnit:           "mcg/kg/min",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "ICU_SHOCK",
				Rationale:          "MAP support",
				Urgency:            "STAT",
			},
		},
		{
			name:    "Fluids in hypovolemic shock",
			patient: SepticShockPatient(),
			rec:     FluidRecommendation(2000),
		},
		{
			name:    "Vasopressor in cardiogenic shock",
			patient: ICUPatientWithMultiOrganFailure(),
			rec: SimulatedRecommendation{
				Target:             "Norepinephrine",
				TargetRxNorm:       "7512",
				DrugClass:          "VASOPRESSOR",
				RecommendedDose:    0.2,
				DoseUnit:           "mcg/kg/min",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "ICU_SHOCK",
				Rationale:          "Hemodynamic support",
				Urgency:            "STAT",
			},
		},
	}

	blockedCount := 0
	for _, scenario := range shockScenarios {
		result, err := ctx.ExecuteE2EFlow(scenario.patient, scenario.rec)
		if err != nil {
			t.Errorf("Scenario '%s' failed: %v", scenario.name, err)
			continue
		}

		if result.IsBlocked() {
			blockedCount++
			t.Errorf("❌ PERFUSION PRIORITY VIOLATION: %s was BLOCKED", scenario.name)
		}
	}

	if blockedCount > 0 {
		t.Errorf("❌ HEMODYNAMIC INVARIANT FAILURE: %d/%d perfusion-critical interventions blocked",
			blockedCount, len(shockScenarios))
	} else {
		t.Logf("✅ HEMODYNAMIC INVARIANT VERIFIED: All %d perfusion-critical interventions allowed",
			len(shockScenarios))
	}
}
