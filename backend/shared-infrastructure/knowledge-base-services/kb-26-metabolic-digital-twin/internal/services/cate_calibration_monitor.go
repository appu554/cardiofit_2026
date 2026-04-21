package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

// CalibrationConfig parameterises the calibration monitor. Populated from
// cate_parameters.yaml (Task 2) via the parameters loader in Task 7.
type CalibrationConfig struct {
	AbsDiffAlarm      float64 // mean |attributed − predicted| above which the alarm fires
	RollingWindowDays int     // how far back the rolling window reaches
	MinMatchedPairs   int     // below this, summary is INSUFFICIENT_SIGNAL (no alarm either way)
}

// CalibrationStatus is the three-valued output of ComputeCalibrationSummary.
type CalibrationStatus string

const (
	CalibrationOK                 CalibrationStatus = "OK"
	CalibrationAlarm              CalibrationStatus = "ALARM"
	CalibrationInsufficientSignal CalibrationStatus = "INSUFFICIENT_SIGNAL"
)

// CalibrationSummary is the per-(cohort, intervention, horizon) output of the monitor.
type CalibrationSummary struct {
	CohortID       string            `json:"cohort_id"`
	InterventionID string            `json:"intervention_id"`
	HorizonDays    int               `json:"horizon_days"`
	MatchedPairs   int               `json:"matched_pairs"`
	MeanAbsDiff    float64           `json:"mean_abs_diff"`
	WindowStart    time.Time         `json:"window_start"`
	WindowEnd      time.Time         `json:"window_end"`
	Status         CalibrationStatus `json:"status"`
	AlarmTriggered bool              `json:"alarm_triggered"`
}

// CATECalibrationMonitor joins prior-CATE estimates (Task 2+5) with post-hoc
// Gap 21 attribution verdicts to compute a rolling mean |attributed − predicted|
// per (cohort, intervention, horizon). When the mean exceeds AbsDiffAlarm, it
// fires a CATE_MISCALIBRATION entry onto the governance ledger.
//
// Sprint 2 replaces the simple mean-abs-diff metric with calibration-by-decile
// (equivalent to Brier skill); the public Compute/Evaluate surface stays stable.
type CATECalibrationMonitor struct {
	db  *gorm.DB
	cfg CalibrationConfig
}

func NewCATECalibrationMonitor(db *gorm.DB, cfg CalibrationConfig) *CATECalibrationMonitor {
	return &CATECalibrationMonitor{db: db, cfg: cfg}
}

// ComputeCalibrationSummary joins cate_estimates with attribution_verdicts by
// ConsolidatedRecordID over the rolling window, filters to (cohort, intervention,
// horizon), restricts CATE rows to OverlapPass (inconclusive estimates are not
// calibration signal), and returns the summary.
func (m *CATECalibrationMonitor) ComputeCalibrationSummary(cohortID, interventionID string, horizonDays int) (CalibrationSummary, error) {
	windowEnd := time.Now().UTC()
	windowStart := windowEnd.AddDate(0, 0, -m.cfg.RollingWindowDays)

	type joined struct {
		PredCATE   float64
		Attributed float64
	}
	var rows []joined
	err := m.db.Raw(`
		SELECT c.point_estimate AS pred_cate, a.risk_difference AS attributed
		FROM cate_estimates c
		INNER JOIN attribution_verdicts a ON c.consolidated_record_id = a.consolidated_record_id
		WHERE c.cohort_id = ? AND c.intervention_id = ? AND c.horizon_days = ?
		  AND c.overlap_status = ?
		  AND a.computed_at BETWEEN ? AND ?
	`, cohortID, interventionID, horizonDays, string(models.OverlapPass), windowStart, windowEnd).Scan(&rows).Error
	if err != nil {
		return CalibrationSummary{}, fmt.Errorf("join cate_estimates and attribution_verdicts for cohort %s intervention %s: %w", cohortID, interventionID, err)
	}

	sum := CalibrationSummary{
		CohortID:       cohortID,
		InterventionID: interventionID,
		HorizonDays:    horizonDays,
		MatchedPairs:   len(rows),
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
	}
	if len(rows) < m.cfg.MinMatchedPairs {
		sum.Status = CalibrationInsufficientSignal
		return sum, nil
	}
	var total float64
	for _, r := range rows {
		total += math.Abs(r.Attributed - r.PredCATE)
	}
	sum.MeanAbsDiff = total / float64(len(rows))
	if sum.MeanAbsDiff > m.cfg.AbsDiffAlarm {
		sum.Status = CalibrationAlarm
		sum.AlarmTriggered = true
	} else {
		sum.Status = CalibrationOK
	}
	return sum, nil
}

// EvaluateAndAlarm runs ComputeCalibrationSummary for every (intervention, horizon)
// triple with data in the cohort and appends a CATE_MISCALIBRATION ledger entry
// for each alarm. Intended to be called on a schedule (cron / Kafka trigger).
// Sprint 1 exposes only this programmatic entry point; Task 7 wires an HTTP
// trigger; Sprint 2 wires up the scheduler.
func (m *CATECalibrationMonitor) EvaluateAndAlarm(cohortID string, ledger *InMemoryLedger) error {
	if ledger == nil {
		return errors.New("ledger required")
	}
	type triple struct {
		InterventionID string
		HorizonDays    int
	}
	var triples []triple
	if err := m.db.Raw(`
		SELECT DISTINCT intervention_id, horizon_days FROM cate_estimates WHERE cohort_id = ?
	`, cohortID).Scan(&triples).Error; err != nil {
		return fmt.Errorf("enumerate (intervention, horizon) triples for cohort %s: %w", cohortID, err)
	}
	for _, t := range triples {
		sum, err := m.ComputeCalibrationSummary(cohortID, t.InterventionID, t.HorizonDays)
		if err != nil {
			return err
		}
		if !sum.AlarmTriggered {
			continue
		}
		payload, marshalErr := json.Marshal(sum)
		if marshalErr != nil {
			return fmt.Errorf("marshal calibration summary: %w", marshalErr)
		}
		subject := cohortID + ":" + t.InterventionID
		if _, err := ledger.AppendEntry("CATE_MISCALIBRATION", subject, string(payload)); err != nil {
			return fmt.Errorf("append CATE_MISCALIBRATION ledger entry: %w", err)
		}
	}
	return nil
}
