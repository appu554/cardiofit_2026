-- ============================================================================
-- Phase 3: Governance Queue and Facts Tables for Auto-Governance Engine
-- ============================================================================
-- This migration adds the tables required by the governance engine (engine.go)
-- for automated fact lifecycle management with confidence-based approval.
--
-- GOVERNANCE THRESHOLDS:
-- - ≥0.85: Auto-approve and activate
-- - 0.65-0.84: Queue for human review
-- - <0.65: Auto-reject
-- ============================================================================

-- ============================================================================
-- FACTS TABLE (Unified Fact Store)
-- ============================================================================
-- This is the activated facts table - facts that have passed governance

CREATE TABLE IF NOT EXISTS facts (
    fact_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Fact identification
    fact_type VARCHAR(100) NOT NULL,        -- RENAL_DOSE_ADJUST, QT_RISK, etc.
    rxcui VARCHAR(50),                       -- RxNorm concept ID
    drug_name VARCHAR(255),                  -- Human-readable drug name

    -- Content
    content JSONB NOT NULL,                  -- Structured fact content

    -- Confidence scores (multi-dimensional)
    confidence_overall DECIMAL(5,4) NOT NULL,
    confidence_source DECIMAL(5,4),          -- Source quality score
    confidence_extraction DECIMAL(5,4),      -- Extraction certainty score

    -- Lifecycle
    status VARCHAR(20) DEFAULT 'DRAFT',      -- DRAFT, ACTIVE, DEPRECATED, SUPERSEDED
    effective_from TIMESTAMPTZ DEFAULT NOW(),
    effective_until TIMESTAMPTZ,

    -- Extractor metadata
    extractor_id VARCHAR(100),               -- Which extractor created this
    extractor_version VARCHAR(50),           -- Extractor version

    -- Governance tracking
    governance_decision VARCHAR(50),          -- AUTO_APPROVED, REVIEW_REQUIRED, AUTO_REJECTED, HUMAN_APPROVED, HUMAN_REJECTED
    governance_timestamp TIMESTAMPTZ,
    reviewed_by VARCHAR(100),
    review_notes TEXT,

    -- Lineage (links to derived_facts)
    derived_fact_id UUID REFERENCES derived_facts(id),
    source_document_id UUID REFERENCES source_documents(id),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for facts table
CREATE INDEX IF NOT EXISTS idx_facts_type ON facts(fact_type);
CREATE INDEX IF NOT EXISTS idx_facts_rxcui ON facts(rxcui);
CREATE INDEX IF NOT EXISTS idx_facts_status ON facts(status);
CREATE INDEX IF NOT EXISTS idx_facts_drug ON facts(drug_name);
CREATE INDEX IF NOT EXISTS idx_facts_confidence ON facts(confidence_overall);
CREATE INDEX IF NOT EXISTS idx_facts_governance ON facts(governance_decision);
CREATE INDEX IF NOT EXISTS idx_facts_content ON facts USING GIN(content);

-- ============================================================================
-- GOVERNANCE QUEUE TABLE
-- ============================================================================
-- Queue for facts requiring human review (confidence 0.65-0.84)

CREATE TABLE IF NOT EXISTS governance_queue (
    queue_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fact_id UUID NOT NULL REFERENCES facts(fact_id),

    -- Queue metadata
    priority INTEGER DEFAULT 5,               -- 1-10, higher = more urgent
    confidence_score DECIMAL(5,4) NOT NULL,
    fact_type VARCHAR(100) NOT NULL,
    rxcui VARCHAR(50),
    drug_name VARCHAR(255),

    -- Timing
    queued_at TIMESTAMPTZ DEFAULT NOW(),
    review_deadline TIMESTAMPTZ,

    -- Assignment
    assigned_to VARCHAR(100),
    assigned_at TIMESTAMPTZ,

    -- Escalation
    escalation_count INTEGER DEFAULT 0,
    escalated_at TIMESTAMPTZ,

    -- Resolution
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMPTZ,
    resolution_decision VARCHAR(50),          -- APPROVED, REJECTED, ESCALATED
    resolution_notes TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_fact_in_queue UNIQUE (fact_id)
);

-- Indexes for governance queue
CREATE INDEX IF NOT EXISTS idx_gov_queue_resolved ON governance_queue(resolved);
CREATE INDEX IF NOT EXISTS idx_gov_queue_priority ON governance_queue(priority DESC);
CREATE INDEX IF NOT EXISTS idx_gov_queue_assigned ON governance_queue(assigned_to);
CREATE INDEX IF NOT EXISTS idx_gov_queue_deadline ON governance_queue(review_deadline);
CREATE INDEX IF NOT EXISTS idx_gov_queue_escalation ON governance_queue(escalation_count);

-- ============================================================================
-- GOVERNANCE AUDIT LOG
-- ============================================================================
-- Complete audit trail of all governance decisions

CREATE TABLE IF NOT EXISTS governance_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fact_id UUID REFERENCES facts(fact_id),

    -- Decision details
    decision VARCHAR(50) NOT NULL,            -- AUTO_APPROVED, REVIEW_REQUIRED, AUTO_REJECTED, etc.
    decision_reason TEXT,
    confidence_at_decision DECIMAL(5,4),

    -- Actor
    actor_type VARCHAR(20) NOT NULL,          -- SYSTEM, HUMAN
    actor_id VARCHAR(100),

    -- Timing
    decided_at TIMESTAMPTZ DEFAULT NOW(),

    -- Additional context
    metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_gov_audit_fact ON governance_audit_log(fact_id);
CREATE INDEX IF NOT EXISTS idx_gov_audit_decision ON governance_audit_log(decision);
CREATE INDEX IF NOT EXISTS idx_gov_audit_time ON governance_audit_log(decided_at);

-- ============================================================================
-- FUNCTION: Link derived_facts to facts table
-- ============================================================================
-- When a derived_fact is approved, create corresponding entry in facts table

CREATE OR REPLACE FUNCTION activate_derived_fact(
    p_derived_fact_id UUID,
    p_governance_decision VARCHAR(50),
    p_reviewed_by VARCHAR(100) DEFAULT 'SYSTEM'
) RETURNS UUID AS $$
DECLARE
    v_fact_id UUID;
    v_derived RECORD;
BEGIN
    -- Get the derived fact
    SELECT * INTO v_derived
    FROM derived_facts
    WHERE id = p_derived_fact_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Derived fact not found: %', p_derived_fact_id;
    END IF;

    -- Create activated fact
    INSERT INTO facts (
        fact_type, rxcui, drug_name, content,
        confidence_overall, confidence_source, confidence_extraction,
        status, governance_decision, governance_timestamp, reviewed_by,
        derived_fact_id, source_document_id
    )
    SELECT
        df.fact_type,
        sd.rxcui,
        sd.drug_name,
        df.fact_data,
        df.extraction_confidence,
        df.extraction_confidence,
        df.extraction_confidence,
        CASE WHEN p_governance_decision IN ('AUTO_APPROVED', 'HUMAN_APPROVED')
             THEN 'ACTIVE' ELSE 'DEPRECATED' END,
        p_governance_decision,
        NOW(),
        p_reviewed_by,
        df.id,
        df.source_document_id
    FROM derived_facts df
    JOIN source_documents sd ON df.source_document_id = sd.id
    WHERE df.id = p_derived_fact_id
    RETURNING fact_id INTO v_fact_id;

    -- Update derived_fact governance status
    UPDATE derived_facts
    SET governance_status =
        CASE WHEN p_governance_decision IN ('AUTO_APPROVED', 'HUMAN_APPROVED')
             THEN 'APPROVED'
             WHEN p_governance_decision = 'REVIEW_REQUIRED'
             THEN 'PENDING_REVIEW'
             ELSE 'REJECTED' END,
        reviewed_by = p_reviewed_by,
        reviewed_at = NOW(),
        updated_at = NOW()
    WHERE id = p_derived_fact_id;

    -- Log the governance decision
    INSERT INTO governance_audit_log (
        fact_id, decision, confidence_at_decision, actor_type, actor_id
    ) VALUES (
        v_fact_id, p_governance_decision, v_derived.extraction_confidence,
        CASE WHEN p_reviewed_by = 'SYSTEM' THEN 'SYSTEM' ELSE 'HUMAN' END,
        p_reviewed_by
    );

    RETURN v_fact_id;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- VIEW: Active Facts with Full Lineage
-- ============================================================================

CREATE OR REPLACE VIEW v_active_facts AS
SELECT
    f.fact_id,
    f.fact_type,
    f.rxcui,
    f.drug_name,
    f.content,
    f.confidence_overall,
    f.status,
    f.governance_decision,
    df.extraction_method,
    df.target_kb,
    sd.source_type,
    sd.document_id AS source_document,
    ss.section_code,
    ss.section_name,
    f.effective_from,
    f.created_at
FROM facts f
LEFT JOIN derived_facts df ON f.derived_fact_id = df.id
LEFT JOIN source_documents sd ON f.source_document_id = sd.id
LEFT JOIN source_sections ss ON df.source_section_id = ss.id
WHERE f.status = 'ACTIVE';

-- ============================================================================
-- VIEW: Governance Queue Dashboard
-- ============================================================================

CREATE OR REPLACE VIEW v_governance_dashboard AS
SELECT
    gq.queue_id,
    gq.fact_id,
    gq.priority,
    gq.confidence_score,
    gq.fact_type,
    gq.drug_name,
    gq.assigned_to,
    gq.review_deadline,
    gq.escalation_count,
    CASE
        WHEN gq.review_deadline < NOW() THEN 'OVERDUE'
        WHEN gq.assigned_to IS NOT NULL THEN 'IN_REVIEW'
        ELSE 'PENDING'
    END AS queue_status,
    f.content,
    df.extraction_method,
    df.evidence_spans
FROM governance_queue gq
JOIN facts f ON gq.fact_id = f.fact_id
LEFT JOIN derived_facts df ON f.derived_fact_id = df.id
WHERE gq.resolved = FALSE
ORDER BY
    CASE WHEN gq.review_deadline < NOW() THEN 0 ELSE 1 END,
    gq.priority DESC,
    gq.queued_at;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE facts IS 'Activated facts that have passed governance - the production fact store';
COMMENT ON TABLE governance_queue IS 'Queue of facts requiring human review (confidence 0.65-0.84)';
COMMENT ON TABLE governance_audit_log IS 'Complete audit trail of all governance decisions';
COMMENT ON FUNCTION activate_derived_fact IS 'Links a derived_fact to the facts table with governance decision';
