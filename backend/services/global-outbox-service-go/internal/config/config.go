package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the Global Outbox Service
type Config struct {
	// Service Configuration
	ProjectName string `mapstructure:"project_name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	Debug       bool   `mapstructure:"debug"`

	// Server Configuration
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	GRPCPort int    `mapstructure:"grpc_port"`

	// Database Configuration
	DatabaseURL         string `mapstructure:"database_url"`
	DatabasePoolSize    int    `mapstructure:"database_pool_size"`
	DatabaseMaxOverflow int    `mapstructure:"database_max_overflow"`
	DatabasePoolTimeout int    `mapstructure:"database_pool_timeout"`

	// Kafka Configuration
	KafkaBootstrapServers string `mapstructure:"kafka_bootstrap_servers"`
	KafkaAPIKey          string `mapstructure:"kafka_api_key"`
	KafkaAPISecret       string `mapstructure:"kafka_api_secret"`
	KafkaSecurityProtocol string `mapstructure:"kafka_security_protocol"`
	KafkaSASLMechanism   string `mapstructure:"kafka_sasl_mechanism"`

	// Publisher Configuration
	PublisherEnabled     bool          `mapstructure:"publisher_enabled"`
	PublisherPollInterval time.Duration `mapstructure:"publisher_poll_interval"`
	PublisherBatchSize   int           `mapstructure:"publisher_batch_size"`
	PublisherMaxWorkers  int           `mapstructure:"publisher_max_workers"`

	// Retry Configuration
	MaxRetryAttempts      int           `mapstructure:"max_retry_attempts"`
	RetryBaseDelay        time.Duration `mapstructure:"retry_base_delay"`
	RetryMaxDelay         time.Duration `mapstructure:"retry_max_delay"`
	RetryExponentialBase  float64       `mapstructure:"retry_exponential_base"`
	RetryJitter          bool          `mapstructure:"retry_jitter"`

	// Dead Letter Queue Configuration
	DLQEnabled    bool `mapstructure:"dlq_enabled"`
	DLQMaxRetries int  `mapstructure:"dlq_max_retries"`

	// Medical Circuit Breaker Configuration
	MedicalCircuitBreakerEnabled          bool    `mapstructure:"medical_circuit_breaker_enabled"`
	MedicalCircuitBreakerMaxQueueDepth    int     `mapstructure:"medical_circuit_breaker_max_queue_depth"`
	MedicalCircuitBreakerCriticalThreshold float64 `mapstructure:"medical_circuit_breaker_critical_threshold"`
	MedicalCircuitBreakerRecoveryTimeout   int     `mapstructure:"medical_circuit_breaker_recovery_timeout"`

	// Security Configuration
	GRPCAPIKey string `mapstructure:"grpc_api_key"`
	EnableAuth bool   `mapstructure:"enable_auth"`

	// Monitoring Configuration
	EnableMetrics bool   `mapstructure:"enable_metrics"`
	MetricsPort   int    `mapstructure:"metrics_port"`
	LogLevel      string `mapstructure:"log_level"`

	// Supported Services
	SupportedServices []string `mapstructure:"supported_services"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/global-outbox-service/")

	// Set defaults
	setDefaults()

	// Enable VIPER to read Environment Variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Environment-specific adjustments
	if config.IsProduction() {
		config.Debug = false
		config.LogLevel = "INFO"
	} else if config.IsDevelopment() {
		config.Debug = true
		config.LogLevel = "DEBUG"
	}

	return &config, nil
}

func setDefaults() {
	// Service Configuration
	viper.SetDefault("project_name", "Global Outbox Service Go")
	viper.SetDefault("version", "1.0.0")
	viper.SetDefault("environment", "development")
	viper.SetDefault("debug", false)

	// Server Configuration
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("port", 8042)
	viper.SetDefault("grpc_port", 50052)

	// Database Configuration
	viper.SetDefault("database_url", "postgresql://postgres.auugxeqzgrnknklgwqrh:9FTqQnA4LRCsu8sw@aws-0-ap-south-1.pooler.supabase.com:5432/postgres")
	viper.SetDefault("database_pool_size", 20)
	viper.SetDefault("database_max_overflow", 30)
	viper.SetDefault("database_pool_timeout", 30)

	// Kafka Configuration
	viper.SetDefault("kafka_bootstrap_servers", "pkc-619z3.us-east1.gcp.confluent.cloud:9092")
	viper.SetDefault("kafka_api_key", "LGJ3AQ2L6VRPW4S2")
	viper.SetDefault("kafka_api_secret", "2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl")
	viper.SetDefault("kafka_security_protocol", "SASL_SSL")
	viper.SetDefault("kafka_sasl_mechanism", "PLAIN")

	// Publisher Configuration
	viper.SetDefault("publisher_enabled", true)
	viper.SetDefault("publisher_poll_interval", "2s")
	viper.SetDefault("publisher_batch_size", 100)
	viper.SetDefault("publisher_max_workers", 4)

	// Retry Configuration
	viper.SetDefault("max_retry_attempts", 5)
	viper.SetDefault("retry_base_delay", "1s")
	viper.SetDefault("retry_max_delay", "60s")
	viper.SetDefault("retry_exponential_base", 2.0)
	viper.SetDefault("retry_jitter", true)

	// Dead Letter Queue Configuration
	viper.SetDefault("dlq_enabled", true)
	viper.SetDefault("dlq_max_retries", 10)

	// Medical Circuit Breaker Configuration
	viper.SetDefault("medical_circuit_breaker_enabled", true)
	viper.SetDefault("medical_circuit_breaker_max_queue_depth", 1000)
	viper.SetDefault("medical_circuit_breaker_critical_threshold", 0.8)
	viper.SetDefault("medical_circuit_breaker_recovery_timeout", 30)

	// Security Configuration
	viper.SetDefault("grpc_api_key", "global-outbox-service-go-key")
	viper.SetDefault("enable_auth", false)

	// Monitoring Configuration
	viper.SetDefault("enable_metrics", true)
	viper.SetDefault("metrics_port", 8043)
	viper.SetDefault("log_level", "INFO")

	// Supported Services
	viper.SetDefault("supported_services", []string{
		"patient-service",
		"observation-service",
		"condition-service",
		"medication-service",
		"encounter-service",
		"timeline-service",
		"workflow-engine-service",
		"order-management-service",
		"scheduling-service",
		"organization-service",
		"device-data-ingestion-service",
		"lab-service",
		"fhir-service",
		"generic-service",
		"ingestion-service",
	})
}

// IsProduction checks if running in production environment
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// IsDevelopment checks if running in development environment
func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

// GetKafkaConfig returns Kafka configuration map
func (c *Config) GetKafkaConfig() map[string]interface{} {
	return map[string]interface{}{
		"bootstrap.servers":  c.KafkaBootstrapServers,
		"security.protocol":  c.KafkaSecurityProtocol,
		"sasl.mechanism":     c.KafkaSASLMechanism,
		"sasl.username":      c.KafkaAPIKey,
		"sasl.password":      c.KafkaAPISecret,
		"client.id":          strings.ToLower(strings.ReplaceAll(c.ProjectName, " ", "-")) + "-producer",
		"acks":               "all",
		"retries":            3,
		"retry.backoff.ms":   1000,
		"request.timeout.ms": 30000,
		"delivery.timeout.ms": 120000,
	}
}