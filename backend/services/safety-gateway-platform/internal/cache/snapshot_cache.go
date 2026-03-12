package cache

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"golang.org/x/net/context"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// SnapshotCache provides multi-level caching for clinical snapshots
type SnapshotCache struct {
	l1Cache    *MemoryCache     // In-memory LRU cache
	l2Cache    *RedisCache      // Redis distributed cache
	metrics    *CacheMetrics
	config     *config.CacheConfig
	logger     *logger.Logger
	mu         sync.RWMutex
}

// CacheMetrics tracks cache performance metrics
type CacheMetrics struct {
	L1Hits     int64
	L1Misses   int64
	L2Hits     int64
	L2Misses   int64
	Evictions  int64
	Errors     int64
	TotalRequests int64
	mu         sync.RWMutex
}

// NewSnapshotCache creates a new multi-level snapshot cache
func NewSnapshotCache(cfg *config.CacheConfig, logger *logger.Logger) (*SnapshotCache, error) {
	// Initialize L1 cache (in-memory)
	l1Cache, err := NewMemoryCache(cfg.L1MaxSize, cfg.L1TTL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create L1 cache: %w", err)
	}

	// Initialize L2 cache (Redis)
	var l2Cache *RedisCache
	if cfg.EnableL2Cache {
		l2Cache, err = NewRedisCache(cfg.Redis, logger)
		if err != nil {
			logger.Warn("Failed to create L2 cache, continuing with L1 only", zap.Error(err))
			l2Cache = nil
		}
	}

	cache := &SnapshotCache{
		l1Cache: l1Cache,
		l2Cache: l2Cache,
		metrics: &CacheMetrics{},
		config:  cfg,
		logger:  logger,
	}

	logger.Info("Snapshot cache initialized",
		zap.Int("l1_max_size", cfg.L1MaxSize),
		zap.Duration("l1_ttl", cfg.L1TTL),
		zap.Bool("l2_enabled", cfg.EnableL2Cache),
	)

	return cache, nil
}

// Get retrieves a snapshot from the cache
func (c *SnapshotCache) Get(snapshotID string) (*types.ClinicalSnapshot, bool) {
	c.metrics.incrementTotalRequests()

	// 1. Check L1 cache (in-memory)
	if snapshot, exists := c.l1Cache.Get(snapshotID); exists {
		c.metrics.incrementL1Hits()
		c.logger.Debug("L1 cache hit", zap.String("snapshot_id", snapshotID))
		return snapshot, true
	}
	c.metrics.incrementL1Misses()

	// 2. Check L2 cache (Redis) if available
	if c.l2Cache != nil {
		if snapshot, exists := c.l2Cache.Get(snapshotID); exists {
			c.metrics.incrementL2Hits()
			c.logger.Debug("L2 cache hit", zap.String("snapshot_id", snapshotID))
			
			// Store in L1 for faster future access
			c.l1Cache.Set(snapshotID, snapshot, c.config.L1TTL)
			return snapshot, true
		}
		c.metrics.incrementL2Misses()
	}

	c.logger.Debug("Cache miss", zap.String("snapshot_id", snapshotID))
	return nil, false
}

// Set stores a snapshot in the cache
func (c *SnapshotCache) Set(snapshotID string, snapshot *types.ClinicalSnapshot, ttl time.Duration) error {
	// Store in L1 cache
	c.l1Cache.Set(snapshotID, snapshot, ttl)

	// Store in L2 cache if available
	if c.l2Cache != nil {
		if err := c.l2Cache.Set(snapshotID, snapshot, c.config.L2TTL); err != nil {
			c.metrics.incrementErrors()
			c.logger.Warn("Failed to store snapshot in L2 cache",
				zap.String("snapshot_id", snapshotID),
				zap.Error(err),
			)
			// Don't fail the operation if L2 cache fails
		}
	}

	c.logger.Debug("Snapshot cached",
		zap.String("snapshot_id", snapshotID),
		zap.Duration("ttl", ttl),
	)

	return nil
}

// Delete removes a snapshot from all cache levels
func (c *SnapshotCache) Delete(snapshotID string) error {
	// Remove from L1 cache
	c.l1Cache.Delete(snapshotID)

	// Remove from L2 cache if available
	if c.l2Cache != nil {
		if err := c.l2Cache.Delete(snapshotID); err != nil {
			c.metrics.incrementErrors()
			c.logger.Warn("Failed to delete snapshot from L2 cache",
				zap.String("snapshot_id", snapshotID),
				zap.Error(err),
			)
		}
	}

	c.logger.Debug("Snapshot removed from cache", zap.String("snapshot_id", snapshotID))
	return nil
}

// GetStats returns cache statistics
func (c *SnapshotCache) GetStats() *types.SnapshotCacheStats {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	stats := &types.SnapshotCacheStats{
		L1CacheHits:   c.metrics.L1Hits,
		L1CacheMisses: c.metrics.L1Misses,
		L2CacheHits:   c.metrics.L2Hits,
		L2CacheMisses: c.metrics.L2Misses,
		TotalRequests: c.metrics.TotalRequests,
		CacheSize:     int64(c.l1Cache.Size()),
		Metadata:      make(map[string]interface{}),
	}

	// Calculate hit rates
	if stats.TotalRequests > 0 {
		l1Total := stats.L1CacheHits + stats.L1CacheMisses
		if l1Total > 0 {
			stats.L1HitRate = float64(stats.L1CacheHits) / float64(l1Total) * 100
		}

		l2Total := stats.L2CacheHits + stats.L2CacheMisses
		if l2Total > 0 {
			stats.L2HitRate = float64(stats.L2CacheHits) / float64(l2Total) * 100
		}

		totalHits := stats.L1CacheHits + stats.L2CacheHits
		stats.OverallHitRate = float64(totalHits) / float64(stats.TotalRequests) * 100
	}

	// Add L2 cache stats if available
	if c.l2Cache != nil {
		stats.Metadata["l2_enabled"] = true
		stats.Metadata["l2_connected"] = c.l2Cache.IsConnected()
	} else {
		stats.Metadata["l2_enabled"] = false
	}

	return stats
}

// Clear clears all cache levels
func (c *SnapshotCache) Clear() error {
	c.l1Cache.Clear()

	if c.l2Cache != nil {
		if err := c.l2Cache.Clear(); err != nil {
			c.metrics.incrementErrors()
			return fmt.Errorf("failed to clear L2 cache: %w", err)
		}
	}

	c.logger.Info("All cache levels cleared")
	return nil
}

// Close closes the cache and cleans up resources
func (c *SnapshotCache) Close() error {
	if c.l1Cache != nil {
		c.l1Cache.Close()
	}

	if c.l2Cache != nil {
		if err := c.l2Cache.Close(); err != nil {
			return fmt.Errorf("failed to close L2 cache: %w", err)
		}
	}

	c.logger.Info("Snapshot cache closed")
	return nil
}

// Metrics accessor methods
func (c *CacheMetrics) incrementL1Hits() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.L1Hits++
}

func (c *CacheMetrics) incrementL1Misses() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.L1Misses++
}

func (c *CacheMetrics) incrementL2Hits() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.L2Hits++
}

func (c *CacheMetrics) incrementL2Misses() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.L2Misses++
}

func (c *CacheMetrics) incrementEvictions() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Evictions++
}

func (c *CacheMetrics) incrementErrors() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Errors++
}

func (c *CacheMetrics) incrementTotalRequests() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.TotalRequests++
}

// MemoryCache implements in-memory LRU cache
type MemoryCache struct {
	cache    map[string]*cacheItem
	maxSize  int
	ttl      time.Duration
	logger   *logger.Logger
	mu       sync.RWMutex
}

type cacheItem struct {
	snapshot  *types.ClinicalSnapshot
	expiresAt time.Time
	accessTime time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(maxSize int, ttl time.Duration, logger *logger.Logger) (*MemoryCache, error) {
	cache := &MemoryCache{
		cache:   make(map[string]*cacheItem),
		maxSize: maxSize,
		ttl:     ttl,
		logger:  logger,
	}

	// Start cleanup goroutine
	go cache.cleanupExpired()

	return cache, nil
}

// Get retrieves an item from memory cache
func (m *MemoryCache) Get(key string) (*types.ClinicalSnapshot, bool) {
	m.mu.RLock()
	item, exists := m.cache[key]
	m.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.expiresAt) {
		m.mu.Lock()
		delete(m.cache, key)
		m.mu.Unlock()
		return nil, false
	}

	// Update access time
	m.mu.Lock()
	item.accessTime = time.Now()
	m.mu.Unlock()

	return item.snapshot, true
}

// Set stores an item in memory cache
func (m *MemoryCache) Set(key string, snapshot *types.ClinicalSnapshot, ttl time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Evict if at capacity
	if len(m.cache) >= m.maxSize {
		m.evictLRU()
	}

	expiresAt := time.Now().Add(ttl)
	m.cache[key] = &cacheItem{
		snapshot:   snapshot,
		expiresAt:  expiresAt,
		accessTime: time.Now(),
	}
}

// Delete removes an item from memory cache
func (m *MemoryCache) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, key)
}

// Size returns the current cache size
func (m *MemoryCache) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.cache)
}

// Clear clears all items from memory cache
func (m *MemoryCache) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]*cacheItem)
}

// Close closes the memory cache
func (m *MemoryCache) Close() {
	m.Clear()
}

// evictLRU evicts the least recently used item
func (m *MemoryCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range m.cache {
		if oldestKey == "" || item.accessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.accessTime
		}
	}

	if oldestKey != "" {
		delete(m.cache, oldestKey)
	}
}

// cleanupExpired removes expired items periodically
func (m *MemoryCache) cleanupExpired() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		now := time.Now()

		m.mu.Lock()
		for key, item := range m.cache {
			if now.After(item.expiresAt) {
				delete(m.cache, key)
			}
		}
		m.mu.Unlock()
	}
}

// RedisCache implements Redis-based distributed cache
type RedisCache struct {
	client *redis.Client
	logger *logger.Logger
	prefix string
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(cfg *config.RedisConfig, logger *logger.Logger) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		logger: logger,
		prefix: "snapshot:",
	}, nil
}

// Get retrieves an item from Redis cache
func (r *RedisCache) Get(key string) (*types.ClinicalSnapshot, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := r.client.Get(ctx, r.prefix+key).Result()
	if err != nil {
		if err != redis.Nil {
			r.logger.Warn("Redis get failed", zap.String("key", key), zap.Error(err))
		}
		return nil, false
	}

	var snapshot types.ClinicalSnapshot
	if err := json.Unmarshal([]byte(data), &snapshot); err != nil {
		r.logger.Warn("Failed to unmarshal snapshot from Redis",
			zap.String("key", key),
			zap.Error(err),
		)
		return nil, false
	}

	return &snapshot, true
}

// Set stores an item in Redis cache
func (r *RedisCache) Set(key string, snapshot *types.ClinicalSnapshot, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	if err := r.client.Set(ctx, r.prefix+key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set Redis key: %w", err)
	}

	return nil
}

// Delete removes an item from Redis cache
func (r *RedisCache) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := r.client.Del(ctx, r.prefix+key).Err(); err != nil {
		return fmt.Errorf("failed to delete Redis key: %w", err)
	}

	return nil
}

// Clear clears all snapshots from Redis cache
func (r *RedisCache) Clear() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keys, err := r.client.Keys(ctx, r.prefix+"*").Result()
	if err != nil {
		return fmt.Errorf("failed to get Redis keys: %w", err)
	}

	if len(keys) > 0 {
		if err := r.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete Redis keys: %w", err)
		}
	}

	return nil
}

// IsConnected checks if Redis connection is healthy
func (r *RedisCache) IsConnected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return r.client.Ping(ctx).Err() == nil
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}