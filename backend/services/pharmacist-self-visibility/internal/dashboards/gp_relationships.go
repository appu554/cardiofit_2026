// Package dashboards provides the pharmacist self-visibility dashboard surfaces.
//
// VisibilityClass: PDP — never aggregated to employer. This file implements
// Surface 3: My GP Relationships. GP framing patterns are derived from the
// pharmacist's own work only; they are never used to rank or score GPs, and
// the underlying acceptance-rate data is never surfaced to the UI layer.
package dashboards

import (
	"context"

	"github.com/google/uuid"
)

// gpPattern holds per-GP framing intelligence for a single pharmacist–GP
// relationship. It is an internal projection type: the acceptanceRate field
// exists so source implementations can compute it, but For() deliberately
// discards it and NEVER exposes it to callers.
//
// The type lives here (in the production file) so that GPSource can reference
// it in its interface signature and the Go compiler can resolve it from both
// production code and test files.
type gpPattern struct {
	// framingObservation is the human-readable framing hint, e.g.
	// "recommendations land better with monitoring plan up front".
	// MUST NOT contain acceptance percentages or comparative ranks.
	framingObservation string
	// acceptanceRate is the raw signal used only inside source implementations.
	// It is intentionally unexported at the GPCard level so no UI layer can
	// accidentally render it.
	acceptanceRate float64
	// optedOut indicates the GP has exercised their opt-out right. When true,
	// For() returns "default_framing" as the Display value regardless of any
	// observation text or rate.
	optedOut bool
}

// GPCard is the pharmacist's read-only view of their relationship with a
// single GP.
//
// VisibilityClass: PDP (Pharmacist-Default-Private) — employer MUST NOT
// receive this data without a separately recorded explicit consent event.
//
// Display carries a framing observation such as "recommendations to Dr X land
// better when a monitoring plan is attached" or the sentinel value
// "default_framing" when the GP has opted out. It MUST NEVER contain an
// acceptance percentage, rate figure, or comparative ranking.
type GPCard struct {
	// GPID is the unique identifier of the general practitioner.
	GPID uuid.UUID
	// Display is the framing observation text or "default_framing".
	// It intentionally omits any acceptance-rate figures.
	Display string
}

// GPSource is the data-access interface that backs GPRelationships.
// Implementations must:
//   - Respect context cancellation.
//   - Return the pharmacist's own relationship patterns only (never aggregate
//     across pharmacists).
//   - Honour GP opt-out flags before returning rows; the GPRelationships layer
//     applies a second-pass guard.
type GPSource interface {
	// PatternsForPharmacist returns a map of GP UUID → gpPattern for the given
	// pharmacist. The acceptanceRate in each gpPattern is available for source
	// logic but MUST NOT be surfaced via GPCard.Display.
	PatternsForPharmacist(ctx context.Context, pharmacistID uuid.UUID) (map[uuid.UUID]gpPattern, error)
}

// GPRelationships surfaces a pharmacist's per-GP framing pattern view.
// Construct with NewGPRelationships; call For to obtain the card list for a
// given pharmacist.
//
// VisibilityClass: PDP (Pharmacist-Default-Private)
//
// No-scorecard guarantee: For() never populates GPCard.Display with
// acceptance-rate figures or GP rankings. The raw acceptanceRate field in
// gpPattern is silently discarded during card construction.
type GPRelationships struct{ src GPSource }

// NewGPRelationships constructs a GPRelationships backed by the given
// GPSource.
func NewGPRelationships(src GPSource) *GPRelationships {
	return &GPRelationships{src: src}
}

// For returns the GPCard slice for the given pharmacist UUID.
//
// Opt-out guard: any GP whose gpPattern.optedOut flag is true will have
// Display set to "default_framing", regardless of any observation text or
// rate stored in the pattern.
//
// No-scorecard guard: acceptanceRate is never written to Display.
//
// A defensive context cancellation check is applied before source access. If
// the context is already cancelled, ctx.Err() is returned immediately.
//
// When the source returns no patterns, For returns a non-nil empty slice so
// callers can distinguish "no GP relationships" from an uninitialised result.
func (d *GPRelationships) For(ctx context.Context, pharmacistID uuid.UUID) ([]GPCard, error) {
	// Defensive context check: return early so callers receive an explicit
	// signal rather than a silently empty or partial result.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	patterns, err := d.src.PatternsForPharmacist(ctx, pharmacistID)
	if err != nil {
		return nil, err
	}

	cards := make([]GPCard, 0, len(patterns))
	for gpID, p := range patterns {
		c := GPCard{GPID: gpID}
		if p.optedOut {
			// GP has exercised their opt-out right; surface only the neutral
			// sentinel value regardless of observation text or rate.
			c.Display = "default_framing"
		} else {
			// Surface the framing observation only — never the acceptance rate.
			c.Display = p.framingObservation
		}
		cards = append(cards, c)
	}
	return cards, nil
}
