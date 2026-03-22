-- Phase 4: Lab adapters + ABDM consent + SFTP state tables

-- ABDM consent artifacts (ingestion side — HIU/HIP data exchange)
CREATE TABLE IF NOT EXISTS abdm_consent_artifacts (
    consent_id      TEXT PRIMARY KEY,
    patient_id      TEXT NOT NULL,
    hiu_request_id  TEXT NOT NULL,
    purpose         TEXT NOT NULL,
    date_from       TIMESTAMPTZ,
    date_to         TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL,
    signature       TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'GRANTED',
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_abdm_consent_status ON abdm_consent_artifacts(status);

-- SFTP poll state tracking
CREATE TABLE IF NOT EXISTS sftp_poll_state (
    hospital_id     TEXT PRIMARY KEY,
    last_poll_at    TIMESTAMPTZ,
    last_file       TEXT,
    files_processed INT DEFAULT 0,
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- Lab code registry: per-lab LOINC mapping
CREATE TABLE IF NOT EXISTS lab_code_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lab_id          TEXT NOT NULL,
    lab_code        TEXT NOT NULL,
    loinc_code      TEXT NOT NULL,
    display_name    TEXT NOT NULL,
    unit            TEXT,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now(),
    UNIQUE(lab_id, lab_code)
);

CREATE INDEX idx_lab_code_mappings_lab ON lab_code_mappings(lab_id);
CREATE INDEX idx_lab_code_mappings_loinc ON lab_code_mappings(loinc_code);

-- Seed common lab code mappings for Indian lab partners
INSERT INTO lab_code_mappings (lab_id, lab_code, loinc_code, display_name, unit) VALUES
    ('thyrocare', 'HBA1C', '4548-4', 'Hemoglobin A1c', '%'),
    ('thyrocare', 'FBS', '1558-6', 'Fasting glucose', 'mg/dL'),
    ('thyrocare', 'CREATININE', '2160-0', 'Creatinine', 'mg/dL'),
    ('thyrocare', 'EGFR', '33914-3', 'eGFR', 'mL/min/1.73m2'),
    ('thyrocare', 'CHOLESTEROL', '2093-3', 'Total cholesterol', 'mg/dL'),
    ('thyrocare', 'HDL', '2085-9', 'HDL cholesterol', 'mg/dL'),
    ('thyrocare', 'LDL', '13457-7', 'LDL cholesterol', 'mg/dL'),
    ('thyrocare', 'TRIGLYCERIDES', '2571-8', 'Triglycerides', 'mg/dL'),
    ('thyrocare', 'POTASSIUM', '2823-3', 'Potassium', 'mEq/L'),
    ('thyrocare', 'SODIUM', '2951-2', 'Sodium', 'mEq/L'),
    ('thyrocare', 'URIC_ACID', '3084-1', 'Uric acid', 'mg/dL'),
    ('thyrocare', 'TSH', '3016-3', 'TSH', 'mIU/L')
ON CONFLICT (lab_id, lab_code) DO NOTHING;
