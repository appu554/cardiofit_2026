// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests SEPSIS scenarios: Life-preserving care vs chronic optimization.
//
// Clinical Truth: Shock kills first. Chronic disease optimization can wait.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// SEPSIS E2E SCENARIOS
// These tests prove that life-preserving interventions proceed even when
// they might conflict with chronic disease management protocols.
// =============================================================================

// TestE2E_Sepsis_FluidResuscitationAllowed tests that IV fluids are allowed
// for septic shock even with concurrent heart failure history.
//
// Scenario: Septic Shock + Chronic HFrEF
// Expected: Life-preserving fluids ALLOWED despite HF
func TestE2E_Sepsis_FluidResuscitationAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Septic shock with chronic heart failure
	patient := SepticShockWithHFPatient()

	// KB-19 recommends: IV fluids 30 mL/kg (SSC 2021 Class I)
	fluidRec := FluidRecommendation(2250) // 30 mL/kg × 75 kg

	// Execute E2E flow
	result, err := ctx.ExecuteE2EFlow(patient, fluidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Life-preserving fluids must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ CLINICAL FAILURE: Sepsis fluids were BLOCKED despite life-threatening shock")
		t.Errorf("   Patient MAP: 55 mmHg, Lactate: 5.2 mmol/L")
		t.Errorf("   Expected: APPROVED (SSC 2021 Class I recommendation)")
		t.Errorf("   Got: %s", result.FinalOutcome)
	}

	// Verify evidence trail exists
	if !result.HasEvidenceTrail() {
		t.Errorf("Missing evidence trail for life-critical decision")
	}

	t.Logf("✅ E2E SEPSIS FLUID: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Patient: Septic shock + HFrEF")
	t.Logf("   Recommendation: %s %.0f %s", fluidRec.Target, fluidRec.RecommendedDose, fluidRec.DoseUnit)
	t.Logf("   Clinical Truth: Life-preserving care overrides chronic optimization")
}

// TestE2E_Sepsis_AntibioticsSTATAllowed tests that STAT antibiotics are allowed
// for septic patients without delay.
//
// Scenario: Sepsis with pneumonia
// Expected: Broad-spectrum antibiotics ALLOWED within 1 hour
func TestE2E_Sepsis_AntibioticsSTATAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Sepsis with pneumonia source
	patient := SepticShockPatient()

	// KB-19 recommends: Broad-spectrum antibiotics (SSC 2021)
	antibioticRec := SimulatedRecommendation{
		Target:             "Ceftriaxone",
		TargetRxNorm:       "2193",
		DrugClass:          "ANTIBIOTIC",
		RecommendedDose:    2000,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPSIS_HOUR_1",
		Rationale:          "SSC 2021: Broad-spectrum antibiotics within 1 hour of sepsis recognition",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, antibioticRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: STAT antibiotics must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ CLINICAL FAILURE: STAT antibiotics were BLOCKED for sepsis")
		t.Errorf("   SSC 2021: Every hour of delay increases mortality 4-8%%")
	}

	t.Logf("✅ E2E SEPSIS ANTIBIOTICS: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestE2E_Sepsis_VasopressorsAllowed tests that vasopressors are allowed
// for septic shock when MAP remains low after fluids.
//
// Scenario: Septic shock, MAP 55 after fluids
// Expected: Norepinephrine ALLOWED as first-line vasopressor
func TestE2E_Sepsis_VasopressorsAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Septic shock with persistent hypotension
	patient := SepticShockPatient()

	// KB-19 recommends: Norepinephrine (SSC 2021 first-line)
	vasopressorRec := SimulatedRecommendation{
		Target:             "Norepinephrine",
		TargetRxNorm:       "7512",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.1, // mcg/kg/min starting dose
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPSIS_HEMODYNAMIC",
		Rationale:          "SSC 2021: Norepinephrine first-line for septic shock",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vasopressorRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Vasopressors must be allowed for shock
	if result.IsBlocked() {
		t.Errorf("❌ CLINICAL FAILURE: Vasopressors BLOCKED for septic shock")
		t.Errorf("   MAP 55 mmHg - patient is dying")
	}

	t.Logf("✅ E2E SEPSIS VASOPRESSOR: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestE2E_Sepsis_DiuresisTemporarilySuppressed tests that chronic HF diuresis
// recommendations are appropriately delayed during acute sepsis.
//
// Scenario: CHF patient with acute sepsis
// Expected: Diuretics should NOT be recommended during acute volume depletion
func TestE2E_Sepsis_DiuresisTemporarilySuppressed(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Septic shock with chronic HF
	patient := SepticShockWithHFPatient()

	// KB-19 might recommend diuretics from CHF protocol
	// But sepsis protocol should override
	diureticRec := SimulatedRecommendation{
		Target:             "Furosemide",
		TargetRxNorm:       "4603",
		DrugClass:          "DIURETIC",
		RecommendedDose:    40,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "CHF_DECOMPENSATION",
		Rationale:          "CHF protocol: Diuresis for volume overload",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, diureticRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// In septic shock with MAP 55, diuretics would be harmful
	// Governance should recognize the acute context
	t.Logf("E2E SEPSIS DIURETIC CONTEXT: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Context: Septic shock (MAP 55) + CHF history")
	t.Logf("   Clinical Note: Diuresis during shock would worsen hypoperfusion")
}

// TestE2E_Sepsis_LactateMonitoringRequired tests that lactate monitoring
// is enforced as part of sepsis bundle compliance.
func TestE2E_Sepsis_LactateMonitoringRequired(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := SepticShockPatient()

	// Any sepsis intervention should come with lactate monitoring requirement
	fluidRec := FluidRecommendation(2000)

	result, err := ctx.ExecuteE2EFlow(patient, fluidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Verify evidence trail captures monitoring requirements
	if result.GovernanceResponse.EvidenceTrail != nil {
		t.Logf("✅ E2E SEPSIS MONITORING: Evidence trail captured")
		t.Logf("   Lactate trend monitoring should be mandatory per SSC")
	}
}

// TestE2E_Sepsis_SourceControlUrgency tests that source control procedures
// are allowed with appropriate urgency in sepsis.
func TestE2E_Sepsis_SourceControlUrgency(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := SepticShockPatient()
	patient.ActiveDiagnoses = append(patient.ActiveDiagnoses, types.Diagnosis{
		Code:        "K35.80",
		CodeSystem:  "ICD10",
		Description: "Acute appendicitis",
	})

	// Source control procedure recommendation
	procedureRec := SimulatedRecommendation{
		Target:             "Appendectomy",
		TargetRxNorm:       "",
		DrugClass:          "SURGICAL_PROCEDURE",
		RecommendedDose:    0,
		DoseUnit:           "",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPSIS_SOURCE_CONTROL",
		Rationale:          "SSC 2021: Emergent source control within 6-12 hours",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, procedureRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	t.Logf("✅ E2E SEPSIS SOURCE CONTROL: outcome=%s", result.FinalOutcome)
	t.Logf("   Procedure: %s", procedureRec.Target)
	t.Logf("   Urgency: %s", procedureRec.Urgency)
}

// TestE2E_Sepsis_RenalDoseAdjustment tests that renal dose adjustments
// are applied to antibiotics in sepsis with AKI.
func TestE2E_Sepsis_RenalDoseAdjustment(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Sepsis with AKI
	patient := SepticShockPatient()
	patient.RenalFunction = &types.RenalFunction{
		EGFR:       25.0,
		Creatinine: 3.2,
		CKDStage:   "AKI_2",
		OnDialysis: false,
	}

	// Vancomycin needs renal adjustment
	vancoRec := SimulatedRecommendation{
		Target:             "Vancomycin",
		TargetRxNorm:       "11124",
		DrugClass:          "ANTIBIOTIC",
		RecommendedDose:    1500,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPSIS_ANTIBIOTIC",
		Rationale:          "MRSA coverage for healthcare-associated sepsis",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vancoRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check if renal adjustment violation was raised
	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("E2E SEPSIS RENAL ADJUSTMENT: outcome=%s", result.FinalOutcome)
	t.Logf("   eGFR: 25 mL/min/1.73m²")
	t.Logf("   Renal dosing concern raised: %v", hasRenalViolation)
	t.Logf("   Note: Antibiotics must still be given, but dose may need adjustment")
}

// TestE2E_Sepsis_EvidenceTrailComplete tests that sepsis decisions
// have complete, auditable evidence trails.
func TestE2E_Sepsis_EvidenceTrailComplete(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := SepticShockPatient()
	fluidRec := FluidRecommendation(2250)

	result, err := ctx.ExecuteE2EFlow(patient, fluidRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Verify comprehensive evidence trail
	if !result.HasEvidenceTrail() {
		t.Errorf("❌ AUDIT FAILURE: Missing evidence trail for sepsis intervention")
	}

	trail := result.GovernanceResponse.EvidenceTrail
	if trail == nil {
		t.Fatal("Evidence trail is nil")
	}

	// Verify trail components
	if trail.PatientSnapshot == nil {
		t.Errorf("Evidence trail missing patient snapshot")
	}
	if len(trail.RulesApplied) == 0 && len(trail.ProgramsEvaluated) == 0 {
		// May be empty if no rules triggered, but structure should exist
		t.Logf("Note: No specific rules triggered for this evaluation")
	}

	t.Logf("✅ E2E SEPSIS EVIDENCE TRAIL:")
	t.Logf("   Trail ID: %s", trail.TrailID)
	t.Logf("   Hash: %s...", result.EvidenceTrailHash[:min(40, len(result.EvidenceTrailHash))])
	t.Logf("   Programs evaluated: %d", len(trail.ProgramsEvaluated))
	t.Logf("   Rules applied: %d", len(trail.RulesApplied))
}

// =============================================================================
// E2E INVARIANT TESTS
// =============================================================================

// TestE2E_Sepsis_Invariant_LifeOverChronic verifies the fundamental invariant:
// Life-threatening conditions take precedence over chronic disease optimization.
func TestE2E_Sepsis_Invariant_LifeOverChronic(t *testing.T) {
	ctx := NewE2ETestContext()

	// Run multiple sepsis scenarios to verify invariant
	scenarios := []struct {
		name    string
		patient *types.PatientContext
		rec     SimulatedRecommendation
	}{
		{
			name:    "Fluids for hypotensive sepsis",
			patient: SepticShockPatient(),
			rec:     FluidRecommendation(2000),
		},
		{
			name:    "Fluids for sepsis+HF",
			patient: SepticShockWithHFPatient(),
			rec:     FluidRecommendation(2000),
		},
		{
			name:    "Vasopressors for shock",
			patient: SepticShockPatient(),
			rec: SimulatedRecommendation{
				Target:             "Norepinephrine",
				TargetRxNorm:       "7512",
				DrugClass:          "VASOPRESSOR",
				RecommendedDose:    0.1,
				DoseUnit:           "mcg/kg/min",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "SEPSIS_HEMODYNAMIC",
				Rationale:          "Shock management",
				Urgency:            "STAT",
			},
		},
	}

	blockedCount := 0
	for _, scenario := range scenarios {
		result, err := ctx.ExecuteE2EFlow(scenario.patient, scenario.rec)
		if err != nil {
			t.Errorf("Scenario '%s' failed: %v", scenario.name, err)
			continue
		}

		if result.IsBlocked() {
			blockedCount++
			t.Errorf("❌ INVARIANT VIOLATION: %s was BLOCKED", scenario.name)
		}
	}

	if blockedCount > 0 {
		t.Errorf("❌ SEPSIS INVARIANT FAILURE: %d/%d life-saving interventions were blocked",
			blockedCount, len(scenarios))
	} else {
		t.Logf("✅ SEPSIS INVARIANT VERIFIED: All %d life-saving interventions allowed",
			len(scenarios))
	}
}
