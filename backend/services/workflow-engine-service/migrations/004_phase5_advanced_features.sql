-- Migration for Phase 5 Advanced Features
-- Creates tables for escalations, gateways, and error recovery

-- Create workflow_escalations table
CREATE TABLE IF NOT EXISTS workflow_escalations (
    id SERIAL PRIMARY KEY,
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    task_id INTEGER REFERENCES workflow_tasks(id),
    escalation_level INTEGER DEFAULT 1,
    escalation_type VARCHAR(100),
    escalation_target VARCHAR(255),
    escalation_reason VARCHAR(500),
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    escalation_data JSONB DEFAULT '{}'
);

-- Create indexes for workflow_escalations
CREATE INDEX IF NOT EXISTS idx_workflow_escalations_workflow_instance_id ON workflow_escalations(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_escalations_task_id ON workflow_escalations(task_id);
CREATE INDEX IF NOT EXISTS idx_workflow_escalations_status ON workflow_escalations(status);
CREATE INDEX IF NOT EXISTS idx_workflow_escalations_created_at ON workflow_escalations(created_at);

-- Create workflow_gateways table
CREATE TABLE IF NOT EXISTS workflow_gateways (
    id SERIAL PRIMARY KEY,
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    gateway_id VARCHAR(255) UNIQUE,
    gateway_type VARCHAR(50),
    required_tokens JSONB DEFAULT '[]',
    received_tokens JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'waiting',
    timeout_minutes INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    gateway_data JSONB DEFAULT '{}'
);

-- Create indexes for workflow_gateways
CREATE INDEX IF NOT EXISTS idx_workflow_gateways_workflow_instance_id ON workflow_gateways(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_gateways_gateway_id ON workflow_gateways(gateway_id);
CREATE INDEX IF NOT EXISTS idx_workflow_gateways_status ON workflow_gateways(status);
CREATE INDEX IF NOT EXISTS idx_workflow_gateways_created_at ON workflow_gateways(created_at);

-- Create workflow_errors table
CREATE TABLE IF NOT EXISTS workflow_errors (
    id SERIAL PRIMARY KEY,
    workflow_instance_id INTEGER REFERENCES workflow_instances(id),
    task_id INTEGER REFERENCES workflow_tasks(id),
    error_id VARCHAR(255) UNIQUE,
    error_type VARCHAR(100),
    error_message TEXT,
    recovery_strategy VARCHAR(100),
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    error_data JSONB DEFAULT '{}',
    recovery_data JSONB DEFAULT '{}'
);

-- Create indexes for workflow_errors
CREATE INDEX IF NOT EXISTS idx_workflow_errors_workflow_instance_id ON workflow_errors(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_errors_task_id ON workflow_errors(task_id);
CREATE INDEX IF NOT EXISTS idx_workflow_errors_error_id ON workflow_errors(error_id);
CREATE INDEX IF NOT EXISTS idx_workflow_errors_status ON workflow_errors(status);
CREATE INDEX IF NOT EXISTS idx_workflow_errors_created_at ON workflow_errors(created_at);
CREATE INDEX IF NOT EXISTS idx_workflow_errors_error_type ON workflow_errors(error_type);

-- Add additional indexes for workflow_timers (if not already present)
CREATE INDEX IF NOT EXISTS idx_workflow_timers_workflow_instance_id ON workflow_timers(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_timers_due_date ON workflow_timers(due_date);
CREATE INDEX IF NOT EXISTS idx_workflow_timers_status ON workflow_timers(status);

-- Add escalation fields to workflow_tasks table (if not already present)
ALTER TABLE workflow_tasks 
ADD COLUMN IF NOT EXISTS escalation_level INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS escalated BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS escalated_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS escalation_data JSONB DEFAULT '{}';

-- Create indexes for new escalation fields
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_escalated ON workflow_tasks(escalated);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_escalation_level ON workflow_tasks(escalation_level);

-- Add gateway tracking fields to workflow_instances table (if not already present)
ALTER TABLE workflow_instances 
ADD COLUMN IF NOT EXISTS active_gateways JSONB DEFAULT '[]',
ADD COLUMN IF NOT EXISTS completed_gateways JSONB DEFAULT '[]';

-- Add error tracking fields to workflow_instances table (if not already present)
ALTER TABLE workflow_instances 
ADD COLUMN IF NOT EXISTS error_count INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS recovery_attempts INTEGER DEFAULT 0;

-- Create indexes for new workflow_instances fields
CREATE INDEX IF NOT EXISTS idx_workflow_instances_error_count ON workflow_instances(error_count);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_last_error_at ON workflow_instances(last_error_at);

-- Insert initial escalation rules (optional)
INSERT INTO workflow_events (workflow_instance_id, event_type, event_data, created_at, user_id, source)
VALUES 
(0, 'system_initialization', 
 '{"phase": "5", "feature": "advanced_features", "tables_created": ["workflow_escalations", "workflow_gateways", "workflow_errors"]}',
 NOW(), 'system', 'migration')
ON CONFLICT DO NOTHING;

-- Create a view for active escalations
CREATE OR REPLACE VIEW active_escalations AS
SELECT 
    e.*,
    wi.patient_id,
    wi.status as workflow_status,
    wt.name as task_name,
    wt.assignee as task_assignee
FROM workflow_escalations e
JOIN workflow_instances wi ON e.workflow_instance_id = wi.id
LEFT JOIN workflow_tasks wt ON e.task_id = wt.id
WHERE e.status = 'active';

-- Create a view for active gateways
CREATE OR REPLACE VIEW active_gateways AS
SELECT 
    g.*,
    wi.patient_id,
    wi.status as workflow_status,
    ARRAY_LENGTH(g.required_tokens, 1) as total_tokens_required,
    ARRAY_LENGTH(g.received_tokens, 1) as tokens_received
FROM workflow_gateways g
JOIN workflow_instances wi ON g.workflow_instance_id = wi.id
WHERE g.status = 'waiting';

-- Create a view for active errors
CREATE OR REPLACE VIEW active_errors AS
SELECT 
    e.*,
    wi.patient_id,
    wi.status as workflow_status,
    wt.name as task_name,
    wt.assignee as task_assignee
FROM workflow_errors e
JOIN workflow_instances wi ON e.workflow_instance_id = wi.id
LEFT JOIN workflow_tasks wt ON e.task_id = wt.id
WHERE e.status = 'active';

-- Create a function to clean up old completed records (optional)
CREATE OR REPLACE FUNCTION cleanup_old_workflow_records()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- Clean up old completed escalations (older than 30 days)
    DELETE FROM workflow_escalations 
    WHERE status IN ('resolved', 'cancelled') 
    AND resolved_at < NOW() - INTERVAL '30 days';
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Clean up old completed gateways (older than 30 days)
    DELETE FROM workflow_gateways 
    WHERE status IN ('completed', 'timeout') 
    AND completed_at < NOW() - INTERVAL '30 days';
    
    -- Clean up old resolved errors (older than 30 days)
    DELETE FROM workflow_errors 
    WHERE status IN ('resolved', 'failed') 
    AND resolved_at < NOW() - INTERVAL '30 days';
    
    -- Clean up old fired timers (older than 30 days)
    DELETE FROM workflow_timers 
    WHERE status IN ('fired', 'cancelled') 
    AND fired_at < NOW() - INTERVAL '30 days';
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust as needed for your setup)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_escalations TO workflow_user;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_gateways TO workflow_user;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON workflow_errors TO workflow_user;
-- GRANT SELECT ON active_escalations TO workflow_user;
-- GRANT SELECT ON active_gateways TO workflow_user;
-- GRANT SELECT ON active_errors TO workflow_user;

-- Add comments for documentation
COMMENT ON TABLE workflow_escalations IS 'Stores escalation records for overdue tasks and workflow issues';
COMMENT ON TABLE workflow_gateways IS 'Stores gateway states for parallel, inclusive, and event-based gateways';
COMMENT ON TABLE workflow_errors IS 'Stores error records and recovery attempts for workflow failures';

COMMENT ON COLUMN workflow_escalations.escalation_level IS 'Level of escalation (1=first level, 2=second level, etc.)';
COMMENT ON COLUMN workflow_escalations.escalation_type IS 'Type of escalation (task_overdue, workflow_timeout, manual)';
COMMENT ON COLUMN workflow_escalations.escalation_target IS 'Target for escalation (user ID, role, or group)';

COMMENT ON COLUMN workflow_gateways.gateway_type IS 'Type of gateway (parallel, inclusive, event)';
COMMENT ON COLUMN workflow_gateways.required_tokens IS 'JSON array of required token names';
COMMENT ON COLUMN workflow_gateways.received_tokens IS 'JSON array of received token names';

COMMENT ON COLUMN workflow_errors.error_type IS 'Type of error (task_failure, service_unavailable, timeout, etc.)';
COMMENT ON COLUMN workflow_errors.recovery_strategy IS 'Recovery strategy (retry, compensate, escalate, skip, abort)';
COMMENT ON COLUMN workflow_errors.retry_count IS 'Number of retry attempts made';

-- Migration complete
SELECT 'Phase 5 Advanced Features migration completed successfully' as result;
