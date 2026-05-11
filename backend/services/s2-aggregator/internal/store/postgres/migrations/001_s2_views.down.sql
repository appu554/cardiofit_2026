-- Rollback for migration 001: drops the bare-minimum view cache table.
BEGIN;

DROP TABLE IF EXISTS s2_view_cache;

COMMIT;
