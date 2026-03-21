-- Ingestion Service — initial schema
-- PostgreSQL on port 5433 (Docker/KB shared instance)

CREATE TABLE lab_code_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lab_id          TEXT NOT NULL,
    lab_code        TEXT NOT NULL,
    loinc_code      TEXT NOT NULL,
    display_name    TEXT,
    unit            TEXT,
    created_at      TIMESTAMPTZ DEFAULT now(),
    UNIQUE (lab_id, lab_code)
);

CREATE TABLE dlq_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    error_class     TEXT NOT NULL,
    source_type     TEXT NOT NULL,
    source_id       TEXT,
    raw_payload     BYTEA NOT NULL,
    error_message   TEXT,
    retry_count     INT DEFAULT 0,
    status          TEXT DEFAULT 'PENDING',
    created_at      TIMESTAMPTZ DEFAULT now(),
    resolved_at     TIMESTAMPTZ
);

CREATE TABLE patient_pending_queue (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    identifier_type TEXT NOT NULL,
    identifier_value TEXT NOT NULL,
    raw_payload     JSONB NOT NULL,
    source_type     TEXT NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    resolved_at     TIMESTAMPTZ,
    patient_id      UUID,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_dlq_status ON dlq_messages(status);
CREATE INDEX idx_pending_expires ON patient_pending_queue(expires_at) WHERE resolved_at IS NULL;
