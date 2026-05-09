// Package permissions defines the permission model and HTTP middleware for the
// v2 substrate self-visibility framework.
//
// # Two-store check for PDP and PFA non-subject reads
//
// For resources classified as PDP (Pharmacist-Default-Private) or PFA
// (Pharmacist-First-Then-Aggregated), a request by a viewer who is NOT the
// data subject requires both of the following to be true:
//
//  1. A non-revoked, non-expired ViewPermission exists for the
//     (subjectID, viewerRoleID) pair that covers the requested resourceType.
//
//  2. An active DataAggregationConsent exists for (subjectID, resourceType, "")
//     at the time of the request — the empty aggregationTarget means "any target
//     matches", ensuring that element-level consent is present regardless of
//     which downstream aggregator will consume the data.
//
// For PFA resources, the AggregationGate satisfaction check (minimum observation
// count + delay window) requires data-shape context (observation count, period
// start) that only the downstream View adapters possess. That check is therefore
// deferred to Task 5 View adapters. At the middleware level, PFA non-subject
// semantics are equivalent to PDP non-subject semantics: viewer == subject OR
// active DataAggregationConsent.
//
// For POA (Pharmacist-Only-Always) resources only the data subject themselves
// may read their own record — no third-party path exists.
//
// For WO (Workflow-Operational) and AD (Audit-Defensible) resources, the
// ViewPermission alone is sufficient; no consent record is required.
package permissions

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

type viewerRoleKey struct{}

// WithViewerRole stores the viewer's role UUID in ctx. The JWT/auth middleware
// upstream calls this after validating the bearer token.
func WithViewerRole(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, viewerRoleKey{}, id)
}

// ViewerRoleFrom retrieves the viewer UUID previously stored by WithViewerRole.
// ok is false if the context carries no viewer role — callers should return 401.
func ViewerRoleFrom(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(viewerRoleKey{}).(uuid.UUID)
	return v, ok
}

// ---------------------------------------------------------------------------
// AuditEvent + AuditEmitter
// ---------------------------------------------------------------------------

// AuditEvent is emitted for every read request — both allowed and denied — so
// that the EvidenceTrace pipeline can materialise view_access edges.
type AuditEvent struct {
	Subject  uuid.UUID
	Viewer   uuid.UUID
	Resource string
	Allowed  bool
}

// AuditEmitter writes AuditEvents to whatever sink is configured for the
// deployment (EvidenceTrace, structured log, etc.).
type AuditEmitter interface {
	Emit(ctx context.Context, e AuditEvent) error
}

// NoopAuditEmitter discards all events. Use as a default when no audit
// pipeline is wired yet.
type NoopAuditEmitter struct{}

// Compile-time interface satisfaction assertion.
var _ AuditEmitter = (*NoopAuditEmitter)(nil)

// Emit discards e and returns nil.
func (NoopAuditEmitter) Emit(_ context.Context, _ AuditEvent) error { return nil }

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// Middleware enforces ViewPermission and, for PDP/PFA non-subject reads,
// additionally requires an active DataAggregationConsent (two-store check).
// See package-level godoc for full semantics.
type Middleware struct {
	store        Store
	consentStore DataConsentStore
	audit        AuditEmitter
}

// NewMiddleware constructs a Middleware. store and consentStore must be non-nil.
// If audit is nil, a NoopAuditEmitter is used.
func NewMiddleware(store Store, consentStore DataConsentStore, audit AuditEmitter) *Middleware {
	if audit == nil {
		audit = NoopAuditEmitter{}
	}
	return &Middleware{store: store, consentStore: consentStore, audit: audit}
}

// Wrap returns an HTTP handler that enforces read permissions for resourceType
// at the given VisibilityClass before delegating to next.
//
// Request preconditions:
//   - The context must carry a viewer role UUID (set by upstream JWT middleware
//     via WithViewerRole); absent → 401 Unauthorized.
//   - The URL query must contain a valid UUID in the "subject_id" parameter;
//     missing or malformed → 400 Bad Request.
//
// Access decision:
//   - No active ViewPermission for (subject, viewer) → 403 Forbidden.
//   - class == PDP or PFA, viewer != subject, no active DataAggregationConsent → 403.
//   - Otherwise → call next and 200 (or whatever next writes).
//
// An AuditEvent is emitted for every request after the access decision is made.
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
		_ = m.audit.Emit(r.Context(), AuditEvent{
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

// resolveAccess implements the two-store check described in the package godoc.
//
// For POA resources: the ViewPermission must exist and the viewer must be the
// subject (POA = pharmacist-only-always, no third-party path).
//
// For PDP and PFA resources:
//   - If viewer == subject: a ViewPermission covering the resource is sufficient.
//   - If viewer != subject: both a ViewPermission AND an active
//     DataAggregationConsent are required.
//
// For WO and AD resources: the ViewPermission alone is sufficient.
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

	// Verify the permission covers the requested resource type.
	if !scopeCoversResource(perm.Scope, resourceType) {
		return false
	}

	// POA: only the subject themselves may access their own record.
	if class == POA {
		return viewerRole == subjectID
	}

	// PDP / PFA: subject can always access their own data. Non-subject viewers
	// require an active DataAggregationConsent in addition to the ViewPermission.
	// (PFA gate satisfaction — observationCount + periodStart — is deferred to
	// Task 5 View adapters which have the data-shape context.)
	if class == PDP || class == PFA {
		if viewerRole == subjectID {
			return true
		}
		consent, err := m.consentStore.FindActiveConsent(
			ctx, subjectID, resourceType, "", time.Now().UTC(),
		)
		return err == nil && consent != nil
	}

	// WO and AD: ViewPermission alone is sufficient.
	return true
}

// scopeCoversResource reports whether the scope's ResourceTypes slice contains
// resourceType. It does not validate the scope — call Scope.Validate() separately
// when creating permissions.
func scopeCoversResource(s Scope, resourceType string) bool {
	for _, rt := range s.ResourceTypes {
		if rt == resourceType {
			return true
		}
	}
	return false
}
