package candidatebuilder

import (
	"context"
	"testing"
	"time"
)

// TestCandidateBuilder_BasicFiltering tests the basic filtering functionality
func TestCandidateBuilder_BasicFiltering(t *testing.T) {
	// Create candidate builder
	builder := NewCandidateBuilder()
	
	// Test health check
	if err := builder.HealthCheck(); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
	
	// Create test data
	testDrugs := []Drug{
		{
			Code:                "LIS001",
			Name:                "Lisinopril",
			TherapeuticClasses:  []string{"ACE_INHIBITOR"},
			Contraindications:   []string{"ANGIOEDEMA_HISTORY", "PREGNANCY"},
			EfficacyScore:       85.0,
			IsGeneric:           true,
		},
		{
			Code:                "LOS001",
			Name:                "Losartan",
			TherapeuticClasses:  []string{"ARB"},
			Contraindications:   []string{"PREGNANCY"},
			EfficacyScore:       88.0,
			IsGeneric:           true,
		},
		{
			Code:                "HCT001",
			Name:                "Hydrochlorothiazide",
			TherapeuticClasses:  []string{"THIAZIDE_DIURETIC"},
			Contraindications:   []string{"SEVERE_KIDNEY_DISEASE"},
			EfficacyScore:       82.0,
			IsGeneric:           true,
		},
		{
			Code:                "MET001",
			Name:                "Metformin",
			TherapeuticClasses:  []string{"ANTIDIABETIC"},
			Contraindications:   []string{"SEVERE_KIDNEY_DISEASE"},
			EfficacyScore:       90.0,
			IsGeneric:           true,
		},
	}
	
	testPatientFlags := map[string]bool{
		"ANGIOEDEMA_HISTORY":        true,  // Should exclude ACE inhibitors (Lisinopril)
		"is_pregnant":               false, // No pregnancy exclusions
		"has_kidney_disease":        false, // No kidney disease exclusions
	}
	
	testActiveMedications := []ActiveMedication{
		{
			MedicationCode: "VAL001",
			Name:           "Valsartan",
			IsActive:       true,
			StartDate:      time.Now().AddDate(0, -6, 0), // Started 6 months ago
		},
	}
	
	testDDIRules := []DrugInteraction{
		{
			ID:          "DDI001",
			Drug1:       "LOS001", // Losartan
			Drug2:       "VAL001", // Valsartan
			Severity:    "Contraindicated",
			Description: "Dual ARB therapy increases hyperkalemia risk",
			Mechanism:   "Additive potassium retention",
		},
	}
	
	// Create input
	input := CandidateBuilderInput{
		RequestID:              "test-request-123",
		PatientID:              "test-patient-456",
		RecommendedDrugClasses: []string{"ACE_INHIBITOR", "ARB", "THIAZIDE_DIURETIC"},
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
	
	// Expected: Only HCTZ should remain
	// - Lisinopril excluded by angioedema history
	// - Losartan excluded by DDI with Valsartan
	// - Metformin excluded by class filter (not in recommended classes)
	// - HCTZ should pass all filters
	
	expectedCandidates := 1
	if len(result.CandidateProposals) != expectedCandidates {
		t.Errorf("Expected %d candidates, got %d", expectedCandidates, len(result.CandidateProposals))
	}
	
	// Check that HCTZ is the remaining candidate
	if len(result.CandidateProposals) > 0 {
		candidate := result.CandidateProposals[0]
		if candidate.MedicationName != "Hydrochlorothiazide" {
			t.Errorf("Expected Hydrochlorothiazide, got %s", candidate.MedicationName)
		}
		
		if candidate.Status != "candidate" {
			t.Errorf("Expected status 'candidate', got %s", candidate.Status)
		}
	}
	
	// Validate filtering statistics
	stats := result.FilteringStatistics
	if stats.InitialDrugCount != 4 {
		t.Errorf("Expected initial drug count 4, got %d", stats.InitialDrugCount)
	}
	
	if stats.FinalCandidateCount != 1 {
		t.Errorf("Expected final candidate count 1, got %d", stats.FinalCandidateCount)
	}
	
	// Check that reduction percentage is calculated correctly
	expectedReduction := 75.0 // 3 out of 4 drugs filtered out
	if stats.OverallReductionPercent != expectedReduction {
		t.Errorf("Expected overall reduction %.1f%%, got %.1f%%", expectedReduction, stats.OverallReductionPercent)
	}
	
	// Validate processing metadata
	if result.ProcessingMetadata.EngineVersion != "v2.0" {
		t.Errorf("Expected engine version v2.0, got %s", result.ProcessingMetadata.EngineVersion)
	}
	
	if len(result.ProcessingMetadata.FilterStagesRun) != 3 {
		t.Errorf("Expected 3 filter stages, got %d", len(result.ProcessingMetadata.FilterStagesRun))
	}
	
	t.Logf("Test passed: %d candidates generated from %d initial drugs (%.1f%% reduction)", 
		len(result.CandidateProposals), stats.InitialDrugCount, stats.OverallReductionPercent)
}

// TestCandidateBuilder_EmptyResults tests handling of empty results
func TestCandidateBuilder_EmptyResults(t *testing.T) {
	builder := NewCandidateBuilder()
	
	// Create test data where all drugs will be filtered out
	testDrugs := []Drug{
		{
			Code:                "LIS001",
			Name:                "Lisinopril",
			TherapeuticClasses:  []string{"ACE_INHIBITOR"},
			Contraindications:   []string{"ANGIOEDEMA_HISTORY"},
		},
		{
			Code:                "ENA001",
			Name:                "Enalapril",
			TherapeuticClasses:  []string{"ACE_INHIBITOR"},
			Contraindications:   []string{"ANGIOEDEMA_HISTORY"},
		},
	}
	
	// Patient has angioedema history - should exclude all ACE inhibitors
	testPatientFlags := map[string]bool{
		"ANGIOEDEMA_HISTORY": true,
	}
	
	input := CandidateBuilderInput{
		RequestID:              "test-empty-123",
		PatientID:              "test-patient-789",
		RecommendedDrugClasses: []string{"ACE_INHIBITOR"}, // Only ACE inhibitors requested
		PatientFlags:           testPatientFlags,
		ActiveMedications:      []ActiveMedication{}, // No active medications
		DrugMasterList:         testDrugs,
		DDIRules:              []DrugInteraction{}, // No DDI rules
	}
	
	// Execute candidate building
	result, err := builder.BuildCandidateProposals(context.Background(), input)
	if err != nil {
		t.Fatalf("BuildCandidateProposals failed: %v", err)
	}
	
	// Should have specialist review proposal
	if len(result.CandidateProposals) != 1 {
		t.Errorf("Expected 1 specialist review proposal, got %d", len(result.CandidateProposals))
	}
	
	if result.CandidateProposals[0].MedicationCode != "CLINICAL_REVIEW_REQUIRED" {
		t.Errorf("Expected clinical review proposal, got %s", result.CandidateProposals[0].MedicationCode)
	}
	
	// Should require specialist review
	if !result.FilteringStatistics.RequiresSpecialistReview {
		t.Error("Expected RequiresSpecialistReview to be true")
	}
	
	// Should have clinical guidance
	if result.ClinicalGuidance == nil {
		t.Error("Expected clinical guidance for empty results")
	}
	
	if result.ClinicalGuidance.Severity != "HIGH" {
		t.Errorf("Expected HIGH severity, got %s", result.ClinicalGuidance.Severity)
	}
	
	t.Logf("Empty results test passed: specialist review required with clinical guidance")
}

// TestCandidateBuilder_InputValidation tests input validation
func TestCandidateBuilder_InputValidation(t *testing.T) {
	builder := NewCandidateBuilder()
	
	// Test with invalid input (missing request ID)
	invalidInput := CandidateBuilderInput{
		RequestID: "", // Missing request ID
		PatientID: "test-patient-123",
	}
	
	_, err := builder.BuildCandidateProposals(context.Background(), invalidInput)
	if err == nil {
		t.Error("Expected validation error for missing request ID")
	}
	
	// Test with nil patient flags
	invalidInput2 := CandidateBuilderInput{
		RequestID:    "test-request-123",
		PatientID:    "test-patient-123",
		PatientFlags: nil, // Nil patient flags
	}
	
	_, err = builder.BuildCandidateProposals(context.Background(), invalidInput2)
	if err == nil {
		t.Error("Expected validation error for nil patient flags")
	}
	
	t.Logf("Input validation test passed: properly rejects invalid inputs")
}
