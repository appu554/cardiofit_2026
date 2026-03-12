// Package cdc provides Change Data Capture functionality for KB-7.
// Phase 6.2: CDC Sync (GraphDB → Neo4j)
//
// This consumer synchronizes GraphDB changes to Neo4j in real-time via Kafka,
// ensuring the read replica stays consistent with the master.
//
// Flow:
//   1. GraphDB Change Event (INSERT/DELETE triple)
//   2. Kafka Topic: "kb7.graphdb.changes"
//   3. Neo4j Consumer (this service)
//   4. Neo4j Updated (Latency: <1 second)
//
// The "Commit-Last" strategy ensures Neo4j only receives events after
// GraphDB has successfully committed changes.
package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// OperationType represents the type of graph change
type OperationType string

const (
	OperationInsert OperationType = "INSERT"
	OperationDelete OperationType = "DELETE"
	OperationUpdate OperationType = "UPDATE"
)

// GraphDBChange represents a change event from GraphDB
type GraphDBChange struct {
	Operation   OperationType `json:"operation"`
	Subject     string        `json:"subject"`
	Predicate   string        `json:"predicate"`
	Object      string        `json:"object"`
	ObjectType  string        `json:"object_type,omitempty"` // "uri" or "literal"
	Graph       string        `json:"graph,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	TransactionID string      `json:"transaction_id,omitempty"`
}

// ConsumerConfig holds configuration for the CDC consumer
type ConsumerConfig struct {
	// Kafka configuration
	KafkaBrokers []string
	Topic        string
	GroupID      string
	Partition    int

	// Neo4j configuration
	Neo4jURL      string
	Neo4jUsername string
	Neo4jPassword string
	Neo4jDatabase string

	// Processing configuration
	BatchSize        int
	BatchTimeout     time.Duration
	MaxRetries       int
	RetryBackoff     time.Duration
	CommitInterval   time.Duration
	WorkerCount      int
}

// ConsumerStats holds runtime statistics
type ConsumerStats struct {
	MessagesReceived   int64     `json:"messages_received"`
	MessagesProcessed  int64     `json:"messages_processed"`
	MessagesFailed     int64     `json:"messages_failed"`
	BatchesCommitted   int64     `json:"batches_committed"`
	LastCommitTime     time.Time `json:"last_commit_time"`
	LastError          string    `json:"last_error,omitempty"`
	Lag                int64     `json:"lag"`
	ProcessingRate     float64   `json:"processing_rate"` // messages per second
}

// Neo4jConsumer synchronizes GraphDB changes to Neo4j
type Neo4jConsumer struct {
	config   *ConsumerConfig
	reader   *kafka.Reader
	neo4j    neo4j.DriverWithContext
	logger   *logrus.Logger
	stats    ConsumerStats
	statsMu  sync.RWMutex
	shutdown chan struct{}
	wg       sync.WaitGroup
}

// NewNeo4jConsumer creates a new CDC consumer
func NewNeo4jConsumer(config *ConsumerConfig, logger *logrus.Logger) (*Neo4jConsumer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Set defaults
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = 5 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = time.Second
	}
	if config.CommitInterval == 0 {
		config.CommitInterval = 10 * time.Second
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 4
	}

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  config.KafkaBrokers,
		Topic:    config.Topic,
		GroupID:  config.GroupID,
		MinBytes: 1e3,  // 1KB
		MaxBytes: 10e6, // 10MB
		MaxWait:  config.BatchTimeout,
	})

	// Create Neo4j driver
	neo4jDriver, err := neo4j.NewDriverWithContext(
		config.Neo4jURL,
		neo4j.BasicAuth(config.Neo4jUsername, config.Neo4jPassword, ""),
	)
	if err != nil {
		reader.Close()
		return nil, fmt.Errorf("creating neo4j driver: %w", err)
	}

	// Verify Neo4j connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := neo4jDriver.VerifyConnectivity(ctx); err != nil {
		reader.Close()
		neo4jDriver.Close(ctx)
		return nil, fmt.Errorf("verifying neo4j connectivity: %w", err)
	}

	consumer := &Neo4jConsumer{
		config:   config,
		reader:   reader,
		neo4j:    neo4jDriver,
		logger:   logger,
		shutdown: make(chan struct{}),
	}

	logger.WithFields(logrus.Fields{
		"brokers":    config.KafkaBrokers,
		"topic":      config.Topic,
		"group_id":   config.GroupID,
		"neo4j_url":  config.Neo4jURL,
		"batch_size": config.BatchSize,
	}).Info("CDC consumer initialized")

	return consumer, nil
}

// Start begins consuming and processing CDC events
func (c *Neo4jConsumer) Start(ctx context.Context) error {
	c.logger.Info("Starting CDC consumer...")

	// Start worker pool
	changes := make(chan []*GraphDBChange, c.config.WorkerCount*2)

	c.wg.Add(c.config.WorkerCount)
	for i := 0; i < c.config.WorkerCount; i++ {
		go c.worker(ctx, i, changes)
	}

	// Start batch collector
	c.wg.Add(1)
	go c.batchCollector(ctx, changes)

	// Wait for shutdown
	select {
	case <-ctx.Done():
		c.logger.Info("Context cancelled, shutting down consumer...")
	case <-c.shutdown:
		c.logger.Info("Shutdown signal received...")
	}

	close(changes)
	c.wg.Wait()

	return nil
}

// Stop gracefully stops the consumer
func (c *Neo4jConsumer) Stop() error {
	close(c.shutdown)
	c.wg.Wait()

	if err := c.reader.Close(); err != nil {
		c.logger.WithError(err).Error("Error closing Kafka reader")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.neo4j.Close(ctx); err != nil {
		c.logger.WithError(err).Error("Error closing Neo4j driver")
	}

	c.logger.Info("CDC consumer stopped")
	return nil
}

// Stats returns current consumer statistics
func (c *Neo4jConsumer) Stats() ConsumerStats {
	c.statsMu.RLock()
	defer c.statsMu.RUnlock()
	return c.stats
}

// batchCollector reads from Kafka and batches messages
func (c *Neo4jConsumer) batchCollector(ctx context.Context, changes chan<- []*GraphDBChange) {
	defer c.wg.Done()

	batch := make([]*GraphDBChange, 0, c.config.BatchSize)
	ticker := time.NewTicker(c.config.BatchTimeout)
	defer ticker.Stop()

	flushBatch := func() {
		if len(batch) > 0 {
			// Send batch to workers
			batchCopy := make([]*GraphDBChange, len(batch))
			copy(batchCopy, batch)

			select {
			case changes <- batchCopy:
			case <-ctx.Done():
				return
			}
			batch = batch[:0]
		}
	}

	for {
		select {
		case <-ctx.Done():
			flushBatch()
			return
		case <-c.shutdown:
			flushBatch()
			return
		case <-ticker.C:
			flushBatch()
		default:
			// Read message with timeout
			readCtx, cancel := context.WithTimeout(ctx, c.config.BatchTimeout)
			msg, err := c.reader.ReadMessage(readCtx)
			cancel()

			if err != nil {
				if ctx.Err() != nil {
					flushBatch()
					return
				}
				// Log and continue on transient errors
				c.logger.WithError(err).Debug("Error reading Kafka message")
				continue
			}

			atomic.AddInt64(&c.stats.MessagesReceived, 1)

			var change GraphDBChange
			if err := json.Unmarshal(msg.Value, &change); err != nil {
				c.logger.WithError(err).WithField("offset", msg.Offset).
					Error("Failed to unmarshal change event")
				atomic.AddInt64(&c.stats.MessagesFailed, 1)
				continue
			}

			batch = append(batch, &change)

			if len(batch) >= c.config.BatchSize {
				flushBatch()
			}
		}
	}
}

// worker processes batches of changes
func (c *Neo4jConsumer) worker(ctx context.Context, id int, changes <-chan []*GraphDBChange) {
	defer c.wg.Done()

	logger := c.logger.WithField("worker_id", id)
	logger.Debug("Worker started")

	for batch := range changes {
		if err := c.processBatch(ctx, batch); err != nil {
			logger.WithError(err).Error("Failed to process batch")
			c.statsMu.Lock()
			c.stats.LastError = err.Error()
			c.statsMu.Unlock()
		} else {
			atomic.AddInt64(&c.stats.BatchesCommitted, 1)
		}
	}

	logger.Debug("Worker stopped")
}

// processBatch applies a batch of changes to Neo4j
func (c *Neo4jConsumer) processBatch(ctx context.Context, batch []*GraphDBChange) error {
	if len(batch) == 0 {
		return nil
	}

	session := c.neo4j.NewSession(ctx, neo4j.SessionConfig{
		AccessMode:   neo4j.AccessModeWrite,
		DatabaseName: c.config.Neo4jDatabase,
	})
	defer session.Close(ctx)

	// Process in a single transaction for atomicity
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		for _, change := range batch {
			var err error
			switch change.Operation {
			case OperationInsert:
				err = c.applyInsert(ctx, tx, change)
			case OperationDelete:
				err = c.applyDelete(ctx, tx, change)
			case OperationUpdate:
				// Update = Delete + Insert
				_ = c.applyDelete(ctx, tx, change)
				err = c.applyInsert(ctx, tx, change)
			default:
				c.logger.WithField("operation", change.Operation).
					Warn("Unknown operation type, skipping")
				continue
			}

			if err != nil {
				return nil, fmt.Errorf("applying %s for %s: %w",
					change.Operation, change.Subject, err)
			}
			atomic.AddInt64(&c.stats.MessagesProcessed, 1)
		}
		return nil, nil
	})

	if err != nil {
		atomic.AddInt64(&c.stats.MessagesFailed, int64(len(batch)))
		return err
	}

	c.statsMu.Lock()
	c.stats.LastCommitTime = time.Now()
	c.statsMu.Unlock()

	return nil
}

// applyInsert inserts a triple into Neo4j
func (c *Neo4jConsumer) applyInsert(ctx context.Context, tx neo4j.ManagedTransaction, change *GraphDBChange) error {
	// Parse subject and object URIs
	subjectURI := change.Subject
	predicateURI := change.Predicate
	objectURI := change.Object

	// Convert predicate URI to Neo4j relationship type
	relType := uriToRelationType(predicateURI)

	// Handle different relationship types
	if strings.Contains(predicateURI, "subClassOf") {
		// Hierarchy relationship
		query := `
			MERGE (s:Class {uri: $subjectUri})
			ON CREATE SET s.code = $subjectCode, s.system = $subjectSystem
			MERGE (o:Class {uri: $objectUri})
			ON CREATE SET o.code = $objectCode, o.system = $objectSystem
			MERGE (s)-[:rdfs__subClassOf]->(o)
		`
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"subjectUri":    subjectURI,
			"subjectCode":   extractCode(subjectURI),
			"subjectSystem": extractSystem(subjectURI),
			"objectUri":     objectURI,
			"objectCode":    extractCode(objectURI),
			"objectSystem":  extractSystem(objectURI),
		})
		return err
	} else if change.ObjectType == "literal" {
		// Property with literal value
		query := fmt.Sprintf(`
			MERGE (s:Class {uri: $subjectUri})
			SET s.%s = $objectValue
		`, sanitizePropertyName(relType))
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"subjectUri":  subjectURI,
			"objectValue": objectURI,
		})
		return err
	} else {
		// Generic relationship
		query := fmt.Sprintf(`
			MERGE (s:Class {uri: $subjectUri})
			MERGE (o:Class {uri: $objectUri})
			MERGE (s)-[:%s]->(o)
		`, sanitizeRelationType(relType))
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"subjectUri": subjectURI,
			"objectUri":  objectURI,
		})
		return err
	}
}

// applyDelete removes a triple from Neo4j
func (c *Neo4jConsumer) applyDelete(ctx context.Context, tx neo4j.ManagedTransaction, change *GraphDBChange) error {
	subjectURI := change.Subject
	predicateURI := change.Predicate
	objectURI := change.Object

	relType := uriToRelationType(predicateURI)

	if strings.Contains(predicateURI, "subClassOf") {
		// Delete hierarchy relationship
		query := `
			MATCH (s:Class {uri: $subjectUri})-[r:rdfs__subClassOf]->(o:Class {uri: $objectUri})
			DELETE r
		`
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"subjectUri": subjectURI,
			"objectUri":  objectURI,
		})
		return err
	} else if change.ObjectType == "literal" {
		// Remove property
		query := fmt.Sprintf(`
			MATCH (s:Class {uri: $subjectUri})
			REMOVE s.%s
		`, sanitizePropertyName(relType))
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"subjectUri": subjectURI,
		})
		return err
	} else {
		// Delete generic relationship
		query := fmt.Sprintf(`
			MATCH (s:Class {uri: $subjectUri})-[r:%s]->(o:Class {uri: $objectUri})
			DELETE r
		`, sanitizeRelationType(relType))
		_, err := tx.Run(ctx, query, map[string]interface{}{
			"subjectUri": subjectURI,
			"objectUri":  objectURI,
		})
		return err
	}
}

// Helper functions

// uriToRelationType converts a predicate URI to a Neo4j relationship type
func uriToRelationType(uri string) string {
	// Extract local name from URI
	parts := strings.Split(uri, "#")
	if len(parts) == 2 {
		return strings.ReplaceAll(parts[1], "-", "_")
	}
	parts = strings.Split(uri, "/")
	if len(parts) > 0 {
		return strings.ReplaceAll(parts[len(parts)-1], "-", "_")
	}
	return "related_to"
}

// sanitizeRelationType ensures the relationship type is valid for Neo4j
func sanitizeRelationType(relType string) string {
	// Neo4j relationship types must be alphanumeric with underscores
	result := strings.ReplaceAll(relType, "-", "_")
	result = strings.ReplaceAll(result, ":", "__")
	result = strings.ReplaceAll(result, ".", "_")
	return result
}

// sanitizePropertyName ensures the property name is valid for Neo4j
func sanitizePropertyName(name string) string {
	result := strings.ReplaceAll(name, "-", "_")
	result = strings.ReplaceAll(result, ":", "_")
	result = strings.ReplaceAll(result, ".", "_")
	// Property names cannot start with a number
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = "p_" + result
	}
	return result
}

// extractCode extracts the concept code from a URI
func extractCode(uri string) string {
	// Handle SNOMED: http://snomed.info/sct/12345
	if strings.Contains(uri, "snomed.info/sct/") {
		parts := strings.Split(uri, "/")
		return parts[len(parts)-1]
	}
	// Handle RxNorm: http://www.nlm.nih.gov/research/umls/rxnorm/12345
	if strings.Contains(uri, "rxnorm/") {
		parts := strings.Split(uri, "/")
		return parts[len(parts)-1]
	}
	// Handle LOINC: http://loinc.org/12345-6
	if strings.Contains(uri, "loinc.org/") {
		parts := strings.Split(uri, "/")
		return parts[len(parts)-1]
	}
	// Default: use last segment
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return uri
}

// extractSystem extracts the terminology system from a URI
func extractSystem(uri string) string {
	if strings.Contains(uri, "snomed.info") {
		return "http://snomed.info/sct"
	}
	if strings.Contains(uri, "rxnorm") {
		return "http://www.nlm.nih.gov/research/umls/rxnorm"
	}
	if strings.Contains(uri, "loinc.org") {
		return "http://loinc.org"
	}
	if strings.Contains(uri, "icd-10") {
		return "http://hl7.org/fhir/sid/icd-10-cm"
	}
	return ""
}
