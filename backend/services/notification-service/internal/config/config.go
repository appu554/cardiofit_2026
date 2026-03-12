package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the notification service
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	Delivery   DeliveryConfig   `mapstructure:"delivery"`
	Routing    RoutingConfig    `mapstructure:"routing"`
	Escalation EscalationConfig `mapstructure:"escalation"`
	Fatigue    FatigueConfig    `mapstructure:"fatigue"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// ServerConfig contains server configuration
type ServerConfig struct {
	HTTPPort int    `mapstructure:"http_port"`
	GRPCPort int    `mapstructure:"grpc_port"`
	Env      string `mapstructure:"env"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxConnections  int           `mapstructure:"max_connections"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// KafkaConfig contains Kafka configuration
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	GroupID       string   `mapstructure:"group_id"`
	Topic         string   `mapstructure:"topic"`
	AutoOffsetReset string `mapstructure:"auto_offset_reset"`
}

// DeliveryConfig contains delivery provider configurations
type DeliveryConfig struct {
	Email EmailConfig `mapstructure:"email"`
	SMS   SMSConfig   `mapstructure:"sms"`
	Push  PushConfig  `mapstructure:"push"`
}

// EmailConfig contains email provider configuration
type EmailConfig struct {
	Provider       string `mapstructure:"provider"`
	SendGridAPIKey string `mapstructure:"sendgrid_api_key"`
	FromEmail      string `mapstructure:"from_email"`
	FromName       string `mapstructure:"from_name"`
}

// SMSConfig contains SMS provider configuration
type SMSConfig struct {
	Provider       string `mapstructure:"provider"`
	TwilioSID      string `mapstructure:"twilio_sid"`
	TwilioToken    string `mapstructure:"twilio_token"`
	TwilioFromNumber string `mapstructure:"twilio_from_number"`
}

// PushConfig contains push notification configuration
type PushConfig struct {
	Provider              string `mapstructure:"provider"`
	FirebaseCredentials   string `mapstructure:"firebase_credentials"`
	FirebaseProjectID     string `mapstructure:"firebase_project_id"`
}

// RoutingConfig contains routing engine configuration
type RoutingConfig struct {
	DefaultChannel    string        `mapstructure:"default_channel"`
	RetryAttempts     int           `mapstructure:"retry_attempts"`
	RetryDelay        time.Duration `mapstructure:"retry_delay"`
	ChannelPriorities map[string]int `mapstructure:"channel_priorities"`
}

// EscalationConfig contains escalation engine configuration
type EscalationConfig struct {
	Enabled                bool          `mapstructure:"enabled"`
	MaxEscalationLevel     int           `mapstructure:"max_escalation_level"`
	EscalationDelay        time.Duration `mapstructure:"escalation_delay"`
	CriticalTimeoutMinutes int           `mapstructure:"critical_timeout_minutes"`
	HighTimeoutMinutes     int           `mapstructure:"high_timeout_minutes"`
	EnableVoiceEscalation  bool          `mapstructure:"enable_voice_escalation"`
	CriticalChannels       []string      `mapstructure:"critical_channels"`
}

// FatigueConfig contains fatigue management configuration
type FatigueConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	WindowDuration    time.Duration `mapstructure:"window_duration"`
	MaxNotifications  int           `mapstructure:"max_notifications"`
	QuietHoursStart   string        `mapstructure:"quiet_hours_start"`
	QuietHoursEnd     string        `mapstructure:"quiet_hours_end"`
	PriorityThreshold string        `mapstructure:"priority_threshold"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Set defaults
	setDefaults()

	// Enable environment variable binding
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.http_port", 8060)
	viper.SetDefault("server.grpc_port", 50060)
	viper.SetDefault("server.env", "development")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	// Kafka defaults
	viper.SetDefault("kafka.group_id", "notification-service")
	viper.SetDefault("kafka.topic", "clinical-alerts")
	viper.SetDefault("kafka.auto_offset_reset", "earliest")

	// Routing defaults
	viper.SetDefault("routing.default_channel", "email")
	viper.SetDefault("routing.retry_attempts", 3)
	viper.SetDefault("routing.retry_delay", "30s")

	// Escalation defaults
	viper.SetDefault("escalation.enabled", true)
	viper.SetDefault("escalation.max_escalation_level", 3)
	viper.SetDefault("escalation.escalation_delay", "5m")
	viper.SetDefault("escalation.critical_timeout_minutes", 5)
	viper.SetDefault("escalation.high_timeout_minutes", 15)
	viper.SetDefault("escalation.enable_voice_escalation", true)
	viper.SetDefault("escalation.critical_channels", []string{"sms", "push"})

	// Fatigue defaults
	viper.SetDefault("fatigue.enabled", true)
	viper.SetDefault("fatigue.window_duration", "1h")
	viper.SetDefault("fatigue.max_notifications", 10)
	viper.SetDefault("fatigue.quiet_hours_start", "22:00")
	viper.SetDefault("fatigue.quiet_hours_end", "07:00")
	viper.SetDefault("fatigue.priority_threshold", "high")

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
}
