package workers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultAutoEnrollmentConfig(t *testing.T) {
	config := DefaultAutoEnrollmentConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 5*time.Minute, config.CheckInterval)
	assert.Equal(t, 100, config.BatchSize)
	assert.Equal(t, 3, config.WorkerCount)
	assert.Equal(t, 30*time.Second, config.RetryDelay)
	assert.Equal(t, 3, config.MaxRetries)
}

func TestDefaultStatsRefreshConfig(t *testing.T) {
	config := DefaultStatsRefreshConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 15*time.Minute, config.RefreshInterval)
	assert.Equal(t, 30*time.Minute, config.StaleThreshold)
}

func TestDefaultReevaluationConfig(t *testing.T) {
	config := DefaultReevaluationConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 1*time.Hour, config.CheckInterval)
	assert.Equal(t, 50, config.BatchSize)
	assert.Equal(t, 24*time.Hour, config.StaleThreshold)
	assert.Equal(t, 2, config.WorkerCount)
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()

	assert.NotNil(t, config.AutoEnrollment)
	assert.NotNil(t, config.StatsRefresh)
	assert.NotNil(t, config.Reevaluation)

	assert.True(t, config.AutoEnrollment.Enabled)
	assert.True(t, config.StatsRefresh.Enabled)
	assert.True(t, config.Reevaluation.Enabled)
}

func TestReevaluationStats(t *testing.T) {
	stats := ReevaluationStats{
		TotalEvaluated:    100,
		LastRunTime:       time.Now(),
		LastRunDuration:   5 * time.Second,
		PatientsProcessed: 50,
		RiskTierChanges:   10,
		Errors:            2,
	}

	assert.Equal(t, int64(100), stats.TotalEvaluated)
	assert.Equal(t, 50, stats.PatientsProcessed)
	assert.Equal(t, 10, stats.RiskTierChanges)
	assert.Equal(t, 2, stats.Errors)
	assert.Equal(t, 5*time.Second, stats.LastRunDuration)
}

func TestAutoEnrollmentWorker_IsRunning(t *testing.T) {
	config := &AutoEnrollmentConfig{
		Enabled: false,
	}

	worker := &AutoEnrollmentWorker{
		config:  config,
		running: false,
	}

	assert.False(t, worker.IsRunning())

	worker.running = true
	assert.True(t, worker.IsRunning())
}

func TestStatsRefreshWorker_IsStale(t *testing.T) {
	config := &StatsRefreshConfig{
		StaleThreshold: 30 * time.Minute,
	}

	worker := &StatsRefreshWorker{
		config: config,
	}

	// No last refresh should be stale
	assert.True(t, worker.IsStale())

	// Recent refresh should not be stale
	worker.lastRefresh = time.Now()
	assert.False(t, worker.IsStale())

	// Old refresh should be stale
	worker.lastRefresh = time.Now().Add(-1 * time.Hour)
	assert.True(t, worker.IsStale())
}

func TestStatsRefreshWorker_GetStatus(t *testing.T) {
	config := &StatsRefreshConfig{
		Enabled:         true,
		RefreshInterval: 15 * time.Minute,
		StaleThreshold:  30 * time.Minute,
	}

	worker := &StatsRefreshWorker{
		config:      config,
		running:     true,
		lastRefresh: time.Now(),
	}

	status := worker.GetStatus()

	assert.True(t, status["running"].(bool))
	assert.True(t, status["enabled"].(bool))
	assert.NotNil(t, status["last_refresh"])
	assert.Equal(t, "15m0s", status["interval"].(string))
}

func TestReevaluationWorker_GetStatus(t *testing.T) {
	config := &ReevaluationConfig{
		Enabled: true,
	}

	worker := &ReevaluationWorker{
		config:  config,
		running: true,
		stats: ReevaluationStats{
			TotalEvaluated:    50,
			PatientsProcessed: 25,
			RiskTierChanges:   5,
			Errors:            1,
		},
	}

	status := worker.GetStatus()

	assert.True(t, status["running"].(bool))
	assert.True(t, status["enabled"].(bool))
	assert.Equal(t, int64(50), status["total_evaluated"].(int64))
	assert.Equal(t, 25, status["patients_processed"].(int))
	assert.Equal(t, 5, status["risk_tier_changes"].(int))
	assert.Equal(t, 1, status["errors"].(int))
}

func TestManager_GetStatus(t *testing.T) {
	autoEnrollConfig := &AutoEnrollmentConfig{Enabled: true}
	statsRefreshConfig := &StatsRefreshConfig{Enabled: true, RefreshInterval: 15 * time.Minute, StaleThreshold: 30 * time.Minute}
	reevalConfig := &ReevaluationConfig{Enabled: true}

	manager := &Manager{
		autoEnrollmentWorker: &AutoEnrollmentWorker{config: autoEnrollConfig, running: true},
		statsRefreshWorker:   &StatsRefreshWorker{config: statsRefreshConfig, running: true},
		reevaluationWorker:   &ReevaluationWorker{config: reevalConfig, running: true},
		running:              true,
	}

	status := manager.GetStatus()

	assert.True(t, status["manager_running"].(bool))

	workers := status["workers"].(map[string]interface{})
	assert.NotNil(t, workers["auto_enrollment"])
	assert.NotNil(t, workers["stats_refresh"])
	assert.NotNil(t, workers["reevaluation"])
}

func TestManager_HealthCheck(t *testing.T) {
	autoEnrollConfig := &AutoEnrollmentConfig{Enabled: true}
	statsRefreshConfig := &StatsRefreshConfig{Enabled: true, RefreshInterval: 15 * time.Minute, StaleThreshold: 30 * time.Minute}
	reevalConfig := &ReevaluationConfig{Enabled: true}

	manager := &Manager{
		autoEnrollmentWorker: &AutoEnrollmentWorker{config: autoEnrollConfig, running: true},
		statsRefreshWorker:   &StatsRefreshWorker{config: statsRefreshConfig, running: true},
		reevaluationWorker:   &ReevaluationWorker{config: reevalConfig, running: false},
		running:              true,
	}

	health := manager.HealthCheck()

	assert.True(t, health["auto_enrollment"])
	assert.True(t, health["stats_refresh"])
	assert.False(t, health["reevaluation"])
}

func TestManager_IsRunning(t *testing.T) {
	manager := &Manager{
		running: false,
	}

	assert.False(t, manager.IsRunning())

	manager.running = true
	assert.True(t, manager.IsRunning())
}

func TestStatsRefreshWorker_GetLastRefresh(t *testing.T) {
	now := time.Now()
	worker := &StatsRefreshWorker{
		lastRefresh: now,
	}

	lastRefresh := worker.GetLastRefresh()
	assert.Equal(t, now, lastRefresh)
}

func TestReevaluationWorker_GetStats(t *testing.T) {
	expectedStats := ReevaluationStats{
		TotalEvaluated:    100,
		PatientsProcessed: 50,
		RiskTierChanges:   10,
		Errors:            2,
	}

	worker := &ReevaluationWorker{
		stats: expectedStats,
	}

	stats := worker.GetStats()
	assert.Equal(t, expectedStats.TotalEvaluated, stats.TotalEvaluated)
	assert.Equal(t, expectedStats.PatientsProcessed, stats.PatientsProcessed)
	assert.Equal(t, expectedStats.RiskTierChanges, stats.RiskTierChanges)
	assert.Equal(t, expectedStats.Errors, stats.Errors)
}
