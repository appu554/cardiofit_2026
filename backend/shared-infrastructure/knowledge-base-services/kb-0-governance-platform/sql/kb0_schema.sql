-- =============================================================================
-- KB-0 UNIFIED GOVERNANCE PLATFORM DATABASE SCHEMA
-- PostgreSQL 14+
-- 
-- This schema supports governance for all 19 Knowledge Bases
-- =============================================================================

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE kb_id AS ENUM (
    'KB-1', 'KB-2', 'KB-3', 'KB-4', 'KB-5', 'KB-6', 'KB-7', 'KB-8', 'KB-9', 'KB-10',
    'KB-11', 'KB-12', 'KB-13', 'KB-14', 'KB-15', 'KB-16', 'KB-17', 'KB-18', 'KB-19'
);

CREATE TYPE knowledge_type AS ENUM (
    'DOSING_RULE',
    'SAFETY_ALERT',
    'INTERACTION',
    'FORMULARY_ENTRY',
    'QUALITY_MEASURE',
    'CARE_GAP',
    'ORDER_SET',
    'PROTOCOL',
    'GUIDELINE',
    'LAB_RANGE',
    'CALCULATOR',
    'TERMINOLOGY',
    'VALUE_SET',
    'CQL_LIBRARY'
);

CREATE TYPE authority AS ENUM (
    'FDA', 'TGA', 'CDSCO', 'EMA', 'MHRA', 'NICE', 'CMS', 'NCQA', 'WHO',
    'IDSA', 'ACCP', 'ACC_AHA', 'NLM', 'SNOMED', 'LOINC',
    'LEXICOMP', 'MICROMEDEX', 'INTERNAL'
);

CREATE TYPE jurisdiction AS ENUM (
    'US', 'AU', 'IN', 'UK', 'EU', 'GLOBAL'
);

CREATE TYPE risk_level AS ENUM (
    'HIGH', 'MEDIUM', 'LOW'
);

CREATE TYPE workflow_template AS ENUM (
    'CLINICAL_HIGH',
    'QUALITY_MED',
    'INFRA_LOW'
);

CREATE TYPE item_state AS ENUM (
    'DRAFT',
    'PRIMARY_REVIEW',
    'SECONDARY_REVIEW',
    'REVIEWED',
    'DIRECTOR_APPROVAL',
    'CMO_APPROVAL',
    'APPROVED',
    'ACTIVE',
    'HOLD',
    'RETIRED',
    'REJECTED',
    'REVISE',
    'AUTO_VALIDATION',
    'LEAD_APPROVAL',
    'EMERGENCY_ACTIVE'
);

CREATE TYPE audit_action AS ENUM (
    'ITEM_CREATED',
    'ITEM_INGESTED',
    'ITEM_REVIEWED',
    'ITEM_APPROVED',
    'ITEM_ACTIVATED',
    'ITEM_RETIRED',
    'ITEM_REJECTED',
    'ITEM_HELD',
    'ITEM_SENT_TO_REVISE',
    'EMERGENCY_OVERRIDE',
    'EMERGENCY_EXPIRED'
);

CREATE TYPE actor_role AS ENUM (
    'system',
    'pharmacist',
    'physician',
    'specialist',
    'cmo',
    'pt_chair',
    'quality_analyst',
    'quality_director',
    'clinical_lead',
    'pathologist',
    'lab_director',
    'terminology_manager',
    'tech_lead',
    'analytics_lead',
    'compliance_officer'
);

-- =============================================================================
-- ACTORS TABLE (Users and Systems)
-- =============================================================================

CREATE TABLE actors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(255) UNIQUE NOT NULL,
    role actor_role NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    credentials VARCHAR(255),
    institution VARCHAR(255),
    
    -- Which KBs this actor can review/approve
    allowed_kbs kb_id[],
    
    -- Active status
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_actors_role ON actors(role);
CREATE INDEX idx_actors_external ON actors(external_id);
CREATE INDEX idx_actors_kbs ON actors USING GIN (allowed_kbs);

-- =============================================================================
-- KNOWLEDGE ITEMS TABLE (Universal across all KBs)
-- =============================================================================

CREATE TABLE knowledge_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id VARCHAR(255) UNIQUE NOT NULL,  -- e.g., "kb1:warfarin:us:2025.1"
    
    -- KB identification
    kb kb_id NOT NULL,
    item_type knowledge_type NOT NULL,
    
    -- Human-readable
    name VARCHAR(500) NOT NULL,
    description TEXT,
    
    -- Content reference
    content_ref VARCHAR(500) NOT NULL,     -- Path to KB-specific content
    content_hash VARCHAR(64) NOT NULL,     -- SHA256
    
    -- Source attribution
    source_authority authority NOT NULL,
    source_document VARCHAR(500),
    source_section VARCHAR(255),
    source_url TEXT,
    source_jurisdiction jurisdiction NOT NULL,
    source_effective_date DATE,
    source_expiration_date DATE,
    
    -- Classification
    risk_level risk_level NOT NULL,
    workflow_template workflow_template NOT NULL,
    requires_dual_review BOOLEAN DEFAULT false,
    
    -- Risk flags (JSONB for flexibility)
    risk_flags JSONB DEFAULT '{}',
    
    -- State
    state item_state NOT NULL DEFAULT 'DRAFT',
    version VARCHAR(20) NOT NULL,
    
    -- Governance trail (denormalized for query performance)
    created_by VARCHAR(255) NOT NULL,
    primary_reviewer_id UUID REFERENCES actors(id),
    primary_reviewed_at TIMESTAMPTZ,
    secondary_reviewer_id UUID REFERENCES actors(id),
    secondary_reviewed_at TIMESTAMPTZ,
    approver_id UUID REFERENCES actors(id),
    approved_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    retired_at TIMESTAMPTZ,
    superseded_by VARCHAR(255),
    
    -- Full governance (JSONB for complete trail)
    governance_trail JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX idx_items_kb ON knowledge_items(kb);
CREATE INDEX idx_items_state ON knowledge_items(state);
CREATE INDEX idx_items_kb_state ON knowledge_items(kb, state);
CREATE INDEX idx_items_type ON knowledge_items(item_type);
CREATE INDEX idx_items_authority ON knowledge_items(source_authority);
CREATE INDEX idx_items_jurisdiction ON knowledge_items(source_jurisdiction);
CREATE INDEX idx_items_risk ON knowledge_items(risk_level);
CREATE INDEX idx_items_workflow ON knowledge_items(workflow_template);

-- Index for active items lookup
CREATE INDEX idx_items_active ON knowledge_items(kb, source_jurisdiction) 
    WHERE state IN ('ACTIVE', 'EMERGENCY_ACTIVE');

-- Index for pending reviews
CREATE INDEX idx_items_pending_review ON knowledge_items(kb, created_at) 
    WHERE state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE');

-- Index for pending approvals
CREATE INDEX idx_items_pending_approval ON knowledge_items(kb, primary_reviewed_at) 
    WHERE state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL');

-- Full-text search on name and description
CREATE INDEX idx_items_search ON knowledge_items 
    USING GIN (to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- =============================================================================
-- REVIEWS TABLE (Detailed review records)
-- =============================================================================

CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id VARCHAR(255) NOT NULL REFERENCES knowledge_items(item_id),
    
    -- Review metadata
    review_type VARCHAR(20) NOT NULL,  -- PRIMARY, SECONDARY, SPECIALIST
    reviewer_id UUID NOT NULL REFERENCES actors(id),
    reviewer_name VARCHAR(255) NOT NULL,
    reviewer_credentials VARCHAR(255),
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Decision
    decision VARCHAR(20) NOT NULL,  -- ACCEPT, REJECT, REVISE
    
    -- Checklist (JSONB)
    checklist JSONB,
    
    -- Notes
    notes TEXT,
    
    -- Request metadata
    ip_address INET,
    session_id VARCHAR(255),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_reviews_item ON reviews(item_id);
CREATE INDEX idx_reviews_reviewer ON reviews(reviewer_id);
CREATE INDEX idx_reviews_decision ON reviews(decision);

-- =============================================================================
-- APPROVALS TABLE (Detailed approval records)
-- =============================================================================

CREATE TABLE approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id VARCHAR(255) NOT NULL REFERENCES knowledge_items(item_id),
    
    -- Approver metadata
    approver_id UUID NOT NULL REFERENCES actors(id),
    approver_name VARCHAR(255) NOT NULL,
    approver_role actor_role NOT NULL,
    approver_credentials VARCHAR(255),
    approved_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Decision
    decision VARCHAR(20) NOT NULL,  -- APPROVE, REJECT, HOLD
    
    -- Attestations
    attestations JSONB,  -- {"medical_responsibility": true, "clinical_standards": true}
    
    -- Notes
    notes TEXT,
    
    -- Request metadata
    ip_address INET,
    session_id VARCHAR(255),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_approvals_item ON approvals(item_id);
CREATE INDEX idx_approvals_approver ON approvals(approver_id);

-- =============================================================================
-- AUDIT LOG TABLE (IMMUTABLE)
-- =============================================================================

CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    audit_id VARCHAR(50) UNIQUE NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action audit_action NOT NULL,
    
    -- Actor
    actor_id UUID REFERENCES actors(id),
    actor_external_id VARCHAR(255),
    actor_role actor_role NOT NULL,
    actor_name VARCHAR(255),
    actor_credentials VARCHAR(255),
    
    -- Item reference
    item_id VARCHAR(255) NOT NULL,
    kb kb_id NOT NULL,
    item_version VARCHAR(20),
    
    -- State transition
    previous_state item_state,
    new_state item_state NOT NULL,
    
    -- Decision details
    decision VARCHAR(50),
    notes TEXT,
    checklist JSONB,
    attestations JSONB,
    
    -- Request metadata
    ip_address INET,
    session_id VARCHAR(255),
    user_agent TEXT,
    
    -- Content integrity
    content_hash VARCHAR(64),
    
    -- Metadata
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit log indexes
CREATE INDEX idx_audit_item ON audit_log(item_id);
CREATE INDEX idx_audit_kb ON audit_log(kb);
CREATE INDEX idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_actor ON audit_log(actor_id);
CREATE INDEX idx_audit_action ON audit_log(action);

-- Prevent modifications to audit log
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit log entries cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_immutable
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();

-- =============================================================================
-- INGESTION JOBS TABLE
-- =============================================================================

CREATE TABLE ingestion_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    adapter_name VARCHAR(100) NOT NULL,
    authority authority NOT NULL,
    target_kbs kb_id[] NOT NULL,
    
    -- Status
    status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING, RUNNING, COMPLETED, FAILED
    
    -- Progress
    items_discovered INT DEFAULT 0,
    items_ingested INT DEFAULT 0,
    items_failed INT DEFAULT 0,
    
    -- Timing
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    
    -- Error tracking
    error_message TEXT,
    errors JSONB,  -- Array of item-level errors
    
    -- Metadata
    triggered_by VARCHAR(255),
    trigger_type VARCHAR(20),  -- SCHEDULED, MANUAL, WEBHOOK
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ingestion_status ON ingestion_jobs(status);
CREATE INDEX idx_ingestion_authority ON ingestion_jobs(authority);
CREATE INDEX idx_ingestion_created ON ingestion_jobs(created_at DESC);

-- =============================================================================
-- EMERGENCY OVERRIDES TABLE
-- =============================================================================

CREATE TABLE emergency_overrides (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id VARCHAR(255) NOT NULL REFERENCES knowledge_items(item_id),
    
    -- Override details
    invoked_by UUID NOT NULL REFERENCES actors(id),
    invoked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- Justification (REQUIRED)
    justification TEXT NOT NULL,
    patient_ids TEXT[],
    
    -- Resolution
    resolved_at TIMESTAMPTZ,
    resolution VARCHAR(50),  -- APPROVED, EXPIRED, CANCELLED
    resolved_by UUID REFERENCES actors(id),
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_emergency_item ON emergency_overrides(item_id);
CREATE INDEX idx_emergency_active ON emergency_overrides(expires_at) 
    WHERE resolved_at IS NULL;

-- =============================================================================
-- ITEM VERSIONS TABLE (Historical snapshots)
-- =============================================================================

CREATE TABLE item_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id VARCHAR(255) NOT NULL,
    version VARCHAR(20) NOT NULL,
    kb kb_id NOT NULL,
    
    -- Full snapshot
    item_snapshot JSONB NOT NULL,
    
    -- Version metadata
    created_at TIMESTAMPTZ NOT NULL,
    activated_at TIMESTAMPTZ,
    retired_at TIMESTAMPTZ,
    superseded_by VARCHAR(255),
    
    -- Source
    source_authority authority,
    source_document VARCHAR(500),
    content_hash VARCHAR(64),
    
    UNIQUE(item_id, version)
);

CREATE INDEX idx_versions_item ON item_versions(item_id);
CREATE INDEX idx_versions_kb ON item_versions(kb);

-- =============================================================================
-- NOTIFICATION QUEUE TABLE
-- =============================================================================

CREATE TABLE notification_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_type VARCHAR(50) NOT NULL,  -- REVIEW_REQUIRED, APPROVAL_NEEDED, SLA_BREACH, etc.
    
    -- Recipient
    recipient_id UUID REFERENCES actors(id),
    recipient_email VARCHAR(255),
    
    -- Content
    subject VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    metadata JSONB,  -- Item details, links, etc.
    
    -- Related item
    item_id VARCHAR(255),
    kb kb_id,
    
    -- Status
    status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING, SENT, FAILED
    sent_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_status ON notification_queue(status) 
    WHERE status = 'PENDING';
CREATE INDEX idx_notifications_recipient ON notification_queue(recipient_id);

-- =============================================================================
-- VIEWS
-- =============================================================================

-- Active items across all KBs
CREATE VIEW active_items AS
SELECT 
    item_id,
    kb,
    item_type,
    name,
    source_authority,
    source_jurisdiction,
    risk_level,
    version,
    activated_at,
    risk_flags
FROM knowledge_items
WHERE state IN ('ACTIVE', 'EMERGENCY_ACTIVE');

-- Pending reviews by KB
CREATE VIEW pending_reviews AS
SELECT 
    ki.item_id,
    ki.kb,
    ki.item_type,
    ki.name,
    ki.source_authority,
    ki.risk_level,
    ki.requires_dual_review,
    ki.created_at,
    ki.state,
    ki.primary_reviewer_id IS NOT NULL AS has_primary_review,
    ki.secondary_reviewer_id IS NOT NULL AS has_secondary_review,
    EXTRACT(EPOCH FROM (NOW() - ki.created_at)) / 3600 AS waiting_hours
FROM knowledge_items ki
WHERE ki.state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE', 'AUTO_VALIDATION')
ORDER BY 
    ki.risk_level DESC,
    ki.created_at ASC;

-- Pending approvals by KB
CREATE VIEW pending_approvals AS
SELECT 
    ki.item_id,
    ki.kb,
    ki.item_type,
    ki.name,
    ki.source_authority,
    ki.risk_level,
    ki.primary_reviewed_at,
    pa.name AS primary_reviewer_name,
    pa.credentials AS primary_reviewer_credentials,
    ki.requires_dual_review,
    ki.secondary_reviewer_id IS NOT NULL AS has_secondary_review,
    EXTRACT(EPOCH FROM (NOW() - ki.primary_reviewed_at)) / 3600 AS waiting_hours
FROM knowledge_items ki
LEFT JOIN actors pa ON ki.primary_reviewer_id = pa.id
WHERE ki.state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL')
ORDER BY 
    ki.risk_level DESC,
    ki.primary_reviewed_at ASC;

-- Dashboard metrics by KB
CREATE VIEW kb_metrics AS
SELECT
    kb,
    COUNT(*) FILTER (WHERE state = 'ACTIVE') AS active_count,
    COUNT(*) FILTER (WHERE state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE')) AS pending_review_count,
    COUNT(*) FILTER (WHERE state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL')) AS pending_approval_count,
    COUNT(*) FILTER (WHERE state = 'HOLD') AS hold_count,
    COUNT(*) FILTER (WHERE state = 'EMERGENCY_ACTIVE') AS emergency_count,
    COUNT(*) FILTER (WHERE state = 'RETIRED') AS retired_count,
    COUNT(*) FILTER (WHERE state = 'REJECTED') AS rejected_count,
    COUNT(*) FILTER (WHERE state = 'ACTIVE' AND risk_level = 'HIGH') AS high_risk_active,
    COUNT(*) AS total_count
FROM knowledge_items
GROUP BY kb;

-- Cross-KB summary
CREATE VIEW governance_summary AS
SELECT
    COUNT(*) FILTER (WHERE state = 'ACTIVE') AS total_active,
    COUNT(*) FILTER (WHERE state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE')) AS total_pending_review,
    COUNT(*) FILTER (WHERE state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL')) AS total_pending_approval,
    COUNT(*) FILTER (WHERE state = 'HOLD') AS total_hold,
    COUNT(*) FILTER (WHERE state = 'EMERGENCY_ACTIVE') AS total_emergency,
    COUNT(*) FILTER (WHERE state = 'ACTIVE' AND risk_level = 'HIGH') AS total_high_risk_active,
    COUNT(DISTINCT kb) AS active_kbs,
    COUNT(*) AS total_items
FROM knowledge_items;

-- =============================================================================
-- FUNCTIONS
-- =============================================================================

-- Update timestamps
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_items_timestamp
    BEFORE UPDATE ON knowledge_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_actors_timestamp
    BEFORE UPDATE ON actors
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Expire emergency overrides
CREATE OR REPLACE FUNCTION expire_emergency_overrides()
RETURNS void AS $$
BEGIN
    -- Update items
    UPDATE knowledge_items
    SET state = 'HOLD'
    WHERE state = 'EMERGENCY_ACTIVE'
      AND item_id IN (
          SELECT item_id FROM emergency_overrides
          WHERE resolved_at IS NULL AND expires_at < NOW()
      );
    
    -- Update override records
    UPDATE emergency_overrides
    SET resolved_at = NOW(),
        resolution = 'EXPIRED'
    WHERE resolved_at IS NULL
      AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Generate audit ID
CREATE OR REPLACE FUNCTION generate_audit_id()
RETURNS VARCHAR(50) AS $$
BEGIN
    RETURN 'aud_' || TO_CHAR(NOW(), 'YYYYMMDDHH24MISS') || '_' || 
           SUBSTR(MD5(RANDOM()::TEXT), 1, 8);
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- INITIAL DATA: System Actors
-- =============================================================================

INSERT INTO actors (external_id, role, name, email, allowed_kbs, is_active)
VALUES 
    ('system:ingestion:fda', 'system', 'FDA Ingestion Engine', 'system@kb0.internal', 
     ARRAY['KB-1', 'KB-4', 'KB-5']::kb_id[], true),
    ('system:ingestion:tga', 'system', 'TGA Ingestion Engine', 'system@kb0.internal', 
     ARRAY['KB-1', 'KB-4', 'KB-5', 'KB-6']::kb_id[], true),
    ('system:ingestion:cdsco', 'system', 'CDSCO Ingestion Engine', 'system@kb0.internal', 
     ARRAY['KB-1', 'KB-4', 'KB-5', 'KB-6']::kb_id[], true),
    ('system:ingestion:cms', 'system', 'CMS eCQM Ingestion Engine', 'system@kb0.internal', 
     ARRAY['KB-9', 'KB-13']::kb_id[], true),
    ('system:ingestion:snomed', 'system', 'SNOMED CT Ingestion Engine', 'system@kb0.internal', 
     ARRAY['KB-7']::kb_id[], true),
    ('system:ingestion:rxnorm', 'system', 'RxNorm Ingestion Engine', 'system@kb0.internal', 
     ARRAY['KB-7']::kb_id[], true),
    ('system:ingestion:loinc', 'system', 'LOINC Ingestion Engine', 'system@kb0.internal', 
     ARRAY['KB-7', 'KB-16']::kb_id[], true),
    ('system:activation', 'system', 'Activation Engine', 'system@kb0.internal', 
     NULL, true),
    ('system:validation', 'system', 'Auto-Validation Engine', 'system@kb0.internal', 
     ARRAY['KB-2', 'KB-3', 'KB-7', 'KB-10', 'KB-11', 'KB-14', 'KB-17', 'KB-18']::kb_id[], true);

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE knowledge_items IS 'Universal table for all governed knowledge across all 19 KBs';
COMMENT ON TABLE audit_log IS 'Immutable audit trail for all governance actions';
COMMENT ON TABLE actors IS 'Users and systems that interact with the governance platform';
COMMENT ON TABLE reviews IS 'Detailed pharmacist/specialist review records';
COMMENT ON TABLE approvals IS 'Detailed CMO/director approval records';
COMMENT ON TABLE ingestion_jobs IS 'Tracking for batch ingestion jobs';
COMMENT ON TABLE emergency_overrides IS 'Emergency CMO override records with expiry';
COMMENT ON TABLE item_versions IS 'Historical version snapshots for audit';

COMMENT ON COLUMN knowledge_items.item_id IS 'Format: kb{N}:{name}:{jurisdiction}:{version}';
COMMENT ON COLUMN knowledge_items.content_ref IS 'Path to KB-specific YAML/CQL/JSON content';
COMMENT ON COLUMN knowledge_items.content_hash IS 'SHA256 hash of content for integrity';
COMMENT ON COLUMN knowledge_items.governance_trail IS 'Complete governance history as JSONB';
