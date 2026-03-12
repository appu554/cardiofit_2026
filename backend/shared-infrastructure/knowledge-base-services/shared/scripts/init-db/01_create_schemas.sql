-- =============================================================================
-- INITIALIZATION SCRIPT 01: Database Schemas
-- Purpose: Create logical schema separation for clinical data organization
-- Runs: First time container starts (via docker-entrypoint-initdb.d)
-- =============================================================================

-- =============================================================================
-- SCHEMA ORGANIZATION
-- Following the KB1 Implementation Plan's layered architecture:
--   - public:       Core tables (drug_master, clinical_facts)
--   - kb_views:     Read-only KB projections (kb1_*, kb4_*, etc.)
--   - staging:      Data ingestion staging tables
--   - audit:        Audit trails and compliance logging
--   - cache:        Materialized views for hot path optimization
-- =============================================================================

-- Create staging schema for data ingestion
CREATE SCHEMA IF NOT EXISTS staging;
COMMENT ON SCHEMA staging IS 'Staging area for data ingestion pipelines';

-- Create audit schema for compliance
CREATE SCHEMA IF NOT EXISTS audit;
COMMENT ON SCHEMA audit IS 'Audit trails and compliance logging';

-- Create cache schema for performance optimization
CREATE SCHEMA IF NOT EXISTS cache;
COMMENT ON SCHEMA cache IS 'Materialized views and cache tables';

-- =============================================================================
-- AUDIT INFRASTRUCTURE
-- =============================================================================

-- Audit log for all clinical fact changes
CREATE TABLE IF NOT EXISTS audit.fact_audit_log (
    audit_id            BIGSERIAL PRIMARY KEY,
    fact_id             UUID NOT NULL,
    operation           VARCHAR(10) NOT NULL,  -- INSERT, UPDATE, DELETE
    old_values          JSONB,
    new_values          JSONB,
    changed_by          VARCHAR(255) NOT NULL DEFAULT current_user,
    changed_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    client_ip           INET,
    session_id          VARCHAR(100)
);

CREATE INDEX idx_audit_fact_id ON audit.fact_audit_log(fact_id);
CREATE INDEX idx_audit_changed_at ON audit.fact_audit_log(changed_at DESC);
CREATE INDEX idx_audit_operation ON audit.fact_audit_log(operation);

-- API access log for compliance
CREATE TABLE IF NOT EXISTS audit.api_access_log (
    log_id              BIGSERIAL PRIMARY KEY,
    endpoint            VARCHAR(255) NOT NULL,
    method              VARCHAR(10) NOT NULL,
    request_params      JSONB,
    response_status     INTEGER,
    response_time_ms    INTEGER,
    user_id             VARCHAR(255),
    client_ip           INET,
    user_agent          TEXT,
    accessed_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_log_accessed_at ON audit.api_access_log(accessed_at DESC);
CREATE INDEX idx_api_log_endpoint ON audit.api_access_log(endpoint);
CREATE INDEX idx_api_log_user ON audit.api_access_log(user_id);

-- =============================================================================
-- STAGING TABLES
-- =============================================================================

-- Generic staging table for CSV/file imports
CREATE TABLE IF NOT EXISTS staging.import_queue (
    import_id           SERIAL PRIMARY KEY,
    source_name         VARCHAR(100) NOT NULL,
    source_file         VARCHAR(500),
    raw_data            JSONB NOT NULL,
    row_number          INTEGER,
    validation_status   VARCHAR(20) DEFAULT 'PENDING',
    validation_errors   JSONB,
    processed_at        TIMESTAMP WITH TIME ZONE,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_staging_source ON staging.import_queue(source_name, validation_status);
CREATE INDEX idx_staging_pending ON staging.import_queue(created_at)
    WHERE validation_status = 'PENDING';

-- =============================================================================
-- SCHEMA MIGRATIONS TRACKING
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.schema_migrations (
    version             INTEGER PRIMARY KEY,
    name                VARCHAR(255) NOT NULL,
    applied_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Record this initialization
INSERT INTO public.schema_migrations (version, name)
VALUES (0, '01_create_schemas')
ON CONFLICT (version) DO NOTHING;

-- Verify schemas
SELECT 'Schemas initialized:' AS status;
SELECT schema_name FROM information_schema.schemata
WHERE schema_name IN ('public', 'staging', 'audit', 'cache')
ORDER BY schema_name;
