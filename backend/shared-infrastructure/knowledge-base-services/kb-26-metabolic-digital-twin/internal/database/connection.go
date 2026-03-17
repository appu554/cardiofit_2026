package database

import (
	"fmt"

	"kb-26-metabolic-digital-twin/internal/config"

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

// NewConnection initialises a PostgreSQL connection and configures pooling.
// AutoMigrate is NOT called here — the caller (main.go) is responsible for
// running AutoMigrate after all model packages have been imported.
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

	log.Info("Database connection established",
		zap.Int("max_open_conns", cfg.Database.MaxConnections),
	)

	return &Database{DB: db, logger: log}, nil
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
