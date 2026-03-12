// Package config provides configuration management for the Clinical Rules Engine
package config

import (
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Rules     RulesConfig     `mapstructure:"rules"`
	Vaidshala VaidshalaConfig `mapstructure:"vaidshala"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port            int           `mapstructure:"port"`
	Host            string        `mapstructure:"host"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	MaxRequestSize  int64         `mapstructure:"max_request_size"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Name            string        `mapstructure:"name"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConnections  int           `mapstructure:"max_connections"`
	MinConnections  int           `mapstructure:"min_connections"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// RulesConfig holds rules engine configuration
type RulesConfig struct {
	Path          string        `mapstructure:"path"`
	EnableCaching bool          `mapstructure:"enable_caching"`
	CacheTTL      time.Duration `mapstructure:"cache_ttl"`
	WatchInterval time.Duration `mapstructure:"watch_interval"`
	WatchEnabled  bool          `mapstructure:"watch_enabled"`
	ValidateOnLoad bool         `mapstructure:"validate_on_load"`
}

// VaidshalaConfig holds CQL engine configuration
type VaidshalaConfig struct {
	URL            string        `mapstructure:"url"`
	Enabled        bool          `mapstructure:"enabled"`
	Timeout        time.Duration `mapstructure:"timeout"`
	RetryAttempts  int           `mapstructure:"retry_attempts"`
	RetryDelay     time.Duration `mapstructure:"retry_delay"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	TimeFormat string `mapstructure:"time_format"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	Path       string `mapstructure:"path"`
	Namespace  string `mapstructure:"namespace"`
	Subsystem  string `mapstructure:"subsystem"`
}

// Load loads configuration from environment variables and defaults
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Enable reading from environment variables
	v.SetEnvPrefix("KB10")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Also check without prefix for common vars
	bindEnvVars(v)

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.port", 8100)
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.read_timeout", 30*time.Second)
	v.SetDefault("server.write_timeout", 30*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)
	v.SetDefault("server.max_request_size", 10*1024*1024) // 10MB

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5433)
	v.SetDefault("database.name", "kb10_rules")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "password")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_connections", 25)
	v.SetDefault("database.min_connections", 5)
	v.SetDefault("database.max_conn_lifetime", time.Hour)
	v.SetDefault("database.max_conn_idle_time", 30*time.Minute)

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6380)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.dial_timeout", 5*time.Second)
	v.SetDefault("redis.read_timeout", 3*time.Second)
	v.SetDefault("redis.write_timeout", 3*time.Second)

	// Rules defaults
	v.SetDefault("rules.path", "./rules")
	v.SetDefault("rules.enable_caching", true)
	v.SetDefault("rules.cache_ttl", 5*time.Minute)
	v.SetDefault("rules.watch_interval", 30*time.Second)
	v.SetDefault("rules.watch_enabled", false)
	v.SetDefault("rules.validate_on_load", true)

	// Vaidshala CQL defaults
	v.SetDefault("vaidshala.url", "http://localhost:8096")
	v.SetDefault("vaidshala.enabled", false)
	v.SetDefault("vaidshala.timeout", 30*time.Second)
	v.SetDefault("vaidshala.retry_attempts", 3)
	v.SetDefault("vaidshala.retry_delay", time.Second)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
	v.SetDefault("logging.time_format", time.RFC3339)

	// Metrics defaults
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("metrics.namespace", "kb10")
	v.SetDefault("metrics.subsystem", "rules_engine")
}

// bindEnvVars binds specific environment variables
func bindEnvVars(v *viper.Viper) {
	// Bind common environment variables with and without prefix
	envMappings := map[string]string{
		"server.port":        "KB10_PORT",
		"database.host":      "KB10_DB_HOST",
		"database.port":      "KB10_DB_PORT",
		"database.name":      "KB10_DB_NAME",
		"database.user":      "KB10_DB_USER",
		"database.password":  "KB10_DB_PASSWORD",
		"redis.host":         "KB10_REDIS_HOST",
		"redis.port":         "KB10_REDIS_PORT",
		"redis.password":     "KB10_REDIS_PASSWORD",
		"rules.path":         "KB10_RULES_PATH",
		"rules.enable_caching": "KB10_ENABLE_CACHING",
		"rules.cache_ttl":    "KB10_CACHE_TTL",
		"vaidshala.url":      "VAIDSHALA_URL",
		"logging.level":      "KB10_LOG_LEVEL",
	}

	for key, env := range envMappings {
		_ = v.BindEnv(key, env)
	}
}

// SetupLogger configures the global logger based on config
func SetupLogger(cfg *LoggingConfig) *logrus.Logger {
	logger := logrus.New()

	// Set level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set format
	switch cfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: cfg.TimeFormat,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: cfg.TimeFormat,
		})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: cfg.TimeFormat,
		})
	}

	return logger
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + string(rune(c.Port)) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Name +
		" sslmode=" + c.SSLMode
}

// ConnectionString returns the PostgreSQL connection URL
func (c *DatabaseConfig) ConnectionString() string {
	return "postgres://" + c.User + ":" + c.Password +
		"@" + c.Host + ":" + intToString(c.Port) +
		"/" + c.Name + "?sslmode=" + c.SSLMode
}

// RedisAddr returns the Redis address string
func (c *RedisConfig) Addr() string {
	return c.Host + ":" + intToString(c.Port)
}

func intToString(i int) string {
	return strconv.Itoa(i)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return &ConfigError{Field: "server.port", Message: "invalid port number"}
	}
	if c.Database.Host == "" {
		return &ConfigError{Field: "database.host", Message: "database host is required"}
	}
	if c.Rules.Path == "" {
		return &ConfigError{Field: "rules.path", Message: "rules path is required"}
	}
	return nil
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + ": " + e.Message
}
