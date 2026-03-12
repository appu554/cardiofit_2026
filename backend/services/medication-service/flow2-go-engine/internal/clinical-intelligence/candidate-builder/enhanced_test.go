package candidatebuilder

import (
	"context"
	"testing"
	"time"
)

// TestEnhancedCandidateBuilder_SafetyScoring tests the enhanced safety scoring functionality
func TestEnhancedCandidateBuilder_SafetyScoring(t *testing.T) {
	// Create candidate builder with enhanced config
	config := &BuilderConfig{
		MaxWorkers:           5,
		DDITimeout:           30 * time.Second,
		EnableBlackBoxFilter: true,
		StrictSafetyMode:     true,
		MaxSafetyScore:       1.0,
		MinSafetyScore:       0.0,
	}
	
	builder := NewCandidateBuilderWithConfig(config, nil)
	
	// Test health check
	if err := builder.HealthCheck(); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	
	// Create test data with enhanced drug fields
	testDrugs := []Drug{
		{
			Code:                  "LIS001",
			Name:                  "Lisinopril",
			TherapeuticClasses:    []string{"ACE_INHIBITOR"},
			Contraindications:     []string{"ANGIOEDEMA_HISTORY"},
			ContraindicationCodes: []string{"ANGIOEDEMA_HISTORY"},
			EfficacyScore:         85.0,
			IsGeneric:             true,
			PregnancyCategory:     "D", // Should reduce safety score
			BlackBoxWarning:       false,
			RenalAdjustment:       true,
			HepaticAdjustment:     false,
		},
		{
			Code:                  "LOS001",
			Name:                  "Losartan",
			TherapeuticClasses:    []string{"ARB"},
			Contraindications:     []string{"PREGNANCY"},
			ContraindicationCodes: []string{"PREGNANCY"},
			EfficacyScore:         88.0,
			IsGeneric:             true,
			PregnancyCategory:     "D", // Should reduce safety score
			BlackBoxWarning:       false,
			RenalAdjustment:       false,
			HepaticAdjustment:     false,
		},
		{
			Code:                  "WAR001",
			Name:                  "Warfarin",
			TherapeuticClasses:    []string{"ANTICOAGULANT"},
			Contraindications:     []string{"BLEEDING_DISORDER"},
			ContraindicationCodes: []string{"BLEEDING_DISORDER"},
			EfficacyScore:         90.0,
			IsGeneric:             true,
			PregnancyCategory:     "X", // Should significantly reduce safety score
			BlackBoxWarning:       true, // Should reduce safety score
			RenalAdjustment:       true,
			HepaticAdjustment:     true,
		},
	}
	
	testPatientFlags := map[string]bool{
		"has_history_of_angioedema": false, // No angioedema history
		"is_pregnant":               false, // Not pregnant
		"has_kidney_disease":        false, // No kidney disease
		"has_liver_disease":         false, // No liver disease
		"high_risk_patient":         false, // Not high risk
	}
	
	testActiveMedications := []ActiveMedication{}
	testDDIRules := []DrugInteraction{}
	
	// Create input
	input := CandidateBuilderInput{
		RequestID:              "test-enhanced-123",
		PatientID:              "test-patient-456",
		RecommendedDrugClasses: []string{"ACE_INHIBITOR", "ARB", "ANTICOAGULANT"},
		PatientFlags:           testPatientFlags,
		ActiveMedications:      testActiveMedications,
		DrugMasterList:         testDrugs,
		DDIRules:              testDDIRules,
	}
	
	// Execute candidate building
	result, err := builder.BuildCandidateProposals(context.Background(), input)
	if err != nil {
		t.Fatalf("BuildCandidateProposals failed: %v", err)
	}
	
	// Validate results
	if result == nil {
		t.Fatal("Result is nil")
	}
	
	// Should have 3 candidates (all pass safety filters)
	expectedCandidates := 3
	if len(result.CandidateProposals) != expectedCandidates {
		t.Errorf("Expected %d candidates, got %d", expectedCandidates, len(result.CandidateProposals))
	}
	
	// Verify safety scoring and ranking
	if len(result.CandidateProposals) >= 3 {
		// Losartan should have highest safety score (no black box, pregnancy D)
		// Lisinopril should be second (pregnancy D, renal adjustment)
		// Warfarin should have lowest safety score (pregnancy X, black box, both adjustments)
		
		losartan := result.CandidateProposals[0]
		lisinopril := result.CandidateProposals[1]
		warfarin := result.CandidateProposals[2]
		
		if losartan.MedicationName != "Losartan" {
			t.Errorf("Expected Losartan to be ranked first, got %s", losartan.MedicationName)
		}
		
		if lisinopril.MedicationName != "Lisinopril" {
			t.Errorf("Expected Lisinopril to be ranked second, got %s", lisinopril.MedicationName)
		}
		
		if warfarin.MedicationName != "Warfarin" {
			t.Errorf("Expected Warfarin to be ranked third, got %s", warfarin.MedicationName)
		}
		
		// Verify safety scores are in descending order
		if losartan.SafetyScore <= lisinopril.SafetyScore {
			t.Errorf("Expected Losartan safety score (%.2f) > Lisinopril safety score (%.2f)", 
				losartan.SafetyScore, lisinopril.SafetyScore)
		}
		
		if lisinopril.SafetyScore <= warfarin.SafetyScore {
			t.Errorf("Expected Lisinopril safety score (%.2f) > Warfarin safety score (%.2f)", 
				lisinopril.SafetyScore, warfarin.SafetyScore)
		}
		
		// Verify safety scores are reasonable
		if warfarin.SafetyScore >= 0.5 {
			t.Errorf("Expected Warfarin to have low safety score due to black box warning and pregnancy X, got %.2f", 
				warfarin.SafetyScore)
		}
		
		t.Logf("Safety scores - Losartan: %.2f, Lisinopril: %.2f, Warfarin: %.2f", 
			losartan.SafetyScore, lisinopril.SafetyScore, warfarin.SafetyScore)
	}
	
	// Validate filtering statistics
	stats := result.FilteringStatistics
	if stats.InitialDrugCount != 3 {
		t.Errorf("Expected initial drug count 3, got %d", stats.InitialDrugCount)
	}
	
	if stats.FinalCandidateCount != 3 {
		t.Errorf("Expected final candidate count 3, got %d", stats.FinalCandidateCount)
	}
	
	t.Logf("Enhanced test passed: %d candidates with safety scoring", len(result.CandidateProposals))
}

// TestEnhancedCandidateBuilder_PregnancyFiltering tests pregnancy-specific filtering
func TestEnhancedCandidateBuilder_PregnancyFiltering(t *testing.T) {
	config := DefaultBuilderConfig()
	builder := NewCandidateBuilderWithConfig(config, nil)
	
	// Create test data with pregnancy category X drug
	testDrugs := []Drug{
		{
			Code:                  "WAR001",
			Name:                  "Warfarin",
			TherapeuticClasses:    []string{"ANTICOAGULANT"},
			Contraindications:     []string{},
			ContraindicationCodes: []string{},
			EfficacyScore:         90.0,
			PregnancyCategory:     "X", // Contraindicated in pregnancy
			BlackBoxWarning:       true,
		},
		{
			Code:                  "HEP001",
			Name:                  "Heparin",
			TherapeuticClasses:    []string{"ANTICOAGULANT"},
			Contraindications:     []string{},
			ContraindicationCodes: []string{},
			EfficacyScore:         85.0,
			PregnancyCategory:     "B", // Safe in pregnancy
			BlackBoxWarning:       false,
		},
	}
	
	// Pregnant patient
	testPatientFlags := map[string]bool{
		"is_pregnant": true,
	}
	
	input := CandidateBuilderInput{
		RequestID:              "test-pregnancy-123",
		PatientID:              "test-pregnant-patient",
		RecommendedDrugClasses: []string{"ANTICOAGULANT"},
		PatientFlags:           testPatientFlags,
		ActiveMedications:      []ActiveMedication{},
		DrugMasterList:         testDrugs,
		DDIRules:              []DrugInteraction{},
	}
	
	result, err := builder.BuildCandidateProposals(context.Background(), input)
	if err != nil {
		t.Fatalf("BuildCandidateProposals failed: %v", err)
	}
	
	// Should exclude Warfarin due to pregnancy category X
	// Should include Heparin (pregnancy category B)
	expectedCandidates := 1
	if len(result.CandidateProposals) != expectedCandidates {
		t.Errorf("Expected %d candidates for pregnant patient, got %d", expectedCandidates, len(result.CandidateProposals))
	}
	
	if len(result.CandidateProposals) > 0 {
		if result.CandidateProposals[0].MedicationName != "Heparin" {
			t.Errorf("Expected Heparin to be the only candidate, got %s", result.CandidateProposals[0].MedicationName)
		}
	}
	
	t.Logf("Pregnancy filtering test passed: %d safe candidates for pregnant patient", len(result.CandidateProposals))
}
