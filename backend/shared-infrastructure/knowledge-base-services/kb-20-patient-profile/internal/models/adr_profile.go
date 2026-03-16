package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AdverseReactionProfile stores an ADR for a drug with onset window, contextual modifiers,
// and completeness grading. Mirrors the Python schema at shared/extraction/schemas/kb20_contextual.py.
type AdverseReactionProfile struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Drug identification
	RxNormCode string `gorm:"size:50;index" json:"rxnorm_code"`
	DrugName   string `gorm:"size:200;not null" json:"drug_name"`
	DrugClass  string `gorm:"size:50;not null;index" json:"drug_class"`

	// Reaction
	Reaction      string `gorm:"type:text;not null" json:"reaction"`
	ReactionSNOMED string `gorm:"size:50" json:"reaction_snomed,omitempty"`
	Mechanism     string `gorm:"type:text" json:"mechanism,omitempty"`
	Symptom       string `gorm:"size:100" json:"symptom,omitempty"`

	// Onset window
	OnsetWindow   string `gorm:"size:50" json:"onset_window,omitempty"`
	OnsetCategory string `gorm:"size:20;check:onset_category IN ('IMMEDIATE','ACUTE','SUBACUTE','CHRONIC','DELAYED','IDIOSYNCRATIC','')" json:"onset_category,omitempty"`

	// Frequency and severity
	Frequency string `gorm:"size:20" json:"frequency,omitempty"`
	Severity  string `gorm:"size:20" json:"severity,omitempty"`

	// Risk factors
	RiskFactors pq.StringArray `gorm:"type:text[]" json:"risk_factors,omitempty"`

	// Context modifier rule (JSONB for flexibility)
	ContextModifierRule string `gorm:"type:jsonb" json:"context_modifier_rule,omitempty"`

	// Source: PIPELINE, SPL, or MANUAL_CURATED (F-01)
	Source     string          `gorm:"size:30;not null;default:'PIPELINE'" json:"source"`
	Confidence decimal.Decimal `gorm:"type:decimal(3,2);default:0.50" json:"confidence"`

	// Completeness grading (auto-computed on write)
	CompletenessGrade string `gorm:"size:10;not null;default:'STUB';check:completeness_grade IN ('FULL','PARTIAL','STUB')" json:"completeness_grade"`

	// Source snippet for audit
	SourceSnippet string `gorm:"type:text" json:"source_snippet,omitempty"`

	// Governance provenance
	SourceAuthority string `gorm:"size:50" json:"source_authority,omitempty"`
	SourceDocument  string `gorm:"size:200" json:"source_document,omitempty"`
	SourceSection   string `gorm:"size:100" json:"source_section,omitempty"`
	EvidenceLevel   string `gorm:"size:10" json:"evidence_level,omitempty"`

	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate generates UUID and computes completeness grade.
func (a *AdverseReactionProfile) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.ComputeCompletenessGrade()
	return nil
}

// BeforeUpdate recomputes completeness grade.
func (a *AdverseReactionProfile) BeforeUpdate(tx *gorm.DB) error {
	a.ComputeCompletenessGrade()
	return nil
}

// ComputeCompletenessGrade determines FULL/PARTIAL/STUB based on populated fields.
// Mirrors the Python logic in kb20_contextual.py.
func (a *AdverseReactionProfile) ComputeCompletenessGrade() {
	hasDrug := a.DrugName != "" && a.Reaction != ""
	hasOnset := a.OnsetWindow != ""
	hasMechanism := a.Mechanism != ""
	hasModifier := a.ContextModifierRule != "" && a.ContextModifierRule != "{}" && a.ContextModifierRule != "null"

	conf, _ := a.Confidence.Float64()

	if hasDrug && hasOnset && hasMechanism && hasModifier && conf >= 0.70 {
		a.CompletenessGrade = "FULL"
	} else if hasDrug && (hasOnset || hasMechanism) {
		a.CompletenessGrade = "PARTIAL"
	} else {
		a.CompletenessGrade = "STUB"
	}
}
