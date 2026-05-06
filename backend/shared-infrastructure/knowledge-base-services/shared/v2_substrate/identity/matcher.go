// Package identity implements the v2 substrate's IdentityMatcher per
// Layer 2 doc §3.3: IHI-primary matching with confidence-tiered fuzzy
// fallback (Medicare+name+DOB → name+DOB+facility) and a manual-review
// queue for low-confidence and no-match cases.
//
// Identity matching errors are clinically dangerous: a misrouted
// pathology result lands on the wrong resident's chart, and a missed
// match fragments a resident's record across sources. The package's
// design choices reflect that:
//
//   - The matcher itself is pure and IO-free. It takes an
//     IdentityCandidateLookup interface (LookupByIHI, LookupByMedicare,
//     LookupByFacilityAndDOB), so unit tests use an in-memory fixture
//     and there is no transitive dependency on a database driver.
//   - EvidenceTrace logging is a service-layer concern (kb-20 storage),
//     not a matcher concern. The wrapper writes a node every time
//     Match returns — success, fuzzy hit, or queued — so audit
//     completeness is structural rather than discretionary.
//   - Confidence tiers are explicit. HIGH = IHI exact; MEDIUM =
//     Medicare+name+DOB fuzzy with name distance ≤ 2; LOW =
//     name+DOB+facility fuzzy with distance ≤ 3 (always
//     RequiresReview); NONE = no match (also RequiresReview).
//
// The pseudocode at Layer 2 doc §3.3 is implemented faithfully in
// Match (see standardMatcher.Match in fuzzy_matcher.go); deviations
// are documented inline.
package identity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Confidence ranks how strongly an IncomingIdentifier matches an
// existing Resident. High → IHI exact match (deterministic identifier
// reconciliation). Medium → Medicare+name+DOB fuzzy. Low → name+DOB
// scoped to a facility. None → no match found.
type Confidence string

// Confidence values per Layer 2 doc §3.3.
const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
	ConfidenceNone   Confidence = "none"
)

// IsValidConfidence reports whether s is a recognised Confidence value.
func IsValidConfidence(s string) bool {
	switch Confidence(s) {
	case ConfidenceHigh, ConfidenceMedium, ConfidenceLow, ConfidenceNone:
		return true
	}
	return false
}

// MatchPath identifies which algorithmic branch produced a MatchResult.
// Useful for audit (recorded on the EvidenceTrace node) and for
// diagnosing why a particular match landed in a particular tier.
type MatchPath string

// MatchPath values.
const (
	MatchPathIHI             MatchPath = "ihi"
	MatchPathMedicareNameDOB MatchPath = "medicare+name+dob"
	MatchPathNameDOBFacility MatchPath = "name+dob+facility"
	MatchPathNoMatch         MatchPath = "no_match"
)

// IsValidMatchPath reports whether s is a recognised MatchPath value.
func IsValidMatchPath(s string) bool {
	switch MatchPath(s) {
	case MatchPathIHI, MatchPathMedicareNameDOB, MatchPathNameDOBFacility, MatchPathNoMatch:
		return true
	}
	return false
}

// IncomingIdentifier is the inbound identity-bearing payload from a
// source (eNRMC CSV import, MHR pathology push, hospital discharge,
// etc.) before it has been bound to a canonical Resident. Empty
// fields signal "not provided"; the matcher tolerates partial data
// and fails cleanly to lower-confidence paths.
type IncomingIdentifier struct {
	IHI        string     `json:"ihi,omitempty"`        // 16-digit Individual Healthcare Identifier
	Medicare   string     `json:"medicare,omitempty"`   // Medicare card number
	DVA        string     `json:"dva,omitempty"`        // Department of Veterans' Affairs number
	GivenName  string     `json:"given_name,omitempty"`
	FamilyName string     `json:"family_name,omitempty"`
	DOB        time.Time  `json:"dob,omitempty"` // zero time treated as "not provided"
	FacilityID *uuid.UUID `json:"facility_id,omitempty"`
	Source     string     `json:"source,omitempty"` // e.g. "enrmc-csv-import", "mhr-pathology"
}

// MatchResult is the outcome of an IdentityMatcher.Match call. ResidentRef
// is nil when no match was found. RequiresReview is true for LOW-confidence
// and NONE cases; the service layer is responsible for enqueuing those
// onto the manual-review queue.
type MatchResult struct {
	// ResidentRef is the matched resident's canonical UUID, or nil when
	// no candidate satisfied any path.
	ResidentRef *uuid.UUID `json:"resident_ref,omitempty"`

	// Confidence is the strongest tier reached by the match.
	Confidence Confidence `json:"confidence"`

	// Path identifies the algorithmic branch that produced the result.
	Path MatchPath `json:"path"`

	// RequiresReview is true when the result must not be auto-accepted
	// without human verification (LOW or NONE confidence).
	RequiresReview bool `json:"requires_review"`

	// NameDistance is the Levenshtein distance on the combined-name string
	// when a fuzzy path produced the match; zero for IHI exact matches and
	// for no-match results. Recorded for audit.
	NameDistance int `json:"name_distance"`

	// Candidates lists every resident whose data passed the fuzzy
	// distance threshold on the producing path. For LOW-confidence matches
	// it is the full set of plausible residents — useful context for the
	// reviewer. For HIGH/MEDIUM it is typically empty (the chosen ref is
	// the only candidate at that tier).
	Candidates []uuid.UUID `json:"candidates,omitempty"`
}

// IdentityMatcher is the service contract: convert an IncomingIdentifier
// into a MatchResult. Implementations are expected to be deterministic
// for a fixed candidate-store snapshot. Concurrency: the standard
// implementation in this package is goroutine-safe assuming the
// underlying IdentityCandidateLookup is.
type IdentityMatcher interface {
	Match(ctx context.Context, incoming IncomingIdentifier) (MatchResult, error)
}

// ResidentCandidate is the minimum projection of a Resident that the
// fuzzy paths need: the canonical id plus the fields used for fuzzy
// scoring. Source-of-truth for these fields lives in
// shared/v2_substrate/models.Resident; we copy a narrow shape here so
// the identity package does not depend on the full Resident model
// (and so DB-backed lookups can avoid loading every column).
type ResidentCandidate struct {
	ID         uuid.UUID
	GivenName  string
	FamilyName string
	DOB        time.Time
}

// IdentityCandidateLookup is the storage abstraction the matcher
// depends on. Implementations live behind kb-20's storage layer
// (residents_v2 view + identity_mappings table). Keeping this
// interface in the identity package — rather than appending to the
// global ResidentStore — makes the matcher's IO surface explicit and
// keeps the core Resident interface lean.
type IdentityCandidateLookup interface {
	// LookupByIHI returns the canonical Resident UUID mapped to the given
	// IHI, or interfaces.ErrNotFound (the kb-20 sentinel) when nothing is
	// mapped. Implementations may also surface arbitrary errors.
	LookupByIHI(ctx context.Context, ihi string) (*uuid.UUID, error)

	// LookupByMedicare returns every Resident whose Medicare number maps
	// to it. The fuzzy path then narrows on DOB + name distance. An
	// empty result set is NOT an error — it signals "no Medicare match"
	// and the matcher falls through to the next path.
	LookupByMedicare(ctx context.Context, medicare string) ([]ResidentCandidate, error)

	// LookupByFacilityAndDOB returns every Resident at the given facility
	// whose DOB equals dob. The fuzzy path then scores name distance.
	LookupByFacilityAndDOB(ctx context.Context, facility uuid.UUID, dob time.Time) ([]ResidentCandidate, error)
}

// ErrNoLookup is returned by Match when the matcher was constructed
// without a candidate store. Distinct sentinel so service-layer code
// can distinguish configuration errors from genuine no-match results
// (which return MatchResult{Confidence: ConfidenceNone}, nil).
var ErrNoLookup = errors.New("identity: no IdentityCandidateLookup configured")
