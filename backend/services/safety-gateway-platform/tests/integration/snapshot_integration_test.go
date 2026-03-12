package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"safety-gateway-platform/internal/reproducibility"
	"safety-gateway-platform/internal/types"
)

// SnapshotIntegrationTestSuite provides comprehensive end-to-end testing
// for the entire snapshot transformation workflow
type SnapshotIntegrationTestSuite struct {
	suite.Suite
	
	// Core services
	contextBuilder    *context.ContextBuilder
	assemblyService   *context.AssemblyService
	cacheManager      *context.CacheManager
	snapshotCache     *cache.SnapshotCache
	
	// Enhanced Phase 4 services
	tokenGenerator    *override.EnhancedTokenGenerator
	overrideService   *override.SnapshotAwareOverrideService
	eventPublisher    *learning.LearningEventPublisher
	overrideAnalyzer  *learning.OverrideAnalyzer
	replayService     *reproducibility.DecisionReplayService
	
	// Mock services
	mockCAEEngine     *engines.MockCAEEngine
	mockKafka        *MockKafkaProducer
	mockEventStore   *MockOverrideEventStore
	
	// Test fixtures
	testConfig       *config.Config
	testSnapshots    []*types.ClinicalSnapshot
	testRequests     []*types.SafetyRequest
	testResponses    []*types.SafetyResponse
	
	logger           *zap.Logger
	ctx              context.Context
	cancel           context.CancelFunc
}

// SetupSuite initializes the test environment
func (s *SnapshotIntegrationTestSuite) SetupSuite() {
	s.logger = zaptest.NewLogger(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())
	
	// Load test configuration
	s.testConfig = s.loadTestConfig()
	
	// Initialize mock services
	s.setupMockServices()
	
	// Initialize core services
	s.setupCoreServices()
	
	// Initialize Phase 4 enhanced services
	s.setupEnhancedServices()
	
	// Prepare test fixtures
	s.prepareTestFixtures()
}

// TearDownSuite cleans up the test environment
func (s *SnapshotIntegrationTestSuite) TearDownSuite() {
	s.cancel()
	s.cleanupTestEnvironment()
}

// TestCompleteSnapshotWorkflow tests the end-to-end snapshot transformation process
func (s *SnapshotIntegrationTestSuite) TestCompleteSnapshotWorkflow() {
	testCases := []struct {
		name              string
		scenario          string
		patientID         string
		expectedLatency   time.Duration
		expectedCacheHit  bool
		expectedOverride  bool
		validationChecks  []func(*types.ClinicalSnapshot, *types.SafetyResponse) error
	}{
		{
			name:             "Happy Path - Routine Medication Review",
			scenario:         "routine_medication_review",
			patientID:        "patient-001",
			expectedLatency:  150 * time.Millisecond,
			expectedCacheHit: false, // First request
			expectedOverride: false,
			validationChecks: s.validateRoutineMedicationReview,
		},
		{
			name:             "Cache Hit - Subsequent Request",
			scenario:         "routine_medication_review",
			patientID:        "patient-001",
			expectedLatency:  20 * time.Millisecond,
			expectedCacheHit: true,
			expectedOverride: false,
			validationChecks: s.validateCacheHitScenario,
		},
		{
			name:             "High-Risk Scenario - Override Required",
			scenario:         "high_risk_drug_interaction",
			patientID:        "patient-002",
			expectedLatency:  180 * time.Millisecond,
			expectedCacheHit: false,
			expectedOverride: true,
			validationChecks: s.validateHighRiskScenario,
		},
		{
			name:             "Complex Patient - Multiple Conditions",
			scenario:         "complex_patient_multi_conditions",
			patientID:        "patient-003",
			expectedLatency:  200 * time.Millisecond,
			expectedCacheHit: false,
			expectedOverride: false,
			validationChecks: s.validateComplexPatientScenario,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.runSnapshotWorkflowTest(tc)
		})
	}
}

// runSnapshotWorkflowTest executes a complete snapshot workflow test
func (s *SnapshotIntegrationTestSuite) runSnapshotWorkflowTest(tc struct {
	name              string
	scenario          string
	patientID         string
	expectedLatency   time.Duration
	expectedCacheHit  bool
	expectedOverride  bool
	validationChecks  []func(*types.ClinicalSnapshot, *types.SafetyResponse) error
}) {
	startTime := time.Now()
	
	// Step 1: Create safety request
	request := s.createTestRequest(tc.scenario, tc.patientID)
	
	// Step 2: Process through context builder
	snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
	require.NoError(s.T(), err, "Failed to build clinical snapshot")
	assert.NotNil(s.T(), snapshot, "Snapshot should not be nil")
	
	// Step 3: Validate cache behavior
	cacheHit := s.validateCacheBehavior(snapshot, tc.expectedCacheHit)
	assert.Equal(s.T(), tc.expectedCacheHit, cacheHit, "Cache behavior mismatch")
	
	// Step 4: Process safety decision
	response, err := s.processSafety Decision(s.ctx, request, snapshot)
	require.NoError(s.T(), err, "Failed to process safety decision")
	assert.NotNil(s.T(), response, "Safety response should not be nil")
	
	// Step 5: Check if override is required
	overrideRequired := s.isOverrideRequired(response)
	assert.Equal(s.T(), tc.expectedOverride, overrideRequired, 
		"Override requirement mismatch")
	
	// Step 6: If override required, generate enhanced token
	var overrideToken *types.EnhancedOverrideToken
	if overrideRequired {
		overrideToken, err = s.tokenGenerator.GenerateEnhancedToken(
			request, response, snapshot)
		require.NoError(s.T(), err, "Failed to generate enhanced override token")
		assert.NotNil(s.T(), overrideToken, "Override token should not be nil")
		
		// Validate token structure
		s.validateEnhancedToken(overrideToken, request, response, snapshot)
	}
	
	// Step 7: Publish learning events
	err = s.eventPublisher.PublishSafetyDecisionEvent(s.ctx, request, response, snapshot)
	require.NoError(s.T(), err, "Failed to publish learning event")
	
	// Step 8: Test decision reproducibility
	if overrideToken != nil {
		s.testDecisionReproducibility(overrideToken, request, response, snapshot)
	}
	
	// Step 9: Validate performance
	actualLatency := time.Since(startTime)
	assert.LessOrEqual(s.T(), actualLatency, tc.expectedLatency,
		fmt.Sprintf("Latency %v exceeded expected %v", actualLatency, tc.expectedLatency))
	
	// Step 10: Run scenario-specific validation checks
	for _, check := range tc.validationChecks {
		err := check(snapshot, response)
		assert.NoError(s.T(), err, "Validation check failed")
	}
	
	// Step 11: Validate learning analytics
	s.validateLearningAnalytics(request.PatientID, overrideToken)
}

// TestSnapshotCacheIntegrity tests cache consistency and integrity
func (s *SnapshotIntegrationTestSuite) TestSnapshotCacheIntegrity() {
	patientID := "cache-test-patient-001"
	
	// Create initial snapshot
	request1 := s.createTestRequest("routine_check", patientID)
	snapshot1, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request1)
	require.NoError(s.T(), err)
	
	// Verify cache entry
	cacheKey := s.snapshotCache.GenerateCacheKey(request1)
	cachedSnapshot, found := s.snapshotCache.Get(cacheKey)
	assert.True(s.T(), found, "Snapshot should be cached")
	assert.Equal(s.T(), snapshot1.SnapshotID, cachedSnapshot.SnapshotID)
	
	// Test cache invalidation
	s.snapshotCache.InvalidatePatientCache(patientID)
	_, found = s.snapshotCache.Get(cacheKey)
	assert.False(s.T(), found, "Snapshot should be invalidated")
	
	// Test cache consistency after data update
	request2 := s.createTestRequestWithDataUpdate(patientID)
	snapshot2, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request2)
	require.NoError(s.T(), err)
	
	// Verify new snapshot reflects updates
	assert.NotEqual(s.T(), snapshot1.DataHash, snapshot2.DataHash,
		"Data hash should differ after update")
	assert.True(s.T(), snapshot2.CreatedAt.After(snapshot1.CreatedAt),
		"New snapshot should have later timestamp")
}

// TestParallelSnapshotProcessing tests concurrent snapshot processing
func (s *SnapshotIntegrationTestSuite) TestParallelSnapshotProcessing() {
	numConcurrentRequests := 10
	patientIDs := make([]string, numConcurrentRequests)
	
	for i := 0; i < numConcurrentRequests; i++ {
		patientIDs[i] = fmt.Sprintf("parallel-patient-%03d", i+1)
	}
	
	// Channel to collect results
	results := make(chan *parallelTestResult, numConcurrentRequests)
	
	// Launch concurrent snapshot processing
	startTime := time.Now()
	for i, patientID := range patientIDs {
		go s.processSnapshotConcurrently(patientID, i, results)
	}
	
	// Collect and validate results
	var allResults []*parallelTestResult
	for i := 0; i < numConcurrentRequests; i++ {
		result := <-results
		allResults = append(allResults, result)
		
		// Validate individual result
		assert.NoError(s.T(), result.Error, 
			fmt.Sprintf("Concurrent processing failed for patient %s", result.PatientID))
		assert.NotNil(s.T(), result.Snapshot)
		assert.LessOrEqual(s.T(), result.ProcessingTime, 300*time.Millisecond,
			"Individual processing time exceeded threshold")
	}
	
	totalTime := time.Since(startTime)
	
	// Validate overall performance
	assert.LessOrEqual(s.T(), totalTime, 500*time.Millisecond,
		"Total parallel processing time exceeded threshold")
	
	// Validate no race conditions or data corruption
	s.validateParallelProcessingIntegrity(allResults)
}

// TestOverrideTokenReproducibility tests enhanced token reproducibility
func (s *SnapshotIntegrationTestSuite) TestOverrideTokenReproducibility() {
	// Create high-risk scenario requiring override
	request := s.createTestRequest("high_risk_drug_interaction", "repro-patient-001")
	snapshot, err := s.contextBuilder.BuildClinicalSnapshot(s.ctx, request)
	require.NoError(s.T(), err)
	
	// Generate unsafe response requiring override
	response := s.generateUnsafeResponse(request, snapshot)
	
	// Generate enhanced override token
	token, err := s.tokenGenerator.GenerateEnhancedToken(request, response, snapshot)
	require.NoError(s.T(), err)
	
	// Test immediate reproducibility
	replayResult, err := s.replayService.ReplayDecision(s.ctx, token)
	require.NoError(s.T(), err)
	
	// Validate reproducibility score
	assert.GreaterOrEqual(s.T(), replayResult.ReproducibilityScore, 0.95,
		"Reproducibility score below threshold")
	
	// Test reproducibility after cache eviction
	s.snapshotCache.InvalidatePatientCache(request.PatientID)
	
	replayResult2, err := s.replayService.ReplayDecision(s.ctx, token)
	require.NoError(s.T(), err)
	
	assert.GreaterOrEqual(s.T(), replayResult2.ReproducibilityScore, 0.95,
		"Reproducibility score degraded after cache eviction")
	
	// Test reproducibility with system restart simulation
	s.simulateSystemRestart()
	
	replayResult3, err := s.replayService.ReplayDecision(s.ctx, token)
	require.NoError(s.T(), err)
	
	assert.GreaterOrEqual(s.T(), replayResult3.ReproducibilityScore, 0.95,
		"Reproducibility score degraded after system restart")
}

// TestLearningAnalyticsIntegration tests learning gateway integration
func (s *SnapshotIntegrationTestSuite) TestLearningAnalyticsIntegration() {
	patientID := "learning-test-patient-001"
	
	// Generate multiple override events to create patterns
	overrideTokens := s.generateOverridePattern(patientID, 5)
	
	// Wait for event processing
	time.Sleep(2 * time.Second)
	
	// Analyze override patterns
	analysis, err := s.overrideAnalyzer.AnalyzeOverridePatterns(s.ctx, patientID)
	require.NoError(s.T(), err)
	
	// Validate analysis results
	assert.Equal(s.T(), 5, analysis.BasicStats.TotalOverrides)
	assert.NotEmpty(s.T(), analysis.DetectedPatterns)
	assert.NotNil(s.T(), analysis.RiskPrediction)
	
	// Validate Kafka events were published
	publishedEvents := s.mockKafka.GetPublishedEvents()
	assert.GreaterOrEqual(s.T(), len(publishedEvents), 5,
		"Expected at least 5 published events")
	
	// Validate event structure
	for _, event := range publishedEvents {
		s.validateLearningEvent(event)
	}
}

// Helper methods for test setup and validation

func (s *SnapshotIntegrationTestSuite) loadTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:         8030,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Cache: config.CacheConfig{
			TTL:              1 * time.Hour,
			MaxSize:          1000,
			EvictionPolicy:   "lru",
			CompressionLevel: 6,
		},
		Learning: config.LearningConfig{
			Enabled: true,
			EventPublisher: config.EventPublisherConfig{
				EnableEventPublishing: true,
				BatchSize:            100,
				FlushIntervalSeconds: 1, // Fast for testing
				RetryAttempts:       3,
			},
		},
		Reproducibility: config.ReproducibilityConfig{
			Enabled:                true,
			MaxConcurrentReplays:   5,
			ReplayTimeoutMS:       30000,
			CacheReplayResults:    true,
			VerifyReproducibility: true,
		},
	}
}

func (s *SnapshotIntegrationTestSuite) setupMockServices() {
	s.mockCAEEngine = &engines.MockCAEEngine{}
	s.mockKafka = NewMockKafkaProducer()
	s.mockEventStore = NewMockOverrideEventStore()
}

func (s *SnapshotIntegrationTestSuite) setupCoreServices() {
	var err error
	
	// Initialize cache
	s.snapshotCache, err = cache.NewSnapshotCache(&s.testConfig.Cache, s.logger)
	require.NoError(s.T(), err)
	
	// Initialize cache manager
	s.cacheManager = context.NewCacheManager(s.snapshotCache, s.logger)
	
	// Initialize assembly service
	s.assemblyService = context.NewAssemblyService(s.testConfig, s.logger)
	
	// Initialize context builder
	s.contextBuilder = context.NewContextBuilder(
		s.cacheManager,
		s.assemblyService,
		s.testConfig,
		s.logger,
	)
}

func (s *SnapshotIntegrationTestSuite) setupEnhancedServices() {
	// Initialize token generator
	signingKey := []byte("test-signing-key-32-bytes-long!!")
	s.tokenGenerator = override.NewEnhancedTokenGenerator(signingKey, s.logger)
	
	// Initialize override service
	s.overrideService = override.NewSnapshotAwareOverrideService(
		s.tokenGenerator,
		s.testConfig,
		s.logger,
	)
	
	// Initialize event publisher
	s.eventPublisher = learning.NewLearningEventPublisher(
		s.mockKafka,
		&s.testConfig.Learning.EventPublisher,
		s.logger,
	)
	
	// Initialize override analyzer
	s.overrideAnalyzer = learning.NewOverrideAnalyzer(
		s.mockEventStore,
		&s.testConfig.Learning.OverrideAnalyzer,
		s.logger,
	)
	
	// Initialize replay service
	s.replayService = reproducibility.NewDecisionReplayService(
		s.snapshotCache,
		s.mockCAEEngine,
		&s.testConfig.Reproducibility,
		s.logger,
	)
}

// Additional helper structs and methods

type parallelTestResult struct {
	PatientID      string
	Snapshot       *types.ClinicalSnapshot
	ProcessingTime time.Duration
	Error          error
}

// Mock implementations for testing

type MockKafkaProducer struct {
	publishedEvents []interface{}
	mu              sync.RWMutex
}

func NewMockKafkaProducer() *MockKafkaProducer {
	return &MockKafkaProducer{
		publishedEvents: make([]interface{}, 0),
	}
}

func (m *MockKafkaProducer) PublishEvent(topic string, event interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedEvents = append(m.publishedEvents, event)
	return nil
}

func (m *MockKafkaProducer) GetPublishedEvents() []interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]interface{}(nil), m.publishedEvents...)
}

type MockOverrideEventStore struct {
	events map[string][]*learning.OverrideEvent
	mu     sync.RWMutex
}

func NewMockOverrideEventStore() *MockOverrideEventStore {
	return &MockOverrideEventStore{
		events: make(map[string][]*learning.OverrideEvent),
	}
}

func (m *MockOverrideEventStore) StoreEvent(event *learning.OverrideEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.events[event.PatientID] == nil {
		m.events[event.PatientID] = make([]*learning.OverrideEvent, 0)
	}
	m.events[event.PatientID] = append(m.events[event.PatientID], event)
	return nil
}

func (m *MockOverrideEventStore) GetEvents(patientID string, since time.Time) ([]*learning.OverrideEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	events, exists := m.events[patientID]
	if !exists {
		return nil, nil
	}
	
	var result []*learning.OverrideEvent
	for _, event := range events {
		if event.Timestamp.After(since) || event.Timestamp.Equal(since) {
			result = append(result, event)
		}
	}
	return result, nil
}

// TestSnapshotIntegrationTestSuite runs the integration test suite
func TestSnapshotIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SnapshotIntegrationTestSuite))
}