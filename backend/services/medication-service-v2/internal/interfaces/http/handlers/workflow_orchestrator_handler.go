package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"medication-service-v2/internal/application/services"
	
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// WorkflowOrchestratorHandler handles HTTP requests for workflow orchestration
type WorkflowOrchestratorHandler struct {
	workflowOrchestrator *services.WorkflowOrchestratorService
	workflowStateService *services.WorkflowStateService
	metricsService       *services.MetricsService
	logger               *zap.Logger
}

// NewWorkflowOrchestratorHandler creates a new workflow orchestrator handler
func NewWorkflowOrchestratorHandler(
	workflowOrchestrator *services.WorkflowOrchestratorService,
	workflowStateService *services.WorkflowStateService,
	metricsService *services.MetricsService,
	logger *zap.Logger,
) *WorkflowOrchestratorHandler {
	return &WorkflowOrchestratorHandler{
		workflowOrchestrator: workflowOrchestrator,
		workflowStateService: workflowStateService,
		metricsService:       metricsService,
		logger:               logger,
	}
}

// RegisterRoutes registers all workflow orchestrator routes
func (h *WorkflowOrchestratorHandler) RegisterRoutes(router *mux.Router) {
	// Main workflow execution
	router.HandleFunc("/api/v1/workflows/execute", h.ExecuteWorkflow).Methods("POST")
	router.HandleFunc("/api/v1/workflows/{workflowId}", h.GetWorkflowResult).Methods("GET")
	router.HandleFunc("/api/v1/workflows/{workflowId}/cancel", h.CancelWorkflow).Methods("POST")
	
	// Workflow state management
	router.HandleFunc("/api/v1/workflows/{workflowId}/status", h.GetWorkflowStatus).Methods("GET")
	router.HandleFunc("/api/v1/workflows/{workflowId}/progress", h.GetWorkflowProgress).Methods("GET")
	router.HandleFunc("/api/v1/workflows/active", h.ListActiveWorkflows).Methods("GET")
	router.HandleFunc("/api/v1/workflows/query", h.QueryWorkflows).Methods("POST")
	
	// Performance and monitoring
	router.HandleFunc("/api/v1/workflows/metrics", h.GetWorkflowMetrics).Methods("GET")
	router.HandleFunc("/api/v1/workflows/performance", h.GetPerformanceReport).Methods("GET")
	router.HandleFunc("/api/v1/workflows/{workflowId}/performance", h.GetWorkflowPerformance).Methods("GET")
	
	// Health and diagnostics
	router.HandleFunc("/api/v1/workflows/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/api/v1/workflows/statistics", h.GetStatistics).Methods("GET")
}

// ExecuteWorkflow executes a complete 4-phase medication workflow
func (h *WorkflowOrchestratorHandler) ExecuteWorkflow(w http.ResponseWriter, r *http.Request) {
	var request services.WorkflowExecutionRequest
	
	// Parse request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.logger.Error("Failed to parse workflow request", zap.Error(err))
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	
	// Generate workflow ID if not provided
	if request.WorkflowID == uuid.Nil {
		request.WorkflowID = uuid.New()
	}
	
	// Set timestamps
	request.CreatedAt = time.Now()
	
	// Validate request
	if err := h.validateWorkflowRequest(&request); err != nil {
		h.logger.Error("Invalid workflow request", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	h.logger.Info("Executing workflow",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("patient_id", request.PatientID),
		zap.String("recipe_id", request.RecipeID),
		zap.String("requested_by", request.RequestedBy),
	)
	
	// Execute workflow
	result, err := h.workflowOrchestrator.ExecuteWorkflow(r.Context(), &request)
	if err != nil {
		h.logger.Error("Workflow execution failed",
			zap.String("workflow_id", request.WorkflowID.String()),
			zap.Error(err),
		)
		
		// Return appropriate error response
		statusCode := h.getErrorStatusCode(err)
		errorResponse := map[string]interface{}{
			"error":       "workflow_execution_failed",
			"message":     err.Error(),
			"workflow_id": request.WorkflowID.String(),
			"timestamp":   time.Now(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	
	h.logger.Info("Workflow completed successfully",
		zap.String("workflow_id", result.WorkflowID.String()),
		zap.String("status", result.Status.String()),
		zap.Duration("duration", result.TotalDuration),
		zap.Float64("quality", result.QualityMetrics.OverallQuality),
	)
	
	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// GetWorkflowResult gets the result of a completed workflow
func (h *WorkflowOrchestratorHandler) GetWorkflowResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowIDStr := vars["workflowId"]
	
	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		http.Error(w, "Invalid workflow ID format", http.StatusBadRequest)
		return
	}
	
	// Get workflow state from state service
	state, err := h.workflowStateService.GetState(r.Context(), workflowID)
	if err != nil {
		h.logger.Error("Failed to get workflow state",
			zap.String("workflow_id", workflowID.String()),
			zap.Error(err),
		)
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}
	
	// Build response from state
	response := h.buildWorkflowResultFromState(state)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetWorkflowStatus gets the current status of a workflow
func (h *WorkflowOrchestratorHandler) GetWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowIDStr := vars["workflowId"]
	
	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		http.Error(w, "Invalid workflow ID format", http.StatusBadRequest)
		return
	}
	
	// Get current workflow status
	state, err := h.workflowOrchestrator.GetWorkflowStatus(r.Context(), workflowID)
	if err != nil {
		h.logger.Error("Failed to get workflow status",
			zap.String("workflow_id", workflowID.String()),
			zap.Error(err),
		)
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}
	
	statusResponse := map[string]interface{}{
		"workflow_id":    state.WorkflowID.String(),
		"status":         state.Status,
		"current_phase":  state.CurrentPhase,
		"created_at":     state.CreatedAt,
		"updated_at":     state.UpdatedAt,
		"completed_at":   state.CompletedAt,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(statusResponse)
}

// GetWorkflowProgress gets detailed progress information for a workflow
func (h *WorkflowOrchestratorHandler) GetWorkflowProgress(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowIDStr := vars["workflowId"]
	
	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		http.Error(w, "Invalid workflow ID format", http.StatusBadRequest)
		return
	}
	
	// Get workflow progress
	progress, err := h.workflowStateService.GetWorkflowProgress(r.Context(), workflowID)
	if err != nil {
		h.logger.Error("Failed to get workflow progress",
			zap.String("workflow_id", workflowID.String()),
			zap.Error(err),
		)
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(progress)
}

// CancelWorkflow cancels an active workflow
func (h *WorkflowOrchestratorHandler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowIDStr := vars["workflowId"]
	
	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		http.Error(w, "Invalid workflow ID format", http.StatusBadRequest)
		return
	}
	
	// Parse cancellation request
	var cancelRequest struct {
		Reason      string `json:"reason"`
		RequestedBy string `json:"requested_by"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&cancelRequest); err != nil {
		// Use default values if parsing fails
		cancelRequest.Reason = "user_requested"
		cancelRequest.RequestedBy = "unknown"
	}
	
	// Cancel workflow
	if err := h.workflowOrchestrator.CancelWorkflow(r.Context(), workflowID, cancelRequest.Reason); err != nil {
		h.logger.Error("Failed to cancel workflow",
			zap.String("workflow_id", workflowID.String()),
			zap.Error(err),
		)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	h.logger.Info("Workflow cancelled",
		zap.String("workflow_id", workflowID.String()),
		zap.String("reason", cancelRequest.Reason),
		zap.String("requested_by", cancelRequest.RequestedBy),
	)
	
	response := map[string]interface{}{
		"success":     true,
		"message":     "Workflow cancelled successfully",
		"workflow_id": workflowID.String(),
		"timestamp":   time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ListActiveWorkflows lists all currently active workflows
func (h *WorkflowOrchestratorHandler) ListActiveWorkflows(w http.ResponseWriter, r *http.Request) {
	// Get query parameters for pagination
	limit := 50 // Default limit
	offset := 0
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}
	
	// Get active workflows
	activeWorkflowIDs := h.workflowOrchestrator.ListActiveWorkflows(r.Context())
	
	// Apply pagination
	total := len(activeWorkflowIDs)
	start := offset
	if start > total {
		start = total
	}
	
	end := start + limit
	if end > total {
		end = total
	}
	
	paginatedIDs := activeWorkflowIDs[start:end]
	
	// Convert to response format
	var workflows []map[string]interface{}
	for _, workflowID := range paginatedIDs {
		if state, err := h.workflowStateService.GetState(r.Context(), workflowID); err == nil {
			workflowInfo := map[string]interface{}{
				"workflow_id":   workflowID.String(),
				"patient_id":    state.PatientID,
				"status":        state.Status,
				"current_phase": state.CurrentPhase,
				"created_at":    state.CreatedAt,
				"updated_at":    state.UpdatedAt,
			}
			workflows = append(workflows, workflowInfo)
		}
	}
	
	response := map[string]interface{}{
		"workflows": workflows,
		"pagination": map[string]interface{}{
			"total":  total,
			"offset": offset,
			"limit":  limit,
			"count":  len(workflows),
		},
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// QueryWorkflows queries workflows based on criteria
func (h *WorkflowOrchestratorHandler) QueryWorkflows(w http.ResponseWriter, r *http.Request) {
	var query services.WorkflowStateQuery
	
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid query format", http.StatusBadRequest)
		return
	}
	
	// Set default values
	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 100 {
		query.Limit = 100
	}
	
	// Execute query
	states, total, err := h.workflowStateService.QueryStates(r.Context(), &query)
	if err != nil {
		h.logger.Error("Failed to query workflows", zap.Error(err))
		http.Error(w, "Query execution failed", http.StatusInternalServerError)
		return
	}
	
	// Convert to response format
	var workflows []map[string]interface{}
	for _, state := range states {
		workflowInfo := map[string]interface{}{
			"workflow_id":   state.WorkflowID.String(),
			"request_id":    state.RequestID,
			"patient_id":    state.PatientID,
			"status":        state.Status,
			"current_phase": state.CurrentPhase,
			"created_at":    state.CreatedAt,
			"updated_at":    state.UpdatedAt,
			"completed_at":  state.CompletedAt,
			"metadata":      state.Metadata,
		}
		workflows = append(workflows, workflowInfo)
	}
	
	response := map[string]interface{}{
		"workflows": workflows,
		"pagination": map[string]interface{}{
			"total":  total,
			"offset": query.Offset,
			"limit":  query.Limit,
			"count":  len(workflows),
		},
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetWorkflowMetrics gets workflow execution metrics
func (h *WorkflowOrchestratorHandler) GetWorkflowMetrics(w http.ResponseWriter, r *http.Request) {
	if h.metricsService == nil {
		http.Error(w, "Metrics service not available", http.StatusServiceUnavailable)
		return
	}
	
	// Get timeframe query parameter
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "1h" // Default to last hour
	}
	
	// Get metrics summary
	summary := h.metricsService.GetMetricsSummary()
	workflowMetrics := h.metricsService.GetWorkflowMetrics()
	
	response := map[string]interface{}{
		"summary":          summary,
		"workflow_metrics": workflowMetrics,
		"timeframe":        timeframe,
		"timestamp":        time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetPerformanceReport gets a comprehensive performance report
func (h *WorkflowOrchestratorHandler) GetPerformanceReport(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the performance monitor
	// For now, return basic performance data from metrics
	if h.metricsService == nil {
		http.Error(w, "Metrics service not available", http.StatusServiceUnavailable)
		return
	}
	
	summary := h.metricsService.GetMetricsSummary()
	performanceMetrics := h.metricsService.GetPerformanceMetrics()
	
	report := map[string]interface{}{
		"generated_at":        time.Now(),
		"overall_health":      summary.SystemHealth,
		"success_rate":        summary.SuccessRate,
		"average_latency":     summary.AverageLatency.String(),
		"throughput_rps":      summary.ThroughputRPS,
		"error_rate":          summary.ErrorRate,
		"performance_metrics": performanceMetrics,
		"recommendations":     h.generatePerformanceRecommendations(summary),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(report)
}

// GetWorkflowPerformance gets performance data for a specific workflow
func (h *WorkflowOrchestratorHandler) GetWorkflowPerformance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowIDStr := vars["workflowId"]
	
	workflowID, err := uuid.Parse(workflowIDStr)
	if err != nil {
		http.Error(w, "Invalid workflow ID format", http.StatusBadRequest)
		return
	}
	
	// Get workflow state for performance data
	state, err := h.workflowStateService.GetState(r.Context(), workflowID)
	if err != nil {
		http.Error(w, "Workflow not found", http.StatusNotFound)
		return
	}
	
	// Build performance response from state
	performance := map[string]interface{}{
		"workflow_id": workflowID.String(),
		"status":      state.Status,
		"created_at":  state.CreatedAt,
		"updated_at":  state.UpdatedAt,
	}
	
	if state.ExecutionContext != nil {
		performance["performance_data"] = state.ExecutionContext.PerformanceData
		performance["resource_usage"] = state.ExecutionContext.ResourceUsage
		performance["retry_count"] = state.ExecutionContext.RetryCount
		performance["error_count"] = len(state.ExecutionContext.ErrorHistory)
		performance["warning_count"] = len(state.ExecutionContext.WarningHistory)
	}
	
	if state.PhaseResults != nil {
		phasePerformance := make(map[string]interface{})
		for phase, result := range state.PhaseResults {
			phasePerformance[fmt.Sprintf("phase_%d", phase)] = map[string]interface{}{
				"status":       result.Status,
				"duration":     result.Duration.String(),
				"quality_score": result.QualityScore,
				"start_time":   result.StartTime,
				"end_time":     result.EndTime,
				"metrics":      result.Metrics,
			}
		}
		performance["phase_performance"] = phasePerformance
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(performance)
}

// GetStatistics gets workflow statistics
func (h *WorkflowOrchestratorHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	// Get statistics from state service
	stats, err := h.workflowStateService.GetStatistics(r.Context())
	if err != nil {
		h.logger.Error("Failed to get workflow statistics", zap.Error(err))
		http.Error(w, "Failed to get statistics", http.StatusInternalServerError)
		return
	}
	
	// Combine with metrics service data if available
	response := map[string]interface{}{
		"workflow_statistics": stats,
		"timestamp":           time.Now(),
	}
	
	if h.metricsService != nil {
		summary := h.metricsService.GetMetricsSummary()
		response["metrics_summary"] = summary
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// HealthCheck performs a health check on the workflow system
func (h *WorkflowOrchestratorHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"components": map[string]interface{}{
			"workflow_orchestrator": "healthy",
			"workflow_state_service": "healthy",
		},
	}
	
	// Check workflow state service
	if !h.workflowStateService.IsHealthy(r.Context()) {
		health["status"] = "degraded"
		health["components"].(map[string]interface{})["workflow_state_service"] = "unhealthy"
	}
	
	// Check metrics service
	if h.metricsService != nil {
		if h.metricsService.IsHealthy() {
			health["components"].(map[string]interface{})["metrics_service"] = "healthy"
		} else {
			health["status"] = "degraded"
			health["components"].(map[string]interface{})["metrics_service"] = "unhealthy"
		}
	}
	
	// Set HTTP status based on overall health
	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// Helper methods

func (h *WorkflowOrchestratorHandler) validateWorkflowRequest(request *services.WorkflowExecutionRequest) error {
	if request.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}
	
	if request.RecipeID == "" {
		return fmt.Errorf("recipe_id is required")
	}
	
	if request.RequestedBy == "" {
		return fmt.Errorf("requested_by is required")
	}
	
	return nil
}

func (h *WorkflowOrchestratorHandler) getErrorStatusCode(err error) int {
	// Map different error types to appropriate HTTP status codes
	errStr := err.Error()
	
	if contains(errStr, "timeout") || contains(errStr, "context deadline exceeded") {
		return http.StatusRequestTimeout
	}
	
	if contains(errStr, "not found") || contains(errStr, "workflow not found") {
		return http.StatusNotFound
	}
	
	if contains(errStr, "validation") || contains(errStr, "invalid") {
		return http.StatusBadRequest
	}
	
	if contains(errStr, "service unavailable") || contains(errStr, "connection refused") {
		return http.StatusServiceUnavailable
	}
	
	return http.StatusInternalServerError
}

func (h *WorkflowOrchestratorHandler) buildWorkflowResultFromState(state *services.WorkflowState) map[string]interface{} {
	result := map[string]interface{}{
		"workflow_id": state.WorkflowID.String(),
		"request_id":  state.RequestID,
		"patient_id":  state.PatientID,
		"status":      state.Status,
		"created_at":  state.CreatedAt,
		"updated_at":  state.UpdatedAt,
		"completed_at": state.CompletedAt,
		"metadata":    state.Metadata,
	}
	
	if state.PhaseResults != nil {
		phases := make(map[string]interface{})
		for phase, phaseResult := range state.PhaseResults {
			phases[fmt.Sprintf("phase_%d", phase)] = phaseResult
		}
		result["phases"] = phases
	}
	
	if state.ExecutionContext != nil {
		result["execution_context"] = map[string]interface{}{
			"start_time":        state.ExecutionContext.StartTime,
			"last_activity":     state.ExecutionContext.LastActivity,
			"retry_count":       state.ExecutionContext.RetryCount,
			"error_count":       len(state.ExecutionContext.ErrorHistory),
			"warning_count":     len(state.ExecutionContext.WarningHistory),
			"performance_data":  state.ExecutionContext.PerformanceData,
			"resource_usage":    state.ExecutionContext.ResourceUsage,
		}
		
		// Include errors and warnings if present
		if len(state.ExecutionContext.ErrorHistory) > 0 {
			result["errors"] = state.ExecutionContext.ErrorHistory
		}
		if len(state.ExecutionContext.WarningHistory) > 0 {
			result["warnings"] = state.ExecutionContext.WarningHistory
		}
	}
	
	return result
}

func (h *WorkflowOrchestratorHandler) generatePerformanceRecommendations(summary *services.MetricsSummary) []string {
	var recommendations []string
	
	if summary.SuccessRate < 0.90 {
		recommendations = append(recommendations, "Consider investigating workflow failures to improve success rate")
	}
	
	if summary.AverageLatency > 250*time.Millisecond {
		recommendations = append(recommendations, "Average latency exceeds target (250ms) - consider performance optimization")
	}
	
	if summary.ErrorRate > 0.10 {
		recommendations = append(recommendations, "High error rate detected - review error patterns and implement fixes")
	}
	
	if summary.ThroughputRPS < 10 {
		recommendations = append(recommendations, "Low throughput - consider scaling or performance improvements")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System performance is within acceptable parameters")
	}
	
	return recommendations
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}