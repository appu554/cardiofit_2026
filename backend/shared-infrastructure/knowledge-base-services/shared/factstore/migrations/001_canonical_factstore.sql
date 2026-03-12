-- ============================================================================
-- CANONICAL FACT STORE SCHEMA
-- ============================================================================
-- Version: 1.0.0
-- Description: Core schema for the Canonical Fact Store - the immutable
--              knowledge spine for all KB services.
--
-- DESIGN PRINCIPLE: "Freeze meaning. Fluidly replace intelligence."
-- Facts are immutable once approved. Model changes create new versions.
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search

-- ============================================================================
-- ENUMS
-- ============================================================================

-- Fact types - the six canonical clinical domains
CREATE TYPE fact_type AS ENUM (
    'ORGAN_IMPAIRMENT',     -- Renal/hepatic dosing adjustments
    'SAFETY_SIGNAL',        -- Warnings, contraindications, precautions
    'REPRODUCTIVE_SAFETY',  -- Pregnancy, lactation, fertility
    'INTERACTION',          -- Drug-drug, drug-food interactions
    'FORMULARY',           -- Coverage, tiering, prior auth
    'LAB_REFERENCE'        -- Lab values, monitoring requirements
);

-- Fact lifecycle status
CREATE TYPE fact_status AS ENUM (
    'DRAFT',       -- Extracted, awaiting review
    'APPROVED',    -- Reviewed, awaiting activation
    'ACTIVE',      -- Live, used in KB projections
    'SUPERSEDED',  -- Replaced by newer version
    'DEPRECATED',  -- Marked for removal
    'ARCHIVED'     -- Historical record only
);

-- Governance decision outcomes
CREATE TYPE governance_decision AS ENUM (
    'AUTO_APPROVED',   -- Confidence >= 0.85, auto-activated
    'REVIEW_REQUIRED', -- 0.65 <= confidence < 0.85
    'AUTO_REJECTED',   -- Confidence < 0.65
    'HUMAN_APPROVED',  -- Manually approved after review
    'HUMAN_REJECTED',  -- Manually rejected after review
    'ESCALATED'        -- Sent to clinical review board
);

-- Organ system for impairment facts
CREATE TYPE organ_system AS ENUM (
    'RENAL',
    'HEPATIC',
    'CARDIAC',
    'PULMONARY',
    'NEUROLOGICAL'
);

-- Severity levels
CREATE TYPE severity_level AS ENUM (
    'MILD',
    'MODERATE',
    'SEVERE',
    'CONTRAINDICATED'
);

-- Interaction severity
CREATE TYPE interaction_severity AS ENUM (
    'MINOR',
    'MODERATE',
    'MAJOR',
    'CONTRAINDICATED'
);

-- FDA pregnancy categories (legacy + new system)
CREATE TYPE pregnancy_category AS ENUM (
    'A', 'B', 'C', 'D', 'X',  -- Legacy categories
    'SAFE',                    -- New system
    'CAUTION',
    'AVOID',
    'CONTRAINDICATED'
);

-- Evidence levels
CREATE TYPE evidence_level AS ENUM (
    'META_ANALYSIS',
    'RANDOMIZED_CONTROLLED_TRIAL',
    'COHORT_STUDY',
    'CASE_CONTROL',
    'CASE_SERIES',
    'EXPERT_OPINION',
    'MANUFACTURER_LABEL',
    'REGULATORY_GUIDANCE'
);

-- ============================================================================
-- CORE TABLES
-- ============================================================================

-- Main facts table with temporal versioning
CREATE TABLE facts (
    -- Identity
    fact_id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fact_type       fact_type NOT NULL,

    -- Drug reference (normalized)
    rxcui           VARCHAR(20) NOT NULL,
    drug_name       VARCHAR(500),
    ndc             VARCHAR(20),

    -- Content (JSONB for flexibility within type constraints)
    content         JSONB NOT NULL,

    -- Confidence (denormalized for query performance)
    confidence_overall    DECIMAL(4,3) NOT NULL CHECK (confidence_overall >= 0 AND confidence_overall <= 1),
    confidence_source     DECIMAL(4,3) CHECK (confidence_source >= 0 AND confidence_source <= 1),
    confidence_extraction DECIMAL(4,3) CHECK (confidence_extraction >= 0 AND confidence_extraction <= 1),
    confidence_model      VARCHAR(100),

    -- Lifecycle
    status              fact_status NOT NULL DEFAULT 'DRAFT',
    effective_from      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to        TIMESTAMPTZ,
    superseded_by       UUID REFERENCES facts(fact_id),
    supersedes          UUID REFERENCES facts(fact_id),

    -- Provenance
    extractor_id        VARCHAR(100) NOT NULL,
    extractor_version   VARCHAR(50),
    extraction_id       UUID,
    prompt_version      VARCHAR(50),
    model_id            VARCHAR(100),
    source_document_id  VARCHAR(500),
    source_url          TEXT,

    -- Governance
    governance_decision     governance_decision,
    governance_timestamp    TIMESTAMPTZ,
    reviewed_by             VARCHAR(100),
    review_notes            TEXT,

    -- Regulatory
    jurisdiction        VARCHAR(10) DEFAULT 'US',
    regulatory_body     VARCHAR(50),
    regulatory_reference VARCHAR(500),

    -- Audit
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      VARCHAR(100),

    -- Constraints
    CONSTRAINT valid_temporal CHECK (effective_to IS NULL OR effective_to > effective_from),
    CONSTRAINT valid_supersession CHECK (
        (superseded_by IS NULL) OR (status IN ('SUPERSEDED', 'ARCHIVED'))
    )
);

-- ============================================================================
-- CONTENT VALIDATION (Check constraints for JSONB content by type)
-- ============================================================================

-- Validate ORGAN_IMPAIRMENT content structure
CREATE OR REPLACE FUNCTION validate_organ_impairment_content(content JSONB)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN (
        content ? 'organSystem' AND
        content ? 'severityStage' AND
        content ? 'doseAdjustment'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Validate SAFETY_SIGNAL content structure
CREATE OR REPLACE FUNCTION validate_safety_signal_content(content JSONB)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN (
        content ? 'signalType' AND
        content ? 'description' AND
        content ? 'severity'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Validate INTERACTION content structure
CREATE OR REPLACE FUNCTION validate_interaction_content(content JSONB)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN (
        content ? 'interactingDrugRxCUI' AND
        content ? 'severity' AND
        content ? 'mechanism'
    );
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Content validation trigger
CREATE OR REPLACE FUNCTION validate_fact_content()
RETURNS TRIGGER AS $$
BEGIN
    CASE NEW.fact_type
        WHEN 'ORGAN_IMPAIRMENT' THEN
            IF NOT validate_organ_impairment_content(NEW.content) THEN
                RAISE EXCEPTION 'Invalid ORGAN_IMPAIRMENT content structure';
            END IF;
        WHEN 'SAFETY_SIGNAL' THEN
            IF NOT validate_safety_signal_content(NEW.content) THEN
                RAISE EXCEPTION 'Invalid SAFETY_SIGNAL content structure';
            END IF;
        WHEN 'INTERACTION' THEN
            IF NOT validate_interaction_content(NEW.content) THEN
                RAISE EXCEPTION 'Invalid INTERACTION content structure';
            END IF;
        ELSE
            -- Other types: basic validation only
            NULL;
    END CASE;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fact_content_validation
    BEFORE INSERT OR UPDATE ON facts
    FOR EACH ROW
    EXECUTE FUNCTION validate_fact_content();

-- ============================================================================
-- INDEXES
-- ============================================================================

-- Primary access patterns
CREATE INDEX idx_facts_rxcui ON facts(rxcui);
CREATE INDEX idx_facts_type ON facts(fact_type);
CREATE INDEX idx_facts_status ON facts(status);
CREATE INDEX idx_facts_type_status ON facts(fact_type, status);
CREATE INDEX idx_facts_rxcui_type ON facts(rxcui, fact_type);
CREATE INDEX idx_facts_rxcui_status ON facts(rxcui, status);

-- Temporal queries (active facts at a point in time)
CREATE INDEX idx_facts_temporal ON facts(effective_from, effective_to);
CREATE INDEX idx_facts_active ON facts(rxcui, fact_type)
    WHERE status = 'ACTIVE' AND effective_to IS NULL;

-- Governance workflow
CREATE INDEX idx_facts_governance ON facts(governance_decision, governance_timestamp);
CREATE INDEX idx_facts_pending_review ON facts(fact_type, created_at)
    WHERE status = 'DRAFT';

-- Provenance queries
CREATE INDEX idx_facts_extractor ON facts(extractor_id, extractor_version);
CREATE INDEX idx_facts_source_doc ON facts(source_document_id);

-- Full-text search on drug name
CREATE INDEX idx_facts_drug_name_trgm ON facts USING gin(drug_name gin_trgm_ops);

-- JSONB content indexing for common queries
CREATE INDEX idx_facts_content ON facts USING gin(content);
CREATE INDEX idx_facts_content_organ ON facts((content->>'organSystem'))
    WHERE fact_type = 'ORGAN_IMPAIRMENT';
CREATE INDEX idx_facts_content_interaction ON facts((content->>'interactingDrugRxCUI'))
    WHERE fact_type = 'INTERACTION';

-- ============================================================================
-- AUDIT TRAIL TABLE
-- ============================================================================

CREATE TABLE fact_audit_log (
    audit_id        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fact_id         UUID NOT NULL REFERENCES facts(fact_id),
    action          VARCHAR(50) NOT NULL, -- INSERT, UPDATE, STATUS_CHANGE, etc.
    old_status      fact_status,
    new_status      fact_status,
    old_content     JSONB,
    new_content     JSONB,
    changed_by      VARCHAR(100),
    change_reason   TEXT,
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_fact_id ON fact_audit_log(fact_id);
CREATE INDEX idx_audit_timestamp ON fact_audit_log(timestamp);
CREATE INDEX idx_audit_action ON fact_audit_log(action);

-- Audit trigger
CREATE OR REPLACE FUNCTION audit_fact_changes()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        INSERT INTO fact_audit_log (
            fact_id, action, old_status, new_status,
            old_content, new_content, changed_by, timestamp
        ) VALUES (
            NEW.fact_id,
            CASE
                WHEN OLD.status != NEW.status THEN 'STATUS_CHANGE'
                WHEN OLD.content != NEW.content THEN 'CONTENT_UPDATE'
                ELSE 'UPDATE'
            END,
            OLD.status, NEW.status,
            OLD.content, NEW.content,
            NEW.created_by,
            NOW()
        );
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO fact_audit_log (
            fact_id, action, new_status, new_content, changed_by, timestamp
        ) VALUES (
            NEW.fact_id, 'INSERT', NEW.status, NEW.content, NEW.created_by, NOW()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER fact_audit_trigger
    AFTER INSERT OR UPDATE ON facts
    FOR EACH ROW
    EXECUTE FUNCTION audit_fact_changes();

-- ============================================================================
-- GOVERNANCE QUEUE TABLE
-- ============================================================================

CREATE TABLE governance_queue (
    queue_id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fact_id             UUID NOT NULL REFERENCES facts(fact_id),
    priority            INTEGER NOT NULL DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    confidence_score    DECIMAL(4,3) NOT NULL,
    fact_type           fact_type NOT NULL,
    rxcui               VARCHAR(20) NOT NULL,
    drug_name           VARCHAR(500),

    -- Queue status
    queued_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_to         VARCHAR(100),
    assigned_at         TIMESTAMPTZ,

    -- Review tracking
    review_deadline     TIMESTAMPTZ,
    escalation_count    INTEGER DEFAULT 0,

    -- Resolution
    resolved            BOOLEAN DEFAULT FALSE,
    resolved_at         TIMESTAMPTZ,
    resolution_decision governance_decision,
    resolution_notes    TEXT
);

CREATE INDEX idx_governance_pending ON governance_queue(priority DESC, queued_at)
    WHERE resolved = FALSE;
CREATE INDEX idx_governance_assigned ON governance_queue(assigned_to)
    WHERE resolved = FALSE;
CREATE INDEX idx_governance_fact ON governance_queue(fact_id);

-- ============================================================================
-- EXTRACTOR REGISTRY TABLE
-- ============================================================================

CREATE TABLE extractor_registry (
    extractor_id        VARCHAR(100) PRIMARY KEY,
    extractor_type      VARCHAR(50) NOT NULL, -- LLM, API, ETL, HYBRID, MANUAL
    version             VARCHAR(50) NOT NULL,

    -- Capabilities
    source_types        TEXT[] NOT NULL,
    fact_types          TEXT[] NOT NULL,
    clinical_domains    TEXT[] NOT NULL,

    -- Model info (for LLM extractors)
    model_id            VARCHAR(100),
    model_version       VARCHAR(50),
    prompt_template_version VARCHAR(50),

    -- Validation
    validation_status   VARCHAR(50) DEFAULT 'PENDING',
    last_validated      TIMESTAMPTZ,
    validation_score    DECIMAL(4,3),

    -- Confidence model
    confidence_model_name    VARCHAR(100),
    confidence_model_version VARCHAR(50),
    min_acceptable_confidence DECIMAL(4,3) DEFAULT 0.65,
    auto_approve_threshold    DECIMAL(4,3) DEFAULT 0.85,

    -- Operational
    enabled             BOOLEAN DEFAULT TRUE,
    rate_limit_per_minute INTEGER,
    cost_per_extraction   DECIMAL(10,4),
    average_latency_ms    INTEGER,

    -- Audit
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_confidence_thresholds CHECK (
        min_acceptable_confidence < auto_approve_threshold AND
        auto_approve_threshold <= 1.0
    )
);

CREATE INDEX idx_extractor_type ON extractor_registry(extractor_type);
CREATE INDEX idx_extractor_enabled ON extractor_registry(enabled) WHERE enabled = TRUE;

-- ============================================================================
-- KB PROJECTION TRACKING
-- ============================================================================

CREATE TABLE kb_projections (
    projection_id       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kb_name             VARCHAR(50) NOT NULL UNIQUE, -- KB-1, KB-2, etc.
    kb_description      TEXT,

    -- Projection criteria (which facts this KB uses)
    fact_types          fact_type[] NOT NULL,
    clinical_domains    TEXT[],
    jurisdictions       TEXT[] DEFAULT ARRAY['US'],

    -- Refresh tracking
    last_refreshed      TIMESTAMPTZ,
    refresh_interval_minutes INTEGER DEFAULT 60,

    -- Statistics
    active_fact_count   INTEGER DEFAULT 0,
    last_fact_added     TIMESTAMPTZ,

    -- Status
    enabled             BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kb_projections_name ON kb_projections(kb_name);

-- Junction table: which facts are in which KB projection
CREATE TABLE kb_fact_membership (
    kb_name             VARCHAR(50) NOT NULL REFERENCES kb_projections(kb_name),
    fact_id             UUID NOT NULL REFERENCES facts(fact_id),
    added_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (kb_name, fact_id)
);

CREATE INDEX idx_kb_membership_kb ON kb_fact_membership(kb_name);
CREATE INDEX idx_kb_membership_fact ON kb_fact_membership(fact_id);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Get active facts for an RxCUI
CREATE OR REPLACE FUNCTION get_active_facts(
    p_rxcui VARCHAR(20),
    p_fact_type fact_type DEFAULT NULL,
    p_as_of TIMESTAMPTZ DEFAULT NOW()
)
RETURNS SETOF facts AS $$
BEGIN
    RETURN QUERY
    SELECT f.*
    FROM facts f
    WHERE f.rxcui = p_rxcui
      AND f.status = 'ACTIVE'
      AND f.effective_from <= p_as_of
      AND (f.effective_to IS NULL OR f.effective_to > p_as_of)
      AND (p_fact_type IS NULL OR f.fact_type = p_fact_type)
    ORDER BY f.confidence_overall DESC;
END;
$$ LANGUAGE plpgsql;

-- Supersede a fact with a new version
CREATE OR REPLACE FUNCTION supersede_fact(
    p_old_fact_id UUID,
    p_new_fact_id UUID,
    p_superseded_by VARCHAR(100) DEFAULT NULL
)
RETURNS VOID AS $$
BEGIN
    -- Mark old fact as superseded
    UPDATE facts
    SET status = 'SUPERSEDED',
        superseded_by = p_new_fact_id,
        effective_to = NOW(),
        updated_at = NOW()
    WHERE fact_id = p_old_fact_id;

    -- Link new fact to old
    UPDATE facts
    SET supersedes = p_old_fact_id,
        updated_at = NOW()
    WHERE fact_id = p_new_fact_id;

    -- Log the supersession
    INSERT INTO fact_audit_log (
        fact_id, action, old_status, new_status, changed_by, change_reason, timestamp
    ) VALUES (
        p_old_fact_id, 'SUPERSEDED', 'ACTIVE', 'SUPERSEDED',
        p_superseded_by, 'Superseded by ' || p_new_fact_id::text, NOW()
    );
END;
$$ LANGUAGE plpgsql;

-- Auto-governance function (called after fact insertion)
CREATE OR REPLACE FUNCTION auto_govern_fact()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'DRAFT' THEN
        IF NEW.confidence_overall >= 0.85 THEN
            -- Auto-approve high confidence facts
            NEW.status := 'ACTIVE';
            NEW.governance_decision := 'AUTO_APPROVED';
            NEW.governance_timestamp := NOW();
        ELSIF NEW.confidence_overall >= 0.65 THEN
            -- Queue for human review
            NEW.governance_decision := 'REVIEW_REQUIRED';
            INSERT INTO governance_queue (
                fact_id, priority, confidence_score, fact_type, rxcui, drug_name
            ) VALUES (
                NEW.fact_id,
                CASE
                    WHEN NEW.confidence_overall >= 0.80 THEN 3
                    WHEN NEW.confidence_overall >= 0.75 THEN 5
                    ELSE 7
                END,
                NEW.confidence_overall,
                NEW.fact_type,
                NEW.rxcui,
                NEW.drug_name
            );
        ELSE
            -- Auto-reject low confidence
            NEW.status := 'DEPRECATED';
            NEW.governance_decision := 'AUTO_REJECTED';
            NEW.governance_timestamp := NOW();
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER auto_governance_trigger
    BEFORE INSERT ON facts
    FOR EACH ROW
    EXECUTE FUNCTION auto_govern_fact();

-- Updated timestamp trigger
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER facts_updated_at
    BEFORE UPDATE ON facts
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER extractor_registry_updated_at
    BEFORE UPDATE ON extractor_registry
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER kb_projections_updated_at
    BEFORE UPDATE ON kb_projections
    FOR EACH ROW
    EXECUTE FUNCTION update_timestamp();

-- ============================================================================
-- INITIAL DATA: KB PROJECTIONS
-- ============================================================================

INSERT INTO kb_projections (kb_name, kb_description, fact_types, clinical_domains) VALUES
('KB-1', 'Drug Dosing & Organ Impairment Rules',
    ARRAY['ORGAN_IMPAIRMENT']::fact_type[],
    ARRAY['renal', 'hepatic']),
('KB-2', 'Clinical Context & Protocols',
    ARRAY['SAFETY_SIGNAL', 'LAB_REFERENCE']::fact_type[],
    ARRAY['safety', 'lab']),
('KB-3', 'Guidelines & Evidence Base',
    ARRAY['SAFETY_SIGNAL', 'ORGAN_IMPAIRMENT']::fact_type[],
    ARRAY['cardiac', 'geriatric', 'pediatric']),
('KB-4', 'Patient Safety Signals',
    ARRAY['SAFETY_SIGNAL', 'REPRODUCTIVE_SAFETY']::fact_type[],
    ARRAY['safety', 'reproductive']),
('KB-5', 'Drug Interactions',
    ARRAY['INTERACTION']::fact_type[],
    ARRAY['interaction']),
('KB-6', 'Formulary & Coverage',
    ARRAY['FORMULARY']::fact_type[],
    ARRAY['formulary']),
('KB-7', 'Terminology & Mappings',
    ARRAY['LAB_REFERENCE']::fact_type[],
    ARRAY['lab']);

-- ============================================================================
-- GRANTS (adjust based on your user setup)
-- ============================================================================

-- Read-only role for KB services
-- CREATE ROLE kb_reader;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb_reader;

-- Write role for extraction services
-- CREATE ROLE kb_writer;
-- GRANT SELECT, INSERT, UPDATE ON facts, fact_audit_log, governance_queue TO kb_writer;
-- GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO kb_writer;

-- Admin role
-- CREATE ROLE kb_admin;
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO kb_admin;

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================
