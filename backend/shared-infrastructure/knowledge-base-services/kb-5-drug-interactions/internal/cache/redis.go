package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"kb-drug-interactions/internal/config"
)

// CacheClient wraps Redis client with KB-5 specific functionality
type CacheClient struct {
	client *redis.Client
	config *config.Config
	ctx    context.Context
}

// NewCacheClient creates a new Redis cache client
func NewCacheClient(cfg *config.Config) (*CacheClient, error) {
	// Parse Redis URL
	opts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override password if provided
	if cfg.Redis.Password != "" {
		opts.Password = cfg.Redis.Password
	}

	// Override DB if provided
	if cfg.Redis.DB > 0 {
		opts.DB = cfg.Redis.DB
	}

	// Create Redis client
	rdb := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Connected to Redis successfully (DB: %d)", opts.DB)

	return &CacheClient{
		client: rdb,
		config: cfg,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (c *CacheClient) Close() error {
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis connection: %w", err)
	}
	log.Println("Redis connection closed")
	return nil
}

// HealthCheck performs a Redis health check
func (c *CacheClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis health check failed: %w", err)
	}
	return nil
}

// Cache keys and TTL configurations
const (
	InteractionCheckCacheKeyPrefix = "kb5:interaction_check:"
	InteractionDetailCacheKeyPrefix = "kb5:interaction_detail:"
	DrugSynonymCacheKeyPrefix      = "kb5:drug_synonym:"
	PatientAlertCacheKeyPrefix     = "kb5:patient_alert:"
	InteractionRuleCacheKeyPrefix  = "kb5:interaction_rule:"
	CDSConfigCacheKeyPrefix        = "kb5:cds_config:"
	AnalyticsCacheKeyPrefix        = "kb5:analytics:"
	DrugAlternativesCacheKeyPrefix = "kb5:drug_alternatives:"
	AllInteractionsCacheKeyPrefix  = "kb5:all_interactions:"
	InteractionMatrixCacheKeyPrefix = "kb5:interaction_matrix:"

	DefaultInteractionCheckTTL = 1 * time.Hour    // Interaction checks cached for 1 hour
	DefaultInteractionDetailTTL = 6 * time.Hour   // Individual interactions cached longer
	DefaultDrugSynonymTTL      = 24 * time.Hour   // Drug mappings rarely change
	DefaultPatientAlertTTL     = 30 * time.Minute // Patient alerts change frequently
	DefaultInteractionRuleTTL  = 12 * time.Hour   // Rules change infrequently
	DefaultCDSConfigTTL        = 24 * time.Hour   // CDS configs stable
	DefaultAnalyticsTTL        = 15 * time.Minute // Analytics for dashboards
	DefaultDrugAlternativesTTL = 2 * time.Hour    // Alternative suggestions
	DefaultAllInteractionsTTL  = 6 * time.Hour    // All interactions matrix data
)

// Interaction Check caching methods

// InteractionCheckCacheKey generates cache key for interaction checks
func InteractionCheckCacheKey(drugCodes string) string {
	return fmt.Sprintf("%s%s", InteractionCheckCacheKeyPrefix, drugCodes)
}

// SetInteractionCheck caches interaction check results
func (c *CacheClient) SetInteractionCheck(key string, response interface{}) error {
	return c.setJSON(key, response, DefaultInteractionCheckTTL)
}

// SetInteractionCheckWithTTL caches interaction check results with custom TTL
func (c *CacheClient) SetInteractionCheckWithTTL(key string, response interface{}, ttl time.Duration) error {
	return c.setJSON(key, response, ttl)
}

// GetInteractionCheck retrieves cached interaction check results
func (c *CacheClient) GetInteractionCheck(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Interaction Detail caching methods

// InteractionDetailCacheKey generates cache key for interaction details
func InteractionDetailCacheKey(interactionID string) string {
	return fmt.Sprintf("%s%s", InteractionDetailCacheKeyPrefix, interactionID)
}

// SetInteractionDetail caches interaction detail
func (c *CacheClient) SetInteractionDetail(key string, interaction interface{}) error {
	return c.setJSON(key, interaction, DefaultInteractionDetailTTL)
}

// GetInteractionDetail retrieves cached interaction detail
func (c *CacheClient) GetInteractionDetail(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Drug Synonym caching methods

// DrugSynonymCacheKey generates cache key for drug synonyms
func DrugSynonymCacheKey(drugCode string) string {
	return fmt.Sprintf("%s%s", DrugSynonymCacheKeyPrefix, drugCode)
}

// SetDrugSynonyms caches drug synonym mappings
func (c *CacheClient) SetDrugSynonyms(key string, synonyms interface{}) error {
	return c.setJSON(key, synonyms, DefaultDrugSynonymTTL)
}

// GetDrugSynonyms retrieves cached drug synonyms
func (c *CacheClient) GetDrugSynonyms(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Patient Alert caching methods

// PatientAlertCacheKey generates cache key for patient alerts
func PatientAlertCacheKey(patientID string) string {
	return fmt.Sprintf("%s%s", PatientAlertCacheKeyPrefix, patientID)
}

// PatientAlertHistoryCacheKey generates cache key for patient alert history
func PatientAlertHistoryCacheKey(patientID string, page, limit int) string {
	return fmt.Sprintf("%s%s:history:page:%d:limit:%d", PatientAlertCacheKeyPrefix, patientID, page, limit)
}

// SetPatientAlerts caches patient alerts
func (c *CacheClient) SetPatientAlerts(key string, alerts interface{}) error {
	return c.setJSON(key, alerts, DefaultPatientAlertTTL)
}

// GetPatientAlerts retrieves cached patient alerts
func (c *CacheClient) GetPatientAlerts(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Interaction Rule caching methods

// InteractionRuleCacheKey generates cache key for interaction rules
func InteractionRuleCacheKey(ruleType string) string {
	if ruleType != "" {
		return fmt.Sprintf("%srules:type:%s", InteractionRuleCacheKeyPrefix, ruleType)
	}
	return fmt.Sprintf("%srules:all", InteractionRuleCacheKeyPrefix)
}

// SetInteractionRules caches interaction rules
func (c *CacheClient) SetInteractionRules(key string, rules interface{}) error {
	return c.setJSON(key, rules, DefaultInteractionRuleTTL)
}

// GetInteractionRules retrieves cached interaction rules
func (c *CacheClient) GetInteractionRules(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// CDS Configuration caching methods

// CDSConfigCacheKey generates cache key for CDS configurations
func CDSConfigCacheKey(configName string) string {
	return fmt.Sprintf("%s%s", CDSConfigCacheKeyPrefix, configName)
}

// SetCDSConfig caches CDS configuration
func (c *CacheClient) SetCDSConfig(key string, config interface{}) error {
	return c.setJSON(key, config, DefaultCDSConfigTTL)
}

// GetCDSConfig retrieves cached CDS configuration
func (c *CacheClient) GetCDSConfig(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Analytics caching methods

// AnalyticsCacheKey generates cache key for analytics data
func AnalyticsCacheKey(analyticsType string, startTime, endTime time.Time) string {
	return fmt.Sprintf("%s%s:%d:%d", AnalyticsCacheKeyPrefix, analyticsType, 
		startTime.Unix(), endTime.Unix())
}

// SetAnalytics caches analytics data
func (c *CacheClient) SetAnalytics(key string, analytics interface{}) error {
	return c.setJSON(key, analytics, DefaultAnalyticsTTL)
}

// GetAnalytics retrieves cached analytics
func (c *CacheClient) GetAnalytics(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Drug Alternatives caching methods

// DrugAlternativesCacheKey generates cache key for drug alternatives
func DrugAlternativesCacheKey(drugCode string, indication string) string {
	return fmt.Sprintf("%s%s:indication:%s", DrugAlternativesCacheKeyPrefix, drugCode, indication)
}

// SetDrugAlternatives caches drug alternatives
func (c *CacheClient) SetDrugAlternatives(key string, alternatives interface{}) error {
	return c.setJSON(key, alternatives, DefaultDrugAlternativesTTL)
}

// GetDrugAlternatives retrieves cached drug alternatives
func (c *CacheClient) GetDrugAlternatives(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// Generic caching methods

// setJSON stores a value as JSON in Redis with TTL
func (c *CacheClient) setJSON(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
	}

	if err := c.client.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	return nil
}

// getJSON retrieves a JSON value from Redis and unmarshals it
func (c *CacheClient) getJSON(key string, result interface{}) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	jsonData, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(jsonData), result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON for key %s: %w", key, err)
	}

	return nil
}

// Delete removes a key from cache
func (c *CacheClient) Delete(key string) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete cache key %s: %w", key, err)
	}

	return nil
}

// DeletePattern removes all keys matching a pattern
func (c *CacheClient) DeletePattern(pattern string) error {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// Get all keys matching pattern
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all matching keys
	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete keys for pattern %s: %w", pattern, err)
	}

	log.Printf("Deleted %d cache keys matching pattern: %s", len(keys), pattern)
	return nil
}

// Exists checks if a key exists in cache
func (c *CacheClient) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	exists, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence of key %s: %w", key, err)
	}

	return exists > 0, nil
}

// SetTTL updates the TTL for a key
func (c *CacheClient) SetTTL(key string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL for key %s: %w", key, err)
	}

	return nil
}

// GetTTL returns the remaining TTL for a key
func (c *CacheClient) GetTTL(key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL for key %s: %w", key, err)
	}

	return ttl, nil
}

// Cache invalidation methods for drug interactions

// InvalidatePatientAlerts invalidates all cached alerts for a patient
func (c *CacheClient) InvalidatePatientAlerts(patientID string) error {
	patterns := []string{
		fmt.Sprintf("%s%s*", PatientAlertCacheKeyPrefix, patientID),
	}

	for _, pattern := range patterns {
		if err := c.DeletePattern(pattern); err != nil {
			return fmt.Errorf("failed to invalidate cache pattern %s: %w", pattern, err)
		}
	}

	return nil
}

// InvalidateInteractionChecks invalidates interaction check caches
func (c *CacheClient) InvalidateInteractionChecks() error {
	return c.DeletePattern(fmt.Sprintf("%s*", InteractionCheckCacheKeyPrefix))
}

// InvalidateDrugSynonyms invalidates drug synonym caches
func (c *CacheClient) InvalidateDrugSynonyms() error {
	return c.DeletePattern(fmt.Sprintf("%s*", DrugSynonymCacheKeyPrefix))
}

// InvalidateInteractionRules invalidates interaction rule caches
func (c *CacheClient) InvalidateInteractionRules() error {
	return c.DeletePattern(fmt.Sprintf("%s*", InteractionRuleCacheKeyPrefix))
}

// Real-time caching for high-frequency operations

// SetActiveInteractionAlert caches an active interaction alert with short TTL
func (c *CacheClient) SetActiveInteractionAlert(alertID string, alert interface{}) error {
	key := fmt.Sprintf("%sactive:%s", PatientAlertCacheKeyPrefix, alertID)
	return c.setJSON(key, alert, 5*time.Minute) // Short TTL for active monitoring
}

// GetActiveInteractionAlert retrieves an active interaction alert
func (c *CacheClient) GetActiveInteractionAlert(alertID string, result interface{}) error {
	key := fmt.Sprintf("%sactive:%s", PatientAlertCacheKeyPrefix, alertID)
	return c.getJSON(key, result)
}

// SetDrugInteractionStatus caches current drug interaction status for a patient
func (c *CacheClient) SetDrugInteractionStatus(patientID string, status interface{}) error {
	key := fmt.Sprintf("%sstatus:%s", PatientAlertCacheKeyPrefix, patientID)
	return c.setJSON(key, status, 10*time.Minute) // Medium TTL for status
}

// GetDrugInteractionStatus retrieves current drug interaction status
func (c *CacheClient) GetDrugInteractionStatus(patientID string, result interface{}) error {
	key := fmt.Sprintf("%sstatus:%s", PatientAlertCacheKeyPrefix, patientID)
	return c.getJSON(key, result)
}

// Batch operations for improved performance

// MGet retrieves multiple keys at once
func (c *CacheClient) MGet(keys []string) ([]string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	result, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple keys: %w", err)
	}

	values := make([]string, len(result))
	for i, val := range result {
		if val != nil {
			values[i] = val.(string)
		}
	}

	return values, nil
}

// SetMultiple sets multiple key-value pairs with same TTL
func (c *CacheClient) SetMultiple(keyValues map[string]interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
	defer cancel()

	// Use pipeline for better performance
	pipe := c.client.Pipeline()
	
	for key, value := range keyValues {
		jsonData, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, jsonData, ttl)
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline: %w", err)
	}

	return nil
}

// GetStats returns cache statistics
func (c *CacheClient) GetStats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	info := c.client.Info(ctx, "memory", "stats", "keyspace").Val()
	
	return map[string]interface{}{
		"redis_info": info,
		"connection_pool_stats": c.client.PoolStats(),
	}
}

// Warming methods for frequently accessed data

// WarmInteractionCache pre-loads frequently checked drug combinations
func (c *CacheClient) WarmInteractionCache(drugCombinations []string) error {
	for i, combo := range drugCombinations {
		// This is a placeholder - in practice, you'd pre-compute interaction results
		key := fmt.Sprintf("%swarm:%d", InteractionCheckCacheKeyPrefix, i)
		if err := c.setJSON(key, combo, DefaultInteractionCheckTTL); err != nil {
			log.Printf("Failed to warm cache for drug combination %d: %v", i, err)
		}
	}
	return nil
}

// Matrix caching methods for enhanced performance

// AllInteractionsCacheKey generates cache key for all interactions data
func AllInteractionsCacheKey() string {
	return fmt.Sprintf("%sall", AllInteractionsCacheKeyPrefix)
}

// SetAllInteractionsWithTTL caches all interactions data with custom TTL
func (c *CacheClient) SetAllInteractionsWithTTL(key string, interactions interface{}, ttl time.Duration) error {
	return c.setJSON(key, interactions, ttl)
}

// GetAllInteractions retrieves cached all interactions data
func (c *CacheClient) GetAllInteractions(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// InteractionMatrixCacheKey generates cache key for interaction matrix
func InteractionMatrixCacheKey(matrixType string) string {
	return fmt.Sprintf("%s%s", InteractionMatrixCacheKeyPrefix, matrixType)
}

// SetInteractionMatrix caches interaction matrix data
func (c *CacheClient) SetInteractionMatrix(key string, matrix interface{}) error {
	return c.setJSON(key, matrix, DefaultAllInteractionsTTL)
}

// GetInteractionMatrix retrieves cached interaction matrix data
func (c *CacheClient) GetInteractionMatrix(key string, result interface{}) error {
	return c.getJSON(key, result)
}

// InvalidateMatrixCache invalidates all matrix-related caches
func (c *CacheClient) InvalidateMatrixCache() error {
	patterns := []string{
		fmt.Sprintf("%s*", AllInteractionsCacheKeyPrefix),
		fmt.Sprintf("%s*", InteractionMatrixCacheKeyPrefix),
	}

	for _, pattern := range patterns {
		if err := c.DeletePattern(pattern); err != nil {
			return fmt.Errorf("failed to invalidate matrix cache pattern %s: %w", pattern, err)
		}
	}

	return nil
}

// Custom errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)