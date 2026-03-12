-- ============================================================================
-- VAIDSHALA CLINICAL SOURCE REGISTRY
-- Migration V001: Core Schema
-- Date: 2026-02-22
-- Purpose: Governance infrastructure for Clinical Canon Framework
--          Tracks provenance of every clinical decision element from
--          Canon source → abstraction → YAML encoding → calibration
-- ============================================================================

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- TABLE 1: clinical_sources
-- Master reference metadata for all Canon sources
-- ============================================================================
CREATE TABLE clinical_sources (
    source_id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            TEXT NOT NULL UNIQUE,          -- e.g. 'BMJ_2025', 'ADA_2025'
    name            TEXT NOT NULL,                 -- 'BMJ Best Practice'
    type            TEXT NOT NULL CHECK (type IN (
                        'guideline',               -- clinical practice guideline
                        'book',                    -- textbook (McGee)
                        'journal',                 -- journal article / systematic review
                        'consensus',               -- expert consensus statement
                        'position_statement',      -- professional body position
                        'formulary',               -- drug reference
                        'internal_kb'              -- v3 Knowledge Base (runtime)
                    )),
    source_category TEXT NOT NULL CHECK (source_category IN (
                        'canon',                   -- external guideline, authoring-time only
                        'runtime_kb',              -- v3 internal KB, called at runtime
                        'cohort'                   -- population data, feeds recalibration
                    )),
    edition         TEXT,                          -- '2025', '4th Edition'
    publication_year INT,
    publisher       TEXT,                          -- 'American Diabetes Association'
    region          TEXT NOT NULL DEFAULT 'global' -- 'global', 'UK', 'India', 'US'
                    CHECK (region IN ('global','UK','India','US','Europe','Asia-Pacific')),
    url             TEXT,
    isbn_issn       TEXT,
    license_type    TEXT NOT NULL CHECK (license_type IN (
                        'public',                  -- freely accessible
                        'licensed',                -- institutional license required
                        'subscription',            -- journal subscription
                        'proprietary',             -- commercial product
                        'internal'                 -- Vaidshala-owned
                    )),
    runtime_allowed BOOLEAN NOT NULL DEFAULT FALSE, -- TRUE only for v3 KBs
    update_cycle    TEXT,                          -- 'annual', 'every_3_5_years', 'continuous'
    next_review_due DATE,
    last_reviewed   DATE,
    reviewed_by     TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE clinical_sources IS
'Master registry of all clinical knowledge sources. Canon sources (BMJ, NICE, ADA, etc.) are authoring-time references only — never called at runtime. Runtime KBs (v3 KB-1 through KB-19) are internal, version-controlled, and called at runtime. Cohort sources are population data from Vaidshala patient interactions.';

-- Seed Canon sources
INSERT INTO clinical_sources (code, name, type, source_category, edition, publication_year, publisher, region, license_type, runtime_allowed, update_cycle) VALUES
('BMJ_2025',     'BMJ Best Practice',                       'guideline',          'canon',      '2025',        2025, 'BMJ Publishing Group',            'UK',     'licensed',     FALSE, 'continuous'),
('NICE_2025',    'NICE Clinical Pathways',                   'guideline',          'canon',      '2025',        2025, 'National Institute for Health and Care Excellence', 'UK', 'public', FALSE, 'continuous'),
('ADA_2025',     'ADA Standards of Care in Diabetes',        'guideline',          'canon',      '2025',        2025, 'American Diabetes Association',    'US',     'subscription', FALSE, 'annual'),
('API_ICP_2024', 'API-ICP Guidelines for Management of T2DM','guideline',          'canon',      '2024',        2024, 'Association of Physicians of India','India', 'public',       FALSE, 'every_3_5_years'),
('KDIGO_2024',   'KDIGO Clinical Practice Guidelines',       'guideline',          'canon',      '2024',        2024, 'Kidney Disease Improving Global Outcomes','global','public', FALSE, 'every_3_5_years'),
('McGee_2022',   'Evidence-Based Physical Diagnosis (McGee)', 'book',              'canon',      '5th Edition', 2022, 'Elsevier',                         'global', 'proprietary',  FALSE, 'every_5_10_years'),
('JAMA_RCE',     'JAMA Rational Clinical Examination Series', 'journal',           'canon',      'Ongoing',     2024, 'American Medical Association',     'global', 'subscription', FALSE, 'continuous'),
('InSH_2023',    'Indian Society of Hypertension Position Statement','position_statement','canon','2023',        2023, 'Indian Society of Hypertension',   'India',  'public',       FALSE, 'every_3_5_years'),
-- v3 Runtime KBs
('KB_1',  'KB-1 Drug Rules (Port 8081)',           'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_4',  'KB-4 Patient Safety (Port 8088)',       'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_5',  'KB-5 Drug Interactions (Port 8089)',    'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_7',  'KB-7 Terminology (Port 8087)',          'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_8',  'KB-8 Calculator (Port 8093)',           'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous'),
('KB_16', 'KB-16 Lab Interpretation (Port 8096)',  'formulary',  'runtime_kb', 'v3', 2026, 'Vaidshala', 'global', 'internal', TRUE,  'continuous');


-- ============================================================================
-- TABLE 2: node_derivations
-- Structural derivation per presentation node — how each node was built
-- ============================================================================
CREATE TABLE node_derivations (
    derivation_id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id             TEXT NOT NULL,              -- 'P01', 'P02', ..., 'P10'
    node_name           TEXT NOT NULL,              -- 'Dizziness', 'Hypoglycemia Symptoms'
    source_id           UUID NOT NULL REFERENCES clinical_sources(source_id),
    structural_reference TEXT NOT NULL,             -- 'BMJ: Approach to Dizziness'
    section_reference   TEXT,                       -- specific chapter/section
    what_was_extracted  TEXT NOT NULL,              -- 'Differential skeleton', 'Red flag list'
    differentials_retained JSONB,                   -- ["DX01","DX04","DX05"] from this source
    differentials_added    JSONB,                   -- ["DX01b","DX02"] cohort-specific additions
    differentials_excluded JSONB,                   -- ["Multiple sclerosis","Acoustic neuroma"]
    exclusion_rationale TEXT,                       -- why excluded from this cohort
    red_flags_extracted JSONB,                      -- red flags taken from this source
    lr_values_extracted JSONB,                      -- LR values taken from this source
    context_modifiers_derived JSONB,                -- context modifiers derived from this source
    questions_informed  JSONB,                      -- questions whose design was informed
    version             INT NOT NULL DEFAULT 1,
    authored_by         TEXT NOT NULL,
    reviewed_by         TEXT,
    review_date         DATE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(node_id, source_id, version)
);

COMMENT ON TABLE node_derivations IS
'Documents how each Presentation Node was constructed from Canon sources. Every node has multiple rows — one per source consulted. Provides traceability: "P01 Dizziness differential skeleton derived from BMJ, 5 differentials retained, 3 medication-specific added, 2 rare diagnoses excluded with rationale."';

CREATE INDEX idx_node_derivations_node ON node_derivations(node_id);
CREATE INDEX idx_node_derivations_source ON node_derivations(source_id);


-- ============================================================================
-- TABLE 3: element_attributions
-- Per-element source tracing — every red flag, LR, context modifier traced
-- ============================================================================
CREATE TABLE element_attributions (
    attribution_id  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id         TEXT NOT NULL,                  -- 'P01'
    element_type    TEXT NOT NULL CHECK (element_type IN (
                        'red_flag',
                        'differential',
                        'likelihood_ratio',
                        'context_modifier',
                        'discriminating_question',
                        'base_prior',
                        'completion_criteria',
                        'safety_guidance',
                        'pertinent_negative'
                    )),
    element_key     TEXT NOT NULL,                  -- 'RF01_focal_deficit', 'DX01_orthostatic'
    element_value   JSONB,                          -- the actual value (LR number, prior, etc.)
    source_id       UUID NOT NULL REFERENCES clinical_sources(source_id),
    source_category TEXT NOT NULL CHECK (source_category IN (
                        'canon',                    -- external guideline
                        'runtime_kb',               -- v3 internal KB
                        'cohort'                    -- population data
                    )),
    citation        TEXT,                           -- 'McGee Ch.19, Table 19-3'
    etiologic_archetype TEXT CHECK (etiologic_archetype IN (
                        'vascular','metabolic','medication','structural',
                        'functional','iatrogenic','psychiatric','hematologic',
                        'infectious','neoplastic', NULL
                    )),
    canonical_concept_id TEXT,                      -- cross-node shared concept reference
    confidence_level TEXT NOT NULL CHECK (confidence_level IN (
                        'literature',               -- published study with defined population
                        'consensus',                -- 2+ clinicians agreed
                        'estimated',                -- single clinician estimate
                        'cohort_calibrated'          -- validated against population data
                    )),
    population_validation TEXT,                     -- 'Cohort: 89 patients, observed 0.34'
    introduced_version INT NOT NULL DEFAULT 1,
    last_reviewed   DATE,
    reviewed_by     TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE element_attributions IS
'Every clinical decision element in every YAML Differential Tree has a row here. Traces red flags to NICE, LRs to McGee, context modifiers to API-ICP. Supports regulatory defense: "This LR value came from [source], was reviewed by [clinician] on [date], confidence level is [literature/consensus/estimated/cohort_calibrated]."';

CREATE INDEX idx_element_attr_node ON element_attributions(node_id);
CREATE INDEX idx_element_attr_type ON element_attributions(element_type);
CREATE INDEX idx_element_attr_concept ON element_attributions(canonical_concept_id);
CREATE INDEX idx_element_attr_source ON element_attributions(source_id);


-- ============================================================================
-- TABLE 4: canonical_concepts
-- Cross-node shared clinical concepts — asked once, reused across nodes
-- ============================================================================
CREATE TABLE canonical_concepts (
    concept_id      TEXT PRIMARY KEY,               -- 'postural_vs_constant'
    concept_name    TEXT NOT NULL,                   -- 'Postural vs Constant symptom pattern'
    snomed_code     TEXT,
    icd11_code      TEXT,
    loinc_code      TEXT,
    used_in_nodes   TEXT[] NOT NULL,                 -- '{P01,P08,P10}'
    extraction_mode TEXT NOT NULL CHECK (extraction_mode IN (
                        'BUTTON', 'REGEX', 'LLM', 'ASR_BUTTON', 'COMPOSITE'
                    )),
    hindi_question_template TEXT,                    -- standard Hindi phrasing
    hindi_variants  JSONB,                           -- regional/dialectal variants
    expected_pata_nahi_rate DECIMAL(3,2),            -- estimated "don't know" rate 0.00-1.00
    created_version INT NOT NULL DEFAULT 1,
    last_updated    DATE,
    updated_by      TEXT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE canonical_concepts IS
'Clinical concepts shared across multiple Presentation Nodes. When a patient answers a question in one node, the answer is available to other nodes that use the same canonical concept. Prevents redundant questioning in multi-complaint sessions and ensures consistent Hindi phrasing across nodes.';

CREATE INDEX idx_canon_concepts_nodes ON canonical_concepts USING GIN(used_in_nodes);


-- ============================================================================
-- TABLE 5: calibration_events
-- Full recalibration audit trail — every change to every clinical value
-- ============================================================================
CREATE TABLE calibration_events (
    event_id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id             TEXT NOT NULL,
    element_type        TEXT NOT NULL CHECK (element_type IN (
                            'likelihood_ratio',
                            'base_prior',
                            'context_modifier',
                            'question_rephrase',
                            'question_replace',
                            'differential_added',
                            'differential_removed',
                            'red_flag_added',
                            'red_flag_modified',
                            'completion_criteria',
                            'prior_floor'
                        )),
    element_key         TEXT NOT NULL,              -- 'Q01_postural_LR_orthostatic'
    trigger_type        TEXT NOT NULL CHECK (trigger_type IN (
                            'scheduled_quarterly',
                            'red_flag_miss',
                            'diagnosis_miss',
                            'calibration_drift',
                            'question_failure',
                            'low_agreement',
                            'canon_update',        -- external guideline updated
                            'outcome_validation',
                            'structural_review'
                        )),
    trigger_layer       INT NOT NULL CHECK (trigger_layer BETWEEN 1 AND 5),
                        -- 1=Pre-deployment, 2=Physician agreement,
                        -- 3=LR recalibration, 4=Outcome validation,
                        -- 5=Structural improvement
    old_value           JSONB,                     -- flexible: LR, prior, Hindi text, etc.
    new_value           JSONB,
    evidence_package    JSONB NOT NULL,             -- metrics that justified the change
                        -- Example: {"top1_agreement": 0.62, "sample_size": 50,
                        --           "observed_ppv": 0.45, "predicted_ppv": 0.67,
                        --           "pata_nahi_rate": 0.40, "info_gain_actual": 0.12}
    sample_size         INT,
    blend_ratio         TEXT,                      -- '0.8_observed_0.2_current'
    credible_interval   JSONB,                     -- {"lower": 0.30, "upper": 0.66, "width": 0.36}
    proposed_by         TEXT NOT NULL,
    reviewed_by         TEXT,                      -- second clinician (governance requirement)
    m8_validation_result TEXT CHECK (m8_validation_result IN (
                            'PASS', 'FAIL', 'PASS_WITH_NOTES', NULL
                        )),
    m8_red_flag_sensitivity DECIMAL(3,2),          -- must be 1.00
    m8_top1_accuracy    DECIMAL(3,2),              -- target >= 0.80
    m8_closure_guard_rate DECIMAL(3,2),            -- should be < 0.20
    approval_date       DATE,
    deployed_date       DATE,
    version_before      INT NOT NULL,
    version_after       INT NOT NULL,
    rollback_deadline   DATE,                      -- 48h post-deployment
    rolled_back         BOOLEAN NOT NULL DEFAULT FALSE,
    rollback_reason     TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE calibration_events IS
'Complete regulatory defense artifact. Every change to every clinical value in every Differential Tree is logged with: what triggered it (which quality layer), what evidence justified it, who proposed and reviewed it, M8 validation results, and deployment/rollback status. Court-defensible chain of custody for clinical reasoning changes.';

CREATE INDEX idx_calibration_node ON calibration_events(node_id);
CREATE INDEX idx_calibration_trigger ON calibration_events(trigger_type);
CREATE INDEX idx_calibration_layer ON calibration_events(trigger_layer);
CREATE INDEX idx_calibration_date ON calibration_events(deployed_date);


-- ============================================================================
-- TABLE 6: source_conflicts
-- Canon source disagreement resolution — when guidelines contradict
-- ============================================================================
CREATE TABLE source_conflicts (
    conflict_id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    node_id             TEXT NOT NULL,
    element_key         TEXT NOT NULL,              -- 'eGFR_threshold_metformin'
    sources_in_conflict TEXT[] NOT NULL,            -- '{ADA_2025,KDIGO_2024}'
    conflict_description TEXT NOT NULL,
    values_from_sources JSONB NOT NULL,             -- {"ADA_2025": 45, "KDIGO_2024": 30}
    resolution          TEXT NOT NULL,              -- what was decided
    resolution_rationale TEXT NOT NULL,             -- clinical reasoning for the choice
    population_factor   BOOLEAN NOT NULL DEFAULT FALSE, -- TRUE if Indian population was factor
    resolved_by         TEXT NOT NULL,
    reviewed_by         TEXT,
    resolution_date     DATE NOT NULL,
    node_version        INT NOT NULL,
    superseded_by       UUID REFERENCES source_conflicts(conflict_id), -- if re-resolved later
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE source_conflicts IS
'Documents when Canon sources disagree and how conflicts were resolved. Critical for Indian context where API-ICP may contradict global guidelines (ADA, KDIGO) on drug combinations, thresholds, or monitoring. Resolution rationale must be clinically justified and traceable.';

CREATE INDEX idx_conflicts_node ON source_conflicts(node_id);
CREATE INDEX idx_conflicts_sources ON source_conflicts USING GIN(sources_in_conflict);


-- ============================================================================
-- GOVERNANCE VIEWS
-- ============================================================================

-- View: Per-node source coverage summary
CREATE VIEW v_node_source_coverage AS
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

-- View: Elements needing recalibration (no cohort validation yet)
CREATE VIEW v_elements_pending_validation AS
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

-- View: Canon sources due for review
CREATE VIEW v_sources_due_for_review AS
SELECT
    code,
    name,
    update_cycle,
    next_review_due,
    last_reviewed,
    reviewed_by,
    CASE
        WHEN next_review_due < CURRENT_DATE THEN 'OVERDUE'
        WHEN next_review_due < CURRENT_DATE + INTERVAL '30 days' THEN 'DUE_SOON'
        ELSE 'OK'
    END AS review_status
FROM clinical_sources
WHERE source_category = 'canon'
ORDER BY next_review_due NULLS FIRST;

-- View: Calibration history per node with quality metrics
CREATE VIEW v_calibration_timeline AS
SELECT
    ce.node_id,
    ce.element_key,
    ce.trigger_type,
    ce.trigger_layer,
    ce.old_value,
    ce.new_value,
    ce.sample_size,
    ce.m8_red_flag_sensitivity,
    ce.m8_top1_accuracy,
    ce.proposed_by,
    ce.reviewed_by,
    ce.deployed_date,
    ce.rolled_back,
    ce.version_before,
    ce.version_after
FROM calibration_events ce
ORDER BY ce.node_id, ce.deployed_date;

-- View: Unresolved source conflicts
CREATE VIEW v_active_conflicts AS
SELECT
    sc.node_id,
    sc.element_key,
    sc.sources_in_conflict,
    sc.conflict_description,
    sc.resolution,
    sc.resolution_date,
    sc.resolved_by
FROM source_conflicts sc
WHERE sc.superseded_by IS NULL
ORDER BY sc.node_id;


-- ============================================================================
-- TRIGGER: Auto-update updated_at timestamps
-- ============================================================================
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_sources_updated
    BEFORE UPDATE ON clinical_sources
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trg_derivations_updated
    BEFORE UPDATE ON node_derivations
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trg_attributions_updated
    BEFORE UPDATE ON element_attributions
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trg_concepts_updated
    BEFORE UPDATE ON canonical_concepts
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- VALIDATION CONSTRAINTS
-- ============================================================================

-- Ensure calibration events always increment version
ALTER TABLE calibration_events
    ADD CONSTRAINT chk_version_increment
    CHECK (version_after > version_before);

-- Ensure red flag sensitivity is always 1.00 when M8 validated
ALTER TABLE calibration_events
    ADD CONSTRAINT chk_red_flag_sensitivity
    CHECK (
        m8_validation_result IS NULL
        OR m8_red_flag_sensitivity IS NULL
        OR m8_red_flag_sensitivity = 1.00
    );

-- Ensure Canon sources are never marked as runtime
ALTER TABLE clinical_sources
    ADD CONSTRAINT chk_canon_not_runtime
    CHECK (
        source_category != 'canon' OR runtime_allowed = FALSE
    );


-- ============================================================================
-- END OF MIGRATION V001
-- ============================================================================
