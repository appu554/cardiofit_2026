package services

import (
	"context"
	"fmt"
	"time"

	"medication-service-v2/internal/infrastructure/cache"
	"go.uber.org/zap"
)

// CacheService provides high-level caching operations for the medication service
type CacheService struct {
	cacheManager      *cache.MultiLevelCache
	performanceCache  *cache.PerformanceCache
	serviceCache      *cache.ServiceSpecificCaches
	monitor           *cache.CacheMonitor
	logger            *zap.Logger
	
	// Service-specific caches
	recipeCache       *cache.RecipeResolverCache
	clinicalCache     *cache.ClinicalEngineCache
	workflowCache     *cache.WorkflowStateCache
	fhirCache         *cache.GoogleFHIRCache
	apolloCache       *cache.ApolloFederationCache
}

// CacheServiceConfig contains configuration for the cache service
type CacheServiceConfig struct {
	RedisURL             string        `json:"redis_url"`
	EnableAnalytics      bool          `json:"enable_analytics"`
	EnablePerformanceOpt bool          `json:"enable_performance_opt"`
	L1CacheSize          int64         `json:"l1_cache_size"`
	L1TTL                time.Duration `json:"l1_ttl"`
	L2TTL                time.Duration `json:"l2_ttl"`
	HotCacheSize         int64         `json:"hot_cache_size"`
	PromotionThreshold   int64         `json:"promotion_threshold"`
	OptimizeForLatency   bool          `json:"optimize_for_latency"`
}

// NewCacheService creates a new cache service with all caching layers
func NewCacheService(config CacheServiceConfig, logger *zap.Logger) (*CacheService, error) {
	// Create multi-level cache
	cacheConfig := cache.CacheConfig{
		RedisURL:           config.RedisURL,
		L1MaxSize:          config.L1CacheSize,
		L1TTL:              config.L1TTL,
		L2TTL:              config.L2TTL,
		PromotionThreshold: config.PromotionThreshold,
		DemotionTimeout:    15 * time.Minute,
		EncryptionEnabled:  true,  // HIPAA compliance
		AuditEnabled:       true,  // HIPAA compliance
	}
	
	cacheManager, err := cache.NewMultiLevelCache(cacheConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-level cache: %w", err)
	}
	
	// Create performance cache if enabled
	var performanceCache *cache.PerformanceCache
	if config.EnablePerformanceOpt {
		// Get Redis client from cache manager for performance cache
		redisClient := cacheManager.GetRedisClient() // This method would need to be added to MultiLevelCache
		if redisClient == nil {
			return nil, fmt.Errorf("failed to get Redis client for performance cache")
		}
		
		performanceCache = cache.NewPerformanceCache(cacheManager, redisClient, logger)
		
		if config.OptimizeForLatency {
			performanceCache.OptimizeForLatency()
		} else {
			performanceCache.OptimizeForThroughput()
		}
	}
	
	// Create service-specific caches
	serviceCache := cache.NewServiceSpecificCaches(cacheManager, logger)
	
	// Create monitoring
	var monitor *cache.CacheMonitor
	if config.EnableAnalytics {
		redisClient := cacheManager.GetRedisClient()
		monitor = cache.NewCacheMonitor(cacheManager, performanceCache, redisClient, logger)
		monitor.EnableAnalytics()
	}
	
	cs := &CacheService{
		cacheManager:     cacheManager,
		performanceCache: performanceCache,
		serviceCache:     serviceCache,
		monitor:          monitor,
		logger:           logger.Named("cache_service"),
		
		// Initialize service-specific caches
		recipeCache:   serviceCache.RecipeResolver(),
		clinicalCache: serviceCache.ClinicalEngine(),
		workflowCache: serviceCache.WorkflowState(),
		fhirCache:     serviceCache.GoogleFHIR(),
		apolloCache:   serviceCache.ApolloFederation(),
	}
	
	// Setup cache warming rules
	cs.setupCacheWarmingRules()
	
	// Setup monitoring alerts
	cs.setupMonitoringAlerts()
	
	logger.Info("Cache service initialized successfully",
		zap.String("redis_url", config.RedisURL),
		zap.Bool("analytics_enabled", config.EnableAnalytics),
		zap.Bool("performance_opt_enabled", config.EnablePerformanceOpt),
		zap.Int64("l1_cache_size", config.L1CacheSize),
	)
	
	return cs, nil
}

// Recipe Resolver Cache Operations - Target <10ms response time

// GetRecipe retrieves a cached recipe with patient context
func (cs *CacheService) GetRecipe(ctx context.Context, protocolID string, patientContext map[string]interface{}) (*cache.RecipeCache, error) {
	start := time.Now()
	
	result, err := cs.recipeCache.GetRecipe(ctx, protocolID, patientContext)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("recipe_resolver", "get_recipe", latency, cache.L2Cache, err == nil, err)
	}
	
	if err == nil && latency > 10*time.Millisecond {
		cs.logger.Warn("Recipe cache latency exceeded target",
			zap.String("protocol_id", protocolID),
			zap.Duration("latency", latency),
			zap.Duration("target", 10*time.Millisecond),
		)
	}
	
	return result, err
}

// CacheRecipe stores a recipe with intelligent TTL
func (cs *CacheService) CacheRecipe(ctx context.Context, protocolID string, recipe map[string]interface{}, patientContext map[string]interface{}, dependencies []string) error {
	start := time.Now()
	
	err := cs.recipeCache.SetRecipe(ctx, protocolID, recipe, patientContext, dependencies)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("recipe_resolver", "cache_recipe", latency, cache.L2Cache, err == nil, err)
	}
	
	return err
}

// Clinical Engine Cache Operations

// GetClinicalCalculation retrieves cached calculation results
func (cs *CacheService) GetClinicalCalculation(ctx context.Context, calculationID string, inputParams map[string]interface{}) (*cache.ClinicalCalculationResult, error) {
	start := time.Now()
	
	result, err := cs.clinicalCache.GetCalculationResult(ctx, calculationID, inputParams)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("clinical_engine", "get_calculation", latency, cache.L2Cache, err == nil, err)
	}
	
	return result, err
}

// CacheClinicalCalculation stores calculation results with confidence-based TTL
func (cs *CacheService) CacheClinicalCalculation(ctx context.Context, result *cache.ClinicalCalculationResult) error {
	start := time.Now()
	
	err := cs.clinicalCache.SetCalculationResult(ctx, result)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("clinical_engine", "cache_calculation", latency, cache.L2Cache, err == nil, err)
	}
	
	return err
}

// Workflow State Cache Operations

// GetWorkflowState retrieves cached workflow state
func (cs *CacheService) GetWorkflowState(ctx context.Context, workflowID string) (*cache.WorkflowState, error) {
	start := time.Now()
	
	result, err := cs.workflowCache.GetWorkflowState(ctx, workflowID)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("workflow_orchestrator", "get_state", latency, cache.L2Cache, err == nil, err)
	}
	
	return result, err
}

// CacheWorkflowState stores workflow state with dynamic TTL
func (cs *CacheService) CacheWorkflowState(ctx context.Context, state *cache.WorkflowState) error {
	start := time.Now()
	
	err := cs.workflowCache.SetWorkflowState(ctx, state)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("workflow_orchestrator", "cache_state", latency, cache.L2Cache, err == nil, err)
	}
	
	return err
}

// Google FHIR Cache Operations

// GetFHIRResource retrieves cached FHIR resource
func (cs *CacheService) GetFHIRResource(ctx context.Context, projectID, datasetID, fhirStoreID, resourceType, resourceID string) (*cache.FHIRResourceCache, error) {
	start := time.Now()
	
	result, err := cs.fhirCache.GetFHIRResource(ctx, projectID, datasetID, fhirStoreID, resourceType, resourceID)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("google_fhir", "get_resource", latency, cache.L2Cache, err == nil, err)
	}
	
	return result, err
}

// CacheFHIRResource stores FHIR resource with metadata
func (cs *CacheService) CacheFHIRResource(ctx context.Context, resource *cache.FHIRResourceCache) error {
	start := time.Now()
	
	err := cs.fhirCache.SetFHIRResource(ctx, resource)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("google_fhir", "cache_resource", latency, cache.L2Cache, err == nil, err)
	}
	
	return err
}

// Apollo Federation Cache Operations

// GetQueryResult retrieves cached GraphQL query result
func (cs *CacheService) GetQueryResult(ctx context.Context, queryHash string, variables map[string]interface{}) (*cache.GraphQLQueryCache, error) {
	start := time.Now()
	
	result, err := cs.apolloCache.GetQueryResult(ctx, queryHash, variables)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("apollo_federation", "get_query", latency, cache.L2Cache, err == nil, err)
	}
	
	return result, err
}

// CacheQueryResult stores GraphQL query result
func (cs *CacheService) CacheQueryResult(ctx context.Context, result *cache.GraphQLQueryCache) error {
	start := time.Now()
	
	err := cs.apolloCache.SetQueryResult(ctx, result)
	latency := time.Since(start)
	
	if cs.monitor != nil {
		cs.monitor.RecordOperation("apollo_federation", "cache_query", latency, cache.L2Cache, err == nil, err)
	}
	
	return err
}

// High-Performance Operations

// FastGet provides ultra-fast cache retrieval using hot cache
func (cs *CacheService) FastGet(ctx context.Context, key string, dest interface{}) error {
	if cs.performanceCache != nil {
		return cs.performanceCache.FastGet(ctx, key, dest)
	}
	return cs.cacheManager.Get(ctx, key, dest)
}

// FastSet provides optimized cache storage
func (cs *CacheService) FastSet(ctx context.Context, key string, value interface{}, ttl time.Duration, tags ...string) error {
	if cs.performanceCache != nil {
		return cs.performanceCache.FastSet(ctx, key, value, ttl, tags...)
	}
	return cs.cacheManager.Set(ctx, key, value, ttl, tags...)
}

// BatchOperations provides efficient batch processing
func (cs *CacheService) BatchGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if cs.performanceCache != nil {
		return cs.performanceCache.BatchGet(ctx, keys)
	}
	
	// Fallback to individual gets
	results := make(map[string]interface{})
	for _, key := range keys {
		var value interface{}
		if err := cs.cacheManager.Get(ctx, key, &value); err == nil {
			results[key] = value
		}
	}
	return results, nil
}

func (cs *CacheService) BatchSet(ctx context.Context, items map[string]interface{}, ttl time.Duration, tags ...string) error {
	if cs.performanceCache != nil {
		return cs.performanceCache.BatchSet(ctx, items, ttl, tags...)
	}
	
	// Fallback to individual sets
	for key, value := range items {
		if err := cs.cacheManager.Set(ctx, key, value, ttl, tags...); err != nil {
			return err
		}
	}
	return nil
}

// Cache Management Operations

// InvalidateByTags removes all cached entries with matching tags
func (cs *CacheService) InvalidateByTags(ctx context.Context, tags ...string) error {
	return cs.cacheManager.InvalidateByTags(ctx, tags...)
}

// InvalidateService clears all cache entries for a specific service
func (cs *CacheService) InvalidateService(ctx context.Context, serviceName string) error {
	return cs.InvalidateByTags(ctx, serviceName)
}

// WarmupCache preloads frequently accessed data
func (cs *CacheService) WarmupCache(ctx context.Context, serviceName string, data map[string]interface{}) error {
	ttl := 1 * time.Hour // Default warmup TTL
	tags := []string{serviceName, "warmup"}
	
	return cs.BatchSet(ctx, data, ttl, tags...)
}

// Monitoring and Analytics

// GetCacheMetrics returns comprehensive cache performance metrics
func (cs *CacheService) GetCacheMetrics() *CacheMetrics {
	if cs.monitor == nil {
		return &CacheMetrics{}
	}
	
	metrics := cs.monitor.GetMetrics()
	cacheStats := cs.cacheManager.GetStats()
	
	return &CacheMetrics{
		OverallStats:     cacheStats,
		DetailedMetrics:  metrics,
		HealthStatus:     cs.monitor.GetHealthStatus(),
	}
}

// GetServiceReport returns performance report for a specific service
func (cs *CacheService) GetServiceReport(serviceName string) *cache.ServiceReport {
	if cs.monitor == nil {
		return nil
	}
	return cs.monitor.GetServiceReport(serviceName)
}

// CacheMetrics aggregates all cache performance data
type CacheMetrics struct {
	OverallStats    cache.CacheStats       `json:"overall_stats"`
	DetailedMetrics cache.CacheMetrics     `json:"detailed_metrics"`
	HealthStatus    cache.HealthStatus     `json:"health_status"`
	PerformanceOpt  interface{}            `json:"performance_optimization,omitempty"`
}

// Health Check

// HealthCheck performs comprehensive cache health verification
func (cs *CacheService) HealthCheck(ctx context.Context) *CacheHealthCheck {
	start := time.Now()
	
	result := &CacheHealthCheck{
		Timestamp:   time.Now(),
		TestResults: make(map[string]bool),
	}
	
	// Test basic connectivity
	testKey := fmt.Sprintf("health_check_%d", time.Now().UnixNano())
	testValue := map[string]interface{}{"test": true}
	
	// Test set operation
	if err := cs.FastSet(ctx, testKey, testValue, 1*time.Minute, "health_check"); err != nil {
		result.TestResults["redis_write"] = false
		result.Errors = append(result.Errors, fmt.Sprintf("Write test failed: %v", err))
	} else {
		result.TestResults["redis_write"] = true
	}
	
	// Test get operation
	var retrieved interface{}
	if err := cs.FastGet(ctx, testKey, &retrieved); err != nil {
		result.TestResults["redis_read"] = false
		result.Errors = append(result.Errors, fmt.Sprintf("Read test failed: %v", err))
	} else {
		result.TestResults["redis_read"] = true
	}
	
	// Cleanup test key
	cs.cacheManager.Delete(ctx, testKey)
	
	// Get overall health status
	if cs.monitor != nil {
		healthStatus := cs.monitor.GetHealthStatus()
		result.OverallStatus = healthStatus.Status
		result.Issues = healthStatus.Issues
		result.Recommendations = healthStatus.Recommendations
	} else {
		// Determine status based on test results
		allPassed := true
		for _, passed := range result.TestResults {
			if !passed {
				allPassed = false
				break
			}
		}
		
		if allPassed {
			result.OverallStatus = "healthy"
		} else {
			result.OverallStatus = "unhealthy"
		}
	}
	
	result.ResponseTime = time.Since(start)
	
	return result
}

// CacheHealthCheck represents cache health check results
type CacheHealthCheck struct {
	Timestamp       time.Time         `json:"timestamp"`
	OverallStatus   string            `json:"overall_status"`
	TestResults     map[string]bool   `json:"test_results"`
	ResponseTime    time.Duration     `json:"response_time"`
	Issues          []string          `json:"issues,omitempty"`
	Recommendations []string          `json:"recommendations,omitempty"`
	Errors          []string          `json:"errors,omitempty"`
}

// Close gracefully shuts down the cache service
func (cs *CacheService) Close() error {
	cs.logger.Info("Shutting down cache service")
	
	if cs.monitor != nil {
		// Flush any remaining analytics
		cs.monitor.DisableAnalytics()
	}
	
	if cs.performanceCache != nil {
		// Performance cache cleanup would go here
	}
	
	return cs.cacheManager.Close()
}

// Internal helper methods

func (cs *CacheService) setupCacheWarmingRules() {
	if cs.performanceCache == nil {
		return
	}
	
	warmupRules := []cache.WarmupRule{
		{
			Name:      "recipe_warmup",
			Pattern:   "recipe:*",
			Frequency: 15 * time.Minute,
			Priority:  1,
			DataProvider: func(ctx context.Context) (map[string]interface{}, error) {
				// In production, this would query frequently accessed recipes
				return map[string]interface{}{
					"recipe:common_protocol_1": map[string]interface{}{"protocol": "example"},
					"recipe:common_protocol_2": map[string]interface{}{"protocol": "example2"},
				}, nil
			},
			TTL:  1 * time.Hour,
			Tags: []string{"recipe_resolver", "warmup"},
		},
		{
			Name:      "fhir_warmup",
			Pattern:   "fhir:*",
			Frequency: 30 * time.Minute,
			Priority:  2,
			DataProvider: func(ctx context.Context) (map[string]interface{}, error) {
				// In production, this would query frequently accessed FHIR resources
				return map[string]interface{}{
					"fhir:common_patient_data": map[string]interface{}{"resource": "Patient"},
				}, nil
			},
			TTL:  2 * time.Hour,
			Tags: []string{"google_fhir", "warmup"},
		},
	}
	
	cs.performanceCache.SetupCacheWarming(warmupRules)
	cs.logger.Info("Cache warming rules configured", zap.Int("rules", len(warmupRules)))
}

func (cs *CacheService) setupMonitoringAlerts() {
	if cs.monitor == nil {
		return
	}
	
	// Add alert callback for critical issues
	cs.monitor.AddAlertCallback(func(alert cache.Alert) {
		cs.logger.Error("Cache alert triggered",
			zap.String("type", alert.Type),
			zap.String("severity", alert.Severity),
			zap.String("message", alert.Message),
			zap.Time("timestamp", alert.Timestamp),
		)
		
		// In production, this would send alerts to monitoring systems
		// like PagerDuty, Slack, or email notifications
	})
	
	cs.logger.Info("Cache monitoring alerts configured")
}