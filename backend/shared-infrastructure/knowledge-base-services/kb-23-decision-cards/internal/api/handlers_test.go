package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
)

// testServer creates a minimal Server suitable for unit tests that exercise
// request parsing, validation, and CORS without requiring DB, Redis, or
// downstream service dependencies.
func testServer() *Server {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Environment: "test"}
	log := zap.NewNop()

	router := gin.New()
	router.Use(gin.Recovery())

	return &Server{
		Router: router,
		cfg:    cfg,
		log:    log,
	}
}

// testServerWithCORS creates a server with the CORS middleware applied so
// that CORS behaviour can be verified in isolation.
func testServerWithCORS() *Server {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Environment: "test"}
	log := zap.NewNop()

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	return &Server{
		Router: router,
		cfg:    cfg,
		log:    log,
	}
}

// ---------------------------------------------------------------------------
// 1. Health endpoint
// ---------------------------------------------------------------------------

func TestHandleHealth(t *testing.T) {
	s := testServer()
	s.Router.GET("/health", s.handleHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}

	if body["status"] != "healthy" {
		t.Errorf("expected status=healthy, got %v", body["status"])
	}
	if body["service"] != "kb-23-decision-cards" {
		t.Errorf("expected service=kb-23-decision-cards, got %v", body["service"])
	}
}

// ---------------------------------------------------------------------------
// 2. Readiness with nil dependencies -- nil pointer dereference is the risk
// ---------------------------------------------------------------------------

func TestHandleReadiness_ServiceUnavailable(t *testing.T) {
	s := testServer()
	// db and cache are nil -- handleReadiness calls s.db.Health() which will
	// panic on a nil receiver. The gin Recovery middleware should catch this
	// and return 500 rather than crashing the process.
	s.Router.GET("/readiness", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/readiness", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	// Recovery middleware converts panics into 500.
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 from nil-dep panic recovery, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// 3. Generate card -- invalid JSON payload
// ---------------------------------------------------------------------------

func TestHandleGenerateCard_InvalidPayload(t *testing.T) {
	s := testServer()
	s.Router.POST("/api/v1/decision-cards", s.handleGenerateCard)

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{not-json`},
		{"empty object missing required fields", `{}`},
		{"missing patient_id", `{"session_id":"` + uuid.New().String() + `"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/decision-cards",
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d; body: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if resp["error"] != "invalid_payload" {
				t.Errorf("expected error=invalid_payload, got %v", resp["error"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 4. Get card -- invalid UUID path parameter
// ---------------------------------------------------------------------------

func TestHandleGetCard_InvalidUUID(t *testing.T) {
	s := testServer()
	s.Router.GET("/api/v1/cards/:id", s.handleGetCard)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/cards/not-a-uuid", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if resp["error"] != "invalid_card_id" {
		t.Errorf("expected error=invalid_card_id, got %v", resp["error"])
	}
}

// ---------------------------------------------------------------------------
// 5. MCU gate resume -- invalid UUID path parameter
// ---------------------------------------------------------------------------

func TestHandleMCUGateResume_InvalidUUID(t *testing.T) {
	s := testServer()
	s.Router.POST("/api/v1/cards/:id/mcu-gate-resume", s.handleMCUGateResume)

	body := `{"clinician_id":"dr-smith","reason":"patient stable"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cards/not-a-uuid/mcu-gate-resume",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if resp["error"] != "invalid_card_id" {
		t.Errorf("expected error=invalid_card_id, got %v", resp["error"])
	}
}

// ---------------------------------------------------------------------------
// 6. MCU gate resume -- missing body (clinician_id is required)
// ---------------------------------------------------------------------------

func TestHandleMCUGateResume_MissingBody(t *testing.T) {
	s := testServer()
	s.Router.POST("/api/v1/cards/:id/mcu-gate-resume", s.handleMCUGateResume)

	validUUID := uuid.New().String()
	tests := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"empty JSON object", "{}"},
		{"missing clinician_id", `{"reason":"stable"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost,
				"/api/v1/cards/"+validUUID+"/mcu-gate-resume",
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d; body: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if resp["error"] != "invalid_payload" {
				t.Errorf("expected error=invalid_payload, got %v", resp["error"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 7. Hypoglycaemia alert -- invalid payload
// ---------------------------------------------------------------------------

func TestHandleHypoglycaemiaAlert_InvalidPayload(t *testing.T) {
	s := testServer()
	s.Router.POST("/api/v1/safety/hypoglycaemia-alert", s.handleHypoglycaemiaAlert)

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{bad`},
		{"empty object missing required fields", `{}`},
		{"missing source", `{"patient_id":"` + uuid.New().String() + `"}`},
		{"missing patient_id", `{"source":"CGM"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/safety/hypoglycaemia-alert",
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d; body: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if resp["error"] != "invalid_payload" {
				t.Errorf("expected error=invalid_payload, got %v", resp["error"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 8. Behavioral gap alert -- invalid payload
// ---------------------------------------------------------------------------

func TestHandleBehavioralGapAlert_InvalidPayload(t *testing.T) {
	s := testServer()
	s.Router.POST("/api/v1/safety/behavioral-gap-alert", s.handleBehavioralGapAlert)

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{bad`},
		{"empty object missing required fields", `{}`},
		{"missing source", `{"patient_id":"` + uuid.New().String() + `"}`},
		{"missing patient_id", `{"source":"KB21_BEHAVIORAL"}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/safety/behavioral-gap-alert",
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d; body: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if resp["error"] != "invalid_payload" {
				t.Errorf("expected error=invalid_payload, got %v", resp["error"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 9. Create perturbation -- invalid payload
// ---------------------------------------------------------------------------

func TestHandleCreatePerturbation_InvalidPayload(t *testing.T) {
	s := testServer()
	s.Router.POST("/api/v1/perturbations", s.handleCreatePerturbation)

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{bad`},
		{"wrong type for dose_delta", `{"dose_delta":"not-a-number"}`},
		{"array instead of object", `[1,2,3]`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/perturbations",
				bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			s.Router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d; body: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse error response: %v", err)
			}
			if resp["error"] != "invalid_payload" {
				t.Errorf("expected error=invalid_payload, got %v", resp["error"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 10. CORS middleware
// ---------------------------------------------------------------------------

func TestCORSMiddleware(t *testing.T) {
	s := testServerWithCORS()
	// Register a simple endpoint to test non-OPTIONS requests.
	s.Router.GET("/health", s.handleHealth)

	t.Run("OPTIONS preflight returns 204 with CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/health", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204 for OPTIONS, got %d", w.Code)
		}

		assertHeader(t, w, "Access-Control-Allow-Origin", "*")
		assertHeader(t, w, "Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		assertHeader(t, w, "Access-Control-Allow-Headers", "Content-Type, Authorization")
	})

	t.Run("GET request includes CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		assertHeader(t, w, "Access-Control-Allow-Origin", "*")
	})

	t.Run("OPTIONS on unregistered route still returns 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/anything", nil)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, req)

		// CORS middleware aborts with 204 before routing, so even an
		// unregistered path returns 204 for OPTIONS.
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204 for OPTIONS on unregistered route, got %d", w.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertHeader(t *testing.T, w *httptest.ResponseRecorder, key, expected string) {
	t.Helper()
	got := w.Header().Get(key)
	if got != expected {
		t.Errorf("header %q: expected %q, got %q", key, expected, got)
	}
}
