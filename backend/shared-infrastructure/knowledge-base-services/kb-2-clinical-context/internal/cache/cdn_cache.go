package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CDNCache implements L3 caching for static clinical definitions
type CDNCache struct {
	baseURL     string
	httpClient  *http.Client
	etagCache   map[string]string
	etag_mutex  sync.RWMutex
	logger      *zap.Logger
	metrics     *CDNMetrics
}

type CDNMetrics struct {
	Hits           int64
	Misses         int64
	Errors         int64
	LastAccess     time.Time
	AverageLatency time.Duration
	TotalRequests  int64
	mutex          sync.RWMutex
}

type CDNItem struct {
	Content     interface{} `json:"content"`
	ETag        string      `json:"etag"`
	Version     string      `json:"version"`
	LastUpdated time.Time   `json:"last_updated"`
	ContentType string      `json:"content_type"`
}

// NewCDNCache creates a new CDN cache instance
func NewCDNCache(baseURL string, logger *zap.Logger) *CDNCache {
	return &CDNCache{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		etagCache: make(map[string]string),
		logger:    logger,
		metrics:   &CDNMetrics{},
	}
}

// Get retrieves static content from CDN with ETag validation
func (c *CDNCache) Get(ctx context.Context, key string) (interface{}, error) {
	start := time.Now()
	defer func() {
		c.updateMetrics(time.Since(start))
	}()

	// Build CDN URL
	url := fmt.Sprintf("%s/%s", strings.TrimRight(c.baseURL, "/"), key)
	
	c.logger.Debug("CDN cache request", 
		zap.String("key", key),
		zap.String("url", url))

	// Create request with conditional headers
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		c.recordError()
		return nil, fmt.Errorf("failed to create CDN request: %w", err)
	}

	// Add ETag header if we have one cached
	c.etag_mutex.RLock()
	if etag, exists := c.etagCache[key]; exists {
		req.Header.Set("If-None-Match", etag)
	}
	c.etag_mutex.RUnlock()

	// Add standard headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "max-age=3600")
	req.Header.Set("User-Agent", "KB2-Clinical-Context/2.0")

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordError()
		return nil, fmt.Errorf("CDN request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		c.recordHit()
		c.logger.Debug("CDN cache hit (304 Not Modified)", zap.String("key", key))
		// Return cached content - for simplicity, we'll return a cache marker
		return map[string]interface{}{
			"cache_status": "not_modified",
			"etag":         resp.Header.Get("ETag"),
		}, nil
	}

	// Handle other non-200 responses
	if resp.StatusCode != http.StatusOK {
		c.recordError()
		return nil, fmt.Errorf("CDN returned status %d for key %s", resp.StatusCode, key)
	}

	// Parse response
	var cdnItem CDNItem
	if err := json.NewDecoder(resp.Body).Decode(&cdnItem); err != nil {
		c.recordError()
		return nil, fmt.Errorf("failed to decode CDN response: %w", err)
	}

	// Update ETag cache
	if etag := resp.Header.Get("ETag"); etag != "" {
		c.etag_mutex.Lock()
		c.etagCache[key] = etag
		c.etag_mutex.Unlock()
	}

	c.recordHit()
	c.logger.Debug("CDN cache content retrieved", 
		zap.String("key", key),
		zap.String("etag", cdnItem.ETag),
		zap.String("version", cdnItem.Version))

	return cdnItem.Content, nil
}

// Set is not implemented for CDN cache (read-only)
func (c *CDNCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return fmt.Errorf("CDN cache is read-only - cannot set key %s", key)
}

// Delete is not implemented for CDN cache (read-only)
func (c *CDNCache) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("CDN cache is read-only - cannot delete key %s", key)
}

// InvalidatePattern invalidates cached ETags matching a pattern
func (c *CDNCache) InvalidatePattern(pattern string) error {
	c.etag_mutex.Lock()
	defer c.etag_mutex.Unlock()

	count := 0
	for key := range c.etagCache {
		// Simple pattern matching - in production, use regex
		if strings.Contains(key, pattern) {
			delete(c.etagCache, key)
			count++
		}
	}

	c.logger.Info("CDN cache pattern invalidation",
		zap.String("pattern", pattern),
		zap.Int("invalidated_keys", count))

	return nil
}

// GetStaticContent retrieves static clinical definitions
func (c *CDNCache) GetStaticContent(ctx context.Context, contentType string, version string) (interface{}, error) {
	key := fmt.Sprintf("static/%s/%s", contentType, version)
	return c.Get(ctx, key)
}

// GetPhenotypeDefinitions retrieves phenotype definitions from CDN
func (c *CDNCache) GetPhenotypeDefinitions(ctx context.Context, domain string, version string) (interface{}, error) {
	key := fmt.Sprintf("phenotypes/%s/%s", domain, version)
	return c.Get(ctx, key)
}

// GetRiskModels retrieves risk models from CDN
func (c *CDNCache) GetRiskModels(ctx context.Context, domain string, version string) (interface{}, error) {
	key := fmt.Sprintf("risk-models/%s/%s", domain, version)
	return c.Get(ctx, key)
}

// GetTreatmentPreferences retrieves treatment preferences from CDN
func (c *CDNCache) GetTreatmentPreferences(ctx context.Context, condition string, version string) (interface{}, error) {
	key := fmt.Sprintf("treatment-preferences/%s/%s", condition, version)
	return c.Get(ctx, key)
}

// Metrics and monitoring methods

func (c *CDNCache) updateMetrics(duration time.Duration) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()
	
	c.metrics.TotalRequests++
	c.metrics.LastAccess = time.Now()
	
	// Update average latency
	if c.metrics.AverageLatency == 0 {
		c.metrics.AverageLatency = duration
	} else {
		c.metrics.AverageLatency = (c.metrics.AverageLatency + duration) / 2
	}
}

func (c *CDNCache) recordHit() {
	c.metrics.mutex.Lock()
	c.metrics.Hits++
	c.metrics.mutex.Unlock()
}

func (c *CDNCache) recordMiss() {
	c.metrics.mutex.Lock()
	c.metrics.Misses++
	c.metrics.mutex.Unlock()
}

func (c *CDNCache) recordError() {
	c.metrics.mutex.Lock()
	c.metrics.Errors++
	c.metrics.mutex.Unlock()
}

// GetMetrics returns current CDN cache metrics
func (c *CDNCache) GetMetrics() map[string]interface{} {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()

	totalRequests := c.metrics.Hits + c.metrics.Misses + c.metrics.Errors
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(c.metrics.Hits) / float64(totalRequests) * 100
	}

	return map[string]interface{}{
		"hits":             c.metrics.Hits,
		"misses":           c.metrics.Misses,
		"errors":           c.metrics.Errors,
		"total_requests":   totalRequests,
		"hit_rate_pct":     hitRate,
		"average_latency":  c.metrics.AverageLatency,
		"last_access":      c.metrics.LastAccess,
	}
}

// GenerateContentHash generates MD5 hash for content versioning
func (c *CDNCache) GenerateContentHash(content interface{}) (string, error) {
	data, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("failed to marshal content for hashing: %w", err)
	}
	
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash), nil
}

// BuildCacheKey builds standardized cache keys for CDN
func (c *CDNCache) BuildCacheKey(components ...string) string {
	return strings.Join(components, "/")
}

// Close cleans up CDN cache resources
func (c *CDNCache) Close() error {
	c.httpClient.CloseIdleConnections()
	
	c.logger.Info("CDN cache closed", 
		zap.Int64("total_requests", c.metrics.TotalRequests),
		zap.Int64("total_hits", c.metrics.Hits))
	
	return nil
}

// Health check for CDN cache
func (c *CDNCache) Health(ctx context.Context) error {
	// Simple health check - try to access a health endpoint
	healthURL := fmt.Sprintf("%s/health", strings.TrimRight(c.baseURL, "/"))
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CDN health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CDN health check returned status %d", resp.StatusCode)
	}

	return nil
}