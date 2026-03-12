package services

import (
	"context"
	"fmt"
	"time"

	"flow2-go-engine/internal/config"

	"github.com/redis/go-redis/v9"
)

// redisCacheService implements CacheService using Redis
type redisCacheService struct {
	client *redis.Client
}

// NewCacheService creates a new cache service
func NewCacheService(cfg config.RedisConfig) (CacheService, error) {
	// Only real Redis - no mocks or fallbacks
	if cfg.Address == "" {
		return nil, fmt.Errorf("Redis address is required - no fallback available")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Test connection - fail if Redis is not available
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", cfg.Address, err)
	}

	return &redisCacheService{
		client: client,
	}, nil
}

// Get retrieves a value from cache
func (r *redisCacheService) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// Set stores a value in cache with TTL
func (r *redisCacheService) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes a value from cache
func (r *redisCacheService) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in cache
func (r *redisCacheService) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, key).Result()
	return result > 0, err
}

// Close closes the cache connection
func (r *redisCacheService) Close() error {
	return r.client.Close()
}
