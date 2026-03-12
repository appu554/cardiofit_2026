-- Enhanced PostgreSQL Setup Script for KB Services
-- This script sets up multiple databases and users for the enhanced architecture

-- Enable necessary extensions at the template level
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- TimescaleDB will be enabled per database as needed

-- ==================== KB DRUG RULES DATABASE ====================

-- Create KB Drug Rules user and database
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'kb_drug_rules_user') THEN
      CREATE ROLE kb_drug_rules_user LOGIN PASSWORD 'kb_password';
   END IF;
END
$do$;

-- Create database if it doesn't exist
SELECT 'CREATE DATABASE kb_drug_rules OWNER kb_drug_rules_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'kb_drug_rules')\gexec

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE kb_drug_rules TO kb_drug_rules_user;

-- ==================== KB CLINICAL PATHWAYS DATABASE ====================

-- Create KB Clinical Pathways user and database
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'kb_clinical_pathways_user') THEN
      CREATE ROLE kb_clinical_pathways_user LOGIN PASSWORD 'kb_password';
   END IF;
END
$do$;

-- Create database if it doesn't exist
SELECT 'CREATE DATABASE kb_clinical_pathways OWNER kb_clinical_pathways_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'kb_clinical_pathways')\gexec

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE kb_clinical_pathways TO kb_clinical_pathways_user;

-- ==================== SHARED ANALYTICS DATABASE ====================

-- Create analytics user for cross-KB analytics
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'kb_analytics_user') THEN
      CREATE ROLE kb_analytics_user LOGIN PASSWORD 'kb_analytics_password';
   END IF;
END
$do$;

-- Create analytics database if it doesn't exist
SELECT 'CREATE DATABASE kb_analytics OWNER kb_analytics_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'kb_analytics')\gexec

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE kb_analytics TO kb_analytics_user;

-- Grant read access to KB databases for analytics
GRANT CONNECT ON DATABASE kb_drug_rules TO kb_analytics_user;
GRANT CONNECT ON DATABASE kb_clinical_pathways TO kb_analytics_user;

-- ==================== ML MODELS DATABASE ====================

-- Create MLFlow user and database for ML model registry
DO
$do$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'mlflow_user') THEN
      CREATE ROLE mlflow_user LOGIN PASSWORD 'mlflow_password';
   END IF;
END
$do$;

-- Create MLFlow database if it doesn't exist
SELECT 'CREATE DATABASE mlflow OWNER mlflow_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'mlflow')\gexec

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE mlflow TO mlflow_user;

-- ==================== SETUP INDIVIDUAL DATABASES ====================

-- Connect to kb_drug_rules database and set up extensions
\c kb_drug_rules;

-- Enable extensions for kb_drug_rules
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO kb_drug_rules_user;
GRANT ALL ON ALL TABLES IN SCHEMA public TO kb_drug_rules_user;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO kb_drug_rules_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO kb_drug_rules_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO kb_drug_rules_user;

-- Grant analytics user read access to kb_drug_rules tables
GRANT USAGE ON SCHEMA public TO kb_analytics_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb_analytics_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO kb_analytics_user;

-- Connect to kb_clinical_pathways database and set up extensions
\c kb_clinical_pathways;

-- Enable extensions for kb_clinical_pathways
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "timescaledb";

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO kb_clinical_pathways_user;
GRANT ALL ON ALL TABLES IN SCHEMA public TO kb_clinical_pathways_user;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO kb_clinical_pathways_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO kb_clinical_pathways_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO kb_clinical_pathways_user;

-- Grant analytics user read access to kb_clinical_pathways tables
GRANT USAGE ON SCHEMA public TO kb_analytics_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb_analytics_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO kb_analytics_user;

-- Connect to kb_analytics database and set up extensions
\c kb_analytics;

-- Enable extensions for analytics database
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "timescaledb";
CREATE EXTENSION IF NOT EXISTS "postgres_fdw";

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO kb_analytics_user;
GRANT ALL ON ALL TABLES IN SCHEMA public TO kb_analytics_user;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO kb_analytics_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO kb_analytics_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO kb_analytics_user;

-- Set up foreign data wrappers for cross-database analytics
CREATE SERVER IF NOT EXISTS kb_drug_rules_server
    FOREIGN DATA WRAPPER postgres_fdw
    OPTIONS (host 'localhost', port '5432', dbname 'kb_drug_rules');

CREATE SERVER IF NOT EXISTS kb_clinical_pathways_server
    FOREIGN DATA WRAPPER postgres_fdw
    OPTIONS (host 'localhost', port '5432', dbname 'kb_clinical_pathways');

-- Create user mappings for foreign data wrappers
CREATE USER MAPPING IF NOT EXISTS FOR kb_analytics_user
    SERVER kb_drug_rules_server
    OPTIONS (user 'kb_drug_rules_user', password 'kb_password');

CREATE USER MAPPING IF NOT EXISTS FOR kb_analytics_user
    SERVER kb_clinical_pathways_server
    OPTIONS (user 'kb_clinical_pathways_user', password 'kb_password');

-- Connect to mlflow database and set up extensions
\c mlflow;

-- Enable extensions for mlflow database
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO mlflow_user;
GRANT ALL ON ALL TABLES IN SCHEMA public TO mlflow_user;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO mlflow_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO mlflow_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO mlflow_user;

-- Return to default database
\c postgres;

-- ==================== CREATE SHARED VIEWS FOR MONITORING ====================

-- Create a monitoring schema for cross-database views
CREATE SCHEMA IF NOT EXISTS monitoring;

-- Grant access to monitoring schema
GRANT ALL ON SCHEMA monitoring TO postgres;
GRANT USAGE ON SCHEMA monitoring TO kb_analytics_user;

-- Create a function to get database sizes
CREATE OR REPLACE FUNCTION monitoring.get_database_sizes()
RETURNS TABLE(database_name text, size_mb numeric) AS $$
BEGIN
    RETURN QUERY
    SELECT datname::text, 
           round((pg_database_size(datname) / 1024.0 / 1024.0)::numeric, 2) as size_mb
    FROM pg_database
    WHERE datname IN ('kb_drug_rules', 'kb_clinical_pathways', 'kb_analytics', 'mlflow')
    ORDER BY pg_database_size(datname) DESC;
END;
$$ LANGUAGE plpgsql;

-- Grant execute permission on monitoring functions
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA monitoring TO kb_analytics_user;

-- ==================== SETUP BASIC HEALTH CHECK FUNCTIONS ====================

-- Create health check function for each service
CREATE OR REPLACE FUNCTION public.kb_health_check()
RETURNS json AS $$
BEGIN
    RETURN json_build_object(
        'status', 'healthy',
        'timestamp', NOW(),
        'databases', json_build_object(
            'kb_drug_rules', CASE WHEN EXISTS(SELECT 1 FROM pg_database WHERE datname = 'kb_drug_rules') THEN 'available' ELSE 'missing' END,
            'kb_clinical_pathways', CASE WHEN EXISTS(SELECT 1 FROM pg_database WHERE datname = 'kb_clinical_pathways') THEN 'available' ELSE 'missing' END,
            'kb_analytics', CASE WHEN EXISTS(SELECT 1 FROM pg_database WHERE datname = 'kb_analytics') THEN 'available' ELSE 'missing' END,
            'mlflow', CASE WHEN EXISTS(SELECT 1 FROM pg_database WHERE datname = 'mlflow') THEN 'available' ELSE 'missing' END
        ),
        'extensions', json_build_object(
            'uuid-ossp', CASE WHEN EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp') THEN 'loaded' ELSE 'missing' END,
            'pgcrypto', CASE WHEN EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'pgcrypto') THEN 'loaded' ELSE 'missing' END,
            'timescaledb', CASE WHEN EXISTS(SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN 'loaded' ELSE 'missing' END
        )
    );
END;
$$ LANGUAGE plpgsql;

-- Grant execute permission on health check
GRANT EXECUTE ON FUNCTION public.kb_health_check() TO PUBLIC;

-- Create log table for setup tracking
CREATE TABLE IF NOT EXISTS setup_log (
    log_id SERIAL PRIMARY KEY,
    setup_timestamp TIMESTAMPTZ DEFAULT NOW(),
    setup_component VARCHAR(100) NOT NULL,
    setup_status VARCHAR(20) NOT NULL,
    setup_details JSONB,
    setup_duration_seconds INTEGER
);

-- Insert setup completion log
INSERT INTO setup_log (setup_component, setup_status, setup_details)
VALUES ('enhanced_postgres_setup', 'completed', json_build_object(
    'databases_created', ARRAY['kb_drug_rules', 'kb_clinical_pathways', 'kb_analytics', 'mlflow'],
    'users_created', ARRAY['kb_drug_rules_user', 'kb_clinical_pathways_user', 'kb_analytics_user', 'mlflow_user'],
    'extensions_enabled', ARRAY['uuid-ossp', 'pgcrypto', 'timescaledb', 'postgres_fdw'],
    'setup_version', '1.0'
));

-- Display setup summary
SELECT 'Enhanced PostgreSQL Setup Complete!' as status,
       COUNT(*) FILTER (WHERE datname LIKE 'kb_%' OR datname = 'mlflow') as databases_created
FROM pg_database;