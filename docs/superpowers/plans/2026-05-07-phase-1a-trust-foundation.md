# Phase 1a — Trust Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land v3 Architectural Commitment 6 (view-permission layer, v3 §3 line 202) plus the trust substrate that underlies all pharmacist self-visibility. Five view-types — pharmacist, pharmacy employer, RACH, chain, regulator — sit over one substrate; each is permission-scoped using a 5-class `VisibilityClass` enum; pharmacist data is pharmacist-controlled by default; algorithmic decisions are distinguishable from human decisions in the EvidenceTrace; and a contestation pathway exists for any KPI feeding employment decisions. This is the trust foundation that the v3 bottom-up adoption motion (Buyer 4 in v3 §5 line 280) depends on.

**Phase 1a / 1b / 1c split:**
Phase 1a (this plan) builds the substrate: the `VisibilityClass` enum, `AggregationGate`, `DataAggregationConsent`, `ViewPermission` store, permission middleware, and the `Contestation` pathway. It does not surface any UI or data API endpoints — those are Phase 1b.
- **Phase 1b** (`2026-05-07-phase-1b-self-visibility-surfaces.md`) wires the five view-type adapters and exposes the pharmacist self-visibility dashboard data API (`/views/pharmacist/own/*`). Phase 1b depends on Phase 1a completing cleanly.
- **Phase 1c** (`2026-05-07-phase-1c-ethical-architecture-substrate.md`) implements the ethical architecture substrate per the *Ethical Architecture Implementation Guidelines v1.0*: human-in-the-loop commitment records, the "algorithmic management prohibition" contract clause engine, and independent-review pathway for high-stakes contestations. Phase 1c depends on Phase 1b.

**Architecture:** New package `github.com/cardiofit/shared/v2_substrate/permissions` with `VisibilityClass` enum (5 classes per Self-Visibility Guidelines §2.1), `AggregationGate` (§2.3), `DataAggregationConsent` (§8.1), `ViewPermission` entity, DSL parser, middleware that wraps every read API, and five view-type adapter stubs. The `ActorClass` enum from Plan 0.1 already records algorithmic-vs-human; this plan extends EvidenceTrace queries to filter by class. A `Contestation` entity records pharmacist challenges; the integration point is any KPI surfaced upward via the employer view.

**Tech Stack:** Go, PostgreSQL, depends on Plans 0.1 (Recommendation entity → RIR queries are the canonical KPI being permission-scoped) and 0.2 (Consent entity for SDM-permission interactions — distinct from `DataAggregationConsent` defined here, which covers data-aggregation for pharmacists, not clinical/treatment consent for residents).

---

## File Structure

**New files:**
- `shared/v2_substrate/permissions/permission.go` — `VisibilityClass` enum + `AggregationGate` + `ViewPermission` entity + `Scope` struct
- `shared/v2_substrate/permissions/permission_test.go`
- `shared/v2_substrate/permissions/data_consent.go` — `DataAggregationConsent` entity + Purpose constants
- `shared/v2_substrate/permissions/data_consent_test.go`
- `shared/v2_substrate/permissions/store.go` + `store_test.go`
- `shared/v2_substrate/permissions/middleware.go` — request-time permission resolution (consults ViewPermission + DataAggregationConsent)
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
- `migrations/029_data_aggregation_consent.sql` + rollback

**Modified files:**
- `shared/v2_substrate/evidence_trace/query.go` — add `WhereActorClass()` query method
- Plan 0.1 / 0.4 read APIs — wrap with permission middleware

---

### Task 1: Define ViewPermission entity + VisibilityClass enum + AggregationGate

**Files:**
- Create: `shared/v2_substrate/permissions/permission.go`
- Create: `shared/v2_substrate/permissions/permission_test.go`

The `VisibilityClass` enum has five values per Self-Visibility Guidelines §2.1. The `Scope` struct carries a `VisibilityClass` (not a raw bool) and, for PFA class only, a non-nil `*AggregationGate`. The `ViewPermission` entity records: subject (whose data is being viewed), viewer_role (who's viewing), scope (what slices are visible), granted_by, granted_at, expires_at, and contestation_record_ref if granted under contestation.

- [ ] **Step 1: Write failing test**

```go
package permissions

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestViewPermissionAllows_POA_SubjectOnly(t *testing.T) {
	subject := uuid.New()
	other := uuid.New()
	p := ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: subject, // pharmacist viewing own POA data
		Scope: Scope{
			Class:         POA,
			ResourceTypes: []string{"reflective_writing"},
		},
		GrantedAt: time.Now().UTC(),
	}
	if !p.Allows("reflective_writing", subject) {
		t.Errorf("POA: expected allow when viewerRoleID == subjectID")
	}
	// Reassign viewer to a different role; POA must deny
	p.ViewerRoleID = other
	if p.Allows("reflective_writing", subject) {
		t.Errorf("POA: expected deny when viewerRoleID != subjectID")
	}
}

func TestViewPermissionAllows_PDP_RequiresConsent(t *testing.T) {
	subject := uuid.New()
	employer := uuid.New()
	p := ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: employer,
		Scope: Scope{
			Class:         PDP,
			ResourceTypes: []string{"rir_summary"},
		},
		GrantedAt: time.Now().UTC(),
	}
	// Non-subject viewer without consent path — Allows() enforces subject==viewer for PDP
	if p.Allows("rir_summary", subject) {
		t.Errorf("PDP: expected deny when viewer != subject (consent check is in middleware)")
	}
	// Subject viewing their own PDP data is always allowed
	p.ViewerRoleID = subject
	if !p.Allows("rir_summary", subject) {
		t.Errorf("PDP: expected allow when viewer == subject")
	}
}

func TestViewPermissionAllows_WO_WorkflowParticipants(t *testing.T) {
	subject := uuid.New()
	viewer := uuid.New()
	p := ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: viewer,
		Scope: Scope{
			Class:         WO,
			ResourceTypes: []string{"active_recommendations", "monitoring_obligations"},
		},
		GrantedAt: time.Now().UTC(),
	}
	if !p.Allows("active_recommendations", subject) {
		t.Errorf("WO: expected allow for workflow participant")
	}
	if p.Allows("reflective_writing", subject) {
		t.Errorf("WO: expected deny for resource not in scope")
	}
}

func TestScope_Validate_PFA_RequiresGate(t *testing.T) {
	s := Scope{Class: PFA, ResourceTypes: []string{"rir_summary"}}
	if err := s.Validate(); err == nil {
		t.Error("PFA scope with nil Gate must fail Validate()")
	}
	gate := &AggregationGate{
		MinObservations:  30,
		TimeWindow:       90 * 24 * time.Hour,
		DelayWindow:      30 * 24 * time.Hour,
		ContractualBasis: "enterprise tier deployment §4.2",
		ExplicitNotice:   true,
	}
	s.Gate = gate
	if err := s.Validate(); err != nil {
		t.Errorf("PFA scope with valid Gate must pass Validate(): %v", err)
	}
}

func TestScope_Validate_NonPFA_GateMustBeNil(t *testing.T) {
	gate := &AggregationGate{MinObservations: 30}
	s := Scope{Class: POA, ResourceTypes: []string{"reflective_writing"}, Gate: gate}
	if err := s.Validate(); err == nil {
		t.Error("non-PFA scope with non-nil Gate must fail Validate()")
	}
}

func TestAggregationGate_Satisfied(t *testing.T) {
	gate := AggregationGate{
		MinObservations: 30,
		TimeWindow:      90 * 24 * time.Hour,
		DelayWindow:     30 * 24 * time.Hour,
	}
	now := time.Now().UTC()
	periodStart := now.Add(-100 * 24 * time.Hour) // 100 days ago — within 90d window

	// Happy path: 35 obs, period within window, delay elapsed
	if !gate.Satisfied(35, now, periodStart) {
		t.Error("gate: expected satisfied with 35 obs, 100d period")
	}
	// Insufficient observations
	if gate.Satisfied(10, now, periodStart) {
		t.Error("gate: expected unsatisfied with only 10 obs")
	}
	// Outside time window (period started 200d ago)
	if gate.Satisfied(35, now, now.Add(-200*24*time.Hour)) {
		t.Error("gate: expected unsatisfied when period outside time window")
	}
	// Before delay window (asOf is only 10d after periodStart — less than 30d delay)
	if gate.Satisfied(35, periodStart.Add(10*24*time.Hour), periodStart) {
		t.Error("gate: expected unsatisfied before delay window elapses")
	}
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
package permissions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// VisibilityClass encodes who can see what, per Self-Visibility Guidelines §2.1.
type VisibilityClass int

const (
	POA VisibilityClass = iota // Pharmacist-Only-Always
	PDP                        // Pharmacist-Default-Private
	PFA                        // Pharmacist-First-Then-Aggregated
	WO                         // Workflow-Operational
	AD                         // Audit-Defensible
)

const (
	ViewTypePharmacist = "pharmacist"
	ViewTypeEmployer   = "pharmacy_employer"
	ViewTypeRACH       = "rach"
	ViewTypeChain      = "chain"
	ViewTypeRegulator  = "regulator"
)

// AggregationGate guards PFA-class aggregation per Self-Visibility Guidelines §2.3.
type AggregationGate struct {
	MinObservations  int           `json:"min_observations"`
	TimeWindow       time.Duration `json:"time_window"`
	DelayWindow      time.Duration `json:"delay_window"`
	ContractualBasis string        `json:"contractual_basis"`
	ExplicitNotice   bool          `json:"explicit_notice"`
}

// Satisfied returns true when all PFA gating conditions are met:
//   - observationCount >= MinObservations
//   - periodStart is within TimeWindow looking back from asOf
//   - asOf is at least DelayWindow after periodStart (pharmacist sees first)
func (ag AggregationGate) Satisfied(observationCount int, asOf time.Time, periodStart time.Time) bool {
	if observationCount < ag.MinObservations {
		return false
	}
	windowStart := asOf.Add(-ag.TimeWindow)
	if periodStart.Before(windowStart) {
		return false
	}
	if asOf.Before(periodStart.Add(ag.DelayWindow)) {
		return false
	}
	return true
}

// Scope defines what a ViewPermission grants access to.
type Scope struct {
	ViewType      string          `json:"view_type"`
	ResourceTypes []string        `json:"resource_types"`
	Class         VisibilityClass `json:"class"`
	FacilityIDs   []uuid.UUID     `json:"facility_ids,omitempty"`
	// Gate is required for PFA class and must be nil for all other classes.
	Gate *AggregationGate `json:"gate,omitempty"`
}

// Validate enforces the PFA/non-PFA gate invariant.
func (s Scope) Validate() error {
	if s.Class == PFA && s.Gate == nil {
		return errors.New("PFA scope must have a non-nil AggregationGate")
	}
	if s.Class != PFA && s.Gate != nil {
		return errors.New("non-PFA scope must have nil AggregationGate")
	}
	return nil
}

type ViewPermission struct {
	ID                    uuid.UUID  `json:"id"`
	SubjectID             uuid.UUID  `json:"subject_id"`
	ViewerRoleID          uuid.UUID  `json:"viewer_role_id"`
	Scope                 Scope      `json:"scope"`
	GrantedAt             time.Time  `json:"granted_at"`
	GrantedByID           uuid.UUID  `json:"granted_by_id"`
	ExpiresAt             *time.Time `json:"expires_at,omitempty"`
	ContestationRecordRef *uuid.UUID `json:"contestation_record_ref,omitempty"`
}

// Allows reports whether p permits a read of resourceType,
// given subjectID as the identity of the data subject being requested.
//
// Semantics by VisibilityClass:
//
//	POA — only the subject themselves: returns true iff ViewerRoleID == subjectID
//	PDP — subject always; non-subject viewer requires a DataAggregationConsent (enforced in Middleware)
//	PFA — subject always; aggregator path requires AggregationGate.Satisfied() (enforced in Middleware)
//	WO  — any holder of this ViewPermission may read workflow-operational resources
//	AD  — any holder of this ViewPermission with an AD-grant may read audit-defensible resources
func (p ViewPermission) Allows(resourceType string, subjectID uuid.UUID) bool {
	if p.ExpiresAt != nil && time.Now().UTC().After(*p.ExpiresAt) {
		return false
	}
	// POA and PDP: only the subject themselves at the ViewPermission level.
	// Non-subject PDP/PFA access is gated additionally in the middleware.
	if p.Scope.Class == POA || p.Scope.Class == PDP {
		if p.ViewerRoleID != subjectID {
			return false
		}
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
git commit -m "feat(substrate): VisibilityClass enum + AggregationGate + ViewPermission entity with 5-class scope"
```

---

### Task 2: Migration 027 — view_permissions table

The `scope` JSONB column stores the full `Scope` struct including the `class` enum value and the nullable `gate` object. There is no `identifiable_subjects` bool column — that concept is fully absorbed by `VisibilityClass`.

```sql
-- migrations/027_view_permissions.sql
BEGIN;
CREATE TABLE view_permissions (
    id                       UUID PRIMARY KEY,
    subject_id               UUID NOT NULL,
    viewer_role_id           UUID NOT NULL,
    scope                    JSONB NOT NULL,
    -- scope JSONB shape: { "class": 0..4 (POA/PDP/PFA/WO/AD), "resource_types": [...],
    --                       "view_type": "...", "gate": {...} | null }
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
-- GIN index for class-filtered queries (e.g. find all PFA records for subject)
CREATE INDEX idx_view_permissions_class ON view_permissions
    USING GIN ((scope -> 'class'));
COMMIT;

-- rollback:
-- BEGIN;
-- DROP TABLE IF EXISTS view_permissions;
-- COMMIT;
```

- [ ] Apply, write rollback, commit. Migration `028_contestation.sql` follows in Task 7. Migration `029_data_aggregation_consent.sql` follows in Task 2.5.

```bash
git commit -m "feat(migrations): 027 view_permissions table with VisibilityClass in JSONB scope"
```

---

### Task 2.5: DataAggregationConsent entity + migration 029

**Files:**
- Create: `shared/v2_substrate/permissions/data_consent.go`
- Create: `shared/v2_substrate/permissions/data_consent_test.go`
- Create: `migrations/029_data_aggregation_consent.sql` + rollback

This is **not** the clinical/treatment consent from Plan 0.2 (package `consent/`, table `consents` — covers resident SDM consent). This is data-aggregation consent for pharmacists: purpose-bounded, time-bounded, per-element, revocable. Different semantics, different table, different package path (`github.com/cardiofit/shared/v2_substrate/permissions`).

Per Self-Visibility Guidelines §8.1, a `DataAggregationConsent` specifies the pharmacist, the exact data element (e.g., `rir_class_specific`), the aggregation target, the bounded purpose, and explicit expiry and revocation fields. Consent for one purpose does not extend to another.

- [ ] **Step 1: Write failing tests**

```go
package permissions

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDataAggregationConsent_RevokedDenies(t *testing.T) {
	now := time.Now().UTC()
	revokedAt := now.Add(-1 * time.Hour)
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         now.Add(-30 * 24 * time.Hour),
		ExpiresAt:         now.Add(335 * 24 * time.Hour),
		RevokedAt:         &revokedAt,
	}
	if c.Active(now) {
		t.Error("revoked consent must not be active")
	}
}

func TestDataAggregationConsent_ExpiredDenies(t *testing.T) {
	now := time.Now().UTC()
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeContractRetention,
		GrantedAt:         now.Add(-400 * 24 * time.Hour),
		ExpiresAt:         now.Add(-1 * time.Hour), // already expired
	}
	if c.Active(now) {
		t.Error("expired consent must not be active")
	}
}

func TestDataAggregationConsent_WrongPurposeDenies(t *testing.T) {
	now := time.Now().UTC()
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         now.Add(-30 * 24 * time.Hour),
		ExpiresAt:         now.Add(335 * 24 * time.Hour),
	}
	if c.ActiveForPurpose(PurposeRegulatoryEvidence, now) {
		t.Error("consent granted for workforce_planning must not cover regulatory_evidence")
	}
}

func TestDataAggregationConsent_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	c := DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      uuid.New(),
		DataElement:       "rir_class_specific",
		AggregationTarget: "employer_pharmacy_xyz",
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         now.Add(-30 * 24 * time.Hour),
		ExpiresAt:         now.Add(335 * 24 * time.Hour),
	}
	if !c.Active(now) {
		t.Error("valid non-revoked non-expired consent must be active")
	}
	if !c.ActiveForPurpose(PurposeWorkforcePlanning, now) {
		t.Error("consent must be active for its declared purpose")
	}
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
// shared/v2_substrate/permissions/data_consent.go
package permissions

import (
	"time"

	"github.com/google/uuid"
)

// DataAggregationConsent records a pharmacist's consent for a specific data element
// to be aggregated to a specific target for a specific bounded purpose.
//
// This is distinct from the clinical/treatment consent in Plan 0.2 (package consent/,
// table consents) which covers resident SDM consent. This covers data-aggregation
// consent for pharmacists per Self-Visibility Guidelines §8.1.
type DataAggregationConsent struct {
	ID                uuid.UUID  `json:"id"`
	PharmacistID      uuid.UUID  `json:"pharmacist_id"`
	DataElement       string     `json:"data_element"`       // e.g. "rir_class_specific"
	AggregationTarget string     `json:"aggregation_target"` // e.g. "employer_pharmacy_xyz"
	Purpose           string     `json:"purpose"`            // bounded by Purpose constants
	GrantedAt         time.Time  `json:"granted_at"`
	ExpiresAt         time.Time  `json:"expires_at"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty"`
	RevocationReason  *string    `json:"revocation_reason,omitempty"`
}

// Bounded purpose values per Self-Visibility Guidelines §8.1.
const (
	PurposeWorkforcePlanning  = "workforce_planning"
	PurposeContractRetention  = "contract_retention"
	PurposeRegulatoryEvidence = "regulatory_evidence"
	PurposePeerDevelopment    = "peer_development"
)

// Active returns true if the consent is currently in effect at asOf:
// not revoked and not expired.
func (c DataAggregationConsent) Active(asOf time.Time) bool {
	if c.RevokedAt != nil && !c.RevokedAt.After(asOf) {
		return false
	}
	if !c.ExpiresAt.After(asOf) {
		return false
	}
	return true
}

// ActiveForPurpose returns true if the consent is active and matches the requested
// purpose exactly. Consent for one purpose does not extend to another (§8.1).
func (c DataAggregationConsent) ActiveForPurpose(purpose string, asOf time.Time) bool {
	return c.Active(asOf) && c.Purpose == purpose
}
```

```sql
-- migrations/029_data_aggregation_consent.sql
BEGIN;
CREATE TABLE data_aggregation_consents (
    id                  UUID PRIMARY KEY,
    pharmacist_id       UUID NOT NULL,
    data_element        TEXT NOT NULL,         -- e.g. 'rir_class_specific'
    aggregation_target  TEXT NOT NULL,         -- e.g. 'employer_pharmacy_xyz'
    purpose             TEXT NOT NULL
        CHECK (purpose IN (
            'workforce_planning',
            'contract_retention',
            'regulatory_evidence',
            'peer_development'
        )),
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    revocation_reason   TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary lookup: all consents for a pharmacist on a specific element + target
CREATE INDEX idx_dac_pharmacist_element ON data_aggregation_consents
    (pharmacist_id, data_element, aggregation_target);

-- Partial index for active (non-revoked, non-expired) records — query hot path
CREATE INDEX idx_dac_active ON data_aggregation_consents
    (pharmacist_id, data_element)
    WHERE revoked_at IS NULL AND expires_at > NOW();

COMMIT;

-- rollback:
-- BEGIN;
-- DROP TABLE IF EXISTS data_aggregation_consents;
-- COMMIT;
```

```bash
git add backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/permissions/data_consent.go \
        backend/shared-infrastructure/knowledge-base-services/shared/v2_substrate/permissions/data_consent_test.go \
        backend/shared-infrastructure/knowledge-base-services/migrations/029_data_aggregation_consent.sql
git commit -m "feat(substrate): DataAggregationConsent entity + migration 029 (pharmacist data-aggregation consent, distinct from Plan 0.2 clinical consent)"
```

---

### Task 3: PostgresStore for view permissions

**Files:**
- Create: `shared/v2_substrate/permissions/store.go`, `store_test.go`

Same pattern as Plan 0.1 Task 4. `Store` interface key methods: `Create`, `Get`, `FindForSubjectAndViewer(subjectID, viewerRoleID)`, `ListBySubject(subjectID)`, `Revoke(id)`. Add a `DataConsentStore` interface with: `CreateConsent`, `FindActiveConsent(ctx, pharmacistID, dataElement, aggregationTarget, asOf)`, `ListByPharmacist(pharmacistID)`, `RevokeConsent(id, reason)`.

- [ ] **Step 1-5: Implement both stores, test, commit**

```bash
git commit -m "feat(substrate): ViewPermission + DataConsentStore Postgres implementations"
```

---

### Task 4: Permission middleware

**Files:**
- Create: `shared/v2_substrate/permissions/middleware.go`
- Create: `shared/v2_substrate/permissions/middleware_test.go`

Wraps any HTTP handler. On entry: identify viewer via JWT/auth context, look up active `ViewPermission` for the subject the request targets, deny if none. For **PDP and PFA reads where the viewer is not the subject**, the middleware must additionally consult `DataConsentStore` — a `ViewPermission` alone is insufficient. Audit log every access (allow + deny) into a new EvidenceTrace edge of type `view_access`.

- [ ] **Step 1: Write failing tests**

```go
func TestMiddleware_DeniesUnpermittedAccess(t *testing.T) {
	store := newFakePermStore()           // empty; no permissions exist
	consentStore := newFakeConsentStore() // empty
	mw := NewMiddleware(store, consentStore, fakeAuditEmitter{})
	called := false
	handler := mw.Wrap("recommendation", WO,
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

func TestMiddleware_PDP_DeniesWithoutConsent(t *testing.T) {
	subject := uuid.New()
	employer := uuid.New()
	perm := &ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: employer,
		Scope: Scope{
			Class:         PDP,
			ResourceTypes: []string{"rir_summary"},
		},
		GrantedAt: time.Now().UTC(),
	}
	store := newFakePermStoreWith(perm)
	consentStore := newFakeConsentStore() // no consent record
	mw := NewMiddleware(store, consentStore, fakeAuditEmitter{})
	called := false
	handler := mw.Wrap("rir_summary", PDP,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest("GET", "/foo?subject_id="+subject.String(), nil)
	req = req.WithContext(WithViewerRole(req.Context(), employer))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("PDP without consent: expected 403; got %d", w.Code)
	}
	if called {
		t.Errorf("inner handler must not be called without consent")
	}
}
```

- [ ] **Step 2-5: Implement, test, commit**

```go
package permissions

import (
	"context"
	"net/http"
	"time"

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
	store        Store
	consentStore DataConsentStore
	audit        AuditEmitter
}

func NewMiddleware(store Store, consentStore DataConsentStore, audit AuditEmitter) *Middleware {
	return &Middleware{store: store, consentStore: consentStore, audit: audit}
}

// Wrap returns an HTTP handler that enforces permission for resourceType at the given
// VisibilityClass. For PDP and PFA resources accessed by a non-subject viewer, the
// middleware requires both a ViewPermission record AND an active DataAggregationConsent.
func (m *Middleware) Wrap(resourceType string, class VisibilityClass, next http.Handler) http.Handler {
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

		allowed := m.resolveAccess(r.Context(), viewerRole, subjectID, resourceType, class)
		m.audit.Emit(r.Context(), AuditEvent{
			Subject:  subjectID,
			Viewer:   viewerRole,
			Resource: resourceType,
			Allowed:  allowed,
		})
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// resolveAccess applies the two-store check for PDP/PFA non-subject reads.
func (m *Middleware) resolveAccess(
	ctx context.Context,
	viewerRole, subjectID uuid.UUID,
	resourceType string,
	class VisibilityClass,
) bool {
	perm, err := m.store.FindForSubjectAndViewer(ctx, subjectID, viewerRole)
	if err != nil || perm == nil {
		return false
	}
	if !perm.Allows(resourceType, subjectID) {
		return false
	}
	// For PDP and PFA reads by a non-subject viewer, require an active DataAggregationConsent.
	if (class == PDP || class == PFA) && viewerRole != subjectID {
		consent, err := m.consentStore.FindActiveConsent(ctx, subjectID, resourceType, "", time.Now().UTC())
		if err != nil || consent == nil {
			return false
		}
	}
	return true
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
git commit -m "feat(substrate): permission middleware with two-store check (ViewPermission + DataAggregationConsent) for PDP/PFA reads"
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
		28 /* fakeSource returns mismatched ID; verify rejection */)
	if err == nil {
		t.Errorf("expected cross-pharmacist access to error")
	}
}
```

- [ ] **Step 2-5: Implement each view adapter following the pattern**

PharmacistView reads its own RIR trajectory, recommendation pipeline, CPD-relevant cases, per-GP acceptance — all subject = self. Refuses cross-pharmacist queries. POA data is never returned to non-self callers.

EmployerView returns aggregate: distribution of RIR across pharmacists, with no per-pharmacist identifiable data unless a valid `DataAggregationConsent` exists for that pharmacist and the `AggregationGate` is satisfied.

RACHView returns pharmacy-partner roll-up.

ChainView returns network-level roll-up.

RegulatorView returns audit-grade evidence packs with cryptographic provenance (AD-class only).

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

-- rollback:
-- BEGIN;
-- DROP TABLE IF EXISTS contestations;
-- COMMIT;
```

```go
type Contestation struct {
	ID                 uuid.UUID
	PharmacistID       uuid.UUID
	EmployerID         uuid.UUID
	KPIType            string
	KPISnapshot        map[string]any
	PharmacistArgument string
	EmployerResponse   string
	Status             string
	FiledAt            time.Time
	ResolvedAt         *time.Time
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

- [ ] **Step 1-5: Wire endpoints behind permission middleware (subject = self only; VisibilityClass enforced at data-fetch boundary, not UI layer); test; commit**

```bash
git commit -m "feat: pharmacist self-visibility data API behind permission middleware"
```

---

### Task 9: Integration test — bottom-up adoption motion

**Files:**
- Create: `shared/v2_substrate/permissions/integration_test.go`

Exercise the full chain: pharmacist creates recommendations (Plan 0.1) → reads own RIR via PharmacistView (POA/PDP data visible to self) → grants employer aggregate `ViewPermission` with PFA `Scope` and a valid `AggregationGate` → grants `DataAggregationConsent` for `rir_class_specific` with purpose `workforce_planning` → employer middleware checks both stores → employer sees aggregated data only after gate satisfied → pharmacist contests; contestation surfaced alongside employer-visible KPI.

- [ ] **Step 1-5: Write, run, commit**

```bash
git commit -m "test: bottom-up adoption motion end-to-end with 5-class visibility + aggregation gate + data consent"
```

---

## Spec coverage

- [x] 5-class VisibilityClass enum (POA / PDP / PFA / WO / AD) — Task 1
- [x] AggregationGate struct + Satisfied() with unit tests (insufficient obs / outside window / before delay / happy path) — Task 1
- [x] Scope validation: PFA requires Gate; non-PFA Gate must be nil — Task 1
- [x] ViewPermission entity with Allows() semantics per class — Task 1
- [x] Migration 027 — view_permissions table (VisibilityClass in JSONB scope; no identifiable_subjects bool) — Task 2
- [x] DataAggregationConsent entity + Purpose constants (workforce_planning / contract_retention / regulatory_evidence / peer_development) — Task 2.5
- [x] Migration 029 — data_aggregation_consents table (distinct from Plan 0.2 clinical consent) — Task 2.5
- [x] DataConsentStore interface — Task 3
- [x] Permission middleware with two-store check (ViewPermission + DataAggregationConsent for PDP/PFA non-subject reads) — Task 4
- [x] Five view-type adapters — Task 5
- [x] Algorithmic-vs-human EvidenceTrace filtering — Task 6
- [x] Contestation pathway — Task 7
- [x] Pharmacist self-visibility data API — Task 8
- [x] End-to-end bottom-up adoption integration — Task 9

Plan complete and saved.
