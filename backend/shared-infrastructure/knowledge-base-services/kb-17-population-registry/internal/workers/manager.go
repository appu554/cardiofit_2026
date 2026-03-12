// Package workers provides background workers for KB-17 Population Registry
package workers

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/database"
	"kb-17-population-registry/internal/services"
)

// Manager coordinates all background workers
type Manager struct {
	autoEnrollmentWorker *AutoEnrollmentWorker
	statsRefreshWorker   *StatsRefreshWorker
	reevaluationWorker   *ReevaluationWorker

	logger  *logrus.Entry
	running bool
	mu      sync.RWMutex
}

// ManagerConfig holds configuration for all workers
type ManagerConfig struct {
	AutoEnrollment *AutoEnrollmentConfig
	StatsRefresh   *StatsRefreshConfig
	Reevaluation   *ReevaluationConfig
}

// DefaultManagerConfig returns default configuration for all workers
func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		AutoEnrollment: DefaultAutoEnrollmentConfig(),
		StatsRefresh:   DefaultStatsRefreshConfig(),
		Reevaluation:   DefaultReevaluationConfig(),
	}
}

// NewManager creates a new worker manager
func NewManager(
	repo *database.Repository,
	enrollmentService *services.EnrollmentService,
	evaluationService *services.EvaluationService,
	analyticsService *services.AnalyticsService,
	config *ManagerConfig,
	logger *logrus.Entry,
) *Manager {
	managerLogger := logger.WithField("component", "worker_manager")

	return &Manager{
		autoEnrollmentWorker: NewAutoEnrollmentWorker(
			enrollmentService,
			evaluationService,
			config.AutoEnrollment,
			managerLogger,
		),
		statsRefreshWorker: NewStatsRefreshWorker(
			analyticsService,
			config.StatsRefresh,
			managerLogger,
		),
		reevaluationWorker: NewReevaluationWorker(
			repo,
			enrollmentService,
			evaluationService,
			config.Reevaluation,
			managerLogger,
		),
		logger: managerLogger,
	}
}

// Start starts all workers
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.logger.Warn("Worker manager already running")
		return nil
	}

	m.logger.Info("Starting all background workers")

	// Start workers in parallel
	var wg sync.WaitGroup
	errCh := make(chan error, 3)

	wg.Add(3)

	go func() {
		defer wg.Done()
		if err := m.autoEnrollmentWorker.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := m.statsRefreshWorker.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := m.reevaluationWorker.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	wg.Wait()
	close(errCh)

	// Check for errors
	for err := range errCh {
		if err != nil {
			m.logger.WithError(err).Error("Failed to start worker")
			m.Stop()
			return err
		}
	}

	m.running = true
	m.logger.Info("All background workers started")

	return nil
}

// Stop stops all workers
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.logger.Info("Stopping all background workers")

	// Stop workers in parallel
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		m.autoEnrollmentWorker.Stop()
	}()

	go func() {
		defer wg.Done()
		m.statsRefreshWorker.Stop()
	}()

	go func() {
		defer wg.Done()
		m.reevaluationWorker.Stop()
	}()

	wg.Wait()

	m.running = false
	m.logger.Info("All background workers stopped")
}

// GetAutoEnrollmentWorker returns the auto-enrollment worker
func (m *Manager) GetAutoEnrollmentWorker() *AutoEnrollmentWorker {
	return m.autoEnrollmentWorker
}

// GetStatsRefreshWorker returns the stats refresh worker
func (m *Manager) GetStatsRefreshWorker() *StatsRefreshWorker {
	return m.statsRefreshWorker
}

// GetReevaluationWorker returns the re-evaluation worker
func (m *Manager) GetReevaluationWorker() *ReevaluationWorker {
	return m.reevaluationWorker
}

// GetStatus returns the status of all workers
func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"manager_running": m.running,
		"workers": map[string]interface{}{
			"auto_enrollment": map[string]interface{}{
				"running": m.autoEnrollmentWorker.IsRunning(),
			},
			"stats_refresh": m.statsRefreshWorker.GetStatus(),
			"reevaluation":  m.reevaluationWorker.GetStatus(),
		},
	}
}

// IsRunning returns whether the manager is running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// HealthCheck performs health check on all workers
func (m *Manager) HealthCheck() map[string]bool {
	return map[string]bool{
		"auto_enrollment": m.autoEnrollmentWorker.IsRunning(),
		"stats_refresh":   m.statsRefreshWorker.IsRunning(),
		"reevaluation":    m.reevaluationWorker.IsRunning(),
	}
}
