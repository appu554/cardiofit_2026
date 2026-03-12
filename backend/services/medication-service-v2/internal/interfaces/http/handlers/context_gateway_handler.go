package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
)

// ContextGatewayHandler handles Context Gateway integration endpoints
type ContextGatewayHandler struct {
	contextIntegration *services.RecipeResolverContextIntegration
	contextGateway     *services.ContextGatewayService
	logger             *zap.Logger
}

// NewContextGatewayHandler creates a new Context Gateway handler
func NewContextGatewayHandler(
	contextIntegration *services.RecipeResolverContextIntegration,
	contextGateway *services.ContextGatewayService,
	logger *zap.Logger,
) *ContextGatewayHandler {
	return &ContextGatewayHandler{
		contextIntegration: contextIntegration,
		contextGateway:     contextGateway,
		logger:             logger,
	}
}

// IntegratedWorkflowRequest represents the HTTP request for integrated workflow
type IntegratedWorkflowRequestDTO struct {
	// Recipe resolution parameters
	RecipeID      string `json:"recipe_id" binding:"required"`
	PatientID     string `json:"patient_id" binding:"required"`
	
	// Patient context
	PatientContext PatientContextDTO `json:"patient_context" binding:"required"`
	
	// Resolution options
	Options RecipeResolutionOptionsDTO `json:"options,omitempty"`
	
	// Snapshot creation options
	CreateSnapshot        bool                     `json:"create_snapshot"`
	SnapshotType         string                   `json:"snapshot_type,omitempty"`
	SnapshotPriority     string                   `json:"snapshot_priority,omitempty"`
	CustomFreshness      map[string]string        `json:"custom_freshness_requirements,omitempty"`
	RequireValidation    bool                     `json:"require_validation"`
	
	// Workflow metadata
	RequestedBy          string                   `json:"requested_by,omitempty"`
	ClientContext        map[string]string        `json:"client_context,omitempty"`
}

// PatientContextDTO represents patient context for HTTP requests
type PatientContextDTO struct {
	PatientID        string                 `json:"patient_id" binding:"required"`
	Demographics     map[string]interface{} `json:"demographics,omitempty"`
	ClinicalData     map[string]interface{} `json:"clinical_data,omitempty"`
	PreferredUnits   map[string]string      `json:"preferred_units,omitempty"`
	ContextTimestamp string                 `json:"context_timestamp,omitempty"`
}

// RecipeResolutionOptionsDTO represents resolution options for HTTP requests
type RecipeResolutionOptionsDTO struct {
	UseCache            bool   `json:"use_cache"`
	CacheTTL            string `json:"cache_ttl,omitempty"`
	ForceRecalculation  bool   `json:"force_recalculation"`
	IncludeAuditTrail   bool   `json:"include_audit_trail"`
	PerformanceMode     string `json:"performance_mode,omitempty"`
}

// IntegratedWorkflowResponse represents the HTTP response for integrated workflow
type IntegratedWorkflowResponseDTO struct {
	WorkflowID           string                    `json:"workflow_id"`
	Status               string                    `json:"status"`
	ProcessingTime       string                    `json:"processing_time"`
	RecipeResolution     interface{}               `json:"recipe_resolution,omitempty"`
	ResolutionTime       string                    `json:"resolution_time"`
	ResolutionQuality    float64                   `json:"resolution_quality"`
	SnapshotResult       *SnapshotResultDTO        `json:"snapshot_result,omitempty"`
	SnapshotCreationTime string                    `json:"snapshot_creation_time"`
	QualityScore         float64                   `json:"overall_quality_score"`
	Warnings             []string                  `json:"warnings,omitempty"`
	Errors               []string                  `json:"errors,omitempty"`
	CreatedAt            string                    `json:"created_at"`
	CompletedAt          string                    `json:"completed_at"`
	ProcessedBy          string                    `json:"processed_by"`
}

// SnapshotResultDTO represents snapshot creation result
type SnapshotResultDTO struct {
	SnapshotID       string            `json:"snapshot_id"`
	Status           string            `json:"status"`
	CreatedAt        string            `json:"created_at"`
	ExpiresAt        string            `json:"expires_at"`
	QualityScore     float64           `json:"quality_score"`
	ProcessingTime   string            `json:"processing_time"`
	ValidationResult interface{}       `json:"validation_result,omitempty"`
	Warnings         []string          `json:"warnings,omitempty"`
	DataSources      map[string]interface{} `json:"data_sources"`
}

// ExecuteIntegratedWorkflow handles the integrated Phase 1 → Phase 2 workflow
// POST /api/v1/context-gateway/integrated-workflow
func (h *ContextGatewayHandler) ExecuteIntegratedWorkflow(c *gin.Context) {
	var req IntegratedWorkflowRequestDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request payload", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Convert HTTP request to service request
	serviceRequest, err := h.convertToServiceRequest(&req)
	if err != nil {
		h.logger.Error("Failed to convert request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Execute workflow
	result, err := h.contextIntegration.ExecuteIntegratedWorkflow(c.Request.Context(), serviceRequest)
	if err != nil {
		h.logger.Error("Integrated workflow failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Workflow execution failed", "details": err.Error()})
		return
	}

	// Convert response
	response := h.convertToHTTPResponse(result)
	c.JSON(http.StatusOK, response)
}

// CreateSnapshot creates a snapshot from recipe resolution
// POST /api/v1/context-gateway/snapshots
func (h *ContextGatewayHandler) CreateSnapshot(c *gin.Context) {
	var req struct {
		PatientID         string                   `json:"patient_id" binding:"required"`
		RecipeID          string                   `json:"recipe_id" binding:"required"`
		RecipeResolution  interface{}              `json:"recipe_resolution" binding:"required"`
		PatientContext    PatientContextDTO        `json:"patient_context" binding:"required"`
		SnapshotType      string                   `json:"snapshot_type"`
		Priority          string                   `json:"priority"`
		CreatedBy         string                   `json:"created_by"`
		RequireValidation bool                     `json:"require_validation"`
		CustomFreshness   map[string]string        `json:"custom_freshness,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid snapshot request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Convert to service request
	snapshotReq, err := h.convertToSnapshotRequest(&req)
	if err != nil {
		h.logger.Error("Failed to convert snapshot request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid snapshot data", "details": err.Error()})
		return
	}

	// Create snapshot
	result, err := h.contextGateway.CreateSnapshotFromResolution(c.Request.Context(), snapshotReq)
	if err != nil {
		h.logger.Error("Snapshot creation failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Snapshot creation failed", "details": err.Error()})
		return
	}

	// Convert response
	response := &SnapshotResultDTO{
		SnapshotID:     result.SnapshotID,
		Status:         result.Status,
		CreatedAt:      result.CreatedAt.Format(time.RFC3339),
		ExpiresAt:      result.ExpiresAt.Format(time.RFC3339),
		QualityScore:   result.QualityScore,
		ProcessingTime: result.ProcessingTime.String(),
		Warnings:       result.Warnings,
		DataSources:    result.DataSources,
	}

	if result.ValidationResult != nil {
		response.ValidationResult = result.ValidationResult
	}

	c.JSON(http.StatusCreated, response)
}

// GetSnapshot retrieves a snapshot by ID
// GET /api/v1/context-gateway/snapshots/:id
func (h *ContextGatewayHandler) GetSnapshot(c *gin.Context) {
	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Snapshot ID is required"})
		return
	}

	snapshot, err := h.contextGateway.GetSnapshot(c.Request.Context(), snapshotID)
	if err != nil {
		h.logger.Error("Failed to get snapshot", zap.String("snapshot_id", snapshotID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Snapshot not found", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// SupersedeSnapshot creates a new snapshot and marks old one as superseded
// POST /api/v1/context-gateway/snapshots/:id/supersede
func (h *ContextGatewayHandler) SupersedeSnapshot(c *gin.Context) {
	oldSnapshotID := c.Param("id")
	if oldSnapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Snapshot ID is required"})
		return
	}

	var req struct {
		IntegratedWorkflowRequestDTO
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid supersede request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	// Convert to service request
	serviceRequest, err := h.convertToServiceRequest(&req.IntegratedWorkflowRequestDTO)
	if err != nil {
		h.logger.Error("Failed to convert supersede request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Execute supersession
	result, err := h.contextIntegration.SupersedeSnapshot(
		c.Request.Context(),
		oldSnapshotID,
		serviceRequest,
		req.Reason,
	)
	if err != nil {
		h.logger.Error("Snapshot supersession failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Supersession failed", "details": err.Error()})
		return
	}

	response := h.convertToHTTPResponse(result)
	c.JSON(http.StatusOK, response)
}

// ListSnapshotsForPatient lists snapshots for a patient
// GET /api/v1/context-gateway/patients/:patient_id/snapshots
func (h *ContextGatewayHandler) ListSnapshotsForPatient(c *gin.Context) {
	patientIDStr := c.Param("patient_id")
	if patientIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Patient ID is required"})
		return
	}

	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid patient ID format"})
		return
	}

	// Parse query parameters
	status := c.Query("status")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	snapshots, err := h.contextGateway.ListSnapshotsForPatient(
		c.Request.Context(),
		patientID,
		status,
		limit,
	)
	if err != nil {
		h.logger.Error("Failed to list snapshots", zap.String("patient_id", patientIDStr), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list snapshots", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshots)
}

// GetWorkflowMetrics returns workflow performance metrics
// GET /api/v1/context-gateway/metrics
func (h *ContextGatewayHandler) GetWorkflowMetrics(c *gin.Context) {
	metrics := h.contextIntegration.GetWorkflowMetrics()
	c.JSON(http.StatusOK, metrics)
}

// HealthCheck performs health check for Context Gateway integration
// GET /api/v1/context-gateway/health
func (h *ContextGatewayHandler) HealthCheck(c *gin.Context) {
	health, err := h.contextIntegration.HealthCheck(c.Request.Context())
	if err != nil {
		h.logger.Error("Health check failed", zap.Error(err))
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Health check failed", "details": err.Error()})
		return
	}

	status := http.StatusOK
	if health["status"] != "healthy" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, health)
}

// Helper methods

func (h *ContextGatewayHandler) convertToServiceRequest(req *IntegratedWorkflowRequestDTO) (*services.IntegratedWorkflowRequest, error) {
	// Generate workflow ID
	workflowID := uuid.New()

	// Parse patient context
	var contextTimestamp time.Time
	var err error
	if req.PatientContext.ContextTimestamp != "" {
		contextTimestamp, err = time.Parse(time.RFC3339, req.PatientContext.ContextTimestamp)
		if err != nil {
			contextTimestamp = time.Now()
		}
	} else {
		contextTimestamp = time.Now()
	}

	// Parse cache TTL
	var cacheTTL time.Duration
	if req.Options.CacheTTL != "" {
		cacheTTL, err = time.ParseDuration(req.Options.CacheTTL)
		if err != nil {
			cacheTTL = 5 * time.Minute // Default
		}
	}

	// Convert patient context
	patientContext := entities.PatientContext{
		PatientID:        req.PatientContext.PatientID,
		Demographics:     req.PatientContext.Demographics,
		ClinicalData:     req.PatientContext.ClinicalData,
		PreferredUnits:   req.PatientContext.PreferredUnits,
		ContextTimestamp: contextTimestamp,
	}

	// Convert resolution options
	options := entities.RecipeResolutionOptions{
		UseCache:           req.Options.UseCache,
		CacheTTL:           cacheTTL,
		ForceRecalculation: req.Options.ForceRecalculation,
		IncludeAuditTrail:  req.Options.IncludeAuditTrail,
		PerformanceMode:    req.Options.PerformanceMode,
	}

	// Parse custom freshness requirements
	var customFreshness map[string]time.Duration
	if req.CustomFreshness != nil {
		customFreshness = make(map[string]time.Duration)
		for key, value := range req.CustomFreshness {
			if duration, err := time.ParseDuration(value); err == nil {
				customFreshness[key] = duration
			}
		}
	}

	serviceRequest := &services.IntegratedWorkflowRequest{
		RecipeResolutionRequest: entities.RecipeResolutionRequest{
			RecipeID:       req.RecipeID,
			PatientContext: patientContext,
			Options:        options,
		},
		CreateSnapshot:      req.CreateSnapshot,
		SnapshotType:       req.SnapshotType,
		SnapshotPriority:   req.SnapshotPriority,
		CustomFreshnessReqs: customFreshness,
		RequireValidation:  req.RequireValidation,
		WorkflowID:         workflowID,
		RequestedBy:        req.RequestedBy,
		ClientContext:      req.ClientContext,
	}

	return serviceRequest, nil
}

func (h *ContextGatewayHandler) convertToSnapshotRequest(req interface{}) (*services.SnapshotCreationRequest, error) {
	// This would need proper implementation based on the specific request structure
	// For now, return a basic implementation
	return nil, nil
}

func (h *ContextGatewayHandler) convertToHTTPResponse(result *services.IntegratedWorkflowResponse) *IntegratedWorkflowResponseDTO {
	response := &IntegratedWorkflowResponseDTO{
		WorkflowID:        result.WorkflowID.String(),
		Status:            result.Status,
		ProcessingTime:    result.ProcessingTime.String(),
		ResolutionTime:    result.ResolutionTime.String(),
		ResolutionQuality: result.ResolutionQuality,
		QualityScore:      result.QualityScore,
		Warnings:          result.Warnings,
		Errors:            result.Errors,
		CreatedAt:         result.CreatedAt.Format(time.RFC3339),
		CompletedAt:       result.CompletedAt.Format(time.RFC3339),
		ProcessedBy:       result.ProcessedBy,
	}

	// Add recipe resolution if present
	if result.RecipeResolution != nil {
		response.RecipeResolution = result.RecipeResolution
	}

	// Add snapshot result if present
	if result.SnapshotResult != nil {
		response.SnapshotResult = &SnapshotResultDTO{
			SnapshotID:     result.SnapshotResult.SnapshotID,
			Status:         result.SnapshotResult.Status,
			CreatedAt:      result.SnapshotResult.CreatedAt.Format(time.RFC3339),
			ExpiresAt:      result.SnapshotResult.ExpiresAt.Format(time.RFC3339),
			QualityScore:   result.SnapshotResult.QualityScore,
			ProcessingTime: result.SnapshotResult.ProcessingTime.String(),
			Warnings:       result.SnapshotResult.Warnings,
			DataSources:    result.SnapshotResult.DataSources,
		}

		if result.SnapshotResult.ValidationResult != nil {
			response.SnapshotResult.ValidationResult = result.SnapshotResult.ValidationResult
		}

		response.SnapshotCreationTime = result.SnapshotCreationTime.String()
	}

	return response
}