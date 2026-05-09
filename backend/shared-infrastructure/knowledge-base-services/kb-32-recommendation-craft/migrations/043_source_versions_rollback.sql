-- Rollback for migration 043: source_versions + recommendation_citations
--
-- Drops both tables added in 043. Recommendation citations are audit-critical
-- records — ensure clinical audit data is preserved before rolling back in
-- production environments (e.g. export to cold storage prior to DROP).

BEGIN;

DROP TABLE IF EXISTS recommendation_citations;
DROP TABLE IF EXISTS source_versions;

COMMIT;
