package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
)

// CacheLayer represents the different cache layers
type CacheLayer int

const (
	L1Memory CacheLayer = iota // Hot cache - in memory
	L2Redis                    // Warm cache - Redis
	L3Persistent              // Cold cache - Redis cluster/persistent
)

// CacheConfig holds configuration for the multi-layer cache
type CacheConfig struct {
	L1Config L1Config `json:"l1_config"`
	L2Config L2Config `json:"l2_config"`
	L3Config L3Config `json:"l3_config"`
}

// L1Config configuration for in-memory cache
type L1Config struct {
	MaxSizeMB     int64         `json:"max_size_mb"`
	TTL           time.Duration `json:"ttl"`
	NumCounters   int64         `json:"num_counters"`
	MaxCost       int64         `json:"max_cost"`
	BufferItems   int64         `json:"buffer_items"`
	HitRateTarget float64       `json:"hit_rate_target"`
}

// L2Config configuration for Redis cache
type L2Config struct {
	TTL           time.Duration `json:"ttl"`
	MaxSizeMB     int           `json:"max_size_mb"`
	HitRateTarget float64       `json:"hit_rate_target"`
}

// L3Config configuration for persistent cache
type L3Config struct {
	TTL           time.Duration `json:"ttl"`
	MaxSizeMB     int           `json:"max_size_mb"`
	HitRateTarget float64       `json:"hit_rate_target"`
}

// InvalidationPattern defines cache invalidation strategies
type InvalidationPattern struct {
	Type       string                 `json:"type"`        // concept, valueSet, expansion, cascade
	System     string                 `json:"system,omitempty"`
	Code       string                 `json:"code,omitempty"`
	Pattern    string                 `json:"pattern,omitempty"`
	Priority   int                    `json:"priority,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CacheStatistics holds cache performance metrics
type CacheStatistics struct {
	L1Stats LayerStats `json:"l1_stats"`
	L2Stats LayerStats `json:"l2_stats"`
	L3Stats LayerStats `json:"l3_stats"`
	
	TotalRequests int64   `json:"total_requests"`
	TotalHits     int64   `json:"total_hits"`
	OverallHitRate float64 `json:"overall_hit_rate"`
	
	mutex sync.RWMutex
}

// LayerStats holds statistics for individual cache layers
type LayerStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	Size        int64   `json:"size"`
	MaxSize     int64   `json:"max_size"`
	Evictions   int64   `json:"evictions"`
	LastUpdated time.Time `json:"last_updated"`
}

// MultiLayerCache implements a sophisticated multi-tier caching system
type MultiLayerCache struct {
	l1Cache    *ristretto.Cache  // In-memory cache
	l2Redis    *redis.Client     // Primary Redis cache
	l3Redis    *redis.Client     // Persistent Redis cluster
	config     CacheConfig
	stats      *CacheStatistics
	ctx        context.Context
	
	// Cache warming
	warmupPatterns []string
	warmupMutex    sync.RWMutex
	
	// Invalidation channels
	invalidationCh chan InvalidationPattern
	
	mutex sync.RWMutex
}

// NewMultiLayerCache creates a new multi-layer cache instance
func NewMultiLayerCache(config CacheConfig, l2Redis, l3Redis *redis.Client) (*MultiLayerCache, error) {
	// Initialize L1 cache (Ristretto)
	l1Cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: config.L1Config.NumCounters,
		MaxCost:     config.L1Config.MaxCost,
		BufferItems: config.L1Config.BufferItems,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}
	
	mlc := &MultiLayerCache{
		l1Cache:        l1Cache,
		l2Redis:        l2Redis,
		l3Redis:        l3Redis,
		config:         config,
		ctx:            context.Background(),
		stats:          &CacheStatistics{},
		invalidationCh: make(chan InvalidationPattern, 1000),
		warmupPatterns: []string{
			"kb7:concept:RxNorm:*",
			"kb7:concept:LOINC:*",
			"kb7:concept:ICD10:*",
		},
	}
	
	// Start background workers
	go mlc.invalidationWorker()
	go mlc.statisticsUpdater()
	
	return mlc, nil
}

// Get retrieves a value from the cache, trying each layer in order
func (c *MultiLayerCache) Get(key string) (interface{}, error) {
	c.stats.mutex.Lock()
	c.stats.TotalRequests++
	c.stats.mutex.Unlock()
	
	// Try L1 cache first
	if value, found := c.l1Cache.Get(key); found {
		c.recordHit(L1Memory)
		return value, nil
	}
	
	// Try L2 cache (Redis)
	if value, err := c.getFromRedis(c.l2Redis, key); err == nil {
		c.recordHit(L2Redis)
		// Promote to L1
		c.promoteToL1(key, value)
		return value, nil
	}
	
	// Try L3 cache (Persistent Redis)
	if value, err := c.getFromRedis(c.l3Redis, key); err == nil {
		c.recordHit(L3Persistent)
		// Promote to L2 and L1
		c.promoteToL2(key, value, c.config.L2Config.TTL)
		c.promoteToL1(key, value)
		return value, nil
	}
	
	c.recordMiss()
	return nil, fmt.Errorf("key not found in any cache layer: %s", key)
}

// GetWithLoader retrieves a value, loading it if not found
func (c *MultiLayerCache) GetWithLoader(key string, loader func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, err := c.Get(key); err == nil {
		return value, nil
	}
	
	// Load the value
	value, err := loader()
	if err != nil {
		return nil, err
	}
	
	// Store in all layers
	if err := c.SetAllLayers(key, value); err != nil {
		log.Printf("Warning: failed to cache loaded value: %v", err)
	}
	
	return value, nil
}

// Set stores a value in the appropriate cache layer
func (c *MultiLayerCache) Set(key string, value interface{}, ttl time.Duration) error {
	// Determine which layers to store based on key pattern and size
	cost := c.calculateCost(value)
	
	// Always try to store in L1 if it fits
	if cost <= c.config.L1Config.MaxCost/100 { // Use at most 1% of L1 for single items
		c.l1Cache.SetWithTTL(key, value, cost, ttl)
	}
	
	// Store in L2 Redis
	if err := c.setInRedis(c.l2Redis, key, value, ttl); err != nil {
		log.Printf("Failed to set in L2 cache: %v", err)
	}
	
	// Store in L3 for longer-term caching if TTL is long enough
	if ttl > time.Hour {
		l3TTL := ttl
		if l3TTL > c.config.L3Config.TTL {
			l3TTL = c.config.L3Config.TTL
		}
		if err := c.setInRedis(c.l3Redis, key, value, l3TTL); err != nil {
			log.Printf("Failed to set in L3 cache: %v", err)
		}
	}
	
	return nil
}

// SetAllLayers stores a value in all cache layers
func (c *MultiLayerCache) SetAllLayers(key string, value interface{}) error {
	cost := c.calculateCost(value)
	
	// L1 - short TTL for hot data
	c.l1Cache.SetWithTTL(key, value, cost, c.config.L1Config.TTL)
	
	// L2 - medium TTL
	if err := c.setInRedis(c.l2Redis, key, value, c.config.L2Config.TTL); err != nil {
		return fmt.Errorf("failed to set in L2: %w", err)
	}
	
	// L3 - long TTL
	if err := c.setInRedis(c.l3Redis, key, value, c.config.L3Config.TTL); err != nil {
		return fmt.Errorf("failed to set in L3: %w", err)
	}
	
	return nil
}

// Delete removes a key from all cache layers
func (c *MultiLayerCache) Delete(key string) error {
	// Delete from L1
	c.l1Cache.Del(key)

	// Delete from L2 (if available)
	if c.l2Redis != nil {
		if err := c.l2Redis.Del(c.ctx, key).Err(); err != nil {
			log.Printf("Failed to delete from L2 cache: %v", err)
		}
	}

	// Delete from L3 (if available)
	if c.l3Redis != nil {
		if err := c.l3Redis.Del(c.ctx, key).Err(); err != nil {
			log.Printf("Failed to delete from L3 cache: %v", err)
		}
	}

	return nil
}

// Exists checks if a key exists in any cache layer
func (c *MultiLayerCache) Exists(key string) (bool, error) {
	// Check L1 first (fastest)
	if value, found := c.l1Cache.Get(key); found && value != nil {
		return true, nil
	}

	// Check L2 Redis
	if c.l2Redis != nil {
		exists, err := c.l2Redis.Exists(c.ctx, key).Result()
		if err == nil && exists > 0 {
			return true, nil
		}
	}

	// Check L3 Redis
	if c.l3Redis != nil {
		exists, err := c.l3Redis.Exists(c.ctx, key).Result()
		if err == nil && exists > 0 {
			return true, nil
		}
	}

	return false, nil
}

// DeletePattern removes all keys matching a pattern from all cache layers
func (c *MultiLayerCache) DeletePattern(pattern string) error {
	// Delete from L1 (in-memory cache doesn't support pattern deletion, skip for now)
	// TODO: Implement pattern-based deletion for in-memory cache if needed

	// Delete from L2 Redis
	if c.l2Redis != nil {
		keys, err := c.l2Redis.Keys(c.ctx, pattern).Result()
		if err == nil && len(keys) > 0 {
			c.l2Redis.Del(c.ctx, keys...)
		}
	}

	// Delete from L3 Redis
	if c.l3Redis != nil {
		keys, err := c.l3Redis.Keys(c.ctx, pattern).Result()
		if err == nil && len(keys) > 0 {
			c.l3Redis.Del(c.ctx, keys...)
		}
	}

	return nil
}

// Invalidate removes keys based on a pattern
func (c *MultiLayerCache) Invalidate(pattern InvalidationPattern) error {
	select {
	case c.invalidationCh <- pattern:
		return nil
	default:
		// Channel is full, process immediately
		return c.processInvalidation(pattern)
	}
}

// WarmCache preloads frequently accessed data
func (c *MultiLayerCache) WarmCache(warmupData map[string]interface{}) error {
	c.warmupMutex.Lock()
	defer c.warmupMutex.Unlock()
	
	for key, value := range warmupData {
		if err := c.SetAllLayers(key, value); err != nil {
			log.Printf("Failed to warm cache for key %s: %v", key, err)
		}
	}
	
	return nil
}

// GetStatistics returns current cache statistics
func (c *MultiLayerCache) GetStatistics() CacheStatistics {
	c.stats.mutex.RLock()
	defer c.stats.mutex.RUnlock()
	
	// Update hit rates
	if c.stats.TotalRequests > 0 {
		c.stats.OverallHitRate = float64(c.stats.TotalHits) / float64(c.stats.TotalRequests)
	}
	
	return *c.stats
}

// Helper methods

func (c *MultiLayerCache) getFromRedis(client *redis.Client, key string) (interface{}, error) {
	data, err := client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("key not found")
	} else if err != nil {
		return nil, err
	}
	
	var value interface{}
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return nil, err
	}
	
	return value, nil
}

func (c *MultiLayerCache) setInRedis(client *redis.Client, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	
	return client.Set(c.ctx, key, data, ttl).Err()
}

func (c *MultiLayerCache) promoteToL1(key string, value interface{}) {
	cost := c.calculateCost(value)
	c.l1Cache.SetWithTTL(key, value, cost, c.config.L1Config.TTL)
}

func (c *MultiLayerCache) promoteToL2(key string, value interface{}, ttl time.Duration) {
	if err := c.setInRedis(c.l2Redis, key, value, ttl); err != nil {
		log.Printf("Failed to promote to L2: %v", err)
	}
}

func (c *MultiLayerCache) calculateCost(value interface{}) int64 {
	// Estimate cost based on serialized size
	data, err := json.Marshal(value)
	if err != nil {
		return 1 // Default cost
	}
	return int64(len(data))
}

func (c *MultiLayerCache) recordHit(layer CacheLayer) {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()
	
	c.stats.TotalHits++
	
	switch layer {
	case L1Memory:
		c.stats.L1Stats.Hits++
	case L2Redis:
		c.stats.L2Stats.Hits++
	case L3Persistent:
		c.stats.L3Stats.Hits++
	}
	
	c.updateLayerStats()
}

func (c *MultiLayerCache) recordMiss() {
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()
	
	c.stats.L1Stats.Misses++
	c.stats.L2Stats.Misses++
	c.stats.L3Stats.Misses++
	
	c.updateLayerStats()
}

func (c *MultiLayerCache) updateLayerStats() {
	now := time.Now()
	
	// Update L1 stats
	total := c.stats.L1Stats.Hits + c.stats.L1Stats.Misses
	if total > 0 {
		c.stats.L1Stats.HitRate = float64(c.stats.L1Stats.Hits) / float64(total)
	}
	c.stats.L1Stats.LastUpdated = now
	
	// Update L2 stats
	total = c.stats.L2Stats.Hits + c.stats.L2Stats.Misses
	if total > 0 {
		c.stats.L2Stats.HitRate = float64(c.stats.L2Stats.Hits) / float64(total)
	}
	c.stats.L2Stats.LastUpdated = now
	
	// Update L3 stats
	total = c.stats.L3Stats.Hits + c.stats.L3Stats.Misses
	if total > 0 {
		c.stats.L3Stats.HitRate = float64(c.stats.L3Stats.Hits) / float64(total)
	}
	c.stats.L3Stats.LastUpdated = now
}

func (c *MultiLayerCache) invalidationWorker() {
	for pattern := range c.invalidationCh {
		if err := c.processInvalidation(pattern); err != nil {
			log.Printf("Failed to process invalidation: %v", err)
		}
	}
}

func (c *MultiLayerCache) processInvalidation(pattern InvalidationPattern) error {
	switch pattern.Type {
	case "concept":
		return c.invalidateConcept(pattern)
	case "valueSet":
		return c.invalidateValueSet(pattern)
	case "expansion":
		return c.invalidateExpansion(pattern)
	case "cascade":
		return c.cascadeInvalidation(pattern)
	default:
		return c.invalidateByPattern(pattern.Pattern)
	}
}

func (c *MultiLayerCache) invalidateConcept(pattern InvalidationPattern) error {
	// Invalidate concept and related cache entries
	conceptKey := fmt.Sprintf("kb7:concept:%s:%s", pattern.System, pattern.Code)
	
	// Also invalidate search results that might contain this concept
	searchPattern := fmt.Sprintf("kb7:search:*")
	validationPattern := fmt.Sprintf("kb7:validation:%s:%s:*", pattern.Code, pattern.System)
	
	c.Delete(conceptKey)
	c.invalidateByPattern(searchPattern)
	c.invalidateByPattern(validationPattern)
	
	return nil
}

func (c *MultiLayerCache) invalidateValueSet(pattern InvalidationPattern) error {
	// Invalidate value set and all its expansions
	valueSetPattern := fmt.Sprintf("kb7:valueset:%s:*", pattern.Code)
	expansionPattern := fmt.Sprintf("kb7:expansion:%s:*", pattern.Code)
	
	c.invalidateByPattern(valueSetPattern)
	c.invalidateByPattern(expansionPattern)
	
	return nil
}

func (c *MultiLayerCache) invalidateExpansion(pattern InvalidationPattern) error {
	expansionPattern := fmt.Sprintf("kb7:expansion:%s:*", pattern.Code)
	return c.invalidateByPattern(expansionPattern)
}

func (c *MultiLayerCache) cascadeInvalidation(pattern InvalidationPattern) error {
	// Find dependent cached items and invalidate in order
	// This would require dependency tracking implementation
	log.Printf("Cascade invalidation not yet implemented for pattern: %+v", pattern)
	return nil
}

func (c *MultiLayerCache) invalidateByPattern(pattern string) error {
	// L1 cache doesn't support pattern deletion, so we'd need to track keys
	// For now, we'll clear Redis caches only

	// Invalidate L2 (if available)
	if c.l2Redis != nil {
		keys, err := c.l2Redis.Keys(c.ctx, pattern).Result()
		if err == nil && len(keys) > 0 {
			c.l2Redis.Del(c.ctx, keys...)
		}
	}

	// Invalidate L3 (if available)
	if c.l3Redis != nil {
		keys, err := c.l3Redis.Keys(c.ctx, pattern).Result()
		if err == nil && len(keys) > 0 {
			c.l3Redis.Del(c.ctx, keys...)
		}
	}

	return nil
}

func (c *MultiLayerCache) statisticsUpdater() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		c.updateStatistics()
	}
}

func (c *MultiLayerCache) updateStatistics() {
	// Update cache size information and other metrics
	// This would query Redis for memory usage statistics
	// For now, we'll just update timestamps
	
	c.stats.mutex.Lock()
	defer c.stats.mutex.Unlock()
	
	now := time.Now()
	c.stats.L1Stats.LastUpdated = now
	c.stats.L2Stats.LastUpdated = now
	c.stats.L3Stats.LastUpdated = now
}

// HealthCheck verifies cache connectivity and functionality
func (c *MultiLayerCache) HealthCheck() (bool, error) {
	// Test L1 cache (in-memory)
	testKey := "health_check_" + fmt.Sprintf("%d", time.Now().Unix())
	testValue := "ok"

	c.l1Cache.SetWithTTL(testKey, testValue, 1, time.Minute)
	if value, found := c.l1Cache.Get(testKey); !found || value != testValue {
		return false, fmt.Errorf("L1 cache health check failed")
	}
	c.l1Cache.Del(testKey)

	// Test L2 Redis if available
	if c.l2Redis != nil {
		if err := c.l2Redis.Set(c.ctx, testKey, testValue, time.Minute).Err(); err != nil {
			return false, fmt.Errorf("L2 cache health check failed: %w", err)
		}
		c.l2Redis.Del(c.ctx, testKey)
	}

	// Test L3 Redis if available
	if c.l3Redis != nil {
		if err := c.l3Redis.Set(c.ctx, testKey, testValue, time.Minute).Err(); err != nil {
			return false, fmt.Errorf("L3 cache health check failed: %w", err)
		}
		c.l3Redis.Del(c.ctx, testKey)
	}

	return true, nil
}

// Close cleanly shuts down the cache
func (c *MultiLayerCache) Close() error {
	close(c.invalidationCh)
	c.l1Cache.Close()

	if c.l2Redis != nil {
		c.l2Redis.Close()
	}

	if c.l3Redis != nil {
		c.l3Redis.Close()
	}
	
	return nil
}

// Utility functions for cache key generation

// ConceptCacheKey generates a cache key for concept lookups
func ConceptCacheKey(system, code string) string {
	return fmt.Sprintf("kb7:concept:%s:%s", system, code)
}

// SearchCacheKey generates a cache key for search results
func SearchCacheKey(query, systemURI string, count, offset int) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:%d:%d", query, systemURI, count, offset)))
	hash := hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 characters
	return fmt.Sprintf("kb7:search:%s", hash)
}

// ValueSetCacheKey generates a cache key for value sets
func ValueSetCacheKey(url, version string) string {
	return fmt.Sprintf("kb7:valueset:%s:%s", url, version)
}

// ExpansionCacheKey generates a cache key for value set expansions
func ExpansionCacheKey(url string, params map[string]interface{}) string {
	hasher := sha256.New()
	hasher.Write([]byte(url))
	
	// Add parameters to hash
	paramsJson, _ := json.Marshal(params)
	hasher.Write(paramsJson)
	
	hash := hex.EncodeToString(hasher.Sum(nil))[:16]
	return fmt.Sprintf("kb7:expansion:%s:%s", url, hash)
}

// ValidationCacheKey generates a cache key for code validations
func ValidationCacheKey(code, system, version string) string {
	return fmt.Sprintf("kb7:validation:%s:%s:%s", code, system, version)
}

// MappingCacheKey generates a cache key for concept mappings
func MappingCacheKey(sourceSystem, sourceCode, targetSystem string) string {
	return fmt.Sprintf("kb7:mapping:%s:%s:%s", sourceSystem, sourceCode, targetSystem)
}