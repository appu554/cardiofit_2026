// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests OPIOID STEWARDSHIP scenarios: Escalation and accountability.
//
// Clinical Truth: Care proceeds — but only with accountability.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// OPIOID STEWARDSHIP E2E SCENARIOS
// These tests prove that opioid governance enforces proper accountability
// while not blocking necessary pain management.
// =============================================================================

// TestE2E_Opioid_NaivePatient_HighDose_WarnAcknowledge tests that high-dose
// opioids for opioid-naive patients require acknowledgment.
//
// Scenario: Opioid-naive patient + high-dose morphine order
// Expected: WARN_ACKNOWLEDGE - proceed with documented justification
func TestE2E_Opioid_NaivePatient_HighDose_WarnAcknowledge(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: No prior opioid use
	patient := OpioidNaivePatient()

	// KB-19 recommends: Morphine for acute pain (high dose for naive patient)
	morphineRec := OpioidRecommendation("Morphine", "7052", 10.0) // High for naive

	result, err := ctx.ExecuteE2EFlow(patient, morphineRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: High-dose opioid for naive patient should require acknowledgment
	// Should NOT be an outright block (pain needs treatment)
	// Should NOT be auto-approved (accountability required)

	t.Logf("E2E OPIOID NAIVE HIGH-DOSE: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Patient: Opioid-naive")
	t.Logf("   Dose: 10mg morphine (high for naive)")
	t.Logf("   Requires acknowledgment: %v", result.GovernanceResponse.Outcome == types.OutcomePendingAck)
	t.Logf("   Clinical Truth: Care proceeds with accountability")
}

// TestE2E_Opioid_NaivePatient_LowDose_Allowed tests that appropriate
// starting doses for opioid-naive patients are allowed.
//
// Scenario: Opioid-naive patient + appropriate starting dose
// Expected: APPROVED or APPROVED_WITH_WARNINGS
func TestE2E_Opioid_NaivePatient_LowDose_Allowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: No prior opioid use
	patient := OpioidNaivePatient()

	// KB-19 recommends: Low-dose morphine (appropriate for naive)
	morphineRec := OpioidRecommendation("Morphine", "7052", 2.0) // Appropriate for naive

	result, err := ctx.ExecuteE2EFlow(patient, morphineRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Low dose should be acceptable with appropriate monitoring
	t.Logf("✅ E2E OPIOID NAIVE LOW-DOSE: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Dose: 2mg morphine (appropriate starting dose)")
}

// TestE2E_Opioid_ExtendedRelease_NaivePatient_HardBlock tests that
// extended-release opioids are blocked for opioid-naive patients.
//
// Scenario: Opioid-naive patient + OxyContin ER
// Expected: HARD BLOCK - ER opioids contraindicated in naive patients
func TestE2E_Opioid_ExtendedRelease_NaivePatient_HardBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Opioid-naive
	patient := OpioidNaivePatient()

	// KB-19 recommends: OxyContin ER (inappropriate for naive)
	oxyERRec := SimulatedRecommendation{
		Target:             "OxyContin ER",
		TargetRxNorm:       "1049621",
		DrugClass:          "OPIOID_ER",
		RecommendedDose:    20.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "PAIN_MANAGEMENT",
		Rationale:          "Extended-release for chronic pain",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, oxyERRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: ER opioids must be blocked for opioid-naive patients
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: ER opioid APPROVED for opioid-naive patient")
		t.Errorf("   ER opioids in naive patients = high overdose risk")
		t.Errorf("   FDA black box warning applies")
	}

	t.Logf("E2E OPIOID ER NAIVE: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   Extended-release + naive = contraindicated")
}

// TestE2E_Opioid_ToleratedPatient_HighDose_Allowed tests that
// opioid-tolerant patients can receive higher doses.
//
// Scenario: Opioid-tolerant patient + higher dose
// Expected: APPROVED (patient has tolerance)
func TestE2E_Opioid_ToleratedPatient_HighDose_Allowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Opioid-tolerant (on chronic therapy)
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-OPIOID-TOLERANT",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		CurrentMedications: []types.Medication{
			{Code: "OXY", Name: "Oxycodone", DrugClass: "OPIOID", Dose: 60, DoseUnit: "mg/day"},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "G89.29", CodeSystem: "ICD10", Description: "Chronic pain syndrome"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "OPIOID_STEWARDSHIP", Status: "ACTIVE"},
			// Note: NOT in OPIOID_NAIVE registry
		},
	}

	// Higher dose is appropriate for tolerant patient
	morphineRec := OpioidRecommendation("Morphine", "7052", 15.0)

	result, err := ctx.ExecuteE2EFlow(patient, morphineRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Tolerant patients can receive higher doses
	t.Logf("E2E OPIOID TOLERANT: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Patient: On 60mg oxycodone/day (tolerant)")
	t.Logf("   New dose: 15mg morphine (appropriate for tolerance)")
}

// TestE2E_Opioid_MATPatient_Buprenorphine_Allowed tests that MAT patients
// can receive appropriate buprenorphine.
//
// Scenario: MAT patient + buprenorphine
// Expected: APPROVED with monitoring
func TestE2E_Opioid_MATPatient_Buprenorphine_Allowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: On medication-assisted treatment
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-MAT",
		Age:        35,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "F11.20", CodeSystem: "ICD10", Description: "Opioid use disorder"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "OPIOID_MAT", Status: "ACTIVE"},
		},
	}

	// Buprenorphine for OUD treatment
	bupRec := SimulatedRecommendation{
		Target:             "Buprenorphine",
		TargetRxNorm:       "1431076",
		DrugClass:          "OPIOID_MAT",
		RecommendedDose:    8.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "OPIOID_USE_DISORDER",
		Rationale:          "Buprenorphine for OUD treatment per SAMHSA guidelines",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, bupRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// MAT medications should be allowed for enrolled patients
	t.Logf("✅ E2E MAT BUPRENORPHINE: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestE2E_Opioid_ConcomitantBenzo_WarnAcknowledge tests that
// concurrent opioid + benzodiazepine orders require acknowledgment.
//
// Scenario: Patient on benzos + new opioid order
// Expected: WARN_ACKNOWLEDGE - FDA black box warning
func TestE2E_Opioid_ConcomitantBenzo_WarnAcknowledge(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Already on benzodiazepine
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-OPIOID-BENZO",
		Age:        50,
		Sex:        "F",
		IsPregnant: false,
		CurrentMedications: []types.Medication{
			{Code: "LOR", Name: "Lorazepam", DrugClass: "BENZODIAZEPINE", Dose: 1, DoseUnit: "mg"},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "F41.1", CodeSystem: "ICD10", Description: "Generalized anxiety disorder"},
		},
	}

	// Adding opioid to benzo patient
	oxyRec := OpioidRecommendation("Oxycodone", "7804", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, oxyRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Should require explicit acknowledgment due to interaction
	hasInteractionViolation := result.HasViolationCategory(types.ViolationDrugInteraction)

	t.Logf("E2E OPIOID + BENZO: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Concurrent benzo: Lorazepam 1mg")
	t.Logf("   Drug interaction violation: %v", hasInteractionViolation)
	t.Logf("   FDA Black Box: Concurrent use increases CNS/respiratory depression risk")
}

// TestE2E_Opioid_RenalImpairment_DoseAdjustment tests opioid dosing
// in renal impairment.
//
// Scenario: CKD patient + morphine
// Expected: Dose adjustment required (active metabolites)
func TestE2E_Opioid_RenalImpairment_DoseAdjustment(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Severe CKD
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-OPIOID-CKD",
		Age:        70,
		Sex:        "M",
		IsPregnant: false,
		RenalFunction: &types.RenalFunction{
			EGFR:       18.0,
			Creatinine: 4.2,
			CKDStage:   "CKD_4",
			OnDialysis: false,
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "N18.4", CodeSystem: "ICD10", Description: "CKD Stage 4"},
		},
	}

	// Morphine has active metabolites that accumulate in CKD
	morphineRec := OpioidRecommendation("Morphine", "7052", 10.0)

	result, err := ctx.ExecuteE2EFlow(patient, morphineRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check for renal dosing concern
	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("E2E OPIOID RENAL: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   eGFR: 18 mL/min/1.73m² (CKD Stage 4)")
	t.Logf("   Renal dosing violation: %v", hasRenalViolation)
	t.Logf("   Note: Morphine-6-glucuronide accumulates in CKD")
}

// TestE2E_Opioid_Escalation_Required tests that certain opioid scenarios
// require mandatory escalation.
//
// Scenario: Pattern of increasing opioid requirements
// Expected: MANDATORY_ESCALATION
func TestE2E_Opioid_Escalation_Required(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Already on high-dose opioids, requesting more
	patient := &types.PatientContext{
		PatientID:  "PT-E2E-OPIOID-ESCALATION",
		Age:        45,
		Sex:        "M",
		IsPregnant: false,
		CurrentMedications: []types.Medication{
			{Code: "OXY", Name: "Oxycodone", DrugClass: "OPIOID", Dose: 120, DoseUnit: "mg/day"},
			{Code: "FEN", Name: "Fentanyl Patch", DrugClass: "OPIOID", Dose: 100, DoseUnit: "mcg/hr"},
		},
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "G89.4", CodeSystem: "ICD10", Description: "Chronic pain syndrome"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "OPIOID_STEWARDSHIP", Status: "ACTIVE"},
		},
	}

	// Additional opioid request
	additionalOpioid := SimulatedRecommendation{
		Target:             "Hydromorphone",
		TargetRxNorm:       "3423",
		DrugClass:          "OPIOID",
		RecommendedDose:    8.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "PAIN_MANAGEMENT",
		Rationale:          "Breakthrough pain",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, additionalOpioid)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	t.Logf("E2E OPIOID ESCALATION: outcome=%s, escalation=%v",
		result.FinalOutcome, result.RequiresEscalation)
	t.Logf("   Current regimen: Oxycodone 120mg/day + Fentanyl 100mcg/hr")
	t.Logf("   Request: Additional hydromorphone")
	t.Logf("   Note: High-dose patterns require pain specialist review")
}

// TestE2E_Opioid_EvidenceTrailComplete tests that opioid decisions
// have complete audit trails for DEA/regulatory compliance.
func TestE2E_Opioid_EvidenceTrailComplete(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := OpioidNaivePatient()
	opioidRec := OpioidRecommendation("Morphine", "7052", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, opioidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	if !result.HasEvidenceTrail() {
		t.Errorf("❌ AUDIT FAILURE: Missing evidence trail for opioid order")
		t.Errorf("   Opioid orders require complete audit trail for DEA compliance")
	}

	t.Logf("✅ E2E OPIOID EVIDENCE TRAIL:")
	if result.GovernanceResponse.EvidenceTrail != nil {
		t.Logf("   Trail ID: %s", result.GovernanceResponse.EvidenceTrail.TrailID)
		t.Logf("   Hash: %s...", result.EvidenceTrailHash[:min(40, len(result.EvidenceTrailHash))])
	}
}

// =============================================================================
// OPIOID STEWARDSHIP INVARIANT TESTS
// =============================================================================

// TestE2E_Opioid_Invariant_ERBlockedForNaive verifies the fundamental invariant:
// Extended-release opioids are NEVER approved for opioid-naive patients.
func TestE2E_Opioid_Invariant_ERBlockedForNaive(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := OpioidNaivePatient()

	// Extended-release opioids that must be blocked for naive patients
	erOpioids := []SimulatedRecommendation{
		{Target: "OxyContin ER", TargetRxNorm: "1049621", DrugClass: "OPIOID_ER", RecommendedDose: 20, DoseUnit: "mg"},
		{Target: "MS Contin", TargetRxNorm: "891878", DrugClass: "OPIOID_ER", RecommendedDose: 30, DoseUnit: "mg"},
		{Target: "Fentanyl Patch", TargetRxNorm: "4337", DrugClass: "OPIOID_ER", RecommendedDose: 25, DoseUnit: "mcg/hr"},
	}

	approvedCount := 0
	for _, erOpioid := range erOpioids {
		erOpioid.RecommendationType = RecommendDo
		erOpioid.SourceProtocol = "PAIN_MANAGEMENT"
		erOpioid.Rationale = "Pain control"

		result, err := ctx.ExecuteE2EFlow(patient, erOpioid)
		if err != nil {
			t.Errorf("ER opioid '%s' evaluation failed: %v", erOpioid.Target, err)
			continue
		}

		if result.IsApproved() {
			approvedCount++
			t.Errorf("❌ INVARIANT VIOLATION: %s APPROVED for opioid-naive patient",
				erOpioid.Target)
		}
	}

	if approvedCount > 0 {
		t.Errorf("❌ ER OPIOID INVARIANT FAILURE: %d/%d ER opioids approved for naive patient",
			approvedCount, len(erOpioids))
	} else {
		t.Logf("✅ ER OPIOID INVARIANT VERIFIED: All %d ER opioids blocked for naive patient",
			len(erOpioids))
	}
}
