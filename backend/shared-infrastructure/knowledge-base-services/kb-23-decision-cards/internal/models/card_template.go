package models

import (
	"time"
)

// CardTemplate is loaded from YAML and stored in the database for audit reference.
type CardTemplate struct {
	TemplateID                  string     `gorm:"primaryKey" json:"template_id" yaml:"template_id"`
	NodeID                      string     `gorm:"index;not null" json:"node_id" yaml:"node_id"`
	DifferentialID              string     `json:"differential_id" yaml:"differential_id"`
	TemplateVersion             string     `json:"template_version" yaml:"version"`
	ContentSHA256               string     `json:"content_sha256"`
	ConfidenceThresholds        JSONB      `gorm:"type:jsonb" json:"confidence_thresholds" yaml:"-"`
	MCUGateDefault              MCUGate    `gorm:"type:varchar(10)" json:"mcu_gate_default" yaml:"mcu_gate_default"`
	RecommendationsCount        int        `json:"recommendations_count"`
	HasSafetyInstructions       bool       `json:"has_safety_instructions"`
	RequiresDoseAdjustmentNotes bool       `json:"requires_dose_adjustment_notes"`
	ClinicalReviewer            string     `json:"clinical_reviewer" yaml:"clinical_reviewer"`
	ApprovedAt                  *time.Time `json:"approved_at,omitempty"`
	LoadedAt                    time.Time  `json:"loaded_at"`

	// YAML-only fields (not persisted to DB)
	Thresholds      TemplateThresholds       `gorm:"-" json:"-" yaml:"confidence_thresholds"`
	Recommendations []TemplateRecommendation `gorm:"-" json:"-" yaml:"recommendations"`
	Fragments       []TemplateFragment       `gorm:"-" json:"-" yaml:"fragments"`
	GateRules       []GateRule               `gorm:"-" json:"-" yaml:"gate_rules"`
}

// TableName sets the PostgreSQL table name.
func (CardTemplate) TableName() string { return "card_templates" }

// ---------------------------------------------------------------------------
// YAML sub-structures used during template loading
// ---------------------------------------------------------------------------

// TemplateThresholds defines posterior probability cutoffs for confidence tiers.
type TemplateThresholds struct {
	FirmPosterior        float64 `yaml:"firm_posterior"`
	FirmMedicationChange float64 `yaml:"firm_medication_change"`
	ProbablePosterior    float64 `yaml:"probable_posterior"`
	PossiblePosterior    float64 `yaml:"possible_posterior"`
}

// TemplateRecommendation is a recommendation definition within a template YAML.
type TemplateRecommendation struct {
	RecType                RecommendationType `yaml:"rec_type"`
	Urgency                Urgency            `yaml:"urgency"`
	Target                 string             `yaml:"target"`
	ActionTextEn           string             `yaml:"action_text_en"`
	ActionTextHi           string             `yaml:"action_text_hi"`
	RationaleEn            string             `yaml:"rationale_en"`
	GuidelineRef           string             `yaml:"guideline_ref"`
	ConfidenceTierRequired ConfidenceTier     `yaml:"confidence_tier_required"`
	BypassesConfidenceGate bool               `yaml:"bypasses_confidence_gate"`
	TriggerConditionEn     string             `yaml:"trigger_condition_en,omitempty"`
	TriggerConditionHi     string             `yaml:"trigger_condition_hi,omitempty"`
	SortOrder              int                `yaml:"sort_order"`

	// CTL Panel 2: Guideline condition criteria authored in template YAML.
	// Each criterion describes a clinical condition that must be met for the
	// recommendation to be fully guideline-aligned.
	ConditionCriteria []ConditionCriterionDef `yaml:"condition_criteria,omitempty"`
}

// ConditionCriterionDef is a guideline criterion definition within a template YAML.
type ConditionCriterionDef struct {
	CriterionID string `yaml:"criterion_id"`
	Description string `yaml:"description"`
	// Condition is a simple token evaluated at card-build time (e.g. "EGFR_LOW", "TIER_FIRM")
	Condition string `yaml:"condition"`
}

// TemplateFragment is a pre-authored summary text block within a template.
type TemplateFragment struct {
	FragmentID                string       `yaml:"fragment_id"`
	FragmentType              FragmentType `yaml:"fragment_type"`
	TextEn                    string       `yaml:"text_en"`
	TextHi                    string       `yaml:"text_hi"`
	TextLocal                 string       `yaml:"text_local,omitempty"`
	LocaleCode                string       `yaml:"locale_code,omitempty"`
	PatientAdvocateReviewedBy string       `yaml:"patient_advocate_reviewed_by,omitempty"`
	ReadingLevelValidated     bool         `yaml:"reading_level_validated"`
	GuidelineRef              string       `yaml:"guideline_ref,omitempty"`
	Version                   string       `yaml:"version"`
}

// GateRule defines a condition-based MCU gate override within a template.
type GateRule struct {
	Condition       string  `yaml:"condition"`
	Gate            MCUGate `yaml:"gate"`
	Rationale       string  `yaml:"rationale"`
	AdjustmentNotes string  `yaml:"adjustment_notes,omitempty"`
}
