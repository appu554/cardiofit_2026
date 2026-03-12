package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"notification-service/internal/database"
)

func main() {
	// Define command-line flags
	command := flag.String("command", "up", "Migration command: up, down, status, seed")
	host := flag.String("host", getEnv("DB_HOST", "localhost"), "Database host")
	port := flag.Int("port", getEnvInt("DB_PORT", 5433), "Database port")
	dbname := flag.String("database", getEnv("DB_NAME", "cardiofit_analytics"), "Database name")
	user := flag.String("user", getEnv("DB_USER", "cardiofit"), "Database user")
	password := flag.String("password", getEnv("DB_PASSWORD", ""), "Database password")
	sslmode := flag.String("sslmode", getEnv("DB_SSLMODE", "disable"), "SSL mode")
	migrationsPath := flag.String("migrations", "./migrations", "Path to migrations directory")
	seedFile := flag.String("seed", "./migrations/seed_test_data.sql", "Path to seed data file")

	flag.Parse()

	// Create database config
	config := &database.Config{
		Host:     *host,
		Port:     *port,
		Database: *dbname,
		User:     *user,
		Password: *password,
		SSLMode:  *sslmode,
	}

	// Get absolute path for migrations
	absPath, err := filepath.Abs(*migrationsPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for migrations: %v", err)
	}

	// Create migration manager
	manager, err := database.NewMigrationManager(config, absPath)
	if err != nil {
		log.Fatalf("Failed to create migration manager: %v", err)
	}
	defer manager.Close()

	// Execute command
	switch *command {
	case "up":
		if err := manager.Up(); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
	case "down":
		if err := manager.Down(); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
	case "status":
		if err := manager.Status(); err != nil {
			log.Fatalf("Migration status failed: %v", err)
		}
	case "seed":
		// Get absolute path for seed file
		absSeedPath, err := filepath.Abs(*seedFile)
		if err != nil {
			log.Fatalf("Failed to get absolute path for seed file: %v", err)
		}
		if err := manager.Seed(absSeedPath); err != nil {
			log.Fatalf("Seed data load failed: %v", err)
		}
	default:
		log.Fatalf("Unknown command: %s. Valid commands: up, down, status, seed", *command)
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as int or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
