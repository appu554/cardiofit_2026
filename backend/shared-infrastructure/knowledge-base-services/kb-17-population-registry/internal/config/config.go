// Package config provides configuration management for KB-17 Population Registry
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Kafka      KafkaConfig
	KBServices KBServicesConfig
	Workers    WorkerConfig
	Logging    LoggingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port         int
	Environment  string
	Version      string
	LogLevel     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis cache configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
	Prefix   string
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers string
	GroupID string
	Enabled bool
}

// KBServicesConfig holds KB service client configuration
type KBServicesConfig struct {
	KB2  KBClientConfig
	KB8  KBClientConfig
	KB9  KBClientConfig
	KB14 KBClientConfig
}

// KBClientConfig holds configuration for a single KB service client
type KBClientConfig struct {
	URL     string
	Timeout time.Duration
	Enabled bool
}

// WorkerConfig holds background worker configuration
type WorkerConfig struct {
	ReevaluationIntervalMinutes int
	CareGapSyncIntervalMinutes  int
	Enabled                     bool
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("PORT", 8017)
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")
	viper.SetDefault("READ_TIMEOUT_SECONDS", 30)
	viper.SetDefault("WRITE_TIMEOUT_SECONDS", 30)

	// Database defaults
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME_MINUTES", 5)

	// Redis defaults
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_PREFIX", "kb17:")

	// Kafka defaults
	viper.SetDefault("KAFKA_GROUP_ID", "kb17-population-registry")
	viper.SetDefault("KAFKA_ENABLED", true)

	// KB Service defaults (actual ports from KB service configs)
	viper.SetDefault("KB2_URL", "http://localhost:8082")   // KB-2 Clinical Context
	viper.SetDefault("KB8_URL", "http://localhost:8080")   // KB-8 Calculator Service
	viper.SetDefault("KB9_URL", "http://localhost:8089")   // KB-9 Care Gaps
	viper.SetDefault("KB14_URL", "http://localhost:8091")  // KB-14 Care Navigator
	viper.SetDefault("KB_CLIENT_TIMEOUT_SECONDS", 10)
	viper.SetDefault("KB2_ENABLED", true)
	viper.SetDefault("KB8_ENABLED", true)
	viper.SetDefault("KB9_ENABLED", true)
	viper.SetDefault("KB14_ENABLED", true)

	// Worker defaults
	viper.SetDefault("REEVALUATION_INTERVAL_MINUTES", 60)
	viper.SetDefault("CARE_GAP_SYNC_INTERVAL_MINUTES", 30)
	viper.SetDefault("WORKERS_ENABLED", true)

	cfg := &Config{
		Server: ServerConfig{
			Port:         viper.GetInt("PORT"),
			Environment:  viper.GetString("ENVIRONMENT"),
			Version:      "1.0.0",
			LogLevel:     viper.GetString("LOG_LEVEL"),
			ReadTimeout:  time.Duration(viper.GetInt("READ_TIMEOUT_SECONDS")) * time.Second,
			WriteTimeout: time.Duration(viper.GetInt("WRITE_TIMEOUT_SECONDS")) * time.Second,
		},
		Database: DatabaseConfig{
			URL:             getRequiredEnv("DATABASE_URL"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(viper.GetInt("DB_CONN_MAX_LIFETIME_MINUTES")) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnvWithDefault("REDIS_URL", "redis://localhost:6379/0"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
			Prefix:   viper.GetString("REDIS_PREFIX"),
		},
		Kafka: KafkaConfig{
			Brokers: getEnvWithDefault("KAFKA_BROKERS", "localhost:9092"),
			GroupID: viper.GetString("KAFKA_GROUP_ID"),
			Enabled: viper.GetBool("KAFKA_ENABLED"),
		},
		KBServices: KBServicesConfig{
			KB2: KBClientConfig{
				URL:     viper.GetString("KB2_URL"),
				Timeout: time.Duration(viper.GetInt("KB_CLIENT_TIMEOUT_SECONDS")) * time.Second,
				Enabled: viper.GetBool("KB2_ENABLED"),
			},
			KB8: KBClientConfig{
				URL:     viper.GetString("KB8_URL"),
				Timeout: time.Duration(viper.GetInt("KB_CLIENT_TIMEOUT_SECONDS")) * time.Second,
				Enabled: viper.GetBool("KB8_ENABLED"),
			},
			KB9: KBClientConfig{
				URL:     viper.GetString("KB9_URL"),
				Timeout: time.Duration(viper.GetInt("KB_CLIENT_TIMEOUT_SECONDS")) * time.Second,
				Enabled: viper.GetBool("KB9_ENABLED"),
			},
			KB14: KBClientConfig{
				URL:     viper.GetString("KB14_URL"),
				Timeout: time.Duration(viper.GetInt("KB_CLIENT_TIMEOUT_SECONDS")) * time.Second,
				Enabled: viper.GetBool("KB14_ENABLED"),
			},
		},
		Workers: WorkerConfig{
			ReevaluationIntervalMinutes: viper.GetInt("REEVALUATION_INTERVAL_MINUTES"),
			CareGapSyncIntervalMinutes:  viper.GetInt("CARE_GAP_SYNC_INTERVAL_MINUTES"),
			Enabled:                     viper.GetBool("WORKERS_ENABLED"),
		},
		Logging: LoggingConfig{
			Level:  viper.GetString("LOG_LEVEL"),
			Format: viper.GetString("LOG_FORMAT"),
		},
	}

	return cfg, nil
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// getRequiredEnv retrieves a required environment variable or returns a default for development
func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// In development, provide defaults
		if key == "DATABASE_URL" && os.Getenv("ENVIRONMENT") != "production" {
			return "postgres://kb17user:kb17password@localhost:5439/kb_population_registry?sslmode=disable"
		}
		panic(fmt.Sprintf("Required environment variable %s is not set", key))
	}
	return value
}

// getEnvWithDefault retrieves an environment variable with a default value
func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
