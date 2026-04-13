package services

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kb-26-metabolic-digital-twin/internal/models"
)

// BPContextRepository persists BP context classification snapshots for
// progression tracking (e.g. WCH -> SH conversion over months).
type BPContextRepository struct {
	db *gorm.DB
}

// NewBPContextRepository constructs a repository bound to the given DB handle.
func NewBPContextRepository(db *gorm.DB) *BPContextRepository {
	return &BPContextRepository{db: db}
}

// SaveSnapshot inserts a new snapshot, or upserts if one already exists for
// the same (patient_id, snapshot_date) — reclassification on the same day
// replaces the prior row rather than creating a duplicate.
func (r *BPContextRepository) SaveSnapshot(snapshot *models.BPContextHistory) error {
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "patient_id"},
			{Name: "snapshot_date"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"phenotype",
			"clinic_sbp_mean",
			"home_sbp_mean",
			"gap_sbp",
			"confidence",
		}),
	}).Create(snapshot).Error
}

// FetchLatest returns the most recent snapshot for a patient, or nil if none.
func (r *BPContextRepository) FetchLatest(patientID string) (*models.BPContextHistory, error) {
	var snapshot models.BPContextHistory
	err := r.db.
		Where("patient_id = ?", patientID).
		Order("snapshot_date DESC").
		First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// FetchHistory returns up to `limit` snapshots for a patient, newest first.
func (r *BPContextRepository) FetchHistory(patientID string, limit int) ([]models.BPContextHistory, error) {
	var snapshots []models.BPContextHistory
	err := r.db.
		Where("patient_id = ?", patientID).
		Order("snapshot_date DESC").
		Limit(limit).
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}
