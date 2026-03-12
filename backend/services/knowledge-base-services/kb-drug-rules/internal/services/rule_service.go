package services

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-drug-rules/internal/cache"
	"kb-drug-rules/internal/models"
)

// ruleService implements the RuleService interface
type ruleService struct {
	db     *gorm.DB
	cache  cache.Client
	logger *logrus.Logger
}

// NewRuleService creates a new rule service
func NewRuleService(db *gorm.DB, cache cache.Client, logger *logrus.Logger) RuleService {
	return &ruleService{
		db:     db,
		cache:  cache,
		logger: logger,
	}
}

// GetDrugRules retrieves drug rules by ID, version, and region
func (r *ruleService) GetDrugRules(drugID string, version *string, region *string) (*models.DrugRulePack, error) {
	var rulePack models.DrugRulePack
	
	query := r.db.Where("drug_id = ?", drugID)
	
	if version != nil {
		query = query.Where("version = ?", *version)
	} else {
		// Get latest version
		latestVersion, err := r.GetLatestVersion(drugID)
		if err != nil {
			return nil, err
		}
		query = query.Where("version = ?", latestVersion)
	}
	
	err := query.First(&rulePack).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRuleNotFound
		}
		return nil, fmt.Errorf("failed to get drug rules: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"drug_id": drugID,
		"version": rulePack.Version,
		"region":  getStringValue(region),
	}).Debug("Retrieved drug rules")
	
	return &rulePack, nil
}

// SaveRulePack saves a new rule pack to the database
func (r *ruleService) SaveRulePack(rulePack *models.DrugRulePack) error {
	// Check if version already exists
	var existing models.DrugRulePack
	err := r.db.Where("drug_id = ? AND version = ?", rulePack.DrugID, rulePack.Version).First(&existing).Error
	if err == nil {
		return ErrDuplicateRule
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check existing rule: %w", err)
	}
	
	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Save rule pack
	if err := tx.Create(rulePack).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save rule pack: %w", err)
	}
	
	// Update latest version pointer
	if err := r.updateLatestVersionInTx(tx, rulePack.DrugID, rulePack.Version); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update latest version: %w", err)
	}
	
	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"drug_id": rulePack.DrugID,
		"version": rulePack.Version,
		"regions": rulePack.Regions,
	}).Info("Rule pack saved")
	
	return nil
}

// ListVersions lists all versions for a drug
func (r *ruleService) ListVersions(drugID string) ([]string, error) {
	var versions []string
	
	err := r.db.Model(&models.DrugRulePack{}).
		Where("drug_id = ?", drugID).
		Order("created_at DESC").
		Pluck("version", &versions).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	
	return versions, nil
}

// DeleteVersion deletes a specific version of drug rules
func (r *ruleService) DeleteVersion(drugID, version string) error {
	// Check if version exists
	var rulePack models.DrugRulePack
	err := r.db.Where("drug_id = ? AND version = ?", drugID, version).First(&rulePack).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrRuleNotFound
		}
		return fmt.Errorf("failed to find rule version: %w", err)
	}
	
	// Start transaction
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// Delete the version
	if err := tx.Delete(&rulePack).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete rule version: %w", err)
	}
	
	// Check if this was the latest version
	latestVersion, err := r.getLatestVersionInTx(tx, drugID)
	if err == nil && latestVersion == version {
		// Find the next latest version
		var nextLatest string
		err := tx.Model(&models.DrugRulePack{}).
			Where("drug_id = ? AND version != ?", drugID, version).
			Order("created_at DESC").
			Limit(1).
			Pluck("version", &nextLatest).Error
		
		if err == nil && nextLatest != "" {
			if err := r.updateLatestVersionInTx(tx, drugID, nextLatest); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to update latest version: %w", err)
			}
		}
	}
	
	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	r.logger.WithFields(logrus.Fields{
		"drug_id": drugID,
		"version": version,
	}).Info("Rule version deleted")
	
	return nil
}

// GetLatestVersion gets the latest version for a drug
func (r *ruleService) GetLatestVersion(drugID string) (string, error) {
	var version string
	
	// Try to get from latest_versions table first (if it exists)
	err := r.db.Raw("SELECT version FROM drug_latest_versions WHERE drug_id = ?", drugID).Scan(&version).Error
	if err == nil && version != "" {
		return version, nil
	}
	
	// Fallback to getting the most recent version by created_at
	err = r.db.Model(&models.DrugRulePack{}).
		Where("drug_id = ?", drugID).
		Order("created_at DESC").
		Limit(1).
		Pluck("version", &version).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrRuleNotFound
		}
		return "", fmt.Errorf("failed to get latest version: %w", err)
	}
	
	if version == "" {
		return "", ErrRuleNotFound
	}
	
	return version, nil
}

// UpdateLatestVersion updates the latest version pointer
func (r *ruleService) UpdateLatestVersion(drugID, version string) error {
	return r.updateLatestVersionInTx(r.db, drugID, version)
}

// Helper methods

func (r *ruleService) updateLatestVersionInTx(tx *gorm.DB, drugID, version string) error {
	// Create or update latest version record
	err := tx.Exec(`
		INSERT INTO drug_latest_versions (drug_id, version, updated_at) 
		VALUES (?, ?, NOW()) 
		ON CONFLICT (drug_id) 
		DO UPDATE SET version = EXCLUDED.version, updated_at = EXCLUDED.updated_at
	`, drugID, version).Error
	
	if err != nil {
		// If table doesn't exist, create it
		if err := tx.Exec(`
			CREATE TABLE IF NOT EXISTS drug_latest_versions (
				drug_id VARCHAR(255) PRIMARY KEY,
				version VARCHAR(50) NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			)
		`).Error; err != nil {
			return fmt.Errorf("failed to create latest_versions table: %w", err)
		}
		
		// Try the insert again
		err = tx.Exec(`
			INSERT INTO drug_latest_versions (drug_id, version, updated_at) 
			VALUES (?, ?, NOW()) 
			ON CONFLICT (drug_id) 
			DO UPDATE SET version = EXCLUDED.version, updated_at = EXCLUDED.updated_at
		`, drugID, version).Error
	}
	
	return err
}

func (r *ruleService) getLatestVersionInTx(tx *gorm.DB, drugID string) (string, error) {
	var version string
	err := tx.Raw("SELECT version FROM drug_latest_versions WHERE drug_id = ?", drugID).Scan(&version).Error
	return version, err
}

func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
