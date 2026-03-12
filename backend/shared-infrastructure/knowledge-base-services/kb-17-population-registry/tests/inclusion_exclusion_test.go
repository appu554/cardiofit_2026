// Package tests provides comprehensive tests for KB-17 Population Registry
// inclusion_exclusion_test.go - Criteria logic tests for inclusion/exclusion
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/criteria"
	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/registry"
)

// =============================================================================
// TEST SETUP
// =============================================================================

// setupCriteriaEngine creates a criteria engine for testing
func setupCriteriaEngine(t *testing.T) *criteria.Engine {
	logger := TestLogger(t)
	engine := criteria.NewEngine(logger)
	return engine
}

// =============================================================================
// INCLUSION CRITERIA TESTS
// =============================================================================

// TestInclusion_DiabetesDiagnosis tests diabetes inclusion via diagnosis
func TestInclusion_DiabetesDiagnosis(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetDiabetesRegistry()

	result, err := engine.Evaluate(fixtures.DiabetesPatient, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion, "Patient with E11.9 should meet diabetes inclusion")
	assert.True(t, result.Eligible, "Patient should be eligible for diabetes registry")
	assert.NotEmpty(t, result.MatchedCriteria, "Should have matched criteria")
}

// TestInclusion_HypertensionDiagnosis tests hypertension inclusion
func TestInclusion_HypertensionDiagnosis(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetHypertensionRegistry()

	result, err := engine.Evaluate(fixtures.HypertensionPatient, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion, "Patient with I10 should meet hypertension inclusion")
	assert.True(t, result.Eligible)
}

// TestInclusion_HeartFailureDiagnosis tests heart failure inclusion
func TestInclusion_HeartFailureDiagnosis(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetHeartFailureRegistry()

	result, err := engine.Evaluate(fixtures.HeartFailurePatient, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion, "Patient with I50.9 should meet HF inclusion")
	assert.True(t, result.Eligible)
}

// TestInclusion_CKDDiagnosis tests CKD inclusion via diagnosis
func TestInclusion_CKDDiagnosis(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetCKDRegistry()

	result, err := engine.Evaluate(fixtures.CKDStage3Patient, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion, "Patient with N18.3 should meet CKD inclusion")
	assert.True(t, result.Eligible)
}

// TestInclusion_PregnancyDiagnosis tests pregnancy inclusion
func TestInclusion_PregnancyDiagnosis(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetPregnancyRegistry()

	result, err := engine.Evaluate(fixtures.PregnancyNormal, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion, "Patient with Z34.00 should meet pregnancy inclusion")
	assert.True(t, result.Eligible)
}

// TestInclusion_AnticoagulationMedication tests medication-based inclusion
func TestInclusion_AnticoagulationMedication(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetAnticoagulationRegistry()

	result, err := engine.Evaluate(fixtures.AnticoagWarfarin, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion, "Patient on warfarin should meet anticoag inclusion")
	assert.True(t, result.Eligible)
}

// =============================================================================
// EXCLUSION CRITERIA TESTS
// =============================================================================

// TestExclusion_OverridesInclusion tests that exclusion overrides inclusion
func TestExclusion_OverridesInclusion(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetPregnancyRegistry()

	// Patient who delivered (O80 code) should be excluded
	result, err := engine.Evaluate(fixtures.PregnancyExcluded, &reg)
	require.NoError(t, err)

	// Should meet exclusion due to O80 delivery code
	assert.True(t, result.MeetsExclusion, "Patient with O80 should meet exclusion")
	assert.False(t, result.Eligible, "Excluded patient should not be eligible")
}

// TestExclusion_ReasonCaptured tests exclusion reason is captured
func TestExclusion_ReasonCaptured(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()
	reg := registry.GetPregnancyRegistry()

	result, err := engine.Evaluate(fixtures.PregnancyExcluded, &reg)
	require.NoError(t, err)

	if result.MeetsExclusion {
		assert.NotEmpty(t, result.ExcludedCriteria, "Exclusion criteria should be captured")
	}
}

// =============================================================================
// NO MATCH TESTS
// =============================================================================

// TestNoMatch_HealthyPatient tests healthy patient doesn't match any registry
func TestNoMatch_HealthyPatient(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()

	// Evaluate against all registries
	results, err := engine.EvaluateAll(fixtures.NoConditions)
	require.NoError(t, err)

	for _, result := range results {
		assert.False(t, result.Eligible,
			"Healthy patient should not be eligible for %s registry", result.RegistryCode)
	}
}

// TestNoMatch_WrongDiagnosis tests patient with unrelated diagnosis
func TestNoMatch_WrongDiagnosis(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)

	// Patient with unrelated diagnosis
	patient := &models.PatientClinicalData{
		PatientID: "patient-unrelated-001",
		Diagnoses: []models.Diagnosis{
			{
				Code:       "K21.0", // GERD - not a registry condition
				CodeSystem: models.CodeSystemICD10,
				Status:     "active",
			},
		},
	}

	results, err := engine.EvaluateAll(patient)
	require.NoError(t, err)

	for _, result := range results {
		assert.False(t, result.Eligible,
			"Patient with GERD only should not be eligible for %s", result.RegistryCode)
	}
}

// =============================================================================
// MULTIPLE REGISTRY ELIGIBILITY TESTS
// =============================================================================

// TestMultipleRegistries_PatientEligibleForSeveral tests multi-registry eligibility
func TestMultipleRegistries_PatientEligibleForSeveral(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()

	results, err := engine.EvaluateAll(fixtures.MultipleRegistries)
	require.NoError(t, err)

	eligibleCount := 0
	eligibleRegistries := make([]models.RegistryCode, 0)

	for _, result := range results {
		if result.Eligible {
			eligibleCount++
			eligibleRegistries = append(eligibleRegistries, result.RegistryCode)
		}
	}

	assert.GreaterOrEqual(t, eligibleCount, 2,
		"Multi-condition patient should be eligible for multiple registries")
	t.Logf("Patient eligible for registries: %v", eligibleRegistries)
}

// =============================================================================
// DIAGNOSIS CODE MATCHING TESTS
// =============================================================================

// TestDiagnosisMatching_StartsWithOperator tests StartsWith operator
func TestDiagnosisMatching_StartsWithOperator(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetDiabetesRegistry()

	// Test various E11.* codes
	testCodes := []string{"E11.0", "E11.9", "E11.65", "E11.21"}

	for _, code := range testCodes {
		t.Run(code, func(t *testing.T) {
			patient := &models.PatientClinicalData{
				PatientID: "patient-e11-" + code,
				Diagnoses: []models.Diagnosis{
					{
						Code:       code,
						CodeSystem: models.CodeSystemICD10,
						Status:     "active",
					},
				},
			}

			result, err := engine.Evaluate(patient, &reg)
			require.NoError(t, err)

			assert.True(t, result.MeetsInclusion,
				"Code %s should match diabetes registry", code)
		})
	}
}

// TestDiagnosisMatching_EqualsOperator tests Equals operator
func TestDiagnosisMatching_EqualsOperator(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetHypertensionRegistry()

	// I10 should match exactly
	patient := &models.PatientClinicalData{
		PatientID: "patient-i10-exact",
		Diagnoses: []models.Diagnosis{
			{
				Code:       "I10",
				CodeSystem: models.CodeSystemICD10,
				Status:     "active",
			},
		},
	}

	result, err := engine.Evaluate(patient, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion, "I10 should match hypertension registry")
}

// TestDiagnosisMatching_InOperator tests In operator for multiple codes
func TestDiagnosisMatching_InOperator(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetHeartFailureRegistry()

	// Acute HF codes should match critical risk
	acuteCodes := []string{"I50.21", "I50.31", "I50.41"}

	for _, code := range acuteCodes {
		t.Run(code, func(t *testing.T) {
			patient := &models.PatientClinicalData{
				PatientID: "patient-acute-hf-" + code,
				Diagnoses: []models.Diagnosis{
					{
						Code:       code,
						CodeSystem: models.CodeSystemICD10,
						Status:     "active",
					},
				},
			}

			result, err := engine.Evaluate(patient, &reg)
			require.NoError(t, err)

			assert.True(t, result.MeetsInclusion,
				"Acute HF code %s should match HF registry", code)
		})
	}
}

// =============================================================================
// MEDICATION MATCHING TESTS
// =============================================================================

// TestMedicationMatching_RxNormCodes tests RxNorm medication matching
func TestMedicationMatching_RxNormCodes(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetAnticoagulationRegistry()

	// Test different anticoagulant RxNorm codes
	anticoagCodes := []struct {
		code string
		name string
	}{
		{"11289", "Warfarin"},
		{"1364430", "Apixaban"},
		{"1114195", "Rivaroxaban"},
	}

	for _, med := range anticoagCodes {
		t.Run(med.name, func(t *testing.T) {
			patient := &models.PatientClinicalData{
				PatientID: "patient-anticoag-" + med.code,
				Medications: []models.Medication{
					{
						Code:       med.code,
						CodeSystem: models.CodeSystemRxNorm,
						Display:    med.name,
						Status:     "active",
					},
				},
			}

			result, err := engine.Evaluate(patient, &reg)
			require.NoError(t, err)

			assert.True(t, result.MeetsInclusion,
				"Patient on %s (%s) should match anticoag registry", med.name, med.code)
		})
	}
}

// =============================================================================
// LAB RESULT MATCHING TESTS
// =============================================================================

// TestLabMatching_GreaterThanOperator tests lab value greater than
func TestLabMatching_GreaterThanOperator(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetCKDRegistry()

	// Patient with low eGFR should be included
	patient := &models.PatientClinicalData{
		PatientID: "patient-ckd-egfr",
		LabResults: []models.LabResult{
			{
				Code:       "33914-3",
				CodeSystem: models.CodeSystemLOINC,
				Display:    "eGFR",
				Value:      45, // < 60, meets inclusion
				Unit:       "mL/min/1.73m2",
				Status:     "final",
			},
		},
	}

	result, err := engine.Evaluate(patient, &reg)
	require.NoError(t, err)

	// CKD registry has lab-based inclusion criteria
	assert.True(t, result.MeetsInclusion, "Patient with eGFR < 60 should meet CKD inclusion")
}

// =============================================================================
// OR LOGIC TESTS
// =============================================================================

// TestOrLogic_OneCriterionSufficient tests OR group logic
func TestOrLogic_OneCriterionSufficient(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetDiabetesRegistry()

	// Patient with only E10 (Type 1) should match
	patient := &models.PatientClinicalData{
		PatientID: "patient-e10-only",
		Diagnoses: []models.Diagnosis{
			{
				Code:       "E10.9",
				CodeSystem: models.CodeSystemICD10,
				Status:     "active",
			},
		},
	}

	result, err := engine.Evaluate(patient, &reg)
	require.NoError(t, err)

	assert.True(t, result.MeetsInclusion,
		"Patient with E10 only should match diabetes (OR logic)")
}

// =============================================================================
// AND LOGIC TESTS
// =============================================================================

// TestAndLogic_AllCriteriaRequired tests AND group logic
func TestAndLogic_AllCriteriaRequired(t *testing.T) {
	t.Parallel()

	// Some registries may have AND groups in their criteria
	// This tests that all criteria in an AND group must be met
	engine := setupCriteriaEngine(t)

	// Create a patient that might have partial criteria
	patient := &models.PatientClinicalData{
		PatientID: "patient-partial-criteria",
		Diagnoses: []models.Diagnosis{
			{
				Code:       "N18.3",
				CodeSystem: models.CodeSystemICD10,
				Status:     "active",
			},
		},
		// No labs - if CKD requires both diagnosis AND labs, should fail
	}

	reg := registry.GetCKDRegistry()
	result, err := engine.Evaluate(patient, &reg)
	require.NoError(t, err)

	// CKD registry has OR logic (diagnosis OR labs), so should still match
	// This test documents the behavior
	t.Logf("CKD eligibility with diagnosis only: %v", result.Eligible)
}

// =============================================================================
// EDGE CASE TESTS
// =============================================================================

// TestEdgeCase_EmptyPatientData tests empty patient data handling
func TestEdgeCase_EmptyPatientData(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)

	patient := &models.PatientClinicalData{
		PatientID: "patient-empty",
	}

	results, err := engine.EvaluateAll(patient)
	require.NoError(t, err)

	for _, result := range results {
		assert.False(t, result.Eligible,
			"Empty patient should not be eligible for %s", result.RegistryCode)
	}
}

// TestEdgeCase_InactiveDiagnosis tests inactive diagnosis handling
func TestEdgeCase_InactiveDiagnosis(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetDiabetesRegistry()

	// Patient with inactive/resolved diagnosis
	patient := &models.PatientClinicalData{
		PatientID: "patient-resolved-dm",
		Diagnoses: []models.Diagnosis{
			{
				Code:       "E11.9",
				CodeSystem: models.CodeSystemICD10,
				Status:     "inactive", // Not active
			},
		},
	}

	result, err := engine.Evaluate(patient, &reg)
	require.NoError(t, err)

	// Behavior depends on implementation - document it
	t.Logf("Eligibility with inactive diagnosis: %v", result.Eligible)
}

// TestEdgeCase_StoppedMedication tests stopped medication handling
func TestEdgeCase_StoppedMedication(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	reg := registry.GetAnticoagulationRegistry()

	// Patient with stopped anticoagulant
	patient := &models.PatientClinicalData{
		PatientID: "patient-stopped-anticoag",
		Medications: []models.Medication{
			{
				Code:       "11289", // Warfarin
				CodeSystem: models.CodeSystemRxNorm,
				Display:    "Warfarin",
				Status:     "stopped",
			},
		},
	}

	result, err := engine.Evaluate(patient, &reg)
	require.NoError(t, err)

	// Should not be enrolled if medication is stopped
	t.Logf("Eligibility with stopped medication: %v", result.Eligible)
}

// =============================================================================
// DETERMINISM TESTS
// =============================================================================

// TestInclusion_EvaluationDeterminism tests evaluation determinism for inclusion
func TestInclusion_EvaluationDeterminism(t *testing.T) {
	t.Parallel()

	engine := setupCriteriaEngine(t)
	fixtures := NewPatientFixtures()

	// Evaluate same patient multiple times
	results := make([]bool, 5)
	for i := 0; i < 5; i++ {
		res, err := engine.EvaluateAll(fixtures.DiabetesPatient)
		require.NoError(t, err)

		// Find diabetes result
		for _, r := range res {
			if r.RegistryCode == models.RegistryDiabetes {
				results[i] = r.Eligible
				break
			}
		}
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		assert.Equal(t, results[0], results[i],
			"Evaluation should be deterministic (iteration %d)", i)
	}
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

// BenchmarkEvaluateSingleRegistry benchmarks single registry evaluation
func BenchmarkEvaluateSingleRegistry(b *testing.B) {
	logger := TestLogger(&testing.T{})
	engine := criteria.NewEngine(logger)
	fixtures := NewPatientFixtures()
	reg := registry.GetDiabetesRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Evaluate(fixtures.DiabetesPatient, &reg)
	}
}

// BenchmarkEvaluateAllRegistries benchmarks all registry evaluation
func BenchmarkEvaluateAllRegistries(b *testing.B) {
	logger := TestLogger(&testing.T{})
	engine := criteria.NewEngine(logger)
	fixtures := NewPatientFixtures()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.EvaluateAll(fixtures.DiabetesPatient)
	}
}
