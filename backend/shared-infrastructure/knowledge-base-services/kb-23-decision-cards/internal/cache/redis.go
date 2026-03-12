package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
)

// Cache key prefixes — namespaced for KB-23.
const (
	MCUGatePrefix     = "kb23:mcu_gate:"     // per-patient enriched gate
	GateHistoryPrefix = "kb23:gate_history:"  // per-patient gate history array
	PerturbationPrefix = "kb23:perturbation:" // active perturbations
	AdherencePrefix   = "kb23:adherence:"     // KB-21 adherence cache
	TemplatePrefix    = "kb23:template:"      // parsed templates
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

// MCU Gate operations.
func (c *CacheClient) SetMCUGate(patientID string, gate interface{}) error {
	return c.SetJSON(MCUGatePrefix+patientID, gate, c.config.RedisMCUGateTTL)
}

func (c *CacheClient) GetMCUGate(patientID string, result interface{}) error {
	return c.GetJSON(MCUGatePrefix+patientID, result)
}

func (c *CacheClient) InvalidateMCUGate(patientID string) error {
	return c.Delete(MCUGatePrefix + patientID)
}

// Gate history operations.
func (c *CacheClient) SetGateHistory(patientID string, history interface{}) error {
	return c.SetJSON(GateHistoryPrefix+patientID, history, c.config.RedisGateHistoryTTL)
}

func (c *CacheClient) GetGateHistory(patientID string, result interface{}) error {
	return c.GetJSON(GateHistoryPrefix+patientID, result)
}

// Perturbation operations (dynamic TTL).
func (c *CacheClient) SetPerturbations(patientID string, perturbations interface{}, ttl time.Duration) error {
	return c.SetJSON(PerturbationPrefix+patientID, perturbations, ttl)
}

func (c *CacheClient) GetPerturbations(patientID string, result interface{}) error {
	return c.GetJSON(PerturbationPrefix+patientID, result)
}

// Adherence operations.
func (c *CacheClient) SetAdherence(patientID string, adherence interface{}) error {
	return c.SetJSON(AdherencePrefix+patientID, adherence, c.config.RedisAdherenceTTL)
}

func (c *CacheClient) GetAdherence(patientID string, result interface{}) error {
	return c.GetJSON(AdherencePrefix+patientID, result)
}

// Template operations.
func (c *CacheClient) SetTemplate(templateID string, template interface{}) error {
	return c.SetJSON(TemplatePrefix+templateID, template, 0) // no expiry, until reload
}

func (c *CacheClient) GetTemplate(templateID string, result interface{}) error {
	return c.GetJSON(TemplatePrefix+templateID, result)
}

func (c *CacheClient) InvalidateTemplates() error {
	return c.DeletePattern(TemplatePrefix + "*")
}

func (c *CacheClient) Health() error {
	return c.client.Ping(c.ctx).Err()
}

func (c *CacheClient) Close() error {
	return c.client.Close()
}
