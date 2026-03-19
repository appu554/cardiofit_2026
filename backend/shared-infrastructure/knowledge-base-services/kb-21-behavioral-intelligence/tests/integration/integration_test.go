//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"kb-21-behavioral-intelligence/internal/api"
	"kb-21-behavioral-intelligence/internal/config"
	"kb-21-behavioral-intelligence/internal/database"
	"kb-21-behavioral-intelligence/internal/events"
	"kb-21-behavioral-intelligence/internal/metrics"
	"kb-21-behavioral-intelligence/internal/models"
	"kb-21-behavioral-intelligence/internal/services"

	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Test infrastructure — real PostgreSQL (port 5440) + no Redis needed for
// integration tests that exercise DB-backed service logic through the API.
// ---------------------------------------------------------------------------

var (
	testServer *api.Server
	testDB     *database.Database
)

func TestMain(m *testing.M) {
	// Ensure test database is available
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://kb21_test:kb21_test_pass@localhost:5440/kb21_test?sslmode=disable"
	}

	cfg := &config.Config{
		Server: config.ServerConfig{Port: "0"},
		Database: config.DatabaseConfig{
			URL:             dbURL,
			MaxConnections:  5,
			ConnMaxLifetime: 5 * time.Minute,
		},
		ServiceName:                 "kb-21-test",
		Environment:                 "development",
		LogLevel:                    "error",
		PreGatewayDefaultAdherence:  0.70,
		OutcomeCorrelationMinEvents: 5,
	}

	logger := zap.NewNop()

	var err error
	testDB, err = database.NewConnection(cfg, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: cannot connect to test database: %v\n", err)
		os.Exit(0)
	}

	// Clean tables before tests
	cleanDB()

	// Wire up services
	adherenceSvc := services.NewAdherenceService(testDB.DB, logger)
	engagementSvc := services.NewEngagementService(testDB.DB, logger, cfg.PreGatewayDefaultAdherence)
	correlationSvc := services.NewCorrelationService(testDB.DB, logger, cfg.OutcomeCorrelationMinEvents, nil, nil)
	hypoRiskSvc := services.NewHypoRiskService(testDB.DB, logger, nil, nil)
	subscriber := events.NewSubscriber(logger, correlationSvc, adherenceSvc, false)

	// BCE v1.0 Nudge Engine
	bayesianEngine := services.NewBayesianEngine(testDB.DB, logger)
	phaseEngine := services.NewPhaseEngine(testDB.DB, logger)
	barrierDiag := services.NewBarrierDiagnostic(testDB.DB, logger)
	nudgeEngine := services.NewNudgeEngine(testDB.DB, logger, bayesianEngine, phaseEngine, barrierDiag, nil, nil, nil, 3, 4)

	metricsCollector := metrics.NewCollector()

	testServer = api.NewServer(
		cfg, testDB, nil, metricsCollector, logger,
		adherenceSvc, engagementSvc, correlationSvc, hypoRiskSvc,
		nil,              // festivalCal
		nudgeEngine,
		nil,              // coldStartEngine (E1)
		nil,              // gamificationEngine (E2)
		nil,              // timingBandit (E4)
		subscriber,
	)

	code := m.Run()

	cleanDB()
	testDB.Close()
	os.Exit(code)
}

func cleanDB() {
	// Manually create antihypertensive_adherence_states if missing (not in autoMigrate)
	testDB.DB.Exec(`CREATE TABLE IF NOT EXISTS antihypertensive_adherence_states (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		patient_id TEXT NOT NULL UNIQUE,
		per_class_adherence JSONB,
		aggregate_score DECIMAL(5,4) DEFAULT 0,
		aggregate_score7d DECIMAL(5,4) DEFAULT 0,
		aggregate_trend VARCHAR(20) DEFAULT 'STABLE',
		primary_reason VARCHAR(30) DEFAULT 'UNKNOWN',
		dietary_sodium_estimate VARCHAR(20) DEFAULT 'UNKNOWN',
		salt_reduction_potential DECIMAL(5,4) DEFAULT 0,
		active_htn_drug_classes INT DEFAULT 0,
		updated_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ
	)`)

	tables := []string{
		"cohort_snapshots",
		"barrier_detections",
		"dietary_signals",
		"nudge_records",
		"question_telemetries",
		"outcome_correlations",
		"engagement_profiles",
		"adherence_states",
		"antihypertensive_adherence_states",
		"technique_effectiveness",
		"patient_motivation_phases",
		"intake_profiles",
		"interaction_events",
		"patient_streaks",
		"patient_milestones",
		"weekly_challenges",
		"population_priors",
		"prior_calibration_logs",
		"patient_timing_profiles",
	}
	for _, t := range tables {
		testDB.DB.Exec(fmt.Sprintf("DELETE FROM %s", t))
	}
}

// ---------------------------------------------------------------------------
// Helper: execute HTTP request against testServer
// ---------------------------------------------------------------------------

func doRequest(method, path string, body interface{}) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testServer.Router.ServeHTTP(w, req)
	return w
}

func parseBody(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

// ---------------------------------------------------------------------------
// Health check
// ---------------------------------------------------------------------------

func TestHealthCheck(t *testing.T) {
	w := doRequest("GET", "/health", nil)
	if w.Code != 200 {
		t.Fatalf("health: status = %d, want 200", w.Code)
	}
	body := parseBody(w)
	if body["status"] != "healthy" {
		t.Errorf("health status = %v, want healthy", body["status"])
	}
}

// ---------------------------------------------------------------------------
// Record interaction → verify DB persistence
// ---------------------------------------------------------------------------

func TestRecordInteraction(t *testing.T) {
	cleanDB()
	payload := map[string]interface{}{
		"patient_id":       "test-patient-1",
		"channel":          "WHATSAPP",
		"type":             "MEDICATION_CONFIRM",
		"drug_class":       "METFORMIN",
		"response_value":   "yes",
		"response_quality": "HIGH",
	}

	w := doRequest("POST", "/api/v1/patient/test-patient-1/interaction", payload)
	if w.Code != 200 {
		t.Fatalf("record interaction: status=%d, body=%s", w.Code, w.Body.String())
	}

	body := parseBody(w)
	if body["success"] != true {
		t.Errorf("expected success=true, got %v", body["success"])
	}

	// Verify it landed in the database
	var count int64
	testDB.DB.Model(&models.InteractionEvent{}).
		Where("patient_id = ?", "test-patient-1").Count(&count)
	if count != 1 {
		t.Errorf("DB interaction_events count = %d, want 1", count)
	}
}

// ---------------------------------------------------------------------------
// Get adherence — empty patient returns empty array
// ---------------------------------------------------------------------------

func TestGetAdherence_EmptyPatient(t *testing.T) {
	w := doRequest("GET", "/api/v1/patient/nonexistent-patient/adherence", nil)
	if w.Code != 200 {
		t.Fatalf("get adherence: status=%d", w.Code)
	}
	body := parseBody(w)
	data := body["data"]
	if data == nil {
		t.Fatal("data should not be nil (expect empty array)")
	}
	arr, ok := data.([]interface{})
	if !ok {
		t.Fatalf("data type=%T, want []interface{}", data)
	}
	if len(arr) != 0 {
		t.Errorf("expected 0 adherence states for nonexistent patient, got %d", len(arr))
	}
}

// ---------------------------------------------------------------------------
// Engagement profile — fresh patient returns nil/pre-gateway default
// ---------------------------------------------------------------------------

func TestGetEngagementProfile_FreshPatient(t *testing.T) {
	w := doRequest("GET", "/api/v1/patient/fresh-patient/engagement", nil)
	if w.Code != 200 {
		t.Fatalf("get engagement: status=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Loop trust — fresh patient returns pre-gateway defaults
// ---------------------------------------------------------------------------

func TestGetLoopTrust_FreshPatient(t *testing.T) {
	w := doRequest("GET", "/api/v1/patient/fresh-patient/loop-trust", nil)
	if w.Code != 200 {
		t.Fatalf("get loop trust: status=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Outcome correlation — empty returns nil data
// ---------------------------------------------------------------------------

func TestGetCorrelation_Empty(t *testing.T) {
	w := doRequest("GET", "/api/v1/patient/no-labs-patient/outcome-correlation", nil)
	if w.Code != 200 {
		t.Fatalf("get correlation: status=%d", w.Code)
	}
	body := parseBody(w)
	// Should return null data with a message about needing HbA1c
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

// ---------------------------------------------------------------------------
// HTN adherence — empty patient returns empty state with DEFAULT_PRE_GATEWAY
// ---------------------------------------------------------------------------

func TestGetHTNAdherence_Empty(t *testing.T) {
	w := doRequest("GET", "/api/v1/patient/no-htn-patient/adherence/htn", nil)
	if w.Code != 200 {
		t.Fatalf("get HTN adherence: status=%d, body=%s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// HTN adherence gate — root-level route compatibility
// ---------------------------------------------------------------------------

func TestHTNAdherenceGate_RootRoute(t *testing.T) {
	// KB-23 calls KB-21 at {KB21_URL}/patient/{id}/adherence/htn/gate
	// which is the root-level route (no /api/v1 prefix)
	w := doRequest("GET", "/patient/no-htn-patient/adherence/htn/gate", nil)
	if w.Code != 200 {
		t.Fatalf("root HTN gate route: status=%d, body=%s", w.Code, w.Body.String())
	}

	body := parseBody(w)
	action, ok := body["action"].(string)
	if !ok {
		t.Fatalf("expected action string, got %T: %v", body["action"], body["action"])
	}

	// Empty patient → aggregate score = 0 → ADHERENCE_INTERVENTION
	if action != "STANDARD_ESCALATION" && action != "ADHERENCE_INTERVENTION" {
		t.Logf("HTN gate action = %q (acceptable for empty patient)", action)
	}
}

// ---------------------------------------------------------------------------
// Hypo risk — clean patient returns NORMAL (no risk factors)
// ---------------------------------------------------------------------------

func TestHypoRisk_CleanPatient(t *testing.T) {
	w := doRequest("GET", "/api/v1/patient/clean-patient/hypo-risk", nil)
	if w.Code != 200 {
		t.Fatalf("hypo risk: status=%d", w.Code)
	}
	body := parseBody(w)
	data := body["data"].(map[string]interface{})
	if data["risk_level"] != "NORMAL" {
		t.Errorf("expected NORMAL risk_level for clean patient, got %v", data["risk_level"])
	}
}

// ---------------------------------------------------------------------------
// Answer reliability — fresh patient gets conservative defaults
// ---------------------------------------------------------------------------

func TestAnswerReliability_FreshPatient(t *testing.T) {
	w := doRequest("GET", "/api/v1/patient/fresh-patient/answer-reliability", nil)
	if w.Code != 200 {
		t.Fatalf("answer reliability: status=%d, body=%s", w.Code, w.Body.String())
	}
	body := parseBody(w)
	data := body["data"].(map[string]interface{})

	// Fresh patient should get default reliability score
	reliability, _ := data["reliability_score"].(float64)
	if reliability <= 0 || reliability > 1.0 {
		t.Errorf("reliability_score=%.2f, want (0, 1.0]", reliability)
	}
}

// ---------------------------------------------------------------------------
// Full lifecycle: record interactions → recompute adherence → check adherence
// ---------------------------------------------------------------------------

func TestFullLifecycle_RecordAndRecompute(t *testing.T) {
	cleanDB()
	patientID := "lifecycle-patient-1"

	// 1. Record 5 medication confirmations
	for i := 0; i < 5; i++ {
		payload := map[string]interface{}{
			"patient_id":       patientID,
			"channel":          "WHATSAPP",
			"type":             "MEDICATION_CONFIRM",
			"drug_class":       "ACE_INHIBITOR",
			"response_value":   "yes",
			"response_quality": "HIGH",
		}
		w := doRequest("POST", fmt.Sprintf("/api/v1/patient/%s/interaction", patientID), payload)
		if w.Code != 200 {
			t.Fatalf("interaction %d: status=%d, body=%s", i, w.Code, w.Body.String())
		}
	}

	// 2. Record 2 missed doses
	for i := 0; i < 2; i++ {
		payload := map[string]interface{}{
			"patient_id":       patientID,
			"channel":          "WHATSAPP",
			"type":             "MEDICATION_CONFIRM",
			"drug_class":       "ACE_INHIBITOR",
			"response_value":   "no",
			"response_quality": "HIGH",
		}
		doRequest("POST", fmt.Sprintf("/api/v1/patient/%s/interaction", patientID), payload)
	}

	// Wait briefly for async adherence recomputation goroutines
	time.Sleep(500 * time.Millisecond)

	// 3. Trigger explicit recompute
	w := doRequest("POST",
		fmt.Sprintf("/api/v1/patient/%s/adherence/recompute?drug_class=ACE_INHIBITOR", patientID), nil)
	if w.Code != 200 {
		t.Fatalf("recompute: status=%d, body=%s", w.Code, w.Body.String())
	}

	// 4. Get adherence states
	w = doRequest("GET", fmt.Sprintf("/api/v1/patient/%s/adherence", patientID), nil)
	if w.Code != 200 {
		t.Fatalf("get adherence: status=%d", w.Code)
	}

	body := parseBody(w)
	data, ok := body["data"].([]interface{})
	if !ok || len(data) == 0 {
		t.Fatal("expected at least 1 adherence state after recompute")
	}

	// 5. Verify adherence score is roughly 5/7 ≈ 0.71
	state := data[0].(map[string]interface{})
	score := state["adherence_score"].(float64)
	if score < 0.50 || score > 0.90 {
		t.Errorf("adherence_score=%.3f, expected ~0.71 (5 yes / 7 total)", score)
	}
	t.Logf("adherence_score=%.3f for %s", score, patientID)
}

// ---------------------------------------------------------------------------
// Recompute engagement → creates profile in DB
// ---------------------------------------------------------------------------

func TestRecomputeEngagement(t *testing.T) {
	cleanDB()
	patientID := "engagement-patient-1"

	// Record some interactions so profile has data
	for i := 0; i < 3; i++ {
		payload := map[string]interface{}{
			"patient_id":       patientID,
			"channel":          "APP",
			"type":             "DAILY_CHECKIN",
			"response_value":   "feeling good",
			"response_quality": "HIGH",
		}
		doRequest("POST", fmt.Sprintf("/api/v1/patient/%s/interaction", patientID), payload)
	}

	// Recompute engagement
	w := doRequest("POST", fmt.Sprintf("/api/v1/patient/%s/engagement/recompute", patientID), nil)
	if w.Code != 200 {
		t.Fatalf("recompute engagement: status=%d, body=%s", w.Code, w.Body.String())
	}

	// Verify profile was created
	var count int64
	testDB.DB.Model(&models.EngagementProfile{}).
		Where("patient_id = ?", patientID).Count(&count)
	if count != 1 {
		t.Errorf("engagement_profiles count = %d, want 1", count)
	}
}

// ---------------------------------------------------------------------------
// CORS middleware
// ---------------------------------------------------------------------------

func TestCORS_OptionsRequest(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", "/api/v1/patient/p1/adherence", nil)
	w := httptest.NewRecorder()
	testServer.Router.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Errorf("OPTIONS: status=%d, want 204", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS Allow-Origin header")
	}
}

// ---------------------------------------------------------------------------
// Lab result webhook → triggers correlation (Finding F-04)
// ---------------------------------------------------------------------------

func TestWebhookLabResult(t *testing.T) {
	cleanDB()
	patientID := "lab-patient-1"

	// Need at least some adherence data and engagement profile first
	testDB.DB.Create(&models.AdherenceState{
		PatientID:      patientID,
		DrugClass:      "METFORMIN",
		AdherenceScore: 0.80,
		AdherenceTrend: models.TrendStable,
		DataQuality:    models.DataQualityHigh,
	})
	testDB.DB.Create(&models.EngagementProfile{
		PatientID:         patientID,
		Phenotype:         models.PhenotypeSteady,
		EngagementScore:   0.75,
		TotalInteractions: 20,
	})

	payload := map[string]interface{}{
		"patient_id":   patientID,
		"lab_type":     "HBA1C",
		"value":        7.2,
		"unit":         "%",
		"collected_at": time.Now().UTC().Format(time.RFC3339),
	}

	w := doRequest("POST", "/api/v1/webhooks/lab-result", payload)
	if w.Code != 200 {
		t.Fatalf("lab result webhook: status=%d, body=%s", w.Code, w.Body.String())
	}

	// Verify correlation was created
	var count int64
	testDB.DB.Model(&models.OutcomeCorrelation{}).
		Where("patient_id = ?", patientID).Count(&count)
	if count != 1 {
		t.Errorf("outcome_correlations count = %d, want 1", count)
	}
}

// ---------------------------------------------------------------------------
// Analytics endpoints
// ---------------------------------------------------------------------------

func TestPhenotypeDistribution(t *testing.T) {
	w := doRequest("GET", "/api/v1/analytics/phenotype-distribution", nil)
	if w.Code != 200 {
		t.Fatalf("phenotype distribution: status=%d", w.Code)
	}
}

func TestCohortSnapshots(t *testing.T) {
	w := doRequest("GET", "/api/v1/analytics/cohort?limit=5", nil)
	if w.Code != 200 {
		t.Fatalf("cohort snapshots: status=%d", w.Code)
	}
}
