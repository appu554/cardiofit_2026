package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// GraphQL Request/Response structures
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   json.RawMessage   `json:"data,omitempty"`
	Errors []GraphQLError    `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message    string                 `json:"message"`
	Path       []interface{}         `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// Apollo Federation Client interface
type ApolloFederationClient interface {
	// Knowledge Base Queries
	GetDrugRules(ctx context.Context, drugID string, region *string) (*DrugRules, error)
	CheckDrugInteractions(ctx context.Context, activeMedications []string, candidateDrug string) (*InteractionCheckResponse, error)
	GeneratePatientSafetyProfile(ctx context.Context, patientID string, patientData *PatientClinicalData) (*PatientSafetyProfile, error)
	GetClinicalPathway(ctx context.Context, condition string, patientContext *PatientClinicalData) (*ClinicalPathway, error)
	CheckFormularyCoverage(ctx context.Context, drugID, payerID, planID string) (*CoverageResponse, error)
	GetDrugMasterInfo(ctx context.Context, drugID string) (*DrugMasterEntry, error)
	MapTerminologyCodes(ctx context.Context, sourceSystem, sourceCode, targetSystem string) (*TerminologyMapping, error)

	// Batch Operations
	BatchGetDrugRules(ctx context.Context, drugIDs []string, region *string) (map[string]*DrugRules, error)
	BatchCheckInteractions(ctx context.Context, requests []InteractionCheckRequest) ([]InteractionCheckResponse, error)

	// Clinical Intelligence
	EvaluatePatientPhenotypes(ctx context.Context, patients []PatientClinicalData) (*PhenotypeEvaluationResponse, error)
	AssessPatientRisk(ctx context.Context, patientID string, patientData *PatientClinicalData) (*RiskAssessmentResponse, error)
	GetTreatmentPreferences(ctx context.Context, patientID, condition string, patientData *PatientClinicalData) (*TreatmentPreferencesResponse, error)
	AssemblePatientContext(ctx context.Context, patientID string, patientData *PatientClinicalData, detailLevel ContextDetailLevel) (*ClinicalContextResponse, error)

	// Health and Monitoring
	HealthCheck(ctx context.Context) error
	GetMetrics(ctx context.Context) (*FederationMetrics, error)
}

// Apollo Federation Client implementation
type apolloFederationClient struct {
	httpClient  *http.Client
	baseURL     string
	logger      *zap.Logger
	cache       Cache
	metrics     *federationMetrics
	rateLimiter *rateLimiter
	mu          sync.RWMutex
}

// Configuration for Apollo Federation Client
type ApolloFederationConfig struct {
	BaseURL        string
	Timeout        time.Duration
	MaxRetries     int
	RetryDelay     time.Duration
	EnableCache    bool
	CacheTTL       time.Duration
	RateLimitRPS   int
	EnableMetrics  bool
	Logger         *zap.Logger
}

// Prometheus metrics
type federationMetrics struct {
	requestsTotal     *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	cacheHits         *prometheus.CounterVec
	cacheMisses       *prometheus.CounterVec
	errors            *prometheus.CounterVec
	rateLimitHits     prometheus.Counter
	batchRequestSize  *prometheus.HistogramVec
}

// Rate limiter for Apollo Federation requests
type rateLimiter struct {
	tokens chan struct{}
	ticker *time.Ticker
	done   chan bool
}

// Cache interface for Federation responses
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// NewApolloFederationClient creates a new Apollo Federation client
func NewApolloFederationClient(config *ApolloFederationConfig) ApolloFederationClient {
	client := &apolloFederationClient{
		httpClient: &http.Client{
			Timeout: config.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
		baseURL: config.BaseURL,
		logger:  config.Logger,
	}

	// Initialize metrics if enabled
	if config.EnableMetrics {
		client.metrics = initFederationMetrics()
	}

	// Initialize rate limiter
	if config.RateLimitRPS > 0 {
		client.rateLimiter = newRateLimiter(config.RateLimitRPS)
	}

	return client
}

// initFederationMetrics initializes Prometheus metrics
func initFederationMetrics() *federationMetrics {
	return &federationMetrics{
		requestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "apollo_federation_requests_total",
			Help: "Total number of Apollo Federation requests",
		}, []string{"operation", "status"}),

		requestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "apollo_federation_request_duration_seconds",
			Help:    "Duration of Apollo Federation requests",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),

		cacheHits: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "apollo_federation_cache_hits_total",
			Help: "Total number of cache hits",
		}, []string{"operation"}),

		cacheMisses: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "apollo_federation_cache_misses_total",
			Help: "Total number of cache misses",
		}, []string{"operation"}),

		errors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "apollo_federation_errors_total",
			Help: "Total number of Apollo Federation errors",
		}, []string{"operation", "error_type"}),

		rateLimitHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "apollo_federation_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		}),

		batchRequestSize: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "apollo_federation_batch_request_size",
			Help:    "Size of batch requests",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
		}, []string{"operation"}),
	}
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(rps int) *rateLimiter {
	rl := &rateLimiter{
		tokens: make(chan struct{}, rps),
		ticker: time.NewTicker(time.Second / time.Duration(rps)),
		done:   make(chan bool),
	}

	// Fill initial tokens
	for i := 0; i < rps; i++ {
		rl.tokens <- struct{}{}
	}

	// Start token refill goroutine
	go func() {
		for {
			select {
			case <-rl.ticker.C:
				select {
				case rl.tokens <- struct{}{}:
				default:
				}
			case <-rl.done:
				return
			}
		}
	}()

	return rl
}

// GetDrugRules retrieves drug dosing rules from KB-Drug-Rules service
func (c *apolloFederationClient) GetDrugRules(ctx context.Context, drugID string, region *string) (*DrugRules, error) {
	operation := "getDrugRules"
	timer := prometheus.NewTimer(c.metrics.requestDuration.WithLabelValues(operation))
	defer timer.ObserveDuration()

	query := `
		query GetDrugRules($drugId: String!, $region: String) {
			drugRules(drugId: $drugId, region: $region) {
				drugId
				version
				contentSha
				signatureValid
				selectedRegion
				content {
					meta {
						name
						version
						description
						effectiveDate
					}
					doseCalculation {
						standardDose
						maxDailyDose
						renalAdjustment {
							gfrThreshold
							adjustment
						}
						hepaticAdjustment {
							severity
							adjustment
						}
					}
					safetyVerification {
						contraindications
						warnings
						precautions
						monitoring {
							parameter
							frequency
							thresholds
						}
					}
					monitoringRequirements {
						parameter
						frequency
						normalRange
						criticalRange
					}
				}
				cacheControl
				etag
			}
		}
	`

	variables := map[string]interface{}{
		"drugId": drugID,
	}
	if region != nil {
		variables["region"] = *region
	}

	var response struct {
		DrugRules *DrugRules `json:"drugRules"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to get drug rules: %w", err)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.DrugRules, nil
}

// CheckDrugInteractions checks for drug-drug interactions
func (c *apolloFederationClient) CheckDrugInteractions(ctx context.Context, activeMedications []string, candidateDrug string) (*InteractionCheckResponse, error) {
	operation := "checkDrugInteractions"
	timer := prometheus.NewTimer(c.metrics.requestDuration.WithLabelValues(operation))
	defer timer.ObserveDuration()

	query := `
		query CheckDrugInteractions($activeMedications: [String!]!, $candidateDrug: String!) {
			checkDrugInteractions(activeMedications: $activeMedications, candidateDrug: $candidateDrug) {
				candidateDrug
				interactions {
					substrate
					perpetrator
					severity
					mechanism
					clinicalEffect
					management {
						action
						doseAdjustment {
							type
							factor
						}
						monitoring {
							parameter
							frequency
						}
						alternatives
					}
					evidenceLevel
					onset
					probability
				}
				overallAction
				clinicalSummary
				processingTime
				slaCompliant
			}
		}
	`

	variables := map[string]interface{}{
		"activeMedications": activeMedications,
		"candidateDrug":     candidateDrug,
	}

	var response struct {
		CheckDrugInteractions *InteractionCheckResponse `json:"checkDrugInteractions"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to check drug interactions: %w", err)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.CheckDrugInteractions, nil
}

// GeneratePatientSafetyProfile generates comprehensive patient safety profile
func (c *apolloFederationClient) GeneratePatientSafetyProfile(ctx context.Context, patientID string, patientData *PatientClinicalData) (*PatientSafetyProfile, error) {
	operation := "generatePatientSafetyProfile"
	timer := prometheus.NewTimer(c.metrics.requestDuration.WithLabelValues(operation))
	defer timer.ObserveDuration()

	query := `
		query GeneratePatientSafetyProfile($patientId: String!, $patientData: PatientClinicalDataInput!) {
			generatePatientSafetyProfile(patientId: $patientId, patientData: $patientData) {
				patientId
				safetyFlags {
					flagType
					value
					confidence
					source
					lastVerified
				}
				contraindicationCodes {
					code
					system
					description
					severity
				}
				riskScores {
					category
					score
					percentile
					level
				}
				phenotypes {
					name
					category
					matched
					confidence
				}
				generatedAt
			}
		}
	`

	variables := map[string]interface{}{
		"patientId":   patientID,
		"patientData": patientData,
	}

	var response struct {
		GeneratePatientSafetyProfile *PatientSafetyProfile `json:"generatePatientSafetyProfile"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to generate patient safety profile: %w", err)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.GeneratePatientSafetyProfile, nil
}

// EvaluatePatientPhenotypes evaluates clinical phenotypes using CEL engine
func (c *apolloFederationClient) EvaluatePatientPhenotypes(ctx context.Context, patients []PatientClinicalData) (*PhenotypeEvaluationResponse, error) {
	operation := "evaluatePatientPhenotypes"
	timer := prometheus.NewTimer(c.metrics.requestDuration.WithLabelValues(operation))
	defer timer.ObserveDuration()

	c.metrics.batchRequestSize.WithLabelValues(operation).Observe(float64(len(patients)))

	query := `
		query EvaluatePatientPhenotypes($input: PhenotypeEvaluationInput!) {
			evaluatePatientPhenotypes(input: $input) {
				results {
					patientId
					phenotypes {
						id
						name
						category
						domain
						priority
						matched
						confidence
						celRule
						implications {
							type
							severity
							description
							recommendations
							clinicalEvidence
						}
						evaluationDetails {
							evaluationPath
							factorsConsidered {
								name
								value
								weight
								contribution
							}
							celExpression
							executionTime
						}
						lastEvaluated
					}
					evaluationSummary {
						totalPhenotypes
						matchedPhenotypes
						highConfidenceMatches
						averageConfidence
						processingTime
					}
				}
				processingTime
				batchSize
				slaCompliant
				metadata {
					cacheHitRate
					averageProcessingTime
					componentsProcessed
					errorCount
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"patients":             patients,
			"includeExplanation":   true,
			"includeImplications":  true,
			"confidenceThreshold":  0.7,
		},
	}

	var response struct {
		EvaluatePatientPhenotypes *PhenotypeEvaluationResponse `json:"evaluatePatientPhenotypes"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to evaluate patient phenotypes: %w", err)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.EvaluatePatientPhenotypes, nil
}

// AssessPatientRisk performs comprehensive patient risk assessment
func (c *apolloFederationClient) AssessPatientRisk(ctx context.Context, patientID string, patientData *PatientClinicalData) (*RiskAssessmentResponse, error) {
	operation := "assessPatientRisk"
	timer := prometheus.NewTimer(c.metrics.requestDuration.WithLabelValues(operation))
	defer timer.ObserveDuration()

	query := `
		query AssessPatientRisk($input: RiskAssessmentInput!) {
			assessPatientRisk(input: $input) {
				patientId
				riskAssessments {
					id
					model
					category
					score
					percentile
					categoryResult: category_result
					recommendations {
						priority
						action
						rationale
						urgency
						clinicalEvidence
					}
					riskFactors {
						name
						value
						contribution
						modifiable
						severity
					}
					calculationMethod
					validUntil
					lastCalculated
				}
				overallRiskProfile {
					overallRisk
					primaryConcerns
					riskDistribution {
						category
						score
						level
						trend
					}
					recommendedActions
				}
				processingTime
				slaCompliant
			}
		}
	`

	variables := map[string]interface{}{
		"input": map[string]interface{}{
			"patientId":             patientID,
			"patientData":          patientData,
			"riskCategories":       []string{"CARDIOVASCULAR", "DIABETES", "MEDICATION", "FALL", "BLEEDING"},
			"includeFactors":       true,
			"includeRecommendations": true,
		},
	}

	var response struct {
		AssessPatientRisk *RiskAssessmentResponse `json:"assessPatientRisk"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to assess patient risk: %w", err)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.AssessPatientRisk, nil
}

// executeQuery executes a GraphQL query with caching and error handling
func (c *apolloFederationClient) executeQuery(ctx context.Context, operation, query string, variables map[string]interface{}, result interface{}) error {
	// Check rate limiter
	if c.rateLimiter != nil {
		select {
		case <-c.rateLimiter.tokens:
			// Token acquired
		case <-ctx.Done():
			return ctx.Err()
		default:
			c.metrics.rateLimitHits.Inc()
			return fmt.Errorf("rate limit exceeded for operation %s", operation)
		}
	}

	// Check cache first
	cacheKey := c.generateCacheKey(operation, variables)
	if c.cache != nil {
		if cached, err := c.cache.Get(ctx, cacheKey); err == nil {
			c.metrics.cacheHits.WithLabelValues(operation).Inc()
			return json.Unmarshal(cached, result)
		}
		c.metrics.cacheMisses.WithLabelValues(operation).Inc()
	}

	// Prepare GraphQL request
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/graphql", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse GraphQL response
	var gqlResp GraphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("failed to parse GraphQL response: %w", err)
	}

	// Check for GraphQL errors
	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %v", gqlResp.Errors)
	}

	// Parse result
	if err := json.Unmarshal(gqlResp.Data, result); err != nil {
		return fmt.Errorf("failed to parse result: %w", err)
	}

	// Cache successful response
	if c.cache != nil {
		if cachedData, err := json.Marshal(result); err == nil {
			c.cache.Set(ctx, cacheKey, cachedData, 5*time.Minute)
		}
	}

	return nil
}

// generateCacheKey generates a cache key for the request
func (c *apolloFederationClient) generateCacheKey(operation string, variables map[string]interface{}) string {
	data, _ := json.Marshal(map[string]interface{}{
		"operation":  operation,
		"variables":  variables,
	})
	return fmt.Sprintf("apollo_federation:%s", string(data))
}

// HealthCheck performs a health check on the Apollo Federation Gateway
func (c *apolloFederationClient) HealthCheck(ctx context.Context) error {
	query := `
		query HealthCheck {
			__schema {
				queryType {
					name
				}
			}
		}
	`

	var result map[string]interface{}
	return c.executeQuery(ctx, "healthCheck", query, nil, &result)
}

// GetMetrics retrieves federation metrics
func (c *apolloFederationClient) GetMetrics(ctx context.Context) (*FederationMetrics, error) {
	// Implementation would query federation-specific metrics
	// This is a placeholder for the actual implementation
	return &FederationMetrics{
		RequestsTotal:      0,
		CacheHitRate:      0.95,
		AverageLatencyMs:  45,
		ErrorRate:         0.001,
		SLACompliance:     0.998,
	}, nil
}

// Additional batch operations and utility methods would be implemented here...

// BatchGetDrugRules retrieves drug rules for multiple drugs efficiently
func (c *apolloFederationClient) BatchGetDrugRules(ctx context.Context, drugIDs []string, region *string) (map[string]*DrugRules, error) {
	operation := "batchGetDrugRules"
	timer := prometheus.NewTimer(c.metrics.requestDuration.WithLabelValues(operation))
	defer timer.ObserveDuration()

	c.metrics.batchRequestSize.WithLabelValues(operation).Observe(float64(len(drugIDs)))

	query := `
		query BatchGetDrugRules($drugIds: [String!]!, $region: String) {
			batchDrugRules(drugIds: $drugIds, region: $region) {
				drugId
				rules {
					drugId
					version
					contentSha
					signatureValid
					selectedRegion
					content {
						meta {
							name
							version
							description
							effectiveDate
						}
						doseCalculation {
							standardDose
							maxDailyDose
							renalAdjustment {
								gfrThreshold
								adjustment
							}
							hepaticAdjustment {
								severity
								adjustment
							}
						}
					}
				}
				error
			}
		}
	`

	variables := map[string]interface{}{
		"drugIds": drugIDs,
	}
	if region != nil {
		variables["region"] = *region
	}

	var response struct {
		BatchDrugRules []struct {
			DrugID string     `json:"drugId"`
			Rules  *DrugRules `json:"rules"`
			Error  *string    `json:"error"`
		} `json:"batchDrugRules"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to batch get drug rules: %w", err)
	}

	// Convert response to map
	result := make(map[string]*DrugRules)
	for _, item := range response.BatchDrugRules {
		if item.Rules != nil {
			result[item.DrugID] = item.Rules
		}
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return result, nil
}

// AssemblePatientContext assembles comprehensive patient context for clinical decision making
func (c *apolloFederationClient) AssemblePatientContext(ctx context.Context, patientID string, patientData *PatientClinicalData, detailLevel ContextDetailLevel) (*ClinicalContextResponse, error) {
	operation := "assemblePatientContext"
	c.metrics.requestsTotal.WithLabelValues(operation, "attempt").Inc()

	query := `
		query AssemblePatientContext($patientId: ID!, $patientData: PatientClinicalDataInput!, $detailLevel: ContextDetailLevel!) {
			assemblePatientContext(patientId: $patientId, patientData: $patientData, detailLevel: $detailLevel) {
				patientId
				contextType
				detailLevel
				createdAt
				clinicalSummary {
					demographics {
						age
						gender
						ethnicity
					}
					conditions
					medications
					allergies
					vitalSigns
					labResults
					riskFactors
					socialDeterminants
				}
				assessments {
					riskScore
					frailtyIndex
					cognitiveStatus
					functionalStatus
				}
				recommendations {
					level
					category
					description
					evidence
					urgency
				}
				contextMetadata {
					version
					completeness
					lastUpdated
					sources
				}
			}
		}
	`

	variables := map[string]interface{}{
		"patientId":   patientID,
		"patientData": patientData,
		"detailLevel": detailLevel,
	}

	var response struct {
		AssemblePatientContext *ClinicalContextResponse `json:"assemblePatientContext"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to assemble patient context: %w", err)
	}

	if response.AssemblePatientContext == nil {
		c.metrics.errors.WithLabelValues(operation, "null_response").Inc()
		return nil, fmt.Errorf("no patient context returned for patient %s", patientID)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.AssemblePatientContext, nil
}

// BatchCheckInteractions performs batch interaction checks for multiple drug combinations
func (c *apolloFederationClient) BatchCheckInteractions(ctx context.Context, requests []InteractionCheckRequest) ([]InteractionCheckResponse, error) {
	operation := "batchCheckInteractions"
	c.metrics.requestsTotal.WithLabelValues(operation, "attempt").Inc()

	query := `
		query BatchCheckInteractions($requests: [InteractionCheckRequestInput!]!) {
			batchCheckInteractions(requests: $requests) {
				activeMedications
				candidateDrug
				severityLevel
				interactions {
					drug1
					drug2
					severity
					mechanism
					clinicalImplication
					management
					evidence
				}
				contraindications {
					type
					description
					severity
				}
				warnings
				recommendations
				lastUpdated
			}
		}
	`

	variables := map[string]interface{}{
		"requests": requests,
	}

	var response struct {
		BatchCheckInteractions []InteractionCheckResponse `json:"batchCheckInteractions"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to batch check interactions: %w", err)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.BatchCheckInteractions, nil
}

// CheckFormularyCoverage checks if a drug is covered under a specific payer plan
func (c *apolloFederationClient) CheckFormularyCoverage(ctx context.Context, drugID, payerID, planID string) (*CoverageResponse, error) {
	operation := "checkFormularyCoverage"
	c.metrics.requestsTotal.WithLabelValues(operation, "attempt").Inc()

	query := `
		query CheckFormularyCoverage($drugId: ID!, $payerId: ID!, $planId: ID!) {
			checkFormularyCoverage(drugId: $drugId, payerId: $payerId, planId: $planId) {
				drugId
				payerId
				planId
				covered
				tier
				restrictions {
					type
					description
					requirements
				}
				copayAmount
				copayPercentage
				priorAuthRequired
				stepTherapyRequired
				quantityLimits {
					amount
					period
				}
				lastUpdated
				effectiveDate
				expirationDate
			}
		}
	`

	variables := map[string]interface{}{
		"drugId":  drugID,
		"payerId": payerID,
		"planId":  planID,
	}

	var response struct {
		CheckFormularyCoverage *CoverageResponse `json:"checkFormularyCoverage"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to check formulary coverage: %w", err)
	}

	if response.CheckFormularyCoverage == nil {
		c.metrics.errors.WithLabelValues(operation, "null_response").Inc()
		return nil, fmt.Errorf("no coverage information returned for drug %s", drugID)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.CheckFormularyCoverage, nil
}

// GetClinicalPathway retrieves clinical pathway for a condition with patient context
func (c *apolloFederationClient) GetClinicalPathway(ctx context.Context, condition string, patientContext *PatientClinicalData) (*ClinicalPathway, error) {
	operation := "getClinicalPathway"
	c.metrics.requestsTotal.WithLabelValues(operation, "attempt").Inc()

	query := `
		query GetClinicalPathway($condition: String!, $patientContext: PatientClinicalDataInput!) {
			getClinicalPathway(condition: $condition, patientContext: $patientContext) {
				pathwayId
				condition
				version
				steps {
					stepId
					title
					description
					type
					requirements
					outcomes
					timeframe
					alternatives
				}
				eligibilityCriteria
				contraindications
				expectedOutcomes
				qualityMeasures
				evidenceLevel
				lastUpdated
				approvalStatus
			}
		}
	`

	variables := map[string]interface{}{
		"condition":      condition,
		"patientContext": patientContext,
	}

	var response struct {
		GetClinicalPathway *ClinicalPathway `json:"getClinicalPathway"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to get clinical pathway: %w", err)
	}

	if response.GetClinicalPathway == nil {
		c.metrics.errors.WithLabelValues(operation, "null_response").Inc()
		return nil, fmt.Errorf("no clinical pathway returned for condition %s", condition)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.GetClinicalPathway, nil
}

// GetDrugMasterInfo retrieves comprehensive drug master data
func (c *apolloFederationClient) GetDrugMasterInfo(ctx context.Context, drugID string) (*DrugMasterEntry, error) {
	operation := "getDrugMasterInfo"
	c.metrics.requestsTotal.WithLabelValues(operation, "attempt").Inc()

	query := `
		query GetDrugMasterInfo($drugId: ID!) {
			getDrugMasterInfo(drugId: $drugId) {
				drugId
				genericName
				brandNames
				activeIngredients {
					name
					strength
					unit
				}
				dosageForms
				routesOfAdministration
				therapeuticClasses
				pharmacologicClasses
				indications
				contraindications
				warnings
				interactions
				pediatricDosing
				geriatricDosing
				renalAdjustment
				hepaticAdjustment
				pregnancy
				lactation
				lastUpdated
				status
			}
		}
	`

	variables := map[string]interface{}{
		"drugId": drugID,
	}

	var response struct {
		GetDrugMasterInfo *DrugMasterEntry `json:"getDrugMasterInfo"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to get drug master info: %w", err)
	}

	if response.GetDrugMasterInfo == nil {
		c.metrics.errors.WithLabelValues(operation, "null_response").Inc()
		return nil, fmt.Errorf("no drug master info returned for drug %s", drugID)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.GetDrugMasterInfo, nil
}

// GetTreatmentPreferences retrieves treatment preferences for a patient and condition
func (c *apolloFederationClient) GetTreatmentPreferences(ctx context.Context, patientID, condition string, patientData *PatientClinicalData) (*TreatmentPreferencesResponse, error) {
	operation := "getTreatmentPreferences"
	c.metrics.requestsTotal.WithLabelValues(operation, "attempt").Inc()

	query := `
		query GetTreatmentPreferences($patientId: ID!, $condition: String!, $patientData: PatientClinicalDataInput!) {
			getTreatmentPreferences(patientId: $patientId, condition: $condition, patientData: $patientData) {
				patientId
				condition
				preferences {
					category
					preference
					strength
					rationale
				}
				constraints {
					type
					description
					severity
				}
				goals
				prioritization
				sharedDecisionMaking
				culturalConsiderations
				languagePreferences
				lastUpdated
				version
			}
		}
	`

	variables := map[string]interface{}{
		"patientId":   patientID,
		"condition":   condition,
		"patientData": patientData,
	}

	var response struct {
		GetTreatmentPreferences *TreatmentPreferencesResponse `json:"getTreatmentPreferences"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to get treatment preferences: %w", err)
	}

	if response.GetTreatmentPreferences == nil {
		c.metrics.errors.WithLabelValues(operation, "null_response").Inc()
		return nil, fmt.Errorf("no treatment preferences returned for patient %s", patientID)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.GetTreatmentPreferences, nil
}

// MapTerminologyCodes maps codes between different medical terminology systems
func (c *apolloFederationClient) MapTerminologyCodes(ctx context.Context, sourceSystem, sourceCode, targetSystem string) (*TerminologyMapping, error) {
	operation := "mapTerminologyCodes"
	c.metrics.requestsTotal.WithLabelValues(operation, "attempt").Inc()

	query := `
		query MapTerminologyCodes($sourceSystem: String!, $sourceCode: String!, $targetSystem: String!) {
			mapTerminologyCodes(sourceSystem: $sourceSystem, sourceCode: $sourceCode, targetSystem: $targetSystem) {
				sourceSystem
				sourceCode
				sourceDescription
				targetSystem
				mappings {
					code
					description
					equivalence
					confidence
				}
				version
				lastUpdated
				status
			}
		}
	`

	variables := map[string]interface{}{
		"sourceSystem": sourceSystem,
		"sourceCode":   sourceCode,
		"targetSystem": targetSystem,
	}

	var response struct {
		MapTerminologyCodes *TerminologyMapping `json:"mapTerminologyCodes"`
	}

	err := c.executeQuery(ctx, operation, query, variables, &response)
	if err != nil {
		c.metrics.errors.WithLabelValues(operation, "query_error").Inc()
		return nil, fmt.Errorf("failed to map terminology codes: %w", err)
	}

	if response.MapTerminologyCodes == nil {
		c.metrics.errors.WithLabelValues(operation, "null_response").Inc()
		return nil, fmt.Errorf("no terminology mapping returned for %s:%s", sourceSystem, sourceCode)
	}

	c.metrics.requestsTotal.WithLabelValues(operation, "success").Inc()
	return response.MapTerminologyCodes, nil
}