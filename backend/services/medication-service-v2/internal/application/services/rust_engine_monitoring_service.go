package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"medication-service-v2/internal/infrastructure/clients"
	"medication-service-v2/config"
)

// RustEngineMonitoringService monitors and manages the Rust Clinical Engine performance and health
type RustEngineMonitoringService struct {
	rustClient         *clients.RustClinicalEngineClient
	metricsService     *MetricsService
	config             *RustEngineMonitoringConfig
	logger             *zap.Logger

	// Performance tracking
	operationMetrics   sync.Map // map[operation_type]*OperationMetrics
	circuitBreaker     *CircuitBreaker
	healthCache        *HealthCache
	performanceAlerts  []PerformanceAlert

	// Monitoring state
	isMonitoring       bool
	monitoringMutex    sync.RWMutex
	lastHealthCheck    time.Time
	healthCheckTicker  *time.Ticker
}

// RustEngineMonitoringConfig holds configuration for engine monitoring
type RustEngineMonitoringConfig struct {
	HealthCheckInterval      time.Duration `json:"health_check_interval" mapstructure:"health_check_interval"`
	MetricsCollectionInterval time.Duration `json:"metrics_collection_interval" mapstructure:"metrics_collection_interval"`
	PerformanceAlertThreshold map[string]time.Duration `json:"performance_alert_threshold" mapstructure:"performance_alert_threshold"`
	CircuitBreakerEnabled    bool          `json:"circuit_breaker_enabled" mapstructure:"circuit_breaker_enabled"`
	HealthCacheExpiration    time.Duration `json:"health_cache_expiration" mapstructure:"health_cache_expiration"`
	AlertOnFailureCount      int           `json:"alert_on_failure_count" mapstructure:"alert_on_failure_count"`
	EnablePerformanceLogging bool          `json:"enable_performance_logging" mapstructure:"enable_performance_logging"`
}

// OperationMetrics tracks metrics for specific operation types
type OperationMetrics struct {
	OperationType           string        `json:"operation_type"`
	TotalRequests          int64         `json:"total_requests"`
	SuccessfulRequests     int64         `json:"successful_requests"`
	FailedRequests         int64         `json:"failed_requests"`
	AverageResponseTime    time.Duration `json:"average_response_time"`
	MinResponseTime        time.Duration `json:"min_response_time"`
	MaxResponseTime        time.Duration `json:"max_response_time"`
	P95ResponseTime        time.Duration `json:"p95_response_time"`
	LastRequestTime        time.Time     `json:"last_request_time"`
	LastSuccessTime        time.Time     `json:"last_success_time"`
	LastFailureTime        time.Time     `json:"last_failure_time"`
	PerformanceTargetMisses int64        `json:"performance_target_misses"`
	
	// Response time history for percentile calculations
	responseTimes          []time.Duration
	responseTimeMutex      sync.RWMutex
}

// CircuitBreaker implements circuit breaker pattern for Rust engine
type CircuitBreaker struct {
	state              CircuitBreakerState `json:"state"`
	failureCount       int                 `json:"failure_count"`
	lastFailureTime    time.Time           `json:"last_failure_time"`
	nextRetryTime      time.Time           `json:"next_retry_time"`
	successCount       int                 `json:"success_count"`
	config             *config.CircuitBreakerConfig
	mutex              sync.RWMutex
}

// CircuitBreakerState represents the state of the circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// HealthCache caches health check results to reduce overhead
type HealthCache struct {
	isHealthy          bool
	lastHealthCheck    time.Time
	lastHealthResult   map[string]interface{}
	expirationTime     time.Duration
	mutex              sync.RWMutex
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	AlertID           uuid.UUID `json:"alert_id"`
	AlertType         string    `json:"alert_type"`
	OperationType     string    `json:"operation_type"`
	Severity          string    `json:"severity"`
	Message           string    `json:"message"`
	ActualPerformance time.Duration `json:"actual_performance"`
	ExpectedPerformance time.Duration `json:"expected_performance"`
	Timestamp         time.Time `json:"timestamp"`
	Acknowledged      bool      `json:"acknowledged"`
}

// NewRustEngineMonitoringService creates a new monitoring service
func NewRustEngineMonitoringService(
	rustClient *clients.RustClinicalEngineClient,
	metricsService *MetricsService,
	config *RustEngineMonitoringConfig,
	logger *zap.Logger,
) *RustEngineMonitoringService {
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.MetricsCollectionInterval == 0 {
		config.MetricsCollectionInterval = 60 * time.Second
	}
	if config.HealthCacheExpiration == 0 {
		config.HealthCacheExpiration = 10 * time.Second
	}
	if config.AlertOnFailureCount == 0 {
		config.AlertOnFailureCount = 5
	}

	// Initialize default performance alert thresholds
	if config.PerformanceAlertThreshold == nil {
		config.PerformanceAlertThreshold = map[string]time.Duration{
			"drug_interactions":  100 * time.Millisecond,
			"dosage_calculation": 75 * time.Millisecond,
			"safety_validation":  150 * time.Millisecond,
			"rule_evaluation":    100 * time.Millisecond,
		}
	}

	service := &RustEngineMonitoringService{
		rustClient:     rustClient,
		metricsService: metricsService,
		config:        config,
		logger:        logger.Named("rust-engine-monitoring"),
		healthCache: &HealthCache{
			expirationTime: config.HealthCacheExpiration,
		},
		performanceAlerts: make([]PerformanceAlert, 0),
	}

	// Initialize circuit breaker if enabled
	if config.CircuitBreakerEnabled {
		service.circuitBreaker = &CircuitBreaker{
			state: CircuitBreakerClosed,
			config: &config.CircuitBreakerConfig{
				Enabled:          true,
				FailureThreshold: 5,
				SuccessThreshold: 3,
				Timeout:          30 * time.Second,
				MaxRequests:      10,
				Interval:         10 * time.Second,
			},
		}
	}

	return service
}

// StartMonitoring begins monitoring the Rust engine
func (s *RustEngineMonitoringService) StartMonitoring(ctx context.Context) error {
	s.monitoringMutex.Lock()
	defer s.monitoringMutex.Unlock()

	if s.isMonitoring {
		return fmt.Errorf("monitoring is already running")
	}

	s.logger.Info("Starting Rust engine monitoring")

	// Start health check ticker
	s.healthCheckTicker = time.NewTicker(s.config.HealthCheckInterval)
	s.isMonitoring = true

	// Start monitoring goroutines
	go s.healthCheckLoop(ctx)
	go s.metricsCollectionLoop(ctx)
	go s.performanceAlertLoop(ctx)

	s.logger.Info("Rust engine monitoring started successfully")
	return nil
}

// StopMonitoring stops monitoring the Rust engine
func (s *RustEngineMonitoringService) StopMonitoring() error {
	s.monitoringMutex.Lock()
	defer s.monitoringMutex.Unlock()

	if !s.isMonitoring {
		return fmt.Errorf("monitoring is not running")
	}

	s.logger.Info("Stopping Rust engine monitoring")

	if s.healthCheckTicker != nil {
		s.healthCheckTicker.Stop()
	}
	s.isMonitoring = false

	s.logger.Info("Rust engine monitoring stopped")
	return nil
}

// RecordOperation records metrics for a Rust engine operation
func (s *RustEngineMonitoringService) RecordOperation(operationType string, duration time.Duration, success bool, performanceTarget time.Duration) {
	// Load or create operation metrics
	metricsInterface, _ := s.operationMetrics.LoadOrStore(operationType, &OperationMetrics{
		OperationType:   operationType,
		responseTimes:   make([]time.Duration, 0, 1000), // Keep last 1000 response times
		MinResponseTime: duration,
		MaxResponseTime: duration,
	})

	metrics := metricsInterface.(*OperationMetrics)
	
	// Update metrics
	metrics.responseTimeMutex.Lock()
	defer metrics.responseTimeMutex.Unlock()

	metrics.TotalRequests++
	metrics.LastRequestTime = time.Now()

	if success {
		metrics.SuccessfulRequests++
		metrics.LastSuccessTime = time.Now()
		
		// Update circuit breaker on success
		if s.circuitBreaker != nil {
			s.circuitBreaker.recordSuccess()
		}
	} else {
		metrics.FailedRequests++
		metrics.LastFailureTime = time.Now()
		
		// Update circuit breaker on failure
		if s.circuitBreaker != nil {
			s.circuitBreaker.recordFailure()
		}
	}

	// Update response time statistics
	if duration < metrics.MinResponseTime {
		metrics.MinResponseTime = duration
	}
	if duration > metrics.MaxResponseTime {
		metrics.MaxResponseTime = duration
	}

	// Add to response times history (keep last 1000)
	if len(metrics.responseTimes) >= 1000 {
		metrics.responseTimes = metrics.responseTimes[1:]
	}
	metrics.responseTimes = append(metrics.responseTimes, duration)

	// Calculate average response time
	total := time.Duration(0)
	for _, rt := range metrics.responseTimes {
		total += rt
	}
	metrics.AverageResponseTime = total / time.Duration(len(metrics.responseTimes))

	// Calculate P95 response time
	metrics.P95ResponseTime = s.calculatePercentile(metrics.responseTimes, 0.95)

	// Check for performance target miss
	if performanceTarget > 0 && duration > performanceTarget {
		metrics.PerformanceTargetMisses++
		
		// Generate performance alert
		s.generatePerformanceAlert(operationType, duration, performanceTarget)
	}

	// Log performance if enabled
	if s.config.EnablePerformanceLogging {
		s.logger.Debug("Rust engine operation recorded",
			zap.String("operation_type", operationType),
			zap.Duration("duration", duration),
			zap.Bool("success", success),
			zap.Duration("performance_target", performanceTarget),
			zap.Bool("target_met", duration <= performanceTarget))
	}

	// Record metrics to metrics service
	if s.metricsService != nil {
		s.metricsService.RecordRustEngineOperation(operationType, duration, success)
	}
}

// IsHealthy checks if the Rust engine is healthy (with caching)
func (s *RustEngineMonitoringService) IsHealthy(ctx context.Context) (bool, map[string]interface{}) {
	s.healthCache.mutex.RLock()
	
	// Return cached result if still valid
	if time.Since(s.healthCache.lastHealthCheck) < s.healthCache.expirationTime {
		isHealthy := s.healthCache.isHealthy
		result := s.healthCache.lastHealthResult
		s.healthCache.mutex.RUnlock()
		return isHealthy, result
	}
	
	s.healthCache.mutex.RUnlock()

	// Perform health check
	return s.performHealthCheck(ctx)
}

// GetOperationMetrics returns metrics for all operations
func (s *RustEngineMonitoringService) GetOperationMetrics() map[string]*OperationMetrics {
	metrics := make(map[string]*OperationMetrics)
	
	s.operationMetrics.Range(func(key, value interface{}) bool {
		operationType := key.(string)
		operationMetrics := value.(*OperationMetrics)
		
		// Create a copy to avoid race conditions
		operationMetrics.responseTimeMutex.RLock()
		metricsCopy := &OperationMetrics{
			OperationType:           operationMetrics.OperationType,
			TotalRequests:          operationMetrics.TotalRequests,
			SuccessfulRequests:     operationMetrics.SuccessfulRequests,
			FailedRequests:         operationMetrics.FailedRequests,
			AverageResponseTime:    operationMetrics.AverageResponseTime,
			MinResponseTime:        operationMetrics.MinResponseTime,
			MaxResponseTime:        operationMetrics.MaxResponseTime,
			P95ResponseTime:        operationMetrics.P95ResponseTime,
			LastRequestTime:        operationMetrics.LastRequestTime,
			LastSuccessTime:        operationMetrics.LastSuccessTime,
			LastFailureTime:        operationMetrics.LastFailureTime,
			PerformanceTargetMisses: operationMetrics.PerformanceTargetMisses,
		}
		operationMetrics.responseTimeMutex.RUnlock()
		
		metrics[operationType] = metricsCopy
		return true
	})
	
	return metrics
}

// GetPerformanceAlerts returns current performance alerts
func (s *RustEngineMonitoringService) GetPerformanceAlerts() []PerformanceAlert {
	s.monitoringMutex.RLock()
	defer s.monitoringMutex.RUnlock()
	
	// Return a copy of alerts
	alerts := make([]PerformanceAlert, len(s.performanceAlerts))
	copy(alerts, s.performanceAlerts)
	
	return alerts
}

// AcknowledgeAlert acknowledges a performance alert
func (s *RustEngineMonitoringService) AcknowledgeAlert(alertID uuid.UUID) error {
	s.monitoringMutex.Lock()
	defer s.monitoringMutex.Unlock()
	
	for i := range s.performanceAlerts {
		if s.performanceAlerts[i].AlertID == alertID {
			s.performanceAlerts[i].Acknowledged = true
			s.logger.Info("Performance alert acknowledged", zap.String("alert_id", alertID.String()))
			return nil
		}
	}
	
	return fmt.Errorf("alert not found: %s", alertID.String())
}

// CanExecuteOperation checks if operation can be executed based on circuit breaker state
func (s *RustEngineMonitoringService) CanExecuteOperation() bool {
	if s.circuitBreaker == nil {
		return true
	}
	
	return s.circuitBreaker.canExecute()
}

// Private methods

func (s *RustEngineMonitoringService) healthCheckLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.healthCheckTicker.C:
			if !s.isMonitoring {
				return
			}
			
			isHealthy, result := s.performHealthCheck(ctx)
			s.logger.Debug("Health check completed",
				zap.Bool("healthy", isHealthy),
				zap.Any("result", result))
		}
	}
}

func (s *RustEngineMonitoringService) metricsCollectionLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.MetricsCollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.isMonitoring {
				return
			}
			
			s.collectAndReportMetrics(ctx)
		}
	}
}

func (s *RustEngineMonitoringService) performanceAlertLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Check for stale alerts every 5 minutes
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !s.isMonitoring {
				return
			}
			
			s.cleanupStaleAlerts()
		}
	}
}

func (s *RustEngineMonitoringService) performHealthCheck(ctx context.Context) (bool, map[string]interface{}) {
	s.healthCache.mutex.Lock()
	defer s.healthCache.mutex.Unlock()
	
	startTime := time.Now()
	result, err := s.rustClient.HealthCheck(ctx)
	duration := time.Since(startTime)
	
	isHealthy := err == nil
	if result == nil {
		result = make(map[string]interface{})
	}
	
	// Add monitoring metadata
	result["health_check_duration"] = duration
	result["health_check_timestamp"] = startTime
	result["health_check_error"] = ""
	
	if err != nil {
		result["health_check_error"] = err.Error()
		s.logger.Warn("Rust engine health check failed",
			zap.Error(err),
			zap.Duration("duration", duration))
	}
	
	// Update cache
	s.healthCache.isHealthy = isHealthy
	s.healthCache.lastHealthCheck = time.Now()
	s.healthCache.lastHealthResult = result
	
	// Record health check metrics
	if s.metricsService != nil {
		s.metricsService.RecordHealthCheck("rust_engine", isHealthy, duration)
	}
	
	return isHealthy, result
}

func (s *RustEngineMonitoringService) collectAndReportMetrics(ctx context.Context) {
	// Collect metrics from Rust engine
	engineMetrics, err := s.rustClient.GetMetrics(ctx)
	if err != nil {
		s.logger.Warn("Failed to collect Rust engine metrics", zap.Error(err))
		return
	}
	
	// Combine with local metrics
	localMetrics := s.GetOperationMetrics()
	
	combinedMetrics := map[string]interface{}{
		"engine_metrics":    engineMetrics,
		"operation_metrics": localMetrics,
		"circuit_breaker":   s.getCircuitBreakerStatus(),
		"health_cache":      s.getHealthCacheStatus(),
		"alerts":           s.GetPerformanceAlerts(),
		"collection_time":   time.Now(),
	}
	
	// Report to metrics service
	if s.metricsService != nil {
		s.metricsService.RecordComplexMetric("rust_engine_comprehensive", combinedMetrics)
	}
	
	s.logger.Debug("Metrics collected and reported",
		zap.Int("operation_types", len(localMetrics)),
		zap.Int("alert_count", len(s.performanceAlerts)))
}

func (s *RustEngineMonitoringService) generatePerformanceAlert(operationType string, actualTime, expectedTime time.Duration) {
	alert := PerformanceAlert{
		AlertID:             uuid.New(),
		AlertType:           "performance_degradation",
		OperationType:       operationType,
		Severity:            s.determineSeverity(actualTime, expectedTime),
		Message:             fmt.Sprintf("Performance target missed for %s: %v > %v", operationType, actualTime, expectedTime),
		ActualPerformance:   actualTime,
		ExpectedPerformance: expectedTime,
		Timestamp:           time.Now(),
		Acknowledged:        false,
	}
	
	s.monitoringMutex.Lock()
	s.performanceAlerts = append(s.performanceAlerts, alert)
	
	// Keep only last 100 alerts
	if len(s.performanceAlerts) > 100 {
		s.performanceAlerts = s.performanceAlerts[len(s.performanceAlerts)-100:]
	}
	s.monitoringMutex.Unlock()
	
	s.logger.Warn("Performance alert generated",
		zap.String("alert_id", alert.AlertID.String()),
		zap.String("operation_type", operationType),
		zap.Duration("actual_time", actualTime),
		zap.Duration("expected_time", expectedTime),
		zap.String("severity", alert.Severity))
}

func (s *RustEngineMonitoringService) determineSeverity(actualTime, expectedTime time.Duration) string {
	ratio := float64(actualTime) / float64(expectedTime)
	
	switch {
	case ratio >= 3.0:
		return "critical"
	case ratio >= 2.0:
		return "high"
	case ratio >= 1.5:
		return "medium"
	default:
		return "low"
	}
}

func (s *RustEngineMonitoringService) cleanupStaleAlerts() {
	s.monitoringMutex.Lock()
	defer s.monitoringMutex.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour) // Remove alerts older than 24 hours
	validAlerts := make([]PerformanceAlert, 0)
	
	for _, alert := range s.performanceAlerts {
		if alert.Timestamp.After(cutoff) {
			validAlerts = append(validAlerts, alert)
		}
	}
	
	removedCount := len(s.performanceAlerts) - len(validAlerts)
	s.performanceAlerts = validAlerts
	
	if removedCount > 0 {
		s.logger.Debug("Cleaned up stale alerts", zap.Int("removed_count", removedCount))
	}
}

func (s *RustEngineMonitoringService) calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	
	// Create a copy and sort it
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	
	// Simple insertion sort for small arrays
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1
		for j >= 0 && sorted[j] > key {
			sorted[j+1] = sorted[j]
			j--
		}
		sorted[j+1] = key
	}
	
	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

func (s *RustEngineMonitoringService) getCircuitBreakerStatus() map[string]interface{} {
	if s.circuitBreaker == nil {
		return map[string]interface{}{
			"enabled": false,
		}
	}
	
	s.circuitBreaker.mutex.RLock()
	defer s.circuitBreaker.mutex.RUnlock()
	
	return map[string]interface{}{
		"enabled":         true,
		"state":          s.circuitBreaker.state,
		"failure_count":  s.circuitBreaker.failureCount,
		"success_count":  s.circuitBreaker.successCount,
		"last_failure":   s.circuitBreaker.lastFailureTime,
		"next_retry":     s.circuitBreaker.nextRetryTime,
	}
}

func (s *RustEngineMonitoringService) getHealthCacheStatus() map[string]interface{} {
	s.healthCache.mutex.RLock()
	defer s.healthCache.mutex.RUnlock()
	
	return map[string]interface{}{
		"is_healthy":       s.healthCache.isHealthy,
		"last_check":       s.healthCache.lastHealthCheck,
		"expiration_time":  s.healthCache.expirationTime,
		"cache_valid":      time.Since(s.healthCache.lastHealthCheck) < s.healthCache.expirationTime,
	}
}

// Circuit breaker methods

func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		if time.Now().After(cb.nextRetryTime) {
			// Transition to half-open state
			cb.state = CircuitBreakerHalfOpen
			return true
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.successCount++
	
	switch cb.state {
	case CircuitBreakerHalfOpen:
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = CircuitBreakerClosed
			cb.failureCount = 0
		}
	case CircuitBreakerClosed:
		cb.failureCount = 0 // Reset failure count on success
	}
}

func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	if cb.failureCount >= cb.config.FailureThreshold {
		cb.state = CircuitBreakerOpen
		cb.nextRetryTime = time.Now().Add(cb.config.Timeout)
		cb.successCount = 0
	}
}