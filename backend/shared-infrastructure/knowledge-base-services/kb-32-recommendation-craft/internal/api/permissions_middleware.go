// Permissions middleware adapter for kb-32.
//
// The Phase 1a permissions.Middleware (shared/v2_substrate/permissions) is
// implemented over net/http. kb-32, however, uses Gin throughout. This file
// provides a thin adapter — GinPermMW — that bridges the two contracts so the
// Phase 1a Wrap(resource, class, next) flow can guard Gin routes without
// changing the middleware package itself.
//
// # Pattern
//
// Mirrors kb-30's Server.wrapRead helper in
//   backend/shared-infrastructure/knowledge-base-services/kb-30-authorisation-evaluator/internal/api/rest.go
// (read lines 122–131). kb-30 uses native net/http, so it can pass the
// downstream handler directly to PermMW.Wrap. kb-32 cannot — Gin's handler
// chain is gin.HandlerFunc, not http.Handler — so we run the permissions
// Wrap against a *passthrough* http.Handler that flips an "allowed" flag, and
// then either continue Gin's chain (allowed) or abort it (denied).
//
// The denial response (status + body) is written by permissions.Middleware
// itself onto c.Writer — see middleware.go lines 125–151. When permissions
// denies, it calls http.Error on the writer (401 / 400 / 403). All we have
// to do on the Gin side is c.Abort() so no downstream handler runs after the
// status has been written.
//
// # Passthrough mode
//
// When KB32_PERMISSIONS_ENFORCED is unset (or false), main.go constructs the
// adapter with mw == nil. In that case GinPermMW returns a handler that
// simply calls c.Next() — equivalent to no middleware. This is what keeps
// the existing handlers_test.go suite (which never sets the env var and has
// no JWT/permission setup) green.
//
// # Class choice
//
// craft/draft and craft/override are write-side decisions on resident
// clinical data. The kb-32 main.go comments (line 259 and 306–309) already
// document the intent to gate these with "PDP middleware" — Pharmacist-
// Default-Private, the Phase 1a class that requires both a ViewPermission
// and an active DataAggregationConsent for non-subject viewers. We adopt
// PDP accordingly; AD (audit-defensible) would be wrong here because these
// are clinical-care actions, not regulatory event reads.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// GinPermMW returns a gin.HandlerFunc that enforces the Phase 1a permissions
// check for (resource, class) before the next Gin handler runs.
//
// Behaviour:
//   - mw == nil  → passthrough: calls c.Next() unconditionally. Used when
//     KB32_PERMISSIONS_ENFORCED is unset/false.
//   - mw != nil  → runs mw.Wrap against a flag-setting http.Handler. If the
//     middleware allows the request, the flag flips true and Gin's chain
//     continues. If denied, the middleware has already written status + body
//     onto c.Writer, so we c.Abort() to short-circuit Gin's chain.
//
// resource identifies the substrate resource type (e.g. "kb32_craft_draft").
// class is the Self-Visibility classification (PDP for craft writes; see
// package doc for rationale).
func GinPermMW(mw *permissions.Middleware, resource string, class permissions.VisibilityClass) gin.HandlerFunc {
	if mw == nil {
		// Passthrough — kept as a no-op closure so callers can mount the
		// returned HandlerFunc unconditionally.
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		// allowed flips to true only if permissions.Middleware decides to
		// invoke its `next` handler. On any failure path (no viewer role,
		// bad subject_id, no ViewPermission, missing consent) the middleware
		// writes the 4xx response itself and never calls next, so allowed
		// stays false.
		var allowed bool
		passthrough := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			allowed = true
		})
		wrapped := mw.Wrap(resource, class, passthrough)
		wrapped.ServeHTTP(c.Writer, c.Request)
		if !allowed {
			// Response already written by the permissions middleware.
			c.Abort()
			return
		}
		c.Next()
	}
}
