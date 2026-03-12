// Package services provides the TerminologyBridge - a high-performance multi-layer caching
// system for clinical terminology validation.
//
// Architecture: L0 (Bloom) → L1 (Hot Sets) → L2 (Local Cache) → L2.5 (Redis) → L3 (Neo4j)
//
// This bridge is designed for healthcare systems requiring sub-millisecond validation
// of clinical codes against value sets, with support for SNOMED CT subsumption.
package services

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/semantic"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// CONFIGURATION
// ============================================================================

// TerminologyBridgeConfig holds configuration for the multi-layer caching bridge
type TerminologyBridgeConfig struct {
	// L1: Hot Sets Configuration - Pre-loaded value sets for fastest access
	HotValueSets []string // e.g., ["SepsisDiagnosis", "AcuteRenalFailure", "DiabetesMellitus"]

	// L2: Local Cache Configuration
	LocalCacheTTL     time.Duration // e.g., 1 hour
	LocalCacheMaxSize int           // e.g., 100,000 entries

	// L2.5: Redis Configuration (for distributed deployments)
	RedisEnabled bool
	RedisTTL     time.Duration // e.g., 24 hours

	// Bloom Filter Configuration
	BloomFilterSize      uint    // Expected number of elements, e.g., 1,000,000
	BloomFilterFalseRate float64 // Acceptable false positive rate, e.g., 0.01 (1%)

	// Neo4j Configuration (inherited from Neo4jBridgeConfig)
	Neo4jConfig *Neo4jBridgeConfig

	// Subsumption Configuration
	EnableSubsumption bool // Enable Step 3 (subsumption) in THREE-CHECK PIPELINE
	SubsumptionMaxDepth int // Maximum hierarchy depth for subsumption, e.g., 10
}

// DefaultTerminologyBridgeConfig returns production-ready defaults
func DefaultTerminologyBridgeConfig() *TerminologyBridgeConfig {
	return &TerminologyBridgeConfig{
		HotValueSets: []string{
			"SepsisDiagnosis",
			"AcuteRenalFailure",
			"AUAKIConditions",
			"AUSepsisConditions",
			"DiabetesMellitus",
			"Hypertension",
		},
		LocalCacheTTL:        1 * time.Hour,
		LocalCacheMaxSize:    100000,
		RedisEnabled:         true,
		RedisTTL:             24 * time.Hour,
		BloomFilterSize:      1000000,
		BloomFilterFalseRate: 0.01,
		Neo4jConfig:          DefaultNeo4jBridgeConfig(),
		EnableSubsumption:    true,
		SubsumptionMaxDepth:  10,
	}
}

// ============================================================================
// CACHE ENTRY WITH TTL
// ============================================================================

type localCacheEntry struct {
	value     bool
	expiresAt time.Time
}

func (e *localCacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// ============================================================================
// METRICS (Observability)
// ============================================================================

// TerminologyBridgeMetrics tracks multi-layer cache performance
type TerminologyBridgeMetrics struct {
	// Layer hit counters (atomic for thread safety)
	L0BloomRejects int64 // Bloom filter definite rejections
	L1HotSetHits   int64 // Hot set exact matches
	L2LocalHits    int64 // Local cache hits
	L25RedisHits   int64 // Redis cache hits
	L3Neo4jHits    int64 // Neo4j queries (cache misses)
	TotalQueries   int64 // Total validation requests

	// Timing metrics
	AvgL1LatencyNs  int64 // Average L1 latency in nanoseconds
	AvgL3LatencyNs  int64 // Average L3 (Neo4j) latency in nanoseconds

	// Error counters
	Neo4jErrors  int64
	RedisErrors  int64
}

// GetHitRates returns cache hit rate percentages by layer
func (m *TerminologyBridgeMetrics) GetHitRates() map[string]float64 {
	total := float64(atomic.LoadInt64(&m.TotalQueries))
	if total == 0 {
		return map[string]float64{}
	}
	return map[string]float64{
		"l0_bloom_reject_rate": float64(atomic.LoadInt64(&m.L0BloomRejects)) / total * 100,
		"l1_hot_set_hit_rate":  float64(atomic.LoadInt64(&m.L1HotSetHits)) / total * 100,
		"l2_local_hit_rate":    float64(atomic.LoadInt64(&m.L2LocalHits)) / total * 100,
		"l25_redis_hit_rate":   float64(atomic.LoadInt64(&m.L25RedisHits)) / total * 100,
		"l3_neo4j_hit_rate":    float64(atomic.LoadInt64(&m.L3Neo4jHits)) / total * 100,
		"cache_hit_rate":       float64(atomic.LoadInt64(&m.L1HotSetHits)+atomic.LoadInt64(&m.L2LocalHits)+atomic.LoadInt64(&m.L25RedisHits)) / total * 100,
	}
}

// ============================================================================
// MEMBERSHIP RESULT
// ============================================================================

// MembershipResult represents the result of a code membership check
type MembershipResult struct {
	Valid        bool    `json:"valid"`                    // Is the code valid for the value set?
	MatchType    string  `json:"match_type"`               // "exact", "subsumption", "none"
	MatchedCode  string  `json:"matched_code,omitempty"`   // For subsumption: the ancestor that matched
	Source       string  `json:"source"`                   // Which layer answered: "L0_bloom", "L1_hot", "L2_local", "L25_redis", "L3_neo4j"
	PathLength   int     `json:"path_length,omitempty"`    // For subsumption: hierarchy distance
	DurationMs   float64 `json:"duration_ms"`              // How long the lookup took
	FromCache    bool    `json:"from_cache"`               // Was result from cache?
}

// ============================================================================
// THE TERMINOLOGY BRIDGE SERVICE
// ============================================================================

// TerminologyBridge provides high-performance clinical terminology validation
// with a multi-layer caching architecture.
//
// Layer 0: Bloom Filter - Ultra-fast negative lookups (~0.001ms)
// Layer 1: Hot Sets - Pre-loaded value sets in memory (~0.01ms)
// Layer 2: Local Cache - TTL-based local cache (~0.1ms)
// Layer 2.5: Redis Cache - Distributed cache (~1ms)
// Layer 3: Neo4j - THREE-CHECK PIPELINE fallback (~5-10ms)
type TerminologyBridge struct {
	config  *TerminologyBridgeConfig
	neo4j   *semantic.Neo4jClient
	redis   *cache.RedisClient
	logger  *logrus.Logger
	metrics *TerminologyBridgeMetrics

	// Rule Manager reference for value set expansion
	ruleManager RuleManager

	// L0: Bloom Filters (Ultra-fast negative lookups)
	// Key: ValueSetID, Value: BloomFilter containing all codes
	bloomFilters   map[string]*bloom.BloomFilter
	bloomFiltersMu sync.RWMutex

	// L1: Hot Sets (Pre-loaded in memory)
	// Key: ValueSetID, Value: map of codes for O(1) lookup
	hotSets   map[string]map[string]struct{}
	hotSetsMu sync.RWMutex

	// L2: Local Cache with TTL
	// Key: hash of (valueSetID:code), Value: cached result
	localCache   map[string]*localCacheEntry
	localCacheMu sync.RWMutex

	// Health status
	neo4jHealthy bool
	healthMu     sync.RWMutex

	// Lifecycle
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewTerminologyBridge creates a new high-performance terminology bridge
func NewTerminologyBridge(
	ctx context.Context,
	config *TerminologyBridgeConfig,
	neo4jClient *semantic.Neo4jClient,
	redisClient *cache.RedisClient,
	ruleManager RuleManager,
	logger *logrus.Logger,
) (*TerminologyBridge, error) {
	if config == nil {
		config = DefaultTerminologyBridgeConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	bridgeCtx, cancel := context.WithCancel(ctx)

	bridge := &TerminologyBridge{
		config:       config,
		neo4j:        neo4jClient,
		redis:        redisClient,
		ruleManager:  ruleManager,
		logger:       logger,
		metrics:      &TerminologyBridgeMetrics{},
		bloomFilters: make(map[string]*bloom.BloomFilter),
		hotSets:      make(map[string]map[string]struct{}),
		localCache:   make(map[string]*localCacheEntry),
		neo4jHealthy: neo4jClient != nil,
		ctx:          bridgeCtx,
		cancelFunc:   cancel,
	}

	// Pre-load hot value sets at startup (async)
	go bridge.initializeHotSets()

	// Start background workers
	go bridge.cacheCleanupWorker()
	go bridge.healthCheckWorker()

	logger.WithFields(logrus.Fields{
		"hot_value_sets":     len(config.HotValueSets),
		"bloom_filter_size":  config.BloomFilterSize,
		"local_cache_max":    config.LocalCacheMaxSize,
		"redis_enabled":      config.RedisEnabled,
		"subsumption_enabled": config.EnableSubsumption,
	}).Info("TerminologyBridge initialized with multi-layer caching")

	return bridge, nil
}

// ============================================================================
// MAIN API: CHECK MEMBERSHIP (Multi-Layer Pipeline)
// ============================================================================

// CheckMembership validates if a code belongs to a value set using the multi-layer cache.
// Returns immediately if any layer can answer definitively.
//
// Pipeline:
// L0 (Bloom) → L1 (Hot Set) → L2 (Local) → L2.5 (Redis) → L3 (Neo4j THREE-CHECK)
func (b *TerminologyBridge) CheckMembership(
	ctx context.Context,
	valueSetID string,
	candidateCode string,
	system string,
) (*MembershipResult, error) {
	start := time.Now()
	atomic.AddInt64(&b.metrics.TotalQueries, 1)

	// Default system to SNOMED CT
	if system == "" {
		system = "http://snomed.info/sct"
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LAYER 0: BLOOM FILTER (Ultra-fast negative lookup ~0.001ms)
	// If bloom filter says "definitely not in set", skip everything
	// ─────────────────────────────────────────────────────────────────────────
	if b.bloomFilterRejects(valueSetID, candidateCode) {
		atomic.AddInt64(&b.metrics.L0BloomRejects, 1)
		return &MembershipResult{
			Valid:      false,
			MatchType:  "none",
			Source:     "L0_bloom_filter",
			DurationMs: float64(time.Since(start).Microseconds()) / 1000,
			FromCache:  true,
		}, nil
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LAYER 1: HOT SET (Pre-loaded in memory ~0.01ms)
	// ─────────────────────────────────────────────────────────────────────────
	if result, found := b.checkHotSet(valueSetID, candidateCode); found {
		atomic.AddInt64(&b.metrics.L1HotSetHits, 1)
		return &MembershipResult{
			Valid:       result,
			MatchType:   "exact",
			MatchedCode: candidateCode,
			Source:      "L1_hot_set",
			DurationMs:  float64(time.Since(start).Microseconds()) / 1000,
			FromCache:   true,
		}, nil
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LAYER 2: LOCAL CACHE (Recent lookups with TTL ~0.1ms)
	// ─────────────────────────────────────────────────────────────────────────
	cacheKey := b.makeCacheKey(valueSetID, candidateCode, system)
	if entry, found := b.checkLocalCache(cacheKey); found {
		atomic.AddInt64(&b.metrics.L2LocalHits, 1)
		return &MembershipResult{
			Valid:       entry.value,
			MatchType:   "cached",
			MatchedCode: candidateCode,
			Source:      "L2_local_cache",
			DurationMs:  float64(time.Since(start).Microseconds()) / 1000,
			FromCache:   true,
		}, nil
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LAYER 2.5: REDIS CACHE (Distributed cache ~1ms)
	// ─────────────────────────────────────────────────────────────────────────
	if b.config.RedisEnabled && b.redis != nil {
		if result, found := b.checkRedisCache(ctx, cacheKey); found {
			atomic.AddInt64(&b.metrics.L25RedisHits, 1)
			// Also populate local cache for future requests
			b.setLocalCache(cacheKey, result)
			return &MembershipResult{
				Valid:       result,
				MatchType:   "cached",
				MatchedCode: candidateCode,
				Source:      "L25_redis_cache",
				DurationMs:  float64(time.Since(start).Microseconds()) / 1000,
				FromCache:   true,
			}, nil
		}
	}

	// ─────────────────────────────────────────────────────────────────────────
	// LAYER 3: NEO4J (THREE-CHECK PIPELINE ~5-10ms)
	// This is the fallback for cache misses - uses full Rule Manager validation
	// ─────────────────────────────────────────────────────────────────────────
	atomic.AddInt64(&b.metrics.L3Neo4jHits, 1)
	result, err := b.executeThreeCheckPipeline(ctx, valueSetID, candidateCode, system)
	if err != nil {
		atomic.AddInt64(&b.metrics.Neo4jErrors, 1)
		return nil, err
	}

	// Cache the result for future queries
	b.setLocalCache(cacheKey, result.Valid)
	if b.config.RedisEnabled && b.redis != nil {
		b.setRedisCache(ctx, cacheKey, result.Valid)
	}

	result.Source = "L3_neo4j"
	result.DurationMs = float64(time.Since(start).Microseconds()) / 1000
	return result, nil
}

// ============================================================================
// LAYER 0: BLOOM FILTER
// ============================================================================

func (b *TerminologyBridge) bloomFilterRejects(valueSetID, code string) bool {
	b.bloomFiltersMu.RLock()
	defer b.bloomFiltersMu.RUnlock()

	filter, exists := b.bloomFilters[valueSetID]
	if !exists {
		return false // No bloom filter = can't reject
	}

	// If bloom filter says NO, it's DEFINITELY not in the set
	// If bloom filter says YES, it MIGHT be (false positive possible, continue to next layer)
	return !filter.TestString(code)
}

// PreloadBloomFilter creates a bloom filter for a value set
func (b *TerminologyBridge) PreloadBloomFilter(ctx context.Context, valueSetID string, codes []string) error {
	// Create bloom filter with estimated size
	filterSize := uint(len(codes) * 2) // 2x for safety margin
	if filterSize < b.config.BloomFilterSize {
		filterSize = b.config.BloomFilterSize
	}

	filter := bloom.NewWithEstimates(filterSize, b.config.BloomFilterFalseRate)
	for _, code := range codes {
		filter.AddString(code)
	}

	b.bloomFiltersMu.Lock()
	b.bloomFilters[valueSetID] = filter
	b.bloomFiltersMu.Unlock()

	b.logger.WithFields(logrus.Fields{
		"value_set_id": valueSetID,
		"codes_count":  len(codes),
		"filter_size":  filterSize,
	}).Debug("Bloom filter preloaded for value set")

	return nil
}

// ============================================================================
// LAYER 1: HOT SETS
// ============================================================================

func (b *TerminologyBridge) checkHotSet(valueSetID, code string) (bool, bool) {
	b.hotSetsMu.RLock()
	defer b.hotSetsMu.RUnlock()

	set, exists := b.hotSets[valueSetID]
	if !exists {
		return false, false // Value set not pre-loaded
	}

	_, found := set[code]
	return found, true // Return (membership result, was_checked)
}

// PreloadHotSet loads a value set into the L1 hot set cache
func (b *TerminologyBridge) PreloadHotSet(ctx context.Context, valueSetID string) error {
	if b.ruleManager == nil {
		return fmt.Errorf("rule manager not available for value set expansion")
	}

	// Use Rule Manager to expand the value set
	expanded, err := b.ruleManager.ExpandValueSet(ctx, valueSetID, "")
	if err != nil {
		return fmt.Errorf("failed to expand value set %s: %w", valueSetID, err)
	}

	// Build the hot set
	hotSet := make(map[string]struct{}, len(expanded.Codes))
	codes := make([]string, 0, len(expanded.Codes))
	for _, code := range expanded.Codes {
		hotSet[code.Code] = struct{}{}
		codes = append(codes, code.Code)
	}

	// Atomic update
	b.hotSetsMu.Lock()
	b.hotSets[valueSetID] = hotSet
	b.hotSetsMu.Unlock()

	// Also build bloom filter for this value set
	if err := b.PreloadBloomFilter(ctx, valueSetID, codes); err != nil {
		b.logger.WithError(err).WithField("value_set_id", valueSetID).Warn("Failed to create bloom filter")
	}

	b.logger.WithFields(logrus.Fields{
		"value_set_id": valueSetID,
		"codes_count":  len(hotSet),
	}).Info("Hot set preloaded for value set")

	return nil
}

// initializeHotSets preloads all configured hot value sets at startup
func (b *TerminologyBridge) initializeHotSets() {
	// Wait a bit for rule manager to be fully initialized
	time.Sleep(2 * time.Second)

	for _, vsID := range b.config.HotValueSets {
		ctx, cancel := context.WithTimeout(b.ctx, 30*time.Second)
		if err := b.PreloadHotSet(ctx, vsID); err != nil {
			b.logger.WithError(err).WithField("value_set_id", vsID).Warn("Failed to preload hot set (may be seeded later)")
		}
		cancel()
	}

	b.logger.WithField("hot_sets_loaded", len(b.config.HotValueSets)).Info("Hot set initialization complete")
}

// ============================================================================
// LAYER 2: LOCAL CACHE
// ============================================================================

func (b *TerminologyBridge) makeCacheKey(valueSetID, code, system string) string {
	// Use FNV hash for consistent, compact key
	h := fnv.New64a()
	h.Write([]byte(valueSetID))
	h.Write([]byte(":"))
	h.Write([]byte(code))
	h.Write([]byte(":"))
	h.Write([]byte(system))
	return fmt.Sprintf("term:%x", h.Sum64())
}

func (b *TerminologyBridge) checkLocalCache(key string) (*localCacheEntry, bool) {
	b.localCacheMu.RLock()
	defer b.localCacheMu.RUnlock()

	entry, exists := b.localCache[key]
	if !exists || entry.isExpired() {
		return nil, false
	}
	return entry, true
}

func (b *TerminologyBridge) setLocalCache(key string, value bool) {
	b.localCacheMu.Lock()
	defer b.localCacheMu.Unlock()

	// Evict if cache is too large (simple LRU alternative)
	if len(b.localCache) >= b.config.LocalCacheMaxSize {
		// Simple eviction: remove expired entries first
		for k, v := range b.localCache {
			if v.isExpired() {
				delete(b.localCache, k)
			}
			if len(b.localCache) < b.config.LocalCacheMaxSize*9/10 {
				break
			}
		}
	}

	b.localCache[key] = &localCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(b.config.LocalCacheTTL),
	}
}

func (b *TerminologyBridge) cacheCleanupWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.cleanExpiredEntries()
		}
	}
}

func (b *TerminologyBridge) cleanExpiredEntries() {
	b.localCacheMu.Lock()
	defer b.localCacheMu.Unlock()

	cleaned := 0
	for key, entry := range b.localCache {
		if entry.isExpired() {
			delete(b.localCache, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		b.logger.WithField("entries_cleaned", cleaned).Debug("Cache cleanup complete")
	}
}

// ============================================================================
// LAYER 2.5: REDIS CACHE
// ============================================================================

func (b *TerminologyBridge) checkRedisCache(ctx context.Context, key string) (bool, bool) {
	if b.redis == nil {
		return false, false
	}

	var result bool
	if err := b.redis.Get(key, &result); err != nil {
		return false, false
	}
	return result, true
}

func (b *TerminologyBridge) setRedisCache(ctx context.Context, key string, value bool) {
	if b.redis == nil {
		return
	}

	if err := b.redis.Set(key, value, b.config.RedisTTL); err != nil {
		atomic.AddInt64(&b.metrics.RedisErrors, 1)
		b.logger.WithError(err).WithField("key", key).Debug("Failed to set Redis cache")
	}
}

// ============================================================================
// LAYER 3: NEO4J THREE-CHECK PIPELINE
// ============================================================================

func (b *TerminologyBridge) executeThreeCheckPipeline(
	ctx context.Context,
	valueSetID string,
	candidateCode string,
	system string,
) (*MembershipResult, error) {
	// Use the Rule Manager's existing THREE-CHECK PIPELINE
	if b.ruleManager == nil {
		return nil, fmt.Errorf("rule manager not available")
	}

	result, err := b.ruleManager.ValidateCodeInValueSet(ctx, candidateCode, system, valueSetID)
	if err != nil {
		return nil, err
	}

	return &MembershipResult{
		Valid:       result.Valid,
		MatchType:   string(result.MatchType),
		MatchedCode: result.MatchedCode,
		FromCache:   false,
	}, nil
}

// ============================================================================
// HEALTH & LIFECYCLE
// ============================================================================

func (b *TerminologyBridge) healthCheckWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.updateHealthStatus()
		}
	}
}

func (b *TerminologyBridge) updateHealthStatus() {
	healthy := b.neo4j != nil
	if healthy {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := b.neo4j.Health(ctx)
		cancel()
		healthy = (err == nil)
	}

	b.healthMu.Lock()
	b.neo4jHealthy = healthy
	b.healthMu.Unlock()
}

// IsNeo4jHealthy returns true if Neo4j is connected and healthy
func (b *TerminologyBridge) IsNeo4jHealthy() bool {
	b.healthMu.RLock()
	defer b.healthMu.RUnlock()
	return b.neo4jHealthy
}

// GetMetrics returns current bridge metrics
func (b *TerminologyBridge) GetMetrics() map[string]interface{} {
	b.hotSetsMu.RLock()
	hotSetCount := len(b.hotSets)
	var totalHotCodes int
	for _, set := range b.hotSets {
		totalHotCodes += len(set)
	}
	b.hotSetsMu.RUnlock()

	b.bloomFiltersMu.RLock()
	bloomFilterCount := len(b.bloomFilters)
	b.bloomFiltersMu.RUnlock()

	b.localCacheMu.RLock()
	localCacheSize := len(b.localCache)
	b.localCacheMu.RUnlock()

	return map[string]interface{}{
		"hot_sets_loaded":      hotSetCount,
		"hot_codes_total":      totalHotCodes,
		"bloom_filters_loaded": bloomFilterCount,
		"local_cache_size":     localCacheSize,
		"local_cache_max":      b.config.LocalCacheMaxSize,
		"hit_rates":            b.metrics.GetHitRates(),
		"total_queries":        atomic.LoadInt64(&b.metrics.TotalQueries),
		"l0_bloom_rejects":     atomic.LoadInt64(&b.metrics.L0BloomRejects),
		"l1_hot_hits":          atomic.LoadInt64(&b.metrics.L1HotSetHits),
		"l2_local_hits":        atomic.LoadInt64(&b.metrics.L2LocalHits),
		"l25_redis_hits":       atomic.LoadInt64(&b.metrics.L25RedisHits),
		"l3_neo4j_hits":        atomic.LoadInt64(&b.metrics.L3Neo4jHits),
		"neo4j_errors":         atomic.LoadInt64(&b.metrics.Neo4jErrors),
		"redis_errors":         atomic.LoadInt64(&b.metrics.RedisErrors),
		"neo4j_healthy":        b.IsNeo4jHealthy(),
		"redis_enabled":        b.config.RedisEnabled,
		"subsumption_enabled":  b.config.EnableSubsumption,
	}
}

// InvalidateValueSet removes a value set from all caches and reloads it
func (b *TerminologyBridge) InvalidateValueSet(ctx context.Context, valueSetID string) error {
	// Clear hot set
	b.hotSetsMu.Lock()
	delete(b.hotSets, valueSetID)
	b.hotSetsMu.Unlock()

	// Clear bloom filter
	b.bloomFiltersMu.Lock()
	delete(b.bloomFilters, valueSetID)
	b.bloomFiltersMu.Unlock()

	// Clear related local cache entries (this is approximate since we use hashed keys)
	// In production, you might want to track keys by value set for precise invalidation
	b.localCacheMu.Lock()
	for key := range b.localCache {
		delete(b.localCache, key)
	}
	b.localCacheMu.Unlock()

	// Reload the value set into hot cache
	return b.PreloadHotSet(ctx, valueSetID)
}

// RefreshHotSets reloads all configured hot value sets
func (b *TerminologyBridge) RefreshHotSets(ctx context.Context) error {
	for _, vsID := range b.config.HotValueSets {
		if err := b.PreloadHotSet(ctx, vsID); err != nil {
			b.logger.WithError(err).WithField("value_set_id", vsID).Warn("Failed to refresh hot set")
		}
	}
	return nil
}

// Close cleanly shuts down the bridge
func (b *TerminologyBridge) Close() error {
	b.cancelFunc()

	b.localCacheMu.Lock()
	b.localCache = make(map[string]*localCacheEntry)
	b.localCacheMu.Unlock()

	b.hotSetsMu.Lock()
	b.hotSets = make(map[string]map[string]struct{})
	b.hotSetsMu.Unlock()

	b.bloomFiltersMu.Lock()
	b.bloomFilters = make(map[string]*bloom.BloomFilter)
	b.bloomFiltersMu.Unlock()

	b.logger.Info("TerminologyBridge closed")
	return nil
}

// HealthCheck returns comprehensive health status
func (b *TerminologyBridge) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":             "healthy",
		"neo4j_healthy":      b.IsNeo4jHealthy(),
		"redis_available":    b.redis != nil,
		"hot_sets_loaded":    len(b.hotSets),
		"bloom_filters":      len(b.bloomFilters),
		"local_cache_size":   len(b.localCache),
		"metrics":            b.GetMetrics(),
	}
}
