package services

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// stubInertiaTimelineFetcher returns a fixed intervention timeline for tests.
type stubInertiaTimelineFetcher struct {
	timeline *KB20InterventionTimeline
	err      error
}

func (s *stubInertiaTimelineFetcher) FetchInterventionTimeline(ctx context.Context, patientID string) (*KB20InterventionTimeline, error) {
	return s.timeline, s.err
}

// stubInertiaPatientContextFetcher returns fixed summary + renal context.
type stubInertiaPatientContextFetcher struct {
	summary    *PatientContext
	summaryErr error
	renal      *KB20RenalStatus
	renalErr   error
}

func (s *stubInertiaPatientContextFetcher) FetchSummaryContext(ctx context.Context, patientID string) (*PatientContext, error) {
	return s.summary, s.summaryErr
}

func (s *stubInertiaPatientContextFetcher) FetchRenalStatus(ctx context.Context, patientID string) (*KB20RenalStatus, error) {
	return s.renal, s.renalErr
}

// stubInertiaTargetStatusFetcher returns fixed KB-26 target status.
type stubInertiaTargetStatusFetcher struct {
	resp *KB26TargetStatusResponse
	err  error
}

func (s *stubInertiaTargetStatusFetcher) FetchTargetStatus(ctx context.Context, patientID string, req KB26TargetStatusRequest) (*KB26TargetStatusResponse, error) {
	return s.resp, s.err
}

// stubInertiaCGMLatestFetcher returns a fixed CGM period report.
type stubInertiaCGMLatestFetcher struct {
	report *KB26CGMLatestReport
	err    error
}

func (s *stubInertiaCGMLatestFetcher) FetchLatestCGMReport(ctx context.Context, patientID string) (*KB26CGMLatestReport, error) {
	return s.report, s.err
}

// TestAssembleInertiaInput_CGMTIRBranchOverridesHbA1c verifies that
// when a recent CGM period report is available, the glycaemic domain
// input uses DataSource="CGM_TIR" with the TIR value as the clinical
// indicator, overriding the HbA1c-based target status from KB-26.
// This is the Phase 7 P7-E Milestone 2 verification: TIR reaches the
// inertia detector's cgmMinDays=14 branch.
func TestAssembleInertiaInput_CGMTIRBranchOverridesHbA1c(t *testing.T) {
	timeline := &stubInertiaTimelineFetcher{
		timeline: &KB20InterventionTimeline{
			PatientID: "p-cgm",
			ByDomain:  map[string]KB20LatestDomainAction{},
		},
	}
	ctxFetcher := &stubInertiaPatientContextFetcher{
		summary: &PatientContext{
			PatientID:   "p-cgm",
			Medications: []string{"METFORMIN"},
			LatestHbA1c: 8.2, // Would drive HbA1c path if CGM branch didn't override
		},
	}
	// HbA1c-based glycaemic verdict: off-target (current=8.2, target=7.0)
	targetFetcher := &stubInertiaTargetStatusFetcher{
		resp: &KB26TargetStatusResponse{
			Glycaemic: KB26DomainTargetStatus{
				Domain:       "GLYCAEMIC",
				AtTarget:     false,
				CurrentValue: 8.2,
				TargetValue:  7.0,
				DataSource:   "HBA1C",
			},
		},
	}
	// CGM report: off-target via TIR path (current=55, target=70)
	cgmFetcher := &stubInertiaCGMLatestFetcher{
		report: &KB26CGMLatestReport{
			PatientID:   "p-cgm",
			PeriodEnd:   time.Now().UTC(),
			PeriodStart: time.Now().UTC().AddDate(0, 0, -14),
			TIRPct:      55.0,
			MeanGlucose: 180.0,
			GRIZone:     "C",
		},
	}

	assembler := NewInertiaInputAssembler(timeline, ctxFetcher, targetFetcher, cgmFetcher, zap.NewNop())
	input, err := assembler.AssembleInertiaInput(context.Background(), "p-cgm")
	if err != nil {
		t.Fatalf("AssembleInertiaInput: %v", err)
	}

	if input.Glycaemic == nil {
		t.Fatal("expected glycaemic input, got nil")
	}
	if input.Glycaemic.DataSource != "CGM_TIR" {
		t.Errorf("DataSource = %q, want CGM_TIR (expected CGM branch to override HbA1c)",
			input.Glycaemic.DataSource)
	}
	if input.Glycaemic.CurrentValue != 55.0 {
		t.Errorf("CurrentValue = %f, want 55.0 (TIR from CGM)", input.Glycaemic.CurrentValue)
	}
	if input.Glycaemic.TargetValue != 70.0 {
		t.Errorf("TargetValue = %f, want 70.0 (TIR target)", input.Glycaemic.TargetValue)
	}
	if input.Glycaemic.AtTarget {
		t.Error("AtTarget should be false when TIR=55 < target=70")
	}
	if input.Glycaemic.DaysUncontrolled != 14 {
		t.Errorf("DaysUncontrolled = %d, want 14 (CGM 14-day window minimum)",
			input.Glycaemic.DaysUncontrolled)
	}
}

// TestAssembleInertiaInput_CGMTIRBranch_AtTarget verifies that a
// patient whose CGM TIR is above target produces a glycaemic input
// with AtTarget=true, bypassing the inertia detector's detection
// branch entirely.
func TestAssembleInertiaInput_CGMTIRBranch_AtTarget(t *testing.T) {
	assembler := NewInertiaInputAssembler(
		&stubInertiaTimelineFetcher{timeline: &KB20InterventionTimeline{ByDomain: map[string]KB20LatestDomainAction{}}},
		&stubInertiaPatientContextFetcher{summary: &PatientContext{PatientID: "p", Medications: []string{}}},
		&stubInertiaTargetStatusFetcher{resp: &KB26TargetStatusResponse{}},
		&stubInertiaCGMLatestFetcher{
			report: &KB26CGMLatestReport{
				PatientID: "p",
				TIRPct:    85.0, // well above target
				GRIZone:   "A",
				PeriodEnd: time.Now().UTC(),
			},
		},
		zap.NewNop(),
	)

	input, err := assembler.AssembleInertiaInput(context.Background(), "p")
	if err != nil {
		t.Fatalf("AssembleInertiaInput: %v", err)
	}
	if input.Glycaemic == nil {
		t.Fatal("expected glycaemic input")
	}
	if !input.Glycaemic.AtTarget {
		t.Error("AtTarget should be true when TIR=85 >= target=70")
	}
	if input.Glycaemic.DataSource != "CGM_TIR" {
		t.Errorf("DataSource = %q, want CGM_TIR", input.Glycaemic.DataSource)
	}
	if input.Glycaemic.DaysUncontrolled != 0 {
		t.Errorf("DaysUncontrolled = %d, want 0 for at-target patient",
			input.Glycaemic.DaysUncontrolled)
	}
}

// TestAssembleInertiaInput_FallsBackToHbA1cWhenNoCGMReport verifies
// that a patient without CGM data degrades gracefully — the
// HbA1c-based target status from KB-26 populates the glycaemic input
// as before, leaving DataSource=HBA1C.
func TestAssembleInertiaInput_FallsBackToHbA1cWhenNoCGMReport(t *testing.T) {
	assembler := NewInertiaInputAssembler(
		&stubInertiaTimelineFetcher{timeline: &KB20InterventionTimeline{ByDomain: map[string]KB20LatestDomainAction{}}},
		&stubInertiaPatientContextFetcher{summary: &PatientContext{PatientID: "p", LatestHbA1c: 8.5}},
		&stubInertiaTargetStatusFetcher{resp: &KB26TargetStatusResponse{
			Glycaemic: KB26DomainTargetStatus{
				Domain:       "GLYCAEMIC",
				AtTarget:     false,
				CurrentValue: 8.5,
				TargetValue:  7.0,
				DataSource:   "HBA1C",
				ConsecutiveReadings: 1,
			},
		}},
		&stubInertiaCGMLatestFetcher{report: nil}, // no CGM data
		zap.NewNop(),
	)

	input, err := assembler.AssembleInertiaInput(context.Background(), "p")
	if err != nil {
		t.Fatalf("AssembleInertiaInput: %v", err)
	}
	if input.Glycaemic == nil {
		t.Fatal("expected glycaemic input from HbA1c path")
	}
	if input.Glycaemic.DataSource != "HBA1C" {
		t.Errorf("DataSource = %q, want HBA1C (CGM unavailable)", input.Glycaemic.DataSource)
	}
	if input.Glycaemic.CurrentValue != 8.5 {
		t.Errorf("CurrentValue = %f, want 8.5 (HbA1c)", input.Glycaemic.CurrentValue)
	}
}

// TestAssembleInertiaInput_NilCGMFetcherDegradesCleanly asserts that
// a ConcreteInertiaInputAssembler with no CGM fetcher wired still
// works — the assembler skips the CGM branch entirely.
func TestAssembleInertiaInput_NilCGMFetcherDegradesCleanly(t *testing.T) {
	assembler := NewInertiaInputAssembler(
		&stubInertiaTimelineFetcher{timeline: &KB20InterventionTimeline{ByDomain: map[string]KB20LatestDomainAction{}}},
		&stubInertiaPatientContextFetcher{summary: &PatientContext{PatientID: "p"}},
		&stubInertiaTargetStatusFetcher{resp: &KB26TargetStatusResponse{}},
		nil, // nil CGM fetcher
		zap.NewNop(),
	)

	_, err := assembler.AssembleInertiaInput(context.Background(), "p")
	if err != nil {
		t.Errorf("expected nil error with nil CGM fetcher, got %v", err)
	}
}

// TestBuildCGMGlycaemicInput_PureMapping verifies the pure helper
// that converts a CGMLatestReport into a DomainInertiaInput.
func TestBuildCGMGlycaemicInput_PureMapping(t *testing.T) {
	cgm := &KB26CGMLatestReport{
		TIRPct:      65.0,
		MeanGlucose: 165.0,
		PeriodEnd:   time.Now().UTC(),
	}
	timeline := KB20LatestDomainAction{
		DrugClass:  "METFORMIN",
		ActionDate: time.Now().UTC().AddDate(0, 0, -30),
	}
	meds := []string{"METFORMIN", "SGLT2I"}

	input := buildCGMGlycaemicInput(cgm, timeline, meds)
	if input == nil {
		t.Fatal("expected non-nil input")
	}
	if input.DataSource != "CGM_TIR" {
		t.Errorf("DataSource = %q, want CGM_TIR", input.DataSource)
	}
	if input.CurrentValue != 65.0 {
		t.Errorf("CurrentValue = %f, want 65.0", input.CurrentValue)
	}
	if input.AtTarget {
		t.Error("AtTarget should be false for TIR=65 < target=70")
	}
	if input.DaysUncontrolled != 14 {
		t.Errorf("DaysUncontrolled = %d, want 14", input.DaysUncontrolled)
	}
	if input.LastIntervention == nil {
		t.Error("expected non-nil LastIntervention from timeline entry")
	}
	if len(input.CurrentMeds) != 2 {
		t.Errorf("expected 2 current meds, got %d", len(input.CurrentMeds))
	}
}
