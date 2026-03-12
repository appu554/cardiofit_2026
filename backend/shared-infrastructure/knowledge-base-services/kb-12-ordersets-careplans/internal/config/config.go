// Package config provides configuration management for KB-12 Order Sets & Care Plans Service
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds all configuration for the KB-12 service
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	KBServices  KBServicesConfig
	Logging     LoggingConfig
	HealthCheck HealthCheckConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int
	Host            string
	Environment     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	MaxRequestSize  int64
	TrustedProxies  []string
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	URL             string
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
	MigrationsPath  string
}

// RedisConfig holds Redis cache configuration
type RedisConfig struct {
	URL              string
	Host             string
	Port             int
	Password         string
	Database         int
	MaxRetries       int
	PoolSize         int
	MinIdleConns     int
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PoolTimeout      time.Duration
	DefaultTTL       time.Duration
	OrderSetTTL      time.Duration
	CarePlanTTL      time.Duration
	TemplateTTL      time.Duration
}

// KBServicesConfig holds HTTP client configuration for other KB services
type KBServicesConfig struct {
	KB1Dosing      KBClientConfig
	KB3Temporal    KBClientConfig
	KB6Formulary   KBClientConfig
	KB7Terminology KBClientConfig
}

// KBClientConfig holds configuration for an individual KB service client
type KBClientConfig struct {
	BaseURL        string
	Timeout        time.Duration
	MaxRetries     int
	RetryWaitMin   time.Duration
	RetryWaitMax   time.Duration
	MaxIdleConns   int
	IdleConnTimeout time.Duration
	Enabled        bool
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string
	Format     string
	Output     string
	TimeFormat string
}

// HealthCheckConfig holds health check configuration
type HealthCheckConfig struct {
	Interval       time.Duration
	Timeout        time.Duration
	FailureThreshold int
}

// Load reads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            getEnvInt("PORT", 8090),
			Host:            getEnv("HOST", "0.0.0.0"),
			Environment:     getEnv("ENVIRONMENT", "development"),
			ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
			MaxRequestSize:  getEnvInt64("SERVER_MAX_REQUEST_SIZE", 10*1024*1024), // 10MB
			TrustedProxies:  []string{"127.0.0.1", "::1"},
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", ""),
			Host:            getEnv("POSTGRES_HOST", "localhost"),
			Port:            getEnvInt("POSTGRES_PORT", 5437),
			User:            getEnv("POSTGRES_USER", "kb12_user"),
			Password:        getEnv("POSTGRES_PASSWORD", "kb12_password"),
			Database:        getEnv("POSTGRES_DB", "kb12_ordersets"),
			SSLMode:         getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("POSTGRES_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("POSTGRES_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvDuration("POSTGRES_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvDuration("POSTGRES_CONN_MAX_IDLE_TIME", 3*time.Minute),
			MigrationsPath:  getEnv("MIGRATIONS_PATH", "./migrations"),
		},
		Redis: RedisConfig{
			URL:              getEnv("REDIS_URL", ""),
			Host:             getEnv("REDIS_HOST", "localhost"),
			Port:             getEnvInt("REDIS_PORT", 6385),
			Password:         getEnv("REDIS_PASSWORD", ""),
			Database:         getEnvInt("REDIS_DB", 0),
			MaxRetries:       getEnvInt("REDIS_MAX_RETRIES", 3),
			PoolSize:         getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns:     getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
			DialTimeout:      getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:      getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout:     getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolTimeout:      getEnvDuration("REDIS_POOL_TIMEOUT", 4*time.Second),
			DefaultTTL:       getEnvDuration("REDIS_DEFAULT_TTL", 1*time.Hour),
			OrderSetTTL:      getEnvDuration("REDIS_ORDERSET_TTL", 24*time.Hour),
			CarePlanTTL:      getEnvDuration("REDIS_CAREPLAN_TTL", 12*time.Hour),
			TemplateTTL:      getEnvDuration("REDIS_TEMPLATE_TTL", 6*time.Hour),
		},
		KBServices: KBServicesConfig{
			KB1Dosing: KBClientConfig{
				BaseURL:         getEnv("KB1_DOSING_URL", "http://kb-drug-rules:8081"),
				Timeout:         getEnvDuration("KB1_TIMEOUT", 10*time.Second),
				MaxRetries:      getEnvInt("KB1_MAX_RETRIES", 3),
				RetryWaitMin:    getEnvDuration("KB1_RETRY_WAIT_MIN", 100*time.Millisecond),
				RetryWaitMax:    getEnvDuration("KB1_RETRY_WAIT_MAX", 2*time.Second),
				MaxIdleConns:    getEnvInt("KB1_MAX_IDLE_CONNS", 10),
				IdleConnTimeout: getEnvDuration("KB1_IDLE_CONN_TIMEOUT", 90*time.Second),
				Enabled:         getEnvBool("KB1_ENABLED", true),
			},
			KB3Temporal: KBClientConfig{
				BaseURL:         getEnv("KB3_TEMPORAL_URL", "http://kb3-guidelines:8083"),
				Timeout:         getEnvDuration("KB3_TIMEOUT", 15*time.Second),
				MaxRetries:      getEnvInt("KB3_MAX_RETRIES", 3),
				RetryWaitMin:    getEnvDuration("KB3_RETRY_WAIT_MIN", 100*time.Millisecond),
				RetryWaitMax:    getEnvDuration("KB3_RETRY_WAIT_MAX", 2*time.Second),
				MaxIdleConns:    getEnvInt("KB3_MAX_IDLE_CONNS", 10),
				IdleConnTimeout: getEnvDuration("KB3_IDLE_CONN_TIMEOUT", 90*time.Second),
				Enabled:         getEnvBool("KB3_ENABLED", true),
			},
			KB6Formulary: KBClientConfig{
				BaseURL:         getEnv("KB6_FORMULARY_URL", "http://kb-formulary:8086"),
				Timeout:         getEnvDuration("KB6_TIMEOUT", 10*time.Second),
				MaxRetries:      getEnvInt("KB6_MAX_RETRIES", 3),
				RetryWaitMin:    getEnvDuration("KB6_RETRY_WAIT_MIN", 100*time.Millisecond),
				RetryWaitMax:    getEnvDuration("KB6_RETRY_WAIT_MAX", 2*time.Second),
				MaxIdleConns:    getEnvInt("KB6_MAX_IDLE_CONNS", 10),
				IdleConnTimeout: getEnvDuration("KB6_IDLE_CONN_TIMEOUT", 90*time.Second),
				Enabled:         getEnvBool("KB6_ENABLED", true),
			},
			KB7Terminology: KBClientConfig{
				BaseURL:         getEnv("KB7_TERMINOLOGY_URL", "http://kb-terminology:8087"),
				Timeout:         getEnvDuration("KB7_TIMEOUT", 10*time.Second),
				MaxRetries:      getEnvInt("KB7_MAX_RETRIES", 3),
				RetryWaitMin:    getEnvDuration("KB7_RETRY_WAIT_MIN", 100*time.Millisecond),
				RetryWaitMax:    getEnvDuration("KB7_RETRY_WAIT_MAX", 2*time.Second),
				MaxIdleConns:    getEnvInt("KB7_MAX_IDLE_CONNS", 10),
				IdleConnTimeout: getEnvDuration("KB7_IDLE_CONN_TIMEOUT", 90*time.Second),
				Enabled:         getEnvBool("KB7_ENABLED", true),
			},
		},
		Logging: LoggingConfig{
			Level:      getEnv("LOG_LEVEL", "info"),
			Format:     getEnv("LOG_FORMAT", "json"),
			Output:     getEnv("LOG_OUTPUT", "stdout"),
			TimeFormat: getEnv("LOG_TIME_FORMAT", time.RFC3339),
		},
		HealthCheck: HealthCheckConfig{
			Interval:         getEnvDuration("HEALTH_CHECK_INTERVAL", 30*time.Second),
			Timeout:          getEnvDuration("HEALTH_CHECK_TIMEOUT", 5*time.Second),
			FailureThreshold: getEnvInt("HEALTH_CHECK_FAILURE_THRESHOLD", 3),
		},
	}
}

// GetDSN returns the PostgreSQL connection string
func (c *DatabaseConfig) GetDSN() string {
	if c.URL != "" {
		return c.URL
	}
	return "host=" + c.Host +
		" port=" + strconv.Itoa(c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Database +
		" sslmode=" + c.SSLMode
}

// GetRedisAddr returns the Redis address string
func (c *RedisConfig) GetRedisAddr() string {
	if c.URL != "" {
		return c.URL
	}
	return c.Host + ":" + strconv.Itoa(c.Port)
}

// IsProduction returns true if running in production environment
func (c *ServerConfig) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment
func (c *ServerConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

// SetupLogging configures logrus based on LoggingConfig
func (c *LoggingConfig) SetupLogging() {
	level, err := logrus.ParseLevel(c.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	if c.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: c.TimeFormat,
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: c.TimeFormat,
			FullTimestamp:   true,
		})
	}
}

// Helper functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
