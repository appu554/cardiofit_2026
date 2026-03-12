// Package cache provides Redis caching capabilities for KB-12
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"kb-12-ordersets-careplans/internal/config"
)

// RedisCache is an alias for Cache for backward compatibility
type RedisCache = Cache

// Cache wraps Redis client with domain-specific caching methods
type Cache struct {
	client *redis.Client
	config *config.RedisConfig
	log    *logrus.Entry
}

// CacheKey prefixes for different entity types
const (
	PrefixOrderSetTemplate  = "kb12:template:orderset:"
	PrefixCarePlanTemplate  = "kb12:template:careplan:"
	PrefixOrderSetInstance  = "kb12:instance:orderset:"
	PrefixCarePlanInstance  = "kb12:instance:careplan:"
	PrefixPatientOrderSets  = "kb12:patient:ordersets:"
	PrefixPatientCarePlans  = "kb12:patient:careplans:"
	PrefixCPOEDraft         = "kb12:cpoe:draft:"
	PrefixConstraintStatus  = "kb12:constraint:"
	PrefixCDSHook           = "kb12:cds:hook:"
	PrefixTemplateSearch    = "kb12:search:templates:"
)

// NewCache creates a new Redis cache client
func NewCache(cfg *config.RedisConfig) (*Cache, error) {
	log := logrus.WithField("component", "cache")

	var client *redis.Client

	if cfg.URL != "" {
		opts, err := redis.ParseURL(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
		}
		opts.MaxRetries = cfg.MaxRetries
		opts.PoolSize = cfg.PoolSize
		opts.MinIdleConns = cfg.MinIdleConns
		opts.DialTimeout = cfg.DialTimeout
		opts.ReadTimeout = cfg.ReadTimeout
		opts.WriteTimeout = cfg.WriteTimeout
		opts.PoolTimeout = cfg.PoolTimeout
		client = redis.NewClient(opts)
	} else {
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.GetRedisAddr(),
			Password:     cfg.Password,
			DB:           cfg.Database,
			MaxRetries:   cfg.MaxRetries,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConns,
			DialTimeout:  cfg.DialTimeout,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			PoolTimeout:  cfg.PoolTimeout,
		})
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.WithField("addr", cfg.GetRedisAddr()).Info("Successfully connected to Redis")

	return &Cache{
		client: client,
		config: cfg,
		log:    log,
	}, nil
}

// Health performs a Redis health check
func (c *Cache) Health(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// HealthCheck returns detailed health status
func (c *Cache) HealthCheck(ctx context.Context) *HealthStatus {
	start := time.Now()
	err := c.client.Ping(ctx).Err()
	latency := time.Since(start)

	if err != nil {
		return &HealthStatus{
			Status:  "unhealthy",
			Error:   err.Error(),
			Latency: latency,
		}
	}

	poolStats := c.client.PoolStats()
	return &HealthStatus{
		Status:     "healthy",
		Latency:    latency,
		Hits:       poolStats.Hits,
		Misses:     poolStats.Misses,
		Timeouts:   poolStats.Timeouts,
		TotalConns: poolStats.TotalConns,
		IdleConns:  poolStats.IdleConns,
		StaleConns: poolStats.StaleConns,
	}
}

// HealthStatus contains detailed Redis health information
type HealthStatus struct {
	Status     string        `json:"status"`
	Error      string        `json:"error,omitempty"`
	Latency    time.Duration `json:"latency_ms"`
	Hits       uint32        `json:"hits"`
	Misses     uint32        `json:"misses"`
	Timeouts   uint32        `json:"timeouts"`
	TotalConns uint32        `json:"total_conns"`
	IdleConns  uint32        `json:"idle_conns"`
	StaleConns uint32        `json:"stale_conns"`
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	c.log.Info("Closing Redis connection")
	return c.client.Close()
}

// Generic cache operations

// Get retrieves a value by key
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// GetJSON retrieves and unmarshals a JSON value
func (c *Cache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Set stores a value with expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// SetJSON marshals and stores a JSON value
func (c *Cache) SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes a key
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

// Expire sets expiration on a key
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// TTL returns the remaining TTL of a key
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// Keys returns all keys matching a pattern
func (c *Cache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

// DeletePattern deletes all keys matching a pattern
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

// Order Set Template caching

// GetOrderSetTemplate retrieves a cached order set template
func (c *Cache) GetOrderSetTemplate(ctx context.Context, templateID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixOrderSetTemplate+templateID).Bytes()
}

// SetOrderSetTemplate caches an order set template
func (c *Cache) SetOrderSetTemplate(ctx context.Context, templateID string, data []byte) error {
	return c.client.Set(ctx, PrefixOrderSetTemplate+templateID, data, c.config.TemplateTTL).Err()
}

// InvalidateOrderSetTemplate removes a cached order set template
func (c *Cache) InvalidateOrderSetTemplate(ctx context.Context, templateID string) error {
	return c.client.Del(ctx, PrefixOrderSetTemplate+templateID).Err()
}

// Care Plan Template caching

// GetCarePlanTemplate retrieves a cached care plan template
func (c *Cache) GetCarePlanTemplate(ctx context.Context, planID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixCarePlanTemplate+planID).Bytes()
}

// SetCarePlanTemplate caches a care plan template
func (c *Cache) SetCarePlanTemplate(ctx context.Context, planID string, data []byte) error {
	return c.client.Set(ctx, PrefixCarePlanTemplate+planID, data, c.config.TemplateTTL).Err()
}

// InvalidateCarePlanTemplate removes a cached care plan template
func (c *Cache) InvalidateCarePlanTemplate(ctx context.Context, planID string) error {
	return c.client.Del(ctx, PrefixCarePlanTemplate+planID).Err()
}

// Order Set Instance caching

// GetOrderSetInstance retrieves a cached order set instance
func (c *Cache) GetOrderSetInstance(ctx context.Context, instanceID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixOrderSetInstance+instanceID).Bytes()
}

// SetOrderSetInstance caches an order set instance
func (c *Cache) SetOrderSetInstance(ctx context.Context, instanceID string, data []byte) error {
	return c.client.Set(ctx, PrefixOrderSetInstance+instanceID, data, c.config.OrderSetTTL).Err()
}

// InvalidateOrderSetInstance removes a cached order set instance
func (c *Cache) InvalidateOrderSetInstance(ctx context.Context, instanceID string) error {
	return c.client.Del(ctx, PrefixOrderSetInstance+instanceID).Err()
}

// Care Plan Instance caching

// GetCarePlanInstance retrieves a cached care plan instance
func (c *Cache) GetCarePlanInstance(ctx context.Context, instanceID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixCarePlanInstance+instanceID).Bytes()
}

// SetCarePlanInstance caches a care plan instance
func (c *Cache) SetCarePlanInstance(ctx context.Context, instanceID string, data []byte) error {
	return c.client.Set(ctx, PrefixCarePlanInstance+instanceID, data, c.config.CarePlanTTL).Err()
}

// InvalidateCarePlanInstance removes a cached care plan instance
func (c *Cache) InvalidateCarePlanInstance(ctx context.Context, instanceID string) error {
	return c.client.Del(ctx, PrefixCarePlanInstance+instanceID).Err()
}

// Patient-specific caching

// GetPatientOrderSets retrieves cached order sets for a patient
func (c *Cache) GetPatientOrderSets(ctx context.Context, patientID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixPatientOrderSets+patientID).Bytes()
}

// SetPatientOrderSets caches order sets for a patient
func (c *Cache) SetPatientOrderSets(ctx context.Context, patientID string, data []byte) error {
	return c.client.Set(ctx, PrefixPatientOrderSets+patientID, data, c.config.OrderSetTTL).Err()
}

// InvalidatePatientOrderSets removes cached order sets for a patient
func (c *Cache) InvalidatePatientOrderSets(ctx context.Context, patientID string) error {
	return c.client.Del(ctx, PrefixPatientOrderSets+patientID).Err()
}

// GetPatientCarePlans retrieves cached care plans for a patient
func (c *Cache) GetPatientCarePlans(ctx context.Context, patientID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixPatientCarePlans+patientID).Bytes()
}

// SetPatientCarePlans caches care plans for a patient
func (c *Cache) SetPatientCarePlans(ctx context.Context, patientID string, data []byte) error {
	return c.client.Set(ctx, PrefixPatientCarePlans+patientID, data, c.config.CarePlanTTL).Err()
}

// InvalidatePatientCarePlans removes cached care plans for a patient
func (c *Cache) InvalidatePatientCarePlans(ctx context.Context, patientID string) error {
	return c.client.Del(ctx, PrefixPatientCarePlans+patientID).Err()
}

// CPOE Draft caching

// GetCPOEDraft retrieves a cached CPOE draft session
func (c *Cache) GetCPOEDraft(ctx context.Context, sessionID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixCPOEDraft+sessionID).Bytes()
}

// SetCPOEDraft caches a CPOE draft session with 24-hour TTL
func (c *Cache) SetCPOEDraft(ctx context.Context, sessionID string, data []byte) error {
	return c.client.Set(ctx, PrefixCPOEDraft+sessionID, data, 24*time.Hour).Err()
}

// InvalidateCPOEDraft removes a cached CPOE draft session
func (c *Cache) InvalidateCPOEDraft(ctx context.Context, sessionID string) error {
	return c.client.Del(ctx, PrefixCPOEDraft+sessionID).Err()
}

// Constraint Status caching (for time-critical protocols)

// GetConstraintStatus retrieves cached constraint status
func (c *Cache) GetConstraintStatus(ctx context.Context, instanceID string) ([]byte, error) {
	return c.client.Get(ctx, PrefixConstraintStatus+instanceID).Bytes()
}

// SetConstraintStatus caches constraint status with short TTL for freshness
func (c *Cache) SetConstraintStatus(ctx context.Context, instanceID string, data []byte) error {
	return c.client.Set(ctx, PrefixConstraintStatus+instanceID, data, 5*time.Minute).Err()
}

// InvalidateConstraintStatus removes cached constraint status
func (c *Cache) InvalidateConstraintStatus(ctx context.Context, instanceID string) error {
	return c.client.Del(ctx, PrefixConstraintStatus+instanceID).Err()
}

// Template Search caching

// GetTemplateSearch retrieves cached search results
func (c *Cache) GetTemplateSearch(ctx context.Context, queryHash string) ([]byte, error) {
	return c.client.Get(ctx, PrefixTemplateSearch+queryHash).Bytes()
}

// SetTemplateSearch caches search results with short TTL
func (c *Cache) SetTemplateSearch(ctx context.Context, queryHash string, data []byte) error {
	return c.client.Set(ctx, PrefixTemplateSearch+queryHash, data, 15*time.Minute).Err()
}

// InvalidateAllTemplateSearches removes all cached template search results
func (c *Cache) InvalidateAllTemplateSearches(ctx context.Context) error {
	return c.DeletePattern(ctx, PrefixTemplateSearch+"*")
}

// GetClient returns the underlying Redis client for advanced operations
func (c *Cache) GetClient() *redis.Client {
	return c.client
}
