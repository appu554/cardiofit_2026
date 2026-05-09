// Package kpis implements KPI computation for the pharmacist self-visibility service.
//
// VisibilityClass: PFA — aggregation gate enforced by Phase 1a middleware.
package kpis

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// RecRow is a recommendation record used for KPI computation.
// Class is the drug/therapeutic class of the recommendation; used by
// ComputeClassSpecificRate.
type RecRow struct {
	AuthorID    uuid.UUID
	State       string
	SubmittedAt time.Time
	Class       string
}

// implementedOrBeyond returns true for states that count toward the RIR numerator.
// "implemented or beyond" per Guidelines §4.1: implemented, outcome_recorded, closed.
func implementedOrBeyond(state string) bool {
	return state == "implemented" || state == "outcome_recorded" || state == "closed"
}

// ComputeRIR computes the Recommendation Implementation Rate for a single author.
//
//	RIR = count(implemented or beyond) / count(submitted age > windowDays)
//
// Returns math.NaN() when the denominator is zero.
func ComputeRIR(rows []RecRow, author uuid.UUID, asOf time.Time, windowDays int) float64 {
	cutoff := asOf.AddDate(0, 0, -windowDays)
	var num, den int
	for _, r := range rows {
		if r.AuthorID != author {
			continue
		}
		if r.SubmittedAt.After(cutoff) {
			continue // not aged enough to be eligible
		}
		den++
		if implementedOrBeyond(r.State) {
			num++
		}
	}
	if den == 0 {
		return math.NaN()
	}
	return float64(num) / float64(den)
}
