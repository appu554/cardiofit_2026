package services

import (
	"gorm.io/gorm"
	"kb-26-metabolic-digital-twin/internal/models"
)

// PAIRepository handles persistence for PAI scores and history.
type PAIRepository struct {
	db *gorm.DB
}

// NewPAIRepository creates a new PAIRepository backed by the given DB.
func NewPAIRepository(db *gorm.DB) *PAIRepository {
	return &PAIRepository{db: db}
}

// SaveScore persists a PAI score and adds a history snapshot.
func (r *PAIRepository) SaveScore(score models.PAIScore) error {
	if err := r.db.Create(&score).Error; err != nil {
		return err
	}
	// Also write to history for trend analysis
	history := models.PAIHistory{
		PatientID:   score.PatientID,
		Score:       score.Score,
		Tier:        score.Tier,
		VelocityS:   score.VelocityScore,
		ProximityS:  score.ProximityScore,
		BehavioralS: score.BehavioralScore,
		ContextS:    score.ContextScore,
		AttentionS:  score.AttentionScore,
		TriggerEvt:  score.TriggerEvent,
		ComputedAt:  score.ComputedAt,
	}
	return r.db.Create(&history).Error
}

// FetchLatest returns the most recent PAI score for a patient.
func (r *PAIRepository) FetchLatest(patientID string) (*models.PAIScore, error) {
	var score models.PAIScore
	err := r.db.Where("patient_id = ?", patientID).
		Order("computed_at DESC").
		First(&score).Error
	if err != nil {
		return nil, err
	}
	return &score, nil
}

// FetchTrend returns the last N PAI history entries for a patient.
func (r *PAIRepository) FetchTrend(patientID string, limit int) ([]models.PAIHistory, error) {
	var entries []models.PAIHistory
	err := r.db.Where("patient_id = ?", patientID).
		Order("computed_at DESC").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}
