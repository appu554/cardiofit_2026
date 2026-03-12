package cache

import (
	"context"
	//"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CacheMonitor provides comprehensive cache monitoring and analytics
type CacheMonitor struct {
	cache       *MultiLevelCache
	perfCache   *PerformanceCache
	redis       *redis.Client
	logger      *zap.Logger
	
	// Monitoring state
	metrics     *CacheMetrics
	metricsMutex sync.RWMutex
	
	// Alerting
	alertThresholds *AlertThresholds
	alertCallbacks  []AlertCallback
	
	// Health checking
	healthStatus *HealthStatus
	
	// Analytics storage
	analyticsEnabled bool
	analyticsBuffer  []AnalyticsEvent
	analyticsMutex   sync.Mutex
}

// CacheMetrics provides detailed cache performance metrics
type CacheMetrics struct {
	// Overall statistics
	TotalRequests     int64     `json:"total_requests"`
	TotalHits         int64     `json:"total_hits"`
	TotalMisses       int64     `json:"total_misses"`
	HitRate           float64   `json:"hit_rate"`
	MissRate          float64   `json:"miss_rate"`
	LastUpdated       time.Time `json:"last_updated"`
	
	// Performance metrics
	AverageLatency    time.Duration `json:"average_latency"`
	P50Latency        time.Duration `json:"p50_latency"`
	P90Latency        time.Duration `json:"p90_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	
	// Memory and resource usage
	MemoryUsage       int64   `json:"memory_usage"`
	MemoryUtilization float64 `json:"memory_utilization"`
	ConnectionsActive int64   `json:"connections_active"`
	ConnectionsIdle   int64   `json:"connections_idle"`
	
	// Service-specific metrics
	ServiceMetrics map[string]*ServiceMetrics `json:"service_metrics"`
	
	// Error tracking
	Errors    int64            `json:"errors"`
	Timeouts  int64            `json:"timeouts"`
	ErrorRate float64          `json:"error_rate"`
	ErrorBreakdown map[string]int64 `json:"error_breakdown"`
	
	// Cache efficiency
	L1Efficiency    float64 `json:"l1_efficiency"`
	L2Efficiency    float64 `json:"l2_efficiency"`
	HotCacheEfficiency float64 `json:"hot_cache_efficiency"`
	
	// Throughput
	RequestsPerSecond   float64 `json:"requests_per_second"`
	OperationsPerSecond float64 `json:"operations_per_second"`
}

// ServiceMetrics tracks metrics per service type
type ServiceMetrics struct {
	ServiceName     string        `json:"service_name"`
	RequestCount    int64         `json:"request_count"`
	HitCount        int64         `json:"hit_count"`
	MissCount       int64         `json:"miss_count"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorCount      int64         `json:"error_count"`
	LastAccessed    time.Time     `json:"last_accessed"`
}

// HealthStatus represents cache system health
type HealthStatus struct {
	Status           string            `json:"status"` // healthy, degraded, unhealthy
	LastCheck        time.Time         `json:"last_check"`
	ResponseTime     time.Duration     `json:"response_time"`
	RedisConnected   bool              `json:"redis_connected"`
	MemoryPressure   bool              `json:"memory_pressure"`
	ErrorRateHigh    bool              `json:"error_rate_high"`
	LatencyHigh      bool              `json:"latency_high"`
	Issues           []string          `json:"issues,omitempty"`
	Recommendations  []string          `json:"recommendations,omitempty"`
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	HitRateBelow      float64       `json:"hit_rate_below"`
	ErrorRateAbove    float64       `json:"error_rate_above"`
	LatencyAbove      time.Duration `json:"latency_above"`
	MemoryUsageAbove  float64       `json:"memory_usage_above"`
	ConnectionsBelow  int64         `json:"connections_below"`
}

// AlertCallback function type for handling alerts
type AlertCallback func(alert Alert)

// Alert represents a cache monitoring alert
type Alert struct {
	Type        string            `json:"type"`
	Severity    string            `json:"severity"` // low, medium, high, critical
	Message     string            `json:"message"`
	Metrics     map[string]interface{} `json:"metrics"`
	Timestamp   time.Time         `json:"timestamp"`
	ServiceName string            `json:"service_name,omitempty"`
}

// AnalyticsEvent represents a cache operation for analytics
type AnalyticsEvent struct {
	EventType   string                 `json:"event_type"`
	ServiceName string                 `json:"service_name"`
	Operation   string                 `json:"operation"`
	Key         string                 `json:"key"`
	Latency     time.Duration          `json:"latency"`
	CacheLevel  CacheLevel             `json:"cache_level"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewCacheMonitor creates a new cache monitoring system
func NewCacheMonitor(cache *MultiLevelCache, perfCache *PerformanceCache, redis *redis.Client, logger *zap.Logger) *CacheMonitor {
	monitor := &CacheMonitor{
		cache:     cache,
		perfCache: perfCache,
		redis:     redis,
		logger:    logger.Named("cache_monitor"),
		metrics: &CacheMetrics{
			ServiceMetrics: make(map[string]*ServiceMetrics),
			ErrorBreakdown: make(map[string]int64),
			LastUpdated:    time.Now(),
		},
		alertThresholds: &AlertThresholds{
			HitRateBelow:     0.80, // Alert if hit rate drops below 80%
			ErrorRateAbove:   0.05, // Alert if error rate exceeds 5%
			LatencyAbove:     100 * time.Millisecond,
			MemoryUsageAbove: 0.85, // Alert if memory usage exceeds 85%
			ConnectionsBelow: 5,    // Alert if active connections drop below 5
		},
		healthStatus: &HealthStatus{
			Status: "healthy",
			LastCheck: time.Now(),
		},
		analyticsEnabled: true,
		analyticsBuffer:  make([]AnalyticsEvent, 0, 1000),
	}
	
	// Start monitoring loops
	go monitor.metricsCollectionLoop()
	go monitor.healthCheckLoop()
	go monitor.analyticsFlushLoop()
	
	logger.Info("Cache monitoring system initialized")
	return monitor
}

// RecordOperation records a cache operation for monitoring
func (cm *CacheMonitor) RecordOperation(serviceName, operation string, latency time.Duration, cacheLevel CacheLevel, success bool, err error) {
	cm.metricsMutex.Lock()
	defer cm.metricsMutex.Unlock()
	
	// Update overall metrics
	cm.metrics.TotalRequests++
	if success {
		cm.metrics.TotalHits++
	} else {
		cm.metrics.TotalMisses++
		if err != nil {
			cm.metrics.Errors++
			cm.metrics.ErrorBreakdown[err.Error()]++
		}
	}
	
	// Update service-specific metrics
	serviceMetrics, exists := cm.metrics.ServiceMetrics[serviceName]
	if !exists {
		serviceMetrics = &ServiceMetrics{
			ServiceName: serviceName,
		}
		cm.metrics.ServiceMetrics[serviceName] = serviceMetrics
	}
	
	serviceMetrics.RequestCount++
	if success {
		serviceMetrics.HitCount++
	} else {
		serviceMetrics.MissCount++
		if err != nil {
			serviceMetrics.ErrorCount++
		}
	}
	serviceMetrics.AverageLatency = cm.updateAverage(serviceMetrics.AverageLatency, latency)
	serviceMetrics.LastAccessed = time.Now()
	
	// Update overall averages
	cm.metrics.AverageLatency = cm.updateAverage(cm.metrics.AverageLatency, latency)
	
	// Calculate rates
	if cm.metrics.TotalRequests > 0 {
		cm.metrics.HitRate = float64(cm.metrics.TotalHits) / float64(cm.metrics.TotalRequests)
		cm.metrics.MissRate = float64(cm.metrics.TotalMisses) / float64(cm.metrics.TotalRequests)
		cm.metrics.ErrorRate = float64(cm.metrics.Errors) / float64(cm.metrics.TotalRequests)
	}
	
	cm.metrics.LastUpdated = time.Now()
	
	// Record analytics event if enabled
	if cm.analyticsEnabled {
		event := AnalyticsEvent{
			EventType:   "cache_operation",
			ServiceName: serviceName,
			Operation:   operation,
			Latency:     latency,
			CacheLevel:  cacheLevel,
			Success:     success,
			Timestamp:   time.Now(),
		}
		if err != nil {
			event.Error = err.Error()
		}
		
		cm.recordAnalyticsEvent(event)
	}
	
	// Check for alerts
	cm.checkAlerts()
}

// GetMetrics returns current cache metrics
func (cm *CacheMonitor) GetMetrics() CacheMetrics {
	cm.metricsMutex.RLock()
	defer cm.metricsMutex.RUnlock()
	
	metrics := *cm.metrics
	// Deep copy service metrics
	metrics.ServiceMetrics = make(map[string]*ServiceMetrics)
	for k, v := range cm.metrics.ServiceMetrics {
		serviceMetrics := *v
		metrics.ServiceMetrics[k] = &serviceMetrics
	}
	
	return metrics
}

// GetHealthStatus returns current health status
func (cm *CacheMonitor) GetHealthStatus() HealthStatus {
	return *cm.healthStatus
}

// AddAlertCallback adds a callback function for handling alerts
func (cm *CacheMonitor) AddAlertCallback(callback AlertCallback) {
	cm.alertCallbacks = append(cm.alertCallbacks, callback)
}

// SetAlertThresholds updates alert thresholds
func (cm *CacheMonitor) SetAlertThresholds(thresholds *AlertThresholds) {
	cm.alertThresholds = thresholds
}

// GetServiceReport generates a detailed report for a specific service
func (cm *CacheMonitor) GetServiceReport(serviceName string) *ServiceReport {
	cm.metricsMutex.RLock()
	defer cm.metricsMutex.RUnlock()
	
	serviceMetrics, exists := cm.metrics.ServiceMetrics[serviceName]
	if !exists {
		return nil
	}
	
	hitRate := float64(0)
	if serviceMetrics.RequestCount > 0 {
		hitRate = float64(serviceMetrics.HitCount) / float64(serviceMetrics.RequestCount)
	}
	
	report := &ServiceReport{
		ServiceName:    serviceName,
		RequestCount:   serviceMetrics.RequestCount,
		HitRate:        hitRate,
		AverageLatency: serviceMetrics.AverageLatency,
		ErrorRate:      float64(serviceMetrics.ErrorCount) / float64(serviceMetrics.RequestCount),
		LastAccessed:   serviceMetrics.LastAccessed,
		Status:         cm.getServiceStatus(serviceMetrics),
		Recommendations: cm.generateRecommendations(serviceMetrics),
	}
	
	return report
}

// ServiceReport contains detailed service performance report
type ServiceReport struct {
	ServiceName     string        `json:"service_name"`
	RequestCount    int64         `json:"request_count"`
	HitRate         float64       `json:"hit_rate"`
	AverageLatency  time.Duration `json:"average_latency"`
	ErrorRate       float64       `json:"error_rate"`
	LastAccessed    time.Time     `json:"last_accessed"`
	Status          string        `json:"status"`
	Recommendations []string      `json:"recommendations"`
}

// ResetMetrics resets all metrics
func (cm *CacheMonitor) ResetMetrics() {
	cm.metricsMutex.Lock()
	defer cm.metricsMutex.Unlock()
	
	cm.metrics = &CacheMetrics{
		ServiceMetrics: make(map[string]*ServiceMetrics),
		ErrorBreakdown: make(map[string]int64),
		LastUpdated:    time.Now(),
	}
	
	cm.logger.Info("Cache metrics reset")
}

// EnableAnalytics enables detailed analytics collection
func (cm *CacheMonitor) EnableAnalytics() {
	cm.analyticsEnabled = true
	cm.logger.Info("Cache analytics enabled")
}

// DisableAnalytics disables analytics collection
func (cm *CacheMonitor) DisableAnalytics() {
	cm.analyticsEnabled = false
	cm.logger.Info("Cache analytics disabled")
}

// Internal helper methods

func (cm *CacheMonitor) metricsCollectionLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		cm.collectSystemMetrics()
	}
}

func (cm *CacheMonitor) collectSystemMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get Redis info
	_, err := cm.redis.Info(ctx, "memory", "clients", "stats").Result()
	if err != nil {
		cm.logger.Error("Failed to get Redis info", zap.Error(err))
		return
	}
	
	// Parse memory usage (simplified - in production would parse actual Redis INFO output)
	cm.metricsMutex.Lock()
	cm.metrics.MemoryUsage = 1024 * 1024 * 50 // Placeholder: 50MB
	cm.metrics.MemoryUtilization = 0.45       // Placeholder: 45%
	cm.metrics.ConnectionsActive = 10         // Placeholder
	cm.metrics.ConnectionsIdle = 5           // Placeholder
	cm.metricsMutex.Unlock()
	
	// Get performance cache metrics if available
	if cm.perfCache != nil {
		perfMetrics := cm.perfCache.GetPerformanceMetrics()
		cm.metricsMutex.Lock()
		cm.metrics.RequestsPerSecond = perfMetrics.RequestsPerSecond
		cm.metrics.HotCacheEfficiency = perfMetrics.HotCacheHitRate
		cm.metricsMutex.Unlock()
	}
}

func (cm *CacheMonitor) healthCheckLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		cm.performHealthCheck()
	}
}

func (cm *CacheMonitor) performHealthCheck() {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	issues := make([]string, 0)
	recommendations := make([]string, 0)
	status := "healthy"
	
	// Check Redis connectivity
	redisConnected := true
	if err := cm.redis.Ping(ctx).Err(); err != nil {
		redisConnected = false
		issues = append(issues, "Redis connection failed")
		status = "unhealthy"
	}
	
	// Check metrics thresholds
	cm.metricsMutex.RLock()
	metrics := *cm.metrics
	cm.metricsMutex.RUnlock()
	
	// Check hit rate
	if metrics.HitRate < cm.alertThresholds.HitRateBelow {
		issues = append(issues, fmt.Sprintf("Hit rate is %.2f%%, below threshold %.2f%%", 
			metrics.HitRate*100, cm.alertThresholds.HitRateBelow*100))
		recommendations = append(recommendations, "Consider cache warming or TTL optimization")
		if status == "healthy" {
			status = "degraded"
		}
	}
	
	// Check error rate
	if metrics.ErrorRate > cm.alertThresholds.ErrorRateAbove {
		issues = append(issues, fmt.Sprintf("Error rate is %.2f%%, above threshold %.2f%%",
			metrics.ErrorRate*100, cm.alertThresholds.ErrorRateAbove*100))
		recommendations = append(recommendations, "Investigate cache connection and configuration")
		status = "unhealthy"
	}
	
	// Check average latency
	if metrics.AverageLatency > cm.alertThresholds.LatencyAbove {
		issues = append(issues, fmt.Sprintf("Average latency is %v, above threshold %v",
			metrics.AverageLatency, cm.alertThresholds.LatencyAbove))
		recommendations = append(recommendations, "Consider performance optimization or hot cache tuning")
		if status == "healthy" {
			status = "degraded"
		}
	}
	
	// Check memory usage
	if metrics.MemoryUtilization > cm.alertThresholds.MemoryUsageAbove {
		issues = append(issues, fmt.Sprintf("Memory utilization is %.2f%%, above threshold %.2f%%",
			metrics.MemoryUtilization*100, cm.alertThresholds.MemoryUsageAbove*100))
		recommendations = append(recommendations, "Consider memory optimization or cache eviction tuning")
		if status == "healthy" {
			status = "degraded"
		}
	}
	
	cm.healthStatus = &HealthStatus{
		Status:          status,
		LastCheck:       time.Now(),
		ResponseTime:    time.Since(start),
		RedisConnected:  redisConnected,
		MemoryPressure:  metrics.MemoryUtilization > 0.80,
		ErrorRateHigh:   metrics.ErrorRate > 0.03,
		LatencyHigh:     metrics.AverageLatency > 50*time.Millisecond,
		Issues:          issues,
		Recommendations: recommendations,
	}
	
	if len(issues) > 0 {
		cm.logger.Warn("Cache health check found issues",
			zap.String("status", status),
			zap.Strings("issues", issues),
			zap.Strings("recommendations", recommendations),
		)
	} else {
		cm.logger.Debug("Cache health check passed", zap.String("status", status))
	}
}

func (cm *CacheMonitor) checkAlerts() {
	// This method is called frequently, so we do lightweight checks only
	// Heavy alerting logic is handled in the health check loop
	
	metrics := *cm.metrics
	
	// Critical alerts only
	if metrics.ErrorRate > 0.20 { // 20% error rate is critical
		alert := Alert{
			Type:      "error_rate_critical",
			Severity:  "critical",
			Message:   fmt.Sprintf("Cache error rate has reached %.2f%%", metrics.ErrorRate*100),
			Metrics:   map[string]interface{}{"error_rate": metrics.ErrorRate},
			Timestamp: time.Now(),
		}
		cm.sendAlert(alert)
	}
	
	if metrics.AverageLatency > 5*time.Second { // 5 second latency is critical
		alert := Alert{
			Type:      "latency_critical",
			Severity:  "critical",
			Message:   fmt.Sprintf("Cache average latency has reached %v", metrics.AverageLatency),
			Metrics:   map[string]interface{}{"average_latency": metrics.AverageLatency.String()},
			Timestamp: time.Now(),
		}
		cm.sendAlert(alert)
	}
}

func (cm *CacheMonitor) sendAlert(alert Alert) {
	for _, callback := range cm.alertCallbacks {
		go callback(alert)
	}
	
	cm.logger.Warn("Cache alert triggered",
		zap.String("type", alert.Type),
		zap.String("severity", alert.Severity),
		zap.String("message", alert.Message),
	)
}

func (cm *CacheMonitor) recordAnalyticsEvent(event AnalyticsEvent) {
	cm.analyticsMutex.Lock()
	defer cm.analyticsMutex.Unlock()
	
	cm.analyticsBuffer = append(cm.analyticsBuffer, event)
	
	// Flush buffer if it's getting full
	if len(cm.analyticsBuffer) >= 900 {
		go cm.flushAnalytics()
	}
}

func (cm *CacheMonitor) analyticsFlushLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		cm.flushAnalytics()
	}
}

func (cm *CacheMonitor) flushAnalytics() {
	cm.analyticsMutex.Lock()
	events := make([]AnalyticsEvent, len(cm.analyticsBuffer))
	copy(events, cm.analyticsBuffer)
	cm.analyticsBuffer = cm.analyticsBuffer[:0]
	cm.analyticsMutex.Unlock()
	
	if len(events) == 0 {
		return
	}
	
	// In production, this would write to a time-series database or analytics service
	cm.logger.Debug("Flushing analytics events", zap.Int("count", len(events)))
}

func (cm *CacheMonitor) updateAverage(current, new time.Duration) time.Duration {
	alpha := 0.1
	return time.Duration(float64(current)*(1-alpha) + float64(new)*alpha)
}

func (cm *CacheMonitor) getServiceStatus(metrics *ServiceMetrics) string {
	hitRate := float64(0)
	if metrics.RequestCount > 0 {
		hitRate = float64(metrics.HitCount) / float64(metrics.RequestCount)
	}
	
	if hitRate > 0.90 && metrics.AverageLatency < 50*time.Millisecond {
		return "excellent"
	} else if hitRate > 0.80 && metrics.AverageLatency < 100*time.Millisecond {
		return "good"
	} else if hitRate > 0.60 {
		return "fair"
	} else {
		return "poor"
	}
}

func (cm *CacheMonitor) generateRecommendations(metrics *ServiceMetrics) []string {
	var recommendations []string
	
	hitRate := float64(0)
	if metrics.RequestCount > 0 {
		hitRate = float64(metrics.HitCount) / float64(metrics.RequestCount)
	}
	
	if hitRate < 0.80 {
		recommendations = append(recommendations, "Consider implementing cache warming strategies")
		recommendations = append(recommendations, "Review TTL settings for better hit rates")
	}
	
	if metrics.AverageLatency > 100*time.Millisecond {
		recommendations = append(recommendations, "Optimize cache key structure for better performance")
		recommendations = append(recommendations, "Consider promoting frequently accessed data to hot cache")
	}
	
	if metrics.ErrorCount > 0 {
		recommendations = append(recommendations, "Investigate and resolve cache errors")
	}
	
	return recommendations
}