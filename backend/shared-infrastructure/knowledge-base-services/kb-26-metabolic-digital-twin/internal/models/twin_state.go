package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// TwinState represents a versioned snapshot of a patient's metabolic digital twin.
// Each update creates a new row (append-only) for full audit trail.
type TwinState struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID    uuid.UUID `gorm:"type:uuid;not null;index:idx_twin_patient,priority:1" json:"patient_id"`
	StateVersion int       `gorm:"not null" json:"state_version"`
	UpdateSource string    `gorm:"size:50;not null" json:"update_source"`
	UpdatedAt    time.Time `gorm:"not null;default:now();index:idx_twin_patient,priority:2,sort:desc" json:"updated_at"`

	// Tier 1: Directly Measured
	FBG7dMean        *float64   `json:"fbg_7d_mean,omitempty"`
	FBG14dTrend      *string    `json:"fbg_14d_trend,omitempty"`
	PPBG7dMean       *float64   `json:"ppbg_7d_mean,omitempty"`
	HbA1c            *float64   `json:"hba1c,omitempty"`
	HbA1cDate        *time.Time `json:"hba1c_date,omitempty"`
	SBP14dMean       *float64   `json:"sbp_14d_mean,omitempty"`
	DBP14dMean       *float64   `json:"dbp_14d_mean,omitempty"`
	EGFR             *float64   `json:"egfr,omitempty"`
	EGFRDate         *time.Time `json:"egfr_date,omitempty"`
	WaistCm          *float64   `json:"waist_cm,omitempty"`
	WeightKg         *float64   `json:"weight_kg,omitempty"`
	BMI              *float64   `json:"bmi,omitempty"`
	DailySteps7dMean *float64   `json:"daily_steps_7d_mean,omitempty"`
	RestingHR        *float64   `json:"resting_hr,omitempty"`

	// Tier 2: Reliably Derived
	VisceralFatProxy    *float64 `json:"visceral_fat_proxy,omitempty"`
	VisceralFatTrend    *string  `json:"visceral_fat_trend,omitempty"`
	RenalSlope          *float64 `json:"renal_slope,omitempty"`
	RenalClassification *string  `json:"renal_classification,omitempty"`
	MAPValue            *float64 `json:"map_value,omitempty"`
	GlycemicVariability *float64 `json:"glycemic_variability,omitempty"`
	DawnPhenomenon      *bool    `json:"dawn_phenomenon,omitempty"`
	TrigHDLRatio        *float64 `json:"trig_hdl_ratio,omitempty"`

	// Tier 3: Estimated (JSONB)
	InsulinSensitivity   datatypes.JSON `gorm:"type:jsonb" json:"insulin_sensitivity,omitempty"`
	HepaticGlucoseOutput datatypes.JSON `gorm:"type:jsonb" json:"hepatic_glucose_output,omitempty"`
	MuscleMassProxy      datatypes.JSON `gorm:"type:jsonb" json:"muscle_mass_proxy,omitempty"`
	BetaCellFunction     datatypes.JSON `gorm:"type:jsonb" json:"beta_cell_function,omitempty"`
	SympatheticTone      datatypes.JSON `gorm:"type:jsonb" json:"sympathetic_tone,omitempty"`
}

func (TwinState) TableName() string { return "twin_states" }
