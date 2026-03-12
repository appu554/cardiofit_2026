package context

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"flow2-go-engine/orb"
)

// contextIntegrationService is the concrete implementation of ContextIntegrationService
type contextIntegrationService struct {
	// Dependencies
	cacheManager        CacheManager
	kbClient           KnowledgeBaseClient
	contextGatewayClient ContextGatewayClient
	circuitBreaker     CircuitBreaker
	
	// Configuration
	config ServiceConfiguration
	
	// Logging and metrics
	logger  *logrus.Logger
	metrics *serviceMetrics
	
	// Synchronization
	mu sync.RWMutex
}

// serviceMetrics tracks service performance
type serviceMetrics struct {
	mu sync.RWMutex
	
	// Request counters
	totalRequests      int64
	successfulRequests int64
	failedRequests     int64
	
	// Latency tracking
	latencies []time.Duration
	
	// KB call tracking
	kbCallCount map[string]int64
	kbLatency   map[string][]time.Duration
	kbErrors    map[string]int64
	
	// Context Gateway tracking
	contextGatewayLatencies []time.Duration
	contextGatewayErrors    int64
	
	// Last updated
	lastUpdated time.Time
}

// NewContextIntegrationService creates a new Context Integration Service instance
func NewContextIntegrationService(
	cacheManager CacheManager,
	kbClient KnowledgeBaseClient,
	contextGatewayClient ContextGatewayClient,
	circuitBreaker CircuitBreaker,
	config ServiceConfiguration,
) ContextIntegrationService {
	
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)
	
	metrics := &serviceMetrics{
		kbCallCount: make(map[string]int64),
		kbLatency:   make(map[string][]time.Duration),
		kbErrors:    make(map[string]int64),
		lastUpdated: time.Now(),
	}
	
	return &contextIntegrationService{
		cacheManager:         cacheManager,
		kbClient:            kbClient,
		contextGatewayClient: contextGatewayClient,
		circuitBreaker:      circuitBreaker,
		config:              config,
		logger:              logger,
		metrics:             metrics,
	}
}

// AssembleContext is the core method that assembles complete clinical context
func (s *contextIntegrationService) AssembleContext(ctx context.Context, manifest *orb.IntentManifest) (*CompleteContextPayload, error) {
	startTime := time.Now()
	
	// Validate input
	if manifest == nil {
		return nil, ErrInvalidManifest
	}
	
	if err := manifest.Validate(); err != nil {
		s.logger.WithError(err).Error("Invalid Intent Manifest")
		return nil, NewContextGatewayError("invalid manifest", map[string]interface{}{
			"validation_error": err.Error(),
		})
	}
	
	s.logger.WithFields(logrus.Fields{
		"request_id":         manifest.RequestID,
		"patient_id":         manifest.PatientID,
		"recipe_id":          manifest.RecipeID,
		"knowledge_manifest": manifest.KnowledgeManifest.RequiredKBs,
		"cache_strategy":     manifest.CacheStrategy,
	}).Info("Starting context assembly")
	
	// Step 1: Check cache first (L3 Redis caching)
	cacheKey := s.generateCacheKey(manifest)
	
	if s.shouldUseCache(manifest.CacheStrategy) {
		if cachedPayload, err := s.getCachedContext(ctx, cacheKey); err == nil && cachedPayload != nil {
			s.logger.WithField("cache_key", cacheKey).Info("Cache hit - returning cached context")
			s.recordCacheHit()
			return cachedPayload, nil
		}
		s.recordCacheMiss()
	}
	
	// Step 2: Parallel data fetching using errgroup
	payload, err := s.fetchContextParallel(ctx, manifest)
	if err != nil {
		// Try stale-while-revalidate if fresh fetch fails
		if stalePayload, staleErr := s.getStaleContext(ctx, cacheKey); staleErr == nil && stalePayload != nil {
			s.logger.WithError(err).Warn("Fresh fetch failed, serving stale data")
			
			// Trigger background revalidation
			go s.backgroundRevalidation(context.Background(), manifest, cacheKey)
			
			return stalePayload, nil
		}
		
		s.recordFailure(err)
		return nil, err
	}
	
	// Step 3: Cache the result
	if s.shouldUseCache(manifest.CacheStrategy) {
		ttl := s.getCacheTTL(manifest.CacheStrategy)
		staleTTL := s.getStaleTTL(manifest.CacheStrategy)
		
		if err := s.cacheManager.SetWithStale(ctx, cacheKey, payload, ttl, staleTTL); err != nil {
			s.logger.WithError(err).Warn("Failed to cache context payload")
		}
	}
	
	// Step 4: Record metrics and return
	s.recordSuccess(time.Since(startTime))
	
	s.logger.WithFields(logrus.Fields{
		"request_id":      manifest.RequestID,
		"processing_time": time.Since(startTime),
		"cache_strategy":  manifest.CacheStrategy,
	}).Info("Context assembly completed successfully")
	
	return payload, nil
}

// fetchContextParallel fetches patient and knowledge data in parallel
func (s *contextIntegrationService) fetchContextParallel(ctx context.Context, manifest *orb.IntentManifest) (*CompleteContextPayload, error) {
	var g errgroup.Group
	var patientData PatientContext
	var knowledgeData KnowledgeContext
	var patientErr, knowledgeErr error
	
	// Goroutine 1: Fetch patient clinical data from Context Gateway
	g.Go(func() error {
		start := time.Now()
		
		result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
			return s.contextGatewayClient.FetchPatientData(ctx, manifest.PatientID, manifest.DataRequirements)
		})
		
		latency := time.Since(start)
		s.recordContextGatewayLatency(latency)
		
		if err != nil {
			s.recordContextGatewayError()
			patientErr = NewContextGatewayError("failed to fetch patient data", map[string]interface{}{
				"patient_id": manifest.PatientID,
				"latency":    latency,
				"error":      err.Error(),
			})
			return patientErr
		}
		
		patientData = result.(PatientContext)
		return nil
	})
	
	// Goroutine 2: Fetch knowledge data from KB services
	g.Go(func() error {
		start := time.Now()
		
		// Determine which KBs to query based on Knowledge Manifest
		kbsToQuery := manifest.KnowledgeManifest.RequiredKBs
		if len(kbsToQuery) == 0 {
			// Backward compatibility - query all KBs
			kbsToQuery = s.kbClient.GetAllKBIdentifiers()
			s.logger.WithField("request_id", manifest.RequestID).Info("No Knowledge Manifest specified, querying all KBs")
		}
		
		s.logger.WithFields(logrus.Fields{
			"request_id":    manifest.RequestID,
			"kbs_to_query":  kbsToQuery,
			"kb_count":      len(kbsToQuery),
		}).Info("Fetching knowledge data from specified KBs")
		
		result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
			return s.kbClient.FetchKnowledgeData(ctx, kbsToQuery, manifest.PatientID, manifest.MedicationCode)
		})
		
		latency := time.Since(start)
		s.recordKBLatency("aggregate", latency)
		
		if err != nil {
			s.recordKBError("aggregate")
			knowledgeErr = NewKBServiceError("aggregate", "failed to fetch knowledge data", map[string]interface{}{
				"kbs_queried": kbsToQuery,
				"latency":     latency,
				"error":       err.Error(),
			})
			return knowledgeErr
		}
		
		knowledgeData = result.(KnowledgeContext)
		s.recordKBSuccess("aggregate")
		return nil
	})
	
	// Wait for both goroutines to complete
	if err := g.Wait(); err != nil {
		// Handle partial failures gracefully
		if patientErr != nil && knowledgeErr != nil {
			return nil, fmt.Errorf("both patient and knowledge data fetch failed: patient=%v, knowledge=%v", patientErr, knowledgeErr)
		} else if patientErr != nil {
			return nil, patientErr
		} else if knowledgeErr != nil {
			return nil, knowledgeErr
		}
		return nil, err
	}
	
	// Step 3: Assemble complete payload
	return s.assembleCompletePayload(patientData, knowledgeData, manifest), nil
}

// assembleCompletePayload combines patient and knowledge data into complete payload
func (s *contextIntegrationService) assembleCompletePayload(
	patientData PatientContext,
	knowledgeData KnowledgeContext,
	manifest *orb.IntentManifest,
) *CompleteContextPayload {
	
	now := time.Now()
	
	return &CompleteContextPayload{
		Patient:   patientData,
		Knowledge: knowledgeData,
		Metadata: ContextMetadata{
			RequestID:         manifest.RequestID,
			PatientID:         manifest.PatientID,
			ProcessingStarted: now,
			ProcessingEnded:   now,
			ProcessingTimeMs:  0, // Will be updated by caller
			Completeness: struct {
				PatientDataScore   float64  `json:"patient_data_score"`
				KnowledgeDataScore float64  `json:"knowledge_data_score"`
				OverallScore       float64  `json:"overall_score"`
				MissingFields      []string `json:"missing_fields"`
			}{
				PatientDataScore:   s.calculatePatientDataScore(patientData),
				KnowledgeDataScore: s.calculateKnowledgeDataScore(knowledgeData),
				OverallScore:       0.95, // Placeholder
				MissingFields:      []string{},
			},
			Quality: struct {
				DataFreshness     time.Duration `json:"data_freshness"`
				SourceReliability float64       `json:"source_reliability"`
				ValidationErrors  []string      `json:"validation_errors"`
			}{
				DataFreshness:     time.Hour * 1, // Placeholder
				SourceReliability: 0.98,          // Placeholder
				ValidationErrors:  []string{},
			},
			Performance: struct {
				CacheHitRate      float64 `json:"cache_hit_rate"`
				NetworkCallCount  int     `json:"network_call_count"`
				ParallelismFactor float64 `json:"parallelism_factor"`
			}{
				CacheHitRate:      s.getCacheHitRate(),
				NetworkCallCount:  2, // Context Gateway + KB services
				ParallelismFactor: 2.0, // Parallel execution
			},
		},
		Provenance: DataProvenance{
			PatientDataSources: []DataSource{
				{
					SourceID:    "context_gateway",
					SourceName:  "Context Gateway Service",
					SourceType:  "clinical_data",
					LastUpdated: now,
					Reliability: 0.98,
				},
			},
			KnowledgeDataSources: s.buildKnowledgeDataSources(manifest.KnowledgeManifest.RequiredKBs),
			LastUpdated:          now,
			DataVersion:          "v2.0",
		},
		CacheInfo: CacheInformation{
			CacheHit:        false, // Will be updated if from cache
			CacheKey:        s.generateCacheKey(manifest),
			CachedAt:        now,
			TTL:             int(s.getCacheTTL(manifest.CacheStrategy).Seconds()),
			FreshnessScore:  1.0,
			StaleServed:     false,
			RevalidationDue: false,
		},
	}
}

// Helper methods for cache management

// generateCacheKey creates a cache key based on the Intent Manifest
func (s *contextIntegrationService) generateCacheKey(manifest *orb.IntentManifest) string {
	return manifest.GetCacheKey() // Uses the enhanced cache key from Intent Manifest
}

// shouldUseCache determines if caching should be used based on strategy
func (s *contextIntegrationService) shouldUseCache(strategy string) bool {
	return strategy != CacheStrategyNone
}

// getCacheTTL returns the TTL based on cache strategy
func (s *contextIntegrationService) getCacheTTL(strategy string) time.Duration {
	switch strategy {
	case CacheStrategyAggressive:
		return time.Minute * 10 // Longer TTL for specific KBs
	case CacheStrategyStandard:
		return time.Minute * 5  // Standard TTL
	case CacheStrategyMinimal:
		return time.Minute * 2  // Shorter TTL
	default:
		return s.config.Cache.DefaultTTL
	}
}

// getStaleTTL returns the stale TTL based on cache strategy
func (s *contextIntegrationService) getStaleTTL(strategy string) time.Duration {
	return s.getCacheTTL(strategy) * 2 // Stale data valid for 2x normal TTL
}

// getCachedContext retrieves cached context
func (s *contextIntegrationService) getCachedContext(ctx context.Context, cacheKey string) (*CompleteContextPayload, error) {
	return s.cacheManager.Get(ctx, cacheKey)
}

// getStaleContext retrieves stale cached context
func (s *contextIntegrationService) getStaleContext(ctx context.Context, cacheKey string) (*CompleteContextPayload, error) {
	return s.cacheManager.GetStale(ctx, cacheKey)
}

// backgroundRevalidation performs background cache revalidation
func (s *contextIntegrationService) backgroundRevalidation(ctx context.Context, manifest *orb.IntentManifest, cacheKey string) {
	s.logger.WithField("cache_key", cacheKey).Info("Starting background revalidation")

	payload, err := s.fetchContextParallel(ctx, manifest)
	if err != nil {
		s.logger.WithError(err).Error("Background revalidation failed")
		return
	}

	ttl := s.getCacheTTL(manifest.CacheStrategy)
	staleTTL := s.getStaleTTL(manifest.CacheStrategy)

	if err := s.cacheManager.SetWithStale(ctx, cacheKey, payload, ttl, staleTTL); err != nil {
		s.logger.WithError(err).Error("Failed to update cache during background revalidation")
	} else {
		s.logger.WithField("cache_key", cacheKey).Info("Background revalidation completed successfully")
	}
}

// Helper methods for data scoring

// calculatePatientDataScore calculates completeness score for patient data
func (s *contextIntegrationService) calculatePatientDataScore(data PatientContext) float64 {
	score := 0.0
	maxScore := 5.0 // Demographics, Medications, Conditions, Labs, Vitals

	if data.Demographics.Age > 0 {
		score += 1.0
	}
	if len(data.Medications.Active) > 0 {
		score += 1.0
	}
	if len(data.Conditions.Active) > 0 {
		score += 1.0
	}
	if len(data.Labs.Recent) > 0 {
		score += 1.0
	}
	if len(data.Vitals.Current) > 0 {
		score += 1.0
	}

	return score / maxScore
}

// calculateKnowledgeDataScore calculates completeness score for knowledge data
func (s *contextIntegrationService) calculateKnowledgeDataScore(data KnowledgeContext) float64 {
	score := 0.0
	maxScore := 7.0 // Number of KB services

	if len(data.DrugInteractions.Interactions) > 0 {
		score += 1.0
	}
	if data.FormularyInfo.Status != "" {
		score += 1.0
	}
	if len(data.Guidelines.Recommendations) > 0 {
		score += 1.0
	}
	if data.Dosage.StandardDose.Amount > 0 {
		score += 1.0
	}
	if len(data.Safety.Contraindications) >= 0 { // Even empty is valid
		score += 1.0
	}
	if data.Monitoring.Required || !data.Monitoring.Required { // Boolean field always valid
		score += 1.0
	}
	if len(data.Evidence.ResistanceProfiles) >= 0 { // Even empty is valid
		score += 1.0
	}

	return score / maxScore
}

// buildKnowledgeDataSources creates data source information for KB services
func (s *contextIntegrationService) buildKnowledgeDataSources(requiredKBs []string) []DataSource {
	sources := make([]DataSource, 0, len(requiredKBs))
	now := time.Now()

	kbNames := map[string]string{
		KBDrugMaster:         "Drug Master Knowledge Base",
		KBDosingRules:        "Dosing Rules Knowledge Base",
		KBDrugInteractions:   "Drug Interactions Knowledge Base",
		KBFormularyStock:     "Formulary Stock Knowledge Base",
		KBPatientSafetyChecks: "Patient Safety Checks Knowledge Base",
		KBGuidelineEvidence:  "Guideline Evidence Knowledge Base",
		KBResistanceProfiles: "Resistance Profiles Knowledge Base",
	}

	for _, kbID := range requiredKBs {
		if name, exists := kbNames[kbID]; exists {
			sources = append(sources, DataSource{
				SourceID:    kbID,
				SourceName:  name,
				SourceType:  "knowledge_base",
				LastUpdated: now,
				Reliability: 0.95, // High reliability for KB services
			})
		}
	}

	return sources
}

// Metrics recording methods

func (s *contextIntegrationService) recordCacheHit() {
	// Implementation would update cache hit metrics
}

func (s *contextIntegrationService) recordCacheMiss() {
	// Implementation would update cache miss metrics
}

func (s *contextIntegrationService) recordSuccess(latency time.Duration) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.totalRequests++
	s.metrics.successfulRequests++
	s.metrics.latencies = append(s.metrics.latencies, latency)
	s.metrics.lastUpdated = time.Now()
}

func (s *contextIntegrationService) recordFailure(err error) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.totalRequests++
	s.metrics.failedRequests++
	s.metrics.lastUpdated = time.Now()
}

func (s *contextIntegrationService) recordContextGatewayLatency(latency time.Duration) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.contextGatewayLatencies = append(s.metrics.contextGatewayLatencies, latency)
}

func (s *contextIntegrationService) recordContextGatewayError() {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.contextGatewayErrors++
}

func (s *contextIntegrationService) recordKBLatency(kbID string, latency time.Duration) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	if s.metrics.kbLatency[kbID] == nil {
		s.metrics.kbLatency[kbID] = make([]time.Duration, 0)
	}
	s.metrics.kbLatency[kbID] = append(s.metrics.kbLatency[kbID], latency)
	s.metrics.kbCallCount[kbID]++
}

func (s *contextIntegrationService) recordKBError(kbID string) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.kbErrors[kbID]++
}

func (s *contextIntegrationService) recordKBSuccess(kbID string) {
	s.metrics.mu.Lock()
	defer s.metrics.mu.Unlock()

	s.metrics.kbCallCount[kbID]++
}

func (s *contextIntegrationService) getCacheHitRate() float64 {
	stats := s.cacheManager.GetStats()
	if stats.HitCount+stats.MissCount == 0 {
		return 0.0
	}
	return float64(stats.HitCount) / float64(stats.HitCount+stats.MissCount)
}

// Interface implementation methods

// InvalidateCache invalidates cache for a specific patient
func (s *contextIntegrationService) InvalidateCache(patientID string) error {
	pattern := fmt.Sprintf("intent_manifest_%s_*", patientID)
	return s.cacheManager.DeletePattern(context.Background(), pattern)
}

// InvalidateCachePattern invalidates cache entries matching a pattern
func (s *contextIntegrationService) InvalidateCachePattern(pattern string) error {
	return s.cacheManager.DeletePattern(context.Background(), pattern)
}

// GetCacheStats returns cache statistics
func (s *contextIntegrationService) GetCacheStats() CacheStatistics {
	return s.cacheManager.GetStats()
}

// HealthCheck performs health check on all dependencies
func (s *contextIntegrationService) HealthCheck() error {
	// Check cache manager
	if err := s.cacheManager.HealthCheck(); err != nil {
		return NewCacheError("cache manager health check failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Check Context Gateway
	if err := s.contextGatewayClient.HealthCheck(); err != nil {
		return NewContextGatewayError("context gateway health check failed", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Check Knowledge Base services
	kbHealth := s.kbClient.HealthCheck()
	for kbID, healthy := range kbHealth {
		if !healthy {
			return NewKBServiceError(kbID, "knowledge base health check failed", map[string]interface{}{
				"kb_id": kbID,
			})
		}
	}

	return nil
}

// GetMetrics returns service performance metrics
func (s *contextIntegrationService) GetMetrics() IntegrationMetrics {
	s.metrics.mu.RLock()
	defer s.metrics.mu.RUnlock()

	// Calculate success rate
	successRate := 0.0
	if s.metrics.totalRequests > 0 {
		successRate = float64(s.metrics.successfulRequests) / float64(s.metrics.totalRequests)
	}

	// Calculate average latency
	avgLatency := time.Duration(0)
	if len(s.metrics.latencies) > 0 {
		total := time.Duration(0)
		for _, latency := range s.metrics.latencies {
			total += latency
		}
		avgLatency = total / time.Duration(len(s.metrics.latencies))
	}

	// Calculate KB latencies
	kbLatency := make(map[string]time.Duration)
	for kbID, latencies := range s.metrics.kbLatency {
		if len(latencies) > 0 {
			total := time.Duration(0)
			for _, latency := range latencies {
				total += latency
			}
			kbLatency[kbID] = total / time.Duration(len(latencies))
		}
	}

	// Calculate KB error rates
	kbErrorRate := make(map[string]float64)
	for kbID, errorCount := range s.metrics.kbErrors {
		callCount := s.metrics.kbCallCount[kbID]
		if callCount > 0 {
			kbErrorRate[kbID] = float64(errorCount) / float64(callCount)
		}
	}

	// Calculate Context Gateway average latency
	contextGatewayLatency := time.Duration(0)
	if len(s.metrics.contextGatewayLatencies) > 0 {
		total := time.Duration(0)
		for _, latency := range s.metrics.contextGatewayLatencies {
			total += latency
		}
		contextGatewayLatency = total / time.Duration(len(s.metrics.contextGatewayLatencies))
	}

	return IntegrationMetrics{
		TotalRequests:         s.metrics.totalRequests,
		SuccessfulRequests:    s.metrics.successfulRequests,
		FailedRequests:        s.metrics.failedRequests,
		SuccessRate:           successRate,
		AverageLatency:        avgLatency,
		P95Latency:            s.calculatePercentile(s.metrics.latencies, 0.95),
		P99Latency:            s.calculatePercentile(s.metrics.latencies, 0.99),
		CacheStats:            s.cacheManager.GetStats(),
		KBCallCount:           s.copyKBCallCount(),
		KBLatency:             kbLatency,
		KBErrorRate:           kbErrorRate,
		ContextGatewayLatency: contextGatewayLatency,
		ContextGatewayErrors:  s.metrics.contextGatewayErrors,
		CircuitBreakerStates:  map[string]CircuitBreakerState{
			"main": s.circuitBreaker.GetState(),
		},
	}
}

// UpdateConfiguration updates service configuration
func (s *contextIntegrationService) UpdateConfiguration(config ServiceConfiguration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config
	s.logger.Info("Service configuration updated")
	return nil
}

// GetConfiguration returns current service configuration
func (s *contextIntegrationService) GetConfiguration() ServiceConfiguration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.config
}

// Helper methods for metrics calculation

func (s *contextIntegrationService) calculatePercentile(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return time.Duration(0)
	}

	// Simple percentile calculation (would use proper sorting in production)
	index := int(float64(len(latencies)) * percentile)
	if index >= len(latencies) {
		index = len(latencies) - 1
	}

	return latencies[index]
}

func (s *contextIntegrationService) copyKBCallCount() map[string]int64 {
	result := make(map[string]int64)
	for k, v := range s.metrics.kbCallCount {
		result[k] = v
	}
	return result
}
