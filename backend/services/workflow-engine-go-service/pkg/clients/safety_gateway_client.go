package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// SafetyGatewayClient defines the interface for Safety Gateway communication
type SafetyGatewayClient interface {
	ComprehensiveValidation(ctx context.Context, request *SafetyValidationRequest) (*SafetyValidationResponse, error)
	ValidateProposal(ctx context.Context, request *ProposalValidationRequest) (*ValidationResult, error)
	HealthCheck(ctx context.Context) error
}

// SafetyValidationRequest represents a comprehensive validation request
type SafetyValidationRequest struct {
	ProposalSetID           string                   `json:"proposal_set_id"`
	SnapshotID              string                   `json:"snapshot_id"`
	Proposals               []map[string]interface{} `json:"proposals"`
	PatientContext          map[string]interface{}   `json:"patient_context"`
	ValidationRequirements  map[string]interface{}   `json:"validation_requirements"`
	CorrelationID           string                   `json:"correlation_id"`
	ValidationScope         []string                 `json:"validation_scope,omitempty"`
	RiskTolerance          string                   `json:"risk_tolerance,omitempty"`
}

// ProposalValidationRequest represents a single proposal validation
type ProposalValidationRequest struct {
	ProposalID      string                 `json:"proposal_id"`
	SnapshotID      string                 `json:"snapshot_id"`
	Proposal        map[string]interface{} `json:"proposal"`
	PatientContext  map[string]interface{} `json:"patient_context"`
	ValidationLevel string                 `json:"validation_level"` // "basic", "comprehensive", "critical"
	CorrelationID   string                 `json:"correlation_id"`
}

// SafetyValidationResponse represents the comprehensive validation response
type SafetyValidationResponse struct {
	ValidationID         string                   `json:"validation_id"`
	Verdict              string                   `json:"verdict"` // "SAFE", "WARNING", "UNSAFE", "ERROR"
	OverallRiskScore     float64                  `json:"overall_risk_score"`
	Findings             []ValidationFinding      `json:"findings"`
	OverrideTokens       []string                 `json:"override_tokens,omitempty"`
	OverrideRequirements map[string]interface{}   `json:"override_requirements,omitempty"`
	ValidationMetrics    map[string]interface{}   `json:"validation_metrics"`
	ExecutedEngines      []string                 `json:"executed_engines"`
	Status               string                   `json:"status"`
	Message              string                   `json:"message,omitempty"`
}

// ValidationFinding represents a single validation finding
type ValidationFinding struct {
	FindingID            string                 `json:"finding_id"`
	Severity             string                 `json:"severity"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
	Category             string                 `json:"category"`
	Description          string                 `json:"description"`
	ClinicalSignificance string                 `json:"clinical_significance"`
	Recommendation       string                 `json:"recommendation"`
	ConfidenceScore      float64                `json:"confidence_score"`
	Source               string                 `json:"source"`
	Evidence             map[string]interface{} `json:"evidence,omitempty"`
	Overridable          bool                   `json:"overridable"`
}

// ValidationResult represents a validation result
type ValidationResult struct {
	ValidationID    string              `json:"validation_id"`
	ProposalID      string              `json:"proposal_id"`
	Verdict         string              `json:"verdict"`
	RiskScore       float64             `json:"risk_score"`
	Findings        []ValidationFinding `json:"findings"`
	ProcessingTime  float64             `json:"processing_time_ms"`
	ValidatedBy     []string            `json:"validated_by"`
}

// safetyGatewayClientImpl implements SafetyGatewayClient
type safetyGatewayClientImpl struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewSafetyGatewayClient creates a new Safety Gateway client
func NewSafetyGatewayClient(baseURL string, logger *zap.Logger) SafetyGatewayClient {
	return &safetyGatewayClientImpl{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 45 * time.Second, // Longer timeout for comprehensive validation
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		logger: logger,
	}
}

// ComprehensiveValidation performs comprehensive safety validation
func (c *safetyGatewayClientImpl) ComprehensiveValidation(ctx context.Context, request *SafetyValidationRequest) (*SafetyValidationResponse, error) {
	c.logger.Info("Starting comprehensive safety validation",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("snapshot_id", request.SnapshotID),
		zap.Int("proposal_count", len(request.Proposals)))

	// Set default validation requirements if not specified
	if request.ValidationRequirements == nil {
		request.ValidationRequirements = map[string]interface{}{
			"cae_engine":              true,
			"protocol_engine":         true,
			"interaction_engine":      true,
			"contraindication_engine": true,
			"allergy_engine":          true,
			"dosing_engine":           true,
			"comprehensive_validation": true,
		}
	}

	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/validation/comprehensive", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")
	httpReq.Header.Set("X-Correlation-ID", request.CorrelationID)
	httpReq.Header.Set("X-Validation-Priority", "HIGH") // Priority for workflow orchestration

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	c.logger.Info("Safety Gateway validation completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.Duration("duration", duration),
		zap.Int("status_code", func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}()))

	if err != nil {
		c.logger.Error("Failed to call Safety Gateway",
			zap.String("correlation_id", request.CorrelationID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute validation request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("Safety Gateway returned error",
			zap.String("correlation_id", request.CorrelationID),
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("Safety Gateway returned status %d", resp.StatusCode)
	}

	// Parse response
	var response SafetyValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Validate response
	if response.ValidationID == "" {
		return nil, fmt.Errorf("invalid response: missing validation_id")
	}

	// Log validation results
	c.logger.Info("Safety validation completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("validation_id", response.ValidationID),
		zap.String("verdict", response.Verdict),
		zap.Float64("risk_score", response.OverallRiskScore),
		zap.Int("findings_count", len(response.Findings)),
		zap.Strings("engines", response.ExecutedEngines))

	// Log critical findings
	for _, finding := range response.Findings {
		if finding.Severity == "CRITICAL" || finding.Severity == "HIGH" {
			c.logger.Warn("Critical safety finding",
				zap.String("correlation_id", request.CorrelationID),
				zap.String("finding_id", finding.FindingID),
				zap.String("severity", finding.Severity),
				zap.String("category", finding.Category),
				zap.String("description", finding.Description))
		}
	}

	return &response, nil
}

// ValidateProposal validates a single proposal
func (c *safetyGatewayClientImpl) ValidateProposal(ctx context.Context, request *ProposalValidationRequest) (*ValidationResult, error) {
	c.logger.Info("Validating single proposal",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("proposal_id", request.ProposalID),
		zap.String("validation_level", request.ValidationLevel))

	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/validation/proposal", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")
	httpReq.Header.Set("X-Correlation-ID", request.CorrelationID)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("Failed to validate proposal",
			zap.String("correlation_id", request.CorrelationID),
			zap.String("proposal_id", request.ProposalID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute validation request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Safety Gateway returned status %d", resp.StatusCode)
	}

	// Parse response
	var response ValidationResult
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Proposal validation completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("proposal_id", request.ProposalID),
		zap.String("validation_id", response.ValidationID),
		zap.String("verdict", response.Verdict),
		zap.Float64("risk_score", response.RiskScore),
		zap.Float64("processing_time", response.ProcessingTime))

	return &response, nil
}

// HealthCheck checks if Safety Gateway is healthy
func (c *safetyGatewayClientImpl) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Safety Gateway unhealthy: status %d", resp.StatusCode)
	}

	// Parse health response to check validation engines status
	var healthResponse struct {
		Status  string            `json:"status"`
		Engines map[string]string `json:"engines,omitempty"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		c.logger.Warn("Failed to decode health response", zap.Error(err))
		// Still return nil if basic health check passed
		return nil
	}

	// Check individual engines if available
	if healthResponse.Engines != nil {
		for engine, status := range healthResponse.Engines {
			if status != "healthy" && status != "ok" {
				c.logger.Warn("Safety Gateway engine unhealthy",
					zap.String("engine", engine),
					zap.String("status", status))
			}
		}
	}

	return nil
}