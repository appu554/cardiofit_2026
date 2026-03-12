package etl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kb-7-terminology/internal/models"
	"kb-7-terminology/internal/semantic"
	"kb-7-terminology/internal/transformer"

	"go.uber.org/zap"
)

// TripleStoreCoordinator extends DualStoreCoordinator with GraphDB triple loading
type TripleStoreCoordinator struct {
	*DualStoreCoordinator

	// GraphDB integration
	graphDBClient   *semantic.GraphDBClient
	graphDBLoader   *GraphDBLoader
	rdfTransformer  *transformer.SNOMEDToRDFTransformer
	tripleValidator *TripleStoreValidator
	graphDBConfig   *GraphDBConfig

	// Triple store state
	tripleStoreStatus *TripleStoreStatus
	statusMutex       sync.RWMutex
}

// GraphDBConfig holds GraphDB-specific configuration
type GraphDBConfig struct {
	Enabled            bool          `json:"enabled"`
	ServerURL          string        `json:"server_url"`
	RepositoryID       string        `json:"repository_id"`
	BatchSize          int           `json:"batch_size"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
	EnableInference    bool          `json:"enable_inference"`
	NamedGraph         string        `json:"named_graph"`
	TransactionTimeout time.Duration `json:"transaction_timeout"`
	ValidateTriples    bool          `json:"validate_triples"`
	ConceptBatchSize   int           `json:"concept_batch_size"` // Concepts per batch for conversion
}

// TripleStoreStatus tracks triple store operation status
type TripleStoreStatus struct {
	PostgreSQLStatus    StoreOperationStatus   `json:"postgresql_status"`
	ElasticsearchStatus StoreOperationStatus   `json:"elasticsearch_status"`
	GraphDBStatus       StoreOperationStatus   `json:"graphdb_status"`
	ConsistencyStatus   TripleStoreConsistency `json:"consistency_status"`
	OverallHealth       string                 `json:"overall_health"`
	LastSyncTime        time.Time              `json:"last_sync_time"`
}

// TripleStoreConsistency tracks 3-way data consistency
type TripleStoreConsistency struct {
	PostgreSQLCount    int64         `json:"postgresql_count"`
	ElasticsearchCount int64         `json:"elasticsearch_count"`
	GraphDBTripleCount int64         `json:"graphdb_triple_count"`
	IsConsistent       bool          `json:"is_consistent"`
	Discrepancy        int64         `json:"discrepancy"`
	LastCheck          time.Time     `json:"last_check"`
	CheckDuration      time.Duration `json:"check_duration"`
	ConsistencyScore   float64       `json:"consistency_score"`
}

// NewTripleStoreCoordinator creates a new triple-store coordinator
func NewTripleStoreCoordinator(
	dualStoreCoordinator *DualStoreCoordinator,
	graphDBConfig *GraphDBConfig,
	logger *zap.Logger,
) (*TripleStoreCoordinator, error) {

	if graphDBConfig == nil {
		graphDBConfig = DefaultGraphDBConfig()
	}

	coordinator := &TripleStoreCoordinator{
		DualStoreCoordinator: dualStoreCoordinator,
		graphDBConfig:       graphDBConfig,
		tripleStoreStatus: &TripleStoreStatus{
			GraphDBStatus: StoreOperationStatus{
				Status:  "idle",
				Health:  "unknown",
				Metrics: make(map[string]int64),
			},
			OverallHealth: "unknown",
		},
	}

	// Initialize GraphDB client if enabled
	if graphDBConfig.Enabled {
		if err := coordinator.initializeGraphDB(); err != nil {
			logger.Warn("Failed to initialize GraphDB, continuing without it", zap.Error(err))
			coordinator.graphDBConfig.Enabled = false
			coordinator.tripleStoreStatus.GraphDBStatus.Health = "disabled"
		}
	}

	logger.Info("Triple-store coordinator initialized",
		zap.Bool("graphdb_enabled", graphDBConfig.Enabled),
		zap.String("repository_id", graphDBConfig.RepositoryID),
	)

	return coordinator, nil
}

// DefaultGraphDBConfig returns default GraphDB configuration
func DefaultGraphDBConfig() *GraphDBConfig {
	return &GraphDBConfig{
		Enabled:            false, // Disabled by default for backward compatibility
		ServerURL:          "http://localhost:7200",
		RepositoryID:       "kb7-terminology",
		BatchSize:          10000,  // Triples per batch upload
		MaxRetries:         3,
		RetryDelay:         5 * time.Second,
		EnableInference:    true,
		NamedGraph:         "http://cardiofit.ai/kb7/graph/default",
		TransactionTimeout: 10 * time.Minute,
		ValidateTriples:    true,
		ConceptBatchSize:   1000, // Concepts per conversion batch
	}
}

// initializeGraphDB sets up GraphDB client connection
func (tsc *TripleStoreCoordinator) initializeGraphDB() error {
	tsc.logger.Info("Initializing GraphDB client",
		zap.String("server_url", tsc.graphDBConfig.ServerURL),
		zap.String("repository", tsc.graphDBConfig.RepositoryID),
	)

	// Create GraphDB client (using existing implementation)
	client := semantic.NewGraphDBClient(
		tsc.graphDBConfig.ServerURL,
		tsc.graphDBConfig.RepositoryID,
		nil, // Will use zap logger instead
	)

	tsc.graphDBClient = client

	// Create RDF transformer
	tsc.rdfTransformer = transformer.NewSNOMEDToRDFTransformer(tsc.logger)

	// Create GraphDB loader
	loaderConfig := &GraphDBLoaderConfig{
		BatchSize:          tsc.graphDBConfig.BatchSize,
		MaxConcurrent:      4,
		UploadTimeout:      5 * time.Minute,
		EnableCompression:  true,
		ValidateBeforeLoad: false,
		NamedGraph:         tsc.graphDBConfig.NamedGraph,
		MaxRetries:         tsc.graphDBConfig.MaxRetries,
		RetryDelay:         tsc.graphDBConfig.RetryDelay,
	}
	tsc.graphDBLoader = NewGraphDBLoader(client, tsc.rdfTransformer, tsc.logger, loaderConfig)

	// Create triple validator
	tsc.tripleValidator = NewTripleStoreValidator(tsc.db, client, tsc.logger)

	// Test GraphDB connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := tsc.graphDBClient.HealthCheck(ctx)
	if err != nil {
		tsc.tripleStoreStatus.GraphDBStatus.Health = "unhealthy"
		tsc.tripleStoreStatus.GraphDBStatus.LastError = err.Error()
		tsc.logger.Warn("GraphDB health check failed", zap.Error(err))
		return fmt.Errorf("GraphDB health check failed: %w", err)
	}

	tsc.tripleStoreStatus.GraphDBStatus.Health = "healthy"
	tsc.logger.Info("GraphDB connection established successfully")

	return nil
}

// LoadAllTerminologiesTripleStore loads all terminologies to PostgreSQL + GraphDB + Elasticsearch
func (tsc *TripleStoreCoordinator) LoadAllTerminologiesTripleStore(
	ctx context.Context,
	dataSources map[string]string,
) error {

	tsc.statusMutex.Lock()
	tsc.tripleStoreStatus.OverallHealth = "loading"
	tsc.statusMutex.Unlock()

	tsc.logger.Info("Starting triple-store terminology loading",
		zap.Bool("graphdb_enabled", tsc.graphDBConfig.Enabled))

	// PHASE 1: Load to PostgreSQL + Elasticsearch (existing DualStoreCoordinator logic)
	tsc.logger.Info("Phase 1: Loading to PostgreSQL + Elasticsearch")
	pgStartTime := time.Now()

	err := tsc.DualStoreCoordinator.LoadAllTerminologiesDualStore(ctx, dataSources)
	pgDuration := time.Since(pgStartTime)

	if err != nil {
		tsc.tripleStoreStatus.OverallHealth = "failed"
		tsc.logger.Error("PostgreSQL/Elasticsearch loading failed", zap.Error(err))
		return fmt.Errorf("PostgreSQL/Elasticsearch loading failed: %w", err)
	}

	tsc.logger.Info("PostgreSQL + Elasticsearch loading completed",
		zap.Duration("duration", pgDuration),
	)

	// PHASE 2: Sync to GraphDB (NEW logic)
	if tsc.graphDBConfig.Enabled && tsc.graphDBClient != nil {
		tsc.logger.Info("Phase 2: Syncing to GraphDB")

		if err := tsc.syncToGraphDB(ctx); err != nil {
			tsc.logger.Error("GraphDB sync failed", zap.Error(err))
			tsc.updateStoreStatus("graphdb", "failed", err.Error())
			tsc.tripleStoreStatus.OverallHealth = "degraded"

			// GraphDB failure does not block PostgreSQL success
			tsc.logger.Warn("Continuing with PostgreSQL-only operation (GraphDB sync failed)")
		} else {
			tsc.updateStoreStatus("graphdb", "completed", "")
			tsc.tripleStoreStatus.OverallHealth = "healthy"
		}
	} else {
		tsc.logger.Info("GraphDB disabled, skipping triple loading")
		tsc.tripleStoreStatus.OverallHealth = "healthy"
	}

	// PHASE 3: Perform 3-way consistency check
	if tsc.graphDBConfig.Enabled && tsc.graphDBConfig.ValidateTriples && tsc.graphDBClient != nil {
		tsc.logger.Info("Phase 3: Performing consistency validation")

		if err := tsc.performTripleStoreConsistencyCheck(ctx); err != nil {
			tsc.logger.Warn("Consistency check failed", zap.Error(err))
		}
	}

	tsc.tripleStoreStatus.LastSyncTime = time.Now()
	tsc.logger.Info("Triple-store loading completed",
		zap.String("overall_health", tsc.tripleStoreStatus.OverallHealth),
	)

	return nil
}

// syncToGraphDB reads concepts from PostgreSQL and loads them to GraphDB as RDF triples
func (tsc *TripleStoreCoordinator) syncToGraphDB(ctx context.Context) error {
	if tsc.graphDBClient == nil {
		return fmt.Errorf("GraphDB client not initialized")
	}

	graphStartTime := time.Now()
	tsc.updateStoreStatus("graphdb", "running", "Converting PostgreSQL to RDF triples")

	// Step 1: Read all concepts from PostgreSQL
	tsc.logger.Info("Reading concepts from PostgreSQL")

	concepts, err := tsc.readConceptsFromPostgreSQL(ctx)
	if err != nil {
		return fmt.Errorf("failed to read concepts from PostgreSQL: %w", err)
	}

	tsc.logger.Info("Concepts loaded from PostgreSQL", zap.Int("count", len(concepts)))

	if len(concepts) == 0 {
		tsc.logger.Warn("No concepts found in PostgreSQL, skipping GraphDB sync")
		return nil
	}

	// Step 2: Convert concepts to RDF triples in batches
	tsc.logger.Info("Converting concepts to RDF triples",
		zap.Int("concept_batch_size", tsc.graphDBConfig.ConceptBatchSize))

	totalTriplesLoaded := int64(0)
	batchSize := tsc.graphDBConfig.ConceptBatchSize

	for i := 0; i < len(concepts); i += batchSize {
		end := i + batchSize
		if end > len(concepts) {
			end = len(concepts)
		}

		batch := concepts[i:end]

		// Convert batch to Turtle document
		turtleString, err := tsc.rdfTransformer.ConvertBatchToTurtleString(ctx, batch)
		if err != nil {
			tsc.logger.Error("Failed to convert batch to Turtle",
				zap.Int("batch_start", i),
				zap.Error(err))
			continue
		}

		// Step 3: Upload Turtle document to GraphDB
		tsc.logger.Debug("Uploading batch to GraphDB",
			zap.Int("batch_start", i),
			zap.Int("batch_end", end),
			zap.Int("content_size", len(turtleString)))

		err = tsc.graphDBLoader.LoadTurtleString(ctx, turtleString, tsc.graphDBConfig.NamedGraph)
		if err != nil {
			tsc.logger.Error("Failed to upload batch to GraphDB",
				zap.Int("batch_start", i),
				zap.Error(err))
			return fmt.Errorf("GraphDB upload failed for batch %d-%d: %w", i, end, err)
		}

		// Estimate triples (average 8-24 per concept)
		estimatedTriples := int64(len(batch) * 12)
		totalTriplesLoaded += estimatedTriples

		// Progress logging
		if (i/batchSize)%10 == 0 && i > 0 {
			tsc.logger.Info("GraphDB sync progress",
				zap.Int("concepts_processed", i),
				zap.Int64("estimated_triples", totalTriplesLoaded))
		}
	}

	graphDuration := time.Since(graphStartTime)

	tsc.statusMutex.Lock()
	tsc.tripleStoreStatus.GraphDBStatus.RecordsWritten = totalTriplesLoaded
	tsc.tripleStoreStatus.GraphDBStatus.ResponseTime = graphDuration
	tsc.tripleStoreStatus.GraphDBStatus.Metrics["concepts_converted"] = int64(len(concepts))
	tsc.tripleStoreStatus.GraphDBStatus.Metrics["batches_processed"] = int64((len(concepts) + batchSize - 1) / batchSize)
	tsc.statusMutex.Unlock()

	tsc.logger.Info("GraphDB sync completed",
		zap.Int64("estimated_triples", totalTriplesLoaded),
		zap.Int("concepts_processed", len(concepts)),
		zap.Duration("duration", graphDuration))

	return nil
}

// readConceptsFromPostgreSQL reads all concepts from PostgreSQL database
func (tsc *TripleStoreCoordinator) readConceptsFromPostgreSQL(ctx context.Context) ([]*models.Concept, error) {
	query := `
		SELECT id, system, code, preferred_term, definition, active, version,
		       properties, status, created_at, updated_at
		FROM concepts
		WHERE active = true
		ORDER BY system, code
	`

	rows, err := tsc.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query concepts: %w", err)
	}
	defer rows.Close()

	concepts := make([]*models.Concept, 0, 520000)

	for rows.Next() {
		concept := &models.Concept{
			Properties: make(models.JSONB),
		}

		err := rows.Scan(
			&concept.ID,
			&concept.System,
			&concept.Code,
			&concept.PreferredTerm,
			&concept.Definition,
			&concept.Active,
			&concept.Version,
			&concept.Properties,
			&concept.Status,
			&concept.CreatedAt,
			&concept.UpdatedAt,
		)

		if err != nil {
			tsc.logger.Warn("Failed to scan concept row", zap.Error(err))
			continue
		}

		concepts = append(concepts, concept)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return concepts, nil
}

// performTripleStoreConsistencyCheck validates data consistency across all 3 stores
func (tsc *TripleStoreCoordinator) performTripleStoreConsistencyCheck(ctx context.Context) error {
	checkStartTime := time.Now()
	tsc.logger.Info("Starting 3-way consistency check")

	// Use the validator
	result, err := tsc.tripleValidator.ValidateConsistency(ctx)
	if err != nil {
		return fmt.Errorf("consistency validation failed: %w", err)
	}

	// Update consistency status
	tsc.statusMutex.Lock()
	tsc.tripleStoreStatus.ConsistencyStatus = TripleStoreConsistency{
		PostgreSQLCount:    result.PostgreSQLCount,
		ElasticsearchCount: result.ElasticsearchCount,
		GraphDBTripleCount: result.GraphDBTripleCount,
		IsConsistent:       result.IsConsistent,
		Discrepancy:        result.Discrepancy,
		LastCheck:          checkStartTime,
		CheckDuration:      result.Duration,
		ConsistencyScore:   result.ConsistencyScore,
	}
	tsc.statusMutex.Unlock()

	tsc.logger.Info("Consistency check completed",
		zap.Int64("postgresql_concepts", result.PostgreSQLCount),
		zap.Int64("graphdb_triples", result.GraphDBTripleCount),
		zap.Bool("is_consistent", result.IsConsistent),
		zap.Float64("consistency_score", result.ConsistencyScore))

	if !result.IsConsistent {
		tsc.logger.Warn("Consistency check failed",
			zap.Int64("expected_triples", result.ExpectedTriples),
			zap.Int64("actual_triples", result.GraphDBTripleCount),
			zap.Int64("discrepancy", result.Discrepancy))
	}

	return nil
}

// Helper methods

func (tsc *TripleStoreCoordinator) updateStoreStatus(store, status, errorMsg string) {
	tsc.statusMutex.Lock()
	defer tsc.statusMutex.Unlock()

	switch store {
	case "graphdb":
		tsc.tripleStoreStatus.GraphDBStatus.Status = status
		tsc.tripleStoreStatus.GraphDBStatus.LastOperation = fmt.Sprintf("status_update_%d", time.Now().Unix())
		if errorMsg != "" {
			tsc.tripleStoreStatus.GraphDBStatus.LastError = errorMsg
			tsc.tripleStoreStatus.GraphDBStatus.RecordsFailed++
			tsc.tripleStoreStatus.GraphDBStatus.Health = "unhealthy"
		} else if status == "completed" {
			tsc.tripleStoreStatus.GraphDBStatus.Health = "healthy"
		}
	case "postgresql":
		if tsc.DualStoreCoordinator != nil {
			tsc.DualStoreCoordinator.updateStoreStatus(store, status, errorMsg)
		}
	case "elasticsearch":
		if tsc.DualStoreCoordinator != nil {
			tsc.DualStoreCoordinator.updateStoreStatus(store, status, errorMsg)
		}
	}
}

// GetTripleStoreStatus returns current triple-store status
func (tsc *TripleStoreCoordinator) GetTripleStoreStatus() *TripleStoreStatus {
	tsc.statusMutex.RLock()
	defer tsc.statusMutex.RUnlock()

	// Copy status to avoid race conditions
	status := *tsc.tripleStoreStatus

	// Copy nested status from DualStoreCoordinator
	if tsc.DualStoreCoordinator != nil {
		dualStatus := tsc.DualStoreCoordinator.GetDualStoreStatus()
		status.PostgreSQLStatus = dualStatus.PostgreSQLStatus
		status.ElasticsearchStatus = dualStatus.ElasticsearchStatus
	}

	return &status
}

// GetStatus returns the current ETL status from the embedded coordinator
func (tsc *TripleStoreCoordinator) GetStatus() *ETLStatus {
	if tsc.DualStoreCoordinator != nil {
		return tsc.DualStoreCoordinator.GetStatus()
	}
	return &ETLStatus{
		OverallStatus:  "unknown",
		SystemStatuses: make(map[string]SystemStatus),
	}
}

// Close closes all connections
func (tsc *TripleStoreCoordinator) Close() error {
	// GraphDB client doesn't need explicit close (HTTP client)
	if tsc.DualStoreCoordinator != nil {
		return tsc.DualStoreCoordinator.Close()
	}
	return nil
}

// GetGraphDBConfig returns the GraphDB configuration
func (tsc *TripleStoreCoordinator) GetGraphDBConfig() *GraphDBConfig {
	return tsc.graphDBConfig
}

// IsGraphDBEnabled returns whether GraphDB is enabled
func (tsc *TripleStoreCoordinator) IsGraphDBEnabled() bool {
	return tsc.graphDBConfig != nil && tsc.graphDBConfig.Enabled
}
