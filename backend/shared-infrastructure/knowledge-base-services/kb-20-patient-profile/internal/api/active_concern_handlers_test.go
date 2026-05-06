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

	"github.com/cardiofit/shared/v2_substrate/models"

	"kb-patient-profile/internal/storage"
)

func setupActiveConcernRouter(t *testing.T) (*gin.Engine, *sql.DB) {
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

	store := storage.NewActiveConcernStore(db)
	h := NewActiveConcernHandlers(store)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/v2"))
	return r, db
}

func TestActiveConcernHandlers_PostListPatch(t *testing.T) {
	r, _ := setupActiveConcernRouter(t)

	rid := uuid.New()
	startedBy := uuid.New()
	started := time.Now().UTC().Truncate(time.Second)
	body := map[string]interface{}{
		"concern_type":           models.ActiveConcernPostFall72h,
		"started_at":             started,
		"started_by_event_ref":   startedBy,
		"expected_resolution_at": started.Add(72 * time.Hour),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost,
		"/v2/residents/"+rid.String()+"/active-concerns",
		bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST: status=%d body=%s", w.Code, w.Body.String())
	}
	var created models.ActiveConcern
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ResidentID != rid {
		t.Errorf("ResidentID drift")
	}
	if created.ResolutionStatus != models.ResolutionStatusOpen {
		t.Errorf("expected open status; got %s", created.ResolutionStatus)
	}

	// LIST
	req2 := httptest.NewRequest(http.MethodGet,
		"/v2/residents/"+rid.String()+"/active-concerns?status=open", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET list: status=%d body=%s", w2.Code, w2.Body.String())
	}
	var listed []models.ActiveConcern
	if err := json.Unmarshal(w2.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != created.ID {
		t.Errorf("expected 1 listed concern matching created")
	}

	// PATCH (resolve)
	patchBody := map[string]interface{}{
		"resolution_status": models.ResolutionStatusResolvedStopCriteria,
		"resolved_at":       time.Now().UTC().Truncate(time.Second),
	}
	patchJSON, _ := json.Marshal(patchBody)
	req3 := httptest.NewRequest(http.MethodPatch,
		"/v2/active-concerns/"+created.ID.String(),
		bytes.NewReader(patchJSON))
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("PATCH: status=%d body=%s", w3.Code, w3.Body.String())
	}
	var resolved models.ActiveConcern
	if err := json.Unmarshal(w3.Body.Bytes(), &resolved); err != nil {
		t.Fatalf("decode patch: %v", err)
	}
	if resolved.ResolutionStatus != models.ResolutionStatusResolvedStopCriteria {
		t.Errorf("expected resolved_stop_criteria; got %s", resolved.ResolutionStatus)
	}

	// PATCH again — must fail (terminal source).
	req4 := httptest.NewRequest(http.MethodPatch,
		"/v2/active-concerns/"+created.ID.String(),
		bytes.NewReader(patchJSON))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code == http.StatusOK {
		t.Errorf("expected non-200 on terminal→terminal transition; got 200")
	}
}

func TestActiveConcernHandlers_PostInvalidType_400(t *testing.T) {
	r, _ := setupActiveConcernRouter(t)
	rid := uuid.New()
	now := time.Now().UTC()
	body := map[string]interface{}{
		"concern_type":           "made_up",
		"started_at":             now,
		"expected_resolution_at": now.Add(time.Hour),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost,
		"/v2/residents/"+rid.String()+"/active-concerns",
		bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid concern_type; got %d", w.Code)
	}
}

func TestActiveConcernHandlers_GetExpiring_PathOnly(t *testing.T) {
	r, _ := setupActiveConcernRouter(t)
	req := httptest.NewRequest(http.MethodGet,
		"/v2/active-concerns/expiring?within=24h", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200; got %d body=%s", w.Code, w.Body.String())
	}
	// Body must be a JSON array (possibly empty).
	var arr []models.ActiveConcern
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Errorf("body not an array: %v body=%s", err, w.Body.String())
	}
}
