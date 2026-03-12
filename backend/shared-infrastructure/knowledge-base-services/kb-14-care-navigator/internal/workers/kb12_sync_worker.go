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

// KB12SyncWorker periodically syncs tasks from KB-12 Order Sets/Care Plans Service
type KB12SyncWorker struct {
	client      *clients.KB12Client
	taskRepo    *database.TaskRepository
	taskFactory *services.TaskFactory
	log         *logrus.Entry
	interval    time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
	running     bool
	mu          sync.Mutex
}

// NewKB12SyncWorker creates a new KB12SyncWorker
func NewKB12SyncWorker(
	client *clients.KB12Client,
	taskRepo *database.TaskRepository,
	taskFactory *services.TaskFactory,
	interval time.Duration,
) *KB12SyncWorker {
	return &KB12SyncWorker{
		client:      client,
		taskRepo:    taskRepo,
		taskFactory: taskFactory,
		log:         logrus.WithField("worker", "kb12-sync"),
		interval:    interval,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the KB-12 sync worker
func (w *KB12SyncWorker) Start(ctx context.Context) error {
	if !w.client.IsEnabled() {
		w.log.Info("KB-12 client disabled, sync worker not starting")
		return nil
	}

	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	w.log.WithField("interval", w.interval).Info("Starting KB-12 sync worker")

	w.wg.Add(1)
	go w.run(ctx)

	return nil
}

// Stop stops the KB-12 sync worker
func (w *KB12SyncWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()
	w.log.Info("KB-12 sync worker stopped")
}

// run is the main worker loop
func (w *KB12SyncWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.sync(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Context cancelled, stopping KB-12 sync worker")
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.sync(ctx)
		}
	}
}

// sync syncs tasks from KB-12
func (w *KB12SyncWorker) sync(ctx context.Context) {
	startTime := time.Now()
	w.log.Debug("Starting KB-12 sync")

	var createdCount, errorCount int

	// Get overdue activities (doesn't require patient ID)
	overdueActivities, err := w.client.GetOverdueActivities(ctx)
	if err != nil {
		w.log.WithError(err).Error("Failed to fetch KB-12 overdue activities")
	} else {
		for _, activity := range overdueActivities {
			// Check if task already exists
			existing, _ := w.taskRepo.FindBySource(ctx, models.TaskSourceKB12, activity.ActivityID)
			if len(existing) > 0 {
				continue
			}

			// Create task from activity (use CarePlanID as the plan ID)
			_, err := w.taskFactory.CreateFromCarePlanActivity(ctx, activity.CarePlanID, &activity)
			if err != nil {
				w.log.WithError(err).WithField("activity_id", activity.ActivityID).Error("Failed to create task from overdue activity")
				errorCount++
				continue
			}
			createdCount++
		}
	}

	// Get activities due soon (next 3 days)
	dueSoonActivities, err := w.client.GetActivitiesDueSoon(ctx, 3)
	if err != nil {
		w.log.WithError(err).Error("Failed to fetch KB-12 activities due soon")
	} else {
		for _, activity := range dueSoonActivities {
			// Check if task already exists
			existing, _ := w.taskRepo.FindBySource(ctx, models.TaskSourceKB12, activity.ActivityID)
			if len(existing) > 0 {
				continue
			}

			// Create task from activity
			_, err := w.taskFactory.CreateFromCarePlanActivity(ctx, activity.CarePlanID, &activity)
			if err != nil {
				w.log.WithError(err).WithField("activity_id", activity.ActivityID).Error("Failed to create task from due soon activity")
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
	}).Info("KB-12 sync completed")
}
