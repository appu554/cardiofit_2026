package models

import (
	"time"

	"github.com/google/uuid"
)

// CalibratedEffect stores a patient-specific Bayesian-calibrated treatment effect.
// The population effect comes from KB-25 edge weights; the patient effect is
// updated as observations accumulate via conjugate Normal-Normal updates.
type CalibratedEffect struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID        uuid.UUID `gorm:"type:uuid;not null;index:idx_calibrated_patient" json:"patient_id"`
	KB25EdgeType     string    `gorm:"size:50;not null" json:"kb25_edge_type"`
	InterventionCode string    `gorm:"size:50;not null" json:"intervention_code"`
	TargetVariable   string    `gorm:"size:50;not null" json:"target_variable"`
	PopulationEffect float64   `gorm:"not null" json:"population_effect"`
	PatientEffect    float64   `gorm:"not null" json:"patient_effect"`
	Observations     int       `gorm:"not null;default:0" json:"observations"`
	Confidence       float64   `gorm:"not null;default:0" json:"confidence"`
	PriorMean        *float64  `json:"prior_mean,omitempty"`
	PriorSD          *float64  `json:"prior_sd,omitempty"`
	PosteriorMean    *float64  `json:"posterior_mean,omitempty"`
	PosteriorSD      *float64  `json:"posterior_sd,omitempty"`
	UpdatedAt        time.Time `gorm:"not null;default:now()" json:"updated_at"`
}

func (CalibratedEffect) TableName() string { return "calibrated_effects" }
