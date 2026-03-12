package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-guideline-evidence/internal/config"
	"kb-guideline-evidence/internal/models"
)

// Connection holds the database connection and configuration
type Connection struct {
	DB     *gorm.DB
	Config *config.Config
}

// NewConnection creates a new database connection
func NewConnection(cfg *config.Config) (*Connection, error) {
	// Configure GORM logger based on environment
	var gormLogger logger.Interface
	if cfg.Debug {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	conn := &Connection{
		DB:     db,
		Config: cfg,
	}

	log.Printf("Connected to database successfully")
	return conn, nil
}

// AutoMigrate runs database migrations
func (c *Connection) AutoMigrate() error {
	log.Println("Running database migrations...")
	
	// Migrate the schema
	err := c.DB.AutoMigrate(
		&models.GuidelineDocument{},
		&models.Recommendation{},
		&models.RegionalProfile{},
		&models.GuidelineVersion{},
	)
	
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}
	
	log.Println("Database migrations completed successfully")
	return nil
}

// Close closes the database connection
func (c *Connection) Close() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	
	log.Println("Database connection closed")
	return nil
}

// HealthCheck performs a database health check
func (c *Connection) HealthCheck() error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	
	return nil
}

// GetStats returns database connection statistics
func (c *Connection) GetStats() map[string]interface{} {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return map[string]interface{}{
			"error": "failed to get sql.DB",
		}
	}
	
	stats := sqlDB.Stats()
	return map[string]interface{}{
		"open_connections":     stats.OpenConnections,
		"in_use":              stats.InUse,
		"idle":                stats.Idle,
		"wait_count":          stats.WaitCount,
		"wait_duration":       stats.WaitDuration.String(),
		"max_idle_closed":     stats.MaxIdleClosed,
		"max_idle_time_closed": stats.MaxIdleTimeClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
	}
}

// Transaction executes a function within a database transaction
func (c *Connection) Transaction(fn func(*gorm.DB) error) error {
	return c.DB.Transaction(fn)
}

// GuidelineRepository provides methods for guideline operations
type GuidelineRepository struct {
	db *gorm.DB
}

// NewGuidelineRepository creates a new guideline repository
func NewGuidelineRepository(db *gorm.DB) *GuidelineRepository {
	return &GuidelineRepository{db: db}
}

// GetByID retrieves a guideline by ID
func (r *GuidelineRepository) GetByID(id string) (*models.GuidelineDocument, error) {
	var guideline models.GuidelineDocument
	err := r.db.Preload("Recommendations").
		Where("guideline_id = ? AND deleted_at IS NULL", id).
		First(&guideline).Error
	if err != nil {
		return nil, err
	}
	return &guideline, nil
}

// GetByCondition retrieves guidelines by medical condition
func (r *GuidelineRepository) GetByCondition(condition string, region *string) ([]models.GuidelineDocument, error) {
	var guidelines []models.GuidelineDocument
	
	query := r.db.Preload("Recommendations").
		Where("is_active = ? AND deleted_at IS NULL", true).
		Where("condition->>'primary' ILIKE ?", "%"+condition+"%")
	
	if region != nil {
		query = query.Where("source->>'region' = ?", *region)
	}
	
	err := query.Order("effective_date DESC").Find(&guidelines).Error
	return guidelines, err
}

// GetEffectiveGuidelines retrieves guidelines effective at a specific date
func (r *GuidelineRepository) GetEffectiveGuidelines(date time.Time, region *string) ([]models.GuidelineDocument, error) {
	var guidelines []models.GuidelineDocument
	
	query := r.db.Preload("Recommendations").
		Where("is_active = ? AND deleted_at IS NULL", true).
		Where("effective_date <= ?", date).
		Where("superseded_date IS NULL OR superseded_date > ?", date)
	
	if region != nil {
		query = query.Where("source->>'region' = ?", *region)
	}
	
	err := query.Order("effective_date DESC").Find(&guidelines).Error
	return guidelines, err
}

// Search performs full-text search across guidelines and recommendations
func (r *GuidelineRepository) Search(query string, limit int) ([]models.GuidelineDocument, error) {
	var guidelines []models.GuidelineDocument
	
	err := r.db.Preload("Recommendations").
		Joins("LEFT JOIN recommendations ON recommendations.guideline_id = guideline_documents.id").
		Where(`
			guideline_documents.deleted_at IS NULL 
			AND guideline_documents.is_active = true
			AND (
				to_tsvector('english', guideline_documents.condition->>'primary') @@ plainto_tsquery('english', ?)
				OR to_tsvector('english', recommendations.recommendation) @@ plainto_tsquery('english', ?)
			)
		`, query, query).
		Group("guideline_documents.id").
		Limit(limit).
		Find(&guidelines).Error
	
	return guidelines, err
}

// RecommendationRepository provides methods for recommendation operations  
type RecommendationRepository struct {
	db *gorm.DB
}

// NewRecommendationRepository creates a new recommendation repository
func NewRecommendationRepository(db *gorm.DB) *RecommendationRepository {
	return &RecommendationRepository{db: db}
}

// GetByRecID retrieves a recommendation by rec_id
func (r *RecommendationRepository) GetByRecID(recID string) (*models.Recommendation, error) {
	var recommendation models.Recommendation
	err := r.db.Where("rec_id = ? AND deleted_at IS NULL", recID).
		First(&recommendation).Error
	if err != nil {
		return nil, err
	}
	return &recommendation, nil
}

// GetByDomain retrieves recommendations by domain
func (r *RecommendationRepository) GetByDomain(domain string, region *string) ([]models.Recommendation, error) {
	var recommendations []models.Recommendation
	
	query := r.db.Joins("JOIN guideline_documents ON guideline_documents.id = recommendations.guideline_id").
		Where("recommendations.domain = ? AND recommendations.deleted_at IS NULL", domain).
		Where("guideline_documents.is_active = ? AND guideline_documents.deleted_at IS NULL", true)
	
	if region != nil {
		query = query.Where("guideline_documents.source->>'region' = ?", *region)
	}
	
	err := query.Order("recommendations.evidence_grade ASC").Find(&recommendations).Error
	return recommendations, err
}

// GetWithCrossKBLinks retrieves recommendations that have cross-KB links
func (r *RecommendationRepository) GetWithCrossKBLinks(kbName *string) ([]models.Recommendation, error) {
	var recommendations []models.Recommendation
	
	query := r.db.Where("linked_kb_refs IS NOT NULL AND deleted_at IS NULL")
	
	if kbName != nil {
		// Filter by specific KB that has links
		switch *kbName {
		case "kb1":
			query = query.Where("linked_kb_refs->'kb1_dosing' IS NOT NULL")
		case "kb2":
			query = query.Where("linked_kb_refs->'kb2_phenotypes' IS NOT NULL")
		case "kb4":
			query = query.Where("linked_kb_refs->'kb4_safety' IS NOT NULL")
		case "kb5":
			query = query.Where("linked_kb_refs->'kb5_interactions' IS NOT NULL")
		case "kb6":
			query = query.Where("linked_kb_refs->'kb6_formulary' IS NOT NULL")
		case "kb7":
			query = query.Where("linked_kb_refs->'kb7_terminology' IS NOT NULL")
		}
	}
	
	err := query.Find(&recommendations).Error
	return recommendations, err
}

// RegionalProfileRepository provides methods for regional profile operations
type RegionalProfileRepository struct {
	db *gorm.DB
}

// NewRegionalProfileRepository creates a new regional profile repository
func NewRegionalProfileRepository(db *gorm.DB) *RegionalProfileRepository {
	return &RegionalProfileRepository{db: db}
}

// GetByRegion retrieves a regional profile by region
func (r *RegionalProfileRepository) GetByRegion(region string) (*models.RegionalProfile, error) {
	var profile models.RegionalProfile
	err := r.db.Where("region = ? AND deleted_at IS NULL", region).
		First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetAll retrieves all regional profiles
func (r *RegionalProfileRepository) GetAll() ([]models.RegionalProfile, error) {
	var profiles []models.RegionalProfile
	err := r.db.Where("deleted_at IS NULL").Find(&profiles).Error
	return profiles, err
}