package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"medication-service-v2/internal/infrastructure"
)

// KnowledgeBaseIntegrationService provides unified access to all knowledge bases through Apollo Federation
type KnowledgeBaseIntegrationService struct {
	apolloFederationService *ApolloFederationService
	logger                  *zap.Logger
	cacheService           CacheServiceInterface
	queryBuilder           *infrastructure.GraphQLQueryBuilder
	performanceMonitor     PerformanceMonitorInterface
	circuitBreaker         CircuitBreakerInterface
	
	// Configuration
	defaultCacheTTL     time.Duration
	batchSize          int
	maxConcurrency     int
	queryTimeout       time.Duration
}

// CircuitBreakerInterface defines circuit breaker operations
type CircuitBreakerInterface interface {
	Execute(ctx context.Context, operation string, fn func() error) error
	GetStatus(operation string) string
	IsOpen(operation string) bool
}

// KnowledgeBaseQueryRequest represents a unified knowledge base query
type KnowledgeBaseQueryRequest struct {
	// Core identification
	DrugCode     string    `json:"drug_code"`
	DrugCodes    []string  `json:"drug_codes,omitempty"`    // For batch queries
	PatientID    *string   `json:"patient_id,omitempty"`
	
	// Patient context for personalized recommendations
	PatientContext *infrastructure.PatientContextInput `json:"patient_context,omitempty"`
	
	// Query parameters
	Version      *string   `json:"version,omitempty"`
	Region       *string   `json:"region,omitempty"`
	QueryTypes   []string  `json:"query_types"`              // ["dosing", "guidelines", "interactions", "safety"]
	
	// Filters and options
	Filters      map[string]interface{} `json:"filters,omitempty"`
	Limit        *int32                 `json:"limit,omitempty"`
	Fields       []string               `json:"fields,omitempty"`      // For optimized queries
	
	// Performance options
	CacheEnabled     bool          `json:"cache_enabled"`
	CacheTTL         time.Duration `json:"cache_ttl,omitempty"`
	MaxConcurrency   int           `json:"max_concurrency,omitempty"`
	TimeoutOverride  *time.Duration `json:"timeout_override,omitempty"`
	Priority         string        `json:"priority,omitempty"`        // "low", "normal", "high", "critical"
}

// KnowledgeBaseQueryResponse represents a unified knowledge base response
type KnowledgeBaseQueryResponse struct {
	// Request context
	DrugCode       string    `json:"drug_code"`
	DrugCodes      []string  `json:"drug_codes,omitempty"`
	QueryTypes     []string  `json:"query_types"`
	RequestID      string    `json:"request_id"`
	
	// Knowledge base results
	DosingRules           []infrastructure.DosingRule              `json:"dosing_rules,omitempty"`
	DosingRecommendations []infrastructure.DosingRecommendation    `json:"dosing_recommendations,omitempty"`
	ClinicalGuidelines    []infrastructure.ClinicalGuideline       `json:"clinical_guidelines,omitempty"`
	DrugInteractions      []DrugInteraction                        `json:"drug_interactions,omitempty"`
	SafetyAlerts          []infrastructure.SafetyAlert             `json:"safety_alerts,omitempty"`
	
	// Availability and metadata
	AvailabilityStatus    map[string]bool        `json:"availability_status,omitempty"`
	KnowledgeBaseStatus   map[string]string      `json:"knowledge_base_status,omitempty"`
	
	// Performance metadata
	QueryMetrics     QueryMetrics  `json:"query_metrics"`
	CacheStatus      CacheStatus   `json:"cache_status"`
	ExecutionSummary ExecutionSummary `json:"execution_summary"`
}

// DrugInteraction represents drug interaction data (future KB5 integration)
type DrugInteraction struct {
	InteractionID    string                 `json:"interaction_id"`
	DrugA           DrugInfo               `json:"drug_a"`
	DrugB           DrugInfo               `json:"drug_b"`
	Severity        string                 `json:"severity"`
	Mechanism       string                 `json:"mechanism"`
	ClinicalEffect  string                 `json:"clinical_effect"`
	Management      string                 `json:"management"`
	Evidence        InteractionEvidence    `json:"evidence"`
	LastUpdated     time.Time             `json:"last_updated"`
}

type DrugInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type InteractionEvidence struct {
	Level       string   `json:"level"`
	Studies     []string `json:"studies"`
	References  []string `json:"references"`
}

// QueryMetrics tracks performance metrics for knowledge base queries
type QueryMetrics struct {
	TotalDuration        time.Duration `json:"total_duration"`
	KnowledgeBaseLatency map[string]time.Duration `json:"knowledge_base_latency"`
	NetworkLatency       time.Duration `json:"network_latency"`
	ProcessingTime       time.Duration `json:"processing_time"`
	RetryCount          int           `json:"retry_count"`
	ErrorCount          int           `json:"error_count"`
}

// CacheStatus tracks cache hit/miss information
type CacheStatus struct {
	Enabled     bool              `json:"enabled"`
	HitRate     float64           `json:"hit_rate"`
	HitsByType  map[string]int    `json:"hits_by_type"`
	MissesByType map[string]int   `json:"misses_by_type"`
	TTLUsed     time.Duration     `json:"ttl_used"`
}

// ExecutionSummary provides high-level execution information
type ExecutionSummary struct {
	TotalQueries      int       `json:"total_queries"`
	SuccessfulQueries int       `json:"successful_queries"`
	FailedQueries     int       `json:"failed_queries"`
	PartialResults    bool      `json:"partial_results"`
	Warnings          []string  `json:"warnings"`
	Timestamp         time.Time `json:"timestamp"`
}

// NewKnowledgeBaseIntegrationService creates a new knowledge base integration service
func NewKnowledgeBaseIntegrationService(
	apolloFederationService *ApolloFederationService,
	logger *zap.Logger,
	cacheService CacheServiceInterface,
	performanceMonitor PerformanceMonitorInterface,
	circuitBreaker CircuitBreakerInterface,
) *KnowledgeBaseIntegrationService {
	return &KnowledgeBaseIntegrationService{
		apolloFederationService: apolloFederationService,
		logger:                  logger,
		cacheService:           cacheService,
		queryBuilder:           infrastructure.NewGraphQLQueryBuilder(),
		performanceMonitor:     performanceMonitor,
		circuitBreaker:         circuitBreaker,
		
		// Default configuration
		defaultCacheTTL: 30 * time.Minute,
		batchSize:       50,
		maxConcurrency:  10,
		queryTimeout:    30 * time.Second,
	}
}

// QueryKnowledgeBases executes a unified query across all relevant knowledge bases
func (s *KnowledgeBaseIntegrationService) QueryKnowledgeBases(ctx context.Context, request *KnowledgeBaseQueryRequest) (*KnowledgeBaseQueryResponse, error) {
	start := time.Now()
	requestID := fmt.Sprintf("kb_query_%d", time.Now().UnixNano())
	
	s.logger.Info("Starting knowledge base query",
		zap.String("request_id", requestID),
		zap.String("drug_code", request.DrugCode),
		zap.Strings("query_types", request.QueryTypes),
	)

	response := &KnowledgeBaseQueryResponse{
		DrugCode:    request.DrugCode,
		DrugCodes:   request.DrugCodes,
		QueryTypes:  request.QueryTypes,
		RequestID:   requestID,
		QueryMetrics: QueryMetrics{
			KnowledgeBaseLatency: make(map[string]time.Duration),
		},
		CacheStatus: CacheStatus{
			Enabled:      request.CacheEnabled,
			HitsByType:   make(map[string]int),
			MissesByType: make(map[string]int),
		},
	}

	// Apply timeout override if specified
	timeout := s.queryTimeout
	if request.TimeoutOverride != nil {
		timeout = *request.TimeoutOverride
	}
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute queries concurrently based on request types
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	
	for _, queryType := range request.QueryTypes {
		wg.Add(1)
		go func(qType string) {
			defer wg.Done()
			
			queryStart := time.Now()
			err := s.executeKnowledgeBaseQuery(ctx, qType, request, response, &mu)
			queryDuration := time.Since(queryStart)
			
			mu.Lock()
			response.QueryMetrics.KnowledgeBaseLatency[qType] = queryDuration
			if err != nil {
				errors = append(errors, fmt.Errorf("%s query failed: %w", qType, err))
				response.QueryMetrics.ErrorCount++
			}
			response.QueryMetrics.TotalQueries++
			if err == nil {
				response.QueryMetrics.SuccessfulQueries++
			} else {
				response.QueryMetrics.FailedQueries++
			}
			mu.Unlock()
		}(queryType)
	}

	wg.Wait()

	// Calculate final metrics
	response.QueryMetrics.TotalDuration = time.Since(start)
	response.ExecutionSummary = ExecutionSummary{
		TotalQueries:      response.QueryMetrics.TotalQueries,
		SuccessfulQueries: response.QueryMetrics.SuccessfulQueries,
		FailedQueries:     response.QueryMetrics.FailedQueries,
		PartialResults:    len(errors) > 0 && len(errors) < len(request.QueryTypes),
		Timestamp:         start,
	}

	// Add warnings for failed queries
	for _, err := range errors {
		response.ExecutionSummary.Warnings = append(response.ExecutionSummary.Warnings, err.Error())
	}

	// Record performance metrics
	if s.performanceMonitor != nil {
		s.performanceMonitor.RecordAPICall(
			"knowledge_base_unified_query",
			response.QueryMetrics.TotalDuration,
			len(errors) == 0,
		)
	}

	s.logger.Info("Knowledge base query completed",
		zap.String("request_id", requestID),
		zap.Duration("total_duration", response.QueryMetrics.TotalDuration),
		zap.Int("successful_queries", response.QueryMetrics.SuccessfulQueries),
		zap.Int("failed_queries", response.QueryMetrics.FailedQueries),
		zap.Bool("partial_results", response.ExecutionSummary.PartialResults),
	)

	// Return partial results if some queries succeeded
	if response.QueryMetrics.SuccessfulQueries > 0 {
		return response, nil
	}

	// All queries failed
	if len(errors) > 0 {
		return nil, fmt.Errorf("all knowledge base queries failed: first error: %w", errors[0])
	}

	return response, nil
}

// executeKnowledgeBaseQuery executes a specific type of knowledge base query
func (s *KnowledgeBaseIntegrationService) executeKnowledgeBaseQuery(
	ctx context.Context,
	queryType string,
	request *KnowledgeBaseQueryRequest, 
	response *KnowledgeBaseQueryResponse,
	mu *sync.Mutex,
) error {
	
	// Use circuit breaker for reliability
	return s.circuitBreaker.Execute(ctx, fmt.Sprintf("kb_%s", queryType), func() error {
		switch queryType {
		case "dosing":
			return s.queryDosingKnowledge(ctx, request, response, mu)
		case "guidelines":
			return s.queryGuidelinesKnowledge(ctx, request, response, mu)
		case "interactions":
			return s.queryInteractionsKnowledge(ctx, request, response, mu)
		case "safety":
			return s.querySafetyKnowledge(ctx, request, response, mu)
		case "availability":
			return s.queryAvailabilityKnowledge(ctx, request, response, mu)
		default:
			return fmt.Errorf("unsupported query type: %s", queryType)
		}
	})
}

// queryDosingKnowledge queries dosing-related knowledge (KB1)
func (s *KnowledgeBaseIntegrationService) queryDosingKnowledge(
	ctx context.Context,
	request *KnowledgeBaseQueryRequest,
	response *KnowledgeBaseQueryResponse,
	mu *sync.Mutex,
) error {
	
	// Build Apollo Federation query request
	fedRequest := &KnowledgeQueryRequest{
		DrugCode:     request.DrugCode,
		Version:      request.Version,
		Region:       request.Region,
		QueryType:    "dosing",
		CacheEnabled: request.CacheEnabled,
		CacheTTL:     s.getCacheTTL(request),
	}

	if request.PatientContext != nil {
		fedRequest.PatientContext = request.PatientContext
	}

	fedResponse, err := s.apolloFederationService.QueryKnowledge(ctx, fedRequest)
	if err != nil {
		return fmt.Errorf("failed to query dosing knowledge: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Add results to response
	if fedResponse.DosingRule != nil {
		response.DosingRules = append(response.DosingRules, *fedResponse.DosingRule)
	}
	if fedResponse.DosingRecommendation != nil {
		response.DosingRecommendations = append(response.DosingRecommendations, *fedResponse.DosingRecommendation)
	}

	// Update cache status
	if fedResponse.CacheHit {
		response.CacheStatus.HitsByType["dosing"]++
	} else {
		response.CacheStatus.MissesByType["dosing"]++
	}

	return nil
}

// queryGuidelinesKnowledge queries clinical guidelines (KB3)
func (s *KnowledgeBaseIntegrationService) queryGuidelinesKnowledge(
	ctx context.Context,
	request *KnowledgeBaseQueryRequest,
	response *KnowledgeBaseQueryResponse,
	mu *sync.Mutex,
) error {
	
	// Build guidelines query
	fedRequest := &KnowledgeQueryRequest{
		DrugCode:     request.DrugCode,
		QueryType:    "guidelines",
		CacheEnabled: request.CacheEnabled,
		CacheTTL:     s.getCacheTTL(request),
		Filters:      request.Filters,
	}

	fedResponse, err := s.apolloFederationService.QueryKnowledge(ctx, fedRequest)
	if err != nil {
		return fmt.Errorf("failed to query guidelines knowledge: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Add results to response
	if len(fedResponse.ClinicalGuidelines) > 0 {
		response.ClinicalGuidelines = append(response.ClinicalGuidelines, fedResponse.ClinicalGuidelines...)
	}

	// Update cache status
	if fedResponse.CacheHit {
		response.CacheStatus.HitsByType["guidelines"]++
	} else {
		response.CacheStatus.MissesByType["guidelines"]++
	}

	return nil
}

// queryInteractionsKnowledge queries drug interactions (future KB5)
func (s *KnowledgeBaseIntegrationService) queryInteractionsKnowledge(
	ctx context.Context,
	request *KnowledgeBaseQueryRequest,
	response *KnowledgeBaseQueryResponse,
	mu *sync.Mutex,
) error {
	
	// Placeholder for future KB5 drug interactions integration
	s.logger.Debug("Drug interactions knowledge base not yet implemented",
		zap.String("drug_code", request.DrugCode),
	)

	mu.Lock()
	defer mu.Unlock()

	// Initialize empty interactions list
	response.DrugInteractions = []DrugInteraction{}
	response.CacheStatus.MissesByType["interactions"]++

	return nil
}

// querySafetyKnowledge queries patient safety knowledge (KB4)
func (s *KnowledgeBaseIntegrationService) querySafetyKnowledge(
	ctx context.Context,
	request *KnowledgeBaseQueryRequest,
	response *KnowledgeBaseQueryResponse,
	mu *sync.Mutex,
) error {
	
	// Placeholder for future KB4 patient safety integration
	s.logger.Debug("Patient safety knowledge base not yet fully implemented",
		zap.String("drug_code", request.DrugCode),
	)

	mu.Lock()
	defer mu.Unlock()

	// Initialize empty safety alerts list
	response.SafetyAlerts = []infrastructure.SafetyAlert{}
	response.CacheStatus.MissesByType["safety"]++

	return nil
}

// queryAvailabilityKnowledge queries knowledge base availability
func (s *KnowledgeBaseIntegrationService) queryAvailabilityKnowledge(
	ctx context.Context,
	request *KnowledgeBaseQueryRequest,
	response *KnowledgeBaseQueryResponse,
	mu *sync.Mutex,
) error {
	
	fedRequest := &KnowledgeQueryRequest{
		DrugCode:     request.DrugCode,
		QueryType:    "availability",
		Region:       request.Region,
		CacheEnabled: request.CacheEnabled,
		CacheTTL:     s.getCacheTTL(request),
	}

	fedResponse, err := s.apolloFederationService.QueryKnowledge(ctx, fedRequest)
	if err != nil {
		return fmt.Errorf("failed to query availability knowledge: %w", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Add availability results
	if response.AvailabilityStatus == nil {
		response.AvailabilityStatus = make(map[string]bool)
	}
	
	if fedResponse.Availability != nil {
		response.AvailabilityStatus[request.DrugCode] = *fedResponse.Availability
	}

	// Update cache status
	if fedResponse.CacheHit {
		response.CacheStatus.HitsByType["availability"]++
	} else {
		response.CacheStatus.MissesByType["availability"]++
	}

	return nil
}

// BatchQueryKnowledgeBases executes batch queries across knowledge bases
func (s *KnowledgeBaseIntegrationService) BatchQueryKnowledgeBases(ctx context.Context, requests []*KnowledgeBaseQueryRequest) (map[string]*KnowledgeBaseQueryResponse, error) {
	start := time.Now()
	
	s.logger.Info("Starting batch knowledge base queries",
		zap.Int("batch_size", len(requests)),
	)

	// Group requests by type for optimization
	groupedRequests := s.groupRequestsByType(requests)
	
	results := make(map[string]*KnowledgeBaseQueryResponse)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errors := make([]error, 0)

	// Process each group concurrently
	for queryType, typeRequests := range groupedRequests {
		wg.Add(1)
		go func(qType string, reqs []*KnowledgeBaseQueryRequest) {
			defer wg.Done()
			
			typeResults, err := s.processBatchGroup(ctx, qType, reqs)
			
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("batch %s queries failed: %w", qType, err))
			} else {
				// Merge results
				for k, v := range typeResults {
					if existing, exists := results[k]; exists {
						// Merge with existing response
						s.mergeResponses(existing, v)
					} else {
						results[k] = v
					}
				}
			}
			mu.Unlock()
		}(queryType, typeRequests)
	}

	wg.Wait()

	s.logger.Info("Batch knowledge base queries completed",
		zap.Int("batch_size", len(requests)),
		zap.Int("result_count", len(results)),
		zap.Int("error_count", len(errors)),
		zap.Duration("total_duration", time.Since(start)),
	)

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("all batch queries failed: first error: %w", errors[0])
	}

	return results, nil
}

// groupRequestsByType groups requests by their primary query types for batch optimization
func (s *KnowledgeBaseIntegrationService) groupRequestsByType(requests []*KnowledgeBaseQueryRequest) map[string][]*KnowledgeBaseQueryRequest {
	groups := make(map[string][]*KnowledgeBaseQueryRequest)
	
	for _, request := range requests {
		// Use the first query type as the primary grouping
		if len(request.QueryTypes) > 0 {
			primaryType := request.QueryTypes[0]
			groups[primaryType] = append(groups[primaryType], request)
		}
	}
	
	return groups
}

// processBatchGroup processes a batch of requests of the same type
func (s *KnowledgeBaseIntegrationService) processBatchGroup(ctx context.Context, queryType string, requests []*KnowledgeBaseQueryRequest) (map[string]*KnowledgeBaseQueryResponse, error) {
	results := make(map[string]*KnowledgeBaseQueryResponse)
	
	// For dosing queries, use the Apollo Federation batch optimization
	if queryType == "dosing" {
		return s.processBatchDosingQueries(ctx, requests)
	}

	// For other types, process individually with concurrency control
	return s.processIndividualBatchQueries(ctx, requests)
}

// processBatchDosingQueries optimizes batch dosing queries
func (s *KnowledgeBaseIntegrationService) processBatchDosingQueries(ctx context.Context, requests []*KnowledgeBaseQueryRequest) (map[string]*KnowledgeBaseQueryResponse, error) {
	
	// Extract drug codes and common parameters
	drugCodes := make([]string, len(requests))
	var commonRegion *string
	
	for i, req := range requests {
		drugCodes[i] = req.DrugCode
		if i == 0 {
			commonRegion = req.Region
		}
	}

	// Execute batch query
	batchQuery := &BatchKnowledgeQuery{
		DrugCodes:      drugCodes,
		QueryType:      "batch_dosing",
		Region:         commonRegion,
		CacheEnabled:   true,
		CacheTTL:       s.defaultCacheTTL,
		MaxConcurrency: s.maxConcurrency,
	}

	batchResults, err := s.apolloFederationService.BatchQueryKnowledge(ctx, batchQuery)
	if err != nil {
		return nil, fmt.Errorf("batch dosing query failed: %w", err)
	}

	// Convert to standard format
	results := make(map[string]*KnowledgeBaseQueryResponse)
	for drugCode, fedResponse := range batchResults {
		response := &KnowledgeBaseQueryResponse{
			DrugCode:   drugCode,
			QueryTypes: []string{"dosing"},
			RequestID:  fmt.Sprintf("batch_%s_%d", drugCode, time.Now().UnixNano()),
		}

		if fedResponse.DosingRule != nil {
			response.DosingRules = []infrastructure.DosingRule{*fedResponse.DosingRule}
		}

		results[drugCode] = response
	}

	return results, nil
}

// processIndividualBatchQueries processes batch queries individually with concurrency
func (s *KnowledgeBaseIntegrationService) processIndividualBatchQueries(ctx context.Context, requests []*KnowledgeBaseQueryRequest) (map[string]*KnowledgeBaseQueryResponse, error) {
	
	results := make(map[string]*KnowledgeBaseQueryResponse)
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// Use semaphore for concurrency control
	semaphore := make(chan struct{}, s.maxConcurrency)
	errors := make([]error, 0)

	for _, request := range requests {
		wg.Add(1)
		go func(req *KnowledgeBaseQueryRequest) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			response, err := s.QueryKnowledgeBases(ctx, req)
			
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("query for %s failed: %w", req.DrugCode, err))
			} else {
				results[req.DrugCode] = response
			}
			mu.Unlock()
		}(request)
	}

	wg.Wait()

	if len(errors) > 0 && len(results) == 0 {
		return nil, fmt.Errorf("all individual batch queries failed: first error: %w", errors[0])
	}

	return results, nil
}

// mergeResponses merges two knowledge base responses
func (s *KnowledgeBaseIntegrationService) mergeResponses(existing, new *KnowledgeBaseQueryResponse) {
	// Merge query types
	queryTypes := make(map[string]bool)
	for _, qt := range existing.QueryTypes {
		queryTypes[qt] = true
	}
	for _, qt := range new.QueryTypes {
		queryTypes[qt] = true
	}
	
	existing.QueryTypes = make([]string, 0, len(queryTypes))
	for qt := range queryTypes {
		existing.QueryTypes = append(existing.QueryTypes, qt)
	}

	// Merge results
	existing.DosingRules = append(existing.DosingRules, new.DosingRules...)
	existing.DosingRecommendations = append(existing.DosingRecommendations, new.DosingRecommendations...)
	existing.ClinicalGuidelines = append(existing.ClinicalGuidelines, new.ClinicalGuidelines...)
	existing.DrugInteractions = append(existing.DrugInteractions, new.DrugInteractions...)
	existing.SafetyAlerts = append(existing.SafetyAlerts, new.SafetyAlerts...)

	// Merge availability status
	if existing.AvailabilityStatus == nil {
		existing.AvailabilityStatus = make(map[string]bool)
	}
	for k, v := range new.AvailabilityStatus {
		existing.AvailabilityStatus[k] = v
	}

	// Update metrics
	existing.QueryMetrics.TotalQueries += new.QueryMetrics.TotalQueries
	existing.QueryMetrics.SuccessfulQueries += new.QueryMetrics.SuccessfulQueries
	existing.QueryMetrics.FailedQueries += new.QueryMetrics.FailedQueries
	
	// Merge cache hit counts
	for k, v := range new.CacheStatus.HitsByType {
		existing.CacheStatus.HitsByType[k] += v
	}
	for k, v := range new.CacheStatus.MissesByType {
		existing.CacheStatus.MissesByType[k] += v
	}
}

// getCacheTTL returns the appropriate cache TTL for a request
func (s *KnowledgeBaseIntegrationService) getCacheTTL(request *KnowledgeBaseQueryRequest) time.Duration {
	if request.CacheTTL > 0 {
		return request.CacheTTL
	}
	
	// Different TTLs based on query type and priority
	switch {
	case request.Priority == "critical":
		return 5 * time.Minute  // Short TTL for critical queries
	case request.PatientContext != nil:
		return 15 * time.Minute // Shorter TTL for personalized queries
	default:
		return s.defaultCacheTTL
	}
}

// GetServiceHealth returns the health status of all knowledge base services
func (s *KnowledgeBaseIntegrationService) GetServiceHealth(ctx context.Context) (map[string]interface{}, error) {
	health := make(map[string]interface{})
	
	// Check Apollo Federation service health
	federationHealth := s.apolloFederationService.HealthCheck(ctx)
	health["apollo_federation"] = map[string]interface{}{
		"healthy": federationHealth == nil,
		"error":   formatError(federationHealth),
	}

	// Add service metrics
	health["metrics"] = s.apolloFederationService.GetServiceMetrics()
	
	// Add configuration info
	health["configuration"] = map[string]interface{}{
		"default_cache_ttl": s.defaultCacheTTL,
		"batch_size":       s.batchSize,
		"max_concurrency":  s.maxConcurrency,
		"query_timeout":    s.queryTimeout,
	}

	return health, nil
}

// formatError safely formats an error for health check responses
func formatError(err error) interface{} {
	if err == nil {
		return nil
	}
	return err.Error()
}