-- ============================================================================
-- Migration 004: Clinical Source Registry
-- Adapted from V001_clinical_source_registry_7.sql for KB-22 HPI Engine.
-- Provides governance infrastructure for the Clinical Canon Framework:
--   Canon source → abstraction → YAML encoding → calibration
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- TABLE 1: clinical_sources
-- Master reference metadata for all Canon sources
-- ============================================================================
CREATE TABLE IF NOT EXISTS clinical_sources (
    source_id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL CHECK (type IN (
                        'guideline', 'book', 'journal', 'consensus',
                        'position_statement', 'formulary', 'internal_kb'
                    )),
    source_category TEXT NOT NULL CHECK (source_category IN (
                        'canon', 'runtime_kb', 'cohort'
                    )),
    edition         TEXT,
    publication_year INT,
    publisher       TEXT,
    region          TEXT NOT NULL DEFAULT 'global'
                    CHECK (region IN ('global','UK','India','US','Europe','Asia-Pacific')),
    url             TEXT,
    isbn_issn       TEXT,
    license_type    TEXT NOT NULL CHECK (license_type IN (
                        'public', 'licensed', 'subscription', 'proprietary', 'internal'
                    )),
    runtime_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    update_cycle    TEXT,
    next_review_due DATE,
    last_reviewed   DATE,
    reviewed_by     TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE clinical_sources IS
'Master registry of clinical knowledge sources. Canon sources (BMJ, NICE, ADA, etc.) are authoring-time references only. Runtime KBs (v3 KB-1 through KB-19) are internal. Cohort sources are population data from Vaidshala patient interactions.';

-- Seed Canon sources
INSERT INTO clinical_sources (code, name, type, source_category, edition, publication_year, publisher, region, license_type, runtime_allowed, update_cycle) VALUES
('BMJ_2025',     'BMJ Best Practice',                        'guideline',          'canon',      '2025',        2025, 'BMJ Publishing Group',            'UK',     'licensed',     FALSE, 'continuous'),
('NICE_2025',    'NICE Clinical Pathways',                    'guideline',          'canon',      '2025',        2025, 'National Institute for Health and Care Excellence', 'UK', 'public', FALSE, 'continuous'),
('ADA_2025',     'ADA Standards of Care in Diabetes',         'guideline',          'canon',      '2025',        2025, 'American Diabetes Association',    'US',     'subscription', FALSE, 'annual'),
('API_ICP_2024', 'API-ICP Guidelines for Management of T2DM', 'guideline',          'canon',      '2024',        2024, 'Association of Physicians of India','India', 'public',       FALSE, 'every_3_5_years'),
('KDIGO_2024',   'KDIGO Clinical Practice Guidelines',        'guideline',          'canon',      '2024',        2024, 'Kidney Disease Improving Global Outcomes','global','public', FALSE, 'every_3_5_years'),
('McGee_2022',   'Evidence-Based Physical Diagnosis (McGee)',  'book',              'canon',      '5th Edition', 2022, 'Elsevier',                         'global', 'proprietary',  FALSE, 'every_5_10_years'),
('JAMA_RCE',     'JAMA Rational Clinical Examination Series',  'journal',           'canon',      'Ongoing',     2024, 'American Medical Association',     'global', 'subscription', FALSE, 'continuous'),
('InSH_2023',    'Indian Society of Hypertension Position Statement','position_statement','canon','2023',        2023, 'Indian Society of Hypertension',   'India',  'public',       FALSE, 'every_3_5_years'),
-- v3 Runtime KBs
('KB_1',  'KB-1 Drug Rules (Port 8081)',           'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_4',  'KB-4 Patient Safety (Port 8088)',       'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_5',  'KB-5 Drug Interactions (Port 8089)',    'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_7',  'KB-7 Terminology (Port 8087)',          'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_8',  'KB-8 Calculator (Port 8093)',           'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_16', 'KB-16 Lab Interpretation (Port 8096)',  'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous')
ON CONFLICT (code) DO NOTHING;

-- ============================================================================
-- TABLE 2: node_derivations
-- Structural derivation per presentation node — how each node was built
-- ============================================================================
CREATE TABLE IF NOT EXISTS node_derivations (
    derivation_id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id             TEXT NOT NULL,
    node_name           TEXT NOT NULL,
    source_id           UUID NOT NULL REFERENCES clinical_sources(source_id),
    structural_reference TEXT NOT NULL,
    section_reference   TEXT,
    what_was_extracted  TEXT NOT NULL,
    differentials_retained JSONB,
    differentials_added    JSONB,
    differentials_excluded JSONB,
    exclusion_rationale TEXT,
    red_flags_extracted JSONB,
    lr_values_extracted JSONB,
    context_modifiers_derived JSONB,
    questions_informed  JSONB,
    version             INT NOT NULL DEFAULT 1,
    authored_by         TEXT NOT NULL,
    reviewed_by         TEXT,
    review_date         DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(node_id, source_id, version)
);

COMMENT ON TABLE node_derivations IS
'Documents how each Presentation Node was constructed from Canon sources. Every node has multiple rows — one per source consulted.';

CREATE INDEX IF NOT EXISTS idx_node_derivations_node ON node_derivations(node_id);
CREATE INDEX IF NOT EXISTS idx_node_derivations_source ON node_derivations(source_id);

-- ============================================================================
-- TABLE 3: element_attributions
-- Per-element source tracing — every red flag, LR, context modifier traced
-- ============================================================================
CREATE TABLE IF NOT EXISTS element_attributions (
    attribution_id  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id         TEXT NOT NULL,
    element_type    TEXT NOT NULL CHECK (element_type IN (
                        'red_flag', 'differential', 'likelihood_ratio',
                        'context_modifier', 'discriminating_question',
                        'base_prior', 'completion_criteria',
                        'safety_guidance', 'pertinent_negative'
                    )),
    element_key     TEXT NOT NULL,
    element_value   JSONB,
    source_id       UUID NOT NULL REFERENCES clinical_sources(source_id),
    source_category TEXT NOT NULL CHECK (source_category IN (
                        'canon', 'runtime_kb', 'cohort'
                    )),
    citation        TEXT,
    etiologic_archetype TEXT CHECK (etiologic_archetype IN (
                        'vascular','metabolic','medication','structural',
                        'functional','iatrogenic','psychiatric','hematologic',
                        'infectious','neoplastic', NULL
                    )),
    canonical_concept_id TEXT,
    confidence_level TEXT NOT NULL CHECK (confidence_level IN (
                        'literature', 'consensus', 'estimated', 'cohort_calibrated'
                    )),
    population_validation TEXT,
    introduced_version INT NOT NULL DEFAULT 1,
    last_reviewed   DATE,
    reviewed_by     TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE element_attributions IS
'Every clinical decision element in every YAML Differential Tree has a row here. Traces red flags to NICE, LRs to McGee, context modifiers to API-ICP.';

CREATE INDEX IF NOT EXISTS idx_element_attr_node ON element_attributions(node_id);
CREATE INDEX IF NOT EXISTS idx_element_attr_type ON element_attributions(element_type);
CREATE INDEX IF NOT EXISTS idx_element_attr_concept ON element_attributions(canonical_concept_id);
CREATE INDEX IF NOT EXISTS idx_element_attr_source ON element_attributions(source_id);

-- ============================================================================
-- TABLE 4: canonical_concepts
-- Cross-node shared clinical concepts — asked once, reused across nodes
-- ============================================================================
CREATE TABLE IF NOT EXISTS canonical_concepts (
    concept_id      TEXT PRIMARY KEY,
    concept_name    TEXT NOT NULL,
    snomed_code     TEXT,
    icd11_code      TEXT,
    loinc_code      TEXT,
    used_in_nodes   TEXT[] NOT NULL,
    extraction_mode TEXT NOT NULL CHECK (extraction_mode IN (
                        'BUTTON', 'REGEX', 'LLM', 'ASR_BUTTON', 'COMPOSITE'
                    )),
    hindi_question_template TEXT,
    hindi_variants  JSONB,
    expected_pata_nahi_rate DECIMAL(3,2),
    created_version INT NOT NULL DEFAULT 1,
    last_updated    DATE,
    updated_by      TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE canonical_concepts IS
'Clinical concepts shared across multiple Presentation Nodes. Prevents redundant questioning in multi-complaint sessions and ensures consistent Hindi phrasing.';

CREATE INDEX IF NOT EXISTS idx_canon_concepts_nodes ON canonical_concepts USING GIN(used_in_nodes);

-- ============================================================================
-- TABLE 5: calibration_events
-- Full recalibration audit trail — every change to every clinical value
-- ============================================================================
CREATE TABLE IF NOT EXISTS calibration_events (
    event_id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id             TEXT NOT NULL,
    element_type        TEXT NOT NULL CHECK (element_type IN (
                            'likelihood_ratio', 'base_prior', 'context_modifier',
                            'question_rephrase', 'question_replace',
                            'differential_added', 'differential_removed',
                            'red_flag_added', 'red_flag_modified',
                            'completion_criteria', 'prior_floor'
                        )),
    element_key         TEXT NOT NULL,
    trigger_type        TEXT NOT NULL CHECK (trigger_type IN (
                            'scheduled_quarterly', 'red_flag_miss', 'diagnosis_miss',
                            'calibration_drift', 'question_failure', 'low_agreement',
                            'canon_update', 'outcome_validation', 'structural_review'
                        )),
    trigger_layer       INT NOT NULL CHECK (trigger_layer BETWEEN 1 AND 5),
    old_value           JSONB,
    new_value           JSONB,
    evidence_package    JSONB NOT NULL,
    sample_size         INT,
    blend_ratio         TEXT,
    credible_interval   JSONB,
    proposed_by         TEXT NOT NULL,
    reviewed_by         TEXT,
    m8_validation_result TEXT CHECK (m8_validation_result IN (
                            'PASS', 'FAIL', 'PASS_WITH_NOTES', NULL
                        )),
    m8_red_flag_sensitivity DECIMAL(3,2),
    m8_top1_accuracy    DECIMAL(3,2),
    m8_closure_guard_rate DECIMAL(3,2),
    approval_date       DATE,
    deployed_date       DATE,
    version_before      INT NOT NULL,
    version_after       INT NOT NULL,
    rollback_deadline   DATE,
    rolled_back         BOOLEAN NOT NULL DEFAULT FALSE,
    rollback_reason     TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE calibration_events IS
'Complete regulatory defense artifact. Every change to every clinical value is logged with trigger, evidence, reviewers, M8 validation results, and deployment/rollback status.';

CREATE INDEX IF NOT EXISTS idx_calibration_node ON calibration_events(node_id);
CREATE INDEX IF NOT EXISTS idx_calibration_trigger ON calibration_events(trigger_type);
CREATE INDEX IF NOT EXISTS idx_calibration_layer ON calibration_events(trigger_layer);
CREATE INDEX IF NOT EXISTS idx_calibration_date ON calibration_events(deployed_date);

-- ============================================================================
-- TABLE 6: source_conflicts
-- Canon source disagreement resolution — when guidelines contradict
-- ============================================================================
CREATE TABLE IF NOT EXISTS source_conflicts (
    conflict_id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id             TEXT NOT NULL,
    element_key         TEXT NOT NULL,
    sources_in_conflict TEXT[] NOT NULL,
    conflict_description TEXT NOT NULL,
    values_from_sources JSONB NOT NULL,
    resolution          TEXT NOT NULL,
    resolution_rationale TEXT NOT NULL,
    population_factor   BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_by         TEXT NOT NULL,
    reviewed_by         TEXT,
    resolution_date     DATE NOT NULL,
    node_version        INT NOT NULL,
    superseded_by       UUID REFERENCES source_conflicts(conflict_id),
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE source_conflicts IS
'Documents when Canon sources disagree and how conflicts were resolved. Critical for Indian context where API-ICP may contradict global guidelines.';

CREATE INDEX IF NOT EXISTS idx_conflicts_node ON source_conflicts(node_id);
CREATE INDEX IF NOT EXISTS idx_conflicts_sources ON source_conflicts USING GIN(sources_in_conflict);

-- ============================================================================
-- GOVERNANCE VIEWS
-- ============================================================================

CREATE OR REPLACE VIEW v_node_source_coverage AS
SELECT
    nd.node_id,
    nd.node_name,
    cs.code AS source_code,
    cs.name AS source_name,
    cs.source_category,
    nd.what_was_extracted,
    nd.version,
    nd.review_date
FROM node_derivations nd
JOIN clinical_sources cs ON nd.source_id = cs.source_id
ORDER BY nd.node_id, cs.source_category, cs.code;

CREATE OR REPLACE VIEW v_elements_pending_validation AS
SELECT
    ea.node_id,
    ea.element_type,
    ea.element_key,
    ea.confidence_level,
    cs.code AS source_code,
    ea.introduced_version,
    ea.last_reviewed
FROM element_attributions ea
JOIN clinical_sources cs ON ea.source_id = cs.source_id
WHERE ea.confidence_level IN ('estimated', 'consensus')
  AND ea.population_validation IS NULL
ORDER BY ea.node_id, ea.element_type;

CREATE OR REPLACE VIEW v_sources_due_for_review AS
SELECT
    code, name, update_cycle, next_review_due, last_reviewed, reviewed_by,
    CASE
        WHEN next_review_due < CURRENT_DATE THEN 'OVERDUE'
        WHEN next_review_due < CURRENT_DATE + INTERVAL '30 days' THEN 'DUE_SOON'
        ELSE 'OK'
    END AS review_status
FROM clinical_sources
WHERE source_category = 'canon'
ORDER BY next_review_due NULLS FIRST;

CREATE OR REPLACE VIEW v_calibration_timeline AS
SELECT
    ce.node_id, ce.element_key, ce.trigger_type, ce.trigger_layer,
    ce.old_value, ce.new_value, ce.sample_size,
    ce.m8_red_flag_sensitivity, ce.m8_top1_accuracy,
    ce.proposed_by, ce.reviewed_by, ce.deployed_date,
    ce.rolled_back, ce.version_before, ce.version_after
FROM calibration_events ce
ORDER BY ce.node_id, ce.deployed_date;

CREATE OR REPLACE VIEW v_active_conflicts AS
SELECT
    sc.node_id, sc.element_key, sc.sources_in_conflict,
    sc.conflict_description, sc.resolution,
    sc.resolution_date, sc.resolved_by
FROM source_conflicts sc
WHERE sc.superseded_by IS NULL
ORDER BY sc.node_id;

-- ============================================================================
-- TRIGGERS: Auto-update updated_at timestamps
-- ============================================================================
CREATE OR REPLACE FUNCTION csr_update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sources_updated
    BEFORE UPDATE ON clinical_sources
    FOR EACH ROW EXECUTE FUNCTION csr_update_timestamp();

CREATE TRIGGER trg_derivations_updated
    BEFORE UPDATE ON node_derivations
    FOR EACH ROW EXECUTE FUNCTION csr_update_timestamp();

CREATE TRIGGER trg_attributions_updated
    BEFORE UPDATE ON element_attributions
    FOR EACH ROW EXECUTE FUNCTION csr_update_timestamp();

CREATE TRIGGER trg_concepts_updated
    BEFORE UPDATE ON canonical_concepts
    FOR EACH ROW EXECUTE FUNCTION csr_update_timestamp();

-- ============================================================================
-- VALIDATION CONSTRAINTS
-- ============================================================================

ALTER TABLE calibration_events
    ADD CONSTRAINT chk_version_increment
    CHECK (version_after > version_before);

ALTER TABLE calibration_events
    ADD CONSTRAINT chk_red_flag_sensitivity
    CHECK (
        m8_validation_result IS NULL
        OR m8_red_flag_sensitivity IS NULL
        OR m8_red_flag_sensitivity = 1.00
    );

ALTER TABLE clinical_sources
    ADD CONSTRAINT chk_canon_not_runtime
    CHECK (
        source_category != 'canon' OR runtime_allowed = FALSE
    );

-- ============================================================================
-- END OF MIGRATION 004
-- ============================================================================
