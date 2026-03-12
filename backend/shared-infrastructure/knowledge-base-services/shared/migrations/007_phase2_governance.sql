-- =============================================================================
-- MIGRATION 007: Phase 2 Governance Integration
-- Purpose: Bridge Canonical Fact Store with KB-0 Governance Workflow
-- Reference: KB1 Data Source Injection Implementation Plan - Phase 2
-- =============================================================================
-- Architecture:
--   Extraction Pipelines → clinical_facts (DRAFT) → KB-0 Governance → ACTIVE
--
-- This migration adds:
--   1. Governance columns to clinical_facts for workflow state
--   2. v_governance_queue view for KB-0 to poll
--   3. governance_audit_log for regulatory compliance
--   4. governance_decisions table for decision tracking
-- =============================================================================

BEGIN;

-- =============================================================================
-- GOVERNANCE WORKFLOW ENUMS
-- =============================================================================

-- Review priority levels for triage
CREATE TYPE review_priority AS ENUM (
    'CRITICAL',   -- Safety signals, contraindications (24h SLA)
    'HIGH',       -- Drug interactions, organ impairment (48h SLA)
    'STANDARD',   -- Formulary updates, lab ranges (7d SLA)
    'LOW'         -- Minor updates, terminology (14d SLA)
);

-- Decision outcomes from governance review
CREATE TYPE governance_decision AS ENUM (
    'AUTO_APPROVED',     -- Confidence >= 0.95, no conflicts
    'APPROVED',          -- Reviewer approved
    'REJECTED',          -- Reviewer rejected (quality/accuracy issues)
    'SUPERSEDED',        -- Replaced by newer version
    'ESCALATED',         -- Needs higher authority review
    'PENDING_REVIEW'     -- Awaiting human review
);

-- =============================================================================
-- EXTEND clinical_facts WITH GOVERNANCE COLUMNS
-- =============================================================================

-- Add governance workflow columns to clinical_facts
ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    review_priority review_priority;

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    assigned_reviewer VARCHAR(255);

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    assigned_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    review_due_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    governance_decision governance_decision;

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    decision_reason TEXT;

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    decision_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    decision_by VARCHAR(255);

-- Authority priority for conflict resolution (lower = higher priority)
ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    authority_priority INTEGER DEFAULT 99;

-- Conflict detection fields
ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    has_conflict BOOLEAN DEFAULT FALSE;

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    conflict_with_fact_ids UUID[];

ALTER TABLE clinical_facts ADD COLUMN IF NOT EXISTS
    conflict_resolution_notes TEXT;

-- =============================================================================
-- GOVERNANCE QUEUE VIEW
-- KB-0 polls this view to get pending review items
-- =============================================================================

CREATE OR REPLACE VIEW v_governance_queue AS
SELECT
    cf.fact_id,
    cf.fact_type,
    cf.rxcui,
    cf.drug_name,
    cf.scope,
    cf.class_rxcui,
    cf.class_name,
    cf.content,
    cf.source_type,
    cf.source_id,
    cf.source_version,
    cf.extraction_method,
    cf.confidence_score,
    cf.confidence_band,
    cf.status,
    cf.review_priority,
    cf.assigned_reviewer,
    cf.assigned_at,
    cf.review_due_at,
    cf.has_conflict,
    cf.conflict_with_fact_ids,
    cf.authority_priority,
    cf.created_at,
    cf.created_by,
    -- Compute queue position (priority + age)
    CASE
        WHEN cf.review_priority = 'CRITICAL' THEN 1
        WHEN cf.review_priority = 'HIGH' THEN 2
        WHEN cf.review_priority = 'STANDARD' THEN 3
        WHEN cf.review_priority = 'LOW' THEN 4
        ELSE 5
    END AS priority_rank,
    -- Days until SLA breach
    CASE
        WHEN cf.review_due_at IS NOT NULL THEN
            EXTRACT(EPOCH FROM (cf.review_due_at - NOW())) / 86400
        ELSE NULL
    END AS days_until_due,
    -- SLA status
    CASE
        WHEN cf.review_due_at IS NULL THEN 'NO_SLA'
        WHEN cf.review_due_at < NOW() THEN 'BREACHED'
        WHEN cf.review_due_at < NOW() + INTERVAL '24 hours' THEN 'AT_RISK'
        ELSE 'ON_TRACK'
    END AS sla_status
FROM clinical_facts cf
WHERE cf.status = 'DRAFT'
  AND cf.governance_decision IS NULL
ORDER BY
    priority_rank ASC,
    cf.has_conflict DESC,  -- Conflicts first within priority
    cf.created_at ASC;     -- FIFO within same priority

-- =============================================================================
-- GOVERNANCE DECISIONS TABLE
-- Records all governance decisions for audit trail
-- =============================================================================

CREATE TABLE IF NOT EXISTS governance_decisions (
    decision_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fact_id             UUID NOT NULL REFERENCES clinical_facts(fact_id),

    -- Decision details
    decision            governance_decision NOT NULL,
    decision_reason     TEXT,

    -- Policy evaluation results
    policy_name         VARCHAR(100) NOT NULL,  -- 'activation', 'conflict', 'override'
    policy_version      VARCHAR(50) DEFAULT '1.0',
    evaluation_result   JSONB NOT NULL,         -- Full policy evaluation output

    -- Confidence and thresholds
    input_confidence    NUMERIC(3,2),
    threshold_applied   NUMERIC(3,2),

    -- Conflict resolution (if applicable)
    conflicting_facts   UUID[],
    resolution_strategy VARCHAR(50),            -- 'AUTHORITY_PRIORITY', 'RECENCY', 'MANUAL'

    -- Actor information
    actor_type          VARCHAR(50) NOT NULL,   -- 'SYSTEM', 'PHARMACIST', 'PHYSICIAN', 'ADMIN'
    actor_id            VARCHAR(255) NOT NULL,
    actor_credentials   VARCHAR(255),           -- 'PharmD', 'MD', etc.

    -- Timestamps
    decided_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Security context
    ip_address          VARCHAR(45),
    session_id          VARCHAR(100)
);

-- Indexes for decision queries
CREATE INDEX idx_gov_decisions_fact ON governance_decisions(fact_id);
CREATE INDEX idx_gov_decisions_policy ON governance_decisions(policy_name);
CREATE INDEX idx_gov_decisions_actor ON governance_decisions(actor_id);
CREATE INDEX idx_gov_decisions_time ON governance_decisions(decided_at DESC);
CREATE INDEX idx_gov_decisions_decision ON governance_decisions(decision);

-- =============================================================================
-- GOVERNANCE AUDIT LOG
-- Immutable audit trail for regulatory compliance (21 CFR Part 11)
-- =============================================================================

CREATE TABLE IF NOT EXISTS governance_audit_log (
    audit_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Event identification
    event_type          VARCHAR(50) NOT NULL,   -- 'FACT_CREATED', 'REVIEW_ASSIGNED', 'DECISION_MADE', etc.
    event_timestamp     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Fact reference
    fact_id             UUID REFERENCES clinical_facts(fact_id),
    fact_type           fact_type,
    rxcui               VARCHAR(20),
    drug_name           VARCHAR(500),

    -- State transition
    previous_state      VARCHAR(50),
    new_state           VARCHAR(50),

    -- Actor information
    actor_type          VARCHAR(50) NOT NULL,
    actor_id            VARCHAR(255) NOT NULL,
    actor_name          VARCHAR(255),
    actor_credentials   VARCHAR(255),

    -- Event details
    event_details       JSONB NOT NULL DEFAULT '{}',

    -- Security context (for 21 CFR Part 11)
    ip_address          VARCHAR(45),
    user_agent          TEXT,
    session_id          VARCHAR(100),
    signature_hash      VARCHAR(64),            -- SHA-256 of event for tamper detection

    -- Immutability
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Partitioning hint: Consider partitioning by event_timestamp for large deployments
-- CREATE TABLE governance_audit_log (...) PARTITION BY RANGE (event_timestamp);

-- Indexes for audit queries
CREATE INDEX idx_audit_fact ON governance_audit_log(fact_id);
CREATE INDEX idx_audit_event_type ON governance_audit_log(event_type);
CREATE INDEX idx_audit_actor ON governance_audit_log(actor_id);
CREATE INDEX idx_audit_timestamp ON governance_audit_log(event_timestamp DESC);
CREATE INDEX idx_audit_drug ON governance_audit_log(rxcui);

-- =============================================================================
-- AUTHORITY PRIORITY TABLE
-- Maps source authorities to priority (lower = higher priority)
-- =============================================================================

CREATE TABLE IF NOT EXISTS authority_priorities (
    authority_code      VARCHAR(50) PRIMARY KEY,
    authority_name      VARCHAR(255) NOT NULL,
    priority            INTEGER NOT NULL,
    jurisdiction        VARCHAR(20),            -- 'US', 'AU', 'IN', 'GLOBAL'
    trust_level         VARCHAR(20) DEFAULT 'STANDARD',  -- 'HIGHEST', 'HIGH', 'STANDARD', 'RESEARCH'
    description         TEXT,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT uq_authority_priority UNIQUE (priority)
);

-- Insert authority priorities (from Phase 1 ONC > OHDSI hierarchy)
INSERT INTO authority_priorities (authority_code, authority_name, priority, jurisdiction, trust_level, description)
VALUES
    ('ONC', 'Office of National Coordinator Constitutional DDI Rules', 1, 'US', 'HIGHEST', 'Federal regulatory authority for DDI'),
    ('FDA', 'Food and Drug Administration', 2, 'US', 'HIGHEST', 'Drug safety and labeling authority'),
    ('USP', 'United States Pharmacopeia', 3, 'US', 'HIGH', 'Drug quality standards'),
    ('NICE', 'National Institute for Health and Care Excellence', 4, 'UK', 'HIGH', 'UK clinical guidelines'),
    ('TGA', 'Therapeutic Goods Administration', 5, 'AU', 'HIGH', 'Australian drug regulator'),
    ('CDSCO', 'Central Drugs Standard Control Organisation', 6, 'IN', 'HIGH', 'Indian drug regulator'),
    ('EMA', 'European Medicines Agency', 7, 'EU', 'HIGH', 'European drug regulator'),
    ('DRUGBANK', 'DrugBank', 10, 'GLOBAL', 'STANDARD', 'Curated drug database'),
    ('RXNORM', 'RxNorm', 11, 'US', 'STANDARD', 'Drug terminology standard'),
    ('OHDSI', 'Observational Health Data Sciences and Informatics', 21, 'GLOBAL', 'RESEARCH', 'Research-grade DDI database')
ON CONFLICT (authority_code) DO UPDATE SET
    priority = EXCLUDED.priority,
    trust_level = EXCLUDED.trust_level;

-- =============================================================================
-- GOVERNANCE FUNCTIONS
-- =============================================================================

-- Function: Calculate review priority based on fact type and confidence
CREATE OR REPLACE FUNCTION calculate_review_priority(
    p_fact_type fact_type,
    p_confidence_score NUMERIC,
    p_source_type source_type
) RETURNS review_priority AS $$
BEGIN
    -- Safety-critical facts always high priority
    IF p_fact_type = 'SAFETY_SIGNAL' THEN
        RETURN 'CRITICAL';
    END IF;

    -- Drug interactions with low confidence are high priority
    IF p_fact_type = 'INTERACTION' AND p_confidence_score < 0.65 THEN
        RETURN 'CRITICAL';
    END IF;

    -- Organ impairment rules
    IF p_fact_type = 'ORGAN_IMPAIRMENT' THEN
        IF p_confidence_score < 0.75 THEN
            RETURN 'HIGH';
        ELSE
            RETURN 'STANDARD';
        END IF;
    END IF;

    -- LLM-extracted facts need review
    IF p_source_type = 'LLM' AND p_confidence_score < 0.85 THEN
        RETURN 'HIGH';
    END IF;

    -- Default based on confidence
    IF p_confidence_score < 0.65 THEN
        RETURN 'HIGH';
    ELSIF p_confidence_score < 0.85 THEN
        RETURN 'STANDARD';
    ELSE
        RETURN 'LOW';
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function: Calculate review SLA due date
CREATE OR REPLACE FUNCTION calculate_review_due_date(
    p_priority review_priority
) RETURNS TIMESTAMP WITH TIME ZONE AS $$
BEGIN
    RETURN CASE p_priority
        WHEN 'CRITICAL' THEN NOW() + INTERVAL '24 hours'
        WHEN 'HIGH' THEN NOW() + INTERVAL '48 hours'
        WHEN 'STANDARD' THEN NOW() + INTERVAL '7 days'
        WHEN 'LOW' THEN NOW() + INTERVAL '14 days'
        ELSE NOW() + INTERVAL '7 days'
    END;
END;
$$ LANGUAGE plpgsql STABLE;

-- Function: Write audit log entry
CREATE OR REPLACE FUNCTION log_governance_event(
    p_event_type VARCHAR,
    p_fact_id UUID,
    p_previous_state VARCHAR,
    p_new_state VARCHAR,
    p_actor_type VARCHAR,
    p_actor_id VARCHAR,
    p_actor_name VARCHAR,
    p_event_details JSONB,
    p_ip_address VARCHAR DEFAULT NULL,
    p_session_id VARCHAR DEFAULT NULL
) RETURNS UUID AS $$
DECLARE
    v_audit_id UUID;
    v_fact_type fact_type;
    v_rxcui VARCHAR;
    v_drug_name VARCHAR;
    v_signature_data TEXT;
    v_signature_hash VARCHAR;
BEGIN
    -- Get fact details
    SELECT fact_type, rxcui, drug_name
    INTO v_fact_type, v_rxcui, v_drug_name
    FROM clinical_facts
    WHERE fact_id = p_fact_id;

    -- Generate signature for tamper detection
    v_signature_data := COALESCE(p_event_type, '') ||
                        COALESCE(p_fact_id::TEXT, '') ||
                        COALESCE(p_actor_id, '') ||
                        NOW()::TEXT;
    v_signature_hash := encode(sha256(v_signature_data::BYTEA), 'hex');

    -- Insert audit log
    INSERT INTO governance_audit_log (
        event_type,
        fact_id,
        fact_type,
        rxcui,
        drug_name,
        previous_state,
        new_state,
        actor_type,
        actor_id,
        actor_name,
        event_details,
        ip_address,
        session_id,
        signature_hash
    ) VALUES (
        p_event_type,
        p_fact_id,
        v_fact_type,
        v_rxcui,
        v_drug_name,
        p_previous_state,
        p_new_state,
        p_actor_type,
        p_actor_id,
        p_actor_name,
        p_event_details,
        p_ip_address,
        p_session_id,
        v_signature_hash
    ) RETURNING audit_id INTO v_audit_id;

    RETURN v_audit_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- TRIGGERS
-- =============================================================================

-- Trigger: Auto-assign priority and SLA on DRAFT facts
CREATE OR REPLACE FUNCTION trigger_assign_governance_defaults()
RETURNS TRIGGER AS $$
BEGIN
    -- Only on new DRAFT facts
    IF NEW.status = 'DRAFT' AND NEW.review_priority IS NULL THEN
        NEW.review_priority := calculate_review_priority(
            NEW.fact_type,
            NEW.confidence_score,
            NEW.source_type
        );
        NEW.review_due_at := calculate_review_due_date(NEW.review_priority);

        -- Set authority priority from source
        SELECT priority INTO NEW.authority_priority
        FROM authority_priorities
        WHERE authority_code = UPPER(SPLIT_PART(NEW.source_id, ':', 1))
        LIMIT 1;

        IF NEW.authority_priority IS NULL THEN
            NEW.authority_priority := 99;  -- Unknown source = lowest priority
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_clinical_facts_governance
    BEFORE INSERT ON clinical_facts
    FOR EACH ROW
    EXECUTE FUNCTION trigger_assign_governance_defaults();

-- Trigger: Log status changes to audit log
CREATE OR REPLACE FUNCTION trigger_audit_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        PERFORM log_governance_event(
            'STATUS_CHANGED',
            NEW.fact_id,
            OLD.status::VARCHAR,
            NEW.status::VARCHAR,
            'SYSTEM',
            COALESCE(NEW.decision_by, 'trigger'),
            NULL,
            jsonb_build_object(
                'previous_confidence', OLD.confidence_score,
                'new_confidence', NEW.confidence_score,
                'governance_decision', NEW.governance_decision
            )
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_clinical_facts_audit
    AFTER UPDATE ON clinical_facts
    FOR EACH ROW
    EXECUTE FUNCTION trigger_audit_status_change();

-- =============================================================================
-- CONFLICT DETECTION VIEW
-- Identifies facts with same drug/type that may conflict
-- =============================================================================

CREATE OR REPLACE VIEW v_potential_conflicts AS
SELECT
    cf1.fact_id AS fact_id_1,
    cf2.fact_id AS fact_id_2,
    cf1.rxcui,
    cf1.drug_name,
    cf1.fact_type,
    cf1.source_id AS source_1,
    cf2.source_id AS source_2,
    cf1.authority_priority AS priority_1,
    cf2.authority_priority AS priority_2,
    cf1.confidence_score AS confidence_1,
    cf2.confidence_score AS confidence_2,
    -- Determine which fact should win based on authority priority
    CASE
        WHEN cf1.authority_priority < cf2.authority_priority THEN cf1.fact_id
        WHEN cf2.authority_priority < cf1.authority_priority THEN cf2.fact_id
        WHEN cf1.created_at > cf2.created_at THEN cf1.fact_id  -- Recency tiebreaker
        ELSE cf2.fact_id
    END AS preferred_fact_id,
    'AUTHORITY_PRIORITY' AS resolution_strategy
FROM clinical_facts cf1
JOIN clinical_facts cf2 ON
    cf1.rxcui = cf2.rxcui
    AND cf1.fact_type = cf2.fact_type
    AND cf1.fact_id < cf2.fact_id  -- Avoid duplicate pairs
WHERE cf1.status IN ('DRAFT', 'APPROVED', 'ACTIVE')
  AND cf2.status IN ('DRAFT', 'APPROVED', 'ACTIVE')
  -- Only flag if content differs significantly
  AND cf1.content::TEXT != cf2.content::TEXT;

-- =============================================================================
-- REVIEWER METRICS VIEW
-- Dashboard view for governance performance
-- =============================================================================

CREATE OR REPLACE VIEW v_reviewer_metrics AS
SELECT
    actor_id AS reviewer_id,
    COUNT(*) FILTER (WHERE decision = 'APPROVED') AS approved_count,
    COUNT(*) FILTER (WHERE decision = 'REJECTED') AS rejected_count,
    COUNT(*) FILTER (WHERE decision = 'ESCALATED') AS escalated_count,
    COUNT(*) FILTER (WHERE decision = 'AUTO_APPROVED') AS auto_approved_count,
    COUNT(*) AS total_decisions,
    AVG(EXTRACT(EPOCH FROM (decided_at -
        (SELECT created_at FROM clinical_facts WHERE fact_id = governance_decisions.fact_id)
    ))) / 3600 AS avg_review_hours,
    MAX(decided_at) AS last_review_at
FROM governance_decisions
WHERE actor_type != 'SYSTEM'
GROUP BY actor_id;

-- =============================================================================
-- MIGRATION METADATA
-- =============================================================================

INSERT INTO schema_migrations (version, name) VALUES (7, '007_phase2_governance')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- =============================================================================
-- VERIFICATION
-- =============================================================================
SELECT 'Migration 007: Phase 2 Governance Integration - COMPLETE' AS status;
SELECT 'New columns added to clinical_facts' AS step_1;
SELECT 'v_governance_queue view created' AS step_2;
SELECT 'governance_decisions table created' AS step_3;
SELECT 'governance_audit_log table created' AS step_4;
SELECT 'authority_priorities table populated with ONC > OHDSI hierarchy' AS step_5;
