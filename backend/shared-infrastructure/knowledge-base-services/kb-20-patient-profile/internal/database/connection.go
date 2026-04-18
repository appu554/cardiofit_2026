package database

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-patient-profile/internal/config"
	"kb-patient-profile/internal/models"
)

// Database wraps the GORM database connection.
type Database struct {
	DB *gorm.DB
}

// NewConnection creates a PostgreSQL connection with GORM, configures the pool,
// and runs auto-migrations for all KB-20 models.
func NewConnection(cfg *config.Config, zapLogger *zap.Logger) (*Database, error) {
	var gormLogger logger.Interface
	if cfg.IsDevelopment() {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.Database.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxConnections / 2)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return &Database{DB: db}, nil
}

// GetDB returns the underlying *gorm.DB for use by services that need direct access.
func (d *Database) GetDB() *gorm.DB {
	return d.DB
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.PatientProfile{},
		&models.LabEntry{},
		&models.MedicationState{},
		&models.ContextModifier{},
		&models.AdverseReactionProfile{},
		&models.EventOutboxEntry{},
		&models.FHIRSyncLog{},
		&models.ProtocolState{},
		&models.ProtocolMetrics{},
		// Phase 8 P8-5: safety_events audit table feeds the
		// summary-context confounder flags.
		&models.SafetyEvent{},
		// V4-7: phenotype stability history tables.
		&models.ClusterAssignmentRecord{},
		&models.ClusterTransitionRecord{},
		// Gap 17: Care Transition Bridge tables.
		&models.CareTransition{},
		&models.DischargeMedication{},
		&models.TransitionMilestone{},
		&models.TransitionOutcome{},
	)
}

// Close closes the underlying database connection.
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB for closing: %w", err)
	}
	return sqlDB.Close()
}

// HealthCheck pings the database to verify connectivity.
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}
	return sqlDB.Ping()
}
