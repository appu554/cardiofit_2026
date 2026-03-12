-- Initial schema migration for Workflow Engine Service
-- This script will be automatically executed when the PostgreSQL container starts

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workflow definitions table
CREATE TABLE workflow_definitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    description TEXT,
    bpmn_data TEXT NOT NULL,
    variables JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    active BOOLEAN DEFAULT true,
    tags TEXT[] DEFAULT '{}',
    category VARCHAR(100),
    CONSTRAINT uk_workflow_definitions_name_version UNIQUE(name, version)
);

-- Workflow status enum
CREATE TYPE workflow_status AS ENUM (
    'pending', 'running', 'completed', 'failed', 'cancelled', 'suspended'
);

-- Workflow instances table
CREATE TABLE workflow_instances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    definition_id UUID REFERENCES workflow_definitions(id) ON DELETE CASCADE,
    patient_id VARCHAR(255) NOT NULL,
    status workflow_status DEFAULT 'pending',
    variables JSONB DEFAULT '{}',
    start_time TIMESTAMPTZ DEFAULT NOW(),
    end_time TIMESTAMPTZ,
    correlation_id VARCHAR(255) NOT NULL,
    snapshot_chain JSONB,
    parent_instance_id UUID REFERENCES workflow_instances(id),
    business_key VARCHAR(255),
    priority INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(255),
    tags TEXT[] DEFAULT '{}'
);

-- Snapshot status enum
CREATE TYPE snapshot_status AS ENUM (
    'created', 'active', 'expired', 'archived', 'corrupted'
);

-- Workflow phase enum
CREATE TYPE workflow_phase AS ENUM (
    'calculate', 'validate', 'commit', 'override'
);

-- Snapshots table
CREATE TABLE snapshots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    snapshot_id VARCHAR(255) UNIQUE NOT NULL,
    checksum VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    status snapshot_status DEFAULT 'created',
    phase_created workflow_phase NOT NULL,
    patient_id VARCHAR(255) NOT NULL,
    context_version VARCHAR(100) NOT NULL,
    metadata JSONB DEFAULT '{}',
    data JSONB NOT NULL
);

-- Task status enum
CREATE TYPE task_status AS ENUM (
    'created', 'assigned', 'in_progress', 'completed', 'cancelled', 'escalated'
);

-- Workflow tasks table
CREATE TABLE workflow_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id) ON DELETE CASCADE,
    task_definition_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    assignee_id VARCHAR(255),
    candidate_groups TEXT[] DEFAULT '{}',
    status task_status DEFAULT 'created',
    variables JSONB DEFAULT '{}',
    form_key VARCHAR(255),
    due_date TIMESTAMPTZ,
    follow_up_date TIMESTAMPTZ,
    priority INTEGER DEFAULT 50,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    completed_by VARCHAR(255)
);

-- Workflow events table (audit trail)
CREATE TABLE workflow_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id) ON DELETE CASCADE,
    task_id UUID REFERENCES workflow_tasks(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB DEFAULT '{}',
    user_id VARCHAR(255),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    correlation_id VARCHAR(255)
);

-- Workflow timers table
CREATE TABLE workflow_timers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id) ON DELETE CASCADE,
    timer_name VARCHAR(255) NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    executed_at TIMESTAMPTZ,
    configuration JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'scheduled',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Recipe references table
CREATE TABLE recipe_references (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    recipe_id VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    resolved_at TIMESTAMPTZ DEFAULT NOW(),
    resolution_source VARCHAR(50) NOT NULL,
    metadata JSONB DEFAULT '{}'
);

-- Evidence envelopes table
CREATE TABLE evidence_envelopes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evidence_id VARCHAR(255) UNIQUE NOT NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    phase workflow_phase NOT NULL,
    evidence_type VARCHAR(100) NOT NULL,
    content JSONB NOT NULL,
    confidence_score NUMERIC(3,2) NOT NULL CHECK (confidence_score >= 0 AND confidence_score <= 1),
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    source VARCHAR(100) NOT NULL
);

-- Clinical overrides table
CREATE TABLE clinical_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    override_id VARCHAR(255) UNIQUE NOT NULL,
    workflow_id VARCHAR(255) NOT NULL,
    snapshot_id VARCHAR(255) NOT NULL,
    override_type VARCHAR(100) NOT NULL,
    original_verdict VARCHAR(50) NOT NULL,
    overridden_to VARCHAR(50) NOT NULL,
    clinician_id VARCHAR(255) NOT NULL,
    justification TEXT NOT NULL,
    override_tokens TEXT[] DEFAULT '{}',
    override_timestamp TIMESTAMPTZ DEFAULT NOW(),
    patient_context JSONB DEFAULT '{}'
);

-- Workflow metrics table
CREATE TABLE workflow_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_instance_id UUID REFERENCES workflow_instances(id) ON DELETE CASCADE,
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC NOT NULL,
    recorded_at TIMESTAMPTZ DEFAULT NOW(),
    correlation_id VARCHAR(255)
);

-- Create indexes for better performance
CREATE INDEX idx_workflow_instances_patient_id ON workflow_instances(patient_id);
CREATE INDEX idx_workflow_instances_status ON workflow_instances(status);
CREATE INDEX idx_workflow_instances_correlation_id ON workflow_instances(correlation_id);
CREATE INDEX idx_workflow_instances_created_at ON workflow_instances(created_at);
CREATE INDEX idx_workflow_instances_definition_id ON workflow_instances(definition_id);
CREATE INDEX idx_workflow_instances_business_key ON workflow_instances(business_key) WHERE business_key IS NOT NULL;

CREATE INDEX idx_workflow_definitions_name ON workflow_definitions(name);
CREATE INDEX idx_workflow_definitions_active ON workflow_definitions(active);
CREATE INDEX idx_workflow_definitions_category ON workflow_definitions(category) WHERE category IS NOT NULL;

CREATE INDEX idx_snapshots_snapshot_id ON snapshots(snapshot_id);
CREATE INDEX idx_snapshots_patient_id ON snapshots(patient_id);
CREATE INDEX idx_snapshots_expires_at ON snapshots(expires_at);
CREATE INDEX idx_snapshots_status ON snapshots(status);
CREATE INDEX idx_snapshots_phase_created ON snapshots(phase_created);

CREATE INDEX idx_workflow_tasks_workflow_instance_id ON workflow_tasks(workflow_instance_id);
CREATE INDEX idx_workflow_tasks_assignee_id ON workflow_tasks(assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX idx_workflow_tasks_status ON workflow_tasks(status);
CREATE INDEX idx_workflow_tasks_due_date ON workflow_tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_workflow_tasks_priority ON workflow_tasks(priority);
CREATE INDEX idx_workflow_tasks_candidate_groups ON workflow_tasks USING GIN(candidate_groups);

CREATE INDEX idx_workflow_events_workflow_instance_id ON workflow_events(workflow_instance_id);
CREATE INDEX idx_workflow_events_task_id ON workflow_events(task_id) WHERE task_id IS NOT NULL;
CREATE INDEX idx_workflow_events_event_type ON workflow_events(event_type);
CREATE INDEX idx_workflow_events_timestamp ON workflow_events(timestamp);
CREATE INDEX idx_workflow_events_correlation_id ON workflow_events(correlation_id) WHERE correlation_id IS NOT NULL;

CREATE INDEX idx_workflow_timers_workflow_instance_id ON workflow_timers(workflow_instance_id);
CREATE INDEX idx_workflow_timers_scheduled_at ON workflow_timers(scheduled_at);
CREATE INDEX idx_workflow_timers_status ON workflow_timers(status);

CREATE INDEX idx_workflow_metrics_workflow_instance_id ON workflow_metrics(workflow_instance_id);
CREATE INDEX idx_workflow_metrics_correlation_id ON workflow_metrics(correlation_id) WHERE correlation_id IS NOT NULL;
CREATE INDEX idx_workflow_metrics_metric_name ON workflow_metrics(metric_name);
CREATE INDEX idx_workflow_metrics_recorded_at ON workflow_metrics(recorded_at);

CREATE INDEX idx_evidence_envelopes_snapshot_id ON evidence_envelopes(snapshot_id);
CREATE INDEX idx_evidence_envelopes_phase ON evidence_envelopes(phase);
CREATE INDEX idx_evidence_envelopes_evidence_type ON evidence_envelopes(evidence_type);

CREATE INDEX idx_clinical_overrides_workflow_id ON clinical_overrides(workflow_id);
CREATE INDEX idx_clinical_overrides_snapshot_id ON clinical_overrides(snapshot_id);
CREATE INDEX idx_clinical_overrides_clinician_id ON clinical_overrides(clinician_id);
CREATE INDEX idx_clinical_overrides_override_timestamp ON clinical_overrides(override_timestamp);

-- Create functions and triggers for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply the trigger to tables with updated_at columns
CREATE TRIGGER update_workflow_definitions_updated_at
    BEFORE UPDATE ON workflow_definitions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_instances_updated_at
    BEFORE UPDATE ON workflow_instances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_tasks_updated_at
    BEFORE UPDATE ON workflow_tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically expire snapshots
CREATE OR REPLACE FUNCTION expire_old_snapshots()
RETURNS void AS $$
BEGIN
    UPDATE snapshots
    SET status = 'expired'
    WHERE status = 'active' AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to calculate workflow metrics automatically
CREATE OR REPLACE FUNCTION calculate_workflow_duration()
RETURNS TRIGGER AS $$
BEGIN
    -- Only insert metric when workflow actually ends (not when just updating)
    IF NEW.end_time IS NOT NULL AND (OLD.end_time IS NULL OR OLD.end_time != NEW.end_time) THEN
        INSERT INTO workflow_metrics (workflow_instance_id, metric_name, metric_value, correlation_id)
        VALUES (
            NEW.id,
            'duration_seconds',
            EXTRACT(EPOCH FROM (NEW.end_time - NEW.start_time)),
            NEW.correlation_id
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically calculate duration when workflow completes
CREATE TRIGGER calculate_workflow_duration_trigger
    AFTER UPDATE ON workflow_instances
    FOR EACH ROW
    WHEN (NEW.end_time IS NOT NULL)
    EXECUTE FUNCTION calculate_workflow_duration();

-- Insert some sample workflow definitions for testing
INSERT INTO workflow_definitions (name, version, description, bpmn_data, category) VALUES
('medication-ordering', '1.0', 'Standard medication ordering workflow with safety validation', 
 '<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions></bpmn:definitions>', 'clinical'),
('patient-admission', '1.0', 'Patient admission workflow with documentation requirements',
 '<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions></bpmn:definitions>', 'administrative'),
('emergency-response', '1.0', 'Emergency response protocol with escalation procedures',
 '<?xml version="1.0" encoding="UTF-8"?><bpmn:definitions></bpmn:definitions>', 'emergency');

-- Create a view for workflow summary statistics
CREATE OR REPLACE VIEW workflow_summary AS
SELECT 
    wd.name as workflow_name,
    wd.version,
    wd.category,
    COUNT(wi.id) as total_instances,
    COUNT(CASE WHEN wi.status = 'completed' THEN 1 END) as completed_instances,
    COUNT(CASE WHEN wi.status = 'failed' THEN 1 END) as failed_instances,
    COUNT(CASE WHEN wi.status = 'running' THEN 1 END) as running_instances,
    AVG(CASE WHEN wi.end_time IS NOT NULL THEN 
        EXTRACT(EPOCH FROM (wi.end_time - wi.start_time)) 
    END) as avg_duration_seconds
FROM workflow_definitions wd
LEFT JOIN workflow_instances wi ON wd.id = wi.definition_id
WHERE wd.active = true
GROUP BY wd.id, wd.name, wd.version, wd.category;

-- Grant necessary permissions
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO workflow_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO workflow_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO workflow_user;