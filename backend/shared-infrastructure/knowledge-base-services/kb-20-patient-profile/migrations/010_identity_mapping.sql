-- ============================================================================
-- Migration 010: Identity mapping + manual-review queue (Wave 1R.3)
--
-- Implements two greenfield tables backing the IdentityMatcher per Layer 2
-- doc §3.3:
--
--   identity_mappings        — identifier-kind/value → resident_ref,
--                              with confidence tier, match path, source,
--                              and (for verified-by-human) verifier metadata
--
--   identity_review_queue    — low-confidence and no-match decisions queued
--                              for human verification with full audit
--                              context (incoming identifier JSONB,
--                              candidate list, best candidate + distance,
--                              EvidenceTrace cross-reference)
--
-- Design notes (mirroring Layer 2 §3.3):
--   - identifier_kind is constrained to a closed set so callers can't
--     accidentally write 'IHI' (uppercase) and silently miss 'ihi' on
--     read. The list is open to extension by future migrations.
--   - UNIQUE (identifier_kind, identifier_value, resident_ref) lets a
--     resident hold multiple identifier kinds (which is the norm) but
--     prevents duplicate same-kind mappings; UPSERTs become idempotent.
--   - confidence is constrained to {high,medium,low}. NONE matches are
--     never persisted to identity_mappings — they live exclusively in
--     identity_review_queue until a human resolves them.
--   - match_path stays free-text (no CHECK) so the matcher's MatchPath
--     enum can evolve without DDL churn.
--   - identity_review_queue.confidence is constrained to {low,none} so
--     callers can't accidentally enqueue a HIGH-confidence decision.
--   - status uses a closed set {pending, resolved, rejected}. resolved
--     records carry resolved_resident_ref + resolved_by + resolved_at;
--     rejected captures "no resident here, this is bad data".
--
-- Foreign-key policy: NO DB-level FKs to residents/persons. Validation
-- lives in the application layer (and resident_ref values are sourced
-- from kb-20's own residents_v2 view), matching the cross-DB-safe
-- approach taken by migration 009.
--
-- Plan: docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md §1R.3
-- Spec: Layer2_Implementation_Guidelines.md §3.3
-- Date: 2026-05-06
-- ============================================================================

BEGIN;

-- pgcrypto re-declared defensively for self-contained safety; already
-- enabled by 001.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ============================================================================
-- identity_mappings
-- ============================================================================
CREATE TABLE IF NOT EXISTS identity_mappings (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier_kind    TEXT NOT NULL CHECK (identifier_kind IN (
        'ihi','medicare','dva','facility_internal','hospital_mrn',
        'dispensing_pharmacy','gp_system'
    )),
    identifier_value   TEXT NOT NULL,
    resident_ref       UUID NOT NULL,
    confidence         TEXT NOT NULL CHECK (confidence IN ('high','medium','low')),
    match_path         TEXT NOT NULL,
    source             TEXT NOT NULL,
    verified_by        UUID,
    verified_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (identifier_kind, identifier_value, resident_ref)
);

-- Lookup index: the dominant read path is "given a (kind, value)
-- arriving from a source, which residents are mapped?". Without this
-- the IdentityCandidateLookup goes to seqscan as the table grows.
CREATE INDEX IF NOT EXISTS idx_identity_mappings_lookup
    ON identity_mappings (identifier_kind, identifier_value);

-- Reverse lookup: every identifier mapped to a given resident.
-- Used by the manual-override re-routing path on resolve.
CREATE INDEX IF NOT EXISTS idx_identity_mappings_resident
    ON identity_mappings (resident_ref);

COMMENT ON TABLE identity_mappings IS
    'Identifier (kind, value) -> Resident mappings persisted by the IdentityMatcher. Multiple kinds per resident expected; UNIQUE (kind, value, resident_ref) prevents duplicate mappings.';
COMMENT ON COLUMN identity_mappings.confidence IS
    'high (IHI exact) | medium (Medicare+name+DOB fuzzy) | low (name+DOB+facility fuzzy + reviewer-resolved). NONE matches never appear here — see identity_review_queue.';
COMMENT ON COLUMN identity_mappings.match_path IS
    'Free-text MatchPath identifier matching shared/v2_substrate/identity.MatchPath; intentionally unconstrained so the enum can evolve without DDL churn.';
COMMENT ON COLUMN identity_mappings.verified_by IS
    'NULL when the mapping was auto-accepted (HIGH/MEDIUM); UUID of the reviewer when promoted from identity_review_queue.';

-- ============================================================================
-- identity_review_queue
-- ============================================================================
CREATE TABLE IF NOT EXISTS identity_review_queue (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incoming_identifier      JSONB NOT NULL,
    candidate_resident_refs  UUID[] NOT NULL DEFAULT '{}',
    best_candidate           UUID,
    best_distance            INTEGER,
    match_path               TEXT NOT NULL,
    confidence               TEXT NOT NULL CHECK (confidence IN ('low','none')),
    source                   TEXT NOT NULL,
    status                   TEXT NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending','resolved','rejected')),
    resolved_resident_ref    UUID,
    resolved_by              UUID,
    resolved_at              TIMESTAMPTZ,
    resolution_note          TEXT,
    evidence_trace_node_ref  UUID,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Worklist index: reviewers fetch pending entries newest-first.
CREATE INDEX IF NOT EXISTS idx_identity_review_queue_status
    ON identity_review_queue (status, created_at DESC);

-- Reverse lookup by best_candidate is rare; skip the index until
-- access patterns warrant it.

COMMENT ON TABLE identity_review_queue IS
    'Low-confidence and no-match identity decisions queued for human verification per Layer 2 §3.3. Resolution promotes the entry to identity_mappings and (per the post-hoc re-routing requirement) reassigns subsequent mappings written against the prior best_candidate.';
COMMENT ON COLUMN identity_review_queue.evidence_trace_node_ref IS
    'EvidenceTrace node UUID written when this queue entry was created; lets reviewers trace the decision back to its inputs and reasoning summary.';

COMMIT;
