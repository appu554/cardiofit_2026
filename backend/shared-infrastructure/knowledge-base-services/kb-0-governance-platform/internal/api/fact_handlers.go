// Package api provides HTTP handlers for the KB-0 Governance Platform API.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"kb-0-governance-platform/internal/database"
	"kb-0-governance-platform/internal/governance"
	"kb-0-governance-platform/internal/policy"
)

// =============================================================================
// FACT GOVERNANCE SERVER
// =============================================================================
// Phase 2 handlers for clinical facts governance from the Canonical Fact Store.
// These endpoints are consumed by the Angular UI for pharmacist review workflows.
// =============================================================================

// FactServer handles HTTP requests for clinical fact governance.
type FactServer struct {
	executor  *governance.Executor
	factStore *database.FactStore
	router    *http.ServeMux
}

// NewFactServer creates a new fact governance API server.
func NewFactServer(executor *governance.Executor, factStore *database.FactStore) *FactServer {
	s := &FactServer{
		executor:  executor,
		factStore: factStore,
		router:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *FactServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers for Angular frontend
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-ID, X-Reviewer-ID")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.router.ServeHTTP(w, r)
}

func (s *FactServer) registerRoutes() {
	// Health check
	s.router.HandleFunc("GET /health", s.handleHealth)

	// ==========================================================================
	// QUEUE ENDPOINTS (For pharmacist review dashboard)
	// ==========================================================================
	s.router.HandleFunc("GET /api/v2/governance/queue", s.handleGetQueue)
	s.router.HandleFunc("GET /api/v2/governance/queue/priority/{priority}", s.handleGetQueueByPriority)
	s.router.HandleFunc("GET /api/v2/governance/queue/reviewer/{reviewer_id}", s.handleGetReviewerQueue)

	// ==========================================================================
	// FACT OPERATIONS (For fact listing and detail view)
	// ==========================================================================
	s.router.HandleFunc("GET /api/v2/governance/facts", s.handleGetAllFacts)
	s.router.HandleFunc("GET /api/v2/governance/facts/{fact_id}", s.handleGetFact)
	s.router.HandleFunc("GET /api/v2/governance/facts/{fact_id}/conflicts", s.handleGetFactConflicts)
	s.router.HandleFunc("GET /api/v2/governance/facts/{fact_id}/history", s.handleGetFactHistory)

	// ==========================================================================
	// CONFLICT RESOLUTION
	// ==========================================================================
	s.router.HandleFunc("GET /api/v2/governance/conflicts", s.handleGetAllConflicts)

	// ==========================================================================
	// AUDIT LOG
	// ==========================================================================
	s.router.HandleFunc("GET /api/v2/governance/audit", s.handleGetAuditLog)

	// ==========================================================================
	// REVIEW ACTIONS (For pharmacist review workflow)
	// ==========================================================================
	s.router.HandleFunc("POST /api/v2/governance/facts/{fact_id}/approve", s.handleApproveFact)
	s.router.HandleFunc("POST /api/v2/governance/facts/{fact_id}/reject", s.handleRejectFact)
	s.router.HandleFunc("POST /api/v2/governance/facts/{fact_id}/escalate", s.handleEscalateFact)
	s.router.HandleFunc("POST /api/v2/governance/facts/{fact_id}/assign", s.handleAssignReviewer)

	// ==========================================================================
	// METRICS AND DASHBOARD
	// ==========================================================================
	s.router.HandleFunc("GET /api/v2/governance/metrics", s.handleGetMetrics)
	s.router.HandleFunc("GET /api/v2/governance/dashboard", s.handleGetDashboard)

	// ==========================================================================
	// EXECUTOR CONTROL (For admin)
	// ==========================================================================
	s.router.HandleFunc("POST /api/v2/governance/executor/start", s.handleStartExecutor)
	s.router.HandleFunc("POST /api/v2/governance/executor/stop", s.handleStopExecutor)
	s.router.HandleFunc("GET /api/v2/governance/executor/status", s.handleExecutorStatus)
	s.router.HandleFunc("POST /api/v2/governance/executor/process/{fact_id}", s.handleProcessFact)
}

// =============================================================================
// HEALTH
// =============================================================================

func (s *FactServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"service":   "kb-0-governance-platform-v2",
		"phase":     "Phase 2 - Canonical Fact Store Governance",
		"timestamp": time.Now().Format(time.RFC3339),
		"executor":  s.executor.IsRunning(),
	})
}

// =============================================================================
// QUEUE HANDLERS
// =============================================================================

// GetQueueResponse represents the queue listing response.
type GetQueueResponse struct {
	Items      []*policy.QueueItem `json:"items"`
	TotalCount int                 `json:"totalCount"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
}

func (s *FactServer) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	// Parse pagination
	limit := 500
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}
	if psStr := r.URL.Query().Get("pageSize"); psStr != "" {
		if ps, err := strconv.Atoi(psStr); err == nil && ps > 0 && ps <= 1000 {
			limit = ps
		}
	}

	items, err := s.executor.GetReviewQueue(r.Context(), limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, GetQueueResponse{
		Items:      items,
		TotalCount: len(items),
		Page:       1,
		PageSize:   limit,
	})
}

func (s *FactServer) handleGetQueueByPriority(w http.ResponseWriter, r *http.Request) {
	priorityStr := r.PathValue("priority")
	if priorityStr == "" {
		respondError(w, http.StatusBadRequest, "Priority required")
		return
	}

	priority := policy.ReviewPriority(priorityStr)

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	items, err := s.factStore.GetQueueByPriority(r.Context(), priority, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, GetQueueResponse{
		Items:      items,
		TotalCount: len(items),
		Page:       1,
		PageSize:   limit,
	})
}

func (s *FactServer) handleGetReviewerQueue(w http.ResponseWriter, r *http.Request) {
	reviewerID := r.PathValue("reviewer_id")
	if reviewerID == "" {
		respondError(w, http.StatusBadRequest, "Reviewer ID required")
		return
	}

	items, err := s.executor.GetReviewerQueue(r.Context(), reviewerID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, GetQueueResponse{
		Items:      items,
		TotalCount: len(items),
		Page:       1,
		PageSize:   len(items),
	})
}

// =============================================================================
// FACT HANDLERS
// =============================================================================

func (s *FactServer) handleGetFact(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	fact, err := s.factStore.GetFact(r.Context(), factID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Fact not found")
		return
	}

	respondJSON(w, http.StatusOK, fact)
}

func (s *FactServer) handleGetFactConflicts(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	fact, err := s.factStore.GetFact(r.Context(), factID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Fact not found")
		return
	}

	// Get conflicting facts
	var conflicts []*policy.ClinicalFact
	for _, conflictID := range fact.ConflictWithFactIDs {
		conflictFact, err := s.factStore.GetFact(r.Context(), conflictID)
		if err == nil {
			conflicts = append(conflicts, conflictFact)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"factId":     factID,
		"hasConflict": fact.HasConflict,
		"conflicts":  conflicts,
	})
}

// GetFactHistoryResponse represents the fact-specific audit history response.
type GetFactHistoryResponse struct {
	FactID  string                  `json:"factId"`
	Items   []*policy.AuditLogEntry `json:"items"`
	Total   int                     `json:"total"`
}

func (s *FactServer) handleGetFactHistory(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	// Get fact-specific audit history (21 CFR Part 11 compliant audit trail)
	history, err := s.factStore.GetFactHistory(r.Context(), factID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, GetFactHistoryResponse{
		FactID: factIDStr,
		Items:  history,
		Total:  len(history),
	})
}

// =============================================================================
// LISTING HANDLERS (For facts, conflicts, and audit pages)
// =============================================================================

// GetAllFactsResponse represents the paginated facts listing response.
type GetAllFactsResponse struct {
	Items    []*policy.ClinicalFact `json:"items"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
	HasMore  bool                   `json:"hasMore"`
}

func (s *FactServer) handleGetAllFacts(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	status := r.URL.Query().Get("status")
	factType := r.URL.Query().Get("factType")
	search := r.URL.Query().Get("search")

	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	facts, total, err := s.factStore.GetAllFacts(r.Context(), status, factType, search, page, pageSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, GetAllFactsResponse{
		Items:    facts,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  page*pageSize < total,
	})
}

// GetAllConflictsResponse represents the conflicts listing response.
type GetAllConflictsResponse struct {
	Groups []*policy.ConflictGroup `json:"groups"`
	Total  int                     `json:"total"`
}

func (s *FactServer) handleGetAllConflicts(w http.ResponseWriter, r *http.Request) {
	groups, err := s.factStore.GetAllConflictGroups(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, GetAllConflictsResponse{
		Groups: groups,
		Total:  len(groups),
	})
}

// GetAuditLogResponse represents the paginated audit log response.
type GetAuditLogResponse struct {
	Items    []*policy.AuditLogEntry `json:"items"`
	Total    int                     `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"pageSize"`
	HasMore  bool                    `json:"hasMore"`
}

func (s *FactServer) handleGetAuditLog(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	eventType := r.URL.Query().Get("eventType")
	actorID := r.URL.Query().Get("actorId")
	fromDate := r.URL.Query().Get("fromDate")
	toDate := r.URL.Query().Get("toDate")

	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 50
	if pageSizeStr := r.URL.Query().Get("pageSize"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 200 {
			pageSize = ps
		}
	}

	entries, total, err := s.factStore.GetAuditLog(r.Context(), eventType, actorID, fromDate, toDate, page, pageSize)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, GetAuditLogResponse{
		Items:    entries,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  page*pageSize < total,
	})
}

// =============================================================================
// REVIEW ACTION HANDLERS
// =============================================================================

// FactReviewRequest represents a review action from the UI.
type FactReviewRequest struct {
	ReviewerID       string `json:"reviewerId"`
	ReviewerName     string `json:"reviewerName"`
	Credentials      string `json:"credentials,omitempty"`
	Reason           string `json:"reason"`
	EscalateToRole   string `json:"escalateToRole,omitempty"`   // For escalation
	EscalateToUserID string `json:"escalateToUserId,omitempty"` // For escalation
}

func (s *FactServer) handleApproveFact(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	var req FactReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "Reviewer ID required")
		return
	}

	result, err := s.executor.ApproveReview(r.Context(), &policy.ReviewRequest{
		FactID:       factID,
		Decision:     policy.DecisionApproved,
		Reason:       req.Reason,
		ReviewerID:   req.ReviewerID,
		ReviewerName: req.ReviewerName,
		Credentials:  req.Credentials,
		IPAddress:    r.RemoteAddr,
		SessionID:    r.Header.Get("X-Session-ID"),
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *FactServer) handleRejectFact(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	var req FactReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "Reviewer ID required")
		return
	}
	if req.Reason == "" {
		respondError(w, http.StatusBadRequest, "Reason required for rejection")
		return
	}

	result, err := s.executor.RejectReview(r.Context(), &policy.ReviewRequest{
		FactID:       factID,
		Decision:     policy.DecisionRejected,
		Reason:       req.Reason,
		ReviewerID:   req.ReviewerID,
		ReviewerName: req.ReviewerName,
		Credentials:  req.Credentials,
		IPAddress:    r.RemoteAddr,
		SessionID:    r.Header.Get("X-Session-ID"),
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (s *FactServer) handleEscalateFact(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	var req FactReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	escalateTo := req.EscalateToUserID
	if escalateTo == "" {
		escalateTo = req.EscalateToRole // Can escalate to role if no specific user
	}
	if escalateTo == "" {
		escalateTo = "cmo" // Default escalation target
	}

	result, err := s.executor.EscalateReview(r.Context(), &policy.ReviewRequest{
		FactID:       factID,
		Decision:     policy.DecisionEscalated,
		Reason:       req.Reason,
		ReviewerID:   req.ReviewerID,
		ReviewerName: req.ReviewerName,
		Credentials:  req.Credentials,
		IPAddress:    r.RemoteAddr,
		SessionID:    r.Header.Get("X-Session-ID"),
	}, escalateTo)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// AssignReviewerRequest represents a reviewer assignment request.
type AssignReviewerRequest struct {
	ReviewerID string `json:"reviewerId"`
	Priority   string `json:"priority,omitempty"`
}

func (s *FactServer) handleAssignReviewer(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	var req AssignReviewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.ReviewerID == "" {
		respondError(w, http.StatusBadRequest, "Reviewer ID required")
		return
	}

	priority := policy.ReviewPriorityStandard
	if req.Priority != "" {
		priority = policy.ReviewPriority(req.Priority)
	}

	err = s.factStore.AssignReviewer(r.Context(), factID, req.ReviewerID, priority)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"factId":     factID,
		"assignedTo": req.ReviewerID,
		"priority":   priority,
	})
}

// =============================================================================
// METRICS AND DASHBOARD HANDLERS
// =============================================================================

func (s *FactServer) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := s.executor.GetQueueMetrics(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, metrics)
}

// DashboardResponse contains the full governance dashboard data.
type DashboardResponse struct {
	Metrics       *database.FactMetrics   `json:"metrics"`
	CriticalQueue []*policy.QueueItem     `json:"criticalQueue"`
	RecentItems   []*policy.QueueItem     `json:"recentItems"`
	SLAAtRisk     []*policy.QueueItem     `json:"slaAtRisk"`
	ExecutorState bool                    `json:"executorRunning"`
	GeneratedAt   time.Time               `json:"generatedAt"`
}

func (s *FactServer) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get metrics
	metrics, err := s.executor.GetQueueMetrics(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get critical queue
	critical, _ := s.factStore.GetQueueByPriority(ctx, policy.ReviewPriorityCritical, 10)

	// Get recent items (full queue, limited)
	recent, _ := s.executor.GetReviewQueue(ctx, 20)

	// Get SLA at-risk items
	// TODO: Add filter for at-risk SLA status

	respondJSON(w, http.StatusOK, DashboardResponse{
		Metrics:       metrics,
		CriticalQueue: critical,
		RecentItems:   recent,
		SLAAtRisk:     nil, // TODO: Implement
		ExecutorState: s.executor.IsRunning(),
		GeneratedAt:   time.Now(),
	})
}

// =============================================================================
// EXECUTOR CONTROL HANDLERS
// =============================================================================

func (s *FactServer) handleStartExecutor(w http.ResponseWriter, r *http.Request) {
	if err := s.executor.Start(r.Context()); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Governance executor started",
		"running": s.executor.IsRunning(),
	})
}

func (s *FactServer) handleStopExecutor(w http.ResponseWriter, r *http.Request) {
	s.executor.Stop()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Governance executor stopped",
		"running": s.executor.IsRunning(),
	})
}

func (s *FactServer) handleExecutorStatus(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"running": s.executor.IsRunning(),
	})
}

func (s *FactServer) handleProcessFact(w http.ResponseWriter, r *http.Request) {
	factIDStr := r.PathValue("fact_id")
	if factIDStr == "" {
		respondError(w, http.StatusBadRequest, "Fact ID required")
		return
	}

	factID, err := uuid.Parse(factIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid fact ID format")
		return
	}

	if err := s.executor.ProcessFact(r.Context(), factID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"factId":  factID,
		"message": "Fact processed",
	})
}
