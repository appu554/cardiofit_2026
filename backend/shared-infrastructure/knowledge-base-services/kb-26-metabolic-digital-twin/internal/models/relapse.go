package models

import (
	"time"

	"github.com/google/uuid"
)

// MRINadir tracks the best (lowest) MRI score achieved per patient.
// Updated after every MRI computation. Used for relapse trigger (MRI rises >15 from nadir).
type MRINadir struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID    uuid.UUID  `gorm:"uniqueIndex;not null" json:"patient_id"`
	NadirScore   float64    `gorm:"not null" json:"nadir_score"`
	NadirDate    time.Time  `gorm:"not null" json:"nadir_date"`
	HbA1cNadir   *float64   `json:"hba1c_nadir,omitempty"`
	HbA1cNadirAt *time.Time `json:"hba1c_nadir_at,omitempty"`
	UpdatedAt    time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

// RelapseEvent records a detected relapse trigger and the action taken.
type RelapseEvent struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID    uuid.UUID `gorm:"index;not null" json:"patient_id"`
	TriggerType  string    `gorm:"type:varchar(50);not null" json:"trigger_type"` // MRI_RISE, HBA1C_RISE
	TriggerValue float64   `gorm:"not null" json:"trigger_value"`
	NadirValue   float64   `gorm:"not null" json:"nadir_value"`
	CurrentValue float64   `gorm:"not null" json:"current_value"`
	ActionTaken  string    `gorm:"type:varchar(50)" json:"action_taken"` // RECORRECTION_ACTIVATED, ESCALATED
	DetectedAt   time.Time `gorm:"not null;index" json:"detected_at"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// QuarterlySummary stores aggregated MRI + HbA1c per calendar quarter.
type QuarterlySummary struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID   uuid.UUID `gorm:"index;not null" json:"patient_id"`
	Year        int       `gorm:"not null" json:"year"`
	Quarter     int       `gorm:"not null" json:"quarter"`
	MeanMRI     float64   `json:"mean_mri"`
	MinMRI      float64   `json:"min_mri"`
	MaxMRI      float64   `json:"max_mri"`
	MRICount    int       `json:"mri_count"`
	LatestHbA1c *float64  `json:"latest_hba1c,omitempty"`
	ComputedAt  time.Time `gorm:"not null" json:"computed_at"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
}
