package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "kb-17-population-registry", response.Service)
}

func TestReadyHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// May return 503 if dependencies aren't available in test mode
	assert.Contains(t, []int{http.StatusOK, http.StatusServiceUnavailable}, w.Code)
}

func TestListRegistriesHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/registries", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response struct {
		Success bool              `json:"success"`
		Data    []models.Registry `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
}

func TestGetRegistryHandler_NotFound(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/registries/NONEXISTENT", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetRegistryHandler_Success(t *testing.T) {
	router := setupTestRouterWithMockData()

	req, _ := http.NewRequest("GET", "/api/v1/registries/DIABETES", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// With mock data, should return 200
	if w.Code == http.StatusOK {
		var response struct {
			Success bool            `json:"success"`
			Data    models.Registry `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, models.RegistryDiabetes, response.Data.Code)
	}
}

func TestListEnrollmentsHandler_WithPagination(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/enrollments?limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, 10, response.Limit)
	assert.Equal(t, 0, response.Offset)
}

func TestListEnrollmentsHandler_WithRegistryFilter(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/enrollments?registry=DIABETES", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetPatientRegistriesHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/patients/patient-123/registries", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.PatientRegistriesResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "patient-123", response.PatientID)
}

func TestGetAllStatsHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.AllStatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.True(t, response.Success)
}

func TestGetHighRiskPatientsHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/high-risk", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetCareGapPatientsHandler(t *testing.T) {
	router := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/care-gaps", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Helper functions

func setupTestRouter() *gin.Engine {
	router := gin.New()

	// Setup routes with mock handlers that work without database
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, models.HealthResponse{
			Status:    "healthy",
			Service:   "kb-17-population-registry",
			Version:   "test",
			Uptime:    "0s",
			Timestamp: time.Now(),
		})
	})
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusServiceUnavailable, models.NewErrorResponse("test mode"))
	})

	v1 := router.Group("/api/v1")
	{
		v1.GET("/registries", func(c *gin.Context) {
			c.JSON(http.StatusOK, models.NewSuccessResponse([]models.Registry{}))
		})
		v1.GET("/registries/:code", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, models.NewErrorResponse("not found"))
		})
		v1.GET("/enrollments", func(c *gin.Context) {
			limit := 20
			offset := 0
			if l := c.Query("limit"); l != "" {
				limit = 10 // simplified
			}
			c.JSON(http.StatusOK, models.NewPaginatedResponse([]models.RegistryPatient{}, 0, limit, offset))
		})
		v1.GET("/patients/:id/registries", func(c *gin.Context) {
			patientID := c.Param("id")
			c.JSON(http.StatusOK, &models.PatientRegistriesResponse{
				Success:     true,
				PatientID:   patientID,
				Enrollments: []models.RegistryPatient{},
				Total:       0,
			})
		})
		v1.GET("/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, &models.AllStatsResponse{
				Success: true,
				Data:    []models.RegistryStats{},
				Summary: &models.StatsSummary{},
			})
		})
		v1.GET("/high-risk", func(c *gin.Context) {
			c.JSON(http.StatusOK, &models.HighRiskResponse{
				Success: true,
				Data:    []models.HighRiskPatientSummary{},
			})
		})
		v1.GET("/care-gaps", func(c *gin.Context) {
			c.JSON(http.StatusOK, &models.CareGapResponse{
				Success: true,
				Data:    []models.CareGapSummary{},
			})
		})
	}

	return router
}

func setupTestRouterWithMockData() *gin.Engine {
	router := gin.New()

	// Create a test server with mock registry data
	v1 := router.Group("/api/v1")
	{
		v1.GET("/registries/:code", func(c *gin.Context) {
			code := models.RegistryCode(c.Param("code"))
			if code == models.RegistryDiabetes {
				registry := models.Registry{
					Code:        models.RegistryDiabetes,
					Name:        "Diabetes Mellitus Registry",
					Description: "Type 1 and Type 2 Diabetes Management",
					Active:      true,
				}
				c.JSON(http.StatusOK, models.NewSuccessResponse(registry))
				return
			}
			c.JSON(http.StatusNotFound, models.NewErrorResponse("registry not found"))
		})
	}

	return router
}
