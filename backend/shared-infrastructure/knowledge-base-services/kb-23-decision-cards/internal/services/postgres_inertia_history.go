package services

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kb-23-decision-cards/internal/models"
)

// InertiaVerdictRow is the GORM model for the inertia_verdict_history
// table. One row per (patient_id, week_start_date) — upserted on
// each weekly batch run. Phase 9 P9-C replaces the Phase 7 P7-D
// in-memory store with this persistent table so dampening survives
// service restart.
type InertiaVerdictRow struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID         string    `gorm:"size:100;uniqueIndex:idx_inertia_patient_week;not null" json:"patient_id"`
	WeekStartDate     time.Time `gorm:"uniqueIndex:idx_inertia_patient_week;not null" json:"week_start_date"`
	VerdictsJSON      string    `gorm:"type:text;not null" json:"verdicts_json"`
	DualDomainDetected bool     `gorm:"default:false" json:"dual_domain_detected"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the Postgres table name.
func (InertiaVerdictRow) TableName() string { return "inertia_verdict_history" }

// postgresInertiaHistory implements the InertiaVerdictHistory interface
// backed by the inertia_verdict_history Postgres table. Phase 9 P9-C.
type postgresInertiaHistory struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewPostgresInertiaHistory constructs the persistent store.
func NewPostgresInertiaHistory(db *gorm.DB, logger *zap.Logger) InertiaVerdictHistory {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &postgresInertiaHistory{db: db, logger: logger}
}

// SaveVerdict implements InertiaVerdictHistory. Upserts on
// (patient_id, week_start_date) so repeated calls in the same
// week update the existing row rather than creating duplicates.
func (h *postgresInertiaHistory) SaveVerdict(patientID string, weekStart time.Time, report models.PatientInertiaReport) error {
	if h.db == nil {
		return nil
	}

	verdictsBytes, err := json.Marshal(report)
	if err != nil {
		return err
	}

	hasDualDomain := false
	detectedCount := 0
	for _, v := range report.Verdicts {
		if v.Detected {
			detectedCount++
		}
	}
	if detectedCount >= 2 {
		hasDualDomain = true
	}

	row := InertiaVerdictRow{
		ID:                 uuid.New(),
		PatientID:          patientID,
		WeekStartDate:      weekStart,
		VerdictsJSON:       string(verdictsBytes),
		DualDomainDetected: hasDualDomain,
	}

	// Upsert: ON CONFLICT (patient_id, week_start_date) DO UPDATE
	result := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "patient_id"}, {Name: "week_start_date"}},
		DoUpdates: clause.AssignmentColumns([]string{"verdicts_json", "dual_domain_detected", "updated_at"}),
	}).Create(&row)

	if result.Error != nil {
		h.logger.Warn("failed to upsert inertia verdict history",
			zap.String("patient_id", patientID),
			zap.Error(result.Error))
		return result.Error
	}
	return nil
}

// FetchLatest implements InertiaVerdictHistory. Returns the most
// recent verdict for the patient (by week_start_date DESC).
// Returns (zero, zero, false) when no history exists.
func (h *postgresInertiaHistory) FetchLatest(patientID string) (models.PatientInertiaReport, time.Time, bool) {
	if h.db == nil {
		return models.PatientInertiaReport{}, time.Time{}, false
	}

	var row InertiaVerdictRow
	err := h.db.
		Where("patient_id = ?", patientID).
		Order("week_start_date DESC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return models.PatientInertiaReport{}, time.Time{}, false
		}
		h.logger.Warn("failed to fetch inertia verdict history",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return models.PatientInertiaReport{}, time.Time{}, false
	}

	var report models.PatientInertiaReport
	if err := json.Unmarshal([]byte(row.VerdictsJSON), &report); err != nil {
		h.logger.Warn("failed to unmarshal inertia verdict history",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return models.PatientInertiaReport{}, time.Time{}, false
	}

	return report, row.WeekStartDate, true
}
