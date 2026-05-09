// Package urgency derives a three-tier urgency classification from a
// ClinicalSnapshot's substrate signals.
//
// VisibilityClass: PDP — urgency derived from substrate signals
//
// Tag returns one of the three exported urgency constants (UrgencyRed,
// UrgencyAmber, UrgencyGreen) using a strict priority order: red signals
// are checked first and short-circuit amber evaluation. This mirrors the
// clinical intent — a recent fall or hospital admission demands immediate
// attention regardless of anticholinergic burden or care-intensity pathway.
package urgency

import kb32ctx "github.com/cardiofit/kb32/internal/context"

// Urgency level constants. Use these instead of string literals so that
// callers are insulated from spelling changes and benefit from IDE completion.
const (
	// UrgencyRed signals immediate clinical review required.
	// Triggered by: RecentFall72h or RecentAdmission72h.
	UrgencyRed = "red"

	// UrgencyAmber signals elevated clinical attention.
	// Triggered by: ACB >= 3, DBI >= 1.0, or CareIntensity in {palliative, end_of_life}.
	UrgencyAmber = "amber"

	// UrgencyGreen signals routine review interval.
	// Default when no red or amber signals are present.
	UrgencyGreen = "green"
)

// validUrgencies is the canonical set of urgency values produced by this package.
var validUrgencies = map[string]struct{}{
	UrgencyRed:   {},
	UrgencyAmber: {},
	UrgencyGreen: {},
}

// IsValidUrgency reports whether s is one of the three recognised urgency
// level strings. The check is case-sensitive.
func IsValidUrgency(s string) bool {
	_, ok := validUrgencies[s]
	return ok
}

// amberCareIntensities is the set of care-intensity values that trigger amber.
var amberCareIntensities = map[string]struct{}{
	"palliative":  {},
	"end_of_life": {},
}

// Tag derives the urgency tier for snap.
//
// Priority order (red short-circuits amber):
//  1. Red:   snap.RecentFall72h || snap.RecentAdmission72h
//  2. Amber: snap.ACB >= 3 || snap.DBI >= 1.0 || snap.CareIntensity in {palliative, end_of_life}
//  3. Green: default
//
// The returned string is always one of UrgencyRed, UrgencyAmber, UrgencyGreen.
func Tag(snap kb32ctx.ClinicalSnapshot) string {
	// Red signals: recent fall or recent hospital admission within 72 h.
	// These represent acute safety events that demand immediate pharmacist review.
	if snap.RecentFall72h || snap.RecentAdmission72h {
		return UrgencyRed
	}

	// Amber signals: high anticholinergic/sedative burden, or palliative pathway.
	_, careIsAmber := amberCareIntensities[snap.CareIntensity]
	if snap.ACB >= 3 || snap.DBI >= 1.0 || careIsAmber {
		return UrgencyAmber
	}

	return UrgencyGreen
}
