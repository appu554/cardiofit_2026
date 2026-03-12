package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/metrics"
	"kb-2-clinical-context-go/internal/models"
)

// CacheItem represents a cached item with metadata
type CacheItem struct {
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
	TTL       time.Duration `json:"ttl"`
	Version   string      `json:"version"`
	Size      int         `json:"size"`
	AccessCount int       `json:"access_count"`
	LastAccessed time.Time `json:"last_accessed"`
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	HitRate      float64 `json:"hit_rate"`
	MissRate     float64 `json:"miss_rate"`
	Size         int     `json:"size"`
	Evictions    int64   `json:"evictions"`
	Operations   int64   `json:"operations"`
	MemoryUsage  int64   `json:"memory_usage_bytes"`
	LastEviction time.Time `json:"last_eviction"`
}

// MultiTierCache implements a comprehensive 3-tier caching strategy
type MultiTierCache struct {
	config  *config.Config
	metrics *metrics.PrometheusMetrics
	
	// L1 Cache - In-Memory LRU (5min TTL, 100MB limit)
	l1Cache *MemoryCache
	
	// L2 Cache - Redis Distributed (1hr TTL, 1GB per node)
	l2Cache *RedisCache
	
	// L3 Cache - CDN Static (immutable content with versioning)
	l3Cache *CDNCache
	
	// Cache coordination
	mu sync.RWMutex
	
	// Performance tracking
	stats     map[string]*CacheStats
	statsMu   sync.RWMutex
	
	// Cache warming
	warmer *CacheWarmer
}

// NewMultiTierCache creates a new multi-tier cache system
func NewMultiTierCache(cfg *config.Config, redisClient *redis.Client, metricsCollector *metrics.PrometheusMetrics) *MultiTierCache {
	cache := &MultiTierCache{
		config:  cfg,
		metrics: metricsCollector,
		stats:   make(map[string]*CacheStats),
	}
	
	// Initialize L1 Cache - In-Memory LRU
	cache.l1Cache = NewMemoryCache(&MemoryCacheConfig{
		MaxSize:     100 * 1024 * 1024, // 100MB
		DefaultTTL:  5 * time.Minute,   // 5min TTL
		MaxItems:    10000,             // Max 10k items
		EvictionRate: 0.1,              // Evict 10% when full
		HitRateTarget: 0.85,            // 85% hit rate target
	}, metricsCollector)
	
	// Initialize L2 Cache - Redis Distributed
	cache.l2Cache = NewRedisCache(&RedisCacheConfig{
		Client:        redisClient,
		DefaultTTL:    time.Hour,       // 1hr TTL
		MaxMemory:     1024 * 1024 * 1024, // 1GB per node
		KeyPrefix:     "kb2:l2:",
		Compression:   true,            // Enable compression for large objects
		HitRateTarget: 0.95,            // 95% hit rate target
	}, metricsCollector)
	
	// Initialize L3 Cache - CDN Static
	cache.l3Cache = NewCDNCache(&CDNCacheConfig{
		BaseURL:       cfg.GetStringEnv("CDN_BASE_URL", "https://cdn.clinical-hub.com"),
		VersionPrefix: "v1",
		CacheHeaders:  map[string]string{
			"Cache-Control": "public, max-age=86400, immutable",
			"ETag":          "\"phenotype-definitions-v1\"",
		},
		StaticPaths: []string{
			"/phenotypes",
			"/risk-models", 
			"/treatment-preferences",
		},
	}, metricsCollector)
	
	// Initialize cache warmer
	cache.warmer = NewCacheWarmer(cache, cfg)
	
	// Initialize stats
	cache.initializeStats()
	
	return cache
}

// Get retrieves data using cache-aside pattern: L1 → L2 → L3 → Database
func (mtc *MultiTierCache) Get(ctx context.Context, key string, loader func() (interface{}, error)) (interface{}, error) {
	startTime := time.Now()
	defer func() {
		mtc.recordOperation("get", time.Since(startTime))
	}()
	
	// Try L1 Cache first (in-memory)
	if data, found := mtc.l1Cache.Get(key); found {
		mtc.recordHit("l1", key)
		mtc.metrics.RecordCacheHit("l1")
		return data, nil
	}
	mtc.recordMiss("l1", key)
	mtc.metrics.RecordCacheMiss("l1")
	
	// Try L2 Cache (Redis)
	if data, found := mtc.l2Cache.Get(ctx, key); found {
		mtc.recordHit("l2", key)
		mtc.metrics.RecordCacheHit("l2")
		
		// Promote to L1 cache asynchronously
		go func() {
			mtc.l1Cache.Set(key, data, 5*time.Minute)
		}()
		
		return data, nil
	}
	mtc.recordMiss("l2", key)
	mtc.metrics.RecordCacheMiss("l2")
	
	// Try L3 Cache (CDN/Static)
	if mtc.isStaticContent(key) {
		if data, found := mtc.l3Cache.Get(ctx, key); found {
			mtc.recordHit("l3", key)
			mtc.metrics.RecordCacheHit("l3")
			
			// Promote to L2 and L1 caches asynchronously
			go func() {
				mtc.l2Cache.Set(ctx, key, data, time.Hour)
				mtc.l1Cache.Set(key, data, 5*time.Minute)
			}()
			
			return data, nil
		}
	}
	mtc.recordMiss("l3", key)
	mtc.metrics.RecordCacheMiss("l3")
	
	// Cache miss - load from source
	data, err := loader()
	if err != nil {
		return nil, fmt.Errorf("failed to load data for key %s: %w", key, err)
	}
	
	// Store in all appropriate cache tiers asynchronously
	go mtc.setMultiTier(ctx, key, data)
	
	return data, nil
}

// Set stores data in appropriate cache tiers based on content type
func (mtc *MultiTierCache) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	return mtc.setWithTTL(ctx, key, data, ttl)
}

// setMultiTier stores data in appropriate cache tiers
func (mtc *MultiTierCache) setMultiTier(ctx context.Context, key string, data interface{}) {
	// Always store in L1 (memory)
	mtc.l1Cache.Set(key, data, 5*time.Minute)
	
	// Store in L2 (Redis) for distributed access
	mtc.l2Cache.Set(ctx, key, data, time.Hour)
	
	// Store in L3 (CDN) if it's static content
	if mtc.isStaticContent(key) {
		mtc.l3Cache.Set(ctx, key, data, 24*time.Hour) // 24h TTL for static content
	}
}

// setWithTTL stores data with specific TTL
func (mtc *MultiTierCache) setWithTTL(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	// Store in L1 with shorter TTL (min of requested TTL and 5min)
	l1TTL := ttl
	if l1TTL > 5*time.Minute {
		l1TTL = 5 * time.Minute
	}
	mtc.l1Cache.Set(key, data, l1TTL)
	
	// Store in L2 with requested TTL (up to 1 hour)
	l2TTL := ttl
	if l2TTL > time.Hour {
		l2TTL = time.Hour
	}
	return mtc.l2Cache.Set(ctx, key, data, l2TTL)
}

// GetBatch retrieves multiple items efficiently using parallel cache lookups
func (mtc *MultiTierCache) GetBatch(ctx context.Context, keys []string, loader func([]string) (map[string]interface{}, error)) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	missedKeys := []string{}
	
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// Parallel cache lookups
	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			
			// Try cache tiers
			if data, found := mtc.l1Cache.Get(k); found {
				mu.Lock()
				results[k] = data
				mu.Unlock()
				mtc.metrics.RecordCacheHit("l1")
				return
			}
			
			if data, found := mtc.l2Cache.Get(ctx, k); found {
				mu.Lock()
				results[k] = data
				mu.Unlock()
				mtc.metrics.RecordCacheHit("l2")
				
				// Promote to L1
				go mtc.l1Cache.Set(k, data, 5*time.Minute)
				return
			}
			
			// Add to missed keys
			mu.Lock()
			missedKeys = append(missedKeys, k)
			mu.Unlock()
			mtc.metrics.RecordCacheMiss("l1")
			mtc.metrics.RecordCacheMiss("l2")
		}(key)
	}
	
	wg.Wait()
	
	// Load missed keys from source
	if len(missedKeys) > 0 {
		loadedData, err := loader(missedKeys)
		if err != nil {
			return results, fmt.Errorf("failed to load missed keys: %w", err)
		}
		
		// Merge loaded data and cache asynchronously
		for key, data := range loadedData {
			results[key] = data
			
			// Cache in background
			go mtc.setMultiTier(ctx, key, data)
		}
	}
	
	return results, nil
}

// Invalidate removes data from all cache tiers
func (mtc *MultiTierCache) Invalidate(ctx context.Context, key string) error {
	var errors []error
	
	// Invalidate L1
	mtc.l1Cache.Delete(key)
	
	// Invalidate L2
	if err := mtc.l2Cache.Delete(ctx, key); err != nil {
		errors = append(errors, fmt.Errorf("L2 invalidation failed: %w", err))
	}
	
	// Invalidate L3 (if applicable)
	if mtc.isStaticContent(key) {
		if err := mtc.l3Cache.Invalidate(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("L3 invalidation failed: %w", err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("partial invalidation failures: %v", errors)
	}
	
	mtc.recordInvalidation(key)
	return nil
}

// InvalidatePattern removes all keys matching a pattern from all cache tiers
func (mtc *MultiTierCache) InvalidatePattern(ctx context.Context, pattern string) error {
	var errors []error
	
	// Invalidate L1 by pattern
	if err := mtc.l1Cache.DeletePattern(pattern); err != nil {
		errors = append(errors, fmt.Errorf("L1 pattern invalidation failed: %w", err))
	}
	
	// Invalidate L2 by pattern
	if err := mtc.l2Cache.DeletePattern(ctx, pattern); err != nil {
		errors = append(errors, fmt.Errorf("L2 pattern invalidation failed: %w", err))
	}
	
	// Invalidate L3 by pattern (if static content)
	if mtc.isStaticContentPattern(pattern) {
		if err := mtc.l3Cache.InvalidatePattern(ctx, pattern); err != nil {
			errors = append(errors, fmt.Errorf("L3 pattern invalidation failed: %w", err))
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("partial pattern invalidation failures: %v", errors)
	}
	
	return nil
}

// WarmCache preloads frequently accessed data
func (mtc *MultiTierCache) WarmCache(ctx context.Context) error {
	return mtc.warmer.WarmCache(ctx)
}

// GetStats returns comprehensive cache statistics
func (mtc *MultiTierCache) GetStats() map[string]*CacheStats {
	mtc.statsMu.RLock()
	defer mtc.statsMu.RUnlock()
	
	stats := make(map[string]*CacheStats)
	for tier, stat := range mtc.stats {
		// Create copy to avoid race conditions
		statsCopy := *stat
		stats[tier] = &statsCopy
	}
	
	// Add real-time stats from individual caches
	if l1Stats := mtc.l1Cache.GetStats(); l1Stats != nil {
		if existing, ok := stats["l1"]; ok {
			existing.MemoryUsage = l1Stats.MemoryUsage
			existing.Size = l1Stats.Size
		}
	}
	
	return stats
}

// GetHitRates returns current hit rates for all tiers
func (mtc *MultiTierCache) GetHitRates() map[string]float64 {
	stats := mtc.GetStats()
	hitRates := make(map[string]float64)
	
	for tier, stat := range stats {
		hitRates[tier] = stat.HitRate
	}
	
	return hitRates
}

// CheckSLACompliance verifies if cache performance meets SLA targets
func (mtc *MultiTierCache) CheckSLACompliance() map[string]bool {
	hitRates := mtc.GetHitRates()
	
	compliance := map[string]bool{
		"l1_hit_rate": hitRates["l1"] >= 0.85, // 85% target
		"l2_hit_rate": hitRates["l2"] >= 0.95, // 95% target
		"combined_performance": (hitRates["l1"] >= 0.85 && hitRates["l2"] >= 0.95),
	}
	
	return compliance
}

// OptimizeCache performs cache optimization based on usage patterns
func (mtc *MultiTierCache) OptimizeCache(ctx context.Context) error {
	stats := mtc.GetStats()
	
	// Optimize L1 cache if hit rate is below target
	if stats["l1"].HitRate < 0.85 {
		if err := mtc.l1Cache.Optimize(); err != nil {
			log.Printf("L1 cache optimization failed: %v", err)
		}
	}
	
	// Optimize L2 cache if hit rate is below target  
	if stats["l2"].HitRate < 0.95 {
		if err := mtc.l2Cache.Optimize(ctx); err != nil {
			log.Printf("L2 cache optimization failed: %v", err)
		}
	}
	
	// Trigger cache warming for frequently accessed data
	return mtc.warmer.OptimizeWarmingStrategy(ctx, stats)
}

// Utility methods for cache management

// isStaticContent determines if content should be cached in L3 (CDN)
func (mtc *MultiTierCache) isStaticContent(key string) bool {
	staticPrefixes := []string{
		"phenotype_definition:",
		"risk_model:",
		"treatment_preference_template:",
		"institutional_rule:",
		"static:",
	}
	
	for _, prefix := range staticPrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// isStaticContentPattern determines if pattern matches static content
func (mtc *MultiTierCache) isStaticContentPattern(pattern string) bool {
	staticPatterns := []string{
		"phenotype_definition:*",
		"risk_model:*",
		"treatment_preference_template:*",
		"institutional_rule:*",
		"static:*",
	}
	
	for _, staticPattern := range staticPatterns {
		if pattern == staticPattern || mtc.patternMatches(pattern, staticPattern) {
			return true
		}
	}
	return false
}

// patternMatches checks if pattern matches static pattern
func (mtc *MultiTierCache) patternMatches(pattern, staticPattern string) bool {
	// Simple wildcard matching for cache patterns
	if len(staticPattern) > 0 && staticPattern[len(staticPattern)-1] == '*' {
		prefix := staticPattern[:len(staticPattern)-1]
		return len(pattern) >= len(prefix) && pattern[:len(prefix)] == prefix
	}
	return pattern == staticPattern
}

// Statistics and monitoring methods

// initializeStats initializes cache statistics tracking
func (mtc *MultiTierCache) initializeStats() {
	mtc.statsMu.Lock()
	defer mtc.statsMu.Unlock()
	
	tiers := []string{"l1", "l2", "l3", "combined"}
	for _, tier := range tiers {
		mtc.stats[tier] = &CacheStats{
			HitRate:     0.0,
			MissRate:    1.0,
			Size:        0,
			Evictions:   0,
			Operations:  0,
			MemoryUsage: 0,
		}
	}
}

// recordHit records a cache hit for statistics
func (mtc *MultiTierCache) recordHit(tier, key string) {
	mtc.statsMu.Lock()
	defer mtc.statsMu.Unlock()
	
	if stat, exists := mtc.stats[tier]; exists {
		stat.Operations++
		// Recalculate hit rate (running average)
		hits := stat.Operations * stat.HitRate + 1
		stat.HitRate = hits / float64(stat.Operations)
		stat.MissRate = 1.0 - stat.HitRate
	}
	
	// Update combined stats
	if combined, exists := mtc.stats["combined"]; exists {
		combined.Operations++
		hits := combined.Operations * combined.HitRate + 1
		combined.HitRate = hits / float64(combined.Operations)
		combined.MissRate = 1.0 - combined.HitRate
	}
}

// recordMiss records a cache miss for statistics
func (mtc *MultiTierCache) recordMiss(tier, key string) {
	mtc.statsMu.Lock()
	defer mtc.statsMu.Unlock()
	
	if stat, exists := mtc.stats[tier]; exists {
		stat.Operations++
		// Recalculate hit rate (running average)
		hits := stat.Operations * stat.HitRate // Don't increment hits for miss
		stat.HitRate = hits / float64(stat.Operations)
		stat.MissRate = 1.0 - stat.HitRate
	}
}

// recordOperation records a cache operation
func (mtc *MultiTierCache) recordOperation(operation string, duration time.Duration) {
	mtc.metrics.CacheOperations.WithLabelValues(operation, "multi_tier").Inc()
	
	// Record operation duration (convert to seconds for Prometheus)
	// Note: You might want to add a duration histogram metric
}

// recordInvalidation records a cache invalidation
func (mtc *MultiTierCache) recordInvalidation(key string) {
	mtc.metrics.CacheOperations.WithLabelValues("invalidate", "multi_tier").Inc()
}

// Specialized cache methods for KB-2 domain objects

// GetPhenotypeDefinition retrieves cached phenotype definition
func (mtc *MultiTierCache) GetPhenotypeDefinition(ctx context.Context, phenotypeID string, loader func() (*models.PhenotypeDefinition, error)) (*models.PhenotypeDefinition, error) {
	key := fmt.Sprintf("phenotype_definition:%s", phenotypeID)
	
	data, err := mtc.Get(ctx, key, func() (interface{}, error) {
		return loader()
	})
	
	if err != nil {
		return nil, err
	}
	
	if phenotype, ok := data.(*models.PhenotypeDefinition); ok {
		return phenotype, nil
	}
	
	return nil, fmt.Errorf("invalid data type for phenotype definition")
}

// GetPatientContext retrieves cached patient context
func (mtc *MultiTierCache) GetPatientContext(ctx context.Context, patientID string, contextType string, loader func() (*models.ClinicalContext, error)) (*models.ClinicalContext, error) {
	key := fmt.Sprintf("patient_context:%s:%s", patientID, contextType)
	
	data, err := mtc.Get(ctx, key, func() (interface{}, error) {
		return loader()
	})
	
	if err != nil {
		return nil, err
	}
	
	if context, ok := data.(*models.ClinicalContext); ok {
		return context, nil
	}
	
	return nil, fmt.Errorf("invalid data type for patient context")
}

// GetRiskAssessment retrieves cached risk assessment
func (mtc *MultiTierCache) GetRiskAssessment(ctx context.Context, patientID string, riskType string, loader func() (*models.RiskAssessmentResult, error)) (*models.RiskAssessmentResult, error) {
	key := fmt.Sprintf("risk_assessment:%s:%s", patientID, riskType)
	
	data, err := mtc.Get(ctx, key, func() (interface{}, error) {
		return loader()
	})
	
	if err != nil {
		return nil, err
	}
	
	if risk, ok := data.(*models.RiskAssessmentResult); ok {
		return risk, nil
	}
	
	return nil, fmt.Errorf("invalid data type for risk assessment")
}

// GetTreatmentPreferences retrieves cached treatment preferences
func (mtc *MultiTierCache) GetTreatmentPreferences(ctx context.Context, patientID string, condition string, loader func() (*models.TreatmentPreferencesResult, error)) (*models.TreatmentPreferencesResult, error) {
	key := fmt.Sprintf("treatment_preferences:%s:%s", patientID, condition)
	
	data, err := mtc.Get(ctx, key, func() (interface{}, error) {
		return loader()
	})
	
	if err != nil {
		return nil, err
	}
	
	if prefs, ok := data.(*models.TreatmentPreferencesResult); ok {
		return prefs, nil
	}
	
	return nil, fmt.Errorf("invalid data type for treatment preferences")
}

// Cache maintenance methods

// Cleanup performs cache cleanup and maintenance
func (mtc *MultiTierCache) Cleanup(ctx context.Context) error {
	var errors []error
	
	// Cleanup L1 cache
	if err := mtc.l1Cache.Cleanup(); err != nil {
		errors = append(errors, fmt.Errorf("L1 cleanup failed: %w", err))
	}
	
	// Cleanup L2 cache  
	if err := mtc.l2Cache.Cleanup(ctx); err != nil {
		errors = append(errors, fmt.Errorf("L2 cleanup failed: %w", err))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("cache cleanup had errors: %v", errors)
	}
	
	return nil
}

// StartBackgroundOptimization starts background optimization routines
func (mtc *MultiTierCache) StartBackgroundOptimization(ctx context.Context) {
	// Start cache optimization every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := mtc.OptimizeCache(ctx); err != nil {
					log.Printf("Background cache optimization failed: %v", err)
				}
			}
		}
	}()
	
	// Start cache warming every 15 minutes
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := mtc.WarmCache(ctx); err != nil {
					log.Printf("Background cache warming failed: %v", err)
				}
			}
		}
	}()
	
	// Start statistics reporting every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				mtc.reportStatisticsToMetrics()
			}
		}
	}()
}

// reportStatisticsToMetrics reports cache statistics to Prometheus metrics
func (mtc *MultiTierCache) reportStatisticsToMetrics() {
	stats := mtc.GetStats()
	
	// Report hit rates to metrics (you may need to add these metrics to prometheus.go)
	for tier, stat := range stats {
		// Record hit rate as a gauge metric
		// Note: You might need to add hit rate gauges to metrics.PrometheusMetrics
		log.Printf("Cache %s: Hit Rate=%.2f%%, Miss Rate=%.2f%%, Size=%d, Operations=%d", 
			tier, stat.HitRate*100, stat.MissRate*100, stat.Size, stat.Operations)
	}
}

// Configuration helper methods for the config struct
func (c *config.Config) GetStringEnv(key, defaultValue string) string {
	// This method should be added to config.go if it doesn't exist
	// For now, return the default value
	return defaultValue
}