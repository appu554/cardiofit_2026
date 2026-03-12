package monitoring

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PerformanceTargets defines and monitors performance SLA targets
type PerformanceTargets struct {
	// Phase-specific targets (as per implementation gaps document)
	CalculateTarget   time.Duration `json:"calculate_target"`   // 110ms
	ValidateTarget    time.Duration `json:"validate_target"`    // 50ms (safe path)
	ValidateRework    time.Duration `json:"validate_rework"`    // 150ms (rework path)
	CommitTarget      time.Duration `json:"commit_target"`      // 100ms

	// End-to-end targets
	TotalSafePath     time.Duration `json:"total_safe_path"`    // 260ms
	TotalReworkPath   time.Duration `json:"total_rework_path"`  // 410ms

	// Human review SLA
	HumanReviewSLA    time.Duration `json:"human_review_sla"`   // 2 hours

	// Performance monitoring
	metrics           *PerformanceMetrics
	logger            *zap.Logger
	alertThresholds   *AlertThresholds
	mu                sync.RWMutex
}

// PerformanceMetrics tracks performance statistics
type PerformanceMetrics struct {
	// Phase performance counters
	CalculatePhaseMetrics *PhaseMetrics `json:"calculate_phase"`
	ValidatePhaseMetrics  *PhaseMetrics `json:"validate_phase"`
	CommitPhaseMetrics    *PhaseMetrics `json:"commit_phase"`

	// End-to-end workflow metrics
	SafePathMetrics       *PathMetrics  `json:"safe_path"`
	ReworkPathMetrics     *PathMetrics  `json:"rework_path"`

	// SLA compliance metrics
	HumanReviewMetrics    *SLAMetrics   `json:"human_review"`

	// Overall system health
	SystemHealth          *SystemHealthMetrics `json:"system_health"`

	// Metrics collection period
	CollectionStarted     time.Time     `json:"collection_started"`
	LastUpdated          time.Time     `json:"last_updated"`
}

// PhaseMetrics tracks metrics for individual workflow phases
type PhaseMetrics struct {
	TotalExecutions       int64         `json:"total_executions"`
	SuccessfulExecutions  int64         `json:"successful_executions"`
	FailedExecutions      int64         `json:"failed_executions"`

	// Duration statistics
	MinDuration          time.Duration `json:"min_duration"`
	MaxDuration          time.Duration `json:"max_duration"`
	AvgDuration          time.Duration `json:"avg_duration"`
	P50Duration          time.Duration `json:"p50_duration"`
	P95Duration          time.Duration `json:"p95_duration"`
	P99Duration          time.Duration `json:"p99_duration"`

	// Target compliance
	WithinTarget         int64         `json:"within_target"`
	ExceededTarget       int64         `json:"exceeded_target"`
	ComplianceRate       float64       `json:"compliance_rate"`

	// Recent durations for percentile calculation
	recentDurations      []time.Duration
	maxRecentSamples     int
}

// PathMetrics tracks end-to-end path performance
type PathMetrics struct {
	TotalWorkflows       int64         `json:"total_workflows"`
	CompletedWorkflows   int64         `json:"completed_workflows"`
	FailedWorkflows      int64         `json:"failed_workflows"`

	MinDuration          time.Duration `json:"min_duration"`
	MaxDuration          time.Duration `json:"max_duration"`
	AvgDuration          time.Duration `json:"avg_duration"`
	P95Duration          time.Duration `json:"p95_duration"`

	WithinSLA            int64         `json:"within_sla"`
	ExceededSLA          int64         `json:"exceeded_sla"`
	SLAComplianceRate    float64       `json:"sla_compliance_rate"`
}

// SLAMetrics tracks SLA compliance for human review processes
type SLAMetrics struct {
	TotalReviews         int64         `json:"total_reviews"`
	CompletedOnTime      int64         `json:"completed_on_time"`
	CompletedLate        int64         `json:"completed_late"`
	Escalated           int64         `json:"escalated"`

	AvgResponseTime      time.Duration `json:"avg_response_time"`
	P50ResponseTime      time.Duration `json:"p50_response_time"`
	P95ResponseTime      time.Duration `json:"p95_response_time"`

	SLAComplianceRate    float64       `json:"sla_compliance_rate"`
}

// SystemHealthMetrics tracks overall system health indicators
type SystemHealthMetrics struct {
	CPUUsage             float64       `json:"cpu_usage"`
	MemoryUsage          float64       `json:"memory_usage"`
	ActiveWorkflows      int64         `json:"active_workflows"`
	QueuedWorkflows      int64         `json:"queued_workflows"`

	ErrorRate            float64       `json:"error_rate"`
	ThroughputPerMinute  float64       `json:"throughput_per_minute"`

	LastHealthCheck      time.Time     `json:"last_health_check"`
}

// AlertThresholds defines when to trigger performance alerts
type AlertThresholds struct {
	// Phase duration thresholds (when to alert)
	CalculateWarnThreshold    time.Duration `json:"calculate_warn"`     // 130ms (118% of target)
	CalculateCriticalThreshold time.Duration `json:"calculate_critical"` // 165ms (150% of target)

	ValidateWarnThreshold     time.Duration `json:"validate_warn"`      // 60ms (120% of target)
	ValidateCriticalThreshold time.Duration `json:"validate_critical"`  // 75ms (150% of target)

	CommitWarnThreshold       time.Duration `json:"commit_warn"`        // 120ms (120% of target)
	CommitCriticalThreshold   time.Duration `json:"commit_critical"`    // 150ms (150% of target)

	// Compliance rate thresholds
	ComplianceWarnRate        float64       `json:"compliance_warn"`    // 85%
	ComplianceCriticalRate    float64       `json:"compliance_critical"` // 75%

	// System health thresholds
	CPUWarnThreshold         float64       `json:"cpu_warn"`           // 80%
	CPUCriticalThreshold     float64       `json:"cpu_critical"`       // 90%
	MemoryWarnThreshold      float64       `json:"memory_warn"`        // 80%
	MemoryCriticalThreshold  float64       `json:"memory_critical"`    // 90%

	ErrorRateWarnThreshold   float64       `json:"error_rate_warn"`    // 5%
	ErrorRateCriticalThreshold float64     `json:"error_rate_critical"` // 10%
}

// Performance measurement result
type PerformanceMeasurement struct {
	Phase          string        `json:"phase"`
	Duration       time.Duration `json:"duration"`
	Target         time.Duration `json:"target"`
	WithinTarget   bool          `json:"within_target"`
	OveragePercent float64       `json:"overage_percent"`
	IsRework       bool          `json:"is_rework"`
	Timestamp      time.Time     `json:"timestamp"`
}

// NewPerformanceTargets creates a new performance monitoring system
func NewPerformanceTargets(logger *zap.Logger) *PerformanceTargets {
	return &PerformanceTargets{
		// Targets from implementation gaps document
		CalculateTarget:   110 * time.Millisecond,
		ValidateTarget:    50 * time.Millisecond,
		ValidateRework:    150 * time.Millisecond,
		CommitTarget:      100 * time.Millisecond,
		TotalSafePath:     260 * time.Millisecond,
		TotalReworkPath:   410 * time.Millisecond,
		HumanReviewSLA:    2 * time.Hour,

		metrics:         NewPerformanceMetrics(),
		logger:          logger,
		alertThresholds: DefaultAlertThresholds(),
	}
}

// NewPerformanceMetrics creates a new metrics collection
func NewPerformanceMetrics() *PerformanceMetrics {
	now := time.Now()
	return &PerformanceMetrics{
		CalculatePhaseMetrics: NewPhaseMetrics(1000), // Keep last 1000 samples
		ValidatePhaseMetrics:  NewPhaseMetrics(1000),
		CommitPhaseMetrics:    NewPhaseMetrics(1000),
		SafePathMetrics:       &PathMetrics{},
		ReworkPathMetrics:     &PathMetrics{},
		HumanReviewMetrics:    &SLAMetrics{},
		SystemHealth:          &SystemHealthMetrics{LastHealthCheck: now},
		CollectionStarted:     now,
		LastUpdated:          now,
	}
}

// NewPhaseMetrics creates a new phase metrics tracker
func NewPhaseMetrics(maxSamples int) *PhaseMetrics {
	return &PhaseMetrics{
		recentDurations:  make([]time.Duration, 0, maxSamples),
		maxRecentSamples: maxSamples,
		MinDuration:      time.Duration(0),
		MaxDuration:      time.Duration(0),
	}
}

// DefaultAlertThresholds returns default alert thresholds
func DefaultAlertThresholds() *AlertThresholds {
	return &AlertThresholds{
		CalculateWarnThreshold:     130 * time.Millisecond, // 118% of 110ms
		CalculateCriticalThreshold: 165 * time.Millisecond, // 150% of 110ms
		ValidateWarnThreshold:      60 * time.Millisecond,  // 120% of 50ms
		ValidateCriticalThreshold:  75 * time.Millisecond,  // 150% of 50ms
		CommitWarnThreshold:        120 * time.Millisecond, // 120% of 100ms
		CommitCriticalThreshold:    150 * time.Millisecond, // 150% of 100ms

		ComplianceWarnRate:        0.85, // 85%
		ComplianceCriticalRate:    0.75, // 75%

		CPUWarnThreshold:          0.80, // 80%
		CPUCriticalThreshold:      0.90, // 90%
		MemoryWarnThreshold:       0.80, // 80%
		MemoryCriticalThreshold:   0.90, // 90%

		ErrorRateWarnThreshold:    0.05, // 5%
		ErrorRateCriticalThreshold: 0.10, // 10%
	}
}

// CheckCompliance verifies if a phase duration meets target
func (p *PerformanceTargets) CheckCompliance(phase string, duration time.Duration, isRework bool) *PerformanceMeasurement {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var target time.Duration
	switch phase {
	case "CALCULATE":
		target = p.CalculateTarget
	case "VALIDATE":
		if isRework {
			target = p.ValidateRework
		} else {
			target = p.ValidateTarget
		}
	case "COMMIT":
		target = p.CommitTarget
	default:
		target = 100 * time.Millisecond // Default target
	}

	withinTarget := duration <= target
	overagePercent := 0.0
	if !withinTarget {
		overagePercent = float64(duration-target) / float64(target) * 100
	}

	measurement := &PerformanceMeasurement{
		Phase:          phase,
		Duration:       duration,
		Target:         target,
		WithinTarget:   withinTarget,
		OveragePercent: overagePercent,
		IsRework:       isRework,
		Timestamp:      time.Now(),
	}

	// Log performance measurement
	if withinTarget {
		p.logger.Debug("Phase performance within target",
			zap.String("phase", phase),
			zap.Duration("duration", duration),
			zap.Duration("target", target))
	} else {
		p.logger.Warn("Phase performance exceeded target",
			zap.String("phase", phase),
			zap.Duration("duration", duration),
			zap.Duration("target", target),
			zap.Float64("overage_percent", overagePercent))
	}

	return measurement
}

// RecordPhaseMeasurement records a performance measurement for a phase
func (p *PerformanceTargets) RecordPhaseMeasurement(ctx context.Context, measurement *PerformanceMeasurement) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var phaseMetrics *PhaseMetrics
	switch measurement.Phase {
	case "CALCULATE":
		phaseMetrics = p.metrics.CalculatePhaseMetrics
	case "VALIDATE":
		phaseMetrics = p.metrics.ValidatePhaseMetrics
	case "COMMIT":
		phaseMetrics = p.metrics.CommitPhaseMetrics
	default:
		p.logger.Warn("Unknown phase for measurement", zap.String("phase", measurement.Phase))
		return
	}

	p.updatePhaseMetrics(phaseMetrics, measurement)
	p.metrics.LastUpdated = time.Now()

	// Check for alerts
	p.checkPhaseAlerts(measurement)
}

// RecordWorkflowPath records end-to-end workflow performance
func (p *PerformanceTargets) RecordWorkflowPath(ctx context.Context, pathType string, totalDuration time.Duration, success bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var pathMetrics *PathMetrics
	var target time.Duration

	switch pathType {
	case "SAFE_PATH":
		pathMetrics = p.metrics.SafePathMetrics
		target = p.TotalSafePath
	case "REWORK_PATH":
		pathMetrics = p.metrics.ReworkPathMetrics
		target = p.TotalReworkPath
	default:
		p.logger.Warn("Unknown path type", zap.String("path_type", pathType))
		return
	}

	p.updatePathMetrics(pathMetrics, totalDuration, target, success)
	p.metrics.LastUpdated = time.Now()

	// Log path performance
	withinSLA := totalDuration <= target
	if !withinSLA {
		overage := float64(totalDuration-target) / float64(target) * 100
		p.logger.Warn("Workflow path exceeded SLA",
			zap.String("path_type", pathType),
			zap.Duration("duration", totalDuration),
			zap.Duration("target", target),
			zap.Float64("overage_percent", overage))
	}
}

// RecordHumanReviewSLA records human review SLA compliance
func (p *PerformanceTargets) RecordHumanReviewSLA(ctx context.Context, responseTime time.Duration, completedOnTime bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	metrics := p.metrics.HumanReviewMetrics
	metrics.TotalReviews++

	if completedOnTime {
		metrics.CompletedOnTime++
	} else {
		metrics.CompletedLate++
	}

	// Update response time statistics
	if metrics.TotalReviews == 1 {
		metrics.AvgResponseTime = responseTime
		metrics.P50ResponseTime = responseTime
		metrics.P95ResponseTime = responseTime
	} else {
		// Simple moving average (could be improved with more sophisticated statistics)
		metrics.AvgResponseTime = time.Duration(
			(int64(metrics.AvgResponseTime)*(metrics.TotalReviews-1) + int64(responseTime)) / metrics.TotalReviews,
		)
	}

	// Update compliance rate
	metrics.SLAComplianceRate = float64(metrics.CompletedOnTime) / float64(metrics.TotalReviews)

	p.metrics.LastUpdated = time.Now()

	// Log SLA compliance
	if !completedOnTime {
		p.logger.Warn("Human review SLA missed",
			zap.Duration("response_time", responseTime),
			zap.Duration("sla_target", p.HumanReviewSLA),
			zap.Float64("compliance_rate", metrics.SLAComplianceRate))
	}
}

// GetCurrentMetrics returns a copy of current performance metrics
func (p *PerformanceTargets) GetCurrentMetrics() *PerformanceMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Return a deep copy to avoid race conditions
	return p.copyMetrics(p.metrics)
}

// GetTargets returns the current performance targets
func (p *PerformanceTargets) GetTargets() map[string]time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]time.Duration{
		"calculate_target":    p.CalculateTarget,
		"validate_target":     p.ValidateTarget,
		"validate_rework":     p.ValidateRework,
		"commit_target":       p.CommitTarget,
		"total_safe_path":     p.TotalSafePath,
		"total_rework_path":   p.TotalReworkPath,
		"human_review_sla":    p.HumanReviewSLA,
	}
}

// UpdateSystemHealth updates system health metrics
func (p *PerformanceTargets) UpdateSystemHealth(ctx context.Context, cpuUsage, memoryUsage float64, activeWorkflows, queuedWorkflows int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	health := p.metrics.SystemHealth
	health.CPUUsage = cpuUsage
	health.MemoryUsage = memoryUsage
	health.ActiveWorkflows = activeWorkflows
	health.QueuedWorkflows = queuedWorkflows
	health.LastHealthCheck = time.Now()

	// Calculate error rate based on recent metrics
	totalExecutions := p.metrics.CalculatePhaseMetrics.TotalExecutions +
		p.metrics.ValidatePhaseMetrics.TotalExecutions +
		p.metrics.CommitPhaseMetrics.TotalExecutions

	totalFailures := p.metrics.CalculatePhaseMetrics.FailedExecutions +
		p.metrics.ValidatePhaseMetrics.FailedExecutions +
		p.metrics.CommitPhaseMetrics.FailedExecutions

	if totalExecutions > 0 {
		health.ErrorRate = float64(totalFailures) / float64(totalExecutions)
	}

	p.metrics.LastUpdated = time.Now()

	// Check system health alerts
	p.checkSystemHealthAlerts(health)
}

// Helper methods

func (p *PerformanceTargets) updatePhaseMetrics(metrics *PhaseMetrics, measurement *PerformanceMeasurement) {
	metrics.TotalExecutions++

	// Update success/failure counts (assuming success if measurement was recorded)
	metrics.SuccessfulExecutions++

	// Update duration statistics
	duration := measurement.Duration
	if metrics.TotalExecutions == 1 {
		metrics.MinDuration = duration
		metrics.MaxDuration = duration
		metrics.AvgDuration = duration
	} else {
		if duration < metrics.MinDuration {
			metrics.MinDuration = duration
		}
		if duration > metrics.MaxDuration {
			metrics.MaxDuration = duration
		}

		// Update average
		metrics.AvgDuration = time.Duration(
			(int64(metrics.AvgDuration)*(metrics.TotalExecutions-1) + int64(duration)) / metrics.TotalExecutions,
		)
	}

	// Track recent durations for percentile calculation
	metrics.recentDurations = append(metrics.recentDurations, duration)
	if len(metrics.recentDurations) > metrics.maxRecentSamples {
		metrics.recentDurations = metrics.recentDurations[1:]
	}

	// Update target compliance
	if measurement.WithinTarget {
		metrics.WithinTarget++
	} else {
		metrics.ExceededTarget++
	}
	metrics.ComplianceRate = float64(metrics.WithinTarget) / float64(metrics.TotalExecutions)

	// Calculate percentiles (simplified implementation)
	p.calculatePercentiles(metrics)
}

func (p *PerformanceTargets) updatePathMetrics(pathMetrics *PathMetrics, duration, target time.Duration, success bool) {
	pathMetrics.TotalWorkflows++

	if success {
		pathMetrics.CompletedWorkflows++
	} else {
		pathMetrics.FailedWorkflows++
	}

	// Update duration statistics
	if pathMetrics.TotalWorkflows == 1 {
		pathMetrics.MinDuration = duration
		pathMetrics.MaxDuration = duration
		pathMetrics.AvgDuration = duration
		pathMetrics.P95Duration = duration
	} else {
		if duration < pathMetrics.MinDuration {
			pathMetrics.MinDuration = duration
		}
		if duration > pathMetrics.MaxDuration {
			pathMetrics.MaxDuration = duration
		}

		pathMetrics.AvgDuration = time.Duration(
			(int64(pathMetrics.AvgDuration)*(pathMetrics.TotalWorkflows-1) + int64(duration)) / pathMetrics.TotalWorkflows,
		)
	}

	// Update SLA compliance
	if duration <= target {
		pathMetrics.WithinSLA++
	} else {
		pathMetrics.ExceededSLA++
	}
	pathMetrics.SLAComplianceRate = float64(pathMetrics.WithinSLA) / float64(pathMetrics.TotalWorkflows)
}

func (p *PerformanceTargets) calculatePercentiles(metrics *PhaseMetrics) {
	if len(metrics.recentDurations) == 0 {
		return
	}

	// Simple percentile calculation (could be improved)
	durations := make([]time.Duration, len(metrics.recentDurations))
	copy(durations, metrics.recentDurations)

	// Sort durations for percentile calculation
	for i := 0; i < len(durations)-1; i++ {
		for j := 0; j < len(durations)-i-1; j++ {
			if durations[j] > durations[j+1] {
				durations[j], durations[j+1] = durations[j+1], durations[j]
			}
		}
	}

	n := len(durations)
	if n >= 2 {
		p50Index := n / 2
		metrics.P50Duration = durations[p50Index]
	}
	if n >= 20 {
		p95Index := int(float64(n) * 0.95)
		if p95Index >= n {
			p95Index = n - 1
		}
		metrics.P95Duration = durations[p95Index]
	}
	if n >= 100 {
		p99Index := int(float64(n) * 0.99)
		if p99Index >= n {
			p99Index = n - 1
		}
		metrics.P99Duration = durations[p99Index]
	}
}

func (p *PerformanceTargets) checkPhaseAlerts(measurement *PerformanceMeasurement) {
	var warnThreshold, criticalThreshold time.Duration

	switch measurement.Phase {
	case "CALCULATE":
		warnThreshold = p.alertThresholds.CalculateWarnThreshold
		criticalThreshold = p.alertThresholds.CalculateCriticalThreshold
	case "VALIDATE":
		warnThreshold = p.alertThresholds.ValidateWarnThreshold
		criticalThreshold = p.alertThresholds.ValidateCriticalThreshold
	case "COMMIT":
		warnThreshold = p.alertThresholds.CommitWarnThreshold
		criticalThreshold = p.alertThresholds.CommitCriticalThreshold
	default:
		return
	}

	if measurement.Duration >= criticalThreshold {
		p.logger.Error("CRITICAL: Phase performance threshold exceeded",
			zap.String("phase", measurement.Phase),
			zap.Duration("duration", measurement.Duration),
			zap.Duration("critical_threshold", criticalThreshold),
			zap.Float64("overage_percent", measurement.OveragePercent))
	} else if measurement.Duration >= warnThreshold {
		p.logger.Warn("WARNING: Phase performance threshold exceeded",
			zap.String("phase", measurement.Phase),
			zap.Duration("duration", measurement.Duration),
			zap.Duration("warning_threshold", warnThreshold),
			zap.Float64("overage_percent", measurement.OveragePercent))
	}
}

func (p *PerformanceTargets) checkSystemHealthAlerts(health *SystemHealthMetrics) {
	// Check CPU usage
	if health.CPUUsage >= p.alertThresholds.CPUCriticalThreshold {
		p.logger.Error("CRITICAL: CPU usage exceeded critical threshold",
			zap.Float64("cpu_usage", health.CPUUsage),
			zap.Float64("critical_threshold", p.alertThresholds.CPUCriticalThreshold))
	} else if health.CPUUsage >= p.alertThresholds.CPUWarnThreshold {
		p.logger.Warn("WARNING: CPU usage exceeded warning threshold",
			zap.Float64("cpu_usage", health.CPUUsage),
			zap.Float64("warning_threshold", p.alertThresholds.CPUWarnThreshold))
	}

	// Check Memory usage
	if health.MemoryUsage >= p.alertThresholds.MemoryCriticalThreshold {
		p.logger.Error("CRITICAL: Memory usage exceeded critical threshold",
			zap.Float64("memory_usage", health.MemoryUsage),
			zap.Float64("critical_threshold", p.alertThresholds.MemoryCriticalThreshold))
	} else if health.MemoryUsage >= p.alertThresholds.MemoryWarnThreshold {
		p.logger.Warn("WARNING: Memory usage exceeded warning threshold",
			zap.Float64("memory_usage", health.MemoryUsage),
			zap.Float64("warning_threshold", p.alertThresholds.MemoryWarnThreshold))
	}

	// Check Error rate
	if health.ErrorRate >= p.alertThresholds.ErrorRateCriticalThreshold {
		p.logger.Error("CRITICAL: Error rate exceeded critical threshold",
			zap.Float64("error_rate", health.ErrorRate),
			zap.Float64("critical_threshold", p.alertThresholds.ErrorRateCriticalThreshold))
	} else if health.ErrorRate >= p.alertThresholds.ErrorRateWarnThreshold {
		p.logger.Warn("WARNING: Error rate exceeded warning threshold",
			zap.Float64("error_rate", health.ErrorRate),
			zap.Float64("warning_threshold", p.alertThresholds.ErrorRateWarnThreshold))
	}
}

func (p *PerformanceTargets) copyMetrics(original *PerformanceMetrics) *PerformanceMetrics {
	// Create a deep copy of metrics for safe external access
	copy := &PerformanceMetrics{
		CalculatePhaseMetrics: p.copyPhaseMetrics(original.CalculatePhaseMetrics),
		ValidatePhaseMetrics:  p.copyPhaseMetrics(original.ValidatePhaseMetrics),
		CommitPhaseMetrics:    p.copyPhaseMetrics(original.CommitPhaseMetrics),
		SafePathMetrics:       p.copyPathMetrics(original.SafePathMetrics),
		ReworkPathMetrics:     p.copyPathMetrics(original.ReworkPathMetrics),
		HumanReviewMetrics:    p.copySLAMetrics(original.HumanReviewMetrics),
		SystemHealth:          p.copySystemHealthMetrics(original.SystemHealth),
		CollectionStarted:     original.CollectionStarted,
		LastUpdated:          original.LastUpdated,
	}
	return copy
}

func (p *PerformanceTargets) copyPhaseMetrics(original *PhaseMetrics) *PhaseMetrics {
	return &PhaseMetrics{
		TotalExecutions:      original.TotalExecutions,
		SuccessfulExecutions: original.SuccessfulExecutions,
		FailedExecutions:     original.FailedExecutions,
		MinDuration:          original.MinDuration,
		MaxDuration:          original.MaxDuration,
		AvgDuration:          original.AvgDuration,
		P50Duration:          original.P50Duration,
		P95Duration:          original.P95Duration,
		P99Duration:          original.P99Duration,
		WithinTarget:         original.WithinTarget,
		ExceededTarget:       original.ExceededTarget,
		ComplianceRate:       original.ComplianceRate,
	}
}

func (p *PerformanceTargets) copyPathMetrics(original *PathMetrics) *PathMetrics {
	return &PathMetrics{
		TotalWorkflows:    original.TotalWorkflows,
		CompletedWorkflows: original.CompletedWorkflows,
		FailedWorkflows:   original.FailedWorkflows,
		MinDuration:       original.MinDuration,
		MaxDuration:       original.MaxDuration,
		AvgDuration:       original.AvgDuration,
		P95Duration:       original.P95Duration,
		WithinSLA:         original.WithinSLA,
		ExceededSLA:       original.ExceededSLA,
		SLAComplianceRate: original.SLAComplianceRate,
	}
}

func (p *PerformanceTargets) copySLAMetrics(original *SLAMetrics) *SLAMetrics {
	return &SLAMetrics{
		TotalReviews:      original.TotalReviews,
		CompletedOnTime:   original.CompletedOnTime,
		CompletedLate:     original.CompletedLate,
		Escalated:        original.Escalated,
		AvgResponseTime:   original.AvgResponseTime,
		P50ResponseTime:   original.P50ResponseTime,
		P95ResponseTime:   original.P95ResponseTime,
		SLAComplianceRate: original.SLAComplianceRate,
	}
}

func (p *PerformanceTargets) copySystemHealthMetrics(original *SystemHealthMetrics) *SystemHealthMetrics {
	return &SystemHealthMetrics{
		CPUUsage:             original.CPUUsage,
		MemoryUsage:          original.MemoryUsage,
		ActiveWorkflows:      original.ActiveWorkflows,
		QueuedWorkflows:      original.QueuedWorkflows,
		ErrorRate:            original.ErrorRate,
		ThroughputPerMinute:  original.ThroughputPerMinute,
		LastHealthCheck:      original.LastHealthCheck,
	}
}