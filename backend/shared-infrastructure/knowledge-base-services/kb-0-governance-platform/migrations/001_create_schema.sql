-- =============================================================================
-- KB-0 Governance Platform Database Schema
-- =============================================================================
-- Unified governance workflow engine for all Knowledge Bases
-- Manages review, approval, and audit trail for clinical knowledge items
-- =============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- =============================================================================
-- KNOWLEDGE ITEMS TABLE
-- =============================================================================
-- Core table for tracking governance state of all knowledge items

CREATE TABLE knowledge_items (
    -- Primary Key
    item_id VARCHAR(100) PRIMARY KEY,

    -- Knowledge Base Reference
    kb VARCHAR(20) NOT NULL,  -- KB1, KB2, KB3, etc.
    item_type VARCHAR(50) NOT NULL,  -- DOSING_RULE, INTERACTION, GUIDELINE, etc.

    -- Item Identity
    name VARCHAR(500) NOT NULL,
    description TEXT,
    content_ref VARCHAR(500),  -- Path to content file or external ID
    content_hash VARCHAR(64),  -- SHA-256 hash for change detection

    -- Source Attribution
    source_authority VARCHAR(50),  -- FDA, TGA, CDSCO, NICE, etc.
    source_document VARCHAR(500),
    source_section VARCHAR(200),
    source_url TEXT,
    source_jurisdiction VARCHAR(20),  -- US, AU, IN, GLOBAL
    source_effective_date VARCHAR(50),
    source_expiration_date VARCHAR(50),

    -- Risk Assessment
    risk_level VARCHAR(20) DEFAULT 'MODERATE',  -- LOW, MODERATE, HIGH, CRITICAL
    workflow_template VARCHAR(50) NOT NULL,  -- CLINICAL_HIGH, CLINICAL_STANDARD, etc.
    requires_dual_review BOOLEAN DEFAULT FALSE,
    risk_flags JSONB DEFAULT '[]'::jsonb,

    -- Governance State
    state VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    version VARCHAR(20) DEFAULT '1.0',

    -- Governance Trail (JSON for flexibility)
    created_by VARCHAR(200),
    governance_trail JSONB DEFAULT '{}'::jsonb,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    activated_at TIMESTAMP WITH TIME ZONE,
    retired_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT chk_state CHECK (state IN (
        'DRAFT', 'SUBMITTED',
        'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE', 'AUTO_VALIDATION',
        'REVIEWED',
        'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL',
        'APPROVED', 'ACTIVE', 'EMERGENCY_ACTIVE',
        'HOLD', 'RETIRED', 'REJECTED'
    )),
    CONSTRAINT chk_risk_level CHECK (risk_level IN ('LOW', 'MODERATE', 'HIGH', 'CRITICAL'))
);

-- Indexes for common queries
CREATE INDEX idx_knowledge_items_kb ON knowledge_items(kb);
CREATE INDEX idx_knowledge_items_state ON knowledge_items(state);
CREATE INDEX idx_knowledge_items_kb_state ON knowledge_items(kb, state);
CREATE INDEX idx_knowledge_items_risk_level ON knowledge_items(risk_level);
CREATE INDEX idx_knowledge_items_name ON knowledge_items USING gin(name gin_trgm_ops);
CREATE INDEX idx_knowledge_items_authority ON knowledge_items(source_authority);
CREATE INDEX idx_knowledge_items_created_at ON knowledge_items(created_at DESC);

-- =============================================================================
-- AUDIT ENTRIES TABLE
-- =============================================================================
-- Complete audit trail for all governance actions

CREATE TABLE audit_entries (
    id VARCHAR(100) PRIMARY KEY,

    -- Timestamp
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Action Details
    action VARCHAR(50) NOT NULL,  -- ITEM_CREATED, ITEM_REVIEWED, ITEM_APPROVED, etc.
    decision VARCHAR(50),  -- ACCEPT, REJECT, REVISE, HOLD, etc.

    -- Actor Information
    actor_id VARCHAR(100) NOT NULL,
    actor_name VARCHAR(200),
    actor_role VARCHAR(50),
    credentials VARCHAR(200),  -- Professional credentials (MD, PharmD, etc.)

    -- Item Reference
    item_id VARCHAR(100) NOT NULL,
    kb VARCHAR(20) NOT NULL,
    item_version VARCHAR(20),

    -- State Transition
    previous_state VARCHAR(30),
    new_state VARCHAR(30),

    -- Review Details
    notes TEXT,
    checklist JSONB,  -- Review checklist items
    attestations JSONB,  -- Required attestations

    -- Security Context
    ip_address VARCHAR(45),
    session_id VARCHAR(100),
    content_hash VARCHAR(64),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Foreign Key
    CONSTRAINT fk_audit_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE CASCADE
);

-- Indexes for audit queries
CREATE INDEX idx_audit_entries_item ON audit_entries(item_id);
CREATE INDEX idx_audit_entries_actor ON audit_entries(actor_id);
CREATE INDEX idx_audit_entries_action ON audit_entries(action);
CREATE INDEX idx_audit_entries_timestamp ON audit_entries(timestamp DESC);
CREATE INDEX idx_audit_entries_kb ON audit_entries(kb);

-- =============================================================================
-- SLA TRACKING TABLE
-- =============================================================================
-- Service Level Agreement tracking for governance workflows

CREATE TABLE sla_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    item_id VARCHAR(100) NOT NULL,

    -- SLA Type
    sla_type VARCHAR(50) NOT NULL,  -- INITIAL_REVIEW, APPROVAL, ACTIVATION

    -- Timing
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    due_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Status
    status VARCHAR(20) DEFAULT 'ACTIVE',  -- ACTIVE, MET, BREACHED, CANCELLED
    breach_notified BOOLEAN DEFAULT FALSE,

    -- Assignment
    assigned_to VARCHAR(100),
    escalated_to VARCHAR(100),
    escalation_count INTEGER DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Foreign Key
    CONSTRAINT fk_sla_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE CASCADE
);

CREATE INDEX idx_sla_records_item ON sla_records(item_id);
CREATE INDEX idx_sla_records_status ON sla_records(status);
CREATE INDEX idx_sla_records_due ON sla_records(due_at) WHERE status = 'ACTIVE';

-- =============================================================================
-- NOTIFICATION QUEUE TABLE
-- =============================================================================
-- Queue for pending notifications

CREATE TABLE notification_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Notification Details
    notification_type VARCHAR(50) NOT NULL,
    priority VARCHAR(20) DEFAULT 'NORMAL',

    -- Target
    recipient_id VARCHAR(100) NOT NULL,
    recipient_email VARCHAR(255),

    -- Content
    subject VARCHAR(500),
    body TEXT,
    metadata JSONB,

    -- Item Reference
    item_id VARCHAR(100),

    -- Status
    status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING, SENT, FAILED
    retry_count INTEGER DEFAULT 0,
    last_error TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    sent_at TIMESTAMP WITH TIME ZONE,

    -- Foreign Key
    CONSTRAINT fk_notification_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE CASCADE
);

CREATE INDEX idx_notification_queue_status ON notification_queue(status);
CREATE INDEX idx_notification_queue_recipient ON notification_queue(recipient_id);

-- =============================================================================
-- WORKFLOW TEMPLATES TABLE
-- =============================================================================
-- Configuration for different workflow types

CREATE TABLE workflow_templates (
    template_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    description TEXT,

    -- Configuration
    transitions JSONB NOT NULL,  -- State machine transitions
    sla_config JSONB,  -- SLA requirements per state
    checklist_config JSONB,  -- Required checklist items
    attestation_config JSONB,  -- Required attestations

    -- Applicability
    applicable_kbs TEXT[],  -- Which KBs use this template
    applicable_item_types TEXT[],  -- Which item types

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =============================================================================
-- KB-1 INTEGRATION TRACKING
-- =============================================================================
-- Track items that are managed via KB-1 API integration

CREATE TABLE kb1_integration_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- KB-1 Reference
    kb1_rule_id VARCHAR(100) NOT NULL,
    rxnorm_code VARCHAR(20),

    -- KB-0 Item Reference
    item_id VARCHAR(100),

    -- Operation
    operation VARCHAR(50) NOT NULL,  -- SYNC, REVIEW, APPROVE, REJECT
    operation_status VARCHAR(20) NOT NULL,  -- SUCCESS, FAILED, PENDING

    -- Request/Response
    request_payload JSONB,
    response_payload JSONB,
    error_message TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Foreign Key
    CONSTRAINT fk_kb1_log_item FOREIGN KEY (item_id)
        REFERENCES knowledge_items(item_id) ON DELETE SET NULL
);

CREATE INDEX idx_kb1_log_rule ON kb1_integration_log(kb1_rule_id);
CREATE INDEX idx_kb1_log_operation ON kb1_integration_log(operation);
CREATE INDEX idx_kb1_log_status ON kb1_integration_log(operation_status);

-- =============================================================================
-- FUNCTIONS AND TRIGGERS
-- =============================================================================

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_knowledge_items_updated
    BEFORE UPDATE ON knowledge_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER tr_sla_records_updated
    BEFORE UPDATE ON sla_records
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER tr_workflow_templates_updated
    BEFORE UPDATE ON workflow_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- VIEWS
-- =============================================================================

-- Dashboard view for pending items
CREATE VIEW v_pending_governance AS
SELECT
    ki.item_id,
    ki.kb,
    ki.item_type,
    ki.name,
    ki.state,
    ki.risk_level,
    ki.requires_dual_review,
    ki.source_authority,
    ki.source_jurisdiction,
    ki.created_at,
    ki.updated_at,
    sr.due_at as sla_due_at,
    sr.status as sla_status
FROM knowledge_items ki
LEFT JOIN sla_records sr ON ki.item_id = sr.item_id AND sr.status = 'ACTIVE'
WHERE ki.state NOT IN ('ACTIVE', 'RETIRED', 'REJECTED')
ORDER BY
    CASE ki.risk_level
        WHEN 'CRITICAL' THEN 1
        WHEN 'HIGH' THEN 2
        WHEN 'MODERATE' THEN 3
        ELSE 4
    END,
    ki.created_at ASC;

-- Metrics view per KB
CREATE VIEW v_kb_metrics AS
SELECT
    kb,
    COUNT(*) FILTER (WHERE state = 'ACTIVE') AS active_count,
    COUNT(*) FILTER (WHERE state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE')) AS pending_review_count,
    COUNT(*) FILTER (WHERE state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL')) AS pending_approval_count,
    COUNT(*) FILTER (WHERE state = 'HOLD') AS hold_count,
    COUNT(*) FILTER (WHERE state = 'EMERGENCY_ACTIVE') AS emergency_count,
    COUNT(*) FILTER (WHERE state = 'ACTIVE' AND risk_level IN ('HIGH', 'CRITICAL')) AS high_risk_active_count,
    COUNT(*) AS total_count
FROM knowledge_items
GROUP BY kb;

-- =============================================================================
-- INSERT DEFAULT WORKFLOW TEMPLATES
-- =============================================================================

INSERT INTO workflow_templates (template_id, name, description, transitions, applicable_kbs, applicable_item_types)
VALUES
(
    'CLINICAL_HIGH',
    'Clinical High-Risk Workflow',
    'For high-risk clinical items requiring dual review and CMO approval',
    '{
        "transitions": [
            {"from": ["DRAFT"], "to": "SUBMITTED", "action": "submit", "actors": ["author", "system"]},
            {"from": ["SUBMITTED"], "to": "PRIMARY_REVIEW", "action": "assign_review", "actors": ["coordinator", "system"]},
            {"from": ["PRIMARY_REVIEW"], "to": "SECONDARY_REVIEW", "action": "submit_review", "actors": ["clinical_pharmacist", "physician"]},
            {"from": ["SECONDARY_REVIEW"], "to": "REVIEWED", "action": "submit_review", "actors": ["clinical_pharmacist", "physician"]},
            {"from": ["REVIEWED"], "to": "CMO_APPROVAL", "action": "request_approval", "actors": ["coordinator", "system"]},
            {"from": ["CMO_APPROVAL"], "to": "APPROVED", "action": "approve", "actors": ["cmo", "director"]},
            {"from": ["APPROVED"], "to": "ACTIVE", "action": "activate", "actors": ["system"]}
        ]
    }'::jsonb,
    ARRAY['KB1', 'KB2', 'KB4'],
    ARRAY['DOSING_RULE', 'INTERACTION', 'SAFETY_ALERT']
),
(
    'CLINICAL_STANDARD',
    'Clinical Standard Workflow',
    'For moderate-risk clinical items with single review',
    '{
        "transitions": [
            {"from": ["DRAFT"], "to": "SUBMITTED", "action": "submit", "actors": ["author", "system"]},
            {"from": ["SUBMITTED"], "to": "PRIMARY_REVIEW", "action": "assign_review", "actors": ["coordinator", "system"]},
            {"from": ["PRIMARY_REVIEW"], "to": "REVIEWED", "action": "submit_review", "actors": ["clinical_pharmacist", "physician"]},
            {"from": ["REVIEWED"], "to": "DIRECTOR_APPROVAL", "action": "request_approval", "actors": ["coordinator", "system"]},
            {"from": ["DIRECTOR_APPROVAL"], "to": "APPROVED", "action": "approve", "actors": ["director"]},
            {"from": ["APPROVED"], "to": "ACTIVE", "action": "activate", "actors": ["system"]}
        ]
    }'::jsonb,
    ARRAY['KB1', 'KB2', 'KB3', 'KB5'],
    ARRAY['GUIDELINE', 'PROTOCOL', 'PATHWAY']
),
(
    'TERMINOLOGY_UPDATE',
    'Terminology Update Workflow',
    'For terminology and code system updates',
    '{
        "transitions": [
            {"from": ["DRAFT"], "to": "SUBMITTED", "action": "submit", "actors": ["author", "system"]},
            {"from": ["SUBMITTED"], "to": "AUTO_VALIDATION", "action": "auto_validate", "actors": ["system"]},
            {"from": ["AUTO_VALIDATION"], "to": "LEAD_APPROVAL", "action": "request_approval", "actors": ["system"]},
            {"from": ["LEAD_APPROVAL"], "to": "APPROVED", "action": "approve", "actors": ["terminology_lead"]},
            {"from": ["APPROVED"], "to": "ACTIVE", "action": "activate", "actors": ["system"]}
        ]
    }'::jsonb,
    ARRAY['KB7'],
    ARRAY['TERMINOLOGY', 'CODE_SYSTEM', 'VALUE_SET']
);

-- =============================================================================
-- GRANT PERMISSIONS (if using specific database users)
-- =============================================================================
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO kb0_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kb0_app;

-- =============================================================================
-- COMPLETION MESSAGE
-- =============================================================================
DO $$
BEGIN
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'KB-0 Governance Platform Schema Created Successfully';
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Tables: knowledge_items, audit_entries, sla_records,';
    RAISE NOTICE '        notification_queue, workflow_templates,';
    RAISE NOTICE '        kb1_integration_log';
    RAISE NOTICE 'Views: v_pending_governance, v_kb_metrics';
    RAISE NOTICE '===================================================';
END $$;
