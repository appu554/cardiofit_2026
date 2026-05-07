package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// EventEmitter is the Event-substrate boundary. Production wiring uses
// kb-20-patient-profile's event store; tests use a fake. The threshold
// evaluator emits exactly one Event per threshold crossing — the Event's
// EventType is models.EventTypeMonitoringThresholdCrossed.
type EventEmitter interface {
	Emit(ctx context.Context, e models.Event) error
}

// ThresholdEvaluator inspects each incoming Observation against any active
// monitoring plans referencing the resident + observation code. When an
// observation lands:
//  1. If the obligation hasn't been fulfilled yet, mark fulfilled (with
//     the observation ID and now timestamp).
//  2. If the observation value crosses the threshold spec, mark
//     threshold-crossed AND emit a models.EventTypeMonitoringThresholdCrossed
//     Event AND transition the plan to escalated via Lifecycle.
//
// This is the closure of the v2 §3 line 136 outcome loop: a threshold-cross
// produces a new Event that re-enters the Recommendation trigger surface.
type ThresholdEvaluator struct {
	store  Store
	lc     *Lifecycle
	events EventEmitter
	now    func() time.Time
}

// NewThresholdEvaluator constructs an evaluator. now() is injectable for
// deterministic tests.
func NewThresholdEvaluator(store Store, lc *Lifecycle, events EventEmitter) *ThresholdEvaluator {
	return &ThresholdEvaluator{
		store: store, lc: lc, events: events,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// Evaluate inspects all active monitoring plans for the observation's
// resident + LOINCCode, marks matching obligations fulfilled, and emits
// an Event + transitions the plan to escalated when the threshold is
// crossed. Observations with nil Value are ignored (textual observations
// can't be threshold-evaluated by this engine).
func (e *ThresholdEvaluator) Evaluate(ctx context.Context, obs models.Observation) error {
	if obs.Value == nil {
		return nil
	}

	// Find active plans for this resident. We could narrow by LOINCCode
	// using monitoring_obligations_unrolled (Migration 025 view), but for
	// Plan 0.3 scope we scan plans-by-resident in Go. The view-based
	// lookup is a Plan 0.4 optimisation.
	const q = `
SELECT id FROM monitoring_plans
WHERE resident_id = $1 AND state = 'active'
LIMIT 100`
	pgStore, ok := e.store.(*PostgresStore)
	if !ok {
		return fmt.Errorf("threshold evaluator requires *PostgresStore for active-plan query")
	}
	rows, err := pgStore.db.QueryContext(ctx, q, obs.ResidentID)
	if err != nil {
		return fmt.Errorf("query active plans: %w", err)
	}
	defer rows.Close()

	var planIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		planIDs = append(planIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, pid := range planIDs {
		plan, err := e.store.Get(ctx, pid)
		if err != nil {
			continue // plan disappeared between query and Get; skip
		}
		for i, ob := range plan.Obligations {
			if ob.Type != models.MonitoringObligationTypeObservation {
				continue
			}
			if ob.ObservationCode != obs.LOINCCode {
				continue
			}
			if ob.FulfilledAt != nil {
				continue // already fulfilled
			}
			// Mark fulfilled
			if err := e.store.MarkObligationFulfilled(ctx, pid, i, obs.ID, e.now()); err != nil {
				return fmt.Errorf("mark fulfilled: %w", err)
			}
			// Check threshold
			if !crossThreshold(ob.ThresholdSpec, *obs.Value) {
				continue // in-range; no event, no escalation
			}
			// Threshold crossed — mark + emit + escalate
			if err := e.store.MarkThresholdCrossed(ctx, pid, i, e.now()); err != nil {
				return fmt.Errorf("mark threshold crossed: %w", err)
			}
			payload, _ := json.Marshal(map[string]any{
				"plan_id":          pid,
				"obligation_index": i,
				"observation_id":   obs.ID,
				"observed_value":   *obs.Value,
				"threshold_spec":   ob.ThresholdSpec,
			})
			ev := models.Event{
				ID:                    uuid.New(),
				EventType:             models.EventTypeMonitoringThresholdCrossed,
				ResidentID:            obs.ResidentID,
				OccurredAt:            e.now(),
				ReportedByRef:         uuid.Nil, // system-generated
				DescriptionStructured: payload,
				DescriptionFreeText: fmt.Sprintf(
					"Observation %s value %v crossed threshold %q",
					obs.LOINCCode, *obs.Value, ob.ThresholdSpec),
			}
			if err := e.events.Emit(ctx, ev); err != nil {
				return fmt.Errorf("emit event: %w", err)
			}
			if err := e.lc.Transition(ctx, TransitionRequest{
				PlanID:     pid,
				ToState:    models.MonitoringPlanStateEscalated,
				ActorID:    uuid.Nil,
				ActorClass: ActorClassAlgorithmic,
				OccurredAt: e.now(),
				Notes:      "threshold crossed",
			}); err != nil {
				return fmt.Errorf("escalate plan: %w", err)
			}
		}
	}
	return nil
}

// crossThreshold parses a tiny DSL: "value <op> <number>" or
// "<clause> OR <clause>". Production should swap this for CQL evaluation
// through the HAPI runtime (Plan 0.5). For Plan 0.3 we cover the common
// cases inline with explicit tests.
//
// Supported operators: > < >= <= ==
// Supported combinator: OR (case-sensitive, space-padded)
func crossThreshold(spec string, value float64) bool {
	if spec == "" {
		return false
	}
	clauses := strings.Split(spec, " OR ")
	for _, c := range clauses {
		if evalClause(strings.TrimSpace(c), value) {
			return true
		}
	}
	return false
}

func evalClause(c string, value float64) bool {
	c = strings.TrimSpace(c)
	c = strings.TrimPrefix(c, "value")
	c = strings.TrimSpace(c)
	for _, op := range []string{">=", "<=", "==", ">", "<"} {
		if strings.HasPrefix(c, op) {
			rest := strings.TrimSpace(strings.TrimPrefix(c, op))
			n, err := strconv.ParseFloat(rest, 64)
			if err != nil {
				return false
			}
			switch op {
			case ">":
				return value > n
			case "<":
				return value < n
			case ">=":
				return value >= n
			case "<=":
				return value <= n
			case "==":
				return value == n
			}
		}
	}
	return false
}
