-- =====================================================
-- KB-7 Terminology Service Phase 3.5.1 Optimized Schema
-- Fast Lookup Tier for Clinical Decision Support
-- =====================================================
-- Version: 3.5.1
-- Date: 2025-09-22
-- Purpose: High-performance terminology service with <10ms lookup (95th percentile)
-- Supports: SNOMED CT, RxNorm, ICD-10, LOINC, AMT terminologies

-- Enable required extensions for optimal performance
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";            -- Text similarity and trigram matching
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements"; -- Query performance monitoring
CREATE EXTENSION IF NOT EXISTS "fuzzystrmatch";      -- Phonetic matching algorithms
CREATE EXTENSION IF NOT EXISTS "btree_gin";          -- GIN indexes on btree-indexable types
CREATE EXTENSION IF NOT EXISTS "pg_buffercache";     -- Buffer cache inspection for tuning

-- =====================================================
-- 1. TERMINOLOGY_CONCEPTS - Core concept storage
-- =====================================================
-- Purpose: Fast exact code lookups with comprehensive metadata
-- Performance target: <5ms for exact code+system lookups
-- Estimated size: 3M+ concepts (SNOMED: 350K, RxNorm: 200K, ICD-10: 70K, LOINC: 90K, AMT: 50K)

CREATE TABLE IF NOT EXISTS terminology_concepts (
    -- Primary identification
    id BIGSERIAL PRIMARY KEY,
    concept_uuid UUID DEFAULT gen_random_uuid() UNIQUE NOT NULL,

    -- Terminology system identification (constrained for performance)
    system VARCHAR(20) NOT NULL CHECK (system IN ('SNOMED', 'RxNorm', 'ICD10', 'LOINC', 'AMT', 'CPT', 'HCPCS')),
    code VARCHAR(50) NOT NULL,
    version VARCHAR(20) NOT NULL DEFAULT 'latest',

    -- Core concept attributes
    display_name TEXT NOT NULL,
    fully_specified_name TEXT,
    definition TEXT,

    -- Status and lifecycle management
    active BOOLEAN DEFAULT true NOT NULL,
    effective_date DATE,
    inactive_date DATE,

    -- Hierarchy information (denormalized for performance)
    parent_codes TEXT[] DEFAULT '{}',          -- Array of direct parent codes
    ancestor_codes TEXT[] DEFAULT '{}',        -- Array of all ancestor codes (for fast hierarchy queries)
    child_count INTEGER DEFAULT 0,             -- Count of direct children (for UI optimization)
    descendant_count INTEGER DEFAULT 0,        -- Count of all descendants

    -- Search optimization fields
    search_terms TEXT[] DEFAULT '{}',          -- Additional search terms and synonyms
    search_text TEXT,                          -- Concatenated searchable text (auto-populated by trigger)

    -- Flexible metadata storage
    metadata JSONB DEFAULT '{}',
    /* Example metadata structure:
    {
      "attributes": {
        "moduleId": "900000000000207008",
        "definitionStatusId": "900000000000074008"
      },
      "synonyms": ["Alternative term 1", "Alternative term 2"],
      "properties": {
        "strength": "10mg",
        "unit": "tablet",
        "route": "oral"
      },
      "clinical_flags": {
        "high_risk": true,
        "requires_monitoring": true,
        "contraindicated_pregnancy": false
      },
      "regulatory": {
        "fda_approved": true,
        "controlled_substance": false,
        "schedule": null
      }
    }
    */

    -- Policy and governance flags
    policy_flags JSONB DEFAULT '{}',
    /* Example policy flags:
    {
      "doNotAutoMap": false,
      "requiresClinicalReview": false,
      "safetyLevel": "standard",
      "australianOnly": false,
      "regulatoryStatus": "approved",
      "deprecationDate": null,
      "replacementConcept": null
    }
    */

    -- Audit and versioning
    created_by VARCHAR(100),
    updated_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Performance constraint: unique per system+code+version
    CONSTRAINT unique_system_code_version UNIQUE(system, code, version)
);

-- Add table-level comment
COMMENT ON TABLE terminology_concepts IS 'Core terminology concepts with optimized structure for fast lookups and comprehensive metadata support';

-- Add column comments for documentation
COMMENT ON COLUMN terminology_concepts.system IS 'Terminology system identifier (SNOMED, RxNorm, ICD10, LOINC, AMT)';
COMMENT ON COLUMN terminology_concepts.ancestor_codes IS 'Denormalized ancestor hierarchy for fast parent/child queries without recursion';
COMMENT ON COLUMN terminology_concepts.search_text IS 'Auto-populated concatenated search text for full-text search optimization';
COMMENT ON COLUMN terminology_concepts.metadata IS 'Flexible JSONB field for system-specific attributes and clinical properties';
COMMENT ON COLUMN terminology_concepts.policy_flags IS 'Clinical governance and policy enforcement flags';

-- =====================================================
-- 2. TERMINOLOGY_MAPPINGS - Cross-system mappings
-- =====================================================
-- Purpose: Fast cross-terminology translation with confidence scoring
-- Performance target: <10ms for mapping lookups
-- Use cases: ICD-10 to SNOMED mapping, RxNorm to AMT translation

CREATE TABLE IF NOT EXISTS terminology_mappings (
    -- Primary identification
    mapping_id BIGSERIAL PRIMARY KEY,
    mapping_uuid UUID DEFAULT gen_random_uuid() UNIQUE NOT NULL,

    -- Source terminology
    source_system VARCHAR(20) NOT NULL CHECK (source_system IN ('SNOMED', 'RxNorm', 'ICD10', 'LOINC', 'AMT', 'CPT', 'HCPCS')),
    source_code VARCHAR(50) NOT NULL,
    source_version VARCHAR(20) DEFAULT 'latest',

    -- Target terminology
    target_system VARCHAR(20) NOT NULL CHECK (target_system IN ('SNOMED', 'RxNorm', 'ICD10', 'LOINC', 'AMT', 'CPT', 'HCPCS')),
    target_code VARCHAR(50) NOT NULL,
    target_version VARCHAR(20) DEFAULT 'latest',

    -- Mapping metadata
    mapping_type VARCHAR(30) NOT NULL DEFAULT 'equivalent',
    -- Types: 'equivalent', 'broader', 'narrower', 'inexact', 'unmatched', 'disjoint'

    confidence_score DECIMAL(3,2) DEFAULT 1.00 CHECK (confidence_score >= 0.00 AND confidence_score <= 1.00),
    -- 1.00 = exact match, 0.90-0.99 = high confidence, 0.70-0.89 = medium, 0.50-0.69 = low, <0.50 = poor

    -- Status and validation
    active BOOLEAN DEFAULT true NOT NULL,
    verified BOOLEAN DEFAULT false,             -- Manual clinical verification status
    auto_generated BOOLEAN DEFAULT false,       -- Machine-generated vs manual mapping

    -- Source of mapping
    mapping_source VARCHAR(100),               -- Source organization or algorithm
    mapping_authority VARCHAR(100),            -- Authority that validated the mapping

    -- Effective period
    effective_date DATE,
    expiry_date DATE,

    -- Additional mapping context
    mapping_context JSONB DEFAULT '{}',
    /* Example mapping context:
    {
      "algorithm": "lexical_similarity",
      "algorithm_version": "1.2.3",
      "match_strength": 0.95,
      "clinical_context": ["ambulatory", "inpatient"],
      "use_case": ["billing", "clinical_documentation"],
      "notes": "Verified by clinical terminology team",
      "evidence": {
        "publications": ["PMID:12345678"],
        "standards": ["HL7 FHIR ConceptMap"]
      }
    }
    */

    -- Audit trail
    created_by VARCHAR(100),
    verified_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Performance constraints
    CONSTRAINT unique_mapping UNIQUE(source_system, source_code, target_system, target_code, source_version, target_version),
    CONSTRAINT no_self_mapping CHECK (NOT (source_system = target_system AND source_code = target_code))
);

COMMENT ON TABLE terminology_mappings IS 'Cross-terminology mappings with confidence scoring and clinical validation support';
COMMENT ON COLUMN terminology_mappings.confidence_score IS 'Mapping confidence from 0.00 (poor) to 1.00 (exact match)';
COMMENT ON COLUMN terminology_mappings.mapping_context IS 'Algorithm details, clinical context, and validation evidence';

-- =====================================================
-- 3. CONCEPT_RELATIONSHIPS - Semantic relationships
-- =====================================================
-- Purpose: Concept-to-concept relationships for navigation and reasoning
-- Performance target: <15ms for relationship traversal queries
-- Use cases: Parent-child hierarchy, part-of relationships, has-ingredient

CREATE TABLE IF NOT EXISTS concept_relationships (
    -- Primary identification
    relationship_id BIGSERIAL PRIMARY KEY,
    relationship_uuid UUID DEFAULT gen_random_uuid() UNIQUE NOT NULL,

    -- Source and target concepts (foreign keys to terminology_concepts)
    source_concept_id BIGINT NOT NULL REFERENCES terminology_concepts(id) ON DELETE CASCADE,
    target_concept_id BIGINT NOT NULL REFERENCES terminology_concepts(id) ON DELETE CASCADE,

    -- Relationship metadata
    relationship_type VARCHAR(50) NOT NULL,
    -- Common types: 'is_a', 'part_of', 'has_ingredient', 'has_component', 'has_strength',
    --               'has_dose_form', 'contraindicated_with', 'interacts_with'

    relationship_group INTEGER DEFAULT 0,      -- For grouping related relationships
    characteristic_type VARCHAR(20) DEFAULT 'stated' CHECK (characteristic_type IN ('stated', 'inferred')),

    -- Status and lifecycle
    active BOOLEAN DEFAULT true NOT NULL,
    effective_date DATE,
    inactive_date DATE,

    -- Relationship strength and context
    strength DECIMAL(3,2) DEFAULT 1.00 CHECK (strength >= 0.00 AND strength <= 1.00),
    relationship_context JSONB DEFAULT '{}',
    /* Example relationship context:
    {
      "clinical_significance": "high",
      "evidence_level": "A",
      "frequency": "common",
      "severity": "moderate",
      "onset": "immediate",
      "mechanism": "pharmacokinetic",
      "references": ["PMID:87654321"]
    }
    */

    -- Audit trail
    created_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,

    -- Performance constraints
    CONSTRAINT unique_relationship UNIQUE(source_concept_id, target_concept_id, relationship_type, relationship_group),
    CONSTRAINT no_self_relationship CHECK (source_concept_id != target_concept_id)
);

COMMENT ON TABLE concept_relationships IS 'Semantic relationships between concepts for hierarchy navigation and clinical reasoning';
COMMENT ON COLUMN concept_relationships.relationship_group IS 'Groups related relationships together (e.g., multiple ingredients)';
COMMENT ON COLUMN concept_relationships.characteristic_type IS 'Whether relationship is explicitly stated or computationally inferred';

-- =====================================================
-- 4. PERFORMANCE INDEXES - Optimized for <10ms lookups
-- =====================================================

-- *** TERMINOLOGY_CONCEPTS INDEXES ***

-- Primary lookup index: system + code (most common query pattern)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_lookup
    ON terminology_concepts(system, code)
    INCLUDE (display_name, active, version);
COMMENT ON INDEX idx_terminology_lookup IS 'Primary lookup index for system+code queries with included columns';

-- Active concepts filter index (with partial index for performance)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_active
    ON terminology_concepts(system, active, version)
    WHERE active = true;
COMMENT ON INDEX idx_terminology_active IS 'Partial index for active concepts only - reduces index size by 15-20%';

-- UUID lookup index (for federated queries)
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_uuid
    ON terminology_concepts(concept_uuid);

-- Full-text search index using GIN
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_fulltext
    ON terminology_concepts USING gin(to_tsvector('english',
        COALESCE(display_name, '') || ' ' ||
        COALESCE(fully_specified_name, '') || ' ' ||
        COALESCE(search_text, '')
    ));
COMMENT ON INDEX idx_terminology_fulltext IS 'Full-text search across all searchable text fields';

-- Trigram search index for fuzzy matching
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_trigram
    ON terminology_concepts USING gin(display_name gin_trgm_ops);
COMMENT ON INDEX idx_terminology_trigram IS 'Trigram index for fuzzy text matching and similarity search';

-- Hierarchy navigation indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_parents
    ON terminology_concepts USING gin(parent_codes);
COMMENT ON INDEX idx_terminology_parents IS 'GIN index for parent code array queries';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_ancestors
    ON terminology_concepts USING gin(ancestor_codes);
COMMENT ON INDEX idx_terminology_ancestors IS 'GIN index for ancestor hierarchy queries';

-- JSONB metadata index for flexible queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_metadata
    ON terminology_concepts USING gin(metadata);
COMMENT ON INDEX idx_terminology_metadata IS 'GIN index for JSONB metadata queries';

-- Policy flags index for governance queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_policy
    ON terminology_concepts USING gin(policy_flags);
COMMENT ON INDEX idx_terminology_policy IS 'GIN index for policy flag queries';

-- Composite index for version-aware queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_terminology_system_version
    ON terminology_concepts(system, version, active)
    WHERE active = true;

-- *** TERMINOLOGY_MAPPINGS INDEXES ***

-- Primary mapping lookup: source system + code
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mapping_source
    ON terminology_mappings(source_system, source_code)
    INCLUDE (target_system, target_code, confidence_score, active);
COMMENT ON INDEX idx_mapping_source IS 'Primary source lookup with included target information';

-- Reverse mapping lookup: target system + code
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mapping_target
    ON terminology_mappings(target_system, target_code)
    INCLUDE (source_system, source_code, confidence_score, active);
COMMENT ON INDEX idx_mapping_target IS 'Reverse lookup index for bidirectional mapping queries';

-- Confidence-based filtering (partial index for high-confidence mappings)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mapping_high_confidence
    ON terminology_mappings(source_system, source_code, confidence_score)
    WHERE confidence_score >= 0.80 AND active = true;
COMMENT ON INDEX idx_mapping_high_confidence IS 'Partial index for high-confidence mappings only';

-- Mapping type analysis index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mapping_type
    ON terminology_mappings(mapping_type, active)
    WHERE active = true;

-- Cross-system mapping matrix index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_mapping_systems
    ON terminology_mappings(source_system, target_system, active)
    WHERE active = true;

-- UUID lookup for federated queries
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_mapping_uuid
    ON terminology_mappings(mapping_uuid);

-- *** CONCEPT_RELATIONSHIPS INDEXES ***

-- Primary relationship lookup: source concept
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationship_source
    ON concept_relationships(source_concept_id, relationship_type)
    INCLUDE (target_concept_id, active, strength);
COMMENT ON INDEX idx_relationship_source IS 'Primary source concept relationship lookup';

-- Reverse relationship lookup: target concept
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationship_target
    ON concept_relationships(target_concept_id, relationship_type)
    INCLUDE (source_concept_id, active, strength);
COMMENT ON INDEX idx_relationship_target IS 'Reverse relationship lookup for target concepts';

-- Relationship type analysis
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationship_type
    ON concept_relationships(relationship_type, active)
    WHERE active = true;

-- Bi-directional relationship index
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationship_bidirectional
    ON concept_relationships(source_concept_id, target_concept_id, relationship_type);

-- Relationship strength filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_relationship_strength
    ON concept_relationships(relationship_type, strength)
    WHERE active = true AND strength >= 0.70;

-- UUID lookup
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_relationship_uuid
    ON concept_relationships(relationship_uuid);

-- =====================================================
-- 5. TRIGGERS FOR AUTOMATIC MAINTENANCE
-- =====================================================

-- Function to update search_text column automatically
CREATE OR REPLACE FUNCTION update_concept_search_text()
RETURNS TRIGGER AS $$
BEGIN
    -- Concatenate all searchable fields for full-text search
    NEW.search_text := COALESCE(NEW.display_name, '') || ' ' ||
                       COALESCE(NEW.fully_specified_name, '') || ' ' ||
                       COALESCE(NEW.definition, '') || ' ' ||
                       COALESCE(array_to_string(NEW.search_terms, ' '), '');

    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain search_text
CREATE TRIGGER trigger_update_concept_search_text
    BEFORE INSERT OR UPDATE ON terminology_concepts
    FOR EACH ROW
    EXECUTE FUNCTION update_concept_search_text();

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at maintenance
CREATE TRIGGER trigger_terminology_mappings_updated_at
    BEFORE UPDATE ON terminology_mappings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_concept_relationships_updated_at
    BEFORE UPDATE ON concept_relationships
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- 6. PARTITIONING STRATEGY (for large-scale deployment)
-- =====================================================

-- Note: Partitioning can be enabled later when table size exceeds 10M records
-- Recommended partitioning strategy:
--   - terminology_concepts: PARTITION BY HASH(system) or RANGE(created_at)
--   - terminology_mappings: PARTITION BY HASH(source_system)
--   - concept_relationships: PARTITION BY HASH(source_concept_id)

-- Example partitioning setup (commented out for initial deployment):
/*
-- Create partitioned table for concepts
CREATE TABLE terminology_concepts_partitioned (
    LIKE terminology_concepts INCLUDING ALL
) PARTITION BY HASH(system);

-- Create partitions for each major terminology system
CREATE TABLE terminology_concepts_snomed PARTITION OF terminology_concepts_partitioned
    FOR VALUES WITH (modulus 5, remainder 0);
CREATE TABLE terminology_concepts_rxnorm PARTITION OF terminology_concepts_partitioned
    FOR VALUES WITH (modulus 5, remainder 1);
-- ... additional partitions
*/

-- =====================================================
-- 7. PERFORMANCE TUNING PARAMETERS
-- =====================================================

-- Set optimal PostgreSQL parameters for terminology workload
-- These should be added to postgresql.conf for production deployment

/*
# Memory configuration for large terminology datasets
shared_buffers = 2GB                    # 25% of system RAM for large datasets
effective_cache_size = 6GB              # 75% of system RAM
work_mem = 256MB                        # For large sort operations
maintenance_work_mem = 1GB              # For index creation and VACUUM

# Parallel query configuration
max_parallel_workers_per_gather = 4     # Parallel index scans
max_parallel_workers = 8                # Total parallel workers
max_parallel_maintenance_workers = 4    # Parallel index builds

# I/O configuration
random_page_cost = 1.1                  # Assumes SSD storage
seq_page_cost = 1.0                     # Sequential scan cost
effective_io_concurrency = 200          # SSD concurrent I/O

# Query planning
default_statistics_target = 500         # Higher statistics for better plans
constraint_exclusion = partition        # Partition pruning

# Logging and monitoring
log_statement = 'mod'                   # Log DDL statements
log_min_duration_statement = 100        # Log slow queries > 100ms
track_activity_query_size = 16384       # Larger query text tracking

# Autovacuum tuning for terminology data
autovacuum_vacuum_scale_factor = 0.05   # More frequent vacuum
autovacuum_analyze_scale_factor = 0.025 # More frequent analyze
autovacuum_vacuum_cost_limit = 2000     # Higher vacuum throughput
*/

-- =====================================================
-- 8. SCHEMA VALIDATION AND CONSTRAINTS
-- =====================================================

-- Add additional constraints for data integrity
ALTER TABLE terminology_concepts
    ADD CONSTRAINT check_concept_dates
    CHECK (inactive_date IS NULL OR effective_date IS NULL OR inactive_date >= effective_date);

ALTER TABLE terminology_mappings
    ADD CONSTRAINT check_mapping_dates
    CHECK (expiry_date IS NULL OR effective_date IS NULL OR expiry_date >= effective_date);

ALTER TABLE concept_relationships
    ADD CONSTRAINT check_relationship_dates
    CHECK (inactive_date IS NULL OR effective_date IS NULL OR inactive_date >= effective_date);

-- =====================================================
-- 9. MATERIALIZED VIEWS FOR COMPLEX QUERIES
-- =====================================================

-- Materialized view for terminology system statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS terminology_stats AS
SELECT
    system,
    version,
    COUNT(*) as total_concepts,
    COUNT(*) FILTER (WHERE active = true) as active_concepts,
    COUNT(*) FILTER (WHERE active = false) as inactive_concepts,
    MIN(created_at) as first_loaded,
    MAX(updated_at) as last_updated,
    AVG(array_length(parent_codes, 1)) as avg_parents,
    AVG(array_length(ancestor_codes, 1)) as avg_ancestors
FROM terminology_concepts
GROUP BY system, version
ORDER BY system, version;

CREATE UNIQUE INDEX ON terminology_stats (system, version);

-- Materialized view for mapping coverage matrix
CREATE MATERIALIZED VIEW IF NOT EXISTS mapping_coverage AS
SELECT
    source_system,
    target_system,
    COUNT(*) as total_mappings,
    COUNT(*) FILTER (WHERE active = true) as active_mappings,
    AVG(confidence_score) as avg_confidence,
    COUNT(*) FILTER (WHERE confidence_score >= 0.90) as high_confidence_mappings,
    COUNT(*) FILTER (WHERE verified = true) as verified_mappings
FROM terminology_mappings
GROUP BY source_system, target_system
ORDER BY source_system, target_system;

CREATE UNIQUE INDEX ON mapping_coverage (source_system, target_system);

-- =====================================================
-- 10. INITIAL DATA VALIDATION
-- =====================================================

-- Create function to validate schema integrity
CREATE OR REPLACE FUNCTION validate_terminology_schema()
RETURNS TABLE(
    check_name TEXT,
    status TEXT,
    details TEXT
) AS $$
BEGIN
    -- Check if all required indexes exist
    RETURN QUERY
    SELECT
        'Required Indexes'::TEXT,
        CASE WHEN COUNT(*) >= 15 THEN 'PASS' ELSE 'FAIL' END::TEXT,
        FORMAT('Found %s of 15+ required indexes', COUNT(*))::TEXT
    FROM pg_indexes
    WHERE schemaname = 'public'
    AND tablename IN ('terminology_concepts', 'terminology_mappings', 'concept_relationships');

    -- Check foreign key constraints
    RETURN QUERY
    SELECT
        'Foreign Key Constraints'::TEXT,
        CASE WHEN COUNT(*) >= 2 THEN 'PASS' ELSE 'FAIL' END::TEXT,
        FORMAT('Found %s foreign key constraints', COUNT(*))::TEXT
    FROM information_schema.table_constraints
    WHERE constraint_type = 'FOREIGN KEY'
    AND table_name IN ('terminology_mappings', 'concept_relationships');

    -- Check trigger existence
    RETURN QUERY
    SELECT
        'Maintenance Triggers'::TEXT,
        CASE WHEN COUNT(*) >= 3 THEN 'PASS' ELSE 'FAIL' END::TEXT,
        FORMAT('Found %s maintenance triggers', COUNT(*))::TEXT
    FROM information_schema.triggers
    WHERE event_object_table IN ('terminology_concepts', 'terminology_mappings', 'concept_relationships');

END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- SCHEMA DEPLOYMENT COMPLETE
-- =====================================================

-- Log successful schema creation
INSERT INTO terminology_concepts (
    system, code, display_name,
    metadata, created_by
) VALUES (
    'SYSTEM', 'SCHEMA_DEPLOYED', 'KB7 Terminology Schema v3.5.1 Deployed',
    jsonb_build_object(
        'schema_version', '3.5.1',
        'deployment_date', NOW(),
        'performance_target', '<10ms lookups',
        'supported_systems', ARRAY['SNOMED', 'RxNorm', 'ICD10', 'LOINC', 'AMT']
    ),
    'kb7_schema_migration'
) ON CONFLICT (system, code, version) DO NOTHING;

-- Run validation
SELECT * FROM validate_terminology_schema();

-- Display schema summary
SELECT
    'Schema Deployment Summary'::TEXT as summary,
    FORMAT('Tables: %s, Indexes: %s, Triggers: %s',
        (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name LIKE '%terminology%' OR table_name LIKE '%concept%'),
        (SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public' AND (tablename LIKE '%terminology%' OR tablename LIKE '%concept%')),
        (SELECT COUNT(*) FROM information_schema.triggers WHERE event_object_table LIKE '%terminology%' OR event_object_table LIKE '%concept%')
    )::TEXT as details;