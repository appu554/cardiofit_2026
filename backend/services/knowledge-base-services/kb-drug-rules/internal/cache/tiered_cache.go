package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kb-drug-rules/internal/models"

	"github.com/redis/go-redis/v9"
)

// TieredCache implements a 3-tier caching strategy
type TieredCache struct {
	// L1: In-memory cache (fastest)
	l1Cache *MemoryCache
	
	// L2: Redis cache (shared across instances)
	l2Cache *RedisCache
	
	// L3: CDN cache (global distribution) - interface for future implementation
	l3Cache CDNCache
	
	// Metrics
	metrics *CacheMetrics
}

// CacheConfig holds configuration for the tiered cache
type CacheConfig struct {
	L1TTL        time.Duration
	L2TTL        time.Duration
	L3TTL        time.Duration
	L1MaxSize    int
	RedisAddr    string
	RedisPassword string
	RedisDB      int
}

// NewTieredCache creates a new tiered cache instance
func NewTieredCache(config CacheConfig) *TieredCache {
	return &TieredCache{
		l1Cache: NewMemoryCache(config.L1MaxSize, config.L1TTL),
		l2Cache: NewRedisCache(config.RedisAddr, config.RedisPassword, config.RedisDB, config.L2TTL),
		l3Cache: &NoOpCDNCache{}, // Placeholder for CDN implementation
		metrics: NewCacheMetrics(),
	}
}

// Get retrieves a value from the cache, checking L1 -> L2 -> L3
func (tc *TieredCache) Get(ctx context.Context, key string) (*models.DrugRulePack, error) {
	// Try L1 cache first
	if value, found := tc.l1Cache.Get(key); found {
		tc.metrics.RecordHit("L1")
		return value, nil
	}
	tc.metrics.RecordMiss("L1")
	
	// Try L2 cache
	value, err := tc.l2Cache.Get(ctx, key)
	if err == nil && value != nil {
		tc.metrics.RecordHit("L2")
		// Populate L1 cache
		tc.l1Cache.Set(key, value)
		return value, nil
	}
	tc.metrics.RecordMiss("L2")
	
	// Try L3 cache (CDN)
	value, err = tc.l3Cache.Get(ctx, key)
	if err == nil && value != nil {
		tc.metrics.RecordHit("L3")
		// Populate L2 and L1 caches
		tc.l2Cache.Set(ctx, key, value)
		tc.l1Cache.Set(key, value)
		return value, nil
	}
	tc.metrics.RecordMiss("L3")
	
	return nil, fmt.Errorf("cache miss on all levels for key: %s", key)
}

// Set stores a value in all cache levels
func (tc *TieredCache) Set(ctx context.Context, key string, value *models.DrugRulePack) error {
	// Set in L1 cache
	tc.l1Cache.Set(key, value)
	
	// Set in L2 cache
	if err := tc.l2Cache.Set(ctx, key, value); err != nil {
		return fmt.Errorf("failed to set L2 cache: %w", err)
	}
	
	// Set in L3 cache (CDN)
	if err := tc.l3Cache.Set(ctx, key, value); err != nil {
		// CDN errors are non-fatal, just log
		fmt.Printf("Warning: failed to set L3 cache: %v\n", err)
	}
	
	return nil
}

// Delete removes a value from all cache levels
func (tc *TieredCache) Delete(ctx context.Context, key string) error {
	// Delete from L1
	tc.l1Cache.Delete(key)
	
	// Delete from L2
	if err := tc.l2Cache.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete from L2 cache: %w", err)
	}
	
	// Delete from L3
	if err := tc.l3Cache.Delete(ctx, key); err != nil {
		fmt.Printf("Warning: failed to delete from L3 cache: %v\n", err)
	}
	
	return nil
}

// InvalidatePattern invalidates cache entries matching a pattern
func (tc *TieredCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// Invalidate L1 cache pattern
	tc.l1Cache.InvalidatePattern(pattern)
	
	// Invalidate L2 cache pattern
	if err := tc.l2Cache.InvalidatePattern(ctx, pattern); err != nil {
		return fmt.Errorf("failed to invalidate L2 cache pattern: %w", err)
	}
	
	// Invalidate L3 cache pattern
	if err := tc.l3Cache.InvalidatePattern(ctx, pattern); err != nil {
		fmt.Printf("Warning: failed to invalidate L3 cache pattern: %v\n", err)
	}
	
	return nil
}

// GetMetrics returns cache metrics
func (tc *TieredCache) GetMetrics() *CacheMetrics {
	return tc.metrics
}

// MemoryCache implements L1 in-memory caching
type MemoryCache struct {
	data    map[string]*CacheEntry
	mutex   sync.RWMutex
	maxSize int
	ttl     time.Duration
}

// CacheEntry represents a cached item with expiration
type CacheEntry struct {
	Value     *models.DrugRulePack
	ExpiresAt time.Time
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache(maxSize int, ttl time.Duration) *MemoryCache {
	mc := &MemoryCache{
		data:    make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
	
	// Start cleanup goroutine
	go mc.cleanup()
	
	return mc
}

// Get retrieves a value from memory cache
func (mc *MemoryCache) Get(key string) (*models.DrugRulePack, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	
	entry, exists := mc.data[key]
	if !exists {
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		// Expired, remove it
		delete(mc.data, key)
		return nil, false
	}
	
	return entry.Value, true
}

// Set stores a value in memory cache
func (mc *MemoryCache) Set(key string, value *models.DrugRulePack) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	// Check if we need to evict entries
	if len(mc.data) >= mc.maxSize {
		mc.evictOldest()
	}
	
	mc.data[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(mc.ttl),
	}
}

// Delete removes a value from memory cache
func (mc *MemoryCache) Delete(key string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	delete(mc.data, key)
}

// InvalidatePattern removes entries matching a pattern
func (mc *MemoryCache) InvalidatePattern(pattern string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	
	// Simple pattern matching (could be enhanced with regex)
	for key := range mc.data {
		if matchesPattern(key, pattern) {
			delete(mc.data, key)
		}
	}
}

// evictOldest removes the oldest entry (LRU-like behavior)
func (mc *MemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	
	for key, entry := range mc.data {
		if oldestKey == "" || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
		}
	}
	
	if oldestKey != "" {
		delete(mc.data, oldestKey)
	}
}

// cleanup removes expired entries periodically
func (mc *MemoryCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		mc.mutex.Lock()
		now := time.Now()
		for key, entry := range mc.data {
			if now.After(entry.ExpiresAt) {
				delete(mc.data, key)
			}
		}
		mc.mutex.Unlock()
	}
}

// RedisCache implements L2 Redis caching
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(addr, password string, db int, ttl time.Duration) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	
	return &RedisCache{
		client: rdb,
		ttl:    ttl,
	}
}

// Get retrieves a value from Redis cache
func (rc *RedisCache) Get(ctx context.Context, key string) (*models.DrugRulePack, error) {
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("redis get error: %w", err)
	}
	
	var rulePack models.DrugRulePack
	if err := json.Unmarshal([]byte(val), &rulePack); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}
	
	return &rulePack, nil
}

// Set stores a value in Redis cache
func (rc *RedisCache) Set(ctx context.Context, key string, value *models.DrugRulePack) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	
	err = rc.client.Set(ctx, key, jsonData, rc.ttl).Err()
	if err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}
	
	return nil
}

// Delete removes a value from Redis cache
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	err := rc.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("redis delete error: %w", err)
	}
	
	return nil
}

// InvalidatePattern removes entries matching a pattern from Redis
func (rc *RedisCache) InvalidatePattern(ctx context.Context, pattern string) error {
	keys, err := rc.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("redis keys error: %w", err)
	}
	
	if len(keys) > 0 {
		err = rc.client.Del(ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("redis delete pattern error: %w", err)
		}
	}
	
	return nil
}

// CDNCache interface for L3 caching (CDN)
type CDNCache interface {
	Get(ctx context.Context, key string) (*models.DrugRulePack, error)
	Set(ctx context.Context, key string, value *models.DrugRulePack) error
	Delete(ctx context.Context, key string) error
	InvalidatePattern(ctx context.Context, pattern string) error
}

// NoOpCDNCache is a no-op implementation of CDNCache
type NoOpCDNCache struct{}

func (n *NoOpCDNCache) Get(ctx context.Context, key string) (*models.DrugRulePack, error) {
	return nil, fmt.Errorf("CDN cache not implemented")
}

func (n *NoOpCDNCache) Set(ctx context.Context, key string, value *models.DrugRulePack) error {
	return nil // No-op
}

func (n *NoOpCDNCache) Delete(ctx context.Context, key string) error {
	return nil // No-op
}

func (n *NoOpCDNCache) InvalidatePattern(ctx context.Context, pattern string) error {
	return nil // No-op
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	hits   map[string]int64
	misses map[string]int64
	mutex  sync.RWMutex
}

// NewCacheMetrics creates new cache metrics
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{
		hits:   make(map[string]int64),
		misses: make(map[string]int64),
	}
}

// RecordHit records a cache hit
func (cm *CacheMetrics) RecordHit(level string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.hits[level]++
}

// RecordMiss records a cache miss
func (cm *CacheMetrics) RecordMiss(level string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.misses[level]++
}

// GetHitRate returns hit rate for a cache level
func (cm *CacheMetrics) GetHitRate(level string) float64 {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	hits := cm.hits[level]
	misses := cm.misses[level]
	total := hits + misses
	
	if total == 0 {
		return 0
	}
	
	return float64(hits) / float64(total)
}

// GetStats returns cache statistics
func (cm *CacheMetrics) GetStats() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	
	for level := range cm.hits {
		stats[level] = map[string]interface{}{
			"hits":     cm.hits[level],
			"misses":   cm.misses[level],
			"hit_rate": cm.GetHitRate(level),
		}
	}
	
	return stats
}

// Helper function for pattern matching
func matchesPattern(key, pattern string) bool {
	// Simple wildcard matching - could be enhanced
	if pattern == "*" {
		return true
	}
	
	// For now, just check if pattern is a prefix
	return len(key) >= len(pattern) && key[:len(pattern)] == pattern
}
