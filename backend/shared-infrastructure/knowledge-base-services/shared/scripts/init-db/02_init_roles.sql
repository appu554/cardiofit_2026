-- =============================================================================
-- INITIALIZATION SCRIPT 02: Database Roles and Permissions
-- Purpose: Create application roles with principle of least privilege
-- Runs: First time container starts (via docker-entrypoint-initdb.d)
-- =============================================================================

-- =============================================================================
-- ROLE HIERARCHY
-- Following security best practices for clinical data:
--   - kb_readonly:    Read-only access for query services
--   - kb_writer:      Write access for ingestion services
--   - kb_admin:       Full access for administration (created by docker-compose)
--   - kb_auditor:     Access to audit logs only
-- =============================================================================

-- Read-only role for KB query services
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_readonly') THEN
        CREATE ROLE kb_readonly;
    END IF;
END
$$;

COMMENT ON ROLE kb_readonly IS 'Read-only access for KB query services';

-- Grant read access to public schema
GRANT USAGE ON SCHEMA public TO kb_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO kb_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO kb_readonly;

-- Grant read access to cache schema (for materialized views)
GRANT USAGE ON SCHEMA cache TO kb_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA cache TO kb_readonly;
ALTER DEFAULT PRIVILEGES IN SCHEMA cache GRANT SELECT ON TABLES TO kb_readonly;

-- =============================================================================
-- Writer role for ingestion services
-- =============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_writer') THEN
        CREATE ROLE kb_writer;
    END IF;
END
$$;

COMMENT ON ROLE kb_writer IS 'Write access for KB ingestion services';

-- Writer inherits read access
GRANT kb_readonly TO kb_writer;

-- Grant write access to public schema
GRANT INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO kb_writer;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kb_writer;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT INSERT, UPDATE, DELETE ON TABLES TO kb_writer;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO kb_writer;

-- Grant full access to staging schema
GRANT USAGE ON SCHEMA staging TO kb_writer;
GRANT ALL ON ALL TABLES IN SCHEMA staging TO kb_writer;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA staging TO kb_writer;
ALTER DEFAULT PRIVILEGES IN SCHEMA staging GRANT ALL ON TABLES TO kb_writer;
ALTER DEFAULT PRIVILEGES IN SCHEMA staging GRANT USAGE, SELECT ON SEQUENCES TO kb_writer;

-- Grant insert to audit schema (for audit logging)
GRANT USAGE ON SCHEMA audit TO kb_writer;
GRANT INSERT ON ALL TABLES IN SCHEMA audit TO kb_writer;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA audit TO kb_writer;
ALTER DEFAULT PRIVILEGES IN SCHEMA audit GRANT INSERT ON TABLES TO kb_writer;
ALTER DEFAULT PRIVILEGES IN SCHEMA audit GRANT USAGE, SELECT ON SEQUENCES TO kb_writer;

-- =============================================================================
-- Auditor role for compliance queries
-- =============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_auditor') THEN
        CREATE ROLE kb_auditor;
    END IF;
END
$$;

COMMENT ON ROLE kb_auditor IS 'Read access to audit logs for compliance';

-- Grant read access to audit schema only
GRANT USAGE ON SCHEMA audit TO kb_auditor;
GRANT SELECT ON ALL TABLES IN SCHEMA audit TO kb_auditor;
ALTER DEFAULT PRIVILEGES IN SCHEMA audit GRANT SELECT ON TABLES TO kb_auditor;

-- =============================================================================
-- Application User Creation
-- These users will be used by the actual services
-- =============================================================================

-- Create read-only service user
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_query_svc') THEN
        CREATE USER kb_query_svc WITH PASSWORD 'kb_query_svc_2024';
    END IF;
END
$$;
GRANT kb_readonly TO kb_query_svc;
COMMENT ON ROLE kb_query_svc IS 'Service account for KB query services';

-- Create writer service user
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_ingest_svc') THEN
        CREATE USER kb_ingest_svc WITH PASSWORD 'kb_ingest_svc_2024';
    END IF;
END
$$;
GRANT kb_writer TO kb_ingest_svc;
COMMENT ON ROLE kb_ingest_svc IS 'Service account for KB ingestion services';

-- Create auditor service user
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'kb_audit_svc') THEN
        CREATE USER kb_audit_svc WITH PASSWORD 'kb_audit_svc_2024';
    END IF;
END
$$;
GRANT kb_auditor TO kb_audit_svc;
COMMENT ON ROLE kb_audit_svc IS 'Service account for audit and compliance queries';

-- =============================================================================
-- Verify roles
-- =============================================================================

SELECT 'Roles initialized:' AS status;
SELECT rolname, rolsuper, rolcreatedb, rolcanlogin
FROM pg_roles
WHERE rolname LIKE 'kb_%'
ORDER BY rolname;
