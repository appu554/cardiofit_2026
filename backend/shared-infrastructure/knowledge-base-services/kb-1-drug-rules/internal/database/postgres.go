package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Config holds PostgreSQL connection configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig returns default database configuration
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            5481,
		User:            "kb1_user",
		Password:        "kb1_password",
		Database:        "kb1_drug_rules",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// ConnectionString builds the PostgreSQL connection string
func (c Config) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode,
	)
}

// DB wraps the sql.DB with additional functionality
type DB struct {
	*sql.DB
	log *logrus.Entry
}

// Connect establishes a connection to PostgreSQL
func Connect(cfg Config, log *logrus.Entry) (*DB, error) {
	logger := log.WithField("component", "database")

	logger.WithFields(logrus.Fields{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"database": cfg.Database,
		"user":     cfg.User,
	}).Info("Connecting to PostgreSQL")

	db, err := sql.Open("postgres", cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL")

	return &DB{DB: db, log: logger}, nil
}

// ConnectWithURL connects using a connection URL string
func ConnectWithURL(url string, log *logrus.Entry) (*DB, error) {
	logger := log.WithField("component", "database")

	logger.Info("Connecting to PostgreSQL via URL")

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool with defaults
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL")

	return &DB{DB: db, log: logger}, nil
}

// Health checks database connection health
func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}

// Stats returns database connection pool statistics
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// Close closes the database connection
func (db *DB) Close() error {
	db.log.Info("Closing database connection")
	return db.DB.Close()
}

// WithTransaction executes a function within a transaction
func (db *DB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// MigrateUp runs database migrations
func (db *DB) MigrateUp(ctx context.Context, migrationsPath string) error {
	db.log.WithField("path", migrationsPath).Info("Running database migrations")

	// For production, use a migration tool like golang-migrate
	// This is a simplified version for the initial schema
	// In production, you would use:
	// migrate -path migrations -database "postgresql://..." up

	return nil
}

// TableExists checks if a table exists in the database
func (db *DB) TableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`
	err := db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	return exists, nil
}

// GetDrugCount returns the total number of drugs in the database
func (db *DB) GetDrugCount(ctx context.Context) (int, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM drug_rules").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get drug count: %w", err)
	}
	return count, nil
}

// GetDrugCountByJurisdiction returns drug counts grouped by jurisdiction
func (db *DB) GetDrugCountByJurisdiction(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT jurisdiction, COUNT(*) as count
		FROM drug_rules
		GROUP BY jurisdiction
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query drug counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var jurisdiction string
		var count int
		if err := rows.Scan(&jurisdiction, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		counts[jurisdiction] = count
	}

	return counts, rows.Err()
}
