package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for KB-26 Metabolic Digital Twin Service.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig

	// Service identity
	ServiceName string
	Environment string
	LogLevel    string

	// Cross-KB integration
	KB19ProtocolURL        string
	KB20PatientProfileURL  string
	KB21BehavioralURL      string
	KB25LifestyleURL       string
	KB22HPIURL             string
	KB22SignalTimeoutMS    int
	KB23DecisionCardsURL   string

	// BP context market configuration
	MarketConfigDir string
	MarketCode      string

	// Twin computation
	ObservationWindowDays int
	BurnInWeeks           int

	// BP context daily batch scheduler (Phase 3)
	BPBatchEnabled     bool
	BPBatchHourUTC     int
	BPBatchConcurrency int
	BPActiveWindowDays int

	// Performance
	QueryTimeout    time.Duration
	MaxConnections  int
	ConnMaxLifetime time.Duration
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL             string
	Password        string
	MaxConnections  int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8137"),
		},
		ServiceName: "kb-26-metabolic-digital-twin",
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://kb_user:kb26_password@localhost:5443/kb26_mdt?sslmode=disable"),
			Password:        getEnv("DATABASE_PASSWORD", ""),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},

		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6394"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},

		// Cross-KB URLs
		KB19ProtocolURL:       getEnv("KB19_URL", "http://localhost:8103"),
		KB20PatientProfileURL: getEnv("KB20_URL", "http://localhost:8131"),
		KB21BehavioralURL:     getEnv("KB21_URL", "http://localhost:8133"),
		KB25LifestyleURL:      getEnv("KB25_URL", "http://localhost:8136"),
		KB22HPIURL:           getEnv("KB22_URL", "http://localhost:8132"),
		KB22SignalTimeoutMS:  getEnvAsInt("KB22_SIGNAL_TIMEOUT_MS", 500),
		KB23DecisionCardsURL: getEnv("KB23_URL", "http://localhost:8134"),

		// BP context market configuration
		MarketConfigDir: getEnv("MARKET_CONFIG_DIR", "../../market-configs"),
		MarketCode:      getEnv("MARKET_CODE", "shared"),

		// Twin computation defaults
		ObservationWindowDays: getEnvAsInt("OBSERVATION_WINDOW_DAYS", 14),
		BurnInWeeks:           getEnvAsInt("BURN_IN_WEEKS", 12),

		// BP context daily batch scheduler (Phase 3)
		BPBatchEnabled:     getEnv("BP_BATCH_ENABLED", "true") == "true",
		BPBatchHourUTC:     getEnvAsInt("BP_BATCH_HOUR_UTC", 2),
		BPBatchConcurrency: getEnvAsInt("BP_BATCH_CONCURRENCY", 10),
		BPActiveWindowDays: getEnvAsInt("BP_ACTIVE_WINDOW_DAYS", 30),

		// Performance
		QueryTimeout:    getEnvAsDuration("QUERY_TIMEOUT", "10s"),
		MaxConnections:  getEnvAsInt("MAX_CONNECTIONS", 25),
		ConnMaxLifetime: getEnvAsDuration("CONN_MAX_LIFETIME", "5m"),
	}

	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool { return c.Environment == "development" }
func (c *Config) IsProduction() bool  { return c.Environment == "production" }
func (c *Config) GetDatabaseDSN() string { return c.Database.URL }

// --- Environment helpers ---

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
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
