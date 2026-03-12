package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchConnection manages Elasticsearch connections for formulary data
type ElasticsearchConnection struct {
	client *elasticsearch.Client
	config ElasticsearchConfig
}

// ElasticsearchConfig holds Elasticsearch connection configuration
// Updated to match config package structure
type ElasticsearchConfig struct {
	Addresses []string
	Username  string
	Password  string
	CloudID   string
	APIKey    string
	Enabled   bool
}

// IndexTemplates defines the index templates for formulary data
type IndexTemplates struct {
	FormularyDrugs    string
	CoverageRules     string
	PriorAuthRules    string
	FormularyUpdates  string
	DrugInteractions  string
}

// NewElasticsearchConnection creates a new Elasticsearch connection
func NewElasticsearchConnection(config ElasticsearchConfig) (*ElasticsearchConnection, error) {
	// Check if Elasticsearch is enabled
	if !config.Enabled {
		log.Println("Elasticsearch is disabled, creating stub connection")
		return &ElasticsearchConnection{
			client: nil,
			config: config,
		}, nil
	}

	cfg := elasticsearch.Config{
		Addresses: config.Addresses,
		Username:  config.Username,
		Password:  config.Password,
		CloudID:   config.CloudID,
		APIKey:    config.APIKey,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Test the connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get Elasticsearch info: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch returned error: %s", res.String())
	}

	log.Printf("Connected to Elasticsearch cluster")

	connection := &ElasticsearchConnection{
		client: client,
		config: config,
	}

	// Initialize indexes and mappings
	if err := connection.initializeIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize indexes: %w", err)
	}

	return connection, nil
}

// GetClient returns the Elasticsearch client
func (e *ElasticsearchConnection) GetClient() *elasticsearch.Client {
	return e.client
}

// initializeIndexes creates indexes and mappings for formulary data
func (e *ElasticsearchConnection) initializeIndexes(ctx context.Context) error {
	// Skip initialization if Elasticsearch is disabled
	if e.client == nil {
		log.Println("Elasticsearch is disabled, skipping index initialization")
		return nil
	}
	indexes := map[string]string{
		"formulary_drugs": `{
			"settings": {
				"number_of_shards": 3,
				"number_of_replicas": 1,
				"analysis": {
					"analyzer": {
						"drug_name_analyzer": {
							"tokenizer": "standard",
							"filter": ["lowercase", "asciifolding", "synonyms"]
						},
						"autocomplete_analyzer": {
							"tokenizer": "autocomplete_tokenizer",
							"filter": ["lowercase", "asciifolding"]
						}
					},
					"tokenizer": {
						"autocomplete_tokenizer": {
							"type": "edge_ngram",
							"min_gram": 2,
							"max_gram": 10,
							"token_chars": ["letter", "digit"]
						}
					},
					"filter": {
						"synonyms": {
							"type": "synonym",
							"synonyms": [
								"acetaminophen,paracetamol,tylenol",
								"ibuprofen,advil,motrin",
								"aspirin,acetylsalicylic acid"
							]
						}
					}
				}
			},
			"mappings": {
				"properties": {
					"drug_id": {"type": "keyword"},
					"drug_name": {
						"type": "text",
						"analyzer": "drug_name_analyzer",
						"fields": {
							"keyword": {"type": "keyword"},
							"autocomplete": {
								"type": "text",
								"analyzer": "autocomplete_analyzer"
							}
						}
					},
					"generic_name": {
						"type": "text",
						"analyzer": "drug_name_analyzer",
						"fields": {"keyword": {"type": "keyword"}}
					},
					"brand_names": {
						"type": "text",
						"analyzer": "drug_name_analyzer"
					},
					"rxnorm_code": {"type": "keyword"},
					"ndc_codes": {"type": "keyword"},
					"therapeutic_class": {"type": "keyword"},
					"drug_class": {"type": "keyword"},
					"route_of_administration": {"type": "keyword"},
					"dosage_forms": {"type": "keyword"},
					"strengths": {"type": "keyword"},
					"formulary_status": {"type": "keyword"},
					"tier": {"type": "integer"},
					"coverage_status": {"type": "keyword"},
					"prior_auth_required": {"type": "boolean"},
					"step_therapy_required": {"type": "boolean"},
					"quantity_limits": {
						"properties": {
							"limit_type": {"type": "keyword"},
							"limit_value": {"type": "integer"},
							"limit_period": {"type": "keyword"}
						}
					},
					"copay_info": {
						"properties": {
							"tier_1": {"type": "float"},
							"tier_2": {"type": "float"},
							"tier_3": {"type": "float"},
							"specialty": {"type": "float"}
						}
					},
					"alternatives": {
						"properties": {
							"drug_id": {"type": "keyword"},
							"drug_name": {"type": "text"},
							"tier": {"type": "integer"},
							"cost_difference": {"type": "float"}
						}
					},
					"contraindications": {"type": "text"},
					"age_restrictions": {
						"properties": {
							"min_age": {"type": "integer"},
							"max_age": {"type": "integer"}
						}
					},
					"pregnancy_category": {"type": "keyword"},
					"formulary_id": {"type": "keyword"},
					"effective_date": {"type": "date"},
					"expiration_date": {"type": "date"},
					"last_updated": {"type": "date"},
					"created_at": {"type": "date"},
					"metadata": {"type": "object", "enabled": false}
				}
			}
		}`,
		"coverage_rules": `{
			"settings": {
				"number_of_shards": 1,
				"number_of_replicas": 1
			},
			"mappings": {
				"properties": {
					"rule_id": {"type": "keyword"},
					"rule_name": {"type": "text"},
					"rule_type": {"type": "keyword"},
					"drug_criteria": {
						"properties": {
							"drug_ids": {"type": "keyword"},
							"therapeutic_classes": {"type": "keyword"},
							"generic_required": {"type": "boolean"}
						}
					},
					"patient_criteria": {
						"properties": {
							"age_min": {"type": "integer"},
							"age_max": {"type": "integer"},
							"gender": {"type": "keyword"},
							"diagnosis_codes": {"type": "keyword"},
							"prior_medications": {"type": "keyword"}
						}
					},
					"coverage_decision": {"type": "keyword"},
					"prior_auth_required": {"type": "boolean"},
					"step_therapy_drugs": {"type": "keyword"},
					"quantity_limits": {"type": "object"},
					"effective_date": {"type": "date"},
					"expiration_date": {"type": "date"},
					"priority": {"type": "integer"},
					"created_by": {"type": "keyword"},
					"approved_by": {"type": "keyword"},
					"approval_date": {"type": "date"},
					"status": {"type": "keyword"}
				}
			}
		}`,
		"prior_auth_rules": `{
			"settings": {
				"number_of_shards": 1,
				"number_of_replicas": 1
			},
			"mappings": {
				"properties": {
					"rule_id": {"type": "keyword"},
					"drug_id": {"type": "keyword"},
					"drug_name": {"type": "text"},
					"criteria_type": {"type": "keyword"},
					"clinical_criteria": {
						"properties": {
							"diagnosis_required": {"type": "keyword"},
							"failed_therapies": {"type": "keyword"},
							"contraindications": {"type": "keyword"},
							"lab_requirements": {"type": "text"}
						}
					},
					"documentation_required": {"type": "text"},
					"approval_duration": {"type": "integer"},
					"renewal_criteria": {"type": "text"},
					"override_codes": {"type": "keyword"},
					"emergency_override": {"type": "boolean"},
					"effective_date": {"type": "date"},
					"review_date": {"type": "date"},
					"status": {"type": "keyword"}
				}
			}
		}`,
		"formulary_updates": `{
			"settings": {
				"number_of_shards": 1,
				"number_of_replicas": 1
			},
			"mappings": {
				"properties": {
					"update_id": {"type": "keyword"},
					"formulary_id": {"type": "keyword"},
					"update_type": {"type": "keyword"},
					"drug_id": {"type": "keyword"},
					"drug_name": {"type": "text"},
					"change_description": {"type": "text"},
					"old_value": {"type": "object", "enabled": false},
					"new_value": {"type": "object", "enabled": false},
					"effective_date": {"type": "date"},
					"notification_date": {"type": "date"},
					"impact_assessment": {"type": "text"},
					"affected_members": {"type": "integer"},
					"update_source": {"type": "keyword"},
					"updated_by": {"type": "keyword"},
					"approval_status": {"type": "keyword"},
					"created_at": {"type": "date"}
				}
			}
		}`,
		"drug_interactions": `{
			"settings": {
				"number_of_shards": 2,
				"number_of_replicas": 1
			},
			"mappings": {
				"properties": {
					"interaction_id": {"type": "keyword"},
					"drug_a_id": {"type": "keyword"},
					"drug_a_name": {"type": "text"},
					"drug_b_id": {"type": "keyword"},
					"drug_b_name": {"type": "text"},
					"interaction_type": {"type": "keyword"},
					"severity": {"type": "keyword"},
					"mechanism": {"type": "text"},
					"clinical_effect": {"type": "text"},
					"management": {"type": "text"},
					"evidence_level": {"type": "keyword"},
					"source": {"type": "keyword"},
					"last_reviewed": {"type": "date"},
					"status": {"type": "keyword"}
				}
			}
		}`,
	}

	for indexName, mappingJSON := range indexes {
		if err := e.createIndexIfNotExists(ctx, indexName, mappingJSON); err != nil {
			log.Printf("Warning: failed to create index %s: %v", indexName, err)
		}
	}

	return nil
}

// createIndexIfNotExists creates an index if it doesn't exist
func (e *ElasticsearchConnection) createIndexIfNotExists(ctx context.Context, indexName, mappingJSON string) error {
	// Check if index exists
	req := esapi.IndicesExistsRequest{
		Index: []string{indexName},
	}
	
	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		log.Printf("Index %s already exists", indexName)
		return nil
	}

	// Create index
	req2 := esapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(mappingJSON),
	}

	res2, err := req2.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res2.Body.Close()

	if res2.IsError() {
		return fmt.Errorf("failed to create index %s: %s", indexName, res2.String())
	}

	log.Printf("Created index: %s", indexName)
	return nil
}

// Index documents into Elasticsearch
func (e *ElasticsearchConnection) IndexDocument(ctx context.Context, indexName, documentID string, document interface{}) error {
	data, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: documentID,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index document: %s", res.String())
	}

	return nil
}

// BulkIndex indexes multiple documents efficiently
func (e *ElasticsearchConnection) BulkIndex(ctx context.Context, indexName string, documents []map[string]interface{}) error {
	if len(documents) == 0 {
		return nil
	}

	var buf bytes.Buffer
	
	for _, doc := range documents {
		// Index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": indexName,
			},
		}
		
		// Add document ID if present
		if id, exists := doc["id"]; exists {
			action["index"].(map[string]interface{})["_id"] = id
		}

		actionData, _ := json.Marshal(action)
		buf.Write(actionData)
		buf.WriteByte('\n')

		// Document data
		docData, _ := json.Marshal(doc)
		buf.Write(docData)
		buf.WriteByte('\n')
	}

	req := esapi.BulkRequest{
		Body:    bytes.NewReader(buf.Bytes()),
		Index:   indexName,
		Refresh: "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("failed to bulk index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk index failed: %s", res.String())
	}

	return nil
}

// Search performs a search query
func (e *ElasticsearchConnection) Search(ctx context.Context, indexName string, query map[string]interface{}) (map[string]interface{}, error) {
	queryData, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{indexName},
		Body:  bytes.NewReader(queryData),
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search failed: %s", res.String())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetDocument retrieves a document by ID
func (e *ElasticsearchConnection) GetDocument(ctx context.Context, indexName, documentID string) (map[string]interface{}, error) {
	req := esapi.GetRequest{
		Index:      indexName,
		DocumentID: documentID,
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return nil, fmt.Errorf("get request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("get failed: %s", res.String())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// DeleteDocument deletes a document by ID
func (e *ElasticsearchConnection) DeleteDocument(ctx context.Context, indexName, documentID string) error {
	req := esapi.DeleteRequest{
		Index:      indexName,
		DocumentID: documentID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, e.client)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("delete failed: %s", res.String())
	}

	return nil
}

// Health check for Elasticsearch
func (e *ElasticsearchConnection) HealthCheck(ctx context.Context) error {
	res, err := e.client.Cluster.Health()
	if err != nil {
		return fmt.Errorf("Elasticsearch health check failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch cluster unhealthy: %s", res.String())
	}

	return nil
}

// GetClusterInfo returns cluster information
func (e *ElasticsearchConnection) GetClusterInfo(ctx context.Context) (map[string]interface{}, error) {
	res, err := e.client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster info: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return info, nil
}

// Close closes the Elasticsearch connection and cleans up resources
func (e *ElasticsearchConnection) Close() error {
	// Elasticsearch Go client doesn't require explicit close
	// This method exists to satisfy interface requirements
	log.Println("Closing Elasticsearch connection")
	return nil
}

// InitializeIndices is a public wrapper for initializeIndexes
func (e *ElasticsearchConnection) InitializeIndices(ctx context.Context) error {
	return e.initializeIndexes(ctx)
}