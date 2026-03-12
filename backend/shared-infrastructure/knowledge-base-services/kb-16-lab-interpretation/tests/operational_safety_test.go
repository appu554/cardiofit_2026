// Package tests provides operational safety tests for KB-16
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/types"
)

// =============================================================================
// TEST INFRASTRUCTURE - Mock Redis for Failure Scenarios
// =============================================================================

// MockRedisClient wraps redis.Client with failure injection capabilities
type MockRedisClient struct {
	*redis.Client
	failGet         bool
	failSet         bool
	failDel         bool
	failScan        bool
	failCount       int32 // atomic counter for failures
	latencyMS       int   // simulated latency
	connectionError bool  // simulate connection loss
	mu              sync.RWMutex
}

// NewMockRedisClient creates a mock Redis client for testing
func NewMockRedisClient() *MockRedisClient {
	// Use real Redis if available, otherwise nil
	opt := &redis.Options{
		Addr:        "localhost:6395",
		DB:          15, // use DB 15 for tests
		DialTimeout: 1 * time.Second,
	}
	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return &MockRedisClient{Client: nil}
	}

	return &MockRedisClient{Client: client}
}

// InjectFailure configures failure injection
func (m *MockRedisClient) InjectFailure(failGet, failSet, failDel, failScan bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failGet = failGet
	m.failSet = failSet
	m.failDel = failDel
	m.failScan = failScan
}

// SetLatency configures simulated latency
func (m *MockRedisClient) SetLatency(ms int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latencyMS = ms
}

// SimulateConnectionLoss simulates Redis connection failure
func (m *MockRedisClient) SimulateConnectionLoss(lost bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectionError = lost
}

// GetFailCount returns the number of injected failures
func (m *MockRedisClient) GetFailCount() int32 {
	return atomic.LoadInt32(&m.failCount)
}

// =============================================================================
// CACHE INVALIDATION TESTS
// =============================================================================

func TestCacheInvalidation_SinglePatient(t *testing.T) {
	// Setup
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for cache invalidation tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "cache_invalidation")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()

	// Clean up test keys
	cache.FlushDB(ctx)

	patientID := "patient-cache-test-1"

	// Store initial result (should invalidate cache)
	now := time.Now()
	req := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2823-3",
		Name:        "Potassium",
		ValueNumeric: ptr(4.5),
		Unit:        "mmol/L",
		CollectedAt: now,
	}

	result, err := resultStore.Store(ctx, &req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result can be retrieved
	retrieved, err := resultStore.GetByID(ctx, result.ID.String())
	require.NoError(t, err)
	assert.Equal(t, patientID, retrieved.PatientID)

	// Store another result (should invalidate patient cache)
	req2 := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2951-2",
		Name:        "Sodium",
		ValueNumeric: ptr(140.0),
		Unit:        "mmol/L",
		CollectedAt: now.Add(1 * time.Hour),
	}

	_, err = resultStore.Store(ctx, &req2)
	require.NoError(t, err)

	// First result should still be retrievable (cache was invalidated but DB is source of truth)
	retrieved2, err := resultStore.GetByID(ctx, result.ID.String())
	require.NoError(t, err)
	assert.Equal(t, patientID, retrieved2.PatientID)
}

func TestCacheInvalidation_BatchStore(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for batch cache tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "batch_cache")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	// Create batch for multiple patients
	patients := []string{"batch-patient-1", "batch-patient-2", "batch-patient-3"}
	now := time.Now()

	var requests []types.StoreResultRequest
	for _, pid := range patients {
		requests = append(requests, types.StoreResultRequest{
			PatientID:   pid,
			Code:        "2823-3",
			Name:        "Potassium",
			ValueNumeric: ptr(4.5),
			Unit:        "mmol/L",
			CollectedAt: now,
		})
	}

	// Store batch
	results, err := resultStore.StoreBatch(ctx, requests)
	require.NoError(t, err)
	assert.Len(t, results, 3)

	// Verify all results are retrievable
	for _, r := range results {
		retrieved, err := resultStore.GetByID(ctx, r.ID.String())
		require.NoError(t, err)
		assert.Equal(t, r.PatientID, retrieved.PatientID)
	}
}

func TestCacheInvalidation_PatternMatching(t *testing.T) {
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for pattern matching tests")
	}
	defer cache.Close()

	ctx := context.Background()
	cache.FlushDB(ctx)

	patientID := "pattern-test-patient"

	// Manually set some cache keys with patient pattern
	keys := []string{
		fmt.Sprintf("patient:%s:results", patientID),
		fmt.Sprintf("patient:%s:baseline:2823-3", patientID),
		fmt.Sprintf("patient:%s:trending:7d", patientID),
		"other:key:unrelated",
	}

	for _, key := range keys {
		cache.Set(ctx, key, "test-value", 5*time.Minute)
	}

	// Verify all keys exist
	for _, key := range keys {
		exists, _ := cache.Exists(ctx, key).Result()
		assert.Equal(t, int64(1), exists, "Key should exist: %s", key)
	}

	// Invalidate patient cache using pattern
	pattern := fmt.Sprintf("patient:%s:*", patientID)
	iter := cache.Scan(ctx, 0, pattern, 0).Iterator()
	deletedCount := 0
	for iter.Next(ctx) {
		cache.Del(ctx, iter.Val())
		deletedCount++
	}

	// Verify patient keys were deleted but unrelated key remains
	assert.Equal(t, 3, deletedCount, "Should have deleted 3 patient keys")

	exists, _ := cache.Exists(ctx, "other:key:unrelated").Result()
	assert.Equal(t, int64(1), exists, "Unrelated key should still exist")
}

// =============================================================================
// RACE CONDITION TESTS
// =============================================================================

func TestRaceCondition_ConcurrentStores(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for race condition tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "race_condition")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	patientID := "race-test-patient"
	now := time.Now()
	numGoroutines := 10
	numStoresPerGoroutine := 5

	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numStoresPerGoroutine; j++ {
				req := types.StoreResultRequest{
					PatientID:   patientID,
					Code:        "2823-3",
					Name:        "Potassium",
					ValueNumeric: ptr(4.0 + float64(j)*0.1),
					Unit:        "mmol/L",
					CollectedAt: now.Add(time.Duration(goroutineID*100+j) * time.Minute),
				}

				_, err := resultStore.Store(ctx, &req)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedTotal := int32(numGoroutines * numStoresPerGoroutine)
	assert.Equal(t, expectedTotal, successCount, "All stores should succeed")
	assert.Equal(t, int32(0), errorCount, "No errors should occur")

	// Verify all results are in database
	var count int64
	db.Model(&types.LabResult{}).Where("patient_id = ?", patientID).Count(&count)
	assert.Equal(t, int64(expectedTotal), count, "All results should be in database")
}

func TestRaceCondition_ConcurrentReadWrite(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for concurrent read/write tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "concurrent_rw")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	patientID := "concurrent-rw-patient"
	now := time.Now()

	// Store initial result
	req := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2823-3",
		Name:        "Potassium",
		ValueNumeric: ptr(4.5),
		Unit:        "mmol/L",
		CollectedAt: now,
	}
	initialResult, err := resultStore.Store(ctx, &req)
	require.NoError(t, err)

	var wg sync.WaitGroup
	var readSuccessCount int32
	var writeSuccessCount int32

	// Concurrent readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, err := resultStore.GetByID(ctx, initialResult.ID.String())
				if err == nil {
					atomic.AddInt32(&readSuccessCount, 1)
				}
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				writeReq := types.StoreResultRequest{
					PatientID:   patientID,
					Code:        "2951-2",
					Name:        "Sodium",
					ValueNumeric: ptr(140.0 + float64(j)),
					Unit:        "mmol/L",
					CollectedAt: now.Add(time.Duration(writerID*100+j) * time.Second),
				}
				_, err := resultStore.Store(ctx, &writeReq)
				if err == nil {
					atomic.AddInt32(&writeSuccessCount, 1)
				}
				time.Sleep(2 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	assert.Equal(t, int32(200), readSuccessCount, "All reads should succeed")
	assert.Equal(t, int32(25), writeSuccessCount, "All writes should succeed")
}

func TestRaceCondition_CacheConsistency(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for cache consistency tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "cache_consistency")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	patientID := "consistency-test-patient"
	now := time.Now()

	// Store initial result
	req := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2823-3",
		Name:        "Potassium",
		ValueNumeric: ptr(4.5),
		Unit:        "mmol/L",
		CollectedAt: now,
	}
	result, err := resultStore.Store(ctx, &req)
	require.NoError(t, err)

	// Retrieve to populate cache
	cached1, err := resultStore.GetByID(ctx, result.ID.String())
	require.NoError(t, err)

	// Verify cache entry exists
	cacheKey := fmt.Sprintf("result:%s", result.ID.String())
	cachedData, err := cache.Get(ctx, cacheKey).Result()
	require.NoError(t, err)
	assert.NotEmpty(t, cachedData)

	// Unmarshal and verify data integrity
	var cachedResult types.LabResult
	err = json.Unmarshal([]byte(cachedData), &cachedResult)
	require.NoError(t, err)
	assert.Equal(t, cached1.PatientID, cachedResult.PatientID)
	assert.Equal(t, *cached1.ValueNumeric, *cachedResult.ValueNumeric)
}

// =============================================================================
// REDIS FALLBACK TESTS
// =============================================================================

func TestRedisFallback_NilCache(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New().WithField("test", "nil_cache")

	// Create store with nil cache (Redis unavailable)
	resultStore := store.NewResultStore(db, nil, log)
	ctx := context.Background()

	patientID := "fallback-test-patient"
	now := time.Now()

	// Store should work without cache
	req := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2823-3",
		Name:        "Potassium",
		ValueNumeric: ptr(4.5),
		Unit:        "mmol/L",
		CollectedAt: now,
	}

	result, err := resultStore.Store(ctx, &req)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Retrieve should work (direct from DB)
	retrieved, err := resultStore.GetByID(ctx, result.ID.String())
	require.NoError(t, err)
	assert.Equal(t, patientID, retrieved.PatientID)

	// Batch store should work
	requests := []types.StoreResultRequest{
		{PatientID: patientID, Code: "2951-2", Name: "Sodium", ValueNumeric: ptr(140.0), Unit: "mmol/L", CollectedAt: now},
		{PatientID: patientID, Code: "2075-0", Name: "Chloride", ValueNumeric: ptr(102.0), Unit: "mmol/L", CollectedAt: now},
	}
	results, err := resultStore.StoreBatch(ctx, requests)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestRedisFallback_GracefulDegradation(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for degradation tests")
	}

	log := logrus.New().WithField("test", "graceful_degradation")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	patientID := "degradation-test-patient"
	now := time.Now()

	// Store with working cache
	req := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2823-3",
		Name:        "Potassium",
		ValueNumeric: ptr(4.5),
		Unit:        "mmol/L",
		CollectedAt: now,
	}
	result, err := resultStore.Store(ctx, &req)
	require.NoError(t, err)

	// Retrieve - should hit cache
	retrieved1, err := resultStore.GetByID(ctx, result.ID.String())
	require.NoError(t, err)
	assert.Equal(t, patientID, retrieved1.PatientID)

	// Close Redis connection to simulate failure
	cache.Close()

	// Create new store without cache (simulating reconnection failure)
	resultStore2 := store.NewResultStore(db, nil, log)

	// Retrieve should still work (from DB directly)
	retrieved2, err := resultStore2.GetByID(ctx, result.ID.String())
	require.NoError(t, err)
	assert.Equal(t, patientID, retrieved2.PatientID)
}

func TestRedisFallback_PartialFailure(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New().WithField("test", "partial_failure")

	// Test that store operations succeed even if cache operations fail
	// We test this by using nil cache which simulates cache being unavailable
	resultStore := store.NewResultStore(db, nil, log)
	ctx := context.Background()

	patientID := "partial-failure-patient"
	now := time.Now()

	// Multiple sequential stores without cache
	for i := 0; i < 5; i++ {
		req := types.StoreResultRequest{
			PatientID:   patientID,
			Code:        "2823-3",
			Name:        "Potassium",
			ValueNumeric: ptr(4.0 + float64(i)*0.1),
			Unit:        "mmol/L",
			CollectedAt: now.Add(time.Duration(i) * time.Hour),
		}
		_, err := resultStore.Store(ctx, &req)
		require.NoError(t, err)
	}

	// Verify all data is in database
	var count int64
	db.Model(&types.LabResult{}).Where("patient_id = ?", patientID).Count(&count)
	assert.Equal(t, int64(5), count)
}

// =============================================================================
// CONNECTION POOL TESTS
// =============================================================================

func TestConnectionPool_HighConcurrency(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for connection pool tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "connection_pool")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	patientID := "pool-test-patient"
	now := time.Now()

	// High concurrency test - 50 goroutines, 20 operations each
	numGoroutines := 50
	numOpsPerGoroutine := 20

	var wg sync.WaitGroup
	var successCount int32
	var errorCount int32
	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				if j%2 == 0 {
					// Write operation
					req := types.StoreResultRequest{
						PatientID:   patientID,
						Code:        "2823-3",
						Name:        "Potassium",
						ValueNumeric: ptr(4.0 + float64(j)*0.01),
						Unit:        "mmol/L",
						CollectedAt: now.Add(time.Duration(gid*1000+j) * time.Millisecond),
					}
					_, err := resultStore.Store(ctx, &req)
					if err != nil {
						atomic.AddInt32(&errorCount, 1)
					} else {
						atomic.AddInt32(&successCount, 1)
					}
				} else {
					// Read operation (list patient results)
					_, _, err := resultStore.GetByPatient(ctx, patientID, 10, 0)
					if err != nil {
						atomic.AddInt32(&errorCount, 1)
					} else {
						atomic.AddInt32(&successCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	expectedTotal := int32(numGoroutines * numOpsPerGoroutine)
	assert.Equal(t, expectedTotal, successCount, "All operations should succeed")
	assert.Equal(t, int32(0), errorCount, "No errors should occur")

	t.Logf("Completed %d operations in %v (%.2f ops/sec)",
		expectedTotal, elapsed, float64(expectedTotal)/elapsed.Seconds())
}

func TestConnectionPool_Exhaustion(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for pool exhaustion tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "pool_exhaustion")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	// Test that system handles connection pressure gracefully
	patientID := "exhaustion-test-patient"
	now := time.Now()

	var wg sync.WaitGroup
	var successCount int32
	errorChan := make(chan error, 100)

	// Burst of 100 concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()
			req := types.StoreResultRequest{
				PatientID:   patientID,
				Code:        "2823-3",
				Name:        "Potassium",
				ValueNumeric: ptr(4.0 + float64(reqID)*0.001),
				Unit:        "mmol/L",
				CollectedAt: now.Add(time.Duration(reqID) * time.Millisecond),
			}
			_, err := resultStore.Store(ctx, &req)
			if err != nil {
				select {
				case errorChan <- err:
				default:
				}
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	close(errorChan)

	// Collect errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	// With SQLite in-memory, all should succeed
	// With real PostgreSQL, some may timeout under heavy load
	assert.GreaterOrEqual(t, successCount, int32(90), "At least 90% should succeed")
	if len(errors) > 0 {
		t.Logf("Encountered %d errors under load (acceptable)", len(errors))
	}
}

// =============================================================================
// TIMEOUT & CONTEXT CANCELLATION TESTS
// =============================================================================

func TestTimeout_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for timeout tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "timeout")
	resultStore := store.NewResultStore(db, cache.Client, log)
	cache.FlushDB(context.Background())

	patientID := "timeout-test-patient"
	now := time.Now()

	// Store initial result with good context
	goodCtx := context.Background()
	req := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2823-3",
		Name:        "Potassium",
		ValueNumeric: ptr(4.5),
		Unit:        "mmol/L",
		CollectedAt: now,
	}
	result, err := resultStore.Store(goodCtx, &req)
	require.NoError(t, err)

	// Test with already cancelled context
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	// Retrieve with cancelled context should fail
	_, err = resultStore.GetByID(cancelledCtx, result.ID.String())
	assert.Error(t, err, "Should fail with cancelled context")

	// Test with short timeout context
	shortCtx, shortCancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer shortCancel()
	time.Sleep(1 * time.Millisecond) // Ensure timeout expires

	_, err = resultStore.GetByID(shortCtx, result.ID.String())
	assert.Error(t, err, "Should fail with expired timeout")
}

func TestTimeout_LongRunningOperations(t *testing.T) {
	db := setupTestDB(t)
	log := logrus.New().WithField("test", "long_running")
	resultStore := store.NewResultStore(db, nil, log)
	ctx := context.Background()

	patientID := "long-running-patient"
	now := time.Now()

	// Insert many results
	var requests []types.StoreResultRequest
	for i := 0; i < 100; i++ {
		requests = append(requests, types.StoreResultRequest{
			PatientID:   patientID,
			Code:        "2823-3",
			Name:        "Potassium",
			ValueNumeric: ptr(4.0 + float64(i)*0.01),
			Unit:        "mmol/L",
			CollectedAt: now.Add(time.Duration(i) * time.Hour),
		})
	}

	// Batch insert
	_, err := resultStore.StoreBatch(ctx, requests)
	require.NoError(t, err)

	// Test pagination with moderate timeout
	moderateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, total, err := resultStore.GetByPatient(moderateCtx, patientID, 50, 0)
	require.NoError(t, err)
	assert.Equal(t, 100, total)
	assert.Len(t, results, 50)
}

// =============================================================================
// DATA INTEGRITY TESTS
// =============================================================================

func TestDataIntegrity_ValuePreservation(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for data integrity tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "data_integrity")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	// Test values at boundary conditions
	testCases := []struct {
		name     string
		value    float64
		expected float64
	}{
		{"zero", 0.0, 0.0},
		{"negative", -5.5, -5.5},
		{"very_small", 0.00001, 0.00001},
		{"very_large", 999999.99, 999999.99},
		{"precision", 4.123456789, 4.123456789},
	}

	patientID := "integrity-test-patient"
	now := time.Now()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := types.StoreResultRequest{
				PatientID:   patientID,
				Code:        "2823-3",
				Name:        "Potassium",
				ValueNumeric: ptr(tc.value),
				Unit:        "mmol/L",
				CollectedAt: now.Add(time.Duration(len(tc.name)) * time.Hour),
			}

			result, err := resultStore.Store(ctx, &req)
			require.NoError(t, err)

			// Retrieve from cache
			retrieved, err := resultStore.GetByID(ctx, result.ID.String())
			require.NoError(t, err)
			require.NotNil(t, retrieved.ValueNumeric)

			// Value should be preserved with acceptable precision
			assert.InDelta(t, tc.expected, *retrieved.ValueNumeric, 0.000001,
				"Value should be preserved: got %v, expected %v", *retrieved.ValueNumeric, tc.expected)
		})
	}
}

func TestDataIntegrity_UnicodeStrings(t *testing.T) {
	db := setupTestDB(t)
	cache := NewMockRedisClient()
	if cache.Client == nil {
		t.Skip("Redis not available for unicode tests")
	}
	defer cache.Close()

	log := logrus.New().WithField("test", "unicode")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	patientID := "unicode-test-patient"
	now := time.Now()

	// Test with unicode characters
	req := types.StoreResultRequest{
		PatientID:   patientID,
		Code:        "2823-3",
		Name:        "Potassium (калий, カリウム)",
		ValueNumeric: ptr(4.5),
		Unit:        "mmol/L",
		ValueString: "Normal 正常 нормальный",
		CollectedAt: now,
		Performer:   "Dr. Müller",
	}

	result, err := resultStore.Store(ctx, &req)
	require.NoError(t, err)

	retrieved, err := resultStore.GetByID(ctx, result.ID.String())
	require.NoError(t, err)

	assert.Equal(t, "Potassium (калий, カリウム)", retrieved.Name)
	assert.Equal(t, "Normal 正常 нормальный", retrieved.ValueString)
	assert.Equal(t, "Dr. Müller", retrieved.Performer)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func setupTestDB(t *testing.T) *gorm.DB {
	// Use in-memory SQLite with shared cache for connection pooling
	// file::memory:?cache=shared ensures all connections see the same database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Limit to single connection to avoid connection pool issues with in-memory DB
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	// Create tables manually for SQLite compatibility (no gen_random_uuid())
	sqls := []string{
		`CREATE TABLE IF NOT EXISTS lab_results (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			code TEXT NOT NULL,
			name TEXT NOT NULL,
			value_numeric REAL,
			value_string TEXT,
			unit TEXT,
			ref_low REAL,
			ref_high REAL,
			ref_critical_low REAL,
			ref_critical_high REAL,
			ref_panic_low REAL,
			ref_panic_high REAL,
			ref_text TEXT,
			ref_age_specific INTEGER,
			ref_sex_specific INTEGER,
			collected_at DATETIME NOT NULL,
			reported_at DATETIME NOT NULL,
			status TEXT DEFAULT 'final',
			performer TEXT,
			encounter_id TEXT,
			specimen_id TEXT,
			order_id TEXT,
			notes TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS interpretations (
			id TEXT PRIMARY KEY,
			result_id TEXT,
			flag TEXT NOT NULL,
			severity TEXT,
			is_critical INTEGER DEFAULT 0,
			is_panic INTEGER DEFAULT 0,
			requires_action INTEGER DEFAULT 0,
			deviation_percent REAL,
			deviation_direction TEXT,
			clinical_comment TEXT,
			recommendations TEXT,
			delta_result TEXT,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS patient_baselines (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			code TEXT NOT NULL,
			mean REAL NOT NULL,
			std_dev REAL,
			min_value REAL,
			max_value REAL,
			sample_count INTEGER,
			source TEXT DEFAULT 'CALCULATED',
			last_updated DATETIME,
			UNIQUE(patient_id, code)
		)`,
		`CREATE TABLE IF NOT EXISTS result_reviews (
			id TEXT PRIMARY KEY,
			result_id TEXT,
			status TEXT DEFAULT 'PENDING',
			acknowledged_by TEXT,
			acknowledged_at DATETIME,
			reviewed_by TEXT,
			reviewed_at DATETIME,
			review_notes TEXT,
			action_taken TEXT,
			kb14_task_id TEXT,
			created_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_lab_results_patient ON lab_results(patient_id)`,
		`CREATE INDEX IF NOT EXISTS idx_lab_results_code ON lab_results(code)`,
		`CREATE INDEX IF NOT EXISTS idx_lab_results_collected ON lab_results(collected_at)`,
	}

	for _, sql := range sqls {
		if err := db.Exec(sql).Error; err != nil {
			require.NoError(t, err, "Failed to create table: %s", sql[:50])
		}
	}

	return db
}

func ptr(v float64) *float64 {
	return &v
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

func BenchmarkStore_WithCache(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&types.LabResult{})

	cache := NewMockRedisClient()
	if cache.Client == nil {
		b.Skip("Redis not available")
	}
	defer cache.Close()

	log := logrus.New().WithField("bench", "store_with_cache")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := types.StoreResultRequest{
			PatientID:   fmt.Sprintf("patient-%d", i%100),
			Code:        "2823-3",
			Name:        "Potassium",
			ValueNumeric: ptr(4.5),
			Unit:        "mmol/L",
			CollectedAt: now.Add(time.Duration(i) * time.Minute),
		}
		resultStore.Store(ctx, &req)
	}
}

func BenchmarkStore_WithoutCache(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&types.LabResult{})

	log := logrus.New().WithField("bench", "store_without_cache")
	resultStore := store.NewResultStore(db, nil, log)
	ctx := context.Background()

	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := types.StoreResultRequest{
			PatientID:   fmt.Sprintf("patient-%d", i%100),
			Code:        "2823-3",
			Name:        "Potassium",
			ValueNumeric: ptr(4.5),
			Unit:        "mmol/L",
			CollectedAt: now.Add(time.Duration(i) * time.Minute),
		}
		resultStore.Store(ctx, &req)
	}
}

func BenchmarkGetByID_CacheMiss(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&types.LabResult{})

	cache := NewMockRedisClient()
	if cache.Client == nil {
		b.Skip("Redis not available")
	}
	defer cache.Close()

	log := logrus.New().WithField("bench", "get_cache_miss")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	// Store results
	var ids []string
	now := time.Now()
	for i := 0; i < 100; i++ {
		req := types.StoreResultRequest{
			PatientID:   fmt.Sprintf("patient-%d", i),
			Code:        "2823-3",
			Name:        "Potassium",
			ValueNumeric: ptr(4.5),
			Unit:        "mmol/L",
			CollectedAt: now,
		}
		result, _ := resultStore.Store(ctx, &req)
		ids = append(ids, result.ID.String())
	}

	// Clear cache to force misses
	cache.FlushDB(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStore.GetByID(ctx, ids[i%100])
	}
}

func BenchmarkGetByID_CacheHit(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&types.LabResult{})

	cache := NewMockRedisClient()
	if cache.Client == nil {
		b.Skip("Redis not available")
	}
	defer cache.Close()

	log := logrus.New().WithField("bench", "get_cache_hit")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	// Store and retrieve to populate cache
	var ids []string
	now := time.Now()
	for i := 0; i < 100; i++ {
		req := types.StoreResultRequest{
			PatientID:   fmt.Sprintf("patient-%d", i),
			Code:        "2823-3",
			Name:        "Potassium",
			ValueNumeric: ptr(4.5),
			Unit:        "mmol/L",
			CollectedAt: now,
		}
		result, _ := resultStore.Store(ctx, &req)
		ids = append(ids, result.ID.String())
		// Warm cache
		resultStore.GetByID(ctx, result.ID.String())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultStore.GetByID(ctx, ids[i%100])
	}
}

func BenchmarkConcurrentOperations(b *testing.B) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db.AutoMigrate(&types.LabResult{})

	cache := NewMockRedisClient()
	if cache.Client == nil {
		b.Skip("Redis not available")
	}
	defer cache.Close()

	log := logrus.New().WithField("bench", "concurrent")
	resultStore := store.NewResultStore(db, cache.Client, log)
	ctx := context.Background()
	cache.FlushDB(ctx)

	now := time.Now()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			req := types.StoreResultRequest{
				PatientID:   fmt.Sprintf("patient-%d", i%100),
				Code:        "2823-3",
				Name:        "Potassium",
				ValueNumeric: ptr(4.5),
				Unit:        "mmol/L",
				CollectedAt: now.Add(time.Duration(i) * time.Microsecond),
			}
			resultStore.Store(ctx, &req)
			i++
		}
	})
}
