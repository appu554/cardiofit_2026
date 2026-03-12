-- Migration script to create workflow engine service tables in Supabase
-- Run this in the Supabase SQL Editor

-- Create service_task_logs table
CREATE TABLE IF NOT EXISTS public.service_task_logs (
    id BIGSERIAL PRIMARY KEY,
    service_name VARCHAR(255) NOT NULL,
    operation VARCHAR(255) NOT NULL,
    parameters TEXT,
    result TEXT,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create event_store table
CREATE TABLE IF NOT EXISTS public.event_store (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    event_data TEXT NOT NULL,
    source VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMPTZ,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Create event_processing_logs table
CREATE TABLE IF NOT EXISTS public.event_processing_logs (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT REFERENCES public.event_store(id),
    event_type VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    source VARCHAR(255) NOT NULL,
    processing_duration_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create workflow_events_log table
CREATE TABLE IF NOT EXISTS public.workflow_events_log (
    id BIGSERIAL PRIMARY KEY,
    workflow_instance_id VARCHAR(255),
    task_id VARCHAR(255),
    event_type VARCHAR(255) NOT NULL,
    event_data TEXT,
    user_id VARCHAR(255),
    source VARCHAR(255) NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_service_task_logs_service_name ON public.service_task_logs(service_name);
CREATE INDEX IF NOT EXISTS idx_service_task_logs_status ON public.service_task_logs(status);
CREATE INDEX IF NOT EXISTS idx_service_task_logs_executed_at ON public.service_task_logs(executed_at);

CREATE INDEX IF NOT EXISTS idx_event_store_event_type ON public.event_store(event_type);
CREATE INDEX IF NOT EXISTS idx_event_store_source ON public.event_store(source);
CREATE INDEX IF NOT EXISTS idx_event_store_created_at ON public.event_store(created_at);
CREATE INDEX IF NOT EXISTS idx_event_store_processed ON public.event_store(processed);

CREATE INDEX IF NOT EXISTS idx_event_processing_logs_event_type ON public.event_processing_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_event_processing_logs_status ON public.event_processing_logs(status);
CREATE INDEX IF NOT EXISTS idx_event_processing_logs_processed_at ON public.event_processing_logs(processed_at);

CREATE INDEX IF NOT EXISTS idx_workflow_events_log_workflow_instance_id ON public.workflow_events_log(workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_workflow_events_log_event_type ON public.workflow_events_log(event_type);
CREATE INDEX IF NOT EXISTS idx_workflow_events_log_timestamp ON public.workflow_events_log(timestamp);

-- Enable Row Level Security (RLS) for security
ALTER TABLE public.service_task_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.event_store ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.event_processing_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.workflow_events_log ENABLE ROW LEVEL SECURITY;

-- Create policies to allow service access (adjust as needed for your security requirements)
CREATE POLICY "Allow service access to service_task_logs" ON public.service_task_logs
    FOR ALL USING (true);

CREATE POLICY "Allow service access to event_store" ON public.event_store
    FOR ALL USING (true);

CREATE POLICY "Allow service access to event_processing_logs" ON public.event_processing_logs
    FOR ALL USING (true);

CREATE POLICY "Allow service access to workflow_events_log" ON public.workflow_events_log
    FOR ALL USING (true);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
CREATE TRIGGER update_service_task_logs_updated_at 
    BEFORE UPDATE ON public.service_task_logs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Grant necessary permissions to authenticated users
GRANT ALL ON public.service_task_logs TO authenticated;
GRANT ALL ON public.event_store TO authenticated;
GRANT ALL ON public.event_processing_logs TO authenticated;
GRANT ALL ON public.workflow_events_log TO authenticated;

-- Grant sequence permissions
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO authenticated;

-- Comment the tables for documentation
COMMENT ON TABLE public.service_task_logs IS 'Logs of service task executions from the workflow engine';
COMMENT ON TABLE public.event_store IS 'Event store for workflow and system events';
COMMENT ON TABLE public.event_processing_logs IS 'Logs of event processing activities';
COMMENT ON TABLE public.workflow_events_log IS 'Workflow-specific event logs for analytics and monitoring';
