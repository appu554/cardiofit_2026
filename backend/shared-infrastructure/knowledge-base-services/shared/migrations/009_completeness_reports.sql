-- ============================================================================
-- COMPLETENESS REPORTS TABLE
-- ============================================================================
-- Version: 9.0.0
-- Description: Persists per-drug quality reports from the P3 Completeness
--              Checker. Previously these reports were only logged to stdout
--              and discarded (pipeline.go:402). Now they're persisted for
--              trend analysis, gate verdicts, and governance dashboards.
-- ============================================================================

CREATE TABLE IF NOT EXISTS completeness_reports (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drug_name       TEXT NOT NULL,
    rxcui           TEXT NOT NULL,

    -- Section coverage
    sections_covered    TEXT[] NOT NULL DEFAULT '{}',
    sections_missing    TEXT[] NOT NULL DEFAULT '{}',
    section_coverage_pct NUMERIC(5,2) NOT NULL DEFAULT 0,

    -- Fact counts
    fact_counts         JSONB NOT NULL DEFAULT '{}',
    total_facts         INTEGER NOT NULL DEFAULT 0,
    fact_types_covered  INTEGER NOT NULL DEFAULT 0,

    -- Quality metrics
    meddra_match_rate   NUMERIC(5,2) NOT NULL DEFAULT 0,
    frequency_cov_rate  NUMERIC(5,2) NOT NULL DEFAULT 0,
    interaction_qual    NUMERIC(5,2) NOT NULL DEFAULT 0,

    -- Row extraction
    total_source_rows   INTEGER NOT NULL DEFAULT 0,
    extracted_rows      INTEGER NOT NULL DEFAULT 0,
    skipped_rows        INTEGER NOT NULL DEFAULT 0,
    row_coverage_pct    NUMERIC(5,2) NOT NULL DEFAULT 0,
    skip_reason_breakdown JSONB NOT NULL DEFAULT '{}',

    -- Method distribution
    structured_count    INTEGER NOT NULL DEFAULT 0,
    llm_count           INTEGER NOT NULL DEFAULT 0,
    grammar_count       INTEGER NOT NULL DEFAULT 0,
    deterministic_pct   NUMERIC(5,2) NOT NULL DEFAULT 0,

    -- Quality assessment
    warnings            TEXT[] NOT NULL DEFAULT '{}',
    grade               CHAR(1) NOT NULL CHECK (grade IN ('A', 'B', 'C', 'D', 'F')),
    gate_verdict        TEXT NOT NULL DEFAULT 'PASS' CHECK (gate_verdict IN ('PASS', 'WARNING', 'BLOCK')),

    -- Timestamps
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for querying by drug
CREATE INDEX IF NOT EXISTS idx_completeness_reports_drug
    ON completeness_reports (drug_name, created_at DESC);

-- Index for querying by grade (find problematic drugs)
CREATE INDEX IF NOT EXISTS idx_completeness_reports_grade
    ON completeness_reports (grade, created_at DESC);

-- Index for gate verdicts (find blocked runs)
CREATE INDEX IF NOT EXISTS idx_completeness_reports_verdict
    ON completeness_reports (gate_verdict) WHERE gate_verdict != 'PASS';

COMMENT ON TABLE completeness_reports IS 'Per-drug quality reports from the SPL FactStore Pipeline completeness checker (P3)';
COMMENT ON COLUMN completeness_reports.gate_verdict IS 'A/B=PASS, C=WARNING, D/F=BLOCK — derived from grade at persistence time';
