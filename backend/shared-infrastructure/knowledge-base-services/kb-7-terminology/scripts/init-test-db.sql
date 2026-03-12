-- KB-7 Terminology Service - Test Database Initialization
-- Creates test database with required extensions and minimal test data

-- Enable required PostgreSQL extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create test user with appropriate permissions
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'kb_test_user') THEN
      CREATE ROLE kb_test_user LOGIN PASSWORD 'kb_test_password';
   END IF;
END
$do$;

-- Grant all privileges on the test database
GRANT ALL PRIVILEGES ON DATABASE clinical_governance_test TO kb_test_user;
GRANT ALL ON SCHEMA public TO kb_test_user;

-- Set default search path
ALTER DATABASE clinical_governance_test SET search_path TO public;

-- Create test-specific configurations
CREATE TABLE IF NOT EXISTS test_configurations (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert test configuration values
INSERT INTO test_configurations (config_key, config_value, description) VALUES
('test_environment', 'docker', 'Indicates this is a Docker test environment'),
('test_dataset_size', 'minimal', 'Test uses minimal dataset for fast execution'),
('test_started_at', NOW()::TEXT, 'Timestamp when test environment was initialized'),
('postgres_version', version(), 'PostgreSQL version information');

-- Create test data tracking table
CREATE TABLE IF NOT EXISTS test_data_status (
    id SERIAL PRIMARY KEY,
    system_name VARCHAR(50) UNIQUE NOT NULL,
    loaded BOOLEAN DEFAULT FALSE,
    concept_count INTEGER DEFAULT 0,
    load_started_at TIMESTAMPTZ,
    load_completed_at TIMESTAMPTZ,
    load_duration_seconds INTEGER
);

-- Pre-populate with expected test systems
INSERT INTO test_data_status (system_name, loaded) VALUES
('snomed', FALSE),
('rxnorm', FALSE),
('loinc', FALSE)
ON CONFLICT (system_name) DO NOTHING;

-- Function to track test data loading
CREATE OR REPLACE FUNCTION update_test_data_status(
    p_system_name VARCHAR(50),
    p_concept_count INTEGER DEFAULT 0,
    p_completed BOOLEAN DEFAULT FALSE
)
RETURNS VOID AS $$
BEGIN
    IF p_completed THEN
        UPDATE test_data_status 
        SET loaded = TRUE,
            concept_count = p_concept_count,
            load_completed_at = NOW(),
            load_duration_seconds = EXTRACT(EPOCH FROM (NOW() - load_started_at))::INTEGER
        WHERE system_name = p_system_name;
    ELSE
        UPDATE test_data_status 
        SET load_started_at = NOW()
        WHERE system_name = p_system_name;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to get test environment status
CREATE OR REPLACE FUNCTION get_test_status()
RETURNS JSON AS $$
DECLARE
    result JSON;
BEGIN
    SELECT json_build_object(
        'database', 'ready',
        'extensions', json_build_array('uuid-ossp', 'pg_trgm'),
        'systems', (
            SELECT json_agg(
                json_build_object(
                    'name', system_name,
                    'loaded', loaded,
                    'concept_count', concept_count,
                    'load_duration', load_duration_seconds
                )
            )
            FROM test_data_status
        ),
        'initialized_at', (
            SELECT config_value FROM test_configurations 
            WHERE config_key = 'test_started_at'
        )
    ) INTO result;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Create indexes for better test performance
CREATE INDEX IF NOT EXISTS idx_test_data_status_system ON test_data_status(system_name);
CREATE INDEX IF NOT EXISTS idx_test_configurations_key ON test_configurations(config_key);

-- Log initialization completion
INSERT INTO test_configurations (config_key, config_value, description) VALUES
('db_initialization_completed', NOW()::TEXT, 'Database initialization completed successfully')
ON CONFLICT (config_key) DO UPDATE SET 
    config_value = EXCLUDED.config_value;

-- Display initialization summary
DO $$
BEGIN
    RAISE NOTICE 'KB-7 Terminology Test Database Initialized Successfully';
    RAISE NOTICE 'Database: clinical_governance_test';
    RAISE NOTICE 'User: kb_test_user';
    RAISE NOTICE 'Extensions: uuid-ossp, pg_trgm';
    RAISE NOTICE 'Test tracking tables created';
    RAISE NOTICE 'Ready for terminology service testing';
END $$;