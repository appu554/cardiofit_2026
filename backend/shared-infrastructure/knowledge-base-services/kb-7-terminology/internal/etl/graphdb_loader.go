package etl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kb-7-terminology/internal/semantic"
	"kb-7-terminology/internal/transformer"

	"go.uber.org/zap"
)

// GraphDBLoader handles GraphDB-specific loading operations
type GraphDBLoader struct {
	client      *semantic.GraphDBClient
	transformer *transformer.SNOMEDToRDFTransformer
	logger      *zap.Logger
	config      *GraphDBLoaderConfig
}

// GraphDBLoaderConfig holds loader configuration
type GraphDBLoaderConfig struct {
	BatchSize          int           `json:"batch_size"`
	MaxConcurrent      int           `json:"max_concurrent"`
	UploadTimeout      time.Duration `json:"upload_timeout"`
	EnableCompression  bool          `json:"enable_compression"`
	ValidateBeforeLoad bool          `json:"validate_before_load"`
	NamedGraph         string        `json:"named_graph"`
	MaxRetries         int           `json:"max_retries"`
	RetryDelay         time.Duration `json:"retry_delay"`
}

// LoadMetrics tracks loading performance metrics
type LoadMetrics struct {
	TotalTriples     int64         `json:"total_triples"`
	TotalConcepts    int64         `json:"total_concepts"`
	BatchesProcessed int64         `json:"batches_processed"`
	BatchesFailed    int64         `json:"batches_failed"`
	Duration         time.Duration `json:"duration"`
	TriplePerSecond  float64       `json:"triples_per_second"`
}

// NewGraphDBLoader creates a new GraphDB loader
func NewGraphDBLoader(
	client *semantic.GraphDBClient,
	transformer *transformer.SNOMEDToRDFTransformer,
	logger *zap.Logger,
	config *GraphDBLoaderConfig,
) *GraphDBLoader {
	if config == nil {
		config = DefaultGraphDBLoaderConfig()
	}

	return &GraphDBLoader{
		client:      client,
		transformer: transformer,
		logger:      logger,
		config:      config,
	}
}

// DefaultGraphDBLoaderConfig returns default loader configuration
func DefaultGraphDBLoaderConfig() *GraphDBLoaderConfig {
	return &GraphDBLoaderConfig{
		BatchSize:          10000,
		MaxConcurrent:      4,
		UploadTimeout:      5 * time.Minute,
		EnableCompression:  true,
		ValidateBeforeLoad: false,
		NamedGraph:         "http://cardiofit.ai/kb7/graph/default",
		MaxRetries:         3,
		RetryDelay:         5 * time.Second,
	}
}

// LoadTriples loads RDF triples to GraphDB in batches
func (gbl *GraphDBLoader) LoadTriples(ctx context.Context, triples []transformer.RDFTriple) (*LoadMetrics, error) {
	startTime := time.Now()
	metrics := &LoadMetrics{
		TotalTriples: int64(len(triples)),
	}

	gbl.logger.Info("Loading triples to GraphDB",
		zap.Int("triple_count", len(triples)),
		zap.Int("batch_size", gbl.config.BatchSize))

	// Group triples into batches
	batches := gbl.batchTriples(triples, gbl.config.BatchSize)
	gbl.logger.Info("Divided triples into batches", zap.Int("batch_count", len(batches)))

	// Upload batches
	for i, batch := range batches {
		gbl.logger.Debug("Processing batch",
			zap.Int("batch_index", i),
			zap.Int("batch_size", len(batch)))

		turtleDoc := gbl.triplesToTurtle(batch)

		// Upload with timeout
		uploadCtx, cancel := context.WithTimeout(ctx, gbl.config.UploadTimeout)
		err := gbl.uploadWithRetry(uploadCtx, turtleDoc, gbl.config.NamedGraph)
		cancel()

		if err != nil {
			metrics.BatchesFailed++
			gbl.logger.Error("Failed to upload batch",
				zap.Int("batch_index", i),
				zap.Error(err))
			return metrics, fmt.Errorf("failed to upload batch %d: %w", i, err)
		}

		metrics.BatchesProcessed++

		// Progress logging every 10 batches
		if (i+1)%10 == 0 {
			gbl.logger.Info("Batch upload progress",
				zap.Int("batches_completed", i+1),
				zap.Int("total_batches", len(batches)),
				zap.Int64("triples_uploaded", int64(i+1)*int64(gbl.config.BatchSize)))
		}
	}

	// Calculate final metrics
	metrics.Duration = time.Since(startTime)
	if metrics.Duration.Seconds() > 0 {
		metrics.TriplePerSecond = float64(metrics.TotalTriples) / metrics.Duration.Seconds()
	}

	gbl.logger.Info("All triples loaded successfully",
		zap.Int64("total_triples", metrics.TotalTriples),
		zap.Int64("batches_processed", metrics.BatchesProcessed),
		zap.Duration("duration", metrics.Duration),
		zap.Float64("triples_per_second", metrics.TriplePerSecond))

	return metrics, nil
}

// LoadTurtleString loads a Turtle string to GraphDB
func (gbl *GraphDBLoader) LoadTurtleString(ctx context.Context, turtleContent string, namedGraph string) error {
	if namedGraph == "" {
		namedGraph = gbl.config.NamedGraph
	}

	gbl.logger.Debug("Loading Turtle string to GraphDB",
		zap.Int("content_size", len(turtleContent)),
		zap.String("named_graph", namedGraph))

	return gbl.uploadWithRetry(ctx, turtleContent, namedGraph)
}

// ClearRepository clears all data from GraphDB repository
func (gbl *GraphDBLoader) ClearRepository(ctx context.Context) error {
	gbl.logger.Warn("Clearing GraphDB repository")

	// Use SPARQL UPDATE to clear all triples
	updateQuery := "CLEAR ALL"

	if err := gbl.client.ExecuteUpdate(ctx, updateQuery); err != nil {
		return fmt.Errorf("failed to clear repository: %w", err)
	}

	gbl.logger.Info("GraphDB repository cleared successfully")
	return nil
}

// ClearGraph clears a specific named graph
func (gbl *GraphDBLoader) ClearGraph(ctx context.Context, graphURI string) error {
	gbl.logger.Warn("Clearing named graph", zap.String("graph", graphURI))

	updateQuery := fmt.Sprintf("CLEAR GRAPH <%s>", graphURI)

	if err := gbl.client.ExecuteUpdate(ctx, updateQuery); err != nil {
		return fmt.Errorf("failed to clear graph %s: %w", graphURI, err)
	}

	gbl.logger.Info("Named graph cleared successfully", zap.String("graph", graphURI))
	return nil
}

// ValidateTriples validates triples before loading
func (gbl *GraphDBLoader) ValidateTriples(triples []transformer.RDFTriple) error {
	if !gbl.config.ValidateBeforeLoad {
		return nil
	}

	gbl.logger.Info("Validating triples", zap.Int("count", len(triples)))

	invalidCount := 0
	for i, triple := range triples {
		if err := gbl.validateTriple(triple); err != nil {
			gbl.logger.Warn("Invalid triple",
				zap.Int("index", i),
				zap.String("subject", triple.Subject),
				zap.Error(err))
			invalidCount++
		}
	}

	if invalidCount > 0 {
		return fmt.Errorf("%d invalid triples found out of %d", invalidCount, len(triples))
	}

	gbl.logger.Info("Triple validation passed", zap.Int("count", len(triples)))
	return nil
}

// CountTriples counts the number of triples in the repository
func (gbl *GraphDBLoader) CountTriples(ctx context.Context) (int64, error) {
	query := &semantic.SPARQLQuery{
		Query: "SELECT (COUNT(*) as ?count) WHERE { ?s ?p ?o }",
	}

	results, err := gbl.client.ExecuteSPARQL(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count triples: %w", err)
	}

	// Parse count from results
	if len(results.Results.Bindings) > 0 {
		if countBinding, ok := results.Results.Bindings[0]["count"]; ok {
			var count int64
			fmt.Sscanf(countBinding.Value, "%d", &count)
			return count, nil
		}
	}

	return 0, fmt.Errorf("no count result returned")
}

// Helper methods

func (gbl *GraphDBLoader) uploadWithRetry(ctx context.Context, turtleContent string, namedGraph string) error {
	var lastErr error

	for attempt := 1; attempt <= gbl.config.MaxRetries; attempt++ {
		// Convert string to byte array for LoadTurtleData
		err := gbl.client.LoadTurtleData(ctx, []byte(turtleContent), namedGraph)
		if err == nil {
			if attempt > 1 {
				gbl.logger.Info("Upload succeeded after retry", zap.Int("attempt", attempt))
			}
			return nil
		}

		lastErr = err
		gbl.logger.Warn("Upload attempt failed",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", gbl.config.MaxRetries),
			zap.Error(err))

		// Don't sleep after the last attempt
		if attempt < gbl.config.MaxRetries {
			// Exponential backoff
			delay := gbl.config.RetryDelay * time.Duration(attempt)
			gbl.logger.Debug("Retrying after delay", zap.Duration("delay", delay))
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("upload failed after %d attempts: %w", gbl.config.MaxRetries, lastErr)
}

func (gbl *GraphDBLoader) batchTriples(triples []transformer.RDFTriple, batchSize int) [][]transformer.RDFTriple {
	batches := make([][]transformer.RDFTriple, 0)

	for i := 0; i < len(triples); i += batchSize {
		end := i + batchSize
		if end > len(triples) {
			end = len(triples)
		}
		batches = append(batches, triples[i:end])
	}

	return batches
}

func (gbl *GraphDBLoader) triplesToTurtle(triples []transformer.RDFTriple) string {
	// Generate Turtle document from triples
	var builder strings.Builder

	// Write prefixes
	builder.WriteString("@prefix kb7: <http://cardiofit.ai/kb7/ontology#> .\n")
	builder.WriteString("@prefix sct: <http://snomed.info/id/> .\n")
	builder.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	builder.WriteString("@prefix skos: <http://www.w3.org/2004/02/skos/core#> .\n")
	builder.WriteString("@prefix owl: <http://www.w3.org/2002/07/owl#> .\n")
	builder.WriteString("@prefix xsd: <http://www.w3.org/2001/XMLSchema#> .\n")
	builder.WriteString("@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n\n")

	// Write triples
	for _, triple := range triples {
		object := gbl.formatTurtleObject(triple)
		builder.WriteString(fmt.Sprintf("%s %s %s .\n",
			triple.Subject,
			triple.Predicate,
			object))
	}

	return builder.String()
}

func (gbl *GraphDBLoader) formatTurtleObject(triple transformer.RDFTriple) string {
	switch triple.ObjectType {
	case transformer.RDFObjectURI:
		return triple.Object
	case transformer.RDFObjectLiteral:
		// Escape special characters
		escaped := gbl.escapeTurtleLiteral(triple.Object)

		if triple.Language != "" {
			return fmt.Sprintf(`"%s"@%s`, escaped, triple.Language)
		}
		if triple.DataType != "" {
			return fmt.Sprintf(`"%s"^^%s`, escaped, triple.DataType)
		}
		return fmt.Sprintf(`"%s"`, escaped)
	case transformer.RDFObjectBlankNode:
		return triple.Object
	default:
		return fmt.Sprintf(`"%s"`, triple.Object)
	}
}

func (gbl *GraphDBLoader) escapeTurtleLiteral(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", "\\n")
	value = strings.ReplaceAll(value, "\r", "\\r")
	value = strings.ReplaceAll(value, "\t", "\\t")
	return value
}

func (gbl *GraphDBLoader) validateTriple(triple transformer.RDFTriple) error {
	if triple.Subject == "" {
		return fmt.Errorf("empty subject")
	}
	if triple.Predicate == "" {
		return fmt.Errorf("empty predicate")
	}
	if triple.Object == "" {
		return fmt.Errorf("empty object")
	}

	// Validate URI format for subject and predicate
	if !strings.Contains(triple.Subject, ":") {
		return fmt.Errorf("invalid subject URI format: %s", triple.Subject)
	}
	if !strings.Contains(triple.Predicate, ":") {
		return fmt.Errorf("invalid predicate URI format: %s", triple.Predicate)
	}

	return nil
}

// GetConfig returns the loader configuration
func (gbl *GraphDBLoader) GetConfig() *GraphDBLoaderConfig {
	return gbl.config
}

// SetBatchSize updates the batch size configuration
func (gbl *GraphDBLoader) SetBatchSize(size int) {
	if size > 0 {
		gbl.config.BatchSize = size
		gbl.logger.Info("Updated batch size", zap.Int("new_size", size))
	}
}
