package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// MultiTierCache implements a 3-tier caching strategy
type MultiTierCache struct {
	l1Cache  *MemoryCache  // L1: In-memory LRU cache
	l2Cache  *CacheClient  // L2: Redis distributed cache
	l3Cache  *CDNCache     // L3: CDN for static definitions
	logger   *zap.Logger
	metrics  *MultiTierMetrics
	config   *CacheConfig
}

type CacheConfig struct {
	L1TTL          time.Duration
	L2TTL          time.Duration
	L3TTL          time.Duration
	L1MaxSize      int
	L2MaxSize      int64
	EnableL3       bool
	CDNBaseURL     string
	PrefetchStatic bool
}

type MultiTierMetrics struct {
	L1Hits    int64
	L1Misses  int64
	L2Hits    int64
	L2Misses  int64
	L3Hits    int64
	L3Misses  int64
	L1Errors  int64
	L2Errors  int64
	L3Errors  int64
	mutex     sync.RWMutex
}

type CacheItem struct {
	Value      interface{} `json:"value"`
	Timestamp  time.Time   `json:"timestamp"`
	TTL        time.Duration `json:"ttl"`
	Source     string      `json:"source"` // "L1", "L2", "L3", "DB"
	Version    string      `json:"version,omitempty"`
}

// NewMultiTierCache creates a new multi-tier cache instance
func NewMultiTierCache(l2Cache *CacheClient, logger *zap.Logger, config *CacheConfig) *MultiTierCache {
	if config == nil {
		config = &CacheConfig{
			L1TTL:          5 * time.Minute,
			L2TTL:          1 * time.Hour,
			L3TTL:          24 * time.Hour,
			L1MaxSize:      10000,
			L2MaxSize:      1024 * 1024 * 1024, // 1GB
			EnableL3:       false, // Disabled by default until CDN setup
			CDNBaseURL:     "https://cdn.clinical-kb.health",
			PrefetchStatic: true,
		}
	}

	l1Cache := NewMemoryCache(config.L1MaxSize, config.L1TTL, logger)
	
	var l3Cache *CDNCache
	if config.EnableL3 && config.CDNBaseURL != "" {
		l3Cache = NewCDNCache(config.CDNBaseURL, logger)
	}

	mtc := &MultiTierCache{
		l1Cache: l1Cache,
		l2Cache: l2Cache,
		l3Cache: l3Cache,
		logger:  logger,
		metrics: &MultiTierMetrics{},
		config:  config,
	}

	// Prefetch static content if enabled
	if config.PrefetchStatic {
		go mtc.prefetchStaticContent(context.Background())
	}

	return mtc
}

// Get implements cache-aside pattern with cascade lookup
func (mtc *MultiTierCache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	
	// L1 Cache check
	if value, found := mtc.l1Cache.Get(key); found {
		mtc.recordL1Hit()
		mtc.logger.Debug("L1 cache hit", zap.String("key", key))
		return value, nil
	}
	mtc.recordL1Miss()

	// L2 Cache check
	if mtc.l2Cache != nil {
		if value, err := mtc.l2Cache.Get(ctx, key); err == nil && value != nil {
			mtc.recordL2Hit()
			mtc.logger.Debug("L2 cache hit", zap.String("key", key))
			
			// Promote to L1
			mtc.l1Cache.Set(key, value, mtc.config.L1TTL)
			return value, nil
		}
		mtc.recordL2Miss()
	}

	// L3 Cache check (for static content only)
	if mtc.l3Cache != nil && mtc.isStaticContent(key) {
		if value, err := mtc.l3Cache.Get(ctx, key); err == nil && value != nil {
			mtc.recordL3Hit()
			mtc.logger.Debug("L3 cache hit", zap.String("key", key))
			
			// Promote to L2 and L1
			if mtc.l2Cache != nil {
				mtc.l2Cache.Set(ctx, key, value, mtc.config.L2TTL)
			}
			mtc.l1Cache.Set(key, value, mtc.config.L1TTL)
			return value, nil
		}
		mtc.recordL3Miss()
	}

	mtc.logger.Debug("All cache tiers missed", 
		zap.String("key", key),
		zap.Duration("lookup_time", time.Since(start)))

	return nil, fmt.Errorf("cache miss on all tiers for key: %s", key)
}

// Set writes through all appropriate cache tiers
func (mtc *MultiTierCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Always write to L1
	mtc.l1Cache.Set(key, value, mtc.config.L1TTL)

	// Write to L2 if available
	if mtc.l2Cache != nil {
		if err := mtc.l2Cache.Set(ctx, key, value, mtc.config.L2TTL); err != nil {
			mtc.logger.Warn("Failed to write to L2 cache", 
				zap.String("key", key),
				zap.Error(err))
		}
	}

	// L3 is read-only (CDN)
	mtc.logger.Debug("Multi-tier cache set", 
		zap.String("key", key),
		zap.Duration("ttl", ttl))

	return nil
}

// Delete removes from all cache tiers
func (mtc *MultiTierCache) Delete(ctx context.Context, key string) error {
	// Delete from L1
	mtc.l1Cache.Delete(key)

	// Delete from L2
	if mtc.l2Cache != nil {
		if err := mtc.l2Cache.Delete(ctx, key); err != nil {
			mtc.logger.Warn("Failed to delete from L2 cache", 
				zap.String("key", key),
				zap.Error(err))
		}
	}

	// L3 CDN cannot be deleted from application layer

	mtc.logger.Debug("Multi-tier cache delete", zap.String("key", key))
	return nil
}

// InvalidatePattern removes all keys matching pattern from all tiers
func (mtc *MultiTierCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// Invalidate L1
	mtc.l1Cache.InvalidatePattern(pattern)

	// Invalidate L2
	if mtc.l2Cache != nil {
		if err := mtc.l2Cache.InvalidatePattern(ctx, pattern); err != nil {
			mtc.logger.Warn("Failed to invalidate L2 pattern", 
				zap.String("pattern", pattern),
				zap.Error(err))
		}
	}

	// Invalidate L3 ETags
	if mtc.l3Cache != nil {
		if err := mtc.l3Cache.InvalidatePattern(pattern); err != nil {
			mtc.logger.Warn("Failed to invalidate L3 pattern", 
				zap.String("pattern", pattern),
				zap.Error(err))
		}
	}

	mtc.logger.Info("Multi-tier pattern invalidation", zap.String("pattern", pattern))
	return nil
}

// GetOrCompute implements cache-aside pattern with computation
func (mtc *MultiTierCache) GetOrCompute(ctx context.Context, key string, computeFn func() (interface{}, error)) (interface{}, error) {
	// Try to get from cache first
	if value, err := mtc.Get(ctx, key); err == nil {
		return value, nil
	}

	// Compute the value
	value, err := computeFn()
	if err != nil {
		return nil, fmt.Errorf("compute function failed for key %s: %w", key, err)
	}

	// Store in cache
	if err := mtc.Set(ctx, key, value, mtc.config.L1TTL); err != nil {
		mtc.logger.Warn("Failed to cache computed value", 
			zap.String("key", key),
			zap.Error(err))
	}

	return value, nil
}

// Warmup preloads frequently accessed data
func (mtc *MultiTierCache) Warmup(ctx context.Context, keys []string) error {
	start := time.Now()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent warmup requests

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if already cached
			if _, err := mtc.Get(ctx, k); err == nil {
				return // Already cached
			}

			// For static content, try to preload from L3
			if mtc.l3Cache != nil && mtc.isStaticContent(k) {
				if value, err := mtc.l3Cache.Get(ctx, k); err == nil {
					mtc.Set(ctx, k, value, mtc.config.L1TTL)
				}
			}
		}(key)
	}

	wg.Wait()
	
	mtc.logger.Info("Cache warmup completed", 
		zap.Int("keys", len(keys)),
		zap.Duration("duration", time.Since(start)))

	return nil
}

// GetMetrics returns comprehensive metrics for all cache tiers
func (mtc *MultiTierCache) GetMetrics() map[string]interface{} {
	mtc.metrics.mutex.RLock()
	defer mtc.metrics.mutex.RUnlock()

	l1Total := mtc.metrics.L1Hits + mtc.metrics.L1Misses
	l2Total := mtc.metrics.L2Hits + mtc.metrics.L2Misses
	l3Total := mtc.metrics.L3Hits + mtc.metrics.L3Misses

	metrics := map[string]interface{}{
		"l1": map[string]interface{}{
			"hits":     mtc.metrics.L1Hits,
			"misses":   mtc.metrics.L1Misses,
			"errors":   mtc.metrics.L1Errors,
			"hit_rate": mtc.calculateHitRate(mtc.metrics.L1Hits, l1Total),
		},
		"l2": map[string]interface{}{
			"hits":     mtc.metrics.L2Hits,
			"misses":   mtc.metrics.L2Misses,
			"errors":   mtc.metrics.L2Errors,
			"hit_rate": mtc.calculateHitRate(mtc.metrics.L2Hits, l2Total),
		},
		"l3": map[string]interface{}{
			"hits":     mtc.metrics.L3Hits,
			"misses":   mtc.metrics.L3Misses,
			"errors":   mtc.metrics.L3Errors,
			"hit_rate": mtc.calculateHitRate(mtc.metrics.L3Hits, l3Total),
			"enabled":  mtc.l3Cache != nil,
		},
		"overall": map[string]interface{}{
			"total_hits":   mtc.metrics.L1Hits + mtc.metrics.L2Hits + mtc.metrics.L3Hits,
			"total_misses": mtc.metrics.L1Misses + mtc.metrics.L2Misses + mtc.metrics.L3Misses,
			"total_errors": mtc.metrics.L1Errors + mtc.metrics.L2Errors + mtc.metrics.L3Errors,
		},
	}

	// Add L1 memory cache metrics
	if mtc.l1Cache != nil {
		l1Metrics := mtc.l1Cache.GetMetrics()
		metrics["l1"].(map[string]interface{})["memory_usage"] = l1Metrics["memory_usage"]
		metrics["l1"].(map[string]interface{})["item_count"] = l1Metrics["item_count"]
	}

	// Add L3 CDN metrics if available
	if mtc.l3Cache != nil {
		l3Metrics := mtc.l3Cache.GetMetrics()
		for k, v := range l3Metrics {
			metrics["l3"].(map[string]interface{})[k] = v
		}
	}

	return metrics
}

// Helper methods

func (mtc *MultiTierCache) isStaticContent(key string) bool {
	staticPrefixes := []string{
		"phenotypes:",
		"risk-models:",
		"treatment-preferences:",
		"static:",
		"definitions:",
	}

	for _, prefix := range staticPrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func (mtc *MultiTierCache) calculateHitRate(hits, total int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total) * 100.0
}

// Metrics recording methods
func (mtc *MultiTierCache) recordL1Hit() {
	mtc.metrics.mutex.Lock()
	mtc.metrics.L1Hits++
	mtc.metrics.mutex.Unlock()
}

func (mtc *MultiTierCache) recordL1Miss() {
	mtc.metrics.mutex.Lock()
	mtc.metrics.L1Misses++
	mtc.metrics.mutex.Unlock()
}

func (mtc *MultiTierCache) recordL2Hit() {
	mtc.metrics.mutex.Lock()
	mtc.metrics.L2Hits++
	mtc.metrics.mutex.Unlock()
}

func (mtc *MultiTierCache) recordL2Miss() {
	mtc.metrics.mutex.Lock()
	mtc.metrics.L2Misses++
	mtc.metrics.mutex.Unlock()
}

func (mtc *MultiTierCache) recordL3Hit() {
	mtc.metrics.mutex.Lock()
	mtc.metrics.L3Hits++
	mtc.metrics.mutex.Unlock()
}

func (mtc *MultiTierCache) recordL3Miss() {
	mtc.metrics.mutex.Lock()
	mtc.metrics.L3Misses++
	mtc.metrics.mutex.Unlock()
}

// prefetchStaticContent loads frequently accessed static content
func (mtc *MultiTierCache) prefetchStaticContent(ctx context.Context) {
	if mtc.l3Cache == nil {
		return
	}

	staticKeys := []string{
		"phenotypes/cardiovascular/v2.0",
		"phenotypes/diabetes/v2.0",
		"risk-models/cardiovascular/v1.0",
		"risk-models/diabetes/v1.0",
		"treatment-preferences/hypertension/v1.0",
		"treatment-preferences/diabetes/v1.0",
	}

	mtc.logger.Info("Starting static content prefetch", zap.Int("keys", len(staticKeys)))

	for _, key := range staticKeys {
		if value, err := mtc.l3Cache.Get(ctx, key); err == nil {
			// Load into L2 and L1
			if mtc.l2Cache != nil {
				mtc.l2Cache.Set(ctx, key, value, mtc.config.L2TTL)
			}
			mtc.l1Cache.Set(key, value, mtc.config.L1TTL)
			
			mtc.logger.Debug("Prefetched static content", zap.String("key", key))
		}
	}

	mtc.logger.Info("Static content prefetch completed")
}

// Health check for all cache tiers
func (mtc *MultiTierCache) Health(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"l1": map[string]interface{}{"status": "ok"},
		"l2": map[string]interface{}{"status": "unknown"},
		"l3": map[string]interface{}{"status": "disabled"},
	}

	// Check L1 (always healthy - in-memory)
	if mtc.l1Cache != nil {
		health["l1"] = map[string]interface{}{
			"status":     "ok",
			"item_count": mtc.l1Cache.ItemCount(),
		}
	}

	// Check L2 Redis
	if mtc.l2Cache != nil {
		if err := mtc.l2Cache.Health(ctx); err != nil {
			health["l2"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			health["l2"] = map[string]interface{}{
				"status": "ok",
			}
		}
	}

	// Check L3 CDN
	if mtc.l3Cache != nil {
		if err := mtc.l3Cache.Health(ctx); err != nil {
			health["l3"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			health["l3"] = map[string]interface{}{
				"status": "ok",
			}
		}
	}

	return health
}

// Close gracefully shuts down all cache tiers
func (mtc *MultiTierCache) Close() error {
	var errors []string

	// Close L1
	if mtc.l1Cache != nil {
		if err := mtc.l1Cache.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("L1: %v", err))
		}
	}

	// Close L2
	if mtc.l2Cache != nil {
		if err := mtc.l2Cache.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("L2: %v", err))
		}
	}

	// Close L3
	if mtc.l3Cache != nil {
		if err := mtc.l3Cache.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("L3: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache tier close errors: %v", errors)
	}

	mtc.logger.Info("Multi-tier cache closed successfully")
	return nil
}

// Performance monitoring methods

// CacheClient compatibility methods
// These methods provide compatibility with the existing CacheClient interface

// HealthCheck provides compatibility with the CacheClient interface
func (mtc *MultiTierCache) HealthCheck() error {
	if mtc.l2Cache != nil {
		return mtc.l2Cache.HealthCheck()
	}
	return nil
}

// Get method for Redis-style key operations
func (mtc *MultiTierCache) GetRedisKey(ctx context.Context, key string) ([]byte, error) {
	value, err := mtc.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, nil
	}
	
	return json.Marshal(value)
}

// Set method for Redis-style key operations
func (mtc *MultiTierCache) SetRedisKey(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return mtc.Set(ctx, key, value, ttl)
}

// Patient Context compatibility methods
func (mtc *MultiTierCache) GetPatientContext(patientID string) ([]byte, error) {
	key := fmt.Sprintf("context:patient:%s", patientID)
	return mtc.GetRedisKey(context.Background(), key)
}

func (mtc *MultiTierCache) CachePatientContext(patientID string, ctxData interface{}) error {
	key := fmt.Sprintf("context:patient:%s", patientID)
	return mtc.Set(context.Background(), key, ctxData, mtc.config.L2TTL)
}

func (mtc *MultiTierCache) InvalidatePatientContext(patientID string) error {
	key := fmt.Sprintf("context:patient:%s", patientID)
	return mtc.Delete(context.Background(), key)
}

// Phenotype compatibility methods
func (mtc *MultiTierCache) GetPhenotypes() ([]byte, error) {
	key := "phenotypes:active"
	return mtc.GetRedisKey(context.Background(), key)
}

func (mtc *MultiTierCache) CachePhenotypes(phenotypes interface{}) error {
	key := "phenotypes:active"
	return mtc.Set(context.Background(), key, phenotypes, mtc.config.L2TTL)
}

func (mtc *MultiTierCache) InvalidatePhenotypes() error {
	key := "phenotypes:active"
	return mtc.Delete(context.Background(), key)
}

// Risk Assessment compatibility methods
func (mtc *MultiTierCache) GetRiskAssessment(patientID string, riskType string) ([]byte, error) {
	key := fmt.Sprintf("risk:%s:%s", patientID, riskType)
	return mtc.GetRedisKey(context.Background(), key)
}

func (mtc *MultiTierCache) CacheRiskAssessment(patientID string, riskType string, assessment interface{}) error {
	key := fmt.Sprintf("risk:%s:%s", patientID, riskType)
	return mtc.Set(context.Background(), key, assessment, 2*time.Hour)
}

func (mtc *MultiTierCache) InvalidateRiskAssessments(patientID string) error {
	pattern := fmt.Sprintf("risk:%s:*", patientID)
	return mtc.InvalidatePattern(context.Background(), pattern)
}

// Context Stats compatibility methods
func (mtc *MultiTierCache) GetContextStats() ([]byte, error) {
	key := "stats:context"
	return mtc.GetRedisKey(context.Background(), key)
}

func (mtc *MultiTierCache) CacheContextStats(stats interface{}) error {
	key := "stats:context"
	return mtc.Set(context.Background(), key, stats, 30*time.Minute)
}

// GetStats returns Redis-compatible stats format
func (mtc *MultiTierCache) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"multi_tier_metrics": mtc.GetMetrics(),
		"connected":          mtc.l2Cache != nil,
	}
}

// GetPerformanceStats returns detailed performance statistics
func (mtc *MultiTierCache) GetPerformanceStats() map[string]interface{} {
	mtc.metrics.mutex.RLock()
	defer mtc.metrics.mutex.RUnlock()

	totalRequests := mtc.metrics.L1Hits + mtc.metrics.L1Misses
	overallHitRate := float64(0)
	if totalRequests > 0 {
		overallHitRate = float64(mtc.metrics.L1Hits) / float64(totalRequests) * 100
	}

	return map[string]interface{}{
		"performance": map[string]interface{}{
			"overall_hit_rate":     overallHitRate,
			"l1_hit_rate":          mtc.calculateHitRate(mtc.metrics.L1Hits, mtc.metrics.L1Hits+mtc.metrics.L1Misses),
			"l2_hit_rate":          mtc.calculateHitRate(mtc.metrics.L2Hits, mtc.metrics.L2Hits+mtc.metrics.L2Misses),
			"l3_hit_rate":          mtc.calculateHitRate(mtc.metrics.L3Hits, mtc.metrics.L3Hits+mtc.metrics.L3Misses),
			"total_requests":       totalRequests,
		},
		"sla_compliance": map[string]interface{}{
			"l1_target_hit_rate":   85.0,
			"l2_target_hit_rate":   95.0,
			"l1_actual_hit_rate":   mtc.calculateHitRate(mtc.metrics.L1Hits, mtc.metrics.L1Hits+mtc.metrics.L1Misses),
			"l2_actual_hit_rate":   mtc.calculateHitRate(mtc.metrics.L2Hits, mtc.metrics.L2Hits+mtc.metrics.L2Misses),
			"l1_sla_met":          mtc.calculateHitRate(mtc.metrics.L1Hits, mtc.metrics.L1Hits+mtc.metrics.L1Misses) >= 85.0,
			"l2_sla_met":          mtc.calculateHitRate(mtc.metrics.L2Hits, mtc.metrics.L2Hits+mtc.metrics.L2Misses) >= 95.0,
		},
	}
}