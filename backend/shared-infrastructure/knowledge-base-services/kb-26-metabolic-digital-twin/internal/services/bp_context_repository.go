package services

import (
	"errors"
	"time"

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
			"raw_phenotype",
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

// ListActivePatientIDs returns distinct patient IDs from twin_state whose
// most recent update is within the given activity window. The query
// dedups via SELECT DISTINCT — three snapshots for the same patient
// return one ID. IDs are returned as strings to match the BP context
// orchestrator's signature.
func (r *BPContextRepository) ListActivePatientIDs(window time.Duration) ([]string, error) {
	cutoff := time.Now().UTC().Add(-window)
	var ids []string
	err := r.db.Model(&models.TwinState{}).
		Distinct("patient_id").
		Where("updated_at > ?", cutoff).
		Pluck("patient_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// FetchHistorySince returns all snapshots for a patient with snapshot_date
// on or after the given time. Results are ordered oldest-first, matching
// the stability engine's expected History ordering (latest last).
func (r *BPContextRepository) FetchHistorySince(patientID string, since time.Time) ([]models.BPContextHistory, error) {
	var snapshots []models.BPContextHistory
	err := r.db.
		Where("patient_id = ? AND snapshot_date >= ?", patientID, since).
		Order("snapshot_date ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

// DB returns the underlying GORM handle. Intended for tests and admin
// utilities that need raw query access; production code should call
// repository methods.
func (r *BPContextRepository) DB() *gorm.DB { return r.db }
