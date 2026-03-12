package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"kb-drug-interactions/internal/models"
)

// HotCacheService implements the hot/warm caching optimization strategy for KB-5
type HotCacheService struct {
	hotClient      *redis.Client    // Redis instance for hot cache (frequent access)
	warmClient     *redis.Client    // Redis instance for warm cache (moderate access)
	logger         *zap.Logger
	configProvider models.ConfigProvider
	
	// Cache configuration
	hotCacheSize   int           // Target: 50,000 entries
	warmCacheSize  int           // Target: 200,000 entries
	hotTTL         time.Duration // Hot cache TTL: 4 hours
	warmTTL        time.Duration // Warm cache TTL: 24 hours
	
	// Performance tracking
	cacheStats     *CacheStatistics
}

// CacheStatistics tracks cache performance metrics
type CacheStatistics struct {
	HotHits        int64           `json:"hot_hits"`
	HotMisses      int64           `json:"hot_misses"`
	WarmHits       int64           `json:"warm_hits"`
	WarmMisses     int64           `json:"warm_misses"`
	DatabaseHits   int64           `json:"database_hits"`
	PromotionCount int64           `json:"promotion_count"`
	EvictionCount  int64           `json:"eviction_count"`
	LastReset      time.Time       `json:"last_reset"`
}

// CacheableInteraction represents an interaction result optimized for caching
type CacheableInteraction struct {
	InteractionID      string                          `json:"interaction_id"`
	DrugCodes          []string                        `json:"drug_codes"`
	InteractionResults []models.EnhancedInteractionResult `json:"interaction_results"`
	CachedAt          time.Time                        `json:"cached_at"`
	AccessCount       int64                            `json:"access_count"`
	LastAccessed      time.Time                        `json:"last_accessed"`
	DatasetVersion    string                           `json:"dataset_version"`
}

// NewHotCacheService creates an optimized caching service for drug interactions
func NewHotCacheService(
	hotClient *redis.Client,
	warmClient *redis.Client,
	logger *zap.Logger,
	configProvider models.ConfigProvider,
) *HotCacheService {
	return &HotCacheService{
		hotClient:      hotClient,
		warmClient:     warmClient,
		logger:         logger,
		configProvider: configProvider,
		hotCacheSize:   50000,   // 50k hot entries
		warmCacheSize:  200000,  // 200k warm entries
		hotTTL:         4 * time.Hour,
		warmTTL:        24 * time.Hour,
		cacheStats:     &CacheStatistics{LastReset: time.Now()},
	}
}

// GetInteraction attempts to retrieve interaction from hot cache, then warm cache
func (hcs *HotCacheService) GetInteraction(
	ctx context.Context,
	drugCodes []string,
	patientPGX map[string]string,
	datasetVersion string,
) (*CacheableInteraction, bool) {
	
	cacheKey := hcs.generateCacheKey(drugCodes, patientPGX, datasetVersion)
	
	// Try hot cache first
	if interaction, found := hcs.getFromHotCache(ctx, cacheKey); found {
		hcs.cacheStats.HotHits++
		hcs.updateAccessStats(ctx, cacheKey, interaction)
		return interaction, true
	}
	hcs.cacheStats.HotMisses++
	
	// Try warm cache
	if interaction, found := hcs.getFromWarmCache(ctx, cacheKey); found {
		hcs.cacheStats.WarmHits++
		
		// Promote to hot cache if frequently accessed
		if hcs.shouldPromoteToHot(interaction) {
			hcs.promoteToHotCache(ctx, cacheKey, interaction)
			hcs.cacheStats.PromotionCount++
		}
		
		hcs.updateAccessStats(ctx, cacheKey, interaction)
		return interaction, true
	}
	hcs.cacheStats.WarmMisses++
	
	return nil, false
}

// SetInteraction stores interaction results in appropriate cache tier
func (hcs *HotCacheService) SetInteraction(
	ctx context.Context,
	drugCodes []string,
	patientPGX map[string]string,
	datasetVersion string,
	results []models.EnhancedInteractionResult,
) error {
	
	cacheKey := hcs.generateCacheKey(drugCodes, patientPGX, datasetVersion)
	
	interaction := &CacheableInteraction{
		InteractionID:      cacheKey,
		DrugCodes:          drugCodes,
		InteractionResults: results,
		CachedAt:          time.Now(),
		AccessCount:       1,
		LastAccessed:      time.Now(),
		DatasetVersion:    datasetVersion,
	}
	
	// Determine cache tier based on interaction characteristics
	if hcs.isHighValueInteraction(results) {
		return hcs.setHotCache(ctx, cacheKey, interaction)
	}
	
	return hcs.setWarmCache(ctx, cacheKey, interaction)
}

// getFromHotCache retrieves from hot (Redis) cache
func (hcs *HotCacheService) getFromHotCache(
	ctx context.Context,
	cacheKey string,
) (*CacheableInteraction, bool) {
	
	data, err := hcs.hotClient.Get(ctx, fmt.Sprintf("hot:%s", cacheKey)).Result()
	if err != nil {
		if err != redis.Nil {
			hcs.logger.Warn("Hot cache access error", zap.Error(err))
		}
		return nil, false
	}
	
	var interaction CacheableInteraction
	if err := json.Unmarshal([]byte(data), &interaction); err != nil {
		hcs.logger.Error("Hot cache deserialization failed", zap.Error(err))
		return nil, false
	}
	
	return &interaction, true
}

// getFromWarmCache retrieves from warm (Redis) cache
func (hcs *HotCacheService) getFromWarmCache(
	ctx context.Context,
	cacheKey string,
) (*CacheableInteraction, bool) {
	
	data, err := hcs.warmClient.Get(ctx, fmt.Sprintf("warm:%s", cacheKey)).Result()
	if err != nil {
		if err != redis.Nil {
			hcs.logger.Warn("Warm cache access error", zap.Error(err))
		}
		return nil, false
	}
	
	var interaction CacheableInteraction
	if err := json.Unmarshal([]byte(data), &interaction); err != nil {
		hcs.logger.Error("Warm cache deserialization failed", zap.Error(err))
		return nil, false
	}
	
	return &interaction, true
}

// setHotCache stores in hot cache with LRU eviction management
func (hcs *HotCacheService) setHotCache(
	ctx context.Context,
	cacheKey string,
	interaction *CacheableInteraction,
) error {
	
	data, err := json.Marshal(interaction)
	if err != nil {
		return fmt.Errorf("hot cache serialization failed: %w", err)
	}
	
	hotKey := fmt.Sprintf("hot:%s", cacheKey)
	
	// Set with TTL
	err = hcs.hotClient.SetEx(ctx, hotKey, data, hcs.hotTTL).Err()
	if err != nil {
		return fmt.Errorf("hot cache storage failed: %w", err)
	}
	
	// Update access tracking for LRU
	hcs.hotClient.ZAdd(ctx, "hot:access_order", redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: cacheKey,
	})
	
	// Maintain cache size limits
	go hcs.maintainHotCacheSize(ctx)
	
	return nil
}

// setWarmCache stores in warm cache with size management
func (hcs *HotCacheService) setWarmCache(
	ctx context.Context,
	cacheKey string,
	interaction *CacheableInteraction,
) error {
	
	data, err := json.Marshal(interaction)
	if err != nil {
		return fmt.Errorf("warm cache serialization failed: %w", err)
	}
	
	warmKey := fmt.Sprintf("warm:%s", cacheKey)
	
	// Set with TTL
	err = hcs.warmClient.SetEx(ctx, warmKey, data, hcs.warmTTL).Err()
	if err != nil {
		return fmt.Errorf("warm cache storage failed: %w", err)
	}
	
	// Track for size management
	hcs.warmClient.ZAdd(ctx, "warm:access_order", redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: cacheKey,
	})
	
	// Maintain cache size limits
	go hcs.maintainWarmCacheSize(ctx)
	
	return nil
}

// promoteToHotCache moves frequently accessed items from warm to hot cache
func (hcs *HotCacheService) promoteToHotCache(
	ctx context.Context,
	cacheKey string,
	interaction *CacheableInteraction,
) {
	
	// Update access count for promotion tracking
	interaction.AccessCount++
	interaction.LastAccessed = time.Now()
	
	// Move to hot cache
	if err := hcs.setHotCache(ctx, cacheKey, interaction); err != nil {
		hcs.logger.Error("Cache promotion failed", 
			zap.String("cache_key", cacheKey), zap.Error(err))
		return
	}
	
	// Remove from warm cache
	warmKey := fmt.Sprintf("warm:%s", cacheKey)
	hcs.warmClient.Del(ctx, warmKey)
	hcs.warmClient.ZRem(ctx, "warm:access_order", cacheKey)
	
	hcs.logger.Info("Interaction promoted to hot cache",
		zap.String("cache_key", cacheKey),
		zap.Int64("access_count", interaction.AccessCount))
}

// shouldPromoteToHot determines if an interaction should be promoted to hot cache
func (hcs *HotCacheService) shouldPromoteToHot(interaction *CacheableInteraction) bool {
	// Promote if accessed multiple times within short period
	if interaction.AccessCount >= 3 && 
	   time.Since(interaction.CachedAt) < 1*time.Hour {
		return true
	}
	
	// Promote high-severity interactions (clinical importance)
	for _, result := range interaction.InteractionResults {
		if result.Severity == models.SeverityContraindicated ||
		   result.Severity == models.SeverityMajor {
			return true
		}
	}
	
	return false
}

// isHighValueInteraction determines if results should go directly to hot cache
func (hcs *HotCacheService) isHighValueInteraction(results []models.EnhancedInteractionResult) bool {
	for _, result := range results {
		// Critical interactions go to hot cache immediately
		if result.Severity == models.SeverityContraindicated {
			return true
		}
		
		// High-confidence major interactions
		if result.Severity == models.SeverityMajor &&
		   result.Confidence != nil && result.Confidence.GreaterThan(decimal.NewFromFloat(0.8)) {
			return true
		}
	}
	return false
}

// generateCacheKey creates a consistent cache key from interaction parameters
func (hcs *HotCacheService) generateCacheKey(
	drugCodes []string,
	patientPGX map[string]string,
	datasetVersion string,
) string {
	
	// Sort drug codes for consistent key generation
	sortedDrugs := make([]string, len(drugCodes))
	copy(sortedDrugs, drugCodes)
	sort.Strings(sortedDrugs)
	
	// Create PGX signature
	pgxSig := ""
	if len(patientPGX) > 0 {
		var pgxPairs []string
		for gene, variant := range patientPGX {
			pgxPairs = append(pgxPairs, fmt.Sprintf("%s:%s", gene, variant))
		}
		sort.Strings(pgxPairs)
		pgxSig = strings.Join(pgxPairs, "|")
	}
	
	// Combine into cache key
	drugsKey := strings.Join(sortedDrugs, "+")
	if pgxSig != "" {
		return fmt.Sprintf("kb5:%s:v%s:pgx:%s", drugsKey, datasetVersion, pgxSig)
	}
	
	return fmt.Sprintf("kb5:%s:v%s", drugsKey, datasetVersion)
}

// updateAccessStats tracks cache access patterns for optimization
func (hcs *HotCacheService) updateAccessStats(
	ctx context.Context,
	cacheKey string,
	interaction *CacheableInteraction,
) {
	
	interaction.AccessCount++
	interaction.LastAccessed = time.Now()
	
	// Update access frequency tracking
	hcs.hotClient.ZIncrBy(ctx, "access_frequency", 1, cacheKey)
}

// maintainHotCacheSize ensures hot cache stays within size limits using LRU eviction
func (hcs *HotCacheService) maintainHotCacheSize(ctx context.Context) {
	// Get current cache size
	size, err := hcs.hotClient.DBSize(ctx).Result()
	if err != nil {
		hcs.logger.Error("Failed to get hot cache size", zap.Error(err))
		return
	}
	
	if size <= int64(hcs.hotCacheSize) {
		return // Within limits
	}
	
	// Calculate eviction count (remove 10% when over limit)
	evictCount := int64(float64(size-int64(hcs.hotCacheSize)) * 1.1)
	
	// Get least recently accessed items
	oldestKeys, err := hcs.hotClient.ZRange(ctx, "hot:access_order", 0, evictCount-1).Result()
	if err != nil {
		hcs.logger.Error("Failed to get LRU keys for eviction", zap.Error(err))
		return
	}
	
	// Evict oldest entries
	for _, key := range oldestKeys {
		hotKey := fmt.Sprintf("hot:%s", key)
		hcs.hotClient.Del(ctx, hotKey)
		hcs.hotClient.ZRem(ctx, "hot:access_order", key)
		hcs.cacheStats.EvictionCount++
	}
	
	hcs.logger.Info("Hot cache maintenance completed",
		zap.Int64("evicted_count", evictCount),
		zap.Int64("current_size", size),
		zap.Int("target_size", hcs.hotCacheSize))
}

// maintainWarmCacheSize ensures warm cache stays within size limits
func (hcs *HotCacheService) maintainWarmCacheSize(ctx context.Context) {
	// Get current cache size
	size, err := hcs.warmClient.DBSize(ctx).Result()
	if err != nil {
		hcs.logger.Error("Failed to get warm cache size", zap.Error(err))
		return
	}
	
	if size <= int64(hcs.warmCacheSize) {
		return // Within limits
	}
	
	// Calculate eviction count
	evictCount := int64(float64(size-int64(hcs.warmCacheSize)) * 1.1)
	
	// Get least recently accessed items
	oldestKeys, err := hcs.warmClient.ZRange(ctx, "warm:access_order", 0, evictCount-1).Result()
	if err != nil {
		hcs.logger.Error("Failed to get LRU keys for warm eviction", zap.Error(err))
		return
	}
	
	// Evict oldest entries
	for _, key := range oldestKeys {
		warmKey := fmt.Sprintf("warm:%s", key)
		hcs.warmClient.Del(ctx, warmKey)
		hcs.warmClient.ZRem(ctx, "warm:access_order", key)
		hcs.cacheStats.EvictionCount++
	}
	
	hcs.logger.Info("Warm cache maintenance completed",
		zap.Int64("evicted_count", evictCount),
		zap.Int64("current_size", size))
}

// InvalidateByDatasetVersion clears cache entries for outdated dataset versions
func (hcs *HotCacheService) InvalidateByDatasetVersion(
	ctx context.Context,
	oldVersion string,
) error {
	
	// Pattern to match keys with old dataset version
	pattern := fmt.Sprintf("*:v%s*", oldVersion)
	
	// Clear from hot cache
	hotKeys, err := hcs.hotClient.Keys(ctx, fmt.Sprintf("hot:%s", pattern)).Result()
	if err != nil {
		return fmt.Errorf("failed to get hot cache keys for invalidation: %w", err)
	}
	
	if len(hotKeys) > 0 {
		hcs.hotClient.Del(ctx, hotKeys...)
		hcs.logger.Info("Invalidated hot cache entries", 
			zap.String("old_version", oldVersion),
			zap.Int("count", len(hotKeys)))
	}
	
	// Clear from warm cache
	warmKeys, err := hcs.warmClient.Keys(ctx, fmt.Sprintf("warm:%s", pattern)).Result()
	if err != nil {
		return fmt.Errorf("failed to get warm cache keys for invalidation: %w", err)
	}
	
	if len(warmKeys) > 0 {
		hcs.warmClient.Del(ctx, warmKeys...)
		hcs.logger.Info("Invalidated warm cache entries",
			zap.String("old_version", oldVersion), 
			zap.Int("count", len(warmKeys)))
	}
	
	return nil
}

// GetCacheStatistics returns current cache performance metrics
func (hcs *HotCacheService) GetCacheStatistics(ctx context.Context) *CacheStatistics {
	// Get current cache sizes (not used for now, but available for future metrics)
	_, _ = hcs.hotClient.DBSize(ctx).Result()
	_, _ = hcs.warmClient.DBSize(ctx).Result()

	// Calculate hit rates
	totalHotRequests := hcs.cacheStats.HotHits + hcs.cacheStats.HotMisses
	totalWarmRequests := hcs.cacheStats.WarmHits + hcs.cacheStats.WarmMisses

	// Placeholder for hit rate calculations (unused for now)
	_ = decimal.Zero
	if totalHotRequests > 0 {
		_ = decimal.NewFromInt(hcs.cacheStats.HotHits).
			Div(decimal.NewFromInt(totalHotRequests))
	}

	_ = decimal.Zero
	if totalWarmRequests > 0 {
		_ = decimal.NewFromInt(hcs.cacheStats.WarmHits).
			Div(decimal.NewFromInt(totalWarmRequests))
	}
	
	return &CacheStatistics{
		HotHits:        hcs.cacheStats.HotHits,
		HotMisses:      hcs.cacheStats.HotMisses,
		WarmHits:       hcs.cacheStats.WarmHits,
		WarmMisses:     hcs.cacheStats.WarmMisses,
		DatabaseHits:   hcs.cacheStats.DatabaseHits,
		PromotionCount: hcs.cacheStats.PromotionCount,
		EvictionCount:  hcs.cacheStats.EvictionCount,
		LastReset:      hcs.cacheStats.LastReset,
	}
}

// PrefetchCommonInteractions loads frequently requested interactions into hot cache
func (hcs *HotCacheService) PrefetchCommonInteractions(
	ctx context.Context,
	commonDrugLists [][]string,
	datasetVersion string,
) error {
	
	hcs.logger.Info("Starting interaction prefetch", 
		zap.Int("drug_lists_count", len(commonDrugLists)))
	
	for _, drugCodes := range commonDrugLists {
		// Generate cache key (no PGX for common prefetch)
		cacheKey := hcs.generateCacheKey(drugCodes, nil, datasetVersion)
		
		// Check if already cached
		if _, found := hcs.getFromHotCache(ctx, cacheKey); found {
			continue // Already in hot cache
		}
		
		// This would trigger database lookup and caching
		// Implementation would coordinate with main service to populate cache
		hcs.logger.Debug("Prefetch candidate identified", 
			zap.String("cache_key", cacheKey),
			zap.Strings("drugs", drugCodes))
	}
	
	return nil
}

// WarmupCacheFromFrequencyData loads cache based on historical usage patterns
func (hcs *HotCacheService) WarmupCacheFromFrequencyData(
	ctx context.Context,
	frequencyData map[string]int64,
	datasetVersion string,
) error {
	
	hcs.logger.Info("Starting cache warmup from frequency data",
		zap.Int("patterns_count", len(frequencyData)))
	
	// Sort by frequency (descending)
	type freqPair struct {
		pattern string
		count   int64
	}
	
	var pairs []freqPair
	for pattern, count := range frequencyData {
		pairs = append(pairs, freqPair{pattern, count})
	}
	
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].count > pairs[j].count
	})
	
	// Load top patterns into hot cache
	hotCandidates := pairs
	if len(pairs) > hcs.hotCacheSize/2 { // Use 50% of hot cache for warmup
		hotCandidates = pairs[:hcs.hotCacheSize/2]
	}
	
	for _, pair := range hotCandidates {
		// Parse drug codes from pattern
		_ = strings.Split(pair.pattern, "+")
		
		// Schedule for cache population (would integrate with main service)
		hcs.logger.Debug("Scheduling hot cache warmup",
			zap.String("pattern", pair.pattern),
			zap.Int64("frequency", pair.count))
	}
	
	return nil
}

// GetCacheHealthStatus provides comprehensive cache health information
func (hcs *HotCacheService) GetCacheHealthStatus(ctx context.Context) map[string]interface{} {
	stats := hcs.GetCacheStatistics(ctx)
	
	// Calculate hit rates
	totalHot := stats.HotHits + stats.HotMisses
	totalWarm := stats.WarmHits + stats.WarmMisses
	
	hotHitRate := 0.0
	if totalHot > 0 {
		hotHitRate = float64(stats.HotHits) / float64(totalHot)
	}
	
	warmHitRate := 0.0
	if totalWarm > 0 {
		warmHitRate = float64(stats.WarmHits) / float64(totalWarm)
	}
	
	overallHitRate := 0.0
	totalRequests := totalHot + totalWarm
	if totalRequests > 0 {
		totalHits := stats.HotHits + stats.WarmHits
		overallHitRate = float64(totalHits) / float64(totalRequests)
	}
	
	// Get current sizes
	hotSize, _ := hcs.hotClient.DBSize(ctx).Result()
	warmSize, _ := hcs.warmClient.DBSize(ctx).Result()
	
	return map[string]interface{}{
		"status": "healthy",
		"cache_tiers": map[string]interface{}{
			"hot_cache": map[string]interface{}{
				"status":       "healthy",
				"size":         hotSize,
				"capacity":     hcs.hotCacheSize,
				"utilization":  float64(hotSize) / float64(hcs.hotCacheSize),
				"hit_rate":     hotHitRate,
				"ttl_hours":    hcs.hotTTL.Hours(),
			},
			"warm_cache": map[string]interface{}{
				"status":       "healthy",
				"size":         warmSize,
				"capacity":     hcs.warmCacheSize,
				"utilization":  float64(warmSize) / float64(hcs.warmCacheSize),
				"hit_rate":     warmHitRate,
				"ttl_hours":    hcs.warmTTL.Hours(),
			},
		},
		"performance": map[string]interface{}{
			"overall_hit_rate":    overallHitRate,
			"promotion_count":     stats.PromotionCount,
			"eviction_count":      stats.EvictionCount,
			"database_fallback":   stats.DatabaseHits,
		},
		"configuration": map[string]interface{}{
			"hot_cache_target":    hcs.hotCacheSize,
			"warm_cache_target":   hcs.warmCacheSize,
			"hot_ttl_hours":       hcs.hotTTL.Hours(),
			"warm_ttl_hours":      hcs.warmTTL.Hours(),
		},
	}
}

// ResetCacheStatistics clears performance counters for fresh monitoring period
func (hcs *HotCacheService) ResetCacheStatistics(ctx context.Context) {
	hcs.cacheStats = &CacheStatistics{LastReset: time.Now()}
	hcs.logger.Info("Cache statistics reset")
}

// BulkCachePreload efficiently loads multiple interactions into appropriate cache tiers
func (hcs *HotCacheService) BulkCachePreload(
	ctx context.Context,
	interactions []CacheableInteraction,
) error {
	
	var hotCount, warmCount int
	
	for _, interaction := range interactions {
		cacheKey := hcs.generateCacheKey(
			interaction.DrugCodes,
			nil, // Bulk preload doesn't include PGX context
			interaction.DatasetVersion,
		)
		
		// Determine cache tier
		if hcs.isHighValueInteraction(interaction.InteractionResults) {
			if err := hcs.setHotCache(ctx, cacheKey, &interaction); err != nil {
				hcs.logger.Error("Bulk hot cache preload failed", 
					zap.String("cache_key", cacheKey), zap.Error(err))
				continue
			}
			hotCount++
		} else {
			if err := hcs.setWarmCache(ctx, cacheKey, &interaction); err != nil {
				hcs.logger.Error("Bulk warm cache preload failed",
					zap.String("cache_key", cacheKey), zap.Error(err))
				continue
			}
			warmCount++
		}
	}
	
	hcs.logger.Info("Bulk cache preload completed",
		zap.Int("hot_loaded", hotCount),
		zap.Int("warm_loaded", warmCount),
		zap.Int("total_attempted", len(interactions)))
	
	return nil
}

// OptimizeCacheDistribution rebalances cache based on access patterns
func (hcs *HotCacheService) OptimizeCacheDistribution(ctx context.Context) error {
	hcs.logger.Info("Starting cache optimization")
	
	// Get access frequency data
	frequencyData, err := hcs.hotClient.ZRevRangeWithScores(ctx, "access_frequency", 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get frequency data: %w", err)
	}
	
	// Analyze patterns and suggest cache redistributions
	var promotionCandidates []string
	var demotionCandidates []string
	
	for _, item := range frequencyData {
		key := item.Member.(string)
		frequency := item.Score
		
		// Check if item should be promoted to hot cache
		if frequency >= 10 && !hcs.isInHotCache(ctx, key) {
			promotionCandidates = append(promotionCandidates, key)
		}
		
		// Check if hot cache items should be demoted due to low usage
		if frequency < 2 && hcs.isInHotCache(ctx, key) {
			demotionCandidates = append(demotionCandidates, key)
		}
	}
	
	// Execute optimizations
	for _, key := range promotionCandidates[:min(len(promotionCandidates), 100)] {
		if interaction, found := hcs.getFromWarmCache(ctx, key); found {
			hcs.promoteToHotCache(ctx, key, interaction)
		}
	}
	
	// Note: Demotion would move items to warm cache
	// Implementation depends on cache tier management strategy
	
	hcs.logger.Info("Cache optimization completed",
		zap.Int("promotion_candidates", len(promotionCandidates)),
		zap.Int("demotion_candidates", len(demotionCandidates)))
	
	return nil
}

// isInHotCache checks if a key exists in hot cache
func (hcs *HotCacheService) isInHotCache(ctx context.Context, key string) bool {
	hotKey := fmt.Sprintf("hot:%s", key)
	exists, err := hcs.hotClient.Exists(ctx, hotKey).Result()
	if err != nil {
		return false
	}
	return exists > 0
}

// Helper function for minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}