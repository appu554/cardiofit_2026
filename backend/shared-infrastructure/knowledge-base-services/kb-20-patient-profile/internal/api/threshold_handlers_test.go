package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// newTestThresholdServer creates a minimal Server with only the fields needed
// for threshold handler tests.
func newTestThresholdServer() *Server {
	gin.SetMode(gin.TestMode)
	logger, _ := zap.NewDevelopment()
	s := &Server{
		Router: gin.New(),
		logger: logger,
	}
	thresholds := s.Router.Group("/api/v1/thresholds")
	{
		thresholds.GET("/labs", s.getLabThresholds)
	}
	return s
}

func TestGetLabThresholds_ReturnsOK(t *testing.T) {
	s := newTestThresholdServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thresholds/labs", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("expected application/json content type, got %q", contentType)
	}
}

func TestGetLabThresholds_ContainsAllAnalytes(t *testing.T) {
	s := newTestThresholdServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thresholds/labs", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var body map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	requiredKeys := []string{
		"creatinine", "potassium", "glucose", "egfr",
		"hba1c", "lactate", "troponin", "wbc", "version",
	}
	for _, key := range requiredKeys {
		if _, ok := body[key]; !ok {
			t.Errorf("missing required key %q in response", key)
		}
	}
}

func TestGetLabThresholds_CreatinineShape(t *testing.T) {
	s := newTestThresholdServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thresholds/labs", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var body struct {
		Creatinine struct {
			PlausibleRange      [2]float64 `json:"plausible_range"`
			NormalRange         [2]float64 `json:"normal_range"`
			AKIStage1Delta48h   float64    `json:"aki_stage1_delta_48h"`
			AKIStage1PctInc     float64    `json:"aki_stage1_pct_increase"`
			AKIStage2Multiplier float64    `json:"aki_stage2_multiplier"`
			AKIStage3Multiplier float64    `json:"aki_stage3_multiplier"`
			AKIStage3Absolute   float64    `json:"aki_stage3_absolute"`
			WorseningSlope      float64    `json:"worsening_slope"`
		} `json:"creatinine"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse creatinine: %v", err)
	}

	cr := body.Creatinine
	if cr.PlausibleRange != [2]float64{0.2, 20.0} {
		t.Errorf("creatinine plausible_range = %v, want [0.2, 20.0]", cr.PlausibleRange)
	}
	if cr.NormalRange != [2]float64{0.6, 1.2} {
		t.Errorf("creatinine normal_range = %v, want [0.6, 1.2]", cr.NormalRange)
	}
	if cr.AKIStage1Delta48h != 0.3 {
		t.Errorf("creatinine aki_stage1_delta_48h = %v, want 0.3", cr.AKIStage1Delta48h)
	}
	if cr.AKIStage3Absolute != 4.0 {
		t.Errorf("creatinine aki_stage3_absolute = %v, want 4.0", cr.AKIStage3Absolute)
	}
}

func TestGetLabThresholds_PotassiumAlertVsHalt(t *testing.T) {
	s := newTestThresholdServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thresholds/labs", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var body struct {
		Potassium struct {
			PlausibleRange [2]float64 `json:"plausible_range"`
			NormalRange    [2]float64 `json:"normal_range"`
			AlertLow       float64    `json:"alert_low"`
			AlertHigh      float64    `json:"alert_high"`
			HaltLow        float64    `json:"halt_low"`
			HaltHigh       float64    `json:"halt_high"`
		} `json:"potassium"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse potassium: %v", err)
	}

	k := body.Potassium
	if k.AlertHigh != 5.5 {
		t.Errorf("potassium alert_high = %v, want 5.5 (Flink threshold)", k.AlertHigh)
	}
	if k.HaltHigh != 6.0 {
		t.Errorf("potassium halt_high = %v, want 6.0 (V-MCU Channel B threshold)", k.HaltHigh)
	}
	if k.AlertHigh >= k.HaltHigh {
		t.Error("alert_high must be strictly less than halt_high")
	}
}

func TestGetLabThresholds_EGFRHaltVsPause(t *testing.T) {
	s := newTestThresholdServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thresholds/labs", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var body struct {
		EGFR struct {
			PlausibleRange [2]float64 `json:"plausible_range"`
			Halt           float64    `json:"halt"`
			Pause          float64    `json:"pause"`
			CKDStage3a     float64    `json:"ckd_stage3a"`
			CKDStage3b     float64    `json:"ckd_stage3b"`
		} `json:"egfr"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse egfr: %v", err)
	}

	e := body.EGFR
	if e.Halt != 15 {
		t.Errorf("egfr halt = %v, want 15", e.Halt)
	}
	if e.Pause != 30 {
		t.Errorf("egfr pause = %v, want 30", e.Pause)
	}
	if e.Halt >= e.Pause {
		t.Error("halt must be strictly less than pause (more severe)")
	}
}

func TestGetLabThresholds_GlucoseSeverityLadder(t *testing.T) {
	s := newTestThresholdServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thresholds/labs", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var body struct {
		Glucose struct {
			PlausibleRange [2]float64 `json:"plausible_range"`
			NormalFasting  [2]float64 `json:"normal_fasting"`
			Hypo           float64    `json:"hypo"`
			SevereHypo     float64    `json:"severe_hypo"`
			SevereHyper    float64    `json:"severe_hyper"`
			CriticalHigh   float64    `json:"critical_high"`
			CVThreshold    float64    `json:"cv_threshold"`
		} `json:"glucose"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse glucose: %v", err)
	}

	g := body.Glucose
	// Severity ladder: severe_hypo < hypo < normal_fasting[0] < normal_fasting[1] < severe_hyper < critical_high
	if g.SevereHypo >= g.Hypo {
		t.Errorf("severe_hypo (%v) must be < hypo (%v)", g.SevereHypo, g.Hypo)
	}
	if g.SevereHyper >= g.CriticalHigh {
		t.Errorf("severe_hyper (%v) must be < critical_high (%v)", g.SevereHyper, g.CriticalHigh)
	}
	if g.CVThreshold != 36.0 {
		t.Errorf("glucose cv_threshold = %v, want 36.0", g.CVThreshold)
	}
}

func TestGetLabThresholds_VersionPresent(t *testing.T) {
	s := newTestThresholdServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/thresholds/labs", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse version: %v", err)
	}
	if body.Version == "" {
		t.Error("version field must not be empty")
	}
}
