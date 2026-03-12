-- =============================================================================
-- KB-0 Governance Platform — Idempotent GCP Cloud SQL Migration
-- =============================================================================
-- Combines migrations 001–004 with IF NOT EXISTS / ON CONFLICT safety.
-- Safe to re-run on a database that already has some or all tables.
-- =============================================================================

BEGIN;

-- ═══════════════════════════════════════════════════════════════════════════════
-- MIGRATION 001: Core Governance Schema
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ── KNOWLEDGE ITEMS ──

CREATE TABLE IF NOT EXISTS knowledge_items (
    item_id VARCHAR(100) PRIMARY KEY,
    kb VARCHAR(20) NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    name VARCHAR(500) NOT NULL,
    description TEXT,
    content_ref VARCHAR(500),
    content_hash VARCHAR(64),
    source_authority VARCHAR(50),
    source_document VARCHAR(500),
    source_section VARCHAR(200),
    source_url TEXT,
    source_jurisdiction VARCHAR(20),
    source_effective_date VARCHAR(50),
    source_expiration_date VARCHAR(50),
    risk_level VARCHAR(20) DEFAULT 'MODERATE',
    workflow_template VARCHAR(50) NOT NULL,
    requires_dual_review BOOLEAN DEFAULT FALSE,
    risk_flags JSONB DEFAULT '[]'::jsonb,
    state VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    version VARCHAR(20) DEFAULT '1.0',
    created_by VARCHAR(200),
    governance_trail JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    activated_at TIMESTAMP WITH TIME ZONE,
    retired_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_state CHECK (state IN (
        'DRAFT', 'SUBMITTED',
        'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE', 'AUTO_VALIDATION',
        'REVIEWED',
        'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL',
        'APPROVED', 'ACTIVE', 'EMERGENCY_ACTIVE',
        'HOLD', 'RETIRED', 'REJECTED'
    )),
    CONSTRAINT chk_risk_level CHECK (risk_level IN ('LOW', 'MODERATE', 'HIGH', 'CRITICAL'))
);

CREATE INDEX IF NOT EXISTS idx_knowledge_items_kb ON knowledge_items(kb);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_state ON knowledge_items(state);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_kb_state ON knowledge_items(kb, state);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_risk_level ON knowledge_items(risk_level);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_name ON knowledge_items USING gin(name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_authority ON knowledge_items(source_authority);
CREATE INDEX IF NOT EXISTS idx_knowledge_items_created_at ON knowledge_items(created_at DESC);

-- ── AUDIT ENTRIES ──

CREATE TABLE IF NOT EXISTS audit_entries (
    id VARCHAR(100) PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    action VARCHAR(50) NOT NULL,
    decision VARCHAR(50),
    actor_id VARCHAR(100) NOT NULL,
    actor_name VARCHAR(200),
    actor_role VARCHAR(50),
    credentials VARCHAR(200),
    item_id VARCHAR(100) NOT NULL,
    kb VARCHAR(20) NOT NULL,
    item_version VARCHAR(20),
    previous_state VARCHAR(30),
    new_state VARCHAR(30),
    notes TEXT,
    checklist JSONB,
    attestations JSONB,
    ip_address VARCHAR(45),
    session_id VARCHAR(100),
    content_hash VARCHAR(64),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_audit_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_audit_entries_item ON audit_entries(item_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_actor ON audit_entries(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_entries_action ON audit_entries(action);
CREATE INDEX IF NOT EXISTS idx_audit_entries_timestamp ON audit_entries(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_entries_kb ON audit_entries(kb);

-- ── SLA RECORDS ──

CREATE TABLE IF NOT EXISTS sla_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id VARCHAR(100) NOT NULL,
    sla_type VARCHAR(50) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    due_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    breach_notified BOOLEAN DEFAULT FALSE,
    assigned_to VARCHAR(100),
    escalated_to VARCHAR(100),
    escalation_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT fk_sla_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sla_records_item ON sla_records(item_id);
CREATE INDEX IF NOT EXISTS idx_sla_records_status ON sla_records(status);
CREATE INDEX IF NOT EXISTS idx_sla_records_due ON sla_records(due_at) WHERE status = 'ACTIVE';

-- ── NOTIFICATION QUEUE ──

CREATE TABLE IF NOT EXISTS notification_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_type VARCHAR(50) NOT NULL,
    priority VARCHAR(20) DEFAULT 'NORMAL',
    recipient_id VARCHAR(100) NOT NULL,
    recipient_email VARCHAR(255),
    subject VARCHAR(500),
    body TEXT,
    metadata JSONB,
    item_id VARCHAR(100),
    status VARCHAR(20) DEFAULT 'PENDING',
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_notification_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notification_queue_status ON notification_queue(status);
CREATE INDEX IF NOT EXISTS idx_notification_queue_recipient ON notification_queue(recipient_id);

-- ── WORKFLOW TEMPLATES ──

CREATE TABLE IF NOT EXISTS workflow_templates (
    template_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    transitions JSONB NOT NULL,
    sla_config JSONB,
    checklist_config JSONB,
    attestation_config JSONB,
    applicable_kbs TEXT[],
    applicable_item_types TEXT[],
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ── KB-1 INTEGRATION LOG ──

CREATE TABLE IF NOT EXISTS kb1_integration_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kb1_rule_id VARCHAR(100) NOT NULL,
    rxnorm_code VARCHAR(20),
    item_id VARCHAR(100),
    operation VARCHAR(50) NOT NULL,
    operation_status VARCHAR(20) NOT NULL,
    request_payload JSONB,
    response_payload JSONB,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_kb1_log_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_kb1_log_rule ON kb1_integration_log(kb1_rule_id);
CREATE INDEX IF NOT EXISTS idx_kb1_log_operation ON kb1_integration_log(operation);
CREATE INDEX IF NOT EXISTS idx_kb1_log_status ON kb1_integration_log(operation_status);

-- ── FUNCTIONS ──

CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ── TRIGGERS (idempotent via pg_trigger check) ──

DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tr_knowledge_items_updated') THEN
    CREATE TRIGGER tr_knowledge_items_updated
        BEFORE UPDATE ON knowledge_items
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
END IF;
END $$;

DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tr_sla_records_updated') THEN
    CREATE TRIGGER tr_sla_records_updated
        BEFORE UPDATE ON sla_records
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
END IF;
END $$;

DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tr_workflow_templates_updated') THEN
    CREATE TRIGGER tr_workflow_templates_updated
        BEFORE UPDATE ON workflow_templates
        FOR EACH ROW EXECUTE FUNCTION update_updated_at();
END IF;
END $$;

-- ── VIEWS ──

CREATE OR REPLACE VIEW v_pending_governance AS
SELECT
    ki.item_id, ki.kb, ki.item_type, ki.name, ki.state, ki.risk_level,
    ki.requires_dual_review, ki.source_authority, ki.source_jurisdiction,
    ki.created_at, ki.updated_at,
    sr.due_at as sla_due_at, sr.status as sla_status
FROM knowledge_items ki
LEFT JOIN sla_records sr ON ki.item_id = sr.item_id AND sr.status = 'ACTIVE'
WHERE ki.state NOT IN ('ACTIVE', 'RETIRED', 'REJECTED')
ORDER BY
    CASE ki.risk_level
        WHEN 'CRITICAL' THEN 1 WHEN 'HIGH' THEN 2
        WHEN 'MODERATE' THEN 3 ELSE 4
    END,
    ki.created_at ASC;

CREATE OR REPLACE VIEW v_kb_metrics AS
SELECT
    kb,
    COUNT(*) FILTER (WHERE state = 'ACTIVE') AS active_count,
    COUNT(*) FILTER (WHERE state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE')) AS pending_review_count,
    COUNT(*) FILTER (WHERE state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL')) AS pending_approval_count,
    COUNT(*) FILTER (WHERE state = 'HOLD') AS hold_count,
    COUNT(*) FILTER (WHERE state = 'EMERGENCY_ACTIVE') AS emergency_count,
    COUNT(*) FILTER (WHERE state = 'ACTIVE' AND risk_level IN ('HIGH', 'CRITICAL')) AS high_risk_active_count,
    COUNT(*) AS total_count
FROM knowledge_items
GROUP BY kb;

-- ── SEED WORKFLOW TEMPLATES (ON CONFLICT = skip if exists) ──

INSERT INTO workflow_templates (template_id, name, description, transitions, applicable_kbs, applicable_item_types)
VALUES
(
    'CLINICAL_HIGH',
    'Clinical High-Risk Workflow',
    'For high-risk clinical items requiring dual review and CMO approval',
    '{"transitions": [
        {"from": ["DRAFT"], "to": "SUBMITTED", "action": "submit", "actors": ["author", "system"]},
        {"from": ["SUBMITTED"], "to": "PRIMARY_REVIEW", "action": "assign_review", "actors": ["coordinator", "system"]},
        {"from": ["PRIMARY_REVIEW"], "to": "SECONDARY_REVIEW", "action": "submit_review", "actors": ["clinical_pharmacist", "physician"]},
        {"from": ["SECONDARY_REVIEW"], "to": "REVIEWED", "action": "submit_review", "actors": ["clinical_pharmacist", "physician"]},
        {"from": ["REVIEWED"], "to": "CMO_APPROVAL", "action": "request_approval", "actors": ["coordinator", "system"]},
        {"from": ["CMO_APPROVAL"], "to": "APPROVED", "action": "approve", "actors": ["cmo", "director"]},
        {"from": ["APPROVED"], "to": "ACTIVE", "action": "activate", "actors": ["system"]}
    ]}'::jsonb,
    ARRAY['KB1', 'KB2', 'KB4'],
    ARRAY['DOSING_RULE', 'INTERACTION', 'SAFETY_ALERT']
),
(
    'CLINICAL_STANDARD',
    'Clinical Standard Workflow',
    'For moderate-risk clinical items with single review',
    '{"transitions": [
        {"from": ["DRAFT"], "to": "SUBMITTED", "action": "submit", "actors": ["author", "system"]},
        {"from": ["SUBMITTED"], "to": "PRIMARY_REVIEW", "action": "assign_review", "actors": ["coordinator", "system"]},
        {"from": ["PRIMARY_REVIEW"], "to": "REVIEWED", "action": "submit_review", "actors": ["clinical_pharmacist", "physician"]},
        {"from": ["REVIEWED"], "to": "DIRECTOR_APPROVAL", "action": "request_approval", "actors": ["coordinator", "system"]},
        {"from": ["DIRECTOR_APPROVAL"], "to": "APPROVED", "action": "approve", "actors": ["director"]},
        {"from": ["APPROVED"], "to": "ACTIVE", "action": "activate", "actors": ["system"]}
    ]}'::jsonb,
    ARRAY['KB1', 'KB2', 'KB3', 'KB5'],
    ARRAY['GUIDELINE', 'PROTOCOL', 'PATHWAY']
),
(
    'TERMINOLOGY_UPDATE',
    'Terminology Update Workflow',
    'For terminology and code system updates',
    '{"transitions": [
        {"from": ["DRAFT"], "to": "SUBMITTED", "action": "submit", "actors": ["author", "system"]},
        {"from": ["SUBMITTED"], "to": "AUTO_VALIDATION", "action": "auto_validate", "actors": ["system"]},
        {"from": ["AUTO_VALIDATION"], "to": "LEAD_APPROVAL", "action": "request_approval", "actors": ["system"]},
        {"from": ["LEAD_APPROVAL"], "to": "APPROVED", "action": "approve", "actors": ["terminology_lead"]},
        {"from": ["APPROVED"], "to": "ACTIVE", "action": "activate", "actors": ["system"]}
    ]}'::jsonb,
    ARRAY['KB7'],
    ARRAY['TERMINOLOGY', 'CODE_SYSTEM', 'VALUE_SET']
)
ON CONFLICT (template_id) DO NOTHING;


-- ═══════════════════════════════════════════════════════════════════════════════
-- MIGRATION 002: Pipeline 1 Review Schema
-- ═══════════════════════════════════════════════════════════════════════════════

-- ── L2_EXTRACTION_JOBS ──

CREATE TABLE IF NOT EXISTS l2_extraction_jobs (
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
    spans_confirmed INTEGER NOT NULL DEFAULT 0,
    spans_rejected  INTEGER NOT NULL DEFAULT 0,
    spans_edited    INTEGER NOT NULL DEFAULT 0,
    spans_added     INTEGER NOT NULL DEFAULT 0,
    spans_pending   INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'PENDING_REVIEW'
                    CHECK (status IN ('PENDING_REVIEW', 'IN_PROGRESS', 'COMPLETED', 'ARCHIVED')),
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_l2_jobs_status ON l2_extraction_jobs(status);
CREATE INDEX IF NOT EXISTS idx_l2_jobs_created ON l2_extraction_jobs(created_at DESC);

-- ── L2_MERGED_SPANS ──

CREATE TABLE IF NOT EXISTS l2_merged_spans (
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
    review_status       TEXT NOT NULL DEFAULT 'PENDING'
                        CHECK (review_status IN ('PENDING', 'CONFIRMED', 'REJECTED', 'EDITED', 'ADDED')),
    reviewer_text       TEXT,
    reviewed_by         TEXT,
    reviewed_at         TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_l2_spans_job ON l2_merged_spans(job_id);
CREATE INDEX IF NOT EXISTS idx_l2_spans_status ON l2_merged_spans(review_status);
CREATE INDEX IF NOT EXISTS idx_l2_spans_job_status ON l2_merged_spans(job_id, review_status);
CREATE INDEX IF NOT EXISTS idx_l2_spans_job_section ON l2_merged_spans(job_id, section_id);
CREATE INDEX IF NOT EXISTS idx_l2_spans_job_page ON l2_merged_spans(job_id, page_number);
CREATE INDEX IF NOT EXISTS idx_l2_spans_confidence ON l2_merged_spans(merged_confidence);
CREATE INDEX IF NOT EXISTS idx_l2_spans_text_search ON l2_merged_spans USING gin(to_tsvector('english', text));

-- ── L2_REVIEWER_DECISIONS (immutable audit trail) ──

CREATE TABLE IF NOT EXISTS l2_reviewer_decisions (
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

CREATE INDEX IF NOT EXISTS idx_l2_decisions_span ON l2_reviewer_decisions(merged_span_id);
CREATE INDEX IF NOT EXISTS idx_l2_decisions_job ON l2_reviewer_decisions(job_id);
CREATE INDEX IF NOT EXISTS idx_l2_decisions_reviewer ON l2_reviewer_decisions(reviewer_id);
CREATE INDEX IF NOT EXISTS idx_l2_decisions_decided ON l2_reviewer_decisions(decided_at DESC);

-- Immutability function
CREATE OR REPLACE FUNCTION prevent_decision_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'l2_reviewer_decisions is immutable — UPDATE and DELETE are prohibited';
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tr_l2_decisions_immutable_update') THEN
    CREATE TRIGGER tr_l2_decisions_immutable_update
        BEFORE UPDATE ON l2_reviewer_decisions
        FOR EACH ROW EXECUTE FUNCTION prevent_decision_mutation();
END IF;
END $$;

DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tr_l2_decisions_immutable_delete') THEN
    CREATE TRIGGER tr_l2_decisions_immutable_delete
        BEFORE DELETE ON l2_reviewer_decisions
        FOR EACH ROW EXECUTE FUNCTION prevent_decision_mutation();
END IF;
END $$;

-- ── L2_SECTION_PASSAGES ──

CREATE TABLE IF NOT EXISTS l2_section_passages (
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

CREATE INDEX IF NOT EXISTS idx_l2_passages_job ON l2_section_passages(job_id);

-- ── L2_GUIDELINE_TREE ──

CREATE TABLE IF NOT EXISTS l2_guideline_tree (
    job_id          UUID PRIMARY KEY REFERENCES l2_extraction_jobs(job_id) ON DELETE CASCADE,
    tree_json       JSONB NOT NULL,
    normalized_text TEXT
);

-- ── VIEW: v_l2_job_progress ──

CREATE OR REPLACE VIEW v_l2_job_progress AS
SELECT
    j.job_id, j.source_pdf, j.page_range, j.pipeline_version, j.l1_tag,
    j.total_merged_spans, j.total_sections, j.total_pages, j.status,
    j.created_at, j.updated_at, j.completed_at,
    j.spans_confirmed, j.spans_rejected, j.spans_edited, j.spans_added, j.spans_pending,
    CASE
        WHEN j.total_merged_spans + j.spans_added = 0 THEN 0
        ELSE ROUND(
            ((j.spans_confirmed + j.spans_rejected + j.spans_edited)::numeric
             / (j.total_merged_spans + j.spans_added)::numeric) * 100, 1
        )
    END AS completion_pct
FROM l2_extraction_jobs j;

-- Auto-update trigger for jobs
CREATE OR REPLACE FUNCTION l2_update_job_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$ BEGIN
IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'tr_l2_jobs_updated') THEN
    CREATE TRIGGER tr_l2_jobs_updated
        BEFORE UPDATE ON l2_extraction_jobs
        FOR EACH ROW EXECUTE FUNCTION l2_update_job_timestamp();
END IF;
END $$;


-- ═══════════════════════════════════════════════════════════════════════════════
-- MIGRATION 003: Page Decisions
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS l2_page_decisions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id) ON DELETE CASCADE,
    page_number     INTEGER NOT NULL,
    action          TEXT NOT NULL CHECK (action IN ('ACCEPT', 'FLAG', 'ESCALATE')),
    reviewer_id     TEXT NOT NULL,
    note            TEXT,
    decided_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_l2_page_decision UNIQUE (job_id, page_number)
);

CREATE INDEX IF NOT EXISTS idx_l2_page_decisions_job ON l2_page_decisions(job_id);
CREATE INDEX IF NOT EXISTS idx_l2_page_decisions_job_page ON l2_page_decisions(job_id, page_number);

CREATE OR REPLACE VIEW l2_passages_for_l3 AS
SELECT
    sp.job_id, sp.section_id, sp.heading, sp.page_number, sp.child_section_ids,
    COALESCE(
        STRING_AGG(
            COALESCE(ms.reviewer_text, ms.text),
            ' ' ORDER BY ms.start_offset
        ),
        sp.prose_text
    ) AS prose_text,
    ARRAY_AGG(ms.id ORDER BY ms.start_offset)
        FILTER (WHERE ms.id IS NOT NULL) AS span_ids,
    COUNT(ms.id) AS span_count,
    MAX(ms.reviewed_at) AS patched_at
FROM l2_section_passages sp
LEFT JOIN l2_merged_spans ms
    ON ms.job_id = sp.job_id
    AND ms.id = ANY(sp.span_ids)
    AND ms.review_status != 'REJECTED'
GROUP BY sp.job_id, sp.section_id, sp.heading,
         sp.page_number, sp.child_section_ids, sp.prose_text;


-- ═══════════════════════════════════════════════════════════════════════════════
-- MIGRATION 004: Highlight HTML + Source PDF Path
-- ═══════════════════════════════════════════════════════════════════════════════

ALTER TABLE l2_guideline_tree ADD COLUMN IF NOT EXISTS highlight_html TEXT;
ALTER TABLE l2_extraction_jobs ADD COLUMN IF NOT EXISTS source_pdf_path TEXT;


-- ═══════════════════════════════════════════════════════════════════════════════
-- MIGRATION 005: Bounding Box + Surrounding Context
-- ═══════════════════════════════════════════════════════════════════════════════

-- PDF bounding box as JSONB array [x0, y0, x1, y1] in PDF points.
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS bbox JSONB;

-- Adjacent block text for L1_RECOVERY spans (reviewer context).
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS surrounding_context TEXT;


-- ═══════════════════════════════════════════════════════════════════════════════
-- MIGRATION 006: CoverageGuard Sprint 1 + Sprint 2
-- ═══════════════════════════════════════════════════════════════════════════════

-- Risk tier: 1 = critical, 2 = warning, 3 = info.
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS tier SMALLINT
    CHECK (tier IS NULL OR tier IN (1, 2, 3));

-- CoverageGuard alert payload (numeric_mismatch, branch_loss, llm_only, negation_flip).
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS coverage_guard_alert JSONB;

-- Semantic highlighting tokens (numerics, conditions, negations).
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS semantic_tokens JSONB;

-- Structured reject reason for auditable rejection categories.
ALTER TABLE l2_reviewer_decisions ADD COLUMN IF NOT EXISTS reject_reason TEXT
    CHECK (reject_reason IS NULL OR reject_reason IN (
        'not_in_source', 'numeric_mismatch', 'negation_error', 'out_of_scope',
        'duplicate', 'hallucination', 'branch_incomplete', 'other'
    ));

-- Source PDF path and reviewer who completed the job.
ALTER TABLE l2_extraction_jobs ADD COLUMN IF NOT EXISTS source_pdf_path TEXT;
ALTER TABLE l2_extraction_jobs ADD COLUMN IF NOT EXISTS completed_by TEXT;


-- ═══════════════════════════════════════════════════════════════════════════════
-- MIGRATION 007: Revalidation Runs + Output Contract Assembly
-- ═══════════════════════════════════════════════════════════════════════════════

-- Revalidation runs: tracks each re-validation iteration and its delta.
CREATE TABLE IF NOT EXISTS l2_revalidation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id),
    iteration       INTEGER NOT NULL CHECK (iteration >= 1),
    verdict         TEXT NOT NULL CHECK (verdict IN ('PASS', 'BLOCK')),
    edited_span_count   INTEGER NOT NULL DEFAULT 0,
    rejected_span_count INTEGER NOT NULL DEFAULT 0,
    added_span_count    INTEGER NOT NULL DEFAULT 0,
    deltas          JSONB NOT NULL DEFAULT '[]'::jsonb,
    triggered_by    TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, iteration)
);

CREATE INDEX IF NOT EXISTS idx_l2_revalidation_runs_job
    ON l2_revalidation_runs(job_id);

-- Output contracts: 5-section package for Pipeline 2 handoff.
CREATE TABLE IF NOT EXISTS l2_output_contracts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id),
    confirmed_facts JSONB NOT NULL DEFAULT '[]'::jsonb,
    added_facts     JSONB NOT NULL DEFAULT '[]'::jsonb,
    section_tree    JSONB NOT NULL DEFAULT '{}'::jsonb,
    evidence_envelope JSONB NOT NULL DEFAULT '{}'::jsonb,
    rejection_log   JSONB NOT NULL DEFAULT '[]'::jsonb,
    assembled_by    TEXT NOT NULL,
    assembled_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id)
);

CREATE INDEX IF NOT EXISTS idx_l2_output_contracts_job
    ON l2_output_contracts(job_id);


-- ═══════════════════════════════════════════════════════════════════════════════
-- COMPLETION
-- ═══════════════════════════════════════════════════════════════════════════════

COMMIT;

DO $$
BEGIN
    RAISE NOTICE '═══════════════════════════════════════════════════════';
    RAISE NOTICE 'KB-0 GCP Migration Complete (001–007)';
    RAISE NOTICE '═══════════════════════════════════════════════════════';
    RAISE NOTICE 'Tables (14): knowledge_items, audit_entries, sla_records,';
    RAISE NOTICE '  notification_queue, workflow_templates, kb1_integration_log,';
    RAISE NOTICE '  l2_extraction_jobs, l2_merged_spans, l2_reviewer_decisions,';
    RAISE NOTICE '  l2_section_passages, l2_guideline_tree, l2_page_decisions,';
    RAISE NOTICE '  l2_revalidation_runs, l2_output_contracts';
    RAISE NOTICE 'Views (4): v_pending_governance, v_kb_metrics,';
    RAISE NOTICE '  v_l2_job_progress, l2_passages_for_l3';
    RAISE NOTICE '═══════════════════════════════════════════════════════';
END $$;
