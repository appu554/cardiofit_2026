package drill_through

import (
	"fmt"
	"time"
)

// NegativeEvidenceSearch is the input record for negative-evidence
// rendering per v1.0 Part 10.4. The pharmacist (or upstream substrate)
// records what was searched, what was found, and what could not be
// searched in the current implementation. The rendering function
// translates this into the epistemic-humility framing v1.0 Part 10.4
// requires.
type NegativeEvidenceSearch struct {
	Claim             string    // the negative claim being made (e.g., "no current indication for omeprazole")
	SearchedSources   []string  // sources actually searched (e.g., "eNRMC indication field", "progress notes (24mo)", "care plan")
	UnsearchedSources []string  // sources that exist but were NOT searchable (e.g., "scanned discharge summaries")
	SearchedAt        time.Time // when the search was executed
	Confidence        string    // "high" | "moderate" | "low"
}

// NegativeEvidenceRendering is the v1.0 Part 10.4 epistemic-humility
// presentation envelope. Statement is the carefully-framed assertion;
// EvidenceLines are the substrate-reviewed bullet points (each prefixed
// with "✓" by the renderer); Caveat is the explicit acknowledgment of
// what was not searched.
//
// The discipline is structural: "we searched for X; found nothing" vs
// "X is absent". The framing distinction is not a clinical judgment —
// it is a substrate-grounding pattern.
type NegativeEvidenceRendering struct {
	Statement     string   // the carefully-framed assertion (humility-preserving)
	EvidenceLines []string // bullet points of what was searched + result
	Caveat        string   // explicit acknowledgment of unsearched sources
	Confidence    string
}

// RenderNegativeEvidence transforms a search record into the
// rendering envelope. The function is pure; no I/O.
//
// Framing rules per v1.0 Part 10.4 lines 847–858:
//   - Statement uses "No X identified in available records" (not "X is absent")
//   - Each searched source becomes a "✓ <source>: <result>" evidence line
//   - Unsearched sources are surfaced as a caveat ("Note: ... may not be searchable in current implementation")
func RenderNegativeEvidence(search NegativeEvidenceSearch) NegativeEvidenceRendering {
	stmt := search.Claim
	if stmt == "" {
		stmt = "No matching evidence identified in available records"
	} else {
		// Lightly normalise: ensure the assertion uses the available-records
		// framing rather than absolute-absence framing. This is mechanical,
		// not clinical — it does not change the underlying claim.
		stmt = fmt.Sprintf("%s (in available records)", stmt)
	}

	lines := make([]string, 0, len(search.SearchedSources))
	for _, s := range search.SearchedSources {
		lines = append(lines, fmt.Sprintf("✓ %s: searched, no matching evidence found", s))
	}

	caveat := ""
	if len(search.UnsearchedSources) > 0 {
		caveat = "Note: negative-evidence claim. Substrate searched what was available; the following sources are not searchable in current implementation: "
		for i, s := range search.UnsearchedSources {
			if i > 0 {
				caveat += ", "
			}
			caveat += s
		}
		caveat += "."
	}

	conf := search.Confidence
	if conf == "" {
		conf = "moderate" // v1.0 Substrate Query Feasibility Analysis rates negative-evidence search as medium-higher risk
	}

	return NegativeEvidenceRendering{
		Statement:     stmt,
		EvidenceLines: lines,
		Caveat:        caveat,
		Confidence:    conf,
	}
}
