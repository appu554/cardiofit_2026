-- =============================================================================
-- Migration 006: CoverageGuard Sprint 1 + Sprint 2 columns
-- =============================================================================
-- Adds four columns to l2_merged_spans to support:
--   1. tier           — Risk tier (1=critical, 2=warning, 3=info) from CoverageGuard
--   2. coverage_guard_alert — JSONB alert payload (numeric_mismatch, branch_loss, etc.)
--   3. semantic_tokens — JSONB semantic highlighting tokens (numerics, conditions, negations)
--
-- Adds one column to l2_reviewer_decisions for auditable reject reasons:
--   4. reject_reason  — Structured category (not_in_source, numeric_mismatch, etc.)
--
-- Adds source_pdf_path to l2_extraction_jobs for Phase 5 job completion flow.
-- =============================================================================

-- ---- l2_merged_spans: CoverageGuard analysis output ----

-- Risk tier assigned by CoverageGuard: 1 = critical, 2 = warning, 3 = info.
-- NULL for spans ingested before CoverageGuard was enabled.
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS tier SMALLINT
    CHECK (tier IS NULL OR tier IN (1, 2, 3));

-- CoverageGuard alert payload. Structure:
-- { "type": "numeric_mismatch"|"branch_loss"|"llm_only"|"negation_flip",
--   "label": "...", "detail": "...", "alertSeverity": "critical"|"warning"|"info",
--   "sourceValue?": "...", "extractedValue?": "...",
--   "sourceThresholds?": N, "extractedThresholds?": N }
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS coverage_guard_alert JSONB;

-- Semantic highlighting tokens for the reviewer UI. Structure:
-- { "numerics": ["≥30", "eGFR"], "conditions": ["if", "when"], "negations": ["not", "avoid"] }
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS semantic_tokens JSONB;

-- ---- l2_reviewer_decisions: structured reject reason ----

-- Auditable rejection category. Must be one of the enumerated values.
-- NULL for non-reject actions (CONFIRM, EDIT, ADD).
ALTER TABLE l2_reviewer_decisions ADD COLUMN IF NOT EXISTS reject_reason TEXT
    CHECK (reject_reason IS NULL OR reject_reason IN (
        'not_in_source', 'numeric_mismatch', 'negation_error', 'out_of_scope',
        'duplicate', 'hallucination', 'branch_incomplete', 'other'
    ));

-- ---- l2_extraction_jobs: source PDF path (for job completion) ----
-- Already existed in some deployments but not in the migration chain.
ALTER TABLE l2_extraction_jobs ADD COLUMN IF NOT EXISTS source_pdf_path TEXT;

-- ---- l2_extraction_jobs: reviewer who completed the job ----
ALTER TABLE l2_extraction_jobs ADD COLUMN IF NOT EXISTS completed_by TEXT;

-- =============================================================================
-- COMPLETION MESSAGE
-- =============================================================================
DO $$
BEGIN
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Migration 006: CoverageGuard Sprint 1+2 Applied';
    RAISE NOTICE '  l2_merged_spans: +tier, +coverage_guard_alert, +semantic_tokens';
    RAISE NOTICE '  l2_reviewer_decisions: +reject_reason';
    RAISE NOTICE '  l2_extraction_jobs: +source_pdf_path, +completed_by';
    RAISE NOTICE '===================================================';
END $$;
