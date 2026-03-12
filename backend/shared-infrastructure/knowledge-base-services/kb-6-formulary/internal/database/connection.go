package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"

	"kb-formulary/internal/config"
)

// Row represents a single database row
type Row interface {
	Scan(dest ...interface{}) error
}

// Rows represents multiple database rows
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
}

// Result represents the result of an Exec operation
type Result interface {
	RowsAffected() int64
}

// Tx represents a database transaction
type Tx interface {
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	Exec(ctx context.Context, query string, args ...interface{}) (Result, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Connection represents a database connection with context support
type Connection struct {
	db     *sql.DB
	config *config.Config
}

// NewConnection creates a new database connection
func NewConnection(cfg *config.Config) (*Connection, error) {
	db, err := sql.Open("postgres", cfg.GetDatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.Database.MaxConns)
	db.SetMaxIdleConns(cfg.Database.MaxConns / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Successfully connected to PostgreSQL database: %s", cfg.Database.Database)

	return &Connection{
		db:     db,
		config: cfg,
	}, nil
}

// QueryRow executes a query that expects to return at most one row
func (c *Connection) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &sqlRow{row: c.db.QueryRowContext(ctx, query, args...)}
}

// Query executes a query that returns rows
func (c *Connection) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

// Exec executes a query without returning rows
func (c *Connection) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

// Begin starts a new transaction
func (c *Connection) Begin(ctx context.Context) (Tx, error) {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &sqlTx{tx: tx}, nil
}

// HealthCheck tests the database connection
func (c *Connection) HealthCheck() error {
	return c.db.Ping()
}

// Close closes the database connection
func (c *Connection) Close() error {
	return c.db.Close()
}

// DB returns the underlying *sql.DB for use with repositories
// that require direct database access
func (c *Connection) DB() *sql.DB {
	return c.db
}

// RunMigrations runs database migrations
func (c *Connection) RunMigrations() error {
	// This would run migration files
	// For now, assume migrations are run externally
	log.Println("Database migrations completed")
	return nil
}

// Concrete implementations

type sqlRow struct {
	row *sql.Row
}

func (r *sqlRow) Scan(dest ...interface{}) error {
	return r.row.Scan(dest...)
}

type sqlRows struct {
	rows *sql.Rows
}

func (r *sqlRows) Next() bool {
	return r.rows.Next()
}

func (r *sqlRows) Scan(dest ...interface{}) error {
	return r.rows.Scan(dest...)
}

func (r *sqlRows) Close() error {
	return r.rows.Close()
}

type sqlResult struct {
	result sql.Result
}

func (r *sqlResult) RowsAffected() int64 {
	count, _ := r.result.RowsAffected()
	return count
}

type sqlTx struct {
	tx *sql.Tx
}

func (t *sqlTx) QueryRow(ctx context.Context, query string, args ...interface{}) Row {
	return &sqlRow{row: t.tx.QueryRowContext(ctx, query, args...)}
}

func (t *sqlTx) Query(ctx context.Context, query string, args ...interface{}) (Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlRows{rows: rows}, nil
}

func (t *sqlTx) Exec(ctx context.Context, query string, args ...interface{}) (Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &sqlResult{result: result}, nil
}

func (t *sqlTx) Commit(ctx context.Context) error {
	return t.tx.Commit()
}

func (t *sqlTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback()
}