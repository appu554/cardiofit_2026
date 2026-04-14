package services

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
)

// stubInertiaPatientLister is a minimal implementation of
// InertiaActivePatientLister for tests. It records how many times it
// was called and returns a fixed list of IDs.
type stubInertiaPatientLister struct {
	ids  []string
	err  error
	hits int
}

func (s *stubInertiaPatientLister) ListActivePatientIDs(window time.Duration) ([]string, error) {
	s.hits++
	return s.ids, s.err
}

func TestInertiaWeeklyBatch_ShouldRun_OnlyOnMonday(t *testing.T) {
	repo := &stubInertiaPatientLister{}
	job := NewInertiaWeeklyBatch(repo, zap.NewNop())

	// 2026-04-13 is a Monday.
	monday := time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	if !job.ShouldRun(context.Background(), monday) {
		t.Error("expected ShouldRun=true on Monday")
	}
	// 2026-04-14 is a Tuesday.
	tuesday := time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	if job.ShouldRun(context.Background(), tuesday) {
		t.Error("expected ShouldRun=false on Tuesday")
	}
}

func TestInertiaWeeklyBatch_Run_ListsActivePatients(t *testing.T) {
	repo := &stubInertiaPatientLister{ids: []string{"p1", "p2", "p3"}}
	job := NewInertiaWeeklyBatch(repo, zap.NewNop())

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if repo.hits != 1 {
		t.Errorf("expected repo to be called once, got %d", repo.hits)
	}
}

func TestInertiaWeeklyBatch_Name(t *testing.T) {
	job := NewInertiaWeeklyBatch(&stubInertiaPatientLister{}, zap.NewNop())
	if job.Name() != "inertia_weekly" {
		t.Errorf("expected name 'inertia_weekly', got %q", job.Name())
	}
}
