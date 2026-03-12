package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MetricsService provides comprehensive metrics collection and reporting
type MetricsService struct {
	logger          *zap.Logger
	metricsStore    *MetricsStore
	aggregator      *MetricsAggregator
	config          MetricsServiceConfig
	mutex           sync.RWMutex
}

// MetricsStore holds various metric collections
type MetricsStore struct {
	WorkflowMetrics           *WorkflowMetricsCollection           `json:"workflow_metrics"`
	PhaseMetrics              *PhaseMetricsCollection              `json:"phase_metrics"`
	ClinicalIntelligenceMetrics *ClinicalIntelligenceMetricsCollection `json:"clinical_intelligence_metrics"`
	ProposalGenerationMetrics *ProposalGenerationMetricsCollection `json:"proposal_generation_metrics"`
	PerformanceMetrics        *SystemPerformanceMetricsCollection  `json:"performance_metrics"`
	ErrorMetrics              *ErrorMetricsCollection              `json:"error_metrics"`
	QualityMetrics            *QualityMetricsCollection            `json:"quality_metrics"`
	LastUpdated               time.Time                            `json:"last_updated"`
	mutex                     sync.RWMutex
}

// WorkflowMetricsCollection tracks workflow-level metrics
type WorkflowMetricsCollection struct {
	TotalWorkflows      int64                               `json:"total_workflows"`
	ActiveWorkflows     int64                               `json:"active_workflows"`
	CompletedWorkflows  int64                               `json:"completed_workflows"`
	FailedWorkflows     int64                               `json:"failed_workflows"`
	StatusCounts        map[WorkflowStatus]int64            `json:"status_counts"`
	AverageLatency      time.Duration                       `json:"average_latency"`
	LatencyDistribution map[string]int64                    `json:"latency_distribution"`
	ThroughputRPS       float64                             `json:"throughput_rps"`
	LastUpdated         time.Time                           `json:"last_updated"`
}

// PhaseMetricsCollection tracks phase-level metrics
type PhaseMetricsCollection struct {
	PhaseCounts         map[WorkflowPhase]int64             `json:"phase_counts"`
	PhaseLatencies      map[WorkflowPhase]time.Duration     `json:"phase_latencies"`
	PhaseSuccessRates   map[WorkflowPhase]float64           `json:"phase_success_rates"`
	PhaseErrorCounts    map[WorkflowPhase]int64             `json:"phase_error_counts"`
	LastUpdated         time.Time                           `json:"last_updated"`
}

// ClinicalIntelligenceMetricsCollection tracks clinical intelligence metrics
type ClinicalIntelligenceMetricsCollection struct {
	TotalProcessings      int64                           `json:"total_processings"`
	SuccessfulProcessings int64                           `json:"successful_processings"`
	FailedProcessings     int64                           `json:"failed_processings"`
	AverageProcessingTime time.Duration                   `json:"average_processing_time"`
	AverageQualityScore   float64                         `json:"average_quality_score"`
	RuleEngineUsage       map[string]int64                `json:"rule_engine_usage"`
	WarningCounts         map[string]int64                `json:"warning_counts"`
	LastUpdated           time.Time                       `json:"last_updated"`
}

// ProposalGenerationMetricsCollection tracks proposal generation metrics
type ProposalGenerationMetricsCollection struct {
	TotalGenerations        int64                         `json:"total_generations"`
	SuccessfulGenerations   int64                         `json:"successful_generations"`
	FailedGenerations       int64                         `json:"failed_generations"`
	AverageGenerationTime   time.Duration                 `json:"average_generation_time"`
	TotalProposalsGenerated int64                         `json:"total_proposals_generated"`
	AverageProposalsPerRequest int64                      `json:"average_proposals_per_request"`
	QualityDistribution     map[string]int64              `json:"quality_distribution"`
	FHIRValidationRate      float64                       `json:"fhir_validation_rate"`
	LastUpdated             time.Time                     `json:"last_updated"`
}

// SystemPerformanceMetricsCollection tracks system performance metrics
type SystemPerformanceMetricsCollection struct {
	CPUUsage            CPUMetrics                        `json:"cpu_usage"`
	MemoryUsage         MemoryMetrics                     `json:"memory_usage"`
	NetworkUsage        NetworkMetrics                    `json:"network_usage"`
	DatabaseMetrics     DatabaseMetrics                   `json:"database_metrics"`
	CacheMetrics        CacheMetrics                      `json:"cache_metrics"`
	LastUpdated         time.Time                         `json:"last_updated"`
}

// ErrorMetricsCollection tracks error metrics
type ErrorMetricsCollection struct {
	TotalErrors         int64                             `json:"total_errors"`
	ErrorsByType        map[string]int64                  `json:"errors_by_type"`
	ErrorsBySeverity    map[string]int64                  `json:"errors_by_severity"`
	ErrorsByComponent   map[string]int64                  `json:"errors_by_component"`
	ErrorRate           float64                           `json:"error_rate"`
	LastUpdated         time.Time                         `json:"last_updated"`
}

// QualityMetricsCollection tracks quality metrics
type QualityMetricsCollection struct {
	AverageOverallQuality     float64                     `json:"average_overall_quality"`
	ClinicalAccuracyScore     float64                     `json:"clinical_accuracy_score"`
	SafetyScore               float64                     `json:"safety_score"`
	FHIRComplianceScore       float64                     `json:"fhir_compliance_score"`
	QualityDistribution       map[string]int64            `json:"quality_distribution"`
	QualityTrends             []QualityTrendPoint         `json:"quality_trends"`
	LastUpdated               time.Time                   `json:"last_updated"`
}

// Individual metric types
type CPUMetrics struct {
	Current   float64   `json:"current"`
	Average   float64   `json:"average"`
	Peak      float64   `json:"peak"`
	Samples   []float64 `json:"samples"`
}

type MemoryMetrics struct {
	Current   int64   `json:"current"`
	Average   int64   `json:"average"`
	Peak      int64   `json:"peak"`
	Samples   []int64 `json:"samples"`
}

type NetworkMetrics struct {
	BytesIn     int64 `json:"bytes_in"`
	BytesOut    int64 `json:"bytes_out"`
	RequestsIn  int64 `json:"requests_in"`
	RequestsOut int64 `json:"requests_out"`
}

type DatabaseMetrics struct {
	ActiveConnections int64         `json:"active_connections"`
	QueryLatency      time.Duration `json:"query_latency"`
	TotalQueries      int64         `json:"total_queries"`
	FailedQueries     int64         `json:"failed_queries"`
}

type CacheMetrics struct {
	HitRate     float64 `json:"hit_rate"`
	MissRate    float64 `json:"miss_rate"`
	TotalHits   int64   `json:"total_hits"`
	TotalMisses int64   `json:"total_misses"`
}

type QualityTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
}

// MetricsAggregator handles metric aggregation and reporting
type MetricsAggregator struct {
	windowSize      time.Duration
	aggregationFunc map[string]func([]float64) float64
	samples         map[string][]TimestampedValue
	mutex           sync.RWMutex
}

type TimestampedValue struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// MetricsServiceConfig contains configuration for the metrics service
type MetricsServiceConfig struct {
	CollectionInterval   time.Duration `mapstructure:"collection_interval" default:"30s"`
	AggregationWindow    time.Duration `mapstructure:"aggregation_window" default:"5m"`
	RetentionPeriod      time.Duration `mapstructure:"retention_period" default:"24h"`
	MaxSampleSize        int           `mapstructure:"max_sample_size" default:"1000"`
	EnableDetailedMetrics bool         `mapstructure:"enable_detailed_metrics" default:"true"`
	ExportInterval       time.Duration `mapstructure:"export_interval" default:"1m"`
	ExportEnabled        bool          `mapstructure:"export_enabled" default:"false"`
	ExportEndpoint       string        `mapstructure:"export_endpoint"`
}

// NewMetricsService creates a new metrics service
func NewMetricsService(config MetricsServiceConfig, logger *zap.Logger) *MetricsService {
	service := &MetricsService{
		logger:       logger,
		metricsStore: NewMetricsStore(),
		aggregator:   NewMetricsAggregator(config.AggregationWindow),
		config:       config,
	}
	
	// Start background metric collection
	go service.startMetricCollection()
	
	return service
}

// NewMetricsStore creates a new metrics store
func NewMetricsStore() *MetricsStore {
	now := time.Now()
	return &MetricsStore{
		WorkflowMetrics: &WorkflowMetricsCollection{
			StatusCounts:        make(map[WorkflowStatus]int64),
			LatencyDistribution: make(map[string]int64),
			LastUpdated:         now,
		},
		PhaseMetrics: &PhaseMetricsCollection{
			PhaseCounts:       make(map[WorkflowPhase]int64),
			PhaseLatencies:    make(map[WorkflowPhase]time.Duration),
			PhaseSuccessRates: make(map[WorkflowPhase]float64),
			PhaseErrorCounts:  make(map[WorkflowPhase]int64),
			LastUpdated:       now,
		},
		ClinicalIntelligenceMetrics: &ClinicalIntelligenceMetricsCollection{
			RuleEngineUsage: make(map[string]int64),
			WarningCounts:   make(map[string]int64),
			LastUpdated:     now,
		},
		ProposalGenerationMetrics: &ProposalGenerationMetricsCollection{
			QualityDistribution: make(map[string]int64),
			LastUpdated:         now,
		},
		PerformanceMetrics: &SystemPerformanceMetricsCollection{
			CPUUsage:    CPUMetrics{Samples: []float64{}},
			MemoryUsage: MemoryMetrics{Samples: []int64{}},
			LastUpdated: now,
		},
		ErrorMetrics: &ErrorMetricsCollection{
			ErrorsByType:      make(map[string]int64),
			ErrorsBySeverity:  make(map[string]int64),
			ErrorsByComponent: make(map[string]int64),
			LastUpdated:       now,
		},
		QualityMetrics: &QualityMetricsCollection{
			QualityDistribution: make(map[string]int64),
			QualityTrends:       []QualityTrendPoint{},
			LastUpdated:         now,
		},
		LastUpdated: now,
	}
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(windowSize time.Duration) *MetricsAggregator {
	return &MetricsAggregator{
		windowSize: windowSize,
		aggregationFunc: map[string]func([]float64) float64{
			"average": calculateAverage,
			"sum":     calculateSum,
			"max":     calculateMax,
			"min":     calculateMin,
		},
		samples: make(map[string][]TimestampedValue),
	}
}

// Workflow metrics recording methods

// RecordWorkflowExecution records workflow execution metrics
func (m *MetricsService) RecordWorkflowExecution(status WorkflowStatus, duration time.Duration, qualityScore float64) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	workflow := m.metricsStore.WorkflowMetrics
	workflow.TotalWorkflows++
	workflow.StatusCounts[status]++
	
	switch status {
	case WorkflowStatusCompleted:
		workflow.CompletedWorkflows++
	case WorkflowStatusFailed:
		workflow.FailedWorkflows++
	case WorkflowStatusInProgress:
		workflow.ActiveWorkflows++
	}
	
	// Update average latency
	if duration > 0 {
		if workflow.AverageLatency == 0 {
			workflow.AverageLatency = duration
		} else {
			workflow.AverageLatency = (workflow.AverageLatency + duration) / 2
		}
		
		// Update latency distribution
		latencyBucket := m.getLatencyBucket(duration)
		workflow.LatencyDistribution[latencyBucket]++
	}
	
	workflow.LastUpdated = time.Now()
	
	// Update quality metrics
	if qualityScore > 0 {
		m.updateQualityMetrics(qualityScore)
	}
	
	// Add to aggregator
	m.aggregator.AddSample("workflow_duration", duration.Seconds())
	m.aggregator.AddSample("workflow_quality", qualityScore)
}

// RecordPhaseExecution records phase execution metrics
func (m *MetricsService) RecordPhaseExecution(phase WorkflowPhase, status WorkflowPhaseStatus, duration time.Duration, qualityScore float64) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	phases := m.metricsStore.PhaseMetrics
	phases.PhaseCounts[phase]++
	
	if duration > 0 {
		if phases.PhaseLatencies[phase] == 0 {
			phases.PhaseLatencies[phase] = duration
		} else {
			phases.PhaseLatencies[phase] = (phases.PhaseLatencies[phase] + duration) / 2
		}
	}
	
	if status == StatusFailed {
		phases.PhaseErrorCounts[phase]++
	}
	
	// Update success rate
	if phases.PhaseCounts[phase] > 0 {
		successCount := phases.PhaseCounts[phase] - phases.PhaseErrorCounts[phase]
		phases.PhaseSuccessRates[phase] = float64(successCount) / float64(phases.PhaseCounts[phase])
	}
	
	phases.LastUpdated = time.Now()
	
	// Add to aggregator
	phaseKey := fmt.Sprintf("phase_%d_duration", phase)
	m.aggregator.AddSample(phaseKey, duration.Seconds())
}

// RecordClinicalIntelligenceProcessing records clinical intelligence metrics
func (m *MetricsService) RecordClinicalIntelligenceProcessing(duration time.Duration, qualityScore float64, warningCount int) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	ci := m.metricsStore.ClinicalIntelligenceMetrics
	ci.TotalProcessings++
	
	if qualityScore > 0 {
		ci.SuccessfulProcessings++
		
		// Update average processing time
		if ci.AverageProcessingTime == 0 {
			ci.AverageProcessingTime = duration
		} else {
			ci.AverageProcessingTime = (ci.AverageProcessingTime + duration) / 2
		}
		
		// Update average quality score
		if ci.AverageQualityScore == 0 {
			ci.AverageQualityScore = qualityScore
		} else {
			ci.AverageQualityScore = (ci.AverageQualityScore + qualityScore) / 2
		}
	} else {
		ci.FailedProcessings++
	}
	
	// Record warnings
	if warningCount > 0 {
		ci.WarningCounts["total"] += int64(warningCount)
	}
	
	ci.LastUpdated = time.Now()
	
	// Add to aggregator
	m.aggregator.AddSample("clinical_intelligence_duration", duration.Seconds())
	m.aggregator.AddSample("clinical_intelligence_quality", qualityScore)
}

// RecordProposalGeneration records proposal generation metrics
func (m *MetricsService) RecordProposalGeneration(duration time.Duration, proposalCount int, qualityScore float64, warningCount int) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	pg := m.metricsStore.ProposalGenerationMetrics
	pg.TotalGenerations++
	
	if proposalCount > 0 {
		pg.SuccessfulGenerations++
		pg.TotalProposalsGenerated += int64(proposalCount)
		
		// Update average proposals per request
		if pg.TotalGenerations > 0 {
			pg.AverageProposalsPerRequest = pg.TotalProposalsGenerated / pg.TotalGenerations
		}
		
		// Update average generation time
		if pg.AverageGenerationTime == 0 {
			pg.AverageGenerationTime = duration
		} else {
			pg.AverageGenerationTime = (pg.AverageGenerationTime + duration) / 2
		}
		
		// Update quality distribution
		qualityBucket := m.getQualityBucket(qualityScore)
		pg.QualityDistribution[qualityBucket]++
	} else {
		pg.FailedGenerations++
	}
	
	pg.LastUpdated = time.Now()
	
	// Add to aggregator
	m.aggregator.AddSample("proposal_generation_duration", duration.Seconds())
	m.aggregator.AddSample("proposal_generation_quality", qualityScore)
}

// State management metrics

// RecordWorkflowStateCreated records workflow state creation
func (m *MetricsService) RecordWorkflowStateCreated(status WorkflowStatus) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	m.metricsStore.WorkflowMetrics.StatusCounts[status]++
	m.metricsStore.WorkflowMetrics.LastUpdated = time.Now()
}

// RecordWorkflowStateUpdated records workflow state update
func (m *MetricsService) RecordWorkflowStateUpdated(status WorkflowStatus) {
	// Similar to above, could track state transitions
	m.RecordWorkflowStateCreated(status)
}

// RecordWorkflowStatesCleanup records cleanup operations
func (m *MetricsService) RecordWorkflowStatesCleanup(deletedCount int) {
	// Track cleanup metrics
	m.aggregator.AddSample("states_cleaned_up", float64(deletedCount))
}

// System performance metrics

// RecordSystemPerformance records system performance metrics
func (m *MetricsService) RecordSystemPerformance(cpu float64, memory int64, networkIn, networkOut int64) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	perf := m.metricsStore.PerformanceMetrics
	
	// Update CPU metrics
	perf.CPUUsage.Current = cpu
	perf.CPUUsage.Samples = append(perf.CPUUsage.Samples, cpu)
	if cpu > perf.CPUUsage.Peak {
		perf.CPUUsage.Peak = cpu
	}
	
	// Update memory metrics
	perf.MemoryUsage.Current = memory
	perf.MemoryUsage.Samples = append(perf.MemoryUsage.Samples, memory)
	if memory > perf.MemoryUsage.Peak {
		perf.MemoryUsage.Peak = memory
	}
	
	// Update network metrics
	perf.NetworkUsage.BytesIn += networkIn
	perf.NetworkUsage.BytesOut += networkOut
	if networkIn > 0 {
		perf.NetworkUsage.RequestsIn++
	}
	if networkOut > 0 {
		perf.NetworkUsage.RequestsOut++
	}
	
	// Calculate averages (simplified)
	if len(perf.CPUUsage.Samples) > 0 {
		var total float64
		for _, sample := range perf.CPUUsage.Samples {
			total += sample
		}
		perf.CPUUsage.Average = total / float64(len(perf.CPUUsage.Samples))
	}
	
	if len(perf.MemoryUsage.Samples) > 0 {
		var total int64
		for _, sample := range perf.MemoryUsage.Samples {
			total += sample
		}
		perf.MemoryUsage.Average = total / int64(len(perf.MemoryUsage.Samples))
	}
	
	perf.LastUpdated = time.Now()
	
	// Add to aggregator
	m.aggregator.AddSample("cpu_usage", cpu)
	m.aggregator.AddSample("memory_usage", float64(memory))
}

// RecordDatabaseMetrics records database performance metrics
func (m *MetricsService) RecordDatabaseMetrics(activeConnections int64, queryLatency time.Duration, totalQueries, failedQueries int64) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	db := &m.metricsStore.PerformanceMetrics.DatabaseMetrics
	db.ActiveConnections = activeConnections
	db.QueryLatency = queryLatency
	db.TotalQueries = totalQueries
	db.FailedQueries = failedQueries
	
	m.metricsStore.PerformanceMetrics.LastUpdated = time.Now()
	
	// Add to aggregator
	m.aggregator.AddSample("db_query_latency", queryLatency.Seconds())
	m.aggregator.AddSample("db_active_connections", float64(activeConnections))
}

// RecordCacheMetrics records cache performance metrics
func (m *MetricsService) RecordCacheMetrics(hits, misses int64) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	cache := &m.metricsStore.PerformanceMetrics.CacheMetrics
	cache.TotalHits += hits
	cache.TotalMisses += misses
	
	total := cache.TotalHits + cache.TotalMisses
	if total > 0 {
		cache.HitRate = float64(cache.TotalHits) / float64(total)
		cache.MissRate = float64(cache.TotalMisses) / float64(total)
	}
	
	m.metricsStore.PerformanceMetrics.LastUpdated = time.Now()
	
	// Add to aggregator
	m.aggregator.AddSample("cache_hit_rate", cache.HitRate)
}

// Error tracking methods

// RecordError records an error metric
func (m *MetricsService) RecordError(errorType, severity, component string) {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	errors := m.metricsStore.ErrorMetrics
	errors.TotalErrors++
	errors.ErrorsByType[errorType]++
	errors.ErrorsBySeverity[severity]++
	errors.ErrorsByComponent[component]++
	
	// Calculate error rate (simplified)
	if m.metricsStore.WorkflowMetrics.TotalWorkflows > 0 {
		errors.ErrorRate = float64(errors.TotalErrors) / float64(m.metricsStore.WorkflowMetrics.TotalWorkflows)
	}
	
	errors.LastUpdated = time.Now()
	
	// Add to aggregator
	m.aggregator.AddSample("error_count", 1.0)
}

// Quality tracking methods

func (m *MetricsService) updateQualityMetrics(qualityScore float64) {
	quality := m.metricsStore.QualityMetrics
	
	// Update average overall quality
	if quality.AverageOverallQuality == 0 {
		quality.AverageOverallQuality = qualityScore
	} else {
		quality.AverageOverallQuality = (quality.AverageOverallQuality + qualityScore) / 2
	}
	
	// Update quality distribution
	qualityBucket := m.getQualityBucket(qualityScore)
	quality.QualityDistribution[qualityBucket]++
	
	// Add trend point
	trendPoint := QualityTrendPoint{
		Timestamp: time.Now(),
		Score:     qualityScore,
	}
	quality.QualityTrends = append(quality.QualityTrends, trendPoint)
	
	// Keep only recent trend points (last 100)
	if len(quality.QualityTrends) > 100 {
		quality.QualityTrends = quality.QualityTrends[len(quality.QualityTrends)-100:]
	}
	
	quality.LastUpdated = time.Now()
}

// Metric retrieval methods

// GetAllMetrics returns all collected metrics
func (m *MetricsService) GetAllMetrics() *MetricsStore {
	m.metricsStore.mutex.RLock()
	defer m.metricsStore.mutex.RUnlock()
	
	// Create a deep copy to avoid race conditions
	return m.copyMetricsStore()
}

// GetWorkflowMetrics returns workflow metrics
func (m *MetricsService) GetWorkflowMetrics() *WorkflowMetricsCollection {
	m.metricsStore.mutex.RLock()
	defer m.metricsStore.mutex.RUnlock()
	
	// Create a copy
	metrics := *m.metricsStore.WorkflowMetrics
	metrics.StatusCounts = make(map[WorkflowStatus]int64)
	metrics.LatencyDistribution = make(map[string]int64)
	
	for k, v := range m.metricsStore.WorkflowMetrics.StatusCounts {
		metrics.StatusCounts[k] = v
	}
	for k, v := range m.metricsStore.WorkflowMetrics.LatencyDistribution {
		metrics.LatencyDistribution[k] = v
	}
	
	return &metrics
}

// GetPerformanceMetrics returns system performance metrics
func (m *MetricsService) GetPerformanceMetrics() *SystemPerformanceMetricsCollection {
	m.metricsStore.mutex.RLock()
	defer m.metricsStore.mutex.RUnlock()
	
	return m.metricsStore.PerformanceMetrics
}

// GetQualityMetrics returns quality metrics
func (m *MetricsService) GetQualityMetrics() *QualityMetricsCollection {
	m.metricsStore.mutex.RLock()
	defer m.metricsStore.mutex.RUnlock()
	
	return m.metricsStore.QualityMetrics
}

// GetMetricsSummary returns a summary of key metrics
func (m *MetricsService) GetMetricsSummary() *MetricsSummary {
	m.metricsStore.mutex.RLock()
	defer m.metricsStore.mutex.RUnlock()
	
	workflow := m.metricsStore.WorkflowMetrics
	errors := m.metricsStore.ErrorMetrics
	quality := m.metricsStore.QualityMetrics
	
	summary := &MetricsSummary{
		TotalWorkflows:      workflow.TotalWorkflows,
		ActiveWorkflows:     workflow.ActiveWorkflows,
		CompletedWorkflows:  workflow.CompletedWorkflows,
		FailedWorkflows:     workflow.FailedWorkflows,
		AverageLatency:      workflow.AverageLatency,
		ThroughputRPS:       workflow.ThroughputRPS,
		ErrorRate:           errors.ErrorRate,
		AverageQuality:      quality.AverageOverallQuality,
		SystemHealth:        m.calculateSystemHealth(),
		LastUpdated:         time.Now(),
	}
	
	// Calculate success rate
	if workflow.TotalWorkflows > 0 {
		summary.SuccessRate = float64(workflow.CompletedWorkflows) / float64(workflow.TotalWorkflows)
	}
	
	return summary
}

// MetricsSummary provides a high-level summary of metrics
type MetricsSummary struct {
	TotalWorkflows     int64         `json:"total_workflows"`
	ActiveWorkflows    int64         `json:"active_workflows"`
	CompletedWorkflows int64         `json:"completed_workflows"`
	FailedWorkflows    int64         `json:"failed_workflows"`
	SuccessRate        float64       `json:"success_rate"`
	AverageLatency     time.Duration `json:"average_latency"`
	ThroughputRPS      float64       `json:"throughput_rps"`
	ErrorRate          float64       `json:"error_rate"`
	AverageQuality     float64       `json:"average_quality"`
	SystemHealth       string        `json:"system_health"`
	LastUpdated        time.Time     `json:"last_updated"`
}

// Helper methods

func (m *MetricsService) getLatencyBucket(duration time.Duration) string {
	ms := duration.Milliseconds()
	if ms <= 50 {
		return "0-50ms"
	} else if ms <= 100 {
		return "50-100ms"
	} else if ms <= 250 {
		return "100-250ms"
	} else if ms <= 500 {
		return "250-500ms"
	} else if ms <= 1000 {
		return "500ms-1s"
	} else {
		return ">1s"
	}
}

func (m *MetricsService) getQualityBucket(score float64) string {
	if score >= 0.9 {
		return "excellent"
	} else if score >= 0.8 {
		return "good"
	} else if score >= 0.7 {
		return "fair"
	} else if score >= 0.6 {
		return "poor"
	} else {
		return "very_poor"
	}
}

func (m *MetricsService) calculateSystemHealth() string {
	workflow := m.metricsStore.WorkflowMetrics
	errors := m.metricsStore.ErrorMetrics
	
	if workflow.TotalWorkflows == 0 {
		return "unknown"
	}
	
	successRate := float64(workflow.CompletedWorkflows) / float64(workflow.TotalWorkflows)
	errorRate := errors.ErrorRate
	
	if successRate >= 0.95 && errorRate <= 0.05 {
		return "excellent"
	} else if successRate >= 0.90 && errorRate <= 0.10 {
		return "good"
	} else if successRate >= 0.80 && errorRate <= 0.20 {
		return "fair"
	} else {
		return "poor"
	}
}

func (m *MetricsService) copyMetricsStore() *MetricsStore {
	// Create a deep copy of the metrics store
	// This is a simplified version - in practice you'd use a proper deep copy library
	copy := *m.metricsStore
	
	// Copy maps
	copy.WorkflowMetrics = &WorkflowMetricsCollection{}
	*copy.WorkflowMetrics = *m.metricsStore.WorkflowMetrics
	copy.WorkflowMetrics.StatusCounts = make(map[WorkflowStatus]int64)
	copy.WorkflowMetrics.LatencyDistribution = make(map[string]int64)
	
	for k, v := range m.metricsStore.WorkflowMetrics.StatusCounts {
		copy.WorkflowMetrics.StatusCounts[k] = v
	}
	for k, v := range m.metricsStore.WorkflowMetrics.LatencyDistribution {
		copy.WorkflowMetrics.LatencyDistribution[k] = v
	}
	
	return &copy
}

// Background metric collection

func (m *MetricsService) startMetricCollection() {
	ticker := time.NewTicker(m.config.CollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			m.collectSystemMetrics()
			m.cleanupOldSamples()
		}
	}
}

func (m *MetricsService) collectSystemMetrics() {
	// This would collect actual system metrics
	// For now, we'll just update the timestamp
	m.metricsStore.mutex.Lock()
	m.metricsStore.LastUpdated = time.Now()
	m.metricsStore.mutex.Unlock()
}

func (m *MetricsService) cleanupOldSamples() {
	cutoff := time.Now().Add(-m.config.RetentionPeriod)
	
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	// Clean up quality trends
	quality := m.metricsStore.QualityMetrics
	var cleanedTrends []QualityTrendPoint
	for _, trend := range quality.QualityTrends {
		if trend.Timestamp.After(cutoff) {
			cleanedTrends = append(cleanedTrends, trend)
		}
	}
	quality.QualityTrends = cleanedTrends
	
	// Clean up performance samples
	perf := m.metricsStore.PerformanceMetrics
	if len(perf.CPUUsage.Samples) > m.config.MaxSampleSize {
		excess := len(perf.CPUUsage.Samples) - m.config.MaxSampleSize
		perf.CPUUsage.Samples = perf.CPUUsage.Samples[excess:]
	}
	if len(perf.MemoryUsage.Samples) > m.config.MaxSampleSize {
		excess := len(perf.MemoryUsage.Samples) - m.config.MaxSampleSize
		perf.MemoryUsage.Samples = perf.MemoryUsage.Samples[excess:]
	}
}

// Aggregator methods

// AddSample adds a sample to the aggregator
func (a *MetricsAggregator) AddSample(metric string, value float64) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	
	sample := TimestampedValue{
		Timestamp: time.Now(),
		Value:     value,
	}
	
	a.samples[metric] = append(a.samples[metric], sample)
	
	// Clean up old samples
	cutoff := time.Now().Add(-a.windowSize)
	var cleanedSamples []TimestampedValue
	for _, s := range a.samples[metric] {
		if s.Timestamp.After(cutoff) {
			cleanedSamples = append(cleanedSamples, s)
		}
	}
	a.samples[metric] = cleanedSamples
}

// GetAggregatedValue returns an aggregated value for a metric
func (a *MetricsAggregator) GetAggregatedValue(metric string, aggregationType string) float64 {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	
	samples := a.samples[metric]
	if len(samples) == 0 {
		return 0
	}
	
	values := make([]float64, len(samples))
	for i, sample := range samples {
		values[i] = sample.Value
	}
	
	if aggFunc, exists := a.aggregationFunc[aggregationType]; exists {
		return aggFunc(values)
	}
	
	return 0
}

// Aggregation functions
func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateSum(values []float64) float64 {
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum
}

func calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// IsHealthy returns the health status of the metrics service
func (m *MetricsService) IsHealthy() bool {
	// Check if metrics are being updated
	return time.Since(m.metricsStore.LastUpdated) < 5*time.Minute
}

// Reset resets all metrics (for testing/maintenance)
func (m *MetricsService) Reset() {
	m.metricsStore.mutex.Lock()
	defer m.metricsStore.mutex.Unlock()
	
	m.metricsStore = NewMetricsStore()
	
	m.aggregator.mutex.Lock()
	m.aggregator.samples = make(map[string][]TimestampedValue)
	m.aggregator.mutex.Unlock()
	
	m.logger.Info("Metrics service reset")
}