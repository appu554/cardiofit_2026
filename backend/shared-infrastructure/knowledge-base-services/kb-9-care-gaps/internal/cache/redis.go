// Package cache provides Redis-based caching for KB-9 Care Gaps Service.
// This caches CQL libraries, measure definitions, and evaluation results.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// Cache provides caching operations for care gaps service.
type Cache struct {
	client     *redis.Client
	logger     *zap.Logger
	ttl        time.Duration
	enabled    bool
	prefix     string
	memCache   *sync.Map // Local in-memory cache as L1
	memCacheTTL time.Duration
}

// Config holds cache configuration.
type Config struct {
	RedisURL string
	TTL      time.Duration
	Enabled  bool
	Prefix   string
}

// NewCache creates a new cache instance.
func NewCache(cfg Config, logger *zap.Logger) (*Cache, error) {
	c := &Cache{
		logger:      logger,
		ttl:         cfg.TTL,
		enabled:     cfg.Enabled,
		prefix:      cfg.Prefix,
		memCache:    &sync.Map{},
		memCacheTTL: 1 * time.Minute, // L1 cache expires faster
	}

	if cfg.Prefix == "" {
		c.prefix = "kb9:"
	}

	if !cfg.Enabled {
		logger.Info("Cache disabled, using in-memory cache only")
		return c, nil
	}

	// Parse Redis URL
	opt, err := parseRedisURL(cfg.RedisURL)
	if err != nil {
		logger.Warn("Failed to parse Redis URL, using in-memory cache only",
			zap.String("url", cfg.RedisURL),
			zap.Error(err),
		)
		c.enabled = false
		return c, nil
	}

	c.client = redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		logger.Warn("Redis connection failed, using in-memory cache only",
			zap.Error(err),
		)
		c.enabled = false
		return c, nil
	}

	logger.Info("Redis cache connected",
		zap.String("addr", opt.Addr),
		zap.Duration("ttl", cfg.TTL),
	)

	return c, nil
}

// parseRedisURL parses a Redis connection URL.
func parseRedisURL(redisURL string) (*redis.Options, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("empty Redis URL")
	}

	// Handle redis:// scheme
	u, err := url.Parse(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}

	opt := &redis.Options{
		Addr: u.Host,
		DB:   0,
	}

	// Parse database number from path
	if len(u.Path) > 1 {
		dbStr := strings.TrimPrefix(u.Path, "/")
		if db, err := strconv.Atoi(dbStr); err == nil {
			opt.DB = db
		}
	}

	// Parse password from user info
	if u.User != nil {
		if pw, ok := u.User.Password(); ok {
			opt.Password = pw
		}
	}

	return opt, nil
}

// Close closes the Redis connection.
func (c *Cache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// memCacheEntry holds in-memory cache entries with TTL.
type memCacheEntry struct {
	value     interface{}
	expiresAt time.Time
}

// ========== CQL Library Caching ==========

// CachedLibrary represents a cached CQL library.
type CachedLibrary struct {
	LibraryID   string    `json:"libraryId"`
	Version     string    `json:"version"`
	Content     string    `json:"content"`
	CompiledELM string    `json:"compiledElm,omitempty"`
	CachedAt    time.Time `json:"cachedAt"`
}

// GetCQLLibrary retrieves a cached CQL library.
func (c *Cache) GetCQLLibrary(ctx context.Context, libraryID, version string) (*CachedLibrary, error) {
	key := c.libraryKey(libraryID, version)

	// Check L1 memory cache first
	if entry, ok := c.memCache.Load(key); ok {
		if e, ok := entry.(*memCacheEntry); ok {
			if time.Now().Before(e.expiresAt) {
				if lib, ok := e.value.(*CachedLibrary); ok {
					return lib, nil
				}
			}
			c.memCache.Delete(key) // Expired
		}
	}

	if !c.enabled || c.client == nil {
		return nil, nil
	}

	// Check Redis L2 cache
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		c.logger.Warn("Redis GET failed", zap.String("key", key), zap.Error(err))
		return nil, nil
	}

	var lib CachedLibrary
	if err := json.Unmarshal(data, &lib); err != nil {
		c.logger.Warn("Failed to unmarshal cached library", zap.Error(err))
		return nil, nil
	}

	// Populate L1 cache
	c.memCache.Store(key, &memCacheEntry{
		value:     &lib,
		expiresAt: time.Now().Add(c.memCacheTTL),
	})

	return &lib, nil
}

// SetCQLLibrary caches a CQL library.
func (c *Cache) SetCQLLibrary(ctx context.Context, lib *CachedLibrary) error {
	if lib == nil {
		return nil
	}

	key := c.libraryKey(lib.LibraryID, lib.Version)
	lib.CachedAt = time.Now().UTC()

	// Update L1 memory cache
	c.memCache.Store(key, &memCacheEntry{
		value:     lib,
		expiresAt: time.Now().Add(c.memCacheTTL),
	})

	if !c.enabled || c.client == nil {
		return nil
	}

	// Update Redis L2 cache
	data, err := json.Marshal(lib)
	if err != nil {
		return fmt.Errorf("failed to marshal library: %w", err)
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *Cache) libraryKey(libraryID, version string) string {
	return fmt.Sprintf("%slibrary:%s:%s", c.prefix, libraryID, version)
}

// ========== Measure Definition Caching ==========

// CachedMeasure represents a cached measure definition.
type CachedMeasure struct {
	MeasureID   string                 `json:"measureId"`
	Version     string                 `json:"version"`
	Definition  map[string]interface{} `json:"definition"`
	CachedAt    time.Time              `json:"cachedAt"`
}

// GetMeasure retrieves a cached measure definition.
func (c *Cache) GetMeasure(ctx context.Context, measureID string) (*CachedMeasure, error) {
	key := c.measureKey(measureID)

	// Check L1 cache
	if entry, ok := c.memCache.Load(key); ok {
		if e, ok := entry.(*memCacheEntry); ok {
			if time.Now().Before(e.expiresAt) {
				if m, ok := e.value.(*CachedMeasure); ok {
					return m, nil
				}
			}
			c.memCache.Delete(key)
		}
	}

	if !c.enabled || c.client == nil {
		return nil, nil
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		c.logger.Warn("Redis GET failed", zap.String("key", key), zap.Error(err))
		return nil, nil
	}

	var measure CachedMeasure
	if err := json.Unmarshal(data, &measure); err != nil {
		return nil, nil
	}

	c.memCache.Store(key, &memCacheEntry{
		value:     &measure,
		expiresAt: time.Now().Add(c.memCacheTTL),
	})

	return &measure, nil
}

// SetMeasure caches a measure definition.
func (c *Cache) SetMeasure(ctx context.Context, measure *CachedMeasure) error {
	if measure == nil {
		return nil
	}

	key := c.measureKey(measure.MeasureID)
	measure.CachedAt = time.Now().UTC()

	c.memCache.Store(key, &memCacheEntry{
		value:     measure,
		expiresAt: time.Now().Add(c.memCacheTTL),
	})

	if !c.enabled || c.client == nil {
		return nil
	}

	data, err := json.Marshal(measure)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}

func (c *Cache) measureKey(measureID string) string {
	return fmt.Sprintf("%smeasure:%s", c.prefix, measureID)
}

// ========== Evaluation Result Caching ==========

// CachedEvaluation represents a cached evaluation result.
type CachedEvaluation struct {
	PatientID string                 `json:"patientId"`
	MeasureID string                 `json:"measureId"`
	PeriodKey string                 `json:"periodKey"`
	Result    map[string]interface{} `json:"result"`
	CachedAt  time.Time              `json:"cachedAt"`
}

// GetEvaluation retrieves a cached evaluation result.
func (c *Cache) GetEvaluation(ctx context.Context, patientID, measureID, periodStart, periodEnd string) (*CachedEvaluation, error) {
	key := c.evaluationKey(patientID, measureID, periodStart, periodEnd)

	// Check L1 cache
	if entry, ok := c.memCache.Load(key); ok {
		if e, ok := entry.(*memCacheEntry); ok {
			if time.Now().Before(e.expiresAt) {
				if eval, ok := e.value.(*CachedEvaluation); ok {
					return eval, nil
				}
			}
			c.memCache.Delete(key)
		}
	}

	if !c.enabled || c.client == nil {
		return nil, nil
	}

	// Short TTL for evaluation results (they can become stale)
	evalTTL := 5 * time.Minute
	if c.ttl < evalTTL {
		evalTTL = c.ttl
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, nil
	}

	var eval CachedEvaluation
	if err := json.Unmarshal(data, &eval); err != nil {
		return nil, nil
	}

	// Check if cache is stale (evaluation results have shorter effective TTL)
	if time.Since(eval.CachedAt) > evalTTL {
		return nil, nil
	}

	c.memCache.Store(key, &memCacheEntry{
		value:     &eval,
		expiresAt: time.Now().Add(c.memCacheTTL),
	})

	return &eval, nil
}

// SetEvaluation caches an evaluation result.
func (c *Cache) SetEvaluation(ctx context.Context, eval *CachedEvaluation) error {
	if eval == nil {
		return nil
	}

	key := c.evaluationKey(eval.PatientID, eval.MeasureID, eval.PeriodKey, "")
	eval.CachedAt = time.Now().UTC()

	c.memCache.Store(key, &memCacheEntry{
		value:     eval,
		expiresAt: time.Now().Add(c.memCacheTTL),
	})

	if !c.enabled || c.client == nil {
		return nil
	}

	data, err := json.Marshal(eval)
	if err != nil {
		return err
	}

	// Shorter TTL for evaluation results
	evalTTL := 5 * time.Minute
	return c.client.Set(ctx, key, data, evalTTL).Err()
}

func (c *Cache) evaluationKey(patientID, measureID, periodStart, periodEnd string) string {
	return fmt.Sprintf("%seval:%s:%s:%s-%s", c.prefix, patientID, measureID, periodStart, periodEnd)
}

// ========== Cache Stats & Management ==========

// Stats returns cache statistics.
type Stats struct {
	Enabled     bool   `json:"enabled"`
	RedisAddr   string `json:"redisAddr,omitempty"`
	MemoryItems int    `json:"memoryItems"`
	TTL         string `json:"ttl"`
}

// GetStats returns cache statistics.
func (c *Cache) GetStats() Stats {
	count := 0
	c.memCache.Range(func(_, _ interface{}) bool {
		count++
		return true
	})

	stats := Stats{
		Enabled:     c.enabled,
		MemoryItems: count,
		TTL:         c.ttl.String(),
	}

	if c.client != nil {
		stats.RedisAddr = c.client.Options().Addr
	}

	return stats
}

// Clear clears all cached items.
func (c *Cache) Clear(ctx context.Context) error {
	// Clear memory cache
	c.memCache = &sync.Map{}

	if !c.enabled || c.client == nil {
		return nil
	}

	// Clear Redis keys with prefix
	pattern := c.prefix + "*"
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		c.client.Del(ctx, iter.Val())
	}

	return iter.Err()
}

// InvalidatePatient invalidates all cached data for a patient.
func (c *Cache) InvalidatePatient(ctx context.Context, patientID string) error {
	// Clear from memory cache
	c.memCache.Range(func(key, _ interface{}) bool {
		if k, ok := key.(string); ok {
			if strings.Contains(k, patientID) {
				c.memCache.Delete(key)
			}
		}
		return true
	})

	if !c.enabled || c.client == nil {
		return nil
	}

	// Clear from Redis
	pattern := c.prefix + "eval:" + patientID + ":*"
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		c.client.Del(ctx, iter.Val())
	}

	return iter.Err()
}
