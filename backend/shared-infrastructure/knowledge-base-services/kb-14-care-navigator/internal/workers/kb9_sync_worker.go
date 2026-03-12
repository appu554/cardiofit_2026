// Package workers provides background worker processes for KB-14 Care Navigator
package workers

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/clients"
	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/models"
	"kb-14-care-navigator/internal/services"
)

// KB9SyncWorker periodically syncs tasks from KB-9 Care Gaps Service
type KB9SyncWorker struct {
	client      *clients.KB9Client
	taskRepo    *database.TaskRepository
	taskFactory *services.TaskFactory
	log         *logrus.Entry
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// NewKB9SyncWorker creates a new KB9SyncWorker
func NewKB9SyncWorker(
	client *clients.KB9Client,
	taskRepo *database.TaskRepository,
	taskFactory *services.TaskFactory,
	interval time.Duration,
) *KB9SyncWorker {
	return &KB9SyncWorker{
		client:      client,
		taskRepo:    taskRepo,
		taskFactory: taskFactory,
		log:         logrus.WithField("worker", "kb9-sync"),
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the KB-9 sync worker
func (w *KB9SyncWorker) Start(ctx context.Context) error {
	if !w.client.IsEnabled() {
		w.log.Info("KB-9 client disabled, sync worker not starting")
		return nil
	}

	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	w.log.WithField("interval", w.interval).Info("Starting KB-9 sync worker")

	w.wg.Add(1)
	go w.run(ctx)

	return nil
}

// Stop stops the KB-9 sync worker
func (w *KB9SyncWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()
	w.log.Info("KB-9 sync worker stopped")
}

// run is the main worker loop
func (w *KB9SyncWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.sync(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Context cancelled, stopping KB-9 sync worker")
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.sync(ctx)
		}
	}
}

// sync syncs tasks from KB-9
func (w *KB9SyncWorker) sync(ctx context.Context) {
	startTime := time.Now()
	w.log.Debug("Starting KB-9 sync")

	var createdCount, errorCount int

	// Get open care gaps
	gaps, err := w.client.GetOpenCareGaps(ctx)
	if err != nil {
		w.log.WithError(err).Error("Failed to fetch KB-9 care gaps")
		return
	}

	for _, gap := range gaps {
		// Check if task already exists
		existing, _ := w.taskRepo.FindBySource(ctx, models.TaskSourceKB9, gap.GapID)
		if len(existing) > 0 {
			continue
		}

		// Create task from care gap
		_, err := w.taskFactory.CreateFromCareGap(ctx, &gap)
		if err != nil {
			w.log.WithError(err).WithField("gap_id", gap.GapID).Error("Failed to create task from care gap")
			errorCount++
			continue
		}
		createdCount++
	}

	// Get high priority gaps separately
	highPriorityGaps, err := w.client.GetHighPriorityCareGaps(ctx)
	if err != nil {
		w.log.WithError(err).Error("Failed to fetch KB-9 high priority care gaps")
	} else {
		for _, gap := range highPriorityGaps {
			existing, _ := w.taskRepo.FindBySource(ctx, models.TaskSourceKB9, gap.GapID)
			if len(existing) > 0 {
				continue
			}

			_, err := w.taskFactory.CreateFromCareGap(ctx, &gap)
			if err != nil {
				w.log.WithError(err).WithField("gap_id", gap.GapID).Error("Failed to create task from high priority gap")
				errorCount++
				continue
			}
			createdCount++
		}
	}

	duration := time.Since(startTime)
	w.log.WithFields(logrus.Fields{
		"created":  createdCount,
		"errors":   errorCount,
		"duration": duration,
	}).Info("KB-9 sync completed")
}
