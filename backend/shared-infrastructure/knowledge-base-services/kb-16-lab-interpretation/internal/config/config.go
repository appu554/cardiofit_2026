// Package config provides configuration management for KB-16
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Integration IntegrationConfig
	Governance  GovernanceConfig
	Metrics     MetricsConfig
	Logging     LoggingConfig
}

// GovernanceConfig holds Tier-7 governance event configuration
type GovernanceConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	CriticalChannel  string `mapstructure:"critical_channel"`
	StandardChannel  string `mapstructure:"standard_channel"`
	AuditChannel     string `mapstructure:"audit_channel"`
	AuditEnabled     bool   `mapstructure:"audit_enabled"`
	AsyncPublish     bool   `mapstructure:"async_publish"`
	BufferSize       int    `mapstructure:"buffer_size"`
	PanicAckSLAMin   int    `mapstructure:"panic_ack_sla_min"`
	CriticalAckSLAMin int   `mapstructure:"critical_ack_sla_min"`
	HighAckSLAMin    int    `mapstructure:"high_ack_sla_min"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Environment     string        `mapstructure:"environment"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL          string        `mapstructure:"url"`
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	TTL          time.Duration `mapstructure:"ttl"`
	Enabled      bool          `mapstructure:"enabled"`
}

// IntegrationConfig holds configuration for KB service integrations
type IntegrationConfig struct {
	KB2URL     string        `mapstructure:"kb2_url"`
	KB8URL     string        `mapstructure:"kb8_url"`
	KB8Enabled bool          `mapstructure:"kb8_enabled"`
	KB9URL     string        `mapstructure:"kb9_url"`
	KB9Enabled bool          `mapstructure:"kb9_enabled"`
	KB14URL    string        `mapstructure:"kb14_url"`
	Timeout    time.Duration `mapstructure:"timeout"`
	RetryCount int           `mapstructure:"retry_count"`
}

// MetricsConfig holds Prometheus metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"` // json, text
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Read from environment
	v.AutomaticEnv()

	// Build config
	cfg := &Config{
		Server: ServerConfig{
			Port:            v.GetInt("PORT"),
			Environment:     v.GetString("ENVIRONMENT"),
			ReadTimeout:     v.GetDuration("SERVER_READ_TIMEOUT"),
			WriteTimeout:    v.GetDuration("SERVER_WRITE_TIMEOUT"),
			ShutdownTimeout: v.GetDuration("SERVER_SHUTDOWN_TIMEOUT"),
		},
		Database: DatabaseConfig{
			URL:             v.GetString("DATABASE_URL"),
			Host:            v.GetString("DB_HOST"),
			Port:            v.GetInt("DB_PORT"),
			User:            v.GetString("DB_USER"),
			Password:        v.GetString("DB_PASSWORD"),
			Database:        v.GetString("DB_NAME"),
			SSLMode:         v.GetString("DB_SSL_MODE"),
			MaxOpenConns:    v.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: v.GetDuration("DB_CONN_MAX_LIFETIME"),
		},
		Redis: RedisConfig{
			URL:          v.GetString("REDIS_URL"),
			Host:         v.GetString("REDIS_HOST"),
			Port:         v.GetInt("REDIS_PORT"),
			Password:     v.GetString("REDIS_PASSWORD"),
			DB:           v.GetInt("REDIS_DB"),
			PoolSize:     v.GetInt("REDIS_POOL_SIZE"),
			MinIdleConns: v.GetInt("REDIS_MIN_IDLE_CONNS"),
			TTL:          v.GetDuration("REDIS_TTL"),
			Enabled:      v.GetBool("REDIS_ENABLED"),
		},
		Integration: IntegrationConfig{
			KB2URL:     v.GetString("KB2_SERVICE_URL"),
			KB8URL:     v.GetString("KB8_SERVICE_URL"),
			KB8Enabled: v.GetBool("KB8_ENABLED"),
			KB9URL:     v.GetString("KB9_SERVICE_URL"),
			KB9Enabled: v.GetBool("KB9_ENABLED"),
			KB14URL:    v.GetString("KB14_SERVICE_URL"),
			Timeout:    v.GetDuration("INTEGRATION_TIMEOUT"),
			RetryCount: v.GetInt("INTEGRATION_RETRY_COUNT"),
		},
		Governance: GovernanceConfig{
			Enabled:          v.GetBool("GOVERNANCE_ENABLED"),
			CriticalChannel:  v.GetString("GOVERNANCE_CRITICAL_CHANNEL"),
			StandardChannel:  v.GetString("GOVERNANCE_STANDARD_CHANNEL"),
			AuditChannel:     v.GetString("GOVERNANCE_AUDIT_CHANNEL"),
			AuditEnabled:     v.GetBool("GOVERNANCE_AUDIT_ENABLED"),
			AsyncPublish:     v.GetBool("GOVERNANCE_ASYNC_PUBLISH"),
			BufferSize:       v.GetInt("GOVERNANCE_BUFFER_SIZE"),
			PanicAckSLAMin:   v.GetInt("GOVERNANCE_PANIC_ACK_SLA_MIN"),
			CriticalAckSLAMin: v.GetInt("GOVERNANCE_CRITICAL_ACK_SLA_MIN"),
			HighAckSLAMin:    v.GetInt("GOVERNANCE_HIGH_ACK_SLA_MIN"),
		},
		Metrics: MetricsConfig{
			Enabled: v.GetBool("METRICS_ENABLED"),
			Path:    v.GetString("METRICS_PATH"),
		},
		Logging: LoggingConfig{
			Level:  v.GetString("LOG_LEVEL"),
			Format: v.GetString("LOG_FORMAT"),
		},
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("PORT", 8095)
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("SERVER_READ_TIMEOUT", 30*time.Second)
	v.SetDefault("SERVER_WRITE_TIMEOUT", 30*time.Second)
	v.SetDefault("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second)

	// Database defaults - Connect to shared canonical_facts DB (docker-compose.phase1.yml)
	// This contains the loinc_reference_ranges table with 6041 LOINC codes
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5433)
	v.SetDefault("DB_USER", "kb_admin")
	v.SetDefault("DB_PASSWORD", "kb_secure_password_2024")
	v.SetDefault("DB_NAME", "canonical_facts")
	v.SetDefault("DB_SSL_MODE", "disable")
	v.SetDefault("DB_MAX_OPEN_CONNS", 25)
	v.SetDefault("DB_MAX_IDLE_CONNS", 5)
	v.SetDefault("DB_CONN_MAX_LIFETIME", 5*time.Minute)

	// Redis defaults
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6380)
	v.SetDefault("REDIS_PASSWORD", "")
	v.SetDefault("REDIS_DB", 6)
	v.SetDefault("REDIS_POOL_SIZE", 10)
	v.SetDefault("REDIS_MIN_IDLE_CONNS", 2)
	v.SetDefault("REDIS_TTL", 30*time.Minute)
	v.SetDefault("REDIS_ENABLED", true)

	// Integration defaults
	v.SetDefault("KB2_SERVICE_URL", "http://localhost:8086")
	v.SetDefault("KB8_SERVICE_URL", "http://localhost:8080")
	v.SetDefault("KB8_ENABLED", true)
	v.SetDefault("KB9_SERVICE_URL", "http://localhost:8094")
	v.SetDefault("KB9_ENABLED", true)
	v.SetDefault("KB14_SERVICE_URL", "http://localhost:8093")
	v.SetDefault("INTEGRATION_TIMEOUT", 30*time.Second)
	v.SetDefault("INTEGRATION_RETRY_COUNT", 3)

	// Governance defaults (Tier-7 clinical accountability)
	v.SetDefault("GOVERNANCE_ENABLED", true)
	v.SetDefault("GOVERNANCE_CRITICAL_CHANNEL", "kb16:governance:critical")
	v.SetDefault("GOVERNANCE_STANDARD_CHANNEL", "kb16:governance:events")
	v.SetDefault("GOVERNANCE_AUDIT_CHANNEL", "kb16:governance:audit")
	v.SetDefault("GOVERNANCE_AUDIT_ENABLED", true)
	v.SetDefault("GOVERNANCE_ASYNC_PUBLISH", true)
	v.SetDefault("GOVERNANCE_BUFFER_SIZE", 100)
	v.SetDefault("GOVERNANCE_PANIC_ACK_SLA_MIN", 15)
	v.SetDefault("GOVERNANCE_CRITICAL_ACK_SLA_MIN", 30)
	v.SetDefault("GOVERNANCE_HIGH_ACK_SLA_MIN", 60)

	// Metrics defaults
	v.SetDefault("METRICS_ENABLED", true)
	v.SetDefault("METRICS_PATH", "/metrics")

	// Logging defaults
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	// If DATABASE_URL is not set, ensure we have individual components
	if c.Database.URL == "" {
		if c.Database.Host == "" {
			return fmt.Errorf("database host is required")
		}
		if c.Database.Database == "" {
			return fmt.Errorf("database name is required")
		}
	}

	return nil
}

// GetDatabaseURL returns the database connection URL
func (c *Config) GetDatabaseURL() string {
	if c.Database.URL != "" {
		return c.Database.URL
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.SSLMode,
	)
}

// GetRedisURL returns the Redis connection URL
func (c *Config) GetRedisURL() string {
	if c.Redis.URL != "" {
		return c.Redis.URL
	}

	if c.Redis.Password != "" {
		return fmt.Sprintf("redis://:%s@%s:%d/%d",
			c.Redis.Password,
			c.Redis.Host,
			c.Redis.Port,
			c.Redis.DB,
		)
	}

	return fmt.Sprintf("redis://%s:%d/%d",
		c.Redis.Host,
		c.Redis.Port,
		c.Redis.DB,
	)
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}
