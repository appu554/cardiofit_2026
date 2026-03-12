package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Client wraps the Redis client and provides healthcare-specific caching methods
type Client struct {
	client *redis.Client
	logger *zap.Logger
}

// NewClient creates a new Redis client
func NewClient(redisURL string) (*Client, error) {
	logger, _ := zap.NewProduction()

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure for healthcare workloads
	opts.PoolSize = 10
	opts.MaxRetries = 3
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Successfully connected to Redis")

	return &Client{
		client: client,
		logger: logger,
	}, nil
}

// GetClient returns the underlying Redis client for advanced operations
func (c *Client) GetClient() *redis.Client {
	return c.client
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.client.Close()
}

// Ping checks Redis connectivity
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Recipe caching methods
func (c *Client) CacheRecipe(ctx context.Context, protocolID string, recipe interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("recipe:%s", protocolID)
	data, err := json.Marshal(recipe)
	if err != nil {
		return fmt.Errorf("failed to marshal recipe: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.logger.Error("Failed to cache recipe", zap.String("protocol_id", protocolID), zap.Error(err))
		return fmt.Errorf("failed to cache recipe: %w", err)
	}

	c.logger.Debug("Recipe cached successfully", zap.String("protocol_id", protocolID))
	return nil
}

func (c *Client) GetCachedRecipe(ctx context.Context, protocolID string, dest interface{}) error {
	key := fmt.Sprintf("recipe:%s", protocolID)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to get cached recipe: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached recipe: %w", err)
	}

	return nil
}

func (c *Client) InvalidateRecipe(ctx context.Context, protocolID string) error {
	key := fmt.Sprintf("recipe:%s", protocolID)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate recipe cache: %w", err)
	}
	return nil
}

// Clinical calculation caching
func (c *Client) CacheCalculation(ctx context.Context, calculationKey string, result interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("calc:%s", calculationKey)
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal calculation: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache calculation: %w", err)
	}

	return nil
}

func (c *Client) GetCachedCalculation(ctx context.Context, calculationKey string, dest interface{}) error {
	key := fmt.Sprintf("calc:%s", calculationKey)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to get cached calculation: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached calculation: %w", err)
	}

	return nil
}

// Patient context caching
func (c *Client) CachePatientContext(ctx context.Context, patientID string, context interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("patient_ctx:%s", patientID)
	data, err := json.Marshal(context)
	if err != nil {
		return fmt.Errorf("failed to marshal patient context: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to cache patient context: %w", err)
	}

	return nil
}

func (c *Client) GetCachedPatientContext(ctx context.Context, patientID string, dest interface{}) error {
	key := fmt.Sprintf("patient_ctx:%s", patientID)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to get cached patient context: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached patient context: %w", err)
	}

	return nil
}

// Session management for clinical workflows
func (c *Client) CreateSession(ctx context.Context, sessionID, userID string, sessionData interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	
	session := SessionData{
		ID:        sessionID,
		UserID:    userID,
		Data:      sessionData,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (c *Client) GetSession(ctx context.Context, sessionID string) (*SessionData, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session SessionData
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

func (c *Client) ExtendSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	if err := c.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}
	return nil
}

func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// Rate limiting for clinical operations
func (c *Client) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)
	
	current, err := c.client.Get(ctx, rateLimitKey).Int()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("failed to get rate limit: %w", err)
	}

	if current >= limit {
		return false, nil // Rate limit exceeded
	}

	// Increment counter
	pipe := c.client.Pipeline()
	pipe.Incr(ctx, rateLimitKey)
	pipe.Expire(ctx, rateLimitKey, window)
	
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("failed to update rate limit: %w", err)
	}

	return true, nil
}

// Distributed locking for critical operations
func (c *Client) AcquireLock(ctx context.Context, lockKey string, ttl time.Duration) (*DistributedLock, error) {
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())
	key := fmt.Sprintf("lock:%s", lockKey)

	acquired, err := c.client.SetNX(ctx, key, lockValue, ttl).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !acquired {
		return nil, ErrLockNotAcquired
	}

	return &DistributedLock{
		client: c.client,
		key:    key,
		value:  lockValue,
	}, nil
}

// Health check for monitoring
type RedisHealth struct {
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time"`
	Memory       string        `json:"memory,omitempty"`
	Connections  string        `json:"connections,omitempty"`
	Error        string        `json:"error,omitempty"`
}

func (c *Client) HealthCheck(ctx context.Context) *RedisHealth {
	startTime := time.Now()
	
	health := &RedisHealth{
		Status: "healthy",
	}

	// Test basic connectivity
	if err := c.Ping(ctx); err != nil {
		health.Status = "unhealthy"
		health.Error = err.Error()
		health.ResponseTime = time.Since(startTime)
		return health
	}

	// Get Redis info
	_, err := c.client.Info(ctx, "memory", "clients").Result()
	if err != nil {
		health.Status = "degraded"
		health.Error = fmt.Sprintf("failed to get Redis info: %v", err)
	} else {
		// Parse memory and connection info (simplified)
		health.Memory = "available"
		health.Connections = "available"
	}

	health.ResponseTime = time.Since(startTime)
	return health
}

// Supporting types
type SessionData struct {
	ID        string      `json:"id"`
	UserID    string      `json:"user_id"`
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

type DistributedLock struct {
	client *redis.Client
	key    string
	value  string
}

func (lock *DistributedLock) Release(ctx context.Context) error {
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end`
	
	result, err := lock.client.Eval(ctx, script, []string{lock.key}, lock.value).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	
	if result.(int64) != 1 {
		return ErrLockNotOwned
	}
	
	return nil
}

// Custom errors
var (
	ErrCacheMiss        = fmt.Errorf("cache miss")
	ErrSessionNotFound  = fmt.Errorf("session not found")
	ErrLockNotAcquired  = fmt.Errorf("lock not acquired")
	ErrLockNotOwned     = fmt.Errorf("lock not owned")
)