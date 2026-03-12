package config

import (
	"log"
	"os"
)

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Elasticsearch ElasticsearchConfig
	Metrics       MetricsConfig
}

type ServerConfig struct {
	Port string
	Environment string
	Debug bool
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
	SSLMode  string
	MaxConns int
	Timeout  int
}

type RedisConfig struct {
	Address  string
	Password string
	Database int
	Timeout  int
}

type ElasticsearchConfig struct {
	Addresses []string
	Username  string
	Password  string
	CloudID   string
	APIKey    string
	Enabled   bool
}

type MetricsConfig struct {
	Enabled bool
	Path    string
}

func LoadConfig() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port:        getEnvWithDefault("PORT", "8086"),
			Environment: getEnvWithDefault("ENVIRONMENT", "development"),
			Debug:       getEnvWithDefault("DEBUG", "true") == "true",
		},
		Database: DatabaseConfig{
			Host:     getEnvWithDefault("DB_HOST", "localhost"),
			Port:     getEnvWithDefault("DB_PORT", "5433"),
			Database: getEnvWithDefault("DB_NAME", "kb_formulary"),
			Username: getEnvWithDefault("DB_USER", "postgres"),
			Password: getEnvWithDefault("DB_PASSWORD", "password"),
			SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
			MaxConns: 25,
			Timeout:  30,
		},
		Redis: RedisConfig{
			Address:  getEnvWithDefault("REDIS_URL", "localhost:6380"),
			Password: os.Getenv("REDIS_PASSWORD"),
			Database: 6, // Use DB 6 for KB-6
			Timeout:  5,
		},
		Elasticsearch: ElasticsearchConfig{
			Addresses: []string{getEnvWithDefault("ELASTICSEARCH_URL", "http://localhost:9200")},
			Username:  os.Getenv("ELASTICSEARCH_USERNAME"),
			Password:  os.Getenv("ELASTICSEARCH_PASSWORD"),
			CloudID:   os.Getenv("ELASTICSEARCH_CLOUD_ID"),
			APIKey:    os.Getenv("ELASTICSEARCH_API_KEY"),
			Enabled:   getEnvWithDefault("ELASTICSEARCH_ENABLED", "false") == "true",
		},
		Metrics: MetricsConfig{
			Enabled: getEnvWithDefault("METRICS_ENABLED", "true") == "true",
			Path:    getEnvWithDefault("METRICS_PATH", "/metrics"),
		},
	}, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

func (c *Config) GetDatabaseURL() string {
	return "postgres://" + c.Database.Username + ":" + c.Database.Password +
		"@" + c.Database.Host + ":" + c.Database.Port +
		"/" + c.Database.Database + "?sslmode=" + c.Database.SSLMode
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func init() {
	// Load .env file if it exists (for development)
	if _, err := os.Stat(".env"); err == nil {
		// Could load .env file here if needed
		log.Println("Loading environment variables from .env file")
	}
}