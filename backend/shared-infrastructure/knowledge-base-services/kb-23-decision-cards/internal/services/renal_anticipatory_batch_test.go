package services

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// stubRenalActivePatientLister is a minimal implementation of
// RenalActivePatientLister for tests.
type stubRenalActivePatientLister struct {
	ids  []string
	err  error
	hits int
}

func (s *stubRenalActivePatientLister) ListRenalActivePatientIDs(ctx context.Context) ([]string, error) {
	s.hits++
	return s.ids, s.err
}

func TestRenalAnticipatoryBatch_ShouldRun_OnlyFirstOfMonthAt04UTC(t *testing.T) {
	job := NewRenalAnticipatoryBatch(&stubRenalActivePatientLister{}, nil, nil, nil, nil, nil, nil, zap.NewNop())

	cases := []struct {
		name     string
		when     time.Time
		expected bool
	}{
		{"1st 04:00 UTC", time.Date(2026, 5, 1, 4, 0, 0, 0, time.UTC), true},
		{"1st 03:00 UTC (wrong hour)", time.Date(2026, 5, 1, 3, 0, 0, 0, time.UTC), false},
		{"1st 05:00 UTC (wrong hour)", time.Date(2026, 5, 1, 5, 0, 0, 0, time.UTC), false},
		{"2nd 04:00 UTC (wrong day)", time.Date(2026, 5, 2, 4, 0, 0, 0, time.UTC), false},
		{"15th 04:00 UTC (mid-month)", time.Date(2026, 5, 15, 4, 0, 0, 0, time.UTC), false},
		{"end of month 04:00 UTC", time.Date(2026, 5, 31, 4, 0, 0, 0, time.UTC), false},
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

func TestRenalAnticipatoryBatch_Run_ListsActivePatients(t *testing.T) {
	repo := &stubRenalActivePatientLister{ids: []string{"p1", "p2", "p3"}}
	job := NewRenalAnticipatoryBatch(repo, nil, nil, nil, nil, nil, nil, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if repo.hits != 1 {
		t.Errorf("expected repo to be called once, got %d", repo.hits)
	}
}

func TestRenalAnticipatoryBatch_Run_PropagatesRepoError(t *testing.T) {
	repo := &stubRenalActivePatientLister{err: context.DeadlineExceeded}
	job := NewRenalAnticipatoryBatch(repo, nil, nil, nil, nil, nil, nil, zap.NewNop())

	if err := job.Run(context.Background()); err == nil {
		t.Error("expected Run to propagate repo error, got nil")
	}
}

func TestRenalAnticipatoryBatch_Run_NilRepoIsNoop(t *testing.T) {
	job := NewRenalAnticipatoryBatch(nil, nil, nil, nil, nil, nil, nil, zap.NewNop())
	if err := job.Run(context.Background()); err != nil {
		t.Errorf("expected nil repo Run to be no-op, got %v", err)
	}
}

func TestRenalAnticipatoryBatch_Name(t *testing.T) {
	job := NewRenalAnticipatoryBatch(&stubRenalActivePatientLister{}, nil, nil, nil, nil, nil, nil, zap.NewNop())
	if job.Name() != "renal_anticipatory_monthly" {
		t.Errorf("expected name 'renal_anticipatory_monthly', got %q", job.Name())
	}
}
