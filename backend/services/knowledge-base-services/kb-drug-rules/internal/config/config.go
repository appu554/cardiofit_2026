package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the KB-Drug-Rules service
type Config struct {
	// Server configuration
	Port     int  `mapstructure:"port"`
	GRPCPort int  `mapstructure:"grpc_port"`
	Debug    bool `mapstructure:"debug"`

	// KB-1 specific configuration
	SchemaPath string `mapstructure:"schema_path"`

	// Database configuration
	DatabaseURL string `mapstructure:"database_url"`

	// Supabase configuration
	SupabaseURL    string `mapstructure:"supabase_url"`
	SupabaseAPIKey string `mapstructure:"supabase_api_key"`
	SupabaseJWT    string `mapstructure:"supabase_jwt_secret"`

	// Redis configuration
	RedisURL string `mapstructure:"redis_url"`

	// S3/MinIO configuration
	S3Endpoint   string `mapstructure:"s3_endpoint"`
	S3Bucket     string `mapstructure:"s3_bucket"`
	S3AccessKey  string `mapstructure:"s3_access_key"`
	S3SecretKey  string `mapstructure:"s3_secret_key"`

	// Kafka configuration
	KafkaBrokers []string `mapstructure:"kafka_brokers"`
	KafkaTopic   string   `mapstructure:"kafka_topic"`

	// Security configuration
	SigningKeyPath string `mapstructure:"signing_key_path"`
	JWTSecret      string `mapstructure:"jwt_secret"`

	// Cache configuration
	CacheTTL         int `mapstructure:"cache_ttl"`
	CacheMaxSize     int `mapstructure:"cache_max_size"`
	CacheEvictionTTL int `mapstructure:"cache_eviction_ttl"`

	// Governance configuration
	RequireApproval      bool `mapstructure:"require_approval"`
	RequireSignature     bool `mapstructure:"require_signature"`
	AllowedReviewers     []string `mapstructure:"allowed_reviewers"`
	AllowedSigners       []string `mapstructure:"allowed_signers"`

	// Regional configuration
	DefaultRegion     string   `mapstructure:"default_region"`
	SupportedRegions  []string `mapstructure:"supported_regions"`

	// Monitoring configuration
	MetricsEnabled    bool   `mapstructure:"metrics_enabled"`
	MetricsPath       string `mapstructure:"metrics_path"`
	TracingEnabled    bool   `mapstructure:"tracing_enabled"`
	TracingEndpoint   string `mapstructure:"tracing_endpoint"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	// Set defaults
	viper.SetDefault("port", 8081)
	viper.SetDefault("grpc_port", 9081)
	viper.SetDefault("debug", false)
	viper.SetDefault("schema_path", "./config/schema")
	viper.SetDefault("database_url", "postgresql://kb_drug_rules_user:kb_password@localhost:5432/kb_drug_rules")
	viper.SetDefault("redis_url", "redis://localhost:6379/0")
	viper.SetDefault("cache_ttl", 3600)
	viper.SetDefault("cache_max_size", 10000)
	viper.SetDefault("cache_eviction_ttl", 300)
	viper.SetDefault("require_approval", false)
	viper.SetDefault("require_signature", false)
	viper.SetDefault("default_region", "US")
	viper.SetDefault("supported_regions", []string{"US", "EU", "CA", "AU"})
	viper.SetDefault("metrics_enabled", true)
	viper.SetDefault("metrics_path", "/metrics")
	viper.SetDefault("tracing_enabled", false)
	viper.SetDefault("kafka_topic", "kb-events")

	// Read from environment variables
	viper.AutomaticEnv()

	// Read from config file if it exists
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/kb-drug-rules")

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Override with environment variables
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			viper.Set("port", p)
		}
	}

	if debug := os.Getenv("DEBUG"); debug != "" {
		if d, err := strconv.ParseBool(debug); err == nil {
			viper.Set("debug", d)
		}
	}

	// Required environment variables (either DATABASE_URL or Supabase config)
	databaseURL := os.Getenv("DATABASE_URL")
	supabaseURL := os.Getenv("SUPABASE_URL")

	if databaseURL != "" {
		viper.Set("database_url", databaseURL)
	} else if supabaseURL != "" {
		// Use Supabase configuration
		viper.Set("supabase_url", supabaseURL)
		if apiKey := os.Getenv("SUPABASE_API_KEY"); apiKey != "" {
			viper.Set("supabase_api_key", apiKey)
		}
		if jwtSecret := os.Getenv("SUPABASE_JWT_SECRET"); jwtSecret != "" {
			viper.Set("supabase_jwt_secret", jwtSecret)
		}

		// Build database URL from Supabase config
		if dbURL := buildSupabaseDatabaseURL(supabaseURL); dbURL != "" {
			viper.Set("database_url", dbURL)
		}
	} else {
		return nil, fmt.Errorf("either DATABASE_URL or SUPABASE_URL must be set")
	}

	// Redis is required
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		viper.Set("redis_url", redisURL)
	} else {
		return nil, fmt.Errorf("REDIS_URL is required")
	}

	// Optional environment variables
	optionalEnvVars := map[string]string{
		"S3_ENDPOINT":      "s3_endpoint",
		"S3_BUCKET":        "s3_bucket",
		"S3_ACCESS_KEY":    "s3_access_key",
		"S3_SECRET_KEY":    "s3_secret_key",
		"KAFKA_BROKERS":    "kafka_brokers",
		"SIGNING_KEY_PATH": "signing_key_path",
		"JWT_SECRET":       "jwt_secret",
		"TRACING_ENDPOINT": "tracing_endpoint",
	}

	for envVar, configKey := range optionalEnvVars {
		if value := os.Getenv(envVar); value != "" {
			viper.Set(configKey, value)
		}
	}

	// Parse Kafka brokers from comma-separated string
	if kafkaBrokers := os.Getenv("KAFKA_BROKERS"); kafkaBrokers != "" {
		viper.Set("kafka_brokers", parseCommaSeparated(kafkaBrokers))
	}

	// Parse supported regions from comma-separated string
	if regions := os.Getenv("SUPPORTED_REGIONS"); regions != "" {
		viper.Set("supported_regions", parseCommaSeparated(regions))
	}

	// Unmarshal into config struct
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}

	var result []string
	for _, item := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	if config.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}

	if config.RedisURL == "" {
		return fmt.Errorf("redis_url is required")
	}

	if config.CacheTTL <= 0 {
		return fmt.Errorf("cache_ttl must be positive")
	}

	if config.CacheMaxSize <= 0 {
		return fmt.Errorf("cache_max_size must be positive")
	}

	if len(config.SupportedRegions) == 0 {
		return fmt.Errorf("supported_regions cannot be empty")
	}

	// Validate default region is in supported regions
	defaultRegionValid := false
	for _, region := range config.SupportedRegions {
		if region == config.DefaultRegion {
			defaultRegionValid = true
			break
		}
	}
	if !defaultRegionValid {
		return fmt.Errorf("default_region %s is not in supported_regions", config.DefaultRegion)
	}

	return nil
}

// GetEnv gets an environment variable with a default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt gets an integer environment variable with a default value
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool gets a boolean environment variable with a default value
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// buildSupabaseDatabaseURL builds a database URL from Supabase URL
func buildSupabaseDatabaseURL(supabaseURL string) string {
	// Extract project reference from Supabase URL
	// Example: https://abcdefghijklmnop.supabase.co -> abcdefghijklmnop
	if !strings.HasPrefix(supabaseURL, "https://") {
		return ""
	}

	urlPart := strings.TrimPrefix(supabaseURL, "https://")
	if !strings.HasSuffix(urlPart, ".supabase.co") {
		return ""
	}

	projectRef := strings.TrimSuffix(urlPart, ".supabase.co")

	// Get database password from environment
	dbPassword := os.Getenv("SUPABASE_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = os.Getenv("DATABASE_PASSWORD")
	}

	if dbPassword == "" {
		return ""
	}

	// Build PostgreSQL connection string for Supabase
	return fmt.Sprintf("postgresql://postgres:%s@db.%s.supabase.co:5432/postgres?sslmode=require",
		dbPassword, projectRef)
}
