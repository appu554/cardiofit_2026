package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/sirupsen/logrus"
)

// Client provides Elasticsearch integration for the KB7 terminology service
type Client struct {
	es     *elasticsearch.Client
	logger *logrus.Logger
	index  string
}

// SearchRequest represents a clinical search request to Elasticsearch
type SearchRequest struct {
	Query                string                 `json:"query"`
	Systems             []string               `json:"systems,omitempty"`
	Mode                string                 `json:"mode,omitempty"`
	Filters             map[string]interface{} `json:"filters,omitempty"`
	Preferences         map[string]interface{} `json:"preferences,omitempty"`
	UserContext         map[string]interface{} `json:"user_context,omitempty"`
	IncludeHighlights   bool                   `json:"include_highlights,omitempty"`
	IncludeFacets       bool                   `json:"include_facets,omitempty"`
	MaxResults          int                    `json:"max_results,omitempty"`
	Offset              int                    `json:"offset,omitempty"`
}

// SearchResult represents a clinical search result from Elasticsearch
type SearchResult struct {
	Code           string                 `json:"code"`
	System         string                 `json:"system"`
	Display        string                 `json:"display"`
	Definition     string                 `json:"definition,omitempty"`
	Synonyms       []string               `json:"synonyms,omitempty"`
	Domain         string                 `json:"domain,omitempty"`
	Status         string                 `json:"status"`
	Score          float64                `json:"score"`
	Highlights     []string               `json:"highlights,omitempty"`
	Properties     map[string]interface{} `json:"properties,omitempty"`
	Relationships  []Relationship         `json:"relationships,omitempty"`
}

// SearchResponse represents the response from Elasticsearch search
type SearchResponse struct {
	RequestID      string                            `json:"request_id"`
	TotalResults   int64                             `json:"total_results"`
	Results        []SearchResult                    `json:"results"`
	Facets         map[string]map[string]interface{} `json:"facets,omitempty"`
	Suggestions    map[string][]string               `json:"suggestions,omitempty"`
	QueryTimeMs    int64                             `json:"query_time_ms"`
	TotalTimeMs    int64                             `json:"total_time_ms"`
}

// Relationship represents a concept relationship
type Relationship struct {
	Type          string `json:"type"`
	TargetCode    string `json:"target_code"`
	TargetSystem  string `json:"target_system"`
	TargetDisplay string `json:"target_display"`
}

// AutocompleteRequest represents an autocomplete request
type AutocompleteRequest struct {
	Query       string                 `json:"query"`
	Systems     []string               `json:"systems,omitempty"`
	UserContext map[string]interface{} `json:"user_context,omitempty"`
	MaxResults  int                    `json:"max_results,omitempty"`
}

// AutocompleteResponse represents autocomplete suggestions
type AutocompleteResponse struct {
	RequestID   string               `json:"request_id"`
	Suggestions []AutocompleteSuggestion `json:"suggestions"`
	QueryTimeMs int64                `json:"query_time_ms"`
}

// AutocompleteSuggestion represents a single suggestion
type AutocompleteSuggestion struct {
	Text        string  `json:"text"`
	DisplayText string  `json:"display_text"`
	Code        string  `json:"code,omitempty"`
	System      string  `json:"system,omitempty"`
	Type        string  `json:"type"`
	Score       float64 `json:"score"`
}

// NewClient creates a new Elasticsearch client for KB7 terminology service
func NewClient(config elasticsearch.Config, index string, logger *logrus.Logger) (*Client, error) {
	es, err := elasticsearch.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	client := &Client{
		es:     es,
		logger: logger,
		index:  index,
	}

	return client, nil
}

// Ping checks if Elasticsearch is reachable
func (c *Client) Ping() error {
	res, err := c.es.Ping()
	if err != nil {
		return fmt.Errorf("elasticsearch ping failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch ping returned error: %s", res.Status())
	}

	return nil
}

// Search performs clinical terminology search
func (c *Client) Search(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	start := time.Now()

	// Build Elasticsearch query based on search mode
	esQuery, err := c.buildSearchQuery(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	// Execute search
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(esQuery); err != nil {
		return nil, fmt.Errorf("failed to encode search query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{c.index},
		Body:  &buf,
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search returned error: %s", res.Status())
	}

	// Parse response
	var esResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Convert to our response format
	response, err := c.parseSearchResponse(esResponse, start)
	if err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return response, nil
}

// GetAutocompleteSuggestions provides real-time autocomplete suggestions
func (c *Client) GetAutocompleteSuggestions(ctx context.Context, request *AutocompleteRequest) (*AutocompleteResponse, error) {
	start := time.Now()

	// Build autocomplete query
	esQuery := c.buildAutocompleteQuery(request)

	// Execute search
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(esQuery); err != nil {
		return nil, fmt.Errorf("failed to encode autocomplete query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{c.index},
		Body:  &buf,
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, fmt.Errorf("autocomplete request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("autocomplete returned error: %s", res.Status())
	}

	// Parse response
	var esResponse map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&esResponse); err != nil {
		return nil, fmt.Errorf("failed to decode autocomplete response: %w", err)
	}

	// Convert to our response format
	response, err := c.parseAutocompleteResponse(esResponse, start)
	if err != nil {
		return nil, fmt.Errorf("failed to parse autocomplete response: %w", err)
	}

	return response, nil
}

// buildSearchQuery constructs Elasticsearch query based on search parameters
func (c *Client) buildSearchQuery(request *SearchRequest) (map[string]interface{}, error) {
	query := make(map[string]interface{})

	// Set size and from for pagination
	maxResults := request.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}
	query["size"] = maxResults
	query["from"] = request.Offset

	// Build the main query based on mode
	var mainQuery map[string]interface{}

	switch request.Mode {
	case "exact":
		mainQuery = map[string]interface{}{
			"term": map[string]interface{}{
				"display.exact": request.Query,
			},
		}
	case "fuzzy":
		mainQuery = map[string]interface{}{
			"fuzzy": map[string]interface{}{
				"display": map[string]interface{}{
					"value":     request.Query,
					"fuzziness": "AUTO",
				},
			},
		}
	case "semantic":
		mainQuery = map[string]interface{}{
			"more_like_this": map[string]interface{}{
				"fields":               []string{"display", "definition", "synonyms"},
				"like":                 request.Query,
				"min_term_freq":        1,
				"min_doc_freq":         1,
				"minimum_should_match": "30%",
			},
		}
	case "hybrid":
		mainQuery = map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"display": map[string]interface{}{
								"query": request.Query,
								"boost": 3.0,
							},
						},
					},
					{
						"match": map[string]interface{}{
							"synonyms": map[string]interface{}{
								"query": request.Query,
								"boost": 2.0,
							},
						},
					},
					{
						"match": map[string]interface{}{
							"definition": request.Query,
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	default: // standard
		mainQuery = map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  request.Query,
				"fields": []string{"display^3", "synonyms^2", "definition"},
				"type":   "best_fields",
			},
		}
	}

	// Add filters if specified
	boolQuery := map[string]interface{}{
		"must": mainQuery,
	}

	var filters []map[string]interface{}

	// System filters
	if len(request.Systems) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"system": request.Systems,
			},
		})
	}

	// Status filters
	if request.Filters != nil {
		if status, exists := request.Filters["status"]; exists {
			if statusList, ok := status.([]string); ok && len(statusList) > 0 {
				filters = append(filters, map[string]interface{}{
					"terms": map[string]interface{}{
						"status": statusList,
					},
				})
			}
		}
	}

	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	query["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	// Add highlighting if requested
	if request.IncludeHighlights {
		query["highlight"] = map[string]interface{}{
			"fields": map[string]interface{}{
				"display":    map[string]interface{}{},
				"synonyms":   map[string]interface{}{},
				"definition": map[string]interface{}{},
			},
		}
	}

	// Add facets if requested
	if request.IncludeFacets {
		query["aggs"] = map[string]interface{}{
			"systems": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "system",
					"size":  20,
				},
			},
			"status": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "status",
					"size":  10,
				},
			},
		}
	}

	return query, nil
}

// buildAutocompleteQuery constructs Elasticsearch query for autocomplete
func (c *Client) buildAutocompleteQuery(request *AutocompleteRequest) map[string]interface{} {
	maxResults := request.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}

	query := map[string]interface{}{
		"size": maxResults,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []map[string]interface{}{
					{
						"prefix": map[string]interface{}{
							"display": request.Query,
						},
					},
					{
						"prefix": map[string]interface{}{
							"synonyms": request.Query,
						},
					},
				},
				"minimum_should_match": 1,
			},
		},
		"sort": []map[string]interface{}{
			{"_score": map[string]string{"order": "desc"}},
		},
	}

	// Add system filters if specified
	if len(request.Systems) > 0 {
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = map[string]interface{}{
			"terms": map[string]interface{}{
				"system": request.Systems,
			},
		}
	}

	return query
}

// parseSearchResponse converts Elasticsearch response to our format
func (c *Client) parseSearchResponse(esResponse map[string]interface{}, startTime time.Time) (*SearchResponse, error) {
	hits, ok := esResponse["hits"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid search response format")
	}

	total, ok := hits["total"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid total format in search response")
	}

	totalValue, ok := total["value"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid total value in search response")
	}

	documents, ok := hits["hits"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid hits format in search response")
	}

	results := make([]SearchResult, 0, len(documents))
	for _, doc := range documents {
		hit := doc.(map[string]interface{})
		source := hit["_source"].(map[string]interface{})

		result := SearchResult{
			Code:       getString(source, "code"),
			System:     getString(source, "system"),
			Display:    getString(source, "display"),
			Definition: getString(source, "definition"),
			Domain:     getString(source, "domain"),
			Status:     getString(source, "status"),
			Score:      getFloat64(hit, "_score"),
		}

		// Parse synonyms
		if synonyms, exists := source["synonyms"]; exists {
			if synList, ok := synonyms.([]interface{}); ok {
				result.Synonyms = make([]string, len(synList))
				for i, syn := range synList {
					result.Synonyms[i] = syn.(string)
				}
			}
		}

		// Parse highlights
		if highlight, exists := hit["highlight"]; exists {
			hlMap := highlight.(map[string]interface{})
			for _, hlList := range hlMap {
				if highlights, ok := hlList.([]interface{}); ok {
					for _, hl := range highlights {
						result.Highlights = append(result.Highlights, hl.(string))
					}
				}
			}
		}

		results = append(results, result)
	}

	queryTime := time.Since(startTime)

	response := &SearchResponse{
		TotalResults: int64(totalValue),
		Results:      results,
		QueryTimeMs:  queryTime.Milliseconds(),
		TotalTimeMs:  queryTime.Milliseconds(),
	}

	// Parse aggregations if present
	if aggs, exists := esResponse["aggregations"]; exists {
		response.Facets = make(map[string]map[string]interface{})
		aggMap := aggs.(map[string]interface{})

		for aggName, aggResult := range aggMap {
			if buckets, ok := aggResult.(map[string]interface{})["buckets"]; ok {
				response.Facets[aggName] = map[string]interface{}{
					"buckets": buckets,
				}
			}
		}
	}

	return response, nil
}

// parseAutocompleteResponse converts Elasticsearch response to autocomplete format
func (c *Client) parseAutocompleteResponse(esResponse map[string]interface{}, startTime time.Time) (*AutocompleteResponse, error) {
	hits, ok := esResponse["hits"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid autocomplete response format")
	}

	documents, ok := hits["hits"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid hits format in autocomplete response")
	}

	suggestions := make([]AutocompleteSuggestion, 0, len(documents))
	for _, doc := range documents {
		hit := doc.(map[string]interface{})
		source := hit["_source"].(map[string]interface{})

		suggestion := AutocompleteSuggestion{
			Text:        getString(source, "display"),
			DisplayText: getString(source, "display"),
			Code:        getString(source, "code"),
			System:      getString(source, "system"),
			Type:        "prefix_match",
			Score:       getFloat64(hit, "_score"),
		}

		suggestions = append(suggestions, suggestion)
	}

	queryTime := time.Since(startTime)

	response := &AutocompleteResponse{
		Suggestions: suggestions,
		QueryTimeMs: queryTime.Milliseconds(),
	}

	return response, nil
}

// Helper functions for type conversion
func getString(source map[string]interface{}, key string) string {
	if val, exists := source[key]; exists {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64(source map[string]interface{}, key string) float64 {
	if val, exists := source[key]; exists {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0.0
}