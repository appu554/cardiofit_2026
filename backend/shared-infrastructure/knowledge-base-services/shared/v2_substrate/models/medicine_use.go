package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// MedicineUse represents a v2 substrate medication record for a Resident.
// Distinguished from kb-20's legacy medication_states by three v2-specific
// JSONB fields: Intent (why), Target (what success looks like), and
// StopCriteria (when to stop). These fields are the basis for the Recommendation
// state machine's deprescribing logic, which arrives in a later phase.
//
// Canonical storage: kb-20-patient-profile (medicine_uses_v2 view over
// medication_states + the v2 columns added in migration 008_part2 part A).
//
// FHIR boundary: maps to AU FHIR MedicationRequest at integration boundaries
// via shared/v2_substrate/fhir/medication_request_mapper.go. Intent / Target /
// StopCriteria do not have native FHIR representations and are encoded as
// Vaidshala-namespaced FHIR extensions.
type MedicineUse struct {
	ID           uuid.UUID    `json:"id"`
	ResidentID   uuid.UUID    `json:"resident_id"`
	AMTCode      string       `json:"amt_code,omitempty"`      // Australian Medicines Terminology code
	DisplayName  string       `json:"display_name"`            // human-readable; falls back to legacy drug_name
	Intent       Intent       `json:"intent"`                  // v2-distinguishing
	Target       Target       `json:"target"`                  // v2-distinguishing (JSONB)
	StopCriteria StopCriteria `json:"stop_criteria"`           // v2-distinguishing (JSONB)
	Dose         string       `json:"dose,omitempty"`          // unstructured form
	Route        string       `json:"route,omitempty"`         // ORAL, IV, IM, etc.
	Frequency    string       `json:"frequency,omitempty"`     // e.g., "BID", "QD"
	PrescriberID *uuid.UUID   `json:"prescriber_id,omitempty"` // v2 Person.id; nullable for legacy records
	StartedAt    time.Time    `json:"started_at"`
	EndedAt      *time.Time   `json:"ended_at,omitempty"`
	Status       string       `json:"status"` // see MedicineUseStatus* constants
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Intent describes WHY a medicine is used.
type Intent struct {
	Category   string `json:"category"`   // see Intent* constants in enums.go
	Indication string `json:"indication"` // free text or SNOMED-CT-AU code
	Notes      string `json:"notes,omitempty"`
}

// Target describes WHAT successful therapy looks like for this medicine.
//
// Spec is JSON.RawMessage stored opaquely at the model layer; per-Kind
// shape contracts live in target_schemas.go. Validators in
// validation/target_validator.go delegate to per-Kind validators based
// on the Kind discriminator.
type Target struct {
	Kind string          `json:"kind"` // see TargetKind* constants in enums.go
	Spec json.RawMessage `json:"spec"`
}

// StopCriteria describes WHEN the medicine should stop.
//
// Triggers is a list of structured reasons (see StopTrigger* constants);
// ReviewDate is the next required clinical review; Spec is an optional
// JSONB shape (see stop_criteria_schemas.go) for additional structured
// criteria like threshold-based stops.
type StopCriteria struct {
	Triggers   []string        `json:"triggers"` // see StopTrigger* constants in enums.go
	ReviewDate *time.Time      `json:"review_date,omitempty"`
	Spec       json.RawMessage `json:"spec,omitempty"`
}
