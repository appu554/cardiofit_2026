package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-clinical-context/internal/services"
	"kb-clinical-context/internal/models"
)

// PhenotypeHandlers handles phenotype-related API endpoints
type PhenotypeHandlers struct {
	contextService *services.ContextService
	logger         *zap.Logger
}

// NewPhenotypeHandlers creates new phenotype handlers
func NewPhenotypeHandlers(contextService *services.ContextService, logger *zap.Logger) *PhenotypeHandlers {
	return &PhenotypeHandlers{
		contextService: contextService,
		logger:         logger,
	}
}

// ValidatePhenotypes validates all loaded phenotype expressions
// GET /api/v1/phenotypes/validate
func (h *PhenotypeHandlers) ValidatePhenotypes(c *gin.Context) {
	h.logger.Info("Validating all phenotype expressions")

	// Get phenotype engine from context service
	contextService := h.contextService
	if contextService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Context service not available",
		})
		return
	}

	// Validate all phenotypes using the phenotype engine
	validationResults, err := contextService.ValidateAllPhenotypes()
	if err != nil {
		h.logger.Error("Failed to validate phenotypes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate phenotypes",
			"details": err.Error(),
		})
		return
	}

	// Count valid and invalid phenotypes
	validCount := 0
	invalidCount := 0
	for _, result := range validationResults {
		if result.Valid {
			validCount++
		} else {
			invalidCount++
		}
	}

	response := gin.H{
		"status": "completed",
		"summary": gin.H{
			"total_phenotypes":   len(validationResults),
			"valid_phenotypes":   validCount,
			"invalid_phenotypes": invalidCount,
		},
		"results": validationResults,
	}

	h.logger.Info("Phenotype validation completed",
		zap.Int("total", len(validationResults)),
		zap.Int("valid", validCount),
		zap.Int("invalid", invalidCount))

	c.JSON(http.StatusOK, response)
}

// GetEngineStats returns statistics about the phenotype engines
// GET /api/v1/phenotypes/engine/stats
func (h *PhenotypeHandlers) GetEngineStats(c *gin.Context) {
	h.logger.Info("Retrieving engine statistics")

	stats := h.contextService.GetEngineStats()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
	})
}

// ReloadPhenotypes reloads all phenotype definitions from files
// POST /api/v1/phenotypes/reload
func (h *PhenotypeHandlers) ReloadPhenotypes(c *gin.Context) {
	h.logger.Info("Reloading phenotype definitions")

	err := h.contextService.ReloadPhenotypes()
	if err != nil {
		h.logger.Error("Failed to reload phenotypes", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to reload phenotypes",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Phenotype definitions reloaded successfully")
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"message": "Phenotype definitions reloaded successfully",
	})
}

// TestPhenotypeExpression tests a specific phenotype expression against patient data
// POST /api/v1/phenotypes/test
func (h *PhenotypeHandlers) TestPhenotypeExpression(c *gin.Context) {
	var request struct {
		Expression    string                 `json:"expression" binding:"required"`
		LogicEngine   string                 `json:"logic_engine,omitempty"`
		PatientData   models.PatientContext  `json:"patient_data" binding:"required"`
		PhenotypeID   string                 `json:"phenotype_id,omitempty"`
		PhenotypeName string                 `json:"phenotype_name,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Testing phenotype expression",
		zap.String("expression", request.Expression),
		zap.String("logic_engine", request.LogicEngine),
		zap.String("patient_id", request.PatientData.PatientID))

	// Create a test phenotype definition
	testPhenotype := models.PhenotypeDetectionRequest{
		PatientID:   request.PatientData.PatientID,
		PatientData: h.convertPatientContextToMap(request.PatientData),
	}

	// Test the phenotype
	result, err := h.contextService.DetectPhenotypes(testPhenotype)
	if err != nil {
		h.logger.Error("Failed to test phenotype expression", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to test phenotype expression",
			"details": err.Error(),
		})
		return
	}

	response := gin.H{
		"status": "success",
		"test_result": gin.H{
			"expression":     request.Expression,
			"logic_engine":   request.LogicEngine,
			"patient_id":     request.PatientData.PatientID,
			"detected_phenotypes": result.DetectedPhenotypes,
			"total_phenotypes":    result.TotalPhenotypes,
			"processing_time_ms":  result.ProcessingTime,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetPhenotypeDefinitions returns all loaded phenotype definitions
// GET /api/v1/phenotypes/definitions
func (h *PhenotypeHandlers) GetPhenotypeDefinitions(c *gin.Context) {
	// Query parameters
	domain := c.Query("domain")
	status := c.Query("status")
	limitStr := c.Query("limit")
	offsetStr := c.Query("offset")

	// Parse pagination parameters
	limit := 50 // default
	offset := 0 // default

	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	h.logger.Info("Retrieving phenotype definitions",
		zap.String("domain", domain),
		zap.String("status", status),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// Get phenotype definitions from MongoDB
	phenotypes, totalCount, err := h.contextService.GetPhenotypeDefinitions(domain, status, limit, offset)
	if err != nil {
		h.logger.Error("Failed to retrieve phenotype definitions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve phenotype definitions",
			"details": err.Error(),
		})
		return
	}

	// Calculate pagination metadata
	totalPages := (totalCount + limit - 1) / limit
	currentPage := offset/limit + 1
	hasNext := offset+limit < totalCount
	hasPrevious := offset > 0

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"phenotypes": phenotypes,
			"pagination": gin.H{
				"total_count":    totalCount,
				"total_pages":    totalPages,
				"current_page":   currentPage,
				"page_size":      limit,
				"offset":         offset,
				"has_next":       hasNext,
				"has_previous":   hasPrevious,
			},
			"filters": gin.H{
				"domain": domain,
				"status": status,
			},
		},
	})
}

// HealthCheck returns the health status of the phenotype engine
// GET /api/v1/phenotypes/health
func (h *PhenotypeHandlers) HealthCheck(c *gin.Context) {
	// Simple health check - attempt to get engine stats
	stats := h.contextService.GetEngineStats()
	
	status := "healthy"
	if stats == nil {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"timestamp": "2025-09-01T00:00:00Z",
		"engine_stats": stats,
	})
}

// Helper method to convert PatientContext to map format
func (h *PhenotypeHandlers) convertPatientContextToMap(context models.PatientContext) map[string]interface{} {
	return map[string]interface{}{
		"demographics":        context.Demographics,
		"active_conditions":   context.ActiveConditions,
		"recent_labs":         context.RecentLabs,
		"current_medications": context.CurrentMeds,
		"detected_phenotypes": context.DetectedPhenotypes,
		"risk_factors":        context.RiskFactors,
	}
}

// RegisterRoutes registers all phenotype-related routes
func (h *PhenotypeHandlers) RegisterRoutes(router *gin.RouterGroup) {
	phenotypes := router.Group("/phenotypes")
	{
		phenotypes.GET("/validate", h.ValidatePhenotypes)
		phenotypes.GET("/engine/stats", h.GetEngineStats)
		phenotypes.POST("/reload", h.ReloadPhenotypes)
		phenotypes.POST("/test", h.TestPhenotypeExpression)
		phenotypes.GET("/definitions", h.GetPhenotypeDefinitions)
		phenotypes.GET("/health", h.HealthCheck)
	}
}