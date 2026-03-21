package config

import (
	"os"
	"strconv"
	"time"

	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// Config holds all configuration for the Ingestion Service.
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	FHIR        fhirclient.GoogleFHIRConfig
	Kafka       KafkaConfig
	Environment string
	LogLevel    string
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	URL             string
	MaxConnections  int32
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// KafkaConfig holds Kafka connection settings.
type KafkaConfig struct {
	Brokers []string
	GroupID string
}

// IsDevelopment returns true when running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "dev"
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8140"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://ingestion_user:ingestion_password@localhost:5433/ingestion_service?sslmode=disable"),
			MaxConnections:  int32(getEnvAsInt("DATABASE_MAX_CONNECTIONS", 10)),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DATABASE_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6380"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 2),
		},
		FHIR: fhirclient.GoogleFHIRConfig{
			Enabled:         getEnvAsBool("FHIR_ENABLED", false),
			ProjectID:       getEnv("FHIR_PROJECT_ID", ""),
			Location:        getEnv("FHIR_LOCATION", ""),
			DatasetID:       getEnv("FHIR_DATASET_ID", ""),
			FhirStoreID:     getEnv("FHIR_STORE_ID", ""),
			CredentialsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "ingestion-service"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv reads an environment variable or returns the fallback value.
func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// getEnvAsInt reads an environment variable as an integer or returns the fallback.
func getEnvAsInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

// getEnvAsBool reads an environment variable as a boolean or returns the fallback.
func getEnvAsBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
