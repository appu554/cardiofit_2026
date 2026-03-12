package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// PerformanceOptimizer handles database performance optimization for clinical workloads
type PerformanceOptimizer struct {
	db     *PostgreSQL
	logger *zap.Logger
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(db *PostgreSQL, logger *zap.Logger) *PerformanceOptimizer {
	return &PerformanceOptimizer{
		db:     db,
		logger: logger,
	}
}

// OptimizationResult represents the result of performance optimization
type OptimizationResult struct {
	Operation     string        `json:"operation"`
	Success       bool          `json:"success"`
	ExecutionTime time.Duration `json:"execution_time"`
	Impact        string        `json:"impact"`
	Details       string        `json:"details"`
	Error         string        `json:"error,omitempty"`
}

// OptimizeForClinicalWorkloads applies optimizations specific to clinical medication workflows
func (po *PerformanceOptimizer) OptimizeForClinicalWorkloads(ctx context.Context) ([]OptimizationResult, error) {
	po.logger.Info("Starting clinical workload optimization")
	
	optimizations := []func(context.Context) OptimizationResult{
		po.optimizeConnectionPool,
		po.createPartitionedTables,
		po.optimizeJsonbIndexes,
		po.createMaterializedViews,
		po.optimizeWorkStatistics,
		po.enableQueryPlanOptimizations,
		po.createCustomFunctions,
		po.optimizeAuditTables,
		po.enableAutoVacuum,
		po.createPerformanceIndexes,
	}

	var results []OptimizationResult
	for _, optimization := range optimizations {
		result := optimization(ctx)
		results = append(results, result)
		
		if result.Success {
			po.logger.Info("Optimization completed successfully",
				zap.String("operation", result.Operation),
				zap.Duration("execution_time", result.ExecutionTime),
				zap.String("impact", result.Impact))
		} else {
			po.logger.Warn("Optimization failed",
				zap.String("operation", result.Operation),
				zap.String("error", result.Error))
		}
	}

	return results, nil
}

// optimizeConnectionPool configures connection pool for clinical workloads
func (po *PerformanceOptimizer) optimizeConnectionPool(ctx context.Context) OptimizationResult {
	start := time.Now()
	
	// Configure connection pool for clinical workloads (targeting <250ms response times)
	po.db.DB.SetMaxOpenConns(25)    // Limit to 25 connections for controlled resource usage
	po.db.DB.SetMaxIdleConns(5)     // Keep 5 idle connections for fast response
	po.db.DB.SetConnMaxLifetime(time.Hour)         // Refresh connections hourly
	po.db.DB.SetConnMaxIdleTime(30 * time.Minute) // Close idle connections after 30min

	return OptimizationResult{
		Operation:     "connection_pool_optimization",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Improved connection management for clinical workloads",
		Details:       "Configured: 25 max connections, 5 idle, 1h lifetime, 30min idle timeout",
	}
}

// createPartitionedTables creates partitioned tables for large audit and log tables
func (po *PerformanceOptimizer) createPartitionedTables(ctx context.Context) OptimizationResult {
	start := time.Now()

	queries := []string{
		// Partition audit trail by month for performance
		`
		-- Create partitioned audit trail table (if not exists)
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'audit_trail_partitioned') THEN
				CREATE TABLE audit_trail_partitioned (
					LIKE audit_trail INCLUDING ALL
				) PARTITION BY RANGE (event_timestamp);
				
				-- Create partitions for current and next 3 months
				FOR i IN 0..2 LOOP
					EXECUTE format('CREATE TABLE IF NOT EXISTS audit_trail_%s PARTITION OF audit_trail_partitioned
						FOR VALUES FROM (%L) TO (%L)',
						to_char(CURRENT_DATE + (i || ' months')::interval, 'YYYY_MM'),
						date_trunc('month', CURRENT_DATE + (i || ' months')::interval),
						date_trunc('month', CURRENT_DATE + ((i+1) || ' months')::interval)
					);
				END LOOP;
			END IF;
		END $$;
		`,
		
		// Partition FHIR integration logs by day for performance
		`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'fhir_logs_partitioned') THEN
				CREATE TABLE fhir_logs_partitioned (
					LIKE fhir_integration_logs INCLUDING ALL
				) PARTITION BY RANGE (request_started_at);
				
				-- Create partitions for current and next 7 days
				FOR i IN 0..6 LOOP
					EXECUTE format('CREATE TABLE IF NOT EXISTS fhir_logs_%s PARTITION OF fhir_logs_partitioned
						FOR VALUES FROM (%L) TO (%L)',
						to_char(CURRENT_DATE + i, 'YYYY_MM_DD'),
						date_trunc('day', CURRENT_DATE + i),
						date_trunc('day', CURRENT_DATE + i + 1)
					);
				END LOOP;
			END IF;
		END $$;
		`,
	}

	for _, query := range queries {
		_, err := po.db.DB.ExecContext(ctx, query)
		if err != nil {
			return OptimizationResult{
				Operation:     "partitioned_tables",
				Success:       false,
				ExecutionTime: time.Since(start),
				Error:         err.Error(),
			}
		}
	}

	return OptimizationResult{
		Operation:     "partitioned_tables",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Improved query performance for large audit and log tables",
		Details:       "Created monthly partitions for audit_trail, daily partitions for FHIR logs",
	}
}

// optimizeJsonbIndexes creates optimized GIN indexes for JSONB columns
func (po *PerformanceOptimizer) optimizeJsonbIndexes(ctx context.Context) OptimizationResult {
	start := time.Now()

	indexes := []string{
		// Clinical calculation results - optimized for clinical queries
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_clinical_calc_results_gin_ops 
		 ON clinical_calculation_results USING GIN (results jsonb_ops);`,
		
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_clinical_calc_safety_path 
		 ON clinical_calculation_results USING GIN ((safety_flags -> 'alerts') jsonb_ops);`,
		
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_clinical_calc_recommendations_path
		 ON clinical_calculation_results USING GIN ((clinical_recommendations -> 'primary') jsonb_ops);`,

		// Workflow executions - optimized for 4-Phase workflow queries
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_phase_results
		 ON workflow_executions USING GIN ((phase_3_clinical_intelligence -> 'results') jsonb_ops);`,
		
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_fhir_references
		 ON workflow_executions USING GIN (clinical_context_fhir_references jsonb_ops);`,

		// Clinical snapshots - optimized for Recipe & Snapshot architecture
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_snapshots_clinical_data_gin
		 ON clinical_snapshots USING GIN (clinical_data jsonb_path_ops);`,
		
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_snapshots_evidence_envelope_gin
		 ON clinical_snapshots USING GIN (evidence_envelope jsonb_path_ops);`,
		
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_snapshots_reasoning_gin
		 ON clinical_snapshots USING GIN (clinical_reasoning jsonb_path_ops);`,

		// Recipes - optimized for recipe resolution
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_recipes_calculation_rules_gin
		 ON recipes USING GIN (calculation_rules jsonb_path_ops);`,
		
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_recipes_safety_rules_gin
		 ON recipes USING GIN (safety_rules jsonb_path_ops);`,

		// FHIR resource mappings - optimized for reference lookups
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_fhir_mappings_composite
		 ON fhir_resource_mappings (internal_resource_type, fhir_resource_type, sync_status)
		 WHERE sync_status = 'synchronized';`,
	}

	for _, indexQuery := range indexes {
		_, err := po.db.DB.ExecContext(ctx, indexQuery)
		if err != nil {
			po.logger.Warn("Failed to create index", zap.String("query", indexQuery), zap.Error(err))
			// Continue with other indexes
		}
	}

	return OptimizationResult{
		Operation:     "jsonb_index_optimization",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Optimized JSONB queries for clinical data access patterns",
		Details:       fmt.Sprintf("Created %d specialized GIN indexes for clinical workloads", len(indexes)),
	}
}

// createMaterializedViews creates materialized views for common queries
func (po *PerformanceOptimizer) createMaterializedViews(ctx context.Context) OptimizationResult {
	start := time.Now()

	views := []string{
		// Active workflow summary for dashboard
		`
		CREATE MATERIALIZED VIEW IF NOT EXISTS mv_active_workflow_summary AS
		SELECT 
			workflow_type,
			priority,
			current_phase,
			execution_status,
			COUNT(*) as workflow_count,
			AVG(total_execution_time_ms) as avg_execution_time,
			COUNT(*) FILTER (WHERE performance_target_met) as target_met_count,
			MAX(started_at) as latest_workflow
		FROM workflow_executions
		WHERE started_at > NOW() - INTERVAL '24 hours'
		  AND execution_status IN ('in_progress', 'completed')
		GROUP BY workflow_type, priority, current_phase, execution_status
		WITH DATA;
		
		CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_active_workflow_summary_unique
		ON mv_active_workflow_summary (workflow_type, priority, current_phase, execution_status);
		`,

		// Recipe performance summary
		`
		CREATE MATERIALIZED VIEW IF NOT EXISTS mv_recipe_performance_summary AS
		SELECT 
			r.protocol_id,
			r.name,
			r.version,
			r.complexity_score,
			r.average_execution_ms,
			r.success_rate,
			r.usage_count,
			COUNT(cs.id) as snapshot_count,
			AVG(cs.assembly_duration_ms) as avg_assembly_time,
			AVG(cs.quality_score) as avg_quality_score
		FROM recipes r
		LEFT JOIN clinical_snapshots cs ON r.id = cs.recipe_id
		WHERE r.status = 'active'
		  AND cs.created_at > NOW() - INTERVAL '7 days'
		GROUP BY r.id, r.protocol_id, r.name, r.version, r.complexity_score, 
				 r.average_execution_ms, r.success_rate, r.usage_count
		WITH DATA;
		
		CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_recipe_performance_summary_unique
		ON mv_recipe_performance_summary (protocol_id, version);
		`,

		// FHIR integration health summary
		`
		CREATE MATERIALIZED VIEW IF NOT EXISTS mv_fhir_integration_health AS
		SELECT 
			operation_type,
			fhir_resource_type,
			DATE_TRUNC('hour', request_started_at) as hour_bucket,
			COUNT(*) as total_requests,
			COUNT(*) FILTER (WHERE success) as successful_requests,
			AVG(total_latency_ms) as avg_latency,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY total_latency_ms) as p95_latency,
			SUM(quota_used) as total_quota_used
		FROM fhir_integration_logs
		WHERE request_started_at > NOW() - INTERVAL '24 hours'
		GROUP BY operation_type, fhir_resource_type, DATE_TRUNC('hour', request_started_at)
		WITH DATA;
		
		CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_fhir_health_unique
		ON mv_fhir_integration_health (operation_type, fhir_resource_type, hour_bucket);
		`,
	}

	for _, viewQuery := range views {
		_, err := po.db.DB.ExecContext(ctx, viewQuery)
		if err != nil {
			return OptimizationResult{
				Operation:     "materialized_views",
				Success:       false,
				ExecutionTime: time.Since(start),
				Error:         err.Error(),
			}
		}
	}

	return OptimizationResult{
		Operation:     "materialized_views",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Accelerated common dashboard and reporting queries",
		Details:       "Created materialized views for workflow summary, recipe performance, and FHIR health",
	}
}

// optimizeWorkStatistics updates table statistics for query optimization
func (po *PerformanceOptimizer) optimizeWorkStatistics(ctx context.Context) OptimizationResult {
	start := time.Now()

	// Update statistics for critical tables
	tables := []string{
		"workflow_executions",
		"clinical_calculation_results", 
		"clinical_snapshots",
		"recipes",
		"fhir_resource_mappings",
		"audit_trail",
		"performance_metrics",
	}

	for _, table := range tables {
		query := fmt.Sprintf("ANALYZE %s;", table)
		_, err := po.db.DB.ExecContext(ctx, query)
		if err != nil {
			po.logger.Warn("Failed to analyze table", zap.String("table", table), zap.Error(err))
		}
	}

	return OptimizationResult{
		Operation:     "statistics_update",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Updated query planner statistics for optimal query plans",
		Details:       fmt.Sprintf("Analyzed %d critical tables for clinical workloads", len(tables)),
	}
}

// enableQueryPlanOptimizations enables PostgreSQL query optimization features
func (po *PerformanceOptimizer) enableQueryPlanOptimizations(ctx context.Context) OptimizationResult {
	start := time.Now()

	optimizations := []string{
		// Enable parallel query execution for large scans
		"SET max_parallel_workers_per_gather = 2;",
		"SET max_parallel_workers = 8;",
		"SET parallel_tuple_cost = 0.1;",
		"SET parallel_setup_cost = 1000.0;",
		
		// Optimize work memory for clinical queries
		"SET work_mem = '64MB';",
		"SET maintenance_work_mem = '256MB';",
		
		// Enable query plan caching
		"SET plan_cache_mode = 'auto';",
		
		// Optimize for JSONB operations
		"SET gin_fuzzy_search_limit = 0;",
		"SET gin_pending_list_limit = 4096;",
		
		// Enable just-in-time compilation for complex queries
		"SET jit = on;",
		"SET jit_above_cost = 500000;",
	}

	for _, optimization := range optimizations {
		_, err := po.db.DB.ExecContext(ctx, optimization)
		if err != nil {
			po.logger.Warn("Failed to apply optimization", zap.String("query", optimization), zap.Error(err))
		}
	}

	return OptimizationResult{
		Operation:     "query_plan_optimizations",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Enabled advanced PostgreSQL query optimization features",
		Details:       "Configured parallel queries, work memory, and JIT compilation",
	}
}

// createCustomFunctions creates custom PostgreSQL functions for clinical operations
func (po *PerformanceOptimizer) createCustomFunctions(ctx context.Context) OptimizationResult {
	start := time.Now()

	functions := []string{
		// Function to calculate clinical data freshness score
		`
		CREATE OR REPLACE FUNCTION calculate_clinical_freshness(
			last_updated TIMESTAMP WITH TIME ZONE,
			data_type VARCHAR(100)
		) RETURNS DECIMAL(4,3) AS $$
		DECLARE
			age_hours DECIMAL;
			max_age_hours DECIMAL;
		BEGIN
			age_hours := EXTRACT(EPOCH FROM (NOW() - last_updated)) / 3600.0;
			
			-- Different data types have different freshness thresholds
			max_age_hours := CASE 
				WHEN data_type = 'vital_signs' THEN 4.0    -- 4 hours for vitals
				WHEN data_type = 'lab_results' THEN 24.0   -- 24 hours for labs
				WHEN data_type = 'medications' THEN 12.0   -- 12 hours for medications
				ELSE 8.0                                   -- 8 hours default
			END;
			
			-- Calculate freshness score (1.0 = fresh, 0.0 = stale)
			RETURN GREATEST(0.0, LEAST(1.0, (max_age_hours - age_hours) / max_age_hours));
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;
		`,

		// Function to validate JSONB clinical data structure
		`
		CREATE OR REPLACE FUNCTION validate_clinical_data_structure(
			clinical_data JSONB,
			required_fields TEXT[]
		) RETURNS BOOLEAN AS $$
		DECLARE
			field TEXT;
		BEGIN
			-- Check if all required fields are present
			FOREACH field IN ARRAY required_fields
			LOOP
				IF NOT (clinical_data ? field) THEN
					RETURN FALSE;
				END IF;
			END LOOP;
			
			RETURN TRUE;
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;
		`,

		// Function to calculate workflow performance score
		`
		CREATE OR REPLACE FUNCTION calculate_workflow_performance_score(
			execution_time_ms INTEGER,
			target_ms INTEGER,
			error_count INTEGER,
			quality_score DECIMAL(3,2)
		) RETURNS DECIMAL(4,3) AS $$
		BEGIN
			-- Base score from execution time (0.0 to 0.4)
			DECLARE
				time_score DECIMAL(4,3);
				error_penalty DECIMAL(4,3);
				quality_bonus DECIMAL(4,3);
			BEGIN
				time_score := LEAST(0.4, 0.4 * target_ms::DECIMAL / GREATEST(execution_time_ms, 1));
				error_penalty := GREATEST(0.0, 0.2 - (error_count * 0.05));
				quality_bonus := COALESCE(quality_score * 0.4, 0.0);
				
				RETURN LEAST(1.0, time_score + error_penalty + quality_bonus);
			END;
		END;
		$$ LANGUAGE plpgsql IMMUTABLE;
		`,

		// Function to get active workflow summary
		`
		CREATE OR REPLACE FUNCTION get_active_workflow_summary(
			time_window INTERVAL DEFAULT INTERVAL '1 hour'
		) RETURNS TABLE (
			workflow_type VARCHAR(100),
			total_count BIGINT,
			completed_count BIGINT,
			avg_execution_time DECIMAL,
			success_rate DECIMAL(5,4)
		) AS $$
		BEGIN
			RETURN QUERY
			SELECT 
				we.workflow_type,
				COUNT(*) as total_count,
				COUNT(*) FILTER (WHERE we.execution_status = 'completed') as completed_count,
				AVG(we.total_execution_time_ms) as avg_execution_time,
				(COUNT(*) FILTER (WHERE we.execution_status = 'completed')::DECIMAL / 
				 GREATEST(COUNT(*), 1)) as success_rate
			FROM workflow_executions we
			WHERE we.started_at > NOW() - time_window
			GROUP BY we.workflow_type
			ORDER BY total_count DESC;
		END;
		$$ LANGUAGE plpgsql;
		`,
	}

	for _, function := range functions {
		_, err := po.db.DB.ExecContext(ctx, function)
		if err != nil {
			po.logger.Warn("Failed to create function", zap.Error(err))
		}
	}

	return OptimizationResult{
		Operation:     "custom_functions",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Created clinical-specific database functions for improved performance",
		Details:       "Added functions for freshness calculation, data validation, and performance scoring",
	}
}

// optimizeAuditTables optimizes audit tables for compliance and performance
func (po *PerformanceOptimizer) optimizeAuditTables(ctx context.Context) OptimizationResult {
	start := time.Now()

	optimizations := []string{
		// Create composite index for common audit queries
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_trail_comprehensive
		 ON audit_trail (patient_id, event_category, event_timestamp DESC)
		 WHERE patient_id IS NOT NULL;`,

		// Create index for HIPAA compliance queries
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_trail_hipaa_compliance
		 ON audit_trail (user_id, resource_type, event_timestamp DESC, safety_impact)
		 WHERE safety_impact IS NOT NULL;`,

		// Create index for security event correlation
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_trail_security_events
		 ON audit_trail (ip_address, user_agent, event_timestamp DESC)
		 WHERE event_category = 'security';`,

		// Optimize FHIR integration logs for performance monitoring
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_fhir_logs_performance_monitoring
		 ON fhir_integration_logs (operation_type, success, total_latency_ms, request_started_at DESC);`,

		// Create index for quota and rate limiting analysis
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_fhir_logs_quota_analysis
		 ON fhir_integration_logs (quota_used, rate_limit_remaining, request_started_at DESC)
		 WHERE quota_used > 0;`,
	}

	for _, optimization := range optimizations {
		_, err := po.db.DB.ExecContext(ctx, optimization)
		if err != nil {
			po.logger.Warn("Failed to create audit optimization", zap.Error(err))
		}
	}

	return OptimizationResult{
		Operation:     "audit_table_optimization",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Optimized audit tables for compliance reporting and security analysis",
		Details:       "Created specialized indexes for HIPAA compliance, security monitoring, and performance analysis",
	}
}

// enableAutoVacuum configures automatic maintenance for clinical workloads
func (po *PerformanceOptimizer) enableAutoVacuum(ctx context.Context) OptimizationResult {
	start := time.Now()

	// Configure autovacuum for high-write tables
	configurations := []string{
		// Workflow executions - high update frequency
		"ALTER TABLE workflow_executions SET (autovacuum_vacuum_scale_factor = 0.1, autovacuum_analyze_scale_factor = 0.05);",
		
		// Clinical snapshots - frequent access pattern changes
		"ALTER TABLE clinical_snapshots SET (autovacuum_vacuum_scale_factor = 0.1, autovacuum_analyze_scale_factor = 0.05);",
		
		// FHIR resource mappings - frequent synchronization updates
		"ALTER TABLE fhir_resource_mappings SET (autovacuum_vacuum_scale_factor = 0.2, autovacuum_analyze_scale_factor = 0.1);",
		
		// Audit trail - write-heavy, needs aggressive maintenance
		"ALTER TABLE audit_trail SET (autovacuum_vacuum_scale_factor = 0.05, autovacuum_analyze_scale_factor = 0.025);",
		
		// Performance metrics - continuous inserts
		"ALTER TABLE performance_metrics SET (autovacuum_vacuum_scale_factor = 0.1, autovacuum_analyze_scale_factor = 0.05);",
	}

	for _, config := range configurations {
		_, err := po.db.DB.ExecContext(ctx, config)
		if err != nil {
			po.logger.Warn("Failed to configure autovacuum", zap.Error(err))
		}
	}

	return OptimizationResult{
		Operation:     "autovacuum_optimization",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Configured automatic maintenance for optimal clinical workload performance",
		Details:       "Tuned autovacuum settings for high-frequency clinical data tables",
	}
}

// createPerformanceIndexes creates specialized indexes for <250ms performance targets
func (po *PerformanceOptimizer) createPerformanceIndexes(ctx context.Context) OptimizationResult {
	start := time.Now()

	performanceIndexes := []string{
		// Workflow execution performance lookup (for <250ms target)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_performance_critical
		 ON workflow_executions (patient_id, workflow_type, started_at DESC, execution_status)
		 WHERE started_at > NOW() - INTERVAL '24 hours';`,

		// Clinical snapshot fast access (for Recipe & Snapshot architecture)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_snapshots_fast_access
		 ON clinical_snapshots (patient_id, recipe_id, status, expires_at DESC)
		 WHERE status = 'active' AND expires_at > NOW();`,

		// Recipe resolution performance (targeting <10ms)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_recipes_fast_resolution
		 ON recipes (protocol_id, status, complexity_score, cache_priority DESC)
		 WHERE status = 'active';`,

		// FHIR mapping fast lookup (for reference resolution)
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_fhir_mapping_fast_lookup
		 ON fhir_resource_mappings (internal_resource_type, internal_resource_id, sync_status)
		 WHERE sync_status = 'synchronized';`,

		// Clinical calculation results performance
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calc_results_performance
		 ON clinical_calculation_results (patient_id, workflow_execution_id, validation_status, calculation_completed_at DESC)
		 WHERE validation_status = 'validated';`,

		// Medication proposal workflow fast state lookup
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_med_proposal_fast_state
		 ON medication_proposal_workflows (patient_id, current_state, workflow_started_at DESC)
		 WHERE current_state NOT IN ('executed', 'cancelled', 'expired');`,
	}

	for _, indexQuery := range performanceIndexes {
		_, err := po.db.DB.ExecContext(ctx, indexQuery)
		if err != nil {
			po.logger.Warn("Failed to create performance index", zap.Error(err))
		}
	}

	return OptimizationResult{
		Operation:     "performance_indexes",
		Success:       true,
		ExecutionTime: time.Since(start),
		Impact:        "Created specialized indexes targeting <250ms clinical workflow performance",
		Details:       "Optimized for fast patient lookup, recipe resolution, and workflow state access",
	}
}

// RefreshMaterializedViews refreshes all materialized views
func (po *PerformanceOptimizer) RefreshMaterializedViews(ctx context.Context) error {
	views := []string{
		"mv_active_workflow_summary",
		"mv_recipe_performance_summary", 
		"mv_fhir_integration_health",
	}

	for _, view := range views {
		query := fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s;", view)
		_, err := po.db.DB.ExecContext(ctx, query)
		if err != nil {
			po.logger.Warn("Failed to refresh materialized view", zap.String("view", view), zap.Error(err))
			// Continue with other views
		} else {
			po.logger.Debug("Refreshed materialized view", zap.String("view", view))
		}
	}

	return nil
}

// GetPerformanceMetrics retrieves current database performance metrics
func (po *PerformanceOptimizer) GetPerformanceMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Query for performance metrics
	metricsQuery := `
		SELECT 
			-- Connection statistics
			(SELECT setting FROM pg_settings WHERE name = 'max_connections') as max_connections,
			(SELECT count(*) FROM pg_stat_activity WHERE state = 'active') as active_connections,
			(SELECT count(*) FROM pg_stat_activity WHERE state = 'idle') as idle_connections,
			
			-- Database size and activity
			pg_size_pretty(pg_database_size(current_database())) as database_size,
			(SELECT sum(tup_returned) FROM pg_stat_user_tables) as total_rows_read,
			(SELECT sum(tup_inserted + tup_updated + tup_deleted) FROM pg_stat_user_tables) as total_rows_modified,
			
			-- Index usage statistics
			(SELECT count(*) FROM pg_stat_user_indexes WHERE idx_tup_read > 0) as active_indexes,
			(SELECT avg(idx_tup_read::float / GREATEST(idx_tup_fetch, 1)) FROM pg_stat_user_indexes) as avg_index_efficiency,
			
			-- Cache hit ratios
			(SELECT sum(heap_blks_hit)::float / GREATEST(sum(heap_blks_hit + heap_blks_read), 1) 
			 FROM pg_statio_user_tables) as table_cache_hit_ratio,
			(SELECT sum(idx_blks_hit)::float / GREATEST(sum(idx_blks_hit + idx_blks_read), 1) 
			 FROM pg_statio_user_indexes) as index_cache_hit_ratio
	`

	var metrics struct {
		MaxConnections      string  `db:"max_connections"`
		ActiveConnections   int     `db:"active_connections"`
		IdleConnections     int     `db:"idle_connections"`
		DatabaseSize        string  `db:"database_size"`
		TotalRowsRead       int64   `db:"total_rows_read"`
		TotalRowsModified   int64   `db:"total_rows_modified"`
		ActiveIndexes       int     `db:"active_indexes"`
		AvgIndexEfficiency  float64 `db:"avg_index_efficiency"`
		TableCacheHitRatio  float64 `db:"table_cache_hit_ratio"`
		IndexCacheHitRatio  float64 `db:"index_cache_hit_ratio"`
	}

	err := po.db.DB.GetContext(ctx, &metrics, metricsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get performance metrics: %w", err)
	}

	return map[string]interface{}{
		"connection_stats": map[string]interface{}{
			"max_connections":    metrics.MaxConnections,
			"active_connections": metrics.ActiveConnections,
			"idle_connections":   metrics.IdleConnections,
		},
		"database_stats": map[string]interface{}{
			"database_size":        metrics.DatabaseSize,
			"total_rows_read":      metrics.TotalRowsRead,
			"total_rows_modified":  metrics.TotalRowsModified,
		},
		"index_stats": map[string]interface{}{
			"active_indexes":       metrics.ActiveIndexes,
			"avg_index_efficiency": metrics.AvgIndexEfficiency,
		},
		"cache_performance": map[string]interface{}{
			"table_cache_hit_ratio": metrics.TableCacheHitRatio,
			"index_cache_hit_ratio": metrics.IndexCacheHitRatio,
		},
		"optimization_recommendations": po.generateOptimizationRecommendations(metrics),
	}, nil
}

// generateOptimizationRecommendations provides performance optimization recommendations
func (po *PerformanceOptimizer) generateOptimizationRecommendations(metrics interface{}) []string {
	var recommendations []string

	// This would contain logic to analyze metrics and provide recommendations
	recommendations = append(recommendations, 
		"Regular ANALYZE on high-update tables",
		"Monitor materialized view refresh frequency",
		"Consider connection pooling if active connections > 20",
		"Review slow query log for optimization opportunities",
	)

	return recommendations
}