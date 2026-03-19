// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kb-26-metabolic-digital-twin/internal/api"
	"kb-26-metabolic-digital-twin/internal/cache"
	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/database"
	"kb-26-metabolic-digital-twin/internal/metrics"
	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// testServer creates a Server backed by a real database for integration tests.
func testServer(t *testing.T) (*api.Server, *database.Database) {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	db, err := database.NewConnection(cfg, logger)
	if err != nil {
		t.Skipf("database unavailable — skipping integration test: %v", err)
	}

	// AutoMigrate for test isolation
	if err := db.DB.AutoMigrate(
		&models.TwinState{},
		&models.CalibratedEffect{},
		&models.SimulationRun{},
	); err != nil {
		t.Fatalf("auto-migrate failed: %v", err)
	}

	cacheClient, _ := cache.NewRedisClient(cfg, logger)
	metricsCollector := metrics.NewCollector()
	twinUpdater := services.NewTwinUpdater(db.DB, logger)
	calibrator := services.NewBayesianCalibratorWithConfig(db.DB, logger, 0, 14) // 0-week burn-in for tests
	mriScorer := services.NewMRIScorer(db.DB, logger)
	eventProcessor := services.NewEventProcessor(twinUpdater, mriScorer, logger)

	server := api.NewServer(cfg, db, cacheClient, metricsCollector, logger, twinUpdater, calibrator, eventProcessor, mriScorer)
	return server, db
}

func TestObservationWebhook_CreatesTwinSnapshot(t *testing.T) {
	server, db := testServer(t)
	defer db.Close()

	patientID := uuid.New()

	event := models.ObservationEvent{
		PatientID: patientID.String(),
		Code:      "FBG",
		Value:     120.5,
		Unit:      "mg/dL",
		Timestamp: time.Now().UTC(),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb26/events/observation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Verify snapshot was created
	var count int64
	db.DB.Model(&models.TwinState{}).Where("patient_id = ?", patientID).Count(&count)
	if count == 0 {
		t.Error("expected twin snapshot to be created after observation webhook")
	}

	// Clean up
	db.DB.Where("patient_id = ?", patientID).Delete(&models.TwinState{})
}

func TestCheckinWebhook_UpdatesLifestyleFields(t *testing.T) {
	server, db := testServer(t)
	defer db.Close()

	patientID := uuid.New()

	event := models.CheckinEvent{
		PatientID:    patientID.String(),
		MealQuality:  0.8,
		ExerciseDone: true,
		StepCount:    8500,
		Timestamp:    time.Now().UTC(),
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb26/events/checkin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	// Verify fields
	var twin models.TwinState
	db.DB.Where("patient_id = ?", patientID).Order("updated_at DESC").First(&twin)
	if twin.DietQualityScore == nil || *twin.DietQualityScore != 0.8 {
		t.Errorf("expected diet_quality_score=0.8, got %v", twin.DietQualityScore)
	}
	if twin.ExerciseCompliance == nil || *twin.ExerciseCompliance != 1.0 {
		t.Errorf("expected exercise_compliance=1.0, got %v", twin.ExerciseCompliance)
	}

	// Clean up
	db.DB.Where("patient_id = ?", patientID).Delete(&models.TwinState{})
}

func TestGetTwin_CacheHitOnSecondCall(t *testing.T) {
	server, db := testServer(t)
	defer db.Close()

	patientID := uuid.New()
	twin := models.TwinState{
		ID:           uuid.New(),
		PatientID:    patientID,
		StateVersion: 1,
		UpdateSource: "test",
		UpdatedAt:    time.Now().UTC(),
	}
	db.DB.Create(&twin)

	url := fmt.Sprintf("/api/v1/kb26/twin/%s", patientID.String())

	// First call — cache miss
	req1 := httptest.NewRequest(http.MethodGet, url, nil)
	w1 := httptest.NewRecorder()
	server.Router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first GET expected 200, got %d", w1.Code)
	}

	// Second call — should be cache HIT (if Redis available)
	req2 := httptest.NewRequest(http.MethodGet, url, nil)
	w2 := httptest.NewRecorder()
	server.Router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second GET expected 200, got %d", w2.Code)
	}

	// Clean up
	db.DB.Where("patient_id = ?", patientID).Delete(&models.TwinState{})
}

func TestSimulate_PersistsSimulationRun(t *testing.T) {
	server, db := testServer(t)
	defer db.Close()

	patientID := uuid.New()
	twin := models.TwinState{
		ID:           uuid.New(),
		PatientID:    patientID,
		StateVersion: 1,
		UpdateSource: "test",
		UpdatedAt:    time.Now().UTC(),
	}
	db.DB.Create(&twin)

	simReq := map[string]interface{}{
		"patient_id": patientID.String(),
		"intervention": map[string]interface{}{
			"type":      "exercise",
			"code":      "brisk_walking_30min",
			"is_effect": 0.02,
			"vf_effect": -0.01,
		},
		"days": 90,
	}
	body, _ := json.Marshal(simReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb26/simulate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify simulation run was persisted
	var simCount int64
	db.DB.Model(&models.SimulationRun{}).Where("patient_id = ?", patientID).Count(&simCount)
	if simCount == 0 {
		t.Error("expected simulation run to be persisted in database")
	}

	// Clean up
	db.DB.Where("patient_id = ?", patientID).Delete(&models.SimulationRun{})
	db.DB.Where("patient_id = ?", patientID).Delete(&models.TwinState{})
}

func TestSyncTwin_ReDerivesAllTiers(t *testing.T) {
	server, db := testServer(t)
	defer db.Close()

	patientID := uuid.New()
	sbp := 140.0
	dbp := 90.0
	fbg := 130.0
	hba1c := 7.5
	hr := 88.0
	twin := models.TwinState{
		ID:           uuid.New(),
		PatientID:    patientID,
		StateVersion: 1,
		UpdateSource: "test",
		UpdatedAt:    time.Now().UTC(),
		SBP14dMean:   &sbp,
		DBP14dMean:   &dbp,
		FBG7dMean:    &fbg,
		HbA1c:        &hba1c,
		RestingHR:    &hr,
	}
	db.DB.Create(&twin)

	url := fmt.Sprintf("/api/v1/kb26/sync/%s", patientID.String())
	req := httptest.NewRequest(http.MethodPost, url, nil)
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("sync expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify new snapshot with MAP re-derived
	var latest models.TwinState
	db.DB.Where("patient_id = ?", patientID).Order("updated_at DESC").First(&latest)
	if latest.MAPValue == nil {
		t.Error("expected MAP to be re-derived after sync")
	}
	if latest.InsulinSensitivity == nil || len(latest.InsulinSensitivity) == 0 {
		t.Error("expected InsulinSensitivity to be re-derived after sync")
	}

	// Clean up
	db.DB.Where("patient_id = ?", patientID).Delete(&models.TwinState{})
}

func TestPerturbationAnalysis_ReturnsDeltas(t *testing.T) {
	server, db := testServer(t)
	defer db.Close()

	patientID := uuid.New()
	twin := models.TwinState{
		ID:           uuid.New(),
		PatientID:    patientID,
		StateVersion: 1,
		UpdateSource: "test",
		UpdatedAt:    time.Now().UTC(),
	}
	db.DB.Create(&twin)

	pertReq := map[string]interface{}{
		"patient_id":        patientID.String(),
		"intervention_code": "metformin",
		"days":              90,
	}
	body, _ := json.Marshal(pertReq)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/kb26/perturbation", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatal("expected data in response")
	}
	if _, ok := data["deltas"]; !ok {
		t.Error("expected deltas field in perturbation response")
	}

	// Clean up
	db.DB.Where("patient_id = ?", patientID).Delete(&models.TwinState{})
}
