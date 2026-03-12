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

// CacheLevel represents the cache level in the hierarchy
type CacheLevel int

const (
	L1Cache CacheLevel = 1 // In-memory cache
	L2Cache CacheLevel = 2 // Redis cache
	L3Cache CacheLevel = 3 // Database fallback
)

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key        string      `json:"key"`
	Value      interface{} `json:"value"`
	Level      CacheLevel  `json:"level"`
	TTL        time.Duration `json:"ttl"`
	CreatedAt  time.Time   `json:"created_at"`
	AccessedAt time.Time   `json:"accessed_at"`
	AccessCount int64      `json:"access_count"`
	Size       int64      `json:"size"`
	Tags       []string   `json:"tags,omitempty"`
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	L1Hits       int64 `json:"l1_hits"`
	L1Misses     int64 `json:"l1_misses"`
	L2Hits       int64 `json:"l2_hits"`
	L2Misses     int64 `json:"l2_misses"`
	L3Hits       int64 `json:"l3_hits"`
	L3Misses     int64 `json:"l3_misses"`
	Promotions   int64 `json:"promotions"`
	Demotions    int64 `json:"demotions"`
	Invalidations int64 `json:"invalidations"`
	TotalSize    int64 `json:"total_size"`
	LastReset    time.Time `json:"last_reset"`
}

// MultiLevelCache implements intelligent multi-level caching
type MultiLevelCache struct {
	l1Cache    *sync.Map      // In-memory cache
	l2Client   *redis.Client  // Redis cache
	logger     *zap.Logger
	stats      *CacheStats
	statsMutex sync.RWMutex
	
	// Configuration
	l1MaxSize     int64
	l1TTL         time.Duration
	l2TTL         time.Duration
	promotionThreshold int64
	demotionThreshold  time.Duration
	
	// HIPAA compliance
	encryptionEnabled bool
	auditLog         *CacheAuditLog
}

// CacheAuditLog tracks cache operations for HIPAA compliance
type CacheAuditLog struct {
	logger *zap.Logger
	mutex  sync.Mutex
}

// CacheConfig contains configuration for the multi-level cache
type CacheConfig struct {
	RedisURL           string        `json:"redis_url"`
	L1MaxSize          int64         `json:"l1_max_size"`          // Max entries in L1
	L1TTL              time.Duration `json:"l1_ttl"`               // L1 cache TTL
	L2TTL              time.Duration `json:"l2_ttl"`               // L2 cache TTL
	PromotionThreshold int64         `json:"promotion_threshold"`  // Access count for L1 promotion
	DemotionTimeout    time.Duration `json:"demotion_timeout"`     // Time before L1 demotion
	EncryptionEnabled  bool          `json:"encryption_enabled"`   // Enable data encryption
	AuditEnabled       bool          `json:"audit_enabled"`        // Enable audit logging
}

// NewMultiLevelCache creates a new multi-level cache manager
func NewMultiLevelCache(config CacheConfig, logger *zap.Logger) (*MultiLevelCache, error) {
	// Parse Redis URL and create client
	opts, err := redis.ParseURL(config.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	
	// Configure Redis for healthcare workloads
	opts.PoolSize = 20
	opts.MaxRetries = 3
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 2 * time.Second
	opts.WriteTimeout = 2 * time.Second
	opts.MaxIdleConns = 10
	opts.ConnMaxIdleTime = 30 * time.Minute
	
	client := redis.NewClient(opts)
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	// Initialize audit logging
	var auditLog *CacheAuditLog
	if config.AuditEnabled {
		auditLog = &CacheAuditLog{
			logger: logger.Named("cache_audit"),
		}
	}
	
	cache := &MultiLevelCache{
		l1Cache:            &sync.Map{},
		l2Client:           client,
		logger:             logger.Named("multi_level_cache"),
		stats:              &CacheStats{LastReset: time.Now()},
		l1MaxSize:          config.L1MaxSize,
		l1TTL:              config.L1TTL,
		l2TTL:              config.L2TTL,
		promotionThreshold: config.PromotionThreshold,
		demotionThreshold:  config.DemotionTimeout,
		encryptionEnabled:  config.EncryptionEnabled,
		auditLog:           auditLog,
	}
	
	// Start background maintenance
	go cache.maintenanceLoop()
	
	logger.Info("Multi-level cache initialized successfully",
		zap.String("redis_addr", opts.Addr),
		zap.Int64("l1_max_size", config.L1MaxSize),
		zap.Duration("l1_ttl", config.L1TTL),
		zap.Duration("l2_ttl", config.L2TTL),
	)
	
	return cache, nil
}

// Get retrieves a value from the cache hierarchy
func (c *MultiLevelCache) Get(ctx context.Context, key string, dest interface{}) error {
	// Try L1 cache first
	if entry, ok := c.getFromL1(key); ok {
		c.incrementStats("l1_hits")
		c.updateAccess(entry)
		return c.deserialize(entry.Value, dest)
	}
	c.incrementStats("l1_misses")
	
	// Try L2 cache (Redis)
	entry, err := c.getFromL2(ctx, key)
	if err == nil {
		c.incrementStats("l2_hits")
		c.updateAccess(entry)
		
		// Consider promoting to L1 if frequently accessed
		if entry.AccessCount >= c.promotionThreshold {
			c.promoteToL1(key, entry)
		}
		
		return c.deserialize(entry.Value, dest)
	} else if err != redis.Nil {
		c.logger.Error("L2 cache error", zap.String("key", key), zap.Error(err))
	}
	c.incrementStats("l2_misses")
	
	// Cache miss - caller should handle L3 (database) lookup
	c.auditOperation("MISS", key, "")
	return ErrCacheMiss
}

// Set stores a value in the appropriate cache level
func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration, tags ...string) error {
	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Level:      L2Cache, // Default to L2
		TTL:        ttl,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
		AccessCount: 1,
		Size:       c.estimateSize(value),
		Tags:       tags,
	}
	
	// Store in L2 (Redis) by default
	if err := c.setToL2(ctx, key, entry, ttl); err != nil {
		c.logger.Error("Failed to set L2 cache", zap.String("key", key), zap.Error(err))
		return err
	}
	
	// Also store in L1 if it's a small, frequently accessed item
	if entry.Size < 1024 { // 1KB threshold for L1
		c.setToL1(key, entry)
		entry.Level = L1Cache
	}
	
	c.auditOperation("SET", key, fmt.Sprintf("ttl=%v,level=%d", ttl, entry.Level))
	return nil
}

// Delete removes a key from all cache levels
func (c *MultiLevelCache) Delete(ctx context.Context, key string) error {
	// Remove from L1
	c.l1Cache.Delete(key)
	
	// Remove from L2
	if err := c.l2Client.Del(ctx, c.l2Key(key)).Err(); err != nil {
		c.logger.Error("Failed to delete from L2 cache", zap.String("key", key), zap.Error(err))
		return err
	}
	
	c.incrementStats("invalidations")
	c.auditOperation("DELETE", key, "")
	return nil
}

// InvalidateByTags removes all entries with matching tags
func (c *MultiLevelCache) InvalidateByTags(ctx context.Context, tags ...string) error {
	// For L2 (Redis), use pattern matching to find keys with tags
	for _, tag := range tags {
		pattern := fmt.Sprintf("*:tag:%s:*", tag)
		keys, err := c.l2Client.Keys(ctx, pattern).Result()
		if err != nil {
			c.logger.Error("Failed to find keys by tag", zap.String("tag", tag), zap.Error(err))
			continue
		}
		
		if len(keys) > 0 {
			if err := c.l2Client.Del(ctx, keys...).Err(); err != nil {
				c.logger.Error("Failed to delete tagged keys", zap.String("tag", tag), zap.Error(err))
			}
		}
	}
	
	// For L1, iterate through entries and remove matching tags
	c.l1Cache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*CacheEntry); ok {
			for _, entryTag := range entry.Tags {
				for _, targetTag := range tags {
					if entryTag == targetTag {
						c.l1Cache.Delete(key)
						return true
					}
				}
			}
		}
		return true
	})
	
	c.logger.Info("Invalidated cache entries by tags", zap.Strings("tags", tags))
	return nil
}

// GetStats returns current cache statistics
func (c *MultiLevelCache) GetStats() CacheStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()
	return *c.stats
}

// ResetStats resets cache statistics
func (c *MultiLevelCache) ResetStats() {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()
	
	c.stats = &CacheStats{LastReset: time.Now()}
}

// GetRedisClient returns the underlying Redis client for advanced operations
func (c *MultiLevelCache) GetRedisClient() *redis.Client {
	return c.l2Client
}

// Close closes the cache and cleans up resources
func (c *MultiLevelCache) Close() error {
	return c.l2Client.Close()
}

// Internal helper methods

func (c *MultiLevelCache) getFromL1(key string) (*CacheEntry, bool) {
	if value, ok := c.l1Cache.Load(key); ok {
		if entry, ok := value.(*CacheEntry); ok {
			// Check if expired
			if time.Since(entry.CreatedAt) > c.l1TTL {
				c.l1Cache.Delete(key)
				return nil, false
			}
			return entry, true
		}
	}
	return nil, false
}

func (c *MultiLevelCache) getFromL2(ctx context.Context, key string) (*CacheEntry, error) {
	data, err := c.l2Client.Get(ctx, c.l2Key(key)).Result()
	if err != nil {
		return nil, err
	}
	
	var entry CacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}
	
	return &entry, nil
}

func (c *MultiLevelCache) setToL1(key string, entry *CacheEntry) {
	// Implement LRU eviction if needed
	c.l1Cache.Store(key, entry)
}

func (c *MultiLevelCache) setToL2(ctx context.Context, key string, entry *CacheEntry, ttl time.Duration) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}
	
	// Store with tags for invalidation
	pipe := c.l2Client.Pipeline()
	pipe.Set(ctx, c.l2Key(key), data, ttl)
	
	// Add tag indices for invalidation
	for _, tag := range entry.Tags {
		tagKey := fmt.Sprintf("tag:%s:%s", tag, key)
		pipe.Set(ctx, tagKey, "1", ttl)
	}
	
	_, err = pipe.Exec(ctx)
	return err
}

func (c *MultiLevelCache) promoteToL1(key string, entry *CacheEntry) {
	c.setToL1(key, entry)
	entry.Level = L1Cache
	c.incrementStats("promotions")
	
	c.logger.Debug("Promoted entry to L1 cache",
		zap.String("key", key),
		zap.Int64("access_count", entry.AccessCount),
	)
}

func (c *MultiLevelCache) demoteFromL1(key string) {
	c.l1Cache.Delete(key)
	c.incrementStats("demotions")
	
	c.logger.Debug("Demoted entry from L1 cache", zap.String("key", key))
}

func (c *MultiLevelCache) updateAccess(entry *CacheEntry) {
	entry.AccessedAt = time.Now()
	entry.AccessCount++
}

func (c *MultiLevelCache) l2Key(key string) string {
	return fmt.Sprintf("med_cache:%s", key)
}

func (c *MultiLevelCache) estimateSize(value interface{}) int64 {
	data, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return int64(len(data))
}

func (c *MultiLevelCache) deserialize(value interface{}, dest interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal to destination: %w", err)
	}
	
	return nil
}

func (c *MultiLevelCache) incrementStats(metric string) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()
	
	switch metric {
	case "l1_hits":
		c.stats.L1Hits++
	case "l1_misses":
		c.stats.L1Misses++
	case "l2_hits":
		c.stats.L2Hits++
	case "l2_misses":
		c.stats.L2Misses++
	case "l3_hits":
		c.stats.L3Hits++
	case "l3_misses":
		c.stats.L3Misses++
	case "promotions":
		c.stats.Promotions++
	case "demotions":
		c.stats.Demotions++
	case "invalidations":
		c.stats.Invalidations++
	}
}

func (c *MultiLevelCache) maintenanceLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.performMaintenance()
	}
}

func (c *MultiLevelCache) performMaintenance() {
	// Clean expired entries from L1
	var expiredKeys []interface{}
	
	c.l1Cache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(*CacheEntry); ok {
			if time.Since(entry.AccessedAt) > c.demotionThreshold {
				expiredKeys = append(expiredKeys, key)
			}
		}
		return true
	})
	
	// Remove expired entries
	for _, key := range expiredKeys {
		c.demoteFromL1(key.(string))
	}
	
	if len(expiredKeys) > 0 {
		c.logger.Debug("Cleaned expired L1 cache entries", zap.Int("count", len(expiredKeys)))
	}
}

func (c *MultiLevelCache) auditOperation(operation, key, details string) {
	if c.auditLog != nil {
		c.auditLog.Log(operation, key, details)
	}
}

// CacheAuditLog methods
func (al *CacheAuditLog) Log(operation, key, details string) {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	
	al.logger.Info("Cache operation",
		zap.String("operation", operation),
		zap.String("key", key),
		zap.String("details", details),
		zap.Time("timestamp", time.Now()),
	)
}

// Custom errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
	ErrCacheTimeout = fmt.Errorf("cache operation timeout")
)