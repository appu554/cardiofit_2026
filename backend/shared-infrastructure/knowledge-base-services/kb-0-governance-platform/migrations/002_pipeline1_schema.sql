-- =============================================================================
-- Pipeline 1 Review Schema — L2 Extraction Review Tables
-- =============================================================================
-- Text QA tables for reviewing V4.2.1 guideline extraction output.
-- Separate from Phase 2 clinical fact governance (clinical_facts, derived_facts).
-- All tables prefixed with l2_ to avoid collision.
-- =============================================================================

-- Enable uuid-ossp if not already available
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- L2_EXTRACTION_JOBS — One row per Pipeline 1 run
-- =============================================================================

CREATE TABLE l2_extraction_jobs (
    job_id          UUID PRIMARY KEY,
    source_pdf      TEXT NOT NULL,
    page_range      TEXT,
    pipeline_version TEXT NOT NULL DEFAULT 'V4.2.1',
    l1_tag          TEXT,
    total_merged_spans  INTEGER NOT NULL DEFAULT 0,
    total_sections      INTEGER NOT NULL DEFAULT 0,
    total_pages         INTEGER NOT NULL DEFAULT 0,
    alignment_confidence DOUBLE PRECISION,
    l1_oracle_stats     JSONB DEFAULT '{}'::jsonb,

    -- Denormalized review counters (updated transactionally with span reviews)
    spans_confirmed INTEGER NOT NULL DEFAULT 0,
    spans_rejected  INTEGER NOT NULL DEFAULT 0,
    spans_edited    INTEGER NOT NULL DEFAULT 0,
    spans_added     INTEGER NOT NULL DEFAULT 0,
    spans_pending   INTEGER NOT NULL DEFAULT 0,

    -- Job lifecycle
    status          TEXT NOT NULL DEFAULT 'PENDING_REVIEW'
                    CHECK (status IN ('PENDING_REVIEW', 'IN_PROGRESS', 'COMPLETED', 'ARCHIVED')),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_l2_jobs_status ON l2_extraction_jobs(status);
CREATE INDEX idx_l2_jobs_created ON l2_extraction_jobs(created_at DESC);

-- =============================================================================
-- L2_MERGED_SPANS — One row per merged span from Signal Merger
-- =============================================================================

CREATE TABLE l2_merged_spans (
    id                  UUID PRIMARY KEY,
    job_id              UUID NOT NULL REFERENCES l2_extraction_jobs(job_id) ON DELETE CASCADE,
    text                TEXT NOT NULL,
    start_offset        INTEGER NOT NULL,
    end_offset          INTEGER NOT NULL,
    contributing_channels TEXT[] NOT NULL DEFAULT '{}',
    channel_confidences JSONB DEFAULT '{}'::jsonb,
    merged_confidence   DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    has_disagreement    BOOLEAN NOT NULL DEFAULT FALSE,
    disagreement_detail TEXT,
    page_number         INTEGER,
    section_id          TEXT,
    table_id            TEXT,

    -- Review state
    review_status       TEXT NOT NULL DEFAULT 'PENDING'
                        CHECK (review_status IN ('PENDING', 'CONFIRMED', 'REJECTED', 'EDITED', 'ADDED')),
    reviewer_text       TEXT,
    reviewed_by         TEXT,
    reviewed_at         TIMESTAMP WITH TIME ZONE,

    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_l2_spans_job ON l2_merged_spans(job_id);
CREATE INDEX idx_l2_spans_status ON l2_merged_spans(review_status);
CREATE INDEX idx_l2_spans_job_status ON l2_merged_spans(job_id, review_status);
CREATE INDEX idx_l2_spans_job_section ON l2_merged_spans(job_id, section_id);
CREATE INDEX idx_l2_spans_job_page ON l2_merged_spans(job_id, page_number);
CREATE INDEX idx_l2_spans_confidence ON l2_merged_spans(merged_confidence);
CREATE INDEX idx_l2_spans_text_search ON l2_merged_spans USING gin(to_tsvector('english', text));

-- =============================================================================
-- L2_REVIEWER_DECISIONS — Immutable audit trail (21 CFR Part 11)
-- =============================================================================

CREATE TABLE l2_reviewer_decisions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    merged_span_id  UUID NOT NULL REFERENCES l2_merged_spans(id) ON DELETE CASCADE,
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id) ON DELETE CASCADE,
    action          TEXT NOT NULL CHECK (action IN ('CONFIRM', 'REJECT', 'EDIT', 'ADD')),
    original_text   TEXT,
    edited_text     TEXT,
    reviewer_id     TEXT NOT NULL,
    decided_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    note            TEXT
);

CREATE INDEX idx_l2_decisions_span ON l2_reviewer_decisions(merged_span_id);
CREATE INDEX idx_l2_decisions_job ON l2_reviewer_decisions(job_id);
CREATE INDEX idx_l2_decisions_reviewer ON l2_reviewer_decisions(reviewer_id);
CREATE INDEX idx_l2_decisions_decided ON l2_reviewer_decisions(decided_at DESC);

-- Immutability trigger: prevent UPDATE and DELETE on decisions
CREATE OR REPLACE FUNCTION prevent_decision_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'l2_reviewer_decisions is immutable — UPDATE and DELETE are prohibited';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_l2_decisions_immutable_update
    BEFORE UPDATE ON l2_reviewer_decisions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_decision_mutation();

CREATE TRIGGER tr_l2_decisions_immutable_delete
    BEFORE DELETE ON l2_reviewer_decisions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_decision_mutation();

-- =============================================================================
-- L2_SECTION_PASSAGES — Section prose text with span provenance
-- =============================================================================

CREATE TABLE l2_section_passages (
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id) ON DELETE CASCADE,
    section_id      TEXT NOT NULL,
    heading         TEXT NOT NULL,
    page_number     INTEGER,
    prose_text      TEXT,
    span_ids        UUID[] DEFAULT '{}',
    span_count      INTEGER NOT NULL DEFAULT 0,
    child_section_ids TEXT[] DEFAULT '{}',
    start_offset    INTEGER,
    end_offset      INTEGER,

    PRIMARY KEY (job_id, section_id)
);

CREATE INDEX idx_l2_passages_job ON l2_section_passages(job_id);

-- =============================================================================
-- L2_GUIDELINE_TREE — Full tree structure + normalized text for offset lookup
-- =============================================================================

CREATE TABLE l2_guideline_tree (
    job_id          UUID PRIMARY KEY REFERENCES l2_extraction_jobs(job_id) ON DELETE CASCADE,
    tree_json       JSONB NOT NULL,
    normalized_text TEXT
);

-- =============================================================================
-- VIEW: v_l2_job_progress — Pre-computed progress for job list page
-- =============================================================================

CREATE OR REPLACE VIEW v_l2_job_progress AS
SELECT
    j.job_id,
    j.source_pdf,
    j.page_range,
    j.pipeline_version,
    j.l1_tag,
    j.total_merged_spans,
    j.total_sections,
    j.total_pages,
    j.status,
    j.created_at,
    j.updated_at,
    j.completed_at,
    j.spans_confirmed,
    j.spans_rejected,
    j.spans_edited,
    j.spans_added,
    j.spans_pending,
    CASE
        WHEN j.total_merged_spans + j.spans_added = 0 THEN 0
        ELSE ROUND(
            ((j.spans_confirmed + j.spans_rejected + j.spans_edited)::numeric
             / (j.total_merged_spans + j.spans_added)::numeric) * 100, 1
        )
    END AS completion_pct
FROM l2_extraction_jobs j;

-- =============================================================================
-- AUTO-UPDATE updated_at trigger for jobs
-- =============================================================================

CREATE OR REPLACE FUNCTION l2_update_job_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_l2_jobs_updated
    BEFORE UPDATE ON l2_extraction_jobs
    FOR EACH ROW
    EXECUTE FUNCTION l2_update_job_timestamp();

-- =============================================================================
-- COMPLETION MESSAGE
-- =============================================================================
DO $$
BEGIN
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Pipeline 1 Review Schema Created Successfully';
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Tables: l2_extraction_jobs, l2_merged_spans,';
    RAISE NOTICE '        l2_reviewer_decisions, l2_section_passages,';
    RAISE NOTICE '        l2_guideline_tree';
    RAISE NOTICE 'View:   v_l2_job_progress';
    RAISE NOTICE '===================================================';
END $$;
