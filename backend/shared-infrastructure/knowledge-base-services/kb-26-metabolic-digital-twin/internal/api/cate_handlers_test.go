package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/database"
	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// newCATETestDB builds a sqlite in-memory DB with cate_estimates and
// attribution_verdicts tables — the two tables the CATE handler tests need.
// Intervention definitions are not created here because no Sprint 1 handler
// test queries them; extend with CREATE TABLE intervention_definitions when
// Sprint 3's recommender endpoint tests land. Mirrors the newTestDB helper
// pattern but scoped to Gap 22 handler tests.
func newCATETestDB(t *testing.T) *database.Database {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	ddl := []string{
		`CREATE TABLE cate_estimates (
			id                         TEXT PRIMARY KEY,
			consolidated_record_id     TEXT NOT NULL,
			patient_id                 TEXT NOT NULL,
			cohort_id                  TEXT NOT NULL,
			intervention_id            TEXT NOT NULL,
			learner_type               TEXT NOT NULL,
			point_estimate             REAL,
			ci_lower                   REAL,
			ci_upper                   REAL,
			horizon_days               INTEGER,
			propensity                 REAL,
			overlap_status             TEXT NOT NULL,
			training_n                 INTEGER,
			cohort_treated_n           INTEGER,
			cohort_control_n           INTEGER,
			feature_contributions_json TEXT,
			feature_contribution_keys  TEXT,
			model_version              TEXT,
			ledger_entry_id            TEXT,
			computed_at                DATETIME
		)`,
		`CREATE TABLE attribution_verdicts (
			id                         TEXT PRIMARY KEY,
			consolidated_record_id     TEXT NOT NULL,
			patient_id                 TEXT NOT NULL,
			cohort_id                  TEXT,
			clinician_label            TEXT NOT NULL,
			technical_label            TEXT,
			risk_difference            REAL,
			risk_reduction_pct         REAL,
			counterfactual_risk        REAL,
			observed_outcome           INTEGER,
			prediction_window_days     INTEGER,
			attribution_method         TEXT NOT NULL DEFAULT 'RULE_BASED',
			method_version             TEXT,
			rationale                  TEXT,
			ledger_entry_id            TEXT,
			computed_at                DATETIME
		)`,
	}
	for _, s := range ddl {
		if err := gdb.Exec(s).Error; err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	return &database.Database{DB: gdb}
}

func TestPostCATEEstimate_ReturnsSprint1Stub(t *testing.T) {
	r := newTestEngine()
	srv := &Server{}
	r.POST("/cate/estimate", srv.postCATEEstimate)

	req := httptest.NewRequest(http.MethodPost, "/cate/estimate", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("want 501 Not Implemented, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if _, ok := body["sprint_2_plan"]; !ok {
		t.Fatalf("response body missing sprint_2_plan field: %s", w.Body.String())
	}
}

func TestGetCATEEstimate_NotFoundReturns404(t *testing.T) {
	r := newTestEngine()
	db := newCATETestDB(t)
	srv := &Server{db: db}
	r.GET("/cate/:id", srv.getCATEEstimate)

	req := httptest.NewRequest(http.MethodGet, "/cate/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCATEEstimate_FoundReturnsRow(t *testing.T) {
	r := newTestEngine()
	db := newCATETestDB(t)
	srv := &Server{db: db}
	r.GET("/cate/:id", srv.getCATEEstimate)

	// Seed one row.
	id := uuid.New()
	rid := uuid.New()
	res := db.DB.Exec(`INSERT INTO cate_estimates
		(id, consolidated_record_id, patient_id, cohort_id, intervention_id, learner_type,
		 point_estimate, ci_lower, ci_upper, horizon_days, overlap_status, computed_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), rid.String(), "P1", "hcf_catalyst_chf", "nurse_phone_48h",
		string(models.LearnerBaselineDiffMeans), 0.15, 0.12, 0.18, 30, string(models.OverlapPass), time.Now().UTC())
	if res.Error != nil {
		t.Fatalf("seed: %v", res.Error)
	}

	req := httptest.NewRequest(http.MethodGet, "/cate/"+id.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var out models.CATEEstimate
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.PointEstimate != 0.15 {
		t.Fatalf("want point=0.15, got %.3f", out.PointEstimate)
	}
	if out.InterventionID != "nurse_phone_48h" {
		t.Fatalf("want intervention=nurse_phone_48h, got %s", out.InterventionID)
	}
}

func TestGetCalibrationSummary_NoMonitorReturns503(t *testing.T) {
	r := newTestEngine()
	srv := &Server{}
	r.GET("/cate/calibration/summary/:cohortId", srv.getCalibrationSummary)

	req := httptest.NewRequest(http.MethodGet, "/cate/calibration/summary/hcf_catalyst_chf?intervention=nurse_phone_48h&horizon=30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", w.Code)
	}
}

func TestGetCalibrationSummary_MissingIntervention_Returns400(t *testing.T) {
	r := newTestEngine()
	db := newCATETestDB(t)
	srv := &Server{db: db, cateMonitor: services.NewCATECalibrationMonitor(db.DB, services.CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})}
	r.GET("/cate/calibration/summary/:cohortId", srv.getCalibrationSummary)

	req := httptest.NewRequest(http.MethodGet, "/cate/calibration/summary/hcf_catalyst_chf", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCalibrationSummary_InsufficientSignalReturnsOK(t *testing.T) {
	r := newTestEngine()
	db := newCATETestDB(t)
	srv := &Server{db: db, cateMonitor: services.NewCATECalibrationMonitor(db.DB, services.CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})}
	r.GET("/cate/calibration/summary/:cohortId", srv.getCalibrationSummary)

	// Empty DB → should return 200 with INSUFFICIENT_SIGNAL status.
	req := httptest.NewRequest(http.MethodGet, "/cate/calibration/summary/hcf_catalyst_chf?intervention=nurse_phone_48h&horizon=30", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "INSUFFICIENT_SIGNAL" {
		t.Fatalf("want INSUFFICIENT_SIGNAL, got %v", body["status"])
	}
}

func TestGetCalibrationSummary_MalformedHorizonReturns400(t *testing.T) {
	r := newTestEngine()
	db := newCATETestDB(t)
	srv := &Server{db: db, cateMonitor: services.NewCATECalibrationMonitor(db.DB, services.CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})}
	r.GET("/cate/calibration/summary/:cohortId", srv.getCalibrationSummary)

	// Malformed horizon value → should reject with 400, not silently default.
	req := httptest.NewRequest(http.MethodGet, "/cate/calibration/summary/hcf_catalyst_chf?intervention=nurse_phone_48h&horizon=abc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for malformed horizon, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetCalibrationSummary_NegativeHorizonReturns400(t *testing.T) {
	r := newTestEngine()
	db := newCATETestDB(t)
	srv := &Server{db: db, cateMonitor: services.NewCATECalibrationMonitor(db.DB, services.CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})}
	r.GET("/cate/calibration/summary/:cohortId", srv.getCalibrationSummary)

	req := httptest.NewRequest(http.MethodGet, "/cate/calibration/summary/hcf_catalyst_chf?intervention=nurse_phone_48h&horizon=-5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 for negative horizon, got %d", w.Code)
	}
}

// Sanity check that the helper compiles.
var _ = fmt.Sprintf("%v", "")
