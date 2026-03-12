package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all KB-22 HPI Engine configuration.
// Pattern: environment-based (no Viper), matching KB-5.
type Config struct {
	Port        string
	Environment string

	// Database
	DatabaseURL      string
	DBMaxConnections int
	DBConnMaxLife    time.Duration

	// Redis
	RedisURL      string
	RedisPassword string
	RedisDB       int

	// Upstream KB dependencies
	KB20URL string // Patient Profile (required)
	KB21URL string // Behavioral Intelligence (optional)
	KB3URL  string // KB-3 Guidelines Repository (optional, prior adjustments + management summaries)
	KB5URL  string // KB-5 Drug Interactions (optional, medication safety / contraindications)

	// Downstream KB dependencies
	KB19URL string // Protocol Orchestrator
	KB23URL string // Decision Cards

	// Kafka telemetry
	KafkaEnabled          bool
	KafkaBootstrapServers string
	KafkaClientID         string

	// Timeouts (milliseconds)
	KB20TimeoutMS int
	KB21TimeoutMS int
	KB3TimeoutMS  int
	KB5TimeoutMS  int
	KB23TimeoutMS int

	// Session
	SessionTTLHours int
	NodesDir        string

	// Telemetry
	TelemetryMaxRetries   int
	TelemetryRetryDelay   time.Duration
	OutcomeRetryDelay     time.Duration
	SafetyAlertRetryDelay time.Duration
}

func Load() *Config {
	return &Config{
		Port:        envOrDefault("PORT", "8132"),
		Environment: envOrDefault("ENVIRONMENT", "development"),

		DatabaseURL:      envOrDefault("DATABASE_URL", "postgres://kb22_user:kb22_password@localhost:5437/kb_service_22?sslmode=disable"),
		DBMaxConnections: envIntOrDefault("DB_MAX_CONNECTIONS", 25),
		DBConnMaxLife:    envDurationOrDefault("DB_CONN_MAX_LIFETIME", 30*time.Minute),

		RedisURL:      envOrDefault("REDIS_URL", "redis://localhost:6386"),
		RedisPassword: envOrDefault("REDIS_PASSWORD", ""),
		RedisDB:       envIntOrDefault("REDIS_DB", 0),

		KB20URL: envOrDefault("KB20_URL", "http://localhost:8131"),
		KB21URL: envOrDefault("KB21_URL", "http://localhost:8133"),
		KB3URL:  envOrDefault("KB3_URL", "http://localhost:8087"),
		KB5URL:  envOrDefault("KB5_URL", "http://localhost:8089"),
		KB19URL: envOrDefault("KB19_URL", "http://localhost:8103"),
		KB23URL: envOrDefault("KB23_URL", "http://localhost:8134"),

		KafkaEnabled:          envBoolOrDefault("KAFKA_ENABLED", false),
		KafkaBootstrapServers: envOrDefault("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		KafkaClientID:         envOrDefault("KAFKA_CLIENT_ID", "kb22-hpi-engine"),

		KB20TimeoutMS: envIntOrDefault("KB20_TIMEOUT_MS", 40),
		KB21TimeoutMS: envIntOrDefault("KB21_TIMEOUT_MS", 40),
		KB3TimeoutMS:  envIntOrDefault("KB3_TIMEOUT_MS", 40),
		KB5TimeoutMS:  envIntOrDefault("KB5_TIMEOUT_MS", 30),
		KB23TimeoutMS: envIntOrDefault("KB23_TIMEOUT_MS", 40),

		SessionTTLHours: envIntOrDefault("SESSION_TTL_HOURS", 24),
		NodesDir:        envOrDefault("NODES_DIR", "./nodes"),

		TelemetryMaxRetries:   envIntOrDefault("TELEMETRY_MAX_RETRIES", 3),
		TelemetryRetryDelay:   envDurationOrDefault("TELEMETRY_RETRY_DELAY", 30*time.Second),
		OutcomeRetryDelay:     envDurationOrDefault("OUTCOME_RETRY_DELAY", 30*time.Second),
		SafetyAlertRetryDelay: envDurationOrDefault("SAFETY_ALERT_RETRY_DELAY", 5*time.Second),
	}
}

func (c *Config) IsDevelopment() bool { return c.Environment == "development" }
func (c *Config) IsProduction() bool  { return c.Environment == "production" }

func (c *Config) KB20Timeout() time.Duration {
	return time.Duration(c.KB20TimeoutMS) * time.Millisecond
}
func (c *Config) KB21Timeout() time.Duration {
	return time.Duration(c.KB21TimeoutMS) * time.Millisecond
}
func (c *Config) KB3Timeout() time.Duration { return time.Duration(c.KB3TimeoutMS) * time.Millisecond }
func (c *Config) KB5Timeout() time.Duration { return time.Duration(c.KB5TimeoutMS) * time.Millisecond }
func (c *Config) KB23Timeout() time.Duration {
	return time.Duration(c.KB23TimeoutMS) * time.Millisecond
}

func (c *Config) SessionTTL() time.Duration {
	return time.Duration(c.SessionTTLHours) * time.Hour
}

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

func envDurationOrDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
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
