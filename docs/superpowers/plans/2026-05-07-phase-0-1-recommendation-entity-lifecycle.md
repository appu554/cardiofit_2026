# Recommendation Entity + Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the v2/v3 substrate's missing keystone — a first-class `Recommendation` entity with a 9-state lifecycle, EvidenceTrace integration, and a deferred-state escalator — so that MVP exit criterion is demonstrable, RIR (Recommendation Implementation Rate, the v3 Layer-C North Star) is computable, and downstream v3 modules (craft engine, self-visibility) have a substrate to render against.

**Architecture:** Mirror the existing v2 substrate pattern: pure Go entity in `shared/v2_substrate/models/recommendation.go`, lifecycle engine in `shared/v2_substrate/recommendation/`, root migration `023_recommendation_lifecycle.sql`. Every state transition emits an EvidenceTrace edge via the existing `evidence_trace/edge_store.go`. The `deferred` state is enforced with a forced `review_due_at` timestamp; an escalator worker sweeps overdue deferrals into a Worklist event. Consent gating is a *pre-condition check* on `Submit()`, not a separate state — it returns `ErrConsentRequired` when a recommendation class needs Consent and none is active. (Consent and Monitoring entities are sibling plans 0.2 and 0.3; this plan stubs the integration points.)

**Tech Stack:** Go 1.21+, `google/uuid`, Go standard library `testing`, PostgreSQL 15 (jsonb columns), existing substrate patterns (UUID PKs, JSONB for structured fields, table-driven tests with JSON round-trip verification).

---

## File Structure

**New files:**
- `shared/v2_substrate/models/recommendation.go` — entity struct, state enum, transition matrix, JSONB sub-types
- `shared/v2_substrate/models/recommendation_test.go` — JSON round-trip + transition validity tests
- `shared/v2_substrate/recommendation/store.go` — `Store` interface + `PostgresStore` impl
- `shared/v2_substrate/recommendation/store_test.go` — store CRUD + concurrent transition tests
- `shared/v2_substrate/recommendation/lifecycle.go` — `Lifecycle` engine: transition with guard + EvidenceTrace emission
- `shared/v2_substrate/recommendation/lifecycle_test.go` — transition guard tests, EvidenceTrace emission tests
- `shared/v2_substrate/recommendation/deferred_escalator.go` — periodic sweep worker
- `shared/v2_substrate/recommendation/deferred_escalator_test.go` — escalator behaviour tests
- `shared/v2_substrate/recommendation/rir.go` — RIR computation query (v3 Layer-C metric)
- `shared/v2_substrate/recommendation/rir_test.go` — RIR query tests
- `migrations/023_recommendation_lifecycle.sql` — table + indexes + materialised view for RIR
- `migrations/023_recommendation_lifecycle_rollback.sql`

**Modified files:**
- `shared/v2_substrate/models/enums.go` — add `RecommendationState*`, `RecommendationType*`, `RecommendationUrgency*` constants
- `shared/v2_substrate/evidence_trace/edge_store.go:` (read-only review — confirm `EmitEdge` signature; do not modify)

**Conventions followed:**
- UUIDs everywhere (matches `medicine_use.go`, `event.go`)
- JSONB for structured sub-fields (matches `MedicineUse.Intent`, `MedicineUse.Target`)
- Pure Go std-lib testing (matches `medicine_use_test.go`)
- Migration numbering continues from `022` (root `migrations/`)

---

## Background context the engineer needs

**Why this entity is the keystone.** The v2 substrate ships with five other entities (Resident, Person+Role, MedicineUse, Observation, Event, EvidenceTrace) but no Recommendation entity. The CDS Hooks emitter currently produces *cards* (transient JSON payloads), not *entities* (durable rows). This means: (a) RIR cannot be computed because there's nothing to count; (b) the `deferred` state with forced review_date — which v2 §3 line 134 calls out as the cure for the Ramsey 50% non-implementation problem — has no place to live; (c) the v3 craft engine (kb-32) cannot render packets against a missing entity; (d) MVP exit criterion ("recommendation reaches the GP with rationale preserved, audit trail intact") is undemonstrable. Closing this single gap unblocks four downstream phases.

**The 9 states (v2 §3 line 134, v3 §3 line 174):**
`detected → drafted → submitted → viewed → decided → implemented → monitoring-active → outcome-recorded → closed`
Plus `deferred` as a parallel state reachable from `submitted` and `viewed`, with a forced `review_due_at` timestamp.

**Allowed transitions (the matrix):**
```
detected           → drafted | closed (rejected-pre-draft)
drafted            → submitted | closed (withdrawn)
submitted          → viewed | deferred | closed (rescinded)
viewed             → decided | deferred | closed (rejected-after-view)
deferred           → submitted (re-surfaced) | closed (expired-without-action)
decided            → implemented | closed (decided-no-action)
implemented        → monitoring-active | outcome-recorded
monitoring-active  → outcome-recorded
outcome-recorded   → closed
closed             → (terminal)
```

**EvidenceTrace integration.** Every transition writes a directed edge from the prior state node to the new state node, recording: `actor_role`, `actor_class` (human|algorithmic|human-with-suggestion|human-overriding), `inputs` (substrate refs), `reasoning_summary`, `alternatives_considered` (populated by the future craft engine; nullable for now). The `edge_store.go` API already exists — use `EmitEdge(ctx, edge)` directly.

**Consent gating.** Per v2 §3 line 140, when a recommendation class requires Consent and no `active` Consent matches, `Submit()` returns `ErrConsentRequired` with a structured detail. The Consent entity arrives in plan 0.2 — for now, the gate calls a `ConsentChecker` interface; the test suite ships an `AlwaysPassConsentChecker`. Plan 0.2 wires a real Postgres-backed checker.

**RIR formula (v3 §11 line 588):** `% of pharmacist recommendations with documented prescriber action within agreed window`. Operationally: `count(state in {decided, implemented, monitoring-active, outcome-recorded, closed-decided-no-action}) / count(submitted_at within window)`. Window is configurable (default 28 days, matching Ramsey 2025 measurement basis).

---

## Self-Review Anchors

After Task 9 (the final task), verify against the original spec:
- v2 §3 line 134 (9 states + deferred) → covered by Tasks 1, 4
- v3 §11 line 588 (RIR computation) → covered by Task 8
- v2 §3 line 140 (Consent gating) → covered by Task 5
- v3 §3 line 222 (algorithmic-vs-human distinction) → covered by Task 4
- MVP-3 / V1 critical path (deferred forced review_date) → covered by Tasks 6, 7

---

### Task 1: Define Recommendation entity model + state constants

**Files:**
- Create: `shared/v2_substrate/models/recommendation.go`
- Modify: `shared/v2_substrate/models/enums.go` (append constants)
- Test: `shared/v2_substrate/models/recommendation_test.go`

- [ ] **Step 1: Append state, type, urgency constants to enums.go**

Add at the bottom of `shared/v2_substrate/models/enums.go`:

```go
// RecommendationState* — the 9 lifecycle states plus deferred.
// See plan: docs/superpowers/plans/2026-05-07-phase-0-1-recommendation-entity-lifecycle.md
const (
	RecommendationStateDetected         = "detected"
	RecommendationStateDrafted          = "drafted"
	RecommendationStateSubmitted        = "submitted"
	RecommendationStateViewed           = "viewed"
	RecommendationStateDeferred         = "deferred"
	RecommendationStateDecided          = "decided"
	RecommendationStateImplemented      = "implemented"
	RecommendationStateMonitoringActive = "monitoring-active"
	RecommendationStateOutcomeRecorded  = "outcome-recorded"
	RecommendationStateClosed           = "closed"
)

// RecommendationType* — ordering hint per v3 §7 line 384.
// STOP > MONITOR > DOSE_CHANGE > ADD by acceptance probability.
const (
	RecommendationTypeStop       = "stop"
	RecommendationTypeMonitor    = "monitor"
	RecommendationTypeDoseChange = "dose_change"
	RecommendationTypeAdd        = "add"
)

// RecommendationUrgency* — three tiers per v3 §7 line 396.
const (
	RecommendationUrgencyRed   = "red"   // 24-48h
	RecommendationUrgencyAmber = "amber" // 1-2 weeks
	RecommendationUrgencyGreen = "green" // next review
)

// IsValidRecommendationState reports whether s is a known lifecycle state.
func IsValidRecommendationState(s string) bool {
	switch s {
	case RecommendationStateDetected, RecommendationStateDrafted,
		RecommendationStateSubmitted, RecommendationStateViewed,
		RecommendationStateDeferred, RecommendationStateDecided,
		RecommendationStateImplemented, RecommendationStateMonitoringActive,
		RecommendationStateOutcomeRecorded, RecommendationStateClosed:
		return true
	}
	return false
}
```

- [ ] **Step 2: Write the failing test for entity JSON round-trip**

Create `shared/v2_substrate/models/recommendation_test.go`:

```go
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRecommendationJSONRoundTrip(t *testing.T) {
	medUse := uuid.New()
	in := Recommendation{
		ID:           uuid.New(),
		ResidentID:   uuid.New(),
		AuthorID:     uuid.New(),
		State:        RecommendationStateDrafted,
		Type:         RecommendationTypeStop,
		Urgency:      RecommendationUrgencyAmber,
		Title:        "Cease oxybutynin",
		ClinicalContent: ClinicalContent{
			Issue:           "Anticholinergic burden contributing to fall risk",
			ClinicalContext: "87yo female, eGFR 32, recent fall, ACB 4",
			Rationale:       "DBI 0.8 attributable; alternatives reviewed",
			EvidenceRefs:    []string{"ADG-2025-Rec-42", "Beers-2023-OAB"},
			ProposedPlan:    "Cease oxybutynin 5mg BD; monitor for urinary retention 14 days",
			MonitoringPlan:  "Voiding diary 14 days; falls reassessment at 30 days",
		},
		MedicineUseRefs:    []uuid.UUID{medUse},
		ConsentRequired:    false,
		ReviewDueAt:        nil,
		SubmittedAt:        nil,
		CreatedAt:          time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:          time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Recommendation
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.ID != in.ID || out.State != in.State || out.Type != in.Type {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
	if out.ClinicalContent.Issue != in.ClinicalContent.Issue {
		t.Errorf("clinical content lost in round trip")
	}
	if len(out.MedicineUseRefs) != 1 || out.MedicineUseRefs[0] != medUse {
		t.Errorf("medicine use refs lost: %v", out.MedicineUseRefs)
	}
}

func TestIsValidRecommendationState(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{RecommendationStateDetected, true},
		{RecommendationStateDeferred, true},
		{RecommendationStateClosed, true},
		{"bogus", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsValidRecommendationState(c.s); got != c.want {
			t.Errorf("IsValidRecommendationState(%q)=%v want %v", c.s, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services && go test ./shared/v2_substrate/models/ -run TestRecommendation -v`
Expected: FAIL with `undefined: Recommendation` and `undefined: ClinicalContent`.

- [ ] **Step 4: Write the entity struct**

Create `shared/v2_substrate/models/recommendation.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

// Recommendation is a v2/v3 substrate entity capturing a proposed clinical
// action through its lifecycle. It is the keystone entity for the v3 product
// thesis: without persistence of recommendations, RIR cannot be computed,
// rationale survival cannot be measured, and the craft engine has no
// substrate to render against.
//
// State transitions are governed by recommendation.Lifecycle, which writes
// an EvidenceTrace edge per transition. Direct State mutation outside the
// Lifecycle engine is a contract violation.
//
// Canonical storage: migrations/023_recommendation_lifecycle.sql
// (table: recommendations).
type Recommendation struct {
	ID         uuid.UUID `json:"id"`
	ResidentID uuid.UUID `json:"resident_id"`
	AuthorID   uuid.UUID `json:"author_id"` // Person.id (typically the ACOP pharmacist)

	State   string `json:"state"`   // see RecommendationState* constants
	Type    string `json:"type"`    // see RecommendationType* constants
	Urgency string `json:"urgency"` // see RecommendationUrgency* constants

	Title           string          `json:"title"`
	ClinicalContent ClinicalContent `json:"clinical_content"` // v3 §7: invariant across framings

	// MedicineUseRefs links to MedicineUse entities this recommendation
	// targets (cease X, dose-change Y, add Z).
	MedicineUseRefs []uuid.UUID `json:"medicine_use_refs"`

	// ConsentRequired is set true at draft time when the recommendation class
	// requires a matching active Consent (e.g. psychotropic, restrictive
	// practice). The Lifecycle.Submit guard enforces this.
	ConsentRequired bool `json:"consent_required"`

	// ReviewDueAt is the forced review timestamp when in deferred state.
	// Nil for non-deferred states. The deferred escalator sweeps this column.
	ReviewDueAt *time.Time `json:"review_due_at,omitempty"`

	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClinicalContent is the audience-invariant clinical substance of a
// recommendation. v3 §7 line 416 mandates that this is recorded separately
// from any per-audience framing so a regulator audit query can verify
// content invariance across framings.
type ClinicalContent struct {
	Issue           string   `json:"issue"`
	ClinicalContext string   `json:"clinical_context"`
	Rationale       string   `json:"rationale"`
	EvidenceRefs    []string `json:"evidence_refs"`
	ProposedPlan    string   `json:"proposed_plan"`
	MonitoringPlan  string   `json:"monitoring_plan"`
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./shared/v2_substrate/models/ -run TestRecommendation -v`
Expected: PASS for both `TestRecommendationJSONRoundTrip` and `TestIsValidRecommendationState`.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/recommendation.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/recommendation_test.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/enums.go
git commit -m "feat(substrate): add Recommendation entity with 9-state lifecycle constants"
```

---

### Task 2: Define the transition matrix

**Files:**
- Modify: `shared/v2_substrate/models/recommendation.go` (append `IsValidTransition`)
- Test: `shared/v2_substrate/models/recommendation_test.go` (append transition tests)

- [ ] **Step 1: Write the failing test**

Append to `shared/v2_substrate/models/recommendation_test.go`:

```go
func TestRecommendationTransitionMatrix(t *testing.T) {
	type tc struct {
		from, to string
		want     bool
	}
	cases := []tc{
		// Happy path
		{RecommendationStateDetected, RecommendationStateDrafted, true},
		{RecommendationStateDrafted, RecommendationStateSubmitted, true},
		{RecommendationStateSubmitted, RecommendationStateViewed, true},
		{RecommendationStateViewed, RecommendationStateDecided, true},
		{RecommendationStateDecided, RecommendationStateImplemented, true},
		{RecommendationStateImplemented, RecommendationStateMonitoringActive, true},
		{RecommendationStateMonitoringActive, RecommendationStateOutcomeRecorded, true},
		{RecommendationStateOutcomeRecorded, RecommendationStateClosed, true},

		// Deferred branches
		{RecommendationStateSubmitted, RecommendationStateDeferred, true},
		{RecommendationStateViewed, RecommendationStateDeferred, true},
		{RecommendationStateDeferred, RecommendationStateSubmitted, true},
		{RecommendationStateDeferred, RecommendationStateClosed, true},

		// Direct-to-closed escapes
		{RecommendationStateDetected, RecommendationStateClosed, true},
		{RecommendationStateDrafted, RecommendationStateClosed, true},
		{RecommendationStateSubmitted, RecommendationStateClosed, true},
		{RecommendationStateViewed, RecommendationStateClosed, true},
		{RecommendationStateDecided, RecommendationStateClosed, true},
		{RecommendationStateImplemented, RecommendationStateOutcomeRecorded, true},

		// Forbidden: terminal
		{RecommendationStateClosed, RecommendationStateDrafted, false},
		{RecommendationStateClosed, RecommendationStateSubmitted, false},

		// Forbidden: skipping decided
		{RecommendationStateViewed, RecommendationStateImplemented, false},
		{RecommendationStateSubmitted, RecommendationStateDecided, false},

		// Forbidden: backwards
		{RecommendationStateDecided, RecommendationStateSubmitted, false},
		{RecommendationStateMonitoringActive, RecommendationStateImplemented, false},

		// Forbidden: bogus
		{"bogus", RecommendationStateDrafted, false},
		{RecommendationStateDrafted, "bogus", false},
	}
	for _, c := range cases {
		if got := IsValidTransition(c.from, c.to); got != c.want {
			t.Errorf("IsValidTransition(%q, %q) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./shared/v2_substrate/models/ -run TestRecommendationTransitionMatrix -v`
Expected: FAIL with `undefined: IsValidTransition`.

- [ ] **Step 3: Implement IsValidTransition**

Append to `shared/v2_substrate/models/recommendation.go`:

```go
// validTransitions encodes the recommendation lifecycle DAG. A pair (from, to)
// is in the map iff the transition is permitted. Direct mutation outside
// recommendation.Lifecycle is a contract violation; this function exists so
// the Lifecycle engine and storage layer share one source of truth.
var validTransitions = map[string]map[string]bool{
	RecommendationStateDetected: {
		RecommendationStateDrafted: true,
		RecommendationStateClosed:  true,
	},
	RecommendationStateDrafted: {
		RecommendationStateSubmitted: true,
		RecommendationStateClosed:    true,
	},
	RecommendationStateSubmitted: {
		RecommendationStateViewed:   true,
		RecommendationStateDeferred: true,
		RecommendationStateClosed:   true,
	},
	RecommendationStateViewed: {
		RecommendationStateDecided:  true,
		RecommendationStateDeferred: true,
		RecommendationStateClosed:   true,
	},
	RecommendationStateDeferred: {
		RecommendationStateSubmitted: true, // re-surfaced
		RecommendationStateClosed:    true, // expired without action
	},
	RecommendationStateDecided: {
		RecommendationStateImplemented: true,
		RecommendationStateClosed:      true, // decided-no-action
	},
	RecommendationStateImplemented: {
		RecommendationStateMonitoringActive: true,
		RecommendationStateOutcomeRecorded:  true, // skip monitoring if not warranted
	},
	RecommendationStateMonitoringActive: {
		RecommendationStateOutcomeRecorded: true,
	},
	RecommendationStateOutcomeRecorded: {
		RecommendationStateClosed: true,
	},
	// RecommendationStateClosed is terminal; no entry.
}

// IsValidTransition reports whether the lifecycle DAG permits from → to.
func IsValidTransition(from, to string) bool {
	if !IsValidRecommendationState(from) || !IsValidRecommendationState(to) {
		return false
	}
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./shared/v2_substrate/models/ -run TestRecommendationTransitionMatrix -v`
Expected: PASS — all 26 cases.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/recommendation.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/recommendation_test.go
git commit -m "feat(substrate): add Recommendation lifecycle transition matrix"
```

---

### Task 3: Postgres migration for recommendations table

**Files:**
- Create: `migrations/023_recommendation_lifecycle.sql`
- Create: `migrations/023_recommendation_lifecycle_rollback.sql`

- [ ] **Step 1: Write the migration**

Create `migrations/023_recommendation_lifecycle.sql`:

```sql
-- Migration 023: Recommendation lifecycle
-- Adds the keystone v2/v3 substrate entity. See plan:
-- docs/superpowers/plans/2026-05-07-phase-0-1-recommendation-entity-lifecycle.md

BEGIN;

CREATE TABLE recommendations (
    id                  UUID PRIMARY KEY,
    resident_id         UUID NOT NULL,
    author_id           UUID NOT NULL,

    state               TEXT NOT NULL CHECK (state IN (
                            'detected','drafted','submitted','viewed','deferred',
                            'decided','implemented','monitoring-active',
                            'outcome-recorded','closed')),
    type                TEXT NOT NULL CHECK (type IN (
                            'stop','monitor','dose_change','add')),
    urgency             TEXT NOT NULL CHECK (urgency IN ('red','amber','green')),

    title               TEXT NOT NULL,
    clinical_content    JSONB NOT NULL,
    medicine_use_refs   UUID[] NOT NULL DEFAULT '{}',

    consent_required    BOOLEAN NOT NULL DEFAULT FALSE,
    review_due_at       TIMESTAMPTZ,
    submitted_at        TIMESTAMPTZ,
    decided_at          TIMESTAMPTZ,
    closed_at           TIMESTAMPTZ,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recommendations_resident       ON recommendations (resident_id);
CREATE INDEX idx_recommendations_author         ON recommendations (author_id);
CREATE INDEX idx_recommendations_state          ON recommendations (state);
CREATE INDEX idx_recommendations_review_due     ON recommendations (review_due_at)
    WHERE state = 'deferred';
CREATE INDEX idx_recommendations_submitted_at   ON recommendations (submitted_at)
    WHERE submitted_at IS NOT NULL;

-- Materialised view supporting RIR (Recommendation Implementation Rate),
-- the v3 Layer-C operational North Star (v3 §11 line 588).
-- "Documented prescriber action" = state in {decided, implemented,
-- monitoring-active, outcome-recorded, closed} reached within the window.
-- Refreshed by the lifecycle engine (cheap; small table) or on-demand.
CREATE MATERIALIZED VIEW recommendation_rir_28d AS
SELECT
    author_id,
    DATE_TRUNC('day', submitted_at) AS submission_day,
    COUNT(*)                                                AS submitted_count,
    COUNT(*) FILTER (WHERE state IN (
        'decided','implemented','monitoring-active',
        'outcome-recorded','closed'
    ) AND COALESCE(decided_at, closed_at) <= submitted_at + INTERVAL '28 days')
                                                            AS actioned_count
FROM recommendations
WHERE submitted_at IS NOT NULL
GROUP BY author_id, DATE_TRUNC('day', submitted_at);

CREATE UNIQUE INDEX idx_rir_28d_pk
    ON recommendation_rir_28d (author_id, submission_day);

COMMIT;
```

- [ ] **Step 2: Write the rollback**

Create `migrations/023_recommendation_lifecycle_rollback.sql`:

```sql
BEGIN;
DROP MATERIALIZED VIEW IF EXISTS recommendation_rir_28d;
DROP TABLE IF EXISTS recommendations;
COMMIT;
```

- [ ] **Step 3: Apply migration locally and verify**

Run:
```bash
cd /Volumes/Vaidshala/cardiofit
make run-kb-docker  # if not already running; idempotent
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -f migrations/023_recommendation_lifecycle.sql
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -c "\d recommendations"
```
Expected: `\d recommendations` lists the table with all 14 columns and 5 indexes.

- [ ] **Step 4: Verify rollback is safe**

Run:
```bash
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -f migrations/023_recommendation_lifecycle_rollback.sql
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -c "\d recommendations" 2>&1 | grep -q "Did not find any relation" && echo OK
```
Expected: `OK` printed.

- [ ] **Step 5: Re-apply the forward migration (leave DB in committed state)**

Run:
```bash
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -f migrations/023_recommendation_lifecycle.sql
```
Expected: `COMMIT` printed; table re-created.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add migrations/023_recommendation_lifecycle.sql \
        migrations/023_recommendation_lifecycle_rollback.sql
git commit -m "feat(migrations): 023 recommendation lifecycle table + RIR materialised view"
```

---

### Task 4: Store interface + PostgresStore CRUD

**Files:**
- Create: `shared/v2_substrate/recommendation/store.go`
- Create: `shared/v2_substrate/recommendation/store_test.go`

- [ ] **Step 1: Write the failing test**

Create `shared/v2_substrate/recommendation/store_test.go`:

```go
package recommendation

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"shared/v2_substrate/models"
)

// testDB returns a connection to the local Docker Postgres for integration
// tests. Tests that need DB access skip if VAIDSHALA_TEST_DSN is unset, so
// `go test ./...` still passes in CI environments without a database.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN unset; skipping DB integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

func TestPostgresStore_CreateAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	rec := models.Recommendation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AuthorID:   uuid.New(),
		State:      models.RecommendationStateDrafted,
		Type:       models.RecommendationTypeStop,
		Urgency:    models.RecommendationUrgencyAmber,
		Title:      "Cease oxybutynin",
		ClinicalContent: models.ClinicalContent{
			Issue: "test", Rationale: "test", ProposedPlan: "test",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, &rec); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := store.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != rec.ID || got.State != rec.State {
		t.Errorf("got %+v want %+v", got, rec)
	}
	if got.ClinicalContent.Issue != "test" {
		t.Errorf("clinical content not persisted: %+v", got.ClinicalContent)
	}

	// cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM recommendations WHERE id = $1", rec.ID)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
export VAIDSHALA_TEST_DSN="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules?sslmode=disable"
go test ./shared/v2_substrate/recommendation/ -run TestPostgresStore_CreateAndGet -v
```
Expected: FAIL with `undefined: NewPostgresStore`.

- [ ] **Step 3: Implement Store interface and PostgresStore**

Create `shared/v2_substrate/recommendation/store.go`:

```go
package recommendation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"shared/v2_substrate/models"
)

// ErrNotFound is returned when the requested recommendation does not exist.
var ErrNotFound = errors.New("recommendation not found")

// Store is the persistence boundary for recommendations. The Lifecycle engine
// is the only legitimate caller of UpdateState; all other mutations go via
// Create or attribute-specific update methods. This is the contract that
// keeps the transition matrix authoritative.
type Store interface {
	Create(ctx context.Context, rec *models.Recommendation) error
	Get(ctx context.Context, id uuid.UUID) (*models.Recommendation, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string,
		reviewDueAt *interface{}) error // reviewDueAt is *time.Time, kept generic for swap
	ListDeferredOverdue(ctx context.Context, before TimeProvider) ([]models.Recommendation, error)
}

// TimeProvider abstracts wall clock so tests can inject deterministic times.
type TimeProvider interface{ Now() (any, error) }

// NewPostgresStore returns a Store backed by db.
func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

type PostgresStore struct{ db *sql.DB }

func (s *PostgresStore) Create(ctx context.Context, rec *models.Recommendation) error {
	if !models.IsValidRecommendationState(rec.State) {
		return fmt.Errorf("invalid initial state: %q", rec.State)
	}
	cc, err := json.Marshal(rec.ClinicalContent)
	if err != nil {
		return fmt.Errorf("marshal clinical_content: %w", err)
	}
	const q = `
INSERT INTO recommendations
  (id, resident_id, author_id, state, type, urgency, title,
   clinical_content, medicine_use_refs, consent_required,
   review_due_at, submitted_at, decided_at, closed_at,
   created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`
	_, err = s.db.ExecContext(ctx, q,
		rec.ID, rec.ResidentID, rec.AuthorID,
		rec.State, rec.Type, rec.Urgency, rec.Title,
		cc, pq.Array(rec.MedicineUseRefs), rec.ConsentRequired,
		rec.ReviewDueAt, rec.SubmittedAt, rec.DecidedAt, rec.ClosedAt,
		rec.CreatedAt, rec.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*models.Recommendation, error) {
	const q = `
SELECT id, resident_id, author_id, state, type, urgency, title,
       clinical_content, medicine_use_refs, consent_required,
       review_due_at, submitted_at, decided_at, closed_at,
       created_at, updated_at
FROM recommendations WHERE id = $1`
	row := s.db.QueryRowContext(ctx, q, id)

	var rec models.Recommendation
	var ccRaw []byte
	var medRefs pq.StringArray
	err := row.Scan(
		&rec.ID, &rec.ResidentID, &rec.AuthorID,
		&rec.State, &rec.Type, &rec.Urgency, &rec.Title,
		&ccRaw, &medRefs, &rec.ConsentRequired,
		&rec.ReviewDueAt, &rec.SubmittedAt, &rec.DecidedAt, &rec.ClosedAt,
		&rec.CreatedAt, &rec.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(ccRaw, &rec.ClinicalContent); err != nil {
		return nil, fmt.Errorf("unmarshal clinical_content: %w", err)
	}
	rec.MedicineUseRefs = make([]uuid.UUID, 0, len(medRefs))
	for _, s := range medRefs {
		u, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("parse medicine_use_ref %q: %w", s, err)
		}
		rec.MedicineUseRefs = append(rec.MedicineUseRefs, u)
	}
	return &rec, nil
}

// UpdateState is implemented in lifecycle.go to keep the transition guard
// adjacent to the Lifecycle engine that owns it. Stub here to satisfy
// interface; actual impl in Task 5.
```

The full `UpdateState` and `ListDeferredOverdue` implementations are added in Tasks 5 and 7 — leave them as TODO comments **only in this task**, and return `errors.New("not implemented; see lifecycle.go")` so a build still succeeds:

```go
func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID,
	newState string, reviewDueAt *interface{}) error {
	return errors.New("UpdateState: implemented in lifecycle.go (Task 5)")
}

func (s *PostgresStore) ListDeferredOverdue(ctx context.Context,
	before TimeProvider) ([]models.Recommendation, error) {
	return nil, errors.New("ListDeferredOverdue: implemented in Task 7")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:
```bash
go test ./shared/v2_substrate/recommendation/ -run TestPostgresStore_CreateAndGet -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/store.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/store_test.go
git commit -m "feat(substrate): Recommendation Store interface + PostgresStore Create/Get"
```

---

### Task 5: Lifecycle engine with EvidenceTrace emission + transition guard

**Files:**
- Create: `shared/v2_substrate/recommendation/lifecycle.go`
- Create: `shared/v2_substrate/recommendation/lifecycle_test.go`
- Modify: `shared/v2_substrate/recommendation/store.go` (replace `UpdateState` stub with real impl)

- [ ] **Step 1: Write the failing test for Lifecycle.Transition**

Create `shared/v2_substrate/recommendation/lifecycle_test.go`:

```go
package recommendation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

// fakeStore implements Store for unit tests, with no DB.
type fakeStore struct {
	rec  *models.Recommendation
	last struct {
		newState   string
		reviewDue  *time.Time
		updateErr  error
	}
}

func (f *fakeStore) Create(_ context.Context, r *models.Recommendation) error {
	f.rec = r
	return nil
}
func (f *fakeStore) Get(_ context.Context, _ uuid.UUID) (*models.Recommendation, error) {
	return f.rec, nil
}
func (f *fakeStore) UpdateState(_ context.Context, _ uuid.UUID, newState string,
	reviewDue *time.Time) error {
	f.last.newState = newState
	f.last.reviewDue = reviewDue
	if f.rec != nil {
		f.rec.State = newState
		f.rec.ReviewDueAt = reviewDue
	}
	return f.last.updateErr
}
func (f *fakeStore) ListDeferredOverdue(_ context.Context, _ time.Time) (
	[]models.Recommendation, error) {
	return nil, nil
}

// fakeEdgeStore captures EmitEdge calls for assertion.
type fakeEdgeStore struct {
	emitted []EvidenceEdge
}

func (f *fakeEdgeStore) EmitEdge(_ context.Context, e EvidenceEdge) error {
	f.emitted = append(f.emitted, e)
	return nil
}

// alwaysPassConsent satisfies ConsentChecker for tests not exercising consent.
type alwaysPassConsent struct{}

func (alwaysPassConsent) ConsentActive(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return true, nil
}

func TestLifecycle_TransitionHappyPath(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDrafted,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, alwaysPassConsent{})

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		ReasoningSummary: "pharmacist completed draft",
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if store.last.newState != models.RecommendationStateSubmitted {
		t.Errorf("state = %q want submitted", store.last.newState)
	}
	if len(edges.emitted) != 1 {
		t.Fatalf("expected 1 EvidenceTrace edge; got %d", len(edges.emitted))
	}
	got := edges.emitted[0]
	if got.FromState != models.RecommendationStateDrafted ||
		got.ToState != models.RecommendationStateSubmitted {
		t.Errorf("edge states wrong: %+v", got)
	}
	if got.ActorClass != ActorClassHuman {
		t.Errorf("actor class wrong: %v", got.ActorClass)
	}
}

func TestLifecycle_TransitionForbidden(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateDrafted,
	}}
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges, alwaysPassConsent{})

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateImplemented, // skips submitted/viewed/decided
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
	})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition; got %v", err)
	}
	if len(edges.emitted) != 0 {
		t.Errorf("expected no edges emitted on rejected transition; got %d",
			len(edges.emitted))
	}
}

func TestLifecycle_DeferredRequiresReviewDue(t *testing.T) {
	store := &fakeStore{rec: &models.Recommendation{
		ID:    uuid.New(),
		State: models.RecommendationStateSubmitted,
	}}
	lc := NewLifecycle(store, &fakeEdgeStore{}, alwaysPassConsent{})

	err := lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateDeferred,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		// ReviewDueAt deliberately omitted
	})
	if !errors.Is(err, ErrReviewDueRequired) {
		t.Fatalf("expected ErrReviewDueRequired; got %v", err)
	}

	due := time.Now().Add(72 * time.Hour)
	err = lc.Transition(context.Background(), TransitionRequest{
		RecommendationID: store.rec.ID,
		ToState:          models.RecommendationStateDeferred,
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		ReviewDueAt:      &due,
	})
	if err != nil {
		t.Fatalf("expected success with ReviewDueAt; got %v", err)
	}
	if store.last.reviewDue == nil || !store.last.reviewDue.Equal(due) {
		t.Errorf("reviewDue not propagated: %v", store.last.reviewDue)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./shared/v2_substrate/recommendation/ -run TestLifecycle -v`
Expected: FAIL with `undefined: NewLifecycle`, `undefined: TransitionRequest`, etc.

- [ ] **Step 3: Implement Lifecycle and ConsentChecker**

Create `shared/v2_substrate/recommendation/lifecycle.go`:

```go
package recommendation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

// Sentinel errors. The HTTP layer maps these to status codes; the lifecycle
// engine never returns raw SQL errors for guard violations.
var (
	ErrInvalidTransition = errors.New("invalid recommendation transition")
	ErrReviewDueRequired = errors.New("review_due_at required for deferred state")
	ErrConsentRequired   = errors.New("recommendation requires active consent before submission")
)

// ActorClass distinguishes algorithmic from human actors in the
// EvidenceTrace, satisfying v3 §9 Principle 4 (algorithmic vs human
// distinguishable in audit trail).
type ActorClass string

const (
	ActorClassHuman                       ActorClass = "human"
	ActorClassAlgorithmic                 ActorClass = "algorithmic"
	ActorClassHumanWithAlgorithmic        ActorClass = "human-with-algorithmic-suggestion"
	ActorClassHumanOverridingAlgorithmic  ActorClass = "human-overriding-algorithmic"
)

// EvidenceEdge is the substrate-local record of one lifecycle transition.
// EdgeStore.EmitEdge persists this into the EvidenceTrace graph.
type EvidenceEdge struct {
	RecommendationID uuid.UUID
	FromState        string
	ToState          string
	ActorID          uuid.UUID
	ActorClass       ActorClass
	OccurredAt       time.Time
	ReasoningSummary string
	InputRefs        []uuid.UUID // observation/medicine_use/event refs that justified the transition
}

// EdgeStore is the EvidenceTrace persistence boundary. The real
// implementation wraps shared/v2_substrate/evidence_trace/edge_store.go;
// tests use a fake.
type EdgeStore interface {
	EmitEdge(ctx context.Context, e EvidenceEdge) error
}

// ConsentChecker is the substrate gate ensuring restrictive-practice and
// other consent-required recommendation classes have an active matching
// Consent before they advance from drafted → submitted.
//
// Plan 0.2 ships a real Postgres-backed checker; this plan tests with
// AlwaysPassConsentChecker.
type ConsentChecker interface {
	// ConsentActive reports whether an active matching Consent exists for
	// resident, scoped to the recommendation type / class.
	ConsentActive(ctx context.Context, residentID uuid.UUID, recType string) (bool, error)
}

// TransitionRequest is the input contract for Lifecycle.Transition.
type TransitionRequest struct {
	RecommendationID uuid.UUID
	ToState          string
	ActorID          uuid.UUID
	ActorClass       ActorClass
	OccurredAt       time.Time   // optional; defaults to time.Now() UTC
	ReasoningSummary string
	InputRefs        []uuid.UUID
	ReviewDueAt      *time.Time  // required when ToState == deferred
}

// Lifecycle is the only sanctioned mutator of recommendation state.
type Lifecycle struct {
	store   Store
	edges   EdgeStore
	consent ConsentChecker
	now     func() time.Time
}

// NewLifecycle constructs a Lifecycle wired to the supplied collaborators.
func NewLifecycle(store Store, edges EdgeStore, consent ConsentChecker) *Lifecycle {
	return &Lifecycle{
		store:   store,
		edges:   edges,
		consent: consent,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

// Transition advances a recommendation through one DAG edge. Returns one of
// the sentinel errors on guard violation; otherwise returns the underlying
// store/edge error verbatim.
func (l *Lifecycle) Transition(ctx context.Context, req TransitionRequest) error {
	rec, err := l.store.Get(ctx, req.RecommendationID)
	if err != nil {
		return fmt.Errorf("load recommendation: %w", err)
	}

	if !models.IsValidTransition(rec.State, req.ToState) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, rec.State, req.ToState)
	}

	// deferred requires review_due_at
	if req.ToState == models.RecommendationStateDeferred && req.ReviewDueAt == nil {
		return ErrReviewDueRequired
	}

	// drafted -> submitted requires active consent if the recommendation
	// class declared ConsentRequired at draft time.
	if rec.State == models.RecommendationStateDrafted &&
		req.ToState == models.RecommendationStateSubmitted &&
		rec.ConsentRequired {
		ok, err := l.consent.ConsentActive(ctx, rec.ResidentID, rec.Type)
		if err != nil {
			return fmt.Errorf("consent check: %w", err)
		}
		if !ok {
			return ErrConsentRequired
		}
	}

	occurred := req.OccurredAt
	if occurred.IsZero() {
		occurred = l.now()
	}

	if err := l.store.UpdateState(ctx, rec.ID, req.ToState, req.ReviewDueAt); err != nil {
		return fmt.Errorf("update state: %w", err)
	}

	edge := EvidenceEdge{
		RecommendationID: rec.ID,
		FromState:        rec.State,
		ToState:          req.ToState,
		ActorID:          req.ActorID,
		ActorClass:       req.ActorClass,
		OccurredAt:       occurred,
		ReasoningSummary: req.ReasoningSummary,
		InputRefs:        req.InputRefs,
	}
	if err := l.edges.EmitEdge(ctx, edge); err != nil {
		// State change already committed; emit failure is operationally
		// loud but not a state rollback condition. Surface to caller.
		return fmt.Errorf("emit evidence edge: %w", err)
	}
	return nil
}

// AlwaysPassConsentChecker is a test/dev double. Wire the real Consent
// implementation from plan 0.2 in production deployments.
type AlwaysPassConsentChecker struct{}

func (AlwaysPassConsentChecker) ConsentActive(_ context.Context,
	_ uuid.UUID, _ string) (bool, error) {
	return true, nil
}
```

- [ ] **Step 4: Update Store interface signatures to match**

Replace the stub `UpdateState` and the `TimeProvider` machinery in `shared/v2_substrate/recommendation/store.go`. Locate the lines:

```go
type Store interface {
	Create(ctx context.Context, rec *models.Recommendation) error
	Get(ctx context.Context, id uuid.UUID) (*models.Recommendation, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string,
		reviewDueAt *interface{}) error // reviewDueAt is *time.Time, kept generic for swap
	ListDeferredOverdue(ctx context.Context, before TimeProvider) ([]models.Recommendation, error)
}

// TimeProvider abstracts wall clock so tests can inject deterministic times.
type TimeProvider interface{ Now() (any, error) }
```

Replace the entire block above with:

```go
type Store interface {
	Create(ctx context.Context, rec *models.Recommendation) error
	Get(ctx context.Context, id uuid.UUID) (*models.Recommendation, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string,
		reviewDueAt *time.Time) error
	ListDeferredOverdue(ctx context.Context, before time.Time) ([]models.Recommendation, error)
}
```

Then locate the stub:

```go
func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID,
	newState string, reviewDueAt *interface{}) error {
	return errors.New("UpdateState: implemented in lifecycle.go (Task 5)")
}
```

Replace with:

```go
func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID,
	newState string, reviewDueAt *time.Time) error {
	if !models.IsValidRecommendationState(newState) {
		return fmt.Errorf("invalid state: %q", newState)
	}
	const q = `
UPDATE recommendations
SET state = $1,
    review_due_at = $2,
    submitted_at = CASE WHEN $1 = 'submitted' AND submitted_at IS NULL
                        THEN NOW() ELSE submitted_at END,
    decided_at   = CASE WHEN $1 = 'decided'   AND decided_at   IS NULL
                        THEN NOW() ELSE decided_at   END,
    closed_at    = CASE WHEN $1 = 'closed'    AND closed_at    IS NULL
                        THEN NOW() ELSE closed_at    END,
    updated_at   = NOW()
WHERE id = $3`
	res, err := s.db.ExecContext(ctx, q, newState, reviewDueAt, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
```

Add the import for `time`:

```go
import (
	// existing imports
	"time"
)
```

Also locate the `ListDeferredOverdue` stub:

```go
func (s *PostgresStore) ListDeferredOverdue(ctx context.Context,
	before TimeProvider) ([]models.Recommendation, error) {
	return nil, errors.New("ListDeferredOverdue: implemented in Task 7")
}
```

Replace with:

```go
func (s *PostgresStore) ListDeferredOverdue(ctx context.Context,
	before time.Time) ([]models.Recommendation, error) {
	return nil, errors.New("ListDeferredOverdue: implemented in Task 7")
}
```

- [ ] **Step 5: Run test to verify lifecycle tests pass**

Run: `go test ./shared/v2_substrate/recommendation/ -run TestLifecycle -v`
Expected: PASS for all three Lifecycle tests.

- [ ] **Step 6: Run full package test to verify no regression**

Run:
```bash
export VAIDSHALA_TEST_DSN="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules?sslmode=disable"
go test ./shared/v2_substrate/recommendation/ -v
```
Expected: PASS for all tests in the package.

- [ ] **Step 7: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/
git commit -m "feat(substrate): Recommendation Lifecycle with EvidenceTrace + consent gate"
```

---

### Task 6: Wire real EdgeStore into evidence_trace

**Files:**
- Create: `shared/v2_substrate/recommendation/edge_adapter.go`
- Create: `shared/v2_substrate/recommendation/edge_adapter_test.go`

The lifecycle.go EdgeStore interface is satisfied here by an adapter over the existing evidence_trace package, so the real graph receives recommendation transition edges.

- [ ] **Step 1: Inspect the evidence_trace EmitEdge signature (read-only)**

Run:
```bash
grep -n "func.*EmitEdge\|^type Edge\b\|^type Node\b" \
  shared/v2_substrate/evidence_trace/edge_store.go \
  shared/v2_substrate/evidence_trace/graph.go 2>/dev/null
```
Note the exact signature for use in step 3. (The plan assumes `EmitEdge(ctx context.Context, edge evidence_trace.Edge) error` and an `Edge` type with `FromNodeID`, `ToNodeID`, `EdgeType`, `ActorID`, `OccurredAt`, `Metadata map[string]any`. If the real signature differs, adapt the adapter — the contract from this plan is just "translate `recommendation.EvidenceEdge` into the substrate's existing edge format".)

- [ ] **Step 2: Write the failing test**

Create `shared/v2_substrate/recommendation/edge_adapter_test.go`:

```go
package recommendation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/evidence_trace"
)

// captureEdgeStore wraps the real evidence_trace.EdgeStore for assertion in
// integration tests. We don't depend on a real graph DB here; we use the
// in-memory test double the evidence_trace package already exports.
func TestEdgeAdapter_EmitsToEvidenceTraceGraph(t *testing.T) {
	mem := evidence_trace.NewInMemoryEdgeStore()
	adapter := NewEvidenceTraceAdapter(mem)

	edge := EvidenceEdge{
		RecommendationID: uuid.New(),
		FromState:        "drafted",
		ToState:          "submitted",
		ActorID:          uuid.New(),
		ActorClass:       ActorClassHuman,
		OccurredAt:       time.Now().UTC(),
		ReasoningSummary: "test",
	}
	if err := adapter.EmitEdge(context.Background(), edge); err != nil {
		t.Fatalf("emit: %v", err)
	}
	got := mem.AllEdges()
	if len(got) != 1 {
		t.Fatalf("expected 1 edge in graph; got %d", len(got))
	}
	if got[0].EdgeType != "recommendation_transition" {
		t.Errorf("edge_type = %q want recommendation_transition", got[0].EdgeType)
	}
}
```

If `evidence_trace.NewInMemoryEdgeStore` does not exist, replace with whatever in-memory test double the package exposes (e.g. `evidence_trace.NewMemoryGraph`). Inspect first:

```bash
grep -n "NewInMemory\|NewMemory\|type.*EdgeStore" \
  shared/v2_substrate/evidence_trace/*.go
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./shared/v2_substrate/recommendation/ -run TestEdgeAdapter -v`
Expected: FAIL with `undefined: NewEvidenceTraceAdapter`.

- [ ] **Step 4: Write the adapter**

Create `shared/v2_substrate/recommendation/edge_adapter.go`. Adjust the `evidence_trace.Edge` field names to match the actual signature found in step 1.

```go
package recommendation

import (
	"context"

	"shared/v2_substrate/evidence_trace"
)

// EvidenceTraceAdapter satisfies recommendation.EdgeStore by translating
// recommendation.EvidenceEdge into the substrate's evidence_trace.Edge
// format and writing through the underlying graph store.
type EvidenceTraceAdapter struct {
	graph evidence_trace.EdgeStore
}

func NewEvidenceTraceAdapter(graph evidence_trace.EdgeStore) *EvidenceTraceAdapter {
	return &EvidenceTraceAdapter{graph: graph}
}

func (a *EvidenceTraceAdapter) EmitEdge(ctx context.Context, e EvidenceEdge) error {
	graphEdge := evidence_trace.Edge{
		FromNodeID: e.RecommendationID.String() + ":" + e.FromState,
		ToNodeID:   e.RecommendationID.String() + ":" + e.ToState,
		EdgeType:   "recommendation_transition",
		ActorID:    e.ActorID,
		OccurredAt: e.OccurredAt,
		Metadata: map[string]any{
			"actor_class":       string(e.ActorClass),
			"reasoning_summary": e.ReasoningSummary,
			"input_refs":        e.InputRefs,
		},
	}
	return a.graph.EmitEdge(ctx, graphEdge)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./shared/v2_substrate/recommendation/ -run TestEdgeAdapter -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/edge_adapter.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/edge_adapter_test.go
git commit -m "feat(substrate): Recommendation EvidenceTrace adapter"
```

---

### Task 7: Deferred-state escalator

**Files:**
- Create: `shared/v2_substrate/recommendation/deferred_escalator.go`
- Create: `shared/v2_substrate/recommendation/deferred_escalator_test.go`
- Modify: `shared/v2_substrate/recommendation/store.go` (real `ListDeferredOverdue` impl)

- [ ] **Step 1: Write the failing test**

Create `shared/v2_substrate/recommendation/deferred_escalator_test.go`:

```go
package recommendation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

// listOverdueFakeStore lets the test inject a fixed list to be returned.
type listOverdueFakeStore struct {
	fakeStore
	list []models.Recommendation
}

func (l *listOverdueFakeStore) ListDeferredOverdue(_ context.Context,
	_ time.Time) ([]models.Recommendation, error) {
	return l.list, nil
}

// recordingEvents captures all events the escalator emits.
type recordingEvents struct {
	emitted []EscalationEvent
}

func (r *recordingEvents) Emit(_ context.Context, ev EscalationEvent) error {
	r.emitted = append(r.emitted, ev)
	return nil
}

func TestDeferredEscalator_EmitsEventForOverdue(t *testing.T) {
	overdueID := uuid.New()
	store := &listOverdueFakeStore{
		list: []models.Recommendation{
			{ID: overdueID, State: models.RecommendationStateDeferred,
				ReviewDueAt: timePtr(time.Now().Add(-1 * time.Hour))},
		},
	}
	events := &recordingEvents{}
	esc := NewDeferredEscalator(store, events,
		func() time.Time { return time.Now().UTC() })

	if err := esc.RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(events.emitted) != 1 {
		t.Fatalf("expected 1 escalation event; got %d", len(events.emitted))
	}
	if events.emitted[0].RecommendationID != overdueID {
		t.Errorf("escalation for wrong rec: got %v want %v",
			events.emitted[0].RecommendationID, overdueID)
	}
}

func timePtr(t time.Time) *time.Time { return &t }
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./shared/v2_substrate/recommendation/ -run TestDeferredEscalator -v`
Expected: FAIL with `undefined: NewDeferredEscalator`, `undefined: EscalationEvent`.

- [ ] **Step 3: Implement the escalator**

Create `shared/v2_substrate/recommendation/deferred_escalator.go`:

```go
package recommendation

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EscalationEvent is emitted to the Worklist event bus when a deferred
// recommendation passes its review_due_at without action. The Worklist
// surface (Layer 4) renders these as "needs attention" cards.
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
// Operational note: this worker is idempotent at the event-bus boundary
// (Worklist consumer dedupes on RecommendationID + day). The escalator
// itself does NOT mutate recommendation state; the human action of
// re-surfacing or closing is the source of truth.
type DeferredEscalator struct {
	store  Store
	sink   EscalationSink
	now    func() time.Time
}

func NewDeferredEscalator(store Store, sink EscalationSink, now func() time.Time) *DeferredEscalator {
	return &DeferredEscalator{store: store, sink: sink, now: now}
}

// RunOnce performs a single sweep. Production deployment wires this on
// a 5-minute ticker.
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
```

- [ ] **Step 4: Implement the real ListDeferredOverdue in store.go**

Locate in `shared/v2_substrate/recommendation/store.go`:

```go
func (s *PostgresStore) ListDeferredOverdue(ctx context.Context,
	before time.Time) ([]models.Recommendation, error) {
	return nil, errors.New("ListDeferredOverdue: implemented in Task 7")
}
```

Replace with:

```go
func (s *PostgresStore) ListDeferredOverdue(ctx context.Context,
	before time.Time) ([]models.Recommendation, error) {
	const q = `
SELECT id, resident_id, author_id, state, type, urgency, title,
       clinical_content, medicine_use_refs, consent_required,
       review_due_at, submitted_at, decided_at, closed_at,
       created_at, updated_at
FROM recommendations
WHERE state = 'deferred' AND review_due_at IS NOT NULL AND review_due_at < $1
ORDER BY review_due_at ASC
LIMIT 1000`
	rows, err := s.db.QueryContext(ctx, q, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Recommendation
	for rows.Next() {
		var rec models.Recommendation
		var ccRaw []byte
		var medRefs pq.StringArray
		if err := rows.Scan(
			&rec.ID, &rec.ResidentID, &rec.AuthorID,
			&rec.State, &rec.Type, &rec.Urgency, &rec.Title,
			&ccRaw, &medRefs, &rec.ConsentRequired,
			&rec.ReviewDueAt, &rec.SubmittedAt, &rec.DecidedAt, &rec.ClosedAt,
			&rec.CreatedAt, &rec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(ccRaw, &rec.ClinicalContent); err != nil {
			return nil, fmt.Errorf("unmarshal clinical_content: %w", err)
		}
		rec.MedicineUseRefs = make([]uuid.UUID, 0, len(medRefs))
		for _, s := range medRefs {
			u, err := uuid.Parse(s)
			if err != nil {
				return nil, fmt.Errorf("parse medicine_use_ref %q: %w", s, err)
			}
			rec.MedicineUseRefs = append(rec.MedicineUseRefs, u)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
```

- [ ] **Step 5: Run test to verify pass**

Run: `go test ./shared/v2_substrate/recommendation/ -run TestDeferredEscalator -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/deferred_escalator.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/deferred_escalator_test.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/store.go
git commit -m "feat(substrate): Recommendation deferred-state escalator"
```

---

### Task 8: RIR computation query

**Files:**
- Create: `shared/v2_substrate/recommendation/rir.go`
- Create: `shared/v2_substrate/recommendation/rir_test.go`

- [ ] **Step 1: Write the failing test**

Create `shared/v2_substrate/recommendation/rir_test.go`:

```go
package recommendation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

func TestComputeRIR_SubmittedAndActioned(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()
	author := uuid.New()

	// Seed 5 recommendations: 4 submitted, 3 of which actioned within window.
	now := time.Now().UTC()
	mk := func(state string, submittedDaysAgo, decidedDaysAgo int) {
		rec := models.Recommendation{
			ID:         uuid.New(),
			ResidentID: uuid.New(),
			AuthorID:   author,
			State:      state,
			Type:       models.RecommendationTypeStop,
			Urgency:    models.RecommendationUrgencyAmber,
			Title:      "test",
			ClinicalContent: models.ClinicalContent{Issue: "x"},
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if submittedDaysAgo >= 0 {
			t := now.Add(-time.Duration(submittedDaysAgo) * 24 * time.Hour)
			rec.SubmittedAt = &t
		}
		if decidedDaysAgo >= 0 {
			t := now.Add(-time.Duration(decidedDaysAgo) * 24 * time.Hour)
			rec.DecidedAt = &t
		}
		if err := store.Create(ctx, &rec); err != nil {
			t.Fatalf("seed: %v", err)
		}
		t_ := t // avoid loop-variable capture
		_ = t_
	}
	mk(models.RecommendationStateImplemented, 10, 5) // actioned (5d after submit)
	mk(models.RecommendationStateDecided, 8, 3)      // actioned
	mk(models.RecommendationStateClosed, 12, 7)      // actioned
	mk(models.RecommendationStateSubmitted, 7, -1)   // submitted, not actioned
	mk(models.RecommendationStateDrafted, -1, -1)    // never submitted

	defer func() {
		_, _ = db.ExecContext(ctx,
			"DELETE FROM recommendations WHERE author_id = $1", author)
	}()

	got, err := ComputeRIR(ctx, db, author, 28*24*time.Hour)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if got.Submitted != 4 {
		t.Errorf("submitted = %d want 4", got.Submitted)
	}
	if got.Actioned != 3 {
		t.Errorf("actioned = %d want 3", got.Actioned)
	}
	if got.RatePercent < 74.9 || got.RatePercent > 75.1 {
		t.Errorf("rate = %.2f want ~75.0", got.RatePercent)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
export VAIDSHALA_TEST_DSN="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules?sslmode=disable"
go test ./shared/v2_substrate/recommendation/ -run TestComputeRIR -v
```
Expected: FAIL with `undefined: ComputeRIR`.

- [ ] **Step 3: Implement ComputeRIR**

Create `shared/v2_substrate/recommendation/rir.go`:

```go
package recommendation

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// RIRResult is the v3 §11 line 588 Layer-C operational North Star metric.
// "Documented prescriber action" = recommendation reached one of the
// terminal-or-progressed states {decided, implemented, monitoring-active,
// outcome-recorded, closed} within the configured window after submission.
type RIRResult struct {
	AuthorID    uuid.UUID
	Window      time.Duration
	Submitted   int
	Actioned    int
	RatePercent float64
}

// ComputeRIR returns the rolling-window RIR for one author. Window is
// typically 28 days (default) per Ramsey 2025 measurement basis.
func ComputeRIR(ctx context.Context, db *sql.DB, authorID uuid.UUID,
	window time.Duration) (RIRResult, error) {
	const q = `
WITH eligible AS (
  SELECT id, state, submitted_at, decided_at, closed_at
  FROM recommendations
  WHERE author_id = $1
    AND submitted_at IS NOT NULL
    AND submitted_at >= NOW() - $2::interval
)
SELECT
  COUNT(*),
  COUNT(*) FILTER (WHERE state IN (
    'decided','implemented','monitoring-active','outcome-recorded','closed'
  ) AND COALESCE(decided_at, closed_at) <= submitted_at + $3::interval)
FROM eligible`
	// Postgres requires a string-formatted interval; encode the same window twice.
	wind := durationToInterval(window)
	row := db.QueryRowContext(ctx, q, authorID, wind, wind)

	var submitted, actioned int
	if err := row.Scan(&submitted, &actioned); err != nil {
		return RIRResult{}, err
	}
	rate := 0.0
	if submitted > 0 {
		rate = 100.0 * float64(actioned) / float64(submitted)
	}
	return RIRResult{
		AuthorID:    authorID,
		Window:      window,
		Submitted:   submitted,
		Actioned:    actioned,
		RatePercent: rate,
	}, nil
}

// durationToInterval renders a Go time.Duration as a Postgres interval string,
// e.g. 28*24h → "28 days". Resolution is days only; sub-day windows are
// rendered in hours.
func durationToInterval(d time.Duration) string {
	if d >= 24*time.Hour && d%(24*time.Hour) == 0 {
		days := int(d / (24 * time.Hour))
		return formatDays(days)
	}
	hours := int(d / time.Hour)
	return formatHours(hours)
}

func formatDays(n int) string  { return itoa(n) + " days" }
func formatHours(n int) string { return itoa(n) + " hours" }
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./shared/v2_substrate/recommendation/ -run TestComputeRIR -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/rir.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/rir_test.go
git commit -m "feat(substrate): Recommendation Implementation Rate (RIR) computation"
```

---

### Task 9: End-to-end happy-path integration test

**Files:**
- Create: `shared/v2_substrate/recommendation/integration_test.go`

This test exercises the full chain: Create → Transition (drafted→submitted→viewed→decided→implemented→closed) with EvidenceTrace edges emitted, then ComputeRIR returns the expected rate. It is the executable definition of "Recommendation entity + lifecycle is shipped."

- [ ] **Step 1: Write the integration test**

Create `shared/v2_substrate/recommendation/integration_test.go`:

```go
package recommendation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/evidence_trace"
	"shared/v2_substrate/models"
)

func TestIntegration_FullLifecycleEndToEnd(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	store := NewPostgresStore(db)
	graph := evidence_trace.NewInMemoryEdgeStore()
	edges := NewEvidenceTraceAdapter(graph)
	lc := NewLifecycle(store, edges, AlwaysPassConsentChecker{})

	ctx := context.Background()
	author := uuid.New()
	rec := models.Recommendation{
		ID:         uuid.New(),
		ResidentID: uuid.New(),
		AuthorID:   author,
		State:      models.RecommendationStateDrafted,
		Type:       models.RecommendationTypeStop,
		Urgency:    models.RecommendationUrgencyAmber,
		Title:      "Cease oxybutynin",
		ClinicalContent: models.ClinicalContent{
			Issue: "ACB", Rationale: "DBI 0.8", ProposedPlan: "cease",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, &rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx,
			"DELETE FROM recommendations WHERE id = $1", rec.ID)
	}()

	steps := []string{
		models.RecommendationStateSubmitted,
		models.RecommendationStateViewed,
		models.RecommendationStateDecided,
		models.RecommendationStateImplemented,
		models.RecommendationStateOutcomeRecorded,
		models.RecommendationStateClosed,
	}
	for _, s := range steps {
		err := lc.Transition(ctx, TransitionRequest{
			RecommendationID: rec.ID,
			ToState:          s,
			ActorID:          uuid.New(),
			ActorClass:       ActorClassHuman,
			ReasoningSummary: "test step " + s,
		})
		if err != nil {
			t.Fatalf("transition to %s: %v", s, err)
		}
	}

	got, err := store.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != models.RecommendationStateClosed {
		t.Errorf("final state = %q want closed", got.State)
	}
	if got.SubmittedAt == nil || got.DecidedAt == nil || got.ClosedAt == nil {
		t.Errorf("timestamp columns not populated: %+v", got)
	}
	if len(graph.AllEdges()) != len(steps) {
		t.Errorf("edges in graph = %d want %d", len(graph.AllEdges()), len(steps))
	}

	// RIR check: 1 submitted, 1 actioned, 100%
	rir, err := ComputeRIR(ctx, db, author, 28*24*time.Hour)
	if err != nil {
		t.Fatalf("rir: %v", err)
	}
	if rir.Submitted != 1 || rir.Actioned != 1 || rir.RatePercent < 99.9 {
		t.Errorf("RIR result wrong: %+v", rir)
	}
}
```

- [ ] **Step 2: Run the integration test**

Run:
```bash
export VAIDSHALA_TEST_DSN="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules?sslmode=disable"
go test ./shared/v2_substrate/recommendation/ -run TestIntegration_FullLifecycleEndToEnd -v
```
Expected: PASS.

- [ ] **Step 3: Run the entire substrate test suite to verify no regression**

Run:
```bash
go test ./shared/v2_substrate/... -v 2>&1 | tail -40
```
Expected: PASS for all packages — `models/`, `recommendation/`, `evidence_trace/`, `clinical_state/`, `delta/`, `permissions/` (if present).

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/recommendation/integration_test.go
git commit -m "test(substrate): Recommendation lifecycle end-to-end integration test"
```

---

## Plan complete

Verify each item against the spec one more time:

- [x] **9-state lifecycle** (v2 §3 line 134) — Tasks 1, 2 (constants + matrix); Task 5 (engine)
- [x] **Deferred state with forced review_due_at** (Ramsey 50% non-implementation cure) — Task 5 (`ErrReviewDueRequired`); Task 7 (escalator)
- [x] **EvidenceTrace integration** (v2 §3 line 144) — Tasks 5, 6 (adapter)
- [x] **Algorithmic-vs-human distinction** (v3 §9 Principle 4) — Task 5 (`ActorClass` enum)
- [x] **Consent gating pre-condition** (v2 §3 line 140) — Task 5 (`ConsentChecker`, `ErrConsentRequired`)
- [x] **RIR computation** (v3 §11 line 588) — Tasks 3 (matview), 8 (query)
- [x] **Recommendation type ordering hints** (v3 §7 line 384) — Task 1 (`RecommendationType*` constants; ordering enforced by craft engine in plan 0.5/Phase 2)
- [x] **Urgency tiers** (v3 §7 line 396) — Task 1 (`RecommendationUrgency*` constants)
- [x] **Frame-vs-content separation foundation** (v3 §7 line 416) — Task 1 (`ClinicalContent` is a separate JSONB field; framing layer added in Phase 2)

---

## Plan complete and saved to `docs/superpowers/plans/2026-05-07-phase-0-1-recommendation-entity-lifecycle.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
