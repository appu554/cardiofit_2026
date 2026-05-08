package consent

import (
	"context"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/recommendation"
)

// RecommendationTypeMapper maps a recommendation Type string to the
// Consent class that gates it. Returning ("", false) means no consent
// is required for this recommendation type — the checker passes
// unconditionally.
//
// Production callers wire a real mapping table (e.g., from kb-30
// configuration). Tests pass a small literal mapper.
type RecommendationTypeMapper func(recType string) (consentClass string, required bool)

// PostgresConsentChecker satisfies recommendation.ConsentChecker (Plan 0.1
// Task 5) by querying the consents table for an active matching Consent.
//
// The interface contract: ConsentActive(ctx, residentID, recType) →
// (bool, error). Returns true when either:
//   - The recType is not gated by any consent class (per the mapper), or
//   - An active consent of the required class exists for the resident.
//
// Returns false when consent is required but no active consent exists.
// Errors propagate Store-level errors verbatim.
type PostgresConsentChecker struct {
	store  Store
	mapper RecommendationTypeMapper
}

// NewPostgresConsentChecker constructs a checker wired to a Consent Store
// and a recommendation-type → consent-class mapper.
func NewPostgresConsentChecker(store Store, mapper RecommendationTypeMapper) *PostgresConsentChecker {
	return &PostgresConsentChecker{store: store, mapper: mapper}
}

// ConsentActive implements recommendation.ConsentChecker. The recType
// argument is the Recommendation's Type field; the mapper translates it
// to a Consent class for lookup.
func (c *PostgresConsentChecker) ConsentActive(ctx context.Context,
	residentID uuid.UUID, recType string) (bool, error) {
	class, required := c.mapper(recType)
	if !required {
		return true, nil
	}
	got, err := c.store.FindActive(ctx, residentID, class)
	if err != nil {
		return false, err
	}
	return got != nil, nil
}

// Compile-time check that PostgresConsentChecker satisfies the
// recommendation.ConsentChecker interface. If the recommendation
// package's interface signature changes, this line will fail to
// compile — catching drift at build time rather than at integration
// test time.
var _ recommendation.ConsentChecker = (*PostgresConsentChecker)(nil)
