// Package scheduler provides automated quality measure calculation scheduling.
//
// 🔴 CRITICAL ARCHITECTURE:
//   - All calculations use BATCH CQL evaluation
//   - Scheduler respects measurement period boundaries
//   - Results persisted with ExecutionContextVersion for audit
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/calculator"
	"kb-13-quality-measures/internal/config"
	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/repository"
)

// Scheduler manages automated quality measure calculations.
type Scheduler struct {
	engine     *calculator.Engine
	resultRepo *repository.ResultRepository
	store      *models.MeasureStore
	config     *config.SchedulerConfig
	logger     *zap.Logger

	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	running    bool
	runningMu  sync.RWMutex

	// Job tracking
	lastRun    map[string]time.Time
	lastRunMu  sync.RWMutex
	jobHistory []*JobRun
	historyMu  sync.RWMutex
}

// JobRun represents a single scheduler execution.
type JobRun struct {
	ID            string    `json:"id"`
	ScheduleType  string    `json:"schedule_type"` // daily, weekly, monthly, quarterly
	StartedAt     time.Time `json:"started_at"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
	MeasuresRun   int       `json:"measures_run"`
	Successful    int       `json:"successful"`
	Failed        int       `json:"failed"`
	Status        string    `json:"status"` // running, completed, failed
	Error         string    `json:"error,omitempty"`
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(
	engine *calculator.Engine,
	resultRepo *repository.ResultRepository,
	store *models.MeasureStore,
	cfg *config.SchedulerConfig,
	logger *zap.Logger,
) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		engine:     engine,
		resultRepo: resultRepo,
		store:      store,
		config:     cfg,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		lastRun:    make(map[string]time.Time),
		jobHistory: make([]*JobRun, 0, 100),
	}
}

// Start begins the scheduler loops.
func (s *Scheduler) Start() error {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.runningMu.Unlock()

	s.logger.Info("Starting quality measure scheduler",
		zap.Duration("daily_interval", s.config.DailyInterval),
		zap.Duration("weekly_interval", s.config.WeeklyInterval),
		zap.Duration("monthly_interval", s.config.MonthlyInterval),
	)

	// Start schedule loops
	if s.config.DailyEnabled {
		s.wg.Add(1)
		go s.dailyLoop()
	}

	if s.config.WeeklyEnabled {
		s.wg.Add(1)
		go s.weeklyLoop()
	}

	if s.config.MonthlyEnabled {
		s.wg.Add(1)
		go s.monthlyLoop()
	}

	if s.config.QuarterlyEnabled {
		s.wg.Add(1)
		go s.quarterlyLoop()
	}

	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.runningMu.Lock()
	if !s.running {
		s.runningMu.Unlock()
		return
	}
	s.running = false
	s.runningMu.Unlock()

	s.logger.Info("Stopping quality measure scheduler")
	s.cancel()
	s.wg.Wait()
	s.logger.Info("Scheduler stopped")
}

// IsRunning returns whether the scheduler is active.
func (s *Scheduler) IsRunning() bool {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()
	return s.running
}

// GetJobHistory returns recent job runs.
func (s *Scheduler) GetJobHistory(limit int) []*JobRun {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	if limit <= 0 || limit > len(s.jobHistory) {
		limit = len(s.jobHistory)
	}

	// Return most recent jobs
	start := len(s.jobHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*JobRun, limit)
	copy(result, s.jobHistory[start:])
	return result
}

// GetLastRun returns the last run time for a schedule type.
func (s *Scheduler) GetLastRun(scheduleType string) (time.Time, bool) {
	s.lastRunMu.RLock()
	defer s.lastRunMu.RUnlock()
	t, ok := s.lastRun[scheduleType]
	return t, ok
}

// RunNow triggers an immediate calculation run.
func (s *Scheduler) RunNow(scheduleType string) (*JobRun, error) {
	switch scheduleType {
	case "daily":
		return s.runDailyCalculations(), nil
	case "weekly":
		return s.runWeeklyCalculations(), nil
	case "monthly":
		return s.runMonthlyCalculations(), nil
	case "quarterly":
		return s.runQuarterlyCalculations(), nil
	default:
		return nil, fmt.Errorf("unknown schedule type: %s", scheduleType)
	}
}

// --- Schedule Loops ---

func (s *Scheduler) dailyLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.DailyInterval)
	defer ticker.Stop()

	// Run immediately on start if configured
	if s.config.RunOnStart {
		s.runDailyCalculations()
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runDailyCalculations()
		}
	}
}

func (s *Scheduler) weeklyLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.WeeklyInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Only run on configured day of week
			if time.Now().Weekday() == s.config.WeeklyRunDay {
				s.runWeeklyCalculations()
			}
		}
	}
}

func (s *Scheduler) monthlyLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.MonthlyInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Only run on configured day of month
			if time.Now().Day() == s.config.MonthlyRunDay {
				s.runMonthlyCalculations()
			}
		}
	}
}

func (s *Scheduler) quarterlyLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Run on first day of quarter months (Jan, Apr, Jul, Oct)
			now := time.Now()
			if now.Day() == 1 && isQuarterStart(now.Month()) {
				s.runQuarterlyCalculations()
			}
		}
	}
}

func isQuarterStart(m time.Month) bool {
	return m == time.January || m == time.April || m == time.July || m == time.October
}

// --- Calculation Runners ---

func (s *Scheduler) runDailyCalculations() *JobRun {
	return s.runCalculations("daily", s.getDailyMeasures())
}

func (s *Scheduler) runWeeklyCalculations() *JobRun {
	return s.runCalculations("weekly", s.getWeeklyMeasures())
}

func (s *Scheduler) runMonthlyCalculations() *JobRun {
	return s.runCalculations("monthly", s.getMonthlyMeasures())
}

func (s *Scheduler) runQuarterlyCalculations() *JobRun {
	return s.runCalculations("quarterly", s.getQuarterlyMeasures())
}

func (s *Scheduler) runCalculations(scheduleType string, measures []*models.Measure) *JobRun {
	job := &JobRun{
		ID:           fmt.Sprintf("%s-%d", scheduleType, time.Now().UnixNano()),
		ScheduleType: scheduleType,
		StartedAt:    time.Now(),
		MeasuresRun:  len(measures),
		Status:       "running",
	}

	s.addJobToHistory(job)

	s.logger.Info("Starting scheduled calculations",
		zap.String("schedule_type", scheduleType),
		zap.Int("measure_count", len(measures)),
	)

	// Extract measure IDs
	measureIDs := make([]string, len(measures))
	for i, m := range measures {
		measureIDs[i] = m.ID
	}

	// Run batch calculation
	ctx, cancel := context.WithTimeout(s.ctx, s.config.CalculationTimeout)
	defer cancel()

	results, err := s.engine.CalculateBatch(ctx, measureIDs, time.Now().Year())

	job.CompletedAt = time.Now()

	if err != nil {
		job.Status = "completed_with_errors"
		job.Error = err.Error()
		s.logger.Warn("Scheduled calculations completed with errors",
			zap.String("schedule_type", scheduleType),
			zap.Error(err),
		)
	} else {
		job.Status = "completed"
	}

	// Count successes and failures
	for _, result := range results {
		if result != nil {
			job.Successful++
			// Persist result if repository available
			if s.resultRepo != nil {
				if err := s.resultRepo.Save(ctx, result); err != nil {
					s.logger.Error("Failed to save calculation result",
						zap.String("measure_id", result.MeasureID),
						zap.Error(err),
					)
				}
			}
		} else {
			job.Failed++
		}
	}

	// Update last run time
	s.lastRunMu.Lock()
	s.lastRun[scheduleType] = job.CompletedAt
	s.lastRunMu.Unlock()

	s.logger.Info("Scheduled calculations completed",
		zap.String("schedule_type", scheduleType),
		zap.Int("successful", job.Successful),
		zap.Int("failed", job.Failed),
		zap.Duration("duration", job.CompletedAt.Sub(job.StartedAt)),
	)

	return job
}

func (s *Scheduler) addJobToHistory(job *JobRun) {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	s.jobHistory = append(s.jobHistory, job)

	// Trim history to max 100 entries
	if len(s.jobHistory) > 100 {
		s.jobHistory = s.jobHistory[len(s.jobHistory)-100:]
	}
}

// --- Measure Selectors ---

func (s *Scheduler) getDailyMeasures() []*models.Measure {
	// Daily: High-priority measures that need frequent monitoring
	all := s.store.GetActiveMeasures()
	return filterBySchedule(all, "daily")
}

func (s *Scheduler) getWeeklyMeasures() []*models.Measure {
	// Weekly: Standard quality measures
	all := s.store.GetActiveMeasures()
	return filterBySchedule(all, "weekly")
}

func (s *Scheduler) getMonthlyMeasures() []*models.Measure {
	// Monthly: All active measures for regular reporting
	all := s.store.GetActiveMeasures()
	return filterBySchedule(all, "monthly")
}

// filterBySchedule filters measures by their calculation schedule.
func filterBySchedule(measures []*models.Measure, schedule string) []*models.Measure {
	var result []*models.Measure
	for _, m := range measures {
		if containsSchedule(m.CalculationSchedule, schedule) {
			result = append(result, m)
		}
	}
	return result
}

func (s *Scheduler) getQuarterlyMeasures() []*models.Measure {
	// Quarterly: All measures for quarterly reporting cycles
	return s.store.GetActiveMeasures()
}

func containsSchedule(schedules []string, target string) bool {
	for _, s := range schedules {
		if s == target {
			return true
		}
	}
	return false
}

// --- Status ---

// Status returns current scheduler status.
type Status struct {
	Running         bool                 `json:"running"`
	DailyEnabled    bool                 `json:"daily_enabled"`
	WeeklyEnabled   bool                 `json:"weekly_enabled"`
	MonthlyEnabled  bool                 `json:"monthly_enabled"`
	QuarterlyEnabled bool               `json:"quarterly_enabled"`
	LastRuns        map[string]time.Time `json:"last_runs"`
	RecentJobs      []*JobRun            `json:"recent_jobs"`
}

// GetStatus returns the current scheduler status.
func (s *Scheduler) GetStatus() *Status {
	s.lastRunMu.RLock()
	lastRuns := make(map[string]time.Time)
	for k, v := range s.lastRun {
		lastRuns[k] = v
	}
	s.lastRunMu.RUnlock()

	return &Status{
		Running:          s.IsRunning(),
		DailyEnabled:     s.config.DailyEnabled,
		WeeklyEnabled:    s.config.WeeklyEnabled,
		MonthlyEnabled:   s.config.MonthlyEnabled,
		QuarterlyEnabled: s.config.QuarterlyEnabled,
		LastRuns:         lastRuns,
		RecentJobs:       s.GetJobHistory(10),
	}
}
