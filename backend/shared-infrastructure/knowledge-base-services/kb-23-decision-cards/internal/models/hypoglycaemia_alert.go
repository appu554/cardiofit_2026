package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HypoglycaemiaAlert records detected or predicted hypoglycaemia events
// and tracks the decision card generated in response.
type HypoglycaemiaAlert struct {
	AlertID         uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"alert_id"`
	PatientID       uuid.UUID             `gorm:"type:uuid;index;not null" json:"patient_id"`
	Source          HypoglycaemiaSource   `gorm:"type:varchar(20);not null" json:"source"`
	GlucoseMmolL    float64              `json:"glucose_mmol_l"`
	DurationMinutes *int                 `json:"duration_minutes,omitempty"`
	Severity        HypoglycaemiaSeverity `gorm:"type:varchar(10);not null" json:"severity"`
	PredictedAtHours *float64            `json:"predicted_at_hours,omitempty"`
	HaltSource      HaltSource           `gorm:"type:varchar(10)" json:"halt_source"`
	GeneratedCardID *uuid.UUID           `gorm:"type:uuid" json:"generated_card_id,omitempty"`
	EventTimestamp  time.Time            `json:"event_timestamp"`
	ProcessedAt     time.Time            `json:"processed_at"`
}

// TableName sets the PostgreSQL table name.
func (HypoglycaemiaAlert) TableName() string { return "hypoglycaemia_alerts" }

// BeforeCreate generates a UUID primary key if not already set.
func (a *HypoglycaemiaAlert) BeforeCreate(tx *gorm.DB) error {
	if a.AlertID == uuid.Nil {
		a.AlertID = uuid.New()
	}
	return nil
}
