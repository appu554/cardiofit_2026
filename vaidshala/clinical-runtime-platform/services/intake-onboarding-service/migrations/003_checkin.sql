-- Check-in sessions (biweekly M0-CI)
CREATE TABLE checkin_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL REFERENCES enrollments(patient_id),
    encounter_id    UUID NOT NULL,
    cycle_number    INT NOT NULL,
    state           TEXT NOT NULL DEFAULT 'CS1_SCHEDULED',
    trajectory      TEXT,
    slots_filled    INT DEFAULT 0,
    slots_total     INT DEFAULT 12,
    scheduled_at    TIMESTAMPTZ NOT NULL,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now(),
    UNIQUE (patient_id, cycle_number)
);

CREATE TABLE checkin_slot_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES checkin_sessions(id),
    patient_id      UUID NOT NULL,
    slot_name       TEXT NOT NULL,
    domain          TEXT NOT NULL,
    value           JSONB NOT NULL,
    extraction_mode TEXT NOT NULL,
    confidence      REAL,
    fhir_resource_id TEXT,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_checkin_sessions_patient ON checkin_sessions(patient_id, state);
CREATE INDEX idx_checkin_sessions_scheduled ON checkin_sessions(scheduled_at) WHERE state = 'CS1_SCHEDULED';
CREATE INDEX idx_checkin_slot_events_session ON checkin_slot_events(session_id, slot_name);
