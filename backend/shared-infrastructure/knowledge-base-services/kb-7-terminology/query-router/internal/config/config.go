package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the query router service
type Config struct {
	// Server configuration
	APIPort     string
	MetricsPort string

	// Database connections
	PostgresURL       string
	RedisURL          string
	GraphDBEndpoint   string
	ElasticsearchURL  string
	ElasticsearchIndex string

	// Cache configuration
	DefaultCacheTTL time.Duration
	MaxCacheSize    int

	// Performance settings
	MaxConcurrentQueries int
	QueryTimeout         time.Duration

	// Circuit breaker settings
	CircuitBreakerTimeout   time.Duration
	CircuitBreakerThreshold int

	// Monitoring
	JaegerEndpoint string
	LogLevel       string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// Default values
		APIPort:                 getEnv("API_PORT", "8087"),
		MetricsPort:             getEnv("METRICS_PORT", "8088"),
		PostgresURL:             getEnv("POSTGRES_URL", "postgres://postgres:password@localhost:5432/kb7_terminology?sslmode=disable"),
		RedisURL:                getEnv("REDIS_URL", "redis://localhost:6379"),
		GraphDBEndpoint:         getEnv("GRAPHDB_ENDPOINT", "http://localhost:7200"),
		ElasticsearchURL:        getEnv("ELASTICSEARCH_URL", "http://localhost:9200"),
		ElasticsearchIndex:      getEnv("ELASTICSEARCH_INDEX", "clinical_terms"),
		DefaultCacheTTL:         parseDuration(getEnv("DEFAULT_CACHE_TTL", "1h")),
		MaxCacheSize:            parseInt(getEnv("MAX_CACHE_SIZE", "10000")),
		MaxConcurrentQueries:    parseInt(getEnv("MAX_CONCURRENT_QUERIES", "100")),
		QueryTimeout:            parseDuration(getEnv("QUERY_TIMEOUT", "30s")),
		CircuitBreakerTimeout:   parseDuration(getEnv("CIRCUIT_BREAKER_TIMEOUT", "60s")),
		CircuitBreakerThreshold: parseInt(getEnv("CIRCUIT_BREAKER_THRESHOLD", "5")),
		JaegerEndpoint:          getEnv("JAEGER_ENDPOINT", "http://localhost:14268/api/traces"),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}