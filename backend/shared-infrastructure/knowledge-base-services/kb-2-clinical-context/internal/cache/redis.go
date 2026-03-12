package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"kb-clinical-context/internal/config"
)

type CacheClient struct {
	client *redis.Client
	config *config.Config
}

func NewCacheClient(cfg *config.Config) (*CacheClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Address,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.Database,
	})

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &CacheClient{
		client: rdb,
		config: cfg,
	}, nil
}

func (c *CacheClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := c.client.Ping(ctx).Result()
	return err
}

func (c *CacheClient) Close() error {
	return c.client.Close()
}

// Patient Context Caching

func (c *CacheClient) CachePatientContext(patientID string, ctxData interface{}) error {
	key := fmt.Sprintf("context:patient:%s", patientID)
	
	data, err := json.Marshal(ctxData)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Cache for 1 hour
	return c.client.Set(ctx, key, data, time.Hour).Err()
}

func (c *CacheClient) GetPatientContext(patientID string) ([]byte, error) {
	key := fmt.Sprintf("context:patient:%s", patientID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		return nil, fmt.Errorf("failed to get context from cache: %w", err)
	}

	return []byte(data), nil
}

func (c *CacheClient) InvalidatePatientContext(patientID string) error {
	key := fmt.Sprintf("context:patient:%s", patientID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return c.client.Del(ctx, key).Err()
}

// Phenotype Caching

func (c *CacheClient) CachePhenotypes(phenotypes interface{}) error {
	key := "phenotypes:active"
	
	data, err := json.Marshal(phenotypes)
	if err != nil {
		return fmt.Errorf("failed to marshal phenotypes: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Cache for 6 hours
	return c.client.Set(ctx, key, data, 6*time.Hour).Err()
}

func (c *CacheClient) GetPhenotypes() ([]byte, error) {
	key := "phenotypes:active"
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		return nil, fmt.Errorf("failed to get phenotypes from cache: %w", err)
	}

	return []byte(data), nil
}

func (c *CacheClient) InvalidatePhenotypes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Invalidate all phenotype-related caches
	keys := []string{
		"phenotypes:active",
	}

	return c.client.Del(ctx, keys...).Err()
}

// Risk Assessment Caching

func (c *CacheClient) CacheRiskAssessment(patientID string, riskType string, assessment interface{}) error {
	key := fmt.Sprintf("risk:%s:%s", patientID, riskType)
	
	data, err := json.Marshal(assessment)
	if err != nil {
		return fmt.Errorf("failed to marshal risk assessment: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Cache for 2 hours
	return c.client.Set(ctx, key, data, 2*time.Hour).Err()
}

func (c *CacheClient) GetRiskAssessment(patientID string, riskType string) ([]byte, error) {
	key := fmt.Sprintf("risk:%s:%s", patientID, riskType)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		return nil, fmt.Errorf("failed to get risk assessment from cache: %w", err)
	}

	return []byte(data), nil
}

func (c *CacheClient) InvalidateRiskAssessments(patientID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Use pattern to delete all risk assessments for patient
	pattern := fmt.Sprintf("risk:%s:*", patientID)
	
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}

	return nil
}

// Context Statistics Caching

func (c *CacheClient) CacheContextStats(stats interface{}) error {
	key := "stats:context"
	
	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal context stats: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Cache for 30 minutes
	return c.client.Set(ctx, key, data, 30*time.Minute).Err()
}

func (c *CacheClient) GetContextStats() ([]byte, error) {
	key := "stats:context"
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	data, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Cache miss
	} else if err != nil {
		return nil, fmt.Errorf("failed to get context stats from cache: %w", err)
	}

	return []byte(data), nil
}

// Additional methods required by MultiTierCache

// Get retrieves a value from Redis cache
func (c *CacheClient) Get(ctx context.Context, key string) (interface{}, error) {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	
	var value interface{}
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}
	
	return value, nil
}

// Set stores a value in Redis cache
func (c *CacheClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	
	return c.client.Set(ctx, key, data, ttl).Err()
}

// Delete removes a key from Redis cache
func (c *CacheClient) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// InvalidatePattern removes all keys matching a pattern
func (c *CacheClient) InvalidatePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	
	return nil
}

// Health provides a more generic health check interface
func (c *CacheClient) Health(ctx context.Context) error {
	return c.HealthCheck()
}

// General cache operations

func (c *CacheClient) GetStats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	return map[string]interface{}{
		"info": info,
		"connected": true,
	}
}