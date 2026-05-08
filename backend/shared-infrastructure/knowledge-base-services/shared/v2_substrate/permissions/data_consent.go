package permissions

import (
	"time"

	"github.com/google/uuid"
)

// DataAggregationConsent records a pharmacist's consent for a specific data element
// to be aggregated to a specific target for a specific bounded purpose.
//
// This is distinct from the clinical/treatment consent in Plan 0.2 (package consent/,
// table consents) which covers resident SDM consent. This covers data-aggregation
// consent for pharmacists per Self-Visibility Guidelines §8.1: purpose-bounded,
// time-bounded, per-element, revocable. Consent for one purpose does not extend to
// another.
//
// The Active and ActiveForPurpose methods enforce expiry at the application layer;
// the persistence layer (migration 029) enforces the bounded purpose values via a
// CHECK constraint on the purpose column.
type DataAggregationConsent struct {
	ID                uuid.UUID  `json:"id"`
	PharmacistID      uuid.UUID  `json:"pharmacist_id"`
	DataElement       string     `json:"data_element"`       // e.g. "rir_class_specific"
	AggregationTarget string     `json:"aggregation_target"` // e.g. "employer_pharmacy_xyz"
	Purpose           string     `json:"purpose"`            // bounded by Purpose constants
	GrantedAt         time.Time  `json:"granted_at"`
	ExpiresAt         time.Time  `json:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
	RevocationReason  *string    `json:"revocation_reason,omitempty"`
}

// Bounded purpose values per Self-Visibility Guidelines §8.1.
// These four values are enforced both here (IsValidPurpose) and at the
// database layer via the CHECK constraint on data_aggregation_consents.purpose.
const (
	PurposeWorkforcePlanning  = "workforce_planning"
	PurposeContractRetention  = "contract_retention"
	PurposeRegulatoryEvidence = "regulatory_evidence"
	PurposePeerDevelopment    = "peer_development"
)

// IsValidPurpose reports whether purpose is one of the four bounded values
// defined by Self-Visibility Guidelines §8.1.
// Callers (store layer, HTTP handler) should use this to validate incoming
// purpose strings before constructing a DataAggregationConsent.
func IsValidPurpose(purpose string) bool {
	return purpose == PurposeWorkforcePlanning ||
		purpose == PurposeContractRetention ||
		purpose == PurposeRegulatoryEvidence ||
		purpose == PurposePeerDevelopment
}

// Active returns true if the consent is currently in effect at asOf:
// not revoked and not expired.
func (c DataAggregationConsent) Active(asOf time.Time) bool {
	if c.RevokedAt != nil && !c.RevokedAt.After(asOf) {
		return false
	}
	if !c.ExpiresAt.After(asOf) {
		return false
	}
	return true
}

// ActiveForPurpose returns true if the consent is active and matches the requested
// purpose exactly. Consent for one purpose does not extend to another (§8.1).
func (c DataAggregationConsent) ActiveForPurpose(purpose string, asOf time.Time) bool {
	return c.Active(asOf) && c.Purpose == purpose
}
