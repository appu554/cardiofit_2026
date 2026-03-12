// Package config provides configuration management for KB-14 Care Navigator
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	KBServices  KBServicesConfig
	Escalation  EscalationConfig
	Workers     WorkerConfig
	Logging     LoggingConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port        string
	Environment string
	Version     string
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

// KBServicesConfig holds KB service client configuration
type KBServicesConfig struct {
	KB3Temporal   KBClientConfig
	KB9CareGaps   KBClientConfig
	KB12OrderSets KBClientConfig
}

// KBClientConfig holds configuration for a single KB service client
type KBClientConfig struct {
	URL     string
	Timeout time.Duration
	Enabled bool
}

// EscalationConfig holds escalation engine configuration
type EscalationConfig struct {
	CheckIntervalSeconds int
	Enabled              bool
}

// WorkerConfig holds background worker configuration
type WorkerConfig struct {
	SyncIntervalMinutes int
	Enabled             bool
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
	viper.SetDefault("PORT", "8091")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")

	// Database defaults
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)
	viper.SetDefault("DB_CONN_MAX_LIFETIME_MINUTES", 5)

	// Redis defaults
	viper.SetDefault("REDIS_DB", 0)

	// KB Service defaults
	viper.SetDefault("KB3_TEMPORAL_URL", "http://localhost:8087")
	viper.SetDefault("KB9_CARE_GAPS_URL", "http://localhost:8089")
	viper.SetDefault("KB12_ORDER_SETS_URL", "http://localhost:8090")
	viper.SetDefault("KB_CLIENT_TIMEOUT_SECONDS", 10)
	viper.SetDefault("KB3_ENABLED", true)
	viper.SetDefault("KB9_ENABLED", true)
	viper.SetDefault("KB12_ENABLED", true)

	// Worker defaults
	viper.SetDefault("ESCALATION_CHECK_INTERVAL", 60)
	viper.SetDefault("SYNC_INTERVAL_MINUTES", 5)
	viper.SetDefault("WORKERS_ENABLED", true)
	viper.SetDefault("ESCALATION_ENABLED", true)

	cfg := &Config{
		Server: ServerConfig{
			Port:        viper.GetString("PORT"),
			Environment: viper.GetString("ENVIRONMENT"),
			Version:     "1.0.0",
		},
		Database: DatabaseConfig{
			URL:             getRequiredEnv("DATABASE_URL"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(viper.GetInt("DB_CONN_MAX_LIFETIME_MINUTES")) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnvWithDefault("REDIS_URL", "redis://localhost:6386/0"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		KBServices: KBServicesConfig{
			KB3Temporal: KBClientConfig{
				URL:     viper.GetString("KB3_TEMPORAL_URL"),
				Timeout: time.Duration(viper.GetInt("KB_CLIENT_TIMEOUT_SECONDS")) * time.Second,
				Enabled: viper.GetBool("KB3_ENABLED"),
			},
			KB9CareGaps: KBClientConfig{
				URL:     viper.GetString("KB9_CARE_GAPS_URL"),
				Timeout: time.Duration(viper.GetInt("KB_CLIENT_TIMEOUT_SECONDS")) * time.Second,
				Enabled: viper.GetBool("KB9_ENABLED"),
			},
			KB12OrderSets: KBClientConfig{
				URL:     viper.GetString("KB12_ORDER_SETS_URL"),
				Timeout: time.Duration(viper.GetInt("KB_CLIENT_TIMEOUT_SECONDS")) * time.Second,
				Enabled: viper.GetBool("KB12_ENABLED"),
			},
		},
		Escalation: EscalationConfig{
			CheckIntervalSeconds: viper.GetInt("ESCALATION_CHECK_INTERVAL"),
			Enabled:              viper.GetBool("ESCALATION_ENABLED"),
		},
		Workers: WorkerConfig{
			SyncIntervalMinutes: viper.GetInt("SYNC_INTERVAL_MINUTES"),
			Enabled:             viper.GetBool("WORKERS_ENABLED"),
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

// getRequiredEnv retrieves a required environment variable or panics
func getRequiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		// In development, provide a default DATABASE_URL
		if key == "DATABASE_URL" && os.Getenv("ENVIRONMENT") != "production" {
			return "postgres://kb14user:kb14password@localhost:5438/kb_care_navigator?sslmode=disable"
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
