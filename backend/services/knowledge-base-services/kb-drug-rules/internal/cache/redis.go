package cache

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Client defines the cache client interface
type Client interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	InvalidatePattern(pattern string) error
	Ping() error
	Close() error
	GetStats() (map[string]interface{}, error)
}

// RedisClient implements the cache client using Redis
type RedisClient struct {
	client *redis.Client
	logger *logrus.Logger
	ctx    context.Context
}

// NewRedisClient creates a new Redis cache client
func NewRedisClient(redisURL string) (Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opts)
	
	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	return &RedisClient{
		client: client,
		logger: logger,
		ctx:    ctx,
	}, nil
}

// Get retrieves a value from cache
func (r *RedisClient) Get(key string) ([]byte, error) {
	start := time.Now()
	defer func() {
		r.logger.WithFields(logrus.Fields{
			"operation": "cache_get",
			"key":       key,
			"duration":  time.Since(start),
		}).Debug("Cache get operation")
	}()

	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Key not found
		}
		return nil, fmt.Errorf("failed to get key %s: %w", key, err)
	}

	return []byte(val), nil
}

// Set stores a value in cache with TTL
func (r *RedisClient) Set(key string, value []byte, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		r.logger.WithFields(logrus.Fields{
			"operation": "cache_set",
			"key":       key,
			"ttl":       ttl,
			"size":      len(value),
			"duration":  time.Since(start),
		}).Debug("Cache set operation")
	}()

	err := r.client.Set(r.ctx, key, value, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set key %s: %w", key, err)
	}

	return nil
}

// Delete removes a value from cache
func (r *RedisClient) Delete(key string) error {
	start := time.Now()
	defer func() {
		r.logger.WithFields(logrus.Fields{
			"operation": "cache_delete",
			"key":       key,
			"duration":  time.Since(start),
		}).Debug("Cache delete operation")
	}()

	err := r.client.Del(r.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}

	return nil
}

// InvalidatePattern invalidates all keys matching a pattern
func (r *RedisClient) InvalidatePattern(pattern string) error {
	start := time.Now()
	defer func() {
		r.logger.WithFields(logrus.Fields{
			"operation": "cache_invalidate_pattern",
			"pattern":   pattern,
			"duration":  time.Since(start),
		}).Debug("Cache pattern invalidation")
	}()

	// Use SCAN to find matching keys
	var cursor uint64
	var keys []string
	
	for {
		var scanKeys []string
		var err error
		
		scanKeys, cursor, err = r.client.Scan(r.ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan keys with pattern %s: %w", pattern, err)
		}
		
		keys = append(keys, scanKeys...)
		
		if cursor == 0 {
			break
		}
	}

	// Delete found keys in batches
	if len(keys) > 0 {
		batchSize := 100
		for i := 0; i < len(keys); i += batchSize {
			end := i + batchSize
			if end > len(keys) {
				end = len(keys)
			}
			
			batch := keys[i:end]
			if err := r.client.Del(r.ctx, batch...).Err(); err != nil {
				return fmt.Errorf("failed to delete batch of keys: %w", err)
			}
		}
		
		r.logger.WithFields(logrus.Fields{
			"pattern":     pattern,
			"keys_deleted": len(keys),
		}).Info("Cache pattern invalidation completed")
	}

	return nil
}

// Ping checks if cache is available
func (r *RedisClient) Ping() error {
	return r.client.Ping(r.ctx).Err()
}

// Close closes the cache connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// GetStats returns cache statistics
func (r *RedisClient) GetStats() (map[string]interface{}, error) {
	info, err := r.client.Info(r.ctx, "stats", "memory", "clients").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	stats := make(map[string]interface{})
	
	// Parse Redis INFO output
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				stats[key] = value
			}
		}
	}

	// Add computed stats
	if keyspaceHits, ok := stats["keyspace_hits"]; ok {
		if keyspaceMisses, ok := stats["keyspace_misses"]; ok {
			hits := parseIntOrZero(keyspaceHits)
			misses := parseIntOrZero(keyspaceMisses)
			total := hits + misses
			
			stats["hit_rate"] = 0.0
			if total > 0 {
				stats["hit_rate"] = float64(hits) / float64(total)
			}
		}
	}

	return stats, nil
}

// Helper function to parse integer or return zero
func parseIntOrZero(value interface{}) int64 {
	if str, ok := value.(string); ok {
		if val, err := strconv.ParseInt(str, 10, 64); err == nil {
			return val
		}
	}
	return 0
}

// LocalCache implements a simple in-memory cache for testing
type LocalCache struct {
	data   map[string]cacheItem
	mutex  sync.RWMutex
	logger *logrus.Logger
}

type cacheItem struct {
	value     []byte
	expiresAt time.Time
}

// NewLocalCache creates a new local in-memory cache
func NewLocalCache() Client {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	cache := &LocalCache{
		data:   make(map[string]cacheItem),
		logger: logger,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from local cache
func (l *LocalCache) Get(key string) ([]byte, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	item, exists := l.data[key]
	if !exists {
		return nil, nil
	}

	if time.Now().After(item.expiresAt) {
		// Item expired, remove it
		delete(l.data, key)
		return nil, nil
	}

	return item.value, nil
}

// Set stores a value in local cache with TTL
func (l *LocalCache) Set(key string, value []byte, ttl time.Duration) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.data[key] = cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
}

// Delete removes a value from local cache
func (l *LocalCache) Delete(key string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	delete(l.data, key)
	return nil
}

// InvalidatePattern invalidates all keys matching a pattern in local cache
func (l *LocalCache) InvalidatePattern(pattern string) error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Simple pattern matching - in production, use more sophisticated matching
	for key := range l.data {
		if strings.Contains(key, strings.Replace(pattern, "*", "", -1)) {
			delete(l.data, key)
		}
	}

	return nil
}

// Ping always returns nil for local cache
func (l *LocalCache) Ping() error {
	return nil
}

// Close is a no-op for local cache
func (l *LocalCache) Close() error {
	return nil
}

// GetStats returns local cache statistics
func (l *LocalCache) GetStats() (map[string]interface{}, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	return map[string]interface{}{
		"type":  "local",
		"size":  len(l.data),
		"items": len(l.data),
	}, nil
}

// cleanup removes expired items from local cache
func (l *LocalCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mutex.Lock()
		now := time.Now()
		for key, item := range l.data {
			if now.After(item.expiresAt) {
				delete(l.data, key)
			}
		}
		l.mutex.Unlock()
	}
}
