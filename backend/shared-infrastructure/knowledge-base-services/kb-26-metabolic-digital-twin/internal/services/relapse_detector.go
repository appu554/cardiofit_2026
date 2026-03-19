package services

import (
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	mriRiseThreshold   = 15.0
	hba1cRiseThreshold = 0.5
)

// RelapseDetector monitors patient MRI and HbA1c nadirs, detecting sustained
// rises that indicate metabolic relapse requiring re-correction.
type RelapseDetector struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewRelapseDetector(db *gorm.DB, logger *zap.Logger) *RelapseDetector {
	return &RelapseDetector{db: db, logger: logger}
}

// UpdateNadir records a new MRI score (and optional HbA1c) for a patient,
// lowering the nadir if the new value is the best seen so far.
func (rd *RelapseDetector) UpdateNadir(patientID uuid.UUID, mriScore float64, hba1c *float64) error {
	var nadir models.MRINadir
	err := rd.db.Where("patient_id = ?", patientID).First(&nadir).Error

	now := time.Now()

	if err == gorm.ErrRecordNotFound {
		nadir = models.MRINadir{
			ID:         uuid.New(),
			PatientID:  patientID,
			NadirScore: mriScore,
			NadirDate:  now,
		}
		if hba1c != nil {
			nadir.HbA1cNadir = hba1c
			nadir.HbA1cNadirAt = &now
		}
		return rd.db.Create(&nadir).Error
	}
	if err != nil {
		return err
	}

	updated := false
	if mriScore < nadir.NadirScore {
		nadir.NadirScore = mriScore
		nadir.NadirDate = now
		updated = true
	}
	if hba1c != nil && (nadir.HbA1cNadir == nil || *hba1c < *nadir.HbA1cNadir) {
		nadir.HbA1cNadir = hba1c
		nadir.HbA1cNadirAt = &now
		updated = true
	}

	if updated {
		return rd.db.Save(&nadir).Error
	}
	return nil
}

// GetNadir returns the stored nadir record for a patient.
func (rd *RelapseDetector) GetNadir(patientID uuid.UUID) (*models.MRINadir, error) {
	var nadir models.MRINadir
	err := rd.db.Where("patient_id = ?", patientID).First(&nadir).Error
	if err != nil {
		return nil, err
	}
	return &nadir, nil
}

// CheckRelapse examines the two most recent quarterly summaries against the
// stored nadir. It fires a RelapseEvent when HbA1c rises >0.5 from nadir or
// MRI rises >15 from nadir over two consecutive quarters.
func (rd *RelapseDetector) CheckRelapse(patientID uuid.UUID) (*models.RelapseEvent, error) {
	nadir, err := rd.GetNadir(patientID)
	if err != nil {
		return nil, nil
	}

	var summaries []models.QuarterlySummary
	if err := rd.db.Where("patient_id = ?", patientID).
		Order("year DESC, quarter DESC").Limit(2).
		Find(&summaries).Error; err != nil {
		return nil, err
	}
	if len(summaries) < 2 {
		return nil, nil
	}

	now := time.Now()

	// Check HbA1c rise from nadir in most recent quarter
	if nadir.HbA1cNadir != nil && summaries[0].LatestHbA1c != nil {
		delta := *summaries[0].LatestHbA1c - *nadir.HbA1cNadir
		if delta > hba1cRiseThreshold {
			event := &models.RelapseEvent{
				ID:           uuid.New(),
				PatientID:    patientID,
				TriggerType:  "HBA1C_RISE",
				TriggerValue: delta,
				NadirValue:   *nadir.HbA1cNadir,
				CurrentValue: *summaries[0].LatestHbA1c,
				DetectedAt:   now,
			}
			if err := rd.db.Create(event).Error; err != nil {
				return nil, err
			}
			rd.logger.Warn("relapse detected: HbA1c rise from nadir",
				zap.String("patient_id", patientID.String()),
				zap.Float64("delta", delta))
			return event, nil
		}
	}

	// Check MRI rise sustained over 2 consecutive quarters
	q1Delta := summaries[0].MeanMRI - nadir.NadirScore
	q2Delta := summaries[1].MeanMRI - nadir.NadirScore
	if q1Delta > mriRiseThreshold && q2Delta > mriRiseThreshold {
		event := &models.RelapseEvent{
			ID:           uuid.New(),
			PatientID:    patientID,
			TriggerType:  "MRI_RISE",
			TriggerValue: q1Delta,
			NadirValue:   nadir.NadirScore,
			CurrentValue: summaries[0].MeanMRI,
			DetectedAt:   now,
		}
		if err := rd.db.Create(event).Error; err != nil {
			return nil, err
		}
		rd.logger.Warn("relapse detected: MRI rise sustained 2 consecutive quarters",
			zap.String("patient_id", patientID.String()),
			zap.Float64("q1_delta", q1Delta),
			zap.Float64("q2_delta", q2Delta))
		return event, nil
	}

	return nil, nil
}

// GetRelapseHistory returns all relapse events for a patient, most recent first.
func (rd *RelapseDetector) GetRelapseHistory(patientID uuid.UUID) ([]models.RelapseEvent, error) {
	var events []models.RelapseEvent
	err := rd.db.Where("patient_id = ?", patientID).
		Order("detected_at DESC").
		Find(&events).Error
	return events, err
}

// DaysSinceLastRelapse returns the number of days since the patient's most
// recent relapse event, or -1 if no relapse has been recorded.
func (rd *RelapseDetector) DaysSinceLastRelapse(patientID uuid.UUID) int {
	var event models.RelapseEvent
	err := rd.db.Where("patient_id = ?", patientID).
		Order("detected_at DESC").First(&event).Error
	if err != nil {
		return -1
	}
	return int(time.Since(event.DetectedAt).Hours() / 24)
}
