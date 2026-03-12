package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// RustEngineConfiguration holds all Rust Clinical Engine configuration
type RustEngineConfiguration struct {
	// Connection settings
	BaseURL             string        `json:"base_url" mapstructure:"base_url"`
	HealthCheckURL      string        `json:"health_check_url" mapstructure:"health_check_url"`
	MetricsURL          string        `json:"metrics_url" mapstructure:"metrics_url"`
	Timeout             time.Duration `json:"timeout" mapstructure:"timeout"`
	
	// Retry settings
	MaxRetries          int           `json:"max_retries" mapstructure:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay" mapstructure:"retry_delay"`
	ExponentialBackoff  bool          `json:"exponential_backoff" mapstructure:"exponential_backoff"`
	
	// Performance settings
	MaxConcurrentRequests int          `json:"max_concurrent_requests" mapstructure:"max_concurrent_requests"`
	RequestTimeout        time.Duration `json:"request_timeout" mapstructure:"request_timeout"`
	KeepAlive            time.Duration `json:"keep_alive" mapstructure:"keep_alive"`
	
	// Feature flags
	EnableCaching        bool          `json:"enable_caching" mapstructure:"enable_caching"`
	EnableMetrics        bool          `json:"enable_metrics" mapstructure:"enable_metrics"`
	EnableHealthChecks   bool          `json:"enable_health_checks" mapstructure:"enable_health_checks"`
	EnableTracing        bool          `json:"enable_tracing" mapstructure:"enable_tracing"`
	
	// Performance targets
	PerformanceTargets   *RustEnginePerformanceTargets `json:"performance_targets" mapstructure:"performance_targets"`
	
	// Service endpoints
	Endpoints           *RustEngineEndpoints `json:"endpoints" mapstructure:"endpoints"`
	
	// Circuit breaker settings
	CircuitBreaker      *CircuitBreakerConfig `json:"circuit_breaker" mapstructure:"circuit_breaker"`
}

// RustEnginePerformanceTargets defines performance targets for Rust engine operations
type RustEnginePerformanceTargets struct {
	DrugInteractionAnalysis time.Duration `json:"drug_interaction_analysis" mapstructure:"drug_interaction_analysis"`
	DosageCalculation      time.Duration `json:"dosage_calculation" mapstructure:"dosage_calculation"`
	SafetyValidation       time.Duration `json:"safety_validation" mapstructure:"safety_validation"`
	RuleEvaluation         time.Duration `json:"rule_evaluation" mapstructure:"rule_evaluation"`
	HealthCheck            time.Duration `json:"health_check" mapstructure:"health_check"`
	MetricsRetrieval       time.Duration `json:"metrics_retrieval" mapstructure:"metrics_retrieval"`
}

// RustEngineEndpoints defines all Rust engine API endpoints
type RustEngineEndpoints struct {
	// Core API endpoints
	DrugInteractions    string `json:"drug_interactions" mapstructure:"drug_interactions"`
	DosageCalculation   string `json:"dosage_calculation" mapstructure:"dosage_calculation"`
	SafetyValidation    string `json:"safety_validation" mapstructure:"safety_validation"`
	RuleEvaluation      string `json:"rule_evaluation" mapstructure:"rule_evaluation"`
	
	// Batch processing endpoints
	BatchDrugInteractions string `json:"batch_drug_interactions" mapstructure:"batch_drug_interactions"`
	BatchDosageCalculation string `json:"batch_dosage_calculation" mapstructure:"batch_dosage_calculation"`
	BatchSafetyValidation string `json:"batch_safety_validation" mapstructure:"batch_safety_validation"`
	
	// Management endpoints
	Health              string `json:"health" mapstructure:"health"`
	Metrics             string `json:"metrics" mapstructure:"metrics"`
	Status              string `json:"status" mapstructure:"status"`
	Version             string `json:"version" mapstructure:"version"`
	
	// Advanced endpoints
	ClinicalIntelligence string `json:"clinical_intelligence" mapstructure:"clinical_intelligence"`
	RiskAssessment      string `json:"risk_assessment" mapstructure:"risk_assessment"`
	RecommendationEngine string `json:"recommendation_engine" mapstructure:"recommendation_engine"`
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	Enabled             bool          `json:"enabled" mapstructure:"enabled"`
	FailureThreshold    int           `json:"failure_threshold" mapstructure:"failure_threshold"`
	SuccessThreshold    int           `json:"success_threshold" mapstructure:"success_threshold"`
	Timeout             time.Duration `json:"timeout" mapstructure:"timeout"`
	MaxRequests         int           `json:"max_requests" mapstructure:"max_requests"`
	Interval            time.Duration `json:"interval" mapstructure:"interval"`
}

// LoadRustEngineConfiguration loads Rust engine configuration from various sources
func LoadRustEngineConfiguration() (*RustEngineConfiguration, error) {
	config := &RustEngineConfiguration{}
	
	// Set configuration path and name
	viper.SetConfigName("rust_engine_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	
	// Set environment variable prefix
	viper.SetEnvPrefix("RUST_ENGINE")
	viper.AutomaticEnv()
	
	// Set defaults
	setRustEngineDefaults()
	
	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading rust engine config file: %w", err)
		}
		// Config file not found, use defaults and environment variables
	}
	
	// Unmarshal configuration
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling rust engine config: %w", err)
	}
	
	// Validate configuration
	if err := validateRustEngineConfig(config); err != nil {
		return nil, fmt.Errorf("invalid rust engine configuration: %w", err)
	}
	
	return config, nil
}

// setRustEngineDefaults sets default values for Rust engine configuration
func setRustEngineDefaults() {
	// Connection settings
	viper.SetDefault("base_url", "http://localhost:8090")
	viper.SetDefault("health_check_url", "/health")
	viper.SetDefault("metrics_url", "/metrics")
	viper.SetDefault("timeout", "5s")
	
	// Retry settings
	viper.SetDefault("max_retries", 3)
	viper.SetDefault("retry_delay", "100ms")
	viper.SetDefault("exponential_backoff", true)
	
	// Performance settings
	viper.SetDefault("max_concurrent_requests", 100)
	viper.SetDefault("request_timeout", "2s")
	viper.SetDefault("keep_alive", "30s")
	
	// Feature flags
	viper.SetDefault("enable_caching", true)
	viper.SetDefault("enable_metrics", true)
	viper.SetDefault("enable_health_checks", true)
	viper.SetDefault("enable_tracing", true)
	
	// Performance targets
	viper.SetDefault("performance_targets.drug_interaction_analysis", "50ms")
	viper.SetDefault("performance_targets.dosage_calculation", "30ms")
	viper.SetDefault("performance_targets.safety_validation", "75ms")
	viper.SetDefault("performance_targets.rule_evaluation", "40ms")
	viper.SetDefault("performance_targets.health_check", "500ms")
	viper.SetDefault("performance_targets.metrics_retrieval", "1s")
	
	// Service endpoints
	viper.SetDefault("endpoints.drug_interactions", "/api/v1/drug-interactions")
	viper.SetDefault("endpoints.dosage_calculation", "/api/v1/dosage-calculation")
	viper.SetDefault("endpoints.safety_validation", "/api/v1/safety-validation")
	viper.SetDefault("endpoints.rule_evaluation", "/api/v1/evaluate-rules")
	viper.SetDefault("endpoints.batch_drug_interactions", "/api/v1/batch/drug-interactions")
	viper.SetDefault("endpoints.batch_dosage_calculation", "/api/v1/batch/dosage-calculation")
	viper.SetDefault("endpoints.batch_safety_validation", "/api/v1/batch/safety-validation")
	viper.SetDefault("endpoints.health", "/health")
	viper.SetDefault("endpoints.metrics", "/metrics")
	viper.SetDefault("endpoints.status", "/status")
	viper.SetDefault("endpoints.version", "/version")
	viper.SetDefault("endpoints.clinical_intelligence", "/api/v1/clinical-intelligence")
	viper.SetDefault("endpoints.risk_assessment", "/api/v1/risk-assessment")
	viper.SetDefault("endpoints.recommendation_engine", "/api/v1/recommendations")
	
	// Circuit breaker settings
	viper.SetDefault("circuit_breaker.enabled", true)
	viper.SetDefault("circuit_breaker.failure_threshold", 5)
	viper.SetDefault("circuit_breaker.success_threshold", 3)
	viper.SetDefault("circuit_breaker.timeout", "30s")
	viper.SetDefault("circuit_breaker.max_requests", 10)
	viper.SetDefault("circuit_breaker.interval", "10s")
}

// validateRustEngineConfig validates the Rust engine configuration
func validateRustEngineConfig(config *RustEngineConfiguration) error {
	if config.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	
	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	
	if config.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	
	if config.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("max_concurrent_requests must be positive")
	}
	
	if config.PerformanceTargets == nil {
		return fmt.Errorf("performance_targets configuration is required")
	}
	
	if config.Endpoints == nil {
		return fmt.Errorf("endpoints configuration is required")
	}
	
	// Validate performance targets
	if err := validatePerformanceTargets(config.PerformanceTargets); err != nil {
		return fmt.Errorf("invalid performance targets: %w", err)
	}
	
	// Validate circuit breaker config if enabled
	if config.CircuitBreaker != nil && config.CircuitBreaker.Enabled {
		if err := validateCircuitBreakerConfig(config.CircuitBreaker); err != nil {
			return fmt.Errorf("invalid circuit breaker config: %w", err)
		}
	}
	
	return nil
}

// validatePerformanceTargets validates performance targets
func validatePerformanceTargets(targets *RustEnginePerformanceTargets) error {
	if targets.DrugInteractionAnalysis <= 0 {
		return fmt.Errorf("drug_interaction_analysis target must be positive")
	}
	if targets.DosageCalculation <= 0 {
		return fmt.Errorf("dosage_calculation target must be positive")
	}
	if targets.SafetyValidation <= 0 {
		return fmt.Errorf("safety_validation target must be positive")
	}
	if targets.RuleEvaluation <= 0 {
		return fmt.Errorf("rule_evaluation target must be positive")
	}
	return nil
}

// validateCircuitBreakerConfig validates circuit breaker configuration
func validateCircuitBreakerConfig(cb *CircuitBreakerConfig) error {
	if cb.FailureThreshold <= 0 {
		return fmt.Errorf("failure_threshold must be positive")
	}
	if cb.SuccessThreshold <= 0 {
		return fmt.Errorf("success_threshold must be positive")
	}
	if cb.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if cb.MaxRequests <= 0 {
		return fmt.Errorf("max_requests must be positive")
	}
	return nil
}

// GetRustEngineEnvironmentConfig creates configuration from environment variables
func GetRustEngineEnvironmentConfig() *RustEngineConfiguration {
	config := &RustEngineConfiguration{
		BaseURL:               getEnvString("RUST_ENGINE_BASE_URL", "http://localhost:8090"),
		HealthCheckURL:        getEnvString("RUST_ENGINE_HEALTH_CHECK_URL", "/health"),
		MetricsURL:           getEnvString("RUST_ENGINE_METRICS_URL", "/metrics"),
		Timeout:              getEnvDuration("RUST_ENGINE_TIMEOUT", 5*time.Second),
		MaxRetries:           getEnvInt("RUST_ENGINE_MAX_RETRIES", 3),
		RetryDelay:           getEnvDuration("RUST_ENGINE_RETRY_DELAY", 100*time.Millisecond),
		ExponentialBackoff:   getEnvBool("RUST_ENGINE_EXPONENTIAL_BACKOFF", true),
		MaxConcurrentRequests: getEnvInt("RUST_ENGINE_MAX_CONCURRENT_REQUESTS", 100),
		RequestTimeout:       getEnvDuration("RUST_ENGINE_REQUEST_TIMEOUT", 2*time.Second),
		KeepAlive:           getEnvDuration("RUST_ENGINE_KEEP_ALIVE", 30*time.Second),
		EnableCaching:       getEnvBool("RUST_ENGINE_ENABLE_CACHING", true),
		EnableMetrics:       getEnvBool("RUST_ENGINE_ENABLE_METRICS", true),
		EnableHealthChecks:  getEnvBool("RUST_ENGINE_ENABLE_HEALTH_CHECKS", true),
		EnableTracing:       getEnvBool("RUST_ENGINE_ENABLE_TRACING", true),
	}

	// Set performance targets
	config.PerformanceTargets = &RustEnginePerformanceTargets{
		DrugInteractionAnalysis: getEnvDuration("RUST_ENGINE_PERFORMANCE_DRUG_INTERACTION_ANALYSIS", 50*time.Millisecond),
		DosageCalculation:      getEnvDuration("RUST_ENGINE_PERFORMANCE_DOSAGE_CALCULATION", 30*time.Millisecond),
		SafetyValidation:       getEnvDuration("RUST_ENGINE_PERFORMANCE_SAFETY_VALIDATION", 75*time.Millisecond),
		RuleEvaluation:         getEnvDuration("RUST_ENGINE_PERFORMANCE_RULE_EVALUATION", 40*time.Millisecond),
		HealthCheck:            getEnvDuration("RUST_ENGINE_PERFORMANCE_HEALTH_CHECK", 500*time.Millisecond),
		MetricsRetrieval:       getEnvDuration("RUST_ENGINE_PERFORMANCE_METRICS_RETRIEVAL", 1*time.Second),
	}

	// Set endpoints
	config.Endpoints = &RustEngineEndpoints{
		DrugInteractions:       getEnvString("RUST_ENGINE_ENDPOINT_DRUG_INTERACTIONS", "/api/v1/drug-interactions"),
		DosageCalculation:     getEnvString("RUST_ENGINE_ENDPOINT_DOSAGE_CALCULATION", "/api/v1/dosage-calculation"),
		SafetyValidation:      getEnvString("RUST_ENGINE_ENDPOINT_SAFETY_VALIDATION", "/api/v1/safety-validation"),
		RuleEvaluation:        getEnvString("RUST_ENGINE_ENDPOINT_RULE_EVALUATION", "/api/v1/evaluate-rules"),
		BatchDrugInteractions: getEnvString("RUST_ENGINE_ENDPOINT_BATCH_DRUG_INTERACTIONS", "/api/v1/batch/drug-interactions"),
		BatchDosageCalculation: getEnvString("RUST_ENGINE_ENDPOINT_BATCH_DOSAGE_CALCULATION", "/api/v1/batch/dosage-calculation"),
		BatchSafetyValidation: getEnvString("RUST_ENGINE_ENDPOINT_BATCH_SAFETY_VALIDATION", "/api/v1/batch/safety-validation"),
		Health:                getEnvString("RUST_ENGINE_ENDPOINT_HEALTH", "/health"),
		Metrics:               getEnvString("RUST_ENGINE_ENDPOINT_METRICS", "/metrics"),
		Status:                getEnvString("RUST_ENGINE_ENDPOINT_STATUS", "/status"),
		Version:               getEnvString("RUST_ENGINE_ENDPOINT_VERSION", "/version"),
		ClinicalIntelligence:  getEnvString("RUST_ENGINE_ENDPOINT_CLINICAL_INTELLIGENCE", "/api/v1/clinical-intelligence"),
		RiskAssessment:        getEnvString("RUST_ENGINE_ENDPOINT_RISK_ASSESSMENT", "/api/v1/risk-assessment"),
		RecommendationEngine:  getEnvString("RUST_ENGINE_ENDPOINT_RECOMMENDATION_ENGINE", "/api/v1/recommendations"),
	}

	// Set circuit breaker config
	config.CircuitBreaker = &CircuitBreakerConfig{
		Enabled:          getEnvBool("RUST_ENGINE_CIRCUIT_BREAKER_ENABLED", true),
		FailureThreshold: getEnvInt("RUST_ENGINE_CIRCUIT_BREAKER_FAILURE_THRESHOLD", 5),
		SuccessThreshold: getEnvInt("RUST_ENGINE_CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3),
		Timeout:          getEnvDuration("RUST_ENGINE_CIRCUIT_BREAKER_TIMEOUT", 30*time.Second),
		MaxRequests:      getEnvInt("RUST_ENGINE_CIRCUIT_BREAKER_MAX_REQUESTS", 10),
		Interval:         getEnvDuration("RUST_ENGINE_CIRCUIT_BREAKER_INTERVAL", 10*time.Second),
	}

	return config
}

// CreateRustEngineConfigTemplate creates a template configuration file
func CreateRustEngineConfigTemplate() map[string]interface{} {
	return map[string]interface{}{
		"base_url":                "http://localhost:8090",
		"health_check_url":        "/health",
		"metrics_url":            "/metrics",
		"timeout":                "5s",
		"max_retries":            3,
		"retry_delay":            "100ms",
		"exponential_backoff":    true,
		"max_concurrent_requests": 100,
		"request_timeout":        "2s",
		"keep_alive":            "30s",
		"enable_caching":        true,
		"enable_metrics":        true,
		"enable_health_checks":  true,
		"enable_tracing":        true,
		
		"performance_targets": map[string]interface{}{
			"drug_interaction_analysis": "50ms",
			"dosage_calculation":        "30ms",
			"safety_validation":         "75ms",
			"rule_evaluation":           "40ms",
			"health_check":              "500ms",
			"metrics_retrieval":         "1s",
		},
		
		"endpoints": map[string]interface{}{
			"drug_interactions":         "/api/v1/drug-interactions",
			"dosage_calculation":        "/api/v1/dosage-calculation",
			"safety_validation":         "/api/v1/safety-validation",
			"rule_evaluation":           "/api/v1/evaluate-rules",
			"batch_drug_interactions":   "/api/v1/batch/drug-interactions",
			"batch_dosage_calculation":  "/api/v1/batch/dosage-calculation",
			"batch_safety_validation":   "/api/v1/batch/safety-validation",
			"health":                    "/health",
			"metrics":                   "/metrics",
			"status":                    "/status",
			"version":                   "/version",
			"clinical_intelligence":     "/api/v1/clinical-intelligence",
			"risk_assessment":           "/api/v1/risk-assessment",
			"recommendation_engine":     "/api/v1/recommendations",
		},
		
		"circuit_breaker": map[string]interface{}{
			"enabled":           true,
			"failure_threshold": 5,
			"success_threshold": 3,
			"timeout":          "30s",
			"max_requests":     10,
			"interval":         "10s",
		},
	}
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := viper.GetString(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if viper.IsSet(key) {
		return viper.GetInt(key)
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if viper.IsSet(key) {
		return viper.GetDuration(key)
	}
	return defaultValue
}

// Configuration validation utilities

// IsRustEngineAvailable checks if the Rust engine is available and responsive
func IsRustEngineAvailable(config *RustEngineConfiguration) bool {
	// This would implement a basic connectivity check
	// For now, just validate the configuration
	return config != nil && config.BaseURL != ""
}

// GetOptimalPerformanceTargets returns performance targets optimized for the current environment
func GetOptimalPerformanceTargets(environment string) *RustEnginePerformanceTargets {
	switch environment {
	case "production":
		return &RustEnginePerformanceTargets{
			DrugInteractionAnalysis: 25 * time.Millisecond, // More aggressive in production
			DosageCalculation:      15 * time.Millisecond,
			SafetyValidation:       40 * time.Millisecond,
			RuleEvaluation:         20 * time.Millisecond,
			HealthCheck:            250 * time.Millisecond,
			MetricsRetrieval:       500 * time.Millisecond,
		}
	case "development":
		return &RustEnginePerformanceTargets{
			DrugInteractionAnalysis: 100 * time.Millisecond, // More lenient in development
			DosageCalculation:      75 * time.Millisecond,
			SafetyValidation:       150 * time.Millisecond,
			RuleEvaluation:         100 * time.Millisecond,
			HealthCheck:            1 * time.Second,
			MetricsRetrieval:       2 * time.Second,
		}
	default:
		return &RustEnginePerformanceTargets{
			DrugInteractionAnalysis: 50 * time.Millisecond,
			DosageCalculation:      30 * time.Millisecond,
			SafetyValidation:       75 * time.Millisecond,
			RuleEvaluation:         40 * time.Millisecond,
			HealthCheck:            500 * time.Millisecond,
			MetricsRetrieval:       1 * time.Second,
		}
	}
}