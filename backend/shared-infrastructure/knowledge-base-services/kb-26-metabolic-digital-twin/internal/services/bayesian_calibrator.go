package services

import (
	"errors"
	"fmt"
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PosteriorResult holds the result of a Bayesian update.
type PosteriorResult struct {
	Mean float64
	SD   float64
}

// NormalPosterior computes the posterior of a Normal-Normal conjugate update.
func NormalPosterior(priorMean, priorSD, obs, obsSD float64) PosteriorResult {
	if priorSD <= 0 || obsSD <= 0 {
		return PosteriorResult{Mean: obs, SD: 1.0}
	}
	priorPrec := 1.0 / (priorSD * priorSD)
	obsPrec := 1.0 / (obsSD * obsSD)

	postPrec := priorPrec + obsPrec
	postVar := 1.0 / postPrec
	postMean := postVar * (priorMean*priorPrec + obs*obsPrec)
	postSD := math.Sqrt(postVar)

	return PosteriorResult{Mean: postMean, SD: postSD}
}

// CalibrationConfidence returns confidence (0-1) based on observation count.
func CalibrationConfidence(observations int) float64 {
	return 1.0 - math.Exp(-0.3*float64(observations))
}

// BayesianCalibrator manages per-patient effect calibration.
type BayesianCalibrator struct {
	db                  *gorm.DB
	logger              *zap.Logger
	burnInWeeks         int
	observationWindowDays int
}

// NewBayesianCalibrator creates a new BayesianCalibrator.
func NewBayesianCalibrator(db *gorm.DB, logger *zap.Logger) *BayesianCalibrator {
	return &BayesianCalibrator{db: db, logger: logger, burnInWeeks: 12, observationWindowDays: 14}
}

// NewBayesianCalibratorWithConfig creates a calibrator with custom burn-in and observation window.
func NewBayesianCalibratorWithConfig(db *gorm.DB, logger *zap.Logger, burnInWeeks, observationWindowDays int) *BayesianCalibrator {
	return &BayesianCalibrator{
		db:                    db,
		logger:                logger,
		burnInWeeks:           burnInWeeks,
		observationWindowDays: observationWindowDays,
	}
}

// CheckBurnIn verifies the patient has completed the burn-in period.
// Returns nil if burn-in is satisfied, error otherwise.
func (bc *BayesianCalibrator) CheckBurnIn(patientID uuid.UUID) error {
	var firstState models.TwinState
	result := bc.db.Where("patient_id = ?", patientID).
		Order("updated_at ASC").First(&firstState)
	if result.Error != nil {
		return fmt.Errorf("no twin state found: %w", result.Error)
	}

	burnInEnd := firstState.UpdatedAt.Add(time.Duration(bc.burnInWeeks) * 7 * 24 * time.Hour)
	if time.Now().UTC().Before(burnInEnd) {
		remaining := time.Until(burnInEnd).Hours() / 24
		return fmt.Errorf("burn-in period active: %.0f days remaining (started %s)",
			remaining, firstState.UpdatedAt.Format("2006-01-02"))
	}
	return nil
}

// CheckObservationWindow verifies that no more than one intervention change
// occurred within the observation window (default 14 days) for valid calibration.
func (bc *BayesianCalibrator) CheckObservationWindow(patientID uuid.UUID) error {
	windowStart := time.Now().UTC().Add(-time.Duration(bc.observationWindowDays) * 24 * time.Hour)

	var medChangeCount int64
	bc.db.Model(&models.TwinState{}).
		Where("patient_id = ? AND update_source = ? AND updated_at >= ?",
			patientID, "VMCU_MED_CHANGE", windowStart).
		Count(&medChangeCount)

	if medChangeCount > 1 {
		return fmt.Errorf("observation window violated: %d medication changes in last %d days (max 1 allowed)",
			medChangeCount, bc.observationWindowDays)
	}
	return nil
}

// Calibrate performs a single Bayesian update for a patient-intervention pair.
// Enforces burn-in period and observation window validation before applying.
func (bc *BayesianCalibrator) Calibrate(
	patientID uuid.UUID,
	interventionCode, targetVariable string,
	populationEffect, observedEffect, observationSD float64,
) (*models.CalibratedEffect, error) {
	// Enforce burn-in period
	if err := bc.CheckBurnIn(patientID); err != nil {
		bc.logger.Info("calibration blocked by burn-in", zap.Error(err))
		return nil, fmt.Errorf("calibration rejected: %w", err)
	}

	// Enforce observation window
	if err := bc.CheckObservationWindow(patientID); err != nil {
		bc.logger.Info("calibration blocked by observation window", zap.Error(err))
		return nil, fmt.Errorf("calibration rejected: %w", err)
	}

	var existing models.CalibratedEffect
	result := bc.db.Where(
		"patient_id = ? AND intervention_code = ? AND target_variable = ?",
		patientID, interventionCode, targetVariable,
	).First(&existing)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	if result.Error != nil {
		// No existing record — create new with population prior.
		priorSD := observationSD
		if priorSD <= 0 {
			priorSD = 2.0
		}

		post := NormalPosterior(populationEffect, priorSD, observedEffect, observationSD)
		conf := CalibrationConfidence(1)

		newEffect := models.CalibratedEffect{
			PatientID:        patientID,
			InterventionCode: interventionCode,
			TargetVariable:   targetVariable,
			PopulationEffect: populationEffect,
			PatientEffect:    post.Mean,
			Observations:     1,
			Confidence:       conf,
			PriorMean:        &populationEffect,
			PriorSD:          &priorSD,
			PosteriorMean:    &post.Mean,
			PosteriorSD:      &post.SD,
		}
		if err := bc.db.Create(&newEffect).Error; err != nil {
			return nil, err
		}
		return &newEffect, nil
	}

	// Existing record — update with new observation.
	priorMean := existing.PatientEffect
	priorSD := 1.0
	if existing.PosteriorSD != nil {
		priorSD = *existing.PosteriorSD
	}

	post := NormalPosterior(priorMean, priorSD, observedEffect, observationSD)
	existing.PatientEffect = post.Mean
	existing.Observations++
	existing.Confidence = CalibrationConfidence(existing.Observations)
	existing.PosteriorMean = &post.Mean
	existing.PosteriorSD = &post.SD

	if err := bc.db.Save(&existing).Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

// GetPatientConfidence returns the mean calibration confidence across all
// calibrated effects for a patient. Returns 0.65 (default) if no effects exist.
func (bc *BayesianCalibrator) GetPatientConfidence(patientID uuid.UUID) float64 {
	if bc.db == nil {
		return 0.65
	}

	var effects []models.CalibratedEffect
	bc.db.Where("patient_id = ?", patientID).Find(&effects)
	if len(effects) == 0 {
		return 0.65 // spec default
	}

	var sum float64
	for _, e := range effects {
		sum += e.Confidence
	}
	return sum / float64(len(effects))
}
