// Package baseline provides patient-specific baseline tracking and calculation
package baseline

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-16-lab-interpretation/pkg/store"
	"kb-16-lab-interpretation/pkg/types"
)

// Tracker manages patient baselines
type Tracker struct {
	db          *gorm.DB
	resultStore *store.ResultStore
	log         *logrus.Entry
}

// NewTracker creates a new baseline tracker
func NewTracker(db *gorm.DB, resultStore *store.ResultStore, log *logrus.Entry) *Tracker {
	return &Tracker{
		db:          db,
		resultStore: resultStore,
		log:         log.WithField("component", "baseline_tracker"),
	}
}

// GetBaseline retrieves a patient's baseline for a specific test
func (t *Tracker) GetBaseline(ctx context.Context, patientID, code string) (*types.PatientBaseline, error) {
	var baseline types.PatientBaseline
	err := t.db.WithContext(ctx).
		Where("patient_id = ? AND code = ?", patientID, code).
		First(&baseline).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get baseline: %w", err)
	}

	return &baseline, nil
}

// GetAllBaselines retrieves all baselines for a patient
func (t *Tracker) GetAllBaselines(ctx context.Context, patientID string) ([]types.PatientBaseline, error) {
	var baselines []types.PatientBaseline
	err := t.db.WithContext(ctx).
		Where("patient_id = ?", patientID).
		Find(&baselines).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get baselines: %w", err)
	}

	return baselines, nil
}

// CalculateBaseline computes a baseline from historical data
func (t *Tracker) CalculateBaseline(ctx context.Context, patientID, code string, lookbackDays int) (*types.PatientBaseline, error) {
	if lookbackDays <= 0 {
		lookbackDays = 365 // Default to 1 year
	}

	// Get historical results
	results, err := t.resultStore.GetByPatientAndCode(ctx, patientID, code, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical results: %w", err)
	}

	if len(results) < 3 {
		return nil, fmt.Errorf("insufficient data points for baseline calculation: need at least 3, have %d", len(results))
	}

	// Extract numeric values
	values := make([]float64, 0, len(results))
	for _, r := range results {
		if r.ValueNumeric != nil {
			values = append(values, *r.ValueNumeric)
		}
	}

	if len(values) < 3 {
		return nil, fmt.Errorf("insufficient numeric values for baseline calculation")
	}

	// Filter stable period (exclude values during acute illness if detectable)
	stableValues := t.filterStablePeriod(values)

	// Remove outliers using IQR method
	cleanValues := t.removeOutliers(stableValues)

	if len(cleanValues) < 2 {
		cleanValues = stableValues // Fall back to unfiltered if too aggressive
	}

	// Calculate statistics
	mean := t.mean(cleanValues)
	stdDev := t.stdDev(cleanValues, mean)
	min, max := t.minMax(cleanValues)

	baseline := &types.PatientBaseline{
		ID:            uuid.New(),
		PatientID:     patientID,
		Code:          code,
		Mean:          mean,
		StdDev:        stdDev,
		Min:           min,
		Max:           max,
		SampleCount:   len(cleanValues),
		Source:        types.BaselineSourceCalculated,
		LastUpdated:   time.Now(),
		LookbackDays:  lookbackDays,
	}

	// Save or update baseline
	err = t.saveBaseline(ctx, baseline)
	if err != nil {
		return nil, err
	}

	t.log.WithFields(logrus.Fields{
		"patient_id":   patientID,
		"code":         code,
		"mean":         mean,
		"std_dev":      stdDev,
		"sample_count": len(cleanValues),
	}).Info("Calculated baseline")

	return baseline, nil
}

// SetManualBaseline sets a manually specified baseline
func (t *Tracker) SetManualBaseline(ctx context.Context, patientID, code string, mean, stdDev float64, setBy string) (*types.PatientBaseline, error) {
	baseline := &types.PatientBaseline{
		ID:          uuid.New(),
		PatientID:   patientID,
		Code:        code,
		Mean:        mean,
		StdDev:      stdDev,
		Min:         mean - 2*stdDev,
		Max:         mean + 2*stdDev,
		SampleCount: 0,
		Source:      types.BaselineSourceManual,
		SetBy:       setBy,
		LastUpdated: time.Now(),
	}

	err := t.saveBaseline(ctx, baseline)
	if err != nil {
		return nil, err
	}

	t.log.WithFields(logrus.Fields{
		"patient_id": patientID,
		"code":       code,
		"mean":       mean,
		"set_by":     setBy,
	}).Info("Set manual baseline")

	return baseline, nil
}

// CompareToBaseline compares a value to the patient's baseline
func (t *Tracker) CompareToBaseline(ctx context.Context, patientID, code string, value float64) (*types.BaselineComparison, error) {
	baseline, err := t.GetBaseline(ctx, patientID, code)
	if err != nil {
		return nil, err
	}

	if baseline == nil {
		// Try to calculate baseline
		baseline, err = t.CalculateBaseline(ctx, patientID, code, 365)
		if err != nil {
			return nil, fmt.Errorf("no baseline available and cannot calculate: %w", err)
		}
	}

	// Calculate z-score
	zScore := 0.0
	if baseline.StdDev > 0 {
		zScore = (value - baseline.Mean) / baseline.StdDev
	}

	// Calculate percent deviation
	percentDeviation := 0.0
	if baseline.Mean != 0 {
		percentDeviation = ((value - baseline.Mean) / baseline.Mean) * 100
	}

	// Determine if significant deviation
	isSignificant := math.Abs(zScore) > 2.0

	// Determine direction
	direction := "within"
	if zScore > 2.0 {
		direction = "above"
	} else if zScore < -2.0 {
		direction = "below"
	}

	return &types.BaselineComparison{
		Baseline:         *baseline,
		CurrentValue:     value,
		ZScore:           zScore,
		PercentDeviation: percentDeviation,
		IsSignificant:    isSignificant,
		Direction:        direction,
	}, nil
}

// saveBaseline saves or updates a baseline
func (t *Tracker) saveBaseline(ctx context.Context, baseline *types.PatientBaseline) error {
	// Try to find existing
	var existing types.PatientBaseline
	err := t.db.WithContext(ctx).
		Where("patient_id = ? AND code = ?", baseline.PatientID, baseline.Code).
		First(&existing).Error

	if err == nil {
		// Update existing
		baseline.ID = existing.ID
		return t.db.WithContext(ctx).Save(baseline).Error
	}

	if err == gorm.ErrRecordNotFound {
		// Create new
		return t.db.WithContext(ctx).Create(baseline).Error
	}

	return err
}

// filterStablePeriod filters out values that may be from acute illness
// This is a simplified implementation - could be enhanced with clinical rules
func (t *Tracker) filterStablePeriod(values []float64) []float64 {
	if len(values) < 5 {
		return values // Not enough data to filter
	}

	// Use values within 1.5 IQR of median as "stable"
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	q1 := sorted[len(sorted)/4]
	q3 := sorted[3*len(sorted)/4]
	iqr := q3 - q1

	lower := q1 - 1.5*iqr
	upper := q3 + 1.5*iqr

	stable := make([]float64, 0, len(values))
	for _, v := range values {
		if v >= lower && v <= upper {
			stable = append(stable, v)
		}
	}

	return stable
}

// removeOutliers removes outliers using IQR method
func (t *Tracker) removeOutliers(values []float64) []float64 {
	if len(values) < 4 {
		return values
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	q1 := sorted[len(sorted)/4]
	q3 := sorted[3*len(sorted)/4]
	iqr := q3 - q1

	lower := q1 - 1.5*iqr
	upper := q3 + 1.5*iqr

	clean := make([]float64, 0, len(values))
	for _, v := range values {
		if v >= lower && v <= upper {
			clean = append(clean, v)
		}
	}

	return clean
}

// Statistical helper functions

func (t *Tracker) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (t *Tracker) stdDev(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

func (t *Tracker) minMax(values []float64) (min, max float64) {
	if len(values) == 0 {
		return 0, 0
	}
	min, max = values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return
}
