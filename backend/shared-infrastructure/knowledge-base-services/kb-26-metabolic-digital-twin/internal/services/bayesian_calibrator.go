package services

import (
	"math"

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
	db     *gorm.DB
	logger *zap.Logger
}

// NewBayesianCalibrator creates a new BayesianCalibrator.
func NewBayesianCalibrator(db *gorm.DB, logger *zap.Logger) *BayesianCalibrator {
	return &BayesianCalibrator{db: db, logger: logger}
}

// Calibrate performs a single Bayesian update for a patient-intervention pair.
func (bc *BayesianCalibrator) Calibrate(
	patientID uuid.UUID,
	interventionCode, targetVariable string,
	populationEffect, observedEffect, observationSD float64,
) (*models.CalibratedEffect, error) {
	var existing models.CalibratedEffect
	result := bc.db.Where(
		"patient_id = ? AND intervention_code = ? AND target_variable = ?",
		patientID, interventionCode, targetVariable,
	).First(&existing)

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
