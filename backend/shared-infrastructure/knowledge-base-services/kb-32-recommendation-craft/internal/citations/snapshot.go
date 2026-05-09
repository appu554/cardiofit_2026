// Package citations — snapshot.go implements fire-time citation pinning.
//
// VisibilityClass: AD — citation versioning per Guidelines §6 audit defensibility
//
// PinAtFireTime is the entry point called by the recommendation generator at
// the moment a recommendation is produced. It locks in the exact source
// version active at that instant, creating an immutable audit trail.
package citations

import (
	"context"
	"fmt"
	"time"
)

// PinAtFireTime creates and persists a RecommendationCitation for each source
// ID in anchors, pinning the version that was active at asOf.
//
// For each anchor SourceID the function:
//  1. Looks up the active SourceVersion at asOf via registry.ActiveVersion.
//  2. Constructs a RecommendationCitation with PinnedAt = asOf.
//  3. Persists the citation via registry.SaveCitation.
//
// If any anchor has no active version at asOf, PinAtFireTime returns
// ErrNoActiveVersion (wrapped) without persisting any citation for that anchor.
// Other anchors that were already processed in the loop are persisted.
//
// Empty anchors slice: returns an empty (non-nil) slice with no error.
//
// Audit guarantee: once citations are pinned, subsequent calls to Amend or
// Retract on the underlying source do NOT modify these records. The pinned
// Version field is the permanent fire-time pointer.
func PinAtFireTime(
	ctx context.Context,
	registry Registry,
	recID string,
	anchors []string,
	asOf time.Time,
) ([]RecommendationCitation, error) {
	result := make([]RecommendationCitation, 0, len(anchors))

	for _, sourceID := range anchors {
		sv, err := registry.ActiveVersion(ctx, sourceID, asOf)
		if err != nil {
			return result, fmt.Errorf("citations: pin_at_fire_time: source %q: %w", sourceID, err)
		}

		c := RecommendationCitation{
			RecommendationID: recID,
			SourceID:         sourceID,
			Version:          sv.Version,
			PinnedAt:         asOf,
		}

		if err := registry.SaveCitation(ctx, c); err != nil {
			return result, fmt.Errorf("citations: pin_at_fire_time: save citation for source %q: %w", sourceID, err)
		}

		result = append(result, c)
	}

	return result, nil
}
