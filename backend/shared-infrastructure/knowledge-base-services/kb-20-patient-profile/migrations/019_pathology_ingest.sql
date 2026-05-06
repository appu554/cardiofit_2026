-- ============================================================================
-- Migration 019 — pathology_ingest_log idempotency table
-- Layer 2 substrate plan, Wave 3.1/3.2/3.3: a single document arriving twice
-- (e.g. via SOAP/CDA on Monday and via the MHR FHIR Gateway dual-mode replay
-- on Tuesday) MUST produce a single substrate write. The dedupe key is
-- (source, document_id) — both are stable across re-deliveries.
--
-- Schema source-of-truth:
--   docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md (Wave 3 task)
--   docs/adr/2026-05-06-mhr-integration-strategy.md
--
-- IHI is denormalised onto the row so an operator triaging an ingestion
-- failure can scan-by-IHI without joining residents_v2.
-- ============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS pathology_ingest_log (
    source       TEXT NOT NULL CHECK (source IN ('mhr_soap_cda','mhr_fhir_gateway','hl7_oru')),
    document_id  TEXT NOT NULL,
    ihi          TEXT,
    ingested_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status       TEXT NOT NULL CHECK (status IN ('ingested','duplicate_skipped','identity_review','errored')),
    error        TEXT,
    PRIMARY KEY (source, document_id)
);

CREATE INDEX IF NOT EXISTS idx_pathology_ingest_log_ihi
    ON pathology_ingest_log (ihi);
CREATE INDEX IF NOT EXISTS idx_pathology_ingest_log_status_time
    ON pathology_ingest_log (status, ingested_at DESC);

COMMENT ON TABLE pathology_ingest_log IS
    'Idempotency log for pathology document ingestion. PK (source, document_id) ensures the same document re-delivered from the same upstream is dedupe-skipped at the application layer.';
COMMENT ON COLUMN pathology_ingest_log.source IS
    'Origin of the document: mhr_soap_cda | mhr_fhir_gateway | hl7_oru. Distinct sources can carry the same logical document_id without collision.';
COMMENT ON COLUMN pathology_ingest_log.status IS
    'ingested = new write committed; duplicate_skipped = PK collision before write; identity_review = matched no resident, queued for manual review; errored = parse / write failure (see error column).';
COMMENT ON COLUMN pathology_ingest_log.ihi IS
    'Patient IHI from the document header. Denormalised for operator triage; no FK to residents (an ingestion that landed in identity_review by definition has no resident binding yet).';

COMMIT;
