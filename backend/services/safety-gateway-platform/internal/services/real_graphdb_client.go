package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// GraphDBClient interface defines the contract for GraphDB clients
type GraphDBClient interface {
	GetPatientContext(ctx context.Context, patientID string) (map[string]interface{}, error)
	GetClinicalRelationships(ctx context.Context, patientID string) (map[string]interface{}, error)
	ExecuteSPARQLQuery(ctx context.Context, query string) (map[string]interface{}, error)
	GetRelatedEntities(ctx context.Context, entityID string, entityType string) ([]types.GraphEntity, error)
	Close() error
}

// RealGraphDBClient implements a real GraphDB client that connects to GraphDB
type RealGraphDBClient struct {
	logger   *logger.Logger
	baseURL  string
	client   *http.Client
	username string
	password string
}

// NewRealGraphDBClient creates a new real GraphDB client
func NewRealGraphDBClient(cfg *config.Config, logger *logger.Logger) *RealGraphDBClient {
	return &RealGraphDBClient{
		logger:  logger,
		baseURL: cfg.ExternalServices.GraphDBService.Endpoint,
		client:  &http.Client{Timeout: 30 * time.Second},
		// Add credentials if needed
		username: "", // Add from config if needed
		password: "", // Add from config if needed
	}
}

// NewGraphDBClient creates a new GraphDB client based on configuration
func NewGraphDBClient(cfg *config.Config, logger *logger.Logger) GraphDBClient {
	return NewRealGraphDBClient(cfg, logger)
}

// GetPatientContext fetches patient context from GraphDB
func (g *RealGraphDBClient) GetPatientContext(ctx context.Context, patientID string) (map[string]interface{}, error) {
	// SPARQL query to get patient context
	query := fmt.Sprintf(`
		PREFIX fhir: <http://hl7.org/fhir/>
		PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		
		SELECT ?property ?value ?type WHERE {
			<http://example.org/patient/%s> ?property ?value .
			OPTIONAL { ?value rdf:type ?type }
		}
		LIMIT 100
	`, patientID)

	result, err := g.ExecuteSPARQLQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute patient context query: %w", err)
	}

	g.logger.Debug("GraphDB patient context fetched",
		zap.String("patient_id", patientID))

	return result, nil
}

// GetClinicalRelationships fetches clinical relationships for a patient from GraphDB
func (g *RealGraphDBClient) GetClinicalRelationships(ctx context.Context, patientID string) (map[string]interface{}, error) {
	// SPARQL query to get clinical relationships
	query := fmt.Sprintf(`
		PREFIX fhir: <http://hl7.org/fhir/>
		PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

		SELECT ?subject ?predicate ?object WHERE {
			{
				<http://example.org/patient/%s> ?predicate ?object .
				BIND(<http://example.org/patient/%s> AS ?subject)
			}
			UNION
			{
				?subject ?predicate <http://example.org/patient/%s> .
				BIND(<http://example.org/patient/%s> AS ?object)
			}
		}
		LIMIT 200
	`, patientID, patientID, patientID, patientID)

	result, err := g.ExecuteSPARQLQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute clinical relationships query: %w", err)
	}

	g.logger.Debug("GraphDB clinical relationships fetched",
		zap.String("patient_id", patientID))

	return result, nil
}

// ExecuteSPARQLQuery executes a SPARQL query against GraphDB
func (g *RealGraphDBClient) ExecuteSPARQLQuery(ctx context.Context, query string) (map[string]interface{}, error) {
	// Prepare the SPARQL query request
	queryData := map[string]string{
		"query": query,
	}
	
	jsonData, err := json.Marshal(queryData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/repositories/clinical-synthesis-hub", g.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/sparql-query")
	req.Header.Set("Accept", "application/sparql-results+json")

	// Add authentication if needed
	if g.username != "" && g.password != "" {
		req.SetBasicAuth(g.username, g.password)
	}

	// Execute request
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphDB API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	g.logger.Debug("SPARQL query executed successfully",
		zap.String("query_preview", query[:min(100, len(query))]),
		zap.Int("status_code", resp.StatusCode))

	return result, nil
}

// GetRelatedEntities fetches entities related to a given entity
func (g *RealGraphDBClient) GetRelatedEntities(ctx context.Context, entityID string, entityType string) ([]types.GraphEntity, error) {
	// SPARQL query to get related entities
	query := fmt.Sprintf(`
		PREFIX fhir: <http://hl7.org/fhir/>
		PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
		
		SELECT ?entity ?relation ?target WHERE {
			{
				<%s> ?relation ?target .
				?target rdf:type ?entityType .
				FILTER(?entityType = fhir:%s)
			}
			UNION
			{
				?entity ?relation <%s> .
				?entity rdf:type ?entityType .
				FILTER(?entityType = fhir:%s)
			}
		}
		LIMIT 50
	`, entityID, entityType, entityID, entityType)

	result, err := g.ExecuteSPARQLQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute related entities query: %w", err)
	}

	// Convert SPARQL result to GraphEntity slice
	entities := g.convertToGraphEntities(result)
	
	g.logger.Debug("GraphDB related entities fetched", 
		zap.String("entity_id", entityID),
		zap.String("entity_type", entityType),
		zap.Int("related_count", len(entities)))

	return entities, nil
}

// Close closes the GraphDB client connection
func (g *RealGraphDBClient) Close() error {
	g.logger.Info("GraphDB client connection closed")
	return nil
}

// Helper functions

// convertToGraphContext converts SPARQL results to GraphContext
func (g *RealGraphDBClient) convertToGraphContext(result map[string]interface{}, patientID string) *types.GraphContext {
	graphContext := &types.GraphContext{
		PatientID:   patientID,
		Entities:    []types.GraphEntity{},
		Relationships: []types.GraphRelationship{},
		Timestamp:   time.Now(),
	}

	// Parse SPARQL results
	if results, ok := result["results"].(map[string]interface{}); ok {
		if bindings, ok := results["bindings"].([]interface{}); ok {
			for _, binding := range bindings {
				if bindingMap, ok := binding.(map[string]interface{}); ok {
					entity := g.parseGraphEntity(bindingMap)
					if entity.ID != "" {
						graphContext.Entities = append(graphContext.Entities, entity)
					}
				}
			}
		}
	}

	return graphContext
}

// convertToGraphEntities converts SPARQL results to GraphEntity slice
func (g *RealGraphDBClient) convertToGraphEntities(result map[string]interface{}) []types.GraphEntity {
	var entities []types.GraphEntity

	if results, ok := result["results"].(map[string]interface{}); ok {
		if bindings, ok := results["bindings"].([]interface{}); ok {
			for _, binding := range bindings {
				if bindingMap, ok := binding.(map[string]interface{}); ok {
					entity := g.parseGraphEntity(bindingMap)
					if entity.ID != "" {
						entities = append(entities, entity)
					}
				}
			}
		}
	}

	return entities
}

// parseGraphEntity parses a SPARQL binding into a GraphEntity
func (g *RealGraphDBClient) parseGraphEntity(binding map[string]interface{}) types.GraphEntity {
	entity := types.GraphEntity{
		Properties: make(map[string]interface{}),
	}

	// Extract entity ID
	if entityData, ok := binding["entity"].(map[string]interface{}); ok {
		if value, ok := entityData["value"].(string); ok {
			entity.ID = value
		}
	}

	// Extract entity type
	if typeData, ok := binding["type"].(map[string]interface{}); ok {
		if value, ok := typeData["value"].(string); ok {
			entity.Type = value
		}
	}

	// Extract properties
	if propertyData, ok := binding["property"].(map[string]interface{}); ok {
		if propName, ok := propertyData["value"].(string); ok {
			if valueData, ok := binding["value"].(map[string]interface{}); ok {
				if propValue, ok := valueData["value"]; ok {
					entity.Properties[propName] = propValue
				}
			}
		}
	}

	return entity
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
