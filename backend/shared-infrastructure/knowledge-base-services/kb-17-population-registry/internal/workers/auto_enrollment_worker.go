// Package workers provides background workers for KB-17 Population Registry
package workers

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/services"
)

// AutoEnrollmentWorker handles automatic patient enrollment based on eligibility
type AutoEnrollmentWorker struct {
	enrollmentService *services.EnrollmentService
	evaluationService *services.EvaluationService
	logger            *logrus.Entry
	config            *AutoEnrollmentConfig
	stopCh            chan struct{}
	wg                sync.WaitGroup
	running           bool
	mu                sync.RWMutex
}

// AutoEnrollmentConfig holds auto-enrollment worker configuration
type AutoEnrollmentConfig struct {
	Enabled           bool
	CheckInterval     time.Duration
	BatchSize         int
	WorkerCount       int
	RetryDelay        time.Duration
	MaxRetries        int
}

// DefaultAutoEnrollmentConfig returns default configuration
func DefaultAutoEnrollmentConfig() *AutoEnrollmentConfig {
	return &AutoEnrollmentConfig{
		Enabled:       true,
		CheckInterval: 5 * time.Minute,
		BatchSize:     100,
		WorkerCount:   3,
		RetryDelay:    30 * time.Second,
		MaxRetries:    3,
	}
}

// NewAutoEnrollmentWorker creates a new auto-enrollment worker
func NewAutoEnrollmentWorker(
	enrollmentService *services.EnrollmentService,
	evaluationService *services.EvaluationService,
	config *AutoEnrollmentConfig,
	logger *logrus.Entry,
) *AutoEnrollmentWorker {
	return &AutoEnrollmentWorker{
		enrollmentService: enrollmentService,
		evaluationService: evaluationService,
		config:            config,
		logger:            logger.WithField("worker", "auto_enrollment"),
		stopCh:            make(chan struct{}),
	}
}

// Start starts the auto-enrollment worker
func (w *AutoEnrollmentWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	if !w.config.Enabled {
		w.logger.Info("Auto-enrollment worker is disabled")
		return nil
	}

	w.logger.Info("Starting auto-enrollment worker")

	w.wg.Add(1)
	go w.runLoop(ctx)

	return nil
}

// Stop stops the auto-enrollment worker
func (w *AutoEnrollmentWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.logger.Info("Stopping auto-enrollment worker")
	close(w.stopCh)
	w.wg.Wait()
	w.logger.Info("Auto-enrollment worker stopped")
}

func (w *AutoEnrollmentWorker) runLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.CheckInterval)
	defer ticker.Stop()

	// Run immediately on start
	w.processAutoEnrollments(ctx)

	for {
		select {
		case <-ticker.C:
			w.processAutoEnrollments(ctx)
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *AutoEnrollmentWorker) processAutoEnrollments(ctx context.Context) {
	w.logger.Debug("Processing auto-enrollment queue")

	startTime := time.Now()

	// Get pending auto-enrollment candidates
	// In a real implementation, this would query a queue or database
	// For now, we'll implement the structure for future integration

	w.logger.WithFields(logrus.Fields{
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Debug("Auto-enrollment processing completed")
}

// ProcessPatientEvent handles a patient clinical event for auto-enrollment
func (w *AutoEnrollmentWorker) ProcessPatientEvent(ctx context.Context, event *models.PatientClinicalEvent) error {
	w.logger.WithFields(logrus.Fields{
		"patient_id": event.PatientID,
		"event_type": event.EventType,
	}).Debug("Processing patient event for auto-enrollment")

	// Evaluate eligibility
	eligibility, err := w.evaluationService.EvaluatePatientEligibility(ctx, event.PatientID, event.ClinicalData)
	if err != nil {
		w.logger.WithError(err).Error("Failed to evaluate patient eligibility")
		return err
	}

	// Process eligible registries
	for _, regEligibility := range eligibility.RegistryEligibility {
		if !regEligibility.Eligible {
			continue
		}

		// Attempt auto-enrollment
		_, err := w.enrollmentService.EnrollPatient(ctx, &models.EnrollmentRequest{
			PatientID:    event.PatientID,
			RegistryCode: regEligibility.RegistryCode,
			Source:       models.EnrollmentSourceAutomatic,
			PatientData:  event.ClinicalData,
		})

		if err != nil {
			if err != services.ErrAlreadyEnrolled {
				w.logger.WithError(err).WithFields(logrus.Fields{
					"patient_id":    event.PatientID,
					"registry_code": regEligibility.RegistryCode,
				}).Warn("Failed to auto-enroll patient")
			}
			continue
		}

		w.logger.WithFields(logrus.Fields{
			"patient_id":    event.PatientID,
			"registry_code": regEligibility.RegistryCode,
		}).Info("Patient auto-enrolled in registry")
	}

	return nil
}

// IsRunning returns whether the worker is running
func (w *AutoEnrollmentWorker) IsRunning() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.running
}
