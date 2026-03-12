package performance

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/metrics"
)

// Monitor provides comprehensive performance monitoring and alerting
type Monitor struct {
	config         *config.SnapshotConfig
	logger         *logger.Logger
	metricsCollector *metrics.SnapshotMetricsCollector
	
	// Performance tracking
	requestLatencies   *LatencyTracker
	throughputTracker  *ThroughputTracker
	resourceTracker    *ResourceTracker
	slaTracker        *SLATracker
	alertManager      *AlertManager
	
	// Real-time metrics
	currentMetrics    *RealTimeMetrics
	performanceScore  *PerformanceScore
	
	// Monitoring state
	isMonitoring      bool
	monitoringCtx     context.Context
	monitoringCancel  context.CancelFunc
	mu                sync.RWMutex
}

// LatencyTracker tracks request latency metrics with statistical analysis
type LatencyTracker struct {
	samples       []time.Duration
	sortedSamples []time.Duration
	capacity      int
	writeIndex    int
	isSorted      bool
	mu            sync.RWMutex
	
	// Cached statistics
	cachedP95      time.Duration
	cachedP99      time.Duration
	cachedAverage  time.Duration
	cacheValid     bool
}

// ThroughputTracker tracks request throughput and patterns
type ThroughputTracker struct {
	requestCounts    []int64
	timestamps       []time.Time
	windowSize       time.Duration
	bucketCount      int
	currentBucket    int
	totalRequests    int64
	mu               sync.RWMutex
}

// ResourceTracker monitors system resource utilization
type ResourceTracker struct {
	memoryUsage      []MemorySnapshot
	cpuUsage         []CPUSnapshot
	connectionMetrics []ConnectionSnapshot
	gcMetrics        []GCSnapshot
	capacity         int
	mu               sync.RWMutex
}

// SLATracker monitors SLA compliance and violations
type SLATracker struct {
	targets          SLATargets
	violations       []SLAViolation
	complianceHistory []ComplianceSnapshot
	currentCompliance ComplianceScore
	mu               sync.RWMutex
}

// AlertManager handles performance alerts and notifications
type AlertManager struct {
	alerts          []PerformanceAlert
	rules           []AlertRule
	suppressions    map[string]time.Time
	logger          *logger.Logger
	mu              sync.RWMutex
}

// Data structures for metrics
type RealTimeMetrics struct {
	Timestamp         time.Time     `json:"timestamp"`
	P95Latency        time.Duration `json:"p95_latency"`
	P99Latency        time.Duration `json:"p99_latency"`
	AverageLatency    time.Duration `json:"average_latency"`
	ThroughputQPS     float64       `json:"throughput_qps"`
	SLACompliance     float64       `json:"sla_compliance"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	MemoryPressure    float64       `json:"memory_pressure"`
	CPUUtilization    float64       `json:"cpu_utilization"`
	ActiveConnections int           `json:"active_connections"`
	ErrorRate         float64       `json:"error_rate"`
}

type PerformanceScore struct {
	Overall           int                    `json:"overall"`
	Latency           int                    `json:"latency"`
	Throughput        int                    `json:"throughput"`
	Reliability       int                    `json:"reliability"`
	Efficiency        int                    `json:"efficiency"`
	Breakdown         map[string]interface{} `json:"breakdown"`
	CalculatedAt      time.Time             `json:"calculated_at"`
}

type MemorySnapshot struct {
	Timestamp     time.Time `json:"timestamp"`
	HeapAlloc     uint64    `json:"heap_alloc"`
	HeapSys       uint64    `json:"heap_sys"`
	HeapInuse     uint64    `json:"heap_inuse"`
	StackInuse    uint64    `json:"stack_inuse"`
	TotalAlloc    uint64    `json:"total_alloc"`
	NumGC         uint32    `json:"num_gc"`
	PauseTotalNs  uint64    `json:"pause_total_ns"`
}

type CPUSnapshot struct {
	Timestamp       time.Time `json:"timestamp"`
	UserPercent     float64   `json:"user_percent"`
	SystemPercent   float64   `json:"system_percent"`
	IdlePercent     float64   `json:"idle_percent"`
	TotalPercent    float64   `json:"total_percent"`
	Goroutines      int       `json:"goroutines"`
}

type ConnectionSnapshot struct {
	Timestamp        time.Time `json:"timestamp"`
	ActiveConnections int      `json:"active_connections"`
	IdleConnections   int      `json:"idle_connections"`
	WaitingRequests   int      `json:"waiting_requests"`
	PoolUtilization   float64  `json:"pool_utilization"`
}

type GCSnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	NumGC        uint32    `json:"num_gc"`
	PauseNs      uint64    `json:"pause_ns"`
	PausePercent float64   `json:"pause_percent"`
}

type SLATargets struct {
	P95LatencyTarget    time.Duration `json:"p95_latency_target"`
	P99LatencyTarget    time.Duration `json:"p99_latency_target"`
	AvailabilityTarget  float64       `json:"availability_target"`
	ErrorRateTarget     float64       `json:"error_rate_target"`
	CacheHitRateTarget  float64       `json:"cache_hit_rate_target"`
	ThroughputTarget    float64       `json:"throughput_target"`
}

type SLAViolation struct {
	Timestamp    time.Time `json:"timestamp"`
	Metric       string    `json:"metric"`
	Target       float64   `json:"target"`
	Actual       float64   `json:"actual"`
	Severity     string    `json:"severity"`
	Duration     time.Duration `json:"duration"`
	Description  string    `json:"description"`
}

type ComplianceSnapshot struct {
	Timestamp        time.Time `json:"timestamp"`
	LatencyCompliance float64  `json:"latency_compliance"`
	ErrorRateCompliance float64 `json:"error_rate_compliance"`
	CacheHitCompliance  float64 `json:"cache_hit_compliance"`
	OverallCompliance   float64 `json:"overall_compliance"`
}

type ComplianceScore struct {
	Score            float64   `json:"score"`
	Grade            string    `json:"grade"`
	Trend            string    `json:"trend"`
	LastUpdated      time.Time `json:"last_updated"`
	PerformanceLevel string    `json:"performance_level"`
}

type PerformanceAlert struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    string                 `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Metric      string                 `json:"metric"`
	Threshold   float64                `json:"threshold"`
	ActualValue float64                `json:"actual_value"`
	Duration    time.Duration          `json:"duration"`
	Tags        map[string]string      `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type AlertRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Metric      string        `json:"metric"`
	Condition   string        `json:"condition"`
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Severity    string        `json:"severity"`
	Enabled     bool          `json:"enabled"`
	Suppression time.Duration `json:"suppression"`
}

// NewMonitor creates a new performance monitor
func NewMonitor(cfg *config.SnapshotConfig, logger *logger.Logger, metricsCollector *metrics.SnapshotMetricsCollector) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	monitor := &Monitor{
		config:           cfg,
		logger:           logger,
		metricsCollector: metricsCollector,
		requestLatencies: NewLatencyTracker(10000), // Track last 10k requests
		throughputTracker: NewThroughputTracker(time.Minute, 60), // 1 hour with 1-minute buckets
		resourceTracker:  NewResourceTracker(3600), // Track last hour
		slaTracker:      NewSLATracker(),
		alertManager:    NewAlertManager(logger),
		currentMetrics:  &RealTimeMetrics{},
		performanceScore: &PerformanceScore{},
		monitoringCtx:   ctx,
		monitoringCancel: cancel,
	}
	
	monitor.initializeAlertRules()
	
	return monitor
}

// StartMonitoring starts the performance monitoring process
func (m *Monitor) StartMonitoring() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.isMonitoring {
		return fmt.Errorf("monitoring already started")
	}
	
	m.isMonitoring = true
	
	// Start monitoring goroutines
	go m.monitorLatency()
	go m.monitorThroughput()
	go m.monitorResources()
	go m.monitorSLA()
	go m.processAlerts()
	go m.calculatePerformanceScore()
	
	m.logger.Info("Performance monitoring started",
		zap.Duration("p95_target", 200*time.Millisecond),
		zap.Float64("cache_hit_target", 85.0),
	)
	
	return nil
}

// StopMonitoring stops the performance monitoring process
func (m *Monitor) StopMonitoring() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.isMonitoring {
		return fmt.Errorf("monitoring not started")
	}
	
	m.isMonitoring = false
	m.monitoringCancel()
	
	m.logger.Info("Performance monitoring stopped")
	return nil
}

// RecordRequest records request metrics for performance tracking
func (m *Monitor) RecordRequest(latency time.Duration, success bool) {
	// Record latency
	m.requestLatencies.Record(latency)
	
	// Record throughput
	m.throughputTracker.RecordRequest()
	
	// Update SLA tracking
	m.slaTracker.RecordRequest(latency, success)
	
	// Update real-time metrics
	m.updateRealTimeMetrics()
}

// GetCurrentMetrics returns current performance metrics
func (m *Monitor) GetCurrentMetrics() *RealTimeMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *m.currentMetrics
	return &metrics
}

// GetPerformanceScore returns the current performance score
func (m *Monitor) GetPerformanceScore() *PerformanceScore {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	score := *m.performanceScore
	return &score
}

// GetPerformanceReport generates a comprehensive performance report
func (m *Monitor) GetPerformanceReport() map[string]interface{} {
	metrics := m.GetCurrentMetrics()
	score := m.GetPerformanceScore()
	slaStatus := m.slaTracker.GetComplianceStatus()
	recentAlerts := m.alertManager.GetRecentAlerts(24 * time.Hour)
	
	report := map[string]interface{}{
		"timestamp":        time.Now(),
		"report_version":   "3.0",
		"monitoring_status": m.isMonitoring,
		
		"current_metrics": metrics,
		"performance_score": score,
		"sla_compliance": slaStatus,
		
		"latency_analysis": map[string]interface{}{
			"p95":     metrics.P95Latency,
			"p99":     metrics.P99Latency,
			"average": metrics.AverageLatency,
			"target_compliance": fmt.Sprintf("%.1f%%", m.calculateLatencyCompliance()),
		},
		
		"throughput_analysis": map[string]interface{}{
			"current_qps":     metrics.ThroughputQPS,
			"peak_qps":        m.throughputTracker.GetPeakThroughput(),
			"average_qps":     m.throughputTracker.GetAverageThroughput(),
		},
		
		"resource_utilization": map[string]interface{}{
			"memory_pressure": metrics.MemoryPressure,
			"cpu_utilization": metrics.CPUUtilization,
			"connection_health": m.getConnectionHealth(),
		},
		
		"cache_performance": map[string]interface{}{
			"hit_rate":        metrics.CacheHitRate,
			"target":          85.0,
			"compliance":      metrics.CacheHitRate >= 85.0,
		},
		
		"alerts": map[string]interface{}{
			"active_alerts":   m.alertManager.GetActiveAlertCount(),
			"recent_alerts":   len(recentAlerts),
			"critical_alerts": m.alertManager.GetCriticalAlertCount(),
		},
		
		"recommendations": m.generatePerformanceRecommendations(),
	}
	
	return report
}

// Monitoring goroutines
func (m *Monitor) monitorLatency() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			m.updateLatencyMetrics()
		}
	}
}

func (m *Monitor) monitorThroughput() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			m.updateThroughputMetrics()
		}
	}
}

func (m *Monitor) monitorResources() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			m.updateResourceMetrics()
		}
	}
}

func (m *Monitor) monitorSLA() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			m.updateSLAMetrics()
		}
	}
}

func (m *Monitor) processAlerts() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			m.evaluateAlertRules()
		}
	}
}

func (m *Monitor) calculatePerformanceScore() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.monitoringCtx.Done():
			return
		case <-ticker.C:
			m.updatePerformanceScore()
		}
	}
}

// Update methods
func (m *Monitor) updateRealTimeMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.currentMetrics.Timestamp = time.Now()
	m.currentMetrics.P95Latency = m.requestLatencies.GetP95()
	m.currentMetrics.P99Latency = m.requestLatencies.GetP99()
	m.currentMetrics.AverageLatency = m.requestLatencies.GetAverage()
	m.currentMetrics.ThroughputQPS = m.throughputTracker.GetCurrentThroughput()
	m.currentMetrics.SLACompliance = m.slaTracker.GetCurrentCompliance()
}

func (m *Monitor) updateLatencyMetrics() {
	p95 := m.requestLatencies.GetP95()
	p99 := m.requestLatencies.GetP99()
	
	if m.metricsCollector != nil {
		m.metricsCollector.UpdatePerformanceLatency("snapshot_request", "standard", p95, p99)
	}
	
	// Check for latency violations
	if p95 > 200*time.Millisecond {
		m.alertManager.TriggerAlert(&PerformanceAlert{
			ID:          fmt.Sprintf("latency_violation_%d", time.Now().Unix()),
			Timestamp:   time.Now(),
			Severity:    "warning",
			Title:       "P95 Latency Target Exceeded",
			Description: fmt.Sprintf("P95 latency (%.1fms) exceeds target (200ms)", float64(p95.Nanoseconds())/1000000.0),
			Metric:      "p95_latency",
			Threshold:   200.0,
			ActualValue: float64(p95.Nanoseconds()) / 1000000.0,
		})
	}
}

func (m *Monitor) updateThroughputMetrics() {
	currentQPS := m.throughputTracker.GetCurrentThroughput()
	
	m.mu.Lock()
	m.currentMetrics.ThroughputQPS = currentQPS
	m.mu.Unlock()
}

func (m *Monitor) updateResourceMetrics() {
	// Collect memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	memSnapshot := MemorySnapshot{
		Timestamp:    time.Now(),
		HeapAlloc:    memStats.HeapAlloc,
		HeapSys:      memStats.HeapSys,
		HeapInuse:    memStats.HeapInuse,
		StackInuse:   memStats.StackInuse,
		TotalAlloc:   memStats.TotalAlloc,
		NumGC:        memStats.NumGC,
		PauseTotalNs: memStats.PauseTotalNs,
	}
	
	m.resourceTracker.RecordMemory(memSnapshot)
	
	// Calculate memory pressure
	memoryPressure := float64(memStats.HeapInuse) / float64(memStats.HeapSys)
	
	m.mu.Lock()
	m.currentMetrics.MemoryPressure = memoryPressure
	m.currentMetrics.CPUUtilization = float64(runtime.NumGoroutine()) / 1000.0 // Simplified CPU metric
	m.mu.Unlock()
	
	if m.metricsCollector != nil {
		memoryUsage := map[string]int64{
			"heap":  int64(memStats.HeapInuse),
			"stack": int64(memStats.StackInuse),
		}
		m.metricsCollector.UpdateMemoryMetrics(memoryPressure, memoryUsage)
	}
}

func (m *Monitor) updateSLAMetrics() {
	compliance := m.slaTracker.CalculateCompliance()
	
	m.mu.Lock()
	m.currentMetrics.SLACompliance = compliance
	m.mu.Unlock()
	
	if m.metricsCollector != nil {
		m.metricsCollector.UpdateSLACompliance("standard", "1h", compliance)
	}
}

func (m *Monitor) updatePerformanceScore() {
	score := m.calculateOverallPerformanceScore()
	
	m.mu.Lock()
	m.performanceScore = score
	m.mu.Unlock()
}

func (m *Monitor) calculateOverallPerformanceScore() *PerformanceScore {
	metrics := m.GetCurrentMetrics()
	
	// Calculate individual component scores (0-100)
	latencyScore := m.calculateLatencyScore(metrics.P95Latency, metrics.P99Latency)
	throughputScore := m.calculateThroughputScore(metrics.ThroughputQPS)
	reliabilityScore := m.calculateReliabilityScore(metrics.ErrorRate, metrics.SLACompliance)
	efficiencyScore := m.calculateEfficiencyScore(metrics.CacheHitRate, metrics.MemoryPressure)
	
	// Calculate weighted overall score
	overallScore := int(
		latencyScore*0.3 +
		throughputScore*0.2 +
		reliabilityScore*0.3 +
		efficiencyScore*0.2,
	)
	
	return &PerformanceScore{
		Overall:      overallScore,
		Latency:      latencyScore,
		Throughput:   throughputScore,
		Reliability:  reliabilityScore,
		Efficiency:   efficiencyScore,
		CalculatedAt: time.Now(),
		Breakdown: map[string]interface{}{
			"latency_weight":     30.0,
			"throughput_weight":  20.0,
			"reliability_weight": 30.0,
			"efficiency_weight":  20.0,
			"target_p95_ms":      200.0,
			"target_cache_hit":   85.0,
		},
	}
}

func (m *Monitor) calculateLatencyScore(p95, p99 time.Duration) int {
	p95Ms := float64(p95.Nanoseconds()) / 1000000.0
	p99Ms := float64(p99.Nanoseconds()) / 1000000.0
	
	// Score based on P95 latency target (200ms)
	p95Score := math.Max(0, 100-((p95Ms-200)/200)*100)
	
	// Score based on P99 latency target (500ms)
	p99Score := math.Max(0, 100-((p99Ms-500)/500)*100)
	
	// Weighted average (P95 is more important)
	return int(p95Score*0.7 + p99Score*0.3)
}

func (m *Monitor) calculateThroughputScore(qps float64) int {
	// Simple scoring based on throughput capacity
	// This would be customized based on expected load
	if qps >= 100 {
		return 100
	} else if qps >= 50 {
		return 80
	} else if qps >= 25 {
		return 60
	} else if qps >= 10 {
		return 40
	}
	return 20
}

func (m *Monitor) calculateReliabilityScore(errorRate, slaCompliance float64) int {
	errorScore := math.Max(0, 100-errorRate*10) // Each 1% error reduces score by 10
	slaScore := slaCompliance
	
	return int((errorScore + slaScore) / 2)
}

func (m *Monitor) calculateEfficiencyScore(cacheHitRate, memoryPressure float64) int {
	cacheScore := math.Min(100, cacheHitRate) // Cache hit rate as percentage
	memoryScore := math.Max(0, 100-memoryPressure*100) // Lower pressure = higher score
	
	return int((cacheScore + memoryScore) / 2)
}

func (m *Monitor) calculateLatencyCompliance() float64 {
	return m.slaTracker.GetLatencyCompliance()
}

func (m *Monitor) getConnectionHealth() float64 {
	// This would integrate with actual connection pool monitoring
	return 95.0 // Placeholder
}

func (m *Monitor) generatePerformanceRecommendations() []string {
	recommendations := []string{}
	metrics := m.GetCurrentMetrics()
	
	// Latency recommendations
	if metrics.P95Latency > 200*time.Millisecond {
		recommendations = append(recommendations, "P95 latency exceeds target. Consider cache optimization or query optimization.")
	}
	
	// Cache recommendations
	if metrics.CacheHitRate < 85.0 {
		recommendations = append(recommendations, "Cache hit rate below target. Consider cache warming or TTL adjustment.")
	}
	
	// Memory recommendations
	if metrics.MemoryPressure > 0.8 {
		recommendations = append(recommendations, "High memory pressure detected. Consider increasing heap size or implementing compression.")
	}
	
	// Throughput recommendations
	if metrics.ThroughputQPS > 0 && metrics.ThroughputQPS < 10 {
		recommendations = append(recommendations, "Low throughput detected. Consider connection pooling optimization or async processing.")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System performance is within acceptable parameters.")
	}
	
	return recommendations
}

func (m *Monitor) initializeAlertRules() {
	rules := []AlertRule{
		{
			ID:        "p95_latency",
			Name:      "P95 Latency Threshold",
			Metric:    "p95_latency",
			Condition: ">",
			Threshold: 200.0, // milliseconds
			Duration:  1 * time.Minute,
			Severity:  "warning",
			Enabled:   true,
		},
		{
			ID:        "cache_hit_rate",
			Name:      "Cache Hit Rate Threshold",
			Metric:    "cache_hit_rate",
			Condition: "<",
			Threshold: 85.0, // percentage
			Duration:  5 * time.Minute,
			Severity:  "warning",
			Enabled:   true,
		},
		{
			ID:        "memory_pressure",
			Name:      "Memory Pressure Threshold",
			Metric:    "memory_pressure",
			Condition: ">",
			Threshold: 0.9, // 90%
			Duration:  2 * time.Minute,
			Severity:  "critical",
			Enabled:   true,
		},
	}
	
	m.alertManager.SetRules(rules)
}

func (m *Monitor) evaluateAlertRules() {
	metrics := m.GetCurrentMetrics()
	m.alertManager.EvaluateRules(metrics)
}

// Helper constructors
func NewLatencyTracker(capacity int) *LatencyTracker {
	return &LatencyTracker{
		samples:       make([]time.Duration, capacity),
		sortedSamples: make([]time.Duration, 0, capacity),
		capacity:      capacity,
	}
}

func NewThroughputTracker(windowSize time.Duration, bucketCount int) *ThroughputTracker {
	return &ThroughputTracker{
		requestCounts: make([]int64, bucketCount),
		timestamps:    make([]time.Time, bucketCount),
		windowSize:    windowSize,
		bucketCount:   bucketCount,
	}
}

func NewResourceTracker(capacity int) *ResourceTracker {
	return &ResourceTracker{
		memoryUsage:       make([]MemorySnapshot, 0, capacity),
		cpuUsage:         make([]CPUSnapshot, 0, capacity),
		connectionMetrics: make([]ConnectionSnapshot, 0, capacity),
		gcMetrics:        make([]GCSnapshot, 0, capacity),
		capacity:         capacity,
	}
}

func NewSLATracker() *SLATracker {
	return &SLATracker{
		targets: SLATargets{
			P95LatencyTarget:   200 * time.Millisecond,
			P99LatencyTarget:   500 * time.Millisecond,
			AvailabilityTarget: 99.9,
			ErrorRateTarget:    1.0,
			CacheHitRateTarget: 85.0,
			ThroughputTarget:   100.0,
		},
		violations:       make([]SLAViolation, 0),
		complianceHistory: make([]ComplianceSnapshot, 0),
	}
}

func NewAlertManager(logger *logger.Logger) *AlertManager {
	return &AlertManager{
		alerts:       make([]PerformanceAlert, 0),
		rules:        make([]AlertRule, 0),
		suppressions: make(map[string]time.Time),
		logger:       logger,
	}
}

// LatencyTracker methods
func (lt *LatencyTracker) Record(latency time.Duration) {
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	lt.samples[lt.writeIndex] = latency
	lt.writeIndex = (lt.writeIndex + 1) % lt.capacity
	lt.isSorted = false
	lt.cacheValid = false
}

func (lt *LatencyTracker) GetP95() time.Duration {
	lt.mu.RLock()
	if lt.cacheValid {
		result := lt.cachedP95
		lt.mu.RUnlock()
		return result
	}
	lt.mu.RUnlock()
	
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	lt.ensureSorted()
	if len(lt.sortedSamples) == 0 {
		return 0
	}
	
	index := int(float64(len(lt.sortedSamples)) * 0.95)
	if index >= len(lt.sortedSamples) {
		index = len(lt.sortedSamples) - 1
	}
	
	lt.cachedP95 = lt.sortedSamples[index]
	lt.updateCache()
	return lt.cachedP95
}

func (lt *LatencyTracker) GetP99() time.Duration {
	lt.mu.RLock()
	if lt.cacheValid {
		result := lt.cachedP99
		lt.mu.RUnlock()
		return result
	}
	lt.mu.RUnlock()
	
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	lt.ensureSorted()
	if len(lt.sortedSamples) == 0 {
		return 0
	}
	
	index := int(float64(len(lt.sortedSamples)) * 0.99)
	if index >= len(lt.sortedSamples) {
		index = len(lt.sortedSamples) - 1
	}
	
	lt.cachedP99 = lt.sortedSamples[index]
	lt.updateCache()
	return lt.cachedP99
}

func (lt *LatencyTracker) GetAverage() time.Duration {
	lt.mu.RLock()
	if lt.cacheValid {
		result := lt.cachedAverage
		lt.mu.RUnlock()
		return result
	}
	lt.mu.RUnlock()
	
	lt.mu.Lock()
	defer lt.mu.Unlock()
	
	var total time.Duration
	count := 0
	
	for _, sample := range lt.samples {
		if sample > 0 {
			total += sample
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	lt.cachedAverage = total / time.Duration(count)
	lt.updateCache()
	return lt.cachedAverage
}

func (lt *LatencyTracker) ensureSorted() {
	if lt.isSorted {
		return
	}
	
	lt.sortedSamples = lt.sortedSamples[:0]
	for _, sample := range lt.samples {
		if sample > 0 {
			lt.sortedSamples = append(lt.sortedSamples, sample)
		}
	}
	
	sort.Slice(lt.sortedSamples, func(i, j int) bool {
		return lt.sortedSamples[i] < lt.sortedSamples[j]
	})
	
	lt.isSorted = true
}

func (lt *LatencyTracker) updateCache() {
	lt.cacheValid = true
}

// ThroughputTracker methods
func (tt *ThroughputTracker) RecordRequest() {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	
	now := time.Now()
	currentBucket := int(now.Unix()/int64(tt.windowSize.Seconds())) % tt.bucketCount
	
	if currentBucket != tt.currentBucket {
		// New time bucket, reset counter
		tt.requestCounts[currentBucket] = 0
		tt.timestamps[currentBucket] = now
		tt.currentBucket = currentBucket
	}
	
	tt.requestCounts[currentBucket]++
	tt.totalRequests++
}

func (tt *ThroughputTracker) GetCurrentThroughput() float64 {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	
	if tt.requestCounts[tt.currentBucket] == 0 {
		return 0
	}
	
	windowSeconds := tt.windowSize.Seconds()
	return float64(tt.requestCounts[tt.currentBucket]) / windowSeconds
}

func (tt *ThroughputTracker) GetPeakThroughput() float64 {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	
	var peak int64
	for _, count := range tt.requestCounts {
		if count > peak {
			peak = count
		}
	}
	
	return float64(peak) / tt.windowSize.Seconds()
}

func (tt *ThroughputTracker) GetAverageThroughput() float64 {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	
	var total int64
	for _, count := range tt.requestCounts {
		total += count
	}
	
	if total == 0 {
		return 0
	}
	
	totalWindowSeconds := float64(tt.bucketCount) * tt.windowSize.Seconds()
	return float64(total) / totalWindowSeconds
}

// ResourceTracker methods
func (rt *ResourceTracker) RecordMemory(snapshot MemorySnapshot) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	
	rt.memoryUsage = append(rt.memoryUsage, snapshot)
	
	// Keep only the most recent entries
	if len(rt.memoryUsage) > rt.capacity {
		rt.memoryUsage = rt.memoryUsage[1:]
	}
}

// SLATracker methods
func (st *SLATracker) RecordRequest(latency time.Duration, success bool) {
	// This would track individual request compliance
}

func (st *SLATracker) GetCurrentCompliance() float64 {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	return st.currentCompliance.Score
}

func (st *SLATracker) CalculateCompliance() float64 {
	// This would calculate compliance based on recent data
	return 95.0 // Placeholder
}

func (st *SLATracker) GetComplianceStatus() ComplianceScore {
	st.mu.RLock()
	defer st.mu.RUnlock()
	
	return st.currentCompliance
}

func (st *SLATracker) GetLatencyCompliance() float64 {
	// This would calculate latency-specific compliance
	return 90.0 // Placeholder
}

// AlertManager methods
func (am *AlertManager) SetRules(rules []AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.rules = rules
}

func (am *AlertManager) TriggerAlert(alert *PerformanceAlert) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	// Check if alert is suppressed
	if suppressedUntil, exists := am.suppressions[alert.ID]; exists {
		if time.Now().Before(suppressedUntil) {
			return
		}
	}
	
	am.alerts = append(am.alerts, *alert)
	
	am.logger.Warn("Performance alert triggered",
		zap.String("id", alert.ID),
		zap.String("severity", alert.Severity),
		zap.String("title", alert.Title),
		zap.String("description", alert.Description),
	)
	
	// Set suppression
	am.suppressions[alert.ID] = time.Now().Add(5 * time.Minute)
}

func (am *AlertManager) EvaluateRules(metrics *RealTimeMetrics) {
	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}
		
		violation := am.evaluateRule(rule, metrics)
		if violation {
			alert := &PerformanceAlert{
				ID:        rule.ID,
				Timestamp: time.Now(),
				Severity:  rule.Severity,
				Title:     rule.Name,
				Metric:    rule.Metric,
				Threshold: rule.Threshold,
			}
			
			am.TriggerAlert(alert)
		}
	}
}

func (am *AlertManager) evaluateRule(rule AlertRule, metrics *RealTimeMetrics) bool {
	var value float64
	
	switch rule.Metric {
	case "p95_latency":
		value = float64(metrics.P95Latency.Nanoseconds()) / 1000000.0 // Convert to ms
	case "cache_hit_rate":
		value = metrics.CacheHitRate
	case "memory_pressure":
		value = metrics.MemoryPressure
	default:
		return false
	}
	
	switch rule.Condition {
	case ">":
		return value > rule.Threshold
	case "<":
		return value < rule.Threshold
	case ">=":
		return value >= rule.Threshold
	case "<=":
		return value <= rule.Threshold
	case "==":
		return value == rule.Threshold
	case "!=":
		return value != rule.Threshold
	}
	
	return false
}

func (am *AlertManager) GetActiveAlertCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	// This would filter for active (non-resolved) alerts
	return len(am.alerts)
}

func (am *AlertManager) GetCriticalAlertCount() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	count := 0
	for _, alert := range am.alerts {
		if alert.Severity == "critical" {
			count++
		}
	}
	
	return count
}

func (am *AlertManager) GetRecentAlerts(duration time.Duration) []PerformanceAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	cutoff := time.Now().Add(-duration)
	recentAlerts := []PerformanceAlert{}
	
	for _, alert := range am.alerts {
		if alert.Timestamp.After(cutoff) {
			recentAlerts = append(recentAlerts, alert)
		}
	}
	
	return recentAlerts
}