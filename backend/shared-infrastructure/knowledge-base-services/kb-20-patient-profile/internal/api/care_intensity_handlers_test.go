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

func setupCareIntensityRouter(t *testing.T) (*gin.Engine, *sql.DB) {
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
	store := storage.NewCareIntensityStore(db, v2)
	h := NewCareIntensityHandlers(store)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/v2"))
	return r, db
}

func TestCareIntensityHandlers_PostThenGetCurrentAndHistory(t *testing.T) {
	r, _ := setupCareIntensityRouter(t)

	rid := uuid.New()
	roleRef := uuid.New()
	t0 := time.Now().UTC().Truncate(time.Second)

	// First POST: active_treatment.
	body1 := map[string]interface{}{
		"tag":                    models.CareIntensityTagActiveTreatment,
		"effective_date":         t0.Add(-72 * time.Hour),
		"documented_by_role_ref": roleRef,
	}
	postCareIntensity(t, r, rid, body1, http.StatusOK)

	// Second POST: transition into palliative.
	body2 := map[string]interface{}{
		"tag":                    models.CareIntensityTagPalliative,
		"effective_date":         t0,
		"documented_by_role_ref": roleRef,
	}
	w2 := postCareIntensity(t, r, rid, body2, http.StatusOK)
	var result interfaces.CareIntensityTransitionResult
	if err := json.Unmarshal(w2.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.CareIntensity == nil || result.CareIntensity.Tag != models.CareIntensityTagPalliative {
		t.Errorf("CareIntensity drift: %+v", result.CareIntensity)
	}
	if result.Event == nil || result.Event.EventType != models.EventTypeCareIntensityTransition {
		t.Errorf("Event drift: %+v", result.Event)
	}
	if len(result.Cascades) != 3 {
		t.Errorf("expected 3 cascades for active→palliative; got %d", len(result.Cascades))
	}

	// GET current.
	req := httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+rid.String()+"/care-intensity/current", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET current: status=%d body=%s", w.Code, w.Body.String())
	}
	var current models.CareIntensity
	if err := json.Unmarshal(w.Body.Bytes(), &current); err != nil {
		t.Fatalf("decode current: %v", err)
	}
	if current.Tag != models.CareIntensityTagPalliative {
		t.Errorf("expected current=palliative; got %s", current.Tag)
	}

	// GET history.
	req = httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+rid.String()+"/care-intensity/history", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET history: status=%d body=%s", w.Code, w.Body.String())
	}
	var history []models.CareIntensity
	if err := json.Unmarshal(w.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 history rows; got %d", len(history))
	}
	if len(history) >= 1 && history[0].Tag != models.CareIntensityTagPalliative {
		t.Errorf("expected newest=palliative; got %s", history[0].Tag)
	}
}

func TestCareIntensityHandlers_GetCurrentMissingReturnsNotFound(t *testing.T) {
	r, _ := setupCareIntensityRouter(t)
	req := httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+uuid.New().String()+"/care-intensity/current", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404; got %d", w.Code)
	}
}

func TestCareIntensityHandlers_PostInvalidTagReturns400(t *testing.T) {
	r, _ := setupCareIntensityRouter(t)
	rid := uuid.New()
	body := map[string]interface{}{
		"tag":                    "active", // legacy short form, rejected
		"effective_date":         time.Now().UTC(),
		"documented_by_role_ref": uuid.New(),
	}
	w := postCareIntensity(t, r, rid, body, http.StatusBadRequest)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400; got %d", w.Code)
	}
}

func postCareIntensity(t *testing.T, r *gin.Engine, rid uuid.UUID, body map[string]interface{}, expect int) *httptest.ResponseRecorder {
	t.Helper()
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost,
		"/v2/residents/"+rid.String()+"/care-intensity",
		bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != expect {
		t.Fatalf("POST: expected status=%d, got %d body=%s", expect, w.Code, w.Body.String())
	}
	return w
}
