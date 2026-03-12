package config

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds all configuration for the Flow 2 Go Engine
type Config struct {
	Server         ServerConfig         `mapstructure:"server"`
	RustEngine     RustEngineConfig     `mapstructure:"rust_engine"`
	JITSafety      JITSafetyConfig      `mapstructure:"jit_safety"`
	ContextService ContextServiceConfig `mapstructure:"context_service"`
	MedicationAPI  MedicationAPIConfig  `mapstructure:"medication_api"`
	Redis          RedisConfig          `mapstructure:"redis"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit"`
	Observability  ObservabilityConfig  `mapstructure:"observability"`
	Phase2         Phase2Config         `mapstructure:"phase2"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Environment  string        `mapstructure:"environment"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// RustEngineConfig holds Rust recipe engine configuration
type RustEngineConfig struct {
	Address            string        `mapstructure:"address"`
	Timeout            time.Duration `mapstructure:"timeout"`
	MaxRetries         int           `mapstructure:"max_retries"`
	CircuitBreakerConfig CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           `mapstructure:"failure_threshold"`
	RecoveryTimeout  time.Duration `mapstructure:"recovery_timeout"`
	MaxRequests      int           `mapstructure:"max_requests"`
}

// JITSafetyConfig holds JIT Safety Engine configuration
type JITSafetyConfig struct {
	BaseURL              string        `mapstructure:"base_url"`
	TimeoutSeconds       int           `mapstructure:"timeout_seconds"`
	RetryAttempts        int           `mapstructure:"retry_attempts"`
	RetryDelay           time.Duration `mapstructure:"retry_delay"`
	EnableCircuitBreaker bool          `mapstructure:"enable_circuit_breaker"`
	Logger               *logrus.Logger `mapstructure:"-"` // Not serialized
}

// ContextServiceConfig holds context service configuration
type ContextServiceConfig struct {
	URL     string        `mapstructure:"url"`
	Timeout time.Duration `mapstructure:"timeout"`
	APIKey  string        `mapstructure:"api_key"`
}

// MedicationAPIConfig holds medication API configuration
type MedicationAPIConfig struct {
	URL     string        `mapstructure:"url"`
	Timeout time.Duration `mapstructure:"timeout"`
	APIKey  string        `mapstructure:"api_key"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int           `mapstructure:"requests_per_second"`
	BurstSize         int           `mapstructure:"burst_size"`
	WindowSize        time.Duration `mapstructure:"window_size"`
}

// ObservabilityConfig holds observability configuration
type ObservabilityConfig struct {
	MetricsEnabled bool   `mapstructure:"metrics_enabled"`
	TracingEnabled bool   `mapstructure:"tracing_enabled"`
	LogLevel       string `mapstructure:"log_level"`
}

// Phase2Config holds Phase 2 Context Assembly configuration
type Phase2Config struct {
	KnowledgeBroker  KnowledgeBrokerConfig  `mapstructure:"knowledge_broker"`
	ContextGateway   Phase2ContextConfig    `mapstructure:"context_gateway"`
	ParallelExecution ParallelExecutionConfig `mapstructure:"parallel_execution"`
	PhenotypeEvaluation PhenotypeConfig     `mapstructure:"phenotype_evaluation"`
	Performance      PerformanceConfig      `mapstructure:"performance"`
}

// KnowledgeBrokerConfig holds Knowledge Broker configuration
type KnowledgeBrokerConfig struct {
	URL             string        `mapstructure:"url"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	Environment     string        `mapstructure:"environment"`
}

// Phase2ContextConfig holds Phase 2 context gateway configuration
type Phase2ContextConfig struct {
	URL        string        `mapstructure:"url"`
	Timeout    time.Duration `mapstructure:"timeout"`
	SnapshotTTL time.Duration `mapstructure:"snapshot_ttl"`
}

// ParallelExecutionConfig holds parallel execution configuration
type ParallelExecutionConfig struct {
	MaxConcurrency   int           `mapstructure:"max_concurrency"`
	DefaultTimeout   time.Duration `mapstructure:"default_timeout"`
	CircuitBreaker   CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// PhenotypeConfig holds phenotype evaluation configuration
type PhenotypeConfig struct {
	RustEngineURL      string        `mapstructure:"rust_engine_url"`
	CacheSize          int           `mapstructure:"cache_size"`
	RuleTTL            time.Duration `mapstructure:"rule_ttl"`
	EvaluationTimeout  time.Duration `mapstructure:"evaluation_timeout"`
}

// PerformanceConfig holds Phase 2 performance configuration
type PerformanceConfig struct {
	TargetLatencyMS         int      `mapstructure:"target_latency_ms"`
	CacheWarmup            bool     `mapstructure:"cache_warmup"`
	PreloadCommonPhenotypes []string `mapstructure:"preload_common_phenotypes"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set default values
	setDefaults()

	// Read environment variables
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.environment", "development")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")
	viper.SetDefault("server.idle_timeout", "60s")

	// Rust engine defaults
	viper.SetDefault("rust_engine.address", "localhost:50051")
	viper.SetDefault("rust_engine.timeout", "30s")
	viper.SetDefault("rust_engine.max_retries", 3)
	viper.SetDefault("rust_engine.circuit_breaker.failure_threshold", 5)
	viper.SetDefault("rust_engine.circuit_breaker.recovery_timeout", "60s")
	viper.SetDefault("rust_engine.circuit_breaker.max_requests", 10)

	// JIT Safety defaults
	viper.SetDefault("jit_safety.base_url", "http://localhost:8080")
	viper.SetDefault("jit_safety.timeout_seconds", 30)
	viper.SetDefault("jit_safety.retry_attempts", 3)
	viper.SetDefault("jit_safety.retry_delay", "100ms")
	viper.SetDefault("jit_safety.enable_circuit_breaker", true)

	// Context service defaults
	viper.SetDefault("context_service.url", "http://localhost:8080")
	viper.SetDefault("context_service.timeout", "10s")

	// Medication API defaults
	viper.SetDefault("medication_api.url", "http://localhost:8009")
	viper.SetDefault("medication_api.timeout", "10s")

	// Redis defaults
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)

	// Rate limit defaults
	viper.SetDefault("rate_limit.requests_per_second", 1000)
	viper.SetDefault("rate_limit.burst_size", 2000)
	viper.SetDefault("rate_limit.window_size", "1m")

	// Observability defaults
	viper.SetDefault("observability.metrics_enabled", true)
	viper.SetDefault("observability.tracing_enabled", true)
	viper.SetDefault("observability.log_level", "info")

	// Phase 2 defaults
	viper.SetDefault("phase2.knowledge_broker.url", "https://kb-broker.internal:8443")
	viper.SetDefault("phase2.knowledge_broker.timeout", "30s")
	viper.SetDefault("phase2.knowledge_broker.refresh_interval", "5m")
	viper.SetDefault("phase2.knowledge_broker.environment", "development")

	viper.SetDefault("phase2.context_gateway.url", "http://localhost:8015")
	viper.SetDefault("phase2.context_gateway.timeout", "30s")
	viper.SetDefault("phase2.context_gateway.snapshot_ttl", "300s")

	viper.SetDefault("phase2.parallel_execution.max_concurrency", 10)
	viper.SetDefault("phase2.parallel_execution.default_timeout", "25ms")
	viper.SetDefault("phase2.parallel_execution.circuit_breaker.failure_threshold", 3)
	viper.SetDefault("phase2.parallel_execution.circuit_breaker.recovery_timeout", "10s")
	viper.SetDefault("phase2.parallel_execution.circuit_breaker.max_requests", 30)

	viper.SetDefault("phase2.phenotype_evaluation.rust_engine_url", "http://localhost:8090")
	viper.SetDefault("phase2.phenotype_evaluation.cache_size", 1000)
	viper.SetDefault("phase2.phenotype_evaluation.rule_ttl", "1h")
	viper.SetDefault("phase2.phenotype_evaluation.evaluation_timeout", "5ms")

	viper.SetDefault("phase2.performance.target_latency_ms", 50)
	viper.SetDefault("phase2.performance.cache_warmup", true)
	viper.SetDefault("phase2.performance.preload_common_phenotypes", []string{
		"htn_stage2_high_risk",
		"diabetes_ckd",
		"heart_failure_preserved_ef",
	})
}
