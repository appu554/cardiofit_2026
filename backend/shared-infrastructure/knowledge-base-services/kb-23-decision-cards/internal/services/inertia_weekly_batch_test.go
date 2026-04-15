package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// stubInertiaActivePatientLister is a minimal implementation for tests.
type stubInertiaActivePatientLister struct {
	ids  []string
	err  error
	hits int
}

func (s *stubInertiaActivePatientLister) ListInertiaActivePatientIDs(ctx context.Context) ([]string, error) {
	s.hits++
	return s.ids, s.err
}

// stubInertiaInputAssembler returns a constant input for every patient.
type stubInertiaInputAssembler struct {
	callCount int
	err       error
}

func (s *stubInertiaInputAssembler) AssembleInertiaInput(ctx context.Context, patientID string) (InertiaDetectorInput, error) {
	s.callCount++
	if s.err != nil {
		return InertiaDetectorInput{}, s.err
	}
	// Minimal input — DetectInertia handles nil domain inputs gracefully.
	return InertiaDetectorInput{PatientID: patientID}, nil
}

func TestInertiaWeeklyBatch_ShouldRun_OnlyOnSundayAt03UTC(t *testing.T) {
	job := NewInertiaWeeklyBatch(&stubInertiaActivePatientLister{}, nil, nil, zap.NewNop())

	cases := []struct {
		name     string
		when     time.Time
		expected bool
	}{
		// 2026-04-19 is a Sunday
		{"Sunday 03:00 UTC fires", time.Date(2026, 4, 19, 3, 0, 0, 0, time.UTC), true},
		{"Sunday 02:00 UTC skips (wrong hour)", time.Date(2026, 4, 19, 2, 0, 0, 0, time.UTC), false},
		{"Sunday 04:00 UTC skips (wrong hour)", time.Date(2026, 4, 19, 4, 0, 0, 0, time.UTC), false},
		{"Monday 03:00 UTC skips (wrong day)", time.Date(2026, 4, 20, 3, 0, 0, 0, time.UTC), false},
		{"Saturday 03:00 UTC skips (wrong day)", time.Date(2026, 4, 18, 3, 0, 0, 0, time.UTC), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := job.ShouldRun(context.Background(), tc.when)
			if got != tc.expected {
				t.Errorf("ShouldRun(%v) = %v, want %v", tc.when, got, tc.expected)
			}
		})
	}
}

func TestInertiaWeeklyBatch_Run_HeartbeatMode(t *testing.T) {
	// When assembler + orchestrator are nil, the batch logs the patient
	// count without per-patient evaluation.
	repo := &stubInertiaActivePatientLister{ids: []string{"p1", "p2", "p3"}}
	job := NewInertiaWeeklyBatch(repo, nil, nil, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if repo.hits != 1 {
		t.Errorf("expected repo called once, got %d", repo.hits)
	}
}

func TestInertiaWeeklyBatch_Run_FullMode_EvaluatesEveryPatient(t *testing.T) {
	repo := &stubInertiaActivePatientLister{ids: []string{"p1", "p2", "p3"}}
	assembler := &stubInertiaInputAssembler{}
	orch := NewInertiaOrchestrator(nil, nil, nil, nil, nil, nil, zap.NewNop())
	job := NewInertiaWeeklyBatch(repo, assembler, orch, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if assembler.callCount != 3 {
		t.Errorf("expected 3 assembler calls, got %d", assembler.callCount)
	}
}

func TestInertiaWeeklyBatch_Run_AssemblyErrorIsPerPatientIsolated(t *testing.T) {
	// One patient's assembly error must not abort the batch — other
	// patients should still be evaluated.
	repo := &stubInertiaActivePatientLister{ids: []string{"p1", "p2", "p3"}}
	assembler := &stubInertiaInputAssembler{err: errors.New("simulated assembly failure")}
	orch := NewInertiaOrchestrator(nil, nil, nil, nil, nil, nil, zap.NewNop())
	job := NewInertiaWeeklyBatch(repo, assembler, orch, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Errorf("expected nil (per-patient errors should be isolated), got %v", err)
	}
}

func TestInertiaWeeklyBatch_Run_NilRepoIsNoop(t *testing.T) {
	job := NewInertiaWeeklyBatch(nil, nil, nil, zap.NewNop())
	if err := job.Run(context.Background()); err != nil {
		t.Errorf("expected nil repo Run to be no-op, got %v", err)
	}
}

func TestInertiaWeeklyBatch_Name(t *testing.T) {
	job := NewInertiaWeeklyBatch(&stubInertiaActivePatientLister{}, nil, nil, zap.NewNop())
	if job.Name() != "inertia_weekly" {
		t.Errorf("expected name 'inertia_weekly', got %q", job.Name())
	}
}

func TestInertiaOrchestrator_Evaluate_NoInputProducesEmptyReport(t *testing.T) {
	orch := NewInertiaOrchestrator(nil, nil, nil, nil, nil, nil, zap.NewNop())
	report := orch.Evaluate(context.Background(), InertiaDetectorInput{PatientID: "p1"})
	if report.PatientID != "p1" {
		t.Errorf("expected report.PatientID=p1, got %q", report.PatientID)
	}
	if len(report.Verdicts) != 0 {
		t.Errorf("expected empty verdicts for minimal input, got %d", len(report.Verdicts))
	}
}
