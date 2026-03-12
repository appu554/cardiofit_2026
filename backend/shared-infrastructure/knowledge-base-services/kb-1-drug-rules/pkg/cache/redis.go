package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// Config holds Redis connection configuration
type Config struct {
	Host        string
	Port        int
	Password    string
	DB          int
	MaxRetries  int
	PoolSize    int
	DialTimeout time.Duration
	ReadTimeout time.Duration
}

// DefaultConfig returns default Redis configuration
func DefaultConfig() Config {
	return Config{
		Host:        "localhost",
		Port:        6380,
		Password:    "",
		DB:          0,
		MaxRetries:  3,
		PoolSize:    10,
		DialTimeout: 5 * time.Second,
		ReadTimeout: 3 * time.Second,
	}
}

// RedisCache implements caching using Redis
type RedisCache struct {
	client *redis.Client
	log    *logrus.Entry
	prefix string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(cfg Config, log *logrus.Entry) (*RedisCache, error) {
	logger := log.WithField("component", "redis-cache")

	client := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:    cfg.Password,
		DB:          cfg.DB,
		MaxRetries:  cfg.MaxRetries,
		PoolSize:    cfg.PoolSize,
		DialTimeout: cfg.DialTimeout,
		ReadTimeout: cfg.ReadTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"host": cfg.Host,
		"port": cfg.Port,
		"db":   cfg.DB,
	}).Info("Connected to Redis")

	return &RedisCache{
		client: client,
		log:    logger,
		prefix: "kb1:",
	}, nil
}

// NewRedisCacheFromURL creates a Redis cache from connection URL
func NewRedisCacheFromURL(url string, log *logrus.Entry) (*RedisCache, error) {
	logger := log.WithField("component", "redis-cache")

	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis via URL")

	return &RedisCache{
		client: client,
		log:    logger,
		prefix: "kb1:",
	}, nil
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	fullKey := c.prefix + key

	result, err := c.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		c.log.WithError(err).WithField("key", fullKey).Warn("Redis GET error")
		return nil, err
	}

	return result, nil
}

// Set stores a value in cache with TTL
func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	fullKey := c.prefix + key

	err := c.client.Set(ctx, fullKey, value, ttl).Err()
	if err != nil {
		c.log.WithError(err).WithField("key", fullKey).Warn("Redis SET error")
		return err
	}

	return nil
}

// Delete removes a value from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key

	err := c.client.Del(ctx, fullKey).Err()
	if err != nil {
		c.log.WithError(err).WithField("key", fullKey).Warn("Redis DEL error")
		return err
	}

	return nil
}

// DeletePattern removes all keys matching a pattern
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := c.prefix + pattern

	iter := c.client.Scan(ctx, 0, fullPattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
		c.log.WithField("count", len(keys)).Info("Deleted keys matching pattern")
	}

	return nil
}

// Exists checks if a key exists in cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.prefix + key

	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// SetWithTags stores a value with associated tags for grouped invalidation
func (c *RedisCache) SetWithTags(ctx context.Context, key string, value []byte, ttl time.Duration, tags []string) error {
	fullKey := c.prefix + key

	pipe := c.client.Pipeline()
	pipe.Set(ctx, fullKey, value, ttl)

	// Add key to tag sets
	for _, tag := range tags {
		tagKey := c.prefix + "tag:" + tag
		pipe.SAdd(ctx, tagKey, fullKey)
		pipe.Expire(ctx, tagKey, 24*time.Hour) // Tags expire in 24 hours
	}

	_, err := pipe.Exec(ctx)
	return err
}

// InvalidateTag removes all cached values associated with a tag
func (c *RedisCache) InvalidateTag(ctx context.Context, tag string) error {
	tagKey := c.prefix + "tag:" + tag

	// Get all keys in the tag set
	keys, err := c.client.SMembers(ctx, tagKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get tag members: %w", err)
	}

	if len(keys) > 0 {
		// Delete all cached values and the tag set
		allKeys := append(keys, tagKey)
		if err := c.client.Del(ctx, allKeys...).Err(); err != nil {
			return fmt.Errorf("failed to delete tag keys: %w", err)
		}
		c.log.WithFields(logrus.Fields{
			"tag":   tag,
			"count": len(keys),
		}).Info("Invalidated cache tag")
	}

	return nil
}

// Health checks Redis connection health
func (c *RedisCache) Health(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Stats returns cache statistics
func (c *RedisCache) Stats(ctx context.Context) (*CacheStats, error) {
	info, err := c.client.Info(ctx, "stats", "memory").Result()
	if err != nil {
		return nil, err
	}

	// Count keys with our prefix
	var keyCount int64
	iter := c.client.Scan(ctx, 0, c.prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		keyCount++
	}

	return &CacheStats{
		KeyCount: keyCount,
		Info:     info,
	}, nil
}

// CacheStats contains cache statistics
type CacheStats struct {
	KeyCount int64  `json:"key_count"`
	Info     string `json:"info"`
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	c.log.Info("Closing Redis connection")
	return c.client.Close()
}

// FlushAll removes all keys with our prefix (use with caution)
func (c *RedisCache) FlushAll(ctx context.Context) error {
	return c.DeletePattern(ctx, "*")
}

// =============================================================================
// SPECIALIZED CACHE OPERATIONS FOR KB-1
// =============================================================================

// GetDrugRule gets a cached drug rule
func (c *RedisCache) GetDrugRule(ctx context.Context, rxnormCode, jurisdiction string) ([]byte, error) {
	key := fmt.Sprintf("drug_rule:%s:%s", jurisdiction, rxnormCode)
	return c.Get(ctx, key)
}

// SetDrugRule caches a drug rule
func (c *RedisCache) SetDrugRule(ctx context.Context, rxnormCode, jurisdiction string, data []byte, ttl time.Duration) error {
	key := fmt.Sprintf("drug_rule:%s:%s", jurisdiction, rxnormCode)
	tags := []string{
		fmt.Sprintf("jurisdiction:%s", jurisdiction),
		fmt.Sprintf("rxnorm:%s", rxnormCode),
	}
	return c.SetWithTags(ctx, key, data, ttl, tags)
}

// InvalidateDrugRule invalidates a cached drug rule
func (c *RedisCache) InvalidateDrugRule(ctx context.Context, rxnormCode, jurisdiction string) error {
	key := fmt.Sprintf("drug_rule:%s:%s", jurisdiction, rxnormCode)
	return c.Delete(ctx, key)
}

// InvalidateJurisdiction invalidates all cached rules for a jurisdiction
func (c *RedisCache) InvalidateJurisdiction(ctx context.Context, jurisdiction string) error {
	return c.InvalidateTag(ctx, fmt.Sprintf("jurisdiction:%s", jurisdiction))
}

// =============================================================================
// NO-OP CACHE FOR TESTING
// =============================================================================

// NoOpCache implements Cache interface but does nothing (for testing or when cache is disabled)
type NoOpCache struct{}

// NewNoOpCache creates a no-op cache
func NewNoOpCache() *NoOpCache {
	return &NoOpCache{}
}

// Get always returns nil (cache miss)
func (c *NoOpCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, nil
}

// Set does nothing
func (c *NoOpCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

// Delete does nothing
func (c *NoOpCache) Delete(ctx context.Context, key string) error {
	return nil
}
