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

// Flow2GoClient defines the interface for Flow2 Go Engine communication
type Flow2GoClient interface {
	ExecuteAdvanced(ctx context.Context, request *Flow2ExecuteRequest) (*Flow2ExecuteResponse, error)
	GenerateAlternatives(ctx context.Context, request *Flow2AlternativesRequest) (*Flow2AlternativesResponse, error)
	HealthCheck(ctx context.Context) error
}

// Flow2ExecuteRequest represents a request to Flow2 Go Engine
type Flow2ExecuteRequest struct {
	PatientID         string                 `json:"patient_id"`
	Medication        map[string]interface{} `json:"medication"`
	ClinicalIntent    map[string]interface{} `json:"clinical_intent"`
	ProviderContext   map[string]interface{} `json:"provider_context"`
	ExecutionMode     string                 `json:"execution_mode"`
	CorrelationID     string                 `json:"correlation_id"`
	SnapshotOptimized bool                   `json:"snapshot_optimized,omitempty"`
	UseCache          bool                   `json:"use_cache,omitempty"`
}

// Flow2ExecuteResponse represents a response from Flow2 Go Engine
type Flow2ExecuteResponse struct {
	ProposalSetID     string                   `json:"proposal_set_id"`
	SnapshotID        string                   `json:"snapshot_id"`
	RankedProposals   []map[string]interface{} `json:"ranked_proposals"`
	ClinicalEvidence  map[string]interface{}   `json:"clinical_evidence"`
	MonitoringPlan    map[string]interface{}   `json:"monitoring_plan"`
	KBVersions        map[string]string        `json:"kb_versions"`
	ExecutionMetrics  map[string]interface{}   `json:"execution_metrics"`
	RecipeReference   map[string]interface{}   `json:"recipe_reference,omitempty"`
	Status            string                   `json:"status"`
	Message           string                   `json:"message,omitempty"`
}

// Flow2AlternativesRequest represents a request for alternative proposals
type Flow2AlternativesRequest struct {
	SnapshotID       string                   `json:"snapshot_id"`
	BlockingFindings []map[string]interface{} `json:"blocking_findings"`
	PatientContext   map[string]interface{}   `json:"patient_context,omitempty"`
	CorrelationID    string                   `json:"correlation_id,omitempty"`
}

// Flow2AlternativesResponse represents alternative proposals response
type Flow2AlternativesResponse struct {
	Alternatives      []map[string]interface{} `json:"alternatives"`
	ReasoningContext  map[string]interface{}   `json:"reasoning_context"`
	ConfidenceScores  map[string]float64       `json:"confidence_scores"`
	Status            string                   `json:"status"`
}

// flow2GoClientImpl implements Flow2GoClient
type flow2GoClientImpl struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewFlow2GoClient creates a new Flow2 Go Engine client
func NewFlow2GoClient(baseURL string, logger *zap.Logger) Flow2GoClient {
	return &flow2GoClientImpl{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		logger: logger,
	}
}

// ExecuteAdvanced executes advanced medication intelligence via Flow2 Go Engine
func (c *flow2GoClientImpl) ExecuteAdvanced(ctx context.Context, request *Flow2ExecuteRequest) (*Flow2ExecuteResponse, error) {
	c.logger.Info("Calling Flow2 Go Engine execute advanced",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("patient_id", request.PatientID))

	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/snapshots/execute-advanced", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")
	httpReq.Header.Set("X-Correlation-ID", request.CorrelationID)

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	c.logger.Info("Flow2 Go Engine request completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.Duration("duration", duration),
		zap.Int("status_code", func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}()))

	if err != nil {
		c.logger.Error("Failed to call Flow2 Go Engine",
			zap.String("correlation_id", request.CorrelationID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Flow2 Go Engine returned status %d", resp.StatusCode)
	}

	// Parse response
	var response Flow2ExecuteResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Validate response
	if response.Status != "success" && response.Status != "completed" {
		return nil, fmt.Errorf("Flow2 Go Engine execution failed: %s", response.Message)
	}

	if response.ProposalSetID == "" || response.SnapshotID == "" {
		return nil, fmt.Errorf("invalid response: missing proposal_set_id or snapshot_id")
	}

	c.logger.Info("Flow2 Go Engine execution successful",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("proposal_set_id", response.ProposalSetID),
		zap.String("snapshot_id", response.SnapshotID),
		zap.Int("proposal_count", len(response.RankedProposals)))

	return &response, nil
}

// GenerateAlternatives generates alternative medication proposals
func (c *flow2GoClientImpl) GenerateAlternatives(ctx context.Context, request *Flow2AlternativesRequest) (*Flow2AlternativesResponse, error) {
	c.logger.Info("Generating alternatives via Flow2 Go Engine",
		zap.String("snapshot_id", request.SnapshotID),
		zap.Int("blocking_findings_count", len(request.BlockingFindings)))

	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/snapshots/generate-alternatives", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")
	if request.CorrelationID != "" {
		httpReq.Header.Set("X-Correlation-ID", request.CorrelationID)
	}

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("Failed to generate alternatives",
			zap.String("snapshot_id", request.SnapshotID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Flow2 Go Engine returned status %d", resp.StatusCode)
	}

	// Parse response
	var response Flow2AlternativesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Alternatives generated successfully",
		zap.String("snapshot_id", request.SnapshotID),
		zap.Int("alternatives_count", len(response.Alternatives)))

	return &response, nil
}

// HealthCheck checks if Flow2 Go Engine is healthy
func (c *flow2GoClientImpl) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("Flow2 Go Engine unhealthy: status %d", resp.StatusCode)
	}

	return nil
}