-- ============================================================================
-- Migration 021 — Hospital discharge reconciliation tables (Wave 4 of
-- Layer 2 substrate plan; Layer 2 doc §3.2).
--
-- Four greenfield tables + one CHECK extension to events:
--
--   1. discharge_documents          — ingested PDF / MHR-CDA / manual
--      discharge documents with raw_text + structured_payload.
--   2. discharge_medication_lines   — per-line parsed medication rows
--      with AMT code, dose, frequency, route, indication, notes.
--   3. reconciliation_worklists     — one row per discharge document
--      with status lifecycle (pending → in_progress → completed |
--      abandoned), assigned ACOP role, due window.
--   4. reconciliation_decisions     — one row per non-unchanged diff;
--      records ACOP decision (accept | modify | reject | defer),
--      intent class, and links to the resulting MedicineUse change +
--      EvidenceTrace audit node.
--   5. events.event_type CHECK extended to admit
--      reconciliation_completed (system bucket).
--
-- Foreign-key policy: identical to migration 009 — no DB-level FKs to
-- residents / events / medicine_uses / roles. Cross-table integrity is
-- enforced at the application boundary so the schema stays cross-DB
-- safe and refactor-friendly.
-- ============================================================================

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. discharge_documents -----------------------------------------------------
CREATE TABLE IF NOT EXISTS discharge_documents (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_ref                UUID NOT NULL,
    source                      TEXT NOT NULL CHECK (source IN ('pdf','mhr_cda','manual')),
    document_id                 TEXT,
    discharge_date              TIMESTAMPTZ NOT NULL,
    discharging_facility_name   TEXT,
    raw_text                    TEXT,
    structured_payload          JSONB,
    ingested_at                 TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Idempotency guard: re-ingest of the same external doc id from the
-- same source returns conflict. document_id is nullable so multiple
-- manual / source-without-id rows coexist.
CREATE UNIQUE INDEX IF NOT EXISTS ux_discharge_docs_source_docid
    ON discharge_documents (source, document_id)
    WHERE document_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_discharge_docs_resident
    ON discharge_documents (resident_ref, discharge_date DESC);

COMMENT ON TABLE discharge_documents IS
    'Hospital discharge documents (PDF / MHR-CDA / manual) ingested for reconciliation. Wave 4.1.';
COMMENT ON COLUMN discharge_documents.raw_text IS
    'OCR/parser output. Empty for structured-only sources where no narrative is captured.';
COMMENT ON COLUMN discharge_documents.structured_payload IS
    'Parser-specific structured fields (PDF metadata, CDA root attributes). Opaque to the substrate.';

-- 2. discharge_medication_lines ---------------------------------------------
CREATE TABLE IF NOT EXISTS discharge_medication_lines (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discharge_document_ref      UUID NOT NULL REFERENCES discharge_documents(id) ON DELETE CASCADE,
    line_number                 INTEGER NOT NULL,
    medication_name_raw         TEXT NOT NULL,
    amt_code                    TEXT,
    dose_raw                    TEXT,
    frequency_raw               TEXT,
    route_raw                   TEXT,
    indication_text             TEXT,
    notes                       TEXT,
    UNIQUE (discharge_document_ref, line_number)
);

CREATE INDEX IF NOT EXISTS idx_discharge_med_lines_doc
    ON discharge_medication_lines (discharge_document_ref);

COMMENT ON TABLE discharge_medication_lines IS
    'Parsed per-line discharge medications. Feeds the diff engine + classifier. Wave 4.1.';

-- 3. reconciliation_worklists -----------------------------------------------
CREATE TABLE IF NOT EXISTS reconciliation_worklists (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discharge_document_ref      UUID NOT NULL REFERENCES discharge_documents(id) ON DELETE CASCADE,
    resident_ref                UUID NOT NULL,
    assigned_role_ref           UUID,
    facility_id                 UUID,
    status                      TEXT NOT NULL CHECK (status IN ('pending','in_progress','completed','abandoned')) DEFAULT 'pending',
    due_at                      TIMESTAMPTZ NOT NULL,
    completed_at                TIMESTAMPTZ,
    completed_by_role_ref       UUID,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reconciliation_worklists_assigned
    ON reconciliation_worklists (assigned_role_ref, status, due_at);
CREATE INDEX IF NOT EXISTS idx_reconciliation_worklists_facility
    ON reconciliation_worklists (facility_id, status, due_at);
CREATE INDEX IF NOT EXISTS idx_reconciliation_worklists_resident
    ON reconciliation_worklists (resident_ref, created_at DESC);

COMMENT ON TABLE reconciliation_worklists IS
    'ACOP reconciliation worklist per discharge document. Wave 4.3.';

-- 4. reconciliation_decisions -----------------------------------------------
CREATE TABLE IF NOT EXISTS reconciliation_decisions (
    id                              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worklist_ref                    UUID NOT NULL REFERENCES reconciliation_worklists(id) ON DELETE CASCADE,
    discharge_med_line_ref          UUID REFERENCES discharge_medication_lines(id),
    pre_admission_medicine_use_ref  UUID,
    diff_class                      TEXT NOT NULL CHECK (diff_class IN ('new_medication','ceased_medication','dose_change','unchanged')),
    intent_class                    TEXT NOT NULL CHECK (intent_class IN ('acute_illness_temporary','new_chronic','reconciled_change','unclear')),
    acop_decision                   TEXT NOT NULL CHECK (acop_decision IN ('','accept','modify','reject','defer')) DEFAULT '',
    acop_role_ref                   UUID,
    decided_at                      TIMESTAMPTZ,
    notes                           TEXT,
    resulting_medicine_use_ref      UUID,
    evidence_trace_node_ref         UUID,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reconciliation_decisions_worklist
    ON reconciliation_decisions (worklist_ref);
CREATE INDEX IF NOT EXISTS idx_reconciliation_decisions_evidence
    ON reconciliation_decisions (evidence_trace_node_ref)
    WHERE evidence_trace_node_ref IS NOT NULL;

COMMENT ON TABLE reconciliation_decisions IS
    'One row per non-unchanged diff entry. Records ACOP decision + EvidenceTrace audit node + resulting MedicineUse change. Waves 4.3 + 4.4.';
COMMENT ON COLUMN reconciliation_decisions.acop_decision IS
    'Empty string until the ACOP records a decision. CHECK admits empty for the pending-decision state.';

-- 5. events.event_type CHECK extension --------------------------------------
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_event_type_check;
ALTER TABLE events ADD CONSTRAINT events_event_type_check CHECK (event_type IN (
    -- Clinical
    'fall','pressure_injury','behavioural_incident',
    'medication_error','adverse_drug_event',
    -- Care transitions
    'hospital_admission','hospital_discharge','GP_visit','specialist_visit',
    'emergency_department_presentation','end_of_life_recognition','death',
    'care_intensity_transition',
    -- Administrative
    'admission_to_facility','transfer_between_facilities',
    'care_planning_meeting','family_meeting',
    -- System (for EvidenceTrace)
    'rule_fire','recommendation_submitted','recommendation_decided',
    'monitoring_plan_activated','consent_granted_or_withdrawn',
    'credential_verified_or_expired',
    'concern_expired_unresolved',
    'capacity_change',
    'reconciliation_completed'
));

COMMIT;
