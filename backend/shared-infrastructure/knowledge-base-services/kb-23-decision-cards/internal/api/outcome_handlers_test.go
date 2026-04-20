package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"kb-23-decision-cards/internal/models"
)

func newTestGinEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestIngestOutcome_SingleSource_ReturnsResolved(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:       "P-ing-001",
		LifecycleID:     &lifecycleID,
		CohortID:        "hcf_catalyst_chf",
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
		IngestedAt:      time.Now(),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	rec, _ := resp["record"].(map[string]interface{})
	if rec == nil {
		t.Fatalf("response missing 'record' object: %s", w.Body.String())
	}
	if rec["reconciliation"] != "RESOLVED" {
		t.Fatalf("expected reconciliation=RESOLVED, got %v", rec["reconciliation"])
	}
}

func TestIngestOutcome_MissingPatientID_Returns400(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	body := models.OutcomeRecord{
		OutcomeType: "READMISSION_30D",
		Source:      string(models.OutcomeSourceHospitalDischarge),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing patient_id, got %d", w.Code)
	}
}

func TestIngestOutcome_InvalidJSON_Returns400(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader([]byte(`{"bad json`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", w.Code)
	}
}

func TestIngestOutcome_IdempotencyKey_DeduplicatesDuplicate(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	idemKey := "feed-msg-abc-123"
	body := models.OutcomeRecord{
		PatientID:       "P-idem-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
		IdempotencyKey:  &idemKey,
	}
	post := func() int {
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if c := post(); c != http.StatusOK {
		t.Fatalf("first POST: expected 200, got %d", c)
	}
	if c := post(); c != http.StatusOK {
		t.Fatalf("duplicate POST: expected 200 (idempotent), got %d", c)
	}

	var count int64
	db.DB.Model(&models.OutcomeRecord{}).Where("idempotency_key = ?", "feed-msg-abc-123").Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 row for idempotency key, got %d", count)
	}
}

func TestIngestOutcome_Transactional_PriorRowsMarkedResolved(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	// Seed a PENDING prior row directly in the DB (simulates an earlier
	// ingest where reconciliation stayed PENDING because min_sources wasn't met).
	lifecycleID := uuid.New()
	prior := models.OutcomeRecord{
		PatientID:       "P-txn-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
		Reconciliation:  string(models.ReconciliationPending),
	}
	if err := db.DB.Create(&prior).Error; err != nil {
		t.Fatalf("seed prior row: %v", err)
	}

	// Ingest a second source; reconciliation should now resolve and the
	// prior row should be marked RESOLVED with ReconciledID pointing at
	// the new authoritative row.
	incoming := models.OutcomeRecord{
		PatientID:       "P-txn-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceClaimsFeed),
	}
	bodyJSON, _ := json.Marshal(incoming)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var all []models.OutcomeRecord
	if err := db.DB.Where("patient_id = ?", "P-txn-001").Find(&all).Error; err != nil {
		t.Fatalf("load all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 rows (authoritative + prior), got %d", len(all))
	}
	// Locate rows by ReconciledID presence — deterministic regardless of insertion order
	// or which source ReconcileOutcomes chose as the base for the authoritative record.
	// The prior row is updated in-place with a non-nil ReconciledID pointing at the
	// new authoritative row; the authoritative row has ReconciledID == nil.
	var priorLoaded, authoritative models.OutcomeRecord
	for _, r := range all {
		if r.ReconciledID != nil {
			priorLoaded = r
		} else {
			authoritative = r
		}
	}
	if priorLoaded.ID == uuid.Nil {
		t.Fatalf("prior row (ReconciledID != nil) not found in result set")
	}
	if authoritative.ID == uuid.Nil {
		t.Fatalf("authoritative row (ReconciledID == nil) not found in result set")
	}
	if priorLoaded.Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("prior row should be RESOLVED, got %s", priorLoaded.Reconciliation)
	}
	if *priorLoaded.ReconciledID != authoritative.ID {
		t.Fatalf("prior ReconciledID should point at authoritative; got %v", priorLoaded.ReconciledID)
	}
}

func TestIngestOutcome_WithoutIdempotencyKey_StillWorks(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:       "P-noidem-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	req2 := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second POST: expected 200, got %d", w2.Code)
	}

	// Verify: without an idempotency key, the two keyless POSTs created 2 distinct rows.
	var count int64
	db.DB.Model(&models.OutcomeRecord{}).Where("patient_id = ?", "P-noidem-001").Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows (no dedup without idempotency key), got %d", count)
	}
}
