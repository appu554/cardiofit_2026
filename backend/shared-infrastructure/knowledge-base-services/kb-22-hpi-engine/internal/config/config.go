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

	// CC-1: Safety Constraint Engine (sidecar on same host)
	SCEURL string // KB-24 SCE (default: http://localhost:8201)

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
	SCETimeoutMS  int

	// Session
	SessionTTLHours int
	NodesDir        string

	// Telemetry
	TelemetryMaxRetries   int
	TelemetryRetryDelay   time.Duration
	OutcomeRetryDelay     time.Duration
	SafetyAlertRetryDelay time.Duration

	// KB-26 Metabolic Digital Twin
	KB26URL string

	// PM/MD Node directories
	MonitoringNodesDir    string
	DeteriorationNodesDir string

	// Signal evaluation
	KB26TimeoutMS               int
	KB20ObservationTimeoutMS    int
	SignalDebounceTTLSec        int
	SignalPublisherRetryCount   int
	SignalPublisherRetryDelaySec int
	KafkaSignalTopic            string
	KB26StalenessDays           int
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
		KB5URL:  envOrDefault("KB5_URL", "http://localhost:8085"),
		KB19URL: envOrDefault("KB19_URL", "http://localhost:8103"),
		KB23URL: envOrDefault("KB23_URL", "http://localhost:8134"),
		SCEURL:  envOrDefault("SCE_URL", "http://localhost:8201"),

		KafkaEnabled:          envBoolOrDefault("KAFKA_ENABLED", false),
		KafkaBootstrapServers: envOrDefault("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		KafkaClientID:         envOrDefault("KAFKA_CLIENT_ID", "kb22-hpi-engine"),

		KB20TimeoutMS: envIntOrDefault("KB20_TIMEOUT_MS", 40),
		KB21TimeoutMS: envIntOrDefault("KB21_TIMEOUT_MS", 40),
		KB3TimeoutMS:  envIntOrDefault("KB3_TIMEOUT_MS", 40),
		KB5TimeoutMS:  envIntOrDefault("KB5_TIMEOUT_MS", 30),
		KB23TimeoutMS: envIntOrDefault("KB23_TIMEOUT_MS", 40),
		SCETimeoutMS:  envIntOrDefault("SCE_TIMEOUT_MS", 10),

		SessionTTLHours: envIntOrDefault("SESSION_TTL_HOURS", 24),
		NodesDir:        envOrDefault("NODES_DIR", "./nodes"),

		TelemetryMaxRetries:   envIntOrDefault("TELEMETRY_MAX_RETRIES", 3),
		TelemetryRetryDelay:   envDurationOrDefault("TELEMETRY_RETRY_DELAY", 30*time.Second),
		OutcomeRetryDelay:     envDurationOrDefault("OUTCOME_RETRY_DELAY", 30*time.Second),
		SafetyAlertRetryDelay: envDurationOrDefault("SAFETY_ALERT_RETRY_DELAY", 5*time.Second),

		KB26URL:                    envOrDefault("KB26_URL", "http://localhost:8137"),
		MonitoringNodesDir:         envOrDefault("MONITORING_NODES_DIR", "./pm-nodes"),
		DeteriorationNodesDir:      envOrDefault("DETERIORATION_NODES_DIR", "./deterioration"),
		KB26TimeoutMS:              envIntOrDefault("KB26_TIMEOUT_MS", 5000),
		KB20ObservationTimeoutMS:   envIntOrDefault("KB20_OBSERVATION_TIMEOUT_MS", 10000),
		SignalDebounceTTLSec:       envIntOrDefault("SIGNAL_DEBOUNCE_TTL_SEC", 300),
		SignalPublisherRetryCount:     envIntOrDefault("SIGNAL_PUBLISHER_RETRY_COUNT", 3),
		SignalPublisherRetryDelaySec:  envIntOrDefault("SIGNAL_PUBLISHER_RETRY_DELAY_SEC", 30),
		KafkaSignalTopic:           envOrDefault("KAFKA_SIGNAL_TOPIC", "clinical.signal.events"),
		KB26StalenessDays:          envIntOrDefault("KB26_STALENESS_DAYS", 21),
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
func (c *Config) SCETimeout() time.Duration {
	return time.Duration(c.SCETimeoutMS) * time.Millisecond
}

func (c *Config) KB26Timeout() time.Duration {
	return time.Duration(c.KB26TimeoutMS) * time.Millisecond
}
func (c *Config) KB20ObservationTimeout() time.Duration {
	return time.Duration(c.KB20ObservationTimeoutMS) * time.Millisecond
}
func (c *Config) KB26StalenessThreshold() time.Duration {
	return time.Duration(c.KB26StalenessDays) * 24 * time.Hour
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
