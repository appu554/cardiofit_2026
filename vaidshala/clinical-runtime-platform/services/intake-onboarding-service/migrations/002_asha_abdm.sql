-- Phase 4: ASHA tablet sync + ABDM consent tables

-- ASHA device sync state tracking
CREATE TABLE IF NOT EXISTS asha_device_sync (
    device_id       TEXT PRIMARY KEY,
    asha_id         UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    last_sync_seq   BIGINT DEFAULT 0,
    last_sync_at    TIMESTAMPTZ,
    pending_count   INT DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_asha_device_sync_asha_id ON asha_device_sync(asha_id);

-- ASHA offline queue for conflict resolution
CREATE TABLE IF NOT EXISTS asha_offline_queue (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       TEXT NOT NULL,
    patient_id      UUID NOT NULL,
    tenant_id       UUID NOT NULL,
    sync_seq_no     BIGINT NOT NULL,
    slot_name       TEXT NOT NULL,
    slot_value      JSONB NOT NULL,
    collected_at    TIMESTAMPTZ NOT NULL,
    status          TEXT NOT NULL DEFAULT 'PENDING', -- PENDING, ACCEPTED, CONFLICT, REJECTED
    conflict_reason TEXT,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_asha_offline_queue_device ON asha_offline_queue(device_id, sync_seq_no);
CREATE INDEX idx_asha_offline_queue_patient ON asha_offline_queue(patient_id);

-- ABDM DPDPA consent records (intake side)
CREATE TABLE IF NOT EXISTS abdm_dpdpa_consent (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id       UUID NOT NULL,
    consent_version  TEXT NOT NULL,
    purpose_of_use   TEXT NOT NULL,
    data_categories  TEXT[] NOT NULL,
    retention_period TEXT NOT NULL,
    granted_at       TIMESTAMPTZ NOT NULL,
    channel          TEXT NOT NULL,
    ip_address       TEXT,
    user_agent       TEXT,
    created_at       TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_abdm_dpdpa_consent_patient ON abdm_dpdpa_consent(patient_id);

-- ABDM consent request tracking (intake side)
CREATE TABLE IF NOT EXISTS abdm_consent_requests (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id        UUID NOT NULL,
    abha_number       TEXT NOT NULL,
    purpose           TEXT NOT NULL,
    hi_types          TEXT[] NOT NULL,
    date_range_from   TIMESTAMPTZ NOT NULL,
    date_range_to     TIMESTAMPTZ NOT NULL,
    expiry_date       TIMESTAMPTZ NOT NULL,
    dpdpa_consent     BOOLEAN NOT NULL DEFAULT false,
    abdm_consent_id   TEXT,
    status            TEXT NOT NULL DEFAULT 'REQUESTED',
    created_at        TIMESTAMPTZ DEFAULT now(),
    updated_at        TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_abdm_consent_requests_patient ON abdm_consent_requests(patient_id);
CREATE INDEX idx_abdm_consent_requests_status ON abdm_consent_requests(status);
