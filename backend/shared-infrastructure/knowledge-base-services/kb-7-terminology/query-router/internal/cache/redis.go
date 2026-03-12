package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// RedisClient wraps the Redis client with caching functionality
type RedisClient struct {
	client *redis.Client
	logger *logrus.Logger
}

// NewRedisClient creates a new Redis client
func NewRedisClient(redisURL string) (*RedisClient, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &RedisClient{
		client: client,
		logger: logger,
	}, nil
}

// Get retrieves a value from cache
func (r *RedisClient) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	// Try to unmarshal as JSON
	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		// If JSON unmarshal fails, return as string
		return val, nil
	}

	return result, nil
}

// Set stores a value in cache with TTL
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Marshal value to JSON
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	if err := r.client.Set(ctx, key, jsonValue, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	r.logger.WithFields(logrus.Fields{
		"key": key,
		"ttl": ttl,
	}).Debug("Cached value")

	return nil
}

// Delete removes a key from cache
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Exists checks if a key exists in cache
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}
	return result > 0, nil
}

// SetNX sets a key only if it doesn't exist
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value for key %s: %w", key, err)
	}

	result, err := r.client.SetNX(ctx, key, jsonValue, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to setnx key %s: %w", key, err)
	}

	return result, nil
}

// Increment increments a numeric value
func (r *RedisClient) Increment(ctx context.Context, key string) (int64, error) {
	result, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment key %s: %w", key, err)
	}
	return result, nil
}

// GetMultiple retrieves multiple keys at once
func (r *RedisClient) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	results, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple keys: %w", err)
	}

	response := make(map[string]interface{})
	for i, key := range keys {
		if results[i] != nil {
			val := results[i].(string)
			var jsonResult interface{}
			if err := json.Unmarshal([]byte(val), &jsonResult); err != nil {
				// If JSON unmarshal fails, store as string
				response[key] = val
			} else {
				response[key] = jsonResult
			}
		}
	}

	return response, nil
}

// SetMultiple sets multiple key-value pairs
func (r *RedisClient) SetMultiple(ctx context.Context, pairs map[string]interface{}, ttl time.Duration) error {
	pipe := r.client.Pipeline()

	for key, value := range pairs {
		jsonValue, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, jsonValue, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set multiple keys: %w", err)
	}

	return nil
}

// GetStats returns cache statistics
func (r *RedisClient) GetStats(ctx context.Context) (map[string]string, error) {
	info, err := r.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	stats := make(map[string]string)
	stats["info"] = info

	// Get memory info
	memoryInfo, err := r.client.Info(ctx, "memory").Result()
	if err == nil {
		stats["memory"] = memoryInfo
	}

	// Get keyspace info
	keyspaceInfo, err := r.client.Info(ctx, "keyspace").Result()
	if err == nil {
		stats["keyspace"] = keyspaceInfo
	}

	return stats, nil
}

// FlushAll clears all keys from the cache (use with caution)
func (r *RedisClient) FlushAll(ctx context.Context) error {
	if err := r.client.FlushAll(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush all keys: %w", err)
	}

	r.logger.Warn("All Redis keys have been flushed")
	return nil
}

// Keys returns all keys matching a pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys with pattern %s: %w", pattern, err)
	}
	return keys, nil
}

// TTL returns the time to live for a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}
	return ttl, nil
}

// Ping tests the connection to Redis
func (r *RedisClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis ping failed: %w", err)
	}
	return nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}
	return nil
}

// GetCacheKeyPattern generates consistent cache key patterns
func GetCacheKeyPattern(prefix, suffix string) string {
	return fmt.Sprintf("%s:*:%s", prefix, suffix)
}

// InvalidatePattern invalidates all keys matching a pattern
func (r *RedisClient) InvalidatePattern(ctx context.Context, pattern string) error {
	keys, err := r.Keys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return nil
	}

	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete keys matching pattern %s: %w", pattern, err)
	}

	r.logger.WithFields(logrus.Fields{
		"pattern": pattern,
		"count":   len(keys),
	}).Info("Invalidated cache keys")

	return nil
}