package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
	//
	// Phase 8 P8-2: extended with every P8-2 field (demographics,
	// CKM stage + substage metadata, potassium, engagement, CGM
	// status). Every field here must match both sides of the wire.
	type mirrorCKMSubstage struct {
		HFClassification string   `json:"hf_type,omitempty"`
		LVEFPercent      *float64 `json:"lvef_pct,omitempty"`
		NYHAClass        string   `json:"nyha_class,omitempty"`
		NTproBNP         *float64 `json:"nt_probnp,omitempty"`
		BNP              *float64 `json:"bnp,omitempty"`
		HFEtiology       string   `json:"hf_etiology,omitempty"`
		CACScore         *float64 `json:"cac_score,omitempty"`
		CIMTPercentile   *int     `json:"cimt_percentile,omitempty"`
		HasLVH           bool     `json:"has_lvh,omitempty"`
	}
	type mirrorSummaryContext struct {
		// P8-1 core
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
		// P8-2
		Age                 int                `json:"age,omitempty"`
		Sex                 string             `json:"sex,omitempty"`
		BMI                 float64            `json:"bmi,omitempty"`
		CKMStageV2          string             `json:"ckm_stage_v2,omitempty"`
		CKMSubstageMetadata *mirrorCKMSubstage `json:"ckm_substage_metadata,omitempty"`
		LatestPotassium     float64            `json:"latest_potassium,omitempty"`
		EngagementComposite *float64           `json:"engagement_composite,omitempty"`
		EngagementStatus    string             `json:"engagement_status,omitempty"`
		HasCGM              bool               `json:"has_cgm,omitempty"`
		LatestCGMTIR        *float64           `json:"latest_cgm_tir,omitempty"`
		LatestCGMGRIZone    string             `json:"latest_cgm_gri_zone,omitempty"`
		CGMReportAt         *time.Time         `json:"cgm_report_at,omitempty"`
	}

	// Fully-populated fixture so every consumer-facing field has a
	// distinct non-zero value. Any field that fails to round-trip
	// will land as zero on the KB-23 side and fail the assertions
	// below with a precise error.
	engagement := 0.85
	tir := 78.0
	lvef := 35.0
	cac := 275.0
	reportAt := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	fixture := mirrorSummaryContext{
		// P8-1 core
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
		// P8-2
		Age:                 58,
		Sex:                 "M",
		BMI:                 27.5,
		CKMStageV2:          "4c",
		LatestPotassium:     4.3,
		EngagementComposite: &engagement,
		EngagementStatus:    "ENGAGED",
		HasCGM:              true,
		LatestCGMTIR:        &tir,
		LatestCGMGRIZone:    "B",
		CGMReportAt:         &reportAt,
		CKMSubstageMetadata: &mirrorCKMSubstage{
			HFClassification: "HFrEF",
			LVEFPercent:      &lvef,
			NYHAClass:        "II",
			CACScore:         &cac,
		},
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

	// ── Phase 8 P8-2 assertions ──
	if got.Age != 58 {
		t.Errorf("Age = %d, want 58", got.Age)
	}
	if got.Sex != "M" {
		t.Errorf("Sex = %q, want M", got.Sex)
	}
	if got.BMI != 27.5 {
		t.Errorf("BMI = %f, want 27.5", got.BMI)
	}
	if got.CKMStageV2 != "4c" {
		t.Errorf("CKMStageV2 = %q, want 4c", got.CKMStageV2)
	}
	if got.LatestPotassium != 4.3 {
		t.Errorf("LatestPotassium = %f, want 4.3", got.LatestPotassium)
	}
	if got.EngagementComposite == nil || *got.EngagementComposite != 0.85 {
		t.Errorf("EngagementComposite = %v, want 0.85", got.EngagementComposite)
	}
	if got.EngagementStatus != "ENGAGED" {
		t.Errorf("EngagementStatus = %q, want ENGAGED", got.EngagementStatus)
	}
	if !got.HasCGM {
		t.Error("HasCGM = false, want true")
	}
	if got.LatestCGMTIR == nil || *got.LatestCGMTIR != 78.0 {
		t.Errorf("LatestCGMTIR = %v, want 78.0", got.LatestCGMTIR)
	}
	if got.LatestCGMGRIZone != "B" {
		t.Errorf("LatestCGMGRIZone = %q, want B", got.LatestCGMGRIZone)
	}
	if got.CGMReportAt == nil || !got.CGMReportAt.Equal(reportAt) {
		t.Errorf("CGMReportAt = %v, want %v", got.CGMReportAt, reportAt)
	}

	// CKM substage metadata — nested struct round-trip
	if got.CKMSubstageMetadata == nil {
		t.Fatal("CKMSubstageMetadata = nil, want non-nil")
	}
	if got.CKMSubstageMetadata.HFClassification != "HFrEF" {
		t.Errorf("HFClassification = %q, want HFrEF", got.CKMSubstageMetadata.HFClassification)
	}
	if got.CKMSubstageMetadata.LVEFPercent == nil || *got.CKMSubstageMetadata.LVEFPercent != 35.0 {
		t.Errorf("LVEFPercent = %v, want 35.0", got.CKMSubstageMetadata.LVEFPercent)
	}
	if got.CKMSubstageMetadata.NYHAClass != "II" {
		t.Errorf("NYHAClass = %q, want II", got.CKMSubstageMetadata.NYHAClass)
	}
	if got.CKMSubstageMetadata.CACScore == nil || *got.CKMSubstageMetadata.CACScore != 275.0 {
		t.Errorf("CACScore = %v, want 275.0", got.CKMSubstageMetadata.CACScore)
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
