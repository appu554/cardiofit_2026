package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// scoring_handlers_test.go covers the handler-shape contract: validation
// 400s for malformed payloads, path-parsing 400s for invalid UUIDs.
//
// End-to-end success paths are exercised by storage/scoring_store_test.go
// (DB-gated). Mounting the full handler stack with a mock store would
// require a fakeable storage interface that the kb-20 package does not
// have today; expanding that here would add scope beyond the Wave 2.6
// plan and is left for a follow-up wave.

func newScoringTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestScoringHandler_RejectsInvalidResidentID(t *testing.T) {
	// Construct handlers with a nil store — the path-parsing 400 fires
	// before the store is touched, so nil is safe for this test.
	h := NewScoringHandlers(nil)
	r := newScoringTestRouter()
	h.RegisterRoutes(r.Group("/v2"))

	for _, path := range []string{
		"/v2/residents/not-a-uuid/cfs",
		"/v2/residents/not-a-uuid/akps",
	} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("POST %s: expected 400 for invalid resident_id; got %d", path, w.Code)
		}
	}
	for _, path := range []string{
		"/v2/residents/not-a-uuid/scores/current",
		"/v2/residents/not-a-uuid/cfs/history",
		"/v2/residents/not-a-uuid/akps/history",
		"/v2/residents/not-a-uuid/dbi/history",
		"/v2/residents/not-a-uuid/acb/history",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("GET %s: expected 400 for invalid resident_id; got %d", path, w.Code)
		}
	}
}

func TestScoringHandler_RejectsMissingAssessedAt(t *testing.T) {
	h := NewScoringHandlers(nil)
	r := newScoringTestRouter()
	h.RegisterRoutes(r.Group("/v2"))

	rid := uuid.New()
	body := map[string]interface{}{
		"assessor_role_ref":  uuid.New().String(),
		"instrument_version": "v2.0",
		"score":              5,
		// assessed_at intentionally omitted
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v2/residents/"+rid.String()+"/cfs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing assessed_at; got %d", w.Code)
	}
}

func TestScoringHandler_RejectsValidationFailures(t *testing.T) {
	h := NewScoringHandlers(nil)
	r := newScoringTestRouter()
	h.RegisterRoutes(r.Group("/v2"))

	rid := uuid.New()
	// CFS score=0 — out of range
	body := map[string]interface{}{
		"assessed_at":        "2026-05-01T09:30:00Z",
		"assessor_role_ref":  uuid.New().String(),
		"instrument_version": "v2.0",
		"score":              0,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v2/residents/"+rid.String()+"/cfs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid CFS score; got %d", w.Code)
	}

	// AKPS score=33 — not multiple of 10
	body["score"] = 33
	b, _ = json.Marshal(body)
	body["instrument_version"] = "abernethy_2005"
	req = httptest.NewRequest(http.MethodPost, "/v2/residents/"+rid.String()+"/akps", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid AKPS score; got %d", w.Code)
	}
}

func TestScoringHandler_RoutesRegistered(t *testing.T) {
	h := NewScoringHandlers(nil)
	r := newScoringTestRouter()
	h.RegisterRoutes(r.Group("/v2"))

	// Verify the seven endpoints respond (even with a 400 from invalid
	// resident_id) — confirms RegisterRoutes wired everything.
	expected := []struct {
		method, path string
	}{
		{http.MethodPost, "/v2/residents/x/cfs"},
		{http.MethodPost, "/v2/residents/x/akps"},
		{http.MethodGet, "/v2/residents/x/scores/current"},
		{http.MethodGet, "/v2/residents/x/cfs/history"},
		{http.MethodGet, "/v2/residents/x/akps/history"},
		{http.MethodGet, "/v2/residents/x/dbi/history"},
		{http.MethodGet, "/v2/residents/x/acb/history"},
	}
	for _, ep := range expected {
		req := httptest.NewRequest(ep.method, ep.path, bytes.NewReader([]byte(`{}`)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code == http.StatusNotFound {
			t.Errorf("%s %s: expected route to be registered; got 404", ep.method, ep.path)
		}
	}
}
