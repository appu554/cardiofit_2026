// Package database provides PostgreSQL connection management for KB-12
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-12-ordersets-careplans/internal/config"
)

// Connection wraps the GORM database connection with health check capabilities
type Connection struct {
	DB     *gorm.DB
	config *config.DatabaseConfig
	log    *logrus.Entry
}

// NewConnection creates a new database connection
func NewConnection(cfg *config.DatabaseConfig) (*Connection, error) {
	log := logrus.WithField("component", "database")

	// Configure GORM logger based on environment
	var gormLogger logger.Interface
	if cfg.SSLMode == "disable" { // development mode indicator
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	dsn := cfg.GetDSN()
	log.WithField("host", cfg.Host).WithField("database", cfg.Database).Info("Connecting to PostgreSQL")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   gormLogger,
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: false,
		SkipDefaultTransaction:                   false,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL connection for pool configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Successfully connected to PostgreSQL")

	return &Connection{
		DB:     db,
		config: cfg,
		log:    log,
	}, nil
}

// AutoMigrate runs database migrations for all models
func (c *Connection) AutoMigrate(models ...interface{}) error {
	c.log.Info("Running database migrations")
	return c.DB.AutoMigrate(models...)
}

// Health performs a database health check
func (c *Connection) Health(ctx context.Context) error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.PingContext(ctx)
}

// HealthCheck returns detailed health status
func (c *Connection) HealthCheck(ctx context.Context) *HealthStatus {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return &HealthStatus{
			Status:  "unhealthy",
			Error:   err.Error(),
			Latency: 0,
		}
	}

	start := time.Now()
	err = sqlDB.PingContext(ctx)
	latency := time.Since(start)

	if err != nil {
		return &HealthStatus{
			Status:  "unhealthy",
			Error:   err.Error(),
			Latency: latency,
		}
	}

	stats := sqlDB.Stats()
	return &HealthStatus{
		Status:      "healthy",
		Latency:     latency,
		OpenConns:   stats.OpenConnections,
		InUse:       stats.InUse,
		Idle:        stats.Idle,
		WaitCount:   stats.WaitCount,
		WaitDuration: stats.WaitDuration,
		MaxOpenConns: stats.MaxOpenConnections,
	}
}

// HealthStatus contains detailed database health information
type HealthStatus struct {
	Status       string        `json:"status"`
	Error        string        `json:"error,omitempty"`
	Latency      time.Duration `json:"latency_ms"`
	OpenConns    int           `json:"open_connections"`
	InUse        int           `json:"in_use"`
	Idle         int           `json:"idle"`
	WaitCount    int64         `json:"wait_count"`
	WaitDuration time.Duration `json:"wait_duration_ms"`
	MaxOpenConns int           `json:"max_open_connections"`
}

// Close closes the database connection
func (c *Connection) Close() error {
	c.log.Info("Closing database connection")
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	return sqlDB.Close()
}

// Transaction wraps a function in a database transaction
func (c *Connection) Transaction(fn func(tx *gorm.DB) error) error {
	return c.DB.Transaction(fn)
}

// WithContext returns a new DB with context
func (c *Connection) WithContext(ctx context.Context) *gorm.DB {
	return c.DB.WithContext(ctx)
}

// GetDB returns the underlying GORM DB instance
func (c *Connection) GetDB() *gorm.DB {
	return c.DB
}

// Model provides a convenient wrapper for GORM Model
func (c *Connection) Model(value interface{}) *gorm.DB {
	return c.DB.Model(value)
}

// Create inserts a new record
func (c *Connection) Create(value interface{}) *gorm.DB {
	return c.DB.Create(value)
}

// Save saves a record (insert or update)
func (c *Connection) Save(value interface{}) *gorm.DB {
	return c.DB.Save(value)
}

// Delete deletes records matching the given conditions
func (c *Connection) Delete(value interface{}, conds ...interface{}) *gorm.DB {
	return c.DB.Delete(value, conds...)
}

// First finds the first record matching conditions
func (c *Connection) First(dest interface{}, conds ...interface{}) *gorm.DB {
	return c.DB.First(dest, conds...)
}

// Find finds all records matching conditions
func (c *Connection) Find(dest interface{}, conds ...interface{}) *gorm.DB {
	return c.DB.Find(dest, conds...)
}

// Where adds a WHERE clause
func (c *Connection) Where(query interface{}, args ...interface{}) *gorm.DB {
	return c.DB.Where(query, args...)
}

// Preload preloads associations
func (c *Connection) Preload(query string, args ...interface{}) *gorm.DB {
	return c.DB.Preload(query, args...)
}

// Exec executes raw SQL
func (c *Connection) Exec(sql string, values ...interface{}) *gorm.DB {
	return c.DB.Exec(sql, values...)
}

// Raw executes raw SQL returning rows
func (c *Connection) Raw(sql string, values ...interface{}) *gorm.DB {
	return c.DB.Raw(sql, values...)
}
