package services

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// Note: testConfigDir is defined in renal_formulary_test.go in the same package.

// stubRenalContextFetcher lets the orchestrator tests run without a
// real KB-20 HTTP server. The returned KB20RenalStatus is fully formed
// so the orchestrator's pure-logic path can be exercised end-to-end.
type stubRenalContextFetcher struct {
	status *KB20RenalStatus
	err    error
}

func (s *stubRenalContextFetcher) FetchRenalStatus(ctx context.Context, patientID string) (*KB20RenalStatus, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.status, nil
}

// testRenalFormulary loads the real shared renal formulary from disk
// so the orchestrator tests exercise the same rules the production
// FindApproachingThresholds path does. Reuses the testConfigDir helper
// from renal_formulary_test.go which correctly resolves the market-
// configs directory relative to this package.
func testRenalFormulary(t *testing.T) *RenalFormulary {
	t.Helper()
	formulary, err := LoadRenalFormulary(testConfigDir(t), "")
	if err != nil {
		t.Fatalf("LoadRenalFormulary: %v", err)
	}
	return formulary
}

// TestRenalAnticipatoryOrchestrator_ApproachingMetformin verifies that
// a patient with eGFR=33 and a -8/year decline trajectory, on metformin,
// produces a RENAL_THRESHOLD_APPROACHING alert for the metformin class.
// The shared formulary sets metformin anticipate_months=6 and
// contraindicated_below=30, so the crossing projection must land within
// 6 months. eGFR=33 slope=-8 → gap=3, years=0.375, months≈4.5 — fires.
// Note: the plan's original "eGFR=45 + slope=-6 → ~30 months" scenario
// does not fire under the default 6-month horizon; the assertion here
// uses values calibrated to the actual formulary so the test exercises
// the full projection→alert pipeline.
func TestRenalAnticipatoryOrchestrator_ApproachingMetformin(t *testing.T) {
	formulary := testRenalFormulary(t)
	orch := NewRenalAnticipatoryOrchestrator(nil, formulary, zap.NewNop())

	status := &KB20RenalStatus{
		PatientID:      "p1",
		EGFR:           33.0,
		EGFRSlope:      -8.0,
		EGFRMeasuredAt: time.Now(),
		CKDStage:       "G3b",
		ActiveMedications: []KB20MedSummary{
			{DrugName: "metformin", DrugClass: "METFORMIN", IsActive: true},
		},
	}

	result := orch.EvaluateWithContext("p1", status)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.ApproachingAlerts) == 0 {
		t.Fatal("expected at least one approaching alert for metformin at eGFR=45 slope=-6")
	}

	found := false
	for _, alert := range result.ApproachingAlerts {
		if alert.DrugClass == "METFORMIN" {
			found = true
			if alert.MonthsToThreshold <= 0 {
				t.Errorf("expected positive months to threshold, got %.1f", alert.MonthsToThreshold)
			}
		}
	}
	if !found {
		t.Errorf("expected METFORMIN alert, got alerts = %+v", result.ApproachingAlerts)
	}
}

// TestRenalAnticipatoryOrchestrator_StableNoAlerts asserts that a stable
// patient (positive slope = improving) produces no approaching alerts.
// Prevents false positives on patients who shouldn't be on the batch's
// radar.
func TestRenalAnticipatoryOrchestrator_StableNoAlerts(t *testing.T) {
	formulary := testRenalFormulary(t)
	orch := NewRenalAnticipatoryOrchestrator(nil, formulary, zap.NewNop())

	status := &KB20RenalStatus{
		PatientID:      "p-stable",
		EGFR:           75.0,
		EGFRSlope:      1.0, // improving
		EGFRMeasuredAt: time.Now(),
		CKDStage:       "G2",
		ActiveMedications: []KB20MedSummary{
			{DrugName: "metformin", DrugClass: "METFORMIN", IsActive: true},
		},
	}

	result := orch.EvaluateWithContext("p-stable", status)
	if len(result.ApproachingAlerts) != 0 {
		t.Errorf("expected no alerts for stable improving patient, got %+v", result.ApproachingAlerts)
	}
}

// TestRenalAnticipatoryOrchestrator_StaleEGFRDetected asserts that a
// patient on a renal-sensitive medication whose eGFR is 200 days old
// produces a stale-eGFR verdict. Maps to P7-C verification question #4.
func TestRenalAnticipatoryOrchestrator_StaleEGFRDetected(t *testing.T) {
	formulary := testRenalFormulary(t)
	orch := NewRenalAnticipatoryOrchestrator(nil, formulary, zap.NewNop())

	status := &KB20RenalStatus{
		PatientID:      "p-stale",
		EGFR:           55.0,
		EGFRSlope:      -1.0,
		EGFRMeasuredAt: time.Now().AddDate(0, 0, -270), // 9 months old
		CKDStage:       "G3a",
		ActiveMedications: []KB20MedSummary{
			// DrugClass "ACEi" matches the shared formulary key. The test
			// previously used "ACE_INHIBITOR" which silently left
			// onRenalSensitive=false because the formulary lookup missed.
			{DrugName: "lisinopril", DrugClass: "ACEi", IsActive: true},
		},
	}

	result := orch.EvaluateWithContext("p-stale", status)
	if !result.StaleEGFRTriggered {
		t.Errorf("expected stale-eGFR triggered for 270-day-old reading on renal-sensitive med, got %+v", result.StaleEGFR)
	}
}

// TestRenalAnticipatoryOrchestrator_StaleEGFR_NoTriggerWithoutRenalSensitiveMed
// covers the inverse: a patient with a 9-month-old eGFR but no renal-
// sensitive medications is NOT flagged, because lab-planning for
// unmedicated patients is out of scope for this batch.
func TestRenalAnticipatoryOrchestrator_StaleEGFR_NoTriggerWithoutRenalSensitiveMed(t *testing.T) {
	formulary := testRenalFormulary(t)
	orch := NewRenalAnticipatoryOrchestrator(nil, formulary, zap.NewNop())

	status := &KB20RenalStatus{
		PatientID:      "p-unmedicated",
		EGFR:           55.0,
		EGFRSlope:      -1.0,
		EGFRMeasuredAt: time.Now().AddDate(0, 0, -270),
		CKDStage:       "G3a",
		ActiveMedications: []KB20MedSummary{
			{DrugName: "paracetamol", DrugClass: "ANALGESIC", IsActive: true},
		},
	}

	result := orch.EvaluateWithContext("p-unmedicated", status)
	if result.StaleEGFRTriggered {
		t.Errorf("expected no stale-eGFR trigger for unmedicated patient, got triggered")
	}
}

// TestRenalAnticipatoryOrchestrator_EvaluatePatient_FetcherError asserts
// that a fetcher error propagates cleanly as an error return (not a
// silent nil result).
func TestRenalAnticipatoryOrchestrator_EvaluatePatient_FetcherError(t *testing.T) {
	formulary := testRenalFormulary(t)
	fetcher := &stubRenalContextFetcher{err: context.DeadlineExceeded}
	orch := NewRenalAnticipatoryOrchestrator(fetcher, formulary, zap.NewNop())

	_, err := orch.EvaluatePatient(context.Background(), "p1")
	if err == nil {
		t.Error("expected fetcher error to propagate, got nil")
	}
}

// TestRenalAnticipatoryOrchestrator_EvaluatePatient_NilFetcherReturnsError
// asserts that the orchestrator is defensive against a missing fetcher
// dependency rather than panicking.
func TestRenalAnticipatoryOrchestrator_EvaluatePatient_NilFetcherReturnsError(t *testing.T) {
	formulary := testRenalFormulary(t)
	orch := NewRenalAnticipatoryOrchestrator(nil, formulary, zap.NewNop())

	_, err := orch.EvaluatePatient(context.Background(), "p1")
	if err == nil {
		t.Error("expected error when fetcher nil, got nil")
	}
}

// TestRenalAnticipatoryTemplates_LoadFromDisk verifies both P7-C YAML
// templates parse via TemplateLoader. Cheap upstream guard against
// template typos.
func TestRenalAnticipatoryTemplates_LoadFromDisk(t *testing.T) {
	templatesDir, err := filepath.Abs("../../templates")
	if err != nil {
		t.Fatalf("resolve templates dir: %v", err)
	}
	loader := NewTemplateLoader(templatesDir, zap.NewNop())
	if err := loader.Load(); err != nil {
		t.Fatalf("TemplateLoader.Load: %v", err)
	}

	tests := []struct {
		templateID       string
		wantDifferential string
	}{
		{"dc-renal-threshold-approaching-v1", "RENAL_THRESHOLD_APPROACHING"},
		{"dc-stale-egfr-v1", "STALE_EGFR"},
	}
	for _, tc := range tests {
		t.Run(tc.templateID, func(t *testing.T) {
			tmpl, ok := loader.Get(tc.templateID)
			if !ok {
				t.Fatalf("template %s not loaded", tc.templateID)
			}
			if tmpl.DifferentialID != tc.wantDifferential {
				t.Errorf("differential_id = %q, want %q", tmpl.DifferentialID, tc.wantDifferential)
			}
			if len(tmpl.Fragments) == 0 {
				t.Errorf("template %s: expected fragments, got none", tc.templateID)
			}
		})
	}
}

// TestRenderApproachingSummaries_SubstitutesDrugClassAndHorizon verifies
// the pure helper that renders the approaching-threshold template with
// runtime alert data.
func TestRenderApproachingSummaries_SubstitutesDrugClassAndHorizon(t *testing.T) {
	tmpl := &models.CardTemplate{
		Fragments: []models.TemplateFragment{
			{FragmentType: models.FragClinician, TextEn: "eGFR {{.EGFR}} {{.DrugClass}} {{.ThresholdType}} {{.MonthsToThreshold}}"},
		},
	}
	result := &RenalAnticipatoryResult{EGFR: 45.0}
	alert := AnticipatoryAlert{
		DrugClass:         "METFORMIN",
		ThresholdType:     "CONTRAINDICATION",
		ThresholdValue:    30.0,
		MonthsToThreshold: 30.0,
	}
	clinician, _, _ := renderApproachingSummaries(tmpl, result, alert)
	if !strings.Contains(clinician, "45.0") || !strings.Contains(clinician, "METFORMIN") ||
		!strings.Contains(clinician, "CONTRAINDICATION") || !strings.Contains(clinician, "30.0") {
		t.Errorf("missing substitution tokens in %q", clinician)
	}
}

// TestRenderStaleEGFRSummaries_SubstitutesDaysAndExpected verifies the
// stale-eGFR pure helper.
func TestRenderStaleEGFRSummaries_SubstitutesDaysAndExpected(t *testing.T) {
	tmpl := &models.CardTemplate{
		Fragments: []models.TemplateFragment{
			{FragmentType: models.FragClinician, TextEn: "{{.DaysSince}}/{{.ExpectedMaxDays}} {{.Severity}}"},
		},
	}
	result := &RenalAnticipatoryResult{
		EGFR:     55.0,
		CKDStage: "G3a",
		StaleEGFR: StaleEGFRResult{
			DaysSince:       270,
			ExpectedMaxDays: 90,
			Severity:        "CRITICAL",
		},
	}
	clinician, _, _ := renderStaleEGFRSummaries(tmpl, result)
	if !strings.Contains(clinician, "270") || !strings.Contains(clinician, "90") ||
		!strings.Contains(clinician, "CRITICAL") {
		t.Errorf("missing substitution tokens in %q", clinician)
	}
}
