// Package workers provides background worker processes for KB-14 Care Navigator
package workers

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/cache"
	"kb-14-care-navigator/internal/clients"
	"kb-14-care-navigator/internal/config"
	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/services"
)

// WorkerManager manages all background workers
type WorkerManager struct {
	escalationWorker *EscalationWorker
	kb3SyncWorker    *KB3SyncWorker
	kb9SyncWorker    *KB9SyncWorker
	kb12SyncWorker   *KB12SyncWorker
	log              *logrus.Entry
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewWorkerManager creates a new WorkerManager
func NewWorkerManager(
	cfg *config.Config,
	taskRepo *database.TaskRepository,
	escalationRepo *database.EscalationRepository,
	escalationEngine *services.EscalationEngine,
	taskFactory *services.TaskFactory,
	kb3Client *clients.KB3Client,
	kb9Client *clients.KB9Client,
	kb12Client *clients.KB12Client,
	redisCache *cache.RedisCache,
) *WorkerManager {
	log := logrus.WithField("component", "worker-manager")

	// Create workers
	escalationWorker := NewEscalationWorker(
		taskRepo,
		escalationEngine,
		redisCache,
		time.Duration(cfg.Escalation.CheckIntervalSeconds)*time.Second,
	)

	kb3SyncWorker := NewKB3SyncWorker(
		kb3Client,
		taskRepo,
		taskFactory,
		time.Duration(cfg.Workers.SyncIntervalMinutes)*time.Minute,
	)

	kb9SyncWorker := NewKB9SyncWorker(
		kb9Client,
		taskRepo,
		taskFactory,
		time.Duration(cfg.Workers.SyncIntervalMinutes)*time.Minute,
	)

	kb12SyncWorker := NewKB12SyncWorker(
		kb12Client,
		taskRepo,
		taskFactory,
		time.Duration(cfg.Workers.SyncIntervalMinutes)*time.Minute,
	)

	return &WorkerManager{
		escalationWorker: escalationWorker,
		kb3SyncWorker:    kb3SyncWorker,
		kb9SyncWorker:    kb9SyncWorker,
		kb12SyncWorker:   kb12SyncWorker,
		log:              log,
	}
}

// Start starts all workers
func (m *WorkerManager) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	m.log.Info("Starting all workers")

	// Start escalation worker
	if err := m.escalationWorker.Start(m.ctx); err != nil {
		return err
	}

	// Start sync workers
	if err := m.kb3SyncWorker.Start(m.ctx); err != nil {
		m.log.WithError(err).Warn("Failed to start KB-3 sync worker")
	}

	if err := m.kb9SyncWorker.Start(m.ctx); err != nil {
		m.log.WithError(err).Warn("Failed to start KB-9 sync worker")
	}

	if err := m.kb12SyncWorker.Start(m.ctx); err != nil {
		m.log.WithError(err).Warn("Failed to start KB-12 sync worker")
	}

	m.log.Info("All workers started")
	return nil
}

// Stop stops all workers
func (m *WorkerManager) Stop() {
	m.log.Info("Stopping all workers")

	if m.cancel != nil {
		m.cancel()
	}

	m.escalationWorker.Stop()
	m.kb3SyncWorker.Stop()
	m.kb9SyncWorker.Stop()
	m.kb12SyncWorker.Stop()

	m.wg.Wait()
	m.log.Info("All workers stopped")
}

// TriggerSync manually triggers a sync for all sources
func (m *WorkerManager) TriggerSync(ctx context.Context) {
	m.log.Info("Triggering manual sync for all sources")

	var wg sync.WaitGroup

	// Sync KB-3
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.kb3SyncWorker.sync(ctx)
	}()

	// Sync KB-9
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.kb9SyncWorker.sync(ctx)
	}()

	// Sync KB-12
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.kb12SyncWorker.sync(ctx)
	}()

	wg.Wait()
	m.log.Info("Manual sync completed for all sources")
}

// TriggerEscalationCheck manually triggers an escalation check
func (m *WorkerManager) TriggerEscalationCheck(ctx context.Context) {
	m.log.Info("Triggering manual escalation check")
	m.escalationWorker.checkEscalations(ctx)
	m.log.Info("Manual escalation check completed")
}
