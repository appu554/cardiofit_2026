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

// CAEEngineClient handles communication with the Clinical Assertion Engine
type CAEEngineClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logger.Logger
}

// NewCAEEngineClient creates a new CAE Engine client
func NewCAEEngineClient(baseURL string, logger *logger.Logger) (*CAEEngineClient, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("CAE Engine URL is required")
	}

	httpClient := &http.Client{
		Timeout: 60 * time.Second, // CAE evaluations can take longer
		Transport: &http.Transport{
			MaxIdleConns:        50,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &CAEEngineClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// Evaluate performs a single CAE evaluation
func (c *CAEEngineClient) Evaluate(ctx context.Context, request *CAERequest) (*CAEResponse, error) {
	c.logger.Debug("Sending CAE evaluation request",
		zap.String("request_id", request.RequestID),
		zap.String("snapshot_id", request.SnapshotID),
		zap.String("evaluation_mode", request.EvaluationMode),
	)

	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CAE request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2.1/evaluate",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Safety-Gateway-Platform/2.0")
	req.Header.Set("X-Request-ID", request.RequestID)

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		c.logger.Error("CAE evaluation request failed",
			zap.String("request_id", request.RequestID),
			zap.Error(err),
			zap.Duration("duration", requestDuration),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("CAE Engine returned non-200 status",
			zap.String("request_id", request.RequestID),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", requestDuration),
		)
		return nil, fmt.Errorf("CAE Engine returned status %d", resp.StatusCode)
	}

	// Parse response
	var caeResponse CAEResponse
	if err := json.NewDecoder(resp.Body).Decode(&caeResponse); err != nil {
		return nil, fmt.Errorf("failed to decode CAE response: %w", err)
	}

	// Set processing time
	caeResponse.ProcessingTime = requestDuration

	c.logger.Info("CAE evaluation completed",
		zap.String("request_id", request.RequestID),
		zap.String("snapshot_id", request.SnapshotID),
		zap.String("decision", caeResponse.Decision),
		zap.Float64("risk_score", caeResponse.RiskScore),
		zap.Duration("processing_time", requestDuration),
		zap.Bool("ml_modulated", caeResponse.MLModulated),
	)

	return &caeResponse, nil
}

// BatchEvaluate performs batch CAE evaluation
func (c *CAEEngineClient) BatchEvaluate(ctx context.Context, requests []*CAERequest) ([]*CAEResponse, error) {
	c.logger.Info("Sending CAE batch evaluation request",
		zap.Int("request_count", len(requests)),
	)

	// Create batch request
	batchRequest := struct {
		Requests []*CAERequest `json:"requests"`
		BatchID  string        `json:"batch_id"`
	}{
		Requests: requests,
		BatchID:  fmt.Sprintf("batch-%d", time.Now().Unix()),
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(batchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CAE batch request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2.1/evaluate/batch",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Safety-Gateway-Platform/2.0")
	req.Header.Set("X-Batch-ID", batchRequest.BatchID)

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	batchDuration := time.Since(startTime)

	if err != nil {
		c.logger.Error("CAE batch evaluation request failed",
			zap.String("batch_id", batchRequest.BatchID),
			zap.Error(err),
			zap.Duration("duration", batchDuration),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("CAE Engine returned non-200 status for batch request",
			zap.String("batch_id", batchRequest.BatchID),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", batchDuration),
		)
		return nil, fmt.Errorf("CAE Engine returned status %d", resp.StatusCode)
	}

	// Parse response
	var batchResponse struct {
		BatchID   string         `json:"batch_id"`
		Responses []*CAEResponse `json:"responses"`
		Summary   struct {
			TotalRequests    int           `json:"total_requests"`
			SuccessfulCount  int           `json:"successful_count"`
			FailedCount      int           `json:"failed_count"`
			ProcessingTime   time.Duration `json:"processing_time"`
			AverageRiskScore float64       `json:"average_risk_score"`
		} `json:"summary"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&batchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode CAE batch response: %w", err)
	}

	c.logger.Info("CAE batch evaluation completed",
		zap.String("batch_id", batchRequest.BatchID),
		zap.Int("request_count", len(requests)),
		zap.Int("response_count", len(batchResponse.Responses)),
		zap.Int("successful_count", batchResponse.Summary.SuccessfulCount),
		zap.Int("failed_count", batchResponse.Summary.FailedCount),
		zap.Duration("total_duration", batchDuration),
		zap.Float64("average_risk_score", batchResponse.Summary.AverageRiskScore),
	)

	return batchResponse.Responses, nil
}

// WhatIfAnalysis performs what-if scenario analysis
func (c *CAEEngineClient) WhatIfAnalysis(
	ctx context.Context,
	baselineSnapshotID string,
	scenarios []MedicationScenario,
) (*WhatIfAnalysisResponse, error) {
	c.logger.Debug("Sending CAE what-if analysis request",
		zap.String("baseline_snapshot_id", baselineSnapshotID),
		zap.Int("scenario_count", len(scenarios)),
	)

	// Create what-if request
	whatIfRequest := struct {
		BaselineSnapshotID string               `json:"baseline_snapshot_id"`
		Scenarios          []MedicationScenario `json:"scenarios"`
		AnalysisType       string               `json:"analysis_type"`
	}{
		BaselineSnapshotID: baselineSnapshotID,
		Scenarios:          scenarios,
		AnalysisType:       "comparative",
	}

	// Marshal request to JSON
	requestBody, err := json.Marshal(whatIfRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal what-if request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/api/v2.1/what-if",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Safety-Gateway-Platform/2.0")

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(req)
	requestDuration := time.Since(startTime)

	if err != nil {
		c.logger.Error("CAE what-if analysis request failed",
			zap.Error(err),
			zap.Duration("duration", requestDuration),
		)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("CAE Engine returned non-200 status for what-if request",
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", requestDuration),
		)
		return nil, fmt.Errorf("CAE Engine returned status %d", resp.StatusCode)
	}

	// Parse response
	var whatIfResponse WhatIfAnalysisResponse
	if err := json.NewDecoder(resp.Body).Decode(&whatIfResponse); err != nil {
		return nil, fmt.Errorf("failed to decode what-if response: %w", err)
	}

	c.logger.Info("CAE what-if analysis completed",
		zap.String("baseline_snapshot_id", baselineSnapshotID),
		zap.Int("scenario_count", len(scenarios)),
		zap.Duration("processing_time", requestDuration),
	)

	return &whatIfResponse, nil
}

// GetEngineInfo retrieves information about the CAE Engine
func (c *CAEEngineClient) GetEngineInfo(ctx context.Context) (*CAEEngineInfo, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v2.1/info",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Safety-Gateway-Platform/2.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CAE Engine returned status %d", resp.StatusCode)
	}

	var engineInfo CAEEngineInfo
	if err := json.NewDecoder(resp.Body).Decode(&engineInfo); err != nil {
		return nil, fmt.Errorf("failed to decode engine info: %w", err)
	}

	return &engineInfo, nil
}

// Health checks the health of the CAE Engine
func (c *CAEEngineClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/health",
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CAE Engine health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// Close cleans up the client resources
func (c *CAEEngineClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// Supporting types for CAE Engine communication

// MedicationScenario represents a medication scenario for what-if analysis
type MedicationScenario struct {
	ScenarioID      string          `json:"scenario_id"`
	Description     string          `json:"description"`
	ProposedAction  *ClinicalAction `json:"proposed_action"`
	Modifications   []Modification  `json:"modifications"`
}

// Modification represents a change to the patient's clinical state
type Modification struct {
	Type        string      `json:"type"`        // "add_medication", "remove_medication", "change_dose", etc.
	Target      string      `json:"target"`      // medication ID, condition ID, etc.
	Value       interface{} `json:"value"`       // new value or modification details
	Description string      `json:"description"`
}

// WhatIfAnalysisResponse represents the response from what-if analysis
type WhatIfAnalysisResponse struct {
	BaselineSnapshotID string                   `json:"baseline_snapshot_id"`
	Scenarios          []ScenarioResult         `json:"scenarios"`
	Recommendations    []Recommendation         `json:"recommendations"`
	Summary            WhatIfSummary            `json:"summary"`
	Disclaimer         string                   `json:"disclaimer"`
}

// ScenarioResult represents the result of a what-if scenario
type ScenarioResult struct {
	Scenario   MedicationScenario `json:"scenario"`
	Result     *CAEResponse       `json:"result"`
	RiskDelta  float64            `json:"risk_delta"`
	Comparison string             `json:"comparison"` // "safer", "riskier", "similar"
}

// WhatIfSummary provides summary information about the what-if analysis
type WhatIfSummary struct {
	TotalScenarios   int     `json:"total_scenarios"`
	SaferScenarios   int     `json:"safer_scenarios"`
	RiskierScenarios int     `json:"riskier_scenarios"`
	SimilarScenarios int     `json:"similar_scenarios"`
	RecommendedOption string `json:"recommended_option"`
	ConfidenceLevel   string `json:"confidence_level"`
}

// CAEEngineInfo provides information about the CAE Engine
type CAEEngineInfo struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	APIVersion      string            `json:"api_version"`
	Features        []string          `json:"features"`
	KnowledgeBases  map[string]string `json:"knowledge_bases"`
	HealthStatus    string            `json:"health_status"`
	Uptime          time.Duration     `json:"uptime"`
	ProcessedCount  int64             `json:"processed_count"`
	ErrorRate       float64           `json:"error_rate"`
	AverageLatency  time.Duration     `json:"average_latency"`
}

// SetTimeout sets the HTTP client timeout
func (c *CAEEngineClient) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// GetBaseURL returns the base URL of the CAE Engine
func (c *CAEEngineClient) GetBaseURL() string {
	return c.baseURL
}