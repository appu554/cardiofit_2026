package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func init() {
	loadDotEnv(".env")
}

// loadDotEnv reads a .env file and sets vars not already in the environment.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}

// Config holds all configuration for the Ingestion Service.
type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	FHIR        fhirclient.GoogleFHIRConfig
	Kafka       KafkaConfig
	Outbox      OutboxConfig
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

// OutboxConfig holds Global Outbox SDK settings for atomic event publishing.
type OutboxConfig struct {
	Enabled         bool
	DatabaseURL     string
	GRPCAddress     string
	DefaultPriority int32
}

// IsDevelopment returns true when running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "dev"
}

// GetLabAPIKey returns the API key for a specific lab provider.
// Keys are stored as LAB_API_KEY_{UPPER_LAB_ID} env vars.
func (c *Config) GetLabAPIKey(labID string) string {
	return getEnv("LAB_API_KEY_"+strings.ToUpper(labID), "")
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
			ProjectID:       getEnv("FHIR_PROJECT_ID", "project-2bbef9ac-174b-4b59-8fe"),
			Location:        getEnv("FHIR_LOCATION", "asia-south1"),
			DatasetID:       getEnv("FHIR_DATASET_ID", "vaidshala-clinical"),
			FhirStoreID:     getEnv("FHIR_STORE_ID", "cardiofit-fhir-r4"),
			CredentialsPath: getEnv("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json"),
		},
		Kafka: KafkaConfig{
			Brokers: []string{getEnv("KAFKA_BROKERS", "localhost:9092")},
			GroupID: getEnv("KAFKA_GROUP_ID", "ingestion-service"),
		},
		Outbox: OutboxConfig{
			Enabled:         getEnvAsBool("OUTBOX_ENABLED", false),
			DatabaseURL:     getEnv("OUTBOX_DATABASE_URL", ""),
			GRPCAddress:     getEnv("OUTBOX_GRPC_ADDRESS", "localhost:50052"),
			DefaultPriority: int32(getEnvAsInt("OUTBOX_DEFAULT_PRIORITY", 5)),
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
