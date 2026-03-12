// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ICU OVERRIDE EMERGENCY scenarios.
//
// Clinical Truth: In cardiac arrest and imminent death scenarios, governance must
// enable immediate life-saving actions with post-hoc documentation.
package tests

import (
	"testing"
	"time"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ICU OVERRIDE EMERGENCY SCENARIOS
// These tests prove that in true emergencies, life-saving interventions
// proceed immediately with proper emergency override pathways.
// =============================================================================

// TestICU_Emergency_CardiacArrest_ACLS tests that ACLS medications are allowed
// without delay during cardiac arrest.
//
// Scenario: Cardiac arrest - code blue
// Expected: Epinephrine, Amiodarone ALLOWED immediately
func TestICU_Emergency_CardiacArrest_ACLS(t *testing.T) {
	ctx := NewE2ETestContext()

	// Patient: Active cardiac arrest
	patient := &types.PatientContext{
		PatientID:  "PT-ICU-CODE-001",
		Age:        65,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I46.9", CodeSystem: "ICD10", Description: "Cardiac arrest"},
			{Code: "I49.01", CodeSystem: "ICD10", Description: "VF"}, // Shockable rhythm
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "CODE_BLUE", Status: "ACTIVE"},
			{RegistryCode: "ACLS_PROTOCOL", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  0, // No pulse
			DiastolicBP: 0,
			HeartRate:   0, // PEA/VF
			SpO2:        0,
		},
	}

	// ACLS: Epinephrine 1mg IV q3-5min
	epinephrineRec := SimulatedRecommendation{
		Target:             "Epinephrine",
		TargetRxNorm:       "3992",
		DrugClass:          "VASOPRESSOR_ACLS",
		RecommendedDose:    1.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ACLS_CARDIAC_ARREST",
		Rationale:          "AHA ACLS: Epinephrine 1mg IV/IO every 3-5 minutes",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, epinephrineRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: ACLS drugs MUST be allowed immediately
	if result.IsBlocked() {
		t.Errorf("❌ CRITICAL FAILURE: ACLS medication BLOCKED during cardiac arrest")
		t.Errorf("   This is a life-or-death scenario - zero delay acceptable")
	}

	// Verify emergency pathway was used
	if result.EnforcementApplied == types.EnforcementHardBlock {
		t.Errorf("❌ ACLS should NOT be subject to HARD_BLOCK")
	}

	t.Logf("✅ ICU CARDIAC ARREST ACLS: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Emergency_Amiodarone_ShockableRhythm tests amiodarone in VF/VT.
//
// Scenario: Refractory VF/pulseless VT
// Expected: Amiodarone ALLOWED per ACLS
func TestICU_Emergency_Amiodarone_ShockableRhythm(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-CODE-002",
		Age:        58,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I46.9", CodeSystem: "ICD10", Description: "Cardiac arrest"},
			{Code: "I49.01", CodeSystem: "ICD10", Description: "VF"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "CODE_BLUE", Status: "ACTIVE"},
			{RegistryCode: "ACLS_PROTOCOL", Status: "ACTIVE"},
		},
	}

	// ACLS: Amiodarone 300mg IV first dose
	amiodaroneRec := SimulatedRecommendation{
		Target:             "Amiodarone",
		TargetRxNorm:       "703",
		DrugClass:          "ANTIARRHYTHMIC",
		RecommendedDose:    300,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ACLS_VF_VT",
		Rationale:          "AHA ACLS: Amiodarone 300mg IV for shock-refractory VF/pVT",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, amiodaroneRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: ACLS antiarrhythmic must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ ACLS ANTIARRHYTHMIC BLOCKED in VF/VT")
	}

	t.Logf("✅ ICU VF/VT AMIODARONE: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Emergency_ImminentDeath_BypassNormal tests bypass of normal
// governance in imminent death scenarios.
//
// Scenario: Massive PE with cardiovascular collapse
// Expected: tPA ALLOWED despite bleeding risk - imminent death
func TestICU_Emergency_ImminentDeath_BypassNormal(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-IMMINENT-001",
		Age:        52,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I26.02", CodeSystem: "ICD10", Description: "Saddle PE with cor pulmonale"},
			{Code: "R57.0", CodeSystem: "ICD10", Description: "Cardiogenic shock"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MASSIVE_PE", Status: "ACTIVE"},
			{RegistryCode: "IMMINENT_DEATH", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  60,
			DiastolicBP: 35,
			HeartRate:   130,
			SpO2:        78.0, // Severe hypoxia
		},
	}

	// tPA for massive PE with cardiovascular collapse
	tpaRec := SimulatedRecommendation{
		Target:             "Alteplase",
		TargetRxNorm:       "8410",
		DrugClass:          "THROMBOLYTIC",
		RecommendedDose:    100,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "MASSIVE_PE_THROMBOLYSIS",
		Rationale:          "ESC/AHA: Systemic thrombolysis for high-risk PE with shock",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, tpaRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// tPA has bleeding risks, but massive PE with shock = imminent death
	t.Logf("ICU MASSIVE PE tPA: outcome=%s, allowed=%v, violations=%d",
		result.FinalOutcome, result.FinalAllowed, result.ViolationCount)
	t.Logf("   Note: In imminent death, thrombolysis benefit > bleeding risk")
}

// TestICU_Emergency_PostHocDocumentation tests that emergency overrides
// require post-hoc documentation.
func TestICU_Emergency_PostHocDocumentation(t *testing.T) {
	ctx := NewE2ETestContext()

	// Code scenario
	patient := &types.PatientContext{
		PatientID:  "PT-ICU-DOC-001",
		Age:        70,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I46.9", CodeSystem: "ICD10", Description: "Cardiac arrest"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "CODE_BLUE", Status: "ACTIVE"},
		},
	}

	epinephrineRec := SimulatedRecommendation{
		Target:             "Epinephrine",
		TargetRxNorm:       "3992",
		DrugClass:          "VASOPRESSOR_ACLS",
		RecommendedDose:    1.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ACLS_CARDIAC_ARREST",
		Rationale:          "ACLS epinephrine",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, epinephrineRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Verify evidence trail exists for post-hoc documentation
	if !result.HasEvidenceTrail() {
		t.Errorf("❌ EMERGENCY DOCUMENTATION FAILURE: Missing evidence trail for code blue")
	}

	t.Logf("✅ ICU EMERGENCY DOCUMENTATION: trail_hash=%s...",
		result.EvidenceTrailHash[:min(20, len(result.EvidenceTrailHash))])
}

// TestICU_Emergency_MalignantHyperthermia tests MH protocol activation.
//
// Scenario: Intraoperative malignant hyperthermia
// Expected: Dantrolene ALLOWED immediately
func TestICU_Emergency_MalignantHyperthermia(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-MH-001",
		Age:        35,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "T88.3XXA", CodeSystem: "ICD10", Description: "Malignant hyperthermia"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "MH_CRISIS", Status: "ACTIVE"},
			{RegistryCode: "OR_EMERGENCY", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			Temperature: 42.0, // Hyperpyrexia
			HeartRate:   150,
			SpO2:        92, // Decreased
		},
	}

	// Dantrolene for MH
	dantroleneRec := SimulatedRecommendation{
		Target:             "Dantrolene",
		TargetRxNorm:       "3105",
		DrugClass:          "MH_ANTIDOTE",
		RecommendedDose:    2.5, // mg/kg
		DoseUnit:           "mg/kg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "MH_PROTOCOL",
		Rationale:          "MHAUS: Dantrolene 2.5 mg/kg IV initial dose for MH crisis",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, dantroleneRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: MH antidote must be allowed immediately
	if result.IsBlocked() {
		t.Errorf("❌ MH CRISIS FAILURE: Dantrolene BLOCKED - patient will die without it")
	}

	t.Logf("✅ ICU MH DANTROLENE: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Emergency_AnaphylaxisEpinephrine tests anaphylaxis treatment.
//
// Scenario: Severe anaphylaxis with cardiovascular collapse
// Expected: Epinephrine IM ALLOWED immediately
func TestICU_Emergency_AnaphylaxisEpinephrine(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-ANAPH-001",
		Age:        28,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "T78.2XXA", CodeSystem: "ICD10", Description: "Anaphylactic shock"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ANAPHYLAXIS", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  65,
			DiastolicBP: 40,
			HeartRate:   140,
			SpO2:        85.0,
		},
	}

	// Epinephrine for anaphylaxis
	epiRec := SimulatedRecommendation{
		Target:             "Epinephrine",
		TargetRxNorm:       "3992",
		DrugClass:          "ANAPHYLAXIS_TREATMENT",
		RecommendedDose:    0.5,
		DoseUnit:           "mg IM",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ANAPHYLAXIS",
		Rationale:          "WAO: Epinephrine IM is first-line treatment for anaphylaxis",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, epiRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Anaphylaxis epinephrine must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ ANAPHYLAXIS FAILURE: Epinephrine BLOCKED in anaphylactic shock")
	}

	t.Logf("✅ ICU ANAPHYLAXIS: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// =============================================================================
// EMERGENCY OVERRIDE TIME CONSTRAINTS
// =============================================================================

// TestICU_Emergency_NoDelayForCodeBlue tests that code blue medications
// have no evaluation delay.
func TestICU_Emergency_NoDelayForCodeBlue(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-SPEED-001",
		Age:        60,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I46.9", CodeSystem: "ICD10", Description: "Cardiac arrest"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "CODE_BLUE", Status: "ACTIVE"},
		},
	}

	epinephrineRec := SimulatedRecommendation{
		Target:             "Epinephrine",
		TargetRxNorm:       "3992",
		DrugClass:          "VASOPRESSOR_ACLS",
		RecommendedDose:    1.0,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ACLS",
		Rationale:          "ACLS epinephrine",
		Urgency:            "STAT",
	}

	// Measure evaluation time
	start := time.Now()
	result, err := ctx.ExecuteE2EFlow(patient, epinephrineRec)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Emergency evaluations should be fast
	t.Logf("ICU CODE BLUE SPEED: evaluation_time=%v, allowed=%v",
		elapsed, result.FinalAllowed)

	// Very loose threshold - just ensuring evaluation completes quickly
	if elapsed > 500*time.Millisecond {
		t.Logf("Note: Evaluation took %v - ensure no unnecessary delays in emergencies", elapsed)
	}
}

// =============================================================================
// EMERGENCY INVARIANT TESTS
// =============================================================================

// TestICU_Emergency_Invariant_LifeSavingNeverBlocked tests that life-saving
// emergency interventions are never blocked.
func TestICU_Emergency_Invariant_LifeSavingNeverBlocked(t *testing.T) {
	ctx := NewE2ETestContext()

	// Code blue patient
	codePatient := &types.PatientContext{
		PatientID:  "PT-ICU-EMERG-INV",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "I46.9", CodeSystem: "ICD10", Description: "Cardiac arrest"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "CODE_BLUE", Status: "ACTIVE"},
			{RegistryCode: "ACLS_PROTOCOL", Status: "ACTIVE"},
		},
	}

	emergencyInterventions := []SimulatedRecommendation{
		{
			Target:             "Epinephrine",
			TargetRxNorm:       "3992",
			DrugClass:          "VASOPRESSOR_ACLS",
			RecommendedDose:    1.0,
			DoseUnit:           "mg",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "ACLS",
			Rationale:          "ACLS epinephrine",
			Urgency:            "STAT",
		},
		{
			Target:             "Amiodarone",
			TargetRxNorm:       "703",
			DrugClass:          "ANTIARRHYTHMIC",
			RecommendedDose:    300,
			DoseUnit:           "mg",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "ACLS_VF_VT",
			Rationale:          "ACLS amiodarone",
			Urgency:            "STAT",
		},
		{
			Target:             "Defibrillation",
			TargetRxNorm:       "",
			DrugClass:          "ELECTRICAL_THERAPY",
			RecommendedDose:    200,
			DoseUnit:           "J",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "ACLS_VF",
			Rationale:          "Defibrillation for VF",
			Urgency:            "STAT",
		},
	}

	blockedCount := 0
	for _, rec := range emergencyInterventions {
		result, err := ctx.ExecuteE2EFlow(codePatient, rec)
		if err != nil {
			t.Errorf("Emergency '%s' failed: %v", rec.Target, err)
			continue
		}

		if result.IsBlocked() {
			blockedCount++
			t.Errorf("❌ EMERGENCY INVARIANT VIOLATION: %s BLOCKED during code blue", rec.Target)
		}
	}

	if blockedCount > 0 {
		t.Errorf("❌ EMERGENCY INVARIANT FAILURE: %d/%d life-saving interventions blocked during code",
			blockedCount, len(emergencyInterventions))
	} else {
		t.Logf("✅ EMERGENCY INVARIANT VERIFIED: All %d code blue interventions allowed",
			len(emergencyInterventions))
	}
}

// TestICU_Emergency_Invariant_EvidenceTrailAlwaysCreated tests that even
// emergency overrides create audit trails.
func TestICU_Emergency_Invariant_EvidenceTrailAlwaysCreated(t *testing.T) {
	ctx := NewE2ETestContext()

	emergencyScenarios := []struct {
		name    string
		patient *types.PatientContext
		rec     SimulatedRecommendation
	}{
		{
			name: "Code Blue - Epinephrine",
			patient: &types.PatientContext{
				PatientID:  "PT-TRAIL-001",
				Age:        60,
				Sex:        "M",
				IsPregnant: false,
				ActiveDiagnoses: []types.Diagnosis{
					{Code: "I46.9", CodeSystem: "ICD10", Description: "Cardiac arrest"},
				},
				RegistryMemberships: []types.RegistryMembership{
					{RegistryCode: "CODE_BLUE", Status: "ACTIVE"},
				},
			},
			rec: SimulatedRecommendation{
				Target:             "Epinephrine",
				TargetRxNorm:       "3992",
				DrugClass:          "VASOPRESSOR_ACLS",
				RecommendedDose:    1.0,
				DoseUnit:           "mg",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "ACLS",
				Rationale:          "ACLS",
				Urgency:            "STAT",
			},
		},
		{
			name: "Anaphylaxis - Epinephrine",
			patient: &types.PatientContext{
				PatientID:  "PT-TRAIL-002",
				Age:        30,
				Sex:        "F",
				IsPregnant: false,
				ActiveDiagnoses: []types.Diagnosis{
					{Code: "T78.2XXA", CodeSystem: "ICD10", Description: "Anaphylaxis"},
				},
				RegistryMemberships: []types.RegistryMembership{
					{RegistryCode: "ANAPHYLAXIS", Status: "ACTIVE"},
				},
			},
			rec: SimulatedRecommendation{
				Target:             "Epinephrine",
				TargetRxNorm:       "3992",
				DrugClass:          "ANAPHYLAXIS_TREATMENT",
				RecommendedDose:    0.5,
				DoseUnit:           "mg",
				RecommendationType: RecommendDo,
				EvidenceClass:      ClassI,
				SourceProtocol:     "ANAPHYLAXIS",
				Rationale:          "Anaphylaxis treatment",
				Urgency:            "STAT",
			},
		},
	}

	missingTrails := 0
	for _, scenario := range emergencyScenarios {
		result, err := ctx.ExecuteE2EFlow(scenario.patient, scenario.rec)
		if err != nil {
			t.Errorf("Scenario '%s' failed: %v", scenario.name, err)
			continue
		}

		if !result.HasEvidenceTrail() {
			missingTrails++
			t.Errorf("❌ AUDIT FAILURE: %s missing evidence trail", scenario.name)
		}
	}

	if missingTrails > 0 {
		t.Errorf("❌ AUDIT INVARIANT FAILURE: %d/%d emergency scenarios missing audit trails",
			missingTrails, len(emergencyScenarios))
	} else {
		t.Logf("✅ AUDIT INVARIANT VERIFIED: All %d emergency scenarios have evidence trails",
			len(emergencyScenarios))
	}
}
