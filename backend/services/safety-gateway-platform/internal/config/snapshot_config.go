package config

import (
	"time"
)

// SnapshotConfig contains configuration for snapshot-based processing
type SnapshotConfig struct {
	// Core snapshot processing settings
	Enabled                 bool          `yaml:"enabled"`
	RequestTimeout          time.Duration `yaml:"request_timeout"`
	EngineExecutionTimeout  time.Duration `yaml:"engine_execution_timeout"`
	MinDataCompleteness     float64       `yaml:"min_data_completeness"`
	
	// Cache settings
	CacheMinTTL             time.Duration `yaml:"cache_min_ttl"`
	CacheMaxTTL             time.Duration `yaml:"cache_max_ttl"`
	EnablePreWarming        bool          `yaml:"enable_pre_warming"`
	
	// Fallback and error handling
	AllowFallbackToLegacy   bool          `yaml:"allow_fallback_to_legacy"`
	MaxRetries              int           `yaml:"max_retries"`
	RetryBackoff            time.Duration `yaml:"retry_backoff"`
	
	// Live fetch settings (for incomplete snapshots)
	EnableLiveFetch         bool          `yaml:"enable_live_fetch"`
	LiveFetchTimeout        time.Duration `yaml:"live_fetch_timeout"`
	MaxLiveFetchFields      int           `yaml:"max_live_fetch_fields"`
	
	// Validation settings
	StrictValidation        bool          `yaml:"strict_validation"`
	RequireSignature        bool          `yaml:"require_signature"`
	SigningKeyPath          string        `yaml:"signing_key_path"`
	
	// Performance settings
	EnableConcurrentRetrieval bool        `yaml:"enable_concurrent_retrieval"`
	MaxConcurrentRetrievals   int         `yaml:"max_concurrent_retrievals"`
}

// ContextGatewayConfig contains configuration for Context Gateway client
type ContextGatewayConfig struct {
	Endpoint     string        `yaml:"endpoint"`
	Timeout      time.Duration `yaml:"timeout"`
	MaxRetries   int           `yaml:"max_retries"`
	ServiceName  string        `yaml:"service_name"`
	EnableTLS    bool          `yaml:"enable_tls"`
	HealthCheck  bool          `yaml:"health_check"`
	
	// Connection pooling
	MaxConnections      int           `yaml:"max_connections"`
	ConnectionTimeout   time.Duration `yaml:"connection_timeout"`
	KeepAliveTimeout    time.Duration `yaml:"keep_alive_timeout"`
	
	// Retry configuration
	RetryBackoff        time.Duration `yaml:"retry_backoff"`
	MaxRetryBackoff     time.Duration `yaml:"max_retry_backoff"`
	RetryMultiplier     float64       `yaml:"retry_multiplier"`
}

// CacheConfig contains configuration for snapshot caching
type CacheConfig struct {
	// L1 Cache (In-Memory)
	L1MaxSize           int           `yaml:"l1_max_size"`
	L1TTL               time.Duration `yaml:"l1_ttl"`
	L1CleanupInterval   time.Duration `yaml:"l1_cleanup_interval"`
	
	// L2 Cache (Redis)
	EnableL2Cache       bool          `yaml:"enable_l2_cache"`
	L2TTL               time.Duration `yaml:"l2_ttl"`
	Redis               RedisConfig   `yaml:"redis"`
	
	// Cache behavior
	WriteThrough        bool          `yaml:"write_through"`
	WriteBack           bool          `yaml:"write_back"`
	EnableCompression   bool          `yaml:"enable_compression"`
	CompressionLevel    int           `yaml:"compression_level"`
	
	// Cache warming
	EnableWarming       bool          `yaml:"enable_warming"`
	WarmingConcurrency  int           `yaml:"warming_concurrency"`
}

// RedisConfig contains Redis-specific configuration
type RedisConfig struct {
	Address          string        `yaml:"address"`
	Password         string        `yaml:"password"`
	DB               int           `yaml:"db"`
	PoolSize         int           `yaml:"pool_size"`
	MinIdleConns     int           `yaml:"min_idle_conns"`
	MaxConnAge       time.Duration `yaml:"max_conn_age"`
	PoolTimeout      time.Duration `yaml:"pool_timeout"`
	IdleTimeout      time.Duration `yaml:"idle_timeout"`
	IdleCheckFrequency time.Duration `yaml:"idle_check_frequency"`
	
	// Cluster settings
	EnableCluster    bool     `yaml:"enable_cluster"`
	ClusterAddresses []string `yaml:"cluster_addresses"`
}

// GetDefaultSnapshotConfig returns default snapshot configuration
func GetDefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		Enabled:                 false, // Disabled by default for safety
		RequestTimeout:          5 * time.Second,
		EngineExecutionTimeout:  3 * time.Second,
		MinDataCompleteness:     60.0, // 60% minimum data completeness
		
		CacheMinTTL:             1 * time.Minute,
		CacheMaxTTL:             30 * time.Minute,
		EnablePreWarming:        false,
		
		AllowFallbackToLegacy:   true, // Allow fallback during transition
		MaxRetries:              3,
		RetryBackoff:            100 * time.Millisecond,
		
		EnableLiveFetch:         false, // Disabled by default
		LiveFetchTimeout:        2 * time.Second,
		MaxLiveFetchFields:      5,
		
		StrictValidation:        true,
		RequireSignature:        false, // Optional by default
		SigningKeyPath:          "",
		
		EnableConcurrentRetrieval: true,
		MaxConcurrentRetrievals:   10,
	}
}

// GetDefaultContextGatewayConfig returns default Context Gateway configuration
func GetDefaultContextGatewayConfig() *ContextGatewayConfig {
	return &ContextGatewayConfig{
		Endpoint:     "localhost:8050",
		Timeout:      3 * time.Second,
		MaxRetries:   3,
		ServiceName:  "safety-gateway-platform",
		EnableTLS:    false,
		HealthCheck:  true,
		
		MaxConnections:    10,
		ConnectionTimeout: 5 * time.Second,
		KeepAliveTimeout:  30 * time.Second,
		
		RetryBackoff:     100 * time.Millisecond,
		MaxRetryBackoff:  2 * time.Second,
		RetryMultiplier:  2.0,
	}
}

// GetDefaultCacheConfig returns default cache configuration
func GetDefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		L1MaxSize:         1000,
		L1TTL:             5 * time.Minute,
		L1CleanupInterval: 1 * time.Minute,
		
		EnableL2Cache:     true,
		L2TTL:             30 * time.Minute,
		Redis:             *GetDefaultRedisConfig(),
		
		WriteThrough:      true,
		WriteBack:         false,
		EnableCompression: false,
		CompressionLevel:  6, // gzip default
		
		EnableWarming:     false,
		WarmingConcurrency: 5,
	}
}

// GetDefaultRedisConfig returns default Redis configuration
func GetDefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Address:            "localhost:6379",
		Password:           "",
		DB:                 0,
		PoolSize:           10,
		MinIdleConns:       2,
		MaxConnAge:         30 * time.Minute,
		PoolTimeout:        4 * time.Second,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute,
		
		EnableCluster:      false,
		ClusterAddresses:   []string{},
	}
}

// Validate validates the snapshot configuration
func (c *SnapshotConfig) Validate() error {
	if c.MinDataCompleteness < 0 || c.MinDataCompleteness > 100 {
		return fmt.Errorf("min_data_completeness must be between 0 and 100, got %f", c.MinDataCompleteness)
	}
	
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive, got %v", c.RequestTimeout)
	}
	
	if c.EngineExecutionTimeout <= 0 {
		return fmt.Errorf("engine_execution_timeout must be positive, got %v", c.EngineExecutionTimeout)
	}
	
	if c.CacheMinTTL <= 0 || c.CacheMaxTTL <= 0 {
		return fmt.Errorf("cache TTL values must be positive")
	}
	
	if c.CacheMinTTL > c.CacheMaxTTL {
		return fmt.Errorf("cache_min_ttl (%v) cannot be greater than cache_max_ttl (%v)", 
			c.CacheMinTTL, c.CacheMaxTTL)
	}
	
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative, got %d", c.MaxRetries)
	}
	
	if c.MaxConcurrentRetrievals <= 0 {
		return fmt.Errorf("max_concurrent_retrievals must be positive, got %d", c.MaxConcurrentRetrievals)
	}
	
	return nil
}

// Validate validates the Context Gateway configuration
func (c *ContextGatewayConfig) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint cannot be empty")
	}
	
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}
	
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative, got %d", c.MaxRetries)
	}
	
	if c.ServiceName == "" {
		return fmt.Errorf("service_name cannot be empty")
	}
	
	if c.MaxConnections <= 0 {
		return fmt.Errorf("max_connections must be positive, got %d", c.MaxConnections)
	}
	
	return nil
}

// Validate validates the cache configuration
func (c *CacheConfig) Validate() error {
	if c.L1MaxSize <= 0 {
		return fmt.Errorf("l1_max_size must be positive, got %d", c.L1MaxSize)
	}
	
	if c.L1TTL <= 0 {
		return fmt.Errorf("l1_ttl must be positive, got %v", c.L1TTL)
	}
	
	if c.EnableL2Cache && c.L2TTL <= 0 {
		return fmt.Errorf("l2_ttl must be positive when L2 cache is enabled, got %v", c.L2TTL)
	}
	
	if c.EnableCompression && (c.CompressionLevel < 1 || c.CompressionLevel > 9) {
		return fmt.Errorf("compression_level must be between 1 and 9, got %d", c.CompressionLevel)
	}
	
	if c.EnableL2Cache {
		if err := c.Redis.Validate(); err != nil {
			return fmt.Errorf("redis config validation failed: %w", err)
		}
	}
	
	return nil
}

// Validate validates the Redis configuration
func (c *RedisConfig) Validate() error {
	if c.Address == "" {
		return fmt.Errorf("address cannot be empty")
	}
	
	if c.PoolSize <= 0 {
		return fmt.Errorf("pool_size must be positive, got %d", c.PoolSize)
	}
	
	if c.MinIdleConns < 0 {
		return fmt.Errorf("min_idle_conns cannot be negative, got %d", c.MinIdleConns)
	}
	
	if c.EnableCluster && len(c.ClusterAddresses) == 0 {
		return fmt.Errorf("cluster_addresses cannot be empty when cluster is enabled")
	}
	
	return nil
}

// IsSnapshotModeEnabled returns whether snapshot mode is enabled and properly configured
func (c *SnapshotConfig) IsSnapshotModeEnabled() bool {
	return c.Enabled
}

// GetCacheTTLRange returns the min and max cache TTL values
func (c *SnapshotConfig) GetCacheTTLRange() (time.Duration, time.Duration) {
	return c.CacheMinTTL, c.CacheMaxTTL
}

// ShouldRetry determines if an operation should be retried based on configuration
func (c *SnapshotConfig) ShouldRetry(attempt int) bool {
	return attempt < c.MaxRetries
}

// GetRetryDelay calculates retry delay with exponential backoff
func (c *SnapshotConfig) GetRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}
	
	delay := c.RetryBackoff
	for i := 1; i < attempt; i++ {
		delay *= 2
	}
	
	// Cap at maximum of 5 seconds
	maxDelay := 5 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	
	return delay
}