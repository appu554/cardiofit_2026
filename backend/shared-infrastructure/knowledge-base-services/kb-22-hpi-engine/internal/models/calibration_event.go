package models

import (
	"time"

	"github.com/google/uuid"
)

// CalibrationSourceTier represents the three-tier calibration strategy (CC-4).
type CalibrationSourceTier string

const (
	CalibrationTierExpertPanel CalibrationSourceTier = "EXPERT_PANEL"   // Tier A: Month 0-6
	CalibrationTierBayesBlend  CalibrationSourceTier = "BAYESIAN_BLEND" // Tier B: Month 6-18
	CalibrationTierDataDriven  CalibrationSourceTier = "DATA_DRIVEN"    // Tier C: Month 18+
)

// CalibrationEvent is an immutable log entry for every LR/prior adjustment (E04).
// Every number in every YAML node should trace to one of these events.
// This is SEPARATE from CalibrationRecord which tracks concordance.
type CalibrationEvent struct {
	EventID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"event_id"`
	NodeID      string    `gorm:"type:varchar(64);index;not null" json:"node_id"`
	NodeVersion string    `gorm:"type:varchar(32);not null" json:"node_version"`

	// What was changed
	ElementType  string  `gorm:"type:varchar(32);not null" json:"element_type"` // LR_POSITIVE, LR_NEGATIVE, PRIOR, CM_MAGNITUDE, SAFETY_FLOOR
	ElementKey   string  `gorm:"type:varchar(128);not null" json:"element_key"` // e.g. "Q001:OH" (question:differential)
	StratumLabel *string `gorm:"type:varchar(32);index" json:"stratum_label,omitempty"`

	// Old and new values
	OldValue float64 `gorm:"type:float8;not null" json:"old_value"`
	NewValue float64 `gorm:"type:float8;not null" json:"new_value"`

	// Source and justification
	SourceTier   CalibrationSourceTier `gorm:"type:varchar(32);not null" json:"source_tier"`
	SourceID     *uuid.UUID            `gorm:"type:uuid;index" json:"source_id,omitempty"` // FK to clinical_sources
	SampleSize   *int                  `gorm:"type:int" json:"sample_size,omitempty"`       // for Tier B/C
	WFactor      *float64              `gorm:"type:float8" json:"w_factor,omitempty"`       // Tier B blending weight
	Deviation    *float64              `gorm:"type:float8" json:"deviation,omitempty"`      // observed vs expected
	Rationale    string                `gorm:"type:text;not null" json:"rationale"`

	// Governance
	ApprovedBy   string  `gorm:"type:varchar(128);not null" json:"approved_by"`
	PanelMembers *string `gorm:"type:text" json:"panel_members,omitempty"` // Tier A: comma-separated

	CreatedAt time.Time `gorm:"type:timestamptz;not null;autoCreateTime;index" json:"created_at"`
}

func (CalibrationEvent) TableName() string { return "calibration_events" }

// MaxAdjustmentPerCycle is the Tier A constraint: ±30% LR adjustment per review cycle.
const MaxAdjustmentPerCycle = 0.30
