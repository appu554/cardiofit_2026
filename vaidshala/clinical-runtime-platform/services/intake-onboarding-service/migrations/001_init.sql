-- Intake-Onboarding Service — initial schema
-- PostgreSQL on port 5433 (Docker/KB shared instance)

CREATE TABLE enrollments (
    patient_id      UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    channel_type    TEXT NOT NULL,
    state           TEXT NOT NULL DEFAULT 'CREATED',
    encounter_id    UUID,
    assigned_pharmacist UUID,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE slot_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL REFERENCES enrollments(patient_id),
    slot_name       TEXT NOT NULL,
    domain          TEXT NOT NULL,
    value           JSONB NOT NULL,
    extraction_mode TEXT NOT NULL,
    confidence      REAL,
    safety_result   JSONB,
    source_channel  TEXT NOT NULL,
    fhir_resource_id TEXT,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE VIEW current_slots AS
SELECT DISTINCT ON (patient_id, slot_name)
    patient_id, slot_name, domain, value, extraction_mode,
    confidence, safety_result, fhir_resource_id, created_at
FROM slot_events
ORDER BY patient_id, slot_name, created_at DESC;

CREATE TABLE flow_positions (
    patient_id      UUID NOT NULL REFERENCES enrollments(patient_id),
    flow_type       TEXT NOT NULL,
    current_node    TEXT NOT NULL,
    state           TEXT DEFAULT 'ACTIVE',
    started_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (patient_id, flow_type)
);

CREATE TABLE review_queue (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL REFERENCES enrollments(patient_id),
    encounter_id    UUID NOT NULL,
    risk_stratum    TEXT NOT NULL,
    status          TEXT DEFAULT 'PENDING',
    reviewer_id     UUID,
    reviewed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_slot_events_patient ON slot_events(patient_id, slot_name);
CREATE INDEX idx_enrollments_state ON enrollments(state);
CREATE INDEX idx_review_queue_status ON review_queue(status, risk_stratum);
