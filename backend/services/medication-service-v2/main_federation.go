package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"

	// Import our internal packages
	"medication-service-v2/internal/graphql/types"
	"medication-service-v2/internal/graphql/resolvers"
	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/config"
	"medication-service-v2/internal/infrastructure/google_fhir"
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

# FHIR Medication Resource
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

# FHIR MedicationRequest Resource
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

# External Patient entity
type Patient @key(fields: "id") @extends {
  id: ID! @external
  medicationRequests: [MedicationRequest!]!
}

# FHIR Common Types (shareable across services)
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

	logger.Info("Starting medication-service-v2 with full Apollo Federation support")

	// Initialize Google FHIR client (simplified for demo)
	googleFHIRConfig := &google_fhir.Config{
		ProjectID:       "your-project-id",
		Location:        "us-central1",
		DatasetID:       "medication-dataset",
		FHIRStoreID:     "medication-fhir-store",
		CredentialsPath: "",
	}

	googleFHIRClient := google_fhir.NewGoogleFHIRClient(googleFHIRConfig)

	// Initialize services
	fhirMedicationService := services.NewFHIRMedicationService(logger, googleFHIRClient)
	medicationResolver := resolvers.NewMedicationResolver(fhirMedicationService, logger)

	// Create GraphQL schema with federation support
	schema := createFederationSchema(medicationResolver, logger)

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up routes
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "medication-service-v2",
			"version":   "1.0.0",
			"timestamp": time.Now().ISO8601(),
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
	logger.Info("🚀 GraphQL Federation endpoint: http://localhost:" + port + "/api/federation")
	logger.Info("🎮 GraphQL Playground: http://localhost:" + port + "/graphql")
	logger.Info("💚 Health check: http://localhost:" + port + "/health")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func createFederationSchema(medicationResolver *resolvers.MedicationResolver, logger *zap.Logger) graphql.Schema {
	// Query type with federation support
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"medication": &graphql.Field{
				Type: types.MedicationType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: medicationResolver.GetMedication,
			},
			"medications": &graphql.Field{
				Type: graphql.NewList(types.MedicationType),
				Args: graphql.FieldConfigArgument{
					"patientId": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"status": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 50,
					},
					"offset": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: medicationResolver.GetMedications,
			},
			"medicationRequest": &graphql.Field{
				Type: types.MedicationRequestType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: medicationResolver.GetMedicationRequest,
			},
			"medicationRequests": &graphql.Field{
				Type: graphql.NewList(types.MedicationRequestType),
				Args: graphql.FieldConfigArgument{
					"patientId": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"status": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 50,
					},
					"offset": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: medicationResolver.GetMedicationRequests,
			},
			// Federation required fields
			"_service": &graphql.Field{
				Type: types.ServiceType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return map[string]interface{}{
						"sdl": federationSDL,
					}, nil
				},
			},
			"_entities": &graphql.Field{
				Type: graphql.NewList(types.EntityUnion),
				Args: graphql.FieldConfigArgument{
					"representations": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(types.AnyScalar))),
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
							medication, err := medicationResolver.GetMedication(graphql.ResolveParams{
								Context: p.Context,
								Args:    map[string]interface{}{"id": id},
							})
							if err != nil {
								logger.Error("Failed to resolve medication entity", zap.String("id", id), zap.Error(err))
								continue
							}
							entities = append(entities, medication)

						case "MedicationRequest":
							medicationRequest, err := medicationResolver.GetMedicationRequest(graphql.ResolveParams{
								Context: p.Context,
								Args:    map[string]interface{}{"id": id},
							})
							if err != nil {
								logger.Error("Failed to resolve medication request entity", zap.String("id", id), zap.Error(err))
								continue
							}
							entities = append(entities, medicationRequest)

						case "Patient":
							// For Patient entities, we return medication requests
							medicationRequests, err := medicationResolver.GetMedicationRequests(graphql.ResolveParams{
								Context: p.Context,
								Args:    map[string]interface{}{"patientId": id},
							})
							if err != nil {
								logger.Error("Failed to resolve patient medication requests", zap.String("patientId", id), zap.Error(err))
								continue
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

	// Mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createMedicationRequest": &graphql.Field{
				Type: types.MedicationRequestType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(types.CreateMedicationRequestInput),
					},
				},
				Resolve: medicationResolver.CreateMedicationRequest,
			},
			"updateMedicationRequest": &graphql.Field{
				Type: types.MedicationRequestType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(types.UpdateMedicationRequestInput),
					},
				},
				Resolve: medicationResolver.UpdateMedicationRequest,
			},
			"deleteMedicationRequest": &graphql.Field{
				Type: graphql.Boolean,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
				},
				Resolve: medicationResolver.DeleteMedicationRequest,
			},
		},
	})

	// Create the schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
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

func graphqlPlayground(endpoint string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>GraphQL Playground</title>
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
        <div>Loading...</div>
    </div>
    <script>
        window.addEventListener('load', function (event) {
            GraphQLPlayground.init(document.getElementById('root'), {
                endpoint: '%s',
                settings: {
                    'editor.theme': 'light',
                    'editor.fontSize': 14,
                    'editor.fontFamily': '"Source Code Pro", "Consolas", "Inconsolata", "Droid Sans Mono", "Monaco", monospace',
                }
            })
        })
    </script>
</body>
</html>`, endpoint)
}