package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/config"
)

// Cache key prefixes — namespaced for KB-22.
const (
	SessionPrefix       = "kb22:session:"
	NodePrefix          = "kb22:node:"
	DifferentialPrefix  = "kb22:diff:"
	CalibrationPrefix   = "kb22:cal:"
	ReasoningPrefix     = "kb22:reasoning:"
	ContradictionPrefix = "kb22:contradiction:"
)

// Default TTLs per data type.
const (
	SessionTTL     = 24 * time.Hour
	NodeTTL        = 1 * time.Hour
	DifferentialTTL = 10 * time.Minute
	CalibrationTTL = 30 * time.Minute
)

var ErrCacheMiss = errors.New("cache miss")

type CacheClient struct {
	client *redis.Client
	config *config.Config
	log    *zap.Logger
	ctx    context.Context
}

func NewCacheClient(cfg *config.Config, log *zap.Logger) (*CacheClient, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	if cfg.RedisPassword != "" {
		opts.Password = cfg.RedisPassword
	}
	opts.DB = cfg.RedisDB
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opts)
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Info("redis connected", zap.String("url", cfg.RedisURL))

	return &CacheClient{
		client: client,
		config: cfg,
		log:    log,
		ctx:    ctx,
	}, nil
}

// SetJSON marshals value to JSON and stores with TTL.
func (c *CacheClient) SetJSON(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return c.client.Set(c.ctx, key, data, ttl).Err()
}

// GetJSON retrieves and unmarshals JSON from cache.
func (c *CacheClient) GetJSON(key string, result interface{}) error {
	data, err := c.client.Get(c.ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCacheMiss
		}
		return fmt.Errorf("redis get: %w", err)
	}
	return json.Unmarshal(data, result)
}

func (c *CacheClient) Delete(key string) error {
	return c.client.Del(c.ctx, key).Err()
}

func (c *CacheClient) DeletePattern(pattern string) error {
	iter := c.client.Scan(c.ctx, 0, pattern, 100).Iterator()
	for iter.Next(c.ctx) {
		if err := c.client.Del(c.ctx, iter.Val()).Err(); err != nil {
			c.log.Warn("failed to delete key", zap.String("key", iter.Val()), zap.Error(err))
		}
	}
	return iter.Err()
}

// Session cache operations.
func (c *CacheClient) SetSession(sessionID string, session interface{}) error {
	return c.SetJSON(SessionPrefix+sessionID, session, SessionTTL)
}

func (c *CacheClient) GetSession(sessionID string, result interface{}) error {
	return c.GetJSON(SessionPrefix+sessionID, result)
}

func (c *CacheClient) InvalidateSession(sessionID string) error {
	return c.Delete(SessionPrefix + sessionID)
}

// Node cache operations.
func (c *CacheClient) SetNode(nodeID string, node interface{}) error {
	return c.SetJSON(NodePrefix+nodeID, node, NodeTTL)
}

func (c *CacheClient) GetNode(nodeID string, result interface{}) error {
	return c.GetJSON(NodePrefix+nodeID, result)
}

func (c *CacheClient) InvalidateNodes() error {
	return c.DeletePattern(NodePrefix + "*")
}

// Differential cache operations.
func (c *CacheClient) SetDifferential(sessionID string, diff interface{}) error {
	return c.SetJSON(DifferentialPrefix+sessionID, diff, DifferentialTTL)
}

func (c *CacheClient) GetDifferential(sessionID string, result interface{}) error {
	return c.GetJSON(DifferentialPrefix+sessionID, result)
}

// Reasoning chain cache operations (CTL Panel 4).
// Reasoning steps are accumulated per-session during SubmitAnswer and flushed
// to the DifferentialSnapshot JSONB on session completion.
func (c *CacheClient) SetReasoningChain(sessionID string, chain interface{}) error {
	return c.SetJSON(ReasoningPrefix+sessionID, chain, SessionTTL)
}

func (c *CacheClient) GetReasoningChain(sessionID string, result interface{}) error {
	return c.GetJSON(ReasoningPrefix+sessionID, result)
}

// G17: Contradiction detection cache — tracks which contradiction pairs
// have already fired within a session to avoid duplicate detections.
func (c *CacheClient) SetContradictions(sessionID string, detected interface{}) error {
	return c.SetJSON(ContradictionPrefix+sessionID, detected, SessionTTL)
}

func (c *CacheClient) GetContradictions(sessionID string, result interface{}) error {
	return c.GetJSON(ContradictionPrefix+sessionID, result)
}

// Calibration cache operations.
func (c *CacheClient) SetCalibrationStatus(nodeID string, status interface{}) error {
	return c.SetJSON(CalibrationPrefix+nodeID, status, CalibrationTTL)
}

func (c *CacheClient) GetCalibrationStatus(nodeID string, result interface{}) error {
	return c.GetJSON(CalibrationPrefix+nodeID, result)
}

func (c *CacheClient) Health() error {
	return c.client.Ping(c.ctx).Err()
}

func (c *CacheClient) Close() error {
	return c.client.Close()
}
