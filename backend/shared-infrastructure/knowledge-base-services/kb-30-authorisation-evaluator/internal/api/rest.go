// Package api hosts the kb-30 HTTP surface.
//
// Endpoints:
//   GET  /health                       - liveness
//   POST /v1/authorise                 - runtime authorisation evaluation
//   GET  /v1/audit/resident/:id        - audit query 1 (Q1)
//   GET  /v1/audit/credential/:id      - audit query 2 (Q2)
//   GET  /v1/audit/jurisdiction/:juri  - audit query 3 (Q3)
//   GET  /v1/audit/authorisation/:id/chain - audit query 4 (Q4)
//
// gRPC is intentionally a stub at this MVP layer; the REST surface exposes
// every capability the integration tests need.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"kb-authorisation-evaluator/internal/audit"
	"kb-authorisation-evaluator/internal/cache"
	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

// Server bundles the runtime evaluator + caches + audit query API.
type Server struct {
	Evaluator *evaluator.Evaluator
	Cache     cache.Cache
	Audit     *audit.Service
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

// Routes returns an *http.ServeMux with all handlers wired.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/authorise", s.handleAuthorise)
	mux.HandleFunc("/v1/audit/resident/", s.handleAuditResident)
	mux.HandleFunc("/v1/audit/credential/", s.handleAuditCredential)
	mux.HandleFunc("/v1/audit/jurisdiction/", s.handleAuditJurisdiction)
	mux.HandleFunc("/v1/audit/authorisation/", s.handleAuditChain)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "kb-30-authorisation-evaluator"})
}

func (s *Server) handleAuthorise(w http.ResponseWriter, r *http.Request) {
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
	if s.Cache != nil {
		ttl := cache.DefaultTTL(res)
		_ = s.Cache.Set(r.Context(), q.CacheKey(), &res, ttl)

		// Record on EvidenceTrace via audit service for regulator queries.
		if s.Audit != nil {
			s.Audit.Record(audit.EvaluationRecord{
				ID:                   uuid.New(),
				Query:                q,
				Result:               res,
				EvaluatedAt:          res.EvaluatedAt,
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

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
