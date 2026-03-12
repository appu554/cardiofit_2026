// Package models contains domain models for KB-17 Population Registry
package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// RegistryCode represents the unique identifier for a disease registry
type RegistryCode string

const (
	RegistryDiabetes        RegistryCode = "DIABETES"
	RegistryHypertension    RegistryCode = "HYPERTENSION"
	RegistryHeartFailure    RegistryCode = "HEART_FAILURE"
	RegistryCKD             RegistryCode = "CKD"
	RegistryCOPD            RegistryCode = "COPD"
	RegistryPregnancy       RegistryCode = "PREGNANCY"
	RegistryOpioidUse       RegistryCode = "OPIOID_USE"
	RegistryAnticoagulation RegistryCode = "ANTICOAGULATION"
)

// IsValid checks if the registry code is valid
func (r RegistryCode) IsValid() bool {
	switch r {
	case RegistryDiabetes, RegistryHypertension, RegistryHeartFailure,
		RegistryCKD, RegistryCOPD, RegistryPregnancy,
		RegistryOpioidUse, RegistryAnticoagulation:
		return true
	}
	return false
}

// String returns the string representation
func (r RegistryCode) String() string {
	return string(r)
}

// RegistryCategory represents the category of a registry
type RegistryCategory string

const (
	CategoryChronic     RegistryCategory = "CHRONIC"
	CategoryAcute       RegistryCategory = "ACUTE"
	CategoryPreventive  RegistryCategory = "PREVENTIVE"
	CategoryMedication  RegistryCategory = "MEDICATION"
	CategorySpecialty   RegistryCategory = "SPECIALTY"
	CategoryCustom      RegistryCategory = "CUSTOM"
)

// RiskTier represents the risk stratification tier for a patient
type RiskTier string

const (
	RiskTierLow      RiskTier = "LOW"
	RiskTierModerate RiskTier = "MODERATE"
	RiskTierHigh     RiskTier = "HIGH"
	RiskTierCritical RiskTier = "CRITICAL"
)

// IsValid checks if the risk tier is valid
func (r RiskTier) IsValid() bool {
	switch r {
	case RiskTierLow, RiskTierModerate, RiskTierHigh, RiskTierCritical:
		return true
	}
	return false
}

// Priority returns the numeric priority (higher = more urgent)
func (r RiskTier) Priority() int {
	switch r {
	case RiskTierCritical:
		return 4
	case RiskTierHigh:
		return 3
	case RiskTierModerate:
		return 2
	case RiskTierLow:
		return 1
	default:
		return 0
	}
}

// RiskStratificationMethod defines how risk is calculated
type RiskStratificationMethod string

const (
	RiskMethodRules RiskStratificationMethod = "RULES"
	RiskMethodScore RiskStratificationMethod = "SCORE"
	RiskMethodML    RiskStratificationMethod = "ML"
)

// Registry represents a disease registry definition
type Registry struct {
	ID                  uuid.UUID               `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code                RegistryCode            `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Name                string                  `gorm:"size:200;not null" json:"name"`
	Description         string                  `gorm:"type:text" json:"description,omitempty"`
	Category            RegistryCategory        `gorm:"size:50;not null;default:CHRONIC" json:"category"`
	AutoEnroll          bool                    `gorm:"default:true" json:"auto_enroll"`
	Active              bool                    `gorm:"default:true" json:"active"`
	InclusionCriteria   CriteriaGroupSlice      `gorm:"type:jsonb;default:'[]'" json:"inclusion_criteria"`
	ExclusionCriteria   CriteriaGroupSlice      `gorm:"type:jsonb;default:'[]'" json:"exclusion_criteria"`
	RiskStratification  *RiskStratificationConfig `gorm:"type:jsonb" json:"risk_stratification,omitempty"`
	CareGapMeasures     StringSlice             `gorm:"type:jsonb;default:'[]'" json:"care_gap_measures,omitempty"`
	Metadata            JSONMap                 `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt           time.Time               `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time               `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for Registry
func (Registry) TableName() string {
	return "registries"
}

// RiskStratificationConfig defines how risk is calculated for a registry
type RiskStratificationConfig struct {
	Method    RiskStratificationMethod `json:"method"`
	ScoreType string                   `json:"score_type,omitempty"` // e.g., "HAS-BLED", "ASCVD"
	Rules     []RiskRule               `json:"rules,omitempty"`
	Thresholds map[string]interface{}  `json:"thresholds,omitempty"`
}

// RiskRule defines a rule for risk tier assignment
type RiskRule struct {
	Tier     RiskTier        `json:"tier"`
	Priority int             `json:"priority"`
	Criteria []CriteriaGroup `json:"criteria"`
}

// RegistryStats holds statistics for a registry
type RegistryStats struct {
	RegistryCode   RegistryCode `json:"registry_code"`
	TotalEnrolled  int64        `json:"total_enrolled"`
	ActiveCount    int64        `json:"active_count"`
	PendingCount   int64        `json:"pending_count"`
	LowRiskCount   int64        `json:"low_risk_count"`
	ModerateCount  int64        `json:"moderate_risk_count"`
	HighRiskCount  int64        `json:"high_risk_count"`
	CriticalCount  int64        `json:"critical_risk_count"`
	CareGapCount   int64        `json:"care_gap_count"`
	LastUpdated    time.Time    `json:"last_updated"`
}

// CreateRegistryRequest represents the request body for creating a registry
type CreateRegistryRequest struct {
	Code               RegistryCode              `json:"code" binding:"required"`
	Name               string                    `json:"name" binding:"required"`
	Description        string                    `json:"description,omitempty"`
	Category           RegistryCategory          `json:"category,omitempty"`
	AutoEnroll         bool                      `json:"auto_enroll"`
	InclusionCriteria  []CriteriaGroup           `json:"inclusion_criteria,omitempty"`
	ExclusionCriteria  []CriteriaGroup           `json:"exclusion_criteria,omitempty"`
	RiskStratification *RiskStratificationConfig `json:"risk_stratification,omitempty"`
	CareGapMeasures    []string                  `json:"care_gap_measures,omitempty"`
}

// RegistryResponse wraps a registry for API responses
type RegistryResponse struct {
	Success bool      `json:"success"`
	Data    *Registry `json:"data,omitempty"`
	Error   string    `json:"error,omitempty"`
}

// RegistryListResponse wraps a list of registries for API responses
type RegistryListResponse struct {
	Success bool       `json:"success"`
	Data    []Registry `json:"data,omitempty"`
	Total   int64      `json:"total"`
	Error   string     `json:"error,omitempty"`
}

// CriteriaGroupSlice is a custom type for JSONB array of CriteriaGroup
type CriteriaGroupSlice []CriteriaGroup

// Value implements the driver.Valuer interface
func (c CriteriaGroupSlice) Value() (driver.Value, error) {
	if c == nil {
		return "[]", nil
	}
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface
func (c *CriteriaGroupSlice) Scan(value interface{}) error {
	if value == nil {
		*c = CriteriaGroupSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("CriteriaGroupSlice.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, c)
}

// StringSlice is a custom type for JSONB array of strings
type StringSlice []string

// Value implements the driver.Valuer interface
func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("StringSlice.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, s)
}

// JSONMap is a custom type for JSONB map
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface
func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("JSONMap.Scan: unsupported type")
	}
	return json.Unmarshal(bytes, m)
}
