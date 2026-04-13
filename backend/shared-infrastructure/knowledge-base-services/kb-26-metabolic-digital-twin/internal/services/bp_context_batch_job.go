package services

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
)

// BPContextClassifier is the narrow interface the batch job needs from
// the orchestrator. Defined here so tests can stub it without instantiating
// real KB-20/KB-21 clients.
type BPContextClassifier interface {
	Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error)
}

// BPContextDailyBatch classifies every active patient once per run.
// Active = twin_state.updated_at within the configured activeWindow.
// Implements the BatchJob interface registered with BatchScheduler.
type BPContextDailyBatch struct {
	repo         *BPContextRepository
	classifier   BPContextClassifier
	activeWindow time.Duration
	concurrency  int
	log          *zap.Logger
}

// NewBPContextDailyBatch constructs the job.
// concurrency controls the maximum number of patients classified in parallel.
// Values < 1 are clamped to 1.
func NewBPContextDailyBatch(
	repo *BPContextRepository,
	classifier BPContextClassifier,
	activeWindow time.Duration,
	concurrency int,
	log *zap.Logger,
) *BPContextDailyBatch {
	if concurrency < 1 {
		concurrency = 1
	}
	return &BPContextDailyBatch{
		repo:         repo,
		classifier:   classifier,
		activeWindow: activeWindow,
		concurrency:  concurrency,
		log:          log,
	}
}

// Name implements BatchJob.
func (j *BPContextDailyBatch) Name() string { return "bp_context_daily" }

// Run implements BatchJob. It fetches active patient IDs, classifies each
// with bounded concurrency, and tolerates per-patient errors (logged but
// not propagated to callers). Context cancellation is respected for
// graceful shutdown — the method returns context.Canceled as soon as the
// context is done.
func (j *BPContextDailyBatch) Run(ctx context.Context) error {
	// Fast-path: don't bother listing patients if already cancelled.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	patientIDs, err := j.repo.ListActivePatientIDs(j.activeWindow)
	if err != nil {
		return err
	}

	j.log.Info("BP context batch starting",
		zap.Int("patients", len(patientIDs)),
		zap.Int("concurrency", j.concurrency))

	if len(patientIDs) == 0 {
		j.log.Info("BP context batch complete — no active patients")
		return nil
	}

	// Bounded concurrency via a semaphore channel.
	sem := make(chan struct{}, j.concurrency)
	var wg sync.WaitGroup
	var processed, errored atomic.Int32

	for _, pid := range patientIDs {
		// Check cancellation before acquiring a semaphore slot.
		if ctx.Err() != nil {
			break
		}

		// Block until a worker slot is free or context is cancelled.
		select {
		case <-ctx.Done():
			// Do not acquire the slot; exit the loop.
		case sem <- struct{}{}:
			// Slot acquired — fall through to dispatch the goroutine.
		}

		// Re-check after the select in case we exited via context.Done().
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(patientID string) {
			defer wg.Done()
			defer func() { <-sem }()

			if _, classErr := j.classifier.Classify(ctx, patientID); classErr != nil {
				j.log.Warn("BP context classification failed in batch",
					zap.String("patient_id", patientID),
					zap.Error(classErr))
				errored.Add(1)
				return
			}
			processed.Add(1)
		}(pid)
	}

	// Wait for all in-flight goroutines to finish before deciding the outcome.
	wg.Wait()

	if ctx.Err() != nil {
		j.log.Warn("BP context batch cancelled",
			zap.Int32("processed", processed.Load()),
			zap.Int32("errored", errored.Load()),
			zap.Int("total", len(patientIDs)))
		return ctx.Err()
	}

	j.log.Info("BP context batch complete",
		zap.Int32("processed", processed.Load()),
		zap.Int32("errored", errored.Load()),
		zap.Int("total", len(patientIDs)))
	return nil
}
