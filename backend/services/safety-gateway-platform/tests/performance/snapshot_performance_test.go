package performance

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	"safety-gateway-platform/internal/cache"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/context"
	"safety-gateway-platform/internal/engines"
	"safety-gateway-platform/internal/learning"
	"safety-gateway-platform/internal/override"
	"safety-gateway-platform/internal/types"
)

// SnapshotPerformanceTestSuite provides comprehensive performance benchmarking
// for all snapshot transformation components
type SnapshotPerformanceTestSuite struct {
	suite.Suite
	
	// Core services
	contextBuilder    *context.ContextBuilder
	assemblyService   *context.AssemblyService
	cacheManager      *context.CacheManager
	snapshotCache     *cache.SnapshotCache
	tokenGenerator    *override.EnhancedTokenGenerator
	eventPublisher    *learning.LearningEventPublisher
	
	// Performance targets
	targets          *PerformanceTargets
	
	// Test configuration
	testConfig       *config.Config
	logger           *zap.Logger
	ctx              context.Context
	cancel           context.CancelFunc
	
	// Performance metrics
	metrics          *PerformanceMetrics
}

// PerformanceTargets defines the performance benchmarks to achieve
type PerformanceTargets struct {
	// Latency targets (P95)
	SnapshotCreation     time.Duration // 180ms
	CacheHit             time.Duration // 10ms
	OverrideGeneration   time.Duration // 50ms
	DecisionReplay       time.Duration // 30s
	
	// Throughput targets
	RequestsPerSecond    int64         // 500 req/sec
	ConcurrentUsers      int           // 100 concurrent
	
	// Resource targets
	MaxMemoryUsage       int64         // 2GB per instance
	CacheHitRate         float64       // 90%
	ErrorRate            float64       // <0.1%
	
	// System targets
	CPUUtilization       float64       // <80%
	GCPause              time.Duration // <10ms
}

// PerformanceMetrics tracks actual performance measurements
type PerformanceMetrics struct {
	// Latency measurements
	SnapshotLatencies    []time.Duration
	CacheLatencies       []time.Duration
	OverrideLatencies    []time.Duration
	ReplayLatencies      []time.Duration
	
	// Throughput measurements
	RequestsProcessed    int64
	RequestsPerSecond    float64
	ConcurrentPeak       int32
	
	// Resource measurements
	PeakMemoryUsage      int64
	CacheHitCount        int64
	CacheMissCount       int64
	ErrorCount           int64
	
	// System measurements
	GCPauses             []time.Duration
	CPUUsage             []float64
	
	mu                   sync.RWMutex
}

// SetupSuite initializes the performance test environment
func (s *SnapshotPerformanceTestSuite) SetupSuite() {
	s.logger = zaptest.NewLogger(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	
	// Initialize performance targets
	s.targets = &PerformanceTargets{
		SnapshotCreation:   180 * time.Millisecond,
		CacheHit:          10 * time.Millisecond,
		OverrideGeneration: 50 * time.Millisecond,
		DecisionReplay:    30 * time.Second,
		RequestsPerSecond: 500,
		ConcurrentUsers:   100,
		MaxMemoryUsage:    2 * 1024 * 1024 * 1024, // 2GB
		CacheHitRate:      0.90,
		ErrorRate:         0.001, // 0.1%
		CPUUtilization:    0.80,
		GCPause:          10 * time.Millisecond,
	}
	
	// Initialize performance metrics
	s.metrics = &PerformanceMetrics{
		SnapshotLatencies: make([]time.Duration, 0),
		CacheLatencies:   make([]time.Duration, 0),
		OverrideLatencies: make([]time.Duration, 0),
		ReplayLatencies:  make([]time.Duration, 0),
		GCPauses:        make([]time.Duration, 0),
		CPUUsage:        make([]float64, 0),
	}
	
	// Load optimized configuration for performance testing
	s.testConfig = s.loadPerformanceConfig()
	
	// Initialize services
	s.setupServices()
}

// TearDownSuite cleans up the test environment
func (s *SnapshotPerformanceTestSuite) TearDownSuite() {
	s.cancel()
	s.generatePerformanceReport()
}

// BenchmarkSnapshotCreation tests snapshot creation performance
func (s *SnapshotPerformanceTestSuite) TestSnapshotCreationPerformance() {
	testCases := []struct {
		name           string
		complexity     string
		expectedP95    time.Duration
		requestCount   int
	}{
		{
			name:         "Simple Patient",
			complexity:   "simple",
			expectedP95:  120 * time.Millisecond,
			requestCount: 1000,
		},
		{
			name:         "Complex Patient",
			complexity:   "complex",
			expectedP95:  180 * time.Millisecond,
			requestCount: 500,
		},
		{
			name:         "High-Risk Patient",
			complexity:   "high_risk",
			expectedP95:  200 * time.Millisecond,
			requestCount: 250,
		},
	}
	
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.benchmarkSnapshotCreation(tc.complexity, tc.expectedP95, tc.requestCount)
		})
	}
}

// benchmarkSnapshotCreation measures snapshot creation performance
func (s *SnapshotPerformanceTestSuite) benchmarkSnapshotCreation(
	complexity string,
	expectedP95 time.Duration,
	requestCount int,
) {
	latencies := make([]time.Duration, requestCount)
	errors := int32(0)
	
	// Warm up
	s.warmUpSnapshots(complexity, 100)
	
	// Start memory monitoring
	memMonitor := s.startMemoryMonitoring()
	defer close(memMonitor)
	
	// Run benchmark
	startTime := time.Now()
	var wg sync.WaitGroup
	
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			request := s.createComplexityRequest(complexity, index)
			
			start := time.Now()
			snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
			latency := time.Since(start)
			
			latencies[index] = latency
			
			if err != nil || snapshot == nil {
				atomic.AddInt32(&errors, 1)
			}
		}(i)
	}
	
	wg.Wait()
	totalTime := time.Since(startTime)
	
	// Calculate metrics
	p95Latency := s.calculateP95(latencies)
	avgLatency := s.calculateAverage(latencies)
	throughput := float64(requestCount) / totalTime.Seconds()
	errorRate := float64(errors) / float64(requestCount)
	
	// Record metrics
	s.recordLatencies(latencies)
	s.metrics.RequestsProcessed += int64(requestCount)
	s.metrics.ErrorCount += int64(errors)
	
	// Validate performance
	assert.LessOrEqual(s.T(), p95Latency, expectedP95,
		fmt.Sprintf("P95 latency %v exceeded target %v for %s complexity",
			p95Latency, expectedP95, complexity))
	
	assert.LessOrEqual(s.T(), errorRate, s.targets.ErrorRate,
		fmt.Sprintf("Error rate %.4f exceeded target %.4f", errorRate, s.targets.ErrorRate))
	
	s.T().Logf("Complexity: %s, Requests: %d, P95: %v, Avg: %v, Throughput: %.2f req/sec, Errors: %.4f%%",
		complexity, requestCount, p95Latency, avgLatency, throughput, errorRate*100)
}

// TestCachePerformance tests cache hit rates and latencies
func (s *SnapshotPerformanceTestSuite) TestCachePerformance() {
	// Test cache performance with different hit ratios
	testCases := []struct {
		name            string
		totalRequests   int
		uniquePatients  int
		expectedHitRate float64
	}{
		{
			name:           "High Cache Hit Rate",
			totalRequests:  5000,
			uniquePatients: 500, // 90% hit rate expected
			expectedHitRate: 0.85,
		},
		{
			name:           "Medium Cache Hit Rate",
			totalRequests:  2000,
			uniquePatients: 800, // 60% hit rate expected
			expectedHitRate: 0.50,
		},
		{
			name:           "Low Cache Hit Rate",
			totalRequests:  1000,
			uniquePatients: 1000, // 0% hit rate expected
			expectedHitRate: 0.0,
		},
	}
	
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.benchmarkCachePerformance(tc.totalRequests, tc.uniquePatients, tc.expectedHitRate)
		})
	}
}

// benchmarkCachePerformance measures cache performance
func (s *SnapshotPerformanceTestSuite) benchmarkCachePerformance(
	totalRequests int,
	uniquePatients int,
	expectedHitRate float64,
) {
	// Clear cache
	s.snapshotCache.Clear()
	
	cacheHits := int64(0)
	cacheMisses := int64(0)
	hitLatencies := make([]time.Duration, 0)
	missLatencies := make([]time.Duration, 0)
	
	// Generate requests with controlled patient distribution
	requests := s.generateCacheTestRequests(totalRequests, uniquePatients)
	
	for _, request := range requests {
		start := time.Now()
		snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
		latency := time.Since(start)
		
		require.NoError(s.T(), err)
		require.NotNil(s.T(), snapshot)
		
		// Determine if this was a cache hit
		cacheKey := s.snapshotCache.GenerateCacheKey(request)
		_, wasHit := s.snapshotCache.Get(cacheKey)
		
		if wasHit && latency < 50*time.Millisecond { // Reasonable threshold for cache hit
			atomic.AddInt64(&cacheHits, 1)
			hitLatencies = append(hitLatencies, latency)
		} else {
			atomic.AddInt64(&cacheMisses, 1)
			missLatencies = append(missLatencies, latency)
		}
	}
	
	// Calculate metrics
	hitRate := float64(cacheHits) / float64(totalRequests)
	avgHitLatency := s.calculateAverage(hitLatencies)
	avgMissLatency := s.calculateAverage(missLatencies)
	
	// Validate performance
	assert.GreaterOrEqual(s.T(), hitRate, expectedHitRate,
		fmt.Sprintf("Cache hit rate %.4f below expected %.4f", hitRate, expectedHitRate))
	
	if len(hitLatencies) > 0 {
		assert.LessOrEqual(s.T(), avgHitLatency, s.targets.CacheHit,
			fmt.Sprintf("Average cache hit latency %v exceeded target %v", avgHitLatency, s.targets.CacheHit))
	}
	
	s.T().Logf("Requests: %d, Hit Rate: %.4f, Hit Latency: %v, Miss Latency: %v",
		totalRequests, hitRate, avgHitLatency, avgMissLatency)
}

// TestThroughputPerformance tests sustained throughput under load
func (s *SnapshotPerformanceTestSuite) TestThroughputPerformance() {
	testDuration := 60 * time.Second
	targetRPS := s.targets.RequestsPerSecond
	
	var (
		requestsProcessed int64
		errors           int64
		activeGoroutines int32
		peakGoroutines   int32
	)
	
	// Start monitoring
	memMonitor := s.startMemoryMonitoring()
	defer close(memMonitor)
	
	// Start throughput test
	s.T().Logf("Starting throughput test: %d req/sec for %v", targetRPS, testDuration)
	
	ctx, cancel := context.WithTimeout(s.ctx, testDuration)
	defer cancel()
	
	ticker := time.NewTicker(time.Second / time.Duration(targetRPS))
	defer ticker.Stop()
	
	startTime := time.Now()
	
throughputLoop:
	for {
		select {
		case <-ctx.Done():
			break throughputLoop
		case <-ticker.C:
			// Launch request processing goroutine
			go func() {
				current := atomic.AddInt32(&activeGoroutines, 1)
				defer atomic.AddInt32(&activeGoroutines, -1)
				
				// Track peak concurrent goroutines
				for {
					peak := atomic.LoadInt32(&peakGoroutines)
					if current <= peak {
						break
					}
					if atomic.CompareAndSwapInt32(&peakGoroutines, peak, current) {
						break
					}
				}
				
				// Process request
				request := s.createRandomRequest()
				_, err := s.contextBuilder.BuildClinicalSnapshot(context.Background(), request)
				
				atomic.AddInt64(&requestsProcessed, 1)
				if err != nil {
					atomic.AddInt64(&errors, 1)
				}
			}()
		}
	}
	
	// Wait for remaining goroutines
	for atomic.LoadInt32(&activeGoroutines) > 0 {
		time.Sleep(10 * time.Millisecond)
	}
	
	actualDuration := time.Since(startTime)
	actualRPS := float64(requestsProcessed) / actualDuration.Seconds()
	errorRate := float64(errors) / float64(requestsProcessed)
	
	// Validate throughput
	assert.GreaterOrEqual(s.T(), actualRPS, float64(targetRPS)*0.95,
		fmt.Sprintf("Actual RPS %.2f below 95%% of target %d", actualRPS, targetRPS))
	
	assert.LessOrEqual(s.T(), errorRate, s.targets.ErrorRate,
		fmt.Sprintf("Error rate %.4f exceeded target %.4f", errorRate, s.targets.ErrorRate))
	
	s.T().Logf("Duration: %v, Processed: %d, RPS: %.2f, Peak Concurrent: %d, Errors: %.4f%%",
		actualDuration, requestsProcessed, actualRPS, peakGoroutines, errorRate*100)
}

// TestMemoryPerformance tests memory usage and garbage collection
func (s *SnapshotPerformanceTestSuite) TestMemoryPerformance() {
	// Force initial garbage collection
	runtime.GC()
	
	var initialMem runtime.MemStats
	runtime.ReadMemStats(&initialMem)
	
	// Run memory-intensive operations
	requestCount := 10000
	requests := make([]*types.SafetyRequest, requestCount)
	snapshots := make([]*types.ClinicalSnapshot, requestCount)
	
	for i := 0; i < requestCount; i++ {
		requests[i] = s.createRandomRequest()
	}
	
	startTime := time.Now()
	
	// Process all requests
	for i, request := range requests {
		snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
		require.NoError(s.T(), err)
		snapshots[i] = snapshot
		
		// Monitor memory every 1000 requests
		if i%1000 == 999 {
			runtime.GC()
			var currentMem runtime.MemStats
			runtime.ReadMemStats(&currentMem)
			
			allocatedMB := float64(currentMem.Alloc) / 1024 / 1024
			s.T().Logf("Processed %d requests, Memory: %.2f MB", i+1, allocatedMB)
		}
	}
	
	processingTime := time.Since(startTime)
	
	// Final memory measurement
	runtime.GC()
	var finalMem runtime.MemStats
	runtime.ReadMemStats(&finalMem)
	
	// Calculate metrics
	peakMemoryMB := float64(finalMem.Alloc) / 1024 / 1024
	gcCount := finalMem.NumGC - initialMem.NumGC
	totalGCPause := time.Duration(finalMem.PauseTotalNs - initialMem.PauseTotalNs)
	avgGCPause := time.Duration(0)
	if gcCount > 0 {
		avgGCPause = totalGCPause / time.Duration(gcCount)
	}
	
	// Validate memory performance
	maxMemoryMB := float64(s.targets.MaxMemoryUsage) / 1024 / 1024
	assert.LessOrEqual(s.T(), peakMemoryMB, maxMemoryMB,
		fmt.Sprintf("Peak memory %.2f MB exceeded target %.2f MB", peakMemoryMB, maxMemoryMB))
	
	assert.LessOrEqual(s.T(), avgGCPause, s.targets.GCPause,
		fmt.Sprintf("Average GC pause %v exceeded target %v", avgGCPause, s.targets.GCPause))
	
	s.T().Logf("Requests: %d, Time: %v, Memory: %.2f MB, GC Count: %d, Avg GC Pause: %v",
		requestCount, processingTime, peakMemoryMB, gcCount, avgGCPause)
	
	// Clear references to allow garbage collection
	for i := range snapshots {
		snapshots[i] = nil
	}
	runtime.GC()
}

// TestOverrideTokenPerformance tests enhanced token generation performance
func (s *SnapshotPerformanceTestSuite) TestOverrideTokenPerformance() {
	tokenCount := 1000
	latencies := make([]time.Duration, tokenCount)
	errors := int32(0)
	
	// Pre-create test data
	requests := make([]*types.SafetyRequest, tokenCount)
	responses := make([]*types.SafetyResponse, tokenCount)
	snapshots := make([]*types.ClinicalSnapshot, tokenCount)
	
	for i := 0; i < tokenCount; i++ {
		requests[i] = s.createRandomRequest()
		snapshots[i] = s.createTestSnapshot(requests[i])
		responses[i] = s.createUnsafeResponse(requests[i], snapshots[i])
	}
	
	// Benchmark token generation
	var wg sync.WaitGroup
	
	for i := 0; i < tokenCount; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			start := time.Now()
			token, err := s.tokenGenerator.GenerateEnhancedToken(
				requests[index],
				responses[index],
				snapshots[index],
			)
			latency := time.Since(start)
			
			latencies[index] = latency
			
			if err != nil || token == nil {
				atomic.AddInt32(&errors, 1)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Calculate metrics
	p95Latency := s.calculateP95(latencies)
	avgLatency := s.calculateAverage(latencies)
	errorRate := float64(errors) / float64(tokenCount)
	
	// Validate performance
	assert.LessOrEqual(s.T(), p95Latency, s.targets.OverrideGeneration,
		fmt.Sprintf("P95 token generation latency %v exceeded target %v", 
			p95Latency, s.targets.OverrideGeneration))
	
	assert.LessOrEqual(s.T(), errorRate, s.targets.ErrorRate,
		fmt.Sprintf("Token generation error rate %.4f exceeded target %.4f", 
			errorRate, s.targets.ErrorRate))
	
	s.T().Logf("Token Generation - Count: %d, P95: %v, Avg: %v, Errors: %.4f%%",
		tokenCount, p95Latency, avgLatency, errorRate*100)
}

// Helper methods for performance testing

func (s *SnapshotPerformanceTestSuite) loadPerformanceConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:            8030,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			MaxHeaderBytes:  1 << 20,
		},
		Cache: config.CacheConfig{
			TTL:              2 * time.Hour,
			MaxSize:          10000, // Larger cache for performance testing
			EvictionPolicy:   "lru",
			CompressionLevel: 3, // Lower compression for speed
		},
		Learning: config.LearningConfig{
			Enabled: true,
			EventPublisher: config.EventPublisherConfig{
				EnableEventPublishing: true,
				BatchSize:            500, // Larger batches for performance
				FlushIntervalSeconds: 10,
				RetryAttempts:       2, // Fewer retries for speed
			},
		},
	}
}

func (s *SnapshotPerformanceTestSuite) calculateP95(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	// Sort latencies
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	// Calculate P95 index
	index := int(math.Ceil(float64(len(sorted)) * 0.95)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	
	return sorted[index]
}

func (s *SnapshotPerformanceTestSuite) calculateAverage(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	
	return total / time.Duration(len(latencies))
}

func (s *SnapshotPerformanceTestSuite) startMemoryMonitoring() chan struct{} {
	done := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)
				
				s.metrics.mu.Lock()
				if int64(mem.Alloc) > s.metrics.PeakMemoryUsage {
					s.metrics.PeakMemoryUsage = int64(mem.Alloc)
				}
				s.metrics.mu.Unlock()
			}
		}
	}()
	
	return done
}

func (s *SnapshotPerformanceTestSuite) generatePerformanceReport() {
	s.T().Logf("\n=== Performance Test Report ===")
	s.T().Logf("Total Requests Processed: %d", s.metrics.RequestsProcessed)
	s.T().Logf("Total Errors: %d", s.metrics.ErrorCount)
	s.T().Logf("Peak Memory Usage: %.2f MB", float64(s.metrics.PeakMemoryUsage)/1024/1024)
	
	if len(s.metrics.SnapshotLatencies) > 0 {
		p95 := s.calculateP95(s.metrics.SnapshotLatencies)
		avg := s.calculateAverage(s.metrics.SnapshotLatencies)
		s.T().Logf("Snapshot Creation - P95: %v, Avg: %v", p95, avg)
	}
	
	s.T().Logf("=== End Report ===\n")
}

// TestSnapshotPerformanceTestSuite runs the performance test suite
func TestSnapshotPerformanceTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	suite.Run(t, new(SnapshotPerformanceTestSuite))
}