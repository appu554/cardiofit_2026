package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cardiofit/shared/v2_substrate/permissions"

	"kb-authorisation-evaluator/internal/audit"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newPermMW builds a Middleware backed by in-memory stores so no database is
// needed. Call addPermission to grant access before exercising a route.
func newPermMW() (*permissions.Middleware, *permissions.InMemoryStore, *permissions.InMemoryDataConsentStore) {
	ps := &permissions.InMemoryStore{}
	cs := &permissions.InMemoryDataConsentStore{}
	mw := permissions.NewMiddleware(ps, cs, nil)
	return mw, ps, cs
}

// addPermission creates a ViewPermission granting viewerID access to
// resourceType at the given VisibilityClass for subject subjectID.
func addPermission(
	t *testing.T,
	ps *permissions.InMemoryStore,
	subjectID, viewerID uuid.UUID,
	resourceType string,
	class permissions.VisibilityClass,
) {
	t.Helper()
	_, err := ps.Create(context.Background(), permissions.ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subjectID,
		ViewerRoleID: viewerID,
		Scope: permissions.Scope{
			ViewType:      permissions.ViewTypeRegulator,
			ResourceTypes: []string{resourceType},
			Class:         class,
		},
		GrantedAt:   time.Now().UTC(),
		GrantedByID: uuid.New(),
	})
	require.NoError(t, err)
}

// serverWithPerm returns a Server with PermMW wired and a seeded audit record.
// The returned uuid.UUID is the resident ID used for the seed.
func serverWithPerm(t *testing.T) (*Server, uuid.UUID) {
	t.Helper()
	auditSvc := audit.NewService()
	mw, _, _ := newPermMW()

	srv := &Server{Audit: auditSvc, PermMW: mw}
	residentID := uuid.New()
	seedAudit(auditSvc, residentID, uuid.New())
	return srv, residentID
}

// ---------------------------------------------------------------------------
// Tests: enforcement=ON (PermMW != nil)
// ---------------------------------------------------------------------------

// TestPermissionWiring_WrappedRoute_NoPermission verifies that a GET request
// to a wrapped audit route without any ViewPermission returns 403 Forbidden.
func TestPermissionWiring_WrappedRoute_NoPermission(t *testing.T) {
	srv, residentID := serverWithPerm(t)

	viewerID := uuid.New()
	subjectID := uuid.New() // different from residentID — no permission granted

	req := httptest.NewRequest(http.MethodGet,
		"/v1/audit/resident/"+residentID.String()+"?subject_id="+subjectID.String(), nil)
	// Inject viewer role into context (normally done by upstream JWT middleware).
	req = req.WithContext(permissions.WithViewerRole(req.Context(), viewerID))

	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"wrapped route must return 403 when no ViewPermission exists")
}

// TestPermissionWiring_WrappedRoute_WithPermission verifies that a GET
// request to a wrapped audit route WITH a valid ViewPermission returns 200.
func TestPermissionWiring_WrappedRoute_WithPermission(t *testing.T) {
	auditSvc := audit.NewService()
	mw, ps, _ := newPermMW()
	srv := &Server{Audit: auditSvc, PermMW: mw}

	residentID := uuid.New()
	seedAudit(auditSvc, residentID, uuid.New())

	subjectID := uuid.New()
	viewerID := uuid.New()
	// Grant access: viewer can read audit_resident records for subject.
	addPermission(t, ps, subjectID, viewerID, "audit_resident", permissions.AD)

	req := httptest.NewRequest(http.MethodGet,
		"/v1/audit/resident/"+residentID.String()+"?format=json&subject_id="+subjectID.String(), nil)
	req = req.WithContext(permissions.WithViewerRole(req.Context(), viewerID))

	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"wrapped route must return 200 when a valid ViewPermission exists")
}

// TestPermissionWiring_WrappedRoute_NoViewerRole verifies that a request
// carrying no viewer-role in the context returns 401 Unauthorized.
func TestPermissionWiring_WrappedRoute_NoViewerRole(t *testing.T) {
	srv, residentID := serverWithPerm(t)

	req := httptest.NewRequest(http.MethodGet,
		"/v1/audit/resident/"+residentID.String()+"?subject_id="+uuid.New().String(), nil)
	// No WithViewerRole call — simulates missing/invalid JWT.

	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"wrapped route must return 401 when viewer role is absent from context")
}

// TestPermissionWiring_HealthNotWrapped verifies that /health always returns
// 200 regardless of viewer-role presence or permission records.
func TestPermissionWiring_HealthNotWrapped(t *testing.T) {
	srv, _ := serverWithPerm(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// No viewer role, no permission records — health must still return 200.

	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"/health must always return 200 regardless of permission middleware state")
}

// ---------------------------------------------------------------------------
// Tests: enforcement=OFF (passthrough — PermMW == nil)
// ---------------------------------------------------------------------------

// TestPermissionWiring_Passthrough verifies that when PermMW is nil (i.e.
// KB30_PERMISSIONS_ENFORCED=false), wrapped routes respond with 200 without
// any permission records or JWT tokens. This preserves backward compatibility
// for CI tests that do not set up the permissions stack.
func TestPermissionWiring_Passthrough(t *testing.T) {
	auditSvc := audit.NewService()
	residentID := uuid.New()

	// PermMW intentionally nil — passthrough mode.
	srv := &Server{Audit: auditSvc, PermMW: nil}
	seedAudit(auditSvc, residentID, uuid.New())

	req := httptest.NewRequest(http.MethodGet,
		"/v1/audit/resident/"+residentID.String()+"?format=json", nil)
	// No viewer role, no subject_id — should sail through in passthrough mode.

	w := httptest.NewRecorder()
	srv.Routes().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code,
		"passthrough mode must not block requests lacking JWT/permission context")
}

// ---------------------------------------------------------------------------
// Tests: registry completeness
// ---------------------------------------------------------------------------

// TestPermissionWiredRoutes_RegistryComplete verifies that every entry in
// PermissionWiredRoutes has a non-empty Method, Path, Resource, and a valid
// VisibilityClass.  This is a fast structural guard so the registry stays
// honest as routes are added.
func TestPermissionWiredRoutes_RegistryComplete(t *testing.T) {
	require.NotEmpty(t, PermissionWiredRoutes,
		"PermissionWiredRoutes must not be empty")

	for _, rd := range PermissionWiredRoutes {
		t.Run(rd.Path, func(t *testing.T) {
			assert.NotEmpty(t, rd.Method, "Method must be set")
			assert.NotEmpty(t, rd.Path, "Path must be set")
			assert.NotEmpty(t, rd.Resource, "Resource must be set")
			assert.True(t, rd.Class.Valid(),
				"VisibilityClass %v must be a valid, non-zero class", rd.Class)
		})
	}
}

// TestPermissionWiring_AllAuditRoutesWrapped verifies that every path prefix
// in PermissionWiredRoutes actually returns 401 (not 200) when no viewer role
// is in context — proving the middleware is wired and not a passthrough.
func TestPermissionWiring_AllAuditRoutesWrapped(t *testing.T) {
	auditSvc := audit.NewService()
	mw, _, _ := newPermMW()
	srv := &Server{Audit: auditSvc, PermMW: mw}

	for _, rd := range PermissionWiredRoutes {
		if rd.Method != http.MethodGet {
			continue
		}
		t.Run(rd.Path, func(t *testing.T) {
			// Build a minimal URL for the route — append a dummy UUID where needed.
			target := rd.Path + uuid.New().String() + "?subject_id=" + uuid.New().String()
			req := httptest.NewRequest(http.MethodGet, target, nil)
			// No WithViewerRole — simulates missing JWT.

			w := httptest.NewRecorder()
			srv.Routes().ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code,
				"route %s must be wrapped (returns 401 without viewer role)", rd.Path)
		})
	}
}
