package cache

import (
	"context"
	"fmt"
	"time"

	"medication-service-v2/internal/config"
	"go.uber.org/zap"
)

// CacheIntegration provides a unified interface for all caching operations
type CacheIntegration struct {
	manager       *MultiLevelCache
	performance   *PerformanceCache
	serviceCache  *ServiceSpecificCaches
	monitor       *CacheMonitor
	logger        *zap.Logger
	config        config.MultiLevelCacheConfig
}

// NewCacheIntegration creates a new cache integration with full stack
func NewCacheIntegration(redisURL string, cacheConfig config.MultiLevelCacheConfig, logger *zap.Logger) (*CacheIntegration, error) {
	if !cacheConfig.Enabled {
		logger.Info("Multi-level cache disabled in configuration")
		return nil, nil
	}
	
	// Create multi-level cache manager
	managerConfig := CacheConfig{
		RedisURL:           redisURL,
		L1MaxSize:          cacheConfig.L1CacheSize,
		L1TTL:              cacheConfig.L1TTL,
		L2TTL:              cacheConfig.L2TTL,
		PromotionThreshold: cacheConfig.PromotionThreshold,
		DemotionTimeout:    cacheConfig.DemotionTimeout,
		EncryptionEnabled:  cacheConfig.EncryptionEnabled,
		AuditEnabled:       cacheConfig.AuditEnabled,
	}
	
	manager, err := NewMultiLevelCache(managerConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-level cache manager: %w", err)
	}
	
	// Create service-specific caches
	serviceCache := NewServiceSpecificCaches(manager, logger)
	
	// Create performance cache if enabled
	var performance *PerformanceCache
	if cacheConfig.PerformanceOpt {
		performance = NewPerformanceCache(manager, manager.GetRedisClient(), logger)
		
		if cacheConfig.OptimizeForLatency {
			performance.OptimizeForLatency()
		} else {
			performance.OptimizeForThroughput()
		}
		
		logger.Info("Performance cache optimization enabled",
			zap.Bool("optimize_for_latency", cacheConfig.OptimizeForLatency))
	}
	
	// Create monitoring if enabled
	var monitor *CacheMonitor
	if cacheConfig.MonitoringEnabled {
		monitor = NewCacheMonitor(manager, performance, manager.GetRedisClient(), logger)
		
		if cacheConfig.AnalyticsEnabled {
			monitor.EnableAnalytics()
		}
		
		logger.Info("Cache monitoring enabled",
			zap.Bool("analytics_enabled", cacheConfig.AnalyticsEnabled))
	}
	
	integration := &CacheIntegration{
		manager:      manager,
		performance:  performance,
		serviceCache: serviceCache,
		monitor:      monitor,
		logger:       logger.Named("cache_integration"),
		config:       cacheConfig,
	}
	
	// Setup cache warming if enabled
	if cacheConfig.WarmupEnabled && performance != nil {
		integration.setupCacheWarming()
	}
	
	logger.Info("Cache integration initialized successfully",
		zap.String("redis_url", redisURL),
		zap.Bool("performance_enabled", cacheConfig.PerformanceOpt),
		zap.Bool("monitoring_enabled", cacheConfig.MonitoringEnabled),
		zap.Bool("warmup_enabled", cacheConfig.WarmupEnabled),
	)
	
	return integration, nil
}

// Recipe Resolver Cache Operations - <10ms target

// GetRecipe retrieves cached recipe with ultra-fast performance
func (ci *CacheIntegration) GetRecipe(ctx context.Context, protocolID string, patientContext map[string]interface{}) (*RecipeCache, error) {
	start := time.Now()
	
	recipe, err := ci.serviceCache.RecipeResolver().GetRecipe(ctx, protocolID, patientContext)
	latency := time.Since(start)
	
	// Monitor performance against <10ms target
	if ci.monitor != nil {
		ci.monitor.RecordOperation("recipe_resolver", "get_recipe", latency, L2Cache, err == nil, err)
	}
	
	// Alert if performance target is missed
	if err == nil && latency > 10*time.Millisecond {
		ci.logger.Warn("Recipe resolution latency exceeded 10ms target",
			zap.String("protocol_id", protocolID),
			zap.Duration("actual_latency", latency),
			zap.Duration("target", 10*time.Millisecond),
		)
	}
	
	return recipe, err
}

// CacheRecipe stores recipe with intelligent TTL and tagging
func (ci *CacheIntegration) CacheRecipe(ctx context.Context, protocolID string, recipe map[string]interface{}, patientContext map[string]interface{}, dependencies []string) error {
	start := time.Now()
	
	err := ci.serviceCache.RecipeResolver().SetRecipe(ctx, protocolID, recipe, patientContext, dependencies)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("recipe_resolver", "cache_recipe", time.Since(start), L2Cache, err == nil, err)
	}
	
	return err
}

// Clinical Engine Cache Operations

// GetClinicalCalculation retrieves cached Rust engine calculation results
func (ci *CacheIntegration) GetClinicalCalculation(ctx context.Context, calculationID string, inputParams map[string]interface{}) (*ClinicalCalculationResult, error) {
	start := time.Now()
	
	result, err := ci.serviceCache.ClinicalEngine().GetCalculationResult(ctx, calculationID, inputParams)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("clinical_engine", "get_calculation", time.Since(start), L2Cache, err == nil, err)
	}
	
	return result, err
}

// CacheClinicalCalculation stores calculation results with confidence-based TTL
func (ci *CacheIntegration) CacheClinicalCalculation(ctx context.Context, result *ClinicalCalculationResult) error {
	start := time.Now()
	
	err := ci.serviceCache.ClinicalEngine().SetCalculationResult(ctx, result)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("clinical_engine", "cache_calculation", time.Since(start), L2Cache, err == nil, err)
	}
	
	return err
}

// Workflow State Cache Operations

// GetWorkflowState retrieves cached 4-Phase workflow orchestration state
func (ci *CacheIntegration) GetWorkflowState(ctx context.Context, workflowID string) (*WorkflowState, error) {
	start := time.Now()
	
	state, err := ci.serviceCache.WorkflowState().GetWorkflowState(ctx, workflowID)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("workflow_orchestrator", "get_state", time.Since(start), L2Cache, err == nil, err)
	}
	
	return state, err
}

// CacheWorkflowState stores workflow state with dynamic TTL based on phase
func (ci *CacheIntegration) CacheWorkflowState(ctx context.Context, state *WorkflowState) error {
	start := time.Now()
	
	err := ci.serviceCache.WorkflowState().SetWorkflowState(ctx, state)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("workflow_orchestrator", "cache_state", time.Since(start), L2Cache, err == nil, err)
	}
	
	return err
}

// Google FHIR Cache Operations

// GetFHIRResource retrieves cached FHIR resource with ETag validation
func (ci *CacheIntegration) GetFHIRResource(ctx context.Context, projectID, datasetID, fhirStoreID, resourceType, resourceID string) (*FHIRResourceCache, error) {
	start := time.Now()
	
	resource, err := ci.serviceCache.GoogleFHIR().GetFHIRResource(ctx, projectID, datasetID, fhirStoreID, resourceType, resourceID)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("google_fhir", "get_resource", time.Since(start), L2Cache, err == nil, err)
	}
	
	return resource, err
}

// CacheFHIRResource stores FHIR resource with metadata and appropriate TTL
func (ci *CacheIntegration) CacheFHIRResource(ctx context.Context, resource *FHIRResourceCache) error {
	start := time.Now()
	
	err := ci.serviceCache.GoogleFHIR().SetFHIRResource(ctx, resource)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("google_fhir", "cache_resource", time.Since(start), L2Cache, err == nil, err)
	}
	
	return err
}

// Apollo Federation Cache Operations

// GetGraphQLQuery retrieves cached GraphQL query results
func (ci *CacheIntegration) GetGraphQLQuery(ctx context.Context, queryHash string, variables map[string]interface{}) (*GraphQLQueryCache, error) {
	start := time.Now()
	
	query, err := ci.serviceCache.ApolloFederation().GetQueryResult(ctx, queryHash, variables)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("apollo_federation", "get_query", time.Since(start), L2Cache, err == nil, err)
	}
	
	return query, err
}

// CacheGraphQLQuery stores GraphQL query results with intelligent TTL
func (ci *CacheIntegration) CacheGraphQLQuery(ctx context.Context, result *GraphQLQueryCache) error {
	start := time.Now()
	
	err := ci.serviceCache.ApolloFederation().SetQueryResult(ctx, result)
	
	if ci.monitor != nil {
		ci.monitor.RecordOperation("apollo_federation", "cache_query", time.Since(start), L2Cache, err == nil, err)
	}
	
	return err
}

// High-Performance Operations

// FastGet provides ultra-fast retrieval using hot cache optimization
func (ci *CacheIntegration) FastGet(ctx context.Context, key string, dest interface{}) error {
	if ci.performance != nil {
		return ci.performance.FastGet(ctx, key, dest)
	}
	return ci.manager.Get(ctx, key, dest)
}

// FastSet provides optimized storage with hot cache promotion
func (ci *CacheIntegration) FastSet(ctx context.Context, key string, value interface{}, ttl time.Duration, tags ...string) error {
	if ci.performance != nil {
		return ci.performance.FastSet(ctx, key, value, ttl, tags...)
	}
	return ci.manager.Set(ctx, key, value, ttl, tags...)
}

// BatchGet retrieves multiple keys efficiently using pipelining
func (ci *CacheIntegration) BatchGet(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if ci.performance != nil {
		return ci.performance.BatchGet(ctx, keys)
	}
	
	// Fallback implementation
	results := make(map[string]interface{})
	for _, key := range keys {
		var value interface{}
		if err := ci.manager.Get(ctx, key, &value); err == nil {
			results[key] = value
		}
	}
	return results, nil
}

// BatchSet stores multiple key-value pairs efficiently
func (ci *CacheIntegration) BatchSet(ctx context.Context, items map[string]interface{}, ttl time.Duration, tags ...string) error {
	if ci.performance != nil {
		return ci.performance.BatchSet(ctx, items, ttl, tags...)
	}
	
	// Fallback implementation
	for key, value := range items {
		if err := ci.manager.Set(ctx, key, value, ttl, tags...); err != nil {
			return err
		}
	}
	return nil
}

// Cache Management Operations

// InvalidateByTags removes cached entries by tags
func (ci *CacheIntegration) InvalidateByTags(ctx context.Context, tags ...string) error {
	return ci.manager.InvalidateByTags(ctx, tags...)
}

// InvalidateService clears all cache for a service
func (ci *CacheIntegration) InvalidateService(ctx context.Context, serviceName string) error {
	return ci.InvalidateByTags(ctx, serviceName)
}

// InvalidateRecipesByProtocol clears recipe cache for specific protocol
func (ci *CacheIntegration) InvalidateRecipesByProtocol(ctx context.Context, protocolID string) error {
	return ci.serviceCache.RecipeResolver().InvalidateRecipesByProtocol(ctx, protocolID)
}

// WarmupCache preloads frequently accessed data
func (ci *CacheIntegration) WarmupCache(ctx context.Context, serviceName string, data map[string]interface{}) error {
	ttl := 1 * time.Hour
	tags := []string{serviceName, "warmup"}
	
	return ci.BatchSet(ctx, data, ttl, tags...)
}

// Monitoring and Analytics

// GetPerformanceMetrics returns comprehensive cache performance data
func (ci *CacheIntegration) GetPerformanceMetrics() *CachePerformanceReport {
	report := &CachePerformanceReport{
		Timestamp: time.Now(),
		Config:    ci.config,
	}
	
	// Get basic cache stats
	if ci.manager != nil {
		report.BasicStats = ci.manager.GetStats()
	}
	
	// Get detailed monitoring metrics
	if ci.monitor != nil {
		report.DetailedMetrics = ci.monitor.GetMetrics()
		report.HealthStatus = ci.monitor.GetHealthStatus()
	}
	
	// Get performance cache metrics
	if ci.performance != nil {
		perfMetrics := ci.performance.GetPerformanceMetrics()
		report.PerformanceMetrics = &perfMetrics
	}
	
	// Calculate performance grades
	report.calculatePerformanceGrades()
	
	return report
}

// CachePerformanceReport provides comprehensive cache performance analysis
type CachePerformanceReport struct {
	Timestamp          time.Time                      `json:"timestamp"`
	Config             config.MultiLevelCacheConfig  `json:"config"`
	BasicStats         CacheStats                     `json:"basic_stats"`
	DetailedMetrics    CacheMetrics                   `json:"detailed_metrics,omitempty"`
	PerformanceMetrics *PerformanceMetrics            `json:"performance_metrics,omitempty"`
	HealthStatus       HealthStatus                   `json:"health_status,omitempty"`
	
	// Performance grades
	OverallGrade       string            `json:"overall_grade"`
	LatencyGrade       string            `json:"latency_grade"`
	ThroughputGrade    string            `json:"throughput_grade"`
	HitRateGrade       string            `json:"hit_rate_grade"`
	ReliabilityGrade   string            `json:"reliability_grade"`
	Recommendations    []string          `json:"recommendations"`
}

// GetServiceReport returns detailed performance report for specific service
func (ci *CacheIntegration) GetServiceReport(serviceName string) *ServiceReport {
	if ci.monitor == nil {
		return &ServiceReport{
			ServiceName: serviceName,
			Status:      "monitoring_disabled",
		}
	}
	return ci.monitor.GetServiceReport(serviceName)
}

// HealthCheck performs comprehensive health verification
func (ci *CacheIntegration) HealthCheck(ctx context.Context) *CacheHealthResult {
	start := time.Now()
	
	result := &CacheHealthResult{
		Timestamp:    time.Now(),
		ServiceName:  "medication-service-v2-cache",
		TestResults:  make(map[string]bool),
	}
	
	// Test basic operations
	testKey := fmt.Sprintf("health_test_%d", time.Now().UnixNano())
	testData := map[string]interface{}{
		"test": true,
		"timestamp": time.Now(),
	}
	
	// Test write operation
	if err := ci.FastSet(ctx, testKey, testData, 30*time.Second, "health_test"); err != nil {
		result.TestResults["write_operation"] = false
		result.Issues = append(result.Issues, fmt.Sprintf("Write operation failed: %v", err))
	} else {
		result.TestResults["write_operation"] = true
	}
	
	// Test read operation
	var retrieved interface{}
	if err := ci.FastGet(ctx, testKey, &retrieved); err != nil {
		result.TestResults["read_operation"] = false
		result.Issues = append(result.Issues, fmt.Sprintf("Read operation failed: %v", err))
	} else {
		result.TestResults["read_operation"] = true
	}
	
	// Test batch operations
	batchData := map[string]interface{}{
		fmt.Sprintf("batch_test_1_%d", time.Now().UnixNano()): "value1",
		fmt.Sprintf("batch_test_2_%d", time.Now().UnixNano()): "value2",
	}
	
	if err := ci.BatchSet(ctx, batchData, 30*time.Second, "health_test"); err != nil {
		result.TestResults["batch_operations"] = false
		result.Issues = append(result.Issues, fmt.Sprintf("Batch operations failed: %v", err))
	} else {
		result.TestResults["batch_operations"] = true
	}
	
	// Cleanup test data
	ci.InvalidateByTags(ctx, "health_test")
	
	// Get overall health status from monitor
	if ci.monitor != nil {
		healthStatus := ci.monitor.GetHealthStatus()
		result.OverallStatus = healthStatus.Status
		result.Issues = append(result.Issues, healthStatus.Issues...)
		result.Recommendations = healthStatus.Recommendations
	} else {
		// Calculate status based on test results
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
	
	// Performance assessment
	if result.ResponseTime > 100*time.Millisecond {
		result.Issues = append(result.Issues, fmt.Sprintf("Health check response time %v exceeds 100ms", result.ResponseTime))
		result.Recommendations = append(result.Recommendations, "Consider cache optimization or system resource review")
	}
	
	return result
}

// CacheHealthResult represents comprehensive health check results
type CacheHealthResult struct {
	Timestamp       time.Time         `json:"timestamp"`
	ServiceName     string            `json:"service_name"`
	OverallStatus   string            `json:"overall_status"`
	TestResults     map[string]bool   `json:"test_results"`
	ResponseTime    time.Duration     `json:"response_time"`
	Issues          []string          `json:"issues,omitempty"`
	Recommendations []string          `json:"recommendations,omitempty"`
}

// Close gracefully shuts down cache integration
func (ci *CacheIntegration) Close() error {
	ci.logger.Info("Shutting down cache integration")
	
	// Disable analytics to flush remaining events
	if ci.monitor != nil {
		ci.monitor.DisableAnalytics()
	}
	
	// Close cache manager
	if ci.manager != nil {
		return ci.manager.Close()
	}
	
	return nil
}

// Internal helper methods

func (ci *CacheIntegration) setupCacheWarming() {
	if ci.performance == nil {
		return
	}
	
	warmupRules := []WarmupRule{
		{
			Name:      "common_recipes_warmup",
			Pattern:   "recipe:common_*",
			Frequency: ci.config.WarmupInterval,
			Priority:  1,
			DataProvider: func(ctx context.Context) (map[string]interface{}, error) {
				// In production, this would query the most frequently accessed recipes
				commonRecipes := map[string]interface{}{
					"recipe:common_hypertension_protocol": map[string]interface{}{
						"protocol_id": "hypertension_v1",
						"steps": []string{"assessment", "medication", "monitoring"},
					},
					"recipe:common_diabetes_protocol": map[string]interface{}{
						"protocol_id": "diabetes_v1",
						"steps": []string{"glucose_check", "insulin_calc", "dosing"},
					},
				}
				return commonRecipes, nil
			},
			TTL:  2 * time.Hour,
			Tags: []string{"recipe_resolver", "common", "warmup"},
		},
		{
			Name:      "fhir_metadata_warmup",
			Pattern:   "fhir:*:metadata",
			Frequency: 30 * time.Minute,
			Priority:  2,
			DataProvider: func(ctx context.Context) (map[string]interface{}, error) {
				// Warm up common FHIR resource metadata
				metadata := map[string]interface{}{
					"fhir:patient:metadata": map[string]interface{}{
						"resource_type": "Patient",
						"common_fields": []string{"id", "name", "birthDate"},
					},
					"fhir:medication:metadata": map[string]interface{}{
						"resource_type": "Medication",
						"common_fields": []string{"id", "code", "form"},
					},
				}
				return metadata, nil
			},
			TTL:  4 * time.Hour,
			Tags: []string{"google_fhir", "metadata", "warmup"},
		},
	}
	
	ci.performance.SetupCacheWarming(warmupRules)
	ci.logger.Info("Cache warming configured", 
		zap.Int("rules", len(warmupRules)),
		zap.Duration("interval", ci.config.WarmupInterval),
	)
}

func (report *CachePerformanceReport) calculatePerformanceGrades() {
	// Calculate overall hit rate
	hitRate := float64(0)
	if report.BasicStats.L1Hits+report.BasicStats.L1Misses+report.BasicStats.L2Hits+report.BasicStats.L2Misses > 0 {
		totalHits := report.BasicStats.L1Hits + report.BasicStats.L2Hits
		totalRequests := totalHits + report.BasicStats.L1Misses + report.BasicStats.L2Misses
		hitRate = float64(totalHits) / float64(totalRequests)
	}
	
	// Grade hit rate
	if hitRate >= 0.95 {
		report.HitRateGrade = "A+"
	} else if hitRate >= 0.90 {
		report.HitRateGrade = "A"
	} else if hitRate >= 0.80 {
		report.HitRateGrade = "B"
	} else if hitRate >= 0.70 {
		report.HitRateGrade = "C"
	} else {
		report.HitRateGrade = "D"
	}
	
	// Grade latency (if performance metrics available)
	if report.PerformanceMetrics != nil {
		avgLatency := report.PerformanceMetrics.L2AvgLatency
		if avgLatency <= 10*time.Millisecond {
			report.LatencyGrade = "A+"
		} else if avgLatency <= 25*time.Millisecond {
			report.LatencyGrade = "A"
		} else if avgLatency <= 50*time.Millisecond {
			report.LatencyGrade = "B"
		} else if avgLatency <= 100*time.Millisecond {
			report.LatencyGrade = "C"
		} else {
			report.LatencyGrade = "D"
		}
		
		// Grade throughput
		rps := report.PerformanceMetrics.RequestsPerSecond
		if rps >= 1000 {
			report.ThroughputGrade = "A+"
		} else if rps >= 500 {
			report.ThroughputGrade = "A"
		} else if rps >= 100 {
			report.ThroughputGrade = "B"
		} else if rps >= 50 {
			report.ThroughputGrade = "C"
		} else {
			report.ThroughputGrade = "D"
		}
	}
	
	// Grade reliability based on health status
	if report.HealthStatus.Status == "healthy" {
		report.ReliabilityGrade = "A"
	} else if report.HealthStatus.Status == "degraded" {
		report.ReliabilityGrade = "B"
	} else {
		report.ReliabilityGrade = "D"
	}
	
	// Calculate overall grade (weighted average)
	grades := map[string]int{
		"A+": 100, "A": 90, "B": 80, "C": 70, "D": 60,
	}
	
	hitRateScore := grades[report.HitRateGrade] * 30      // 30% weight
	latencyScore := grades[report.LatencyGrade] * 25      // 25% weight  
	throughputScore := grades[report.ThroughputGrade] * 25 // 25% weight
	reliabilityScore := grades[report.ReliabilityGrade] * 20 // 20% weight
	
	totalScore := (hitRateScore + latencyScore + throughputScore + reliabilityScore) / 100
	
	if totalScore >= 95 {
		report.OverallGrade = "A+"
	} else if totalScore >= 85 {
		report.OverallGrade = "A"
	} else if totalScore >= 75 {
		report.OverallGrade = "B"
	} else if totalScore >= 65 {
		report.OverallGrade = "C"
	} else {
		report.OverallGrade = "D"
	}
	
	// Generate recommendations
	if hitRate < 0.80 {
		report.Recommendations = append(report.Recommendations, "Consider implementing cache warming strategies to improve hit rate")
	}
	
	if report.PerformanceMetrics != nil && report.PerformanceMetrics.L2AvgLatency > 50*time.Millisecond {
		report.Recommendations = append(report.Recommendations, "Optimize cache key structure and consider hot cache tuning for better latency")
	}
	
	if report.HealthStatus.Status != "healthy" {
		report.Recommendations = append(report.Recommendations, "Address health issues to improve reliability")
	}
}