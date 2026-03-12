// Package calculator provides the quality measure calculation engine for KB-13.
//
// 🔴 CRITICAL ARCHITECTURE (CTO/CMO Gate):
//   - All calculations use BATCH CQL evaluation (never per-patient)
//   - All date logic goes through period.Resolver
//   - All results include ExecutionContextVersion for audit
//
// The engine orchestrates:
//   - Measure definition loading
//   - CQL batch evaluation
//   - Population count aggregation
//   - Score calculation
//   - Care gap identification
package calculator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-13-quality-measures/internal/config"
	"kb-13-quality-measures/internal/cql"
	"kb-13-quality-measures/internal/models"
	"kb-13-quality-measures/internal/period"
)

// Engine orchestrates quality measure calculations.
type Engine struct {
	cqlClient      *cql.Client
	periodResolver *period.Resolver
	store          *models.MeasureStore
	config         *config.CalculatorConfig
	logger         *zap.Logger

	// Job tracking
	jobs   map[string]*models.CalculationJob
	jobsMu sync.RWMutex

	// Execution context for audit
	kb13Version        string
	cqlLibraryVersion  string
	terminologyVersion string
}

// NewEngine creates a new calculation engine.
func NewEngine(
	cqlClient *cql.Client,
	store *models.MeasureStore,
	cfg *config.CalculatorConfig,
	logger *zap.Logger,
) *Engine {
	return &Engine{
		cqlClient:          cqlClient,
		periodResolver:     period.NewResolver(),
		store:              store,
		config:             cfg,
		logger:             logger,
		jobs:               make(map[string]*models.CalculationJob),
		kb13Version:        config.Version,
		cqlLibraryVersion:  "1.0.0", // TODO: Get from CQL engine
		terminologyVersion: "2024-01", // TODO: Get from KB-7
	}
}

// CalculateRequest defines input for measure calculation.
type CalculateRequest struct {
	MeasureID   string
	ReportType  models.ReportType
	PeriodStart *time.Time // Optional, uses resolver if nil
	PeriodEnd   *time.Time // Optional, uses resolver if nil
	Year        int        // For calendar year calculations
}

// Calculate performs a synchronous measure calculation.
func (e *Engine) Calculate(ctx context.Context, req *CalculateRequest) (*models.CalculationResult, error) {
	startTime := time.Now()

	e.logger.Info("Starting measure calculation",
		zap.String("measure_id", req.MeasureID),
		zap.String("report_type", string(req.ReportType)),
	)

	// Get measure definition
	measure := e.store.GetMeasure(req.MeasureID)
	if measure == nil {
		return nil, fmt.Errorf("measure not found: %s", req.MeasureID)
	}

	// 🔴 CRITICAL: Resolve measurement period using the dedicated module
	measurementPeriod := e.resolvePeriod(req, measure)

	e.logger.Debug("Resolved measurement period",
		zap.String("measure_id", req.MeasureID),
		zap.Time("period_start", measurementPeriod.Start),
		zap.Time("period_end", measurementPeriod.End),
		zap.String("period_label", measurementPeriod.Label),
	)

	// Build CQL evaluation request
	cqlReq := e.buildCQLRequest(measure, measurementPeriod)

	// 🔴 CRITICAL: Execute batch CQL evaluation (never per-patient)
	cqlResult, err := e.cqlClient.EvaluateMeasure(ctx, cqlReq)
	if err != nil {
		return nil, fmt.Errorf("CQL evaluation failed: %w", err)
	}

	// Calculate score and build result
	result := e.buildResult(measure, measurementPeriod, cqlResult, req.ReportType, startTime)

	e.logger.Info("Measure calculation completed",
		zap.String("measure_id", req.MeasureID),
		zap.Int("initial_population", result.InitialPopulation),
		zap.Int("denominator", result.Denominator),
		zap.Int("numerator", result.Numerator),
		zap.Float64("score", result.Score),
		zap.Int64("execution_time_ms", result.ExecutionTimeMs),
	)

	return result, nil
}

// CalculateAsync starts an async calculation job.
func (e *Engine) CalculateAsync(ctx context.Context, req *CalculateRequest) (*models.CalculationJob, error) {
	// Create job
	jobID := uuid.New().String()
	now := time.Now()

	job := &models.CalculationJob{
		ID:         jobID,
		MeasureID:  req.MeasureID,
		ReportType: req.ReportType,
		Status:     "pending",
		Progress:   0,
		CreatedAt:  now,
	}

	// Store job
	e.jobsMu.Lock()
	e.jobs[jobID] = job
	e.jobsMu.Unlock()

	// Start calculation in background
	go e.runAsyncCalculation(ctx, job, req)

	return job, nil
}

// runAsyncCalculation executes the calculation in background.
func (e *Engine) runAsyncCalculation(ctx context.Context, job *models.CalculationJob, req *CalculateRequest) {
	// Update status to running
	e.updateJobStatus(job.ID, "running", 10, nil, nil)

	// Perform calculation
	result, err := e.Calculate(ctx, req)

	if err != nil {
		e.updateJobStatus(job.ID, "failed", 0, nil, &err)
		return
	}

	// Update with result
	e.updateJobStatus(job.ID, "completed", 100, result, nil)
}

// updateJobStatus updates a job's status.
func (e *Engine) updateJobStatus(jobID, status string, progress int, result *models.CalculationResult, err *error) {
	e.jobsMu.Lock()
	defer e.jobsMu.Unlock()

	job, exists := e.jobs[jobID]
	if !exists {
		return
	}

	job.Status = status
	job.Progress = progress
	job.Result = result

	if err != nil {
		job.Error = (*err).Error()
	}

	now := time.Now()
	if status == "running" {
		job.StartedAt = &now
	}
	if status == "completed" || status == "failed" {
		job.CompletedAt = &now
	}
}

// GetJob retrieves a calculation job by ID.
func (e *Engine) GetJob(jobID string) (*models.CalculationJob, error) {
	e.jobsMu.RLock()
	defer e.jobsMu.RUnlock()

	job, exists := e.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// resolvePeriod determines the measurement period.
// 🔴 CRITICAL: All date logic MUST go through period.Resolver
func (e *Engine) resolvePeriod(req *CalculateRequest, measure *models.Measure) *period.MeasurementPeriod {
	// If explicit dates provided, use them
	if req.PeriodStart != nil && req.PeriodEnd != nil {
		return &period.MeasurementPeriod{
			Start: *req.PeriodStart,
			End:   *req.PeriodEnd,
			Type:  period.PeriodTypeCalendar,
			Label: "Custom Period",
		}
	}

	// If year specified, use calendar year
	if req.Year > 0 {
		return e.periodResolver.ResolveForYear(req.Year)
	}

	// Use measure's configured period type
	cfg := period.Config{
		Type:     period.PeriodType(measure.MeasurementPeriod.Type),
		Duration: measure.MeasurementPeriod.Duration,
		Anchor:   period.AnchorType(measure.MeasurementPeriod.Anchor),
	}

	resolved, err := e.periodResolver.Resolve(cfg)
	if err != nil {
		// Fallback to current calendar year
		e.logger.Warn("Failed to resolve period, using current year",
			zap.String("measure_id", measure.ID),
			zap.Error(err),
		)
		return e.periodResolver.CurrentCalendarYear()
	}

	return resolved
}

// buildCQLRequest constructs the CQL evaluation request from measure definition.
func (e *Engine) buildCQLRequest(measure *models.Measure, mp *period.MeasurementPeriod) *cql.MeasureEvaluationRequest {
	populations := make([]cql.PopulationEvaluation, 0, len(measure.Populations))

	for _, pop := range measure.Populations {
		populations = append(populations, cql.PopulationEvaluation{
			PopulationType: string(pop.Type),
			CQLExpression:  pop.CQLExpression,
		})
	}

	return &cql.MeasureEvaluationRequest{
		MeasureID:   measure.ID,
		PeriodStart: mp.Start.Format(time.RFC3339),
		PeriodEnd:   mp.End.Format(time.RFC3339),
		Populations: populations,
	}
}

// buildResult constructs the calculation result from CQL response.
func (e *Engine) buildResult(
	measure *models.Measure,
	mp *period.MeasurementPeriod,
	cqlResult *cql.MeasureEvaluationResponse,
	reportType models.ReportType,
	startTime time.Time,
) *models.CalculationResult {
	result := &models.CalculationResult{
		ID:          uuid.New().String(),
		MeasureID:   measure.ID,
		ReportType:  reportType,
		PeriodStart: mp.Start,
		PeriodEnd:   mp.End,
		CreatedAt:   time.Now(),
	}

	// Extract population counts from CQL result
	if pop, ok := cqlResult.Populations["initial-population"]; ok {
		result.InitialPopulation = pop.Count
	}
	if pop, ok := cqlResult.Populations["denominator"]; ok {
		result.Denominator = pop.Count
	}
	if pop, ok := cqlResult.Populations["denominator-exclusion"]; ok {
		result.DenominatorExclusion = pop.Count
	}
	if pop, ok := cqlResult.Populations["denominator-exception"]; ok {
		result.DenominatorException = pop.Count
	}
	if pop, ok := cqlResult.Populations["numerator"]; ok {
		result.Numerator = pop.Count
	}
	if pop, ok := cqlResult.Populations["numerator-exclusion"]; ok {
		result.NumeratorExclusion = pop.Count
	}

	// Calculate score
	result.Score = e.calculateScore(result, measure)

	// Add execution context (🟡 REQUIRED per CTO/CMO gate)
	result.ExecutionContext = models.ExecutionContextVersion{
		KB13Version:        e.kb13Version,
		CQLLibraryVersion:  cqlResult.EngineVersion,
		TerminologyVersion: e.terminologyVersion,
		MeasureYAMLVersion: measure.Version,
		ExecutedAt:         time.Now(),
	}

	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	return result
}

// calculateScore computes the measure score based on scoring type.
func (e *Engine) calculateScore(result *models.CalculationResult, measure *models.Measure) float64 {
	// Adjusted denominator = denominator - exclusions - exceptions
	adjustedDenom := result.Denominator - result.DenominatorExclusion - result.DenominatorException
	if adjustedDenom <= 0 {
		return 0.0
	}

	// Adjusted numerator = numerator - exclusions
	adjustedNum := result.Numerator - result.NumeratorExclusion

	switch measure.Scoring {
	case models.ScoringProportion:
		return float64(adjustedNum) / float64(adjustedDenom)
	case models.ScoringRatio:
		// For ratio measures, denominator might be different
		return float64(adjustedNum) / float64(adjustedDenom)
	case models.ScoringContinuous:
		// Continuous measures need different handling
		return float64(adjustedNum)
	default:
		return float64(adjustedNum) / float64(adjustedDenom)
	}
}

// CalculateBatch calculates multiple measures in parallel.
func (e *Engine) CalculateBatch(ctx context.Context, measureIDs []string, year int) ([]*models.CalculationResult, error) {
	var wg sync.WaitGroup
	results := make([]*models.CalculationResult, len(measureIDs))
	errors := make([]error, len(measureIDs))

	// Limit concurrent calculations
	sem := make(chan struct{}, e.config.MaxConcurrent)

	for i, measureID := range measureIDs {
		wg.Add(1)
		go func(idx int, mID string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			req := &CalculateRequest{
				MeasureID:  mID,
				ReportType: models.ReportSummary,
				Year:       year,
			}

			result, err := e.Calculate(ctx, req)
			results[idx] = result
			errors[idx] = err
		}(i, measureID)
	}

	wg.Wait()

	// Collect any errors
	var errs []string
	for i, err := range errors {
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", measureIDs[i], err))
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("batch calculation had %d errors: %v", len(errs), errs)
	}

	return results, nil
}
