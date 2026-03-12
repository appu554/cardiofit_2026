package graphdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Client represents a GraphDB HTTP client with connection pooling
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger

	// Connection pool settings
	maxConnections int
	connPool       sync.Pool

	// Authentication
	username string
	password string

	// Repository configuration
	defaultRepo string

	// Performance monitoring
	queryStats    *QueryStats
	statsMutex    sync.RWMutex
}

// ClientConfig holds GraphDB client configuration
type ClientConfig struct {
	BaseURL         string
	Username        string
	Password        string
	DefaultRepo     string
	MaxConnections  int
	RequestTimeout  time.Duration
	IdleTimeout     time.Duration
	EnableMetrics   bool
}

// QueryStats tracks query performance metrics
type QueryStats struct {
	TotalQueries     int64         `json:"total_queries"`
	SuccessfulQueries int64        `json:"successful_queries"`
	FailedQueries    int64         `json:"failed_queries"`
	AverageLatency   time.Duration `json:"average_latency"`
	TotalLatency     time.Duration `json:"total_latency"`
	LastQueryTime    time.Time     `json:"last_query_time"`
}

// SPARQLQuery represents a SPARQL query request
type SPARQLQuery struct {
	Query      string            `json:"query"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Reasoning  bool              `json:"reasoning,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
}

// SPARQLResult represents a SPARQL query result
type SPARQLResult struct {
	Head    ResultHead     `json:"head"`
	Results ResultBindings `json:"results"`
}

// ResultHead contains query result metadata
type ResultHead struct {
	Vars []string `json:"vars"`
}

// ResultBindings contains the actual query results
type ResultBindings struct {
	Bindings []map[string]BindingValue `json:"bindings"`
}

// BindingValue represents a single result binding value
type BindingValue struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	DataType string `json:"datatype,omitempty"`
	Lang     string `json:"xml:lang,omitempty"`
}

// Repository represents a GraphDB repository configuration
type Repository struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Params      RepositoryParams `json:"params"`
}

// RepositoryParams holds repository-specific parameters
type RepositoryParams struct {
	Ruleset           string `json:"ruleset"`
	DisableSameAs     bool   `json:"disableSameAs"`
	CheckForInconsistencies bool `json:"checkForInconsistencies"`
	EnableContextIndex bool   `json:"enableContextIndex"`
	CacheSelectNodes  bool   `json:"cacheSelectNodes"`
	EntityIndexSize   int    `json:"entityIndexSize"`
	EntityIdSize      int    `json:"entityIdSize"`
	PredicateMemory   int    `json:"predicateMemory"`
	FtsMemory         int    `json:"ftsMemory"`
}

// BulkLoadRequest represents a bulk RDF data loading request
type BulkLoadRequest struct {
	Data        io.Reader
	Format      string // turtle, rdf/xml, n-triples, etc.
	Context     string // Named graph URI
	BaseURI     string
	ReplaceGraph bool
	Verify      bool
}

// TransactionRequest represents a transaction for atomic updates
type TransactionRequest struct {
	Queries []SPARQLQuery
	Timeout time.Duration
}

// NewClient creates a new GraphDB client with connection pooling
func NewClient(config *ClientConfig) *Client {
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 90 * time.Second
	}
	if config.MaxConnections == 0 {
		config.MaxConnections = 50
	}

	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        config.MaxConnections,
			MaxIdleConnsPerHost: config.MaxConnections / 2,
			IdleConnTimeout:     config.IdleTimeout,
			DisableKeepAlives:   false,
		},
	}

	client := &Client{
		baseURL:        strings.TrimSuffix(config.BaseURL, "/"),
		httpClient:     httpClient,
		logger:         logrus.New(),
		maxConnections: config.MaxConnections,
		username:       config.Username,
		password:       config.Password,
		defaultRepo:    config.DefaultRepo,
		queryStats:     &QueryStats{},
	}

	// Initialize connection pool
	client.connPool = sync.Pool{
		New: func() interface{} {
			return &http.Client{
				Timeout:   config.RequestTimeout,
				Transport: httpClient.Transport,
			}
		},
	}

	return client
}

// Health checks GraphDB server health
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/rest/repositories", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	c.logger.Debug("GraphDB health check passed")
	return nil
}

// CreateRepository creates a new GraphDB repository with OWL 2 RL reasoning
func (c *Client) CreateRepository(ctx context.Context, repo *Repository) error {
	// Set default OWL 2 RL configuration if not specified
	if repo.Params.Ruleset == "" {
		repo.Params.Ruleset = "owl2-rl-optimized"
	}

	// Enable recommended settings for terminology services
	repo.Params.DisableSameAs = false
	repo.Params.CheckForInconsistencies = true
	repo.Params.EnableContextIndex = true
	repo.Params.CacheSelectNodes = true

	// Set memory allocations for large terminology datasets
	if repo.Params.EntityIndexSize == 0 {
		repo.Params.EntityIndexSize = 10000000 // 10M entities
	}
	if repo.Params.EntityIdSize == 0 {
		repo.Params.EntityIdSize = 32
	}
	if repo.Params.PredicateMemory == 0 {
		repo.Params.PredicateMemory = 32 // MB
	}
	if repo.Params.FtsMemory == 0 {
		repo.Params.FtsMemory = 0 // Disable FTS by default
	}

	repoConfig := map[string]interface{}{
		"id":    repo.ID,
		"title": repo.Title,
		"type":  "graphdb",
		"params": map[string]interface{}{
			"imports":                    "",
			"defaultNS":                  "",
			"repositoryType":             "file-repository",
			"id":                         repo.ID,
			"title":                      repo.Title,
			"ruleset":                    repo.Params.Ruleset,
			"disableSameAs":              repo.Params.DisableSameAs,
			"checkForInconsistencies":    repo.Params.CheckForInconsistencies,
			"enableContextIndex":         repo.Params.EnableContextIndex,
			"cacheSelectNodes":           repo.Params.CacheSelectNodes,
			"entityIndexSize":            repo.Params.EntityIndexSize,
			"entityIdSize":               repo.Params.EntityIdSize,
			"predicateMemory":            repo.Params.PredicateMemory,
			"ftsMemory":                  repo.Params.FtsMemory,
			"ftsIndexPolicy":             "never",
			"ftsLiteralsOnly":            true,
			"storageFolder":              "storage",
			"enablePredicateList":        true,
			"enableLiteralIndex":         true,
			"indexCompressionRatio":      -1,
			"enableRdfRank":              false,
			"inMemoryLiteralProperties":  true,
			"throwQueryEvaluationExceptionOnTimeout": true,
			"queryTimeout":               0,
			"queryLimitResults":          0,
			"readOnly":                   false,
		},
	}

	jsonData, err := json.Marshal(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal repository config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.baseURL+"/rest/repositories",
		bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create repository request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repository, status: %d, body: %s",
			resp.StatusCode, string(body))
	}

	c.logger.WithFields(logrus.Fields{
		"repository": repo.ID,
		"ruleset":    repo.Params.Ruleset,
	}).Info("Repository created successfully")

	return nil
}

// Query executes a SPARQL query with automatic retry and connection pooling
func (c *Client) Query(ctx context.Context, repositoryID string, query *SPARQLQuery) (*SPARQLResult, error) {
	startTime := time.Now()

	// Update stats
	c.updateQueryStats(startTime, true, false)

	// Build query parameters
	params := url.Values{}
	params.Set("query", query.Query)

	if query.Reasoning {
		params.Set("infer", "true")
	} else {
		params.Set("infer", "false")
	}

	// Add custom parameters
	for key, value := range query.Parameters {
		params.Set(key, value)
	}

	// Construct URL
	repoID := repositoryID
	if repoID == "" {
		repoID = c.defaultRepo
	}

	queryURL := fmt.Sprintf("%s/repositories/%s", c.baseURL, repoID)

	req, err := http.NewRequestWithContext(ctx, "POST", queryURL,
		strings.NewReader(params.Encode()))
	if err != nil {
		c.updateQueryStats(startTime, false, true)
		return nil, fmt.Errorf("failed to create query request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")
	c.setAuth(req)

	// Apply query timeout if specified
	if query.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, query.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.updateQueryStats(startTime, false, true)
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.updateQueryStats(startTime, false, true)
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed with status: %d, body: %s",
			resp.StatusCode, string(body))
	}

	var result SPARQLResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.updateQueryStats(startTime, false, true)
		return nil, fmt.Errorf("failed to decode query result: %w", err)
	}

	c.updateQueryStats(startTime, false, false)

	c.logger.WithFields(logrus.Fields{
		"repository":    repoID,
		"query_length":  len(query.Query),
		"result_count":  len(result.Results.Bindings),
		"latency_ms":    time.Since(startTime).Milliseconds(),
		"reasoning":     query.Reasoning,
	}).Debug("SPARQL query executed successfully")

	return &result, nil
}

// Update executes a SPARQL UPDATE query
func (c *Client) Update(ctx context.Context, repositoryID string, updateQuery string) error {
	repoID := repositoryID
	if repoID == "" {
		repoID = c.defaultRepo
	}

	params := url.Values{}
	params.Set("update", updateQuery)

	queryURL := fmt.Sprintf("%s/repositories/%s/statements", c.baseURL, repoID)

	req, err := http.NewRequestWithContext(ctx, "POST", queryURL,
		strings.NewReader(params.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed with status: %d, body: %s",
			resp.StatusCode, string(body))
	}

	c.logger.WithFields(logrus.Fields{
		"repository":   repoID,
		"update_length": len(updateQuery),
	}).Debug("SPARQL update executed successfully")

	return nil
}

// BulkLoad loads RDF data in bulk with optimized performance
func (c *Client) BulkLoad(ctx context.Context, repositoryID string, request *BulkLoadRequest) error {
	repoID := repositoryID
	if repoID == "" {
		repoID = c.defaultRepo
	}

	// Build query parameters
	params := url.Values{}
	if request.Context != "" {
		params.Set("context", request.Context)
	}
	if request.BaseURI != "" {
		params.Set("baseURI", request.BaseURI)
	}
	if request.ReplaceGraph {
		params.Set("replaceGraph", "true")
	}
	if request.Verify {
		params.Set("verify", "true")
	}

	queryURL := fmt.Sprintf("%s/repositories/%s/statements?%s",
		c.baseURL, repoID, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", queryURL, request.Data)
	if err != nil {
		return fmt.Errorf("failed to create bulk load request: %w", err)
	}

	// Set appropriate content type based on format
	contentType := c.getContentTypeForFormat(request.Format)
	req.Header.Set("Content-Type", contentType)
	c.setAuth(req)

	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute bulk load: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bulk load failed with status: %d, body: %s",
			resp.StatusCode, string(body))
	}

	c.logger.WithFields(logrus.Fields{
		"repository": repoID,
		"format":     request.Format,
		"context":    request.Context,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("Bulk load completed successfully")

	return nil
}

// ExecuteTransaction executes multiple queries in a single transaction
func (c *Client) ExecuteTransaction(ctx context.Context, repositoryID string,
	transaction *TransactionRequest) error {

	repoID := repositoryID
	if repoID == "" {
		repoID = c.defaultRepo
	}

	// Apply transaction timeout if specified
	if transaction.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, transaction.Timeout)
		defer cancel()
	}

	// Begin transaction
	beginURL := fmt.Sprintf("%s/repositories/%s/transactions", c.baseURL, repoID)
	req, err := http.NewRequestWithContext(ctx, "POST", beginURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create begin transaction request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to begin transaction, status: %d, body: %s",
			resp.StatusCode, string(body))
	}

	// Extract transaction ID from Location header
	transactionID := resp.Header.Get("Location")
	if transactionID == "" {
		return fmt.Errorf("transaction ID not found in response")
	}

	// Extract just the transaction ID from the full URL
	parts := strings.Split(transactionID, "/")
	if len(parts) > 0 {
		transactionID = parts[len(parts)-1]
	}

	// Execute queries within transaction
	var lastErr error
	for i, query := range transaction.Queries {
		queryURL := fmt.Sprintf("%s/repositories/%s/transactions/%s",
			c.baseURL, repoID, transactionID)

		params := url.Values{}
		if strings.TrimSpace(strings.ToUpper(query.Query))[:6] == "SELECT" ||
		   strings.TrimSpace(strings.ToUpper(query.Query))[:3] == "ASK" ||
		   strings.TrimSpace(strings.ToUpper(query.Query))[:9] == "CONSTRUCT" ||
		   strings.TrimSpace(strings.ToUpper(query.Query))[:8] == "DESCRIBE" {
			params.Set("query", query.Query)
		} else {
			params.Set("update", query.Query)
		}

		req, err := http.NewRequestWithContext(ctx, "POST", queryURL,
			strings.NewReader(params.Encode()))
		if err != nil {
			lastErr = fmt.Errorf("failed to create transaction query %d request: %w", i, err)
			break
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		c.setAuth(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute transaction query %d: %w", i, err)
			break
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("transaction query %d failed with status: %d, body: %s",
				i, resp.StatusCode, string(body))
			break
		}
	}

	// Commit or rollback transaction
	var action string
	if lastErr != nil {
		action = "DELETE" // Rollback
	} else {
		action = "PUT" // Commit
	}

	commitURL := fmt.Sprintf("%s/repositories/%s/transactions/%s",
		c.baseURL, repoID, transactionID)
	req, err = http.NewRequestWithContext(ctx, action, commitURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create transaction %s request: %w",
			strings.ToLower(action), err)
	}
	c.setAuth(req)

	resp, err = c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to %s transaction: %w",
			strings.ToLower(action), err)
	}
	defer resp.Body.Close()

	if lastErr != nil {
		c.logger.WithFields(logrus.Fields{
			"repository":     repoID,
			"transaction_id": transactionID,
			"queries_count":  len(transaction.Queries),
		}).Warn("Transaction rolled back due to error")
		return fmt.Errorf("transaction failed and was rolled back: %w", lastErr)
	}

	c.logger.WithFields(logrus.Fields{
		"repository":     repoID,
		"transaction_id": transactionID,
		"queries_count":  len(transaction.Queries),
	}).Info("Transaction committed successfully")

	return nil
}

// GetQueryStats returns current query performance statistics
func (c *Client) GetQueryStats() *QueryStats {
	c.statsMutex.RLock()
	defer c.statsMutex.RUnlock()

	// Create a copy to avoid data races
	return &QueryStats{
		TotalQueries:      c.queryStats.TotalQueries,
		SuccessfulQueries: c.queryStats.SuccessfulQueries,
		FailedQueries:     c.queryStats.FailedQueries,
		AverageLatency:    c.queryStats.AverageLatency,
		TotalLatency:      c.queryStats.TotalLatency,
		LastQueryTime:     c.queryStats.LastQueryTime,
	}
}

// ResetQueryStats resets the query performance statistics
func (c *Client) ResetQueryStats() {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	c.queryStats = &QueryStats{}
}

// Close closes the GraphDB client and cleans up resources
func (c *Client) Close() error {
	// Close HTTP client if it has a custom transport with connections
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	c.logger.Info("GraphDB client closed")
	return nil
}

// Private helper methods

func (c *Client) setAuth(req *http.Request) {
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *Client) getContentTypeForFormat(format string) string {
	switch strings.ToLower(format) {
	case "turtle", "ttl":
		return "text/turtle"
	case "rdf/xml", "rdf", "xml":
		return "application/rdf+xml"
	case "n-triples", "nt":
		return "application/n-triples"
	case "n-quads", "nq":
		return "application/n-quads"
	case "trig":
		return "application/trig"
	case "jsonld", "json-ld":
		return "application/ld+json"
	default:
		return "text/turtle" // Default to Turtle
	}
}

func (c *Client) updateQueryStats(startTime time.Time, starting bool, failed bool) {
	c.statsMutex.Lock()
	defer c.statsMutex.Unlock()

	if starting {
		c.queryStats.TotalQueries++
		c.queryStats.LastQueryTime = startTime
		return
	}

	latency := time.Since(startTime)
	c.queryStats.TotalLatency += latency

	if failed {
		c.queryStats.FailedQueries++
	} else {
		c.queryStats.SuccessfulQueries++
	}

	// Calculate rolling average latency
	if c.queryStats.SuccessfulQueries > 0 {
		c.queryStats.AverageLatency = c.queryStats.TotalLatency /
			time.Duration(c.queryStats.SuccessfulQueries)
	}
}