// Package config provides configuration management for KB-3 service
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds all configuration for the KB-3 service
type Config struct {
	// Server settings
	Port        int
	ServiceName string
	Environment string
	LogLevel    string

	// Database settings
	DatabaseURL string
	DBMaxConns  int
	DBTimeout   time.Duration

	// Neo4j settings
	Neo4jURL      string
	Neo4jUser     string
	Neo4jPassword string

	// Redis settings
	RedisURL     string
	RedisTTL     time.Duration
	RedisTimeout time.Duration

	// Service settings
	CacheTTL        time.Duration
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		// Server
		Port:        getEnvInt("PORT", 8083),
		ServiceName: getEnv("SERVICE_NAME", "kb-3-guidelines"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// Database
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/kb3?sslmode=disable"),
		DBMaxConns:  getEnvInt("DB_MAX_CONNS", 25),
		DBTimeout:   getEnvDuration("DB_TIMEOUT", 30*time.Second),

		// Neo4j
		Neo4jURL:      getEnv("NEO4J_URL", "bolt://localhost:7687"),
		Neo4jUser:     getEnv("NEO4J_USER", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "password"),

		// Redis
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379"),
		RedisTTL:     getEnvDuration("REDIS_TTL", 30*time.Minute),
		RedisTimeout: getEnvDuration("REDIS_TIMEOUT", 5*time.Second),

		// Service
		CacheTTL:        getEnvDuration("CACHE_TTL", 30*time.Minute),
		RequestTimeout:  getEnvDuration("REQUEST_TIMEOUT", 30*time.Second),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	logrus.WithFields(logrus.Fields{
		"port":        c.Port,
		"environment": c.Environment,
		"db_url":      maskConnectionString(c.DatabaseURL),
		"neo4j_url":   c.Neo4jURL,
		"redis_url":   c.RedisURL,
	}).Info("Configuration loaded")

	return nil
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

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func maskConnectionString(connStr string) string {
	if len(connStr) > 20 {
		return connStr[:20] + "..."
	}
	return connStr
}
