// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ICU NEUROLOGICAL CRITICAL scenarios.
//
// Clinical Truth: Brain protection overrides everything - hemorrhagic stroke
// with anticoagulation is a catastrophic iatrogenic harm.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ICU NEUROLOGICAL CRITICAL SCENARIOS
// These tests prove that brain-protective measures are prioritized absolutely
// in neurological emergencies, especially intracranial hemorrhage.
// =============================================================================

// TestICU_Neuro_ICH_AnticoagBlocked tests that anticoagulation is absolutely
// blocked in patients with intracranial hemorrhage, regardless of other indications.
//
// Scenario: ICH patient with AFib (high stroke risk)
// Expected: Anticoagulation HARD_BLOCK - brain protection absolute
func TestICU_Neuro_ICH_AnticoagBlocked(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: ICH + AFib (CHA2DS2-VASc would normally indicate anticoag)
	patient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-001",
		Age:        72,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I61.9", CodeSystem: "ICD10", Description: "Intracerebral hemorrhage"},
			{Code: "I48.0", CodeSystem: "ICD10", Description: "Paroxysmal atrial fibrillation"},
			{Code: "I10", CodeSystem: "ICD10", Description: "Essential hypertension"},
			{Code: "E11.9", CodeSystem: "ICD10", Description: "Type 2 diabetes"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "STROKE_UNIT", Status: "ACTIVE"},
			{RegistryCode: "INTRACRANIAL_HEMORRHAGE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  165,
			DiastolicBP: 95,
			HeartRate:   88,
			SpO2:        97,
		},
		RecentLabs: []types.LabResult{
			{Code: "6301-6", CodeSystem: "LOINC", Name: "INR", Value: 1.1, Unit: "ratio"},
			{Code: "3255-7", CodeSystem: "LOINC", Name: "Fibrinogen", Value: 320, Unit: "mg/dL"},
		},
	}

	// KB-19 inappropriately recommends: Anticoagulation for AFib stroke prevention
	anticoagRec := AnticoagulationRecommendation("Apixaban", "1364430", 5.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Anticoag MUST be blocked - ICH is absolute contraindication
	if result.FinalAllowed {
		t.Errorf("❌ ICU NEURO SAFETY FAILURE: Anticoag ALLOWED in active ICH")
		t.Errorf("   CHA2DS2-VASc = 5 (high stroke risk) BUT ACTIVE BRAIN BLEED")
		t.Errorf("   Outcome: %s", result.FinalOutcome)
		t.Errorf("   Expected: HARD_BLOCK (brain protection absolute)")
	}

	// Check for ICH-specific violation (contraindication category)
	hasBleedingViolation := result.HasViolationCategory(types.ViolationContraindication)

	t.Logf("✅ ICU NEURO ICH: outcome=%s, blocked=%v, bleeding_violation=%v",
		result.FinalOutcome, !result.FinalAllowed, hasBleedingViolation)
	t.Logf("   Clinical Truth: ICH + anticoag = catastrophic iatrogenic harm")
}

// TestICU_Neuro_ICH_ReversalAgentsAllowed tests that anticoagulant reversal
// agents are allowed (and encouraged) in ICH patients on anticoagulation.
//
// Scenario: ICH in patient on warfarin
// Expected: Reversal agents ALLOWED - part of ICH management
func TestICU_Neuro_ICH_ReversalAgentsAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-002",
		Age:        68,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I61.4", CodeSystem: "ICD10", Description: "Cerebellar hemorrhage"},
			{Code: "I48.91", CodeSystem: "ICD10", Description: "AFib on anticoagulation"},
		},
		CurrentMedications: []types.Medication{
			{
				Code:       "11289",
				CodeSystem: "RXNORM",
				Name:       "Warfarin",
				DrugClass:  "VITAMIN_K_ANTAGONIST",
				Dose:       5.0,
				DoseUnit:   "mg",
				Frequency:  "daily",
			},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "INTRACRANIAL_HEMORRHAGE", Status: "ACTIVE"},
			{RegistryCode: "ANTICOAGULATION", Status: "SUSPENDED"},
		},
		RecentLabs: []types.LabResult{
			{Code: "6301-6", CodeSystem: "LOINC", Name: "INR", Value: 3.2, Unit: "ratio"},
		},
	}

	// KB-19 recommends: 4-Factor PCC for warfarin reversal
	reversalRec := SimulatedRecommendation{
		Target:             "Prothrombin Complex Concentrate",
		TargetRxNorm:       "1163289",
		DrugClass:          "COAGULATION_FACTOR",
		RecommendedDose:    2000,
		DoseUnit:           "units",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICH_REVERSAL",
		Rationale:          "Urgent INR reversal in warfarin-associated ICH",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, reversalRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Reversal agent must be allowed - life-saving in ICH
	if result.IsBlocked() {
		t.Errorf("❌ ICU NEURO FAILURE: Reversal agent BLOCKED in anticoag-ICH")
		t.Errorf("   INR 3.2 with active cerebellar hemorrhage")
		t.Errorf("   4-Factor PCC is standard of care for reversal")
	}

	t.Logf("✅ ICU NEURO REVERSAL: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Clinical Truth: Reversal agents are THERAPEUTIC in anticoag-ICH")
}

// TestICU_Neuro_ICH_BPControlAllowed tests that blood pressure control
// medications are allowed in ICH for hematoma expansion prevention.
//
// Scenario: ICH with severe hypertension
// Expected: IV antihypertensives ALLOWED for BP target <140 systolic
func TestICU_Neuro_ICH_BPControlAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-003",
		Age:        65,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I61.0", CodeSystem: "ICD10", Description: "Basal ganglia hemorrhage"},
			{Code: "I10", CodeSystem: "ICD10", Description: "Essential hypertension"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "INTRACRANIAL_HEMORRHAGE", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  195,
			DiastolicBP: 110,
			HeartRate:   78,
			SpO2:        98,
		},
	}

	// KB-19 recommends: Nicardipine for BP control
	bpControlRec := SimulatedRecommendation{
		Target:             "Nicardipine",
		TargetRxNorm:       "7393",
		DrugClass:          "CALCIUM_CHANNEL_BLOCKER",
		RecommendedDose:    5.0,
		DoseUnit:           "mg/hr",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICH_BP_MANAGEMENT",
		Rationale:          "BP target <140 systolic to prevent hematoma expansion (AHA/ASA 2022)",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, bpControlRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: BP control must be allowed - prevents hematoma expansion
	if result.IsBlocked() {
		t.Errorf("❌ ICU NEURO FAILURE: BP control BLOCKED in hypertensive ICH")
		t.Errorf("   SBP 195 mmHg with active hemorrhage")
	}

	t.Logf("✅ ICU NEURO BP CONTROL: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Neuro_IschemicStroke_ThrombolyticsAllowed tests that thrombolytics
// are allowed in acute ischemic stroke within the treatment window.
//
// Scenario: Acute ischemic stroke, within 4.5 hour window
// Expected: tPA ALLOWED with appropriate governance review
func TestICU_Neuro_IschemicStroke_ThrombolyticsAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-004",
		Age:        70,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I63.9", CodeSystem: "ICD10", Description: "Acute ischemic stroke"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "STROKE_CODE", Status: "ACTIVE"},
			{RegistryCode: "TPA_WINDOW", Status: "ACTIVE"}, // Within treatment window
		},
		Vitals: &types.Vitals{
			SystolicBP:  165,
			DiastolicBP: 90,
			HeartRate:   82,
			SpO2:        96,
		},
		RecentLabs: []types.LabResult{
			{Code: "6301-6", CodeSystem: "LOINC", Name: "INR", Value: 1.0, Unit: "ratio"},
			{Code: "777-3", CodeSystem: "LOINC", Name: "Platelets", Value: 180, Unit: "10*3/uL"},
			{Code: "2339-0", CodeSystem: "LOINC", Name: "Glucose", Value: 145, Unit: "mg/dL"},
		},
	}

	// KB-19 recommends: Alteplase for acute ischemic stroke
	tpaRec := SimulatedRecommendation{
		Target:             "Alteplase",
		TargetRxNorm:       "8410",
		DrugClass:          "THROMBOLYTIC",
		RecommendedDose:    0.9,
		DoseUnit:           "mg/kg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ACUTE_STROKE_TPA",
		Rationale:          "Within 4.5h window, no contraindications, NIHSS >4",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, tpaRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: tPA should be allowed within proper window
	if result.IsBlocked() {
		t.Errorf("❌ ICU NEURO FAILURE: tPA BLOCKED in eligible ischemic stroke")
		t.Errorf("   Within treatment window with no contraindications")
	}

	t.Logf("✅ ICU NEURO ISCHEMIC: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Neuro_StatusEpilepticus_BenzoAllowed tests that benzodiazepines
// are allowed for seizure termination despite respiratory concerns.
//
// Scenario: Status epilepticus requiring urgent treatment
// Expected: Lorazepam ALLOWED - seizure termination priority
func TestICU_Neuro_StatusEpilepticus_BenzoAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-005",
		Age:        45,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "G41.0", CodeSystem: "ICD10", Description: "Grand mal status epilepticus"},
			{Code: "J44.1", CodeSystem: "ICD10", Description: "COPD with acute exacerbation"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "STATUS_EPILEPTICUS", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  155,
			DiastolicBP: 95,
			HeartRate:   120,
			SpO2:        91,
		},
	}

	// KB-19 recommends: Lorazepam for status epilepticus
	benzoRec := SimulatedRecommendation{
		Target:             "Lorazepam",
		TargetRxNorm:       "6470",
		DrugClass:          "BENZODIAZEPINE",
		RecommendedDose:    4.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "STATUS_EPILEPTICUS",
		Rationale:          "First-line therapy for status epilepticus (AES 2016)",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, benzoRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Benzo must be allowed despite respiratory concerns
	if result.IsBlocked() {
		t.Errorf("❌ ICU NEURO FAILURE: Benzo BLOCKED in status epilepticus")
		t.Errorf("   COPD concern does NOT override seizure termination need")
		t.Errorf("   Untreated status = brain damage in minutes")
	}

	t.Logf("✅ ICU NEURO STATUS: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Clinical Truth: Seizure termination > respiratory caution")
}

// TestICU_Neuro_RaisedICP_OsmoticTherapyAllowed tests that osmotic therapy
// is allowed for raised intracranial pressure management.
//
// Scenario: Raised ICP requiring urgent treatment
// Expected: Mannitol ALLOWED despite renal considerations
func TestICU_Neuro_RaisedICP_OsmoticTherapyAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-006",
		Age:        55,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "G93.2", CodeSystem: "ICD10", Description: "Benign intracranial hypertension"},
			{Code: "S06.2X0A", CodeSystem: "ICD10", Description: "Traumatic brain injury"},
			{Code: "N18.3", CodeSystem: "ICD10", Description: "CKD Stage 3"},
		},
		RenalFunction: &types.RenalFunction{
			EGFR:       45.0,
			Creatinine: 1.8,
			CKDStage:   "CKD_3",
			OnDialysis: false,
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "RAISED_ICP", Status: "ACTIVE"},
			{RegistryCode: "NEURO_ICU", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  180,
			DiastolicBP: 100,
			HeartRate:   58, // Bradycardia (Cushing response)
			SpO2:        95,
		},
	}

	// KB-19 recommends: Mannitol for ICP management
	mannitolRec := SimulatedRecommendation{
		Target:             "Mannitol",
		TargetRxNorm:       "6851",
		DrugClass:          "OSMOTIC_DIURETIC",
		RecommendedDose:    100,
		DoseUnit:           "g",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "RAISED_ICP",
		Rationale:          "ICP reduction for herniation prevention",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, mannitolRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Osmotic therapy must be allowed despite CKD
	if result.IsBlocked() {
		t.Errorf("❌ ICU NEURO FAILURE: Mannitol BLOCKED in raised ICP")
		t.Errorf("   CKD Stage 3 does NOT override brain herniation risk")
		t.Errorf("   Cushing triad present (BP↑, HR↓)")
	}

	t.Logf("✅ ICU NEURO ICP: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Clinical Truth: Brain protection > renal preservation")
}

// =============================================================================
// ICU NEUROLOGICAL INVARIANT TESTS
// =============================================================================

// TestICU_Neuro_Invariant_BrainProtectionAbsolute tests that brain-protective
// measures are consistently prioritized and bleeding risks are blocked.
func TestICU_Neuro_Invariant_BrainProtectionAbsolute(t *testing.T) {
	ctx := NewE2ETestContext()

	// ICH patient - absolute contraindication to anticoagulation
	ichPatient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-INV",
		Age:        70,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I61.9", CodeSystem: "ICD10", Description: "Intracerebral hemorrhage"},
			{Code: "I48.0", CodeSystem: "ICD10", Description: "Atrial fibrillation"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "INTRACRANIAL_HEMORRHAGE", Status: "ACTIVE"},
		},
	}

	anticoagulants := []SimulatedRecommendation{
		AnticoagulationRecommendation("Heparin", "5224", 80.0),
		AnticoagulationRecommendation("Enoxaparin", "67108", 40.0),
		AnticoagulationRecommendation("Warfarin", "11289", 5.0),
		AnticoagulationRecommendation("Apixaban", "1364430", 5.0),
		AnticoagulationRecommendation("Rivaroxaban", "1114195", 20.0),
	}

	allowedCount := 0
	for _, anticoag := range anticoagulants {
		result, err := ctx.ExecuteE2EFlow(ichPatient, anticoag)
		if err != nil {
			t.Errorf("Drug '%s' evaluation failed: %v", anticoag.Target, err)
			continue
		}

		if result.FinalAllowed {
			allowedCount++
			t.Errorf("❌ BRAIN PROTECTION VIOLATION: %s ALLOWED in ICH patient",
				anticoag.Target)
		} else {
			t.Logf("✅ %s correctly blocked in ICH", anticoag.Target)
		}
	}

	if allowedCount > 0 {
		t.Errorf("❌ ICU NEURO INVARIANT FAILURE: %d/%d anticoagulants allowed in active ICH",
			allowedCount, len(anticoagulants))
	} else {
		t.Logf("✅ ICU NEURO INVARIANT VERIFIED: All %d anticoagulants blocked in ICH",
			len(anticoagulants))
	}

	t.Logf("   Clinical Truth: ICH is ABSOLUTE contraindication to anticoagulation")
}

// TestICU_Neuro_Invariant_SeizureTerminationPriority tests that seizure
// termination is always prioritized over secondary concerns.
func TestICU_Neuro_Invariant_SeizureTerminationPriority(t *testing.T) {
	ctx := NewE2ETestContext()

	statusPatient := &types.PatientContext{
		PatientID:  "PT-ICU-NEURO-SE-INV",
		Age:        50,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "G41.0", CodeSystem: "ICD10", Description: "Status epilepticus"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "STATUS_EPILEPTICUS", Status: "ACTIVE"},
		},
	}

	seizureMeds := []SimulatedRecommendation{
		{
			Target:             "Lorazepam",
			TargetRxNorm:       "6470",
			DrugClass:          "BENZODIAZEPINE",
			RecommendedDose:    4.0,
			DoseUnit:           "mg",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "STATUS_EPILEPTICUS",
			Rationale:          "First-line seizure termination",
			Urgency:            "STAT",
		},
		{
			Target:             "Fosphenytoin",
			TargetRxNorm:       "93959",
			DrugClass:          "ANTICONVULSANT",
			RecommendedDose:    20.0,
			DoseUnit:           "mg PE/kg",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "STATUS_EPILEPTICUS",
			Rationale:          "Second-line seizure control",
			Urgency:            "STAT",
		},
		{
			Target:             "Levetiracetam",
			TargetRxNorm:       "39998",
			DrugClass:          "ANTICONVULSANT",
			RecommendedDose:    60.0,
			DoseUnit:           "mg/kg",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "STATUS_EPILEPTICUS",
			Rationale:          "Alternative second-line therapy",
			Urgency:            "STAT",
		},
	}

	blockedCount := 0
	for _, med := range seizureMeds {
		result, err := ctx.ExecuteE2EFlow(statusPatient, med)
		if err != nil {
			t.Errorf("Drug '%s' evaluation failed: %v", med.Target, err)
			continue
		}

		if result.IsBlocked() {
			blockedCount++
			t.Errorf("❌ SEIZURE TERMINATION FAILURE: %s BLOCKED in status epilepticus",
				med.Target)
		}
	}

	if blockedCount > 0 {
		t.Errorf("❌ SEIZURE INVARIANT FAILURE: %d/%d seizure meds blocked",
			blockedCount, len(seizureMeds))
	} else {
		t.Logf("✅ SEIZURE INVARIANT VERIFIED: All %d seizure medications allowed",
			len(seizureMeds))
	}
}
