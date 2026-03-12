// Package config provides configuration management for KB-9 Care Gaps Service.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the Care Gaps Service.
type Config struct {
	// Server settings
	Port         int
	Environment  string
	LogLevel     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	// Google Cloud Healthcare API (FHIR)
	GoogleCloudProjectID   string
	GoogleCloudLocation    string
	GoogleCloudDatasetID   string
	GoogleCloudFHIRStoreID string
	GoogleCredentialsPath  string

	// FHIR Server settings (fallback for non-Google)
	FHIRServerURL string
	FHIRTimeout   time.Duration
	UseGoogleFHIR bool

	// CQL settings (Vaidshala integration)
	CQLLibraryPath string
	CQLEnginePath  string

	// Terminology service (KB-7)
	TerminologyURL string

	// KB-3 Temporal/Guidelines service (Tier 7 sibling)
	KB3URL     string        // KB-3 service URL (e.g., http://kb-3-guidelines:8083)
	KB3Timeout time.Duration // Timeout for KB-3 requests
	KB3Enabled bool          // Enable KB-3 integration for temporal enrichment

	// Redis caching
	RedisURL string
	CacheTTL time.Duration

	// Feature flags
	MetricsEnabled    bool
	PlaygroundEnabled bool
	FederationEnabled bool

	// CQL Engine settings (vaidshala integration)
	UseCQLEngine bool   // Enable vaidshala CQL/Measure engine for evaluation
	Region       string // Active region for CQL evaluation (AU, IN, US)

	// Regional settings
	SupportedRegions []string
	DefaultRegion    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		// Server defaults
		Port:         getEnvInt("PORT", 8089),
		Environment:  getEnv("ENVIRONMENT", "development"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		ReadTimeout:  getEnvDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout: getEnvDuration("WRITE_TIMEOUT", 30*time.Second),

		// Google Cloud Healthcare API (FHIR)
		GoogleCloudProjectID:   getEnv("GOOGLE_CLOUD_PROJECT_ID", "cardiofit-905a8"),
		GoogleCloudLocation:    getEnv("GOOGLE_CLOUD_LOCATION", "asia-south1"),
		GoogleCloudDatasetID:   getEnv("GOOGLE_CLOUD_DATASET_ID", "clinical-synthesis-hub"),
		GoogleCloudFHIRStoreID: getEnv("GOOGLE_CLOUD_FHIR_STORE_ID", "fhir-store"),
		GoogleCredentialsPath:  getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
		UseGoogleFHIR:          getEnvBool("USE_GOOGLE_FHIR", true),

		// FHIR defaults (fallback)
		FHIRServerURL: getEnv("FHIR_SERVER_URL", "http://localhost:8080/fhir"),
		FHIRTimeout:   getEnvDuration("FHIR_TIMEOUT", 30*time.Second),

		// CQL defaults (Vaidshala)
		CQLLibraryPath: getEnv("CQL_LIBRARY_PATH", "../../vaidshala/clinical-knowledge-core"),
		CQLEnginePath:  getEnv("CQL_ENGINE_PATH", "../../vaidshala/clinical-runtime-platform"),

		// Terminology (KB-7)
		TerminologyURL: getEnv("TERMINOLOGY_URL", "http://localhost:8092"),

		// KB-3 Temporal/Guidelines (Tier 7 sibling)
		KB3URL:     getEnv("KB3_URL", "http://localhost:8083"),
		KB3Timeout: getEnvDuration("KB3_TIMEOUT", 10*time.Second),
		KB3Enabled: getEnvBool("KB3_ENABLED", true),

		// Redis
		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379/9"),
		CacheTTL: getEnvDuration("CACHE_TTL", 5*time.Minute),

		// Features
		MetricsEnabled:    getEnvBool("METRICS_ENABLED", true),
		PlaygroundEnabled: getEnvBool("ENABLE_PLAYGROUND", true),
		FederationEnabled: getEnvBool("FEDERATION_ENABLED", true),

		// CQL Engine (vaidshala)
		UseCQLEngine: getEnvBool("USE_CQL_ENGINE", true), // Enable by default
		Region:       getEnv("REGION", "AU"),             // Default to Australia

		// Regional
		SupportedRegions: getEnvSlice("SUPPORTED_REGIONS", []string{"US", "IN", "AU"}),
		DefaultRegion:    getEnv("DEFAULT_REGION", "AU"),
	}

	return cfg, nil
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// Helper functions for environment variable parsing

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvSlice(key string, defaultVal []string) []string {
	if val := os.Getenv(key); val != "" {
		return strings.Split(val, ",")
	}
	return defaultVal
}
