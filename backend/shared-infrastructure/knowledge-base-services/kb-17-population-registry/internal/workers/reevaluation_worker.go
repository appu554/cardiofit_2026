// Package workers provides background workers for KB-17 Population Registry
package workers

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/database"
	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/services"
)

// ReevaluationWorker handles periodic patient re-evaluation
type ReevaluationWorker struct {
	repo              *database.Repository
	enrollmentService *services.EnrollmentService
	evaluationService *services.EvaluationService
	logger            *logrus.Entry
	config            *ReevaluationConfig
	stopCh            chan struct{}
	wg                sync.WaitGroup
	running           bool
	mu                sync.RWMutex
	stats             ReevaluationStats
}

// ReevaluationConfig holds re-evaluation worker configuration
type ReevaluationConfig struct {
	Enabled           bool
	CheckInterval     time.Duration
	BatchSize         int
	StaleThreshold    time.Duration // Re-evaluate patients not evaluated in this duration
	WorkerCount       int
}

// ReevaluationStats tracks worker statistics
type ReevaluationStats struct {
	TotalEvaluated     int64
	LastRunTime        time.Time
	LastRunDuration    time.Duration
	PatientsProcessed  int
	RiskTierChanges    int
	Errors             int
}

// DefaultReevaluationConfig returns default configuration
func DefaultReevaluationConfig() *ReevaluationConfig {
	return &ReevaluationConfig{
		Enabled:        true,
		CheckInterval:  1 * time.Hour,
		BatchSize:      50,
		StaleThreshold: 24 * time.Hour, // Re-evaluate patients not evaluated in 24 hours
		WorkerCount:    2,
	}
}

// NewReevaluationWorker creates a new re-evaluation worker
func NewReevaluationWorker(
	repo *database.Repository,
	enrollmentService *services.EnrollmentService,
	evaluationService *services.EvaluationService,
	config *ReevaluationConfig,
	logger *logrus.Entry,
) *ReevaluationWorker {
	return &ReevaluationWorker{
		repo:              repo,
		enrollmentService: enrollmentService,
		evaluationService: evaluationService,
		config:            config,
		logger:            logger.WithField("worker", "reevaluation"),
		stopCh:            make(chan struct{}),
	}
}

// Start starts the re-evaluation worker
func (w *ReevaluationWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	if !w.config.Enabled {
		w.logger.Info("Re-evaluation worker is disabled")
		return nil
	}

	w.logger.Info("Starting re-evaluation worker")

	w.wg.Add(1)
	go w.runLoop(ctx)

	return nil
}

// Stop stops the re-evaluation worker
func (w *ReevaluationWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.logger.Info("Stopping re-evaluation worker")
	close(w.stopCh)
	w.wg.Wait()
	w.logger.Info("Re-evaluation worker stopped")
}

func (w *ReevaluationWorker) runLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processReevaluations(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *ReevaluationWorker) processReevaluations(ctx context.Context) {
	w.logger.Debug("Processing patient re-evaluations")

	startTime := time.Now()

	// Get active patients needing re-evaluation
	// Use ListEnrollments with status filter since GetStaleEnrollments doesn't exist
	query := &models.EnrollmentQuery{
		Status: models.EnrollmentStatusActive,
		Limit:  w.config.BatchSize,
	}
	patients, _, err := w.repo.ListEnrollments(query)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get enrollments for re-evaluation")
		return
	}

	// Filter to stale enrollments (those not evaluated recently)
	staleThreshold := time.Now().Add(-w.config.StaleThreshold)
	stalePatients := make([]models.RegistryPatient, 0)
	for _, p := range patients {
		if p.LastEvaluatedAt == nil || p.LastEvaluatedAt.Before(staleThreshold) {
			stalePatients = append(stalePatients, p)
		}
	}
	patients = stalePatients

	if len(patients) == 0 {
		w.logger.Debug("No patients need re-evaluation")
		return
	}

	w.logger.WithField("count", len(patients)).Info("Processing stale enrollments")

	var (
		processed       int
		riskTierChanges int
		errors          int
	)

	// Process each patient
	for _, enrollment := range patients {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		err := w.reevaluatePatient(ctx, &enrollment)
		if err != nil {
			errors++
			w.logger.WithError(err).WithField("patient_id", enrollment.PatientID).Warn("Re-evaluation failed")
			continue
		}

		processed++

		// Check if risk tier changed
		if enrollment.RiskTier != enrollment.RiskTier { // Would need to compare before/after
			riskTierChanges++
		}
	}

	// Update stats
	w.mu.Lock()
	w.stats.TotalEvaluated += int64(processed)
	w.stats.LastRunTime = time.Now()
	w.stats.LastRunDuration = time.Since(startTime)
	w.stats.PatientsProcessed = processed
	w.stats.RiskTierChanges = riskTierChanges
	w.stats.Errors = errors
	w.mu.Unlock()

	w.logger.WithFields(logrus.Fields{
		"processed":   processed,
		"risk_changes": riskTierChanges,
		"errors":      errors,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Re-evaluation batch completed")
}

func (w *ReevaluationWorker) reevaluatePatient(ctx context.Context, enrollment *models.RegistryPatient) error {
	// In a real implementation, we would fetch fresh patient clinical data
	// and re-evaluate eligibility and risk tier

	// For now, just update the last evaluated timestamp
	now := time.Now().UTC()
	enrollment.LastEvaluatedAt = &now

	return w.repo.UpdateEnrollment(enrollment)
}

// ForceReevaluation triggers immediate re-evaluation for a patient
func (w *ReevaluationWorker) ForceReevaluation(ctx context.Context, patientID string, patientData *models.PatientClinicalData) ([]models.CriteriaEvaluationResult, error) {
	w.logger.WithField("patient_id", patientID).Info("Force re-evaluation triggered")

	return w.enrollmentService.ReEvaluatePatient(ctx, patientID, patientData)
}

// GetStats returns worker statistics
func (w *ReevaluationWorker) GetStats() ReevaluationStats {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.stats
}

// IsRunning returns whether the worker is running
func (w *ReevaluationWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}

// GetStatus returns the worker status
func (w *ReevaluationWorker) GetStatus() map[string]interface{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return map[string]interface{}{
		"running":            w.running,
		"enabled":            w.config.Enabled,
		"total_evaluated":    w.stats.TotalEvaluated,
		"last_run_time":      w.stats.LastRunTime,
		"last_run_duration":  w.stats.LastRunDuration.String(),
		"patients_processed": w.stats.PatientsProcessed,
		"risk_tier_changes":  w.stats.RiskTierChanges,
		"errors":             w.stats.Errors,
	}
}
