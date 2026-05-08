package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// fakeRuntimeProviders implements all five reader interfaces RuntimeHandlers
// depends on. Each method returns a deterministic stub so we can assert
// the JSON response shape without standing up real stores.
type fakeRuntimeProviders struct {
	baselineValue float64
	baselineConf  string
	baselineN     int
}

func (f *fakeRuntimeProviders) GetBaseline(_ uuid.UUID, _ string) (float64, string, int, error) {
	return f.baselineValue, f.baselineConf, f.baselineN, nil
}
func (f *fakeRuntimeProviders) GetActiveConcerns(_ uuid.UUID) ([]string, error) {
	return []string{"post_fall_72h"}, nil
}
func (f *fakeRuntimeProviders) GetCareIntensity(_ uuid.UUID) (string, error) {
	return "active_treatment", nil
}
func (f *fakeRuntimeProviders) GetMedicineUse(_ uuid.UUID) ([]map[string]any, error) {
	return []map[string]any{
		{"amt_code": "AMT_TEST", "display_name": "Test Med", "dose": "5mg"},
	}, nil
}
func (f *fakeRuntimeProviders) GetObservations(_ uuid.UUID, _ string, _ int) ([]map[string]any, error) {
	return []map[string]any{
		{"loinc_code": "potassium", "value": 4.2},
	}, nil
}

func newTestRuntimeRouter(t *testing.T, p *fakeRuntimeProviders) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewRuntimeHandlers(p)
	g := r.Group("/v2")
	h.RegisterRoutes(g)
	return r
}

func TestRuntimeHandlers_GetBaseline(t *testing.T) {
	p := &fakeRuntimeProviders{baselineValue: 4.5, baselineConf: "high", baselineN: 7}
	r := newTestRuntimeRouter(t, p)

	req := httptest.NewRequest("GET",
		"/v2/runtime/baseline?resident_id="+uuid.New().String()+"&type=potassium", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["baseline_value"].(float64) != 4.5 {
		t.Errorf("baseline_value = %v want 4.5", got["baseline_value"])
	}
	if got["baseline_confidence"] != "high" {
		t.Errorf("baseline_confidence = %v want high", got["baseline_confidence"])
	}
	if got["baseline_n_observations"].(float64) != 7 {
		t.Errorf("baseline_n_observations = %v want 7", got["baseline_n_observations"])
	}
}

func TestRuntimeHandlers_GetBaselineMissingResidentID(t *testing.T) {
	r := newTestRuntimeRouter(t, &fakeRuntimeProviders{})
	req := httptest.NewRequest("GET", "/v2/runtime/baseline?type=potassium", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing resident_id; got %d", w.Code)
	}
}

func TestRuntimeHandlers_GetActiveConcerns(t *testing.T) {
	r := newTestRuntimeRouter(t, &fakeRuntimeProviders{})
	req := httptest.NewRequest("GET",
		"/v2/runtime/active-concerns?resident_id="+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var got []string
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 1 || got[0] != "post_fall_72h" {
		t.Errorf("got %v want [post_fall_72h]", got)
	}
}

func TestRuntimeHandlers_GetCareIntensity(t *testing.T) {
	r := newTestRuntimeRouter(t, &fakeRuntimeProviders{})
	req := httptest.NewRequest("GET",
		"/v2/runtime/care-intensity?resident_id="+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["tag"] != "active_treatment" {
		t.Errorf("tag = %v want active_treatment", got["tag"])
	}
}

func TestRuntimeHandlers_GetMedicineUse(t *testing.T) {
	r := newTestRuntimeRouter(t, &fakeRuntimeProviders{})
	req := httptest.NewRequest("GET",
		"/v2/runtime/medicine-use?resident_id="+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var got []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 {
		t.Errorf("expected 1 medicine; got %d", len(got))
	}
}

func TestRuntimeHandlers_GetObservations(t *testing.T) {
	r := newTestRuntimeRouter(t, &fakeRuntimeProviders{})
	req := httptest.NewRequest("GET",
		"/v2/runtime/observations?resident_id="+uuid.New().String()+"&type=potassium&limit=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var got []map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0]["loinc_code"] != "potassium" {
		t.Errorf("got %v", got)
	}
}
