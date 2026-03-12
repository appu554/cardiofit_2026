// Package database provides database connection and management for KB-16
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-16-lab-interpretation/internal/config"
	"kb-16-lab-interpretation/pkg/types"
)

// DB holds database connections
type DB struct {
	Postgres *gorm.DB
	Redis    *redis.Client
	log      *logrus.Entry
}

// New creates a new database connection manager
func New(cfg *config.Config, log *logrus.Entry) (*DB, error) {
	db := &DB{
		log: log.WithField("component", "database"),
	}

	// Connect to PostgreSQL
	if err := db.connectPostgres(cfg); err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Connect to Redis (optional)
	if cfg.Redis.Enabled {
		if err := db.connectRedis(cfg); err != nil {
			db.log.WithError(err).Warn("Failed to connect to Redis, caching disabled")
		}
	}

	return db, nil
}

// connectPostgres establishes connection to PostgreSQL
func (db *DB) connectPostgres(cfg *config.Config) error {
	dsn := cfg.GetDatabaseURL()

	// Configure GORM logger
	logLevel := logger.Silent
	if cfg.IsDevelopment() {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	pgDB, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := pgDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	db.Postgres = pgDB
	db.log.Info("Connected to PostgreSQL")

	return nil
}

// connectRedis establishes connection to Redis
func (db *DB) connectRedis(cfg *config.Config) error {
	opt, err := redis.ParseURL(cfg.GetRedisURL())
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	opt.PoolSize = cfg.Redis.PoolSize
	opt.MinIdleConns = cfg.Redis.MinIdleConns

	client := redis.NewClient(opt)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	db.Redis = client
	db.log.Info("Connected to Redis")

	return nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	db.log.Info("Running database migrations...")

	// Auto-migrate core models
	err := db.Postgres.AutoMigrate(
		&types.LabResult{},
		&types.Interpretation{},
		&types.PatientBaseline{},
		&types.ResultReview{},
	)

	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create additional indexes
	if err := db.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	db.log.Info("Database migrations completed")
	return nil
}

// createIndexes creates additional database indexes
func (db *DB) createIndexes() error {
	indexes := []string{
		// Lab results indexes
		`CREATE INDEX IF NOT EXISTS idx_lab_results_patient_code_collected
		 ON lab_results(patient_id, code, collected_at DESC)`,

		`CREATE INDEX IF NOT EXISTS idx_lab_results_collected_desc
		 ON lab_results(collected_at DESC)`,

		// Interpretations indexes
		`CREATE INDEX IF NOT EXISTS idx_interpretations_critical
		 ON interpretations(is_critical) WHERE is_critical = true`,

		`CREATE INDEX IF NOT EXISTS idx_interpretations_panic
		 ON interpretations(is_panic) WHERE is_panic = true`,

		// Reviews indexes
		`CREATE INDEX IF NOT EXISTS idx_reviews_pending
		 ON result_reviews(status) WHERE status = 'PENDING'`,

		`CREATE INDEX IF NOT EXISTS idx_reviews_critical_pending
		 ON result_reviews(status, created_at)
		 WHERE status = 'PENDING'`,
	}

	for _, idx := range indexes {
		if err := db.Postgres.Exec(idx).Error; err != nil {
			db.log.WithError(err).Warn("Failed to create index")
		}
	}

	return nil
}

// Close closes all database connections
func (db *DB) Close() error {
	var errs []error

	// Close PostgreSQL
	if db.Postgres != nil {
		sqlDB, err := db.Postgres.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close PostgreSQL: %w", err))
			}
		}
	}

	// Close Redis
	if db.Redis != nil {
		if err := db.Redis.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing databases: %v", errs)
	}

	db.log.Info("Database connections closed")
	return nil
}

// Health checks database health
func (db *DB) Health(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"postgres": "unknown",
		"redis":    "disabled",
	}

	// Check PostgreSQL
	if db.Postgres != nil {
		sqlDB, err := db.Postgres.DB()
		if err == nil {
			if err := sqlDB.PingContext(ctx); err == nil {
				health["postgres"] = "healthy"
			} else {
				health["postgres"] = "unhealthy"
			}
		}
	}

	// Check Redis
	if db.Redis != nil {
		if err := db.Redis.Ping(ctx).Err(); err == nil {
			health["redis"] = "healthy"
		} else {
			health["redis"] = "unhealthy"
		}
	}

	return health
}

// CacheGet retrieves a value from Redis cache
func (db *DB) CacheGet(ctx context.Context, key string) (string, error) {
	if db.Redis == nil {
		return "", fmt.Errorf("redis not available")
	}
	return db.Redis.Get(ctx, key).Result()
}

// CacheSet stores a value in Redis cache
func (db *DB) CacheSet(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if db.Redis == nil {
		return fmt.Errorf("redis not available")
	}
	return db.Redis.Set(ctx, key, value, ttl).Err()
}

// CacheDelete removes a value from Redis cache
func (db *DB) CacheDelete(ctx context.Context, key string) error {
	if db.Redis == nil {
		return fmt.Errorf("redis not available")
	}
	return db.Redis.Del(ctx, key).Err()
}

// CacheInvalidatePattern invalidates all keys matching a pattern
func (db *DB) CacheInvalidatePattern(ctx context.Context, pattern string) error {
	if db.Redis == nil {
		return fmt.Errorf("redis not available")
	}

	iter := db.Redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := db.Redis.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}
