package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"medication-service-v2/internal/infrastructure"
)

// ApolloFederationService provides high-level access to knowledge bases through Apollo Federation
type ApolloFederationService struct {
	client              *infrastructure.ApolloFederationClient
	logger              *zap.Logger
	cacheService        CacheServiceInterface
	performanceMonitor  PerformanceMonitorInterface
	healthChecker       HealthCheckerInterface
	mu                  sync.RWMutex
	healthy             bool
	lastHealthCheck     time.Time
	healthCheckInterval time.Duration
}

// CacheServiceInterface defines caching operations
type CacheServiceInterface interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
}

// HealthCheckerInterface defines health checking operations
type HealthCheckerInterface interface {
	CheckService(ctx context.Context, serviceName string) (bool, error)
	GetServiceHealth(serviceName string) (bool, time.Time, error)
}

// KnowledgeQueryRequest represents a unified knowledge query request
type KnowledgeQueryRequest struct {
	DrugCode        string                                      `json:"drug_code"`
	PatientContext  *infrastructure.PatientContextInput        `json:"patient_context,omitempty"`
	Version         *string                                     `json:"version,omitempty"`
	Region          *string                                     `json:"region,omitempty"`
	QueryType       string                                      `json:"query_type"` // "dosing", "guidelines", "interactions", "availability"
	Filters         map[string]interface{}                      `json:"filters,omitempty"`
	CacheEnabled    bool                                        `json:"cache_enabled"`
	CacheTTL        time.Duration                              `json:"cache_ttl"`
}

// KnowledgeQueryResponse represents a unified knowledge query response
type KnowledgeQueryResponse struct {
	DrugCode            string                                  `json:"drug_code"`
	DosingRule          *infrastructure.DosingRule             `json:"dosing_rule,omitempty"`
	DosingRecommendation *infrastructure.DosingRecommendation   `json:"dosing_recommendation,omitempty"`
	ClinicalGuidelines  []infrastructure.ClinicalGuideline     `json:"clinical_guidelines,omitempty"`
	Availability        *bool                                   `json:"availability,omitempty"`
	CacheHit            bool                                    `json:"cache_hit"`
	ResponseTime        time.Duration                           `json:"response_time"`
	QueryTimestamp      time.Time                               `json:"query_timestamp"`
}

// BatchKnowledgeQuery represents batch knowledge queries
type BatchKnowledgeQuery struct {
	DrugCodes       []string    `json:"drug_codes"`
	QueryType       string      `json:"query_type"`
	Region          *string     `json:"region,omitempty"`
	CacheEnabled    bool        `json:"cache_enabled"`
	CacheTTL        time.Duration `json:"cache_ttl"`
	MaxConcurrency  int         `json:"max_concurrency"`
}

// NewApolloFederationService creates a new Apollo Federation service
func NewApolloFederationService(
	client *infrastructure.ApolloFederationClient,
	logger *zap.Logger,
	cacheService CacheServiceInterface,
	performanceMonitor PerformanceMonitorInterface,
	healthChecker HealthCheckerInterface,
) *ApolloFederationService {
	service := &ApolloFederationService{
		client:              client,
		logger:              logger,
		cacheService:        cacheService,
		performanceMonitor:  performanceMonitor,
		healthChecker:       healthChecker,
		healthy:             true,
		healthCheckInterval: 30 * time.Second,
	}

	// Start background health checking
	go service.backgroundHealthCheck()

	return service
}

// QueryKnowledge executes a unified knowledge query
func (s *ApolloFederationService) QueryKnowledge(ctx context.Context, request *KnowledgeQueryRequest) (*KnowledgeQueryResponse, error) {
	start := time.Now()
	
	// Check service health first
	if !s.isHealthy() {
		return nil, fmt.Errorf("apollo federation service is unhealthy")
	}

	response := &KnowledgeQueryResponse{
		DrugCode:       request.DrugCode,
		QueryTimestamp: start,
	}

	// Check cache first if enabled
	if request.CacheEnabled {
		cacheKey := s.buildCacheKey(request)
		if err := s.cacheService.Get(ctx, cacheKey, response); err == nil {
			response.CacheHit = true
			response.ResponseTime = time.Since(start)
			
			s.logger.Debug("Knowledge query cache hit",
				zap.String("drug_code", request.DrugCode),
				zap.String("query_type", request.QueryType),
			)
			
			return response, nil
		}
	}

	// Execute query based on type
	var err error
	switch request.QueryType {
	case "dosing":
		if request.PatientContext != nil {
			response.DosingRecommendation, err = s.client.CalculateDosing(
				ctx, request.DrugCode, *request.PatientContext, request.Version, request.Region,
			)
		} else {
			response.DosingRule, err = s.client.GetDosingRule(
				ctx, request.DrugCode, request.Version, request.Region,
			)
		}

	case "guidelines":
		drugClass, _ := request.Filters["drug_class"].(string)
		condition, _ := request.Filters["condition"].(string)
		limitFloat, _ := request.Filters["limit"].(float64)
		limit := int32(limitFloat)
		
		response.ClinicalGuidelines, err = s.client.GetClinicalGuidelines(
			ctx, &drugClass, &condition, &limit,
		)

	case "availability":
		available, availErr := s.client.CheckDosingAvailability(ctx, request.DrugCode, request.Region)
		response.Availability = &available
		err = availErr

	default:
		err = fmt.Errorf("unsupported query type: %s", request.QueryType)
	}

	if err != nil {
		s.logger.Error("Knowledge query failed",
			zap.String("drug_code", request.DrugCode),
			zap.String("query_type", request.QueryType),
			zap.Error(err),
		)
		return nil, fmt.Errorf("knowledge query failed: %w", err)
	}

	response.CacheHit = false
	response.ResponseTime = time.Since(start)

	// Cache the response if enabled
	if request.CacheEnabled && request.CacheTTL > 0 {
		cacheKey := s.buildCacheKey(request)
		if cacheErr := s.cacheService.Set(ctx, cacheKey, response, request.CacheTTL); cacheErr != nil {
			s.logger.Warn("Failed to cache knowledge query response",
				zap.String("cache_key", cacheKey),
				zap.Error(cacheErr),
			)
		}
	}

	// Record performance metrics
	if s.performanceMonitor != nil {
		s.performanceMonitor.RecordAPICall(
			fmt.Sprintf("apollo_federation_%s", request.QueryType),
			response.ResponseTime,
			err == nil,
		)
	}

	s.logger.Info("Knowledge query completed",
		zap.String("drug_code", request.DrugCode),
		zap.String("query_type", request.QueryType),
		zap.Duration("response_time", response.ResponseTime),
		zap.Bool("cache_hit", response.CacheHit),
	)

	return response, nil
}

// BatchQueryKnowledge executes batch knowledge queries with concurrency control
func (s *ApolloFederationService) BatchQueryKnowledge(ctx context.Context, batchQuery *BatchKnowledgeQuery) (map[string]*KnowledgeQueryResponse, error) {
	start := time.Now()
	
	if !s.isHealthy() {
		return nil, fmt.Errorf("apollo federation service is unhealthy")
	}

	if len(batchQuery.DrugCodes) == 0 {
		return make(map[string]*KnowledgeQueryResponse), nil
	}

	// Handle special case for batch dosing rules
	if batchQuery.QueryType == "batch_dosing" {
		return s.executeBatchDosingQuery(ctx, batchQuery)
	}

	// Regular concurrent processing
	results := make(map[string]*KnowledgeQueryResponse)
	resultsMu := sync.Mutex{}
	
	// Control concurrency
	concurrency := batchQuery.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 10 // Default concurrency
	}
	
	semaphore := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	errChan := make(chan error, len(batchQuery.DrugCodes))

	for _, drugCode := range batchQuery.DrugCodes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Create individual query request
			request := &KnowledgeQueryRequest{
				DrugCode:     code,
				Version:      nil,
				Region:       batchQuery.Region,
				QueryType:    batchQuery.QueryType,
				CacheEnabled: batchQuery.CacheEnabled,
				CacheTTL:     batchQuery.CacheTTL,
			}

			response, err := s.QueryKnowledge(ctx, request)
			
			resultsMu.Lock()
			if err != nil {
				// Create error response
				results[code] = &KnowledgeQueryResponse{
					DrugCode:       code,
					QueryTimestamp: time.Now(),
					ResponseTime:   0,
					CacheHit:       false,
				}
				select {
				case errChan <- fmt.Errorf("failed to query %s: %w", code, err):
				default:
				}
			} else {
				results[code] = response
			}
			resultsMu.Unlock()
		}(drugCode)
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var batchErrors []error
	for err := range errChan {
		batchErrors = append(batchErrors, err)
	}

	if len(batchErrors) > 0 && len(batchErrors) == len(batchQuery.DrugCodes) {
		return nil, fmt.Errorf("all batch queries failed: first error: %v", batchErrors[0])
	}

	s.logger.Info("Batch knowledge query completed",
		zap.Int("requested_count", len(batchQuery.DrugCodes)),
		zap.Int("successful_count", len(results)),
		zap.Int("error_count", len(batchErrors)),
		zap.Duration("total_time", time.Since(start)),
	)

	return results, nil
}

// executeBatchDosingQuery executes optimized batch dosing query
func (s *ApolloFederationService) executeBatchDosingQuery(ctx context.Context, batchQuery *BatchKnowledgeQuery) (map[string]*KnowledgeQueryResponse, error) {
	start := time.Now()

	// Use the client's batch method for efficiency
	dosingRules, err := s.client.GetBatchDosingRules(ctx, batchQuery.DrugCodes, batchQuery.Region)
	if err != nil {
		return nil, fmt.Errorf("batch dosing query failed: %w", err)
	}

	// Convert to response format
	results := make(map[string]*KnowledgeQueryResponse)
	for _, rule := range dosingRules {
		response := &KnowledgeQueryResponse{
			DrugCode:       rule.DrugCode,
			DosingRule:     &rule,
			CacheHit:       false,
			ResponseTime:   time.Since(start),
			QueryTimestamp: start,
		}

		// Cache individual responses if enabled
		if batchQuery.CacheEnabled && batchQuery.CacheTTL > 0 {
			request := &KnowledgeQueryRequest{
				DrugCode:     rule.DrugCode,
				QueryType:    "dosing",
				Region:       batchQuery.Region,
				CacheEnabled: true,
				CacheTTL:     batchQuery.CacheTTL,
			}
			cacheKey := s.buildCacheKey(request)
			if cacheErr := s.cacheService.Set(ctx, cacheKey, response, batchQuery.CacheTTL); cacheErr != nil {
				s.logger.Warn("Failed to cache batch dosing response",
					zap.String("drug_code", rule.DrugCode),
					zap.Error(cacheErr),
				)
			}
		}

		results[rule.DrugCode] = response
	}

	// Add empty responses for drugs without rules
	for _, drugCode := range batchQuery.DrugCodes {
		if _, exists := results[drugCode]; !exists {
			results[drugCode] = &KnowledgeQueryResponse{
				DrugCode:       drugCode,
				DosingRule:     nil,
				CacheHit:       false,
				ResponseTime:   time.Since(start),
				QueryTimestamp: start,
			}
		}
	}

	s.logger.Info("Batch dosing query completed",
		zap.Int("requested_count", len(batchQuery.DrugCodes)),
		zap.Int("rules_found", len(dosingRules)),
		zap.Duration("total_time", time.Since(start)),
	)

	return results, nil
}

// GetClinicalIntelligence retrieves comprehensive clinical intelligence for a drug
func (s *ApolloFederationService) GetClinicalIntelligence(ctx context.Context, drugCode string, patientContext *infrastructure.PatientContextInput, region *string) (*ClinicalIntelligenceResponse, error) {
	start := time.Now()
	
	if !s.isHealthy() {
		return nil, fmt.Errorf("apollo federation service is unhealthy")
	}

	response := &ClinicalIntelligenceResponse{
		DrugCode:       drugCode,
		QueryTimestamp: start,
	}

	// Execute multiple queries concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)

	// Query dosing rule/recommendation
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		if patientContext != nil {
			dosingRec, err := s.client.CalculateDosing(ctx, drugCode, *patientContext, nil, region)
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("dosing calculation failed: %w", err))
			} else {
				response.DosingRecommendation = dosingRec
			}
			mu.Unlock()
		} else {
			dosingRule, err := s.client.GetDosingRule(ctx, drugCode, nil, region)
			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("dosing rule query failed: %w", err))
			} else {
				response.DosingRule = dosingRule
			}
			mu.Unlock()
		}
	}()

	// Query clinical guidelines
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		limit := int32(10)
		guidelines, err := s.client.GetClinicalGuidelines(ctx, nil, nil, &limit)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("guidelines query failed: %w", err))
		} else {
			response.ClinicalGuidelines = guidelines
		}
		mu.Unlock()
	}()

	// Check availability
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		available, err := s.client.CheckDosingAvailability(ctx, drugCode, region)
		mu.Lock()
		if err != nil {
			errors = append(errors, fmt.Errorf("availability check failed: %w", err))
		} else {
			response.Available = available
		}
		mu.Unlock()
	}()

	wg.Wait()

	response.ResponseTime = time.Since(start)

	if len(errors) > 0 {
		s.logger.Warn("Some clinical intelligence queries failed",
			zap.String("drug_code", drugCode),
			zap.Int("error_count", len(errors)),
		)
		// Don't fail entirely if some queries succeed
	}

	s.logger.Info("Clinical intelligence query completed",
		zap.String("drug_code", drugCode),
		zap.Duration("response_time", response.ResponseTime),
		zap.Int("error_count", len(errors)),
	)

	return response, nil
}

// ClinicalIntelligenceResponse represents comprehensive clinical intelligence
type ClinicalIntelligenceResponse struct {
	DrugCode             string                                   `json:"drug_code"`
	DosingRule           *infrastructure.DosingRule              `json:"dosing_rule,omitempty"`
	DosingRecommendation *infrastructure.DosingRecommendation    `json:"dosing_recommendation,omitempty"`
	ClinicalGuidelines   []infrastructure.ClinicalGuideline      `json:"clinical_guidelines"`
	Available            bool                                     `json:"available"`
	ResponseTime         time.Duration                            `json:"response_time"`
	QueryTimestamp       time.Time                                `json:"query_timestamp"`
}

// HealthCheck performs a health check on the Apollo Federation gateway
func (s *ApolloFederationService) HealthCheck(ctx context.Context) error {
	return s.client.HealthCheck(ctx)
}

// isHealthy returns the current health status
func (s *ApolloFederationService) isHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.healthy
}

// setHealthy sets the health status
func (s *ApolloFederationService) setHealthy(healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.healthy = healthy
	s.lastHealthCheck = time.Now()
}

// backgroundHealthCheck performs periodic health checks
func (s *ApolloFederationService) backgroundHealthCheck() {
	ticker := time.NewTicker(s.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := s.HealthCheck(ctx)
			cancel()

			if err != nil {
				s.logger.Warn("Apollo Federation health check failed",
					zap.Error(err),
				)
				s.setHealthy(false)
			} else {
				if !s.isHealthy() {
					s.logger.Info("Apollo Federation service recovered")
				}
				s.setHealthy(true)
			}
		}
	}
}

// buildCacheKey builds a cache key for knowledge query requests
func (s *ApolloFederationService) buildCacheKey(request *KnowledgeQueryRequest) string {
	key := fmt.Sprintf("apollo_federation:%s:%s", request.QueryType, request.DrugCode)
	
	if request.Version != nil {
		key += fmt.Sprintf(":v%s", *request.Version)
	}
	
	if request.Region != nil {
		key += fmt.Sprintf(":r%s", *request.Region)
	}

	if request.PatientContext != nil {
		// Add patient context hash for personalized queries
		key += fmt.Sprintf(":ctx_%d_%d_%.1f", 
			request.PatientContext.AgeYears,
			int(request.PatientContext.WeightKg),
			request.PatientContext.EGFR,
		)
	}

	return key
}

// GetServiceMetrics returns performance metrics for the service
func (s *ApolloFederationService) GetServiceMetrics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"healthy":            s.healthy,
		"last_health_check":  s.lastHealthCheck,
		"gateway_url":        s.client,  // Will show gateway URL in logs
	}
}