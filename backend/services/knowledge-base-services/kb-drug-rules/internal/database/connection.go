package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewConnection creates a new database connection
func NewConnection(databaseURL string) (*gorm.DB, error) {
	return newPostgreSQLConnection(databaseURL)
}

// isSupabaseURL checks if the URL is a Supabase URL
func isSupabaseURL(databaseURL string) bool {
	return strings.Contains(databaseURL, "supabase.co")
}

// newPostgreSQLConnection creates a regular PostgreSQL connection
func newPostgreSQLConnection(databaseURL string) (*gorm.DB, error) {
	// Configure GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(databaseURL), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB for configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// RunMigrations runs database migrations
func RunMigrations(databaseURL string) error {
	// For now, we'll use GORM's AutoMigrate
	// In production, you should use proper migration files
	
	db, err := NewConnection(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect for migrations: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB for migrations: %w", err)
	}
	defer sqlDB.Close()

	// Auto-migrate basic tables
	// Note: In production, use proper migration files
	if err := db.Exec(`
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE EXTENSION IF NOT EXISTS "pg_trgm";
	`).Error; err != nil {
		// Ignore errors for extensions that might already exist
	}

	return nil
}

// RunMigrationsWithFiles runs migrations from migration files
func RunMigrationsWithFiles(databaseURL, migrationsPath string) error {
	// Get underlying SQL DB
	db, err := NewConnection(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect for migrations: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}
	defer sqlDB.Close()

	// Create migration driver
	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// HealthCheck checks database health
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// GetStats returns database statistics
func GetStats(db *gorm.DB) (map[string]interface{}, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"max_open_connections":     stats.MaxOpenConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                  stats.InUse,
		"idle":                    stats.Idle,
		"wait_count":              stats.WaitCount,
		"wait_duration":           stats.WaitDuration,
		"max_idle_closed":         stats.MaxIdleClosed,
		"max_idle_time_closed":    stats.MaxIdleTimeClosed,
		"max_lifetime_closed":     stats.MaxLifetimeClosed,
		"provider":                "postgresql",
	}, nil
}

// RunKB1Migration runs KB-1 specific database migrations
func RunKB1Migration(db *gorm.DB, logger interface{}) error {
	// Enable required extensions
	extensions := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE EXTENSION IF NOT EXISTS "pg_trgm"`,
		`CREATE EXTENSION IF NOT EXISTS "btree_gin"`,
	}

	for _, ext := range extensions {
		db.Exec(ext) // Ignore errors - extensions may already exist
	}

	// Create materialized view for active dosing rules (KB-1 performance optimization)
	materializedView := `
		CREATE MATERIALIZED VIEW IF NOT EXISTS active_dosing_rules AS
		SELECT
			id as rule_id,
			drug_id as drug_code,
			COALESCE(content->>'drug_name', drug_id) as drug_name,
			version as semantic_version,
			content as compiled_json,
			content_sha as checksum,
			content->'adjustments' as adjustments,
			content->'titration_schedule' as titration_schedule,
			content->'population_rules' as population_rules,
			jsonb_build_object(
				'signed_by', signed_by,
				'clinical_reviewer', clinical_reviewer,
				'clinical_review_date', clinical_review_date
			) as provenance,
			created_at,
			updated_at
		FROM drug_rule_packs
		WHERE signature_valid = true
		ORDER BY drug_id, version DESC
	`
	db.Exec(materializedView)

	// Create unique index on materialized view
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_active_dosing_rules_drug ON active_dosing_rules(drug_code, semantic_version)")

	// Create refresh function for materialized view
	refreshFunction := `
		CREATE OR REPLACE FUNCTION refresh_active_dosing_rules()
		RETURNS void AS $$
		BEGIN
			REFRESH MATERIALIZED VIEW CONCURRENTLY active_dosing_rules;
		EXCEPTION WHEN OTHERS THEN
			REFRESH MATERIALIZED VIEW active_dosing_rules;
		END;
		$$ LANGUAGE plpgsql
	`
	db.Exec(refreshFunction)

	return nil
}

// AutoMigrate runs GORM auto-migration for all models
func AutoMigrate(db *gorm.DB) error {
	// Enable extensions
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		// Ignore error if extension already exists
	}
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"pg_trgm\"").Error; err != nil {
		// Ignore error if extension already exists
	}

	// Create drug_rule_packs table
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS drug_rule_packs (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			drug_id VARCHAR(255) NOT NULL,
			version VARCHAR(50) NOT NULL,
			content_sha VARCHAR(64) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			signed_by VARCHAR(255) NOT NULL,
			signature_valid BOOLEAN NOT NULL DEFAULT false,
			clinical_reviewer VARCHAR(255),
			clinical_review_date TIMESTAMP WITH TIME ZONE,
			regions TEXT[] NOT NULL DEFAULT '{}',
			content JSONB NOT NULL,
			signature TEXT,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(drug_id, version)
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create drug_rule_packs table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_drug_id ON drug_rule_packs(drug_id)",
		"CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_version ON drug_rule_packs(version)",
		"CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_created_at ON drug_rule_packs(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_regions ON drug_rule_packs USING GIN(regions)",
		"CREATE INDEX IF NOT EXISTS idx_drug_rule_packs_content ON drug_rule_packs USING GIN(content)",
	}

	for _, index := range indexes {
		if err := db.Exec(index).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
