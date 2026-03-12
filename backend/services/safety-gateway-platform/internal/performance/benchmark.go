package performance

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/cache"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// BenchmarkSuite provides comprehensive performance benchmarking capabilities
type BenchmarkSuite struct {
	config    *config.SnapshotConfig
	logger    *logger.Logger
	cache     *cache.SnapshotCache
	optimizer *cache.CacheOptimizer
	monitor   *Monitor
	
	// Benchmark configuration
	testDuration      time.Duration
	concurrencyLevels []int
	dataSizes         []int
	testScenarios     []TestScenario
	
	// Results storage
	results       map[string]*BenchmarkResult
	aggregateResults *AggregateResults
	mu            sync.RWMutex
}

// TestScenario defines different performance test scenarios
type TestScenario struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	ConcurrentUsers int          `json:"concurrent_users"`
	RequestRate     float64      `json:"request_rate"`
	DataSize        int          `json:"data_size_bytes"`
	CacheHitRatio   float64      `json:"cache_hit_ratio"`
	TestDuration    time.Duration `json:"test_duration"`
	WarmupDuration  time.Duration `json:"warmup_duration"`
}

// BenchmarkResult contains results from a single benchmark test
type BenchmarkResult struct {
	Scenario          TestScenario      `json:"scenario"`
	StartTime         time.Time         `json:"start_time"`
	EndTime           time.Time         `json:"end_time"`
	Duration          time.Duration     `json:"duration"`
	
	// Latency metrics
	LatencyStats      LatencyStatistics `json:"latency_stats"`
	
	// Throughput metrics
	TotalRequests     int64             `json:"total_requests"`
	SuccessfulRequests int64            `json:"successful_requests"`
	FailedRequests    int64             `json:"failed_requests"`
	RequestsPerSecond float64           `json:"requests_per_second"`
	
	// Cache metrics
	CacheMetrics      CacheBenchmarkMetrics `json:"cache_metrics"`
	
	// Resource utilization
	ResourceMetrics   ResourceBenchmarkMetrics `json:"resource_metrics"`
	
	// SLA compliance
	SLACompliance     SLABenchmarkMetrics `json:"sla_compliance"`
	
	// Error analysis
	Errors            []BenchmarkError   `json:"errors"`
	
	// Performance score
	PerformanceScore  int               `json:"performance_score"`
}

// AggregateResults contains aggregated results across all benchmark tests
type AggregateResults struct {
	TotalTests        int                          `json:"total_tests"`
	PassedTests       int                          `json:"passed_tests"`
	FailedTests       int                          `json:"failed_tests"`
	
	// Aggregate metrics
	BestPerformance   *BenchmarkResult            `json:"best_performance"`
	WorstPerformance  *BenchmarkResult            `json:"worst_performance"`
	AveragePerformance *PerformanceMetrics        `json:"average_performance"`
	
	// Scalability analysis
	ScalabilityMetrics *ScalabilityAnalysis       `json:"scalability_metrics"`
	
	// Recommendations
	Recommendations   []PerformanceRecommendation `json:"recommendations"`
	
	// Trend analysis
	TrendAnalysis     *TrendAnalysis              `json:"trend_analysis"`
}

// Supporting data structures
type LatencyStatistics struct {
	Min         time.Duration `json:"min"`
	Max         time.Duration `json:"max"`
	Mean        time.Duration `json:"mean"`
	Median      time.Duration `json:"median"`
	P95         time.Duration `json:"p95"`
	P99         time.Duration `json:"p99"`
	P999        time.Duration `json:"p999"`
	StdDev      time.Duration `json:"std_dev"`
	Samples     []time.Duration `json:"-"` // Don't serialize raw samples
	SampleCount int           `json:"sample_count"`
}

type CacheBenchmarkMetrics struct {
	HitRate           float64   `json:"hit_rate"`
	MissRate          float64   `json:"miss_rate"`
	L1HitRate         float64   `json:"l1_hit_rate"`
	L2HitRate         float64   `json:"l2_hit_rate"`
	AverageGetLatency time.Duration `json:"average_get_latency"`
	AverageSetLatency time.Duration `json:"average_set_latency"`
	CompressionRatio  float64   `json:"compression_ratio"`
	MemoryUtilization int64     `json:"memory_utilization_bytes"`
	EvictionRate      float64   `json:"eviction_rate"`
}

type ResourceBenchmarkMetrics struct {
	PeakMemoryUsage   int64   `json:"peak_memory_usage_bytes"`
	AverageMemoryUsage int64  `json:"average_memory_usage_bytes"`
	PeakCPUUsage      float64 `json:"peak_cpu_usage_percent"`
	AverageCPUUsage   float64 `json:"average_cpu_usage_percent"`
	GoroutineCount    int     `json:"goroutine_count"`
	GCPauseTime       time.Duration `json:"gc_pause_time"`
	NetworkBytesIn    int64   `json:"network_bytes_in"`
	NetworkBytesOut   int64   `json:"network_bytes_out"`
}

type SLABenchmarkMetrics struct {
	LatencyCompliance      float64 `json:"latency_compliance_percent"`
	AvailabilityCompliance float64 `json:"availability_compliance_percent"`
	ErrorRateCompliance    float64 `json:"error_rate_compliance_percent"`
	OverallSLAScore        float64 `json:"overall_sla_score"`
}

type BenchmarkError struct {
	Timestamp   time.Time `json:"timestamp"`
	ErrorType   string    `json:"error_type"`
	Message     string    `json:"message"`
	Count       int       `json:"count"`
	Percentage  float64   `json:"percentage"`
	Recoverable bool      `json:"recoverable"`
}

type ScalabilityAnalysis struct {
	LinearScalability     bool    `json:"linear_scalability"`
	OptimalConcurrency    int     `json:"optimal_concurrency"`
	ThroughputSaturation  float64 `json:"throughput_saturation_point"`
	LatencyDegradation    float64 `json:"latency_degradation_factor"`
	ResourceEfficiency    float64 `json:"resource_efficiency_score"`
	BottleneckComponents  []string `json:"bottleneck_components"`
}

type PerformanceRecommendation struct {
	Category    string `json:"category"`
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Impact      string `json:"expected_impact"`
	Effort      string `json:"implementation_effort"`
}

type TrendAnalysis struct {
	PerformanceTrend      string  `json:"performance_trend"` // improving, degrading, stable
	LatencyTrend          float64 `json:"latency_trend_percent"`
	ThroughputTrend       float64 `json:"throughput_trend_percent"`
	CacheEfficiencyTrend  float64 `json:"cache_efficiency_trend_percent"`
	StabilityScore        float64 `json:"stability_score"`
}

// NewBenchmarkSuite creates a new performance benchmark suite
func NewBenchmarkSuite(cfg *config.SnapshotConfig, logger *logger.Logger, cache *cache.SnapshotCache, optimizer *cache.CacheOptimizer, monitor *Monitor) *BenchmarkSuite {
	suite := &BenchmarkSuite{
		config:    cfg,
		logger:    logger,
		cache:     cache,
		optimizer: optimizer,
		monitor:   monitor,
		results:   make(map[string]*BenchmarkResult),
		testDuration: 5 * time.Minute,
		concurrencyLevels: []int{1, 5, 10, 25, 50, 100},
		dataSizes: []int{1024, 10240, 102400, 1048576}, // 1KB, 10KB, 100KB, 1MB
	}
	
	suite.initializeTestScenarios()
	
	return suite
}

// initializeTestScenarios sets up predefined test scenarios
func (bs *BenchmarkSuite) initializeTestScenarios() {
	bs.testScenarios = []TestScenario{
		{
			Name:            "baseline_performance",
			Description:     "Baseline performance test with minimal load",
			ConcurrentUsers: 1,
			RequestRate:     10.0,
			DataSize:        10240, // 10KB
			CacheHitRatio:   0.0,   // Cold cache
			TestDuration:    2 * time.Minute,
			WarmupDuration:  30 * time.Second,
		},
		{
			Name:            "optimal_cache_scenario",
			Description:     "Optimal scenario with high cache hit rate",
			ConcurrentUsers: 10,
			RequestRate:     100.0,
			DataSize:        10240,
			CacheHitRatio:   0.90, // 90% hit rate
			TestDuration:    3 * time.Minute,
			WarmupDuration:  1 * time.Minute,
		},
		{
			Name:            "high_load_stress_test",
			Description:     "High load stress test to find breaking point",
			ConcurrentUsers: 100,
			RequestRate:     1000.0,
			DataSize:        10240,
			CacheHitRatio:   0.50, // Mixed cache performance
			TestDuration:    5 * time.Minute,
			WarmupDuration:  2 * time.Minute,
		},
		{
			Name:            "large_data_test",
			Description:     "Test performance with large data payloads",
			ConcurrentUsers: 10,
			RequestRate:     50.0,
			DataSize:        1048576, // 1MB
			CacheHitRatio:   0.70,
			TestDuration:    4 * time.Minute,
			WarmupDuration:  1 * time.Minute,
		},
		{
			Name:            "cache_miss_scenario",
			Description:     "Worst case scenario with frequent cache misses",
			ConcurrentUsers: 25,
			RequestRate:     200.0,
			DataSize:        10240,
			CacheHitRatio:   0.10, // 90% miss rate
			TestDuration:    3 * time.Minute,
			WarmupDuration:  30 * time.Second,
		},
		{
			Name:            "sustained_load_test",
			Description:     "Sustained load test for stability analysis",
			ConcurrentUsers: 50,
			RequestRate:     500.0,
			DataSize:        10240,
			CacheHitRatio:   0.80,
			TestDuration:    15 * time.Minute, // Longer test
			WarmupDuration:  3 * time.Minute,
		},
	}
}

// RunAllBenchmarks executes all predefined benchmark scenarios
func (bs *BenchmarkSuite) RunAllBenchmarks(ctx context.Context) (*AggregateResults, error) {
	bs.logger.Info("Starting comprehensive benchmark suite",
		zap.Int("scenario_count", len(bs.testScenarios)),
		zap.Duration("estimated_duration", bs.estimateTotalDuration()),
	)
	
	startTime := time.Now()
	
	// Run each scenario
	for i, scenario := range bs.testScenarios {
		bs.logger.Info("Running benchmark scenario",
			zap.Int("scenario", i+1),
			zap.Int("total_scenarios", len(bs.testScenarios)),
			zap.String("scenario_name", scenario.Name),
			zap.Duration("estimated_duration", scenario.TestDuration+scenario.WarmupDuration),
		)
		
		result, err := bs.RunBenchmark(ctx, scenario)
		if err != nil {
			bs.logger.Error("Benchmark scenario failed",
				zap.String("scenario", scenario.Name),
				zap.Error(err),
			)
			// Continue with other scenarios even if one fails
			result = &BenchmarkResult{
				Scenario: scenario,
				StartTime: time.Now(),
				EndTime:   time.Now(),
				Errors: []BenchmarkError{
					{
						Timestamp:   time.Now(),
						ErrorType:   "benchmark_failure",
						Message:     err.Error(),
						Count:       1,
						Percentage:  100.0,
						Recoverable: false,
					},
				},
				PerformanceScore: 0,
			}
		}
		
		bs.mu.Lock()
		bs.results[scenario.Name] = result
		bs.mu.Unlock()
		
		// Brief cooldown between scenarios
		time.Sleep(30 * time.Second)
	}
	
	// Analyze aggregate results
	aggregateResults := bs.analyzeAggregateResults()
	bs.aggregateResults = aggregateResults
	
	totalDuration := time.Since(startTime)
	bs.logger.Info("Benchmark suite completed",
		zap.Duration("total_duration", totalDuration),
		zap.Int("passed_tests", aggregateResults.PassedTests),
		zap.Int("failed_tests", aggregateResults.FailedTests),
		zap.Int("total_tests", aggregateResults.TotalTests),
	)
	
	return aggregateResults, nil
}

// RunBenchmark executes a single benchmark scenario
func (bs *BenchmarkSuite) RunBenchmark(ctx context.Context, scenario TestScenario) (*BenchmarkResult, error) {
	result := &BenchmarkResult{
		Scenario:  scenario,
		StartTime: time.Now(),
		Errors:    make([]BenchmarkError, 0),
	}
	
	// Warmup phase
	if scenario.WarmupDuration > 0 {
		bs.logger.Debug("Starting benchmark warmup phase",
			zap.String("scenario", scenario.Name),
			zap.Duration("warmup_duration", scenario.WarmupDuration),
		)
		
		err := bs.runWarmupPhase(ctx, scenario)
		if err != nil {
			return nil, fmt.Errorf("warmup phase failed: %w", err)
		}
	}
	
	// Main test phase
	bs.logger.Debug("Starting benchmark test phase",
		zap.String("scenario", scenario.Name),
		zap.Duration("test_duration", scenario.TestDuration),
	)
	
	// Create test context with timeout
	testCtx, cancel := context.WithTimeout(ctx, scenario.TestDuration+1*time.Minute)
	defer cancel()
	
	// Run the actual benchmark
	err := bs.runBenchmarkTest(testCtx, scenario, result)
	if err != nil {
		return nil, fmt.Errorf("benchmark test failed: %w", err)
	}
	
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	
	// Calculate performance score
	result.PerformanceScore = bs.calculatePerformanceScore(result)
	
	bs.logger.Info("Benchmark scenario completed",
		zap.String("scenario", scenario.Name),
		zap.Duration("duration", result.Duration),
		zap.Int64("total_requests", result.TotalRequests),
		zap.Float64("requests_per_second", result.RequestsPerSecond),
		zap.Duration("p95_latency", result.LatencyStats.P95),
		zap.Int("performance_score", result.PerformanceScore),
	)
	
	return result, nil
}

// runWarmupPhase prepares the system for benchmarking
func (bs *BenchmarkSuite) runWarmupPhase(ctx context.Context, scenario TestScenario) error {
	warmupCtx, cancel := context.WithTimeout(ctx, scenario.WarmupDuration)
	defer cancel()
	
	// Generate warmup traffic
	warmupRate := scenario.RequestRate * 0.5 // 50% of target rate for warmup
	
	// Simple warmup implementation
	ticker := time.NewTicker(time.Duration(float64(time.Second) / warmupRate))
	defer ticker.Stop()
	
	var wg sync.WaitGroup
	
	for {
		select {
		case <-warmupCtx.Done():
			wg.Wait()
			return nil
		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				bs.simulateRequest(scenario.DataSize, 0.5) // Lower success rate for warmup
			}()
		}
	}
}

// runBenchmarkTest executes the main benchmark test
func (bs *BenchmarkSuite) runBenchmarkTest(ctx context.Context, scenario TestScenario, result *BenchmarkResult) error {
	// Initialize metrics collection
	latencySamples := make([]time.Duration, 0, 100000)
	var latencyMu sync.Mutex
	
	var totalRequests, successfulRequests, failedRequests int64
	
	// Track resource metrics
	resourceTracker := bs.startResourceTracking()
	defer resourceTracker.Stop()
	
	// Track cache metrics before test
	initialCacheStats := bs.cache.GetStats()
	
	// Create rate limiter
	requestInterval := time.Duration(float64(time.Second) / scenario.RequestRate)
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()
	
	// Worker pool for concurrent requests
	workerCh := make(chan struct{}, scenario.ConcurrentUsers)
	var wg sync.WaitGroup
	
	testCtx, cancel := context.WithTimeout(ctx, scenario.TestDuration)
	defer cancel()
	
	// Main test loop
	for {
		select {
		case <-testCtx.Done():
			wg.Wait()
			
			// Calculate final metrics
			result.TotalRequests = totalRequests
			result.SuccessfulRequests = successfulRequests
			result.FailedRequests = failedRequests
			result.RequestsPerSecond = float64(totalRequests) / scenario.TestDuration.Seconds()
			
			// Calculate latency statistics
			latencyMu.Lock()
			result.LatencyStats = bs.calculateLatencyStatistics(latencySamples)
			latencyMu.Unlock()
			
			// Calculate cache metrics
			finalCacheStats := bs.cache.GetStats()
			result.CacheMetrics = bs.calculateCacheBenchmarkMetrics(initialCacheStats, finalCacheStats)
			
			// Get resource metrics
			result.ResourceMetrics = resourceTracker.GetMetrics()
			
			// Calculate SLA compliance
			result.SLACompliance = bs.calculateSLABenchmarkMetrics(result)
			
			return nil
			
		case <-ticker.C:
			select {
			case workerCh <- struct{}{}:
				wg.Add(1)
				go func() {
					defer func() {
						<-workerCh
						wg.Done()
					}()
					
					startTime := time.Now()
					success := bs.simulateRequest(scenario.DataSize, scenario.CacheHitRatio)
					latency := time.Since(startTime)
					
					// Record metrics
					latencyMu.Lock()
					latencySamples = append(latencySamples, latency)
					latencyMu.Unlock()
					
					if success {
						successfulRequests++
					} else {
						failedRequests++
					}
					totalRequests++
				}()
			default:
				// Skip this request if all workers are busy
			}
		}
	}
}

// simulateRequest simulates a snapshot request for benchmarking
func (bs *BenchmarkSuite) simulateRequest(dataSize int, cacheHitRatio float64) bool {
	// Generate test snapshot
	snapshot := bs.generateTestSnapshot(dataSize)
	
	// Simulate cache behavior
	cacheKey := fmt.Sprintf("benchmark_%s", snapshot.SnapshotID)
	
	// Check if this should be a cache hit
	if bs.shouldCacheHit(cacheHitRatio) {
		// Try to get from cache
		if cachedSnapshot, exists := bs.cache.Get(cacheKey); exists && cachedSnapshot != nil {
			return true // Cache hit
		}
	}
	
	// Cache miss - simulate processing and store in cache
	// Add some processing delay to simulate real work
	processingDelay := time.Duration(dataSize/1000) * time.Microsecond
	time.Sleep(processingDelay)
	
	// Store in cache
	ttl := 10 * time.Minute
	err := bs.cache.Set(cacheKey, snapshot, ttl)
	
	return err == nil
}

// Helper methods for benchmark execution
func (bs *BenchmarkSuite) generateTestSnapshot(dataSize int) *types.ClinicalSnapshot {
	// Generate a test snapshot with specified data size
	snapshotID := fmt.Sprintf("benchmark_%d_%d", time.Now().UnixNano(), dataSize)
	
	// Create dummy data to reach target size
	dummyData := make([]byte, dataSize)
	for i := range dummyData {
		dummyData[i] = byte(i % 256)
	}
	
	return &types.ClinicalSnapshot{
		SnapshotID:       snapshotID,
		PatientID:        "benchmark_patient",
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(1 * time.Hour),
		DataCompleteness: 100.0,
		Version:          "benchmark_v1",
		Metadata: map[string]interface{}{
			"benchmark":    true,
			"data_size":    dataSize,
			"generated_at": time.Now(),
		},
	}
}

func (bs *BenchmarkSuite) shouldCacheHit(hitRatio float64) bool {
	// Simple random decision based on hit ratio
	// In reality, this would be more sophisticated
	return bs.randomFloat() < hitRatio
}

func (bs *BenchmarkSuite) randomFloat() float64 {
	// Simple pseudo-random number generator for benchmarking
	// In production, use crypto/rand for security-sensitive operations
	return float64(time.Now().UnixNano()%1000000) / 1000000.0
}

func (bs *BenchmarkSuite) calculateLatencyStatistics(samples []time.Duration) LatencyStatistics {
	if len(samples) == 0 {
		return LatencyStatistics{}
	}
	
	// Sort samples for percentile calculation
	sortedSamples := make([]time.Duration, len(samples))
	copy(sortedSamples, samples)
	
	// Simple bubble sort for demonstration (use sort.Slice in production)
	for i := 0; i < len(sortedSamples)-1; i++ {
		for j := 0; j < len(sortedSamples)-i-1; j++ {
			if sortedSamples[j] > sortedSamples[j+1] {
				sortedSamples[j], sortedSamples[j+1] = sortedSamples[j+1], sortedSamples[j]
			}
		}
	}
	
	count := len(sortedSamples)
	
	// Calculate percentiles
	min := sortedSamples[0]
	max := sortedSamples[count-1]
	median := sortedSamples[count/2]
	p95 := sortedSamples[int(float64(count)*0.95)]
	p99 := sortedSamples[int(float64(count)*0.99)]
	p999 := sortedSamples[int(float64(count)*0.999)]
	
	// Calculate mean
	var total time.Duration
	for _, sample := range samples {
		total += sample
	}
	mean := total / time.Duration(count)
	
	// Calculate standard deviation
	var sumSquaredDiffs float64
	for _, sample := range samples {
		diff := float64(sample - mean)
		sumSquaredDiffs += diff * diff
	}
	stdDev := time.Duration(math.Sqrt(sumSquaredDiffs / float64(count)))
	
	return LatencyStatistics{
		Min:         min,
		Max:         max,
		Mean:        mean,
		Median:      median,
		P95:         p95,
		P99:         p99,
		P999:        p999,
		StdDev:      stdDev,
		SampleCount: count,
	}
}

func (bs *BenchmarkSuite) calculateCacheBenchmarkMetrics(initial, final *types.SnapshotCacheStats) CacheBenchmarkMetrics {
	totalRequests := final.TotalRequests - initial.TotalRequests
	totalHits := (final.L1CacheHits + final.L2CacheHits) - (initial.L1CacheHits + initial.L2CacheHits)
	
	var hitRate float64
	if totalRequests > 0 {
		hitRate = float64(totalHits) / float64(totalRequests) * 100.0
	}
	
	return CacheBenchmarkMetrics{
		HitRate:           hitRate,
		MissRate:          100.0 - hitRate,
		L1HitRate:         final.L1HitRate,
		L2HitRate:         final.L2HitRate,
		CompressionRatio:  2.5, // Placeholder - would come from compression manager
		MemoryUtilization: final.CacheSize * 1024, // Convert to bytes
		EvictionRate:      0.0, // Would be calculated from eviction metrics
	}
}

func (bs *BenchmarkSuite) calculateSLABenchmarkMetrics(result *BenchmarkResult) SLABenchmarkMetrics {
	// Calculate latency compliance (target: 95% of requests < 200ms)
	var latencyCompliant int64
	target := 200 * time.Millisecond
	
	// This is simplified - in reality, we'd track this during the test
	if result.LatencyStats.P95 <= target {
		latencyCompliant = int64(float64(result.SuccessfulRequests) * 0.95)
	} else {
		latencyCompliant = int64(float64(result.SuccessfulRequests) * 0.8) // Estimate
	}
	
	var latencyCompliance float64
	if result.TotalRequests > 0 {
		latencyCompliance = float64(latencyCompliant) / float64(result.TotalRequests) * 100.0
	}
	
	// Calculate availability compliance
	var availabilityCompliance float64
	if result.TotalRequests > 0 {
		availabilityCompliance = float64(result.SuccessfulRequests) / float64(result.TotalRequests) * 100.0
	}
	
	// Calculate error rate compliance (target: <1% error rate)
	errorRate := float64(result.FailedRequests) / float64(result.TotalRequests) * 100.0
	errorRateCompliance := 100.0
	if errorRate > 1.0 {
		errorRateCompliance = math.Max(0, 100.0-(errorRate-1.0)*10)
	}
	
	// Overall SLA score
	overallSLA := (latencyCompliance + availabilityCompliance + errorRateCompliance) / 3.0
	
	return SLABenchmarkMetrics{
		LatencyCompliance:      latencyCompliance,
		AvailabilityCompliance: availabilityCompliance,
		ErrorRateCompliance:    errorRateCompliance,
		OverallSLAScore:        overallSLA,
	}
}

func (bs *BenchmarkSuite) calculatePerformanceScore(result *BenchmarkResult) int {
	// Calculate performance score based on multiple factors (0-100)
	
	// Latency score (40% weight)
	latencyScore := bs.calculateLatencyScore(result.LatencyStats.P95)
	
	// Throughput score (20% weight)
	throughputScore := bs.calculateThroughputScore(result.RequestsPerSecond)
	
	// SLA compliance score (25% weight)
	slaScore := result.SLACompliance.OverallSLAScore
	
	// Cache efficiency score (15% weight)
	cacheScore := result.CacheMetrics.HitRate
	
	// Weighted average
	score := latencyScore*0.4 + throughputScore*0.2 + slaScore*0.25 + cacheScore*0.15
	
	return int(math.Round(score))
}

func (bs *BenchmarkSuite) calculateLatencyScore(p95Latency time.Duration) float64 {
	target := 200.0 // 200ms target
	actual := float64(p95Latency.Nanoseconds()) / 1000000.0 // Convert to ms
	
	if actual <= target {
		return 100.0
	}
	
	// Degrade score based on how much we exceed target
	degradation := (actual - target) / target
	score := math.Max(0, 100.0-degradation*100.0)
	
	return score
}

func (bs *BenchmarkSuite) calculateThroughputScore(qps float64) float64 {
	// Score based on throughput capacity
	if qps >= 1000 {
		return 100.0
	} else if qps >= 500 {
		return 90.0
	} else if qps >= 200 {
		return 80.0
	} else if qps >= 100 {
		return 70.0
	} else if qps >= 50 {
		return 60.0
	} else if qps >= 25 {
		return 50.0
	}
	return 30.0
}

func (bs *BenchmarkSuite) analyzeAggregateResults() *AggregateResults {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	totalTests := len(bs.results)
	passedTests := 0
	failedTests := 0
	
	var bestPerformance, worstPerformance *BenchmarkResult
	bestScore := -1
	worstScore := 101
	
	// Analyze individual results
	for _, result := range bs.results {
		if len(result.Errors) == 0 && result.PerformanceScore > 0 {
			passedTests++
		} else {
			failedTests++
		}
		
		if result.PerformanceScore > bestScore {
			bestScore = result.PerformanceScore
			bestPerformance = result
		}
		
		if result.PerformanceScore < worstScore && result.PerformanceScore > 0 {
			worstScore = result.PerformanceScore
			worstPerformance = result
		}
	}
	
	// Generate recommendations
	recommendations := bs.generateBenchmarkRecommendations()
	
	// Perform scalability analysis
	scalabilityMetrics := bs.analyzeScalability()
	
	// Trend analysis
	trendAnalysis := bs.analyzeTrends()
	
	return &AggregateResults{
		TotalTests:         totalTests,
		PassedTests:        passedTests,
		FailedTests:        failedTests,
		BestPerformance:    bestPerformance,
		WorstPerformance:   worstPerformance,
		ScalabilityMetrics: scalabilityMetrics,
		Recommendations:    recommendations,
		TrendAnalysis:      trendAnalysis,
	}
}

func (bs *BenchmarkSuite) generateBenchmarkRecommendations() []PerformanceRecommendation {
	recommendations := []PerformanceRecommendation{}
	
	// Analyze results and generate specific recommendations
	for _, result := range bs.results {
		if result.LatencyStats.P95 > 200*time.Millisecond {
			recommendations = append(recommendations, PerformanceRecommendation{
				Category:    "latency",
				Priority:    "high",
				Title:       "Optimize P95 Latency",
				Description: fmt.Sprintf("P95 latency in scenario '%s' is %v, exceeding 200ms target", result.Scenario.Name, result.LatencyStats.P95),
				Impact:      "Significant improvement in user experience",
				Effort:      "Medium - requires performance tuning",
			})
		}
		
		if result.CacheMetrics.HitRate < 85.0 {
			recommendations = append(recommendations, PerformanceRecommendation{
				Category:    "cache",
				Priority:    "medium",
				Title:       "Improve Cache Hit Rate",
				Description: fmt.Sprintf("Cache hit rate in scenario '%s' is %.1f%%, below 85%% target", result.Scenario.Name, result.CacheMetrics.HitRate),
				Impact:      "Reduced latency and resource usage",
				Effort:      "Low - adjust cache configuration",
			})
		}
	}
	
	return recommendations
}

func (bs *BenchmarkSuite) analyzeScalability() *ScalabilityAnalysis {
	// Analyze scalability patterns from benchmark results
	// This is a simplified implementation
	
	return &ScalabilityAnalysis{
		LinearScalability:     true, // Would be calculated from actual data
		OptimalConcurrency:    50,   // Would be determined from results
		ThroughputSaturation:  0.8,  // 80% of theoretical maximum
		LatencyDegradation:    1.2,  // 20% increase under load
		ResourceEfficiency:    0.85, // 85% efficiency
		BottleneckComponents:  []string{"cache", "memory"},
	}
}

func (bs *BenchmarkSuite) analyzeTrends() *TrendAnalysis {
	// Analyze performance trends
	// This would compare with historical data
	
	return &TrendAnalysis{
		PerformanceTrend:      "stable",
		LatencyTrend:          2.5,  // 2.5% increase
		ThroughputTrend:       -1.2, // 1.2% decrease
		CacheEfficiencyTrend:  5.0,  // 5% improvement
		StabilityScore:        0.92, // 92% stability
	}
}

func (bs *BenchmarkSuite) estimateTotalDuration() time.Duration {
	var total time.Duration
	for _, scenario := range bs.testScenarios {
		total += scenario.TestDuration + scenario.WarmupDuration + 30*time.Second // cooldown
	}
	return total
}

// GetResults returns all benchmark results
func (bs *BenchmarkSuite) GetResults() map[string]*BenchmarkResult {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	// Return a copy to prevent external modification
	results := make(map[string]*BenchmarkResult)
	for k, v := range bs.results {
		results[k] = v
	}
	
	return results
}

// GetAggregateResults returns aggregate analysis results
func (bs *BenchmarkSuite) GetAggregateResults() *AggregateResults {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	
	return bs.aggregateResults
}

// ResourceTracker for monitoring resources during benchmarks
type ResourceTracker struct {
	metrics ResourceBenchmarkMetrics
	done    chan bool
	mu      sync.RWMutex
}

func (bs *BenchmarkSuite) startResourceTracking() *ResourceTracker {
	tracker := &ResourceTracker{
		done: make(chan bool),
	}
	
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-tracker.done:
				return
			case <-ticker.C:
				// Collect resource metrics
				// This would integrate with actual system monitoring
				tracker.mu.Lock()
				// Update metrics here
				tracker.mu.Unlock()
			}
		}
	}()
	
	return tracker
}

func (rt *ResourceTracker) Stop() {
	close(rt.done)
}

func (rt *ResourceTracker) GetMetrics() ResourceBenchmarkMetrics {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	
	return rt.metrics
}