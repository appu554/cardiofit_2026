-- Migration: Create workflow engine tables in Supabase PostgreSQL
-- This script creates the necessary tables for workflow state management

-- Create workflow_definitions table
CREATE TABLE IF NOT EXISTS workflow_definitions (
    id SERIAL PRIMARY KEY,
    fhir_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'draft',
    category VARCHAR(100),
    bpmn_xml TEXT,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255)
);

-- Create index on fhir_id for fast lookups
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_fhir_id ON workflow_definitions(fhir_id);
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_status ON workflow_definitions(status);
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_category ON workflow_definitions(category);

-- Create workflow_instances table
CREATE TABLE IF NOT EXISTS workflow_instances (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    definition_id INTEGER REFERENCES workflow_definitions(id),
    patient_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    start_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE,
    variables JSONB DEFAULT '{}',
    context JSONB DEFAULT '{}',
    created_by VARCHAR(255)
);

-- Create indexes for workflow_instances
CREATE INDEX IF NOT EXISTS idx_workflow_instances_external_id ON workflow_instances(external_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_patient_id ON workflow_instances(patient_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_status ON workflow_instances(status);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_definition_id ON workflow_instances(definition_id);

-- Create workflow_tasks table
CREATE TABLE IF NOT EXISTS workflow_tasks (
    id SERIAL PRIMARY KEY,
    fhir_id VARCHAR(255) UNIQUE NOT NULL,
    external_id VARCHAR(255),
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    task_definition_key VARCHAR(255),
    name VARCHAR(255),
    description TEXT,
    status VARCHAR(50) DEFAULT 'ready',
    priority VARCHAR(20) DEFAULT 'routine',
    assignee VARCHAR(255),
    candidate_groups JSONB DEFAULT '[]',
    due_date TIMESTAMP WITH TIME ZONE,
    follow_up_date TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    input_variables JSONB DEFAULT '{}',
    output_variables JSONB DEFAULT '{}'
);

-- Create indexes for workflow_tasks
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_fhir_id ON workflow_tasks(fhir_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_external_id ON workflow_tasks(external_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_workflow_instance_id ON workflow_tasks(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_status ON workflow_tasks(status);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_assignee ON workflow_tasks(assignee);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_due_date ON workflow_tasks(due_date);

-- Create workflow_events table for audit trail
CREATE TABLE IF NOT EXISTS workflow_events (
    id SERIAL PRIMARY KEY,
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    task_id INTEGER REFERENCES workflow_tasks(id),
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id VARCHAR(255),
    source VARCHAR(100)
);

-- Create indexes for workflow_events
CREATE INDEX IF NOT EXISTS idx_workflow_events_workflow_instance_id ON workflow_events(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_events_task_id ON workflow_events(task_id);
CREATE INDEX IF NOT EXISTS idx_workflow_events_event_type ON workflow_events(event_type);
CREATE INDEX IF NOT EXISTS idx_workflow_events_timestamp ON workflow_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_workflow_events_user_id ON workflow_events(user_id);

-- Create workflow_timers table
CREATE TABLE IF NOT EXISTS workflow_timers (
    id SERIAL PRIMARY KEY,
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    timer_name VARCHAR(255),
    due_date TIMESTAMP WITH TIME ZONE NOT NULL,
    repeat_interval VARCHAR(100),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    fired_at TIMESTAMP WITH TIME ZONE,
    timer_data JSONB DEFAULT '{}'
);

-- Create indexes for workflow_timers
CREATE INDEX IF NOT EXISTS idx_workflow_timers_workflow_instance_id ON workflow_timers(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_timers_due_date ON workflow_timers(due_date);
CREATE INDEX IF NOT EXISTS idx_workflow_timers_status ON workflow_timers(status);

-- Create task_assignments table
CREATE TABLE IF NOT EXISTS task_assignments (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES workflow_tasks(id),
    assignee_id VARCHAR(255) NOT NULL,
    assigned_by VARCHAR(255),
    assignment_type VARCHAR(50) DEFAULT 'direct',
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    revoked_at TIMESTAMP WITH TIME ZONE,
    revoked_by VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    notes TEXT
);

-- Create indexes for task_assignments
CREATE INDEX IF NOT EXISTS idx_task_assignments_task_id ON task_assignments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_assignments_assignee_id ON task_assignments(assignee_id);
CREATE INDEX IF NOT EXISTS idx_task_assignments_is_active ON task_assignments(is_active);

-- Create task_comments table
CREATE TABLE IF NOT EXISTS task_comments (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES workflow_tasks(id),
    author_id VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_internal BOOLEAN DEFAULT FALSE
);

-- Create indexes for task_comments
CREATE INDEX IF NOT EXISTS idx_task_comments_task_id ON task_comments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_comments_author_id ON task_comments(author_id);
CREATE INDEX IF NOT EXISTS idx_task_comments_created_at ON task_comments(created_at);

-- Create task_attachments table
CREATE TABLE IF NOT EXISTS task_attachments (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES workflow_tasks(id),
    filename VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size INTEGER,
    mime_type VARCHAR(100),
    uploaded_by VARCHAR(255) NOT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description TEXT
);

-- Create indexes for task_attachments
CREATE INDEX IF NOT EXISTS idx_task_attachments_task_id ON task_attachments(task_id);
CREATE INDEX IF NOT EXISTS idx_task_attachments_uploaded_by ON task_attachments(uploaded_by);

-- Create task_escalations table
CREATE TABLE IF NOT EXISTS task_escalations (
    id SERIAL PRIMARY KEY,
    task_id INTEGER REFERENCES workflow_tasks(id),
    escalation_level INTEGER DEFAULT 1,
    escalated_to VARCHAR(255) NOT NULL,
    escalated_by VARCHAR(255),
    escalation_reason VARCHAR(255),
    escalated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    is_active BOOLEAN DEFAULT TRUE
);

-- Create indexes for task_escalations
CREATE INDEX IF NOT EXISTS idx_task_escalations_task_id ON task_escalations(task_id);
CREATE INDEX IF NOT EXISTS idx_task_escalations_escalated_to ON task_escalations(escalated_to);
CREATE INDEX IF NOT EXISTS idx_task_escalations_is_active ON task_escalations(is_active);

-- Create workflow_events_log table for analytics (separate from audit trail)
CREATE TABLE IF NOT EXISTS workflow_events_log (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    workflow_definition_id INTEGER,
    workflow_instance_id INTEGER,
    task_id INTEGER,
    user_id VARCHAR(255),
    patient_id VARCHAR(255),
    event_data JSONB DEFAULT '{}',
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source VARCHAR(100)
);

-- Create indexes for workflow_events_log
CREATE INDEX IF NOT EXISTS idx_workflow_events_log_event_type ON workflow_events_log(event_type);
CREATE INDEX IF NOT EXISTS idx_workflow_events_log_timestamp ON workflow_events_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_workflow_events_log_user_id ON workflow_events_log(user_id);
CREATE INDEX IF NOT EXISTS idx_workflow_events_log_patient_id ON workflow_events_log(patient_id);

-- Create triggers to automatically update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply the trigger to relevant tables
CREATE TRIGGER update_workflow_definitions_updated_at BEFORE UPDATE ON workflow_definitions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_workflow_tasks_updated_at BEFORE UPDATE ON workflow_tasks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_task_comments_updated_at BEFORE UPDATE ON task_comments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grant necessary permissions (adjust as needed for your Supabase setup)
-- These might need to be run with appropriate privileges
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;
