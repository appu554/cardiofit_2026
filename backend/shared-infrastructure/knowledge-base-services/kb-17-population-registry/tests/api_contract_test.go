// Package tests provides comprehensive test utilities for KB-17 Population Registry
// api_contract_test.go - Tests for API endpoint contracts and response formats
// This validates the API surface that external systems depend on
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

// =============================================================================
// API CONTRACT TEST SETUP
// =============================================================================

func setupTestRouter() (*gin.Engine, *MockRepository, *MockEventProducer) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	repo := NewMockRepository()
	producer := NewMockEventProducer()

	// Register test routes that match production API contracts
	api := router.Group("/api/v1")
	{
		// Registries
		api.GET("/registries", createListRegistriesHandler(repo))
		api.GET("/registries/:code", createGetRegistryHandler(repo))
		api.GET("/registries/:code/patients", createListRegistryPatientsHandler(repo))

		// Enrollments
		api.GET("/enrollments", createListEnrollmentsHandler(repo))
		api.POST("/enrollments", createCreateEnrollmentHandler(repo, producer))
		api.GET("/enrollments/:id", createGetEnrollmentHandler(repo))
		api.PUT("/enrollments/:id", createUpdateEnrollmentHandler(repo, producer))
		api.DELETE("/enrollments/:id", createDeleteEnrollmentHandler(repo, producer))
		api.POST("/enrollments/bulk", createBulkEnrollmentHandler(repo, producer))

		// Patient lookups
		api.GET("/patients/:id/registries", createGetPatientRegistriesHandler(repo))
		api.GET("/patients/:id/enrollment/:code", createGetPatientEnrollmentHandler(repo))

		// Operations
		api.POST("/evaluate", createEvaluateHandler(repo))
		api.GET("/stats", createGetStatsHandler(repo))
		api.GET("/stats/:code", createGetRegistryStatsHandler(repo))
		api.GET("/high-risk", createGetHighRiskHandler(repo))
		api.GET("/care-gaps", createGetCareGapsHandler(repo))
	}

	// Health endpoints
	router.GET("/health", createHealthHandler(repo))
	router.GET("/ready", createReadyHandler(repo))

	return router, repo, producer
}

// =============================================================================
// REGISTRY ENDPOINT TESTS
// =============================================================================

// TestAPIContract_ListRegistries tests GET /api/v1/registries
func TestAPIContract_ListRegistries(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/registries", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result struct {
		Data  []models.Registry `json:"data"`
		Count int               `json:"count"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	// Contract: Should return all 8 pre-configured registries
	assert.Equal(t, 8, result.Count)
	assert.Len(t, result.Data, 8)

	// Contract: Each registry must have required fields
	for _, reg := range result.Data {
		assert.NotEmpty(t, reg.Code, "Registry must have code")
		assert.NotEmpty(t, reg.Name, "Registry must have name")
		assert.NotEmpty(t, reg.Description, "Registry must have description")
		assert.NotEmpty(t, reg.Category, "Registry must have category")
	}
}

// TestAPIContract_GetRegistry tests GET /api/v1/registries/:code
func TestAPIContract_GetRegistry(t *testing.T) {
	router, _, _ := setupTestRouter()

	testCases := []struct {
		code           string
		expectedStatus int
	}{
		{"DIABETES", http.StatusOK},
		{"HYPERTENSION", http.StatusOK},
		{"HEART_FAILURE", http.StatusOK},
		{"INVALID_CODE", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.code, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/registries/"+tc.code, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)

			if tc.expectedStatus == http.StatusOK {
				var result struct {
					Data models.Registry `json:"data"`
				}
				err := json.Unmarshal(resp.Body.Bytes(), &result)
				require.NoError(t, err)
				assert.Equal(t, models.RegistryCode(tc.code), result.Data.Code)
			}
		})
	}
}

// =============================================================================
// ENROLLMENT ENDPOINT TESTS
// =============================================================================

// TestAPIContract_CreateEnrollment tests POST /api/v1/enrollments
func TestAPIContract_CreateEnrollment(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Valid enrollment request
	enrollmentReq := map[string]interface{}{
		"patient_id":    "patient-api-001",
		"registry_code": "DIABETES",
		"risk_tier":     "MODERATE",
		"source":        "MANUAL",
		"enrolled_by":   "dr.smith",
	}

	body, _ := json.Marshal(enrollmentReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusCreated, resp.Code)

	var result struct {
		Data models.RegistryPatient `json:"data"`
	}
	err := json.Unmarshal(resp.Body.Bytes(), &result)
	require.NoError(t, err)

	// Contract: Response must include these fields
	assert.NotEqual(t, uuid.Nil, result.Data.ID, "Must return enrollment ID")
	assert.Equal(t, "patient-api-001", result.Data.PatientID)
	assert.Equal(t, models.RegistryDiabetes, result.Data.RegistryCode)
	assert.Equal(t, models.EnrollmentStatusActive, result.Data.Status)
	assert.False(t, result.Data.EnrolledAt.IsZero(), "Must have enrolled_at timestamp")
}

// TestAPIContract_CreateEnrollment_ValidationErrors tests validation
func TestAPIContract_CreateEnrollment_ValidationErrors(t *testing.T) {
	router, _, _ := setupTestRouter()

	testCases := []struct {
		name           string
		request        map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Missing_PatientID",
			request: map[string]interface{}{
				"registry_code": "DIABETES",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "patient_id is required",
		},
		{
			name: "Missing_RegistryCode",
			request: map[string]interface{}{
				"patient_id": "patient-001",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "registry_code is required",
		},
		{
			name: "Invalid_RegistryCode",
			request: map[string]interface{}{
				"patient_id":    "patient-001",
				"registry_code": "INVALID",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid registry code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedStatus, resp.Code)

			var result struct {
				Error string `json:"error"`
			}
			_ = json.Unmarshal(resp.Body.Bytes(), &result)
			assert.Contains(t, result.Error, tc.expectedError)
		})
	}
}

// TestAPIContract_GetEnrollment tests GET /api/v1/enrollments/:id
func TestAPIContract_GetEnrollment(t *testing.T) {
	router, repo, _ := setupTestRouter()

	// Setup: Create enrollment
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "patient-get-001",
		RegistryCode: models.RegistryHypertension,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierHigh,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	// Test valid request
	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments/"+enrollment.ID.String(), nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result struct {
		Data models.RegistryPatient `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	assert.Equal(t, enrollment.ID, result.Data.ID)

	// Test not found
	req = httptest.NewRequest(http.MethodGet, "/api/v1/enrollments/"+uuid.New().String(), nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

// TestAPIContract_UpdateEnrollment tests PUT /api/v1/enrollments/:id
func TestAPIContract_UpdateEnrollment(t *testing.T) {
	router, repo, _ := setupTestRouter()

	// Setup
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "patient-update-001",
		RegistryCode: models.RegistryCKD,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierModerate,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	// Update request
	updateReq := map[string]interface{}{
		"status":     "SUSPENDED",
		"risk_tier":  "HIGH",
		"reason":     "Patient hospitalized",
		"updated_by": "system",
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/enrollments/"+enrollment.ID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result struct {
		Data models.RegistryPatient `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	assert.Equal(t, models.EnrollmentStatusSuspended, result.Data.Status)
}

// TestAPIContract_DeleteEnrollment tests DELETE /api/v1/enrollments/:id
func TestAPIContract_DeleteEnrollment(t *testing.T) {
	router, repo, _ := setupTestRouter()

	// Setup
	enrollment := &models.RegistryPatient{
		ID:           uuid.New(),
		PatientID:    "patient-delete-001",
		RegistryCode: models.RegistryCOPD,
		Status:       models.EnrollmentStatusActive,
		RiskTier:     models.RiskTierLow,
		EnrolledAt:   time.Now(),
	}
	_ = repo.CreateEnrollment(enrollment)

	// Delete request
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/enrollments/"+enrollment.ID.String()+"?reason=Patient%20request&actor=patient", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	// Verify disenrolled
	updated, _ := repo.GetEnrollment(enrollment.ID)
	assert.Equal(t, models.EnrollmentStatusDisenrolled, updated.Status)
}

// =============================================================================
// BULK ENROLLMENT TESTS
// =============================================================================

// TestAPIContract_BulkEnrollment tests POST /api/v1/enrollments/bulk
func TestAPIContract_BulkEnrollment(t *testing.T) {
	router, _, _ := setupTestRouter()

	bulkReq := map[string]interface{}{
		"registry_code": "DIABETES",
		"patient_ids":   []string{"bulk-001", "bulk-002", "bulk-003"},
		"source":        "BULK",
		"enrolled_by":   "admin",
	}

	body, _ := json.Marshal(bulkReq)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/enrollments/bulk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result struct {
		Data models.BulkEnrollmentResult `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)

	// Contract: Bulk result must include these fields
	assert.True(t, len(result.Data.Enrolled)+result.Data.Failed+len(result.Data.Skipped) == 3,
		"All patients must be accounted for")
}

// =============================================================================
// PATIENT LOOKUP TESTS
// =============================================================================

// TestAPIContract_GetPatientRegistries tests GET /api/v1/patients/:id/registries
func TestAPIContract_GetPatientRegistries(t *testing.T) {
	router, repo, _ := setupTestRouter()

	patientID := "patient-registries-001"

	// Enroll in multiple registries
	for _, code := range []models.RegistryCode{models.RegistryDiabetes, models.RegistryHypertension} {
		enrollment := &models.RegistryPatient{
			ID:           uuid.New(),
			PatientID:    patientID,
			RegistryCode: code,
			Status:       models.EnrollmentStatusActive,
			RiskTier:     models.RiskTierModerate,
			EnrolledAt:   time.Now(),
		}
		_ = repo.CreateEnrollment(enrollment)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/patients/"+patientID+"/registries", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result struct {
		Data []models.RegistryPatient `json:"data"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	assert.Len(t, result.Data, 2)
}

// =============================================================================
// HEALTH ENDPOINT TESTS
// =============================================================================

// TestAPIContract_HealthEndpoint tests GET /health
func TestAPIContract_HealthEndpoint(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result struct {
		Status   string `json:"status"`
		Database string `json:"database"`
		Redis    string `json:"redis"`
		Kafka    string `json:"kafka"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)

	// Contract: Health response must include these fields
	assert.Equal(t, "healthy", result.Status)
	assert.NotEmpty(t, result.Database)
}

// TestAPIContract_ReadyEndpoint tests GET /ready
func TestAPIContract_ReadyEndpoint(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var result struct {
		Ready bool `json:"ready"`
	}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)
	assert.True(t, result.Ready)
}

// =============================================================================
// RESPONSE FORMAT TESTS
// =============================================================================

// TestAPIContract_ResponseFormat_Success tests success response format
func TestAPIContract_ResponseFormat_Success(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/registries", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Contract: Success responses must have "data" field
	var result map[string]interface{}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)

	assert.Contains(t, result, "data", "Success response must have 'data' field")
	assert.NotContains(t, result, "error", "Success response must not have 'error' field")
}

// TestAPIContract_ResponseFormat_Error tests error response format
func TestAPIContract_ResponseFormat_Error(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/enrollments/invalid-uuid", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Contract: Error responses must have "error" field
	var result map[string]interface{}
	_ = json.Unmarshal(resp.Body.Bytes(), &result)

	assert.Contains(t, result, "error", "Error response must have 'error' field")
}

// TestAPIContract_ContentType tests response content type
func TestAPIContract_ContentType(t *testing.T) {
	router, _, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/registries", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	contentType := resp.Header().Get("Content-Type")
	assert.Contains(t, contentType, "application/json")
}

// =============================================================================
// HANDLER IMPLEMENTATIONS (Mock for testing)
// =============================================================================

func createListRegistriesHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		registries, _ := repo.ListRegistries(true)
		c.JSON(http.StatusOK, gin.H{"data": registries, "count": len(registries)})
	}
}

func createGetRegistryHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := models.RegistryCode(c.Param("code"))
		reg, _ := repo.GetRegistry(code)
		if reg == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "registry not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": reg})
	}
}

func createListRegistryPatientsHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := models.RegistryCode(c.Param("code"))
		enrollments, count, _ := repo.ListEnrollments(&models.EnrollmentQuery{RegistryCode: code})
		c.JSON(http.StatusOK, gin.H{"data": enrollments, "count": count})
	}
}

func createListEnrollmentsHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		enrollments, count, _ := repo.ListEnrollments(&models.EnrollmentQuery{})
		c.JSON(http.StatusOK, gin.H{"data": enrollments, "count": count})
	}
}

func createCreateEnrollmentHandler(repo *MockRepository, producer *MockEventProducer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			PatientID    string `json:"patient_id"`
			RegistryCode string `json:"registry_code"`
			RiskTier     string `json:"risk_tier"`
			Source       string `json:"source"`
			EnrolledBy   string `json:"enrolled_by"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.PatientID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id is required"})
			return
		}
		if req.RegistryCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "registry_code is required"})
			return
		}

		// Validate registry code
		validCodes := map[string]bool{
			"DIABETES": true, "HYPERTENSION": true, "HEART_FAILURE": true,
			"CKD": true, "COPD": true, "PREGNANCY": true,
			"OPIOID_USE": true, "ANTICOAGULATION": true,
		}
		if !validCodes[req.RegistryCode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registry code"})
			return
		}

		enrollment := &models.RegistryPatient{
			ID:               uuid.New(),
			PatientID:        req.PatientID,
			RegistryCode:     models.RegistryCode(req.RegistryCode),
			Status:           models.EnrollmentStatusActive,
			RiskTier:         models.RiskTier(req.RiskTier),
			EnrollmentSource: models.EnrollmentSource(req.Source),
			EnrolledAt:       time.Now(),
			EnrolledBy:       req.EnrolledBy,
		}
		if enrollment.RiskTier == "" {
			enrollment.RiskTier = models.RiskTierModerate
		}

		if err := repo.CreateEnrollment(enrollment); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"data": enrollment})
	}
}

func createGetEnrollmentHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment ID"})
			return
		}

		enrollment, _ := repo.GetEnrollment(id)
		if enrollment == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "enrollment not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": enrollment})
	}
}

func createUpdateEnrollmentHandler(repo *MockRepository, producer *MockEventProducer) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment ID"})
			return
		}

		var req struct {
			Status    string `json:"status"`
			RiskTier  string `json:"risk_tier"`
			Reason    string `json:"reason"`
			UpdatedBy string `json:"updated_by"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		enrollment, _ := repo.GetEnrollment(id)
		if enrollment == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "enrollment not found"})
			return
		}

		if req.Status != "" {
			_ = repo.UpdateEnrollmentStatus(id, enrollment.Status, models.EnrollmentStatus(req.Status), req.Reason, req.UpdatedBy)
		}
		if req.RiskTier != "" {
			_ = repo.UpdateEnrollmentRiskTier(id, enrollment.RiskTier, models.RiskTier(req.RiskTier), req.UpdatedBy)
		}

		updated, _ := repo.GetEnrollment(id)
		c.JSON(http.StatusOK, gin.H{"data": updated})
	}
}

func createDeleteEnrollmentHandler(repo *MockRepository, producer *MockEventProducer) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid enrollment ID"})
			return
		}

		reason := c.Query("reason")
		actor := c.Query("actor")

		if err := repo.DeleteEnrollment(id, reason, actor); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "enrollment disenrolled"})
	}
}

func createBulkEnrollmentHandler(repo *MockRepository, producer *MockEventProducer) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			RegistryCode string   `json:"registry_code"`
			PatientIDs   []string `json:"patient_ids"`
			Source       string   `json:"source"`
			EnrolledBy   string   `json:"enrolled_by"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result := &models.BulkEnrollmentResult{
			Enrolled: make([]string, 0),
			Skipped:  make([]string, 0),
			Errors:   make([]string, 0),
		}
		for _, pid := range req.PatientIDs {
			enrollment := &models.RegistryPatient{
				ID:               uuid.New(),
				PatientID:        pid,
				RegistryCode:     models.RegistryCode(req.RegistryCode),
				Status:           models.EnrollmentStatusActive,
				RiskTier:         models.RiskTierModerate,
				EnrollmentSource: models.EnrollmentSourceBulk,
				EnrolledAt:       time.Now(),
			}
			if err := repo.CreateEnrollment(enrollment); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, pid+": "+err.Error())
			} else {
				result.Enrolled = append(result.Enrolled, pid)
				result.Success++
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": result})
	}
}

func createGetPatientRegistriesHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		patientID := c.Param("id")
		enrollments, _, _ := repo.ListEnrollments(&models.EnrollmentQuery{PatientID: patientID})
		c.JSON(http.StatusOK, gin.H{"data": enrollments})
	}
}

func createGetPatientEnrollmentHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		patientID := c.Param("id")
		code := models.RegistryCode(c.Param("code"))
		enrollment, _ := repo.GetEnrollmentByPatientRegistry(patientID, code)
		if enrollment == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "enrollment not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": enrollment})
	}
}

func createEvaluateHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"evaluated": true}})
	}
}

func createGetStatsHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"total_enrollments": repo.GetEnrollmentCount()}})
	}
}

func createGetRegistryStatsHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := models.RegistryCode(c.Param("code"))
		enrollments, count, _ := repo.ListEnrollments(&models.EnrollmentQuery{RegistryCode: code})
		_ = enrollments
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"registry": code, "count": count}})
	}
}

func createGetHighRiskHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		enrollments, _, _ := repo.ListEnrollments(&models.EnrollmentQuery{RiskTier: models.RiskTierCritical})
		c.JSON(http.StatusOK, gin.H{"data": enrollments})
	}
}

func createGetCareGapsHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
	}
}

func createHealthHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := "healthy"
		if repo.Health() != nil {
			status = "unhealthy"
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   status,
			"database": "connected",
			"redis":    "connected",
			"kafka":    "connected",
		})
	}
}

func createReadyHandler(repo *MockRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	}
}
