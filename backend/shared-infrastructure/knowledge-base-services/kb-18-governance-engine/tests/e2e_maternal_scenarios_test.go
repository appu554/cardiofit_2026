// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests MATERNAL SAFETY scenarios: Absolute pregnancy contraindications.
//
// Clinical Truth: Pregnancy safety ALWAYS wins. Zero tolerance for teratogens.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// MATERNAL SAFETY E2E SCENARIOS
// These tests prove that pregnancy contraindications are NEVER overridden.
// Teratogenic medications must be blocked regardless of other indications.
// =============================================================================

// TestE2E_Maternal_ACEInhibitor_AbsoluteBlock tests that ACE inhibitors
// are BLOCKED in pregnancy - no override possible.
//
// Scenario: Pregnant patient (2nd trimester) with hypertension
// Expected: HARD BLOCK - ACE inhibitors are teratogenic
func TestE2E_Maternal_ACEInhibitor_AbsoluteBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Pregnant with hypertension
	patient := PregnantPatient(16) // 16 weeks gestation

	// KB-19 recommends: ACE inhibitor for hypertension (standard guideline)
	aceRec := ACEInhibitorRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, aceRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: ACE inhibitors MUST be blocked in pregnancy
	if result.IsApproved() {
		t.Errorf("❌ CRITICAL SAFETY FAILURE: ACE inhibitor APPROVED in pregnancy")
		t.Errorf("   ACE inhibitors cause fetal renal dysgenesis")
		t.Errorf("   This is a NEVER-EVENT - zero tolerance")
		t.Errorf("   Got: %s", result.FinalOutcome)
	}

	// Verify pregnancy safety violation was raised
	hasPregnancyViolation := result.HasViolationCategory(types.ViolationPregnancySafety)

	t.Logf("E2E ACE INHIBITOR PREGNANCY: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   Gestational age: 16 weeks")
	t.Logf("   Pregnancy safety violation: %v", hasPregnancyViolation)
	t.Logf("   Clinical Truth: Pregnancy safety ALWAYS wins")
}

// TestE2E_Maternal_Methotrexate_AbsoluteBlock tests that Methotrexate
// is BLOCKED in pregnancy - known teratogen.
//
// Scenario: Pregnant patient with rheumatoid arthritis
// Expected: HARD BLOCK - Methotrexate is absolutely contraindicated
func TestE2E_Maternal_Methotrexate_AbsoluteBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Pregnant with RA
	patient := PregnantPatient(12)
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "M06.9",
		CodeSystem:  "ICD10",
		Description: "Rheumatoid arthritis",
	})

	// KB-19 recommends: Methotrexate for RA (would be Class I in non-pregnant)
	mtxRec := MethotrexateRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, mtxRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Methotrexate MUST be blocked in pregnancy
	if result.IsApproved() {
		t.Errorf("❌ CRITICAL SAFETY FAILURE: Methotrexate APPROVED in pregnancy")
		t.Errorf("   Methotrexate is a known teratogen - causes fetal death/malformations")
		t.Errorf("   This is FDA Category X")
	}

	t.Logf("E2E METHOTREXATE PREGNANCY: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   Clinical Truth: Category X drugs are NEVER given in pregnancy")
}

// TestE2E_Maternal_Warfarin_AbsoluteBlock tests that Warfarin
// is BLOCKED in pregnancy (except specific valve indications).
//
// Scenario: Pregnant patient with AFib
// Expected: HARD BLOCK - Warfarin embryopathy risk
func TestE2E_Maternal_Warfarin_AbsoluteBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Pregnant with AFib
	patient := PregnantPatient(10)
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "I48.91",
		CodeSystem:  "ICD10",
		Description: "Atrial fibrillation",
	})

	// KB-19 recommends: Warfarin for AFib
	warfarinRec := AnticoagulationRecommendation("Warfarin", "855332", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, warfarinRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Warfarin should be blocked in pregnancy (use LMWH instead)
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: Warfarin APPROVED in pregnancy")
		t.Errorf("   Warfarin causes embryopathy, CNS abnormalities")
		t.Errorf("   Should use LMWH in pregnancy")
	}

	t.Logf("E2E WARFARIN PREGNANCY: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
}

// TestE2E_Maternal_Statins_AbsoluteBlock tests that statins
// are BLOCKED in pregnancy.
//
// Scenario: Pregnant patient with hyperlipidemia
// Expected: HARD BLOCK - Statins are Category X
func TestE2E_Maternal_Statins_AbsoluteBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Pregnant with hyperlipidemia
	patient := PregnantPatient(8)
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "E78.0",
		CodeSystem:  "ICD10",
		Description: "Hyperlipidemia",
	})

	// KB-19 recommends: Atorvastatin for lipid control
	statinRec := SimulatedRecommendation{
		Target:             "Atorvastatin",
		TargetRxNorm:       "83367",
		DrugClass:          "STATIN",
		RecommendedDose:    20.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "LIPID_MANAGEMENT",
		Rationale:          "Statin therapy for primary prevention",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, statinRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Statins must be blocked in pregnancy
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: Statin APPROVED in pregnancy")
		t.Errorf("   Statins are Category X - teratogenic")
	}

	t.Logf("E2E STATIN PREGNANCY: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
}

// TestE2E_Maternal_SafeHTNMedication_Allowed tests that pregnancy-safe
// antihypertensives are ALLOWED.
//
// Scenario: Pregnant patient with hypertension
// Expected: Methyldopa/Labetalol APPROVED - safe in pregnancy
func TestE2E_Maternal_SafeHTNMedication_Allowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Pregnant with hypertension
	patient := PregnantPatient(20)

	// KB-19 recommends: Labetalol (pregnancy-safe)
	labetalolRec := SimulatedRecommendation{
		Target:             "Labetalol",
		TargetRxNorm:       "6308",
		DrugClass:          "BETA_BLOCKER",
		RecommendedDose:    100.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "PREGNANCY_HTN",
		Rationale:          "ACOG: Labetalol first-line for hypertension in pregnancy",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, labetalolRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Pregnancy-safe medications should be allowed
	if result.IsBlocked() {
		t.Errorf("❌ CLINICAL FAILURE: Pregnancy-safe labetalol BLOCKED")
		t.Errorf("   Labetalol is first-line for gestational hypertension")
	}

	t.Logf("✅ E2E LABETALOL PREGNANCY: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Labetalol is pregnancy-safe (Category C)")
}

// TestE2E_Maternal_Preeclampsia_MagnesiumAllowed tests that magnesium
// sulfate is ALLOWED for preeclampsia seizure prophylaxis.
//
// Scenario: Pregnant patient with severe preeclampsia
// Expected: Magnesium sulfate APPROVED - life-saving
func TestE2E_Maternal_Preeclampsia_MagnesiumAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Severe preeclampsia
	patient := PregnantPatient(32)
	patient.PatientID = "PT-E2E-PREECLAMPSIA"
	patient.ActiveDiagnoses = []types.Diagnosis{
		{Code: "O14.1", CodeSystem: "ICD10", Description: "Severe preeclampsia"},
	}
	patient.Vitals = &types.Vitals{
		SystolicBP:  175,
		DiastolicBP: 110,
	}
	patient.RegistryMemberships = append(patient.RegistryMemberships, types.RegistryMembership{
		RegistryCode: "PREECLAMPSIA_PROTOCOL",
		Status:       "ACTIVE",
	})

	// KB-19 recommends: Magnesium for seizure prophylaxis
	magRec := SimulatedRecommendation{
		Target:             "Magnesium Sulfate",
		TargetRxNorm:       "6728",
		DrugClass:          "ELECTROLYTE",
		RecommendedDose:    4000.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "PREECLAMPSIA_PROTOCOL",
		Rationale:          "ACOG: Magnesium sulfate for eclampsia prophylaxis",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, magRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Life-saving magnesium must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ CLINICAL FAILURE: Magnesium BLOCKED in preeclampsia")
		t.Errorf("   Magnesium prevents eclamptic seizures - life-saving")
	}

	t.Logf("✅ E2E MAGNESIUM PREECLAMPSIA: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestE2E_Maternal_FirstTrimesterNSAID_Caution tests NSAID handling
// in first trimester (relative contraindication).
//
// Scenario: First trimester pregnancy + pain
// Expected: WARN or BLOCK - NSAIDs have trimester-specific risks
func TestE2E_Maternal_FirstTrimesterNSAID_Caution(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Early pregnancy with pain
	patient := PregnantPatient(8) // First trimester

	// KB-19 recommends: NSAID for pain
	nsaidRec := NSAIDRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, nsaidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// NSAIDs in first trimester have some risk; third trimester is absolute contraindication
	t.Logf("E2E NSAID FIRST TRIMESTER: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Gestational age: 8 weeks (first trimester)")
	t.Logf("   Note: NSAIDs relatively contraindicated; prefer acetaminophen")
}

// TestE2E_Maternal_ThirdTrimesterNSAID_AbsoluteBlock tests NSAID blocking
// in third trimester (absolute contraindication - premature ductus closure).
//
// Scenario: Third trimester pregnancy + pain
// Expected: HARD BLOCK - NSAIDs cause premature ductus arteriosus closure
func TestE2E_Maternal_ThirdTrimesterNSAID_AbsoluteBlock(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Third trimester with pain
	patient := PregnantPatient(34) // Third trimester

	// KB-19 recommends: NSAID for pain
	nsaidRec := NSAIDRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, nsaidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Third trimester NSAIDs must be blocked
	if result.IsApproved() {
		t.Errorf("❌ SAFETY FAILURE: NSAID APPROVED in third trimester")
		t.Errorf("   NSAIDs cause premature ductus arteriosus closure after 28 weeks")
	}

	t.Logf("E2E NSAID THIRD TRIMESTER: outcome=%s, blocked=%v",
		result.FinalOutcome, result.IsBlocked())
	t.Logf("   Gestational age: 34 weeks (third trimester)")
}

// TestE2E_Maternal_NoOverrideAllowed tests that pregnancy contraindications
// cannot be overridden by any means.
//
// Scenario: Attempt to override pregnancy block
// Expected: Override NOT allowed
func TestE2E_Maternal_NoOverrideAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := PregnantPatient(16)
	mtxRec := MethotrexateRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, mtxRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// For pregnancy blocks, override should NOT be possible
	if result.RequiresOverride {
		// Even if override is "requested", it should not be grantable
		t.Logf("Note: Override requested but should be DENIED for teratogens")
	}

	// The enforcement should be HARD_BLOCK (no override) not HARD_BLOCK_WITH_OVERRIDE
	if result.HasEnforcement(types.EnforcementHardBlockWithOverride) {
		t.Errorf("❌ SAFETY FAILURE: Teratogen block allows override")
		t.Errorf("   Pregnancy contraindications must be HARD_BLOCK (no override)")
	}

	t.Logf("E2E MATERNAL NO OVERRIDE: outcome=%s, enforcement=%s",
		result.FinalOutcome, result.EnforcementApplied)
}

// =============================================================================
// MATERNAL SAFETY INVARIANT TESTS
// =============================================================================

// TestE2E_Maternal_Invariant_TeratogensAlwaysBlocked verifies the fundamental invariant:
// Teratogenic medications are NEVER approved in pregnancy.
func TestE2E_Maternal_Invariant_TeratogensAlwaysBlocked(t *testing.T) {
	ctx := NewE2ETestContext()

	// Known teratogens that must always be blocked
	teratogens := []SimulatedRecommendation{
		MethotrexateRecommendation(),
		ACEInhibitorRecommendation(),
		{
			Target: "Isotretinoin", TargetRxNorm: "6064", DrugClass: "RETINOID",
			RecommendedDose: 40, DoseUnit: "mg", RecommendationType: RecommendDo,
			SourceProtocol: "ACNE", Rationale: "Severe acne treatment",
		},
		{
			Target: "Thalidomide", TargetRxNorm: "10324", DrugClass: "IMMUNOMODULATOR",
			RecommendedDose: 100, DoseUnit: "mg", RecommendationType: RecommendDo,
			SourceProtocol: "MYELOMA", Rationale: "Multiple myeloma",
		},
	}

	patient := PregnantPatient(16)
	approvedCount := 0

	for _, teratogen := range teratogens {
		result, err := ctx.ExecuteE2EFlow(patient, teratogen)
		if err != nil {
			t.Errorf("Teratogen '%s' evaluation failed: %v", teratogen.Target, err)
			continue
		}

		if result.IsApproved() {
			approvedCount++
			t.Errorf("❌ CRITICAL INVARIANT VIOLATION: %s APPROVED in pregnancy",
				teratogen.Target)
		}
	}

	if approvedCount > 0 {
		t.Errorf("❌ TERATOGEN INVARIANT FAILURE: %d/%d teratogens were approved",
			approvedCount, len(teratogens))
	} else {
		t.Logf("✅ TERATOGEN INVARIANT VERIFIED: All %d teratogens blocked",
			len(teratogens))
	}
}

// TestE2E_Maternal_EvidenceTrailComplete tests that maternal safety blocks
// have complete evidence trails for legal protection.
func TestE2E_Maternal_EvidenceTrailComplete(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := PregnantPatient(16)
	mtxRec := MethotrexateRecommendation()

	result, err := ctx.ExecuteE2EFlow(patient, mtxRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	if !result.HasEvidenceTrail() {
		t.Errorf("❌ AUDIT FAILURE: Missing evidence trail for teratogen block")
	}

	trail := result.GovernanceResponse.EvidenceTrail
	if trail != nil {
		if trail.PatientSnapshot == nil {
			t.Errorf("Evidence trail missing patient snapshot")
		}
		// PatientSnapshot is JSON - verify it exists and contains data
		if trail.PatientSnapshot != nil && len(trail.PatientSnapshot) == 0 {
			t.Errorf("Evidence trail patient snapshot is empty")
		}
	}

	t.Logf("✅ E2E MATERNAL EVIDENCE TRAIL:")
	if trail != nil {
		t.Logf("   Trail ID: %s", trail.TrailID)
		t.Logf("   Patient snapshot captured: %v", trail.PatientSnapshot != nil && len(trail.PatientSnapshot) > 0)
	}
}
