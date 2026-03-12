package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"kb-patient-profile/internal/config"
)

const (
	PatientProfilePrefix = "kb20:patient:"
	StratumPrefix        = "kb20:stratum:"
	ModifierPrefix       = "kb20:modifier:"
	ADRProfilePrefix     = "kb20:adr:"

	DefaultProfileTTL  = 15 * time.Minute
	DefaultStratumTTL  = 5 * time.Minute
	DefaultModifierTTL = 30 * time.Minute
	DefaultADRTTL      = 1 * time.Hour
)

// Client wraps a Redis connection for KB-20 caching.
type Client struct {
	rdb *redis.Client
	ctx context.Context
}

// NewClient creates a Redis client from configuration.
func NewClient(cfg *config.Config) (*Client, error) {
	opts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if cfg.Redis.Password != "" {
		opts.Password = cfg.Redis.Password
	}
	if cfg.Redis.DB > 0 {
		opts.DB = cfg.Redis.DB
	}

	rdb := redis.NewClient(opts)
	ctx := context.Background()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{rdb: rdb, ctx: ctx}, nil
}

// Get retrieves a cached value and unmarshals it into dest.
func (c *Client) Get(key string, dest interface{}) error {
	val, err := c.rdb.Get(c.ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

// Set marshals value and stores it with the given TTL.
func (c *Client) Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(c.ctx, key, data, ttl).Err()
}

// Delete removes a cached key.
func (c *Client) Delete(key string) error {
	return c.rdb.Del(c.ctx, key).Err()
}

// DeletePattern removes all keys matching a glob pattern.
func (c *Client) DeletePattern(pattern string) error {
	iter := c.rdb.Scan(c.ctx, 0, pattern, 100).Iterator()
	for iter.Next(c.ctx) {
		c.rdb.Del(c.ctx, iter.Val())
	}
	return iter.Err()
}

// HealthCheck pings Redis.
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()
	return c.rdb.Ping(ctx).Err()
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
