// Package cache provides Redis-based caching for KB-17 Population Registry
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/models"
)

// Cache keys
const (
	KeyPrefixRegistry     = "kb17:registry:"
	KeyPrefixEnrollment   = "kb17:enrollment:"
	KeyPrefixPatient      = "kb17:patient:"
	KeyPrefixStats        = "kb17:stats:"
	KeyPrefixEligibility  = "kb17:eligibility:"
	KeyAllRegistries      = "kb17:registries:all"
)

// Default TTLs
const (
	TTLRegistry    = 30 * time.Minute
	TTLEnrollment  = 5 * time.Minute
	TTLStats       = 2 * time.Minute
	TTLEligibility = 10 * time.Minute
)

// RedisCache provides caching operations for the registry service
type RedisCache struct {
	client *redis.Client
	logger *logrus.Entry
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client, logger *logrus.Entry) *RedisCache {
	return &RedisCache{
		client: client,
		logger: logger.WithField("component", "cache"),
	}
}

// GetRegistry retrieves a registry from cache
func (c *RedisCache) GetRegistry(ctx context.Context, code models.RegistryCode) (*models.Registry, error) {
	key := KeyPrefixRegistry + string(code)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		c.logger.WithError(err).WithField("key", key).Warn("Failed to get registry from cache")
		return nil, err
	}

	var registry models.Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		c.logger.WithError(err).Warn("Failed to unmarshal cached registry")
		return nil, err
	}

	return &registry, nil
}

// SetRegistry stores a registry in cache
func (c *RedisCache) SetRegistry(ctx context.Context, registry *models.Registry) error {
	key := KeyPrefixRegistry + string(registry.Code)

	data, err := json.Marshal(registry)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, TTLRegistry).Err()
}

// GetAllRegistries retrieves all registries from cache
func (c *RedisCache) GetAllRegistries(ctx context.Context) ([]models.Registry, error) {
	data, err := c.client.Get(ctx, KeyAllRegistries).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var registries []models.Registry
	if err := json.Unmarshal(data, &registries); err != nil {
		return nil, err
	}

	return registries, nil
}

// SetAllRegistries stores all registries in cache
func (c *RedisCache) SetAllRegistries(ctx context.Context, registries []models.Registry) error {
	data, err := json.Marshal(registries)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, KeyAllRegistries, data, TTLRegistry).Err()
}

// GetEnrollment retrieves an enrollment from cache
func (c *RedisCache) GetEnrollment(ctx context.Context, patientID string, registryCode models.RegistryCode) (*models.RegistryPatient, error) {
	key := fmt.Sprintf("%s%s:%s", KeyPrefixEnrollment, patientID, registryCode)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var enrollment models.RegistryPatient
	if err := json.Unmarshal(data, &enrollment); err != nil {
		return nil, err
	}

	return &enrollment, nil
}

// SetEnrollment stores an enrollment in cache
func (c *RedisCache) SetEnrollment(ctx context.Context, enrollment *models.RegistryPatient) error {
	key := fmt.Sprintf("%s%s:%s", KeyPrefixEnrollment, enrollment.PatientID, enrollment.RegistryCode)

	data, err := json.Marshal(enrollment)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, TTLEnrollment).Err()
}

// GetPatientRegistries retrieves patient's registries from cache
func (c *RedisCache) GetPatientRegistries(ctx context.Context, patientID string) ([]models.RegistryPatient, error) {
	key := KeyPrefixPatient + patientID

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var enrollments []models.RegistryPatient
	if err := json.Unmarshal(data, &enrollments); err != nil {
		return nil, err
	}

	return enrollments, nil
}

// SetPatientRegistries stores patient's registries in cache
func (c *RedisCache) SetPatientRegistries(ctx context.Context, patientID string, enrollments []models.RegistryPatient) error {
	key := KeyPrefixPatient + patientID

	data, err := json.Marshal(enrollments)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, TTLEnrollment).Err()
}

// GetStats retrieves registry stats from cache
func (c *RedisCache) GetStats(ctx context.Context, registryCode models.RegistryCode) (*models.RegistryStats, error) {
	key := KeyPrefixStats + string(registryCode)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var stats models.RegistryStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// SetStats stores registry stats in cache
func (c *RedisCache) SetStats(ctx context.Context, stats *models.RegistryStats) error {
	key := KeyPrefixStats + string(stats.RegistryCode)

	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, TTLStats).Err()
}

// GetEligibility retrieves eligibility result from cache
func (c *RedisCache) GetEligibility(ctx context.Context, patientID string, registryCode models.RegistryCode) (*models.CriteriaEvaluationResult, error) {
	key := fmt.Sprintf("%s%s:%s", KeyPrefixEligibility, patientID, registryCode)

	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result models.CriteriaEvaluationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetEligibility stores eligibility result in cache
func (c *RedisCache) SetEligibility(ctx context.Context, result *models.CriteriaEvaluationResult) error {
	key := fmt.Sprintf("%s%s:%s", KeyPrefixEligibility, result.PatientID, result.RegistryCode)

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, TTLEligibility).Err()
}

// InvalidateRegistry removes a registry from cache
func (c *RedisCache) InvalidateRegistry(ctx context.Context, code models.RegistryCode) error {
	key := KeyPrefixRegistry + string(code)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return err
	}
	// Also invalidate the all registries cache
	return c.client.Del(ctx, KeyAllRegistries).Err()
}

// InvalidateEnrollment removes an enrollment from cache
func (c *RedisCache) InvalidateEnrollment(ctx context.Context, patientID string, registryCode models.RegistryCode) error {
	key := fmt.Sprintf("%s%s:%s", KeyPrefixEnrollment, patientID, registryCode)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return err
	}
	// Also invalidate patient registries cache
	patientKey := KeyPrefixPatient + patientID
	return c.client.Del(ctx, patientKey).Err()
}

// InvalidatePatient removes all cached data for a patient
func (c *RedisCache) InvalidatePatient(ctx context.Context, patientID string) error {
	// Get all keys for this patient
	pattern := fmt.Sprintf("kb17:*%s*", patientID)
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// InvalidateStats removes stats cache for a registry
func (c *RedisCache) InvalidateStats(ctx context.Context, registryCode models.RegistryCode) error {
	key := KeyPrefixStats + string(registryCode)
	return c.client.Del(ctx, key).Err()
}

// InvalidateAllStats removes all stats cache
func (c *RedisCache) InvalidateAllStats(ctx context.Context) error {
	pattern := KeyPrefixStats + "*"
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Health checks Redis connection health
func (c *RedisCache) Health(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}
