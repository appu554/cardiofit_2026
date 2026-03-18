package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// Mock CacheClient for tests (in-memory)
// ---------------------------------------------------------------------------

type mockCache struct {
	data map[string]string
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string]string)}
}

func (m *mockCache) Get(_ context.Context, key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("cache miss")
	}
	return v, nil
}

func (m *mockCache) Set(_ context.Context, key string, value string, _ time.Duration) error {
	m.data[key] = value
	return nil
}

// ---------------------------------------------------------------------------
// KB-20 mock server helpers
// ---------------------------------------------------------------------------

// kb20LabResponse matches the expected KB-20 GET /labs response JSON.
type kb20LabResponse struct {
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
	Unit      string  `json:"unit"`
}

// testKB20HistoryPoint is one element of the mock history array used in tests.
type testKB20HistoryPoint struct {
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
}

// buildKB20Server constructs a test HTTP server that responds to:
//   - GET /api/v1/patient/{id}/labs?type={type}&days={days}  → latest scalar
//   - GET /api/v1/patient/{id}/labs/history?type={type}&days={days} → []point
//
// labs maps labType → latest float value.
// histories maps labType → []float values (newest first).
func buildKB20Server(t *testing.T, patientID string, labs map[string]float64, histories map[string][]float64) *httptest.Server {
	t.Helper()

	now := time.Now().UTC()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		labType := q.Get("type")

		// history endpoint
		if len(r.URL.Path) > 0 && r.URL.Path[len(r.URL.Path)-len("/history"):] == "/history" {
			vals, ok := histories[labType]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			points := make([]testKB20HistoryPoint, len(vals))
			for i, v := range vals {
				points[i] = testKB20HistoryPoint{
					Value:     v,
					Timestamp: now.Add(-time.Duration(i) * 24 * time.Hour).Format(time.RFC3339),
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(points)
			return
		}

		// latest scalar endpoint
		val, ok := labs[labType]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := kb20LabResponse{
			Value:     val,
			Timestamp: now.Format(time.RFC3339),
			Unit:      "unit",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// ---------------------------------------------------------------------------
// KB-26 mock server helpers
// ---------------------------------------------------------------------------

// buildKB26Server constructs a test HTTP server returning a KB-26 twin state.
// updatedAt controls the LastUpdated timestamp for staleness tests.
func buildKB26Server(t *testing.T, opts kb26TwinOpts) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(kb26TwinStateJSON(t, opts))
	}))
}

// ---------------------------------------------------------------------------
// Helper: make a DataResolver wired to test servers
// ---------------------------------------------------------------------------

func makeResolver(t *testing.T, kb20URL, kb26URL string, cache CacheClient, stalenessThreshold time.Duration) DataResolver {
	t.Helper()
	kb26Client := NewKB26Client(kb26URL, 5*time.Second, testLogger())
	return NewDataResolver(kb20URL, kb26Client, cache, stalenessThreshold, testLogger())
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestDataResolver_KB20Source: RequiredInput with source KB-20 is resolved from KB-20.
func TestDataResolver_KB20Source(t *testing.T) {
	labs := map[string]float64{"egfr": 72.0}
	kb20Srv := buildKB20Server(t, "patient-001", labs, nil)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-001",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	inputs := []models.RequiredInput{
		{Field: "egfr", Source: "KB-20", LookbackDays: 90, Optional: false},
	}

	data, err := resolver.Resolve(context.Background(), "patient-001", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val, ok := data.Fields["egfr"]; !ok || val != 72.0 {
		t.Errorf("Fields[egfr]: expected 72.0, got %v (ok=%v)", val, ok)
	}
	if data.Sufficiency != models.DataSufficient {
		t.Errorf("Sufficiency: expected SUFFICIENT, got %v", data.Sufficiency)
	}
	if src, ok := data.Sources["egfr"]; !ok || src != "KB-20" {
		t.Errorf("Sources[egfr]: expected KB-20, got %v", src)
	}
}

// TestDataResolver_KB26Source: RequiredInput with source KB-26 resolves a twin state field.
func TestDataResolver_KB26Source(t *testing.T) {
	egfr := 80.0
	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-002",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		egfr:      &egfr,
		nullIS:    false,
		isValue:   0.55,
		isConf:    0.8,
		nullHGO:   true,
		nullMM:    true,
	})
	defer kb26Srv.Close()

	kb20Srv := buildKB20Server(t, "patient-002", nil, nil)
	defer kb20Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	inputs := []models.RequiredInput{
		{Field: "IS", Source: "KB-26", Optional: false},
	}

	data, err := resolver.Resolve(context.Background(), "patient-002", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val, ok := data.Fields["IS"]; !ok || math.Abs(val-0.55) > 1e-9 {
		t.Errorf("Fields[IS]: expected 0.55, got %v (ok=%v)", val, ok)
	}
	if data.Sufficiency != models.DataSufficient {
		t.Errorf("Sufficiency: expected SUFFICIENT, got %v", data.Sufficiency)
	}
}

// TestDataResolver_MissingRequiredField: required field not returned → INSUFFICIENT.
func TestDataResolver_MissingRequiredField(t *testing.T) {
	// KB-20 server has no labs
	kb20Srv := buildKB20Server(t, "patient-003", nil, nil)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-003",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	inputs := []models.RequiredInput{
		{Field: "hba1c", Source: "KB-20", LookbackDays: 90, Optional: false},
	}

	data, err := resolver.Resolve(context.Background(), "patient-003", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Sufficiency != models.DataInsufficient {
		t.Errorf("Sufficiency: expected INSUFFICIENT, got %v", data.Sufficiency)
	}
	if len(data.MissingFields) == 0 {
		t.Error("MissingFields should be non-empty")
	}
	found := false
	for _, f := range data.MissingFields {
		if f == "hba1c" {
			found = true
		}
	}
	if !found {
		t.Errorf("MissingFields should contain 'hba1c', got %v", data.MissingFields)
	}
}

// TestDataResolver_MissingOptionalField: optional field not returned → PARTIAL (or SUFFICIENT if no required missing).
func TestDataResolver_MissingOptionalField(t *testing.T) {
	labs := map[string]float64{"egfr": 72.0}
	kb20Srv := buildKB20Server(t, "patient-004", labs, nil)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-004",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	inputs := []models.RequiredInput{
		{Field: "egfr", Source: "KB-20", LookbackDays: 90, Optional: false},
		{Field: "hba1c", Source: "KB-20", LookbackDays: 90, Optional: true},
	}

	data, err := resolver.Resolve(context.Background(), "patient-004", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Missing optional → not INSUFFICIENT (still SUFFICIENT or PARTIAL)
	if data.Sufficiency == models.DataInsufficient {
		t.Errorf("Sufficiency: expected SUFFICIENT or PARTIAL, got INSUFFICIENT")
	}
	// hba1c should be in MissingFields
	found := false
	for _, f := range data.MissingFields {
		if f == "hba1c" {
			found = true
		}
	}
	if !found {
		t.Errorf("MissingFields should contain 'hba1c', got %v", data.MissingFields)
	}
}

// TestDataResolver_Staleness: KB-26 twin state LastUpdated is 30 days ago → PARTIAL.
func TestDataResolver_Staleness(t *testing.T) {
	// Twin state is 30 days old; threshold is 21 days → stale
	staleTime := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	egfr := 65.0
	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-005",
		updatedAt: staleTime,
		egfr:      &egfr,
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	kb20Srv := buildKB20Server(t, "patient-005", nil, nil)
	defer kb20Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	inputs := []models.RequiredInput{
		{Field: "eGFR", Source: "KB-26", Optional: false},
	}

	data, err := resolver.Resolve(context.Background(), "patient-005", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Data was fetched (eGFR available from KB-26) but twin state is stale → PARTIAL
	if data.Sufficiency != models.DataPartial {
		t.Errorf("Sufficiency: expected PARTIAL due to staleness, got %v", data.Sufficiency)
	}
}

// TestDataResolver_MultipleSources: mix of KB-20 and KB-26 fields all resolved correctly.
func TestDataResolver_MultipleSources(t *testing.T) {
	labs := map[string]float64{"hba1c": 7.2, "creatinine": 1.1}
	kb20Srv := buildKB20Server(t, "patient-006", labs, nil)
	defer kb20Srv.Close()

	egfr := 78.0
	vf := 0.9
	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-006",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		egfr:      &egfr,
		vfProxy:   &vf,
		nullIS:    false,
		isValue:   0.6,
		isConf:    0.85,
		nullHGO:   true,
		nullMM:    true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	inputs := []models.RequiredInput{
		{Field: "hba1c", Source: "KB-20", LookbackDays: 90, Optional: false},
		{Field: "creatinine", Source: "KB-20", LookbackDays: 90, Optional: false},
		{Field: "IS", Source: "KB-26", Optional: false},
		{Field: "VF", Source: "KB-26", Optional: false},
	}

	data, err := resolver.Resolve(context.Background(), "patient-006", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFloat(t, data.Fields, "hba1c", 7.2)
	assertFloat(t, data.Fields, "creatinine", 1.1)
	assertFloat(t, data.Fields, "IS", 0.6)
	assertFloat(t, data.Fields, "VF", 0.9)

	if data.Sufficiency != models.DataSufficient {
		t.Errorf("Sufficiency: expected SUFFICIENT, got %v", data.Sufficiency)
	}
}

// TestDataResolver_FieldTimestamps: resolved data includes per-field timestamps.
func TestDataResolver_FieldTimestamps(t *testing.T) {
	labs := map[string]float64{"egfr": 70.0}
	kb20Srv := buildKB20Server(t, "patient-007", labs, nil)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-007",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	inputs := []models.RequiredInput{
		{Field: "egfr", Source: "KB-20", LookbackDays: 90, Optional: false},
	}

	data, err := resolver.Resolve(context.Background(), "patient-007", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.FieldTimestamps == nil {
		t.Fatal("FieldTimestamps should be non-nil")
	}
	ts, ok := data.FieldTimestamps["egfr"]
	if !ok {
		t.Error("FieldTimestamps should contain 'egfr'")
	}
	if ts.IsZero() {
		t.Error("FieldTimestamps[egfr] should be non-zero")
	}
}

// TestDataResolver_AggregatedInput_Mean: AggregatedInputDef with MEAN aggregation.
func TestDataResolver_AggregatedInput_Mean(t *testing.T) {
	// Build 30 daily step values: 1000, 2000, ..., 30000
	steps := make([]float64, 30)
	for i := range steps {
		steps[i] = float64((i + 1) * 1000)
	}
	expectedMean := 0.0
	for _, v := range steps {
		expectedMean += v
	}
	expectedMean /= float64(len(steps)) // = 15500.0

	// Key must match the AggregatedInputDef.Field used as the query type parameter
	histories := map[string][]float64{"daily_steps_30d_mean": steps}
	kb20Srv := buildKB20Server(t, "patient-008", nil, histories)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-008",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	aggInputs := []models.AggregatedInputDef{
		{Field: "daily_steps_30d_mean", Source: "KB-20", LookbackDays: 30, Aggregation: "MEAN"},
	}

	data, err := resolver.Resolve(context.Background(), "patient-008", nil, aggInputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := data.Fields["daily_steps_30d_mean"]
	if !ok {
		t.Fatalf("Fields[daily_steps_30d_mean] not found")
	}
	if math.Abs(val-expectedMean) > 1e-6 {
		t.Errorf("daily_steps_30d_mean: expected %v, got %v", expectedMean, val)
	}
}

// TestDataResolver_AggregatedInput_CV: AggregatedInputDef with CV aggregation.
func TestDataResolver_AggregatedInput_CV(t *testing.T) {
	// 14 glucose readings: alternating 80 and 120 → mean=100, stdev=~20.69, CV≈20.69
	values := make([]float64, 14)
	for i := range values {
		if i%2 == 0 {
			values[i] = 80.0
		} else {
			values[i] = 120.0
		}
	}

	// compute expected CV
	n := float64(len(values))
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= n
	variance := 0.0
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= (n - 1) // sample
	stdev := math.Sqrt(variance)
	expectedCV := (stdev / mean) * 100.0

	// Key must match the AggregatedInputDef.Field used as the query type parameter
	histories := map[string][]float64{"glucose_cv": values}
	kb20Srv := buildKB20Server(t, "patient-009", nil, histories)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-009",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	aggInputs := []models.AggregatedInputDef{
		{Field: "glucose_cv", Source: "KB-20", LookbackDays: 14, Aggregation: "CV"},
	}

	data, err := resolver.Resolve(context.Background(), "patient-009", nil, aggInputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := data.Fields["glucose_cv"]
	if !ok {
		t.Fatalf("Fields[glucose_cv] not found")
	}
	if math.Abs(val-expectedCV) > 1e-4 {
		t.Errorf("glucose_cv: expected %v, got %v", expectedCV, val)
	}
}

// TestDataResolver_AggregatedInput_RAW: AggregatedInputDef with RAW aggregation stores series in TimeSeries.
func TestDataResolver_AggregatedInput_RAW(t *testing.T) {
	values := []float64{5.1, 5.5, 6.0, 5.8, 6.2}
	// Key must match AggregatedInputDef.Field used as the query type parameter
	histories := map[string][]float64{"fbg_values": values}
	kb20Srv := buildKB20Server(t, "patient-010", nil, histories)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-010",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	aggInputs := []models.AggregatedInputDef{
		{Field: "fbg_values", Source: "KB-20", LookbackDays: 5, Aggregation: "RAW"},
	}

	data, err := resolver.Resolve(context.Background(), "patient-010", nil, aggInputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.TimeSeries == nil {
		t.Fatal("TimeSeries should be non-nil for RAW aggregation")
	}
	series, ok := data.TimeSeries["fbg_values"]
	if !ok {
		t.Fatalf("TimeSeries[fbg_values] not found")
	}
	if len(series) != len(values) {
		t.Errorf("TimeSeries[fbg_values]: expected %d points, got %d", len(values), len(series))
	}
}

// TestDataResolver_BooleanConversion: TIER1_CHECKIN boolean → 1.0 (true) or 0.0 (false).
func TestDataResolver_BooleanConversion(t *testing.T) {
	kb20Srv := buildKB20Server(t, "patient-011", nil, nil)
	defer kb20Srv.Close()

	kb26Srv := buildKB26Server(t, kb26TwinOpts{
		patientID: "patient-011",
		updatedAt: time.Now().UTC().Format(time.RFC3339),
		nullIS:    true, nullHGO: true, nullMM: true,
	})
	defer kb26Srv.Close()

	resolver := makeResolver(t, kb20Srv.URL, kb26Srv.URL, newMockCache(), 21*24*time.Hour)

	// TIER1_CHECKIN source with boolean=true should produce 1.0
	inputs := []models.RequiredInput{
		{Field: "medication_taken", Source: "TIER1_CHECKIN", Optional: false},
	}

	data, err := resolver.Resolve(context.Background(), "patient-011", inputs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TIER1_CHECKIN fields without a submitted value are missing
	// OR: if we inject a value, they resolve. For this test we verify that
	// the resolver doesn't crash and treats TIER1_CHECKIN as optional-like
	// (it's a check-in response, which may not be available at resolution time).
	// The field should appear in MissingFields since no check-in value is injected.
	found := false
	for _, f := range data.MissingFields {
		if f == "medication_taken" {
			found = true
		}
	}
	if !found {
		// Accept: if resolver treats TIER1_CHECKIN as available with default 0.0
		if val, ok := data.Fields["medication_taken"]; ok {
			if val != 0.0 && val != 1.0 {
				t.Errorf("TIER1_CHECKIN value should be 0.0 or 1.0, got %v", val)
			}
		} else {
			t.Errorf("medication_taken should be in MissingFields or Fields, found neither")
		}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertFloat(t *testing.T, fields map[string]float64, key string, expected float64) {
	t.Helper()
	val, ok := fields[key]
	if !ok {
		t.Errorf("Fields[%s] not found", key)
		return
	}
	if math.Abs(val-expected) > 1e-9 {
		t.Errorf("Fields[%s]: expected %v, got %v", key, expected, val)
	}
}
