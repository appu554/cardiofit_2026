-- KB-14 Care Navigator: Immutable Audit Ledger
-- Purpose: Court-proof traceability for all task operations
-- This table is APPEND-ONLY - no updates or deletes allowed

-- =============================================================================
-- AUDIT EVENTS TABLE
-- Captures every state change with cryptographic hash chain
-- =============================================================================

CREATE TABLE IF NOT EXISTS task_audit_log (
    -- Primary Key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Sequence for hash chain integrity
    sequence_number BIGSERIAL NOT NULL,

    -- Task Reference
    task_id UUID NOT NULL REFERENCES tasks(id),
    task_number VARCHAR(50) NOT NULL,

    -- Event Details
    event_type VARCHAR(50) NOT NULL,  -- CREATED, ASSIGNED, STARTED, COMPLETED, ESCALATED, etc.
    event_category VARCHAR(30) NOT NULL, -- LIFECYCLE, ASSIGNMENT, ESCALATION, MODIFICATION, GOVERNANCE

    -- State Change
    previous_status VARCHAR(30),
    new_status VARCHAR(30),
    previous_value JSONB DEFAULT '{}',
    new_value JSONB DEFAULT '{}',

    -- Actor Information (WHO)
    actor_id UUID,
    actor_type VARCHAR(30) NOT NULL, -- USER, SYSTEM, WORKER, INTEGRATION
    actor_name VARCHAR(100),
    actor_role VARCHAR(50),

    -- Clinical Context
    patient_id VARCHAR(50) NOT NULL,
    encounter_id VARCHAR(50),

    -- Source Information (WHY)
    source_service VARCHAR(50), -- KB3_TEMPORAL, KB9_CARE_GAPS, KB12_ORDER_SETS, MANUAL, SYSTEM
    source_event_id VARCHAR(100),

    -- Governance Fields
    reason_code VARCHAR(50), -- CLINICAL_NECESSITY, PATIENT_REQUEST, PROTOCOL_REQUIREMENT, etc.
    reason_text TEXT,
    clinical_justification TEXT,

    -- Evidence Snapshot
    evidence_snapshot JSONB DEFAULT '{}', -- Frozen copy of clinical evidence at time of action

    -- Hash Chain for Immutability
    previous_hash VARCHAR(64), -- SHA-256 of previous record
    record_hash VARCHAR(64) NOT NULL, -- SHA-256 of this record's content

    -- Timestamps
    event_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Metadata
    ip_address TEXT,  -- Changed from INET to TEXT for flexibility with empty strings
    user_agent TEXT,
    session_id VARCHAR(100),
    metadata JSONB DEFAULT '{}'
);

-- =============================================================================
-- GOVERNANCE EVENTS TABLE
-- High-level governance events for Tier-7 compliance
-- =============================================================================

CREATE TABLE IF NOT EXISTS governance_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Event Classification
    event_type VARCHAR(50) NOT NULL, -- COMPLIANCE_CHECK, AUDIT_REQUIRED, POLICY_VIOLATION, SLA_BREACH
    severity VARCHAR(20) NOT NULL, -- INFO, WARNING, CRITICAL, ALERT

    -- Context
    task_id UUID REFERENCES tasks(id),
    patient_id VARCHAR(50),
    organization_id VARCHAR(50),

    -- Event Details
    title VARCHAR(200) NOT NULL,
    description TEXT,

    -- Governance Metrics
    compliance_score DECIMAL(5,2), -- 0.00 to 100.00
    risk_score DECIMAL(5,2), -- 0.00 to 100.00

    -- Resolution
    requires_action BOOLEAN DEFAULT false,
    action_deadline TIMESTAMPTZ,
    resolved BOOLEAN DEFAULT false,
    resolved_by UUID,
    resolved_at TIMESTAMPTZ,
    resolution_notes TEXT,

    -- Audit Trail
    triggered_by VARCHAR(50) NOT NULL, -- SYSTEM, USER, WORKER, EXTERNAL
    triggered_by_id VARCHAR(100),

    -- Evidence
    evidence JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- REASON CODES REFERENCE TABLE
-- Standardized reason codes for governance compliance
-- =============================================================================

CREATE TABLE IF NOT EXISTS reason_codes (
    code VARCHAR(50) PRIMARY KEY,
    category VARCHAR(30) NOT NULL, -- ACCEPTANCE, REJECTION, ESCALATION, COMPLETION, CANCELLATION
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    requires_justification BOOLEAN DEFAULT false,
    requires_supervisor_approval BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- TASK INTELLIGENCE TRACKING
-- Ensures no orphan intelligence (every alert becomes accountable)
-- =============================================================================

CREATE TABLE IF NOT EXISTS intelligence_tracking (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Source Intelligence
    source_service VARCHAR(50) NOT NULL, -- KB3, KB9, KB12
    source_id VARCHAR(100) NOT NULL,
    source_type VARCHAR(50) NOT NULL, -- TEMPORAL_ALERT, CARE_GAP, CARE_PLAN_ACTIVITY, PROTOCOL_STEP

    -- Patient Context
    patient_id VARCHAR(50) NOT NULL,

    -- Processing Status
    status VARCHAR(30) NOT NULL, -- RECEIVED, PROCESSED, TASK_CREATED, DECLINED, ERROR

    -- Task Linkage
    task_id UUID REFERENCES tasks(id),

    -- Disposition (if not creating task)
    disposition_code VARCHAR(50), -- NOT_CLINICALLY_RELEVANT, DUPLICATE, PATIENT_DECLINED, etc.
    disposition_reason TEXT,
    disposition_by UUID,
    disposition_at TIMESTAMPTZ,

    -- Evidence
    intelligence_snapshot JSONB NOT NULL, -- Full copy of original intelligence

    -- Timestamps
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,

    -- Unique constraint to prevent duplicates
    UNIQUE(source_service, source_id)
);

-- =============================================================================
-- INDEXES FOR PERFORMANCE
-- =============================================================================

-- Audit Log Indexes
CREATE INDEX idx_audit_log_task_id ON task_audit_log(task_id);
CREATE INDEX idx_audit_log_event_type ON task_audit_log(event_type);
CREATE INDEX idx_audit_log_actor_id ON task_audit_log(actor_id);
CREATE INDEX idx_audit_log_patient_id ON task_audit_log(patient_id);
CREATE INDEX idx_audit_log_event_timestamp ON task_audit_log(event_timestamp DESC);
CREATE INDEX idx_audit_log_sequence ON task_audit_log(sequence_number);
CREATE INDEX idx_audit_log_hash_chain ON task_audit_log(previous_hash, record_hash);

-- Governance Events Indexes
CREATE INDEX idx_governance_events_task_id ON governance_events(task_id);
CREATE INDEX idx_governance_events_patient_id ON governance_events(patient_id);
CREATE INDEX idx_governance_events_type ON governance_events(event_type);
CREATE INDEX idx_governance_events_severity ON governance_events(severity);
CREATE INDEX idx_governance_events_unresolved ON governance_events(resolved) WHERE resolved = false;
CREATE INDEX idx_governance_events_created ON governance_events(created_at DESC);

-- Intelligence Tracking Indexes
CREATE INDEX idx_intelligence_source ON intelligence_tracking(source_service, source_id);
CREATE INDEX idx_intelligence_patient ON intelligence_tracking(patient_id);
CREATE INDEX idx_intelligence_status ON intelligence_tracking(status);
CREATE INDEX idx_intelligence_unprocessed ON intelligence_tracking(status) WHERE status = 'RECEIVED';

-- =============================================================================
-- INSERT DEFAULT REASON CODES
-- =============================================================================

INSERT INTO reason_codes (code, category, display_name, description, requires_justification, requires_supervisor_approval, sort_order) VALUES
-- Acceptance Codes
('CLINICAL_NECESSITY', 'ACCEPTANCE', 'Clinical Necessity', 'Task accepted due to clinical requirements', false, false, 1),
('PROTOCOL_REQUIREMENT', 'ACCEPTANCE', 'Protocol Requirement', 'Task required by active clinical protocol', false, false, 2),
('PATIENT_REQUEST', 'ACCEPTANCE', 'Patient Request', 'Task created based on patient request', false, false, 3),
('PREVENTIVE_CARE', 'ACCEPTANCE', 'Preventive Care', 'Task for preventive care measure', false, false, 4),
('REGULATORY_COMPLIANCE', 'ACCEPTANCE', 'Regulatory Compliance', 'Task required for regulatory compliance', false, false, 5),

-- Rejection/Decline Codes
('NOT_CLINICALLY_RELEVANT', 'REJECTION', 'Not Clinically Relevant', 'Intelligence not clinically relevant for this patient', true, false, 10),
('DUPLICATE_INTELLIGENCE', 'REJECTION', 'Duplicate Intelligence', 'Already addressed by existing task', false, false, 11),
('PATIENT_DECLINED', 'REJECTION', 'Patient Declined', 'Patient declined the recommended action', true, false, 12),
('CONTRAINDICATED', 'REJECTION', 'Contraindicated', 'Action is contraindicated for this patient', true, true, 13),
('ALTERNATE_PATH', 'REJECTION', 'Alternate Clinical Path', 'Addressed through alternative clinical pathway', true, false, 14),
('OUTSIDE_SCOPE', 'REJECTION', 'Outside Care Scope', 'Outside the scope of care for this provider', true, false, 15),

-- Escalation Codes
('SLA_BREACH', 'ESCALATION', 'SLA Breach', 'Task escalated due to SLA breach', false, false, 20),
('CLINICAL_URGENCY', 'ESCALATION', 'Clinical Urgency', 'Task escalated due to clinical urgency', false, false, 21),
('RESOURCE_UNAVAILABLE', 'ESCALATION', 'Resource Unavailable', 'Escalated due to unavailable resources', true, false, 22),
('EXPERTISE_REQUIRED', 'ESCALATION', 'Expertise Required', 'Escalated to specialist for expertise', true, false, 23),
('MANUAL_ESCALATION', 'ESCALATION', 'Manual Escalation', 'Manually escalated by care team', true, false, 24),

-- Completion Codes
('RESOLVED', 'COMPLETION', 'Resolved', 'Task completed successfully', false, false, 30),
('RESOLVED_WITH_FOLLOWUP', 'COMPLETION', 'Resolved with Follow-up', 'Completed with scheduled follow-up', false, false, 31),
('PARTIALLY_RESOLVED', 'COMPLETION', 'Partially Resolved', 'Some actions completed, others pending', true, false, 32),
('REFERRED', 'COMPLETION', 'Referred to Specialist', 'Completed by referral to specialist', false, false, 33),

-- Cancellation Codes
('NO_LONGER_APPLICABLE', 'CANCELLATION', 'No Longer Applicable', 'Condition or need no longer applies', true, false, 40),
('PATIENT_TRANSFERRED', 'CANCELLATION', 'Patient Transferred', 'Patient transferred to another facility', false, false, 41),
('PATIENT_DECEASED', 'CANCELLATION', 'Patient Deceased', 'Patient deceased', false, false, 42),
('SUPERSEDED', 'CANCELLATION', 'Superseded', 'Task superseded by newer task', false, false, 43),
('ERROR_CREATED', 'CANCELLATION', 'Created in Error', 'Task was created in error', true, true, 44)

ON CONFLICT (code) DO NOTHING;

-- =============================================================================
-- TRIGGER: Prevent Updates/Deletes on Audit Log (Immutability)
-- =============================================================================

CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit log records are immutable and cannot be modified or deleted';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_immutable_update
    BEFORE UPDATE ON task_audit_log
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER audit_log_immutable_delete
    BEFORE DELETE ON task_audit_log
    FOR EACH ROW
    EXECUTE FUNCTION prevent_audit_modification();

-- =============================================================================
-- FUNCTION: Calculate Record Hash
-- =============================================================================

CREATE OR REPLACE FUNCTION calculate_audit_hash(
    p_task_id UUID,
    p_event_type VARCHAR,
    p_actor_id UUID,
    p_previous_status VARCHAR,
    p_new_status VARCHAR,
    p_event_timestamp TIMESTAMPTZ,
    p_previous_hash VARCHAR
) RETURNS VARCHAR AS $$
DECLARE
    v_content TEXT;
    v_hash VARCHAR;
BEGIN
    v_content := COALESCE(p_task_id::TEXT, '') || '|' ||
                 COALESCE(p_event_type, '') || '|' ||
                 COALESCE(p_actor_id::TEXT, '') || '|' ||
                 COALESCE(p_previous_status, '') || '|' ||
                 COALESCE(p_new_status, '') || '|' ||
                 COALESCE(p_event_timestamp::TEXT, '') || '|' ||
                 COALESCE(p_previous_hash, 'GENESIS');

    v_hash := encode(sha256(v_content::bytea), 'hex');
    RETURN v_hash;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- =============================================================================
-- VIEW: Audit Trail Summary
-- =============================================================================

CREATE OR REPLACE VIEW v_task_audit_summary AS
SELECT
    t.id AS task_id,
    t.task_id AS task_number,
    t.patient_id,
    t.type AS task_type,
    t.status AS current_status,
    COUNT(a.id) AS total_events,
    MIN(a.event_timestamp) AS first_event,
    MAX(a.event_timestamp) AS last_event,
    COUNT(DISTINCT a.actor_id) AS unique_actors,
    ARRAY_AGG(DISTINCT a.event_type ORDER BY a.event_type) AS event_types,
    BOOL_OR(a.reason_code IS NOT NULL) AS has_reason_codes
FROM tasks t
LEFT JOIN task_audit_log a ON t.id = a.task_id
GROUP BY t.id, t.task_id, t.patient_id, t.type, t.status;

-- =============================================================================
-- VIEW: Governance Dashboard
-- =============================================================================

CREATE OR REPLACE VIEW v_governance_dashboard AS
SELECT
    DATE(created_at) AS date,
    event_type,
    severity,
    COUNT(*) AS event_count,
    COUNT(*) FILTER (WHERE resolved = true) AS resolved_count,
    COUNT(*) FILTER (WHERE resolved = false AND requires_action = true) AS pending_action_count,
    AVG(compliance_score) AS avg_compliance_score,
    AVG(risk_score) AS avg_risk_score
FROM governance_events
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY DATE(created_at), event_type, severity
ORDER BY date DESC, severity DESC;

-- =============================================================================
-- VIEW: Intelligence Accountability
-- =============================================================================

CREATE OR REPLACE VIEW v_intelligence_accountability AS
SELECT
    source_service,
    source_type,
    status,
    COUNT(*) AS count,
    COUNT(*) FILTER (WHERE task_id IS NOT NULL) AS tasks_created,
    COUNT(*) FILTER (WHERE disposition_code IS NOT NULL) AS dispositioned,
    COUNT(*) FILTER (WHERE status = 'RECEIVED' AND processed_at IS NULL) AS pending
FROM intelligence_tracking
WHERE received_at >= NOW() - INTERVAL '7 days'
GROUP BY source_service, source_type, status
ORDER BY source_service, count DESC;

COMMENT ON TABLE task_audit_log IS 'Immutable audit ledger for all task operations - court-proof traceability';
COMMENT ON TABLE governance_events IS 'High-level governance events for Tier-7 compliance reporting';
COMMENT ON TABLE reason_codes IS 'Standardized reason codes for governance compliance';
COMMENT ON TABLE intelligence_tracking IS 'Tracks all incoming intelligence to ensure no orphan data';
