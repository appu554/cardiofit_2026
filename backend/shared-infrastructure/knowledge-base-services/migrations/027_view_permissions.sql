-- Migration 027: view_permissions table
-- Persists the ViewPermission entity defined in
-- shared/v2_substrate/permissions/permission.go (Phase 1a Task 1).
-- See plan: docs/superpowers/plans/2026-05-07-phase-1a-trust-foundation.md Task 2.
--
-- The scope JSONB column stores the full Scope struct:
--   {
--     "class":          1..5 (POA/PDP/PFA/WO/AD);
--                       0 is reserved for VisibilityClassUnset and MUST NEVER appear
--                       in stored data — rejected at the application layer by
--                       Scope.Validate() / ErrMissingVisibilityClass and enforced
--                       here by a CHECK constraint,
--     "resource_types": [...],
--     "gate":           { ... } | null   (non-null only for PFA class = 3)
--   }

BEGIN;

CREATE TABLE view_permissions (
    id                       UUID PRIMARY KEY,
    subject_id               UUID NOT NULL,
    viewer_role_id           UUID NOT NULL,

    -- scope encodes VisibilityClass + optional AggregationGate (see above).
    -- The CHECK below mirrors Scope.Validate(): class must be 1..5.
    scope                    JSONB NOT NULL
                                 CHECK ((scope ->> 'class')::int BETWEEN 1 AND 5),

    granted_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    granted_by_id            UUID NOT NULL,
    expires_at               TIMESTAMPTZ,
    contestation_record_ref  UUID,
    revoked_at               TIMESTAMPTZ,

    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hot path: middleware looks up the active permission for a
-- (subject, viewer_role) pair on every protected read.
-- Partial index restricted to non-revoked rows.
--
-- NOTE: The plan spec included `expires_at > NOW()` in the partial index
-- predicate, but NOW() is non-IMMUTABLE and Postgres rejects it in a
-- partial index expression (same issue documented in the Phase 0.4 commit
-- history — see migration 023 comment on idx_recommendations_review_due).
-- We drop that time predicate here and rely solely on `revoked_at IS NULL`.
-- Expired rows are excluded at query time by the application layer
-- (Middleware.Wrap checks ExpiresAt against the current clock); they are
-- also cheap to sweep with a background job because the index stays small.
CREATE INDEX idx_view_permissions_active ON view_permissions (subject_id, viewer_role_id)
    WHERE revoked_at IS NULL;

-- GIN index for class-filtered queries
-- (e.g. find all PFA permissions for a subject without a full-table scan).
CREATE INDEX idx_view_permissions_class ON view_permissions
    USING GIN ((scope -> 'class'));

COMMIT;
