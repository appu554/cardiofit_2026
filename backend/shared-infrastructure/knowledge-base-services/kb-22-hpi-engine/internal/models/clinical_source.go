package models

import (
	"time"

	"github.com/google/uuid"
)

// StudyType classifies the clinical evidence source.
type StudyType string

const (
	StudyTypeMeta           StudyType = "META_ANALYSIS"
	StudyTypeRCT            StudyType = "RCT"
	StudyTypeCohort         StudyType = "COHORT"
	StudyTypeCaseControl    StudyType = "CASE_CONTROL"
	StudyTypeCrossSectional StudyType = "CROSS_SECTIONAL"
	StudyTypeCaseSeries     StudyType = "CASE_SERIES"
	StudyTypeExpertOpinion  StudyType = "EXPERT_OPINION"
	StudyTypeGuideline      StudyType = "GUIDELINE"
)

// QualityGrade for the clinical source (Oxford CEBM-inspired).
type QualityGrade string

const (
	QualityGradeA QualityGrade = "A" // High: meta-analysis / large RCT
	QualityGradeB QualityGrade = "B" // Moderate: smaller RCT / cohort
	QualityGradeC QualityGrade = "C" // Low: case-control / case series
	QualityGradeD QualityGrade = "D" // Very low: expert opinion
)

// ClinicalSource is a canonical source record for LR provenance tracking (E06/DIZ-7).
// Referenced by ElementAttribution for linking YAML fields to evidence.
type ClinicalSource struct {
	SourceID     uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"source_id"`
	PubMedID     *string      `gorm:"type:varchar(32);uniqueIndex" json:"pubmed_id,omitempty"`
	DOI          *string      `gorm:"type:varchar(128)" json:"doi,omitempty"`
	Journal      string       `gorm:"type:varchar(256);not null" json:"journal"`
	Year         int          `gorm:"type:int;not null;index" json:"year"`
	Title        string       `gorm:"type:text;not null" json:"title"`
	Authors      string       `gorm:"type:text;not null" json:"authors"`
	StudyType    StudyType    `gorm:"type:varchar(32);not null" json:"study_type"`
	Population   string       `gorm:"type:text" json:"population,omitempty"`
	QualityGrade QualityGrade `gorm:"type:varchar(4);not null" json:"quality_grade"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
}

func (ClinicalSource) TableName() string { return "clinical_sources" }

// ElementAttribution links a YAML field (LR, prior, CM delta) to its source.
type ElementAttribution struct {
	AttributionID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"attribution_id"`
	SourceID      uuid.UUID `gorm:"type:uuid;index;not null" json:"source_id"` // FK to clinical_sources

	NodeID       string  `gorm:"type:varchar(64);index;not null" json:"node_id"`
	ElementType  string  `gorm:"type:varchar(32);not null" json:"element_type"` // LR_POSITIVE, LR_NEGATIVE, PRIOR, CM_MAGNITUDE
	ElementKey   string  `gorm:"type:varchar(128);not null" json:"element_key"` // e.g. "Q001:OH"
	StratumLabel *string `gorm:"type:varchar(32)" json:"stratum_label,omitempty"`

	ConfidenceLevel string  `gorm:"type:varchar(16);not null" json:"confidence_level"` // HIGH, MODERATE, LOW, EXTRAPOLATED
	Notes           *string `gorm:"type:text" json:"notes,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
}

func (ElementAttribution) TableName() string { return "element_attributions" }
