package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// GraphDBClient provides interface to OntoNext GraphDB triplestore
type GraphDBClient struct {
	baseURL      string
	repository   string
	httpClient   *http.Client
	logger       *logrus.Logger
	username     string
	password     string
}

// SPARQLQuery represents a SPARQL query request
type SPARQLQuery struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"bindings,omitempty"`
	Format    string            `json:"format,omitempty"`
}

// SPARQLResults represents SPARQL query results
type SPARQLResults struct {
	Head    SPARQLHead           `json:"head"`
	Results SPARQLResultSet      `json:"results"`
	Boolean *bool                `json:"boolean,omitempty"` // For ASK query responses
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// IsAskResult checks if this is an ASK query result
func (r *SPARQLResults) IsAskResult() bool {
	return r.Boolean != nil
}

// GetBooleanResult returns the boolean result for ASK queries
func (r *SPARQLResults) GetBooleanResult() bool {
	if r.Boolean != nil {
		return *r.Boolean
	}
	return false
}

// SPARQLHead contains query metadata
type SPARQLHead struct {
	Vars []string `json:"vars"`
}

// SPARQLResultSet contains result bindings
type SPARQLResultSet struct {
	Bindings []map[string]SPARQLBinding `json:"bindings"`
}

// SPARQLBinding represents a variable binding
type SPARQLBinding struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	DataType string `json:"datatype,omitempty"`
	Lang     string `json:"xml:lang,omitempty"`
}

// TripleData represents RDF triple for insertion
type TripleData struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Context   string `json:"context,omitempty"`
}

// NewGraphDBClient creates a new GraphDB client
func NewGraphDBClient(baseURL, repository string, logger *logrus.Logger) *GraphDBClient {
	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	return &GraphDBClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		repository: repository,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// SetAuthentication configures basic authentication
func (g *GraphDBClient) SetAuthentication(username, password string) {
	g.username = username
	g.password = password
}

// ExecuteSPARQL executes a SPARQL query against the repository
func (g *GraphDBClient) ExecuteSPARQL(ctx context.Context, query *SPARQLQuery) (*SPARQLResults, error) {
	endpoint := fmt.Sprintf("%s/repositories/%s", g.baseURL, g.repository)

	// Prepare form data
	data := url.Values{}
	data.Set("query", query.Query)

	// Set default format if not specified
	format := query.Format
	if format == "" {
		format = "application/sparql-results+json"
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating SPARQL request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", format)

	// Add authentication if configured
	if g.username != "" && g.password != "" {
		req.SetBasicAuth(g.username, g.password)
	}

	// Execute request
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing SPARQL query: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphDB error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading SPARQL response: %w", err)
	}

	var results SPARQLResults
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("parsing SPARQL results: %w", err)
	}

	// Add metadata
	results.Meta = map[string]interface{}{
		"executionTime": time.Now().Format(time.RFC3339),
		"repository":    g.repository,
		"queryType":     g.detectQueryType(query.Query),
	}

	g.logger.WithFields(logrus.Fields{
		"repository": g.repository,
		"queryType":  results.Meta["queryType"],
		"resultCount": len(results.Results.Bindings),
	}).Debug("SPARQL query executed successfully")

	return &results, nil
}

// LoadTurtleFile loads a Turtle (.ttl) file into the repository
func (g *GraphDBClient) LoadTurtleFile(ctx context.Context, filepath string, context string) error {
	// Read file content
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("reading turtle file %s: %w", filepath, err)
	}

	return g.LoadTurtleData(ctx, content, context)
}

// LoadTurtleData loads Turtle data into the repository
func (g *GraphDBClient) LoadTurtleData(ctx context.Context, data []byte, context string) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", g.baseURL, g.repository)

	// Add context parameter if specified
	if context != "" {
		endpoint += "?context=" + url.QueryEscape(context)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating turtle load request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-turtle")

	// Add authentication if configured
	if g.username != "" && g.password != "" {
		req.SetBasicAuth(g.username, g.password)
	}

	// Execute request
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("loading turtle data: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GraphDB load error %d: %s", resp.StatusCode, string(body))
	}

	g.logger.WithFields(logrus.Fields{
		"repository": g.repository,
		"context":    context,
		"dataSize":   len(data),
	}).Info("Turtle data loaded successfully")

	return nil
}

// UploadTurtle is an alias for LoadTurtleData that accepts a string
func (g *GraphDBClient) UploadTurtle(ctx context.Context, turtleContent string, context string) error {
	return g.LoadTurtleData(ctx, []byte(turtleContent), context)
}

// InsertTriples inserts RDF triples into the repository
func (g *GraphDBClient) InsertTriples(ctx context.Context, triples []TripleData) error {
	// Convert triples to SPARQL INSERT
	var insertStatements []string
	for _, triple := range triples {
		stmt := fmt.Sprintf("<%s> <%s> ", triple.Subject, triple.Predicate)

		// Handle object type (URI vs literal)
		if strings.HasPrefix(triple.Object, "http://") || strings.HasPrefix(triple.Object, "https://") {
			stmt += fmt.Sprintf("<%s>", triple.Object)
		} else {
			// Escape quotes in literals
			escaped := strings.ReplaceAll(triple.Object, "\"", "\\\"")
			stmt += fmt.Sprintf("\"%s\"", escaped)
		}

		insertStatements = append(insertStatements, stmt)
	}

	// Build SPARQL INSERT query
	query := fmt.Sprintf(`
		INSERT DATA {
			%s
		}
	`, strings.Join(insertStatements, " .\n"))

	// Execute update
	return g.ExecuteUpdate(ctx, query)
}

// ExecuteUpdate executes a SPARQL UPDATE query
func (g *GraphDBClient) ExecuteUpdate(ctx context.Context, updateQuery string) error {
	endpoint := fmt.Sprintf("%s/repositories/%s/statements", g.baseURL, g.repository)

	// Prepare form data
	data := url.Values{}
	data.Set("update", updateQuery)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("creating SPARQL update request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Add authentication if configured
	if g.username != "" && g.password != "" {
		req.SetBasicAuth(g.username, g.password)
	}

	// Execute request
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing SPARQL update: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GraphDB update error %d: %s", resp.StatusCode, string(body))
	}

	g.logger.WithFields(logrus.Fields{
		"repository": g.repository,
		"operation":  "update",
	}).Debug("SPARQL update executed successfully")

	return nil
}

// GetRepositoryInfo retrieves repository information
func (g *GraphDBClient) GetRepositoryInfo(ctx context.Context) (map[string]interface{}, error) {
	endpoint := fmt.Sprintf("%s/rest/repositories/%s", g.baseURL, g.repository)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating repository info request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	// Add authentication if configured
	if g.username != "" && g.password != "" {
		req.SetBasicAuth(g.username, g.password)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting repository info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphDB info error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading repository info: %w", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parsing repository info: %w", err)
	}

	return info, nil
}

// HealthCheck verifies GraphDB connectivity
func (g *GraphDBClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/rest/repositories", g.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating health check request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphDB health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// Clinical Terminology Specific Methods

// GetConcept retrieves a clinical concept by URI
func (g *GraphDBClient) GetConcept(ctx context.Context, conceptURI string) (*SPARQLResults, error) {
	query := &SPARQLQuery{
		Query: fmt.Sprintf(`
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
			PREFIX pav: <http://purl.org/pav/>

			SELECT ?property ?value WHERE {
				<%s> ?property ?value .
			}
			ORDER BY ?property
		`, conceptURI),
	}

	return g.ExecuteSPARQL(ctx, query)
}

// GetMappings retrieves concept mappings for a given source code
func (g *GraphDBClient) GetMappings(ctx context.Context, sourceCode string, sourceSystem string) (*SPARQLResults, error) {
	query := &SPARQLQuery{
		Query: fmt.Sprintf(`
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
			PREFIX pav: <http://purl.org/pav/>

			SELECT ?mapping ?targetCode ?confidence ?safetyLevel ?reviewStatus WHERE {
				?mapping a kb7:ConceptMapping ;
					kb7:sourceCode "%s" ;
					kb7:targetCode ?targetCode ;
					kb7:mappingConfidence ?confidence .
				OPTIONAL { ?mapping kb7:safetyLevel ?safetyLevel }
				OPTIONAL { ?mapping kb7:validationStatus ?reviewStatus }
			}
			ORDER BY DESC(?confidence)
		`, sourceCode),
	}

	return g.ExecuteSPARQL(ctx, query)
}

// GetDrugInteractions retrieves drug interactions for a medication concept
func (g *GraphDBClient) GetDrugInteractions(ctx context.Context, medicationURI string) (*SPARQLResults, error) {
	query := &SPARQLQuery{
		Query: fmt.Sprintf(`
			PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>

			SELECT ?interaction ?interactingDrug ?severity ?mechanism WHERE {
				<%s> kb7:hasInteraction ?interaction .
				?interaction kb7:severity ?severity ;
					kb7:mechanism ?mechanism ;
					kb7:involves ?interactingDrug .
				FILTER(?interactingDrug != <%s>)
			}
			ORDER BY ?severity
		`, medicationURI, medicationURI),
	}

	return g.ExecuteSPARQL(ctx, query)
}

// Utility methods

func (g *GraphDBClient) detectQueryType(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))
	switch {
	case strings.HasPrefix(query, "SELECT"):
		return "SELECT"
	case strings.HasPrefix(query, "ASK"):
		return "ASK"
	case strings.HasPrefix(query, "CONSTRUCT"):
		return "CONSTRUCT"
	case strings.HasPrefix(query, "DESCRIBE"):
		return "DESCRIBE"
	case strings.HasPrefix(query, "INSERT"):
		return "INSERT"
	case strings.HasPrefix(query, "DELETE"):
		return "DELETE"
	default:
		return "UNKNOWN"
	}
}