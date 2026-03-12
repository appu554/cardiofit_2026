package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/orchestration"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/services"
)

// OrchestrationHandler handles HTTP requests for workflow orchestration
type OrchestrationHandler struct {
	orchestrationService *services.OrchestrationService
	logger               *zap.Logger
}

// NewOrchestrationHandler creates a new orchestration handler
func NewOrchestrationHandler(
	orchestrationService *services.OrchestrationService,
	logger *zap.Logger,
) *OrchestrationHandler {
	return &OrchestrationHandler{
		orchestrationService: orchestrationService,
		logger:               logger,
	}
}

// ExecuteMedicationWorkflow handles POST /api/v1/orchestration/medication
func (h *OrchestrationHandler) ExecuteMedicationWorkflow(c *gin.Context) {
	var request orchestration.OrchestrationRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.Warn("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Set correlation ID if not provided
	if request.CorrelationID == "" {
		request.CorrelationID = h.generateCorrelationID()
	}

	h.logger.Info("Received medication workflow request",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("patient_id", request.PatientID),
		zap.String("execution_mode", request.ExecutionMode))

	// Execute workflow
	response, err := h.orchestrationService.ExecuteMedicationWorkflow(c.Request.Context(), &request)
	if err != nil {
		h.logger.Error("Workflow execution failed",
			zap.String("correlation_id", request.CorrelationID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":          "Workflow execution failed",
			"correlation_id": request.CorrelationID,
			"details":        err.Error(),
		})
		return
	}

	// Set appropriate HTTP status based on workflow outcome
	statusCode := h.getStatusCodeFromResponse(response)
	
	h.logger.Info("Medication workflow completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("workflow_instance_id", response.WorkflowInstanceID),
		zap.String("status", response.Status),
		zap.Int("http_status", statusCode))

	c.JSON(statusCode, response)
}

// GetWorkflowStatus handles GET /api/v1/orchestration/workflows/{workflowId}/status
func (h *OrchestrationHandler) GetWorkflowStatus(c *gin.Context) {
	workflowID := c.Param("workflowId")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing workflow ID",
		})
		return
	}

	response, err := h.orchestrationService.GetWorkflowStatus(c.Request.Context(), workflowID)
	if err != nil {
		if err.Error() == "workflow instance not found: "+workflowID {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Workflow not found",
				"workflow_id": workflowID,
			})
			return
		}

		h.logger.Error("Failed to get workflow status",
			zap.String("workflow_id", workflowID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":       "Failed to retrieve workflow status",
			"workflow_id": workflowID,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListWorkflows handles GET /api/v1/orchestration/workflows
func (h *OrchestrationHandler) ListWorkflows(c *gin.Context) {
	// Parse query parameters
	filters := &services.WorkflowListFilters{
		Limit:  50,  // Default limit
		Offset: 0,   // Default offset
	}

	if patientID := c.Query("patient_id"); patientID != "" {
		filters.PatientID = patientID
	}

	if status := c.Query("status"); status != "" {
		filters.Status = status
	}

	if definitionID := c.Query("definition_id"); definitionID != "" {
		filters.DefinitionID = definitionID
	}

	if limit := c.Query("limit"); limit != "" {
		if parsedLimit, err := strconv.Atoi(limit); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			filters.Limit = parsedLimit
		}
	}

	if offset := c.Query("offset"); offset != "" {
		if parsedOffset, err := strconv.Atoi(offset); err == nil && parsedOffset >= 0 {
			filters.Offset = parsedOffset
		}
	}

	if startedAfter := c.Query("started_after"); startedAfter != "" {
		if parsed, err := time.Parse(time.RFC3339, startedAfter); err == nil {
			filters.StartedAfter = parsed
		}
	}

	if startedBefore := c.Query("started_before"); startedBefore != "" {
		if parsed, err := time.Parse(time.RFC3339, startedBefore); err == nil {
			filters.StartedBefore = parsed
		}
	}

	response, err := h.orchestrationService.ListWorkflowInstances(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("Failed to list workflows", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve workflows",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetSystemHealth handles GET /health
func (h *OrchestrationHandler) GetSystemHealth(c *gin.Context) {
	response := h.orchestrationService.GetSystemHealth(c.Request.Context())
	
	// Set HTTP status based on system health
	statusCode := http.StatusOK
	if response.Status == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

// Helper methods

func (h *OrchestrationHandler) generateCorrelationID() string {
	// Simple correlation ID generation using timestamp and random suffix
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

func (h *OrchestrationHandler) getStatusCodeFromResponse(response *orchestration.OrchestrationResponse) int {
	switch response.Status {
	case "completed":
		return http.StatusOK
	case "completed_no_commit":
		return http.StatusAccepted // 202 - completed but not fully processed
	case "failed":
		return http.StatusUnprocessableEntity // 422 - workflow failed due to business logic
	default:
		return http.StatusOK
	}
}

// RegisterRoutes registers all orchestration routes
func (h *OrchestrationHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Orchestration endpoints
	orchestration := r.Group("/orchestration")
	{
		orchestration.POST("/medication", h.ExecuteMedicationWorkflow)
		orchestration.GET("/workflows", h.ListWorkflows)
		orchestration.GET("/workflows/:workflowId/status", h.GetWorkflowStatus)
	}

	// Health check endpoint (at root level)
	r.GET("/health", h.GetSystemHealth)
}

// Middleware for request logging and correlation tracking
func (h *OrchestrationHandler) RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Get or generate correlation ID
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = h.generateCorrelationID()
		}
		
		// Set correlation ID in response header
		c.Header("X-Correlation-ID", correlationID)
		
		// Add correlation ID to context for downstream usage
		c.Set("correlation_id", correlationID)

		h.logger.Info("Incoming request",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("correlation_id", correlationID),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("remote_addr", c.ClientIP()))

		// Process request
		c.Next()

		// Log response
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		
		logLevel := zap.InfoLevel
		if statusCode >= 400 {
			logLevel = zap.WarnLevel
		}
		if statusCode >= 500 {
			logLevel = zap.ErrorLevel
		}

		h.logger.Log(logLevel, "Request completed",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("correlation_id", correlationID),
			zap.Int("status_code", statusCode),
			zap.Duration("duration", duration),
			zap.Int("response_size", c.Writer.Size()))
	}
}

// CORS middleware for cross-origin requests
func (h *OrchestrationHandler) CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// In production, this should be configured with specific allowed origins
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Correlation-ID")
		c.Header("Access-Control-Expose-Headers", "X-Correlation-ID")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Error handling middleware
func (h *OrchestrationHandler) ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				correlationID, _ := c.Get("correlation_id")
				
				h.logger.Error("Panic recovered",
					zap.Any("panic", r),
					zap.String("path", c.Request.URL.Path),
					zap.String("method", c.Request.Method),
					zap.Any("correlation_id", correlationID))

				c.JSON(http.StatusInternalServerError, gin.H{
					"error":          "Internal server error",
					"correlation_id": correlationID,
				})
			}
		}()
		
		c.Next()
	}
}