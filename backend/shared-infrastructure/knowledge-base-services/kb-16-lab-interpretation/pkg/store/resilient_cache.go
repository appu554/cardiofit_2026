// Package store provides data storage and retrieval for lab results
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// =============================================================================
// RESILIENT CACHE - Circuit Breaker + Graceful Degradation
// =============================================================================

// CacheState represents the circuit breaker state
type CacheState int

const (
	CacheStateClosed   CacheState = iota // Normal operation
	CacheStateOpen                       // Circuit open, all calls fail fast
	CacheStateHalfOpen                   // Testing if cache is back
)

// ResilientCache wraps Redis with circuit breaker and fallback
type ResilientCache struct {
	client          *redis.Client
	log             *logrus.Entry
	state           CacheState
	failureCount    int32
	successCount    int32
	lastFailure     time.Time
	lastSuccess     time.Time
	mu              sync.RWMutex

	// Configuration
	failureThreshold int32         // Failures before opening circuit
	successThreshold int32         // Successes in half-open to close
	openDuration     time.Duration // Time before trying half-open
	timeout          time.Duration // Operation timeout
}

// ResilientCacheConfig holds configuration for the resilient cache
type ResilientCacheConfig struct {
	FailureThreshold int32
	SuccessThreshold int32
	OpenDuration     time.Duration
	Timeout          time.Duration
}

// DefaultResilientCacheConfig returns sensible defaults
func DefaultResilientCacheConfig() *ResilientCacheConfig {
	return &ResilientCacheConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		OpenDuration:     30 * time.Second,
		Timeout:          500 * time.Millisecond,
	}
}

// NewResilientCache creates a new resilient cache wrapper
func NewResilientCache(client *redis.Client, log *logrus.Entry, cfg *ResilientCacheConfig) *ResilientCache {
	if cfg == nil {
		cfg = DefaultResilientCacheConfig()
	}

	return &ResilientCache{
		client:           client,
		log:              log.WithField("component", "resilient_cache"),
		state:            CacheStateClosed,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		openDuration:     cfg.OpenDuration,
		timeout:          cfg.Timeout,
	}
}

// IsAvailable returns true if the cache is currently available
func (rc *ResilientCache) IsAvailable() bool {
	if rc.client == nil {
		return false
	}

	rc.mu.RLock()
	state := rc.state
	lastFailure := rc.lastFailure
	rc.mu.RUnlock()

	switch state {
	case CacheStateClosed:
		return true
	case CacheStateOpen:
		// Check if we should transition to half-open
		if time.Since(lastFailure) > rc.openDuration {
			rc.transitionToHalfOpen()
			return true
		}
		return false
	case CacheStateHalfOpen:
		return true
	}

	return false
}

// Get retrieves a value with circuit breaker protection
func (rc *ResilientCache) Get(ctx context.Context, key string) (string, error) {
	if !rc.IsAvailable() {
		return "", fmt.Errorf("cache circuit open")
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	result, err := rc.client.Get(ctx, key).Result()
	rc.recordResult(err)

	if err == redis.Nil {
		return "", nil // Cache miss, not an error
	}

	return result, err
}

// Set stores a value with circuit breaker protection
func (rc *ResilientCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !rc.IsAvailable() {
		return fmt.Errorf("cache circuit open")
	}

	// Serialize value
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	err = rc.client.Set(ctx, key, data, ttl).Err()
	rc.recordResult(err)

	return err
}

// Delete removes a value with circuit breaker protection
func (rc *ResilientCache) Delete(ctx context.Context, key string) error {
	if !rc.IsAvailable() {
		return fmt.Errorf("cache circuit open")
	}

	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	err := rc.client.Del(ctx, key).Err()
	rc.recordResult(err)

	return err
}

// InvalidatePattern removes all keys matching a pattern
func (rc *ResilientCache) InvalidatePattern(ctx context.Context, pattern string) error {
	if !rc.IsAvailable() {
		return fmt.Errorf("cache circuit open")
	}

	ctx, cancel := context.WithTimeout(ctx, rc.timeout*10) // More time for scan
	defer cancel()

	var deletedCount int
	iter := rc.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := rc.client.Del(ctx, iter.Val()).Err(); err != nil {
			rc.recordResult(err)
			return err
		}
		deletedCount++
	}

	if err := iter.Err(); err != nil {
		rc.recordResult(err)
		return err
	}

	rc.recordResult(nil)

	if deletedCount > 0 {
		rc.log.WithFields(logrus.Fields{
			"pattern": pattern,
			"deleted": deletedCount,
		}).Debug("Invalidated cache pattern")
	}

	return nil
}

// GetJSON retrieves and unmarshals a JSON value
func (rc *ResilientCache) GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := rc.Get(ctx, key)
	if err != nil {
		return false, err
	}

	if data == "" {
		return false, nil // Cache miss
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return true, nil
}

// recordResult records success or failure for circuit breaker
func (rc *ResilientCache) recordResult(err error) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if err != nil && err != redis.Nil {
		atomic.AddInt32(&rc.failureCount, 1)
		atomic.StoreInt32(&rc.successCount, 0)
		rc.lastFailure = time.Now()

		rc.log.WithError(err).Debug("Cache operation failed")

		// Check if we should open the circuit
		if rc.state == CacheStateClosed && atomic.LoadInt32(&rc.failureCount) >= rc.failureThreshold {
			rc.state = CacheStateOpen
			rc.log.Warn("Cache circuit breaker opened")
		} else if rc.state == CacheStateHalfOpen {
			// Failed in half-open, go back to open
			rc.state = CacheStateOpen
			rc.log.Warn("Cache circuit breaker re-opened from half-open")
		}
	} else {
		atomic.AddInt32(&rc.successCount, 1)
		rc.lastSuccess = time.Now()

		// Check if we should close the circuit
		if rc.state == CacheStateHalfOpen && atomic.LoadInt32(&rc.successCount) >= rc.successThreshold {
			rc.state = CacheStateClosed
			atomic.StoreInt32(&rc.failureCount, 0)
			rc.log.Info("Cache circuit breaker closed")
		}
	}
}

// transitionToHalfOpen moves circuit to half-open state
func (rc *ResilientCache) transitionToHalfOpen() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if rc.state == CacheStateOpen {
		rc.state = CacheStateHalfOpen
		atomic.StoreInt32(&rc.successCount, 0)
		rc.log.Info("Cache circuit breaker half-open, testing connection")
	}
}

// GetState returns the current circuit breaker state
func (rc *ResilientCache) GetState() CacheState {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.state
}

// GetStats returns cache statistics
func (rc *ResilientCache) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	stateStr := "unknown"
	switch rc.state {
	case CacheStateClosed:
		stateStr = "closed"
	case CacheStateOpen:
		stateStr = "open"
	case CacheStateHalfOpen:
		stateStr = "half-open"
	}

	return map[string]interface{}{
		"state":           stateStr,
		"failure_count":   atomic.LoadInt32(&rc.failureCount),
		"success_count":   atomic.LoadInt32(&rc.successCount),
		"last_failure":    rc.lastFailure,
		"last_success":    rc.lastSuccess,
		"is_available":    rc.IsAvailable(),
	}
}

// Reset resets the circuit breaker to closed state
func (rc *ResilientCache) Reset() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.state = CacheStateClosed
	atomic.StoreInt32(&rc.failureCount, 0)
	atomic.StoreInt32(&rc.successCount, 0)
	rc.log.Info("Cache circuit breaker reset")
}

// Ping tests the cache connection
func (rc *ResilientCache) Ping(ctx context.Context) error {
	if rc.client == nil {
		return fmt.Errorf("cache client is nil")
	}

	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	err := rc.client.Ping(ctx).Err()
	rc.recordResult(err)
	return err
}

// =============================================================================
// CACHE WITH FALLBACK - Automatic fallback to database
// =============================================================================

// CacheWithFallback provides cache operations with automatic DB fallback
type CacheWithFallback struct {
	cache     *ResilientCache
	log       *logrus.Entry
	hitCount  int64
	missCount int64
	errorCount int64
}

// NewCacheWithFallback creates a new cache with fallback wrapper
func NewCacheWithFallback(cache *ResilientCache, log *logrus.Entry) *CacheWithFallback {
	return &CacheWithFallback{
		cache: cache,
		log:   log.WithField("component", "cache_fallback"),
	}
}

// GetOrLoad attempts cache first, falls back to loader function
func (cwf *CacheWithFallback) GetOrLoad(ctx context.Context, key string, loader func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	// Try cache if available
	if cwf.cache != nil && cwf.cache.IsAvailable() {
		var result interface{}
		found, err := cwf.cache.GetJSON(ctx, key, &result)
		if err != nil {
			atomic.AddInt64(&cwf.errorCount, 1)
			cwf.log.WithError(err).WithField("key", key).Debug("Cache get failed, falling back to loader")
		} else if found {
			atomic.AddInt64(&cwf.hitCount, 1)
			return result, nil
		}
	}

	// Cache miss or unavailable, load from source
	atomic.AddInt64(&cwf.missCount, 1)
	result, err := loader()
	if err != nil {
		return nil, err
	}

	// Try to cache the result
	if cwf.cache != nil && cwf.cache.IsAvailable() {
		if err := cwf.cache.Set(ctx, key, result, ttl); err != nil {
			cwf.log.WithError(err).WithField("key", key).Debug("Failed to cache result")
		}
	}

	return result, nil
}

// Invalidate removes a key from cache
func (cwf *CacheWithFallback) Invalidate(ctx context.Context, key string) error {
	if cwf.cache == nil || !cwf.cache.IsAvailable() {
		return nil // Silently succeed if cache unavailable
	}
	return cwf.cache.Delete(ctx, key)
}

// InvalidatePattern removes keys matching pattern
func (cwf *CacheWithFallback) InvalidatePattern(ctx context.Context, pattern string) error {
	if cwf.cache == nil || !cwf.cache.IsAvailable() {
		return nil // Silently succeed if cache unavailable
	}
	return cwf.cache.InvalidatePattern(ctx, pattern)
}

// GetStats returns cache hit/miss statistics
func (cwf *CacheWithFallback) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"hit_count":   atomic.LoadInt64(&cwf.hitCount),
		"miss_count":  atomic.LoadInt64(&cwf.missCount),
		"error_count": atomic.LoadInt64(&cwf.errorCount),
	}

	total := float64(cwf.hitCount + cwf.missCount)
	if total > 0 {
		stats["hit_rate"] = float64(cwf.hitCount) / total
	} else {
		stats["hit_rate"] = 0.0
	}

	if cwf.cache != nil {
		stats["circuit_breaker"] = cwf.cache.GetStats()
	} else {
		stats["circuit_breaker"] = "disabled"
	}

	return stats
}

// =============================================================================
// WRITE-THROUGH CACHE
// =============================================================================

// WriteThroughCache ensures DB writes and cache updates are coordinated
type WriteThroughCache struct {
	cache *ResilientCache
	log   *logrus.Entry
}

// NewWriteThroughCache creates a new write-through cache
func NewWriteThroughCache(cache *ResilientCache, log *logrus.Entry) *WriteThroughCache {
	return &WriteThroughCache{
		cache: cache,
		log:   log.WithField("component", "write_through"),
	}
}

// WriteThrough writes to DB then updates cache
// dbWriter performs the actual DB write and returns the result
// Returns the DB result; cache errors are logged but don't fail the operation
func (wtc *WriteThroughCache) WriteThrough(ctx context.Context, cacheKey string,
	dbWriter func() (interface{}, error), ttl time.Duration) (interface{}, error) {

	// Always write to DB first (source of truth)
	result, err := dbWriter()
	if err != nil {
		return nil, err
	}

	// Update cache (best effort)
	if wtc.cache != nil && wtc.cache.IsAvailable() {
		if err := wtc.cache.Set(ctx, cacheKey, result, ttl); err != nil {
			wtc.log.WithError(err).WithField("key", cacheKey).Warn("Failed to update cache after DB write")
		}
	}

	return result, nil
}

// WriteAndInvalidate writes to DB then invalidates related cache entries
func (wtc *WriteThroughCache) WriteAndInvalidate(ctx context.Context, invalidatePattern string,
	dbWriter func() (interface{}, error)) (interface{}, error) {

	// Write to DB first
	result, err := dbWriter()
	if err != nil {
		return nil, err
	}

	// Invalidate cache (best effort)
	if wtc.cache != nil && wtc.cache.IsAvailable() {
		if err := wtc.cache.InvalidatePattern(ctx, invalidatePattern); err != nil {
			wtc.log.WithError(err).WithField("pattern", invalidatePattern).Warn("Failed to invalidate cache")
		}
	}

	return result, nil
}
