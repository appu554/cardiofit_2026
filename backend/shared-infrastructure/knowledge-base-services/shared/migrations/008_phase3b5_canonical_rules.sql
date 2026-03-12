-- =============================================================================
-- MIGRATION 008: Phase 3b.5 Canonical Rule Generation
-- Purpose: Tables for DraftRule storage, semantic deduplication, and human review
-- Reference: PHASE3_IMPLEMENTATION_PLAN.md - Section 3b.5
-- =============================================================================
-- Architecture:
--   ExtractedTable → NormalizedTable → Condition/Action → DraftRule
--
-- Key Principles:
--   1. Same clinical meaning = same fingerprint (semantic deduplication)
--   2. Tables that can't be translated → Human review (NOT LLM)
--   3. Full provenance chain from source to computable rule
--   4. SLA tracking for untranslatable queue items
-- =============================================================================

BEGIN;

-- =============================================================================
-- RULE TYPE AND STATUS ENUMS
-- =============================================================================

-- Types of clinical rules extracted from SPL tables
CREATE TYPE rule_type AS ENUM (
    'DOSING',               -- Dose adjustment rules (GFR, hepatic, etc.)
    'CONTRAINDICATION',     -- Absolute/relative contraindications
    'INTERACTION',          -- Drug-drug interactions
    'PRECAUTION',           -- Warnings and precautions
    'MONITORING'            -- Lab/vital monitoring requirements
);

-- Governance lifecycle for draft rules
CREATE TYPE rule_governance_status AS ENUM (
    'DRAFT',                -- Just extracted, not yet reviewed
    'PENDING_REVIEW',       -- In governance queue
    'APPROVED',             -- Approved by reviewer
    'REJECTED',             -- Rejected (quality/accuracy)
    'SUPERSEDED',           -- Replaced by newer version
    'ARCHIVED'              -- Historical, no longer active
);

-- Operators for conditions
CREATE TYPE condition_operator AS ENUM (
    'LT',                   -- Less than (<)
    'LTE',                  -- Less than or equal (<=)
    'GT',                   -- Greater than (>)
    'GTE',                  -- Greater than or equal (>=)
    'EQ',                   -- Equal (==)
    'NEQ',                  -- Not equal (!=)
    'BETWEEN',              -- Range (min <= x <= max)
    'IN',                   -- Set membership
    'NOT_IN'                -- Set exclusion
);

-- Effects/actions for rules
CREATE TYPE rule_effect AS ENUM (
    'CONTRAINDICATED',      -- Do not use
    'DOSE_ADJUST',          -- Modify dose
    'AVOID',                -- Use with extreme caution
    'USE_WITH_CAUTION',     -- Use carefully with monitoring
    'MONITOR',              -- Requires monitoring
    'MAX_DOSE',             -- Ceiling dose
    'REDUCE_FREQUENCY',     -- Extend dosing interval
    'NO_CHANGE'             -- Informational, no action needed
);

-- Severity levels for actions
CREATE TYPE rule_severity AS ENUM (
    'CRITICAL',             -- Life-threatening
    'HIGH',                 -- Serious adverse event risk
    'MODERATE',             -- Significant clinical impact
    'LOW',                  -- Minor clinical impact
    'INFO'                  -- Informational only
);

-- =============================================================================
-- DRAFT RULES TABLE
-- =============================================================================
-- Stores canonical computable rules extracted from clinical sources

CREATE TABLE IF NOT EXISTS draft_rules (
    -- Primary key
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Domain classification
    domain VARCHAR(20) NOT NULL,                    -- KB-1, KB-4, KB-5, etc.
    rule_type rule_type NOT NULL,

    -- Condition (IF part) - stored as JSONB for flexibility
    condition_variable VARCHAR(100) NOT NULL,       -- e.g., renal_function.crcl
    condition_operator condition_operator NOT NULL,
    condition_value DECIMAL(10,4),                  -- For simple comparisons
    condition_min_value DECIMAL(10,4),              -- For BETWEEN ranges
    condition_max_value DECIMAL(10,4),              -- For BETWEEN ranges
    condition_string_value VARCHAR(255),            -- For categorical (Child-Pugh)
    condition_string_values TEXT[],                 -- For IN/NOT_IN
    condition_unit VARCHAR(50),                     -- e.g., mL/min
    condition_full JSONB NOT NULL,                  -- Complete condition object

    -- Action (THEN part) - stored as JSONB for flexibility
    action_effect rule_effect NOT NULL,
    action_adjustment_type VARCHAR(50),             -- PERCENTAGE, ABSOLUTE, INTERVAL
    action_adjustment_value DECIMAL(10,4),
    action_adjustment_unit VARCHAR(50),
    action_max_dose DECIMAL(10,4),
    action_max_dose_unit VARCHAR(50),
    action_message TEXT,
    action_severity rule_severity,
    action_full JSONB NOT NULL,                     -- Complete action object

    -- Provenance chain
    source_document_id UUID NOT NULL,               -- Link to source document
    source_section_id UUID,                         -- Link to specific section
    source_type VARCHAR(50) NOT NULL,               -- DAILYMED_SPL, GUIDELINE, etc.
    document_id VARCHAR(255),                       -- External document identifier
    section_code VARCHAR(20),                       -- SPL section code
    table_id VARCHAR(100),                          -- Source table identifier
    evidence_span TEXT,                             -- Specific text/row extracted from
    extraction_method VARCHAR(50) NOT NULL,         -- TABLE_PARSE, PATTERN_MATCH, etc.
    extracted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    provenance_full JSONB NOT NULL,                 -- Complete provenance object

    -- Semantic fingerprint for deduplication
    fingerprint_hash VARCHAR(64) NOT NULL UNIQUE,   -- SHA256 hash
    fingerprint_version INT NOT NULL DEFAULT 1,
    fingerprint_components JSONB NOT NULL,          -- What was hashed

    -- Quality metrics
    confidence DECIMAL(3,2) NOT NULL DEFAULT 0.0,   -- 0.00-1.00
    validation_errors TEXT[],
    validation_warnings TEXT[],

    -- Governance workflow
    governance_status rule_governance_status NOT NULL DEFAULT 'DRAFT',
    assigned_reviewer VARCHAR(255),
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,

    -- Lifecycle
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL DEFAULT 'SYSTEM',
    updated_by VARCHAR(255) NOT NULL DEFAULT 'SYSTEM',
    version INT NOT NULL DEFAULT 1,
    is_latest BOOLEAN NOT NULL DEFAULT TRUE,
    previous_version_id UUID REFERENCES draft_rules(rule_id)
);

-- Indexes for draft_rules
CREATE INDEX idx_draft_rules_domain ON draft_rules(domain);
CREATE INDEX idx_draft_rules_type ON draft_rules(rule_type);
CREATE INDEX idx_draft_rules_status ON draft_rules(governance_status);
CREATE INDEX idx_draft_rules_fingerprint ON draft_rules(fingerprint_hash);
CREATE INDEX idx_draft_rules_source_doc ON draft_rules(source_document_id);
CREATE INDEX idx_draft_rules_confidence ON draft_rules(confidence);
CREATE INDEX idx_draft_rules_condition_var ON draft_rules(condition_variable);
CREATE INDEX idx_draft_rules_created ON draft_rules(created_at);
CREATE INDEX idx_draft_rules_latest ON draft_rules(is_latest) WHERE is_latest = TRUE;

-- GIN index for JSONB searches
CREATE INDEX idx_draft_rules_condition_gin ON draft_rules USING GIN (condition_full);
CREATE INDEX idx_draft_rules_action_gin ON draft_rules USING GIN (action_full);
CREATE INDEX idx_draft_rules_provenance_gin ON draft_rules USING GIN (provenance_full);

-- =============================================================================
-- FINGERPRINT REGISTRY
-- =============================================================================
-- Stores semantic fingerprints for deduplication across all rules

CREATE TABLE IF NOT EXISTS fingerprint_registry (
    -- Primary key is the hash itself
    hash VARCHAR(64) PRIMARY KEY,

    -- Link to the canonical rule
    rule_id UUID NOT NULL REFERENCES draft_rules(rule_id),

    -- Classification
    domain VARCHAR(20) NOT NULL,
    rule_type rule_type NOT NULL,

    -- Fingerprint metadata
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Deduplication tracking
    source_count INT NOT NULL DEFAULT 1,            -- How many sources produced this rule
    sources JSONB NOT NULL DEFAULT '[]'::jsonb      -- List of source documents
);

-- Indexes for fingerprint_registry
CREATE INDEX idx_fingerprint_domain ON fingerprint_registry(domain);
CREATE INDEX idx_fingerprint_type ON fingerprint_registry(rule_type);
CREATE INDEX idx_fingerprint_rule ON fingerprint_registry(rule_id);
CREATE INDEX idx_fingerprint_created ON fingerprint_registry(created_at);

-- =============================================================================
-- UNTRANSLATABLE QUEUE
-- =============================================================================
-- Tables that cannot be automatically translated go here for human review
-- Per Navigation Rule 4: Untranslatable → HUMAN REVIEW, NOT LLM

CREATE TYPE untranslatable_status AS ENUM (
    'PENDING',              -- Not yet assigned
    'IN_REVIEW',            -- Being reviewed by human
    'RESOLVED',             -- Successfully translated manually
    'REJECTED',             -- Table cannot/should not be translated
    'ESCALATED'             -- Needs expert review
);

CREATE TYPE untranslatable_reason AS ENUM (
    'NO_CONDITION_COLUMN',      -- Cannot identify condition column
    'NO_ACTION_COLUMN',         -- Cannot identify action column
    'COMPLEX_STRUCTURE',        -- Multi-level headers, merged cells, etc.
    'NARRATIVE_CONTENT',        -- Table contains prose, not structured data
    'AMBIGUOUS_UNITS',          -- Cannot determine units/variables
    'MISSING_CONTEXT',          -- Needs surrounding text for interpretation
    'MULTIPLE_CONDITIONS',      -- Complex multi-variable conditions
    'INVALID_FORMAT',           -- Malformed table structure
    'OTHER'                     -- Other reasons (specify in notes)
);

CREATE TYPE resolution_outcome AS ENUM (
    'MANUALLY_TRANSLATED',      -- Human created the rule
    'NOT_A_RULE',               -- Table doesn't contain clinical rules
    'DUPLICATE',                -- Already exists in system
    'DEFERRED',                 -- Will handle in future version
    'INVALID'                   -- Table data is incorrect/outdated
);

CREATE TABLE IF NOT EXISTS untranslatable_queue (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Table identification
    table_id VARCHAR(100) NOT NULL,
    headers TEXT[] NOT NULL,                        -- Original column headers
    row_count INT NOT NULL,
    sample_rows JSONB,                              -- First few rows for review

    -- Source information
    source_document_id UUID NOT NULL,
    source_section_id UUID,
    source_info VARCHAR(255),                       -- e.g., "NDC12345/34073-1"
    table_type VARCHAR(50),                         -- Original table classification

    -- Why it's untranslatable
    reason untranslatable_reason NOT NULL,
    reason_details TEXT,                            -- Specific explanation

    -- Status and workflow
    status untranslatable_status NOT NULL DEFAULT 'PENDING',
    priority review_priority NOT NULL DEFAULT 'STANDARD',

    -- Assignment
    assigned_to VARCHAR(255),
    assigned_at TIMESTAMPTZ,

    -- Resolution
    resolution_outcome resolution_outcome,
    resolution_notes TEXT,
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(255),
    resulting_rule_id UUID REFERENCES draft_rules(rule_id),

    -- SLA tracking
    sla_deadline TIMESTAMPTZ NOT NULL,              -- When review must complete
    sla_breached BOOLEAN NOT NULL DEFAULT FALSE,

    -- Lifecycle
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for untranslatable_queue
CREATE INDEX idx_untranslatable_status ON untranslatable_queue(status);
CREATE INDEX idx_untranslatable_priority ON untranslatable_queue(priority);
CREATE INDEX idx_untranslatable_assigned ON untranslatable_queue(assigned_to);
CREATE INDEX idx_untranslatable_source ON untranslatable_queue(source_document_id);
CREATE INDEX idx_untranslatable_sla ON untranslatable_queue(sla_deadline);
CREATE INDEX idx_untranslatable_reason ON untranslatable_queue(reason);
CREATE INDEX idx_untranslatable_pending ON untranslatable_queue(status, priority)
    WHERE status = 'PENDING';

-- =============================================================================
-- TRANSLATION STATISTICS
-- =============================================================================
-- Aggregate statistics for monitoring translation pipeline performance

CREATE TABLE IF NOT EXISTS translation_stats (
    -- Primary key
    id SERIAL PRIMARY KEY,

    -- Time window
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,

    -- Processing metrics
    tables_processed INT NOT NULL DEFAULT 0,
    tables_translated INT NOT NULL DEFAULT 0,
    tables_untranslatable INT NOT NULL DEFAULT 0,
    tables_skipped INT NOT NULL DEFAULT 0,

    -- Rule metrics
    rules_generated INT NOT NULL DEFAULT 0,
    duplicates_skipped INT NOT NULL DEFAULT 0,

    -- Row metrics
    rows_processed INT NOT NULL DEFAULT 0,
    rows_skipped INT NOT NULL DEFAULT 0,

    -- Quality metrics
    average_confidence DECIMAL(3,2),
    min_confidence DECIMAL(3,2),
    max_confidence DECIMAL(3,2),

    -- Performance
    processing_time_ms BIGINT,

    -- Breakdown by domain
    stats_by_domain JSONB NOT NULL DEFAULT '{}'::jsonb,
    stats_by_rule_type JSONB NOT NULL DEFAULT '{}'::jsonb,
    stats_by_reason JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for time-based queries
CREATE INDEX idx_translation_stats_window ON translation_stats(window_start, window_end);

-- =============================================================================
-- VIEWS FOR GOVERNANCE INTEGRATION
-- =============================================================================

-- View: Pending rules for governance review
CREATE OR REPLACE VIEW v_draft_rules_pending_review AS
SELECT
    r.rule_id,
    r.domain,
    r.rule_type,
    r.condition_variable,
    r.condition_operator,
    r.action_effect,
    r.action_severity,
    r.confidence,
    r.fingerprint_hash,
    r.governance_status,
    r.created_at,
    r.source_type,
    r.document_id,
    r.table_id
FROM draft_rules r
WHERE r.governance_status IN ('DRAFT', 'PENDING_REVIEW')
  AND r.is_latest = TRUE
ORDER BY
    CASE r.action_severity
        WHEN 'CRITICAL' THEN 1
        WHEN 'HIGH' THEN 2
        WHEN 'MODERATE' THEN 3
        WHEN 'LOW' THEN 4
        ELSE 5
    END,
    r.confidence DESC,
    r.created_at ASC;

-- View: Untranslatable queue for review dashboard
CREATE OR REPLACE VIEW v_untranslatable_dashboard AS
SELECT
    q.id,
    q.table_id,
    q.reason,
    q.reason_details,
    q.status,
    q.priority,
    q.assigned_to,
    q.sla_deadline,
    q.sla_breached,
    q.created_at,
    q.row_count,
    q.headers,
    q.source_info,
    CASE
        WHEN q.sla_deadline < NOW() AND q.status = 'PENDING' THEN TRUE
        ELSE FALSE
    END AS sla_at_risk
FROM untranslatable_queue q
WHERE q.status IN ('PENDING', 'IN_REVIEW')
ORDER BY
    CASE q.priority
        WHEN 'CRITICAL' THEN 1
        WHEN 'HIGH' THEN 2
        WHEN 'STANDARD' THEN 3
        WHEN 'LOW' THEN 4
    END,
    q.sla_deadline ASC;

-- View: Fingerprint statistics
CREATE OR REPLACE VIEW v_fingerprint_stats AS
SELECT
    domain,
    rule_type::text,
    COUNT(*) AS unique_rules,
    SUM(source_count) AS total_sources,
    ROUND(1.0 - (COUNT(*)::numeric / NULLIF(SUM(source_count), 0)), 3) AS deduplication_rate
FROM fingerprint_registry
GROUP BY domain, rule_type
ORDER BY domain, rule_type;

-- =============================================================================
-- FUNCTIONS
-- =============================================================================

-- Function: Update timestamp trigger
CREATE OR REPLACE FUNCTION update_draft_rule_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Auto-update timestamps on draft_rules
DROP TRIGGER IF EXISTS trg_draft_rules_updated ON draft_rules;
CREATE TRIGGER trg_draft_rules_updated
    BEFORE UPDATE ON draft_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_draft_rule_timestamp();

-- Trigger: Auto-update timestamps on untranslatable_queue
DROP TRIGGER IF EXISTS trg_untranslatable_updated ON untranslatable_queue;
CREATE TRIGGER trg_untranslatable_updated
    BEFORE UPDATE ON untranslatable_queue
    FOR EACH ROW
    EXECUTE FUNCTION update_draft_rule_timestamp();

-- Function: Check and set SLA breach
CREATE OR REPLACE FUNCTION check_sla_breach()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.sla_deadline < NOW() AND NEW.status = 'PENDING' THEN
        NEW.sla_breached = TRUE;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger: Auto-check SLA on status change
DROP TRIGGER IF EXISTS trg_untranslatable_sla ON untranslatable_queue;
CREATE TRIGGER trg_untranslatable_sla
    BEFORE UPDATE ON untranslatable_queue
    FOR EACH ROW
    EXECUTE FUNCTION check_sla_breach();

-- Function: Increment fingerprint source count
CREATE OR REPLACE FUNCTION increment_fingerprint_source(
    p_hash VARCHAR(64),
    p_source_doc_id UUID
)
RETURNS VOID AS $$
BEGIN
    UPDATE fingerprint_registry
    SET source_count = source_count + 1,
        sources = sources || jsonb_build_array(jsonb_build_object('doc_id', p_source_doc_id, 'added_at', NOW()))
    WHERE hash = p_hash;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE draft_rules IS
    'Canonical computable rules extracted from clinical sources (Phase 3b.5)';
COMMENT ON TABLE fingerprint_registry IS
    'Semantic fingerprints for rule deduplication - same clinical meaning = same hash';
COMMENT ON TABLE untranslatable_queue IS
    'Human review queue for tables that cannot be automatically translated (NOT LLM)';
COMMENT ON TABLE translation_stats IS
    'Aggregate metrics for translation pipeline monitoring';

COMMENT ON COLUMN draft_rules.fingerprint_hash IS
    'SHA256 of canonical JSON (domain + ruleType + condition + action)';
COMMENT ON COLUMN draft_rules.confidence IS
    'Extraction confidence score (0.00-1.00)';
COMMENT ON COLUMN untranslatable_queue.sla_deadline IS
    'Review must complete by this time (default: 72h for standard priority)';

COMMIT;

-- =============================================================================
-- POST-MIGRATION VERIFICATION
-- =============================================================================
-- Run these queries to verify the migration succeeded:

-- SELECT COUNT(*) FROM information_schema.tables
-- WHERE table_schema = 'public'
-- AND table_name IN ('draft_rules', 'fingerprint_registry', 'untranslatable_queue', 'translation_stats');
-- Expected: 4

-- SELECT COUNT(*) FROM information_schema.views
-- WHERE table_schema = 'public'
-- AND table_name IN ('v_draft_rules_pending_review', 'v_untranslatable_dashboard', 'v_fingerprint_stats');
-- Expected: 3
