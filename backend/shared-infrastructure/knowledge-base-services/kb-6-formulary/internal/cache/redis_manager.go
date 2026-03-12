package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"kb-formulary/internal/config"
)

// RedisManager manages Redis cache operations for KB-6
type RedisManager struct {
	client      *redis.Client
	cfg         *config.RedisConfig
	keyPrefix   string
	datasetVersion string
}

// NewRedisManager creates a new Redis cache manager
func NewRedisManager(cfg *config.RedisConfig) (*RedisManager, error) {
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.Database,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	manager := &RedisManager{
		client:         client,
		cfg:            cfg,
		keyPrefix:      "kb6:",
		datasetVersion: "kb6.formulary.2025Q3.v1", // TODO: Make this configurable
	}

	log.Printf("Redis cache manager initialized (DB: %d, Dataset: %s)", cfg.Database, manager.datasetVersion)
	return manager, nil
}

// Close closes the Redis connection
func (rm *RedisManager) Close() error {
	return rm.client.Close()
}

// GetCoverage retrieves formulary coverage from cache
func (rm *RedisManager) GetCoverage(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Get from Redis
	fullKey := rm.buildCoverageKey(key)
	result := rm.client.Get(ctx, fullKey)
	if result.Err() == redis.Nil {
		return nil, nil // Cache miss
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("Redis get error: %w", result.Err())
	}

	log.Printf("Cache hit: formulary coverage for key %s", key)
	return []byte(result.Val()), nil
}

// SetCoverage stores formulary coverage in cache
func (rm *RedisManager) SetCoverage(key string, data []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Store in Redis
	fullKey := rm.buildCoverageKey(key)
	if err := rm.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("Redis set error: %w", err)
	}

	log.Printf("Cache set: formulary coverage for key %s (TTL: %v)", key, ttl)
	return nil
}

// GetStock retrieves stock information from cache
func (rm *RedisManager) GetStock(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond) // Shorter timeout for stock
	defer cancel()

	// Get from Redis
	fullKey := rm.buildStockKey(key)
	result := rm.client.Get(ctx, fullKey)
	if result.Err() == redis.Nil {
		return nil, nil // Cache miss
	}
	if result.Err() != nil {
		return nil, fmt.Errorf("Redis get error: %w", result.Err())
	}

	log.Printf("Cache hit: stock info for key %s", key)
	return []byte(result.Val()), nil
}

// SetStock stores stock information in cache
func (rm *RedisManager) SetStock(key string, data []byte, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Store in Redis
	fullKey := rm.buildStockKey(key)
	if err := rm.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("Redis set error: %w", err)
	}

	log.Printf("Cache set: stock info for key %s (TTL: %v)", key, ttl)
	return nil
}

// InvalidateFormulary invalidates all formulary cache entries for a dataset version
func (rm *RedisManager) InvalidateFormulary(datasetVersion string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all keys with the dataset version
	pattern := fmt.Sprintf("%sformulary:%s:*", rm.keyPrefix, datasetVersion)
	keys := rm.client.Keys(ctx, pattern)
	
	if keys.Err() != nil {
		return fmt.Errorf("failed to get keys for invalidation: %w", keys.Err())
	}

	keyList := keys.Val()
	if len(keyList) == 0 {
		log.Printf("No formulary cache keys to invalidate for dataset %s", datasetVersion)
		return nil
	}

	// Delete keys in batches
	batchSize := 100
	for i := 0; i < len(keyList); i += batchSize {
		end := i + batchSize
		if end > len(keyList) {
			end = len(keyList)
		}
		
		batch := keyList[i:end]
		if err := rm.client.Del(ctx, batch...).Err(); err != nil {
			return fmt.Errorf("failed to delete cache keys: %w", err)
		}
	}

	log.Printf("Invalidated %d formulary cache keys for dataset %s", len(keyList), datasetVersion)
	return nil
}

// InvalidateStock invalidates stock cache entries for a location
func (rm *RedisManager) InvalidateStock(locationID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all stock keys for the location
	pattern := fmt.Sprintf("%sstock:%s:%s:*", rm.keyPrefix, rm.datasetVersion, locationID)
	keys := rm.client.Keys(ctx, pattern)
	
	if keys.Err() != nil {
		return fmt.Errorf("failed to get stock keys for invalidation: %w", keys.Err())
	}

	keyList := keys.Val()
	if len(keyList) == 0 {
		log.Printf("No stock cache keys to invalidate for location %s", locationID)
		return nil
	}

	// Delete keys
	if err := rm.client.Del(ctx, keyList...).Err(); err != nil {
		return fmt.Errorf("failed to delete stock cache keys: %w", err)
	}

	log.Printf("Invalidated %d stock cache keys for location %s", len(keyList), locationID)
	return nil
}

// GetCacheStats retrieves cache performance statistics
func (rm *RedisManager) GetCacheStats() (*CacheStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Get Redis info
	info := rm.client.Info(ctx, "stats", "memory")
	if info.Err() != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", info.Err())
	}

	// Parse basic stats (simplified)
	stats := &CacheStats{
		RedisInfo:        info.Val(),
		ConnectedClients: 1, // TODO: Parse from info
		UsedMemoryMB:     0, // TODO: Parse from info
		KeyspaceHits:     0, // TODO: Parse from info
		KeyspaceMisses:   0, // TODO: Parse from info
		HitRate:          0.0,
	}

	// Calculate hit rate if we have the data
	if stats.KeyspaceHits > 0 || stats.KeyspaceMisses > 0 {
		total := stats.KeyspaceHits + stats.KeyspaceMisses
		stats.HitRate = float64(stats.KeyspaceHits) / float64(total)
	}

	return stats, nil
}

// Ping tests Redis connectivity
func (rm *RedisManager) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	return rm.client.Ping(ctx).Err()
}

// buildCoverageKey builds a deterministic cache key for formulary coverage
func (rm *RedisManager) buildCoverageKey(baseKey string) string {
	return fmt.Sprintf("%sformulary:%s:%s", rm.keyPrefix, rm.datasetVersion, baseKey)
}

// buildStockKey builds a deterministic cache key for stock information
func (rm *RedisManager) buildStockKey(baseKey string) string {
	return fmt.Sprintf("%sstock:%s:%s", rm.keyPrefix, rm.datasetVersion, baseKey)
}

// CacheStats represents cache performance statistics
type CacheStats struct {
	RedisInfo        string  `json:"redis_info"`
	ConnectedClients int     `json:"connected_clients"`
	UsedMemoryMB     int     `json:"used_memory_mb"`
	KeyspaceHits     int64   `json:"keyspace_hits"`
	KeyspaceMisses   int64   `json:"keyspace_misses"`
	HitRate          float64 `json:"hit_rate"`
}

// CacheKeyBuilder provides utilities for building consistent cache keys
type CacheKeyBuilder struct {
	datasetVersion string
}

// NewCacheKeyBuilder creates a new cache key builder
func NewCacheKeyBuilder(datasetVersion string) *CacheKeyBuilder {
	return &CacheKeyBuilder{
		datasetVersion: datasetVersion,
	}
}

// FormularyCoverageKey builds a cache key for formulary coverage
func (ckb *CacheKeyBuilder) FormularyCoverageKey(drugRxNorm, payerID, planID string, planYear int) string {
	return fmt.Sprintf("formulary:coverage:%s:%s:%s:%d", drugRxNorm, payerID, planID, planYear)
}

// StockAvailabilityKey builds a cache key for stock availability
func (ckb *CacheKeyBuilder) StockAvailabilityKey(drugRxNorm, locationID string) string {
	return fmt.Sprintf("stock:availability:%s:%s", locationID, drugRxNorm)
}

// CostAnalysisKey builds a cache key for cost analysis
func (ckb *CacheKeyBuilder) CostAnalysisKey(drugRxNorms []string, payerID, planID string) string {
	drugsHash := fmt.Sprintf("%x", drugRxNorms) // Simple hash of drug list
	return fmt.Sprintf("cost:analysis:%s:%s:%s", payerID, planID, drugsHash)
}

// SearchResultsKey builds a cache key for search results
func (ckb *CacheKeyBuilder) SearchResultsKey(query, payerID, planID string) string {
	queryHash := fmt.Sprintf("%x", query) // Simple hash of query
	return fmt.Sprintf("search:results:%s:%s:%s", payerID, planID, queryHash)
}

// Cache warming utilities

// WarmFormularyCache pre-loads frequently accessed formulary data
func (rm *RedisManager) WarmFormularyCache(ctx context.Context, topDrugs []string, topPayers []string) error {
	log.Printf("Starting formulary cache warming for %d drugs and %d payers", len(topDrugs), len(topPayers))
	
	// TODO: Implement cache warming logic
	// This would query the database for the most frequently accessed
	// drug/payer/plan combinations and pre-load them into cache
	
	return nil
}

// WarmStockCache pre-loads frequently accessed stock data
func (rm *RedisManager) WarmStockCache(ctx context.Context, topDrugs []string, locations []string) error {
	log.Printf("Starting stock cache warming for %d drugs and %d locations", len(topDrugs), len(locations))

	// TODO: Implement stock cache warming logic
	// This would query the inventory database for the most frequently
	// accessed drug/location combinations and pre-load them into cache

	return nil
}

// =============================================================================
// LIST OPERATIONS (for Event Queue)
// =============================================================================

// LPush pushes a value to the left of a Redis list (for event queuing)
func (rm *RedisManager) LPush(ctx context.Context, key string, value string) error {
	fullKey := rm.keyPrefix + key
	if err := rm.client.LPush(ctx, fullKey, value).Err(); err != nil {
		return fmt.Errorf("Redis LPush error: %w", err)
	}
	return nil
}

// RPop pops a value from the right of a Redis list (FIFO consumption)
func (rm *RedisManager) RPop(ctx context.Context, key string) (string, error) {
	fullKey := rm.keyPrefix + key
	result := rm.client.RPop(ctx, fullKey)
	if result.Err() == redis.Nil {
		return "", nil // Empty list
	}
	if result.Err() != nil {
		return "", fmt.Errorf("Redis RPop error: %w", result.Err())
	}
	return result.Val(), nil
}

// LLen returns the length of a Redis list
func (rm *RedisManager) LLen(ctx context.Context, key string) (int64, error) {
	fullKey := rm.keyPrefix + key
	result := rm.client.LLen(ctx, fullKey)
	if result.Err() != nil {
		return 0, fmt.Errorf("Redis LLen error: %w", result.Err())
	}
	return result.Val(), nil
}

// LRange returns a range of elements from a Redis list
func (rm *RedisManager) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	fullKey := rm.keyPrefix + key
	result := rm.client.LRange(ctx, fullKey, start, stop)
	if result.Err() != nil {
		return nil, fmt.Errorf("Redis LRange error: %w", result.Err())
	}
	return result.Val(), nil
}

// =============================================================================
// GENERIC OPERATIONS
// =============================================================================

// Set stores a value in Redis with an optional TTL
func (rm *RedisManager) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	fullKey := rm.keyPrefix + key
	if err := rm.client.Set(ctx, fullKey, value, ttl).Err(); err != nil {
		return fmt.Errorf("Redis Set error: %w", err)
	}
	return nil
}

// Get retrieves a value from Redis
func (rm *RedisManager) Get(ctx context.Context, key string) (string, error) {
	fullKey := rm.keyPrefix + key
	result := rm.client.Get(ctx, fullKey)
	if result.Err() == redis.Nil {
		return "", nil // Key not found
	}
	if result.Err() != nil {
		return "", fmt.Errorf("Redis Get error: %w", result.Err())
	}
	return result.Val(), nil
}

// Delete removes a key from Redis
func (rm *RedisManager) Delete(ctx context.Context, key string) error {
	fullKey := rm.keyPrefix + key
	if err := rm.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("Redis Del error: %w", err)
	}
	return nil
}

// Exists checks if a key exists in Redis
func (rm *RedisManager) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := rm.keyPrefix + key
	result := rm.client.Exists(ctx, fullKey)
	if result.Err() != nil {
		return false, fmt.Errorf("Redis Exists error: %w", result.Err())
	}
	return result.Val() > 0, nil
}