package services

import (
	"fmt"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TwinUpdater manages CRUD operations on TwinState snapshots.
type TwinUpdater struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewTwinUpdater(db *gorm.DB, logger *zap.Logger) *TwinUpdater {
	return &TwinUpdater{db: db, logger: logger}
}

// GetLatest returns the most recent twin state for the given patient.
func (u *TwinUpdater) GetLatest(patientID uuid.UUID) (*models.TwinState, error) {
	var twin models.TwinState
	result := u.db.Where("patient_id = ?", patientID).Order("updated_at DESC").First(&twin)
	if result.Error != nil {
		return nil, fmt.Errorf("twin state not found: %w", result.Error)
	}
	return &twin, nil
}

// CreateSnapshot persists a new twin state snapshot with auto-incremented version.
func (u *TwinUpdater) CreateSnapshot(twin *models.TwinState) error {
	var maxVersion int
	u.db.Model(&models.TwinState{}).Where("patient_id = ?", twin.PatientID).
		Select("COALESCE(MAX(state_version), 0)").Scan(&maxVersion)
	twin.StateVersion = maxVersion + 1
	return u.db.Create(twin).Error
}

// GetHistory returns the N most recent twin states for a patient.
func (u *TwinUpdater) GetHistory(patientID uuid.UUID, limit int) ([]models.TwinState, error) {
	var states []models.TwinState
	result := u.db.Where("patient_id = ?", patientID).Order("updated_at DESC").Limit(limit).Find(&states)
	return states, result.Error
}
