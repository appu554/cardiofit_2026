package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PerformanceCache implements high-performance caching optimizations
type PerformanceCache struct {
	cache       *MultiLevelCache
	redis       *redis.Client
	logger      *zap.Logger
	
	// Hot cache for ultra-fast access (<1ms)
	hotCache    *sync.Map
	hotCacheTTL time.Duration
	hotMaxSize  int64
	hotCount    int64
	
	// Write-through cache for consistency
	writeThrough bool
	
	// Connection pooling optimization
	pipelineSize int
	
	// Cache warming
	warmupScheduler *CacheWarmupScheduler
	
	// Performance monitoring
	perfMetrics *PerformanceMetrics
}

// PerformanceMetrics tracks detailed performance statistics
type PerformanceMetrics struct {
	mutex sync.RWMutex
	
	// Latency tracking
	L1AvgLatency time.Duration `json:"l1_avg_latency"`
	L2AvgLatency time.Duration `json:"l2_avg_latency"`
	L3AvgLatency time.Duration `json:"l3_avg_latency"`
	
	// Throughput tracking
	RequestsPerSecond float64   `json:"requests_per_second"`
	LastSecondCount   int64     `json:"last_second_count"`
	LastSecondTime    time.Time `json:"last_second_time"`
	
	// Hot cache performance
	HotCacheHitRate   float64 `json:"hot_cache_hit_rate"`
	HotCacheLatency   time.Duration `json:"hot_cache_latency"`
	
	// Pipeline performance
	PipelineUtilization float64 `json:"pipeline_utilization"`
	PipelineLatency     time.Duration `json:"pipeline_latency"`
	
	// Error rates
	ErrorRate     float64 `json:"error_rate"`
	TimeoutRate   float64 `json:"timeout_rate"`
	
	// Memory usage
	MemoryUsage   int64 `json:"memory_usage"`
	MaxMemoryUsage int64 `json:"max_memory_usage"`
}

// CacheWarmupScheduler handles proactive cache warming
type CacheWarmupScheduler struct {
	cache       *PerformanceCache
	logger      *zap.Logger
	warmupRules []WarmupRule
	scheduler   *time.Ticker
	mutex       sync.RWMutex
}

// WarmupRule defines cache warming strategy
type WarmupRule struct {
	Name         string                               `json:"name"`
	Pattern      string                               `json:"pattern"`
	Frequency    time.Duration                        `json:"frequency"`
	Priority     int                                  `json:"priority"`
	DataProvider func(ctx context.Context) (map[string]interface{}, error) `json:"-"`
	TTL          time.Duration                        `json:"ttl"`
	Tags         []string                             `json:"tags"`
}

// HotCacheEntry represents ultra-fast cache entry
type HotCacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	HitCount  int64       `json:"hit_count"`
	Size      int64       `json:"size"`
}

// NewPerformanceCache creates an optimized high-performance cache
func NewPerformanceCache(cacheManager *MultiLevelCache, redisClient *redis.Client, logger *zap.Logger) *PerformanceCache {
	pc := &PerformanceCache{
		cache:        cacheManager,
		redis:        redisClient,
		logger:       logger.Named("performance_cache"),
		hotCache:     &sync.Map{},
		hotCacheTTL:  5 * time.Minute,
		hotMaxSize:   1000, // 1000 hot entries max
		writeThrough: true,
		pipelineSize: 100,
		perfMetrics: &PerformanceMetrics{
			LastSecondTime: time.Now(),
		},
	}
	
	// Initialize cache warmup scheduler
	pc.warmupScheduler = &CacheWarmupScheduler{
		cache:  pc,
		logger: logger.Named("cache_warmup"),
	}
	
	// Start performance monitoring
	go pc.performanceMonitoringLoop()
	
	// Start hot cache maintenance
	go pc.hotCacheMaintenance()
	
	logger.Info("Performance cache initialized",
		zap.Duration("hot_cache_ttl", pc.hotCacheTTL),
		zap.Int64("hot_max_size", pc.hotMaxSize),
		zap.Int("pipeline_size", pc.pipelineSize),
	)
	
	return pc
}

// FastGet provides ultra-fast cache retrieval with hot cache optimization
func (pc *PerformanceCache) FastGet(ctx context.Context, key string, dest interface{}) error {
	start := time.Now()
	
	// Try hot cache first (target <1ms)
	if entry, ok := pc.getFromHotCache(key); ok {
		pc.updatePerfMetrics("hot_cache_hit", time.Since(start))
		return pc.deserializeValue(entry.Value, dest)
	}
	
	// Fall back to multi-level cache
	err := pc.cache.Get(ctx, key, dest)
	latency := time.Since(start)
	
	if err == nil {
		// Promote to hot cache if frequently accessed
		pc.promoteToHotCache(key, dest)
		pc.updatePerfMetrics("cache_hit", latency)
	} else {
		pc.updatePerfMetrics("cache_miss", latency)
	}
	
	return err
}

// FastSet provides optimized cache storage with write-through option
func (pc *PerformanceCache) FastSet(ctx context.Context, key string, value interface{}, ttl time.Duration, tags ...string) error {
	start := time.Now()
	
	// Set in multi-level cache
	if err := pc.cache.Set(ctx, key, value, ttl, tags...); err != nil {
		pc.updatePerfMetrics("cache_error", time.Since(start))
		return err
	}
	
	// Also set in hot cache if small enough
	if size := pc.estimateSize(value); size < 1024 { // 1KB threshold
		pc.setToHotCache(key, value, ttl)
	}
	
	pc.updatePerfMetrics("cache_set", time.Since(start))
	return nil
}

// BatchGet retrieves multiple keys efficiently using pipelining
func (pc *PerformanceCache) BatchGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	start := time.Now()
	results := make(map[string]interface{})
	
	// Check hot cache first
	hotHits := make(map[string]interface{})
	remainingKeys := make([]string, 0, len(keys))
	
	for _, key := range keys {
		if entry, ok := pc.getFromHotCache(key); ok {
			hotHits[key] = entry.Value
		} else {
			remainingKeys = append(remainingKeys, key)
		}
	}
	
	// Pipeline remaining keys from Redis
	if len(remainingKeys) > 0 {
		pipe := pc.redis.Pipeline()
		cmds := make(map[string]*redis.StringCmd)
		
		for _, key := range remainingKeys {
			cacheKey := fmt.Sprintf("med_cache:%s", key)
			cmds[key] = pipe.Get(ctx, cacheKey)
		}
		
		_, err := pipe.Exec(ctx)
		if err != nil && err != redis.Nil {
			pc.logger.Error("Batch pipeline failed", zap.Error(err))
			return hotHits, err
		}
		
		// Process pipeline results
		for key, cmd := range cmds {
			if result, err := cmd.Result(); err == nil {
				var entry CacheEntry
				if err := json.Unmarshal([]byte(result), &entry); err == nil {
					results[key] = entry.Value
					// Consider promoting to hot cache
					pc.promoteToHotCache(key, entry.Value)
				}
			}
		}
	}
	
	// Combine hot cache hits with pipeline results
	for k, v := range hotHits {
		results[k] = v
	}
	
	latency := time.Since(start)
	pc.logger.Debug("Batch get completed",
		zap.Int("total_keys", len(keys)),
		zap.Int("hot_hits", len(hotHits)),
		zap.Int("redis_hits", len(results)-len(hotHits)),
		zap.Duration("latency", latency),
	)
	
	pc.updatePerfMetrics("batch_get", latency)
	return results, nil
}

// BatchSet efficiently stores multiple key-value pairs using pipelining
func (pc *PerformanceCache) BatchSet(ctx context.Context, items map[string]interface{}, ttl time.Duration, tags ...string) error {
	start := time.Now()
	
	if len(items) == 0 {
		return nil
	}
	
	// Create pipeline for Redis operations
	pipe := pc.redis.Pipeline()
	
	for key, value := range items {
		entry := &CacheEntry{
			Key:        key,
			Value:      value,
			Level:      L2Cache,
			TTL:        ttl,
			CreatedAt:  time.Now(),
			AccessedAt: time.Now(),
			AccessCount: 1,
			Size:       pc.estimateSize(value),
			Tags:       tags,
		}
		
		data, err := json.Marshal(entry)
		if err != nil {
			pc.logger.Error("Failed to marshal cache entry", zap.String("key", key), zap.Error(err))
			continue
		}
		
		cacheKey := fmt.Sprintf("med_cache:%s", key)
		pipe.Set(ctx, cacheKey, data, ttl)
		
		// Add to hot cache if small
		if entry.Size < 1024 {
			pc.setToHotCache(key, value, ttl)
		}
		
		// Add tag indices
		for _, tag := range tags {
			tagKey := fmt.Sprintf("tag:%s:%s", tag, key)
			pipe.Set(ctx, tagKey, "1", ttl)
		}
	}
	
	// Execute pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		pc.updatePerfMetrics("batch_error", time.Since(start))
		return fmt.Errorf("batch set pipeline failed: %w", err)
	}
	
	latency := time.Since(start)
	pc.logger.Debug("Batch set completed",
		zap.Int("items", len(items)),
		zap.Duration("latency", latency),
	)
	
	pc.updatePerfMetrics("batch_set", latency)
	return nil
}

// SetupCacheWarming configures proactive cache warming rules
func (pc *PerformanceCache) SetupCacheWarming(rules []WarmupRule) {
	pc.warmupScheduler.mutex.Lock()
	defer pc.warmupScheduler.mutex.Unlock()
	
	pc.warmupScheduler.warmupRules = rules
	
	// Start warmup scheduler if not already running
	if pc.warmupScheduler.scheduler == nil {
		pc.warmupScheduler.scheduler = time.NewTicker(1 * time.Minute)
		go pc.warmupScheduler.run()
	}
	
	pc.logger.Info("Cache warming configured", zap.Int("rules", len(rules)))
}

// GetPerformanceMetrics returns current performance statistics
func (pc *PerformanceCache) GetPerformanceMetrics() PerformanceMetrics {
	pc.perfMetrics.mutex.RLock()
	defer pc.perfMetrics.mutex.RUnlock()
	return *pc.perfMetrics
}

// OptimizeForLatency configures cache for minimal latency
func (pc *PerformanceCache) OptimizeForLatency() {
	pc.hotCacheTTL = 10 * time.Minute
	pc.hotMaxSize = 2000
	pc.pipelineSize = 50
	
	pc.logger.Info("Cache optimized for latency",
		zap.Duration("hot_ttl", pc.hotCacheTTL),
		zap.Int64("hot_max_size", pc.hotMaxSize),
	)
}

// OptimizeForThroughput configures cache for maximum throughput
func (pc *PerformanceCache) OptimizeForThroughput() {
	pc.hotCacheTTL = 2 * time.Minute
	pc.hotMaxSize = 500
	pc.pipelineSize = 200
	
	pc.logger.Info("Cache optimized for throughput",
		zap.Duration("hot_ttl", pc.hotCacheTTL),
		zap.Int("pipeline_size", pc.pipelineSize),
	)
}

// Internal helper methods

func (pc *PerformanceCache) getFromHotCache(key string) (*HotCacheEntry, bool) {
	if value, ok := pc.hotCache.Load(key); ok {
		if entry, ok := value.(*HotCacheEntry); ok {
			if time.Now().Before(entry.ExpiresAt) {
				entry.HitCount++
				return entry, true
			}
			// Expired entry
			pc.hotCache.Delete(key)
		}
	}
	return nil, false
}

func (pc *PerformanceCache) setToHotCache(key string, value interface{}, ttl time.Duration) {
	// Check if we're at capacity
	if pc.hotCount >= pc.hotMaxSize {
		pc.evictFromHotCache()
	}
	
	entry := &HotCacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(pc.hotCacheTTL),
		HitCount:  1,
		Size:      pc.estimateSize(value),
	}
	
	pc.hotCache.Store(key, entry)
	pc.hotCount++
}

func (pc *PerformanceCache) promoteToHotCache(key string, value interface{}) {
	// Only promote if there's space or if this is a frequently accessed item
	if pc.hotCount < pc.hotMaxSize {
		pc.setToHotCache(key, value, pc.hotCacheTTL)
	}
}

func (pc *PerformanceCache) evictFromHotCache() {
	// Simple LRU eviction - remove oldest entries
	var toEvict []string
	oldestTime := time.Now()
	
	pc.hotCache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*HotCacheEntry); ok {
			if entry.ExpiresAt.Before(oldestTime) {
				oldestTime = entry.ExpiresAt
				toEvict = []string{key.(string)}
			}
		}
		return true
	})
	
	for _, key := range toEvict {
		pc.hotCache.Delete(key)
		pc.hotCount--
	}
}

func (pc *PerformanceCache) estimateSize(value interface{}) int64 {
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return int64(len(data))
}

func (pc *PerformanceCache) deserializeValue(value interface{}, dest interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal to destination: %w", err)
	}
	
	return nil
}

func (pc *PerformanceCache) updatePerfMetrics(operation string, latency time.Duration) {
	pc.perfMetrics.mutex.Lock()
	defer pc.perfMetrics.mutex.Unlock()
	
	now := time.Now()
	
	// Update request per second counter
	if now.Sub(pc.perfMetrics.LastSecondTime) >= time.Second {
		pc.perfMetrics.RequestsPerSecond = float64(pc.perfMetrics.LastSecondCount)
		pc.perfMetrics.LastSecondCount = 0
		pc.perfMetrics.LastSecondTime = now
	}
	pc.perfMetrics.LastSecondCount++
	
	// Update latency metrics
	switch operation {
	case "hot_cache_hit":
		pc.perfMetrics.HotCacheLatency = pc.updateAverage(pc.perfMetrics.HotCacheLatency, latency)
	case "cache_hit", "cache_set":
		pc.perfMetrics.L2AvgLatency = pc.updateAverage(pc.perfMetrics.L2AvgLatency, latency)
	}
}

func (pc *PerformanceCache) updateAverage(current, new time.Duration) time.Duration {
	// Simple exponential moving average
	alpha := 0.1
	return time.Duration(float64(current)*(1-alpha) + float64(new)*alpha)
}

func (pc *PerformanceCache) performanceMonitoringLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		metrics := pc.GetPerformanceMetrics()
		
		pc.logger.Debug("Cache performance metrics",
			zap.Float64("requests_per_second", metrics.RequestsPerSecond),
			zap.Duration("hot_cache_latency", metrics.HotCacheLatency),
			zap.Duration("l2_avg_latency", metrics.L2AvgLatency),
			zap.Int64("hot_cache_count", pc.hotCount),
		)
		
		// Auto-optimize based on performance
		if metrics.RequestsPerSecond > 1000 && metrics.L2AvgLatency > 50*time.Millisecond {
			pc.OptimizeForLatency()
		}
	}
}

func (pc *PerformanceCache) hotCacheMaintenance() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		// Remove expired entries
		var expired []string
		now := time.Now()
		
		pc.hotCache.Range(func(key, value interface{}) bool {
			if entry, ok := value.(*HotCacheEntry); ok {
				if now.After(entry.ExpiresAt) {
					expired = append(expired, key.(string))
				}
			}
			return true
		})
		
		for _, key := range expired {
			pc.hotCache.Delete(key)
			pc.hotCount--
		}
		
		if len(expired) > 0 {
			pc.logger.Debug("Hot cache maintenance", zap.Int("expired_entries", len(expired)))
		}
	}
}

// Cache warmup scheduler methods

func (cws *CacheWarmupScheduler) run() {
	for range cws.scheduler.C {
		cws.executeWarmupRules()
	}
}

func (cws *CacheWarmupScheduler) executeWarmupRules() {
	cws.mutex.RLock()
	rules := make([]WarmupRule, len(cws.warmupRules))
	copy(rules, cws.warmupRules)
	cws.mutex.RUnlock()
	
	for _, rule := range rules {
		go cws.executeRule(rule)
	}
}

func (cws *CacheWarmupScheduler) executeRule(rule WarmupRule) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	start := time.Now()
	
	// Get data from provider
	data, err := rule.DataProvider(ctx)
	if err != nil {
		cws.logger.Error("Cache warmup rule failed",
			zap.String("rule", rule.Name),
			zap.Error(err),
		)
		return
	}
	
	// Store in cache
	if err := cws.cache.BatchSet(ctx, data, rule.TTL, rule.Tags...); err != nil {
		cws.logger.Error("Cache warmup batch set failed",
			zap.String("rule", rule.Name),
			zap.Error(err),
		)
		return
	}
	
	cws.logger.Debug("Cache warmup rule executed",
		zap.String("rule", rule.Name),
		zap.Int("items", len(data)),
		zap.Duration("duration", time.Since(start)),
	)
}