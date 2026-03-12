package orb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestORBBrainFunctionality tests THE BRAIN - the core ORB functionality
func TestORBBrainFunctionality(t *testing.T) {
	// This test validates that THE BRAIN can make intelligent clinical decisions
	
	// Note: This test requires the knowledge base files to exist
	// In a real environment, we would use the actual knowledge base path
	knowledgeBasePath := "../../../knowledge"
	
	// Skip test if knowledge base not available (for CI/CD)
	orb, err := NewOrchestratorRuleBase(knowledgeBasePath)
	if err != nil {
		t.Skipf("Skipping ORB test - knowledge base not available: %v", err)
		return
	}
	
	require.NotNil(t, orb, "ORB should be initialized")
	
	t.Run("Vancomycin with Renal Impairment - THE BRAIN Test", func(t *testing.T) {
		// Test the most critical functionality: intelligent routing
		request := &MedicationRequest{
			RequestID:         "test-001",
			PatientID:         "patient-123",
			MedicationCode:    "vancomycin",
			MedicationName:    "Vancomycin",
			Indication:        "sepsis",
			PatientConditions: []string{"chronic_kidney_disease"},
			Timestamp:         time.Now(),
		}
		
		// Execute THE BRAIN
		intentManifest, err := orb.ExecuteLocal(context.Background(), request)
		
		// Validate THE BRAIN made the correct decision
		require.NoError(t, err, "ORB should successfully evaluate vancomycin with renal impairment")
		require.NotNil(t, intentManifest, "Intent Manifest should be generated")
		
		// Verify intelligent routing decision
		assert.Equal(t, "vancomycin-renal-v2", intentManifest.RecipeID, 
			"THE BRAIN should select renal-adjusted vancomycin recipe")
		
		// Verify data requirements intelligence
		expectedDataRequirements := []string{"creatinine_clearance", "current_weight", "age", "dialysis_status", "baseline_hearing"}
		assert.ElementsMatch(t, expectedDataRequirements, intentManifest.DataRequirements,
			"THE BRAIN should request exactly the right clinical data")
		
		// Verify clinical rationale
		assert.Contains(t, intentManifest.ClinicalRationale, "renal",
			"THE BRAIN should provide renal-specific rationale")
		
		// Verify priority assignment
		assert.Equal(t, "high", intentManifest.Priority,
			"THE BRAIN should assign high priority to renal impairment")
		
		// Verify request tracking
		assert.Equal(t, request.RequestID, intentManifest.RequestID)
		assert.Equal(t, request.PatientID, intentManifest.PatientID)
		assert.Equal(t, request.MedicationCode, intentManifest.MedicationCode)
	})
	
	t.Run("Vancomycin without Renal Issues - Standard Routing", func(t *testing.T) {
		request := &MedicationRequest{
			RequestID:         "test-002",
			PatientID:         "patient-456",
			MedicationCode:    "vancomycin",
			MedicationName:    "Vancomycin",
			Indication:        "endocarditis",
			PatientConditions: []string{}, // No renal conditions
			Timestamp:         time.Now(),
		}
		
		intentManifest, err := orb.ExecuteLocal(context.Background(), request)
		
		require.NoError(t, err, "ORB should handle standard vancomycin")
		require.NotNil(t, intentManifest, "Intent Manifest should be generated")
		
		// Should select standard vancomycin recipe (lower priority rule)
		assert.Equal(t, "vancomycin-standard-v1", intentManifest.RecipeID,
			"THE BRAIN should select standard vancomycin recipe for normal renal function")
		
		// Should require less data
		assert.Contains(t, intentManifest.DataRequirements, "current_weight")
		assert.Contains(t, intentManifest.DataRequirements, "age")
		assert.NotContains(t, intentManifest.DataRequirements, "dialysis_status",
			"Standard dosing shouldn't require dialysis status")
	})
	
	t.Run("Warfarin with Genetic Testing Available", func(t *testing.T) {
		request := &MedicationRequest{
			RequestID:      "test-003",
			PatientID:      "patient-789",
			MedicationCode: "warfarin",
			MedicationName: "Warfarin",
			Indication:     "atrial_fibrillation",
			ClinicalContext: map[string]interface{}{
				"genetic_testing_available": true,
			},
			Timestamp: time.Now(),
		}
		
		intentManifest, err := orb.ExecuteLocal(context.Background(), request)
		
		require.NoError(t, err, "ORB should handle warfarin with genetic testing")
		require.NotNil(t, intentManifest, "Intent Manifest should be generated")
		
		// Should select genetic-guided warfarin recipe
		assert.Equal(t, "warfarin-initiation-v2", intentManifest.RecipeID,
			"THE BRAIN should select genetic-guided warfarin when testing available")
		
		// Should request genetic data
		assert.Contains(t, intentManifest.DataRequirements, "cyp2c9_genotype")
		assert.Contains(t, intentManifest.DataRequirements, "vkorc1_genotype")
	})
	
	t.Run("Acetaminophen for Pediatric Patient", func(t *testing.T) {
		patientAge := 10.0
		request := &MedicationRequest{
			RequestID:      "test-004",
			PatientID:      "patient-child",
			MedicationCode: "acetaminophen",
			MedicationName: "Acetaminophen",
			Indication:     "fever",
			PatientAge:     &patientAge,
			Timestamp:      time.Now(),
		}
		
		intentManifest, err := orb.ExecuteLocal(context.Background(), request)
		
		require.NoError(t, err, "ORB should handle pediatric acetaminophen")
		require.NotNil(t, intentManifest, "Intent Manifest should be generated")
		
		// Should select pediatric recipe
		assert.Equal(t, "acetaminophen-pediatric-v1", intentManifest.RecipeID,
			"THE BRAIN should select pediatric acetaminophen recipe for children")
	})
	
	t.Run("Unknown Medication - Proper Rejection", func(t *testing.T) {
		request := &MedicationRequest{
			RequestID:      "test-005",
			PatientID:      "patient-unknown",
			MedicationCode: "unknown_drug_xyz",
			MedicationName: "Unknown Drug",
			Timestamp:      time.Now(),
		}

		intentManifest, err := orb.ExecuteLocal(context.Background(), request)

		require.Error(t, err, "ORB should reject unknown medications for clinical safety")
		require.Nil(t, intentManifest, "No Intent Manifest should be generated for unknown drugs")

		// Should contain proper error message
		assert.Contains(t, err.Error(), "not found in knowledge base",
			"Error should indicate medication not in knowledge base")
		assert.Contains(t, err.Error(), "clinical safety",
			"Error should mention clinical safety rationale")
	})
	
	t.Run("ORB Performance Metrics", func(t *testing.T) {
		// Test that THE BRAIN tracks its own performance
		metrics := orb.GetEvaluationMetrics()
		
		assert.NotNil(t, metrics, "ORB should track evaluation metrics")
		assert.Greater(t, metrics.TotalEvaluations, int64(0), "Should have evaluation count")
		assert.Greater(t, metrics.SuccessfulMatches, int64(0), "Should have successful matches")
		assert.NotNil(t, metrics.RuleHitCounts, "Should track rule hit counts")
	})
	
	t.Run("ORB Rule Management", func(t *testing.T) {
		// Test THE BRAIN's self-awareness
		availableRules := orb.GetAvailableRules()
		
		assert.NotEmpty(t, availableRules, "ORB should know its available rules")
		assert.Contains(t, availableRules, "vancomycin-renal-impairment", 
			"Should include vancomycin renal rule")
		
		// Test rule retrieval
		rule, err := orb.GetRuleByID("vancomycin-renal-impairment")
		require.NoError(t, err, "Should retrieve rule by ID")
		assert.Equal(t, "vancomycin", rule.MedicationCode, "Rule should match medication")
	})
}

// TestIntentManifestValidation tests Intent Manifest validation
func TestIntentManifestValidation(t *testing.T) {
	t.Run("Valid Intent Manifest", func(t *testing.T) {
		manifest := NewIntentManifestBuilder().
			WithRequestInfo("req-001", "patient-001").
			WithRecipe("test-recipe-v1").
			WithDataRequirements([]string{"weight", "age"}).
			WithPriority("medium").
			WithRationale("Test rationale").
			Build()
		
		err := manifest.Validate()
		assert.NoError(t, err, "Valid manifest should pass validation")
	})
	
	t.Run("Invalid Intent Manifest - Missing Recipe", func(t *testing.T) {
		manifest := NewIntentManifestBuilder().
			WithRequestInfo("req-001", "patient-001").
			WithDataRequirements([]string{"weight", "age"}).
			WithPriority("medium").
			Build()
		
		err := manifest.Validate()
		assert.Error(t, err, "Manifest without recipe should fail validation")
		assert.Contains(t, err.Error(), "recipe ID", "Error should mention missing recipe ID")
	})
}

// BenchmarkORBEvaluation benchmarks THE BRAIN performance
func BenchmarkORBEvaluation(b *testing.B) {
	knowledgeBasePath := "../../../knowledge"
	
	orb, err := NewOrchestratorRuleBase(knowledgeBasePath)
	if err != nil {
		b.Skipf("Skipping benchmark - knowledge base not available: %v", err)
		return
	}
	
	request := &MedicationRequest{
		RequestID:         "bench-001",
		PatientID:         "patient-bench",
		MedicationCode:    "vancomycin",
		PatientConditions: []string{"chronic_kidney_disease"},
		Timestamp:         time.Now(),
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := orb.ExecuteLocal(context.Background(), request)
		if err != nil {
			b.Fatalf("ORB evaluation failed: %v", err)
		}
	}
}
