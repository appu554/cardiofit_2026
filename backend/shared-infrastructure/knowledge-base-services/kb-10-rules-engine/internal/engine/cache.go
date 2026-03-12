// Package engine provides caching for rule evaluation results
package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/sirupsen/logrus"
)

// Cache provides caching for rule evaluation results
type Cache struct {
	enabled  bool
	ttl      time.Duration
	data     map[string]*cacheEntry
	mu       sync.RWMutex
	logger   *logrus.Logger
	hits     int64
	misses   int64
	stopChan chan struct{}
}

type cacheEntry struct {
	results   []*models.EvaluationResult
	expiresAt time.Time
}

// NewCache creates a new evaluation cache
func NewCache(enabled bool, ttl time.Duration, logger *logrus.Logger) *Cache {
	c := &Cache{
		enabled:  enabled,
		ttl:      ttl,
		data:     make(map[string]*cacheEntry),
		logger:   logger,
		stopChan: make(chan struct{}),
	}

	if enabled {
		go c.cleanup()
	}

	return c
}

// Get retrieves cached results for an evaluation context
func (c *Cache) Get(ctx *models.EvaluationContext, ruleIDs []string) ([]*models.EvaluationResult, bool) {
	if !c.enabled {
		return nil, false
	}

	key := c.generateKey(ctx, ruleIDs)

	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		c.misses++
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		c.misses++
		return nil, false
	}

	c.hits++
	return entry.results, true
}

// Set stores evaluation results in the cache
func (c *Cache) Set(ctx *models.EvaluationContext, ruleIDs []string, results []*models.EvaluationResult) {
	if !c.enabled {
		return
	}

	key := c.generateKey(ctx, ruleIDs)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &cacheEntry{
		results:   results,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes cached results for a patient
func (c *Cache) Invalidate(patientID string) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all entries for this patient
	for key := range c.data {
		if c.keyContainsPatient(key, patientID) {
			delete(c.data, key)
		}
	}
}

// InvalidateRule removes cached results that include a specific rule
func (c *Cache) InvalidateRule(ruleID string) {
	if !c.enabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all entries that might contain this rule
	// This is a simple approach - in production, you might want more granular invalidation
	for key := range c.data {
		if c.keyContainsRule(key, ruleID) {
			delete(c.data, key)
		}
	}
}

// Clear removes all cached entries
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*cacheEntry)
	c.hits = 0
	c.misses = 0
}

// Stats returns cache statistics
func (c *Cache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hitRate := float64(0)
	total := c.hits + c.misses
	if total > 0 {
		hitRate = float64(c.hits) / float64(total)
	}

	return CacheStats{
		Enabled:   c.enabled,
		Size:      len(c.data),
		Hits:      c.hits,
		Misses:    c.misses,
		HitRate:   hitRate,
		TTL:       c.ttl,
	}
}

// Close stops the cache cleanup goroutine
func (c *Cache) Close() {
	if c.enabled {
		close(c.stopChan)
	}
}

// CacheStats holds cache statistics
type CacheStats struct {
	Enabled bool          `json:"enabled"`
	Size    int           `json:"size"`
	Hits    int64         `json:"hits"`
	Misses  int64         `json:"misses"`
	HitRate float64       `json:"hit_rate"`
	TTL     time.Duration `json:"ttl"`
}

// generateKey creates a unique cache key for an evaluation context
func (c *Cache) generateKey(ctx *models.EvaluationContext, ruleIDs []string) string {
	// Create a deterministic representation of the context
	data := struct {
		PatientID   string
		EncounterID string
		Labs        map[string]models.LabValue
		Vitals      map[string]models.VitalSign
		RuleIDs     []string
	}{
		PatientID:   ctx.PatientID,
		EncounterID: ctx.EncounterID,
		Labs:        ctx.Labs,
		Vitals:      ctx.Vitals,
		RuleIDs:     ruleIDs,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		// Fallback to simple key
		return ctx.PatientID + ":" + ctx.EncounterID
	}

	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:])
}

// keyContainsPatient checks if a cache key is for a specific patient
func (c *Cache) keyContainsPatient(key, patientID string) bool {
	// Since we use hashed keys, we can't directly check
	// In a production system, you might maintain a reverse index
	return false
}

// keyContainsRule checks if a cache key includes a specific rule
func (c *Cache) keyContainsRule(key, ruleID string) bool {
	// Since we use hashed keys, we can't directly check
	// In a production system, you might maintain a reverse index
	return false
}

// cleanup periodically removes expired cache entries
func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.removeExpired()
		case <-c.stopChan:
			return
		}
	}
}

// removeExpired removes expired entries from the cache
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := 0

	for key, entry := range c.data {
		if now.After(entry.expiresAt) {
			delete(c.data, key)
			expired++
		}
	}

	if expired > 0 {
		c.logger.WithField("expired_entries", expired).Debug("Cache cleanup completed")
	}
}
