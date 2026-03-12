package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CompositeCardSignal aggregates multiple concurrent decision cards into a
// single composite signal for the clinician dashboard.
type CompositeCardSignal struct {
	CompositeID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"composite_id"`
	PatientID           uuid.UUID `gorm:"type:uuid;index;not null" json:"patient_id"`
	CardIDs             JSONB     `gorm:"type:jsonb" json:"card_ids"`
	MostRestrictiveGate MCUGate   `gorm:"type:varchar(10)" json:"most_restrictive_gate"`
	RecurrenceCount     int       `json:"recurrence_count"`
	UrgencyUpgraded     bool      `gorm:"default:false" json:"urgency_upgraded"`
	SynthesisSummaryEn  string    `gorm:"type:text" json:"synthesis_summary_en"`
	SynthesisSummaryHi  string    `gorm:"type:text" json:"synthesis_summary_hi"`
	WindowStart         time.Time `json:"window_start"`
	WindowEnd           time.Time `json:"window_end"`
	CreatedAt           time.Time `json:"created_at"`
}

// TableName sets the PostgreSQL table name.
func (CompositeCardSignal) TableName() string { return "composite_card_signals" }

// BeforeCreate generates a UUID primary key if not already set.
func (c *CompositeCardSignal) BeforeCreate(tx *gorm.DB) error {
	if c.CompositeID == uuid.Nil {
		c.CompositeID = uuid.New()
	}
	return nil
}
