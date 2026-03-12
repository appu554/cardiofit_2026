// Package cache provides caching implementations for data sources.
// The primary implementation is Redis-based, with an in-memory fallback
// for development and testing.
package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/shared/datasources"
)

// =============================================================================
// REDIS CACHE IMPLEMENTATION
// =============================================================================

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	// Connection
	Host     string
	Port     int
	Password string
	DB       int

	// Pool settings
	PoolSize     int
	MinIdleConns int

	// Timeouts
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Key prefix for namespacing
	KeyPrefix string

	// Logger
	Logger *logrus.Entry
}

// DefaultRedisConfig returns sensible defaults
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		Host:         "localhost",
		Port:         6380, // KB services use port 6380
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		KeyPrefix:    "kb:",
	}
}

// RedisCache implements the Cache interface using Redis
type RedisCache struct {
	client    *redis.Client
	config    RedisConfig
	log       *logrus.Entry
	keyPrefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log := config.Logger
	if log == nil {
		log = logrus.NewEntry(logrus.StandardLogger())
	}

	return &RedisCache{
		client:    client,
		config:    config,
		log:       log.WithField("component", "redis-cache"),
		keyPrefix: config.KeyPrefix,
	}, nil
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.keyPrefix + key

	val, err := c.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss, not an error
	}
	if err != nil {
		c.log.WithError(err).WithField("key", key).Debug("Cache get error")
		return nil, err
	}

	return val, nil
}

// Set stores a value in cache with TTL
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := c.keyPrefix + key

	err := c.client.Set(ctx, fullKey, value, ttl).Err()
	if err != nil {
		c.log.WithError(err).WithField("key", key).Debug("Cache set error")
		return err
	}

	return nil
}

// Delete removes a value from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.keyPrefix + key
	return c.client.Del(ctx, fullKey).Err()
}

// Exists checks if a key exists
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.keyPrefix + key
	count, err := c.client.Exists(ctx, fullKey).Result()
	return count > 0, err
}

// Clear removes all cached values with the prefix
func (c *RedisCache) Clear(ctx context.Context) error {
	pattern := c.keyPrefix + "*"

	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// =============================================================================
// ADDITIONAL REDIS OPERATIONS
// =============================================================================

// SetNX sets a value only if it doesn't exist (for distributed locking)
func (c *RedisCache) SetNX(ctx context.Context, key string, value []byte, ttl time.Duration) (bool, error) {
	fullKey := c.keyPrefix + key
	return c.client.SetNX(ctx, fullKey, value, ttl).Result()
}

// GetMulti retrieves multiple values at once
func (c *RedisCache) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.keyPrefix + key
	}

	vals, err := c.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte)
	for i, val := range vals {
		if val != nil {
			if strVal, ok := val.(string); ok {
				result[keys[i]] = []byte(strVal)
			}
		}
	}

	return result, nil
}

// SetMulti stores multiple values at once
func (c *RedisCache) SetMulti(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	pipe := c.client.Pipeline()

	for key, value := range items {
		fullKey := c.keyPrefix + key
		pipe.Set(ctx, fullKey, value, ttl)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// Increment atomically increments a counter
func (c *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	fullKey := c.keyPrefix + key
	return c.client.Incr(ctx, fullKey).Result()
}

// GetStats returns cache statistics
func (c *RedisCache) GetStats(ctx context.Context) (*CacheStats, error) {
	info, err := c.client.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return nil, err
	}

	// Parse basic stats from info string
	stats := &CacheStats{
		Info: info,
	}

	// Get key count for our prefix
	pattern := c.keyPrefix + "*"
	var cursor uint64
	var count int64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, err
		}
		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	stats.KeyCount = count

	return stats, nil
}

// CacheStats contains cache statistics
type CacheStats struct {
	KeyCount int64  `json:"keyCount"`
	Info     string `json:"info"`
}

// =============================================================================
// IN-MEMORY CACHE IMPLEMENTATION (for testing/development)
// =============================================================================

// MemoryCache implements the Cache interface using in-memory storage
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]*memoryCacheItem
}

type memoryCacheItem struct {
	value      []byte
	expiration time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*memoryCacheItem),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

func (c *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, nil
	}

	if time.Now().After(item.expiration) {
		return nil, nil
	}

	return item.value, nil
}

func (c *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &memoryCacheItem{
		value:      value,
		expiration: time.Now().Add(ttl),
	}

	return nil
}

func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}

func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return false, nil
	}

	return time.Now().Before(item.expiration), nil
}

func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*memoryCacheItem)
	return nil
}

func (c *MemoryCache) Close() error {
	return c.Clear(context.Background())
}

func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// =============================================================================
// CACHE FACTORY
// =============================================================================

// CacheType identifies the cache implementation
type CacheType string

const (
	CacheTypeRedis  CacheType = "redis"
	CacheTypeMemory CacheType = "memory"
)

// CacheFactory creates cache instances
func CacheFactory(cacheType CacheType, config interface{}) (datasources.Cache, error) {
	switch cacheType {
	case CacheTypeRedis:
		redisConfig, ok := config.(RedisConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config for Redis cache")
		}
		return NewRedisCache(redisConfig)
	case CacheTypeMemory:
		return NewMemoryCache(), nil
	default:
		return nil, fmt.Errorf("unknown cache type: %s", cacheType)
	}
}
