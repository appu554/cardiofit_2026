-- Migration 009: add provenance_v5 JSONB column to l2_merged_spans
-- V5 Bbox Provenance subsystem: per-channel attribution + bbox data
-- Safe to apply repeatedly (IF NOT EXISTS guards).

ALTER TABLE l2_merged_spans
    ADD COLUMN IF NOT EXISTS provenance_v5 JSONB DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_l2_merged_spans_provenance_v5
    ON l2_merged_spans USING gin (provenance_v5)
    WHERE provenance_v5 IS NOT NULL;

COMMENT ON COLUMN l2_merged_spans.provenance_v5 IS
    'V5 Bbox Provenance: list of ChannelProvenance dicts [{channel_id, bbox, page_number, confidence, model_version, notes}]. NULL when V5 flag off.';
