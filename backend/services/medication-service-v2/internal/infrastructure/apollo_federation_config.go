package infrastructure

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ApolloFederationConfig holds configuration for Apollo Federation client
type ApolloFederationConfig struct {
	// Connection settings
	URL             string        `yaml:"url" mapstructure:"url"`
	Timeout         time.Duration `yaml:"timeout" mapstructure:"timeout"`
	MaxRetries      int           `yaml:"max_retries" mapstructure:"max_retries"`
	RetryDelay      time.Duration `yaml:"retry_delay" mapstructure:"retry_delay"`
	
	// Circuit breaker settings
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker" mapstructure:"circuit_breaker"`
	
	// Performance settings
	MaxConcurrency     int           `yaml:"max_concurrency" mapstructure:"max_concurrency"`
	BatchSize          int           `yaml:"batch_size" mapstructure:"batch_size"`
	QueryComplexityMax int           `yaml:"query_complexity_max" mapstructure:"query_complexity_max"`
	
	// Cache settings
	CacheEnabled     bool          `yaml:"cache_enabled" mapstructure:"cache_enabled"`
	DefaultCacheTTL  time.Duration `yaml:"default_cache_ttl" mapstructure:"default_cache_ttl"`
	
	// Health check settings
	HealthCheckEnabled  bool          `yaml:"health_check_enabled" mapstructure:"health_check_enabled"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval" mapstructure:"health_check_interval"`
	
	// Monitoring settings
	MetricsEnabled   bool   `yaml:"metrics_enabled" mapstructure:"metrics_enabled"`
	LogQueries       bool   `yaml:"log_queries" mapstructure:"log_queries"`
	LogQueryDetails  bool   `yaml:"log_query_details" mapstructure:"log_query_details"`
	
	// Security settings
	TLSEnabled       bool     `yaml:"tls_enabled" mapstructure:"tls_enabled"`
	TLSInsecure      bool     `yaml:"tls_insecure" mapstructure:"tls_insecure"`
	AuthToken        string   `yaml:"auth_token" mapstructure:"auth_token"`
	AllowedOperations []string `yaml:"allowed_operations" mapstructure:"allowed_operations"`
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	Enabled          bool          `yaml:"enabled" mapstructure:"enabled"`
	MaxRequests      uint32        `yaml:"max_requests" mapstructure:"max_requests"`
	Interval         time.Duration `yaml:"interval" mapstructure:"interval"`
	Timeout          time.Duration `yaml:"timeout" mapstructure:"timeout"`
	FailureThreshold float64       `yaml:"failure_threshold" mapstructure:"failure_threshold"`
}

// Validate validates the Apollo Federation configuration
func (c *ApolloFederationConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("apollo federation URL is required")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	if c.CircuitBreaker.Enabled {
		if c.CircuitBreaker.FailureThreshold <= 0 || c.CircuitBreaker.FailureThreshold > 1 {
			return fmt.Errorf("circuit breaker failure threshold must be between 0 and 1")
		}
		if c.CircuitBreaker.MaxRequests == 0 {
			return fmt.Errorf("circuit breaker max requests must be greater than 0")
		}
	}

	if c.MaxConcurrency <= 0 {
		c.MaxConcurrency = 10 // Set default
	}

	if c.BatchSize <= 0 {
		c.BatchSize = 50 // Set default
	}

	return nil
}

// SetDefaults sets default values for configuration
func (c *ApolloFederationConfig) SetDefaults() {
	if c.URL == "" {
		c.URL = "http://localhost:4000/graphql"
	}
	
	if c.Timeout == 0 {
		c.Timeout = 15 * time.Second
	}
	
	if c.MaxRetries == 0 {
		c.MaxRetries = 2
	}
	
	if c.RetryDelay == 0 {
		c.RetryDelay = 1 * time.Second
	}
	
	if c.MaxConcurrency == 0 {
		c.MaxConcurrency = 10
	}
	
	if c.BatchSize == 0 {
		c.BatchSize = 50
	}
	
	if c.QueryComplexityMax == 0 {
		c.QueryComplexityMax = 1000
	}
	
	if c.DefaultCacheTTL == 0 {
		c.DefaultCacheTTL = 30 * time.Minute
	}
	
	if c.HealthCheckInterval == 0 {
		c.HealthCheckInterval = 30 * time.Second
	}
	
	// Circuit breaker defaults
	if c.CircuitBreaker.MaxRequests == 0 {
		c.CircuitBreaker.MaxRequests = 5
	}
	if c.CircuitBreaker.Interval == 0 {
		c.CircuitBreaker.Interval = 30 * time.Second
	}
	if c.CircuitBreaker.Timeout == 0 {
		c.CircuitBreaker.Timeout = 5 * time.Second
	}
	if c.CircuitBreaker.FailureThreshold == 0 {
		c.CircuitBreaker.FailureThreshold = 0.5
	}
	
	// Enable defaults
	c.HealthCheckEnabled = true
	c.MetricsEnabled = true
	c.CacheEnabled = true
}

// GetLogLevel returns appropriate log level based on configuration
func (c *ApolloFederationConfig) GetLogLevel() zap.AtomicLevel {
	if c.LogQueryDetails {
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	if c.LogQueries {
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	return zap.NewAtomicLevelAt(zap.WarnLevel)
}

// GetClientOptions returns HTTP client configuration options
func (c *ApolloFederationConfig) GetClientOptions() map[string]interface{} {
	options := make(map[string]interface{})
	
	options["timeout"] = c.Timeout
	options["max_retries"] = c.MaxRetries
	options["retry_delay"] = c.RetryDelay
	
	if c.TLSEnabled {
		options["tls_enabled"] = true
		options["tls_insecure"] = c.TLSInsecure
	}
	
	if c.AuthToken != "" {
		options["auth_token"] = c.AuthToken
	}
	
	return options
}

// IsOperationAllowed checks if a GraphQL operation is allowed
func (c *ApolloFederationConfig) IsOperationAllowed(operation string) bool {
	if len(c.AllowedOperations) == 0 {
		return true // Allow all if no restrictions
	}
	
	for _, allowed := range c.AllowedOperations {
		if allowed == operation || allowed == "*" {
			return true
		}
	}
	
	return false
}

// GetPriorityTimeouts returns timeout settings based on priority
func (c *ApolloFederationConfig) GetPriorityTimeouts() map[string]time.Duration {
	return map[string]time.Duration{
		"critical": c.Timeout * 2,     // Extended timeout for critical queries
		"high":     c.Timeout,         // Standard timeout  
		"normal":   c.Timeout,         // Standard timeout
		"low":      c.Timeout / 2,     // Reduced timeout for low priority
	}
}

// GetCircuitBreakerConfig returns circuit breaker configuration
func (c *ApolloFederationConfig) GetCircuitBreakerConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled":           c.CircuitBreaker.Enabled,
		"max_requests":      c.CircuitBreaker.MaxRequests,
		"interval":          c.CircuitBreaker.Interval,
		"timeout":           c.CircuitBreaker.Timeout,
		"failure_threshold": c.CircuitBreaker.FailureThreshold,
	}
}

// NewApolloFederationConfigFromMap creates configuration from a map
func NewApolloFederationConfigFromMap(configMap map[string]interface{}) (*ApolloFederationConfig, error) {
	config := &ApolloFederationConfig{}
	
	// Extract basic settings
	if url, ok := configMap["url"].(string); ok {
		config.URL = url
	}
	
	if timeoutStr, ok := configMap["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.Timeout = timeout
		}
	}
	
	if maxRetries, ok := configMap["max_retries"].(int); ok {
		config.MaxRetries = maxRetries
	}
	
	// Extract circuit breaker settings
	if cbConfig, ok := configMap["circuit_breaker"].(map[string]interface{}); ok {
		if enabled, ok := cbConfig["enabled"].(bool); ok {
			config.CircuitBreaker.Enabled = enabled
		}
		
		if maxRequests, ok := cbConfig["max_requests"].(int); ok {
			config.CircuitBreaker.MaxRequests = uint32(maxRequests)
		}
		
		if intervalStr, ok := cbConfig["interval"].(string); ok {
			if interval, err := time.ParseDuration(intervalStr); err == nil {
				config.CircuitBreaker.Interval = interval
			}
		}
		
		if timeoutStr, ok := cbConfig["timeout"].(string); ok {
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				config.CircuitBreaker.Timeout = timeout
			}
		}
		
		if failureThreshold, ok := cbConfig["failure_threshold"].(float64); ok {
			config.CircuitBreaker.FailureThreshold = failureThreshold
		}
	}
	
	// Set defaults and validate
	config.SetDefaults()
	
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid apollo federation config: %w", err)
	}
	
	return config, nil
}

// ToMap converts configuration to a map for serialization
func (c *ApolloFederationConfig) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"url":                   c.URL,
		"timeout":              c.Timeout.String(),
		"max_retries":          c.MaxRetries,
		"retry_delay":          c.RetryDelay.String(),
		"max_concurrency":      c.MaxConcurrency,
		"batch_size":           c.BatchSize,
		"query_complexity_max": c.QueryComplexityMax,
		"cache_enabled":        c.CacheEnabled,
		"default_cache_ttl":    c.DefaultCacheTTL.String(),
		"health_check_enabled": c.HealthCheckEnabled,
		"health_check_interval": c.HealthCheckInterval.String(),
		"metrics_enabled":      c.MetricsEnabled,
		"log_queries":         c.LogQueries,
		"log_query_details":   c.LogQueryDetails,
		"tls_enabled":         c.TLSEnabled,
		"tls_insecure":        c.TLSInsecure,
		"circuit_breaker": map[string]interface{}{
			"enabled":           c.CircuitBreaker.Enabled,
			"max_requests":      c.CircuitBreaker.MaxRequests,
			"interval":          c.CircuitBreaker.Interval.String(),
			"timeout":           c.CircuitBreaker.Timeout.String(),
			"failure_threshold": c.CircuitBreaker.FailureThreshold,
		},
	}
}

// Clone creates a deep copy of the configuration
func (c *ApolloFederationConfig) Clone() *ApolloFederationConfig {
	clone := &ApolloFederationConfig{
		URL:                   c.URL,
		Timeout:              c.Timeout,
		MaxRetries:           c.MaxRetries,
		RetryDelay:           c.RetryDelay,
		MaxConcurrency:       c.MaxConcurrency,
		BatchSize:            c.BatchSize,
		QueryComplexityMax:   c.QueryComplexityMax,
		CacheEnabled:         c.CacheEnabled,
		DefaultCacheTTL:      c.DefaultCacheTTL,
		HealthCheckEnabled:   c.HealthCheckEnabled,
		HealthCheckInterval:  c.HealthCheckInterval,
		MetricsEnabled:       c.MetricsEnabled,
		LogQueries:          c.LogQueries,
		LogQueryDetails:     c.LogQueryDetails,
		TLSEnabled:          c.TLSEnabled,
		TLSInsecure:         c.TLSInsecure,
		AuthToken:           c.AuthToken,
		CircuitBreaker:      c.CircuitBreaker,
	}
	
	// Deep copy allowed operations
	clone.AllowedOperations = make([]string, len(c.AllowedOperations))
	copy(clone.AllowedOperations, c.AllowedOperations)
	
	return clone
}