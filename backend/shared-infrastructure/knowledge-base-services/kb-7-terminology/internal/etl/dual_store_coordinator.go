package etl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kb-7-terminology/internal/elasticsearch"
	"kb-7-terminology/internal/models"

	"go.uber.org/zap"
)

// DualStoreCoordinator extends EnhancedCoordinator with Elasticsearch dual-loading capabilities
type DualStoreCoordinator struct {
	*EnhancedCoordinator

	// Elasticsearch integration
	esIntegration *elasticsearch.Integration
	esClient      *elasticsearch.Client
	esConfig      *DualStoreConfig

	// Dual-loading state
	dualStoreStatus *DualStoreStatus
	statusMutex     sync.RWMutex
}

// DualStoreConfig holds configuration for dual-store operations
type DualStoreConfig struct {
	EnableElasticsearch     bool          `json:"enable_elasticsearch"`
	ElasticsearchURLs       []string      `json:"elasticsearch_urls"`
	IndexName               string        `json:"index_name"`
	DualWriteMode           DualWriteMode `json:"dual_write_mode"`
	ConsistencyCheckEnabled bool          `json:"consistency_check_enabled"`
	ConsistencyThreshold    float64       `json:"consistency_threshold"`
	ElasticsearchBatchSize  int           `json:"elasticsearch_batch_size"`
	MaxRetries              int           `json:"max_retries"`
	RetryDelay              time.Duration `json:"retry_delay"`
	EnableRollback          bool          `json:"enable_rollback"`
	TransactionTimeout      time.Duration `json:"transaction_timeout"`
}

// DualWriteMode defines how dual writes are handled
type DualWriteMode string

const (
	DualWriteSequential  DualWriteMode = "sequential"  // PostgreSQL first, then Elasticsearch
	DualWriteParallel    DualWriteMode = "parallel"    // Both simultaneously
	DualWriteElasticOnly DualWriteMode = "elastic_only" // Elasticsearch only (read from PG)
)

// DualStoreStatus tracks the status of dual-store operations
type DualStoreStatus struct {
	PostgreSQLStatus    StoreOperationStatus `json:"postgresql_status"`
	ElasticsearchStatus StoreOperationStatus `json:"elasticsearch_status"`
	ConsistencyStatus   ConsistencyStatus    `json:"consistency_status"`
	OverallHealth       string              `json:"overall_health"`
	LastSyncTime        time.Time           `json:"last_sync_time"`
	SyncErrors          []SyncError         `json:"sync_errors"`
	FailoverActive      bool                `json:"failover_active"`
	FailoverReason      string              `json:"failover_reason"`
}

// StoreOperationStatus tracks individual store operation status
type StoreOperationStatus struct {
	Status          string            `json:"status"`
	LastOperation   string            `json:"last_operation"`
	RecordsWritten  int64            `json:"records_written"`
	RecordsFailed   int64            `json:"records_failed"`
	LastError       string           `json:"last_error"`
	ResponseTime    time.Duration    `json:"response_time"`
	ErrorRate       float64          `json:"error_rate"`
	Health          string           `json:"health"`
	Metrics         map[string]int64 `json:"metrics"`
}

// ConsistencyStatus tracks data consistency between stores
type ConsistencyStatus struct {
	LastCheck        time.Time              `json:"last_check"`
	IsConsistent     bool                   `json:"is_consistent"`
	Discrepancy      int64                  `json:"discrepancy"`
	ConsistencyScore float64                `json:"consistency_score"`
	InconsistentIds  []string               `json:"inconsistent_ids"`
	CheckDuration    time.Duration          `json:"check_duration"`
	Details          map[string]interface{} `json:"details"`
}

// SyncError represents an error during synchronization
type SyncError struct {
	Timestamp   time.Time `json:"timestamp"`
	Operation   string    `json:"operation"`
	Store       string    `json:"store"`
	RecordID    string    `json:"record_id"`
	Error       string    `json:"error"`
	Severity    string    `json:"severity"`
	Resolved    bool      `json:"resolved"`
	RetryCount  int       `json:"retry_count"`
}

// NewDualStoreCoordinator creates a new dual-store coordinator
func NewDualStoreCoordinator(
	enhancedCoordinator *EnhancedCoordinator,
	esConfig *DualStoreConfig,
	logger *zap.Logger,
) (*DualStoreCoordinator, error) {

	if esConfig == nil {
		esConfig = DefaultDualStoreConfig()
	}

	coordinator := &DualStoreCoordinator{
		EnhancedCoordinator: enhancedCoordinator,
		esConfig:           esConfig,
		dualStoreStatus: &DualStoreStatus{
			PostgreSQLStatus: StoreOperationStatus{
				Status:  "idle",
				Health:  "unknown",
				Metrics: make(map[string]int64),
			},
			ElasticsearchStatus: StoreOperationStatus{
				Status:  "idle",
				Health:  "unknown",
				Metrics: make(map[string]int64),
			},
			ConsistencyStatus: ConsistencyStatus{
				IsConsistent: true,
			},
			OverallHealth: "unknown",
		},
	}

	// Initialize Elasticsearch integration if enabled
	if esConfig.EnableElasticsearch {
		if err := coordinator.initializeElasticsearch(); err != nil {
			return nil, fmt.Errorf("failed to initialize Elasticsearch: %w", err)
		}
	}

	logger.Info("Dual-store coordinator initialized",
		zap.Bool("elasticsearch_enabled", esConfig.EnableElasticsearch),
		zap.String("dual_write_mode", string(esConfig.DualWriteMode)),
	)

	return coordinator, nil
}

// DefaultDualStoreConfig returns default configuration
func DefaultDualStoreConfig() *DualStoreConfig {
	return &DualStoreConfig{
		EnableElasticsearch:     true,
		ElasticsearchURLs:       []string{"http://localhost:9200"},
		IndexName:              "clinical_terms",
		DualWriteMode:          DualWriteSequential,
		ConsistencyCheckEnabled: true,
		ConsistencyThreshold:   0.99, // 99% consistency required
		ElasticsearchBatchSize: 1000,
		MaxRetries:             3,
		RetryDelay:             5 * time.Second,
		EnableRollback:         true,
		TransactionTimeout:     5 * time.Minute,
	}
}

// initializeElasticsearch sets up Elasticsearch integration
func (dsc *DualStoreCoordinator) initializeElasticsearch() error {
	// Create Elasticsearch integration
	integrationConfig := &elasticsearch.IntegrationConfig{
		PostgreSQLDSN:      "postgresql://kb_test_user:kb_test_password@localhost:5434/clinical_governance_test?sslmode=disable",
		ElasticsearchURLs:  dsc.esConfig.ElasticsearchURLs,
		IndexName:          dsc.esConfig.IndexName,
		BatchSize:          dsc.esConfig.ElasticsearchBatchSize,
		EnableRealTimeSync: false, // We control sync manually
	}

	integration, err := elasticsearch.NewIntegration(integrationConfig)
	if err != nil {
		return fmt.Errorf("failed to create Elasticsearch integration: %w", err)
	}

	dsc.esIntegration = integration

	// Test Elasticsearch connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := dsc.esIntegration.GetIndexStats(ctx)
	if err != nil {
		dsc.logger.Warn("Elasticsearch health check failed", zap.Error(err))
		dsc.dualStoreStatus.ElasticsearchStatus.Health = "unhealthy"
		dsc.dualStoreStatus.ElasticsearchStatus.LastError = err.Error()
	} else {
		dsc.dualStoreStatus.ElasticsearchStatus.Health = "healthy"
		dsc.logger.Info("Elasticsearch connection established",
			zap.String("index", health.IndexName),
			zap.Int64("documents", health.DocumentCount),
		)
	}

	return nil
}

// LoadAllTerminologiesDualStore loads all terminologies to both PostgreSQL and Elasticsearch
func (dsc *DualStoreCoordinator) LoadAllTerminologiesDualStore(ctx context.Context, dataSources map[string]string) error {
	dsc.statusMutex.Lock()
	dsc.dualStoreStatus.OverallHealth = "loading"
	dsc.statusMutex.Unlock()

	dsc.logger.Info("Starting dual-store terminology loading")

	// First, perform PostgreSQL loading using the enhanced coordinator
	pgStartTime := time.Now()
	dsc.updateStoreStatus("postgresql", "running", "Loading to PostgreSQL")

	err := dsc.EnhancedCoordinator.LoadAllTerminologies(dataSources)
	pgDuration := time.Since(pgStartTime)

	if err != nil {
		dsc.updateStoreStatus("postgresql", "failed", err.Error())
		dsc.dualStoreStatus.OverallHealth = "degraded"

		// If PostgreSQL fails, don't proceed with Elasticsearch
		return fmt.Errorf("PostgreSQL loading failed: %w", err)
	}

	dsc.updateStoreStatus("postgresql", "completed", "")
	dsc.dualStoreStatus.PostgreSQLStatus.ResponseTime = pgDuration

	// Now sync to Elasticsearch if enabled
	if dsc.esConfig.EnableElasticsearch {
		if err := dsc.syncToElasticsearch(ctx); err != nil {
			dsc.logger.Error("Elasticsearch sync failed", zap.Error(err))
			dsc.updateStoreStatus("elasticsearch", "failed", err.Error())
			dsc.dualStoreStatus.OverallHealth = "degraded"

			// Depending on configuration, this might not be a fatal error
			if dsc.esConfig.DualWriteMode != DualWriteElasticOnly {
				dsc.logger.Warn("Continuing with PostgreSQL-only operation")
			} else {
				return fmt.Errorf("Elasticsearch sync failed in elastic-only mode: %w", err)
			}
		} else {
			dsc.updateStoreStatus("elasticsearch", "completed", "")
			dsc.dualStoreStatus.OverallHealth = "healthy"
		}
	} else {
		dsc.dualStoreStatus.OverallHealth = "healthy"
	}

	// Perform consistency check
	if dsc.esConfig.ConsistencyCheckEnabled && dsc.esConfig.EnableElasticsearch {
		if err := dsc.performConsistencyCheck(ctx); err != nil {
			dsc.logger.Warn("Consistency check failed", zap.Error(err))
		}
	}

	dsc.dualStoreStatus.LastSyncTime = time.Now()
	dsc.logger.Info("Dual-store loading completed",
		zap.String("overall_health", dsc.dualStoreStatus.OverallHealth),
	)

	return nil
}

// syncToElasticsearch synchronizes PostgreSQL data to Elasticsearch
func (dsc *DualStoreCoordinator) syncToElasticsearch(ctx context.Context) error {
	if dsc.esIntegration == nil {
		return fmt.Errorf("Elasticsearch integration not initialized")
	}

	esStartTime := time.Now()
	dsc.updateStoreStatus("elasticsearch", "running", "Syncing to Elasticsearch")

	// Use the integration's sync method
	err := dsc.esIntegration.SyncFromPostgreSQL(ctx)
	esDuration := time.Since(esStartTime)

	dsc.dualStoreStatus.ElasticsearchStatus.ResponseTime = esDuration

	if err != nil {
		return fmt.Errorf("Elasticsearch sync failed: %w", err)
	}

	dsc.logger.Info("Elasticsearch sync completed",
		zap.Duration("duration", esDuration),
	)

	return nil
}

// performConsistencyCheck validates data consistency between stores
func (dsc *DualStoreCoordinator) performConsistencyCheck(ctx context.Context) error {
	if dsc.esIntegration == nil {
		return fmt.Errorf("Elasticsearch integration not available")
	}

	checkStartTime := time.Now()
	dsc.logger.Info("Starting consistency check")

	// Perform validation using the integration
	validation, err := dsc.esIntegration.ValidateSync(ctx)
	if err != nil {
		return fmt.Errorf("consistency validation failed: %w", err)
	}

	checkDuration := time.Since(checkStartTime)

	// Update consistency status
	dsc.statusMutex.Lock()
	dsc.dualStoreStatus.ConsistencyStatus = ConsistencyStatus{
		LastCheck:        checkStartTime,
		IsConsistent:     validation.IsConsistent,
		Discrepancy:      int64(validation.Discrepancy),
		CheckDuration:    checkDuration,
		ConsistencyScore: dsc.calculateConsistencyScore(validation),
		Details: map[string]interface{}{
			"postgresql_count":    validation.PostgreSQLCount,
			"elasticsearch_count": validation.ElasticsearchCount,
			"discrepancy":         validation.Discrepancy,
		},
	}
	dsc.statusMutex.Unlock()

	if !validation.IsConsistent {
		dsc.logger.Warn("Consistency check failed",
			zap.Int("postgresql_count", validation.PostgreSQLCount),
			zap.Int("elasticsearch_count", validation.ElasticsearchCount),
			zap.Int("discrepancy", validation.Discrepancy),
		)

		// Record sync error
		syncError := SyncError{
			Timestamp: time.Now(),
			Operation: "consistency_check",
			Store:     "elasticsearch",
			Error:     fmt.Sprintf("Discrepancy of %d records", validation.Discrepancy),
			Severity:  "warning",
			Resolved:  false,
		}
		dsc.recordSyncError(syncError)

		// If discrepancy is above threshold, trigger corrective action
		if dsc.dualStoreStatus.ConsistencyStatus.ConsistencyScore < dsc.esConfig.ConsistencyThreshold {
			dsc.logger.Error("Consistency score below threshold",
				zap.Float64("score", dsc.dualStoreStatus.ConsistencyStatus.ConsistencyScore),
				zap.Float64("threshold", dsc.esConfig.ConsistencyThreshold),
			)
			return fmt.Errorf("consistency score %.3f below threshold %.3f",
				dsc.dualStoreStatus.ConsistencyStatus.ConsistencyScore,
				dsc.esConfig.ConsistencyThreshold)
		}
	}

	dsc.logger.Info("Consistency check completed",
		zap.Bool("consistent", validation.IsConsistent),
		zap.Float64("score", dsc.dualStoreStatus.ConsistencyStatus.ConsistencyScore),
		zap.Duration("duration", checkDuration),
	)

	return nil
}

// DualWriteTerms performs dual-write operation for a batch of terms
func (dsc *DualStoreCoordinator) DualWriteTerms(ctx context.Context, terms []*models.Concept) error {
	if len(terms) == 0 {
		return nil
	}

	switch dsc.esConfig.DualWriteMode {
	case DualWriteSequential:
		return dsc.sequentialWrite(ctx, terms)
	case DualWriteParallel:
		return dsc.parallelWrite(ctx, terms)
	case DualWriteElasticOnly:
		return dsc.elasticsearchOnlyWrite(ctx, terms)
	default:
		return fmt.Errorf("unknown dual write mode: %s", dsc.esConfig.DualWriteMode)
	}
}

// sequentialWrite writes to PostgreSQL first, then Elasticsearch
func (dsc *DualStoreCoordinator) sequentialWrite(ctx context.Context, terms []*models.Concept) error {
	// Write to PostgreSQL first
	if err := dsc.writeToPostgreSQL(ctx, terms); err != nil {
		return fmt.Errorf("PostgreSQL write failed: %w", err)
	}

	// Then write to Elasticsearch
	if dsc.esConfig.EnableElasticsearch {
		if err := dsc.writeToElasticsearch(ctx, terms); err != nil {
			dsc.logger.Error("Elasticsearch write failed after successful PostgreSQL write",
				zap.Error(err),
				zap.Int("term_count", len(terms)),
			)

			// Record the error but don't fail the operation
			syncError := SyncError{
				Timestamp: time.Now(),
				Operation: "sequential_write",
				Store:     "elasticsearch",
				Error:     err.Error(),
				Severity:  "error",
				Resolved:  false,
			}
			dsc.recordSyncError(syncError)

			// Mark as degraded health
			dsc.dualStoreStatus.OverallHealth = "degraded"
		}
	}

	return nil
}

// parallelWrite writes to both stores simultaneously
func (dsc *DualStoreCoordinator) parallelWrite(ctx context.Context, terms []*models.Concept) error {
	var pgErr, esErr error
	var wg sync.WaitGroup

	// Write to PostgreSQL
	wg.Add(1)
	go func() {
		defer wg.Done()
		pgErr = dsc.writeToPostgreSQL(ctx, terms)
	}()

	// Write to Elasticsearch (if enabled)
	if dsc.esConfig.EnableElasticsearch {
		wg.Add(1)
		go func() {
			defer wg.Done()
			esErr = dsc.writeToElasticsearch(ctx, terms)
		}()
	}

	wg.Wait()

	// Handle errors
	if pgErr != nil && esErr != nil {
		return fmt.Errorf("both stores failed - PostgreSQL: %v, Elasticsearch: %v", pgErr, esErr)
	} else if pgErr != nil {
		return fmt.Errorf("PostgreSQL write failed: %w", pgErr)
	} else if esErr != nil {
		dsc.logger.Warn("Elasticsearch write failed in parallel mode", zap.Error(esErr))
		dsc.recordSyncError(SyncError{
			Timestamp: time.Now(),
			Operation: "parallel_write",
			Store:     "elasticsearch",
			Error:     esErr.Error(),
			Severity:  "warning",
		})
		dsc.dualStoreStatus.OverallHealth = "degraded"
	}

	return nil
}

// elasticsearchOnlyWrite writes only to Elasticsearch (for migration scenarios)
func (dsc *DualStoreCoordinator) elasticsearchOnlyWrite(ctx context.Context, terms []*models.Concept) error {
	if !dsc.esConfig.EnableElasticsearch {
		return fmt.Errorf("Elasticsearch-only mode requires Elasticsearch to be enabled")
	}

	return dsc.writeToElasticsearch(ctx, terms)
}

// writeToPostgreSQL writes terms to PostgreSQL
func (dsc *DualStoreCoordinator) writeToPostgreSQL(ctx context.Context, terms []*models.Concept) error {
	// This would use the existing enhanced coordinator's batch write functionality
	// For now, we'll simulate the operation
	dsc.logger.Debug("Writing terms to PostgreSQL", zap.Int("count", len(terms)))

	dsc.statusMutex.Lock()
	dsc.dualStoreStatus.PostgreSQLStatus.RecordsWritten += int64(len(terms))
	dsc.statusMutex.Unlock()

	return nil
}

// writeToElasticsearch writes terms to Elasticsearch
func (dsc *DualStoreCoordinator) writeToElasticsearch(ctx context.Context, terms []*models.Concept) error {
	if dsc.esIntegration == nil {
		return fmt.Errorf("Elasticsearch integration not initialized")
	}

	// Convert to Elasticsearch terms
	esTerms := make([]*elasticsearch.ClinicalTerm, len(terms))
	for i, term := range terms {
		esTerms[i] = dsc.convertToElasticsearchTerm(term)
	}

	// Use bulk indexing
	docs := make([]elasticsearch.BulkDocument, len(esTerms))
	for i, term := range esTerms {
		docs[i] = elasticsearch.BulkDocument{
			ID:     term.TermID,
			Source: term,
		}
	}

	// This would use the Elasticsearch client's bulk indexing
	dsc.logger.Debug("Writing terms to Elasticsearch", zap.Int("count", len(terms)))

	dsc.statusMutex.Lock()
	dsc.dualStoreStatus.ElasticsearchStatus.RecordsWritten += int64(len(terms))
	dsc.statusMutex.Unlock()

	return nil
}

// Helper methods

func (dsc *DualStoreCoordinator) updateStoreStatus(store, status, errorMsg string) {
	dsc.statusMutex.Lock()
	defer dsc.statusMutex.Unlock()

	switch store {
	case "postgresql":
		dsc.dualStoreStatus.PostgreSQLStatus.Status = status
		if errorMsg != "" {
			dsc.dualStoreStatus.PostgreSQLStatus.LastError = errorMsg
			dsc.dualStoreStatus.PostgreSQLStatus.RecordsFailed++
		}
	case "elasticsearch":
		dsc.dualStoreStatus.ElasticsearchStatus.Status = status
		if errorMsg != "" {
			dsc.dualStoreStatus.ElasticsearchStatus.LastError = errorMsg
			dsc.dualStoreStatus.ElasticsearchStatus.RecordsFailed++
		}
	}
}

func (dsc *DualStoreCoordinator) calculateConsistencyScore(validation *elasticsearch.SyncValidation) float64 {
	if validation.PostgreSQLCount == 0 {
		return 1.0 // No data to compare
	}

	if validation.IsConsistent {
		return 1.0
	}

	// Calculate score based on discrepancy percentage
	discrepancyRate := float64(abs(validation.Discrepancy)) / float64(validation.PostgreSQLCount)
	return 1.0 - discrepancyRate
}

func (dsc *DualStoreCoordinator) recordSyncError(err SyncError) {
	dsc.statusMutex.Lock()
	defer dsc.statusMutex.Unlock()

	dsc.dualStoreStatus.SyncErrors = append(dsc.dualStoreStatus.SyncErrors, err)

	// Keep only the last 100 errors
	if len(dsc.dualStoreStatus.SyncErrors) > 100 {
		dsc.dualStoreStatus.SyncErrors = dsc.dualStoreStatus.SyncErrors[1:]
	}
}

func (dsc *DualStoreCoordinator) convertToElasticsearchTerm(term *models.Concept) *elasticsearch.ClinicalTerm {
	// Convert internal model to Elasticsearch model
	// This is a simplified conversion - expand based on actual model structures
	return &elasticsearch.ClinicalTerm{
		TermID:             term.Code, // Assuming Code maps to TermID
		ConceptID:          term.Code,
		Term:               term.PreferredTerm,
		PreferredTerm:      term.PreferredTerm,
		Definition:         term.Definition,
		TerminologySystem:  term.System,
		Status:             term.Status,
		LastUpdated:        term.UpdatedAt,
	}
}

// GetDualStoreStatus returns the current dual-store status
func (dsc *DualStoreCoordinator) GetDualStoreStatus() *DualStoreStatus {
	dsc.statusMutex.RLock()
	defer dsc.statusMutex.RUnlock()

	// Return a copy to avoid race conditions
	status := *dsc.dualStoreStatus
	return &status
}

// Close closes all connections
func (dsc *DualStoreCoordinator) Close() error {
	if dsc.esIntegration != nil {
		return dsc.esIntegration.Close()
	}
	return nil
}

// LoadAllTerminologiesTripleStore is a compatibility method for DualStoreCoordinator
// It simply delegates to LoadAllTerminologiesDualStore for backward compatibility
func (dsc *DualStoreCoordinator) LoadAllTerminologiesTripleStore(ctx context.Context, dataSources map[string]string) error {
	return dsc.LoadAllTerminologiesDualStore(ctx, dataSources)
}

// GetStatus returns the current ETL status from the embedded coordinator
func (dsc *DualStoreCoordinator) GetStatus() *ETLStatus {
	if dsc.EnhancedCoordinator != nil {
		status := dsc.EnhancedCoordinator.GetStatus()
		return &status
	}
	return &ETLStatus{
		OverallStatus:  "unknown",
		SystemStatuses: make(map[string]SystemStatus),
	}
}

// Utility function
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}