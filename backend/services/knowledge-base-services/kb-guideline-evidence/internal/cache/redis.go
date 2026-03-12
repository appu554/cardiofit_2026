package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"kb-guideline-evidence/internal/config"
)

// CacheClient wraps Redis client with KB-3 specific functionality
type CacheClient struct {
	client *redis.Client
	config *config.Config
	ctx    context.Context
}

// NewCacheClient creates a new Redis cache client
func NewCacheClient(cfg *config.Config) (*CacheClient, error) {
	// Parse Redis URL
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override password if provided
	if cfg.RedisPassword != "" {
		opts.Password = cfg.RedisPassword
	}

	// Override DB if provided
	if cfg.RedisDB > 0 {
		opts.DB = cfg.RedisDB
	}

	// Create Redis client
	rdb := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis successfully (DB: %d)", opts.DB)

	return &CacheClient{
		client: rdb,
		config: cfg,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (c *CacheClient) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}
	log.Println("Redis connection closed")
	return nil
}

// HealthCheck performs a Redis health check
func (c *CacheClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}
	return nil
}

// Cache keys and TTL configurations
const (
	GuidelineCacheKeyPrefix      = "kb3:guideline:"
	RecommendationCacheKeyPrefix = "kb3:recommendation:"
	SearchCacheKeyPrefix         = "kb3:search:"
	CrossKBCacheKeyPrefix        = "kb3:crosskb:"
	RegionalCacheKeyPrefix       = "kb3:regional:"

	DefaultGuidelineTTL      = 1 * time.Hour    // Guidelines change infrequently
	DefaultRecommendationTTL = 1 * time.Hour    // Recommendations change infrequently
	DefaultSearchTTL         = 30 * time.Minute // Search results can be cached for shorter periods
	DefaultCrossKBTTL        = 6 * time.Hour    // Cross-KB validation expensive, cache longer
	DefaultRegionalTTL       = 24 * time.Hour   // Regional profiles change rarely
)

// Guideline caching methods

// GuidelineCacheKey generates cache key for guideline
func GuidelineCacheKey(guidelineID string, version *string) string {
	if version != nil {
		return fmt.Sprintf("%s%s:v:%s", GuidelineCacheKeyPrefix, guidelineID, *version)
	}
	return fmt.Sprintf("%s%s", GuidelineCacheKeyPrefix, guidelineID)
}

// SetGuideline caches a guideline document
func (c *CacheClient) SetGuideline(key string, guideline interface{}) error {
	return c.setJSON(key, guideline, DefaultGuidelineTTL)
}

// GetGuideline retrieves a cached guideline
func (c *CacheClient) GetGuideline(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Recommendation caching methods

// RecommendationCacheKey generates cache key for recommendation
func RecommendationCacheKey(recID string) string {
	return fmt.Sprintf("%s%s", RecommendationCacheKeyPrefix, recID)
}

// SetRecommendation caches a recommendation
func (c *CacheClient) SetRecommendation(key string, recommendation interface{}) error {
	return c.setJSON(key, recommendation, DefaultRecommendationTTL)
}

// GetRecommendation retrieves a cached recommendation
func (c *CacheClient) GetRecommendation(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Search caching methods

// SearchCacheKey generates cache key for search results
func SearchCacheKey(query string, region *string) string {
	if region != nil {
		return fmt.Sprintf("%ssq:%s:region:%s", SearchCacheKeyPrefix, query, *region)
	}
	return fmt.Sprintf("%ssq:%s", SearchCacheKeyPrefix, query)
}

// SetSearchResults caches search results
func (c *CacheClient) SetSearchResults(key string, results interface{}) error {
	return c.setJSON(key, results, DefaultSearchTTL)
}

// GetSearchResults retrieves cached search results
func (c *CacheClient) GetSearchResults(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Cross-KB validation caching methods

// CrossKBCacheKey generates cache key for cross-KB validation
func CrossKBCacheKey(recID string, kbName string) string {
	return fmt.Sprintf("%s%s:%s", CrossKBCacheKeyPrefix, recID, kbName)
}

// SetCrossKBValidation caches cross-KB validation results
func (c *CacheClient) SetCrossKBValidation(key string, validation interface{}) error {
	return c.setJSON(key, validation, DefaultCrossKBTTL)
}

// GetCrossKBValidation retrieves cached cross-KB validation
func (c *CacheClient) GetCrossKBValidation(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Regional profile caching methods

// RegionalCacheKey generates cache key for regional profiles
func RegionalCacheKey(region string) string {
	return fmt.Sprintf("%s%s", RegionalCacheKeyPrefix, region)
}

// SetRegionalProfile caches a regional profile
func (c *CacheClient) SetRegionalProfile(key string, profile interface{}) error {
	return c.setJSON(key, profile, DefaultRegionalTTL)
}

// GetRegionalProfile retrieves a cached regional profile
func (c *CacheClient) GetRegionalProfile(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Generic caching methods

// setJSON stores a value as JSON in Redis with TTL
func (c *CacheClient) setJSON(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
	}

	if err := c.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	return nil
}

// getJSON retrieves a JSON value from Redis and unmarshals it
func (c *CacheClient) getJSON(key string, result interface{}) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	jsonData, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(jsonData), result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key from cache
func (c *CacheClient) Delete(key string) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}

// DeletePattern removes all keys matching a pattern
func (c *CacheClient) DeletePattern(pattern string) error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// Get all keys matching pattern
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all matching keys
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete keys for pattern %s: %w", pattern, err)
	}

	log.Printf("Deleted %d cache keys matching pattern: %s", len(keys), pattern)
	return nil
}

// Exists checks if a key exists in cache
func (c *CacheClient) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}

	return exists > 0, nil
}

// SetTTL updates the TTL for a key
func (c *CacheClient) SetTTL(key string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL for key %s: %w", key, err)
	}

	return nil
}

// GetTTL returns the remaining TTL for a key
func (c *CacheClient) GetTTL(key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	return ttl, nil
}

// GetStats returns cache statistics
func (c *CacheClient) GetStats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	info := c.client.Info(ctx, "memory", "stats", "keyspace").Val()
	
	return map[string]interface{}{
		"redis_info": info,
		"connection_pool_stats": c.client.PoolStats(),
	}
}

// InvalidateGuidelineCache invalidates all cache entries for a guideline
func (c *CacheClient) InvalidateGuidelineCache(guidelineID string) error {
	patterns := []string{
		fmt.Sprintf("%s%s*", GuidelineCacheKeyPrefix, guidelineID),
		fmt.Sprintf("%ssq:*", SearchCacheKeyPrefix), // Invalidate search results as they might include this guideline
	}

	for _, pattern := range patterns {
		if err := c.DeletePattern(pattern); err != nil {
			return fmt.Errorf("failed to invalidate cache pattern %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidateSearchCache invalidates all search cache entries
func (c *CacheClient) InvalidateSearchCache() error {
	return c.DeletePattern(fmt.Sprintf("%s*", SearchCacheKeyPrefix))
}

// InvalidateCrossKBCache invalidates cross-KB validation cache for a recommendation
func (c *CacheClient) InvalidateCrossKBCache(recID string) error {
	return c.DeletePattern(fmt.Sprintf("%s%s:*", CrossKBCacheKeyPrefix, recID))
}

// Cache warming methods

// WarmGuidelineCache pre-loads frequently accessed guidelines
func (c *CacheClient) WarmGuidelineCache(guidelines []interface{}) error {
	for i, guideline := range guidelines {
		// Type assertion would be needed here based on actual guideline struct
		// This is a placeholder implementation
		key := fmt.Sprintf("%swarm:%d", GuidelineCacheKeyPrefix, i)
		if err := c.setJSON(key, guideline, DefaultGuidelineTTL); err != nil {
			log.Printf("Failed to warm cache for guideline %d: %v", i, err)
		}
	}
	return nil
}

// Custom errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)

// Batch operations for improved performance

// MGet retrieves multiple keys at once
func (c *CacheClient) MGet(keys []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	result, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple keys: %w", err)
	}

	values := make([]string, len(result))
	for i, val := range result {
		if val != nil {
			values[i] = val.(string)
		}
	}

	return values, nil
}

// MSet sets multiple key-value pairs at once
func (c *CacheClient) MSet(keyValues map[string]interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	// Prepare key-value pairs for Redis MSet
	pairs := make([]interface{}, 0, len(keyValues)*2)
	for key, value := range keyValues {
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
		}
		pairs = append(pairs, key, string(jsonData))
	}

	// Set all key-value pairs
	if err := c.client.MSet(ctx, pairs...).Err(); err != nil {
		return fmt.Errorf("failed to set multiple keys: %w", err)
	}

	// Set TTL for all keys (Redis doesn't support TTL in MSet)
	if ttl > 0 {
		keys := make([]string, 0, len(keyValues))
		for key := range keyValues {
			keys = append(keys, key)
		}
		
		pipe := c.client.Pipeline()
		for _, key := range keys {
			pipe.Expire(ctx, key, ttl)
		}
		
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("failed to set TTL for multiple keys: %w", err)
		}
	}

	return nil
}