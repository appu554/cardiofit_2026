package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-drug-interactions/internal/config"
	"kb-drug-interactions/internal/models"
)

type Database struct {
	DB *gorm.DB
}

// NewSharedConnection creates a connection to the shared canonical_facts database
// Used for OHDSI vocabulary and Constitutional DDI rules
func NewSharedConnection(cfg *config.Config) (*Database, error) {
	// Configure GORM logger (minimal for shared DB)
	gormLogger := logger.Default.LogMode(logger.Error)

	dsn := cfg.GetSharedDatabaseDSN()
	if dsn == "" {
		return nil, fmt.Errorf("shared database URL not configured")
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to shared database: %w", err)
	}

	// Configure connection pool (smaller for shared DB)
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL database: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping shared database: %w", err)
	}

	log.Printf("Connected to shared database (canonical_facts) successfully")

	return &Database{DB: db}, nil
}

func NewConnection(cfg *config.Config) (*Database, error) {
	// Configure GORM logger
	var gormLogger logger.Interface
	if cfg.IsDevelopment() {
		gormLogger = logger.Default.LogMode(logger.Info)
	} else {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	// Open database connection
	db, err := gorm.Open(postgres.Open(cfg.GetDatabaseDSN()), &gorm.Config{
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL database: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.MaxConnections / 2)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test the connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Auto-migrate models
	if err := autoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate: %w", err)
	}

	log.Printf("Connected to PostgreSQL successfully (Max connections: %d)", cfg.MaxConnections)

	return &Database{DB: db}, nil
}

func autoMigrate(db *gorm.DB) error {
	// Note: PatientInteractionAlert and InteractionAnalytics are NOT auto-migrated
	// because their schemas are defined in SQL migrations with foreign keys.
	// GORM auto-migrate conflicts with pre-existing SQL-defined FK constraints.
	return db.AutoMigrate(
		&models.DrugInteraction{},
		&models.DrugSynonym{},
		&models.InteractionPattern{},
		&models.InteractionRule{},
		&models.DrugFormulation{},
		&models.CDSConfiguration{},
	)
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL database for closing: %w", err)
	}
	
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	
	log.Println("Database connection closed")
	return nil
}

// HealthCheck performs a database health check
func (d *Database) HealthCheck() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL database: %w", err)
	}
	
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	
	return nil
}

// Repository implementations

type InteractionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) *InteractionRepository {
	return &InteractionRepository{db: db}
}

func (r *InteractionRepository) FindInteractionsBetweenDrugs(drugCodes []string) ([]models.DrugInteraction, error) {
	var interactions []models.DrugInteraction

	// V3 Enhancement: Support RxNorm code lookup via drug_rxnorm_mappings table
	// First, try to resolve RxNorm codes to KB5 drug codes
	resolvedCodes := r.resolveRxNormCodes(drugCodes)

	// Use IN clause instead of ANY() since GORM doesn't properly convert slices to PostgreSQL arrays.
	// This finds all interactions where BOTH drugs are in the provided drug codes list.
	query := r.db.Where("active = true AND drug_a_code IN ? AND drug_b_code IN ?",
		resolvedCodes, resolvedCodes).
		Where("drug_a_code != drug_b_code")

	err := query.Find(&interactions).Error
	return interactions, err
}

// resolveRxNormCodes translates RxNorm codes to KB5 drug codes using the mapping table.
// If no mapping exists, returns the original code (for backwards compatibility).
func (r *InteractionRepository) resolveRxNormCodes(inputCodes []string) []string {
	if len(inputCodes) == 0 {
		return inputCodes
	}

	// Query the mapping table for any RxNorm codes
	type Mapping struct {
		RxNormCode  string `gorm:"column:rxnorm_code"`
		KB5DrugCode string `gorm:"column:kb5_drug_code"`
	}

	var mappings []Mapping
	err := r.db.Table("drug_rxnorm_mappings").
		Select("rxnorm_code, kb5_drug_code").
		Where("rxnorm_code IN ?", inputCodes).
		Find(&mappings).Error

	if err != nil {
		// If mapping table doesn't exist or query fails, return original codes
		return inputCodes
	}

	// Build code map
	codeMap := make(map[string]string)
	for _, m := range mappings {
		codeMap[m.RxNormCode] = m.KB5DrugCode
	}

	// Resolve codes - use mapped code if found, otherwise keep original
	resolvedCodes := make([]string, 0, len(inputCodes)*2)
	for _, code := range inputCodes {
		if mappedCode, exists := codeMap[code]; exists {
			resolvedCodes = append(resolvedCodes, mappedCode)
		}
		// Always include original code for backwards compatibility
		resolvedCodes = append(resolvedCodes, code)
	}

	return resolvedCodes
}

func (r *InteractionRepository) FindInteractionByID(interactionID string) (*models.DrugInteraction, error) {
	var interaction models.DrugInteraction
	err := r.db.Where("interaction_id = ? AND active = true", interactionID).First(&interaction).Error
	return &interaction, err
}

func (r *InteractionRepository) FindInteractionsBySeverity(severity string, limit int) ([]models.DrugInteraction, error) {
	var interactions []models.DrugInteraction
	err := r.db.Where("severity = ? AND active = true", severity).
		Limit(limit).
		Find(&interactions).Error
	return interactions, err
}

func (r *InteractionRepository) Create(interaction *models.DrugInteraction) error {
	return r.db.Create(interaction).Error
}

func (r *InteractionRepository) Update(interaction *models.DrugInteraction) error {
	return r.db.Save(interaction).Error
}

func (r *InteractionRepository) GetAllActiveInteractions() ([]models.DrugInteraction, error) {
	var interactions []models.DrugInteraction
	err := r.db.Where("active = true").
		Order("severity DESC, drug_a_code ASC, drug_b_code ASC").
		Find(&interactions).Error
	return interactions, err
}

func (r *InteractionRepository) FindInteractionsByDrugClass(drugClassA, drugClassB string) ([]models.InteractionPattern, error) {
	var patterns []models.InteractionPattern
	err := r.db.Where("(drug_class_a = ? AND drug_class_b = ?) OR (drug_class_a = ? AND drug_class_b = ?) AND active = true", 
		drugClassA, drugClassB, drugClassB, drugClassA).Find(&patterns).Error
	return patterns, err
}

type SynonymRepository struct {
	db *gorm.DB
}

func NewSynonymRepository(db *gorm.DB) *SynonymRepository {
	return &SynonymRepository{db: db}
}

func (r *SynonymRepository) ResolveDrugCodes(inputCodes []string) (map[string]string, error) {
	var synonyms []models.DrugSynonym
	// Use IN clause instead of ANY() since GORM doesn't properly convert slices to PostgreSQL arrays.
	err := r.db.Where("synonym_code IN ? AND active = true", inputCodes).Find(&synonyms).Error
	if err != nil {
		return nil, err
	}
	
	codeMap := make(map[string]string)
	for _, synonym := range synonyms {
		codeMap[synonym.SynonymCode] = synonym.PrimaryDrugCode
	}
	
	// Add direct mappings for codes not found in synonyms
	for _, code := range inputCodes {
		if _, exists := codeMap[code]; !exists {
			codeMap[code] = code // Use original code if no mapping found
		}
	}
	
	return codeMap, nil
}

func (r *SynonymRepository) FindSynonymsByDrug(drugCode string) ([]models.DrugSynonym, error) {
	var synonyms []models.DrugSynonym
	err := r.db.Where("primary_drug_code = ? AND active = true", drugCode).Find(&synonyms).Error
	return synonyms, err
}

type PatientAlertRepository struct {
	db *gorm.DB
}

func NewPatientAlertRepository(db *gorm.DB) *PatientAlertRepository {
	return &PatientAlertRepository{db: db}
}

func (r *PatientAlertRepository) Create(alert *models.PatientInteractionAlert) error {
	return r.db.Create(alert).Error
}

func (r *PatientAlertRepository) FindActiveAlertsByPatient(patientID string) ([]models.PatientInteractionAlert, error) {
	var alerts []models.PatientInteractionAlert
	err := r.db.Preload("Interaction").
		Where("patient_id = ? AND alert_status = 'active'", patientID).
		Order("alert_triggered_at DESC").
		Find(&alerts).Error
	return alerts, err
}

func (r *PatientAlertRepository) AcknowledgeAlert(alertID string, acknowledgedBy string) error {
	now := time.Now().UTC()
	return r.db.Model(&models.PatientInteractionAlert{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"alert_status":    "acknowledged",
			"acknowledged_by": acknowledgedBy,
			"acknowledged_at": now,
			"updated_at":      now,
		}).Error
}

func (r *PatientAlertRepository) OverrideAlert(alertID string, overrideReason string, overriddenBy string) error {
	now := time.Now().UTC()
	return r.db.Model(&models.PatientInteractionAlert{}).
		Where("id = ?", alertID).
		Updates(map[string]interface{}{
			"alert_status":     "overridden",
			"override_reason":  overrideReason,
			"acknowledged_by":  overriddenBy,
			"acknowledged_at":  now,
			"updated_at":       now,
		}).Error
}

func (r *PatientAlertRepository) FindPatientHistory(patientID string, limit, offset int) ([]models.PatientInteractionAlert, int64, error) {
	var alerts []models.PatientInteractionAlert
	var total int64
	
	// Get total count
	if err := r.db.Model(&models.PatientInteractionAlert{}).
		Where("patient_id = ?", patientID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Get paginated results
	err := r.db.Preload("Interaction").
		Where("patient_id = ?", patientID).
		Order("alert_triggered_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&alerts).Error
		
	return alerts, total, err
}

type RuleRepository struct {
	db *gorm.DB
}

func NewRuleRepository(db *gorm.DB) *RuleRepository {
	return &RuleRepository{db: db}
}

func (r *RuleRepository) FindActiveRules() ([]models.InteractionRule, error) {
	var rules []models.InteractionRule
	now := time.Now().UTC()
	err := r.db.Where("active = true AND effective_date <= ? AND (expiry_date IS NULL OR expiry_date > ?)", now, now).
		Order("priority ASC").
		Find(&rules).Error
	return rules, err
}

func (r *RuleRepository) FindRulesByType(ruleType string) ([]models.InteractionRule, error) {
	var rules []models.InteractionRule
	now := time.Now().UTC()
	err := r.db.Where("rule_type = ? AND active = true AND effective_date <= ? AND (expiry_date IS NULL OR expiry_date > ?)", 
		ruleType, now, now).
		Order("priority ASC").
		Find(&rules).Error
	return rules, err
}

func (r *RuleRepository) Create(rule *models.InteractionRule) error {
	return r.db.Create(rule).Error
}

func (r *RuleRepository) Update(rule *models.InteractionRule) error {
	return r.db.Save(rule).Error
}

type CDSRepository struct {
	db *gorm.DB
}

func NewCDSRepository(db *gorm.DB) *CDSRepository {
	return &CDSRepository{db: db}
}

func (r *CDSRepository) GetConfiguration(configName string) (*models.CDSConfiguration, error) {
	var config models.CDSConfiguration
	err := r.db.Where("config_name = ? AND active = true", configName).First(&config).Error
	return &config, err
}

func (r *CDSRepository) GetDefaultConfiguration() (*models.CDSConfiguration, error) {
	var config models.CDSConfiguration
	err := r.db.Where("active = true").First(&config).Error
	return &config, err
}

type AnalyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) CreateAnalytics(analytics *models.InteractionAnalytics) error {
	return r.db.Create(analytics).Error
}

func (r *AnalyticsRepository) GetInteractionFrequency(interactionID string, days int) (*models.InteractionAnalytics, error) {
	var analytics models.InteractionAnalytics
	startDate := time.Now().AddDate(0, 0, -days)
	err := r.db.Where("interaction_id = ? AND analysis_date >= ?", interactionID, startDate).
		Order("analysis_date DESC").
		First(&analytics).Error
	return &analytics, err
}

func (r *AnalyticsRepository) GetTopInteractions(limit int, days int) ([]models.InteractionAnalytics, error) {
	var analytics []models.InteractionAnalytics
	startDate := time.Now().AddDate(0, 0, -days)
	err := r.db.Where("analysis_date >= ?", startDate).
		Order("alert_frequency DESC").
		Limit(limit).
		Find(&analytics).Error
	return analytics, err
}

// Transaction helpers
func (d *Database) WithTransaction(fn func(*gorm.DB) error) error {
	return d.DB.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// Batch operations
func (r *InteractionRepository) CreateBatch(interactions []models.DrugInteraction) error {
	if len(interactions) == 0 {
		return nil
	}
	
	batchSize := 100
	for i := 0; i < len(interactions); i += batchSize {
		end := i + batchSize
		if end > len(interactions) {
			end = len(interactions)
		}
		
		if err := r.db.CreateInBatches(interactions[i:end], batchSize).Error; err != nil {
			return err
		}
	}
	
	return nil
}

func (r *PatientAlertRepository) CreateBatch(alerts []models.PatientInteractionAlert) error {
	if len(alerts) == 0 {
		return nil
	}
	
	batchSize := 50
	for i := 0; i < len(alerts); i += batchSize {
		end := i + batchSize
		if end > len(alerts) {
			end = len(alerts)
		}
		
		if err := r.db.CreateInBatches(alerts[i:end], batchSize).Error; err != nil {
			return err
		}
	}
	
	return nil
}