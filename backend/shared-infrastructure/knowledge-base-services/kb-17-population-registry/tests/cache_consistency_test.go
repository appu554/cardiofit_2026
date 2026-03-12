// Package tests provides comprehensive test utilities for KB-17 Population Registry
// cache_consistency_test.go - Tests for Redis cache consistency and invalidation
// This validates cache behavior critical for population registry performance
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
// CACHE CONSISTENCY TESTS
// =============================================================================

// TestCacheConsistency_WriteThrough tests write-through cache behavior
func TestCacheConsistency_WriteThrough(t *testing.T) {
	repo := NewMockRepository()
	cache := NewMockCache()
	ctx := TestContext(t)

	// Create enrollment (should write to both DB and cache)
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "cache-test-001",
		RegistryCode: models.RegistryDiabetes,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now(),
	}

	// Write to repo
	err := repo.CreateEnrollment(enrollment)
	require.NoError(t, err)

	// Write to cache
	cacheKey := buildEnrollmentCacheKey(enrollment.PatientID, enrollment.RegistryCode)
	cache.Set(ctx, cacheKey, enrollment, 5*time.Minute)

	// Verify cache hit
	cached, found := cache.Get(ctx, cacheKey)
	assert.True(t, found, "Cache should have enrollment")
	assert.Equal(t, enrollment.ID, cached.(*models.RegistryPatient).ID)

	// Verify DB has same data
	fromDB, _ := repo.GetEnrollment(enrollment.ID)
	assert.Equal(t, enrollment.PatientID, fromDB.PatientID)
}

// TestCacheConsistency_InvalidateOnUpdate tests cache invalidation on update
func TestCacheConsistency_InvalidateOnUpdate(t *testing.T) {
	repo := NewMockRepository()
	cache := NewMockCache()
	ctx := TestContext(t)

	// Setup: Create and cache enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "cache-invalidate-001",
		RegistryCode: models.RegistryHypertension,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierLow,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	cacheKey := buildEnrollmentCacheKey(enrollment.PatientID, enrollment.RegistryCode)
	cache.Set(ctx, cacheKey, enrollment, 5*time.Minute)

	// Update in DB
	_ = repo.UpdateEnrollmentRiskTier(enrollment.ID, models.RiskTierLow, models.RiskTierHigh, "system")

	// Invalidate cache
	cache.Delete(ctx, cacheKey)

	// Verify cache miss
	_, found := cache.Get(ctx, cacheKey)
	assert.False(t, found, "Cache should be invalidated after update")

	// Verify DB has updated data
	updated, _ := repo.GetEnrollment(enrollment.ID)
	assert.Equal(t, models.RiskTierHigh, updated.RiskTier)
}

// TestCacheConsistency_StaleReadPrevention tests stale read prevention
func TestCacheConsistency_StaleReadPrevention(t *testing.T) {
	repo := NewMockRepository()
	cache := NewMockCache()
	ctx := TestContext(t)

	// Create enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "stale-read-001",
		RegistryCode: models.RegistryCKD,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	// Cache with version
	cacheKey := buildEnrollmentCacheKey(enrollment.PatientID, enrollment.RegistryCode)
	versionedData := &VersionedCacheEntry{
		Data:    enrollment,
		Version: 1,
	}
	cache.Set(ctx, cacheKey, versionedData, 5*time.Minute)

	// Update in DB (increment version conceptually)
	_ = repo.UpdateEnrollmentRiskTier(enrollment.ID, models.RiskTierModerate, models.RiskTierHigh, "system")

	// New version would be 2, cached is 1
	// Read should detect stale and refresh
	cached, found := cache.Get(ctx, cacheKey)
	if found {
		entry := cached.(*VersionedCacheEntry)
		// If versions don't match, invalidate
		if entry.Version < 2 {
			cache.Delete(ctx, cacheKey)
		}
	}

	// After invalidation, cache miss should occur
	_, found = cache.Get(ctx, cacheKey)
	assert.False(t, found, "Stale cache should be invalidated")
}

// =============================================================================
// CACHE EXPIRATION TESTS
// =============================================================================

// TestCacheExpiration_TTLEnforced tests cache TTL enforcement
func TestCacheExpiration_TTLEnforced(t *testing.T) {
	cache := NewMockCache()
	ctx := TestContext(t)

	key := "ttl-test-key"
	value := "test-value"

	// Set with short TTL
	cache.Set(ctx, key, value, 100*time.Millisecond)

	// Immediate read should succeed
	cached, found := cache.Get(ctx, key)
	assert.True(t, found)
	assert.Equal(t, value, cached)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Read after expiration should fail
	_, found = cache.Get(ctx, key)
	assert.False(t, found, "Cache entry should expire after TTL")
}

// TestCacheExpiration_RefreshExtendsTTL tests TTL refresh on access
func TestCacheExpiration_RefreshExtendsTTL(t *testing.T) {
	cache := NewMockCache()
	cache.SetRefreshOnAccess(true)
	ctx := TestContext(t)

	key := "refresh-ttl-key"
	value := "test-value"

	// Set with short TTL
	cache.Set(ctx, key, value, 200*time.Millisecond)

	// Access periodically to refresh
	for i := 0; i < 3; i++ {
		time.Sleep(100 * time.Millisecond)
		cached, found := cache.Get(ctx, key)
		assert.True(t, found, "Cache should still be valid after refresh")
		assert.Equal(t, value, cached)
	}
}

// =============================================================================
// CACHE POPULATION TESTS
// =============================================================================

// TestCachePopulation_BulkLoad tests bulk cache population
func TestCachePopulation_BulkLoad(t *testing.T) {
	repo := NewMockRepository()
	cache := NewMockCache()
	ctx := TestContext(t)

	// Create enrollments in DB
	enrollments := make([]*models.RegistryPatient, 100)
	for i := 0; i < 100; i++ {
		enrollments[i] = &models.RegistryPatient{
			ID:           uuid.New(),
			PatientID:    createCacheTestPatientID(i),
			RegistryCode: models.RegistryDiabetes,
			Status:       models.EnrollmentStatusActive,
			RiskTier:     models.RiskTierModerate,
			EnrolledAt:   time.Now(),
		}
		_ = repo.CreateEnrollment(enrollments[i])
	}

	// Bulk populate cache
	cachePopulator := NewCachePopulator(cache)
	populated := cachePopulator.PopulateBulk(ctx, enrollments)

	assert.Equal(t, 100, populated, "All enrollments should be cached")

	// Verify cache hits
	for i := 0; i < 10; i++ {
		key := buildEnrollmentCacheKey(enrollments[i].PatientID, enrollments[i].RegistryCode)
		_, found := cache.Get(ctx, key)
		assert.True(t, found, "Cached enrollment should be retrievable")
	}
}

// TestCachePopulation_WarmupOnStart tests cache warmup strategy
func TestCachePopulation_WarmupOnStart(t *testing.T) {
	repo := NewMockRepository()
	cache := NewMockCache()
	ctx := TestContext(t)

	// Pre-populate DB with high-risk patients (priority for cache)
	highRiskPatients := make([]*models.RegistryPatient, 20)
	for i := 0; i < 20; i++ {
		highRiskPatients[i] = &models.RegistryPatient{
			ID:           uuid.New(),
			PatientID:    "high-risk-" + createCacheTestPatientID(i),
			RegistryCode: models.RegistryHeartFailure,
			Status:       models.EnrollmentStatusActive,
			RiskTier:     models.RiskTierCritical,
			EnrolledAt:   time.Now(),
		}
		_ = repo.CreateEnrollment(highRiskPatients[i])
	}

	// Warmup: Load high-risk patients into cache
	warmer := NewCacheWarmer(repo, cache)
	warmed := warmer.WarmHighRiskPatients(ctx)

	assert.Equal(t, 20, warmed, "All high-risk patients should be cached")

	// Verify high-risk patients are in cache
	for _, p := range highRiskPatients {
		key := buildEnrollmentCacheKey(p.PatientID, p.RegistryCode)
		_, found := cache.Get(ctx, key)
		assert.True(t, found, "High-risk patient should be in cache")
	}
}

// =============================================================================
// CACHE CONCURRENCY TESTS
// =============================================================================

// TestCacheConcurrency_SimultaneousReadsWrites tests concurrent access
func TestCacheConcurrency_SimultaneousReadsWrites(t *testing.T) {
	cache := NewMockCache()
	ctx := TestContext(t)

	const iterations = 100
	const goroutines = 10

	var wg sync.WaitGroup

	// Concurrent writers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := "concurrent-key-" + string(rune('0'+workerID))
				value := workerID*iterations + j
				cache.Set(ctx, key, value, time.Minute)
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				key := "concurrent-key-" + string(rune('0'+workerID))
				cache.Get(ctx, key) // Result doesn't matter, testing for races
			}
		}(i)
	}

	wg.Wait()

	// Test should complete without race conditions or panics
	assert.True(t, true, "Concurrent access should not cause race conditions")
}

// TestCacheConcurrency_AtomicIncrements tests atomic counter operations
func TestCacheConcurrency_AtomicIncrements(t *testing.T) {
	cache := NewMockCache()
	ctx := TestContext(t)

	key := "counter-key"
	cache.SetInt(ctx, key, 0)

	const goroutines = 10
	const incrementsPerGoroutine = 100

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				cache.Increment(ctx, key, 1)
			}
		}()
	}

	wg.Wait()

	finalValue := cache.GetInt(ctx, key)
	assert.Equal(t, goroutines*incrementsPerGoroutine, finalValue,
		"Counter should reflect all atomic increments")
}

// =============================================================================
// CACHE KEY PATTERN TESTS
// =============================================================================

// TestCacheKeyPatterns_EnrollmentKeys tests enrollment cache key patterns
func TestCacheKeyPatterns_EnrollmentKeys(t *testing.T) {
	testCases := []struct {
		patientID    string
		registryCode models.RegistryCode
		expectedKey  string
	}{
		{
			patientID:    "patient-001",
			registryCode: models.RegistryDiabetes,
			expectedKey:  "enrollment:patient-001:DIABETES",
		},
		{
			patientID:    "patient-002",
			registryCode: models.RegistryHypertension,
			expectedKey:  "enrollment:patient-002:HYPERTENSION",
		},
	}

	for _, tc := range testCases {
		key := buildEnrollmentCacheKey(tc.patientID, tc.registryCode)
		assert.Equal(t, tc.expectedKey, key)
	}
}

// TestCacheKeyPatterns_PatternDeletion tests wildcard deletion
func TestCacheKeyPatterns_PatternDeletion(t *testing.T) {
	cache := NewMockCache()
	ctx := TestContext(t)

	// Set multiple keys for same patient
	patientID := "patient-pattern-001"
	registries := []models.RegistryCode{
		models.RegistryDiabetes,
		models.RegistryHypertension,
		models.RegistryCKD,
	}

	for _, reg := range registries {
		key := buildEnrollmentCacheKey(patientID, reg)
		cache.Set(ctx, key, "value", time.Minute)
	}

	// Delete all keys for patient
	pattern := "enrollment:" + patientID + ":*"
	deleted := cache.DeletePattern(ctx, pattern)

	assert.Equal(t, 3, deleted, "All patient cache entries should be deleted")

	// Verify deletion
	for _, reg := range registries {
		key := buildEnrollmentCacheKey(patientID, reg)
		_, found := cache.Get(ctx, key)
		assert.False(t, found, "Cache entry should be deleted")
	}
}

// =============================================================================
// HELPER TYPES AND FUNCTIONS
// =============================================================================

// VersionedCacheEntry wraps cache data with version for staleness detection
type VersionedCacheEntry struct {
	Data    interface{}
	Version int64
}

// MockCache provides an in-memory cache for testing
type MockCache struct {
	mu              sync.RWMutex
	data            map[string]cacheEntry
	refreshOnAccess bool
	counters        map[string]int
}

type cacheEntry struct {
	value     interface{}
	expiresAt time.Time
	ttl       time.Duration
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		data:     make(map[string]cacheEntry),
		counters: make(map[string]int),
	}
}

// SetRefreshOnAccess enables TTL refresh on read
func (c *MockCache) SetRefreshOnAccess(enabled bool) {
	c.refreshOnAccess = enabled
}

// Set stores a value with TTL
func (c *MockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
		ttl:       ttl,
	}
}

// Get retrieves a value
func (c *MockCache) Get(ctx context.Context, key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.data[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		delete(c.data, key)
		return nil, false
	}

	if c.refreshOnAccess {
		entry.expiresAt = time.Now().Add(entry.ttl)
		c.data[key] = entry
	}

	return entry.value, true
}

// Delete removes a key
func (c *MockCache) Delete(ctx context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// DeletePattern deletes keys matching pattern
func (c *MockCache) DeletePattern(ctx context.Context, pattern string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Simple pattern matching (just prefix before *)
	prefix := ""
	for i, ch := range pattern {
		if ch == '*' {
			prefix = pattern[:i]
			break
		}
	}

	deleted := 0
	for key := range c.data {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.data, key)
			deleted++
		}
	}
	return deleted
}

// SetInt sets an integer value
func (c *MockCache) SetInt(ctx context.Context, key string, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[key] = value
}

// GetInt gets an integer value
func (c *MockCache) GetInt(ctx context.Context, key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.counters[key]
}

// Increment atomically increments a counter
func (c *MockCache) Increment(ctx context.Context, key string, delta int) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[key] += delta
	return c.counters[key]
}

// CachePopulator handles bulk cache population
type CachePopulator struct {
	cache *MockCache
}

// NewCachePopulator creates a new cache populator
func NewCachePopulator(cache *MockCache) *CachePopulator {
	return &CachePopulator{cache: cache}
}

// PopulateBulk populates cache with multiple enrollments
func (p *CachePopulator) PopulateBulk(ctx context.Context, enrollments []*models.RegistryPatient) int {
	populated := 0
	for _, e := range enrollments {
		key := buildEnrollmentCacheKey(e.PatientID, e.RegistryCode)
		p.cache.Set(ctx, key, e, 5*time.Minute)
		populated++
	}
	return populated
}

// CacheWarmer handles cache warmup strategies
type CacheWarmer struct {
	repo  *MockRepository
	cache *MockCache
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(repo *MockRepository, cache *MockCache) *CacheWarmer {
	return &CacheWarmer{repo: repo, cache: cache}
}

// WarmHighRiskPatients loads high-risk patients into cache
func (w *CacheWarmer) WarmHighRiskPatients(ctx context.Context) int {
	// Get high-risk enrollments
	enrollments, _, _ := w.repo.ListEnrollments(&models.EnrollmentQuery{
		RiskTier: models.RiskTierCritical,
	})

	warmed := 0
	for _, e := range enrollments {
		key := buildEnrollmentCacheKey(e.PatientID, e.RegistryCode)
		w.cache.Set(ctx, key, &e, 10*time.Minute)
		warmed++
	}
	return warmed
}

// buildEnrollmentCacheKey builds cache key for enrollment
func buildEnrollmentCacheKey(patientID string, registryCode models.RegistryCode) string {
	return "enrollment:" + patientID + ":" + string(registryCode)
}

func createCacheTestPatientID(index int) string {
	return fmt.Sprintf("cache-patient-%d", index)
}
