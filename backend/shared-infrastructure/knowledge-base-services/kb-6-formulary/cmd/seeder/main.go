// Package main provides the database seeder for KB-6 Formulary Service.
// It populates the PostgreSQL database with production drug data, PA requirements,
// step therapy rules, and drug alternatives.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// SeedFile represents a SQL seed file to be executed
type SeedFile struct {
	Name     string
	Path     string
	Priority int
}

func main() {
	log.Println("╔══════════════════════════════════════════════════════════════╗")
	log.Println("║       KB-6 Formulary Database Seeder - Production Data       ║")
	log.Println("╚══════════════════════════════════════════════════════════════╝")

	// Build database connection string from environment variables
	dbURL := buildDatabaseURL()
	log.Printf("Connecting to database at %s:%s/%s...",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "kb_formulary"))

	// Connect to database with retry logic
	db, err := connectWithRetry(dbURL, 5, 3*time.Second)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("✓ Connected to database successfully")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Discover and sort seed files
	seedDir := getEnv("SEED_DIR", "./seeds")
	seedFiles, err := discoverSeedFiles(seedDir)
	if err != nil {
		log.Fatalf("❌ Failed to discover seed files: %v", err)
	}

	if len(seedFiles) == 0 {
		log.Printf("⚠️ No seed files found in %s", seedDir)
		return
	}

	log.Printf("Found %d seed files to process", len(seedFiles))

	// Execute each seed file in order
	successCount := 0
	for _, sf := range seedFiles {
		log.Printf("────────────────────────────────────────")
		log.Printf("📄 Processing: %s", sf.Name)

		if err := executeSeedFile(ctx, db, sf); err != nil {
			log.Printf("❌ Failed to execute %s: %v", sf.Name, err)
			// Continue with other files but track failure
			continue
		}

		log.Printf("✓ Successfully executed: %s", sf.Name)
		successCount++
	}

	// Summary
	log.Println("════════════════════════════════════════════════════════════════")
	log.Printf("📊 Seeding Summary:")
	log.Printf("   Total files: %d", len(seedFiles))
	log.Printf("   Successful:  %d", successCount)
	log.Printf("   Failed:      %d", len(seedFiles)-successCount)

	if successCount == len(seedFiles) {
		log.Println("✅ Database seeding completed successfully!")
	} else {
		log.Println("⚠️ Database seeding completed with some failures")
		os.Exit(1)
	}

	// Verify seeded data
	if err := verifySeedData(ctx, db); err != nil {
		log.Printf("⚠️ Verification warning: %v", err)
	}
}

// buildDatabaseURL constructs a PostgreSQL connection string from environment variables
func buildDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getEnv("DB_USER", "kb6_admin"),
		getEnv("DB_PASSWORD", "kb6_secure_password"),
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "kb_formulary"),
	)
}

// connectWithRetry attempts to connect to the database with retry logic
func connectWithRetry(dbURL string, maxRetries int, delay time.Duration) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 0; i < maxRetries; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err != nil {
			log.Printf("Connection attempt %d/%d failed: %v", i+1, maxRetries, err)
			time.Sleep(delay)
			continue
		}

		// Test the connection
		if err = db.Ping(); err != nil {
			log.Printf("Ping attempt %d/%d failed: %v", i+1, maxRetries, err)
			db.Close()
			time.Sleep(delay)
			continue
		}

		// Connection successful
		return db, nil
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}

// discoverSeedFiles finds and sorts SQL files in the seeds directory
func discoverSeedFiles(seedDir string) ([]SeedFile, error) {
	files, err := ioutil.ReadDir(seedDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read seed directory: %w", err)
	}

	var seedFiles []SeedFile
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasSuffix(f.Name(), ".sql") {
			continue
		}

		// Extract priority from filename (e.g., "001_insurance_payers.sql" -> 1)
		priority := extractPriority(f.Name())

		seedFiles = append(seedFiles, SeedFile{
			Name:     f.Name(),
			Path:     filepath.Join(seedDir, f.Name()),
			Priority: priority,
		})
	}

	// Sort by priority (filename prefix)
	sort.Slice(seedFiles, func(i, j int) bool {
		return seedFiles[i].Priority < seedFiles[j].Priority
	})

	return seedFiles, nil
}

// extractPriority extracts numeric prefix from filename for ordering
func extractPriority(filename string) int {
	// Handle filenames like "001_insurance_payers.sql"
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return 999 // Default low priority for non-prefixed files
	}

	var priority int
	_, err := fmt.Sscanf(parts[0], "%d", &priority)
	if err != nil {
		return 999
	}

	return priority
}

// executeSeedFile reads and executes a SQL seed file
func executeSeedFile(ctx context.Context, db *sql.DB, sf SeedFile) error {
	// Read the SQL file
	content, err := ioutil.ReadFile(sf.Path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Execute within a transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute the SQL
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// verifySeedData checks that seeded data was inserted correctly
func verifySeedData(ctx context.Context, db *sql.DB) error {
	log.Println("────────────────────────────────────────")
	log.Println("🔍 Verifying seeded data...")

	verifications := []struct {
		name  string
		query string
	}{
		{"Insurance Payers", "SELECT COUNT(*) FROM insurance_payers"},
		{"Insurance Plans", "SELECT COUNT(*) FROM insurance_plans"},
		{"Formulary Entries", "SELECT COUNT(*) FROM formulary_entries"},
		{"PA Requirements", "SELECT COUNT(*) FROM pa_requirements"},
		{"Step Therapy Rules", "SELECT COUNT(*) FROM step_therapy_rules"},
		{"Drug Alternatives", "SELECT COUNT(*) FROM drug_alternatives"},
	}

	for _, v := range verifications {
		var count int
		if err := db.QueryRowContext(ctx, v.query).Scan(&count); err != nil {
			log.Printf("   ⚠️ %s: query failed - %v", v.name, err)
			continue
		}
		log.Printf("   ✓ %s: %d records", v.name, count)
	}

	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
