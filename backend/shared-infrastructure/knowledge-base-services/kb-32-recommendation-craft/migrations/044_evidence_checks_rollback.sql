-- Rollback for migration 044: evidence_checks table
--
-- Drops the evidence_checks table and its indexes introduced in
-- migrations/044_evidence_checks.sql.

DROP TABLE IF EXISTS evidence_checks;
