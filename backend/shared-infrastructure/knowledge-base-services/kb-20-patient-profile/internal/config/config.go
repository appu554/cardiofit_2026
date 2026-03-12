package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all service configuration loaded from environment variables.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	FHIR     GoogleFHIRConfig
	KB7      KB7Config

	Environment string
	LogLevel    string
}

// GoogleFHIRConfig configures the Google Cloud Healthcare FHIR Store integration.
type GoogleFHIRConfig struct {
	Enabled         bool   // FHIR_SYNC_ENABLED — enable FHIR→KB-20 sync
	ProjectID       string // GOOGLE_CLOUD_PROJECT_ID
	Location        string // GOOGLE_CLOUD_LOCATION
	DatasetID       string // GOOGLE_CLOUD_DATASET_ID
	FhirStoreID     string // GOOGLE_CLOUD_FHIR_STORE_ID
	CredentialsPath string // GOOGLE_CLOUD_CREDENTIALS_PATH
	WriteBack       bool   // FHIR_WRITE_BACK — write CKD Conditions back to FHIR Store
}

// BaseURL constructs the FHIR Store REST endpoint URL.
func (f *GoogleFHIRConfig) BaseURL() string {
	return "https://healthcare.googleapis.com/v1/projects/" + f.ProjectID +
		"/locations/" + f.Location +
		"/datasets/" + f.DatasetID +
		"/fhirStores/" + f.FhirStoreID +
		"/fhir"
}

// KB7Config configures the connection to KB-7 Terminology Service for LOINC lookups.
type KB7Config struct {
	BaseURL string // KB7_BASE_URL — default http://localhost:8092
}

type ServerConfig struct {
	Port string
}

type DatabaseConfig struct {
	URL             string
	MaxConnections  int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8131"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://kb20_user:kb20_password@localhost:5436/kb_service_20?sslmode=disable"),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6385"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		FHIR: GoogleFHIRConfig{
			Enabled:         getEnvAsBool("FHIR_SYNC_ENABLED", false),
			ProjectID:       getEnv("GOOGLE_CLOUD_PROJECT_ID", "cardiofit-905a8"),
			Location:        getEnv("GOOGLE_CLOUD_LOCATION", "asia-south1"),
			DatasetID:       getEnv("GOOGLE_CLOUD_DATASET_ID", "clinical-synthesis-hub"),
			FhirStoreID:     getEnv("GOOGLE_CLOUD_FHIR_STORE_ID", "fhir-store"),
			CredentialsPath: getEnv("GOOGLE_CLOUD_CREDENTIALS_PATH", "credentials/google-credentials.json"),
			WriteBack:       getEnvAsBool("FHIR_WRITE_BACK", false),
		},
		KB7: KB7Config{
			BaseURL: getEnv("KB7_BASE_URL", "http://localhost:8092"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
