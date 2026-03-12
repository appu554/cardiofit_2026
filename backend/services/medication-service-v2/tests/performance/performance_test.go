// +build performance

package performance_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/tests/helpers/fixtures"
	"medication-service-v2/tests/helpers/testsetup"
)

// PerformanceTestSuite validates performance targets for the medication service
type PerformanceTestSuite struct {
	suite.Suite
	
	medicationService *services.MedicationService
	recipeService     *services.RecipeService
	clinicalEngine    *services.ClinicalEngineService
	
	ctx        context.Context
	testRecipe *entities.Recipe
}

func TestPerformanceTestSuite(t *testing.T) {
	if os.Getenv("SKIP_PERFORMANCE_TESTS") == "true" {
		t.Skip("Skipping performance tests")
	}
	
	suite.Run(t, new(PerformanceTestSuite))
}

func (suite *PerformanceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	
	// Setup services with performance-optimized configuration
	suite.setupOptimizedServices()
	
	// Pre-load test data
	suite.setupPerformanceTestData()
}

func (suite *PerformanceTestSuite) setupOptimizedServices() {
	// Use in-memory databases and optimized configurations for performance testing
	testDB := testsetup.SetupOptimizedTestDatabase(suite.T())
	testRedis := testsetup.SetupOptimizedTestRedis(suite.T())
	
	// Setup services with performance optimizations
	medicationRepo := testsetup.SetupOptimizedMedicationRepository(testDB)
	recipeRepo := testsetup.SetupOptimizedRecipeRepository(testDB)
	
	rustEngine := testsetup.SetupOptimizedRustEngine(suite.T())
	apolloClient := testsetup.SetupOptimizedApolloClient(suite.T())
	contextGateway := testsetup.SetupOptimizedContextGateway(suite.T())
	
	auditService := services.NewAuditService(testDB)
	notificationService := services.NewNotificationService()
	
	suite.clinicalEngine = services.NewClinicalEngineService(
		rustEngine,
		apolloClient,
		testRedis,
	)
	
	snapshotService := services.NewSnapshotService(
		contextGateway,
		testRedis,
		testDB,
	)
	
	suite.recipeService = services.NewRecipeService(
		recipeRepo,
		medicationRepo,
		testRedis,
	)
	
	suite.medicationService = services.NewMedicationService(
		medicationRepo,
		suite.recipeService,
		snapshotService,
		suite.clinicalEngine,
		auditService,
		notificationService,
		testsetup.TestLogger(),
		testsetup.TestMetrics(),
	)
}

func (suite *PerformanceTestSuite) setupPerformanceTestData() {
	// Pre-create and cache recipe data
	suite.testRecipe = fixtures.ValidRecipeWithRules()
	
	// Warm up caches
	suite.warmupCaches()
}

func (suite *PerformanceTestSuite) warmupCaches() {
	// Execute a few requests to warm up caches
	warmupRequest := &services.ProposeMedicationRequest{
		PatientID:       uuid.New(),
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Cache warmup",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "warmup",
	}
	
	for i := 0; i < 3; i++ {
		_, _ = suite.medicationService.ProposeMedication(suite.ctx, warmupRequest)
	}
}

func (suite *PerformanceTestSuite) TestMedicationProposalEndToEndPerformance() {
	t := suite.T()
	
	// Performance target: <250ms end-to-end for 95% of requests
	targetDuration := 250 * time.Millisecond
	testCount := 100
	
	durations := make([]time.Duration, 0, testCount)
	successCount := int64(0)
	errorCount := int64(0)
	
	t.Logf("Testing %d medication proposal requests with target <250ms", testCount)
	
	for i := 0; i < testCount; i++ {
		request := &services.ProposeMedicationRequest{
			PatientID:       uuid.New(),
			ProtocolID:      suite.testRecipe.ProtocolID,
			Indication:      fmt.Sprintf("Performance test %d", i+1),
			ClinicalContext: fixtures.ValidClinicalContext(),
			CreatedBy:       "perf-test",
		}
		
		startTime := time.Now()
		response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
		duration := time.Since(startTime)
		
		durations = append(durations, duration)
		
		if err != nil {
			atomic.AddInt64(&errorCount, 1)
			t.Errorf("Request %d failed: %v", i+1, err)
		} else {
			atomic.AddInt64(&successCount, 1)
			require.NotNil(t, response)
			require.NotNil(t, response.Proposal)
		}
	}
	
	// Analyze results
	stats := analyzePerformanceResults(durations)
	
	t.Logf("Performance Results:")
	t.Logf("  Success rate: %.1f%% (%d/%d)", 
		float64(successCount)/float64(testCount)*100, successCount, testCount)
	t.Logf("  Average: %v", stats.Average)
	t.Logf("  Median: %v", stats.Median)
	t.Logf("  95th percentile: %v", stats.P95)
	t.Logf("  99th percentile: %v", stats.P99)
	t.Logf("  Maximum: %v", stats.Max)
	t.Logf("  Minimum: %v", stats.Min)
	
	// Assert performance requirements
	assert.Equal(t, int64(testCount), successCount, "All requests should succeed")
	assert.True(t, stats.P95 < targetDuration, 
		"95th percentile (%v) should be under %v", stats.P95, targetDuration)
	
	// Additional performance assertions
	assert.True(t, stats.Average < targetDuration,
		"Average duration (%v) should be under target", stats.Average)
}

func (suite *PerformanceTestSuite) TestRecipeResolverPerformance() {
	t := suite.T()
	
	// Performance target: <10ms for recipe resolution
	targetDuration := 10 * time.Millisecond
	testCount := 200
	
	durations := make([]time.Duration, 0, testCount)
	
	t.Logf("Testing %d recipe resolution requests with target <10ms", testCount)
	
	for i := 0; i < testCount; i++ {
		request := entities.RecipeResolutionRequest{
			RecipeID:       suite.testRecipe.ID,
			PatientContext: fixtures.ValidPatientContext(),
			Options: entities.ResolutionOptions{
				UseCache:    true,
				CacheTTL:    5 * time.Minute,
				EnableDebug: false,
			},
		}
		
		startTime := time.Now()
		resolution, err := suite.recipeService.ResolveRecipe(suite.ctx, request)
		duration := time.Since(startTime)
		
		durations = append(durations, duration)
		
		require.NoError(t, err, "Recipe resolution %d should succeed", i+1)
		require.NotNil(t, resolution)
		assert.True(t, resolution.ProcessingTimeMs < 10,
			"Recipe resolution %d took %dms, expected <10ms", i+1, resolution.ProcessingTimeMs)
	}
	
	// Analyze results
	stats := analyzePerformanceResults(durations)
	
	t.Logf("Recipe Resolution Performance:")
	t.Logf("  Average: %v", stats.Average)
	t.Logf("  95th percentile: %v", stats.P95)
	t.Logf("  Maximum: %v", stats.Max)
	
	// Assert performance requirements
	assert.True(t, stats.P95 < targetDuration,
		"95th percentile (%v) should be under %v", stats.P95, targetDuration)
}

func (suite *PerformanceTestSuite) TestClinicalEnginePerformance() {
	t := suite.T()
	
	// Performance target for clinical calculations
	targetDuration := 50 * time.Millisecond
	testCount := 150
	
	durations := make([]time.Duration, 0, testCount)
	
	t.Logf("Testing %d clinical engine calculations with target <50ms", testCount)
	
	for i := 0; i < testCount; i++ {
		request := &services.CalculateDosagesRequest{
			Recipe:     suite.testRecipe,
			Snapshot:   fixtures.ValidSnapshot(),
			PatientID:  uuid.New(),
			Parameters: make(map[string]interface{}),
		}
		
		startTime := time.Now()
		response, err := suite.clinicalEngine.CalculateDosages(suite.ctx, request)
		duration := time.Since(startTime)
		
		durations = append(durations, duration)
		
		require.NoError(t, err, "Clinical calculation %d should succeed", i+1)
		require.NotNil(t, response)
		require.NotEmpty(t, response.DosageRecommendations)
	}
	
	// Analyze results
	stats := analyzePerformanceResults(durations)
	
	t.Logf("Clinical Engine Performance:")
	t.Logf("  Average: %v", stats.Average)
	t.Logf("  95th percentile: %v", stats.P95)
	t.Logf("  Maximum: %v", stats.Max)
	
	// Assert performance requirements
	assert.True(t, stats.P95 < targetDuration,
		"95th percentile (%v) should be under %v", stats.P95, targetDuration)
}

func (suite *PerformanceTestSuite) TestThroughputUnderLoad() {
	t := suite.T()
	
	// Throughput target: 1000+ RPS sustained
	targetRPS := 1000.0
	testDuration := 30 * time.Second
	concurrency := 50
	
	var requestCount int64
	var successCount int64
	var errorCount int64
	
	ctx, cancel := context.WithTimeout(suite.ctx, testDuration)
	defer cancel()
	
	var wg sync.WaitGroup
	startTime := time.Now()
	
	t.Logf("Testing throughput with %d concurrent workers for %v", concurrency, testDuration)
	
	// Start concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for {
				select {
				case <-ctx.Done():
					return
				default:
					atomic.AddInt64(&requestCount, 1)
					
					request := &services.ProposeMedicationRequest{
						PatientID:       uuid.New(),
						ProtocolID:      suite.testRecipe.ProtocolID,
						Indication:      fmt.Sprintf("Throughput test - worker %d", workerID),
						ClinicalContext: fixtures.ValidClinicalContext(),
						CreatedBy:       fmt.Sprintf("worker-%d", workerID),
					}
					
					_, err := suite.medicationService.ProposeMedication(context.Background(), request)
					if err != nil {
						atomic.AddInt64(&errorCount, 1)
					} else {
						atomic.AddInt64(&successCount, 1)
					}
				}
			}
		}(i)
	}
	
	wg.Wait()
	actualDuration := time.Since(startTime)
	
	// Calculate throughput metrics
	totalRequests := atomic.LoadInt64(&requestCount)
	successRequests := atomic.LoadInt64(&successCount)
	errorRequests := atomic.LoadInt64(&errorCount)
	
	actualRPS := float64(successRequests) / actualDuration.Seconds()
	successRate := float64(successRequests) / float64(totalRequests) * 100
	
	t.Logf("Throughput Results:")
	t.Logf("  Duration: %v", actualDuration)
	t.Logf("  Total requests: %d", totalRequests)
	t.Logf("  Successful requests: %d", successRequests)
	t.Logf("  Failed requests: %d", errorRequests)
	t.Logf("  Success rate: %.1f%%", successRate)
	t.Logf("  Actual RPS: %.1f", actualRPS)
	t.Logf("  Target RPS: %.1f", targetRPS)
	
	// Assert throughput requirements
	assert.True(t, actualRPS >= targetRPS,
		"Actual RPS (%.1f) should meet target (%.1f)", actualRPS, targetRPS)
	assert.True(t, successRate >= 99.0,
		"Success rate (%.1f%%) should be at least 99%%", successRate)
}

func (suite *PerformanceTestSuite) TestMemoryUsageUnderLoad() {
	t := suite.T()
	
	// Memory target: <512MB per service instance
	targetMemoryMB := 512.0
	
	// Measure baseline memory
	baselineMemory := testsetup.GetMemoryUsageMB()
	
	// Execute load test
	requestCount := 1000
	var wg sync.WaitGroup
	concurrency := 20
	
	t.Logf("Testing memory usage with %d requests across %d workers", requestCount, concurrency)
	
	requestsPerWorker := requestCount / concurrency
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerWorker; j++ {
				request := &services.ProposeMedicationRequest{
					PatientID:       uuid.New(),
					ProtocolID:      suite.testRecipe.ProtocolID,
					Indication:      fmt.Sprintf("Memory test - worker %d request %d", workerID, j),
					ClinicalContext: fixtures.ValidClinicalContext(),
					CreatedBy:       fmt.Sprintf("worker-%d", workerID),
				}
				
				_, _ = suite.medicationService.ProposeMedication(suite.ctx, request)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Force garbage collection and measure memory
	testsetup.ForceGarbageCollection()
	time.Sleep(1 * time.Second)
	
	peakMemory := testsetup.GetMemoryUsageMB()
	memoryIncrease := peakMemory - baselineMemory
	
	t.Logf("Memory Usage Results:")
	t.Logf("  Baseline memory: %.1f MB", baselineMemory)
	t.Logf("  Peak memory: %.1f MB", peakMemory)
	t.Logf("  Memory increase: %.1f MB", memoryIncrease)
	t.Logf("  Target limit: %.1f MB", targetMemoryMB)
	
	// Assert memory requirements
	assert.True(t, peakMemory < targetMemoryMB,
		"Peak memory usage (%.1f MB) should be under target (%.1f MB)", 
		peakMemory, targetMemoryMB)
}

func (suite *PerformanceTestSuite) TestCachePerformanceImprovement() {
	t := suite.T()
	
	// Test cache effectiveness by comparing with and without cache
	testCount := 50
	
	patientContext := fixtures.ValidPatientContext()
	
	// Test without cache
	noCacheDurations := make([]time.Duration, 0, testCount)
	for i := 0; i < testCount; i++ {
		request := entities.RecipeResolutionRequest{
			RecipeID:       suite.testRecipe.ID,
			PatientContext: patientContext,
			Options: entities.ResolutionOptions{
				UseCache:    false, // No cache
				EnableDebug: false,
			},
		}
		
		startTime := time.Now()
		_, err := suite.recipeService.ResolveRecipe(suite.ctx, request)
		duration := time.Since(startTime)
		
		require.NoError(t, err)
		noCacheDurations = append(noCacheDurations, duration)
	}
	
	// Test with cache (should be faster after first request)
	cacheDurations := make([]time.Duration, 0, testCount)
	for i := 0; i < testCount; i++ {
		request := entities.RecipeResolutionRequest{
			RecipeID:       suite.testRecipe.ID,
			PatientContext: patientContext,
			Options: entities.ResolutionOptions{
				UseCache:    true, // Use cache
				CacheTTL:    5 * time.Minute,
				EnableDebug: false,
			},
		}
		
		startTime := time.Now()
		_, err := suite.recipeService.ResolveRecipe(suite.ctx, request)
		duration := time.Since(startTime)
		
		require.NoError(t, err)
		cacheDurations = append(cacheDurations, duration)
	}
	
	// Analyze cache effectiveness (skip first few requests for cache warm-up)
	noCacheStats := analyzePerformanceResults(noCacheDurations)
	cacheStats := analyzePerformanceResults(cacheDurations[5:]) // Skip first 5 for warmup
	
	improvement := float64(noCacheStats.Average-cacheStats.Average) / float64(noCacheStats.Average) * 100
	
	t.Logf("Cache Performance Analysis:")
	t.Logf("  No cache average: %v", noCacheStats.Average)
	t.Logf("  With cache average: %v", cacheStats.Average)
	t.Logf("  Performance improvement: %.1f%%", improvement)
	
	// Cache should provide at least 30% improvement
	assert.True(t, improvement >= 30.0,
		"Cache should provide at least 30%% improvement, got %.1f%%", improvement)
}

// BenchmarkMedicationProposal provides detailed benchmarking
func (suite *PerformanceTestSuite) TestBenchmarkMedicationProposal() {
	t := suite.T()
	
	request := &services.ProposeMedicationRequest{
		PatientID:       uuid.New(),
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Benchmark test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "benchmark",
	}
	
	// Warm up
	for i := 0; i < 5; i++ {
		_, _ = suite.medicationService.ProposeMedication(suite.ctx, request)
	}
	
	// Benchmark
	benchmarkCount := 100
	startTime := time.Now()
	
	for i := 0; i < benchmarkCount; i++ {
		request.PatientID = uuid.New() // Vary patient ID to avoid cache hits
		_, err := suite.medicationService.ProposeMedication(suite.ctx, request)
		require.NoError(t, err)
	}
	
	totalTime := time.Since(startTime)
	averageTime := totalTime / time.Duration(benchmarkCount)
	operationsPerSecond := float64(benchmarkCount) / totalTime.Seconds()
	
	t.Logf("Benchmark Results:")
	t.Logf("  Total time: %v", totalTime)
	t.Logf("  Average time per operation: %v", averageTime)
	t.Logf("  Operations per second: %.1f", operationsPerSecond)
	
	// Should be able to handle at least 20 operations per second
	assert.True(t, operationsPerSecond >= 20.0,
		"Should handle at least 20 ops/sec, got %.1f", operationsPerSecond)
}

// PerformanceStats holds performance analysis results
type PerformanceStats struct {
	Min     time.Duration
	Max     time.Duration
	Average time.Duration
	Median  time.Duration
	P95     time.Duration
	P99     time.Duration
}

func analyzePerformanceResults(durations []time.Duration) PerformanceStats {
	if len(durations) == 0 {
		return PerformanceStats{}
	}
	
	// Sort durations
	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	
	// Simple sorting (bubble sort for small datasets)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	// Calculate statistics
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	
	stats := PerformanceStats{
		Min:     sorted[0],
		Max:     sorted[len(sorted)-1],
		Average: total / time.Duration(len(durations)),
		Median:  sorted[len(sorted)/2],
		P95:     sorted[int(float64(len(sorted))*0.95)],
		P99:     sorted[int(float64(len(sorted))*0.99)],
	}
	
	return stats
}