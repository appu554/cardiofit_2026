package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the KB-3 service
type Config struct {
	// Server configuration
	Port         string
	Environment  string
	Debug        bool
	ReadTimeout  int
	WriteTimeout int

	// Database configuration
	DatabaseURL      string
	MaxOpenConns     int
	MaxIdleConns     int
	ConnMaxLifetime  int

	// Redis configuration
	RedisURL         string
	RedisPassword    string
	RedisDB          int
	CacheTTL         int

	// KB-3 specific configuration
	KBVersion        string
	DefaultRegion    string
	SupportedRegions []string
	DataPath         string

	// Cross-KB service URLs
	KB1URL           string
	KB2URL           string  
	KB4URL           string
	KB5URL           string
	KB6URL           string
	KB7URL           string

	// Security and governance
	RequireApproval  bool
	RequireSignature bool
	SigningKeyPath   string
	JWTSecret        string

	// Monitoring
	MetricsEnabled   bool
	TracingEnabled   bool
	LogLevel         string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	config := &Config{
		// Server defaults
		Port:         getEnv("PORT", "8083"),
		Environment:  getEnv("ENVIRONMENT", "development"),
		Debug:        getEnvAsBool("DEBUG", true),
		ReadTimeout:  getEnvAsInt("READ_TIMEOUT", 30),
		WriteTimeout: getEnvAsInt("WRITE_TIMEOUT", 30),

		// Database defaults
		DatabaseURL:     getEnv("DATABASE_URL", "postgresql://kb_guideline_evidence_user:kb_password@localhost:5433/kb_guideline_evidence"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 300),

		// Redis defaults
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6380/3"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 3),
		CacheTTL:      getEnvAsInt("CACHE_TTL", 3600), // 1 hour

		// KB-3 defaults
		KBVersion:        getEnv("KB_VERSION", "3.0.0"),
		DefaultRegion:    getEnv("DEFAULT_REGION", "US"),
		SupportedRegions: getEnvAsSlice("SUPPORTED_REGIONS", []string{"US", "EU", "AU", "WHO"}),
		DataPath:         getEnv("DATA_PATH", "./data"),

		// Cross-KB service URLs
		KB1URL: getEnv("KB1_URL", "http://localhost:8081"),
		KB2URL: getEnv("KB2_URL", "http://localhost:8082"), 
		KB4URL: getEnv("KB4_URL", "http://localhost:8084"),
		KB5URL: getEnv("KB5_URL", "http://localhost:8085"),
		KB6URL: getEnv("KB6_URL", "http://localhost:8086"),
		KB7URL: getEnv("KB7_URL", "http://localhost:8087"),

		// Security defaults
		RequireApproval:  getEnvAsBool("REQUIRE_APPROVAL", false),
		RequireSignature: getEnvAsBool("REQUIRE_SIGNATURE", false),
		SigningKeyPath:   getEnv("SIGNING_KEY_PATH", "./keys/signing.key"),
		JWTSecret:        getEnv("JWT_SECRET", "kb3-jwt-secret-key-for-development"),

		// Monitoring defaults
		MetricsEnabled: getEnvAsBool("METRICS_ENABLED", true),
		TracingEnabled: getEnvAsBool("TRACING_ENABLED", false),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
	}

	// Validate configuration
	if err := config.validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	return config
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if c.Port == "" {
		return fmt.Errorf("PORT cannot be empty")
	}
	
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL cannot be empty")
	}

	if c.KBVersion == "" {
		return fmt.Errorf("KB_VERSION cannot be empty")
	}

	// Validate default region is in supported regions
	found := false
	for _, region := range c.SupportedRegions {
		if region == c.DefaultRegion {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("DEFAULT_REGION %s not found in SUPPORTED_REGIONS %v", 
			c.DefaultRegion, c.SupportedRegions)
	}

	return nil
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}

// GetLogLevel returns the log level for structured logging
func (c *Config) GetLogLevel() string {
	if c.Debug {
		return "debug"
	}
	return c.LogLevel
}

// GetServerAddress returns the complete server address
func (c *Config) GetServerAddress() string {
	return ":" + c.Port
}

// GetCorsOrigins returns allowed CORS origins based on environment
func (c *Config) GetCorsOrigins() []string {
	if c.IsProduction() {
		// In production, specify exact origins
		return []string{
			"https://clinical-hub.health",
			"https://app.clinical-hub.health",
		}
	}
	// In development, allow all origins
	return []string{"*"}
}

// Utility functions for environment variable parsing

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
		log.Printf("Warning: Invalid boolean value for %s: %s, using default: %t", key, value, defaultValue)
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}