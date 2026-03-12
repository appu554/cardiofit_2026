-- ============================================================================
-- WIDEN row_coverage_pct COLUMN
-- ============================================================================
-- Version: 10.0.0
-- Description: row_coverage_pct can exceed 999.99% because ExtractedRows
--              counts ALL facts (table + prose + LLM), not just table rows.
--              A drug with 20 source rows producing 210 facts = 1050%.
--              Widen from NUMERIC(5,2) → NUMERIC(7,2) to support up to 99999.99%.
-- ============================================================================

ALTER TABLE completeness_reports
    ALTER COLUMN row_coverage_pct TYPE NUMERIC(7,2);

COMMENT ON COLUMN completeness_reports.row_coverage_pct IS 'Facts/source-rows ratio as percentage — can exceed 100% when prose/LLM extraction supplements table parsing';
