// Package api hosts the kb-31-scope-rules HTTP surface.
//
// Endpoints:
//   GET  /health                       liveness
//   GET  /v1/scope-rules               jurisdiction-aware listing
//                                      (?jurisdiction=AU/VIC&at=ISO8601)
//   GET  /v1/scope-rules/{id}          fetch single rule (id = UUID)
//   POST /v1/scope-rules               accept YAML body, parse, insert
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"kb-scope-rules/internal/dsl"
	"kb-scope-rules/internal/store"
)

// Server bundles a Store and exposes the REST surface.
type Server struct {
	Store store.Store
}

// Routes returns an *http.ServeMux with all handlers wired.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/scope-rules", s.handleScopeRules)
	mux.HandleFunc("/v1/scope-rules/", s.handleScopeRuleByID)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"service": "kb-31-scope-rules",
	})
}

// handleScopeRules dispatches to GET (list) or POST (insert).
func (s *Server) handleScopeRules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listScopeRules(w, r)
	case http.MethodPost:
		s.insertScopeRule(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listScopeRules(w http.ResponseWriter, r *http.Request) {
	jurisdiction := r.URL.Query().Get("jurisdiction")
	if jurisdiction == "" {
		http.Error(w, "jurisdiction query parameter is required", http.StatusBadRequest)
		return
	}
	atTime := time.Now().UTC()
	if atStr := r.URL.Query().Get("at"); atStr != "" {
		t, err := time.Parse(time.RFC3339, atStr)
		if err != nil {
			http.Error(w, "invalid 'at' (expected RFC3339): "+err.Error(), http.StatusBadRequest)
			return
		}
		atTime = t
	}
	includeDraft := r.URL.Query().Get("include_draft") == "true"
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var (
		rules []store.StoredRule
		err   error
	)
	if includeDraft {
		rules, err = s.Store.AllForJurisdiction(ctx, jurisdiction)
	} else {
		rules, err = s.Store.ActiveForJurisdiction(ctx, jurisdiction, atTime)
	}
	if err != nil {
		http.Error(w, "store error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"jurisdiction":  jurisdiction,
		"at":            atTime,
		"include_draft": includeDraft,
		"count":         len(rules),
		"scope_rules":   rules,
	})
}

func (s *Server) insertScopeRule(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	rule, err := dsl.ParseRule(body)
	if err != nil {
		http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	id, err := s.Store.Insert(ctx, *rule, body)
	if err != nil {
		http.Error(w, "insert: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":      id,
		"rule_id": rule.RuleID,
	})
}

func (s *Server) handleScopeRuleByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/v1/scope-rules/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id (expected UUID): "+err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rule, err := s.Store.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "store error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
