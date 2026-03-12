// Package api provides governance API handlers for KB-14 Care Navigator
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-14-care-navigator/internal/models"
	"kb-14-care-navigator/internal/services"
)

// GovernanceHandlers handles governance and audit API endpoints
type GovernanceHandlers struct {
	governanceSvc *services.GovernanceService
}

// NewGovernanceHandlers creates new governance handlers
func NewGovernanceHandlers(governanceSvc *services.GovernanceService) *GovernanceHandlers {
	return &GovernanceHandlers{
		governanceSvc: governanceSvc,
	}
}

// RegisterRoutes registers governance routes
func (h *GovernanceHandlers) RegisterRoutes(r *gin.RouterGroup) {
	governance := r.Group("/governance")
	{
		// Audit Trail Endpoints
		governance.GET("/audit/task/:id", h.GetTaskAuditTrail)
		governance.GET("/audit/patient/:id", h.GetPatientAuditTrail)
		governance.GET("/audit/actor/:id", h.GetActorAuditTrail)
		governance.GET("/audit/search", h.SearchAuditLogs)
		governance.GET("/audit/summary/:taskId", h.GetAuditSummary)
		governance.GET("/audit/verify/:taskId", h.VerifyAuditIntegrity)

		// Governance Events Endpoints
		governance.GET("/events", h.ListGovernanceEvents)
		governance.GET("/events/:id", h.GetGovernanceEvent)
		governance.GET("/events/unresolved", h.GetUnresolvedEvents)
		governance.GET("/events/requiring-action", h.GetEventsRequiringAction)
		governance.POST("/events/:id/resolve", h.ResolveGovernanceEvent)

		// Reason Codes Endpoints
		governance.GET("/reason-codes", h.ListReasonCodes)
		governance.GET("/reason-codes/:category", h.GetReasonCodesByCategory)
		governance.GET("/reason-codes/validate/:code", h.ValidateReasonCode)

		// Intelligence Tracking Endpoints
		governance.GET("/intelligence/accountability", h.GetIntelligenceAccountability)
		governance.POST("/intelligence/:id/disposition", h.DispositionIntelligence)

		// Dashboard & Compliance Endpoints
		governance.GET("/dashboard", h.GetGovernanceDashboard)
		governance.GET("/compliance-score", h.GetComplianceScore)
	}
}

// =============================================================================
// AUDIT TRAIL HANDLERS
// =============================================================================

// GetTaskAuditTrail retrieves the complete audit trail for a task
func (h *GovernanceHandlers) GetTaskAuditTrail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid task ID"})
		return
	}

	logs, err := h.governanceSvc.GetTaskAuditTrail(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.AuditLogResponse{
		Success: true,
		Data:    logs,
		Total:   int64(len(logs)),
	})
}

// GetPatientAuditTrail retrieves audit trail for a patient
func (h *GovernanceHandlers) GetPatientAuditTrail(c *gin.Context) {
	patientID := c.Param("id")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Patient ID required"})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	logs, err := h.governanceSvc.GetPatientAuditTrail(c.Request.Context(), patientID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.AuditLogResponse{
		Success: true,
		Data:    logs,
		Total:   int64(len(logs)),
	})
}

// GetActorAuditTrail retrieves audit trail for an actor
func (h *GovernanceHandlers) GetActorAuditTrail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid actor ID"})
		return
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	logs, err := h.governanceSvc.GetActorAuditTrail(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.AuditLogResponse{
		Success: true,
		Data:    logs,
		Total:   int64(len(logs)),
	})
}

// SearchAuditLogs searches audit logs based on query parameters
func (h *GovernanceHandlers) SearchAuditLogs(c *gin.Context) {
	query := &models.AuditLogQuery{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if taskIDStr := c.Query("task_id"); taskIDStr != "" {
		if id, err := uuid.Parse(taskIDStr); err == nil {
			query.TaskID = &id
		}
	}

	if patientID := c.Query("patient_id"); patientID != "" {
		query.PatientID = patientID
	}

	if actorIDStr := c.Query("actor_id"); actorIDStr != "" {
		if id, err := uuid.Parse(actorIDStr); err == nil {
			query.ActorID = &id
		}
	}

	if eventType := c.Query("event_type"); eventType != "" {
		et := models.AuditEventType(eventType)
		query.EventType = &et
	}

	if eventCategory := c.Query("event_category"); eventCategory != "" {
		ec := models.AuditEventCategory(eventCategory)
		query.EventCategory = &ec
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			query.StartDate = &t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			query.EndDate = &t
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = o
		}
	}

	logs, total, err := h.governanceSvc.QueryAuditLogs(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.AuditLogResponse{
		Success: true,
		Data:    logs,
		Total:   total,
	})
}

// GetAuditSummary retrieves summary statistics for a task's audit trail
func (h *GovernanceHandlers) GetAuditSummary(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid task ID"})
		return
	}

	summary, err := h.governanceSvc.GetAuditSummary(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
	})
}

// VerifyAuditIntegrity verifies the hash chain integrity for a task
func (h *GovernanceHandlers) VerifyAuditIntegrity(c *gin.Context) {
	taskIDStr := c.Param("taskId")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid task ID"})
		return
	}

	valid, errors, err := h.governanceSvc.VerifyAuditIntegrity(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"valid":            valid,
			"integrity_errors": errors,
			"verified_at":      time.Now().UTC(),
		},
	})
}

// =============================================================================
// GOVERNANCE EVENTS HANDLERS
// =============================================================================

// ListGovernanceEvents lists governance events with filtering
func (h *GovernanceHandlers) ListGovernanceEvents(c *gin.Context) {
	query := &models.GovernanceEventQuery{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if eventType := c.Query("event_type"); eventType != "" {
		et := models.GovernanceEventType(eventType)
		query.EventType = &et
	}

	if severity := c.Query("severity"); severity != "" {
		s := models.GovernanceSeverity(severity)
		query.Severity = &s
	}

	if taskIDStr := c.Query("task_id"); taskIDStr != "" {
		if id, err := uuid.Parse(taskIDStr); err == nil {
			query.TaskID = &id
		}
	}

	if patientID := c.Query("patient_id"); patientID != "" {
		query.PatientID = patientID
	}

	if resolvedStr := c.Query("resolved"); resolvedStr != "" {
		resolved := resolvedStr == "true"
		query.Resolved = &resolved
	}

	if requiresActionStr := c.Query("requires_action"); requiresActionStr != "" {
		requiresAction := requiresActionStr == "true"
		query.RequiresAction = &requiresAction
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = l
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = o
		}
	}

	events, total, err := h.governanceSvc.QueryGovernanceEvents(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.GovernanceEventResponse{
		Success: true,
		Data:    events,
		Total:   total,
	})
}

// GetGovernanceEvent retrieves a governance event by ID
func (h *GovernanceHandlers) GetGovernanceEvent(c *gin.Context) {
	idStr := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Get governance event: " + idStr,
	})
}

// GetUnresolvedEvents retrieves all unresolved governance events
func (h *GovernanceHandlers) GetUnresolvedEvents(c *gin.Context) {
	events, err := h.governanceSvc.GetUnresolvedGovernanceEvents(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.GovernanceEventResponse{
		Success: true,
		Data:    events,
		Total:   int64(len(events)),
	})
}

// GetEventsRequiringAction retrieves governance events requiring action
func (h *GovernanceHandlers) GetEventsRequiringAction(c *gin.Context) {
	events, err := h.governanceSvc.GetEventsRequiringAction(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.GovernanceEventResponse{
		Success: true,
		Data:    events,
		Total:   int64(len(events)),
	})
}

// ResolveGovernanceEventRequest represents the request body for resolving an event
type ResolveGovernanceEventRequest struct {
	ResolvedBy uuid.UUID `json:"resolved_by" binding:"required"`
	Notes      string    `json:"notes,omitempty"`
}

// ResolveGovernanceEvent resolves a governance event
func (h *GovernanceHandlers) ResolveGovernanceEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid event ID"})
		return
	}

	var req ResolveGovernanceEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.governanceSvc.ResolveGovernanceEvent(c.Request.Context(), id, req.ResolvedBy, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"message":     "Governance event resolved",
		"resolved_at": time.Now().UTC(),
	})
}

// =============================================================================
// REASON CODES HANDLERS
// =============================================================================

// ListReasonCodes lists all active reason codes
func (h *GovernanceHandlers) ListReasonCodes(c *gin.Context) {
	codes, err := h.governanceSvc.GetAllReasonCodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    codes,
		"total":   len(codes),
	})
}

// GetReasonCodesByCategory retrieves reason codes by category
func (h *GovernanceHandlers) GetReasonCodesByCategory(c *gin.Context) {
	category := models.ReasonCodeCategory(c.Param("category"))

	codes, err := h.governanceSvc.GetReasonCodesByCategory(c.Request.Context(), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"category": category,
		"data":     codes,
		"total":    len(codes),
	})
}

// ValidateReasonCode validates a reason code and returns its requirements
func (h *GovernanceHandlers) ValidateReasonCode(c *gin.Context) {
	code := c.Param("code")

	valid, requiresJustification, requiresSupervisor, err := h.governanceSvc.ValidateReasonCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"code":                         code,
			"valid":                        valid,
			"requires_justification":       requiresJustification,
			"requires_supervisor_approval": requiresSupervisor,
		},
	})
}

// =============================================================================
// INTELLIGENCE TRACKING HANDLERS
// =============================================================================

// GetIntelligenceAccountability retrieves intelligence accountability statistics
func (h *GovernanceHandlers) GetIntelligenceAccountability(c *gin.Context) {
	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	accountability, err := h.governanceSvc.GetIntelligenceAccountability(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    accountability,
		"days":    days,
	})
}

// DispositionIntelligenceRequest represents the request body for dispositioning intelligence
type DispositionIntelligenceRequest struct {
	Code   string    `json:"code" binding:"required"`
	Reason string    `json:"reason,omitempty"`
	By     uuid.UUID `json:"by" binding:"required"`
}

// DispositionIntelligence records a disposition for intelligence that won't become a task
func (h *GovernanceHandlers) DispositionIntelligence(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid intelligence ID"})
		return
	}

	var req DispositionIntelligenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if err := h.governanceSvc.DispositionIntelligence(c.Request.Context(), id, req.Code, req.Reason, req.By); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"message":          "Intelligence dispositioned",
		"disposition_code": req.Code,
	})
}

// =============================================================================
// DASHBOARD & COMPLIANCE HANDLERS
// =============================================================================

// GetGovernanceDashboard retrieves governance dashboard statistics
func (h *GovernanceHandlers) GetGovernanceDashboard(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	dashboard, err := h.governanceSvc.GetGovernanceDashboard(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    dashboard,
		"days":    days,
	})
}

// GetComplianceScore retrieves the overall compliance score
func (h *GovernanceHandlers) GetComplianceScore(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	score, err := h.governanceSvc.CalculateComplianceScore(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"data":         score,
		"days":         days,
		"generated_at": time.Now().UTC(),
	})
}
