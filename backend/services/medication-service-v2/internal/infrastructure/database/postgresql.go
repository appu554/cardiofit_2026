package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// PostgreSQL wraps the database connection and provides helper methods
type PostgreSQL struct {
	DB     *sqlx.DB
	logger *zap.Logger
}

// NewPostgreSQL creates a new PostgreSQL database connection
func NewPostgreSQL(databaseURL string) (*PostgreSQL, error) {
	logger, _ := zap.NewProduction()

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(30 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Successfully connected to PostgreSQL database")

	return &PostgreSQL{
		DB:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (pg *PostgreSQL) Close() error {
	if pg.DB != nil {
		return pg.DB.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (pg *PostgreSQL) Ping(ctx context.Context) error {
	return pg.DB.PingContext(ctx)
}

// BeginTx starts a new transaction
func (pg *PostgreSQL) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return pg.DB.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
}

// WithTransaction executes a function within a database transaction
func (pg *PostgreSQL) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := pg.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			pg.logger.Error("Failed to rollback transaction", zap.Error(rollbackErr))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetStats returns database connection statistics
func (pg *PostgreSQL) GetStats() sql.DBStats {
	return pg.DB.Stats()
}

// Health checks database health for monitoring
type DatabaseHealth struct {
	Status          string         `json:"status"`
	ConnectionStats sql.DBStats    `json:"connection_stats"`
	ResponseTime    time.Duration  `json:"response_time"`
	Error           string         `json:"error,omitempty"`
}

// HealthCheck performs a comprehensive health check
func (pg *PostgreSQL) HealthCheck(ctx context.Context) *DatabaseHealth {
	startTime := time.Now()
	
	health := &DatabaseHealth{
		Status: "healthy",
		ConnectionStats: pg.GetStats(),
	}

	// Test basic connectivity
	if err := pg.Ping(ctx); err != nil {
		health.Status = "unhealthy"
		health.Error = err.Error()
		health.ResponseTime = time.Since(startTime)
		return health
	}

	// Test a simple query
	var result int
	if err := pg.DB.GetContext(ctx, &result, "SELECT 1"); err != nil {
		health.Status = "degraded"
		health.Error = fmt.Sprintf("query test failed: %v", err)
	}

	health.ResponseTime = time.Since(startTime)
	return health
}

// Migration support is implemented in migrations.go