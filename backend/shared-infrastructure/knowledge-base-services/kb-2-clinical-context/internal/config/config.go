package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server    ServerConfig
	MongoDB   MongoDBConfig
	Redis     RedisConfig
	Cache     CacheConfig
	Metrics   MetricsConfig
}

type ServerConfig struct {
	Port string
	Environment string
	Debug bool
}

type MongoDBConfig struct {
	URI           string
	Database      string
	Username      string
	Password      string
	Timeout       int
	MaxPoolSize   int
	MinPoolSize   int
	ApplicationName string
}

type RedisConfig struct {
	Address  string
	Password string
	Database int
	Timeout  int
}

type CacheConfig struct {
	L1MaxSize   int
	L1TTL       time.Duration
	CDNBaseURL  string
	CDNEnabled  bool
}

type MetricsConfig struct {
	Enabled bool
	Path    string
}

func LoadConfig() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port:        getEnvWithDefault("PORT", "8082"),
			Environment: getEnvWithDefault("ENVIRONMENT", "development"),
			Debug:       getEnvWithDefault("DEBUG", "true") == "true",
		},
		MongoDB: MongoDBConfig{
			URI:           getEnvWithDefault("MONGODB_URI", "mongodb://localhost:27017"),
			Database:      getEnvWithDefault("MONGODB_DATABASE", "clinical_context"),
			Username:      os.Getenv("MONGODB_USERNAME"),
			Password:      os.Getenv("MONGODB_PASSWORD"),
			Timeout:       30,
			MaxPoolSize:   50,
			MinPoolSize:   5,
			ApplicationName: "kb-2-clinical-context",
		},
		Redis: RedisConfig{
			Address:  getEnvWithDefault("REDIS_URL", "localhost:6380"),
			Password: os.Getenv("REDIS_PASSWORD"),
			Database: 2, // Use DB 2 for KB-2
			Timeout:  5,
		},
		Cache: CacheConfig{
			L1MaxSize:  getEnvAsInt("L1_CACHE_MAX_SIZE", 10000),
			L1TTL:      getEnvAsDuration("L1_CACHE_TTL", "5m"),
			CDNBaseURL: getEnvWithDefault("CDN_BASE_URL", "https://cdn.clinicalknowledge.com"),
			CDNEnabled: getEnvWithDefault("CDN_ENABLED", "true") == "true",
		},
		Metrics: MetricsConfig{
			Enabled: getEnvWithDefault("METRICS_ENABLED", "true") == "true",
			Path:    getEnvWithDefault("METRICS_PATH", "/metrics"),
		},
	}, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	value := getEnvWithDefault(key, defaultValue)
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}
	return 5 * time.Minute
}

func init() {
	// Load .env file if it exists (for development)
	if _, err := os.Stat(".env"); err == nil {
		// Could load .env file here if needed
		log.Println("Loading environment variables from .env file")
	}
}