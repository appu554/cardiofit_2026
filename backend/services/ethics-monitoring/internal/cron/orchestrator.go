// Package cron provides the in-process cron orchestrator for the
// ethics-monitoring service. It wraps github.com/robfig/cron/v3 and exposes a
// minimal Job interface so detector workers (daily/weekly/monthly) can be
// registered declaratively from main.go.
//
// VisibilityClass: AD (audit-defensible) — schedules govern when ethical
// detectors run, which has direct compliance implications under
// Guidelines §10.1.
package cron

import (
	"context"
	"fmt"
	"log"
	"sync"

	robfigcron "github.com/robfig/cron/v3"
)

// Job is the unit of work scheduled by the Orchestrator. Implementations must
// be safe for sequential reuse — robfig/cron will invoke Run() once per fire
// of the schedule.
type Job interface {
	// Name is a stable, human-readable identifier (used in logs).
	Name() string
	// Schedule returns a 5-field crontab expression (m h dom mon dow).
	Schedule() string
	// Run executes the job. The context is cancelled when the orchestrator
	// is stopped so jobs can abort cooperatively. Errors are logged by the
	// orchestrator; the scheduler continues regardless.
	Run(ctx context.Context) error
}

// Orchestrator wraps robfig/cron and tracks registered jobs.
//
// Thread-safety: Register, JobCount, Start, and Stop are all safe for
// concurrent use. The internal cron.Cron is itself thread-safe; the mutex
// guards only the bookkeeping counter and the running flag.
type Orchestrator struct {
	mu      sync.RWMutex
	c       *robfigcron.Cron
	ctx     context.Context
	cancel  context.CancelFunc
	count   int
	running bool
}

// New constructs a fresh Orchestrator. The underlying cron uses standard
// 5-field parsing (no seconds field). The orchestrator owns a cancellable
// context that is propagated to every job's Run; Stop() cancels it.
func New() *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Orchestrator{
		c:      robfigcron.New(),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Register adds j to the schedule. A malformed crontab expression returns an
// error and the job is not added. Safe to call before or after Start.
func (o *Orchestrator) Register(j Job) error {
	if j == nil {
		return fmt.Errorf("register: nil job")
	}
	name := j.Name()
	_, err := o.c.AddFunc(j.Schedule(), func() {
		if err := j.Run(o.ctx); err != nil {
			log.Printf("ethics-monitoring: job %q run error: %v", name, err)
		}
	})
	if err != nil {
		return fmt.Errorf("register %q: %w", name, err)
	}
	o.mu.Lock()
	o.count++
	o.mu.Unlock()
	return nil
}

// JobCount returns the number of jobs successfully registered.
func (o *Orchestrator) JobCount() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.count
}

// Start begins ticking the scheduler. Idempotent — repeated calls are no-ops
// after the first successful Start.
func (o *Orchestrator) Start() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.running {
		return nil
	}
	o.c.Start()
	o.running = true
	return nil
}

// Stop halts the scheduler and cancels the orchestrator context so in-flight
// jobs observing ctx.Done() can abort. It returns the drain context from
// robfig/cron whose Done channel closes once all currently-running job
// goroutines finish — callers should select on it (with a deadline of their
// choice) to await graceful drain. Safe to call without a prior Start (returns
// an already-closed context) and safe to call multiple times.
func (o *Orchestrator) Stop() context.Context {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.cancel()
	if !o.running {
		// Return an already-cancelled context so callers can select uniformly.
		closed, cancel := context.WithCancel(context.Background())
		cancel()
		return closed
	}
	drainCtx := o.c.Stop()
	o.running = false
	return drainCtx
}
