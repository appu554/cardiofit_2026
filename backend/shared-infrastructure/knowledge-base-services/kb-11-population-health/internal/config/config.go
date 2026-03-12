// Package config provides configuration management for KB-11 Population Health Engine.
// Uses Viper for environment variable loading with sensible defaults.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds all configuration for KB-11 Population Health Engine.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	Risk     RiskConfig
	Cohort   CohortConfig
	External ExternalConfig
	Logging  LoggingConfig
	Metrics  MetricsConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port         int
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds PostgreSQL connection configuration.
type DatabaseConfig struct {
	Host            string
	Port            int
	Name            string
	User            string
	Password        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// CacheConfig holds Redis cache configuration.
type CacheConfig struct {
	Enabled bool
	URL     string
	TTL     time.Duration
}

// RiskConfig holds risk engine configuration.
type RiskConfig struct {
	ModelsPath    string
	MaxConcurrent int
}

// CohortConfig holds cohort manager configuration.
type CohortConfig struct {
	MaxSize         int
	RefreshInterval time.Duration
}

// ExternalConfig holds external service URLs.
type ExternalConfig struct {
	FHIRStoreURL  string
	KB7URL        string
	KB13URL       string
	KB17URL       string
	KB18URL       string
	VaidshalaURL  string
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string
	Format string
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// Load reads configuration from environment variables with defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("KB11_PORT", 8111),
			Environment:  getEnv("KB11_ENVIRONMENT", "development"),
			ReadTimeout:  getEnvDuration("KB11_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("KB11_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getEnvDuration("KB11_IDLE_TIMEOUT", 60*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("KB11_DB_HOST", "localhost"),
			Port:            getEnvInt("KB11_DB_PORT", 5433),
			Name:            getEnv("KB11_DB_NAME", "kb11_population"),
			User:            getEnv("KB11_DB_USER", "postgres"),
			Password:        getEnv("KB11_DB_PASSWORD", "password"),
			SSLMode:         getEnv("KB11_DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("KB11_DB_MAX_CONNS", 50),
			MaxIdleConns:    getEnvInt("KB11_DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvDuration("KB11_DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Cache: CacheConfig{
			Enabled: getEnvBool("KB11_CACHE_ENABLED", true),
			URL:     getEnv("KB11_REDIS_URL", "redis://localhost:6380"),
			TTL:     getEnvDuration("KB11_CACHE_TTL", 15*time.Minute),
		},
		Risk: RiskConfig{
			ModelsPath:    getEnv("KB11_RISK_MODELS_PATH", "./models/risk-models"),
			MaxConcurrent: getEnvInt("KB11_MAX_CONCURRENT", 50),
		},
		Cohort: CohortConfig{
			MaxSize:         getEnvInt("KB11_MAX_COHORT_SIZE", 100000),
			RefreshInterval: getEnvDuration("KB11_COHORT_REFRESH", 1*time.Hour),
		},
		External: ExternalConfig{
			FHIRStoreURL: getEnv("FHIR_STORE_URL", ""),
			KB7URL:       getEnv("KB7_URL", "http://localhost:8092"),
			KB13URL:      getEnv("KB13_URL", "http://localhost:8113"),
			KB17URL:      getEnv("KB17_URL", "http://localhost:8117"),
			KB18URL:      getEnv("KB18_URL", "http://localhost:8118"),
			VaidshalaURL: getEnv("VAIDSHALA_URL", "http://localhost:8096"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("KB11_LOG_LEVEL", "info"),
			Format: getEnv("KB11_LOG_FORMAT", "json"),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvBool("KB11_METRICS_ENABLED", true),
			Path:    getEnv("KB11_METRICS_PATH", "/metrics"),
		},
	}

	return cfg, nil
}

// DSN returns the PostgreSQL connection string.
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode,
	)
}

// IsProduction returns true if running in production environment.
func (c *ServerConfig) IsProduction() bool {
	return c.Environment == "production"
}

// InitLogger initializes the global logger based on configuration.
func (c *LoggingConfig) InitLogger() *logrus.Entry {
	logger := logrus.New()

	// Set format
	if c.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Set level
	level, err := logrus.ParseLevel(c.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Output to stdout
	logger.SetOutput(os.Stdout)

	return logger.WithFields(logrus.Fields{
		"service": "kb-11-population-health",
		"version": "1.0.0",
	})
}

// Helper functions for environment variable parsing

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

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
