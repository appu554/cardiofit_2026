package permissions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Test helpers: fake stores
// ---------------------------------------------------------------------------

func newFakePermStore() *InMemoryStore { return &InMemoryStore{} }

func newFakePermStoreWith(p *ViewPermission) *InMemoryStore {
	s := &InMemoryStore{}
	if p != nil {
		s.records = append(s.records, inMemoryViewPerm{p: *p})
	}
	return s
}

func newFakeConsentStore() *InMemoryDataConsentStore { return &InMemoryDataConsentStore{} }

func newFakeConsentStoreWith(c *DataAggregationConsent) *InMemoryDataConsentStore {
	s := &InMemoryDataConsentStore{}
	if c != nil {
		s.records = append(s.records, *c)
	}
	return s
}

// ---------------------------------------------------------------------------
// Test helpers: audit emitter
// ---------------------------------------------------------------------------

// capturingAuditEmitter records every AuditEvent emitted during the request.
type capturingAuditEmitter struct {
	mu     sync.Mutex
	events []AuditEvent
}

func (e *capturingAuditEmitter) Emit(_ context.Context, ev AuditEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
	return nil
}

func (e *capturingAuditEmitter) captured() []AuditEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := make([]AuditEvent, len(e.events))
	copy(cp, e.events)
	return cp
}

// fakeAuditEmitter is the zero-value emitter used for tests that don't inspect audit events.
type fakeAuditEmitter struct{}

func (fakeAuditEmitter) Emit(_ context.Context, _ AuditEvent) error { return nil }

// ---------------------------------------------------------------------------
// Helper: build a minimal ViewPermission
// ---------------------------------------------------------------------------

func buildPerm(subjectID, viewerRoleID uuid.UUID, class VisibilityClass, resourceTypes ...string) *ViewPermission {
	return &ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subjectID,
		ViewerRoleID: viewerRoleID,
		Scope: Scope{
			Class:         class,
			ResourceTypes: resourceTypes,
		},
		GrantedAt: time.Now().UTC().Add(-time.Minute),
	}
}

// buildActiveConsent creates a DataAggregationConsent active well into the future.
func buildActiveConsent(subjectID uuid.UUID, dataElement string) *DataAggregationConsent {
	return &DataAggregationConsent{
		ID:                uuid.New(),
		PharmacistID:      subjectID,
		DataElement:       dataElement,
		AggregationTarget: "test_employer",
		Purpose:           PurposeWorkforcePlanning,
		GrantedAt:         time.Now().UTC().Add(-time.Minute),
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
	}
}

// ---------------------------------------------------------------------------
// Plan-verbatim tests (2)
// ---------------------------------------------------------------------------

// TestMiddleware_DeniesUnpermittedAccess: no ViewPermission exists → 403,
// inner handler not called.
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

// TestMiddleware_PDP_DeniesWithoutConsent: valid ViewPermission exists for PDP
// resource, viewer != subject, but no DataAggregationConsent → 403.
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

// ---------------------------------------------------------------------------
// Additional tests (7)
// ---------------------------------------------------------------------------

// TestMiddleware_AllowsSubjectViewingOwnPOA: viewer == subject for POA-class
// resource with a matching ViewPermission → 200, inner handler called.
func TestMiddleware_AllowsSubjectViewingOwnPOA(t *testing.T) {
	subject := uuid.New()
	perm := buildPerm(subject, subject, POA, "clinical_note")
	store := newFakePermStoreWith(perm)
	consentStore := newFakeConsentStore()
	mw := NewMiddleware(store, consentStore, fakeAuditEmitter{})

	called := false
	handler := mw.Wrap("clinical_note", POA,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest("GET", "/foo?subject_id="+subject.String(), nil)
	req = req.WithContext(WithViewerRole(req.Context(), subject))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POA self-view: expected 200; got %d", w.Code)
	}
	if !called {
		t.Errorf("inner handler should have been called for subject self-view")
	}
}

// TestMiddleware_AllowsPDPSelfWithoutConsent: viewer == subject for PDP-class →
// DataAggregationConsent is NOT required for self-access; should return 200.
func TestMiddleware_AllowsPDPSelfWithoutConsent(t *testing.T) {
	subject := uuid.New()
	perm := buildPerm(subject, subject, PDP, "rir_summary")
	store := newFakePermStoreWith(perm)
	consentStore := newFakeConsentStore() // no consent — should not matter for self
	mw := NewMiddleware(store, consentStore, fakeAuditEmitter{})

	called := false
	handler := mw.Wrap("rir_summary", PDP,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest("GET", "/foo?subject_id="+subject.String(), nil)
	req = req.WithContext(WithViewerRole(req.Context(), subject))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PDP self-view: expected 200; got %d", w.Code)
	}
	if !called {
		t.Errorf("inner handler should be called for PDP self-view")
	}
}

// TestMiddleware_PDP_AllowsWithConsent: viewer != subject, PDP-class, but an active
// DataAggregationConsent exists for (subject, resourceType, "") → 200.
func TestMiddleware_PDP_AllowsWithConsent(t *testing.T) {
	subject := uuid.New()
	employer := uuid.New()
	perm := buildPerm(subject, employer, PDP, "rir_summary")
	store := newFakePermStoreWith(perm)
	consent := buildActiveConsent(subject, "rir_summary")
	consentStore := newFakeConsentStoreWith(consent)
	mw := NewMiddleware(store, consentStore, fakeAuditEmitter{})

	called := false
	handler := mw.Wrap("rir_summary", PDP,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}))

	req := httptest.NewRequest("GET", "/foo?subject_id="+subject.String(), nil)
	req = req.WithContext(WithViewerRole(req.Context(), employer))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PDP with consent: expected 200; got %d", w.Code)
	}
	if !called {
		t.Errorf("inner handler should be called when consent exists")
	}
}

// TestMiddleware_PFA_GateUnsatisfied_Denies: PFA-class, viewer != subject, no
// DataAggregationConsent exists. The middleware requires active consent for PFA
// non-subject reads (gate satisfaction check is deferred to Task 5 View adapters
// which have data-shape context). Without consent → 403.
func TestMiddleware_PFA_GateUnsatisfied_Denies(t *testing.T) {
	subject := uuid.New()
	aggregator := uuid.New()
	gate := &AggregationGate{
		MinObservations:  10,
		TimeWindow:       30 * 24 * time.Hour,
		DelayWindow:      7 * 24 * time.Hour,
		ContractualBasis: "employer_contract_v1",
		ExplicitNotice:   true,
	}
	perm := &ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: aggregator,
		Scope: Scope{
			Class:         PFA,
			ResourceTypes: []string{"shift_utilisation"},
			Gate:          gate,
		},
		GrantedAt: time.Now().UTC().Add(-time.Minute),
	}
	store := newFakePermStoreWith(perm)
	consentStore := newFakeConsentStore() // no consent
	mw := NewMiddleware(store, consentStore, fakeAuditEmitter{})

	called := false
	handler := mw.Wrap("shift_utilisation", PFA,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))

	req := httptest.NewRequest("GET", "/foo?subject_id="+subject.String(), nil)
	req = req.WithContext(WithViewerRole(req.Context(), aggregator))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("PFA without consent: expected 403; got %d", w.Code)
	}
	if called {
		t.Errorf("inner handler must not be called without consent")
	}
}

// TestMiddleware_AuditEmitsAllowAndDeny: verify the audit emitter receives an
// AuditEvent for both allowed and denied requests, with the correct Allowed field.
func TestMiddleware_AuditEmitsAllowAndDeny(t *testing.T) {
	subject := uuid.New()
	viewer := uuid.New()

	// --- allowed case ---
	perm := buildPerm(subject, subject, WO, "schedule")
	allowStore := newFakePermStoreWith(perm)
	allowAudit := &capturingAuditEmitter{}
	mwAllow := NewMiddleware(allowStore, newFakeConsentStore(), allowAudit)
	handlerAllow := mwAllow.Wrap("schedule", WO, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	reqAllow := httptest.NewRequest("GET", "/foo?subject_id="+subject.String(), nil)
	reqAllow = reqAllow.WithContext(WithViewerRole(reqAllow.Context(), subject))
	wAllow := httptest.NewRecorder()
	handlerAllow.ServeHTTP(wAllow, reqAllow)

	allowedEvents := allowAudit.captured()
	if len(allowedEvents) != 1 {
		t.Fatalf("allow: expected 1 audit event; got %d", len(allowedEvents))
	}
	if !allowedEvents[0].Allowed {
		t.Errorf("allow: AuditEvent.Allowed should be true")
	}
	if allowedEvents[0].Subject != subject {
		t.Errorf("allow: AuditEvent.Subject mismatch")
	}

	// --- denied case ---
	denyStore := newFakePermStore() // empty
	denyAudit := &capturingAuditEmitter{}
	mwDeny := NewMiddleware(denyStore, newFakeConsentStore(), denyAudit)
	handlerDeny := mwDeny.Wrap("schedule", WO, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	reqDeny := httptest.NewRequest("GET", "/foo?subject_id="+subject.String(), nil)
	reqDeny = reqDeny.WithContext(WithViewerRole(reqDeny.Context(), viewer))
	wDeny := httptest.NewRecorder()
	handlerDeny.ServeHTTP(wDeny, reqDeny)

	deniedEvents := denyAudit.captured()
	if len(deniedEvents) != 1 {
		t.Fatalf("deny: expected 1 audit event; got %d", len(deniedEvents))
	}
	if deniedEvents[0].Allowed {
		t.Errorf("deny: AuditEvent.Allowed should be false")
	}
	if deniedEvents[0].Viewer != viewer {
		t.Errorf("deny: AuditEvent.Viewer mismatch")
	}
}

// TestMiddleware_NoViewerRole_Returns401: no viewer role in context → 401,
// no audit event emitted (middleware returns before resolving access).
func TestMiddleware_NoViewerRole_Returns401(t *testing.T) {
	mw := NewMiddleware(newFakePermStore(), newFakeConsentStore(), fakeAuditEmitter{})
	called := false
	handler := mw.Wrap("recommendation", WO,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}))

	// No WithViewerRole call — context has no viewer role.
	req := httptest.NewRequest("GET", "/foo?subject_id="+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401; got %d", w.Code)
	}
	if called {
		t.Errorf("inner handler must not be called without viewer role")
	}
}

// TestMiddleware_BadSubjectID_Returns400: missing or malformed subject_id query
// param → 400 Bad Request.
func TestMiddleware_BadSubjectID_Returns400(t *testing.T) {
	mw := NewMiddleware(newFakePermStore(), newFakeConsentStore(), fakeAuditEmitter{})
	handler := mw.Wrap("recommendation", WO, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	cases := []struct {
		name string
		url  string
	}{
		{"missing", "/foo"},
		{"malformed", "/foo?subject_id=not-a-uuid"},
		{"empty", "/foo?subject_id="},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			req = req.WithContext(WithViewerRole(req.Context(), uuid.New()))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: expected 400; got %d", tc.name, w.Code)
			}
		})
	}
}
