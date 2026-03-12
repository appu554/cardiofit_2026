// Package config provides configuration management for KB-1 Drug Rules Service.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service.
type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	RxNav     RxNavConfig   // Direct RxNav API (replaces KB-7)
	KB4       KB4Config     // Patient Safety Service integration
	Ingestion IngestionConfig
	Cache     CacheConfig
	Metrics   MetricsConfig
	Logging   LoggingConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port         int
	Environment  string
	ReadTimeout  int
	WriteTimeout int
}

// DatabaseConfig holds PostgreSQL database configuration.
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ConnectionString returns the PostgreSQL connection string.
func (c DatabaseConfig) ConnectionString() string {
	return "host=" + c.Host +
		" port=" + strconv.Itoa(c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Database +
		" sslmode=" + c.SSLMode
}

// RedisConfig holds Redis cache configuration.
type RedisConfig struct {
	Host        string
	Port        int
	Password    string
	DB          int
	MaxRetries  int
	PoolSize    int
	DialTimeout time.Duration
	ReadTimeout time.Duration
	Enabled     bool
}

// RxNavConfig holds RxNav API configuration (rxnav-in-a-box).
// Replaces KB-7 dependency - connects directly to local RxNav instance.
type RxNavConfig struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	Enabled    bool
}

// KB4Config holds KB-4 Patient Safety Service configuration.
// KB-4 provides comprehensive safety checks including:
// - Black Box Warnings, Contraindications, Dose/Age Limits
// - Pregnancy/Lactation Safety, High-Alert Status
// - Beers Criteria, Anticholinergic Burden, Lab Requirements
type KB4Config struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	Enabled    bool
}

// IngestionConfig holds FDA/TGA/CDSCO ingestion configuration.
type IngestionConfig struct {
	Concurrency  int
	BatchSize    int
	RateLimitMs  int
	FDABaseURL   string
	TGABaseURL   string
	CDSCOBaseURL string
}

// CacheConfig holds caching configuration.
type CacheConfig struct {
	Enabled    bool
	TTLSeconds int
	MaxEntries int
}

// MetricsConfig holds Prometheus metrics configuration.
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	Level  string
	Format string
}

// Load reads configuration from environment variables and defaults.
func Load() (*Config, error) {
	viper.SetDefault("SERVER_PORT", 8081)
	viper.SetDefault("SERVER_ENVIRONMENT", "development")
	viper.SetDefault("SERVER_READ_TIMEOUT", 30)
	viper.SetDefault("SERVER_WRITE_TIMEOUT", 30)

	// Database defaults
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", 5481)
	viper.SetDefault("DB_USER", "kb1_user")
	viper.SetDefault("DB_PASSWORD", "kb1_password")
	viper.SetDefault("DB_NAME", "kb1_drug_rules")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 25)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 5)

	// Redis defaults
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", 6380)
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_ENABLED", true)

	// RxNav defaults (rxnav-in-a-box on port 4000)
	viper.SetDefault("RXNAV_URL", "http://localhost:4000/REST")
	viper.SetDefault("RXNAV_TIMEOUT", 30)
	viper.SetDefault("RXNAV_ENABLED", true)

	// KB-4 Patient Safety defaults
	viper.SetDefault("KB4_URL", "http://localhost:8088")
	viper.SetDefault("KB4_TIMEOUT", 30)
	viper.SetDefault("KB4_ENABLED", true)

	// Ingestion defaults
	viper.SetDefault("INGESTION_CONCURRENCY", 10)
	viper.SetDefault("INGESTION_BATCH_SIZE", 100)
	viper.SetDefault("INGESTION_RATE_LIMIT_MS", 100)
	viper.SetDefault("FDA_BASE_URL", "https://dailymed.nlm.nih.gov/dailymed/services/v2")
	viper.SetDefault("TGA_BASE_URL", "https://www.tga.gov.au/api")
	viper.SetDefault("CDSCO_BASE_URL", "https://cdsco.gov.in/api")

	// Cache defaults
	viper.SetDefault("CACHE_ENABLED", true)
	viper.SetDefault("CACHE_TTL_SECONDS", 300)
	viper.SetDefault("CACHE_MAX_ENTRIES", 1000)

	// Metrics defaults
	viper.SetDefault("METRICS_ENABLED", true)
	viper.SetDefault("METRICS_PATH", "/metrics")

	// Logging defaults
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")

	viper.AutomaticEnv()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnvInt("PORT", 8081),
			Environment:  getEnv("ENVIRONMENT", "development"),
			ReadTimeout:  getEnvInt("SERVER_READ_TIMEOUT", 30),
			WriteTimeout: getEnvInt("SERVER_WRITE_TIMEOUT", 30),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5481),
			User:            getEnv("DB_USER", "kb1_user"),
			Password:        getEnv("DB_PASSWORD", "kb1_password"),
			Database:        getEnv("DB_NAME", "kb1_drug_rules"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_MIN", 5)) * time.Minute,
			ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_TIME_MIN", 1)) * time.Minute,
		},
		Redis: RedisConfig{
			Host:        getEnv("REDIS_HOST", "localhost"),
			Port:        getEnvInt("REDIS_PORT", 6380),
			Password:    getEnv("REDIS_PASSWORD", ""),
			DB:          getEnvInt("REDIS_DB", 0),
			MaxRetries:  getEnvInt("REDIS_MAX_RETRIES", 3),
			PoolSize:    getEnvInt("REDIS_POOL_SIZE", 10),
			DialTimeout: time.Duration(getEnvInt("REDIS_DIAL_TIMEOUT_SEC", 5)) * time.Second,
			ReadTimeout: time.Duration(getEnvInt("REDIS_READ_TIMEOUT_SEC", 3)) * time.Second,
			Enabled:     getEnvBool("REDIS_ENABLED", true),
		},
		RxNav: RxNavConfig{
			BaseURL:    getEnv("RXNAV_URL", "http://localhost:4000/REST"),
			Timeout:    time.Duration(getEnvInt("RXNAV_TIMEOUT_SEC", 30)) * time.Second,
			MaxRetries: getEnvInt("RXNAV_MAX_RETRIES", 3),
			RetryDelay: time.Duration(getEnvInt("RXNAV_RETRY_DELAY_MS", 500)) * time.Millisecond,
			Enabled:    getEnvBool("RXNAV_ENABLED", true),
		},
		KB4: KB4Config{
			BaseURL:    getEnv("KB4_URL", "http://localhost:8088"),
			Timeout:    time.Duration(getEnvInt("KB4_TIMEOUT_SEC", 30)) * time.Second,
			MaxRetries: getEnvInt("KB4_MAX_RETRIES", 3),
			RetryDelay: time.Duration(getEnvInt("KB4_RETRY_DELAY_MS", 500)) * time.Millisecond,
			Enabled:    getEnvBool("KB4_ENABLED", true),
		},
		Ingestion: IngestionConfig{
			Concurrency:  getEnvInt("INGESTION_CONCURRENCY", 10),
			BatchSize:    getEnvInt("INGESTION_BATCH_SIZE", 100),
			RateLimitMs:  getEnvInt("INGESTION_RATE_LIMIT_MS", 100),
			FDABaseURL:   getEnv("FDA_BASE_URL", "https://dailymed.nlm.nih.gov/dailymed/services/v2"),
			TGABaseURL:   getEnv("TGA_BASE_URL", "https://www.tga.gov.au/api"),
			CDSCOBaseURL: getEnv("CDSCO_BASE_URL", "https://cdsco.gov.in/api"),
		},
		Cache: CacheConfig{
			Enabled:    getEnvBool("CACHE_ENABLED", true),
			TTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 300),
			MaxEntries: getEnvInt("CACHE_MAX_ENTRIES", 1000),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvBool("METRICS_ENABLED", true),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	return cfg, nil
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

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
