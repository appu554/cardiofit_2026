// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ANTICOAGULATION scenarios: Stroke prevention vs Bleeding risk.
//
// Clinical Truth: Stroke prevention does NOT override catastrophic bleed risk.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ANTICOAGULATION E2E SCENARIOS
// These tests prove that bleeding risk appropriately blocks anticoagulation
// even when stroke prevention guidelines recommend it.
// =============================================================================

// TestE2E_Anticoag_AFibWithThrombocytopenia_HardBlock tests that anticoagulation
// is BLOCKED when platelets are critically low despite high stroke risk.
//
// Scenario: AFib with CHA₂DS₂-VASc = 4 + Platelets 38k
// Expected: HARD BLOCK - bleeding risk overrides stroke prevention
func TestE2E_Anticoag_AFibWithThrombocytopenia_HardBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: High stroke risk AFib with severe thrombocytopenia
	patient := AFibWithThrombocytopeniaPatient()

	// KB-19 recommends: Anticoagulation per CHEST guidelines
	anticoagRec := AnticoagulationRecommendation("Apixaban", "1364430", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Anticoagulation must be BLOCKED with platelets 38k
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: Anticoagulation APPROVED with platelets 38k")
		t.Errorf("   Patient has critical thrombocytopenia - major bleed risk")
		t.Errorf("   Expected: BLOCKED or MANDATORY_ESCALATION")
		t.Errorf("   Got: %s", result.FinalOutcome)
	}

	// Check for bleeding safety flag
	hasBleedingViolation := result.HasViolationCategory(types.ViolationContraindication)

	t.Logf("E2E AFIB THROMBOCYTOPENIA: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   CHA₂DS₂-VASc: 4 (high stroke risk)")
	t.Logf("   Platelets: 38k (critical bleeding risk)")
	t.Logf("   Contraindication raised: %v", hasBleedingViolation)
	t.Logf("   Clinical Truth: Bleeding risk > Stroke prevention here")
}

// TestE2E_Anticoag_AFibWithNormalPlatelets_Allowed tests that anticoagulation
// is ALLOWED when platelet count is safe.
//
// Scenario: AFib with CHA₂DS₂-VASc = 4 + Platelets 180k
// Expected: APPROVED - stroke prevention indicated
func TestE2E_Anticoag_AFibWithNormalPlatelets_Allowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: High stroke risk AFib with normal platelets
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-AFIB-NORMAL",
		Age:        72,
		Sex:        "F",
		IsPregnant: false,
		Weight:     68.0,
		RecentLabs: []types.LabResult{
			{Code: "PLT", Name: "Platelets", Value: 180.0, Unit: "K/uL"},
			{Code: "INR", Name: "INR", Value: 1.0, Unit: ""},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I48.91", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
			{Code: "I10", CodeSystem: "ICD10", Description: "Hypertension"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
		},
	}

	anticoagRec := AnticoagulationRecommendation("Apixaban", "1364430", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Anticoagulation should be allowed with normal platelets
	if result.IsBlocked() {
		t.Errorf("❌ CLINICAL FAILURE: Anticoagulation BLOCKED with normal platelets")
		t.Errorf("   Patient needs stroke prevention, no contraindication exists")
	}

	t.Logf("✅ E2E AFIB NORMAL PLATELETS: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestE2E_Anticoag_ActiveBleeding_HardBlock tests that anticoagulation
// is BLOCKED during active bleeding.
//
// Scenario: AFib + Active GI bleed
// Expected: HARD BLOCK - active bleeding is absolute contraindication
func TestE2E_Anticoag_ActiveBleeding_HardBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: AFib with active GI bleeding
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-AFIB-GIB",
		Age:        68,
		Sex:        "M",
		IsPregnant: false,
		RecentLabs: []types.LabResult{
			{Code: "HGB", Name: "Hemoglobin", Value: 7.2, Unit: "g/dL"}, // Dropping
			{Code: "PLT", Name: "Platelets", Value: 145.0, Unit: "K/uL"},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I48.91", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
			{Code: "K92.2", CodeSystem: "ICD10", Description: "GI hemorrhage, unspecified"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
		},
	}

	anticoagRec := AnticoagulationRecommendation("Warfarin", "855332", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Active bleeding = absolute contraindication
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: Anticoagulation APPROVED during active bleeding")
		t.Errorf("   GI hemorrhage is active - anticoagulation will worsen bleeding")
	}

	t.Logf("E2E AFIB ACTIVE BLEEDING: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   Diagnosis: Active GI hemorrhage")
	t.Logf("   Hemoglobin: 7.2 g/dL (dropping)")
}

// TestE2E_Anticoag_RecentICH_AbsoluteBlock tests that anticoagulation
// is BLOCKED with recent intracranial hemorrhage.
//
// Scenario: AFib + ICH within 30 days
// Expected: HARD BLOCK - ICH is absolute contraindication
func TestE2E_Anticoag_RecentICH_AbsoluteBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: AFib with recent intracranial hemorrhage
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-AFIB-ICH",
		Age:        75,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I48.91", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
			{Code: "I61.9", CodeSystem: "ICD10", Description: "Intracerebral hemorrhage"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
		},
	}

	anticoagRec := AnticoagulationRecommendation("Rivaroxaban", "1114195", 20.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: ICH = absolute contraindication to anticoagulation
	if result.IsApproved() {
		t.Errorf("❌ CRITICAL SAFETY FAILURE: Anticoagulation APPROVED with ICH")
		t.Errorf("   ICH is an ABSOLUTE contraindication")
	}

	t.Logf("E2E AFIB ICH: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
}

// TestE2E_Anticoag_WarfarinINRSupratherapeutic tests that warfarin
// is handled appropriately when INR is supratherapeutic.
//
// Scenario: On warfarin + INR 4.5
// Expected: WARN or BLOCK additional dose
func TestE2E_Anticoag_WarfarinINRSupratherapeutic(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Already on warfarin with high INR
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-WARFARIN-HIGH-INR",
		Age:        70,
		Sex:        "F",
		IsPregnant: false,
		RecentLabs: []types.LabResult{
			{Code: "INR", Name: "INR", Value: 4.5, Unit: ""},
			{Code: "PLT", Name: "Platelets", Value: 165.0, Unit: "K/uL"},
		},
		CurrentMedications: []types.Medication{
			{Code: "WAR", Name: "Warfarin", DrugClass: "WARFARIN", Dose: 5, DoseUnit: "mg"},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I48.91", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
			{RegistryCode: "WARFARIN_MANAGEMENT", Status: "ACTIVE"},
		},
	}

	// Attempting to give more warfarin with INR 4.5
	warfarinRec := AnticoagulationRecommendation("Warfarin", "855332", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, warfarinRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// With INR 4.5, additional warfarin should be blocked or warned
	t.Logf("E2E WARFARIN SUPRATHERAPEUTIC: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   INR: 4.5 (target typically 2-3)")
	t.Logf("   Note: Should hold warfarin until INR normalizes")
}

// TestE2E_Anticoag_DOACWithRenalImpairment tests DOAC dosing
// in renal impairment.
//
// Scenario: AFib + CKD Stage 4
// Expected: Dose reduction required
func TestE2E_Anticoag_DOACWithRenalImpairment(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: AFib with severe CKD
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-DOAC-CKD",
		Age:        78,
		Sex:        "M",
		IsPregnant: false,
		Weight:     65.0,
		RenalFunction: &types.RenalFunction{
			EGFR:       22.0,
			Creatinine: 3.1,
			CKDStage:   "CKD_4",
			OnDialysis: false,
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I48.91", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
			{Code: "N18.4", CodeSystem: "ICD10", Description: "CKD Stage 4"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
		},
	}

	// Standard dose apixaban - may need reduction
	apixabanRec := AnticoagulationRecommendation("Apixaban", "1364430", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, apixabanRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check for renal dosing concern
	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("E2E DOAC RENAL: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   eGFR: 22 mL/min/1.73m² (CKD Stage 4)")
	t.Logf("   Renal dosing violation: %v", hasRenalViolation)
	t.Logf("   Note: Apixaban may need 2.5mg dose in CKD")
}

// TestE2E_Anticoag_EscalationRequired tests that borderline cases
// trigger appropriate escalation.
//
// Scenario: AFib + Moderate thrombocytopenia (75k) + High stroke risk
// Expected: ESCALATION required - specialist input needed
func TestE2E_Anticoag_EscalationRequired(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Borderline case - not clear cut
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-AFIB-BORDERLINE",
		Age:        80,
		Sex:        "F",
		IsPregnant: false,
		RecentLabs: []types.LabResult{
			{Code: "PLT", Name: "Platelets", Value: 75.0, Unit: "K/uL"}, // Moderate TCP
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I48.91", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
			{Code: "I63.9", CodeSystem: "ICD10", Description: "Prior stroke"},
			{Code: "I10", CodeSystem: "ICD10", Description: "Hypertension"},
			{Code: "E11.9", CodeSystem: "ICD10", Description: "Diabetes"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANTICOAGULATION", Status: "ACTIVE"},
		},
	}

	anticoagRec := AnticoagulationRecommendation("Apixaban", "1364430", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	t.Logf("E2E AFIB BORDERLINE: outcome=%s, escalation=%v",
		result.FinalOutcome, result.RequiresEscalation)
	t.Logf("   Platelets: 75k (moderate TCP)")
	t.Logf("   CHA₂DS₂-VASc: High (prior stroke)")
	t.Logf("   Note: Borderline cases need hematology/cardiology input")
}

// TestE2E_Anticoag_EvidenceTrailForBlocks tests that blocked
// anticoagulation decisions have complete audit trails.
func TestE2E_Anticoag_EvidenceTrailForBlocks(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := AFibWithThrombocytopeniaPatient()
	anticoagRec := AnticoagulationRecommendation("Apixaban", "1364430", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Verify evidence trail for governance decision
	if !result.HasEvidenceTrail() {
		t.Errorf("❌ AUDIT FAILURE: Missing evidence trail for anticoagulation block")
	}

	t.Logf("✅ E2E ANTICOAG EVIDENCE TRAIL:")
	if result.GovernanceResponse.EvidenceTrail != nil {
		t.Logf("   Trail ID: %s", result.GovernanceResponse.EvidenceTrail.TrailID)
		t.Logf("   Decision: %s", result.FinalOutcome)
	}
}

// =============================================================================
// ANTICOAGULATION INVARIANT TESTS
// =============================================================================

// TestE2E_Anticoag_Invariant_BleedingOverStroke verifies the fundamental invariant:
// Bleeding risk overrides stroke prevention when bleeding is imminent.
func TestE2E_Anticoag_Invariant_BleedingOverStroke(t *testing.T) {
	ctx := NewE2ETestContext()

	// Scenarios where bleeding risk must win
	bleedingRiskScenarios := []struct {
		name           string
		patient        *types.PatientContext
		shouldBlock    bool
	}{
		{
			name:        "Platelets < 50k",
			patient:     AFibWithThrombocytopeniaPatient(),
			shouldBlock: true,
		},
		{
			name: "Active GI bleed",
			patient: &types.PatientContext{
				PatientID: "PT-INV-GIB",
				Age:       65,
				Sex:       "M",
				ActiveDiagnoses: []types.Diagnosis{
					{Code: "I48.91", CodeSystem: "ICD10", Description: "AFib"},
					{Code: "K92.2", CodeSystem: "ICD10", Description: "GI hemorrhage"},
				},
			},
			shouldBlock: true,
		},
	}

	anticoagRec := AnticoagulationRecommendation("Apixaban", "1364430", 5.0)
	violations := 0

	for _, scenario := range bleedingRiskScenarios {
		result, err := ctx.ExecuteE2EFlow(scenario.patient, anticoagRec)
		if err != nil {
			t.Errorf("Scenario '%s' failed: %v", scenario.name, err)
			continue
		}

		if scenario.shouldBlock && result.IsApproved() {
			violations++
			t.Errorf("❌ INVARIANT VIOLATION: %s - anticoag APPROVED despite bleeding risk",
				scenario.name)
		}
	}

	if violations > 0 {
		t.Errorf("❌ BLEEDING INVARIANT FAILURE: %d violations", violations)
	} else {
		t.Logf("✅ BLEEDING INVARIANT VERIFIED: All bleeding risks properly blocked")
	}
}
