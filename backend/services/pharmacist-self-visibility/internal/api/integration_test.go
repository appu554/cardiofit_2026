package api

// integration_test.go — end-to-end HTTP smoke tests for the pharmacist
// self-visibility service. Each test boots a complete chi router in-process
// using httptest.NewServer (no external dependencies) and exercises the
// full middleware + handler stack.
//
// The four tests cover:
//  1. /healthz → 200 + {"status":"ok"}
//  2. Valid self-JWT → middleware passes → nil dep → 503 (not 403)
//  3. Cross-subject JWT (viewer ≠ subject) → 403
//  4. No Authorization header → 401
//
// Token signing uses signTestToken from jwt_test.go (same package).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// ---------------------------------------------------------------------------
// Server builder (mirrors main.go structure without Postgres)
// ---------------------------------------------------------------------------

const e2eJWTSecret = "e2e-integration-test-secret"

// buildE2EServer constructs an httptest.Server with the same router topology
// as main.go but with in-memory permission stores (no Postgres required).
//
// If subjectID is non-nil, a self-referential ViewPermission covering all
// resource types (worklist, recommendations, …) is inserted so the middleware
// allows the subject's own requests through.
//
// deps controls which dashboard surfaces are wired; pass a zero-value
// DashboardDeps to leave all surfaces nil (produces 503 on successful auth).
func buildE2EServer(t *testing.T, subjectID *uuid.UUID, deps DashboardDeps) *httptest.Server {
	t.Helper()

	store := &permissions.InMemoryStore{}
	consentStore := &permissions.InMemoryDataConsentStore{}

	if subjectID != nil {
		// Insert a single self-referential ViewPermission covering all six resource
		// types. FindForSubjectAndViewer returns the most recent single record, so
		// inserting one permission with all resource types ensures scopeCoversResource
		// matches regardless of which endpoint is hit.
		//
		// Note: PFA resources would normally require an AggregationGate, but for
		// self-access (viewer == subject) the middleware's resolveAccess function
		// returns true without checking the gate. A nil gate is safe here.
		perm := permissions.ViewPermission{
			ID:           uuid.New(),
			SubjectID:    *subjectID,
			ViewerRoleID: *subjectID,
			Scope: permissions.Scope{
				ViewType: permissions.ViewTypePharmacist,
				ResourceTypes: []string{
					"worklist",
					"recommendations",
					"gp_relationships",
					"reasoning",
					"cpd",
					"portfolio",
				},
				Class: permissions.PDP, // most restrictive — self-access always passes
			},
			GrantedAt:   time.Now().Add(-1 * time.Hour),
			GrantedByID: *subjectID,
		}
		_, _ = store.Create(context.Background(), perm)
	}

	mw := permissions.NewMiddleware(store, consentStore, &permissions.NoopAuditEmitter{})

	router := chi.NewRouter()

	// Healthz — unauthenticated, mirrors main.go.
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
	})

	// Authenticated sub-router: JWT middleware + dashboard routes.
	router.Group(func(r chi.Router) {
		r.Use(JWTMiddleware(e2eJWTSecret))
		MountDashboardRoutes(r, mw, deps)
	})

	return httptest.NewServer(router)
}

// ---------------------------------------------------------------------------
// TestE2E_HealthzReturns200
// ---------------------------------------------------------------------------

// TestE2E_HealthzReturns200 boots the server and hits /healthz without any
// Authorization header. Expects 200 + {"status":"ok"} body.
func TestE2E_HealthzReturns200(t *testing.T) {
	srv := buildE2EServer(t, nil, DashboardDeps{})
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("body[status] = %q, want %q", body["status"], "ok")
	}
}

// ---------------------------------------------------------------------------
// TestE2E_DashboardWithSelfJWTReturns503
// ---------------------------------------------------------------------------

// TestE2E_DashboardWithSelfJWTReturns503 boots the server with a valid
// self-referential ViewPermission for subject S, then hits the recommendations
// endpoint as S with subject_id=S. The JWT is valid and the middleware passes
// (viewer == subject with a WO/PDP permission). Because deps.Recommendations
// is nil, the handler returns 503 with code "dependency_unavailable".
//
// This confirms that:
//  - JWT validation passed (not 401)
//  - Permissions middleware passed (not 403)
//  - The nil-dep guard in the handler fired (503, not 200)
func TestE2E_DashboardWithSelfJWTReturns503(t *testing.T) {
	subjectID := uuid.New()
	srv := buildE2EServer(t, &subjectID, DashboardDeps{} /* all nil deps */)
	defer srv.Close()

	token := signTestToken(t, e2eJWTSecret, subjectID.String())
	url := fmt.Sprintf("%s/v1/views/pharmacist/own/recommendations?subject_id=%s",
		srv.URL, subjectID)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (middleware passed, nil dep); "+
			"403 would mean middleware blocked the request", resp.StatusCode)
	}

	var env ErrorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode ErrorEnvelope: %v", err)
	}
	if env.Code != "dependency_unavailable" {
		t.Errorf("Code = %q, want %q", env.Code, "dependency_unavailable")
	}
}

// ---------------------------------------------------------------------------
// TestE2E_DashboardWithCrossSubjectJWTReturns403
// ---------------------------------------------------------------------------

// TestE2E_DashboardWithCrossSubjectJWTReturns403 signs a JWT for viewer X but
// requests subject_id=Y. Because the permissions middleware finds no permission
// for (Y, X) pair (no ViewPermission was inserted for Y) it returns 403 before
// the handler is ever called.
func TestE2E_DashboardWithCrossSubjectJWTReturns403(t *testing.T) {
	viewerX := uuid.New()
	subjectY := uuid.New() // different from viewerX

	// Build server with a self-perm for viewerX only — no perm covering subjectY.
	srv := buildE2EServer(t, &viewerX, DashboardDeps{})
	defer srv.Close()

	token := signTestToken(t, e2eJWTSecret, viewerX.String())
	// Request data for subjectY — viewer X has no permission to see Y's data.
	url := fmt.Sprintf("%s/v1/views/pharmacist/own/recommendations?subject_id=%s",
		srv.URL, subjectY)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (cross-subject, no permission)", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// TestE2E_DashboardWithNoJWTReturns401
// ---------------------------------------------------------------------------

// TestE2E_DashboardWithNoJWTReturns401 sends a request to a dashboard endpoint
// without any Authorization header. The JWT middleware rejects it with 401
// before the permissions middleware or handler is reached.
func TestE2E_DashboardWithNoJWTReturns401(t *testing.T) {
	subjectID := uuid.New()
	srv := buildE2EServer(t, &subjectID, DashboardDeps{})
	defer srv.Close()

	url := fmt.Sprintf("%s/v1/views/pharmacist/own/recommendations?subject_id=%s",
		srv.URL, subjectID)

	// No Authorization header.
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (no bearer token)", resp.StatusCode)
	}
}
