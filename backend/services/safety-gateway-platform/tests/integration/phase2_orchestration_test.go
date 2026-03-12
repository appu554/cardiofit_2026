package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/orchestration"
	"safety-gateway-platform/internal/registry"
	"safety-gateway-platform/internal/server"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// Phase2OrchestrationTestSuite provides comprehensive integration testing
// for Phase 2 Advanced Orchestration features
type Phase2OrchestrationTestSuite struct {
	suite.Suite
	
	// Core server components
	server          *server.Server
	httpServer      *httptest.Server
	config          *config.Config
	logger          *logger.Logger
	
	// Orchestration components
	orchestrator    types.SafetyOrchestrator
	batchProcessor  *orchestration.EnhancedBatchProcessor
	metrics         *orchestration.ComprehensiveMetricsCollector
	registry        *registry.EngineRegistry
	
	// Test context
	ctx     context.Context
	cancel  context.CancelFunc
	
	// Test fixtures
	testRequests    []*types.SafetyRequest
	testBatches     []*types.BatchRequest
}

// SetupSuite initializes the Phase 2 test environment
func (s *Phase2OrchestrationTestSuite) SetupSuite() {
	s.logger = logger.New(config.LoggingConfig{
		Level:  "debug",
		Format: "json",
	})
	s.ctx, s.cancel = context.WithCancel(context.Background())
	
	// Load Phase 2 test configuration
	s.config = s.loadPhase2Config()
	
	// Initialize server with Phase 2 features
	s.setupServerWithPhase2()
	
	// Start test HTTP server
	s.setupTestHTTPServer()
	
	// Prepare test fixtures
	s.preparePhase2TestFixtures()
}

// TearDownSuite cleans up the test environment
func (s *Phase2OrchestrationTestSuite) TearDownSuite() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
	if s.server != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		s.server.Shutdown(shutdownCtx)
	}
	s.cancel()
}

// TestPhase2ServerBootstrap tests that Phase 2 is properly integrated into server
func (s *Phase2OrchestrationTestSuite) TestPhase2ServerBootstrap() {
	// Test Phase 2 orchestration is enabled
	assert.True(s.T(), s.config.AdvancedOrchestration.Enabled, 
		"Phase 2 advanced orchestration should be enabled")
	
	// Test orchestrator interface is working
	assert.NotNil(s.T(), s.orchestrator, "Orchestrator should be initialized")
	
	// Test advanced orchestrator is being used
	advancedOrch, isAdvanced := s.orchestrator.(*orchestration.AdvancedOrchestrationEngine)
	assert.True(s.T(), isAdvanced, "Should be using AdvancedOrchestrationEngine")
	assert.NotNil(s.T(), advancedOrch, "Advanced orchestrator should not be nil")
	
	// Test batch processor is initialized
	assert.NotNil(s.T(), s.batchProcessor, "Batch processor should be initialized")
	
	// Test metrics collector is initialized
	assert.NotNil(s.T(), s.metrics, "Metrics collector should be initialized")
}

// TestAdvancedOrchestrationProcessing tests advanced orchestration features
func (s *Phase2OrchestrationTestSuite) TestAdvancedOrchestrationProcessing() {
	testCases := []struct {
		name                string
		request             *types.SafetyRequest
		expectedStrategy    string
		expectedLatency     time.Duration
		validateResponse    func(*types.SafetyResponse) error
	}{
		{
			name:             "Critical Priority Routing",
			request:          s.createCriticalPriorityRequest(),
			expectedStrategy: "veto_critical",
			expectedLatency:  100 * time.Millisecond,
			validateResponse: s.validateCriticalResponse,
		},
		{
			name:             "Medication Interaction Routing",
			request:          s.createMedicationInteractionRequest(),
			expectedStrategy: "veto_critical",
			expectedLatency:  150 * time.Millisecond,
			validateResponse: s.validateInteractionResponse,
		},
		{
			name:             "Routine Advisory Routing",
			request:          s.createRoutineAdvisoryRequest(),
			expectedStrategy: "advisory",
			expectedLatency:  75 * time.Millisecond,
			validateResponse: s.validateAdvisoryResponse,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			startTime := time.Now()
			
			// Process request through advanced orchestration
			response, err := s.orchestrator.ProcessSafetyRequest(s.ctx, tc.request)
			
			processingTime := time.Since(startTime)
			
			// Basic validation
			require.NoError(s.T(), err, "Processing should not fail")
			require.NotNil(s.T(), response, "Response should not be nil")
			
			// Performance validation
			assert.LessOrEqual(s.T(), processingTime, tc.expectedLatency,
				fmt.Sprintf("Processing time %v exceeded expected %v", 
					processingTime, tc.expectedLatency))
			
			// Response-specific validation
			err = tc.validateResponse(response)
			assert.NoError(s.T(), err, "Response validation failed")
			
			// Verify routing metadata
			s.validateRoutingMetadata(response, tc.expectedStrategy)
		})
	}
}

// TestBatchProcessingIntegration tests enhanced batch processing
func (s *Phase2OrchestrationTestSuite) TestBatchProcessingIntegration() {
	batchTestCases := []struct {
		name                string
		strategy            string
		requests            []*types.SafetyRequest
		expectedLatency     time.Duration
		expectedThroughput  float64
		validateResults     func([]*types.SafetyResponse) error
	}{
		{
			name:               "Patient Grouped Strategy",
			strategy:           "patient_grouped",
			requests:           s.createPatientGroupedRequests(10),
			expectedLatency:    300 * time.Millisecond,
			expectedThroughput: 30.0, // requests per second
			validateResults:    s.validatePatientGroupedResults,
		},
		{
			name:               "Snapshot Optimized Strategy",
			strategy:           "snapshot_optimized",
			requests:           s.createSnapshotOptimizedRequests(15),
			expectedLatency:    400 * time.Millisecond,
			expectedThroughput: 35.0,
			validateResults:    s.validateSnapshotOptimizedResults,
		},
		{
			name:               "Parallel Direct Strategy",
			strategy:           "parallel_direct",
			requests:           s.createParallelDirectRequests(20),
			expectedLatency:    200 * time.Millisecond,
			expectedThroughput: 80.0,
			validateResults:    s.validateParallelDirectResults,
		},
	}

	for _, tc := range batchTestCases {
		s.Run(tc.name, func() {
			// Create batch request
			batchRequest := &types.BatchRequest{
				BatchID:   fmt.Sprintf("batch-%s-%d", tc.strategy, time.Now().Unix()),
				Requests:  tc.requests,
				Strategy:  tc.strategy,
				Priority:  "normal",
				Metadata: map[string]interface{}{
					"test_case": tc.name,
				},
			}
			
			startTime := time.Now()
			
			// Process batch through enhanced batch processor
			batchResult, err := s.batchProcessor.ProcessBatch(s.ctx, batchRequest)
			
			processingTime := time.Since(startTime)
			
			// Basic validation
			require.NoError(s.T(), err, "Batch processing should not fail")
			require.NotNil(s.T(), batchResult, "Batch result should not be nil")
			
			// Performance validation
			assert.LessOrEqual(s.T(), processingTime, tc.expectedLatency,
				fmt.Sprintf("Batch processing time %v exceeded expected %v", 
					processingTime, tc.expectedLatency))
			
			// Throughput validation
			actualThroughput := float64(len(tc.requests)) / processingTime.Seconds()
			assert.GreaterOrEqual(s.T(), actualThroughput, tc.expectedThroughput,
				fmt.Sprintf("Throughput %.2f below expected %.2f", 
					actualThroughput, tc.expectedThroughput))
			
			// Results validation
			assert.Equal(s.T(), len(tc.requests), len(batchResult.Results),
				"All requests should have results")
			
			err = tc.validateResults(batchResult.Results)
			assert.NoError(s.T(), err, "Batch results validation failed")
			
			// Validate batch metadata
			s.validateBatchMetadata(batchResult, tc.strategy)
		})
	}
}

// TestLoadBalancingStrategies tests different load balancing strategies
func (s *Phase2OrchestrationTestSuite) TestLoadBalancingStrategies() {
	strategies := []string{"adaptive", "round_robin", "least_loaded", "performance_weighted"}
	
	for _, strategy := range strategies {
		s.Run(fmt.Sprintf("LoadBalancing_%s", strategy), func() {
			// Update configuration for this strategy
			s.updateLoadBalancingStrategy(strategy)
			
			// Create multiple concurrent requests
			numRequests := 20
			requests := s.createVariedRequests(numRequests)
			
			// Process requests concurrently
			var wg sync.WaitGroup
			results := make(chan *requestResult, numRequests)
			
			startTime := time.Now()
			for i, request := range requests {
				wg.Add(1)
				go func(req *types.SafetyRequest, index int) {
					defer wg.Done()
					
					reqStart := time.Now()
					response, err := s.orchestrator.ProcessSafetyRequest(s.ctx, req)
					reqDuration := time.Since(reqStart)
					
					results <- &requestResult{
						Index:     index,
						Request:   req,
						Response:  response,
						Error:     err,
						Duration:  reqDuration,
					}
				}(request, i)
			}
			
			wg.Wait()
			close(results)
			
			totalDuration := time.Since(startTime)
			
			// Collect and analyze results
			var allResults []*requestResult
			var errors []error
			var latencies []time.Duration
			
			for result := range results {
				allResults = append(allResults, result)
				if result.Error != nil {
					errors = append(errors, result.Error)
				} else {
					latencies = append(latencies, result.Duration)
				}
			}
			
			// Validate no errors
			assert.Empty(s.T(), errors, "No requests should fail")
			assert.Len(s.T(), allResults, numRequests, "All requests should complete")
			
			// Validate performance characteristics based on strategy
			s.validateLoadBalancingPerformance(strategy, allResults, totalDuration)
		})
	}
}

// TestPhase2APIEndpoints tests the new HTTP API endpoints
func (s *Phase2OrchestrationTestSuite) TestPhase2APIEndpoints() {
	endpointTests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		validateBody   func([]byte) error
	}{
		{
			name:           "Batch Validation Endpoint",
			method:         "POST",
			path:           "/api/v1/batch/validate",
			body:           s.createTestBatchRequest(),
			expectedStatus: http.StatusOK,
			validateBody:   s.validateBatchValidationResponse,
		},
		{
			name:           "Orchestration Stats Endpoint",
			method:         "GET",
			path:           "/api/v1/orchestration/stats",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody:   s.validateOrchestrationStatsResponse,
		},
		{
			name:           "Orchestration Metrics Endpoint",
			method:         "GET",
			path:           "/api/v1/orchestration/metrics",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody:   s.validateOrchestrationMetricsResponse,
		},
		{
			name:           "Orchestration Health Check",
			method:         "GET",
			path:           "/api/v1/health/orchestration",
			body:           nil,
			expectedStatus: http.StatusOK,
			validateBody:   s.validateOrchestrationHealthResponse,
		},
	}

	for _, test := range endpointTests {
		s.Run(test.name, func() {
			var req *http.Request
			var err error
			
			if test.body != nil {
				bodyBytes, _ := json.Marshal(test.body)
				req, err = http.NewRequest(test.method, s.httpServer.URL+test.path, 
					bytes.NewBuffer(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(test.method, s.httpServer.URL+test.path, nil)
			}
			
			require.NoError(s.T(), err, "Request creation should not fail")
			
			// Execute request
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			require.NoError(s.T(), err, "Request should not fail")
			defer resp.Body.Close()
			
			// Validate status code
			assert.Equal(s.T(), test.expectedStatus, resp.StatusCode,
				fmt.Sprintf("Expected status %d, got %d", test.expectedStatus, resp.StatusCode))
			
			// Read and validate response body
			bodyBytes := make([]byte, 0)
			buffer := make([]byte, 1024)
			for {
				n, err := resp.Body.Read(buffer)
				if n > 0 {
					bodyBytes = append(bodyBytes, buffer[:n]...)
				}
				if err != nil {
					break
				}
			}
			
			err = test.validateBody(bodyBytes)
			assert.NoError(s.T(), err, "Response body validation failed")
		})
	}
}

// TestMetricsCollection tests comprehensive metrics collection
func (s *Phase2OrchestrationTestSuite) TestMetricsCollection() {
	// Generate load to collect metrics
	s.generateMetricsLoad()
	
	// Wait for metrics collection
	time.Sleep(2 * time.Second)
	
	// Retrieve metrics
	performanceMetrics := s.metrics.GetPerformanceMetrics()
	loadMetrics := s.metrics.GetLoadMetrics()
	routingMetrics := s.metrics.GetRoutingMetrics()
	batchMetrics := s.metrics.GetBatchMetrics()
	
	// Validate performance metrics
	assert.NotNil(s.T(), performanceMetrics, "Performance metrics should be available")
	assert.Greater(s.T(), performanceMetrics.TotalRequests, int64(0), 
		"Should have processed requests")
	assert.Greater(s.T(), performanceMetrics.AverageLatency, time.Duration(0), 
		"Should have latency data")
	
	// Validate load metrics
	assert.NotNil(s.T(), loadMetrics, "Load metrics should be available")
	assert.GreaterOrEqual(s.T(), loadMetrics.CurrentLoad, 0.0, 
		"Load should be non-negative")
	
	// Validate routing metrics
	assert.NotNil(s.T(), routingMetrics, "Routing metrics should be available")
	assert.NotEmpty(s.T(), routingMetrics.EngineStats, 
		"Should have engine statistics")
	
	// Validate batch metrics
	assert.NotNil(s.T(), batchMetrics, "Batch metrics should be available")
	
	// Test metrics export
	s.testMetricsExport()
}

// TestConfigurationEnvironments tests environment-specific configurations
func (s *Phase2OrchestrationTestSuite) TestConfigurationEnvironments() {
	environments := []struct {
		name             string
		env              string
		expectedStrategy string
		expectedBatchSize int
	}{
		{
			name:             "Development Environment",
			env:              "development",
			expectedStrategy: "round_robin",
			expectedBatchSize: 10,
		},
		{
			name:             "Staging Environment",
			env:              "staging",
			expectedStrategy: "least_loaded",
			expectedBatchSize: 50,
		},
		{
			name:             "Production Environment",
			env:              "production",
			expectedStrategy: "adaptive",
			expectedBatchSize: 50,
		},
	}

	for _, env := range environments {
		s.Run(env.name, func() {
			// Load environment-specific config
			config := s.loadEnvironmentConfig(env.env)
			
			// Validate load balancing strategy
			assert.Equal(s.T(), env.expectedStrategy, 
				config.AdvancedOrchestration.LoadBalancing.Strategy,
				"Load balancing strategy should match environment")
			
			// Validate batch size
			assert.Equal(s.T(), env.expectedBatchSize,
				config.AdvancedOrchestration.BatchProcessing.MaxBatchSize,
				"Batch size should match environment")
		})
	}
}

// Helper methods for test setup and validation

func (s *Phase2OrchestrationTestSuite) loadPhase2Config() *config.Config {
	return &config.Config{
		Service: config.ServiceConfig{
			Name:        "safety-gateway-platform",
			Port:        8030,
			HTTPPort:    8031,
			Environment: "testing",
		},
		Performance: config.PerformanceConfig{
			MaxConcurrentRequests: 100,
			RequestTimeout:        "10s",
			MaxRequestSizeMB:      10,
		},
		AdvancedOrchestration: &config.AdvancedOrchestrationConfig{
			Enabled:               true,
			MaxConcurrentRequests: 100,
			RequestTimeout:        "5s",
			BatchProcessing: config.BatchProcessingConfig{
				Enabled:                true,
				MaxBatchSize:          20,
				BatchTimeout:          "100ms",
				Concurrency:           5,
				PatientGrouping:       true,
				SnapshotOptimized:     true,
				MaxConcurrentBatches:  10,
				WorkerPoolSize:        8,
				Strategies:            []string{"patient_grouped", "snapshot_optimized", "parallel_direct"},
			},
			LoadBalancing: config.LoadBalancingConfig{
				Strategy:           "adaptive",
				EnableHealthCheck:  true,
				HealthCheckInterval: "30s",
				EngineSelectionCriteria: config.EngineSelectionCriteria{
					MaxErrorRate:          0.05,
					MaxAverageLatencyMs:   1000,
					MinThroughputPerSec:   1.0,
					LoadScoreThreshold:    0.8,
				},
			},
			Routing: config.RoutingConfig{
				EnableIntelligentRouting: true,
				DefaultTier:              "veto_critical",
				DynamicRuleEvaluation:    true,
				MaxRoutingTime:           "50ms",
			},
			Metrics: config.MetricsConfig{
				EnableMetrics:             true,
				MetricsInterval:           "5s",
				EnablePerformanceMetrics:  true,
				EnableLoadMetrics:         true,
				EnableRoutingMetrics:      true,
				EnableBatchMetrics:        true,
				ExportJSON:               true,
				JSONExportPath:           "/tmp/test_metrics.json",
			},
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}
}

func (s *Phase2OrchestrationTestSuite) setupServerWithPhase2() {
	var err error
	s.server, err = server.New(s.config, s.logger)
	require.NoError(s.T(), err, "Server creation should not fail")
	
	// Extract orchestration components for testing
	// This would require adding getter methods to the server
	// or using reflection for testing purposes
}

func (s *Phase2OrchestrationTestSuite) setupTestHTTPServer() {
	// Create test HTTP handler from the server's HTTP handler
	s.httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Route to appropriate handler based on path
		s.routeTestRequest(w, r)
	}))
}

func (s *Phase2OrchestrationTestSuite) routeTestRequest(w http.ResponseWriter, r *http.Request) {
	// Simple router for test endpoints
	switch {
	case strings.HasPrefix(r.URL.Path, "/api/v1/batch/validate"):
		s.handleBatchValidate(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/v1/orchestration/stats"):
		s.handleOrchestrationStats(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/v1/orchestration/metrics"):
		s.handleOrchestrationMetrics(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/v1/health/orchestration"):
		s.handleOrchestrationHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Test fixture creation methods

func (s *Phase2OrchestrationTestSuite) createCriticalPriorityRequest() *types.SafetyRequest {
	return &types.SafetyRequest{
		RequestID:     fmt.Sprintf("critical-%d", time.Now().UnixNano()),
		PatientID:     "test-patient-critical",
		ActionType:    "medication_prescribe",
		Priority:      "critical",
		MedicationIDs: []string{"warfarin", "aspirin"},
		ConditionIDs:  []string{"atrial_fibrillation"},
		Timestamp:     time.Now(),
	}
}

func (s *Phase2OrchestrationTestSuite) createMedicationInteractionRequest() *types.SafetyRequest {
	return &types.SafetyRequest{
		RequestID:     fmt.Sprintf("interaction-%d", time.Now().UnixNano()),
		PatientID:     "test-patient-interaction",
		ActionType:    "medication_interaction",
		Priority:      "high",
		MedicationIDs: []string{"warfarin", "amiodarone", "clarithromycin"},
		ConditionIDs:  []string{"heart_failure", "infection"},
		Timestamp:     time.Now(),
		Metadata: map[string]interface{}{
			"medication_count": 3,
		},
	}
}

func (s *Phase2OrchestrationTestSuite) createRoutineAdvisoryRequest() *types.SafetyRequest {
	return &types.SafetyRequest{
		RequestID:     fmt.Sprintf("routine-%d", time.Now().UnixNano()),
		PatientID:     "test-patient-routine",
		ActionType:    "routine_check",
		Priority:      "low",
		MedicationIDs: []string{"acetaminophen"},
		ConditionIDs:  []string{"headache"},
		Timestamp:     time.Now(),
	}
}

// Validation methods

func (s *Phase2OrchestrationTestSuite) validateCriticalResponse(response *types.SafetyResponse) error {
	if response.Status == types.SafetyStatusError {
		return fmt.Errorf("critical request should not result in error status")
	}
	if len(response.EngineResults) == 0 {
		return fmt.Errorf("critical request should have engine results")
	}
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateInteractionResponse(response *types.SafetyResponse) error {
	if response.RiskScore < 0.5 {
		return fmt.Errorf("interaction request should have elevated risk score, got %f", 
			response.RiskScore)
	}
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateAdvisoryResponse(response *types.SafetyResponse) error {
	if response.ProcessingTime > 100*time.Millisecond {
		return fmt.Errorf("advisory request should be processed quickly, took %v", 
			response.ProcessingTime)
	}
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateRoutingMetadata(response *types.SafetyResponse, expectedStrategy string) {
	// Check if routing metadata is present
	if metadata, ok := response.Metadata["routing_strategy"].(string); ok {
		assert.Equal(s.T(), expectedStrategy, metadata, 
			"Routing strategy should match expected")
	}
}

// Additional helper types and methods

type requestResult struct {
	Index    int
	Request  *types.SafetyRequest
	Response *types.SafetyResponse
	Error    error
	Duration time.Duration
}

// TestPhase2OrchestrationTestSuite runs the Phase 2 integration test suite
func TestPhase2OrchestrationTestSuite(t *testing.T) {
	suite.Run(t, new(Phase2OrchestrationTestSuite))
}