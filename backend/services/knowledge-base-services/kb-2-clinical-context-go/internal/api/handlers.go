package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-2-clinical-context-go/internal/models"
)

// Health check endpoint
func (s *Server) healthCheck(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("GET", "/health")
	defer timer.ObserveDuration()
	
	// Perform health checks
	checks := make(map[string]models.Check)
	
	// MongoDB health check
	mongoStart := time.Now()
	mongoErr := s.config.mongoClient.Ping(c.Request.Context(), nil)
	mongoCheck := models.Check{
		Status:      "healthy",
		ResponseTime: time.Since(mongoStart),
		Message:     "MongoDB connection successful",
	}
	if mongoErr != nil {
		mongoCheck.Status = "unhealthy"
		mongoCheck.Message = fmt.Sprintf("MongoDB error: %v", mongoErr)
	}
	checks["mongodb"] = mongoCheck
	
	// Redis health check
	redisStart := time.Now()
	redisErr := s.config.redisClient.Ping(c.Request.Context()).Err()
	redisCheck := models.Check{
		Status:      "healthy",
		ResponseTime: time.Since(redisStart),
		Message:     "Redis connection successful",
	}
	if redisErr != nil {
		redisCheck.Status = "unhealthy"
		redisCheck.Message = fmt.Sprintf("Redis error: %v", redisErr)
	}
	checks["redis"] = redisCheck
	
	// CEL engine health check
	celCheck := models.Check{
		Status:  "healthy",
		Message: "CEL engine operational",
	}
	if s.phenotypeEngine == nil {
		celCheck.Status = "unhealthy"
		celCheck.Message = "CEL engine not initialized"
	}
	checks["cel_engine"] = celCheck
	
	// Overall status
	status := "healthy"
	if mongoErr != nil || redisErr != nil || s.phenotypeEngine == nil {
		status = "degraded"
	}
	
	health := models.HealthStatus{
		Status:      status,
		Timestamp:   time.Now(),
		Version:     "1.0.0",
		Environment: s.config.Config.Environment,
		Checks:      checks,
	}
	
	statusCode := http.StatusOK
	if status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	s.metrics.RecordRequest("GET", "/health", strconv.Itoa(statusCode))
	c.JSON(statusCode, health)
}

// Batch phenotype evaluation endpoint
func (s *Server) evaluatePhenotypes(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("POST", "/v1/phenotypes/evaluate")
	defer timer.ObserveDuration()
	
	var request models.PhenotypeEvaluationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.handleValidationError(c, err, "Invalid phenotype evaluation request")
		return
	}
	
	// Validate batch size
	if len(request.Patients) > s.config.Config.BatchSize {
		s.handleValidationError(c, nil, fmt.Sprintf("Batch size %d exceeds maximum %d", len(request.Patients), s.config.Config.BatchSize))
		return
	}
	
	// Validate patients
	if len(request.Patients) == 0 {
		s.handleValidationError(c, nil, "No patients provided for evaluation")
		return
	}
	
	// Perform phenotype evaluation
	startTime := time.Now()
	results, err := s.phenotypeEngine.EvaluatePhenotypes(c.Request.Context(), &request)
	evaluationDuration := time.Since(startTime)
	
	if err != nil {
		s.metrics.RecordPhenotypeEvaluation("error", "error", evaluationDuration)
		s.handleInternalError(c, err, "Phenotype evaluation failed")
		return
	}
	
	// Record metrics
	s.metrics.RecordPhenotypeEvaluation("batch", "success", evaluationDuration)
	s.metrics.RecordBatchSize(float64(len(request.Patients)))
	
	// Check SLA compliance
	slaThreshold := time.Duration(s.config.Config.PhenotypeEvaluationSLA) * time.Millisecond
	if evaluationDuration > slaThreshold {
		s.metrics.RecordSLAViolation("/v1/phenotypes/evaluate", "evaluation_time")
	}
	
	s.metrics.RecordRequest("POST", "/v1/phenotypes/evaluate", "200")
	c.JSON(http.StatusOK, gin.H{
		"results":         results,
		"processing_time": evaluationDuration.String(),
		"batch_size":      len(request.Patients),
		"sla_compliant":   evaluationDuration <= slaThreshold,
	})
}

// Phenotype reasoning explanation endpoint
func (s *Server) explainPhenotypes(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("POST", "/v1/phenotypes/explain")
	defer timer.ObserveDuration()
	
	var request models.PhenotypeEvaluationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.handleValidationError(c, err, "Invalid phenotype explanation request")
		return
	}
	
	// Force explanation to be included
	request.IncludeExplanation = true
	
	// Validate single patient for explanation
	if len(request.Patients) != 1 {
		s.handleValidationError(c, nil, "Explanation endpoint supports exactly one patient")
		return
	}
	
	// Perform phenotype evaluation with explanation
	startTime := time.Now()
	results, err := s.phenotypeEngine.EvaluatePhenotypes(c.Request.Context(), &request)
	evaluationDuration := time.Since(startTime)
	
	if err != nil {
		s.metrics.RecordPhenotypeEvaluation("explanation", "error", evaluationDuration)
		s.handleInternalError(c, err, "Phenotype explanation failed")
		return
	}
	
	if len(results) == 0 {
		s.handleNotFoundError(c, "No phenotype results found")
		return
	}
	
	// Record metrics
	s.metrics.RecordPhenotypeEvaluation("explanation", "success", evaluationDuration)
	
	// Check SLA compliance
	slaThreshold := time.Duration(s.config.Config.PhenotypeExplanationSLA) * time.Millisecond
	if evaluationDuration > slaThreshold {
		s.metrics.RecordSLAViolation("/v1/phenotypes/explain", "explanation_time")
	}
	
	explanation := results[0].Explanation
	if explanation == nil {
		s.handleInternalError(c, nil, "Explanation not generated")
		return
	}
	
	s.metrics.RecordRequest("POST", "/v1/phenotypes/explain", "200")
	c.JSON(http.StatusOK, gin.H{
		"patient_id":      results[0].PatientID,
		"explanation":     explanation,
		"processing_time": evaluationDuration.String(),
		"sla_compliant":   evaluationDuration <= slaThreshold,
	})
}

// Enhanced risk assessment endpoint
func (s *Server) assessRisk(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("POST", "/v1/risk/assess")
	defer timer.ObserveDuration()
	
	var request models.RiskAssessmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.handleValidationError(c, err, "Invalid risk assessment request")
		return
	}
	
	// Validate request
	if request.PatientID == "" {
		s.handleValidationError(c, nil, "Patient ID is required")
		return
	}
	
	// Perform risk assessment
	startTime := time.Now()
	result, err := s.riskService.AssessRisk(c.Request.Context(), &request)
	assessmentDuration := time.Since(startTime)
	
	if err != nil {
		s.metrics.RecordRiskAssessment("error", "error", assessmentDuration)
		s.handleInternalError(c, err, "Risk assessment failed")
		return
	}
	
	// Record metrics
	s.metrics.RecordRiskAssessment("comprehensive", "success", assessmentDuration)
	s.metrics.RecordRiskScore(result.OverallRisk.Level, result.OverallRisk.Score)
	
	// Check SLA compliance
	slaThreshold := time.Duration(s.config.Config.RiskAssessmentSLA) * time.Millisecond
	if assessmentDuration > slaThreshold {
		s.metrics.RecordSLAViolation("/v1/risk/assess", "assessment_time")
	}
	
	s.metrics.RecordRequest("POST", "/v1/risk/assess", "200")
	c.JSON(http.StatusOK, gin.H{
		"result":          result,
		"processing_time": assessmentDuration.String(),
		"sla_compliant":   assessmentDuration <= slaThreshold,
	})
}

// Treatment preferences endpoint
func (s *Server) evaluateTreatmentPreferences(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("POST", "/v1/treatment/preferences")
	defer timer.ObserveDuration()
	
	var request models.TreatmentPreferencesRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.handleValidationError(c, err, "Invalid treatment preferences request")
		return
	}
	
	// Validate request
	if request.PatientID == "" {
		s.handleValidationError(c, nil, "Patient ID is required")
		return
	}
	
	if request.Condition == "" {
		s.handleValidationError(c, nil, "Condition is required")
		return
	}
	
	// Perform treatment preference evaluation
	startTime := time.Now()
	result, err := s.treatmentService.EvaluateTreatmentPreferences(c.Request.Context(), &request)
	evaluationDuration := time.Since(startTime)
	
	if err != nil {
		s.metrics.RecordTreatmentPreference(request.Condition, "error", evaluationDuration)
		s.handleInternalError(c, err, "Treatment preference evaluation failed")
		return
	}
	
	// Record metrics
	s.metrics.RecordTreatmentPreference(request.Condition, "success", evaluationDuration)
	s.metrics.RecordTreatmentOptions("generated", len(result.TreatmentOptions))
	
	// Check SLA compliance
	slaThreshold := time.Duration(s.config.Config.TreatmentPreferencesSLA) * time.Millisecond
	if evaluationDuration > slaThreshold {
		s.metrics.RecordSLAViolation("/v1/treatment/preferences", "evaluation_time")
	}
	
	s.metrics.RecordRequest("POST", "/v1/treatment/preferences", "200")
	c.JSON(http.StatusOK, gin.H{
		"result":          result,
		"processing_time": evaluationDuration.String(),
		"sla_compliant":   evaluationDuration <= slaThreshold,
	})
}

// Complete context assembly endpoint
func (s *Server) assembleContext(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("POST", "/v1/context/assemble")
	defer timer.ObserveDuration()
	
	var request models.ContextAssemblyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		s.handleValidationError(c, err, "Invalid context assembly request")
		return
	}
	
	// Validate request
	if request.PatientID == "" {
		s.handleValidationError(c, nil, "Patient ID is required")
		return
	}
	
	// Set default detail level
	if request.DetailLevel == "" {
		request.DetailLevel = "standard"
	}
	
	// Perform context assembly
	startTime := time.Now()
	result, err := s.contextService.AssembleContext(c.Request.Context(), &request)
	assemblyDuration := time.Since(startTime)
	
	if err != nil {
		s.metrics.RecordContextAssembly(request.DetailLevel, "error", assemblyDuration)
		s.handleInternalError(c, err, "Context assembly failed")
		return
	}
	
	// Record metrics
	s.metrics.RecordContextAssembly(request.DetailLevel, "success", assemblyDuration)
	
	// Check SLA compliance
	slaThreshold := time.Duration(s.config.Config.ContextAssemblySLA) * time.Millisecond
	if assemblyDuration > slaThreshold {
		s.metrics.RecordSLAViolation("/v1/context/assemble", "assembly_time")
	}
	
	// Cache the result for future use
	if s.config.Config.EnableCaching {
		go func() {
			if cacheErr := s.contextService.CacheContext(c.Request.Context(), result); cacheErr != nil {
				s.metrics.RecordError("cache_error", "context_service")
			}
		}()
	}
	
	s.metrics.RecordRequest("POST", "/v1/context/assemble", "200")
	c.JSON(http.StatusOK, gin.H{
		"context":         result,
		"processing_time": assemblyDuration.String(),
		"sla_compliant":   assemblyDuration <= slaThreshold,
	})
}

// Get available phenotypes endpoint
func (s *Server) getAvailablePhenotypes(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("GET", "/v1/phenotypes")
	defer timer.ObserveDuration()
	
	phenotypes := s.phenotypeEngine.GetAvailablePhenotypes()
	
	s.metrics.RecordRequest("GET", "/v1/phenotypes", "200")
	c.JSON(http.StatusOK, gin.H{
		"phenotypes": phenotypes,
		"count":      len(phenotypes),
	})
}

// Get patient context history endpoint
func (s *Server) getContextHistory(c *gin.Context) {
	s.metrics.IncrementConcurrentRequests()
	defer s.metrics.DecrementConcurrentRequests()
	
	timer := s.metrics.StartTimer("GET", "/v1/context/history/:patient_id")
	defer timer.ObserveDuration()
	
	patientID := c.Param("patient_id")
	if patientID == "" {
		s.handleValidationError(c, nil, "Patient ID is required")
		return
	}
	
	// Parse query parameters
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		s.handleValidationError(c, nil, "Invalid limit parameter (1-100)")
		return
	}
	
	// Get context history
	history, err := s.contextService.GetContextHistory(c.Request.Context(), patientID, limit)
	if err != nil {
		s.handleInternalError(c, err, "Failed to retrieve context history")
		return
	}
	
	s.metrics.RecordRequest("GET", "/v1/context/history/:patient_id", "200")
	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"history":    history,
		"count":      len(history),
	})
}

// API documentation endpoint
func (s *Server) apiDocumentation(c *gin.Context) {
	docs := gin.H{
		"service": "KB-2 Clinical Context Service",
		"version": "1.0.0",
		"endpoints": gin.H{
			"health": gin.H{
				"path":        "/health",
				"method":      "GET",
				"description": "Service health check",
			},
			"evaluate_phenotypes": gin.H{
				"path":        "/v1/phenotypes/evaluate",
				"method":      "POST",
				"description": "Batch phenotype evaluation for up to 1000 patients",
				"sla":         "100ms p95",
			},
			"explain_phenotypes": gin.H{
				"path":        "/v1/phenotypes/explain",
				"method":      "POST",
				"description": "Detailed phenotype reasoning explanation",
				"sla":         "150ms p95",
			},
			"assess_risk": gin.H{
				"path":        "/v1/risk/assess",
				"method":      "POST",
				"description": "Enhanced risk calculation with multiple categories",
				"sla":         "200ms p95",
			},
			"treatment_preferences": gin.H{
				"path":        "/v1/treatment/preferences",
				"method":      "POST",
				"description": "Treatment recommendations with institutional rules",
				"sla":         "50ms p95",
			},
			"assemble_context": gin.H{
				"path":        "/v1/context/assemble",
				"method":      "POST",
				"description": "Complete clinical context assembly",
				"sla":         "200ms p95",
			},
			"get_phenotypes": gin.H{
				"path":        "/v1/phenotypes",
				"method":      "GET",
				"description": "List available phenotype definitions",
			},
			"context_history": gin.H{
				"path":        "/v1/context/history/:patient_id",
				"method":      "GET",
				"description": "Retrieve patient context history",
			},
		},
		"features": []string{
			"CEL-based phenotype evaluation",
			"Batch processing up to 1000 patients",
			"Multi-category risk assessment",
			"Priority-based conflict resolution",
			"Institutional rule compliance",
			"Performance SLA monitoring",
			"Comprehensive caching",
			"RFC 7807 error format",
		},
	}
	
	c.JSON(http.StatusOK, docs)
}

// Error handling helpers

func (s *Server) handleValidationError(c *gin.Context, err error, message string) {
	apiError := &models.APIError{
		Type:      "validation_error",
		Title:     "Validation Error",
		Status:    http.StatusBadRequest,
		Detail:    message,
		Instance:  c.Request.URL.Path,
		Timestamp: time.Now(),
	}
	
	if err != nil {
		apiError.Metadata = map[string]interface{}{
			"validation_error": err.Error(),
		}
	}
	
	s.metrics.RecordValidationError("request_validation")
	s.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, "400")
	c.JSON(http.StatusBadRequest, apiError)
}

func (s *Server) handleInternalError(c *gin.Context, err error, message string) {
	apiError := &models.APIError{
		Type:      "internal_error",
		Title:     "Internal Server Error",
		Status:    http.StatusInternalServerError,
		Detail:    message,
		Instance:  c.Request.URL.Path,
		Timestamp: time.Now(),
	}
	
	if err != nil {
		apiError.Metadata = map[string]interface{}{
			"error": err.Error(),
		}
	}
	
	s.metrics.RecordError("internal_error", "server")
	s.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, "500")
	c.JSON(http.StatusInternalServerError, apiError)
}

func (s *Server) handleNotFoundError(c *gin.Context, message string) {
	apiError := &models.APIError{
		Type:      "not_found",
		Title:     "Not Found",
		Status:    http.StatusNotFound,
		Detail:    message,
		Instance:  c.Request.URL.Path,
		Timestamp: time.Now(),
	}
	
	s.metrics.RecordRequest(c.Request.Method, c.Request.URL.Path, "404")
	c.JSON(http.StatusNotFound, apiError)
}

// Utility methods for metrics

func (s *Server) RecordRiskScore(level string, score float64) {
	s.metrics.RiskScoresGenerated.WithLabelValues(level).Inc()
}

func (s *Server) RecordTreatmentOptions(category string, count int) {
	for i := 0; i < count; i++ {
		s.metrics.TreatmentOptionsGenerated.WithLabelValues(category).Inc()
	}
}