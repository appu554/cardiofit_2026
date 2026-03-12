// Package config provides configuration for KB-18 Governance Engine
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the service
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Features FeatureConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port        string
	Environment string
	LogLevel    string
	ReadTimeout time.Duration
	WriteTimeout time.Duration
	IdleTimeout time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string
	Password string
	DB       int
	PoolSize int
}

// FeatureConfig holds feature flags
type FeatureConfig struct {
	EnablePatternMonitoring bool
	EnableAuditLogging      bool
	EnableMetrics           bool
	PatternThreshold24h     int
	PatternThreshold7d      int
}

// Load loads configuration from environment variables
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8018"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			LogLevel:     getEnv("LOG_LEVEL", "info"),
			ReadTimeout:  getDuration("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDuration("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDuration("IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5433/kb_governance?sslmode=disable"),
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6380"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
			PoolSize: getInt("REDIS_POOL_SIZE", 10),
		},
		Features: FeatureConfig{
			EnablePatternMonitoring: getBool("FEATURE_PATTERN_MONITORING", true),
			EnableAuditLogging:      getBool("FEATURE_AUDIT_LOGGING", true),
			EnableMetrics:           getBool("FEATURE_METRICS", true),
			PatternThreshold24h:     getInt("PATTERN_THRESHOLD_24H", 5),
			PatternThreshold7d:      getInt("PATTERN_THRESHOLD_7D", 20),
		},
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getInt gets an integer environment variable with a default value
func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getBool gets a boolean environment variable with a default value
func getBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

// getDuration gets a duration environment variable with a default value
func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
