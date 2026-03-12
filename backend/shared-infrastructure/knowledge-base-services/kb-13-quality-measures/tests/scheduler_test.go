package tests

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/config"
	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/scheduler"
)

func TestScheduler_StartStop(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	cfg := &config.SchedulerConfig{
		Enabled:            true,
		DailyEnabled:       false,
		WeeklyEnabled:      false,
		MonthlyEnabled:     false,
		QuarterlyEnabled:   false,
		DailyInterval:      24 * time.Hour,
		CalculationTimeout: 5 * time.Minute,
	}

	// Create scheduler without engine/repo (testing control flow only)
	s := scheduler.NewScheduler(nil, nil, store, cfg, logger)

	// Start scheduler
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	if !s.IsRunning() {
		t.Error("Expected scheduler to be running")
	}

	// Starting again should error
	err = s.Start()
	if err == nil {
		t.Error("Expected error when starting already running scheduler")
	}

	// Stop scheduler
	s.Stop()

	if s.IsRunning() {
		t.Error("Expected scheduler to be stopped")
	}
}

func TestScheduler_GetStatus(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	cfg := &config.SchedulerConfig{
		Enabled:          true,
		DailyEnabled:     true,
		WeeklyEnabled:    true,
		MonthlyEnabled:   false,
		QuarterlyEnabled: false,
	}

	s := scheduler.NewScheduler(nil, nil, store, cfg, logger)

	status := s.GetStatus()

	if status.Running {
		t.Error("Expected scheduler not to be running initially")
	}
	if !status.DailyEnabled {
		t.Error("Expected daily to be enabled")
	}
	if !status.WeeklyEnabled {
		t.Error("Expected weekly to be enabled")
	}
	if status.MonthlyEnabled {
		t.Error("Expected monthly to be disabled")
	}
	if status.QuarterlyEnabled {
		t.Error("Expected quarterly to be disabled")
	}
}

func TestScheduler_GetLastRun(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	cfg := &config.SchedulerConfig{
		Enabled: true,
	}

	s := scheduler.NewScheduler(nil, nil, store, cfg, logger)

	// No runs yet
	_, found := s.GetLastRun("daily")
	if found {
		t.Error("Expected no last run initially")
	}
}

func TestScheduler_RunNow_InvalidType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	cfg := &config.SchedulerConfig{
		Enabled: true,
	}

	s := scheduler.NewScheduler(nil, nil, store, cfg, logger)

	_, err := s.RunNow("invalid-type")
	if err == nil {
		t.Error("Expected error for invalid schedule type")
	}
}

func TestScheduler_GetJobHistory(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	store := models.NewMeasureStore()

	cfg := &config.SchedulerConfig{
		Enabled: true,
	}

	s := scheduler.NewScheduler(nil, nil, store, cfg, logger)

	// Initially empty
	history := s.GetJobHistory(10)
	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(history))
	}
}
