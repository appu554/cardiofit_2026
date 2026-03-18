package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// kb26TwinStateJSON builds a JSON payload mimicking the KB-26 API response envelope.
// All Tier 3 fields are encoded as JSONB EstimatedVariable objects.
func kb26TwinStateJSON(t *testing.T, opts kb26TwinOpts) []byte {
	t.Helper()

	type estimatedVariable struct {
		Value          float64 `json:"value"`
		Classification string  `json:"classification"`
		Confidence     float64 `json:"confidence"`
		Method         string  `json:"method"`
	}

	type twinPayload struct {
		ID           string  `json:"id"`
		PatientID    string  `json:"patient_id"`
		StateVersion int     `json:"state_version"`
		UpdatedAt    string  `json:"updated_at"`
		UpdateSource string  `json:"update_source"`

		// Tier 1
		EGFR      *float64 `json:"egfr,omitempty"`
		MAPValue  *float64 `json:"map_value,omitempty"`
		SBP14dMean *float64 `json:"sbp_14d_mean,omitempty"`
		DBP14dMean *float64 `json:"dbp_14d_mean,omitempty"`

		// Tier 2
		VisceralFatProxy    *float64 `json:"visceral_fat_proxy,omitempty"`
		VisceralFatTrend    *string  `json:"visceral_fat_trend,omitempty"`
		RenalSlope          *float64 `json:"renal_slope,omitempty"`
		GlycemicVariability *float64 `json:"glycemic_variability,omitempty"`
		DailySteps7dMean    *float64 `json:"daily_steps_7d_mean,omitempty"`
		RestingHR           *float64 `json:"resting_hr,omitempty"`

		// Tier 3 (JSONB) — use json.RawMessage so nil encodes as JSON null
		InsulinSensitivity   json.RawMessage `json:"insulin_sensitivity,omitempty"`
		HepaticGlucoseOutput json.RawMessage `json:"hepatic_glucose_output,omitempty"`
		MuscleMassProxy      json.RawMessage `json:"muscle_mass_proxy,omitempty"`
	}

	payload := twinPayload{
		ID:           "00000000-0000-0000-0000-000000000001",
		PatientID:    opts.patientID,
		StateVersion: 1,
		UpdatedAt:    opts.updatedAt,
		UpdateSource: "test",
	}

	if opts.egfr != nil {
		payload.EGFR = opts.egfr
	}
	if opts.mapValue != nil {
		payload.MAPValue = opts.mapValue
	}
	if opts.vfProxy != nil {
		payload.VisceralFatProxy = opts.vfProxy
	}
	if opts.vfTrend != "" {
		payload.VisceralFatTrend = &opts.vfTrend
	}
	if opts.renalSlope != nil {
		payload.RenalSlope = opts.renalSlope
	}
	if opts.glycemicVar != nil {
		payload.GlycemicVariability = opts.glycemicVar
	}
	if opts.dailySteps != nil {
		payload.DailySteps7dMean = opts.dailySteps
	}
	if opts.restingHR != nil {
		payload.RestingHR = opts.restingHR
	}

	if !opts.nullIS {
		ev := estimatedVariable{Value: opts.isValue, Confidence: opts.isConf, Classification: "LOW", Method: "HOMA_IR"}
		raw, _ := json.Marshal(ev)
		payload.InsulinSensitivity = json.RawMessage(raw)
	}
	if !opts.nullHGO {
		ev := estimatedVariable{Value: opts.hgoValue, Confidence: opts.hgoConf, Classification: "NORMAL", Method: "MODEL"}
		raw, _ := json.Marshal(ev)
		payload.HepaticGlucoseOutput = json.RawMessage(raw)
	}
	if !opts.nullMM {
		ev := estimatedVariable{Value: opts.mmValue, Confidence: opts.mmConf, Classification: "LOW", Method: "BIA_PROXY"}
		raw, _ := json.Marshal(ev)
		payload.MuscleMassProxy = json.RawMessage(raw)
	}

	// Wrap in success envelope matching KB-26 sendSuccess()
	envelope := map[string]interface{}{
		"success": true,
		"data":    payload,
		"metadata": map[string]interface{}{
			"patient_id":    opts.patientID,
			"state_version": 1,
		},
	}
	out, _ := json.Marshal(envelope)
	return out
}

type kb26TwinOpts struct {
	patientID string
	updatedAt string

	egfr       *float64
	mapValue   *float64
	vfProxy    *float64
	vfTrend    string
	renalSlope *float64
	glycemicVar *float64
	dailySteps  *float64
	restingHR   *float64

	// Tier 3
	nullIS   bool
	isValue  float64
	isConf   float64

	nullHGO  bool
	hgoValue float64
	hgoConf  float64

	nullMM   bool
	mmValue  float64
	mmConf   float64
}

// --- Tests ---

func TestKB26Client_GetTwinState(t *testing.T) {
	egfr := 75.0
	mapVal := 93.3
	vf := 0.82
	trend := "STABLE"
	renal := -0.03
	glyc := 28.5
	steps := 7400.0
	hr := 68.0

	opts := kb26TwinOpts{
		patientID:  "patient-abc-123",
		updatedAt:  "2026-03-17T10:00:00Z",
		egfr:       &egfr,
		mapValue:   &mapVal,
		vfProxy:    &vf,
		vfTrend:    trend,
		renalSlope: &renal,
		glycemicVar: &glyc,
		dailySteps: &steps,
		restingHR:  &hr,
		isValue:    0.45,
		isConf:     0.78,
		hgoValue:   185.0,
		hgoConf:    0.65,
		mmValue:    32.1,
		mmConf:     0.72,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/kb26/twin/patient-abc-123" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(kb26TwinStateJSON(t, opts))
	}))
	defer srv.Close()

	client := NewKB26Client(srv.URL, 5*time.Second, testLogger())
	view, err := client.GetTwinState(context.Background(), "patient-abc-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if view == nil {
		t.Fatal("expected non-nil TwinStateView")
	}

	// IS should be unwrapped from JSONB
	if view.IS.Value != 0.45 {
		t.Errorf("IS.Value: expected 0.45, got %v", view.IS.Value)
	}
	if view.IS.Confidence != 0.78 {
		t.Errorf("IS.Confidence: expected 0.78, got %v", view.IS.Confidence)
	}

	// HGO
	if view.HGO.Value != 185.0 {
		t.Errorf("HGO.Value: expected 185.0, got %v", view.HGO.Value)
	}

	// MM
	if view.MM.Value != 32.1 {
		t.Errorf("MM.Value: expected 32.1, got %v", view.MM.Value)
	}

	// VF direct
	if view.VF != 0.82 {
		t.Errorf("VF: expected 0.82, got %v", view.VF)
	}
	if view.VFTrend != "STABLE" {
		t.Errorf("VFTrend: expected STABLE, got %v", view.VFTrend)
	}

	// VR derived: MAP / 80.0
	expectedVR := mapVal / 80.0
	if abs64(view.VR.Value-expectedVR) > 1e-9 {
		t.Errorf("VR.Value: expected %v, got %v", expectedVR, view.VR.Value)
	}

	// RR derived: eGFR / 120.0
	expectedRR := egfr / 120.0
	if abs64(view.RR.Value-expectedRR) > 1e-9 {
		t.Errorf("RR.Value: expected %v, got %v", expectedRR, view.RR.Value)
	}

	// LastUpdated
	if view.LastUpdated.IsZero() {
		t.Error("LastUpdated should be non-zero")
	}
}

func TestKB26Client_GetTwinState_NilTier3(t *testing.T) {
	opts := kb26TwinOpts{
		patientID: "patient-nil-tier3",
		updatedAt: "2026-03-17T10:00:00Z",
		nullIS:    true,
		nullHGO:   true,
		nullMM:    true,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(kb26TwinStateJSON(t, opts))
	}))
	defer srv.Close()

	client := NewKB26Client(srv.URL, 5*time.Second, testLogger())
	view, err := client.GetTwinState(context.Background(), "patient-nil-tier3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nil Tier 3 → zero EstimatedValue
	if view.IS.Value != 0.0 || view.IS.Confidence != 0.0 {
		t.Errorf("IS should be zero when null, got Value=%v Confidence=%v", view.IS.Value, view.IS.Confidence)
	}
	if view.HGO.Value != 0.0 || view.HGO.Confidence != 0.0 {
		t.Errorf("HGO should be zero when null, got Value=%v Confidence=%v", view.HGO.Value, view.HGO.Confidence)
	}
	if view.MM.Value != 0.0 || view.MM.Confidence != 0.0 {
		t.Errorf("MM should be zero when null, got Value=%v Confidence=%v", view.MM.Value, view.MM.Confidence)
	}

	// VR and RR should be zero when MAP and eGFR are nil
	if view.VR.Value != 0.0 {
		t.Errorf("VR.Value should be 0 when map_value is nil, got %v", view.VR.Value)
	}
	if view.RR.Value != 0.0 {
		t.Errorf("RR.Value should be 0 when egfr is nil, got %v", view.RR.Value)
	}
}

func TestKB26Client_GetVariableHistory(t *testing.T) {
	// Build 10 snapshots with known IS values
	type snapshotPayload struct {
		UpdatedAt          string          `json:"updated_at"`
		InsulinSensitivity json.RawMessage `json:"insulin_sensitivity"`
	}

	type historyItem struct {
		ID        string  `json:"id"`
		PatientID string  `json:"patient_id"`
		UpdatedAt string  `json:"updated_at"`
		InsulinSensitivity json.RawMessage `json:"insulin_sensitivity"`
	}

	snapshots := make([]historyItem, 10)
	for i := 0; i < 10; i++ {
		ev := map[string]interface{}{
			"value":      float64(i) * 0.1,
			"confidence": 0.70,
		}
		raw, _ := json.Marshal(ev)
		snapshots[i] = historyItem{
			ID:                 "id-" + string(rune('0'+i)),
			PatientID:          "patient-history",
			UpdatedAt:          time.Now().Add(-time.Duration(i) * 24 * time.Hour).UTC().Format(time.RFC3339),
			InsulinSensitivity: json.RawMessage(raw),
		}
	}

	envelope := map[string]interface{}{
		"success": true,
		"data":    snapshots,
		"metadata": map[string]interface{}{
			"patient_id": "patient-history",
			"count":      10,
		},
	}
	body, _ := json.Marshal(envelope)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/kb26/twin/patient-history/history" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
	defer srv.Close()

	client := NewKB26Client(srv.URL, 5*time.Second, testLogger())
	points, err := client.GetVariableHistory(context.Background(), "patient-history", "IS", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(points) != 10 {
		t.Fatalf("expected 10 TimeSeriesPoints, got %d", len(points))
	}

	// First snapshot has IS value 0.0, last has 0.9
	if abs64(points[0].Value-0.0) > 1e-9 {
		t.Errorf("points[0].Value: expected 0.0, got %v", points[0].Value)
	}
	if abs64(points[9].Value-0.9) > 1e-6 {
		t.Errorf("points[9].Value: expected 0.9, got %v", points[9].Value)
	}
}

func TestKB26Client_GetTwinState_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than the client timeout to force a deadline error.
		// Use context done to unblock when the test's context expires.
		select {
		case <-r.Context().Done():
		case <-time.After(30 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewKB26Client(srv.URL, 100*time.Millisecond, testLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := client.GetTwinState(ctx, "patient-timeout")
	if err == nil {
		t.Fatal("expected context deadline / timeout error, got nil")
	}
}

func TestKB26Client_VRFallback(t *testing.T) {
	mapVal := 100.0
	opts := kb26TwinOpts{
		patientID: "patient-vr-fallback",
		updatedAt: "2026-03-17T10:00:00Z",
		mapValue:  &mapVal,
		nullIS:    true,
		nullHGO:   true,
		nullMM:    true,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(kb26TwinStateJSON(t, opts))
	}))
	defer srv.Close()

	client := NewKB26Client(srv.URL, 5*time.Second, testLogger())
	view, err := client.GetTwinState(context.Background(), "patient-vr-fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// VR = MAP / 80.0 = 100 / 80 = 1.25
	expectedVR := mapVal / 80.0
	if abs64(view.VR.Value-expectedVR) > 1e-9 {
		t.Errorf("VR fallback: expected %v, got %v", expectedVR, view.VR.Value)
	}
}

func TestKB26Client_RRFallback(t *testing.T) {
	egfr := 90.0
	opts := kb26TwinOpts{
		patientID: "patient-rr-fallback",
		updatedAt: "2026-03-17T10:00:00Z",
		egfr:      &egfr,
		nullIS:    true,
		nullHGO:   true,
		nullMM:    true,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(kb26TwinStateJSON(t, opts))
	}))
	defer srv.Close()

	client := NewKB26Client(srv.URL, 5*time.Second, testLogger())
	view, err := client.GetTwinState(context.Background(), "patient-rr-fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RR = eGFR / 120.0 = 90 / 120 = 0.75
	expectedRR := egfr / 120.0
	if abs64(view.RR.Value-expectedRR) > 1e-9 {
		t.Errorf("RR fallback: expected %v, got %v", expectedRR, view.RR.Value)
	}
}

// abs64 is a small local helper to avoid importing math in tests.
func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
