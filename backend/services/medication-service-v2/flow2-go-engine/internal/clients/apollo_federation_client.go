package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
	"github.com/sirupsen/logrus"
)

// ApolloFederationClient provides access to the Apollo Federation gateway
// for Phase 1 knowledge base access as specified in the GO Orchestrator documentation
type ApolloFederationClient struct {
	endpoint     string
	httpClient   *http.Client
	authToken    string
	
	// Circuit breaker for resilience
	breaker      *gobreaker.CircuitBreaker
	
	// Metrics and logging
	logger       *logrus.Logger
	metrics      *ApolloMetrics
}

// ApolloMetrics tracks Apollo Federation client metrics
type ApolloMetrics struct {
	TotalRequests     int64
	SuccessfulQueries int64
	FailedQueries     int64
	CircuitBreakerTrips int64
	AverageLatencyMs  float64
}

// GraphQLRequest represents a GraphQL query request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	OperationName string             `json:"operationName,omitempty"`
}

// GraphQLResponse represents a GraphQL query response
type GraphQLResponse struct {
	Data   interface{}            `json:"data"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message   string                 `json:"message"`
	Locations []GraphQLLocation      `json:"locations,omitempty"`
	Path      []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLLocation represents error location
type GraphQLLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// ApolloConfig holds configuration for Apollo Federation client
type ApolloConfig struct {
	Endpoint           string
	AuthToken          string
	TimeoutSeconds     int
	MaxRetries         int
	CircuitBreakerSettings *CircuitBreakerSettings
}

// CircuitBreakerSettings holds circuit breaker configuration
type CircuitBreakerSettings struct {
	MaxRequests uint32
	Interval    time.Duration
	Timeout     time.Duration
}

// NewApolloFederationClient creates a new Apollo Federation client
func NewApolloFederationClient(config *ApolloConfig, logger *logrus.Logger) *ApolloFederationClient {
	// Default configuration
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 5 // Aggressive timeout for Phase 1
	}
	
	if config.CircuitBreakerSettings == nil {
		config.CircuitBreakerSettings = &CircuitBreakerSettings{
			MaxRequests: 3,
			Interval:    10 * time.Second,
			Timeout:     30 * time.Second,
		}
	}
	
	httpClient := &http.Client{
		Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	
	client := &ApolloFederationClient{
		endpoint:   config.Endpoint,
		httpClient: httpClient,
		authToken:  config.AuthToken,
		logger:     logger,
		metrics:    &ApolloMetrics{},
	}
	
	// Configure circuit breaker
	client.breaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "apollo-federation",
		MaxRequests: config.CircuitBreakerSettings.MaxRequests,
		Interval:    config.CircuitBreakerSettings.Interval,
		Timeout:     config.CircuitBreakerSettings.Timeout,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			client.logger.WithFields(logrus.Fields{
				"name": name,
				"from": from,
				"to":   to,
			}).Info("Apollo Federation circuit breaker state change")
			
			if to == gobreaker.StateOpen {
				client.metrics.CircuitBreakerTrips++
			}
		},
	})
	
	return client
}

// Query executes a GraphQL query through the circuit breaker
func (c *ApolloFederationClient) Query(
	ctx context.Context,
	query string,
	variables map[string]interface{},
) (interface{}, error) {
	startTime := time.Now()
	
	// Execute with circuit breaker
	result, err := c.breaker.Execute(func() (interface{}, error) {
		return c.executeQuery(ctx, query, variables)
	})
	
	// Track metrics
	c.updateMetrics(time.Since(startTime), err == nil)
	
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"query": query,
			"variables": variables,
			"error": err.Error(),
			"duration_ms": time.Since(startTime).Milliseconds(),
		}).Error("Apollo Federation query failed")
		
		// Check if we should fail open (use cached/default values)
		if c.shouldFailOpen(err) {
			return c.getDefaultResponse(query), nil
		}
		return nil, err
	}
	
	c.logger.WithFields(logrus.Fields{
		"query": query,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Debug("Apollo Federation query succeeded")
	
	return result, nil
}

// executeQuery performs the actual GraphQL query
func (c *ApolloFederationClient) executeQuery(
	ctx context.Context,
	query string,
	variables map[string]interface{},
) (interface{}, error) {
	// Prepare GraphQL request
	gqlRequest := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}
	
	requestBody, err := json.Marshal(gqlRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	
	// Execute HTTP request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GraphQL request failed with status %d: %s", resp.StatusCode, string(responseBody))
	}
	
	// Parse GraphQL response
	var gqlResponse GraphQLResponse
	if err := json.Unmarshal(responseBody, &gqlResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal GraphQL response: %w", err)
	}
	
	// Check for GraphQL errors
	if len(gqlResponse.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL query returned errors: %+v", gqlResponse.Errors)
	}
	
	return gqlResponse.Data, nil
}

// Phase 1 Specific Query Methods

// LoadORBRules loads ORB rules from Apollo Federation as specified in Phase 1
func (c *ApolloFederationClient) LoadORBRules(ctx context.Context) (interface{}, error) {
	query := `
		query LoadORBRules {
			kb_guideline_evidence {
				orbRules {
					ruleId
					priority
					conditions {
						allOf {
							fact
							operator
							value
						}
						anyOf {
							fact
							operator
							value
						}
					}
					action {
						generateManifest {
							recipeId
							variant
							dataManifest {
								required
							}
							knowledgeManifest {
								requiredKBs
							}
						}
					}
					metadata {
						guidelineRef
						evidenceLevel
						lastUpdated
					}
				}
			}
		}
	`
	
	return c.Query(ctx, query, nil)
}

// LoadContextRecipe loads a context recipe by protocol ID
func (c *ApolloFederationClient) LoadContextRecipe(ctx context.Context, protocolID string) (interface{}, error) {
	query := `
		query GetContextRecipe($protocolId: String!) {
			kb_guideline_evidence {
				contextRecipe(protocolId: $protocolId) {
					id
					version
					coreFields {
						name
						type
						required
						maxAgeHours
						clinicalContext
					}
					conditionalRules {
						condition
						requiredFields {
							name
							type
							required
							maxAgeHours
						}
						rationale
					}
					freshnessRules {
						maxAge
						criticalThreshold
						preferredSources
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"protocolId": protocolID,
	}
	
	return c.Query(ctx, query, variables)
}

// LoadClinicalRecipe loads a clinical recipe by protocol ID
func (c *ApolloFederationClient) LoadClinicalRecipe(ctx context.Context, protocolID string) (interface{}, error) {
	query := `
		query GetClinicalRecipe($protocolId: String!) {
			kb_guideline_evidence {
				clinicalRecipe(protocolId: $protocolId) {
					id
					version
					therapySelectionRules {
						priority
						drugClass
						conditions
						contraindications
						evidenceLevel
					}
					dosingStrategy {
						approach
						adjustmentFactors
						startingDose {
							amount
							unit
							frequency
							route
							rationale
						}
						titrationPlan {
							initialDose {
								amount
								unit
								frequency
								route
							}
							titrationSteps {
								stepNumber
								dose {
									amount
									unit
									frequency
								}
								criteria
								minDuration
							}
							maxDose {
								amount
								unit
								frequency
								route
							}
						}
					}
					safetyChecks {
						checkType
						severity
						mandatory
						parameters
					}
					monitoringPlan {
						required {
							parameter
							frequency
							duration
							thresholdAlerts {
								parameter
								threshold
								direction
								severity
								action
							}
						}
						optional {
							parameter
							frequency
							duration
						}
						duration
						escalationPlan
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"protocolId": protocolID,
	}
	
	return c.Query(ctx, query, variables)
}

// LoadProtocol loads protocol details
func (c *ApolloFederationClient) LoadProtocol(ctx context.Context, protocolID string) (interface{}, error) {
	query := `
		query GetProtocol($protocolId: String!) {
			kb_guideline_evidence {
				protocol(id: $protocolId) {
					id
					version
					name
					category
					evidenceLevel
					guidelineSource
					lastUpdated
					metadata
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"protocolId": protocolID,
	}
	
	return c.Query(ctx, query, variables)
}

// Utility methods

// updateMetrics updates client metrics
func (c *ApolloFederationClient) updateMetrics(duration time.Duration, success bool) {
	c.metrics.TotalRequests++
	
	if success {
		c.metrics.SuccessfulQueries++
	} else {
		c.metrics.FailedQueries++
	}
	
	// Update average latency
	totalLatency := c.metrics.AverageLatencyMs * float64(c.metrics.TotalRequests-1)
	c.metrics.AverageLatencyMs = (totalLatency + float64(duration.Milliseconds())) / float64(c.metrics.TotalRequests)
}

// shouldFailOpen determines if we should fail open on errors
func (c *ApolloFederationClient) shouldFailOpen(err error) bool {
	// Check if error suggests temporary network/service issues
	// In Phase 1, we might want to fail open for non-critical errors
	// This depends on the specific error handling policy
	return false // Conservative approach - don't fail open by default
}

// getDefaultResponse returns default response for fail-open scenarios
func (c *ApolloFederationClient) getDefaultResponse(query string) interface{} {
	// Return empty response structure - this would be customized
	// based on the specific query type
	return map[string]interface{}{
		"kb_guideline_evidence": map[string]interface{}{
			"orbRules": []interface{}{},
		},
	}
}

// HealthCheck performs a health check on the Apollo Federation endpoint
func (c *ApolloFederationClient) HealthCheck(ctx context.Context) error {
	query := `
		query HealthCheck {
			__schema {
				queryType {
					name
				}
			}
		}
	`
	
	_, err := c.Query(ctx, query, nil)
	return err
}

// GetMetrics returns current client metrics
func (c *ApolloFederationClient) GetMetrics() *ApolloMetrics {
	return c.metrics
}

// Close performs cleanup
func (c *ApolloFederationClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}