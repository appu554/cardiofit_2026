// Package config provides database connection configuration for Knowledge Base services.
// This configuration follows the KB1 Implementation Plan architecture for the Canonical Fact Store.
package config

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DatabaseConfig holds the PostgreSQL connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// Query settings
	QueryTimeout     time.Duration
	MigrationTimeout time.Duration
}

// DefaultConfig returns the default database configuration
// Uses environment variables with sensible defaults matching docker-compose.phase1.yml
func DefaultConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvIntOrDefault("DB_PORT", 5433),
		User:     getEnvOrDefault("DB_USER", "kb_admin"),
		Password: getEnvOrDefault("DB_PASSWORD", "kb_secure_password_2024"),
		Database: getEnvOrDefault("DB_NAME", "canonical_facts"),
		SSLMode:  getEnvOrDefault("DB_SSL_MODE", "disable"),

		// Connection pool - tuned for clinical workloads
		MaxOpenConns:    getEnvIntOrDefault("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvIntOrDefault("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: getDurationOrDefault("DB_CONN_MAX_LIFETIME", 1*time.Hour),
		ConnMaxIdleTime: getDurationOrDefault("DB_CONN_MAX_IDLE_TIME", 15*time.Minute),

		// Query settings
		QueryTimeout:     getDurationOrDefault("DB_QUERY_TIMEOUT", 30*time.Second),
		MigrationTimeout: getDurationOrDefault("DB_MIGRATION_TIMEOUT", 5*time.Minute),
	}
}

// ProductionConfig returns production-tuned database configuration
func ProductionConfig() *DatabaseConfig {
	cfg := DefaultConfig()

	// Production overrides
	cfg.SSLMode = getEnvOrDefault("DB_SSL_MODE", "require")
	cfg.MaxOpenConns = getEnvIntOrDefault("DB_MAX_OPEN_CONNS", 50)
	cfg.MaxIdleConns = getEnvIntOrDefault("DB_MAX_IDLE_CONNS", 25)
	cfg.ConnMaxLifetime = getDurationOrDefault("DB_CONN_MAX_LIFETIME", 2*time.Hour)

	return cfg
}

// ConnectionString returns the PostgreSQL connection string
func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// DSN returns the data source name for database/sql
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

// Connect establishes a database connection with the configured pool settings
func (c *DatabaseConfig) Connect() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(c.ConnMaxLifetime)
	db.SetConnMaxIdleTime(c.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// ConnectWithRetry attempts to connect with exponential backoff
func (c *DatabaseConfig) ConnectWithRetry(maxAttempts int) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err = c.Connect()
		if err == nil {
			return db, nil
		}

		if attempt < maxAttempts {
			backoff := time.Duration(attempt*attempt) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			fmt.Printf("Database connection attempt %d/%d failed, retrying in %v: %v\n",
				attempt, maxAttempts, backoff, err)
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxAttempts, err)
}

// HealthCheck performs a database health check
func HealthCheck(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Simple query to verify database is responding
	var result int
	err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
