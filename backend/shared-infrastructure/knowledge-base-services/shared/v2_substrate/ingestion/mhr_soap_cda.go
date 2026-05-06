// Package ingestion — MHR SOAP/CDA gateway client.
//
// Wave 3.1 (skeleton): defines the MHRSOAPClient interface that production
// wiring against the ADHA B2B SOAP gateway will satisfy. The default
// implementation returned by NewStubMHRSOAPClient returns a deferred-error
// from every method so a process that wires the stub to the runtime cannot
// silently succeed against a not-yet-configured gateway.
//
// Production wiring requires:
//   - NASH PKI client certificate + private key (held by Vaidshala ops)
//   - ADHA endpoint URL (per environment: test, staging, production)
//   - Subject identifier resolution (IHI-keyed; see identity.MHRIHIResolver)
//   - Throttling against ADHA's published rate limits
//   - XSD-validated CDA document handling per the ADHA conformance pack
//
// All five are deferred to V1 per docs/adr/2026-05-06-mhr-integration-
// strategy.md. The interface contract here is what V1 implementers
// satisfy; the synthetic CDA fixture in testdata/synthetic_cda_pathology.xml
// exercises the parser side independently of the network client.
package ingestion

import (
	"context"
	"errors"
	"time"
)

// MHRSOAPClient fetches clinical documents from the ADHA B2B SOAP gateway.
// Implementations MUST be safe for concurrent use by multiple goroutines.
//
// The two-method shape mirrors the ADHA B2B contract: discover document
// IDs for an IHI within a window (GetPathologyDocumentList), then fetch
// each document's raw bytes (FetchCDADocument). Splitting discovery from
// fetch lets the runtime apply per-document idempotency (via
// pathology_ingest_log) before paying the per-document SOAP round-trip.
type MHRSOAPClient interface {
	// GetPathologyDocumentList returns CDA document refs available for the
	// given IHI authored on or after `since`. Production implementations
	// MUST return refs sorted by AuthoredAt ASC so resumption from a
	// timestamp watermark is deterministic.
	GetPathologyDocumentList(ctx context.Context, ihi string, since time.Time) ([]MHRDocumentRef, error)

	// FetchCDADocument retrieves a CDA document by ID and returns the raw
	// XML bytes. The ADHA gateway returns CDA R2 documents wrapped in a
	// SOAP envelope; the production implementation MUST unwrap the
	// envelope and return only the CDA body bytes. The synthetic test
	// fixture is already in CDA-body form.
	FetchCDADocument(ctx context.Context, documentID string) ([]byte, error)
}

// MHRDocumentRef is the discovery-phase descriptor of a CDA document
// available in MHR for a given IHI. Production implementations populate
// every field; the stub returns a deferred-error before producing any.
type MHRDocumentRef struct {
	// DocumentID is the ADHA-issued opaque document identifier; passed
	// verbatim to FetchCDADocument.
	DocumentID string
	// DocumentType is the CDA template-id-derived classification, e.g.
	// "Pathology Report Document". Used for filtering before fetch so
	// non-pathology documents (discharge summaries, prescriptions) are
	// skipped without paying the fetch cost.
	DocumentType string
	// AuthoredAt is the document's authored-at timestamp from the CDA
	// header. Used for watermark resumption + ordering.
	AuthoredAt time.Time
	// AuthorRoleRef is the author's role identifier from the CDA header
	// (often a HPI-O for the issuing pathology lab). Stored on the
	// resulting EvidenceTrace node for provenance.
	AuthorRoleRef string
}

// CDAPathologyResult is the internal DTO produced by parsing a CDA
// pathology document. Both the SOAP/CDA path (Wave 3.1) and the FHIR
// Gateway path (Wave 3.2) converge on this DTO so downstream substrate
// writes are unified — one code path regardless of source.
type CDAPathologyResult struct {
	DocumentID   string
	PatientIHI   string
	AuthoredAt   time.Time
	Observations []ParsedObservation
}

// ParsedObservation is the unified internal representation of a single
// pathology result. Produced by all three Wave 3 ingestion paths
// (CDA, FHIR DiagnosticReport, HL7 ORU^R01) and consumed by the
// substrate write path that translates to models.Observation.
//
// Value vs ValueText: numeric labs populate Value (and Unit); textual
// results (e.g. microbiology narratives) populate ValueText with Value
// nil. The substrate Observation has the same dichotomy.
type ParsedObservation struct {
	// LOINCCode is the LOINC code identifying the observation (preferred
	// terminology). LOINC-AU codes are LOINC codes — same namespace.
	LOINCCode string
	// SNOMEDCode is the SNOMED-CT-AU code, when the source provides one
	// (CDA pathology results often carry both LOINC and SNOMED).
	SNOMEDCode string
	// DisplayName is the human-readable observation name from the source,
	// e.g. "Serum potassium". Carried for diagnostic display only; the
	// LOINC/SNOMED codes are the source of truth for downstream rules.
	DisplayName string
	// Value is the numeric value when the observation is quantitative.
	// Nil for qualitative results (use ValueText instead).
	Value *float64
	// ValueText is the textual value for qualitative results; empty
	// string when Value is populated.
	ValueText string
	// Unit is the UCUM unit string for quantitative results.
	Unit string
	// ObservedAt is the observation's effective time from the source.
	// For CDA, this is the effectiveTime of the observation activity;
	// for FHIR, Observation.effectiveDateTime; for HL7, OBR-7.
	ObservedAt time.Time
	// AbnormalFlag is sourced from CDA interpretationCode / FHIR
	// Observation.interpretation / HL7 OBX-8. Values: "high", "low",
	// or empty string when within reference range or unspecified.
	AbnormalFlag string
}

// ErrMHRWiringDeferred is returned by the stub MHR clients to signal the
// gateway is not yet wired for production. Callers MUST treat this
// distinctly from network/protocol errors.
var ErrMHRWiringDeferred = errors.New("mhr_soap_cda: production wiring deferred to V1")

// stubMHRSOAPClient is the Wave 3 default. Returns ErrMHRWiringDeferred
// from every method so misconfiguration fails loudly at the call site
// rather than silently producing empty result lists.
type stubMHRSOAPClient struct{}

// NewStubMHRSOAPClient returns the deferred-wiring stub for Wave 3.
// Replace with the production implementation when ADHA NASH PKI +
// endpoint configuration are available (V1 Phase 1).
func NewStubMHRSOAPClient() MHRSOAPClient { return &stubMHRSOAPClient{} }

func (s *stubMHRSOAPClient) GetPathologyDocumentList(_ context.Context, _ string, _ time.Time) ([]MHRDocumentRef, error) {
	return nil, ErrMHRWiringDeferred
}

func (s *stubMHRSOAPClient) FetchCDADocument(_ context.Context, _ string) ([]byte, error) {
	return nil, ErrMHRWiringDeferred
}
