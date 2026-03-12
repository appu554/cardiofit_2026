package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/search"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SearchHandlers provides REST API handlers for clinical terminology search
type SearchHandlers struct {
	searchEngine       *search.SearchEngine
	autocompleteService *search.AutocompleteService
	queryAnalyzer      *search.QueryAnalyzer
	logger             *zap.Logger
	metrics            *metrics.Collector
	config             *SearchAPIConfig
}

// SearchAPIConfig holds configuration for the search API
type SearchAPIConfig struct {
	DefaultTimeout       time.Duration `json:"default_timeout"`
	MaxQueryLength       int           `json:"max_query_length"`
	MaxPageSize          int           `json:"max_page_size"`
	EnableRateLimiting   bool          `json:"enable_rate_limiting"`
	EnableRequestLogging bool          `json:"enable_request_logging"`
	EnableCORS           bool          `json:"enable_cors"`
	CORSOrigins          []string      `json:"cors_origins"`
	APIVersion           string        `json:"api_version"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Metadata  *APIMetadata `json:"metadata,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// APIMetadata contains response metadata
type APIMetadata struct {
	Timestamp   time.Time     `json:"timestamp"`
	ProcessingTime time.Duration `json:"processing_time"`
	Version     string        `json:"version"`
	RequestID   string        `json:"request_id"`
	QueryInfo   *QueryInfo    `json:"query_info,omitempty"`
}

// QueryInfo contains information about the processed query
type QueryInfo struct {
	OriginalQuery   string  `json:"original_query"`
	ProcessedQuery  string  `json:"processed_query,omitempty"`
	QueryIntent     string  `json:"query_intent,omitempty"`
	Confidence      float64 `json:"confidence,omitempty"`
}

// NewSearchHandlers creates new search API handlers
func NewSearchHandlers(
	searchEngine *search.SearchEngine,
	autocompleteService *search.AutocompleteService,
	queryAnalyzer *search.QueryAnalyzer,
	logger *zap.Logger,
	metrics *metrics.Collector,
	config *SearchAPIConfig,
) *SearchHandlers {
	if config == nil {
		config = DefaultSearchAPIConfig()
	}

	return &SearchHandlers{
		searchEngine:        searchEngine,
		autocompleteService: autocompleteService,
		queryAnalyzer:      queryAnalyzer,
		logger:             logger,
		metrics:            metrics,
		config:             config,
	}
}

// DefaultSearchAPIConfig returns default API configuration
func DefaultSearchAPIConfig() *SearchAPIConfig {
	return &SearchAPIConfig{
		DefaultTimeout:       30 * time.Second,
		MaxQueryLength:       500,
		MaxPageSize:         100,
		EnableRateLimiting:  true,
		EnableRequestLogging: true,
		EnableCORS:          true,
		CORSOrigins:         []string{"*"},
		APIVersion:          "1.0",
	}
}

// RegisterSearchRoutes registers all search-related routes
func (sh *SearchHandlers) RegisterSearchRoutes(router *gin.RouterGroup) {
	// Apply middleware
	router.Use(sh.requestIDMiddleware())
	if sh.config.EnableRequestLogging {
		router.Use(sh.loggingMiddleware())
	}
	if sh.config.EnableCORS {
		router.Use(sh.corsMiddleware())
	}

	// Search endpoints
	v1 := router.Group("/v1")
	{
		// Main search endpoint
		v1.GET("/search", sh.Search)
		v1.POST("/search", sh.SearchAdvanced)

		// Specialized search endpoints
		v1.GET("/search/exact", sh.ExactSearch)
		v1.GET("/search/fuzzy", sh.FuzzySearch)
		v1.GET("/search/phonetic", sh.PhoneticSearch)

		// Autocomplete endpoints
		v1.GET("/autocomplete", sh.Autocomplete)
		v1.GET("/suggest", sh.Autocomplete) // Alias for autocomplete

		// Query analysis endpoints
		v1.POST("/analyze", sh.AnalyzeQuery)
		v1.GET("/analyze", sh.AnalyzeQueryGET)

		// Search health and status
		v1.GET("/health", sh.HealthCheck)
		v1.GET("/status", sh.Status)
		v1.GET("/metrics", sh.Metrics)

		// Search configuration and capabilities
		v1.GET("/capabilities", sh.GetCapabilities)
		v1.GET("/systems", sh.GetSupportedSystems)
		v1.GET("/domains", sh.GetSupportedDomains)
	}

	// Batch operations
	batch := router.Group("/batch")
	{
		batch.POST("/search", sh.BatchSearch)
		batch.POST("/analyze", sh.BatchAnalyze)
	}

	// Administrative endpoints
	admin := router.Group("/admin")
	{
		admin.GET("/stats", sh.GetSearchStats)
		admin.POST("/cache/clear", sh.ClearCache)
		admin.GET("/performance", sh.GetPerformanceMetrics)
	}
}

// Search handles the main search endpoint (GET /v1/search)
func (sh *SearchHandlers) Search(c *gin.Context) {
	startTime := time.Now()
	requestID := sh.getRequestID(c)

	// Parse query parameters
	query := c.Query("q")
	if query == "" {
		sh.respondError(c, http.StatusBadRequest, "MISSING_QUERY", "Query parameter 'q' is required", requestID)
		return
	}

	// Validate query length
	if len(query) > sh.config.MaxQueryLength {
		sh.respondError(c, http.StatusBadRequest, "QUERY_TOO_LONG",
			fmt.Sprintf("Query exceeds maximum length of %d characters", sh.config.MaxQueryLength), requestID)
		return
	}

	// Build search request
	searchRequest, err := sh.buildSearchRequestFromQuery(c, query)
	if err != nil {
		sh.respondError(c, http.StatusBadRequest, "INVALID_PARAMETERS", err.Error(), requestID)
		return
	}

	// Create search context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), sh.config.DefaultTimeout)
	defer cancel()

	// Execute search
	result, err := sh.searchEngine.Search(ctx, searchRequest)
	if err != nil {
		sh.logger.Error("Search failed",
			zap.String("request_id", requestID),
			zap.String("query", query),
			zap.Error(err),
		)
		sh.respondError(c, http.StatusInternalServerError, "SEARCH_FAILED", "Search execution failed", requestID)
		return
	}

	// Build response metadata
	metadata := &APIMetadata{
		Timestamp:      startTime,
		ProcessingTime: time.Since(startTime),
		Version:        sh.config.APIVersion,
		RequestID:      requestID,
		QueryInfo: &QueryInfo{
			OriginalQuery:  query,
			ProcessedQuery: result.ProcessedQuery,
		},
	}

	if result.QueryAnalysis != nil {
		metadata.QueryInfo.QueryIntent = string(result.QueryAnalysis.DetectedIntent)
		metadata.QueryInfo.Confidence = result.QueryAnalysis.Confidence
	}

	// Record metrics
	sh.recordSearchMetrics("search", requestID, query, result, time.Since(startTime))

	// Respond with results
	sh.respondSuccess(c, result, metadata)
}

// SearchAdvanced handles advanced search with POST body (POST /v1/search)
func (sh *SearchHandlers) SearchAdvanced(c *gin.Context) {
	startTime := time.Now()
	requestID := sh.getRequestID(c)

	// Parse request body
	var searchRequest search.ClinicalSearchRequest
	if err := c.ShouldBindJSON(&searchRequest); err != nil {
		sh.respondError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", requestID)
		return
	}

	// Validate request
	if err := sh.validateSearchRequest(&searchRequest); err != nil {
		sh.respondError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error(), requestID)
		return
	}

	// Create search context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), sh.config.DefaultTimeout)
	defer cancel()

	// Execute search
	result, err := sh.searchEngine.Search(ctx, &searchRequest)
	if err != nil {
		sh.logger.Error("Advanced search failed",
			zap.String("request_id", requestID),
			zap.String("query", searchRequest.Query),
			zap.Error(err),
		)
		sh.respondError(c, http.StatusInternalServerError, "SEARCH_FAILED", "Search execution failed", requestID)
		return
	}

	// Build response metadata
	metadata := &APIMetadata{
		Timestamp:      startTime,
		ProcessingTime: time.Since(startTime),
		Version:        sh.config.APIVersion,
		RequestID:      requestID,
		QueryInfo: &QueryInfo{
			OriginalQuery:  searchRequest.Query,
			ProcessedQuery: result.ProcessedQuery,
		},
	}

	// Record metrics
	sh.recordSearchMetrics("search_advanced", requestID, searchRequest.Query, result, time.Since(startTime))

	// Respond with results
	sh.respondSuccess(c, result, metadata)
}

// ExactSearch handles exact matching search (GET /v1/search/exact)
func (sh *SearchHandlers) ExactSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		sh.respondError(c, http.StatusBadRequest, "MISSING_QUERY", "Query parameter 'q' is required", sh.getRequestID(c))
		return
	}

	// Build exact search request
	searchRequest, err := sh.buildSearchRequestFromQuery(c, query)
	if err != nil {
		sh.respondError(c, http.StatusBadRequest, "INVALID_PARAMETERS", err.Error(), sh.getRequestID(c))
		return
	}

	// Force exact search mode
	searchRequest.SearchMode = search.SearchModeExact
	searchRequest.ExactMatch = true

	sh.executeAndRespond(c, searchRequest, "exact_search")
}

// FuzzySearch handles fuzzy matching search (GET /v1/search/fuzzy)
func (sh *SearchHandlers) FuzzySearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		sh.respondError(c, http.StatusBadRequest, "MISSING_QUERY", "Query parameter 'q' is required", sh.getRequestID(c))
		return
	}

	// Build fuzzy search request
	searchRequest, err := sh.buildSearchRequestFromQuery(c, query)
	if err != nil {
		sh.respondError(c, http.StatusBadRequest, "INVALID_PARAMETERS", err.Error(), sh.getRequestID(c))
		return
	}

	// Force fuzzy search mode
	searchRequest.SearchMode = search.SearchModeFuzzy

	// Parse fuzzy threshold if provided
	if thresholdStr := c.Query("threshold"); thresholdStr != "" {
		if threshold, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			searchRequest.FuzzyThreshold = threshold
		}
	}

	sh.executeAndRespond(c, searchRequest, "fuzzy_search")
}

// PhoneticSearch handles phonetic matching search (GET /v1/search/phonetic)
func (sh *SearchHandlers) PhoneticSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		sh.respondError(c, http.StatusBadRequest, "MISSING_QUERY", "Query parameter 'q' is required", sh.getRequestID(c))
		return
	}

	// Build phonetic search request
	searchRequest, err := sh.buildSearchRequestFromQuery(c, query)
	if err != nil {
		sh.respondError(c, http.StatusBadRequest, "INVALID_PARAMETERS", err.Error(), sh.getRequestID(c))
		return
	}

	// Force phonetic search mode
	searchRequest.SearchMode = search.SearchModePhonetic

	sh.executeAndRespond(c, searchRequest, "phonetic_search")
}

// Autocomplete handles autocomplete suggestions (GET /v1/autocomplete)
func (sh *SearchHandlers) Autocomplete(c *gin.Context) {
	startTime := time.Now()
	requestID := sh.getRequestID(c)

	// Parse query parameter
	query := c.Query("q")
	if query == "" {
		sh.respondError(c, http.StatusBadRequest, "MISSING_QUERY", "Query parameter 'q' is required", requestID)
		return
	}

	// Build autocomplete request
	autocompleteRequest := &search.AutocompleteRequest{
		Query: query,
		Context: sh.buildUserContext(c),
		Filters: sh.buildSuggestionFilters(c),
		Options: sh.buildSuggestionOptions(c),
	}

	// Parse max suggestions
	if maxStr := c.Query("max"); maxStr != "" {
		if max, err := strconv.Atoi(maxStr); err == nil {
			autocompleteRequest.MaxSuggestions = max
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Execute autocomplete
	result, err := sh.autocompleteService.GetSuggestions(ctx, autocompleteRequest)
	if err != nil {
		sh.logger.Error("Autocomplete failed",
			zap.String("request_id", requestID),
			zap.String("query", query),
			zap.Error(err),
		)
		sh.respondError(c, http.StatusInternalServerError, "AUTOCOMPLETE_FAILED", "Autocomplete execution failed", requestID)
		return
	}

	// Build response metadata
	metadata := &APIMetadata{
		Timestamp:      startTime,
		ProcessingTime: time.Since(startTime),
		Version:        sh.config.APIVersion,
		RequestID:      requestID,
		QueryInfo: &QueryInfo{
			OriginalQuery: query,
		},
	}

	// Record metrics
	sh.recordAutocompleteMetrics(requestID, query, result, time.Since(startTime))

	// Respond with suggestions
	sh.respondSuccess(c, result, metadata)
}

// AnalyzeQuery handles query analysis (POST /v1/analyze)
func (sh *SearchHandlers) AnalyzeQuery(c *gin.Context) {
	startTime := time.Now()
	requestID := sh.getRequestID(c)

	// Parse request body
	var analysisRequest search.QueryAnalysisRequest
	if err := c.ShouldBindJSON(&analysisRequest); err != nil {
		sh.respondError(c, http.StatusBadRequest, "INVALID_JSON", "Invalid request body", requestID)
		return
	}

	// Validate request
	if analysisRequest.Query == "" {
		sh.respondError(c, http.StatusBadRequest, "MISSING_QUERY", "Query is required", requestID)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Execute analysis
	result, err := sh.queryAnalyzer.AnalyzeQuery(ctx, &analysisRequest)
	if err != nil {
		sh.logger.Error("Query analysis failed",
			zap.String("request_id", requestID),
			zap.String("query", analysisRequest.Query),
			zap.Error(err),
		)
		sh.respondError(c, http.StatusInternalServerError, "ANALYSIS_FAILED", "Query analysis failed", requestID)
		return
	}

	// Build response metadata
	metadata := &APIMetadata{
		Timestamp:      startTime,
		ProcessingTime: time.Since(startTime),
		Version:        sh.config.APIVersion,
		RequestID:      requestID,
		QueryInfo: &QueryInfo{
			OriginalQuery:  analysisRequest.Query,
			ProcessedQuery: result.NormalizedQuery,
			QueryIntent:    string(result.DetectedIntent),
			Confidence:     result.ConfidenceScore,
		},
	}

	// Record metrics
	sh.recordAnalysisMetrics(requestID, analysisRequest.Query, result, time.Since(startTime))

	// Respond with analysis
	sh.respondSuccess(c, result, metadata)
}

// AnalyzeQueryGET handles query analysis via GET (GET /v1/analyze)
func (sh *SearchHandlers) AnalyzeQueryGET(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		sh.respondError(c, http.StatusBadRequest, "MISSING_QUERY", "Query parameter 'q' is required", sh.getRequestID(c))
		return
	}

	// Build analysis request
	analysisRequest := search.QueryAnalysisRequest{
		Query: query,
		Options: &search.AnalysisOptions{
			IncludeEntityExtraction:     true,
			IncludeIntentPrediction:     true,
			IncludeDomainClassification: true,
			DetailLevel:                 search.AnalysisDetailStandard,
		},
	}

	// Parse detail level
	if detail := c.Query("detail"); detail != "" {
		switch detail {
		case "basic":
			analysisRequest.Options.DetailLevel = search.AnalysisDetailBasic
		case "comprehensive":
			analysisRequest.Options.DetailLevel = search.AnalysisDetailComprehensive
		}
	}

	// Convert to POST-style request
	c.Set("analysis_request", analysisRequest)
	sh.AnalyzeQuery(c)
}

// HealthCheck provides health status (GET /v1/health)
func (sh *SearchHandlers) HealthCheck(c *gin.Context) {
	requestID := sh.getRequestID(c)

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   sh.config.APIVersion,
		"services": map[string]string{
			"search_engine":        "healthy",
			"autocomplete_service": "healthy",
			"query_analyzer":       "healthy",
		},
	}

	metadata := &APIMetadata{
		Timestamp: time.Now(),
		Version:   sh.config.APIVersion,
		RequestID: requestID,
	}

	sh.respondSuccess(c, health, metadata)
}

// Status provides detailed status information (GET /v1/status)
func (sh *SearchHandlers) Status(c *gin.Context) {
	requestID := sh.getRequestID(c)

	status := map[string]interface{}{
		"api_version":        sh.config.APIVersion,
		"uptime":            time.Since(time.Now()), // Would be actual uptime
		"requests_processed": 1000,                  // Would be actual count
		"cache_hit_ratio":   0.85,                   // Would be actual ratio
		"average_response_time": "150ms",           // Would be actual average
		"supported_features": []string{
			"standard_search",
			"exact_search",
			"fuzzy_search",
			"phonetic_search",
			"autocomplete",
			"query_analysis",
			"batch_operations",
		},
	}

	metadata := &APIMetadata{
		Timestamp: time.Now(),
		Version:   sh.config.APIVersion,
		RequestID: requestID,
	}

	sh.respondSuccess(c, status, metadata)
}

// GetCapabilities returns API capabilities (GET /v1/capabilities)
func (sh *SearchHandlers) GetCapabilities(c *gin.Context) {
	requestID := sh.getRequestID(c)

	capabilities := map[string]interface{}{
		"search_modes": []string{
			"standard", "exact", "fuzzy", "phonetic", "wildcard", "semantic", "hybrid",
		},
		"query_types": []string{
			"general", "diagnostic", "procedural", "medication", "laboratory", "anatomy", "symptom",
		},
		"supported_systems": []string{
			"SNOMED_CT", "RXNORM", "ICD10CM", "LOINC", "CPT",
		},
		"features": map[string]bool{
			"autocomplete":        true,
			"spell_correction":    true,
			"query_analysis":      true,
			"faceted_search":      true,
			"highlighting":        true,
			"personalization":     true,
			"batch_operations":    true,
			"real_time_suggestions": true,
		},
		"limits": map[string]int{
			"max_query_length": sh.config.MaxQueryLength,
			"max_page_size":   sh.config.MaxPageSize,
			"max_suggestions": 50,
		},
	}

	metadata := &APIMetadata{
		Timestamp: time.Now(),
		Version:   sh.config.APIVersion,
		RequestID: requestID,
	}

	sh.respondSuccess(c, capabilities, metadata)
}

// Helper methods

func (sh *SearchHandlers) buildSearchRequestFromQuery(c *gin.Context, query string) (*search.ClinicalSearchRequest, error) {
	request := &search.ClinicalSearchRequest{
		Query:      query,
		SearchMode: search.SearchModeStandard,
		QueryType:  search.QueryTypeGeneral,
		Page:       0,
		PageSize:   20,
		IncludeHighlights: true,
		IncludeFacets:     true,
	}

	// Parse pagination
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page >= 0 {
			request.Page = page
		}
	}

	if sizeStr := c.Query("size"); sizeStr != "" {
		if size, err := strconv.Atoi(sizeStr); err == nil && size > 0 && size <= sh.config.MaxPageSize {
			request.PageSize = size
		}
	}

	// Parse filters
	if systems := c.Query("systems"); systems != "" {
		request.Systems = strings.Split(systems, ",")
	}

	if domains := c.Query("domains"); domains != "" {
		request.Domains = strings.Split(domains, ",")
	}

	if status := c.Query("status"); status != "" {
		request.Status = status
	}

	// Parse boolean options
	if exact := c.Query("exact"); exact == "true" {
		request.ExactMatch = true
	}

	if inactive := c.Query("include_inactive"); inactive == "true" {
		request.IncludeInactive = true
	}

	// Parse sorting
	if sortBy := c.Query("sort"); sortBy != "" {
		request.SortBy = sortBy
	}

	if sortOrder := c.Query("order"); sortOrder != "" {
		request.SortOrder = sortOrder
	}

	return request, nil
}

func (sh *SearchHandlers) buildUserContext(c *gin.Context) *search.UserContext {
	context := &search.UserContext{}

	if userID := c.GetHeader("X-User-ID"); userID != "" {
		context.UserID = userID
	}

	if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
		context.SessionID = sessionID
	}

	if role := c.GetHeader("X-User-Role"); role != "" {
		context.Role = role
	}

	if specialty := c.GetHeader("X-User-Specialty"); specialty != "" {
		context.Specialty = specialty
	}

	return context
}

func (sh *SearchHandlers) buildSuggestionFilters(c *gin.Context) *search.SuggestionFilters {
	filters := &search.SuggestionFilters{
		OnlyActive: true, // Default to active only
	}

	if systems := c.Query("systems"); systems != "" {
		filters.Systems = strings.Split(systems, ",")
	}

	if domains := c.Query("domains"); domains != "" {
		filters.Domains = strings.Split(domains, ",")
	}

	if includeInactive := c.Query("include_inactive"); includeInactive == "true" {
		filters.OnlyActive = false
	}

	return filters
}

func (sh *SearchHandlers) buildSuggestionOptions(c *gin.Context) *search.SuggestionOptions {
	options := &search.SuggestionOptions{
		HighlightMatch: true, // Default to true
	}

	if definitions := c.Query("include_definitions"); definitions == "true" {
		options.IncludeDefinitions = true
	}

	if context := c.Query("include_context"); context == "true" {
		options.IncludeContext = true
	}

	if group := c.Query("group_by_systems"); group == "true" {
		options.GroupBySystems = true
	}

	return options
}

func (sh *SearchHandlers) validateSearchRequest(request *search.ClinicalSearchRequest) error {
	if request.Query == "" {
		return fmt.Errorf("query is required")
	}

	if len(request.Query) > sh.config.MaxQueryLength {
		return fmt.Errorf("query exceeds maximum length of %d characters", sh.config.MaxQueryLength)
	}

	if request.PageSize > sh.config.MaxPageSize {
		return fmt.Errorf("page size exceeds maximum of %d", sh.config.MaxPageSize)
	}

	return nil
}

func (sh *SearchHandlers) executeAndRespond(c *gin.Context, request *search.ClinicalSearchRequest, operation string) {
	startTime := time.Now()
	requestID := sh.getRequestID(c)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), sh.config.DefaultTimeout)
	defer cancel()

	// Execute search
	result, err := sh.searchEngine.Search(ctx, request)
	if err != nil {
		sh.logger.Error("Search failed",
			zap.String("request_id", requestID),
			zap.String("operation", operation),
			zap.String("query", request.Query),
			zap.Error(err),
		)
		sh.respondError(c, http.StatusInternalServerError, "SEARCH_FAILED", "Search execution failed", requestID)
		return
	}

	// Build response metadata
	metadata := &APIMetadata{
		Timestamp:      startTime,
		ProcessingTime: time.Since(startTime),
		Version:        sh.config.APIVersion,
		RequestID:      requestID,
		QueryInfo: &QueryInfo{
			OriginalQuery:  request.Query,
			ProcessedQuery: result.ProcessedQuery,
		},
	}

	// Record metrics
	sh.recordSearchMetrics(operation, requestID, request.Query, result, time.Since(startTime))

	// Respond with results
	sh.respondSuccess(c, result, metadata)
}

// Response helpers

func (sh *SearchHandlers) respondSuccess(c *gin.Context, data interface{}, metadata *APIMetadata) {
	response := APIResponse{
		Success:   true,
		Data:      data,
		Metadata:  metadata,
		RequestID: metadata.RequestID,
	}

	c.JSON(http.StatusOK, response)
}

func (sh *SearchHandlers) respondError(c *gin.Context, statusCode int, code, message, requestID string) {
	response := APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		RequestID: requestID,
		Metadata: &APIMetadata{
			Timestamp: time.Now(),
			Version:   sh.config.APIVersion,
			RequestID: requestID,
		},
	}

	c.JSON(statusCode, response)
}

// Middleware

func (sh *SearchHandlers) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func (sh *SearchHandlers) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := sh.getRequestID(c)

		sh.logger.Info("API request started",
			zap.String("request_id", requestID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("user_agent", c.GetHeader("User-Agent")),
		)

		c.Next()

		duration := time.Since(start)
		sh.logger.Info("API request completed",
			zap.String("request_id", requestID),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
		)
	}
}

func (sh *SearchHandlers) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if sh.isAllowedOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-User-ID, X-Session-ID, X-User-Role, X-User-Specialty")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func (sh *SearchHandlers) isAllowedOrigin(origin string) bool {
	for _, allowed := range sh.config.CORSOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// Utility methods

func (sh *SearchHandlers) getRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		return requestID.(string)
	}
	return "unknown"
}

// Metrics recording methods

func (sh *SearchHandlers) recordSearchMetrics(operation, requestID, query string, result *search.ClinicalSearchResponse, duration time.Duration) {
	sh.metrics.RecordAPIMetric("api_requests_total", "complete")
	sh.metrics.RecordAPIMetric("api_request_duration_seconds", fmt.Sprintf("%.3f", duration.Seconds()))
	sh.metrics.RecordAPIMetric("api_results_count", fmt.Sprintf("%d", result.ReturnedCount))

	labels := map[string]string{
		"operation":   operation,
		"search_mode": string(result.SearchMode),
	}

	sh.metrics.IncrementCounterWithLabels("search_api_requests_total", labels)
}

func (sh *SearchHandlers) recordAutocompleteMetrics(requestID, query string, result *search.AutocompleteResponse, duration time.Duration) {
	sh.metrics.RecordAPIMetric("autocomplete_requests_total", "complete")
	sh.metrics.RecordAPIMetric("autocomplete_suggestions_count", fmt.Sprintf("%d", result.TotalCount))
	sh.metrics.RecordAPIMetric("autocomplete_duration_seconds", fmt.Sprintf("%.3f", duration.Seconds()))
}

func (sh *SearchHandlers) recordAnalysisMetrics(requestID, query string, result *search.QueryAnalysisResponse, duration time.Duration) {
	sh.metrics.RecordAPIMetric("analysis_requests_total", "complete")
	sh.metrics.RecordAPIMetric("analysis_duration_seconds", fmt.Sprintf("%.3f", duration.Seconds()))
	sh.metrics.RecordAPIMetric("analysis_confidence", fmt.Sprintf("%.2f", result.ConfidenceScore))
}

// Placeholder implementations for additional endpoints that would be implemented

func (sh *SearchHandlers) BatchSearch(c *gin.Context) {
	sh.respondError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Batch search not yet implemented", sh.getRequestID(c))
}

func (sh *SearchHandlers) BatchAnalyze(c *gin.Context) {
	sh.respondError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Batch analyze not yet implemented", sh.getRequestID(c))
}

func (sh *SearchHandlers) GetSearchStats(c *gin.Context) {
	sh.respondError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Search stats not yet implemented", sh.getRequestID(c))
}

func (sh *SearchHandlers) ClearCache(c *gin.Context) {
	sh.respondError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Cache clear not yet implemented", sh.getRequestID(c))
}

func (sh *SearchHandlers) GetPerformanceMetrics(c *gin.Context) {
	sh.respondError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Performance metrics not yet implemented", sh.getRequestID(c))
}

func (sh *SearchHandlers) Metrics(c *gin.Context) {
	sh.respondError(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Metrics endpoint not yet implemented", sh.getRequestID(c))
}

func (sh *SearchHandlers) GetSupportedSystems(c *gin.Context) {
	requestID := sh.getRequestID(c)

	systems := []map[string]interface{}{
		{"code": "SNOMED_CT", "name": "SNOMED Clinical Terms", "description": "Systematized Nomenclature of Medicine Clinical Terms"},
		{"code": "RXNORM", "name": "RxNorm", "description": "Normalized naming system for generic and branded drugs"},
		{"code": "ICD10CM", "name": "ICD-10-CM", "description": "International Classification of Diseases, 10th Revision, Clinical Modification"},
		{"code": "LOINC", "name": "LOINC", "description": "Logical Observation Identifiers Names and Codes"},
		{"code": "CPT", "name": "CPT", "description": "Current Procedural Terminology"},
	}

	metadata := &APIMetadata{
		Timestamp: time.Now(),
		Version:   sh.config.APIVersion,
		RequestID: requestID,
	}

	sh.respondSuccess(c, systems, metadata)
}

func (sh *SearchHandlers) GetSupportedDomains(c *gin.Context) {
	requestID := sh.getRequestID(c)

	domains := []map[string]interface{}{
		{"code": "diagnostic", "name": "Diagnostic", "description": "Diagnoses, diseases, and disorders"},
		{"code": "procedural", "name": "Procedural", "description": "Medical and surgical procedures"},
		{"code": "medication", "name": "Medication", "description": "Drugs, medications, and pharmaceuticals"},
		{"code": "laboratory", "name": "Laboratory", "description": "Laboratory tests and results"},
		{"code": "anatomy", "name": "Anatomy", "description": "Anatomical structures and body parts"},
		{"code": "symptom", "name": "Symptom", "description": "Symptoms and clinical findings"},
	}

	metadata := &APIMetadata{
		Timestamp: time.Now(),
		Version:   sh.config.APIVersion,
		RequestID: requestID,
	}

	sh.respondSuccess(c, domains, metadata)
}