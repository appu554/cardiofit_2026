// Package config provides configuration management for KB-13 Quality Measures Engine.
//
// Configuration is loaded from environment variables with sensible defaults.
// All date-related settings use ISO 8601 durations for consistency with CQL.
package config

import (
	"os"
	"strconv"
	"time"
)

const (
	// ServiceName is the canonical name of this service
	ServiceName = "kb-13-quality-measures"
	// Version is the current service version
	Version = "1.0.0"
	// DefaultPort is the default HTTP server port
	DefaultPort = 8113
)

// Config holds all configuration for KB-13
type Config struct {
	Server       ServerConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	Calculator   CalculatorConfig
	Scheduler    SchedulerConfig
	Integrations IntegrationsConfig
	Metrics      MetricsConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port           int
	Environment    string
	LogLevel       string
	MeasuresPath   string
	BenchmarksPath string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
	MaxConns int
}

// RedisConfig holds Redis configuration for caching
type RedisConfig struct {
	URL          string
	EnableCaching bool
	CacheTTL     time.Duration
}

// CalculatorConfig holds calculation engine settings
type CalculatorConfig struct {
	MaxConcurrent int
	Timeout       time.Duration
	BatchSize     int
}

// IntegrationsConfig holds external service URLs
type IntegrationsConfig struct {
	VaidshalaURL      string
	KB7URL            string
	KB18URL           string
	KB19URL           string
	PatientServiceURL string
}

// MetricsConfig holds Prometheus metrics configuration
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// SchedulerConfig holds automated calculation scheduling settings
type SchedulerConfig struct {
	Enabled            bool
	DailyEnabled       bool
	WeeklyEnabled      bool
	MonthlyEnabled     bool
	QuarterlyEnabled   bool
	DailyInterval      time.Duration
	WeeklyInterval     time.Duration
	MonthlyInterval    time.Duration
	WeeklyRunDay       time.Weekday
	MonthlyRunDay      int
	RunOnStart         bool
	CalculationTimeout time.Duration
}

// Load reads configuration from environment variables with defaults
func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port:           getEnvInt("KB13_PORT", DefaultPort),
			Environment:    getEnvString("KB13_ENVIRONMENT", "development"),
			LogLevel:       getEnvString("KB13_LOG_LEVEL", "info"),
			MeasuresPath:   getEnvString("KB13_MEASURES_PATH", "./measures"),
			BenchmarksPath: getEnvString("KB13_BENCHMARKS_PATH", "./benchmarks"),
			ReadTimeout:    getEnvDuration("KB13_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:   getEnvDuration("KB13_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getEnvDuration("KB13_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnvString("KB13_DB_HOST", "localhost"),
			Port:     getEnvInt("KB13_DB_PORT", 5450),
			Name:     getEnvString("KB13_DB_NAME", "kb13_quality"),
			User:     getEnvString("KB13_DB_USER", "kb13user"),
			Password: getEnvString("KB13_DB_PASSWORD", "kb13password"),
			SSLMode:  getEnvString("KB13_DB_SSLMODE", "disable"),
			MaxConns: getEnvInt("KB13_DB_MAX_CONNS", 25),
		},
		Redis: RedisConfig{
			URL:          getEnvString("KB13_REDIS_URL", "redis://localhost:6393"),
			EnableCaching: getEnvBool("KB13_ENABLE_CACHING", true),
			CacheTTL:     getEnvDuration("KB13_CACHE_TTL", 15*time.Minute),
		},
		Calculator: CalculatorConfig{
			MaxConcurrent: getEnvInt("KB13_MAX_CONCURRENT", 50),
			Timeout:       getEnvDuration("KB13_CALC_TIMEOUT", 60*time.Second),
			BatchSize:     getEnvInt("KB13_BATCH_SIZE", 1000),
		},
		Integrations: IntegrationsConfig{
			VaidshalaURL:      getEnvString("VAIDSHALA_URL", "http://localhost:8096"),
			KB7URL:            getEnvString("KB7_URL", "http://localhost:8092"),
			KB18URL:           getEnvString("KB18_URL", "http://localhost:8118"),
			KB19URL:           getEnvString("KB19_URL", "http://localhost:8119"),
			PatientServiceURL: getEnvString("PATIENT_SERVICE_URL", "http://localhost:8080"),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvBool("KB13_METRICS_ENABLED", true),
			Path:    getEnvString("KB13_METRICS_PATH", "/metrics"),
		},
		Scheduler: SchedulerConfig{
			Enabled:            getEnvBool("KB13_SCHEDULER_ENABLED", false),
			DailyEnabled:       getEnvBool("KB13_SCHEDULER_DAILY", true),
			WeeklyEnabled:      getEnvBool("KB13_SCHEDULER_WEEKLY", true),
			MonthlyEnabled:     getEnvBool("KB13_SCHEDULER_MONTHLY", true),
			QuarterlyEnabled:   getEnvBool("KB13_SCHEDULER_QUARTERLY", true),
			DailyInterval:      getEnvDuration("KB13_SCHEDULER_DAILY_INTERVAL", 24*time.Hour),
			WeeklyInterval:     getEnvDuration("KB13_SCHEDULER_WEEKLY_INTERVAL", 24*time.Hour),
			MonthlyInterval:    getEnvDuration("KB13_SCHEDULER_MONTHLY_INTERVAL", 24*time.Hour),
			WeeklyRunDay:       time.Weekday(getEnvInt("KB13_SCHEDULER_WEEKLY_DAY", int(time.Sunday))),
			MonthlyRunDay:      getEnvInt("KB13_SCHEDULER_MONTHLY_DAY", 1),
			RunOnStart:         getEnvBool("KB13_SCHEDULER_RUN_ON_START", false),
			CalculationTimeout: getEnvDuration("KB13_SCHEDULER_CALC_TIMEOUT", 30*time.Minute),
		},
	}, nil
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + strconv.Itoa(c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.Name +
		" sslmode=" + c.SSLMode
}

// Helper functions for environment variable parsing

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
