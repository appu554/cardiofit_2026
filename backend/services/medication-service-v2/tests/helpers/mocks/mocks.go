package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/domain/repositories"
	"medication-service-v2/internal/application/services"
)

// MockMedicationRepository is a mock implementation of MedicationRepository
type MockMedicationRepository struct {
	mock.Mock
}

func NewMockMedicationRepository(t interface{}) *MockMedicationRepository {
	m := &MockMedicationRepository{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockMedicationRepository) CreateProposal(ctx context.Context, proposal *entities.MedicationProposal) error {
	args := m.Called(ctx, proposal)
	return args.Error(0)
}

func (m *MockMedicationRepository) GetProposalByID(ctx context.Context, id uuid.UUID) (*entities.MedicationProposal, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.MedicationProposal), args.Error(1)
}

func (m *MockMedicationRepository) UpdateProposal(ctx context.Context, proposal *entities.MedicationProposal) error {
	args := m.Called(ctx, proposal)
	return args.Error(0)
}

func (m *MockMedicationRepository) DeleteProposal(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMedicationRepository) GetProposalsByPatientID(ctx context.Context, patientID uuid.UUID) ([]*entities.MedicationProposal, error) {
	args := m.Called(ctx, patientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.MedicationProposal), args.Error(1)
}

func (m *MockMedicationRepository) SearchProposals(ctx context.Context, criteria repositories.SearchCriteria) ([]*entities.MedicationProposal, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.MedicationProposal), args.Error(1)
}

func (m *MockMedicationRepository) GetProposalStatistics(ctx context.Context, timeRange repositories.TimeRange) (*repositories.ProposalStatistics, error) {
	args := m.Called(ctx, timeRange)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repositories.ProposalStatistics), args.Error(1)
}

// MockRecipeRepository is a mock implementation of RecipeRepository
type MockRecipeRepository struct {
	mock.Mock
}

func NewMockRecipeRepository(t interface{}) *MockRecipeRepository {
	m := &MockRecipeRepository{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockRecipeRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.Recipe, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Recipe), args.Error(1)
}

func (m *MockRecipeRepository) GetByProtocolID(ctx context.Context, protocolID string) (*entities.Recipe, error) {
	args := m.Called(ctx, protocolID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Recipe), args.Error(1)
}

func (m *MockRecipeRepository) Create(ctx context.Context, recipe *entities.Recipe) error {
	args := m.Called(ctx, recipe)
	return args.Error(0)
}

func (m *MockRecipeRepository) Update(ctx context.Context, recipe *entities.Recipe) error {
	args := m.Called(ctx, recipe)
	return args.Error(0)
}

func (m *MockRecipeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRecipeRepository) List(ctx context.Context, limit, offset int) ([]*entities.Recipe, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Recipe), args.Error(1)
}

func (m *MockRecipeRepository) Search(ctx context.Context, criteria repositories.RecipeSearchCriteria) ([]*entities.Recipe, error) {
	args := m.Called(ctx, criteria)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entities.Recipe), args.Error(1)
}

// MockRecipeService is a mock implementation of RecipeService
type MockRecipeService struct {
	mock.Mock
}

func NewMockRecipeService(t interface{}) *MockRecipeService {
	m := &MockRecipeService{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockRecipeService) ResolveRecipe(ctx context.Context, request *services.ResolveRecipeRequest) (*services.ResolveRecipeResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ResolveRecipeResponse), args.Error(1)
}

func (m *MockRecipeService) GetRecipeByID(ctx context.Context, id uuid.UUID) (*entities.Recipe, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Recipe), args.Error(1)
}

// MockSnapshotService is a mock implementation of SnapshotService
type MockSnapshotService struct {
	mock.Mock
}

func NewMockSnapshotService(t interface{}) *MockSnapshotService {
	m := &MockSnapshotService{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockSnapshotService) CreateSnapshot(ctx context.Context, request *services.CreateSnapshotRequest) (*services.CreateSnapshotResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.CreateSnapshotResponse), args.Error(1)
}

func (m *MockSnapshotService) GetSnapshotByID(ctx context.Context, id uuid.UUID) (*entities.Snapshot, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.Snapshot), args.Error(1)
}

// MockClinicalEngineService is a mock implementation of ClinicalEngineService
type MockClinicalEngineService struct {
	mock.Mock
}

func NewMockClinicalEngineService(t interface{}) *MockClinicalEngineService {
	m := &MockClinicalEngineService{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockClinicalEngineService) CalculateDosages(ctx context.Context, request *services.CalculateDosagesRequest) (*services.CalculateDosagesResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.CalculateDosagesResponse), args.Error(1)
}

func (m *MockClinicalEngineService) ValidateProposal(ctx context.Context, request *services.ValidateProposalEngineRequest) (*services.ValidationResults, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ValidationResults), args.Error(1)
}

func (m *MockClinicalEngineService) FinalSafetyCheck(ctx context.Context, request *services.FinalSafetyCheckRequest) (*services.FinalSafetyCheckResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.FinalSafetyCheckResponse), args.Error(1)
}

// MockAuditService is a mock implementation of AuditService
type MockAuditService struct {
	mock.Mock
}

func NewMockAuditService(t interface{}) *MockAuditService {
	m := &MockAuditService{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockAuditService) RecordEvent(ctx context.Context, event *services.AuditEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// MockNotificationService is a mock implementation of NotificationService
type MockNotificationService struct {
	mock.Mock
}

func NewMockNotificationService(t interface{}) *MockNotificationService {
	m := &MockNotificationService{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockNotificationService) SendNotification(ctx context.Context, notification *services.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

func (m *MockNotificationService) SendAlert(ctx context.Context, alert *services.AlertNotification) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

// MockRedisClient is a mock implementation of Redis client
type MockRedisClient struct {
	mock.Mock
}

func NewMockRedisClient(t interface{}) *MockRedisClient {
	m := &MockRedisClient{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockRedisClient) GetRecipeResolution(ctx context.Context, key string) (*entities.RecipeResolution, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.RecipeResolution), args.Error(1)
}

func (m *MockRedisClient) SetRecipeResolution(ctx context.Context, key string, resolution *entities.RecipeResolution, ttl time.Duration) error {
	args := m.Called(ctx, key, resolution, ttl)
	return args.Error(0)
}

func (m *MockRedisClient) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

// MockRustClinicalEngine is a mock implementation of the Rust clinical engine client
type MockRustClinicalEngine struct {
	mock.Mock
}

func NewMockRustClinicalEngine(t interface{}) *MockRustClinicalEngine {
	m := &MockRustClinicalEngine{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockRustClinicalEngine) CalculateDosage(ctx context.Context, request *services.RustCalculationRequest) (*services.RustCalculationResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.RustCalculationResponse), args.Error(1)
}

func (m *MockRustClinicalEngine) ValidateSafety(ctx context.Context, request *services.RustSafetyRequest) (*services.RustSafetyResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.RustSafetyResponse), args.Error(1)
}

func (m *MockRustClinicalEngine) HealthCheck(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockApolloFederationService is a mock implementation of Apollo Federation service
type MockApolloFederationService struct {
	mock.Mock
}

func NewMockApolloFederationService(t interface{}) *MockApolloFederationService {
	m := &MockApolloFederationService{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockApolloFederationService) QueryKnowledgeBase(ctx context.Context, query string, variables map[string]interface{}) (*services.GraphQLResponse, error) {
	args := m.Called(ctx, query, variables)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.GraphQLResponse), args.Error(1)
}

func (m *MockApolloFederationService) GetDrugInteractions(ctx context.Context, drugName string) (*services.DrugInteractionsResponse, error) {
	args := m.Called(ctx, drugName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.DrugInteractionsResponse), args.Error(1)
}

// MockContextGateway is a mock implementation of Context Gateway
type MockContextGateway struct {
	mock.Mock
}

func NewMockContextGateway(t interface{}) *MockContextGateway {
	m := &MockContextGateway{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockContextGateway) CreateSnapshot(ctx context.Context, request *services.ContextSnapshotRequest) (*services.ContextSnapshotResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ContextSnapshotResponse), args.Error(1)
}

func (m *MockContextGateway) GetSnapshot(ctx context.Context, snapshotID uuid.UUID) (*services.ContextSnapshotResponse, error) {
	args := m.Called(ctx, snapshotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.ContextSnapshotResponse), args.Error(1)
}

// MockWorkflowOrchestrator is a mock implementation of Workflow Orchestrator
type MockWorkflowOrchestrator struct {
	mock.Mock
}

func NewMockWorkflowOrchestrator(t interface{}) *MockWorkflowOrchestrator {
	m := &MockWorkflowOrchestrator{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockWorkflowOrchestrator) ExecutePhase(ctx context.Context, phase services.WorkflowPhase, input *services.PhaseInput) (*services.PhaseOutput, error) {
	args := m.Called(ctx, phase, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.PhaseOutput), args.Error(1)
}

func (m *MockWorkflowOrchestrator) ExecuteWorkflow(ctx context.Context, request *services.WorkflowRequest) (*services.WorkflowResponse, error) {
	args := m.Called(ctx, request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.WorkflowResponse), args.Error(1)
}

// MockHTTPClient is a mock implementation of HTTP client for external services
type MockHTTPClient struct {
	mock.Mock
}

func NewMockHTTPClient(t interface{}) *MockHTTPClient {
	m := &MockHTTPClient{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockHTTPClient) Get(ctx context.Context, url string) (*services.HTTPResponse, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.HTTPResponse), args.Error(1)
}

func (m *MockHTTPClient) Post(ctx context.Context, url string, body interface{}) (*services.HTTPResponse, error) {
	args := m.Called(ctx, url, body)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.HTTPResponse), args.Error(1)
}

// MockMetricsService is a mock implementation of Metrics service
type MockMetricsService struct {
	mock.Mock
}

func NewMockMetricsService(t interface{}) *MockMetricsService {
	m := &MockMetricsService{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockMetricsService) RecordCounter(name string, value int64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsService) RecordDuration(name string, duration time.Duration) {
	m.Called(name, duration)
}

func (m *MockMetricsService) RecordGauge(name string, value float64, labels map[string]string) {
	m.Called(name, value, labels)
}

func (m *MockMetricsService) RecordError(operation string, err error) {
	m.Called(operation, err)
}

// MockDatabaseClient is a mock implementation of database client
type MockDatabaseClient struct {
	mock.Mock
}

func NewMockDatabaseClient(t interface{}) *MockDatabaseClient {
	m := &MockDatabaseClient{}
	if t != nil {
		m.Test(t)
	}
	return m
}

func (m *MockDatabaseClient) Query(ctx context.Context, query string, args ...interface{}) (*services.QueryResult, error) {
	callArgs := []interface{}{ctx, query}
	callArgs = append(callArgs, args...)
	mockArgs := m.Called(callArgs...)
	if mockArgs.Get(0) == nil {
		return nil, mockArgs.Error(1)
	}
	return mockArgs.Get(0).(*services.QueryResult), mockArgs.Error(1)
}

func (m *MockDatabaseClient) Execute(ctx context.Context, query string, args ...interface{}) error {
	callArgs := []interface{}{ctx, query}
	callArgs = append(callArgs, args...)
	mockArgs := m.Called(callArgs...)
	return mockArgs.Error(0)
}

func (m *MockDatabaseClient) BeginTx(ctx context.Context) (*services.Transaction, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.Transaction), args.Error(1)
}