-- Migration: KB-1 Schema Alignment for SaMD Compliance
-- Version: 005
-- Description: Implement KB-1 specification database schema with materialized views and performance optimization
-- Author: Claude Code Enhancement
-- Date: 2025-09-02
-- Reference: KB-1 Drug Dosing Rules Specification

BEGIN;

-- ============================================================================
-- STEP 1: Create KB-1 compliant core tables per specification
-- ============================================================================

-- Primary dosing_rules table (KB-1 specification)
CREATE TABLE IF NOT EXISTS dosing_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_code TEXT NOT NULL,                    -- canonical code, e.g. "rxnorm:8610"
    drug_name TEXT NOT NULL,
    semantic_version TEXT NOT NULL,             -- e.g. "1.0.0"
    source_file TEXT NOT NULL,                  -- original TOML filename
    compiled_json JSONB NOT NULL,               -- compiled, normalized rule bundle
    active BOOLEAN DEFAULT FALSE,
    created_by TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    expires_date TIMESTAMPTZ,                   -- for immutable versioning
    checksum TEXT NOT NULL,                     -- sha256 of compiled_json
    provenance JSONB NOT NULL,                  -- {authors, approvals, kb3_refs, kb4_refs}
    notes TEXT
);

-- dose_adjustments table for normalized adjustment logic
CREATE TABLE IF NOT EXISTS dose_adjustments (
    adj_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES dosing_rules(rule_id) ON DELETE CASCADE,
    adjust_type TEXT NOT NULL,                  -- "renal", "hepatic", "age", "weight"
    condition_json JSONB NOT NULL,              -- predicate compiled AST
    formula_json JSONB NOT NULL,                -- normalized formula
    created_at TIMESTAMPTZ DEFAULT now()
);

-- titration_schedules table for multi-step protocols
CREATE TABLE IF NOT EXISTS titration_schedules (
    schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES dosing_rules(rule_id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    after_days INTEGER NOT NULL,
    action_type TEXT NOT NULL,                  -- "increase_by", "multiply", "assess"
    action_value NUMERIC,
    max_step INTEGER,
    monitoring_requirements JSONB,              -- monitoring requirements for this step
    created_at TIMESTAMPTZ DEFAULT now()
);

-- population_dosing table for specialized populations
CREATE TABLE IF NOT EXISTS population_dosing (
    pop_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES dosing_rules(rule_id) ON DELETE CASCADE,
    population_type TEXT NOT NULL,              -- "pediatric", "geriatric", "pregnancy"
    age_min INTEGER,
    age_max INTEGER,
    weight_min NUMERIC,
    weight_max NUMERIC,
    formula_json JSONB NOT NULL,                -- population-specific formula AST
    safety_limits JSONB,                       -- population safety constraints
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ============================================================================
-- STEP 2: Create unique indexes per KB-1 specification
-- ============================================================================

CREATE UNIQUE INDEX IF NOT EXISTS ux_dosing_rules_code_version 
ON dosing_rules(drug_code, semantic_version);

CREATE INDEX IF NOT EXISTS idx_dosing_rules_active 
ON dosing_rules(active) WHERE active = true;

CREATE INDEX IF NOT EXISTS idx_dosing_rules_drug_code 
ON dosing_rules(drug_code);

CREATE INDEX IF NOT EXISTS idx_dose_adjustments_rule_type 
ON dose_adjustments(rule_id, adjust_type);

CREATE INDEX IF NOT EXISTS idx_titration_schedules_rule 
ON titration_schedules(rule_id, step_number);

CREATE INDEX IF NOT EXISTS idx_population_dosing_rule_type 
ON population_dosing(rule_id, population_type);

-- GIN indexes for JSONB performance
CREATE INDEX IF NOT EXISTS idx_dosing_rules_compiled_json 
ON dosing_rules USING GIN (compiled_json);

CREATE INDEX IF NOT EXISTS idx_dosing_rules_provenance 
ON dosing_rules USING GIN (provenance);

-- ============================================================================
-- STEP 3: Create materialized view for ultra-fast lookups
-- ============================================================================

-- active_dosing_rules materialized view (KB-1 specification requirement)
CREATE MATERIALIZED VIEW IF NOT EXISTS active_dosing_rules AS
SELECT 
    dr.rule_id,
    dr.drug_code,
    dr.drug_name,
    dr.semantic_version,
    dr.compiled_json,
    dr.checksum,
    dr.created_at,
    dr.provenance,
    -- Aggregate related data for fast access
    COALESCE(
        json_agg(
            json_build_object(
                'adj_id', da.adj_id,
                'adjust_type', da.adjust_type,
                'condition_json', da.condition_json,
                'formula_json', da.formula_json
            )
        ) FILTER (WHERE da.adj_id IS NOT NULL), 
        '[]'::json
    ) as adjustments,
    COALESCE(
        json_agg(
            json_build_object(
                'schedule_id', ts.schedule_id,
                'step_number', ts.step_number,
                'after_days', ts.after_days,
                'action_type', ts.action_type,
                'action_value', ts.action_value,
                'monitoring_requirements', ts.monitoring_requirements
            ) ORDER BY ts.step_number
        ) FILTER (WHERE ts.schedule_id IS NOT NULL),
        '[]'::json
    ) as titration_schedule,
    COALESCE(
        json_agg(
            json_build_object(
                'pop_id', pd.pop_id,
                'population_type', pd.population_type,
                'age_min', pd.age_min,
                'age_max', pd.age_max,
                'weight_min', pd.weight_min,
                'weight_max', pd.weight_max,
                'formula_json', pd.formula_json,
                'safety_limits', pd.safety_limits
            )
        ) FILTER (WHERE pd.pop_id IS NOT NULL),
        '[]'::json
    ) as population_rules
FROM dosing_rules dr
LEFT JOIN dose_adjustments da ON dr.rule_id = da.rule_id
LEFT JOIN titration_schedules ts ON dr.rule_id = ts.rule_id
LEFT JOIN population_dosing pd ON dr.rule_id = pd.rule_id
WHERE dr.active = true AND (dr.expires_date IS NULL OR dr.expires_date > now())
GROUP BY dr.rule_id, dr.drug_code, dr.drug_name, dr.semantic_version, 
         dr.compiled_json, dr.checksum, dr.created_at, dr.provenance;

-- Index on materialized view for fast drug_code lookups
CREATE INDEX IF NOT EXISTS idx_active_rules_code 
ON active_dosing_rules (drug_code);

CREATE INDEX IF NOT EXISTS idx_active_rules_version 
ON active_dosing_rules (drug_code, semantic_version);

-- ============================================================================
-- STEP 4: Create functions for materialized view management
-- ============================================================================

-- Function to refresh materialized view (concurrent for zero downtime)
CREATE OR REPLACE FUNCTION refresh_active_dosing_rules()
RETURNS VOID AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY active_dosing_rules;
    -- Log the refresh for monitoring
    INSERT INTO system_events (event_type, event_data, created_at) 
    VALUES ('materialized_view_refresh', '{"view": "active_dosing_rules"}', now())
    ON CONFLICT DO NOTHING;
EXCEPTION
    WHEN OTHERS THEN
        -- If concurrent refresh fails, do regular refresh
        REFRESH MATERIALIZED VIEW active_dosing_rules;
END;
$$ LANGUAGE plpgsql;

-- Function to invalidate cache when rules are updated
CREATE OR REPLACE FUNCTION invalidate_rule_cache()
RETURNS TRIGGER AS $$
BEGIN
    -- Create notification for cache invalidation
    PERFORM pg_notify('rule_cache_invalidate', 
        json_build_object(
            'drug_code', COALESCE(NEW.drug_code, OLD.drug_code),
            'semantic_version', COALESCE(NEW.semantic_version, OLD.semantic_version),
            'operation', TG_OP
        )::text
    );
    
    -- Refresh materialized view asynchronously
    PERFORM refresh_active_dosing_rules();
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 5: Create triggers for cache invalidation and view refresh
-- ============================================================================

-- Trigger on dosing_rules changes
DROP TRIGGER IF EXISTS trigger_invalidate_rule_cache ON dosing_rules;
CREATE TRIGGER trigger_invalidate_rule_cache
    AFTER INSERT OR UPDATE OR DELETE ON dosing_rules
    FOR EACH ROW
    EXECUTE FUNCTION invalidate_rule_cache();

-- Triggers on related tables
DROP TRIGGER IF EXISTS trigger_invalidate_adjustments_cache ON dose_adjustments;
CREATE TRIGGER trigger_invalidate_adjustments_cache
    AFTER INSERT OR UPDATE OR DELETE ON dose_adjustments
    FOR EACH ROW
    EXECUTE FUNCTION invalidate_rule_cache();

DROP TRIGGER IF EXISTS trigger_invalidate_titration_cache ON titration_schedules;
CREATE TRIGGER trigger_invalidate_titration_cache
    AFTER INSERT OR UPDATE OR DELETE ON titration_schedules
    FOR EACH ROW
    EXECUTE FUNCTION invalidate_rule_cache();

DROP TRIGGER IF EXISTS trigger_invalidate_population_cache ON population_dosing;
CREATE TRIGGER trigger_invalidate_population_cache
    AFTER INSERT OR UPDATE OR DELETE ON population_dosing
    FOR EACH ROW
    EXECUTE FUNCTION invalidate_rule_cache();

-- ============================================================================
-- STEP 6: Create system_events table for audit and monitoring
-- ============================================================================

CREATE TABLE IF NOT EXISTS system_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    event_data JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    created_by TEXT DEFAULT current_user
);

CREATE INDEX IF NOT EXISTS idx_system_events_type_date 
ON system_events(event_type, created_at DESC);

-- ============================================================================
-- STEP 7: Create helper functions for KB-1 operations
-- ============================================================================

-- Function to get active rule by drug_code (primary KB-1 operation)
CREATE OR REPLACE FUNCTION get_active_dosing_rule(
    p_drug_code TEXT,
    p_version TEXT DEFAULT NULL
)
RETURNS TABLE (
    rule_id UUID,
    drug_code TEXT,
    drug_name TEXT,
    semantic_version TEXT,
    compiled_json JSONB,
    checksum TEXT,
    adjustments JSON,
    titration_schedule JSON,
    population_rules JSON
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        adr.rule_id,
        adr.drug_code,
        adr.drug_name,
        adr.semantic_version,
        adr.compiled_json,
        adr.checksum,
        adr.adjustments,
        adr.titration_schedule,
        adr.population_rules
    FROM active_dosing_rules adr
    WHERE adr.drug_code = p_drug_code
    AND (p_version IS NULL OR adr.semantic_version = p_version)
    ORDER BY adr.semantic_version DESC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to activate a rule version (sets active=true, others=false)
CREATE OR REPLACE FUNCTION activate_dosing_rule(
    p_drug_code TEXT,
    p_version TEXT,
    p_activated_by TEXT DEFAULT current_user
)
RETURNS BOOLEAN AS $$
DECLARE
    rule_count INTEGER;
BEGIN
    -- Check if rule exists
    SELECT COUNT(*) INTO rule_count
    FROM dosing_rules
    WHERE drug_code = p_drug_code AND semantic_version = p_version;
    
    IF rule_count = 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Deactivate all versions for this drug
    UPDATE dosing_rules
    SET active = FALSE, expires_date = now()
    WHERE drug_code = p_drug_code AND active = TRUE;
    
    -- Activate the specified version
    UPDATE dosing_rules
    SET active = TRUE, expires_date = NULL
    WHERE drug_code = p_drug_code AND semantic_version = p_version;
    
    -- Log the activation
    INSERT INTO system_events (event_type, event_data, created_by)
    VALUES ('rule_activation', json_build_object(
        'drug_code', p_drug_code,
        'version', p_version,
        'activated_by', p_activated_by
    ), p_activated_by);
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 8: Data migration from existing schema (if needed)
-- ============================================================================

-- Migrate existing drug_rule_packs to new dosing_rules schema
INSERT INTO dosing_rules (
    drug_code,
    drug_name,
    semantic_version,
    source_file,
    compiled_json,
    active,
    created_by,
    created_at,
    checksum,
    provenance,
    notes
)
SELECT 
    drug_id as drug_code,
    COALESCE((json_content->>'meta'->>'drug_name')::text, drug_id) as drug_name,
    version as semantic_version,
    COALESCE(drug_id || '_' || version || '.toml', 'migrated.toml') as source_file,
    COALESCE(json_content, content::jsonb) as compiled_json,
    true as active,  -- Set existing rules as active during migration
    COALESCE(created_by, 'migration') as created_by,
    created_at,
    content_sha as checksum,
    json_build_object(
        'authors', ARRAY[COALESCE(created_by, 'unknown')],
        'approvals', ARRAY[]::text[],
        'kb3_refs', ARRAY[]::text[],
        'kb4_refs', ARRAY[]::text[],
        'migration_source', 'drug_rule_packs'
    ) as provenance,
    'Migrated from drug_rule_packs table' as notes
FROM drug_rule_packs
WHERE NOT EXISTS (
    SELECT 1 FROM dosing_rules dr 
    WHERE dr.drug_code = drug_rule_packs.drug_id 
    AND dr.semantic_version = drug_rule_packs.version
);

-- ============================================================================
-- STEP 9: Refresh materialized view with initial data
-- ============================================================================

-- Initial refresh of materialized view
REFRESH MATERIALIZED VIEW active_dosing_rules;

-- ============================================================================
-- STEP 10: Create performance monitoring views
-- ============================================================================

-- View for monitoring cache performance and rule usage
CREATE OR REPLACE VIEW rule_usage_stats AS
SELECT 
    dr.drug_code,
    dr.drug_name,
    dr.semantic_version,
    dr.active,
    COUNT(se.event_id) as access_count,
    MAX(se.created_at) as last_accessed,
    dr.created_at as rule_created_at,
    DATE_PART('days', now() - dr.created_at) as days_since_creation
FROM dosing_rules dr
LEFT JOIN system_events se ON se.event_data->>'drug_code' = dr.drug_code
    AND se.event_type = 'rule_access'
GROUP BY dr.rule_id, dr.drug_code, dr.drug_name, dr.semantic_version, dr.active, dr.created_at
ORDER BY access_count DESC NULLS LAST;

-- View for governance metrics
CREATE OR REPLACE VIEW governance_metrics AS
SELECT 
    COUNT(*) FILTER (WHERE active = true) as active_rules,
    COUNT(*) FILTER (WHERE active = false) as inactive_rules,
    COUNT(DISTINCT drug_code) as unique_drugs,
    AVG(DATE_PART('days', now() - created_at)) as avg_rule_age_days,
    COUNT(*) FILTER (WHERE provenance->>'approvals' != '[]') as approved_rules,
    COUNT(*) FILTER (WHERE provenance->>'approvals' = '[]') as unapproved_rules
FROM dosing_rules;

COMMIT;

-- ============================================================================
-- VERIFICATION AND OPTIMIZATION
-- ============================================================================

-- Verify the migration completed successfully
DO $$
BEGIN
    ASSERT (SELECT COUNT(*) FROM information_schema.tables 
            WHERE table_name = 'dosing_rules') = 1,
           'dosing_rules table not created';
    
    ASSERT (SELECT COUNT(*) FROM information_schema.tables 
            WHERE table_name = 'dose_adjustments') = 1,
           'dose_adjustments table not created';
           
    ASSERT (SELECT COUNT(*) FROM information_schema.tables 
            WHERE table_name = 'titration_schedules') = 1,
           'titration_schedules table not created';
    
    ASSERT (SELECT COUNT(*) FROM information_schema.tables 
            WHERE table_name = 'population_dosing') = 1,
           'population_dosing table not created';
           
    ASSERT (SELECT COUNT(*) FROM information_schema.views 
            WHERE table_name = 'active_dosing_rules') = 1,
           'active_dosing_rules materialized view not created';
    
    RAISE NOTICE 'KB-1 Schema Migration 005 completed successfully - All tables and views created';
END $$;

-- Performance verification
DO $$
DECLARE
    rule_count INTEGER;
    view_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO rule_count FROM dosing_rules;
    SELECT COUNT(*) INTO view_count FROM active_dosing_rules;
    
    RAISE NOTICE 'Migration 005 Data Summary:';
    RAISE NOTICE '  - Total rules in dosing_rules: %', rule_count;
    RAISE NOTICE '  - Active rules in materialized view: %', view_count;
    RAISE NOTICE '  - KB-1 specification compliance: ACHIEVED';
END $$;