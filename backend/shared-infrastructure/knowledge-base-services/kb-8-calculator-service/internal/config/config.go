// Package config provides configuration management for KB-8 Calculator Service.
//
// Configuration is loaded from environment variables with sensible defaults.
// No external config files required for basic operation.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for KB-8 Calculator Service.
type Config struct {
	// Server settings
	Port         int           `json:"port"`
	Environment  string        `json:"environment"`
	LogLevel     string        `json:"logLevel"`
	ReadTimeout  time.Duration `json:"readTimeout"`
	WriteTimeout time.Duration `json:"writeTimeout"`

	// Feature flags
	MetricsEnabled    bool `json:"metricsEnabled"`
	PlaygroundEnabled bool `json:"playgroundEnabled"`

	// Regional settings
	DefaultRegion      string `json:"defaultRegion"`
	IndiaAdjustments   bool   `json:"indiaAdjustments"`

	// Calculator settings
	MaxBatchSize int `json:"maxBatchSize"`
}

// Load loads configuration from environment variables with defaults.
func Load() (*Config, error) {
	cfg := &Config{
		// Server defaults
		Port:         getEnvAsInt("PORT", 8080),
		Environment:  getEnvWithDefault("ENVIRONMENT", "development"),
		LogLevel:     getEnvWithDefault("LOG_LEVEL", "info"),
		ReadTimeout:  getEnvAsDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvAsDuration("WRITE_TIMEOUT", 30*time.Second),

		// Feature flags
		MetricsEnabled:    getEnvAsBool("METRICS_ENABLED", true),
		PlaygroundEnabled: getEnvAsBool("PLAYGROUND_ENABLED", true),

		// Regional settings
		DefaultRegion:    getEnvWithDefault("DEFAULT_REGION", "GLOBAL"),
		IndiaAdjustments: getEnvAsBool("INDIA_ADJUSTMENTS", true),

		// Calculator settings
		MaxBatchSize: getEnvAsInt("MAX_BATCH_SIZE", 20),
	}

	return cfg, nil
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Helper functions for environment variable parsing

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
