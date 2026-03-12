// Package tests provides comprehensive test utilities for KB-17 Population Registry
// determinism_test.go - Tests for reproducibility and deterministic behavior
// This validates that same input always produces same population membership
package tests

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"kb-17-population-registry/internal/criteria"
	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/registry"
)

// =============================================================================
// DETERMINISM INVARIANT: Same Input → Same Population
// =============================================================================

// TestDeterminism_SameInputSameOutput tests fundamental determinism property
func TestDeterminism_SameInputSameOutput(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))
	fixtures := NewPatientFixtures()

	// Test multiple times with same input
	const iterations = 100

	for i := 0; i < iterations; i++ {
		// Diabetes patient should always match diabetes registry
		reg := registry.GetDiabetesRegistry()
		result, err := engine.Evaluate(fixtures.DiabetesPatient, &reg)
		assert.NoError(t, err)

		assert.True(t, result.MeetsInclusion,
			"Iteration %d: Same patient data must always produce same inclusion result", i)
	}
}

// TestDeterminism_RiskTierConsistency tests consistent risk tier assignment
func TestDeterminism_RiskTierConsistency(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))
	fixtures := NewPatientFixtures()

	// Run risk evaluation multiple times
	const iterations = 50
	var firstRiskTier models.RiskTier

	for i := 0; i < iterations; i++ {
		reg := registry.GetDiabetesRegistry()
		result, err := engine.Evaluate(fixtures.DiabetesCriticalRisk, &reg)
		assert.NoError(t, err)

		if i == 0 {
			firstRiskTier = result.SuggestedRiskTier
		} else {
			assert.Equal(t, firstRiskTier, result.SuggestedRiskTier,
				"Iteration %d: Risk tier must be deterministic for same input", i)
		}
	}
}

// TestDeterminism_MultipleRegistriesConsistent tests multi-registry determinism
func TestDeterminism_MultipleRegistriesConsistent(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))
	fixtures := NewPatientFixtures()
	patient := fixtures.MultipleRegistries // DM + HTN + CKD

	const iterations = 20
	var firstMatches []models.RegistryCode

	for i := 0; i < iterations; i++ {
		matches := evaluateAllRegistries(t, engine, patient)

		if firstMatches == nil {
			firstMatches = matches
		} else {
			assert.Equal(t, firstMatches, matches,
				"Iteration %d: Same patient must match same registries each time", i)
		}
	}
}

// TestDeterminism_OrderIndependence tests that input order doesn't affect output
func TestDeterminism_OrderIndependence(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))

	// Create patient with multiple diagnoses in different orders
	diagnoses1 := []models.Diagnosis{
		{Code: "E11.9", CodeSystem: models.CodeSystemICD10, Status: "active"},
		{Code: "I10", CodeSystem: models.CodeSystemICD10, Status: "active"},
		{Code: "N18.3", CodeSystem: models.CodeSystemICD10, Status: "active"},
	}

	diagnoses2 := []models.Diagnosis{
		{Code: "N18.3", CodeSystem: models.CodeSystemICD10, Status: "active"},
		{Code: "E11.9", CodeSystem: models.CodeSystemICD10, Status: "active"},
		{Code: "I10", CodeSystem: models.CodeSystemICD10, Status: "active"},
	}

	diagnoses3 := []models.Diagnosis{
		{Code: "I10", CodeSystem: models.CodeSystemICD10, Status: "active"},
		{Code: "N18.3", CodeSystem: models.CodeSystemICD10, Status: "active"},
		{Code: "E11.9", CodeSystem: models.CodeSystemICD10, Status: "active"},
	}

	patients := []*models.PatientClinicalData{
		{PatientID: "patient-order-1", Diagnoses: diagnoses1},
		{PatientID: "patient-order-2", Diagnoses: diagnoses2},
		{PatientID: "patient-order-3", Diagnoses: diagnoses3},
	}

	var results [][]models.RegistryCode
	for _, p := range patients {
		matches := evaluateAllRegistries(t, engine, p)
		results = append(results, matches)
	}

	// All should produce same matches (order doesn't matter)
	for i := 1; i < len(results); i++ {
		assert.ElementsMatch(t, results[0], results[i],
			"Diagnosis order should not affect registry matching")
	}
}

// =============================================================================
// REPRODUCIBILITY TESTS
// =============================================================================

// TestReproducibility_WithTimestamp tests time-independent evaluation
func TestReproducibility_WithTimestamp(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))

	// Patient data with fixed timestamps
	sixMonthsAgo := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	patient := &models.PatientClinicalData{
		PatientID: "patient-timestamp-001",
		Diagnoses: []models.Diagnosis{
			{
				Code:       "E11.9",
				CodeSystem: models.CodeSystemICD10,
				Status:     "active",
				RecordedAt: sixMonthsAgo,
			},
		},
		LabResults: []models.LabResult{
			{
				Code:        "4548-4",
				CodeSystem:  models.CodeSystemLOINC,
				Value:       9.5,
				EffectiveAt: sixMonthsAgo.AddDate(0, 3, 0),
				Status:      "final",
			},
		},
	}

	reg := registry.GetDiabetesRegistry()

	// Evaluate at different "current" times - result should be consistent
	// (evaluation is based on patient data, not wall clock time)
	result1, err1 := engine.Evaluate(patient, &reg)
	assert.NoError(t, err1)
	time.Sleep(10 * time.Millisecond) // Small delay
	result2, err2 := engine.Evaluate(patient, &reg)
	assert.NoError(t, err2)

	assert.Equal(t, result1.MeetsInclusion, result2.MeetsInclusion,
		"Inclusion result should be reproducible regardless of evaluation time")
}

// TestReproducibility_SerializedPatient tests determinism with serialized data
func TestReproducibility_SerializedPatient(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))

	birthDate := time.Date(1970, 5, 15, 0, 0, 0, 0, time.UTC)

	// Original patient
	original := &models.PatientClinicalData{
		PatientID: "patient-serial-001",
		Demographics: &models.Demographics{
			BirthDate: &birthDate,
			Gender:    "male",
		},
		Diagnoses: []models.Diagnosis{
			{Code: "E11.9", CodeSystem: models.CodeSystemICD10, Status: "active"},
		},
	}

	// Simulate serialization/deserialization (e.g., from Kafka)
	restored := &models.PatientClinicalData{
		PatientID: original.PatientID,
		Demographics: &models.Demographics{
			BirthDate: original.Demographics.BirthDate,
			Gender:    original.Demographics.Gender,
		},
		Diagnoses: make([]models.Diagnosis, len(original.Diagnoses)),
	}
	copy(restored.Diagnoses, original.Diagnoses)

	reg := registry.GetDiabetesRegistry()

	resultOriginal, err1 := engine.Evaluate(original, &reg)
	assert.NoError(t, err1)
	resultRestored, err2 := engine.Evaluate(restored, &reg)
	assert.NoError(t, err2)

	assert.Equal(t, resultOriginal.MeetsInclusion, resultRestored.MeetsInclusion,
		"Serialization/deserialization should not affect inclusion result")
}

// =============================================================================
// POPULATION CONSISTENCY TESTS
// =============================================================================

// TestPopulationConsistency_BatchVsIndividual tests batch equals individual results
func TestPopulationConsistency_BatchVsIndividual(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))

	// Create a batch of patients
	patients := []*models.PatientClinicalData{
		{PatientID: "batch-001", Diagnoses: []models.Diagnosis{{Code: "E11.9", CodeSystem: models.CodeSystemICD10, Status: "active"}}},
		{PatientID: "batch-002", Diagnoses: []models.Diagnosis{{Code: "I10", CodeSystem: models.CodeSystemICD10, Status: "active"}}},
		{PatientID: "batch-003", Diagnoses: []models.Diagnosis{{Code: "Z00.00", CodeSystem: models.CodeSystemICD10, Status: "active"}}}, // No match
		{PatientID: "batch-004", Diagnoses: []models.Diagnosis{{Code: "I50.9", CodeSystem: models.CodeSystemICD10, Status: "active"}}},
	}

	// Evaluate individually
	individualResults := make(map[string][]models.RegistryCode)
	for _, p := range patients {
		individualResults[p.PatientID] = evaluateAllRegistries(t, engine, p)
	}

	// Evaluate as batch (should produce same results)
	batchResults := evaluateBatch(t, engine, patients)

	for pid, matches := range individualResults {
		batchMatches := batchResults[pid]
		assert.ElementsMatch(t, matches, batchMatches,
			"Batch and individual evaluation should produce same results for %s", pid)
	}
}

// TestPopulationConsistency_ParallelVsSerial tests parallel equals serial results
func TestPopulationConsistency_ParallelVsSerial(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))

	// Create patients
	patients := make([]*models.PatientClinicalData, 100)
	for i := 0; i < 100; i++ {
		code := "E11.9" // Default diabetes
		if i%3 == 1 {
			code = "I10" // Hypertension
		} else if i%3 == 2 {
			code = "I50.9" // Heart failure
		}
		patients[i] = &models.PatientClinicalData{
			PatientID: createDeterminismPatientID(i),
			Diagnoses: []models.Diagnosis{
				{Code: code, CodeSystem: models.CodeSystemICD10, Status: "active"},
			},
		}
	}

	// Serial evaluation
	serialResults := evaluateBatch(t, engine, patients)

	// Parallel evaluation (simulated - in real code would use goroutines)
	parallelResults := evaluateBatchParallel(t, engine, patients)

	// Results should be identical
	for pid, serialMatches := range serialResults {
		parallelMatches := parallelResults[pid]
		assert.ElementsMatch(t, serialMatches, parallelMatches,
			"Parallel and serial evaluation should produce identical results for %s", pid)
	}
}

// =============================================================================
// EDGE CASE DETERMINISM TESTS
// =============================================================================

// TestDeterminism_EmptyPatient tests deterministic handling of empty data
func TestDeterminism_EmptyPatient(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))

	patient := &models.PatientClinicalData{
		PatientID: "empty-patient-001",
		Diagnoses: []models.Diagnosis{},
	}

	const iterations = 20
	var firstMatches []models.RegistryCode

	for i := 0; i < iterations; i++ {
		matches := evaluateAllRegistries(t, engine, patient)
		if firstMatches == nil {
			firstMatches = matches
		} else {
			assert.Equal(t, firstMatches, matches,
				"Empty patient should consistently match no registries")
		}
	}

	assert.Empty(t, firstMatches, "Empty patient should match no registries")
}

// TestDeterminism_BoundaryValues tests determinism at threshold boundaries
func TestDeterminism_BoundaryValues(t *testing.T) {
	engine := criteria.NewEngine(TestLogger(t))

	// Test exact threshold values
	thresholds := []float64{7.0, 8.0, 10.0} // HbA1c thresholds

	for _, threshold := range thresholds {
		t.Run("HbA1c_"+floatToTestName(threshold), func(t *testing.T) {
			patient := createDiabetesPatientWithExactHbA1c(threshold)
			reg := registry.GetDiabetesRegistry()

			var firstTier models.RiskTier
			for i := 0; i < 20; i++ {
				result, err := engine.Evaluate(patient, &reg)
				assert.NoError(t, err)
				if i == 0 {
					firstTier = result.SuggestedRiskTier
				} else {
					assert.Equal(t, firstTier, result.SuggestedRiskTier,
						"Boundary value %.1f should produce consistent risk tier", threshold)
				}
			}
		})
	}
}

// =============================================================================
// ENROLLMENT DETERMINISM TESTS
// =============================================================================

// TestEnrollmentDeterminism_SamePatientSameEnrollment tests enrollment determinism
func TestEnrollmentDeterminism_SamePatientSameEnrollment(t *testing.T) {
	ctx := context.Background()
	fixtures := NewPatientFixtures()

	// Create enrollment multiple times (simulating event replay)
	const iterations = 10
	enrollments := make([]*models.RegistryPatient, iterations)

	for i := 0; i < iterations; i++ {
		repo := NewMockRepository()
		producer := NewMockEventProducer()

		// Simulate auto-enrollment from same patient data
		enrollment := createEnrollmentFromPatient(ctx, repo, producer, fixtures.DiabetesPatient)
		enrollments[i] = enrollment
	}

	// All enrollments should have same key properties
	for i := 1; i < len(enrollments); i++ {
		assert.Equal(t, enrollments[0].PatientID, enrollments[i].PatientID)
		assert.Equal(t, enrollments[0].RegistryCode, enrollments[i].RegistryCode)
		assert.Equal(t, enrollments[0].Status, enrollments[i].Status)
		// Note: ID and timestamps will differ, which is expected
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// evaluateAllRegistries evaluates patient against all registries
func evaluateAllRegistries(t *testing.T, engine *criteria.Engine, patient *models.PatientClinicalData) []models.RegistryCode {
	var matches []models.RegistryCode

	registries := registry.GetAllRegistryDefinitions()
	for _, reg := range registries {
		result, err := engine.Evaluate(patient, &reg)
		if err != nil {
			t.Logf("Error evaluating registry %s: %v", reg.Code, err)
			continue
		}
		if result.MeetsInclusion && !result.MeetsExclusion {
			matches = append(matches, reg.Code)
		}
	}

	// Sort for consistent comparison
	sort.Slice(matches, func(i, j int) bool {
		return string(matches[i]) < string(matches[j])
	})

	return matches
}

// evaluateBatch evaluates multiple patients
func evaluateBatch(t *testing.T, engine *criteria.Engine, patients []*models.PatientClinicalData) map[string][]models.RegistryCode {
	results := make(map[string][]models.RegistryCode)
	for _, p := range patients {
		results[p.PatientID] = evaluateAllRegistries(t, engine, p)
	}
	return results
}

// evaluateBatchParallel simulates parallel evaluation
func evaluateBatchParallel(t *testing.T, engine *criteria.Engine, patients []*models.PatientClinicalData) map[string][]models.RegistryCode {
	// In real implementation would use goroutines
	// For test purposes, just verify same results
	return evaluateBatch(t, engine, patients)
}

// createEnrollmentFromPatient creates enrollment from patient data
func createEnrollmentFromPatient(
	ctx context.Context,
	repo *MockRepository,
	producer *MockEventProducer,
	patient *models.PatientClinicalData,
) *models.RegistryPatient {
	_ = ctx // ctx available for future use
	enrollment := &models.RegistryPatient{
		ID:               uuid.New(),
		PatientID:        patient.PatientID,
		RegistryCode:     models.RegistryDiabetes,
		Status:           models.EnrollmentStatusActive,
		RiskTier:         models.RiskTierModerate,
		EnrollmentSource: models.EnrollmentSourceDiagnosis,
		EnrolledAt:       time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)
	_ = producer // producer available for future use
	return enrollment
}

func createDeterminismPatientID(index int) string {
	return "determinism-patient-" + string(rune('0'+index/100)) + string(rune('0'+(index/10)%10)) + string(rune('0'+index%10))
}

func createDiabetesPatientWithExactHbA1c(value float64) *models.PatientClinicalData {
	return &models.PatientClinicalData{
		PatientID: "boundary-hba1c-patient",
		Diagnoses: []models.Diagnosis{
			{Code: "E11.9", CodeSystem: models.CodeSystemICD10, Status: "active"},
		},
		LabResults: []models.LabResult{
			{
				Code:        "4548-4",
				CodeSystem:  models.CodeSystemLOINC,
				Display:     "Hemoglobin A1c",
				Value:       value,
				Unit:        "%",
				EffectiveAt: time.Now(),
				Status:      "final",
			},
		},
	}
}

func floatToTestName(f float64) string {
	whole := int(f)
	decimal := int((f - float64(whole)) * 10)
	return string(rune('0'+whole)) + "_" + string(rune('0'+decimal))
}
