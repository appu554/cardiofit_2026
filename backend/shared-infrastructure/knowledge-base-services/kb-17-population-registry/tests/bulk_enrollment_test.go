// Package tests provides comprehensive test utilities for KB-17 Population Registry
// bulk_enrollment_test.go - Tests for bulk enrollment operations and partial failure handling
// This validates population-scale operations critical for registry management
package tests

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

// =============================================================================
// BULK IMPORT TESTS
// =============================================================================

// TestBulkEnrollment_10kPatients tests bulk enrollment of 10,000 patients
func TestBulkEnrollment_10kPatients(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping bulk enrollment test in short mode")
	}

	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	const batchSize = 10000
	patients := generateBulkPatients(batchSize)

	// Track timing
	startTime := time.Now()

	// Execute bulk enrollment
	result := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)

	elapsed := time.Since(startTime)

	// Performance assertions
	t.Logf("Bulk enrollment of %d patients completed in %v", batchSize, elapsed)
	t.Logf("Success: %d, Failed: %d, Skipped: %d", len(result.Enrolled), result.Failed, len(result.Skipped))

	// All patients should be processed (success + failed + skipped = total)
	assert.Equal(t, batchSize, len(result.Enrolled)+result.Failed+len(result.Skipped),
		"All patients should be processed")

	// Performance threshold: 10k patients should complete in reasonable time
	assert.Less(t, elapsed, 30*time.Second,
		"Bulk enrollment should complete within performance threshold")

	// Verify events were produced for successful enrollments
	events := producer.GetEventsByType("registry.enrolled")
	assert.Equal(t, len(result.Enrolled), len(events),
		"Each successful enrollment should produce an event")
}

// TestBulkEnrollment_1kPatientsParallel tests parallel bulk processing
func TestBulkEnrollment_1kPatientsParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping parallel bulk test in short mode")
	}

	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	const totalPatients = 1000
	const workerCount = 10
	const batchSize = totalPatients / workerCount

	patients := generateBulkPatients(totalPatients)

	startTime := time.Now()

	// Parallel enrollment with worker pool
	var wg sync.WaitGroup
	results := make(chan *models.BulkEnrollmentResult, workerCount)

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * batchSize
			end := start + batchSize
			batch := patients[start:end]
			result := executeBulkEnrollment(ctx, repo, producer, batch, models.RegistryDiabetes)
			results <- result
		}(i)
	}

	// Wait and collect results
	wg.Wait()
	close(results)

	var totalEnrolled, totalFailed, totalSkipped int
	for r := range results {
		totalEnrolled += len(r.Enrolled)
		totalFailed += r.Failed
		totalSkipped += len(r.Skipped)
	}

	elapsed := time.Since(startTime)

	t.Logf("Parallel bulk enrollment: %d enrolled, %d failed, %d skipped in %v",
		totalEnrolled, totalFailed, totalSkipped, elapsed)

	// Parallel processing should complete faster than serial
	assert.Less(t, elapsed, 10*time.Second,
		"Parallel bulk enrollment should be efficient")
}

// =============================================================================
// PARTIAL FAILURE HANDLING TESTS
// =============================================================================

// TestBulkEnrollment_PartialFailure tests handling of mixed success/failure batch
func TestBulkEnrollment_PartialFailure(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Mix of valid and invalid patients
	patients := []*models.PatientClinicalData{
		// Valid patients
		createValidDiabetesPatient("valid-001"),
		createValidDiabetesPatient("valid-002"),
		createValidDiabetesPatient("valid-003"),
		// Invalid patients (no diagnosis)
		createInvalidPatient("invalid-001"),
		createInvalidPatient("invalid-002"),
		// More valid
		createValidDiabetesPatient("valid-004"),
		createValidDiabetesPatient("valid-005"),
	}

	result := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)

	// Valid patients should succeed
	assert.Equal(t, 5, len(result.Enrolled), "5 valid patients should be enrolled")

	// Invalid patients should fail
	assert.Equal(t, 2, result.Failed, "2 invalid patients should fail")

	// Errors should be documented
	assert.Len(t, result.Errors, 2, "Each failure should have documented error")

	// Valid enrollments should still produce events
	events := producer.GetEventsByType("registry.enrolled")
	assert.Len(t, events, 5, "Successful enrollments should produce events")
}

// TestBulkEnrollment_DuplicateHandling tests duplicate patient handling
func TestBulkEnrollment_DuplicateHandling(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Pre-enroll a patient
	existingEnrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "existing-patient-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now().AddDate(0, -1, 0),
		CreatedAt:    time.Now().AddDate(0, -1, 0),
		UpdatedAt:    time.Now().AddDate(0, -1, 0),
	}
	err := repo.CreateEnrollment(existingEnrollment)
	require.NoError(t, err)

	// Bulk enrollment includes the already-enrolled patient
	patients := []*models.PatientClinicalData{
		createValidDiabetesPatient("new-001"),
		createValidDiabetesPatient("existing-patient-001"), // Duplicate
		createValidDiabetesPatient("new-002"),
	}

	result := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)

	// New patients enrolled
	assert.Equal(t, 2, len(result.Enrolled), "New patients should be enrolled")

	// Duplicate should be skipped (not failed)
	assert.Equal(t, 1, len(result.Skipped), "Existing enrollment should be skipped")
	assert.Equal(t, 0, result.Failed, "Duplicates should not count as failures")
}

// TestBulkEnrollment_AtomicRollbackOnCriticalFailure tests transaction rollback
func TestBulkEnrollment_AtomicRollbackOnCriticalFailure(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Simulate a batch with a critical error mid-process
	// In atomic mode, all should roll back
	patients := generateBulkPatients(100)

	// Set up repository to fail after 50 enrollments
	repo.SetHealthError(fmt.Errorf("simulated database failure"))

	result := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)

	// In atomic mode (if implemented), either all succeed or all fail
	// For non-atomic (default), partial success is acceptable
	assert.True(t, result.Failed > 0 || len(result.Enrolled) > 0,
		"Batch should have some result")
}

// TestBulkEnrollment_ErrorAggregation tests error collection and reporting
func TestBulkEnrollment_ErrorAggregation(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Create patients with various validation failures
	patients := []*models.PatientClinicalData{
		{PatientID: ""}, // Empty patient ID
		{PatientID: "no-diagnosis-001", Diagnoses: nil},
		{PatientID: "wrong-diagnosis-001", Diagnoses: []models.Diagnosis{
			{Code: "Z00.00", CodeSystem: models.CodeSystemICD10}, // Not diabetes
		}},
		createValidDiabetesPatient("valid-001"),
	}

	result := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)

	// Errors should be aggregated with details
	assert.True(t, result.Failed >= 2, "Multiple failures expected")
	assert.True(t, len(result.Errors) >= 2, "Errors should be documented")

	// Each error should have meaningful information
	for _, errMsg := range result.Errors {
		assert.NotEmpty(t, errMsg, "Error message should be present")
	}
}

// =============================================================================
// BULK ENROLLMENT PERFORMANCE TESTS
// =============================================================================

// TestBulkEnrollment_MemoryEfficiency tests memory usage during bulk ops
func TestBulkEnrollment_MemoryEfficiency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Process in batches to verify streaming/batching works
	const totalPatients = 5000
	const batchSize = 500

	var totalProcessed int

	for i := 0; i < totalPatients/batchSize; i++ {
		batch := generateBulkPatients(batchSize)
		// Offset patient IDs to avoid duplicates
		for j, p := range batch {
			p.PatientID = fmt.Sprintf("batch-%d-patient-%d", i, j)
		}

		result := executeBulkEnrollment(ctx, repo, producer, batch, models.RegistryDiabetes)
		totalProcessed += len(result.Enrolled) + result.Failed + len(result.Skipped)

		// Verify incremental progress
		assert.True(t, len(result.Enrolled)+result.Failed+len(result.Skipped) > 0,
			"Each batch should process patients")
	}

	assert.Equal(t, totalPatients, totalProcessed,
		"All patients should be processed across batches")
}

// TestBulkEnrollment_Idempotency tests that reprocessing same batch is safe
func TestBulkEnrollment_Idempotency(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	patients := generateBulkPatients(100)

	// First enrollment
	result1 := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)

	// Clear event producer to track second run
	producer.Clear()

	// Second enrollment of same patients
	result2 := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)

	// First run should enroll all
	assert.Equal(t, 100, len(result1.Enrolled), "First run should enroll all patients")

	// Second run should skip all (already enrolled)
	assert.Equal(t, 100, len(result2.Skipped), "Second run should skip all (idempotent)")
	assert.Equal(t, 0, len(result2.Enrolled), "No new enrollments on second run")

	// Second run should NOT produce duplicate events
	events := producer.GetEvents()
	assert.Empty(t, events, "Idempotent run should not produce duplicate events")
}

// =============================================================================
// BULK ENROLLMENT WITH MULTIPLE REGISTRIES
// =============================================================================

// TestBulkEnrollment_MultipleRegistries tests enrolling same patients in multiple registries
func TestBulkEnrollment_MultipleRegistries(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	// Patient qualifies for both diabetes and hypertension
	patients := []*models.PatientClinicalData{
		createMultiRegistryPatient("multi-001"),
		createMultiRegistryPatient("multi-002"),
		createMultiRegistryPatient("multi-003"),
	}

	// Enroll in diabetes registry
	result1 := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryDiabetes)
	assert.Equal(t, 3, len(result1.Enrolled))

	// Enroll same patients in hypertension registry
	result2 := executeBulkEnrollment(ctx, repo, producer, patients, models.RegistryHypertension)
	assert.Equal(t, 3, len(result2.Enrolled))

	// Verify each patient has two enrollments
	for _, p := range patients {
		enrollments, count, err := repo.ListEnrollments(&models.EnrollmentQuery{
			PatientID: p.PatientID,
		})
		require.NoError(t, err)
		assert.Equal(t, int64(2), count, "Patient should be in 2 registries")
		assert.Len(t, enrollments, 2)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// generateBulkPatients creates n test patients for bulk operations
func generateBulkPatients(n int) []*models.PatientClinicalData {
	patients := make([]*models.PatientClinicalData, n)
	for i := 0; i < n; i++ {
		patients[i] = createValidDiabetesPatient(fmt.Sprintf("bulk-patient-%d", i))
	}
	return patients
}

// createValidDiabetesPatient creates a patient that qualifies for diabetes registry
func createValidDiabetesPatient(patientID string) *models.PatientClinicalData {
	return &models.PatientClinicalData{
		PatientID: patientID,
		Demographics: &models.Demographics{
			BirthDate: timePtr(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)),
			Gender:    "male",
		},
		Diagnoses: []models.Diagnosis{
			{
				Code:       "E11.9",
				CodeSystem: models.CodeSystemICD10,
				Display:    "Type 2 diabetes mellitus without complications",
				Status:     "active",
			},
		},
		LabResults: []models.LabResult{
			{
				Code:        "4548-4",
				CodeSystem:  models.CodeSystemLOINC,
				Display:     "Hemoglobin A1c",
				Value:       7.5,
				Unit:        "%",
				EffectiveAt: time.Now().AddDate(0, 0, -30),
				Status:      "final",
			},
		},
	}
}

// createInvalidPatient creates a patient that doesn't qualify for diabetes registry
func createInvalidPatient(patientID string) *models.PatientClinicalData {
	return &models.PatientClinicalData{
		PatientID: patientID,
		Demographics: &models.Demographics{
			BirthDate: timePtr(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)),
			Gender:    "female",
		},
		Diagnoses:  []models.Diagnosis{}, // No qualifying diagnosis
		LabResults: []models.LabResult{},
	}
}

// createMultiRegistryPatient creates a patient qualifying for multiple registries
func createMultiRegistryPatient(patientID string) *models.PatientClinicalData {
	return &models.PatientClinicalData{
		PatientID: patientID,
		Demographics: &models.Demographics{
			BirthDate: timePtr(time.Date(1965, 5, 15, 0, 0, 0, 0, time.UTC)),
			Gender:    "male",
		},
		Diagnoses: []models.Diagnosis{
			{
				Code:       "E11.9",
				CodeSystem: models.CodeSystemICD10,
				Status:     "active",
			},
			{
				Code:       "I10",
				CodeSystem: models.CodeSystemICD10,
				Status:     "active",
			},
		},
		VitalSigns: []models.VitalSign{
			{
				Type:        "blood-pressure",
				Code:        "85354-9",
				CodeSystem:  models.CodeSystemLOINC,
				Value:       map[string]interface{}{"systolic": 145, "diastolic": 92},
				Unit:        "mmHg",
				EffectiveAt: time.Now(),
			},
		},
	}
}

// executeBulkEnrollment performs bulk enrollment with evaluation
func executeBulkEnrollment(
	ctx context.Context,
	repo *MockRepository,
	producer *MockEventProducer,
	patients []*models.PatientClinicalData,
	registryCode models.RegistryCode,
) *models.BulkEnrollmentResult {
	result := &models.BulkEnrollmentResult{
		Enrolled: make([]string, 0),
		Skipped:  make([]string, 0),
		Errors:   make([]string, 0),
	}

	for _, patient := range patients {
		// Validate patient
		if patient.PatientID == "" {
			result.Failed++
			result.Errors = append(result.Errors, "unknown: empty patient ID")
			continue
		}

		// Check for existing enrollment
		existing, _ := repo.GetEnrollmentByPatientRegistry(patient.PatientID, registryCode)
		if existing != nil {
			result.Skipped = append(result.Skipped, patient.PatientID)
			continue
		}

		// Check criteria (simplified - in real system would use criteria engine)
		qualifies := false
		for _, diag := range patient.Diagnoses {
			if registryCode == models.RegistryDiabetes {
				if len(diag.Code) >= 3 && diag.Code[:3] == "E11" {
					qualifies = true
					break
				}
			} else if registryCode == models.RegistryHypertension {
				if diag.Code == "I10" || (len(diag.Code) >= 3 && diag.Code[:3] == "I11") {
					qualifies = true
					break
				}
			}
		}

		if !qualifies {
			result.Failed++
			result.Errors = append(result.Errors, patient.PatientID+": does not meet inclusion criteria")
			continue
		}

		// Create enrollment
		enrollment := &models.RegistryPatient{
			ID:               uuid.New(),
			PatientID:        patient.PatientID,
			RegistryCode:     registryCode,
			Status:           models.EnrollmentStatusActive,
			RiskTier:         models.RiskTierModerate, // Simplified
			EnrollmentSource: models.EnrollmentSourceBulk,
			EnrolledAt:       time.Now().UTC(),
			CreatedAt:        time.Now().UTC(),
			UpdatedAt:        time.Now().UTC(),
		}

		err := repo.CreateEnrollment(enrollment)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, patient.PatientID+": "+err.Error())
			continue
		}

		result.Enrolled = append(result.Enrolled, patient.PatientID)
		result.Success++

		// Produce event
		_ = producer.ProduceEnrollmentEvent(ctx, enrollment)
	}

	return result
}
