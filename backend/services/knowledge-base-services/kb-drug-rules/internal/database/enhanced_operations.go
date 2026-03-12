package database

import (
	"encoding/json"
	"fmt"
	"time"

	"kb-drug-rules/internal/models"

	"gorm.io/gorm"
)

// EnhancedDrugRuleRepository provides enhanced database operations with TOML support
type EnhancedDrugRuleRepository struct {
	db *gorm.DB
}

// NewEnhancedDrugRuleRepository creates a new enhanced repository
func NewEnhancedDrugRuleRepository(db *gorm.DB) *EnhancedDrugRuleRepository {
	return &EnhancedDrugRuleRepository{
		db: db,
	}
}

// SaveRulePackWithVersioning saves a rule pack with automatic versioning and snapshot creation
func (r *EnhancedDrugRuleRepository) SaveRulePackWithVersioning(rulePack *models.DrugRulePack) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Check if this is an update to existing drug
		var existingPack models.DrugRulePack
		err := tx.Where("drug_id = ?", rulePack.DrugID).
			Order("updated_at DESC").
			First(&existingPack).Error

		if err == nil {
			// Existing drug found - set previous version
			rulePack.PreviousVersion = &existingPack.Version
			
			// Create snapshot before updating
			snapshot := &models.DrugRuleSnapshot{
				DrugID:          existingPack.DrugID,
				Version:         existingPack.Version,
				ContentSnapshot: existingPack.JSONContent,
				TOMLSnapshot:    existingPack.TOMLContent,
				CreatedBy:       rulePack.LastModifiedBy,
				Reason:          fmt.Sprintf("Automatic snapshot before version %s", rulePack.Version),
			}
			
			if err := tx.Create(snapshot).Error; err != nil {
				return fmt.Errorf("failed to create snapshot: %w", err)
			}
		} else if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to check existing rule pack: %w", err)
		}

		// Save the new rule pack
		if err := tx.Create(rulePack).Error; err != nil {
			return fmt.Errorf("failed to save rule pack: %w", err)
		}

		return nil
	})
}

// GetRulePackByVersion retrieves a specific version of a rule pack
func (r *EnhancedDrugRuleRepository) GetRulePackByVersion(drugID, version string) (*models.DrugRulePack, error) {
	var rulePack models.DrugRulePack
	
	err := r.db.Where("drug_id = ? AND version = ?", drugID, version).
		First(&rulePack).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("rule pack not found for drug %s version %s", drugID, version)
		}
		return nil, fmt.Errorf("failed to retrieve rule pack: %w", err)
	}
	
	return &rulePack, nil
}

// GetLatestRulePack retrieves the latest version of a rule pack
func (r *EnhancedDrugRuleRepository) GetLatestRulePack(drugID string) (*models.DrugRulePack, error) {
	var rulePack models.DrugRulePack
	
	err := r.db.Where("drug_id = ?", drugID).
		Order("updated_at DESC").
		First(&rulePack).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("no rule pack found for drug %s", drugID)
		}
		return nil, fmt.Errorf("failed to retrieve latest rule pack: %w", err)
	}
	
	return &rulePack, nil
}

// GetVersionHistory retrieves version history for a drug
func (r *EnhancedDrugRuleRepository) GetVersionHistory(drugID string, limit int) ([]models.VersionHistoryEntry, int, error) {
	var rulePacks []models.DrugRulePack
	var total int64
	
	// Get total count
	if err := r.db.Model(&models.DrugRulePack{}).
		Where("drug_id = ?", drugID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count versions: %w", err)
	}
	
	// Get rule packs with limit
	err := r.db.Where("drug_id = ?", drugID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&rulePacks).Error
	
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve version history: %w", err)
	}
	
	// Convert to version history entries
	var history []models.VersionHistoryEntry
	for _, pack := range rulePacks {
		entry := models.VersionHistoryEntry{
			Version:      pack.Version,
			ModifiedDate: pack.UpdatedAt,
			ModifiedBy:   pack.LastModifiedBy,
			ChangeSummary: fmt.Sprintf("Version %s deployment", pack.Version),
		}
		
		// Try to find associated snapshot
		var snapshot models.DrugRuleSnapshot
		if err := r.db.Where("drug_id = ? AND version = ?", drugID, pack.Version).
			First(&snapshot).Error; err == nil {
			entry.SnapshotID = snapshot.ID
		}
		
		history = append(history, entry)
	}
	
	return history, int(total), nil
}

// CreateSnapshot creates a manual snapshot
func (r *EnhancedDrugRuleRepository) CreateSnapshot(snapshot *models.DrugRuleSnapshot) error {
	if err := r.db.Create(snapshot).Error; err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	return nil
}

// GetSnapshot retrieves a snapshot by ID
func (r *EnhancedDrugRuleRepository) GetSnapshot(snapshotID string) (*models.DrugRuleSnapshot, error) {
	var snapshot models.DrugRuleSnapshot
	
	err := r.db.Where("id = ?", snapshotID).First(&snapshot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
		}
		return nil, fmt.Errorf("failed to retrieve snapshot: %w", err)
	}
	
	return &snapshot, nil
}

// ListRulePacksByFormat retrieves rule packs by original format
func (r *EnhancedDrugRuleRepository) ListRulePacksByFormat(format string, limit, offset int) ([]models.DrugRulePack, int, error) {
	var rulePacks []models.DrugRulePack
	var total int64
	
	query := r.db.Model(&models.DrugRulePack{})
	if format != "" {
		query = query.Where("original_format = ?", format)
	}
	
	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count rule packs: %w", err)
	}
	
	// Get rule packs with pagination
	err := query.Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rulePacks).Error
	
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve rule packs: %w", err)
	}
	
	return rulePacks, int(total), nil
}

// SearchRulePacksByTags searches rule packs by tags
func (r *EnhancedDrugRuleRepository) SearchRulePacksByTags(tags []string, limit, offset int) ([]models.DrugRulePack, int, error) {
	var rulePacks []models.DrugRulePack
	var total int64
	
	query := r.db.Model(&models.DrugRulePack{})
	
	// Build tag search query
	if len(tags) > 0 {
		for _, tag := range tags {
			query = query.Where("? = ANY(tags)", tag)
		}
	}
	
	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count rule packs by tags: %w", err)
	}
	
	// Get rule packs with pagination
	err := query.Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rulePacks).Error
	
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search rule packs by tags: %w", err)
	}
	
	return rulePacks, int(total), nil
}

// GetRulePacksByDeploymentStatus retrieves rule packs by deployment status
func (r *EnhancedDrugRuleRepository) GetRulePacksByDeploymentStatus(environment, status string, limit, offset int) ([]models.DrugRulePack, int, error) {
	var rulePacks []models.DrugRulePack
	var total int64
	
	query := r.db.Model(&models.DrugRulePack{}).
		Where("deployment_status->>? = ?", environment, status)
	
	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count rule packs by deployment status: %w", err)
	}
	
	// Get rule packs with pagination
	err := query.Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rulePacks).Error
	
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve rule packs by deployment status: %w", err)
	}
	
	return rulePacks, int(total), nil
}

// UpdateDeploymentStatus updates the deployment status of a rule pack
func (r *EnhancedDrugRuleRepository) UpdateDeploymentStatus(drugID, version, environment, status string) error {
	// Get the rule pack
	var rulePack models.DrugRulePack
	err := r.db.Where("drug_id = ? AND version = ?", drugID, version).
		First(&rulePack).Error
	
	if err != nil {
		return fmt.Errorf("failed to find rule pack: %w", err)
	}
	
	// Update deployment status
	deploymentStatus := models.DeploymentStatus(rulePack.DeploymentStatus)
	switch environment {
	case "staging":
		deploymentStatus.Staging = status
	case "production":
		deploymentStatus.Production = status
	default:
		return fmt.Errorf("unsupported environment: %s", environment)
	}
	
	if status == "deployed" {
		deploymentStatus.LastDeployed = time.Now()
	}
	
	// Save updated status
	err = r.db.Model(&rulePack).
		Update("deployment_status", models.DeploymentStatusJSON(deploymentStatus)).Error
	
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}
	
	return nil
}

// GetRulePackStats retrieves statistics about rule packs
func (r *EnhancedDrugRuleRepository) GetRulePackStats() (*RulePackStats, error) {
	stats := &RulePackStats{}
	
	// Total count
	if err := r.db.Model(&models.DrugRulePack{}).Count(&stats.TotalRulePacks).Error; err != nil {
		return nil, fmt.Errorf("failed to count total rule packs: %w", err)
	}
	
	// Count by format
	var formatCounts []struct {
		OriginalFormat string
		Count          int64
	}
	
	err := r.db.Model(&models.DrugRulePack{}).
		Select("original_format, COUNT(*) as count").
		Group("original_format").
		Find(&formatCounts).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to count by format: %w", err)
	}
	
	stats.FormatCounts = make(map[string]int64)
	for _, fc := range formatCounts {
		stats.FormatCounts[fc.OriginalFormat] = fc.Count
	}
	
	// Count by deployment status
	var stagingDeployed, productionDeployed int64
	
	r.db.Model(&models.DrugRulePack{}).
		Where("deployment_status->>'staging' = ?", "deployed").
		Count(&stagingDeployed)
	
	r.db.Model(&models.DrugRulePack{}).
		Where("deployment_status->>'production' = ?", "deployed").
		Count(&productionDeployed)
	
	stats.StagingDeployed = stagingDeployed
	stats.ProductionDeployed = productionDeployed
	
	// Recent activity (last 24 hours)
	yesterday := time.Now().Add(-24 * time.Hour)
	r.db.Model(&models.DrugRulePack{}).
		Where("updated_at > ?", yesterday).
		Count(&stats.RecentUpdates)
	
	return stats, nil
}

// RulePackStats represents statistics about rule packs
type RulePackStats struct {
	TotalRulePacks      int64            `json:"total_rule_packs"`
	FormatCounts        map[string]int64 `json:"format_counts"`
	StagingDeployed     int64            `json:"staging_deployed"`
	ProductionDeployed  int64            `json:"production_deployed"`
	RecentUpdates       int64            `json:"recent_updates"`
}

// CleanupOldSnapshots removes snapshots older than the specified duration
func (r *EnhancedDrugRuleRepository) CleanupOldSnapshots(olderThan time.Duration) (int64, error) {
	cutoffTime := time.Now().Add(-olderThan)
	
	result := r.db.Where("snapshot_date < ?", cutoffTime).
		Delete(&models.DrugRuleSnapshot{})
	
	if result.Error != nil {
		return 0, fmt.Errorf("failed to cleanup old snapshots: %w", result.Error)
	}
	
	return result.RowsAffected, nil
}

// ValidateRulePackIntegrity validates the integrity of a rule pack
func (r *EnhancedDrugRuleRepository) ValidateRulePackIntegrity(drugID, version string) (*IntegrityReport, error) {
	rulePack, err := r.GetRulePackByVersion(drugID, version)
	if err != nil {
		return nil, err
	}
	
	report := &IntegrityReport{
		DrugID:  drugID,
		Version: version,
		Issues:  []string{},
	}
	
	// Validate JSON content
	if len(rulePack.JSONContent) == 0 {
		report.Issues = append(report.Issues, "JSON content is empty")
	} else {
		var jsonData interface{}
		if err := json.Unmarshal(rulePack.JSONContent, &jsonData); err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("Invalid JSON content: %v", err))
		}
	}
	
	// Validate TOML content if present
	if rulePack.TOMLContent != nil && *rulePack.TOMLContent != "" {
		// TOML validation would go here
		// For now, just check it's not empty
		if len(*rulePack.TOMLContent) == 0 {
			report.Issues = append(report.Issues, "TOML content is empty but format is TOML")
		}
	}
	
	// Validate required fields
	if rulePack.DrugID == "" {
		report.Issues = append(report.Issues, "Drug ID is empty")
	}
	
	if rulePack.Version == "" {
		report.Issues = append(report.Issues, "Version is empty")
	}
	
	if rulePack.ContentSHA == "" {
		report.Issues = append(report.Issues, "Content SHA is empty")
	}
	
	report.IsValid = len(report.Issues) == 0
	
	return report, nil
}

// IntegrityReport represents the result of integrity validation
type IntegrityReport struct {
	DrugID  string   `json:"drug_id"`
	Version string   `json:"version"`
	IsValid bool     `json:"is_valid"`
	Issues  []string `json:"issues"`
}
