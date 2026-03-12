// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ICU SEDATION AND VENTILATION scenarios.
//
// Clinical Truth: Sedation enables life-saving mechanical ventilation.
// Under-sedation risks self-extubation; over-sedation risks delirium and prolonged ICU stay.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ICU SEDATION AND VENTILATION SCENARIOS
// These tests prove that sedation management is appropriately governed
// in ventilated patients.
// =============================================================================

// TestICU_Sedation_PropololAllowedForVentilation tests that sedation is allowed
// for mechanically ventilated patients.
//
// Scenario: Intubated patient requiring sedation
// Expected: Propofol ALLOWED for ventilator synchrony
func TestICU_Sedation_PropololAllowedForVentilation(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-SED-001",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J96.01", CodeSystem: "ICD10", Description: "Acute respiratory failure with hypoxia"},
			{Code: "Z99.11", CodeSystem: "ICD10", Description: "Dependence on respirator status"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MECHANICAL_VENTILATION", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  125,
			DiastolicBP: 75,
			HeartRate:   88,
			SpO2:        94.0,
		},
	}

	// KB-19 recommends: Propofol for sedation
	propofolRec := SimulatedRecommendation{
		Target:             "Propofol",
		TargetRxNorm:       "8782",
		DrugClass:          "SEDATIVE",
		RecommendedDose:    50,
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICU_SEDATION",
		Rationale:          "Sedation for ventilator synchrony, target RASS -2 to 0",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, propofolRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Sedation must be allowed for ventilated patients
	if result.IsBlocked() {
		t.Errorf("❌ ICU SEDATION FAILURE: Propofol BLOCKED for intubated patient")
		t.Errorf("   Without sedation, patient risks self-extubation and injury")
	}

	t.Logf("✅ ICU SEDATION: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Sedation_DexmedetomidineForLightSedation tests dexmedetomidine
// for light sedation allowing neurological assessment.
//
// Scenario: Ventilated patient needing wake-up assessment
// Expected: Dexmedetomidine ALLOWED (allows arousability)
func TestICU_Sedation_DexmedetomidineForLightSedation(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-SED-002",
		Age:        62,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J96.01", CodeSystem: "ICD10", Description: "Acute respiratory failure"},
			{Code: "I63.9", CodeSystem: "ICD10", Description: "Cerebral infarction"}, // Needs neuro checks
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MECHANICAL_VENTILATION", Status: "ACTIVE"},
			{RegistryCode: "NEUROLOGICAL_MONITORING", Status: "ACTIVE"},
		},
	}

	// KB-19 recommends: Dexmedetomidine for light sedation
	dexRec := SimulatedRecommendation{
		Target:             "Dexmedetomidine",
		TargetRxNorm:       "1372718",
		DrugClass:          "SEDATIVE_ALPHA2",
		RecommendedDose:    0.4,
		DoseUnit:           "mcg/kg/hr",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "ICU_LIGHT_SEDATION",
		Rationale:          "Light sedation allowing neurological assessment, RASS target 0 to -1",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, dexRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Light sedation must be allowed for neuro monitoring
	if result.IsBlocked() {
		t.Errorf("❌ ICU LIGHT SEDATION FAILURE: Dexmedetomidine blocked")
	}

	t.Logf("✅ ICU LIGHT SEDATION: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Sedation_AnalgesiaFirst tests the "analgesia first" approach
// where pain control precedes sedation.
//
// Scenario: Agitated ventilated patient - check for pain first
// Expected: Fentanyl ALLOWED before additional sedation
func TestICU_Sedation_AnalgesiaFirst(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-SED-003",
		Age:        48,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J96.01", CodeSystem: "ICD10", Description: "Acute respiratory failure"},
			{Code: "S22.32XA", CodeSystem: "ICD10", Description: "Rib fractures"}, // Painful
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MECHANICAL_VENTILATION", Status: "ACTIVE"},
		},
	}

	// KB-19 recommends: Fentanyl for analgesia-first approach
	fentanylRec := SimulatedRecommendation{
		Target:             "Fentanyl",
		TargetRxNorm:       "4337",
		DrugClass:          "OPIOID_ANALGESIC",
		RecommendedDose:    50,
		DoseUnit:           "mcg/hr",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ICU_ANALGESIA_FIRST",
		Rationale:          "PADIS guidelines: Treat pain before considering additional sedation",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, fentanylRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Analgesia must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ ICU ANALGESIA FAILURE: Pain control blocked for ventilated patient")
	}

	t.Logf("✅ ICU ANALGESIA-FIRST: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Sedation_NMBAForARDS tests neuromuscular blockade in severe ARDS.
//
// Scenario: Severe ARDS with ventilator dyssynchrony
// Expected: NMBA ALLOWED with proper monitoring
func TestICU_Sedation_NMBAForARDS(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-SED-004",
		Age:        52,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J80", CodeSystem: "ICD10", Description: "ARDS"},
			{Code: "J96.00", CodeSystem: "ICD10", Description: "Acute respiratory failure"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MECHANICAL_VENTILATION", Status: "ACTIVE"},
			{RegistryCode: "ARDS_PROTOCOL", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			{Code: "19214-6", CodeSystem: "LOINC", Name: "PaO2/FiO2", Value: 95, Unit: "mmHg"}, // Severe ARDS
		},
	}

	// KB-19 recommends: Cisatracurium for ARDS
	nmbaRec := SimulatedRecommendation{
		Target:             "Cisatracurium",
		TargetRxNorm:       "22698",
		DrugClass:          "NMBA",
		RecommendedDose:    1.5,
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "ICU_ARDS_SEVERE",
		Rationale:          "P/F <100, NMBA for 48h per ACURASYS protocol",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, nmbaRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: NMBA must be allowed in severe ARDS
	if result.IsBlocked() {
		t.Errorf("❌ ICU ARDS NMBA FAILURE: Neuromuscular blockade blocked in severe ARDS")
		t.Errorf("   P/F ratio 95 - severe ARDS requiring ventilator optimization")
	}

	t.Logf("✅ ICU ARDS NMBA: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Sedation_PropololInfusionSyndrome tests for PRIS monitoring
// with prolonged propofol use.
//
// Scenario: Prolonged high-dose propofol
// Expected: ALLOWED with warnings about PRIS risk
func TestICU_Sedation_PropololInfusionSyndrome(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-SED-005",
		Age:        35,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "S06.2X9A", CodeSystem: "ICD10", Description: "Traumatic brain injury"},
			{Code: "Z99.11", CodeSystem: "ICD10", Description: "Mechanical ventilation dependence"},
		},
		CurrentMedications: []types.Medication{
			{
				Code:       "8782",
				CodeSystem: "RxNorm",
				Name:       "Propofol",
				Dose:       80, // Already on high dose
				DoseUnit:   "mcg/kg/min",
			},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MECHANICAL_VENTILATION", Status: "ACTIVE"},
		},
	}

	// KB-19 recommends: Increased propofol dose
	propofolHighRec := SimulatedRecommendation{
		Target:             "Propofol",
		TargetRxNorm:       "8782",
		DrugClass:          "SEDATIVE",
		RecommendedDose:    100, // High dose
		DoseUnit:           "mcg/kg/min",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIa,
		SourceProtocol:     "ICU_SEDATION_TBI",
		Rationale:          "ICP management requiring deep sedation",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, propofolHighRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// High-dose propofol should trigger warnings but may still be allowed
	t.Logf("ICU HIGH-DOSE PROPOFOL: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Note: High-dose propofol should trigger PRIS monitoring recommendations")
}

// =============================================================================
// VENTILATION MANAGEMENT TESTS
// =============================================================================

// TestICU_Ventilation_LowTidalVolumeProtocol tests lung-protective ventilation.
//
// Scenario: ARDS patient requiring lung-protective settings
// Expected: Low tidal volume strategy ALLOWED
func TestICU_Ventilation_LowTidalVolumeProtocol(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-VENT-001",
		Age:        58,
		Sex:        "F",
		IsPregnant: false,
		Weight:     70.0,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J80", CodeSystem: "ICD10", Description: "ARDS"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "ARDS_PROTOCOL", Status: "ACTIVE"},
		},
	}

	// KB-19 recommends: Low tidal volume ventilation
	ltvRec := SimulatedRecommendation{
		Target:             "Mechanical Ventilation Settings",
		TargetRxNorm:       "",
		DrugClass:          "VENTILATOR_SETTING",
		RecommendedDose:    6, // 6 mL/kg IBW
		DoseUnit:           "mL/kg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "ARDS_NET",
		Rationale:          "ARDSNet protocol: Low tidal volume 6 mL/kg IBW",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, ltvRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Lung-protective strategy must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ ARDS LUNG-PROTECTIVE FAILURE: Low TV blocked")
	}

	t.Logf("✅ ICU LUNG-PROTECTIVE VENTILATION: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
}

// TestICU_Ventilation_PronePositioning tests prone positioning for severe ARDS.
//
// Scenario: Severe ARDS not responding to conventional management
// Expected: Prone positioning ALLOWED per PROSEVA
func TestICU_Ventilation_PronePositioning(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-VENT-002",
		Age:        45,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J80", CodeSystem: "ICD10", Description: "ARDS"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "ARDS_PROTOCOL", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			{Code: "19214-6", CodeSystem: "LOINC", Name: "PaO2/FiO2", Value: 110, Unit: "mmHg"},
		},
	}

	// KB-19 recommends: Prone positioning
	proneRec := SimulatedRecommendation{
		Target:             "Prone Positioning",
		TargetRxNorm:       "",
		DrugClass:          "POSITIONING_THERAPY",
		RecommendedDose:    16, // 16 hours/day
		DoseUnit:           "hours",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "PROSEVA",
		Rationale:          "PROSEVA: Prone >16h/day for P/F <150 mmHg",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, proneRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Prone positioning must be allowed in severe ARDS
	if result.IsBlocked() {
		t.Errorf("❌ ARDS PRONE POSITIONING FAILURE: PROSEVA protocol blocked")
	}

	t.Logf("✅ ICU PRONE POSITIONING: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// =============================================================================
// SEDATION/VENTILATION INVARIANT TESTS
// =============================================================================

// TestICU_SedationVent_Invariant_SedationEnabledForVentilation tests that
// sedation is consistently allowed for ventilated patients.
func TestICU_SedationVent_Invariant_SedationEnabledForVentilation(t *testing.T) {
	ctx := NewE2ETestContext()

	// Base ventilated patient
	patient := &types.PatientContext{
		PatientID:  "PT-ICU-SEDVENT-INV",
		Age:        50,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "J96.01", CodeSystem: "ICD10", Description: "Acute respiratory failure"},
			{Code: "Z99.11", CodeSystem: "ICD10", Description: "Ventilator dependence"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MECHANICAL_VENTILATION", Status: "ACTIVE"},
		},
	}

	sedationOptions := []SimulatedRecommendation{
		{
			Target:             "Propofol",
			TargetRxNorm:       "8782",
			DrugClass:          "SEDATIVE",
			RecommendedDose:    50,
			DoseUnit:           "mcg/kg/min",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "ICU_SEDATION",
			Rationale:          "Ventilator sedation",
			Urgency:            "ROUTINE",
		},
		{
			Target:             "Dexmedetomidine",
			TargetRxNorm:       "1372718",
			DrugClass:          "SEDATIVE_ALPHA2",
			RecommendedDose:    0.5,
			DoseUnit:           "mcg/kg/hr",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassIIa,
			SourceProtocol:     "ICU_LIGHT_SEDATION",
			Rationale:          "Light sedation",
			Urgency:            "ROUTINE",
		},
		{
			Target:             "Midazolam",
			TargetRxNorm:       "6960",
			DrugClass:          "SEDATIVE_BENZO",
			RecommendedDose:    2,
			DoseUnit:           "mg/hr",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassIIb,
			SourceProtocol:     "ICU_SEDATION",
			Rationale:          "Sedation if propofol unavailable",
			Urgency:            "ROUTINE",
		},
	}

	blockedCount := 0
	for _, rec := range sedationOptions {
		result, err := ctx.ExecuteE2EFlow(patient, rec)
		if err != nil {
			t.Errorf("Sedation '%s' failed: %v", rec.Target, err)
			continue
		}

		if result.IsBlocked() {
			blockedCount++
			t.Errorf("❌ SEDATION BLOCKED: %s for ventilated patient", rec.Target)
		}
	}

	if blockedCount > 0 {
		t.Errorf("❌ SEDATION INVARIANT FAILURE: %d/%d sedation options blocked",
			blockedCount, len(sedationOptions))
	} else {
		t.Logf("✅ SEDATION INVARIANT VERIFIED: All %d sedation options available for ventilation",
			len(sedationOptions))
	}
}
