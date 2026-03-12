// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-14-care-navigator/internal/models"
)

// ListEscalations retrieves escalations with optional filters
func (s *Server) ListEscalations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	// Parse filters
	var statuses []models.EscalationStatus
	if statusStrs := c.QueryArray("status"); len(statusStrs) > 0 {
		for _, s := range statusStrs {
			statuses = append(statuses, models.EscalationStatus(s))
		}
	}

	var levels []models.EscalationLevel
	if levelStrs := c.QueryArray("level"); len(levelStrs) > 0 {
		for _, l := range levelStrs {
			level, _ := strconv.Atoi(l)
			levels = append(levels, models.EscalationLevel(level))
		}
	}

	escalations, total, err := s.escalationRepo.FindWithFilters(c.Request.Context(), statuses, levels, page, pageSize)
	if err != nil {
		s.log.WithError(err).Error("Failed to list escalations")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"data":     escalations,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// GetEscalation retrieves an escalation by ID
func (s *Server) GetEscalation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid escalation ID format",
		})
		return
	}

	escalation, err := s.escalationRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "escalation not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    escalation,
	})
}

// AcknowledgeEscalation acknowledges an escalation
func (s *Server) AcknowledgeEscalation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid escalation ID format",
		})
		return
	}

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
		Notes          string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Parse the acknowledgedBy as UUID
	acknowledgedByUUID, err := uuid.Parse(req.AcknowledgedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid acknowledged_by UUID format",
		})
		return
	}

	// Acknowledge the escalation
	if err := s.escalationEngine.AcknowledgeEscalation(c.Request.Context(), id, acknowledgedByUUID); err != nil {
		s.log.WithError(err).Error("Failed to acknowledge escalation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Refresh escalation
	escalation, _ := s.escalationRepo.GetByID(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    escalation,
	})
}

// ResolveEscalation resolves an escalation
func (s *Server) ResolveEscalation(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid escalation ID format",
		})
		return
	}

	var req struct {
		ResolvedBy string `json:"resolved_by" binding:"required"`
		Resolution string `json:"resolution" binding:"required"`
		Notes      string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Parse the resolvedBy as UUID
	resolvedByUUID, err := uuid.Parse(req.ResolvedBy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "invalid resolved_by UUID format",
		})
		return
	}

	// Resolve the escalation
	if err := s.escalationEngine.ResolveEscalation(c.Request.Context(), id, resolvedByUUID, req.Resolution); err != nil {
		s.log.WithError(err).Error("Failed to resolve escalation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Refresh escalation
	escalation, _ := s.escalationRepo.GetByID(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    escalation,
	})
}
