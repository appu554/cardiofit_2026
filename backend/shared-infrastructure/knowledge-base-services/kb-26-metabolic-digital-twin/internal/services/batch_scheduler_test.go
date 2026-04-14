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
//
// Phase 5 P5-3: adds a shouldRun predicate so the stub can participate
// in tests that exercise the new ShouldRun gate. When shouldRun is nil,
// the stub returns true (pre-P5-3 behaviour — fires on every tick).
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

// ---------------------------------------------------------------------------
// Phase 5 P5-3: ShouldRun gate tests.
// ---------------------------------------------------------------------------

func TestBatchScheduler_RunOnce_SkipsJobsWhereShouldRunFalse(t *testing.T) {
	sched := NewBatchScheduler(zap.NewNop())
	runs := &stubBatchJob{
		name:      "runs",
		shouldRun: func(now time.Time) bool { return true },
	}
	skips := &stubBatchJob{
		name:      "skips",
		shouldRun: func(now time.Time) bool { return false },
	}
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

func TestBatchScheduler_RunOnce_TwoConsumers_DifferentCadences(t *testing.T) {
	// Simulates one BP-daily consumer (fires at hour 2) + one weekly
	// consumer (fires on Mondays). At Monday 02:00 both should run; at
	// Tuesday 02:00 only BP; at Monday 09:00 only weekly.
	sched := NewBatchScheduler(zap.NewNop())

	var simulatedNow time.Time
	daily := &stubBatchJob{
		name:      "bp_daily",
		shouldRun: func(_ time.Time) bool { return simulatedNow.Hour() == 2 },
	}
	weekly := &stubBatchJob{
		name:      "inertia_weekly",
		shouldRun: func(_ time.Time) bool { return simulatedNow.Weekday() == time.Monday && simulatedNow.Hour() == 2 },
	}
	sched.Register(daily)
	sched.Register(weekly)

	// Monday 02:00 UTC — 2026-04-13 is a Monday.
	simulatedNow = time.Date(2026, 4, 13, 2, 0, 0, 0, time.UTC)
	_ = sched.RunOnce(context.Background())
	if daily.runs.Load() != 1 {
		t.Errorf("Monday 02:00: expected daily to fire, got %d", daily.runs.Load())
	}
	if weekly.runs.Load() != 1 {
		t.Errorf("Monday 02:00: expected weekly to fire, got %d", weekly.runs.Load())
	}

	// Tuesday 02:00 UTC — only daily fires.
	simulatedNow = time.Date(2026, 4, 14, 2, 0, 0, 0, time.UTC)
	_ = sched.RunOnce(context.Background())
	if daily.runs.Load() != 2 {
		t.Errorf("Tuesday 02:00: expected daily total=2, got %d", daily.runs.Load())
	}
	if weekly.runs.Load() != 1 {
		t.Errorf("Tuesday 02:00: expected weekly total still=1 (skipped), got %d", weekly.runs.Load())
	}

	// Monday 09:00 UTC — neither fires (weekly is gated on 02:00, daily on 02:00).
	simulatedNow = time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)
	_ = sched.RunOnce(context.Background())
	if daily.runs.Load() != 2 {
		t.Errorf("Monday 09:00: expected daily total still=2, got %d", daily.runs.Load())
	}
	if weekly.runs.Load() != 1 {
		t.Errorf("Monday 09:00: expected weekly total still=1, got %d", weekly.runs.Load())
	}
}

// Phase 5 P5-3: BPContextDailyBatch gains a BatchHourUTC field and
// implements ShouldRun returning true only when the current hour matches.
func TestBPContextDailyBatch_ShouldRun_OnlyAtConfiguredHour(t *testing.T) {
	// ShouldRun only reads BatchHourUTC; repo/classifier aren't needed.
	job := &BPContextDailyBatch{BatchHourUTC: 2}

	if !job.ShouldRun(context.Background(), time.Date(2026, 4, 13, 2, 0, 0, 0, time.UTC)) {
		t.Error("expected ShouldRun=true at hour 2")
	}
	if job.ShouldRun(context.Background(), time.Date(2026, 4, 13, 9, 0, 0, 0, time.UTC)) {
		t.Error("expected ShouldRun=false at hour 9")
	}
}
