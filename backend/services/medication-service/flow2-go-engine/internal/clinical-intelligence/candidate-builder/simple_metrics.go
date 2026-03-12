package candidatebuilder

import (
	"log"
	"sync"
	"time"
)

// SimpleMetricsCollector provides a basic implementation of MetricsCollector for testing
type SimpleMetricsCollector struct {
	logger                *log.Logger
	filteringCompleteCount int
	exclusionCounts       map[string]int
	ddiFailureCount       int
	formularyFailureCount int
	mutex                 sync.RWMutex
}

// NewSimpleMetricsCollector creates a new simple metrics collector
func NewSimpleMetricsCollector(logger *log.Logger) *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		logger:          logger,
		exclusionCounts: make(map[string]int),
	}
}

// RecordFilteringComplete records completion of filtering process
func (smc *SimpleMetricsCollector) RecordFilteringComplete(requestID string, candidateCount, exclusionCount int, duration time.Duration) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	
	smc.filteringCompleteCount++
	smc.logger.Printf("METRICS: Filtering completed for request %s - candidates: %d, exclusions: %d, duration: %dms", 
		requestID, candidateCount, exclusionCount, duration.Milliseconds())
}

// RecordExclusion records a drug exclusion
func (smc *SimpleMetricsCollector) RecordExclusion(reasonCode string) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	
	smc.exclusionCounts[reasonCode]++
	smc.logger.Printf("METRICS: Drug excluded - reason: %s (total: %d)", reasonCode, smc.exclusionCounts[reasonCode])
}

// RecordDDIServiceFailure records DDI service failures
func (smc *SimpleMetricsCollector) RecordDDIServiceFailure(requestID string, err error) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	
	smc.ddiFailureCount++
	smc.logger.Printf("METRICS: DDI service failure for request %s - error: %v (total failures: %d)", 
		requestID, err, smc.ddiFailureCount)
}

// RecordFormularyServiceFailure records formulary service failures
func (smc *SimpleMetricsCollector) RecordFormularyServiceFailure(requestID string, err error) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	
	smc.formularyFailureCount++
	smc.logger.Printf("METRICS: Formulary service failure for request %s - error: %v (total failures: %d)", 
		requestID, err, smc.formularyFailureCount)
}

// GetMetrics returns current metrics
func (smc *SimpleMetricsCollector) GetMetrics() SimpleMetrics {
	smc.mutex.RLock()
	defer smc.mutex.RUnlock()
	
	exclusionCounts := make(map[string]int)
	for k, v := range smc.exclusionCounts {
		exclusionCounts[k] = v
	}
	
	return SimpleMetrics{
		FilteringCompleteCount: smc.filteringCompleteCount,
		ExclusionCounts:       exclusionCounts,
		DDIFailureCount:       smc.ddiFailureCount,
		FormularyFailureCount: smc.formularyFailureCount,
	}
}

// SimpleMetrics represents collected metrics
type SimpleMetrics struct {
	FilteringCompleteCount int            `json:"filtering_complete_count"`
	ExclusionCounts       map[string]int `json:"exclusion_counts"`
	DDIFailureCount       int            `json:"ddi_failure_count"`
	FormularyFailureCount int            `json:"formulary_failure_count"`
}

// NoOpMetricsCollector provides a no-operation implementation for when metrics are disabled
type NoOpMetricsCollector struct{}

// NewNoOpMetricsCollector creates a new no-op metrics collector
func NewNoOpMetricsCollector() *NoOpMetricsCollector {
	return &NoOpMetricsCollector{}
}

// RecordFilteringComplete does nothing
func (nmc *NoOpMetricsCollector) RecordFilteringComplete(requestID string, candidateCount, exclusionCount int, duration time.Duration) {
	// No-op
}

// RecordExclusion does nothing
func (nmc *NoOpMetricsCollector) RecordExclusion(reasonCode string) {
	// No-op
}

// RecordDDIServiceFailure does nothing
func (nmc *NoOpMetricsCollector) RecordDDIServiceFailure(requestID string, err error) {
	// No-op
}

// RecordFormularyServiceFailure does nothing
func (nmc *NoOpMetricsCollector) RecordFormularyServiceFailure(requestID string, err error) {
	// No-op
}
