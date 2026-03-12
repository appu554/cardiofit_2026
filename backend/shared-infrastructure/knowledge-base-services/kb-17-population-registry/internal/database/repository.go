// Package database provides database connection and repository implementations
package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-17-population-registry/internal/models"
)

// Repository provides database operations for registry and enrollment data
type Repository struct {
	db     *gorm.DB
	logger *logrus.Entry
}

// NewRepository creates a new repository
func NewRepository(conn *Connection, logger *logrus.Entry) *Repository {
	return &Repository{
		db:     conn.DB,
		logger: logger.WithField("component", "repository"),
	}
}

// ========== Registry Operations ==========

// GetRegistry retrieves a registry by code
func (r *Repository) GetRegistry(code models.RegistryCode) (*models.Registry, error) {
	var registry models.Registry
	err := r.db.Where("code = ?", code).First(&registry).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get registry: %w", err)
	}
	return &registry, nil
}

// ListRegistries lists all registries
func (r *Repository) ListRegistries(activeOnly bool) ([]models.Registry, error) {
	var registries []models.Registry
	query := r.db.Model(&models.Registry{})
	if activeOnly {
		query = query.Where("active = ?", true)
	}
	err := query.Order("name ASC").Find(&registries).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list registries: %w", err)
	}
	return registries, nil
}

// CreateRegistry creates a new registry
func (r *Repository) CreateRegistry(registry *models.Registry) error {
	if err := r.db.Create(registry).Error; err != nil {
		return fmt.Errorf("failed to create registry: %w", err)
	}
	return nil
}

// UpdateRegistry updates an existing registry
func (r *Repository) UpdateRegistry(registry *models.Registry) error {
	if err := r.db.Save(registry).Error; err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}
	return nil
}

// ========== Enrollment Operations ==========

// GetEnrollment retrieves an enrollment by ID
func (r *Repository) GetEnrollment(id uuid.UUID) (*models.RegistryPatient, error) {
	var enrollment models.RegistryPatient
	err := r.db.Where("id = ?", id).First(&enrollment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}
	return &enrollment, nil
}

// GetEnrollmentByPatientRegistry retrieves enrollment by patient ID and registry code
func (r *Repository) GetEnrollmentByPatientRegistry(patientID string, registryCode models.RegistryCode) (*models.RegistryPatient, error) {
	var enrollment models.RegistryPatient
	err := r.db.Where("patient_id = ? AND registry_code = ?", patientID, registryCode).First(&enrollment).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get enrollment: %w", err)
	}
	return &enrollment, nil
}

// ListEnrollments lists enrollments based on query parameters
func (r *Repository) ListEnrollments(query *models.EnrollmentQuery) ([]models.RegistryPatient, int64, error) {
	var enrollments []models.RegistryPatient
	var total int64

	db := r.db.Model(&models.RegistryPatient{})

	// Apply filters
	if query.RegistryCode != "" {
		db = db.Where("registry_code = ?", query.RegistryCode)
	}
	if query.PatientID != "" {
		db = db.Where("patient_id = ?", query.PatientID)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.RiskTier != "" {
		db = db.Where("risk_tier = ?", query.RiskTier)
	}
	if query.HasCareGaps != nil && *query.HasCareGaps {
		db = db.Where("jsonb_array_length(care_gaps) > 0")
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count enrollments: %w", err)
	}

	// Apply sorting
	sortBy := query.SortBy
	if sortBy == "" {
		sortBy = "enrolled_at"
	}
	sortOrder := query.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

	// Apply pagination
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	if err := db.Find(&enrollments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list enrollments: %w", err)
	}

	return enrollments, total, nil
}

// CreateEnrollment creates a new enrollment
func (r *Repository) CreateEnrollment(enrollment *models.RegistryPatient) error {
	if err := r.db.Create(enrollment).Error; err != nil {
		return fmt.Errorf("failed to create enrollment: %w", err)
	}

	// Record history
	history := &models.EnrollmentHistory{
		EnrollmentID: enrollment.ID,
		Action:       models.HistoryActionEnrolled,
		NewStatus:    enrollment.Status,
		NewRiskTier:  enrollment.RiskTier,
		ActorID:      enrollment.EnrolledBy,
	}
	r.recordHistory(history)

	return nil
}

// UpdateEnrollment updates an existing enrollment
func (r *Repository) UpdateEnrollment(enrollment *models.RegistryPatient) error {
	if err := r.db.Save(enrollment).Error; err != nil {
		return fmt.Errorf("failed to update enrollment: %w", err)
	}
	return nil
}

// UpdateEnrollmentStatus updates enrollment status with history
func (r *Repository) UpdateEnrollmentStatus(id uuid.UUID, oldStatus, newStatus models.EnrollmentStatus, reason, actorID string) error {
	now := time.Now().UTC()

	updates := map[string]interface{}{
		"status":     newStatus,
		"updated_at": now,
	}

	if newStatus == models.EnrollmentStatusDisenrolled {
		updates["disenrolled_at"] = now
		updates["disenroll_reason"] = reason
		updates["disenrolled_by"] = actorID
	}

	if err := r.db.Model(&models.RegistryPatient{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Record history
	action := models.HistoryActionEnrolled
	switch newStatus {
	case models.EnrollmentStatusDisenrolled:
		action = models.HistoryActionDisenrolled
	case models.EnrollmentStatusSuspended:
		action = models.HistoryActionSuspended
	case models.EnrollmentStatusActive:
		if oldStatus == models.EnrollmentStatusSuspended {
			action = models.HistoryActionReactivated
		}
	}

	history := &models.EnrollmentHistory{
		EnrollmentID: id,
		Action:       action,
		OldStatus:    oldStatus,
		NewStatus:    newStatus,
		Reason:       reason,
		ActorID:      actorID,
	}
	r.recordHistory(history)

	return nil
}

// UpdateEnrollmentRiskTier updates the risk tier with history
func (r *Repository) UpdateEnrollmentRiskTier(id uuid.UUID, oldTier, newTier models.RiskTier, actorID string) error {
	if err := r.db.Model(&models.RegistryPatient{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"risk_tier":  newTier,
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update risk tier: %w", err)
	}

	// Record history
	history := &models.EnrollmentHistory{
		EnrollmentID: id,
		Action:       models.HistoryActionRiskChanged,
		OldRiskTier:  oldTier,
		NewRiskTier:  newTier,
		ActorID:      actorID,
	}
	r.recordHistory(history)

	return nil
}

// UpdateEnrollmentMetrics updates enrollment metrics
func (r *Repository) UpdateEnrollmentMetrics(id uuid.UUID, metrics models.MetricMapSlice) error {
	if err := r.db.Model(&models.RegistryPatient{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"metrics":           metrics,
			"last_evaluated_at": time.Now().UTC(),
			"updated_at":        time.Now().UTC(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update metrics: %w", err)
	}
	return nil
}

// UpdateEnrollmentCareGaps updates care gaps
func (r *Repository) UpdateEnrollmentCareGaps(id uuid.UUID, careGaps []string) error {
	if err := r.db.Model(&models.RegistryPatient{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"care_gaps":  careGaps,
			"updated_at": time.Now().UTC(),
		}).Error; err != nil {
		return fmt.Errorf("failed to update care gaps: %w", err)
	}
	return nil
}

// DeleteEnrollment deletes an enrollment (soft delete via disenroll)
func (r *Repository) DeleteEnrollment(id uuid.UUID, reason, actorID string) error {
	enrollment, err := r.GetEnrollment(id)
	if err != nil {
		return err
	}
	if enrollment == nil {
		return fmt.Errorf("enrollment not found")
	}

	return r.UpdateEnrollmentStatus(id, enrollment.Status, models.EnrollmentStatusDisenrolled, reason, actorID)
}

// ========== Patient-Centric Operations ==========

// GetPatientRegistries gets all registries a patient is enrolled in
func (r *Repository) GetPatientRegistries(patientID string) ([]models.RegistryPatient, error) {
	var enrollments []models.RegistryPatient
	err := r.db.Where("patient_id = ? AND status != ?", patientID, models.EnrollmentStatusDisenrolled).
		Order("enrolled_at DESC").
		Find(&enrollments).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get patient registries: %w", err)
	}
	return enrollments, nil
}

// GetRegistryPatients gets all patients in a registry
func (r *Repository) GetRegistryPatients(registryCode models.RegistryCode, limit, offset int) ([]models.RegistryPatient, int64, error) {
	var enrollments []models.RegistryPatient
	var total int64

	query := r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ?", registryCode, models.EnrollmentStatusActive)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count patients: %w", err)
	}

	err := query.Order("risk_tier DESC, enrolled_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&enrollments).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get registry patients: %w", err)
	}

	return enrollments, total, nil
}

// ========== Analytics Operations ==========

// GetRegistryStats gets statistics for a registry
func (r *Repository) GetRegistryStats(registryCode models.RegistryCode) (*models.RegistryStats, error) {
	stats := &models.RegistryStats{
		RegistryCode: registryCode,
		LastUpdated:  time.Now().UTC(),
	}

	// Total enrolled (ever)
	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ?", registryCode).
		Count(&stats.TotalEnrolled)

	// Active count
	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ?", registryCode, models.EnrollmentStatusActive).
		Count(&stats.ActiveCount)

	// Pending count
	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ?", registryCode, models.EnrollmentStatusPending).
		Count(&stats.PendingCount)

	// Risk tier counts
	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ? AND risk_tier = ?", registryCode, models.EnrollmentStatusActive, models.RiskTierLow).
		Count(&stats.LowRiskCount)

	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ? AND risk_tier = ?", registryCode, models.EnrollmentStatusActive, models.RiskTierModerate).
		Count(&stats.ModerateCount)

	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ? AND risk_tier = ?", registryCode, models.EnrollmentStatusActive, models.RiskTierHigh).
		Count(&stats.HighRiskCount)

	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ? AND risk_tier = ?", registryCode, models.EnrollmentStatusActive, models.RiskTierCritical).
		Count(&stats.CriticalCount)

	// Care gap count
	r.db.Model(&models.RegistryPatient{}).
		Where("registry_code = ? AND status = ? AND jsonb_array_length(care_gaps) > 0", registryCode, models.EnrollmentStatusActive).
		Count(&stats.CareGapCount)

	return stats, nil
}

// GetHighRiskPatients gets high-risk patients across all registries
func (r *Repository) GetHighRiskPatients(limit, offset int) ([]models.RegistryPatient, int64, error) {
	var enrollments []models.RegistryPatient
	var total int64

	query := r.db.Model(&models.RegistryPatient{}).
		Where("status = ? AND risk_tier IN ?", models.EnrollmentStatusActive, []models.RiskTier{models.RiskTierHigh, models.RiskTierCritical})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count high-risk patients: %w", err)
	}

	err := query.Order("CASE risk_tier WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2 ELSE 3 END, enrolled_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&enrollments).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get high-risk patients: %w", err)
	}

	return enrollments, total, nil
}

// GetPatientsWithCareGaps gets patients with care gaps
func (r *Repository) GetPatientsWithCareGaps(limit, offset int) ([]models.RegistryPatient, int64, error) {
	var enrollments []models.RegistryPatient
	var total int64

	query := r.db.Model(&models.RegistryPatient{}).
		Where("status = ? AND jsonb_array_length(care_gaps) > 0", models.EnrollmentStatusActive)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count patients with care gaps: %w", err)
	}

	err := query.Order("jsonb_array_length(care_gaps) DESC, risk_tier DESC").
		Limit(limit).
		Offset(offset).
		Find(&enrollments).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get patients with care gaps: %w", err)
	}

	return enrollments, total, nil
}

// ========== History Operations ==========

// GetEnrollmentHistory gets history for an enrollment
func (r *Repository) GetEnrollmentHistory(enrollmentID uuid.UUID, limit int) ([]models.EnrollmentHistory, error) {
	var history []models.EnrollmentHistory
	err := r.db.Where("enrollment_id = ?", enrollmentID).
		Order("created_at DESC").
		Limit(limit).
		Find(&history).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get enrollment history: %w", err)
	}
	return history, nil
}

// recordHistory records an enrollment history entry
func (r *Repository) recordHistory(history *models.EnrollmentHistory) {
	if err := r.db.Create(history).Error; err != nil {
		r.logger.WithError(err).Error("Failed to record enrollment history")
	}
}

// ========== Bulk Operations ==========

// BulkEnroll enrolls multiple patients in a registry
func (r *Repository) BulkEnroll(enrollments []models.RegistryPatient) (*models.BulkEnrollmentResult, error) {
	result := &models.BulkEnrollmentResult{
		Enrolled: make([]string, 0),
		Skipped:  make([]string, 0),
		Errors:   make([]string, 0),
	}

	for _, enrollment := range enrollments {
		// Check if already enrolled
		existing, err := r.GetEnrollmentByPatientRegistry(enrollment.PatientID, enrollment.RegistryCode)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", enrollment.PatientID, err))
			continue
		}

		if existing != nil && existing.Status.IsActive() {
			result.Skipped = append(result.Skipped, enrollment.PatientID)
			continue
		}

		// Create enrollment
		if err := r.CreateEnrollment(&enrollment); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", enrollment.PatientID, err))
			continue
		}

		result.Success++
		result.Enrolled = append(result.Enrolled, enrollment.PatientID)
	}

	return result, nil
}
