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
	KB21     KB21Config
	KB25     KB25Config
	KB26     KB26Config
	PREVENT  PREVENTConfig

	Environment string
	LogLevel    string
}

// PREVENTConfig holds AHA PREVENT calculator parameters.
// Defaults match config/prevent_config.yaml; override via env vars.
type PREVENTConfig struct {
	IntensiveTargetThreshold float64 // PREVENT_INTENSIVE_THRESHOLD — 10yr CVD risk cutoff for intensive SBP target (default 0.075)

	// South Asian BMI calibration (offset added to BMI for patients in 23-30 range)
	SouthAsianBMICalibrationEnabled bool    // PREVENT_SA_CALIBRATION_ENABLED
	SouthAsianBMICalibrationOffset  float64 // PREVENT_SA_BMI_OFFSET (default 3.0)
	SouthAsianBMICalibrationLower   float64 // PREVENT_SA_BMI_LOWER (default 23.0)
	SouthAsianBMICalibrationUpper   float64 // PREVENT_SA_BMI_UPPER (default 30.0)
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

// KB21Config configures the connection to KB-21 Behavioral Intelligence Service
// for festival calendar data (P4 perturbation).
type KB21Config struct {
	BaseURL string // KB21_BASE_URL — default http://localhost:8133
}

// KB25Config configures the connection to KB-25 Lifestyle Knowledge Graph service.
// Used by ProtocolService to validate safety rules before activation and to
// obtain projected outcomes after phase transitions.
type KB25Config struct {
	BaseURL string // KB25_BASE_URL — default http://localhost:8136
}

// KB26Config configures the connection to KB-26 Metabolic Digital Twin service.
// Used by callers that populate TrajectoryInput to fetch MRI (Metabolic Risk Index)
// data for the MRI forcing rules (Spec §7).
type KB26Config struct {
	BaseURL string // KB26_BASE_URL — default http://localhost:8137
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
			ProjectID:       getEnv("GOOGLE_CLOUD_PROJECT_ID", "project-2bbef9ac-174b-4b59-8fe"),
			Location:        getEnv("GOOGLE_CLOUD_LOCATION", "asia-south1"),
			DatasetID:       getEnv("GOOGLE_CLOUD_DATASET_ID", "vaidshala-clinical"),
			FhirStoreID:     getEnv("GOOGLE_CLOUD_FHIR_STORE_ID", "cardiofit-fhir-r4"),
			CredentialsPath: getEnv("GOOGLE_CLOUD_CREDENTIALS_PATH", "credentials/google-credentials.json"),
			WriteBack:       getEnvAsBool("FHIR_WRITE_BACK", false),
		},
		KB7: KB7Config{
			BaseURL: getEnv("KB7_BASE_URL", "http://localhost:8092"),
		},
		KB21: KB21Config{
			BaseURL: getEnv("KB21_BASE_URL", "http://localhost:8133"),
		},
		KB25: KB25Config{
			BaseURL: getEnv("KB25_BASE_URL", "http://localhost:8136"),
		},
		KB26: KB26Config{
			BaseURL: getEnv("KB26_BASE_URL", "http://localhost:8137"),
		},
		PREVENT: PREVENTConfig{
			IntensiveTargetThreshold:        getEnvAsFloat64("PREVENT_INTENSIVE_THRESHOLD", 0.075),
			SouthAsianBMICalibrationEnabled: getEnvAsBool("PREVENT_SA_CALIBRATION_ENABLED", false),
			SouthAsianBMICalibrationOffset:  getEnvAsFloat64("PREVENT_SA_BMI_OFFSET", 3.0),
			SouthAsianBMICalibrationLower:   getEnvAsFloat64("PREVENT_SA_BMI_LOWER", 23.0),
			SouthAsianBMICalibrationUpper:   getEnvAsFloat64("PREVENT_SA_BMI_UPPER", 30.0),
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

func getEnvAsFloat64(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
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
