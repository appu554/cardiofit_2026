// Package config provides Redis cache configuration for Knowledge Base services.
// This implements the tiered caching strategy from the KB1 Implementation Plan.
package config

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds the Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	Database int

	// Connection pool settings
	PoolSize     int
	MinIdleConns int
	PoolTimeout  time.Duration

	// Timeouts
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Retry settings
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

// TieredCacheConfig implements the HOT/WARM caching strategy
// from the KB1 Implementation Plan for DDI lookups
type TieredCacheConfig struct {
	// HOT cache - for frequently accessed data (ONC ~1,200 pairs)
	HotCache *RedisConfig

	// WARM cache - for less frequently accessed data (OHDSI ~200K pairs)
	WarmCache *RedisConfig

	// Cache TTLs by severity
	TTLContraindicated time.Duration // Longest TTL - critical data
	TTLMajor           time.Duration
	TTLModerate        time.Duration
	TTLMinor           time.Duration
	TTLDefault         time.Duration
}

// DefaultRedisConfig returns the default Redis configuration
// Uses environment variables with sensible defaults matching docker-compose.phase1.yml
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
		Port:     getEnvIntOrDefault("REDIS_PORT", 6380),
		Password: getEnvOrDefault("REDIS_PASSWORD", ""),
		Database: getEnvIntOrDefault("REDIS_DB", 0),

		// Connection pool
		PoolSize:     getEnvIntOrDefault("REDIS_POOL_SIZE", 10),
		MinIdleConns: getEnvIntOrDefault("REDIS_MIN_IDLE_CONNS", 3),
		PoolTimeout:  getDurationOrDefault("REDIS_POOL_TIMEOUT", 4*time.Second),

		// Timeouts
		DialTimeout:  getDurationOrDefault("REDIS_DIAL_TIMEOUT", 5*time.Second),
		ReadTimeout:  getDurationOrDefault("REDIS_READ_TIMEOUT", 3*time.Second),
		WriteTimeout: getDurationOrDefault("REDIS_WRITE_TIMEOUT", 3*time.Second),

		// Retries
		MaxRetries:      getEnvIntOrDefault("REDIS_MAX_RETRIES", 3),
		MinRetryBackoff: getDurationOrDefault("REDIS_MIN_RETRY_BACKOFF", 100*time.Millisecond),
		MaxRetryBackoff: getDurationOrDefault("REDIS_MAX_RETRY_BACKOFF", 500*time.Millisecond),
	}
}

// DefaultTieredCacheConfig returns the default tiered cache configuration
func DefaultTieredCacheConfig() *TieredCacheConfig {
	hotConfig := DefaultRedisConfig()
	hotConfig.Database = 0

	warmConfig := DefaultRedisConfig()
	warmConfig.Database = 1

	return &TieredCacheConfig{
		HotCache:  hotConfig,
		WarmCache: warmConfig,

		// TTLs by severity (higher severity = longer cache)
		TTLContraindicated: getDurationOrDefault("CACHE_TTL_CONTRAINDICATED", 24*time.Hour),
		TTLMajor:           getDurationOrDefault("CACHE_TTL_MAJOR", 12*time.Hour),
		TTLModerate:        getDurationOrDefault("CACHE_TTL_MODERATE", 6*time.Hour),
		TTLMinor:           getDurationOrDefault("CACHE_TTL_MINOR", 2*time.Hour),
		TTLDefault:         getDurationOrDefault("CACHE_TTL_DEFAULT", 1*time.Hour),
	}
}

// Address returns the Redis address string
func (c *RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Options returns the go-redis Options struct
func (c *RedisConfig) Options() *redis.Options {
	return &redis.Options{
		Addr:     c.Address(),
		Password: c.Password,
		DB:       c.Database,

		PoolSize:     c.PoolSize,
		MinIdleConns: c.MinIdleConns,
		PoolTimeout:  c.PoolTimeout,

		DialTimeout:  c.DialTimeout,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimeout,

		MaxRetries:      c.MaxRetries,
		MinRetryBackoff: c.MinRetryBackoff,
		MaxRetryBackoff: c.MaxRetryBackoff,
	}
}

// Connect creates a new Redis client
func (c *RedisConfig) Connect() *redis.Client {
	return redis.NewClient(c.Options())
}

// ConnectWithPing creates a client and verifies connectivity
func (c *RedisConfig) ConnectWithPing(ctx context.Context) (*redis.Client, error) {
	client := c.Connect()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis at %s: %w", c.Address(), err)
	}

	return client, nil
}

// TieredCache implements the HOT/WARM caching strategy
type TieredCache struct {
	HotClient  *redis.Client
	WarmClient *redis.Client
	Config     *TieredCacheConfig
}

// NewTieredCache creates a new tiered cache instance
func NewTieredCache(ctx context.Context, config *TieredCacheConfig) (*TieredCache, error) {
	hotClient, err := config.HotCache.ConnectWithPing(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to HOT cache: %w", err)
	}

	warmClient, err := config.WarmCache.ConnectWithPing(ctx)
	if err != nil {
		hotClient.Close()
		return nil, fmt.Errorf("failed to connect to WARM cache: %w", err)
	}

	return &TieredCache{
		HotClient:  hotClient,
		WarmClient: warmClient,
		Config:     config,
	}, nil
}

// GetTTLBySeverity returns the appropriate TTL based on interaction severity
func (c *TieredCacheConfig) GetTTLBySeverity(severity string) time.Duration {
	switch severity {
	case "CONTRAINDICATED", "contraindicated":
		return c.TTLContraindicated
	case "MAJOR", "major", "HIGH", "high":
		return c.TTLMajor
	case "MODERATE", "moderate":
		return c.TTLModerate
	case "MINOR", "minor", "LOW", "low":
		return c.TTLMinor
	default:
		return c.TTLDefault
	}
}

// Close closes both cache connections
func (tc *TieredCache) Close() error {
	var errs []error

	if err := tc.HotClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("hot cache close error: %w", err))
	}

	if err := tc.WarmClient.Close(); err != nil {
		errs = append(errs, fmt.Errorf("warm cache close error: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("cache close errors: %v", errs)
	}

	return nil
}

// HealthCheck verifies both caches are healthy
func (tc *TieredCache) HealthCheck(ctx context.Context) error {
	if err := tc.HotClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("hot cache health check failed: %w", err)
	}

	if err := tc.WarmClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("warm cache health check failed: %w", err)
	}

	return nil
}

// =============================================================================
// CACHE POLICY ENFORCEMENT
// Per CACHE_POLICY.md: "Redis cache is an optimization layer only.
// Cache misses always fall back to PostgreSQL canonical facts."
// =============================================================================

// CacheStatus represents the cache lookup result
type CacheStatus string

const (
	CacheStatusHit           CacheStatus = "HIT"
	CacheStatusMiss          CacheStatus = "MISS"
	CacheStatusStaleNotServed CacheStatus = "STALE_NOT_SERVED"
)

// CacheTier represents which cache tier was accessed
type CacheTier string

const (
	CacheTierHot       CacheTier = "HOT"
	CacheTierWarm      CacheTier = "WARM"
	CacheTierCanonical CacheTier = "CANONICAL" // PostgreSQL fallback
)

// CacheMetadata provides audit trail information for cached responses
// Required per CACHE_POLICY.md for FDA 21 CFR Part 11 compliance
type CacheMetadata struct {
	Status              CacheStatus `json:"status"`
	Tier                CacheTier   `json:"tier"`
	TTLRemainingSeconds int64       `json:"ttl_remaining_seconds,omitempty"`
	CachedAt            time.Time   `json:"cached_at,omitempty"`
	FactVersion         time.Time   `json:"fact_version,omitempty"`
}

// CacheEntry wraps cached data with metadata
type CacheEntry struct {
	Data        []byte    `json:"data"`
	CachedAt    time.Time `json:"cached_at"`
	FactVersion time.Time `json:"fact_version"`
	Severity    string    `json:"severity,omitempty"`
}

// CachePolicy enforces the documented cache policy
type CachePolicy struct {
	// Never serve stale data on DB failure
	ServeStaleOnDBFailure bool

	// Log all cache operations for audit
	AuditLogging bool

	// Minimum TTL to prevent cache thrashing
	MinTTL time.Duration

	// Maximum TTL to ensure freshness
	MaxTTL time.Duration
}

// DefaultCachePolicy returns the policy per CACHE_POLICY.md
func DefaultCachePolicy() *CachePolicy {
	return &CachePolicy{
		ServeStaleOnDBFailure: false, // CRITICAL: Never serve stale on DB failure
		AuditLogging:          true,
		MinTTL:                1 * time.Minute,
		MaxTTL:                48 * time.Hour,
	}
}

// TieredLookup performs a tiered cache lookup with proper fallback
// Implements the cache miss behavior from CACHE_POLICY.md
func (tc *TieredCache) TieredLookup(
	ctx context.Context,
	key string,
	fetchFromDB func(ctx context.Context) ([]byte, time.Time, error),
) ([]byte, *CacheMetadata, error) {
	meta := &CacheMetadata{}

	// Step 1: Check HOT cache
	hotResult, err := tc.HotClient.Get(ctx, key).Result()
	if err == nil {
		ttl, _ := tc.HotClient.TTL(ctx, key).Result()
		meta.Status = CacheStatusHit
		meta.Tier = CacheTierHot
		meta.TTLRemainingSeconds = int64(ttl.Seconds())
		return []byte(hotResult), meta, nil
	}

	// Step 2: Check WARM cache
	warmResult, err := tc.WarmClient.Get(ctx, key).Result()
	if err == nil {
		// Promote to HOT cache
		ttl, _ := tc.WarmClient.TTL(ctx, key).Result()
		tc.HotClient.Set(ctx, key, warmResult, ttl)

		meta.Status = CacheStatusHit
		meta.Tier = CacheTierWarm
		meta.TTLRemainingSeconds = int64(ttl.Seconds())
		return []byte(warmResult), meta, nil
	}

	// Step 3: Cache miss - fetch from PostgreSQL (canonical store)
	data, factVersion, err := fetchFromDB(ctx)
	if err != nil {
		// CRITICAL: Per policy, never serve stale data on DB failure
		meta.Status = CacheStatusStaleNotServed
		meta.Tier = CacheTierCanonical
		return nil, meta, fmt.Errorf("canonical store unavailable: %w", err)
	}

	// Populate caches
	defaultTTL := tc.Config.TTLDefault
	tc.HotClient.Set(ctx, key, data, defaultTTL)
	tc.WarmClient.Set(ctx, key, data, defaultTTL*6) // WARM has 6x TTL

	meta.Status = CacheStatusMiss
	meta.Tier = CacheTierCanonical
	meta.FactVersion = factVersion
	meta.CachedAt = time.Now()

	return data, meta, nil
}

// InvalidateKey removes a key from all cache tiers
// Called on fact activation/deprecation per CACHE_POLICY.md
func (tc *TieredCache) InvalidateKey(ctx context.Context, key string) error {
	hotErr := tc.HotClient.Del(ctx, key).Err()
	warmErr := tc.WarmClient.Del(ctx, key).Err()

	if hotErr != nil && warmErr != nil {
		return fmt.Errorf("failed to invalidate from both caches: hot=%v, warm=%v", hotErr, warmErr)
	}
	return nil
}

// InvalidatePattern removes all keys matching a pattern
// Used for bulk invalidation after ingestion
func (tc *TieredCache) InvalidatePattern(ctx context.Context, pattern string) (int64, error) {
	var totalDeleted int64

	// Invalidate from HOT cache
	hotKeys, err := tc.HotClient.Keys(ctx, pattern).Result()
	if err == nil && len(hotKeys) > 0 {
		deleted, _ := tc.HotClient.Del(ctx, hotKeys...).Result()
		totalDeleted += deleted
	}

	// Invalidate from WARM cache
	warmKeys, err := tc.WarmClient.Keys(ctx, pattern).Result()
	if err == nil && len(warmKeys) > 0 {
		deleted, _ := tc.WarmClient.Del(ctx, warmKeys...).Result()
		totalDeleted += deleted
	}

	return totalDeleted, nil
}

// FlushAll clears all cache tiers
// Used for emergency cache flush or after schema migration
func (tc *TieredCache) FlushAll(ctx context.Context) error {
	if err := tc.HotClient.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush HOT cache: %w", err)
	}

	if err := tc.WarmClient.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush WARM cache: %w", err)
	}

	return nil
}
