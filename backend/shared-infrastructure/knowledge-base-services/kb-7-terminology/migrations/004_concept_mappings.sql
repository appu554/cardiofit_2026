-- KB-7 Terminology Service: Concept Mapping Tables
-- Phase 3: Complete Concept Mapping Implementation
-- This migration creates comprehensive concept mapping infrastructure

-- Drop the old concept_mappings table from migration 001 (schema incompatible)
-- The old table uses UUID id and source_system_id/target_system_id references
-- The new table uses VARCHAR id and inline source_system/target_system strings
DROP TABLE IF EXISTS concept_mappings CASCADE;

-- Create concept mappings table with confidence scoring
CREATE TABLE concept_mappings (
    id VARCHAR(255) PRIMARY KEY,
    source_system VARCHAR(50) NOT NULL,
    source_code VARCHAR(255) NOT NULL,
    source_display VARCHAR(500),
    target_system VARCHAR(50) NOT NULL, 
    target_code VARCHAR(255) NOT NULL,
    target_display VARCHAR(500),
    equivalence VARCHAR(20) NOT NULL DEFAULT 'relatedto',
    confidence_score NUMERIC(3,2) NOT NULL DEFAULT 0.5,
    comment TEXT,
    depends_on TEXT[], -- Array of mapping IDs this depends on
    product JSONB, -- Complex mappings (one-to-many, many-to-one)
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT chk_confidence_score CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    CONSTRAINT chk_equivalence CHECK (equivalence IN ('equivalent', 'relatedto', 'inexact', 'unmatched', 'disjoint')),
    CONSTRAINT uk_concept_mapping UNIQUE (source_system, source_code, target_system, target_code)
);

-- Create indexes for efficient mapping lookups
CREATE INDEX IF NOT EXISTS idx_concept_mappings_source ON concept_mappings(source_system, source_code);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_target ON concept_mappings(target_system, target_code);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_confidence ON concept_mappings(confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_equivalence ON concept_mappings(equivalence);
CREATE INDEX IF NOT EXISTS idx_concept_mappings_created_at ON concept_mappings(created_at);

-- Create concept map sets table for organizing mappings
CREATE TABLE IF NOT EXISTS concept_map_sets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    source_system VARCHAR(50) NOT NULL,
    target_system VARCHAR(50) NOT NULL,
    version VARCHAR(50),
    publisher VARCHAR(255),
    purpose TEXT,
    copyright TEXT,
    status VARCHAR(20) DEFAULT 'draft',
    experimental BOOLEAN DEFAULT false,
    date_created TIMESTAMP DEFAULT NOW(),
    last_updated TIMESTAMP,
    
    CONSTRAINT chk_status CHECK (status IN ('draft', 'active', 'retired', 'unknown'))
);

-- Link mappings to map sets
CREATE TABLE IF NOT EXISTS concept_map_set_mappings (
    map_set_id UUID REFERENCES concept_map_sets(id) ON DELETE CASCADE,
    mapping_id VARCHAR(255) REFERENCES concept_mappings(id) ON DELETE CASCADE,
    priority INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (map_set_id, mapping_id)
);

-- Create mapping validation results table
CREATE TABLE IF NOT EXISTS mapping_validations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mapping_id VARCHAR(255) REFERENCES concept_mappings(id) ON DELETE CASCADE,
    validation_type VARCHAR(50) NOT NULL, -- 'automated', 'manual', 'expert_review'
    validator VARCHAR(255), -- System or person who performed validation
    result VARCHAR(20) NOT NULL, -- 'valid', 'invalid', 'uncertain'
    confidence_adjustment NUMERIC(3,2), -- Adjustment to confidence score
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT chk_validation_result CHECK (result IN ('valid', 'invalid', 'uncertain')),
    CONSTRAINT chk_confidence_adjustment CHECK (confidence_adjustment >= -1.0 AND confidence_adjustment <= 1.0)
);

-- Create mapping usage statistics table
CREATE TABLE IF NOT EXISTS mapping_usage_stats (
    mapping_id VARCHAR(255) REFERENCES concept_mappings(id) ON DELETE CASCADE,
    usage_date DATE DEFAULT CURRENT_DATE,
    usage_count INTEGER DEFAULT 1,
    success_rate NUMERIC(3,2),
    avg_response_time_ms NUMERIC(8,2),
    last_used TIMESTAMP DEFAULT NOW(),
    
    PRIMARY KEY (mapping_id, usage_date)
);

-- Insert common terminology system mappings
INSERT INTO concept_mappings (id, source_system, source_code, source_display, target_system, target_code, target_display, equivalence, confidence_score, comment)
VALUES
    -- SNOMED to ICD-10 common mappings
    ('snomed_icd10_essential_htn', 'SNOMED', '59621000', 'Essential hypertension', 'ICD-10-CM', 'I10', 'Essential hypertension', 'equivalent', 1.0, 'Direct equivalent mapping'),
    ('snomed_icd10_type2_diabetes', 'SNOMED', '44054006', 'Diabetes mellitus type 2', 'ICD-10-CM', 'E11.9', 'Type 2 diabetes mellitus without complications', 'relatedto', 0.9, 'ICD-10 requires more specific code'),
    ('snomed_icd10_copd', 'SNOMED', '13645005', 'Chronic obstructive lung disease', 'ICD-10-CM', 'J44.1', 'Chronic obstructive pulmonary disease with acute exacerbation', 'inexact', 0.7, 'SNOMED more general'),
    
    -- RxNorm to SNOMED mappings
    ('rxnorm_snomed_lisinopril', 'RxNorm', '29046', 'Lisinopril', 'SNOMED', '386872004', 'Lisinopril', 'equivalent', 1.0, 'Exact drug mapping'),
    ('rxnorm_snomed_metformin', 'RxNorm', '6809', 'Metformin', 'SNOMED', '387467008', 'Metformin', 'equivalent', 1.0, 'Exact drug mapping'),
    
    -- LOINC to SNOMED mappings  
    ('loinc_snomed_glucose', 'LOINC', '2345-7', 'Glucose [Mass/volume] in Serum or Plasma', 'SNOMED', '33747003', 'Glucose measurement', 'relatedto', 0.8, 'LOINC more specific than SNOMED'),
    ('loinc_snomed_hemoglobin', 'LOINC', '718-7', 'Hemoglobin [Mass/volume] in Blood', 'SNOMED', '38082009', 'Hemoglobin measurement', 'relatedto', 0.8, 'LOINC more specific')
ON CONFLICT (source_system, source_code, target_system, target_code) DO UPDATE SET
    confidence_score = EXCLUDED.confidence_score,
    comment = EXCLUDED.comment,
    updated_at = NOW();

-- Create default concept map set
INSERT INTO concept_map_sets (name, title, description, source_system, target_system, version, publisher, status)
VALUES 
    ('snomed-to-icd10-cm', 'SNOMED CT to ICD-10-CM Mapping', 'Community-maintained mappings between SNOMED CT and ICD-10-CM', 'SNOMED', 'ICD-10-CM', '1.0', 'KB-7 Terminology Service', 'active'),
    ('rxnorm-to-snomed', 'RxNorm to SNOMED CT Drug Mapping', 'Drug concept mappings between RxNorm and SNOMED CT', 'RxNorm', 'SNOMED', '1.0', 'KB-7 Terminology Service', 'active'),
    ('loinc-to-snomed', 'LOINC to SNOMED CT Lab Mapping', 'Laboratory test mappings between LOINC and SNOMED CT', 'LOINC', 'SNOMED', '1.0', 'KB-7 Terminology Service', 'active')
ON CONFLICT (name) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    last_updated = NOW();

-- Link mappings to map sets
INSERT INTO concept_map_set_mappings (map_set_id, mapping_id, priority)
SELECT 
    cms.id,
    cm.id,
    1
FROM concept_map_sets cms
JOIN concept_mappings cm ON (
    (cms.source_system = cm.source_system AND cms.target_system = cm.target_system) OR
    (cms.name = 'snomed-to-icd10-cm' AND cm.source_system = 'SNOMED' AND cm.target_system = 'ICD-10-CM') OR
    (cms.name = 'rxnorm-to-snomed' AND cm.source_system = 'RxNorm' AND cm.target_system = 'SNOMED') OR
    (cms.name = 'loinc-to-snomed' AND cm.source_system = 'LOINC' AND cm.target_system = 'SNOMED')
)
ON CONFLICT (map_set_id, mapping_id) DO NOTHING;

-- Create functions for mapping confidence scoring
CREATE OR REPLACE FUNCTION calculate_mapping_confidence(
    source_display TEXT,
    target_display TEXT,
    equivalence_type TEXT DEFAULT 'relatedto'
) RETURNS NUMERIC(3,2) AS $$
DECLARE
    base_score NUMERIC(3,2);
    similarity_score NUMERIC(3,2);
    final_score NUMERIC(3,2);
BEGIN
    -- Base score from equivalence type
    base_score := CASE equivalence_type
        WHEN 'equivalent' THEN 1.0
        WHEN 'relatedto' THEN 0.8
        WHEN 'inexact' THEN 0.6
        WHEN 'unmatched' THEN 0.2
        WHEN 'disjoint' THEN 0.0
        ELSE 0.5
    END;
    
    -- Calculate text similarity if displays are provided
    IF source_display IS NOT NULL AND target_display IS NOT NULL THEN
        similarity_score := similarity(LOWER(source_display), LOWER(target_display));
        
        -- Weighted combination: 70% equivalence type, 30% text similarity
        final_score := (base_score * 0.7) + (similarity_score * 0.3);
    ELSE
        final_score := base_score;
    END IF;
    
    -- Ensure score is within bounds
    final_score := LEAST(1.0, GREATEST(0.0, final_score));
    
    RETURN final_score;
END;
$$ LANGUAGE plpgsql;

-- Create function to find potential mappings using semantic similarity
CREATE OR REPLACE FUNCTION find_semantic_mappings(
    source_system TEXT,
    source_code TEXT,
    target_system TEXT,
    min_similarity NUMERIC(3,2) DEFAULT 0.6,
    max_results INTEGER DEFAULT 10
) RETURNS TABLE (
    target_code TEXT,
    target_display TEXT,
    similarity_score NUMERIC(3,2),
    suggested_equivalence TEXT,
    suggested_confidence NUMERIC(3,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        c2.code as target_code,
        c2.preferred_term as target_display,
        similarity(c1.preferred_term, c2.preferred_term) as similarity_score,
        CASE 
            WHEN similarity(c1.preferred_term, c2.preferred_term) >= 0.95 THEN 'equivalent'
            WHEN similarity(c1.preferred_term, c2.preferred_term) >= 0.8 THEN 'relatedto'
            ELSE 'inexact'
        END as suggested_equivalence,
        calculate_mapping_confidence(
            c1.preferred_term, 
            c2.preferred_term,
            CASE 
                WHEN similarity(c1.preferred_term, c2.preferred_term) >= 0.95 THEN 'equivalent'
                WHEN similarity(c1.preferred_term, c2.preferred_term) >= 0.8 THEN 'relatedto'
                ELSE 'inexact'
            END
        ) as suggested_confidence
    FROM concepts c1
    JOIN concepts c2 ON c2.system = target_system AND c2.active = true
    WHERE c1.system = source_system 
      AND c1.code = source_code 
      AND c1.active = true
      AND similarity(c1.preferred_term, c2.preferred_term) >= min_similarity
    ORDER BY similarity_score DESC
    LIMIT max_results;
END;
$$ LANGUAGE plpgsql;

-- Create mapping quality metrics view
CREATE MATERIALIZED VIEW IF NOT EXISTS mapping_quality_metrics AS
SELECT 
    source_system,
    target_system,
    COUNT(*) as total_mappings,
    COUNT(*) FILTER (WHERE equivalence = 'equivalent') as equivalent_mappings,
    COUNT(*) FILTER (WHERE equivalence = 'relatedto') as related_mappings,
    COUNT(*) FILTER (WHERE equivalence = 'inexact') as inexact_mappings,
    AVG(confidence_score) as avg_confidence,
    MIN(confidence_score) as min_confidence,
    MAX(confidence_score) as max_confidence,
    COUNT(DISTINCT source_code) as unique_source_concepts,
    COUNT(DISTINCT target_code) as unique_target_concepts,
    COUNT(*) FILTER (WHERE confidence_score >= 0.8) as high_confidence_mappings,
    COUNT(*) FILTER (WHERE confidence_score < 0.5) as low_confidence_mappings
FROM concept_mappings
GROUP BY source_system, target_system;

-- Create indexes on materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_mapping_quality_metrics_systems
ON mapping_quality_metrics(source_system, target_system);

-- Create trigger to update mapping confidence scores
CREATE OR REPLACE FUNCTION update_mapping_confidence()
RETURNS TRIGGER AS $$
BEGIN
    -- Recalculate confidence if equivalence changed or displays updated
    IF OLD.equivalence IS DISTINCT FROM NEW.equivalence OR
       OLD.source_display IS DISTINCT FROM NEW.source_display OR
       OLD.target_display IS DISTINCT FROM NEW.target_display THEN
        
        NEW.confidence_score := calculate_mapping_confidence(
            NEW.source_display,
            NEW.target_display,
            NEW.equivalence
        );
        NEW.updated_at := NOW();
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to concept_mappings table
DROP TRIGGER IF EXISTS trigger_update_mapping_confidence ON concept_mappings;
CREATE TRIGGER trigger_update_mapping_confidence
    BEFORE UPDATE ON concept_mappings
    FOR EACH ROW EXECUTE FUNCTION update_mapping_confidence();

-- Create function to refresh mapping quality metrics
CREATE OR REPLACE FUNCTION refresh_mapping_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mapping_quality_metrics;
END;
$$ LANGUAGE plpgsql;

-- Create mapping translation cache table for performance
CREATE TABLE IF NOT EXISTS mapping_translation_cache (
    cache_key VARCHAR(255) PRIMARY KEY,
    source_system VARCHAR(50) NOT NULL,
    source_code VARCHAR(255) NOT NULL,
    target_system VARCHAR(50) NOT NULL,
    mappings JSONB NOT NULL,
    confidence_score NUMERIC(3,2),
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT (NOW() + INTERVAL '1 hour')
);

-- Create index for cache cleanup (PostgreSQL requires separate CREATE INDEX)
CREATE INDEX IF NOT EXISTS idx_mapping_cache_expires ON mapping_translation_cache(expires_at);

-- Create cleanup function for expired cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_mapping_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM mapping_translation_cache WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Schedule cache cleanup (requires pg_cron extension)
-- SELECT cron.schedule('cleanup-mapping-cache', '*/30 * * * *', 'SELECT cleanup_expired_mapping_cache();');

-- Update statistics
ANALYZE concept_mappings;
ANALYZE concept_map_sets;
ANALYZE concept_map_set_mappings;
ANALYZE mapping_validations;
ANALYZE mapping_usage_stats;

-- Refresh materialized view
REFRESH MATERIALIZED VIEW mapping_quality_metrics;

-- Migration completion log
INSERT INTO migration_log (migration_name, status, completed_at)
VALUES ('004_concept_mappings', 'completed', NOW())
ON CONFLICT (migration_name) DO UPDATE SET 
    status = 'completed', 
    completed_at = NOW();

-- Performance validation queries
/*
-- Test mapping lookup performance
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM concept_mappings 
WHERE source_system = 'SNOMED' AND source_code = '59621000';

-- Test semantic mapping function
SELECT * FROM find_semantic_mappings('SNOMED', '44054006', 'ICD-10-CM', 0.6, 5);

-- Test mapping quality metrics
SELECT * FROM mapping_quality_metrics;

-- Test confidence calculation
SELECT calculate_mapping_confidence('Essential hypertension', 'Essential hypertension', 'equivalent');
SELECT calculate_mapping_confidence('Diabetes mellitus', 'Diabetes', 'relatedto');
*/

COMMIT;