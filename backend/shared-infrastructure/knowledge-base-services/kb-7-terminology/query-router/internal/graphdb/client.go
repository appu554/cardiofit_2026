package graphdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cardiofit/kb7-query-router/internal/postgres"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"
)

// Client represents a GraphDB client for SPARQL queries
type Client struct {
	baseURL    string
	httpClient *retryablehttp.Client
	logger     *logrus.Logger
	repoID     string
}

// SPARQLResult represents a SPARQL query result
type SPARQLResult struct {
	Head    Head       `json:"head"`
	Results Results    `json:"results"`
	Query   string     `json:"query,omitempty"`
	Time    time.Time  `json:"execution_time"`
}

// Head contains SPARQL result metadata
type Head struct {
	Vars []string `json:"vars"`
}

// Results contains SPARQL bindings
type Results struct {
	Bindings []map[string]Binding `json:"bindings"`
}

// Binding represents a SPARQL variable binding
type Binding struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// DrugInteraction represents a drug interaction from GraphDB
type DrugInteraction struct {
	ID          string `json:"id"`
	Drug1       string `json:"drug1"`
	Drug2       string `json:"drug2"`
	SafetyLevel string `json:"safety_level"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Evidence    string `json:"evidence"`
}

// ConceptHierarchy represents concept hierarchy relationships
type ConceptHierarchy struct {
	Concept     string   `json:"concept"`
	Label       string   `json:"label"`
	Parents     []string `json:"parents"`
	Children    []string `json:"children"`
	Depth       int      `json:"depth"`
	Ancestors   []string `json:"ancestors"`
	Descendants []string `json:"descendants"`
}

// SemanticRelationship represents semantic relationships between concepts
type SemanticRelationship struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Label     string `json:"label,omitempty"`
	Type      string `json:"type"`
}

// NewClient creates a new GraphDB client
func NewClient(baseURL string) (*Client, error) {
	// Setup retry client
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.HTTPClient.Timeout = 30 * time.Second

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	client := &Client{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		httpClient: retryClient,
		logger:     logger,
		repoID:     "kb7-terminology", // Default repository
	}

	// Test connection
	if err := client.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to GraphDB: %w", err)
	}

	return client, nil
}

// SPARQLQuery executes a SPARQL SELECT query
func (c *Client) SPARQLQuery(ctx context.Context, query string) (*SPARQLResult, error) {
	start := time.Now()
	defer func() {
		c.logger.WithFields(logrus.Fields{
			"query_length": len(query),
			"duration":     time.Since(start),
		}).Debug("SPARQL query completed")
	}()

	// Prepare request
	url := fmt.Sprintf("%s/repositories/%s", c.baseURL, c.repoID)
	reqBody := fmt.Sprintf("query=%s", query)

	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", url, strings.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/sparql-results+json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SPARQL query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SPARQL query failed with status %d", resp.StatusCode)
	}

	// Parse response
	var result SPARQLResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode SPARQL response: %w", err)
	}

	result.Query = query
	result.Time = time.Now()

	return &result, nil
}

// FindSubconcepts finds subconcepts using SPARQL reasoning
func (c *Client) FindSubconcepts(ctx context.Context, system, parentCode string, limit int) ([]postgres.Concept, error) {
	// Map system to namespace prefix
	prefix := c.getNamespacePrefix(system)

	sparql := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX %s: <%s>

		SELECT ?concept ?code ?label ?definition WHERE {
			?concept rdfs:subClassOf* %s:%s .
			?concept skos:notation ?code .
			OPTIONAL { ?concept rdfs:label ?label }
			OPTIONAL { ?concept skos:definition ?definition }
			FILTER(?concept != %s:%s)
		}
		ORDER BY ?code
		LIMIT %d
	`, prefix, c.getNamespaceURI(system), prefix, parentCode, prefix, parentCode, limit)

	result, err := c.SPARQLQuery(ctx, sparql)
	if err != nil {
		return nil, fmt.Errorf("failed to find subconcepts: %w", err)
	}

	return c.convertSPARQLToConcepts(result, system), nil
}

// CheckDrugInteractions checks for drug interactions using SPARQL
func (c *Client) CheckDrugInteractions(ctx context.Context, medicationCodes []string) ([]DrugInteraction, error) {
	if len(medicationCodes) < 2 {
		return []DrugInteraction{}, nil
	}

	// Build medication filter
	medicationFilter := make([]string, len(medicationCodes))
	for i, code := range medicationCodes {
		medicationFilter[i] = fmt.Sprintf("kb7:%s", code)
	}
	medicationList := strings.Join(medicationFilter, ", ")

	sparql := fmt.Sprintf(`
		PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

		SELECT ?interaction ?drug1 ?drug2 ?safetyLevel ?description ?severity ?evidence WHERE {
			?interaction a kb7:DrugInteraction .
			?interaction kb7:involves ?drug1 .
			?interaction kb7:involves ?drug2 .
			?interaction kb7:safetyLevel ?safetyLevel .
			OPTIONAL { ?interaction rdfs:comment ?description }
			OPTIONAL { ?interaction kb7:severity ?severity }
			OPTIONAL { ?interaction kb7:evidence ?evidence }
			FILTER(?drug1 != ?drug2)
			FILTER(?drug1 IN (%s))
			FILTER(?drug2 IN (%s))
		}
		ORDER BY ?safetyLevel ?severity
	`, medicationList, medicationList)

	result, err := c.SPARQLQuery(ctx, sparql)
	if err != nil {
		return nil, fmt.Errorf("failed to check drug interactions: %w", err)
	}

	return c.convertSPARQLToInteractions(result), nil
}

// GetRelationships gets semantic relationships for a concept
func (c *Client) GetRelationships(ctx context.Context, system, code, relType string) ([]SemanticRelationship, error) {
	prefix := c.getNamespacePrefix(system)

	var sparql string
	if relType == "all" {
		sparql = fmt.Sprintf(`
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX %s: <%s>

			SELECT ?predicate ?object ?label WHERE {
				%s:%s ?predicate ?object .
				OPTIONAL { ?object rdfs:label ?label }
				FILTER(isURI(?object))
			}
			ORDER BY ?predicate
			LIMIT 50
		`, prefix, c.getNamespaceURI(system), prefix, code)
	} else {
		sparql = fmt.Sprintf(`
			PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
			PREFIX %s: <%s>

			SELECT ?predicate ?object ?label WHERE {
				%s:%s ?predicate ?object .
				OPTIONAL { ?object rdfs:label ?label }
				FILTER(contains(str(?predicate), "%s"))
				FILTER(isURI(?object))
			}
			ORDER BY ?predicate
			LIMIT 25
		`, prefix, c.getNamespaceURI(system), prefix, code, relType)
	}

	result, err := c.SPARQLQuery(ctx, sparql)
	if err != nil {
		return nil, fmt.Errorf("failed to get relationships: %w", err)
	}

	return c.convertSPARQLToRelationships(result, system, code), nil
}

// GetConceptHierarchy retrieves concept hierarchy information
func (c *Client) GetConceptHierarchy(ctx context.Context, system, code string, depth int) (*ConceptHierarchy, error) {
	prefix := c.getNamespacePrefix(system)

	sparql := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
		PREFIX %s: <%s>

		SELECT ?relation ?concept ?label WHERE {
			{
				# Get parents
				%s:%s rdfs:subClassOf ?concept .
				BIND("parent" AS ?relation)
			} UNION {
				# Get children
				?concept rdfs:subClassOf %s:%s .
				BIND("child" AS ?relation)
			}
			OPTIONAL { ?concept rdfs:label ?label }
		}
		ORDER BY ?relation ?concept
	`, prefix, c.getNamespaceURI(system), prefix, code, prefix, code)

	result, err := c.SPARQLQuery(ctx, sparql)
	if err != nil {
		return nil, fmt.Errorf("failed to get concept hierarchy: %w", err)
	}

	return c.convertSPARQLToHierarchy(result, system, code), nil
}

// Helper methods

func (c *Client) getNamespacePrefix(system string) string {
	switch strings.ToLower(system) {
	case "snomed-ct", "sct":
		return "sct"
	case "loinc":
		return "loinc"
	case "icd-10":
		return "icd10"
	case "rxnorm":
		return "rxnorm"
	case "ucum":
		return "ucum"
	default:
		return "kb7"
	}
}

func (c *Client) getNamespaceURI(system string) string {
	switch strings.ToLower(system) {
	case "snomed-ct", "sct":
		return "http://snomed.info/id/"
	case "loinc":
		return "http://loinc.org/"
	case "icd-10":
		return "http://hl7.org/fhir/sid/icd-10/"
	case "rxnorm":
		return "http://www.nlm.nih.gov/research/umls/rxnorm/"
	case "ucum":
		return "http://unitsofmeasure.org/"
	default:
		return "http://cardiofit.ai/kb7/ontology#"
	}
}

func (c *Client) convertSPARQLToConcepts(result *SPARQLResult, system string) []postgres.Concept {
	var concepts []postgres.Concept

	for _, binding := range result.Results.Bindings {
		concept := postgres.Concept{
			System:      system,
			Status:      "active",
			LastUpdated: time.Now(),
		}

		if code, ok := binding["code"]; ok {
			concept.Code = code.Value
			concept.ID = fmt.Sprintf("%s-%s", system, code.Value)
		}

		if label, ok := binding["label"]; ok {
			concept.Display = label.Value
		}

		if definition, ok := binding["definition"]; ok {
			concept.Definition = definition.Value
		}

		concepts = append(concepts, concept)
	}

	return concepts
}

func (c *Client) convertSPARQLToInteractions(result *SPARQLResult) []DrugInteraction {
	var interactions []DrugInteraction

	for _, binding := range result.Results.Bindings {
		interaction := DrugInteraction{}

		if id, ok := binding["interaction"]; ok {
			interaction.ID = c.extractLocalName(id.Value)
		}

		if drug1, ok := binding["drug1"]; ok {
			interaction.Drug1 = c.extractLocalName(drug1.Value)
		}

		if drug2, ok := binding["drug2"]; ok {
			interaction.Drug2 = c.extractLocalName(drug2.Value)
		}

		if safetyLevel, ok := binding["safetyLevel"]; ok {
			interaction.SafetyLevel = safetyLevel.Value
		}

		if description, ok := binding["description"]; ok {
			interaction.Description = description.Value
		}

		if severity, ok := binding["severity"]; ok {
			interaction.Severity = severity.Value
		}

		if evidence, ok := binding["evidence"]; ok {
			interaction.Evidence = evidence.Value
		}

		interactions = append(interactions, interaction)
	}

	return interactions
}

func (c *Client) convertSPARQLToRelationships(result *SPARQLResult, system, code string) []SemanticRelationship {
	var relationships []SemanticRelationship

	for _, binding := range result.Results.Bindings {
		rel := SemanticRelationship{
			Subject: fmt.Sprintf("%s:%s", system, code),
			Type:    "semantic",
		}

		if predicate, ok := binding["predicate"]; ok {
			rel.Predicate = c.extractLocalName(predicate.Value)
		}

		if object, ok := binding["object"]; ok {
			rel.Object = c.extractLocalName(object.Value)
		}

		if label, ok := binding["label"]; ok {
			rel.Label = label.Value
		}

		relationships = append(relationships, rel)
	}

	return relationships
}

func (c *Client) convertSPARQLToHierarchy(result *SPARQLResult, system, code string) *ConceptHierarchy {
	hierarchy := &ConceptHierarchy{
		Concept:     fmt.Sprintf("%s:%s", system, code),
		Parents:     []string{},
		Children:    []string{},
		Ancestors:   []string{},
		Descendants: []string{},
	}

	for _, binding := range result.Results.Bindings {
		if relation, ok := binding["relation"]; ok {
			if concept, ok := binding["concept"]; ok {
				conceptCode := c.extractLocalName(concept.Value)
				if relation.Value == "parent" {
					hierarchy.Parents = append(hierarchy.Parents, conceptCode)
				} else if relation.Value == "child" {
					hierarchy.Children = append(hierarchy.Children, conceptCode)
				}
			}
		}
	}

	return hierarchy
}

func (c *Client) extractLocalName(uri string) string {
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return uri
}

// Ping tests the connection to GraphDB
func (c *Client) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/repositories", c.baseURL)
	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("GraphDB ping failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphDB ping failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetRepositoryInfo returns information about the repository
func (c *Client) GetRepositoryInfo(ctx context.Context) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/repositories/%s", c.baseURL, c.repoID)
	req, err := retryablehttp.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get repository info: status %d", resp.StatusCode)
	}

	var info map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return info, nil
}

// UpdateRepository updates RDF data in the repository
func (c *Client) UpdateRepository(ctx context.Context, sparqlUpdate string) error {
	url := fmt.Sprintf("%s/repositories/%s/statements", c.baseURL, c.repoID)
	reqBody := fmt.Sprintf("update=%s", sparqlUpdate)

	req, err := retryablehttp.NewRequestWithContext(ctx, "POST", url, strings.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("SPARQL update failed with status %d", resp.StatusCode)
	}

	return nil
}

// SetRepository sets the repository ID for queries
func (c *Client) SetRepository(repoID string) {
	c.repoID = repoID
}