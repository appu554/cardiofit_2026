// Package api hosts the kb-30 HTTP surface.
//
// Endpoints:
//   GET  /health                       - liveness (not wrapped)
//   POST /v1/authorise                 - runtime authorisation evaluation (POST — not wrapped)
//   GET  /v1/audit/resident/:id        - audit query 1 (Q1) — wrapped, VisibilityClass AD
//   GET  /v1/audit/credential/:id      - audit query 2 (Q2) — wrapped, VisibilityClass AD
//   GET  /v1/audit/jurisdiction/:juri  - audit query 3 (Q3) — wrapped, VisibilityClass AD
//   GET  /v1/audit/authorisation/:id/chain - audit query 4 (Q4) — wrapped, VisibilityClass AD
//   GET  /metrics                      - Prometheus metrics (not wrapped)
//
// # Permission-wiring registry
//
// PermissionWiredRoutes lists every GET route that is guarded by the
// permissions middleware. Unwrapped routes are documented with reasons in
// NotWrappedRoutes below.  The registry is exported so that audit/discovery
// tooling (and tests) can enumerate guarded routes without parsing source.
//
// # Default-OFF enforcement flag
//
// Production deployments set KB30_PERMISSIONS_ENFORCED=true.  When the env var
// is absent or "false", Routes() installs a passthrough middleware so existing
// CI tests — which do not carry JWT viewer-role headers — continue to pass
// unmodified.  main.go emits a clear startup warning when enforcement is OFF.
//
// gRPC is intentionally a stub at this MVP layer; the REST surface exposes
// every capability the integration tests need.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/cardiofit/shared/v2_substrate/permissions"

	"kb-authorisation-evaluator/internal/audit"
	"kb-authorisation-evaluator/internal/cache"
	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
	"kb-authorisation-evaluator/internal/metrics"
)

// ---------------------------------------------------------------------------
// Permission-wiring registry
// ---------------------------------------------------------------------------

// RouteDescriptor describes a single HTTP route's permission properties.
type RouteDescriptor struct {
	Method   string
	Path     string
	Resource string
	Class    permissions.VisibilityClass
}

// PermissionWiredRoutes enumerates every GET route that is guarded by the
// permissions middleware. Use this registry for discovery, audit, and tests.
var PermissionWiredRoutes = []RouteDescriptor{
	// Audit-trail reads are Audit-Defensible (AD): the ViewPermission alone
	// is sufficient; no DataAggregationConsent is required. AD is appropriate
	// because these records exist for regulatory/compliance purposes and
	// represent operational system events rather than pharmacist personal data.
	// Self-Visibility Guidelines §2.1 classifies audit event streams as AD.
	{Method: http.MethodGet, Path: "/v1/audit/resident/", Resource: "audit_resident", Class: permissions.AD},
	{Method: http.MethodGet, Path: "/v1/audit/credential/", Resource: "audit_credential", Class: permissions.AD},
	{Method: http.MethodGet, Path: "/v1/audit/jurisdiction/", Resource: "audit_jurisdiction", Class: permissions.AD},
	{Method: http.MethodGet, Path: "/v1/audit/authorisation/", Resource: "audit_authorisation_chain", Class: permissions.AD},
}

// NotWrappedRoutes documents every route that is intentionally NOT wrapped
// by the permissions middleware, with a rationale for each exclusion.
var NotWrappedRoutes = []struct {
	Method  string
	Path    string
	Reason  string
}{
	{http.MethodGet, "/health", "liveness probe — must always respond, no auth context available"},
	{http.MethodPost, "/v1/authorise", "POST write — permissions middleware is GET-read-only by design (Task 5 scope)"},
	{http.MethodGet, "/metrics", "Prometheus scrape endpoint — network-level controls only, no user identity"},
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

// Server bundles the runtime evaluator + caches + audit query API.
type Server struct {
	Evaluator   *evaluator.Evaluator
	Cache       cache.Cache
	Audit       *audit.Service
	// PermMW is the permissions middleware. When nil (KB30_PERMISSIONS_ENFORCED
	// is false/unset), Routes() installs a passthrough wrapper so existing
	// tests pass without JWT tokens or permission records.
	PermMW      *permissions.Middleware
}

// AuthoriseRequest is the wire format for POST /v1/authorise.
type AuthoriseRequest struct {
	Jurisdiction       string    `json:"jurisdiction"`
	Role               string    `json:"role"`
	ActionClass        string    `json:"action_class"`
	MedicationSchedule string    `json:"medication_schedule,omitempty"`
	MedicationClass    string    `json:"medication_class,omitempty"`
	ResidentRef        string    `json:"resident_ref,omitempty"`
	ActorRef           string    `json:"actor_ref,omitempty"`
	ActionDate         time.Time `json:"action_date"`
}

// AuthoriseResponse mirrors evaluator.Result.
type AuthoriseResponse struct {
	evaluator.Result
	CacheHit bool `json:"cache_hit"`
}

// ---------------------------------------------------------------------------
// Middleware helpers
// ---------------------------------------------------------------------------

// wrapRead returns next wrapped with the permission middleware for the
// given resource type and VisibilityClass.  If s.PermMW is nil (passthrough
// mode), next is returned unchanged so legacy tests continue to work.
func (s *Server) wrapRead(resource string, class permissions.VisibilityClass, next http.Handler) http.Handler {
	if s.PermMW == nil {
		// Passthrough mode: KB30_PERMISSIONS_ENFORCED=false (or unset).
		return next
	}
	return s.PermMW.Wrap(resource, class, next)
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

// Routes returns an *http.ServeMux with all handlers wired.
//
// Read audit endpoints are wrapped with s.PermMW (VisibilityClass AD) when
// KB30_PERMISSIONS_ENFORCED=true; otherwise a passthrough is used.
// POST /v1/authorise, GET /health, and GET /metrics are never wrapped.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// --- Not wrapped ---
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/authorise", s.handleAuthorise)
	mux.Handle("/metrics", promhttp.Handler())

	// --- Wrapped GET routes (VisibilityClass AD) ---
	//
	// Audit-trail reads (Q1–Q4) are classified AD (Audit-Defensible) per
	// Self-Visibility Guidelines §2.1.  The ViewPermission alone is sufficient;
	// no DataAggregationConsent is required for AD-class resources.

	// Q1 — audit events by resident UUID
	// Resource: "audit_resident" / Class: AD
	mux.Handle("/v1/audit/resident/",
		s.wrapRead("audit_resident", permissions.AD,
			http.HandlerFunc(s.handleAuditResident)))

	// Q2 — audit events by credential UUID
	// Resource: "audit_credential" / Class: AD
	mux.Handle("/v1/audit/credential/",
		s.wrapRead("audit_credential", permissions.AD,
			http.HandlerFunc(s.handleAuditCredential)))

	// Q3 — audit events by jurisdiction + medication schedule
	// Resource: "audit_jurisdiction" / Class: AD
	mux.Handle("/v1/audit/jurisdiction/",
		s.wrapRead("audit_jurisdiction", permissions.AD,
			http.HandlerFunc(s.handleAuditJurisdiction)))

	// Q4 — authorisation chain for a single evaluation ID
	// Resource: "audit_authorisation_chain" / Class: AD
	mux.Handle("/v1/audit/authorisation/",
		s.wrapRead("audit_authorisation_chain", permissions.AD,
			http.HandlerFunc(s.handleAuditChain)))

	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "kb-30-authorisation-evaluator"})
}

func (s *Server) handleAuthorise(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	outcome := metrics.OutcomeError // default; updated once we have a result
	defer func() {
		metrics.ObserveEvaluation(outcome, time.Since(start).Seconds())
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AuthoriseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	q := evaluator.Query{
		Jurisdiction:       req.Jurisdiction,
		Role:               req.Role,
		ActionClass:        dsl.ActionClass(req.ActionClass),
		MedicationSchedule: req.MedicationSchedule,
		MedicationClass:    req.MedicationClass,
		ActionDate:         req.ActionDate,
	}
	if req.ResidentRef != "" {
		if id, err := uuid.Parse(req.ResidentRef); err == nil {
			q.ResidentRef = id
		}
	}
	if req.ActorRef != "" {
		if id, err := uuid.Parse(req.ActorRef); err == nil {
			q.ActorRef = id
		}
	}

	// Cache lookup.
	if s.Cache != nil {
		if cached, ok, _ := s.Cache.Get(r.Context(), q.CacheKey()); ok && cached != nil {
			outcome = decisionToOutcome(cached.Decision)
			resp := AuthoriseResponse{Result: *cached, CacheHit: true}
			writeJSON(w, http.StatusOK, resp)
			return
		}
	}

	res, err := s.Evaluator.Evaluate(r.Context(), q)
	if err != nil {
		http.Error(w, "evaluation failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	outcome = decisionToOutcome(res.Decision)
	if s.Cache != nil {
		ttl := cache.DefaultTTL(res)
		_ = s.Cache.Set(r.Context(), q.CacheKey(), &res, ttl)

		// Record on EvidenceTrace via audit service for regulator queries.
		if s.Audit != nil {
			s.Audit.Record(audit.EvaluationRecord{
				ID:          uuid.New(),
				Query:       q,
				Result:      res,
				EvaluatedAt: res.EvaluatedAt,
			})
		}
	} else if s.Audit != nil {
		s.Audit.Record(audit.EvaluationRecord{
			ID:          uuid.New(),
			Query:       q,
			Result:      res,
			EvaluatedAt: res.EvaluatedAt,
		})
	}

	writeJSON(w, http.StatusOK, AuthoriseResponse{Result: res, CacheHit: false})
}

// decisionToOutcome maps a dsl.Decision to the metrics outcome label.
// Granted (with or without conditions) is reported as "allow"; Denied as
// "deny"; anything else falls back to "error".
func decisionToOutcome(d dsl.Decision) string {
	switch d {
	case dsl.DecisionGranted, dsl.DecisionGrantedWithConditions:
		return metrics.OutcomeAllow
	case dsl.DecisionDenied:
		return metrics.OutcomeDeny
	default:
		return metrics.OutcomeError
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
