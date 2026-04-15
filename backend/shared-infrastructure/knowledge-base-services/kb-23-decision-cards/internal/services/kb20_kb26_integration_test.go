package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
)

// This file implements the integration-test pattern rollout called
// out in the Phase 7 retrospective and the Phase 8 P8-1 commit
// message. The pattern is: one test per cross-service client method,
// running against a real httptest.Server with a mirror handler that
// pins the EXACT production route and wire shape. No stubs, no
// interfaces, no mocks — real JSON round trips through real HTTP.
//
// Phase 8 P8-4 ships the four remaining methods:
//   - KB20Client.FetchRenalStatus           (production route
//     GET /api/v1/patient/:id/renal-status)
//   - KB20Client.FetchInterventionTimeline  (production route
//     GET /api/v1/patient/:id/intervention-timeline)
//   - KB20Client.FetchRenalActivePatientIDs (production route
//     GET /api/v1/patients/renal-active)
//   - KB26Client.FetchTargetStatus          (production route
//     POST /api/v1/kb26/target-status/:patientId)
//   - KB26Client.FetchLatestCGMReport       (production route
//     GET /api/v1/kb26/cgm-latest/:patientId)
//
// After P8-4 lands, every cross-service client method in the KB-23
// services package has an integration test against a real handler.
// The silent-404 class of bug that survived P7-A through P7-E on
// FetchSummaryContext has no place to hide — any drift between a
// client's URL template and its server-side route fails a test
// deterministically at CI time.

// ─────────────────── FetchRenalStatus ───────────────────

// TestKB20Client_FetchRenalStatus_RoundTripsAgainstRealHandler pins
// the KB-20 renal-status wire contract. Mirror handler is registered
// at the EXACT production route /api/v1/patient/:id/renal-status.
// Phase 8 P8-4.
func TestKB20Client_FetchRenalStatus_RoundTripsAgainstRealHandler(t *testing.T) {
	// Mirror the KB20RenalStatus struct field-for-field so JSON tag
	// drift on either side fails this test.
	type mirrorMed struct {
		DrugName  string `json:"drug_name"`
		DrugClass string `json:"drug_class"`
		DoseMg    string `json:"dose_mg"`
		IsActive  bool   `json:"is_active"`
	}
	type mirrorRenalStatus struct {
		PatientID         string      `json:"patient_id"`
		EGFR              float64     `json:"egfr"`
		EGFRSlope         float64     `json:"egfr_slope"`
		EGFRMeasuredAt    time.Time   `json:"egfr_measured_at"`
		EGFRDataPoints    int         `json:"egfr_data_points"`
		Potassium         *float64    `json:"potassium,omitempty"`
		ACR               *float64    `json:"acr,omitempty"`
		CKDStage          string      `json:"ckd_stage"`
		IsRapidDecliner   bool        `json:"is_rapid_decliner"`
		ActiveMedications []mirrorMed `json:"active_medications"`
	}

	potassium := 4.3
	acr := 15.0
	fixture := mirrorRenalStatus{
		PatientID:      "p-renal",
		EGFR:           42.5,
		EGFRSlope:      -3.2,
		EGFRMeasuredAt: time.Date(2026, 4, 15, 10, 30, 0, 0, time.UTC),
		EGFRDataPoints: 8,
		Potassium:      &potassium,
		ACR:            &acr,
		CKDStage:       "G3b",
		IsRapidDecliner: false,
		ActiveMedications: []mirrorMed{
			{DrugName: "metformin", DrugClass: "METFORMIN", DoseMg: "500", IsActive: true},
			{DrugName: "lisinopril", DrugClass: "ACEi", DoseMg: "10", IsActive: true},
		},
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/api/v1/patient/p-renal/renal-status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    fixture,
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	cfg := &config.Config{KB20URL: server.URL}
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchRenalStatus(context.Background(), "p-renal")
	if err != nil {
		t.Fatalf("FetchRenalStatus: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil renal status, got nil")
	}

	if got.PatientID != "p-renal" {
		t.Errorf("PatientID = %q, want p-renal", got.PatientID)
	}
	if got.EGFR != 42.5 {
		t.Errorf("EGFR = %f, want 42.5", got.EGFR)
	}
	if got.EGFRSlope != -3.2 {
		t.Errorf("EGFRSlope = %f, want -3.2", got.EGFRSlope)
	}
	if got.CKDStage != "G3b" {
		t.Errorf("CKDStage = %q, want G3b", got.CKDStage)
	}
	if got.IsRapidDecliner {
		t.Error("IsRapidDecliner = true, want false")
	}
	if len(got.ActiveMedications) != 2 {
		t.Fatalf("len(ActiveMedications) = %d, want 2", len(got.ActiveMedications))
	}
	if got.ActiveMedications[0].DrugClass != "METFORMIN" {
		t.Errorf("first med DrugClass = %q, want METFORMIN", got.ActiveMedications[0].DrugClass)
	}
	if got.Potassium == nil || *got.Potassium != 4.3 {
		t.Errorf("Potassium = %v, want 4.3", got.Potassium)
	}
	if got.ACR == nil || *got.ACR != 15.0 {
		t.Errorf("ACR = %v, want 15.0", got.ACR)
	}
	// EGFRMeasuredAt must round-trip exactly so downstream time-based
	// calculations (staleness detection, trajectory slope windows)
	// honour the real measurement time, not a zero-value fallback.
	if !got.EGFRMeasuredAt.Equal(fixture.EGFRMeasuredAt) {
		t.Errorf("EGFRMeasuredAt = %v, want %v", got.EGFRMeasuredAt, fixture.EGFRMeasuredAt)
	}
}

// TestKB20Client_FetchRenalStatus_404Propagates verifies the
// silent-404 regression guard for this method too. P8-4.
func TestKB20Client_FetchRenalStatus_404Propagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	cfg := &config.Config{KB20URL: server.URL}
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchRenalStatus(context.Background(), "p-missing")
	if err == nil {
		t.Error("expected error on 404, got nil")
	}
	if got != nil {
		t.Errorf("expected nil on 404, got %+v", got)
	}
}

// ─────────────────── FetchInterventionTimeline ───────────────────

// TestKB20Client_FetchInterventionTimeline_RoundTripsAgainstRealHandler
// pins the KB-20 intervention-timeline wire contract at the
// production route /api/v1/patient/:id/intervention-timeline. P8-4.
func TestKB20Client_FetchInterventionTimeline_RoundTripsAgainstRealHandler(t *testing.T) {
	type mirrorDomainAction struct {
		InterventionID   string    `json:"InterventionID"`
		InterventionType string    `json:"InterventionType"`
		DrugClass        string    `json:"DrugClass"`
		DrugName         string    `json:"DrugName"`
		DoseMg           float64   `json:"DoseMg"`
		ActionDate       time.Time `json:"ActionDate"`
		DaysSince        int       `json:"DaysSince"`
	}
	type mirrorTimeline struct {
		PatientID                string                        `json:"PatientID"`
		ByDomain                 map[string]mirrorDomainAction `json:"ByDomain"`
		AnyChangeInLast12Weeks   bool                          `json:"AnyChangeInLast12Weeks"`
		TotalActiveInterventions int                           `json:"TotalActiveInterventions"`
	}

	fixture := mirrorTimeline{
		PatientID: "p-timeline",
		ByDomain: map[string]mirrorDomainAction{
			"GLYCAEMIC": {
				InterventionID:   "int-1",
				InterventionType: "MEDICATION_CHANGE",
				DrugClass:        "METFORMIN",
				DrugName:         "metformin",
				DoseMg:           1000.0,
				ActionDate:       time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				DaysSince:        45,
			},
			"HEMODYNAMIC": {
				InterventionID:   "int-2",
				InterventionType: "MEDICATION_CHANGE",
				DrugClass:        "ACEi",
				DrugName:         "lisinopril",
				DoseMg:           10.0,
				ActionDate:       time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
				DaysSince:        26,
			},
		},
		AnyChangeInLast12Weeks:   true,
		TotalActiveInterventions: 2,
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/api/v1/patient/p-timeline/intervention-timeline", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    fixture,
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	cfg := &config.Config{KB20URL: server.URL}
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchInterventionTimeline(context.Background(), "p-timeline")
	if err != nil {
		t.Fatalf("FetchInterventionTimeline: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil timeline, got nil")
	}
	if got.PatientID != "p-timeline" {
		t.Errorf("PatientID = %q, want p-timeline", got.PatientID)
	}
	if !got.AnyChangeInLast12Weeks {
		t.Error("AnyChangeInLast12Weeks = false, want true")
	}
	if got.TotalActiveInterventions != 2 {
		t.Errorf("TotalActiveInterventions = %d, want 2", got.TotalActiveInterventions)
	}
	if len(got.ByDomain) != 2 {
		t.Fatalf("len(ByDomain) = %d, want 2", len(got.ByDomain))
	}
	glyc, ok := got.ByDomain["GLYCAEMIC"]
	if !ok {
		t.Fatal("expected GLYCAEMIC in ByDomain")
	}
	if glyc.DrugClass != "METFORMIN" {
		t.Errorf("GLYCAEMIC DrugClass = %q, want METFORMIN", glyc.DrugClass)
	}
	if glyc.DoseMg != 1000.0 {
		t.Errorf("GLYCAEMIC DoseMg = %f, want 1000.0", glyc.DoseMg)
	}
	if glyc.DaysSince != 45 {
		t.Errorf("GLYCAEMIC DaysSince = %d, want 45", glyc.DaysSince)
	}
}

// ─────────────────── FetchRenalActivePatientIDs ───────────────────

// TestKB20Client_FetchRenalActivePatientIDs_RoundTripsAgainstRealHandler
// pins the KB-20 renal-active list endpoint. Bonus method added to
// the integration-test rollout because it's a cross-service client
// method with the same stub-only test risk surface. Phase 8 P8-4.
func TestKB20Client_FetchRenalActivePatientIDs_RoundTripsAgainstRealHandler(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/v1/patients/renal-active", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data": []map[string]interface{}{
				{"patient_id": "p-1"},
				{"patient_id": "p-2"},
				{"patient_id": "p-3"},
			},
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	cfg := &config.Config{KB20URL: server.URL}
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchRenalActivePatientIDs(context.Background())
	if err != nil {
		t.Fatalf("FetchRenalActivePatientIDs: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	expected := map[string]bool{"p-1": true, "p-2": true, "p-3": true}
	for _, id := range got {
		if !expected[id] {
			t.Errorf("unexpected patient id %q", id)
		}
	}
}

// TestKB20Client_FetchRenalActivePatientIDs_EmptyList verifies the
// zero-patients case (valid 200 response with an empty data array).
// The batch consumer treats this as "no renal-active patients this
// cycle" and logs at info level without erroring.
func TestKB20Client_FetchRenalActivePatientIDs_EmptyList(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/api/v1/patients/renal-active", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    []interface{}{},
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	cfg := &config.Config{KB20URL: server.URL}
	client := NewKB20Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchRenalActivePatientIDs(context.Background())
	if err != nil {
		t.Fatalf("FetchRenalActivePatientIDs: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}

// ─────────────────── FetchTargetStatus ───────────────────

// TestKB26Client_FetchTargetStatus_RoundTripsAgainstRealHandler
// pins the KB-26 target-status wire contract. Unlike the other
// methods, this is a POST with a request body carrying raw HbA1c /
// SBP / eGFR measurements. The mirror handler decodes the body
// and verifies the client-to-server direction works end-to-end,
// then encodes the response envelope for the server-to-client
// direction. Both halves of the wire contract are pinned. P8-4.
func TestKB26Client_FetchTargetStatus_RoundTripsAgainstRealHandler(t *testing.T) {
	type mirrorDomainResult struct {
		Domain              string     `json:"Domain"`
		AtTarget            bool       `json:"AtTarget"`
		CurrentValue        float64    `json:"CurrentValue"`
		TargetValue         float64    `json:"TargetValue"`
		FirstUncontrolledAt *time.Time `json:"FirstUncontrolledAt,omitempty"`
		DaysUncontrolled    int        `json:"DaysUncontrolled"`
		ConsecutiveReadings int        `json:"ConsecutiveReadings"`
		DataSource          string     `json:"DataSource"`
		Confidence          string     `json:"Confidence"`
	}
	type mirrorResponse struct {
		Glycaemic   mirrorDomainResult `json:"glycaemic"`
		Hemodynamic mirrorDomainResult `json:"hemodynamic"`
		Renal       mirrorDomainResult `json:"renal"`
	}

	// The handler asserts that the request body carries the fields
	// the client is supposed to send — this is the request-direction
	// half of the wire contract.
	var receivedReq KB26TargetStatusRequest
	handler := http.NewServeMux()
	handler.HandleFunc("/api/v1/kb26/target-status/p-targets", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedReq)

		fixture := mirrorResponse{
			Glycaemic: mirrorDomainResult{
				Domain: "GLYCAEMIC", AtTarget: false, CurrentValue: 8.5,
				TargetValue: 7.0, DataSource: "HBA1C", Confidence: "MODERATE",
				ConsecutiveReadings: 1,
			},
			Hemodynamic: mirrorDomainResult{
				Domain: "HEMODYNAMIC", AtTarget: true, CurrentValue: 128.0,
				TargetValue: 130.0, DataSource: "HOME_BP", Confidence: "HIGH",
				ConsecutiveReadings: 1,
			},
			Renal: mirrorDomainResult{
				Domain: "RENAL", AtTarget: true, CurrentValue: 55.0,
				TargetValue: 45.0, DataSource: "EGFR", Confidence: "MODERATE",
				ConsecutiveReadings: 1,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    fixture,
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	cfg := &config.Config{KB26URL: server.URL}
	client := NewKB26Client(cfg, testMetricsCollector(), zap.NewNop())

	hba1c := 8.5
	sbp := 128.0
	egfr := 55.0
	req := KB26TargetStatusRequest{
		HbA1c:       &hba1c,
		HbA1cTarget: 7.0,
		MeanSBP7d:   &sbp,
		SBPTarget:   130.0,
		EGFR:        &egfr,
		EGFRTarget:  45.0,
	}
	got, err := client.FetchTargetStatus(context.Background(), "p-targets", req)
	if err != nil {
		t.Fatalf("FetchTargetStatus: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil response, got nil")
	}

	// Request-direction assertions: verify the server received what
	// the client was supposed to send.
	if receivedReq.HbA1c == nil || *receivedReq.HbA1c != 8.5 {
		t.Errorf("server received HbA1c = %v, want 8.5", receivedReq.HbA1c)
	}
	if receivedReq.HbA1cTarget != 7.0 {
		t.Errorf("server received HbA1cTarget = %f, want 7.0", receivedReq.HbA1cTarget)
	}
	if receivedReq.MeanSBP7d == nil || *receivedReq.MeanSBP7d != 128.0 {
		t.Errorf("server received MeanSBP7d = %v, want 128.0", receivedReq.MeanSBP7d)
	}
	if receivedReq.EGFR == nil || *receivedReq.EGFR != 55.0 {
		t.Errorf("server received EGFR = %v, want 55.0", receivedReq.EGFR)
	}

	// Response-direction assertions: verify the client correctly
	// decoded the per-domain verdicts.
	if got.Glycaemic.Domain != "GLYCAEMIC" {
		t.Errorf("Glycaemic.Domain = %q, want GLYCAEMIC", got.Glycaemic.Domain)
	}
	if got.Glycaemic.AtTarget {
		t.Error("Glycaemic.AtTarget = true, want false")
	}
	if got.Glycaemic.CurrentValue != 8.5 {
		t.Errorf("Glycaemic.CurrentValue = %f, want 8.5", got.Glycaemic.CurrentValue)
	}
	if !got.Hemodynamic.AtTarget {
		t.Error("Hemodynamic.AtTarget = false, want true")
	}
	if !got.Renal.AtTarget {
		t.Error("Renal.AtTarget = false, want true")
	}
}

// ─────────────────── FetchLatestCGMReport ───────────────────

// TestKB26Client_FetchLatestCGMReport_RoundTripsAgainstRealHandler
// pins the KB-26 cgm-latest wire contract from the KB-23 side.
// (Note: the KB-20 side call to the same KB-26 endpoint was
// already tested in P8-3's kb26_client_test.go at
// kb-20-patient-profile/internal/clients/kb26_client_test.go.
// This test covers the KB-23-side KB26Client which is a separate
// code path with its own struct definitions.) P8-4.
func TestKB26Client_FetchLatestCGMReport_RoundTripsAgainstRealHandler(t *testing.T) {
	fixture := map[string]interface{}{
		"id":               42,
		"patient_id":       "p-cgm-k23",
		"period_start":     "2026-04-01T00:00:00Z",
		"period_end":       "2026-04-15T00:00:00Z",
		"coverage_pct":     96.2,
		"sufficient_data":  true,
		"confidence_level": "HIGH",
		"mean_glucose":     148.5,
		"sd_glucose":       22.4,
		"cv_pct":           15.1,
		"glucose_stable":   true,
		"tir_pct":          75.0,
		"tbr_l1_pct":       1.5,
		"tbr_l2_pct":       0.3,
		"tar_l1_pct":       18.0,
		"tar_l2_pct":       5.2,
		"gmi":              6.8,
		"gri":              15.0,
		"gri_zone":         "B",
		"hypo_events":      1,
		"created_at":       "2026-04-15T10:30:00Z",
	}

	handler := http.NewServeMux()
	handler.HandleFunc("/api/v1/kb26/cgm-latest/p-cgm-k23", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "wrong method", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    fixture,
		})
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	cfg := &config.Config{KB26URL: server.URL}
	client := NewKB26Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchLatestCGMReport(context.Background(), "p-cgm-k23")
	if err != nil {
		t.Fatalf("FetchLatestCGMReport: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil report, got nil")
	}

	if got.PatientID != "p-cgm-k23" {
		t.Errorf("PatientID = %q, want p-cgm-k23", got.PatientID)
	}
	if got.TIRPct != 75.0 {
		t.Errorf("TIRPct = %f, want 75.0", got.TIRPct)
	}
	if got.MeanGlucose != 148.5 {
		t.Errorf("MeanGlucose = %f, want 148.5", got.MeanGlucose)
	}
	if got.GRIZone != "B" {
		t.Errorf("GRIZone = %q, want B", got.GRIZone)
	}
	if got.CoveragePct != 96.2 {
		t.Errorf("CoveragePct = %f, want 96.2", got.CoveragePct)
	}
	if got.GMI != 6.8 {
		t.Errorf("GMI = %f, want 6.8", got.GMI)
	}
	if got.HypoEvents != 1 {
		t.Errorf("HypoEvents = %d, want 1", got.HypoEvents)
	}
}

// TestKB26Client_FetchLatestCGMReport_404ReturnsNilNilOK verifies
// the clean-no-data path on the KB-23 side: a 404 must return
// (nil, nil) so the inertia assembler's CGM override branch can
// skip to the HbA1c fallback without logging noise. This is
// structurally identical to the KB-20 side check but on the
// KB-23 client, which has its own implementation. P8-4.
func TestKB26Client_FetchLatestCGMReport_404ReturnsNilNilOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &config.Config{KB26URL: server.URL}
	client := NewKB26Client(cfg, testMetricsCollector(), zap.NewNop())

	got, err := client.FetchLatestCGMReport(context.Background(), "p-no-cgm")
	if err != nil {
		t.Errorf("expected nil error on 404, got %v", err)
	}
	if got != nil {
		t.Errorf("expected nil report on 404, got %+v", got)
	}
}
