// Package services provides business logic for KB-17 Population Registry
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/cache"
	"kb-17-population-registry/internal/criteria"
	"kb-17-population-registry/internal/database"
	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/producer"
)

// Common errors
var (
	ErrPatientNotFound      = errors.New("patient not found")
	ErrRegistryNotFound     = errors.New("registry not found")
	ErrAlreadyEnrolled      = errors.New("patient already enrolled in registry")
	ErrNotEnrolled          = errors.New("patient not enrolled in registry")
	ErrInvalidEnrollment    = errors.New("invalid enrollment data")
	ErrEnrollmentNotAllowed = errors.New("patient does not meet enrollment criteria")
)

// EnrollmentService handles patient enrollment business logic
type EnrollmentService struct {
	repo           *database.Repository
	cache          *cache.RedisCache
	criteriaEngine *criteria.Engine
	producer       *producer.EventProducer
	logger         *logrus.Entry
}

// NewEnrollmentService creates a new enrollment service
func NewEnrollmentService(
	repo *database.Repository,
	cache *cache.RedisCache,
	engine *criteria.Engine,
	producer *producer.EventProducer,
	logger *logrus.Entry,
) *EnrollmentService {
	return &EnrollmentService{
		repo:           repo,
		cache:          cache,
		criteriaEngine: engine,
		producer:       producer,
		logger:         logger.WithField("service", "enrollment"),
	}
}

// EnrollPatient enrolls a patient in a registry
func (s *EnrollmentService) EnrollPatient(ctx context.Context, req *models.EnrollmentRequest) (*models.RegistryPatient, error) {
	s.logger.WithFields(logrus.Fields{
		"patient_id":    req.PatientID,
		"registry_code": req.RegistryCode,
	}).Info("Enrolling patient in registry")

	// Validate request
	if req.PatientID == "" || req.RegistryCode == "" {
		return nil, ErrInvalidEnrollment
	}

	// Check if registry exists
	registry, err := s.repo.GetRegistry(req.RegistryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}
	if registry == nil {
		return nil, ErrRegistryNotFound
	}

	// Check if already enrolled
	existing, err := s.repo.GetEnrollmentByPatientRegistry(req.PatientID, req.RegistryCode)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing enrollment: %w", err)
	}
	if existing != nil && existing.Status == models.EnrollmentStatusActive {
		return nil, ErrAlreadyEnrolled
	}

	// Determine enrollment source
	source := req.Source
	if source == "" {
		source = models.EnrollmentSourceManual
	}

	// Create enrollment
	now := time.Now().UTC()
	enrollment := &models.RegistryPatient{
		RegistryCode:     req.RegistryCode,
		PatientID:        req.PatientID,
		Status:           models.EnrollmentStatusActive,
		EnrollmentSource: source,
		RiskTier:         models.RiskTierModerate, // Default, will be calculated
		EnrolledAt:       now,
		LastEvaluatedAt:  &now,
		Notes:            req.Notes,
	}

	// If we have patient data, evaluate risk tier
	if req.PatientData != nil {
		result, err := s.criteriaEngine.Evaluate(req.PatientData, registry)
		if err == nil && result != nil {
			enrollment.RiskTier = result.SuggestedRiskTier
			enrollment.SetEligibilityData(result)
		}
	}

	// Save enrollment
	if err := s.repo.CreateEnrollment(enrollment); err != nil {
		s.logger.WithError(err).Error("Failed to create enrollment")
		return nil, err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateEnrollment(ctx, req.PatientID, req.RegistryCode)
		_ = s.cache.InvalidateStats(ctx, req.RegistryCode)
	}

	// Produce enrollment event
	if s.producer != nil {
		if err := s.producer.ProduceEnrollmentEvent(ctx, enrollment); err != nil {
			s.logger.WithError(err).Warn("Failed to produce enrollment event")
		}
	}

	s.logger.WithFields(logrus.Fields{
		"enrollment_id": enrollment.ID,
		"risk_tier":     enrollment.RiskTier,
	}).Info("Patient enrolled successfully")

	return enrollment, nil
}

// DisenrollPatient removes a patient from a registry
func (s *EnrollmentService) DisenrollPatient(ctx context.Context, patientID string, registryCode models.RegistryCode, reason string) error {
	s.logger.WithFields(logrus.Fields{
		"patient_id":    patientID,
		"registry_code": registryCode,
		"reason":        reason,
	}).Info("Disenrolling patient from registry")

	enrollment, err := s.repo.GetEnrollmentByPatientRegistry(patientID, registryCode)
	if err != nil {
		return fmt.Errorf("failed to get enrollment: %w", err)
	}
	if enrollment == nil {
		return ErrNotEnrolled
	}

	now := time.Now().UTC()
	enrollment.Status = models.EnrollmentStatusDisenrolled
	enrollment.DisenrolledAt = &now
	enrollment.DisenrollReason = reason

	if err := s.repo.UpdateEnrollment(enrollment); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateEnrollment(ctx, patientID, registryCode)
		_ = s.cache.InvalidateStats(ctx, registryCode)
	}

	// Produce disenrollment event
	if s.producer != nil {
		_ = s.producer.ProduceDisenrollmentEvent(ctx, enrollment, reason)
	}

	return nil
}

// UpdateRiskTier updates a patient's risk tier
func (s *EnrollmentService) UpdateRiskTier(ctx context.Context, patientID string, registryCode models.RegistryCode, newTier models.RiskTier, reason string) error {
	enrollment, err := s.repo.GetEnrollmentByPatientRegistry(patientID, registryCode)
	if err != nil {
		return fmt.Errorf("failed to get enrollment: %w", err)
	}
	if enrollment == nil {
		return ErrNotEnrolled
	}

	oldTier := enrollment.RiskTier
	now := time.Now().UTC()
	enrollment.RiskTier = newTier
	enrollment.LastEvaluatedAt = &now

	if err := s.repo.UpdateEnrollment(enrollment); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.InvalidateEnrollment(ctx, patientID, registryCode)
	}

	// Produce risk changed event
	if s.producer != nil && oldTier != newTier {
		_ = s.producer.ProduceRiskChangedEvent(ctx, enrollment, oldTier, newTier)
	}

	return nil
}

// GetEnrollment retrieves an enrollment by ID
func (s *EnrollmentService) GetEnrollment(ctx context.Context, id string) (*models.RegistryPatient, error) {
	parsedUUID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid enrollment ID: %w", err)
	}
	return s.repo.GetEnrollment(parsedUUID)
}

// GetPatientEnrollment retrieves enrollment for a patient in a specific registry
func (s *EnrollmentService) GetPatientEnrollment(ctx context.Context, patientID string, registryCode models.RegistryCode) (*models.RegistryPatient, error) {
	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetEnrollment(ctx, patientID, registryCode); err == nil && cached != nil {
			return cached, nil
		}
	}

	enrollment, err := s.repo.GetEnrollmentByPatientRegistry(patientID, registryCode)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.cache != nil && enrollment != nil {
		_ = s.cache.SetEnrollment(ctx, enrollment)
	}

	return enrollment, nil
}

// GetPatientRegistries retrieves all registries for a patient
func (s *EnrollmentService) GetPatientRegistries(ctx context.Context, patientID string) ([]models.RegistryPatient, error) {
	// Try cache first
	if s.cache != nil {
		if cached, err := s.cache.GetPatientRegistries(ctx, patientID); err == nil && cached != nil {
			return cached, nil
		}
	}

	enrollments, err := s.repo.GetPatientRegistries(patientID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if s.cache != nil && len(enrollments) > 0 {
		_ = s.cache.SetPatientRegistries(ctx, patientID, enrollments)
	}

	return enrollments, nil
}

// GetRegistryPatients retrieves all patients in a registry with filters
func (s *EnrollmentService) GetRegistryPatients(ctx context.Context, registryCode models.RegistryCode, limit, offset int) ([]models.RegistryPatient, int64, error) {
	return s.repo.GetRegistryPatients(registryCode, limit, offset)
}

// BulkEnroll enrolls multiple patients in registries
func (s *EnrollmentService) BulkEnroll(ctx context.Context, requests []models.EnrollmentRequest) (*BulkEnrollResult, error) {
	result := &BulkEnrollResult{
		Enrolled: make([]string, 0),
		Skipped:  make([]string, 0),
		Errors:   make([]string, 0),
	}

	for _, req := range requests {
		_, err := s.EnrollPatient(ctx, &req)
		if err != nil {
			if err == ErrAlreadyEnrolled {
				result.Skipped = append(result.Skipped, req.PatientID)
			} else {
				result.Errors = append(result.Errors, req.PatientID+": "+err.Error())
				result.Failed++
			}
		} else {
			result.Enrolled = append(result.Enrolled, req.PatientID)
			result.Success++
		}
	}

	return result, nil
}

// BulkEnrollResult represents the result of a bulk enrollment operation
type BulkEnrollResult struct {
	Success  int      `json:"success_count"`
	Failed   int      `json:"failed_count"`
	Enrolled []string `json:"enrolled_patient_ids"`
	Skipped  []string `json:"skipped_patient_ids,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

// ReEvaluatePatient re-evaluates a patient's eligibility and risk tier
func (s *EnrollmentService) ReEvaluatePatient(ctx context.Context, patientID string, patientData *models.PatientClinicalData) ([]models.CriteriaEvaluationResult, error) {
	results := make([]models.CriteriaEvaluationResult, 0)

	// Get all active registries
	registries, err := s.repo.ListRegistries(true)
	if err != nil {
		return nil, err
	}

	for _, registry := range registries {
		if !registry.Active {
			continue
		}

		result, err := s.criteriaEngine.Evaluate(patientData, &registry)
		if err != nil {
			s.logger.WithError(err).WithField("registry", registry.Code).Warn("Failed to evaluate criteria")
			continue
		}

		if result != nil {
			results = append(results, *result)

			// Update existing enrollment if patient is enrolled
			existing, _ := s.repo.GetEnrollmentByPatientRegistry(patientID, registry.Code)
			if existing != nil && existing.Status == models.EnrollmentStatusActive {
				// Update risk tier if changed
				if existing.RiskTier != result.SuggestedRiskTier {
					_ = s.UpdateRiskTier(ctx, patientID, registry.Code, result.SuggestedRiskTier, "Re-evaluation")
				}
			} else if result.Eligible && registry.AutoEnroll {
				// Auto-enroll if eligible
				_, _ = s.EnrollPatient(ctx, &models.EnrollmentRequest{
					PatientID:    patientID,
					RegistryCode: registry.Code,
					Source:       models.EnrollmentSourceAutomatic,
					PatientData:  patientData,
				})
			}
		}
	}

	return results, nil
}

// GetHighRiskPatients retrieves patients with HIGH or CRITICAL risk tier
func (s *EnrollmentService) GetHighRiskPatients(ctx context.Context, registryCode models.RegistryCode, limit, offset int) ([]models.RegistryPatient, int64, error) {
	if registryCode != "" {
		// Filter by registry code
		query := &models.EnrollmentQuery{
			RegistryCode: registryCode,
			RiskTier:     models.RiskTierHigh, // Will filter both HIGH and CRITICAL
			Limit:        limit,
			Offset:       offset,
		}
		enrollments, total, err := s.repo.ListEnrollments(query)
		if err != nil {
			return nil, 0, err
		}
		// Filter to include CRITICAL as well
		filtered := make([]models.RegistryPatient, 0)
		for _, e := range enrollments {
			if e.RiskTier == models.RiskTierHigh || e.RiskTier == models.RiskTierCritical {
				filtered = append(filtered, e)
			}
		}
		return filtered, total, nil
	}

	return s.repo.GetHighRiskPatients(limit, offset)
}

// GetPatientsWithCareGaps retrieves patients who have care gaps
func (s *EnrollmentService) GetPatientsWithCareGaps(ctx context.Context, registryCode models.RegistryCode, limit, offset int) ([]models.RegistryPatient, int64, error) {
	if registryCode != "" {
		hasCareGaps := true
		query := &models.EnrollmentQuery{
			RegistryCode: registryCode,
			HasCareGaps:  &hasCareGaps,
			Limit:        limit,
			Offset:       offset,
		}
		return s.repo.ListEnrollments(query)
	}

	return s.repo.GetPatientsWithCareGaps(limit, offset)
}

// ValidateEnrollmentRequest validates an enrollment request
func ValidateEnrollmentRequest(req *models.EnrollmentRequest) error {
	if req == nil {
		return fmt.Errorf("enrollment request is nil")
	}
	if req.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}
	if req.RegistryCode == "" {
		return fmt.Errorf("registry_code is required")
	}
	return nil
}
