package services

import (
	"database/sql"
	"testing"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/services"
	"kb-7-terminology/tests/fixtures"
	"kb-7-terminology/tests/mocks"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTerminologyService_GetTerminologySystem(t *testing.T) {
	tests := []struct {
		name           string
		identifier     string
		mockSetup      func(*mocks.MockDB)
		expectedResult *models.TerminologySystem
		expectedError  string
	}{
		{
			name:       "successful retrieval by ID",
			identifier: "test-system-001",
			mockSetup: func(mockDB *mocks.MockDB) {
				rows := mocks.NewMockRows()
				rows.On("Scan", mock.AnythingOfType("*string"), mock.AnythingOfType("*string"), 
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*string"), mock.AnythingOfType("*models.JSONB"),
					mock.AnythingOfType("*[]string"), mock.AnythingOfType("*time.Time"),
					mock.AnythingOfType("*time.Time")).Return(nil).Run(func(args mock.Arguments) {
					// Simulate database row scan
					*args[0].(*string) = fixtures.TestTerminologySystem.ID
					*args[1].(*string) = fixtures.TestTerminologySystem.SystemURI
					*args[2].(*string) = fixtures.TestTerminologySystem.SystemName
					*args[3].(*string) = fixtures.TestTerminologySystem.Version
					*args[4].(*string) = fixtures.TestTerminologySystem.Description
					*args[5].(*string) = fixtures.TestTerminologySystem.Publisher
					*args[6].(*string) = fixtures.TestTerminologySystem.Status
					*args[8].(*[]string) = fixtures.TestTerminologySystem.SupportedRegions
					*args[9].(*time.Time) = fixtures.TestTerminologySystem.CreatedAt
					*args[10].(*time.Time) = fixtures.TestTerminologySystem.UpdatedAt
				})

				mockDB.On("QueryRow", mock.AnythingOfType("string"), "test-system-001").Return(rows)
			},
			expectedResult: &fixtures.TestTerminologySystem,
		},
		{
			name:       "system not found",
			identifier: "nonexistent-system",
			mockSetup: func(mockDB *mocks.MockDB) {
				rows := mocks.NewMockRows()
				rows.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, mock.Anything).Return(sql.ErrNoRows)

				mockDB.On("QueryRow", mock.AnythingOfType("string"), "nonexistent-system").Return(rows)
			},
			expectedError: "terminology system not found: nonexistent-system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := &mocks.MockDB{}
			mockCache := &mocks.MockRedisClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
			metricsCollector := metrics.NewCollector("test")

			tt.mockSetup(mockDB)

			service := services.NewTerminologyService(mockDB, mockCache, logger, metricsCollector)

			// Execute
			result, err := service.GetTerminologySystem(tt.identifier)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.ID, result.ID)
				assert.Equal(t, tt.expectedResult.SystemURI, result.SystemURI)
				assert.Equal(t, tt.expectedResult.SystemName, result.SystemName)
			}

			// Verify mocks
			mockDB.AssertExpectations(t)
		})
	}
}

func TestTerminologyService_LookupConcept(t *testing.T) {
	tests := []struct {
		name             string
		systemIdentifier string
		code             string
		mockSetup        func(*mocks.MockDB, *mocks.MockRedisClient)
		expectedResult   *models.LookupResult
		expectedError    string
	}{
		{
			name:             "successful lookup with cache miss",
			systemIdentifier: "http://snomed.info/sct",
			code:             "387517004",
			mockSetup: func(mockDB *mocks.MockDB, mockCache *mocks.MockRedisClient) {
				// Cache miss
				cacheKey := cache.ConceptCacheKey("http://snomed.info/sct", "387517004")
				mockCache.On("Get", cacheKey, mock.AnythingOfType("*models.LookupResult")).Return(assert.AnError)

				// Database query success
				rows := mocks.NewMockRows()
				concept := fixtures.SNOMEDTestConcepts[0]
				rows.On("Scan", 
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*[]string"), mock.AnythingOfType("*[]string"),
					mock.AnythingOfType("*models.JSONB"), mock.AnythingOfType("*models.JSONB"),
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*time.Time")).Return(nil).Run(func(args mock.Arguments) {
					*args[0].(*string) = concept.ID
					*args[1].(*string) = concept.SystemID
					*args[2].(*string) = concept.Code
					*args[3].(*string) = concept.Display
					*args[4].(*string) = concept.Definition
					*args[5].(*string) = concept.Status
					*args[6].(*[]string) = concept.ParentCodes
					*args[7].(*[]string) = concept.ChildCodes
					*args[10].(*string) = concept.ClinicalDomain
					*args[11].(*string) = concept.Specialty
					*args[12].(*time.Time) = concept.CreatedAt
					*args[13].(*time.Time) = concept.UpdatedAt
				})

				mockDB.On("QueryRow", mock.AnythingOfType("string"), 
					"http://snomed.info/sct", "387517004").Return(rows)

				// Cache set
				mockCache.On("Set", cacheKey, mock.AnythingOfType("models.LookupResult"), 
					1*time.Hour).Return(nil)
			},
			expectedResult: &models.LookupResult{
				Concept: fixtures.SNOMEDTestConcepts[0],
			},
		},
		{
			name:             "concept not found",
			systemIdentifier: "http://snomed.info/sct",
			code:             "INVALID123",
			mockSetup: func(mockDB *mocks.MockDB, mockCache *mocks.MockRedisClient) {
				// Cache miss
				cacheKey := cache.ConceptCacheKey("http://snomed.info/sct", "INVALID123")
				mockCache.On("Get", cacheKey, mock.AnythingOfType("*models.LookupResult")).Return(assert.AnError)

				// Database query - no rows
				rows := mocks.NewMockRows()
				rows.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything, mock.Anything, mock.Anything,
					mock.Anything, mock.Anything).Return(sql.ErrNoRows)

				mockDB.On("QueryRow", mock.AnythingOfType("string"), 
					"http://snomed.info/sct", "INVALID123").Return(rows)
			},
			expectedError: "concept not found: INVALID123 in system http://snomed.info/sct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := &mocks.MockDB{}
			mockCache := &mocks.MockRedisClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			metricsCollector := metrics.NewCollector("test")

			tt.mockSetup(mockDB, mockCache)

			service := services.NewTerminologyService(mockDB, mockCache, logger, metricsCollector)

			// Execute
			result, err := service.LookupConcept(tt.systemIdentifier, tt.code)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.Concept.Code, result.Concept.Code)
				assert.Equal(t, tt.expectedResult.Concept.Display, result.Concept.Display)
			}

			// Verify mocks
			mockDB.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}

func TestTerminologyService_ValidateCode(t *testing.T) {
	tests := []struct {
		name             string
		code             string
		systemURI        string
		version          string
		mockSetup        func(*mocks.MockDB, *mocks.MockRedisClient)
		expectedResult   *models.ValidationResult
		expectedError    string
	}{
		{
			name:      "valid active code",
			code:      "387517004",
			systemURI: "http://snomed.info/sct",
			version:   "",
			mockSetup: func(mockDB *mocks.MockDB, mockCache *mocks.MockRedisClient) {
				// Cache miss
				cacheKey := cache.ValidationCacheKey("387517004", "http://snomed.info/sct", "")
				mockCache.On("Get", cacheKey, mock.AnythingOfType("*models.ValidationResult")).Return(assert.AnError)

				// Database query success
				rows := mocks.NewMockRows()
				rows.On("Scan", 
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*string"), mock.AnythingOfType("*string"),
					mock.AnythingOfType("*string")).Return(nil).Run(func(args mock.Arguments) {
					*args[0].(*string) = "387517004"
					*args[1].(*string) = "Paracetamol"
					*args[2].(*string) = "active"
					*args[3].(*string) = "http://snomed.info/sct"
					*args[4].(*string) = "20250701"
				})

				mockDB.On("QueryRow", mock.AnythingOfType("string"), 
					"387517004", "http://snomed.info/sct").Return(rows)

				// Cache set
				mockCache.On("Set", cacheKey, mock.AnythingOfType("models.ValidationResult"), 
					1*time.Hour).Return(nil)
			},
			expectedResult: &models.ValidationResult{
				Valid:    true,
				Code:     "387517004",
				System:   "http://snomed.info/sct",
				Display:  "Paracetamol",
				Severity: "information",
			},
		},
		{
			name:      "invalid code",
			code:      "INVALID123",
			systemURI: "http://snomed.info/sct",
			version:   "",
			mockSetup: func(mockDB *mocks.MockDB, mockCache *mocks.MockRedisClient) {
				// Cache miss
				cacheKey := cache.ValidationCacheKey("INVALID123", "http://snomed.info/sct", "")
				mockCache.On("Get", cacheKey, mock.AnythingOfType("*models.ValidationResult")).Return(assert.AnError)

				// Database query - no rows
				rows := mocks.NewMockRows()
				rows.On("Scan", mock.Anything, mock.Anything, mock.Anything, 
					mock.Anything, mock.Anything).Return(sql.ErrNoRows)

				mockDB.On("QueryRow", mock.AnythingOfType("string"), 
					"INVALID123", "http://snomed.info/sct").Return(rows)

				// Cache set
				mockCache.On("Set", cacheKey, mock.AnythingOfType("models.ValidationResult"), 
					1*time.Hour).Return(nil)
			},
			expectedResult: &models.ValidationResult{
				Valid:    false,
				Code:     "INVALID123",
				System:   "http://snomed.info/sct",
				Message:  "Code 'INVALID123' not found in system 'http://snomed.info/sct'",
				Severity: "error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := &mocks.MockDB{}
			mockCache := &mocks.MockRedisClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			metricsCollector := metrics.NewCollector("test")

			tt.mockSetup(mockDB, mockCache)

			service := services.NewTerminologyService(mockDB, mockCache, logger, metricsCollector)

			// Execute
			result, err := service.ValidateCode(tt.code, tt.systemURI, tt.version)

			// Assert
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.Valid, result.Valid)
				assert.Equal(t, tt.expectedResult.Code, result.Code)
				assert.Equal(t, tt.expectedResult.System, result.System)
			}

			// Verify mocks
			mockDB.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}

func TestTerminologyService_HealthCheck(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*mocks.MockDB, *mocks.MockRedisClient)
		expectedStatus string
	}{
		{
			name: "all services healthy",
			mockSetup: func(mockDB *mocks.MockDB, mockCache *mocks.MockRedisClient) {
				mockDB.On("Ping").Return(nil)
				mockCache.On("Exists", "health_check").Return(true, nil)
			},
			expectedStatus: "healthy",
		},
		{
			name: "database unhealthy",
			mockSetup: func(mockDB *mocks.MockDB, mockCache *mocks.MockRedisClient) {
				mockDB.On("Ping").Return(assert.AnError)
				mockCache.On("Exists", "health_check").Return(true, nil)
			},
			expectedStatus: "unhealthy",
		},
		{
			name: "cache unhealthy",
			mockSetup: func(mockDB *mocks.MockDB, mockCache *mocks.MockRedisClient) {
				mockDB.On("Ping").Return(nil)
				mockCache.On("Exists", "health_check").Return(false, assert.AnError)
			},
			expectedStatus: "unhealthy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockDB := &mocks.MockDB{}
			mockCache := &mocks.MockRedisClient{}
			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)
			metricsCollector := metrics.NewCollector("test")

			tt.mockSetup(mockDB, mockCache)

			service := services.NewTerminologyService(mockDB, mockCache, logger, metricsCollector)

			// Execute
			result := service.HealthCheck()

			// Assert
			require.NotNil(t, result)
			assert.Equal(t, "kb-7-terminology", result["service"])
			assert.Equal(t, tt.expectedStatus, result["status"])
			assert.Contains(t, result, "checks")

			checks, ok := result["checks"].(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, checks, "database")
			assert.Contains(t, checks, "cache")

			// Verify mocks
			mockDB.AssertExpectations(t)
			mockCache.AssertExpectations(t)
		})
	}
}