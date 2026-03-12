package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/orchestration"
	"safety-gateway-platform/internal/validator"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// HTTPServer provides REST endpoints for the Safety Gateway
type HTTPServer struct {
	config       *config.Config
	logger       *logger.Logger
	validator    *validator.IngressValidator
	orchestrator types.SafetyOrchestrator  // Interface to support both basic and advanced
	server       *http.Server
	router       *mux.Router
}

// ValidationRequest represents the HTTP request format for validation
type ValidationRequest struct {
	ProposalSetID          string                 `json:"proposal_set_id"`
	SnapshotID            string                 `json:"snapshot_id"`
	Proposals             []map[string]interface{} `json:"proposals"`
	PatientContext        map[string]interface{} `json:"patient_context"`
	ValidationRequirements map[string]interface{} `json:"validation_requirements"`
	CorrelationID         string                 `json:"correlation_id"`
	RequestID             string                 `json:"request_id,omitempty"`
	PatientID             string                 `json:"patient_id,omitempty"`
	ClinicianID           string                 `json:"clinician_id,omitempty"`
	Priority              string                 `json:"priority,omitempty"`
	Source                string                 `json:"source,omitempty"`
}

// ValidationResponse represents the HTTP response format for validation
type ValidationResponse struct {
	ValidationID       string                   `json:"validation_id"`
	Verdict           string                   `json:"verdict"` // SAFE, WARNING, UNSAFE
	Findings          []Finding               `json:"findings"`
	OverrideTokens    []string                `json:"override_tokens,omitempty"`
	OverrideRequirements map[string]interface{} `json:"override_requirements,omitempty"`
	ProcessingTimeMS  int64                   `json:"processing_time_ms"`
	EngineResults     []EngineResult          `json:"engine_results"`
	RiskScore         float64                 `json:"risk_score"`
	Timestamp         time.Time               `json:"timestamp"`
	Metadata          map[string]interface{}  `json:"metadata,omitempty"`
}

// Finding represents a validation finding
type Finding struct {
	FindingID             string  `json:"finding_id"`
	Severity              string  `json:"severity"`
	Category              string  `json:"category"`
	Description           string  `json:"description"`
	ClinicalSignificance  string  `json:"clinical_significance"`
	Recommendation        string  `json:"recommendation"`
	ConfidenceScore       float64 `json:"confidence_score"`
	EngineSource          string  `json:"engine_source"`
}

// EngineResult represents individual engine validation results
type EngineResult struct {
	EngineID     string   `json:"engine_id"`
	EngineName   string   `json:"engine_name"`
	Status       string   `json:"status"`
	RiskScore    float64  `json:"risk_score"`
	Violations   []string `json:"violations"`
	Warnings     []string `json:"warnings"`
	Confidence   float64  `json:"confidence"`
	DurationMS   int64    `json:"duration_ms"`
	Tier         int      `json:"tier"`
	Error        string   `json:"error,omitempty"`
}

// BatchValidationRequest represents a batch validation request
type BatchValidationRequest struct {
	BatchID      string              `json:"batch_id,omitempty"`
	Requests     []ValidationRequest `json:"requests"`
	Priority     string              `json:"priority,omitempty"`
	Options      *BatchOptions       `json:"options,omitempty"`
	CorrelationID string             `json:"correlation_id,omitempty"`
}

// BatchOptions configures batch processing behavior
type BatchOptions struct {
	EnablePatientGrouping    bool `json:"enable_patient_grouping,omitempty"`
	EnableSnapshotOptimization bool `json:"enable_snapshot_optimization,omitempty"`
	MaxConcurrency          int  `json:"max_concurrency,omitempty"`
	TimeoutSeconds          int  `json:"timeout_seconds,omitempty"`
}

// BatchValidationResponse represents a batch validation response
type BatchValidationResponse struct {
	BatchID       string                `json:"batch_id"`
	Responses     []ValidationResponse  `json:"responses"`
	Summary       *BatchSummary         `json:"summary"`
	ProcessedAt   time.Time            `json:"processed_at"`
	TotalDuration int64                `json:"total_duration_ms"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// BatchSummary provides aggregate statistics for the batch
type BatchSummary struct {
	TotalRequests     int     `json:"total_requests"`
	SuccessfulResults int     `json:"successful_results"`
	ErrorResults      int     `json:"error_results"`
	WarningResults    int     `json:"warning_results"`
	UnsafeResults     int     `json:"unsafe_results"`
	CacheHitCount     int     `json:"cache_hit_count"`
	AverageRiskScore  float64 `json:"average_risk_score"`
	ProcessingStats   *ProcessingStatistics `json:"processing_stats"`
}

// ProcessingStatistics contains detailed processing metrics
type ProcessingStatistics struct {
	SnapshotRetrievals   int           `json:"snapshot_retrievals"`
	CacheHits            int           `json:"cache_hits"`
	CacheMisses          int           `json:"cache_misses"`
	EngineExecutions     int           `json:"engine_executions"`
	AverageEngineLatency time.Duration `json:"average_engine_latency"`
	ParallelismAchieved  float64       `json:"parallelism_achieved"`
	ResourceUtilization  map[string]float64 `json:"resource_utilization"`
}

// OrchestrationStatsResponse represents orchestration statistics
type OrchestrationStatsResponse struct {
	TotalRequests        int64                  `json:"total_requests"`
	BatchedRequests      int64                  `json:"batched_requests"`
	AverageResponseTimeMS int64                 `json:"average_response_time_ms"`
	LoadBalancingDecisions int64                `json:"load_balancing_decisions"`
	RoutingDecisions     int64                  `json:"routing_decisions"`
	EngineUtilization    map[string]float64     `json:"engine_utilization"`
	EngineMetrics        map[string]interface{} `json:"engine_metrics"`
	RoutingStrategy      string                 `json:"routing_strategy"`
	SnapshotStats        map[string]interface{} `json:"snapshot_stats,omitempty"`
}

// NewHTTPServer creates a new HTTP server for Safety Gateway REST API
func NewHTTPServer(
	config *config.Config,
	logger *logger.Logger,
	validator *validator.IngressValidator,
	orchestrator types.SafetyOrchestrator,
) *HTTPServer {
	router := mux.NewRouter()
	
	httpServer := &HTTPServer{
		config:       config,
		logger:       logger,
		validator:    validator,
		orchestrator: orchestrator,
		router:       router,
	}
	
	// Setup routes
	httpServer.setupRoutes()
	
	// Create HTTP server
	httpServer.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Service.HTTPPort),
		Handler:      httpServer.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return httpServer
}

// setupRoutes configures the HTTP routes
func (h *HTTPServer) setupRoutes() {
	// API v1 routes
	v1 := h.router.PathPrefix("/api/v1").Subrouter()
	
	// Validation endpoints
	v1.HandleFunc("/validate", h.validateHandler).Methods("POST")
	v1.HandleFunc("/validate/comprehensive", h.comprehensiveValidateHandler).Methods("POST")
	
	// Phase 2: Batch processing endpoints
	v1.HandleFunc("/batch/validate", h.batchValidateHandler).Methods("POST")
	v1.HandleFunc("/batch/{batch_id}/status", h.getBatchStatusHandler).Methods("GET")
	
	// Phase 2: Advanced orchestration endpoints
	v1.HandleFunc("/orchestration/stats", h.getOrchestrationStatsHandler).Methods("GET")
	v1.HandleFunc("/orchestration/metrics", h.getOrchestrationMetricsHandler).Methods("GET")
	
	// Health and status endpoints
	v1.HandleFunc("/health", h.healthHandler).Methods("GET")
	v1.HandleFunc("/health/orchestration", h.orchestrationHealthHandler).Methods("GET")
	v1.HandleFunc("/engines/status", h.engineStatusHandler).Methods("GET")
	
	// Override endpoints
	v1.HandleFunc("/override/validate", h.validateOverrideHandler).Methods("POST")
	
	// Middleware
	h.router.Use(h.loggingMiddleware)
	h.router.Use(h.corsMiddleware)
	h.router.Use(h.contentTypeMiddleware)
}

// validateHandler handles comprehensive safety validation requests
func (h *HTTPServer) validateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()
	
	// Parse request
	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Failed to parse request: %v", err))
		return
	}
	
	// Generate request ID if not provided
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("http_%d", time.Now().UnixNano())
	}
	
	requestLogger := h.logger.WithRequestID(req.RequestID)
	requestLogger.Info("HTTP validation request received",
		zap.String("proposal_set_id", req.ProposalSetID),
		zap.String("snapshot_id", req.SnapshotID),
		zap.String("correlation_id", req.CorrelationID),
	)
	
	// Convert to internal safety request
	safetyReq, err := h.convertToSafetyRequest(&req)
	if err != nil {
		requestLogger.Error("Failed to convert request", zap.Error(err))
		h.errorResponse(w, http.StatusBadRequest, "conversion_error", fmt.Sprintf("Request conversion failed: %v", err))
		return
	}
	
	// Validate request
	if err := h.validator.ValidateRequest(ctx, safetyReq); err != nil {
		requestLogger.Warn("Request validation failed", zap.Error(err))
		h.errorResponse(w, http.StatusBadRequest, "validation_failed", fmt.Sprintf("Request validation failed: %v", err))
		return
	}
	
	// Process through orchestrator
	response, err := h.orchestrator.ProcessSafetyRequest(ctx, safetyReq)
	if err != nil {
		requestLogger.Error("Safety processing failed", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "processing_error", fmt.Sprintf("Safety processing failed: %v", err))
		return
	}
	
	// Convert to HTTP response format
	httpResponse := h.convertToHTTPResponse(response, req.RequestID, time.Since(startTime))
	
	requestLogger.Info("HTTP validation completed",
		zap.String("verdict", httpResponse.Verdict),
		zap.Float64("risk_score", httpResponse.RiskScore),
		zap.Int64("processing_time_ms", httpResponse.ProcessingTimeMS),
		zap.Int("findings_count", len(httpResponse.Findings)),
	)
	
	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(httpResponse)
}

// comprehensiveValidateHandler handles comprehensive validation with detailed analysis
func (h *HTTPServer) comprehensiveValidateHandler(w http.ResponseWriter, r *http.Request) {
	// Use the same logic as validateHandler but potentially with different validation requirements
	h.validateHandler(w, r)
}

// healthHandler returns health status
func (h *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"service":   "safety-gateway-platform",
		"version":   h.config.Service.Version,
		"timestamp": time.Now().Format(time.RFC3339),
		"endpoints": map[string]string{
			"validate":                "/api/v1/validate",
			"comprehensive_validate":  "/api/v1/validate/comprehensive",
			"engine_status":          "/api/v1/engines/status",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(health)
}

// engineStatusHandler returns engine status information
func (h *HTTPServer) engineStatusHandler(w http.ResponseWriter, r *http.Request) {
	engines := []map[string]interface{}{
		{
			"id":           "cae_engine",
			"name":         "Clinical Assertion Engine",
			"capabilities": []string{"drug_interaction", "contraindication", "dosing"},
			"tier":         1,
			"priority":     10,
			"timeout_ms":   100000,
			"status":       "healthy",
			"last_check":   time.Now().Format(time.RFC3339),
			"failure_count": 0,
		},
		{
			"id":           "allergy_engine", 
			"name":         "Allergy Check Engine",
			"capabilities": []string{"allergy_check", "contraindication"},
			"tier":         1,
			"priority":     9,
			"status":       "healthy",
			"last_check":   time.Now().Format(time.RFC3339),
			"failure_count": 0,
		},
	}
	
	response := map[string]interface{}{
		"engines": engines,
		"metadata": map[string]interface{}{
			"total_engines":   len(engines),
			"healthy_engines": len(engines),
			"timestamp":       time.Now().Format(time.RFC3339),
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// validateOverrideHandler validates override tokens
func (h *HTTPServer) validateOverrideHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TokenID     string `json:"token_id"`
		ClinicianID string `json:"clinician_id"`
		Reason      string `json:"reason"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Failed to parse request: %v", err))
		return
	}
	
	h.logger.Info("Override validation requested",
		zap.String("token_id", req.TokenID),
		zap.String("clinician_id", req.ClinicianID),
		zap.String("reason", req.Reason),
	)
	
	// Mock override validation - in production, implement proper validation logic
	response := map[string]interface{}{
		"valid":        true,
		"reason":       "Override validated successfully",
		"clinician_id": req.ClinicianID,
		"validated_at": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// convertToSafetyRequest converts HTTP request to internal safety request
func (h *HTTPServer) convertToSafetyRequest(req *ValidationRequest) (*types.SafetyRequest, error) {
	// Extract medication IDs from proposals if available
	var medicationIDs []string
	for _, proposal := range req.Proposals {
		if medID, exists := proposal["medication_id"].(string); exists {
			medicationIDs = append(medicationIDs, medID)
		}
	}
	
	return &types.SafetyRequest{
		RequestID:     req.RequestID,
		PatientID:     req.PatientID,
		ClinicianID:   req.ClinicianID,
		ActionType:    "medication_validation",
		Priority:      req.Priority,
		MedicationIDs: medicationIDs,
		ConditionIDs:  []string{}, // Could be extracted from patient context
		AllergyIDs:    []string{}, // Could be extracted from patient context
		Context: map[string]string{
			"proposal_set_id":    req.ProposalSetID,
			"snapshot_id":        req.SnapshotID,
			"correlation_id":     req.CorrelationID,
			"validation_type":    "comprehensive",
		},
		Timestamp: time.Now(),
		Source:    req.Source,
	}, nil
}

// convertToHTTPResponse converts internal response to HTTP format
func (h *HTTPServer) convertToHTTPResponse(resp *types.SafetyResponse, requestID string, processingTime time.Duration) *ValidationResponse {
	// Convert findings
	var findings []Finding
	for _, result := range resp.EngineResults {
		for _, violation := range result.Violations {
			findings = append(findings, Finding{
				FindingID:            fmt.Sprintf("%s_%d", result.EngineID, len(findings)+1),
				Severity:             "HIGH",
				Category:             "SAFETY_VIOLATION",
				Description:          violation,
				ClinicalSignificance: "Requires clinical review",
				Recommendation:       "Review medication selection and dosing",
				ConfidenceScore:      result.Confidence,
				EngineSource:         result.EngineID,
			})
		}
		
		for _, warning := range result.Warnings {
			findings = append(findings, Finding{
				FindingID:            fmt.Sprintf("%s_%d", result.EngineID, len(findings)+1),
				Severity:             "MEDIUM",
				Category:             "SAFETY_WARNING",
				Description:          warning,
				ClinicalSignificance: "Clinical consideration recommended",
				Recommendation:       "Monitor patient response",
				ConfidenceScore:      result.Confidence,
				EngineSource:         result.EngineID,
			})
		}
	}
	
	// Convert engine results
	var engineResults []EngineResult
	for _, result := range resp.EngineResults {
		engineResults = append(engineResults, EngineResult{
			EngineID:   result.EngineID,
			EngineName: result.EngineName,
			Status:     string(result.Status),
			RiskScore:  result.RiskScore,
			Violations: result.Violations,
			Warnings:   result.Warnings,
			Confidence: result.Confidence,
			DurationMS: result.Duration.Milliseconds(),
			Tier:       result.Tier,
			Error:      result.Error,
		})
	}
	
	// Determine verdict based on safety status
	var verdict string
	switch resp.Status {
	case types.SafetyStatusSafe:
		verdict = "SAFE"
	case types.SafetyStatusWarning:
		verdict = "WARNING"
	case types.SafetyStatusUnsafe:
		verdict = "UNSAFE"
	case types.SafetyStatusManualReview:
		verdict = "MANUAL_REVIEW"
	case types.SafetyStatusError:
		verdict = "ERROR"
	default:
		verdict = "UNKNOWN"
	}
	
	// Extract override tokens if available
	var overrideTokens []string
	if resp.OverrideToken != nil {
		overrideTokens = append(overrideTokens, resp.OverrideToken.TokenID)
	}
	
	return &ValidationResponse{
		ValidationID:         resp.RequestID,
		Verdict:             verdict,
		Findings:            findings,
		OverrideTokens:      overrideTokens,
		OverrideRequirements: nil, // Could be populated from override token requirements
		ProcessingTimeMS:    processingTime.Milliseconds(),
		EngineResults:       engineResults,
		RiskScore:           resp.RiskScore,
		Timestamp:           resp.Timestamp,
		Metadata: map[string]interface{}{
			"engines_executed": len(resp.EngineResults),
			"engines_failed":   resp.EnginesFailed,
			"context_version":  resp.ContextVersion,
		},
	}
}

// Phase 2: Batch Processing Handlers

// batchValidateHandler handles batch validation requests
func (h *HTTPServer) batchValidateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now()
	
	// Parse batch request
	var req BatchValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("Failed to parse batch request: %v", err))
		return
	}
	
	// Generate batch ID if not provided
	if req.BatchID == "" {
		req.BatchID = fmt.Sprintf("batch_%d", time.Now().UnixNano())
	}
	
	h.logger.Info("HTTP batch validation request received",
		zap.String("batch_id", req.BatchID),
		zap.Int("request_count", len(req.Requests)),
		zap.String("priority", req.Priority),
		zap.String("correlation_id", req.CorrelationID),
	)
	
	// Check if advanced orchestration is available for batch processing
	advancedOrch, ok := h.orchestrator.(interface{
		ProcessBatchRequests(ctx context.Context, requests []*types.SafetyRequest) (*orchestration.BatchProcessingResult, error)
	})
	
	if !ok {
		h.logger.Warn("Batch processing not supported by current orchestrator, processing individually")
		h.processBatchSequentially(w, r, &req, startTime)
		return
	}
	
	// Convert HTTP requests to internal safety requests
	var safetyRequests []*types.SafetyRequest
	for i, validationReq := range req.Requests {
		safetyReq, err := h.convertToSafetyRequest(&validationReq)
		if err != nil {
			h.logger.Error("Failed to convert batch request item", 
				zap.Int("item_index", i), 
				zap.Error(err))
			h.errorResponse(w, http.StatusBadRequest, "conversion_error", 
				fmt.Sprintf("Failed to convert request item %d: %v", i, err))
			return
		}
		safetyRequests = append(safetyRequests, safetyReq)
	}
	
	// Process batch through advanced orchestrator
	batchResult, err := advancedOrch.ProcessBatchRequests(ctx, safetyRequests)
	if err != nil {
		h.logger.Error("Batch processing failed", zap.Error(err))
		h.errorResponse(w, http.StatusInternalServerError, "batch_processing_error", 
			fmt.Sprintf("Batch processing failed: %v", err))
		return
	}
	
	// Convert batch result to HTTP response format
	httpResponse := h.convertBatchResultToHTTPResponse(batchResult, req.BatchID, time.Since(startTime))
	
	h.logger.Info("HTTP batch validation completed",
		zap.String("batch_id", req.BatchID),
		zap.Int("total_requests", httpResponse.Summary.TotalRequests),
		zap.Int("successful_results", httpResponse.Summary.SuccessfulResults),
		zap.Int("cache_hits", httpResponse.Summary.CacheHitCount),
		zap.Int64("total_duration_ms", httpResponse.TotalDuration),
	)
	
	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(httpResponse)
}

// processBatchSequentially handles batch processing for non-advanced orchestrators
func (h *HTTPServer) processBatchSequentially(w http.ResponseWriter, r *http.Request, req *BatchValidationRequest, startTime time.Time) {
	ctx := r.Context()
	responses := make([]ValidationResponse, 0, len(req.Requests))
	
	var successCount, errorCount, warningCount, unsafeCount int
	var totalRiskScore float64
	
	// Process each request individually
	for i, validationReq := range req.Requests {
		safetyReq, err := h.convertToSafetyRequest(&validationReq)
		if err != nil {
			// Create error response for this item
			errorResponse := ValidationResponse{
				ValidationID:     fmt.Sprintf("%s_item_%d", req.BatchID, i),
				Verdict:         "ERROR",
				ProcessingTimeMS: 0,
				Timestamp:       time.Now(),
				Metadata: map[string]interface{}{
					"error": fmt.Sprintf("Request conversion failed: %v", err),
				},
			}
			responses = append(responses, errorResponse)
			errorCount++
			continue
		}
		
		// Process individual request
		response, err := h.orchestrator.ProcessSafetyRequest(ctx, safetyReq)
		if err != nil {
			errorResponse := ValidationResponse{
				ValidationID:     safetyReq.RequestID,
				Verdict:         "ERROR",
				ProcessingTimeMS: 0,
				Timestamp:       time.Now(),
				Metadata: map[string]interface{}{
					"error": fmt.Sprintf("Processing failed: %v", err),
				},
			}
			responses = append(responses, errorResponse)
			errorCount++
			continue
		}
		
		// Convert to HTTP response format
		httpResp := h.convertToHTTPResponse(response, safetyReq.RequestID, response.ProcessingTime)
		responses = append(responses, *httpResp)
		
		// Update counters
		switch httpResp.Verdict {
		case "SAFE":
			successCount++
		case "WARNING":
			warningCount++
		case "UNSAFE":
			unsafeCount++
		case "ERROR":
			errorCount++
		}
		totalRiskScore += httpResp.RiskScore
	}
	
	// Calculate average risk score
	var averageRiskScore float64
	if len(responses) > 0 {
		averageRiskScore = totalRiskScore / float64(len(responses))
	}
	
	// Create batch response
	batchResponse := BatchValidationResponse{
		BatchID:       req.BatchID,
		Responses:     responses,
		ProcessedAt:   startTime,
		TotalDuration: time.Since(startTime).Milliseconds(),
		Summary: &BatchSummary{
			TotalRequests:     len(req.Requests),
			SuccessfulResults: successCount,
			ErrorResults:      errorCount,
			WarningResults:    warningCount,
			UnsafeResults:     unsafeCount,
			CacheHitCount:     0, // Not available in sequential processing
			AverageRiskScore:  averageRiskScore,
		},
		Metadata: map[string]interface{}{
			"processing_mode": "sequential",
			"orchestrator_type": "basic",
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(batchResponse)
}

// getBatchStatusHandler handles batch status queries (placeholder for future async processing)
func (h *HTTPServer) getBatchStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	batchID := vars["batch_id"]
	
	h.logger.Debug("Batch status requested", zap.String("batch_id", batchID))
	
	// For now, return a simple response indicating synchronous processing
	status := map[string]interface{}{
		"batch_id": batchID,
		"status":   "completed", // All batches are processed synchronously currently
		"message":  "Batch processing is currently synchronous",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(status)
}

// getOrchestrationStatsHandler returns advanced orchestration statistics
func (h *HTTPServer) getOrchestrationStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if advanced orchestration is available
	advancedOrch, ok := h.orchestrator.(interface{
		GetOrchestrationStats() map[string]interface{}
	})
	
	if !ok {
		h.errorResponse(w, http.StatusNotImplemented, "advanced_orchestration_not_available", 
			"Advanced orchestration statistics not available")
		return
	}
	
	stats := advancedOrch.GetOrchestrationStats()
	
	// Convert to HTTP response format
	response := OrchestrationStatsResponse{
		TotalRequests:          stats["total_requests"].(int64),
		BatchedRequests:        stats["batched_requests"].(int64),
		AverageResponseTimeMS:  stats["average_response_time_ms"].(int64),
		LoadBalancingDecisions: stats["load_balancing_decisions"].(int64),
		RoutingDecisions:       stats["routing_decisions"].(int64),
		EngineUtilization:      stats["engine_utilization"].(map[string]float64),
		EngineMetrics:          stats["engine_metrics"].(map[string]interface{}),
		RoutingStrategy:        stats["routing_strategy"].(string),
	}
	
	if snapshotStats, exists := stats["snapshot_stats"]; exists {
		response.SnapshotStats = snapshotStats.(map[string]interface{})
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// getOrchestrationMetricsHandler returns detailed orchestration metrics
func (h *HTTPServer) getOrchestrationMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if metrics collector is available
	metricsOrch, ok := h.orchestrator.(interface{
		GetComprehensiveMetrics() map[string]interface{}
	})
	
	if !ok {
		h.errorResponse(w, http.StatusNotImplemented, "metrics_not_available", 
			"Comprehensive metrics not available")
		return
	}
	
	metrics := metricsOrch.GetComprehensiveMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// orchestrationHealthHandler returns orchestration health status
func (h *HTTPServer) orchestrationHealthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	// Check if advanced orchestration is available and get its health
	if advancedOrch, ok := h.orchestrator.(interface{
		GetOrchestrationStats() map[string]interface{}
	}); ok {
		stats := advancedOrch.GetOrchestrationStats()
		health["orchestration_type"] = "advanced"
		health["total_requests"] = stats["total_requests"]
		health["average_response_time_ms"] = stats["average_response_time_ms"]
		
		// Determine health based on metrics
		if avgTime, ok := stats["average_response_time_ms"].(int64); ok {
			if avgTime > 2000 { // > 2 seconds
				health["status"] = "degraded"
				health["reason"] = "High average response time"
			}
		}
	} else {
		health["orchestration_type"] = "basic"
	}
	
	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(health)
}

// convertBatchResultToHTTPResponse converts batch processing result to HTTP format
func (h *HTTPServer) convertBatchResultToHTTPResponse(result *orchestration.BatchProcessingResult, batchID string, duration time.Duration) *BatchValidationResponse {
	responses := make([]ValidationResponse, len(result.Responses))
	
	for i, resp := range result.Responses {
		httpResp := h.convertToHTTPResponse(resp, resp.RequestID, resp.ProcessingTime)
		responses[i] = *httpResp
	}
	
	// Convert processing statistics if available
	var processingStats *ProcessingStatistics
	if result.Summary != nil && result.Summary.ProcessingStats != nil {
		processingStats = &ProcessingStatistics{
			SnapshotRetrievals:   result.Summary.ProcessingStats.SnapshotRetrievals,
			CacheHits:            result.Summary.ProcessingStats.CacheHits,
			CacheMisses:          result.Summary.ProcessingStats.CacheMisses,
			EngineExecutions:     result.Summary.ProcessingStats.EngineExecutions,
			AverageEngineLatency: result.Summary.ProcessingStats.AverageEngineLatency,
			ParallelismAchieved:  result.Summary.ProcessingStats.ParallelismAchieved,
			ResourceUtilization:  result.Summary.ProcessingStats.ResourceUtilization,
		}
	}
	
	summary := &BatchSummary{
		TotalRequests:     result.Summary.TotalRequests,
		SuccessfulResults: result.Summary.SuccessfulResults,
		ErrorResults:      result.Summary.ErrorResults,
		WarningResults:    result.Summary.WarningResults,
		UnsafeResults:     result.Summary.UnsafeResults,
		CacheHitCount:     result.Summary.CacheHitCount,
		AverageRiskScore:  result.Summary.AverageRiskScore,
		ProcessingStats:   processingStats,
	}
	
	return &BatchValidationResponse{
		BatchID:       batchID,
		Responses:     responses,
		Summary:       summary,
		ProcessedAt:   result.ProcessedAt,
		TotalDuration: duration.Milliseconds(),
		Metadata: map[string]interface{}{
			"processing_mode": "advanced_batch",
			"orchestrator_type": "advanced",
		},
	}
}

// Start starts the HTTP server
func (h *HTTPServer) Start() error {
	h.logger.Info("Starting Safety Gateway HTTP server",
		zap.String("address", h.server.Addr),
	)
	
	return h.server.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	h.logger.Info("Shutting down Safety Gateway HTTP server")
	return h.server.Shutdown(ctx)
}

// Middleware functions

func (h *HTTPServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		h.logger.Info("HTTP request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Duration("duration", time.Since(start)),
		)
	})
}

func (h *HTTPServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (h *HTTPServer) contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.Header.Get("Content-Type") != "application/json" {
			h.errorResponse(w, http.StatusBadRequest, "invalid_content_type", "Content-Type must be application/json")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// errorResponse sends standardized error responses
func (h *HTTPServer) errorResponse(w http.ResponseWriter, statusCode int, errorCode, message string) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    errorCode,
			"message": message,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}