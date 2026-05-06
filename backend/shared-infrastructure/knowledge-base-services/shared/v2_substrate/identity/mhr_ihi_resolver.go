package identity

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// MHRIHIResolver is the thin wrapper used by the MHR ingestion paths
// (Wave 3.1 SOAP/CDA + Wave 3.2 FHIR Gateway) to convert the IHI
// returned by ADHA into the canonical Vaidshala Resident UUID.
//
// Why a separate type rather than calling IdentityMatcher.Match
// directly: the MHR ingestion path always has an IHI in hand (the
// gateway is queried by IHI; the returned document carries the same
// IHI). The fuzzy paths (Medicare/name/DOB/facility) are not
// applicable here — a no-IHI-match should NOT fall through to fuzzy
// matching against demographic fields, because a wrong fuzzy match on
// a pathology result lands a lab on the wrong chart. Surface the
// no-match outcome explicitly so the runtime can route to the
// manual-review queue rather than auto-accept.
type MHRIHIResolver struct {
	lookup IdentityCandidateLookup
}

// NewMHRIHIResolver wires an IdentityCandidateLookup into the
// MHR-specific resolver. The same lookup that backs the general
// IdentityMatcher is reused; this is just a different policy on top
// (IHI-only, no fuzzy fallback).
func NewMHRIHIResolver(lookup IdentityCandidateLookup) *MHRIHIResolver {
	return &MHRIHIResolver{lookup: lookup}
}

// ErrMHRNoIHIMatch is returned by Resolve when the supplied IHI does
// not map to any Resident. Callers MUST route the originating document
// onto the manual-review queue rather than dropping it.
var ErrMHRNoIHIMatch = errors.New("mhr_ihi_resolver: no resident matches IHI")

// ErrMHREmptyIHI is returned by Resolve when the supplied IHI is empty.
// Distinct from no-match so callers can distinguish "MHR returned a
// document without an IHI" (a protocol-level concern) from "the IHI
// resolved to nothing" (a routing concern).
var ErrMHREmptyIHI = errors.New("mhr_ihi_resolver: empty IHI")

// Resolve converts an IHI to a canonical Resident UUID. Returns
// ErrMHREmptyIHI for empty input, ErrMHRNoIHIMatch when no mapping
// exists, and propagates any other lookup error.
//
// Returns a HIGH-confidence MatchResult on success (IHI is the
// deterministic-identifier path). Callers that need richer audit
// context (Confidence, Path, RequiresReview) consume the MatchResult
// directly; those that just need the UUID can read result.ResidentRef.
func (r *MHRIHIResolver) Resolve(ctx context.Context, ihi string) (MatchResult, error) {
	if ihi == "" {
		return MatchResult{Confidence: ConfidenceNone, Path: MatchPathNoMatch, RequiresReview: true}, ErrMHREmptyIHI
	}
	if r.lookup == nil {
		return MatchResult{}, ErrNoLookup
	}
	rid, err := r.lookup.LookupByIHI(ctx, ihi)
	if err != nil {
		if isNotFoundErr(err) {
			return MatchResult{Confidence: ConfidenceNone, Path: MatchPathNoMatch, RequiresReview: true}, ErrMHRNoIHIMatch
		}
		return MatchResult{}, err
	}
	if rid == nil {
		return MatchResult{Confidence: ConfidenceNone, Path: MatchPathNoMatch, RequiresReview: true}, ErrMHRNoIHIMatch
	}
	return MatchResult{
		ResidentRef:    rid,
		Confidence:     ConfidenceHigh,
		Path:           MatchPathIHI,
		RequiresReview: false,
	}, nil
}

// _ blank reference keeps uuid imported (signatures use *uuid.UUID via
// MatchResult.ResidentRef in the IdentityCandidateLookup contract).
var _ = uuid.Nil
