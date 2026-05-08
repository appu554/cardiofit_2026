-- Migration 028: contestation table
-- Persists the Contestation entity defined in
-- shared/v2_substrate/contestation/contestation.go (Phase 1a Task 7).
-- See plan: docs/superpowers/plans/2026-05-07-phase-1a-trust-foundation.md Task 7.
--
-- Per v3 §9 line 514: a pharmacist may file a contestation against an algorithmic
-- KPI that feeds any employment-affecting decision. The record is visible to both
-- pharmacist (subject) and employer (challenged party) per the dual-disclosure
-- principle. The algorithmic determination cannot be the sole basis for an adverse
-- employment decision while a contestation is open.
--
-- kpi_snapshot is stored as JSONB to preserve the full KPI state at filing time.

BEGIN;

CREATE TABLE contestations (
    id                   UUID PRIMARY KEY,
    pharmacist_id        UUID NOT NULL,
    employer_id          UUID NOT NULL,
    kpi_type             TEXT NOT NULL,
    kpi_snapshot         JSONB NOT NULL,
    pharmacist_argument  TEXT NOT NULL,
    employer_response    TEXT,
    status               TEXT NOT NULL CHECK (status IN ('open','responded','resolved','withdrawn')),
    filed_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hot path: list all open contestations for a pharmacist (subject self-view
-- and employer HR review both query by pharmacist_id).
CREATE INDEX idx_contestations_pharmacist ON contestations (pharmacist_id);

-- Supports regulator / compliance queries by KPI type + status
-- (e.g. "all open dispensing_accuracy contestations this quarter").
CREATE INDEX idx_contestations_kpi        ON contestations (kpi_type, status);

COMMIT;
