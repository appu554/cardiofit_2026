package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/framing"
)

// newOptOutTestRouter wires an OptOutHandler at /v1/framing/optout/:gp_id
// using a fresh InMemoryOptOutStore. Returns both so tests can assert the
// post-call store state directly.
func newOptOutTestRouter(t *testing.T) (*gin.Engine, *framing.InMemoryOptOutStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	store := framing.NewInMemoryOptOutStore()
	h := NewOptOutHandler(store)
	r.POST("/v1/framing/optout/:gp_id", h.HandleRegister)
	r.DELETE("/v1/framing/optout/:gp_id", h.HandleRevoke)
	return r, store
}

func TestOptOut_Register_HappyPath_Returns201AndOptsOut(t *testing.T) {
	r, store := newOptOutTestRouter(t)
	gp := uuid.New()

	body, _ := json.Marshal(OptOutRequest{Reason: "patient request"})
	req := httptest.NewRequest(http.MethodPost,
		"/v1/framing/optout/"+gp.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp OptOutResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.GPID != gp.String() {
		t.Errorf("gp_id mismatch: got %s want %s", resp.GPID, gp.String())
	}
	if resp.Reason != "patient request" {
		t.Errorf("reason mismatch: got %q", resp.Reason)
	}
	got, _ := store.IsOptedOut(context.Background(), gp)
	if !got {
		t.Errorf("expected store.IsOptedOut=true after 201, got false")
	}
}

func TestOptOut_Register_NoBody_Returns201_ReasonEmpty(t *testing.T) {
	r, store := newOptOutTestRouter(t)
	gp := uuid.New()

	// Empty body — reason is optional.
	req := httptest.NewRequest(http.MethodPost,
		"/v1/framing/optout/"+gp.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 with empty body, got %d (body=%s)", w.Code, w.Body.String())
	}
	var resp OptOutResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Reason != "" {
		t.Errorf("expected empty reason on no-body, got %q", resp.Reason)
	}
	got, _ := store.IsOptedOut(context.Background(), gp)
	if !got {
		t.Errorf("expected IsOptedOut=true after register-without-reason, got false")
	}
}

func TestOptOut_Register_BadUUID_Returns400(t *testing.T) {
	r, _ := newOptOutTestRouter(t)

	req := httptest.NewRequest(http.MethodPost,
		"/v1/framing/optout/not-a-uuid",
		strings.NewReader(`{"reason":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "bad_gp_id" {
		t.Errorf("expected error=bad_gp_id, got %q", body["error"])
	}
}

func TestOptOut_RegisterTwice_Idempotent(t *testing.T) {
	r, store := newOptOutTestRouter(t)
	gp := uuid.New()

	for i, reason := range []string{"first", "second"} {
		body, _ := json.Marshal(OptOutRequest{Reason: reason})
		req := httptest.NewRequest(http.MethodPost,
			"/v1/framing/optout/"+gp.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("call %d: expected 201, got %d (body=%s)", i, w.Code, w.Body.String())
		}
	}
	got, _ := store.IsOptedOut(context.Background(), gp)
	if !got {
		t.Errorf("expected IsOptedOut=true after double Register, got false")
	}
}

func TestOptOut_Revoke_HappyPath_Returns204(t *testing.T) {
	r, store := newOptOutTestRouter(t)
	gp := uuid.New()

	// Pre-register so there is something to revoke.
	if err := store.RegisterOptOut(context.Background(), gp, ""); err != nil {
		t.Fatalf("seed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete,
		"/v1/framing/optout/"+gp.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body=%s)", w.Code, w.Body.String())
	}
	if w.Body.Len() != 0 {
		t.Errorf("expected empty body on 204, got %d bytes", w.Body.Len())
	}
	got, _ := store.IsOptedOut(context.Background(), gp)
	if got {
		t.Errorf("expected IsOptedOut=false after 204, got true")
	}
}

func TestOptOut_Revoke_BadUUID_Returns400(t *testing.T) {
	r, _ := newOptOutTestRouter(t)

	req := httptest.NewRequest(http.MethodDelete,
		"/v1/framing/optout/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "bad_gp_id" {
		t.Errorf("expected error=bad_gp_id, got %q", body["error"])
	}
}

func TestOptOut_Revoke_WhenNotOptedOut_Returns204(t *testing.T) {
	// Documented semantic: DELETE on a GP that has never opted out returns
	// 204 (idempotent no-op). The route's contract is "ensure this GP is
	// not opted out", not "delete a specific record". Picking 204 over 404
	// means clients don't need to pre-check state before issuing revoke.
	r, _ := newOptOutTestRouter(t)
	gp := uuid.New()

	req := httptest.NewRequest(http.MethodDelete,
		"/v1/framing/optout/"+gp.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 (idempotent no-op), got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestOptOut_RegisterRevokeRegister_Roundtrip(t *testing.T) {
	r, store := newOptOutTestRouter(t)
	gp := uuid.New()

	// Register.
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodPost,
		"/v1/framing/optout/"+gp.String(), nil))
	if w1.Code != http.StatusCreated {
		t.Fatalf("register-1: expected 201, got %d", w1.Code)
	}

	// Revoke.
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodDelete,
		"/v1/framing/optout/"+gp.String(), nil))
	if w2.Code != http.StatusNoContent {
		t.Fatalf("revoke: expected 204, got %d", w2.Code)
	}
	if got, _ := store.IsOptedOut(context.Background(), gp); got {
		t.Errorf("expected IsOptedOut=false after revoke")
	}

	// Re-register reactivates.
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, httptest.NewRequest(http.MethodPost,
		"/v1/framing/optout/"+gp.String(), nil))
	if w3.Code != http.StatusCreated {
		t.Fatalf("register-2: expected 201, got %d", w3.Code)
	}
	if got, _ := store.IsOptedOut(context.Background(), gp); !got {
		t.Errorf("expected IsOptedOut=true after re-register")
	}
}
