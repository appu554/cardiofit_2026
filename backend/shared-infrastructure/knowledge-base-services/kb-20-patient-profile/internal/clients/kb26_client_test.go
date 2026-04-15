package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// TestKB26Client_GetLatestCGMStatus_HappyPath exercises the full
// HTTP round trip against a real httptest.Server that mirrors the
// KB-26 Phase 7 P7-E Milestone 2 cgm-latest endpoint. This is the
// integration-test pattern the Phase 7 retrospective called out as
// the missing layer — one test per cross-service client method,
// running against a real handler rather than stub interfaces.
// Phase 8 P8-3.
func TestKB26Client_GetLatestCGMStatus_HappyPath(t *testing.T) {
	// Response envelope must match KB-26's sendSuccess wrapper
	// exactly: {"success": true, "data": CGMPeriodReport}.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/kb26/cgm-latest/p-integration" {
			http.Error(w, "wrong path: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		envelope := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"patient_id":       "p-integration",
				"period_start":     "2026-04-01T00:00:00Z",
				"period_end":       "2026-04-15T00:00:00Z",
				"tir_pct":          78.5,
				"mean_glucose":     152.0,
				"gri_zone":         "B",
				"confidence_level": "HIGH",
			},
		}
		_ = json.NewEncoder(w).Encode(envelope)
	}))
	defer server.Close()

	client := NewKB26Client(server.URL, zap.NewNop())
	got, err := client.GetLatestCGMStatus(context.Background(), "p-integration")
	if err != nil {
		t.Fatalf("GetLatestCGMStatus: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil snapshot, got nil")
	}

	if got.PatientID != "p-integration" {
		t.Errorf("PatientID = %q, want p-integration", got.PatientID)
	}
	if got.TIRPct != 78.5 {
		t.Errorf("TIRPct = %f, want 78.5", got.TIRPct)
	}
	if got.MeanGlucose != 152.0 {
		t.Errorf("MeanGlucose = %f, want 152.0", got.MeanGlucose)
	}
	if got.GRIZone != "B" {
		t.Errorf("GRIZone = %q, want B", got.GRIZone)
	}
	if got.ConfidenceLevel != "HIGH" {
		t.Errorf("ConfidenceLevel = %q, want HIGH", got.ConfidenceLevel)
	}
	// The period_end timestamp is the anchor for "when was this
	// CGM report generated" — the summary-context service feeds
	// this into the CGMStatusSnapshot.ReportedAt field.
	if got.PeriodEnd.IsZero() {
		t.Error("PeriodEnd should not be zero")
	}
}

// TestKB26Client_GetLatestCGMStatus_404ReturnsNil verifies the
// expected "no CGM data for this patient" path: a 404 from KB-26
// must return (nil, nil) rather than an error, so the summary-context
// service can degrade cleanly to HasCGM=false without logging noise.
// This is the most common real-world response pattern because most
// patients in a CKM cohort do not wear a CGM device.
func TestKB26Client_GetLatestCGMStatus_404ReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"no CGM period report found for patient"}`))
	}))
	defer server.Close()

	client := NewKB26Client(server.URL, zap.NewNop())
	got, err := client.GetLatestCGMStatus(context.Background(), "p-no-cgm")
	if err != nil {
		t.Errorf("expected nil error on 404, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil snapshot on 404, got %+v", got)
	}
}

// TestKB26Client_GetLatestCGMStatus_5xxPropagates verifies that a
// 500 from KB-26 surfaces as an error (not a silent nil), so the
// summary-context service can log the failure at debug level and
// the downstream alert pipeline has visibility into KB-26 outages.
func TestKB26Client_GetLatestCGMStatus_5xxPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"database connection failed"}`))
	}))
	defer server.Close()

	client := NewKB26Client(server.URL, zap.NewNop())
	_, err := client.GetLatestCGMStatus(context.Background(), "p-test")
	if err == nil {
		t.Error("expected error on 500, got nil")
	}
}

// TestKB26Client_GetLatestCGMStatus_MalformedBodyPropagates verifies
// that a malformed response body (partial write from a crashing
// handler, content-type mismatch, etc.) surfaces as a decode error
// rather than returning a partially-populated snapshot downstream
// consumers would treat as real data.
func TestKB26Client_GetLatestCGMStatus_MalformedBodyPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success": true, "data": {not-json`))
	}))
	defer server.Close()

	client := NewKB26Client(server.URL, zap.NewNop())
	_, err := client.GetLatestCGMStatus(context.Background(), "p-bad")
	if err == nil {
		t.Error("expected decode error on malformed body, got nil")
	}
}

// TestKB26Client_GetLatestCGMStatus_HTTPClientError verifies that a
// network-level failure (server not reachable) surfaces as an error
// rather than hanging indefinitely. Uses a closed httptest.Server
// URL to force an immediate connection refused.
func TestKB26Client_GetLatestCGMStatus_HTTPClientError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close() // immediately close — subsequent requests fail

	client := NewKB26Client(server.URL, zap.NewNop())
	_, err := client.GetLatestCGMStatus(context.Background(), "p-unreachable")
	if err == nil {
		t.Error("expected network error on unreachable server, got nil")
	}
}
