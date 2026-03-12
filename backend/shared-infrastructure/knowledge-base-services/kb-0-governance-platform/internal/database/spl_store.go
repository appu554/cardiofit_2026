// Package database provides data access for the KB-0 Governance Platform.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"
)

// =============================================================================
// SPL STORE
// =============================================================================
// Data access layer for SPL FactStore Pipeline data. Queries the
// canonical_facts database for completeness_reports, derived_facts,
// source_sections, and spl_sign_offs tables.
// =============================================================================

// SPLStore handles database operations for SPL review data.
type SPLStore struct {
	db *sql.DB
}

// NewSPLStore creates a new SPL data store.
func NewSPLStore(db *sql.DB) *SPLStore {
	return &SPLStore{db: db}
}

// =============================================================================
// COMPLETENESS REPORTS
// =============================================================================

// CompletenessReport represents a per-drug quality report.
type CompletenessReport struct {
	ID                   string          `json:"id"`
	DrugName             string          `json:"drugName"`
	RxCUI                string          `json:"rxcui"`
	SectionsCovered      []string        `json:"sectionsCovered"`
	SectionsMissing      []string        `json:"sectionsMissing"`
	SectionCoveragePct   float64         `json:"sectionCoveragePct"`
	FactCounts           json.RawMessage `json:"factCounts"`
	TotalFacts           int             `json:"totalFacts"`
	FactTypesCovered     int             `json:"factTypesCovered"`
	MeddraMatchRate      float64         `json:"meddraMatchRate"`
	FrequencyCovRate     float64         `json:"frequencyCovRate"`
	InteractionQual      float64         `json:"interactionQual"`
	TotalSourceRows      int             `json:"totalSourceRows"`
	ExtractedRows        int             `json:"extractedRows"`
	SkippedRows          int             `json:"skippedRows"`
	RowCoveragePct       float64         `json:"rowCoveragePct"`
	SkipReasonBreakdown  json.RawMessage `json:"skipReasonBreakdown"`
	StructuredCount      int             `json:"structuredCount"`
	LLMCount             int             `json:"llmCount"`
	GrammarCount         int             `json:"grammarCount"`
	DeterministicPct     float64         `json:"deterministicPct"`
	Warnings             []string        `json:"warnings"`
	Grade                string          `json:"grade"`
	GateVerdict          string          `json:"gateVerdict"`
	CreatedAt            time.Time       `json:"createdAt"`
}

// GetAllCompleteness returns the latest completeness report for each drug.
func (s *SPLStore) GetAllCompleteness(ctx context.Context) ([]*CompletenessReport, error) {
	query := `
		SELECT DISTINCT ON (drug_name)
			id, drug_name, rxcui,
			sections_covered, sections_missing, section_coverage_pct,
			fact_counts, total_facts, fact_types_covered,
			meddra_match_rate, frequency_cov_rate, interaction_qual,
			total_source_rows, extracted_rows, skipped_rows, row_coverage_pct,
			skip_reason_breakdown,
			structured_count, llm_count, grammar_count, deterministic_pct,
			warnings, grade, gate_verdict, created_at
		FROM completeness_reports
		ORDER BY drug_name, created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get completeness reports: %w", err)
	}
	defer rows.Close()

	var reports []*CompletenessReport
	for rows.Next() {
		r := &CompletenessReport{}
		var sectionsCovered, sectionsMissing, warnings pq.StringArray

		err := rows.Scan(
			&r.ID, &r.DrugName, &r.RxCUI,
			&sectionsCovered, &sectionsMissing, &r.SectionCoveragePct,
			&r.FactCounts, &r.TotalFacts, &r.FactTypesCovered,
			&r.MeddraMatchRate, &r.FrequencyCovRate, &r.InteractionQual,
			&r.TotalSourceRows, &r.ExtractedRows, &r.SkippedRows, &r.RowCoveragePct,
			&r.SkipReasonBreakdown,
			&r.StructuredCount, &r.LLMCount, &r.GrammarCount, &r.DeterministicPct,
			&warnings, &r.Grade, &r.GateVerdict, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan completeness report: %w", err)
		}
		r.SectionsCovered = sectionsCovered
		r.SectionsMissing = sectionsMissing
		r.Warnings = warnings
		reports = append(reports, r)
	}

	return reports, nil
}

// GetCompletenessByDrug returns the latest completeness report for a specific drug.
func (s *SPLStore) GetCompletenessByDrug(ctx context.Context, drugName string) (*CompletenessReport, error) {
	query := `
		SELECT
			id, drug_name, rxcui,
			sections_covered, sections_missing, section_coverage_pct,
			fact_counts, total_facts, fact_types_covered,
			meddra_match_rate, frequency_cov_rate, interaction_qual,
			total_source_rows, extracted_rows, skipped_rows, row_coverage_pct,
			skip_reason_breakdown,
			structured_count, llm_count, grammar_count, deterministic_pct,
			warnings, grade, gate_verdict, created_at
		FROM completeness_reports
		WHERE LOWER(drug_name) = LOWER($1)
		ORDER BY created_at DESC
		LIMIT 1
	`

	r := &CompletenessReport{}
	var sectionsCovered, sectionsMissing, warnings pq.StringArray

	err := s.db.QueryRowContext(ctx, query, drugName).Scan(
		&r.ID, &r.DrugName, &r.RxCUI,
		&sectionsCovered, &sectionsMissing, &r.SectionCoveragePct,
		&r.FactCounts, &r.TotalFacts, &r.FactTypesCovered,
		&r.MeddraMatchRate, &r.FrequencyCovRate, &r.InteractionQual,
		&r.TotalSourceRows, &r.ExtractedRows, &r.SkippedRows, &r.RowCoveragePct,
		&r.SkipReasonBreakdown,
		&r.StructuredCount, &r.LLMCount, &r.GrammarCount, &r.DeterministicPct,
		&warnings, &r.Grade, &r.GateVerdict, &r.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no completeness report found for drug: %s", drugName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get completeness report: %w", err)
	}
	r.SectionsCovered = sectionsCovered
	r.SectionsMissing = sectionsMissing
	r.Warnings = warnings

	return r, nil
}

// =============================================================================
// DERIVED FACTS (SPL-extracted facts)
// =============================================================================

// SPLDerivedFact represents a fact extracted from an SPL drug label.
type SPLDerivedFact struct {
	ID                   string          `json:"id"`
	SourceDocumentID     string          `json:"sourceDocumentId"`
	SourceSectionID      string          `json:"sourceSectionId"`
	TargetKB             string          `json:"targetKb"`
	FactType             string          `json:"factType"`
	FactKey              string          `json:"factKey"`
	FactData             json.RawMessage `json:"factData"`
	ExtractionMethod     string          `json:"extractionMethod"`
	ExtractionConfidence float64         `json:"extractionConfidence"`
	EvidenceSpans        json.RawMessage `json:"evidenceSpans"`
	GovernanceStatus     string          `json:"governanceStatus"`
	ReviewedBy           *string         `json:"reviewedBy"`
	ReviewedAt           *time.Time      `json:"reviewedAt"`
	ReviewNotes          *string         `json:"reviewNotes"`
	IsActive             bool            `json:"isActive"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
	// Joined fields
	DrugName    string `json:"drugName"`
	RxCUI       string `json:"rxcui"`
	SectionCode string `json:"sectionCode"`
	SectionName string `json:"sectionName"`
}

// SPLFactFilters defines filters for querying derived facts.
type SPLFactFilters struct {
	DrugName         string
	FactType         string
	GovernanceStatus string
	ExtractionMethod string
}

// GetFactsByDrug returns derived facts for a specific drug with filters.
func (s *SPLStore) GetFactsByDrug(ctx context.Context, drugName string, filters SPLFactFilters, page, pageSize int) ([]*SPLDerivedFact, int, error) {
	// Build WHERE clause
	conditions := []string{"LOWER(sd.drug_name) = LOWER($1)"}
	args := []interface{}{drugName}
	argIdx := 2

	if filters.FactType != "" {
		conditions = append(conditions, fmt.Sprintf("df.fact_type = $%d", argIdx))
		args = append(args, filters.FactType)
		argIdx++
	}
	if filters.GovernanceStatus != "" {
		conditions = append(conditions, fmt.Sprintf("df.governance_status = $%d", argIdx))
		args = append(args, filters.GovernanceStatus)
		argIdx++
	}
	if filters.ExtractionMethod != "" {
		conditions = append(conditions, fmt.Sprintf("df.extraction_method = $%d", argIdx))
		args = append(args, filters.ExtractionMethod)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM derived_facts df
		JOIN source_documents sd ON df.source_document_id = sd.id
		WHERE %s
	`, whereClause)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count facts: %w", err)
	}

	// Data query
	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`
		SELECT
			df.id, df.source_document_id, COALESCE(df.source_section_id::text, ''),
			COALESCE(df.target_kb, ''), df.fact_type, COALESCE(df.fact_key, ''),
			df.fact_data, df.extraction_method, df.extraction_confidence,
			COALESCE(df.evidence_spans, '{}'::jsonb),
			df.governance_status, df.reviewed_by, df.reviewed_at, df.review_notes,
			df.is_active, df.created_at, df.updated_at,
			sd.drug_name, COALESCE(sd.rxcui, ''),
			COALESCE(ss.section_code, ''), COALESCE(ss.section_name, '')
		FROM derived_facts df
		JOIN source_documents sd ON df.source_document_id = sd.id
		LEFT JOIN source_sections ss ON df.source_section_id = ss.id
		WHERE %s
		ORDER BY df.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := s.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query facts: %w", err)
	}
	defer rows.Close()

	facts, err := scanDerivedFacts(rows)
	if err != nil {
		return nil, 0, err
	}

	return facts, total, nil
}

// GetPendingReviewFacts returns facts with PENDING_REVIEW status for a drug.
func (s *SPLStore) GetPendingReviewFacts(ctx context.Context, drugName string, limit int) ([]*SPLDerivedFact, error) {
	query := `
		SELECT
			df.id, df.source_document_id, COALESCE(df.source_section_id::text, ''),
			COALESCE(df.target_kb, ''), df.fact_type, COALESCE(df.fact_key, ''),
			df.fact_data, df.extraction_method, df.extraction_confidence,
			COALESCE(df.evidence_spans, '{}'::jsonb),
			df.governance_status, df.reviewed_by, df.reviewed_at, df.review_notes,
			df.is_active, df.created_at, df.updated_at,
			sd.drug_name, COALESCE(sd.rxcui, ''),
			COALESCE(ss.section_code, ''), COALESCE(ss.section_name, '')
		FROM derived_facts df
		JOIN source_documents sd ON df.source_document_id = sd.id
		LEFT JOIN source_sections ss ON df.source_section_id = ss.id
		WHERE LOWER(sd.drug_name) = LOWER($1) AND df.governance_status = 'PENDING_REVIEW'
		ORDER BY df.extraction_confidence ASC, df.created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, drugName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending facts: %w", err)
	}
	defer rows.Close()

	return scanDerivedFacts(rows)
}

// GetAutoApprovedSample returns a random sample of auto-approved facts for spot-checking.
func (s *SPLStore) GetAutoApprovedSample(ctx context.Context, drugName string, sampleSize int) ([]*SPLDerivedFact, error) {
	query := `
		SELECT
			df.id, df.source_document_id, COALESCE(df.source_section_id::text, ''),
			COALESCE(df.target_kb, ''), df.fact_type, COALESCE(df.fact_key, ''),
			df.fact_data, df.extraction_method, df.extraction_confidence,
			COALESCE(df.evidence_spans, '{}'::jsonb),
			df.governance_status, df.reviewed_by, df.reviewed_at, df.review_notes,
			df.is_active, df.created_at, df.updated_at,
			sd.drug_name, COALESCE(sd.rxcui, ''),
			COALESCE(ss.section_code, ''), COALESCE(ss.section_name, '')
		FROM derived_facts df
		JOIN source_documents sd ON df.source_document_id = sd.id
		LEFT JOIN source_sections ss ON df.source_section_id = ss.id
		WHERE LOWER(sd.drug_name) = LOWER($1) AND df.governance_status = 'APPROVED'
		ORDER BY RANDOM()
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, drugName, sampleSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get sample facts: %w", err)
	}
	defer rows.Close()

	return scanDerivedFacts(rows)
}

// SubmitReview updates a derived fact's governance status based on pharmacist review.
func (s *SPLStore) SubmitReview(ctx context.Context, factID string, decision, reviewerID, reason string) error {
	var status string
	switch decision {
	case "CONFIRM":
		status = "APPROVED"
	case "REJECT":
		status = "REJECTED"
	case "EDIT":
		status = "APPROVED" // Edited facts are approved with modifications
	case "ESCALATE":
		status = "PENDING_REVIEW" // Keep in review queue, flag for escalation
	default:
		return fmt.Errorf("invalid decision: %s", decision)
	}

	query := `
		UPDATE derived_facts
		SET governance_status = $1,
		    reviewed_by = $2,
		    reviewed_at = NOW(),
		    review_notes = $3,
		    updated_at = NOW()
		WHERE id = $4
	`

	result, err := s.db.ExecContext(ctx, query, status, reviewerID, reason, factID)
	if err != nil {
		return fmt.Errorf("failed to submit review: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("fact not found: %s", factID)
	}

	return nil
}

// =============================================================================
// SOURCE SECTIONS (SPL HTML)
// =============================================================================

// GetSectionHTML retrieves the raw HTML for a specific section of an SPL document.
func (s *SPLStore) GetSectionHTML(ctx context.Context, docID, sectionCode string) (string, string, error) {
	query := `
		SELECT raw_html, section_name
		FROM source_sections
		WHERE source_document_id = $1 AND section_code = $2
		LIMIT 1
	`

	var rawHTML, sectionName string
	err := s.db.QueryRowContext(ctx, query, docID, sectionCode).Scan(&rawHTML, &sectionName)
	if err == sql.ErrNoRows {
		return "", "", fmt.Errorf("section not found: doc=%s, code=%s", docID, sectionCode)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to get section HTML: %w", err)
	}

	return rawHTML, sectionName, nil
}

// GetDocumentSetID returns the DailyMed set_id (stored as document_id) for a source document.
func (s *SPLStore) GetDocumentSetID(ctx context.Context, docID string) (string, error) {
	var setID string
	err := s.db.QueryRowContext(ctx,
		`SELECT document_id FROM source_documents WHERE id = $1`, docID,
	).Scan(&setID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("source document not found: %s", docID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get document set_id: %w", err)
	}
	return setID, nil
}

// UpdateSectionRawHTML updates the raw_html column for a source section.
// Used to cache DailyMed-fetched XML so subsequent requests don't re-fetch.
func (s *SPLStore) UpdateSectionRawHTML(ctx context.Context, docID, sectionCode, rawHTML string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE source_sections SET raw_html = $1 WHERE source_document_id = $2 AND section_code = $3`,
		rawHTML, docID, sectionCode,
	)
	if err != nil {
		return fmt.Errorf("failed to update section raw_html: %w", err)
	}
	return nil
}

// =============================================================================
// TRIAGE DASHBOARD
// =============================================================================

// DrugTriageState represents a drug's triage state for the dashboard.
type DrugTriageState struct {
	DrugName       string          `json:"drugName"`
	RxCUI          string          `json:"rxcui"`
	Disposition    string          `json:"disposition"` // REVIEW, INVESTIGATE, OUT_OF_SCOPE, UNDECIDED
	Grade          string          `json:"grade"`
	GateVerdict    string          `json:"gateVerdict"`
	TotalFacts     int             `json:"totalFacts"`
	PendingReview  int             `json:"pendingReview"`
	Approved       int             `json:"approved"`
	Rejected       int             `json:"rejected"`
	ReviewProgress float64         `json:"reviewProgress"`
	FactsByType    json.RawMessage `json:"factsByType"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// GetTriageDashboard returns the triage state for all drugs with completeness reports.
func (s *SPLStore) GetTriageDashboard(ctx context.Context) ([]*DrugTriageState, error) {
	query := `
		WITH latest_reports AS (
			SELECT DISTINCT ON (drug_name)
				drug_name, rxcui, fact_counts, total_facts, grade, gate_verdict, created_at
			FROM completeness_reports
			ORDER BY drug_name, created_at DESC
		),
		fact_status_counts AS (
			SELECT
				sd.drug_name,
				COUNT(*) FILTER (WHERE df.governance_status = 'PENDING_REVIEW') AS pending,
				COUNT(*) FILTER (WHERE df.governance_status = 'APPROVED') AS approved,
				COUNT(*) FILTER (WHERE df.governance_status = 'REJECTED') AS rejected,
				COUNT(*) AS total_derived
			FROM derived_facts df
			JOIN source_documents sd ON df.source_document_id = sd.id
			WHERE df.is_active = true
			GROUP BY sd.drug_name
		),
		fact_type_counts AS (
			SELECT
				sd.drug_name,
				jsonb_object_agg(df.fact_type, cnt) AS by_type
			FROM (
				SELECT source_document_id, fact_type, COUNT(*) AS cnt
				FROM derived_facts
				WHERE is_active = true
				GROUP BY source_document_id, fact_type
			) df
			JOIN source_documents sd ON df.source_document_id = sd.id
			GROUP BY sd.drug_name
		)
		SELECT
			lr.drug_name, lr.rxcui,
			lr.grade, lr.gate_verdict, lr.total_facts,
			COALESCE(fsc.pending, 0), COALESCE(fsc.approved, 0), COALESCE(fsc.rejected, 0),
			COALESCE(ftc.by_type, '{}'::jsonb),
			lr.created_at
		FROM latest_reports lr
		LEFT JOIN fact_status_counts fsc ON LOWER(lr.drug_name) = LOWER(fsc.drug_name)
		LEFT JOIN fact_type_counts ftc ON LOWER(lr.drug_name) = LOWER(ftc.drug_name)
		ORDER BY lr.drug_name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get triage dashboard: %w", err)
	}
	defer rows.Close()

	var states []*DrugTriageState
	for rows.Next() {
		st := &DrugTriageState{
			Disposition: "UNDECIDED", // Default — no disposition table yet
		}

		err := rows.Scan(
			&st.DrugName, &st.RxCUI,
			&st.Grade, &st.GateVerdict, &st.TotalFacts,
			&st.PendingReview, &st.Approved, &st.Rejected,
			&st.FactsByType,
			&st.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan triage state: %w", err)
		}

		// Compute review progress as % of non-pending facts
		total := st.PendingReview + st.Approved + st.Rejected
		if total > 0 {
			st.ReviewProgress = float64(st.Approved+st.Rejected) / float64(total) * 100.0
		}

		states = append(states, st)
	}

	return states, nil
}

// =============================================================================
// SIGN-OFF
// =============================================================================

// SPLSignOff represents a pharmacist's sign-off on a drug's fact package.
type SPLSignOff struct {
	ID                        string          `json:"id"`
	DrugName                  string          `json:"drugName"`
	RxCUI                     string          `json:"rxcui"`
	TotalFacts                int             `json:"totalFacts"`
	Confirmed                 int             `json:"confirmed"`
	Edited                    int             `json:"edited"`
	Rejected                  int             `json:"rejected"`
	Added                     int             `json:"added"`
	AutoApprovedSampleSize    int             `json:"autoApprovedSampleSize"`
	AutoApprovedSampleErrors  int             `json:"autoApprovedSampleErrors"`
	FactTypeCoverage          json.RawMessage `json:"factTypeCoverage"`
	ReviewerID                string          `json:"reviewerId"`
	Attestation               string          `json:"attestation"`
	SignedAt                  time.Time       `json:"signedAt"`
	CreatedAt                 time.Time       `json:"createdAt"`
}

// SubmitSignOff records a pharmacist sign-off for a drug.
func (s *SPLStore) SubmitSignOff(ctx context.Context, signOff *SPLSignOff) error {
	query := `
		INSERT INTO spl_sign_offs (
			drug_name, rxcui, total_facts,
			confirmed, edited, rejected, added,
			auto_approved_sample_size, auto_approved_sample_errors,
			fact_type_coverage,
			reviewer_id, attestation, signed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at
	`

	return s.db.QueryRowContext(ctx, query,
		signOff.DrugName, signOff.RxCUI, signOff.TotalFacts,
		signOff.Confirmed, signOff.Edited, signOff.Rejected, signOff.Added,
		signOff.AutoApprovedSampleSize, signOff.AutoApprovedSampleErrors,
		signOff.FactTypeCoverage,
		signOff.ReviewerID, signOff.Attestation, signOff.SignedAt,
	).Scan(&signOff.ID, &signOff.CreatedAt)
}

// GetSignOff returns the latest sign-off for a drug.
func (s *SPLStore) GetSignOff(ctx context.Context, drugName string) (*SPLSignOff, error) {
	query := `
		SELECT id, drug_name, rxcui, total_facts,
		       confirmed, edited, rejected, added,
		       auto_approved_sample_size, auto_approved_sample_errors,
		       fact_type_coverage,
		       reviewer_id, attestation, signed_at, created_at
		FROM spl_sign_offs
		WHERE LOWER(drug_name) = LOWER($1)
		ORDER BY signed_at DESC
		LIMIT 1
	`

	so := &SPLSignOff{}
	err := s.db.QueryRowContext(ctx, query, drugName).Scan(
		&so.ID, &so.DrugName, &so.RxCUI, &so.TotalFacts,
		&so.Confirmed, &so.Edited, &so.Rejected, &so.Added,
		&so.AutoApprovedSampleSize, &so.AutoApprovedSampleErrors,
		&so.FactTypeCoverage,
		&so.ReviewerID, &so.Attestation, &so.SignedAt, &so.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No sign-off yet — not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sign-off: %w", err)
	}

	return so, nil
}

// =============================================================================
// HELPERS
// =============================================================================

// scanDerivedFacts scans rows into SPLDerivedFact slice.
func scanDerivedFacts(rows *sql.Rows) ([]*SPLDerivedFact, error) {
	var facts []*SPLDerivedFact

	for rows.Next() {
		f := &SPLDerivedFact{}
		var reviewedBy, reviewNotes sql.NullString
		var reviewedAt sql.NullTime

		err := rows.Scan(
			&f.ID, &f.SourceDocumentID, &f.SourceSectionID,
			&f.TargetKB, &f.FactType, &f.FactKey,
			&f.FactData, &f.ExtractionMethod, &f.ExtractionConfidence,
			&f.EvidenceSpans,
			&f.GovernanceStatus, &reviewedBy, &reviewedAt, &reviewNotes,
			&f.IsActive, &f.CreatedAt, &f.UpdatedAt,
			&f.DrugName, &f.RxCUI,
			&f.SectionCode, &f.SectionName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan derived fact: %w", err)
		}

		if reviewedBy.Valid {
			f.ReviewedBy = &reviewedBy.String
		}
		if reviewedAt.Valid {
			f.ReviewedAt = &reviewedAt.Time
		}
		if reviewNotes.Valid {
			f.ReviewNotes = &reviewNotes.String
		}

		facts = append(facts, f)
	}

	return facts, nil
}

// ParsePagination extracts page and pageSize from query string with defaults.
func ParsePagination(pageStr, pageSizeStr string, defaultPage, defaultPageSize int) (int, int) {
	page := defaultPage
	pageSize := defaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 5000 {
			pageSize = ps
		}
	}

	return page, pageSize
}
