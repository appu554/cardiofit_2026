// Package workers provides background workers for KB-17 Population Registry
package workers

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/services"
)

// StatsRefreshWorker handles periodic statistics refresh
type StatsRefreshWorker struct {
	analyticsService *services.AnalyticsService
	logger           *logrus.Entry
	config           *StatsRefreshConfig
	stopCh           chan struct{}
	wg               sync.WaitGroup
	running          bool
	mu               sync.RWMutex
	lastRefresh      time.Time
}

// StatsRefreshConfig holds stats refresh worker configuration
type StatsRefreshConfig struct {
	Enabled         bool
	RefreshInterval time.Duration
	StaleThreshold  time.Duration
}

// DefaultStatsRefreshConfig returns default configuration
func DefaultStatsRefreshConfig() *StatsRefreshConfig {
	return &StatsRefreshConfig{
		Enabled:         true,
		RefreshInterval: 15 * time.Minute,
		StaleThreshold:  30 * time.Minute,
	}
}

// NewStatsRefreshWorker creates a new stats refresh worker
func NewStatsRefreshWorker(
	analyticsService *services.AnalyticsService,
	config *StatsRefreshConfig,
	logger *logrus.Entry,
) *StatsRefreshWorker {
	return &StatsRefreshWorker{
		analyticsService: analyticsService,
		config:           config,
		logger:           logger.WithField("worker", "stats_refresh"),
		stopCh:           make(chan struct{}),
	}
}

// Start starts the stats refresh worker
func (w *StatsRefreshWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	if !w.config.Enabled {
		w.logger.Info("Stats refresh worker is disabled")
		return nil
	}

	w.logger.Info("Starting stats refresh worker")

	w.wg.Add(1)
	go w.runLoop(ctx)

	return nil
}

// Stop stops the stats refresh worker
func (w *StatsRefreshWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.logger.Info("Stopping stats refresh worker")
	close(w.stopCh)
	w.wg.Wait()
	w.logger.Info("Stats refresh worker stopped")
}

func (w *StatsRefreshWorker) runLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.RefreshInterval)
	defer ticker.Stop()

	// Initial refresh
	w.refreshStats(ctx)

	for {
		select {
		case <-ticker.C:
			w.refreshStats(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *StatsRefreshWorker) refreshStats(ctx context.Context) {
	w.logger.Debug("Refreshing registry statistics")

	startTime := time.Now()

	err := w.analyticsService.RefreshAllStats(ctx)
	if err != nil {
		w.logger.WithError(err).Error("Failed to refresh statistics")
		return
	}

	w.mu.Lock()
	w.lastRefresh = time.Now()
	w.mu.Unlock()

	w.logger.WithFields(logrus.Fields{
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Registry statistics refreshed")
}

// ForceRefresh triggers an immediate stats refresh
func (w *StatsRefreshWorker) ForceRefresh(ctx context.Context) error {
	w.logger.Info("Force refresh triggered")
	w.refreshStats(ctx)
	return nil
}

// GetLastRefresh returns the time of the last successful refresh
func (w *StatsRefreshWorker) GetLastRefresh() time.Time {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastRefresh
}

// IsStale checks if statistics are stale
func (w *StatsRefreshWorker) IsStale() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	if w.lastRefresh.IsZero() {
		return true
	}

	return time.Since(w.lastRefresh) > w.config.StaleThreshold
}

// IsRunning returns whether the worker is running
func (w *StatsRefreshWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetStatus returns the worker status
func (w *StatsRefreshWorker) GetStatus() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return map[string]interface{}{
		"running":      w.running,
		"enabled":      w.config.Enabled,
		"last_refresh": w.lastRefresh,
		"is_stale":     w.IsStale(),
		"interval":     w.config.RefreshInterval.String(),
	}
}
