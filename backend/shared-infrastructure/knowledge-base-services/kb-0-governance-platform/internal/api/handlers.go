// Package api provides HTTP handlers for the KB-0 Governance Platform API.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"kb-0-governance-platform/internal/audit"
	"kb-0-governance-platform/internal/database"
	"kb-0-governance-platform/internal/models"
	"kb-0-governance-platform/internal/workflow"
	"kb-0-governance-platform/pkg/kb1client"
)

// =============================================================================
// SERVER
// =============================================================================

// Server handles HTTP requests for KB-0 governance.
type Server struct {
	engine   *workflow.Engine
	store    *database.Store
	audit    *audit.Logger
	router   *http.ServeMux
	kb1Store *kb1client.KB1Store // KB-1 integration
}

// NewServer creates a new API server.
func NewServer(engine *workflow.Engine, store *database.Store, auditLogger *audit.Logger) *Server {
	s := &Server{
		engine: engine,
		store:  store,
		audit:  auditLogger,
		router: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// NewServerWithKB1 creates a new API server with KB-1 integration.
func NewServerWithKB1(engine *workflow.Engine, store *database.Store, auditLogger *audit.Logger, kb1URL string) *Server {
	s := &Server{
		engine:   engine,
		store:    store,
		audit:    auditLogger,
		router:   http.NewServeMux(),
		kb1Store: kb1client.NewKB1Store(kb1URL),
	}
	s.registerRoutes()
	s.registerKB1Routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) registerRoutes() {
	// Health check
	s.router.HandleFunc("GET /health", s.handleHealth)

	// Workflow operations
	s.router.HandleFunc("POST /api/v1/workflow/review", s.handleSubmitReview)
	s.router.HandleFunc("POST /api/v1/workflow/approve", s.handleApprove)
	s.router.HandleFunc("POST /api/v1/workflow/reject", s.handleReject)
	s.router.HandleFunc("POST /api/v1/workflow/activate/{id}", s.handleActivate)

	// Item operations
	s.router.HandleFunc("POST /api/v1/items", s.handleCreateItem)
	s.router.HandleFunc("GET /api/v1/items/{id}", s.handleGetItem)
	s.router.HandleFunc("PUT /api/v1/items/{id}", s.handleUpdateItem)

	// Query operations
	s.router.HandleFunc("GET /api/v1/items/pending-review", s.handlePendingReviews)
	s.router.HandleFunc("GET /api/v1/items/pending-approval", s.handlePendingApprovals)
	s.router.HandleFunc("GET /api/v1/items/active", s.handleActiveItems)

	// Metrics
	s.router.HandleFunc("GET /api/v1/metrics/{kb}", s.handleKBMetrics)
	s.router.HandleFunc("GET /api/v1/metrics/all", s.handleCrossKBMetrics)

	// Audit
	s.router.HandleFunc("GET /api/v1/audit/{item_id}", s.handleAuditTrail)
	s.router.HandleFunc("GET /api/v1/audit/export", s.handleAuditExport)
}

// =============================================================================
// HEALTH
// =============================================================================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "kb-0-governance-platform",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// =============================================================================
// WORKFLOW HANDLERS
// =============================================================================

// ReviewRequest represents a review submission.
type ReviewRequest struct {
	ItemID       string                  `json:"item_id"`
	ReviewerID   string                  `json:"reviewer_id"`
	ReviewerName string                  `json:"reviewer_name"`
	ReviewerRole string                  `json:"reviewer_role"`
	Credentials  string                  `json:"credentials,omitempty"`
	Notes        string                  `json:"notes"`
	Checklist    *models.ReviewChecklist `json:"checklist,omitempty"`
}

func (s *Server) handleSubmitReview(w http.ResponseWriter, r *http.Request) {
	var req ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := s.engine.SubmitReview(r.Context(), &workflow.ReviewRequest{
		ItemID:       req.ItemID,
		ReviewerID:   req.ReviewerID,
		ReviewerName: req.ReviewerName,
		ReviewerRole: req.ReviewerRole,
		Credentials:  req.Credentials,
		Notes:        req.Notes,
		Checklist:    req.Checklist,
		IPAddress:    r.RemoteAddr,
		SessionID:    r.Header.Get("X-Session-ID"),
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ApprovalRequest represents an approval/rejection.
type ApprovalRequest struct {
	ItemID       string          `json:"item_id"`
	ApproverID   string          `json:"approver_id"`
	ApproverName string          `json:"approver_name"`
	ApproverRole string          `json:"approver_role"`
	Credentials  string          `json:"credentials,omitempty"`
	Notes        string          `json:"notes"`
	Attestations map[string]bool `json:"attestations,omitempty"`
}

func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := s.engine.Approve(r.Context(), &workflow.ApprovalRequest{
		ItemID:       req.ItemID,
		ApproverID:   req.ApproverID,
		ApproverName: req.ApproverName,
		ApproverRole: req.ApproverRole,
		Credentials:  req.Credentials,
		Notes:        req.Notes,
		Attestations: req.Attestations,
		IPAddress:    r.RemoteAddr,
		SessionID:    r.Header.Get("X-Session-ID"),
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	var req ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	result, err := s.engine.Reject(r.Context(), &workflow.ApprovalRequest{
		ItemID:       req.ItemID,
		ApproverID:   req.ApproverID,
		ApproverName: req.ApproverName,
		ApproverRole: req.ApproverRole,
		Credentials:  req.Credentials,
		Notes:        req.Notes,
		IPAddress:    r.RemoteAddr,
		SessionID:    r.Header.Get("X-Session-ID"),
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleActivate(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")
	if itemID == "" {
		respondError(w, http.StatusBadRequest, "Item ID required")
		return
	}

	result, err := s.engine.Activate(r.Context(), itemID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// =============================================================================
// ITEM HANDLERS
// =============================================================================

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var item models.KnowledgeItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.store.CreateItem(r.Context(), &item); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, item)
}

func (s *Server) handleGetItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")
	if itemID == "" {
		respondError(w, http.StatusBadRequest, "Item ID required")
		return
	}

	item, err := s.store.GetItem(r.Context(), itemID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Item not found")
		return
	}

	respondJSON(w, http.StatusOK, item)
}

func (s *Server) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("id")
	if itemID == "" {
		respondError(w, http.StatusBadRequest, "Item ID required")
		return
	}

	var item models.KnowledgeItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	item.ID = itemID

	if err := s.store.UpdateItem(r.Context(), &item); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// =============================================================================
// QUERY HANDLERS
// =============================================================================

func (s *Server) handlePendingReviews(w http.ResponseWriter, r *http.Request) {
	kb := models.KB(r.URL.Query().Get("kb"))
	if kb == "" {
		respondError(w, http.StatusBadRequest, "KB parameter required")
		return
	}

	items, err := s.engine.GetPendingReviews(r.Context(), kb)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, items)
}

func (s *Server) handlePendingApprovals(w http.ResponseWriter, r *http.Request) {
	kb := models.KB(r.URL.Query().Get("kb"))
	if kb == "" {
		respondError(w, http.StatusBadRequest, "KB parameter required")
		return
	}

	items, err := s.engine.GetPendingApprovals(r.Context(), kb)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, items)
}

func (s *Server) handleActiveItems(w http.ResponseWriter, r *http.Request) {
	kb := models.KB(r.URL.Query().Get("kb"))
	if kb == "" {
		respondError(w, http.StatusBadRequest, "KB parameter required")
		return
	}

	items, err := s.engine.GetActiveItems(r.Context(), kb)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, items)
}

// =============================================================================
// METRICS HANDLERS
// =============================================================================

func (s *Server) handleKBMetrics(w http.ResponseWriter, r *http.Request) {
	kb := models.KB(r.PathValue("kb"))
	if kb == "" {
		respondError(w, http.StatusBadRequest, "KB parameter required")
		return
	}

	metrics, err := s.store.GetMetrics(r.Context(), kb)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

func (s *Server) handleCrossKBMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := s.store.GetCrossKBMetrics(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

// =============================================================================
// AUDIT HANDLERS
// =============================================================================

func (s *Server) handleAuditTrail(w http.ResponseWriter, r *http.Request) {
	itemID := r.PathValue("item_id")
	if itemID == "" {
		respondError(w, http.StatusBadRequest, "Item ID required")
		return
	}

	entries, err := s.audit.GetAuditTrail(r.Context(), itemID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, entries)
}

func (s *Server) handleAuditExport(w http.ResponseWriter, r *http.Request) {
	kb := models.KB(r.URL.Query().Get("kb"))
	sinceStr := r.URL.Query().Get("since")

	if kb == "" {
		respondError(w, http.StatusBadRequest, "KB parameter required")
		return
	}

	// Parse since date (default to 30 days ago)
	since := time.Now().AddDate(0, 0, -30)
	if sinceStr != "" {
		parsed, err := time.Parse("2006-01-02", sinceStr)
		if err == nil {
			since = parsed
		}
	}
	until := time.Now()

	export, err := s.audit.ExportForRegulator(r.Context(), kb, since, until)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, export)
}

// =============================================================================
// HELPERS
// =============================================================================

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// =============================================================================
// KB-1 SPECIFIC ROUTES AND HANDLERS
// =============================================================================

func (s *Server) registerKB1Routes() {
	// KB-1 Drug Rules governance endpoints
	s.router.HandleFunc("GET /api/v1/kb1/pending", s.handleKB1PendingDrugs)
	s.router.HandleFunc("GET /api/v1/kb1/drugs/{rxnorm}", s.handleKB1GetDrug)
	s.router.HandleFunc("POST /api/v1/kb1/drugs/{id}/review", s.handleKB1SubmitReview)
	s.router.HandleFunc("POST /api/v1/kb1/drugs/{id}/approve", s.handleKB1Approve)
}

// handleKB1PendingDrugs returns drugs pending governance review from KB-1
func (s *Server) handleKB1PendingDrugs(w http.ResponseWriter, r *http.Request) {
	if s.kb1Store == nil {
		respondError(w, http.StatusServiceUnavailable, "KB-1 integration not configured")
		return
	}

	pending, err := s.kb1Store.GetPendingDrugs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"count": len(pending),
		"items": pending,
	})
}

// handleKB1GetDrug returns a specific drug rule from KB-1
func (s *Server) handleKB1GetDrug(w http.ResponseWriter, r *http.Request) {
	if s.kb1Store == nil {
		respondError(w, http.StatusServiceUnavailable, "KB-1 integration not configured")
		return
	}

	rxnorm := r.PathValue("rxnorm")
	if rxnorm == "" {
		respondError(w, http.StatusBadRequest, "RxNorm code required")
		return
	}

	rule, err := s.kb1Store.GetDrugRule(r.Context(), rxnorm)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, rule)
}

// KB1ReviewRequest for KB-1 specific review submission
type KB1ReviewRequest struct {
	ReviewedBy  string `json:"reviewed_by"`
	ReviewNotes string `json:"review_notes"`
	Checklist   struct {
		DosingVerified       bool `json:"dosing_verified"`
		RenalVerified        bool `json:"renal_verified"`
		HepaticVerified      bool `json:"hepatic_verified"`
		InteractionsVerified bool `json:"interactions_verified"`
		SafetyVerified       bool `json:"safety_verified"`
	} `json:"checklist"`
}

// handleKB1SubmitReview submits a pharmacist review for a KB-1 drug rule
func (s *Server) handleKB1SubmitReview(w http.ResponseWriter, r *http.Request) {
	if s.kb1Store == nil {
		respondError(w, http.StatusServiceUnavailable, "KB-1 integration not configured")
		return
	}

	ruleID := r.PathValue("id")
	if ruleID == "" {
		respondError(w, http.StatusBadRequest, "Rule ID required")
		return
	}

	var req KB1ReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	checklist := map[string]bool{
		"dosing":       req.Checklist.DosingVerified,
		"renal":        req.Checklist.RenalVerified,
		"hepatic":      req.Checklist.HepaticVerified,
		"interactions": req.Checklist.InteractionsVerified,
		"safety":       req.Checklist.SafetyVerified,
	}

	err := s.kb1Store.SubmitReview(r.Context(), ruleID, req.ReviewedBy, req.ReviewNotes, checklist)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Review submitted successfully",
		"rule_id": ruleID,
		"status":  "REVIEWED",
	})
}

// KB1ApprovalRequest for KB-1 specific approval
type KB1ApprovalRequest struct {
	ApprovedBy  string `json:"approved_by"`
	ReviewNotes string `json:"review_notes"`
	IsHighRisk  bool   `json:"is_high_risk"`
}

// handleKB1Approve submits CMO approval for a KB-1 drug rule
func (s *Server) handleKB1Approve(w http.ResponseWriter, r *http.Request) {
	if s.kb1Store == nil {
		respondError(w, http.StatusServiceUnavailable, "KB-1 integration not configured")
		return
	}

	ruleID := r.PathValue("id")
	if ruleID == "" {
		respondError(w, http.StatusBadRequest, "Rule ID required")
		return
	}

	var req KB1ApprovalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := s.kb1Store.SubmitApproval(r.Context(), ruleID, req.ApprovedBy, req.ReviewNotes, req.IsHighRisk)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Approval submitted successfully - drug rule now ACTIVE",
		"rule_id": ruleID,
		"status":  "ACTIVE",
	})
}
