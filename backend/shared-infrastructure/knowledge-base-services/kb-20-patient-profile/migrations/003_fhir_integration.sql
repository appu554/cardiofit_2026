-- KB-20 FHIR Store Integration
-- Adds FHIR reference columns to existing tables and creates sync log.

-- ============================================================
-- Patient Profiles — FHIR Patient reference
-- ============================================================
ALTER TABLE patient_profiles ADD COLUMN IF NOT EXISTS fhir_patient_id VARCHAR(200);
CREATE INDEX IF NOT EXISTS idx_patient_fhir_id ON patient_profiles(fhir_patient_id) WHERE fhir_patient_id IS NOT NULL;

-- ============================================================
-- Lab Entries — LOINC code + FHIR Observation reference
-- ============================================================
ALTER TABLE lab_entries ADD COLUMN IF NOT EXISTS loinc_code VARCHAR(20);
ALTER TABLE lab_entries ADD COLUMN IF NOT EXISTS fhir_observation_id VARCHAR(200);
CREATE INDEX IF NOT EXISTS idx_lab_fhir_obs_id ON lab_entries(fhir_observation_id) WHERE fhir_observation_id IS NOT NULL;

-- ============================================================
-- Medication State — ATC code + FHIR MedicationRequest reference
-- ============================================================
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS fhir_medication_request_id VARCHAR(200);
ALTER TABLE medication_states ADD COLUMN IF NOT EXISTS atc_code VARCHAR(20);
CREATE INDEX IF NOT EXISTS idx_med_fhir_req_id ON medication_states(fhir_medication_request_id) WHERE fhir_medication_request_id IS NOT NULL;

-- ============================================================
-- FHIR Sync Log — audit trail for sync operations
-- ============================================================
CREATE TABLE IF NOT EXISTS fhir_sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type VARCHAR(50) NOT NULL,
    fhir_id VARCHAR(200) NOT NULL,
    action VARCHAR(20) NOT NULL CHECK (action IN ('CREATED','UPDATED','SKIPPED')),
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_fhir_sync_resource ON fhir_sync_logs(resource_type, synced_at DESC);
CREATE INDEX IF NOT EXISTS idx_fhir_sync_fhir_id ON fhir_sync_logs(fhir_id);
