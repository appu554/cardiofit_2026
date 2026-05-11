package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// ---------------------------------------------------------------------------
// Stub stores for the permissions middleware (avoids requiring a live
// Postgres in unit tests). Only the methods Middleware.resolveAccess actually
// invokes need real behaviour; the rest satisfy the interface contract.
// ---------------------------------------------------------------------------

type stubPermStore struct {
	perm *permissions.ViewPermission
	err  error
}

func (s *stubPermStore) Create(_ context.Context, p permissions.ViewPermission) (permissions.ViewPermission, error) {
	return p, nil
}
func (s *stubPermStore) Get(_ context.Context, _ uuid.UUID) (*permissions.ViewPermission, error) {
	return s.perm, s.err
}
func (s *stubPermStore) FindForSubjectAndViewer(_ context.Context, _, _ uuid.UUID) (*permissions.ViewPermission, error) {
	return s.perm, s.err
}
func (s *stubPermStore) ListBySubject(_ context.Context, _ uuid.UUID) ([]permissions.ViewPermission, error) {
	return nil, nil
}
func (s *stubPermStore) Revoke(_ context.Context, _ uuid.UUID) error { return nil }

type stubConsentStore struct{}

func (stubConsentStore) CreateConsent(_ context.Context, c permissions.DataAggregationConsent) (permissions.DataAggregationConsent, error) {
	return c, nil
}
func (stubConsentStore) FindActiveConsent(_ context.Context, _ uuid.UUID, _, _ string, _ time.Time) (*permissions.DataAggregationConsent, error) {
	return nil, nil
}
func (stubConsentStore) ListByPharmacist(_ context.Context, _ uuid.UUID) ([]permissions.DataAggregationConsent, error) {
	return nil, nil
}
func (stubConsentStore) RevokeConsent(_ context.Context, _ uuid.UUID, _ string) error { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestGinPermMW_Nil_Passthrough verifies that when the middleware is nil
// (KB32_PERMISSIONS_ENFORCED=false / unset), GinPermMW calls c.Next()
// unconditionally and the downstream handler runs.
func TestGinPermMW_Nil_Passthrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	var nextCalled bool
	r.GET("/x",
		GinPermMW(nil, "kb32_test", permissions.PDP),
		func(c *gin.Context) {
			nextCalled = true
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !nextCalled {
		t.Fatalf("expected downstream Gin handler to run in passthrough mode")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}

// TestGinPermMW_Allowed_CallsNext exercises the allow path: a viewer-role
// is set in context, subject_id is a valid UUID, the store returns a
// ViewPermission whose scope covers the resource at class AD, and the
// downstream Gin handler must run with 200 OK.
func TestGinPermMW_Allowed_CallsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	subject := uuid.New()
	viewer := uuid.New()
	perm := &permissions.ViewPermission{
		ID:           uuid.New(),
		SubjectID:    subject,
		ViewerRoleID: viewer,
		Scope: permissions.Scope{
			ViewType:      "test",
			ResourceTypes: []string{"kb32_test"},
			Class:         permissions.AD,
		},
		GrantedAt: time.Now().UTC().Add(-time.Hour),
	}
	mw := permissions.NewMiddleware(&stubPermStore{perm: perm}, stubConsentStore{}, nil)

	r := gin.New()
	var nextCalled bool
	r.GET("/x",
		GinPermMW(mw, "kb32_test", permissions.AD),
		func(c *gin.Context) {
			nextCalled = true
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/x?subject_id="+subject.String(), nil)
	req = req.WithContext(permissions.WithViewerRole(req.Context(), viewer))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !nextCalled {
		t.Fatalf("expected downstream Gin handler to run on allow path")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body=%q)", w.Code, w.Body.String())
	}
}

// TestGinPermMW_Denied_Aborts exercises the deny path: no viewer-role is
// attached to the context, so permissions.Middleware writes
// 401 Unauthorized and never calls its `next` handler. The adapter must
// detect this (allowed flag stays false), c.Abort() the Gin chain, and the
// downstream Gin handler must NOT run.
func TestGinPermMW_Denied_Aborts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Store returns nil — but we never get that far because the middleware
	// rejects the missing viewer-role first (401).
	mw := permissions.NewMiddleware(&stubPermStore{}, stubConsentStore{}, nil)

	r := gin.New()
	var nextCalled bool
	r.GET("/x",
		GinPermMW(mw, "kb32_test", permissions.PDP),
		func(c *gin.Context) {
			nextCalled = true
			c.Status(http.StatusOK)
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if nextCalled {
		t.Fatalf("downstream Gin handler must NOT run when permissions middleware denies")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 from permissions middleware (no viewer role), got %d", w.Code)
	}
}

// TestGinPermMW_Denied_NoPermission_403 exercises a second deny path:
// viewer-role is present and subject_id parses, but the Store has no
// matching ViewPermission, so permissions.Middleware writes 403 Forbidden.
// The Gin handler must not run.
func TestGinPermMW_Denied_NoPermission_403(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mw := permissions.NewMiddleware(&stubPermStore{perm: nil}, stubConsentStore{}, nil)

	r := gin.New()
	var nextCalled bool
	r.GET("/x",
		GinPermMW(mw, "kb32_test", permissions.AD),
		func(c *gin.Context) {
			nextCalled = true
			c.Status(http.StatusOK)
		},
	)

	subject := uuid.New()
	viewer := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/x?subject_id="+subject.String(), nil)
	req = req.WithContext(permissions.WithViewerRole(req.Context(), viewer))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if nextCalled {
		t.Fatalf("downstream Gin handler must NOT run when no ViewPermission exists")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w.Code)
	}
}
