package elasticsearch

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SearchService provides clinical terminology search functionality
type SearchService struct {
	client    *Client
	indexName string
}

// NewSearchService creates a new search service
func NewSearchService(client *Client, indexName string) *SearchService {
	return &SearchService{
		client:    client,
		indexName: indexName,
	}
}

// Search performs a clinical terminology search
func (s *SearchService) Search(ctx context.Context, req *SearchRequest) (*SearchResults, error) {
	if req.Query == "" {
		return s.searchAll(ctx, req)
	}

	query, err := s.buildSearchQuery(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	startTime := time.Now()
	response, err := s.client.Search(ctx, s.indexName, query)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	took := int(time.Since(startTime).Milliseconds())

	results, err := s.parseSearchResponse(response, req.SearchType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	results.Took = took
	results.TimedOut = response.TimedOut

	return results, nil
}

// buildSearchQuery constructs the Elasticsearch query based on search request
func (s *SearchService) buildSearchQuery(req *SearchRequest) (map[string]interface{}, error) {
	query := map[string]interface{}{
		"size": s.getSize(req.Size),
		"from": req.From,
	}

	// Build the main query
	mainQuery := s.buildMainQuery(req)

	// Add filters
	if len(req.Systems) > 0 || len(req.Domains) > 0 || len(req.SemanticTags) > 0 || req.Status != "" || len(req.Filters) > 0 {
		query["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"must":   []interface{}{mainQuery},
				"filter": s.buildFilters(req),
			},
		}
	} else {
		query["query"] = mainQuery
	}

	// Add sorting
	if req.SortBy != "" {
		query["sort"] = s.buildSort(req.SortBy, req.SortOrder)
	} else {
		// Default sorting by relevance score, then by usage frequency
		query["sort"] = []interface{}{
			"_score",
			map[string]interface{}{
				"usage_frequency": map[string]interface{}{
					"order": "desc",
				},
			},
		}
	}

	// Add highlighting
	query["highlight"] = map[string]interface{}{
		"fields": map[string]interface{}{
			"term":           map[string]interface{}{},
			"preferred_term": map[string]interface{}{},
			"synonyms":       map[string]interface{}{},
			"definition":     map[string]interface{}{},
		},
		"pre_tags":  []string{"<em>"},
		"post_tags": []string{"</em>"},
	}

	// Add aggregations for faceted search
	query["aggs"] = map[string]interface{}{
		"terminology_systems": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "terminology_system",
				"size":  20,
			},
		},
		"clinical_domains": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "clinical_domain",
				"size":  20,
			},
		},
		"semantic_tags": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "semantic_tags",
				"size":  20,
			},
		},
	}

	return query, nil
}

// buildMainQuery creates the main search query based on search type
func (s *SearchService) buildMainQuery(req *SearchRequest) map[string]interface{} {
	switch req.SearchType {
	case SearchTypeExact:
		return s.buildExactQuery(req.Query)
	case SearchTypeAutocomplete:
		return s.buildAutocompleteQuery(req.Query)
	case SearchTypePhonetic:
		return s.buildPhoneticQuery(req.Query)
	case SearchTypeFuzzy:
		return s.buildFuzzyQuery(req.Query)
	case SearchTypeWildcard:
		return s.buildWildcardQuery(req.Query)
	default:
		return s.buildStandardQuery(req.Query, req.ExactMatch)
	}
}

// buildStandardQuery creates a standard multi-field search query
func (s *SearchService) buildStandardQuery(query string, exactMatch bool) map[string]interface{} {
	if exactMatch {
		return map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{
						"term": map[string]interface{}{
							"term.exact": map[string]interface{}{
								"value": strings.ToLower(query),
								"boost": 10.0,
							},
						},
					},
					map[string]interface{}{
						"term": map[string]interface{}{
							"preferred_term.exact": map[string]interface{}{
								"value": strings.ToLower(query),
								"boost": 8.0,
							},
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				// Exact matches get highest boost
				map[string]interface{}{
					"term": map[string]interface{}{
						"term.exact": map[string]interface{}{
							"value": strings.ToLower(query),
							"boost": 10.0,
						},
					},
				},
				map[string]interface{}{
					"term": map[string]interface{}{
						"preferred_term.exact": map[string]interface{}{
							"value": strings.ToLower(query),
							"boost": 8.0,
						},
					},
				},
				// Standard analyzed matches with medical synonyms
				map[string]interface{}{
					"match": map[string]interface{}{
						"term": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_search",
							"boost":    5.0,
						},
					},
				},
				map[string]interface{}{
					"match": map[string]interface{}{
						"preferred_term": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_search",
							"boost":    4.0,
						},
					},
				},
				map[string]interface{}{
					"match": map[string]interface{}{
						"synonyms": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_search",
							"boost":    3.0,
						},
					},
				},
				// Definition search with lower boost
				map[string]interface{}{
					"match": map[string]interface{}{
						"definition": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_search",
							"boost":    1.0,
						},
					},
				},
				// Multi-match across all text fields
				map[string]interface{}{
					"multi_match": map[string]interface{}{
						"query":    query,
						"fields":   []string{"term^5", "preferred_term^4", "synonyms^3", "definition^1"},
						"analyzer": "medical_search",
						"type":     "best_fields",
						"boost":    2.0,
					},
				},
			},
		},
	}
}

// buildExactQuery creates an exact match query
func (s *SearchService) buildExactQuery(query string) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				map[string]interface{}{
					"term": map[string]interface{}{
						"term.exact": map[string]interface{}{
							"value": strings.ToLower(query),
							"boost": 10.0,
						},
					},
				},
				map[string]interface{}{
					"term": map[string]interface{}{
						"preferred_term.exact": map[string]interface{}{
							"value": strings.ToLower(query),
							"boost": 8.0,
						},
					},
				},
			},
			"minimum_should_match": 1,
		},
	}
}

// buildAutocompleteQuery creates an autocomplete query
func (s *SearchService) buildAutocompleteQuery(query string) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				map[string]interface{}{
					"match": map[string]interface{}{
						"term.autocomplete": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_autocomplete",
							"boost":    5.0,
						},
					},
				},
				map[string]interface{}{
					"match": map[string]interface{}{
						"preferred_term.autocomplete": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_autocomplete",
							"boost":    4.0,
						},
					},
				},
				// Also include prefix matches
				map[string]interface{}{
					"prefix": map[string]interface{}{
						"term.exact": map[string]interface{}{
							"value": strings.ToLower(query),
							"boost": 3.0,
						},
					},
				},
			},
		},
	}
}

// buildPhoneticQuery creates a phonetic search query
func (s *SearchService) buildPhoneticQuery(query string) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				map[string]interface{}{
					"match": map[string]interface{}{
						"term.phonetic": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_phonetic",
							"boost":    3.0,
						},
					},
				},
				map[string]interface{}{
					"match": map[string]interface{}{
						"preferred_term.phonetic": map[string]interface{}{
							"query":    query,
							"analyzer": "medical_phonetic",
							"boost":    2.0,
						},
					},
				},
			},
		},
	}
}

// buildFuzzyQuery creates a fuzzy search query
func (s *SearchService) buildFuzzyQuery(query string) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				map[string]interface{}{
					"fuzzy": map[string]interface{}{
						"term": map[string]interface{}{
							"value":     query,
							"fuzziness": "AUTO",
							"boost":     3.0,
						},
					},
				},
				map[string]interface{}{
					"fuzzy": map[string]interface{}{
						"preferred_term": map[string]interface{}{
							"value":     query,
							"fuzziness": "AUTO",
							"boost":     2.0,
						},
					},
				},
			},
		},
	}
}

// buildWildcardQuery creates a wildcard search query
func (s *SearchService) buildWildcardQuery(query string) map[string]interface{} {
	// Ensure query has wildcards
	if !strings.Contains(query, "*") && !strings.Contains(query, "?") {
		query = "*" + query + "*"
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": []interface{}{
				map[string]interface{}{
					"wildcard": map[string]interface{}{
						"term.exact": map[string]interface{}{
							"value": strings.ToLower(query),
							"boost": 3.0,
						},
					},
				},
				map[string]interface{}{
					"wildcard": map[string]interface{}{
						"preferred_term.exact": map[string]interface{}{
							"value": strings.ToLower(query),
							"boost": 2.0,
						},
					},
				},
			},
		},
	}
}

// buildFilters creates filter clauses
func (s *SearchService) buildFilters(req *SearchRequest) []interface{} {
	var filters []interface{}

	// Terminology system filter
	if len(req.Systems) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"terminology_system": req.Systems,
			},
		})
	}

	// Clinical domain filter
	if len(req.Domains) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"clinical_domain": req.Domains,
			},
		})
	}

	// Semantic tags filter
	if len(req.SemanticTags) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{
				"semantic_tags": req.SemanticTags,
			},
		})
	}

	// Status filter
	if req.Status != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"status": req.Status,
			},
		})
	} else if !req.IncludeInactive {
		// Default to active terms only
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"status": "active",
			},
		})
	}

	// Custom filters
	for field, value := range req.Filters {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				field: value,
			},
		})
	}

	return filters
}

// buildSort creates sort clauses
func (s *SearchService) buildSort(sortBy, sortOrder string) []interface{} {
	if sortOrder == "" {
		sortOrder = "asc"
	}

	return []interface{}{
		map[string]interface{}{
			sortBy: map[string]interface{}{
				"order": sortOrder,
			},
		},
	}
}

// parseSearchResponse converts Elasticsearch response to SearchResults
func (s *SearchService) parseSearchResponse(response *SearchResponse, searchType SearchType) (*SearchResults, error) {
	results := &SearchResults{
		Total:   response.Hits.Total.Value,
		Results: make([]*SearchResult, 0, len(response.Hits.Hits)),
	}

	for _, hit := range response.Hits.Hits {
		term, err := s.parseHitSource(hit.Source)
		if err != nil {
			continue // Skip malformed documents
		}

		result := &SearchResult{
			Term:        term,
			Score:       hit.Score,
			MatchReason: s.getMatchReason(searchType),
		}

		results.Results = append(results.Results, result)
	}

	return results, nil
}

// parseHitSource converts hit source to ClinicalTerm
func (s *SearchService) parseHitSource(source map[string]interface{}) (*ClinicalTerm, error) {
	term := &ClinicalTerm{}

	if termID, ok := source["term_id"].(string); ok {
		term.TermID = termID
	}

	if conceptID, ok := source["concept_id"].(string); ok {
		term.ConceptID = conceptID
	}

	if termText, ok := source["term"].(string); ok {
		term.Term = termText
	}

	if preferredTerm, ok := source["preferred_term"].(string); ok {
		term.PreferredTerm = preferredTerm
	}

	if definition, ok := source["definition"].(string); ok {
		term.Definition = definition
	}

	if system, ok := source["terminology_system"].(string); ok {
		term.TerminologySystem = system
	}

	if version, ok := source["terminology_version"].(string); ok {
		term.TerminologyVersion = version
	}

	if status, ok := source["status"].(string); ok {
		term.Status = status
	}

	if domain, ok := source["clinical_domain"].(string); ok {
		term.ClinicalDomain = domain
	}

	// Handle arrays
	if synonyms, ok := source["synonyms"].([]interface{}); ok {
		term.Synonyms = make([]string, len(synonyms))
		for i, syn := range synonyms {
			if s, ok := syn.(string); ok {
				term.Synonyms[i] = s
			}
		}
	}

	if tags, ok := source["semantic_tags"].([]interface{}); ok {
		term.SemanticTags = make([]string, len(tags))
		for i, tag := range tags {
			if t, ok := tag.(string); ok {
				term.SemanticTags[i] = t
			}
		}
	}

	// Handle numeric fields
	if score, ok := source["complexity_score"].(float64); ok {
		term.ComplexityScore = float32(score)
	}

	if freq, ok := source["usage_frequency"].(float64); ok {
		term.UsageFrequency = int64(freq)
	}

	return term, nil
}

// getMatchReason returns a description of why the result matched
func (s *SearchService) getMatchReason(searchType SearchType) string {
	switch searchType {
	case SearchTypeExact:
		return "Exact match"
	case SearchTypeAutocomplete:
		return "Autocomplete match"
	case SearchTypePhonetic:
		return "Phonetic match"
	case SearchTypeFuzzy:
		return "Fuzzy match"
	case SearchTypeWildcard:
		return "Wildcard match"
	default:
		return "Standard match"
	}
}

// searchAll returns all terms when no query is specified
func (s *SearchService) searchAll(ctx context.Context, req *SearchRequest) (*SearchResults, error) {
	query := map[string]interface{}{
		"size": s.getSize(req.Size),
		"from": req.From,
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
	}

	// Add filters even for match_all queries
	if len(req.Systems) > 0 || len(req.Domains) > 0 || len(req.SemanticTags) > 0 || req.Status != "" || len(req.Filters) > 0 {
		query["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{
					map[string]interface{}{
						"match_all": map[string]interface{}{},
					},
				},
				"filter": s.buildFilters(req),
			},
		}
	}

	// Add sorting
	if req.SortBy != "" {
		query["sort"] = s.buildSort(req.SortBy, req.SortOrder)
	}

	startTime := time.Now()
	response, err := s.client.Search(ctx, s.indexName, query)
	if err != nil {
		return nil, fmt.Errorf("search all failed: %w", err)
	}
	took := int(time.Since(startTime).Milliseconds())

	results, err := s.parseSearchResponse(response, SearchTypeStandard)
	if err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	results.Took = took
	results.TimedOut = response.TimedOut

	return results, nil
}

// getSize returns the search size with a reasonable default and maximum
func (s *SearchService) getSize(size int) int {
	if size <= 0 {
		return 20 // Default size
	}
	if size > 1000 {
		return 1000 // Maximum size
	}
	return size
}