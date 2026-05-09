-- Migration 030: reflective_entries table
-- Persists the ReflectiveEntry entity defined in
-- backend/services/pharmacist-self-visibility/internal/reflection/entries.go (Phase 1b Task 1).
-- See plan: docs/superpowers/plans/2026-05-07-phase-1b-self-visibility-surfaces.md Task 1
--
-- Visibility class: POA (Pharmacist-Only Access).
-- Only the authoring pharmacist can read their own entries.
-- Pattern detection is explicitly forbidden on this entity per
-- Self-Visibility Guidelines §6.4 (safe-space character of reflective writing).

BEGIN;

CREATE TABLE reflective_entries (
    id              UUID PRIMARY KEY,
    pharmacist_id   UUID        NOT NULL,
    prompt_id       UUID,                        -- nullable: free-form entry has no prompt
    body            TEXT        NOT NULL CHECK (body <> ''),
    tags            TEXT[]      NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Primary lookup: all entries by a given pharmacist (author-only read, POA class).
CREATE INDEX idx_reflective_entries_pharmacist ON reflective_entries (pharmacist_id, created_at DESC);

COMMENT ON TABLE reflective_entries IS
    'POA-class entity: only the authoring pharmacist may read. '
    'Pattern detection and employer aggregation are forbidden on this table.';

COMMIT;
