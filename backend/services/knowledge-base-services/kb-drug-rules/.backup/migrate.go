package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("❌ DATABASE_URL environment variable not set")
	}

	fmt.Println("🔄 Running Database Migration...")
	fmt.Printf("📍 Database: %s\n", maskDatabaseURL(databaseURL))

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}
	fmt.Println("✅ Database connection successful")

	// Read migration file
	migrationFile := "migrations/004_enhance_drug_rules_toml_support_fixed.sql"
	migrationSQL, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("❌ Failed to read migration file: %v", err)
	}
	fmt.Printf("📄 Migration file loaded: %s\n", migrationFile)

	// Execute migration
	fmt.Println("🚀 Executing migration...")
	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("❌ Migration failed: %v", err)
	}

	fmt.Println("✅ Migration completed successfully!")
	
	// Verify migration
	fmt.Println("🔍 Verifying migration...")
	if err := verifyMigration(db); err != nil {
		log.Printf("⚠️  Migration verification failed: %v", err)
	} else {
		fmt.Println("✅ Migration verification successful!")
	}

	fmt.Println("🎉 Database is ready for TOML support!")
}

// maskDatabaseURL masks sensitive information in database URL
func maskDatabaseURL(url string) string {
	// Simple masking - in production you'd want more sophisticated masking
	if len(url) > 20 {
		return url[:10] + "***" + url[len(url)-7:]
	}
	return "***"
}

// verifyMigration checks if the migration was applied correctly
func verifyMigration(db *sql.DB) error {
	// Check if new columns exist
	columns := []string{
		"original_format",
		"toml_content", 
		"version_history",
		"deployment_status",
		"tags",
	}

	for _, column := range columns {
		var exists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'drug_rule_packs' 
				AND column_name = $1
			)
		`
		err := db.QueryRow(query, column).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check column %s: %w", column, err)
		}
		if !exists {
			return fmt.Errorf("column %s was not created", column)
		}
		fmt.Printf("   ✅ Column '%s' exists\n", column)
	}

	// Check if snapshots table exists
	var tableExists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'drug_rule_snapshots'
		)
	`
	err := db.QueryRow(query).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check snapshots table: %w", err)
	}
	if !tableExists {
		return fmt.Errorf("drug_rule_snapshots table was not created")
	}
	fmt.Println("   ✅ Table 'drug_rule_snapshots' exists")

	// Check if indexes exist
	indexes := []string{
		"idx_drug_rule_packs_format",
		"idx_drug_rule_packs_deployment",
		"idx_drug_rule_snapshots_drug_version",
	}

	for _, index := range indexes {
		var indexExists bool
		query := `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes 
				WHERE indexname = $1
			)
		`
		err := db.QueryRow(query, index).Scan(&indexExists)
		if err != nil {
			return fmt.Errorf("failed to check index %s: %w", index, err)
		}
		if !indexExists {
			return fmt.Errorf("index %s was not created", index)
		}
		fmt.Printf("   ✅ Index '%s' exists\n", index)
	}

	return nil
}
