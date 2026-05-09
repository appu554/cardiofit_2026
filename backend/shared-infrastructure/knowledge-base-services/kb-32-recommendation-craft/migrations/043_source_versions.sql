-- Migration 043: source_versions + recommendation_citations tables
--
-- Adds the persistence layer for citation source versioning introduced in
-- internal/citations/ (Phase 2b Tasks 5 + 6). The core audit-defensibility
-- guarantee: source amendments after recommendation fire time do NOT
-- retroactively invalidate already-fired recommendations. Every citation
-- is an immutable fire-time pin.
--
-- Depends on:
--   - (no prior kb-32 table dependency for source_versions)
--   - recommendation_citations has a soft dependency on a recommendations
--     table existing for referential integrity in application logic, but
--     the FK here is only to source_versions to keep migrations self-contained.
--
-- Rollback: migrations/043_source_versions_rollback.sql

BEGIN;

-- ---------------------------------------------------------------------------
-- Table: source_versions
-- ---------------------------------------------------------------------------
-- Each row is a point-in-time snapshot of an evidence source.
-- The closed-open interval [effective_from, effective_to) defines authority.
-- A NULL effective_to means the version is currently open (no planned expiry).

CREATE TABLE source_versions (
    source_id      TEXT        NOT NULL,
    version        TEXT        NOT NULL,
    effective_from TIMESTAMPTZ NOT NULL,
    effective_to   TIMESTAMPTZ,
    content_hash   TEXT        NOT NULL,
    status         TEXT        NOT NULL
                               CHECK (status IN (
                                   'active',
                                   'amended',
                                   'retracted',
                                   'superseded'
                               )),
    PRIMARY KEY (source_id, version)
);

-- Support time-range lookups for ActiveVersion queries (effective_from lookups).
CREATE INDEX idx_sv_source_effective ON source_versions (source_id, effective_from);

-- ---------------------------------------------------------------------------
-- Table: recommendation_citations
-- ---------------------------------------------------------------------------
-- Immutable fire-time pins: each row records the exact source_version that
-- was active when the recommendation was generated.
--
-- The FK into source_versions ensures referential integrity: you cannot pin
-- a citation to a version that does not exist.
--
-- NOTE: Once inserted, rows in this table must NEVER be updated or deleted
-- (only soft-reads via source_versions.status surface retraction/amendment
-- state to callers). This is the foundation of audit defensibility.

CREATE TABLE recommendation_citations (
    recommendation_id UUID        NOT NULL,
    source_id         TEXT        NOT NULL,
    version           TEXT        NOT NULL,
    pinned_at         TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (recommendation_id, source_id, version),
    FOREIGN KEY (source_id, version)
        REFERENCES source_versions (source_id, version)
);

-- Support ListCitations(recommendationID) queries.
CREATE INDEX idx_rc_recommendation ON recommendation_citations (recommendation_id);

-- Support source-lineage queries: which recommendations cite a given version?
CREATE INDEX idx_rc_source ON recommendation_citations (source_id, version);

COMMIT;
