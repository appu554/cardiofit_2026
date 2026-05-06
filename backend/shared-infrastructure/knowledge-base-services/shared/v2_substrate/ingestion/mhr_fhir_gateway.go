// Package ingestion — MHR FHIR Gateway client.
//
// Wave 3.2 (skeleton): defines the MHRFHIRClient interface that production
// wiring against ADHA's FHIR Gateway (target GA July 2026) will satisfy.
// The default implementation returned by NewStubMHRFHIRClient returns a
// deferred-error from every method.
//
// The DiagnosticReport mapper IS a working implementation — V1 needs a
// stable contract to test FHIR Gateway responses against, and the mapper
// has no external dependencies (pure FHIR R4 JSON traversal).
package ingestion

import (
	"context"
	"errors"
	"time"
)

// MHRFHIRClient fetches FHIR R4 DiagnosticReport bundles from ADHA's
// FHIR Gateway. Production implementations satisfy:
//   - OAuth2 client-credentials flow against ADHA's token endpoint
//   - AU Core / AU Base profile compliance
//   - Pagination via Bundle.link rel=next
//   - Content negotiation (application/fhir+json)
//
// The interface returns a slice of DiagnosticReport resource maps
// (json-decoded). The mapper (ParseFHIRDiagnosticReport) consumes
// each map independently so the runtime can interleave parse + write
// without buffering the whole bundle.
type MHRFHIRClient interface {
	// GetDiagnosticReports returns FHIR R4 DiagnosticReport resources
	// for the given IHI authored on or after `since`. Each element is
	// the json-decoded resource as map[string]interface{}.
	GetDiagnosticReports(ctx context.Context, ihi string, since time.Time) ([]map[string]interface{}, error)
}

// ErrMHRFHIRWiringDeferred is the FHIR-Gateway analogue of
// ErrMHRWiringDeferred. Distinct sentinel so callers can tell which
// path is unwired without parsing error text.
var ErrMHRFHIRWiringDeferred = errors.New("mhr_fhir_gateway: production wiring deferred to V1")

type stubMHRFHIRClient struct{}

// NewStubMHRFHIRClient returns the deferred-wiring stub for Wave 3.
// Replace with the production implementation when ADHA FHIR Gateway
// general availability lands (target July 2026, V1 Phase 2).
func NewStubMHRFHIRClient() MHRFHIRClient { return &stubMHRFHIRClient{} }

func (s *stubMHRFHIRClient) GetDiagnosticReports(_ context.Context, _ string, _ time.Time) ([]map[string]interface{}, error) {
	return nil, ErrMHRFHIRWiringDeferred
}
