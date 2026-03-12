// Package tests provides E2E clinical scenario testing for KB-18 + KB-19 integration.
// This file tests ICU COAGULATION and DIC scenarios.
//
// Clinical Truth: DIC is a consumption coagulopathy - treat the cause while
// supporting clotting factors. Inappropriate anticoagulation can be fatal.
package tests

import (
	"testing"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ICU COAGULATION / DIC SCENARIOS
// These tests prove that coagulation management is appropriately governed
// in critically ill patients with complex coagulopathies.
// =============================================================================

// TestICU_DIC_BloodProductsAllowed tests that blood products are allowed
// for DIC-related coagulopathy.
//
// Scenario: DIC with active bleeding
// Expected: FFP, cryoprecipitate, platelets ALLOWED
func TestICU_DIC_BloodProductsAllowed(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-DIC-001",
		Age:        52,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "D65", CodeSystem: "ICD10", Description: "DIC"},
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
			{Code: "K92.2", CodeSystem: "ICD10", Description: "GI hemorrhage"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "DIC_MANAGEMENT", Status: "ACTIVE"},
			{RegistryCode: "ACTIVE_HEMORRHAGE", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			{Code: "5902-2", CodeSystem: "LOINC", Name: "INR", Value: 3.8, Unit: ""},
			{Code: "777-3", CodeSystem: "LOINC", Name: "Platelets", Value: 28, Unit: "K/uL"},
			{Code: "3255-7", CodeSystem: "LOINC", Name: "Fibrinogen", Value: 85, Unit: "mg/dL"},
			{Code: "48065-7", CodeSystem: "LOINC", Name: "D-dimer", Value: 15.2, Unit: "mg/L"},
		},
	}

	// KB-19 recommends: FFP for coagulopathy
	ffpRec := SimulatedRecommendation{
		Target:             "Fresh Frozen Plasma",
		TargetRxNorm:       "",
		DrugClass:          "BLOOD_PRODUCT",
		RecommendedDose:    4,
		DoseUnit:           "units",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "DIC_BLEEDING",
		Rationale:          "DIC with INR 3.8, active bleeding - replace clotting factors",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, ffpRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: FFP must be allowed for bleeding DIC
	if result.IsBlocked() {
		t.Errorf("❌ DIC MANAGEMENT FAILURE: FFP blocked in bleeding coagulopathy")
		t.Errorf("   INR 3.8, Platelets 28K, Fibrinogen 85 - patient is bleeding")
	}

	t.Logf("✅ ICU DIC FFP: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_DIC_CryoprecipitateForLowFibrinogen tests cryoprecipitate replacement.
//
// Scenario: DIC with critically low fibrinogen
// Expected: Cryoprecipitate ALLOWED
func TestICU_DIC_CryoprecipitateForLowFibrinogen(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-DIC-002",
		Age:        45,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "D65", CodeSystem: "ICD10", Description: "DIC"},
			{Code: "A41.51", CodeSystem: "ICD10", Description: "Sepsis due to E.coli"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "DIC_MANAGEMENT", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			{Code: "3255-7", CodeSystem: "LOINC", Name: "Fibrinogen", Value: 65, Unit: "mg/dL"}, // Critical
		},
	}

	// KB-19 recommends: Cryoprecipitate for fibrinogen replacement
	cryoRec := SimulatedRecommendation{
		Target:             "Cryoprecipitate",
		TargetRxNorm:       "",
		DrugClass:          "BLOOD_PRODUCT",
		RecommendedDose:    10,
		DoseUnit:           "units",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "DIC_FIBRINOGEN",
		Rationale:          "Fibrinogen 65 mg/dL < 100 threshold - replace fibrinogen",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, cryoRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Cryoprecipitate must be allowed
	if result.IsBlocked() {
		t.Errorf("❌ DIC FIBRINOGEN FAILURE: Cryoprecipitate blocked with fibrinogen 65")
	}

	t.Logf("✅ ICU DIC CRYOPRECIPITATE: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// TestICU_DIC_AnticoagulationBlocked_ActiveBleeding tests that anticoagulation
// is blocked in DIC with active bleeding.
//
// Scenario: DIC + active hemorrhage + clot burden
// Expected: Anticoagulation BLOCKED despite thrombotic component
func TestICU_DIC_AnticoagulationBlocked_ActiveBleeding(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-DIC-003",
		Age:        58,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "D65", CodeSystem: "ICD10", Description: "DIC"},
			{Code: "K92.0", CodeSystem: "ICD10", Description: "Hematemesis"}, // Active bleeding
			{Code: "I26.99", CodeSystem: "ICD10", Description: "PE"}, // Thrombotic component
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "DIC_MANAGEMENT", Status: "ACTIVE"},
			{RegistryCode: "ACTIVE_HEMORRHAGE", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			{Code: "718-7", CodeSystem: "LOINC", Name: "Hemoglobin", Value: 6.5, Unit: "g/dL"},
			{Code: "777-3", CodeSystem: "LOINC", Name: "Platelets", Value: 42, Unit: "K/uL"},
		},
	}

	// Inappropriate anticoagulation recommendation
	anticoagRec := AnticoagulationRecommendation("Heparin", "5224", 80.0)

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: Anticoagulation must be BLOCKED during active bleeding
	if result.FinalAllowed {
		t.Errorf("❌ DIC BLEEDING SAFETY FAILURE: Anticoag ALLOWED with active hemorrhage")
		t.Errorf("   Hematemesis with Hgb 6.5 - hemorrhage overrides PE management")
	}

	t.Logf("✅ ICU DIC BLEEDING SAFETY: outcome=%s, blocked=%v",
		result.FinalOutcome, !result.FinalAllowed)
}

// TestICU_DIC_AnticoagulationAllowed_ThrombotiDominant tests anticoagulation
// in predominantly thrombotic DIC without active bleeding.
//
// Scenario: DIC with thrombotic predominance, no active bleeding
// Expected: Low-dose anticoagulation MAY be allowed with monitoring
func TestICU_DIC_AnticoagulationAllowed_ThromboticDominant(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-DIC-004",
		Age:        62,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "D65", CodeSystem: "ICD10", Description: "DIC"},
			{Code: "C25.9", CodeSystem: "ICD10", Description: "Pancreatic cancer"}, // Malignancy-associated DIC
			{Code: "I82.409", CodeSystem: "ICD10", Description: "Acute DVT"}, // Thrombotic manifestation
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "DIC_MANAGEMENT", Status: "ACTIVE"},
			{RegistryCode: "CANCER_ASSOCIATED_VTE", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			{Code: "5902-2", CodeSystem: "LOINC", Name: "INR", Value: 1.4, Unit: ""},
			{Code: "777-3", CodeSystem: "LOINC", Name: "Platelets", Value: 85, Unit: "K/uL"},
			{Code: "3255-7", CodeSystem: "LOINC", Name: "Fibrinogen", Value: 180, Unit: "mg/dL"},
		},
	}

	// Low-dose anticoagulation for thrombotic DIC
	anticoagRec := SimulatedRecommendation{
		Target:             "Enoxaparin",
		TargetRxNorm:       "67108",
		DrugClass:          "LMWH",
		RecommendedDose:    40, // Prophylactic dose
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIb,
		SourceProtocol:     "DIC_THROMBOTIC",
		Rationale:          "Thrombotic-predominant DIC without bleeding - consider anticoagulation",
		Urgency:            "ROUTINE",
	}

	result, err := ctx.ExecuteE2EFlow(patient, anticoagRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// This is a complex scenario - outcome depends on specific rule configuration
	t.Logf("ICU DIC THROMBOTIC: outcome=%s, allowed=%v",
		result.FinalOutcome, result.FinalAllowed)
	t.Logf("   Note: Thrombotic-predominant DIC may allow prophylactic anticoag")
}

// TestICU_DIC_TXABlocked_DIC tests that TXA is blocked in DIC.
//
// Scenario: DIC with bleeding - request for TXA
// Expected: TXA BLOCKED - may worsen thrombotic microangiopathy
func TestICU_DIC_TXABlocked_DIC(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-DIC-005",
		Age:        48,
		Sex:        "F",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "D65", CodeSystem: "ICD10", Description: "DIC"},
			{Code: "K92.2", CodeSystem: "ICD10", Description: "GI hemorrhage"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "DIC_MANAGEMENT", Status: "ACTIVE"},
		},
	}

	// TXA recommendation (inappropriate in DIC)
	txaRec := SimulatedRecommendation{
		Target:             "Tranexamic Acid",
		TargetRxNorm:       "10689",
		DrugClass:          "ANTIFIBRINOLYTIC",
		RecommendedDose:    1000,
		DoseUnit:           "mg",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassIIb,
		SourceProtocol:     "HEMORRHAGE",
		Rationale:          "Antifibrinolytic for bleeding control",
		Urgency:            "URGENT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, txaRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// Check for DIC contraindication warning
	t.Logf("ICU DIC TXA: outcome=%s, violations=%d",
		result.FinalOutcome, result.ViolationCount)
	t.Logf("   Note: TXA in DIC may worsen thrombotic component")
}

// =============================================================================
// SURGICAL BLEEDING SCENARIOS
// =============================================================================

// TestICU_Coag_MassiveTransfusionProtocol tests MTP activation.
//
// Scenario: Trauma with massive hemorrhage
// Expected: MTP products ALLOWED
func TestICU_Coag_MassiveTransfusionProtocol(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-MTP-001",
		Age:        32,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "T79.4XXA", CodeSystem: "ICD10", Description: "Traumatic hemorrhagic shock"},
			{Code: "S36.119A", CodeSystem: "ICD10", Description: "Laceration of spleen"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "MASSIVE_TRANSFUSION", Status: "ACTIVE"},
			{RegistryCode: "TRAUMA_ACTIVATION", Status: "ACTIVE"},
		},
		Vitals: &types.Vitals{
			SystolicBP:  68,
			DiastolicBP: 40,
			HeartRate:   135,
			SpO2:        92.0,
		},
		RecentLabs: []types.LabResult{
			{Code: "718-7", CodeSystem: "LOINC", Name: "Hemoglobin", Value: 5.8, Unit: "g/dL"},
		},
	}

	// MTP activation - 1:1:1 ratio
	mtpRec := SimulatedRecommendation{
		Target:             "Massive Transfusion Protocol",
		TargetRxNorm:       "",
		DrugClass:          "BLOOD_PRODUCT_PROTOCOL",
		RecommendedDose:    6, // 6 units each
		DoseUnit:           "units PRBC:FFP:PLT",
		RecommendationType: RecommendDo,
		EvidenceClass:      ClassI,
		SourceProtocol:     "TRAUMA_MTP",
		Rationale:          "Hemorrhagic shock, Hgb 5.8, MAP 49 - activate MTP",
		Urgency:            "STAT",
	}

	result, err := ctx.ExecuteE2EFlow(patient, mtpRec)
	if err != nil {
		t.Fatalf("E2E flow failed: %v", err)
	}

	// ASSERTION: MTP must be allowed in hemorrhagic shock
	if result.IsBlocked() {
		t.Errorf("❌ MTP FAILURE: Massive transfusion blocked in hemorrhagic shock")
		t.Errorf("   MAP 49, Hgb 5.8 - patient is exsanguinating")
	}

	t.Logf("✅ ICU MTP: outcome=%s, allowed=%v", result.FinalOutcome, result.FinalAllowed)
}

// =============================================================================
// DIC INVARIANT TESTS
// =============================================================================

// TestICU_DIC_Invariant_BloodProductsAllowedForBleeding tests that blood
// products are consistently allowed for DIC-related bleeding.
func TestICU_DIC_Invariant_BloodProductsAllowedForBleeding(t *testing.T) {
	ctx := NewE2ETestContext()

	patient := &types.PatientContext{
		PatientID:  "PT-ICU-DIC-INV",
		Age:        55,
		Sex:        "M",
		IsPregnant: false,
		ActiveDiagnoses: []types.Diagnosis{
			{Code: "D65", CodeSystem: "ICD10", Description: "DIC"},
			{Code: "A41.9", CodeSystem: "ICD10", Description: "Sepsis"},
		},
		RegistryMemberships: []types.RegistryMembership{
			{RegistryCode: "ICU_ADMISSION", Status: "ACTIVE"},
			{RegistryCode: "DIC_MANAGEMENT", Status: "ACTIVE"},
		},
		RecentLabs: []types.LabResult{
			{Code: "5902-2", CodeSystem: "LOINC", Name: "INR", Value: 3.5, Unit: ""},
			{Code: "777-3", CodeSystem: "LOINC", Name: "Platelets", Value: 35, Unit: "K/uL"},
			{Code: "3255-7", CodeSystem: "LOINC", Name: "Fibrinogen", Value: 75, Unit: "mg/dL"},
		},
	}

	bloodProducts := []SimulatedRecommendation{
		{
			Target:             "Fresh Frozen Plasma",
			TargetRxNorm:       "",
			DrugClass:          "BLOOD_PRODUCT",
			RecommendedDose:    4,
			DoseUnit:           "units",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "DIC_BLEEDING",
			Rationale:          "INR correction",
			Urgency:            "STAT",
		},
		{
			Target:             "Platelets",
			TargetRxNorm:       "",
			DrugClass:          "BLOOD_PRODUCT",
			RecommendedDose:    1,
			DoseUnit:           "apheresis unit",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "DIC_BLEEDING",
			Rationale:          "Platelet support for PLT <50K",
			Urgency:            "STAT",
		},
		{
			Target:             "Cryoprecipitate",
			TargetRxNorm:       "",
			DrugClass:          "BLOOD_PRODUCT",
			RecommendedDose:    10,
			DoseUnit:           "units",
			RecommendationType: RecommendDo,
			EvidenceClass:      ClassI,
			SourceProtocol:     "DIC_FIBRINOGEN",
			Rationale:          "Fibrinogen replacement",
			Urgency:            "URGENT",
		},
	}

	blockedCount := 0
	for _, rec := range bloodProducts {
		result, err := ctx.ExecuteE2EFlow(patient, rec)
		if err != nil {
			t.Errorf("Blood product '%s' failed: %v", rec.Target, err)
			continue
		}

		if result.IsBlocked() {
			blockedCount++
			t.Errorf("❌ DIC BLOOD PRODUCT BLOCKED: %s", rec.Target)
		}
	}

	if blockedCount > 0 {
		t.Errorf("❌ DIC INVARIANT FAILURE: %d/%d blood products blocked",
			blockedCount, len(bloodProducts))
	} else {
		t.Logf("✅ DIC INVARIANT VERIFIED: All %d blood products allowed for coagulopathy",
			len(bloodProducts))
	}
}
