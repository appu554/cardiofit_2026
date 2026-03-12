// Package reporter provides quality measure report generation for KB-13.
//
// Reports are generated in FHIR R4 MeasureReport format and can be:
//   - Individual: Single patient results
//   - Subject-List: List of patients meeting criteria
//   - Summary: Aggregate statistics
//   - Data-Exchange: For quality reporting submission
package reporter

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/repository"
)

// Reporter generates quality measure reports.
type Reporter struct {
	resultRepo *repository.ResultRepository
	logger     *zap.Logger
}

// NewReporter creates a new reporter.
func NewReporter(resultRepo *repository.ResultRepository, logger *zap.Logger) *Reporter {
	return &Reporter{
		resultRepo: resultRepo,
		logger:     logger,
	}
}

// Report represents a generated quality measure report.
type Report struct {
	ID              string           `json:"id"`
	MeasureID       string           `json:"measure_id"`
	MeasureName     string           `json:"measure_name"`
	ReportType      models.ReportType `json:"report_type"`
	Status          string           `json:"status"` // complete, pending, error
	PeriodStart     time.Time        `json:"period_start"`
	PeriodEnd       time.Time        `json:"period_end"`
	GeneratedAt     time.Time        `json:"generated_at"`

	// Population counts
	InitialPopulation     int `json:"initial_population"`
	Denominator           int `json:"denominator"`
	DenominatorExclusion  int `json:"denominator_exclusion"`
	DenominatorException  int `json:"denominator_exception"`
	Numerator             int `json:"numerator"`
	NumeratorExclusion    int `json:"numerator_exclusion"`

	// Calculated metrics
	Score          float64 `json:"score"`
	PerformanceRate float64 `json:"performance_rate"`

	// Comparison data
	PriorPeriodScore *float64 `json:"prior_period_score,omitempty"`
	BenchmarkScore   *float64 `json:"benchmark_score,omitempty"`

	// Stratifications
	Stratifications []StratificationResult `json:"stratifications,omitempty"`

	// Metadata
	ExecutionContext models.ExecutionContextVersion `json:"execution_context"`
}

// StratificationResult represents stratified results.
type StratificationResult struct {
	ID          string  `json:"id"`
	Description string  `json:"description"`
	Value       string  `json:"value"`
	Count       int     `json:"count"`
	Score       float64 `json:"score"`
}

// GenerateRequest specifies parameters for report generation.
type GenerateRequest struct {
	MeasureID   string
	ReportType  models.ReportType
	PeriodStart time.Time
	PeriodEnd   time.Time
	SubjectID   string // For individual reports
	IncludePriorPeriod bool
	IncludeBenchmark   bool
}

// Generate creates a quality measure report.
func (r *Reporter) Generate(ctx context.Context, req *GenerateRequest) (*Report, error) {
	r.logger.Info("Generating quality measure report",
		zap.String("measure_id", req.MeasureID),
		zap.String("report_type", string(req.ReportType)),
	)

	// Get the most recent calculation result for this measure and period
	result, err := r.resultRepo.GetLatest(ctx, req.MeasureID, req.PeriodStart, req.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("no calculation result found: %w", err)
	}

	report := &Report{
		ID:                   uuid.New().String(),
		MeasureID:            result.MeasureID,
		ReportType:           req.ReportType,
		Status:               "complete",
		PeriodStart:          result.PeriodStart,
		PeriodEnd:            result.PeriodEnd,
		GeneratedAt:          time.Now(),
		InitialPopulation:    result.InitialPopulation,
		Denominator:          result.Denominator,
		DenominatorExclusion: result.DenominatorExclusion,
		DenominatorException: result.DenominatorException,
		Numerator:            result.Numerator,
		NumeratorExclusion:   result.NumeratorExclusion,
		Score:                result.Score,
		PerformanceRate:      result.Score * 100, // Convert to percentage
		ExecutionContext:     result.ExecutionContext,
	}

	// Get prior period score if requested
	if req.IncludePriorPeriod {
		priorStart := req.PeriodStart.AddDate(-1, 0, 0)
		priorEnd := req.PeriodEnd.AddDate(-1, 0, 0)
		if priorResult, err := r.resultRepo.GetLatest(ctx, req.MeasureID, priorStart, priorEnd); err == nil {
			report.PriorPeriodScore = &priorResult.Score
		}
	}

	r.logger.Info("Report generated successfully",
		zap.String("report_id", report.ID),
		zap.String("measure_id", report.MeasureID),
		zap.Float64("score", report.Score),
	)

	return report, nil
}

// GetReport retrieves a previously generated report by ID.
func (r *Reporter) GetReport(ctx context.Context, reportID string) (*Report, error) {
	// In a full implementation, reports would be persisted
	// For now, return not found
	return nil, fmt.Errorf("report not found: %s", reportID)
}

// ListReports returns reports for a measure.
func (r *Reporter) ListReports(ctx context.Context, measureID string, limit int) ([]*Report, error) {
	results, err := r.resultRepo.GetByMeasure(ctx, measureID, limit)
	if err != nil {
		return nil, err
	}

	reports := make([]*Report, len(results))
	for i, result := range results {
		reports[i] = &Report{
			ID:                   result.ID,
			MeasureID:            result.MeasureID,
			ReportType:           result.ReportType,
			Status:               "complete",
			PeriodStart:          result.PeriodStart,
			PeriodEnd:            result.PeriodEnd,
			GeneratedAt:          result.CreatedAt,
			InitialPopulation:    result.InitialPopulation,
			Denominator:          result.Denominator,
			DenominatorExclusion: result.DenominatorExclusion,
			DenominatorException: result.DenominatorException,
			Numerator:            result.Numerator,
			NumeratorExclusion:   result.NumeratorExclusion,
			Score:                result.Score,
			PerformanceRate:      result.Score * 100,
			ExecutionContext:     result.ExecutionContext,
		}
	}

	return reports, nil
}

// GetLatestReport returns the most recent report for a measure.
func (r *Reporter) GetLatestReport(ctx context.Context, measureID string) (*Report, error) {
	reports, err := r.ListReports(ctx, measureID, 1)
	if err != nil {
		return nil, err
	}
	if len(reports) == 0 {
		return nil, fmt.Errorf("no reports found for measure: %s", measureID)
	}
	return reports[0], nil
}
