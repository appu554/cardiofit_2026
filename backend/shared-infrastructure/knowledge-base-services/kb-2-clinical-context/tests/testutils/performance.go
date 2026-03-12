package testutils

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PerformanceMetrics holds performance measurement data
type PerformanceMetrics struct {
	TotalRequests   int
	SuccessfulReqs  int
	FailedReqs      int
	TotalDuration   time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
	AvgDuration     time.Duration
	P50Duration     time.Duration
	P95Duration     time.Duration
	P99Duration     time.Duration
	Throughput      float64 // requests per second
	ErrorRate       float64 // percentage
	MemoryStart     runtime.MemStats
	MemoryEnd       runtime.MemStats
	MemoryIncrease  uint64 // bytes
}

// PerformanceTestConfig configures performance test parameters
type PerformanceTestConfig struct {
	MaxDuration     time.Duration // Maximum allowed duration per request
	MaxThroughput   int           // Minimum required throughput (RPS)
	MaxErrorRate    float64       // Maximum allowed error rate (percentage)
	MaxMemoryMB     int           // Maximum memory usage increase (MB)
	ConcurrentUsers int           // Number of concurrent requests
	TestDuration    time.Duration // Total test duration
	WarmupRequests  int           // Number of warmup requests
}

// SLATarget defines service level agreement targets
type SLATarget struct {
	P50Latency  time.Duration // 50th percentile latency target
	P95Latency  time.Duration // 95th percentile latency target  
	P99Latency  time.Duration // 99th percentile latency target
	Throughput  int           // Minimum throughput (RPS)
	ErrorRate   float64       // Maximum error rate percentage
	Availability float64      // Minimum availability percentage
}

// KB2SLATargets defines the SLA targets for KB-2 Clinical Context service
var KB2SLATargets = SLATarget{
	P50Latency:   5 * time.Millisecond,
	P95Latency:   25 * time.Millisecond,
	P99Latency:   100 * time.Millisecond,
	Throughput:   10000, // 10K RPS
	ErrorRate:    0.1,   // 0.1%
	Availability: 99.9,  // 99.9%
}

// PerformanceTester provides utilities for performance testing
type PerformanceTester struct {
	config  PerformanceTestConfig
	results []time.Duration
	errors  []error
	mutex   sync.Mutex
}

// NewPerformanceTester creates a new performance tester
func NewPerformanceTester(config PerformanceTestConfig) *PerformanceTester {
	return &PerformanceTester{
		config:  config,
		results: make([]time.Duration, 0),
		errors:  make([]error, 0),
	}
}

// DefaultPerformanceConfig returns default performance test configuration
func DefaultPerformanceConfig() PerformanceTestConfig {
	return PerformanceTestConfig{
		MaxDuration:     100 * time.Millisecond,
		MaxThroughput:   1000, // 1K RPS for unit tests
		MaxErrorRate:    1.0,  // 1% for unit tests
		MaxMemoryMB:     100,  // 100MB increase
		ConcurrentUsers: 10,
		TestDuration:    10 * time.Second,
		WarmupRequests:  100,
	}
}

// LoadTestConfig returns configuration for load testing
func LoadTestConfig() PerformanceTestConfig {
	return PerformanceTestConfig{
		MaxDuration:     KB2SLATargets.P99Latency,
		MaxThroughput:   KB2SLATargets.Throughput,
		MaxErrorRate:    KB2SLATargets.ErrorRate,
		MaxMemoryMB:     500, // 500MB increase for load tests
		ConcurrentUsers: 100,
		TestDuration:    60 * time.Second,
		WarmupRequests:  1000,
	}
}

// RunPerformanceTest executes a performance test with the given test function
func (pt *PerformanceTester) RunPerformanceTest(t *testing.T, testFunc func() error) *PerformanceMetrics {
	t.Helper()

	// Collect initial memory stats
	var memStart runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStart)

	// Run warmup requests
	t.Logf("Running %d warmup requests...", pt.config.WarmupRequests)
	for i := 0; i < pt.config.WarmupRequests; i++ {
		_ = testFunc()
	}
	runtime.GC()

	// Reset results for actual test
	pt.mutex.Lock()
	pt.results = make([]time.Duration, 0)
	pt.errors = make([]error, 0)
	pt.mutex.Unlock()

	t.Logf("Starting performance test: %d concurrent users for %v", 
		pt.config.ConcurrentUsers, pt.config.TestDuration)

	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), pt.config.TestDuration)
	defer cancel()

	// Run concurrent test
	var wg sync.WaitGroup
	for i := 0; i < pt.config.ConcurrentUsers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pt.runConcurrentRequests(ctx, testFunc)
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Collect final memory stats
	var memEnd runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memEnd)

	// Calculate metrics
	metrics := pt.calculateMetrics(memStart, memEnd, totalDuration)
	
	t.Logf("Performance test completed: %d requests in %v", 
		metrics.TotalRequests, totalDuration)
	t.Logf("Throughput: %.2f RPS, Error Rate: %.2f%%", 
		metrics.Throughput, metrics.ErrorRate)
	t.Logf("Latency - P50: %v, P95: %v, P99: %v", 
		metrics.P50Duration, metrics.P95Duration, metrics.P99Duration)

	return metrics
}

// runConcurrentRequests runs requests concurrently until context is cancelled
func (pt *PerformanceTester) runConcurrentRequests(ctx context.Context, testFunc func() error) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			err := testFunc()
			duration := time.Since(start)

			pt.mutex.Lock()
			pt.results = append(pt.results, duration)
			if err != nil {
				pt.errors = append(pt.errors, err)
			}
			pt.mutex.Unlock()
		}
	}
}

// calculateMetrics computes performance metrics from test results
func (pt *PerformanceTester) calculateMetrics(memStart, memEnd runtime.MemStats, totalDuration time.Duration) *PerformanceMetrics {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	totalRequests := len(pt.results)
	failedReqs := len(pt.errors)
	successfulReqs := totalRequests - failedReqs

	if totalRequests == 0 {
		return &PerformanceMetrics{
			MemoryStart:    memStart,
			MemoryEnd:      memEnd,
			MemoryIncrease: memEnd.HeapInuse - memStart.HeapInuse,
		}
	}

	// Sort durations for percentile calculations
	sortedDurations := make([]time.Duration, len(pt.results))
	copy(sortedDurations, pt.results)
	
	// Simple bubble sort (sufficient for test data)
	for i := 0; i < len(sortedDurations); i++ {
		for j := 0; j < len(sortedDurations)-i-1; j++ {
			if sortedDurations[j] > sortedDurations[j+1] {
				sortedDurations[j], sortedDurations[j+1] = sortedDurations[j+1], sortedDurations[j]
			}
		}
	}

	// Calculate basic stats
	var totalDur time.Duration
	minDur := sortedDurations[0]
	maxDur := sortedDurations[len(sortedDurations)-1]

	for _, dur := range sortedDurations {
		totalDur += dur
	}

	avgDuration := totalDur / time.Duration(len(sortedDurations))
	
	// Calculate percentiles
	p50Index := len(sortedDurations) * 50 / 100
	p95Index := len(sortedDurations) * 95 / 100
	p99Index := len(sortedDurations) * 99 / 100
	
	if p50Index >= len(sortedDurations) {
		p50Index = len(sortedDurations) - 1
	}
	if p95Index >= len(sortedDurations) {
		p95Index = len(sortedDurations) - 1
	}
	if p99Index >= len(sortedDurations) {
		p99Index = len(sortedDurations) - 1
	}

	p50Duration := sortedDurations[p50Index]
	p95Duration := sortedDurations[p95Index]
	p99Duration := sortedDurations[p99Index]

	// Calculate throughput and error rate
	throughput := float64(totalRequests) / totalDuration.Seconds()
	errorRate := float64(failedReqs) / float64(totalRequests) * 100

	return &PerformanceMetrics{
		TotalRequests:   totalRequests,
		SuccessfulReqs:  successfulReqs,
		FailedReqs:      failedReqs,
		TotalDuration:   totalDur,
		MinDuration:     minDur,
		MaxDuration:     maxDur,
		AvgDuration:     avgDuration,
		P50Duration:     p50Duration,
		P95Duration:     p95Duration,
		P99Duration:     p99Duration,
		Throughput:      throughput,
		ErrorRate:       errorRate,
		MemoryStart:     memStart,
		MemoryEnd:       memEnd,
		MemoryIncrease:  memEnd.HeapInuse - memStart.HeapInuse,
	}
}

// ValidatePerformanceMetrics validates metrics against configuration
func (pt *PerformanceTester) ValidatePerformanceMetrics(t *testing.T, metrics *PerformanceMetrics) {
	t.Helper()

	// Validate error rate
	assert.LessOrEqual(t, metrics.ErrorRate, pt.config.MaxErrorRate,
		"Error rate %.2f%% exceeds maximum allowed %.2f%%", 
		metrics.ErrorRate, pt.config.MaxErrorRate)

	// Validate throughput
	if pt.config.MaxThroughput > 0 {
		assert.GreaterOrEqual(t, metrics.Throughput, float64(pt.config.MaxThroughput),
			"Throughput %.2f RPS is below minimum required %d RPS", 
			metrics.Throughput, pt.config.MaxThroughput)
	}

	// Validate memory usage
	memoryIncreaseMB := float64(metrics.MemoryIncrease) / 1024 / 1024
	assert.LessOrEqual(t, memoryIncreaseMB, float64(pt.config.MaxMemoryMB),
		"Memory increase %.2f MB exceeds maximum allowed %d MB", 
		memoryIncreaseMB, pt.config.MaxMemoryMB)
}

// ValidateSLACompliance validates metrics against SLA targets
func ValidateSLACompliance(t *testing.T, metrics *PerformanceMetrics, targets SLATarget) {
	t.Helper()

	// Validate latency targets
	assert.LessOrEqual(t, metrics.P50Duration, targets.P50Latency,
		"P50 latency %v exceeds SLA target %v", metrics.P50Duration, targets.P50Latency)

	assert.LessOrEqual(t, metrics.P95Duration, targets.P95Latency,
		"P95 latency %v exceeds SLA target %v", metrics.P95Duration, targets.P95Latency)

	assert.LessOrEqual(t, metrics.P99Duration, targets.P99Latency,
		"P99 latency %v exceeds SLA target %v", metrics.P99Duration, targets.P99Latency)

	// Validate throughput target
	assert.GreaterOrEqual(t, metrics.Throughput, float64(targets.Throughput),
		"Throughput %.2f RPS is below SLA target %d RPS", 
		metrics.Throughput, targets.Throughput)

	// Validate error rate target
	assert.LessOrEqual(t, metrics.ErrorRate, targets.ErrorRate,
		"Error rate %.2f%% exceeds SLA target %.2f%%", 
		metrics.ErrorRate, targets.ErrorRate)
}

// BenchmarkFunction runs a benchmark test for a specific function
func BenchmarkFunction(b *testing.B, testFunc func() error) {
	b.Helper()

	// Warmup
	for i := 0; i < 100; i++ {
		_ = testFunc()
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := testFunc()
			if err != nil {
				b.Errorf("Benchmark function failed: %v", err)
			}
		}
	})
}

// StressTestConfig defines configuration for stress testing
type StressTestConfig struct {
	MaxConcurrentUsers int
	RampUpDuration     time.Duration
	SustainDuration    time.Duration
	RampDownDuration   time.Duration
}

// RunStressTest performs stress testing with gradual load increase
func RunStressTest(t *testing.T, testFunc func() error, config StressTestConfig) *PerformanceMetrics {
	t.Helper()

	pt := NewPerformanceTester(PerformanceTestConfig{
		ConcurrentUsers: config.MaxConcurrentUsers,
		TestDuration:    config.RampUpDuration + config.SustainDuration + config.RampDownDuration,
	})

	t.Logf("Starting stress test: ramp up to %d users over %v, sustain for %v, ramp down over %v",
		config.MaxConcurrentUsers, config.RampUpDuration, config.SustainDuration, config.RampDownDuration)

	// Collect initial memory stats
	var memStart runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memStart)

	startTime := time.Now()
	
	// Ramp up phase
	rampUpStep := time.Duration(config.RampUpDuration.Nanoseconds() / int64(config.MaxConcurrentUsers))
	ctx, cancel := context.WithTimeout(context.Background(), 
		config.RampUpDuration + config.SustainDuration + config.RampDownDuration)
	defer cancel()

	var wg sync.WaitGroup
	activeUsers := 0

	// Gradually increase load
	for i := 0; i < config.MaxConcurrentUsers; i++ {
		wg.Add(1)
		activeUsers++
		go func() {
			defer wg.Done()
			pt.runConcurrentRequests(ctx, testFunc)
		}()
		
		select {
		case <-time.After(rampUpStep):
			continue
		case <-ctx.Done():
			break
		}
	}

	// Sustain phase - all users running
	t.Logf("Sustaining %d concurrent users for %v", activeUsers, config.SustainDuration)
	select {
	case <-time.After(config.SustainDuration):
	case <-ctx.Done():
	}

	// Ramp down is handled by context cancellation
	cancel()
	wg.Wait()

	totalDuration := time.Since(startTime)

	// Collect final memory stats
	var memEnd runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memEnd)

	metrics := pt.calculateMetrics(memStart, memEnd, totalDuration)
	
	t.Logf("Stress test completed: %d requests processed", metrics.TotalRequests)
	t.Logf("Peak throughput: %.2f RPS, Error rate: %.2f%%", 
		metrics.Throughput, metrics.ErrorRate)

	return metrics
}

// AssertPerformanceImprovement compares two performance metrics and ensures improvement
func AssertPerformanceImprovement(t *testing.T, baseline, improved *PerformanceMetrics, improvementThreshold float64) {
	t.Helper()

	// Calculate improvement percentages
	throughputImprovement := (improved.Throughput - baseline.Throughput) / baseline.Throughput * 100
	latencyImprovement := (baseline.P95Duration.Seconds() - improved.P95Duration.Seconds()) / baseline.P95Duration.Seconds() * 100
	errorRateImprovement := (baseline.ErrorRate - improved.ErrorRate) / baseline.ErrorRate * 100

	t.Logf("Performance comparison - Throughput: %.2f%% improvement, Latency: %.2f%% improvement, Error rate: %.2f%% improvement",
		throughputImprovement, latencyImprovement, errorRateImprovement)

	// Assert improvements meet threshold
	assert.GreaterOrEqual(t, throughputImprovement, improvementThreshold,
		"Throughput improvement %.2f%% is below threshold %.2f%%", 
		throughputImprovement, improvementThreshold)

	assert.GreaterOrEqual(t, latencyImprovement, improvementThreshold,
		"Latency improvement %.2f%% is below threshold %.2f%%", 
		latencyImprovement, improvementThreshold)
}