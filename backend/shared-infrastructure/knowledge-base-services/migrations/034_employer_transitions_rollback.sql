-- 034_employer_transitions_rollback.sql
-- Rollback for 034_employer_transitions.sql
BEGIN;

DROP TABLE IF EXISTS employer_transitions;

COMMIT;
