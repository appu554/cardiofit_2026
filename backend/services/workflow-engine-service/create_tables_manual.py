#!/usr/bin/env python3
"""
Generate SQL for manual table creation in Supabase dashboard.
"""

def generate_sql():
    """Generate the SQL for creating workflow tables."""
    
    sql = """
-- Workflow Engine Service Tables
-- Run this SQL in your Supabase SQL Editor

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

-- Create workflow_events table
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

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_workflow_definitions_fhir_id ON workflow_definitions(fhir_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instances_patient_id ON workflow_instances(patient_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_fhir_id ON workflow_tasks(fhir_id);
CREATE INDEX IF NOT EXISTS idx_workflow_tasks_assignee ON workflow_tasks(assignee);
CREATE INDEX IF NOT EXISTS idx_workflow_events_timestamp ON workflow_events(timestamp);

-- Create a simple test table to verify everything works
CREATE TABLE IF NOT EXISTS workflow_test (
    id SERIAL PRIMARY KEY,
    test_message VARCHAR(255) DEFAULT 'Workflow Engine Service tables created successfully!',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert a test record
INSERT INTO workflow_test (test_message) VALUES ('Tables created at ' || NOW());

-- Grant permissions (if needed)
-- ALTER TABLE workflow_definitions ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE workflow_instances ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE workflow_tasks ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE workflow_events ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE workflow_timers ENABLE ROW LEVEL SECURITY;

SELECT 'Workflow Engine Service tables created successfully!' as result;
"""
    
    return sql

def main():
    """Main function to display instructions."""
    print("=" * 80)
    print("WORKFLOW ENGINE SERVICE - MANUAL TABLE CREATION")
    print("=" * 80)
    
    print("\n📋 INSTRUCTIONS:")
    print("1. Go to your Supabase dashboard: https://supabase.com/dashboard")
    print("2. Select your project: auugxeqzgrnknklgwqrh")
    print("3. Go to 'SQL Editor' in the left sidebar")
    print("4. Click 'New query'")
    print("5. Copy and paste the SQL below")
    print("6. Click 'Run' to execute the SQL")
    print("7. You should see 'Workflow Engine Service tables created successfully!'")
    
    print("\n" + "=" * 80)
    print("SQL TO RUN IN SUPABASE SQL EDITOR:")
    print("=" * 80)
    
    sql = generate_sql()
    print(sql)
    
    print("=" * 80)
    print("After running the SQL, come back and run:")
    print("python verify_tables.py")
    print("=" * 80)

if __name__ == "__main__":
    main()
