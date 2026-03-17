package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server      ServerConfig
	Neo4j       Neo4jConfig
	Redis       RedisConfig
	ServiceName string
	Environment string
	LogLevel    string

	// Cross-KB integration
	KB20PatientProfileURL string
	KB21BehavioralURL     string
	KB1DrugRulesURL       string
	KB4PatientSafetyURL   string

	// Cache TTLs
	CacheTTLChains  time.Duration
	CacheTTLPatient time.Duration

	// Performance
	QueryTimeout time.Duration
}

type ServerConfig struct {
	Port string
}

type Neo4jConfig struct {
	URI      string
	Database string
	User     string
	Password string
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8136"),
		},
		ServiceName: "kb-25-lifestyle-knowledge-graph",
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		Neo4j: Neo4jConfig{
			URI:      getEnv("NEO4J_URI", "bolt://localhost:7689"),
			Database: getEnv("NEO4J_DATABASE", "lkg"),
			User:     getEnv("NEO4J_USER", "neo4j"),
			Password: getEnv("NEO4J_PASSWORD", "kb25_lkg_password"),
		},

		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6389"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 25),
		},

		KB20PatientProfileURL: getEnv("KB20_URL", "http://localhost:8131"),
		KB21BehavioralURL:     getEnv("KB21_URL", "http://localhost:8133"),
		KB1DrugRulesURL:       getEnv("KB1_URL", "http://localhost:8081"),
		KB4PatientSafetyURL:   getEnv("KB4_URL", "http://localhost:8088"),

		CacheTTLChains:  getEnvAsDuration("CACHE_TTL_CHAINS", "3600s"),
		CacheTTLPatient: getEnvAsDuration("CACHE_TTL_PATIENT", "300s"),

		QueryTimeout: getEnvAsDuration("QUERY_TIMEOUT", "10s"),
	}

	if cfg.Neo4j.URI == "" {
		return nil, fmt.Errorf("NEO4J_URI is required")
	}
	return cfg, nil
}

func (c *Config) IsDevelopment() bool { return c.Environment == "development" }
func (c *Config) IsProduction() bool  { return c.Environment == "production" }

func getEnv(key, defaultValue string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvAsDuration(key, defaultValue string) time.Duration {
	val := getEnv(key, defaultValue)
	d, err := time.ParseDuration(val)
	if err != nil {
		d, _ = time.ParseDuration(defaultValue)
	}
	return d
}
