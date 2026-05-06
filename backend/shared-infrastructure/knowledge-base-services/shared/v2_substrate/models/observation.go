package models

import (
	"time"

	"github.com/google/uuid"
)

// Observation represents a v2 substrate clinical observation for a Resident.
// Distinguished from kb-20's legacy lab_entries by the kind discriminator
// (vital | lab | behavioural | mobility | weight) and the optional Delta
// computed at write time by the delta-on-write service.
//
// Value is *float64 (pointer-nullable) to distinguish "no numeric value, see
// ValueText" from "value=0.0". One of Value or ValueText MUST be present
// (enforced by the DB CHECK constraint observations_value_or_text + by the
// validator in shared/v2_substrate/validation/observation_validator.go).
//
// Canonical storage: kb-20-patient-profile (observations table, greenfield in
// migration 008_part2_partB; observations_v2 view UNIONs lab_entries with
// kind='lab' for backward compatibility).
//
// FHIR boundary: maps to AU FHIR Observation at integration boundaries via
// shared/v2_substrate/fhir/observation_mapper.go. Delta has no native FHIR
// representation and is encoded as a Vaidshala-namespaced FHIR extension.
type Observation struct {
	ID         uuid.UUID  `json:"id"`
	ResidentID uuid.UUID  `json:"resident_id"`
	LOINCCode  string     `json:"loinc_code,omitempty"`
	SNOMEDCode string     `json:"snomed_code,omitempty"`
	Kind       string     `json:"kind"` // see ObservationKind* constants in enums.go
	Value      *float64   `json:"value,omitempty"`
	ValueText  string     `json:"value_text,omitempty"`
	Unit       string     `json:"unit,omitempty"`
	ObservedAt time.Time  `json:"observed_at"`
	SourceID   *uuid.UUID `json:"source_id,omitempty"` // application-validated UUID reference to kb-22.clinical_sources; no DB FK (cross-DB)
	Delta      *Delta     `json:"delta,omitempty"`     // populated on write by delta-on-write service
	CreatedAt  time.Time  `json:"created_at"`
}

// Delta is the directional deviation of an Observation from the resident's
// baseline. Populated at write time by shared/v2_substrate/delta/compute.go.
//
// DirectionalFlag is one of the DeltaFlag* constants. When the baseline is
// unavailable (no historical data, behavioural kind, or nil Value),
// DirectionalFlag is DeltaFlagNoBaseline and BaselineValue + DeviationStdDev
// are zero.
type Delta struct {
	BaselineValue   float64   `json:"baseline_value"`
	DeviationStdDev float64   `json:"deviation_stddev"`
	DirectionalFlag string    `json:"flag"` // see DeltaFlag* constants in enums.go
	ComputedAt      time.Time `json:"computed_at"`
}
