package api

import (
	"bytes"
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

// setupV2Router builds a gin engine wired to a live kb-20 store. Skips the
// surrounding test when KB20_TEST_DATABASE_URL is unset.
func setupV2Router(t *testing.T) *gin.Engine {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set")
	}
	store, err := storage.NewV2SubstrateStore(dsn)
	if err != nil {
		t.Fatalf("NewV2SubstrateStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	h := NewV2SubstrateHandlers(store)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h.RegisterRoutes(r.Group("/v2"))
	return r
}

func TestPOSTResidentRoundTrip(t *testing.T) {
	r := setupV2Router(t)
	in := models.Resident{
		ID:            uuid.New(),
		GivenName:     "Margaret",
		FamilyName:    "Brown",
		DOB:           time.Date(1938, 4, 12, 0, 0, 0, 0, time.UTC),
		Sex:           "female",
		FacilityID:    uuid.New(),
		CareIntensity: models.CareIntensityActive,
		Status:        models.ResidentStatusActive,
	}
	body, _ := json.Marshal(in)
	req := httptest.NewRequest(http.MethodPost, "/v2/residents", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /v2/residents: status=%d body=%s", w.Code, w.Body.String())
	}

	var out models.Resident
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal POST resp: %v", err)
	}
	if out.GivenName != in.GivenName {
		t.Errorf("GivenName mismatch: got %q want %q", out.GivenName, in.GivenName)
	}

	// GET round-trip.
	req2 := httptest.NewRequest(http.MethodGet, "/v2/residents/"+in.ID.String(), nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("GET /v2/residents/{id}: status=%d body=%s", w2.Code, w2.Body.String())
	}
}
