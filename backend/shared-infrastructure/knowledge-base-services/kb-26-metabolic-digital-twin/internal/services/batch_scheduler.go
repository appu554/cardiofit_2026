package services

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BatchJob is the contract for any job runnable by BatchScheduler.
// Implementations must be safe to call from a long-lived goroutine and
// must respect context cancellation for graceful shutdown.
type BatchJob interface {
	// Name returns a human-readable identifier for the job, used in logs
	// and metrics labels.
	Name() string

	// Run executes one full pass of the job. Implementations must process
	// all relevant entities and return nil only on full success. Errors
	// are logged by the scheduler but do not block other registered jobs.
	Run(ctx context.Context) error
}

// BatchScheduler runs registered BatchJobs on a daily cadence.
// Phase 3 registers exactly one job (BPContextDailyBatch); the interface
// is intentionally generic so future jobs (e.g. quarterly_aggregator)
// can be added without rewriting the scheduler.
type BatchScheduler struct {
	jobs  []BatchJob
	mu    sync.RWMutex
	log   *zap.Logger
	runWg sync.WaitGroup
}

// NewBatchScheduler constructs an empty scheduler.
func NewBatchScheduler(log *zap.Logger) *BatchScheduler {
	return &BatchScheduler{log: log}
}

// Register adds a job to the scheduler. Call before StartLoop or RunOnce.
// Registration is goroutine-safe but not expected to happen after start.
func (s *BatchScheduler) Register(job BatchJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
}

// RunOnce executes every registered job sequentially. One job's error
// does NOT prevent subsequent jobs from running — each is isolated.
// Returns the FIRST error encountered, or nil if all succeeded.
func (s *BatchScheduler) RunOnce(ctx context.Context) error {
	s.runWg.Add(1)
	defer s.runWg.Done()

	s.mu.RLock()
	jobs := make([]BatchJob, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.RUnlock()

	var firstErr error
	for _, job := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.log.Info("batch job starting", zap.String("job", job.Name()))
		start := time.Now()
		err := job.Run(ctx)
		duration := time.Since(start)
		if err != nil {
			s.log.Error("batch job failed",
				zap.String("job", job.Name()),
				zap.Duration("duration", duration),
				zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.log.Info("batch job complete",
			zap.String("job", job.Name()),
			zap.Duration("duration", duration))
	}
	return firstErr
}

// StartLoop runs RunOnce on a fixed interval until ctx is cancelled.
// The interval is the wake cadence — production wires this to a daily
// hour-aligned interval, but the function takes a generic Duration so
// tests can drive it faster. The first run happens after one interval
// has elapsed (not immediately at start).
func (s *BatchScheduler) StartLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.log.Info("batch scheduler started", zap.Duration("interval", interval))
	for {
		select {
		case <-ctx.Done():
			s.log.Info("batch scheduler stopped")
			return
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil {
				s.log.Warn("scheduled batch run had errors",
					zap.Error(err))
			}
		}
	}
}

// Drain blocks until any currently-executing RunOnce calls return. Idle
// schedulers return immediately. Used by main.go during graceful shutdown
// to ensure in-flight batch work completes (or is interrupted via context
// cancellation) before the process exits.
func (s *BatchScheduler) Drain() {
	s.runWg.Wait()
}
