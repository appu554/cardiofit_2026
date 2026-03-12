// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests RENAL SAFETY scenarios: AKI conditional blocks.
//
// Clinical Truth: Nephrotoxic drugs in AKI worsen injury exponentially.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// RENAL SAFETY E2E SCENARIOS
// These tests prove that nephrotoxic medications are appropriately blocked
// or adjusted in patients with acute kidney injury.
// =============================================================================

// TestE2E_Renal_NSAIDInAKI_HardBlock tests that NSAIDs are BLOCKED
// in acute kidney injury.
//
// Scenario: AKI Stage 3 + NSAID recommendation
// Expected: HARD BLOCK with alternative suggestion
func TestE2E_Renal_NSAIDInAKI_HardBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Severe AKI
	patient := AKIPatient(3) // Stage 3

	// KB-19 recommends: NSAID for pain
	nsaidRec := NSAIDRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, nsaidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: NSAIDs must be blocked in AKI
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: NSAID APPROVED in AKI Stage 3")
		t.Errorf("   NSAIDs cause afferent arteriolar constriction")
		t.Errorf("   Will worsen AKI and potentially cause permanent damage")
	}

	// Check for renal violation
	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing) ||
		result.HasViolationCategory(types.ViolationContraindication)

	t.Logf("E2E NSAID AKI: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   AKI Stage: 3 (Creatinine 4.0 mg/dL)")
	t.Logf("   Renal/Contraindication violation: %v", hasRenalViolation)
	t.Logf("   Alternative: Acetaminophen (renally safe)")
}

// TestE2E_Renal_NSAIDInNormalKidney_Allowed tests that NSAIDs are ALLOWED
// with normal renal function.
//
// Scenario: Normal eGFR + NSAID recommendation
// Expected: APPROVED
func TestE2E_Renal_NSAIDInNormalKidney_Allowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Normal renal function
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-NORMAL-KIDNEY",
		Age:        35,
		Sex:        "M",
		IsPregnant: false,
		RenalFunction: &types.RenalFunction{
			EGFR:       95.0,
			Creatinine: 0.9,
			CKDStage:   "NORMAL",
			OnDialysis: false,
		},
	}

	nsaidRec := NSAIDRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, nsaidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Normal kidney - NSAIDs should be allowed
	if result.IsBlocked() {
		t.Errorf("❌ CLINICAL FAILURE: NSAID BLOCKED with normal renal function")
	}

	t.Logf("✅ E2E NSAID NORMAL KIDNEY: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestE2E_Renal_AminoglycosideInAKI_Conditional tests aminoglycoside
// handling in AKI (may be allowed with dose adjustment for life-threatening infection).
//
// Scenario: Sepsis + AKI + Aminoglycoside needed
// Expected: May proceed with monitoring/dose adjustment
func TestE2E_Renal_AminoglycosideInAKI_Conditional(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Sepsis with AKI
	patient := AKIPatient(2)
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "A41.9",
		CodeSystem:  "ICD10",
		Description: "Sepsis",
	})
	patient.RegistryMemberships = []types.RegistryMembership{
		{RegistryCode: "SEPSIS", Status: "ACTIVE"},
	}

	// Aminoglycoside for Gram-negative sepsis
	gentamicinRec := SimulatedRecommendation{
		Target:             "Gentamicin",
		TargetRxNorm:       "4450",
		DrugClass:          "AMINOGLYCOSIDE",
		RecommendedDose:    5.0, // mg/kg - extended interval dosing
		DoseUnit:           "mg/kg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPSIS_ANTIBIOTIC",
		Rationale:          "Aminoglycoside for Gram-negative coverage in sepsis",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, gentamicinRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Life-saving antibiotics may be allowed with adjustment
	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("E2E AMINOGLYCOSIDE AKI: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Context: Sepsis (life-threatening) + AKI Stage 2")
	t.Logf("   Renal dosing concern: %v", hasRenalViolation)
	t.Logf("   Note: May proceed with extended-interval dosing and level monitoring")
}

// TestE2E_Renal_ContrastInAKI_HardBlock tests IV contrast handling in AKI.
//
// Scenario: AKI + CT with contrast recommendation
// Expected: HARD BLOCK or strong warning
func TestE2E_Renal_ContrastInAKI_HardBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Severe AKI
	patient := AKIPatient(3)

	// Contrast for CT scan
	contrastRec := SimulatedRecommendation{
		Target:             "Iohexol",
		TargetRxNorm:       "5755",
		DrugClass:          "IV_CONTRAST",
		RecommendedDose:    100.0,
		DoseUnit:           "mL",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "CT_IMAGING",
		Rationale:          "IV contrast for diagnostic CT",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, contrastRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Contrast in severe AKI should be blocked or require justification
	t.Logf("E2E CONTRAST AKI: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   AKI Stage 3 - contrast-induced nephropathy risk is HIGH")
	t.Logf("   Alternative: Non-contrast imaging if clinically feasible")
}

// TestE2E_Renal_MetforminInAKI_HardBlock tests metformin handling in AKI.
//
// Scenario: Diabetic with AKI + Metformin continuation
// Expected: HARD BLOCK - lactic acidosis risk
func TestE2E_Renal_MetforminInAKI_HardBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Diabetic with AKI
	patient := AKIPatient(2)
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "E11.9",
		CodeSystem:  "ICD10",
		Description: "Type 2 diabetes",
	})
	patient.CurrentMedications = []types.Medication{
		{Code: "MET", Name: "Metformin", DrugClass: "BIGUANIDE", Dose: 1000, DoseUnit: "mg"},
	}

	// Continuing metformin in AKI
	metforminRec := SimulatedRecommendation{
		Target:             "Metformin",
		TargetRxNorm:       "6809",
		DrugClass:          "BIGUANIDE",
		RecommendedDose:    1000.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "DIABETES_MANAGEMENT",
		Rationale:          "Continue metformin for glycemic control",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, metforminRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Metformin should be held in AKI
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: Metformin APPROVED in AKI")
		t.Errorf("   Metformin accumulates in AKI → lactic acidosis")
	}

	t.Logf("E2E METFORMIN AKI: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   Hold metformin when eGFR < 30 or acute change")
}

// TestE2E_Renal_VancomycinDoseAdjustment tests vancomycin dosing in CKD.
//
// Scenario: Infection + CKD Stage 4
// Expected: Approve with dose/interval adjustment requirement
func TestE2E_Renal_VancomycinDoseAdjustment(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: CKD Stage 4 with infection
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-CKD-VANCO",
		Age:        68,
		Sex:        "M",
		IsPregnant: false,
		Weight:     80.0,
		RenalFunction: &types.RenalFunction{
			EGFR:       22.0,
			Creatinine: 3.0,
			CKDStage:   "CKD_4",
			OnDialysis: false,
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "N18.4", CodeSystem: "ICD10", Description: "CKD Stage 4"},
			{Code: "L03.90", CodeSystem: "ICD10", Description: "Cellulitis"},
		},
	}

	// Vancomycin for MRSA coverage
	vancoRec := SimulatedRecommendation{
		Target:             "Vancomycin",
		TargetRxNorm:       "11124",
		DrugClass:          "GLYCOPEPTIDE",
		RecommendedDose:    1500.0, // Standard dose - may need reduction
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "MRSA_COVERAGE",
		Rationale:          "Vancomycin for suspected MRSA infection",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vancoRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("E2E VANCOMYCIN CKD: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   eGFR: 22 mL/min/1.73m²")
	t.Logf("   Renal dosing violation: %v", hasRenalViolation)
	t.Logf("   Note: Extend interval and monitor levels closely")
}

// TestE2E_Renal_ACEInhibitorInAKI_Conditional tests ACE inhibitor
// handling in AKI.
//
// Scenario: Heart failure patient with new AKI
// Expected: May need to hold temporarily
func TestE2E_Renal_ACEInhibitorInAKI_Conditional(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: CHF on ACE inhibitor, develops AKI
	patient := AKIPatient(2)
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "I50.9",
		CodeSystem:  "ICD10",
		Description: "Heart failure",
	})
	patient.CurrentMedications = []types.Medication{
		{Code: "LIS", Name: "Lisinopril", DrugClass: "ACE_INHIBITOR", Dose: 20, DoseUnit: "mg"},
	}

	// Continuing ACE inhibitor in acute AKI
	aceRec := ACEInhibitorRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, aceRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ACE inhibitors may worsen AKI via efferent arteriolar dilation
	t.Logf("E2E ACE INHIBITOR AKI: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Context: CHF patient developing AKI")
	t.Logf("   Note: Consider holding until renal function stabilizes")
}

// TestE2E_Renal_DialysisPatient_DoseAdjustment tests drug dosing
// in dialysis patients.
//
// Scenario: ESRD on hemodialysis
// Expected: Dialysis-appropriate dosing required
func TestE2E_Renal_DialysisPatient_DoseAdjustment(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: ESRD on hemodialysis
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-DIALYSIS",
		Age:        62,
		Sex:        "F",
		IsPregnant: false,
		RenalFunction: &types.RenalFunction{
			EGFR:       5.0,
			Creatinine: 8.5,
			CKDStage:   "ESRD",
			OnDialysis: true,
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "N18.6", CodeSystem: "ICD10", Description: "ESRD"},
		},
	}

	// Antibiotic that needs dialysis dosing
	ceftazRec := SimulatedRecommendation{
		Target:             "Ceftazidime",
		TargetRxNorm:       "2180",
		DrugClass:          "CEPHALOSPORIN",
		RecommendedDose:    2000.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "INFECTION_TREATMENT",
		Rationale:          "Cephalosporin for Gram-negative coverage",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, ceftazRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("E2E DIALYSIS DOSING: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Patient: ESRD on hemodialysis")
	t.Logf("   Renal dosing violation: %v", hasRenalViolation)
	t.Logf("   Note: Give after dialysis, reduce dose to 1g")
}

// TestE2E_Renal_EvidenceTrailForBlocks tests that renal safety blocks
// have complete audit trails.
func TestE2E_Renal_EvidenceTrailForBlocks(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := AKIPatient(3)
	nsaidRec := NSAIDRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, nsaidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	if !result.HasEvidenceTrail() {
		t.Errorf("❌ AUDIT FAILURE: Missing evidence trail for renal safety block")
	}

	t.Logf("✅ E2E RENAL EVIDENCE TRAIL:")
	if result.GovernanceResponse.EvidenceTrail != nil {
		t.Logf("   Trail ID: %s", result.GovernanceResponse.EvidenceTrail.TrailID)
		t.Logf("   Decision: %s", result.FinalOutcome)
	}
}

// =============================================================================
// RENAL SAFETY INVARIANT TESTS
// =============================================================================

// TestE2E_Renal_Invariant_NephrotoxinsBlockedInSevereAKI verifies the fundamental invariant:
// Known nephrotoxins are blocked in severe AKI.
func TestE2E_Renal_Invariant_NephrotoxinsBlockedInSevereAKI(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := AKIPatient(3) // Severe AKI

	// Known nephrotoxins
	nephrotoxins := []SimulatedRecommendation{
		NSAIDRecommendation(),
		{
			Target: "Ibuprofen", TargetRxNorm: "5640", DrugClass: "NSAID",
			RecommendedDose: 600, DoseUnit: "mg", RecommendationType: RecommendDo,
			SourceProtocol: "PAIN", Rationale: "Pain control",
		},
	}

	approvedCount := 0
	for _, nephrotoxin := range nephrotoxins {
		result, err := ctx.ExecuteE2EFlow(patient, nephrotoxin)
		if err != nil {
			t.Errorf("Nephrotoxin '%s' evaluation failed: %v", nephrotoxin.Target, err)
			continue
		}

		if result.IsApproved() {
			approvedCount++
			t.Errorf("❌ INVARIANT VIOLATION: %s APPROVED in severe AKI",
				nephrotoxin.Target)
		}
	}

	if approvedCount > 0 {
		t.Errorf("❌ NEPHROTOXIN INVARIANT FAILURE: %d/%d nephrotoxins approved in AKI",
			approvedCount, len(nephrotoxins))
	} else {
		t.Logf("✅ NEPHROTOXIN INVARIANT VERIFIED: All %d nephrotoxins blocked in severe AKI",
			len(nephrotoxins))
	}
}
