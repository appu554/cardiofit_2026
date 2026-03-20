package services

import (
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// QuarterlyAggregator computes per-quarter MRI summary statistics for a patient.
// These summaries feed the RelapseDetector's two-consecutive-quarter check.
type QuarterlyAggregator struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewQuarterlyAggregator creates a new QuarterlyAggregator.
func NewQuarterlyAggregator(db *gorm.DB, logger *zap.Logger) *QuarterlyAggregator {
	return &QuarterlyAggregator{db: db, logger: logger}
}

// ComputeQuarter aggregates all MRI scores for a patient in a given calendar
// quarter and upserts the result into the quarterly_summaries table.
func (qa *QuarterlyAggregator) ComputeQuarter(patientID uuid.UUID, year, quarter int) error {
	start, end := quarterBounds(year, quarter)

	var result struct {
		Mean  float64
		Min   float64
		Max   float64
		Count int
	}

	err := qa.db.Model(&models.MRIScore{}).
		Select("AVG(score) as mean, MIN(score) as min, MAX(score) as max, COUNT(*) as count").
		Where("patient_id = ? AND computed_at >= ? AND computed_at < ?", patientID, start, end).
		Scan(&result).Error
	if err != nil || result.Count == 0 {
		return err
	}

	summary := models.QuarterlySummary{
		ID:         uuid.New(),
		PatientID:  patientID,
		Year:       year,
		Quarter:    quarter,
		MeanMRI:    result.Mean,
		MinMRI:     result.Min,
		MaxMRI:     result.Max,
		MRICount:   result.Count,
		ComputedAt: time.Now(),
	}

	// Upsert: if a summary already exists for this patient/year/quarter, update it.
	return qa.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "patient_id"}, {Name: "year"}, {Name: "quarter"}},
		DoUpdates: clause.AssignmentColumns([]string{"mean_mri", "min_mri", "max_mri", "mri_count", "computed_at"}),
	}).Create(&summary).Error
}

// quarterBounds returns the [start, end) time range for a calendar quarter.
func quarterBounds(year, quarter int) (time.Time, time.Time) {
	startMonth := time.Month((quarter-1)*3 + 1)
	start := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 3, 0)
	return start, end
}
