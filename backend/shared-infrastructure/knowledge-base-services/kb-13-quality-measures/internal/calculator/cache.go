// Package calculator provides the quality measure calculation engine for KB-13.
package calculator

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/models"
)

// Cache provides in-memory caching for calculation results.
// 🔴 CRITICAL: Cache is for performance only - authoritative results are in PostgreSQL
type Cache struct {
	ttl     time.Duration
	maxSize int
	logger  *zap.Logger

	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

type cacheEntry struct {
	result    *models.CalculationResult
	expiresAt time.Time
}

// CacheConfig holds cache configuration.
type CacheConfig struct {
	TTL     time.Duration
	MaxSize int
	Enabled bool
}

// NewCache creates a new result cache.
func NewCache(cfg *CacheConfig, logger *zap.Logger) *Cache {
	c := &Cache{
		ttl:     cfg.TTL,
		maxSize: cfg.MaxSize,
		logger:  logger,
		entries: make(map[string]*cacheEntry),
	}

	// Start background cleanup goroutine
	go c.cleanupLoop()

	return c
}

// Get retrieves a cached result.
func (c *Cache) Get(key string) (*models.CalculationResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	c.logger.Debug("Cache hit",
		zap.String("key", key),
	)

	return entry.result, true
}

// Set stores a result in the cache.
func (c *Cache) Set(key string, result *models.CalculationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce max size - simple LRU would be better but this works
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	c.entries[key] = &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(c.ttl),
	}

	c.logger.Debug("Cache set",
		zap.String("key", key),
		zap.Duration("ttl", c.ttl),
	)
}

// Delete removes a cached result.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

// Clear removes all cached results.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*cacheEntry)
	c.logger.Info("Cache cleared")
}

// Size returns the number of cached entries.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stats returns cache statistics.
type CacheStats struct {
	Size      int       `json:"size"`
	MaxSize   int       `json:"max_size"`
	TTL       string    `json:"ttl"`
	OldestKey string    `json:"oldest_key,omitempty"`
	NewestKey string    `json:"newest_key,omitempty"`
}

func (c *Cache) Stats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &CacheStats{
		Size:    len(c.entries),
		MaxSize: c.maxSize,
		TTL:     c.ttl.String(),
	}

	var oldest, newest time.Time
	for key, entry := range c.entries {
		expiry := entry.expiresAt
		created := expiry.Add(-c.ttl)

		if oldest.IsZero() || created.Before(oldest) {
			oldest = created
			stats.OldestKey = key
		}
		if newest.IsZero() || created.After(newest) {
			newest = created
			stats.NewestKey = key
		}
	}

	return stats
}

// evictOldest removes the oldest entry (must be called with lock held)
func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.entries {
		if oldestTime.IsZero() || entry.expiresAt.Before(oldestTime) {
			oldestTime = entry.expiresAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.logger.Debug("Evicted oldest cache entry",
			zap.String("key", oldestKey),
		)
	}
}

// cleanupLoop periodically removes expired entries.
func (c *Cache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := 0

	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
			expired++
		}
	}

	if expired > 0 {
		c.logger.Debug("Cache cleanup completed",
			zap.Int("expired_entries", expired),
			zap.Int("remaining_entries", len(c.entries)),
		)
	}
}

// CacheKey generates a cache key for a calculation request.
func CacheKey(measureID string, periodStart, periodEnd time.Time, reportType models.ReportType) string {
	return measureID + "|" +
		periodStart.Format("2006-01-02") + "|" +
		periodEnd.Format("2006-01-02") + "|" +
		string(reportType)
}

// CachedEngine wraps Engine with caching support.
type CachedEngine struct {
	engine *Engine
	cache  *Cache
	logger *zap.Logger
}

// NewCachedEngine creates an engine with caching.
func NewCachedEngine(engine *Engine, cache *Cache, logger *zap.Logger) *CachedEngine {
	return &CachedEngine{
		engine: engine,
		cache:  cache,
		logger: logger,
	}
}

// Calculate performs a measure calculation with caching.
func (ce *CachedEngine) Calculate(ctx context.Context, req *CalculateRequest) (*models.CalculationResult, error) {
	// Generate cache key
	var periodStart, periodEnd time.Time
	if req.PeriodStart != nil {
		periodStart = *req.PeriodStart
	}
	if req.PeriodEnd != nil {
		periodEnd = *req.PeriodEnd
	}

	key := CacheKey(req.MeasureID, periodStart, periodEnd, req.ReportType)

	// Check cache
	if result, found := ce.cache.Get(key); found {
		ce.logger.Debug("Returning cached calculation result",
			zap.String("measure_id", req.MeasureID),
		)
		return result, nil
	}

	// Calculate
	result, err := ce.engine.Calculate(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache result
	ce.cache.Set(key, result)

	return result, nil
}

// InvalidateMeasure removes all cached results for a measure.
func (ce *CachedEngine) InvalidateMeasure(measureID string) {
	ce.cache.mu.Lock()
	defer ce.cache.mu.Unlock()

	for key := range ce.cache.entries {
		if len(key) > len(measureID) && key[:len(measureID)] == measureID {
			delete(ce.cache.entries, key)
		}
	}

	ce.logger.Info("Invalidated cache for measure",
		zap.String("measure_id", measureID),
	)
}

// GetEngine returns the underlying engine for operations that shouldn't be cached.
func (ce *CachedEngine) GetEngine() *Engine {
	return ce.engine
}

// GetCache returns the cache for inspection/management.
func (ce *CachedEngine) GetCache() *Cache {
	return ce.cache
}
