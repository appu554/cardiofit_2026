package recommendation

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EscalationEvent is emitted when a deferred recommendation passes its
// review_due_at without action. The Worklist surface (Layer 4) renders these
// as "needs attention" cards. This is the architectural mechanism v2 §3
// line 134 calls out as the cure for the Ramsey 50% non-implementation
// problem: deferred items must be forced back to attention.
type EscalationEvent struct {
	RecommendationID uuid.UUID
	ResidentID       uuid.UUID
	AuthorID         uuid.UUID
	OriginalDueAt    time.Time
	EmittedAt        time.Time
}

// EscalationSink is the Worklist event-bus boundary. In production this
// wraps a Kafka producer; in tests we use a recording double.
type EscalationSink interface {
	Emit(ctx context.Context, ev EscalationEvent) error
}

// DeferredEscalator periodically sweeps deferred recommendations whose
// review_due_at has passed and emits an EscalationEvent for each.
//
// Operational note: this worker is idempotent at the event-bus boundary —
// the Worklist consumer dedupes on (RecommendationID, day). The escalator
// itself does NOT mutate recommendation state; the human action of
// re-surfacing or closing is the source of truth.
type DeferredEscalator struct {
	store Store
	sink  EscalationSink
	now   func() time.Time
}

func NewDeferredEscalator(store Store, sink EscalationSink, now func() time.Time) *DeferredEscalator {
	return &DeferredEscalator{store: store, sink: sink, now: now}
}

// RunOnce performs a single sweep. Production deployment wires this on a
// 5-minute ticker.
func (d *DeferredEscalator) RunOnce(ctx context.Context) error {
	overdue, err := d.store.ListDeferredOverdue(ctx, d.now())
	if err != nil {
		return err
	}
	for _, r := range overdue {
		ev := EscalationEvent{
			RecommendationID: r.ID,
			ResidentID:       r.ResidentID,
			AuthorID:         r.AuthorID,
			EmittedAt:        d.now(),
		}
		if r.ReviewDueAt != nil {
			ev.OriginalDueAt = *r.ReviewDueAt
		}
		if err := d.sink.Emit(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}
