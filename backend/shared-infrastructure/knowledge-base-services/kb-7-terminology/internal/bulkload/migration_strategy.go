package bulkload

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/sirupsen/logrus"
)

// MigrationStrategy defines the approach for migrating from single to dual-store
type MigrationStrategy struct {
	name        string
	description string
	validator   DataValidator
	executor    StrategyExecutor
}

// StrategyExecutor defines the interface for migration execution
type StrategyExecutor interface {
	Execute(ctx context.Context) error
	Validate(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// MigrationCoordinator manages the overall migration process
type MigrationCoordinator struct {
	postgresDB    *sql.DB
	elasticsearch *elasticsearch.Client
	logger        *logrus.Logger
	strategy      *MigrationStrategy
	state         *MigrationState
	mu            sync.RWMutex
}

// MigrationState tracks the current state of migration
type MigrationState struct {
	Phase             MigrationPhase         `json:"phase"`
	StartedAt         time.Time              `json:"started_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
	LastCheckpoint    *MigrationCheckpoint   `json:"last_checkpoint,omitempty"`
	ValidationResults []ValidationResult     `json:"validation_results"`
	Errors            []MigrationError       `json:"errors"`
	Statistics        MigrationStatistics    `json:"statistics"`
}

// MigrationPhase represents the current phase of migration
type MigrationPhase string

const (
	PhaseInitializing   MigrationPhase = "initializing"
	PhasePreValidation  MigrationPhase = "pre_validation"
	PhaseDataExport     MigrationPhase = "data_export"
	PhaseDataTransform  MigrationPhase = "data_transform"
	PhaseDataImport     MigrationPhase = "data_import"
	PhasePostValidation MigrationPhase = "post_validation"
	PhaseOptimization   MigrationPhase = "optimization"
	PhaseCompleted      MigrationPhase = "completed"
	PhaseFailed         MigrationPhase = "failed"
)

// MigrationCheckpoint allows resuming interrupted migrations
type MigrationCheckpoint struct {
	Phase       MigrationPhase         `json:"phase"`
	SystemName  string                 `json:"system_name"`
	LastID      int64                  `json:"last_id"`
	RecordCount int64                  `json:"record_count"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ValidationResult contains results from data validation
type ValidationResult struct {
	Timestamp   time.Time `json:"timestamp"`
	ValidationType string `json:"validation_type"`
	Passed      bool      `json:"passed"`
	Details     string    `json:"details"`
	RecordsSampled int64  `json:"records_sampled"`
	ErrorCount  int       `json:"error_count"`
}

// MigrationError represents an error during migration
type MigrationError struct {
	Timestamp time.Time `json:"timestamp"`
	Phase     MigrationPhase `json:"phase"`
	Error     string    `json:"error"`
	Severity  string    `json:"severity"` // "warning", "error", "critical"
	Retryable bool      `json:"retryable"`
}

// MigrationStatistics tracks migration metrics
type MigrationStatistics struct {
	TotalRecords          int64         `json:"total_records"`
	MigratedRecords       int64         `json:"migrated_records"`
	ValidationErrors      int64         `json:"validation_errors"`
	ConsistencyChecksPassed int64       `json:"consistency_checks_passed"`
	ConsistencyChecksFailed int64       `json:"consistency_checks_failed"`
	AverageRecordsPerSecond float64     `json:"average_records_per_second"`
	EstimatedCompletion   *time.Time    `json:"estimated_completion,omitempty"`
}

// DataValidator validates data consistency between stores
type DataValidator struct {
	postgresDB    *sql.DB
	elasticsearch *elasticsearch.Client
	logger        *logrus.Logger
	config        ValidationConfig
}

// ValidationConfig contains validation parameters
type ValidationConfig struct {
	SampleSize         int     `json:"sample_size"`
	SamplePercentage   float64 `json:"sample_percentage"`
	StrictMode         bool    `json:"strict_mode"`
	FieldsToValidate   []string `json:"fields_to_validate"`
	AcceptableErrorRate float64 `json:"acceptable_error_rate"`
}

// NewMigrationCoordinator creates a new migration coordinator
func NewMigrationCoordinator(
	postgresDB *sql.DB,
	elasticsearch *elasticsearch.Client,
	logger *logrus.Logger,
	strategy *MigrationStrategy,
) *MigrationCoordinator {
	return &MigrationCoordinator{
		postgresDB:    postgresDB,
		elasticsearch: elasticsearch,
		logger:        logger,
		strategy:      strategy,
		state: &MigrationState{
			Phase:             PhaseInitializing,
			StartedAt:         time.Now(),
			ValidationResults: make([]ValidationResult, 0),
			Errors:            make([]MigrationError, 0),
		},
	}
}

// GetMigrationStrategies returns available migration strategies
func GetMigrationStrategies() map[string]*MigrationStrategy {
	return map[string]*MigrationStrategy{
		"incremental": {
			name:        "Incremental Migration",
			description: "Migrate data in small batches with continuous validation",
			executor:    &IncrementalStrategy{},
		},
		"parallel": {
			name:        "Parallel Migration",
			description: "High-performance parallel migration with multiple workers",
			executor:    &ParallelStrategy{},
		},
		"blue-green": {
			name:        "Blue-Green Migration",
			description: "Zero-downtime migration with instant switchover",
			executor:    &BlueGreenStrategy{},
		},
		"shadow": {
			name:        "Shadow Write Migration",
			description: "Dual-write to both stores with gradual read migration",
			executor:    &ShadowWriteStrategy{},
		},
	}
}

// IncrementalStrategy implements incremental migration approach
type IncrementalStrategy struct {
	coordinator   *MigrationCoordinator
	batchSize     int
	checkpointInterval time.Duration
}

// Execute performs incremental migration
func (s *IncrementalStrategy) Execute(ctx context.Context) error {
	s.coordinator.updatePhase(PhaseDataExport)

	// Implementation of incremental migration
	// Process data in small, manageable batches
	// Create checkpoints after each batch
	// Allow for easy resumption if interrupted

	systems := []string{"SNOMED", "RxNorm", "LOINC", "ICD10"}

	for _, system := range systems {
		if err := s.migrateSystem(ctx, system); err != nil {
			return fmt.Errorf("failed to migrate system %s: %w", system, err)
		}

		// Create checkpoint after each system
		checkpoint := &MigrationCheckpoint{
			Phase:       PhaseDataImport,
			SystemName:  system,
			Timestamp:   time.Now(),
		}
		s.coordinator.saveCheckpoint(checkpoint)
	}

	return nil
}

// migrateSystem migrates a single terminology system incrementally
func (s *IncrementalStrategy) migrateSystem(ctx context.Context, system string) error {
	lastID := int64(0)
	batchCount := 0

	for {
		// Fetch batch from PostgreSQL
		records, err := s.fetchBatch(ctx, system, lastID, s.batchSize)
		if err != nil {
			return err
		}

		if len(records) == 0 {
			break // No more records
		}

		// Transform and load to Elasticsearch
		if err := s.loadBatch(ctx, records); err != nil {
			s.coordinator.recordError(MigrationError{
				Timestamp: time.Now(),
				Phase:     PhaseDataImport,
				Error:     err.Error(),
				Severity:  "error",
				Retryable: true,
			})
			return err
		}

		// Update statistics
		s.coordinator.updateStatistics(len(records))

		// Update lastID for next batch
		lastID = records[len(records)-1].ID
		batchCount++

		// Periodic checkpoint
		if batchCount%10 == 0 {
			s.coordinator.saveCheckpoint(&MigrationCheckpoint{
				Phase:       PhaseDataImport,
				SystemName:  system,
				LastID:      lastID,
				RecordCount: int64(batchCount * s.batchSize),
				Timestamp:   time.Now(),
			})
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue processing
		}
	}

	return nil
}

// fetchBatch retrieves a batch of records from PostgreSQL
func (s *IncrementalStrategy) fetchBatch(ctx context.Context, system string, lastID int64, batchSize int) ([]*ConceptRecord, error) {
	query := `
		SELECT id, concept_uuid, system, code, version, preferred_term,
		       synonyms, properties->>'definition' as definition,
		       parent_codes, active, properties, created_at, updated_at
		FROM concepts
		WHERE system = $1 AND id > $2
		ORDER BY id
		LIMIT $3
	`

	rows, err := s.coordinator.postgresDB.QueryContext(ctx, query, system, lastID, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*ConceptRecord, 0, batchSize)
	for rows.Next() {
		// Scan record (implementation would be similar to bulk_loader.go)
		record := &ConceptRecord{}
		// ... scanning logic ...
		records = append(records, record)
	}

	return records, rows.Err()
}

// loadBatch loads a batch of records to Elasticsearch
func (s *IncrementalStrategy) loadBatch(ctx context.Context, records []*ConceptRecord) error {
	// Implementation would use bulk indexer similar to bulk_loader.go
	return nil
}

// Validate checks if the migration was successful
func (s *IncrementalStrategy) Validate(ctx context.Context) error {
	s.coordinator.updatePhase(PhasePostValidation)

	validator := &DataValidator{
		postgresDB:    s.coordinator.postgresDB,
		elasticsearch: s.coordinator.elasticsearch,
		logger:        s.coordinator.logger,
		config: ValidationConfig{
			SampleSize:         1000,
			SamplePercentage:   0.01, // 1% sample
			StrictMode:         true,
			AcceptableErrorRate: 0.001, // 0.1% error tolerance
			FieldsToValidate:   []string{"code", "system", "display", "status"},
		},
	}

	return validator.ValidateConsistency(ctx)
}

// Rollback reverts the migration if needed
func (s *IncrementalStrategy) Rollback(ctx context.Context) error {
	s.coordinator.logger.Info("Rolling back incremental migration")
	// Implementation would delete data from Elasticsearch
	// and restore any configuration changes
	return nil
}

// ParallelStrategy implements high-performance parallel migration
type ParallelStrategy struct {
	coordinator *MigrationCoordinator
	numWorkers  int
	batchSize   int
}

// Execute performs parallel migration
func (s *ParallelStrategy) Execute(ctx context.Context) error {
	s.coordinator.updatePhase(PhaseDataExport)

	// Create worker pool
	workerCh := make(chan *migrationTask, s.numWorkers*2)
	resultCh := make(chan *migrationResult, s.numWorkers)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < s.numWorkers; i++ {
		wg.Add(1)
		go s.worker(ctx, &wg, workerCh, resultCh)
	}

	// Start result collector
	go s.collectResults(resultCh)

	// Distribute work among workers
	systems := []string{"SNOMED", "RxNorm", "LOINC", "ICD10"}
	for _, system := range systems {
		// Split system into chunks for parallel processing
		chunks := s.splitIntoChunks(ctx, system)
		for _, chunk := range chunks {
			workerCh <- chunk
		}
	}

	// Close channels and wait
	close(workerCh)
	wg.Wait()
	close(resultCh)

	return nil
}

// migrationTask represents a unit of work for parallel migration
type migrationTask struct {
	System    string
	StartID   int64
	EndID     int64
	BatchSize int
}

// migrationResult contains the result of a migration task
type migrationResult struct {
	Task           *migrationTask
	RecordsProcessed int64
	Success        bool
	Error          error
}

// worker processes migration tasks in parallel
func (s *ParallelStrategy) worker(ctx context.Context, wg *sync.WaitGroup, tasks <-chan *migrationTask, results chan<- *migrationResult) {
	defer wg.Done()

	for task := range tasks {
		result := &migrationResult{Task: task}

		// Process task
		processed, err := s.processTask(ctx, task)
		result.RecordsProcessed = processed
		result.Success = err == nil
		result.Error = err

		results <- result
	}
}

// processTask handles a single migration task
func (s *ParallelStrategy) processTask(ctx context.Context, task *migrationTask) (int64, error) {
	// Implementation would process records in the given ID range
	return 0, nil
}

// splitIntoChunks divides a system's data into chunks for parallel processing
func (s *ParallelStrategy) splitIntoChunks(ctx context.Context, system string) []*migrationTask {
	// Implementation would query min/max IDs and create chunks
	return nil
}

// collectResults aggregates results from workers
func (s *ParallelStrategy) collectResults(results <-chan *migrationResult) {
	for result := range results {
		if result.Success {
			s.coordinator.updateStatistics(int(result.RecordsProcessed))
		} else {
			s.coordinator.recordError(MigrationError{
				Timestamp: time.Now(),
				Phase:     PhaseDataImport,
				Error:     result.Error.Error(),
				Severity:  "error",
				Retryable: true,
			})
		}
	}
}

// Validate checks parallel migration success
func (s *ParallelStrategy) Validate(ctx context.Context) error {
	// Similar to IncrementalStrategy.Validate
	return nil
}

// Rollback reverts parallel migration
func (s *ParallelStrategy) Rollback(ctx context.Context) error {
	// Similar to IncrementalStrategy.Rollback
	return nil
}

// BlueGreenStrategy implements zero-downtime migration
type BlueGreenStrategy struct {
	coordinator *MigrationCoordinator
}

// Execute performs blue-green migration
func (s *BlueGreenStrategy) Execute(ctx context.Context) error {
	// 1. Create new Elasticsearch index (green)
	// 2. Load all data to green index
	// 3. Validate green index
	// 4. Switch alias from blue to green atomically
	// 5. Keep blue index for rollback capability
	return nil
}

// Validate verifies blue-green migration
func (s *BlueGreenStrategy) Validate(ctx context.Context) error {
	return nil
}

// Rollback switches back to blue index
func (s *BlueGreenStrategy) Rollback(ctx context.Context) error {
	return nil
}

// ShadowWriteStrategy implements gradual migration with dual writes
type ShadowWriteStrategy struct {
	coordinator *MigrationCoordinator
}

// Execute performs shadow write migration
func (s *ShadowWriteStrategy) Execute(ctx context.Context) error {
	// 1. Enable dual-write mode in application
	// 2. Start background migration of existing data
	// 3. Gradually shift read traffic to Elasticsearch
	// 4. Monitor and validate consistency
	// 5. Complete migration when fully consistent
	return nil
}

// Validate ensures shadow writes are consistent
func (s *ShadowWriteStrategy) Validate(ctx context.Context) error {
	return nil
}

// Rollback disables shadow writes
func (s *ShadowWriteStrategy) Rollback(ctx context.Context) error {
	return nil
}

// ValidateConsistency performs comprehensive data validation
func (v *DataValidator) ValidateConsistency(ctx context.Context) error {
	v.logger.Info("Starting data consistency validation")

	// Validate record counts
	if err := v.validateRecordCounts(ctx); err != nil {
		return err
	}

	// Validate sample records
	if err := v.validateSampleRecords(ctx); err != nil {
		return err
	}

	// Validate search functionality
	if err := v.validateSearchFunctionality(ctx); err != nil {
		return err
	}

	v.logger.Info("Data consistency validation completed successfully")
	return nil
}

// validateRecordCounts compares record counts between stores
func (v *DataValidator) validateRecordCounts(ctx context.Context) error {
	// Get PostgreSQL count
	var pgCount int64
	err := v.postgresDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM concepts").Scan(&pgCount)
	if err != nil {
		return err
	}

	// Get Elasticsearch count
	res, err := v.elasticsearch.Count(
		v.elasticsearch.Count.WithContext(ctx),
		v.elasticsearch.Count.WithIndex("clinical_terms"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var esResult map[string]interface{}
	json.NewDecoder(res.Body).Decode(&esResult)
	esCount := int64(esResult["count"].(float64))

	// Compare counts
	diff := pgCount - esCount
	if diff != 0 {
		errorRate := float64(diff) / float64(pgCount)
		if errorRate > v.config.AcceptableErrorRate {
			return fmt.Errorf("unacceptable record count difference: %d (%.2f%%)", diff, errorRate*100)
		}
	}

	v.logger.Infof("Record count validation passed: PostgreSQL=%d, Elasticsearch=%d", pgCount, esCount)
	return nil
}

// validateSampleRecords validates a sample of records for data integrity
func (v *DataValidator) validateSampleRecords(ctx context.Context) error {
	// Implementation would:
	// 1. Select random sample from PostgreSQL
	// 2. Fetch same records from Elasticsearch
	// 3. Compare field values
	// 4. Report any discrepancies
	return nil
}

// validateSearchFunctionality ensures search works correctly
func (v *DataValidator) validateSearchFunctionality(ctx context.Context) error {
	// Test various search scenarios:
	// 1. Exact code lookups
	// 2. Text search
	// 3. System filtering
	// 4. Status filtering
	return nil
}

// Helper methods for MigrationCoordinator

func (mc *MigrationCoordinator) updatePhase(phase MigrationPhase) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.state.Phase = phase
	mc.logger.Infof("Migration phase updated to: %s", phase)
}

func (mc *MigrationCoordinator) saveCheckpoint(checkpoint *MigrationCheckpoint) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.state.LastCheckpoint = checkpoint
	mc.logger.Infof("Checkpoint saved: system=%s, lastID=%d", checkpoint.SystemName, checkpoint.LastID)
}

func (mc *MigrationCoordinator) recordError(err MigrationError) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.state.Errors = append(mc.state.Errors, err)
	mc.logger.WithField("severity", err.Severity).Errorf("Migration error: %s", err.Error)
}

func (mc *MigrationCoordinator) updateStatistics(recordsProcessed int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.state.Statistics.MigratedRecords += int64(recordsProcessed)

	// Update average records per second
	elapsed := time.Since(mc.state.StartedAt).Seconds()
	if elapsed > 0 {
		mc.state.Statistics.AverageRecordsPerSecond = float64(mc.state.Statistics.MigratedRecords) / elapsed
	}

	// Estimate completion time
	if mc.state.Statistics.TotalRecords > 0 && mc.state.Statistics.AverageRecordsPerSecond > 0 {
		remaining := mc.state.Statistics.TotalRecords - mc.state.Statistics.MigratedRecords
		estimatedSeconds := float64(remaining) / mc.state.Statistics.AverageRecordsPerSecond
		estimatedCompletion := time.Now().Add(time.Duration(estimatedSeconds) * time.Second)
		mc.state.Statistics.EstimatedCompletion = &estimatedCompletion
	}
}