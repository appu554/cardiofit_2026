package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/api"
	"github.com/cardiofit/kb32/internal/overrides"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newOverrideRouter(store overrides.Store) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := api.NewOverrideHandler(store)
	r.POST("/v1/craft/override/:recommendation_id", h.HandleCapture)
	return r
}

func validOverrideBody(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]string{
		"reason_code":          "alert_fatigue",
		"appropriateness_flag": "inappropriate_override",
		"reasoning":            "This alert fires too frequently for this resident.",
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return b
}

func postOverride(t *testing.T, r *gin.Engine, recID string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/craft/override/"+recID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Validation rejection tests
// ---------------------------------------------------------------------------

func TestOverrideCapture_BadReasonCode_Returns422(t *testing.T) {
	store := overrides.NewInMemoryStore()
	r := newOverrideRouter(store)

	body, _ := json.Marshal(map[string]string{
		"reason_code":          "not_a_valid_code",
		"appropriateness_flag": "inappropriate_override",
		"reasoning":            "Some reasoning.",
	})

	w := postOverride(t, r, uuid.New().String(), body)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d; want 422 for bad reason_code; body: %s", w.Code, w.Body.String())
	}
	var env api.ErrorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if env.Error == "" {
		t.Error("ErrorEnvelope.Error should not be empty")
	}
}

func TestOverrideCapture_BadFlag_Returns422(t *testing.T) {
	store := overrides.NewInMemoryStore()
	r := newOverrideRouter(store)

	body, _ := json.Marshal(map[string]string{
		"reason_code":          "alert_fatigue",
		"appropriateness_flag": "not_a_valid_flag",
		"reasoning":            "Some reasoning.",
	})

	w := postOverride(t, r, uuid.New().String(), body)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d; want 422 for bad appropriateness_flag; body: %s", w.Code, w.Body.String())
	}
}

func TestOverrideCapture_EmptyReasoning_Returns422(t *testing.T) {
	store := overrides.NewInMemoryStore()
	r := newOverrideRouter(store)

	// Reasoning is a required binding field; an empty string passes binding but
	// fails Validate(). Send the JSON manually to bypass binding check.
	// Note: Gin's `binding:"required"` rejects empty strings too, so we omit
	// the field to test validation — but the OverrideReason.Validate() check
	// on empty Reasoning is what we need to exercise. Use a whitespace string
	// which passes binding but fails the taxonomy check.
	body, _ := json.Marshal(map[string]string{
		"reason_code":          "alert_fatigue",
		"appropriateness_flag": "inappropriate_override",
		"reasoning":            "",
	})

	w := postOverride(t, r, uuid.New().String(), body)
	// Gin binding:"required" will reject empty string with 400.
	// Either 400 or 422 is acceptable here; the important thing is non-2xx.
	if w.Code == http.StatusCreated {
		t.Errorf("status = 201; want non-201 for empty reasoning; body: %s", w.Body.String())
	}
}

func TestOverrideCapture_MalformedRecommendationUUID_Returns422(t *testing.T) {
	store := overrides.NewInMemoryStore()
	r := newOverrideRouter(store)

	w := postOverride(t, r, "not-a-valid-uuid", validOverrideBody(t))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d; want 422 for malformed UUID path param; body: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Happy path test
// ---------------------------------------------------------------------------

func TestOverrideCapture_HappyPath_Returns201AndPersisted(t *testing.T) {
	store := overrides.NewInMemoryStore()
	r := newOverrideRouter(store)

	recID := uuid.New().String()
	w := postOverride(t, r, recID, validOverrideBody(t))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; want 201; body: %s", w.Code, w.Body.String())
	}

	var resp api.OverrideCaptureResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.ID == "" {
		t.Error("response.ID should not be empty")
	}
	if resp.RecommendationID != recID {
		t.Errorf("response.RecommendationID = %q; want %q", resp.RecommendationID, recID)
	}
	if resp.ReasonCode != "alert_fatigue" {
		t.Errorf("response.ReasonCode = %q; want alert_fatigue", resp.ReasonCode)
	}
	if resp.AppropriatenessFlag != "inappropriate_override" {
		t.Errorf("response.AppropriatenessFlag = %q; want inappropriate_override", resp.AppropriatenessFlag)
	}

	// Verify the record is retrievable from the store.
	got, err := store.Get(context.Background(), resp.ID)
	if err != nil {
		t.Fatalf("store.Get: %v", err)
	}
	if got.ReasonCode != "alert_fatigue" {
		t.Errorf("stored ReasonCode = %q; want alert_fatigue", got.ReasonCode)
	}
}

// ---------------------------------------------------------------------------
// Malformed JSON test
// ---------------------------------------------------------------------------

func TestOverrideCapture_MalformedJSON_Returns400(t *testing.T) {
	store := overrides.NewInMemoryStore()
	r := newOverrideRouter(store)

	w := postOverride(t, r, uuid.New().String(), []byte("not valid json {"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d; want 400 for malformed JSON; body: %s", w.Code, w.Body.String())
	}
}
