-- V4 Multi-Channel Extraction Schema
-- Migration: 02-l2-multichannel-schema.sql
-- Extends existing factstore schema (01-schema.sql) with L2 multi-channel tables.
-- Does NOT modify any existing V3 tables.

-- =============================================================================
-- Extraction Jobs for Multi-Channel Runs
-- =============================================================================

CREATE TABLE IF NOT EXISTS l2_extraction_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_pdf VARCHAR(500) NOT NULL,
    source_hash VARCHAR(64) NOT NULL,
    guideline_authority VARCHAR(100) NOT NULL,
    guideline_document VARCHAR(500) NOT NULL,
    normalized_text TEXT,

    -- Channel statuses (Pipeline 1)
    channel_0_status VARCHAR(20) DEFAULT 'PENDING',
    channel_a_status VARCHAR(20) DEFAULT 'PENDING',
    channel_b_status VARCHAR(20) DEFAULT 'PENDING',
    channel_c_status VARCHAR(20) DEFAULT 'PENDING',
    channel_d_status VARCHAR(20) DEFAULT 'PENDING',
    channel_e_status VARCHAR(20) DEFAULT 'PENDING',
    channel_f_status VARCHAR(20) DEFAULT 'PENDING',
    merger_status VARCHAR(20) DEFAULT 'PENDING',
    review_status VARCHAR(20) DEFAULT 'PENDING',   -- PENDING -> IN_REVIEW -> COMPLETED

    -- Pipeline 2 statuses (triggered after review approval)
    dossier_status VARCHAR(20) DEFAULT 'PENDING',
    l3_status VARCHAR(20) DEFAULT 'PENDING',
    l4_status VARCHAR(20) DEFAULT 'PENDING',
    l5_status VARCHAR(20) DEFAULT 'PENDING',

    -- Metrics
    total_raw_spans INT DEFAULT 0,
    total_merged_spans INT DEFAULT 0,
    spans_confirmed INT DEFAULT 0,
    spans_rejected INT DEFAULT 0,
    spans_edited INT DEFAULT 0,
    spans_added INT DEFAULT 0,
    dossiers_created INT DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    review_started_at TIMESTAMPTZ,
    review_completed_at TIMESTAMPTZ,
    pipeline2_started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_l2_jobs_review ON l2_extraction_jobs(review_status);
CREATE INDEX idx_l2_jobs_source ON l2_extraction_jobs(source_hash);

-- Updated_at trigger
CREATE TRIGGER update_l2_jobs_updated_at BEFORE UPDATE ON l2_extraction_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- Per-Channel Raw Spans
-- =============================================================================

CREATE TABLE IF NOT EXISTS l2_raw_spans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    channel VARCHAR(2) NOT NULL,            -- 'B','C','D','E','F'
    text TEXT NOT NULL,
    start_offset INT NOT NULL,
    end_offset INT NOT NULL,
    confidence DECIMAL(4,3) NOT NULL,
    page_number INT,
    section_id VARCHAR(50),
    table_id VARCHAR(50),
    source_block_type VARCHAR(30),
    channel_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_l2_raw_spans_job ON l2_raw_spans(job_id);
CREATE INDEX idx_l2_raw_spans_channel ON l2_raw_spans(job_id, channel);
CREATE INDEX idx_l2_raw_spans_offset ON l2_raw_spans(job_id, start_offset);

-- =============================================================================
-- Merged Spans (Signal Merger Output = Reviewer Queue)
-- =============================================================================

CREATE TABLE IF NOT EXISTS l2_merged_spans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    start_offset INT NOT NULL,
    end_offset INT NOT NULL,
    contributing_channels TEXT[] NOT NULL,
    channel_confidences JSONB NOT NULL,
    merged_confidence DECIMAL(4,3) NOT NULL,
    has_disagreement BOOLEAN DEFAULT FALSE,
    disagreement_detail TEXT,
    page_number INT,
    section_id VARCHAR(50),
    table_id VARCHAR(50),
    review_status VARCHAR(20) DEFAULT 'PENDING',
    reviewer_text TEXT,
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_l2_merged_review ON l2_merged_spans(job_id, review_status);
CREATE INDEX idx_l2_merged_disagreement ON l2_merged_spans(job_id) WHERE has_disagreement = TRUE;

-- =============================================================================
-- Reviewer Decisions (Audit Trail)
-- =============================================================================

CREATE TABLE IF NOT EXISTS l2_reviewer_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merged_span_id UUID NOT NULL REFERENCES l2_merged_spans(id) ON DELETE CASCADE,
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    action VARCHAR(20) NOT NULL,            -- CONFIRM, REJECT, EDIT, ADD
    original_text TEXT,
    edited_text TEXT,
    reviewer_id VARCHAR(100) NOT NULL,
    decided_at TIMESTAMPTZ DEFAULT NOW(),
    note TEXT
);

CREATE INDEX idx_l2_decisions_job ON l2_reviewer_decisions(job_id);
CREATE INDEX idx_l2_decisions_span ON l2_reviewer_decisions(merged_span_id);

-- =============================================================================
-- Per-Drug Dossier Results (Pipeline 2 Per-Drug Tracking)
-- Allows retrying L3 for a single drug without re-running Pipeline 1
-- =============================================================================

CREATE TABLE IF NOT EXISTS l2_dossier_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES l2_extraction_jobs(id) ON DELETE CASCADE,
    drug_name VARCHAR(200) NOT NULL,
    rxnorm_candidate VARCHAR(20),
    span_count INT NOT NULL,
    l3_status VARCHAR(20) DEFAULT 'PENDING',    -- PENDING -> RUNNING -> COMPLETED -> FAILED
    l3_result JSONB,                             -- KB-specific extraction result
    l3_error TEXT,                                -- error message if FAILED
    l4_status VARCHAR(20) DEFAULT 'PENDING',
    l5_status VARCHAR(20) DEFAULT 'PENDING',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_l2_dossier_job ON l2_dossier_results(job_id);
CREATE INDEX idx_l2_dossier_status ON l2_dossier_results(job_id, l3_status);
