package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// SimulationRun records the result of a coupled ODE simulation.
type SimulationRun struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID      uuid.UUID      `gorm:"type:uuid;not null;index:idx_sim_patient,priority:1" json:"patient_id"`
	RequestedAt    time.Time      `gorm:"not null;default:now();index:idx_sim_patient,priority:2,sort:desc" json:"requested_at"`
	Intervention   datatypes.JSON `gorm:"type:jsonb;not null" json:"intervention"`
	ProjectionDays int            `gorm:"not null" json:"projection_days"`
	Results        datatypes.JSON `gorm:"type:jsonb;not null" json:"results"`
	TwinStateID    *uuid.UUID     `gorm:"type:uuid" json:"twin_state_id,omitempty"`
	RequestedBy    *string        `gorm:"size:50" json:"requested_by,omitempty"`
}

func (SimulationRun) TableName() string { return "simulation_runs" }

// SimState holds the 6-dimensional latent state vector used by the coupled ODE.
type SimState struct {
	IS  float64 `json:"insulin_sensitivity"`
	VF  float64 `json:"visceral_fat"`
	HGO float64 `json:"hepatic_glucose_output"`
	MM  float64 `json:"muscle_mass"`
	VR  float64 `json:"vascular_resistance"`
	RR  float64 `json:"renal_reserve"`
}

// ProjectedState is one time-step of simulation output with observable biomarkers.
type ProjectedState struct {
	Day     int      `json:"day"`
	State   SimState `json:"state"`
	FBG     float64  `json:"fbg"`
	PPBG    float64  `json:"ppbg"`
	SBP     float64  `json:"sbp"`
	WaistCm float64  `json:"waist_cm"`
	EGFR    float64  `json:"egfr"`
	HbA1c   float64  `json:"hba1c"`
}

// Intervention describes a single intervention with its effect vector.
type Intervention struct {
	Type        string  `json:"type"`
	Code        string  `json:"code"`
	Description string  `json:"description"`
	ISEffect    float64 `json:"is_effect"`
	VFEffect    float64 `json:"vf_effect"`
	HGOEffect   float64 `json:"hgo_effect"`
	MMEffect    float64 `json:"mm_effect"`
	VREffect    float64 `json:"vr_effect"`
	RREffect    float64 `json:"rr_effect"`
}
