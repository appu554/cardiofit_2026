package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// Escalator periodically sweeps active monitoring plans whose
// expected_end_at has passed (or whose obligations are stacking up
// missed) and emits an EventTypeMonitoringObligationMissed Event +
// transitions the plan to escalated when the missed count reaches the
// plan's EscalateAfterMissed threshold.
//
// Operational note: production deployment wires this on a 1-hour ticker.
// Per-row continue-on-failure semantics (idempotent at consent level —
// next tick retries). The Postgres partial index
// idx_monitoring_active_sweep makes the per-sweep query cheap.
type Escalator struct {
	store       Store
	lc          *Lifecycle
	events      EventEmitter
	now         func() time.Time
	graceWindow time.Duration
}

// NewEscalator constructs an escalator. graceWindow is the leeway granted
// after an obligation's DueAt before counting it as missed. now is
// injectable for deterministic tests; production callers can leave it
// unset (defaults to time.Now().UTC()).
func NewEscalator(store Store, lc *Lifecycle, events EventEmitter,
	graceWindow time.Duration) *Escalator {
	return &Escalator{
		store: store, lc: lc, events: events,
		now:         func() time.Time { return time.Now().UTC() },
		graceWindow: graceWindow,
	}
}

// RunOnce performs a single sweep. For each active plan whose
// expected_end_at has passed:
//  1. Count obligations whose DueAt < now - graceWindow AND FulfilledAt
//     is nil — these are the missed obligations.
//  2. If missed >= EscalateAfterMissed, emit an Event and transition
//     plan to escalated.
//
// Per-row continue-on-failure: a single plan's escalation failure does
// NOT abort the whole sweep; the next tick retries.
func (e *Escalator) RunOnce(ctx context.Context) error {
	cutoff := e.now()
	plans, err := e.store.ListActiveOverdue(ctx, cutoff)
	if err != nil {
		return fmt.Errorf("list overdue: %w", err)
	}
	missCutoff := cutoff.Add(-e.graceWindow)
	for _, plan := range plans {
		missed := 0
		for _, ob := range plan.Obligations {
			if ob.FulfilledAt != nil {
				continue
			}
			if ob.DueAt.Before(missCutoff) {
				missed++
			}
		}
		if missed < plan.EscalateAfterMissed {
			continue
		}
		// Threshold reached — emit event + escalate.
		payload, _ := json.Marshal(map[string]any{
			"plan_id":      plan.ID,
			"missed_count": missed,
			"threshold":    plan.EscalateAfterMissed,
		})
		ev := models.Event{
			ID:                    uuid.New(),
			EventType:             models.EventTypeMonitoringObligationMissed,
			ResidentID:            plan.ResidentID,
			OccurredAt:            cutoff,
			ReportedByRef:         uuid.Nil, // system-generated
			DescriptionStructured: payload,
			DescriptionFreeText: fmt.Sprintf(
				"%d obligations missed past grace window (threshold: %d)",
				missed, plan.EscalateAfterMissed),
		}
		if err := e.events.Emit(ctx, ev); err != nil {
			// Per-row continue: log this failure but don't abort the sweep.
			// Next tick retries (idempotent: dup events de-duped at consumer).
			continue
		}
		if err := e.lc.Transition(ctx, TransitionRequest{
			PlanID:     plan.ID,
			ToState:    models.MonitoringPlanStateEscalated,
			ActorID:    uuid.Nil,
			ActorClass: ActorClassAlgorithmic,
			OccurredAt: cutoff,
			Notes:      fmt.Sprintf("escalated by sweeper: %d missed", missed),
		}); err != nil {
			// Per-row continue: state may already be escalated by the
			// threshold evaluator (race condition); the next tick will
			// observe the terminal state and skip.
			continue
		}
	}
	return nil
}
