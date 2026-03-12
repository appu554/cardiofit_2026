package database

import (
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

// MigrationDirection represents the direction of migration (up or down)
type MigrationDirection string

const (
	MigrationUp   MigrationDirection = "up"
	MigrationDown MigrationDirection = "down"
)

// Migration represents a single migration file
type Migration struct {
	Version   string
	Direction MigrationDirection
	Filename  string
	SQL       string
}

// MigrationManager handles database schema migrations
type MigrationManager struct {
	db             *sql.DB
	migrationsPath string
	logger         *log.Logger
}

// Config holds database connection configuration
type Config struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(config *Config, migrationsPath string) (*MigrationManager, error) {
	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		config.SSLMode,
	)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger := log.New(os.Stdout, "[MIGRATION] ", log.LstdFlags)

	return &MigrationManager{
		db:             db,
		migrationsPath: migrationsPath,
		logger:         logger,
	}, nil
}

// Close closes the database connection
func (m *MigrationManager) Close() error {
	return m.db.Close()
}

// ensureMigrationTable creates the migration tracking table if it doesn't exist
func (m *MigrationManager) ensureMigrationTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW(),
			direction VARCHAR(10) NOT NULL,
			success BOOLEAN NOT NULL DEFAULT TRUE,
			error_message TEXT
		);
	`

	_, err := m.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	return nil
}

// loadMigrations loads all migration files from the migrations directory
func (m *MigrationManager) loadMigrations() ([]Migration, error) {
	files, err := ioutil.ReadDir(m.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	migrations := []Migration{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		// Parse migration files (format: 001_name.up.sql or 001_name.down.sql)
		if !strings.HasSuffix(filename, ".sql") {
			continue
		}

		var direction MigrationDirection
		var version string

		if strings.HasSuffix(filename, ".up.sql") {
			direction = MigrationUp
			version = strings.TrimSuffix(filename, ".up.sql")
		} else if strings.HasSuffix(filename, ".down.sql") {
			direction = MigrationDown
			version = strings.TrimSuffix(filename, ".down.sql")
		} else {
			continue
		}

		// Extract version number (first part before underscore)
		parts := strings.Split(version, "_")
		if len(parts) < 2 {
			continue
		}
		version = parts[0]

		// Read SQL content
		content, err := ioutil.ReadFile(filepath.Join(m.migrationsPath, filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}

		migrations = append(migrations, Migration{
			Version:   version,
			Direction: direction,
			Filename:  filename,
			SQL:       string(content),
		})
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getAppliedMigrations returns a set of applied migration versions
func (m *MigrationManager) getAppliedMigrations() (map[string]bool, error) {
	query := `SELECT version FROM schema_migrations WHERE success = true ORDER BY version`

	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)

	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	return applied, nil
}

// recordMigration records a migration execution in the database
func (m *MigrationManager) recordMigration(version string, direction MigrationDirection, success bool, errorMsg string) error {
	query := `
		INSERT INTO schema_migrations (version, applied_at, direction, success, error_message)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (version) DO UPDATE
		SET applied_at = EXCLUDED.applied_at,
		    direction = EXCLUDED.direction,
		    success = EXCLUDED.success,
		    error_message = EXCLUDED.error_message
	`

	_, err := m.db.Exec(query, version, time.Now(), string(direction), success, errorMsg)
	return err
}

// Up runs all pending migrations
func (m *MigrationManager) Up() error {
	m.logger.Println("Running migrations UP...")

	// Ensure migration tracking table exists
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	// Load all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	// Get already applied migrations
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Filter for UP migrations that haven't been applied
	pendingMigrations := []Migration{}
	for _, migration := range migrations {
		if migration.Direction == MigrationUp && !applied[migration.Version] {
			pendingMigrations = append(pendingMigrations, migration)
		}
	}

	if len(pendingMigrations) == 0 {
		m.logger.Println("No pending migrations to apply")
		return nil
	}

	// Apply each pending migration
	for _, migration := range pendingMigrations {
		m.logger.Printf("Applying migration: %s", migration.Filename)

		// Begin transaction
		tx, err := m.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// Execute migration SQL
		if _, err := tx.Exec(migration.SQL); err != nil {
			tx.Rollback()
			m.recordMigration(migration.Version, migration.Direction, false, err.Error())
			return fmt.Errorf("migration %s failed: %w", migration.Filename, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", migration.Filename, err)
		}

		// Record successful migration
		if err := m.recordMigration(migration.Version, migration.Direction, true, ""); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Filename, err)
		}

		m.logger.Printf("✓ Successfully applied migration: %s", migration.Filename)
	}

	m.logger.Printf("✓ All migrations completed successfully (%d applied)", len(pendingMigrations))
	return nil
}

// Down rolls back the last migration
func (m *MigrationManager) Down() error {
	m.logger.Println("Rolling back last migration...")

	// Ensure migration tracking table exists
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	// Get the last applied migration
	query := `SELECT version FROM schema_migrations WHERE success = true ORDER BY version DESC LIMIT 1`

	var lastVersion string
	err := m.db.QueryRow(query).Scan(&lastVersion)
	if err == sql.ErrNoRows {
		m.logger.Println("No migrations to rollback")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get last applied migration: %w", err)
	}

	// Load all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	// Find the DOWN migration for the last version
	var downMigration *Migration
	for _, migration := range migrations {
		if migration.Version == lastVersion && migration.Direction == MigrationDown {
			downMigration = &migration
			break
		}
	}

	if downMigration == nil {
		return fmt.Errorf("down migration not found for version %s", lastVersion)
	}

	m.logger.Printf("Rolling back migration: %s", downMigration.Filename)

	// Begin transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute rollback SQL
	if _, err := tx.Exec(downMigration.SQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("rollback %s failed: %w", downMigration.Filename, err)
	}

	// Delete migration record
	deleteQuery := `DELETE FROM schema_migrations WHERE version = $1`
	if _, err := tx.Exec(deleteQuery, lastVersion); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete migration record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback: %w", err)
	}

	m.logger.Printf("✓ Successfully rolled back migration: %s", downMigration.Filename)
	return nil
}

// Status prints the current migration status
func (m *MigrationManager) Status() error {
	m.logger.Println("Migration Status:")

	// Ensure migration tracking table exists
	if err := m.ensureMigrationTable(); err != nil {
		return err
	}

	// Load all migrations
	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	// Group migrations by version
	migrationsByVersion := make(map[string][]Migration)
	for _, migration := range migrations {
		migrationsByVersion[migration.Version] = append(migrationsByVersion[migration.Version], migration)
	}

	// Get sorted unique versions
	versions := []string{}
	for version := range migrationsByVersion {
		versions = append(versions, version)
	}
	sort.Strings(versions)

	// Print status for each version
	fmt.Println("\nVersion | Status   | UP File                                | DOWN File")
	fmt.Println("--------|----------|----------------------------------------|----------------------------------------")

	for _, version := range versions {
		status := "Pending"
		if applied[version] {
			status = "Applied"
		}

		upFile := "-"
		downFile := "-"

		for _, migration := range migrationsByVersion[version] {
			if migration.Direction == MigrationUp {
				upFile = migration.Filename
			} else {
				downFile = migration.Filename
			}
		}

		fmt.Printf("%-7s | %-8s | %-38s | %-38s\n", version, status, upFile, downFile)
	}

	fmt.Println()
	return nil
}

// Seed loads seed data from a SQL file
func (m *MigrationManager) Seed(seedFile string) error {
	m.logger.Printf("Loading seed data from: %s", seedFile)

	// Read seed file
	content, err := ioutil.ReadFile(seedFile)
	if err != nil {
		return fmt.Errorf("failed to read seed file: %w", err)
	}

	// Begin transaction
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Execute seed SQL
	if _, err := tx.Exec(string(content)); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to execute seed data: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seed data: %w", err)
	}

	m.logger.Println("✓ Seed data loaded successfully")
	return nil
}
