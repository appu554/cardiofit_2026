package bulkload

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics for monitoring bulk load progress
var (
	recordsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kb7_bulk_load_records_processed_total",
			Help: "Total number of records processed during bulk load",
		},
		[]string{"system", "status"},
	)

	bulkLoadDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "kb7_bulk_load_duration_seconds",
			Help: "Duration of bulk load operations",
			Buckets: prometheus.ExponentialBuckets(1, 2, 15), // 1s to ~32k seconds
		},
		[]string{"operation"},
	)

	bulkLoadErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kb7_bulk_load_errors_total",
			Help: "Total number of errors during bulk load",
		},
		[]string{"error_type", "system"},
	)
)

// BulkLoader manages the migration of data from PostgreSQL to Elasticsearch
type BulkLoader struct {
	postgresDB    *sql.DB
	elasticsearch *elasticsearch.Client
	logger        *logrus.Logger
	config        *BulkLoadConfig
	stats         *LoadStatistics
	mu            sync.RWMutex
}

// BulkLoadConfig contains configuration for bulk loading
type BulkLoadConfig struct {
	PostgresConnStr    string
	ElasticsearchURL   string
	ElasticsearchIndex string
	BatchSize          int
	NumWorkers         int
	FlushInterval      time.Duration
	MaxRetries         int
	RetryBackoff       time.Duration
	ValidateData       bool
	ResumeFromID       int64 // For resuming interrupted loads
	Systems            []string // Specific systems to load (empty = all)
}

// LoadStatistics tracks bulk load progress
type LoadStatistics struct {
	StartTime          time.Time
	EndTime            time.Time
	TotalRecords       int64
	ProcessedRecords   int64
	SuccessfulRecords  int64
	FailedRecords      int64
	ElasticsearchWrites int64
	ValidationErrors   int64
	CurrentSystem      string
	CurrentBatch       int64
	EstimatedCompletion time.Time
	Errors            []LoadError
}

// LoadError represents an error during loading
type LoadError struct {
	Timestamp time.Time
	System    string
	RecordID  int64
	Error     string
	Retryable bool
}

// ConceptRecord represents a clinical terminology concept
type ConceptRecord struct {
	ID              int64                  `json:"-"`
	ConceptUUID     string                 `json:"concept_uuid"`
	System          string                 `json:"system"`
	Code            string                 `json:"code"`
	Version         string                 `json:"version"`
	PreferredTerm   string                 `json:"display"`
	Synonyms        []string               `json:"synonyms,omitempty"`
	Definition      string                 `json:"definition,omitempty"`
	ParentCodes     []string               `json:"parent_codes,omitempty"`
	Active          bool                   `json:"status"`
	Properties      map[string]interface{} `json:"properties,omitempty"`
	Domain          string                 `json:"domain,omitempty"`
	SemanticType    string                 `json:"semantic_type,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	SearchText      string                 `json:"search_text,omitempty"`
}

// NewBulkLoader creates a new bulk loader instance
func NewBulkLoader(config *BulkLoadConfig, logger *logrus.Logger) (*BulkLoader, error) {
	// Initialize PostgreSQL connection
	db, err := sql.Open("postgres", config.PostgresConnStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.NumWorkers * 2)
	db.SetMaxIdleConns(config.NumWorkers)
	db.SetConnMaxLifetime(time.Hour)

	// Initialize Elasticsearch client
	esCfg := elasticsearch.Config{
		Addresses: []string{config.ElasticsearchURL},
		RetryOnStatus: []int{502, 503, 504, 429},
		RetryBackoff: func(i int) time.Duration {
			return time.Duration(i) * config.RetryBackoff
		},
		MaxRetries: config.MaxRetries,
	}

	esClient, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	loader := &BulkLoader{
		postgresDB:    db,
		elasticsearch: esClient,
		logger:        logger,
		config:        config,
		stats: &LoadStatistics{
			StartTime: time.Now(),
			Errors:    make([]LoadError, 0),
		},
	}

	return loader, nil
}

// ExecuteBulkLoad performs the complete bulk load operation
func (bl *BulkLoader) ExecuteBulkLoad(ctx context.Context) error {
	bl.logger.Info("Starting bulk load operation")
	defer bl.recordStats()

	// Step 1: Validate connections
	if err := bl.validateConnections(ctx); err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	// Step 2: Get total record count
	totalCount, err := bl.getTotalRecordCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get record count: %w", err)
	}
	bl.stats.TotalRecords = totalCount
	bl.logger.Infof("Total records to process: %d", totalCount)

	// Step 3: Process each terminology system
	systems := bl.config.Systems
	if len(systems) == 0 {
		systems = []string{"SNOMED", "RxNorm", "LOINC", "ICD10", "ICD9", "CPT", "NDC"}
	}

	for _, system := range systems {
		bl.mu.Lock()
		bl.stats.CurrentSystem = system
		bl.mu.Unlock()

		bl.logger.Infof("Processing system: %s", system)
		if err := bl.processSystem(ctx, system); err != nil {
			bl.logger.WithError(err).Errorf("Failed to process system %s", system)
			bl.recordError(LoadError{
				Timestamp: time.Now(),
				System:    system,
				Error:     err.Error(),
				Retryable: true,
			})

			// Continue with other systems even if one fails
			continue
		}
	}

	// Step 4: Validate data consistency if requested
	if bl.config.ValidateData {
		bl.logger.Info("Starting data consistency validation")
		if err := bl.validateDataConsistency(ctx); err != nil {
			bl.logger.WithError(err).Warn("Data consistency validation failed")
		}
	}

	// Step 5: Refresh Elasticsearch index for optimal search
	if err := bl.refreshElasticsearchIndex(ctx); err != nil {
		bl.logger.WithError(err).Warn("Failed to refresh Elasticsearch index")
	}

	bl.stats.EndTime = time.Now()
	return nil
}

// processSystem handles bulk loading for a specific terminology system
func (bl *BulkLoader) processSystem(ctx context.Context, system string) error {
	timer := prometheus.NewTimer(bulkLoadDuration.WithLabelValues(system))
	defer timer.ObserveDuration()

	// Create a worker pool for parallel processing
	workerCh := make(chan *ConceptRecord, bl.config.BatchSize*2)
	errorCh := make(chan error, bl.config.NumWorkers)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < bl.config.NumWorkers; i++ {
		wg.Add(1)
		go bl.worker(ctx, &wg, workerCh, errorCh)
	}

	// Start error collector
	go bl.collectErrors(errorCh, system)

	// Query and stream records from PostgreSQL
	query := bl.buildQuery(system)
	rows, err := bl.postgresDB.QueryContext(ctx, query, bl.config.ResumeFromID)
	if err != nil {
		return fmt.Errorf("failed to query system %s: %w", system, err)
	}
	defer rows.Close()

	// Process records
	batch := make([]*ConceptRecord, 0, bl.config.BatchSize)
	for rows.Next() {
		record, err := bl.scanRecord(rows)
		if err != nil {
			bl.logger.WithError(err).Warn("Failed to scan record")
			atomic.AddInt64(&bl.stats.FailedRecords, 1)
			continue
		}

		atomic.AddInt64(&bl.stats.ProcessedRecords, 1)
		batch = append(batch, record)

		// Send batch to workers when full
		if len(batch) >= bl.config.BatchSize {
			for _, r := range batch {
				select {
				case workerCh <- r:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			batch = batch[:0]
			atomic.AddInt64(&bl.stats.CurrentBatch, 1)
			bl.updateEstimatedCompletion()
		}
	}

	// Send remaining records
	for _, r := range batch {
		workerCh <- r
	}

	// Close channels and wait for workers
	close(workerCh)
	wg.Wait()
	close(errorCh)

	return rows.Err()
}

// worker processes records and sends them to Elasticsearch
func (bl *BulkLoader) worker(ctx context.Context, wg *sync.WaitGroup, records <-chan *ConceptRecord, errors chan<- error) {
	defer wg.Done()

	// Create bulk indexer for this worker
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:        bl.elasticsearch,
		Index:         bl.config.ElasticsearchIndex,
		NumWorkers:    1,
		FlushInterval: bl.config.FlushInterval,
		FlushBytes:    5 * 1024 * 1024, // 5MB
	})
	if err != nil {
		errors <- fmt.Errorf("failed to create bulk indexer: %w", err)
		return
	}
	defer bi.Close(ctx)

	for record := range records {
		// Prepare document for Elasticsearch
		doc, err := bl.prepareDocument(record)
		if err != nil {
			errors <- fmt.Errorf("failed to prepare document: %w", err)
			atomic.AddInt64(&bl.stats.FailedRecords, 1)
			continue
		}

		// Add to bulk indexer
		err = bi.Add(ctx, esutil.BulkIndexerItem{
			Action:     "index",
			DocumentID: fmt.Sprintf("%s_%s", record.System, record.Code),
			Body:       bytes.NewReader(doc),
			OnSuccess: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem) {
				atomic.AddInt64(&bl.stats.SuccessfulRecords, 1)
				atomic.AddInt64(&bl.stats.ElasticsearchWrites, 1)
				recordsProcessed.WithLabelValues(record.System, "success").Inc()
			},
			OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				atomic.AddInt64(&bl.stats.FailedRecords, 1)
				recordsProcessed.WithLabelValues(record.System, "failure").Inc()
				bulkLoadErrors.WithLabelValues("elasticsearch_write", record.System).Inc()

				bl.recordError(LoadError{
					Timestamp: time.Now(),
					System:    record.System,
					RecordID:  record.ID,
					Error:     fmt.Sprintf("ES error: %s", err),
					Retryable: true,
				})
			},
		})

		if err != nil {
			errors <- fmt.Errorf("failed to add to bulk indexer: %w", err)
			atomic.AddInt64(&bl.stats.FailedRecords, 1)
		}
	}
}

// buildQuery constructs the SQL query for fetching concepts
func (bl *BulkLoader) buildQuery(system string) string {
	return `
		SELECT
			id,
			concept_uuid,
			system,
			code,
			version,
			preferred_term,
			synonyms,
			COALESCE(properties->>'definition', '') as definition,
			parent_codes,
			active,
			properties,
			COALESCE(properties->>'domain', '') as domain,
			COALESCE(properties->>'semantic_type', '') as semantic_type,
			created_at,
			updated_at
		FROM concepts
		WHERE system = $1 AND id > $2
		ORDER BY id
	`
}

// scanRecord scans a database row into a ConceptRecord
func (bl *BulkLoader) scanRecord(rows *sql.Rows) (*ConceptRecord, error) {
	record := &ConceptRecord{}
	var synonyms pq.StringArray
	var parentCodes pq.StringArray
	var properties sql.NullString

	err := rows.Scan(
		&record.ID,
		&record.ConceptUUID,
		&record.System,
		&record.Code,
		&record.Version,
		&record.PreferredTerm,
		&synonyms,
		&record.Definition,
		&parentCodes,
		&record.Active,
		&properties,
		&record.Domain,
		&record.SemanticType,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	record.Synonyms = []string(synonyms)
	record.ParentCodes = []string(parentCodes)

	// Parse properties JSON
	if properties.Valid {
		var props map[string]interface{}
		if err := json.Unmarshal([]byte(properties.String), &props); err == nil {
			record.Properties = props
		}
	}

	// Create search text for better text search
	record.SearchText = bl.createSearchText(record)

	return record, nil
}

// prepareDocument converts a ConceptRecord to JSON for Elasticsearch
func (bl *BulkLoader) prepareDocument(record *ConceptRecord) ([]byte, error) {
	// Map active status to string
	status := "inactive"
	if record.Active {
		status = "active"
	}

	doc := map[string]interface{}{
		"code":           record.Code,
		"system":         record.System,
		"display":        record.PreferredTerm,
		"definition":     record.Definition,
		"synonyms":       record.Synonyms,
		"parent_codes":   record.ParentCodes,
		"status":         status,
		"domain":         record.Domain,
		"semantic_type":  record.SemanticType,
		"version":        record.Version,
		"properties":     record.Properties,
		"search_text":    record.SearchText,
		"created_at":     record.CreatedAt,
		"updated_at":     record.UpdatedAt,
		"indexed_at":     time.Now(),
	}

	return json.Marshal(doc)
}

// createSearchText generates optimized search text
func (bl *BulkLoader) createSearchText(record *ConceptRecord) string {
	parts := []string{record.PreferredTerm}

	// Add unique synonyms
	seen := make(map[string]bool)
	seen[record.PreferredTerm] = true

	for _, syn := range record.Synonyms {
		if !seen[syn] {
			parts = append(parts, syn)
			seen[syn] = true
		}
	}

	// Add definition if available
	if record.Definition != "" {
		parts = append(parts, record.Definition)
	}

	return fmt.Sprintf("%s %s", record.Code, parts)
}

// validateConnections ensures both databases are accessible
func (bl *BulkLoader) validateConnections(ctx context.Context) error {
	// Check PostgreSQL
	if err := bl.postgresDB.PingContext(ctx); err != nil {
		return fmt.Errorf("PostgreSQL connection failed: %w", err)
	}

	// Check Elasticsearch
	res, err := bl.elasticsearch.Ping(
		bl.elasticsearch.Ping.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("Elasticsearch ping failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch returned error: %s", res.Status())
	}

	bl.logger.Info("All connections validated successfully")
	return nil
}

// getTotalRecordCount gets the total number of records to process
func (bl *BulkLoader) getTotalRecordCount(ctx context.Context) (int64, error) {
	var count int64

	query := "SELECT COUNT(*) FROM concepts WHERE id > $1"
	if len(bl.config.Systems) > 0 {
		query = fmt.Sprintf("SELECT COUNT(*) FROM concepts WHERE id > $1 AND system = ANY($2)")
		err := bl.postgresDB.QueryRowContext(ctx, query, bl.config.ResumeFromID, pq.Array(bl.config.Systems)).Scan(&count)
		return count, err
	}

	err := bl.postgresDB.QueryRowContext(ctx, query, bl.config.ResumeFromID).Scan(&count)
	return count, err
}

// validateDataConsistency performs consistency checks between stores
func (bl *BulkLoader) validateDataConsistency(ctx context.Context) error {
	bl.logger.Info("Starting data consistency validation")

	// Sample validation: Check record counts
	var pgCount int64
	err := bl.postgresDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM concepts").Scan(&pgCount)
	if err != nil {
		return fmt.Errorf("failed to get PostgreSQL count: %w", err)
	}

	// Get Elasticsearch count
	res, err := bl.elasticsearch.Count(
		bl.elasticsearch.Count.WithContext(ctx),
		bl.elasticsearch.Count.WithIndex(bl.config.ElasticsearchIndex),
	)
	if err != nil {
		return fmt.Errorf("failed to get Elasticsearch count: %w", err)
	}
	defer res.Body.Close()

	var esResult map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&esResult); err != nil {
		return fmt.Errorf("failed to decode ES response: %w", err)
	}

	esCount := int64(esResult["count"].(float64))

	// Compare counts
	if pgCount != esCount {
		diff := pgCount - esCount
		bl.logger.Warnf("Data inconsistency detected: PostgreSQL has %d records, Elasticsearch has %d records (difference: %d)",
			pgCount, esCount, diff)

		atomic.AddInt64(&bl.stats.ValidationErrors, diff)

		if diff > int64(float64(pgCount)*0.01) { // More than 1% difference
			return fmt.Errorf("significant data inconsistency: %d records difference", diff)
		}
	} else {
		bl.logger.Info("Data consistency validation passed")
	}

	return nil
}

// refreshElasticsearchIndex refreshes the index for optimal search
func (bl *BulkLoader) refreshElasticsearchIndex(ctx context.Context) error {
	res, err := bl.elasticsearch.Indices.Refresh(
		bl.elasticsearch.Indices.Refresh.WithContext(ctx),
		bl.elasticsearch.Indices.Refresh.WithIndex(bl.config.ElasticsearchIndex),
	)
	if err != nil {
		return fmt.Errorf("failed to refresh index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index refresh failed: %s", res.Status())
	}

	bl.logger.Info("Elasticsearch index refreshed successfully")
	return nil
}

// updateEstimatedCompletion calculates estimated time to completion
func (bl *BulkLoader) updateEstimatedCompletion() {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	if bl.stats.ProcessedRecords == 0 {
		return
	}

	elapsed := time.Since(bl.stats.StartTime)
	rate := float64(bl.stats.ProcessedRecords) / elapsed.Seconds()
	remaining := bl.stats.TotalRecords - bl.stats.ProcessedRecords

	if rate > 0 {
		estimatedSeconds := float64(remaining) / rate
		bl.stats.EstimatedCompletion = time.Now().Add(time.Duration(estimatedSeconds) * time.Second)
	}
}

// recordError adds an error to the statistics
func (bl *BulkLoader) recordError(err LoadError) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	bl.stats.Errors = append(bl.stats.Errors, err)

	// Keep only last 1000 errors to prevent memory issues
	if len(bl.stats.Errors) > 1000 {
		bl.stats.Errors = bl.stats.Errors[len(bl.stats.Errors)-1000:]
	}
}

// collectErrors collects errors from workers
func (bl *BulkLoader) collectErrors(errors <-chan error, system string) {
	for err := range errors {
		if err != nil {
			bl.logger.WithError(err).Errorf("Worker error for system %s", system)
			bulkLoadErrors.WithLabelValues("worker_error", system).Inc()
		}
	}
}

// recordStats logs final statistics
func (bl *BulkLoader) recordStats() {
	bl.mu.RLock()
	defer bl.mu.RUnlock()

	duration := bl.stats.EndTime.Sub(bl.stats.StartTime)
	successRate := float64(bl.stats.SuccessfulRecords) / float64(bl.stats.TotalRecords) * 100

	bl.logger.WithFields(logrus.Fields{
		"duration":             duration,
		"total_records":        bl.stats.TotalRecords,
		"processed_records":    bl.stats.ProcessedRecords,
		"successful_records":   bl.stats.SuccessfulRecords,
		"failed_records":       bl.stats.FailedRecords,
		"es_writes":           bl.stats.ElasticsearchWrites,
		"validation_errors":    bl.stats.ValidationErrors,
		"success_rate":        fmt.Sprintf("%.2f%%", successRate),
		"records_per_second":  float64(bl.stats.ProcessedRecords) / duration.Seconds(),
	}).Info("Bulk load completed")
}

// GetStatistics returns current load statistics
func (bl *BulkLoader) GetStatistics() LoadStatistics {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return *bl.stats
}

// Close releases resources
func (bl *BulkLoader) Close() error {
	return bl.postgresDB.Close()
}