package models

import (
	"time"

	"github.com/google/uuid"
)

// MRI Categories (spec §4.4)
const (
	MRICategoryOptimal               = "OPTIMAL"               // 0-25
	MRICategoryMildDysregulation     = "MILD_DYSREGULATION"    // 26-50
	MRICategoryModerateDeterioration = "MODERATE_DETERIORATION" // 51-75
	MRICategoryHighDeterioration     = "HIGH_DETERIORATION"    // 76-100
)

// MRIScore is the persisted result of an MRI computation.
type MRIScore struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID uuid.UUID `gorm:"type:uuid;not null;index:idx_mri_patient,priority:1" json:"patient_id"`
	Score     float64   `gorm:"not null" json:"score"`
	Category  string    `gorm:"size:30;not null" json:"category"`
	Trend     string    `gorm:"size:20" json:"trend,omitempty"` // IMPROVING, STABLE, WORSENING
	TopDriver string    `gorm:"size:30" json:"top_driver,omitempty"`

	// Domain sub-scores (0-100 scaled)
	GlucoseDomain    float64 `json:"glucose_domain"`
	BodyCompDomain   float64 `json:"body_comp_domain"`
	CardioDomain     float64 `json:"cardio_domain"`
	BehavioralDomain float64 `json:"behavioral_domain"`

	// Per-signal z-scores (for decomposition endpoint)
	SignalZScores map[string]float64 `gorm:"serializer:json" json:"signal_z_scores,omitempty"`

	TwinStateID *uuid.UUID `gorm:"type:uuid" json:"twin_state_id,omitempty"`
	ComputedAt  time.Time  `gorm:"not null;default:now();index:idx_mri_patient,priority:2,sort:desc" json:"computed_at"`
}

func (MRIScore) TableName() string { return "mri_scores" }

// DomainScore is an intermediate computation result for one of the 4 MRI domains.
type DomainScore struct {
	Name    string             `json:"name"`
	Score   float64            `json:"score"`   // raw weighted z-score composite
	Scaled  float64            `json:"scaled"`  // 0-100 scaled
	Signals map[string]float64 `json:"signals"` // per-signal z-scores
}

// MRIResult is the full computation output returned by the scorer.
type MRIResult struct {
	Score     float64       `json:"score"`
	Category  string        `json:"category"`
	Trend     string        `json:"trend"`
	TopDriver string        `json:"top_driver"`
	Domains   []DomainScore `json:"domains"`
}
