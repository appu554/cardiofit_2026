-- Migration 026 — placeholder
--
-- This migration intentionally performs no schema changes. It exists solely to
-- close a sequence hole in the shared migrations directory between 025
-- (monitoring_lifecycle, Plan 0.3) and 027 (view_permissions, Phase 1a).
--
-- Some migration runners (especially CI/CD enforcers) reject non-contiguous
-- numbering. By adding a no-op transaction here, the sequence becomes
-- contiguous: 023, 024, 025, 026, 027, 028, 029, 030, 031, 032, 033, 034.
--
-- The reservation 026 is unrelated to the kb-5 service-local file
-- `kb-5-drug-interactions/migrations/026_tier3_qt_antidepressant_matrix.sql`,
-- which lives in a different migrations directory.
--
-- Surfaced by the Phase 1a/1b gap analysis (2026-05-09).

BEGIN;
-- intentional no-op
COMMIT;
