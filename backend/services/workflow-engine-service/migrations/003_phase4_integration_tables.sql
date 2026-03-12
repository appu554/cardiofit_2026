-- Migration 003: Phase 4 Service Integration Tables
-- This migration creates tables for Phase 4 service integration features

-- Service Task Execution Logs
CREATE TABLE IF NOT EXISTS service_task_logs (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    operation VARCHAR(100) NOT NULL,
    parameters JSONB,
    result JSONB,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source VARCHAR(100) DEFAULT 'service-task-executor',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Event Store for inter-service communication
CREATE TABLE IF NOT EXISTS event_store (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    event_type VARCHAR(200) NOT NULL,
    event_data JSONB NOT NULL,
    source VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE,
    processing_status VARCHAR(50) DEFAULT 'pending'
);

-- Event Processing Logs
CREATE TABLE IF NOT EXISTS event_processing_logs (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    event_id UUID,
    event_type VARCHAR(200) NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    source VARCHAR(100) DEFAULT 'event-listener',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- FHIR Resource Monitoring State
CREATE TABLE IF NOT EXISTS fhir_resource_monitor_state (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(200) NOT NULL,
    last_modified TIMESTAMP WITH TIME ZONE,
    last_checked TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    monitoring_status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(resource_type, resource_id)
);

-- Service Integration Configuration
CREATE TABLE IF NOT EXISTS service_integration_config (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL UNIQUE,
    endpoint_url VARCHAR(500) NOT NULL,
    webhook_url VARCHAR(500),
    enabled BOOLEAN DEFAULT true,
    timeout_seconds INTEGER DEFAULT 30,
    retry_attempts INTEGER DEFAULT 3,
    configuration JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Workflow Event Triggers
CREATE TABLE IF NOT EXISTS workflow_event_triggers (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    event_type VARCHAR(200) NOT NULL,
    workflow_definition_id VARCHAR(200) NOT NULL,
    condition_expression TEXT,
    variable_mapping JSONB,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_service_task_logs_service_name ON service_task_logs(service_name);
CREATE INDEX IF NOT EXISTS idx_service_task_logs_status ON service_task_logs(status);
CREATE INDEX IF NOT EXISTS idx_service_task_logs_executed_at ON service_task_logs(executed_at);

CREATE INDEX IF NOT EXISTS idx_event_store_event_type ON event_store(event_type);
CREATE INDEX IF NOT EXISTS idx_event_store_created_at ON event_store(created_at);
CREATE INDEX IF NOT EXISTS idx_event_store_processing_status ON event_store(processing_status);

CREATE INDEX IF NOT EXISTS idx_event_processing_logs_event_type ON event_processing_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_event_processing_logs_status ON event_processing_logs(status);
CREATE INDEX IF NOT EXISTS idx_event_processing_logs_processed_at ON event_processing_logs(processed_at);

CREATE INDEX IF NOT EXISTS idx_fhir_monitor_resource_type ON fhir_resource_monitor_state(resource_type);
CREATE INDEX IF NOT EXISTS idx_fhir_monitor_last_checked ON fhir_resource_monitor_state(last_checked);
CREATE INDEX IF NOT EXISTS idx_fhir_monitor_status ON fhir_resource_monitor_state(monitoring_status);

CREATE INDEX IF NOT EXISTS idx_workflow_event_triggers_event_type ON workflow_event_triggers(event_type);
CREATE INDEX IF NOT EXISTS idx_workflow_event_triggers_enabled ON workflow_event_triggers(enabled);

-- Insert default service integration configurations
INSERT INTO service_integration_config (service_name, endpoint_url, webhook_url, configuration) VALUES
('patient-service', 'http://localhost:8003', 'http://localhost:8003/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}'),
('observation-service', 'http://localhost:8007', 'http://localhost:8007/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}'),
('medication-service', 'http://localhost:8009', 'http://localhost:8009/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}'),
('condition-service', 'http://localhost:8010', 'http://localhost:8010/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}'),
('encounter-service', 'http://localhost:8020', 'http://localhost:8020/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}'),
('organization-service', 'http://localhost:8012', 'http://localhost:8012/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}'),
('order-service', 'http://localhost:8013', 'http://localhost:8013/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}'),
('scheduling-service', 'http://localhost:8014', 'http://localhost:8014/api/webhooks/workflow-events', '{"timeout": 30, "retry_attempts": 3}')
ON CONFLICT (service_name) DO NOTHING;

-- Insert default workflow event triggers
INSERT INTO workflow_event_triggers (event_type, workflow_definition_id, variable_mapping, enabled) VALUES
('patient.admitted', 'patient-admission-workflow', '{"patient_id": "$.patient_id", "encounter_id": "$.encounter_id"}', true),
('order.created', 'order-fulfillment-workflow', '{"order_id": "$.order_id", "patient_id": "$.patient_id"}', true),
('appointment.scheduled', 'appointment-preparation-workflow', '{"appointment_id": "$.appointment_id", "patient_id": "$.patient_id"}', true),
('encounter.created', 'encounter-workflow', '{"encounter_id": "$.encounter_id", "patient_id": "$.patient_id"}', true)
ON CONFLICT DO NOTHING;

-- Add RLS (Row Level Security) policies if needed
-- Note: These are basic policies - adjust based on your security requirements

-- Enable RLS on tables
ALTER TABLE service_task_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_store ENABLE ROW LEVEL SECURITY;
ALTER TABLE event_processing_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE fhir_resource_monitor_state ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_integration_config ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_event_triggers ENABLE ROW LEVEL SECURITY;

-- Create policies for authenticated users
CREATE POLICY "Allow authenticated users to read service_task_logs" ON service_task_logs
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to insert service_task_logs" ON service_task_logs
    FOR INSERT WITH CHECK (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to read event_store" ON event_store
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to insert event_store" ON event_store
    FOR INSERT WITH CHECK (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to update event_store" ON event_store
    FOR UPDATE USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to read event_processing_logs" ON event_processing_logs
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to insert event_processing_logs" ON event_processing_logs
    FOR INSERT WITH CHECK (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to read fhir_resource_monitor_state" ON fhir_resource_monitor_state
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to insert fhir_resource_monitor_state" ON fhir_resource_monitor_state
    FOR INSERT WITH CHECK (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to update fhir_resource_monitor_state" ON fhir_resource_monitor_state
    FOR UPDATE USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to read service_integration_config" ON service_integration_config
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to update service_integration_config" ON service_integration_config
    FOR UPDATE USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to read workflow_event_triggers" ON workflow_event_triggers
    FOR SELECT USING (auth.role() = 'authenticated');

CREATE POLICY "Allow authenticated users to update workflow_event_triggers" ON workflow_event_triggers
    FOR UPDATE USING (auth.role() = 'authenticated');

-- Create a function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_service_integration_config_updated_at 
    BEFORE UPDATE ON service_integration_config 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_event_triggers_updated_at 
    BEFORE UPDATE ON workflow_event_triggers 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE service_task_logs IS 'Logs of service task executions for monitoring and debugging';
COMMENT ON TABLE event_store IS 'Central event store for inter-service communication';
COMMENT ON TABLE event_processing_logs IS 'Logs of event processing activities';
COMMENT ON TABLE fhir_resource_monitor_state IS 'State tracking for FHIR resource monitoring';
COMMENT ON TABLE service_integration_config IS 'Configuration for service integrations';
COMMENT ON TABLE workflow_event_triggers IS 'Configuration for event-triggered workflows';

-- Migration completed
SELECT 'Phase 4 Integration Tables Migration Completed' AS status;
