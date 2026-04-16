package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kb-26-metabolic-digital-twin/internal/models"
)

// CGMPeriodReportRepository persists CGMPeriodReport rows for the
// KB-26 CGM analytics consumer. Phase 7 P7-E Milestone 2.
type CGMPeriodReportRepository struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewCGMPeriodReportRepository constructs the repository.
func NewCGMPeriodReportRepository(db *gorm.DB, log *zap.Logger) *CGMPeriodReportRepository {
	if log == nil {
		log = zap.NewNop()
	}
	return &CGMPeriodReportRepository{db: db, log: log}
}

// SavePeriodReport writes a CGMPeriodReport row. Idempotent at the
// database level only: repeated writes insert duplicate rows. De-dup
// based on (patient_id, period_end) is a Phase 8 refinement once the
// production traffic pattern is understood.
func (r *CGMPeriodReportRepository) SavePeriodReport(report *models.CGMPeriodReport) error {
	if r.db == nil {
		return fmt.Errorf("CGMPeriodReportRepository: db not wired")
	}
	if report == nil {
		return fmt.Errorf("CGMPeriodReportRepository: nil report")
	}
	if report.CreatedAt.IsZero() {
		report.CreatedAt = time.Now().UTC()
	}
	// Phase 9 P9-D: upsert on (patient_id, period_end) to handle
	// at-least-once Kafka delivery. A second delivery of the same
	// CGM analytics event for the same patient + window-end updates
	// ALL columns on the existing row rather than creating a
	// duplicate. The UNIQUE(patient_id, period_end) index on
	// CGMPeriodReport enforces this at the database level.
	// UpdateAll=true avoids manually listing column names, which
	// would diverge from GORM's CamelCase→snake_case derivation.
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "patient_id"}, {Name: "period_end"}},
		UpdateAll: true,
	}).Create(report).Error
}

// FetchLatestPeriodReport returns the most recent CGMPeriodReport for
// a patient, or (nil, nil) when no report exists. Used by KB-20's
// summary-context extension and any downstream consumer that wants
// the freshest TIR / mean glucose / GRI zone for a patient.
func (r *CGMPeriodReportRepository) FetchLatestPeriodReport(patientID string) (*models.CGMPeriodReport, error) {
	if r.db == nil {
		return nil, fmt.Errorf("CGMPeriodReportRepository: db not wired")
	}
	var report models.CGMPeriodReport
	err := r.db.
		Where("patient_id = ?", patientID).
		Order("period_end DESC").
		First(&report).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &report, nil
}

// PersistingCGMAnalyticsHandler is the Milestone 2 replacement for
// LogOnlyCGMAnalyticsHandler. Converts each CGMAnalyticsEventPayload
// into a CGMPeriodReport and writes it via the repository. Conversion
// details:
//
//   - PeriodEnd = evt.WindowEndMs (the Flink window boundary)
//   - PeriodStart = PeriodEnd - evt.WindowDays days
//   - HypoEvents / SevereHypoEvents / HyperEvents / NocturnalHypos:
//     the Flink wire format carries *Detected booleans (one per
//     window detection); we map true → 1 else 0. This is truthful
//     at the 14-day granularity where each detector fires once per
//     window — finer-grained event counts are a Phase 8 refinement
//     that needs the upstream detector to track occurrences, not
//     just a sticky flag.
//
// Persistence failures are logged but the handler still returns nil
// so the consumer commits the Kafka offset and moves on — a failed
// write is recoverable via reprocessing on the next window slide
// (the same patient will produce a fresh event in 24 hours).
func PersistingCGMAnalyticsHandler(
	repo *CGMPeriodReportRepository,
	log *zap.Logger,
) CGMAnalyticsHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return func(ctx context.Context, evt CGMAnalyticsEventPayload) error {
		report := eventToPeriodReport(evt)
		if err := repo.SavePeriodReport(report); err != nil {
			log.Error("CGM period report persistence failed",
				zap.String("patient_id", evt.PatientID),
				zap.Error(err))
			// Return nil so the consumer still commits; reprocessing
			// on the next window slide heals transient DB failures.
			return nil
		}
		log.Info("CGM period report persisted",
			zap.String("patient_id", evt.PatientID),
			zap.Float64("tir_pct", evt.TIRPct),
			zap.String("gri_zone", evt.GRIZone),
			zap.Int("window_days", evt.WindowDays))
		return nil
	}
}

// eventToPeriodReport converts a wire-format event into a GORM model
// row. Extracted as a pure function so unit tests can pin the field
// mapping without a database.
func eventToPeriodReport(evt CGMAnalyticsEventPayload) *models.CGMPeriodReport {
	periodEnd := time.UnixMilli(evt.WindowEndMs).UTC()
	periodStart := periodEnd.AddDate(0, 0, -evt.WindowDays)

	return &models.CGMPeriodReport{
		PatientID:        evt.PatientID,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
		CoveragePct:      evt.CoveragePct,
		SufficientData:   evt.SufficientData,
		ConfidenceLevel:  evt.ConfidenceLvl,
		MeanGlucose:      evt.MeanGlucose,
		SDGlucose:        evt.SDGlucose,
		CVPct:            evt.CVPct,
		GlucoseStable:    evt.GlucoseStable,
		TIRPct:           evt.TIRPct,
		TBRL1Pct:         evt.TBRL1Pct,
		TBRL2Pct:         evt.TBRL2Pct,
		TARL1Pct:         evt.TARL1Pct,
		TARL2Pct:         evt.TARL2Pct,
		GMI:              evt.GMI,
		GRI:              evt.GRI,
		GRIZone:          evt.GRIZone,
		HypoEvents:       boolToInt(evt.SustainedHypoDetected),
		SevereHypoEvents: boolToInt(evt.SustainedSevereHypoDetected),
		HyperEvents:      boolToInt(evt.SustainedHyperDetected),
		NocturnalHypos:   boolToInt(evt.NocturnalHypoDetected),
		CreatedAt:        time.Now().UTC(),
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
