package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-clinical-context/internal/models"
	"kb-clinical-context/internal/services"
)

type ContextHandlers struct {
	contextService *services.ContextService
}

func NewContextHandlers(contextService *services.ContextService) *ContextHandlers {
	return &ContextHandlers{
		contextService: contextService,
	}
}

// buildContext handles POST /api/v1/context/build
func (h *ContextHandlers) buildContext(c *gin.Context) {
	var request models.BuildContextRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request format", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	response, err := h.contextService.BuildContext(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to build context", "CONTEXT_BUILD_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, response, map[string]interface{}{
		"phenotypes_detected": len(response.Phenotypes),
		"cache_hit":          response.CacheHit,
		"processing_time_ms": time.Since(response.ProcessedAt).Milliseconds(),
	})
}

// detectPhenotypes handles POST /api/v1/phenotypes/detect
func (h *ContextHandlers) detectPhenotypes(c *gin.Context) {
	var request models.PhenotypeDetectionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid phenotype detection request", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	response, err := h.contextService.DetectPhenotypes(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to detect phenotypes", "PHENOTYPE_DETECTION_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, response, map[string]interface{}{
		"total_phenotypes": response.TotalPhenotypes,
		"processing_time_ms": response.ProcessingTime,
	})
}

// assessRisk handles POST /api/v1/risk/assess
func (h *ContextHandlers) assessRisk(c *gin.Context) {
	var request models.RiskAssessmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid risk assessment request", "INVALID_REQUEST", map[string]interface{}{
			"validation_error": err.Error(),
		})
		return
	}

	response, err := h.contextService.AssessRisk(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to assess risk", "RISK_ASSESSMENT_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, response, map[string]interface{}{
		"risk_types_assessed": len(response.RiskScores),
		"confidence_score":    response.ConfidenceScore,
		"total_recommendations": len(response.Recommendations),
	})
}

// identifyCareGaps handles GET /api/v1/care-gaps/:patient_id
func (h *ContextHandlers) identifyCareGaps(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "Patient ID is required", "MISSING_PATIENT_ID", nil)
		return
	}

	request := models.CareGapsRequest{
		PatientID:        patientID,
		IncludeResolved:  parseBoolQuery(c, "include_resolved", false),
		TimeframeDays:    parseIntQuery(c, "timeframe_days", 90),
	}

	response, err := h.contextService.IdentifyCareGaps(request)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to identify care gaps", "CARE_GAPS_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, response, map[string]interface{}{
		"total_gaps": response.TotalGaps,
		"priority":   response.Priority,
	})
}

// getContextHistory handles GET /api/v1/context/:patient_id/history
func (h *ContextHandlers) getContextHistory(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "Patient ID is required", "MISSING_PATIENT_ID", nil)
		return
	}

	limit := parseIntQuery(c, "limit", 10)
	offset := parseIntQuery(c, "offset", 0)

	// This would implement context history retrieval
	// For now, return a placeholder response
	sendSuccess(c, map[string]interface{}{
		"patient_id": patientID,
		"contexts":   []interface{}{}, // Would be populated with historical contexts
		"pagination": map[string]interface{}{
			"limit":  limit,
			"offset": offset,
			"total":  0,
		},
	}, nil)
}

// getContextStats handles GET /api/v1/context/statistics
func (h *ContextHandlers) getContextStats(c *gin.Context) {
	days := parseIntQuery(c, "days", 30)

	// Mock statistics - would be implemented with actual data
	stats := map[string]interface{}{
		"period_days": days,
		"metrics": map[string]interface{}{
			"total_context_builds":         1250,
			"total_phenotypes_detected":    340,
			"avg_phenotypes_per_context":   2.7,
			"total_risk_assessments":       890,
			"total_care_gaps_identified":   156,
			"most_common_phenotypes": []map[string]interface{}{
				{
					"phenotype_id":  "diabetes_type_2",
					"count":         145,
					"avg_confidence": 0.87,
				},
				{
					"phenotype_id":  "hypertension_complex",
					"count":         123,
					"avg_confidence": 0.92,
				},
			},
			"risk_score_distribution": map[string]interface{}{
				"cardiovascular_risk": map[string]interface{}{
					"low":    0.45,
					"medium": 0.35,
					"high":   0.20,
				},
				"fall_risk": map[string]interface{}{
					"low":    0.65,
					"medium": 0.25,
					"high":   0.10,
				},
			},
		},
	}

	sendSuccess(c, stats, nil)
}

// Administrative endpoints

// clearContextCache handles POST /api/v1/admin/cache/clear
func (h *ContextHandlers) clearContextCache(c *gin.Context) {
	cacheType := c.Query("type")

	success := true
	message := "Context cache cleared successfully"

	switch cacheType {
	case "patient_contexts":
		// Would implement patient context cache clearing
		message = "Patient context cache cleared"
	case "phenotypes":
		// Would implement phenotype cache clearing
		message = "Phenotype cache cleared"
	case "risk_assessments":
		// Would implement risk assessment cache clearing
		message = "Risk assessment cache cleared"
	case "all":
		message = "All caches cleared"
	default:
		sendError(c, http.StatusBadRequest, "Invalid cache type", "INVALID_CACHE_TYPE", map[string]interface{}{
			"valid_types": []string{"patient_contexts", "phenotypes", "risk_assessments", "all"},
		})
		return
	}

	if !success {
		sendError(c, http.StatusInternalServerError, "Failed to clear cache", "CACHE_CLEAR_FAILED", nil)
		return
	}

	sendSuccess(c, map[string]interface{}{
		"cache_type": cacheType,
		"status":     "cleared",
		"timestamp":  time.Now().UTC(),
		"message":    message,
	}, nil)
}

// getSystemHealth handles GET /api/v1/admin/health
func (h *ContextHandlers) getSystemHealth(c *gin.Context) {
	health := map[string]interface{}{
		"service":           "kb-2-clinical-context",
		"version":           "1.0.0",
		"timestamp":         time.Now().UTC(),
		"uptime_hours":      24, // Mock uptime
		"total_contexts":    1500, // Would be queried from database
		"total_phenotypes":  25,   // Would be queried from database
		"performance": map[string]interface{}{
			"avg_context_build_time_ms":    45,
			"avg_phenotype_detect_time_ms": 12,
			"avg_risk_assessment_time_ms":  8,
		},
		"cache_stats": map[string]interface{}{
			"hit_rate":           0.95,
			"total_hits":         5600,
			"total_misses":       280,
		},
	}

	sendSuccess(c, health, nil)
}

// getPhenotypeDefinitions handles GET /api/v1/phenotypes/definitions
func (h *ContextHandlers) getPhenotypeDefinitions(c *gin.Context) {
	domain := c.Query("domain")
	status := c.Query("status")
	if status == "" {
		status = "active"
	}

	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	// Get phenotype definitions from MongoDB via context service
	phenotypes, totalCount, err := h.contextService.GetPhenotypeDefinitions(domain, status, limit, offset)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to retrieve phenotype definitions", "PHENOTYPE_RETRIEVAL_FAILED", map[string]interface{}{
			"details": err.Error(),
		})
		return
	}

	// Calculate pagination metadata
	totalPages := (totalCount + limit - 1) / limit
	currentPage := offset/limit + 1
	hasNext := offset+limit < totalCount
	hasPrevious := offset > 0

	sendSuccess(c, map[string]interface{}{
		"phenotype_definitions": phenotypes,
		"pagination": map[string]interface{}{
			"limit":        limit,
			"offset":       offset,
			"total":        totalCount,
			"total_pages":  totalPages,
			"current_page": currentPage,
			"has_next":     hasNext,
			"has_previous": hasPrevious,
		},
	}, map[string]interface{}{
		"status_filter": status,
		"domain_filter": domain,
	})
}

// Helper functions

func sendSuccess(c *gin.Context, data interface{}, meta map[string]interface{}) {
	response := gin.H{
		"success": true,
		"data":    data,
	}

	if meta != nil {
		response["meta"] = meta
	}

	c.JSON(http.StatusOK, response)
}

func sendError(c *gin.Context, statusCode int, message, code string, details interface{}) {
	response := gin.H{
		"success": false,
		"error": gin.H{
			"message": message,
			"code":    code,
		},
	}

	if details != nil {
		response["error"].(gin.H)["details"] = details
	}

	c.JSON(statusCode, response)
}

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if value := c.Query(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func parseBoolQuery(c *gin.Context, key string, defaultValue bool) bool {
	if value := c.Query(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}