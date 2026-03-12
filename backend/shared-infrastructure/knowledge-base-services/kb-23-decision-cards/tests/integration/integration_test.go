//go:build integration

// Package integration tests the KB-23 Decision Cards engine against real
// PostgreSQL and Redis containers (docker-compose.test.yml).
//
// Prerequisites:
//   docker-compose -f docker-compose.test.yml up -d
//
// Run:
//   TEST_DATABASE_URL=postgres://kb23_test:kb23_test_pass@localhost:5439/kb23_test?sslmode=disable \
//   TEST_REDIS_URL=redis://localhost:6388 \
//   go test -tags=integration -v -count=1 ./tests/integration/...
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

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/api"
	"kb-23-decision-cards/internal/cache"
	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
	"kb-23-decision-cards/internal/services"
)

var (
	testServer *httptest.Server
	serverObj  *api.Server
	db         *database.Database
	testCache  *cache.CacheClient
	log        *zap.Logger
	cfg        *config.Config
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	log, _ = zap.NewDevelopment()

	dbURL := envOr("TEST_DATABASE_URL", "postgres://kb23_test:kb23_test_pass@localhost:5439/kb23_test?sslmode=disable")
	redisURL := envOr("TEST_REDIS_URL", "redis://localhost:6388")

	// Stub KB-19 (accepts events)
	kb19Stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
	}))
	defer kb19Stub.Close()

	// Stub KB-20 (returns patient context)
	kb20Stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"stratum_label":  "DM_HTN_base",
			"patient_sex":    "Male",
			"patient_age":    62,
			"egfr_value":     55.0,
			"latest_hba1c":   7.2,
			"latest_fbg":     8.5,
			"weight_kg":      78.0,
			"is_acute_ill":   false,
			"medications":    []string{"METFORMIN", "AMLODIPINE"},
			"bp_status":      "CONTROLLED",
		})
	}))
	defer kb20Stub.Close()

	// Stub KB-21 (returns adherence)
	kb21Stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"gain_factor":      0.92,
			"adherence_score":  0.85,
			"reliability_tier": "HIGH",
		})
	}))
	defer kb21Stub.Close()

	// Build config
	cfg = &config.Config{
		Port:                           "0",
		Environment:                    "test",
		DatabaseURL:                    dbURL,
		DBMaxConnections:               5,
		DBConnMaxLife:                  5 * time.Minute,
		RedisURL:                       redisURL,
		RedisDB:                        0,
		RedisMCUGateTTL:               1 * time.Hour,
		RedisAdherenceTTL:             1 * time.Hour,
		RedisGateHistoryTTL:           24 * time.Hour,
		TemplatesDir:                   "../../templates",
		KB19URL:                        kb19Stub.URL,
		KB20URL:                        kb20Stub.URL,
		KB21URL:                        kb21Stub.URL,
		KB19TimeoutMS:                  500,
		KB20TimeoutMS:                  200,
		KB21TimeoutMS:                  200,
		DefaultFirmPosterior:           0.75,
		DefaultFirmMedicationChange:    0.82,
		DefaultProbablePosterior:       0.60,
		DefaultPossiblePosterior:       0.40,
		HysteresisWindowHours:          72,
		HysteresisMinSessions:          2,
		HypoglycaemiaSevereThreshold:   3.0,
		HypoglycaemiaModerateThreshold: 3.9,
		SafetyAlertPublishTimeout:      2 * time.Second,
		MetricsEnabled:                 false,
	}

	// Connect DB
	var err error
	db, err = database.NewConnection(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: DB not reachable: %v\n", err)
		os.Exit(0)
	}
	if err := db.AutoMigrate(); err != nil {
		fmt.Fprintf(os.Stderr, "AutoMigrate failed: %v\n", err)
		os.Exit(1)
	}

	// Connect Redis
	testCache, err = cache.NewCacheClient(cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: Redis not reachable: %v\n", err)
		os.Exit(0)
	}

	// Load templates
	m2 := metrics.NewCollector()
	templateLoader := services.NewTemplateLoader(cfg.TemplatesDir, log)
	if err := templateLoader.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "template load: %v\n", err)
		os.Exit(1)
	}

	// Create server
	serverObj = api.NewServer(cfg, db, testCache, m2, log, templateLoader)
	serverObj.InitServices()
	serverObj.RegisterRoutes()

	testServer = httptest.NewServer(serverObj.Router)
	defer testServer.Close()

	// Run tests
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func httpJSON(t *testing.T, method, path string, body interface{}) (int, map[string]interface{}) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, testServer.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("request build: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(respBody, &result)
	return resp.StatusCode, result
}

func cleanDB(t *testing.T) {
	t.Helper()
	tables := []string{
		"card_recommendations",
		"mcu_gate_history",
		"composite_card_signals",
		"hypoglycaemia_alerts",
		"treatment_perturbations",
		"decision_cards",
	}
	for _, table := range tables {
		db.DB.Exec("DELETE FROM " + table)
	}
}

// ---------------------------------------------------------------------------
// I-01: Health and Readiness
// ---------------------------------------------------------------------------

func TestHealth_RealInfrastructure(t *testing.T) {
	code, body := httpJSON(t, "GET", "/health", nil)
	if code != 200 {
		t.Fatalf("GET /health returned %d", code)
	}
	if body["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", body["status"])
	}
	if body["service"] != "kb-23-decision-cards" {
		t.Errorf("service = %v", body["service"])
	}
}

func TestReadiness_RealInfrastructure(t *testing.T) {
	code, body := httpJSON(t, "GET", "/readiness", nil)
	if code != 200 {
		t.Fatalf("GET /readiness returned %d: %v", code, body)
	}
	if body["status"] != "ready" {
		t.Errorf("status = %v", body["status"])
	}
	count := body["templates_loaded"].(float64)
	if count < 1 {
		t.Errorf("templates_loaded = %v, want >=1", count)
	}
	t.Logf("templates loaded: %v", count)
}

// ---------------------------------------------------------------------------
// I-02: Card Generation (full lifecycle)
// ---------------------------------------------------------------------------

func TestGenerateCard_FullLifecycle(t *testing.T) {
	cleanDB(t)
	patientID := uuid.New()
	sessionID := uuid.New()

	// Generate card from HPI_COMPLETE event — node_id and top_diagnosis
	// must match a loaded template (P01 / ACS_NSTEMI from acs_nstemi.yaml)
	code, body := httpJSON(t, "POST", "/api/v1/decision-cards", map[string]interface{}{
		"event_type":    "HPI_COMPLETE",
		"patient_id":    patientID.String(),
		"session_id":    sessionID.String(),
		"node_id":       "P01",
		"stratum_label": "DM_HTN_base",
		"top_diagnosis": "ACS_NSTEMI",
		"top_posterior":  0.82,
		"ranked_differentials": []map[string]interface{}{
			{"differential_id": "ACS_NSTEMI", "posterior": 0.82},
			{"differential_id": "ACS_STEMI", "posterior": 0.10},
			{"differential_id": "GERD", "posterior": 0.05},
		},
		"safety_flags": []map[string]interface{}{
			{"flag_id": "RF01", "severity": "IMMEDIATE", "recommended_action": "Stat ECG"},
		},
		"convergence_reached": true,
	})

	if code != 201 {
		t.Fatalf("POST /decision-cards returned %d: %v", code, body)
	}

	cardID := body["card_id"].(string)
	t.Logf("created card %s", cardID)

	// Verify card fields
	if body["template_id"] == nil || body["template_id"] == "" {
		t.Error("template_id is empty")
	}
	if body["mcu_gate"] == nil {
		t.Error("mcu_gate is nil")
	}
	if body["diagnostic_confidence_tier"] == nil {
		t.Error("diagnostic_confidence_tier is nil")
	}
	if body["status"] != string(models.StatusActive) {
		t.Errorf("status = %v, want ACTIVE", body["status"])
	}

	t.Logf("card: template=%v tier=%v gate=%v safety=%v",
		body["template_id"], body["diagnostic_confidence_tier"],
		body["mcu_gate"], body["safety_tier"])

	// Fetch the card back
	code2, card := httpJSON(t, "GET", "/api/v1/cards/"+cardID, nil)
	if code2 != 200 {
		t.Fatalf("GET /cards/%s returned %d: %v", cardID, code2, card)
	}
	if card["card_id"] != cardID {
		t.Errorf("card_id mismatch: %v vs %v", card["card_id"], cardID)
	}
}

// ---------------------------------------------------------------------------
// I-03: Card Not Found
// ---------------------------------------------------------------------------

func TestGetCard_NotFound(t *testing.T) {
	code, body := httpJSON(t, "GET", "/api/v1/cards/"+uuid.New().String(), nil)
	if code != 404 {
		t.Errorf("expected 404, got %d: %v", code, body)
	}
}

func TestGetCard_InvalidUUID(t *testing.T) {
	code, body := httpJSON(t, "GET", "/api/v1/cards/not-a-uuid", nil)
	if code != 400 {
		t.Errorf("expected 400, got %d: %v", code, body)
	}
}

// ---------------------------------------------------------------------------
// I-04: Generate Card — No matching template
// ---------------------------------------------------------------------------

func TestGenerateCard_NoTemplate(t *testing.T) {
	code, body := httpJSON(t, "POST", "/api/v1/decision-cards", map[string]interface{}{
		"patient_id":    uuid.New().String(),
		"session_id":    uuid.New().String(),
		"node_id":       "P99_NONEXISTENT",
		"top_diagnosis": "FAKE_DIAGNOSIS",
		"top_posterior":  0.50,
	})
	if code != 422 {
		t.Errorf("expected 422 for no template, got %d: %v", code, body)
	}
}

// ---------------------------------------------------------------------------
// I-05: Generate Card — Invalid payload
// ---------------------------------------------------------------------------

func TestGenerateCard_InvalidPayload(t *testing.T) {
	code, _ := httpJSON(t, "POST", "/api/v1/decision-cards", "not json")
	if code != 400 {
		t.Errorf("expected 400, got %d", code)
	}
}

// ---------------------------------------------------------------------------
// I-06: Active Cards for Patient
// ---------------------------------------------------------------------------

func TestGetActiveCards_RealDB(t *testing.T) {
	cleanDB(t)
	patientID := uuid.New()

	// Create a card directly in DB
	card := models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               "test_tmpl",
		NodeID:                   "P01_CHEST_PAIN",
		PrimaryDifferentialID:    "ACS",
		PrimaryPosterior:         0.80,
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  models.GateSafe,
		SafetyTier:               models.SafetyRoutine,
		CardSource:               models.SourceKB22Session,
		Status:                   models.StatusActive,
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
	if err := db.DB.Create(&card).Error; err != nil {
		t.Fatalf("create card: %v", err)
	}

	code, body := httpJSON(t, "GET", fmt.Sprintf("/api/v1/patients/%s/active-cards", patientID.String()), nil)
	if code != 200 {
		t.Fatalf("GET /patients/:id/active-cards returned %d: %v", code, body)
	}
	count := body["count"].(float64)
	if count != 1 {
		t.Errorf("count = %v, want 1", count)
	}
}

// ---------------------------------------------------------------------------
// I-07: MCU Gate Query
// ---------------------------------------------------------------------------

func TestGetMCUGate_NoGate(t *testing.T) {
	code, body := httpJSON(t, "GET", "/api/v1/patients/"+uuid.New().String()+"/mcu-gate", nil)
	if code != 404 {
		t.Errorf("expected 404 for no gate, got %d: %v", code, body)
	}
}

// ---------------------------------------------------------------------------
// I-08: Hypoglycaemia Fast-Path
// ---------------------------------------------------------------------------

func TestHypoglycaemiaAlert_RealDB(t *testing.T) {
	cleanDB(t)
	patientID := uuid.New()

	code, body := httpJSON(t, "POST", "/api/v1/safety/hypoglycaemia-alert", map[string]interface{}{
		"patient_id":    patientID.String(),
		"source":        "CGM",
		"glucose_mmol_l": 2.8,
		"severity":      "SEVERE",
		"timestamp":     time.Now().Format(time.RFC3339),
	})

	if code != 201 {
		t.Fatalf("POST /safety/hypoglycaemia-alert returned %d: %v", code, body)
	}

	cardID := body["card_id"].(string)
	t.Logf("hypo card created: %s", cardID)

	// Verify HALT gate for severe hypo
	if body["mcu_gate"] != string(models.GateHalt) {
		t.Errorf("gate = %v, want HALT for severe hypo", body["mcu_gate"])
	}
	if body["safety_tier"] != string(models.SafetyImmediate) {
		t.Errorf("safety_tier = %v, want IMMEDIATE", body["safety_tier"])
	}

	// Verify card persisted
	code2, card := httpJSON(t, "GET", "/api/v1/cards/"+cardID, nil)
	if code2 != 200 {
		t.Fatalf("GET /cards/%s returned %d", cardID, code2)
	}
	t.Logf("persisted hypo card: gate=%v tier=%v", card["mcu_gate"], card["diagnostic_confidence_tier"])
}

// ---------------------------------------------------------------------------
// I-09: Hypoglycaemia — Moderate severity
// ---------------------------------------------------------------------------

func TestHypoglycaemiaAlert_Moderate(t *testing.T) {
	cleanDB(t)

	code, body := httpJSON(t, "POST", "/api/v1/safety/hypoglycaemia-alert", map[string]interface{}{
		"patient_id":    uuid.New().String(),
		"source":        "GLUCOMETER",
		"glucose_mmol_l": 3.5,
		"timestamp":     time.Now().Format(time.RFC3339),
	})
	if code != 201 {
		t.Fatalf("returned %d: %v", code, body)
	}
	if body["mcu_gate"] != string(models.GatePause) {
		t.Errorf("gate = %v, want PAUSE for moderate hypo", body["mcu_gate"])
	}
}

// ---------------------------------------------------------------------------
// I-10: Treatment Perturbation Lifecycle
// ---------------------------------------------------------------------------

func TestPerturbationLifecycle(t *testing.T) {
	cleanDB(t)
	patientID := uuid.New()
	now := time.Now()

	// Store perturbation
	code, body := httpJSON(t, "POST", "/api/v1/perturbations", map[string]interface{}{
		"patient_id":        patientID.String(),
		"intervention_type": "INSULIN_INCREASE",
		"dose_delta":        4.0,
		"baseline_dose":     20.0,
		"effect_window_start": now.Format(time.RFC3339),
		"effect_window_end":   now.Add(72 * time.Hour).Format(time.RFC3339),
		"affected_observables": []string{"FBG", "PPG"},
		"stability_factor":    0.7,
	})
	if code != 201 {
		t.Fatalf("POST /perturbations returned %d: %v", code, body)
	}
	t.Logf("perturbation stored: %v", body["perturbation_id"])

	// Query active perturbations
	code2, active := httpJSON(t, "GET", fmt.Sprintf("/api/v1/perturbations/%s/active", patientID.String()), nil)
	if code2 != 200 {
		t.Fatalf("GET /perturbations/.../active returned %d: %v", code2, active)
	}
	count := active["count"].(float64)
	if count < 1 {
		t.Errorf("expected >=1 active perturbation, got %v", count)
	}
}

// ---------------------------------------------------------------------------
// I-11: MCU Gate Resume — Card Not Found
// ---------------------------------------------------------------------------

func TestMCUGateResume_CardNotFound(t *testing.T) {
	code, body := httpJSON(t, "POST",
		fmt.Sprintf("/api/v1/cards/%s/mcu-gate-resume", uuid.New().String()),
		map[string]interface{}{
			"clinician_id": "DR_TEST",
			"reason":       "Patient stabilized",
		})
	// Should be 404 or 500 depending on how lifecycle handles it
	if code == 200 {
		t.Errorf("expected non-200 for missing card, got 200: %v", body)
	}
	t.Logf("gate resume for missing card: status=%d error=%v", code, body["error"])
}

// ---------------------------------------------------------------------------
// I-12: Template Reload
// ---------------------------------------------------------------------------

func TestTemplateReload(t *testing.T) {
	code, body := httpJSON(t, "POST", "/internal/templates/reload", nil)
	if code != 200 {
		t.Fatalf("POST /internal/templates/reload returned %d: %v", code, body)
	}
	if body["status"] != "reloaded" {
		t.Errorf("status = %v, want reloaded", body["status"])
	}
	count := body["templates_loaded"].(float64)
	if count < 1 {
		t.Errorf("templates_loaded = %v after reload, want >=1", count)
	}
	t.Logf("templates after reload: %v", count)
}

// ---------------------------------------------------------------------------
// I-13: Database Persistence — Card Survives Refetch
// ---------------------------------------------------------------------------

func TestDatabasePersistence_CardSurvives(t *testing.T) {
	cleanDB(t)
	patientID := uuid.New()
	card := models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               "persist_test",
		NodeID:                   "P01_CHEST_PAIN",
		PrimaryDifferentialID:    "ACS",
		PrimaryPosterior:         0.88,
		DiagnosticConfidenceTier: models.TierFirm,
		MCUGate:                  models.GateModify,
		MCUGateRationale:         "test rationale",
		SafetyTier:               models.SafetyUrgent,
		CardSource:               models.SourceKB22Session,
		Status:                   models.StatusActive,
		ClinicianSummary:         "Test clinician summary",
		PatientSummaryEn:         "Test patient summary",
		CreatedAt:                time.Now(),
		UpdatedAt:                time.Now(),
	}
	if err := db.DB.Create(&card).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	// Verify via GORM
	var fetched models.DecisionCard
	if err := db.DB.First(&fetched, "card_id = ?", card.CardID).Error; err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if fetched.PrimaryPosterior != 0.88 {
		t.Errorf("posterior = %v, want 0.88", fetched.PrimaryPosterior)
	}
	if fetched.MCUGate != models.GateModify {
		t.Errorf("gate = %v, want MODIFY", fetched.MCUGate)
	}
	if fetched.ClinicianSummary != "Test clinician summary" {
		t.Errorf("clinician_summary mismatch")
	}
}

// ---------------------------------------------------------------------------
// I-14: Behavioral Gap Alert
// ---------------------------------------------------------------------------

func TestBehavioralGapAlert_RealDB(t *testing.T) {
	cleanDB(t)

	code, body := httpJSON(t, "POST", "/api/v1/safety/behavioral-gap-alert", map[string]interface{}{
		"patient_id":              uuid.New().String(),
		"source":                  "KB21",
		"alert_type":              "BEHAVIORAL_GAP",
		"treatment_response_class": "DISCORDANT",
		"mean_adherence_score":    0.45,
		"hba1c_delta":             1.2,
		"timestamp":              time.Now().Format(time.RFC3339),
	})

	if code != 201 {
		t.Fatalf("POST /safety/behavioral-gap-alert returned %d: %v", code, body)
	}
	t.Logf("behavioral gap card: id=%v gate=%v", body["card_id"], body["mcu_gate"])
}

// ---------------------------------------------------------------------------
// I-15: CORS Headers
// ---------------------------------------------------------------------------

func TestCORSHeaders(t *testing.T) {
	req, _ := http.NewRequest("OPTIONS", testServer.URL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS /health: %v", err)
	}
	defer resp.Body.Close()

	allow := resp.Header.Get("Access-Control-Allow-Origin")
	if allow != "*" {
		t.Errorf("CORS Allow-Origin = %q, want *", allow)
	}
	methods := resp.Header.Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("CORS Allow-Methods header missing")
	}
}
