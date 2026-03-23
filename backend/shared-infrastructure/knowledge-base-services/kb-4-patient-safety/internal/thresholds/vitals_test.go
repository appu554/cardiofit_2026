package thresholds_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"kb-patient-safety/internal/thresholds"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/v1")
	v1.GET("/thresholds/vitals", thresholds.HandleGetVitalThresholds)
	v1.GET("/thresholds/early-warning-scores", thresholds.HandleGetEarlyWarningScores)
	return r
}

// ---------------------------------------------------------------------------
// GET /v1/thresholds/vitals
// ---------------------------------------------------------------------------

func TestGetVitalThresholds_Status200(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/vitals", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetVitalThresholds_ContainsAllVitalSigns(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/vitals", nil)
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	required := []string{
		"heart_rate", "systolic_bp", "diastolic_bp",
		"spo2", "respiratory_rate", "temperature", "version",
	}
	for _, key := range required {
		if _, ok := body[key]; !ok {
			t.Errorf("missing required key %q in response", key)
		}
	}
}

func TestGetVitalThresholds_HeartRateValues(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/vitals", nil)
	r.ServeHTTP(w, req)

	var body struct {
		HeartRate struct {
			BradycardiaSevere   float64 `json:"bradycardia_severe"`
			BradycardiaModerate float64 `json:"bradycardia_moderate"`
			NormalLow           float64 `json:"normal_low"`
			NormalHigh          float64 `json:"normal_high"`
			TachycardiaModerate float64 `json:"tachycardia_moderate"`
			TachycardiaSevere   float64 `json:"tachycardia_severe"`
		} `json:"heart_rate"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	hr := body.HeartRate
	if hr.BradycardiaSevere != 40 {
		t.Errorf("bradycardia_severe: want 40, got %v", hr.BradycardiaSevere)
	}
	if hr.NormalLow != 60 {
		t.Errorf("normal_low: want 60, got %v", hr.NormalLow)
	}
	if hr.NormalHigh != 100 {
		t.Errorf("normal_high: want 100, got %v", hr.NormalHigh)
	}
	if hr.TachycardiaSevere != 120 {
		t.Errorf("tachycardia_severe: want 120, got %v", hr.TachycardiaSevere)
	}
}

func TestGetVitalThresholds_TemperatureFloats(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/vitals", nil)
	r.ServeHTTP(w, req)

	var body struct {
		Temperature struct {
			Hypothermia float64 `json:"hypothermia"`
			NormalLow   float64 `json:"normal_low"`
			NormalHigh  float64 `json:"normal_high"`
			HighFever   float64 `json:"high_fever"`
		} `json:"temperature"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if body.Temperature.Hypothermia != 35.0 {
		t.Errorf("hypothermia: want 35.0, got %v", body.Temperature.Hypothermia)
	}
	if body.Temperature.NormalLow != 36.1 {
		t.Errorf("normal_low: want 36.1, got %v", body.Temperature.NormalLow)
	}
	if body.Temperature.HighFever != 39.5 {
		t.Errorf("high_fever: want 39.5, got %v", body.Temperature.HighFever)
	}
}

func TestGetVitalThresholds_VersionPresent(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/vitals", nil)
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)

	v, ok := body["version"]
	if !ok {
		t.Fatal("missing version field")
	}
	vs, ok := v.(string)
	if !ok || vs == "" {
		t.Fatal("version must be a non-empty string")
	}
}

// ---------------------------------------------------------------------------
// GET /v1/thresholds/early-warning-scores
// ---------------------------------------------------------------------------

func TestGetEarlyWarningScores_Status200(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/early-warning-scores", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetEarlyWarningScores_ContainsNEWS2AndMEWS(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/early-warning-scores", nil)
	r.ServeHTTP(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	for _, key := range []string{"news2", "mews", "version"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing required key %q", key)
		}
	}
}

func TestGetEarlyWarningScores_NEWS2Parameters(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/early-warning-scores", nil)
	r.ServeHTTP(w, req)

	var body struct {
		NEWS2 struct {
			RespiratoryRate []thresholds.ScoreBand `json:"respiratory_rate"`
			SpO2Scale1      []thresholds.ScoreBand `json:"spo2_scale1"`
			SpO2Scale2      []thresholds.ScoreBand `json:"spo2_scale2"`
			SystolicBP      []thresholds.ScoreBand `json:"systolic_bp"`
			HeartRate       []thresholds.ScoreBand `json:"heart_rate"`
			Temperature     []thresholds.ScoreBand `json:"temperature"`
			Consciousness   map[string]int         `json:"consciousness"`
			SupplementalO2  map[string]int         `json:"supplemental_o2"`
			Thresholds      map[string]interface{} `json:"thresholds"`
		} `json:"news2"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	n := body.NEWS2

	// Check respiratory_rate has 5 bands
	if len(n.RespiratoryRate) != 5 {
		t.Errorf("NEWS2 respiratory_rate: want 5 bands, got %d", len(n.RespiratoryRate))
	}

	// Verify first respiratory band
	if len(n.RespiratoryRate) > 0 {
		first := n.RespiratoryRate[0]
		if first.Min != 0 || first.Max != 8 || first.Points != 3 {
			t.Errorf("NEWS2 respiratory_rate[0]: want {0,8,3}, got {%v,%v,%v}", first.Min, first.Max, first.Points)
		}
	}

	// Check spo2_scale2 is present (on oxygen)
	if len(n.SpO2Scale2) != 4 {
		t.Errorf("NEWS2 spo2_scale2: want 4 bands, got %d", len(n.SpO2Scale2))
	}

	// Verify consciousness scores
	if n.Consciousness["alert"] != 0 {
		t.Errorf("NEWS2 consciousness alert: want 0, got %d", n.Consciousness["alert"])
	}
	if n.Consciousness["unresponsive"] != 3 {
		t.Errorf("NEWS2 consciousness unresponsive: want 3, got %d", n.Consciousness["unresponsive"])
	}

	// Verify thresholds
	if n.Thresholds["critical"] != float64(7) {
		t.Errorf("NEWS2 critical threshold: want 7, got %v", n.Thresholds["critical"])
	}
}

func TestGetEarlyWarningScores_MEWSParameters(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/early-warning-scores", nil)
	r.ServeHTTP(w, req)

	var body struct {
		MEWS struct {
			RespiratoryRate []thresholds.ScoreBand `json:"respiratory_rate"`
			HeartRate       []thresholds.ScoreBand `json:"heart_rate"`
			SystolicBP      []thresholds.ScoreBand `json:"systolic_bp"`
			Temperature     []thresholds.ScoreBand `json:"temperature"`
			Consciousness   map[string]int         `json:"consciousness"`
			Thresholds      map[string]interface{} `json:"thresholds"`
		} `json:"mews"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	m := body.MEWS

	// MEWS respiratory_rate has 5 bands
	if len(m.RespiratoryRate) != 5 {
		t.Errorf("MEWS respiratory_rate: want 5 bands, got %d", len(m.RespiratoryRate))
	}

	// MEWS consciousness has 4 levels
	if len(m.Consciousness) != 4 {
		t.Errorf("MEWS consciousness: want 4 levels, got %d", len(m.Consciousness))
	}

	// MEWS thresholds
	if m.Thresholds["critical"] != float64(5) {
		t.Errorf("MEWS critical threshold: want 5, got %v", m.Thresholds["critical"])
	}
	if m.Thresholds["high"] != float64(3) {
		t.Errorf("MEWS high threshold: want 3, got %v", m.Thresholds["high"])
	}
}

func TestGetEarlyWarningScores_NEWS2SupplementalO2(t *testing.T) {
	r := setupRouter()
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/thresholds/early-warning-scores", nil)
	r.ServeHTTP(w, req)

	var body struct {
		NEWS2 struct {
			SupplementalO2 map[string]int `json:"supplemental_o2"`
		} `json:"news2"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if body.NEWS2.SupplementalO2["on_oxygen"] != 2 {
		t.Errorf("NEWS2 supplemental_o2 on_oxygen: want 2, got %d", body.NEWS2.SupplementalO2["on_oxygen"])
	}
}
