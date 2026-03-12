package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port     string
	GRPCPort string
}

// RedisConfig holds Redis-specific configuration
type RedisConfig struct {
	URL          string
	Password     string
	DB           int
	HotCacheURL  string
	WarmCacheURL string
}

type Config struct {
	// Server configuration
	Server      ServerConfig
	Environment string
	LogLevel    string

	// Database configuration (KB-5 local database)
	DatabaseURL      string
	DatabasePassword string
	MaxConnections   int
	ConnMaxLifetime  time.Duration

	// Shared database for OHDSI/Constitutional DDI (canonical_facts)
	SharedDatabaseURL      string
	SharedDatabasePassword string

	// Redis configuration
	Redis RedisConfig

	// Interaction checking configuration
	EnableSeverityFiltering    bool
	DefaultSeverityThreshold   string
	MaxInteractionsPerRequest  int
	CacheInteractionResults    bool
	InteractionCacheTTL        time.Duration
	
	// Clinical decision support
	EnableCDSIntegration      bool
	CDSConfigName             string
	RequireOverrideReason     bool
	LogAllInteractionChecks   bool
	
	// Drug database configuration
	EnableSynonymResolution   bool
	SynonymMatchThreshold     float64
	UpdateDrugDatabase        bool
	DrugDatabaseSyncInterval  time.Duration
	
	// Performance tuning
	BatchSize                 int
	QueryTimeout              time.Duration
	ConcurrentChecks          int
	EnableQueryOptimization   bool
	
	// Interaction Matrix configuration
	EnableMatrixCaching       bool
	MaxBatchSize             int
	BatchConcurrency         int
	MaxMatrixSize            int
	MatrixUpdateInterval     time.Duration
	
	// External service URLs
	KB1DrugRulesURL          string
	KB4PatientSafetyURL      string
	KB7TerminologyURL        string
	FHIRServiceURL           string
	
	// Analytics and monitoring
	EnableInteractionAnalytics bool
	AnalyticsRetentionDays     int
	MetricsCollectionInterval  time.Duration
}

func Load() (*Config, error) {
	return &Config{
		// Server
		Server: ServerConfig{
			Port:     getEnv("PORT", "8085"),
			GRPCPort: getEnv("GRPC_PORT", "8086"),
		},
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		// Database (KB-5 local)
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://kb5_user:password@localhost:5432/kb_drug_interactions?sslmode=disable"),
		DatabasePassword: getEnv("DATABASE_PASSWORD", ""),
		MaxConnections:   getEnvAsInt("DATABASE_MAX_CONNECTIONS", 25),
		ConnMaxLifetime:  getEnvAsDuration("DATABASE_CONN_MAX_LIFETIME", "5m"),

		// Shared database for OHDSI/Constitutional DDI (canonical_facts on port 5433)
		SharedDatabaseURL:      getEnv("SHARED_DATABASE_URL", "postgres://kb_admin:kb_secure_password_2024@localhost:5433/canonical_facts?sslmode=disable"),
		SharedDatabasePassword: getEnv("SHARED_DATABASE_PASSWORD", ""),

		// Redis
		Redis: RedisConfig{
			URL:          getEnv("REDIS_URL", "redis://localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvAsInt("REDIS_DB", 5), // Use DB 5 for KB-5
			HotCacheURL:  getEnv("REDIS_HOT_CACHE_URL", "redis://localhost:6379/5"),
			WarmCacheURL: getEnv("REDIS_WARM_CACHE_URL", "redis://localhost:6379/6"),
		},

		// Interaction checking
		EnableSeverityFiltering:   getEnvAsBool("ENABLE_SEVERITY_FILTERING", true),
		DefaultSeverityThreshold:  getEnv("DEFAULT_SEVERITY_THRESHOLD", "moderate"),
		MaxInteractionsPerRequest: getEnvAsInt("MAX_INTERACTIONS_PER_REQUEST", 20),
		CacheInteractionResults:   getEnvAsBool("CACHE_INTERACTION_RESULTS", true),
		InteractionCacheTTL:       getEnvAsDuration("INTERACTION_CACHE_TTL", "1h"),

		// Clinical decision support
		EnableCDSIntegration:    getEnvAsBool("ENABLE_CDS_INTEGRATION", true),
		CDSConfigName:           getEnv("CDS_CONFIG_NAME", "default_hospital_config"),
		RequireOverrideReason:   getEnvAsBool("REQUIRE_OVERRIDE_REASON", true),
		LogAllInteractionChecks: getEnvAsBool("LOG_ALL_INTERACTION_CHECKS", true),

		// Drug database
		EnableSynonymResolution:  getEnvAsBool("ENABLE_SYNONYM_RESOLUTION", true),
		SynonymMatchThreshold:    getEnvAsFloat("SYNONYM_MATCH_THRESHOLD", 0.8),
		UpdateDrugDatabase:       getEnvAsBool("UPDATE_DRUG_DATABASE", false),
		DrugDatabaseSyncInterval: getEnvAsDuration("DRUG_DATABASE_SYNC_INTERVAL", "24h"),

		// Performance
		BatchSize:               getEnvAsInt("BATCH_SIZE", 100),
		QueryTimeout:            getEnvAsDuration("QUERY_TIMEOUT", "10s"),
		ConcurrentChecks:        getEnvAsInt("CONCURRENT_CHECKS", 5),
		EnableQueryOptimization: getEnvAsBool("ENABLE_QUERY_OPTIMIZATION", true),
		
		// Matrix configuration
		EnableMatrixCaching:     getEnvAsBool("ENABLE_MATRIX_CACHING", true),
		MaxBatchSize:           getEnvAsInt("MAX_BATCH_SIZE", 1000),
		BatchConcurrency:       getEnvAsInt("BATCH_CONCURRENCY", 10),
		MaxMatrixSize:          getEnvAsInt("MAX_MATRIX_SIZE", 10000),
		MatrixUpdateInterval:   getEnvAsDuration("MATRIX_UPDATE_INTERVAL", "24h"),

		// External services
		KB1DrugRulesURL:     getEnv("KB1_DRUG_RULES_URL", "http://localhost:8081"),
		KB4PatientSafetyURL: getEnv("KB4_PATIENT_SAFETY_URL", "http://localhost:8084"),
		KB7TerminologyURL:   getEnv("KB7_TERMINOLOGY_URL", "http://localhost:8087"),
		FHIRServiceURL:      getEnv("FHIR_SERVICE_URL", "http://localhost:8014"),

		// Analytics
		EnableInteractionAnalytics: getEnvAsBool("ENABLE_INTERACTION_ANALYTICS", true),
		AnalyticsRetentionDays:     getEnvAsInt("ANALYTICS_RETENTION_DAYS", 365),
		MetricsCollectionInterval:  getEnvAsDuration("METRICS_COLLECTION_INTERVAL", "5m"),
	}, nil
}

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
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) GetDatabaseDSN() string {
	if c.DatabasePassword != "" {
		return fmt.Sprintf("%s&password=%s", c.DatabaseURL, c.DatabasePassword)
	}
	return c.DatabaseURL
}

// GetSharedDatabaseDSN returns the shared database DSN for OHDSI/Constitutional DDI
func (c *Config) GetSharedDatabaseDSN() string {
	if c.SharedDatabasePassword != "" {
		return fmt.Sprintf("%s&password=%s", c.SharedDatabaseURL, c.SharedDatabasePassword)
	}
	return c.SharedDatabaseURL
}

func (c *Config) GetSeverityLevels() []string {
	return []string{"contraindicated", "major", "moderate", "minor"}
}

func (c *Config) GetEvidenceLevels() []string {
	return []string{"established", "probable", "theoretical", "unknown"}
}

func (c *Config) IsValidSeverity(severity string) bool {
	for _, level := range c.GetSeverityLevels() {
		if severity == level {
			return true
		}
	}
	return false
}

func (c *Config) GetSeverityPriority(severity string) int {
	priorities := map[string]int{
		"contraindicated": 4,
		"major":          3,
		"moderate":       2,
		"minor":          1,
	}
	if priority, exists := priorities[severity]; exists {
		return priority
	}
	return 0
}

func (c *Config) ShouldCacheInteraction(severity string) bool {
	if !c.CacheInteractionResults {
		return false
	}
	// Cache all interactions except minor ones in development
	if c.IsDevelopment() && severity == "minor" {
		return false
	}
	return true
}

func (c *Config) GetInteractionCacheTTL(severity string) time.Duration {
	// Higher severity interactions cached longer
	switch severity {
	case "contraindicated":
		return c.InteractionCacheTTL * 2 // 2 hours default
	case "major":
		return c.InteractionCacheTTL * 2 // 2 hours default
	case "moderate":
		return c.InteractionCacheTTL // 1 hour default
	case "minor":
		return c.InteractionCacheTTL / 2 // 30 minutes default
	default:
		return c.InteractionCacheTTL
	}
}

// ConfigProvider interface implementation
func (c *Config) GetDatabaseURL() string {
	return c.GetDatabaseDSN()
}

func (c *Config) GetCacheURL() string {
	return c.Redis.URL
}

func (c *Config) GetLogLevel() string {
	return c.LogLevel
}