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
	_ "github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/models"

	"kb-patient-profile/internal/storage"
)

// setupIdentityRouter builds a gin engine wired to a live IdentityStore.
// Skips the surrounding test when KB20_TEST_DATABASE_URL is unset so
// CI-without-DB stays green. Mirrors setupV2Router from
// v2_substrate_handlers_test.go.
func setupIdentityRouter(t *testing.T) (*gin.Engine, *storage.V2SubstrateStore, *storage.IdentityStore, func()) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	v2, err := storage.NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("NewV2SubstrateStore: %v", err)
	}
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		_ = v2.Close()
		t.Fatalf("sql.Open: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		_ = v2.Close()
		t.Fatalf("ping: %v", err)
	}
	idStore := storage.NewIdentityStore(sqlDB, v2)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewIdentityHandlers(idStore).RegisterRoutes(r.Group("/v2/identity"))
	cleanup := func() {
		_ = sqlDB.Close()
		_ = v2.Close()
	}
	return r, v2, idStore, cleanup
}

func TestIdentityHandlers_PostMatch_HighIHI(t *testing.T) {
	r, v2, _, cleanup := setupIdentityRouter(t)
	defer cleanup()

	ihi := "8003608000088001"
	residentID := uuid.New()
	if _, err := v2.UpsertResident(t.Context(), models.Resident{
		ID:         residentID,
		IHI:        ihi,
		GivenName:  "Bertha",
		FamilyName: "Vector",
		DOB:        time.Date(1928, 3, 1, 0, 0, 0, 0, time.UTC),
		Sex:        "female",
		FacilityID: uuid.New(),
		Status:     models.ResidentStatusActive,
	}); err != nil {
		t.Fatalf("seed resident: %v", err)
	}

	body, _ := json.Marshal(identity.IncomingIdentifier{
		IHI:    ihi,
		Source: "handler-test",
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/identity/match", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d body=%s", w.Code, w.Body.String())
	}
	var got matchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Match.Confidence != identity.ConfidenceHigh {
		t.Errorf("Confidence: got %s want HIGH", got.Match.Confidence)
	}
	if got.Match.ResidentRef == nil || *got.Match.ResidentRef != residentID {
		t.Errorf("ResidentRef mismatch")
	}
	if got.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("EvidenceTrace node ref must be set")
	}
	if got.ReviewQueueEntryID != nil {
		t.Errorf("HIGH must not enqueue")
	}
}

func TestIdentityHandlers_PostMatch_BadJSON(t *testing.T) {
	r, _, _, cleanup := setupIdentityRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/v2/identity/match", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on bad json, got %d", w.Code)
	}
}

func TestIdentityHandlers_GetReviewQueue_Pagination(t *testing.T) {
	r, _, store, cleanup := setupIdentityRouter(t)
	defer cleanup()

	// Enqueue one entry via the service-level NONE path.
	res, err := store.MatchAndPersist(t.Context(), identity.IncomingIdentifier{Source: "handler-test"})
	if err != nil {
		t.Fatalf("MatchAndPersist: %v", err)
	}
	if res.ReviewQueueEntryID == nil {
		t.Fatal("expected NONE to enqueue")
	}

	req := httptest.NewRequest(http.MethodGet, "/v2/identity/review-queue?status=pending&limit=10&offset=0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d", w.Code)
	}
	var entries []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Errorf("expected at least one pending entry")
	}
}

func TestIdentityHandlers_GetReviewQueue_BadStatus(t *testing.T) {
	r, _, _, cleanup := setupIdentityRouter(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v2/identity/review-queue?status=garbage", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on bad status, got %d", w.Code)
	}
}

func TestIdentityHandlers_PostResolve_BadID(t *testing.T) {
	r, _, _, cleanup := setupIdentityRouter(t)
	defer cleanup()

	body, _ := json.Marshal(resolveRequest{
		ResolvedResidentRef: uuid.New(),
		ResolvedBy:          uuid.New(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/identity/review/not-a-uuid/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on bad id, got %d", w.Code)
	}
}

func TestIdentityHandlers_PostResolve_NilResolvedRef(t *testing.T) {
	r, _, _, cleanup := setupIdentityRouter(t)
	defer cleanup()

	body, _ := json.Marshal(resolveRequest{
		ResolvedBy: uuid.New(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/identity/review/"+uuid.New().String()+"/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on nil resolved_resident_ref, got %d", w.Code)
	}
}

func TestIdentityHandlers_PostResolve_NotFound(t *testing.T) {
	r, _, _, cleanup := setupIdentityRouter(t)
	defer cleanup()

	body, _ := json.Marshal(resolveRequest{
		ResolvedResidentRef: uuid.New(),
		ResolvedBy:          uuid.New(),
	})
	req := httptest.NewRequest(http.MethodPost, "/v2/identity/review/"+uuid.New().String()+"/resolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown queue id, got %d body=%s", w.Code, w.Body.String())
	}
}
