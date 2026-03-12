// Package calculator provides care gap detection for KB-13.
//
// 🔴 CRITICAL: KB-13 care gaps are DERIVED, not authoritative.
// The authoritative source for individual patient care gaps is KB-9.
// KB-13 care gaps are for population-level reporting and analysis only.
package calculator

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-13-quality-measures/internal/cql"
	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/period"
)

// CareGapDetector identifies care gaps from measure calculations.
type CareGapDetector struct {
	cqlClient *cql.Client
	logger    *zap.Logger
}

// NewCareGapDetector creates a new care gap detector.
func NewCareGapDetector(cqlClient *cql.Client, logger *zap.Logger) *CareGapDetector {
	return &CareGapDetector{
		cqlClient: cqlClient,
		logger:    logger,
	}
}

// DetectionRequest defines input for care gap detection.
type DetectionRequest struct {
	MeasureID string
	Measure   *models.Measure
	Period    *period.MeasurementPeriod

	// Population results from calculation
	DenominatorPatientIDs []string
	NumeratorPatientIDs   []string
}

// DetectCareGaps identifies patients with care gaps for a measure.
// 🔴 CRITICAL: All gaps are marked as DERIVED with Source = "QUALITY_MEASURE"
func (d *CareGapDetector) DetectCareGaps(ctx context.Context, req *DetectionRequest) ([]*models.CareGap, error) {
	d.logger.Debug("Detecting care gaps",
		zap.String("measure_id", req.MeasureID),
		zap.Int("denominator_count", len(req.DenominatorPatientIDs)),
		zap.Int("numerator_count", len(req.NumeratorPatientIDs)),
	)

	// Create lookup set for numerator patients
	numeratorSet := make(map[string]bool, len(req.NumeratorPatientIDs))
	for _, id := range req.NumeratorPatientIDs {
		numeratorSet[id] = true
	}

	// Find patients in denominator but not in numerator (care gaps)
	var gaps []*models.CareGap
	for _, patientID := range req.DenominatorPatientIDs {
		if !numeratorSet[patientID] {
			gap := d.createCareGap(req.Measure, patientID, req.Period)
			gaps = append(gaps, gap)
		}
	}

	d.logger.Info("Care gaps detected",
		zap.String("measure_id", req.MeasureID),
		zap.Int("gap_count", len(gaps)),
	)

	return gaps, nil
}

// createCareGap creates a new care gap with proper source annotation.
// 🔴 CRITICAL: Uses NewCareGap to ensure Source = "QUALITY_MEASURE"
func (d *CareGapDetector) createCareGap(
	measure *models.Measure,
	patientID string,
	mp *period.MeasurementPeriod,
) *models.CareGap {
	// Determine priority based on measure type
	priority := d.determinePriority(measure)

	// Create gap with proper annotation
	gap := models.NewCareGap(
		measure.ID,
		patientID,
		d.determineGapType(measure),
		d.buildGapDescription(measure),
		priority,
	)

	// Set unique ID
	gap.ID = uuid.New().String()

	// Set due date (end of measurement period)
	gap.DueDate = &mp.End

	// Set intervention based on measure
	gap.Intervention = d.suggestIntervention(measure)

	return gap
}

// determinePriority assigns priority based on measure characteristics.
func (d *CareGapDetector) determinePriority(measure *models.Measure) models.Priority {
	// High priority for outcome measures and certain domains
	if measure.Type == models.MeasureTypeOutcome {
		return models.PriorityHigh
	}

	switch measure.Domain {
	case models.DomainDiabetes, models.DomainCardiovascular:
		return models.PriorityHigh
	case models.DomainPreventive:
		return models.PriorityMedium
	default:
		return models.PriorityMedium
	}
}

// determineGapType categorizes the care gap type.
func (d *CareGapDetector) determineGapType(measure *models.Measure) string {
	switch measure.Type {
	case models.MeasureTypeProcess:
		return "process_gap"
	case models.MeasureTypeOutcome:
		return "outcome_gap"
	case models.MeasureTypeStructure:
		return "structural_gap"
	default:
		return "quality_gap"
	}
}

// buildGapDescription creates a human-readable description.
func (d *CareGapDetector) buildGapDescription(measure *models.Measure) string {
	return "Quality measure gap: " + measure.Title
}

// suggestIntervention provides guidance for closing the gap.
func (d *CareGapDetector) suggestIntervention(measure *models.Measure) string {
	// Use measure's evidence guidelines if available
	if measure.Evidence.Guideline != "" {
		return "Review per guideline: " + measure.Evidence.Guideline
	}

	// Generic interventions by domain
	switch measure.Domain {
	case models.DomainDiabetes:
		return "Schedule diabetes care visit; review HbA1c, medications"
	case models.DomainCardiovascular:
		return "Schedule cardiovascular assessment; review blood pressure, lipids"
	case models.DomainPreventive:
		return "Schedule preventive care visit; review screening status"
	default:
		return "Review patient record for quality measure compliance"
	}
}

// BulkDetectCareGaps detects care gaps for multiple measures efficiently.
func (d *CareGapDetector) BulkDetectCareGaps(
	ctx context.Context,
	requests []*DetectionRequest,
) (map[string][]*models.CareGap, error) {
	results := make(map[string][]*models.CareGap, len(requests))

	for _, req := range requests {
		gaps, err := d.DetectCareGaps(ctx, req)
		if err != nil {
			d.logger.Error("Failed to detect care gaps for measure",
				zap.String("measure_id", req.MeasureID),
				zap.Error(err),
			)
			continue
		}
		results[req.MeasureID] = gaps
	}

	return results, nil
}

// CareGapSummary provides aggregate statistics for care gaps.
type CareGapSummary struct {
	MeasureID     string         `json:"measure_id"`
	TotalGaps     int            `json:"total_gaps"`
	ByPriority    map[string]int `json:"by_priority"`
	ByStatus      map[string]int `json:"by_status"`
	AverageAgeDays float64       `json:"average_age_days"`
}

// SummarizeCareGaps generates summary statistics for a set of care gaps.
func (d *CareGapDetector) SummarizeCareGaps(gaps []*models.CareGap) *CareGapSummary {
	summary := &CareGapSummary{
		TotalGaps:  len(gaps),
		ByPriority: make(map[string]int),
		ByStatus:   make(map[string]int),
	}

	if len(gaps) == 0 {
		return summary
	}

	summary.MeasureID = gaps[0].MeasureID

	var totalAgeDays float64
	for _, gap := range gaps {
		summary.ByPriority[string(gap.Priority)]++
		summary.ByStatus[string(gap.Status)]++

		// Calculate age in days
		ageDays := time.Since(gap.CreatedAt).Hours() / 24
		totalAgeDays += ageDays
	}

	summary.AverageAgeDays = totalAgeDays / float64(len(gaps))

	return summary
}
