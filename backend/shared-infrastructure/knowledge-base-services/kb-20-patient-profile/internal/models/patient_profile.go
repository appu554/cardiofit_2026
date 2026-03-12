package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// PatientProfile stores demographics, disease history, and derived comorbidities.
type PatientProfile struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"size:100;uniqueIndex;not null" json:"patient_id"`

	// Demographics
	Age            int     `gorm:"not null" json:"age"`
	Sex            string  `gorm:"size:10;not null;check:sex IN ('M','F','OTHER')" json:"sex"`
	WeightKg       float64 `gorm:"type:decimal(5,2)" json:"weight_kg,omitempty"`
	HeightCm       float64 `gorm:"type:decimal(5,1)" json:"height_cm,omitempty"`
	BMI            float64 `gorm:"type:decimal(4,1)" json:"bmi,omitempty"`
	SmokingStatus  string  `gorm:"size:20;default:'unknown'" json:"smoking_status"`

	// Disease history
	DMType           string  `gorm:"size:20;check:dm_type IN ('T1DM','T2DM','GDM','NONE')" json:"dm_type"`
	DMDurationYears  float64 `gorm:"type:decimal(4,1)" json:"dm_duration_years"`

	// Derived state
	Comorbidities    pq.StringArray `gorm:"type:text[]" json:"comorbidities"`
	CVRiskCategory   string         `gorm:"size:30" json:"cv_risk_category,omitempty"`
	CKDStatus        string         `gorm:"size:20;default:'NONE';check:ckd_status IN ('NONE','SUSPECTED','CONFIRMED')" json:"ckd_status"`
	CKDStage         string         `gorm:"size:10" json:"ckd_stage,omitempty"`

	// HTN co-management
	HTNStatus string `gorm:"size:20;default:'NONE';check:htn_status IN ('NONE','SUSPECTED','CONFIRMED')" json:"htn_status"`
	Season    string `gorm:"size:10;default:'UNKNOWN'" json:"season,omitempty"` // SUMMER|MONSOON|WINTER|AUTUMN|UNKNOWN — derived from locale+date

	// FHIR integration
	FHIRPatientID string `gorm:"size:200;index;column:fhir_patient_id" json:"fhir_patient_id,omitempty"`

	// Metadata
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate computes BMI and derives season if height/weight are provided.
func (p *PatientProfile) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.computeBMI()
	p.deriveSeason()
	return nil
}

// BeforeUpdate recomputes BMI and re-derives season on update.
func (p *PatientProfile) BeforeUpdate(tx *gorm.DB) error {
	p.computeBMI()
	p.deriveSeason()
	return nil
}

// deriveSeason sets Season from the current time when it is empty or UNKNOWN.
func (p *PatientProfile) deriveSeason() {
	if p.Season == "" || p.Season == SeasonUnknown {
		p.Season = DeriveSeason(time.Now())
	}
}

func (p *PatientProfile) computeBMI() {
	if p.HeightCm > 0 && p.WeightKg > 0 {
		heightM := p.HeightCm / 100.0
		p.BMI = p.WeightKg / (heightM * heightM)
	}
}

// PatientProfileResponse is the full state response returned by GET /patient/:id/profile.
type PatientProfileResponse struct {
	Profile     PatientProfile  `json:"profile"`
	Labs        []LabEntry      `json:"labs"`
	Medications []MedicationState `json:"medications"`
	LatestEGFR  *float64        `json:"latest_egfr,omitempty"`
	CKDSubstage string          `json:"ckd_substage,omitempty"`
}
