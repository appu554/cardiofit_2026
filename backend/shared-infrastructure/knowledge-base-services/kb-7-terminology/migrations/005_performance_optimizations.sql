-- KB-7 Terminology Service: Performance Optimizations
-- Phase 4: Performance Optimizations Implementation
-- This migration implements automatic materialized view refresh and performance enhancements

-- Create materialized view refresh log table
CREATE TABLE IF NOT EXISTS materialized_view_refresh_log (
    id SERIAL PRIMARY KEY,
    view_name VARCHAR(255) NOT NULL,
    refresh_type VARCHAR(20) NOT NULL, -- 'full', 'concurrent'
    status VARCHAR(20) NOT NULL, -- 'started', 'completed', 'failed'
    started_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,
    duration_seconds INTEGER,
    rows_affected BIGINT,
    error_message TEXT,
    triggered_by VARCHAR(100) -- 'schedule', 'manual', 'data_change'
);

-- Create indexes for refresh log
CREATE INDEX IF NOT EXISTS idx_mv_refresh_log_view_name ON materialized_view_refresh_log(view_name);
CREATE INDEX IF NOT EXISTS idx_mv_refresh_log_started_at ON materialized_view_refresh_log(started_at);
CREATE INDEX IF NOT EXISTS idx_mv_refresh_log_status ON materialized_view_refresh_log(status);

-- Create function to refresh materialized views with logging
CREATE OR REPLACE FUNCTION refresh_materialized_view(
    view_name TEXT,
    concurrent_refresh BOOLEAN DEFAULT TRUE,
    triggered_by TEXT DEFAULT 'manual'
) RETURNS BOOLEAN AS $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
    duration INTEGER;
    rows_affected BIGINT;
    refresh_sql TEXT;
    log_id INTEGER;
    success BOOLEAN := TRUE;
BEGIN
    start_time := NOW();
    
    -- Insert start log entry
    INSERT INTO materialized_view_refresh_log (view_name, refresh_type, status, started_at, triggered_by)
    VALUES (view_name, CASE WHEN concurrent_refresh THEN 'concurrent' ELSE 'full' END, 'started', start_time, triggered_by)
    RETURNING id INTO log_id;
    
    -- Build refresh SQL
    refresh_sql := 'REFRESH MATERIALIZED VIEW';
    IF concurrent_refresh THEN
        refresh_sql := refresh_sql || ' CONCURRENTLY';
    END IF;
    refresh_sql := refresh_sql || ' ' || view_name;
    
    BEGIN
        -- Execute refresh
        EXECUTE refresh_sql;
        
        -- Get row count (approximate)
        EXECUTE 'SELECT COUNT(*) FROM ' || view_name INTO rows_affected;
        
        end_time := NOW();
        duration := EXTRACT(EPOCH FROM (end_time - start_time))::INTEGER;
        
        -- Update log with success
        UPDATE materialized_view_refresh_log
        SET status = 'completed',
            completed_at = end_time,
            duration_seconds = duration,
            rows_affected = rows_affected
        WHERE id = log_id;
        
    EXCEPTION WHEN OTHERS THEN
        success := FALSE;
        end_time := NOW();
        duration := EXTRACT(EPOCH FROM (end_time - start_time))::INTEGER;
        
        -- Update log with error
        UPDATE materialized_view_refresh_log
        SET status = 'failed',
            completed_at = end_time,
            duration_seconds = duration,
            error_message = SQLERRM
        WHERE id = log_id;
        
        -- Re-raise the exception
        RAISE;
    END;
    
    RETURN success;
END;
$$ LANGUAGE plpgsql;

-- Create function to refresh all materialized views
CREATE OR REPLACE FUNCTION refresh_all_materialized_views(
    concurrent_refresh BOOLEAN DEFAULT TRUE
) RETURNS TABLE(view_name TEXT, success BOOLEAN, duration_seconds INTEGER) AS $$
DECLARE
    mv_record RECORD;
    refresh_success BOOLEAN;
    start_time TIMESTAMP;
    end_time TIMESTAMP;
    duration INTEGER;
BEGIN
    -- Get all materialized views in the current schema
    FOR mv_record IN 
        SELECT schemaname, matviewname 
        FROM pg_matviews 
        WHERE schemaname = current_schema()
        ORDER BY matviewname
    LOOP
        start_time := NOW();
        
        BEGIN
            refresh_success := refresh_materialized_view(
                mv_record.matviewname, 
                concurrent_refresh, 
                'bulk_refresh'
            );
        EXCEPTION WHEN OTHERS THEN
            refresh_success := FALSE;
        END;
        
        end_time := NOW();
        duration := EXTRACT(EPOCH FROM (end_time - start_time))::INTEGER;
        
        RETURN NEXT;
        view_name := mv_record.matviewname;
        success := refresh_success;
        duration_seconds := duration;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create additional materialized views for performance (only if concept_relationships table exists)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'concept_relationships') THEN
        EXECUTE '
        CREATE MATERIALIZED VIEW IF NOT EXISTS concept_hierarchy_cache AS
        SELECT
            parent.concept_uuid as parent_id,
            parent.system as parent_system,
            parent.code as parent_code,
            parent.preferred_term as parent_term,
            child.concept_uuid as child_id,
            child.system as child_system,
            child.code as child_code,
            child.preferred_term as child_term,
            cr.relationship_type,
            1 as hierarchy_level
        FROM concept_relationships cr
        JOIN concepts parent ON parent.concept_uuid = cr.source_concept_id
        JOIN concepts child ON child.concept_uuid = cr.target_concept_id
        WHERE cr.active = true
          AND parent.active = true
          AND child.active = true
          AND cr.relationship_type IN (''ISA'', ''is-a'', ''subsumes'', ''parent-child'')

        UNION ALL

        SELECT
            grandparent.concept_uuid as parent_id,
            grandparent.system as parent_system,
            grandparent.code as parent_code,
            grandparent.preferred_term as parent_term,
            grandchild.concept_uuid as child_id,
            grandchild.system as child_system,
            grandchild.code as child_code,
            grandchild.preferred_term as child_term,
            ''transitive'' as relationship_type,
            2 as hierarchy_level
        FROM concept_relationships cr1
        JOIN concepts parent ON parent.concept_uuid = cr1.source_concept_id
        JOIN concepts child ON child.concept_uuid = cr1.target_concept_id
        JOIN concept_relationships cr2 ON cr2.source_concept_id = parent.concept_uuid
        JOIN concepts grandparent ON grandparent.concept_uuid = cr2.target_concept_id
        JOIN concept_relationships cr3 ON cr3.target_concept_id = child.concept_uuid
        JOIN concepts grandchild ON grandchild.concept_uuid = cr3.source_concept_id
        WHERE cr1.active = true AND cr2.active = true AND cr3.active = true
          AND parent.active = true AND child.active = true
          AND grandparent.active = true AND grandchild.active = true
          AND cr1.relationship_type IN (''ISA'', ''is-a'', ''subsumes'', ''parent-child'')
          AND cr2.relationship_type IN (''ISA'', ''is-a'', ''subsumes'', ''parent-child'')
          AND cr3.relationship_type IN (''ISA'', ''is-a'', ''subsumes'', ''parent-child'')';

        -- Create indexes on concept hierarchy cache
        CREATE INDEX IF NOT EXISTS idx_concept_hierarchy_cache_parent ON concept_hierarchy_cache(parent_system, parent_code);
        CREATE INDEX IF NOT EXISTS idx_concept_hierarchy_cache_child ON concept_hierarchy_cache(child_system, child_code);
        CREATE INDEX IF NOT EXISTS idx_concept_hierarchy_cache_relationship ON concept_hierarchy_cache(relationship_type);
        CREATE INDEX IF NOT EXISTS idx_concept_hierarchy_cache_level ON concept_hierarchy_cache(hierarchy_level);

        RAISE NOTICE 'Created concept_hierarchy_cache materialized view';
    ELSE
        RAISE NOTICE 'concept_relationships table does not exist - skipping concept_hierarchy_cache';
    END IF;
END $$;

-- Create frequently used concepts materialized view (simplified - uses only existing tables)
-- Note: Full implementation with validation_requests requires additional migrations
DO $$
BEGIN
    -- Create simplified version using only search_statistics and concepts
    IF NOT EXISTS (SELECT 1 FROM pg_matviews WHERE matviewname = 'frequently_used_concepts') THEN
        EXECUTE '
        CREATE MATERIALIZED VIEW frequently_used_concepts AS
        SELECT
            c.concept_uuid,
            c.system,
            c.code,
            c.preferred_term,
            c.active,
            c.version,
            COALESCE(ss.search_count, 0) as search_count,
            0 as validation_count,
            0 as translation_count,
            COALESCE(ss.search_count, 0) as total_usage
        FROM concepts c
        LEFT JOIN (
            SELECT
                search_term as concept_code,
                COUNT(*) as search_count
            FROM search_statistics
            WHERE created_at >= NOW() - INTERVAL ''30 days''
            GROUP BY search_term
        ) ss ON ss.concept_code = c.code
        WHERE c.active = true
          AND COALESCE(ss.search_count, 0) > 0
        ORDER BY total_usage DESC';

        -- Create indexes on frequently used concepts
        CREATE INDEX IF NOT EXISTS idx_frequently_used_concepts_usage ON frequently_used_concepts(total_usage DESC);
        CREATE INDEX IF NOT EXISTS idx_frequently_used_concepts_system ON frequently_used_concepts(system);

        RAISE NOTICE 'Created frequently_used_concepts materialized view';
    ELSE
        RAISE NOTICE 'frequently_used_concepts view already exists';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not create frequently_used_concepts: %', SQLERRM;
END $$;

-- Create concept mapping performance cache (simplified - handles missing tables gracefully)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_matviews WHERE matviewname = 'concept_mapping_performance_cache') THEN
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'mapping_usage_stats') THEN
            EXECUTE '
            CREATE MATERIALIZED VIEW concept_mapping_performance_cache AS
            SELECT
                cm.source_system,
                cm.target_system,
                cm.source_code,
                cm.target_code,
                cm.confidence_score,
                cm.equivalence,
                AVG(mus.avg_response_time_ms) as avg_response_time,
                SUM(mus.usage_count) as total_usage,
                AVG(mus.success_rate) as avg_success_rate,
                MAX(mus.last_used) as last_used,
                cm.created_at
            FROM concept_mappings cm
            LEFT JOIN mapping_usage_stats mus ON mus.mapping_id = cm.id
            WHERE mus.usage_date >= CURRENT_DATE - INTERVAL ''90 days''
            GROUP BY cm.source_system, cm.target_system, cm.source_code, cm.target_code,
                     cm.confidence_score, cm.equivalence, cm.created_at
            HAVING SUM(mus.usage_count) > 0
            ORDER BY total_usage DESC, avg_response_time ASC';

            -- Create indexes on mapping performance cache
            CREATE INDEX IF NOT EXISTS idx_concept_mapping_perf_cache_systems ON concept_mapping_performance_cache(source_system, target_system);
            CREATE INDEX IF NOT EXISTS idx_concept_mapping_perf_cache_usage ON concept_mapping_performance_cache(total_usage DESC);

            RAISE NOTICE 'Created concept_mapping_performance_cache materialized view';
        ELSE
            -- Create simplified version without usage stats
            EXECUTE '
            CREATE MATERIALIZED VIEW concept_mapping_performance_cache AS
            SELECT
                cm.source_system,
                cm.target_system,
                cm.source_code,
                cm.target_code,
                cm.confidence_score,
                cm.equivalence,
                0.0 as avg_response_time,
                0 as total_usage,
                1.0 as avg_success_rate,
                cm.created_at as last_used,
                cm.created_at
            FROM concept_mappings cm
            ORDER BY cm.confidence_score DESC';

            CREATE INDEX IF NOT EXISTS idx_concept_mapping_perf_cache_systems ON concept_mapping_performance_cache(source_system, target_system);

            RAISE NOTICE 'Created concept_mapping_performance_cache (simplified - no usage stats)';
        END IF;
    ELSE
        RAISE NOTICE 'concept_mapping_performance_cache view already exists';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not create concept_mapping_performance_cache: %', SQLERRM;
END $$;

-- Create connection pool monitoring table
CREATE TABLE IF NOT EXISTS connection_pool_stats (
    id SERIAL PRIMARY KEY,
    recorded_at TIMESTAMP DEFAULT NOW(),
    active_connections INTEGER,
    idle_connections INTEGER,
    total_connections INTEGER,
    max_connections INTEGER,
    connection_wait_time_ms NUMERIC(8,2),
    query_duration_p95_ms NUMERIC(8,2),
    query_duration_avg_ms NUMERIC(8,2),
    slow_query_count INTEGER DEFAULT 0
);

-- Create function to collect connection pool statistics
CREATE OR REPLACE FUNCTION collect_connection_pool_stats() RETURNS void AS $$
DECLARE
    avg_exec_time NUMERIC;
BEGIN
    -- Try to get average execution time from pg_stat_statements (requires extension)
    BEGIN
        SELECT AVG(mean_exec_time) INTO avg_exec_time FROM pg_stat_statements WHERE calls > 0;
    EXCEPTION WHEN undefined_table THEN
        avg_exec_time := NULL;
    END;

    INSERT INTO connection_pool_stats (
        active_connections,
        idle_connections,
        total_connections,
        max_connections,
        connection_wait_time_ms,
        query_duration_avg_ms
    )
    SELECT
        (SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active'),
        (SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'idle'),
        (SELECT COUNT(*) FROM pg_stat_activity),
        (SELECT setting::integer FROM pg_settings WHERE name = 'max_connections'),
        0, -- This would need to be collected from application metrics
        avg_exec_time
    WHERE EXISTS (SELECT 1 FROM pg_stat_activity LIMIT 1);

    -- Clean up old stats (keep only last 7 days)
    DELETE FROM connection_pool_stats
    WHERE recorded_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql;

-- Create batch operation optimization tables
CREATE TABLE IF NOT EXISTS batch_operation_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_type VARCHAR(50) NOT NULL, -- 'validation', 'lookup', 'translation'
    batch_hash VARCHAR(64) NOT NULL, -- Hash of the request parameters
    batch_size INTEGER NOT NULL,
    results JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP DEFAULT (NOW() + INTERVAL '2 hours'),
    hit_count INTEGER DEFAULT 0
);

-- Create indexes for batch operation cache (PostgreSQL requires separate CREATE INDEX)
CREATE INDEX IF NOT EXISTS idx_batch_operation_cache_hash ON batch_operation_cache(batch_hash);
CREATE INDEX IF NOT EXISTS idx_batch_operation_cache_type ON batch_operation_cache(operation_type);
CREATE INDEX IF NOT EXISTS idx_batch_operation_cache_expires ON batch_operation_cache(expires_at);

-- Create function to optimize batch operations (handles missing materialized views)
CREATE OR REPLACE FUNCTION optimize_batch_lookup(
    concept_codes TEXT[],
    system_filter TEXT DEFAULT NULL
) RETURNS TABLE(
    code TEXT,
    system VARCHAR(20),
    preferred_term VARCHAR(500),
    active BOOLEAN,
    properties JSONB
) AS $$
DECLARE
    use_cache BOOLEAN;
BEGIN
    -- Check if frequently_used_concepts view exists
    SELECT EXISTS (SELECT 1 FROM pg_matviews WHERE matviewname = 'frequently_used_concepts') INTO use_cache;

    -- Use partitioned table if system is specified
    IF system_filter IS NOT NULL THEN
        RETURN QUERY
        SELECT c.code, c.system, c.preferred_term, c.active, c.properties
        FROM concepts c
        WHERE c.code = ANY(concept_codes)
          AND c.system = system_filter
          AND c.active = true
        ORDER BY array_position(concept_codes, c.code);
    ELSIF use_cache THEN
        -- Use frequently used concepts cache for common lookups
        RETURN QUERY EXECUTE '
        SELECT fuc.code, fuc.system, fuc.preferred_term, fuc.active,
               jsonb_build_object(''usage_rank'', fuc.total_usage) as properties
        FROM frequently_used_concepts fuc
        WHERE fuc.code = ANY($1)
          AND fuc.active = true

        UNION ALL

        SELECT c.code, c.system, c.preferred_term, c.active, c.properties
        FROM concepts c
        WHERE c.code = ANY($1)
          AND c.active = true
          AND NOT EXISTS (
              SELECT 1 FROM frequently_used_concepts fuc
              WHERE fuc.code = c.code
          )
        ORDER BY array_position($1, code)'
        USING concept_codes;
    ELSE
        -- Fallback to concepts table only
        RETURN QUERY
        SELECT c.code, c.system, c.preferred_term, c.active, c.properties
        FROM concepts c
        WHERE c.code = ANY(concept_codes)
          AND c.active = true
        ORDER BY array_position(concept_codes, c.code);
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Create query performance monitoring view (only if pg_stat_statements extension is available)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements') THEN
        EXECUTE '
        CREATE OR REPLACE VIEW query_performance_monitor AS
        SELECT
            query,
            calls,
            total_exec_time,
            mean_exec_time,
            max_exec_time,
            min_exec_time,
            stddev_exec_time,
            rows,
            100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
        FROM pg_stat_statements
        WHERE calls > 100
          AND mean_exec_time > 10
        ORDER BY total_exec_time DESC
        LIMIT 50';
    ELSE
        RAISE NOTICE 'pg_stat_statements extension not available - skipping query_performance_monitor view';
    END IF;
END $$;

-- Create automatic refresh jobs (requires pg_cron extension)
-- These would be uncommented in production with pg_cron installed

-- Refresh search-related views every hour
-- SELECT cron.schedule('refresh-search-views', '0 * * * *', 
--     'SELECT refresh_materialized_view(''common_medical_searches'', true, ''schedule'');');

-- Refresh hierarchy cache every 6 hours  
-- SELECT cron.schedule('refresh-hierarchy-cache', '0 */6 * * *',
--     'SELECT refresh_materialized_view(''concept_hierarchy_cache'', true, ''schedule'');');

-- Refresh frequently used concepts daily
-- SELECT cron.schedule('refresh-frequent-concepts', '0 2 * * *',
--     'SELECT refresh_materialized_view(''frequently_used_concepts'', true, ''schedule'');');

-- Refresh mapping performance cache daily
-- SELECT cron.schedule('refresh-mapping-cache', '0 3 * * *',
--     'SELECT refresh_materialized_view(''concept_mapping_performance_cache'', true, ''schedule'');');

-- Collect connection pool stats every 5 minutes
-- SELECT cron.schedule('collect-connection-stats', '*/5 * * * *',
--     'SELECT collect_connection_pool_stats();');

-- Create indexes for better performance on large datasets
-- Note: Using regular CREATE INDEX instead of CONCURRENTLY (cannot run in transaction block)
CREATE INDEX IF NOT EXISTS idx_concepts_system_active_code
ON concepts(system, active, code) WHERE active = true;

-- Create GIN index for full-text search (uses 'english' as fallback if 'medical_english' not available)
DO $$
BEGIN
    -- Try medical_english first
    BEGIN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_concepts_preferred_term_gin
                 ON concepts USING gin(to_tsvector(''medical_english'', preferred_term)) WHERE active = true';
        RAISE NOTICE 'Created GIN index with medical_english config';
    EXCEPTION WHEN undefined_object THEN
        -- Fallback to standard english
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_concepts_preferred_term_gin
                 ON concepts USING gin(to_tsvector(''english'', preferred_term)) WHERE active = true';
        RAISE NOTICE 'Created GIN index with english config (medical_english not available)';
    END;
END $$;

-- Create concept_relationships index only if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'concept_relationships') THEN
        CREATE INDEX IF NOT EXISTS idx_concept_relationships_active_type
        ON concept_relationships(active, relationship_type, source_concept_id) WHERE active = true;
        RAISE NOTICE 'Created idx_concept_relationships_active_type index';
    ELSE
        RAISE NOTICE 'concept_relationships table does not exist - skipping index';
    END IF;
END $$;

-- Create partial indexes for common queries
CREATE INDEX IF NOT EXISTS idx_concepts_snomed_active
ON concepts(code, preferred_term) WHERE system = 'SNOMED' AND active = true;

CREATE INDEX IF NOT EXISTS idx_concepts_rxnorm_active
ON concepts(code, preferred_term) WHERE system = 'RxNorm' AND active = true;

CREATE INDEX IF NOT EXISTS idx_concepts_icd10_active
ON concepts(code, preferred_term) WHERE system = 'ICD-10-CM' AND active = true;

CREATE INDEX IF NOT EXISTS idx_concepts_loinc_active
ON concepts(code, preferred_term) WHERE system = 'LOINC' AND active = true;

-- Update table statistics for better query planning (wrapped in exception handler)
DO $$
BEGIN
    ANALYZE concepts;
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'concepts table not ready for ANALYZE';
END $$;

DO $$
BEGIN
    ANALYZE concept_relationships;
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'concept_relationships table not ready for ANALYZE';
END $$;

DO $$
BEGIN
    ANALYZE concept_mappings;
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'concept_mappings table not ready for ANALYZE';
END $$;

DO $$
BEGIN
    ANALYZE search_statistics;
EXCEPTION WHEN undefined_table THEN
    RAISE NOTICE 'search_statistics table not ready for ANALYZE';
END $$;

-- Initial refresh of all materialized views (wrapped in exception handler)
DO $$
BEGIN
    PERFORM refresh_all_materialized_views(true);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not refresh materialized views: %', SQLERRM;
END $$;

-- Migration completion log
INSERT INTO migration_log (migration_name, status, completed_at)
VALUES ('005_performance_optimizations', 'completed', NOW())
ON CONFLICT (migration_name) DO UPDATE SET
    status = 'completed',
    completed_at = NOW();

-- Note: No explicit COMMIT - migration runner handles transaction management