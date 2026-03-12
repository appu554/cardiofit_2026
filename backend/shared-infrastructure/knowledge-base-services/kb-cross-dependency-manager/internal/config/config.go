package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config represents the application configuration
type Config struct {
	Environment          string           `json:"environment"`
	Port                 string           `json:"port"`
	Database             DatabaseConfig   `json:"database"`
	DiscoveryInterval    time.Duration    `json:"discovery_interval"`
	HealthCheckInterval  time.Duration    `json:"health_check_interval"`
	Logging              LoggingConfig    `json:"logging"`
	Security             SecurityConfig   `json:"security"`
	Monitoring           MonitoringConfig `json:"monitoring"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	Name            string        `json:"name"`
	SSLMode         string        `json:"ssl_mode"`
	Timezone        string        `json:"timezone"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	MaxOpenConns    int           `json:"max_open_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	EnableAuth     bool     `json:"enable_auth"`
	JWTSecret      string   `json:"jwt_secret"`
	AllowedOrigins []string `json:"allowed_origins"`
	RateLimit      struct {
		RequestsPerMinute int           `json:"requests_per_minute"`
		BurstSize         int           `json:"burst_size"`
		WindowSize        time.Duration `json:"window_size"`
	} `json:"rate_limit"`
}

// MonitoringConfig holds monitoring and metrics configuration
type MonitoringConfig struct {
	EnableMetrics     bool          `json:"enable_metrics"`
	MetricsPort       string        `json:"metrics_port"`
	HealthCheckPort   string        `json:"health_check_port"`
	EnableTracing     bool          `json:"enable_tracing"`
	TracingEndpoint   string        `json:"tracing_endpoint"`
	AlertingWebhook   string        `json:"alerting_webhook"`
	MetricsRetention  time.Duration `json:"metrics_retention"`
}

// Load loads the configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	config := &Config{
		Environment:         getEnvOrDefault("ENVIRONMENT", "development"),
		Port:                getEnvOrDefault("PORT", "8095"),
		DiscoveryInterval:   getDurationOrDefault("DISCOVERY_INTERVAL", 1*time.Hour),
		HealthCheckInterval: getDurationOrDefault("HEALTH_CHECK_INTERVAL", 30*time.Minute),
	}

	// Database configuration
	config.Database = DatabaseConfig{
		Host:            getEnvOrDefault("DB_HOST", "localhost"),
		Port:            getIntOrDefault("DB_PORT", 5432),
		User:            getEnvOrDefault("DB_USER", "kb_user"),
		Password:        getEnvOrDefault("DB_PASSWORD", "kb_password"),
		Name:            getEnvOrDefault("DB_NAME", "knowledge_bases"),
		SSLMode:         getEnvOrDefault("DB_SSL_MODE", "disable"),
		Timezone:        getEnvOrDefault("DB_TIMEZONE", "UTC"),
		MaxIdleConns:    getIntOrDefault("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getIntOrDefault("DB_MAX_OPEN_CONNS", 100),
		ConnMaxLifetime: getDurationOrDefault("DB_CONN_MAX_LIFETIME", 1*time.Hour),
	}

	// Logging configuration
	config.Logging = LoggingConfig{
		Level:  getEnvOrDefault("LOG_LEVEL", "info"),
		Format: getEnvOrDefault("LOG_FORMAT", "json"),
		Output: getEnvOrDefault("LOG_OUTPUT", "stdout"),
	}

	// Security configuration
	config.Security = SecurityConfig{
		EnableAuth:     getBoolOrDefault("ENABLE_AUTH", false),
		JWTSecret:      getEnvOrDefault("JWT_SECRET", "your-secret-key"),
		AllowedOrigins: getStringSliceOrDefault("ALLOWED_ORIGINS", []string{"*"}),
	}
	config.Security.RateLimit.RequestsPerMinute = getIntOrDefault("RATE_LIMIT_RPM", 1000)
	config.Security.RateLimit.BurstSize = getIntOrDefault("RATE_LIMIT_BURST", 100)
	config.Security.RateLimit.WindowSize = getDurationOrDefault("RATE_LIMIT_WINDOW", 1*time.Minute)

	// Monitoring configuration
	config.Monitoring = MonitoringConfig{
		EnableMetrics:     getBoolOrDefault("ENABLE_METRICS", true),
		MetricsPort:       getEnvOrDefault("METRICS_PORT", "9090"),
		HealthCheckPort:   getEnvOrDefault("HEALTH_CHECK_PORT", "8080"),
		EnableTracing:     getBoolOrDefault("ENABLE_TRACING", false),
		TracingEndpoint:   getEnvOrDefault("TRACING_ENDPOINT", ""),
		AlertingWebhook:   getEnvOrDefault("ALERTING_WEBHOOK", ""),
		MetricsRetention:  getDurationOrDefault("METRICS_RETENTION", 30*24*time.Hour), // 30 days
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// IsProduction returns true if the environment is production
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// GetDatabaseDSN returns the database connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		c.Database.Host,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.Port,
		c.Database.SSLMode,
		c.Database.Timezone,
	)
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getStringSliceOrDefault(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple implementation - could be enhanced to parse comma-separated values
		return []string{value}
	}
	return defaultValue
}

func validateConfig(config *Config) error {
	// Validate required fields
	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if config.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if config.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if config.Port == "" {
		return fmt.Errorf("port is required")
	}

	// Validate port ranges
	if config.Database.Port < 1 || config.Database.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", config.Database.Port)
	}

	// Validate intervals
	if config.DiscoveryInterval < 1*time.Minute {
		return fmt.Errorf("discovery interval must be at least 1 minute")
	}
	if config.HealthCheckInterval < 30*time.Second {
		return fmt.Errorf("health check interval must be at least 30 seconds")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true, "fatal": true,
	}
	if !validLogLevels[config.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", config.Logging.Level)
	}

	// Validate environment
	validEnvironments := map[string]bool{
		"development": true, "staging": true, "production": true,
	}
	if !validEnvironments[config.Environment] {
		return fmt.Errorf("invalid environment: %s", config.Environment)
	}

	return nil
}

// Default environment-specific configurations

// GetDevelopmentDefaults returns default configuration for development environment
func GetDevelopmentDefaults() *Config {
	config, _ := Load()
	config.Environment = "development"
	config.Database.Host = "localhost"
	config.Database.Port = 5432
	config.Database.SSLMode = "disable"
	config.Logging.Level = "debug"
	config.Security.EnableAuth = false
	config.DiscoveryInterval = 30 * time.Minute      // More frequent discovery in dev
	config.HealthCheckInterval = 10 * time.Minute    // More frequent health checks in dev
	return config
}

// GetProductionDefaults returns default configuration for production environment
func GetProductionDefaults() *Config {
	config, _ := Load()
	config.Environment = "production"
	config.Database.SSLMode = "require"
	config.Logging.Level = "info"
	config.Security.EnableAuth = true
	config.DiscoveryInterval = 2 * time.Hour        // Less frequent in production
	config.HealthCheckInterval = 1 * time.Hour      // Less frequent in production
	config.Monitoring.EnableMetrics = true
	config.Monitoring.EnableTracing = true
	return config
}

// ConfigurableDefaults allows override of specific configuration values
type ConfigurableDefaults struct {
	Port                string
	DatabaseHost        string
	DatabasePort        int
	DatabaseName        string
	DiscoveryInterval   time.Duration
	HealthCheckInterval time.Duration
	LogLevel            string
}

// LoadWithDefaults loads configuration with custom defaults
func LoadWithDefaults(defaults ConfigurableDefaults) (*Config, error) {
	config, err := Load()
	if err != nil {
		return nil, err
	}

	// Apply custom defaults if not set via environment
	if os.Getenv("PORT") == "" && defaults.Port != "" {
		config.Port = defaults.Port
	}
	if os.Getenv("DB_HOST") == "" && defaults.DatabaseHost != "" {
		config.Database.Host = defaults.DatabaseHost
	}
	if os.Getenv("DB_PORT") == "" && defaults.DatabasePort != 0 {
		config.Database.Port = defaults.DatabasePort
	}
	if os.Getenv("DB_NAME") == "" && defaults.DatabaseName != "" {
		config.Database.Name = defaults.DatabaseName
	}
	if os.Getenv("DISCOVERY_INTERVAL") == "" && defaults.DiscoveryInterval != 0 {
		config.DiscoveryInterval = defaults.DiscoveryInterval
	}
	if os.Getenv("HEALTH_CHECK_INTERVAL") == "" && defaults.HealthCheckInterval != 0 {
		config.HealthCheckInterval = defaults.HealthCheckInterval
	}
	if os.Getenv("LOG_LEVEL") == "" && defaults.LogLevel != "" {
		config.Logging.Level = defaults.LogLevel
	}

	return config, validateConfig(config)
}