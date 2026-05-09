// Package api wires the pharmacist self-visibility HTTP layer.
//
// This file implements MountDashboardRoutes: six dashboard endpoints for the
// "/v1/views/pharmacist/own/*" family, each gated by the permissions middleware
// at the appropriate VisibilityClass.
//
// Route ownership: all routes in this file carry the "/own/" path segment,
// which is the contract that the data subject (viewer) is reading their OWN
// data. A defensive self-identity check (subject_id == viewerRole) reinforces
// this semantic at the handler layer.
//
// Future endpoints that aggregate across subjects — e.g.
// /v1/views/employer/aggregate/... — will NOT carry this check.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/cardiofit/pharmacist-self-visibility/internal/dashboards"
	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// DashboardDeps holds references to the six dashboard surfaces.
//
// Any nil pointer is handled gracefully: the corresponding handler returns
// 503 Service Unavailable with code "dependency_unavailable". This allows
// main.go to use a zero-value DashboardDeps while routes are mounted (so
// paths are registered and health-check tooling sees them), with concrete
// sources injected in Task 4 (Postgres-backed).
type DashboardDeps struct {
	Worklist        *dashboards.Worklist
	Recommendations *dashboards.MyRecommendations
	GPRelationships *dashboards.GPRelationships
	Reasoning       *dashboards.Reasoning
	CPD             *dashboards.CPD
	Portfolio       *dashboards.Portfolio
}

// MountDashboardRoutes registers all six pharmacist self-visibility dashboard
// endpoints on r, each wrapped with mw.Wrap() at the appropriate VisibilityClass.
//
// All routes are mounted under /v1/views/pharmacist/own/. The "own" segment
// encodes the semantic that the viewer is always reading their own data; a
// defensive subject_id == viewer check is enforced in every handler.
//
// VisibilityClass mapping per Self-Visibility Guidelines §2.1:
//
//	worklist        → WO  (workflow-operational; employer may see compliance status)
//	recommendations → PDP (pharmacist-default-private)
//	gp-relationships→ PDP (pharmacist-default-private)
//	reasoning       → PFA (pharmacist-first-then-aggregated)
//	cpd             → WO  (employer may see compliance status)
//	portfolio       → PDP (pharmacist-controlled; default anonymised)
func MountDashboardRoutes(r chi.Router, mw *permissions.Middleware, d DashboardDeps) {
	r.Route("/v1/views/pharmacist/own", func(r chi.Router) {
		r.Method("GET", "/worklist",
			mw.Wrap("worklist", permissions.WO,
				http.HandlerFunc(d.handleWorklist)))

		r.Method("GET", "/recommendations",
			mw.Wrap("recommendations", permissions.PDP,
				http.HandlerFunc(d.handleRecommendations)))

		r.Method("GET", "/gp-relationships",
			mw.Wrap("gp_relationships", permissions.PDP,
				http.HandlerFunc(d.handleGPRelationships)))

		r.Method("GET", "/reasoning",
			mw.Wrap("reasoning", permissions.PFA,
				http.HandlerFunc(d.handleReasoning)))

		r.Method("GET", "/cpd",
			mw.Wrap("cpd", permissions.WO,
				http.HandlerFunc(d.handleCPD)))

		r.Method("GET", "/portfolio",
			mw.Wrap("portfolio", permissions.PDP,
				http.HandlerFunc(d.handlePortfolio)))
	})
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// handleWorklist serves GET /v1/views/pharmacist/own/worklist (WO).
func (d DashboardDeps) handleWorklist(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)
	if d.Worklist == nil {
		WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "worklist source not wired")
		return
	}
	subjectID, ok := parseSubjectID(w, r)
	if !ok {
		return
	}
	if denied := assertSelfView(w, r, subjectID); denied {
		return
	}
	items, err := d.Worklist.Today(r.Context(), subjectID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "worklist_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, items)
}

// handleRecommendations serves GET /v1/views/pharmacist/own/recommendations (PDP).
func (d DashboardDeps) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)
	if d.Recommendations == nil {
		WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "recommendations source not wired")
		return
	}
	subjectID, ok := parseSubjectID(w, r)
	if !ok {
		return
	}
	if denied := assertSelfView(w, r, subjectID); denied {
		return
	}
	cards, err := d.Recommendations.For(r.Context(), subjectID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "recommendations_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, cards)
}

// handleGPRelationships serves GET /v1/views/pharmacist/own/gp-relationships (PDP).
func (d DashboardDeps) handleGPRelationships(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)
	if d.GPRelationships == nil {
		WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "gp-relationships source not wired")
		return
	}
	subjectID, ok := parseSubjectID(w, r)
	if !ok {
		return
	}
	if denied := assertSelfView(w, r, subjectID); denied {
		return
	}
	cards, err := d.GPRelationships.For(r.Context(), subjectID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "gp_relationships_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, cards)
}

// handleReasoning serves GET /v1/views/pharmacist/own/reasoning (PFA).
func (d DashboardDeps) handleReasoning(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)
	if d.Reasoning == nil {
		WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "reasoning source not wired")
		return
	}
	subjectID, ok := parseSubjectID(w, r)
	if !ok {
		return
	}
	if denied := assertSelfView(w, r, subjectID); denied {
		return
	}
	view, err := d.Reasoning.For(r.Context(), subjectID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "reasoning_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, view)
}

// handleCPD serves GET /v1/views/pharmacist/own/cpd (WO).
func (d DashboardDeps) handleCPD(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)
	if d.CPD == nil {
		WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "cpd source not wired")
		return
	}
	subjectID, ok := parseSubjectID(w, r)
	if !ok {
		return
	}
	if denied := assertSelfView(w, r, subjectID); denied {
		return
	}
	view, err := d.CPD.For(r.Context(), subjectID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "cpd_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, view)
}

// handlePortfolio serves GET /v1/views/pharmacist/own/portfolio (PDP).
//
// The ?identifiable= query parameter controls whether PII in the narrative is
// redacted. Strict parsing: only the literal string "true" enables identifiable
// output. Any other value — including "1", "yes", or a missing parameter —
// defaults to false (anonymised). This is intentionally conservative: the
// pharmacist must explicitly opt in to identifiable output.
func (d DashboardDeps) handlePortfolio(w http.ResponseWriter, r *http.Request) {
	setCacheHeaders(w)
	if d.Portfolio == nil {
		WriteError(w, http.StatusServiceUnavailable, "dependency_unavailable", "portfolio source not wired")
		return
	}
	subjectID, ok := parseSubjectID(w, r)
	if !ok {
		return
	}
	if denied := assertSelfView(w, r, subjectID); denied {
		return
	}

	// Strict identifiable parsing: only "true" → true; everything else → false.
	// "1", "yes", absent, or any other value produces anonymised output.
	identifiable := r.URL.Query().Get("identifiable") == "true"

	view, err := d.Portfolio.For(r.Context(), subjectID, identifiable)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "portfolio_failed", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, view)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseSubjectID reads the "subject_id" query parameter and parses it as a
// UUID. On failure it writes a 400 Bad Request and returns (zero, false); the
// caller must return immediately when ok is false.
func parseSubjectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := r.URL.Query().Get("subject_id")
	id, err := uuid.Parse(raw)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "bad_subject_id", "subject_id query param must be a valid UUID")
		return uuid.UUID{}, false
	}
	return id, true
}

// assertSelfView enforces the "/own/" contract: the JWT viewer identity must
// match the requested subject_id. If they differ, it writes 403 with code
// "not_self_view" and returns true (denied).
//
// Note: the permissions middleware has already run at this point and has
// already applied the primary access-control check (ViewPermission + optional
// DataAggregationConsent). This is a secondary, defensive guard that makes the
// "/own/" semantic explicit and guards against any future accidental reuse of
// these handlers on non-"own" routes.
//
// Future endpoints — e.g. /v1/views/employer/aggregate/... — will NOT call this
// helper, as they legitimately operate across subjects.
func assertSelfView(w http.ResponseWriter, r *http.Request, subjectID uuid.UUID) (denied bool) {
	viewerID, ok := permissions.ViewerRoleFrom(r.Context())
	if !ok {
		// Should not happen: JWT middleware guarantees the viewer is in context.
		// Guard defensively rather than panic.
		WriteError(w, http.StatusUnauthorized, "no_viewer_role", "viewer identity missing from context")
		return true
	}
	if viewerID != subjectID {
		WriteError(w, http.StatusForbidden, "not_self_view",
			"subject_id does not match the authenticated viewer; /own/ routes are for self-access only")
		return true
	}
	return false
}

// setCacheHeaders sets Cache-Control and Pragma headers that prevent any proxy,
// CDN, or browser from caching dashboard responses. Dashboard surfaces contain
// PII — they must never be served from a cache after the user's session ends.
//
// Pragma: no-cache is included for HTTP/1.0 proxy compatibility.
func setCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
}

