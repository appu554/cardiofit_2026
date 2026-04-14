package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

// stubCGMActivePatientLister is a minimal lister for tests.
type stubCGMActivePatientLister struct {
	patients []CGMActivePatient
	err      error
}

func (s *stubCGMActivePatientLister) ListCGMActivePatientIDs(ctx context.Context) ([]CGMActivePatient, error) {
	return s.patients, s.err
}

// stubCGMReadingFetcher returns a constant set of readings per call.
type stubCGMReadingFetcher struct {
	readings  []GlucoseReading
	err       error
	callCount int
}

func (s *stubCGMReadingFetcher) FetchCGMReadings(ctx context.Context, patientID string, start, end time.Time) ([]GlucoseReading, error) {
	s.callCount++
	if s.err != nil {
		return nil, s.err
	}
	return s.readings, nil
}

func TestCGMDailyBatch_ShouldRun_OnlyAt01UTC(t *testing.T) {
	job := NewCGMDailyBatch(nil, nil, zap.NewNop())

	cases := []struct {
		name     string
		hour     int
		expected bool
	}{
		{"01:00 fires", 1, true},
		{"00:00 skips", 0, false},
		{"02:00 skips", 2, false},
		{"12:00 skips", 12, false},
		{"23:00 skips", 23, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			when := time.Date(2026, 4, 14, tc.hour, 0, 0, 0, time.UTC)
			got := job.ShouldRun(context.Background(), when)
			if got != tc.expected {
				t.Errorf("ShouldRun at hour %d = %v, want %v", tc.hour, got, tc.expected)
			}
		})
	}
}

func TestCGMDailyBatch_Run_NilRepo_Noop(t *testing.T) {
	job := NewCGMDailyBatch(nil, nil, zap.NewNop())
	if err := job.Run(context.Background()); err != nil {
		t.Errorf("expected nil for nil repo, got %v", err)
	}
}

func TestCGMDailyBatch_Run_HeartbeatMode(t *testing.T) {
	// Fetcher nil → heartbeat: counts patients due for a report but
	// doesn't fetch readings.
	now := time.Now().UTC()
	old := now.Add(-15 * 24 * time.Hour) // older than 14 days
	recent := now.Add(-3 * 24 * time.Hour)

	repo := &stubCGMActivePatientLister{
		patients: []CGMActivePatient{
			{PatientID: "p1", LastReportAt: &old},    // due
			{PatientID: "p2", LastReportAt: &recent}, // not due
			{PatientID: "p3", LastReportAt: nil},     // never — due
		},
	}
	job := NewCGMDailyBatch(repo, nil, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Errorf("expected nil in heartbeat mode, got %v", err)
	}
}

func TestCGMDailyBatch_Run_FullMode_SkipsRecentReports(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-15 * 24 * time.Hour)
	recent := now.Add(-3 * 24 * time.Hour)

	repo := &stubCGMActivePatientLister{
		patients: []CGMActivePatient{
			{PatientID: "p1", LastReportAt: &old},    // due
			{PatientID: "p2", LastReportAt: &recent}, // not due
			{PatientID: "p3", LastReportAt: nil},     // never — due
		},
	}
	fetcher := &stubCGMReadingFetcher{
		readings: []GlucoseReading{
			{Timestamp: now.Add(-5 * 24 * time.Hour), ValueMgDL: 140},
		},
	}
	job := NewCGMDailyBatch(repo, fetcher, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// Expect fetcher called for p1 and p3 (stale), not p2 (recent).
	if fetcher.callCount != 2 {
		t.Errorf("expected 2 fetcher calls (stale patients only), got %d", fetcher.callCount)
	}
}

func TestCGMDailyBatch_Run_FetchErrorIsPerPatientIsolated(t *testing.T) {
	now := time.Now().UTC()
	old := now.Add(-15 * 24 * time.Hour)
	repo := &stubCGMActivePatientLister{
		patients: []CGMActivePatient{
			{PatientID: "p1", LastReportAt: &old},
			{PatientID: "p2", LastReportAt: &old},
		},
	}
	fetcher := &stubCGMReadingFetcher{err: errors.New("simulated fetch failure")}
	job := NewCGMDailyBatch(repo, fetcher, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Errorf("expected nil (fetch errors should be isolated), got %v", err)
	}
	if fetcher.callCount != 2 {
		t.Errorf("expected fetcher called for both patients despite errors, got %d", fetcher.callCount)
	}
}

func TestCGMDailyBatch_Run_RepoError_Propagates(t *testing.T) {
	repo := &stubCGMActivePatientLister{err: errors.New("simulated DB error")}
	job := NewCGMDailyBatch(repo, nil, zap.NewNop())
	if err := job.Run(context.Background()); err == nil {
		t.Error("expected repo error to propagate from Run, got nil")
	}
}

func TestCGMDailyBatch_Name(t *testing.T) {
	job := NewCGMDailyBatch(nil, nil, zap.NewNop())
	if job.Name() != "cgm_daily" {
		t.Errorf("expected name 'cgm_daily', got %q", job.Name())
	}
}
