package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GuidelineDocument represents a clinical guideline document
type GuidelineDocument struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	GuidelineID string    `gorm:"uniqueIndex;not null" json:"guideline_id"` // e.g., "ADA-DM-2025"
	
	// Source information
	Source          GuidelineSource `gorm:"type:jsonb" json:"source"`
	Version         string          `gorm:"not null" json:"version"`
	EffectiveDate   time.Time       `gorm:"not null;index" json:"effective_date"`
	SupersededDate  *time.Time      `json:"superseded_date,omitempty"`
	Supersedes      *string         `json:"supersedes,omitempty"`
	
	// Clinical metadata
	Condition       GuidelineCondition `gorm:"type:jsonb" json:"condition"`
	Publication     *Publication       `gorm:"type:jsonb" json:"publication,omitempty"`
	
	// Content
	Recommendations []Recommendation `gorm:"foreignKey:GuidelineID;references:ID" json:"recommendations"`
	
	// Status and governance
	Status          string    `gorm:"default:'active';index" json:"status"` // draft, review, approved, active, deprecated
	IsActive        bool      `gorm:"default:true;index" json:"is_active"`
	DigitalSignature *string  `json:"digital_signature,omitempty"`
	
	// Audit fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	CreatedBy string         `json:"created_by"`
	UpdatedBy string         `json:"updated_by"`
}

// GuidelineSource represents the source organization of a guideline
type GuidelineSource struct {
	Organization string `json:"organization"` // e.g., "ADA", "ESC/ESH"
	FullName     string `json:"full_name"`    // e.g., "American Diabetes Association"
	Country      string `json:"country"`      // e.g., "United States"
	Region       string `json:"region"`       // e.g., "US", "EU", "AU", "WHO"
}

// GuidelineCondition represents the medical condition covered
type GuidelineCondition struct {
	Primary      string   `json:"primary"`       // e.g., "Type 2 Diabetes Mellitus"
	Secondary    []string `json:"secondary"`     // Additional conditions
	ICD10Codes   []string `json:"icd10_codes"`   // e.g., ["E11", "E11.9"]
	SnomedCodes  []string `json:"snomed_codes"`  // e.g., ["44054006"]
}

// Publication represents publication metadata
type Publication struct {
	DOI     *string `json:"doi,omitempty"`
	PMID    *string `json:"pmid,omitempty"`
	URL     *string `json:"url,omitempty"`
	Journal *string `json:"journal,omitempty"`
	Year    *int    `json:"year,omitempty"`
}

// Recommendation represents a single clinical recommendation
type Recommendation struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	GuidelineID uuid.UUID `gorm:"type:uuid;not null;index" json:"guideline_id"`
	
	// Recommendation identification
	RecID       string `gorm:"uniqueIndex;not null" json:"rec_id"` // e.g., "ADA-DM-2025-001"
	Domain      string `gorm:"not null;index" json:"domain"`       // e.g., "diagnosis", "treatment"
	Subdomain   string `json:"subdomain"`                          // e.g., "criteria", "targets"
	
	// Recommendation content
	Recommendation string `gorm:"type:text;not null" json:"recommendation"`
	
	// Evidence grading
	EvidenceGrade           string  `gorm:"not null;index" json:"evidence_grade"` // A, B, C, D, Expert Opinion, Good Practice Point
	Strength                *string `json:"strength,omitempty"`                   // Strong, Conditional, Weak, Against
	ClassOfRecommendation   *string `json:"class_of_recommendation,omitempty"`    // I, IIa, IIb, III
	LevelOfEvidence         *string `json:"level_of_evidence,omitempty"`          // A, B-R, B-NR, C-LD, C-EO
	
	// Applicability
	Applicability   *Applicability `gorm:"type:jsonb" json:"applicability,omitempty"`
	
	// Cross-KB linkages
	LinkedKBRefs    *CrossKBLinks `gorm:"type:jsonb" json:"linked_kb_refs,omitempty"`
	
	// Outcome metrics
	Metrics         *OutcomeMetrics `gorm:"type:jsonb" json:"metrics,omitempty"`
	
	// Audit fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Applicability represents the applicability of a recommendation
type Applicability struct {
	Populations        []string                  `json:"populations,omitempty"`
	Exclusions         []string                  `json:"exclusions,omitempty"`
	SpecialPopulations map[string]string         `json:"special_populations,omitempty"` // e.g., "elderly": "Consider <8% if frail"
}

// CrossKBLinks represents links to other knowledge bases
type CrossKBLinks struct {
	KB1Dosing      []string `json:"kb1_dosing,omitempty"`      // Links to KB-1 dosing rules
	KB2Phenotypes  []string `json:"kb2_phenotypes,omitempty"`  // Links to KB-2 patient phenotypes  
	KB4Safety      []string `json:"kb4_safety,omitempty"`      // Links to KB-4 safety rules
	KB5Interactions []string `json:"kb5_interactions,omitempty"` // Links to KB-5 drug interactions
	KB6Formulary   []string `json:"kb6_formulary,omitempty"`   // Links to KB-6 formulary
	KB7Terminology []string `json:"kb7_terminology,omitempty"` // Links to KB-7 terminology
}

// OutcomeMetrics represents clinical outcome metrics
type OutcomeMetrics struct {
	OutcomeMeasure           *string  `json:"outcome_measure,omitempty"`
	NNT                      *float64 `json:"nnt,omitempty"`                      // Number needed to treat
	NNH                      *float64 `json:"nnh,omitempty"`                      // Number needed to harm
	AbsoluteRiskReduction    *float64 `json:"absolute_risk_reduction,omitempty"`
	RelativeRiskReduction    *float64 `json:"relative_risk_reduction,omitempty"`
}

// RegionalProfile represents regional preferences and configurations
type RegionalProfile struct {
	ID     uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Region string    `gorm:"uniqueIndex;not null" json:"region"` // US, EU, AU, WHO
	
	// Regional preferences
	PrimarySources       []string          `gorm:"type:jsonb" json:"primary_sources"`       // e.g., ["ADA", "ACC/AHA"]
	MeasurementUnits     map[string]string `gorm:"type:jsonb" json:"measurement_units"`     // e.g., "glucose": "mg/dL"
	RegulatoryFramework  string            `json:"regulatory_framework"`                    // FDA, EMA, TGA
	
	// Special considerations
	Applicability        *string           `json:"applicability,omitempty"`                // e.g., "resource_limited_settings"
	Focus                *string           `json:"focus,omitempty"`                        // e.g., "essential_medicines"
	
	// Audit fields  
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// GuidelineVersion represents version tracking for guidelines
type GuidelineVersion struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	GuidelineID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"guideline_id"`
	Version       string     `gorm:"not null" json:"version"`
	ChangeLog     string     `gorm:"type:text" json:"change_log"`
	IsActive      bool       `gorm:"default:false" json:"is_active"`
	IsDraft       bool       `gorm:"default:true" json:"is_draft"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
	DeprecatedAt  *time.Time `json:"deprecated_at,omitempty"`
	
	// Content snapshot (full guideline as JSON)
	GuidelineSnapshot string `gorm:"type:jsonb" json:"guideline_snapshot"`
	
	// Audit fields
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	CreatedBy string         `json:"created_by"`
	UpdatedBy string         `json:"updated_by"`
}

// GORM custom types for JSON handling

// Scan implements sql.Scanner for GuidelineSource
func (gs *GuidelineSource) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan GuidelineSource")
	}
	return json.Unmarshal(bytes, gs)
}

// Value implements driver.Valuer for GuidelineSource
func (gs GuidelineSource) Value() (driver.Value, error) {
	return json.Marshal(gs)
}

// Scan implements sql.Scanner for GuidelineCondition
func (gc *GuidelineCondition) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan GuidelineCondition")
	}
	return json.Unmarshal(bytes, gc)
}

// Value implements driver.Valuer for GuidelineCondition  
func (gc GuidelineCondition) Value() (driver.Value, error) {
	return json.Marshal(gc)
}

// Scan implements sql.Scanner for Publication
func (p *Publication) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan Publication")
	}
	return json.Unmarshal(bytes, p)
}

// Value implements driver.Valuer for Publication
func (p Publication) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements sql.Scanner for Applicability
func (a *Applicability) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan Applicability")
	}
	return json.Unmarshal(bytes, a)
}

// Value implements driver.Valuer for Applicability
func (a Applicability) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan implements sql.Scanner for CrossKBLinks
func (ckl *CrossKBLinks) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan CrossKBLinks")
	}
	return json.Unmarshal(bytes, ckl)
}

// Value implements driver.Valuer for CrossKBLinks
func (ckl CrossKBLinks) Value() (driver.Value, error) {
	return json.Marshal(ckl)
}

// Scan implements sql.Scanner for OutcomeMetrics
func (om *OutcomeMetrics) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("cannot scan OutcomeMetrics")
	}
	return json.Unmarshal(bytes, om)
}

// Value implements driver.Valuer for OutcomeMetrics
func (om OutcomeMetrics) Value() (driver.Value, error) {
	return json.Marshal(om)
}

// Helper methods

// IsEffective checks if the guideline is currently effective
func (gd *GuidelineDocument) IsEffective() bool {
	now := time.Now()
	if gd.EffectiveDate.After(now) {
		return false
	}
	if gd.SupersededDate != nil && gd.SupersededDate.Before(now) {
		return false
	}
	return gd.IsActive && gd.Status == "active"
}

// GetRegionPriority returns the priority for regional matching
func (gd *GuidelineDocument) GetRegionPriority(preferredRegion string) int {
	if gd.Source.Region == preferredRegion {
		return 1 // Highest priority - exact match
	}
	if gd.Source.Region == "WHO" {
		return 3 // Lowest priority - global default
	}
	return 2 // Medium priority - different region but not WHO
}

// HasCrossKBLinks checks if recommendation has any cross-KB links
func (r *Recommendation) HasCrossKBLinks() bool {
	if r.LinkedKBRefs == nil {
		return false
	}
	return len(r.LinkedKBRefs.KB1Dosing) > 0 ||
		len(r.LinkedKBRefs.KB2Phenotypes) > 0 ||
		len(r.LinkedKBRefs.KB4Safety) > 0 ||
		len(r.LinkedKBRefs.KB5Interactions) > 0 ||
		len(r.LinkedKBRefs.KB6Formulary) > 0 ||
		len(r.LinkedKBRefs.KB7Terminology) > 0
}

// GetEvidenceScore returns numeric score for evidence grade (higher = stronger)
func (r *Recommendation) GetEvidenceScore() int {
	switch r.EvidenceGrade {
	case "A":
		return 4
	case "B":
		return 3
	case "C":
		return 2
	case "D":
		return 1
	case "Expert Opinion":
		return 0
	case "Good Practice Point":
		return 0
	default:
		return 0
	}
}