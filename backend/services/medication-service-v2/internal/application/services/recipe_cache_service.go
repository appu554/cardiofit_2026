package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/infrastructure/redis"
)

// RecipeCacheService manages recipe resolution caching
type RecipeCacheService struct {
	redisClient      redis.Client
	defaultTTL       time.Duration
	performanceTarget time.Duration
	cacheStats       *CacheStatistics
	compressionEnabled bool
}

// CacheStatistics tracks cache performance
type CacheStatistics struct {
	TotalRequests    int64     `json:"total_requests"`
	CacheHits        int64     `json:"cache_hits"`
	CacheMisses      int64     `json:"cache_misses"`
	HitRate          float64   `json:"hit_rate"`
	AverageGetTime   time.Duration `json:"average_get_time"`
	AverageSetTime   time.Duration `json:"average_set_time"`
	TotalMemoryUsed  int64     `json:"total_memory_used"`
	EntriesCount     int64     `json:"entries_count"`
	LastUpdated      time.Time `json:"last_updated"`
}

// CacheKey represents different types of cache keys
type CacheKey struct {
	Type       CacheKeyType `json:"type"`
	Identifier string       `json:"identifier"`
	TTL        time.Duration `json:"ttl"`
}

// CacheKeyType defines different cache key types
type CacheKeyType string

const (
	CacheKeyRecipeResolution CacheKeyType = "recipe_resolution"
	CacheKeyFieldResolution  CacheKeyType = "field_resolution"
	CacheKeyRuleEvaluation   CacheKeyType = "rule_evaluation"
	CacheKeyProtocolData     CacheKeyType = "protocol_data"
	CacheKeyPatientSnapshot  CacheKeyType = "patient_snapshot"
)

// CacheConfig defines cache configuration
type CacheConfig struct {
	DefaultTTL         time.Duration `json:"default_ttl"`
	PerformanceTarget  time.Duration `json:"performance_target"`
	CompressionEnabled bool          `json:"compression_enabled"`
	MaxMemoryUsage     int64         `json:"max_memory_usage"`
	EvictionPolicy     string        `json:"eviction_policy"`
}

// CacheEntry represents a cached entry with metadata
type CacheEntry struct {
	Data        interface{} `json:"data"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	AccessCount int64       `json:"access_count"`
	LastAccessed time.Time  `json:"last_accessed"`
	CacheKey    string      `json:"cache_key"`
	Size        int64       `json:"size"`
}

// NewRecipeCacheService creates a new recipe cache service
func NewRecipeCacheService(redisClient redis.Client, config CacheConfig) *RecipeCacheService {
	return &RecipeCacheService{
		redisClient:        redisClient,
		defaultTTL:         config.DefaultTTL,
		performanceTarget:  config.PerformanceTarget,
		compressionEnabled: config.CompressionEnabled,
		cacheStats: &CacheStatistics{
			LastUpdated: time.Now(),
		},
	}
}

// GetRecipeResolution retrieves a cached recipe resolution
func (c *RecipeCacheService) GetRecipeResolution(ctx context.Context, recipeID uuid.UUID, patientID string) (*entities.RecipeResolution, error) {
	startTime := time.Now()
	defer c.updateGetMetrics(time.Since(startTime))

	cacheKey := c.buildRecipeResolutionKey(recipeID, patientID)
	
	// Get from cache
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err != nil {
		c.cacheStats.CacheMisses++
		return nil, err
	}

	if data == "" {
		c.cacheStats.CacheMisses++
		return nil, errors.New("cache miss")
	}

	// Deserialize
	var resolution entities.RecipeResolution
	if err := json.Unmarshal([]byte(data), &resolution); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize cached resolution")
	}

	// Update access statistics
	c.cacheStats.CacheHits++
	c.updateHitRate()

	return &resolution, nil
}

// SetRecipeResolution caches a recipe resolution
func (c *RecipeCacheService) SetRecipeResolution(ctx context.Context, recipeID uuid.UUID, patientID string, resolution *entities.RecipeResolution, ttl time.Duration) error {
	startTime := time.Now()
	defer c.updateSetMetrics(time.Since(startTime))

	cacheKey := c.buildRecipeResolutionKey(recipeID, patientID)
	
	// Serialize
	data, err := json.Marshal(resolution)
	if err != nil {
		return errors.Wrap(err, "failed to serialize resolution for caching")
	}

	// Use provided TTL or default
	if ttl == 0 {
		ttl = c.defaultTTL
	}

	// Set in cache
	if err := c.redisClient.SetEx(ctx, cacheKey, string(data), ttl); err != nil {
		return errors.Wrap(err, "failed to cache resolution")
	}

	// Update statistics
	c.cacheStats.EntriesCount++
	c.cacheStats.TotalMemoryUsed += int64(len(data))

	return nil
}

// GetFieldResolution retrieves cached field resolution
func (c *RecipeCacheService) GetFieldResolution(ctx context.Context, fieldName string, patientID string, protocolID string) (*entities.ResolvedField, error) {
	startTime := time.Now()
	defer c.updateGetMetrics(time.Since(startTime))

	cacheKey := c.buildFieldResolutionKey(fieldName, patientID, protocolID)
	
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err != nil {
		c.cacheStats.CacheMisses++
		return nil, err
	}

	if data == "" {
		c.cacheStats.CacheMisses++
		return nil, errors.New("cache miss")
	}

	var field entities.ResolvedField
	if err := json.Unmarshal([]byte(data), &field); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize cached field")
	}

	c.cacheStats.CacheHits++
	c.updateHitRate()

	return &field, nil
}

// SetFieldResolution caches a field resolution
func (c *RecipeCacheService) SetFieldResolution(ctx context.Context, fieldName string, patientID string, protocolID string, field *entities.ResolvedField, ttl time.Duration) error {
	startTime := time.Now()
	defer c.updateSetMetrics(time.Since(startTime))

	cacheKey := c.buildFieldResolutionKey(fieldName, patientID, protocolID)
	
	data, err := json.Marshal(field)
	if err != nil {
		return errors.Wrap(err, "failed to serialize field for caching")
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.redisClient.SetEx(ctx, cacheKey, string(data), ttl); err != nil {
		return errors.Wrap(err, "failed to cache field resolution")
	}

	c.cacheStats.EntriesCount++
	c.cacheStats.TotalMemoryUsed += int64(len(data))

	return nil
}

// GetRuleEvaluation retrieves cached rule evaluation
func (c *RecipeCacheService) GetRuleEvaluation(ctx context.Context, ruleID uuid.UUID, patientID string) (*EvaluationResult, error) {
	startTime := time.Now()
	defer c.updateGetMetrics(time.Since(startTime))

	cacheKey := c.buildRuleEvaluationKey(ruleID, patientID)
	
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err != nil {
		c.cacheStats.CacheMisses++
		return nil, err
	}

	if data == "" {
		c.cacheStats.CacheMisses++
		return nil, errors.New("cache miss")
	}

	var result EvaluationResult
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize cached evaluation result")
	}

	c.cacheStats.CacheHits++
	c.updateHitRate()

	return &result, nil
}

// SetRuleEvaluation caches a rule evaluation
func (c *RecipeCacheService) SetRuleEvaluation(ctx context.Context, ruleID uuid.UUID, patientID string, result *EvaluationResult, ttl time.Duration) error {
	startTime := time.Now()
	defer c.updateSetMetrics(time.Since(startTime))

	cacheKey := c.buildRuleEvaluationKey(ruleID, patientID)
	
	data, err := json.Marshal(result)
	if err != nil {
		return errors.Wrap(err, "failed to serialize evaluation result for caching")
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.redisClient.SetEx(ctx, cacheKey, string(data), ttl); err != nil {
		return errors.Wrap(err, "failed to cache evaluation result")
	}

	c.cacheStats.EntriesCount++
	c.cacheStats.TotalMemoryUsed += int64(len(data))

	return nil
}

// GetProtocolData retrieves cached protocol-specific data
func (c *RecipeCacheService) GetProtocolData(ctx context.Context, protocolID string, dataType string) (interface{}, error) {
	startTime := time.Now()
	defer c.updateGetMetrics(time.Since(startTime))

	cacheKey := c.buildProtocolDataKey(protocolID, dataType)
	
	data, err := c.redisClient.Get(ctx, cacheKey)
	if err != nil {
		c.cacheStats.CacheMisses++
		return nil, err
	}

	if data == "" {
		c.cacheStats.CacheMisses++
		return nil, errors.New("cache miss")
	}

	var result interface{}
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, errors.Wrap(err, "failed to deserialize cached protocol data")
	}

	c.cacheStats.CacheHits++
	c.updateHitRate()

	return result, nil
}

// SetProtocolData caches protocol-specific data
func (c *RecipeCacheService) SetProtocolData(ctx context.Context, protocolID string, dataType string, data interface{}, ttl time.Duration) error {
	startTime := time.Now()
	defer c.updateSetMetrics(time.Since(startTime))

	cacheKey := c.buildProtocolDataKey(protocolID, dataType)
	
	serialized, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to serialize protocol data for caching")
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.redisClient.SetEx(ctx, cacheKey, string(serialized), ttl); err != nil {
		return errors.Wrap(err, "failed to cache protocol data")
	}

	c.cacheStats.EntriesCount++
	c.cacheStats.TotalMemoryUsed += int64(len(serialized))

	return nil
}

// InvalidatePatientCache invalidates all cache entries for a patient
func (c *RecipeCacheService) InvalidatePatientCache(ctx context.Context, patientID string) error {
	pattern := fmt.Sprintf("*:patient:%s", patientID)
	
	keys, err := c.redisClient.Keys(ctx, pattern)
	if err != nil {
		return errors.Wrap(err, "failed to get keys for patient cache invalidation")
	}

	if len(keys) > 0 {
		if err := c.redisClient.Del(ctx, keys...); err != nil {
			return errors.Wrap(err, "failed to delete patient cache entries")
		}

		// Update statistics
		c.cacheStats.EntriesCount -= int64(len(keys))
	}

	return nil
}

// InvalidateRecipeCache invalidates all cache entries for a recipe
func (c *RecipeCacheService) InvalidateRecipeCache(ctx context.Context, recipeID uuid.UUID) error {
	pattern := fmt.Sprintf("recipe_resolution:%s:*", recipeID.String())
	
	keys, err := c.redisClient.Keys(ctx, pattern)
	if err != nil {
		return errors.Wrap(err, "failed to get keys for recipe cache invalidation")
	}

	if len(keys) > 0 {
		if err := c.redisClient.Del(ctx, keys...); err != nil {
			return errors.Wrap(err, "failed to delete recipe cache entries")
		}

		c.cacheStats.EntriesCount -= int64(len(keys))
	}

	return nil
}

// InvalidateProtocolCache invalidates all cache entries for a protocol
func (c *RecipeCacheService) InvalidateProtocolCache(ctx context.Context, protocolID string) error {
	patterns := []string{
		fmt.Sprintf("field_resolution:*:*:%s", protocolID),
		fmt.Sprintf("protocol_data:%s:*", protocolID),
	}

	for _, pattern := range patterns {
		keys, err := c.redisClient.Keys(ctx, pattern)
		if err != nil {
			return errors.Wrap(err, "failed to get keys for protocol cache invalidation")
		}

		if len(keys) > 0 {
			if err := c.redisClient.Del(ctx, keys...); err != nil {
				return errors.Wrap(err, "failed to delete protocol cache entries")
			}

			c.cacheStats.EntriesCount -= int64(len(keys))
		}
	}

	return nil
}

// ClearExpiredEntries removes expired cache entries
func (c *RecipeCacheService) ClearExpiredEntries(ctx context.Context) error {
	// Redis automatically handles TTL expiration, but we can scan for expired entries
	// if needed for statistics updates
	return nil
}

// GetCacheStatistics returns current cache statistics
func (c *RecipeCacheService) GetCacheStatistics(ctx context.Context) (*CacheStatistics, error) {
	c.cacheStats.LastUpdated = time.Now()
	return c.cacheStats, nil
}

// GetMemoryUsage returns current memory usage statistics
func (c *RecipeCacheService) GetMemoryUsage(ctx context.Context) (map[string]interface{}, error) {
	info, err := c.redisClient.Info(ctx, "memory")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get memory info from Redis")
	}

	return map[string]interface{}{
		"redis_info": info,
		"cache_stats": c.cacheStats,
	}, nil
}

// SetCacheTTL updates TTL for specific cache patterns
func (c *RecipeCacheService) SetCacheTTL(ctx context.Context, keyPattern string, ttl time.Duration) error {
	keys, err := c.redisClient.Keys(ctx, keyPattern)
	if err != nil {
		return errors.Wrap(err, "failed to get keys for TTL update")
	}

	for _, key := range keys {
		if err := c.redisClient.Expire(ctx, key, ttl); err != nil {
			return errors.Wrapf(err, "failed to update TTL for key %s", key)
		}
	}

	return nil
}

// OptimizeCache performs cache optimization tasks
func (c *RecipeCacheService) OptimizeCache(ctx context.Context) error {
	// Clear expired entries
	if err := c.ClearExpiredEntries(ctx); err != nil {
		return errors.Wrap(err, "failed to clear expired entries")
	}

	// Compress frequently accessed entries if compression is enabled
	if c.compressionEnabled {
		// Implementation would go here
	}

	return nil
}

// Key building methods

func (c *RecipeCacheService) buildRecipeResolutionKey(recipeID uuid.UUID, patientID string) string {
	return fmt.Sprintf("recipe_resolution:%s:patient:%s", recipeID.String(), patientID)
}

func (c *RecipeCacheService) buildFieldResolutionKey(fieldName, patientID, protocolID string) string {
	return fmt.Sprintf("field_resolution:%s:patient:%s:protocol:%s", fieldName, patientID, protocolID)
}

func (c *RecipeCacheService) buildRuleEvaluationKey(ruleID uuid.UUID, patientID string) string {
	return fmt.Sprintf("rule_evaluation:%s:patient:%s", ruleID.String(), patientID)
}

func (c *RecipeCacheService) buildProtocolDataKey(protocolID, dataType string) string {
	return fmt.Sprintf("protocol_data:%s:%s", protocolID, dataType)
}

func (c *RecipeCacheService) buildPatientSnapshotKey(patientID string, timestamp time.Time) string {
	return fmt.Sprintf("patient_snapshot:%s:%d", patientID, timestamp.Unix())
}

// Statistics update methods

func (c *RecipeCacheService) updateGetMetrics(duration time.Duration) {
	c.cacheStats.TotalRequests++
	
	// Update average get time
	totalTime := time.Duration(c.cacheStats.TotalRequests) * c.cacheStats.AverageGetTime
	c.cacheStats.AverageGetTime = (totalTime + duration) / time.Duration(c.cacheStats.TotalRequests)
}

func (c *RecipeCacheService) updateSetMetrics(duration time.Duration) {
	// Update average set time
	c.cacheStats.AverageSetTime = (c.cacheStats.AverageSetTime + duration) / 2
}

func (c *RecipeCacheService) updateHitRate() {
	if c.cacheStats.TotalRequests > 0 {
		c.cacheStats.HitRate = float64(c.cacheStats.CacheHits) / float64(c.cacheStats.TotalRequests)
	}
}

// IsCacheHealthy checks if cache performance meets targets
func (c *RecipeCacheService) IsCacheHealthy() bool {
	// Check if average get time meets performance target
	if c.cacheStats.AverageGetTime > c.performanceTarget {
		return false
	}

	// Check if hit rate is acceptable (>80%)
	if c.cacheStats.HitRate < 0.8 {
		return false
	}

	return true
}

// GetCacheHealth returns detailed cache health information
func (c *RecipeCacheService) GetCacheHealth(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"healthy":               c.IsCacheHealthy(),
		"hit_rate":             c.cacheStats.HitRate,
		"average_get_time_ms":   c.cacheStats.AverageGetTime.Milliseconds(),
		"performance_target_ms": c.performanceTarget.Milliseconds(),
		"total_requests":       c.cacheStats.TotalRequests,
		"cache_hits":           c.cacheStats.CacheHits,
		"cache_misses":         c.cacheStats.CacheMisses,
		"entries_count":        c.cacheStats.EntriesCount,
		"memory_used_bytes":    c.cacheStats.TotalMemoryUsed,
		"last_updated":         c.cacheStats.LastUpdated,
	}
}

// ResetStatistics resets cache statistics
func (c *RecipeCacheService) ResetStatistics() {
	c.cacheStats = &CacheStatistics{
		LastUpdated: time.Now(),
	}
}