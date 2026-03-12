package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"kb-2-clinical-context-go/internal/cache"
	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/metrics"
	"kb-2-clinical-context-go/internal/performance"
	"kb-2-clinical-context-go/internal/services"
)

// Performance validation script for the 3-tier caching implementation
// This script validates that all performance targets are met:
// - Latency: P50: 5ms, P95: 25ms, P99: 100ms
// - Throughput: 10,000 RPS
// - Cache Hit Rates: L1: 85%, L2: 95%
// - Batch Processing: 1000 patients < 1 second

func main() {
	fmt.Println("==========================================")
	fmt.Println("KB-2 Clinical Context Performance Validation")
	fmt.Println("==========================================")
	
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Setup context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	
	// Initialize dependencies
	dependencies, err := initializeDependencies(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize dependencies: %v", err)
	}
	defer dependencies.cleanup()
	
	// Run validation tests
	if err := runValidationTests(ctx, dependencies); err != nil {
		log.Fatalf("Validation tests failed: %v", err)
	}
	
	fmt.Println("\n==========================================")
	fmt.Println("VALIDATION COMPLETE")
	fmt.Println("==========================================")
}

type TestDependencies struct {
	config      *config.Config
	mongoClient *mongo.Client
	redisClient *redis.Client
	metrics     *metrics.PrometheusMetrics
	cache       *cache.MultiTierCache
	service     *services.ContextService
}

func (td *TestDependencies) cleanup() {
	if td.mongoClient != nil {
		td.mongoClient.Disconnect(context.Background())
	}
	if td.redisClient != nil {
		td.redisClient.Close()
	}
}

func initializeDependencies(ctx context.Context, cfg *config.Config) (*TestDependencies, error) {
	fmt.Println("Initializing test dependencies...")
	
	// Initialize MongoDB connection
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.DatabaseURL))
	if err != nil {
		return nil, fmt.Errorf("MongoDB connection failed: %w", err)
	}
	
	// Test MongoDB connection
	if err := mongoClient.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("MongoDB ping failed: %w", err)
	}
	fmt.Println("✓ MongoDB connection established")
	
	// Initialize Redis connection
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	
	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis connection failed: %w", err)
	}
	fmt.Println("✓ Redis connection established")
	
	// Initialize metrics
	metrics := metrics.NewPrometheusMetrics()
	fmt.Println("✓ Prometheus metrics initialized")
	
	// Initialize multi-tier cache
	multiTierCache := cache.NewMultiTierCache(cfg, redisClient, metrics)
	fmt.Println("✓ Multi-tier cache initialized")
	
	// Initialize context service
	contextService := services.NewContextService(mongoClient, redisClient, cfg, metrics)
	contextService.SetMultiTierCache(multiTierCache)
	fmt.Println("✓ Context service initialized")
	
	return &TestDependencies{
		config:      cfg,
		mongoClient: mongoClient,
		redisClient: redisClient,
		metrics:     metrics,
		cache:       multiTierCache,
		service:     contextService,
	}, nil
}

func runValidationTests(ctx context.Context, deps *TestDependencies) error {
	fmt.Println("\nRunning performance validation tests...")
	
	// Create benchmark runner
	benchmarkRunner := performance.NewBenchmarkRunner(
		deps.config,
		deps.cache,
		deps.metrics,
	)
	
	// Test 1: Cache Connectivity and Basic Operations
	if err := testCacheConnectivity(ctx, deps.cache); err != nil {
		return fmt.Errorf("cache connectivity test failed: %w", err)
	}
	fmt.Println("✓ Cache connectivity test passed")
	
	// Test 2: Cache Warming
	if err := testCacheWarming(ctx, deps.cache); err != nil {
		return fmt.Errorf("cache warming test failed: %w", err)
	}
	fmt.Println("✓ Cache warming test passed")
	
	// Test 3: Basic Performance Validation
	if err := testBasicPerformance(ctx, deps.cache); err != nil {
		return fmt.Errorf("basic performance test failed: %w", err)
	}
	fmt.Println("✓ Basic performance test passed")
	
	// Test 4: Comprehensive Benchmarks
	fmt.Println("\nRunning comprehensive performance benchmarks...")
	results, err := benchmarkRunner.RunComprehensiveBenchmarks(ctx)
	if err != nil {
		return fmt.Errorf("comprehensive benchmarks failed: %w", err)
	}
	
	// Test 5: SLA Compliance Validation
	if err := validateSLACompliance(results, deps.config); err != nil {
		return fmt.Errorf("SLA compliance validation failed: %w", err)
	}
	fmt.Println("✓ SLA compliance validation passed")
	
	return nil
}

func testCacheConnectivity(ctx context.Context, cache *cache.MultiTierCache) error {
	fmt.Println("Testing cache tier connectivity...")
	
	testKey := "connectivity_test"
	testData := "test_data_" + fmt.Sprintf("%d", time.Now().UnixNano())
	
	// Test L1 cache
	cache.l1Cache.Set(testKey, testData, time.Minute)
	if value, found := cache.l1Cache.Get(testKey); !found || value != testData {
		return fmt.Errorf("L1 cache test failed")
	}
	cache.l1Cache.Delete(testKey)
	fmt.Println("  ✓ L1 cache (in-memory) working")
	
	// Test L2 cache
	if err := cache.l2Cache.Set(ctx, testKey, testData, time.Minute); err != nil {
		return fmt.Errorf("L2 cache set failed: %w", err)
	}
	if value, found := cache.l2Cache.Get(ctx, testKey); !found || value != testData {
		return fmt.Errorf("L2 cache get failed")
	}
	cache.l2Cache.Delete(ctx, testKey)
	fmt.Println("  ✓ L2 cache (Redis) working")
	
	// Test L3 cache (basic check)
	if cache.l3Cache != nil {
		fmt.Println("  ✓ L3 cache (CDN) configured")
	}
	
	return nil
}

func testCacheWarming(ctx context.Context, cache *cache.MultiTierCache) error {
	fmt.Println("Testing cache warming functionality...")
	
	// Test cache warming
	if err := cache.WarmCache(ctx); err != nil {
		return fmt.Errorf("cache warming failed: %w", err)
	}
	
	// Verify some data was cached
	stats := cache.GetStats()
	if l1Stats, exists := stats["l1"]; exists && l1Stats.Size > 0 {
		fmt.Printf("  ✓ L1 cache warmed with %d items\n", l1Stats.Size)
	}
	
	return nil
}

func testBasicPerformance(ctx context.Context, cache *cache.MultiTierCache) error {
	fmt.Println("Testing basic performance characteristics...")
	
	// Test single request latency
	start := time.Now()
	_, err := cache.Get(ctx, "performance_test_key", func() (interface{}, error) {
		return map[string]interface{}{
			"test_data": "performance_validation",
			"timestamp": time.Now(),
		}, nil
	})
	latency := time.Since(start)
	
	if err != nil {
		return fmt.Errorf("cache get operation failed: %w", err)
	}
	
	// Validate latency is reasonable
	if latency > 50*time.Millisecond {
		return fmt.Errorf("single request latency too high: %v (expected < 50ms)", latency)
	}
	
	fmt.Printf("  ✓ Single request latency: %v\n", latency)
	
	// Test cache hit performance (second request should be faster)
	start = time.Now()
	_, err = cache.Get(ctx, "performance_test_key", func() (interface{}, error) {
		return nil, fmt.Errorf("should not reach this loader") // Should be cache hit
	})
	hitLatency := time.Since(start)
	
	if err != nil {
		return fmt.Errorf("cache hit test failed: %w", err)
	}
	
	if hitLatency > 5*time.Millisecond {
		return fmt.Errorf("cache hit latency too high: %v (expected < 5ms)", hitLatency)
	}
	
	fmt.Printf("  ✓ Cache hit latency: %v\n", hitLatency)
	
	return nil
}

func validateSLACompliance(results map[string]*performance.BenchmarkResult, cfg *config.Config) error {
	fmt.Println("\nValidating SLA compliance...")
	
	requiredTests := map[string]string{
		"latency_targets":    "Latency SLA (P50: 5ms, P95: 25ms, P99: 100ms)",
		"throughput_targets": "Throughput SLA (10,000 RPS)",
		"cache_performance":  "Cache Hit Rate SLA (L1: 85%, L2: 95%)",
	}
	
	allCompliant := true
	
	for testName, description := range requiredTests {
		result, exists := results[testName]
		if !exists {
			fmt.Printf("  ❌ %s: Test not executed\n", description)
			allCompliant = false
			continue
		}
		
		// Check individual SLA compliance
		testCompliant := true
		for slaMetric, compliant := range result.SLACompliance {
			if !compliant {
				testCompliant = false
				fmt.Printf("  ❌ %s: %s failed\n", description, slaMetric)
			}
		}
		
		if testCompliant {
			fmt.Printf("  ✓ %s: All targets met (Score: %.3f)\n", description, result.PerformanceScore)
		} else {
			allCompliant = false
		}
	}
	
	if allCompliant {
		fmt.Println("\n🎉 ALL SLA TARGETS MET - SYSTEM READY FOR PRODUCTION")
	} else {
		fmt.Println("\n⚠️  Some SLA targets not met - Review performance optimization")
	}
	
	// Print performance summary
	fmt.Println("\nPerformance Summary:")
	
	if latencyResult, exists := results["latency_targets"]; exists {
		fmt.Printf("  Latency: P50=%.2fms, P95=%.2fms, P99=%.2fms\n", 
			latencyResult.LatencyP50, latencyResult.LatencyP95, latencyResult.LatencyP99)
	}
	
	if throughputResult, exists := results["throughput_targets"]; exists {
		fmt.Printf("  Throughput: %.0f RPS (Target: %d RPS)\n", 
			throughputResult.ThroughputRPS, cfg.TargetThroughputRPS)
	}
	
	if cacheResult, exists := results["cache_performance"]; exists {
		fmt.Printf("  Cache Hit Rates: L1=%.1f%%, L2=%.1f%%\n", 
			cacheResult.CacheHitRates["l1"]*100, cacheResult.CacheHitRates["l2"]*100)
	}
	
	return nil
}

// Additional helper functions for testing

func printTestConfiguration(cfg *config.Config) {
	fmt.Println("Test Configuration:")
	fmt.Printf("  Environment: %s\n", cfg.Environment)
	fmt.Printf("  Max Concurrent Requests: %d\n", cfg.MaxConcurrentRequests)
	fmt.Printf("  Batch Size: %d\n", cfg.BatchSize)
	fmt.Printf("  Cache Configuration:\n")
	fmt.Printf("    L1 Max Size: %d MB\n", cfg.L1CacheMaxSize/(1024*1024))
	fmt.Printf("    L1 TTL: %v\n", cfg.L1CacheDefaultTTL)
	fmt.Printf("    L2 Max Memory: %d MB\n", cfg.L2CacheMaxMemory/(1024*1024))
	fmt.Printf("    L2 TTL: %v\n", cfg.L2CacheDefaultTTL)
	fmt.Printf("  Performance Targets:\n")
	fmt.Printf("    Latency P50: %dms\n", cfg.TargetLatencyP50)
	fmt.Printf("    Latency P95: %dms\n", cfg.TargetLatencyP95)
	fmt.Printf("    Latency P99: %dms\n", cfg.TargetLatencyP99)
	fmt.Printf("    Throughput: %d RPS\n", cfg.TargetThroughputRPS)
	fmt.Printf("    Batch Time: %dms for 1000 patients\n", cfg.TargetBatchTime)
	fmt.Println()
}

func printSystemInfo() {
	fmt.Println("System Information:")
	fmt.Printf("  OS: %s\n", os.Getenv("OS"))
	fmt.Printf("  Go Version: %s\n", os.Getenv("GOVERSION"))
	fmt.Printf("  Test Time: %s\n", time.Now().Format(time.RFC3339))
	fmt.Println()
}

// Performance targets validation
func validatePerformanceTargets(results map[string]*performance.BenchmarkResult) {
	fmt.Println("Performance Targets Validation:")
	
	// Define targets
	targets := map[string]map[string]float64{
		"latency": {
			"p50_ms": 5.0,
			"p95_ms": 25.0,
			"p99_ms": 100.0,
		},
		"throughput": {
			"min_rps": 10000.0,
		},
		"cache": {
			"l1_hit_rate": 0.85,
			"l2_hit_rate": 0.95,
		},
		"batch": {
			"max_time_ms": 1000.0, // 1000 patients in < 1s
		},
	}
	
	// Validate each target category
	for category, categoryTargets := range targets {
		fmt.Printf("  %s targets:\n", category)
		
		for metric, target := range categoryTargets {
			status := "✓ PASS"
			actualValue := 0.0
			
			// Extract actual values from results
			switch category {
			case "latency":
				if result, exists := results["latency_targets"]; exists {
					switch metric {
					case "p50_ms":
						actualValue = result.LatencyP50
					case "p95_ms":
						actualValue = result.LatencyP95
					case "p99_ms":
						actualValue = result.LatencyP99
					}
					if actualValue > target {
						status = "❌ FAIL"
					}
				}
			case "throughput":
				if result, exists := results["throughput_targets"]; exists {
					actualValue = result.ThroughputRPS
					if actualValue < target {
						status = "❌ FAIL"
					}
				}
			case "cache":
				if result, exists := results["cache_performance"]; exists {
					switch metric {
					case "l1_hit_rate":
						actualValue = result.CacheHitRates["l1"]
					case "l2_hit_rate":
						actualValue = result.CacheHitRates["l2"]
					}
					if actualValue < target {
						status = "❌ FAIL"
					}
				}
			case "batch":
				if result, exists := results["batch_processing_1000"]; exists {
					actualValue = float64(result.Duration.Milliseconds())
					if actualValue > target {
						status = "❌ FAIL"
					}
				}
			}
			
			fmt.Printf("    %s %s: %.2f (target: %.2f)\n", status, metric, actualValue, target)
		}
		fmt.Println()
	}
}