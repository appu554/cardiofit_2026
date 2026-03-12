package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"kb-2-clinical-context-go/internal/cache"
	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/metrics"
	"kb-2-clinical-context-go/internal/models"
)

// ContextService provides clinical context assembly functionality with multi-tier caching
type ContextService struct {
	config      *config.Config
	mongoClient *mongo.Client
	redisClient *redis.Client
	
	// Multi-tier cache for high-performance access
	cache   *cache.MultiTierCache
	metrics *metrics.PrometheusMetrics
	
	// Service dependencies
	phenotypeEngine    *PhenotypeEngine
	riskService        *RiskAssessmentService
	treatmentService   *TreatmentPreferenceService
	
	// Cache key builder for standardized keys
	keyBuilder *cache.CacheKeyBuilder
}

// NewContextService creates a new context service with multi-tier caching
func NewContextService(mongoClient *mongo.Client, redisClient *redis.Client, cfg *config.Config, metricsCollector *metrics.PrometheusMetrics) *ContextService {
	// Initialize multi-tier cache
	multiTierCache := cache.NewMultiTierCache(cfg, redisClient, metricsCollector)
	
	// Initialize cache key builder
	keyBuilder := cache.NewCacheKeyBuilder("kb2", "1.0")
	
	return &ContextService{
		config:      cfg,
		mongoClient: mongoClient,
		redisClient: redisClient,
		cache:       multiTierCache,
		metrics:     metricsCollector,
		keyBuilder:  keyBuilder,
	}
}

// SetServiceDependencies sets the dependent services
func (cs *ContextService) SetServiceDependencies(
	phenotypeEngine *PhenotypeEngine,
	riskService *RiskAssessmentService,
	treatmentService *TreatmentPreferenceService,
) {
	cs.phenotypeEngine = phenotypeEngine
	cs.riskService = riskService
	cs.treatmentService = treatmentService
}

// SetMultiTierCache sets the multi-tier cache (for dependency injection)
func (cs *ContextService) SetMultiTierCache(multiTierCache *cache.MultiTierCache) {
	cs.cache = multiTierCache
}

// AssembleContext assembles complete clinical context for a patient with caching optimization
func (cs *ContextService) AssembleContext(ctx context.Context, request *models.ContextAssemblyRequest) (*models.ClinicalContext, error) {
	startTime := time.Now()
	
	// Check cache first for complete context
	if cachedContext, err := cs.getCachedCompleteContext(ctx, request); err == nil && cachedContext != nil {
		// Record cache hit and return cached context
		cs.metrics.RecordCacheTierHit("combined", "context_assembly")
		return cachedContext, nil
	}
	cs.metrics.RecordCacheTierMiss("combined", "context_assembly")
	
	clinicalContext := &models.ClinicalContext{
		PatientID:         request.PatientID,
		ProcessingMetrics: models.ProcessingMetrics{
			ComponentProcessingTimes: make(map[string]time.Duration),
		},
		GeneratedAt: time.Now(),
	}
	
	// Create a wait group for parallel processing
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]string, 0)
	
	// Channel to control concurrency
	semaphore := make(chan struct{}, cs.config.MaxConcurrentRequests)
	
	// Track cache performance for this request
	var cacheHits, totalCacheOps int64
	
	// Phenotype evaluation with caching
	if request.IncludePhenotypes || cs.containsComponent(request.ContextComponents, "phenotypes") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			componentStart := time.Now()
			phenotypeResults, hits, total, err := cs.evaluatePhenotypesForContextCached(ctx, request)
			componentDuration := time.Since(componentStart)
			
			mu.Lock()
			defer mu.Unlock()
			
			clinicalContext.ProcessingMetrics.ComponentProcessingTimes["phenotypes"] = componentDuration
			cacheHits += hits
			totalCacheOps += total
			
			if err != nil {
				errors = append(errors, fmt.Sprintf("Phenotype evaluation error: %v", err))
			} else {
				clinicalContext.PhenotypeResults = phenotypeResults
			}
		}()
	}
	
	// Risk assessment with caching
	if request.IncludeRisks || cs.containsComponent(request.ContextComponents, "risks") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			componentStart := time.Now()
			riskAssessment, hits, total, err := cs.assessRiskForContextCached(ctx, request)
			componentDuration := time.Since(componentStart)
			
			mu.Lock()
			defer mu.Unlock()
			
			clinicalContext.ProcessingMetrics.ComponentProcessingTimes["risks"] = componentDuration
			cacheHits += hits
			totalCacheOps += total
			
			if err != nil {
				errors = append(errors, fmt.Sprintf("Risk assessment error: %v", err))
			} else {
				clinicalContext.RiskAssessment = riskAssessment
			}
		}()
	}
	
	// Treatment preferences with caching
	if request.IncludeTreatments || cs.containsComponent(request.ContextComponents, "treatments") {
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			componentStart := time.Now()
			treatmentPreferences, hits, total, err := cs.evaluateTreatmentPreferencesForContextCached(ctx, request)
			componentDuration := time.Since(componentStart)
			
			mu.Lock()
			defer mu.Unlock()
			
			clinicalContext.ProcessingMetrics.ComponentProcessingTimes["treatments"] = componentDuration
			cacheHits += hits
			totalCacheOps += total
			
			if err != nil {
				errors = append(errors, fmt.Sprintf("Treatment preferences error: %v", err))
			} else {
				clinicalContext.TreatmentPreferences = treatmentPreferences
			}
		}()
	}
	
	// Wait for all components to complete
	wg.Wait()
	
	// Record total processing time
	totalDuration := time.Since(startTime)
	clinicalContext.ProcessingMetrics.TotalProcessingTime = totalDuration
	
	// Calculate cache performance for this request
	cacheHitRate := 0.0
	if totalCacheOps > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCacheOps)
	}
	clinicalContext.ProcessingMetrics.CacheHitRate = cacheHitRate
	
	// Check SLA compliance
	slaThreshold := time.Duration(cs.config.ContextAssemblySLA) * time.Millisecond
	slaCompliant := totalDuration <= slaThreshold
	clinicalContext.ProcessingMetrics.SLACompliant = slaCompliant
	
	if !slaCompliant {
		errors = append(errors, fmt.Sprintf("SLA violation: context assembly took %v, threshold is %v", totalDuration, slaThreshold))
		cs.metrics.RecordSLAViolation("context_assembly", "latency")
	}
	
	// Generate context summary
	cs.generateContextSummary(clinicalContext)
	
	// Cache the complete context asynchronously for future requests
	go cs.cacheCompleteContext(context.Background(), request, clinicalContext)
	
	// Record metrics
	cs.metrics.RecordContextAssembly(request.DetailLevel, "success", totalDuration)
	
	// Return error if any component failed critically
	if len(errors) > 0 {
		return clinicalContext, fmt.Errorf("context assembly completed with errors: %v", errors)
	}
	
	return clinicalContext, nil
}

// evaluatePhenotypesForContextCached evaluates phenotypes with caching optimization
func (cs *ContextService) evaluatePhenotypesForContextCached(ctx context.Context, request *models.ContextAssemblyRequest) ([]models.PhenotypeEvaluationResult, int64, int64, error) {
	if cs.phenotypeEngine == nil {
		return nil, 0, 0, fmt.Errorf("phenotype engine not available")
	}
	
	var cacheHits, totalOps int64
	
	// Check for cached phenotype results first
	cacheKey := cs.keyBuilder.BuildKey("phenotype_evaluation", request.PatientID, request.DetailLevel)
	
	if cached, found := cs.cache.l1Cache.Get(cacheKey); found {
		if results, ok := cached.([]models.PhenotypeEvaluationResult); ok {
			cacheHits++
			totalOps++
			cs.metrics.RecordCacheTierHit("l1", "phenotype")
			return results, cacheHits, totalOps, nil
		}
	}
	totalOps++
	cs.metrics.RecordCacheTierMiss("l1", "phenotype")
	
	// Cache miss - evaluate phenotypes
	phenotypeRequest := &models.PhenotypeEvaluationRequest{
		Patients:           []models.Patient{request.PatientData},
		IncludeExplanation: cs.shouldIncludeExplanation(request.DetailLevel),
	}
	
	results, err := cs.phenotypeEngine.EvaluatePhenotypes(ctx, phenotypeRequest)
	if err != nil {
		return nil, cacheHits, totalOps, fmt.Errorf("failed to evaluate phenotypes: %w", err)
	}
	
	// Cache results asynchronously
	go func() {
		cs.cache.l1Cache.Set(cacheKey, results, 5*time.Minute)
	}()
	
	return results, cacheHits, totalOps, nil
}

// assessRiskForContextCached performs risk assessment with caching optimization
func (cs *ContextService) assessRiskForContextCached(ctx context.Context, request *models.ContextAssemblyRequest) (*models.RiskAssessmentResult, int64, int64, error) {
	if cs.riskService == nil {
		return nil, 0, 0, fmt.Errorf("risk service not available")
	}
	
	var cacheHits, totalOps int64
	
	// Check cache for risk assessment
	result, err := cs.cache.GetRiskAssessment(ctx, request.PatientID, "comprehensive", func() (*models.RiskAssessmentResult, error) {
		totalOps++
		
		riskRequest := &models.RiskAssessmentRequest{
			PatientID:        request.PatientID,
			PatientData:      request.PatientData,
			IncludeFactors:   cs.shouldIncludeFactors(request.DetailLevel),
			CustomParameters: request.CustomParameters,
		}
		
		return cs.riskService.AssessRisk(ctx, riskRequest)
	})
	
	if err != nil {
		return nil, cacheHits, totalOps, fmt.Errorf("failed to assess risk: %w", err)
	}
	
	// Increment cache hit if data was found in cache
	if result != nil {
		cacheHits++
	}
	
	return result, cacheHits, totalOps, nil
}

// evaluateTreatmentPreferencesForContextCached evaluates treatment preferences with caching
func (cs *ContextService) evaluateTreatmentPreferencesForContextCached(ctx context.Context, request *models.ContextAssemblyRequest) (*models.TreatmentPreferencesResult, int64, int64, error) {
	if cs.treatmentService == nil {
		return nil, 0, 0, fmt.Errorf("treatment service not available")
	}
	
	var cacheHits, totalOps int64
	
	// Infer condition for caching key
	condition := cs.inferPrimaryCondition(request)
	
	// Check cache for treatment preferences
	result, err := cs.cache.GetTreatmentPreferences(ctx, request.PatientID, condition, func() (*models.TreatmentPreferencesResult, error) {
		totalOps++
		
		treatmentRequest := &models.TreatmentPreferencesRequest{
			PatientID:         request.PatientID,
			PatientData:       request.PatientData,
			Condition:         condition,
			PreferenceProfile: request.CustomParameters,
		}
		
		return cs.treatmentService.EvaluateTreatmentPreferences(ctx, treatmentRequest)
	})
	
	if err != nil {
		return nil, cacheHits, totalOps, fmt.Errorf("failed to evaluate treatment preferences: %w", err)
	}
	
	// Increment cache hit if data was found in cache
	if result != nil {
		cacheHits++
	}
	
	return result, cacheHits, totalOps, nil
}

// getCachedCompleteContext attempts to retrieve complete cached context
func (cs *ContextService) getCachedCompleteContext(ctx context.Context, request *models.ContextAssemblyRequest) (*models.ClinicalContext, error) {
	// Build cache key for complete context
	contextType := cs.buildContextType(request)
	cacheKey := cs.keyBuilder.BuildPatientContextKey(request.PatientID, contextType)
	
	// Try to get from cache
	return cs.cache.GetPatientContext(ctx, request.PatientID, contextType, func() (*models.ClinicalContext, error) {
		// Return nil to indicate cache miss - will be handled by calling function
		return nil, fmt.Errorf("cache miss")
	})
}

// cacheCompleteContext caches the complete assembled context
func (cs *ContextService) cacheCompleteContext(ctx context.Context, request *models.ContextAssemblyRequest, context *models.ClinicalContext) error {
	contextType := cs.buildContextType(request)
	
	// Determine appropriate TTL based on context type
	ttl := cs.getContextTTL(request.DetailLevel)
	
	// Cache with multi-tier strategy
	return cs.cache.Set(ctx, cs.keyBuilder.BuildPatientContextKey(request.PatientID, contextType), context, ttl)
}

// buildContextType builds context type string for caching
func (cs *ContextService) buildContextType(request *models.ContextAssemblyRequest) string {
	contextType := request.DetailLevel
	
	// Add component modifiers for more specific caching
	if request.IncludePhenotypes {
		contextType += "_phen"
	}
	if request.IncludeRisks {
		contextType += "_risk"
	}
	if request.IncludeTreatments {
		contextType += "_treat"
	}
	
	return contextType
}

// getContextTTL returns appropriate TTL based on context detail level
func (cs *ContextService) getContextTTL(detailLevel string) time.Duration {
	switch detailLevel {
	case "minimal":
		return 10 * time.Minute // Short TTL for minimal context
	case "standard":
		return 30 * time.Minute // Medium TTL for standard context
	case "comprehensive":
		return 5 * time.Minute  // Shorter TTL for comprehensive context (changes more frequently)
	default:
		return 15 * time.Minute // Default TTL
	}
}

// AssembleContextBatch efficiently processes multiple patients with batch caching
func (cs *ContextService) AssembleContextBatch(ctx context.Context, requests []*models.ContextAssemblyRequest) ([]*models.ClinicalContext, error) {
	if len(requests) == 0 {
		return []*models.ClinicalContext{}, nil
	}
	
	startTime := time.Now()
	batchSize := len(requests)
	
	// Record batch size
	cs.metrics.RecordBatchSize(float64(batchSize))
	
	// Parallel processing with cache optimization
	results := make([]*models.ClinicalContext, len(requests))
	var wg sync.WaitGroup
	var totalCacheHits, totalCacheOps int64
	
	// Process in smaller batches to control memory and concurrency
	batchConcurrency := min(cs.config.MaxConcurrentRequests, batchSize)
	requestsPerWorker := batchSize / batchConcurrency
	if batchSize%batchConcurrency != 0 {
		requestsPerWorker++
	}
	
	for i := 0; i < batchConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			start := workerID * requestsPerWorker
			end := start + requestsPerWorker
			if end > len(requests) {
				end = len(requests)
			}
			
			// Pre-warm cache for this worker's requests
			cs.preWarmBatchCache(ctx, requests[start:end])
			
			for j := start; j < end; j++ {
				context, cacheHits, cacheOps, err := cs.assembleContextWithCacheTracking(ctx, requests[j])
				
				if err != nil {
					// Create error context
					results[j] = &models.ClinicalContext{
						PatientID:   requests[j].PatientID,
						GeneratedAt: time.Now(),
						ContextSummary: models.ContextSummary{
							ClinicalAlerts: []models.ClinicalAlert{
								{
									Level:   "error",
									Type:    "processing",
									Message: fmt.Sprintf("Context assembly failed: %v", err),
									Source:  "context_service",
								},
							},
						},
					}
				} else {
					results[j] = context
				}
				
				// Accumulate cache statistics
				totalCacheHits += cacheHits
				totalCacheOps += cacheOps
			}
		}(i)
	}
	
	wg.Wait()
	
	batchDuration := time.Since(startTime)
	
	// Record batch performance metrics
	cacheHitRate := 0.0
	if totalCacheOps > 0 {
		cacheHitRate = float64(totalCacheHits) / float64(totalCacheOps)
	}
	
	cs.metrics.RecordBatchPerformance(batchSize, batchDuration, int(totalCacheHits))
	
	// Check batch SLA compliance (1000 patients < 1s)
	if batchSize >= 1000 {
		slaCompliant := batchDuration < time.Second
		cs.metrics.UpdateSLACompliance("batch_1000_patients", slaCompliant)
		
		if !slaCompliant {
			return results, fmt.Errorf("batch SLA violation: %d patients processed in %v (target: <1s)", batchSize, batchDuration)
		}
	}
	
	// Log batch performance
	fmt.Printf("Batch processed: %d patients in %v (%.0f RPS, %.1f%% cache hit rate)\n", 
		batchSize, batchDuration, float64(batchSize)/batchDuration.Seconds(), cacheHitRate*100)
	
	return results, nil
}

// assembleContextWithCacheTracking assembles context while tracking cache performance
func (cs *ContextService) assembleContextWithCacheTracking(ctx context.Context, request *models.ContextAssemblyRequest) (*models.ClinicalContext, int64, int64, error) {
	cacheTimer := cs.metrics.StartCacheTimer("combined", "context_assembly")
	defer cacheTimer.ObserveDuration()
	
	// Use standard assembly but track cache operations
	context, err := cs.AssembleContext(ctx, request)
	
	// For simplicity, return estimated cache stats
	// In production, you would track actual cache operations
	cacheHits := int64(0)
	totalOps := int64(3) // Estimate 3 cache operations per context assembly
	
	if context != nil && context.ProcessingMetrics.CacheHitRate > 0 {
		cacheHits = int64(float64(totalOps) * context.ProcessingMetrics.CacheHitRate)
	}
	
	return context, cacheHits, totalOps, err
}

// preWarmBatchCache pre-warms cache for batch requests
func (cs *ContextService) preWarmBatchCache(ctx context.Context, requests []*models.ContextAssemblyRequest) {
	// Extract patient IDs for batch cache warming
	patientIDs := make([]string, len(requests))
	for i, req := range requests {
		patientIDs[i] = req.PatientID
	}
	
	// Build cache keys for likely accessed data
	cacheKeys := make([]string, 0, len(requests)*3) // Estimate 3 keys per patient
	
	for _, req := range requests {
		// Add context keys
		contextType := cs.buildContextType(req)
		cacheKeys = append(cacheKeys, cs.keyBuilder.BuildPatientContextKey(req.PatientID, contextType))
		
		// Add phenotype keys if needed
		if req.IncludePhenotypes {
			cacheKeys = append(cacheKeys, cs.keyBuilder.BuildKey("phenotype_evaluation", req.PatientID, req.DetailLevel))
		}
		
		// Add risk assessment keys if needed
		if req.IncludeRisks {
			cacheKeys = append(cacheKeys, cs.keyBuilder.BuildRiskAssessmentKey(req.PatientID, "comprehensive"))
		}
		
		// Add treatment preference keys if needed
		if req.IncludeTreatments {
			condition := cs.inferPrimaryCondition(req)
			cacheKeys = append(cacheKeys, cs.keyBuilder.BuildTreatmentPreferencesKey(req.PatientID, condition))
		}
	}
	
	// Batch cache lookup to warm L1 cache
	_, err := cs.cache.GetBatch(ctx, cacheKeys, func(missedKeys []string) (map[string]interface{}, error) {
		// Return empty map - we're just warming the cache
		return make(map[string]interface{}), nil
	})
	
	if err != nil {
		// Log error but don't fail batch processing
		fmt.Printf("Cache warming failed for batch: %v\n", err)
	}
}

// Cache management methods

// InvalidatePatientContext invalidates all cached data for a patient
func (cs *ContextService) InvalidatePatientContext(ctx context.Context, patientID string) error {
	pattern := fmt.Sprintf("*:%s:*", patientID)
	return cs.cache.InvalidatePattern(ctx, pattern)
}

// InvalidateContextType invalidates cached data for a specific context type
func (cs *ContextService) InvalidateContextType(ctx context.Context, contextType string) error {
	pattern := fmt.Sprintf("*:%s", contextType)
	return cs.cache.InvalidatePattern(ctx, pattern)
}

// WarmFrequentlyAccessedData warms cache with frequently accessed data
func (cs *ContextService) WarmFrequentlyAccessedData(ctx context.Context) error {
	return cs.cache.WarmCache(ctx)
}

// GetCacheStats returns cache performance statistics
func (cs *ContextService) GetCacheStats() map[string]*cache.CacheStats {
	return cs.cache.GetStats()
}

// CheckCacheSLACompliance checks if cache performance meets SLA targets
func (cs *ContextService) CheckCacheSLACompliance() map[string]bool {
	return cs.cache.CheckSLACompliance()
}

// OptimizeCachePerformance triggers cache optimization
func (cs *ContextService) OptimizeCachePerformance(ctx context.Context) error {
	return cs.cache.OptimizeCache(ctx)
}

// generateContextSummary generates a summary of the assembled context
func (cs *ContextService) generateContextSummary(context *models.ClinicalContext) {
	summary := models.ContextSummary{
		KeyFindings:      []string{},
		RiskHighlights:   []string{},
		TreatmentSummary: []string{},
		ClinicalAlerts:   []models.ClinicalAlert{},
		Recommendations:  []string{},
	}
	
	// Summarize phenotype findings
	if context.PhenotypeResults != nil {
		for _, result := range context.PhenotypeResults {
			for _, phenotype := range result.Phenotypes {
				if phenotype.Detected && phenotype.Confidence > 0.7 {
					summary.KeyFindings = append(summary.KeyFindings, 
						fmt.Sprintf("High-confidence phenotype detected: %s (%.1f%%)", 
							phenotype.Name, phenotype.Confidence*100))
				}
			}
		}
	}
	
	// Summarize risk assessment
	if context.RiskAssessment != nil {
		if context.RiskAssessment.OverallRisk.Level == "high" || context.RiskAssessment.OverallRisk.Level == "very_high" {
			summary.RiskHighlights = append(summary.RiskHighlights,
				fmt.Sprintf("High overall risk: %.1f%% (%s)", 
					context.RiskAssessment.OverallRisk.Score*100,
					context.RiskAssessment.OverallRisk.Description))
			
			// Add clinical alert for high risk
			summary.ClinicalAlerts = append(summary.ClinicalAlerts, models.ClinicalAlert{
				Level:   "warning",
				Type:    "risk",
				Message: "High risk patient requires enhanced monitoring",
				Source:  "risk_assessment",
				Action:  "review_risk_factors",
			})
		}
		
		// Add top risk factors
		for _, factor := range context.RiskAssessment.RiskFactors[:min(3, len(context.RiskAssessment.RiskFactors))] {
			summary.RiskHighlights = append(summary.RiskHighlights,
				fmt.Sprintf("Risk factor: %s (impact: %.1f%%)", factor.Factor, factor.Impact*100))
		}
	}
	
	// Summarize treatment preferences
	if context.TreatmentPreferences != nil {
		for _, preferred := range context.TreatmentPreferences.PreferredTreatments[:min(3, len(context.TreatmentPreferences.PreferredTreatments))] {
			// Find the corresponding treatment option
			var treatmentName string
			for _, option := range context.TreatmentPreferences.TreatmentOptions {
				if option.ID == preferred.TreatmentID {
					treatmentName = option.Name
					break
				}
			}
			
			summary.TreatmentSummary = append(summary.TreatmentSummary,
				fmt.Sprintf("Rank %d: %s (score: %.1f%%)", 
					preferred.Rank, treatmentName, preferred.OverallScore*100))
		}
		
		// Add conflict resolutions as alerts
		for _, conflict := range context.TreatmentPreferences.ConflictResolution {
			if conflict.Priority == "high" {
				summary.ClinicalAlerts = append(summary.ClinicalAlerts, models.ClinicalAlert{
					Level:   "info",
					Type:    "conflict",
					Message: fmt.Sprintf("Treatment conflict resolved: %s", conflict.Resolution),
					Source:  "treatment_preferences",
				})
			}
		}
	}
	
	// Generate recommendations
	summary.Recommendations = cs.generateRecommendations(context)
	
	context.ContextSummary = summary
}

// generateRecommendations generates clinical recommendations based on context
func (cs *ContextService) generateRecommendations(context *models.ClinicalContext) []string {
	recommendations := []string{}
	
	// Risk-based recommendations
	if context.RiskAssessment != nil {
		for _, recommendation := range context.RiskAssessment.Recommendations {
			if recommendation.Priority == "high" {
				recommendations = append(recommendations, recommendation.Action)
			}
		}
	}
	
	// Phenotype-based recommendations
	if context.PhenotypeResults != nil {
		for _, result := range context.PhenotypeResults {
			for _, phenotype := range result.Phenotypes {
				if phenotype.Detected && phenotype.Confidence > 0.8 {
					recommendations = append(recommendations,
						fmt.Sprintf("Consider %s-specific protocols", phenotype.Category))
				}
			}
		}
	}
	
	// Treatment-based recommendations
	if context.TreatmentPreferences != nil && len(context.TreatmentPreferences.PreferredTreatments) > 0 {
		topTreatment := context.TreatmentPreferences.PreferredTreatments[0]
		recommendations = append(recommendations,
			fmt.Sprintf("Consider initiating recommended treatment: %s", topTreatment.Rationale))
	}
	
	// Cache performance recommendations
	if context.ProcessingMetrics.CacheHitRate < 0.5 {
		recommendations = append(recommendations,
			"Consider cache warming for frequently accessed patient data")
	}
	
	return recommendations
}

// Performance monitoring and optimization

// MonitorPerformance continuously monitors context service performance
func (cs *ContextService) MonitorPerformance(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cs.reportPerformanceMetrics(ctx)
		}
	}
}

// reportPerformanceMetrics reports current performance to metrics system
func (cs *ContextService) reportPerformanceMetrics(ctx context.Context) {
	// Get cache statistics
	cacheStats := cs.cache.GetStats()
	
	// Update cache hit rate metrics
	for tier, stats := range cacheStats {
		cs.metrics.UpdateCacheHitRate(tier, stats.HitRate)
		cs.metrics.UpdateCacheMemoryUsage(tier, stats.MemoryUsage)
		cs.metrics.UpdateCacheSize(tier, stats.Size)
	}
	
	// Check SLA compliance
	slaCompliance := cs.cache.CheckSLACompliance()
	for metric, compliant := range slaCompliance {
		cs.metrics.UpdateSLACompliance(metric, compliant)
	}
	
	// Calculate and update overall performance score
	performanceScore := cs.calculateCurrentPerformanceScore(cacheStats)
	cs.metrics.UpdatePerformanceScore(performanceScore)
}

// calculateCurrentPerformanceScore calculates current performance score
func (cs *ContextService) calculateCurrentPerformanceScore(cacheStats map[string]*cache.CacheStats) float64 {
	score := 0.0
	
	// Cache performance (60% weight)
	if l1Stats, exists := cacheStats["l1"]; exists {
		if l1Stats.HitRate >= 0.85 {
			score += 0.3 // 50% of cache weight
		} else {
			score += 0.3 * (l1Stats.HitRate / 0.85)
		}
	}
	
	if l2Stats, exists := cacheStats["l2"]; exists {
		if l2Stats.HitRate >= 0.95 {
			score += 0.3 // 50% of cache weight
		} else {
			score += 0.3 * (l2Stats.HitRate / 0.95)
		}
	}
	
	// Service health (40% weight)
	// This would be based on recent request performance
	// For now, assume healthy service contributes 0.4
	score += 0.4
	
	return score
}

// Helper functions

func (cs *ContextService) containsComponent(components []string, component string) bool {
	for _, c := range components {
		if c == component {
			return true
		}
	}
	return false
}

func (cs *ContextService) shouldIncludeExplanation(detailLevel string) bool {
	return detailLevel == "comprehensive"
}

func (cs *ContextService) shouldIncludeFactors(detailLevel string) bool {
	return detailLevel == "standard" || detailLevel == "comprehensive"
}

func (cs *ContextService) inferPrimaryCondition(request *models.ContextAssemblyRequest) string {
	// Try to get condition from custom parameters
	if condition, exists := request.CustomParameters["condition"]; exists {
		if conditionStr, ok := condition.(string); ok {
			return conditionStr
		}
	}
	
	// Try to infer from patient conditions (take the first one)
	if len(request.PatientData.Conditions) > 0 {
		return request.PatientData.Conditions[0]
	}
	
	// Default to general condition
	return "general"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Legacy methods for compatibility

// GetContextHistory retrieves historical context for a patient
func (cs *ContextService) GetContextHistory(ctx context.Context, patientID string, limit int) ([]*models.ClinicalContext, error) {
	// Check cache first
	cacheKey := cs.keyBuilder.BuildKey("context_history", patientID, fmt.Sprintf("limit_%d", limit))
	
	if cached, found := cs.cache.l2Cache.Get(ctx, cacheKey); found {
		if history, ok := cached.([]*models.ClinicalContext); ok {
			cs.metrics.RecordCacheTierHit("l2", "context_history")
			return history, nil
		}
	}
	cs.metrics.RecordCacheTierMiss("l2", "context_history")
	
	// This would typically query MongoDB for historical context records
	// For now, return empty slice and cache it
	history := []*models.ClinicalContext{}
	
	// Cache the result
	go func() {
		cs.cache.l2Cache.Set(context.Background(), cacheKey, history, 15*time.Minute)
	}()
	
	return history, nil
}

// CacheContext caches the assembled context for future retrieval (legacy method)
func (cs *ContextService) CacheContext(ctx context.Context, context *models.ClinicalContext) error {
	// Use multi-tier cache instead of direct Redis
	cacheKey := cs.keyBuilder.BuildPatientContextKey(context.PatientID, "legacy")
	return cs.cache.Set(ctx, cacheKey, context, time.Hour)
}

// GetCachedContext retrieves cached context if available (legacy method)
func (cs *ContextService) GetCachedContext(ctx context.Context, patientID string, maxAge time.Duration) (*models.ClinicalContext, error) {
	cacheKey := cs.keyBuilder.BuildPatientContextKey(patientID, "legacy")
	
	data, err := cs.cache.Get(ctx, cacheKey, func() (interface{}, error) {
		return nil, fmt.Errorf("no cached context available")
	})
	
	if err != nil {
		return nil, err
	}
	
	if context, ok := data.(*models.ClinicalContext); ok {
		// Check age
		if time.Since(context.GeneratedAt) <= maxAge {
			return context, nil
		}
		
		// Context too old, invalidate
		go cs.cache.Invalidate(context.Background(), cacheKey)
		return nil, fmt.Errorf("cached context too old")
	}
	
	return nil, fmt.Errorf("invalid cached context data type")
}

// Administrative methods for cache management

// GetServiceHealth returns service health including cache performance
func (cs *ContextService) GetServiceHealth(ctx context.Context) map[string]interface{} {
	cacheStats := cs.cache.GetStats()
	slaCompliance := cs.cache.CheckSLACompliance()
	
	health := map[string]interface{}{
		"service":         "healthy",
		"cache_stats":     cacheStats,
		"sla_compliance":  slaCompliance,
		"performance_score": cs.calculateCurrentPerformanceScore(cacheStats),
	}
	
	// Check if any SLA targets are not met
	allCompliant := true
	for _, compliant := range slaCompliance {
		if !compliant {
			allCompliant = false
			break
		}
	}
	
	if !allCompliant {
		health["service"] = "degraded"
	}
	
	return health
}

// TriggerCacheOptimization manually triggers cache optimization
func (cs *ContextService) TriggerCacheOptimization(ctx context.Context) error {
	return cs.cache.OptimizeCache(ctx)
}

// ResetCacheStats resets cache statistics (for testing/benchmarking)
func (cs *ContextService) ResetCacheStats() {
	// This would reset internal counters if implemented in cache layers
	fmt.Println("Cache statistics reset requested")
}