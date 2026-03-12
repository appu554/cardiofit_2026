package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Neo4jRegionConfig holds configuration for a single regional Neo4j instance
// Each region (AU, IN, US) has its own Neo4j database with region-specific terminology
type Neo4jRegionConfig struct {
	URL      string `json:"url"`      // Neo4j bolt URL (e.g., bolt://neo4j-au:7687)
	Username string `json:"username"` // Neo4j username
	Password string `json:"password"` // Neo4j password
	Database string `json:"database"` // Database name (e.g., kb7-au, kb7-in, kb7-us)
	Enabled  bool   `json:"enabled"`  // Whether this region is enabled
}

type Config struct {
	// Server configuration
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	LogLevel    int    `json:"log_level"`

	// Database configuration
	DatabaseURL    string `json:"database_url"`
	MigrationsPath string `json:"migrations_path"`

	// Cache configuration
	RedisURL string `json:"redis_url"`

	// Knowledge Base specific configuration
	SupportedRegions []string `json:"supported_regions"`
	TerminologyDB    string   `json:"terminology_db"`

	// GraphQL configuration
	GraphQLEndpoint    string `json:"graphql_endpoint"`
	GraphQLIntrospect  bool   `json:"graphql_introspect"`
	GraphQLPlayground  bool   `json:"graphql_playground"`

	// Federation configuration
	FederationEnabled bool   `json:"federation_enabled"`
	GatewayURL        string `json:"gateway_url"`

	// Monitoring configuration
	MetricsEnabled bool   `json:"metrics_enabled"`
	HealthEndpoint string `json:"health_endpoint"`

	// Evidence Envelope configuration
	EvidenceEnvelopeEnabled bool   `json:"evidence_envelope_enabled"`
	EvidenceEnvelopeDB      string `json:"evidence_envelope_db"`

	// GraphDB configuration (Semantic Layer)
	GraphDBURL        string `json:"graphdb_url"`
	GraphDBRepository string `json:"graphdb_repository"`
	GraphDBUsername   string `json:"graphdb_username"`
	GraphDBPassword   string `json:"graphdb_password"`
	GraphDBEnabled    bool   `json:"graphdb_enabled"`

	// Neo4j configuration (Read Replica for fast traversals - Phase 6)
	// Single-region mode (legacy, for backward compatibility)
	Neo4jURL      string `json:"neo4j_url"`
	Neo4jUsername string `json:"neo4j_username"`
	Neo4jPassword string `json:"neo4j_password"`
	Neo4jDatabase string `json:"neo4j_database"`
	Neo4jEnabled  bool   `json:"neo4j_enabled"`

	// Multi-region Neo4j configuration (Phase 7 - Regional Kernels)
	// Enables separate Neo4j databases per region for data sovereignty
	Neo4jMultiRegionEnabled bool              `json:"neo4j_multi_region_enabled"`
	Neo4jDefaultRegion      string            `json:"neo4j_default_region"`
	Neo4jRegions            map[string]Neo4jRegionConfig `json:"neo4j_regions"`

	// Kafka configuration (CDC - Phase 6.2)
	KafkaBrokers   string `json:"kafka_brokers"`
	KafkaTopic     string `json:"kafka_topic"`
	KafkaGroupID   string `json:"kafka_group_id"`
	KafkaCDCEnable bool   `json:"kafka_cdc_enable"`

	// RuleManager configuration
	SeedBuiltinValueSets bool `json:"seed_builtin_value_sets"`
}

func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	config := &Config{
		Port:        getEnvAsInt("PORT", 8087),
		Environment: getEnv("ENVIRONMENT", "development"),
		Version:     getEnv("VERSION", "1.0.0+sha.initial"),
		LogLevel:    getEnvAsInt("LOG_LEVEL", int(logrus.InfoLevel)),

		DatabaseURL:    getEnv("DATABASE_URL", "postgresql://kb_user:kb_password@localhost:5433/clinical_governance"),
		MigrationsPath: getEnv("MIGRATIONS_PATH", "./migrations"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6380/7"),

		SupportedRegions: getEnvAsStringSlice("SUPPORTED_REGIONS", []string{"US", "EU", "CA", "AU"}),
		TerminologyDB:    getEnv("TERMINOLOGY_DB", "clinical_governance"),

		GraphQLEndpoint:   getEnv("GRAPHQL_ENDPOINT", "/graphql"),
		GraphQLIntrospect: getEnvAsBool("GRAPHQL_INTROSPECT", true),
		GraphQLPlayground: getEnvAsBool("GRAPHQL_PLAYGROUND", true),

		FederationEnabled: getEnvAsBool("FEDERATION_ENABLED", true),
		GatewayURL:        getEnv("GATEWAY_URL", "http://localhost:4000/graphql"),

		MetricsEnabled: getEnvAsBool("METRICS_ENABLED", true),
		HealthEndpoint: getEnv("HEALTH_ENDPOINT", "/health"),

		EvidenceEnvelopeEnabled: getEnvAsBool("EVIDENCE_ENVELOPE_ENABLED", true),
		EvidenceEnvelopeDB:      getEnv("EVIDENCE_ENVELOPE_DB", "clinical_governance"),

		// GraphDB configuration (Semantic Layer)
		GraphDBURL:        getEnv("GRAPHDB_URL", "http://localhost:7200"),
		GraphDBRepository: getEnv("GRAPHDB_REPOSITORY", "kb7-terminology"),
		GraphDBUsername:   getEnv("GRAPHDB_USERNAME", ""),
		GraphDBPassword:   getEnv("GRAPHDB_PASSWORD", ""),
		GraphDBEnabled:    getEnvAsBool("GRAPHDB_ENABLED", true),

		// Neo4j configuration (Read Replica for fast traversals - Phase 6)
		// Single-region mode (legacy, for backward compatibility)
		Neo4jURL:      getEnv("NEO4J_URL", "bolt://localhost:7687"),
		Neo4jUsername: getEnv("NEO4J_USERNAME", "neo4j"),
		Neo4jPassword: getEnv("NEO4J_PASSWORD", "password"),
		Neo4jDatabase: getEnv("NEO4J_DATABASE", "kb7"),
		Neo4jEnabled:  getEnvAsBool("NEO4J_ENABLED", false),

		// Multi-region Neo4j configuration (Phase 7 - Regional Kernels)
		Neo4jMultiRegionEnabled: getEnvAsBool("NEO4J_MULTI_REGION_ENABLED", false),
		Neo4jDefaultRegion:      getEnv("NEO4J_DEFAULT_REGION", "us"),
		Neo4jRegions:            loadNeo4jRegions(),

		// Kafka configuration (CDC - Phase 6.2)
		KafkaBrokers:   getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:     getEnv("KAFKA_CDC_TOPIC", "kb7.graphdb.changes"),
		KafkaGroupID:   getEnv("KAFKA_GROUP_ID", "kb7-neo4j-sync"),
		KafkaCDCEnable: getEnvAsBool("KAFKA_CDC_ENABLE", false),

		// RuleManager configuration
		// Set SEED_BUILTIN_VALUE_SETS=true on first run to populate database
		SeedBuiltinValueSets: getEnvAsBool("SEED_BUILTIN_VALUE_SETS", false),
	}

	return config, nil
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

func getEnvAsStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		// In production, you might want more sophisticated parsing
		return []string{value}
	}
	return defaultValue
}

// LoadFromFile loads configuration from a file
func LoadFromFile(filename string) (*Config, error) {
	// For now, just load from environment
	// TODO: Implement actual file loading
	return LoadConfig()
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	return Load()
}

// loadNeo4jRegions loads multi-region Neo4j configuration from environment variables
// Environment variable pattern: NEO4J_{REGION}_{FIELD}
// Example: NEO4J_AU_URL, NEO4J_AU_USERNAME, NEO4J_AU_PASSWORD, NEO4J_AU_DATABASE
func loadNeo4jRegions() map[string]Neo4jRegionConfig {
	regions := make(map[string]Neo4jRegionConfig)

	// Define supported regions with their environment variable prefixes
	// AU = Australia (AMT terminology)
	// IN = India (CDCI terminology)
	// US = United States (RxNorm/SNOMED-US/LOINC)
	regionPrefixes := []string{"AU", "IN", "US"}

	for _, region := range regionPrefixes {
		prefix := "NEO4J_" + region + "_"

		// Only add region if URL is configured
		url := getEnv(prefix+"URL", "")
		if url != "" {
			regions[region] = Neo4jRegionConfig{
				URL:      url,
				Username: getEnv(prefix+"USERNAME", "neo4j"),
				Password: getEnv(prefix+"PASSWORD", ""),
				Database: getEnv(prefix+"DATABASE", "kb7-"+strings.ToLower(region)),
				Enabled:  getEnvAsBool(prefix+"ENABLED", true),
			}
		}
	}

	return regions
}

// GetNeo4jConfigForRegion returns the Neo4j configuration for a specific region
// Falls back to default single-region config if multi-region is not enabled
func (c *Config) GetNeo4jConfigForRegion(region string) Neo4jRegionConfig {
	if !c.Neo4jMultiRegionEnabled {
		// Return legacy single-region config
		return Neo4jRegionConfig{
			URL:      c.Neo4jURL,
			Username: c.Neo4jUsername,
			Password: c.Neo4jPassword,
			Database: c.Neo4jDatabase,
			Enabled:  c.Neo4jEnabled,
		}
	}

	// Normalize region to uppercase
	region = strings.ToUpper(region)

	// Look up region-specific config
	if regionConfig, ok := c.Neo4jRegions[region]; ok {
		return regionConfig
	}

	// Fall back to default region
	if regionConfig, ok := c.Neo4jRegions[strings.ToUpper(c.Neo4jDefaultRegion)]; ok {
		return regionConfig
	}

	// Ultimate fallback to legacy config
	return Neo4jRegionConfig{
		URL:      c.Neo4jURL,
		Username: c.Neo4jUsername,
		Password: c.Neo4jPassword,
		Database: c.Neo4jDatabase,
		Enabled:  c.Neo4jEnabled,
	}
}