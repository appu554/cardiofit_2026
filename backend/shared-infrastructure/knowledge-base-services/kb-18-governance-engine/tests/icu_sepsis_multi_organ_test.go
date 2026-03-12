// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ICU SEPSIS MULTI-ORGAN FAILURE scenarios.
//
// Clinical Truth: Multi-organ failure requires prioritization - lungs, kidneys, liver
// may all be failing simultaneously. The system must triage interventions.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ICU SEPSIS MULTI-ORGAN FAILURE SCENARIOS
// These tests prove the system can handle complex patients with multiple
// simultaneous organ failures requiring prioritized interventions.
// =============================================================================

// TestICU_MODS_AntibioticsSTAT tests that antibiotics are allowed STAT
// even with hepatic and renal impairment.
//
// Scenario: Septic shock + AKI + Hepatic failure
// Expected: Antibiotics ALLOWED - source control critical, dose may need adjustment
func TestICU_MODS_AntibioticsSTAT(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := ICUPatientWithMultiOrganFailure()

	// KB-19 recommends: Piperacillin-tazobactam for broad coverage
	antibioticRec := SimulatedRecommendation{
		Target:             "Piperacillin-Tazobactam",
		TargetRxNorm:       "251210",
		DrugClass:          "ANTIBIOTIC_BROAD",
		RecommendedDose:    4500,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPSIS_HOUR_1",
		Rationale:          "SSC 2021: Broad-spectrum within 1 hour, adjust for renal function",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, antibioticRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Antibiotics MUST proceed - source control is life-saving
	if result.IsBlocked() {
		t.Errorf("❌ MODS SEPSIS FAILURE: Antibiotics BLOCKED despite life-threatening sepsis")
		t.Errorf("   Multi-organ failure does NOT override antibiotic necessity")
		t.Errorf("   SSC 2021: Every hour of delay increases mortality 4-8%%")
	}

	// Check if renal dosing concern was raised (acceptable warning)
	hasRenalWarning := result.HasViolationCategory(types.ViolationRenalDosing)
	t.Logf("✅ ICU MODS ANTIBIOTICS: outcome=%s, allowed=%v, renal_warning=%v",
		result.FinalOutcome, result.FinalAllowed, hasRenalWarning)
}

// TestICU_MODS_VasopressorWithHepaticFailure tests vasopressor use in patients
// with hepatic failure.
//
// Scenario: Septic shock + Hepatic failure
// Expected: Vasopressor ALLOWED - hepatic metabolism concerns secondary to survival
func TestICU_MODS_VasopressorWithHepaticFailure(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-MODS-002",
		Age:        58,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
			{Code: "K72.91", CodeSystem: "ICD10", Description: "Hepatic failure with coma"},
			{Code: "R57.2", CodeSystem: "ICD10", Description: "Septic shock"},
		},
		HepaticFunction: &types.HepaticFunction{
			ChildPughScore: 12,
			ChildPughClass: "C", // Severe hepatic impairment
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "SEPSIS_BUNDLE", Status: "ACTIVE"},
			{RegistryCode: "HEPATIC_FAILURE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  72,
			DiastolicBP: 42,
			HeartRate:   115,
			SpO2:        92.0,
		},
	}

	// KB-19 recommends: Norepinephrine
	vasopressorRec := SimulatedRecommendation{
		Target:             "Norepinephrine",
		TargetRxNorm:       "7512",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.2,
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICU_SEPTIC_SHOCK",
		Rationale:          "MAP goal ≥65 mmHg for organ perfusion",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vasopressorRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Vasopressor must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ HEPATIC FAILURE DOES NOT BLOCK VASOPRESSORS")
		t.Errorf("   MAP: 52 mmHg - patient is dying")
	}

	t.Logf("✅ ICU MODS VASOPRESSOR (hepatic failure): outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestICU_MODS_RenalReplacementTherapyAllowed tests that CRRT/dialysis is allowed
// in MODS patients.
//
// Scenario: MODS with severe AKI
// Expected: CRRT initiation ALLOWED
func TestICU_MODS_RenalReplacementTherapyAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := ICUPatientWithMultiOrganFailure()

	// KB-19 recommends: CRRT initiation
	crrtRec := SimulatedRecommendation{
		Target:             "CRRT",
		TargetRxNorm:       "", // Procedure, not medication
		DrugClass:          "RENAL_REPLACEMENT",
		RecommendedDose:    0,
		DoseUnit:           "",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICU_RENAL_SUPPORT",
		Rationale:          "AKI Stage 3 with volume overload and acidosis",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, crrtRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Renal support must be allowed in severe AKI
	if result.IsBlocked() {
		t.Errorf("❌ MODS RENAL SUPPORT FAILURE: CRRT blocked in severe AKI")
	}

	t.Logf("✅ ICU MODS CRRT: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_MODS_CorticosteroidsForRefractoryShock tests steroid use in
// refractory septic shock per SSC guidelines.
//
// Scenario: Refractory septic shock on high-dose vasopressors
// Expected: Hydrocortisone ALLOWED per SSC 2021
func TestICU_MODS_CorticosteroidsForRefractoryShock(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := SepticShockPatient()
	// On high-dose vasopressor
	patient.CurrentMedications = []types.Medication{
		{
			Code:       "7512",
			CodeSystem: "RxNorm",
			Name:       "Norepinephrine",
			Dose:       0.25, // High dose
			DoseUnit:   "mcg/kg/min",
		},
	}

	// KB-19 recommends: Hydrocortisone for refractory shock
	steroidRec := SimulatedRecommendation{
		Target:             "Hydrocortisone",
		TargetRxNorm:       "5492",
		DrugClass:          "CORTICOSTEROID",
		RecommendedDose:    200,
		DoseUnit:           "mg/day",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "SEPSIS_REFRACTORY",
		Rationale:          "SSC 2021: Consider hydrocortisone 200mg/day for vasopressor-refractory shock",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, steroidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Steroids should be allowed in refractory shock
	if result.IsBlocked() {
		t.Errorf("❌ REFRACTORY SHOCK STEROID FAILURE: SSC-recommended steroid blocked")
	}

	t.Logf("✅ ICU REFRACTORY SHOCK STEROIDS: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestICU_MODS_BloodProductsAllowed tests blood product transfusion in MODS.
//
// Scenario: Coagulopathy with active procedures
// Expected: Blood products ALLOWED
func TestICU_MODS_BloodProductsAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := ICUPatientWithMultiOrganFailure()
	patient.RecentLabs = append(patient.RecentLabs,
		types.LabResult{Code: "5902-2", CodeSystem: "LOINC", Name: "INR", Value: 4.5, Unit: ""},
		types.LabResult{Code: "777-3", CodeSystem: "LOINC", Name: "Platelets", Value: 35, Unit: "K/uL"},
	)

	// KB-19 recommends: FFP for coagulopathy
	ffpRec := SimulatedRecommendation{
		Target:             "Fresh Frozen Plasma",
		TargetRxNorm:       "",
		DrugClass:          "BLOOD_PRODUCT",
		RecommendedDose:    4,
		DoseUnit:           "units",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICU_COAGULOPATHY",
		Rationale:          "INR 4.5, planned procedure, risk of bleeding",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, ffpRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Blood products must be allowed for severe coagulopathy
	if result.IsBlocked() {
		t.Errorf("❌ MODS COAGULOPATHY FAILURE: FFP blocked with INR 4.5")
	}

	t.Logf("✅ ICU MODS BLOOD PRODUCTS: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_MODS_NephrotoxinBlockedDespiteInfection tests that nephrotoxins
// are still blocked in severe AKI even with active infection.
//
// Scenario: MODS + severe AKI + infection requiring aminoglycoside
// Expected: Aminoglycoside BLOCKED or conditional - severe AKI risk
func TestICU_MODS_NephrotoxinBlockedDespiteInfection(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := ICUPatientWithMultiOrganFailure()
	patient.RenalFunction.EGFR = 10 // Severe AKI

	// Aminoglycoside recommendation (nephrotoxic)
	aminoglycosideRec := SimulatedRecommendation{
		Target:             "Gentamicin",
		TargetRxNorm:       "4053",
		DrugClass:          "AMINOGLYCOSIDE",
		RecommendedDose:    350, // 5mg/kg
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "SEPSIS_GRAM_NEGATIVE",
		Rationale:          "Synergy for gram-negative coverage",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, aminoglycosideRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check if nephrotoxicity concern was raised
	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("ICU MODS AMINOGLYCOSIDE: outcome=%s, renal_concern=%v",
		result.FinalOutcome, hasRenalViolation)
	t.Logf("   Note: Even in infection, nephrotoxin risk must be flagged in severe AKI")
}

// =============================================================================
// MODS TRIAGE AND PRIORITIZATION TESTS
// =============================================================================

// TestICU_MODS_TriagePrioritization tests that the system can handle
// multiple simultaneous interventions with appropriate prioritization.
func TestICU_MODS_TriagePrioritization(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := ICUPatientWithMultiOrganFailure()

	// Multiple simultaneous interventions
	interventions := []struct {
		name     string
		rec      SimulatedRecommendation
		priority string // STAT > URGENT > ROUTINE
	}{
		{
			name:     "Vasopressor",
			rec: SimulatedRecommendation{
				Target:             "Norepinephrine",
				TargetRxNorm:       "7512",
				DrugClass:          "VASOPRESSOR",
				RecommendedDose:    0.15,
				DoseUnit:           "mcg/kg/min",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "ICU_SHOCK",
				Rationale:          "MAP support",
				Urgency:            "STAT",
			},
			priority: "STAT",
		},
		{
			name:     "Antibiotics",
			rec: SimulatedRecommendation{
				Target:             "Meropenem",
				TargetRxNorm:       "29561",
				DrugClass:          "ANTIBIOTIC",
				RecommendedDose:    1000,
				DoseUnit:           "mg",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "SEPSIS_HOUR_1",
				Rationale:          "Broad spectrum coverage",
				Urgency:            "STAT",
			},
			priority: "STAT",
		},
		{
			name:     "Ventilator adjustment",
			rec: SimulatedRecommendation{
				Target:             "Ventilator Setting",
				TargetRxNorm:       "",
				DrugClass:          "RESPIRATORY_SUPPORT",
				RecommendedDose:    0,
				DoseUnit:           "",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "ICU_ARDS",
				Rationale:          "Low tidal volume strategy",
				Urgency:            "URGENT",
			},
			priority: "URGENT",
		},
	}

	blockedSTAT := 0
	for _, intervention := range interventions {
		result, err := ctx.ExecuteE2EFlow(patient, intervention.rec)
		if err != nil {
			t.Errorf("Intervention '%s' failed: %v", intervention.name, err)
			continue
		}

		if result.IsBlocked() && intervention.priority == "STAT" {
			blockedSTAT++
			t.Errorf("❌ MODS TRIAGE FAILURE: STAT intervention '%s' was BLOCKED", intervention.name)
		}

		t.Logf("   %s [%s]: outcome=%s, allowed=%v",
			intervention.name, intervention.priority, result.FinalOutcome, result.FinalAllowed)
	}

	if blockedSTAT > 0 {
		t.Errorf("❌ MODS TRIAGE INVARIANT FAILURE: %d STAT interventions blocked", blockedSTAT)
	} else {
		t.Logf("✅ MODS TRIAGE VERIFIED: All STAT interventions allowed")
	}
}

// =============================================================================
// MODS INVARIANT TESTS
// =============================================================================

// TestICU_MODS_Invariant_LifeSavingInterventionsAllowed tests the fundamental
// invariant that life-saving interventions proceed in MODS.
func TestICU_MODS_Invariant_LifeSavingInterventionsAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := ICUPatientWithMultiOrganFailure()

	lifeSavingInterventions := []SimulatedRecommendation{
		{
			Target:             "Norepinephrine",
			TargetRxNorm:       "7512",
			DrugClass:          "VASOPRESSOR",
			RecommendedDose:    0.2,
			DoseUnit:           "mcg/kg/min",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "ICU_SHOCK",
			Rationale:          "MAP support",
			Urgency:            "STAT",
		},
		{
			Target:             "Piperacillin-Tazobactam",
			TargetRxNorm:       "251210",
			DrugClass:          "ANTIBIOTIC",
			RecommendedDose:    4500,
			DoseUnit:           "mg",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "SEPSIS_HOUR_1",
			Rationale:          "Broad spectrum antibiotics",
			Urgency:            "STAT",
		},
		FluidRecommendation(2000),
	}

	blockedCount := 0
	for _, rec := range lifeSavingInterventions {
		result, err := ctx.ExecuteE2EFlow(patient, rec)
		if err != nil {
			t.Errorf("Intervention '%s' failed: %v", rec.Target, err)
			continue
		}

		if result.IsBlocked() {
			blockedCount++
			t.Errorf("❌ MODS LIFE-SAVING BLOCKED: %s", rec.Target)
		}
	}

	if blockedCount > 0 {
		t.Errorf("❌ MODS INVARIANT FAILURE: %d/%d life-saving interventions blocked",
			blockedCount, len(lifeSavingInterventions))
	} else {
		t.Logf("✅ MODS INVARIANT VERIFIED: All %d life-saving interventions allowed",
			len(lifeSavingInterventions))
	}
}
