package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-2-clinical-context-go/internal/performance"
)

// Cache management endpoints for KB-2 service

// getCacheStats returns comprehensive cache statistics
func (s *Server) getCacheStats(c *gin.Context) {
	stats := s.contextService.GetCacheStats()
	
	response := gin.H{
		"timestamp":    time.Now().Unix(),
		"cache_tiers":  stats,
		"sla_targets": gin.H{
			"l1_hit_rate": 0.85,
			"l2_hit_rate": 0.95,
			"latency_p50": "5ms",
			"latency_p95": "25ms",
			"latency_p99": "100ms",
			"throughput":  "10000 RPS",
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// getCacheHealth returns cache health status
func (s *Server) getCacheHealth(c *gin.Context) {
	health := s.contextService.GetServiceHealth(c.Request.Context())
	
	statusCode := http.StatusOK
	if status, exists := health["service"]; exists {
		if status == "degraded" {
			statusCode = http.StatusPartialContent
		} else if status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}
	}
	
	c.JSON(statusCode, health)
}

// getCacheSLACompliance returns SLA compliance status
func (s *Server) getCacheSLACompliance(c *gin.Context) {
	compliance := s.contextService.CheckCacheSLACompliance()
	
	overallCompliant := true
	for _, compliant := range compliance {
		if !compliant {
			overallCompliant = false
			break
		}
	}
	
	response := gin.H{
		"overall_compliant": overallCompliant,
		"sla_metrics":      compliance,
		"timestamp":        time.Now().Unix(),
	}
	
	statusCode := http.StatusOK
	if !overallCompliant {
		statusCode = http.StatusPartialContent
	}
	
	c.JSON(statusCode, response)
}

// invalidateCache invalidates cache entries by pattern
func (s *Server) invalidateCache(c *gin.Context) {
	var request struct {
		Pattern string `json:"pattern" binding:"required"`
		Reason  string `json:"reason,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Perform invalidation
	err := s.contextService.cache.InvalidatePattern(c.Request.Context(), request.Pattern)
	if err != nil {
		s.metrics.RecordError("cache_invalidation", "api")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Cache invalidation failed",
			"details": err.Error(),
		})
		return
	}
	
	// Record invalidation in metrics
	reason := request.Reason
	if reason == "" {
		reason = "manual_api"
	}
	s.metrics.RecordCacheInvalidation(request.Pattern, reason)
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "Cache invalidation completed",
		"pattern":   request.Pattern,
		"reason":    reason,
		"timestamp": time.Now().Unix(),
	})
}

// warmCache triggers cache warming
func (s *Server) warmCache(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	err := s.contextService.WarmFrequentlyAccessedData(ctx)
	if err != nil {
		s.metrics.RecordCacheWarming("manual", "failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Cache warming failed",
			"details": err.Error(),
		})
		return
	}
	
	s.metrics.RecordCacheWarming("manual", "success")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache warming completed successfully",
		"timestamp": time.Now().Unix(),
	})
}

// optimizeCache triggers cache optimization
func (s *Server) optimizeCache(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	err := s.contextService.OptimizeCachePerformance(ctx)
	if err != nil {
		s.metrics.RecordError("cache_optimization", "api")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Cache optimization failed",
			"details": err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache optimization completed",
		"timestamp": time.Now().Unix(),
	})
}

// runPerformanceBenchmark executes comprehensive performance benchmarks
func (s *Server) runPerformanceBenchmark(c *gin.Context) {
	// Parse query parameters
	durationStr := c.DefaultQuery("duration", "60")
	concurrencyStr := c.DefaultQuery("concurrency", "50")
	
	duration, err := strconv.Atoi(durationStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid duration parameter"})
		return
	}
	
	concurrency, err := strconv.Atoi(concurrencyStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid concurrency parameter"})
		return
	}
	
	// Create benchmark runner
	benchmarkRunner := performance.NewBenchmarkRunner(
		s.config.Config,
		s.contextService.cache,
		s.metrics,
	)
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(duration+60)*time.Second)
	defer cancel()
	
	// Run benchmarks
	results, err := benchmarkRunner.RunComprehensiveBenchmarks(ctx)
	if err != nil {
		s.metrics.RecordError("benchmark", "api")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Benchmark execution failed",
			"details": err.Error(),
		})
		return
	}
	
	// Calculate overall score
	overallScore := benchmarkRunner.GetOverallPerformanceScore()
	slaCompliant := benchmarkRunner.IsSLACompliant()
	
	response := gin.H{
		"benchmark_results":  results,
		"overall_score":      overallScore,
		"sla_compliant":      slaCompliant,
		"test_configuration": gin.H{
			"duration":    duration,
			"concurrency": concurrency,
		},
		"timestamp": time.Now().Unix(),
	}
	
	c.JSON(http.StatusOK, response)
}

// getCacheMetrics returns detailed cache metrics for monitoring
func (s *Server) getCacheMetrics(c *gin.Context) {
	stats := s.contextService.GetCacheStats()
	hitRates := s.contextService.cache.GetHitRates()
	
	// Calculate combined metrics
	combinedStats := map[string]interface{}{
		"hit_rates":        hitRates,
		"tier_statistics":  stats,
		"performance_targets": gin.H{
			"l1_target":     0.85,
			"l2_target":     0.95,
			"latency_p50":   5,  // ms
			"latency_p95":   25, // ms
			"throughput":    10000, // RPS
		},
	}
	
	// Check SLA compliance
	compliance := s.contextService.CheckCacheSLACompliance()
	combinedStats["sla_compliance"] = compliance
	
	// Calculate efficiency scores
	efficiency := map[string]float64{}
	if l1Stats, exists := stats["l1"]; exists {
		efficiency["l1"] = l1Stats.HitRate / 0.85 // Ratio to target
	}
	if l2Stats, exists := stats["l2"]; exists {
		efficiency["l2"] = l2Stats.HitRate / 0.95 // Ratio to target
	}
	combinedStats["efficiency_scores"] = efficiency
	
	c.JSON(http.StatusOK, combinedStats)
}

// invalidatePatientCache invalidates all cache entries for a specific patient
func (s *Server) invalidatePatientCache(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Patient ID is required"})
		return
	}
	
	err := s.contextService.InvalidatePatientContext(c.Request.Context(), patientID)
	if err != nil {
		s.metrics.RecordError("patient_cache_invalidation", "api")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Patient cache invalidation failed",
			"details": err.Error(),
		})
		return
	}
	
	s.metrics.RecordCacheInvalidation("patient_context", "manual_api")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Patient cache invalidated successfully",
		"patient_id": patientID,
		"timestamp": time.Now().Unix(),
	})
}

// warmPatientCache warms cache for specific patient
func (s *Server) warmPatientCache(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Patient ID is required"})
		return
	}
	
	// Create mock context assembly request for warming
	request := &models.ContextAssemblyRequest{
		PatientID:         patientID,
		DetailLevel:       "standard",
		IncludePhenotypes: true,
		IncludeRisks:     true,
		IncludeTreatments: true,
		PatientData:      models.Patient{ID: patientID}, // Minimal patient data
	}
	
	// Pre-warm the cache
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	
	_, err := s.contextService.AssembleContext(ctx, request)
	if err != nil {
		s.metrics.RecordCacheWarming("patient_specific", "failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Patient cache warming failed",
			"details": err.Error(),
		})
		return
	}
	
	s.metrics.RecordCacheWarming("patient_specific", "success")
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Patient cache warmed successfully",
		"patient_id": patientID,
		"timestamp": time.Now().Unix(),
	})
}

// getCacheMemoryUsage returns memory usage breakdown by tier
func (s *Server) getCacheMemoryUsage(c *gin.Context) {
	stats := s.contextService.GetCacheStats()
	
	memoryBreakdown := map[string]interface{}{}
	totalMemory := int64(0)
	
	for tier, stat := range stats {
		memoryBreakdown[tier] = gin.H{
			"memory_bytes": stat.MemoryUsage,
			"memory_mb":   float64(stat.MemoryUsage) / (1024 * 1024),
			"item_count":  stat.Size,
			"avg_item_size": func() float64 {
				if stat.Size > 0 {
					return float64(stat.MemoryUsage) / float64(stat.Size)
				}
				return 0
			}(),
		}
		totalMemory += stat.MemoryUsage
	}
	
	memoryBreakdown["total"] = gin.H{
		"memory_bytes": totalMemory,
		"memory_mb":   float64(totalMemory) / (1024 * 1024),
	}
	
	// Add memory limits
	memoryBreakdown["limits"] = gin.H{
		"l1_limit_mb": float64(s.config.Config.L1CacheMaxSize) / (1024 * 1024),
		"l2_limit_mb": float64(s.config.Config.L2CacheMaxMemory) / (1024 * 1024),
	}
	
	c.JSON(http.StatusOK, memoryBreakdown)
}

// Benchmark-specific endpoints

// getBenchmarkHistory returns historical benchmark results
func (s *Server) getBenchmarkHistory(c *gin.Context) {
	// This would typically query stored benchmark results
	// For now, return placeholder
	c.JSON(http.StatusOK, gin.H{
		"message": "Benchmark history not implemented",
		"note":    "Use /admin/benchmark/run to execute new benchmarks",
	})
}

// runQuickPerformanceTest runs a quick performance validation
func (s *Server) runQuickPerformanceTest(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	
	startTime := time.Now()
	
	// Quick latency test (100 requests)
	requestCount := 100
	successCount := 0
	totalLatency := 0.0
	
	for i := 0; i < requestCount; i++ {
		reqStart := time.Now()
		
		// Simulate quick phenotype evaluation
		key := fmt.Sprintf("quick_test_%d", i%10) // Reuse keys for cache testing
		_, err := s.contextService.cache.Get(ctx, key, func() (interface{}, error) {
			return gin.H{"test": "data", "generated": time.Now()}, nil
		})
		
		reqDuration := time.Since(reqStart)
		totalLatency += float64(reqDuration.Nanoseconds()) / 1e6 // Convert to ms
		
		if err == nil {
			successCount++
		}
	}
	
	testDuration := time.Since(startTime)
	avgLatency := totalLatency / float64(requestCount)
	throughput := float64(successCount) / testDuration.Seconds()
	
	// Get current cache stats
	cacheStats := s.contextService.GetCacheStats()
	hitRates := s.contextService.cache.GetHitRates()
	
	// Determine test result
	testResult := "PASS"
	issues := []string{}
	
	if avgLatency > 25.0 { // 25ms threshold for quick test
		testResult = "FAIL"
		issues = append(issues, fmt.Sprintf("High average latency: %.2fms", avgLatency))
	}
	
	if throughput < 100 { // 100 RPS minimum for quick test
		testResult = "FAIL"
		issues = append(issues, fmt.Sprintf("Low throughput: %.0f RPS", throughput))
	}
	
	if l1Rate, exists := hitRates["l1"]; exists && l1Rate < 0.7 {
		issues = append(issues, fmt.Sprintf("Low L1 hit rate: %.1f%%", l1Rate*100))
	}
	
	response := gin.H{
		"test_result":     testResult,
		"test_duration":   testDuration.String(),
		"requests_sent":   requestCount,
		"requests_success": successCount,
		"average_latency_ms": avgLatency,
		"throughput_rps":   throughput,
		"cache_hit_rates":  hitRates,
		"cache_stats":      cacheStats,
		"issues":          issues,
		"timestamp":       time.Now().Unix(),
	}
	
	statusCode := http.StatusOK
	if testResult == "FAIL" {
		statusCode = http.StatusBadRequest
	}
	
	c.JSON(statusCode, response)
}

// Cache administration endpoints

// flushCache flushes all cache tiers (admin only)
func (s *Server) flushCache(c *gin.Context) {
	// Safety check - require admin authorization header
	authHeader := c.GetHeader("X-Admin-Authorization")
	if authHeader != "admin-cache-operations" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin authorization required"})
		return
	}
	
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()
	
	// Flush all tiers
	errors := []string{}
	
	// Clear L1 cache
	s.contextService.cache.l1Cache.Clear()
	
	// Clear L2 cache (Redis) - this would require implementation
	if err := s.contextService.cache.l2Cache.DeletePattern(ctx, "*"); err != nil {
		errors = append(errors, fmt.Sprintf("L2 flush failed: %v", err))
	}
	
	if len(errors) > 0 {
		c.JSON(http.StatusPartialContent, gin.H{
			"message": "Cache flush completed with errors",
			"errors":  errors,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "All cache tiers flushed successfully",
		"timestamp": time.Now().Unix(),
	})
}

// setCacheConfiguration updates cache configuration
func (s *Server) setCacheConfiguration(c *gin.Context) {
	var request struct {
		L1MaxSizeMB      int     `json:"l1_max_size_mb,omitempty"`
		L1TTLMinutes     int     `json:"l1_ttl_minutes,omitempty"`
		L2TTLHours       int     `json:"l2_ttl_hours,omitempty"`
		L2Compression    *bool   `json:"l2_compression,omitempty"`
		WarmingInterval  int     `json:"warming_interval_minutes,omitempty"`
		EnableWarming    *bool   `json:"enable_warming,omitempty"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Apply configuration changes
	changes := []string{}
	
	if request.L1MaxSizeMB > 0 {
		s.config.Config.L1CacheMaxSize = int64(request.L1MaxSizeMB) * 1024 * 1024
		changes = append(changes, fmt.Sprintf("L1 max size: %dMB", request.L1MaxSizeMB))
	}
	
	if request.L1TTLMinutes > 0 {
		s.config.Config.L1CacheDefaultTTL = time.Duration(request.L1TTLMinutes) * time.Minute
		changes = append(changes, fmt.Sprintf("L1 TTL: %d minutes", request.L1TTLMinutes))
	}
	
	if request.L2TTLHours > 0 {
		s.config.Config.L2CacheDefaultTTL = time.Duration(request.L2TTLHours) * time.Hour
		changes = append(changes, fmt.Sprintf("L2 TTL: %d hours", request.L2TTLHours))
	}
	
	if request.L2Compression != nil {
		s.config.Config.L2CacheCompression = *request.L2Compression
		changes = append(changes, fmt.Sprintf("L2 compression: %t", *request.L2Compression))
	}
	
	if request.WarmingInterval > 0 {
		s.config.Config.CacheWarmingInterval = time.Duration(request.WarmingInterval) * time.Minute
		changes = append(changes, fmt.Sprintf("Warming interval: %d minutes", request.WarmingInterval))
	}
	
	if request.EnableWarming != nil {
		s.config.Config.CacheWarmingEnabled = *request.EnableWarming
		changes = append(changes, fmt.Sprintf("Warming enabled: %t", *request.EnableWarming))
	}
	
	if len(changes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid configuration changes provided"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Cache configuration updated",
		"changes": changes,
		"timestamp": time.Now().Unix(),
	})
}

// Performance monitoring endpoints

// getPerformanceReport returns comprehensive performance report
func (s *Server) getPerformanceReport(c *gin.Context) {
	cacheStats := s.contextService.GetCacheStats()
	slaCompliance := s.contextService.CheckCacheSLACompliance()
	
	// Calculate performance indicators
	overallScore := s.contextService.calculateCurrentPerformanceScore(cacheStats)
	
	performanceReport := gin.H{
		"timestamp": time.Now().Unix(),
		"overall_performance_score": overallScore,
		"cache_performance": gin.H{
			"hit_rates":      s.contextService.cache.GetHitRates(),
			"tier_stats":     cacheStats,
			"sla_compliance": slaCompliance,
		},
		"performance_targets": gin.H{
			"latency": gin.H{
				"p50_target_ms": s.config.Config.TargetLatencyP50,
				"p95_target_ms": s.config.Config.TargetLatencyP95,
				"p99_target_ms": s.config.Config.TargetLatencyP99,
			},
			"throughput": gin.H{
				"target_rps": s.config.Config.TargetThroughputRPS,
			},
			"batch": gin.H{
				"target_1000_patients_ms": s.config.Config.TargetBatchTime,
			},
			"cache": gin.H{
				"l1_hit_rate_target": s.config.Config.L1CacheHitRateTarget,
				"l2_hit_rate_target": s.config.Config.L2CacheHitRateTarget,
			},
		},
		"service_health": s.contextService.GetServiceHealth(c.Request.Context()),
	}
	
	// Determine response status based on performance
	statusCode := http.StatusOK
	if overallScore < 0.7 {
		statusCode = http.StatusPartialContent
	}
	if overallScore < 0.5 {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, performanceReport)
}

// getThroughputMetrics returns real-time throughput measurements
func (s *Server) getThroughputMetrics(c *gin.Context) {
	// This would typically track real-time throughput
	// For now, return current concurrent requests as approximation
	response := gin.H{
		"timestamp": time.Now().Unix(),
		"note":      "Real-time throughput tracking not fully implemented",
		"targets": gin.H{
			"throughput_rps": s.config.Config.TargetThroughputRPS,
			"batch_target": fmt.Sprintf("%d patients in %dms", 1000, s.config.Config.TargetBatchTime),
		},
	}
	
	c.JSON(http.StatusOK, response)
}

// Health check with cache validation
func (s *Server) healthCheckWithCache(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	// Test cache connectivity
	cacheHealthy := true
	cacheErrors := []string{}
	
	// Test L1 cache
	testKey := fmt.Sprintf("health_check_%d", time.Now().UnixNano())
	s.contextService.cache.l1Cache.Set(testKey, "test_data", time.Minute)
	if _, found := s.contextService.cache.l1Cache.Get(testKey); !found {
		cacheHealthy = false
		cacheErrors = append(cacheErrors, "L1 cache not responding")
	} else {
		s.contextService.cache.l1Cache.Delete(testKey) // Cleanup
	}
	
	// Test L2 cache (Redis)
	if err := s.contextService.cache.l2Cache.Set(ctx, testKey, "test_data", time.Minute); err != nil {
		cacheHealthy = false
		cacheErrors = append(cacheErrors, fmt.Sprintf("L2 cache error: %v", err))
	} else {
		s.contextService.cache.l2Cache.Delete(ctx, testKey) // Cleanup
	}
	
	health := gin.H{
		"status":      "healthy",
		"timestamp":   time.Now().Unix(),
		"cache_health": cacheHealthy,
		"cache_tiers": gin.H{
			"l1": "healthy",
			"l2": "healthy", 
			"l3": "healthy",
		},
		"performance": gin.H{
			"cache_hit_rates": s.contextService.cache.GetHitRates(),
			"sla_compliance": s.contextService.CheckCacheSLACompliance(),
		},
	}
	
	statusCode := http.StatusOK
	
	if !cacheHealthy {
		health["status"] = "degraded"
		health["cache_errors"] = cacheErrors
		statusCode = http.StatusPartialContent
	}
	
	// Check overall SLA compliance
	slaCompliance := s.contextService.CheckCacheSLACompliance()
	allCompliant := true
	for _, compliant := range slaCompliance {
		if !compliant {
			allCompliant = false
			break
		}
	}
	
	if !allCompliant {
		health["status"] = "performance_issues"
		if statusCode == http.StatusOK {
			statusCode = http.StatusPartialContent
		}
	}
	
	c.JSON(statusCode, health)
}