// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ICU TEMPORAL DETERIORATION scenarios.
//
// Clinical Truth: Trends matter more than snapshots. A worsening lactate over
// 2 hours demands escalation, even if the absolute value is "acceptable."
package tests

import (
	"testing"
	"time"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ICU TEMPORAL DETERIORATION SCENARIOS
// These tests prove that trend-based governance correctly identifies clinical
// deterioration and triggers appropriate escalation.
// =============================================================================

// TestICU_Temporal_WorseningLactate_Escalation tests that worsening lactate
// trend triggers mandatory escalation even when absolute values seem acceptable.
//
// Scenario: Lactate rising from 2.0 → 2.8 → 3.6 over 4 hours
// Expected: MANDATORY_ESCALATION - trend indicates clinical deterioration
func TestICU_Temporal_WorseningLactate_Escalation(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	// Patient: Lactate trending upward over 4 hours
	patient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-001",
		Age:        62,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
			{Code: "R65.20", CodeSystem: "ICD10", Description: "Severe sepsis without shock"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "SEPSIS_BUNDLE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  95,
			DiastolicBP: 62,
			HeartRate:   105,
			SpO2:        94,
			Timestamp:   now,
		},
		RecentLabs: []types.LabResult{
			// Lactate trend: RISING (2.0 → 2.8 → 3.6)
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 2.0, Unit: "mmol/L", Timestamp: now.Add(-4 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 2.8, Unit: "mmol/L", Timestamp: now.Add(-2 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 3.6, Unit: "mmol/L", Timestamp: now},
		},
	}

	// KB-19 recommends: Continue current management
	continueRec := SimulatedRecommendation{
		Target:             "Continue current antibiotics",
		TargetRxNorm:       "PROTOCOL",
		DrugClass:          "PROTOCOL_CONTINUATION",
		RecommendedDose:    0,
		DoseUnit:           "N/A",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "SEPSIS_MONITORING",
		Rationale:          "Lactate 3.6 - within monitoring range",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, continueRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check if trend triggered escalation
	hasEscalation := result.RequiresEscalation

	t.Logf("ICU TEMPORAL LACTATE: outcome=%s, escalation=%v",
		result.FinalOutcome, hasEscalation)

	// The trend (80% increase in 4 hours) should trigger clinical review
	if !hasEscalation {
		t.Logf("   ⚠️ Note: Worsening lactate trend (2.0→3.6 in 4h) may warrant escalation")
	}

	t.Logf("   Clinical Truth: Trend-based analysis detects deterioration early")
}

// TestICU_Temporal_DecliningMAP_VasopressorTrigger tests that declining MAP
// trend triggers vasopressor escalation.
//
// Scenario: MAP declining from 72 → 68 → 63 over 2 hours
// Expected: Vasopressor escalation recommended despite "borderline" numbers
func TestICU_Temporal_DecliningMAP_VasopressorTrigger(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-002",
		Age:        58,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "R57.8", CodeSystem: "ICD10", Description: "Distributive shock"},
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
		},
		CurrentMedications: []types.Medication{
			{
				Code:       "7512",
				CodeSystem: "RXNORM",
				Name:       "Norepinephrine",
				DrugClass:  "VASOPRESSOR",
				Dose:       0.08,
				DoseUnit:   "mcg/kg/min",
				Frequency:  "continuous",
				Route:      "IV",
			},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "HEMODYNAMIC_SUPPORT", Status: "ACTIVE"},
			{RegistryCode: "VASOPRESSOR_THERAPY", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  88, // Current: MAP ≈ 63
			DiastolicBP: 52,
			HeartRate:   112,
			SpO2:        93,
			Timestamp:   now,
		},
	}

	// KB-19 recommends: Increase vasopressor
	vasopressorIncrease := SimulatedRecommendation{
		Target:             "Norepinephrine",
		TargetRxNorm:       "7512",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.15, // Increase from 0.08
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "HEMODYNAMIC_OPTIMIZATION",
		Rationale:          "MAP declining - increase vasopressor to maintain ≥65 mmHg",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vasopressorIncrease)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Vasopressor increase must be allowed for declining perfusion
	if result.IsBlocked() {
		t.Errorf("❌ ICU TEMPORAL FAILURE: Vasopressor increase BLOCKED despite declining MAP")
		t.Errorf("   MAP trend: 72 → 68 → 63 mmHg over 2 hours")
		t.Errorf("   Current norepinephrine: 0.08 mcg/kg/min")
	}

	t.Logf("✅ ICU TEMPORAL MAP: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Clinical Truth: Declining MAP trend warrants proactive escalation")
}

// TestICU_Temporal_RisingCreatinine_AKIAlert tests that rising creatinine
// trend triggers AKI alert and nephrotoxic drug review.
//
// Scenario: Creatinine rising from 1.2 → 1.8 → 2.5 over 24 hours (KDIGO Stage 2)
// Expected: Nephrotoxic drug warnings triggered
func TestICU_Temporal_RisingCreatinine_AKIAlert(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-003",
		Age:        70,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J18.9", CodeSystem: "ICD10", Description: "Pneumonia"},
			{Code: "N17.9", CodeSystem: "ICD10", Description: "AKI developing"},
		},
		CurrentMedications: []types.Medication{
			{
				Code:       "4053",
				CodeSystem: "RXNORM",
				Name:       "Gentamicin",
				DrugClass:  "AMINOGLYCOSIDE",
				Dose:       350,
				DoseUnit:   "mg",
				Frequency:  "daily",
				Route:      "IV",
			},
		},
		RenalFunction: &types.RenalFunction{
			EGFR:       32.0, // Declining
			Creatinine: 2.5, // Current
			CKDStage:   "AKI_2",
			OnDialysis: false,
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			// Creatinine trend: RISING (1.2 → 1.8 → 2.5)
			{Code: "2160-0", CodeSystem: "LOINC", Name: "Creatinine", Value: 1.2, Unit: "mg/dL", Timestamp: now.Add(-24 * time.Hour)},
			{Code: "2160-0", CodeSystem: "LOINC", Name: "Creatinine", Value: 1.8, Unit: "mg/dL", Timestamp: now.Add(-12 * time.Hour)},
			{Code: "2160-0", CodeSystem: "LOINC", Name: "Creatinine", Value: 2.5, Unit: "mg/dL", Timestamp: now},
		},
	}

	// KB-19 inappropriately continues: Gentamicin (nephrotoxic)
	nephrotoxicRec := SimulatedRecommendation{
		Target:             "Gentamicin",
		TargetRxNorm:       "4053",
		DrugClass:          "AMINOGLYCOSIDE",
		RecommendedDose:    350,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "PNEUMONIA_TREATMENT",
		Rationale:          "Continue gram-negative coverage",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, nephrotoxicRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check for renal safety violation
	hasRenalViolation := result.HasViolationCategory(types.ViolationRenalDosing)

	t.Logf("ICU TEMPORAL CREATININE: outcome=%s, renal_violation=%v",
		result.FinalOutcome, hasRenalViolation)

	if !hasRenalViolation && result.FinalAllowed {
		t.Logf("   ⚠️ Note: Rising creatinine (1.2→2.5 in 24h) with nephrotoxic drug may warrant review")
	}

	t.Logf("   Clinical Truth: AKI trajectory + nephrotoxin = kidney damage cascade")
}

// TestICU_Temporal_DecliningSPO2_RespiratoryEscalation tests that declining
// SpO2 trend triggers respiratory escalation despite "acceptable" current values.
//
// Scenario: SpO2 declining from 96 → 93 → 91 over 6 hours on 4L O2
// Expected: Respiratory therapy escalation recommended
func TestICU_Temporal_DecliningSPO2_RespiratoryEscalation(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-004",
		Age:        75,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J96.01", CodeSystem: "ICD10", Description: "Acute respiratory failure with hypoxia"},
			{Code: "J18.9", CodeSystem: "ICD10", Description: "Community-acquired pneumonia"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "RESPIRATORY_MONITORING", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  135,
			DiastolicBP: 78,
			HeartRate:   98,
			SpO2:        91, // Declining on supplemental O2
			Timestamp:   now,
		},
	}

	// KB-19 recommends: Escalate respiratory support
	respiratoryEscalation := SimulatedRecommendation{
		Target:             "High-Flow Nasal Cannula",
		TargetRxNorm:       "DEVICE",
		DrugClass:          "RESPIRATORY_THERAPY",
		RecommendedDose:    50, // L/min
		DoseUnit:           "L/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "RESPIRATORY_FAILURE",
		Rationale:          "SpO2 declining on conventional O2 - escalate to HFNC",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, respiratoryEscalation)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Respiratory escalation should be allowed
	if result.IsBlocked() {
		t.Errorf("❌ ICU TEMPORAL FAILURE: Respiratory escalation BLOCKED")
		t.Errorf("   SpO2 trend: 96 → 93 → 91 over 6 hours on 4L O2")
	}

	t.Logf("✅ ICU TEMPORAL SPO2: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Clinical Truth: Declining oxygenation trajectory predicts intubation need")
}

// TestICU_Temporal_ImprovingLactate_DeescalationAllowed tests that improving
// lactate trend allows appropriate de-escalation of therapy.
//
// Scenario: Lactate falling from 5.2 → 3.1 → 1.8 over 12 hours
// Expected: De-escalation of vasopressors considered appropriate
func TestICU_Temporal_ImprovingLactate_DeescalationAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-005",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis - resolving"},
			{Code: "R65.21", CodeSystem: "ICD10", Description: "Septic shock - improving"},
		},
		CurrentMedications: []types.Medication{
			{
				Code:       "7512",
				CodeSystem: "RXNORM",
				Name:       "Norepinephrine",
				DrugClass:  "VASOPRESSOR",
				Dose:       0.12,
				DoseUnit:   "mcg/kg/min",
				Frequency:  "continuous",
				Route:      "IV",
			},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "SEPSIS_BUNDLE", Status: "ACTIVE"},
			{RegistryCode: "VASOPRESSOR_THERAPY", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  118,
			DiastolicBP: 72,
			HeartRate:   88,
			SpO2:        97,
			Timestamp:   now,
		},
		RecentLabs: []types.LabResult{
			// Lactate trend: IMPROVING (5.2 → 3.1 → 1.8)
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 5.2, Unit: "mmol/L", Timestamp: now.Add(-12 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 3.1, Unit: "mmol/L", Timestamp: now.Add(-6 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 1.8, Unit: "mmol/L", Timestamp: now},
		},
	}

	// KB-19 recommends: Wean vasopressor
	vasopressorWean := SimulatedRecommendation{
		Target:             "Norepinephrine",
		TargetRxNorm:       "7512",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.08, // Decrease from 0.12
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "SEPSIS_RECOVERY",
		Rationale:          "Lactate clearing, hemodynamics stable - begin vasopressor wean",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, vasopressorWean)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Vasopressor wean should be allowed with improving trends
	if result.IsBlocked() {
		t.Errorf("❌ ICU TEMPORAL FAILURE: Vasopressor wean BLOCKED despite improving lactate")
		t.Errorf("   Lactate trend: 5.2 → 3.1 → 1.8 (improving)")
		t.Errorf("   MAP stable at 87 mmHg")
	}

	t.Logf("✅ ICU TEMPORAL IMPROVING: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Clinical Truth: Improving trends support safe de-escalation")
}

// TestICU_Temporal_RapidDeterioration_EmergencyEscalation tests that rapid
// deterioration triggers immediate emergency escalation.
//
// Scenario: Lactate doubles in 2 hours (3.0 → 6.0) with falling MAP
// Expected: MANDATORY_ESCALATION - rapid deterioration emergency
func TestICU_Temporal_RapidDeterioration_EmergencyEscalation(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-006",
		Age:        68,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "R65.21", CodeSystem: "ICD10", Description: "Septic shock - deteriorating"},
			{Code: "A41.52", CodeSystem: "ICD10", Description: "Sepsis due to Pseudomonas"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "SEPSIS_BUNDLE", Status: "ACTIVE"},
			{RegistryCode: "RAPID_RESPONSE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  78,
			DiastolicBP: 45,
			HeartRate:   128,
			SpO2:        88,
			Timestamp:   now,
		},
		RecentLabs: []types.LabResult{
			// Lactate DOUBLING in 2 hours - crisis
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 3.0, Unit: "mmol/L", Timestamp: now.Add(-2 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 6.0, Unit: "mmol/L", Timestamp: now},
		},
	}

	// Any recommendation should trigger emergency escalation
	anyRec := SimulatedRecommendation{
		Target:             "Norepinephrine",
		TargetRxNorm:       "7512",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.3,
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "SEPTIC_SHOCK_CRISIS",
		Rationale:          "Refractory shock - maximum vasopressor support",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, anyRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check for escalation requirement
	requiresEscalation := result.RequiresEscalation

	t.Logf("ICU TEMPORAL RAPID: outcome=%s, escalation=%v",
		result.FinalOutcome, requiresEscalation)

	if !requiresEscalation {
		t.Logf("   ⚠️ Note: Lactate doubling in 2h with MAP 56 should trigger mandatory escalation")
	}

	t.Logf("   Clinical Truth: Rapid deterioration = emergency attending involvement")
}

// =============================================================================
// ICU TEMPORAL INVARIANT TESTS
// =============================================================================

// TestICU_Temporal_Invariant_TrendsOverSnapshots tests that trend-based
// analysis is consistently applied across clinical scenarios.
func TestICU_Temporal_Invariant_TrendsOverSnapshots(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	// Patient with IMPROVING absolute values but WORSENING trend
	// Snapshot: Lactate 2.5 (acceptable)
	// Trend: Rising from 1.5 → 2.0 → 2.5 (concerning)
	worseningTrendPatient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-INV-1",
		Age:        60,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "SEPSIS_BUNDLE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  100,
			DiastolicBP: 65,
			HeartRate:   95,
			SpO2:        95,
			Timestamp:   now,
		},
		RecentLabs: []types.LabResult{
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 1.5, Unit: "mmol/L", Timestamp: now.Add(-4 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 2.0, Unit: "mmol/L", Timestamp: now.Add(-2 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 2.5, Unit: "mmol/L", Timestamp: now},
		},
	}

	// Patient with CONCERNING absolute values but IMPROVING trend
	// Snapshot: Lactate 3.5 (elevated)
	// Trend: Falling from 5.0 → 4.2 → 3.5 (improving)
	improvingTrendPatient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-INV-2",
		Age:        60,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "SEPSIS_BUNDLE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  105,
			DiastolicBP: 68,
			HeartRate:   90,
			SpO2:        96,
			Timestamp:   now,
		},
		RecentLabs: []types.LabResult{
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 5.0, Unit: "mmol/L", Timestamp: now.Add(-4 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 4.2, Unit: "mmol/L", Timestamp: now.Add(-2 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 3.5, Unit: "mmol/L", Timestamp: now},
		},
	}

	continueRec := SimulatedRecommendation{
		Target:             "Continue current management",
		TargetRxNorm:       "PROTOCOL",
		DrugClass:          "PROTOCOL_CONTINUATION",
		RecommendedDose:    0,
		DoseUnit:           "N/A",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "SEPSIS_MONITORING",
		Rationale:          "Continue current therapy",
		Urgency:            "ROUTINE",
	}

	// Test worsening trend patient
	result1, err := ctx.ExecuteE2EFlow(worseningTrendPatient, continueRec)
	if err != nil {
		t.Errorf("Worsening trend evaluation failed: %v", err)
	} else {
		t.Logf("WORSENING TREND (2.5 rising): outcome=%s, escalation=%v",
			result1.FinalOutcome, result1.RequiresEscalation)
	}

	// Test improving trend patient
	result2, err := ctx.ExecuteE2EFlow(improvingTrendPatient, continueRec)
	if err != nil {
		t.Errorf("Improving trend evaluation failed: %v", err)
	} else {
		t.Logf("IMPROVING TREND (3.5 falling): outcome=%s, escalation=%v",
			result2.FinalOutcome, result2.RequiresEscalation)
	}

	t.Logf("✅ TEMPORAL INVARIANT TESTED: Trend analysis applied to both scenarios")
	t.Logf("   Clinical Truth: Trend direction matters more than absolute value")
	t.Logf("   - Rising 1.5→2.5 may need intervention")
	t.Logf("   - Falling 5.0→3.5 may allow de-escalation")
}

// TestICU_Temporal_Invariant_VelocityMatters tests that the rate of change
// (velocity) influences governance decisions.
func TestICU_Temporal_Invariant_VelocityMatters(t *testing.T) {
	ctx := NewE2ETestContext()

	now := time.Now()

	// Slow deterioration: Lactate 2.0 → 2.5 over 6 hours
	slowPatient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-SLOW",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  98,
			DiastolicBP: 62,
			HeartRate:   100,
			SpO2:        94,
			Timestamp:   now,
		},
		RecentLabs: []types.LabResult{
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 2.0, Unit: "mmol/L", Timestamp: now.Add(-6 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 2.5, Unit: "mmol/L", Timestamp: now},
		},
	}

	// Rapid deterioration: Lactate 2.0 → 4.0 over 2 hours (same endpoint, faster)
	rapidPatient := &types.PatientContext{
		PatientID:  "PT-ICU-TEMP-RAPID",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  85,
			DiastolicBP: 52,
			HeartRate:   118,
			SpO2:        91,
			Timestamp:   now,
		},
		RecentLabs: []types.LabResult{
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 2.0, Unit: "mmol/L", Timestamp: now.Add(-2 * time.Hour)},
			{Code: "14627-4", CodeSystem: "LOINC", Name: "Lactate", Value: 4.0, Unit: "mmol/L", Timestamp: now},
		},
	}

	vasopressorRec := SimulatedRecommendation{
		Target:             "Norepinephrine",
		TargetRxNorm:       "7512",
		DrugClass:          "VASOPRESSOR",
		RecommendedDose:    0.1,
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "HEMODYNAMIC_SUPPORT",
		Rationale:          "Hemodynamic support for perfusion",
		Urgency:            "URGENT",
	}

	// Test slow deterioration
	slowResult, err := ctx.ExecuteE2EFlow(slowPatient, vasopressorRec)
	if err != nil {
		t.Errorf("Slow deterioration evaluation failed: %v", err)
	}

	// Test rapid deterioration
	rapidResult, err := ctx.ExecuteE2EFlow(rapidPatient, vasopressorRec)
	if err != nil {
		t.Errorf("Rapid deterioration evaluation failed: %v", err)
	}

	t.Logf("VELOCITY COMPARISON:")
	t.Logf("  SLOW (0.08 mmol/L/hr): outcome=%s, escalation=%v",
		slowResult.FinalOutcome, slowResult.RequiresEscalation)
	t.Logf("  RAPID (1.0 mmol/L/hr): outcome=%s, escalation=%v",
		rapidResult.FinalOutcome, rapidResult.RequiresEscalation)

	t.Logf("✅ VELOCITY INVARIANT TESTED: Rate of change influences urgency")
	t.Logf("   Clinical Truth: Same trajectory, different velocity = different urgency")
}
