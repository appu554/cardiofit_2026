package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-drug-rules/internal/cache"
	"kb-drug-rules/internal/metrics"
	"kb-drug-rules/internal/models"
	"kb-drug-rules/internal/services"
)

// MockKB1Cache implements cache.KB1CacheInterface for testing
type MockKB1Cache struct {
	mock.Mock
}

func (m *MockKB1Cache) GetDosingRule(drugCode, contextHash string) ([]byte, error) {
	args := m.Called(drugCode, contextHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKB1Cache) SetDosingRule(drugCode, contextHash string, data []byte) error {
	args := m.Called(drugCode, contextHash, data)
	return args.Error(0)
}

func (m *MockKB1Cache) InvalidateDrugCode(drugCode string) error {
	args := m.Called(drugCode)
	return args.Error(0)
}

func (m *MockKB1Cache) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockKB1Cache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	args := m.Called(ctx, key, value, ttlSeconds)
	return args.Error(0)
}

func (m *MockKB1Cache) GenerateContextHash(patientContext map[string]interface{}) string {
	args := m.Called(patientContext)
	return args.String(0)
}

func (m *MockKB1Cache) PrewarmTopMedications(drugCodes []string, getRuleFunc func(string) ([]byte, error)) error {
	args := m.Called(drugCodes, getRuleFunc)
	return args.Error(0)
}

func (m *MockKB1Cache) ListenForInvalidationEvents() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockKB1Cache) GetKB1Stats() (*cache.KB1CacheStats, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*cache.KB1CacheStats), args.Error(1)
}

func (m *MockKB1Cache) Ping() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockKB1Cache) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockKB1Cache) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockKB1Cache) InvalidatePattern(pattern string) error {
	args := m.Called(pattern)
	return args.Error(0)
}

func (m *MockKB1Cache) GetStats() (map[string]interface{}, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

type MockGovernance struct {
	mock.Mock
}

func (m *MockGovernance) IsApproved(drugID, version string) (bool, error) {
	args := m.Called(drugID, version)
	return args.Bool(0), args.Error(1)
}

func (m *MockGovernance) GetApproval(drugID, version string) (*services.ApprovalDetails, error) {
	args := m.Called(drugID, version)
	return args.Get(0).(*services.ApprovalDetails), args.Error(1)
}

func (m *MockGovernance) SubmitForApproval(request *services.ApprovalRequest) (*services.ApprovalTicket, error) {
	args := m.Called(request)
	return args.Get(0).(*services.ApprovalTicket), args.Error(1)
}

func (m *MockGovernance) ReviewSubmission(ticketID string, review *services.ReviewSubmission) error {
	args := m.Called(ticketID, review)
	return args.Error(0)
}

func (m *MockGovernance) VerifySignature(content, signature, signer string) (bool, error) {
	args := m.Called(content, signature, signer)
	return args.Bool(0), args.Error(1)
}

func (m *MockGovernance) SignContent(content, signer string) (string, error) {
	args := m.Called(content, signer)
	return args.String(0), args.Error(1)
}

// Test setup helper
func setupTestServer() (*Server, *gorm.DB, *MockKB1Cache, *MockGovernance) {
	// Setup in-memory SQLite database
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})

	// Auto-migrate test tables
	db.AutoMigrate(&models.DrugRulePack{})

	// Create mock dependencies
	mockCache := &MockKB1Cache{}
	mockGovernance := &MockGovernance{}
	mockMetrics := metrics.NewNoOpCollector()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Create server
	server := NewServer(&ServerConfig{
		DB:         db,
		Cache:      mockCache,
		Governance: mockGovernance,
		Metrics:    mockMetrics,
		Logger:     logger,
	})

	return server, db, mockCache, mockGovernance
}

// Test cases

func TestHealthCheck(t *testing.T) {
	server, _, mockCache, _ := setupTestServer()
	
	// Setup mocks
	mockCache.On("Ping").Return(nil)
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server.RegisterRoutes(router)
	
	// Create request
	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	// Execute request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response.Status)
	assert.Contains(t, response.Checks, "database")
	assert.Contains(t, response.Checks, "cache")
	
	mockCache.AssertExpectations(t)
}

func TestGetDrugRules_Success(t *testing.T) {
	server, db, mockCache, _ := setupTestServer()
	
	// Insert test data
	testRulePack := &models.DrugRulePack{
		DrugID:         "metformin",
		Version:        "1.0.0",
		ContentSHA:     "abc123",
		SignedBy:       "test-signer",
		SignatureValid: true,
		Regions:        []string{"US"},
		Content: models.DrugRuleContent{
			Meta: models.RuleMetadata{
				DrugName:         "Metformin",
				TherapeuticClass: []string{"Antidiabetic"},
			},
			DoseCalculation: models.DoseCalculation{
				BaseFormula:  "500mg BID",
				MaxDailyDose: 2000,
				MinDailyDose: 500,
			},
		},
	}
	db.Create(testRulePack)
	
	// Setup mocks
	mockCache.On("Get", mock.AnythingOfType("string")).Return([]byte(nil), nil)
	mockCache.On("Set", mock.AnythingOfType("string"), mock.AnythingOfType("[]uint8"), mock.AnythingOfType("time.Duration")).Return(nil)
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server.RegisterRoutes(router)
	
	// Create request
	req, _ := http.NewRequest("GET", "/v1/items/metformin", nil)
	w := httptest.NewRecorder()
	
	// Execute request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.DrugRulesResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "metformin", response.DrugID)
	assert.Equal(t, "1.0.0", response.Version)
	assert.True(t, response.SignatureValid)
	assert.Equal(t, "Metformin", response.Content.Meta.DrugName)
	
	mockCache.AssertExpectations(t)
}

func TestGetDrugRules_NotFound(t *testing.T) {
	server, _, mockCache, _ := setupTestServer()
	
	// Setup mocks
	mockCache.On("Get", mock.AnythingOfType("string")).Return([]byte(nil), nil)
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server.RegisterRoutes(router)
	
	// Create request
	req, _ := http.NewRequest("GET", "/v1/items/nonexistent", nil)
	w := httptest.NewRecorder()
	
	// Execute request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusNotFound, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Not Found", response["error"])
	assert.Contains(t, response["message"], "Drug rules not found")
	
	mockCache.AssertExpectations(t)
}

func TestValidateRules_Success(t *testing.T) {
	server, _, _, _ := setupTestServer()
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server.RegisterRoutes(router)
	
	// Create valid TOML content
	validationRequest := models.ValidationRequest{
		Content: `
[meta]
drug_name = "Test Drug"
therapeutic_class = ["Test Class"]

[dose_calculation]
base_formula = "100mg daily"
max_daily_dose = 200.0
min_daily_dose = 50.0

[[dose_calculation.adjustment_factors]]
factor = "age"
condition = "age > 65"
multiplier = 0.8

[safety_verification]
contraindications = []
warnings = []
precautions = []
interaction_checks = []
lab_monitoring = []

monitoring_requirements = []
regional_variations = {}
		`,
		Regions: []string{"US"},
	}
	
	requestBody, _ := json.Marshal(validationRequest)
	req, _ := http.NewRequest("POST", "/v1/validate", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Execute request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.ValidationResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Valid)
	assert.Empty(t, response.Errors)
}

func TestValidateRules_InvalidTOML(t *testing.T) {
	server, _, _, _ := setupTestServer()
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server.RegisterRoutes(router)
	
	// Create invalid TOML content
	validationRequest := models.ValidationRequest{
		Content: `invalid toml content [[[`,
		Regions: []string{"US"},
	}
	
	requestBody, _ := json.Marshal(validationRequest)
	req, _ := http.NewRequest("POST", "/v1/validate", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Execute request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Bad Request", response["error"])
	assert.Contains(t, response["message"], "Invalid TOML content")
}

func TestHotloadRules_Success(t *testing.T) {
	server, _, mockCache, mockGovernance := setupTestServer()
	
	// Setup mocks
	mockGovernance.On("IsApproved", "test-drug", "1.0.0").Return(true, nil)
	mockGovernance.On("VerifySignature", mock.AnythingOfType("string"), "test-signature", "test-signer").Return(true, nil)
	mockGovernance.On("GetApproval", "test-drug", "1.0.0").Return(&services.ApprovalDetails{
		ClinicalReviewer: "dr.test",
		ClinicalReviewDate: time.Now(),
	}, nil)
	mockCache.On("InvalidatePattern", mock.AnythingOfType("string")).Return(nil)
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server.RegisterRoutes(router)
	
	// Create hotload request
	hotloadRequest := models.HotloadRequest{
		DrugID:    "test-drug",
		Version:   "1.0.0",
		Content:   `[meta]\ndrug_name = "Test Drug"\ntherapeutic_class = ["Test"]\n[dose_calculation]\nbase_formula = "100mg"\nmax_daily_dose = 200.0\nmin_daily_dose = 50.0\n[safety_verification]\ncontraindications = []\nwarnings = []\nprecautions = []\ninteraction_checks = []\nlab_monitoring = []\nmonitoring_requirements = []\nregional_variations = {}`,
		Signature: "test-signature",
		SignedBy:  "test-signer",
		Regions:   []string{"US"},
	}
	
	requestBody, _ := json.Marshal(hotloadRequest)
	req, _ := http.NewRequest("POST", "/v1/hotload", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Execute request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.HotloadResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "test-drug", response.DrugID)
	assert.Equal(t, "1.0.0", response.Version)
	
	mockGovernance.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestHotloadRules_NotApproved(t *testing.T) {
	server, _, _, mockGovernance := setupTestServer()
	
	// Setup mocks
	mockGovernance.On("IsApproved", "test-drug", "1.0.0").Return(false, nil)
	
	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	server.RegisterRoutes(router)
	
	// Create hotload request
	hotloadRequest := models.HotloadRequest{
		DrugID:    "test-drug",
		Version:   "1.0.0",
		Content:   "test content",
		Signature: "test-signature",
		SignedBy:  "test-signer",
		Regions:   []string{"US"},
	}
	
	requestBody, _ := json.Marshal(hotloadRequest)
	req, _ := http.NewRequest("POST", "/v1/hotload", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	// Execute request
	router.ServeHTTP(w, req)
	
	// Assert response
	assert.Equal(t, http.StatusForbidden, w.Code)
	
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Forbidden", response["error"])
	assert.Contains(t, response["message"], "Governance approval required")
	
	mockGovernance.AssertExpectations(t)
}
