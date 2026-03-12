// Package cache provides caching functionality for KB-14 Care Navigator
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/config"
)

// RedisCache provides Redis-based caching
type RedisCache struct {
	client *redis.Client
	prefix string
	log    *logrus.Entry
}

// NewRedisCache creates a new Redis cache client
func NewRedisCache(cfg config.RedisConfig) (*RedisCache, error) {
	opt, err := redis.ParseURL(cfg.URL)
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

	return &RedisCache{
		client: client,
		prefix: cfg.Prefix,
		log:    logrus.WithField("component", "redis-cache"),
	}, nil
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Ping checks if Redis is reachable
func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// key generates a prefixed cache key
func (c *RedisCache) key(parts ...string) string {
	key := c.prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, c.key(key)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return err
	}

	return json.Unmarshal(data, dest)
}

// Set stores a value in cache with TTL
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, c.key(key), data, ttl).Err()
}

// Delete removes a value from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, c.key(key)).Err()
}

// DeletePattern removes all keys matching a pattern
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, c.key(pattern), 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Exists checks if a key exists in cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, c.key(key)).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// TTL returns the remaining TTL for a key
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.key(key)).Result()
}

// Increment atomically increments a counter
func (c *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, c.key(key)).Result()
}

// IncrementBy atomically increments a counter by a specific amount
func (c *RedisCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, c.key(key), value).Result()
}

// SetNX sets a value only if the key doesn't exist (for distributed locking)
func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, err
	}
	return c.client.SetNX(ctx, c.key(key), data, ttl).Result()
}

// Task-specific caching methods

// CacheTask caches a task
func (c *RedisCache) CacheTask(ctx context.Context, taskID string, task interface{}) error {
	return c.Set(ctx, "task:"+taskID, task, 5*time.Minute)
}

// GetCachedTask retrieves a cached task
func (c *RedisCache) GetCachedTask(ctx context.Context, taskID string, dest interface{}) error {
	return c.Get(ctx, "task:"+taskID, dest)
}

// InvalidateTask removes a task from cache
func (c *RedisCache) InvalidateTask(ctx context.Context, taskID string) error {
	return c.Delete(ctx, "task:"+taskID)
}

// CacheWorklist caches a worklist result
func (c *RedisCache) CacheWorklist(ctx context.Context, cacheKey string, worklist interface{}) error {
	return c.Set(ctx, "worklist:"+cacheKey, worklist, 1*time.Minute)
}

// GetCachedWorklist retrieves a cached worklist
func (c *RedisCache) GetCachedWorklist(ctx context.Context, cacheKey string, dest interface{}) error {
	return c.Get(ctx, "worklist:"+cacheKey, dest)
}

// InvalidateWorklists invalidates all worklist caches
func (c *RedisCache) InvalidateWorklists(ctx context.Context) error {
	return c.DeletePattern(ctx, "worklist:*")
}

// CacheDashboardMetrics caches dashboard metrics
func (c *RedisCache) CacheDashboardMetrics(ctx context.Context, metrics interface{}) error {
	return c.Set(ctx, "dashboard:metrics", metrics, 30*time.Second)
}

// GetCachedDashboardMetrics retrieves cached dashboard metrics
func (c *RedisCache) GetCachedDashboardMetrics(ctx context.Context, dest interface{}) error {
	return c.Get(ctx, "dashboard:metrics", dest)
}

// Distributed locking for escalation processing

// AcquireEscalationLock tries to acquire a lock for escalation processing
func (c *RedisCache) AcquireEscalationLock(ctx context.Context, taskID string) (bool, error) {
	return c.SetNX(ctx, "lock:escalation:"+taskID, time.Now().Unix(), 5*time.Minute)
}

// ReleaseEscalationLock releases an escalation lock
func (c *RedisCache) ReleaseEscalationLock(ctx context.Context, taskID string) error {
	return c.Delete(ctx, "lock:escalation:"+taskID)
}

// Rate limiting support

// IncrementRateLimit increments a rate limit counter
func (c *RedisCache) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	fullKey := c.key("ratelimit:" + key)

	pipe := c.client.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// CheckRateLimit checks if a rate limit has been exceeded
func (c *RedisCache) CheckRateLimit(ctx context.Context, key string, limit int64) (bool, int64, error) {
	count, err := c.client.Get(ctx, c.key("ratelimit:"+key)).Int64()
	if err == redis.Nil {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return count >= limit, count, nil
}

// ErrCacheMiss is returned when a cache key is not found
var ErrCacheMiss = fmt.Errorf("cache miss")
