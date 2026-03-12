-- =============================================================================
-- Pipeline 1 Page Decisions — Independent page-level reviewer decisions
-- =============================================================================
-- Page decisions are metadata-only and do NOT cascade to span review_status.
-- A reviewer can Accept/Flag/Escalate a page independently of individual spans.
-- Upsert pattern: latest decision per page wins (ON CONFLICT DO UPDATE).
-- =============================================================================

CREATE TABLE l2_page_decisions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id          UUID NOT NULL REFERENCES l2_extraction_jobs(job_id) ON DELETE CASCADE,
    page_number     INTEGER NOT NULL,
    action          TEXT NOT NULL CHECK (action IN ('ACCEPT', 'FLAG', 'ESCALATE')),
    reviewer_id     TEXT NOT NULL,
    note            TEXT,
    decided_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_l2_page_decision UNIQUE (job_id, page_number)
);

CREATE INDEX idx_l2_page_decisions_job ON l2_page_decisions(job_id);
CREATE INDEX idx_l2_page_decisions_job_page ON l2_page_decisions(job_id, page_number);

-- =============================================================================
-- Patched Passages View for L3 Consumption
-- =============================================================================
-- Applies reviewer edits (reviewer_text) and exclusions (REJECTED spans)
-- so that L3/Pipeline 2 always sees the post-review truth.
-- Key columns:
--   prose_text  → rebuilt from surviving spans with COALESCE(reviewer_text, text)
--   patched_at  → MAX(reviewed_at) for staleness detection vs L3 generated_at
-- =============================================================================

CREATE OR REPLACE VIEW l2_passages_for_l3 AS
SELECT
    sp.job_id,
    sp.section_id,
    sp.heading,
    sp.page_number,
    sp.child_section_ids,
    -- Rebuild prose_text from surviving spans with reviewer edits applied
    COALESCE(
        STRING_AGG(
            COALESCE(ms.reviewer_text, ms.text),
            ' ' ORDER BY ms.start_offset
        ),
        sp.prose_text  -- fallback to original if no spans linked
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

-- =============================================================================
-- COMPLETION MESSAGE
-- =============================================================================
DO $$
BEGIN
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Pipeline 1 Page Decisions Migration Applied';
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Table: l2_page_decisions (ACCEPT / FLAG / ESCALATE)';
    RAISE NOTICE 'View:  l2_passages_for_l3 (patched passages for L3)';
    RAISE NOTICE '===================================================';
END $$;
