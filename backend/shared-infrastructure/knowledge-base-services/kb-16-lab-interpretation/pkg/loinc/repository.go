// Package loinc provides LOINC reference range lookups from the shared canonical_facts database
// This connects to the loinc_reference_ranges table populated by migration 006_expanded_loinc_reference_ranges.sql
package loinc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// =============================================================================
// LOINC Reference Range Repository
// =============================================================================
// Provides database access to the loinc_reference_ranges table in canonical_facts DB.
// This table contains 6041 LOINC codes with reference ranges for DDI context evaluation.
//
// Architecture:
//   Shared DB (canonical_facts:5433) ← KB-16 (this repository) ← Context Router
// =============================================================================

// ReferenceRange represents a LOINC reference range from the database
// Column names match 006_expanded_loinc_reference_ranges.sql migration
type ReferenceRange struct {
	ID                     int64     `json:"id" gorm:"primaryKey"`
	LOINCCode              string    `json:"loinc_code" gorm:"column:loinc_code"`
	Component              string    `json:"component" gorm:"column:component"`
	LongName               string    `json:"long_name" gorm:"column:long_name"`
	Unit                   string    `json:"unit" gorm:"column:unit"`
	LowNormal              *float64  `json:"low_normal" gorm:"column:low_normal"`
	HighNormal             *float64  `json:"high_normal" gorm:"column:high_normal"`
	CriticalLow            *float64  `json:"critical_low" gorm:"column:critical_low"`
	CriticalHigh           *float64  `json:"critical_high" gorm:"column:critical_high"`
	AgeGroup               string    `json:"age_group" gorm:"column:age_group"`
	Sex                    string    `json:"sex" gorm:"column:sex"`
	ClinicalCategory       string    `json:"clinical_category" gorm:"column:clinical_category"`
	InterpretationGuidance string    `json:"interpretation_guidance" gorm:"column:interpretation_guidance"`
	CreatedAt              time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt              time.Time `json:"updated_at" gorm:"column:updated_at"`
}

// TableName specifies the database table name
func (ReferenceRange) TableName() string {
	return "loinc_reference_ranges"
}

// Repository provides access to LOINC reference ranges
type Repository struct {
	db    *gorm.DB
	redis *redis.Client
	log   *logrus.Entry
	ttl   time.Duration
}

// NewRepository creates a new LOINC repository
func NewRepository(db *gorm.DB, redis *redis.Client, log *logrus.Entry) *Repository {
	return &Repository{
		db:    db,
		redis: redis,
		log:   log.WithField("component", "loinc-repository"),
		ttl:   30 * time.Minute,
	}
}

// GetByLOINCCode retrieves reference range by LOINC code
// Returns the default (adult, all) range if no specific age/sex match
func (r *Repository) GetByLOINCCode(ctx context.Context, loincCode string) (*ReferenceRange, error) {
	// Try cache first
	if r.redis != nil {
		cacheKey := fmt.Sprintf("loinc:range:%s", loincCode)
		cached, err := r.redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var ref ReferenceRange
			if json.Unmarshal([]byte(cached), &ref) == nil {
				return &ref, nil
			}
		}
	}

	// Query database - prefer adult default range
	var ref ReferenceRange
	err := r.db.WithContext(ctx).
		Where("loinc_code = ?", loincCode).
		Where("age_group = ? OR age_group IS NULL OR age_group = ''", "adult").
		Where("sex = ? OR sex IS NULL OR sex = ''", "all").
		Order("age_group DESC, sex DESC").
		First(&ref).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Try without age/sex filters
			err = r.db.WithContext(ctx).
				Where("loinc_code = ?", loincCode).
				First(&ref).Error
			if err == gorm.ErrRecordNotFound {
				return nil, nil // Not found
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed to query LOINC reference range: %w", err)
		}
	}

	// Cache result
	if r.redis != nil {
		cacheKey := fmt.Sprintf("loinc:range:%s", loincCode)
		if data, err := json.Marshal(ref); err == nil {
			r.redis.Set(ctx, cacheKey, data, r.ttl)
		}
	}

	return &ref, nil
}

// GetByLOINCCodeWithContext retrieves reference range with age/sex specificity
func (r *Repository) GetByLOINCCodeWithContext(ctx context.Context, loincCode string, age int, sex string) (*ReferenceRange, error) {
	// Determine age group
	ageGroup := r.determineAgeGroup(age)

	// Try cache first
	if r.redis != nil {
		cacheKey := fmt.Sprintf("loinc:range:%s:%s:%s", loincCode, ageGroup, sex)
		cached, err := r.redis.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var ref ReferenceRange
			if json.Unmarshal([]byte(cached), &ref) == nil {
				return &ref, nil
			}
		}
	}

	// Query with specificity - try exact match first, then fall back
	var ref ReferenceRange

	// First try: exact age_group and sex match
	// Order by specificity: exact matches first
	err := r.db.WithContext(ctx).
		Where("loinc_code = ?", loincCode).
		Where("(age_group = ? OR age_group = '' OR age_group IS NULL)", ageGroup).
		Where("(sex = ? OR sex = 'all' OR sex = '' OR sex IS NULL)", sex).
		Order("age_group DESC, sex DESC").
		First(&ref).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Fall back to default range
			return r.GetByLOINCCode(ctx, loincCode)
		}
		return nil, fmt.Errorf("failed to query LOINC reference range: %w", err)
	}

	// Cache result
	if r.redis != nil {
		cacheKey := fmt.Sprintf("loinc:range:%s:%s:%s", loincCode, ageGroup, sex)
		if data, err := json.Marshal(ref); err == nil {
			r.redis.Set(ctx, cacheKey, data, r.ttl)
		}
	}

	return &ref, nil
}

// GetDDIRelevantRanges retrieves LOINC codes relevant for DDI context evaluation
// Returns electrolyte, renal, hepatic, and coagulation categories (most DDI-relevant)
func (r *Repository) GetDDIRelevantRanges(ctx context.Context) ([]ReferenceRange, error) {
	// Try cache
	if r.redis != nil {
		cached, err := r.redis.Get(ctx, "loinc:ddi_relevant").Result()
		if err == nil && cached != "" {
			var refs []ReferenceRange
			if json.Unmarshal([]byte(cached), &refs) == nil {
				return refs, nil
			}
		}
	}

	// DDI-relevant categories for drug interaction context
	ddiCategories := []string{"electrolyte", "renal", "hepatic", "coagulation", "cardiac"}

	var refs []ReferenceRange
	err := r.db.WithContext(ctx).
		Where("clinical_category IN ?", ddiCategories).
		Order("clinical_category, loinc_code").
		Find(&refs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query DDI-relevant LOINC ranges: %w", err)
	}

	// Cache result
	if r.redis != nil {
		if data, err := json.Marshal(refs); err == nil {
			r.redis.Set(ctx, "loinc:ddi_relevant", data, r.ttl)
		}
	}

	return refs, nil
}

// GetByCategory retrieves all reference ranges for a clinical category
func (r *Repository) GetByCategory(ctx context.Context, category string) ([]ReferenceRange, error) {
	var refs []ReferenceRange
	err := r.db.WithContext(ctx).
		Where("clinical_category = ?", category).
		Order("loinc_code").
		Find(&refs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query LOINC ranges by category: %w", err)
	}

	return refs, nil
}

// SearchByName searches LOINC codes by component or long name
func (r *Repository) SearchByName(ctx context.Context, query string, limit int) ([]ReferenceRange, error) {
	if limit <= 0 {
		limit = 50
	}

	var refs []ReferenceRange
	err := r.db.WithContext(ctx).
		Where("component ILIKE ? OR long_name ILIKE ?", "%"+query+"%", "%"+query+"%").
		Order("component").
		Limit(limit).
		Find(&refs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to search LOINC ranges: %w", err)
	}

	return refs, nil
}

// ListCategories returns all unique clinical categories
func (r *Repository) ListCategories(ctx context.Context) ([]string, error) {
	var categories []string
	err := r.db.WithContext(ctx).
		Model(&ReferenceRange{}).
		Distinct("clinical_category").
		Where("clinical_category IS NOT NULL AND clinical_category != ''").
		Order("clinical_category").
		Pluck("clinical_category", &categories).Error

	if err != nil {
		return nil, fmt.Errorf("failed to list LOINC categories: %w", err)
	}

	return categories, nil
}

// GetStatistics returns repository statistics
func (r *Repository) GetStatistics(ctx context.Context) (*RepositoryStats, error) {
	stats := &RepositoryStats{}

	// Total count
	var total int64
	r.db.WithContext(ctx).Model(&ReferenceRange{}).Count(&total)
	stats.TotalCodes = int(total)

	// DDI relevant count
	var ddiCount int64
	r.db.WithContext(ctx).Model(&ReferenceRange{}).Where("ddi_relevant = ?", true).Count(&ddiCount)
	stats.DDIRelevantCodes = int(ddiCount)

	// Categories
	categories, _ := r.ListCategories(ctx)
	stats.Categories = categories
	stats.CategoryCount = len(categories)

	return stats, nil
}

// RepositoryStats contains statistics about the LOINC repository
type RepositoryStats struct {
	TotalCodes       int      `json:"total_codes"`
	DDIRelevantCodes int      `json:"ddi_relevant_codes"`
	CategoryCount    int      `json:"category_count"`
	Categories       []string `json:"categories"`
}

// determineAgeGroup maps age to age group
func (r *Repository) determineAgeGroup(age int) string {
	switch {
	case age < 0:
		return "adult" // default
	case age < 1:
		return "neonate"
	case age < 12:
		return "pediatric"
	case age < 18:
		return "adolescent"
	case age >= 65:
		return "geriatric"
	default:
		return "adult"
	}
}

// =============================================================================
// Context Router Response Models
// =============================================================================

// LOINCReferenceResponse is the response format for Context Router
// Field names match 006 migration column names for consistency
type LOINCReferenceResponse struct {
	LOINCCode              string   `json:"loinc_code"`
	Component              string   `json:"component"`
	LongName               string   `json:"long_name"`
	Unit                   string   `json:"unit"`
	LowNormal              *float64 `json:"low_normal"`
	HighNormal             *float64 `json:"high_normal"`
	CriticalLow            *float64 `json:"critical_low"`
	CriticalHigh           *float64 `json:"critical_high"`
	ClinicalCategory       string   `json:"clinical_category"`
	AgeGroup               string   `json:"age_group"`
	Sex                    string   `json:"sex"`
	InterpretationGuidance string   `json:"interpretation_guidance,omitempty"`
}

// ToResponse converts ReferenceRange to API response
func (ref *ReferenceRange) ToResponse() *LOINCReferenceResponse {
	if ref == nil {
		return nil
	}
	return &LOINCReferenceResponse{
		LOINCCode:              ref.LOINCCode,
		Component:              ref.Component,
		LongName:               ref.LongName,
		Unit:                   ref.Unit,
		LowNormal:              ref.LowNormal,
		HighNormal:             ref.HighNormal,
		CriticalLow:            ref.CriticalLow,
		CriticalHigh:           ref.CriticalHigh,
		ClinicalCategory:       ref.ClinicalCategory,
		AgeGroup:               ref.AgeGroup,
		Sex:                    ref.Sex,
		InterpretationGuidance: ref.InterpretationGuidance,
	}
}
