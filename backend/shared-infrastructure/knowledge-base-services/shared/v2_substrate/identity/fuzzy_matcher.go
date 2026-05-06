package identity

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// standardMatcher implements IdentityMatcher per the Layer 2 doc §3.3
// pseudocode: IHI exact match → Medicare+name+DOB fuzzy → name+DOB+
// facility fuzzy → no match. The struct holds only the candidate-
// lookup dependency; all decision state lives on the stack so the
// matcher is safe for concurrent use.
type standardMatcher struct {
	lookup IdentityCandidateLookup
}

// NewMatcher constructs an IdentityMatcher backed by the given lookup.
// Passing nil returns a matcher that errors on every Match call with
// ErrNoLookup; this is preferable to nil-panicking deep inside.
func NewMatcher(lookup IdentityCandidateLookup) IdentityMatcher {
	return &standardMatcher{lookup: lookup}
}

// Match implements the four-step pseudocode at Layer 2 doc §3.3.
//
// Step 1 (HIGH): if IHI is present and a mapping exists, return that
// resident with ConfidenceHigh. IHI is a 16-digit Australian
// identifier becoming universally available post-July-2026 under the
// Sharing by Default Act 2025; it is the gold standard.
//
// Step 2 (MEDIUM): if Medicare + a name + a non-zero DOB are all
// present, look up by Medicare and accept the first candidate whose
// DOB matches and whose combined-name Levenshtein distance is ≤ 2.
// Distance ≤ 2 is the threshold from the spec; common typos
// (single-character substitutions, transpositions) score ≤ 2.
//
// Step 3 (LOW): if facility + name + DOB are present, look up by
// (facility, dob) and pick the candidate with the smallest combined-
// name distance ≤ 3. Multiple candidates may pass; the chosen one is
// whichever has the smallest distance, and Candidates lists the full
// set so the reviewer sees the alternatives. LOW always sets
// RequiresReview = true.
//
// Step 4 (NONE): no path produced a match; return RequiresReview=true
// so the service layer queues the IncomingIdentifier for human
// reconciliation.
func (m *standardMatcher) Match(ctx context.Context, incoming IncomingIdentifier) (MatchResult, error) {
	if m.lookup == nil {
		return MatchResult{}, ErrNoLookup
	}

	// Step 1 — IHI exact match.
	if res, err := matchByIHI(ctx, m.lookup, incoming.IHI); err != nil {
		return MatchResult{}, err
	} else if res != nil {
		return *res, nil
	}

	// Step 2 — Medicare + name + DOB fuzzy.
	if res, err := matchByMedicareNameDOB(ctx, m.lookup, incoming); err != nil {
		return MatchResult{}, err
	} else if res != nil {
		return *res, nil
	}

	// Step 3 — name + DOB + facility low-confidence fuzzy.
	if res, err := matchByNameDOBFacility(ctx, m.lookup, incoming); err != nil {
		return MatchResult{}, err
	} else if res != nil {
		return *res, nil
	}

	// Step 4 — no match.
	return MatchResult{
		ResidentRef:    nil,
		Confidence:     ConfidenceNone,
		Path:           MatchPathNoMatch,
		RequiresReview: true,
	}, nil
}

// matchByMedicareNameDOB runs the MEDIUM-confidence path. Returns
// (nil, nil) when prerequisites aren't met or no candidate qualifies.
func matchByMedicareNameDOB(ctx context.Context, lookup IdentityCandidateLookup, incoming IncomingIdentifier) (*MatchResult, error) {
	if incoming.Medicare == "" {
		return nil, nil
	}
	if incoming.GivenName == "" && incoming.FamilyName == "" {
		return nil, nil
	}
	if incoming.DOB.IsZero() {
		return nil, nil
	}

	candidates, err := lookup.LookupByMedicare(ctx, incoming.Medicare)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}

	incomingName := combineName(incoming.GivenName, incoming.FamilyName)
	for i := range candidates {
		c := candidates[i]
		if !sameDay(c.DOB, incoming.DOB) {
			continue
		}
		d := levenshtein(combineName(c.GivenName, c.FamilyName), incomingName)
		if d <= 2 {
			id := c.ID
			return &MatchResult{
				ResidentRef:    &id,
				Confidence:     ConfidenceMedium,
				Path:           MatchPathMedicareNameDOB,
				RequiresReview: false,
				NameDistance:   d,
			}, nil
		}
	}
	return nil, nil
}

// matchByNameDOBFacility runs the LOW-confidence path. The matcher
// scans every (facility, dob) candidate, collects all whose name
// distance is ≤ 3, and returns the closest one as the chosen
// ResidentRef while listing every plausible candidate in Candidates.
// LOW always sets RequiresReview = true.
func matchByNameDOBFacility(ctx context.Context, lookup IdentityCandidateLookup, incoming IncomingIdentifier) (*MatchResult, error) {
	if incoming.FacilityID == nil {
		return nil, nil
	}
	if incoming.GivenName == "" && incoming.FamilyName == "" {
		return nil, nil
	}
	if incoming.DOB.IsZero() {
		return nil, nil
	}

	candidates, err := lookup.LookupByFacilityAndDOB(ctx, *incoming.FacilityID, incoming.DOB)
	if err != nil {
		if isNotFoundErr(err) {
			return nil, nil
		}
		return nil, err
	}

	incomingName := combineName(incoming.GivenName, incoming.FamilyName)

	var (
		bestIdx  = -1
		bestDist = math.MaxInt
		matching []uuid.UUID
	)
	for i := range candidates {
		c := candidates[i]
		d := levenshtein(combineName(c.GivenName, c.FamilyName), incomingName)
		if d > 3 {
			continue
		}
		matching = append(matching, c.ID)
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return nil, nil
	}
	chosen := candidates[bestIdx].ID
	return &MatchResult{
		ResidentRef:    &chosen,
		Confidence:     ConfidenceLow,
		Path:           MatchPathNameDOBFacility,
		RequiresReview: true,
		NameDistance:   bestDist,
		Candidates:     matching,
	}, nil
}

// combineName forms the canonical comparison string for fuzzy name
// matching: lowercase, trimmed, single space separator. We match on
// "given family" rather than each part individually because real
// inputs frequently swap parts (e.g. AU FHIR sources sometimes
// surface family-then-given), and a combined-string Levenshtein
// distance handles that correctly.
func combineName(given, family string) string {
	return strings.TrimSpace(strings.ToLower(strings.TrimSpace(given) + " " + strings.TrimSpace(family)))
}

// sameDay compares two times for calendar-day equality in their stored
// location. DOBs are normally written as midnight UTC dates with no
// time-of-day, but defensive code is cheap.
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// levenshtein computes the classic Levenshtein edit distance between
// two strings. Time and space complexity are O(m*n); for the realistic
// resident-name lengths (≤ ~60 chars combined) this is trivial. We
// roll our own because pulling in a third-party module for a 30-line
// algorithm is poor taste and the standard library has no built-in.
//
// Both inputs are compared as-is — call sites are expected to have
// already lowered/trimmed via combineName. The empty string yields a
// distance equal to the other string's length.
//
// Rune-correct: iterating over runes (not bytes) so multi-byte UTF-8
// names (Hà, Müller) score correctly per character rather than per
// byte. This matters for AU FHIR data which legitimately includes
// non-ASCII characters via the AU Core Patient profile.
func levenshtein(a, b string) int {
	ar := []rune(a)
	br := []rune(b)
	m := len(ar)
	n := len(br)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}
	// Two-row rolling buffer (O(min(m,n)) space) — keep prev / curr.
	prev := make([]int, n+1)
	curr := make([]int, n+1)
	for j := 0; j <= n; j++ {
		prev[j] = j
	}
	for i := 1; i <= m; i++ {
		curr[0] = i
		for j := 1; j <= n; j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			// deletion, insertion, substitution
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = minInt(del, minInt(ins, sub))
		}
		prev, curr = curr, prev
	}
	return prev[n]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
