package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// PerformanceMonitor provides real-time performance monitoring for workflows
type PerformanceMonitor struct {
	logger          *zap.Logger
	activeExecutions sync.Map // map[uuid.UUID]*ExecutionMonitor
	metrics         *PerformanceMetrics
	mutex           sync.RWMutex
}

// ExecutionMonitor tracks performance for a single workflow execution
type ExecutionMonitor struct {
	WorkflowID    uuid.UUID              `json:"workflow_id"`
	StartTime     time.Time              `json:"start_time"`
	LastUpdate    time.Time              `json:"last_update"`
	PhaseTimings  map[WorkflowPhase]*PhaseTimingData `json:"phase_timings"`
	TotalLatency  time.Duration          `json:"total_latency"`
	Checkpoints   []PerformanceCheckpoint `json:"checkpoints"`
	ResourceUsage *ResourceUsageTracker  `json:"resource_usage"`
	Violations    []PerformanceViolation  `json:"violations"`
	Warnings      []PerformanceWarning    `json:"warnings"`
	completed     bool
	mutex         sync.RWMutex
}

// PhaseTimingData tracks timing data for a workflow phase
type PhaseTimingData struct {
	Phase          WorkflowPhase   `json:"phase"`
	StartTime      time.Time       `json:"start_time"`
	EndTime        *time.Time      `json:"end_time,omitempty"`
	Duration       time.Duration   `json:"duration"`
	TargetDuration time.Duration   `json:"target_duration"`
	Status         string          `json:"status"`
	Checkpoints    []PhaseCheckpoint `json:"checkpoints"`
	Metrics        map[string]float64 `json:"metrics"`
}

// PhaseCheckpoint represents a checkpoint within a phase
type PhaseCheckpoint struct {
	Name        string        `json:"name"`
	Timestamp   time.Time     `json:"timestamp"`
	ElapsedTime time.Duration `json:"elapsed_time"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// PerformanceCheckpoint represents a performance checkpoint
type PerformanceCheckpoint struct {
	CheckpointID string        `json:"checkpoint_id"`
	Name         string        `json:"name"`
	Timestamp    time.Time     `json:"timestamp"`
	ElapsedTime  time.Duration `json:"elapsed_time"`
	Phase        WorkflowPhase `json:"phase"`
	Metrics      map[string]interface{} `json:"metrics"`
	Alerts       []string      `json:"alerts,omitempty"`
}

// ResourceUsageTracker tracks resource usage during execution
type ResourceUsageTracker struct {
	MemoryUsage    MemoryUsageData    `json:"memory_usage"`
	CPUUsage       CPUUsageData       `json:"cpu_usage"`
	NetworkUsage   NetworkUsageData   `json:"network_usage"`
	IOUsage        IOUsageData        `json:"io_usage"`
	ConnectionUsage ConnectionUsageData `json:"connection_usage"`
	Samples        []ResourceSample   `json:"samples"`
	mutex          sync.RWMutex
}

// MemoryUsageData tracks memory usage
type MemoryUsageData struct {
	Current     int64     `json:"current"`
	Peak        int64     `json:"peak"`
	Average     int64     `json:"average"`
	Samples     []int64   `json:"samples"`
	LastUpdated time.Time `json:"last_updated"`
}

// CPUUsageData tracks CPU usage
type CPUUsageData struct {
	Current     float64   `json:"current"`
	Peak        float64   `json:"peak"`
	Average     float64   `json:"average"`
	Samples     []float64 `json:"samples"`
	LastUpdated time.Time `json:"last_updated"`
}

// NetworkUsageData tracks network usage
type NetworkUsageData struct {
	BytesIn     int64     `json:"bytes_in"`
	BytesOut    int64     `json:"bytes_out"`
	RequestsIn  int64     `json:"requests_in"`
	RequestsOut int64     `json:"requests_out"`
	LastUpdated time.Time `json:"last_updated"`
}

// IOUsageData tracks I/O usage
type IOUsageData struct {
	ReadsTotal  int64     `json:"reads_total"`
	WritesTotal int64     `json:"writes_total"`
	BytesRead   int64     `json:"bytes_read"`
	BytesWritten int64    `json:"bytes_written"`
	LastUpdated time.Time `json:"last_updated"`
}

// ConnectionUsageData tracks connection usage
type ConnectionUsageData struct {
	Active      int       `json:"active"`
	Total       int       `json:"total"`
	Peak        int       `json:"peak"`
	LastUpdated time.Time `json:"last_updated"`
}

// ResourceSample represents a point-in-time resource usage sample
type ResourceSample struct {
	Timestamp       time.Time `json:"timestamp"`
	MemoryUsage     int64     `json:"memory_usage"`
	CPUUsage        float64   `json:"cpu_usage"`
	ActiveConnections int     `json:"active_connections"`
	NetworkBytesIn  int64     `json:"network_bytes_in"`
	NetworkBytesOut int64     `json:"network_bytes_out"`
}

// PerformanceViolation represents a performance violation
type PerformanceViolation struct {
	ViolationID   uuid.UUID     `json:"violation_id"`
	Type          string        `json:"type"`
	Severity      string        `json:"severity"`
	Description   string        `json:"description"`
	Threshold     interface{}   `json:"threshold"`
	ActualValue   interface{}   `json:"actual_value"`
	Phase         WorkflowPhase `json:"phase,omitempty"`
	Timestamp     time.Time     `json:"timestamp"`
	Resolution    string        `json:"resolution,omitempty"`
}

// PerformanceWarning represents a performance warning
type PerformanceWarning struct {
	WarningID   uuid.UUID     `json:"warning_id"`
	Type        string        `json:"type"`
	Message     string        `json:"message"`
	Phase       WorkflowPhase `json:"phase,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Suggestion  string        `json:"suggestion,omitempty"`
}

// PerformanceMetrics tracks overall performance metrics
type PerformanceMetrics struct {
	TotalExecutions      int64                          `json:"total_executions"`
	ActiveExecutions     int64                          `json:"active_executions"`
	CompletedExecutions  int64                          `json:"completed_executions"`
	FailedExecutions     int64                          `json:"failed_executions"`
	
	// Latency metrics
	AverageLatency       time.Duration                  `json:"average_latency"`
	P50Latency          time.Duration                  `json:"p50_latency"`
	P95Latency          time.Duration                  `json:"p95_latency"`
	P99Latency          time.Duration                  `json:"p99_latency"`
	
	// Phase metrics
	PhaseMetrics        map[WorkflowPhase]*PhaseMetrics `json:"phase_metrics"`
	
	// Performance targets
	TargetLatency       time.Duration                  `json:"target_latency"`
	LatencyViolations   int64                          `json:"latency_violations"`
	
	// Resource metrics
	AverageMemoryUsage  int64                          `json:"average_memory_usage"`
	PeakMemoryUsage     int64                          `json:"peak_memory_usage"`
	AverageCPUUsage     float64                        `json:"average_cpu_usage"`
	PeakCPUUsage        float64                        `json:"peak_cpu_usage"`
	
	// Quality metrics
	AverageQualityScore float64                        `json:"average_quality_score"`
	QualityThreshold    float64                        `json:"quality_threshold"`
	QualityViolations   int64                          `json:"quality_violations"`
	
	LastUpdated         time.Time                      `json:"last_updated"`
	mutex               sync.RWMutex
}

// PhaseMetrics tracks metrics for a specific phase
type PhaseMetrics struct {
	Phase               WorkflowPhase `json:"phase"`
	TotalExecutions     int64         `json:"total_executions"`
	SuccessfulExecutions int64        `json:"successful_executions"`
	FailedExecutions    int64         `json:"failed_executions"`
	AverageLatency      time.Duration `json:"average_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	TargetLatency       time.Duration `json:"target_latency"`
	ViolationCount      int64         `json:"violation_count"`
	LastUpdated         time.Time     `json:"last_updated"`
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(logger *zap.Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger: logger,
		metrics: &PerformanceMetrics{
			PhaseMetrics:     make(map[WorkflowPhase]*PhaseMetrics),
			TargetLatency:    250 * time.Millisecond,
			QualityThreshold: 0.8,
			LastUpdated:      time.Now(),
		},
	}
}

// StartExecution starts monitoring a workflow execution
func (p *PerformanceMonitor) StartExecution(workflowID uuid.UUID) *ExecutionMonitor {
	monitor := &ExecutionMonitor{
		WorkflowID:    workflowID,
		StartTime:     time.Now(),
		LastUpdate:    time.Now(),
		PhaseTimings:  make(map[WorkflowPhase]*PhaseTimingData),
		Checkpoints:   []PerformanceCheckpoint{},
		ResourceUsage: p.createResourceUsageTracker(),
		Violations:    []PerformanceViolation{},
		Warnings:      []PerformanceWarning{},
		completed:     false,
	}
	
	// Store active execution
	p.activeExecutions.Store(workflowID, monitor)
	
	// Update metrics
	p.updateMetrics(func(metrics *PerformanceMetrics) {
		metrics.TotalExecutions++
		metrics.ActiveExecutions++
		metrics.LastUpdated = time.Now()
	})
	
	// Initial checkpoint
	monitor.AddCheckpoint("execution_started", WorkflowPhase(0), map[string]interface{}{
		"workflow_id": workflowID.String(),
		"start_time":  monitor.StartTime,
	})
	
	p.logger.Debug("Started performance monitoring",
		zap.String("workflow_id", workflowID.String()),
	)
	
	return monitor
}

// StartPhase starts monitoring a workflow phase
func (monitor *ExecutionMonitor) StartPhase(phase WorkflowPhase, targetDuration time.Duration) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	
	phaseData := &PhaseTimingData{
		Phase:          phase,
		StartTime:      time.Now(),
		TargetDuration: targetDuration,
		Status:         "in_progress",
		Checkpoints:    []PhaseCheckpoint{},
		Metrics:        make(map[string]float64),
	}
	
	monitor.PhaseTimings[phase] = phaseData
	monitor.LastUpdate = time.Now()
	
	// Add checkpoint
	monitor.AddCheckpoint(fmt.Sprintf("phase_%d_started", phase), phase, map[string]interface{}{
		"phase":           int(phase),
		"target_duration": targetDuration.String(),
	})
	
	// Start resource monitoring for this phase
	monitor.ResourceUsage.StartSampling()
}

// CompletePhase completes monitoring a workflow phase
func (monitor *ExecutionMonitor) CompletePhase(phase WorkflowPhase, success bool) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	
	if phaseData, exists := monitor.PhaseTimings[phase]; exists {
		endTime := time.Now()
		phaseData.EndTime = &endTime
		phaseData.Duration = endTime.Sub(phaseData.StartTime)
		phaseData.Status = "completed"
		
		if !success {
			phaseData.Status = "failed"
		}
		
		monitor.LastUpdate = time.Now()
		
		// Check for performance violations
		if phaseData.Duration > phaseData.TargetDuration {
			violation := PerformanceViolation{
				ViolationID:   uuid.New(),
				Type:          "phase_latency",
				Severity:      "medium",
				Description:   fmt.Sprintf("Phase %d exceeded target duration", phase),
				Threshold:     phaseData.TargetDuration,
				ActualValue:   phaseData.Duration,
				Phase:         phase,
				Timestamp:     time.Now(),
				Resolution:    "investigate_phase_performance",
			}
			monitor.Violations = append(monitor.Violations, violation)
		}
		
		// Add completion checkpoint
		monitor.AddCheckpoint(fmt.Sprintf("phase_%d_completed", phase), phase, map[string]interface{}{
			"phase":        int(phase),
			"duration":     phaseData.Duration.String(),
			"success":      success,
			"target_met":   phaseData.Duration <= phaseData.TargetDuration,
		})
	}
}

// Complete completes monitoring for the entire execution
func (monitor *ExecutionMonitor) Complete() {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	
	if monitor.completed {
		return
	}
	
	monitor.completed = true
	monitor.TotalLatency = time.Since(monitor.StartTime)
	monitor.LastUpdate = time.Now()
	
	// Stop resource monitoring
	monitor.ResourceUsage.StopSampling()
	
	// Final checkpoint
	monitor.AddCheckpoint("execution_completed", WorkflowPhase(0), map[string]interface{}{
		"total_latency":      monitor.TotalLatency.String(),
		"phases_completed":   len(monitor.PhaseTimings),
		"violations_count":   len(monitor.Violations),
		"warnings_count":     len(monitor.Warnings),
	})
}

// AddCheckpoint adds a performance checkpoint
func (monitor *ExecutionMonitor) AddCheckpoint(name string, phase WorkflowPhase, metrics map[string]interface{}) {
	checkpoint := PerformanceCheckpoint{
		CheckpointID: uuid.New().String(),
		Name:         name,
		Timestamp:    time.Now(),
		ElapsedTime:  time.Since(monitor.StartTime),
		Phase:        phase,
		Metrics:      metrics,
		Alerts:       []string{},
	}
	
	monitor.Checkpoints = append(monitor.Checkpoints, checkpoint)
}

// AddPhaseCheckpoint adds a checkpoint within a phase
func (monitor *ExecutionMonitor) AddPhaseCheckpoint(phase WorkflowPhase, name string, metrics map[string]interface{}) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	
	if phaseData, exists := monitor.PhaseTimings[phase]; exists {
		checkpoint := PhaseCheckpoint{
			Name:        name,
			Timestamp:   time.Now(),
			ElapsedTime: time.Since(phaseData.StartTime),
			Metrics:     metrics,
		}
		phaseData.Checkpoints = append(phaseData.Checkpoints, checkpoint)
	}
}

// RecordMetric records a metric for a phase
func (monitor *ExecutionMonitor) RecordMetric(phase WorkflowPhase, name string, value float64) {
	monitor.mutex.Lock()
	defer monitor.mutex.Unlock()
	
	if phaseData, exists := monitor.PhaseTimings[phase]; exists {
		phaseData.Metrics[name] = value
	}
}

// AddWarning adds a performance warning
func (monitor *ExecutionMonitor) AddWarning(warningType, message string, phase WorkflowPhase, suggestion string) {
	warning := PerformanceWarning{
		WarningID:  uuid.New(),
		Type:       warningType,
		Message:    message,
		Phase:      phase,
		Timestamp:  time.Now(),
		Suggestion: suggestion,
	}
	
	monitor.mutex.Lock()
	monitor.Warnings = append(monitor.Warnings, warning)
	monitor.mutex.Unlock()
}

// GetCurrentMetrics returns current performance metrics
func (monitor *ExecutionMonitor) GetCurrentMetrics() map[string]interface{} {
	monitor.mutex.RLock()
	defer monitor.mutex.RUnlock()
	
	metrics := map[string]interface{}{
		"workflow_id":       monitor.WorkflowID.String(),
		"elapsed_time":      time.Since(monitor.StartTime).String(),
		"total_latency":     monitor.TotalLatency.String(),
		"phases_count":      len(monitor.PhaseTimings),
		"violations_count":  len(monitor.Violations),
		"warnings_count":    len(monitor.Warnings),
		"checkpoints_count": len(monitor.Checkpoints),
	}
	
	// Add phase metrics
	for phase, data := range monitor.PhaseTimings {
		phaseKey := fmt.Sprintf("phase_%d", phase)
		metrics[phaseKey] = map[string]interface{}{
			"status":         data.Status,
			"duration":       data.Duration.String(),
			"target_duration": data.TargetDuration.String(),
			"target_met":     data.Duration <= data.TargetDuration,
		}
	}
	
	// Add resource usage summary
	if monitor.ResourceUsage != nil {
		metrics["resource_usage"] = map[string]interface{}{
			"memory_peak":       monitor.ResourceUsage.MemoryUsage.Peak,
			"memory_current":    monitor.ResourceUsage.MemoryUsage.Current,
			"cpu_peak":          monitor.ResourceUsage.CPUUsage.Peak,
			"cpu_current":       monitor.ResourceUsage.CPUUsage.Current,
			"connections_active": monitor.ResourceUsage.ConnectionUsage.Active,
			"connections_peak":  monitor.ResourceUsage.ConnectionUsage.Peak,
		}
	}
	
	return metrics
}

// Resource usage tracking methods

func (p *PerformanceMonitor) createResourceUsageTracker() *ResourceUsageTracker {
	return &ResourceUsageTracker{
		MemoryUsage:    MemoryUsageData{Samples: []int64{}, LastUpdated: time.Now()},
		CPUUsage:       CPUUsageData{Samples: []float64{}, LastUpdated: time.Now()},
		NetworkUsage:   NetworkUsageData{LastUpdated: time.Now()},
		IOUsage:        IOUsageData{LastUpdated: time.Now()},
		ConnectionUsage: ConnectionUsageData{LastUpdated: time.Now()},
		Samples:        []ResourceSample{},
	}
}

// StartSampling starts resource usage sampling
func (tracker *ResourceUsageTracker) StartSampling() {
	// Implementation would start background sampling
	// For now, we'll just record the start
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()
	
	now := time.Now()
	tracker.MemoryUsage.LastUpdated = now
	tracker.CPUUsage.LastUpdated = now
	tracker.NetworkUsage.LastUpdated = now
	tracker.IOUsage.LastUpdated = now
	tracker.ConnectionUsage.LastUpdated = now
}

// StopSampling stops resource usage sampling
func (tracker *ResourceUsageTracker) StopSampling() {
	// Implementation would stop background sampling
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()
	
	// Calculate averages
	if len(tracker.MemoryUsage.Samples) > 0 {
		var total int64
		for _, sample := range tracker.MemoryUsage.Samples {
			total += sample
			if sample > tracker.MemoryUsage.Peak {
				tracker.MemoryUsage.Peak = sample
			}
		}
		tracker.MemoryUsage.Average = total / int64(len(tracker.MemoryUsage.Samples))
	}
	
	if len(tracker.CPUUsage.Samples) > 0 {
		var total float64
		for _, sample := range tracker.CPUUsage.Samples {
			total += sample
			if sample > tracker.CPUUsage.Peak {
				tracker.CPUUsage.Peak = sample
			}
		}
		tracker.CPUUsage.Average = total / float64(len(tracker.CPUUsage.Samples))
	}
}

// RecordResourceUsage records a resource usage sample
func (tracker *ResourceUsageTracker) RecordResourceUsage(memory int64, cpu float64, connections int) {
	tracker.mutex.Lock()
	defer tracker.mutex.Unlock()
	
	now := time.Now()
	
	// Update memory usage
	tracker.MemoryUsage.Current = memory
	tracker.MemoryUsage.Samples = append(tracker.MemoryUsage.Samples, memory)
	tracker.MemoryUsage.LastUpdated = now
	if memory > tracker.MemoryUsage.Peak {
		tracker.MemoryUsage.Peak = memory
	}
	
	// Update CPU usage
	tracker.CPUUsage.Current = cpu
	tracker.CPUUsage.Samples = append(tracker.CPUUsage.Samples, cpu)
	tracker.CPUUsage.LastUpdated = now
	if cpu > tracker.CPUUsage.Peak {
		tracker.CPUUsage.Peak = cpu
	}
	
	// Update connection usage
	tracker.ConnectionUsage.Active = connections
	tracker.ConnectionUsage.Total++
	tracker.ConnectionUsage.LastUpdated = now
	if connections > tracker.ConnectionUsage.Peak {
		tracker.ConnectionUsage.Peak = connections
	}
	
	// Add sample
	sample := ResourceSample{
		Timestamp:         now,
		MemoryUsage:       memory,
		CPUUsage:          cpu,
		ActiveConnections: connections,
		NetworkBytesIn:    tracker.NetworkUsage.BytesIn,
		NetworkBytesOut:   tracker.NetworkUsage.BytesOut,
	}
	tracker.Samples = append(tracker.Samples, sample)
}

// Performance monitor management methods

// StopExecution stops monitoring a workflow execution
func (p *PerformanceMonitor) StopExecution(workflowID uuid.UUID, success bool) *ExecutionMonitor {
	if monitorInterface, exists := p.activeExecutions.Load(workflowID); exists {
		monitor := monitorInterface.(*ExecutionMonitor)
		monitor.Complete()
		
		// Remove from active executions
		p.activeExecutions.Delete(workflowID)
		
		// Update overall metrics
		p.updateOverallMetrics(monitor, success)
		
		p.logger.Debug("Stopped performance monitoring",
			zap.String("workflow_id", workflowID.String()),
			zap.Duration("total_latency", monitor.TotalLatency),
			zap.Bool("success", success),
		)
		
		return monitor
	}
	
	return nil
}

// GetActiveExecutions returns all active execution monitors
func (p *PerformanceMonitor) GetActiveExecutions() []*ExecutionMonitor {
	var monitors []*ExecutionMonitor
	
	p.activeExecutions.Range(func(key, value interface{}) bool {
		if monitor, ok := value.(*ExecutionMonitor); ok {
			monitors = append(monitors, monitor)
		}
		return true
	})
	
	return monitors
}

// GetMetrics returns overall performance metrics
func (p *PerformanceMonitor) GetMetrics() *PerformanceMetrics {
	p.metrics.mutex.RLock()
	defer p.metrics.mutex.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *p.metrics
	metrics.PhaseMetrics = make(map[WorkflowPhase]*PhaseMetrics)
	
	for phase, phaseMetrics := range p.metrics.PhaseMetrics {
		phaseMetricsCopy := *phaseMetrics
		metrics.PhaseMetrics[phase] = &phaseMetricsCopy
	}
	
	return &metrics
}

// GetExecutionMonitor returns the monitor for a specific workflow
func (p *PerformanceMonitor) GetExecutionMonitor(workflowID uuid.UUID) *ExecutionMonitor {
	if monitorInterface, exists := p.activeExecutions.Load(workflowID); exists {
		return monitorInterface.(*ExecutionMonitor)
	}
	return nil
}

// Helper methods

func (p *PerformanceMonitor) updateMetrics(updateFunc func(*PerformanceMetrics)) {
	p.metrics.mutex.Lock()
	defer p.metrics.mutex.Unlock()
	
	updateFunc(p.metrics)
}

func (p *PerformanceMonitor) updateOverallMetrics(monitor *ExecutionMonitor, success bool) {
	p.updateMetrics(func(metrics *PerformanceMetrics) {
		metrics.ActiveExecutions--
		
		if success {
			metrics.CompletedExecutions++
		} else {
			metrics.FailedExecutions++
		}
		
		// Update latency metrics (simplified)
		if monitor.TotalLatency > 0 {
			if metrics.AverageLatency == 0 {
				metrics.AverageLatency = monitor.TotalLatency
			} else {
				// Simple moving average
				metrics.AverageLatency = (metrics.AverageLatency + monitor.TotalLatency) / 2
			}
			
			// Check for violations
			if monitor.TotalLatency > metrics.TargetLatency {
				metrics.LatencyViolations++
			}
		}
		
		// Update phase metrics
		for phase, phaseData := range monitor.PhaseTimings {
			if _, exists := metrics.PhaseMetrics[phase]; !exists {
				metrics.PhaseMetrics[phase] = &PhaseMetrics{
					Phase:       phase,
					TargetLatency: phaseData.TargetDuration,
					LastUpdated: time.Now(),
				}
			}
			
			phaseMetrics := metrics.PhaseMetrics[phase]
			phaseMetrics.TotalExecutions++
			
			if phaseData.Status == "completed" {
				phaseMetrics.SuccessfulExecutions++
			} else if phaseData.Status == "failed" {
				phaseMetrics.FailedExecutions++
			}
			
			if phaseData.Duration > 0 {
				if phaseMetrics.AverageLatency == 0 {
					phaseMetrics.AverageLatency = phaseData.Duration
				} else {
					phaseMetrics.AverageLatency = (phaseMetrics.AverageLatency + phaseData.Duration) / 2
				}
				
				if phaseData.Duration > phaseMetrics.TargetLatency {
					phaseMetrics.ViolationCount++
				}
			}
			
			phaseMetrics.LastUpdated = time.Now()
		}
		
		// Update resource metrics from monitor
		if monitor.ResourceUsage != nil {
			if monitor.ResourceUsage.MemoryUsage.Peak > metrics.PeakMemoryUsage {
				metrics.PeakMemoryUsage = monitor.ResourceUsage.MemoryUsage.Peak
			}
			
			if monitor.ResourceUsage.CPUUsage.Peak > metrics.PeakCPUUsage {
				metrics.PeakCPUUsage = monitor.ResourceUsage.CPUUsage.Peak
			}
			
			// Update averages (simplified)
			if metrics.AverageMemoryUsage == 0 {
				metrics.AverageMemoryUsage = monitor.ResourceUsage.MemoryUsage.Average
			} else {
				metrics.AverageMemoryUsage = (metrics.AverageMemoryUsage + monitor.ResourceUsage.MemoryUsage.Average) / 2
			}
			
			if metrics.AverageCPUUsage == 0 {
				metrics.AverageCPUUsage = monitor.ResourceUsage.CPUUsage.Average
			} else {
				metrics.AverageCPUUsage = (metrics.AverageCPUUsage + monitor.ResourceUsage.CPUUsage.Average) / 2
			}
		}
		
		metrics.LastUpdated = time.Now()
	})
}

// GetPerformanceReport generates a comprehensive performance report
func (p *PerformanceMonitor) GetPerformanceReport() *PerformanceReport {
	metrics := p.GetMetrics()
	activeExecutions := p.GetActiveExecutions()
	
	report := &PerformanceReport{
		GeneratedAt:      time.Now(),
		OverallMetrics:   metrics,
		ActiveExecutions: len(activeExecutions),
		SystemHealth:     p.calculateSystemHealth(metrics),
		Recommendations:  p.generateRecommendations(metrics),
		Alerts:           p.generateAlerts(metrics, activeExecutions),
	}
	
	return report
}

// PerformanceReport represents a comprehensive performance report
type PerformanceReport struct {
	GeneratedAt      time.Time              `json:"generated_at"`
	OverallMetrics   *PerformanceMetrics    `json:"overall_metrics"`
	ActiveExecutions int                    `json:"active_executions"`
	SystemHealth     string                 `json:"system_health"`
	Recommendations  []string               `json:"recommendations"`
	Alerts           []PerformanceAlert     `json:"alerts"`
}

// PerformanceAlert represents a performance alert
type PerformanceAlert struct {
	AlertID     uuid.UUID `json:"alert_id"`
	Severity    string    `json:"severity"`
	Type        string    `json:"type"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Resolution  string    `json:"resolution,omitempty"`
}

func (p *PerformanceMonitor) calculateSystemHealth(metrics *PerformanceMetrics) string {
	if metrics.TotalExecutions == 0 {
		return "unknown"
	}
	
	successRate := float64(metrics.CompletedExecutions) / float64(metrics.TotalExecutions)
	violationRate := float64(metrics.LatencyViolations) / float64(metrics.TotalExecutions)
	
	if successRate >= 0.95 && violationRate <= 0.05 {
		return "excellent"
	} else if successRate >= 0.90 && violationRate <= 0.10 {
		return "good"
	} else if successRate >= 0.80 && violationRate <= 0.20 {
		return "fair"
	} else {
		return "poor"
	}
}

func (p *PerformanceMonitor) generateRecommendations(metrics *PerformanceMetrics) []string {
	var recommendations []string
	
	if metrics.LatencyViolations > metrics.TotalExecutions/10 {
		recommendations = append(recommendations, "Consider increasing resource allocation or optimizing slow phases")
	}
	
	if metrics.AverageLatency > metrics.TargetLatency {
		recommendations = append(recommendations, "Review performance targets or investigate bottlenecks")
	}
	
	if metrics.PeakMemoryUsage > 1024*1024*1024 { // 1GB
		recommendations = append(recommendations, "Monitor memory usage and consider optimization")
	}
	
	if metrics.PeakCPUUsage > 80.0 {
		recommendations = append(recommendations, "High CPU usage detected, consider load balancing")
	}
	
	return recommendations
}

func (p *PerformanceMonitor) generateAlerts(metrics *PerformanceMetrics, activeExecutions []*ExecutionMonitor) []PerformanceAlert {
	var alerts []PerformanceAlert
	
	// Check for high latency violations
	if metrics.LatencyViolations > metrics.TotalExecutions/5 {
		alerts = append(alerts, PerformanceAlert{
			AlertID:    uuid.New(),
			Severity:   "high",
			Type:       "latency_violations",
			Message:    "High number of latency violations detected",
			Timestamp:  time.Now(),
			Resolution: "investigate_performance_bottlenecks",
		})
	}
	
	// Check for high failure rate
	if metrics.TotalExecutions > 0 {
		failureRate := float64(metrics.FailedExecutions) / float64(metrics.TotalExecutions)
		if failureRate > 0.10 {
			alerts = append(alerts, PerformanceAlert{
				AlertID:    uuid.New(),
				Severity:   "high",
				Type:       "high_failure_rate",
				Message:    fmt.Sprintf("High failure rate: %.2f%%", failureRate*100),
				Timestamp:  time.Now(),
				Resolution: "investigate_failure_causes",
			})
		}
	}
	
	// Check active executions for long-running workflows
	for _, execution := range activeExecutions {
		if time.Since(execution.StartTime) > 5*time.Minute {
			alerts = append(alerts, PerformanceAlert{
				AlertID:    uuid.New(),
				Severity:   "medium",
				Type:       "long_running_execution",
				Message:    fmt.Sprintf("Workflow %s has been running for %v", execution.WorkflowID.String(), time.Since(execution.StartTime)),
				Timestamp:  time.Now(),
				Resolution: "monitor_workflow_progress",
			})
		}
	}
	
	return alerts
}

// IsHealthy returns the health status of the performance monitor
func (p *PerformanceMonitor) IsHealthy() bool {
	metrics := p.GetMetrics()
	
	// Check if metrics are being updated
	if time.Since(metrics.LastUpdated) > 10*time.Minute {
		return false
	}
	
	// Check system health
	systemHealth := p.calculateSystemHealth(metrics)
	return systemHealth != "poor"
}

// Reset resets all performance metrics (for testing/maintenance)
func (p *PerformanceMonitor) Reset() {
	p.updateMetrics(func(metrics *PerformanceMetrics) {
		*metrics = PerformanceMetrics{
			PhaseMetrics:     make(map[WorkflowPhase]*PhaseMetrics),
			TargetLatency:    250 * time.Millisecond,
			QualityThreshold: 0.8,
			LastUpdated:      time.Now(),
		}
	})
	
	// Clear active executions
	p.activeExecutions.Range(func(key, value interface{}) bool {
		p.activeExecutions.Delete(key)
		return true
	})
	
	p.logger.Info("Performance monitor reset")
}