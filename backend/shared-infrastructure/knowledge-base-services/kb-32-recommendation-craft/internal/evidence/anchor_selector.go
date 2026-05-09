// Package evidence implements Stage 3.5 of the six-stage rendering pipeline:
// evidence anchor selection for recommendation packets.
//
// VisibilityClass: PDP — clinical evidence anchoring
//
// Select picks up to MaxAnchorsPerRec anchors from a candidate slice using a
// stable three-key ordering: AU jurisdiction first, then ascending Rank, then
// ascending SourceID for deterministic tie-breaking. This ordering ensures
// that Australian Deprescribing Guidelines always precede international sources
// (Beers, STOPP/START) in the rendered recommendation.
package evidence

import "sort"

// MaxAnchorsPerRec is the maximum number of evidence anchors returned by Select.
// Limiting to 2 keeps recommendations concise and clinically focused.
const MaxAnchorsPerRec = 2

// Anchor represents a single evidence source that can be attached to a
// recommendation packet. Lower Rank values indicate stronger evidence
// (i.e. Rank 1 is stronger than Rank 2).
type Anchor struct {
	// SourceID is a stable, unique identifier for this evidence source
	// (e.g. "ADG-2025-AU", "BEERS-2023-US"). Used as the final tie-breaker.
	SourceID string

	// Title is the human-readable name of the evidence source.
	Title string

	// Jurisdiction is the regulatory or clinical jurisdiction this anchor
	// applies to. Valid values: "AU", "US", "EU", "INTL".
	// Use IsValidJurisdiction to validate.
	Jurisdiction string

	// Rank indicates evidence strength within a jurisdiction; lower is stronger.
	// Rank 1 is the strongest evidence available.
	Rank int
}

// validJurisdictions is the canonical set of accepted jurisdiction codes.
var validJurisdictions = map[string]struct{}{
	"AU":   {},
	"US":   {},
	"EU":   {},
	"INTL": {},
}

// IsValidJurisdiction reports whether s is one of the four recognised
// jurisdiction codes. The check is case-sensitive.
func IsValidJurisdiction(s string) bool {
	_, ok := validJurisdictions[s]
	return ok
}

// Select returns up to MaxAnchorsPerRec anchors from candidates, applying a
// stable three-key sort:
//  1. AU jurisdiction first (AU < everything else)
//  2. Ascending Rank within each jurisdiction tier (lower Rank = stronger evidence)
//  3. Ascending SourceID for deterministic tie-breaking within the same Rank
//
// The returned slice is always non-nil (it may be empty but is always sliceable).
// Candidates is not modified; a copy is sorted internally.
func Select(candidates []Anchor) []Anchor {
	if len(candidates) == 0 {
		return []Anchor{}
	}

	// Work on a copy to avoid mutating the caller's slice.
	sorted := make([]Anchor, len(candidates))
	copy(sorted, candidates)

	sort.SliceStable(sorted, func(i, j int) bool {
		ai, aj := sorted[i], sorted[j]

		// Key 1: AU jurisdiction sorts before all others.
		iAU := ai.Jurisdiction == "AU"
		jAU := aj.Jurisdiction == "AU"
		if iAU != jAU {
			return iAU // AU first
		}

		// Key 2: Lower Rank wins.
		if ai.Rank != aj.Rank {
			return ai.Rank < aj.Rank
		}

		// Key 3: Lexicographic SourceID for deterministic output.
		return ai.SourceID < aj.SourceID
	})

	if len(sorted) > MaxAnchorsPerRec {
		sorted = sorted[:MaxAnchorsPerRec]
	}
	return sorted
}
