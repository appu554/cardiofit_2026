# Trust Architecture + Pharmacist Self-Visibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land v3 Architectural Commitment 6 (view-permission layer, v3 §3 line 202) plus the pharmacist self-visibility dashboard (MVP-7, v3 §12 line 658). Five view-types — pharmacist, pharmacy employer, RACH, chain, regulator — sit over one substrate; each is permission-scoped; pharmacist data is pharmacist-controlled by default; algorithmic decisions are distinguishable from human decisions in the EvidenceTrace; and a contestation pathway exists for any KPI feeding employment decisions. This is the trust foundation that the v3 bottom-up adoption motion (Buyer 4 in v3 §5 line 280) depends on.

**Architecture:** New package `shared/v2_substrate/permissions/` with `ViewPermission` entity, DSL parser, middleware that wraps every read API, and five view-type adapters (`views/pharmacist.go`, `views/employer.go`, etc.). The `ActorClass` enum from Plan 0.1 already records algorithmic-vs-human; this plan extends EvidenceTrace queries to filter by class. A `Contestation` entity records pharmacist challenges; the integration point is any KPI surfaced upward via the employer view.

**Tech Stack:** Go, PostgreSQL, depends on Plans 0.1 (Recommendation entity → RIR queries are the canonical KPI being permission-scoped) and 0.2 (Consent entity for SDM-permission interactions).

---

## File Structure

**New files:**
- `shared/v2_substrate/permissions/permission.go` — `ViewPermission` entity + scope DSL
- `shared/v2_substrate/permissions/permission_test.go`
- `shared/v2_substrate/permissions/store.go` + `store_test.go`
- `shared/v2_substrate/permissions/middleware.go` — request-time permission resolution
- `shared/v2_substrate/permissions/middleware_test.go`
- `shared/v2_substrate/permissions/views/pharmacist.go` + `_test.go`
- `shared/v2_substrate/permissions/views/employer.go` + `_test.go`
- `shared/v2_substrate/permissions/views/rach.go` + `_test.go`
- `shared/v2_substrate/permissions/views/chain.go` + `_test.go`
- `shared/v2_substrate/permissions/views/regulator.go` + `_test.go`
- `shared/v2_substrate/contestation/contestation.go` + `_test.go`
- `shared/v2_substrate/contestation/store.go` + `_test.go`
- `migrations/027_view_permissions.sql` + rollback
- `migrations/028_contestation.sql` + rollback

**Modified files:**
- `shared/v2_substrate/evidence_trace/query.go` — add `WhereActorClass()` query method
- Plan 0.1 / 0.4 read APIs — wrap with permission middleware

---

### Task 1: Define ViewPermission entity + scope DSL

**Files:**
- Create: `shared/v2_substrate/permissions/permission.go`
- Create: `shared/v2_substrate/permissions/permission_test.go`

The `ViewPermission` entity records: subject (whose data is being viewed), viewer_role (who's viewing), scope (what slices are visible), granted_by, granted_at, expires_at, and contestation_record_ref if granted under contestation.

- [ ] **Step 1: Write failing test**

```go
package permissions

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestViewPermissionEvaluate(t *testing.T) {
	subject := uuid.New()
	viewer := uuid.New()
	p := ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: viewer,
		Scope: Scope{
			ViewType:        ViewTypeEmployer,
			ResourceTypes:   []string{"recommendation_aggregate", "rir_summary"},
			IdentifiableSubjects: false, // aggregate only
		},
		GrantedAt: time.Now().UTC(),
	}
	if !p.Allows("rir_summary", false) {
		t.Errorf("expected allow for rir_summary aggregate")
	}
	if p.Allows("recommendation_individual", false) {
		t.Errorf("expected deny for recommendation_individual not in resource_types")
	}
	if p.Allows("rir_summary", true) {
		t.Errorf("expected deny for identifiable lookup")
	}
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
package permissions

import (
	"time"

	"github.com/google/uuid"
)

const (
	ViewTypePharmacist = "pharmacist"
	ViewTypeEmployer   = "pharmacy_employer"
	ViewTypeRACH       = "rach"
	ViewTypeChain      = "chain"
	ViewTypeRegulator  = "regulator"
)

type Scope struct {
	ViewType             string   `json:"view_type"`
	ResourceTypes        []string `json:"resource_types"` // e.g. ["recommendation","rir_summary"]
	IdentifiableSubjects bool     `json:"identifiable_subjects"` // false = aggregate-only
	FacilityIDs          []uuid.UUID `json:"facility_ids,omitempty"`
}

type ViewPermission struct {
	ID                    uuid.UUID  `json:"id"`
	SubjectID             uuid.UUID  `json:"subject_id"` // whose data
	ViewerRoleID          uuid.UUID  `json:"viewer_role_id"` // role.id of viewer
	Scope                 Scope      `json:"scope"`
	GrantedAt             time.Time  `json:"granted_at"`
	GrantedByID           uuid.UUID  `json:"granted_by_id"` // typically the subject themselves (opt-in)
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
	ContestationRecordRef *uuid.UUID `json:"contestation_record_ref,omitempty"`
}

// Allows reports whether p permits a request to access resourceType,
// identifiable=true means "identifiable individual record" vs aggregate.
func (p ViewPermission) Allows(resourceType string, identifiable bool) bool {
	if p.ExpiresAt != nil && time.Now().After(*p.ExpiresAt) {
		return false
	}
	if identifiable && !p.Scope.IdentifiableSubjects {
		return false
	}
	for _, rt := range p.Scope.ResourceTypes {
		if rt == resourceType {
			return true
		}
	}
	return false
}
```

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/permissions/permission.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/permissions/permission_test.go
git commit -m "feat(substrate): ViewPermission entity with scope DSL"
```

---

### Task 2: Migration 027 — view_permissions table

```sql
BEGIN;
CREATE TABLE view_permissions (
    id                       UUID PRIMARY KEY,
    subject_id               UUID NOT NULL,
    viewer_role_id           UUID NOT NULL,
    scope                    JSONB NOT NULL,
    granted_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by_id            UUID NOT NULL,
    expires_at               TIMESTAMPTZ,
    contestation_record_ref  UUID,
    revoked_at               TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_view_permissions_subject ON view_permissions (subject_id, viewer_role_id)
    WHERE revoked_at IS NULL AND (expires_at IS NULL OR expires_at > NOW());
COMMIT;
```

- [ ] Apply, write rollback, commit. Migration `028_contestation.sql` follows in Task 7.

```bash
git commit -m "feat(migrations): 027 view_permissions table"
```

---

### Task 3: PostgresStore for view permissions

**Files:**
- Create: `shared/v2_substrate/permissions/store.go`, `store_test.go`

Same pattern as Plan 0.1 Task 4. Key methods: `Create`, `Get`, `FindForSubjectAndViewer(subjectID, viewerRoleID)`, `ListBySubject(subjectID)`, `Revoke(id)`.

- [ ] **Step 1-5: Implement, test, commit**

```bash
git commit -m "feat(substrate): ViewPermission Postgres Store"
```

---

### Task 4: Permission middleware

**Files:**
- Create: `shared/v2_substrate/permissions/middleware.go`
- Create: `shared/v2_substrate/permissions/middleware_test.go`

Wraps any HTTP handler. On entry: identify viewer via JWT/auth context, look up active permission for the subject the request targets, deny if none. Audit log every access (allow + deny) into a new EvidenceTrace edge of type `view_access`.

- [ ] **Step 1: Write failing test**

```go
func TestMiddleware_DeniesUnpermittedAccess(t *testing.T) {
	store := newFakeStore() // empty; no permissions exist
	mw := NewMiddleware(store, fakeAuditEmitter{})
	called := false
	handler := mw.Wrap("recommendation", false,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))

	req := httptest.NewRequest("GET", "/foo?subject_id="+uuid.New().String(), nil)
	req = req.WithContext(WithViewerRole(req.Context(), uuid.New()))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403; got %d", w.Code)
	}
	if called {
		t.Errorf("inner handler should not have been called")
	}
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
package permissions

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type viewerRoleKey struct{}

func WithViewerRole(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, viewerRoleKey{}, id)
}
func ViewerRoleFrom(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(viewerRoleKey{}).(uuid.UUID)
	return v, ok
}

type Middleware struct {
	store Store
	audit AuditEmitter // emits view_access EvidenceTrace edges
}

func NewMiddleware(store Store, audit AuditEmitter) *Middleware {
	return &Middleware{store: store, audit: audit}
}

func (m *Middleware) Wrap(resourceType string, identifiable bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viewerRole, ok := ViewerRoleFrom(r.Context())
		if !ok {
			http.Error(w, "no viewer role", http.StatusUnauthorized)
			return
		}
		subjectID, err := uuid.Parse(r.URL.Query().Get("subject_id"))
		if err != nil {
			http.Error(w, "bad subject_id", http.StatusBadRequest)
			return
		}
		perm, err := m.store.FindForSubjectAndViewer(r.Context(), subjectID, viewerRole)
		if err != nil || perm == nil || !perm.Allows(resourceType, identifiable) {
			m.audit.Emit(r.Context(), AuditEvent{
				Subject: subjectID, Viewer: viewerRole,
				Resource: resourceType, Allowed: false,
			})
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		m.audit.Emit(r.Context(), AuditEvent{
			Subject: subjectID, Viewer: viewerRole,
			Resource: resourceType, Allowed: true,
		})
		next.ServeHTTP(w, r)
	})
}

type AuditEvent struct {
	Subject  uuid.UUID
	Viewer   uuid.UUID
	Resource string
	Allowed  bool
}

type AuditEmitter interface {
	Emit(ctx context.Context, e AuditEvent) error
}
```

```bash
git commit -m "feat(substrate): permission middleware with audit emission"
```

---

### Task 5: Five view-type adapters

**Files:**
- Create: `shared/v2_substrate/permissions/views/pharmacist.go` + `_test.go`
- Create: `shared/v2_substrate/permissions/views/employer.go` + `_test.go`
- Create: `shared/v2_substrate/permissions/views/rach.go` + `_test.go`
- Create: `shared/v2_substrate/permissions/views/chain.go` + `_test.go`
- Create: `shared/v2_substrate/permissions/views/regulator.go` + `_test.go`

Each adapter takes a substrate query and returns the slice the view-type is permitted to see. Per v3 §3 line 202: pharmacist sees own data first; employer sees aggregated comparative; RACH sees pharmacy-partner level; chain sees network roll-up; regulator sees evidence-grade audit packs.

- [ ] **Step 1: Write failing test for pharmacist self-view**

```go
package views

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPharmacistView_OwnRIRTrajectory(t *testing.T) {
	pharmacist := uuid.New()
	source := &fakeSource{}
	v := NewPharmacistView(source)

	traj, err := v.OwnRIRTrajectory(context.Background(), pharmacist, 28)
	if err != nil {
		t.Fatalf("traj: %v", err)
	}
	if traj.AuthorID != pharmacist {
		t.Errorf("returned wrong author trajectory: %v", traj.AuthorID)
	}
}

func TestPharmacistView_RejectsCrossPharmacistAccess(t *testing.T) {
	v := NewPharmacistView(&fakeSource{})
	_, err := v.OwnRIRTrajectory(context.Background(), uuid.New(),
		28 /* but the source returned data for a different pharmacist */)
	// The view-adapter constraint: never returns another pharmacist's
	// trajectory in the pharmacist self-view, regardless of underlying
	// source.
	// fakeSource is configured to return mismatched ID; verify rejection.
	if err == nil {
		t.Errorf("expected cross-pharmacist access to error")
	}
}
```

- [ ] **Step 2-5: Implement each view adapter following the pattern**

PharmacistView reads its own RIR trajectory, recommendation pipeline, CPD-relevant cases, per-GP acceptance — all subject = self. Refuses cross-pharmacist queries.

EmployerView returns aggregate: distribution of RIR across pharmacists, with no individual identifiable comparison unless explicit ViewPermission with `IdentifiableSubjects: true` exists.

RACHView returns pharmacy-partner roll-up.

ChainView returns network-level roll-up.

RegulatorView returns audit-grade evidence packs with cryptographic provenance.

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/permissions/views/
git commit -m "feat(substrate): five view-type adapters over single substrate"
```

---

### Task 6: Algorithmic-vs-human EvidenceTrace query

**Files:**
- Modify: `shared/v2_substrate/evidence_trace/query.go`

Plan 0.1 already records `ActorClass` per edge. This task adds query-side filtering: `WhereActorClass(class)` so a Fair Work / AHPRA contestation can pull every algorithmic decision feeding a KPI.

- [ ] **Step 1-5: Add method, test, commit**

```go
// In query.go
func (q *Query) WhereActorClass(class string) *Query {
	q.filters = append(q.filters, "actor_class = $X")
	q.args = append(q.args, class)
	return q
}
```

```bash
git commit -m "feat(substrate): EvidenceTrace WhereActorClass filter for audit queries"
```

---

### Task 7: Contestation entity + pathway

**Files:**
- Create: `shared/v2_substrate/contestation/contestation.go` + `_test.go`
- Create: `shared/v2_substrate/contestation/store.go` + `_test.go`
- Create: `migrations/028_contestation.sql` + rollback

Per v3 §9 line 514. Pharmacist files contestation against an algorithmic KPI; the contestation record is visible to both pharmacist and employer; the algorithmic determination cannot be the sole basis for an adverse employment decision.

- [ ] **Step 1-5: Implement entity + store + integration test (happy path: KPI surfaced; pharmacist contests; record exists; surfaced alongside KPI on next read)**

```sql
-- 028_contestation.sql
BEGIN;
CREATE TABLE contestations (
    id                   UUID PRIMARY KEY,
    pharmacist_id        UUID NOT NULL,
    employer_id          UUID NOT NULL,
    kpi_type             TEXT NOT NULL, -- e.g. 'rir_28d'
    kpi_snapshot         JSONB NOT NULL,
    pharmacist_argument  TEXT NOT NULL,
    employer_response    TEXT,
    status               TEXT NOT NULL CHECK (status IN ('open','responded','resolved','withdrawn')),
    filed_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_contestations_pharmacist ON contestations (pharmacist_id);
CREATE INDEX idx_contestations_kpi        ON contestations (kpi_type, status);
COMMIT;
```

```go
type Contestation struct {
	ID                  uuid.UUID
	PharmacistID        uuid.UUID
	EmployerID          uuid.UUID
	KPIType             string
	KPISnapshot         map[string]any
	PharmacistArgument  string
	EmployerResponse    string
	Status              string
	FiledAt             time.Time
	ResolvedAt          *time.Time
}
```

```bash
git commit -m "feat(substrate): Contestation entity + pathway for algorithmic management compliance"
```

---

### Task 8: Self-visibility dashboard data API

**Files:**
- Modify: kb-20 or new kb-32 service to expose `/views/pharmacist/own` endpoints

Wire the PharmacistView to HTTP endpoints that the future Layer 4 frontend will consume. Endpoints: `/views/pharmacist/own/rir`, `/views/pharmacist/own/pipeline`, `/views/pharmacist/own/cpd-cases`, `/views/pharmacist/own/per-gp-acceptance`.

- [ ] **Step 1-5: Wire endpoints behind permission middleware (subject = self only); test; commit**

```bash
git commit -m "feat: pharmacist self-visibility data API behind permission middleware"
```

---

### Task 9: Integration test — bottom-up adoption motion

**Files:**
- Create: `shared/v2_substrate/permissions/integration_test.go`

Exercise the full chain: pharmacist creates recommendations (Plan 0.1) → reads own RIR via PharmacistView → grants employer aggregate ViewPermission → employer sees aggregate but not identifiable individual data → pharmacist contests; contestation surfaced.

- [ ] **Step 1-5: Write, run, commit**

```bash
git commit -m "test: bottom-up adoption motion end-to-end"
```

---

## Spec coverage

- [x] ViewPermission engine — Tasks 1-3
- [x] Permission middleware — Task 4
- [x] Five view-type adapters — Task 5
- [x] Algorithmic-vs-human EvidenceTrace filtering — Task 6
- [x] Contestation pathway — Task 7
- [x] Pharmacist self-visibility data API — Task 8
- [x] End-to-end bottom-up adoption integration — Task 9

Plan complete and saved.
