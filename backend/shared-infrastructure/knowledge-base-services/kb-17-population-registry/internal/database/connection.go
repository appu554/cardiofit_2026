// Package database provides database connection and repository implementations
package database

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-17-population-registry/internal/config"
	"kb-17-population-registry/internal/models"
)

// Connection holds the database connection
type Connection struct {
	DB     *gorm.DB
	logger *logrus.Entry
}

// NewConnection creates a new database connection
func NewConnection(cfg *config.DatabaseConfig, log *logrus.Entry) (*Connection, error) {
	log = log.WithField("component", "database")

	// Configure GORM logger
	gormLogger := logger.New(
		log,
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	// Open connection
	db, err := gorm.Open(postgres.Open(cfg.URL), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection established")

	return &Connection{
		DB:     db,
		logger: log,
	}, nil
}

// AutoMigrate runs auto-migration for all models
func (c *Connection) AutoMigrate() error {
	c.logger.Info("Running database migrations...")

	err := c.DB.AutoMigrate(
		&models.Registry{},
		&models.RegistryPatient{},
		&models.EnrollmentHistory{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	// Create indexes that GORM doesn't handle automatically
	c.createCustomIndexes()

	c.logger.Info("Database migrations completed")
	return nil
}

// createCustomIndexes creates custom indexes
func (c *Connection) createCustomIndexes() {
	// Composite index for patient registry lookup
	c.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_registry_patients_code_patient
		ON registry_patients(registry_code, patient_id)`)

	// Index for care gap queries
	c.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_registry_patients_care_gaps
		ON registry_patients USING GIN (care_gaps)`)

	// Index for metrics queries
	c.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_registry_patients_metrics
		ON registry_patients USING GIN (metrics)`)

	// Index for history lookups
	c.DB.Exec(`CREATE INDEX IF NOT EXISTS idx_enrollment_history_enrollment
		ON enrollment_history(enrollment_id, created_at DESC)`)
}

// Close closes the database connection
func (c *Connection) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health checks database health
func (c *Connection) Health() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Transaction executes a function within a transaction
func (c *Connection) Transaction(fn func(tx *gorm.DB) error) error {
	return c.DB.Transaction(fn)
}
