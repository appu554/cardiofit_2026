package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ContextModifier is a registry entry representing a condition that modifies
// clinical significance of a drug reaction in a specific HPI node.
type ContextModifier struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	// Modifier identity
	ModifierType     string `gorm:"size:30;not null;index;check:modifier_type IN ('POPULATION','COMORBIDITY','CONCOMITANT_DRUG','LAB_VALUE','TEMPORAL')" json:"modifier_type"`
	ModifierValue    string `gorm:"size:200;not null" json:"modifier_value"`

	// Target — which HPI node and differential this affects
	TargetNodeID     string `gorm:"size:20;not null;index" json:"target_node_id"`
	DrugClassTrigger string `gorm:"size:50;not null;index" json:"drug_class_trigger"`

	// Effect
	Effect             string          `gorm:"size:30;not null;check:effect IN ('INCREASE_PRIOR','DECREASE_PRIOR')" json:"effect"`
	TargetDifferential string          `gorm:"size:100;not null" json:"target_differential"`
	Magnitude          decimal.Decimal `gorm:"type:decimal(5,4);not null" json:"magnitude"`

	// Structured thresholds for LAB_VALUE modifiers
	LabParameter string  `gorm:"size:30" json:"lab_parameter,omitempty"`
	LabOperator  string  `gorm:"size:5" json:"lab_operator,omitempty"`
	LabThreshold float64 `gorm:"type:decimal(10,4)" json:"lab_threshold,omitempty"`
	LabUnit      string  `gorm:"size:20" json:"lab_unit,omitempty"`

	// Completeness and confidence
	CompletenessGrade string          `gorm:"size:10;not null;default:'STUB';check:completeness_grade IN ('FULL','PARTIAL','STUB')" json:"completeness_grade"`
	Confidence        decimal.Decimal `gorm:"type:decimal(3,2);default:0.50" json:"confidence"`

	// Context modifier rule mapping (e.g., CM03, CM07)
	ContextModifierRule string `gorm:"size:20" json:"context_modifier_rule,omitempty"`

	// Source
	Source   string `gorm:"size:30;not null;default:'PIPELINE';check:source IN ('PIPELINE','SPL','MANUAL_CURATED')" json:"source"`

	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate generates a UUID.
func (cm *ContextModifier) BeforeCreate(tx *gorm.DB) error {
	if cm.ID == uuid.Nil {
		cm.ID = uuid.New()
	}
	return nil
}

// EffectiveMagnitude returns the magnitude adjusted for completeness grade.
// FULL = 1.0x, PARTIAL = 0.7x, STUB = 0.0x (ignored).
func (cm *ContextModifier) EffectiveMagnitude() float64 {
	mag, _ := cm.Magnitude.Float64()
	switch cm.CompletenessGrade {
	case "FULL":
		return mag
	case "PARTIAL":
		return mag * 0.7
	default:
		return 0.0
	}
}

// Confidence threshold constants (Section 7.3 of KB-20 spec)
const (
	ConfidenceHighConf  = 0.85
	ConfidenceCalibrate = 0.70
)
