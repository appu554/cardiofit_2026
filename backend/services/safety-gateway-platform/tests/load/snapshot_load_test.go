package load

import (
	"context"
	"fmt"
	"math"
	"math/rand"
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

// SnapshotLoadTestSuite provides comprehensive load testing for snapshot systems
// Tests high-concurrency scenarios with realistic clinical workloads
type SnapshotLoadTestSuite struct {
	suite.Suite
	
	// Core services
	contextBuilder    *context.ContextBuilder
	assemblyService   *context.AssemblyService
	cacheManager      *context.CacheManager
	snapshotCache     *cache.SnapshotCache
	tokenGenerator    *override.EnhancedTokenGenerator
	eventPublisher    *learning.LearningEventPublisher
	
	// Load testing components
	loadGenerator     *LoadGenerator
	metricsCollector  *LoadMetricsCollector
	scenarioRunner    *ScenarioRunner
	
	// Test configuration
	testConfig       *config.Config
	loadConfig       *LoadTestConfig
	logger           *zap.Logger
	ctx              context.Context
	cancel           context.CancelFunc
}

// LoadGenerator manages concurrent load generation
type LoadGenerator struct {
	scenarios        map[string]*LoadScenario
	workerPools      map[string]*WorkerPool
	requestGenerators map[string]*RequestGenerator
	logger           *zap.Logger
}

// LoadScenario defines a specific load testing scenario
type LoadScenario struct {
	Name                string
	Description         string
	Duration           time.Duration
	ConcurrentUsers    int
	RequestsPerSecond  int
	RampUpTime         time.Duration
	RampDownTime       time.Duration
	DistributionType   string // constant, linear, spike, wave
	UserBehavior       *UserBehavior
	PerformanceTargets *PerformanceTargets
}

// UserBehavior defines realistic user interaction patterns
type UserBehavior struct {
	ThinkTime          time.Duration
	SessionDuration    time.Duration
	RequestPatterns    []*RequestPattern
	ErrorRate          float64
	CacheHitRate       float64
	OverrideRate       float64
}

// RequestPattern defines a pattern of requests a user might make
type RequestPattern struct {
	Name         string
	Weight       float64 // Probability weight
	Requests     []*RequestTemplate
	Timing       string // sequential, parallel, random
}

// RequestTemplate defines a request template
type RequestTemplate struct {
	Type         string
	Complexity   string
	PatientType  string
	DataVolume   string
	ExpectedTime time.Duration
}

// WorkerPool manages concurrent workers for load generation
type WorkerPool struct {
	name          string
	workers       []*LoadWorker
	requestChan   chan *LoadRequest
	resultChan    chan *LoadResult
	activeWorkers int32
	totalWorkers  int
	logger        *zap.Logger
}

// LoadWorker represents a single concurrent worker
type LoadWorker struct {
	id            int
	pool          *LoadWorkerPool
	requestChan   chan *LoadRequest
	resultChan    chan *LoadResult
	isActive      bool
	stats         *WorkerStats
	logger        *zap.Logger
}

// LoadRequest represents a single load test request
type LoadRequest struct {
	ID            int64
	Scenario      string
	Type          string
	PatientID     string
	Complexity    string
	Timestamp     time.Time
	UserSession   string
	Metadata      map[string]interface{}
}

// LoadResult represents the result of a load test request
type LoadResult struct {
	RequestID     int64
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration
	Success       bool
	ErrorType     string
	ErrorMessage  string
	ResponseSize  int
	CacheHit      bool
	Metrics       *RequestMetrics
}

// RequestMetrics contains detailed metrics for a single request
type RequestMetrics struct {
	QueueTime         time.Duration
	ProcessingTime    time.Duration
	NetworkTime       time.Duration
	CacheAccessTime   time.Duration
	DatabaseTime      time.Duration
	ComputeTime       time.Duration
	MemoryUsage       int64
	CPUUsage          float64
}

// LoadMetricsCollector aggregates and analyzes load test metrics
type LoadMetricsCollector struct {
	results           []*LoadResult
	aggregatedMetrics *AggregatedMetrics
	realTimeMetrics   *RealTimeMetrics
	mu                sync.RWMutex
	logger            *zap.Logger
}

// AggregatedMetrics contains summarized load test results
type AggregatedMetrics struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	RequestsPerSecond  float64
	
	// Latency metrics
	MinLatency         time.Duration
	MaxLatency         time.Duration
	MeanLatency        time.Duration
	P50Latency         time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	P999Latency        time.Duration
	
	// Error metrics
	ErrorRate          float64
	ErrorBreakdown     map[string]int64
	
	// Resource metrics
	PeakMemoryUsage    int64
	AverageCPUUsage    float64
	PeakCPUUsage       float64
	
	// Cache metrics
	CacheHitRate       float64
	CacheHitLatency    time.Duration
	CacheMissLatency   time.Duration
	
	// Business metrics
	OverrideRate       float64
	SnapshotCreationRate float64
	DataIntegrityScore float64
}

// RealTimeMetrics provides real-time monitoring during load tests
type RealTimeMetrics struct {
	CurrentRPS         float64
	ActiveUsers        int32
	CurrentLatency     time.Duration
	CurrentErrorRate   float64
	LastUpdateTime     time.Time
	
	// Moving averages
	RPS1Min            float64
	RPS5Min            float64
	Latency1Min        time.Duration
	Latency5Min        time.Duration
	ErrorRate1Min      float64
	ErrorRate5Min      float64
}

// LoadTestConfig defines load test configuration
type LoadTestConfig struct {
	// Global settings
	MaxConcurrentUsers  int
	TestDuration       time.Duration
	WarmUpDuration     time.Duration
	CoolDownDuration   time.Duration
	
	// Performance targets
	TargetRPS          int
	MaxLatencyP95      time.Duration
	MaxErrorRate       float64
	MinCacheHitRate    float64
	
	// Resource limits
	MaxMemoryUsage     int64
	MaxCPUUsage        float64
	
	// Clinical scenarios
	PatientDistribution map[string]float64 // patient_type -> percentage
	ComplexityDistribution map[string]float64 // complexity -> percentage
}

// PerformanceTargets defines expected performance for scenarios
type PerformanceTargets struct {
	MaxP95Latency      time.Duration
	MinThroughput      int
	MaxErrorRate       float64
	MinAvailability    float64
	MaxMemoryUsage     int64
	ResourceUtilization float64
}

// SetupSuite initializes the load testing environment
func (s *SnapshotLoadTestSuite) SetupSuite() {
	s.logger = zaptest.NewLogger(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	
	// Load load test configuration
	s.loadConfig = s.loadLoadTestConfig()
	s.testConfig = s.loadServiceConfig()
	
	// Initialize services with load test optimizations
	s.setupServicesForLoad()
	
	// Initialize load testing components
	s.setupLoadTestingComponents()
	
	s.T().Log("Load test suite initialized")
}

// TearDownSuite cleans up the load testing environment
func (s *SnapshotLoadTestSuite) TearDownSuite() {
	s.cancel()
	s.generateLoadTestReport()
	s.T().Log("Load test suite completed")
}

// TestBasicLoadCapacity tests basic system capacity under steady load
func (s *SnapshotLoadTestSuite) TestBasicLoadCapacity() {
	scenario := &LoadScenario{
		Name:               "Basic Load Capacity",
		Description:        "Steady load with realistic clinical request patterns",
		Duration:           3 * time.Minute,
		ConcurrentUsers:    50,
		RequestsPerSecond:  100,
		RampUpTime:         30 * time.Second,
		RampDownTime:       15 * time.Second,
		DistributionType:   "constant",
		UserBehavior: &UserBehavior{
			ThinkTime:       2 * time.Second,
			SessionDuration: 10 * time.Minute,
			RequestPatterns: s.createBasicRequestPatterns(),
			ErrorRate:       0.01, // 1% expected error rate
			CacheHitRate:    0.80, // 80% cache hit rate
			OverrideRate:    0.05, // 5% override rate
		},
		PerformanceTargets: &PerformanceTargets{
			MaxP95Latency:   200 * time.Millisecond,
			MinThroughput:   90, // 90% of target RPS
			MaxErrorRate:    0.02, // 2% max error rate
			MinAvailability: 0.99, // 99% availability
			MaxMemoryUsage:  1 * 1024 * 1024 * 1024, // 1GB
		},
	}
	
	s.runLoadScenario(scenario)
}

// TestPeakLoadCapacity tests system behavior under peak load
func (s *SnapshotLoadTestSuite) TestPeakLoadCapacity() {
	scenario := &LoadScenario{
		Name:               "Peak Load Capacity",
		Description:        "High concurrent load testing system limits",
		Duration:           5 * time.Minute,
		ConcurrentUsers:    200,
		RequestsPerSecond:  500,
		RampUpTime:         60 * time.Second,
		RampDownTime:       30 * time.Second,
		DistributionType:   "linear",
		UserBehavior: &UserBehavior{
			ThinkTime:       1 * time.Second,
			SessionDuration: 15 * time.Minute,
			RequestPatterns: s.createIntensiveRequestPatterns(),
			ErrorRate:       0.02,
			CacheHitRate:    0.85,
			OverrideRate:    0.08,
		},
		PerformanceTargets: &PerformanceTargets{
			MaxP95Latency:   400 * time.Millisecond,
			MinThroughput:   80, // Allow some degradation
			MaxErrorRate:    0.05, // 5% max error rate
			MinAvailability: 0.95, // 95% availability
			MaxMemoryUsage:  2 * 1024 * 1024 * 1024, // 2GB
		},
	}
	
	s.runLoadScenario(scenario)
}

// TestSpikeLoadResilience tests system resilience under traffic spikes
func (s *SnapshotLoadTestSuite) TestSpikeLoadResilience() {
	scenario := &LoadScenario{
		Name:               "Spike Load Resilience",
		Description:        "Sudden traffic spikes and recovery testing",
		Duration:           4 * time.Minute,
		ConcurrentUsers:    100,
		RequestsPerSecond:  300,
		RampUpTime:         10 * time.Second, // Quick spike
		RampDownTime:       60 * time.Second, // Gradual recovery
		DistributionType:   "spike",
		UserBehavior: &UserBehavior{
			ThinkTime:       500 * time.Millisecond,
			SessionDuration: 5 * time.Minute,
			RequestPatterns: s.createSpikeRequestPatterns(),
			ErrorRate:       0.03,
			CacheHitRate:    0.70, // Lower hit rate during spikes
			OverrideRate:    0.10, // Higher override rate during emergencies
		},
		PerformanceTargets: &PerformanceTargets{
			MaxP95Latency:   800 * time.Millisecond, // Allow degradation during spikes
			MinThroughput:   60, // Maintain 60% throughput
			MaxErrorRate:    0.10, // 10% max error rate during spikes
			MinAvailability: 0.90, // 90% availability
			MaxMemoryUsage:  3 * 1024 * 1024 * 1024, // 3GB during spikes
		},
	}
	
	s.runLoadScenario(scenario)
}

// TestSustainedLoadEndurance tests long-term system endurance
func (s *SnapshotLoadTestSuite) TestSustainedLoadEndurance() {
	if testing.Short() {
		s.T().Skip("Skipping endurance test in short mode")
		return
	}
	
	scenario := &LoadScenario{
		Name:               "Sustained Load Endurance",
		Description:        "Long-term system endurance and stability testing",
		Duration:           20 * time.Minute,
		ConcurrentUsers:    75,
		RequestsPerSecond:  150,
		RampUpTime:         2 * time.Minute,
		RampDownTime:       2 * time.Minute,
		DistributionType:   "constant",
		UserBehavior: &UserBehavior{
			ThinkTime:       3 * time.Second,
			SessionDuration: 30 * time.Minute,
			RequestPatterns: s.createEnduranceRequestPatterns(),
			ErrorRate:       0.01,
			CacheHitRate:    0.88, // High cache hit rate for sustained load
			OverrideRate:    0.04,
		},
		PerformanceTargets: &PerformanceTargets{
			MaxP95Latency:   250 * time.Millisecond,
			MinThroughput:   95, // Should maintain high throughput
			MaxErrorRate:    0.02, // Low error rate
			MinAvailability: 0.99, // High availability
			MaxMemoryUsage:  1.5 * 1024 * 1024 * 1024, // 1.5GB
		},
	}
	
	s.runLoadScenario(scenario)
}

// TestComplexClinicalWorkloads tests realistic clinical scenarios
func (s *SnapshotLoadTestSuite) TestComplexClinicalWorkloads() {
	scenario := &LoadScenario{
		Name:               "Complex Clinical Workloads",
		Description:        "Realistic clinical decision support workloads",
		Duration:           6 * time.Minute,
		ConcurrentUsers:    120,
		RequestsPerSecond:  250,
		RampUpTime:         45 * time.Second,
		RampDownTime:       30 * time.Second,
		DistributionType:   "wave",
		UserBehavior: &UserBehavior{
			ThinkTime:       2 * time.Second,
			SessionDuration: 12 * time.Minute,
			RequestPatterns: s.createClinicalRequestPatterns(),
			ErrorRate:       0.015,
			CacheHitRate:    0.82,
			OverrideRate:    0.06,
		},
		PerformanceTargets: &PerformanceTargets{
			MaxP95Latency:   300 * time.Millisecond,
			MinThroughput:   85,
			MaxErrorRate:    0.03,
			MinAvailability: 0.97,
			MaxMemoryUsage:  2 * 1024 * 1024 * 1024, // 2GB
		},
	}
	
	s.runLoadScenario(scenario)
}

// TestConcurrentOverrideGeneration tests concurrent override token generation
func (s *SnapshotLoadTestSuite) TestConcurrentOverrideGeneration() {
	scenario := &LoadScenario{
		Name:               "Concurrent Override Generation",
		Description:        "High concurrency override token generation load",
		Duration:           3 * time.Minute,
		ConcurrentUsers:    80,
		RequestsPerSecond:  200,
		RampUpTime:         20 * time.Second,
		RampDownTime:       20 * time.Second,
		DistributionType:   "constant",
		UserBehavior: &UserBehavior{
			ThinkTime:       1 * time.Second,
			SessionDuration: 8 * time.Minute,
			RequestPatterns: s.createOverrideRequestPatterns(),
			ErrorRate:       0.02,
			CacheHitRate:    0.75, // Lower cache hit for override scenarios
			OverrideRate:    0.80, // High override rate for this test
		},
		PerformanceTargets: &PerformanceTargets{
			MaxP95Latency:   150 * time.Millisecond, // Token generation should be fast
			MinThroughput:   90,
			MaxErrorRate:    0.03,
			MinAvailability: 0.98,
			MaxMemoryUsage:  1 * 1024 * 1024 * 1024, // 1GB
		},
	}
	
	s.runLoadScenario(scenario)
}

// runLoadScenario executes a load testing scenario
func (s *SnapshotLoadTestSuite) runLoadScenario(scenario *LoadScenario) {
	s.T().Logf("Starting load scenario: %s", scenario.Name)
	
	// Initialize metrics collector for this scenario
	s.metricsCollector = NewLoadMetricsCollector(scenario.Name, s.logger)
	
	// Start real-time monitoring
	monitoringCtx, monitoringCancel := context.WithCancel(s.ctx)
	go s.startRealTimeMonitoring(monitoringCtx)
	
	// Configure and start the scenario
	err := s.scenarioRunner.ConfigureScenario(scenario)
	require.NoError(s.T(), err, "Failed to configure scenario")
	
	// Execute the load test
	testCtx, testCancel := context.WithTimeout(s.ctx, scenario.Duration+scenario.RampUpTime+scenario.RampDownTime)
	defer testCancel()
	
	results, err := s.scenarioRunner.ExecuteScenario(testCtx, scenario)
	require.NoError(s.T(), err, "Failed to execute scenario")
	
	// Stop monitoring
	monitoringCancel()
	
	// Collect and analyze results
	s.metricsCollector.ProcessResults(results)
	aggregatedMetrics := s.metricsCollector.GetAggregatedMetrics()
	
	// Validate against performance targets
	s.validatePerformanceTargets(scenario, aggregatedMetrics)
	
	// Log scenario summary
	s.logScenarioSummary(scenario, aggregatedMetrics)
}

// validatePerformanceTargets validates results against performance targets
func (s *SnapshotLoadTestSuite) validatePerformanceTargets(
	scenario *LoadScenario,
	metrics *AggregatedMetrics,
) {
	targets := scenario.PerformanceTargets
	
	// Validate latency
	assert.LessOrEqual(s.T(), metrics.P95Latency, targets.MaxP95Latency,
		fmt.Sprintf("P95 latency %v exceeded target %v", metrics.P95Latency, targets.MaxP95Latency))
	
	// Validate throughput
	actualThroughputPercent := (metrics.RequestsPerSecond / float64(scenario.RequestsPerSecond)) * 100
	assert.GreaterOrEqual(s.T(), actualThroughputPercent, float64(targets.MinThroughput),
		fmt.Sprintf("Throughput %.1f%% below target %d%%", actualThroughputPercent, targets.MinThroughput))
	
	// Validate error rate
	assert.LessOrEqual(s.T(), metrics.ErrorRate, targets.MaxErrorRate,
		fmt.Sprintf("Error rate %.4f exceeded target %.4f", metrics.ErrorRate, targets.MaxErrorRate))
	
	// Validate availability
	availability := float64(metrics.SuccessfulRequests) / float64(metrics.TotalRequests)
	assert.GreaterOrEqual(s.T(), availability, targets.MinAvailability,
		fmt.Sprintf("Availability %.4f below target %.4f", availability, targets.MinAvailability))
	
	// Validate memory usage
	assert.LessOrEqual(s.T(), metrics.PeakMemoryUsage, targets.MaxMemoryUsage,
		fmt.Sprintf("Peak memory %d exceeded target %d", metrics.PeakMemoryUsage, targets.MaxMemoryUsage))
	
	s.T().Logf("Performance targets validation completed for %s", scenario.Name)
}

// Helper methods for request pattern creation

func (s *SnapshotLoadTestSuite) createBasicRequestPatterns() []*RequestPattern {
	return []*RequestPattern{
		{
			Name:   "Routine Medication Check",
			Weight: 0.60, // 60% of requests
			Requests: []*RequestTemplate{
				{
					Type:         "medication_safety_check",
					Complexity:   "simple",
					PatientType:  "standard",
					DataVolume:   "small",
					ExpectedTime: 100 * time.Millisecond,
				},
			},
			Timing: "sequential",
		},
		{
			Name:   "Complex Patient Assessment",
			Weight: 0.25, // 25% of requests
			Requests: []*RequestTemplate{
				{
					Type:         "comprehensive_assessment",
					Complexity:   "complex",
					PatientType:  "multi_condition",
					DataVolume:   "large",
					ExpectedTime: 200 * time.Millisecond,
				},
			},
			Timing: "sequential",
		},
		{
			Name:   "Emergency Override",
			Weight: 0.15, // 15% of requests
			Requests: []*RequestTemplate{
				{
					Type:         "emergency_override",
					Complexity:   "high",
					PatientType:  "critical",
					DataVolume:   "medium",
					ExpectedTime: 150 * time.Millisecond,
				},
			},
			Timing: "sequential",
		},
	}
}

func (s *SnapshotLoadTestSuite) createIntensiveRequestPatterns() []*RequestPattern {
	return []*RequestPattern{
		{
			Name:   "High-Volume Processing",
			Weight: 0.50,
			Requests: []*RequestTemplate{
				{
					Type:         "batch_processing",
					Complexity:   "high",
					PatientType:  "mixed",
					DataVolume:   "very_large",
					ExpectedTime: 300 * time.Millisecond,
				},
			},
			Timing: "parallel",
		},
		{
			Name:   "Concurrent Assessments",
			Weight: 0.50,
			Requests: []*RequestTemplate{
				{
					Type:         "concurrent_assessment",
					Complexity:   "very_high",
					PatientType:  "complex_multi",
					DataVolume:   "huge",
					ExpectedTime: 400 * time.Millisecond,
				},
			},
			Timing: "parallel",
		},
	}
}

func (s *SnapshotLoadTestSuite) loadLoadTestConfig() *LoadTestConfig {
	return &LoadTestConfig{
		MaxConcurrentUsers:  500,
		TestDuration:       10 * time.Minute,
		WarmUpDuration:     2 * time.Minute,
		CoolDownDuration:   1 * time.Minute,
		TargetRPS:          500,
		MaxLatencyP95:      200 * time.Millisecond,
		MaxErrorRate:       0.01,
		MinCacheHitRate:    0.85,
		MaxMemoryUsage:     2 * 1024 * 1024 * 1024, // 2GB
		MaxCPUUsage:        0.80,
		PatientDistribution: map[string]float64{
			"simple":      0.40,
			"complex":     0.35,
			"critical":    0.15,
			"multi_condition": 0.10,
		},
		ComplexityDistribution: map[string]float64{
			"low":         0.30,
			"medium":      0.40,
			"high":        0.20,
			"very_high":   0.10,
		},
	}
}

func (s *SnapshotLoadTestSuite) generateLoadTestReport() {
	s.T().Logf("\n=== Load Test Report ===")
	if s.metricsCollector != nil {
		metrics := s.metricsCollector.GetAggregatedMetrics()
		s.T().Logf("Total Requests: %d", metrics.TotalRequests)
		s.T().Logf("Successful Requests: %d", metrics.SuccessfulRequests)
		s.T().Logf("Failed Requests: %d", metrics.FailedRequests)
		s.T().Logf("Requests Per Second: %.2f", metrics.RequestsPerSecond)
		s.T().Logf("P95 Latency: %v", metrics.P95Latency)
		s.T().Logf("P99 Latency: %v", metrics.P99Latency)
		s.T().Logf("Error Rate: %.4f%%", metrics.ErrorRate*100)
		s.T().Logf("Cache Hit Rate: %.4f%%", metrics.CacheHitRate*100)
		s.T().Logf("Peak Memory Usage: %.2f MB", float64(metrics.PeakMemoryUsage)/1024/1024)
		s.T().Logf("Average CPU Usage: %.2f%%", metrics.AverageCPUUsage*100)
	}
	s.T().Logf("=== End Load Test Report ===\n")
}

// TestSnapshotLoadTestSuite runs the load test suite
func TestSnapshotLoadTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}
	
	// Check if running on sufficient hardware
	if runtime.NumCPU() < 4 {
		t.Skip("Load tests require at least 4 CPU cores")
	}
	
	suite.Run(t, new(SnapshotLoadTestSuite))
}