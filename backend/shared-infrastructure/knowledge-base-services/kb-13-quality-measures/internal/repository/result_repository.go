// Package repository provides database operations for KB-13.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"

	"kb-13-quality-measures/internal/models"
)

// ResultRepository handles calculation result persistence.
type ResultRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewResultRepository creates a new result repository.
func NewResultRepository(db *sql.DB, logger *zap.Logger) *ResultRepository {
	return &ResultRepository{
		db:     db,
		logger: logger,
	}
}

// Save persists a calculation result.
func (r *ResultRepository) Save(ctx context.Context, result *models.CalculationResult) error {
	query := `
		INSERT INTO calculation_results (
			id, measure_id, report_type, period_start, period_end,
			initial_population, denominator, denominator_exclusion, denominator_exception,
			numerator, numerator_exclusion, score, execution_time_ms,
			kb13_version, cql_library_version, terminology_version, measure_yaml_version,
			executed_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		result.ID,
		result.MeasureID,
		string(result.ReportType),
		result.PeriodStart,
		result.PeriodEnd,
		result.InitialPopulation,
		result.Denominator,
		result.DenominatorExclusion,
		result.DenominatorException,
		result.Numerator,
		result.NumeratorExclusion,
		result.Score,
		result.ExecutionTimeMs,
		result.ExecutionContext.KB13Version,
		result.ExecutionContext.CQLLibraryVersion,
		result.ExecutionContext.TerminologyVersion,
		result.ExecutionContext.MeasureYAMLVersion,
		result.ExecutionContext.ExecutedAt,
		result.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save calculation result",
			zap.String("result_id", result.ID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save result: %w", err)
	}

	r.logger.Debug("Saved calculation result",
		zap.String("result_id", result.ID),
		zap.String("measure_id", result.MeasureID),
	)

	return nil
}

// GetByID retrieves a calculation result by ID.
func (r *ResultRepository) GetByID(ctx context.Context, id string) (*models.CalculationResult, error) {
	query := `
		SELECT
			id, measure_id, report_type, period_start, period_end,
			initial_population, denominator, denominator_exclusion, denominator_exception,
			numerator, numerator_exclusion, score, execution_time_ms,
			kb13_version, cql_library_version, terminology_version, measure_yaml_version,
			executed_at, created_at
		FROM calculation_results
		WHERE id = $1
	`

	var result models.CalculationResult
	var reportType string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&result.ID,
		&result.MeasureID,
		&reportType,
		&result.PeriodStart,
		&result.PeriodEnd,
		&result.InitialPopulation,
		&result.Denominator,
		&result.DenominatorExclusion,
		&result.DenominatorException,
		&result.Numerator,
		&result.NumeratorExclusion,
		&result.Score,
		&result.ExecutionTimeMs,
		&result.ExecutionContext.KB13Version,
		&result.ExecutionContext.CQLLibraryVersion,
		&result.ExecutionContext.TerminologyVersion,
		&result.ExecutionContext.MeasureYAMLVersion,
		&result.ExecutionContext.ExecutedAt,
		&result.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("result not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	result.ReportType = models.ReportType(reportType)
	return &result, nil
}

// GetByMeasure retrieves calculation results for a measure.
func (r *ResultRepository) GetByMeasure(ctx context.Context, measureID string, limit int) ([]*models.CalculationResult, error) {
	query := `
		SELECT
			id, measure_id, report_type, period_start, period_end,
			initial_population, denominator, denominator_exclusion, denominator_exception,
			numerator, numerator_exclusion, score, execution_time_ms,
			kb13_version, cql_library_version, terminology_version, measure_yaml_version,
			executed_at, created_at
		FROM calculation_results
		WHERE measure_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, measureID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer rows.Close()

	var results []*models.CalculationResult
	for rows.Next() {
		var result models.CalculationResult
		var reportType string

		err := rows.Scan(
			&result.ID,
			&result.MeasureID,
			&reportType,
			&result.PeriodStart,
			&result.PeriodEnd,
			&result.InitialPopulation,
			&result.Denominator,
			&result.DenominatorExclusion,
			&result.DenominatorException,
			&result.Numerator,
			&result.NumeratorExclusion,
			&result.Score,
			&result.ExecutionTimeMs,
			&result.ExecutionContext.KB13Version,
			&result.ExecutionContext.CQLLibraryVersion,
			&result.ExecutionContext.TerminologyVersion,
			&result.ExecutionContext.MeasureYAMLVersion,
			&result.ExecutionContext.ExecutedAt,
			&result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		result.ReportType = models.ReportType(reportType)
		results = append(results, &result)
	}

	return results, nil
}

// GetLatestByMeasure retrieves the most recent result for a measure.
func (r *ResultRepository) GetLatestByMeasure(ctx context.Context, measureID string) (*models.CalculationResult, error) {
	results, err := r.GetByMeasure(ctx, measureID, 1)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for measure: %s", measureID)
	}
	return results[0], nil
}

// GetByPeriod retrieves results for a specific measurement period.
func (r *ResultRepository) GetByPeriod(ctx context.Context, start, end time.Time) ([]*models.CalculationResult, error) {
	query := `
		SELECT
			id, measure_id, report_type, period_start, period_end,
			initial_population, denominator, denominator_exclusion, denominator_exception,
			numerator, numerator_exclusion, score, execution_time_ms,
			kb13_version, cql_library_version, terminology_version, measure_yaml_version,
			executed_at, created_at
		FROM calculation_results
		WHERE period_start = $1 AND period_end = $2
		ORDER BY measure_id
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer rows.Close()

	var results []*models.CalculationResult
	for rows.Next() {
		var result models.CalculationResult
		var reportType string

		err := rows.Scan(
			&result.ID,
			&result.MeasureID,
			&reportType,
			&result.PeriodStart,
			&result.PeriodEnd,
			&result.InitialPopulation,
			&result.Denominator,
			&result.DenominatorExclusion,
			&result.DenominatorException,
			&result.Numerator,
			&result.NumeratorExclusion,
			&result.Score,
			&result.ExecutionTimeMs,
			&result.ExecutionContext.KB13Version,
			&result.ExecutionContext.CQLLibraryVersion,
			&result.ExecutionContext.TerminologyVersion,
			&result.ExecutionContext.MeasureYAMLVersion,
			&result.ExecutionContext.ExecutedAt,
			&result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		result.ReportType = models.ReportType(reportType)
		results = append(results, &result)
	}

	return results, nil
}

// Delete removes a calculation result.
func (r *ResultRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM calculation_results WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete result: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("result not found: %s", id)
	}

	return nil
}

// GetLatest retrieves the most recent result for a measure within a specific period.
func (r *ResultRepository) GetLatest(ctx context.Context, measureID string, periodStart, periodEnd time.Time) (*models.CalculationResult, error) {
	query := `
		SELECT
			id, measure_id, report_type, period_start, period_end,
			initial_population, denominator, denominator_exclusion, denominator_exception,
			numerator, numerator_exclusion, score, execution_time_ms,
			kb13_version, cql_library_version, terminology_version, measure_yaml_version,
			executed_at, created_at
		FROM calculation_results
		WHERE measure_id = $1 AND period_start = $2 AND period_end = $3
		ORDER BY created_at DESC
		LIMIT 1
	`

	var result models.CalculationResult
	var reportType string

	err := r.db.QueryRowContext(ctx, query, measureID, periodStart, periodEnd).Scan(
		&result.ID,
		&result.MeasureID,
		&reportType,
		&result.PeriodStart,
		&result.PeriodEnd,
		&result.InitialPopulation,
		&result.Denominator,
		&result.DenominatorExclusion,
		&result.DenominatorException,
		&result.Numerator,
		&result.NumeratorExclusion,
		&result.Score,
		&result.ExecutionTimeMs,
		&result.ExecutionContext.KB13Version,
		&result.ExecutionContext.CQLLibraryVersion,
		&result.ExecutionContext.TerminologyVersion,
		&result.ExecutionContext.MeasureYAMLVersion,
		&result.ExecutionContext.ExecutedAt,
		&result.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no result found for measure %s in period %s to %s",
			measureID, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	result.ReportType = models.ReportType(reportType)
	return &result, nil
}
