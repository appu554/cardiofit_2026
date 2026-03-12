package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/cache"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/performance"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/metrics"
)

// Command-line flags
var (
	configPath     = flag.String("config", "config/snapshot_config.yaml", "Path to configuration file")
	benchmarkMode  = flag.String("mode", "comprehensive", "Benchmark mode: comprehensive, quick, stress, cache")
	duration       = flag.Duration("duration", 5*time.Minute, "Test duration")
	concurrency    = flag.Int("concurrency", 10, "Concurrent users")
	optimizationLevel = flag.String("optimization", "standard", "Optimization level: basic, standard, aggressive, maximum")
	verbose        = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Initialize logger
	var zapLogger *zap.Logger
	var err error
	
	if *verbose {
		zapLogger, err = zap.NewDevelopment()
	} else {
		zapLogger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer zapLogger.Sync()

	logger := logger.New(zapLogger)

	// Load configuration
	cfg, err := loadConfiguration(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize metrics collector
	metricsRegistry := metrics.NewRegistry()
	metricsCollector := metrics.NewSnapshotMetricsCollector(logger.GetZapLogger(), metricsRegistry)

	// Initialize cache system
	cacheConfig := config.GetDefaultCacheConfig()
	snapshotCache, err := cache.NewSnapshotCache(cacheConfig, logger)
	if err != nil {
		logger.Fatal("Failed to create snapshot cache", zap.Error(err))
	}
	defer snapshotCache.Close()

	// Initialize Phase 3 Performance System
	phase3System, err := performance.NewPhase3PerformanceSystem(cfg, logger, metricsCollector, snapshotCache)
	if err != nil {
		logger.Fatal("Failed to create Phase 3 performance system", zap.Error(err))
	}

	// Set optimization level
	level := parseOptimizationLevel(*optimizationLevel)
	if err := phase3System.SetOptimizationLevel(level); err != nil {
		logger.Error("Failed to set optimization level", zap.Error(err))
	}

	// Start the performance system
	if err := phase3System.Start(); err != nil {
		logger.Fatal("Failed to start Phase 3 system", zap.Error(err))
	}
	defer phase3System.Stop()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, gracefully shutting down...")
		cancel()
	}()

	// Run benchmark based on mode
	switch *benchmarkMode {
	case "comprehensive":
		runComprehensiveBenchmark(ctx, phase3System, logger)
	case "quick":
		runQuickBenchmark(ctx, phase3System, logger)
	case "stress":
		runStressBenchmark(ctx, phase3System, logger, *concurrency)
	case "cache":
		runCacheBenchmark(ctx, phase3System, logger)
	default:
		logger.Fatal("Unknown benchmark mode", zap.String("mode", *benchmarkMode))
	}

	logger.Info("Performance testing completed successfully")
}

func runComprehensiveBenchmark(ctx context.Context, system *performance.Phase3PerformanceSystem, logger *logger.Logger) {
	logger.Info("Starting comprehensive performance benchmark")

	// Get initial status
	initialStatus := system.GetStatus()
	logger.Info("Initial system status",
		zap.String("performance_grade", initialStatus.PerformanceGrade),
		zap.Int("targets_achieved", initialStatus.TargetsAchieved),
		zap.Int("total_targets", initialStatus.TotalTargets),
	)

	// Run comprehensive benchmark
	benchmarkCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	results, err := system.RunPerformanceBenchmark(benchmarkCtx)
	if err != nil {
		logger.Error("Benchmark failed", zap.Error(err))
		return
	}

	// Display results
	displayBenchmarkResults(results, logger)

	// Generate performance report
	report := system.GetPerformanceReport()
	logger.Info("Performance report generated", zap.Any("report", report))

	// Get final status
	finalStatus := system.GetStatus()
	logger.Info("Final system status",
		zap.String("performance_grade", finalStatus.PerformanceGrade),
		zap.Int("targets_achieved", finalStatus.TargetsAchieved),
		zap.Int("total_targets", finalStatus.TotalTargets),
		zap.Strings("recommendations", finalStatus.Recommendations),
	)
}

func runQuickBenchmark(ctx context.Context, system *performance.Phase3PerformanceSystem, logger *logger.Logger) {
	logger.Info("Starting quick performance benchmark", zap.Duration("duration", *duration))

	// Create a quick benchmark scenario
	benchmarkCtx, cancel := context.WithTimeout(ctx, *duration+2*time.Minute)
	defer cancel()

	// Monitor performance for the specified duration
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	var samples []performance.SystemPerformanceMetrics

	for {
		select {
		case <-benchmarkCtx.Done():
			logger.Info("Quick benchmark completed")
			displayQuickResults(samples, logger)
			return
		case <-ticker.C:
			status := system.GetStatus()
			if status.CurrentMetrics != nil {
				samples = append(samples, *status.CurrentMetrics)
				logger.Info("Performance sample",
					zap.Duration("elapsed", time.Since(startTime)),
					zap.Duration("p95_latency", status.CurrentMetrics.P95Latency),
					zap.Float64("cache_hit_rate", status.CurrentMetrics.CacheHitRate),
					zap.Float64("sla_compliance", status.CurrentMetrics.SLACompliance),
					zap.Int("performance_score", status.CurrentMetrics.PerformanceScore),
				)
			}
		}
	}
}

func runStressBenchmark(ctx context.Context, system *performance.Phase3PerformanceSystem, logger *logger.Logger, maxConcurrency int) {
	logger.Info("Starting stress benchmark", 
		zap.Int("max_concurrency", maxConcurrency),
		zap.Duration("duration", *duration),
	)

	// Gradually increase load
	steps := []int{1, 5, 10, 25, 50, maxConcurrency}
	stepDuration := *duration / time.Duration(len(steps))

	for i, concurrency := range steps {
		logger.Info("Stress test step",
			zap.Int("step", i+1),
			zap.Int("concurrency", concurrency),
			zap.Duration("step_duration", stepDuration),
		)

		// Simulate load for this step
		stepCtx, cancel := context.WithTimeout(ctx, stepDuration)
		
		// Monitor performance during this step
		go simulateLoad(stepCtx, concurrency, logger)
		
		// Collect metrics
		ticker := time.NewTicker(10 * time.Second)
		stepStart := time.Now()
		
		for {
			select {
			case <-stepCtx.Done():
				ticker.Stop()
				cancel()
				goto nextStep
			case <-ticker.C:
				status := system.GetStatus()
				if status.CurrentMetrics != nil {
					logger.Info("Stress test metrics",
						zap.Int("concurrency", concurrency),
						zap.Duration("step_elapsed", time.Since(stepStart)),
						zap.Duration("p95_latency", status.CurrentMetrics.P95Latency),
						zap.Float64("throughput_qps", status.CurrentMetrics.ThroughputQPS),
						zap.Int("performance_score", status.CurrentMetrics.PerformanceScore),
					)
				}
			}
		}
		
		nextStep:
		// Brief cooldown between steps
		time.Sleep(10 * time.Second)
	}

	logger.Info("Stress benchmark completed")
}

func runCacheBenchmark(ctx context.Context, system *performance.Phase3PerformanceSystem, logger *logger.Logger) {
	logger.Info("Starting cache performance benchmark")

	// Test different cache scenarios
	scenarios := []struct {
		name     string
		hitRatio float64
		duration time.Duration
	}{
		{"cold_cache", 0.1, 2 * time.Minute},
		{"warm_cache", 0.5, 2 * time.Minute},
		{"hot_cache", 0.9, 2 * time.Minute},
	}

	for _, scenario := range scenarios {
		logger.Info("Running cache scenario",
			zap.String("scenario", scenario.name),
			zap.Float64("target_hit_ratio", scenario.hitRatio),
			zap.Duration("duration", scenario.duration),
		)

		// Simulate cache scenario
		scenarioCtx, cancel := context.WithTimeout(ctx, scenario.duration)
		
		// Monitor cache performance
		ticker := time.NewTicker(15 * time.Second)
		scenarioStart := time.Now()
		
		for {
			select {
			case <-scenarioCtx.Done():
				ticker.Stop()
				cancel()
				logger.Info("Cache scenario completed", zap.String("scenario", scenario.name))
				goto nextScenario
			case <-ticker.C:
				status := system.GetStatus()
				if status.CurrentMetrics != nil {
					logger.Info("Cache scenario metrics",
						zap.String("scenario", scenario.name),
						zap.Duration("elapsed", time.Since(scenarioStart)),
						zap.Float64("cache_hit_rate", status.CurrentMetrics.CacheHitRate),
						zap.Duration("p95_latency", status.CurrentMetrics.P95Latency),
						zap.Float64("compression_ratio", status.CurrentMetrics.CompressionRatio),
					)
				}
			}
		}
		
		nextScenario:
		// Brief cooldown between scenarios
		time.Sleep(30 * time.Second)
	}

	logger.Info("Cache benchmark completed")
}

func displayBenchmarkResults(results *performance.AggregateResults, logger *logger.Logger) {
	logger.Info("=== BENCHMARK RESULTS ===")
	logger.Info("Test Summary",
		zap.Int("total_tests", results.TotalTests),
		zap.Int("passed_tests", results.PassedTests),
		zap.Int("failed_tests", results.FailedTests),
	)

	if results.BestPerformance != nil {
		logger.Info("Best Performance",
			zap.String("scenario", results.BestPerformance.Scenario.Name),
			zap.Duration("p95_latency", results.BestPerformance.LatencyStats.P95),
			zap.Duration("p99_latency", results.BestPerformance.LatencyStats.P99),
			zap.Float64("requests_per_second", results.BestPerformance.RequestsPerSecond),
			zap.Float64("cache_hit_rate", results.BestPerformance.CacheMetrics.HitRate),
			zap.Int("performance_score", results.BestPerformance.PerformanceScore),
		)
	}

	if results.WorstPerformance != nil {
		logger.Info("Worst Performance",
			zap.String("scenario", results.WorstPerformance.Scenario.Name),
			zap.Duration("p95_latency", results.WorstPerformance.LatencyStats.P95),
			zap.Duration("p99_latency", results.WorstPerformance.LatencyStats.P99),
			zap.Float64("requests_per_second", results.WorstPerformance.RequestsPerSecond),
			zap.Float64("cache_hit_rate", results.WorstPerformance.CacheMetrics.HitRate),
			zap.Int("performance_score", results.WorstPerformance.PerformanceScore),
		)
	}

	if results.ScalabilityMetrics != nil {
		logger.Info("Scalability Analysis",
			zap.Bool("linear_scalability", results.ScalabilityMetrics.LinearScalability),
			zap.Int("optimal_concurrency", results.ScalabilityMetrics.OptimalConcurrency),
			zap.Float64("throughput_saturation", results.ScalabilityMetrics.ThroughputSaturation),
			zap.Float64("resource_efficiency", results.ScalabilityMetrics.ResourceEfficiency),
			zap.Strings("bottlenecks", results.ScalabilityMetrics.BottleneckComponents),
		)
	}

	logger.Info("Recommendations")
	for i, rec := range results.Recommendations {
		logger.Info("Recommendation",
			zap.Int("index", i+1),
			zap.String("category", rec.Category),
			zap.String("priority", rec.Priority),
			zap.String("title", rec.Title),
			zap.String("description", rec.Description),
		)
	}
}

func displayQuickResults(samples []performance.SystemPerformanceMetrics, logger *logger.Logger) {
	if len(samples) == 0 {
		logger.Info("No performance samples collected")
		return
	}

	// Calculate summary statistics
	var totalLatency time.Duration
	var totalCacheHitRate, totalSLACompliance float64
	var totalScore int
	minLatency := samples[0].P95Latency
	maxLatency := samples[0].P95Latency

	for _, sample := range samples {
		totalLatency += sample.P95Latency
		totalCacheHitRate += sample.CacheHitRate
		totalSLACompliance += sample.SLACompliance
		totalScore += sample.PerformanceScore

		if sample.P95Latency < minLatency {
			minLatency = sample.P95Latency
		}
		if sample.P95Latency > maxLatency {
			maxLatency = sample.P95Latency
		}
	}

	count := len(samples)
	avgLatency := totalLatency / time.Duration(count)
	avgCacheHitRate := totalCacheHitRate / float64(count)
	avgSLACompliance := totalSLACompliance / float64(count)
	avgScore := totalScore / count

	logger.Info("=== QUICK BENCHMARK RESULTS ===")
	logger.Info("Performance Summary",
		zap.Int("sample_count", count),
		zap.Duration("avg_p95_latency", avgLatency),
		zap.Duration("min_p95_latency", minLatency),
		zap.Duration("max_p95_latency", maxLatency),
		zap.Float64("avg_cache_hit_rate", avgCacheHitRate),
		zap.Float64("avg_sla_compliance", avgSLACompliance),
		zap.Int("avg_performance_score", avgScore),
	)

	// Performance assessment
	var assessment string
	if avgLatency <= 200*time.Millisecond && avgCacheHitRate >= 85.0 && avgSLACompliance >= 95.0 {
		assessment = "EXCELLENT - All targets achieved"
	} else if avgLatency <= 300*time.Millisecond && avgCacheHitRate >= 75.0 && avgSLACompliance >= 90.0 {
		assessment = "GOOD - Most targets achieved"
	} else if avgLatency <= 500*time.Millisecond && avgCacheHitRate >= 65.0 && avgSLACompliance >= 80.0 {
		assessment = "FAIR - Some optimization needed"
	} else {
		assessment = "POOR - Significant optimization required"
	}

	logger.Info("Performance Assessment", zap.String("assessment", assessment))
}

func simulateLoad(ctx context.Context, concurrency int, logger *logger.Logger) {
	// Simple load simulation
	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// Simulate work
					time.Sleep(time.Duration(50+workerID*10) * time.Millisecond)
				}
			}
		}(i)
	}
}

func parseOptimizationLevel(level string) performance.OptimizationLevel {
	switch level {
	case "basic":
		return performance.OptimizationLevelBasic
	case "standard":
		return performance.OptimizationLevelStandard
	case "aggressive":
		return performance.OptimizationLevelAggressive
	case "maximum":
		return performance.OptimizationLevelMaximum
	default:
		return performance.OptimizationLevelStandard
	}
}

func loadConfiguration(path string) (*config.SnapshotConfig, error) {
	// For this example, we'll use default configuration
	// In production, this would load from the specified file
	cfg := config.GetDefaultSnapshotConfig()
	cfg.Enabled = true
	return cfg, nil
}