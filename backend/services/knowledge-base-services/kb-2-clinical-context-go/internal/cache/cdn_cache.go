package cache

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"kb-2-clinical-context-go/internal/metrics"
)

// CDNCacheConfig configures the CDN L3 cache for static content
type CDNCacheConfig struct {
	BaseURL      string            // CDN base URL
	VersionPrefix string           // Version prefix for cache busting
	CacheHeaders map[string]string // HTTP cache headers
	StaticPaths  []string          // Paths for static content
	Client       *http.Client      // HTTP client for CDN requests
}

// CDNCache implements L3 caching for static clinical definitions
type CDNCache struct {
	config  *CDNCacheConfig
	metrics *metrics.PrometheusMetrics
	client  *http.Client
	
	// Statistics
	hits      int64
	misses    int64
	errors    int64
	requests  int64
	
	// Static content versioning
	versionMap map[string]string
}

// NewCDNCache creates a new CDN cache instance
func NewCDNCache(config *CDNCacheConfig, metricsCollector *metrics.PrometheusMetrics) *CDNCache {
	client := config.Client
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		}
	}
	
	return &CDNCache{
		config:     config,
		metrics:    metricsCollector,
		client:     client,
		versionMap: make(map[string]string),
	}
}

// Get retrieves static content from CDN
func (cc *CDNCache) Get(ctx context.Context, key string) (interface{}, bool) {
	atomic.AddInt64(&cc.requests, 1)
	
	// Convert cache key to CDN path
	cdnPath, err := cc.keyToCDNPath(key)
	if err != nil {
		atomic.AddInt64(&cc.errors, 1)
		return nil, false
	}
	
	// Build full URL with versioning
	url := cc.buildVersionedURL(cdnPath)
	
	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		atomic.AddInt64(&cc.errors, 1)
		return nil, false
	}
	
	// Add cache headers
	for header, value := range cc.config.CacheHeaders {
		req.Header.Set(header, value)
	}
	
	// Set conditional headers for efficient caching
	if etag := cc.getETagForKey(key); etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	
	// Make request
	resp, err := cc.client.Do(req)
	if err != nil {
		atomic.AddInt64(&cc.errors, 1)
		return nil, false
	}
	defer resp.Body.Close()
	
	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		atomic.AddInt64(&cc.hits, 1)
		return cc.getCachedContent(key), true
	}
	
	// Handle successful response
	if resp.StatusCode == http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			atomic.AddInt64(&cc.errors, 1)
			return nil, false
		}
		
		// Parse content based on Content-Type
		contentType := resp.Header.Get("Content-Type")
		value, err := cc.parseContent(data, contentType)
		if err != nil {
			atomic.AddInt64(&cc.errors, 1)
			return nil, false
		}
		
		// Update ETag for future requests
		if etag := resp.Header.Get("ETag"); etag != "" {
			cc.updateETag(key, etag)
		}
		
		// Cache content locally for subsequent 304 responses
		cc.setCachedContent(key, value)
		
		atomic.AddInt64(&cc.hits, 1)
		return value, true
	}
	
	// Cache miss for other status codes
	atomic.AddInt64(&cc.misses, 1)
	return nil, false
}

// Set uploads content to CDN (typically for static definitions)
func (cc *CDNCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// For CDN caching, we typically don't upload content dynamically
	// This would be used for updating static definitions
	
	cdnPath, err := cc.keyToCDNPath(key)
	if err != nil {
		return fmt.Errorf("invalid CDN path for key %s: %w", key, err)
	}
	
	// Serialize content
	data, err := cc.serializeContent(value)
	if err != nil {
		return fmt.Errorf("content serialization failed: %w", err)
	}
	
	// For this implementation, we'll simulate CDN upload
	// In production, this would use CDN-specific APIs
	log := fmt.Sprintf("CDN upload simulated for %s (%d bytes)", cdnPath, len(data))
	_ = log // Avoid unused variable warning
	
	return nil
}

// Invalidate invalidates CDN content (cache busting)
func (cc *CDNCache) Invalidate(ctx context.Context, key string) error {
	// CDN invalidation typically involves:
	// 1. Cache purge requests to CDN provider
	// 2. Version bumping for cache busting
	// 3. Updating internal version mappings
	
	// Update version to bust cache
	currentVersion := cc.getVersionForKey(key)
	newVersion := cc.generateNewVersion(currentVersion)
	cc.updateVersion(key, newVersion)
	
	return nil
}

// InvalidatePattern invalidates all CDN content matching pattern
func (cc *CDNCache) InvalidatePattern(ctx context.Context, pattern string) error {
	// For pattern invalidation, we would typically:
	// 1. Find all matching static content
	// 2. Bulk invalidate through CDN API
	// 3. Update version mappings
	
	keys := cc.findMatchingKeys(pattern)
	for _, key := range keys {
		if err := cc.Invalidate(ctx, key); err != nil {
			return fmt.Errorf("failed to invalidate key %s: %w", key, err)
		}
	}
	
	return nil
}

// GetStats returns CDN cache statistics
func (cc *CDNCache) GetStats() *CacheStats {
	totalReqs := atomic.LoadInt64(&cc.requests)
	hits := atomic.LoadInt64(&cc.hits)
	misses := atomic.LoadInt64(&cc.misses)
	
	hitRate := 0.0
	if totalReqs > 0 {
		hitRate = float64(hits) / float64(totalReqs)
	}
	
	return &CacheStats{
		HitRate:    hitRate,
		MissRate:   1.0 - hitRate,
		Operations: totalReqs,
	}
}

// Private methods

// keyToCDNPath converts cache key to CDN path
func (cc *CDNCache) keyToCDNPath(key string) (string, error) {
	// Convert cache keys to CDN paths
	// Example: "phenotype_definition:diabetes_t2" -> "/phenotypes/diabetes_t2.json"
	
	parts := strings.SplitN(key, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid cache key format: %s", key)
	}
	
	prefix := parts[0]
	id := parts[1]
	
	switch prefix {
	case "phenotype_definition":
		return fmt.Sprintf("/phenotypes/%s.json", id), nil
	case "risk_model":
		return fmt.Sprintf("/risk-models/%s.json", id), nil
	case "treatment_preference_template":
		return fmt.Sprintf("/treatment-preferences/%s.json", id), nil
	case "institutional_rule":
		return fmt.Sprintf("/institutional-rules/%s.json", id), nil
	case "static":
		return fmt.Sprintf("/static/%s", id), nil
	default:
		return "", fmt.Errorf("unsupported CDN content type: %s", prefix)
	}
}

// buildVersionedURL builds URL with version for cache busting
func (cc *CDNCache) buildVersionedURL(path string) string {
	version := cc.getVersionForPath(path)
	baseURL := strings.TrimSuffix(cc.config.BaseURL, "/")
	
	if version != "" {
		return fmt.Sprintf("%s/%s%s?v=%s", baseURL, cc.config.VersionPrefix, path, version)
	}
	
	return fmt.Sprintf("%s/%s%s", baseURL, cc.config.VersionPrefix, path)
}

// getVersionForPath gets version for a CDN path
func (cc *CDNCache) getVersionForPath(path string) string {
	if version, exists := cc.versionMap[path]; exists {
		return version
	}
	return "1.0" // Default version
}

// getVersionForKey gets version for a cache key
func (cc *CDNCache) getVersionForKey(key string) string {
	path, err := cc.keyToCDNPath(key)
	if err != nil {
		return "1.0"
	}
	return cc.getVersionForPath(path)
}

// generateNewVersion generates a new version string
func (cc *CDNCache) generateNewVersion(currentVersion string) string {
	// Simple timestamp-based versioning
	return fmt.Sprintf("%d", time.Now().Unix())
}

// updateVersion updates version mapping
func (cc *CDNCache) updateVersion(key, version string) {
	path, err := cc.keyToCDNPath(key)
	if err != nil {
		return
	}
	cc.versionMap[path] = version
}

// getETagForKey gets ETag for conditional requests
func (cc *CDNCache) getETagForKey(key string) string {
	// In production, you would maintain ETag mappings
	// For now, use version as ETag
	version := cc.getVersionForKey(key)
	if version != "" {
		return fmt.Sprintf("\"%s\"", version)
	}
	return ""
}

// parseContent parses CDN response content based on Content-Type
func (cc *CDNCache) parseContent(data []byte, contentType string) (interface{}, error) {
	// Determine parsing strategy based on content type
	if strings.Contains(contentType, "application/json") {
		// Parse as JSON
		var result interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("JSON parsing failed: %w", err)
		}
		return result, nil
	}
	
	if strings.Contains(contentType, "text/") {
		// Return as string
		return string(data), nil
	}
	
	// Return raw bytes for other content types
	return data, nil
}

// serializeContent serializes content for CDN upload
func (cc *CDNCache) serializeContent(value interface{}) ([]byte, error) {
	// Serialize based on content type
	switch v := value.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		// Default to JSON serialization
		return json.Marshal(v)
	}
}

// Local content caching for 304 responses
var localContentCache = make(map[string]interface{})

// getCachedContent gets locally cached content for 304 responses
func (cc *CDNCache) getCachedContent(key string) interface{} {
	if content, exists := localContentCache[key]; exists {
		return content
	}
	return nil
}

// setCachedContent caches content locally for 304 responses
func (cc *CDNCache) setCachedContent(key string, content interface{}) {
	localContentCache[key] = content
}

// findMatchingKeys finds keys matching a pattern
func (cc *CDNCache) findMatchingKeys(pattern string) []string {
	// This would typically query the CDN or local mapping
	// For now, return empty slice as CDN doesn't support pattern operations
	return []string{}
}

// Specialized methods for clinical content

// GetPhenotypeDefinitions retrieves phenotype definitions from CDN
func (cc *CDNCache) GetPhenotypeDefinitions(ctx context.Context) (interface{}, bool) {
	return cc.Get(ctx, "static:phenotype_definitions_all")
}

// GetRiskModels retrieves risk models from CDN
func (cc *CDNCache) GetRiskModels(ctx context.Context) (interface{}, bool) {
	return cc.Get(ctx, "static:risk_models_all")
}

// GetTreatmentPreferenceTemplates retrieves treatment preference templates from CDN
func (cc *CDNCache) GetTreatmentPreferenceTemplates(ctx context.Context) (interface{}, bool) {
	return cc.Get(ctx, "static:treatment_preference_templates_all")
}

// GetInstitutionalRules retrieves institutional rules from CDN
func (cc *CDNCache) GetInstitutionalRules(ctx context.Context) (interface{}, bool) {
	return cc.Get(ctx, "static:institutional_rules_all")
}

// Health and monitoring

// IsHealthy checks CDN cache health
func (cc *CDNCache) IsHealthy(ctx context.Context) bool {
	// Test CDN connectivity with a simple HEAD request
	testURL := strings.TrimSuffix(cc.config.BaseURL, "/") + "/health"
	
	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		return false
	}
	
	resp, err := cc.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Consider healthy if we can reach the CDN
	return resp.StatusCode < 500
}

// GetCDNLatency measures CDN response latency
func (cc *CDNCache) GetCDNLatency(ctx context.Context) time.Duration {
	start := time.Now()
	
	// Test with a lightweight request
	testURL := strings.TrimSuffix(cc.config.BaseURL, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		return 0
	}
	
	resp, err := cc.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	
	return time.Since(start)
}

// Performance optimization

// PreloadStaticContent preloads all static content into local cache
func (cc *CDNCache) PreloadStaticContent(ctx context.Context) error {
	staticKeys := []string{
		"static:phenotype_definitions_all",
		"static:risk_models_all", 
		"static:treatment_preference_templates_all",
		"static:institutional_rules_all",
	}
	
	for _, key := range staticKeys {
		if _, found := cc.Get(ctx, key); !found {
			// Content not available, log warning
			fmt.Printf("Warning: Static content not available for key: %s\n", key)
		}
	}
	
	return nil
}

// UpdateStaticContentVersions updates version mappings for cache busting
func (cc *CDNCache) UpdateStaticContentVersions(versions map[string]string) {
	for path, version := range versions {
		cc.versionMap[path] = version
	}
}

// GetContentVersions returns current content versions
func (cc *CDNCache) GetContentVersions() map[string]string {
	versions := make(map[string]string)
	for path, version := range cc.versionMap {
		versions[path] = version
	}
	return versions
}

// Cache warming for static content

// WarmStaticContent warms CDN cache with frequently accessed static content
func (cc *CDNCache) WarmStaticContent(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if _, found := cc.Get(ctx, key); !found {
			// Log cache miss for static content
			fmt.Printf("CDN cache miss for static content: %s\n", key)
		}
	}
	return nil
}

// Management methods

// PurgeContent purges content from CDN
func (cc *CDNCache) PurgeContent(ctx context.Context, paths []string) error {
	// In production, this would call CDN purge APIs
	// For now, we'll update versions to bust cache
	
	for _, path := range paths {
		currentVersion := cc.getVersionForPath(path)
		newVersion := cc.generateNewVersion(currentVersion)
		cc.versionMap[path] = newVersion
	}
	
	return nil
}

// UpdateCacheHeaders updates HTTP cache headers for CDN
func (cc *CDNCache) UpdateCacheHeaders(headers map[string]string) {
	for header, value := range headers {
		cc.config.CacheHeaders[header] = value
	}
}

// Statistics and monitoring

// GetDetailedStats returns detailed CDN cache statistics
func (cc *CDNCache) GetDetailedStats() map[string]interface{} {
	totalReqs := atomic.LoadInt64(&cc.requests)
	hits := atomic.LoadInt64(&cc.hits)
	misses := atomic.LoadInt64(&cc.misses)
	errors := atomic.LoadInt64(&cc.errors)
	
	hitRate := 0.0
	if totalReqs > 0 {
		hitRate = float64(hits) / float64(totalReqs)
	}
	
	errorRate := 0.0
	if totalReqs > 0 {
		errorRate = float64(errors) / float64(totalReqs)
	}
	
	return map[string]interface{}{
		"hit_rate":      hitRate,
		"miss_rate":     1.0 - hitRate,
		"error_rate":    errorRate,
		"total_requests": totalReqs,
		"cache_hits":    hits,
		"cache_misses":  misses,
		"errors":        errors,
		"base_url":      cc.config.BaseURL,
		"version_prefix": cc.config.VersionPrefix,
		"content_versions": cc.GetContentVersions(),
	}
}

// ResetStats resets CDN cache statistics
func (cc *CDNCache) ResetStats() {
	atomic.StoreInt64(&cc.hits, 0)
	atomic.StoreInt64(&cc.misses, 0)
	atomic.StoreInt64(&cc.errors, 0)
	atomic.StoreInt64(&cc.requests, 0)
}

// generateNewVersion generates new version string
func (cc *CDNCache) generateNewVersion(currentVersion string) string {
	return fmt.Sprintf("v%d", time.Now().Unix())
}

// Clinical domain-specific helpers

// IsStaticClinicalContent checks if key represents static clinical content
func (cc *CDNCache) IsStaticClinicalContent(key string) bool {
	staticPrefixes := []string{
		"phenotype_definition:",
		"risk_model:",
		"treatment_preference_template:",
		"institutional_rule:",
		"static:",
	}
	
	for _, prefix := range staticPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// GetClinicalContentCategories returns available clinical content categories
func (cc *CDNCache) GetClinicalContentCategories() []string {
	return []string{
		"phenotype_definitions",
		"risk_models",
		"treatment_preferences",
		"institutional_rules",
		"clinical_guidelines",
		"evidence_summaries",
	}
}

// ValidateClinicalContent validates clinical content before caching
func (cc *CDNCache) ValidateClinicalContent(content interface{}) error {
	// Basic validation for clinical content
	switch content.(type) {
	case map[string]interface{}, []interface{}, string, []byte:
		return nil // Valid content types
	default:
		return fmt.Errorf("unsupported clinical content type: %T", content)
	}
}

// Edge case handling

// HandleCDNFailure handles CDN failures gracefully
func (cc *CDNCache) HandleCDNFailure(ctx context.Context, key string, fallbackLoader func() (interface{}, error)) (interface{}, error) {
	// Try CDN first
	if data, found := cc.Get(ctx, key); found {
		return data, nil
	}
	
	// CDN failed or miss, use fallback loader
	if fallbackLoader != nil {
		atomic.AddInt64(&cc.errors, 1)
		return fallbackLoader()
	}
	
	return nil, fmt.Errorf("CDN cache miss and no fallback available for key: %s", key)
}

// GetCDNPerformanceMetrics returns performance metrics specific to CDN
func (cc *CDNCache) GetCDNPerformanceMetrics(ctx context.Context) map[string]interface{} {
	latency := cc.GetCDNLatency(ctx)
	
	return map[string]interface{}{
		"avg_latency_ms": latency.Milliseconds(),
		"is_healthy":     cc.IsHealthy(ctx),
		"hit_rate":       cc.GetStats().HitRate,
		"total_requests": atomic.LoadInt64(&cc.requests),
		"error_rate":     float64(atomic.LoadInt64(&cc.errors)) / float64(atomic.LoadInt64(&cc.requests)),
	}
}

// Configuration management

// UpdateCDNConfig updates CDN configuration
func (cc *CDNCache) UpdateCDNConfig(newBaseURL string, newHeaders map[string]string) {
	cc.config.BaseURL = newBaseURL
	
	if newHeaders != nil {
		for header, value := range newHeaders {
			cc.config.CacheHeaders[header] = value
		}
	}
}

// GetCDNConfig returns current CDN configuration
func (cc *CDNCache) GetCDNConfig() map[string]interface{} {
	return map[string]interface{}{
		"base_url":       cc.config.BaseURL,
		"version_prefix": cc.config.VersionPrefix,
		"cache_headers":  cc.config.CacheHeaders,
		"static_paths":   cc.config.StaticPaths,
		"client_timeout": cc.client.Timeout.String(),
	}
}