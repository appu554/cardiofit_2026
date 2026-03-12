package models

import (
	"time"

	"github.com/google/uuid"
)

// SafetyLevel represents the severity of a safety flag.
type SafetyLevel string

const (
	SafetyImmediate SafetyLevel = "IMMEDIATE" // < 30 min action required
	SafetyUrgent    SafetyLevel = "URGENT"    // < 4h action required
	SafetyWarn      SafetyLevel = "WARN"      // Next appointment
)

// SafetyFlag records a fired safety trigger during an HPI session.
// F-02: evaluated by the parallel SafetyEngine goroutine.
type SafetyFlag struct {
	FlagID    string    `gorm:"type:varchar(64);primaryKey" json:"flag_id"`
	SessionID uuid.UUID `gorm:"type:uuid;primaryKey;index" json:"session_id"`

	Severity          SafetyLevel `gorm:"type:varchar(16);not null" json:"severity"`
	TriggerExpression string      `gorm:"type:text;not null" json:"trigger_expression"`

	// Top-3 differentials with posteriors at time of flag; context for KB-23
	DifferentialContext JSONB `gorm:"type:jsonb;default:'[]'" json:"differential_context"`

	// Plain-language action for KB-23 Decision Card
	RecommendedAction string `gorm:"type:text;not null" json:"recommended_action"`

	// N-02: KB-5 medication safety enrichment (nullable)
	MedicationSafetyContext JSONB `gorm:"type:jsonb" json:"medication_safety_context,omitempty"`

	// true when event published to KB-19 (IMMEDIATE triggers)
	PublishedToKB19 bool `gorm:"type:bool;default:false" json:"published_to_kb19"`

	FiredAt time.Time `gorm:"type:timestamptz;index;not null;autoCreateTime" json:"fired_at"`
}

func (SafetyFlag) TableName() string { return "safety_flags" }

// IsImmediate returns true if this flag requires immediate action.
func (f *SafetyFlag) IsImmediate() bool {
	return f.Severity == SafetyImmediate
}

// IsUrgentOrImmediate returns true for flags that trigger KB-5 medication safety check.
func (f *SafetyFlag) IsUrgentOrImmediate() bool {
	return f.Severity == SafetyImmediate || f.Severity == SafetyUrgent
}
