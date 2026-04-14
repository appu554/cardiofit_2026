package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all KB-23 Decision Cards configuration.
// Pattern: environment-based (no Viper), matching KB-22.
type Config struct {
	Port        string
	Environment string

	// Database
	DatabaseURL      string
	DBMaxConnections int
	DBConnMaxLife    time.Duration

	// Redis
	RedisURL      string
	RedisPassword string
	RedisDB       int

	// Redis TTLs
	RedisMCUGateTTL      time.Duration // MCU gate cache lifetime
	RedisPerturbationTTL time.Duration // 0 = dynamic per record
	RedisAdherenceTTL    time.Duration // adherence summary cache
	RedisGateHistoryTTL  time.Duration // gate history retention

	// Template Loading
	TemplatesDir      string
	TemplatesHotReload bool

	// Cross-KB Service URLs
	KB19URL string // Protocol Orchestrator
	KB20URL string // Patient Profile
	KB21URL string // Behavioural Intelligence
	KB22URL string // HPI Engine
	KB26URL string // Metabolic Digital Twin

	// Cross-KB Timeouts (milliseconds)
	KB19TimeoutMS int
	KB20TimeoutMS int
	KB21TimeoutMS int
	KB26TimeoutMS int

	// Confidence Thresholds (defaults - overridden per template)
	DefaultFirmPosterior        float64
	DefaultFirmMedicationChange float64
	DefaultProbablePosterior    float64
	DefaultPossiblePosterior    float64

	// Hysteresis (N-01)
	HysteresisWindowHours  int // lookback window for downgrade confirmation
	HysteresisMinSessions  int // minimum confirming sessions for downgrade

	// Safety
	HypoglycaemiaSevereThreshold   float64       // mmol/L
	HypoglycaemiaModerateThreshold float64       // mmol/L
	SafetyAlertPublishTimeout      time.Duration

	// Monitoring
	MetricsEnabled bool

	// Seasonal calendar
	Market               string // e.g. "india", "australia"
	SeasonalCalendarPath string // path to seasonal_calendar.yaml; empty = no suppression
}

func Load() *Config {
	return &Config{
		Port:        envOrDefault("PORT", "8134"),
		Environment: envOrDefault("ENVIRONMENT", "development"),

		DatabaseURL:      envOrDefault("DATABASE_URL", "postgres://kb23_user:kb23_password@localhost:5437/kb_service_23?sslmode=disable"),
		DBMaxConnections: envIntOrDefault("DB_MAX_CONNECTIONS", 25),
		DBConnMaxLife:    envDurationOrDefault("DB_CONN_MAX_LIFETIME", 30*time.Minute),

		RedisURL:      envOrDefault("REDIS_URL", "redis://localhost:6386"),
		RedisPassword: envOrDefault("REDIS_PASSWORD", ""),
		RedisDB:       envIntOrDefault("REDIS_DB", 0),

		RedisMCUGateTTL:      envDurationOrDefault("REDIS_MCU_GATE_TTL", 1*time.Hour),
		RedisPerturbationTTL: envDurationOrDefault("REDIS_PERTURBATION_TTL", 0),
		RedisAdherenceTTL:    envDurationOrDefault("REDIS_ADHERENCE_TTL", 6*time.Hour),
		RedisGateHistoryTTL:  envDurationOrDefault("REDIS_GATE_HISTORY_TTL", 30*24*time.Hour),

		TemplatesDir:      envOrDefault("TEMPLATES_DIR", "./templates"),
		TemplatesHotReload: envBoolOrDefault("TEMPLATES_HOT_RELOAD", false),

		KB19URL: envOrDefault("KB19_URL", "http://localhost:8103"),
		KB20URL: envOrDefault("KB20_URL", "http://localhost:8131"),
		KB21URL: envOrDefault("KB21_URL", "http://localhost:8133"),
		KB22URL: envOrDefault("KB22_URL", "http://localhost:8132"),
		KB26URL: envOrDefault("KB26_URL", "http://localhost:8137"),

		KB19TimeoutMS: envIntOrDefault("KB19_TIMEOUT_MS", 500),
		KB20TimeoutMS: envIntOrDefault("KB20_TIMEOUT_MS", 200),
		KB21TimeoutMS: envIntOrDefault("KB21_TIMEOUT_MS", 200),
		KB26TimeoutMS: envIntOrDefault("KB26_TIMEOUT_MS", 3000),

		DefaultFirmPosterior:        envFloatOrDefault("DEFAULT_FIRM_POSTERIOR", 0.75),
		DefaultFirmMedicationChange: envFloatOrDefault("DEFAULT_FIRM_MEDICATION_CHANGE", 0.82),
		DefaultProbablePosterior:    envFloatOrDefault("DEFAULT_PROBABLE_POSTERIOR", 0.60),
		DefaultPossiblePosterior:    envFloatOrDefault("DEFAULT_POSSIBLE_POSTERIOR", 0.40),

		HysteresisWindowHours: envIntOrDefault("HYSTERESIS_WINDOW_HOURS", 72),
		HysteresisMinSessions: envIntOrDefault("HYSTERESIS_MIN_SESSIONS", 2),

		HypoglycaemiaSevereThreshold:   envFloatOrDefault("HYPOGLYCAEMIA_SEVERE_THRESHOLD", 3.0),
		HypoglycaemiaModerateThreshold: envFloatOrDefault("HYPOGLYCAEMIA_MODERATE_THRESHOLD", 3.9),
		SafetyAlertPublishTimeout:      envDurationOrDefault("SAFETY_ALERT_PUBLISH_TIMEOUT", 2*time.Second),

		MetricsEnabled: envBoolOrDefault("METRICS_ENABLED", true),

		Market:               envOrDefault("MARKET", "india"),
		SeasonalCalendarPath: envOrDefault("SEASONAL_CALENDAR_PATH", ""),
	}
}

func (c *Config) IsDevelopment() bool { return c.Environment == "development" }
func (c *Config) IsProduction() bool  { return c.Environment == "production" }

func (c *Config) GetAddr() string { return fmt.Sprintf(":%s", c.Port) }

func (c *Config) KB19Timeout() time.Duration { return time.Duration(c.KB19TimeoutMS) * time.Millisecond }
func (c *Config) KB20Timeout() time.Duration { return time.Duration(c.KB20TimeoutMS) * time.Millisecond }
func (c *Config) KB21Timeout() time.Duration { return time.Duration(c.KB21TimeoutMS) * time.Millisecond }
func (c *Config) KB26Timeout() time.Duration { return time.Duration(c.KB26TimeoutMS) * time.Millisecond }

// HysteresisWindow returns the lookback window for N-01 downgrade confirmation.
func (c *Config) HysteresisWindow() time.Duration {
	return time.Duration(c.HysteresisWindowHours) * time.Hour
}

// HysteresisMinSessionCount returns the minimum sessions needed for N-01 downgrade.
func (c *Config) HysteresisMinSessionCount() int { return c.HysteresisMinSessions }

// --- helpers ---

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func envFloatOrDefault(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envBoolOrDefault(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDurationOrDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
