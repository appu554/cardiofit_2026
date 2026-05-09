package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// MountDashboardRoutes registers all pharmacist self-visibility dashboard
// endpoints on r, gated by the supplied permissions middleware.
//
// STUB (Phase 1b Task 1): All six dashboard endpoints return 501 Not
// Implemented. Task 3 will replace each stub with a real handler.
func MountDashboardRoutes(r chi.Router, mw *permissions.Middleware) {
	_ = mw // used in Task 3 when handlers call mw.Wrap(...)

	stub := func(w http.ResponseWriter, _ *http.Request) {
		WriteError(w, http.StatusNotImplemented, "not_implemented",
			"this dashboard endpoint is not yet implemented")
	}

	r.Route("/views/pharmacist/own", func(r chi.Router) {
		r.Get("/dashboards", stub)
		r.Get("/kpis", stub)
		r.Get("/reflection", stub)
		r.Get("/algorithmic-distinction", stub)
		r.Get("/exports/portfolio", stub)
		r.Get("/portability", stub)
	})
}
