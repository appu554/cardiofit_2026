package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/registry"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// MockContextService implements ContextAssemblyService for testing
type MockContextService struct {
	mock.Mock
}

func (m *MockContextService) AssembleContext(ctx context.Context, patientID string) (*types.ClinicalContext, error) {
	args := m.Called(ctx, patientID)
	return args.Get(0).(*types.ClinicalContext), args.Error(1)
}

// MockSafetyEngine implements SafetyEngine for testing
type MockSafetyEngine struct {
	mock.Mock
	id           string
	name         string
	capabilities []string
}

func (m *MockSafetyEngine) ID() string                { return m.id }
func (m *MockSafetyEngine) Name() string              { return m.name }
func (m *MockSafetyEngine) Capabilities() []string   { return m.capabilities }
func (m *MockSafetyEngine) HealthCheck() error       { return m.Called().Error(0) }
func (m *MockSafetyEngine) Initialize(config types.EngineConfig) error { return m.Called(config).Error(0) }
func (m *MockSafetyEngine) Shutdown() error          { return m.Called().Error(0) }

func (m *MockSafetyEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	args := m.Called(ctx, req, clinicalContext)
	return args.Get(0).(*types.EngineResult), args.Error(1)
}

func TestOrchestrationEngine_ProcessSafetyRequest(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Performance: config.PerformanceConfig{
			RequestTimeoutMs:         200,
			ContextAssemblyTimeoutMs: 20,
			EngineExecutionTimeoutMs: 150,
		},
	}

	testLogger, err := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Create mock context service
	mockContextService := &MockContextService{}
	mockContext := &types.ClinicalContext{
		PatientID:        "patient_123",
		ContextVersion:   "v1_test",
		AssemblyTime:     time.Now(),
		DataSources:      []string{"fhir"},
		ActiveMedications: []types.Medication{
			{
				ID:   "med_1",
				Name: "aspirin",
			},
		},
	}
	mockContextService.On("AssembleContext", mock.Anything, "patient_123").Return(mockContext, nil)

	// Create engine registry
	engineRegistry := registry.NewEngineRegistry(cfg, testLogger)

	// Create mock engine
	mockEngine := &MockSafetyEngine{
		id:           "test_engine",
		name:         "Test Engine",
		capabilities: []string{"drug_interaction"},
	}
	mockEngine.On("Initialize", mock.Anything).Return(nil)
	mockEngine.On("HealthCheck").Return(nil)
	mockEngine.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(&types.EngineResult{
		EngineID:   "test_engine",
		EngineName: "Test Engine",
		Status:     types.SafetyStatusSafe,
		RiskScore:  0.1,
		Confidence: 0.9,
		Tier:       types.TierVetoCritical,
	}, nil)

	// Register mock engine
	err = engineRegistry.RegisterEngine(mockEngine, types.TierVetoCritical, 10)
	require.NoError(t, err)

	// Create orchestration engine
	orchestrator := NewOrchestrationEngine(
		engineRegistry,
		mockContextService,
		cfg,
		testLogger,
	)

	// Test request
	request := &types.SafetyRequest{
		RequestID:     "req_123",
		PatientID:     "patient_123",
		ClinicianID:   "clinician_123",
		ActionType:    "medication_order",
		Priority:      "normal",
		MedicationIDs: []string{"med_1"},
		Timestamp:     time.Now(),
		Source:        "test",
	}

	// Execute test
	ctx := context.Background()
	response, err := orchestrator.ProcessSafetyRequest(ctx, request)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "req_123", response.RequestID)
	assert.Equal(t, types.SafetyStatusSafe, response.Status)
	assert.Equal(t, 1, len(response.EngineResults))
	assert.Equal(t, "test_engine", response.EngineResults[0].EngineID)
	assert.True(t, response.ProcessingTime > 0)

	// Verify mocks
	mockContextService.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}

func TestOrchestrationEngine_ProcessSafetyRequest_ContextAssemblyFailure(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Performance: config.PerformanceConfig{
			RequestTimeoutMs:         200,
			ContextAssemblyTimeoutMs: 20,
			EngineExecutionTimeoutMs: 150,
		},
	}

	testLogger, err := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Create mock context service that fails
	mockContextService := &MockContextService{}
	mockContextService.On("AssembleContext", mock.Anything, "patient_123").Return((*types.ClinicalContext)(nil), assert.AnError)

	// Create engine registry
	engineRegistry := registry.NewEngineRegistry(cfg, testLogger)

	// Create orchestration engine
	orchestrator := NewOrchestrationEngine(
		engineRegistry,
		mockContextService,
		cfg,
		testLogger,
	)

	// Test request
	request := &types.SafetyRequest{
		RequestID:   "req_123",
		PatientID:   "patient_123",
		ClinicianID: "clinician_123",
		ActionType:  "medication_order",
		Priority:    "normal",
		Timestamp:   time.Now(),
		Source:      "test",
	}

	// Execute test
	ctx := context.Background()
	response, err := orchestrator.ProcessSafetyRequest(ctx, request)

	// Assertions
	require.NoError(t, err) // Should not return error, but error response
	assert.NotNil(t, response)
	assert.Equal(t, "req_123", response.RequestID)
	assert.Equal(t, types.SafetyStatusError, response.Status)
	assert.Equal(t, 1.0, response.RiskScore) // Maximum risk for errors

	// Verify mocks
	mockContextService.AssertExpectations(t)
}

func TestOrchestrationEngine_ProcessSafetyRequest_NoEngines(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Performance: config.PerformanceConfig{
			RequestTimeoutMs:         200,
			ContextAssemblyTimeoutMs: 20,
			EngineExecutionTimeoutMs: 150,
		},
	}

	testLogger, err := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Create mock context service
	mockContextService := &MockContextService{}
	mockContext := &types.ClinicalContext{
		PatientID:      "patient_123",
		ContextVersion: "v1_test",
		AssemblyTime:   time.Now(),
		DataSources:    []string{"fhir"},
	}
	mockContextService.On("AssembleContext", mock.Anything, "patient_123").Return(mockContext, nil)

	// Create empty engine registry
	engineRegistry := registry.NewEngineRegistry(cfg, testLogger)

	// Create orchestration engine
	orchestrator := NewOrchestrationEngine(
		engineRegistry,
		mockContextService,
		cfg,
		testLogger,
	)

	// Test request
	request := &types.SafetyRequest{
		RequestID:   "req_123",
		PatientID:   "patient_123",
		ClinicianID: "clinician_123",
		ActionType:  "medication_order",
		Priority:    "normal",
		Timestamp:   time.Now(),
		Source:      "test",
	}

	// Execute test
	ctx := context.Background()
	response, err := orchestrator.ProcessSafetyRequest(ctx, request)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "req_123", response.RequestID)
	assert.Equal(t, types.SafetyStatusError, response.Status)

	// Verify mocks
	mockContextService.AssertExpectations(t)
}

func TestOrchestrationEngine_ProcessSafetyRequest_EngineFailure(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Performance: config.PerformanceConfig{
			RequestTimeoutMs:         200,
			ContextAssemblyTimeoutMs: 20,
			EngineExecutionTimeoutMs: 150,
		},
	}

	testLogger, err := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	// Create mock context service
	mockContextService := &MockContextService{}
	mockContext := &types.ClinicalContext{
		PatientID:      "patient_123",
		ContextVersion: "v1_test",
		AssemblyTime:   time.Now(),
		DataSources:    []string{"fhir"},
	}
	mockContextService.On("AssembleContext", mock.Anything, "patient_123").Return(mockContext, nil)

	// Create engine registry
	engineRegistry := registry.NewEngineRegistry(cfg, testLogger)

	// Create mock engine that fails
	mockEngine := &MockSafetyEngine{
		id:           "failing_engine",
		name:         "Failing Engine",
		capabilities: []string{"drug_interaction"},
	}
	mockEngine.On("Initialize", mock.Anything).Return(nil)
	mockEngine.On("HealthCheck").Return(nil)
	mockEngine.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return((*types.EngineResult)(nil), assert.AnError)

	// Register mock engine
	err = engineRegistry.RegisterEngine(mockEngine, types.TierVetoCritical, 10)
	require.NoError(t, err)

	// Create orchestration engine
	orchestrator := NewOrchestrationEngine(
		engineRegistry,
		mockContextService,
		cfg,
		testLogger,
	)

	// Test request
	request := &types.SafetyRequest{
		RequestID:     "req_123",
		PatientID:     "patient_123",
		ClinicianID:   "clinician_123",
		ActionType:    "medication_order",
		Priority:      "normal",
		MedicationIDs: []string{"med_1"},
		Timestamp:     time.Now(),
		Source:        "test",
	}

	// Execute test
	ctx := context.Background()
	response, err := orchestrator.ProcessSafetyRequest(ctx, request)

	// Assertions
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "req_123", response.RequestID)
	assert.Equal(t, types.SafetyStatusUnsafe, response.Status) // Tier 1 failure = UNSAFE
	assert.Equal(t, 1, len(response.EngineResults))
	assert.Equal(t, "failing_engine", response.EngineResults[0].EngineID)
	assert.NotEmpty(t, response.EngineResults[0].Error)

	// Verify mocks
	mockContextService.AssertExpectations(t)
	mockEngine.AssertExpectations(t)
}

func BenchmarkOrchestrationEngine_ProcessSafetyRequest(b *testing.B) {
	// Setup
	cfg := &config.Config{
		Performance: config.PerformanceConfig{
			RequestTimeoutMs:         200,
			ContextAssemblyTimeoutMs: 20,
			EngineExecutionTimeoutMs: 150,
		},
	}

	testLogger, _ := logger.New(config.LoggingConfig{
		Format: "json",
		Output: "stdout",
	})

	// Create mock context service
	mockContextService := &MockContextService{}
	mockContext := &types.ClinicalContext{
		PatientID:      "patient_123",
		ContextVersion: "v1_test",
		AssemblyTime:   time.Now(),
		DataSources:    []string{"fhir"},
	}
	mockContextService.On("AssembleContext", mock.Anything, mock.Anything).Return(mockContext, nil)

	// Create engine registry with mock engine
	engineRegistry := registry.NewEngineRegistry(cfg, testLogger)
	mockEngine := &MockSafetyEngine{
		id:           "bench_engine",
		name:         "Benchmark Engine",
		capabilities: []string{"drug_interaction"},
	}
	mockEngine.On("Initialize", mock.Anything).Return(nil)
	mockEngine.On("HealthCheck").Return(nil)
	mockEngine.On("Evaluate", mock.Anything, mock.Anything, mock.Anything).Return(&types.EngineResult{
		EngineID:   "bench_engine",
		EngineName: "Benchmark Engine",
		Status:     types.SafetyStatusSafe,
		RiskScore:  0.1,
		Confidence: 0.9,
		Tier:       types.TierVetoCritical,
	}, nil)

	engineRegistry.RegisterEngine(mockEngine, types.TierVetoCritical, 10)

	// Create orchestration engine
	orchestrator := NewOrchestrationEngine(
		engineRegistry,
		mockContextService,
		cfg,
		testLogger,
	)

	// Test request
	request := &types.SafetyRequest{
		RequestID:     "req_123",
		PatientID:     "patient_123",
		ClinicianID:   "clinician_123",
		ActionType:    "medication_order",
		Priority:      "normal",
		MedicationIDs: []string{"med_1"},
		Timestamp:     time.Now(),
		Source:        "benchmark",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = orchestrator.ProcessSafetyRequest(ctx, request)
	}
}
