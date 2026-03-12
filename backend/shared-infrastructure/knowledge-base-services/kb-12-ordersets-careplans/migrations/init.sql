-- KB-12 Order Sets & Care Plans Database Schema
-- Initial migration for PostgreSQL

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- ============================================
-- ORDER SET TEMPLATES
-- ============================================

CREATE TABLE IF NOT EXISTS order_set_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(20) NOT NULL DEFAULT '1.0',
    category VARCHAR(50) NOT NULL,
    subcategory VARCHAR(50),
    specialty VARCHAR(50),
    condition_code VARCHAR(20),
    condition_system VARCHAR(100),
    condition_display VARCHAR(255),
    sections JSONB NOT NULL DEFAULT '[]',
    time_constraints JSONB DEFAULT '[]',
    clinical_context JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    approved_by VARCHAR(100),
    approved_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for order_set_templates
CREATE INDEX idx_order_set_category ON order_set_templates(category);
CREATE INDEX idx_order_set_condition ON order_set_templates(condition_code);
CREATE INDEX idx_order_set_status ON order_set_templates(status);
CREATE INDEX idx_order_set_name_trgm ON order_set_templates USING gin(name gin_trgm_ops);

-- ============================================
-- CARE PLAN TEMPLATES
-- ============================================

CREATE TABLE IF NOT EXISTS care_plan_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    version VARCHAR(20) NOT NULL DEFAULT '1.0',
    category VARCHAR(50) NOT NULL,
    subcategory VARCHAR(50),
    condition_code VARCHAR(20),
    condition_system VARCHAR(100),
    condition_display VARCHAR(255),
    goals JSONB NOT NULL DEFAULT '[]',
    activities JSONB NOT NULL DEFAULT '[]',
    monitoring JSONB NOT NULL DEFAULT '[]',
    duration VARCHAR(50),
    review_period VARCHAR(50),
    guidelines JSONB DEFAULT '[]',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(100),
    approved_by VARCHAR(100),
    approved_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for care_plan_templates
CREATE INDEX idx_care_plan_category ON care_plan_templates(category);
CREATE INDEX idx_care_plan_condition ON care_plan_templates(condition_code);
CREATE INDEX idx_care_plan_status ON care_plan_templates(status);
CREATE INDEX idx_care_plan_name_trgm ON care_plan_templates USING gin(name gin_trgm_ops);

-- ============================================
-- ORDER SESSIONS (CPOE)
-- ============================================

CREATE TABLE IF NOT EXISTS order_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id VARCHAR(100) UNIQUE NOT NULL,
    patient_id VARCHAR(100) NOT NULL,
    encounter_id VARCHAR(100),
    provider_id VARCHAR(100) NOT NULL,
    order_set_id VARCHAR(50),
    orders JSONB NOT NULL DEFAULT '[]',
    alerts JSONB DEFAULT '[]',
    validations JSONB DEFAULT '[]',
    patient_context JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    signed_at TIMESTAMP WITH TIME ZONE,
    signed_by VARCHAR(100)
);

-- Indexes for order_sessions
CREATE INDEX idx_order_session_patient ON order_sessions(patient_id);
CREATE INDEX idx_order_session_encounter ON order_sessions(encounter_id);
CREATE INDEX idx_order_session_provider ON order_sessions(provider_id);
CREATE INDEX idx_order_session_status ON order_sessions(status);
CREATE INDEX idx_order_session_created ON order_sessions(created_at DESC);

-- ============================================
-- WORKFLOW INSTANCES
-- ============================================

CREATE TABLE IF NOT EXISTS workflow_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    instance_id VARCHAR(100) UNIQUE NOT NULL,
    workflow_type VARCHAR(50) NOT NULL,
    template_id VARCHAR(50) NOT NULL,
    template_name VARCHAR(255),
    patient_id VARCHAR(100) NOT NULL,
    encounter_id VARCHAR(100),
    initiated_by VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    current_phase VARCHAR(50),
    steps JSONB NOT NULL DEFAULT '[]',
    time_constraints JSONB DEFAULT '[]',
    variables JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    priority INTEGER DEFAULT 2,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for workflow_instances
CREATE INDEX idx_workflow_patient ON workflow_instances(patient_id);
CREATE INDEX idx_workflow_encounter ON workflow_instances(encounter_id);
CREATE INDEX idx_workflow_status ON workflow_instances(status);
CREATE INDEX idx_workflow_type ON workflow_instances(workflow_type);
CREATE INDEX idx_workflow_started ON workflow_instances(started_at DESC);

-- ============================================
-- TIME CONSTRAINTS
-- ============================================

CREATE TABLE IF NOT EXISTS active_time_constraints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    constraint_id VARCHAR(100) UNIQUE NOT NULL,
    instance_id VARCHAR(100) NOT NULL,
    patient_id VARCHAR(100) NOT NULL,
    action VARCHAR(255) NOT NULL,
    description TEXT,
    deadline TIMESTAMP WITH TIME ZONE NOT NULL,
    warning_time TIMESTAMP WITH TIME ZONE,
    duration_minutes INTEGER,
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    metrics_code VARCHAR(50),
    completed_at TIMESTAMP WITH TIME ZONE,
    alerts_sent INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (instance_id) REFERENCES workflow_instances(instance_id) ON DELETE CASCADE
);

-- Indexes for active_time_constraints
CREATE INDEX idx_constraint_patient ON active_time_constraints(patient_id);
CREATE INDEX idx_constraint_status ON active_time_constraints(status);
CREATE INDEX idx_constraint_deadline ON active_time_constraints(deadline);
CREATE INDEX idx_constraint_metrics ON active_time_constraints(metrics_code);

-- ============================================
-- CDS HOOKS FEEDBACK
-- ============================================

CREATE TABLE IF NOT EXISTS cds_hook_feedback (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    card_id VARCHAR(100) NOT NULL,
    hook_id VARCHAR(50) NOT NULL,
    patient_id VARCHAR(100),
    outcome VARCHAR(20) NOT NULL, -- accepted, overridden, dismissed
    override_reason VARCHAR(100),
    comments TEXT,
    user_id VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for cds_hook_feedback
CREATE INDEX idx_feedback_hook ON cds_hook_feedback(hook_id);
CREATE INDEX idx_feedback_outcome ON cds_hook_feedback(outcome);
CREATE INDEX idx_feedback_user ON cds_hook_feedback(user_id);
CREATE INDEX idx_feedback_created ON cds_hook_feedback(created_at DESC);

-- ============================================
-- CLINICAL ALERTS LOG
-- ============================================

CREATE TABLE IF NOT EXISTS clinical_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    alert_id VARCHAR(100) UNIQUE NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    category VARCHAR(50),
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    details TEXT,
    order_id VARCHAR(100),
    patient_id VARCHAR(100),
    triggering_medication VARCHAR(255),
    interacting_with VARCHAR(255),
    override_allowed BOOLEAN DEFAULT true,
    overridden BOOLEAN DEFAULT false,
    overridden_by VARCHAR(100),
    override_reason TEXT,
    source VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for clinical_alerts
CREATE INDEX idx_alert_patient ON clinical_alerts(patient_id);
CREATE INDEX idx_alert_type ON clinical_alerts(alert_type);
CREATE INDEX idx_alert_severity ON clinical_alerts(severity);
CREATE INDEX idx_alert_created ON clinical_alerts(created_at DESC);

-- ============================================
-- AUDIT LOG
-- ============================================

CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL, -- create, read, update, delete, sign, override
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for audit_log
CREATE INDEX idx_audit_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_user ON audit_log(user_id);
CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_created ON audit_log(created_at DESC);

-- ============================================
-- FUNCTIONS
-- ============================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_order_set_templates_updated_at
    BEFORE UPDATE ON order_set_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_care_plan_templates_updated_at
    BEFORE UPDATE ON care_plan_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_order_sessions_updated_at
    BEFORE UPDATE ON order_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_instances_updated_at
    BEFORE UPDATE ON workflow_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================
-- VIEWS
-- ============================================

-- Active workflows with time constraint status
CREATE OR REPLACE VIEW v_active_workflows AS
SELECT
    w.instance_id,
    w.workflow_type,
    w.template_name,
    w.patient_id,
    w.status,
    w.started_at,
    w.priority,
    COUNT(DISTINCT tc.constraint_id) AS total_constraints,
    COUNT(DISTINCT CASE WHEN tc.status = 'pending' OR tc.status = 'warning_sent' THEN tc.constraint_id END) AS active_constraints,
    COUNT(DISTINCT CASE WHEN tc.status = 'missed' THEN tc.constraint_id END) AS missed_constraints,
    MIN(CASE WHEN tc.status IN ('pending', 'warning_sent') THEN tc.deadline END) AS next_deadline
FROM workflow_instances w
LEFT JOIN active_time_constraints tc ON w.instance_id = tc.instance_id
WHERE w.status = 'active'
GROUP BY w.instance_id, w.workflow_type, w.template_name, w.patient_id, w.status, w.started_at, w.priority;

-- Template usage statistics
CREATE OR REPLACE VIEW v_template_usage AS
SELECT
    t.template_id,
    t.name,
    t.category,
    COUNT(DISTINCT w.instance_id) AS usage_count,
    COUNT(DISTINCT CASE WHEN w.status = 'completed' THEN w.instance_id END) AS completed_count,
    COUNT(DISTINCT CASE WHEN w.status = 'cancelled' THEN w.instance_id END) AS cancelled_count,
    MAX(w.started_at) AS last_used
FROM order_set_templates t
LEFT JOIN workflow_instances w ON t.template_id = w.template_id
GROUP BY t.template_id, t.name, t.category
UNION ALL
SELECT
    t.template_id,
    t.name,
    t.category,
    COUNT(DISTINCT w.instance_id) AS usage_count,
    COUNT(DISTINCT CASE WHEN w.status = 'completed' THEN w.instance_id END) AS completed_count,
    COUNT(DISTINCT CASE WHEN w.status = 'cancelled' THEN w.instance_id END) AS cancelled_count,
    MAX(w.started_at) AS last_used
FROM care_plan_templates t
LEFT JOIN workflow_instances w ON t.template_id = w.template_id
GROUP BY t.template_id, t.name, t.category;

-- Time constraint performance metrics
CREATE OR REPLACE VIEW v_time_constraint_metrics AS
SELECT
    metrics_code,
    COUNT(*) AS total_constraints,
    COUNT(CASE WHEN status = 'completed' AND completed_at <= deadline THEN 1 END) AS met_on_time,
    COUNT(CASE WHEN status = 'missed' THEN 1 END) AS missed,
    ROUND(
        100.0 * COUNT(CASE WHEN status = 'completed' AND completed_at <= deadline THEN 1 END) / NULLIF(COUNT(*), 0),
        2
    ) AS compliance_rate,
    AVG(EXTRACT(EPOCH FROM (completed_at - (deadline - (duration_minutes * interval '1 minute')))) / 60) AS avg_completion_minutes
FROM active_time_constraints
WHERE metrics_code IS NOT NULL
GROUP BY metrics_code;

-- Grant permissions (adjust as needed for your security model)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO kb12user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kb12user;
