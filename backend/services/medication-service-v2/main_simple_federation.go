package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"medication-service-v2/internal/config"
	"medication-service-v2/internal/infrastructure/google_fhir"
	grpcClient "medication-service-v2/internal/grpc"
	"go.uber.org/zap"
	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// Federation SDL for medication service
const federationSDL = `
directive @key(fields: String!) on OBJECT | INTERFACE
directive @external on FIELD_DEFINITION
directive @requires(fields: String!) on FIELD_DEFINITION
directive @provides(fields: String!) on FIELD_DEFINITION
directive @shareable on FIELD_DEFINITION | OBJECT

extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@external", "@requires", "@provides", "@shareable"])

type Query {
  medication(id: ID!): Medication
  medications(patientId: String, status: String, limit: Int = 50, offset: Int = 0): [Medication!]!
  medicationRequest(id: ID!): MedicationRequest
  medicationRequests(patientId: String, status: String, limit: Int = 50, offset: Int = 0): [MedicationRequest!]!
  _service: _Service!
  _entities(representations: [_Any!]!): [_Entity]!
}

type Mutation {
  createMedicationRequest(input: CreateMedicationRequestInput!): MedicationRequest!
  updateMedicationRequest(id: ID!, input: UpdateMedicationRequestInput!): MedicationRequest!
  deleteMedicationRequest(id: ID!): Boolean!
}

scalar _Any

type _Service {
  sdl: String!
}

union _Entity = Medication | MedicationRequest | Patient

# FHIR Medication Resource with Federation Key
type Medication @key(fields: "id") {
  id: ID!
  identifier: [Identifier!]
  code: CodeableConcept
  status: String
  manufacturer: Reference
  form: CodeableConcept
  amount: Ratio
  ingredient: [MedicationIngredient!]
  batch: MedicationBatch
}

# FHIR MedicationRequest Resource with Federation Key
type MedicationRequest @key(fields: "id") {
  id: ID!
  identifier: [Identifier!]
  status: String!
  statusReason: CodeableConcept
  intent: String!
  category: [CodeableConcept!]
  priority: String
  doNotPerform: Boolean
  medicationCodeableConcept: CodeableConcept
  medicationReference: Reference
  subject: Reference!
  encounter: Reference
  authoredOn: String
  requester: Reference
  dosageInstruction: [Dosage!]
  dispenseRequest: MedicationRequestDispenseRequest
  substitution: MedicationRequestSubstitution
  reasonCode: [CodeableConcept!]
  note: [Annotation!]
}

# External Patient entity (from patient service)
extend type Patient @key(fields: "id") {
  id: ID! @external
  medicationRequests: [MedicationRequest!]!
}

# Shareable FHIR Common Types
type CodeableConcept @shareable {
  text: String
  coding: [Coding!]
}

type Coding @shareable {
  system: String
  code: String
  display: String
  version: String
  userSelected: Boolean
}

type Identifier @shareable {
  use: String
  type: CodeableConcept
  system: String
  value: String
  period: Period
  assigner: Reference
}

type Reference @shareable {
  reference: String
  display: String
  type: String
  identifier: Identifier
}

type Period @shareable {
  start: String
  end: String
}

type Quantity @shareable {
  value: Float
  unit: String
  system: String
  code: String
  comparator: String
}

type Ratio @shareable {
  numerator: Quantity
  denominator: Quantity
}

type Dosage @shareable {
  sequence: Int
  text: String
  additionalInstruction: [CodeableConcept!]
  patientInstruction: String
  timing: Timing
  asNeededBoolean: Boolean
  asNeededCodeableConcept: CodeableConcept
  site: CodeableConcept
  route: CodeableConcept
  method: CodeableConcept
  doseAndRate: [DoseAndRate!]
  maxDosePerPeriod: Ratio
  maxDosePerAdministration: Quantity
  maxDosePerLifetime: Quantity
}

type Timing @shareable {
  event: [String!]
  repeat: TimingRepeat
  code: CodeableConcept
}

type TimingRepeat @shareable {
  boundsRange: Range
  boundsPeriod: Period
  boundsQuantity: Quantity
  count: Int
  countMax: Int
  duration: Float
  durationMax: Float
  durationUnit: String
  frequency: Int
  frequencyMax: Int
  period: Float
  periodMax: Float
  periodUnit: String
  dayOfWeek: [String!]
  timeOfDay: [String!]
  when: [String!]
  offset: Int
}

type Range @shareable {
  low: Quantity
  high: Quantity
}

type DoseAndRate @shareable {
  type: CodeableConcept
  doseRange: Range
  doseQuantity: Quantity
  rateRatio: Ratio
  rateRange: Range
  rateQuantity: Quantity
}

type MedicationIngredient {
  itemCodeableConcept: CodeableConcept
  itemReference: Reference
  isActive: Boolean
  strength: Ratio
}

type MedicationBatch {
  lotNumber: String
  expirationDate: String
}

type Annotation @shareable {
  authorReference: Reference
  authorString: String
  time: String
  text: String!
}

type MedicationRequestDispenseRequest {
  initialFill: MedicationRequestInitialFill
  dispenseInterval: Duration
  validityPeriod: Period
  numberOfRepeatsAllowed: Int
  quantity: Quantity
  expectedSupplyDuration: Duration
  performer: Reference
}

type MedicationRequestInitialFill {
  quantity: Quantity
  duration: Duration
}

type Duration @shareable {
  value: Float
  unit: String
  system: String
  code: String
}

type MedicationRequestSubstitution {
  allowedBoolean: Boolean
  allowedCodeableConcept: CodeableConcept
  reason: CodeableConcept
}

# Input Types
input CreateMedicationRequestInput {
  status: String!
  intent: String!
  medicationCodeableConcept: CodeableConceptInput
  medicationReference: ReferenceInput
  subjectId: String!
  requesterId: String
  encounterId: String
  dosageInstructions: [DosageInput!]
  priority: String
  reasonCode: [CodeableConceptInput!]
  note: String
}

input UpdateMedicationRequestInput {
  status: String
  priority: String
  dosageInstructions: [DosageInput!]
  reasonCode: [CodeableConceptInput!]
  note: String
}

input CodeableConceptInput {
  text: String
  coding: [CodingInput!]
}

input CodingInput {
  system: String
  code: String
  display: String
  version: String
}

input ReferenceInput {
  reference: String
  display: String
  type: String
}

input DosageInput {
  sequence: Int
  text: String
  patientInstruction: String
  asNeededBoolean: Boolean
  route: CodeableConceptInput
  method: CodeableConceptInput
  doseQuantity: QuantityInput
}

input QuantityInput {
  value: Float
  unit: String
  system: String
  code: String
}
`

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting medication-service-v2 with Apollo Federation and Google FHIR support")

	// Load configuration from environment
	cfg := loadConfigFromEnv()

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Google FHIR client if enabled
	var googleFHIRClient *google_fhir.GoogleFHIRClient
	if cfg.GoogleFHIR.Enabled {
		googleFHIRConfig := &google_fhir.Config{
			ProjectID:       cfg.GoogleFHIR.ProjectID,
			Location:        cfg.GoogleFHIR.Location,
			DatasetID:       cfg.GoogleFHIR.DatasetID,
			FHIRStoreID:     cfg.GoogleFHIR.FHIRStoreID,
			CredentialsPath: cfg.GoogleFHIR.CredentialsPath,
		}

		googleFHIRClient = google_fhir.NewGoogleFHIRClient(googleFHIRConfig)

		// Initialize the client
		if err := googleFHIRClient.Initialize(ctx); err != nil {
			logger.Error("Failed to initialize Google FHIR client", zap.Error(err))
			logger.Warn("Continuing without Google FHIR - service will use fallback behavior")
			googleFHIRClient = nil
		} else {
			logger.Info("Successfully initialized Google FHIR client")
		}
	}

	// Initialize Flow2 gRPC clients for both engines
	var flow2Clients *grpcClient.Flow2Clients
	flow2Clients, err = grpcClient.NewFlow2Clients()
	if err != nil {
		logger.Error("Failed to initialize Flow2 gRPC clients", zap.Error(err))
		logger.Warn("Continuing without Flow2 engines - service will operate without engine support")
		flow2Clients = nil
	} else {
		logger.Info("Successfully initialized Flow2 gRPC clients for Go and Rust engines")
		// Test connectivity to both engines
		goHealthy, rustHealthy, _ := flow2Clients.HealthCheckBothEngines(ctx)
		logger.Info("Flow2 engine health status",
			zap.Bool("go_engine_healthy", goHealthy),
			zap.Bool("rust_engine_healthy", rustHealthy))
	}

	// Create GraphQL schema with federation support
	schema := createFederationSchema(logger, googleFHIRClient)

	// Set up routes
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "healthy",
			"service":     "medication-service-v2",
			"version":     "2.0.0",
			"timestamp":   time.Now().Format(time.RFC3339),
			"federation":  "enabled",
			"google_fhir": googleFHIRClient != nil,
			"flow2_engines": gin.H{
				"go_engine":   "http://localhost:8080",
				"rust_engine": "http://localhost:8090",
			},
		})
	})

	// Federation endpoint for Apollo Gateway
	router.POST("/api/federation", func(c *gin.Context) {
		executeGraphQL(c, schema, logger)
	})

	// GraphQL endpoint (for playground and direct queries)
	router.POST("/graphql", func(c *gin.Context) {
		executeGraphQL(c, schema, logger)
	})

	// GraphQL Playground
	router.GET("/graphql", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, graphqlPlayground("/graphql"))
	})

	// Federation introspection endpoint
	router.GET("/api/federation", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"sdl": federationSDL,
		})
	})

	// Start server
	port := getPort()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting GraphQL Federation server",
			zap.String("port", port),
			zap.String("federation_url", fmt.Sprintf("http://localhost:%s/api/federation", port)),
			zap.String("playground_url", fmt.Sprintf("http://localhost:%s/graphql", port)),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	logger.Info("medication-service-v2 started successfully!")
	logger.Info("🚀 Apollo Federation endpoint: http://localhost:" + port + "/api/federation")
	logger.Info("🎮 GraphQL Playground: http://localhost:" + port + "/graphql")
	logger.Info("💚 Health check: http://localhost:" + port + "/health")
	if flow2Clients != nil {
		logger.Info("🔗 Connected to Flow2 Go engine via gRPC: localhost:8080")
		logger.Info("🦀 Connected to Flow2 Rust engine via gRPC: localhost:8091")
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Close gRPC connections
	if flow2Clients != nil {
		logger.Info("Closing Flow2 gRPC connections...")
		flow2Clients.Close()
		logger.Info("✅ Flow2 gRPC connections closed")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func createFederationSchema(logger *zap.Logger, googleFHIRClient *google_fhir.GoogleFHIRClient) graphql.Schema {
	// Define _Any scalar for federation
	anyScalar := graphql.NewScalar(graphql.ScalarConfig{
		Name:        "_Any",
		Description: "The _Any scalar is used to pass representations of entities from external services.",
		Serialize: func(value interface{}) interface{} {
			return value
		},
		ParseValue: func(value interface{}) interface{} {
			return value
		},
		ParseLiteral: func(valueAST ast.Value) interface{} {
			return valueAST.GetValue()
		},
	})

	// _Service type for federation
	serviceType := graphql.NewObject(graphql.ObjectConfig{
		Name: "_Service",
		Fields: graphql.Fields{
			"sdl": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

	// CodeableConcept type for FHIR
	codeableConceptType := graphql.NewObject(graphql.ObjectConfig{
		Name: "CodeableConcept",
		Fields: graphql.Fields{
			"text": &graphql.Field{Type: graphql.String},
			"coding": &graphql.Field{
				Type: graphql.NewList(graphql.NewObject(graphql.ObjectConfig{
					Name: "Coding",
					Fields: graphql.Fields{
						"system":  &graphql.Field{Type: graphql.String},
						"code":    &graphql.Field{Type: graphql.String},
						"display": &graphql.Field{Type: graphql.String},
						"version": &graphql.Field{Type: graphql.String},
						"userSelected": &graphql.Field{Type: graphql.Boolean},
					},
				})),
			},
		},
	})

	// Identifier type for FHIR
	identifierType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Identifier",
		Fields: graphql.Fields{
			"use":    &graphql.Field{Type: graphql.String},
			"type":   &graphql.Field{Type: codeableConceptType},
			"system": &graphql.Field{Type: graphql.String},
			"value":  &graphql.Field{Type: graphql.String},
		},
	})

	// Reference type for FHIR
	referenceType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Reference",
		Fields: graphql.Fields{
			"reference":  &graphql.Field{Type: graphql.String},
			"display":    &graphql.Field{Type: graphql.String},
			"type":       &graphql.Field{Type: graphql.String},
			"identifier": &graphql.Field{Type: identifierType},
		},
	})

	// FHIR Medication type with proper structure
	medicationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Medication",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"identifier": &graphql.Field{
				Type: graphql.NewList(identifierType),
			},
			"code": &graphql.Field{
				Type: codeableConceptType,
			},
			"status": &graphql.Field{
				Type: graphql.String,
			},
			"manufacturer": &graphql.Field{
				Type: referenceType,
			},
			"form": &graphql.Field{
				Type: codeableConceptType,
			},
		},
	})

	// Basic MedicationRequest type
	medicationRequestType := graphql.NewObject(graphql.ObjectConfig{
		Name: "MedicationRequest",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"status": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"intent": &graphql.Field{
				Type: graphql.NewNonNull(graphql.String),
			},
			"subject": &graphql.Field{
				Type: graphql.String,
			},
		},
	})

	// Patient type (external)
	patientType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Patient",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"medicationRequests": &graphql.Field{
				Type: graphql.NewList(medicationRequestType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// Mock medication requests for patient
					patientID := p.Source.(map[string]interface{})["id"].(string)
					return []map[string]interface{}{
						{
							"id":      "med-req-1",
							"status":  "active",
							"intent":  "order",
							"subject": patientID,
						},
						{
							"id":      "med-req-2",
							"status":  "completed",
							"intent":  "order",
							"subject": patientID,
						},
					}, nil
				},
			},
		},
	})

	// _Entity union
	entityUnion := graphql.NewUnion(graphql.UnionConfig{
		Name: "_Entity",
		Types: []*graphql.Object{
			medicationType,
			medicationRequestType,
			patientType,
		},
		ResolveType: func(p graphql.ResolveTypeParams) *graphql.Object {
			if obj, ok := p.Value.(map[string]interface{}); ok {
				if resourceType, exists := obj["__typename"]; exists {
					switch resourceType {
					case "Medication":
						return medicationType
					case "MedicationRequest":
						return medicationRequestType
					case "Patient":
						return patientType
					}
				}
			}
			return nil
		},
	})

	// Query type with federation support
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"medication": &graphql.Field{
				Type: medicationType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := p.Args["id"].(string)

					// Only use Google FHIR - no fallbacks
					if googleFHIRClient != nil {
						resource, err := googleFHIRClient.GetResource(p.Context, "Medication", id)
						if err != nil {
							logger.Error("Failed to fetch medication from FHIR store", zap.Error(err), zap.String("id", id))
							return nil, fmt.Errorf("medication not found: %s", id)
						}
						if resource != nil {
							logger.Info("Successfully fetched medication from FHIR store", zap.String("id", id))
							return resource, nil
						}
						logger.Info("Medication not found in FHIR store", zap.String("id", id))
						return nil, fmt.Errorf("medication not found: %s", id)
					}

					logger.Error("Google FHIR client not available")
					return nil, fmt.Errorf("FHIR service unavailable")
				},
			},
			"medications": &graphql.Field{
				Type: graphql.NewList(medicationType),
				Args: graphql.FieldConfigArgument{
					"patientId": &graphql.ArgumentConfig{Type: graphql.String},
					"status":    &graphql.ArgumentConfig{Type: graphql.String},
					"limit":     &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 50},
					"offset":    &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 0},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// Only use Google FHIR - no fallbacks
					if googleFHIRClient != nil {
						// Build search parameters
						params := make(map[string]string)
						if patientId, ok := p.Args["patientId"].(string); ok && patientId != "" {
							params["patient"] = patientId
						}
						if status, ok := p.Args["status"].(string); ok && status != "" {
							params["status"] = status
						}
						if limit, ok := p.Args["limit"].(int); ok && limit > 0 {
							params["_count"] = strconv.Itoa(limit)
						}

						resources, err := googleFHIRClient.SearchResources(p.Context, "Medication", params)
						if err != nil {
							logger.Error("Failed to search medications from FHIR store", zap.Error(err))
							return nil, fmt.Errorf("failed to search medications: %v", err)
						}

						logger.Info("Successfully searched medications from FHIR store", zap.Int("count", len(resources)))
						return resources, nil
					}

					logger.Error("Google FHIR client not available")
					return nil, fmt.Errorf("FHIR service unavailable")
				},
			},
			"medicationRequest": &graphql.Field{
				Type: medicationRequestType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := p.Args["id"].(string)
					return map[string]interface{}{
						"id":      id,
						"status":  "active",
						"intent":  "order",
						"subject": "patient-123",
					}, nil
				},
			},
			"medicationRequests": &graphql.Field{
				Type: graphql.NewList(medicationRequestType),
				Args: graphql.FieldConfigArgument{
					"patientId": &graphql.ArgumentConfig{Type: graphql.String},
					"status":    &graphql.ArgumentConfig{Type: graphql.String},
					"limit":     &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 50},
					"offset":    &graphql.ArgumentConfig{Type: graphql.Int, DefaultValue: 0},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					patientID := p.Args["patientId"]
					return []map[string]interface{}{
						{"id": "req-1", "status": "active", "intent": "order", "subject": patientID},
						{"id": "req-2", "status": "completed", "intent": "order", "subject": patientID},
					}, nil
				},
			},
			// Federation required fields
			"_service": &graphql.Field{
				Type: serviceType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return map[string]interface{}{
						"sdl": federationSDL,
					}, nil
				},
			},
			"_entities": &graphql.Field{
				Type: graphql.NewList(entityUnion),
				Args: graphql.FieldConfigArgument{
					"representations": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(anyScalar))),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					representations := p.Args["representations"].([]interface{})
					var entities []interface{}

					for _, representation := range representations {
						repr := representation.(map[string]interface{})
						typename := repr["__typename"].(string)
						id := repr["id"].(string)

						switch typename {
						case "Medication":
							entities = append(entities, map[string]interface{}{
								"id":         id,
								"status":     "active",
								"code":       "medication-" + id,
								"__typename": "Medication",
							})
						case "MedicationRequest":
							entities = append(entities, map[string]interface{}{
								"id":         id,
								"status":     "active",
								"intent":     "order",
								"subject":    "patient-123",
								"__typename": "MedicationRequest",
							})
						case "Patient":
							medicationRequests := []map[string]interface{}{
								{"id": "req-1", "status": "active", "intent": "order", "subject": id},
								{"id": "req-2", "status": "completed", "intent": "order", "subject": id},
							}
							entities = append(entities, map[string]interface{}{
								"id":                 id,
								"medicationRequests": medicationRequests,
								"__typename":         "Patient",
							})
						}
					}

					return entities, nil
				},
			},
		},
	})

	// Create the schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: queryType,
	})

	if err != nil {
		logger.Fatal("Failed to create GraphQL schema", zap.Error(err))
	}

	return schema
}

func executeGraphQL(c *gin.Context, schema graphql.Schema, logger *zap.Logger) {
	var request struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		logger.Error("Failed to parse GraphQL request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Execute the GraphQL query
	result := graphql.Do(graphql.Params{
		Schema:         schema,
		RequestString:  request.Query,
		VariableValues: request.Variables,
		Context:        c.Request.Context(),
	})

	// Log any errors
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			logger.Error("GraphQL execution error", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, result)
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8005"
	}
	return port
}

// Simplified config loading from environment (same as patient-service)
func loadConfigFromEnv() *config.Config {
	cfg := &config.Config{}

	// Google FHIR Config
	cfg.GoogleFHIR.Enabled = getEnvBool("USE_GOOGLE_HEALTHCARE_API", true)
	cfg.GoogleFHIR.ProjectID = getEnv("GOOGLE_CLOUD_PROJECT_ID", "cardiofit-905a8")
	cfg.GoogleFHIR.Location = getEnv("GOOGLE_CLOUD_LOCATION", "asia-south1")
	cfg.GoogleFHIR.DatasetID = getEnv("GOOGLE_CLOUD_DATASET_ID", "clinical-synthesis-hub")
	cfg.GoogleFHIR.FHIRStoreID = getEnv("GOOGLE_CLOUD_FHIR_STORE_ID", "fhir-store")
	cfg.GoogleFHIR.CredentialsPath = getEnv("GOOGLE_CLOUD_CREDENTIALS_PATH", "credentials/google-credentials.json")

	// Server Config
	cfg.Server.HTTP.Port = getEnv("GRAPHQL_PORT", "8005")

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func graphqlPlayground(endpoint string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>GraphQL Playground - Medication Service</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/css/index.css" />
    <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/favicon.png" />
    <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react/build/static/js/middleware.js"></script>
</head>
<body>
    <div id="root">
        <style>
            body { margin: 0; }
            #root {
                height: 100vh;
                width: 100vw;
                display: flex;
                align-items: center;
                justify-content: center;
                font-family: 'Open Sans', sans-serif;
                font-size: 14px;
            }
        </style>
        <div>Loading Medication Service GraphQL Playground...</div>
    </div>
    <script>
        window.addEventListener('load', function (event) {
            GraphQLPlayground.init(document.getElementById('root'), {
                endpoint: '%s',
                settings: {
                    'editor.theme': 'light',
                    'editor.fontSize': 14,
                    'editor.fontFamily': '"Source Code Pro", "Consolas", "Inconsolata", "Droid Sans Mono", "Monaco", monospace',
                    'general.betaUpdates': false,
                    'request.credentials': 'omit',
                }
            })
        })
    </script>
</body>
</html>`, endpoint)
}