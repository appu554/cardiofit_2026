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
		return // .env is optional
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

// WhatsAppConfig holds WhatsApp Business API settings.
type WhatsAppConfig struct {
	PhoneNumberID string
	AccessToken   string
	AppSecret     string
	VerifyToken   string
}

// ABDMConfig holds ABDM (Ayushman Bharat Digital Mission) settings.
type ABDMConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	IsSandbox    bool
}

// Config holds all configuration for the Intake-Onboarding Service.
type Config struct {
	Server              ServerConfig
	Database            DatabaseConfig
	Redis               RedisConfig
	FHIR                fhirclient.GoogleFHIRConfig
	Kafka               KafkaConfig
	WhatsApp            WhatsAppConfig
	ABDM                ABDMConfig
	KB24URL             string // KB-24 Safety Constraint Engine base URL
	Environment         string
	LogLevel            string
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

// KafkaConfig holds Kafka broker settings.
type KafkaConfig struct {
	Brokers []string
	GroupID string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8141"),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://intake_user:intake_password@localhost:5433/intake_service?sslmode=disable"),
			MaxConnections:  int32(getEnvAsInt("DATABASE_MAX_CONNECTIONS", 10)),
			ConnMaxLifetime: time.Duration(getEnvAsInt("DATABASE_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6380"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 3),
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
			GroupID: getEnv("KAFKA_GROUP_ID", "intake-onboarding-service"),
		},
		WhatsApp: WhatsAppConfig{
			PhoneNumberID: getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),
			AccessToken:   getEnv("WHATSAPP_ACCESS_TOKEN", ""),
			AppSecret:     getEnv("WHATSAPP_APP_SECRET", ""),
			VerifyToken:   getEnv("WHATSAPP_VERIFY_TOKEN", "cardiofit-intake-verify"),
		},
		ABDM: ABDMConfig{
			BaseURL:      getEnv("ABDM_BASE_URL", "https://abdm.gov.in"),
			ClientID:     getEnv("ABDM_CLIENT_ID", ""),
			ClientSecret: getEnv("ABDM_CLIENT_SECRET", ""),
			IsSandbox:    getEnvAsBool("ABDM_SANDBOX", true),
		},
		KB24URL: getEnv("KB24_URL", "http://localhost:8201"),
		Environment:         getEnv("ENVIRONMENT", "development"),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
	}
}

// IsDevelopment returns true when the service runs in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
