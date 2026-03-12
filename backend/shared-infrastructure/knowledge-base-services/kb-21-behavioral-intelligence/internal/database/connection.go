package database

import (
	"fmt"
	"kb-21-behavioral-intelligence/internal/config"
	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps GORM with health-check capability.
type Database struct {
	DB     *gorm.DB
	logger *zap.Logger
}

// NewConnection initialises a PostgreSQL connection, configures pooling,
// and runs auto-migration for all KB-21 domain models.
func NewConnection(cfg *config.Config, log *zap.Logger) (*Database, error) {
	gormLogLevel := logger.Error
	if cfg.IsDevelopment() {
		gormLogLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseDSN()), &gorm.Config{
		Logger:                                   logger.Default.LogMode(gormLogLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
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

	log.Info("Database connection established",
		zap.Int("max_open_conns", cfg.Database.MaxConnections),
	)

	return &Database{DB: db, logger: log}, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.InteractionEvent{},
		&models.AdherenceState{},
		&models.EngagementProfile{},
		&models.OutcomeCorrelation{},
		&models.QuestionTelemetry{},
		&models.NudgeRecord{},
		&models.CohortSnapshot{},
		&models.DietarySignal{},
		&models.BarrierDetection{},
	)
}

// HealthCheck verifies the database connection is alive.
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// Close terminates the database connection pool.
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
