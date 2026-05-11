// Permissions middleware adapter for s2-aggregator.
//
// Mirrors the Phase 2-completion Task 7 GinPermMW adapter shipped with
// kb-32:
//   backend/shared-infrastructure/knowledge-base-services/kb-32-recommendation-craft/internal/api/permissions_middleware.go
//
// Rationale: the shared Phase 1a permissions.Middleware is implemented
// over net/http while s2-aggregator uses Gin. GinPermMW bridges the two
// contracts so the (resource, class) gate can guard Gin routes without
// changing the shared middleware package itself.
//
// Local type mirrors:
//
//   - We do NOT import the shared permissions package directly. Tasks
//     3–7 established a pattern of locally-defined boundary interfaces
//     (see audit.Logger, drill_through.ObservationFetcher,
//     actions.OverrideForwarder). We continue that pattern here: this
//     file declares local VisibilityClass / Middleware shapes that the
//     production cmd/server/main.go satisfies by adapting the shared
//     permissions.Middleware.
//
//   - This keeps go.mod free of a shared-package import while preserving
//     wire-shape compatibility (VisibilityClass values map 1:1 to the
//     shared package's iota order).
//
// Passthrough mode:
//
//   - When S2_PERMISSIONS_ENFORCED is unset / false, main.go constructs
//     the adapter with mw == nil. GinPermMW then returns a handler that
//     simply calls c.Next() — equivalent to no middleware. This keeps
//     existing handler tests (which never set the env var and have no
//     JWT/permission setup) green.
//
// Class choice:
//
//   - v1.0 Part 13.3 + Addendum Part 5.2 classify pharmacist S2 audit
//     rows as PDP (Pharmacist-Default-Private). The S2 action routes
//     are pharmacist-private writes on their own clinical workspace
//     activity, so PDP is the correct class. AD (Audit-Defensible) is
//     reserved for regulator/ethics-committee reads; the routes here
//     are NOT exposed to those roles directly.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// VisibilityClass is the local mirror of
// shared/v2_substrate/permissions.VisibilityClass. Values are kept in
// iota order so that an adapter constructed in cmd/server/main.go can
// translate to the shared package's enum 1:1.
type VisibilityClass int

const (
	// PDP — Pharmacist-Default-Private. Owner-only read access.
	PDP VisibilityClass = iota
	// PEV — Pharmacist-Employer-View. Restricted; not used for any
	// route in this file but enumerated for completeness.
	PEV
	// AD — Audit-Defensible. Regulator/ethics-committee access only.
	AD
	// PDF — Pharmacist-Default-Family. Family-visible records.
	PDF
)

// Middleware is the local interface that production wiring satisfies by
// adapting the shared permissions.Middleware. The single Wrap method
// matches the shared package's signature exactly: it takes a (resource,
// class) gate and returns an http.Handler that runs `next` only if the
// gate allows.
//
// Tests provide stub implementations that decide allow/deny based on a
// fixed predicate.
type Middleware interface {
	Wrap(resource string, class VisibilityClass, next http.Handler) http.Handler
}

// GinPermMW returns a gin.HandlerFunc that enforces the (resource,
// class) permissions gate before the next Gin handler runs.
//
// Behaviour:
//
//   - mw == nil → passthrough: c.Next() unconditionally. Used when
//     S2_PERMISSIONS_ENFORCED is unset/false.
//
//   - mw != nil → runs mw.Wrap against a flag-setting http.Handler. If
//     the middleware allows the request, the flag flips true and Gin's
//     chain continues. If denied, the middleware has already written
//     the status + body onto c.Writer, so we c.Abort() to short-circuit
//     the chain. Pattern is identical to kb-32's adapter.
//
// resource is the substrate-resource tag (e.g. "s2_action_override").
// class is the visibility classification — PDP for every action route
// in S2 per v1.0 Part 13.3.
func GinPermMW(mw Middleware, resource string, class VisibilityClass) gin.HandlerFunc {
	if mw == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
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
