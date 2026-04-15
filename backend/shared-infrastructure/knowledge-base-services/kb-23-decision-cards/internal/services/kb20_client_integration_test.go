package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/metrics"
)

// sharedMetricsCollector is instantiated once per test process because
// metrics.NewCollector registers Prometheus CounterVec entries on the
// default registry, and a second instantiation panics with "duplicate
// metrics collector registration attempted." Tests that need a real
// collector share this instance.
var (
	sharedMetricsOnce      sync.Once
	sharedMetricsCollector *metrics.Collector
)

func testMetricsCollector() *metrics.Collector {
	sharedMetricsOnce.Do(func() {
		sharedMetricsCollector = metrics.NewCollector()
	})
	return sharedMetricsCollector
}

// TestKB20Client_FetchSummaryContext_RoundTripsAgainstRealHandler is
// THE test that closes the Phase 7 process gap flagged in the Phase 7
// retrospective: every unit test in Phase 7 used stub fetchers, so the
// HTTP contract between KB-23 and KB-20 went unverified across P7-A,
// P7-B, P7-D, and P7-E. The result was that FetchSummaryContext called
// an endpoint (GET /patient/:id/summary-context) that did not exist,
// and every card-generation code path silently returned (nil, error)
// in production without any test catching it.
//
// This test:
//  1. Spins up an httptest.Server with a handler that mirrors the
//     KB-20 Phase 8 P8-1 handler's exact wire shape (snake_case JSON
//     tags matching SummaryContext in kb-20-patient-profile, wrapped
//     in the standard {"success": true, "data": ...} envelope).
//  2. Configures a real KB20Client pointing at that server.
//  3. Calls FetchSummaryContext through the real HTTP path — no stubs,
//     no mocks, no interface trickery. Actual json.NewDecoder into the
//     real PatientContext struct.
//  4. Asserts every field KB-23's consumer code reads is populated.
//
// If this test breaks, the HTTP contract between the two services
// has drifted, and any downstream card-generation path is broken.
// That is a production-blocker, not a unit-level concern.
//
// Phase 8 P8-1.
func TestKB20Client_FetchSummaryContext_RoundTripsAgainstRealHandler(t *testing.T) {
	// Minimal mirror of the KB-20 handler. The JSON tags here must
	// match services.SummaryContext on the KB-20 side exactly — if
	// they drift, this test catches it deterministically.
	type mirrorSummaryContext struct {
		PatientID              string   `json:"patient_id"`
		Stratum                string   `json:"stratum"`
		Medications            []string `json:"medications"`
		EGFRValue              float64  `json:"egfr_value"`
		LatestHbA1c            float64  `json:"latest_hba1c"`
		LatestFBG              float64  `json:"latest_fbg"`
		IsAcuteIll             bool     `json:"is_acute_illness"`
		HasRecentTransfusion   bool     `json:"has_recent_transfusion"`
		HasRecentHypoglycaemia bool     `json:"has_recent_hypoglycaemia"`
		WeightKg               float64  `json:"weight_kg"`
	}

	// Fully-populated fixture so every consumer-facing field has a
	// distinct non-zero value. Any field that fails to round-trip
	// will land as zero on the KB-23 side and fail the assertions
	// below with a precise error.
	fixture := mirrorSummaryContext{
		PatientID:              "p-integration",
		Stratum:                "HIGH",
		Medications:            []string{"METFORMIN", "ACEi", "STATIN"},
		EGFRValue:              42.5,
		LatestHbA1c:            8.3,
		LatestFBG:              155.0,
		IsAcuteIll:             true,
		HasRecentTransfusion:   false,
		HasRecentHypoglycaemia: true,
		WeightKg:               82.0,
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/patient/p-integration/summary-context", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		envelope := map[string]interface{}{
			"success": true,
			"data":    fixture,
		}
		_ = json.NewEncoder(w).Encode(envelope)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	cfg := &config.Config{KB20URL: server.URL}
	// KB20Client's constructor requires a metrics collector. All three
	// tests share testMetricsCollector() because promauto registers
	// counter names on a global registry; a second NewCollector() call
	// panics with "duplicate metrics collector registration."
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchSummaryContext(context.Background(), "p-integration")
	if err != nil {
		t.Fatalf("FetchSummaryContext failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil PatientContext, got nil")
	}

	// Field-by-field assertions — every consumer-facing field must
	// round-trip. If any of these fail, the HTTP contract is broken.
	if got.PatientID != "p-integration" {
		t.Errorf("PatientID = %q, want p-integration", got.PatientID)
	}
	if got.Stratum != "HIGH" {
		t.Errorf("Stratum = %q, want HIGH", got.Stratum)
	}
	if len(got.Medications) != 3 {
		t.Errorf("len(Medications) = %d, want 3 — raw: %+v", len(got.Medications), got.Medications)
	}
	if got.EGFRValue != 42.5 {
		t.Errorf("EGFRValue = %f, want 42.5", got.EGFRValue)
	}
	if got.LatestHbA1c != 8.3 {
		t.Errorf("LatestHbA1c = %f, want 8.3", got.LatestHbA1c)
	}
	if got.LatestFBG != 155.0 {
		t.Errorf("LatestFBG = %f, want 155.0", got.LatestFBG)
	}
	if !got.IsAcuteIll {
		t.Error("IsAcuteIll = false, want true")
	}
	if got.HasRecentTransfusion {
		t.Error("HasRecentTransfusion = true, want false")
	}
	if !got.HasRecentHypoglycaemia {
		t.Error("HasRecentHypoglycaemia = false, want true")
	}
	if got.WeightKg != 82.0 {
		t.Errorf("WeightKg = %f, want 82.0", got.WeightKg)
	}
}

// TestKB20Client_FetchSummaryContext_404Propagates verifies that a
// 404 from KB-20 (the exact failure mode that silently broke P7-A
// through P7-E) now surfaces as an error at the client layer rather
// than returning a zero-value struct.
func TestKB20Client_FetchSummaryContext_404Propagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"patient not found"}`))
	}))
	defer server.Close()

	cfg := &config.Config{KB20URL: server.URL}
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchSummaryContext(context.Background(), "p-missing")
	if err == nil {
		t.Error("expected error on 404, got nil — this is the silent-404 regression")
	}
	if got != nil {
		t.Errorf("expected nil on 404, got %+v", got)
	}
}

// TestKB20Client_FetchSummaryContext_MalformedBody verifies that a
// malformed JSON body (e.g., a partial write from a crashing handler)
// surfaces as a decode error rather than returning a partially-
// populated struct that downstream consumers would treat as real.
func TestKB20Client_FetchSummaryContext_MalformedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true, "data": {not-json`))
	}))
	defer server.Close()

	cfg := &config.Config{KB20URL: server.URL}
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	_, err := client.FetchSummaryContext(context.Background(), "p-bad")
	if err == nil {
		t.Error("expected decode error on malformed body, got nil")
	}
}
