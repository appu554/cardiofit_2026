package models

import (
	"time"

	"github.com/google/uuid"
)

// Resident represents an aged-care residential consumer ("person accessing
// funded aged care services in a residential aged care home" per Vic
// DPCS Act §36EA(1)(a) and equivalent Commonwealth definitions).
//
// Resident is the canonical patient-state subject for Vaidshala. It maps
// to AU FHIR Patient at the integration boundary (see fhir/patient_mapper.go)
// but the internal type is intentionally narrower than the FHIR profile.
//
// Canonical storage: kb-20-patient-profile (residents_v2 view over
// patient_profiles + extensions added in migration 008_part1).
type Resident struct {
	ID               uuid.UUID   `json:"id"`
	IHI              string      `json:"ihi,omitempty"` // Individual Healthcare Identifier (16 digits)
	GivenName        string      `json:"given_name"`
	FamilyName       string      `json:"family_name"`
	DOB              time.Time   `json:"dob"`
	Sex              string      `json:"sex"`                         // FHIR AdministrativeGender: male|female|other|unknown
	IndigenousStatus string      `json:"indigenous_status,omitempty"` // AU Core indigenous-status extension: aboriginal|tsi|both|neither|not_stated
	FacilityID       uuid.UUID   `json:"facility_id"`
	AdmissionDate    *time.Time  `json:"admission_date,omitempty"`
	CareIntensity    string      `json:"care_intensity"` // see CareIntensity* constants
	SDMs             []uuid.UUID `json:"sdms,omitempty"` // SubstituteDecisionMaker Person IDs
	Status           string      `json:"status"`         // see ResidentStatus* constants
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}
