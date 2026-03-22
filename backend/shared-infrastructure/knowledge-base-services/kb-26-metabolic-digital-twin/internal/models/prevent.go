package models

import (
	"time"

	"github.com/google/uuid"
)

// PREVENT risk categories (AHA PREVENT 2023)
const (
	PREVENTCategoryLow          = "LOW"          // <5%
	PREVENTCategoryBorderline   = "BORDERLINE"   // 5-7.5%
	PREVENTCategoryIntermediate = "INTERMEDIATE" // 7.5-20%
	PREVENTCategoryHigh         = "HIGH"         // >=20%
)

// PREVENTScore is the persisted result of a PREVENT 10-year CVD risk computation.
type PREVENTScore struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_prevent_patient,priority:1" json:"patient_id"`
	TenYearRisk float64    `gorm:"not null" json:"ten_year_risk"`
	RiskPercent float64    `gorm:"not null" json:"risk_percent"`
	Category    string     `gorm:"size:20;not null" json:"category"`
	InputAge    int        `json:"input_age"`
	InputSBP    float64    `json:"input_sbp"`
	InputTC     float64    `json:"input_tc"`
	InputHDL    float64    `json:"input_hdl"`
	InputEGFR   float64    `json:"input_egfr"`
	InputHbA1c  float64    `json:"input_hba1c"`
	TwinStateID *uuid.UUID `gorm:"type:uuid" json:"twin_state_id,omitempty"`
	ComputedAt  time.Time  `gorm:"not null;default:now();index:idx_prevent_patient,priority:2,sort:desc" json:"computed_at"`
}

func (PREVENTScore) TableName() string { return "prevent_scores" }
