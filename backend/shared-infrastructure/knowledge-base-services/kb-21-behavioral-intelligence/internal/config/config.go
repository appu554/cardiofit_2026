package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for KB-21 Behavioral Intelligence Service.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig

	// Service identity
	ServiceName string
	Environment string
	LogLevel    string

	// Cross-KB integration
	KB20PatientProfileURL string
	KB22DiagnosticURL     string
	KB23SafetyURL         string // G-01/G-03: KB-23 direct fast-path for DecisionCard generation
	KB19OrchestratorURL   string
	KB4PatientSafetyURL   string
	KB1DrugRulesURL       string // FDC component decomposition for adherence tracking

	// G-04: Pre-gateway default adherence.
	// When no InteractionEvents exist (WhatsApp gateway not connected),
	// adherence_score defaults to this instead of 0.
	// 0.70 = assume average adherence when there is no evidence.
	// Source flagged as DEFAULT_PRE_GATEWAY in API responses.
	PreGatewayDefaultAdherence float64

	// Behavioral computation
	AdherenceWindow30d          int
	AdherenceWindow7d           int
	PhenotypeEvalIntervalHours  int
	DecayPredictionWindowDays   int
	OutcomeCorrelationMinEvents int

	// Loop trust thresholds (informational — V-MCU owns control logic)
	LoopTrustAutoThreshold      float64
	LoopTrustAssistedThreshold  float64
	LoopTrustConfirmThreshold   float64
	LoopTrustDisabledThreshold  float64

	// Nudge engine
	NudgeMaxPerDay     int
	NudgeCooldownHours int

	// BCE v2.0 feature flags
	ColdStartEnabled            bool
	GamificationEnabled         bool
	PopulationLearningEnabled   bool
	PopulationLearningMinCohort int  // minimum patients for population learning cycle (default 50)
	TimingOptimizationEnabled   bool

	// Festival calendar
	FestivalCalendarPath string

	// Event bus
	EventBusEnabled bool
	KafkaBrokers    string
	KafkaTopic      string

	// Performance
	BatchSize       int
	QueryTimeout    time.Duration
	MaxConnections  int
	ConnMaxLifetime time.Duration
}

type ServerConfig struct {
	Port     string
	GRPCPort string
}

type DatabaseConfig struct {
	URL             string
	Password        string
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
	cfg := &Config{
		Server: ServerConfig{
			Port:     getEnv("PORT", "8133"),
			GRPCPort: getEnv("GRPC_PORT", "8094"),
		},
		ServiceName: "kb-21-behavioral-intelligence",
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://kb21_user:kb21_pass@localhost:5433/kb_behavioral_intelligence"),
			Password:        getEnv("DATABASE_PASSWORD", ""),
			MaxConnections:  getEnvAsInt("DB_MAX_CONNECTIONS", 25),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},

		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6380/21"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 21),
		},

		// Cross-KB URLs
		KB20PatientProfileURL: getEnv("KB20_PATIENT_PROFILE_URL", "http://localhost:8131"),
		KB22DiagnosticURL:     getEnv("KB22_DIAGNOSTIC_URL", "http://localhost:8132"),
		KB23SafetyURL:         getEnv("KB23_SAFETY_URL", "http://localhost:8134"),
		KB19OrchestratorURL:   getEnv("KB19_ORCHESTRATOR_URL", "http://localhost:8103"),
		KB4PatientSafetyURL:   getEnv("KB4_PATIENT_SAFETY_URL", "http://localhost:8088"),
		KB1DrugRulesURL:       getEnv("KB1_DRUG_RULES_URL", "http://localhost:8085"),

		// G-04: Pre-gateway default
		PreGatewayDefaultAdherence: getEnvAsFloat("PRE_GATEWAY_DEFAULT_ADHERENCE", 0.70),

		// Behavioral computation defaults
		AdherenceWindow30d:          30,
		AdherenceWindow7d:           7,
		PhenotypeEvalIntervalHours:  getEnvAsInt("PHENOTYPE_EVAL_INTERVAL_HOURS", 24),
		DecayPredictionWindowDays:   getEnvAsInt("DECAY_PREDICTION_WINDOW_DAYS", 14),
		OutcomeCorrelationMinEvents: getEnvAsInt("OUTCOME_CORRELATION_MIN_EVENTS", 5),

		// Loop trust thresholds (informational defaults per review Section 1.1)
		LoopTrustAutoThreshold:     getEnvAsFloat("LOOP_TRUST_AUTO_THRESHOLD", 0.75),
		LoopTrustAssistedThreshold: getEnvAsFloat("LOOP_TRUST_ASSISTED_THRESHOLD", 0.55),
		LoopTrustConfirmThreshold:  getEnvAsFloat("LOOP_TRUST_CONFIRM_THRESHOLD", 0.35),
		LoopTrustDisabledThreshold: getEnvAsFloat("LOOP_TRUST_DISABLED_THRESHOLD", 0.20),

		// Nudge engine
		NudgeMaxPerDay:     getEnvAsInt("NUDGE_MAX_PER_DAY", 3),
		NudgeCooldownHours: getEnvAsInt("NUDGE_COOLDOWN_HOURS", 4),

		// BCE v2.0 feature flags
		ColdStartEnabled:            getEnvAsBool("BCE_COLD_START_ENABLED", true),
		GamificationEnabled:         getEnvAsBool("BCE_GAMIFICATION_ENABLED", false),
		PopulationLearningEnabled:   getEnvAsBool("BCE_POPULATION_LEARNING_ENABLED", false),
		PopulationLearningMinCohort: getEnvAsInt("BCE_POPULATION_LEARNING_MIN_COHORT", 50),
		TimingOptimizationEnabled:   getEnvAsBool("BCE_TIMING_OPTIMIZATION_ENABLED", false),

		// Festival calendar
		FestivalCalendarPath: getEnv("FESTIVAL_CALENDAR_PATH", "data/festivals_india_2026.yaml"),

		// Event bus
		EventBusEnabled: getEnvAsBool("EVENT_BUS_ENABLED", false),
		KafkaBrokers:    getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:      getEnv("KAFKA_TOPIC_PREFIX", "kb21"),

		// Performance
		BatchSize:       getEnvAsInt("BATCH_SIZE", 100),
		QueryTimeout:    getEnvAsDuration("QUERY_TIMEOUT", "10s"),
		MaxConnections:  getEnvAsInt("MAX_CONNECTIONS", 25),
		ConnMaxLifetime: getEnvAsDuration("CONN_MAX_LIFETIME", "5m"),
	}

	if cfg.Database.URL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func (c *Config) GetDatabaseDSN() string {
	return c.Database.URL
}

// --- Environment helpers ---

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

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvAsDuration(key, defaultValue string) time.Duration {
	val := getEnv(key, defaultValue)
	d, err := time.ParseDuration(val)
	if err != nil {
		d, _ = time.ParseDuration(defaultValue)
	}
	return d
}
