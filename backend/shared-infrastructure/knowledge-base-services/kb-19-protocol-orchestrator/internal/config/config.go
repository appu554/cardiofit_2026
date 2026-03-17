// Package config provides configuration management for KB-19 Protocol Orchestrator.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for KB-19 Protocol Orchestrator.
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Cache       CacheConfig
	Metrics     MetricsConfig
	Logging     LoggingConfig
	KBServices  KBServicesConfig
	Vaidshala   VaidshalaConfig
	Arbitration ArbitrationConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port         int
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds PostgreSQL configuration for decision audit storage.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// CacheConfig holds Redis caching configuration.
type CacheConfig struct {
	Enabled    bool
	URL        string
	TTLSeconds int
	MaxEntries int
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string
	Format string
}

// KBServicesConfig holds URLs for all dependent KB services.
type KBServicesConfig struct {
	// KB-3 Guidelines Service (temporal reasoning)
	KB3URL string

	// KB-5 Drug-Drug Interaction Service
	KB5URL string

	// KB-7 Terminology Service (RxNorm resolution)
	KB7URL string

	// KB-8 Calculator Service (clinical calculators)
	KB8URL string

	// KB-12 OrderSets/CarePlans Service
	KB12URL string

	// KB-14 Care Navigator (governance)
	KB14URL string

	// Medication Advisor Engine (V3: Risk Computer / Judge)
	MedicationAdvisorURL string

	// V-MCU Clinical Runtime (event forwarding target for MCU_GATE_CHANGED)
	VMCUURL string

	// Timeout for KB service calls
	Timeout time.Duration

	// Retry configuration
	MaxRetries int
	RetryDelay time.Duration
}

// VaidshalaConfig holds Vaidshala CQL Engine configuration.
type VaidshalaConfig struct {
	// CQL Engine URL
	CQLEngineURL string

	// ICU Intelligence URL
	ICUIntelligenceURL string

	// Medication Advisor URL
	MedicationAdvisorURL string

	// Timeout for Vaidshala calls
	Timeout time.Duration
}

// ArbitrationConfig holds arbitration engine configuration.
type ArbitrationConfig struct {
	// Path to protocol definitions (YAML files)
	ProtocolsPath string

	// Path to conflict matrix definitions (YAML files)
	ConflictsPath string

	// Enable parallel protocol evaluation
	ParallelEvaluation bool

	// Max concurrent evaluations
	MaxConcurrent int

	// Default confidence threshold for recommendations
	ConfidenceThreshold float64

	// Enable strict mode (fail on any error vs. partial results)
	StrictMode bool
}

// Load reads configuration from environment variables and returns a Config.
func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("SERVER_PORT", 8103)
	viper.SetDefault("SERVER_ENVIRONMENT", "development")
	viper.SetDefault("SERVER_READ_TIMEOUT", 30)
	viper.SetDefault("SERVER_WRITE_TIMEOUT", 30)
	viper.SetDefault("SERVER_IDLE_TIMEOUT", 120)

	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5432)
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "password")
	viper.SetDefault("DB_NAME", "kb_protocol_orchestrator")
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", 300)

	viper.SetDefault("CACHE_ENABLED", true)
	viper.SetDefault("CACHE_URL", "redis://localhost:6379")
	viper.SetDefault("CACHE_TTL_SECONDS", 300)
	viper.SetDefault("CACHE_MAX_ENTRIES", 10000)

	viper.SetDefault("METRICS_ENABLED", true)
	viper.SetDefault("METRICS_PATH", "/metrics")

	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")

	// KB Services defaults
	viper.SetDefault("KB3_URL", "http://localhost:8087")
	viper.SetDefault("KB5_URL", "http://localhost:8089")
	viper.SetDefault("KB7_URL", "http://localhost:8092")
	viper.SetDefault("KB8_URL", "http://localhost:8093")
	viper.SetDefault("KB12_URL", "http://localhost:8097")
	viper.SetDefault("KB14_URL", "http://localhost:8099")
	viper.SetDefault("MEDICATION_ADVISOR_URL", "http://localhost:8089") // V3: Risk Computer
	viper.SetDefault("VMCU_URL", "http://localhost:8090")             // Clinical Runtime (V-MCU event target)
	viper.SetDefault("KB_TIMEOUT", 30)
	viper.SetDefault("KB_MAX_RETRIES", 3)
	viper.SetDefault("KB_RETRY_DELAY", 1)

	// Vaidshala defaults
	viper.SetDefault("VAIDSHALA_CQL_URL", "http://localhost:9000")
	viper.SetDefault("VAIDSHALA_ICU_URL", "http://localhost:9001")
	viper.SetDefault("VAIDSHALA_MED_ADVISOR_URL", "http://localhost:9002")
	viper.SetDefault("VAIDSHALA_TIMEOUT", 60)

	// Arbitration defaults
	viper.SetDefault("PROTOCOLS_PATH", "./protocols")
	viper.SetDefault("CONFLICTS_PATH", "./conflicts")
	viper.SetDefault("PARALLEL_EVALUATION", true)
	viper.SetDefault("MAX_CONCURRENT", 10)
	viper.SetDefault("CONFIDENCE_THRESHOLD", 0.7)
	viper.SetDefault("STRICT_MODE", false)

	viper.AutomaticEnv()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("PORT", 8103),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  time.Duration(getEnvInt("SERVER_READ_TIMEOUT", 30)) * time.Second,
			WriteTimeout: time.Duration(getEnvInt("SERVER_WRITE_TIMEOUT", 30)) * time.Second,
			IdleTimeout:  time.Duration(getEnvInt("SERVER_IDLE_TIMEOUT", 120)) * time.Second,
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "password"),
			Database:        getEnv("DB_NAME", "kb_protocol_orchestrator"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 300)) * time.Second,
		},
		Cache: CacheConfig{
			Enabled:    getEnvBool("CACHE_ENABLED", true),
			URL:        getEnv("CACHE_URL", "redis://localhost:6379"),
			TTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 300),
			MaxEntries: getEnvInt("CACHE_MAX_ENTRIES", 10000),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvBool("METRICS_ENABLED", true),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		KBServices: KBServicesConfig{
			KB3URL:               getEnv("KB3_URL", "http://localhost:8087"),
			KB5URL:               getEnv("KB5_URL", "http://localhost:8089"),
			KB7URL:               getEnv("KB7_URL", "http://localhost:8092"),
			KB8URL:               getEnv("KB8_URL", "http://localhost:8093"),
			KB12URL:              getEnv("KB12_URL", "http://localhost:8097"),
			KB14URL:              getEnv("KB14_URL", "http://localhost:8099"),
			MedicationAdvisorURL: getEnv("MEDICATION_ADVISOR_URL", "http://localhost:8089"),
			VMCUURL:              getEnv("VMCU_URL", "http://localhost:8090"),
			Timeout:              time.Duration(getEnvInt("KB_TIMEOUT", 30)) * time.Second,
			MaxRetries:           getEnvInt("KB_MAX_RETRIES", 3),
			RetryDelay:           time.Duration(getEnvInt("KB_RETRY_DELAY", 1)) * time.Second,
		},
		Vaidshala: VaidshalaConfig{
			CQLEngineURL:         getEnv("VAIDSHALA_CQL_URL", "http://localhost:9000"),
			ICUIntelligenceURL:   getEnv("VAIDSHALA_ICU_URL", "http://localhost:9001"),
			MedicationAdvisorURL: getEnv("VAIDSHALA_MED_ADVISOR_URL", "http://localhost:9002"),
			Timeout:              time.Duration(getEnvInt("VAIDSHALA_TIMEOUT", 60)) * time.Second,
		},
		Arbitration: ArbitrationConfig{
			ProtocolsPath:       getEnv("PROTOCOLS_PATH", "./protocols"),
			ConflictsPath:       getEnv("CONFLICTS_PATH", "./conflicts"),
			ParallelEvaluation:  getEnvBool("PARALLEL_EVALUATION", true),
			MaxConcurrent:       getEnvInt("MAX_CONCURRENT", 10),
			ConfidenceThreshold: getEnvFloat("CONFIDENCE_THRESHOLD", 0.7),
			StrictMode:          getEnvBool("STRICT_MODE", false),
		},
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + strconv.Itoa(c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Database +
		" sslmode=" + c.SSLMode
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
