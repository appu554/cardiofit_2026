// Package config provides environment-based configuration for KB-24 Safety Constraint Engine.
// Pattern: env-variable loading with sensible defaults, matching KB-22.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all KB-24 SCE configuration values.
type Config struct {
	Port        string
	Environment string

	// Kafka telemetry
	KafkaEnabled          bool
	KafkaBootstrapServers string
	KafkaClientID         string

	// Logging
	LogLevel string

	// Node definitions — path to YAML directory containing safety triggers.
	// SCE only reads the safety_triggers field from each node YAML.
	NodeDefinitionPath string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:        envOrDefault("SCE_PORT", "8201"),
		Environment: envOrDefault("ENVIRONMENT", "development"),

		KafkaEnabled:          envBoolOrDefault("KAFKA_ENABLED", false),
		KafkaBootstrapServers: envOrDefault("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		KafkaClientID:         envOrDefault("KAFKA_CLIENT_ID", "kb24-sce"),

		LogLevel:           envOrDefault("LOG_LEVEL", "info"),
		NodeDefinitionPath: envOrDefault("NODE_DEFINITION_PATH", "./nodes"),
	}
}

// IsDevelopment returns true when running in development mode.
func (c *Config) IsDevelopment() bool { return c.Environment == "development" }

// IsProduction returns true when running in production mode.
func (c *Config) IsProduction() bool { return c.Environment == "production" }

// GetAddr returns the listen address in ":port" format.
func (c *Config) GetAddr() string {
	return fmt.Sprintf(":%s", c.Port)
}

// --- helpers ---

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envBoolOrDefault(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "t", "yes", "y", "on":
			return true
		case "0", "false", "f", "no", "n", "off":
			return false
		}
	}
	return def
}

// Suppress unused-import lint for envIntOrDefault (available for future config).
var _ = envIntOrDefault
