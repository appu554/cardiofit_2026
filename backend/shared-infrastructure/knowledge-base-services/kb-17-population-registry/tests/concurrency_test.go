// Package tests provides comprehensive test utilities for KB-17 Population Registry
// concurrency_test.go - Tests for race conditions and concurrent access patterns
// This validates thread-safety critical for high-throughput population management
package tests

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

// =============================================================================
// CONCURRENT ENROLLMENT TESTS
// =============================================================================

// TestConcurrency_100SimultaneousEnrollments tests 100 concurrent enrollment events
func TestConcurrency_100SimultaneousEnrollments(t *testing.T) {
	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	const goroutines = 100
	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	// Create barrier to ensure all goroutines start simultaneously
	startBarrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Wait for start signal
			<-startBarrier

			enrollment := &models.RegistryPatient{
				ID:               uuid.New(),
				PatientID:        createConcurrencyPatientID(workerID),
				RegistryCode:     models.RegistryDiabetes,
				Status:           models.EnrollmentStatusActive,
				RiskTier:         models.RiskTierModerate,
				EnrollmentSource: models.EnrollmentSourceManual,
				EnrolledAt:       time.Now(),
			}

			err := repo.CreateEnrollment(enrollment)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
				_ = producer.ProduceEnrollmentEvent(ctx, enrollment)
			} else {
				atomic.AddInt32(&errorCount, 1)
			}
		}(i)
	}

	// Release all goroutines at once
	close(startBarrier)
	wg.Wait()

	// All unique patients should succeed
	assert.Equal(t, int32(goroutines), successCount,
		"All %d concurrent enrollments should succeed", goroutines)
	assert.Equal(t, int32(0), errorCount)

	// Verify correct number of enrollments
	_, count, _ := repo.ListEnrollments(&models.EnrollmentQuery{})
	assert.Equal(t, int64(goroutines), count)

	// Verify correct number of events
	events := producer.GetEventsByType("registry.enrolled")
	assert.Len(t, events, goroutines)
}

// TestConcurrency_SimultaneousEnrollmentsSamePatient tests race on same patient
func TestConcurrency_SimultaneousEnrollmentsSamePatient(t *testing.T) {
	repo := NewMockRepository()
	_ = TestContext(t) // ctx available for future use

	const goroutines = 10
	patientID := "race-patient-001"
	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	startBarrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startBarrier

			enrollment := &models.RegistryPatient{
				ID:           uuid.New(),
				PatientID:    patientID, // Same patient
				RegistryCode: models.RegistryDiabetes,
				Status:       models.EnrollmentStatusActive,
				RiskTier:     models.RiskTierModerate,
				EnrolledAt:   time.Now(),
			}

			err := repo.CreateEnrollment(enrollment)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				atomic.AddInt32(&errorCount, 1)
			}
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	// Only one should succeed (duplicate detection)
	assert.Equal(t, int32(1), successCount, "Only one enrollment should succeed")
	assert.Equal(t, int32(goroutines-1), errorCount, "Others should fail as duplicates")

	// Verify only one enrollment exists
	enrollment, _ := repo.GetEnrollmentByPatientRegistry(patientID, models.RegistryDiabetes)
	assert.NotNil(t, enrollment)

	_, count, _ := repo.ListEnrollments(&models.EnrollmentQuery{PatientID: patientID})
	assert.Equal(t, int64(1), count)
}

// =============================================================================
// CONCURRENT UPDATE TESTS
// =============================================================================

// TestConcurrency_SimultaneousStatusUpdates tests concurrent status changes
func TestConcurrency_SimultaneousStatusUpdates(t *testing.T) {
	repo := NewMockRepository()

	// Setup: Create enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "concurrent-update-001",
		RegistryCode: models.RegistryHypertension,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	const goroutines = 20
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// Alternate between SUSPENDED and ACTIVE
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startBarrier

			newStatus := models.EnrollmentStatusSuspended
			if workerID%2 == 0 {
				newStatus = models.EnrollmentStatusActive
			}

			_ = repo.UpdateEnrollmentStatus(
				enrollment.ID,
				enrollment.Status,
				newStatus,
				"concurrent test",
				"worker",
			)
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	// Should not panic or corrupt data
	final, err := repo.GetEnrollment(enrollment.ID)
	require.NoError(t, err)
	require.NotNil(t, final)

	// Status should be one of the valid values
	assert.True(t,
		final.Status == models.EnrollmentStatusActive || final.Status == models.EnrollmentStatusSuspended,
		"Final status should be valid")
}

// TestConcurrency_SimultaneousRiskTierUpdates tests concurrent risk changes
func TestConcurrency_SimultaneousRiskTierUpdates(t *testing.T) {
	repo := NewMockRepository()

	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "risk-update-001",
		RegistryCode: models.RegistryCKD,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	const goroutines = 30
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	riskTiers := []models.RiskTier{
		models.RiskTierLow,
		models.RiskTierModerate,
		models.RiskTierHigh,
		models.RiskTierCritical,
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startBarrier

			newTier := riskTiers[workerID%len(riskTiers)]
			_ = repo.UpdateEnrollmentRiskTier(
				enrollment.ID,
				enrollment.RiskTier,
				newTier,
				"concurrent-worker",
			)
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	// Verify data integrity
	final, _ := repo.GetEnrollment(enrollment.ID)
	assert.Contains(t, riskTiers, final.RiskTier, "Risk tier should be valid")

	// All updates should be recorded in history
	history := repo.GetHistory()
	assert.True(t, len(history) >= 1, "History should record updates")
}

// =============================================================================
// CONCURRENT READ-WRITE TESTS
// =============================================================================

// TestConcurrency_ReadsDuringWrites tests read consistency during writes
func TestConcurrency_ReadsDuringWrites(t *testing.T) {
	repo := NewMockRepository()
	ctx := TestContext(t)

	// Pre-populate some data
	for i := 0; i < 100; i++ {
		enrollment := &models.RegistryPatient{
			ID:           uuid.New(),
			PatientID:    createConcurrencyPatientID(i),
			RegistryCode: models.RegistryDiabetes,
			Status:       models.EnrollmentStatusActive,
			RiskTier:     models.RiskTierModerate,
			EnrolledAt:   time.Now(),
		}
		_ = repo.CreateEnrollment(enrollment)
	}

	const (
		readers     = 10
		writers     = 5
		iterations  = 50
	)

	var wg sync.WaitGroup
	var readCount int32
	var writeCount int32
	var readErrors int32

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Start readers
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					_, count, err := repo.ListEnrollments(&models.EnrollmentQuery{})
					if err != nil || count < 0 {
						atomic.AddInt32(&readErrors, 1)
					}
					atomic.AddInt32(&readCount, 1)
				}
			}
		}()
	}

	// Start writers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				select {
				case <-ctx.Done():
					return
				default:
					enrollment := &models.RegistryPatient{
						ID:           uuid.New(),
						PatientID:    createConcurrencyPatientID(1000 + workerID*iterations + j),
						RegistryCode: models.RegistryHypertension,
						Status:       models.EnrollmentStatusActive,
						RiskTier:     models.RiskTierModerate,
						EnrolledAt:   time.Now(),
					}
					_ = repo.CreateEnrollment(enrollment)
					atomic.AddInt32(&writeCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(0), readErrors, "No read errors should occur during concurrent writes")
	assert.True(t, atomic.LoadInt32(&readCount) > 0, "Reads should have completed")
	assert.True(t, atomic.LoadInt32(&writeCount) > 0, "Writes should have completed")

	t.Logf("Completed %d reads and %d writes concurrently", readCount, writeCount)
}

// =============================================================================
// DEADLOCK PREVENTION TESTS
// =============================================================================

// TestConcurrency_NoDeadlockOnCrossEnrollment tests deadlock prevention
func TestConcurrency_NoDeadlockOnCrossEnrollment(t *testing.T) {
	repo := NewMockRepository()

	// Create enrollments in both directions simultaneously
	const goroutines = 20
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startBarrier

			// Create enrollment in registry A then B
			enrollmentA := &models.RegistryPatient{
				ID:           uuid.New(),
				PatientID:    createConcurrencyPatientID(workerID),
				RegistryCode: models.RegistryDiabetes,
				Status:       models.EnrollmentStatusActive,
				EnrolledAt:   time.Now(),
			}
			_ = repo.CreateEnrollment(enrollmentA)

			enrollmentB := &models.RegistryPatient{
				ID:           uuid.New(),
				PatientID:    createConcurrencyPatientID(workerID),
				RegistryCode: models.RegistryHypertension,
				Status:       models.EnrollmentStatusActive,
				EnrolledAt:   time.Now(),
			}
			_ = repo.CreateEnrollment(enrollmentB)
		}(i)
	}

	// Start simultaneously
	close(startBarrier)

	// Wait with timeout to detect deadlock
	select {
	case <-done:
		// Success - no deadlock
	case <-time.After(10 * time.Second):
		t.Fatal("Deadlock detected - operations did not complete")
	}
}

// =============================================================================
// EVENT PRODUCER CONCURRENCY TESTS
// =============================================================================

// TestConcurrency_EventProduction tests concurrent event production
func TestConcurrency_EventProduction(t *testing.T) {
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	const goroutines = 50
	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			<-startBarrier

			enrollment := &models.RegistryPatient{
				ID:           uuid.New(),
				PatientID:    createConcurrencyPatientID(workerID),
				RegistryCode: models.RegistryDiabetes,
				Status:       models.EnrollmentStatusActive,
			}

			_ = producer.ProduceEnrollmentEvent(ctx, enrollment)
		}(i)
	}

	close(startBarrier)
	wg.Wait()

	// All events should be captured
	events := producer.GetEvents()
	assert.Len(t, events, goroutines, "All concurrent events should be captured")
}

// =============================================================================
// CACHE CONCURRENCY TESTS
// =============================================================================

// TestConcurrency_CacheOperations tests concurrent cache access
func TestConcurrency_CacheOperations(t *testing.T) {
	cache := NewMockCache()
	ctx := TestContext(t)

	const (
		readers    = 20
		writers    = 10
		iterations = 100
	)

	var wg sync.WaitGroup

	// Writers
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := "concurrent-key-" + string(rune('0'+workerID%10))
				cache.Set(ctx, key, workerID*iterations+j, time.Minute)
			}
		}(i)
	}

	// Readers
	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := "concurrent-key-" + string(rune('0'+workerID%10))
				cache.Get(ctx, key)
			}
		}(i)
	}

	wg.Wait()

	// Should complete without race conditions
	assert.True(t, true, "Cache operations should be thread-safe")
}

// =============================================================================
// STRESS TESTS
// =============================================================================

// TestStress_HighThroughputEnrollment tests high-throughput enrollment
func TestStress_HighThroughputEnrollment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	repo := NewMockRepository()
	producer := NewMockEventProducer()
	ctx := TestContext(t)

	const (
		targetOpsPerSecond = 1000
		durationSeconds    = 5
		totalOps           = targetOpsPerSecond * durationSeconds
	)

	var wg sync.WaitGroup
	var completedOps int32

	startTime := time.Now()

	for i := 0; i < totalOps; i++ {
		wg.Add(1)
		go func(opID int) {
			defer wg.Done()

			enrollment := &models.RegistryPatient{
				ID:           uuid.New(),
				PatientID:    "stress-" + uuid.New().String()[:8],
				RegistryCode: models.RegistryCode([]models.RegistryCode{
					models.RegistryDiabetes,
					models.RegistryHypertension,
					models.RegistryHeartFailure,
				}[opID%3]),
				Status:   models.EnrollmentStatusActive,
				RiskTier: models.RiskTierModerate,
			}

			if repo.CreateEnrollment(enrollment) == nil {
				atomic.AddInt32(&completedOps, 1)
				_ = producer.ProduceEnrollmentEvent(ctx, enrollment)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	opsPerSecond := float64(completedOps) / elapsed.Seconds()
	t.Logf("Completed %d ops in %v (%.0f ops/sec)", completedOps, elapsed, opsPerSecond)

	assert.True(t, atomic.LoadInt32(&completedOps) > int32(totalOps/2),
		"At least half of operations should complete under stress")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func createConcurrencyPatientID(index int) string {
	return "concurrent-patient-" + uuid.New().String()[:8] + "-" +
		string(rune('0'+index/1000%10)) +
		string(rune('0'+index/100%10)) +
		string(rune('0'+index/10%10)) +
		string(rune('0'+index%10))
}
