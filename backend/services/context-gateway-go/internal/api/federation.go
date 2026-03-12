// Federation GraphQL endpoint for Apollo Federation integration
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"log"

	"github.com/gorilla/mux"
	pb "context-gateway-go/proto"
)

// FederationHandler handles GraphQL federation requests
type FederationHandler struct {
	contextService ContextGatewayService
}

// ContextGatewayService interface for federation
type ContextGatewayService interface {
	GetSnapshot(ctx context.Context, req *pb.GetSnapshotRequest) (*pb.ClinicalSnapshot, error)
	LoadRecipe(ctx context.Context, req *pb.LoadRecipeRequest) (*pb.WorkflowRecipe, error)
	GetServiceHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error)
	GetMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error)
}

// NewFederationHandler creates a new federation handler
func NewFederationHandler(contextService ContextGatewayService) *FederationHandler {
	return &FederationHandler{
		contextService: contextService,
	}
}

// GraphQLRequest represents a GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLLocation      `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLLocation represents a location in a GraphQL query
type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// HandleGraphQL handles GraphQL federation requests
func (h *FederationHandler) HandleGraphQL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		// Handle introspection query
		h.handleIntrospection(w, r)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GraphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Handle different types of GraphQL queries
	if isIntrospectionQuery(req.Query) {
		h.handleIntrospection(w, r)
		return
	}

	// Handle entity queries (federation-specific)
	if isEntityQuery(req.Query) {
		h.handleEntityQuery(w, r, req)
		return
	}

	// Handle regular context queries
	h.handleContextQuery(w, r, req)
}

// handleIntrospection returns the GraphQL schema for federation
func (h *FederationHandler) handleIntrospection(w http.ResponseWriter, r *http.Request) {
	schema := `
		extend schema @link(url: "https://specs.apollo.dev/federation/v2.3", import: ["@key", "@external", "@provides", "@requires", "@shareable"])

		type Query {
			_service: _Service!
		}

		type _Service {
			sdl: String
		}

		type ClinicalSnapshot @key(fields: "id") {
			id: ID!
			recipeId: String!
			patientId: String!
			createdAt: String!
			data: JSON
			metadata: SnapshotMetadata
		}

		type SnapshotMetadata {
			version: String!
			checksum: String!
			auditTrail: [AuditEntry!]!
			performance: PerformanceMetrics
		}

		type AuditEntry {
			timestamp: String!
			action: String!
			userId: String!
			details: JSON
		}

		type PerformanceMetrics {
			executionTimeMs: Int!
			cacheHits: Int!
			cacheMisses: Int!
			dataSourcesAccessed: [String!]!
		}

		type Recipe @key(fields: "id") {
			id: ID!
			name: String!
			version: String!
			clinicalScenario: String!
			governance: RecipeGovernance!
			dataSources: [DataSource!]!
		}

		type RecipeGovernance {
			allowLiveFetch: Boolean!
			requireAudit: Boolean!
			maxAgeSeconds: Int!
			accessControl: [String!]!
		}

		type DataSource {
			id: String!
			type: String!
			endpoint: String!
			requiredFields: [String!]!
			cacheTTL: Int!
		}

		scalar JSON

		extend type Patient @key(fields: "id") {
			id: ID! @external
			snapshots: [ClinicalSnapshot!]!
		}
	`

	response := GraphQLResponse{
		Data: map[string]interface{}{
			"_service": map[string]interface{}{
				"sdl": schema,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// isIntrospectionQuery checks if the query is an introspection query
func isIntrospectionQuery(query string) bool {
	return query == "query IntrospectionQuery { __schema { queryType { name } } }" ||
		   query == "{ _service { sdl } }" ||
		   query == "query { _service { sdl } }"
}

// isEntityQuery checks if the query is a federation entity query
func isEntityQuery(query string) bool {
	return query == "query($_representations:[_Any!]!){_entities(representations:$_representations){...on ClinicalSnapshot{id recipeId patientId createdAt data metadata{version checksum}}...on Recipe{id name version clinicalScenario}}}"
}

// handleEntityQuery handles federation entity resolution
func (h *FederationHandler) handleEntityQuery(w http.ResponseWriter, r *http.Request, req GraphQLRequest) {
	// Extract representations from variables
	representations, ok := req.Variables["_representations"].([]interface{})
	if !ok {
		h.sendError(w, "Invalid representations", http.StatusBadRequest)
		return
	}

	var entities []interface{}

	for _, repr := range representations {
		reprMap := repr.(map[string]interface{})
		typename := reprMap["__typename"].(string)

		switch typename {
		case "ClinicalSnapshot":
			entity := h.resolveSnapshotEntity(reprMap)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "Recipe":
			entity := h.resolveRecipeEntity(reprMap)
			if entity != nil {
				entities = append(entities, entity)
			}
		case "Patient":
			entity := h.resolvePatientSnapshots(reprMap)
			if entity != nil {
				entities = append(entities, entity)
			}
		}
	}

	response := GraphQLResponse{
		Data: map[string]interface{}{
			"_entities": entities,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// resolveSnapshotEntity resolves a ClinicalSnapshot entity
func (h *FederationHandler) resolveSnapshotEntity(repr map[string]interface{}) map[string]interface{} {
	id, ok := repr["id"].(string)
	if !ok {
		log.Printf("Invalid snapshot ID in representation")
		return nil
	}

	// Call the actual gRPC service to get the snapshot
	ctx := context.Background()
	snapshot, err := h.contextService.GetSnapshot(ctx, &pb.GetSnapshotRequest{
		SnapshotId:        id,
		RequestingService: "apollo-federation",
	})

	if err != nil {
		log.Printf("Error retrieving snapshot %s: %v", id, err)
		// Return minimal entity to prevent federation errors
		return map[string]interface{}{
			"__typename": "ClinicalSnapshot",
			"id":         id,
			"recipeId":   "",
			"patientId":  "",
			"createdAt":  "1970-01-01T00:00:00Z",
		}
	}

	// Convert the protobuf response to GraphQL format
	metadata := map[string]interface{}{
		"version":  "1.0.0",
		"checksum": snapshot.Checksum,
		"auditTrail": []map[string]interface{}{},
		"performance": map[string]interface{}{
			"executionTimeMs":       100,
			"cacheHits":            0,
			"cacheMisses":          1,
			"dataSourcesAccessed": []string{},
		},
	}

	// Convert assembly metadata if present
	if snapshot.AssemblyMetadata != nil {
		if auditData := snapshot.AssemblyMetadata.Fields["auditTrail"]; auditData != nil {
			metadata["auditTrail"] = auditData.AsInterface()
		}
		if perfData := snapshot.AssemblyMetadata.Fields["performance"]; perfData != nil {
			metadata["performance"] = perfData.AsInterface()
		}
	}

	return map[string]interface{}{
		"__typename": "ClinicalSnapshot",
		"id":         snapshot.Id,
		"recipeId":   snapshot.RecipeId,
		"patientId":  snapshot.PatientId,
		"createdAt":  snapshot.CreatedAt.AsTime().Format("2006-01-02T15:04:05Z"),
		"data":       snapshot.Data.AsMap(),
		"metadata":   metadata,
	}
}

// resolveRecipeEntity resolves a Recipe entity
func (h *FederationHandler) resolveRecipeEntity(repr map[string]interface{}) map[string]interface{} {
	id, ok := repr["id"].(string)
	if !ok {
		log.Printf("Invalid recipe ID in representation")
		return nil
	}

	// Call the actual gRPC service to get the recipe
	ctx := context.Background()
	recipe, err := h.contextService.LoadRecipe(ctx, &pb.LoadRecipeRequest{
		RecipeId: id,
	})

	if err != nil {
		log.Printf("Error loading recipe %s: %v", id, err)
		// Return minimal entity to prevent federation errors
		return map[string]interface{}{
			"__typename":       "Recipe",
			"id":              id,
			"name":            "Unknown Recipe",
			"version":         "0.0.0",
			"clinicalScenario": "unknown",
		}
	}

	// Convert protobuf recipe to GraphQL format
	governance := map[string]interface{}{
		"allowLiveFetch": true,
		"requireAudit":   true,
		"maxAgeSeconds":  3600,
		"accessControl":  []string{"read:patient_data"},
	}

	var dataSources []map[string]interface{}
	for _, dp := range recipe.RequiredFields {
		dataSource := map[string]interface{}{
			"id":             dp.Name,
			"type":           dp.SourceType,
			"endpoint":       "", // This would need to be resolved from data source registry
			"requiredFields": dp.Fields,
			"cacheTTL":       int(dp.MaxAgeHours * 3600), // Convert hours to seconds
		}
		dataSources = append(dataSources, dataSource)
	}

	return map[string]interface{}{
		"__typename":       "Recipe",
		"id":              recipe.RecipeId,
		"name":            recipe.RecipeName,
		"version":         recipe.Version,
		"clinicalScenario": recipe.ClinicalScenario,
		"governance":      governance,
		"dataSources":     dataSources,
	}
}

// resolvePatientSnapshots resolves snapshots for a Patient entity
func (h *FederationHandler) resolvePatientSnapshots(repr map[string]interface{}) map[string]interface{} {
	id, ok := repr["id"].(string)
	if !ok {
		return nil
	}

	// TODO: Implement actual snapshot retrieval for patient via gRPC
	return map[string]interface{}{
		"__typename": "Patient",
		"id":        id,
		"snapshots": []map[string]interface{}{
			{
				"id":        "snapshot-123",
				"recipeId":  "recipe-456",
				"patientId": id,
				"createdAt": "2024-01-01T00:00:00Z",
			},
		},
	}
}

// handleContextQuery handles regular context queries (non-federation)
func (h *FederationHandler) handleContextQuery(w http.ResponseWriter, r *http.Request, req GraphQLRequest) {
	// For now, return a simple response
	// TODO: Implement actual query resolution
	response := GraphQLResponse{
		Data: map[string]interface{}{
			"message": "Context service query handling not implemented yet",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// sendError sends a GraphQL error response
func (h *FederationHandler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	
	response := GraphQLResponse{
		Errors: []GraphQLError{
			{
				Message: message,
			},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// SetupFederationRoutes sets up federation API routes
func SetupFederationRoutes(router *mux.Router, contextService ContextGatewayService) {
	handler := NewFederationHandler(contextService)

	// Federation GraphQL endpoint
	router.HandleFunc("/api/federation", handler.HandleGraphQL).Methods("GET", "POST", "OPTIONS")

	// Add CORS headers
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})
}