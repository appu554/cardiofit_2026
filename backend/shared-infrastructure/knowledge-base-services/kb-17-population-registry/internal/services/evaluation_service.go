// Package services provides business logic for KB-17 Population Registry
package services

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/cache"
	"kb-17-population-registry/internal/criteria"
	"kb-17-population-registry/internal/database"
	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/producer"
)

// EvaluationService handles patient eligibility evaluation
type EvaluationService struct {
	repo           *database.Repository
	cache          *cache.RedisCache
	criteriaEngine *criteria.Engine
	producer       *producer.EventProducer
	logger         *logrus.Entry
}

// NewEvaluationService creates a new evaluation service
func NewEvaluationService(
	repo *database.Repository,
	cache *cache.RedisCache,
	engine *criteria.Engine,
	producer *producer.EventProducer,
	logger *logrus.Entry,
) *EvaluationService {
	return &EvaluationService{
		repo:           repo,
		cache:          cache,
		criteriaEngine: engine,
		producer:       producer,
		logger:         logger.WithField("service", "evaluation"),
	}
}

// EvaluatePatientEligibility evaluates a patient's eligibility for all registries
func (s *EvaluationService) EvaluatePatientEligibility(ctx context.Context, patientID string, patientData *models.PatientClinicalData) (*models.EligibilityResult, error) {
	s.logger.WithField("patient_id", patientID).Info("Evaluating patient eligibility for all registries")

	startTime := time.Now()

	// Get all active registries
	registries, err := s.repo.ListRegistries(true)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get registries")
		return nil, err
	}

	result := &models.EligibilityResult{
		PatientID:           patientID,
		EvaluatedAt:         time.Now().UTC(),
		RegistryEligibility: make([]models.RegistryEligibility, 0),
	}

	// Evaluate each registry
	for _, registry := range registries {
		if !registry.Active {
			continue
		}

		eligibility := s.evaluateRegistryEligibility(ctx, patientData, &registry)
		result.RegistryEligibility = append(result.RegistryEligibility, eligibility)

		// Cache individual evaluation result if available
		if s.cache != nil && eligibility.EvaluationDetails != nil {
			_ = s.cache.SetEligibility(ctx, eligibility.EvaluationDetails)
		}
	}

	result.EvaluationDuration = time.Since(startTime)

	s.logger.WithFields(logrus.Fields{
		"patient_id":     patientID,
		"eligible_count": s.countEligible(result.RegistryEligibility),
		"duration_ms":    result.EvaluationDuration.Milliseconds(),
	}).Info("Patient eligibility evaluation completed")

	return result, nil
}

// evaluateRegistryEligibility evaluates eligibility for a specific registry
func (s *EvaluationService) evaluateRegistryEligibility(ctx context.Context, patientData *models.PatientClinicalData, registry *models.Registry) models.RegistryEligibility {
	eligibility := models.RegistryEligibility{
		RegistryCode: registry.Code,
		RegistryName: registry.Name,
		Eligible:     false,
	}

	// Use criteria engine for evaluation
	evalResult, err := s.criteriaEngine.Evaluate(patientData, registry)
	if err != nil {
		s.logger.WithError(err).WithField("registry", registry.Code).Warn("Criteria evaluation failed")
		eligibility.EvaluationError = err.Error()
		return eligibility
	}

	if evalResult != nil {
		eligibility.Eligible = evalResult.Eligible
		eligibility.SuggestedRiskTier = evalResult.SuggestedRiskTier
		eligibility.EvaluationDetails = evalResult

		// Convert MatchedCriterion slice to string slice
		eligibility.MatchedCriteria = matchedCriteriaToStrings(evalResult.MatchedCriteria)

		// Calculate confidence based on matched criteria count
		if len(evalResult.MatchedCriteria) > 0 {
			eligibility.ConfidenceScore = float64(len(evalResult.MatchedCriteria)) / 10.0
			if eligibility.ConfidenceScore > 1.0 {
				eligibility.ConfidenceScore = 1.0
			}
		}
	}

	return eligibility
}

// matchedCriteriaToStrings converts MatchedCriterion slice to string descriptions
func matchedCriteriaToStrings(criteria []models.MatchedCriterion) []string {
	result := make([]string, 0, len(criteria))
	for _, c := range criteria {
		if c.Description != "" {
			result = append(result, c.Description)
		} else {
			result = append(result, c.Field)
		}
	}
	return result
}

// riskFactorsToStrings converts RiskFactor slice to string descriptions
func riskFactorsToStrings(factors []models.RiskFactor) []string {
	result := make([]string, 0, len(factors))
	for _, f := range factors {
		if f.Description != "" {
			result = append(result, f.Description)
		} else {
			result = append(result, f.Name)
		}
	}
	return result
}

// countEligible counts the number of eligible registries
func (s *EvaluationService) countEligible(eligibilities []models.RegistryEligibility) int {
	count := 0
	for _, e := range eligibilities {
		if e.Eligible {
			count++
		}
	}
	return count
}

// EvaluateRiskTier calculates a patient's risk tier for a specific registry
func (s *EvaluationService) EvaluateRiskTier(ctx context.Context, patientID string, registryCode models.RegistryCode, patientData *models.PatientClinicalData) (*models.RiskAssessment, error) {
	s.logger.WithFields(logrus.Fields{
		"patient_id":    patientID,
		"registry_code": registryCode,
	}).Info("Evaluating risk tier")

	// Get registry definition
	registry, err := s.repo.GetRegistry(registryCode)
	if err != nil {
		return nil, ErrRegistryNotFound
	}
	if registry == nil {
		return nil, ErrRegistryNotFound
	}

	// Calculate risk score
	assessment := &models.RiskAssessment{
		PatientID:    patientID,
		RegistryCode: registryCode,
		AssessedAt:   time.Now().UTC(),
	}

	// Use criteria engine for risk evaluation
	evalResult, err := s.criteriaEngine.Evaluate(patientData, registry)
	if err != nil {
		s.logger.WithError(err).Warn("Risk evaluation failed")
		assessment.RiskTier = models.RiskTierModerate // Default
		return assessment, nil
	}

	if evalResult != nil {
		assessment.RiskTier = evalResult.SuggestedRiskTier
		assessment.RiskFactors = riskFactorsToStrings(evalResult.RiskFactors)

		// Calculate risk score based on risk tier and factors
		assessment.RiskScore = calculateRiskScore(evalResult.SuggestedRiskTier, len(evalResult.RiskFactors))
		assessment.ConfidenceScore = 0.8 // Default confidence
	}

	return assessment, nil
}

// calculateRiskScore computes a numeric risk score from tier and factors
func calculateRiskScore(tier models.RiskTier, factorCount int) float64 {
	baseScore := 0.0
	switch tier {
	case models.RiskTierLow:
		baseScore = 0.2
	case models.RiskTierModerate:
		baseScore = 0.5
	case models.RiskTierHigh:
		baseScore = 0.75
	case models.RiskTierCritical:
		baseScore = 0.95
	}
	// Adjust based on factor count
	adjustment := float64(factorCount) * 0.02
	score := baseScore + adjustment
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// BatchEvaluatePatients evaluates multiple patients in parallel
func (s *EvaluationService) BatchEvaluatePatients(ctx context.Context, requests []models.BatchEvaluationRequest) (*models.BatchEvaluationResult, error) {
	s.logger.WithField("count", len(requests)).Info("Starting batch patient evaluation")

	result := &models.BatchEvaluationResult{
		Results: make([]models.EligibilityResult, 0, len(requests)),
		Errors:  make([]models.BatchEvaluationError, 0),
	}

	// Use worker pool for parallel evaluation
	var wg sync.WaitGroup
	resultChan := make(chan models.EligibilityResult, len(requests))
	errorChan := make(chan models.BatchEvaluationError, len(requests))

	// Limit concurrent evaluations
	semaphore := make(chan struct{}, 10)

	for _, req := range requests {
		wg.Add(1)
		go func(r models.BatchEvaluationRequest) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			evalResult, err := s.EvaluatePatientEligibility(ctx, r.PatientID, r.PatientData)
			if err != nil {
				errorChan <- models.BatchEvaluationError{
					PatientID: r.PatientID,
					Error:     err.Error(),
				}
				return
			}

			resultChan <- *evalResult
		}(req)
	}

	// Close channels after all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Collect results
	for evalResult := range resultChan {
		result.Results = append(result.Results, evalResult)
	}

	for evalError := range errorChan {
		result.Errors = append(result.Errors, evalError)
	}

	result.TotalEvaluated = len(requests)
	result.SuccessCount = len(result.Results)
	result.FailedCount = len(result.Errors)

	s.logger.WithFields(logrus.Fields{
		"total":   result.TotalEvaluated,
		"success": result.SuccessCount,
		"failed":  result.FailedCount,
	}).Info("Batch evaluation completed")

	return result, nil
}

// GetCachedEligibility retrieves cached eligibility for a patient and registry
func (s *EvaluationService) GetCachedEligibility(ctx context.Context, patientID string, registryCode models.RegistryCode) (*models.CriteriaEvaluationResult, error) {
	if s.cache == nil {
		return nil, nil
	}
	return s.cache.GetEligibility(ctx, patientID, registryCode)
}

// TriggerAutoEnrollment checks eligibility and auto-enrolls patients where applicable
func (s *EvaluationService) TriggerAutoEnrollment(ctx context.Context, patientID string, patientData *models.PatientClinicalData) ([]models.RegistryPatient, error) {
	s.logger.WithField("patient_id", patientID).Info("Checking for auto-enrollment opportunities")

	enrollments := make([]models.RegistryPatient, 0)

	// Evaluate eligibility
	eligibility, err := s.EvaluatePatientEligibility(ctx, patientID, patientData)
	if err != nil {
		return nil, err
	}

	// Get all registries with auto-enroll enabled
	registries, err := s.repo.ListRegistries(true)
	if err != nil {
		return nil, err
	}

	registryMap := make(map[models.RegistryCode]models.Registry)
	for _, r := range registries {
		registryMap[r.Code] = r
	}

	// Auto-enroll in eligible registries
	for _, regEligibility := range eligibility.RegistryEligibility {
		if !regEligibility.Eligible {
			continue
		}

		registry, exists := registryMap[regEligibility.RegistryCode]
		if !exists || !registry.AutoEnroll {
			continue
		}

		// Check if already enrolled
		existing, err := s.repo.GetEnrollmentByPatientRegistry(patientID, regEligibility.RegistryCode)
		if err == nil && existing != nil && existing.Status == models.EnrollmentStatusActive {
			continue // Already enrolled
		}

		// Create enrollment (would typically call enrollment service)
		// For now, just log the opportunity
		s.logger.WithFields(logrus.Fields{
			"patient_id":    patientID,
			"registry_code": regEligibility.RegistryCode,
		}).Info("Auto-enrollment opportunity detected")
	}

	return enrollments, nil
}

// ValidatePatientData validates that patient data meets minimum requirements
func (s *EvaluationService) ValidatePatientData(patientData *models.PatientClinicalData) *models.ValidationResult {
	result := &models.ValidationResult{
		Valid:    true,
		Warnings: make([]string, 0),
		Errors:   make([]string, 0),
	}

	if patientData == nil {
		result.Valid = false
		result.Errors = append(result.Errors, "patient data is nil")
		return result
	}

	// Check for minimum required fields
	if patientData.PatientID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "patient_id is required")
	}

	// Check for diagnoses
	if len(patientData.Diagnoses) == 0 {
		result.Warnings = append(result.Warnings, "no diagnoses provided - eligibility may be limited")
	}

	// Check for lab results
	if len(patientData.LabResults) == 0 {
		result.Warnings = append(result.Warnings, "no lab results provided - risk stratification may be less accurate")
	}

	// Check for medications
	if len(patientData.Medications) == 0 {
		result.Warnings = append(result.Warnings, "no medications provided - some evaluations may be incomplete")
	}

	return result
}
