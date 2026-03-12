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

// KB3SyncWorker periodically syncs tasks from KB-3 Temporal Service
type KB3SyncWorker struct {
	client      *clients.KB3Client
	taskRepo    *database.TaskRepository
	taskFactory *services.TaskFactory
	log         *logrus.Entry
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// NewKB3SyncWorker creates a new KB3SyncWorker
func NewKB3SyncWorker(
	client *clients.KB3Client,
	taskRepo *database.TaskRepository,
	taskFactory *services.TaskFactory,
	interval time.Duration,
) *KB3SyncWorker {
	return &KB3SyncWorker{
		client:      client,
		taskRepo:    taskRepo,
		taskFactory: taskFactory,
		log:         logrus.WithField("worker", "kb3-sync"),
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the KB-3 sync worker
func (w *KB3SyncWorker) Start(ctx context.Context) error {
	if !w.client.IsEnabled() {
		w.log.Info("KB-3 client disabled, sync worker not starting")
		return nil
	}

	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	w.log.WithField("interval", w.interval).Info("Starting KB-3 sync worker")

	w.wg.Add(1)
	go w.run(ctx)

	return nil
}

// Stop stops the KB-3 sync worker
func (w *KB3SyncWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()
	w.log.Info("KB-3 sync worker stopped")
}

// run is the main worker loop
func (w *KB3SyncWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.sync(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Context cancelled, stopping KB-3 sync worker")
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.sync(ctx)
		}
	}
}

// sync syncs tasks from KB-3
func (w *KB3SyncWorker) sync(ctx context.Context) {
	startTime := time.Now()
	w.log.Debug("Starting KB-3 sync")

	var createdCount, errorCount int

	// Sync overdue alerts
	alerts, err := w.client.GetOverdueAlerts(ctx)
	if err != nil {
		w.log.WithError(err).Error("Failed to fetch KB-3 overdue alerts")
	} else {
		for _, alert := range alerts {
			// Check if task already exists
			existing, _ := w.taskRepo.FindBySource(ctx, models.TaskSourceKB3, alert.AlertID)
			if len(existing) > 0 {
				continue
			}

			// Create task from alert
			_, err := w.taskFactory.CreateFromTemporalAlert(ctx, &alert)
			if err != nil {
				w.log.WithError(err).WithField("alert_id", alert.AlertID).Error("Failed to create task from alert")
				errorCount++
				continue
			}
			createdCount++
		}
	}

	// Sync upcoming deadlines (next 24 hours)
	deadlines, err := w.client.GetUpcomingDeadlines(ctx, 24)
	if err != nil {
		w.log.WithError(err).Error("Failed to fetch KB-3 deadlines")
	} else {
		for _, deadline := range deadlines {
			existing, _ := w.taskRepo.FindBySource(ctx, models.TaskSourceKB3, deadline.DeadlineID)
			if len(existing) > 0 {
				continue
			}

			_, err := w.taskFactory.CreateFromProtocolDeadline(ctx, &deadline)
			if err != nil {
				w.log.WithError(err).WithField("deadline_id", deadline.DeadlineID).Error("Failed to create task from deadline")
				errorCount++
				continue
			}
			createdCount++
		}
	}

	// Sync monitoring overdue
	overdueItems, err := w.client.GetMonitoringOverdue(ctx)
	if err != nil {
		w.log.WithError(err).Error("Failed to fetch KB-3 monitoring overdue")
	} else {
		for _, item := range overdueItems {
			existing, _ := w.taskRepo.FindBySource(ctx, models.TaskSourceKB3, item.OverdueID)
			if len(existing) > 0 {
				continue
			}

			_, err := w.taskFactory.CreateFromMonitoringOverdue(ctx, &item)
			if err != nil {
				w.log.WithError(err).WithField("overdue_id", item.OverdueID).Error("Failed to create task from monitoring overdue")
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
	}).Info("KB-3 sync completed")
}
