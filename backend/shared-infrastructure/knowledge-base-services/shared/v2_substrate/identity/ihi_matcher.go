package identity

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

// matchByIHI runs the high-confidence IHI exact-match path. It returns
// (resultPtr, true) when a mapping exists; (nil, false) otherwise so
// the caller can fall through to the next path.
//
// Errors from the lookup that are *not* "not found" are surfaced to
// the caller; a not-found result is silently treated as "no IHI
// mapping for this identifier" (i.e. fall through), matching the
// pseudocode at Layer 2 doc §3.3.
//
// We rely on the lookup wrapping its not-found sentinel with errors.Is
// against a shared ErrNotFound — kb-20's interfaces.ErrNotFound — but
// to keep the identity package free of a kb-20 dependency we accept any
// implementation that returns a nil *uuid.UUID on miss. The kb-20
// adapter in storage/identity_store.go observes this contract.
func matchByIHI(ctx context.Context, lookup IdentityCandidateLookup, ihi string) (*MatchResult, error) {
	if ihi == "" {
		return nil, nil
	}
	rid, err := lookup.LookupByIHI(ctx, ihi)
	if err != nil {
		// We deliberately swallow not-found errors and fall through.
		// Anything else (DB outage, malformed row) propagates so the
		// caller surfaces a 5xx rather than a misleading no-match.
		if isNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}
	if rid == nil {
		return nil, nil
	}
	return &MatchResult{
		ResidentRef:    rid,
		Confidence:     ConfidenceHigh,
		Path:           MatchPathIHI,
		RequiresReview: false,
	}, nil
}

// isNotFoundErr reports whether err looks like a "no row found" sentinel.
// We unwrap chains and compare error strings to stay decoupled from the
// kb-20 storage package; kb-20's ErrNotFound is "v2_substrate: entity
// not found" which we match by suffix below for resilience to wrapping.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	// errors.Is against a sentinel string-typed value would be cleaner
	// if it weren't for the package-dependency hop. The contract is
	// documented on IdentityCandidateLookup.LookupByIHI.
	for e := err; e != nil; e = errors.Unwrap(e) {
		if msg := e.Error(); msg != "" {
			// Match the kb-20 sentinel verbatim and a few common driver
			// equivalents (sql.ErrNoRows, gorm's ErrRecordNotFound).
			switch msg {
			case "v2_substrate: entity not found",
				"sql: no rows in result set",
				"record not found":
				return true
			}
		}
	}
	return false
}

// _ blank assignment guards against the package declaring uuid only for
// a doc-comment reference; the lookup interface uses uuid.UUID at the
// signature level so the import is real, but having this tiny surface
// makes godoc generation and refactors safer.
var _ = uuid.Nil
