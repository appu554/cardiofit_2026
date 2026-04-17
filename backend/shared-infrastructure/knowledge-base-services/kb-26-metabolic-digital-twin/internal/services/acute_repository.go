package services

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

// AcuteRepository provides persistence for acute-on-chronic detection events
// and patient baseline snapshots.
type AcuteRepository struct {
	db *gorm.DB
}

// NewAcuteRepository constructs a repository bound to the given DB handle.
func NewAcuteRepository(db *gorm.DB) *AcuteRepository {
	return &AcuteRepository{db: db}
}

// SaveEvent persists an acute event.
func (r *AcuteRepository) SaveEvent(event *models.AcuteEvent) error {
	return r.db.Create(event).Error
}

// SaveBaseline upserts a patient baseline snapshot. If a row already exists
// for the same (patient_id, vital_sign_type), the existing row is updated;
// otherwise a new row is created.
func (r *AcuteRepository) SaveBaseline(baseline *models.PatientBaselineSnapshot) error {
	var existing models.PatientBaselineSnapshot
	return r.db.
		Where("patient_id = ? AND vital_sign_type = ?", baseline.PatientID, baseline.VitalSignType).
		Assign(baseline).
		FirstOrCreate(&existing).Error
}

// FetchBaseline returns the current baseline for a patient + vital sign,
// or nil if none exists.
func (r *AcuteRepository) FetchBaseline(patientID, vitalType string) (*models.PatientBaselineSnapshot, error) {
	var baseline models.PatientBaselineSnapshot
	err := r.db.
		Where("patient_id = ? AND vital_sign_type = ?", patientID, vitalType).
		First(&baseline).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &baseline, nil
}

// FetchActiveEvents returns unresolved acute events for a patient.
func (r *AcuteRepository) FetchActiveEvents(patientID string) ([]models.AcuteEvent, error) {
	var events []models.AcuteEvent
	err := r.db.
		Where("patient_id = ? AND resolved_at IS NULL", patientID).
		Order("detected_at DESC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

// FetchRecentDeviations returns acute events from the last 72 hours for
// compound pattern checking.
func (r *AcuteRepository) FetchRecentDeviations(patientID string, since time.Time) ([]models.AcuteEvent, error) {
	var events []models.AcuteEvent
	err := r.db.
		Where("patient_id = ? AND detected_at >= ?", patientID, since).
		Order("detected_at ASC").
		Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

// MarkResolved sets ResolvedAt and ResolutionType on an existing event.
func (r *AcuteRepository) MarkResolved(eventID uuid.UUID, resolutionType string) error {
	now := time.Now().UTC()
	return r.db.Model(&models.AcuteEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"resolved_at":     &now,
			"resolution_type": resolutionType,
		}).Error
}
