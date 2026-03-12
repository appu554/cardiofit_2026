-- Migration 005: Add bounding box and surrounding context to merged spans.
-- bbox enables pixel-perfect PDF overlay in the reviewer UI (Phase 2).
-- surrounding_context provides adjacent block text for L1_RECOVERY spans
-- so reviewers can assess whether Marker's omission was justified.

-- PDF bounding box as JSONB array [x0, y0, x1, y1] in PDF points.
-- NULL for spans where bbox is not available (Channels B-F until future pipeline work).
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS bbox JSONB;

-- Text of adjacent blocks from the same PDF page, providing context around
-- L1_RECOVERY spans. NULL for non-recovery spans (they use normalized_text offsets).
ALTER TABLE l2_merged_spans ADD COLUMN IF NOT EXISTS surrounding_context TEXT;
