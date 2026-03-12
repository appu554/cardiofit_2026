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

// CareGapRepository handles care gap persistence.
// 🔴 CRITICAL: All care gaps from KB-13 are DERIVED, not authoritative.
// The authoritative source for individual patient care gaps is KB-9.
type CareGapRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewCareGapRepository creates a new care gap repository.
func NewCareGapRepository(db *sql.DB, logger *zap.Logger) *CareGapRepository {
	return &CareGapRepository{
		db:     db,
		logger: logger,
	}
}

// Save persists a care gap.
// 🔴 CRITICAL: Enforces Source = "QUALITY_MEASURE" and IsAuthoritative = false
func (r *CareGapRepository) Save(ctx context.Context, gap *models.CareGap) error {
	// Enforce KB-13 care gap constraints
	if gap.Source != models.CareGapSourceQualityMeasure {
		r.logger.Warn("Correcting care gap source to QUALITY_MEASURE",
			zap.String("gap_id", gap.ID),
			zap.String("original_source", string(gap.Source)),
		)
		gap.Source = models.CareGapSourceQualityMeasure
	}
	gap.IsAuthoritative = false

	query := `
		INSERT INTO care_gaps (
			id, measure_id, subject_id, gap_type, description, priority,
			status, due_date, intervention, source, is_authoritative, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			intervention = EXCLUDED.intervention
	`

	_, err := r.db.ExecContext(ctx, query,
		gap.ID,
		gap.MeasureID,
		gap.SubjectID,
		gap.GapType,
		gap.Description,
		string(gap.Priority),
		string(gap.Status),
		gap.DueDate,
		gap.Intervention,
		string(gap.Source),
		gap.IsAuthoritative,
		gap.CreatedAt,
	)

	if err != nil {
		r.logger.Error("Failed to save care gap",
			zap.String("gap_id", gap.ID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save care gap: %w", err)
	}

	r.logger.Debug("Saved care gap",
		zap.String("gap_id", gap.ID),
		zap.String("measure_id", gap.MeasureID),
		zap.String("subject_id", gap.SubjectID),
	)

	return nil
}

// SaveBatch persists multiple care gaps efficiently.
func (r *CareGapRepository) SaveBatch(ctx context.Context, gaps []*models.CareGap) error {
	if len(gaps) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO care_gaps (
			id, measure_id, subject_id, gap_type, description, priority,
			status, due_date, intervention, source, is_authoritative, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			intervention = EXCLUDED.intervention
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, gap := range gaps {
		// 🔴 CRITICAL: Enforce KB-13 constraints
		gap.Source = models.CareGapSourceQualityMeasure
		gap.IsAuthoritative = false

		_, err := stmt.ExecContext(ctx,
			gap.ID,
			gap.MeasureID,
			gap.SubjectID,
			gap.GapType,
			gap.Description,
			string(gap.Priority),
			string(gap.Status),
			gap.DueDate,
			gap.Intervention,
			string(gap.Source),
			gap.IsAuthoritative,
			gap.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert care gap %s: %w", gap.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.Info("Saved care gaps batch",
		zap.Int("count", len(gaps)),
	)

	return nil
}

// GetByID retrieves a care gap by ID.
func (r *CareGapRepository) GetByID(ctx context.Context, id string) (*models.CareGap, error) {
	query := `
		SELECT
			id, measure_id, subject_id, gap_type, description, priority,
			status, due_date, intervention, source, is_authoritative, created_at
		FROM care_gaps
		WHERE id = $1
	`

	var gap models.CareGap
	var priority, status, source string
	var dueDate sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&gap.ID,
		&gap.MeasureID,
		&gap.SubjectID,
		&gap.GapType,
		&gap.Description,
		&priority,
		&status,
		&dueDate,
		&gap.Intervention,
		&source,
		&gap.IsAuthoritative,
		&gap.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("care gap not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get care gap: %w", err)
	}

	gap.Priority = models.Priority(priority)
	gap.Status = models.CareGapStatus(status)
	gap.Source = models.CareGapSource(source)
	if dueDate.Valid {
		gap.DueDate = &dueDate.Time
	}

	return &gap, nil
}

// GetByMeasure retrieves care gaps for a measure.
func (r *CareGapRepository) GetByMeasure(ctx context.Context, measureID string, limit int) ([]*models.CareGap, error) {
	query := `
		SELECT
			id, measure_id, subject_id, gap_type, description, priority,
			status, due_date, intervention, source, is_authoritative, created_at
		FROM care_gaps
		WHERE measure_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	return r.queryGaps(ctx, query, measureID, limit)
}

// GetByPatient retrieves care gaps for a patient (subject).
func (r *CareGapRepository) GetByPatient(ctx context.Context, subjectID string) ([]*models.CareGap, error) {
	query := `
		SELECT
			id, measure_id, subject_id, gap_type, description, priority,
			status, due_date, intervention, source, is_authoritative, created_at
		FROM care_gaps
		WHERE subject_id = $1
		ORDER BY priority, created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, subjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to query care gaps: %w", err)
	}
	defer rows.Close()

	return r.scanGaps(rows)
}

// GetOpenGaps retrieves all open care gaps.
func (r *CareGapRepository) GetOpenGaps(ctx context.Context, limit int) ([]*models.CareGap, error) {
	query := `
		SELECT
			id, measure_id, subject_id, gap_type, description, priority,
			status, due_date, intervention, source, is_authoritative, created_at
		FROM care_gaps
		WHERE status = 'open'
		ORDER BY priority, due_date
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query open care gaps: %w", err)
	}
	defer rows.Close()

	return r.scanGaps(rows)
}

// GetOverdueGaps retrieves care gaps past their due date.
func (r *CareGapRepository) GetOverdueGaps(ctx context.Context) ([]*models.CareGap, error) {
	query := `
		SELECT
			id, measure_id, subject_id, gap_type, description, priority,
			status, due_date, intervention, source, is_authoritative, created_at
		FROM care_gaps
		WHERE status = 'open' AND due_date < $1
		ORDER BY priority, due_date
	`

	rows, err := r.db.QueryContext(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue care gaps: %w", err)
	}
	defer rows.Close()

	return r.scanGaps(rows)
}

// UpdateStatus updates a care gap's status.
func (r *CareGapRepository) UpdateStatus(ctx context.Context, id string, status models.CareGapStatus) error {
	query := `UPDATE care_gaps SET status = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update care gap status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("care gap not found: %s", id)
	}

	return nil
}

// GetSummaryByMeasure returns aggregate stats for a measure's care gaps.
func (r *CareGapRepository) GetSummaryByMeasure(ctx context.Context, measureID string) (*CareGapSummary, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'open') as open_count,
			COUNT(*) FILTER (WHERE status = 'closed') as closed_count,
			COUNT(*) FILTER (WHERE priority = 'high') as high_priority,
			COUNT(*) FILTER (WHERE priority = 'medium') as medium_priority,
			COUNT(*) FILTER (WHERE priority = 'low') as low_priority,
			COUNT(*) FILTER (WHERE status = 'open' AND due_date < NOW()) as overdue_count
		FROM care_gaps
		WHERE measure_id = $1
	`

	var summary CareGapSummary
	summary.MeasureID = measureID

	err := r.db.QueryRowContext(ctx, query, measureID).Scan(
		&summary.TotalGaps,
		&summary.OpenGaps,
		&summary.ClosedGaps,
		&summary.HighPriority,
		&summary.MediumPriority,
		&summary.LowPriority,
		&summary.OverdueGaps,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get care gap summary: %w", err)
	}

	return &summary, nil
}

// CareGapSummary provides aggregate statistics.
type CareGapSummary struct {
	MeasureID      string `json:"measure_id"`
	TotalGaps      int    `json:"total_gaps"`
	OpenGaps       int    `json:"open_gaps"`
	ClosedGaps     int    `json:"closed_gaps"`
	HighPriority   int    `json:"high_priority"`
	MediumPriority int    `json:"medium_priority"`
	LowPriority    int    `json:"low_priority"`
	OverdueGaps    int    `json:"overdue_gaps"`
}

// Helper to query gaps with two params
func (r *CareGapRepository) queryGaps(ctx context.Context, query string, param1 interface{}, param2 interface{}) ([]*models.CareGap, error) {
	rows, err := r.db.QueryContext(ctx, query, param1, param2)
	if err != nil {
		return nil, fmt.Errorf("failed to query care gaps: %w", err)
	}
	defer rows.Close()

	return r.scanGaps(rows)
}

// Helper to scan gap rows
func (r *CareGapRepository) scanGaps(rows *sql.Rows) ([]*models.CareGap, error) {
	var gaps []*models.CareGap

	for rows.Next() {
		var gap models.CareGap
		var priority, status, source string
		var dueDate sql.NullTime

		err := rows.Scan(
			&gap.ID,
			&gap.MeasureID,
			&gap.SubjectID,
			&gap.GapType,
			&gap.Description,
			&priority,
			&status,
			&dueDate,
			&gap.Intervention,
			&source,
			&gap.IsAuthoritative,
			&gap.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan care gap: %w", err)
		}

		gap.Priority = models.Priority(priority)
		gap.Status = models.CareGapStatus(status)
		gap.Source = models.CareGapSource(source)
		if dueDate.Valid {
			gap.DueDate = &dueDate.Time
		}

		gaps = append(gaps, &gap)
	}

	return gaps, nil
}

// Delete removes a care gap.
func (r *CareGapRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM care_gaps WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete care gap: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("care gap not found: %s", id)
	}

	return nil
}
