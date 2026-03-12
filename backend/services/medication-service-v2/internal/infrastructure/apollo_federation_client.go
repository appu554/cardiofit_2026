package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/hasura/go-graphql-client"
	"go.uber.org/zap"
)

// ApolloFederationClient provides GraphQL client for Apollo Federation gateway
type ApolloFederationClient struct {
	client         *graphql.Client
	logger         *zap.Logger
	gatewayURL     string
	timeout        time.Duration
	maxRetries     int
	retryDelay     time.Duration
}

// NewApolloFederationClient creates a new Apollo Federation client
func NewApolloFederationClient(gatewayURL string, timeout time.Duration, logger *zap.Logger) *ApolloFederationClient {
	client := graphql.NewClient(gatewayURL, nil)
	
	return &ApolloFederationClient{
		client:         client,
		logger:         logger,
		gatewayURL:     gatewayURL,
		timeout:        timeout,
		maxRetries:     3,
		retryDelay:     time.Second,
	}
}

// DosingRule represents a medication dosing rule from KB1
type DosingRule struct {
	DrugCode               string                 `graphql:"drugCode"`
	Version                string                 `graphql:"version"`
	DrugName               string                 `graphql:"drugName"`
	ContentSHA             string                 `graphql:"contentSHA"`
	SignatureValid         bool                   `graphql:"signatureValid"`
	Active                 bool                   `graphql:"active"`
	Regions                []string               `graphql:"regions"`
	BaseDose               BaseDose               `graphql:"baseDose"`
	Adjustments            []DoseAdjustment       `graphql:"adjustments"`
	TitrationSchedule      []TitrationStep        `graphql:"titrationSchedule"`
	PopulationRules        []PopulationRule       `graphql:"populationRules"`
	SafetyVerification     SafetyVerification     `graphql:"safetyVerification"`
	CreatedAt              string                 `graphql:"createdAt"`
	CreatedBy              string                 `graphql:"createdBy"`
	Provenance             ProvenanceInfo         `graphql:"provenance"`
}

type BaseDose struct {
	Unit        string   `graphql:"unit"`
	Starting    float64  `graphql:"starting"`
	MaxDaily    float64  `graphql:"maxDaily"`
	MinDaily    float64  `graphql:"minDaily"`
	Frequency   string   `graphql:"frequency"`
	Loading     *string  `graphql:"loading"`
	Maintenance *string  `graphql:"maintenance"`
	MaxSingle   *float64 `graphql:"maxSingle"`
}

type DoseAdjustment struct {
	AdjustmentID        string   `graphql:"adjustmentId"`
	AdjustType          string   `graphql:"adjustType"`
	Description         string   `graphql:"description"`
	ConditionExpression string   `graphql:"conditionExpression"`
	FormulaExpression   string   `graphql:"formulaExpression"`
	Multiplier          *float64 `graphql:"multiplier"`
	AdditiveMg          *float64 `graphql:"additiveMg"`
	MaxDoseMg           *float64 `graphql:"maxDoseMg"`
	MinDoseMg           *float64 `graphql:"minDoseMg"`
	Contraindicated     bool     `graphql:"contraindicated"`
}

type TitrationStep struct {
	StepNumber          int32    `graphql:"stepNumber"`
	AfterDays           int32    `graphql:"afterDays"`
	ActionType          string   `graphql:"actionType"`
	ActionValue         *float64 `graphql:"actionValue"`
	MaxStep             *int32   `graphql:"maxStep"`
	MonitoringRequired  *string  `graphql:"monitoringRequired"`
}

type PopulationRule struct {
	PopulationID    string   `graphql:"populationId"`
	PopulationType  string   `graphql:"populationType"`
	AgeMin          *int32   `graphql:"ageMin"`
	AgeMax          *int32   `graphql:"ageMax"`
	WeightMin       *float64 `graphql:"weightMin"`
	WeightMax       *float64 `graphql:"weightMax"`
	Formula         string   `graphql:"formula"`
	SafetyLimits    *string  `graphql:"safetyLimits"`
	Contraindicated bool     `graphql:"contraindicated"`
}

type SafetyVerification struct {
	Contraindications []string `graphql:"contraindications"`
	Warnings          []string `graphql:"warnings"`
	Precautions       []string `graphql:"precautions"`
	LabMonitoring     []string `graphql:"labMonitoring"`
}

type ProvenanceInfo struct {
	Authors          []string `graphql:"authors"`
	Approvals        []string `graphql:"approvals"`
	KB3Refs          []string `graphql:"kb3Refs"`
	KB4Refs          []string `graphql:"kb4Refs"`
	SourceFile       string   `graphql:"sourceFile"`
	EffectiveFrom    *string  `graphql:"effectiveFrom"`
	LastModifiedBy   *string  `graphql:"lastModifiedBy"`
}

// DosingRecommendation represents calculated dosing recommendation
type DosingRecommendation struct {
	DrugCode                string              `graphql:"drugCode"`
	Version                 string              `graphql:"version"`
	ApplicableAdjustments   []DoseAdjustment    `graphql:"applicableAdjustments"`
	RecommendedDose         RecommendedDose     `graphql:"recommendedDose"`
	SafetyAlerts            []SafetyAlert       `graphql:"safetyAlerts"`
	MonitoringRequirements  []string            `graphql:"monitoringRequirements"`
	CalculationMetadata     CalculationMetadata `graphql:"calculationMetadata"`
}

type RecommendedDose struct {
	AmountMg            float64  `graphql:"amountMg"`
	Frequency           string   `graphql:"frequency"`
	Route               string   `graphql:"route"`
	Duration            *string  `graphql:"duration"`
	SpecialInstructions []string `graphql:"specialInstructions"`
}

type SafetyAlert struct {
	AlertType      string  `graphql:"alertType"`
	Severity       string  `graphql:"severity"`
	Message        string  `graphql:"message"`
	ActionRequired *string `graphql:"actionRequired"`
}

type CalculationMetadata struct {
	BasedOnRuleVersion string   `graphql:"basedOnRuleVersion"`
	AppliedAdjustments []string `graphql:"appliedAdjustments"`
	CalculatedAt       string   `graphql:"calculatedAt"`
	CacheHit           bool     `graphql:"cacheHit"`
	ResponseTimeMs     float64  `graphql:"responseTimeMs"`
}

// Clinical Guidelines from KB3
type ClinicalGuideline struct {
	GuidelineID     string              `graphql:"guidelineId"`
	Title           string              `graphql:"title"`
	Version         string              `graphql:"version"`
	Organization    string              `graphql:"organization"`
	PublicationDate string              `graphql:"publicationDate"`
	Recommendations []Recommendation    `graphql:"recommendations"`
	EvidenceLevel   string              `graphql:"evidenceLevel"`
	Categories      []string            `graphql:"categories"`
	DrugClasses     []string            `graphql:"drugClasses"`
	Conditions      []string            `graphql:"conditions"`
	Metadata        GuidelineMetadata   `graphql:"metadata"`
}

type Recommendation struct {
	RecommendationID string   `graphql:"recommendationId"`
	Text             string   `graphql:"text"`
	Strength         string   `graphql:"strength"`
	Quality          string   `graphql:"quality"`
	Categories       []string `graphql:"categories"`
	Conditions       []string `graphql:"conditions"`
	Evidence         []string `graphql:"evidence"`
}

type GuidelineMetadata struct {
	Authors       []string `graphql:"authors"`
	Reviewers     []string `graphql:"reviewers"`
	DOI           *string  `graphql:"doi"`
	PubMedID      *string  `graphql:"pubmedId"`
	LastUpdated   string   `graphql:"lastUpdated"`
	NextReview    *string  `graphql:"nextReview"`
	Tags          []string `graphql:"tags"`
	SourceURL     string   `graphql:"sourceUrl"`
}

// Patient Context Input for calculations
type PatientContextInput struct {
	WeightKg              float64            `graphql:"weightKg"`
	EGFR                  float64            `graphql:"egfr"`
	AgeYears              int32              `graphql:"ageYears"`
	Sex                   string             `graphql:"sex"`
	Pregnant              *bool              `graphql:"pregnant"`
	CreatinineClearance   *float64           `graphql:"creatinineCleatance"`
	DialysisType          *string            `graphql:"dialysisType"`
	ExtraNumeric          []ExtraNumericInput `graphql:"extraNumeric"`
}

type ExtraNumericInput struct {
	Key   string  `graphql:"key"`
	Value float64 `graphql:"value"`
}

// GetDosingRule retrieves a specific dosing rule from KB1
func (c *ApolloFederationClient) GetDosingRule(ctx context.Context, drugCode string, version *string, region *string) (*DosingRule, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var query struct {
		DosingRule *DosingRule `graphql:"dosingRule(drugCode: $drugCode, version: $version, region: $region)"`
	}

	variables := map[string]interface{}{
		"drugCode": graphql.String(drugCode),
		"version":  (*graphql.String)(version),
		"region":   (*graphql.String)(region),
	}

	c.logger.Info("Querying dosing rule",
		zap.String("drug_code", drugCode),
		zap.Any("version", version),
		zap.Any("region", region),
	)

	err := c.executeWithRetry(ctx, &query, variables)
	if err != nil {
		c.logger.Error("Failed to query dosing rule",
			zap.Error(err),
			zap.String("drug_code", drugCode),
		)
		return nil, fmt.Errorf("failed to query dosing rule: %w", err)
	}

	c.logger.Info("Successfully retrieved dosing rule",
		zap.String("drug_code", drugCode),
		zap.Duration("duration", time.Since(start)),
	)

	return query.DosingRule, nil
}

// CalculateDosing gets dosing recommendation for patient context
func (c *ApolloFederationClient) CalculateDosing(ctx context.Context, drugCode string, patientContext PatientContextInput, version *string, region *string) (*DosingRecommendation, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var query struct {
		CalculateDosing *DosingRecommendation `graphql:"calculateDosing(drugCode: $drugCode, patientContext: $patientContext, version: $version, region: $region)"`
	}

	variables := map[string]interface{}{
		"drugCode":       graphql.String(drugCode),
		"patientContext": patientContext,
		"version":        (*graphql.String)(version),
		"region":         (*graphql.String)(region),
	}

	c.logger.Info("Calculating dosing recommendation",
		zap.String("drug_code", drugCode),
		zap.Float64("patient_weight", patientContext.WeightKg),
		zap.Int32("patient_age", patientContext.AgeYears),
	)

	err := c.executeWithRetry(ctx, &query, variables)
	if err != nil {
		c.logger.Error("Failed to calculate dosing",
			zap.Error(err),
			zap.String("drug_code", drugCode),
		)
		return nil, fmt.Errorf("failed to calculate dosing: %w", err)
	}

	c.logger.Info("Successfully calculated dosing recommendation",
		zap.String("drug_code", drugCode),
		zap.Duration("duration", time.Since(start)),
	)

	return query.CalculateDosing, nil
}

// GetClinicalGuidelines retrieves guidelines from KB3
func (c *ApolloFederationClient) GetClinicalGuidelines(ctx context.Context, drugClass *string, condition *string, limit *int32) ([]ClinicalGuideline, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var query struct {
		ClinicalGuidelines []ClinicalGuideline `graphql:"clinicalGuidelines(drugClass: $drugClass, condition: $condition, first: $limit)"`
	}

	variables := map[string]interface{}{
		"drugClass": (*graphql.String)(drugClass),
		"condition": (*graphql.String)(condition),
		"limit":     (*graphql.Int)(limit),
	}

	c.logger.Info("Querying clinical guidelines",
		zap.Any("drug_class", drugClass),
		zap.Any("condition", condition),
		zap.Any("limit", limit),
	)

	err := c.executeWithRetry(ctx, &query, variables)
	if err != nil {
		c.logger.Error("Failed to query clinical guidelines",
			zap.Error(err),
			zap.Any("drug_class", drugClass),
			zap.Any("condition", condition),
		)
		return nil, fmt.Errorf("failed to query clinical guidelines: %w", err)
	}

	c.logger.Info("Successfully retrieved clinical guidelines",
		zap.Int("count", len(query.ClinicalGuidelines)),
		zap.Duration("duration", time.Since(start)),
	)

	return query.ClinicalGuidelines, nil
}

// CheckDosingAvailability checks if dosing rules are available for a drug
func (c *ApolloFederationClient) CheckDosingAvailability(ctx context.Context, drugCode string, region *string) (bool, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var query struct {
		CheckDosingAvailability bool `graphql:"checkDosingAvailability(drugCode: $drugCode, region: $region)"`
	}

	variables := map[string]interface{}{
		"drugCode": graphql.String(drugCode),
		"region":   (*graphql.String)(region),
	}

	err := c.executeWithRetry(ctx, &query, variables)
	if err != nil {
		c.logger.Error("Failed to check dosing availability",
			zap.Error(err),
			zap.String("drug_code", drugCode),
		)
		return false, fmt.Errorf("failed to check dosing availability: %w", err)
	}

	c.logger.Debug("Checked dosing availability",
		zap.String("drug_code", drugCode),
		zap.Bool("available", query.CheckDosingAvailability),
		zap.Duration("duration", time.Since(start)),
	)

	return query.CheckDosingAvailability, nil
}

// HealthCheck performs a health check against the Apollo Federation gateway
func (c *ApolloFederationClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var query struct {
		ServiceInfo struct {
			Name    string `graphql:"name"`
			Version string `graphql:"version"`
		} `graphql:"serviceInfo"`
	}

	// Use the hasura GraphQL client Query method
	err := c.client.Query(ctx, &query, nil)
	if err != nil {
		return fmt.Errorf("federation gateway health check failed: %w", err)
	}

	c.logger.Debug("Federation gateway health check successful")
	return nil
}

// executeWithRetry executes a GraphQL query with retry logic
func (c *ApolloFederationClient) executeWithRetry(ctx context.Context, query interface{}, variables map[string]interface{}) error {
	var lastErr error
	
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
			
			c.logger.Warn("Retrying GraphQL query",
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)
		}

		err := c.client.Query(ctx, query, variables)
		if err == nil {
			return nil
		}

		lastErr = err
		
		// Don't retry context cancellation or certain GraphQL errors
		if ctx.Err() != nil || c.isNonRetryableError(err) {
			break
		}
	}

	return fmt.Errorf("query failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// isNonRetryableError determines if an error should not be retried
func (c *ApolloFederationClient) isNonRetryableError(err error) bool {
	// Add logic to determine non-retryable errors
	// For now, assume most errors are retryable unless they're clearly client errors
	errorStr := err.Error()
	
	// Don't retry validation errors, syntax errors, etc.
	if contains(errorStr, "validation") ||
		contains(errorStr, "syntax") ||
		contains(errorStr, "field") ||
		contains(errorStr, "argument") {
		return true
	}
	
	return false
}

// Helper function to check if string contains substring
func contains(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || 
		(len(str) > len(substr) && (str[:len(substr)] == substr || 
		str[len(str)-len(substr):] == substr || 
		containsMiddle(str, substr))))
}

func containsMiddle(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetBatchDosingRules retrieves multiple dosing rules efficiently
func (c *ApolloFederationClient) GetBatchDosingRules(ctx context.Context, drugCodes []string, region *string) ([]DosingRule, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout*2) // Extended timeout for batch operations
	defer cancel()

	var query struct {
		DosingRules struct {
			Edges []struct {
				Node DosingRule `graphql:"node"`
			} `graphql:"edges"`
		} `graphql:"dosingRules(drugCodes: $drugCodes, regions: $regions, activeOnly: true, first: $limit)"`
	}

	regions := []string{}
	if region != nil {
		regions = append(regions, *region)
	}

	limit := int32(len(drugCodes))
	if limit > 100 {
		limit = 100 // Cap batch size
	}

	variables := map[string]interface{}{
		"drugCodes": drugCodes,
		"regions":   regions,
		"limit":     limit,
	}

	c.logger.Info("Querying batch dosing rules",
		zap.Int("drug_count", len(drugCodes)),
		zap.Any("region", region),
	)

	err := c.executeWithRetry(ctx, &query, variables)
	if err != nil {
		c.logger.Error("Failed to query batch dosing rules",
			zap.Error(err),
			zap.Int("drug_count", len(drugCodes)),
		)
		return nil, fmt.Errorf("failed to query batch dosing rules: %w", err)
	}

	// Extract dosing rules from edges
	rules := make([]DosingRule, len(query.DosingRules.Edges))
	for i, edge := range query.DosingRules.Edges {
		rules[i] = edge.Node
	}

	c.logger.Info("Successfully retrieved batch dosing rules",
		zap.Int("requested_count", len(drugCodes)),
		zap.Int("retrieved_count", len(rules)),
		zap.Duration("duration", time.Since(start)),
	)

	return rules, nil
}

// LogQueryMetrics logs performance metrics for GraphQL queries
func (c *ApolloFederationClient) LogQueryMetrics(ctx context.Context, queryName string, startTime time.Time, err error) {
	duration := time.Since(startTime)
	
	if err != nil {
		c.logger.Error("GraphQL query failed",
			zap.String("query", queryName),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
	} else {
		c.logger.Info("GraphQL query completed",
			zap.String("query", queryName),
			zap.Duration("duration", duration),
		)
	}
}