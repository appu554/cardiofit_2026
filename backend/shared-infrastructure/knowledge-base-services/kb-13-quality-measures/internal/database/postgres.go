// Package database provides PostgreSQL database connectivity for KB-13.
//
// This package handles connection pooling, migrations, and database operations
// for quality measure calculations, care gaps, and reports.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"

	"kb-13-quality-measures/internal/config"
)

// DB wraps the SQL database connection with logging and convenience methods.
type DB struct {
	*sql.DB
	logger *zap.Logger
}

// New creates a new database connection pool.
func New(cfg *config.DatabaseConfig, logger *zap.Logger) (*DB, error) {
	dsn := cfg.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.Int("max_conns", cfg.MaxConns),
	)

	return &DB{DB: db, logger: logger}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	db.logger.Info("Closing database connection")
	return db.DB.Close()
}

// Health checks database connectivity.
func (db *DB) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

// ExecContext executes a query with logging.
func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := db.DB.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		db.logger.Error("Database query failed",
			zap.String("query", truncateQuery(query)),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	if duration > 100*time.Millisecond {
		db.logger.Warn("Slow query detected",
			zap.String("query", truncateQuery(query)),
			zap.Duration("duration", duration),
		)
	}

	return result, nil
}

// QueryContext executes a query and returns rows with logging.
func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := db.DB.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		db.logger.Error("Database query failed",
			zap.String("query", truncateQuery(query)),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return nil, err
	}

	if duration > 100*time.Millisecond {
		db.logger.Warn("Slow query detected",
			zap.String("query", truncateQuery(query)),
			zap.Duration("duration", duration),
		)
	}

	return rows, nil
}

// QueryRowContext executes a query returning a single row.
func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a new transaction.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, opts)
}

// truncateQuery truncates query for logging (avoid huge queries in logs).
func truncateQuery(query string) string {
	if len(query) > 200 {
		return query[:200] + "..."
	}
	return query
}

// Stats returns database connection pool statistics.
type Stats struct {
	MaxOpenConnections int `json:"max_open_connections"`
	OpenConnections    int `json:"open_connections"`
	InUse              int `json:"in_use"`
	Idle               int `json:"idle"`
	WaitCount          int64 `json:"wait_count"`
	WaitDuration       int64 `json:"wait_duration_ms"`
}

// Stats returns current connection pool statistics.
func (db *DB) Stats() Stats {
	s := db.DB.Stats()
	return Stats{
		MaxOpenConnections: s.MaxOpenConnections,
		OpenConnections:    s.OpenConnections,
		InUse:              s.InUse,
		Idle:               s.Idle,
		WaitCount:          s.WaitCount,
		WaitDuration:       s.WaitDuration.Milliseconds(),
	}
}
