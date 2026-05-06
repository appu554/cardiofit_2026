package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Role represents a Person's authorisation capacity. A single Person can
// hold many Roles (e.g., an EN at one facility and an SDM for a different
// resident).
//
// The Qualifications JSONB shape MUST match the regulatory_scope_rules.role_qualifications
// schema authored in kb-22 migration 007 (Phase 1C-γ). The Authorisation
// evaluator (downstream phase) joins on these keys directly.
//
// Documented Qualifications shapes:
//
//	EN with notation:                  {"notation": true}
//	EN without notation + medication:  {"notation": false, "nmba_medication_qual": true}
//	Designated RN Prescriber:          {"endorsement": "designated_rn_prescriber",
//	                                    "valid_from": "2025-09-30",
//	                                    "prescribing_agreement_id": "..."}
//	ACOP-credentialed pharmacist:      {"apc_training_complete": true,
//	                                    "valid_from": "2026-07-01",
//	                                    "tier": 1}
//	Aboriginal/Torres Strait Islander Health Practitioner:
//	                                   {"atsihp": true}
//
// Canonical storage: kb-20-patient-profile (roles table, greenfield in
// migration 008_part1).
type Role struct {
	ID             uuid.UUID       `json:"id"`
	PersonID       uuid.UUID       `json:"person_id"`
	Kind           string          `json:"kind"` // see Role* constants in enums.go
	Qualifications json.RawMessage `json:"qualifications,omitempty"`
	FacilityID     *uuid.UUID      `json:"facility_id,omitempty"` // role scoped to a single facility, or nil = portable
	ValidFrom      time.Time       `json:"valid_from"`
	ValidTo        *time.Time      `json:"valid_to,omitempty"`
	EvidenceURL    string          `json:"evidence_url,omitempty"`
}
