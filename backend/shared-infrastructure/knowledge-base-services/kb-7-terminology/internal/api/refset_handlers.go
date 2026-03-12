package api

import (
	"net/http"
	"strconv"

	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// Refset HTTP Handlers
// ============================================================================
// HTTP handlers for NCTS reference set operations.
// Provides endpoints for listing refsets, querying members, and checking membership.
// ============================================================================

// RefsetHandlers contains the refset-related handlers
type RefsetHandlers struct {
	refsetService *services.RefsetService
	logger        *logrus.Logger
}

// NewRefsetHandlers creates a new RefsetHandlers instance
func NewRefsetHandlers(refsetService *services.RefsetService, logger *logrus.Logger) *RefsetHandlers {
	return &RefsetHandlers{
		refsetService: refsetService,
		logger:        logger,
	}
}

// ============================================================================
// Refset Listing Endpoints
// ============================================================================

// ListRefsets handles GET /v1/refsets
// Returns all available reference sets with member counts
func (h *RefsetHandlers) ListRefsets(c *gin.Context) {
	ctx := c.Request.Context()

	result, err := h.refsetService.ListRefsets(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list refsets")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to list refsets: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRefset handles GET /v1/refsets/:refsetId
// Returns details for a specific reference set
func (h *RefsetHandlers) GetRefset(c *gin.Context) {
	ctx := c.Request.Context()
	refsetID := c.Param("refsetId")

	if refsetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "refsetId is required",
		})
		return
	}

	result, err := h.refsetService.GetRefset(ctx, refsetID)
	if err != nil {
		h.logger.WithError(err).WithField("refset_id", refsetID).Error("Failed to get refset")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get refset: " + err.Error(),
		})
		return
	}

	if !result.Success || result.Refset == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Refset not found: " + refsetID,
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Refset Members Endpoint
// ============================================================================

// GetRefsetMembers handles GET /v1/refsets/:refsetId/members
// Returns members of a reference set with pagination
// Query params: limit, offset, active_only
func (h *RefsetHandlers) GetRefsetMembers(c *gin.Context) {
	ctx := c.Request.Context()
	refsetID := c.Param("refsetId")

	if refsetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "refsetId is required",
		})
		return
	}

	// Parse query options
	opts := models.DefaultRefsetQueryOptions()

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 1000 {
			opts.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			opts.Offset = offset
		}
	}

	if activeOnly := c.Query("active_only"); activeOnly == "false" {
		opts.ActiveOnly = false
	}

	if includeCounts := c.Query("include_counts"); includeCounts == "false" {
		opts.IncludeCounts = false
	}

	result, err := h.refsetService.GetRefsetMembers(ctx, refsetID, opts)
	if err != nil {
		h.logger.WithError(err).WithField("refset_id", refsetID).Error("Failed to get refset members")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get refset members: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Concept Refsets Endpoint (Reverse Lookup)
// ============================================================================

// GetConceptRefsets handles GET /v1/concepts/:code/refsets
// Returns all reference sets a concept belongs to
func (h *RefsetHandlers) GetConceptRefsets(c *gin.Context) {
	ctx := c.Request.Context()
	conceptCode := c.Param("code")

	if conceptCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "concept code is required",
		})
		return
	}

	result, err := h.refsetService.GetConceptRefsets(ctx, conceptCode)
	if err != nil {
		h.logger.WithError(err).WithField("concept_code", conceptCode).Error("Failed to get concept refsets")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get concept refsets: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Membership Check Endpoint
// ============================================================================

// CheckRefsetMembership handles GET /v1/refsets/:refsetId/contains/:code
// Checks if a concept is a member of a reference set (O(1) lookup)
func (h *RefsetHandlers) CheckRefsetMembership(c *gin.Context) {
	ctx := c.Request.Context()
	refsetID := c.Param("refsetId")
	conceptCode := c.Param("code")

	if refsetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "refsetId is required",
		})
		return
	}

	if conceptCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "concept code is required",
		})
		return
	}

	result, err := h.refsetService.IsConceptInRefset(ctx, conceptCode, refsetID)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"refset_id":    refsetID,
			"concept_code": conceptCode,
		}).Error("Failed to check refset membership")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check membership: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Import Status Endpoint
// ============================================================================

// GetImportStatus handles GET /v1/refsets/import-status
// Returns the current import status and version history
func (h *RefsetHandlers) GetImportStatus(c *gin.Context) {
	ctx := c.Request.Context()

	result, err := h.refsetService.GetImportStatus(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get import status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get import status: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// Additional Handlers
// ============================================================================

// RefsetHealth handles GET /v1/refsets/health
// Returns health status of the refset subsystem
func (h *RefsetHandlers) RefsetHealth(c *gin.Context) {
	ctx := c.Request.Context()

	// Try to get import status to verify connectivity
	status, err := h.refsetService.GetImportStatus(ctx)

	response := gin.H{
		"status":    "healthy",
		"component": "refset_service",
	}

	if err != nil {
		response["status"] = "degraded"
		response["error"] = err.Error()
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	response["current_version"] = status.CurrentVersion
	response["relationship_count"] = status.RelationshipCount

	c.JSON(http.StatusOK, response)
}
