-- =============================================================================
-- INITIALIZATION SCRIPT 00: PostgreSQL Extensions
-- Purpose: Enable required PostgreSQL extensions for clinical data operations
-- Runs: First time container starts (via docker-entrypoint-initdb.d)
-- =============================================================================

-- Enable UUID generation (for fact_id, review_id, etc.)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable cryptographic functions (for checksums, hashing)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enable query statistics tracking (for performance monitoring)
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Enable trigram matching (for fuzzy drug name search)
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Enable full-text search enhancements
CREATE EXTENSION IF NOT EXISTS "unaccent";

-- Verify extensions
SELECT 'Extensions initialized:' AS status;
SELECT extname, extversion FROM pg_extension WHERE extname IN (
    'uuid-ossp', 'pgcrypto', 'pg_stat_statements', 'pg_trgm', 'unaccent'
);
