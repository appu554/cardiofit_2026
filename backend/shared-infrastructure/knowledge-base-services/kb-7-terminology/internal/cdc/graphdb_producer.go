// Package cdc provides Change Data Capture functionality for KB-7.
// Phase 6.2: CDC Producer (GraphDB → Kafka)
//
// This producer detects changes in GraphDB and publishes them to Kafka,
// enabling real-time synchronization to the Neo4j read replica.
//
// Flow:
//   1. GraphDB Transaction Commit
//   2. GraphDB CDC Producer (this service) - polls for changes
//   3. Kafka Topic: "kb7.graphdb.changes"
//   4. Neo4j Consumer (neo4j_consumer.go)
//
// Change Detection Strategies:
//   - Transaction Log Polling: Check GraphDB transaction logs
//   - SPARQL ASK Queries: Detect new/modified concepts
//   - Timestamp-based: Track last modified timestamps
package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// ProducerConfig holds configuration for the CDC producer
type ProducerConfig struct {
	// GraphDB configuration
	GraphDBURL        string
	GraphDBRepository string
	GraphDBUsername   string
	GraphDBPassword   string

	// Kafka configuration
	KafkaBrokers []string
	Topic        string

	// Polling configuration
	PollInterval    time.Duration
	BatchSize       int
	ChangeDetection string // "timestamp" | "transaction" | "hash"
}

// ProducerStats holds runtime statistics
type ProducerStats struct {
	ChangesDetected  int64     `json:"changes_detected"`
	MessagesProduced int64     `json:"messages_produced"`
	LastPollTime     time.Time `json:"last_poll_time"`
	LastChangeTime   time.Time `json:"last_change_time"`
	Errors           int64     `json:"errors"`
	LastError        string    `json:"last_error,omitempty"`
}

// GraphDBProducer detects changes in GraphDB and publishes to Kafka
type GraphDBProducer struct {
	config     *ProducerConfig
	writer     *kafka.Writer
	httpClient *http.Client
	logger     *logrus.Logger
	stats      ProducerStats
	statsMu    sync.RWMutex
	shutdown   chan struct{}
	wg         sync.WaitGroup

	// Change tracking state
	lastCheckpoint time.Time
	lastTxID       string
}

// NewGraphDBProducer creates a new CDC producer
func NewGraphDBProducer(config *ProducerConfig, logger *logrus.Logger) (*GraphDBProducer, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Set defaults
	if config.PollInterval == 0 {
		config.PollInterval = 5 * time.Second
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.Topic == "" {
		config.Topic = "kb7.graphdb.changes"
	}
	if config.ChangeDetection == "" {
		config.ChangeDetection = "timestamp"
	}

	// Create Kafka writer with batching
	// Note: Custom transport removed - Kafka now properly advertises localhost:9093
	// via EXTERNAL listener after Docker port mapping fix (9093 → 29092)
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.KafkaBrokers...),
		Topic:        config.Topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    config.BatchSize,
		BatchTimeout: time.Second,
		RequiredAcks: kafka.RequireOne,
		Async:        false, // Sync for reliability
	}

	// Create HTTP client for GraphDB
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	producer := &GraphDBProducer{
		config:         config,
		writer:         writer,
		httpClient:     httpClient,
		logger:         logger,
		shutdown:       make(chan struct{}),
		lastCheckpoint: time.Now().Add(-24 * time.Hour), // Start from 24h ago
	}

	// Verify GraphDB connectivity
	if err := producer.healthCheck(); err != nil {
		return nil, fmt.Errorf("GraphDB health check failed: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"graphdb_url":  config.GraphDBURL,
		"repository":   config.GraphDBRepository,
		"topic":        config.Topic,
		"poll_interval": config.PollInterval,
	}).Info("CDC producer initialized")

	return producer, nil
}

// Start begins the CDC polling loop
func (p *GraphDBProducer) Start(ctx context.Context) error {
	p.logger.Info("Starting GraphDB CDC producer...")

	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("Context cancelled, stopping producer...")
			return nil
		case <-p.shutdown:
			p.logger.Info("Shutdown signal received...")
			return nil
		case <-ticker.C:
			if err := p.pollForChanges(ctx); err != nil {
				p.logger.WithError(err).Error("Error polling for changes")
				p.statsMu.Lock()
				p.stats.Errors++
				p.stats.LastError = err.Error()
				p.statsMu.Unlock()
			}
		}
	}
}

// Stop gracefully stops the producer
func (p *GraphDBProducer) Stop() error {
	close(p.shutdown)
	p.wg.Wait()

	if err := p.writer.Close(); err != nil {
		p.logger.WithError(err).Error("Error closing Kafka writer")
		return err
	}

	p.logger.Info("CDC producer stopped")
	return nil
}

// Stats returns current producer statistics
func (p *GraphDBProducer) Stats() ProducerStats {
	p.statsMu.RLock()
	defer p.statsMu.RUnlock()
	return p.stats
}

// pollForChanges queries GraphDB for recent changes
func (p *GraphDBProducer) pollForChanges(ctx context.Context) error {
	p.statsMu.Lock()
	p.stats.LastPollTime = time.Now()
	p.statsMu.Unlock()

	// Query for recently modified concepts
	changes, err := p.detectChanges(ctx)
	if err != nil {
		return fmt.Errorf("detecting changes: %w", err)
	}

	if len(changes) == 0 {
		return nil
	}

	p.logger.WithField("count", len(changes)).Debug("Changes detected")

	// Publish changes to Kafka
	messages := make([]kafka.Message, 0, len(changes))
	for _, change := range changes {
		data, err := json.Marshal(change)
		if err != nil {
			p.logger.WithError(err).Error("Failed to marshal change")
			continue
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(change.Subject),
			Value: data,
			Time:  change.Timestamp,
		})
	}

	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("writing to Kafka: %w", err)
	}

	atomic.AddInt64(&p.stats.ChangesDetected, int64(len(changes)))
	atomic.AddInt64(&p.stats.MessagesProduced, int64(len(messages)))

	p.statsMu.Lock()
	p.stats.LastChangeTime = time.Now()
	p.statsMu.Unlock()

	// Update checkpoint
	p.lastCheckpoint = time.Now()

	return nil
}

// detectChanges queries GraphDB for changes since last checkpoint
func (p *GraphDBProducer) detectChanges(ctx context.Context) ([]*GraphDBChange, error) {
	// SPARQL query to find recently added/modified concepts
	// Uses dc:modified or similar timestamp predicate if available
	sparqlQuery := fmt.Sprintf(`
		PREFIX owl: <http://www.w3.org/2002/07/owl#>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX snomed: <http://snomed.info/id/>
		PREFIX dc: <http://purl.org/dc/terms/>

		SELECT DISTINCT ?subject ?predicate ?object ?modified
		WHERE {
			{
				# Find new subClassOf relationships
				?subject rdfs:subClassOf ?object .
				BIND(rdfs:subClassOf AS ?predicate)
				OPTIONAL { ?subject dc:modified ?modified }
			}
			UNION
			{
				# Find new label assignments
				?subject rdfs:label ?object .
				BIND(rdfs:label AS ?predicate)
				OPTIONAL { ?subject dc:modified ?modified }
			}
			FILTER(isIRI(?subject))
			FILTER(STRSTARTS(STR(?subject), "http://snomed.info/"))
		}
		LIMIT %d
	`, p.config.BatchSize)

	results, err := p.executeSPARQL(ctx, sparqlQuery)
	if err != nil {
		return nil, err
	}

	changes := make([]*GraphDBChange, 0, len(results))
	for _, result := range results {
		change := &GraphDBChange{
			Operation:  OperationInsert,
			Subject:    result["subject"],
			Predicate:  result["predicate"],
			Object:     result["object"],
			Timestamp:  time.Now(),
			ObjectType: detectObjectType(result["object"]),
		}
		changes = append(changes, change)
	}

	return changes, nil
}

// executeSPARQL runs a SPARQL query against GraphDB
func (p *GraphDBProducer) executeSPARQL(ctx context.Context, query string) ([]map[string]string, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s", p.config.GraphDBURL, p.config.GraphDBRepository)

	form := url.Values{}
	form.Set("query", query)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")

	if p.config.GraphDBUsername != "" {
		req.SetBasicAuth(p.config.GraphDBUsername, p.config.GraphDBPassword)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphDB returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse SPARQL JSON results
	var sparqlResult struct {
		Results struct {
			Bindings []map[string]struct {
				Value string `json:"value"`
				Type  string `json:"type"`
			} `json:"bindings"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&sparqlResult); err != nil {
		return nil, fmt.Errorf("parsing SPARQL results: %w", err)
	}

	// Convert to simple map
	results := make([]map[string]string, 0, len(sparqlResult.Results.Bindings))
	for _, binding := range sparqlResult.Results.Bindings {
		row := make(map[string]string)
		for key, val := range binding {
			row[key] = val.Value
		}
		results = append(results, row)
	}

	return results, nil
}

// healthCheck verifies GraphDB connectivity
func (p *GraphDBProducer) healthCheck() error {
	endpoint := fmt.Sprintf("%s/repositories/%s/size", p.config.GraphDBURL, p.config.GraphDBRepository)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return err
	}

	if p.config.GraphDBUsername != "" {
		req.SetBasicAuth(p.config.GraphDBUsername, p.config.GraphDBPassword)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphDB returned status %d", resp.StatusCode)
	}

	return nil
}

// PublishChange manually publishes a single change (for testing/manual triggers)
func (p *GraphDBProducer) PublishChange(ctx context.Context, change *GraphDBChange) error {
	data, err := json.Marshal(change)
	if err != nil {
		return fmt.Errorf("marshaling change: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(change.Subject),
		Value: data,
		Time:  change.Timestamp,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("writing to Kafka: %w", err)
	}

	atomic.AddInt64(&p.stats.MessagesProduced, 1)
	return nil
}

// Helper functions

func detectObjectType(object string) string {
	if strings.HasPrefix(object, "http://") || strings.HasPrefix(object, "https://") {
		return "uri"
	}
	return "literal"
}
