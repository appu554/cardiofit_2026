-- KB-7 Terminology Service Enhanced Schema
-- Adds partitioning, performance optimizations, and missing tables per design documents

-- Create extensions needed for enhanced functionality
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";        -- For text similarity matching
-- Note: pg_stat_statements requires server-level config (shared_preload_libraries)
-- CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- fuzzystrmatch for phonetic matching (optional - graceful degradation if unavailable)
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS "fuzzystrmatch";
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'fuzzystrmatch extension not available - phonetic matching disabled';
END $$;

-- =====================================================
-- 1. Enhanced Concepts Table with Partitioning
-- =====================================================

-- First, create the partitioned concepts table
DROP TABLE IF EXISTS concepts_new CASCADE;
CREATE TABLE concepts_new (
    id BIGSERIAL,
    concept_uuid UUID DEFAULT gen_random_uuid(),
    system VARCHAR(20) NOT NULL,
    code VARCHAR(100) NOT NULL,
    version VARCHAR(20) NOT NULL,
    code_system_version_id UUID REFERENCES terminology_systems(id),
    
    -- Core attributes
    preferred_term TEXT NOT NULL,
    fully_specified_name TEXT,
    synonyms TEXT[] DEFAULT '{}',
    
    -- Hierarchy
    parent_codes TEXT[] DEFAULT '{}',
    is_leaf BOOLEAN DEFAULT FALSE,
    depth INTEGER DEFAULT 0,
    
    -- Search optimization
    search_vector tsvector,
    metaphone_key TEXT,
    soundex_key TEXT,
    
    -- Designations (multilingual support)
    designations JSONB DEFAULT '{}',
    
    -- Status and versioning
    active BOOLEAN DEFAULT TRUE,
    replaced_by VARCHAR(100),
    
    -- Metadata
    properties JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    PRIMARY KEY (id, system),
    UNIQUE (system, code, version)
) PARTITION BY LIST (system);

-- Create partitions per terminology system
CREATE TABLE concepts_rxnorm PARTITION OF concepts_new FOR VALUES IN ('RxNorm');
CREATE TABLE concepts_snomed PARTITION OF concepts_new FOR VALUES IN ('SNOMED');
CREATE TABLE concepts_loinc PARTITION OF concepts_new FOR VALUES IN ('LOINC');
CREATE TABLE concepts_icd10 PARTITION OF concepts_new FOR VALUES IN ('ICD10');
CREATE TABLE concepts_icd9 PARTITION OF concepts_new FOR VALUES IN ('ICD9');
CREATE TABLE concepts_cpt PARTITION OF concepts_new FOR VALUES IN ('CPT');
CREATE TABLE concepts_ndc PARTITION OF concepts_new FOR VALUES IN ('NDC');

-- Migrate data from old concepts table to new partitioned table
-- Note: Only migrate if there is data to migrate (skip if empty)
INSERT INTO concepts_new (
    concept_uuid, system, code, version, preferred_term, synonyms,
    parent_codes, active, properties, designations, created_at, updated_at
)
SELECT
    tc.id,
    CASE
        WHEN ts.system_uri LIKE '%rxnorm%' THEN 'RxNorm'
        WHEN ts.system_uri LIKE '%snomed%' THEN 'SNOMED'
        WHEN ts.system_uri LIKE '%loinc%' THEN 'LOINC'
        WHEN ts.system_uri LIKE '%icd-10%' THEN 'ICD10'
        WHEN ts.system_uri LIKE '%icd-9%' THEN 'ICD9'
        ELSE 'RxNorm'
    END as system,
    tc.code,
    COALESCE(ts.version, '2024-01') as version,
    tc.display as preferred_term,
    ARRAY[]::TEXT[] as synonyms,
    tc.parent_codes,
    tc.status = 'active' as active,
    tc.properties,
    tc.designations,
    tc.created_at,
    tc.updated_at
FROM terminology_concepts tc
LEFT JOIN terminology_systems ts ON tc.system_id = ts.id
WHERE EXISTS (SELECT 1 FROM terminology_concepts LIMIT 1);

-- Drop old table and rename new one
DROP TABLE IF EXISTS terminology_concepts CASCADE;
ALTER TABLE concepts_new RENAME TO concepts;

-- =====================================================
-- 2. Drug Concepts Table (Specialized for RxNorm)
-- =====================================================

CREATE TABLE drug_concepts (
    id BIGSERIAL PRIMARY KEY,
    rxnorm_cui VARCHAR(20) UNIQUE NOT NULL,
    
    -- Drug identification
    ingredient VARCHAR(500) NOT NULL,
    strength VARCHAR(100),
    dose_form VARCHAR(200),
    brand_names TEXT[] DEFAULT '{}',
    
    -- Classification
    atc_codes TEXT[] DEFAULT '{}',
    drug_class VARCHAR(200),
    schedule VARCHAR(10),
    
    -- Clinical attributes
    is_generic BOOLEAN,
    is_vaccine BOOLEAN DEFAULT FALSE,
    is_insulin BOOLEAN DEFAULT FALSE,
    is_controlled BOOLEAN DEFAULT FALSE,
    
    -- Relationships
    has_tradename TEXT[],
    consists_of JSONB, -- For multi-ingredient drugs
    
    -- Search optimization
    search_vector tsvector,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 3. Lab Reference Ranges (for LOINC)
-- =====================================================

CREATE TABLE lab_references (
    id BIGSERIAL PRIMARY KEY,
    loinc_code VARCHAR(20) NOT NULL,
    test_name TEXT NOT NULL,
    
    -- Reference ranges
    unit VARCHAR(50) NOT NULL,
    normal_low DECIMAL,
    normal_high DECIMAL,
    critical_low DECIMAL,
    critical_high DECIMAL,
    
    -- Population specifics
    age_low INTEGER,
    age_high INTEGER,
    sex CHAR(1) CHECK (sex IN ('M', 'F', 'U')),
    conditions JSONB DEFAULT '{}',
    
    -- Source information
    source VARCHAR(100) NOT NULL,
    effective_date DATE DEFAULT CURRENT_DATE,
    
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 4. Value Set Expansions with Time-based Partitioning
-- =====================================================

CREATE TABLE value_set_expansions (
    id BIGSERIAL,
    value_set_id UUID REFERENCES value_sets(id) ON DELETE CASCADE,
    params_hash TEXT NOT NULL,
    expansion_params JSONB NOT NULL,
    total INTEGER,
    offset_value INTEGER DEFAULT 0,
    generated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,

    PRIMARY KEY (id, generated_at),
    UNIQUE (value_set_id, params_hash, generated_at)
) PARTITION BY RANGE (generated_at);

-- Create monthly partitions for the next 12 months
CREATE TABLE expansions_2025_01 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE expansions_2025_02 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE expansions_2025_03 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE expansions_2025_04 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE expansions_2025_05 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE expansions_2025_06 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE expansions_2025_07 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE expansions_2025_08 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE expansions_2025_09 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE expansions_2025_10 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE expansions_2025_11 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE expansions_2025_12 PARTITION OF value_set_expansions 
  FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- Expansion contains table with hash partitioning for scale
CREATE TABLE expansion_contains (
    id BIGSERIAL,
    expansion_id BIGINT NOT NULL,
    system TEXT NOT NULL,
    code TEXT NOT NULL,
    display TEXT,
    designation JSONB DEFAULT '[]',
    inactive BOOLEAN DEFAULT FALSE,
    abstract BOOLEAN DEFAULT FALSE,
    
    PRIMARY KEY (id, expansion_id)
) PARTITION BY HASH (expansion_id);

-- Create 8 hash partitions for scaling
CREATE TABLE expansion_contains_p0 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 0);
CREATE TABLE expansion_contains_p1 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 1);
CREATE TABLE expansion_contains_p2 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 2);
CREATE TABLE expansion_contains_p3 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 3);
CREATE TABLE expansion_contains_p4 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 4);
CREATE TABLE expansion_contains_p5 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 5);
CREATE TABLE expansion_contains_p6 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 6);
CREATE TABLE expansion_contains_p7 PARTITION OF expansion_contains 
  FOR VALUES WITH (modulus 8, remainder 7);

-- =====================================================
-- 5. SNOMED Expression Support
-- =====================================================

CREATE TABLE snomed_expressions (
    id BIGSERIAL PRIMARY KEY,
    expression_hash TEXT UNIQUE NOT NULL,
    expression TEXT NOT NULL,
    normal_form TEXT NOT NULL,
    focus_concepts TEXT[] NOT NULL,
    refinements JSONB NOT NULL DEFAULT '{}',
    validation_status VARCHAR(20) DEFAULT 'pending' 
        CHECK (validation_status IN ('valid', 'invalid', 'pending')),
    validation_errors JSONB DEFAULT '[]',
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 6. Tenant Overlays for Multi-tenancy
-- =====================================================

CREATE TABLE tenant_overlays (
    id BIGSERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    priority INTEGER DEFAULT 100,
    overlay_type VARCHAR(20) NOT NULL 
        CHECK (overlay_type IN ('concept', 'valueset', 'map')),
    target_system TEXT NOT NULL,
    target_code TEXT,
    overlay_data JSONB NOT NULL,
    conflict_resolution VARCHAR(20) DEFAULT 'override'
        CHECK (conflict_resolution IN ('override', 'merge', 'skip')),
    active BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- 7. Audit and Provenance Tables
-- =====================================================

CREATE TABLE terminology_audit (
    id BIGSERIAL,
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    user_id TEXT,
    tenant_id TEXT,
    operation TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    parameters JSONB DEFAULT '{}',
    result_count INTEGER,
    duration_ms INTEGER,
    cache_hit BOOLEAN,
    license_check_passed BOOLEAN DEFAULT TRUE,

    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create indexes on the audit table
CREATE INDEX idx_audit_timestamp ON terminology_audit(timestamp);
CREATE INDEX idx_audit_user ON terminology_audit(user_id);
CREATE INDEX idx_audit_tenant ON terminology_audit(tenant_id);

-- Create monthly audit partitions
CREATE TABLE audit_2025_01 PARTITION OF terminology_audit 
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE audit_2025_02 PARTITION OF terminology_audit 
  FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE audit_2025_03 PARTITION OF terminology_audit 
  FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');

-- =====================================================
-- 8. Performance Indexes
-- =====================================================

-- Concepts indexes with proper names
CREATE INDEX idx_concepts_code ON concepts(code);
CREATE INDEX idx_concepts_system_code ON concepts(system, code);
CREATE INDEX idx_concepts_search ON concepts USING GIN(search_vector);
CREATE INDEX idx_concepts_metaphone ON concepts(metaphone_key);
CREATE INDEX idx_concepts_soundex ON concepts(soundex_key);
CREATE INDEX idx_concepts_parent ON concepts USING GIN(parent_codes);
CREATE INDEX idx_concepts_properties ON concepts USING GIN(properties);
CREATE INDEX idx_concepts_designations ON concepts USING GIN(designations);

-- Drug concepts indexes
CREATE INDEX idx_drug_rxnorm_cui ON drug_concepts(rxnorm_cui);
CREATE INDEX idx_drug_ingredient ON drug_concepts(ingredient);
CREATE INDEX idx_drug_search ON drug_concepts USING GIN(search_vector);
CREATE INDEX idx_drug_atc_codes ON drug_concepts USING GIN(atc_codes);
CREATE INDEX idx_drug_brand_names ON drug_concepts USING GIN(brand_names);

-- Lab references indexes
CREATE INDEX idx_lab_loinc ON lab_references(loinc_code);
CREATE INDEX idx_lab_test_name ON lab_references(test_name);
CREATE INDEX idx_lab_population ON lab_references(age_low, age_high, sex);

-- Value set expansions indexes
CREATE INDEX idx_expansion_lookup ON value_set_expansions(value_set_id, params_hash);
CREATE INDEX idx_expansion_expires ON value_set_expansions(expires_at) WHERE expires_at IS NOT NULL;

-- Tenant overlays indexes
CREATE INDEX idx_tenant_overlays_lookup ON tenant_overlays(tenant_id, target_system, target_code);
CREATE INDEX idx_tenant_overlays_priority ON tenant_overlays(tenant_id, priority DESC);

-- SNOMED expressions indexes
CREATE INDEX idx_snomed_expr_hash ON snomed_expressions(expression_hash);
CREATE INDEX idx_snomed_focus_concepts ON snomed_expressions USING GIN(focus_concepts);

-- =====================================================
-- 9. Materialized Views for Performance
-- =====================================================

-- Concept hierarchy view for fast traversal
CREATE MATERIALIZED VIEW concept_hierarchy AS
WITH RECURSIVE hierarchy AS (
    -- Root concepts (no parents)
    SELECT
        system, code, preferred_term, parent_codes,
        0 as level, ARRAY[code]::TEXT[] as path
    FROM concepts
    WHERE array_length(parent_codes, 1) IS NULL OR parent_codes = '{}'

    UNION ALL

    -- Child concepts
    SELECT
        c.system, c.code, c.preferred_term, c.parent_codes,
        h.level + 1, h.path || c.code::TEXT
    FROM concepts c
    JOIN hierarchy h ON c.parent_codes @> ARRAY[h.code]::TEXT[]
    WHERE h.level < 15  -- Prevent infinite recursion
)
SELECT * FROM hierarchy;

CREATE UNIQUE INDEX ON concept_hierarchy(system, code);
CREATE INDEX idx_concept_hierarchy_level ON concept_hierarchy(level);
CREATE INDEX idx_concept_hierarchy_path ON concept_hierarchy USING GIN(path);

-- =====================================================
-- 10. Update Functions and Triggers
-- =====================================================

-- Function to update search vectors with enhanced matching
CREATE OR REPLACE FUNCTION update_search_vector() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.preferred_term, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.fully_specified_name, '')), 'B') ||
        setweight(to_tsvector('english', array_to_string(NEW.synonyms, ' ')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.code, '')), 'C');

    -- Generate phonetic keys for fuzzy matching (graceful degradation if fuzzystrmatch unavailable)
    BEGIN
        NEW.metaphone_key := metaphone(COALESCE(NEW.preferred_term, ''), 8);
        NEW.soundex_key := soundex(COALESCE(NEW.preferred_term, ''));
    EXCEPTION WHEN undefined_function THEN
        NEW.metaphone_key := NULL;
        NEW.soundex_key := NULL;
    END;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to concepts table
CREATE TRIGGER update_concepts_search 
    BEFORE INSERT OR UPDATE ON concepts 
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

-- Function for drug concepts search vector
CREATE OR REPLACE FUNCTION update_drug_search_vector() RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', COALESCE(NEW.ingredient, '')), 'A') ||
        setweight(to_tsvector('english', array_to_string(NEW.brand_names, ' ')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.dose_form, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.strength, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_drug_concepts_search 
    BEFORE INSERT OR UPDATE ON drug_concepts 
    FOR EACH ROW EXECUTE FUNCTION update_drug_search_vector();

-- Function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_concept_hierarchy() RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY concept_hierarchy;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- 11. Initial Data Population
-- =====================================================

-- Populate drug concepts from existing RxNorm data
INSERT INTO drug_concepts (
    rxnorm_cui, ingredient, brand_names, is_generic, search_vector
)
SELECT DISTINCT
    c.code,
    c.preferred_term,
    ARRAY[c.preferred_term],
    true, -- Assume generic for now
    to_tsvector('english', c.preferred_term)
FROM concepts c
WHERE c.system = 'RxNorm'
ON CONFLICT (rxnorm_cui) DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE concepts IS 'Partitioned terminology concepts table with enhanced search and hierarchy support';
COMMENT ON TABLE drug_concepts IS 'Specialized table for drug/medication concepts with clinical attributes';
COMMENT ON TABLE lab_references IS 'Laboratory reference ranges linked to LOINC codes with population-specific values';
COMMENT ON TABLE snomed_expressions IS 'SNOMED CT compositional expressions with validation and normalized forms';
COMMENT ON TABLE tenant_overlays IS 'Multi-tenant customizations and overlays for terminology systems';
COMMENT ON TABLE terminology_audit IS 'Audit trail for all terminology operations with performance metrics';

-- =====================================================
-- 12. Maintenance Functions
-- =====================================================

-- Function to create new monthly partitions
CREATE OR REPLACE FUNCTION create_monthly_partition(table_name TEXT, start_date DATE) 
RETURNS VOID AS $$
DECLARE
    partition_name TEXT;
    end_date DATE;
BEGIN
    partition_name := table_name || '_' || to_char(start_date, 'YYYY_MM');
    end_date := start_date + interval '1 month';
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                   partition_name, table_name, start_date, end_date);
END;
$$ LANGUAGE plpgsql;

-- Create initial search configuration for medical terms
CREATE TEXT SEARCH CONFIGURATION medical_english (COPY = english);
ALTER TEXT SEARCH CONFIGURATION medical_english
    ALTER MAPPING FOR asciiword WITH english_stem;

-- Note: Database-level comments require dynamic SQL
DO $$
BEGIN
    EXECUTE format('COMMENT ON DATABASE %I IS %L', current_database(), 'KB-7 Terminology Service with enhanced partitioning and performance optimizations');
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not add database comment - insufficient privileges';
END $$;