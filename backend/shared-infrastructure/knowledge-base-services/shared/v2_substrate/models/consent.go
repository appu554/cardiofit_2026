package models

import (
	"time"

	"github.com/google/uuid"
)

// Consent is the v2/v3 regulatory substrate entity for restrictive-practice
// and psychotropic medication authorisation under the Aged Care Quality
// Standards 2026 and Restrictive Practice regulations 2019.
//
// One Consent can cover multiple recommendations through its Class scope.
// E.g. a "psychotropic" Consent covers all psychotropic recommendations
// for a resident until withdrawn or expired.
//
// State transitions are governed by consent.Lifecycle (Plan 0.2 Task 4),
// which writes an EvidenceTrace edge per transition. Direct State mutation
// outside the Lifecycle engine is a contract violation.
//
// Canonical storage: migrations/024_consent_lifecycle.sql (table: consents).
type Consent struct {
	ID            uuid.UUID  `json:"id"`
	ResidentID    uuid.UUID  `json:"resident_id"`
	Class         string     `json:"class"`           // see ConsentClass*
	State         string     `json:"state"`           // see ConsentState*
	GrantedByID   uuid.UUID  `json:"granted_by_id"`   // Person.id (SDM, resident-self, guardian)
	GrantedByRole string     `json:"granted_by_role"` // role at time of granting
	Conditions    string     `json:"conditions,omitempty"`  // for granted-with-conditions
	ScopeNotes    string     `json:"scope_notes,omitempty"`
	ValidFrom     time.Time  `json:"valid_from"`
	ValidUntil    *time.Time `json:"valid_until,omitempty"` // nullable = open-ended
	WithdrawnAt   *time.Time `json:"withdrawn_at,omitempty"`
	ExpiredAt     *time.Time `json:"expired_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// validConsentTransitions encodes the consent lifecycle DAG. A pair (from, to)
// is in the map iff the transition is permitted. Refused, Withdrawn, and
// Expired are terminal (not present as keys).
var validConsentTransitions = map[string]map[string]bool{
	ConsentStateRequested: {
		ConsentStateDiscussed: true,
		ConsentStateRefused:   true, // declined before discussion
	},
	ConsentStateDiscussed: {
		ConsentStateGranted:               true,
		ConsentStateGrantedWithConditions: true,
		ConsentStateRefused:               true,
	},
	ConsentStateGranted:               {ConsentStateActive: true},
	ConsentStateGrantedWithConditions: {ConsentStateActive: true},
	ConsentStateActive: {
		ConsentStateUnderReview: true,
		ConsentStateWithdrawn:   true,
		ConsentStateExpired:     true,
	},
	ConsentStateUnderReview: {
		ConsentStateActive:    true, // continued
		ConsentStateWithdrawn: true,
	},
	// Refused, Withdrawn, Expired are terminal.
}

// IsValidConsentTransition reports whether the lifecycle DAG permits from→to.
func IsValidConsentTransition(from, to string) bool {
	if !IsValidConsentState(from) || !IsValidConsentState(to) {
		return false
	}
	allowed, ok := validConsentTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}
