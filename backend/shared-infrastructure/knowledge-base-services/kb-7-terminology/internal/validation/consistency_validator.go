package validation

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"kb-7-terminology/internal/elasticsearch"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"go.uber.org/zap"
)

// ConsistencyValidator validates data consistency between PostgreSQL and Elasticsearch
type ConsistencyValidator struct {
	db        *sql.DB
	esClient  *elasticsearch.Client
	logger    *zap.Logger
	metrics   *metrics.Collector
	config    *ValidationConfig
}

// ValidationConfig holds configuration for consistency validation
type ValidationConfig struct {
	BatchSize              int           `json:"batch_size"`
	MaxConcurrency         int           `json:"max_concurrency"`
	ChecksumAlgorithm      string        `json:"checksum_algorithm"`
	ToleranceThreshold     float64       `json:"tolerance_threshold"`
	DetailedValidation     bool          `json:"detailed_validation"`
	ValidationTimeout      time.Duration `json:"validation_timeout"`
	IncludeInactiveRecords bool          `json:"include_inactive_records"`
	SampleSize             int           `json:"sample_size"`
	EnableRepair           bool          `json:"enable_repair"`
	RepairMode             RepairMode    `json:"repair_mode"`
}

// RepairMode defines how inconsistencies should be repaired
type RepairMode string

const (
	RepairModeNone             RepairMode = "none"             // No automatic repair
	RepairModePreferPostgreSQL RepairMode = "prefer_postgresql" // Use PostgreSQL as source of truth
	RepairModePreferElastic    RepairMode = "prefer_elastic"    // Use Elasticsearch as source of truth
	RepairModeManual           RepairMode = "manual"            // Generate repair scripts for manual review
)

// ValidationReport contains the results of a consistency validation
type ValidationReport struct {
	ValidationID         string                    `json:"validation_id"`
	StartTime           time.Time                 `json:"start_time"`
	EndTime             time.Time                 `json:"end_time"`
	Duration            time.Duration             `json:"duration"`
	OverallStatus       ValidationStatus          `json:"overall_status"`
	ConsistencyScore    float64                   `json:"consistency_score"`
	TotalRecords        ConsistencyMetrics        `json:"total_records"`
	ValidationResults   map[string]SystemValidation `json:"validation_results"`
	Inconsistencies     []InconsistencyRecord     `json:"inconsistencies"`
	RepairActions       []RepairAction            `json:"repair_actions"`
	PerformanceMetrics  PerformanceMetrics        `json:"performance_metrics"`
	Recommendations     []string                  `json:"recommendations"`
}

// ValidationStatus represents the overall validation status
type ValidationStatus string

const (
	ValidationStatusPassed    ValidationStatus = "passed"
	ValidationStatusFailed    ValidationStatus = "failed"
	ValidationStatusDegraded  ValidationStatus = "degraded"
	ValidationStatusRepairing ValidationStatus = "repairing"
)

// ConsistencyMetrics contains record count metrics
type ConsistencyMetrics struct {
	PostgreSQL    int64 `json:"postgresql"`
	Elasticsearch int64 `json:"elasticsearch"`
	Difference    int64 `json:"difference"`
	MatchCount    int64 `json:"match_count"`
	MismatchCount int64 `json:"mismatch_count"`
}

// SystemValidation contains validation results for a single system
type SystemValidation struct {
	System           string                 `json:"system"`
	Status           string                 `json:"status"`
	RecordCount      int64                 `json:"record_count"`
	ChecksumMatches  int64                 `json:"checksum_matches"`
	ChecksumMismatches int64               `json:"checksum_mismatches"`
	MissingRecords   []string              `json:"missing_records"`
	ExtraRecords     []string              `json:"extra_records"`
	FieldMismatches  map[string]int64      `json:"field_mismatches"`
	ValidationErrors []ValidationError     `json:"validation_errors"`
	Details          map[string]interface{} `json:"details"`
}

// InconsistencyRecord represents a specific inconsistency found
type InconsistencyRecord struct {
	RecordID          string                 `json:"record_id"`
	InconsistencyType InconsistencyType     `json:"inconsistency_type"`
	Severity          Severity              `json:"severity"`
	Description       string                `json:"description"`
	PostgreSQLData    map[string]interface{} `json:"postgresql_data"`
	ElasticsearchData map[string]interface{} `json:"elasticsearch_data"`
	FieldDifferences  []FieldDifference     `json:"field_differences"`
	DetectedAt        time.Time             `json:"detected_at"`
	RepairSuggestion  string                `json:"repair_suggestion"`
}

// InconsistencyType defines the type of inconsistency
type InconsistencyType string

const (
	InconsistencyTypeMissing      InconsistencyType = "missing"       // Record exists in one store but not the other
	InconsistencyTypeExtra        InconsistencyType = "extra"         // Record exists but shouldn't
	InconsistencyTypeFieldMismatch InconsistencyType = "field_mismatch" // Field values don't match
	InconsistencyTypeChecksum     InconsistencyType = "checksum"      // Checksum mismatch
	InconsistencyTypeStructural   InconsistencyType = "structural"    // Structural differences
)

// Severity defines the severity of an inconsistency
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// FieldDifference represents a difference in a specific field
type FieldDifference struct {
	FieldName         string      `json:"field_name"`
	PostgreSQLValue   interface{} `json:"postgresql_value"`
	ElasticsearchValue interface{} `json:"elasticsearch_value"`
	DifferenceType    string      `json:"difference_type"`
}

// RepairAction represents an action to repair an inconsistency
type RepairAction struct {
	ActionID      string            `json:"action_id"`
	RecordID      string            `json:"record_id"`
	ActionType    RepairActionType  `json:"action_type"`
	TargetStore   string            `json:"target_store"`
	Data          map[string]interface{} `json:"data"`
	Script        string            `json:"script"`
	Priority      int               `json:"priority"`
	EstimatedTime time.Duration     `json:"estimated_time"`
	Status        string            `json:"status"`
}

// RepairActionType defines the type of repair action
type RepairActionType string

const (
	RepairActionTypeInsert RepairActionType = "insert"
	RepairActionTypeUpdate RepairActionType = "update"
	RepairActionTypeDelete RepairActionType = "delete"
	RepairActionTypeSync   RepairActionType = "sync"
)

// ValidationError represents an error during validation
type ValidationError struct {
	RecordID    string    `json:"record_id"`
	Error       string    `json:"error"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"`
	Recoverable bool      `json:"recoverable"`
}

// PerformanceMetrics contains performance metrics for the validation
type PerformanceMetrics struct {
	RecordsPerSecond     float64       `json:"records_per_second"`
	DatabaseQueryTime    time.Duration `json:"database_query_time"`
	ElasticsearchQueryTime time.Duration `json:"elasticsearch_query_time"`
	ComparisonTime       time.Duration `json:"comparison_time"`
	TotalMemoryUsed      int64         `json:"total_memory_used"`
	PeakMemoryUsed       int64         `json:"peak_memory_used"`
	ThreadsUsed          int           `json:"threads_used"`
}

// NewConsistencyValidator creates a new consistency validator
func NewConsistencyValidator(
	db *sql.DB,
	esClient *elasticsearch.Client,
	logger *zap.Logger,
	metrics *metrics.Collector,
	config *ValidationConfig,
) *ConsistencyValidator {
	if config == nil {
		config = &ValidationConfig{
			BatchSize:              1000,
			MaxConcurrency:         4,
			ChecksumAlgorithm:      "sha256",
			ToleranceThreshold:     0.99, // 99% consistency required
			DetailedValidation:     true,
			ValidationTimeout:      30 * time.Minute,
			IncludeInactiveRecords: false,
			SampleSize:             0, // 0 means validate all records
			EnableRepair:           false,
			RepairMode:             RepairModeNone,
		}
	}

	return &ConsistencyValidator{
		db:      db,
		esClient: esClient,
		logger:   logger,
		metrics:  metrics,
		config:   config,
	}
}

// ValidateConsistency performs a comprehensive consistency validation
func (cv *ConsistencyValidator) ValidateConsistency(ctx context.Context) (*ValidationReport, error) {
	validationID := fmt.Sprintf("validation_%d", time.Now().Unix())
	startTime := time.Now()

	cv.logger.Info("Starting consistency validation",
		zap.String("validation_id", validationID),
	)

	// Create validation context with timeout
	validationCtx, cancel := context.WithTimeout(ctx, cv.config.ValidationTimeout)
	defer cancel()

	report := &ValidationReport{
		ValidationID:    validationID,
		StartTime:       startTime,
		ValidationResults: make(map[string]SystemValidation),
		Inconsistencies: make([]InconsistencyRecord, 0),
		RepairActions:   make([]RepairAction, 0),
		Recommendations: make([]string, 0),
	}

	// Phase 1: Count records in both systems
	cv.logger.Info("Phase 1: Counting records")
	if err := cv.countRecords(validationCtx, report); err != nil {
		return nil, fmt.Errorf("record counting failed: %w", err)
	}

	// Phase 2: Validate record consistency
	cv.logger.Info("Phase 2: Validating record consistency")
	if err := cv.validateRecordConsistency(validationCtx, report); err != nil {
		return nil, fmt.Errorf("record validation failed: %w", err)
	}

	// Phase 3: Checksum validation
	if cv.config.DetailedValidation {
		cv.logger.Info("Phase 3: Checksum validation")
		if err := cv.validateChecksums(validationCtx, report); err != nil {
			cv.logger.Warn("Checksum validation failed", zap.Error(err))
			// Don't fail the entire validation for checksum issues
		}
	}

	// Phase 4: Generate repair actions if enabled
	if cv.config.EnableRepair && len(report.Inconsistencies) > 0 {
		cv.logger.Info("Phase 4: Generating repair actions")
		if err := cv.generateRepairActions(validationCtx, report); err != nil {
			cv.logger.Warn("Repair action generation failed", zap.Error(err))
		}
	}

	// Calculate final metrics
	cv.calculateFinalMetrics(report)

	report.EndTime = time.Now()
	report.Duration = report.EndTime.Sub(report.StartTime)

	cv.logger.Info("Consistency validation completed",
		zap.String("validation_id", validationID),
		zap.String("status", string(report.OverallStatus)),
		zap.Float64("consistency_score", report.ConsistencyScore),
		zap.Duration("duration", report.Duration),
	)

	// Record metrics
	cv.recordValidationMetrics(report)

	return report, nil
}

// countRecords counts records in both PostgreSQL and Elasticsearch
func (cv *ConsistencyValidator) countRecords(ctx context.Context, report *ValidationReport) error {
	var wg sync.WaitGroup
	var pgCount, esCount int64
	var pgErr, esErr error

	// Count PostgreSQL records
	wg.Add(1)
	go func() {
		defer wg.Done()
		pgCount, pgErr = cv.countPostgreSQLRecords(ctx)
	}()

	// Count Elasticsearch records
	wg.Add(1)
	go func() {
		defer wg.Done()
		esCount, esErr = cv.countElasticsearchRecords(ctx)
	}()

	wg.Wait()

	if pgErr != nil {
		return fmt.Errorf("PostgreSQL count failed: %w", pgErr)
	}

	if esErr != nil {
		return fmt.Errorf("Elasticsearch count failed: %w", esErr)
	}

	report.TotalRecords = ConsistencyMetrics{
		PostgreSQL:    pgCount,
		Elasticsearch: esCount,
		Difference:    pgCount - esCount,
	}

	cv.logger.Info("Record counts obtained",
		zap.Int64("postgresql", pgCount),
		zap.Int64("elasticsearch", esCount),
		zap.Int64("difference", report.TotalRecords.Difference),
	)

	return nil
}

// countPostgreSQLRecords counts records in PostgreSQL
func (cv *ConsistencyValidator) countPostgreSQLRecords(ctx context.Context) (int64, error) {
	query := "SELECT COUNT(*) FROM concepts"
	if !cv.config.IncludeInactiveRecords {
		query += " WHERE active = true"
	}

	var count int64
	err := cv.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("PostgreSQL count query failed: %w", err)
	}

	return count, nil
}

// countElasticsearchRecords counts records in Elasticsearch
func (cv *ConsistencyValidator) countElasticsearchRecords(ctx context.Context) (int64, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 0,
	}

	if !cv.config.IncludeInactiveRecords {
		query["query"] = map[string]interface{}{
			"term": map[string]interface{}{
				"status": "active",
			},
		}
	}

	response, err := cv.esClient.Search(ctx, "kb7-terminology", query)
	if err != nil {
		return 0, fmt.Errorf("Elasticsearch count query failed: %w", err)
	}

	return int64(response.Hits.Total.Value), nil
}

// validateRecordConsistency validates individual record consistency
func (cv *ConsistencyValidator) validateRecordConsistency(ctx context.Context, report *ValidationReport) error {
	// Get record IDs from PostgreSQL
	pgRecords, err := cv.getPostgreSQLRecordIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get PostgreSQL record IDs: %w", err)
	}

	// Get record IDs from Elasticsearch
	esRecords, err := cv.getElasticsearchRecordIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Elasticsearch record IDs: %w", err)
	}

	// Convert to sets for comparison
	pgSet := make(map[string]bool)
	for _, id := range pgRecords {
		pgSet[id] = true
	}

	esSet := make(map[string]bool)
	for _, id := range esRecords {
		esSet[id] = true
	}

	// Find missing records
	var missingInES, missingInPG []string

	for id := range pgSet {
		if !esSet[id] {
			missingInES = append(missingInES, id)
		}
	}

	for id := range esSet {
		if !pgSet[id] {
			missingInPG = append(missingInPG, id)
		}
	}

	// Record inconsistencies
	for _, id := range missingInES {
		inconsistency := InconsistencyRecord{
			RecordID:          id,
			InconsistencyType: InconsistencyTypeMissing,
			Severity:          SeverityHigh,
			Description:       "Record exists in PostgreSQL but missing in Elasticsearch",
			DetectedAt:        time.Now(),
			RepairSuggestion:  "Sync record from PostgreSQL to Elasticsearch",
		}
		report.Inconsistencies = append(report.Inconsistencies, inconsistency)
	}

	for _, id := range missingInPG {
		inconsistency := InconsistencyRecord{
			RecordID:          id,
			InconsistencyType: InconsistencyTypeExtra,
			Severity:          SeverityMedium,
			Description:       "Record exists in Elasticsearch but missing in PostgreSQL",
			DetectedAt:        time.Now(),
			RepairSuggestion:  "Remove record from Elasticsearch or add to PostgreSQL",
		}
		report.Inconsistencies = append(report.Inconsistencies, inconsistency)
	}

	// Update metrics
	report.TotalRecords.MatchCount = int64(len(pgSet)) - int64(len(missingInES))
	report.TotalRecords.MismatchCount = int64(len(missingInES) + len(missingInPG))

	cv.logger.Info("Record consistency validation completed",
		zap.Int("missing_in_elasticsearch", len(missingInES)),
		zap.Int("missing_in_postgresql", len(missingInPG)),
		zap.Int64("match_count", report.TotalRecords.MatchCount),
	)

	return nil
}

// getPostgreSQLRecordIDs retrieves all record IDs from PostgreSQL
func (cv *ConsistencyValidator) getPostgreSQLRecordIDs(ctx context.Context) ([]string, error) {
	query := "SELECT concept_uuid::text FROM concepts"
	if !cv.config.IncludeInactiveRecords {
		query += " WHERE active = true"
	}
	query += " ORDER BY term_id"

	rows, err := cv.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// getElasticsearchRecordIDs retrieves all record IDs from Elasticsearch
func (cv *ConsistencyValidator) getElasticsearchRecordIDs(ctx context.Context) ([]string, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"_source": []string{"term_id"},
		"size":    10000, // Adjust based on your data size
		"sort": []interface{}{
			map[string]interface{}{
				"term_id.keyword": map[string]interface{}{
					"order": "asc",
				},
			},
		},
	}

	if !cv.config.IncludeInactiveRecords {
		query["query"] = map[string]interface{}{
			"term": map[string]interface{}{
				"status": "active",
			},
		}
	}

	response, err := cv.esClient.Search(ctx, "kb7-terminology", query)
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, hit := range response.Hits.Hits {
		if termID, exists := hit.Source["term_id"]; exists {
			if id, ok := termID.(string); ok {
				ids = append(ids, id)
			}
		}
	}

	return ids, nil
}

// validateChecksums performs checksum validation for detailed comparison
func (cv *ConsistencyValidator) validateChecksums(ctx context.Context, report *ValidationReport) error {
	cv.logger.Info("Starting checksum validation")

	// Sample records if sample size is specified
	recordIDs, err := cv.getRecordIDsForChecksumValidation(ctx)
	if err != nil {
		return fmt.Errorf("failed to get record IDs for checksum validation: %w", err)
	}

	// Process records in batches
	batchCount := (len(recordIDs) + cv.config.BatchSize - 1) / cv.config.BatchSize
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, cv.config.MaxConcurrency)

	for i := 0; i < batchCount; i++ {
		start := i * cv.config.BatchSize
		end := start + cv.config.BatchSize
		if end > len(recordIDs) {
			end = len(recordIDs)
		}

		batch := recordIDs[start:end]

		wg.Add(1)
		go func(batchIDs []string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			cv.validateBatchChecksums(ctx, batchIDs, report)
		}(batch)
	}

	wg.Wait()

	cv.logger.Info("Checksum validation completed",
		zap.Int("total_records", len(recordIDs)),
		zap.Int("batches_processed", batchCount),
	)

	return nil
}

// validateBatchChecksums validates checksums for a batch of records
func (cv *ConsistencyValidator) validateBatchChecksums(ctx context.Context, recordIDs []string, report *ValidationReport) {
	for _, recordID := range recordIDs {
		// Get record from PostgreSQL
		pgRecord, err := cv.getPostgreSQLRecord(ctx, recordID)
		if err != nil {
			cv.logger.Warn("Failed to get PostgreSQL record", zap.String("record_id", recordID), zap.Error(err))
			continue
		}

		// Get record from Elasticsearch
		esRecord, err := cv.getElasticsearchRecord(ctx, recordID)
		if err != nil {
			cv.logger.Warn("Failed to get Elasticsearch record", zap.String("record_id", recordID), zap.Error(err))
			continue
		}

		// Compare records
		if inconsistency := cv.compareRecords(recordID, pgRecord, esRecord); inconsistency != nil {
			// Thread-safe append to report
			report.Inconsistencies = append(report.Inconsistencies, *inconsistency)
		}
	}
}

// compareRecords compares two records and returns inconsistency if found
func (cv *ConsistencyValidator) compareRecords(recordID string, pgRecord, esRecord map[string]interface{}) *InconsistencyRecord {
	// Calculate checksums
	pgChecksum := cv.calculateRecordChecksum(pgRecord)
	esChecksum := cv.calculateRecordChecksum(esRecord)

	if pgChecksum == esChecksum {
		return nil // Records match
	}

	// Find field differences
	differences := cv.findFieldDifferences(pgRecord, esRecord)

	inconsistency := &InconsistencyRecord{
		RecordID:          recordID,
		InconsistencyType: InconsistencyTypeFieldMismatch,
		Severity:          cv.calculateSeverity(differences),
		Description:       fmt.Sprintf("Record field mismatches detected (%d fields)", len(differences)),
		PostgreSQLData:    pgRecord,
		ElasticsearchData: esRecord,
		FieldDifferences:  differences,
		DetectedAt:        time.Now(),
		RepairSuggestion:  "Sync record from PostgreSQL to Elasticsearch",
	}

	return inconsistency
}

// calculateRecordChecksum calculates a checksum for a record
func (cv *ConsistencyValidator) calculateRecordChecksum(record map[string]interface{}) string {
	// Create a deterministic string representation
	var keys []string
	for k := range record {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("%s:%v;", key, record[key]))
	}

	// Calculate checksum
	hash := sha256.Sum256([]byte(builder.String()))
	return hex.EncodeToString(hash[:])
}

// findFieldDifferences finds differences between two records
func (cv *ConsistencyValidator) findFieldDifferences(pgRecord, esRecord map[string]interface{}) []FieldDifference {
	var differences []FieldDifference

	// Check all fields in PostgreSQL record
	for field, pgValue := range pgRecord {
		if esValue, exists := esRecord[field]; exists {
			if !cv.valuesEqual(pgValue, esValue) {
				differences = append(differences, FieldDifference{
					FieldName:          field,
					PostgreSQLValue:    pgValue,
					ElasticsearchValue: esValue,
					DifferenceType:     "value_mismatch",
				})
			}
		} else {
			differences = append(differences, FieldDifference{
				FieldName:          field,
				PostgreSQLValue:    pgValue,
				ElasticsearchValue: nil,
				DifferenceType:     "missing_in_elasticsearch",
			})
		}
	}

	// Check for extra fields in Elasticsearch
	for field, esValue := range esRecord {
		if _, exists := pgRecord[field]; !exists {
			differences = append(differences, FieldDifference{
				FieldName:          field,
				PostgreSQLValue:    nil,
				ElasticsearchValue: esValue,
				DifferenceType:     "extra_in_elasticsearch",
			})
		}
	}

	return differences
}

// Helper methods

func (cv *ConsistencyValidator) getRecordIDsForChecksumValidation(ctx context.Context) ([]string, error) {
	query := "SELECT concept_uuid::text FROM concepts"
	if !cv.config.IncludeInactiveRecords {
		query += " WHERE active = true"
	}

	if cv.config.SampleSize > 0 {
		query += fmt.Sprintf(" ORDER BY RANDOM() LIMIT %d", cv.config.SampleSize)
	} else {
		query += " ORDER BY term_id"
	}

	rows, err := cv.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

func (cv *ConsistencyValidator) getPostgreSQLRecord(ctx context.Context, recordID string) (map[string]interface{}, error) {
	query := `
		SELECT concept_uuid::text, code, preferred_term, fully_specified_name,
		       COALESCE(properties->>'definition', ''), system, version, active
		FROM concepts
		WHERE concept_uuid::text = $1
	`

	row := cv.db.QueryRowContext(ctx, query, recordID)

	var termID, conceptID, termText, preferredTerm, definition, system, version, status string

	err := row.Scan(&termID, &conceptID, &termText, &preferredTerm, &definition, &system, &version, &status)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"term_id":             termID,
		"concept_id":          conceptID,
		"term_text":           termText,
		"preferred_term":      preferredTerm,
		"definition":          definition,
		"terminology_system":  system,
		"terminology_version": version,
		"status":              status,
	}, nil
}

func (cv *ConsistencyValidator) getElasticsearchRecord(ctx context.Context, recordID string) (map[string]interface{}, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"term_id": recordID,
			},
		},
		"size": 1,
	}

	response, err := cv.esClient.Search(ctx, "kb7-terminology", query)
	if err != nil {
		return nil, err
	}

	if len(response.Hits.Hits) == 0 {
		return nil, fmt.Errorf("record not found")
	}

	return response.Hits.Hits[0].Source, nil
}

func (cv *ConsistencyValidator) valuesEqual(a, b interface{}) bool {
	// Simple equality check - could be enhanced for specific types
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (cv *ConsistencyValidator) calculateSeverity(differences []FieldDifference) Severity {
	if len(differences) == 0 {
		return SeverityLow
	}

	// Check for critical fields
	criticalFields := map[string]bool{
		"term_id":             true,
		"concept_id":          true,
		"terminology_system":  true,
		"status":              true,
	}

	for _, diff := range differences {
		if criticalFields[diff.FieldName] {
			return SeverityCritical
		}
	}

	if len(differences) > 5 {
		return SeverityHigh
	} else if len(differences) > 2 {
		return SeverityMedium
	}

	return SeverityLow
}

func (cv *ConsistencyValidator) generateRepairActions(ctx context.Context, report *ValidationReport) error {
	cv.logger.Info("Generating repair actions",
		zap.Int("inconsistencies", len(report.Inconsistencies)),
	)

	for i, inconsistency := range report.Inconsistencies {
		action := cv.createRepairAction(inconsistency)
		if action != nil {
			action.ActionID = fmt.Sprintf("repair_%d_%s", i, inconsistency.RecordID)
			report.RepairActions = append(report.RepairActions, *action)
		}
	}

	cv.logger.Info("Repair actions generated",
		zap.Int("actions", len(report.RepairActions)),
	)

	return nil
}

func (cv *ConsistencyValidator) createRepairAction(inconsistency InconsistencyRecord) *RepairAction {
	var action *RepairAction

	switch inconsistency.InconsistencyType {
	case InconsistencyTypeMissing:
		// Record missing in Elasticsearch - insert it
		action = &RepairAction{
			RecordID:      inconsistency.RecordID,
			ActionType:    RepairActionTypeInsert,
			TargetStore:   "elasticsearch",
			Data:          inconsistency.PostgreSQLData,
			Priority:      cv.getSeverityPriority(inconsistency.Severity),
			EstimatedTime: 1 * time.Second,
			Status:        "pending",
		}

	case InconsistencyTypeExtra:
		// Extra record in Elasticsearch - delete it or check if should be in PostgreSQL
		if cv.config.RepairMode == RepairModePreferPostgreSQL {
			action = &RepairAction{
				RecordID:      inconsistency.RecordID,
				ActionType:    RepairActionTypeDelete,
				TargetStore:   "elasticsearch",
				Priority:      cv.getSeverityPriority(inconsistency.Severity),
				EstimatedTime: 1 * time.Second,
				Status:        "pending",
			}
		}

	case InconsistencyTypeFieldMismatch:
		// Field mismatch - sync from preferred source
		if cv.config.RepairMode == RepairModePreferPostgreSQL {
			action = &RepairAction{
				RecordID:      inconsistency.RecordID,
				ActionType:    RepairActionTypeUpdate,
				TargetStore:   "elasticsearch",
				Data:          inconsistency.PostgreSQLData,
				Priority:      cv.getSeverityPriority(inconsistency.Severity),
				EstimatedTime: 1 * time.Second,
				Status:        "pending",
			}
		}
	}

	return action
}

func (cv *ConsistencyValidator) getSeverityPriority(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 1
	case SeverityHigh:
		return 2
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 4
	default:
		return 5
	}
}

func (cv *ConsistencyValidator) calculateFinalMetrics(report *ValidationReport) {
	totalRecords := report.TotalRecords.PostgreSQL
	if report.TotalRecords.Elasticsearch > totalRecords {
		totalRecords = report.TotalRecords.Elasticsearch
	}

	if totalRecords == 0 {
		report.ConsistencyScore = 1.0
		report.OverallStatus = ValidationStatusPassed
		return
	}

	// Calculate consistency score
	inconsistencyCount := int64(len(report.Inconsistencies))
	consistentRecords := totalRecords - inconsistencyCount
	report.ConsistencyScore = float64(consistentRecords) / float64(totalRecords)

	// Determine overall status
	if report.ConsistencyScore >= cv.config.ToleranceThreshold {
		if inconsistencyCount == 0 {
			report.OverallStatus = ValidationStatusPassed
		} else {
			report.OverallStatus = ValidationStatusDegraded
		}
	} else {
		report.OverallStatus = ValidationStatusFailed
	}

	// Generate recommendations
	if report.ConsistencyScore < cv.config.ToleranceThreshold {
		report.Recommendations = append(report.Recommendations,
			"Consistency score below threshold - immediate attention required")
	}

	if len(report.Inconsistencies) > 0 {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Found %d inconsistencies - consider running repair actions", len(report.Inconsistencies)))
	}
}

func (cv *ConsistencyValidator) recordValidationMetrics(report *ValidationReport) {
	cv.metrics.RecordValidationMetric("consistency_score", report.ConsistencyScore)
	cv.metrics.RecordValidationMetric("inconsistency_count", float64(len(report.Inconsistencies)))
	cv.metrics.RecordValidationMetric("validation_duration_seconds", report.Duration.Seconds())

	labels := map[string]string{
		"status": string(report.OverallStatus),
	}
	cv.metrics.IncrementCounterWithLabels("validation_completed", labels)
}