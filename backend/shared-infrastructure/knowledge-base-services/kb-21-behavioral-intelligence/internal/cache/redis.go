package cache

import (
	"context"
	"fmt"
	"time"

	"kb-21-behavioral-intelligence/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisClient wraps go-redis with convenience methods for KB-21 caching.
// Used for: engagement profile caching, adherence score caching, loop trust caching.
type RedisClient struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisClient(cfg *config.Config, logger *zap.Logger) (*RedisClient, error) {
	opts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if cfg.Redis.Password != "" {
		opts.Password = cfg.Redis.Password
	}
	opts.DB = cfg.Redis.DB

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed — operating without cache", zap.Error(err))
		return nil, nil
	}

	logger.Info("Redis cache connected", zap.Int("db", cfg.Redis.DB))
	return &RedisClient{client: client, logger: logger}, nil
}

func (r *RedisClient) Ping() error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return r.client.Ping(ctx).Err()
}

func (r *RedisClient) Get(key string) (string, error) {
	if r == nil || r.client == nil {
		return "", fmt.Errorf("no cache")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisClient) Delete(key string) error {
	if r == nil || r.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}
