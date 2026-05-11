package cron

import (
	"context"
	"sync"
	"testing"
)

type fakeJob struct {
	name     string
	schedule string
}

func (f fakeJob) Name() string                 { return f.name }
func (f fakeJob) Schedule() string             { return f.schedule }
func (f fakeJob) Run(_ context.Context) error  { return nil }

func TestOrchestrator_Register_IncrementsCount(t *testing.T) {
	o := New()
	if got := o.JobCount(); got != 0 {
		t.Fatalf("initial JobCount = %d, want 0", got)
	}
	if err := o.Register(fakeJob{name: "j1", schedule: "0 2 * * *"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := o.Register(fakeJob{name: "j2", schedule: "0 3 * * 1"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if got := o.JobCount(); got != 2 {
		t.Fatalf("JobCount = %d, want 2", got)
	}
}

func TestOrchestrator_Register_BadCronReturnsError(t *testing.T) {
	o := New()
	err := o.Register(fakeJob{name: "bad", schedule: "not a cron"})
	if err == nil {
		t.Fatal("expected error for malformed crontab, got nil")
	}
	if got := o.JobCount(); got != 0 {
		t.Fatalf("JobCount after failed Register = %d, want 0", got)
	}
}

func TestOrchestrator_Register_NilJob(t *testing.T) {
	o := New()
	if err := o.Register(nil); err == nil {
		t.Fatal("expected error for nil job, got nil")
	}
}

func TestOrchestrator_Lifecycle_Idempotent(t *testing.T) {
	o := New()
	// Stop before Start is safe and returns an already-closed context.
	preStartCtx := o.Stop()
	select {
	case <-preStartCtx.Done():
	default:
		t.Fatal("Stop before Start should return an already-cancelled context")
	}
	if err := o.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	// Second Start is a no-op.
	if err := o.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	drainCtx := o.Stop()
	if drainCtx == nil {
		t.Fatal("Stop returned nil context")
	}
	// Second Stop is a no-op (orchestrator already not-running).
	_ = o.Stop()
}

func TestOrchestrator_ConcurrentRegisterAndCount(t *testing.T) {
	o := New()
	const N = 50
	var wg sync.WaitGroup
	wg.Add(N * 2)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_ = o.Register(fakeJob{name: "j", schedule: "0 2 * * *"})
		}()
		go func() {
			defer wg.Done()
			_ = o.JobCount()
		}()
	}
	wg.Wait()
	if got := o.JobCount(); got != N {
		t.Fatalf("JobCount after %d concurrent Registers = %d", N, got)
	}
}
