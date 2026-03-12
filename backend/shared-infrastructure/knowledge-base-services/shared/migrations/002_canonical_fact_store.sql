-- =============================================================================
-- MIGRATION 002: Canonical Fact Store (Layer 3)
-- Purpose: Single source of clinical truth - all KBs project from this store
-- Reference: KB1 Implementation Plan Section 3
-- =============================================================================

BEGIN;

-- =============================================================================
-- CANONICAL FACT STORE
-- The Six Fact Types:
--   1. ORGAN_IMPAIRMENT   - Renal, hepatic dosing (KB-1)
--   2. SAFETY_SIGNAL      - Black box warnings, contraindications (KB-4)
--   3. REPRODUCTIVE_SAFETY- Pregnancy/lactation categories (KB-4)
--   4. INTERACTION        - Drug-drug, drug-food interactions (KB-5)
--   5. FORMULARY          - Coverage, prior auth, tier status (KB-6)
--   6. LAB_REFERENCE      - Lab ranges, monitoring requirements (KB-16)
-- =============================================================================

-- Create enum types for validation
CREATE TYPE fact_type AS ENUM (
    'ORGAN_IMPAIRMENT',
    'SAFETY_SIGNAL',
    'REPRODUCTIVE_SAFETY',
    'INTERACTION',
    'FORMULARY',
    'LAB_REFERENCE'
);

CREATE TYPE fact_scope AS ENUM (
    'DRUG',    -- Applies to specific drug only
    'CLASS'    -- Applies to entire drug class
);

CREATE TYPE source_type AS ENUM (
    'LLM',         -- LLM extraction from narrative (KB-1, KB-4)
    'API_SYNC',    -- Structured API response (KB-5 DDI)
    'ETL',         -- CSV/file load (KB-6 Formulary, KB-16 Labs)
    'MANUAL'       -- Human-entered
);

CREATE TYPE confidence_band AS ENUM (
    'HIGH',    -- Human validated or high-confidence automated
    'MEDIUM',  -- Automated extraction, needs review
    'LOW'      -- Uncertain, needs review
);

CREATE TYPE fact_status AS ENUM (
    'DRAFT',       -- Newly extracted, awaiting review
    'APPROVED',    -- Pharmacist approved, ready for activation
    'ACTIVE',      -- In production use
    'SUPERSEDED',  -- Replaced by newer version
    'DEPRECATED'   -- Withdrawn
);

-- =============================================================================
-- MAIN CLINICAL FACTS TABLE
-- =============================================================================

CREATE TABLE IF NOT EXISTS clinical_facts (
    -- Identity
    fact_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fact_type           fact_type NOT NULL,

    -- Drug Reference (always RxCUI, linked to Drug Master)
    rxcui               VARCHAR(20) NOT NULL,
    drug_name           VARCHAR(500) NOT NULL,

    -- Scope Resolution (drug vs class level fact)
    scope               fact_scope NOT NULL DEFAULT 'DRUG',
    class_rxcui         VARCHAR(50),           -- If scope=CLASS, the class identifier
    class_name          VARCHAR(300),

    -- Fact Content (type-specific JSONB - schema depends on fact_type)
    content             JSONB NOT NULL,

    -- Provenance
    source_type         source_type NOT NULL,
    source_id           VARCHAR(255) NOT NULL, -- SPL SetID, API endpoint, file path
    source_version      VARCHAR(100),          -- Version/date of source
    extraction_method   VARCHAR(100) NOT NULL, -- e.g., "claude-renal-v2.3", "onc-ddi-loader"

    -- Confidence & Validation
    confidence_score    NUMERIC(3,2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    confidence_band     confidence_band NOT NULL,
    confidence_signals  JSONB,                 -- What contributed to confidence
    validated_by        VARCHAR(255),          -- Pharmacist who validated
    validated_at        TIMESTAMP WITH TIME ZONE,

    -- Lifecycle & Versioning
    status              fact_status NOT NULL DEFAULT 'DRAFT',
    effective_from      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    effective_to        TIMESTAMP WITH TIME ZONE,  -- NULL = currently active
    superseded_by       UUID,                  -- Points to newer version
    version             INTEGER NOT NULL DEFAULT 1,

    -- Audit Trail
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by          VARCHAR(255) NOT NULL DEFAULT 'system',
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT fk_drug_master FOREIGN KEY (rxcui)
        REFERENCES drug_master(rxcui) ON DELETE RESTRICT,

    CONSTRAINT fk_superseded FOREIGN KEY (superseded_by)
        REFERENCES clinical_facts(fact_id) ON DELETE SET NULL,

    -- Ensure class info is present for CLASS scope
    CONSTRAINT chk_class_scope CHECK (
        (scope = 'CLASS' AND class_rxcui IS NOT NULL) OR
        (scope = 'DRUG')
    ),

    -- Ensure confidence score aligns with band
    CONSTRAINT chk_confidence_alignment CHECK (
        (confidence_band = 'HIGH' AND confidence_score >= 0.85) OR
        (confidence_band = 'MEDIUM' AND confidence_score >= 0.65 AND confidence_score < 0.85) OR
        (confidence_band = 'LOW' AND confidence_score < 0.65) OR
        confidence_score IS NULL
    )
);

-- =============================================================================
-- INDEXES FOR FACT LOOKUPS
-- =============================================================================

-- Primary lookups by drug and type
CREATE INDEX idx_facts_rxcui ON clinical_facts(rxcui);
CREATE INDEX idx_facts_type ON clinical_facts(fact_type);
CREATE INDEX idx_facts_rxcui_type ON clinical_facts(rxcui, fact_type);

-- Active facts only (most common query pattern)
-- Note: Using only status and effective_to IS NULL since NOW() is not IMMUTABLE
CREATE INDEX idx_facts_active ON clinical_facts(rxcui, fact_type)
    WHERE status = 'ACTIVE' AND effective_to IS NULL;

-- Status-based filtering
CREATE INDEX idx_facts_status ON clinical_facts(status);
CREATE INDEX idx_facts_draft ON clinical_facts(created_at DESC) WHERE status = 'DRAFT';
CREATE INDEX idx_facts_pending_review ON clinical_facts(confidence_band, created_at)
    WHERE status = 'DRAFT' AND confidence_band = 'MEDIUM';

-- Temporal queries (for auditing past decisions)
CREATE INDEX idx_facts_temporal ON clinical_facts(rxcui, fact_type, effective_from, effective_to);

-- Source tracking
CREATE INDEX idx_facts_source ON clinical_facts(source_type, source_id);

-- Class-level facts
CREATE INDEX idx_facts_class ON clinical_facts(class_rxcui)
    WHERE scope = 'CLASS' AND status = 'ACTIVE';

-- JSONB content search (GIN for complex queries)
CREATE INDEX idx_facts_content ON clinical_facts USING gin(content);

-- =============================================================================
-- FACT TYPE-SPECIFIC TABLES (for optimized Phase 1 queries)
-- These are materialized views / denormalized tables for performance
-- =============================================================================

-- KB-5: Drug-Drug Interactions (optimized for pair lookups)
CREATE TABLE IF NOT EXISTS interaction_matrix (
    id                  SERIAL PRIMARY KEY,
    drug1_rxcui         VARCHAR(20) NOT NULL,
    drug1_name          VARCHAR(500) NOT NULL,
    drug2_rxcui         VARCHAR(20) NOT NULL,
    drug2_name          VARCHAR(500) NOT NULL,

    -- Interaction details
    severity            VARCHAR(50) NOT NULL,  -- CONTRAINDICATED, HIGH, MODERATE, LOW
    clinical_effect     TEXT,
    management          TEXT,
    mechanism           VARCHAR(255),
    documentation       VARCHAR(50),           -- ESTABLISHED, PROBABLE, SUSPECTED

    -- Directionality (from code review refinements)
    is_bidirectional    BOOLEAN DEFAULT TRUE,
    precipitant_rxcui   VARCHAR(20),           -- Drug causing interaction (perpetrator)
    object_rxcui        VARCHAR(20),           -- Drug affected (victim)
    interaction_mechanism VARCHAR(255),        -- CYP3A4_INHIBITION, etc.

    -- Source tracking
    source_dataset      VARCHAR(50) NOT NULL,  -- ONC, OHDSI, DRUGBANK
    source_pair_id      VARCHAR(50),           -- Original pair ID from source
    evidence_level      VARCHAR(20),
    clinical_source     VARCHAR(100),

    -- Link to canonical fact (optional, for governance)
    fact_id             UUID REFERENCES clinical_facts(fact_id),

    -- Metadata
    source_version      VARCHAR(50),
    last_updated        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Unique constraint prevents duplicate pairs from same source
    CONSTRAINT uq_interaction_pair UNIQUE (drug1_rxcui, drug2_rxcui, source_dataset)
);

-- Indexes for fast DDI lookups (the hot path!)
CREATE INDEX idx_interaction_drug1 ON interaction_matrix(drug1_rxcui);
CREATE INDEX idx_interaction_drug2 ON interaction_matrix(drug2_rxcui);
CREATE INDEX idx_interaction_pair ON interaction_matrix(drug1_rxcui, drug2_rxcui);
CREATE INDEX idx_interaction_severity ON interaction_matrix(severity);
CREATE INDEX idx_interaction_source ON interaction_matrix(source_dataset);

-- Composite index for the most common query pattern
CREATE INDEX idx_interaction_lookup ON interaction_matrix(drug1_rxcui, drug2_rxcui, severity);

-- KB-6: Formulary Coverage
CREATE TABLE IF NOT EXISTS formulary_coverage (
    id                  SERIAL PRIMARY KEY,
    rxcui               VARCHAR(20) NOT NULL,
    drug_name           VARCHAR(500) NOT NULL,
    generic_name        VARCHAR(500),
    ndc                 VARCHAR(20),

    -- Plan information
    contract_id         VARCHAR(20) NOT NULL,
    plan_id             VARCHAR(20) NOT NULL,
    segment_id          VARCHAR(20),
    plan_type           VARCHAR(50),           -- MEDICARE_D, COMMERCIAL, MEDICAID

    -- Coverage details
    on_formulary        BOOLEAN NOT NULL DEFAULT TRUE,
    tier                INTEGER,
    tier_level_code     VARCHAR(50),           -- PREFERRED_GENERIC, GENERIC, BRAND, SPECIALTY

    -- Utilization controls
    prior_auth          BOOLEAN DEFAULT FALSE,
    step_therapy        BOOLEAN DEFAULT FALSE,
    quantity_limit      BOOLEAN DEFAULT FALSE,
    quantity_limit_type VARCHAR(50),
    quantity_limit_amt  INTEGER,
    quantity_limit_days INTEGER,

    -- Dates
    effective_date      DATE,
    effective_year      INTEGER NOT NULL,

    -- Link to canonical fact
    fact_id             UUID REFERENCES clinical_facts(fact_id),

    -- Metadata
    source_version      VARCHAR(50),
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_formulary_entry UNIQUE (contract_id, plan_id, rxcui, ndc, effective_year)
);

CREATE INDEX idx_formulary_rxcui ON formulary_coverage(rxcui);
CREATE INDEX idx_formulary_plan ON formulary_coverage(contract_id, plan_id);
CREATE INDEX idx_formulary_tier ON formulary_coverage(tier_level_code);
CREATE INDEX idx_formulary_year ON formulary_coverage(effective_year);
CREATE INDEX idx_formulary_lookup ON formulary_coverage(rxcui, contract_id, plan_id, effective_year);

-- KB-16: Lab Reference Ranges
CREATE TABLE IF NOT EXISTS lab_reference_ranges (
    id                  SERIAL PRIMARY KEY,
    loinc_code          VARCHAR(20) NOT NULL,
    component           VARCHAR(255) NOT NULL,

    -- LOINC attributes
    property            VARCHAR(50),
    time_aspect         VARCHAR(20),
    system              VARCHAR(100),
    scale_type          VARCHAR(20),
    method_type         VARCHAR(100),
    class               VARCHAR(100),
    short_name          VARCHAR(100),
    long_name           TEXT,

    -- Reference ranges
    unit                VARCHAR(50),
    low_normal          NUMERIC(10,4),
    high_normal         NUMERIC(10,4),
    critical_low        NUMERIC(10,4),
    critical_high       NUMERIC(10,4),

    -- Population specificity
    age_group           VARCHAR(50),           -- adult, pediatric, geriatric, neonate
    sex                 VARCHAR(20),           -- male, female, all
    clinical_category   VARCHAR(50),           -- electrolyte, renal, cardiac, etc.

    -- Clinical guidance
    interpretation_guidance TEXT,

    -- Delta checks (for trending alerts - KDIGO AKI, HIT detection)
    delta_check_percent NUMERIC(5,2),
    delta_check_hours   INTEGER,

    -- Status
    deprecated          BOOLEAN DEFAULT FALSE,

    -- Link to canonical fact
    fact_id             UUID REFERENCES clinical_facts(fact_id),

    -- Metadata
    source_version      VARCHAR(50),
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_lab_range UNIQUE (loinc_code, age_group, sex, source_version)
);

CREATE INDEX idx_lab_loinc ON lab_reference_ranges(loinc_code);
CREATE INDEX idx_lab_category ON lab_reference_ranges(clinical_category);
CREATE INDEX idx_lab_component ON lab_reference_ranges(component);
CREATE INDEX idx_lab_delta ON lab_reference_ranges(loinc_code)
    WHERE delta_check_percent IS NOT NULL;

-- =============================================================================
-- HELPER VIEWS FOR KB PROJECTIONS
-- These implement the "KBs as read-only views" pattern from the plan
-- =============================================================================

-- KB-1 Projection: Renal Dosing Rules
CREATE OR REPLACE VIEW kb1_renal_dosing AS
SELECT
    fact_id,
    rxcui,
    drug_name,
    content->>'organ' AS organ,
    content->>'impairmentLevel' AS impairment_level,
    (content->>'egfrRangeLow')::numeric AS egfr_low,
    (content->>'egfrRangeHigh')::numeric AS egfr_high,
    content->>'ckdStage' AS ckd_stage,
    content->>'action' AS action,
    content->>'doseAdjustment' AS dose_adjustment,
    (content->>'maxDose')::numeric AS max_dose,
    content->>'maxDoseUnit' AS max_dose_unit,
    content->>'rationale' AS rationale,
    confidence_band,
    source_type,
    effective_from
FROM clinical_facts
WHERE fact_type = 'ORGAN_IMPAIRMENT'
  AND content->>'organ' = 'RENAL'
  AND status = 'ACTIVE'
  AND (effective_to IS NULL OR effective_to > NOW());

-- KB-4 Projection: Safety Signals
CREATE OR REPLACE VIEW kb4_safety_signals AS
SELECT
    fact_id,
    rxcui,
    drug_name,
    content->>'signalType' AS signal_type,
    content->>'severity' AS severity,
    content->>'conditionCode' AS condition_code,
    content->>'conditionName' AS condition_name,
    content->>'description' AS description,
    content->>'recommendation' AS recommendation,
    (content->>'requiresMonitor')::boolean AS requires_monitor,
    confidence_band,
    source_type,
    effective_from
FROM clinical_facts
WHERE fact_type = 'SAFETY_SIGNAL'
  AND status = 'ACTIVE'
  AND (effective_to IS NULL OR effective_to > NOW());

-- KB-5 Projection: Drug Interactions (using optimized table)
CREATE OR REPLACE VIEW kb5_interactions AS
SELECT
    drug1_rxcui,
    drug1_name,
    drug2_rxcui,
    drug2_name,
    severity,
    clinical_effect,
    management,
    mechanism,
    documentation,
    is_bidirectional,
    precipitant_rxcui,
    object_rxcui,
    source_dataset,
    evidence_level
FROM interaction_matrix;

-- KB-6 Projection: Formulary Coverage
CREATE OR REPLACE VIEW kb6_formulary AS
SELECT
    rxcui,
    drug_name,
    generic_name,
    contract_id,
    plan_id,
    on_formulary,
    tier,
    tier_level_code,
    prior_auth,
    step_therapy,
    quantity_limit,
    effective_year
FROM formulary_coverage
WHERE on_formulary = TRUE;

-- KB-16 Projection: Lab Reference Ranges
CREATE OR REPLACE VIEW kb16_lab_ranges AS
SELECT
    loinc_code,
    component,
    unit,
    low_normal,
    high_normal,
    critical_low,
    critical_high,
    age_group,
    sex,
    clinical_category,
    interpretation_guidance,
    delta_check_percent,
    delta_check_hours
FROM lab_reference_ranges
WHERE deprecated = FALSE;

-- =============================================================================
-- GOVERNANCE TABLES
-- =============================================================================

-- Track fact review decisions
CREATE TABLE IF NOT EXISTS fact_reviews (
    review_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fact_id             UUID NOT NULL REFERENCES clinical_facts(fact_id),
    reviewer_id         VARCHAR(255) NOT NULL,
    reviewer_role       VARCHAR(100) NOT NULL,  -- PHARMACIST, CLINICIAN, ADMIN
    decision            VARCHAR(50) NOT NULL,   -- APPROVE, REJECT, REQUEST_CHANGES
    comments            TEXT,
    reviewed_at         TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_reviews_fact ON fact_reviews(fact_id);
CREATE INDEX idx_reviews_pending ON fact_reviews(reviewer_id, reviewed_at DESC);

-- Ingestion metadata (track data loads)
CREATE TABLE IF NOT EXISTS ingestion_metadata (
    id                  SERIAL PRIMARY KEY,
    source_name         VARCHAR(100) NOT NULL,
    source_version      VARCHAR(100) NOT NULL,
    records_loaded      INTEGER,
    records_skipped     INTEGER,
    records_failed      INTEGER,
    load_timestamp      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sha256_checksum     VARCHAR(64),
    source_url          TEXT,
    load_duration_ms    INTEGER,
    notes               TEXT,

    CONSTRAINT uq_ingestion UNIQUE (source_name, source_version, load_timestamp)
);

-- Record migration
INSERT INTO schema_migrations (version, name) VALUES (2, '002_canonical_fact_store')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- =============================================================================
-- VERIFICATION
-- =============================================================================
SELECT 'Migration 002: Canonical Fact Store - COMPLETE' AS status;
