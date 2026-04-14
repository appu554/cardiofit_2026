package services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// stubBatchJob is a BatchJob that records when it ran and lets tests
// drive cadence + behavior. Mirrors the KB-26 test stub from Phase 5 P5-3.
type stubBatchJob struct {
	name      string
	runs      atomic.Int32
	delay     time.Duration
	returnErr error
	shouldRun func(now time.Time) bool
	mu        sync.Mutex
	runTimes  []time.Time
}

func (s *stubBatchJob) Name() string { return s.name }

func (s *stubBatchJob) ShouldRun(ctx context.Context, now time.Time) bool {
	if s.shouldRun == nil {
		return true
	}
	return s.shouldRun(now)
}

func (s *stubBatchJob) Run(ctx context.Context) error {
	s.mu.Lock()
	s.runTimes = append(s.runTimes, time.Now())
	s.mu.Unlock()
	s.runs.Add(1)
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.returnErr
}

func TestBatchScheduler_RunOnceImmediately(t *testing.T) {
	job := &stubBatchJob{name: "test"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(job)

	if err := sched.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if job.runs.Load() != 1 {
		t.Errorf("expected 1 run, got %d", job.runs.Load())
	}
}

func TestBatchScheduler_RunOnce_SkipsJobsWhereShouldRunFalse(t *testing.T) {
	sched := NewBatchScheduler(zap.NewNop())
	runs := &stubBatchJob{name: "runs", shouldRun: func(now time.Time) bool { return true }}
	skips := &stubBatchJob{name: "skips", shouldRun: func(now time.Time) bool { return false }}
	sched.Register(runs)
	sched.Register(skips)

	if err := sched.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if runs.runs.Load() != 1 {
		t.Errorf("expected ShouldRun=true job to execute once, got %d", runs.runs.Load())
	}
	if skips.runs.Load() != 0 {
		t.Errorf("expected ShouldRun=false job to be skipped, got %d runs", skips.runs.Load())
	}
}

func TestBatchScheduler_OneJobErrors_OthersStillRun(t *testing.T) {
	sched := NewBatchScheduler(zap.NewNop())
	jobA := &stubBatchJob{name: "a", returnErr: context.Canceled}
	jobB := &stubBatchJob{name: "b"}
	sched.Register(jobA)
	sched.Register(jobB)

	_ = sched.RunOnce(context.Background())
	if jobA.runs.Load() != 1 || jobB.runs.Load() != 1 {
		t.Errorf("expected both jobs to run, got a=%d b=%d", jobA.runs.Load(), jobB.runs.Load())
	}
}

func TestBatchScheduler_Drain_NoOpWhenIdle(t *testing.T) {
	sched := NewBatchScheduler(zap.NewNop())
	drainStart := time.Now()
	sched.Drain()
	if time.Since(drainStart) > 10*time.Millisecond {
		t.Errorf("idle Drain blocked unexpectedly: %v", time.Since(drainStart))
	}
}
