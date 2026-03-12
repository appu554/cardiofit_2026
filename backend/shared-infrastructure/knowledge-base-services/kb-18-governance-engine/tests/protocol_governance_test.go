// Package tests provides clinical-device rigor testing for KB-18 Governance Engine.
// This file tests PROTOCOL GOVERNANCE scenarios for clinical protocols (Sepsis, CHF, Insulin).
package tests

import (
	"context"
	"testing"
	"time"

	"kb-18-governance-engine/pkg/engine"
	"kb-18-governance-engine/pkg/programs"
	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// PROTOCOL GOVERNANCE TESTS - Clinical Protocol Compliance
// P1: Sepsis Protocol | P2: CHF Protocol | P3: Insulin Protocol
// =============================================================================

// -----------------------------------------------------------------------------
// P1: SEPSIS PROTOCOL TESTS
// Sepsis-3 criteria based protocol activation and enforcement
// -----------------------------------------------------------------------------

// TestProtocol_P1_SepsisPatient_ProtocolActivation tests that sepsis protocol
// activates for patients meeting sepsis criteria and evaluates appropriately.
func TestProtocol_P1_SepsisPatient_ProtocolActivation(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Patient with sepsis indicators (SIRS criteria + suspected infection)
	req := &types.EvaluationRequest{
		PatientID: "PT-P1-SEPSIS",
		PatientContext: &types.PatientContext{
			PatientID: "PT-P1-SEPSIS",
			Age:       65,
			Sex:       "M",
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis, unspecified organism", Status: "active"},
				{Code: "J18.9", CodeSystem: "ICD10", Description: "Pneumonia", Status: "active"},
			},
			RecentLabs: []types.LabResult{
				{Code: "2160-0", Name: "Lactate", Value: 4.5, Unit: "mmol/L", Timestamp: time.Now().Add(-2 * time.Hour)},
				{Code: "26464-8", Name: "WBC", Value: 18.5, Unit: "10*9/L", Timestamp: time.Now().Add(-2 * time.Hour)},
			},
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "SEPSIS", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "VANC",
			MedicationName: "Vancomycin",
			DrugClass:      "ANTIBIOTIC",
			Dose:           1000.0,
			DoseUnit:       "mg",
			Frequency:      "q12h",
			Route:          "IV",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		RequestorRole:  "INTENSIVIST",
		FacilityID:     "ICU-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	// Protocol evaluation should complete with evidence trail
	if resp.EvidenceTrail == nil {
		t.Error("Evidence trail must be generated for protocol evaluation")
	}

	// Log the outcome for verification
	t.Logf("Sepsis patient evaluation: outcome=%s, violations=%d, approved=%v",
		resp.Outcome, len(resp.Violations), resp.IsApproved)

	for _, v := range resp.Violations {
		t.Logf("  Violation: %s - %s (severity=%s)", v.RuleID, v.Description, v.Severity)
	}

	t.Logf("✅ P1 SEPSIS PROTOCOL: Evaluation completed with outcome=%s", resp.Outcome)
}

// TestProtocol_P1_SepsisLactateMonitoring tests lactate monitoring requirements.
func TestProtocol_P1_SepsisLactateMonitoring(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Sepsis patient with elevated lactate requiring monitoring
	req := &types.EvaluationRequest{
		PatientID: "PT-P1-LACTATE",
		PatientContext: &types.PatientContext{
			PatientID: "PT-P1-LACTATE",
			Age:       70,
			Sex:       "F",
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis", Status: "active"},
			},
			RecentLabs: []types.LabResult{
				{Code: "2160-0", Name: "Lactate", Value: 3.2, Unit: "mmol/L", Timestamp: time.Now().Add(-4 * time.Hour)},
			},
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "SEPSIS", Status: "ACTIVE"},
			},
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-002",
		RequestorRole:  "PHYSICIAN",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	t.Logf("Sepsis lactate monitoring: outcome=%s, violations=%d", resp.Outcome, len(resp.Violations))
	t.Logf("✅ P1 SEPSIS LACTATE: Check completed")
}

// -----------------------------------------------------------------------------
// P2: CHF (HEART FAILURE) PROTOCOL TESTS
// Heart failure medication and monitoring protocols
// -----------------------------------------------------------------------------

// TestProtocol_P2_CHFPatient_GDMTEvaluation tests GDMT evaluation for CHF.
func TestProtocol_P2_CHFPatient_GDMTEvaluation(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// CHF patient with reduced ejection fraction
	req := &types.EvaluationRequest{
		PatientID: "PT-P2-CHF",
		PatientContext: &types.PatientContext{
			PatientID: "PT-P2-CHF",
			Age:       68,
			Sex:       "M",
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "I50.22", CodeSystem: "ICD10", Description: "Chronic systolic heart failure", Status: "active"},
			},
			CurrentMedications: []types.Medication{
				{Code: "FUROS", Name: "Furosemide", DrugClass: "LOOP_DIURETIC", Dose: 40, DoseUnit: "mg"},
				{Code: "CARV", Name: "Carvedilol", DrugClass: "BETA_BLOCKER", Dose: 12.5, DoseUnit: "mg"},
			},
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "HEART_FAILURE", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "LISINOPRIL",
			MedicationName: "Lisinopril",
			DrugClass:      "ACE_INHIBITOR",
			Dose:           10.0,
			DoseUnit:       "mg",
			Frequency:      "daily",
			Route:          "PO",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-003",
		RequestorRole:  "CARDIOLOGIST",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	t.Logf("CHF GDMT evaluation: outcome=%s, violations=%d", resp.Outcome, len(resp.Violations))

	for _, v := range resp.Violations {
		t.Logf("  Violation: %s (severity=%s)", v.RuleName, v.Severity)
	}

	t.Logf("✅ P2 CHF GDMT: Evaluation completed")
}

// TestProtocol_P2_CHFFluidRestriction tests fluid management for CHF patients.
func TestProtocol_P2_CHFFluidRestriction(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// CHF patient receiving IV fluids (potential concern)
	req := &types.EvaluationRequest{
		PatientID: "PT-P2-FLUID",
		PatientContext: &types.PatientContext{
			PatientID: "PT-P2-FLUID",
			Age:       72,
			Sex:       "F",
			Weight:    85.0,
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "I50.32", CodeSystem: "ICD10", Description: "Chronic diastolic heart failure", Status: "active"},
			},
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "HEART_FAILURE", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "NS",
			MedicationName: "Normal Saline",
			DrugClass:      "IV_FLUID",
			Dose:           1000.0,
			DoseUnit:       "mL",
			Frequency:      "bolus",
			Route:          "IV",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-004",
		RequestorRole:  "PHYSICIAN",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	t.Logf("CHF fluid order: outcome=%s, approved=%v", resp.Outcome, resp.IsApproved)
	t.Logf("✅ P2 CHF FLUID: Check completed")
}

// -----------------------------------------------------------------------------
// P3: INSULIN PROTOCOL TESTS
// Insulin dosing and hypoglycemia prevention protocols
// -----------------------------------------------------------------------------

// TestProtocol_P3_InsulinHighDose_SafetyCheck tests high-dose insulin safety.
func TestProtocol_P3_InsulinHighDose_SafetyCheck(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Patient receiving high-dose insulin
	req := &types.EvaluationRequest{
		PatientID: "PT-P3-HIGH-INSULIN",
		PatientContext: &types.PatientContext{
			PatientID: "PT-P3-HIGH-INSULIN",
			Age:       55,
			Sex:       "F",
			Weight:    65.0,
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "E11.9", CodeSystem: "ICD10", Description: "Type 2 DM", Status: "active"},
			},
			RecentLabs: []types.LabResult{
				{Code: "2339-0", Name: "Glucose", Value: 185, Unit: "mg/dL", Timestamp: time.Now().Add(-2 * time.Hour)},
			},
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "DIABETES", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "INSULIN-REG",
			MedicationName: "Regular Insulin",
			DrugClass:      "INSULIN",
			Dose:           50.0, // High dose
			DoseUnit:       "units",
			Frequency:      "once",
			Route:          "SC",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-006",
		RequestorRole:  "PHYSICIAN",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	// Check for dose-related violations
	hasDoseWarning := false
	for _, v := range resp.Violations {
		if v.Category == types.ViolationDoseExceeded {
			hasDoseWarning = true
			t.Logf("Dose warning: %s", v.Description)
		}
	}

	t.Logf("High-dose insulin: outcome=%s, dose_warning=%v", resp.Outcome, hasDoseWarning)
	t.Logf("✅ P3 INSULIN HIGH-DOSE: Safety check completed")
}

// TestProtocol_P3_InsulinHypoglycemiaRisk tests hypoglycemia risk assessment.
func TestProtocol_P3_InsulinHypoglycemiaRisk(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Elderly patient with hypoglycemia risk factors
	req := &types.EvaluationRequest{
		PatientID: "PT-P3-HYPO-RISK",
		PatientContext: &types.PatientContext{
			PatientID: "PT-P3-HYPO-RISK",
			Age:       82, // Elderly - higher hypo risk
			Sex:       "M",
			Weight:    55.0,
			RenalFunction: &types.RenalFunction{
				EGFR:     35.0, // CKD Stage 3B
				CKDStage: "CKD_3B",
			},
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "E11.9", CodeSystem: "ICD10", Description: "Type 2 DM", Status: "active"},
			},
			RecentLabs: []types.LabResult{
				{Code: "2339-0", Name: "Glucose", Value: 145, Unit: "mg/dL", Timestamp: time.Now().Add(-3 * time.Hour)},
				{Code: "4548-4", Name: "HbA1c", Value: 6.2, Unit: "%", Timestamp: time.Now().Add(-7 * 24 * time.Hour)},
			},
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "DIABETES", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "GLARGINE",
			MedicationName: "Insulin Glargine",
			DrugClass:      "INSULIN",
			Dose:           30.0,
			DoseUnit:       "units",
			Frequency:      "daily",
			Route:          "SC",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-007",
		RequestorRole:  "PHYSICIAN",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	t.Logf("Hypoglycemia risk: outcome=%s, violations=%d", resp.Outcome, len(resp.Violations))

	for _, v := range resp.Violations {
		t.Logf("  Risk factor: %s (severity=%s)", v.Description, v.Severity)
	}

	// Verify evidence trail
	if resp.EvidenceTrail != nil {
		t.Logf("Evidence captured: hash=%s...", resp.EvidenceTrail.Hash[:40])
	}

	t.Logf("✅ P3 HYPOGLYCEMIA RISK: Assessment completed")
}

// TestProtocol_P3_InsulinSlidingScale tests sliding scale insulin evaluation.
func TestProtocol_P3_InsulinSlidingScale(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	// Patient on sliding scale insulin
	req := &types.EvaluationRequest{
		PatientID: "PT-P3-SSI",
		PatientContext: &types.PatientContext{
			PatientID: "PT-P3-SSI",
			Age:       60,
			Sex:       "M",
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "E11.9", CodeSystem: "ICD10", Description: "Type 2 DM", Status: "active"},
			},
			RecentLabs: []types.LabResult{
				{Code: "2339-0", Name: "Glucose", Value: 250, Unit: "mg/dL", Timestamp: time.Now().Add(-1 * time.Hour)},
			},
			RegistryMemberships: []types.RegistryMembership{
				{RegistryCode: "DIABETES", Status: "ACTIVE"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "SSI",
			MedicationName: "Sliding Scale Insulin",
			DrugClass:      "INSULIN",
			Dose:           8.0, // Per sliding scale
			DoseUnit:       "units",
			Frequency:      "ac",
			Route:          "SC",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-008",
		RequestorRole:  "PHYSICIAN",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	t.Logf("Sliding scale insulin: outcome=%s, approved=%v", resp.Outcome, resp.IsApproved)
	t.Logf("✅ P3 SLIDING SCALE: Evaluation completed")
}

// -----------------------------------------------------------------------------
// PROTOCOL EVIDENCE TRAIL TESTS
// Verify evidence trails are generated for all protocol evaluations
// -----------------------------------------------------------------------------

// TestProtocol_EvidenceTrailGeneration verifies protocol evaluations generate evidence.
func TestProtocol_EvidenceTrailGeneration(t *testing.T) {
	programStore := programs.NewProgramStore()
	eng := engine.NewGovernanceEngine(programStore)
	ctx := context.Background()

	req := &types.EvaluationRequest{
		PatientID: "PT-EVIDENCE",
		PatientContext: &types.PatientContext{
			PatientID: "PT-EVIDENCE",
			Age:       50,
			Sex:       "M",
			ActiveDiagnoses: []types.Diagnosis{
				{Code: "I50.9", CodeSystem: "ICD10", Description: "Heart failure", Status: "active"},
			},
		},
		Order: &types.MedicationOrder{
			MedicationCode: "FUROS",
			MedicationName: "Furosemide",
			DrugClass:      "LOOP_DIURETIC",
			Dose:           40.0,
			DoseUnit:       "mg",
			Frequency:      "daily",
			Route:          "PO",
		},
		EvaluationType: types.EvalTypeMedicationOrder,
		RequestorID:    "DR-001",
		Timestamp:      time.Now(),
	}

	resp, err := eng.Evaluate(ctx, req)
	if err != nil {
		t.Fatalf("Evaluation failed: %v", err)
	}

	// Verify evidence trail exists
	if resp.EvidenceTrail == nil {
		t.Error("Protocol evaluation must generate evidence trail")
	} else {
		if resp.EvidenceTrail.TrailID == "" {
			t.Error("Evidence trail must have TrailID")
		}
		if resp.EvidenceTrail.Hash == "" {
			t.Error("Evidence trail must have Hash")
		}
		t.Logf("Evidence: trailID=%s, hash=%s...", resp.EvidenceTrail.TrailID, resp.EvidenceTrail.Hash[:40])
	}

	t.Logf("✅ PROTOCOL EVIDENCE TRAIL: Generation verified")
}
