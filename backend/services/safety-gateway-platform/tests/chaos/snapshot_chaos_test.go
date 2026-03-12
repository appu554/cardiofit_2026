package chaos

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
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

// SnapshotChaosTestSuite provides chaos engineering tests for snapshot resilience
type SnapshotChaosTestSuite struct {
	suite.Suite
	
	// Core services
	contextBuilder    *context.ContextBuilder
	assemblyService   *context.AssemblyService
	cacheManager      *context.CacheManager
	snapshotCache     *cache.SnapshotCache
	tokenGenerator    *override.EnhancedTokenGenerator
	eventPublisher    *learning.LearningEventPublisher
	
	// Chaos components
	chaosController   *ChaosController
	faultInjector     *FaultInjector
	resilenceMetrics  *ResilienceMetrics
	
	// Test configuration
	testConfig       *config.Config
	logger           *zap.Logger
	ctx              context.Context
	cancel           context.CancelFunc
}

// ChaosController manages chaos engineering scenarios
type ChaosController struct {
	scenarios     map[string]*ChaosScenario
	isActive      bool
	mu            sync.RWMutex
	logger        *zap.Logger
}

// ChaosScenario defines a specific chaos engineering test
type ChaosScenario struct {
	Name              string
	Description       string
	Duration          time.Duration
	FaultTypes        []FaultType
	IntensityLevel    int // 1-10 (10 = maximum chaos)
	RecoveryTime      time.Duration
	SuccessCriteria   func(*ResilienceMetrics) bool
}

// FaultType defines different types of faults to inject
type FaultType int

const (
	FaultTypeLatency FaultType = iota
	FaultTypeMemoryLeak
	FaultTypeCacheCorruption
	FaultTypeNetworkPartition
	FaultTypeServiceUnavailable
	FaultTypeDataCorruption
	FaultTypeResourceExhaustion
	FaultTypeCascadingFailure
)

// FaultInjector handles fault injection during testing
type FaultInjector struct {
	activeFaults     map[FaultType]*ActiveFault
	faultHistory     []*FaultEvent
	mu               sync.RWMutex
	logger           *zap.Logger
}

// ActiveFault represents a currently active fault
type ActiveFault struct {
	Type        FaultType
	StartTime   time.Time
	Duration    time.Duration
	Intensity   int
	Config      interface{}
}

// FaultEvent records a fault injection event
type FaultEvent struct {
	Type        FaultType
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Impact      *FaultImpact
}

// FaultImpact measures the impact of a fault
type FaultImpact struct {
	RequestsAffected    int64
	ErrorRateIncrease   float64
	LatencyIncrease     time.Duration
	RecoveryTime        time.Duration
	DataLoss            bool
	ServiceDegradation  bool
}

// ResilienceMetrics tracks system resilience during chaos testing
type ResilienceMetrics struct {
	TotalRequests        int64
	SuccessfulRequests   int64
	FailedRequests       int64
	RecoveredRequests    int64
	
	AverageLatency       time.Duration
	P95Latency          time.Duration
	P99Latency          time.Duration
	
	MTTR                time.Duration // Mean Time To Recovery
	MTBF                time.Duration // Mean Time Between Failures
	
	SystemAvailability   float64
	DataIntegrity        float64
	GracefulDegradation  bool
	
	ChaosEvents         []*ChaosEvent
	mu                  sync.RWMutex
}

// ChaosEvent records a significant event during chaos testing
type ChaosEvent struct {
	Timestamp   time.Time
	Type        string
	Severity    string
	Message     string
	Metrics     map[string]interface{}
}

// SetupSuite initializes the chaos testing environment
func (s *SnapshotChaosTestSuite) SetupSuite() {
	s.logger = zaptest.NewLogger(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	
	// Load chaos testing configuration
	s.testConfig = s.loadChaosConfig()
	
	// Initialize services
	s.setupServices()
	
	// Initialize chaos components
	s.setupChaosComponents()
	
	s.T().Log("Chaos engineering test suite initialized")
}

// TearDownSuite cleans up the chaos testing environment
func (s *SnapshotChaosTestSuite) TearDownSuite() {
	s.cancel()
	s.chaosController.StopAllScenarios()
	s.generateResilienceReport()
	s.T().Log("Chaos engineering test suite completed")
}

// TestSnapshotResilienceUnderLatencyFaults tests system behavior under latency faults
func (s *SnapshotChaosTestSuite) TestSnapshotResilienceUnderLatencyFaults() {
	scenario := &ChaosScenario{
		Name:           "Latency Injection",
		Description:    "Inject random latency into snapshot operations",
		Duration:       2 * time.Minute,
		FaultTypes:     []FaultType{FaultTypeLatency},
		IntensityLevel: 5,
		RecoveryTime:   30 * time.Second,
		SuccessCriteria: func(metrics *ResilienceMetrics) bool {
			return metrics.SystemAvailability > 0.95 && // 95% availability
				metrics.P95Latency < 500*time.Millisecond // Acceptable degradation
		},
	}
	
	s.runChaosScenario(scenario, func() {
		s.executeLatencyFaultTest()
	})
}

// TestSnapshotResilienceUnderMemoryPressure tests behavior under memory pressure
func (s *SnapshotChaosTestSuite) TestSnapshotResilienceUnderMemoryPressure() {
	scenario := &ChaosScenario{
		Name:           "Memory Pressure",
		Description:    "Simulate memory exhaustion scenarios",
		Duration:       3 * time.Minute,
		FaultTypes:     []FaultType{FaultTypeMemoryLeak, FaultTypeResourceExhaustion},
		IntensityLevel: 7,
		RecoveryTime:   45 * time.Second,
		SuccessCriteria: func(metrics *ResilienceMetrics) bool {
			return metrics.SystemAvailability > 0.90 && // Allows some degradation
				!metrics.DataIntegrity && // No data corruption
				metrics.MTTR < 60*time.Second // Quick recovery
		},
	}
	
	s.runChaosScenario(scenario, func() {
		s.executeMemoryPressureTest()
	})
}

// TestSnapshotCacheResilienceUnderCorruption tests cache corruption handling
func (s *SnapshotChaosTestSuite) TestSnapshotCacheResilienceUnderCorruption() {
	scenario := &ChaosScenario{
		Name:           "Cache Corruption",
		Description:    "Inject cache corruption and test recovery",
		Duration:       2 * time.Minute,
		FaultTypes:     []FaultType{FaultTypeCacheCorruption, FaultTypeDataCorruption},
		IntensityLevel: 8,
		RecoveryTime:   20 * time.Second,
		SuccessCriteria: func(metrics *ResilienceMetrics) bool {
			return metrics.DataIntegrity == 1.0 && // Perfect data integrity
				metrics.RecoveredRequests > 0 // System should recover
		},
	}
	
	s.runChaosScenario(scenario, func() {
		s.executeCacheCorruptionTest()
	})
}

// TestCascadingFailureResilience tests resilience against cascading failures
func (s *SnapshotChaosTestSuite) TestCascadingFailureResilience() {
	scenario := &ChaosScenario{
		Name:           "Cascading Failures",
		Description:    "Simulate cascading service failures",
		Duration:       4 * time.Minute,
		FaultTypes:     []FaultType{FaultTypeCascadingFailure, FaultTypeServiceUnavailable},
		IntensityLevel: 9,
		RecoveryTime:   90 * time.Second,
		SuccessCriteria: func(metrics *ResilienceMetrics) bool {
			return metrics.GracefulDegradation && // Graceful degradation
				metrics.SystemAvailability > 0.80 && // Maintains core functionality
				metrics.MTTR < 2*time.Minute // Reasonable recovery time
		},
	}
	
	s.runChaosScenario(scenario, func() {
		s.executeCascadingFailureTest()
	})
}

// TestNetworkPartitionResilience tests behavior under network partitions
func (s *SnapshotChaosTestSuite) TestNetworkPartitionResilience() {
	scenario := &ChaosScenario{
		Name:           "Network Partition",
		Description:    "Simulate network partitions and connectivity issues",
		Duration:       3 * time.Minute,
		FaultTypes:     []FaultType{FaultTypeNetworkPartition},
		IntensityLevel: 6,
		RecoveryTime:   30 * time.Second,
		SuccessCriteria: func(metrics *ResilienceMetrics) bool {
			return metrics.SystemAvailability > 0.85 && // Tolerates network issues
				metrics.RecoveredRequests > 0 // Can recover from partition
		},
	}
	
	s.runChaosScenario(scenario, func() {
		s.executeNetworkPartitionTest()
	})
}

// TestConcurrentChaosScenarios tests system resilience under multiple simultaneous faults
func (s *SnapshotChaosTestSuite) TestConcurrentChaosScenarios() {
	scenario := &ChaosScenario{
		Name:           "Multi-Fault Chaos",
		Description:    "Multiple concurrent fault types",
		Duration:       5 * time.Minute,
		FaultTypes:     []FaultType{FaultTypeLatency, FaultTypeMemoryLeak, FaultTypeCacheCorruption},
		IntensityLevel: 10, // Maximum chaos
		RecoveryTime:   2 * time.Minute,
		SuccessCriteria: func(metrics *ResilienceMetrics) bool {
			return metrics.SystemAvailability > 0.70 && // Survival under extreme conditions
				metrics.DataIntegrity > 0.99 && // Critical: maintain data integrity
				metrics.MTTR < 3*time.Minute // Recovery within reasonable time
		},
	}
	
	s.runChaosScenario(scenario, func() {
		s.executeMultiFaultChaosTest()
	})
}

// runChaosScenario executes a chaos engineering scenario
func (s *SnapshotChaosTestSuite) runChaosScenario(
	scenario *ChaosScenario,
	testFunc func(),
) {
	s.T().Logf("Starting chaos scenario: %s", scenario.Name)
	
	// Initialize metrics tracking
	s.resilenceMetrics = NewResilienceMetrics()
	
	// Start the chaos scenario
	err := s.chaosController.StartScenario(scenario)
	require.NoError(s.T(), err, "Failed to start chaos scenario")
	
	// Start metrics collection
	metricsCtx, metricsCancel := context.WithCancel(s.ctx)
	go s.collectResilienceMetrics(metricsCtx)
	
	// Execute the test workload
	workloadCtx, workloadCancel := context.WithTimeout(s.ctx, scenario.Duration)
	go s.executeTestWorkload(workloadCtx)
	
	// Run scenario-specific test
	testFunc()
	
	// Wait for scenario completion
	<-workloadCtx.Done()
	workloadCancel()
	
	// Allow recovery time
	time.Sleep(scenario.RecoveryTime)
	
	// Stop chaos injection
	err = s.chaosController.StopScenario(scenario.Name)
	require.NoError(s.T(), err, "Failed to stop chaos scenario")
	
	// Stop metrics collection
	metricsCancel()
	
	// Validate success criteria
	success := scenario.SuccessCriteria(s.resilenceMetrics)
	assert.True(s.T(), success, 
		fmt.Sprintf("Scenario %s failed success criteria", scenario.Name))
	
	// Log scenario results
	s.logScenarioResults(scenario, s.resilenceMetrics)
}

// executeLatencyFaultTest implements latency fault testing
func (s *SnapshotChaosTestSuite) executeLatencyFaultTest() {
	s.T().Log("Executing latency fault test...")
	
	// Configure latency injection
	latencyConfig := &LatencyFaultConfig{
		MinLatency:    50 * time.Millisecond,
		MaxLatency:    2 * time.Second,
		Probability:   0.3, // 30% of requests affected
		Distribution:  "exponential",
	}
	
	// Inject latency faults
	s.faultInjector.InjectLatencyFault(latencyConfig)
	
	// Monitor system behavior under latency stress
	s.monitorLatencyImpact()
}

// executeMemoryPressureTest implements memory pressure testing
func (s *SnapshotChaosTestSuite) executeMemoryPressureTest() {
	s.T().Log("Executing memory pressure test...")
	
	// Configure memory pressure
	memoryConfig := &MemoryPressureConfig{
		LeakRate:       1024 * 1024, // 1MB per second
		MaxMemoryUsage: 1.5 * 1024 * 1024 * 1024, // 1.5GB limit
		GCDisabled:     false,
		PressureType:   "gradual",
	}
	
	// Inject memory pressure
	s.faultInjector.InjectMemoryPressure(memoryConfig)
	
	// Monitor memory behavior and recovery
	s.monitorMemoryBehavior()
}

// executeCacheCorruptionTest implements cache corruption testing
func (s *SnapshotChaosTestSuite) executeCacheCorruptionTest() {
	s.T().Log("Executing cache corruption test...")
	
	// Configure cache corruption
	corruptionConfig := &CacheCorruptionConfig{
		CorruptionRate:  0.1, // 10% of cache entries
		CorruptionTypes: []string{"data_scramble", "partial_corruption", "complete_loss"},
		RecoveryEnabled: true,
	}
	
	// Inject cache corruption
	s.faultInjector.InjectCacheCorruption(corruptionConfig)
	
	// Monitor cache integrity and recovery
	s.monitorCacheIntegrity()
}

// executeCascadingFailureTest implements cascading failure testing
func (s *SnapshotChaosTestSuite) executeCascadingFailureTest() {
	s.T().Log("Executing cascading failure test...")
	
	// Configure cascading failure
	cascadeConfig := &CascadingFailureConfig{
		InitialFailure:    "cae_engine_failure",
		PropagationDelay:  5 * time.Second,
		FailureChain:     []string{"cache_service", "event_publisher", "token_generator"},
		RecoveryStrategy: "gradual",
	}
	
	// Inject cascading failure
	s.faultInjector.InjectCascadingFailure(cascadeConfig)
	
	// Monitor system degradation and recovery
	s.monitorSystemDegradation()
}

// executeNetworkPartitionTest implements network partition testing
func (s *SnapshotChaosTestSuite) executeNetworkPartitionTest() {
	s.T().Log("Executing network partition test...")
	
	// Configure network partition
	partitionConfig := &NetworkPartitionConfig{
		PartitionType:     "split_brain",
		Duration:          30 * time.Second,
		AffectedServices:  []string{"kafka", "redis", "grpc"},
		RecoveryDelay:     10 * time.Second,
	}
	
	// Inject network partition
	s.faultInjector.InjectNetworkPartition(partitionConfig)
	
	// Monitor network resilience
	s.monitorNetworkResilience()
}

// executeMultiFaultChaosTest implements multi-fault chaos testing
func (s *SnapshotChaosTestSuite) executeMultiFaultChaosTest() {
	s.T().Log("Executing multi-fault chaos test...")
	
	// Inject multiple faults concurrently
	go s.faultInjector.InjectLatencyFault(&LatencyFaultConfig{
		MinLatency:   100 * time.Millisecond,
		MaxLatency:   1 * time.Second,
		Probability:  0.2,
		Distribution: "uniform",
	})
	
	go s.faultInjector.InjectMemoryPressure(&MemoryPressureConfig{
		LeakRate:       512 * 1024, // 512KB per second
		MaxMemoryUsage: 1 * 1024 * 1024 * 1024, // 1GB limit
		PressureType:   "spike",
	})
	
	go s.faultInjector.InjectCacheCorruption(&CacheCorruptionConfig{
		CorruptionRate:  0.05, // 5% corruption rate
		CorruptionTypes: []string{"data_scramble"},
		RecoveryEnabled: true,
	})
	
	// Monitor system behavior under extreme conditions
	s.monitorExtremeChaosConditions()
}

// executeTestWorkload runs a continuous test workload during chaos scenarios
func (s *SnapshotChaosTestSuite) executeTestWorkload(ctx context.Context) {
	requestCounter := int64(0)
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go s.executeWorkloadRequest(atomic.AddInt64(&requestCounter, 1))
		}
	}
}

// executeWorkloadRequest executes a single workload request
func (s *SnapshotChaosTestSuite) executeWorkloadRequest(requestID int64) {
	start := time.Now()
	
	// Create test request
	request := s.createChaosTestRequest(requestID)
	
	// Process snapshot
	snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
	
	latency := time.Since(start)
	
	// Record metrics
	s.resilenceMetrics.mu.Lock()
	s.resilenceMetrics.TotalRequests++
	if err != nil || snapshot == nil {
		s.resilenceMetrics.FailedRequests++
	} else {
		s.resilenceMetrics.SuccessfulRequests++
	}
	s.updateLatencyMetrics(latency)
	s.resilenceMetrics.mu.Unlock()
}

// Helper methods for chaos testing

func (s *SnapshotChaosTestSuite) loadChaosConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:         8030,
			ReadTimeout:  60 * time.Second, // Longer timeouts for chaos testing
			WriteTimeout: 60 * time.Second,
		},
		Cache: config.CacheConfig{
			TTL:              30 * time.Minute,
			MaxSize:          5000,
			EvictionPolicy:   "lru",
			CompressionLevel: 3,
		},
		Learning: config.LearningConfig{
			Enabled: true,
			EventPublisher: config.EventPublisherConfig{
				EnableEventPublishing: true,
				BatchSize:            100,
				FlushIntervalSeconds: 5,
				RetryAttempts:       5, // More retries for chaos conditions
			},
		},
	}
}

func (s *SnapshotChaosTestSuite) setupChaosComponents() {
	s.chaosController = NewChaosController(s.logger)
	s.faultInjector = NewFaultInjector(s.logger)
	s.resilenceMetrics = NewResilienceMetrics()
}

func (s *SnapshotChaosTestSuite) generateResilienceReport() {
	s.T().Logf("\n=== Chaos Engineering Resilience Report ===")
	s.T().Logf("Total Requests: %d", s.resilenceMetrics.TotalRequests)
	s.T().Logf("Successful Requests: %d", s.resilenceMetrics.SuccessfulRequests)
	s.T().Logf("Failed Requests: %d", s.resilenceMetrics.FailedRequests)
	s.T().Logf("System Availability: %.4f%%", s.resilenceMetrics.SystemAvailability*100)
	s.T().Logf("Data Integrity: %.4f%%", s.resilenceMetrics.DataIntegrity*100)
	s.T().Logf("MTTR: %v", s.resilenceMetrics.MTTR)
	s.T().Logf("MTBF: %v", s.resilenceMetrics.MTBF)
	s.T().Logf("=== End Resilience Report ===\n")
}

// Configuration structs for different fault types

type LatencyFaultConfig struct {
	MinLatency    time.Duration
	MaxLatency    time.Duration
	Probability   float64
	Distribution  string
}

type MemoryPressureConfig struct {
	LeakRate       int64
	MaxMemoryUsage int64
	GCDisabled     bool
	PressureType   string
}

type CacheCorruptionConfig struct {
	CorruptionRate  float64
	CorruptionTypes []string
	RecoveryEnabled bool
}

type CascadingFailureConfig struct {
	InitialFailure    string
	PropagationDelay  time.Duration
	FailureChain      []string
	RecoveryStrategy  string
}

type NetworkPartitionConfig struct {
	PartitionType     string
	Duration          time.Duration
	AffectedServices  []string
	RecoveryDelay     time.Duration
}

// TestSnapshotChaosTestSuite runs the chaos engineering test suite
func TestSnapshotChaosTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos engineering tests in short mode")
	}
	
	suite.Run(t, new(SnapshotChaosTestSuite))
}