package kpis

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// ComputeClassSpecificRate computes the recommendation implementation rate for
// a single author filtered to a specific drug/therapeutic class.
//
//	Rate = count(implemented or beyond in class) / count(eligible in class)
//
// "Eligible" means: authored by author, belonging to class, and submitted more
// than windowDays ago relative to asOf.
//
// Returns math.NaN() when the denominator is zero (no eligible records in class).
func ComputeClassSpecificRate(rows []RecRow, author uuid.UUID, class string, asOf time.Time, windowDays int) float64 {
	cutoff := asOf.AddDate(0, 0, -windowDays)
	var num, den int
	for _, r := range rows {
		if r.AuthorID != author || r.Class != class || r.SubmittedAt.After(cutoff) {
			continue
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
