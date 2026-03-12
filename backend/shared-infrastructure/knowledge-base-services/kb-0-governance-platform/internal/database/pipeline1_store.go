package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"kb-0-governance-platform/internal/pipeline1"
)

// =============================================================================
// PIPELINE 1 STORE
// =============================================================================

// Pipeline1Store provides PostgreSQL persistence for Pipeline 1 review data.
type Pipeline1Store struct {
	db *sql.DB
}

// NewPipeline1Store creates a new Pipeline 1 store.
func NewPipeline1Store(db *sql.DB) *Pipeline1Store {
	return &Pipeline1Store{db: db}
}

// =============================================================================
// JOB OPERATIONS
// =============================================================================

// GetJobs returns extraction jobs from the progress view with pagination.
func (s *Pipeline1Store) GetJobs(ctx context.Context, page, pageSize int) ([]*pipeline1.ExtractionJob, int, error) {
	// Count total
	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM l2_extraction_jobs").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	offset := (page - 1) * pageSize
	query := `
		SELECT
			job_id, source_pdf, page_range, pipeline_version, l1_tag,
			total_merged_spans, total_sections, total_pages,
			status, created_at, updated_at, completed_at,
			spans_confirmed, spans_rejected, spans_edited, spans_added, spans_pending,
			completion_pct
		FROM v_l2_job_progress
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*pipeline1.ExtractionJob
	for rows.Next() {
		j := &pipeline1.ExtractionJob{}
		var pageRange, l1Tag sql.NullString
		var completedAt sql.NullTime
		var completionPct sql.NullFloat64

		if err := rows.Scan(
			&j.JobID, &j.SourcePDF, &pageRange, &j.PipelineVersion, &l1Tag,
			&j.TotalMergedSpans, &j.TotalSections, &j.TotalPages,
			&j.Status, &j.CreatedAt, &j.UpdatedAt, &completedAt,
			&j.SpansConfirmed, &j.SpansRejected, &j.SpansEdited, &j.SpansAdded, &j.SpansPending,
			&completionPct,
		); err != nil {
			return nil, 0, fmt.Errorf("scan job: %w", err)
		}

		if pageRange.Valid {
			j.PageRange = &pageRange.String
		}
		if l1Tag.Valid {
			j.L1Tag = &l1Tag.String
		}
		if completedAt.Valid {
			j.CompletedAt = &completedAt.Time
		}
		if completionPct.Valid {
			j.CompletionPct = &completionPct.Float64
		}

		jobs = append(jobs, j)
	}

	return jobs, total, nil
}

// GetJob returns a single extraction job by ID.
func (s *Pipeline1Store) GetJob(ctx context.Context, jobID uuid.UUID) (*pipeline1.ExtractionJob, error) {
	query := `
		SELECT
			job_id, source_pdf, page_range, pipeline_version, l1_tag,
			total_merged_spans, total_sections, total_pages,
			alignment_confidence, l1_oracle_stats, pdf_page_offset,
			status, created_at, updated_at, completed_at,
			spans_confirmed, spans_rejected, spans_edited, spans_added, spans_pending
		FROM l2_extraction_jobs
		WHERE job_id = $1
	`

	j := &pipeline1.ExtractionJob{}
	var pageRange, l1Tag sql.NullString
	var alignConf sql.NullFloat64
	var oracleStatsJSON []byte
	var completedAt sql.NullTime
	var pdfPageOffset sql.NullInt32

	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&j.JobID, &j.SourcePDF, &pageRange, &j.PipelineVersion, &l1Tag,
		&j.TotalMergedSpans, &j.TotalSections, &j.TotalPages,
		&alignConf, &oracleStatsJSON, &pdfPageOffset,
		&j.Status, &j.CreatedAt, &j.UpdatedAt, &completedAt,
		&j.SpansConfirmed, &j.SpansRejected, &j.SpansEdited, &j.SpansAdded, &j.SpansPending,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}

	if pageRange.Valid {
		j.PageRange = &pageRange.String
	}
	if l1Tag.Valid {
		j.L1Tag = &l1Tag.String
	}
	if alignConf.Valid {
		j.AlignmentConfidence = &alignConf.Float64
	}
	if pdfPageOffset.Valid {
		j.PdfPageOffset = int(pdfPageOffset.Int32)
	}
	if len(oracleStatsJSON) > 0 {
		json.Unmarshal(oracleStatsJSON, &j.L1OracleStats)
	}
	if completedAt.Valid {
		j.CompletedAt = &completedAt.Time
	}

	// Compute completion pct
	denom := j.TotalMergedSpans + j.SpansAdded
	if denom > 0 {
		pct := float64(j.SpansConfirmed+j.SpansRejected+j.SpansEdited) / float64(denom) * 100
		j.CompletionPct = &pct
	}

	return j, nil
}

// =============================================================================
// SPAN OPERATIONS
// =============================================================================

// GetSpans returns filtered, paginated spans for a job.
func (s *Pipeline1Store) GetSpans(ctx context.Context, jobID uuid.UUID, filters pipeline1.SpanFilters, page, pageSize int) ([]*pipeline1.MergedSpan, int, error) {
	// Build WHERE
	where := []string{"job_id = $1"}
	args := []interface{}{jobID}
	idx := 2

	if filters.Status != nil {
		where = append(where, fmt.Sprintf("review_status = $%d", idx))
		args = append(args, string(*filters.Status))
		idx++
	}
	if filters.SectionID != nil {
		where = append(where, fmt.Sprintf("section_id = $%d", idx))
		args = append(args, *filters.SectionID)
		idx++
	}
	if filters.PageNumber != nil {
		where = append(where, fmt.Sprintf("page_number = $%d", idx))
		args = append(args, *filters.PageNumber)
		idx++
	}
	if filters.MinConfidence != nil {
		where = append(where, fmt.Sprintf("merged_confidence >= $%d", idx))
		args = append(args, *filters.MinConfidence)
		idx++
	}
	if filters.MaxConfidence != nil {
		where = append(where, fmt.Sprintf("merged_confidence <= $%d", idx))
		args = append(args, *filters.MaxConfidence)
		idx++
	}
	if filters.HasDisagreement != nil {
		where = append(where, fmt.Sprintf("has_disagreement = $%d", idx))
		args = append(args, *filters.HasDisagreement)
		idx++
	}
	if filters.Search != nil && *filters.Search != "" {
		where = append(where, fmt.Sprintf("to_tsvector('english', text) @@ plainto_tsquery('english', $%d)", idx))
		args = append(args, *filters.Search)
		idx++
	}
	if filters.Tier != nil {
		where = append(where, fmt.Sprintf("tier = $%d", idx))
		args = append(args, *filters.Tier)
		idx++
	}

	whereStr := strings.Join(where, " AND ")

	// Count
	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM l2_merged_spans WHERE %s", whereStr)
	if err := s.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count spans: %w", err)
	}

	// Query
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	dataQ := fmt.Sprintf(`
		SELECT
			id, job_id, text, start_offset, end_offset,
			contributing_channels, channel_confidences, merged_confidence,
			has_disagreement, disagreement_detail,
			page_number, section_id, table_id,
			bbox, surrounding_context,
			tier, coverage_guard_alert, semantic_tokens,
			review_status, reviewer_text, reviewed_by, reviewed_at,
			created_at
		FROM l2_merged_spans
		WHERE %s
		ORDER BY start_offset ASC
		LIMIT $%d OFFSET $%d
	`, whereStr, idx, idx+1)

	rows, err := s.db.QueryContext(ctx, dataQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query spans: %w", err)
	}
	defer rows.Close()

	var spans []*pipeline1.MergedSpan
	for rows.Next() {
		sp := &pipeline1.MergedSpan{}
		var channels pq.StringArray
		var chanConfJSON, bboxJSON, cgAlertJSON, semTokensJSON []byte
		var disagDetail, sectionID, tableID, surroundingCtx, reviewerText, reviewedBy sql.NullString
		var pageNum, tier sql.NullInt32
		var reviewedAt sql.NullTime

		if err := rows.Scan(
			&sp.ID, &sp.JobID, &sp.Text, &sp.StartOffset, &sp.EndOffset,
			&channels, &chanConfJSON, &sp.MergedConfidence,
			&sp.HasDisagreement, &disagDetail,
			&pageNum, &sectionID, &tableID,
			&bboxJSON, &surroundingCtx,
			&tier, &cgAlertJSON, &semTokensJSON,
			&sp.ReviewStatus, &reviewerText, &reviewedBy, &reviewedAt,
			&sp.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan span: %w", err)
		}

		sp.ContributingChannels = []string(channels)
		if len(chanConfJSON) > 0 {
			json.Unmarshal(chanConfJSON, &sp.ChannelConfidences)
		}
		if disagDetail.Valid {
			sp.DisagreementDetail = &disagDetail.String
		}
		if pageNum.Valid {
			v := int(pageNum.Int32)
			sp.PageNumber = &v
		}
		if sectionID.Valid {
			sp.SectionID = &sectionID.String
		}
		if tableID.Valid {
			sp.TableID = &tableID.String
		}
		if len(bboxJSON) > 0 {
			json.Unmarshal(bboxJSON, &sp.Bbox)
		}
		if surroundingCtx.Valid {
			sp.SurroundingContext = &surroundingCtx.String
		}
		if tier.Valid {
			v := int(tier.Int32)
			sp.Tier = &v
		}
		if len(cgAlertJSON) > 0 {
			json.Unmarshal(cgAlertJSON, &sp.CoverageGuardAlert)
		}
		if len(semTokensJSON) > 0 {
			json.Unmarshal(semTokensJSON, &sp.SemanticTokens)
		}
		if reviewerText.Valid {
			sp.ReviewerText = &reviewerText.String
		}
		if reviewedBy.Valid {
			sp.ReviewedBy = &reviewedBy.String
		}
		if reviewedAt.Valid {
			sp.ReviewedAt = &reviewedAt.Time
		}

		spans = append(spans, sp)
	}

	return spans, total, nil
}

// GetSpan returns a single span by ID.
func (s *Pipeline1Store) GetSpan(ctx context.Context, spanID uuid.UUID) (*pipeline1.MergedSpan, error) {
	query := `
		SELECT
			id, job_id, text, start_offset, end_offset,
			contributing_channels, channel_confidences, merged_confidence,
			has_disagreement, disagreement_detail,
			page_number, section_id, table_id,
			bbox, surrounding_context,
			tier, coverage_guard_alert, semantic_tokens,
			review_status, reviewer_text, reviewed_by, reviewed_at,
			created_at
		FROM l2_merged_spans
		WHERE id = $1
	`

	sp := &pipeline1.MergedSpan{}
	var channels pq.StringArray
	var chanConfJSON, bboxJSON, cgAlertJSON, semTokensJSON []byte
	var disagDetail, sectionID, tableID, surroundingCtx, reviewerText, reviewedBy sql.NullString
	var pageNum, tier sql.NullInt32
	var reviewedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, spanID).Scan(
		&sp.ID, &sp.JobID, &sp.Text, &sp.StartOffset, &sp.EndOffset,
		&channels, &chanConfJSON, &sp.MergedConfidence,
		&sp.HasDisagreement, &disagDetail,
		&pageNum, &sectionID, &tableID,
		&bboxJSON, &surroundingCtx,
		&tier, &cgAlertJSON, &semTokensJSON,
		&sp.ReviewStatus, &reviewerText, &reviewedBy, &reviewedAt,
		&sp.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("span not found: %s", spanID)
	}
	if err != nil {
		return nil, fmt.Errorf("get span: %w", err)
	}

	sp.ContributingChannels = []string(channels)
	if len(chanConfJSON) > 0 {
		json.Unmarshal(chanConfJSON, &sp.ChannelConfidences)
	}
	if disagDetail.Valid {
		sp.DisagreementDetail = &disagDetail.String
	}
	if pageNum.Valid {
		v := int(pageNum.Int32)
		sp.PageNumber = &v
	}
	if sectionID.Valid {
		sp.SectionID = &sectionID.String
	}
	if tableID.Valid {
		sp.TableID = &tableID.String
	}
	if len(bboxJSON) > 0 {
		json.Unmarshal(bboxJSON, &sp.Bbox)
	}
	if surroundingCtx.Valid {
		sp.SurroundingContext = &surroundingCtx.String
	}
	if tier.Valid {
		v := int(tier.Int32)
		sp.Tier = &v
	}
	if len(cgAlertJSON) > 0 {
		json.Unmarshal(cgAlertJSON, &sp.CoverageGuardAlert)
	}
	if len(semTokensJSON) > 0 {
		json.Unmarshal(semTokensJSON, &sp.SemanticTokens)
	}
	if reviewerText.Valid {
		sp.ReviewerText = &reviewerText.String
	}
	if reviewedBy.Valid {
		sp.ReviewedBy = &reviewedBy.String
	}
	if reviewedAt.Valid {
		sp.ReviewedAt = &reviewedAt.Time
	}

	return sp, nil
}

// =============================================================================
// REVIEW OPERATIONS (TRANSACTIONAL)
// =============================================================================

// UpdateSpanStatus performs a transactional span status update:
//  1. Update the span row
//  2. Decrement old counter, increment new counter on the job row
//  3. Insert an immutable decision record
//  4. Auto-complete job if spans_pending reaches 0
func (s *Pipeline1Store) UpdateSpanStatus(
	ctx context.Context,
	spanID uuid.UUID,
	action pipeline1.ReviewAction,
	reviewerText *string,
	reviewerID string,
	note *string,
	rejectReason *string,
) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// 1. Get current span state
	var jobID uuid.UUID
	var oldStatus pipeline1.SpanReviewStatus
	var originalText string
	err = tx.QueryRowContext(ctx,
		"SELECT job_id, review_status, text FROM l2_merged_spans WHERE id = $1 FOR UPDATE",
		spanID,
	).Scan(&jobID, &oldStatus, &originalText)
	if err != nil {
		return fmt.Errorf("lock span: %w", err)
	}

	// Determine new status
	var newStatus pipeline1.SpanReviewStatus
	switch action {
	case pipeline1.ActionConfirm:
		newStatus = pipeline1.SpanStatusConfirmed
	case pipeline1.ActionReject:
		newStatus = pipeline1.SpanStatusRejected
	case pipeline1.ActionEdit:
		newStatus = pipeline1.SpanStatusEdited
	default:
		return fmt.Errorf("invalid action for update: %s", action)
	}

	// 2. Update the span
	_, err = tx.ExecContext(ctx,
		`UPDATE l2_merged_spans
		 SET review_status = $2, reviewer_text = $3, reviewed_by = $4, reviewed_at = $5
		 WHERE id = $1`,
		spanID, string(newStatus), reviewerText, reviewerID, now,
	)
	if err != nil {
		return fmt.Errorf("update span: %w", err)
	}

	// 3. Adjust counters on job
	oldCol := counterColumn(oldStatus)
	newCol := counterColumn(newStatus)

	if oldCol != "" {
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf("UPDATE l2_extraction_jobs SET %s = %s - 1 WHERE job_id = $1", oldCol, oldCol),
			jobID,
		)
		if err != nil {
			return fmt.Errorf("decrement %s: %w", oldCol, err)
		}
	}
	if newCol != "" {
		_, err = tx.ExecContext(ctx,
			fmt.Sprintf("UPDATE l2_extraction_jobs SET %s = %s + 1 WHERE job_id = $1", newCol, newCol),
			jobID,
		)
		if err != nil {
			return fmt.Errorf("increment %s: %w", newCol, err)
		}
	}

	// Transition job to IN_PROGRESS if it was PENDING_REVIEW
	_, _ = tx.ExecContext(ctx,
		"UPDATE l2_extraction_jobs SET status = 'IN_PROGRESS' WHERE job_id = $1 AND status = 'PENDING_REVIEW'",
		jobID,
	)

	// 4. Insert immutable decision (with optional reject_reason for REJECT actions)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO l2_reviewer_decisions (merged_span_id, job_id, action, original_text, edited_text, reviewer_id, decided_at, note, reject_reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		spanID, jobID, string(action), originalText, reviewerText, reviewerID, now, note, rejectReason,
	)
	if err != nil {
		return fmt.Errorf("insert decision: %w", err)
	}

	// 5. Auto-complete if no pending spans remain
	var pending int
	err = tx.QueryRowContext(ctx,
		"SELECT spans_pending FROM l2_extraction_jobs WHERE job_id = $1",
		jobID,
	).Scan(&pending)
	if err == nil && pending <= 0 {
		_, _ = tx.ExecContext(ctx,
			"UPDATE l2_extraction_jobs SET status = 'COMPLETED', completed_at = $2 WHERE job_id = $1 AND status != 'COMPLETED'",
			jobID, now,
		)
	}

	return tx.Commit()
}

// CompleteJob marks a job as COMPLETED by the reviewer (Sprint 2 — Phase 5 Sign-Off).
// Only transitions from IN_PROGRESS; rejects if already COMPLETED or ARCHIVED.
func (s *Pipeline1Store) CompleteJob(ctx context.Context, jobID uuid.UUID, reviewerID string, note *string) error {
	now := time.Now()

	result, err := s.db.ExecContext(ctx,
		`UPDATE l2_extraction_jobs
		 SET status = 'COMPLETED', completed_at = $2, completed_by = $3
		 WHERE job_id = $1 AND status IN ('PENDING_REVIEW', 'IN_PROGRESS')`,
		jobID, now, reviewerID,
	)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job %s is not in a completable state (already COMPLETED or ARCHIVED)", jobID)
	}

	return nil
}

// InsertSpan adds a reviewer-created span (ADD action).
func (s *Pipeline1Store) InsertSpan(ctx context.Context, span *pipeline1.MergedSpan, reviewerID string, note *string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()

	// Insert the span
	_, err = tx.ExecContext(ctx,
		`INSERT INTO l2_merged_spans (
			id, job_id, text, start_offset, end_offset,
			contributing_channels, merged_confidence,
			page_number, section_id,
			review_status, reviewed_by, reviewed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'ADDED', $10, $11, $11)`,
		span.ID, span.JobID, span.Text, span.StartOffset, span.EndOffset,
		pq.Array([]string{"REVIEWER"}), 1.0,
		span.PageNumber, span.SectionID,
		reviewerID, now,
	)
	if err != nil {
		return fmt.Errorf("insert span: %w", err)
	}

	// Increment spans_added on job
	_, err = tx.ExecContext(ctx,
		"UPDATE l2_extraction_jobs SET spans_added = spans_added + 1 WHERE job_id = $1",
		span.JobID,
	)
	if err != nil {
		return fmt.Errorf("increment spans_added: %w", err)
	}

	// Insert decision
	_, err = tx.ExecContext(ctx,
		`INSERT INTO l2_reviewer_decisions (merged_span_id, job_id, action, original_text, edited_text, reviewer_id, decided_at, note)
		 VALUES ($1, $2, 'ADD', NULL, $3, $4, $5, $6)`,
		span.ID, span.JobID, span.Text, reviewerID, now, note,
	)
	if err != nil {
		return fmt.Errorf("insert add decision: %w", err)
	}

	return tx.Commit()
}

// =============================================================================
// CONTEXT OPERATIONS (Passages, Tree, Text)
// =============================================================================

// GetPassages returns all section passages for a job.
func (s *Pipeline1Store) GetPassages(ctx context.Context, jobID uuid.UUID) ([]*pipeline1.SectionPassage, error) {
	query := `
		SELECT section_id, heading, page_number, prose_text, span_ids, span_count,
		       child_section_ids, start_offset, end_offset
		FROM l2_section_passages
		WHERE job_id = $1
		ORDER BY start_offset ASC NULLS LAST
	`

	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("query passages: %w", err)
	}
	defer rows.Close()

	var passages []*pipeline1.SectionPassage
	for rows.Next() {
		p := &pipeline1.SectionPassage{JobID: jobID}
		var pageNum sql.NullInt32
		var proseText sql.NullString
		var spanIDs pq.StringArray
		var childIDs pq.StringArray
		var startOff, endOff sql.NullInt32

		if err := rows.Scan(
			&p.SectionID, &p.Heading, &pageNum, &proseText, &spanIDs, &p.SpanCount,
			&childIDs, &startOff, &endOff,
		); err != nil {
			return nil, fmt.Errorf("scan passage: %w", err)
		}

		if pageNum.Valid {
			v := int(pageNum.Int32)
			p.PageNumber = &v
		}
		if proseText.Valid {
			p.ProseText = &proseText.String
		}
		p.SpanIDs = []string(spanIDs)
		p.ChildSectionIDs = []string(childIDs)
		if startOff.Valid {
			v := int(startOff.Int32)
			p.StartOffset = &v
		}
		if endOff.Valid {
			v := int(endOff.Int32)
			p.EndOffset = &v
		}

		passages = append(passages, p)
	}

	return passages, nil
}

// GetTree returns the guideline tree for a job.
func (s *Pipeline1Store) GetTree(ctx context.Context, jobID uuid.UUID) (*pipeline1.GuidelineTree, error) {
	query := "SELECT tree_json, normalized_text FROM l2_guideline_tree WHERE job_id = $1"

	t := &pipeline1.GuidelineTree{JobID: jobID}
	var treeJSON []byte
	var normText sql.NullString

	err := s.db.QueryRowContext(ctx, query, jobID).Scan(&treeJSON, &normText)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tree not found for job: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}

	if len(treeJSON) > 0 {
		json.Unmarshal(treeJSON, &t.TreeJSON)
	}
	if normText.Valid {
		t.NormalizedText = &normText.String
	}

	return t, nil
}

// GetNormalizedText returns only the normalized text for span offset rendering.
func (s *Pipeline1Store) GetNormalizedText(ctx context.Context, jobID uuid.UUID) (string, error) {
	var text sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT normalized_text FROM l2_guideline_tree WHERE job_id = $1", jobID,
	).Scan(&text)
	if err != nil {
		return "", fmt.Errorf("get normalized text: %w", err)
	}
	if text.Valid {
		return text.String, nil
	}
	return "", nil
}

// GetJobMetrics returns computed review metrics for a job.
func (s *Pipeline1Store) GetJobMetrics(ctx context.Context, jobID uuid.UUID) (*pipeline1.JobMetrics, error) {
	query := `
		SELECT
			total_merged_spans + spans_added,
			spans_pending, spans_confirmed, spans_rejected, spans_edited, spans_added,
			CASE
				WHEN total_merged_spans + spans_added = 0 THEN 0
				ELSE ROUND(((spans_confirmed + spans_rejected + spans_edited)::numeric
				     / (total_merged_spans + spans_added)::numeric) * 100, 1)
			END
		FROM l2_extraction_jobs
		WHERE job_id = $1
	`

	m := &pipeline1.JobMetrics{}
	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&m.TotalSpans, &m.Pending, &m.Confirmed, &m.Rejected, &m.Edited, &m.Added,
		&m.CompletionPct,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("get job metrics: %w", err)
	}

	return m, nil
}

// =============================================================================
// PAGE OPERATIONS
// =============================================================================

// GetPages returns per-page summaries for the PageNavigator.
// Aggregates span data, section IDs, risk classification, and current decision.
func (s *Pipeline1Store) GetPages(ctx context.Context, jobID uuid.UUID) ([]*pipeline1.PageInfo, error) {
	query := `
		WITH page_spans AS (
			SELECT
				page_number,
				COUNT(*) AS span_count,
				COUNT(*) FILTER (WHERE review_status = 'PENDING') AS pending_spans,
				COUNT(*) FILTER (WHERE review_status != 'PENDING') AS reviewed_spans,
				BOOL_OR('L1_RECOVERY' = ANY(contributing_channels)) AS has_oracle,
				BOOL_OR(has_disagreement) AS has_disagreement,
				COUNT(*) FILTER (WHERE tier = 1) AS tier1_total,
				COUNT(*) FILTER (WHERE tier = 1 AND review_status != 'PENDING') AS tier1_reviewed,
				COUNT(*) FILTER (WHERE tier = 2) AS tier2_total,
				COUNT(*) FILTER (WHERE tier = 2 AND review_status != 'PENDING') AS tier2_reviewed,
				COUNT(*) FILTER (WHERE tier = 3) AS tier3_total,
				COUNT(*) FILTER (WHERE tier = 3 AND review_status != 'PENDING') AS tier3_reviewed
			FROM l2_merged_spans
			WHERE job_id = $1 AND page_number IS NOT NULL
			GROUP BY page_number
		),
		page_sections AS (
			SELECT
				page_number,
				ARRAY_AGG(DISTINCT section_id ORDER BY section_id) AS section_ids
			FROM l2_section_passages
			WHERE job_id = $1 AND page_number IS NOT NULL
			GROUP BY page_number
		)
		SELECT
			ps.page_number,
			COALESCE(sec.section_ids, '{}'),
			ps.span_count,
			ps.pending_spans,
			ps.reviewed_spans,
			ps.has_oracle,
			ps.has_disagreement,
			pd.action,
			ps.tier1_total, ps.tier1_reviewed,
			ps.tier2_total, ps.tier2_reviewed,
			ps.tier3_total, ps.tier3_reviewed
		FROM page_spans ps
		LEFT JOIN page_sections sec ON sec.page_number = ps.page_number
		LEFT JOIN l2_page_decisions pd ON pd.job_id = $1 AND pd.page_number = ps.page_number
		ORDER BY ps.page_number ASC
	`

	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("query pages: %w", err)
	}
	defer rows.Close()

	var pages []*pipeline1.PageInfo
	for rows.Next() {
		p := &pipeline1.PageInfo{}
		var sectionIDs pq.StringArray
		var hasOracle, hasDisagreement bool
		var decision sql.NullString

		if err := rows.Scan(
			&p.PageNumber,
			&sectionIDs,
			&p.SpanCount,
			&p.PendingSpans,
			&p.ReviewedSpans,
			&hasOracle,
			&hasDisagreement,
			&decision,
			&p.Tier1Total, &p.Tier1Reviewed,
			&p.Tier2Total, &p.Tier2Reviewed,
			&p.Tier3Total, &p.Tier3Reviewed,
		); err != nil {
			return nil, fmt.Errorf("scan page: %w", err)
		}

		p.SectionIDs = []string(sectionIDs)

		// Risk priority: oracle > disagreement > clean
		switch {
		case hasOracle:
			p.Risk = pipeline1.PageRiskOracle
		case hasDisagreement:
			p.Risk = pipeline1.PageRiskDisagreement
		default:
			p.Risk = pipeline1.PageRiskClean
		}

		if decision.Valid {
			a := pipeline1.PageDecisionAction(decision.String)
			p.Decision = &a
		}

		pages = append(pages, p)
	}

	return pages, nil
}

// DecidePage upserts a page-level decision (ACCEPT/FLAG/ESCALATE).
func (s *Pipeline1Store) DecidePage(
	ctx context.Context,
	jobID uuid.UUID,
	pageNumber int,
	action pipeline1.PageDecisionAction,
	reviewerID string,
	note *string,
) error {
	query := `
		INSERT INTO l2_page_decisions (job_id, page_number, action, reviewer_id, note)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (job_id, page_number)
		DO UPDATE SET action = EXCLUDED.action, reviewer_id = EXCLUDED.reviewer_id,
		              note = EXCLUDED.note, decided_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query, jobID, pageNumber, string(action), reviewerID, note)
	if err != nil {
		return fmt.Errorf("decide page: %w", err)
	}
	return nil
}

// GetPageStats returns aggregated page decision counts for a job.
func (s *Pipeline1Store) GetPageStats(ctx context.Context, jobID uuid.UUID) (*pipeline1.PageStats, error) {
	// Get total distinct pages from spans
	var totalPages int
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(DISTINCT page_number) FROM l2_merged_spans WHERE job_id = $1 AND page_number IS NOT NULL",
		jobID,
	).Scan(&totalPages)
	if err != nil {
		return nil, fmt.Errorf("count pages: %w", err)
	}

	// Count decisions by action type
	query := `
		SELECT
			COUNT(*) FILTER (WHERE action = 'ACCEPT') AS accepted,
			COUNT(*) FILTER (WHERE action = 'FLAG') AS flagged,
			COUNT(*) FILTER (WHERE action = 'ESCALATE') AS escalated
		FROM l2_page_decisions
		WHERE job_id = $1
	`

	stats := &pipeline1.PageStats{TotalPages: totalPages}
	err = s.db.QueryRowContext(ctx, query, jobID).Scan(
		&stats.PagesAccepted,
		&stats.PagesFlagged,
		&stats.PagesEscalated,
	)
	if err != nil {
		return nil, fmt.Errorf("count page decisions: %w", err)
	}

	stats.PagesNoDecision = totalPages - stats.PagesAccepted - stats.PagesFlagged - stats.PagesEscalated

	// Global tier review stats across all pages
	tierQuery := `
		SELECT
			COUNT(*) FILTER (WHERE tier = 1) AS t1_total,
			COUNT(*) FILTER (WHERE tier = 1 AND review_status != 'PENDING') AS t1_reviewed,
			COUNT(*) FILTER (WHERE tier = 2) AS t2_total,
			COUNT(*) FILTER (WHERE tier = 2 AND review_status != 'PENDING') AS t2_reviewed,
			COUNT(*) FILTER (WHERE tier = 3) AS t3_total,
			COUNT(*) FILTER (WHERE tier = 3 AND review_status != 'PENDING') AS t3_reviewed
		FROM l2_merged_spans
		WHERE job_id = $1 AND page_number IS NOT NULL
	`
	ts := &pipeline1.TierReviewStats{}
	tierErr := s.db.QueryRowContext(ctx, tierQuery, jobID).Scan(
		&ts.Tier1Total, &ts.Tier1Reviewed,
		&ts.Tier2Total, &ts.Tier2Reviewed,
		&ts.Tier3Total, &ts.Tier3Reviewed,
	)
	if tierErr == nil && (ts.Tier1Total > 0 || ts.Tier2Total > 0 || ts.Tier3Total > 0) {
		if ts.Tier2Total > 0 {
			ts.Tier2Pct = float64(ts.Tier2Reviewed) / float64(ts.Tier2Total) * 100
		}
		stats.TierStats = ts
	}

	return stats, nil
}

// CountTier1PendingOnPage returns the number of Tier 1 (patient safety) spans
// on a specific page that are still PENDING review. Used to guard page ACCEPT.
func (s *Pipeline1Store) CountTier1PendingOnPage(ctx context.Context, jobID uuid.UUID, pageNumber int) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM l2_merged_spans
		 WHERE job_id = $1 AND page_number = $2 AND tier = 1 AND review_status = 'PENDING'`,
		jobID, pageNumber,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count tier1 pending on page %d: %w", pageNumber, err)
	}
	return count, nil
}

// =============================================================================
// REVIEW TASK QUEUE (Computed — Task-Driven Adjudication)
// =============================================================================

// GetReviewTasks computes the reviewer task queue from pipeline output.
// Tasks are NOT stored in a table — they are derived live from span data and passage data.
// Three categories: L1 recoveries (critical), disagreements (warning), passage spot-checks (info).
func (s *Pipeline1Store) GetReviewTasks(ctx context.Context, jobID uuid.UUID) ([]*pipeline1.ReviewTask, error) {
	var tasks []*pipeline1.ReviewTask

	// ---- 1. L1 Recovery tasks: spans with L1 channel ----
	l1Query := `
		SELECT id, text, page_number, section_id, review_status
		FROM l2_merged_spans
		WHERE job_id = $1 AND 'L1_RECOVERY' = ANY(contributing_channels)
		ORDER BY start_offset ASC
	`
	l1Rows, err := s.db.QueryContext(ctx, l1Query, jobID)
	if err != nil {
		return nil, fmt.Errorf("query L1 recovery tasks: %w", err)
	}
	defer l1Rows.Close()

	for l1Rows.Next() {
		var spanID uuid.UUID
		var text, reviewStatus string
		var pageNum sql.NullInt32
		var sectionID sql.NullString

		if err := l1Rows.Scan(&spanID, &text, &pageNum, &sectionID, &reviewStatus); err != nil {
			return nil, fmt.Errorf("scan L1 task: %w", err)
		}

		sid := spanID.String()
		status := "PENDING"
		if reviewStatus != "PENDING" {
			status = "RESOLVED"
		}

		task := &pipeline1.ReviewTask{
			ID:          fmt.Sprintf("l1-%s", sid[:12]),
			TaskType:    pipeline1.TaskL1Recovery,
			Severity:    pipeline1.SeverityCritical,
			Title:       truncateText(text, 80),
			Description: "L1 Oracle recovery — verify text against source PDF",
			SpanID:      &sid,
			Status:      status,
		}
		if pageNum.Valid {
			v := int(pageNum.Int32)
			task.PageNumber = &v
		}
		if sectionID.Valid {
			task.SectionID = &sectionID.String
		}
		tasks = append(tasks, task)
	}

	// ---- 2. Disagreement tasks: only low-corroboration disagreements ----
	// Focus on spans with ≤2 contributing channels (weakly corroborated) to keep
	// the task queue actionable (~15-20 items, not hundreds).
	disagQuery := `
		SELECT id, text, page_number, section_id, review_status, disagreement_detail
		FROM l2_merged_spans
		WHERE job_id = $1 AND has_disagreement = true
		  AND 'L1_RECOVERY' != ALL(contributing_channels)
		  AND array_length(contributing_channels, 1) <= 2
		ORDER BY array_length(contributing_channels, 1) ASC, merged_confidence ASC, start_offset ASC
		LIMIT 20
	`
	disagRows, err := s.db.QueryContext(ctx, disagQuery, jobID)
	if err != nil {
		return nil, fmt.Errorf("query disagreement tasks: %w", err)
	}
	defer disagRows.Close()

	for disagRows.Next() {
		var spanID uuid.UUID
		var text, reviewStatus string
		var pageNum sql.NullInt32
		var sectionID, disagDetail sql.NullString

		if err := disagRows.Scan(&spanID, &text, &pageNum, &sectionID, &reviewStatus, &disagDetail); err != nil {
			return nil, fmt.Errorf("scan disagreement task: %w", err)
		}

		sid := spanID.String()
		status := "PENDING"
		if reviewStatus != "PENDING" {
			status = "RESOLVED"
		}

		desc := "Channel disagreement detected"
		if disagDetail.Valid {
			desc = truncateText(disagDetail.String, 120)
		}

		task := &pipeline1.ReviewTask{
			ID:          fmt.Sprintf("dis-%s", sid[:12]),
			TaskType:    pipeline1.TaskDisagreement,
			Severity:    pipeline1.SeverityWarning,
			Title:       truncateText(text, 80),
			Description: desc,
			SpanID:      &sid,
			Status:      status,
		}
		if pageNum.Valid {
			v := int(pageNum.Int32)
			task.PageNumber = &v
		}
		if sectionID.Valid {
			task.SectionID = &sectionID.String
		}
		tasks = append(tasks, task)
	}

	// ---- 3. Passage spot-check tasks (deterministic selection) ----
	// Include span_ids[1] (first span UUID) so the frontend can link to a real span.
	// JOIN to l2_merged_spans to derive task status from the first span's review_status.
	spotCheckQuery := `
		WITH ranked AS (
			SELECT section_id, heading, page_number, span_count, span_ids, child_section_ids,
				ROW_NUMBER() OVER (ORDER BY span_count DESC) AS density_rank,
				CASE WHEN array_length(child_section_ids, 1) > 0 THEN true ELSE false END AS is_reparented,
				CASE WHEN section_id ~ '^(1\.[3-5]|4\.[1-2])' THEN true ELSE false END AS is_drug_section
			FROM l2_section_passages
			WHERE job_id = $1
		)
		SELECT r.section_id, r.heading, r.page_number, r.span_count,
			CASE
				WHEN r.density_rank = 1 THEN 'highest_density'
				WHEN r.is_reparented THEN 'reparented'
				WHEN r.is_drug_section THEN 'drug_therapy'
			END AS reason,
			CASE WHEN array_length(r.span_ids, 1) > 0 THEN r.span_ids[1]::text ELSE NULL END AS first_span_id,
			COALESCE(ms.review_status, 'PENDING') AS span_review_status
		FROM ranked r
		LEFT JOIN l2_merged_spans ms ON ms.id = (
			CASE WHEN array_length(r.span_ids, 1) > 0 THEN r.span_ids[1]::uuid ELSE NULL END
		)
		WHERE r.density_rank = 1 OR r.is_reparented OR r.is_drug_section
		ORDER BY r.density_rank ASC
		LIMIT 5
	`
	spotRows, err := s.db.QueryContext(ctx, spotCheckQuery, jobID)
	if err != nil {
		return nil, fmt.Errorf("query spot-check tasks: %w", err)
	}
	defer spotRows.Close()

	for spotRows.Next() {
		var sectionID, heading string
		var pageNum sql.NullInt32
		var spanCount int
		var reason sql.NullString
		var firstSpanID sql.NullString
		var spanReviewStatus string

		if err := spotRows.Scan(&sectionID, &heading, &pageNum, &spanCount, &reason, &firstSpanID, &spanReviewStatus); err != nil {
			return nil, fmt.Errorf("scan spot-check task: %w", err)
		}

		reasonLabel := "spot-check"
		if reason.Valid {
			reasonLabel = reason.String
		}

		// Derive task status from the first span's review status (same pattern as L1/disagreement)
		status := "PENDING"
		if spanReviewStatus != "PENDING" {
			status = "RESOLVED"
		}

		sid := sectionID
		task := &pipeline1.ReviewTask{
			ID:          fmt.Sprintf("spot-%s", sectionID),
			TaskType:    pipeline1.TaskPassageSpotCheck,
			Severity:    pipeline1.SeverityInfo,
			Title:       truncateText(heading, 80),
			Description: fmt.Sprintf("Passage spot-check (%s) — %d spans", reasonLabel, spanCount),
			PassageID:   &sid,
			Status:      status,
		}
		if pageNum.Valid {
			v := int(pageNum.Int32)
			task.PageNumber = &v
		}
		if firstSpanID.Valid {
			task.SpanID = &firstSpanID.String
		}
		task.SectionID = &sid
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetPatchedPassages returns passages with reviewer edits applied.
// Reads from the l2_passages_for_l3 view.
func (s *Pipeline1Store) GetPatchedPassages(ctx context.Context, jobID uuid.UUID) ([]*pipeline1.PatchedPassage, error) {
	query := `
		SELECT section_id, heading, page_number, child_section_ids,
		       prose_text, span_ids, span_count, patched_at
		FROM l2_passages_for_l3
		WHERE job_id = $1
		ORDER BY section_id ASC
	`

	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("query patched passages: %w", err)
	}
	defer rows.Close()

	var passages []*pipeline1.PatchedPassage
	for rows.Next() {
		p := &pipeline1.PatchedPassage{}
		var pageNum sql.NullInt32
		var childIDs pq.StringArray
		var proseText sql.NullString
		var spanIDs pq.StringArray
		var patchedAt sql.NullTime

		if err := rows.Scan(
			&p.SectionID, &p.Heading, &pageNum, &childIDs,
			&proseText, &spanIDs, &p.SpanCount, &patchedAt,
		); err != nil {
			return nil, fmt.Errorf("scan patched passage: %w", err)
		}

		if pageNum.Valid {
			v := int(pageNum.Int32)
			p.PageNumber = &v
		}
		p.ChildSectionIDs = []string(childIDs)
		if proseText.Valid {
			p.ProseText = &proseText.String
		}
		p.SpanIDs = []string(spanIDs)
		if patchedAt.Valid {
			p.PatchedAt = &patchedAt.Time
		}

		passages = append(passages, p)
	}

	return passages, nil
}

// =============================================================================
// REVALIDATION OPERATIONS (Phase 4)
// =============================================================================

// RunRevalidation performs a CoverageGuard re-validation check.
// It compares current span states against their CoverageGuard alerts,
// computing which alerts are resolved (span was edited/rejected) and
// which persist. Returns PASS if zero unresolved alerts remain.
func (s *Pipeline1Store) RunRevalidation(ctx context.Context, jobID uuid.UUID, reviewerID string) (*pipeline1.RevalidationRun, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Determine iteration number
	var maxIter sql.NullInt32
	err = tx.QueryRowContext(ctx,
		"SELECT MAX(iteration) FROM l2_revalidation_runs WHERE job_id = $1",
		jobID,
	).Scan(&maxIter)
	if err != nil {
		return nil, fmt.Errorf("get max iteration: %w", err)
	}
	iteration := 1
	if maxIter.Valid {
		iteration = int(maxIter.Int32) + 1
	}

	// Get current span counts
	var editedCount, rejectedCount, addedCount int
	err = tx.QueryRowContext(ctx,
		`SELECT spans_edited, spans_rejected, spans_added
		 FROM l2_extraction_jobs WHERE job_id = $1`,
		jobID,
	).Scan(&editedCount, &rejectedCount, &addedCount)
	if err != nil {
		return nil, fmt.Errorf("get job counters: %w", err)
	}

	// Query all spans with CoverageGuard alerts to compute deltas
	alertQuery := `
		SELECT id, review_status, reviewer_text, coverage_guard_alert
		FROM l2_merged_spans
		WHERE job_id = $1 AND coverage_guard_alert IS NOT NULL
		ORDER BY start_offset ASC
	`
	rows, err := tx.QueryContext(ctx, alertQuery, jobID)
	if err != nil {
		return nil, fmt.Errorf("query alerted spans: %w", err)
	}
	defer rows.Close()

	var deltas []pipeline1.CoverageGuardDelta
	unresolvedCount := 0

	for rows.Next() {
		var spanID uuid.UUID
		var reviewStatus string
		var reviewerText sql.NullString
		var alertJSON []byte

		if err := rows.Scan(&spanID, &reviewStatus, &reviewerText, &alertJSON); err != nil {
			return nil, fmt.Errorf("scan alerted span: %w", err)
		}

		var alert any
		if len(alertJSON) > 0 {
			json.Unmarshal(alertJSON, &alert)
		}

		// A span's alert is "resolved" if the reviewer has taken action on it
		// (CONFIRMED with verification, EDITED to fix, or REJECTED to discard)
		resolved := reviewStatus != string(pipeline1.SpanStatusPending)

		// For EDITED spans, also verify the reviewer actually changed the text
		if reviewStatus == string(pipeline1.SpanStatusEdited) && !reviewerText.Valid {
			resolved = false
		}

		if !resolved {
			unresolvedCount++
		}

		delta := pipeline1.CoverageGuardDelta{
			SpanID:        spanID.String(),
			PreviousAlert: alert,
			CurrentAlert:  alert, // same alert, resolution status changes
			Resolved:      resolved,
		}
		// If resolved, clear the "current" alert to show it's handled
		if resolved {
			delta.CurrentAlert = nil
		}

		deltas = append(deltas, delta)
	}

	// Verdict: PASS if all alerts resolved, BLOCK otherwise
	verdict := pipeline1.VerdictPass
	if unresolvedCount > 0 {
		verdict = pipeline1.VerdictBlock
	}

	// Persist the revalidation run
	deltasJSON, err := json.Marshal(deltas)
	if err != nil {
		return nil, fmt.Errorf("marshal deltas: %w", err)
	}

	runID := uuid.New()
	now := time.Now()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO l2_revalidation_runs
		 (id, job_id, iteration, verdict, edited_span_count, rejected_span_count, added_span_count, deltas, triggered_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		runID, jobID, iteration, string(verdict),
		editedCount, rejectedCount, addedCount,
		deltasJSON, reviewerID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert revalidation run: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	run := &pipeline1.RevalidationRun{
		ID:                runID,
		JobID:             jobID,
		Iteration:         iteration,
		Verdict:           verdict,
		EditedSpanCount:   editedCount,
		RejectedSpanCount: rejectedCount,
		AddedSpanCount:    addedCount,
		Deltas:            deltas,
		CreatedAt:         now,
	}
	if reviewerID != "" {
		run.TriggeredBy = &reviewerID
	}

	return run, nil
}

// GetRevalidationHistory returns all revalidation runs for a job, ordered by iteration.
func (s *Pipeline1Store) GetRevalidationHistory(ctx context.Context, jobID uuid.UUID) ([]*pipeline1.RevalidationRun, error) {
	query := `
		SELECT id, job_id, iteration, verdict,
		       edited_span_count, rejected_span_count, added_span_count,
		       deltas, triggered_by, created_at
		FROM l2_revalidation_runs
		WHERE job_id = $1
		ORDER BY iteration ASC
	`

	rows, err := s.db.QueryContext(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("query revalidation history: %w", err)
	}
	defer rows.Close()

	var runs []*pipeline1.RevalidationRun
	for rows.Next() {
		r := &pipeline1.RevalidationRun{}
		var deltasJSON []byte
		var triggeredBy sql.NullString

		if err := rows.Scan(
			&r.ID, &r.JobID, &r.Iteration, &r.Verdict,
			&r.EditedSpanCount, &r.RejectedSpanCount, &r.AddedSpanCount,
			&deltasJSON, &triggeredBy, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan revalidation run: %w", err)
		}

		if len(deltasJSON) > 0 {
			json.Unmarshal(deltasJSON, &r.Deltas)
		}
		if triggeredBy.Valid {
			r.TriggeredBy = &triggeredBy.String
		}

		runs = append(runs, r)
	}

	return runs, nil
}

// =============================================================================
// OUTPUT CONTRACT OPERATIONS (Phase 5)
// =============================================================================

// PreviewOutputContract assembles a read-only preview of the output contract
// from current span data. Does NOT persist — used by the Phase 5 preview panel.
func (s *Pipeline1Store) PreviewOutputContract(ctx context.Context, jobID uuid.UUID) (*pipeline1.OutputContract, error) {
	return s.assembleContract(ctx, s.db, jobID, "")
}

// AssembleOutputContract assembles and persists the output contract for Pipeline 2.
// This is idempotent — re-assembly overwrites the previous contract via upsert.
func (s *Pipeline1Store) AssembleOutputContract(ctx context.Context, jobID uuid.UUID, reviewerID string) (*pipeline1.OutputContract, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	contract, err := s.assembleContract(ctx, tx, jobID, reviewerID)
	if err != nil {
		return nil, err
	}

	// Persist sections as JSONB
	confirmedJSON, _ := json.Marshal(contract.ConfirmedFacts)
	addedJSON, _ := json.Marshal(contract.AddedFacts)
	sectionTreeJSON, _ := json.Marshal(contract.SectionTree)
	envelopeJSON, _ := json.Marshal(contract.EvidenceEnvelope)
	rejectionJSON, _ := json.Marshal(contract.RejectionLog)

	_, err = tx.ExecContext(ctx,
		`INSERT INTO l2_output_contracts
		 (job_id, confirmed_facts, added_facts, section_tree, evidence_envelope, rejection_log, assembled_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (job_id) DO UPDATE SET
		   confirmed_facts = EXCLUDED.confirmed_facts,
		   added_facts = EXCLUDED.added_facts,
		   section_tree = EXCLUDED.section_tree,
		   evidence_envelope = EXCLUDED.evidence_envelope,
		   rejection_log = EXCLUDED.rejection_log,
		   assembled_by = EXCLUDED.assembled_by,
		   assembled_at = NOW()`,
		jobID, confirmedJSON, addedJSON, sectionTreeJSON, envelopeJSON, rejectionJSON, reviewerID,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert output contract: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return contract, nil
}

// queryExecutor abstracts *sql.DB and *sql.Tx for shared assembly logic.
type queryExecutor interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// assembleContract builds the 5-section output contract from current span data.
func (s *Pipeline1Store) assembleContract(ctx context.Context, db queryExecutor, jobID uuid.UUID, reviewerID string) (*pipeline1.OutputContract, error) {
	contract := &pipeline1.OutputContract{}

	// ── Section 1: Confirmed Facts (CONFIRMED + EDITED) ──
	confirmedQuery := `
		SELECT s.id, s.text, s.contributing_channels, s.merged_confidence,
		       s.page_number, s.section_id, s.reviewer_text, s.review_status,
		       d.reviewer_id, d.action, d.decided_at, d.note
		FROM l2_merged_spans s
		LEFT JOIN LATERAL (
			SELECT reviewer_id, action, decided_at, note
			FROM l2_reviewer_decisions
			WHERE merged_span_id = s.id
			ORDER BY decided_at DESC LIMIT 1
		) d ON true
		WHERE s.job_id = $1 AND s.review_status IN ('CONFIRMED', 'EDITED')
		ORDER BY s.start_offset ASC
	`
	cfRows, err := db.QueryContext(ctx, confirmedQuery, jobID)
	if err != nil {
		return nil, fmt.Errorf("query confirmed facts: %w", err)
	}
	defer cfRows.Close()

	for cfRows.Next() {
		var spanID uuid.UUID
		var text string
		var channels pq.StringArray
		var confidence float64
		var pageNum sql.NullInt32
		var sectionID, reviewerText sql.NullString
		var reviewStatus string
		var decReviewerID, decAction, decNote sql.NullString
		var decTimestamp sql.NullTime

		if err := cfRows.Scan(
			&spanID, &text, &channels, &confidence,
			&pageNum, &sectionID, &reviewerText, &reviewStatus,
			&decReviewerID, &decAction, &decTimestamp, &decNote,
		); err != nil {
			return nil, fmt.Errorf("scan confirmed fact: %w", err)
		}

		fact := pipeline1.ConfirmedFact{
			SpanID:           spanID.String(),
			FactText:         text,
			Channels:         []string(channels),
			MergedConfidence: confidence,
			ReviewAction:     reviewStatus,
		}
		if pageNum.Valid {
			v := int(pageNum.Int32)
			fact.PageNumber = &v
		}
		if sectionID.Valid {
			fact.SectionID = &sectionID.String
		}
		if reviewerText.Valid {
			fact.ReviewerText = &reviewerText.String
		}

		// Build audit trail
		audit := map[string]any{}
		if decReviewerID.Valid {
			audit["reviewerId"] = decReviewerID.String
		}
		if decAction.Valid {
			audit["action"] = decAction.String
		}
		if decTimestamp.Valid {
			audit["timestamp"] = decTimestamp.Time.Format(time.RFC3339)
		}
		if decNote.Valid {
			audit["note"] = decNote.String
		}
		fact.AuditTrail = audit

		contract.ConfirmedFacts = append(contract.ConfirmedFacts, fact)
	}

	// ── Section 2: Added Facts ──
	addedQuery := `
		SELECT s.id, s.text, s.page_number, s.section_id,
		       d.reviewer_id, d.decided_at, d.note
		FROM l2_merged_spans s
		LEFT JOIN LATERAL (
			SELECT reviewer_id, decided_at, note
			FROM l2_reviewer_decisions
			WHERE merged_span_id = s.id AND action = 'ADD'
			ORDER BY decided_at DESC LIMIT 1
		) d ON true
		WHERE s.job_id = $1 AND s.review_status = 'ADDED'
		ORDER BY s.created_at ASC
	`
	afRows, err := db.QueryContext(ctx, addedQuery, jobID)
	if err != nil {
		return nil, fmt.Errorf("query added facts: %w", err)
	}
	defer afRows.Close()

	for afRows.Next() {
		var spanID uuid.UUID
		var text string
		var pageNum sql.NullInt32
		var sectionID sql.NullString
		var decReviewerID, decNote sql.NullString
		var decTimestamp sql.NullTime

		if err := afRows.Scan(
			&spanID, &text, &pageNum, &sectionID,
			&decReviewerID, &decTimestamp, &decNote,
		); err != nil {
			return nil, fmt.Errorf("scan added fact: %w", err)
		}

		fact := pipeline1.AddedFact{
			SpanID:     spanID.String(),
			FactText:   text,
			Channel:    "MANUAL",
			Confidence: 1.0,
		}
		if pageNum.Valid {
			v := int(pageNum.Int32)
			fact.PageNumber = &v
		}
		if sectionID.Valid {
			fact.SectionID = &sectionID.String
		}

		audit := map[string]any{}
		if decReviewerID.Valid {
			audit["reviewerId"] = decReviewerID.String
		}
		if decTimestamp.Valid {
			audit["timestamp"] = decTimestamp.Time.Format(time.RFC3339)
		}
		if decNote.Valid {
			audit["note"] = decNote.String
		}
		fact.AuditTrail = audit

		contract.AddedFacts = append(contract.AddedFacts, fact)
	}

	// ── Section 3: Section Tree ──
	tree, err := s.GetTree(ctx, jobID)
	if err == nil {
		contract.SectionTree = tree
	}

	// ── Section 4: Evidence Envelope ──
	job, err := s.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("get job for envelope: %w", err)
	}

	envelope := &pipeline1.EvidenceEnvelope{
		JobID:           jobID.String(),
		SourcePDF:       job.SourcePDF,
		PipelineVersion: job.PipelineVersion,
		TotalSpans:      job.TotalMergedSpans + job.SpansAdded,
		Confirmed:       job.SpansConfirmed,
		Edited:          job.SpansEdited,
		Rejected:        job.SpansRejected,
		Added:           job.SpansAdded,
		ReviewerID:      reviewerID,
		ReviewStartedAt: job.CreatedAt.Format(time.RFC3339),
		ReviewCompletedAt: time.Now().Format(time.RFC3339),
	}
	if job.CompletedAt != nil {
		envelope.ReviewCompletedAt = job.CompletedAt.Format(time.RFC3339)
	}
	contract.EvidenceEnvelope = envelope

	// ── Section 5: Rejection Log ──
	rejectQuery := `
		SELECT s.id, s.text, s.contributing_channels,
		       d.reject_reason, d.reviewer_id, d.decided_at, d.note
		FROM l2_merged_spans s
		LEFT JOIN LATERAL (
			SELECT reject_reason, reviewer_id, decided_at, note
			FROM l2_reviewer_decisions
			WHERE merged_span_id = s.id AND action = 'REJECT'
			ORDER BY decided_at DESC LIMIT 1
		) d ON true
		WHERE s.job_id = $1 AND s.review_status = 'REJECTED'
		ORDER BY s.start_offset ASC
	`
	rjRows, err := db.QueryContext(ctx, rejectQuery, jobID)
	if err != nil {
		return nil, fmt.Errorf("query rejection log: %w", err)
	}
	defer rjRows.Close()

	for rjRows.Next() {
		var spanID uuid.UUID
		var text string
		var channels pq.StringArray
		var rejectReason, decReviewerID, decNote sql.NullString
		var decTimestamp sql.NullTime

		if err := rjRows.Scan(
			&spanID, &text, &channels,
			&rejectReason, &decReviewerID, &decTimestamp, &decNote,
		); err != nil {
			return nil, fmt.Errorf("scan rejection: %w", err)
		}

		entry := pipeline1.RejectionLogEntry{
			SpanID:   spanID.String(),
			Text:     text,
			Channels: []string(channels),
		}
		if rejectReason.Valid {
			entry.RejectReason = &rejectReason.String
		}
		if decReviewerID.Valid {
			entry.ReviewerID = &decReviewerID.String
		}
		if decTimestamp.Valid {
			ts := decTimestamp.Time.Format(time.RFC3339)
			entry.Timestamp = &ts
		}
		if decNote.Valid {
			entry.Note = &decNote.String
		}

		contract.RejectionLog = append(contract.RejectionLog, entry)
	}

	// Ensure nil slices become empty arrays in JSON
	if contract.ConfirmedFacts == nil {
		contract.ConfirmedFacts = []pipeline1.ConfirmedFact{}
	}
	if contract.AddedFacts == nil {
		contract.AddedFacts = []pipeline1.AddedFact{}
	}
	if contract.RejectionLog == nil {
		contract.RejectionLog = []pipeline1.RejectionLogEntry{}
	}

	return contract, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// truncateText truncates a string to max characters with "..." suffix.
func truncateText(text string, max int) string {
	if len(text) <= max {
		return text
	}
	return text[:max-3] + "..."
}

func counterColumn(status pipeline1.SpanReviewStatus) string {
	switch status {
	case pipeline1.SpanStatusPending:
		return "spans_pending"
	case pipeline1.SpanStatusConfirmed:
		return "spans_confirmed"
	case pipeline1.SpanStatusRejected:
		return "spans_rejected"
	case pipeline1.SpanStatusEdited:
		return "spans_edited"
	case pipeline1.SpanStatusAdded:
		return "spans_added"
	default:
		return ""
	}
}

// =============================================================================
// HIGHLIGHT HTML + SOURCE PDF
// =============================================================================

// GetHighlightHTML returns the pipeline-generated highlight HTML for a job.
func (s *Pipeline1Store) GetHighlightHTML(ctx context.Context, jobID uuid.UUID) (string, error) {
	var html sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT highlight_html FROM l2_guideline_tree WHERE job_id = $1", jobID,
	).Scan(&html)
	if err != nil {
		return "", fmt.Errorf("get highlight html: %w", err)
	}
	if html.Valid {
		return html.String, nil
	}
	return "", nil
}

// GetSourcePDFPath returns the filesystem path to the source PDF for a job.
func (s *Pipeline1Store) GetSourcePDFPath(ctx context.Context, jobID uuid.UUID) (string, error) {
	var path sql.NullString
	err := s.db.QueryRowContext(ctx,
		"SELECT source_pdf_path FROM l2_extraction_jobs WHERE job_id = $1", jobID,
	).Scan(&path)
	if err != nil {
		return "", fmt.Errorf("get source pdf path: %w", err)
	}
	if path.Valid {
		return path.String, nil
	}
	return "", nil
}
