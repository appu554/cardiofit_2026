// api_completeness_test asserts the Task 8 invariant: every one of the
// 17 S2 routes specified in S2 v1.0 Part 16 is registered on the Gin
// engine, and every action route is gated with PDP visibility class.
//
// The route table is authoritative for Task 8; if a route name / path
// changes in the api package, this test must be updated in lockstep so
// the gRPC IDL (proto/v1/s2_workspace.proto) stays aligned with the
// HTTP surface (Step 4 Task E + Phase 2-completion Task 7 pattern).
package structural

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/actions"
	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/api"
)

func newRecorder() *httptest.ResponseRecorder { return httptest.NewRecorder() }

func newRequestJSON(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestAPICompleteness_AllSeventeenRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := actions.NewInMemoryActionStore()
	sessions := actions.NewInMemorySessionStore()
	forwarder := actions.NewInMemoryOverrideForwarder()
	vb := aggregation.NewDefaultViewBuilder()
	h := actions.NewHandler(store, sessions, forwarder, vb)
	srv := api.NewServer(api.Dependencies{
		ViewBuilder:   vb,
		ActionHandler: h,
		SessionStore:  sessions,
	})
	r := gin.New()
	srv.RegisterRoutes(r)

	type want struct {
		method string
		path   string
	}
	expected := []want{
		// Rendering
		{http.MethodPost, "/v1/s2/workspace"},
		{http.MethodPost, "/v1/s2/workspace/refresh"},
		// Drill-through
		{http.MethodGet, "/v1/s2/substrate/:resident_id/:substrate_type/:substrate_id"},
		{http.MethodGet, "/v1/s2/trajectory/:resident_id/:parameter"},
		// 11 actions
		{http.MethodPost, "/v1/s2/actions/open"},
		{http.MethodPost, "/v1/s2/actions/modify"},
		{http.MethodPost, "/v1/s2/actions/defer"},
		{http.MethodPost, "/v1/s2/actions/override"},
		{http.MethodPost, "/v1/s2/actions/mark_reviewed"},
		{http.MethodPost, "/v1/s2/actions/flag"},
		{http.MethodPost, "/v1/s2/actions/note"},
		{http.MethodPost, "/v1/s2/actions/escalate_to_complex"},
		{http.MethodPost, "/v1/s2/actions/acknowledge_restraint"},
		{http.MethodPost, "/v1/s2/actions/safety_bypass"},
		// Audit + session
		{http.MethodGet, "/v1/s2/audit/:resident_id"},
		{http.MethodPost, "/v1/s2/session/start"},
		{http.MethodPost, "/v1/s2/session/end"},
	}
	if len(expected) != 17 {
		t.Fatalf("expected route table length 17, got %d (test bug)", len(expected))
	}
	got := r.Routes()
	gotIndex := map[string]bool{}
	for _, route := range got {
		gotIndex[route.Method+" "+route.Path] = true
	}
	for _, want := range expected {
		key := want.method + " " + want.path
		if !gotIndex[key] {
			t.Errorf("missing route: %s %s", want.method, want.path)
		}
	}
	// Strict count: no surprise routes registered alongside the
	// canonical set. Allow non-/v1/s2 routes (e.g. /healthz) but
	// every /v1/s2 route must be in the expected set.
	for _, route := range got {
		if len(route.Path) >= 6 && route.Path[:6] == "/v1/s2" {
			key := route.Method + " " + route.Path
			found := false
			for _, want := range expected {
				if want.method+" "+want.path == key {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("unexpected route registered: %s", key)
			}
		}
	}
}

// TestAPICompleteness_AllActionsGatedByPDP verifies that every action
// route runs through GinPermMW with the PDP visibility class. The test
// uses a recording Middleware that captures (resource, class) pairs
// and asserts each action route's class is PDP.
type recordingMW struct {
	calls []recordedCall
}

type recordedCall struct {
	resource string
	class    api.VisibilityClass
}

func (m *recordingMW) Wrap(resource string, class api.VisibilityClass, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.calls = append(m.calls, recordedCall{resource: resource, class: class})
		next.ServeHTTP(w, r)
	})
}

func TestAPICompleteness_ActionsGatedByPDP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := actions.NewInMemoryActionStore()
	sessions := actions.NewInMemorySessionStore()
	forwarder := actions.NewInMemoryOverrideForwarder()
	vb := aggregation.NewDefaultViewBuilder()
	h := actions.NewHandler(store, sessions, forwarder, vb)
	rec := &recordingMW{}
	srv := api.NewServer(api.Dependencies{
		ViewBuilder:   vb,
		ActionHandler: h,
		SessionStore:  sessions,
		PermsMW:       rec,
	})
	r := gin.New()
	srv.RegisterRoutes(r)

	// Hit every action route once. Bodies are intentionally minimal
	// (and may 400) — we're asserting only on the middleware-recorded
	// (resource, class) tuples, which are captured before body validation.
	actionRoutes := []string{
		"/v1/s2/actions/open",
		"/v1/s2/actions/modify",
		"/v1/s2/actions/defer",
		"/v1/s2/actions/override",
		"/v1/s2/actions/mark_reviewed",
		"/v1/s2/actions/flag",
		"/v1/s2/actions/note",
		"/v1/s2/actions/escalate_to_complex",
		"/v1/s2/actions/acknowledge_restraint",
		"/v1/s2/actions/safety_bypass",
	}
	for _, path := range actionRoutes {
		w := newRecorder()
		req := newRequestJSON(t, http.MethodPost, path, map[string]string{
			"pharmacist_id": uuid.New().String(),
			"resident_id":   uuid.New().String(),
			"session_id":    uuid.New().String(),
		})
		r.ServeHTTP(w, req)
	}
	if len(rec.calls) < len(actionRoutes) {
		t.Fatalf("middleware recorded %d calls, want >= %d",
			len(rec.calls), len(actionRoutes))
	}
	for i, call := range rec.calls {
		if call.class != api.PDP {
			t.Errorf("call %d (resource=%q): class=%v, want PDP",
				i, call.resource, call.class)
		}
	}
}
