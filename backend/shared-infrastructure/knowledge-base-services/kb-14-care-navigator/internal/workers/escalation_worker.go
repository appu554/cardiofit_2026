// Package workers provides background worker processes for KB-14 Care Navigator
package workers

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/cache"
	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/services"
)

// EscalationWorker periodically checks for tasks that need escalation
type EscalationWorker struct {
	taskRepo         *database.TaskRepository
	escalationEngine *services.EscalationEngine
	cache            *cache.RedisCache
	log              *logrus.Entry
	interval         time.Duration
	stopCh           chan struct{}
	wg               sync.WaitGroup
	running          bool
	mu               sync.Mutex
}

// NewEscalationWorker creates a new EscalationWorker
func NewEscalationWorker(
	taskRepo *database.TaskRepository,
	escalationEngine *services.EscalationEngine,
	redisCache *cache.RedisCache,
	interval time.Duration,
) *EscalationWorker {
	return &EscalationWorker{
		taskRepo:         taskRepo,
		escalationEngine: escalationEngine,
		cache:            redisCache,
		log:              logrus.WithField("worker", "escalation"),
		interval:         interval,
		stopCh:           make(chan struct{}),
	}
}

// Start starts the escalation worker
func (w *EscalationWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	w.log.WithField("interval", w.interval).Info("Starting escalation worker")

	w.wg.Add(1)
	go w.run(ctx)

	return nil
}

// Stop stops the escalation worker
func (w *EscalationWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	w.wg.Wait()
	w.log.Info("Escalation worker stopped")
}

// run is the main worker loop
func (w *EscalationWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.checkEscalations(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Context cancelled, stopping escalation worker")
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkEscalations(ctx)
		}
	}
}

// checkEscalations checks all active tasks for escalation needs
func (w *EscalationWorker) checkEscalations(ctx context.Context) {
	startTime := time.Now()
	w.log.Debug("Starting escalation check")

	// Use the EscalationEngine's built-in check method which handles
	// finding tasks needing escalation and creating escalations
	escalatedCount, err := w.escalationEngine.CheckAndEscalate(ctx)
	if err != nil {
		w.log.WithError(err).Error("Failed to run escalation check")
		return
	}

	duration := time.Since(startTime)
	w.log.WithFields(logrus.Fields{
		"escalated": escalatedCount,
		"duration":  duration,
	}).Info("Escalation check completed")
}
