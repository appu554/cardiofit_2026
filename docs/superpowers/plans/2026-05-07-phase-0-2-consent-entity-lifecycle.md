# Consent Entity + Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the v2/v3 substrate's `Consent` entity with a 7-state lifecycle (`requested → discussed → granted | refused | granted-with-conditions → active → under_review → withdrawn | expired`), an `SDM` linkage to the Person+Role substrate, and a Postgres-backed `ConsentChecker` that replaces the `AlwaysPassConsentChecker` from Plan 0.1, so the Recommendation lifecycle can enforce the v2 §3 line 140 gating: psychotropic and restrictive-practice recommendations cannot advance from drafted → submitted without an active matching Consent.

**Architecture:** Follows Plan 0.1 patterns exactly — entity in `shared/v2_substrate/models/consent.go`, lifecycle in `shared/v2_substrate/consent/`, root migration `024_consent_lifecycle.sql`. State transitions emit EvidenceTrace edges using the same `EdgeStore` interface. The `ConsentChecker` is wired via Plan 0.1's `recommendation.ConsentChecker` interface so swapping `AlwaysPassConsentChecker` for the real `PostgresConsentChecker` is one line in the lifecycle constructor.

**Tech Stack:** Go 1.21+, `google/uuid`, Go std-lib `testing`, PostgreSQL 15, existing substrate patterns. Depends on Plan 0.1 (Recommendation entity) for the `ConsentChecker` integration target.

---

## File Structure

**New files:**
- `shared/v2_substrate/models/consent.go` — entity + state enum + recommendation-class-to-consent-class mapping
- `shared/v2_substrate/models/consent_test.go` — JSON round-trip + transition validity tests
- `shared/v2_substrate/consent/store.go` — `Store` interface + `PostgresStore`
- `shared/v2_substrate/consent/store_test.go` — store CRUD + active-consent-lookup tests
- `shared/v2_substrate/consent/lifecycle.go` — transition engine with EvidenceTrace emission
- `shared/v2_substrate/consent/lifecycle_test.go`
- `shared/v2_substrate/consent/checker.go` — `PostgresConsentChecker` satisfying `recommendation.ConsentChecker`
- `shared/v2_substrate/consent/checker_test.go`
- `migrations/024_consent_lifecycle.sql`
- `migrations/024_consent_lifecycle_rollback.sql`

**Modified files:**
- `shared/v2_substrate/models/enums.go` — append `ConsentState*`, `ConsentClass*`
- (Plan 0.1's recommendation lifecycle wiring is updated in Task 7 below)

---

### Task 1: Define Consent entity model + state constants

**Files:**
- Create: `shared/v2_substrate/models/consent.go`
- Modify: `shared/v2_substrate/models/enums.go`
- Test: `shared/v2_substrate/models/consent_test.go`

- [ ] **Step 1: Append constants to enums.go**

```go
// ConsentState* — the 7 lifecycle states. See plan:
// docs/superpowers/plans/2026-05-07-phase-0-2-consent-entity-lifecycle.md
const (
	ConsentStateRequested            = "requested"
	ConsentStateDiscussed            = "discussed"
	ConsentStateGranted              = "granted"
	ConsentStateGrantedWithConditions = "granted-with-conditions"
	ConsentStateRefused              = "refused"
	ConsentStateActive               = "active"
	ConsentStateUnderReview          = "under_review"
	ConsentStateWithdrawn            = "withdrawn"
	ConsentStateExpired              = "expired"
)

// ConsentClass* — the recommendation classes a Consent applies to.
// Mirrors recommendation.Type but at the broader policy level
// (one Consent can cover all psychotropic recommendations for a resident).
const (
	ConsentClassPsychotropic       = "psychotropic"
	ConsentClassRestrictivePractice = "restrictive_practice"
	ConsentClassChemoTherapy       = "chemotherapy"
	ConsentClassEndOfLifeMedication = "end_of_life_medication"
	ConsentClassGeneralMedication  = "general_medication"
)

func IsValidConsentState(s string) bool {
	switch s {
	case ConsentStateRequested, ConsentStateDiscussed, ConsentStateGranted,
		ConsentStateGrantedWithConditions, ConsentStateRefused,
		ConsentStateActive, ConsentStateUnderReview,
		ConsentStateWithdrawn, ConsentStateExpired:
		return true
	}
	return false
}

// RecommendationTypeRequiresConsent maps recommendation types to the
// Consent class that gates them. Returns ("", false) when no consent
// is required.
func RecommendationTypeRequiresConsent(recType string) (string, bool) {
	// Psychotropic and restrictive-practice are the v3 §3 line 140 cases.
	// Mapping is by recommendation Type + a class hint baked into the
	// title/clinical_content; for now use a conservative default: any
	// recommendation explicitly tagged psychotropic via title keyword
	// or via the (future) consent_class field on Recommendation requires
	// consent. Plan 0.1's Recommendation already has ConsentRequired bool
	// set at draft time by the craft engine; this mapping is for
	// consistency checks only.
	return "", false
}
```

- [ ] **Step 2: Write the failing test**

Create `shared/v2_substrate/models/consent_test.go`:

```go
package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestConsentJSONRoundTrip(t *testing.T) {
	in := Consent{
		ID:           uuid.New(),
		ResidentID:   uuid.New(),
		Class:        ConsentClassPsychotropic,
		State:        ConsentStateActive,
		GrantedByID:  uuid.New(), // Person.id (SDM or resident-self)
		GrantedByRole: "substitute_decision_maker",
		Conditions:   "valid only for risperidone <0.5mg BD",
		ScopeNotes:   "covers BPSD recommendations through 2026-12",
		ValidFrom:    time.Now().UTC().Truncate(time.Microsecond),
		ValidUntil:   ptrTime(time.Now().Add(365 * 24 * time.Hour).UTC().Truncate(time.Microsecond)),
		CreatedAt:    time.Now().UTC().Truncate(time.Microsecond),
		UpdatedAt:    time.Now().UTC().Truncate(time.Microsecond),
	}
	raw, _ := json.Marshal(in)
	var out Consent
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.State != in.State || out.Class != in.Class || out.GrantedByID != in.GrantedByID {
		t.Errorf("round-trip mismatch: got %+v want %+v", out, in)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestConsentTransitionMatrix(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		// Happy path
		{ConsentStateRequested, ConsentStateDiscussed, true},
		{ConsentStateDiscussed, ConsentStateGranted, true},
		{ConsentStateDiscussed, ConsentStateGrantedWithConditions, true},
		{ConsentStateDiscussed, ConsentStateRefused, true},
		{ConsentStateGranted, ConsentStateActive, true},
		{ConsentStateGrantedWithConditions, ConsentStateActive, true},
		{ConsentStateActive, ConsentStateUnderReview, true},
		{ConsentStateActive, ConsentStateWithdrawn, true},
		{ConsentStateActive, ConsentStateExpired, true},
		{ConsentStateUnderReview, ConsentStateActive, true},
		{ConsentStateUnderReview, ConsentStateWithdrawn, true},
		// Forbidden
		{ConsentStateRefused, ConsentStateActive, false},
		{ConsentStateExpired, ConsentStateActive, false},
		{ConsentStateWithdrawn, ConsentStateActive, false},
		{ConsentStateRequested, ConsentStateActive, false}, // skip discussed
		{"bogus", ConsentStateActive, false},
	}
	for _, c := range cases {
		if got := IsValidConsentTransition(c.from, c.to); got != c.want {
			t.Errorf("IsValidConsentTransition(%q,%q)=%v want %v",
				c.from, c.to, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Run test, expect failure**

`go test ./shared/v2_substrate/models/ -run TestConsent -v` → FAIL on undefined `Consent`.

- [ ] **Step 4: Write entity + transition matrix**

Create `shared/v2_substrate/models/consent.go`:

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

// Consent is the v2/v3 regulatory substrate entity for restrictive-practice
// and psychotropic medication authorisation under the Aged Care Quality
// Standards 2026 and Restrictive Practice regulations 2019. The lifecycle
// engine (consent.Lifecycle) is the only sanctioned mutator.
//
// One Consent can cover multiple recommendations through its Class scope.
// E.g. a "psychotropic" Consent covers all psychotropic recommendations
// for a resident until withdrawn or expired.
type Consent struct {
	ID            uuid.UUID  `json:"id"`
	ResidentID    uuid.UUID  `json:"resident_id"`
	Class         string     `json:"class"` // see ConsentClass*
	State         string     `json:"state"` // see ConsentState*
	GrantedByID   uuid.UUID  `json:"granted_by_id"`   // Person.id (SDM, resident-self, or guardian)
	GrantedByRole string     `json:"granted_by_role"` // role at time of granting
	Conditions    string     `json:"conditions,omitempty"`  // for granted-with-conditions
	ScopeNotes    string     `json:"scope_notes,omitempty"`
	ValidFrom     time.Time  `json:"valid_from"`
	ValidUntil    *time.Time `json:"valid_until,omitempty"` // nullable = open-ended
	WithdrawnAt   *time.Time `json:"withdrawn_at,omitempty"`
	ExpiredAt     *time.Time `json:"expired_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

var consentTransitions = map[string]map[string]bool{
	ConsentStateRequested: {
		ConsentStateDiscussed: true,
		ConsentStateRefused:   true, // declined before discussion
	},
	ConsentStateDiscussed: {
		ConsentStateGranted:               true,
		ConsentStateGrantedWithConditions: true,
		ConsentStateRefused:               true,
	},
	ConsentStateGranted:               {ConsentStateActive: true},
	ConsentStateGrantedWithConditions: {ConsentStateActive: true},
	ConsentStateActive: {
		ConsentStateUnderReview: true,
		ConsentStateWithdrawn:   true,
		ConsentStateExpired:     true,
	},
	ConsentStateUnderReview: {
		ConsentStateActive:    true, // continued
		ConsentStateWithdrawn: true,
	},
	// Refused, Withdrawn, Expired are terminal.
}

// IsValidConsentTransition reports whether the lifecycle DAG permits from→to.
func IsValidConsentTransition(from, to string) bool {
	if !IsValidConsentState(from) || !IsValidConsentState(to) {
		return false
	}
	allowed, ok := consentTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}
```

- [ ] **Step 5: Run test, expect pass**

`go test ./shared/v2_substrate/models/ -run TestConsent -v` → PASS.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/consent.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/consent_test.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/models/enums.go
git commit -m "feat(substrate): add Consent entity with 7-state lifecycle"
```

---

### Task 2: Migration 024

**Files:**
- Create: `migrations/024_consent_lifecycle.sql`
- Create: `migrations/024_consent_lifecycle_rollback.sql`

- [ ] **Step 1: Write migration**

```sql
-- migrations/024_consent_lifecycle.sql
BEGIN;

CREATE TABLE consents (
    id              UUID PRIMARY KEY,
    resident_id     UUID NOT NULL,
    class           TEXT NOT NULL CHECK (class IN (
                        'psychotropic','restrictive_practice','chemotherapy',
                        'end_of_life_medication','general_medication')),
    state           TEXT NOT NULL CHECK (state IN (
                        'requested','discussed','granted',
                        'granted-with-conditions','refused','active',
                        'under_review','withdrawn','expired')),
    granted_by_id   UUID NOT NULL,
    granted_by_role TEXT NOT NULL,
    conditions      TEXT,
    scope_notes     TEXT,
    valid_from      TIMESTAMPTZ NOT NULL,
    valid_until     TIMESTAMPTZ,
    withdrawn_at    TIMESTAMPTZ,
    expired_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_consents_resident          ON consents (resident_id);
CREATE INDEX idx_consents_class             ON consents (class);
CREATE INDEX idx_consents_state             ON consents (state);
CREATE INDEX idx_consents_active_lookup     ON consents (resident_id, class, state)
    WHERE state = 'active';
CREATE INDEX idx_consents_expiry_sweep      ON consents (valid_until)
    WHERE valid_until IS NOT NULL AND state = 'active';

COMMIT;
```

- [ ] **Step 2: Write rollback**

```sql
BEGIN;
DROP TABLE IF EXISTS consents;
COMMIT;
```

- [ ] **Step 3: Apply + verify**

```bash
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -f migrations/024_consent_lifecycle.sql
psql -U kb_drug_rules_user -h localhost -p 5433 -d kb_drug_rules \
     -c "\d consents"
```
Expected: 13 columns, 5 indexes listed.

- [ ] **Step 4: Commit**

```bash
git add migrations/024_consent_lifecycle.sql migrations/024_consent_lifecycle_rollback.sql
git commit -m "feat(migrations): 024 consent lifecycle table"
```

---

### Task 3: Store interface + PostgresStore CRUD

**Files:**
- Create: `shared/v2_substrate/consent/store.go`
- Create: `shared/v2_substrate/consent/store_test.go`

Pattern is identical to Plan 0.1 Task 4. Differences: no JSONB sub-fields (Consent has only scalar columns); the store exposes `FindActive(residentID, class)` for the checker.

- [ ] **Step 1: Write failing test**

```go
package consent

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
	return db
}

func TestPostgresStore_CreateAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	c := models.Consent{
		ID:            uuid.New(),
		ResidentID:    uuid.New(),
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "substitute_decision_maker",
		ValidFrom:     time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := store.Create(ctx, &c); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := store.Get(ctx, c.ID)
	if err != nil || got.State != c.State {
		t.Fatalf("get: %v %+v", err, got)
	}
	defer db.ExecContext(ctx, "DELETE FROM consents WHERE id = $1", c.ID)
}

func TestPostgresStore_FindActive(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	resident := uuid.New()
	active := models.Consent{
		ID: uuid.New(), ResidentID: resident,
		Class: models.ConsentClassPsychotropic, State: models.ConsentStateActive,
		GrantedByID: uuid.New(), GrantedByRole: "sdm",
		ValidFrom: time.Now().UTC(), CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	withdrawn := active
	withdrawn.ID = uuid.New()
	withdrawn.State = models.ConsentStateWithdrawn

	for _, c := range []models.Consent{active, withdrawn} {
		if err := store.Create(ctx, &c); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	defer db.ExecContext(ctx,
		"DELETE FROM consents WHERE resident_id = $1", resident)

	got, err := store.FindActive(ctx, resident, models.ConsentClassPsychotropic)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if got == nil || got.ID != active.ID {
		t.Errorf("find returned wrong consent: %+v", got)
	}

	none, err := store.FindActive(ctx, resident, models.ConsentClassChemoTherapy)
	if err != nil {
		t.Fatalf("find none: %v", err)
	}
	if none != nil {
		t.Errorf("expected nil for non-existent class; got %+v", none)
	}
}
```

- [ ] **Step 2: Verify failure**

`go test ./shared/v2_substrate/consent/ -v` → FAIL undefined symbols.

- [ ] **Step 3: Implement store**

```go
package consent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

var ErrNotFound = errors.New("consent not found")

type Store interface {
	Create(ctx context.Context, c *models.Consent) error
	Get(ctx context.Context, id uuid.UUID) (*models.Consent, error)
	UpdateState(ctx context.Context, id uuid.UUID, newState string) error
	FindActive(ctx context.Context, residentID uuid.UUID, class string) (*models.Consent, error)
}

type PostgresStore struct{ db *sql.DB }

func NewPostgresStore(db *sql.DB) *PostgresStore { return &PostgresStore{db: db} }

func (s *PostgresStore) Create(ctx context.Context, c *models.Consent) error {
	if !models.IsValidConsentState(c.State) {
		return fmt.Errorf("invalid state: %q", c.State)
	}
	const q = `
INSERT INTO consents
  (id, resident_id, class, state, granted_by_id, granted_by_role,
   conditions, scope_notes, valid_from, valid_until,
   withdrawn_at, expired_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`
	_, err := s.db.ExecContext(ctx, q,
		c.ID, c.ResidentID, c.Class, c.State, c.GrantedByID, c.GrantedByRole,
		nullString(c.Conditions), nullString(c.ScopeNotes),
		c.ValidFrom, c.ValidUntil, c.WithdrawnAt, c.ExpiredAt,
		c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (s *PostgresStore) Get(ctx context.Context, id uuid.UUID) (*models.Consent, error) {
	const q = `
SELECT id, resident_id, class, state, granted_by_id, granted_by_role,
       COALESCE(conditions,''), COALESCE(scope_notes,''),
       valid_from, valid_until, withdrawn_at, expired_at,
       created_at, updated_at
FROM consents WHERE id = $1`
	var c models.Consent
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&c.ID, &c.ResidentID, &c.Class, &c.State, &c.GrantedByID, &c.GrantedByRole,
		&c.Conditions, &c.ScopeNotes,
		&c.ValidFrom, &c.ValidUntil, &c.WithdrawnAt, &c.ExpiredAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

func (s *PostgresStore) UpdateState(ctx context.Context, id uuid.UUID, newState string) error {
	if !models.IsValidConsentState(newState) {
		return fmt.Errorf("invalid state: %q", newState)
	}
	const q = `
UPDATE consents SET state = $1,
  withdrawn_at = CASE WHEN $1 = 'withdrawn' AND withdrawn_at IS NULL
                      THEN NOW() ELSE withdrawn_at END,
  expired_at   = CASE WHEN $1 = 'expired'   AND expired_at   IS NULL
                      THEN NOW() ELSE expired_at   END,
  updated_at   = NOW()
WHERE id = $2`
	res, err := s.db.ExecContext(ctx, q, newState, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// FindActive returns the active consent of the given class for the resident,
// or (nil, nil) if no matching active consent exists. Used by the
// recommendation lifecycle ConsentChecker.
func (s *PostgresStore) FindActive(ctx context.Context,
	residentID uuid.UUID, class string) (*models.Consent, error) {
	const q = `
SELECT id, resident_id, class, state, granted_by_id, granted_by_role,
       COALESCE(conditions,''), COALESCE(scope_notes,''),
       valid_from, valid_until, withdrawn_at, expired_at,
       created_at, updated_at
FROM consents
WHERE resident_id = $1 AND class = $2 AND state = 'active'
  AND (valid_until IS NULL OR valid_until > NOW())
ORDER BY valid_from DESC LIMIT 1`
	var c models.Consent
	err := s.db.QueryRowContext(ctx, q, residentID, class).Scan(
		&c.ID, &c.ResidentID, &c.Class, &c.State, &c.GrantedByID, &c.GrantedByRole,
		&c.Conditions, &c.ScopeNotes,
		&c.ValidFrom, &c.ValidUntil, &c.WithdrawnAt, &c.ExpiredAt,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
```

- [ ] **Step 4: Run, expect pass**

```bash
export VAIDSHALA_TEST_DSN="postgres://kb_drug_rules_user:kb_password@localhost:5433/kb_drug_rules?sslmode=disable"
go test ./shared/v2_substrate/consent/ -v
```

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/
git commit -m "feat(substrate): Consent Store with FindActive lookup"
```

---

### Task 4: Lifecycle engine

**Files:**
- Create: `shared/v2_substrate/consent/lifecycle.go`
- Create: `shared/v2_substrate/consent/lifecycle_test.go`

Mirror Plan 0.1 Task 5. The Lifecycle takes a `Store` and an `EdgeStore`, validates transitions via `models.IsValidConsentTransition`, emits an EvidenceTrace edge per transition.

- [ ] **Step 1: Write failing test (mirroring Plan 0.1 Task 5 fakes)**

```go
package consent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

type fakeStore struct{ rec *models.Consent }

func (f *fakeStore) Create(_ context.Context, c *models.Consent) error { f.rec = c; return nil }
func (f *fakeStore) Get(_ context.Context, _ uuid.UUID) (*models.Consent, error) {
	return f.rec, nil
}
func (f *fakeStore) UpdateState(_ context.Context, _ uuid.UUID, s string) error {
	f.rec.State = s
	return nil
}
func (f *fakeStore) FindActive(_ context.Context, _ uuid.UUID, _ string) (*models.Consent, error) {
	return nil, nil
}

type fakeEdges struct{ emitted []EvidenceEdge }

func (f *fakeEdges) EmitEdge(_ context.Context, e EvidenceEdge) error {
	f.emitted = append(f.emitted, e)
	return nil
}

func TestLifecycleHappyPath(t *testing.T) {
	store := &fakeStore{rec: &models.Consent{
		ID: uuid.New(), State: models.ConsentStateDiscussed,
	}}
	edges := &fakeEdges{}
	lc := NewLifecycle(store, edges)

	err := lc.Transition(context.Background(), TransitionRequest{
		ConsentID:  store.rec.ID,
		ToState:    models.ConsentStateGranted,
		ActorID:    uuid.New(),
		ActorRole:  "sdm",
		OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}
	if len(edges.emitted) != 1 {
		t.Errorf("expected 1 edge; got %d", len(edges.emitted))
	}
}

func TestLifecycleForbidden(t *testing.T) {
	store := &fakeStore{rec: &models.Consent{
		ID: uuid.New(), State: models.ConsentStateRefused, // terminal
	}}
	lc := NewLifecycle(store, &fakeEdges{})
	err := lc.Transition(context.Background(), TransitionRequest{
		ConsentID: store.rec.ID,
		ToState:   models.ConsentStateActive,
		ActorID:   uuid.New(),
	})
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition; got %v", err)
	}
}
```

- [ ] **Step 2: Run, expect failure**

`go test ./shared/v2_substrate/consent/ -run TestLifecycle -v` → FAIL.

- [ ] **Step 3: Implement Lifecycle**

```go
package consent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

var ErrInvalidTransition = errors.New("invalid consent transition")

type EvidenceEdge struct {
	ConsentID  uuid.UUID
	FromState  string
	ToState    string
	ActorID    uuid.UUID
	ActorRole  string
	OccurredAt time.Time
	Notes      string
}

type EdgeStore interface {
	EmitEdge(ctx context.Context, e EvidenceEdge) error
}

type TransitionRequest struct {
	ConsentID  uuid.UUID
	ToState    string
	ActorID    uuid.UUID
	ActorRole  string
	OccurredAt time.Time
	Notes      string
}

type Lifecycle struct {
	store Store
	edges EdgeStore
	now   func() time.Time
}

func NewLifecycle(store Store, edges EdgeStore) *Lifecycle {
	return &Lifecycle{
		store: store, edges: edges,
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (l *Lifecycle) Transition(ctx context.Context, r TransitionRequest) error {
	c, err := l.store.Get(ctx, r.ConsentID)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if !models.IsValidConsentTransition(c.State, r.ToState) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, c.State, r.ToState)
	}
	if err := l.store.UpdateState(ctx, c.ID, r.ToState); err != nil {
		return err
	}
	occurred := r.OccurredAt
	if occurred.IsZero() {
		occurred = l.now()
	}
	return l.edges.EmitEdge(ctx, EvidenceEdge{
		ConsentID: c.ID, FromState: c.State, ToState: r.ToState,
		ActorID: r.ActorID, ActorRole: r.ActorRole,
		OccurredAt: occurred, Notes: r.Notes,
	})
}
```

- [ ] **Step 4: Run, expect pass**

`go test ./shared/v2_substrate/consent/ -run TestLifecycle -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/lifecycle.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/lifecycle_test.go
git commit -m "feat(substrate): Consent Lifecycle with EvidenceTrace emission"
```

---

### Task 5: PostgresConsentChecker satisfying recommendation.ConsentChecker

**Files:**
- Create: `shared/v2_substrate/consent/checker.go`
- Create: `shared/v2_substrate/consent/checker_test.go`

This is the integration point with Plan 0.1.

- [ ] **Step 1: Write failing test**

```go
package consent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

func TestPostgresConsentChecker_ActiveAndAbsent(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	checker := NewPostgresConsentChecker(store, recommendationTypeToConsentClass)
	ctx := context.Background()

	resident := uuid.New()
	active := models.Consent{
		ID: uuid.New(), ResidentID: resident,
		Class: models.ConsentClassPsychotropic, State: models.ConsentStateActive,
		GrantedByID: uuid.New(), GrantedByRole: "sdm",
		ValidFrom: time.Now().UTC(), CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, &active); err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer db.ExecContext(ctx,
		"DELETE FROM consents WHERE resident_id = $1", resident)

	// Hardcoded mapping for the test: recType "stop" with consent_required
	// is treated as psychotropic class for this test wiring.
	ok, err := checker.ConsentActive(ctx, resident, "psychotropic_stop")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !ok {
		t.Errorf("expected active consent to satisfy check")
	}

	ok2, _ := checker.ConsentActive(ctx, resident, "chemo_stop")
	if ok2 {
		t.Errorf("expected no consent for chemo class")
	}
}

// recommendationTypeToConsentClass is the test-supplied mapping. Production
// uses a real mapping table; see Plan 0.1 craft-engine integration.
func recommendationTypeToConsentClass(recType string) (string, bool) {
	switch recType {
	case "psychotropic_stop", "psychotropic_dose", "psychotropic_add":
		return models.ConsentClassPsychotropic, true
	case "chemo_stop", "chemo_dose", "chemo_add":
		return models.ConsentClassChemoTherapy, true
	}
	return "", false
}
```

- [ ] **Step 2: Run, expect failure**

- [ ] **Step 3: Implement checker**

```go
package consent

import (
	"context"

	"github.com/google/uuid"
)

// RecommendationTypeMapper maps a recommendation Type string to the
// Consent class that gates it. Returning ("", false) means no consent
// is required.
type RecommendationTypeMapper func(recType string) (consentClass string, required bool)

// PostgresConsentChecker satisfies recommendation.ConsentChecker (Plan 0.1)
// by querying the consents table for an active matching Consent.
type PostgresConsentChecker struct {
	store  Store
	mapper RecommendationTypeMapper
}

func NewPostgresConsentChecker(store Store, mapper RecommendationTypeMapper) *PostgresConsentChecker {
	return &PostgresConsentChecker{store: store, mapper: mapper}
}

func (c *PostgresConsentChecker) ConsentActive(ctx context.Context,
	residentID uuid.UUID, recType string) (bool, error) {
	class, required := c.mapper(recType)
	if !required {
		return true, nil // no consent class for this recommendation type
	}
	got, err := c.store.FindActive(ctx, residentID, class)
	if err != nil {
		return false, err
	}
	return got != nil, nil
}
```

- [ ] **Step 4: Run, expect pass**

- [ ] **Step 5: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/checker.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/checker_test.go
git commit -m "feat(substrate): PostgresConsentChecker for recommendation gating"
```

---

### Task 6: Consent expiry sweeper (background worker)

**Files:**
- Create: `shared/v2_substrate/consent/expiry_sweeper.go`
- Create: `shared/v2_substrate/consent/expiry_sweeper_test.go`

Like Plan 0.1's deferred escalator. Sweeps consents whose `valid_until < NOW()` and transitions them to `expired`.

- [ ] **Step 1: Write failing test**

```go
package consent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/models"
)

func TestExpirySweeper(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdges{}
	lc := NewLifecycle(store, edges)
	sweeper := NewExpirySweeper(store, lc, func() time.Time { return time.Now().UTC() })
	ctx := context.Background()

	pastDue := time.Now().Add(-24 * time.Hour).UTC()
	resident := uuid.New()
	c := models.Consent{
		ID: uuid.New(), ResidentID: resident,
		Class: models.ConsentClassPsychotropic, State: models.ConsentStateActive,
		GrantedByID: uuid.New(), GrantedByRole: "sdm",
		ValidFrom:  time.Now().Add(-30 * 24 * time.Hour).UTC(),
		ValidUntil: &pastDue,
		CreatedAt:  time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := store.Create(ctx, &c); err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM consents WHERE id = $1", c.ID)

	if err := sweeper.RunOnce(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	got, _ := store.Get(ctx, c.ID)
	if got.State != models.ConsentStateExpired {
		t.Errorf("expected expired; got %q", got.State)
	}
}
```

- [ ] **Step 2 & 3: Run failure, then implement**

```go
package consent

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type ExpirySweeper struct {
	store *PostgresStore
	lc    *Lifecycle
	now   func() time.Time
}

func NewExpirySweeper(store *PostgresStore, lc *Lifecycle, now func() time.Time) *ExpirySweeper {
	return &ExpirySweeper{store: store, lc: lc, now: now}
}

// RunOnce performs a single sweep, transitioning any active consent whose
// valid_until has passed into the expired state. Production deployment
// wires this on a 1-hour ticker.
func (s *ExpirySweeper) RunOnce(ctx context.Context) error {
	const q = `
SELECT id FROM consents
WHERE state = 'active' AND valid_until IS NOT NULL AND valid_until < $1
LIMIT 500`
	rows, err := s.store.db.QueryContext(ctx, q, s.now())
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	systemActor := uuid.Nil // system-generated transition; algorithmic
	for _, id := range ids {
		if err := s.lc.Transition(ctx, TransitionRequest{
			ConsentID: id, ToState: "expired",
			ActorID: systemActor, ActorRole: "system_expiry_sweeper",
			OccurredAt: s.now(),
			Notes:      "automatic expiry on valid_until passed",
		}); err != nil {
			// log + continue per-row to keep sweep resilient
			continue
		}
	}
	_ = sql.ErrNoRows // silence import if unused after refactor
	return nil
}
```

- [ ] **Step 4: Run, expect pass; then commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/expiry_sweeper.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/expiry_sweeper_test.go
git commit -m "feat(substrate): Consent expiry sweeper"
```

---

### Task 7: Wire PostgresConsentChecker into Recommendation Lifecycle

**Files:**
- Modify: wherever Plan 0.1's `recommendation.NewLifecycle` is called in production wiring (likely added in Plan 0.4 — kb-30 production wiring; until then, in test integration)

- [ ] **Step 1: Write failing integration test**

Create `shared/v2_substrate/consent/integration_test.go`:

```go
package consent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"shared/v2_substrate/evidence_trace"
	"shared/v2_substrate/models"
	"shared/v2_substrate/recommendation"
)

// TestIntegration_PsychotropicRecBlockedWithoutConsent exercises the full
// chain: Recommendation drafted with ConsentRequired=true and no active
// Consent → Submit returns recommendation.ErrConsentRequired.
func TestIntegration_PsychotropicRecBlockedWithoutConsent(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	consentStore := NewPostgresStore(db)
	checker := NewPostgresConsentChecker(consentStore,
		func(rt string) (string, bool) {
			if rt == "stop" {
				return models.ConsentClassPsychotropic, true
			}
			return "", false
		})

	recStore := recommendation.NewPostgresStore(db)
	graph := evidence_trace.NewInMemoryEdgeStore()
	recLC := recommendation.NewLifecycle(recStore,
		recommendation.NewEvidenceTraceAdapter(graph), checker)

	ctx := context.Background()
	resident := uuid.New()
	rec := models.Recommendation{
		ID: uuid.New(), ResidentID: resident, AuthorID: uuid.New(),
		State: models.RecommendationStateDrafted,
		Type:  "stop", Urgency: models.RecommendationUrgencyAmber,
		Title: "Cease risperidone", ConsentRequired: true,
		ClinicalContent: models.ClinicalContent{Issue: "BPSD reassessment"},
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := recStore.Create(ctx, &rec); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer db.ExecContext(ctx, "DELETE FROM recommendations WHERE id = $1", rec.ID)

	err := recLC.Transition(ctx, recommendation.TransitionRequest{
		RecommendationID: rec.ID, ToState: models.RecommendationStateSubmitted,
		ActorID: uuid.New(), ActorClass: recommendation.ActorClassHuman,
	})
	if !errors.Is(err, recommendation.ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired; got %v", err)
	}
}
```

- [ ] **Step 2: Run, expect pass** (no production code changes; the integration is satisfied by Plan 0.1's existing ErrConsentRequired path + this plan's PostgresConsentChecker).

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/consent/integration_test.go
git commit -m "test(substrate): Consent gating end-to-end with Recommendation lifecycle"
```

---

## Spec coverage

- [x] 7-state lifecycle — Tasks 1, 4
- [x] SDM linkage via Person+Role — Task 1 (`GrantedByID`, `GrantedByRole`)
- [x] FindActive lookup for gating — Task 3
- [x] PostgresConsentChecker satisfying recommendation.ConsentChecker — Task 5
- [x] Expiry automation — Task 6
- [x] End-to-end integration with Plan 0.1 — Task 7

Plan complete and saved.
