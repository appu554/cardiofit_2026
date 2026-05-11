package aggregation

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FamilyCommunicationContext is the Layer 1 family-engagement panel
// per v1.0 Part 9.3. Layer 1 surfaces only the most-recent family
// meeting date — the full chronology (attendees, structured priorities,
// scheduled meetings) is a Layer 2 escalation per Addendum Part 3.2.
//
// TODO(Layer 2 expansion: full family meeting chronology per Addendum
// Part 3.2)
type FamilyCommunicationContext struct {
	LastMeetingDate *time.Time
	SubstrateRefs   []SubstrateRef
}

// BuildFamilyCommunicationContext returns the Layer 1 family panel.
// When no meeting is on record, LastMeetingDate is nil and the
// returned context's SubstrateRefs is a non-nil empty slice (so
// downstream verification-not-belief checks treat the empty state as
// "fetched, zero" rather than "no data").
func BuildFamilyCommunicationContext(
	ctx context.Context,
	client SubstrateClient,
	residentID uuid.UUID,
) (FamilyCommunicationContext, error) {
	if client == nil {
		return FamilyCommunicationContext{}, fmt.Errorf("BuildFamilyCommunicationContext: nil SubstrateClient")
	}
	last, err := client.LastFamilyMeetingDate(ctx, residentID)
	if err != nil {
		return FamilyCommunicationContext{}, fmt.Errorf("LastFamilyMeetingDate: %w", err)
	}
	fc := FamilyCommunicationContext{
		LastMeetingDate: last,
		SubstrateRefs:   []SubstrateRef{},
	}
	if last != nil {
		fc.SubstrateRefs = append(fc.SubstrateRefs, SubstrateRef{
			Source:      "kb-family-comms",
			ID:          residentID,
			Description: fmt.Sprintf("last family meeting %s", last.Format("2006-01-02")),
		})
	}
	return fc, nil
}
