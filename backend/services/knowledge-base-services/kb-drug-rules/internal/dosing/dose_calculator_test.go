package dosing

import (
	"testing"
)

// ============================================================================
// DOSE CALCULATOR SERVICE TESTS
// ============================================================================

func TestNewDoseCalculatorService(t *testing.T) {
	svc := NewDoseCalculatorService()
	if svc == nil {
		t.Fatal("NewDoseCalculatorService returned nil")
	}
	if svc.calculator == nil {
		t.Error("Calculator not initialized")
	}
	if svc.drugRules == nil || len(svc.drugRules) == 0 {
		t.Error("Drug rules not initialized")
	}
}

func TestListDrugRules(t *testing.T) {
	svc := NewDoseCalculatorService()
	rules := svc.ListDrugRules()

	// Should have 24 built-in rules
	if len(rules) < 20 {
		t.Errorf("Expected at least 20 drug rules, got %d", len(rules))
	}

	// Check for specific drugs
	expectedDrugs := []string{
		"6809",    // Metformin
		"8610",    // Lisinopril
		"11289",   // Warfarin
		"11124",   // Vancomycin
		"7052",    // Morphine
	}

	for _, rxnorm := range expectedDrugs {
		if _, exists := rules[rxnorm]; !exists {
			t.Errorf("Expected drug %s not found in rules", rxnorm)
		}
	}
}

func TestGetDrugRule(t *testing.T) {
	svc := NewDoseCalculatorService()

	// Test existing drug
	rule, exists := svc.GetDrugRule("8610") // Lisinopril
	if !exists {
		t.Fatal("Expected Lisinopril rule to exist")
	}
	if rule.DrugName != "Lisinopril" {
		t.Errorf("Expected drug name 'Lisinopril', got '%s'", rule.DrugName)
	}

	// Test non-existing drug
	_, exists = svc.GetDrugRule("99999")
	if exists {
		t.Error("Expected non-existing drug to return false")
	}
}

// ============================================================================
// DOSE CALCULATION TESTS
// ============================================================================

func TestCalculateDose_FixedDosing(t *testing.T) {
	svc := NewDoseCalculatorService()

	req := DoseCalculationRequest{
		RxNormCode: "8610", // Lisinopril (fixed dosing)
		Patient: PatientParameters{
			Age:             45,
			Gender:          "M",
			WeightKg:        80.0,
			HeightCm:        175.0,
			SerumCreatinine: 1.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	if result.DrugName != "Lisinopril" {
		t.Errorf("Expected drug name 'Lisinopril', got '%s'", result.DrugName)
	}

	if result.DosingMethod != DosingMethodFixed {
		t.Errorf("Expected FIXED dosing method, got %s", result.DosingMethod)
	}

	if result.RecommendedDose != 10 { // Starting dose
		t.Errorf("Expected starting dose 10, got %v", result.RecommendedDose)
	}
}

func TestCalculateDose_WeightBasedDosing(t *testing.T) {
	svc := NewDoseCalculatorService()

	req := DoseCalculationRequest{
		RxNormCode: "11124", // Vancomycin (weight-based: 15 mg/kg)
		Patient: PatientParameters{
			Age:             45,
			Gender:          "M",
			WeightKg:        80.0,
			HeightCm:        175.0,
			SerumCreatinine: 1.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !result.Success {
		t.Fatalf("Expected success, got error: %s", result.Error)
	}

	// 15 mg/kg × 80 kg = 1200 mg
	expectedDose := 15.0 * 80.0
	if result.RecommendedDose < expectedDose*0.9 || result.RecommendedDose > expectedDose*1.1 {
		t.Errorf("Expected dose around %v, got %v", expectedDose, result.RecommendedDose)
	}

	if result.DosingMethod != DosingMethodWeightBased {
		t.Errorf("Expected WEIGHT_BASED dosing method, got %s", result.DosingMethod)
	}
}

func TestCalculateDose_RenalAdjustment(t *testing.T) {
	svc := NewDoseCalculatorService()

	tests := []struct {
		name        string
		egfr        float64
		expectLower bool
	}{
		{"Normal renal function (eGFR 95)", 95.0, false},
		{"Moderate CKD (eGFR 40)", 40.0, true},
		{"Severe CKD (eGFR 20)", 20.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			egfr := tt.egfr
			req := DoseCalculationRequest{
				RxNormCode: "6809", // Metformin (has renal adjustments)
				Patient: PatientParameters{
					Age:      50,
					Gender:   "M",
					WeightKg: 75.0,
					HeightCm: 175.0,
					EGFR:     &egfr,
				},
			}

			result, err := svc.CalculateDose(req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.RenalAdjustment != nil && result.RenalAdjustment.Applied != tt.expectLower {
				if tt.expectLower {
					t.Errorf("Expected renal adjustment to be applied for eGFR %v", tt.egfr)
				}
			}
		})
	}
}

func TestCalculateDose_AgeAdjustment(t *testing.T) {
	svc := NewDoseCalculatorService()

	// Test geriatric adjustment for Lisinopril
	req := DoseCalculationRequest{
		RxNormCode: "8610", // Lisinopril (has age adjustment for 65+)
		Patient: PatientParameters{
			Age:             75,
			Gender:          "F",
			WeightKg:        60.0,
			HeightCm:        160.0,
			SerumCreatinine: 1.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.AgeAdjustment == nil || !result.AgeAdjustment.Applied {
		t.Error("Expected age adjustment to be applied for 75-year-old")
	}
}

func TestCalculateDose_DrugNotFound(t *testing.T) {
	svc := NewDoseCalculatorService()

	req := DoseCalculationRequest{
		RxNormCode: "99999", // Non-existent drug
		Patient: PatientParameters{
			Age:      45,
			Gender:   "M",
			WeightKg: 80.0,
			HeightCm: 175.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Success {
		t.Error("Expected failure for non-existent drug")
	}

	if result.ErrorCode != "DRUG_NOT_FOUND" {
		t.Errorf("Expected error code DRUG_NOT_FOUND, got %s", result.ErrorCode)
	}
}

// ============================================================================
// SAFETY ALERT TESTS
// ============================================================================

func TestSafetyAlerts_HighAlert(t *testing.T) {
	svc := NewDoseCalculatorService()

	// Test high-alert medication (Warfarin)
	req := DoseCalculationRequest{
		RxNormCode: "11289", // Warfarin
		Patient: PatientParameters{
			Age:      50,
			Gender:   "M",
			WeightKg: 75.0,
			HeightCm: 175.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	foundHighAlert := false
	for _, alert := range result.Alerts {
		if alert.AlertType == "high_alert" {
			foundHighAlert = true
			break
		}
	}

	if !foundHighAlert {
		t.Error("Expected high_alert for Warfarin")
	}
}

func TestSafetyAlerts_NarrowTI(t *testing.T) {
	svc := NewDoseCalculatorService()

	// Test narrow therapeutic index (Warfarin)
	req := DoseCalculationRequest{
		RxNormCode: "11289", // Warfarin
		Patient: PatientParameters{
			Age:      50,
			Gender:   "M",
			WeightKg: 75.0,
			HeightCm: 175.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	foundNarrowTI := false
	for _, alert := range result.Alerts {
		if alert.AlertType == "narrow_ti" {
			foundNarrowTI = true
			break
		}
	}

	if !foundNarrowTI {
		t.Error("Expected narrow_ti alert for Warfarin")
	}
}

func TestSafetyAlerts_BlackBox(t *testing.T) {
	svc := NewDoseCalculatorService()

	// Test black box warning (Liraglutide)
	req := DoseCalculationRequest{
		RxNormCode: "475968", // Liraglutide (thyroid C-cell tumors warning)
		Patient: PatientParameters{
			Age:      45,
			Gender:   "F",
			WeightKg: 85.0,
			HeightCm: 165.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	foundBlackBox := false
	for _, alert := range result.Alerts {
		if alert.AlertType == "black_box" {
			foundBlackBox = true
			break
		}
	}

	if !foundBlackBox {
		t.Error("Expected black_box alert for Liraglutide")
	}
}

func TestSafetyAlerts_BeersGeriatric(t *testing.T) {
	svc := NewDoseCalculatorService()

	// Test Beers Criteria for elderly patient (Ibuprofen)
	req := DoseCalculationRequest{
		RxNormCode: "5640", // Ibuprofen (Beers list)
		Patient: PatientParameters{
			Age:      75,
			Gender:   "M",
			WeightKg: 70.0,
			HeightCm: 170.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	foundBeers := false
	for _, alert := range result.Alerts {
		if alert.AlertType == "beers" {
			foundBeers = true
			break
		}
	}

	if !foundBeers {
		t.Error("Expected Beers Criteria alert for elderly patient on Ibuprofen")
	}
}

// ============================================================================
// SPECIFIC DRUG RULE TESTS
// ============================================================================

func TestDrugRules_Metformin(t *testing.T) {
	svc := NewDoseCalculatorService()

	rule, exists := svc.GetDrugRule("6809")
	if !exists {
		t.Fatal("Metformin rule not found")
	}

	if rule.MaxDailyDose != 2000 {
		t.Errorf("Expected max daily dose 2000, got %v", rule.MaxDailyDose)
	}

	if len(rule.RenalAdjustments) < 3 {
		t.Error("Expected at least 3 renal adjustment tiers for Metformin")
	}
}

func TestDrugRules_InsulinGlargine(t *testing.T) {
	svc := NewDoseCalculatorService()

	rule, exists := svc.GetDrugRule("261551")
	if !exists {
		t.Fatal("Insulin Glargine rule not found")
	}

	if !rule.IsHighAlert {
		t.Error("Insulin Glargine should be marked as high-alert")
	}

	if rule.DosingMethod != DosingMethodWeightBased {
		t.Errorf("Expected weight-based dosing for insulin, got %s", rule.DosingMethod)
	}
}

func TestDrugRules_Opioids(t *testing.T) {
	svc := NewDoseCalculatorService()

	opioids := []string{"7052", "7804"} // Morphine, Oxycodone
	for _, rxnorm := range opioids {
		rule, exists := svc.GetDrugRule(rxnorm)
		if !exists {
			t.Errorf("Opioid %s not found", rxnorm)
			continue
		}

		if !rule.IsHighAlert {
			t.Errorf("%s should be marked as high-alert", rule.DrugName)
		}

		if !rule.HasBlackBoxWarning {
			t.Errorf("%s should have black box warning", rule.DrugName)
		}
	}
}

func TestDrugRules_Anticoagulants(t *testing.T) {
	svc := NewDoseCalculatorService()

	anticoagulants := []string{"11289", "1364430", "67108", "5224"} // Warfarin, Apixaban, Enoxaparin, Heparin
	for _, rxnorm := range anticoagulants {
		rule, exists := svc.GetDrugRule(rxnorm)
		if !exists {
			t.Errorf("Anticoagulant %s not found", rxnorm)
			continue
		}

		if !rule.IsHighAlert {
			t.Errorf("%s should be marked as high-alert", rule.DrugName)
		}
	}
}

// ============================================================================
// RXNORM LOOKUP TESTS
// ============================================================================

func TestGetRxNormCodeByName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"metformin", "6809"},
		{"Metformin", "6809"},
		{"lisinopril", "8610"},
		{"warfarin", "11289"},
		{"vancomycin", "11124"},
		{"morphine", "7052"},
		{"unknown_drug", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetRxNormCodeByName(tt.name)
			if result != tt.expected {
				t.Errorf("GetRxNormCodeByName(%s) = %s, want %s",
					tt.name, result, tt.expected)
			}
		})
	}
}

// ============================================================================
// EDGE CASE TESTS
// ============================================================================

func TestCalculateDose_ExtremeValues(t *testing.T) {
	svc := NewDoseCalculatorService()

	tests := []struct {
		name     string
		patient  PatientParameters
		hasError bool
	}{
		{
			name: "Very elderly patient",
			patient: PatientParameters{
				Age:      100,
				Gender:   "F",
				WeightKg: 50.0,
				HeightCm: 155.0,
			},
			hasError: false,
		},
		{
			name: "Very obese patient",
			patient: PatientParameters{
				Age:      45,
				Gender:   "M",
				WeightKg: 200.0,
				HeightCm: 175.0,
			},
			hasError: false,
		},
		{
			name: "Pediatric patient",
			patient: PatientParameters{
				Age:      5,
				Gender:   "M",
				WeightKg: 20.0,
				HeightCm: 110.0,
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := DoseCalculationRequest{
				RxNormCode: "8610", // Lisinopril
				Patient:    tt.patient,
			}

			result, err := svc.CalculateDose(req)
			if tt.hasError && err == nil && result.Success {
				t.Error("Expected error for extreme values")
			}
			if !tt.hasError && !result.Success {
				t.Errorf("Unexpected failure: %s", result.Error)
			}
		})
	}
}

func TestCalculateDose_MaxDoseEnforcement(t *testing.T) {
	svc := NewDoseCalculatorService()

	// Very heavy patient to trigger max dose cap
	req := DoseCalculationRequest{
		RxNormCode: "11124", // Vancomycin (15 mg/kg, max 4000 mg)
		Patient: PatientParameters{
			Age:             45,
			Gender:          "M",
			WeightKg:        300.0, // Would be 4500 mg at 15 mg/kg
			HeightCm:        175.0,
			SerumCreatinine: 1.0,
		},
	}

	result, err := svc.CalculateDose(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should be capped at max daily dose (4000 mg)
	if result.RecommendedDose > 4000 {
		t.Errorf("Expected dose to be capped at 4000, got %v", result.RecommendedDose)
	}
}
