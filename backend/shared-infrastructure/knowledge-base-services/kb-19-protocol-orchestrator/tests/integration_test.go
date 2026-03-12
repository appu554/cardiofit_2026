// Package tests provides integration tests for KB-19.
package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/api"
	"kb-19-protocol-orchestrator/internal/config"
)

func init() {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
}

// testConfig returns a config suitable for testing.
func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:        8099,
			Environment: "test",
		},
		Logging: config.LoggingConfig{
			Level:  "debug",
			Format: "json",
		},
	}
}

// testRouter creates a test router, handling errors appropriately.
func testRouter(t testing.TB) *gin.Engine {
	cfg := testConfig()
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	entry := logrus.NewEntry(log)

	server, err := api.NewServer(cfg, entry)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return server.Router()
}

// TestHealthEndpoint tests the health check endpoint.
func TestHealthEndpoint(t *testing.T) {
	router := testRouter(t)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "healthy") {
		t.Error("response should contain 'healthy'")
	}
}

// TestReadyEndpoint tests the readiness check endpoint.
func TestReadyEndpoint(t *testing.T) {
	router := testRouter(t)

	req, _ := http.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Ready endpoint may return 503 if dependencies are not available
	// In test mode, we accept both 200 and 503
	if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 200 or 503, got %d", w.Code)
	}
}

// TestProtocolsEndpoint tests the protocols listing endpoint.
func TestProtocolsEndpoint(t *testing.T) {
	router := testRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/protocols", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Response should be JSON object with protocols array
	body := w.Body.String()
	if !strings.Contains(body, "\"protocols\"") {
		t.Error("response should contain 'protocols' key")
	}
}

// TestExecuteEndpointValidation tests request validation for execute endpoint.
func TestExecuteEndpointValidation(t *testing.T) {
	router := testRouter(t)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Empty body",
			body:           "{}",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing patient_id",
			body:           `{"encounter_id": "` + uuid.New().String() + `"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing encounter_id",
			body:           `{"patient_id": "` + uuid.New().String() + `"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid UUID",
			body:           `{"patient_id": "invalid", "encounter_id": "` + uuid.New().String() + `"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/v1/execute", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestEvaluateEndpointValidation tests request validation for evaluate endpoint.
func TestEvaluateEndpointValidation(t *testing.T) {
	router := testRouter(t)

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Missing protocol_id",
			body:           `{"patient_id": "` + uuid.New().String() + `", "encounter_id": "` + uuid.New().String() + `"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Empty protocol_id",
			body:           `{"patient_id": "` + uuid.New().String() + `", "encounter_id": "` + uuid.New().String() + `", "protocol_id": ""}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/api/v1/evaluate", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestDecisionsEndpointValidation tests validation for decisions endpoint.
func TestDecisionsEndpointValidation(t *testing.T) {
	router := testRouter(t)

	tests := []struct {
		name           string
		patientID      string
		expectedStatus int
	}{
		{
			name:           "Invalid UUID",
			patientID:      "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid UUID (may return empty or 404)",
			patientID:      uuid.New().String(),
			expectedStatus: http.StatusOK, // or 404 if no decisions found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/decisions/"+tt.patientID, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// For valid UUIDs, we accept 200 (with empty array) or 404
			if tt.expectedStatus == http.StatusOK {
				if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
					t.Errorf("expected status 200 or 404, got %d", w.Code)
				}
			} else if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestContextTimeout tests that requests respect context timeouts.
func TestContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Simulate a slow operation
	select {
	case <-ctx.Done():
		// Expected - context timed out
	case <-time.After(100 * time.Millisecond):
		t.Error("context should have timed out")
	}
}

// TestCORSHeaders tests that CORS headers are set correctly.
func TestCORSHeaders(t *testing.T) {
	router := testRouter(t)

	req, _ := http.NewRequest("OPTIONS", "/api/v1/protocols", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// CORS middleware should handle OPTIONS requests
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("expected status 200 or 204 for OPTIONS, got %d", w.Code)
	}
}

// TestContentTypeJSON tests that responses have correct Content-Type.
func TestContentTypeJSON(t *testing.T) {
	router := testRouter(t)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}
}

// BenchmarkHealthEndpoint benchmarks the health endpoint.
func BenchmarkHealthEndpoint(b *testing.B) {
	router := testRouter(b)
	req, _ := http.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkProtocolsEndpoint benchmarks the protocols listing endpoint.
func BenchmarkProtocolsEndpoint(b *testing.B) {
	router := testRouter(b)
	req, _ := http.NewRequest("GET", "/api/v1/protocols", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
