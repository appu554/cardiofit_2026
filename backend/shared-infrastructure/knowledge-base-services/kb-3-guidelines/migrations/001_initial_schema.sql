-- KB-3 Guidelines Service Database Schema
-- Version: 1.0.0
-- Description: Initial schema for guidelines, temporal logic, and pathway management

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ================================================
-- GUIDELINES AND GOVERNANCE
-- ================================================

-- Guidelines table
CREATE TABLE IF NOT EXISTS guidelines (
    guideline_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    source VARCHAR(255) NOT NULL,
    version VARCHAR(32) NOT NULL,
    effective_date TIMESTAMP,
    status VARCHAR(32) DEFAULT 'draft',
    domain VARCHAR(64),
    evidence_grade VARCHAR(8),
    recommendations JSONB,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_guidelines_active ON guidelines(active);
CREATE INDEX idx_guidelines_domain ON guidelines(domain);
CREATE INDEX idx_guidelines_source ON guidelines(source);

-- Conflicts table
CREATE TABLE IF NOT EXISTS conflicts (
    conflict_id VARCHAR(64) PRIMARY KEY,
    guideline1_id VARCHAR(64) REFERENCES guidelines(guideline_id),
    guideline2_id VARCHAR(64) REFERENCES guidelines(guideline_id),
    recommendation1 JSONB,
    recommendation2 JSONB,
    type VARCHAR(32) NOT NULL,
    severity VARCHAR(16) NOT NULL,
    domain VARCHAR(64),
    status VARCHAR(32) DEFAULT 'detected',
    resolution JSONB,
    detected_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP
);

CREATE INDEX idx_conflicts_status ON conflicts(status);
CREATE INDEX idx_conflicts_severity ON conflicts(severity);

-- Safety overrides table
CREATE TABLE IF NOT EXISTS safety_overrides (
    override_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    trigger_conditions JSONB NOT NULL,
    override_action JSONB NOT NULL,
    priority INT DEFAULT 100,
    active BOOLEAN DEFAULT true,
    affected_guidelines JSONB,
    requires_signature BOOLEAN DEFAULT false,
    clinical_rationale TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_safety_overrides_active ON safety_overrides(active);
CREATE INDEX idx_safety_overrides_priority ON safety_overrides(priority);

-- Guideline versions table
CREATE TABLE IF NOT EXISTS guideline_versions (
    version_id VARCHAR(64) PRIMARY KEY,
    guideline_id VARCHAR(64) REFERENCES guidelines(guideline_id),
    version VARCHAR(32) NOT NULL,
    change_type VARCHAR(32) NOT NULL,
    changes JSONB,
    clinical_impact JSONB,
    approval_chain JSONB,
    transition_plan JSONB,
    status VARCHAR(32) DEFAULT 'draft',
    created_by VARCHAR(64),
    created_at TIMESTAMP DEFAULT NOW(),
    effective_date TIMESTAMP
);

CREATE INDEX idx_guideline_versions_guideline ON guideline_versions(guideline_id);
CREATE INDEX idx_guideline_versions_status ON guideline_versions(status);

-- ================================================
-- PROTOCOLS AND PATHWAYS
-- ================================================

-- Protocol definitions table
CREATE TABLE IF NOT EXISTS protocols (
    protocol_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    protocol_type VARCHAR(20) NOT NULL, -- acute, chronic, preventive
    guideline_source VARCHAR(255),
    definition JSONB NOT NULL,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_protocols_type ON protocols(protocol_type);
CREATE INDEX idx_protocols_active ON protocols(active);

-- Pathway instances table
CREATE TABLE IF NOT EXISTS pathway_instances (
    instance_id VARCHAR(64) PRIMARY KEY,
    pathway_id VARCHAR(64) NOT NULL REFERENCES protocols(protocol_id),
    patient_id VARCHAR(64) NOT NULL,
    current_stage VARCHAR(64),
    status VARCHAR(20) DEFAULT 'active', -- active, completed, suspended, cancelled
    context JSONB,
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    suspended_at TIMESTAMP,
    CONSTRAINT fk_protocol FOREIGN KEY (pathway_id) REFERENCES protocols(protocol_id)
);

CREATE INDEX idx_pathway_instances_patient ON pathway_instances(patient_id);
CREATE INDEX idx_pathway_instances_status ON pathway_instances(status);
CREATE INDEX idx_pathway_instances_pathway ON pathway_instances(pathway_id);

-- Pathway actions table
CREATE TABLE IF NOT EXISTS pathway_actions (
    action_id VARCHAR(64) PRIMARY KEY,
    instance_id VARCHAR(64) NOT NULL REFERENCES pathway_instances(instance_id),
    name VARCHAR(255) NOT NULL,
    action_type VARCHAR(50),
    status VARCHAR(20) DEFAULT 'pending', -- pending, met, approaching, overdue, missed
    deadline TIMESTAMP NOT NULL,
    grace_period INTERVAL,
    required BOOLEAN DEFAULT true,
    stage_id VARCHAR(64),
    description TEXT,
    completed_at TIMESTAMP,
    completed_by VARCHAR(64),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_pathway_actions_instance ON pathway_actions(instance_id);
CREATE INDEX idx_pathway_actions_status ON pathway_actions(status);
CREATE INDEX idx_pathway_actions_deadline ON pathway_actions(deadline);

-- ================================================
-- SCHEDULING
-- ================================================

-- Scheduled items table
CREATE TABLE IF NOT EXISTS scheduled_items (
    item_id VARCHAR(64) PRIMARY KEY,
    patient_id VARCHAR(64) NOT NULL,
    item_type VARCHAR(50) NOT NULL, -- lab, appointment, medication, procedure, screening, assessment
    name VARCHAR(255) NOT NULL,
    description TEXT,
    due_date TIMESTAMP NOT NULL,
    priority INT DEFAULT 2, -- 1=highest, 5=lowest
    is_recurring BOOLEAN DEFAULT false,
    recurrence JSONB,
    status VARCHAR(20) DEFAULT 'pending', -- pending, completed, overdue, cancelled, skipped
    completed_at TIMESTAMP,
    source_protocol VARCHAR(64),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_scheduled_items_patient ON scheduled_items(patient_id);
CREATE INDEX idx_scheduled_items_due ON scheduled_items(due_date);
CREATE INDEX idx_scheduled_items_status ON scheduled_items(status);
CREATE INDEX idx_scheduled_items_type ON scheduled_items(item_type);
CREATE INDEX idx_scheduled_items_priority ON scheduled_items(priority);

-- ================================================
-- AUDIT LOGGING
-- ================================================

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_log (
    entry_id VARCHAR(64) PRIMARY KEY,
    action VARCHAR(100) NOT NULL,
    user_id VARCHAR(64),
    checksum VARCHAR(64),
    timestamp TIMESTAMP DEFAULT NOW(),
    details JSONB,
    resource_type VARCHAR(50),
    resource_id VARCHAR(64)
);

CREATE INDEX idx_audit_log_timestamp ON audit_log(timestamp);
CREATE INDEX idx_audit_log_action ON audit_log(action);
CREATE INDEX idx_audit_log_user ON audit_log(user_id);
CREATE INDEX idx_audit_log_resource ON audit_log(resource_type, resource_id);

-- ================================================
-- HELPER FUNCTIONS
-- ================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_guidelines_updated_at
    BEFORE UPDATE ON guidelines
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_safety_overrides_updated_at
    BEFORE UPDATE ON safety_overrides
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_protocols_updated_at
    BEFORE UPDATE ON protocols
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scheduled_items_updated_at
    BEFORE UPDATE ON scheduled_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ================================================
-- VIEWS
-- ================================================

-- View for overdue actions
CREATE OR REPLACE VIEW v_overdue_actions AS
SELECT
    pa.action_id,
    pa.instance_id,
    pi.patient_id,
    pi.pathway_id,
    pa.name AS action_name,
    pa.deadline,
    NOW() - pa.deadline AS overdue_by,
    pa.status,
    CASE
        WHEN NOW() - pa.deadline > INTERVAL '2 hours' THEN 'critical'
        WHEN NOW() - pa.deadline > INTERVAL '30 minutes' THEN 'major'
        ELSE 'warning'
    END AS severity
FROM pathway_actions pa
JOIN pathway_instances pi ON pa.instance_id = pi.instance_id
WHERE pa.status = 'pending'
  AND pa.deadline < NOW()
  AND pi.status = 'active'
ORDER BY pa.deadline ASC;

-- View for overdue scheduled items
CREATE OR REPLACE VIEW v_overdue_scheduled_items AS
SELECT
    si.item_id,
    si.patient_id,
    si.name,
    si.item_type,
    si.due_date,
    NOW() - si.due_date AS overdue_by,
    si.priority,
    si.source_protocol
FROM scheduled_items si
WHERE si.status = 'pending'
  AND si.due_date < NOW()
ORDER BY si.priority ASC, si.due_date ASC;

-- View for patient schedule summary
CREATE OR REPLACE VIEW v_patient_schedule_summary AS
SELECT
    patient_id,
    COUNT(*) AS total_items,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) AS pending_items,
    COUNT(CASE WHEN status = 'pending' AND due_date < NOW() THEN 1 END) AS overdue_items,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) AS completed_items,
    COUNT(CASE WHEN status = 'pending' AND due_date <= NOW() + INTERVAL '7 days' THEN 1 END) AS upcoming_week,
    COUNT(CASE WHEN status = 'pending' AND due_date <= NOW() + INTERVAL '30 days' THEN 1 END) AS upcoming_month
FROM scheduled_items
GROUP BY patient_id;

-- ================================================
-- COMMENTS
-- ================================================

COMMENT ON TABLE guidelines IS 'Clinical guideline definitions from authoritative sources';
COMMENT ON TABLE conflicts IS 'Detected conflicts between guidelines with resolution tracking';
COMMENT ON TABLE safety_overrides IS 'Safety override rules that take precedence over guidelines';
COMMENT ON TABLE guideline_versions IS 'Version history and approval workflow for guidelines';
COMMENT ON TABLE protocols IS 'Clinical protocol definitions (acute, chronic, preventive)';
COMMENT ON TABLE pathway_instances IS 'Active patient pathway instances';
COMMENT ON TABLE pathway_actions IS 'Individual actions within pathway instances';
COMMENT ON TABLE scheduled_items IS 'Scheduled care items for patients (labs, appointments, etc.)';
COMMENT ON TABLE audit_log IS 'Compliance audit log for all operations';
