package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"

	"kb-patient-profile/internal/storage"
)

func setupCapacityRouter(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	v2 := storage.NewV2SubstrateStoreWithDB(db)
	store := storage.NewCapacityAssessmentStore(db, v2)
	h := NewCapacityHandlers(store)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/v2"))
	return r, db
}

func postCapacity(t *testing.T, r *gin.Engine, rid uuid.UUID, body map[string]interface{}, expect int) *httptest.ResponseRecorder {
	t.Helper()
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost,
		"/v2/residents/"+rid.String()+"/capacity",
		bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != expect {
		t.Fatalf("POST: expected status=%d, got %d body=%s", expect, w.Code, w.Body.String())
	}
	return w
}

func TestCapacityHandlers_PostImpairedMedicalReturnsEvent(t *testing.T) {
	r, _ := setupCapacityRouter(t)

	rid := uuid.New()
	body := map[string]interface{}{
		"assessed_at":       time.Now().UTC().Truncate(time.Second),
		"assessor_role_ref": uuid.New(),
		"domain":            models.CapacityDomainMedical,
		"outcome":           models.CapacityOutcomeImpaired,
		"duration":          models.CapacityDurationPermanent,
	}
	w := postCapacity(t, r, rid, body, http.StatusOK)
	var result interfaces.CapacityAssessmentResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Event == nil {
		t.Errorf("expected capacity_change Event in response for impaired+medical")
	} else if result.Event.EventType != models.EventTypeCapacityChange {
		t.Errorf("Event type drift: %s", result.Event.EventType)
	}
	if result.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("expected EvidenceTraceNodeRef in response")
	}
}

func TestCapacityHandlers_PostImpairedFinancialNoEvent(t *testing.T) {
	r, _ := setupCapacityRouter(t)

	rid := uuid.New()
	body := map[string]interface{}{
		"assessed_at":       time.Now().UTC().Truncate(time.Second),
		"assessor_role_ref": uuid.New(),
		"domain":            models.CapacityDomainFinancial,
		"outcome":           models.CapacityOutcomeImpaired,
		"duration":          models.CapacityDurationPermanent,
	}
	w := postCapacity(t, r, rid, body, http.StatusOK)
	var result interfaces.CapacityAssessmentResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Event != nil {
		t.Errorf("expected NO Event for impaired+financial; got %+v", result.Event)
	}
}

func TestCapacityHandlers_GetCurrentByDomainAndAll(t *testing.T) {
	r, _ := setupCapacityRouter(t)

	rid := uuid.New()
	roleRef := uuid.New()
	t0 := time.Now().UTC().Truncate(time.Second)

	// medical (intact)
	postCapacity(t, r, rid, map[string]interface{}{
		"assessed_at": t0, "assessor_role_ref": roleRef,
		"domain":  models.CapacityDomainMedical,
		"outcome": models.CapacityOutcomeIntact, "duration": models.CapacityDurationPermanent,
	}, http.StatusOK)
	// financial (impaired)
	postCapacity(t, r, rid, map[string]interface{}{
		"assessed_at": t0, "assessor_role_ref": roleRef,
		"domain":  models.CapacityDomainFinancial,
		"outcome": models.CapacityOutcomeImpaired, "duration": models.CapacityDurationPermanent,
	}, http.StatusOK)

	// GET current/medical
	req := httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+rid.String()+"/capacity/current/"+models.CapacityDomainMedical, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET current/medical: status=%d body=%s", w.Code, w.Body.String())
	}
	var med models.CapacityAssessment
	_ = json.Unmarshal(w.Body.Bytes(), &med)
	if med.Outcome != models.CapacityOutcomeIntact {
		t.Errorf("medical outcome drift: %s", med.Outcome)
	}

	// GET current (all domains)
	req = httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+rid.String()+"/capacity/current", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET current: status=%d body=%s", w.Code, w.Body.String())
	}
	var all []models.CapacityAssessment
	_ = json.Unmarshal(w.Body.Bytes(), &all)
	if len(all) != 2 {
		t.Errorf("expected 2 current rows; got %d", len(all))
	}
}

func TestCapacityHandlers_GetCurrentMissingReturns404(t *testing.T) {
	r, _ := setupCapacityRouter(t)
	req := httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+uuid.New().String()+"/capacity/current/"+models.CapacityDomainMedical, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Code)
	}
}

func TestCapacityHandlers_PostInvalidDomainReturns400(t *testing.T) {
	r, _ := setupCapacityRouter(t)
	body := map[string]interface{}{
		"assessed_at":       time.Now().UTC(),
		"assessor_role_ref": uuid.New(),
		"domain":            "bogus",
		"outcome":           models.CapacityOutcomeIntact,
		"duration":          models.CapacityDurationPermanent,
	}
	w := postCapacity(t, r, uuid.New(), body, http.StatusBadRequest)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", w.Code)
	}
}

func TestCapacityHandlers_HistoryDescendingOrder(t *testing.T) {
	r, _ := setupCapacityRouter(t)
	rid := uuid.New()
	roleRef := uuid.New()
	t0 := time.Now().UTC().Truncate(time.Second)

	for _, at := range []time.Time{t0.Add(-48 * time.Hour), t0.Add(-24 * time.Hour), t0} {
		postCapacity(t, r, rid, map[string]interface{}{
			"assessed_at": at, "assessor_role_ref": roleRef,
			"domain":  models.CapacityDomainAccommodation,
			"outcome": models.CapacityOutcomeIntact, "duration": models.CapacityDurationPermanent,
		}, http.StatusOK)
	}
	req := httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+rid.String()+"/capacity/history/"+models.CapacityDomainAccommodation, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var hist []models.CapacityAssessment
	_ = json.Unmarshal(w.Body.Bytes(), &hist)
	if len(hist) != 3 {
		t.Fatalf("expected 3 rows; got %d", len(hist))
	}
	if !hist[0].AssessedAt.Equal(t0) {
		t.Errorf("expected newest first; got %v", hist[0].AssessedAt)
	}
}
