# Monitoring Entity + Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the v2/v3 substrate's `MonitoringPlan` entity that *outlives* the recommendation that triggered it (per v2 §3 line 136), with observation-obligation tracking, threshold-cross detection that produces new Events, and an escalator that fires when expected observations don't land. This closes the outcome loop the v2 spec calls out as the cause of the Ramsey 50% non-implementation problem: "monitoring is treated as a free-text follow-up note attached to a closed recommendation. That's why the outcome loop never closes."

**Architecture:** Same patterns as Plans 0.1/0.2. Entity in `shared/v2_substrate/models/monitoring_plan.go`, lifecycle in `shared/v2_substrate/monitoring/`, root migration `025_monitoring_lifecycle.sql`. The MonitoringPlan references the Recommendation that spawned it but persists independently — a recommendation can close while its monitoring runs another 30 days. Threshold-cross events feed back into the Recommendation trigger surface (Plan 0.1's Event substrate).

**Tech Stack:** Go, PostgreSQL, depends on Plan 0.1 (Recommendation entity, Event entity already shipped, EvidenceTrace).

---

## File Structure

**New files:**
- `shared/v2_substrate/models/monitoring_plan.go` — entity + state enum + threshold-spec JSONB type
- `shared/v2_substrate/models/monitoring_plan_test.go`
- `shared/v2_substrate/monitoring/store.go` — `Store` + `PostgresStore`
- `shared/v2_substrate/monitoring/store_test.go`
- `shared/v2_substrate/monitoring/lifecycle.go` — transition engine
- `shared/v2_substrate/monitoring/lifecycle_test.go`
- `shared/v2_substrate/monitoring/threshold_evaluator.go` — observation-arrival check + threshold-cross
- `shared/v2_substrate/monitoring/threshold_evaluator_test.go`
- `shared/v2_substrate/monitoring/escalator.go` — sweeps overdue plans + missing-observation plans
- `shared/v2_substrate/monitoring/escalator_test.go`
- `migrations/025_monitoring_lifecycle.sql`
- `migrations/025_monitoring_lifecycle_rollback.sql`

**Modified files:**
- `shared/v2_substrate/models/enums.go` — append `MonitoringPlanState*`, `MonitoringObligationType*`

---

### Task 1: Define MonitoringPlan entity

**Files:**
- Create: `shared/v2_substrate/models/monitoring_plan.go`
- Modify: `shared/v2_substrate/models/enums.go`
- Test: `shared/v2_substrate/models/monitoring_plan_test.go`

States: `pending → active → completed | escalated | abandoned`
- `pending` — created but not yet started (recommendation not yet implemented)
- `active` — collecting observations
- `completed` — all obligations satisfied
- `escalated` — threshold crossed; new Event emitted; recommendation re-triggered
- `abandoned` — recommendation reversed before monitoring meaningful

- [ ] **Step 1: Append constants to enums.go**

```go
const (
	MonitoringPlanStatePending    = "pending"
	MonitoringPlanStateActive     = "active"
	MonitoringPlanStateCompleted  = "completed"
	MonitoringPlanStateEscalated  = "escalated"
	MonitoringPlanStateAbandoned  = "abandoned"
)

const (
	MonitoringObligationTypeObservation     = "observation"
	MonitoringObligationTypeFollowUpReview  = "follow_up_review"
	MonitoringObligationTypeBehaviouralChart = "behavioural_chart"
	MonitoringObligationTypeLab             = "lab"
)

func IsValidMonitoringPlanState(s string) bool {
	switch s {
	case MonitoringPlanStatePending, MonitoringPlanStateActive,
		MonitoringPlanStateCompleted, MonitoringPlanStateEscalated,
		MonitoringPlanStateAbandoned:
		return true
	}
	return false
}

var monitoringTransitions = map[string]map[string]bool{
	MonitoringPlanStatePending: {
		MonitoringPlanStateActive:    true,
		MonitoringPlanStateAbandoned: true,
	},
	MonitoringPlanStateActive: {
		MonitoringPlanStateCompleted: true,
		MonitoringPlanStateEscalated: true,
		MonitoringPlanStateAbandoned: true,
	},
	// completed/escalated/abandoned are terminal
}

func IsValidMonitoringTransition(from, to string) bool {
	if !IsValidMonitoringPlanState(from) || !IsValidMonitoringPlanState(to) {
		return false
	}
	return monitoringTransitions[from][to]
}
```

- [ ] **Step 2: Write failing test**

```go
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMonitoringPlanJSONRoundTrip(t *testing.T) {
	due := time.Now().Add(14 * 24 * time.Hour).UTC()
	in := MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: uuid.New(),
		ResidentID:       uuid.New(),
		State:            MonitoringPlanStateActive,
		Obligations: []MonitoringObligation{
			{
				Type:               MonitoringObligationTypeObservation,
				ObservationCode:    "blood_pressure",
				FrequencyHours:     24,
				DueAt:              due,
				ThresholdSpec:      "systolic > 160 OR systolic < 90",
				FulfilledAt:        nil,
			},
		},
		StartedAt:        time.Now().UTC().Truncate(time.Microsecond),
		ExpectedEndAt:    due,
		EscalateAfterMissed: 2,
		CreatedAt:        time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:        time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, _ := json.Marshal(in)
	var out MonitoringPlan
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.State != in.State || len(out.Obligations) != 1 {
		t.Errorf("round-trip mismatch: %+v", out)
	}
}

func TestMonitoringTransitionMatrix(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{MonitoringPlanStatePending, MonitoringPlanStateActive, true},
		{MonitoringPlanStateActive, MonitoringPlanStateCompleted, true},
		{MonitoringPlanStateActive, MonitoringPlanStateEscalated, true},
		{MonitoringPlanStateCompleted, MonitoringPlanStateActive, false}, // terminal
		{MonitoringPlanStateEscalated, MonitoringPlanStateActive, false},
	}
	for _, c := range cases {
		if got := IsValidMonitoringTransition(c.from, c.to); got != c.want {
			t.Errorf("%s -> %s = %v want %v", c.from, c.to, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Run, expect failure**

- [ ] **Step 4: Implement entity**

Create `shared/v2_substrate/models/monitoring_plan.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

// MonitoringPlan is the v2/v3 substrate entity that ensures the outcome
// loop closes after a recommendation is implemented. Per v2 §3 line 136:
// "monitoring outlives the recommendation that triggered it. The cessation
// closes Monday; the monitoring plan ('watch for urinary retention 14 days,
// falls 30 days, cognition 30 days') runs for a month."
//
// MonitoringPlan can be live with multiple ObligationTypes (an observation
// to land, a follow-up review to occur, a behavioural chart to populate).
// Threshold crossings produce new Events that re-enter the Recommendation
// trigger surface.
type MonitoringPlan struct {
	ID                  uuid.UUID              `json:"id"`
	RecommendationID    uuid.UUID              `json:"recommendation_id"`
	ResidentID          uuid.UUID              `json:"resident_id"`
	State               string                 `json:"state"` // see MonitoringPlanState*
	Obligations         []MonitoringObligation `json:"obligations"`
	StartedAt           time.Time              `json:"started_at"`
	ExpectedEndAt       time.Time              `json:"expected_end_at"`
	EscalateAfterMissed int                    `json:"escalate_after_missed"` // # missed obligations before auto-escalate
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// MonitoringObligation is one expected observation/review/chart entry the
// plan tracks. FulfilledAt nullable; DueAt drives the escalator sweep.
type MonitoringObligation struct {
	Type               string     `json:"type"` // see MonitoringObligationType*
	ObservationCode    string     `json:"observation_code,omitempty"` // e.g. "blood_pressure"
	FrequencyHours     int        `json:"frequency_hours,omitempty"`  // 0 = one-shot
	DueAt              time.Time  `json:"due_at"`
	ThresholdSpec      string     `json:"threshold_spec,omitempty"` // CQL-evaluable string
	FulfilledAt        *time.Time `json:"fulfilled_at,omitempty"`
	FulfilledByObsID   *uuid.UUID `json:"fulfilled_by_obs_id,omitempty"`
	ThresholdCrossedAt *time.Time `json:"threshold_crossed_at,omitempty"`
}
```

- [ ] **Step 5: Run, expect pass; commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/monitoring_plan.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/monitoring_plan_test.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/enums.go
git commit -m "feat(substrate): add MonitoringPlan entity with 5-state lifecycle"
```

---

### Task 2: Migration 025

**Files:**
- Create: `migrations/025_monitoring_lifecycle.sql`
- Create: `migrations/025_monitoring_lifecycle_rollback.sql`

- [ ] **Step 1: Write migration**

```sql
BEGIN;

CREATE TABLE monitoring_plans (
    id                    UUID PRIMARY KEY,
    recommendation_id     UUID NOT NULL,
    resident_id           UUID NOT NULL,
    state                 TEXT NOT NULL CHECK (state IN (
                              'pending','active','completed',
                              'escalated','abandoned')),
    obligations           JSONB NOT NULL,
    started_at            TIMESTAMPTZ NOT NULL,
    expected_end_at       TIMESTAMPTZ NOT NULL,
    escalate_after_missed INT NOT NULL DEFAULT 2,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitoring_recommendation ON monitoring_plans (recommendation_id);
CREATE INDEX idx_monitoring_resident       ON monitoring_plans (resident_id);
CREATE INDEX idx_monitoring_state          ON monitoring_plans (state);
CREATE INDEX idx_monitoring_active_sweep   ON monitoring_plans (expected_end_at)
    WHERE state = 'active';

-- Pre-extracted obligation rows for query-friendly scanning
CREATE OR REPLACE VIEW monitoring_obligations_unrolled AS
SELECT
    mp.id            AS plan_id,
    mp.resident_id,
    mp.state         AS plan_state,
    obligation->>'type' AS obligation_type,
    obligation->>'observation_code' AS observation_code,
    (obligation->>'due_at')::TIMESTAMPTZ AS due_at,
    obligation->>'fulfilled_at' AS fulfilled_at,
    obligation->>'threshold_crossed_at' AS threshold_crossed_at
FROM monitoring_plans mp,
LATERAL jsonb_array_elements(mp.obligations) AS obligation;

COMMIT;
```

- [ ] **Step 2: Rollback**

```sql
BEGIN;
DROP VIEW IF EXISTS monitoring_obligations_unrolled;
DROP TABLE IF EXISTS monitoring_plans;
COMMIT;
```

- [ ] **Step 3: Apply, verify; commit**

```bash
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -f migrations/025_monitoring_lifecycle.sql
git add migrations/025_monitoring_lifecycle.sql migrations/025_monitoring_lifecycle_rollback.sql
git commit -m "feat(migrations): 025 monitoring lifecycle table + obligation view"
```

---

### Task 3: Store with obligation-update support

**Files:**
- Create: `shared/v2_substrate/monitoring/store.go`
- Create: `shared/v2_substrate/monitoring/store_test.go`

Pattern matches Plan 0.1 Task 4 / Plan 0.2 Task 3. Adds `MarkObligationFulfilled(planID, obligationIdx, observationID)` so the threshold evaluator (Task 5) can update individual obligations atomically.

- [ ] **Step 1-5: Write the store following the established pattern**

Interface:

```go
type Store interface {
	Create(ctx context.Context, p *models.MonitoringPlan) error
	Get(ctx context.Context, id uuid.UUID) (*models.MonitoringPlan, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string) error
	MarkObligationFulfilled(ctx context.Context, planID uuid.UUID,
		obligationIdx int, observationID uuid.UUID, fulfilledAt time.Time) error
	MarkThresholdCrossed(ctx context.Context, planID uuid.UUID,
		obligationIdx int, crossedAt time.Time) error
	ListActiveOverdue(ctx context.Context, before time.Time) ([]models.MonitoringPlan, error)
	ListByRecommendation(ctx context.Context, recID uuid.UUID) ([]models.MonitoringPlan, error)
}
```

The PostgresStore implements each via `UPDATE monitoring_plans SET obligations = jsonb_set(obligations, '{N,fulfilled_at}', to_jsonb($2::timestamptz))` style mutations.

Test cases mirror Plan 0.2 Task 3: Create+Get round-trip; FindActive lookup; obligation mutation persists across reads.

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/store.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/store_test.go
git commit -m "feat(substrate): MonitoringPlan Store with obligation-mutation API"
```

---

### Task 4: Lifecycle engine

**Files:**
- Create: `shared/v2_substrate/monitoring/lifecycle.go`
- Create: `shared/v2_substrate/monitoring/lifecycle_test.go`

Same shape as Plan 0.2 Task 4. Transition validates via `models.IsValidMonitoringTransition`, emits EvidenceTrace edge, persists state.

- [ ] **Step 1-5: Implement following the pattern**

Sentinel error: `ErrInvalidTransition`. EvidenceEdge struct: `{PlanID, FromState, ToState, ActorID, ActorClass (per Plan 0.1), OccurredAt, Notes}`. Lifecycle constructor takes `Store` and `EdgeStore`.

Test cases: happy path (`pending → active → completed`), forbidden (`completed → active`), forbidden (`pending → escalated` skipping active).

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/lifecycle.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/lifecycle_test.go
git commit -m "feat(substrate): MonitoringPlan Lifecycle with EvidenceTrace emission"
```

---

### Task 5: Threshold evaluator + Event emission

**Files:**
- Create: `shared/v2_substrate/monitoring/threshold_evaluator.go`
- Create: `shared/v2_substrate/monitoring/threshold_evaluator_test.go`

This is the architecturally distinguishing piece. When an Observation lands, the threshold evaluator: (a) finds the active monitoring plan(s) referencing this resident + observation_code, (b) marks the matching obligation fulfilled, (c) evaluates the threshold spec against the new value, (d) if crossed, emits an Event into the substrate (which feeds Plan 0.1's Recommendation trigger surface) and transitions the plan to `escalated`.

- [ ] **Step 1: Write failing test**

```go
package monitoring

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

// fakeEventEmitter captures Events the evaluator would persist.
type fakeEventEmitter struct{ events []models.Event }

func (f *fakeEventEmitter) Emit(_ context.Context, e models.Event) error {
	f.events = append(f.events, e)
	return nil
}

func TestThresholdEvaluator_CrossingEmitsEventAndEscalates(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdges{}
	lc := NewLifecycle(store, edges)
	emitter := &fakeEventEmitter{}
	evaluator := NewThresholdEvaluator(store, lc, emitter)
	ctx := context.Background()

	resident := uuid.New()
	plan := models.MonitoringPlan{
		ID: uuid.New(), RecommendationID: uuid.New(), ResidentID: resident,
		State: models.MonitoringPlanStateActive,
		Obligations: []models.MonitoringObligation{
			{Type: models.MonitoringObligationTypeObservation,
				ObservationCode: "potassium",
				FrequencyHours:  24,
				DueAt:           time.Now().Add(2 * time.Hour),
				ThresholdSpec:   "value > 5.5"},
		},
		StartedAt: time.Now(), ExpectedEndAt: time.Now().Add(7 * 24 * time.Hour),
		EscalateAfterMissed: 2,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := store.Create(ctx, &plan); err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM monitoring_plans WHERE id = $1", plan.ID)

	obs := models.Observation{
		ID: uuid.New(), ResidentID: resident,
		Code: "potassium", Value: 5.8, ObservedAt: time.Now(),
	}
	if err := evaluator.Evaluate(ctx, obs); err != nil {
		t.Fatalf("evaluate: %v", err)
	}

	got, _ := store.Get(ctx, plan.ID)
	if got.State != models.MonitoringPlanStateEscalated {
		t.Errorf("expected escalated; got %q", got.State)
	}
	if len(emitter.events) != 1 {
		t.Errorf("expected 1 event emitted; got %d", len(emitter.events))
	}
	if emitter.events[0].Type != "monitoring_threshold_crossed" {
		t.Errorf("event type wrong: %q", emitter.events[0].Type)
	}
}
```

(`fakeEdges` is the same fake from Task 4's test file; reuse via package-level definition.)

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement**

```go
package monitoring

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

// EventEmitter is the Event-substrate boundary. Production wiring uses
// kb-20-patient-profile's event store; tests use a fake.
type EventEmitter interface {
	Emit(ctx context.Context, e models.Event) error
}

type ThresholdEvaluator struct {
	store    Store
	lc       *Lifecycle
	events   EventEmitter
	now      func() time.Time
}

func NewThresholdEvaluator(store Store, lc *Lifecycle, events EventEmitter) *ThresholdEvaluator {
	return &ThresholdEvaluator{
		store: store, lc: lc, events: events,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// Evaluate inspects all active monitoring plans for the resident +
// observation code, marks matching obligations fulfilled, and emits an
// Event + transitions the plan to escalated when the threshold is crossed.
func (e *ThresholdEvaluator) Evaluate(ctx context.Context, obs models.Observation) error {
	const q = `
SELECT id FROM monitoring_plans
WHERE resident_id = $1 AND state = 'active'
  AND obligations::text LIKE '%' || $2 || '%'
LIMIT 50`
	rows, err := e.store.(*PostgresStore).db.QueryContext(ctx, q, obs.ResidentID, obs.Code)
	if err != nil {
		return err
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
			continue
		}
		for i, ob := range plan.Obligations {
			if ob.Type != models.MonitoringObligationTypeObservation || ob.ObservationCode != obs.Code {
				continue
			}
			if ob.FulfilledAt != nil {
				continue // already marked
			}
			if err := e.store.MarkObligationFulfilled(ctx, pid, i, obs.ID, e.now()); err != nil {
				return err
			}
			if crossThreshold(ob.ThresholdSpec, obs.Value) {
				if err := e.store.MarkThresholdCrossed(ctx, pid, i, e.now()); err != nil {
					return err
				}
				ev := models.Event{
					ID:         uuid.New(),
					ResidentID: obs.ResidentID,
					Type:       "monitoring_threshold_crossed",
					OccurredAt: e.now(),
					Data: map[string]any{
						"plan_id":          pid,
						"obligation_index": i,
						"observation_id":   obs.ID,
						"observed_value":   obs.Value,
						"threshold_spec":   ob.ThresholdSpec,
					},
				}
				if err := e.events.Emit(ctx, ev); err != nil {
					return err
				}
				if err := e.lc.Transition(ctx, TransitionRequest{
					PlanID: pid, ToState: models.MonitoringPlanStateEscalated,
					ActorID: uuid.Nil, ActorClass: "algorithmic",
					OccurredAt: e.now(),
					Notes:      "threshold crossed",
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// crossThreshold parses a tiny DSL: "value <op> <number>" or "value <op> <num> AND value <op> <num>".
// Production should swap this for CQL evaluation through the HAPI runtime
// (Plan 0.5). For Phase 0 we cover the common cases inline with explicit
// tests.
func crossThreshold(spec string, value float64) bool {
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
	for _, op := range []string{">=", "<=", ">", "<", "=="} {
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
```

Add a unit test for `crossThreshold`:

```go
func TestCrossThreshold(t *testing.T) {
	cases := []struct {
		spec  string
		value float64
		want  bool
	}{
		{"value > 5.5", 5.8, true},
		{"value > 5.5", 5.5, false},
		{"value < 3.0", 2.9, true},
		{"value > 5.5 OR value < 3.0", 2.5, true},
		{"value >= 100", 100, true},
		{"bogus spec", 1, false},
	}
	for _, c := range cases {
		if got := crossThreshold(c.spec, c.value); got != c.want {
			t.Errorf("crossThreshold(%q, %v) = %v want %v", c.spec, c.value, got, c.want)
		}
	}
}
```

- [ ] **Step 4-6: Run, pass, commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/threshold_evaluator.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/threshold_evaluator_test.go
git commit -m "feat(substrate): MonitoringPlan threshold evaluator with Event emission"
```

---

### Task 6: Escalator for missing observations

**Files:**
- Create: `shared/v2_substrate/monitoring/escalator.go`
- Create: `shared/v2_substrate/monitoring/escalator_test.go`

If observations don't land within `due_at + grace_window`, escalator emits a `monitoring_obligation_missed` Event and transitions the plan to `escalated` once `EscalateAfterMissed` threshold is reached.

- [ ] **Step 1-5: Implement following Plan 0.1 deferred escalator pattern**

```go
package monitoring

import (
	"context"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

type Escalator struct {
	store        Store
	lc           *Lifecycle
	events       EventEmitter
	now          func() time.Time
	graceWindow  time.Duration
}

func NewEscalator(store Store, lc *Lifecycle, events EventEmitter, grace time.Duration) *Escalator {
	return &Escalator{
		store: store, lc: lc, events: events,
		now: func() time.Time { return time.Now().UTC() },
		graceWindow: grace,
	}
}

// RunOnce sweeps active plans whose obligations are past due.
func (e *Escalator) RunOnce(ctx context.Context) error {
	cutoff := e.now().Add(-e.graceWindow)
	plans, err := e.store.ListActiveOverdue(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, plan := range plans {
		missed := 0
		for _, ob := range plan.Obligations {
			if ob.FulfilledAt == nil && ob.DueAt.Before(cutoff) {
				missed++
			}
		}
		if missed >= plan.EscalateAfterMissed {
			ev := models.Event{
				ID: uuid.New(), ResidentID: plan.ResidentID,
				Type: "monitoring_obligation_missed", OccurredAt: e.now(),
				Data: map[string]any{
					"plan_id": plan.ID, "missed_count": missed,
				},
			}
			if err := e.events.Emit(ctx, ev); err != nil {
				return err
			}
			if err := e.lc.Transition(ctx, TransitionRequest{
				PlanID: plan.ID, ToState: models.MonitoringPlanStateEscalated,
				ActorID: uuid.Nil, ActorClass: "algorithmic",
				OccurredAt: e.now(),
				Notes:      "obligations missed past grace window",
			}); err != nil {
				return err
			}
		}
	}
	return nil
}
```

Test seeds a plan with a past-due obligation, runs sweep, asserts plan is escalated and event emitted.

- [ ] **Step 6: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/escalator.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/escalator_test.go
git commit -m "feat(substrate): MonitoringPlan escalator for missing observations"
```

---

### Task 7: Integration test — Recommendation lifecycle spawns MonitoringPlan

**Files:**
- Create: `shared/v2_substrate/monitoring/integration_test.go`

Exercise: Recommendation transitions to `implemented`, a MonitoringPlan is created (manual creation in test; the spawn is a craft-engine concern in Phase 2). Observation lands that crosses threshold. Plan transitions to `escalated`. Event fires. New Recommendation can be triggered (substrate state correct).

- [ ] **Step 1-5: Write integration test, verify, commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/monitoring/integration_test.go
git commit -m "test(substrate): MonitoringPlan threshold-cross integration with Event substrate"
```

---

## Spec coverage

- [x] MonitoringPlan outlives Recommendation (separate entity, separate lifecycle)
- [x] Threshold-cross emits new Event into trigger surface
- [x] Missing-observation escalation
- [x] EvidenceTrace emission per state transition
- [x] Replaces `MonitoringHelpers.cql` TODO(wave-1-runtime) markers (consumed in Plan 0.5)

Plan complete and saved.
