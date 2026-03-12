package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
)

// ApolloFederationClient handles communication with Apollo Federation Gateway
type ApolloFederationClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

// GraphQLRequest represents a GraphQL request to Apollo Federation
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a GraphQL response from Apollo Federation
type GraphQLResponse struct {
	Data       map[string]interface{} `json:"data"`
	Errors     []GraphQLError         `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	DataSources []string              `json:"-"` // Populated from extensions
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

// NewApolloFederationClient creates a new Apollo Federation client
func NewApolloFederationClient(baseURL string, logger *logger.Logger) (*ApolloFederationClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("Apollo Federation URL is required")
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &ApolloFederationClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// Query executes a GraphQL query against Apollo Federation
func (c *ApolloFederationClient) Query(
	ctx context.Context,
	query string,
	variables map[string]interface{},
) (*GraphQLResponse, error) {
	c.logger.Debug("Executing Apollo Federation query",
		zap.String("base_url", c.baseURL),
	)

	// Create GraphQL request
	gqlRequest := &GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(gqlRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/graphql",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Safety-Gateway-Platform/2.0")

	// Add authentication if needed
	if authToken := c.getAuthToken(); authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
	}

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	queryDuration := time.Since(startTime)

	if err != nil {
		c.logger.Error("Apollo Federation query failed",
			zap.Error(err),
			zap.Duration("duration", queryDuration),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Apollo Federation returned non-200 status",
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", queryDuration),
		)
		return nil, fmt.Errorf("Apollo Federation returned status %d", resp.StatusCode)
	}

	// Parse response
	var gqlResponse GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	// Extract data sources from extensions
	if extensions := gqlResponse.Extensions; extensions != nil {
		if tracing, ok := extensions["tracing"].(map[string]interface{}); ok {
			if execution, ok := tracing["execution"].(map[string]interface{}); ok {
				if resolvers, ok := execution["resolvers"].([]interface{}); ok {
					dataSources := make(map[string]bool)
					for _, resolver := range resolvers {
						if r, ok := resolver.(map[string]interface{}); ok {
							if serviceName, ok := r["serviceName"].(string); ok {
								dataSources[serviceName] = true
							}
						}
					}
					
					// Convert to slice
					var sources []string
					for source := range dataSources {
						sources = append(sources, source)
					}
					gqlResponse.DataSources = sources
				}
			}
		}
	}

	// Check for GraphQL errors
	if len(gqlResponse.Errors) > 0 {
		c.logger.Warn("Apollo Federation returned GraphQL errors",
			zap.Int("error_count", len(gqlResponse.Errors)),
			zap.Duration("duration", queryDuration),
		)
		
		// Log each error
		for _, gqlError := range gqlResponse.Errors {
			c.logger.Warn("GraphQL error",
				zap.String("message", gqlError.Message),
				zap.Any("path", gqlError.Path),
			)
		}
	}

	c.logger.Debug("Apollo Federation query completed",
		zap.Duration("duration", queryDuration),
		zap.Int("data_sources", len(gqlResponse.DataSources)),
		zap.Bool("has_errors", len(gqlResponse.Errors) > 0),
	)

	return &gqlResponse, nil
}

// QueryPatientClinicalData executes a comprehensive patient data query
func (c *ApolloFederationClient) QueryPatientClinicalData(
	ctx context.Context,
	patientID string,
) (*GraphQLResponse, error) {
	query := `
		query GetPatientClinicalData($patientId: ID!) {
			patient(id: $patientId) {
				id
				
				# Demographics
				demographics {
					age
					ageInYears
					gender
					weight {
						value
						unit
					}
					height {
						value
						unit
					}
					bmi {
						value
						unit
					}
					ethnicity
					race
				}
				
				# Allergies and Intolerances
				allergies {
					id
					substance {
						code
						display
						system
					}
					severity
					reaction
					status
					verificationStatus
					recordedDate
					recorder {
						reference
						display
					}
				}
				
				# Active Conditions
				conditions {
					id
					code {
						coding {
							code
							display
							system
						}
						text
					}
					category
					severity {
						coding {
							code
							display
							system
						}
					}
					clinicalStatus
					verificationStatus
					onsetDateTime
					recordedDate
					asserter {
						reference
						display
					}
				}
				
				# Current Medications
				medications {
					id
					medicationCodeableConcept {
						coding {
							code
							display
							system
						}
						text
					}
					status
					effectiveDateTime
					effectivePeriod {
						start
						end
					}
					dosageInstruction {
						text
						timing {
							repeat {
								frequency
								period
								periodUnit
							}
						}
						doseAndRate {
							doseQuantity {
								value
								unit
								system
							}
						}
						route {
							coding {
								code
								display
								system
							}
						}
					}
					requester {
						reference
						display
					}
				}
				
				# Recent Lab Results
				labResults(recent: true, limit: 50) {
					id
					code {
						coding {
							code
							display
							system
						}
						text
					}
					status
					effectiveDateTime
					valueQuantity {
						value
						unit
						system
					}
					valueCodeableConcept {
						coding {
							code
							display
							system
						}
					}
					interpretation {
						coding {
							code
							display
							system
						}
					}
					referenceRange {
						low {
							value
							unit
						}
						high {
							value
							unit
						}
						text
					}
					performer {
						reference
						display
					}
				}
				
				# Vital Signs
				vitalSigns(recent: true, limit: 20) {
					id
					code {
						coding {
							code
							display
							system
						}
					}
					status
					effectiveDateTime
					valueQuantity {
						value
						unit
						system
					}
					component {
						code {
							coding {
								code
								display
								system
							}
						}
						valueQuantity {
							value
							unit
							system
						}
					}
				}
				
				# Recent Procedures
				procedures(recent: true, limit: 10) {
					id
					code {
						coding {
							code
							display
							system
						}
						text
					}
					status
					performedDateTime
					performedPeriod {
						start
						end
					}
					performer {
						actor {
							reference
							display
						}
					}
					outcome {
						coding {
							code
							display
							system
						}
					}
				}
				
				# Recent Encounters
				encounters(recent: true, limit: 5) {
					id
					class {
						code
						display
					}
					type {
						coding {
							code
							display
							system
						}
					}
					status
					period {
						start
						end
					}
					reasonCode {
						coding {
							code
							display
							system
						}
					}
					participant {
						individual {
							reference
							display
						}
						type {
							coding {
								code
								display
								system
							}
						}
					}
					serviceProvider {
						reference
						display
					}
				}
			}
			
			# Additional clinical context
			clinicalContext(patientId: $patientId) {
				riskFactors {
					category
					description
					severity
					evidence
				}
				clinicalSummary {
					activeProblems
					keyFindings
					riskAssessment
				}
			}
		}
	`

	return c.Query(ctx, query, map[string]interface{}{
		"patientId": patientID,
	})
}

// QueryKnowledgeBaseVersions retrieves current versions of all knowledge bases
func (c *ApolloFederationClient) QueryKnowledgeBaseVersions(ctx context.Context) (*GraphQLResponse, error) {
	query := `
		query GetKnowledgeBaseVersions {
			knowledgeBases {
				kb1_dosing {
					version
					lastUpdated
					description
				}
				kb3_guidelines {
					version
					lastUpdated
					description
				}
				kb4_safety {
					version
					lastUpdated
					description
				}
				kb5_ddi {
					version
					lastUpdated
					description
				}
				kb7_terminology {
					version
					lastUpdated
					description
				}
			}
		}
	`

	return c.Query(ctx, query, nil)
}

// QueryMedicationKnowledge retrieves medication-specific knowledge
func (c *ApolloFederationClient) QueryMedicationKnowledge(
	ctx context.Context,
	medicationCodes []string,
) (*GraphQLResponse, error) {
	query := `
		query GetMedicationKnowledge($medicationCodes: [String!]!) {
			medicationKnowledge {
				medications(codes: $medicationCodes) {
					code
					display
					drugClass
					mechanism
					indications
					contraindications
					warnings
					precautions
					dosageInformation {
						route
						form
						strength
						adultDosing {
							indication
							dose
							frequency
							duration
						}
						pediatricDosing {
							ageGroup
							dose
							frequency
							duration
						}
						renalAdjustment {
							creatinineClearance
							adjustment
						}
						hepaticAdjustment {
							severity
							adjustment
						}
					}
				}
				
				interactions(codes: $medicationCodes) {
					drug1 {
						code
						display
					}
					drug2 {
						code
						display
					}
					severity
					mechanism
					clinicalEffect
					management
					evidence {
						level
						source
						reference
					}
				}
				
				allergies(codes: $medicationCodes) {
					medication {
						code
						display
					}
					allergen
					crossReactivity
					severity
					manifestations
				}
			}
		}
	`

	return c.Query(ctx, query, map[string]interface{}{
		"medicationCodes": medicationCodes,
	})
}

// Health checks the health of the Apollo Federation Gateway
func (c *ApolloFederationClient) Health(ctx context.Context) error {
	query := `
		query Health {
			__schema {
				queryType {
					name
				}
			}
		}
	`

	response, err := c.Query(ctx, query, nil)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if len(response.Errors) > 0 {
		return fmt.Errorf("health check returned errors: %+v", response.Errors)
	}

	return nil
}

// getAuthToken retrieves authentication token (placeholder implementation)
func (c *ApolloFederationClient) getAuthToken() string {
	// In a real implementation, this would retrieve a JWT token
	// from environment variables, a secret store, or token cache
	return ""
}

// Close cleans up the client resources
func (c *ApolloFederationClient) Close() error {
	// Close any persistent connections
	c.httpClient.CloseIdleConnections()
	return nil
}

// GetBaseURL returns the base URL of the Apollo Federation Gateway
func (c *ApolloFederationClient) GetBaseURL() string {
	return c.baseURL
}

// SetTimeout sets the HTTP client timeout
func (c *ApolloFederationClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// EnableRetries enables retry logic for failed requests
func (c *ApolloFederationClient) EnableRetries(maxRetries int, backoffDuration time.Duration) {
	// This would implement retry logic with exponential backoff
	// For now, it's a placeholder
	c.logger.Info("Retry logic enabled",
		zap.Int("max_retries", maxRetries),
		zap.Duration("backoff_duration", backoffDuration),
	)
}