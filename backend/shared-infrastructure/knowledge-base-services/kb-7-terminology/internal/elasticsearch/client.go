package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// Client wraps the Elasticsearch client with KB7-specific functionality
type Client struct {
	es     *elasticsearch.Client
	config *Config
}

// Config holds Elasticsearch configuration
type Config struct {
	URLs               []string      `json:"urls"`
	Username           string        `json:"username"`
	Password           string        `json:"password"`
	MaxRetries         int           `json:"max_retries"`
	RequestTimeout     time.Duration `json:"request_timeout"`
	MaxIndexingWorkers int           `json:"max_indexing_workers"`
	BulkSize           int           `json:"bulk_size"`
	FlushInterval      time.Duration `json:"flush_interval"`
}

// DefaultConfig returns default Elasticsearch configuration
func DefaultConfig() *Config {
	return &Config{
		URLs:               []string{"http://localhost:9200"},
		MaxRetries:         3,
		RequestTimeout:     30 * time.Second,
		MaxIndexingWorkers: 4,
		BulkSize:           1000,
		FlushInterval:      5 * time.Second,
	}
}

// NewClient creates a new Elasticsearch client for KB7
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	cfg := elasticsearch.Config{
		Addresses: config.URLs,
		Username:  config.Username,
		Password:  config.Password,
		RetryOnStatus: []int{502, 503, 504, 429},
		MaxRetries:    config.MaxRetries,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: config.RequestTimeout,
		},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	client := &Client{
		es:     es,
		config: config,
	}

	// Verify connection
	if err := client.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping Elasticsearch: %w", err)
	}

	log.Printf("Connected to Elasticsearch cluster: %v", config.URLs)
	return client, nil
}

// Ping checks if Elasticsearch is reachable
func (c *Client) Ping(ctx context.Context) error {
	res, err := c.es.Ping(
		c.es.Ping.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ping failed with status: %s", res.Status())
	}

	return nil
}

// GetClusterHealth returns cluster health information
func (c *Client) GetClusterHealth(ctx context.Context) (*ClusterHealth, error) {
	res, err := c.es.Cluster.Health(
		c.es.Cluster.Health.WithContext(ctx),
		c.es.Cluster.Health.WithWaitForStatus("yellow"),
		c.es.Cluster.Health.WithTimeout(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("cluster health request failed: %s", res.Status())
	}

	var health ClusterHealth
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode cluster health response: %w", err)
	}

	return &health, nil
}

// ClusterHealth represents Elasticsearch cluster health
type ClusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float64 `json:"active_shards_percent_as_number"`
}

// CreateIndex creates an index with the specified mapping
func (c *Client) CreateIndex(ctx context.Context, indexName string, mapping map[string]interface{}) error {
	// Check if index already exists
	exists, err := c.IndexExists(ctx, indexName)
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}

	if exists {
		return fmt.Errorf("index %s already exists", indexName)
	}

	// Convert mapping to JSON
	body, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal index mapping: %w", err)
	}

	// Create index
	res, err := c.es.Indices.Create(
		indexName,
		c.es.Indices.Create.WithContext(ctx),
		c.es.Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var errResp map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("index creation failed with status %s", res.Status())
		}
		return fmt.Errorf("index creation failed: %v", errResp)
	}

	log.Printf("Created index: %s", indexName)
	return nil
}

// IndexExists checks if an index exists
func (c *Client) IndexExists(ctx context.Context, indexName string) (bool, error) {
	res, err := c.es.Indices.Exists(
		[]string{indexName},
		c.es.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()

	return res.StatusCode == 200, nil
}

// DeleteIndex deletes an index
func (c *Client) DeleteIndex(ctx context.Context, indexName string) error {
	res, err := c.es.Indices.Delete(
		[]string{indexName},
		c.es.Indices.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("index deletion failed with status: %s", res.Status())
	}

	log.Printf("Deleted index: %s", indexName)
	return nil
}

// IndexDocument indexes a single document
func (c *Client) IndexDocument(ctx context.Context, indexName, documentID string, document interface{}) error {
	body, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: documentID,
		Body:       bytes.NewReader(body),
		Refresh:    "false", // Don't refresh immediately for better performance
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var errResp map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("document indexing failed with status %s", res.Status())
		}
		return fmt.Errorf("document indexing failed: %v", errResp)
	}

	return nil
}

// BulkIndexDocuments performs bulk indexing of documents
func (c *Client) BulkIndexDocuments(ctx context.Context, indexName string, documents []BulkDocument) error {
	if len(documents) == 0 {
		return nil
	}

	var buf bytes.Buffer

	for _, doc := range documents {
		// Index action
		indexAction := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexName,
				"_id":    doc.ID,
			},
		}

		actionBytes, err := json.Marshal(indexAction)
		if err != nil {
			return fmt.Errorf("failed to marshal index action: %w", err)
		}

		buf.Write(actionBytes)
		buf.WriteByte('\n')

		// Document
		docBytes, err := json.Marshal(doc.Source)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}

		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	res, err := c.es.Bulk(
		bytes.NewReader(buf.Bytes()),
		c.es.Bulk.WithContext(ctx),
		c.es.Bulk.WithRefresh("false"),
	)
	if err != nil {
		return fmt.Errorf("bulk request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk request failed with status: %s", res.Status())
	}

	// Parse response to check for individual document errors
	var bulkRes BulkResponse
	if err := json.NewDecoder(res.Body).Decode(&bulkRes); err != nil {
		return fmt.Errorf("failed to decode bulk response: %w", err)
	}

	if bulkRes.Errors {
		// Log individual errors but don't fail the entire batch
		errorCount := 0
		for _, item := range bulkRes.Items {
			if item.Index.Error != nil {
				log.Printf("Bulk indexing error for document %s: %v", item.Index.ID, item.Index.Error)
				errorCount++
			}
		}
		log.Printf("Bulk indexing completed with %d errors out of %d documents", errorCount, len(documents))
	}

	return nil
}

// BulkDocument represents a document for bulk indexing
type BulkDocument struct {
	ID     string      `json:"id"`
	Source interface{} `json:"source"`
}

// BulkResponse represents the response from a bulk request
type BulkResponse struct {
	Took   int  `json:"took"`
	Errors bool `json:"errors"`
	Items  []struct {
		Index struct {
			ID     string                 `json:"_id"`
			Status int                    `json:"status"`
			Error  map[string]interface{} `json:"error,omitempty"`
		} `json:"index"`
	} `json:"items"`
}

// Search performs a search query
func (c *Client) Search(ctx context.Context, indexName string, query map[string]interface{}) (*SearchResponse, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(ctx),
		c.es.Search.WithIndex(indexName),
		c.es.Search.WithBody(bytes.NewReader(body)),
		c.es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var errResp map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("search failed with status %s", res.Status())
		}
		return nil, fmt.Errorf("search failed: %v", errResp)
	}

	var searchRes SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&searchRes); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return &searchRes, nil
}

// SearchResponse represents a search response
type SearchResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Hits     struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string                 `json:"_index"`
			ID     string                 `json:"_id"`
			Score  float64                `json:"_score"`
			Source map[string]interface{} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}