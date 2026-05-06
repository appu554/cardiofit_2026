package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Person represents a human actor in the v2 substrate — a healthcare
// practitioner, ACOP-credentialed pharmacist, PCW, family member, or
// substitute decision-maker.
//
// Person is paired with one or more Role rows (1:N) capturing each capacity
// the person operates in. A single Person can be both an RN and an SDM
// for a different resident, for example.
//
// Canonical storage: kb-20-patient-profile (persons table, greenfield in
// migration 008_part1).
type Person struct {
	ID                uuid.UUID       `json:"id"`
	GivenName         string          `json:"given_name"`
	FamilyName        string          `json:"family_name"`
	HPII              string          `json:"hpii,omitempty"` // Healthcare Provider Identifier — Individual (16 digits)
	AHPRARegistration string          `json:"ahpra_registration,omitempty"`
	ContactDetails    json.RawMessage `json:"contact,omitempty"`
}
