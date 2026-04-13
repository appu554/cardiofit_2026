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
// drive how long it takes / whether it errors.
type stubBatchJob struct {
	name      string
	runs      atomic.Int32
	delay     time.Duration
	returnErr error
	mu        sync.Mutex
	runTimes  []time.Time
}

func (s *stubBatchJob) Name() string { return s.name }

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

func TestBatchScheduler_RunOnce_MultipleJobs(t *testing.T) {
	jobA := &stubBatchJob{name: "a"}
	jobB := &stubBatchJob{name: "b"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(jobA)
	sched.Register(jobB)

	if err := sched.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if jobA.runs.Load() != 1 || jobB.runs.Load() != 1 {
		t.Errorf("expected both jobs to run once, got a=%d b=%d", jobA.runs.Load(), jobB.runs.Load())
	}
}

func TestBatchScheduler_OneJobErrors_OthersStillRun(t *testing.T) {
	jobA := &stubBatchJob{name: "a", returnErr: errSimulated()}
	jobB := &stubBatchJob{name: "b"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(jobA)
	sched.Register(jobB)

	// RunOnce should not bail on the first error — each job is isolated.
	_ = sched.RunOnce(context.Background())

	if jobA.runs.Load() != 1 {
		t.Errorf("job A should still have run, got %d", jobA.runs.Load())
	}
	if jobB.runs.Load() != 1 {
		t.Errorf("job B should run despite job A error, got %d", jobB.runs.Load())
	}
}

func TestBatchScheduler_StartLoop_RespectsContextCancel(t *testing.T) {
	job := &stubBatchJob{name: "test"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(job)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		sched.StartLoop(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Wait for at least one run, then cancel.
	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Good: scheduler exited on context cancel.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("scheduler did not exit within 500ms of context cancel")
	}

	if job.runs.Load() < 1 {
		t.Errorf("expected at least 1 run, got %d", job.runs.Load())
	}
}

func TestBatchScheduler_Drain_WaitsForInFlightRun(t *testing.T) {
	job := &stubBatchJob{name: "slow", delay: 200 * time.Millisecond}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(job)

	// Start a long-running RunOnce in a goroutine.
	runDone := make(chan struct{})
	go func() {
		_ = sched.RunOnce(context.Background())
		close(runDone)
	}()

	// Give the run a chance to start.
	time.Sleep(20 * time.Millisecond)

	// Drain should block until the in-flight run finishes.
	drainStart := time.Now()
	sched.Drain()
	drainDuration := time.Since(drainStart)

	if drainDuration < 100*time.Millisecond {
		t.Errorf("Drain returned too early: %v (expected at least 100ms)", drainDuration)
	}

	// RunOnce should have completed by now.
	select {
	case <-runDone:
		// Good
	case <-time.After(50 * time.Millisecond):
		t.Fatal("RunOnce did not complete after Drain returned")
	}
}

func TestBatchScheduler_Drain_NoOpWhenIdle(t *testing.T) {
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(&stubBatchJob{name: "idle"})

	// Drain when nothing is running should return immediately.
	drainStart := time.Now()
	sched.Drain()
	if time.Since(drainStart) > 10*time.Millisecond {
		t.Errorf("idle Drain blocked unexpectedly: %v", time.Since(drainStart))
	}
}
