// Package database provides PostgreSQL database connectivity for KB-11.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/config"
)

// DB wraps sql.DB with additional functionality.
type DB struct {
	*sql.DB
	logger *logrus.Entry
	config *config.DatabaseConfig
}

// NewConnection creates a new database connection pool.
func NewConnection(cfg *config.DatabaseConfig, logger *logrus.Entry) (*DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"host":           cfg.Host,
		"port":           cfg.Port,
		"database":       cfg.Name,
		"max_open_conns": cfg.MaxOpenConns,
		"max_idle_conns": cfg.MaxIdleConns,
	}).Info("Database connection established")

	return &DB{
		DB:     db,
		logger: logger,
		config: cfg,
	}, nil
}

// Close closes the database connection pool.
func (db *DB) Close() error {
	db.logger.Info("Closing database connection pool")
	return db.DB.Close()
}

// Health checks the database connection health.
func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}

// Stats returns database connection pool statistics.
func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

// SQLX returns a sqlx.DB wrapper for the underlying connection.
// This is useful for packages that require sqlx functionality.
func (db *DB) SQLX() *sqlx.DB {
	return sqlx.NewDb(db.DB, "postgres")
}

// WithTransaction executes a function within a database transaction.
// Automatically commits on success and rolls back on error.
func (db *DB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			db.logger.WithError(rbErr).Error("Failed to rollback transaction")
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RunMigrations applies database migrations.
// NOTE: In production, use a proper migration tool like golang-migrate.
func (db *DB) RunMigrations(ctx context.Context, migrationsPath string) error {
	db.logger.WithField("path", migrationsPath).Info("Running database migrations")

	// This is a placeholder - in production, use golang-migrate or similar
	// For now, migrations should be applied manually or via docker-compose

	return nil
}

// QueryRowContext wraps sql.DB.QueryRowContext with logging.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := db.DB.QueryRowContext(ctx, query, args...)

	db.logger.WithFields(logrus.Fields{
		"query":    truncateQuery(query),
		"duration": time.Since(start).String(),
	}).Debug("Query executed")

	return row
}

// QueryContext wraps sql.DB.QueryContext with logging.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.DB.QueryContext(ctx, query, args...)

	fields := logrus.Fields{
		"query":    truncateQuery(query),
		"duration": time.Since(start).String(),
	}

	if err != nil {
		db.logger.WithFields(fields).WithError(err).Error("Query failed")
	} else {
		db.logger.WithFields(fields).Debug("Query executed")
	}

	return rows, err
}

// ExecContext wraps sql.DB.ExecContext with logging.
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := db.DB.ExecContext(ctx, query, args...)

	fields := logrus.Fields{
		"query":    truncateQuery(query),
		"duration": time.Since(start).String(),
	}

	if err != nil {
		db.logger.WithFields(fields).WithError(err).Error("Exec failed")
	} else {
		if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			fields["rows_affected"] = rowsAffected
		}
		db.logger.WithFields(fields).Debug("Exec completed")
	}

	return result, err
}

// truncateQuery truncates long queries for logging.
func truncateQuery(query string) string {
	const maxLen = 200
	if len(query) > maxLen {
		return query[:maxLen] + "..."
	}
	return query
}
