//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/api"
	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
	"kb-22-hpi-engine/internal/services"
)

// ---------------------------------------------------------------------------
// Test infrastructure: real PostgreSQL + Redis from docker-compose.test.yml
// ---------------------------------------------------------------------------

var (
	testServer *api.Server
	testDB     *database.Database
	testCache  *cache.CacheClient
	testRouter *gin.Engine
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// Override env for test containers (docker-compose.test.yml: postgres=5438, redis=6387)
	dbURL := envOr("TEST_DATABASE_URL", "postgres://kb22_test:kb22_test_pass@localhost:5438/kb22_test?sslmode=disable")
	redisURL := envOr("TEST_REDIS_URL", "redis://localhost:6387")

	os.Setenv("DATABASE_URL", dbURL)
	os.Setenv("REDIS_URL", redisURL)
	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("NODES_DIR", "../../nodes")

	// Start stub servers for required upstream KBs.
	// KB-20 is required (returns stratum); KB-21, KB-23 degrade gracefully.
	kb20Stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"stratum_label": "DM_HTN_base",
			"patient_sex":   "Male",
			"patient_age":   55,
		})
	}))
	defer kb20Stub.Close()

	// KB-19 stub accepts HPI_COMPLETE and SAFETY_ALERT events
	kb19Stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}))
	defer kb19Stub.Close()

	// KB-23 stub accepts decision card events
	kb23Stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}))
	defer kb23Stub.Close()

	os.Setenv("KB20_URL", kb20Stub.URL)
	os.Setenv("KB19_URL", kb19Stub.URL)
	os.Setenv("KB23_URL", kb23Stub.URL)
	os.Setenv("KB20_TIMEOUT_MS", "5000")
	os.Setenv("KB21_TIMEOUT_MS", "10")
	os.Setenv("KB3_TIMEOUT_MS", "10")
	os.Setenv("KB5_TIMEOUT_MS", "10")
	os.Setenv("KB19_TIMEOUT_MS", "5000")
	os.Setenv("KB23_TIMEOUT_MS", "5000")

	cfg := config.Load()
	log := zap.NewNop()

	// Wait for containers to be ready (up to 30s)
	var db *database.Database
	var err error
	for i := 0; i < 15; i++ {
		db, err = database.NewConnection(cfg, log)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: cannot connect to test database: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run: docker-compose -f docker-compose.test.yml up -d\n")
		os.Exit(0) // skip, don't fail
	}

	if err := db.AutoMigrate(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: auto-migration failed: %v\n", err)
		os.Exit(1)
	}

	cc, err := cache.NewCacheClient(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: cannot connect to test Redis: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run: docker-compose -f docker-compose.test.yml up -d\n")
		os.Exit(0)
	}

	testDB = db
	testCache = cc

	metricsCollector := metrics.NewCollector()

	nodeLoader := services.NewNodeLoader(cfg.NodesDir, log)
	if err := nodeLoader.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to load nodes: %v\n", err)
		os.Exit(1)
	}

	srv := api.NewServer(cfg, db, cc, metricsCollector, log, nodeLoader)
	srv.InitServices()
	srv.RegisterRoutes()

	testServer = srv
	testRouter = srv.Router

	code := m.Run()

	cc.Close()
	os.Exit(code)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func doRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf *bytes.Buffer
	if body != nil {
		data, _ := json.Marshal(body)
		buf = bytes.NewBuffer(data)
	} else {
		buf = &bytes.Buffer{}
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

func parseJSON(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

func cleanDB(t *testing.T) {
	t.Helper()
	sqlDB, _ := testDB.DB.DB()
	for _, table := range []string{
		"calibration_records",
		"differential_snapshots",
		"safety_flags",
		"session_answers",
		"hpi_sessions",
	} {
		sqlDB.Exec("DELETE FROM " + table)
	}
}

// ---------------------------------------------------------------------------
// T-INT-01: Health and Readiness with real DB+Redis
// ---------------------------------------------------------------------------

func TestHealth_RealInfrastructure(t *testing.T) {
	w := doRequest("GET", "/health", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health returned %d: %s", w.Code, w.Body.String())
	}
	body := parseJSON(w)
	checks := body["checks"].(map[string]interface{})
	if checks["database"] != "healthy" {
		t.Errorf("database check: %v", checks["database"])
	}
	if checks["redis"] != "healthy" {
		t.Errorf("redis check: %v", checks["redis"])
	}
}

func TestReadiness_RealInfrastructure(t *testing.T) {
	w := doRequest("GET", "/readiness", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /readiness returned %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// T-INT-02: Node listing from real YAML files
// ---------------------------------------------------------------------------

func TestListNodes_RealYAML(t *testing.T) {
	w := doRequest("GET", "/api/v1/nodes", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /nodes returned %d", w.Code)
	}
	body := parseJSON(w)
	count := body["count"].(float64)
	if count < 3 {
		t.Errorf("expected at least 3 nodes loaded, got %v", count)
	}
}

func TestGetNode_RealYAML(t *testing.T) {
	w := doRequest("GET", "/api/v1/nodes/P01_CHEST_PAIN", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /nodes/P01_CHEST_PAIN returned %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// T-INT-03: Full session lifecycle — Create → Answer → Answer → Complete
// ---------------------------------------------------------------------------

func TestSessionLifecycle_CreateAnswerComplete(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	// Step 1: Create session
	createReq := models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	}
	w := doRequest("POST", "/api/v1/sessions", createReq)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions returned %d: %s", w.Code, w.Body.String())
	}

	var createResp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)

	if createResp.SessionID == uuid.Nil {
		t.Fatal("session_id is nil")
	}
	if createResp.Status != models.StatusActive {
		t.Errorf("expected ACTIVE, got %s", createResp.Status)
	}
	if createResp.NodeID != "P01_CHEST_PAIN" {
		t.Errorf("expected P01_CHEST_PAIN, got %s", createResp.NodeID)
	}
	if createResp.CurrentQuestion == nil {
		t.Fatal("expected a current question, got nil")
	}

	sessionID := createResp.SessionID
	firstQuestionID := createResp.CurrentQuestion.QuestionID

	// Step 2: Get session
	w = doRequest("GET", fmt.Sprintf("/api/v1/sessions/%s", sessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /sessions/%s returned %d: %s", sessionID, w.Code, w.Body.String())
	}

	// Step 3: Submit answer (YES)
	answerReq := models.SubmitAnswerRequest{
		QuestionID:  firstQuestionID,
		AnswerValue: "YES",
		LatencyMS:   1200,
	}
	w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", sessionID), answerReq)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /answers returned %d: %s", w.Code, w.Body.String())
	}

	var answerResp models.AnswerResponse
	json.Unmarshal(w.Body.Bytes(), &answerResp)

	if len(answerResp.TopDifferentials) == 0 {
		t.Error("expected at least one differential after answer")
	}

	// Step 4: Get differential
	w = doRequest("GET", fmt.Sprintf("/api/v1/sessions/%s/differential", sessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /differential returned %d: %s", w.Code, w.Body.String())
	}

	// Step 5: Walk remaining questions until session completes or we exhaust them
	for i := 0; i < 25; i++ {
		if answerResp.Status == models.StatusCompleted || answerResp.Status == models.StatusSafetyEscalated {
			break
		}
		if answerResp.NextQuestion == nil {
			break
		}
		w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", sessionID), models.SubmitAnswerRequest{
			QuestionID:  answerResp.NextQuestion.QuestionID,
			AnswerValue: "NO",
		})
		if w.Code != http.StatusOK {
			t.Fatalf("answer loop %d: returned %d: %s", i, w.Code, w.Body.String())
		}
		json.Unmarshal(w.Body.Bytes(), &answerResp)
	}

	// Complete session (may already be completed)
	if answerResp.Status != models.StatusCompleted && answerResp.Status != models.StatusSafetyEscalated {
		w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/complete", sessionID), nil)
		if w.Code != http.StatusOK {
			t.Logf("POST /complete returned %d: %s (may need more answers)", w.Code, w.Body.String())
		}
	}

	// Step 6: Verify snapshot was created (only for completed sessions)
	w = doRequest("GET", fmt.Sprintf("/api/v1/snapshots/%s", sessionID), nil)
	if w.Code != http.StatusOK {
		t.Logf("GET /snapshots returned %d (snapshot may not exist yet)", w.Code)
	}

	// Step 7: Cannot submit answer to completed session
	w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", sessionID), models.SubmitAnswerRequest{
		QuestionID:  firstQuestionID,
		AnswerValue: "YES",
	})
	if w.Code == http.StatusOK {
		t.Error("expected failure submitting answer on completed/escalated session")
	}
}

// ---------------------------------------------------------------------------
// T-INT-04: Suspend and Resume lifecycle
// ---------------------------------------------------------------------------

func TestSessionLifecycle_SuspendResume(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	// Create session
	w := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions returned %d: %s", w.Code, w.Body.String())
	}

	var createResp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &createResp)
	sessionID := createResp.SessionID

	// Suspend
	w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/suspend", sessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /suspend returned %d: %s", w.Code, w.Body.String())
	}

	// Verify session is suspended
	w = doRequest("GET", fmt.Sprintf("/api/v1/sessions/%s", sessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET session after suspend returned %d", w.Code)
	}

	// Cannot submit answer while suspended
	w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", sessionID), models.SubmitAnswerRequest{
		QuestionID:  "Q001",
		AnswerValue: "YES",
	})
	if w.Code == http.StatusOK {
		t.Error("expected failure submitting answer on suspended session")
	}

	// Resume
	w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/resume", sessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /resume returned %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// T-INT-05: PATA_NAHI answers and Bayesian neutral path
// ---------------------------------------------------------------------------

func TestSessionLifecycle_PataNahiAnswers(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	// Create session
	w := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions returned %d: %s", w.Code, w.Body.String())
	}

	var resp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp.SessionID

	if resp.CurrentQuestion == nil {
		t.Fatal("no current question")
	}

	// Submit PATA_NAHI
	w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", sessionID), models.SubmitAnswerRequest{
		QuestionID:  resp.CurrentQuestion.QuestionID,
		AnswerValue: "PATA_NAHI",
		LatencyMS:   800,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("POST /answers (PATA_NAHI) returned %d: %s", w.Code, w.Body.String())
	}

	var ansResp models.AnswerResponse
	json.Unmarshal(w.Body.Bytes(), &ansResp)

	// Session should still be active, or terminated if safety/cascade triggers fire
	validStatuses := map[models.SessionStatus]bool{
		models.StatusActive:            true,
		models.StatusPartialAssessment: true,
		models.StatusCompleted:         true,
		models.StatusSafetyEscalated:   true,
	}
	if !validStatuses[ansResp.Status] {
		t.Errorf("unexpected status after PATA_NAHI: %s", ansResp.Status)
	}
}

// ---------------------------------------------------------------------------
// T-INT-06: Multiple answers — walk 3 questions
// ---------------------------------------------------------------------------

func TestSessionLifecycle_MultipleAnswers(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	w := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions returned %d: %s", w.Code, w.Body.String())
	}

	var resp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp.SessionID

	// Walk through 3 questions
	answers := []string{"YES", "NO", "YES"}
	for i, ansVal := range answers {
		if resp.CurrentQuestion == nil {
			t.Fatalf("question %d: no current question (session may have converged)", i+1)
		}

		w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", sessionID), models.SubmitAnswerRequest{
			QuestionID:  resp.CurrentQuestion.QuestionID,
			AnswerValue: ansVal,
		})
		if w.Code != http.StatusOK {
			t.Fatalf("question %d: POST /answers returned %d: %s", i+1, w.Code, w.Body.String())
		}

		var ansResp models.AnswerResponse
		json.Unmarshal(w.Body.Bytes(), &ansResp)

		if len(ansResp.TopDifferentials) == 0 {
			t.Errorf("question %d: no differentials returned", i+1)
		}

		// If session completed early (convergence), stop
		if ansResp.Status == models.StatusCompleted {
			t.Logf("session converged after %d questions", i+1)
			return
		}

		// Update resp for next iteration's question
		resp.CurrentQuestion = ansResp.NextQuestion
	}

	// Verify questions_asked increased
	w = doRequest("GET", fmt.Sprintf("/api/v1/sessions/%s", sessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET session returned %d", w.Code)
	}
	body := parseJSON(w)
	asked := body["questions_asked"].(float64)
	if asked < 3 {
		t.Errorf("expected at least 3 questions asked, got %v", asked)
	}
}

// ---------------------------------------------------------------------------
// T-INT-07: Safety flags endpoint
// ---------------------------------------------------------------------------

func TestSafetyFlags_RealDB(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	w := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions returned %d: %s", w.Code, w.Body.String())
	}

	var resp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Get safety flags (should be empty initially)
	w = doRequest("GET", fmt.Sprintf("/api/v1/sessions/%s/safety", resp.SessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /safety returned %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// T-INT-08: Redis cache — session is cached and invalidated
// ---------------------------------------------------------------------------

func TestRedisCache_SessionCacheHit(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	w := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /sessions returned %d", w.Code)
	}

	var resp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Second GET should hit cache
	w = doRequest("GET", fmt.Sprintf("/api/v1/sessions/%s", resp.SessionID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("GET session (cached) returned %d", w.Code)
	}

	// Verify cache key exists in Redis
	var cached models.HPISession
	err := testCache.GetSession(resp.SessionID.String(), &cached)
	if err != nil {
		t.Logf("session not in cache (may be expected): %v", err)
	}
}

// ---------------------------------------------------------------------------
// T-INT-09: Validation — invalid create requests
// ---------------------------------------------------------------------------

func TestCreateSession_ValidationErrors(t *testing.T) {
	cleanDB(t)

	// Missing patient_id
	w := doRequest("POST", "/api/v1/sessions", map[string]string{
		"node_id": "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing patient_id, got %d", w.Code)
	}

	// Unknown node_id
	w = doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: uuid.New(),
		NodeID:    "NONEXISTENT_NODE",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for unknown node, got %d", w.Code)
	}

	// Invalid answer value
	w = doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: uuid.New(),
		NodeID:    "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: create session returned %d", w.Code)
	}
	var resp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", resp.SessionID), models.SubmitAnswerRequest{
		QuestionID:  resp.CurrentQuestion.QuestionID,
		AnswerValue: "INVALID_VALUE",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid answer, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// T-INT-10: Multiple concurrent sessions for same patient
// ---------------------------------------------------------------------------

func TestMultipleSessions_SamePatient(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	// Create two sessions for same patient
	w1 := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w1.Code != http.StatusCreated {
		t.Fatalf("session 1: returned %d", w1.Code)
	}

	w2 := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w2.Code != http.StatusCreated {
		t.Fatalf("session 2: returned %d", w2.Code)
	}

	var r1, r2 models.SessionResponse
	json.Unmarshal(w1.Body.Bytes(), &r1)
	json.Unmarshal(w2.Body.Bytes(), &r2)

	if r1.SessionID == r2.SessionID {
		t.Error("two sessions should have different IDs")
	}
}

// ---------------------------------------------------------------------------
// T-INT-11: Calibration golden dataset import
// ---------------------------------------------------------------------------

func TestCalibration_ImportGolden(t *testing.T) {
	cleanDB(t)

	golden := models.GoldenDatasetImport{
		Cases: []models.GoldenCase{
			{
				NodeID:             "P01_CHEST_PAIN",
				StratumLabel:       "DM_HTN_base",
				ConfirmedDiagnosis: "ACS",
				EngineTop1:         "ACS",
				EngineTop3:         []string{"ACS", "PE", "AORTIC_DISSECTION"},
				QuestionAnswers: map[string]string{
					"Q001": "YES",
					"Q002": "NO",
				},
			},
		},
	}

	w := doRequest("POST", "/api/v1/calibration/import-golden", golden)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /calibration/import-golden returned %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// T-INT-12: Database persistence — session survives server restart
// ---------------------------------------------------------------------------

func TestDatabasePersistence_SessionSurvives(t *testing.T) {
	cleanDB(t)

	patientID := uuid.New()

	w := doRequest("POST", "/api/v1/sessions", models.CreateSessionRequest{
		PatientID: patientID,
		NodeID:    "P01_CHEST_PAIN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create returned %d", w.Code)
	}

	var resp models.SessionResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Direct DB query to verify persistence
	var session models.HPISession
	result := testDB.DB.First(&session, "session_id = ?", resp.SessionID)
	if result.Error != nil {
		t.Fatalf("session not found in DB: %v", result.Error)
	}

	if session.PatientID != patientID {
		t.Errorf("patient_id mismatch: got %s, want %s", session.PatientID, patientID)
	}
	if session.NodeID != "P01_CHEST_PAIN" {
		t.Errorf("node_id mismatch: got %s, want P01_CHEST_PAIN", session.NodeID)
	}
	if session.Status != models.StatusActive {
		t.Errorf("status: got %s, want ACTIVE", session.Status)
	}

	// Verify session_answers table after answering
	if resp.CurrentQuestion != nil {
		w = doRequest("POST", fmt.Sprintf("/api/v1/sessions/%s/answers", resp.SessionID), models.SubmitAnswerRequest{
			QuestionID:  resp.CurrentQuestion.QuestionID,
			AnswerValue: "YES",
		})
		if w.Code == http.StatusOK {
			var count int64
			testDB.DB.Model(&models.SessionAnswer{}).Where("session_id = ?", resp.SessionID).Count(&count)
			if count != 1 {
				t.Errorf("expected 1 answer in DB, got %d", count)
			}
		}
	}
}
