package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// MedicationState stores a patient's current medication with FDC decomposition support.
// Finding F-01 (RED): fdc_components enables fixed-dose combination decomposition
// for Indian-market prescribing patterns (60–70% FDCs).
type MedicationState struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID string    `gorm:"size:100;not null;index" json:"patient_id"`

	// Drug identification
	DrugName  string          `gorm:"size:200;not null" json:"drug_name"`
	DrugClass string          `gorm:"size:50;not null;index" json:"drug_class"`
	DoseMg    decimal.Decimal `gorm:"type:decimal(10,2)" json:"dose_mg"`
	Frequency string          `gorm:"size:50" json:"frequency"`
	Route     string          `gorm:"size:30;default:'ORAL'" json:"route"`

	// Chronotherapy (Wave 3.2, Amendment 10)
	DoseTiming DoseTiming `gorm:"size:20;default:'UNKNOWN'" json:"dose_timing"`

	// Prescriber
	PrescribedBy string `gorm:"size:100" json:"prescribed_by,omitempty"`

	// FDC decomposition (F-01 RED)
	// When non-empty, CM activation evaluates ALL component drug classes.
	FDCComponents pq.StringArray `gorm:"type:text[];column:fdc_components" json:"fdc_components,omitempty"`
	FDCParentID   *uuid.UUID     `gorm:"type:uuid" json:"fdc_parent_id,omitempty"`

	// FHIR integration
	FHIRMedicationRequestID string `gorm:"size:200;index;column:fhir_medication_request_id" json:"fhir_medication_request_id,omitempty"`
	ATCCode                 string `gorm:"size:20;column:atc_code" json:"atc_code,omitempty"`

	// Status
	IsActive  bool      `gorm:"default:true;index" json:"is_active"`
	StartDate time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DoseTiming represents the time-of-day dosing schedule for a medication.
// Used by KB-23 chronotherapy logic to detect timing-before-escalation opportunities.
type DoseTiming string

const (
	DoseTimingMorning    DoseTiming = "MORNING"
	DoseTimingEvening    DoseTiming = "EVENING"
	DoseTimingBedtime    DoseTiming = "BEDTIME"
	DoseTimingTwiceDaily DoseTiming = "TWICE_DAILY"
	DoseTimingWithMeals  DoseTiming = "WITH_MEALS"
	DoseTimingUnknown    DoseTiming = "UNKNOWN"
)

// Drug class enum constants — includes India-specific agents (F-01).
const (
	DrugClassMetformin     = "METFORMIN"
	DrugClassSGLT2I        = "SGLT2I"
	DrugClassDPP4I         = "DPP4I"
	DrugClassSulfonylurea  = "SULFONYLUREA"
	DrugClassCCB           = "CCB"
	DrugClassARB           = "ARB"
	DrugClassACEInhibitor  = "ACE_INHIBITOR"
	DrugClassInsulin       = "INSULIN"
	DrugClassStatin        = "STATIN"
	DrugClassBetaBlocker   = "BETA_BLOCKER"
	DrugClassDiuretic      = "DIURETIC"
	DrugClassGLP1RA        = "GLP1RA"
	DrugClassThiazolidinedione = "THIAZOLIDINEDIONE"

	// India-specific agents (F-01)
	DrugClassTeneligliptin = "TENELIGLIPTIN"
	DrugClassSaroglitazar  = "SAROGLITAZAR"
	DrugClassRemogliflozin = "REMOGLIFLOZIN"
	DrugClassDualPPAR      = "DUAL_PPAR"
	DrugClassVoglibose     = "VOGLIBOSE"
)

// EffectiveDrugClasses returns all drug classes this medication contributes,
// decomposing FDCs into their component classes.
func (m *MedicationState) EffectiveDrugClasses() []string {
	if len(m.FDCComponents) > 0 {
		return []string(m.FDCComponents)
	}
	return []string{m.DrugClass}
}

// AddMedicationRequest is the JSON body for POST /patient/:id/medications.
type AddMedicationRequest struct {
	DrugName      string   `json:"drug_name" binding:"required"`
	DrugClass     string   `json:"drug_class" binding:"required"`
	DoseMg        float64  `json:"dose_mg"`
	Frequency     string   `json:"frequency"`
	Route         string   `json:"route"`
	PrescribedBy  string   `json:"prescribed_by,omitempty"`
	FDCComponents []string `json:"fdc_components,omitempty"`
	StartDate     string   `json:"start_date,omitempty"`
}

// UpdateMedicationRequest is the JSON body for PUT /patient/:id/medications/:med_id.
type UpdateMedicationRequest struct {
	DoseMg    *float64 `json:"dose_mg,omitempty"`
	Frequency *string  `json:"frequency,omitempty"`
	IsActive  *bool    `json:"is_active,omitempty"`
}
