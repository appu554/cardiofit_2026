package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/infrastructure"
)

// ApolloFederationHandler handles HTTP requests for Apollo Federation knowledge base queries
type ApolloFederationHandler struct {
	knowledgeBaseService *services.KnowledgeBaseIntegrationService
	logger               *zap.Logger
}

// NewApolloFederationHandler creates a new Apollo Federation HTTP handler
func NewApolloFederationHandler(
	knowledgeBaseService *services.KnowledgeBaseIntegrationService,
	logger *zap.Logger,
) *ApolloFederationHandler {
	return &ApolloFederationHandler{
		knowledgeBaseService: knowledgeBaseService,
		logger:              logger,
	}
}

// KnowledgeQueryHTTPRequest represents HTTP request for knowledge queries
type KnowledgeQueryHTTPRequest struct {
	DrugCode         string                                 `json:"drug_code" binding:"required"`
	DrugCodes        []string                               `json:"drug_codes,omitempty"`
	PatientContext   *infrastructure.PatientContextInput   `json:"patient_context,omitempty"`
	Version          *string                                `json:"version,omitempty"`
	Region           *string                                `json:"region,omitempty"`
	QueryTypes       []string                               `json:"query_types" binding:"required"`
	Filters          map[string]interface{}                 `json:"filters,omitempty"`
	Limit            *int32                                 `json:"limit,omitempty"`
	Fields           []string                               `json:"fields,omitempty"`
	CacheEnabled     *bool                                  `json:"cache_enabled,omitempty"`
	CacheTTLMinutes  *int                                   `json:"cache_ttl_minutes,omitempty"`
	MaxConcurrency   *int                                   `json:"max_concurrency,omitempty"`
	TimeoutSeconds   *int                                   `json:"timeout_seconds,omitempty"`
	Priority         *string                                `json:"priority,omitempty"`
}

// BatchKnowledgeQueryHTTPRequest represents batch HTTP request
type BatchKnowledgeQueryHTTPRequest struct {
	Requests []KnowledgeQueryHTTPRequest `json:"requests" binding:"required"`
}

// RegisterRoutes registers Apollo Federation routes
func (h *ApolloFederationHandler) RegisterRoutes(router *gin.RouterGroup) {
	federationRoutes := router.Group("/federation")
	{
		// Single query endpoints
		federationRoutes.POST("/query", h.QueryKnowledge)
		federationRoutes.POST("/dosing", h.QueryDosing)
		federationRoutes.POST("/guidelines", h.QueryGuidelines)
		federationRoutes.POST("/interactions", h.QueryInteractions)
		federationRoutes.POST("/safety", h.QuerySafety)
		federationRoutes.POST("/availability", h.QueryAvailability)
		
		// Batch query endpoints
		federationRoutes.POST("/batch", h.BatchQueryKnowledge)
		federationRoutes.POST("/batch/dosing", h.BatchQueryDosing)
		
		// Clinical intelligence endpoint
		federationRoutes.POST("/clinical-intelligence", h.QueryClinicalIntelligence)
		
		// Health and metrics endpoints
		federationRoutes.GET("/health", h.HealthCheck)
		federationRoutes.GET("/metrics", h.GetMetrics)
		
		// Advanced query endpoints
		federationRoutes.POST("/comprehensive", h.ComprehensiveQuery)
		federationRoutes.POST("/personalized", h.PersonalizedQuery)
	}
}

// QueryKnowledge handles unified knowledge base queries
func (h *ApolloFederationHandler) QueryKnowledge(c *gin.Context) {
	var req KnowledgeQueryHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid knowledge query request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Convert HTTP request to service request
	serviceReq, err := h.convertToServiceRequest(&req)
	if err != nil {
		h.logger.Error("Failed to convert request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"message": err.Error(),
		})
		return
	}

	// Execute query
	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Knowledge query failed",
			zap.String("drug_code", req.DrugCode),
			zap.Strings("query_types", req.QueryTypes),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Query failed",
			"message": err.Error(),
		})
		return
	}

	// Add HTTP-specific metadata
	httpResponse := h.convertToHTTPResponse(response)

	c.JSON(http.StatusOK, httpResponse)
}

// QueryDosing handles dosing-specific queries
func (h *ApolloFederationHandler) QueryDosing(c *gin.Context) {
	var req KnowledgeQueryHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force query type to dosing
	req.QueryTypes = []string{"dosing"}

	serviceReq, err := h.convertToServiceRequest(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Dosing query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.convertToHTTPResponse(response))
}

// QueryGuidelines handles clinical guidelines queries
func (h *ApolloFederationHandler) QueryGuidelines(c *gin.Context) {
	var req KnowledgeQueryHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force query type to guidelines
	req.QueryTypes = []string{"guidelines"}

	serviceReq, err := h.convertToServiceRequest(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Guidelines query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.convertToHTTPResponse(response))
}

// QueryInteractions handles drug interactions queries
func (h *ApolloFederationHandler) QueryInteractions(c *gin.Context) {
	var req KnowledgeQueryHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force query type to interactions
	req.QueryTypes = []string{"interactions"}

	serviceReq, err := h.convertToServiceRequest(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Interactions query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.convertToHTTPResponse(response))
}

// QuerySafety handles patient safety queries
func (h *ApolloFederationHandler) QuerySafety(c *gin.Context) {
	var req KnowledgeQueryHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Force query type to safety
	req.QueryTypes = []string{"safety"}

	serviceReq, err := h.convertToServiceRequest(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Safety query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.convertToHTTPResponse(response))
}

// QueryAvailability handles availability check queries
func (h *ApolloFederationHandler) QueryAvailability(c *gin.Context) {
	drugCode := c.Query("drug_code")
	if drugCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "drug_code parameter required"})
		return
	}

	region := c.Query("region")

	req := &services.KnowledgeBaseQueryRequest{
		DrugCode:     drugCode,
		Region:       &region,
		QueryTypes:   []string{"availability"},
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("Availability query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return simplified response for availability
	available := false
	if response.AvailabilityStatus != nil {
		if status, exists := response.AvailabilityStatus[drugCode]; exists {
			available = status
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"drug_code": drugCode,
		"available": available,
		"region":    region,
		"response_time_ms": response.QueryMetrics.TotalDuration.Milliseconds(),
	})
}

// BatchQueryKnowledge handles batch knowledge queries
func (h *ApolloFederationHandler) BatchQueryKnowledge(c *gin.Context) {
	var req BatchKnowledgeQueryHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Requests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No requests provided"})
		return
	}

	if len(req.Requests) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Batch size too large (max 50)"})
		return
	}

	// Convert all requests
	serviceRequests := make([]*services.KnowledgeBaseQueryRequest, len(req.Requests))
	for i, httpReq := range req.Requests {
		serviceReq, err := h.convertToServiceRequest(&httpReq)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid request at index %d: %s", i, err.Error()),
			})
			return
		}
		serviceRequests[i] = serviceReq
	}

	// Execute batch query
	responses, err := h.knowledgeBaseService.BatchQueryKnowledgeBases(c.Request.Context(), serviceRequests)
	if err != nil {
		h.logger.Error("Batch knowledge query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert responses
	httpResponses := make(map[string]interface{})
	for drugCode, response := range responses {
		httpResponses[drugCode] = h.convertToHTTPResponse(response)
	}

	c.JSON(http.StatusOK, gin.H{
		"results": httpResponses,
		"batch_size": len(req.Requests),
		"successful_queries": len(httpResponses),
	})
}

// BatchQueryDosing handles batch dosing queries with optimization
func (h *ApolloFederationHandler) BatchQueryDosing(c *gin.Context) {
	drugCodes := c.QueryArray("drug_codes")
	if len(drugCodes) == 0 {
		// Try to get from body
		var bodyReq struct {
			DrugCodes []string `json:"drug_codes"`
			Region    *string  `json:"region"`
		}
		if err := c.ShouldBindJSON(&bodyReq); err == nil && len(bodyReq.DrugCodes) > 0 {
			drugCodes = bodyReq.DrugCodes
		}
	}

	if len(drugCodes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "drug_codes required"})
		return
	}

	region := c.Query("region")
	
	// Create batch requests
	requests := make([]*services.KnowledgeBaseQueryRequest, len(drugCodes))
	for i, drugCode := range drugCodes {
		requests[i] = &services.KnowledgeBaseQueryRequest{
			DrugCode:     drugCode,
			Region:       &region,
			QueryTypes:   []string{"dosing"},
			CacheEnabled: true,
			CacheTTL:     30 * time.Minute,
		}
	}

	responses, err := h.knowledgeBaseService.BatchQueryKnowledgeBases(c.Request.Context(), requests)
	if err != nil {
		h.logger.Error("Batch dosing query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert to simplified dosing response format
	dosingResults := make(map[string]interface{})
	for drugCode, response := range responses {
		result := gin.H{
			"drug_code": drugCode,
			"available": len(response.DosingRules) > 0,
		}
		
		if len(response.DosingRules) > 0 {
			rule := response.DosingRules[0]
			result["dosing_rule"] = gin.H{
				"version":     rule.Version,
				"drug_name":   rule.DrugName,
				"base_dose":   rule.BaseDose,
				"adjustments": len(rule.Adjustments),
				"active":      rule.Active,
			}
		}
		
		dosingResults[drugCode] = result
	}

	c.JSON(http.StatusOK, gin.H{
		"results":           dosingResults,
		"requested_count":   len(drugCodes),
		"successful_count":  len(responses),
	})
}

// QueryClinicalIntelligence handles comprehensive clinical intelligence queries
func (h *ApolloFederationHandler) QueryClinicalIntelligence(c *gin.Context) {
	var req struct {
		DrugCode       string                                 `json:"drug_code" binding:"required"`
		PatientContext *infrastructure.PatientContextInput   `json:"patient_context,omitempty"`
		Region         *string                                `json:"region,omitempty"`
		IncludeAll     bool                                   `json:"include_all,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine query types based on request
	queryTypes := []string{"dosing", "availability"}
	if req.IncludeAll {
		queryTypes = append(queryTypes, "guidelines", "interactions", "safety")
	}

	serviceReq := &services.KnowledgeBaseQueryRequest{
		DrugCode:       req.DrugCode,
		PatientContext: req.PatientContext,
		Region:         req.Region,
		QueryTypes:     queryTypes,
		CacheEnabled:   true,
		CacheTTL:       15 * time.Minute,
		Priority:       "high",
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Clinical intelligence query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.convertToHTTPResponse(response))
}

// PersonalizedQuery handles personalized clinical queries with patient context
func (h *ApolloFederationHandler) PersonalizedQuery(c *gin.Context) {
	var req struct {
		DrugCode       string                                `json:"drug_code" binding:"required"`
		PatientContext infrastructure.PatientContextInput  `json:"patient_context" binding:"required"`
		Region         *string                               `json:"region,omitempty"`
		QueryTypes     []string                              `json:"query_types,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to dosing queries for personalized requests
	if len(req.QueryTypes) == 0 {
		req.QueryTypes = []string{"dosing"}
	}

	serviceReq := &services.KnowledgeBaseQueryRequest{
		DrugCode:       req.DrugCode,
		PatientContext: &req.PatientContext,
		Region:         req.Region,
		QueryTypes:     req.QueryTypes,
		CacheEnabled:   true,
		CacheTTL:       10 * time.Minute, // Shorter TTL for personalized queries
		Priority:       "high",
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Personalized query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.convertToHTTPResponse(response))
}

// ComprehensiveQuery handles comprehensive queries with all available knowledge
func (h *ApolloFederationHandler) ComprehensiveQuery(c *gin.Context) {
	drugCode := c.Query("drug_code")
	if drugCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "drug_code parameter required"})
		return
	}

	region := c.Query("region")

	serviceReq := &services.KnowledgeBaseQueryRequest{
		DrugCode:     drugCode,
		Region:       &region,
		QueryTypes:   []string{"dosing", "guidelines", "interactions", "safety", "availability"},
		CacheEnabled: true,
		CacheTTL:     20 * time.Minute,
		Priority:     "normal",
	}

	response, err := h.knowledgeBaseService.QueryKnowledgeBases(c.Request.Context(), serviceReq)
	if err != nil {
		h.logger.Error("Comprehensive query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.convertToHTTPResponse(response))
}

// HealthCheck handles health check requests
func (h *ApolloFederationHandler) HealthCheck(c *gin.Context) {
	health, err := h.knowledgeBaseService.GetServiceHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"health": health,
	})
}

// GetMetrics handles metrics requests
func (h *ApolloFederationHandler) GetMetrics(c *gin.Context) {
	health, err := h.knowledgeBaseService.GetServiceHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, health)
}

// Helper methods

// convertToServiceRequest converts HTTP request to service request
func (h *ApolloFederationHandler) convertToServiceRequest(httpReq *KnowledgeQueryHTTPRequest) (*services.KnowledgeBaseQueryRequest, error) {
	req := &services.KnowledgeBaseQueryRequest{
		DrugCode:       httpReq.DrugCode,
		DrugCodes:      httpReq.DrugCodes,
		PatientContext: httpReq.PatientContext,
		Version:        httpReq.Version,
		Region:         httpReq.Region,
		QueryTypes:     httpReq.QueryTypes,
		Filters:        httpReq.Filters,
		Fields:         httpReq.Fields,
	}

	// Set cache settings
	if httpReq.CacheEnabled != nil {
		req.CacheEnabled = *httpReq.CacheEnabled
	} else {
		req.CacheEnabled = true // Default to enabled
	}

	// Set cache TTL
	if httpReq.CacheTTLMinutes != nil {
		req.CacheTTL = time.Duration(*httpReq.CacheTTLMinutes) * time.Minute
	}

	// Set concurrency
	if httpReq.MaxConcurrency != nil {
		req.MaxConcurrency = *httpReq.MaxConcurrency
	}

	// Set timeout
	if httpReq.TimeoutSeconds != nil {
		timeout := time.Duration(*httpReq.TimeoutSeconds) * time.Second
		req.TimeoutOverride = &timeout
	}

	// Set priority
	if httpReq.Priority != nil {
		req.Priority = *httpReq.Priority
	}

	return req, nil
}

// convertToHTTPResponse converts service response to HTTP response
func (h *ApolloFederationHandler) convertToHTTPResponse(serviceResponse *services.KnowledgeBaseQueryResponse) map[string]interface{} {
	return map[string]interface{}{
		"request_id":             serviceResponse.RequestID,
		"drug_code":              serviceResponse.DrugCode,
		"drug_codes":             serviceResponse.DrugCodes,
		"query_types":            serviceResponse.QueryTypes,
		"dosing_rules":           serviceResponse.DosingRules,
		"dosing_recommendations": serviceResponse.DosingRecommendations,
		"clinical_guidelines":    serviceResponse.ClinicalGuidelines,
		"drug_interactions":      serviceResponse.DrugInteractions,
		"safety_alerts":          serviceResponse.SafetyAlerts,
		"availability_status":    serviceResponse.AvailabilityStatus,
		"knowledge_base_status":  serviceResponse.KnowledgeBaseStatus,
		"query_metrics":          map[string]interface{}{
			"total_duration_ms":         serviceResponse.QueryMetrics.TotalDuration.Milliseconds(),
			"knowledge_base_latency_ms": convertDurationMapToMs(serviceResponse.QueryMetrics.KnowledgeBaseLatency),
			"network_latency_ms":        serviceResponse.QueryMetrics.NetworkLatency.Milliseconds(),
			"processing_time_ms":        serviceResponse.QueryMetrics.ProcessingTime.Milliseconds(),
			"retry_count":               serviceResponse.QueryMetrics.RetryCount,
			"error_count":               serviceResponse.QueryMetrics.ErrorCount,
		},
		"cache_status": map[string]interface{}{
			"enabled":         serviceResponse.CacheStatus.Enabled,
			"hit_rate":        serviceResponse.CacheStatus.HitRate,
			"hits_by_type":    serviceResponse.CacheStatus.HitsByType,
			"misses_by_type":  serviceResponse.CacheStatus.MissesByType,
			"ttl_used_minutes": serviceResponse.CacheStatus.TTLUsed.Minutes(),
		},
		"execution_summary": serviceResponse.ExecutionSummary,
	}
}

// convertDurationMapToMs converts duration map to milliseconds
func convertDurationMapToMs(durationMap map[string]time.Duration) map[string]int64 {
	msMap := make(map[string]int64)
	for k, v := range durationMap {
		msMap[k] = v.Milliseconds()
	}
	return msMap
}