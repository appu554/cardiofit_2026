package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PlausibilityAction is the recommended action for a plausibility check result.
type PlausibilityAction string

const (
	PlausibilityAccept      PlausibilityAction = "ACCEPT"
	PlausibilityFlagReview  PlausibilityAction = "FLAG_REVIEW"
	PlausibilityRejectRetest PlausibilityAction = "REJECT_RETEST"
)

// PlausibilityResult describes the outcome of a cross-session plausibility check.
type PlausibilityResult struct {
	Action     PlausibilityAction `json:"action"`
	Confidence float64            `json:"confidence"` // 0.0 = very suspicious, 1.0 = very plausible
	Reason     string             `json:"reason,omitempty"`
	RuleID     string             `json:"rule_id,omitempty"`
}

// PlausibilityConfig holds configurable thresholds for plausibility rules.
type PlausibilityConfig struct {
	// Rate-of-change limits (max physiologically possible change per 24h)
	EGFRMaxDeltaPerDay       float64 // default: 15 mL/min/1.73m²
	CreatinineMaxPctPer24h   float64 // default: 50% (unless dialysis)
	PotassiumMaxDeltaPerDay  float64 // default: 2.0 mEq/L
	GlucoseMaxDeltaPerHour   float64 // default: 10.0 mmol/L
	HbA1cMaxDeltaPer30d      float64 // default: 2.0%

	// Direction reversal sensitivity
	DirectionReversalMinPoints int     // default: 3 consecutive same-direction values
	DirectionReversalStdDevs   float64 // default: 2.0 standard deviations
}

// DefaultPlausibilityConfig returns production-safe defaults.
func DefaultPlausibilityConfig() PlausibilityConfig {
	return PlausibilityConfig{
		EGFRMaxDeltaPerDay:         15.0,
		CreatinineMaxPctPer24h:     50.0,
		PotassiumMaxDeltaPerDay:    2.0,
		GlucoseMaxDeltaPerHour:     10.0,
		HbA1cMaxDeltaPer30d:        2.0,
		DirectionReversalMinPoints: 3,
		DirectionReversalStdDevs:   2.0,
	}
}

// LabHistoryRecord is a minimal lab value record for plausibility checking.
type LabHistoryRecord struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// PlausibilityEngine validates new lab values against a patient's historical
// trajectory to detect physiologically implausible results.
type PlausibilityEngine struct {
	db  *gorm.DB
	log *zap.Logger
	cfg PlausibilityConfig
}

// NewPlausibilityEngine creates a new plausibility engine.
func NewPlausibilityEngine(db *gorm.DB, log *zap.Logger, cfg PlausibilityConfig) *PlausibilityEngine {
	return &PlausibilityEngine{db: db, log: log, cfg: cfg}
}

// CheckPlausibility validates a new lab value against the patient's history.
func (e *PlausibilityEngine) CheckPlausibility(
	ctx context.Context,
	patientID uuid.UUID,
	labType string,
	newValue float64,
	timestamp time.Time,
) (*PlausibilityResult, error) {
	// Fetch recent history for this lab type (last 10 values)
	history, err := e.fetchHistory(ctx, patientID, labType, 10)
	if err != nil {
		return nil, fmt.Errorf("fetch history: %w", err)
	}

	// If insufficient history, accept by default
	if len(history) < 2 {
		return &PlausibilityResult{
			Action:     PlausibilityAccept,
			Confidence: 1.0,
			Reason:     "insufficient history for plausibility check",
		}, nil
	}

	// Rule 1: Rate-of-change check against most recent value
	if result := e.checkRateOfChange(labType, newValue, timestamp, history[0]); result != nil {
		return result, nil
	}

	// Rule 2: Direction reversal check
	if len(history) >= e.cfg.DirectionReversalMinPoints {
		if result := e.checkDirectionReversal(labType, newValue, history); result != nil {
			return result, nil
		}
	}

	// Rule 3: Physiological bounds check (cross-lab impossible combinations)
	// This requires multiple lab types — deferred to caller integration

	return &PlausibilityResult{
		Action:     PlausibilityAccept,
		Confidence: 1.0,
	}, nil
}

// checkRateOfChange validates that the change from the most recent value
// is within physiologically possible bounds.
func (e *PlausibilityEngine) checkRateOfChange(
	labType string,
	newValue float64,
	newTimestamp time.Time,
	lastRecord LabHistoryRecord,
) *PlausibilityResult {
	hoursDelta := newTimestamp.Sub(lastRecord.Timestamp).Hours()
	if hoursDelta <= 0 {
		return nil // same timestamp or out of order, skip
	}

	valueDelta := math.Abs(newValue - lastRecord.Value)
	maxDelta := e.maxDeltaForLabType(labType, hoursDelta, lastRecord.Value)

	if maxDelta > 0 && valueDelta > maxDelta {
		severity := valueDelta / maxDelta // how many times over the limit
		action := PlausibilityFlagReview
		if severity > 3.0 {
			action = PlausibilityRejectRetest
		}

		return &PlausibilityResult{
			Action:     action,
			Confidence: math.Max(0.0, 1.0-severity*0.3),
			Reason: fmt.Sprintf(
				"%s changed by %.2f in %.1fh (max expected: %.2f); previous: %.2f, new: %.2f",
				labType, valueDelta, hoursDelta, maxDelta, lastRecord.Value, newValue,
			),
			RuleID: "PLAUSIBILITY_RATE_OF_CHANGE",
		}
	}

	return nil
}

// checkDirectionReversal detects implausible sudden reversals in trend direction.
func (e *PlausibilityEngine) checkDirectionReversal(
	labType string,
	newValue float64,
	history []LabHistoryRecord,
) *PlausibilityResult {
	minPoints := e.cfg.DirectionReversalMinPoints
	if len(history) < minPoints {
		return nil
	}

	// Check if last N points all trend in the same direction
	recent := history[:minPoints]
	allIncreasing := true
	allDecreasing := true

	for i := 0; i < len(recent)-1; i++ {
		if recent[i].Value <= recent[i+1].Value {
			allDecreasing = false
		}
		if recent[i].Value >= recent[i+1].Value {
			allIncreasing = false
		}
	}

	if !allIncreasing && !allDecreasing {
		return nil // no consistent trend
	}

	// Calculate mean and stddev of deltas
	var deltas []float64
	for i := 0; i < len(recent)-1; i++ {
		deltas = append(deltas, recent[i].Value-recent[i+1].Value)
	}

	mean, stddev := meanStdDev(deltas)
	newDelta := newValue - recent[0].Value

	// Check if the new value reverses direction AND exceeds threshold
	isReversal := (allIncreasing && newDelta < 0) || (allDecreasing && newDelta > 0)
	if !isReversal {
		return nil
	}

	// Check magnitude against historical variation
	deviation := math.Abs(newDelta-mean) / math.Max(stddev, 0.01)
	if deviation > e.cfg.DirectionReversalStdDevs {
		return &PlausibilityResult{
			Action:     PlausibilityFlagReview,
			Confidence: math.Max(0.0, 1.0-deviation*0.2),
			Reason: fmt.Sprintf(
				"%s sudden reversal: %.1f std devs from trend (trend mean delta: %.2f, new delta: %.2f)",
				labType, deviation, mean, newDelta,
			),
			RuleID: "PLAUSIBILITY_DIRECTION_REVERSAL",
		}
	}

	return nil
}

// maxDeltaForLabType returns the maximum expected change for a lab type
// over the given time period.
func (e *PlausibilityEngine) maxDeltaForLabType(labType string, hours float64, baseValue float64) float64 {
	days := hours / 24.0

	switch labType {
	case "EGFR":
		return e.cfg.EGFRMaxDeltaPerDay * math.Max(days, 1.0)
	case "CREATININE":
		return baseValue * (e.cfg.CreatinineMaxPctPer24h / 100.0) * math.Max(days, 1.0)
	case "POTASSIUM":
		return e.cfg.PotassiumMaxDeltaPerDay * math.Max(days, 1.0)
	case "GLUCOSE":
		return e.cfg.GlucoseMaxDeltaPerHour * math.Max(hours, 1.0)
	case "HBA1C":
		months := days / 30.0
		return e.cfg.HbA1cMaxDeltaPer30d * math.Max(months, 1.0)
	default:
		return 0 // unknown lab type, skip rate check
	}
}

// fetchHistory retrieves recent lab values for a patient, ordered newest first.
func (e *PlausibilityEngine) fetchHistory(
	ctx context.Context,
	patientID uuid.UUID,
	labType string,
	limit int,
) ([]LabHistoryRecord, error) {
	var records []LabHistoryRecord

	result := e.db.WithContext(ctx).
		Table("lab_results").
		Select("value, timestamp").
		Where("patient_id = ? AND lab_type = ?", patientID.String(), labType).
		Order("timestamp DESC").
		Limit(limit).
		Scan(&records)

	if result.Error != nil {
		return nil, result.Error
	}

	return records, nil
}

// meanStdDev computes mean and standard deviation of a float64 slice.
func meanStdDev(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))

	if len(vals) < 2 {
		return mean, 0
	}

	sumSq := 0.0
	for _, v := range vals {
		d := v - mean
		sumSq += d * d
	}
	stddev := math.Sqrt(sumSq / float64(len(vals)-1))

	return mean, stddev
}
