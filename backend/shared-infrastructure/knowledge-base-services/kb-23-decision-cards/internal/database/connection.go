package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/models"
)

type Database struct {
	DB *gorm.DB
}

func NewConnection(cfg *config.Config, log *zap.Logger) (*Database, error) {
	logLevel := logger.Info
	if cfg.IsProduction() {
		logLevel = logger.Warn
	}

	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DBMaxConnections)
	sqlDB.SetMaxIdleConns(cfg.DBMaxConnections / 2)
	sqlDB.SetConnMaxLifetime(cfg.DBConnMaxLife)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	log.Info("database connected",
		zap.String("url", maskDSN(cfg.DatabaseURL)),
		zap.Int("max_connections", cfg.DBMaxConnections),
	)

	return &Database{DB: db}, nil
}

// AutoMigrate runs GORM auto-migration for all 8 KB-23 tables.
func (d *Database) AutoMigrate() error {
	return d.DB.AutoMigrate(
		&models.DecisionCard{},
		&models.CardRecommendation{},
		&models.CardTemplate{},
		&models.SummaryFragment{},
		&models.MCUGateHistory{},
		&models.CompositeCardSignal{},
		&models.HypoglycaemiaAlert{},
		&models.TreatmentPerturbation{},
	)
}

func (d *Database) Health() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

func maskDSN(dsn string) string {
	if len(dsn) > 30 {
		return dsn[:20] + "***"
	}
	return "***"
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
