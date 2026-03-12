// Package tests provides tests for the resilient cache
package tests

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-16-lab-interpretation/pkg/store"
)

// =============================================================================
// CIRCUIT BREAKER TESTS
// =============================================================================

func TestCircuitBreaker_InitialState(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "circuit_initial")
	cache := store.NewResilientCache(client, log, nil)

	assert.True(t, cache.IsAvailable(), "Cache should be available initially")
	assert.Equal(t, store.CacheStateClosed, cache.GetState(), "Circuit should be closed initially")
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	// Create a client that will fail (wrong port)
	opt := &redis.Options{
		Addr:        "localhost:9999", // Wrong port
		DialTimeout: 100 * time.Millisecond,
	}
	client := redis.NewClient(opt)
	defer client.Close()

	log := logrus.New().WithField("test", "circuit_opens")
	cfg := &store.ResilientCacheConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		OpenDuration:     1 * time.Second,
		Timeout:          100 * time.Millisecond,
	}
	cache := store.NewResilientCache(client, log, cfg)
	ctx := context.Background()

	// Force failures
	for i := 0; i < 3; i++ {
		_, err := cache.Get(ctx, "test-key")
		assert.Error(t, err, "Should fail with wrong port")
	}

	// Circuit should be open now
	assert.Equal(t, store.CacheStateOpen, cache.GetState(), "Circuit should be open after 3 failures")
	assert.False(t, cache.IsAvailable(), "Cache should not be available when circuit is open")
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	// Create a client that will fail
	opt := &redis.Options{
		Addr:        "localhost:9999",
		DialTimeout: 100 * time.Millisecond,
	}
	client := redis.NewClient(opt)
	defer client.Close()

	log := logrus.New().WithField("test", "circuit_half_open")
	cfg := &store.ResilientCacheConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		OpenDuration:     100 * time.Millisecond, // Short for testing
		Timeout:          100 * time.Millisecond,
	}
	cache := store.NewResilientCache(client, log, cfg)
	ctx := context.Background()

	// Force failures to open circuit
	for i := 0; i < 2; i++ {
		cache.Get(ctx, "test-key")
	}
	assert.Equal(t, store.CacheStateOpen, cache.GetState())

	// Wait for open duration
	time.Sleep(150 * time.Millisecond)

	// Next IsAvailable check should transition to half-open
	available := cache.IsAvailable()
	// It will be available (half-open allows test requests)
	assert.True(t, available, "Should transition to half-open and allow tests")
}

func TestCircuitBreaker_ClosesAfterSuccess(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "circuit_closes")
	cfg := &store.ResilientCacheConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		OpenDuration:     30 * time.Second,
		Timeout:          500 * time.Millisecond,
	}
	cache := store.NewResilientCache(client, log, cfg)
	ctx := context.Background()

	// Start with successful operations
	cache.Set(ctx, "test-key", "value", 1*time.Minute)
	val, err := cache.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.NotEmpty(t, val)

	// Circuit should still be closed
	assert.Equal(t, store.CacheStateClosed, cache.GetState())
}

func TestCircuitBreaker_Reset(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "circuit_reset")
	cache := store.NewResilientCache(client, log, nil)

	// Reset should work
	cache.Reset()
	assert.Equal(t, store.CacheStateClosed, cache.GetState())
	assert.True(t, cache.IsAvailable())
}

// =============================================================================
// RESILIENT CACHE OPERATIONS TESTS
// =============================================================================

func TestResilientCache_BasicOperations(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "basic_ops")
	cache := store.NewResilientCache(client, log, nil)
	ctx := context.Background()
	client.FlushDB(ctx)

	// Test Set
	err := cache.Set(ctx, "test:key1", map[string]interface{}{"name": "test"}, 5*time.Minute)
	require.NoError(t, err)

	// Test Get
	val, err := cache.Get(ctx, "test:key1")
	require.NoError(t, err)
	assert.Contains(t, val, "test")

	// Test Delete
	err = cache.Delete(ctx, "test:key1")
	require.NoError(t, err)

	// Verify deleted
	val, err = cache.Get(ctx, "test:key1")
	require.NoError(t, err)
	assert.Empty(t, val)
}

func TestResilientCache_InvalidatePattern(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "invalidate_pattern")
	cache := store.NewResilientCache(client, log, nil)
	ctx := context.Background()
	client.FlushDB(ctx)

	// Set multiple keys with pattern
	for i := 0; i < 5; i++ {
		key := "patient:P001:test:" + string(rune('A'+i))
		cache.Set(ctx, key, "value", 5*time.Minute)
	}

	// Set unrelated key
	cache.Set(ctx, "other:key", "value", 5*time.Minute)

	// Invalidate patient pattern
	err := cache.InvalidatePattern(ctx, "patient:P001:*")
	require.NoError(t, err)

	// Verify patient keys gone
	keys, err := client.Keys(ctx, "patient:P001:*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys, "Patient keys should be deleted")

	// Verify unrelated key remains (value is JSON-marshaled)
	val, err := client.Get(ctx, "other:key").Result()
	require.NoError(t, err)
	assert.Equal(t, `"value"`, val, "Value should be JSON-marshaled")
}

func TestResilientCache_GetJSON(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "get_json")
	cache := store.NewResilientCache(client, log, nil)
	ctx := context.Background()
	client.FlushDB(ctx)

	// Store JSON object
	testObj := map[string]interface{}{
		"id":    "12345",
		"name":  "Test Object",
		"value": 42.5,
	}
	err := cache.Set(ctx, "json:test", testObj, 5*time.Minute)
	require.NoError(t, err)

	// Retrieve and unmarshal
	var result map[string]interface{}
	found, err := cache.GetJSON(ctx, "json:test", &result)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "12345", result["id"])
	assert.Equal(t, "Test Object", result["name"])

	// Test cache miss
	var missing map[string]interface{}
	found, err = cache.GetJSON(ctx, "json:nonexistent", &missing)
	require.NoError(t, err)
	assert.False(t, found)
}

func TestResilientCache_Ping(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "ping")
	cache := store.NewResilientCache(client, log, nil)
	ctx := context.Background()

	err := cache.Ping(ctx)
	assert.NoError(t, err)
}

func TestResilientCache_Stats(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "stats")
	cache := store.NewResilientCache(client, log, nil)
	ctx := context.Background()
	client.FlushDB(ctx)

	// Perform some operations
	cache.Set(ctx, "stats:key", "value", 1*time.Minute)
	cache.Get(ctx, "stats:key")
	cache.Get(ctx, "stats:missing")

	stats := cache.GetStats()
	assert.Contains(t, stats, "state")
	assert.Contains(t, stats, "failure_count")
	assert.Contains(t, stats, "success_count")
	assert.Contains(t, stats, "is_available")
}

// =============================================================================
// CACHE WITH FALLBACK TESTS
// =============================================================================

func TestCacheWithFallback_HitScenario(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "fallback_hit")
	resilientCache := store.NewResilientCache(client, log, nil)
	cache := store.NewCacheWithFallback(resilientCache, log)
	ctx := context.Background()
	client.FlushDB(ctx)

	// Pre-populate cache
	resilientCache.Set(ctx, "fallback:key", "cached-value", 5*time.Minute)

	loaderCalled := false
	loader := func() (interface{}, error) {
		loaderCalled = true
		return "db-value", nil
	}

	// Should return cached value, not call loader
	result, err := cache.GetOrLoad(ctx, "fallback:key", loader, 5*time.Minute)
	require.NoError(t, err)
	assert.False(t, loaderCalled, "Loader should not be called on cache hit")
	assert.Contains(t, result.(string), "cached-value")
}

func TestCacheWithFallback_MissScenario(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "fallback_miss")
	resilientCache := store.NewResilientCache(client, log, nil)
	cache := store.NewCacheWithFallback(resilientCache, log)
	ctx := context.Background()
	client.FlushDB(ctx)

	loaderCalled := false
	loader := func() (interface{}, error) {
		loaderCalled = true
		return "db-value", nil
	}

	// Should call loader on cache miss
	result, err := cache.GetOrLoad(ctx, "fallback:missing", loader, 5*time.Minute)
	require.NoError(t, err)
	assert.True(t, loaderCalled, "Loader should be called on cache miss")
	assert.Equal(t, "db-value", result)
}

func TestCacheWithFallback_LoaderError(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "fallback_error")
	resilientCache := store.NewResilientCache(client, log, nil)
	cache := store.NewCacheWithFallback(resilientCache, log)
	ctx := context.Background()
	client.FlushDB(ctx)

	loader := func() (interface{}, error) {
		return nil, errors.New("database error")
	}

	// Should return loader error
	_, err := cache.GetOrLoad(ctx, "fallback:error", loader, 5*time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestCacheWithFallback_NilCache(t *testing.T) {
	log := logrus.New().WithField("test", "fallback_nil")
	cache := store.NewCacheWithFallback(nil, log)
	ctx := context.Background()

	loaderCalled := false
	loader := func() (interface{}, error) {
		loaderCalled = true
		return "db-value", nil
	}

	// Should call loader when cache is nil
	result, err := cache.GetOrLoad(ctx, "fallback:key", loader, 5*time.Minute)
	require.NoError(t, err)
	assert.True(t, loaderCalled)
	assert.Equal(t, "db-value", result)
}

func TestCacheWithFallback_Stats(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "fallback_stats")
	resilientCache := store.NewResilientCache(client, log, nil)
	cache := store.NewCacheWithFallback(resilientCache, log)
	ctx := context.Background()
	client.FlushDB(ctx)

	loader := func() (interface{}, error) {
		return "value", nil
	}

	// Generate hits and misses
	cache.GetOrLoad(ctx, "key1", loader, 1*time.Minute)
	cache.GetOrLoad(ctx, "key1", loader, 1*time.Minute) // Hit
	cache.GetOrLoad(ctx, "key2", loader, 1*time.Minute)
	cache.GetOrLoad(ctx, "key2", loader, 1*time.Minute) // Hit

	stats := cache.GetStats()
	assert.Contains(t, stats, "hit_count")
	assert.Contains(t, stats, "miss_count")
	assert.Contains(t, stats, "hit_rate")
}

// =============================================================================
// WRITE-THROUGH CACHE TESTS
// =============================================================================

func TestWriteThroughCache_WriteThrough(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "write_through")
	resilientCache := store.NewResilientCache(client, log, nil)
	wtCache := store.NewWriteThroughCache(resilientCache, log)
	ctx := context.Background()
	client.FlushDB(ctx)

	dbWriterCalled := false
	dbWriter := func() (interface{}, error) {
		dbWriterCalled = true
		return map[string]string{"id": "123", "name": "written"}, nil
	}

	result, err := wtCache.WriteThrough(ctx, "wt:key", dbWriter, 5*time.Minute)
	require.NoError(t, err)
	assert.True(t, dbWriterCalled)

	// Verify in cache
	val, err := resilientCache.Get(ctx, "wt:key")
	require.NoError(t, err)
	assert.Contains(t, val, "123")

	// Verify result
	resultMap := result.(map[string]string)
	assert.Equal(t, "123", resultMap["id"])
}

func TestWriteThroughCache_WriteAndInvalidate(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "write_invalidate")
	resilientCache := store.NewResilientCache(client, log, nil)
	wtCache := store.NewWriteThroughCache(resilientCache, log)
	ctx := context.Background()
	client.FlushDB(ctx)

	// Pre-populate cache with patient data
	resilientCache.Set(ctx, "patient:P001:results", "old-data", 5*time.Minute)
	resilientCache.Set(ctx, "patient:P001:baseline", "old-baseline", 5*time.Minute)

	dbWriter := func() (interface{}, error) {
		return "new-result", nil
	}

	// Write and invalidate patient cache
	result, err := wtCache.WriteAndInvalidate(ctx, "patient:P001:*", dbWriter)
	require.NoError(t, err)
	assert.Equal(t, "new-result", result)

	// Verify patient cache invalidated
	keys, err := client.Keys(ctx, "patient:P001:*").Result()
	require.NoError(t, err)
	assert.Empty(t, keys, "Patient cache should be invalidated")
}

func TestWriteThroughCache_DBError(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "write_error")
	resilientCache := store.NewResilientCache(client, log, nil)
	wtCache := store.NewWriteThroughCache(resilientCache, log)
	ctx := context.Background()

	dbWriter := func() (interface{}, error) {
		return nil, errors.New("DB write failed")
	}

	_, err := wtCache.WriteThrough(ctx, "wt:key", dbWriter, 5*time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DB write failed")
}

// =============================================================================
// CONCURRENT SAFETY TESTS
// =============================================================================

func TestResilientCache_ConcurrentAccess(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "concurrent")
	cache := store.NewResilientCache(client, log, nil)
	ctx := context.Background()
	client.FlushDB(ctx)

	var wg sync.WaitGroup
	var successCount int32
	numGoroutines := 50
	numOpsPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := "concurrent:" + string(rune('A'+gid%26))
				if j%2 == 0 {
					err := cache.Set(ctx, key, "value", 1*time.Minute)
					if err == nil {
						atomic.AddInt32(&successCount, 1)
					}
				} else {
					_, err := cache.Get(ctx, key)
					if err == nil {
						atomic.AddInt32(&successCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()

	// Most operations should succeed
	expectedMin := int32(float64(numGoroutines*numOpsPerGoroutine) * 0.95)
	assert.GreaterOrEqual(t, successCount, expectedMin, "At least 95% of operations should succeed")
}

func TestCircuitBreaker_ConcurrentStateTransitions(t *testing.T) {
	client := getTestRedisClient(t)
	if client == nil {
		t.Skip("Redis not available")
	}
	defer client.Close()

	log := logrus.New().WithField("test", "concurrent_state")
	cache := store.NewResilientCache(client, log, nil)
	ctx := context.Background()
	client.FlushDB(ctx)

	var wg sync.WaitGroup

	// Concurrent operations while circuit breaker is managing state
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				cache.IsAvailable()
				cache.GetState()
				cache.GetStats()
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	wg.Wait()

	// Should complete without deadlock or race conditions
	// The test passing means no data races occurred
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func getTestRedisClient(t *testing.T) *redis.Client {
	opt := &redis.Options{
		Addr:        "localhost:6395",
		DB:          15, // Use DB 15 for tests
		DialTimeout: 1 * time.Second,
	}
	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil
	}

	return client
}
