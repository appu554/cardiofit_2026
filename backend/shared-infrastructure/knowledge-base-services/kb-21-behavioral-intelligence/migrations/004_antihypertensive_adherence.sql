-- Migration 004: Antihypertensive Adherence (Wave 2, Amendment 4)
-- Adds aggregate HTN adherence tracking for KB-23 card_builder gating.

CREATE TABLE IF NOT EXISTS antihypertensive_adherence_states (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id               TEXT NOT NULL,
    per_class_adherence      JSONB,
    aggregate_score          NUMERIC(5,4) NOT NULL DEFAULT 0,
    aggregate_score_7d       NUMERIC(5,4) NOT NULL DEFAULT 0,
    aggregate_trend          VARCHAR(20) DEFAULT 'STABLE',
    primary_reason           VARCHAR(20) DEFAULT 'UNKNOWN',
    dietary_sodium_estimate  VARCHAR(20),
    salt_reduction_potential  NUMERIC(5,4) DEFAULT 0,
    active_htn_drug_classes  INTEGER DEFAULT 0,
    updated_at               TIMESTAMP NOT NULL DEFAULT NOW(),
    created_at               TIMESTAMP NOT NULL DEFAULT NOW()
);

-- One row per patient
CREATE UNIQUE INDEX IF NOT EXISTS idx_htn_adherence_patient
    ON antihypertensive_adherence_states (patient_id);

-- For cohort analytics queries
CREATE INDEX IF NOT EXISTS idx_htn_adherence_aggregate
    ON antihypertensive_adherence_states (aggregate_score);

CREATE INDEX IF NOT EXISTS idx_htn_adherence_reason
    ON antihypertensive_adherence_states (primary_reason)
    WHERE primary_reason != 'UNKNOWN';

-- Add adherence_reason to interaction_events for HTN barrier tracking
-- (reuses existing barrier_code infrastructure but adds HTN-specific reason)
COMMENT ON TABLE antihypertensive_adherence_states IS
    'Aggregate antihypertensive adherence per patient. Consumed by KB-23 card_builder to gate HYPERTENSION_REVIEW cards. Wave 2, Amendment 4.';
