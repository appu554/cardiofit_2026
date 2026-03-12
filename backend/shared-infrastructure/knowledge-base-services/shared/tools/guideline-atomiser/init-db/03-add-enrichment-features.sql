-- Migration 003: Add enrichment features JSONB column to l2_merged_spans
-- Stores computed ML feature vectors for the golden dataset enrichment pipeline.
-- Used by: enrich_golden_dataset.py, nightly_enrichment.py

ALTER TABLE l2_merged_spans
    ADD COLUMN IF NOT EXISTS enrichment_features JSONB DEFAULT NULL;

COMMENT ON COLUMN l2_merged_spans.enrichment_features IS
    'ML feature vector computed by enrich_span_features(). Populated by nightly enrichment batch.';

-- Index for querying spans that need enrichment (NULL = not yet enriched)
CREATE INDEX IF NOT EXISTS idx_l2_merged_spans_enrichment_null
    ON l2_merged_spans (job_id)
    WHERE enrichment_features IS NULL;
