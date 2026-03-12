package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Service          ServiceConfig          `yaml:"service"`
	Performance      PerformanceConfig      `yaml:"performance"`
	ContextAssembly  ContextAssemblyConfig  `yaml:"context_assembly"`
	Engines          map[string]EngineConfig `yaml:"engines"`
	CircuitBreaker   CircuitBreakerConfig   `yaml:"circuit_breaker"`
	Caching          CachingConfig          `yaml:"caching"`
	ExternalServices ExternalServicesConfig `yaml:"external_services"`
	Database         DatabaseConfig         `yaml:"database"`
	Observability    ObservabilityConfig    `yaml:"observability"`
	Security         SecurityConfig         `yaml:"security"`
	OverrideTokens   OverrideTokensConfig   `yaml:"override_tokens"`
}

// ServiceConfig represents basic service configuration
type ServiceConfig struct {
	Name        string `yaml:"name"`
	Port        int    `yaml:"port"`
	HTTPPort    int    `yaml:"http_port"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
}

// PerformanceConfig represents performance-related configuration
type PerformanceConfig struct {
	MaxConcurrentRequests     int `yaml:"max_concurrent_requests"`
	RequestTimeoutMs          int `yaml:"request_timeout_ms"`
	ContextAssemblyTimeoutMs  int `yaml:"context_assembly_timeout_ms"`
	EngineExecutionTimeoutMs  int `yaml:"engine_execution_timeout_ms"`
	MaxRequestSizeMB          int `yaml:"max_request_size_mb"`
}

// ContextAssemblyConfig represents context assembly configuration
type ContextAssemblyConfig struct {
	Enabled                bool `yaml:"enabled"`
	SkipPatientDataFetch   bool `yaml:"skip_patient_data_fetch"`
	SkipGraphDBQueries     bool `yaml:"skip_graphdb_queries"`
}

// EngineConfig represents configuration for a safety engine
type EngineConfig struct {
	Enabled      bool     `yaml:"enabled"`
	TimeoutMs    int      `yaml:"timeout_ms"`
	Priority     int      `yaml:"priority"`
	Tier         int      `yaml:"tier"`
	Capabilities []string `yaml:"capabilities"`
}

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold     int `yaml:"failure_threshold"`
	ResetTimeoutSeconds  int `yaml:"reset_timeout_seconds"`
	HalfOpenMaxCalls     int `yaml:"half_open_max_calls"`
}

// CachingConfig represents caching configuration
type CachingConfig struct {
	ContextTTLMinutes int         `yaml:"context_ttl_minutes"`
	MaxCacheSizeMB    int         `yaml:"max_cache_size_mb"`
	EvictionPolicy    string      `yaml:"eviction_policy"`
	Redis             RedisConfig `yaml:"redis"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// ExternalServicesConfig represents external service configurations
type ExternalServicesConfig struct {
	FHIRService         ExternalServiceConfig         `yaml:"fhir_service"`
	GraphDBService      ExternalServiceConfig         `yaml:"graphdb_service"`
	CAEService          ExternalServiceConfig         `yaml:"cae_service"`
	GoogleHealthcareAPI GoogleHealthcareAPIConfig     `yaml:"google_healthcare_api"`
}

// ExternalServiceConfig represents configuration for an external service
type ExternalServiceConfig struct {
	Endpoint   string `yaml:"endpoint"`
	TimeoutMs  int    `yaml:"timeout_ms"`
	MaxRetries int    `yaml:"max_retries"`
}

// GoogleHealthcareAPIConfig represents Google Cloud Healthcare API configuration
type GoogleHealthcareAPIConfig struct {
	Enabled         bool   `yaml:"enabled"`
	ProjectID       string `yaml:"project_id"`
	Location        string `yaml:"location"`
	DatasetID       string `yaml:"dataset_id"`
	FHIRStoreID     string `yaml:"fhir_store_id"`
	CredentialsPath string `yaml:"credentials_path"`
	TimeoutMs       int    `yaml:"timeout_ms"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	Name               string `yaml:"name"`
	User               string `yaml:"user"`
	Password           string `yaml:"password"`
	SSLMode            string `yaml:"ssl_mode"`
	MaxConnections     int    `yaml:"max_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
}

// ObservabilityConfig represents observability configuration
type ObservabilityConfig struct {
	LogLevel        string          `yaml:"log_level"`
	MetricsEnabled  bool            `yaml:"metrics_enabled"`
	TracingEnabled  bool            `yaml:"tracing_enabled"`
	AuditLogAsync   bool            `yaml:"audit_log_async"`
	Logging         LoggingConfig   `yaml:"logging"`
	Metrics         MetricsConfig   `yaml:"metrics"`
	Tracing         TracingConfig   `yaml:"tracing"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Format         string `yaml:"format"`
	Output         string `yaml:"output"`
	AuditOutput    string `yaml:"audit_output"`
	AuditFilePath  string `yaml:"audit_file_path"`
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	PrometheusEnabled         bool `yaml:"prometheus_enabled"`
	PrometheusPort            int  `yaml:"prometheus_port"`
	CollectionIntervalSeconds int  `yaml:"collection_interval_seconds"`
}

// TracingConfig represents tracing configuration
type TracingConfig struct {
	JaegerEnabled  bool    `yaml:"jaeger_enabled"`
	JaegerEndpoint string  `yaml:"jaeger_endpoint"`
	SampleRate     float64 `yaml:"sample_rate"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	RateLimiting RateLimitingConfig `yaml:"rate_limiting"`
	Encryption   EncryptionConfig   `yaml:"encryption"`
	Compliance   ComplianceConfig   `yaml:"compliance"`
}

// RateLimitingConfig represents rate limiting configuration
type RateLimitingConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	BurstSize         int  `yaml:"burst_size"`
}

// EncryptionConfig represents encryption configuration
type EncryptionConfig struct {
	Enabled           bool `yaml:"enabled"`
	KeyRotationHours  int  `yaml:"key_rotation_hours"`
}

// ComplianceConfig represents compliance configuration
type ComplianceConfig struct {
	HIPAAMode           bool `yaml:"hipaa_mode"`
	AuditRetentionDays  int  `yaml:"audit_retention_days"`
	PIIEncryption       bool `yaml:"pii_encryption"`
}

// OverrideTokensConfig represents override token configuration
type OverrideTokensConfig struct {
	Enabled        bool                       `yaml:"enabled"`
	ExpiryMinutes  int                        `yaml:"expiry_minutes"`
	SigningKey     string                     `yaml:"signing_key"`
	RequiredLevels map[string]string          `yaml:"required_levels"`
}

// Load loads configuration from a YAML file
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	if err := applyEnvOverrides(&config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// applyEnvOverrides applies environment variable overrides to configuration
func applyEnvOverrides(config *Config) error {
	// Override service port if specified
	if port := os.Getenv("SGP_PORT"); port != "" {
		var portInt int
		if _, err := fmt.Sscanf(port, "%d", &portInt); err == nil {
			config.Service.Port = portInt
		}
	}

	// Override HTTP port if specified
	if httpPort := os.Getenv("SGP_HTTP_PORT"); httpPort != "" {
		var httpPortInt int
		if _, err := fmt.Sscanf(httpPort, "%d", &httpPortInt); err == nil {
			config.Service.HTTPPort = httpPortInt
		}
	}

	// Override database configuration
	if dbHost := os.Getenv("SGP_DB_HOST"); dbHost != "" {
		config.Database.Host = dbHost
	}
	if dbUser := os.Getenv("SGP_DB_USER"); dbUser != "" {
		config.Database.User = dbUser
	}
	if dbPassword := os.Getenv("SGP_DB_PASSWORD"); dbPassword != "" {
		config.Database.Password = dbPassword
	}
	if dbName := os.Getenv("SGP_DB_NAME"); dbName != "" {
		config.Database.Name = dbName
	}

	// Override Redis configuration
	if redisAddr := os.Getenv("SGP_REDIS_ADDRESS"); redisAddr != "" {
		config.Caching.Redis.Address = redisAddr
	}
	if redisPassword := os.Getenv("SGP_REDIS_PASSWORD"); redisPassword != "" {
		config.Caching.Redis.Password = redisPassword
	}

	// Override signing key
	if signingKey := os.Getenv("SGP_SIGNING_KEY"); signingKey != "" {
		config.OverrideTokens.SigningKey = signingKey
	}

	return nil
}

// validate validates the configuration
func validate(config *Config) error {
	if config.Service.Port <= 0 || config.Service.Port > 65535 {
		return fmt.Errorf("invalid service port: %d", config.Service.Port)
	}

	if config.Service.HTTPPort <= 0 || config.Service.HTTPPort > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", config.Service.HTTPPort)
	}

	if config.Service.Port == config.Service.HTTPPort {
		return fmt.Errorf("gRPC port and HTTP port cannot be the same: %d", config.Service.Port)
	}

	if config.Performance.RequestTimeoutMs <= 0 {
		return fmt.Errorf("request timeout must be positive")
	}

	if config.Performance.ContextAssemblyTimeoutMs <= 0 {
		return fmt.Errorf("context assembly timeout must be positive")
	}

	if config.Performance.EngineExecutionTimeoutMs <= 0 {
		return fmt.Errorf("engine execution timeout must be positive")
	}

	// Validate that context assembly + engine execution doesn't exceed total timeout
	totalSubTimeout := config.Performance.ContextAssemblyTimeoutMs + config.Performance.EngineExecutionTimeoutMs
	if totalSubTimeout >= config.Performance.RequestTimeoutMs {
		return fmt.Errorf("sum of context assembly and engine execution timeouts (%dms) must be less than total request timeout (%dms)",
			totalSubTimeout, config.Performance.RequestTimeoutMs)
	}

	if config.OverrideTokens.Enabled && config.OverrideTokens.SigningKey == "" {
		return fmt.Errorf("signing key is required when override tokens are enabled")
	}

	return nil
}

// GetRequestTimeout returns the request timeout as a duration
func (c *Config) GetRequestTimeout() time.Duration {
	return time.Duration(c.Performance.RequestTimeoutMs) * time.Millisecond
}

// GetContextAssemblyTimeout returns the context assembly timeout as a duration
func (c *Config) GetContextAssemblyTimeout() time.Duration {
	return time.Duration(c.Performance.ContextAssemblyTimeoutMs) * time.Millisecond
}

// GetEngineExecutionTimeout returns the engine execution timeout as a duration
func (c *Config) GetEngineExecutionTimeout() time.Duration {
	return time.Duration(c.Performance.EngineExecutionTimeoutMs) * time.Millisecond
}
