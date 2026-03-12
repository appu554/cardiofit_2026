package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
	"kb-22-hpi-engine/internal/services"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func init() {
	gin.SetMode(gin.TestMode)
}

// testServer creates a minimal Server with a NodeLoader for handler tests.
// No real DB/cache — only tests that don't hit those paths.
func testServer() *Server {
	log := zap.NewNop()
	loader := services.NewNodeLoaderFromMap(map[string]*models.NodeDefinition{
		"P1_CHEST_PAIN": {
			NodeID:  "P1_CHEST_PAIN",
			Version: "1.0.0",
			Questions: []models.QuestionDef{
				{ID: "Q001", TextEN: "Do you have chest pain?"},
			},
			Differentials: []models.DifferentialDef{
				{ID: "ACS", LabelEN: "Acute Coronary Syndrome", Priors: map[string]float64{"GENERAL": 0.30}},
			},
		},
	})

	router := gin.New()
	s := &Server{
		Router:     router,
		Log:        log,
		NodeLoader: loader,
	}
	s.registerHandlerRoutes()
	return s
}

// registerHandlerRoutes registers only the handler routes needed for testing
// (skipping health/readiness which need DB/Cache).
func (s *Server) registerHandlerRoutes() {
	v1 := s.Router.Group("/api/v1")
	{
		v1.POST("/sessions", s.createSessionHandler)
		v1.GET("/sessions/:id", s.getSessionHandler)
		v1.POST("/sessions/:id/answers", s.submitAnswerHandler)
		v1.POST("/sessions/:id/suspend", s.suspendSessionHandler)
		v1.POST("/sessions/:id/resume", s.resumeSessionHandler)
		v1.POST("/sessions/:id/complete", s.completeSessionHandler)
		v1.GET("/sessions/:id/differential", s.getDifferentialHandler)
		v1.GET("/sessions/:id/safety", s.getSafetyFlagsHandler)
		v1.GET("/snapshots/:session_id", s.getSnapshotHandler)
		v1.GET("/nodes", s.listNodesHandler)
		v1.GET("/nodes/:node_id", s.getNodeHandler)
	}
}

// ---------------------------------------------------------------------------
// Node endpoint tests (no DB/cache needed)
// ---------------------------------------------------------------------------

func TestListNodesHandler(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	count, ok := body["count"].(float64)
	if !ok || int(count) != 1 {
		t.Errorf("count = %v, want 1", body["count"])
	}
}

func TestGetNodeHandler_Found(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/P1_CHEST_PAIN", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var node models.NodeDefinition
	if err := json.Unmarshal(w.Body.Bytes(), &node); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if node.NodeID != "P1_CHEST_PAIN" {
		t.Errorf("node_id = %s, want P1_CHEST_PAIN", node.NodeID)
	}
}

func TestGetNodeHandler_NotFound(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes/P99_NONEXISTENT", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// ---------------------------------------------------------------------------
// Session creation validation tests
// ---------------------------------------------------------------------------

func TestCreateSessionHandler_InvalidBody(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateSessionHandler_UnknownNode(t *testing.T) {
	s := testServer()
	body := models.CreateSessionRequest{
		PatientID: uuid.New(),
		NodeID:    "P99_NONEXISTENT",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

// ---------------------------------------------------------------------------
// Session ID validation tests
// ---------------------------------------------------------------------------

func TestGetSessionHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/not-a-uuid", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSubmitAnswerHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/bad-id/answers",
		bytes.NewBufferString(`{"question_id":"Q001","answer_value":"YES"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSubmitAnswerHandler_InvalidBody(t *testing.T) {
	s := testServer()
	id := uuid.New().String()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+id+"/answers",
		bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSubmitAnswerHandler_InvalidAnswerValue(t *testing.T) {
	s := testServer()
	id := uuid.New().String()
	body := `{"question_id":"Q001","answer_value":"MAYBE"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/"+id+"/answers",
		bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnprocessableEntity)
	}
}

func TestSuspendSessionHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/bad/suspend", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestResumeSessionHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/bad/resume", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCompleteSessionHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sessions/bad/complete", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetDifferentialHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/bad/differential", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetSafetyFlagsHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/bad/safety", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestGetSnapshotHandler_InvalidUUID(t *testing.T) {
	s := testServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/snapshots/bad", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"invalid state transition", "invalid state", true},
		{"Session Is Not Active", "session is not active", true},
		{"session already completed", "ALREADY COMPLETED", true},
		{"no match here", "invalid state", false},
		{"short", "longer than input", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := containsIgnoreCase(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"invalid state", fmt.Errorf("invalid state: cannot answer"), true},
		{"session not active", fmt.Errorf("session is not active"), true},
		{"already completed", fmt.Errorf("session already completed"), true},
		{"cannot suspend", fmt.Errorf("cannot suspend session in current state"), true},
		{"cannot resume", fmt.Errorf("cannot resume: not suspended"), true},
		{"already adjudicated", fmt.Errorf("already adjudicated for this snapshot"), true},
		{"generic error", fmt.Errorf("database connection lost"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConflictError(tt.err)
			if got != tt.want {
				t.Errorf("isConflictError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
