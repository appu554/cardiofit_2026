package graphql

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/graphql-go/graphql"

	"kb-clinical-context/internal/models"
	"kb-clinical-context/internal/services"
)

// GraphQLHandler wraps the context service for GraphQL operations
type GraphQLHandler struct {
	contextService *services.ContextService
	schema         graphql.Schema
}

// NewGraphQLHandler creates a new GraphQL handler with federation support
func NewGraphQLHandler(contextService *services.ContextService) (*GraphQLHandler, error) {
	handler := &GraphQLHandler{
		contextService: contextService,
	}

	schema, err := handler.createSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL schema: %w", err)
	}

	handler.schema = schema
	return handler, nil
}

// GetSchema returns the GraphQL schema
func (h *GraphQLHandler) GetSchema() graphql.Schema {
	return h.schema
}

// createSchema builds the federation-compatible GraphQL schema
func (h *GraphQLHandler) createSchema() (graphql.Schema, error) {
	// Use String for DateTime and UUID for simplicity
	dateTimeType := graphql.String
	jsonType := graphql.String // JSON as string for now

	// Patient type for federation
	patientType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Patient",
		Fields: graphql.Fields{
			"id": &graphql.Field{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"clinicalContext": &graphql.Field{
				Type: graphql.String, // Will be defined later
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// Implementation will be added when we have the full context type
					return nil, nil
				},
			},
		},
	})

	// Phenotype Definition Type
	phenotypeDefinitionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PhenotypeDefinition",
		Fields: graphql.Fields{
			"phenotype_id": &graphql.Field{Type: graphql.String},
			"name": &graphql.Field{Type: graphql.String},
			"category": &graphql.Field{Type: graphql.String},
			"description": &graphql.Field{Type: graphql.String},
			"algorithm_type": &graphql.Field{Type: graphql.String},
			"status": &graphql.Field{Type: graphql.String},
			"severity": &graphql.Field{Type: graphql.String},
			"version": &graphql.Field{Type: graphql.String},
			"confidence_threshold": &graphql.Field{Type: graphql.Float},
			"match_threshold": &graphql.Field{Type: graphql.Float},
			"created_at": &graphql.Field{Type: dateTimeType},
			"updated_at": &graphql.Field{Type: dateTimeType},
			"icd10_codes": &graphql.Field{Type: graphql.NewList(graphql.String)},
			"snomed_codes": &graphql.Field{Type: graphql.NewList(graphql.String)},
			"validation": &graphql.Field{Type: jsonType},
		},
	})

	// CareGap type (defined early for use in query)
	careGapType := graphql.NewObject(graphql.ObjectConfig{
		Name: "CareGap",
		Fields: graphql.Fields{
			"id":          &graphql.Field{Type: graphql.String},
			"type":        &graphql.Field{Type: graphql.String},
			"description": &graphql.Field{Type: graphql.String},
			"priority":    &graphql.Field{Type: graphql.String},
			"dueDays":     &graphql.Field{Type: graphql.Int},
			"actions":     &graphql.Field{Type: graphql.NewList(graphql.String)},
		},
	})

	// Query type with federation support
	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			// Federation queries
			"_entities": &graphql.Field{
				Type: graphql.NewList(patientType),
				Args: graphql.FieldConfigArgument{
					"representations": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.NewList(jsonType)),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					representations, ok := p.Args["representations"].([]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid representations")
					}

					var entities []interface{}
					for _, repr := range representations {
						reprMap, ok := repr.(map[string]interface{})
						if !ok {
							continue
						}

						typename, ok := reprMap["__typename"].(string)
						if !ok || typename != "Patient" {
							continue
						}

						id, ok := reprMap["id"].(string)
						if !ok {
							continue
						}

						// Create patient entity
						patient := map[string]interface{}{
							"id": id,
						}
						entities = append(entities, patient)
					}

					return entities, nil
				},
			},
			"_service": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "_Service",
					Fields: graphql.Fields{
						"sdl": &graphql.Field{
							Type: graphql.String,
							Resolve: func(p graphql.ResolveParams) (interface{}, error) {
								return getSDL(), nil
							},
						},
					},
				}),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return map[string]interface{}{}, nil
				},
			},
			// Phenotype Definitions Query - direct access to our MongoDB data
			"phenotypeDefinitions": &graphql.Field{
				Type: graphql.NewList(phenotypeDefinitionType),
				Args: graphql.FieldConfigArgument{
					"domain": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"status": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
					"limit": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
					"offset": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					domain, _ := p.Args["domain"].(string)
					status, _ := p.Args["status"].(string)
					limit, ok := p.Args["limit"].(int)
					if !ok {
						limit = 50
					}
					offset, ok := p.Args["offset"].(int)
					if !ok {
						offset = 0
					}

					// Use the context service to get phenotype definitions from MongoDB
					phenotypes, _, err := h.contextService.GetPhenotypeDefinitions(domain, status, limit, offset)
					if err != nil {
						return nil, fmt.Errorf("failed to get phenotype definitions: %w", err)
					}

					return phenotypes, nil
				},
			},
			// Health check
			"systemHealth": &graphql.Field{
				Type: graphql.NewObject(graphql.ObjectConfig{
					Name: "SystemHealth",
					Fields: graphql.Fields{
						"status": &graphql.Field{
							Type: graphql.String,
						},
						"timestamp": &graphql.Field{
							Type: dateTimeType,
						},
						"checks": &graphql.Field{
							Type: jsonType,
						},
					},
				}),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return map[string]interface{}{
						"status":    "healthy",
						"timestamp": time.Now(),
						"checks": map[string]interface{}{
							"database": map[string]interface{}{
								"status": "healthy",
							},
							"cache": map[string]interface{}{
								"status": "healthy",
							},
						},
					}, nil
				},
			},
			// KB-2B: Patient Care Gaps Query - returns list of CareGap directly
			"patientCareGaps": &graphql.Field{
				Type: graphql.NewList(careGapType),
				Args: graphql.FieldConfigArgument{
					"patientId": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.ID),
					},
					"includeResolved": &graphql.ArgumentConfig{
						Type: graphql.Boolean,
					},
					"timeframeDays": &graphql.ArgumentConfig{
						Type: graphql.Int,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					patientId, ok := p.Args["patientId"].(string)
					if !ok {
						return nil, fmt.Errorf("patientId is required")
					}

					includeResolved := false
					if ir, ok := p.Args["includeResolved"].(bool); ok {
						includeResolved = ir
					}

					timeframeDays := 90 // Default 90 days
					if tf, ok := p.Args["timeframeDays"].(int); ok {
						timeframeDays = tf
					}

					request := models.CareGapsRequest{
						PatientID:       patientId,
						IncludeResolved: includeResolved,
						TimeframeDays:   timeframeDays,
					}

					response, err := h.contextService.IdentifyCareGaps(request)
					if err != nil {
						return nil, fmt.Errorf("failed to identify care gaps: %w", err)
					}

					// Return list of care gaps directly (matches client expectation)
					careGaps := make([]map[string]interface{}, 0, len(response.CareGaps))
					for _, g := range response.CareGaps {
						careGaps = append(careGaps, map[string]interface{}{
							"id":          g.ID.String(),
							"type":        g.Type,
							"description": g.Description,
							"priority":    g.Priority,
							"dueDays":     g.DueDays,
							"actions":     g.Actions,
						})
					}

					return careGaps, nil
				},
			},
		},
	})

	// Build context mutation
	buildContextInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "BuildContextInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"patientId": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"patient": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(jsonType),
			},
			"transactionId": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
		},
	})

	buildContextResponseType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ContextBuildResponse",
		Fields: graphql.Fields{
			"cacheHit": &graphql.Field{
				Type: graphql.Boolean,
			},
			"processedAt": &graphql.Field{
				Type: dateTimeType,
			},
			"phenotypes": &graphql.Field{
				Type: graphql.NewList(graphql.String),
			},
		},
	})

	// ============================================================================
	// KB-2B: Intelligence Service Types
	// ============================================================================

	// DetectedPhenotype type
	detectedPhenotypeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "DetectedPhenotype",
		Fields: graphql.Fields{
			"phenotypeId":        &graphql.Field{Type: graphql.String},
			"name":               &graphql.Field{Type: graphql.String},
			"confidence":         &graphql.Field{Type: graphql.Float},
			"detectedAt":         &graphql.Field{Type: dateTimeType},
			"supportingEvidence": &graphql.Field{Type: graphql.NewList(graphql.String)},
		},
	})

	// PhenotypeDetection input type
	phenotypeDetectionInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "PhenotypeDetectionInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"patientId": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"patientData": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(jsonType),
			},
			"phenotypeIds": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(graphql.String),
			},
		},
	})

	// PhenotypeDetection response type
	phenotypeDetectionResponseType := graphql.NewObject(graphql.ObjectConfig{
		Name: "PhenotypeDetectionResponse",
		Fields: graphql.Fields{
			"patientId":          &graphql.Field{Type: graphql.String},
			"detectedPhenotypes": &graphql.Field{Type: graphql.NewList(detectedPhenotypeType)},
			"totalPhenotypes":    &graphql.Field{Type: graphql.Int},
			"processingTimeMs":   &graphql.Field{Type: graphql.Int},
		},
	})

	// RiskAssessment input type
	riskAssessmentInputType := graphql.NewInputObject(graphql.InputObjectConfig{
		Name: "RiskAssessmentInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"patientId": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(graphql.ID),
			},
			"patientData": &graphql.InputObjectFieldConfig{
				Type: jsonType,
			},
			"riskTypes": &graphql.InputObjectFieldConfig{
				Type: graphql.NewList(graphql.String),
			},
		},
	})

	// RiskAssessment response type
	riskAssessmentResponseType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RiskAssessment",
		Fields: graphql.Fields{
			"patientId":           &graphql.Field{Type: graphql.String},
			"riskScores":          &graphql.Field{Type: jsonType},
			"riskFactors":         &graphql.Field{Type: jsonType},
			"recommendations":     &graphql.Field{Type: graphql.NewList(graphql.String)},
			"confidenceScore":     &graphql.Field{Type: graphql.Float},
			"assessmentTimestamp": &graphql.Field{Type: dateTimeType},
		},
	})

	// Mutation type
	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"buildContext": &graphql.Field{
				Type: buildContextResponseType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(buildContextInputType),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					inputArg, ok := p.Args["input"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid input")
					}

					patientId, ok := inputArg["patientId"].(string)
					if !ok {
						return nil, fmt.Errorf("patientId is required")
					}

					// Handle patient data - can be either map or JSON string
					var patient map[string]interface{}
					switch v := inputArg["patient"].(type) {
					case map[string]interface{}:
						patient = v
					case string:
						// Parse JSON string to map
						if err := json.Unmarshal([]byte(v), &patient); err != nil {
							return nil, fmt.Errorf("invalid patient JSON: %w", err)
						}
					default:
						return nil, fmt.Errorf("patient data is required (expected object or JSON string)")
					}

					if patient == nil {
						return nil, fmt.Errorf("patient data is required")
					}

					// Convert to our internal request format
					buildRequest := models.BuildContextRequest{
						PatientID: patientId,
						Patient:   patient,
					}

					if transactionId, ok := inputArg["transactionId"].(string); ok {
						buildRequest.TransactionID = transactionId
					}

					// Call the context service
					response, err := h.contextService.BuildContext(buildRequest)
					if err != nil {
						return nil, fmt.Errorf("failed to build context: %w", err)
					}

					return map[string]interface{}{
						"cacheHit":    response.CacheHit,
						"processedAt": response.ProcessedAt,
						"phenotypes":  response.Phenotypes,
					}, nil
				},
			},
			// KB-2B: Detect Phenotypes mutation
			"detectPhenotypes": &graphql.Field{
				Type: phenotypeDetectionResponseType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(phenotypeDetectionInputType),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					inputArg, ok := p.Args["input"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid input")
					}

					patientId, ok := inputArg["patientId"].(string)
					if !ok {
						return nil, fmt.Errorf("patientId is required")
					}

					// Handle patient data - can be either map or JSON string
					var patientData map[string]interface{}
					switch v := inputArg["patientData"].(type) {
					case map[string]interface{}:
						patientData = v
					case string:
						if err := json.Unmarshal([]byte(v), &patientData); err != nil {
							return nil, fmt.Errorf("invalid patientData JSON: %w", err)
						}
					default:
						return nil, fmt.Errorf("patientData is required")
					}

					// Extract phenotype IDs if provided
					var phenotypeIDs []string
					if ids, ok := inputArg["phenotypeIds"].([]interface{}); ok {
						for _, id := range ids {
							if s, ok := id.(string); ok {
								phenotypeIDs = append(phenotypeIDs, s)
							}
						}
					}

					request := models.PhenotypeDetectionRequest{
						PatientID:    patientId,
						PatientData:  patientData,
						PhenotypeIDs: phenotypeIDs,
					}

					response, err := h.contextService.DetectPhenotypes(request)
					if err != nil {
						return nil, fmt.Errorf("failed to detect phenotypes: %w", err)
					}

					// Convert detected phenotypes to response format
					phenotypes := make([]map[string]interface{}, 0, len(response.DetectedPhenotypes))
					for _, p := range response.DetectedPhenotypes {
						evidence := make([]string, 0)
						for _, e := range p.SupportingEvidence {
							if s, ok := e["description"].(string); ok {
								evidence = append(evidence, s)
							}
						}
						phenotypes = append(phenotypes, map[string]interface{}{
							"phenotypeId":        p.PhenotypeID,
							"name":               p.PhenotypeID, // Use PhenotypeID as name (model doesn't have separate Name field)
							"confidence":         p.Confidence,
							"detectedAt":         p.DetectedAt,
							"supportingEvidence": evidence,
						})
					}

					return map[string]interface{}{
						"patientId":          response.PatientID,
						"detectedPhenotypes": phenotypes,
						"totalPhenotypes":    response.TotalPhenotypes,
						"processingTimeMs":   response.ProcessingTime,
					}, nil
				},
			},
			// KB-2B: Assess Risk mutation
			"assessRisk": &graphql.Field{
				Type: riskAssessmentResponseType,
				Args: graphql.FieldConfigArgument{
					"input": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(riskAssessmentInputType),
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					inputArg, ok := p.Args["input"].(map[string]interface{})
					if !ok {
						return nil, fmt.Errorf("invalid input")
					}

					patientId, ok := inputArg["patientId"].(string)
					if !ok {
						return nil, fmt.Errorf("patientId is required")
					}

					// Handle patient data - can be either map or JSON string
					var patientData map[string]interface{}
					if pd := inputArg["patientData"]; pd != nil {
						switch v := pd.(type) {
						case map[string]interface{}:
							patientData = v
						case string:
							if err := json.Unmarshal([]byte(v), &patientData); err != nil {
								return nil, fmt.Errorf("invalid patientData JSON: %w", err)
							}
						}
					}

					// Extract risk types if provided
					var riskTypes []string
					if types, ok := inputArg["riskTypes"].([]interface{}); ok {
						for _, t := range types {
							if s, ok := t.(string); ok {
								riskTypes = append(riskTypes, s)
							}
						}
					}

					request := models.RiskAssessmentRequest{
						PatientID:   patientId,
						PatientData: patientData,
						RiskTypes:   riskTypes,
					}

					response, err := h.contextService.AssessRisk(request)
					if err != nil {
						return nil, fmt.Errorf("failed to assess risk: %w", err)
					}

					return map[string]interface{}{
						"patientId":           response.PatientID,
						"riskScores":          response.RiskScores,
						"riskFactors":         response.RiskFactors,
						"recommendations":     response.Recommendations,
						"confidenceScore":     response.ConfidenceScore,
						"assessmentTimestamp": response.AssessmentTimestamp,
					}, nil
				},
			},
		},
	})

	// Create schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})

	return schema, err
}

// getSDL returns the SDL for federation
func getSDL() string {
	return `
		directive @key(fields: String!) on OBJECT | INTERFACE
		directive @external on FIELD_DEFINITION | OBJECT
		directive @requires(fields: String!) on FIELD_DEFINITION
		directive @provides(fields: String!) on FIELD_DEFINITION

		scalar DateTime
		scalar UUID
		scalar JSON

		type Patient @key(fields: "id") {
			id: ID!
			clinicalContext: PatientContext
		}

		type PatientContext {
			id: ID!
			patientId: ID!
			contextId: String!
			timestamp: DateTime!
		}

		type PhenotypeDefinition {
			phenotype_id: String
			name: String
			category: String
			description: String
			algorithm_type: String
			status: String
			severity: String
			version: String
			confidence_threshold: Float
			match_threshold: Float
			created_at: DateTime
			updated_at: DateTime
			icd10_codes: [String]
			snomed_codes: [String]
			validation: JSON
		}

		type SystemHealth {
			status: String!
			timestamp: DateTime!
			checks: JSON!
		}

		type ContextBuildResponse {
			cacheHit: Boolean!
			processedAt: DateTime!
			phenotypes: [String!]!
		}

		input BuildContextInput {
			patientId: ID!
			patient: JSON!
			transactionId: String
		}

		type Query {
			_entities(representations: [JSON!]!): [Patient]!
			_service: _Service!
			systemHealth: SystemHealth!
			phenotypeDefinitions(domain: String, status: String, limit: Int, offset: Int): [PhenotypeDefinition!]!
		}

		type Mutation {
			buildContext(input: BuildContextInput!): ContextBuildResponse!
		}

		type _Service {
			sdl: String
		}
	`
}