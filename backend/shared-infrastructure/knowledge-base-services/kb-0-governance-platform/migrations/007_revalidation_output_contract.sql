-- =============================================================================
-- Migration 007: Revalidation Runs + Output Contract Assembly
-- =============================================================================
-- Supports Phase 4 (CoverageGuard re-validation with delta tracking) and
-- Phase 5 (output contract assembly for Pipeline 2 handoff).
--
-- New tables:
--   l2_revalidation_runs   — Tracks each re-validation iteration and its delta
--   l2_output_contracts    — Persists the assembled 5-section output contract
-- =============================================================================

-- =============================================================================
-- 1. REVALIDATION RUNS
-- =============================================================================
-- Each time the reviewer triggers re-validation in Phase 4, we record the
-- iteration number, the verdict (PASS/BLOCK), and a JSONB delta report showing
-- which CoverageGuard alerts were resolved, persisted, or newly introduced.

CREATE TABLE IF NOT EXISTS l2_revalidation_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id),
    iteration       INTEGER NOT NULL CHECK (iteration >= 1),
    verdict         TEXT NOT NULL CHECK (verdict IN ('PASS', 'BLOCK')),

    -- Counts at time of revalidation
    edited_span_count   INTEGER NOT NULL DEFAULT 0,
    rejected_span_count INTEGER NOT NULL DEFAULT 0,
    added_span_count    INTEGER NOT NULL DEFAULT 0,

    -- Delta report: array of {spanId, previousAlert?, currentAlert?, resolved}
    deltas          JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- Metadata
    triggered_by    TEXT,          -- reviewer ID who triggered the run
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One iteration number per job (prevents accidental re-runs)
    UNIQUE (job_id, iteration)
);

CREATE INDEX IF NOT EXISTS idx_l2_revalidation_runs_job
    ON l2_revalidation_runs(job_id);

-- =============================================================================
-- 2. OUTPUT CONTRACTS
-- =============================================================================
-- The 5-section output contract assembled in Phase 5. Persisted so Pipeline 2
-- can fetch it via API without requiring the reviewer UI to be open.
-- Only one active output contract per job (latest wins via upsert).

CREATE TABLE IF NOT EXISTS l2_output_contracts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id),

    -- Section 1: Confirmed facts (CONFIRMED + EDITED spans with audit trail)
    confirmed_facts JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- Section 2: Added facts (reviewer-created spans)
    added_facts     JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- Section 3: Section tree (guideline tree with fact counts)
    section_tree    JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Section 4: Evidence envelope (job metadata, SHA256, review stats)
    evidence_envelope JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Section 5: Rejection log (rejected spans with reasons)
    rejection_log   JSONB NOT NULL DEFAULT '[]'::jsonb,

    -- Metadata
    assembled_by    TEXT NOT NULL,          -- reviewer ID
    assembled_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One output contract per job (upsert on reassembly)
    UNIQUE (job_id)
);

CREATE INDEX IF NOT EXISTS idx_l2_output_contracts_job
    ON l2_output_contracts(job_id);

-- =============================================================================
-- COMPLETION MESSAGE
-- =============================================================================
DO $$
BEGIN
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Migration 007: Revalidation + Output Contract Applied';
    RAISE NOTICE '  l2_revalidation_runs: iteration tracking + delta JSONB';
    RAISE NOTICE '  l2_output_contracts: 5-section Pipeline 2 handoff';
    RAISE NOTICE '===================================================';
END $$;
