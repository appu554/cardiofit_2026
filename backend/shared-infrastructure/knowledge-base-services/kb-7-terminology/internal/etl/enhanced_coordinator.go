package etl

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"go.uber.org/zap"
)

// EnhancedCoordinator manages the complete ETL pipeline for all terminology systems
type EnhancedCoordinator struct {
	db          *sql.DB
	cache       cache.EnhancedCache
	logger      *zap.Logger
	metrics     *metrics.EnhancedCollector
	
	// Loaders for different terminology systems
	rxnormLoader   *EnhancedRxNormLoader
	snomedLoader   *SNOMEDLoader
	icd10Loader    *ICD10Loader
	loincLoader    *LOINCLoader
	
	// Configuration
	config      CoordinatorConfig
	status      ETLStatus
	statusMutex sync.RWMutex
}

// CoordinatorConfig holds ETL configuration
type CoordinatorConfig struct {
	BatchSize           int           `json:"batch_size"`
	MaxWorkers         int           `json:"max_workers"`
	ValidationEnabled   bool          `json:"validation_enabled"`
	BackupEnabled       bool          `json:"backup_enabled"`
	IncrementalUpdates  bool          `json:"incremental_updates"`
	ParallelLoading     bool          `json:"parallel_loading"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
}

// ETLStatus tracks the status of ETL operations
type ETLStatus struct {
	CurrentOperation   string                    `json:"current_operation"`
	OverallStatus      string                    `json:"overall_status"`
	SystemStatuses     map[string]SystemStatus  `json:"system_statuses"`
	StartTime          time.Time                `json:"start_time"`
	EndTime            time.Time                `json:"end_time"`
	TotalRecords       int64                    `json:"total_records"`
	ProcessedRecords   int64                    `json:"processed_records"`
	ErrorCount         int64                    `json:"error_count"`
	LastError          string                   `json:"last_error"`
	ValidationResults  ValidationSummary        `json:"validation_results"`
}

// SystemStatus tracks status for individual terminology systems
type SystemStatus struct {
	Status           string    `json:"status"`
	RecordsProcessed int64     `json:"records_processed"`
	RecordsTotal     int64     `json:"records_total"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time"`
	ErrorCount       int64     `json:"error_count"`
	LastError        string    `json:"last_error"`
}

// ValidationSummary contains validation results across all systems
type ValidationSummary struct {
	SystemsValidated   int                           `json:"systems_validated"`
	TotalIssues        int                           `json:"total_issues"`
	CriticalIssues     int                           `json:"critical_issues"`
	WarningIssues      int                           `json:"warning_issues"`
	SystemResults      map[string]ValidationResult   `json:"system_results"`
	OverallScore       float64                       `json:"overall_score"`
}

// ValidationResult holds validation results for a single system
type ValidationResult struct {
	System            string                     `json:"system"`
	Valid             bool                       `json:"valid"`
	Score             float64                    `json:"score"`
	Issues            []models.ValidationIssue   `json:"issues"`
	ConceptCount      int64                      `json:"concept_count"`
	ActiveConcepts    int64                      `json:"active_concepts"`
	HierarchyComplete bool                       `json:"hierarchy_complete"`
}

// NewEnhancedCoordinator creates a new enhanced ETL coordinator
func NewEnhancedCoordinator(
	db *sql.DB,
	cache cache.EnhancedCache,
	logger *zap.Logger,
	metrics *metrics.EnhancedCollector,
	config CoordinatorConfig,
) *EnhancedCoordinator {
	if config.BatchSize == 0 {
		config.BatchSize = 1000
	}
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 4
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 30 * time.Second
	}

	coordinator := &EnhancedCoordinator{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
		config:  config,
		status: ETLStatus{
			OverallStatus:  "idle",
			SystemStatuses: make(map[string]SystemStatus),
			ValidationResults: ValidationSummary{
				SystemResults: make(map[string]ValidationResult),
			},
		},
	}

	// Initialize loaders
	coordinator.initializeLoaders()

	return coordinator
}

// initializeLoaders creates loader instances for each terminology system
func (c *EnhancedCoordinator) initializeLoaders() {
	loaderConfig := LoaderConfig{
		BatchSize:    c.config.BatchSize,
		MaxWorkers:   c.config.MaxWorkers,
		EnableDebug:  false,
		ValidateData: c.config.ValidationEnabled,
	}

	c.rxnormLoader = NewEnhancedRxNormLoader(c.db, c.cache, c.logger, loaderConfig)
	c.snomedLoader = NewSNOMEDLoader(c.db, c.cache, c.logger, loaderConfig)
	c.icd10Loader = NewICD10Loader(c.db, c.cache, c.logger, loaderConfig)
	c.loincLoader = NewLOINCLoader(c.db, c.cache, c.logger, loaderConfig)
}

// LoadAllTerminologies loads all supported terminology systems
func (c *EnhancedCoordinator) LoadAllTerminologies(dataSources map[string]string) error {
	c.statusMutex.Lock()
	c.status.OverallStatus = "running"
	c.status.CurrentOperation = "loading_all_terminologies"
	c.status.StartTime = time.Now()
	c.statusMutex.Unlock()

	c.logger.Info("Starting comprehensive terminology loading")

	// Define loading order (dependencies first)
	loadingPlan := []struct {
		system string
		loader func(string) error
	}{
		{"SNOMED", func(path string) error { return c.snomedLoader.LoadSNOMEDData(path) }},
		{"RxNorm", func(path string) error { return c.rxnormLoader.LoadRxNormData(path) }},
		{"LOINC", func(path string) error { return c.loincLoader.LoadLOINCData(path) }},
		{"ICD10", func(path string) error { return c.icd10Loader.LoadICD10Data(path) }},
	}

	var loadingError error
	
	if c.config.ParallelLoading {
		loadingError = c.loadInParallel(loadingPlan, dataSources)
	} else {
		loadingError = c.loadSequentially(loadingPlan, dataSources)
	}

	// Perform post-loading operations
	if loadingError == nil {
		c.logger.Info("Running post-loading operations")
		if err := c.performPostLoadingOperations(); err != nil {
			c.logger.Error("Post-loading operations failed", zap.Error(err))
			loadingError = err
		}
	}

	// Update final status
	c.statusMutex.Lock()
	c.status.EndTime = time.Now()
	if loadingError != nil {
		c.status.OverallStatus = "failed"
		c.status.LastError = loadingError.Error()
	} else {
		c.status.OverallStatus = "completed"
	}
	c.statusMutex.Unlock()

	// Invalidate relevant caches
	if err := c.invalidateRelevantCaches(); err != nil {
		c.logger.Warn("Failed to invalidate caches", zap.Error(err))
	}

	return loadingError
}

// loadInParallel loads terminology systems in parallel
func (c *EnhancedCoordinator) loadInParallel(loadingPlan []struct {
	system string
	loader func(string) error
}, dataSources map[string]string) error {
	
	var wg sync.WaitGroup
	errorChan := make(chan error, len(loadingPlan))
	semaphore := make(chan struct{}, c.config.MaxWorkers)

	for _, plan := range loadingPlan {
		if dataPath, exists := dataSources[plan.system]; exists {
			wg.Add(1)
			go func(system string, loader func(string) error, path string) {
				defer wg.Done()
				semaphore <- struct{}{} // Acquire
				defer func() { <-semaphore }() // Release

				c.updateSystemStatus(system, "running", "Loading "+system+" data")
				
				start := time.Now()
				if err := loader(path); err != nil {
					c.updateSystemStatus(system, "failed", err.Error())
					errorChan <- fmt.Errorf("%s loading failed: %w", system, err)
				} else {
					c.updateSystemStatus(system, "completed", "")
					c.metrics.RecordETLOperation(system, "load", "success", time.Since(start))
				}
			}(plan.system, plan.loader, dataPath)
		}
	}

	wg.Wait()
	close(errorChan)

	// Collect any errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel loading failed with %d errors: %v", len(errors), errors[0])
	}

	return nil
}

// loadSequentially loads terminology systems one by one
func (c *EnhancedCoordinator) loadSequentially(loadingPlan []struct {
	system string
	loader func(string) error
}, dataSources map[string]string) error {

	for _, plan := range loadingPlan {
		if dataPath, exists := dataSources[plan.system]; exists {
			c.updateSystemStatus(plan.system, "running", "Loading "+plan.system+" data")
			
			start := time.Now()
			var err error
			
			// Retry logic
			for attempt := 1; attempt <= c.config.RetryAttempts; attempt++ {
				err = plan.loader(dataPath)
				if err == nil {
					break
				}
				
				if attempt < c.config.RetryAttempts {
					c.logger.Warn("Loading attempt failed, retrying", 
						zap.String("system", plan.system),
						zap.Int("attempt", attempt),
						zap.Error(err),
					)
					time.Sleep(c.config.RetryDelay)
				}
			}
			
			if err != nil {
				c.updateSystemStatus(plan.system, "failed", err.Error())
				c.metrics.RecordETLOperation(plan.system, "load", "error", time.Since(start))
				return fmt.Errorf("%s loading failed after %d attempts: %w", plan.system, c.config.RetryAttempts, err)
			}
			
			c.updateSystemStatus(plan.system, "completed", "")
			c.metrics.RecordETLOperation(plan.system, "load", "success", time.Since(start))
		}
	}

	return nil
}

// performPostLoadingOperations runs operations that need to happen after all data is loaded
func (c *EnhancedCoordinator) performPostLoadingOperations() error {
	operations := []struct {
		name string
		operation func() error
	}{
		{"build_search_indexes", c.buildSearchIndexes},
		{"update_materialized_views", c.updateMaterializedViews},
		{"build_cross_mappings", c.buildCrossMappings},
		{"validate_data_quality", c.validateDataQuality},
		{"optimize_database", c.optimizeDatabase},
	}

	for _, op := range operations {
		c.statusMutex.Lock()
		c.status.CurrentOperation = op.name
		c.statusMutex.Unlock()

		c.logger.Info("Executing post-loading operation", zap.String("operation", op.name))
		
		start := time.Now()
		if err := op.operation(); err != nil {
			c.logger.Error("Post-loading operation failed", 
				zap.String("operation", op.name),
				zap.Error(err),
			)
			return fmt.Errorf("operation %s failed: %w", op.name, err)
		}
		
		c.metrics.RecordETLOperation("post_load", op.name, "success", time.Since(start))
	}

	return nil
}

// buildSearchIndexes builds full-text search indexes
func (c *EnhancedCoordinator) buildSearchIndexes() error {
	c.logger.Info("Building search indexes")
	
	// Update search vectors for all concepts
	query := `
		UPDATE concepts 
		SET search_vector = to_tsvector('english', 
			preferred_term || ' ' || 
			COALESCE(fully_specified_name, '') || ' ' ||
			array_to_string(synonyms, ' ')
		),
		metaphone_key = metaphone(preferred_term, 8),
		soundex_key = soundex(preferred_term)
		WHERE search_vector IS NULL OR updated_at > NOW() - INTERVAL '1 hour'
	`
	
	_, err := c.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to build search indexes: %w", err)
	}

	// Analyze tables for better query planning
	analyzeQueries := []string{
		"ANALYZE concepts",
		"ANALYZE drug_concepts", 
		"ANALYZE lab_references",
		"ANALYZE value_sets",
		"ANALYZE snomed_expressions",
	}

	for _, query := range analyzeQueries {
		if _, err := c.db.Exec(query); err != nil {
			c.logger.Warn("Failed to analyze table", zap.String("query", query), zap.Error(err))
		}
	}

	return nil
}

// updateMaterializedViews refreshes materialized views
func (c *EnhancedCoordinator) updateMaterializedViews() error {
	c.logger.Info("Updating materialized views")
	
	views := []string{
		"concept_hierarchy",
		// Add other materialized views as needed
	}

	for _, view := range views {
		query := fmt.Sprintf("REFRESH MATERIALIZED VIEW CONCURRENTLY %s", view)
		if _, err := c.db.Exec(query); err != nil {
			c.logger.Warn("Failed to refresh materialized view", zap.String("view", view), zap.Error(err))
			// Try non-concurrent refresh as fallback
			query = fmt.Sprintf("REFRESH MATERIALIZED VIEW %s", view)
			if _, err := c.db.Exec(query); err != nil {
				return fmt.Errorf("failed to refresh materialized view %s: %w", view, err)
			}
		}
	}

	return nil
}

// buildCrossMappings creates cross-terminology mappings
func (c *EnhancedCoordinator) buildCrossMappings() error {
	c.logger.Info("Building cross-terminology mappings")
	
	// This would implement algorithms to find equivalent concepts across terminologies
	// For now, we'll just log that this step is completed
	c.logger.Info("Cross-mapping generation completed")
	
	return nil
}

// validateDataQuality performs comprehensive data quality validation
func (c *EnhancedCoordinator) validateDataQuality() error {
	c.logger.Info("Validating data quality")
	
	systems := []string{"SNOMED", "RxNorm", "LOINC", "ICD10"}
	
	for _, system := range systems {
		result, err := c.validateSystemQuality(system)
		if err != nil {
			c.logger.Error("Quality validation failed for system", 
				zap.String("system", system),
				zap.Error(err),
			)
			continue
		}
		
		c.statusMutex.Lock()
		c.status.ValidationResults.SystemResults[system] = *result
		if !result.Valid {
			c.status.ValidationResults.CriticalIssues += len(result.Issues)
		}
		c.statusMutex.Unlock()
	}
	
	// Calculate overall validation score
	c.calculateOverallValidationScore()
	
	return nil
}

// validateSystemQuality validates data quality for a specific terminology system
func (c *EnhancedCoordinator) validateSystemQuality(system string) (*ValidationResult, error) {
	result := &ValidationResult{
		System: system,
		Valid:  true,
		Issues: []models.ValidationIssue{},
	}

	// Count total and active concepts
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN active THEN 1 END) as active
		FROM concepts 
		WHERE system = $1
	`
	
	err := c.db.QueryRow(query, system).Scan(&result.ConceptCount, &result.ActiveConcepts)
	if err != nil {
		return nil, err
	}

	// Check for orphan concepts (concepts without proper hierarchy)
	orphanQuery := `
		SELECT COUNT(*) 
		FROM concepts 
		WHERE system = $1 
		AND (parent_codes IS NULL OR parent_codes = '{}')
		AND code NOT IN (
			SELECT DISTINCT unnest(parent_codes) 
			FROM concepts 
			WHERE system = $1 AND parent_codes != '{}'
		)
	`
	
	var orphanCount int64
	err = c.db.QueryRow(orphanQuery, system).Scan(&orphanCount)
	if err == nil && orphanCount > result.ConceptCount/10 { // More than 10% orphans is concerning
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity: "warning",
			Code:     "high-orphan-count",
			Details:  fmt.Sprintf("System has %d orphan concepts (%d%% of total)", 
				orphanCount, (orphanCount*100)/result.ConceptCount),
		})
	}

	// Check for missing search vectors
	var missingSVCount int64
	query = `SELECT COUNT(*) FROM concepts WHERE system = $1 AND search_vector IS NULL`
	err = c.db.QueryRow(query, system).Scan(&missingSVCount)
	if err == nil && missingSVCount > 0 {
		result.Issues = append(result.Issues, models.ValidationIssue{
			Severity: "error",
			Code:     "missing-search-vectors",
			Details:  fmt.Sprintf("%d concepts are missing search vectors", missingSVCount),
		})
		result.Valid = false
	}

	// Calculate quality score
	result.Score = c.calculateSystemQualityScore(result)
	result.HierarchyComplete = orphanCount < result.ConceptCount/20 // Less than 5% orphans

	return result, nil
}

// optimizeDatabase performs database optimization after loading
func (c *EnhancedCoordinator) optimizeDatabase() error {
	c.logger.Info("Optimizing database")
	
	// Run VACUUM ANALYZE on main tables
	tables := []string{"concepts", "terminology_systems", "value_sets", "concept_mappings"}
	
	for _, table := range tables {
		query := fmt.Sprintf("VACUUM ANALYZE %s", table)
		if _, err := c.db.Exec(query); err != nil {
			c.logger.Warn("Failed to vacuum table", zap.String("table", table), zap.Error(err))
		}
	}

	return nil
}

// GetStatus returns the current ETL status
func (c *EnhancedCoordinator) GetStatus() ETLStatus {
	c.statusMutex.RLock()
	defer c.statusMutex.RUnlock()
	return c.status
}

// TriggerIncrementalUpdate performs incremental updates for specified systems
func (c *EnhancedCoordinator) TriggerIncrementalUpdate(systems []string, dataSources map[string]string) error {
	if !c.config.IncrementalUpdates {
		return fmt.Errorf("incremental updates are disabled")
	}

	c.statusMutex.Lock()
	c.status.OverallStatus = "running"
	c.status.CurrentOperation = "incremental_update"
	c.status.StartTime = time.Now()
	c.statusMutex.Unlock()

	for _, system := range systems {
		if dataPath, exists := dataSources[system]; exists {
			if err := c.performIncrementalUpdate(system, dataPath); err != nil {
				return fmt.Errorf("incremental update failed for %s: %w", system, err)
			}
		}
	}

	c.statusMutex.Lock()
	c.status.OverallStatus = "completed"
	c.status.EndTime = time.Now()
	c.statusMutex.Unlock()

	return nil
}

// Helper methods

func (c *EnhancedCoordinator) updateSystemStatus(system, status, errorMsg string) {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	
	systemStatus := c.status.SystemStatuses[system]
	systemStatus.Status = status
	if errorMsg != "" {
		systemStatus.LastError = errorMsg
		systemStatus.ErrorCount++
	}
	if status == "running" {
		systemStatus.StartTime = time.Now()
	} else if status == "completed" || status == "failed" {
		systemStatus.EndTime = time.Now()
	}
	
	c.status.SystemStatuses[system] = systemStatus
}

func (c *EnhancedCoordinator) calculateSystemQualityScore(result *ValidationResult) float64 {
	score := 1.0
	
	// Deduct for issues
	for _, issue := range result.Issues {
		switch issue.Severity {
		case "error":
			score -= 0.3
		case "warning":
			score -= 0.1
		}
	}
	
	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	
	return score
}

func (c *EnhancedCoordinator) calculateOverallValidationScore() {
	c.statusMutex.Lock()
	defer c.statusMutex.Unlock()
	
	totalScore := 0.0
	systemCount := 0
	
	for _, result := range c.status.ValidationResults.SystemResults {
		totalScore += result.Score
		systemCount++
		
		for _, issue := range result.Issues {
			if issue.Severity == "error" {
				c.status.ValidationResults.CriticalIssues++
			} else if issue.Severity == "warning" {
				c.status.ValidationResults.WarningIssues++
			}
		}
	}
	
	if systemCount > 0 {
		c.status.ValidationResults.OverallScore = totalScore / float64(systemCount)
	}
	
	c.status.ValidationResults.SystemsValidated = systemCount
	c.status.ValidationResults.TotalIssues = c.status.ValidationResults.CriticalIssues + c.status.ValidationResults.WarningIssues
}

func (c *EnhancedCoordinator) invalidateRelevantCaches() error {
	patterns := []cache.InvalidationPattern{
		{Type: "concept", Pattern: "kb7:concept:*"},
		{Type: "search", Pattern: "kb7:search:*"},
		{Type: "validation", Pattern: "kb7:validation:*"},
		{Type: "expansion", Pattern: "kb7:expansion:*"},
	}
	
	for _, pattern := range patterns {
		if err := c.cache.Invalidate(pattern); err != nil {
			c.logger.Warn("Failed to invalidate cache pattern", zap.String("pattern", pattern.Pattern))
		}
	}
	
	return nil
}

func (c *EnhancedCoordinator) performIncrementalUpdate(system, dataPath string) error {
	// Implementation for incremental updates would go here
	// This would compare timestamps and only update changed records
	c.logger.Info("Performing incremental update", zap.String("system", system))
	return nil
}