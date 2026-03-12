-- KB-18 Governance Engine Database Schema
-- Clinical Governance Enforcement Platform
-- Version: 1.0.0

-- ============================================================================
-- EXTENSIONS
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- ENUMS
-- ============================================================================

-- Enforcement Levels
CREATE TYPE enforcement_level AS ENUM (
    'IGNORE',
    'NOTIFY',
    'WARN_ACKNOWLEDGE',
    'HARD_BLOCK',
    'HARD_BLOCK_WITH_OVERRIDE',
    'MANDATORY_ESCALATION'
);

-- Severity Levels
CREATE TYPE severity_level AS ENUM (
    'INFO',
    'LOW',
    'MODERATE',
    'HIGH',
    'CRITICAL',
    'FATAL'
);

-- Override Status
CREATE TYPE override_status AS ENUM (
    'PENDING',
    'APPROVED',
    'DENIED',
    'EXPIRED'
);

-- Escalation Status
CREATE TYPE escalation_status AS ENUM (
    'OPEN',
    'ACKNOWLEDGED',
    'RESOLVED',
    'CLOSED'
);

-- Evaluation Outcome
CREATE TYPE evaluation_outcome AS ENUM (
    'APPROVED',
    'APPROVED_WITH_WARNINGS',
    'PENDING_ACK',
    'PENDING_OVERRIDE',
    'BLOCKED',
    'ESCALATED'
);

-- ============================================================================
-- EVIDENCE TRAILS TABLE
-- Immutable audit records with cryptographic hashes
-- ============================================================================
CREATE TABLE evidence_trails (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trail_id VARCHAR(100) UNIQUE NOT NULL,
    request_id VARCHAR(100) NOT NULL,
    patient_id VARCHAR(100) NOT NULL,

    -- Evaluation Details
    evaluation_type VARCHAR(50) NOT NULL,
    outcome evaluation_outcome NOT NULL,

    -- Snapshots (JSONB for flexibility)
    patient_snapshot JSONB NOT NULL,
    order_snapshot JSONB,

    -- Evaluation Results
    programs_evaluated TEXT[] NOT NULL DEFAULT '{}',
    rules_applied JSONB NOT NULL DEFAULT '[]',
    violations JSONB NOT NULL DEFAULT '[]',

    -- Requestor Information
    requested_by VARCHAR(100) NOT NULL,
    requested_by_role VARCHAR(50),
    facility_id VARCHAR(100),

    -- Cryptographic Integrity
    decision_hash VARCHAR(100) NOT NULL,
    previous_hash VARCHAR(100),

    -- Timestamps
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Indexes for common queries
    CONSTRAINT evidence_trails_hash_unique UNIQUE (decision_hash)
);

CREATE INDEX idx_evidence_trails_patient ON evidence_trails(patient_id);
CREATE INDEX idx_evidence_trails_request ON evidence_trails(request_id);
CREATE INDEX idx_evidence_trails_outcome ON evidence_trails(outcome);
CREATE INDEX idx_evidence_trails_evaluated_at ON evidence_trails(evaluated_at);

-- ============================================================================
-- OVERRIDE REQUESTS TABLE
-- ============================================================================
CREATE TABLE override_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- References
    violation_id VARCHAR(100) NOT NULL,
    trail_id VARCHAR(100) REFERENCES evidence_trails(trail_id),
    patient_id VARCHAR(100) NOT NULL,

    -- Rule Information
    rule_code VARCHAR(50) NOT NULL,
    program_code VARCHAR(50),

    -- Requestor
    requestor_id VARCHAR(100) NOT NULL,
    requestor_role VARCHAR(50) NOT NULL,
    requestor_name VARCHAR(200),

    -- Request Details
    reason TEXT NOT NULL,
    clinical_justification TEXT,
    risk_accepted BOOLEAN DEFAULT FALSE,

    -- Status
    status override_status NOT NULL DEFAULT 'PENDING',

    -- Approval/Denial
    reviewed_by VARCHAR(100),
    reviewed_by_role VARCHAR(50),
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,
    denial_reason TEXT,

    -- Timestamps
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_override_requests_patient ON override_requests(patient_id);
CREATE INDEX idx_override_requests_requestor ON override_requests(requestor_id);
CREATE INDEX idx_override_requests_status ON override_requests(status);
CREATE INDEX idx_override_requests_rule ON override_requests(rule_code);
CREATE INDEX idx_override_requests_requested_at ON override_requests(requested_at);

-- ============================================================================
-- ACKNOWLEDGMENTS TABLE
-- ============================================================================
CREATE TABLE acknowledgments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- References
    violation_id VARCHAR(100) NOT NULL,
    trail_id VARCHAR(100) REFERENCES evidence_trails(trail_id),
    patient_id VARCHAR(100) NOT NULL,

    -- Rule Information
    rule_code VARCHAR(50) NOT NULL,

    -- User
    user_id VARCHAR(100) NOT NULL,
    user_role VARCHAR(50) NOT NULL,
    user_name VARCHAR(200),

    -- Acknowledgment Details
    statement TEXT NOT NULL,
    risk_understood BOOLEAN DEFAULT TRUE,
    comments TEXT,

    -- Timestamps
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_acknowledgments_patient ON acknowledgments(patient_id);
CREATE INDEX idx_acknowledgments_user ON acknowledgments(user_id);
CREATE INDEX idx_acknowledgments_rule ON acknowledgments(rule_code);
CREATE INDEX idx_acknowledgments_acknowledged_at ON acknowledgments(acknowledged_at);

-- ============================================================================
-- ESCALATIONS TABLE
-- ============================================================================
CREATE TABLE escalations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- References
    violation_id VARCHAR(100) NOT NULL,
    trail_id VARCHAR(100) REFERENCES evidence_trails(trail_id),
    patient_id VARCHAR(100) NOT NULL,

    -- Escalation Details
    level VARCHAR(50) NOT NULL,
    severity severity_level NOT NULL,
    reason TEXT NOT NULL,

    -- Escalation Path
    current_level_index INTEGER DEFAULT 0,
    escalation_path TEXT[] NOT NULL DEFAULT '{}',

    -- Status
    status escalation_status NOT NULL DEFAULT 'OPEN',

    -- Requestor
    requestor_id VARCHAR(100) NOT NULL,
    requestor_role VARCHAR(50),

    -- Resolution
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMPTZ,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,
    resolution TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_escalations_patient ON escalations(patient_id);
CREATE INDEX idx_escalations_status ON escalations(status);
CREATE INDEX idx_escalations_level ON escalations(level);
CREATE INDEX idx_escalations_severity ON escalations(severity);
CREATE INDEX idx_escalations_created_at ON escalations(created_at);

-- ============================================================================
-- OVERRIDE PATTERNS TABLE
-- For monitoring override abuse patterns
-- ============================================================================
CREATE TABLE override_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Pattern Key
    requestor_id VARCHAR(100) NOT NULL,
    rule_code VARCHAR(50) NOT NULL,

    -- Counts
    count_24h INTEGER DEFAULT 0,
    count_7d INTEGER DEFAULT 0,
    approved_count INTEGER DEFAULT 0,
    denied_count INTEGER DEFAULT 0,

    -- Flagging
    flagged BOOLEAN DEFAULT FALSE,
    flag_reason TEXT,
    flagged_at TIMESTAMPTZ,

    -- Timestamps
    last_request TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT override_patterns_unique UNIQUE (requestor_id, rule_code)
);

CREATE INDEX idx_override_patterns_requestor ON override_patterns(requestor_id);
CREATE INDEX idx_override_patterns_flagged ON override_patterns(flagged) WHERE flagged = TRUE;

-- ============================================================================
-- ENGINE STATISTICS TABLE
-- For tracking governance engine performance
-- ============================================================================
CREATE TABLE engine_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Date for aggregation
    stat_date DATE NOT NULL DEFAULT CURRENT_DATE,
    stat_hour INTEGER DEFAULT EXTRACT(HOUR FROM NOW()),

    -- Counts
    total_evaluations BIGINT DEFAULT 0,
    total_violations BIGINT DEFAULT 0,
    total_blocked BIGINT DEFAULT 0,
    total_allowed BIGINT DEFAULT 0,

    -- By Category (JSONB for flexibility)
    by_program JSONB DEFAULT '{}',
    by_severity JSONB DEFAULT '{}',
    by_category JSONB DEFAULT '{}',

    -- Performance
    avg_evaluation_time_ms NUMERIC(10,2),
    max_evaluation_time_ms NUMERIC(10,2),
    min_evaluation_time_ms NUMERIC(10,2),

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT engine_statistics_unique UNIQUE (stat_date, stat_hour)
);

CREATE INDEX idx_engine_statistics_date ON engine_statistics(stat_date);

-- ============================================================================
-- AUDIT LOG TABLE
-- Complete audit trail for all governance actions
-- ============================================================================
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Action Details
    action_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,

    -- Actor
    actor_id VARCHAR(100) NOT NULL,
    actor_role VARCHAR(50),
    actor_name VARCHAR(200),

    -- Change Details
    old_value JSONB,
    new_value JSONB,

    -- Context
    patient_id VARCHAR(100),
    request_id VARCHAR(100),
    ip_address INET,
    user_agent TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_action ON audit_log(action_type);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_actor ON audit_log(actor_id);
CREATE INDEX idx_audit_log_patient ON audit_log(patient_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_override_requests_updated_at
    BEFORE UPDATE ON override_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_escalations_updated_at
    BEFORE UPDATE ON escalations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_override_patterns_updated_at
    BEFORE UPDATE ON override_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_engine_statistics_updated_at
    BEFORE UPDATE ON engine_statistics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- INITIAL DATA
-- ============================================================================

-- Insert initial statistics row for today
INSERT INTO engine_statistics (stat_date, stat_hour)
VALUES (CURRENT_DATE, EXTRACT(HOUR FROM NOW())::INTEGER)
ON CONFLICT (stat_date, stat_hour) DO NOTHING;

-- ============================================================================
-- COMMENTS
-- ============================================================================
COMMENT ON TABLE evidence_trails IS 'Immutable audit records for all governance evaluations with cryptographic hashes';
COMMENT ON TABLE override_requests IS 'Override requests for blocked orders requiring governance approval';
COMMENT ON TABLE acknowledgments IS 'User acknowledgments of warnings before proceeding';
COMMENT ON TABLE escalations IS 'Mandatory escalations requiring supervisor intervention';
COMMENT ON TABLE override_patterns IS 'Tracking patterns of override requests for abuse detection';
COMMENT ON TABLE engine_statistics IS 'Hourly aggregated statistics for governance engine performance';
COMMENT ON TABLE audit_log IS 'Complete audit trail for all governance-related actions';

-- Print success message
DO $$
BEGIN
    RAISE NOTICE 'KB-18 Governance Engine schema created successfully!';
END $$;
