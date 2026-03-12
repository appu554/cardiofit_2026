// Package cql provides CQL (Clinical Quality Language) integration for KB-13.
//
// 🔴 CRITICAL ARCHITECTURE (CTO/CMO Gate):
//   - KB-13 MUST use BATCH CQL evaluation only
//   - NO per-patient CQL calls allowed (scalability requirement)
//   - Uses Vaidshala CQL Engine via HTTP API
//
// This client wraps the Vaidshala clinical-runtime-platform for:
//   - Batch population evaluation
//   - Clinical fact extraction
//   - Measure criteria evaluation
package cql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Client provides CQL evaluation via Vaidshala CQL Engine.
// 🔴 This client enforces BATCH-ONLY evaluation pattern.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// ClientConfig configures the CQL client.
type ClientConfig struct {
	BaseURL string
	Timeout time.Duration
}

// NewClient creates a new CQL client.
func NewClient(cfg ClientConfig, logger *zap.Logger) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger: logger,
	}
}

// BatchEvaluationRequest represents a batch CQL evaluation request.
// 🔴 This is the ONLY way to evaluate CQL in KB-13.
type BatchEvaluationRequest struct {
	// MeasureID is the quality measure being evaluated
	MeasureID string `json:"measure_id"`

	// LibraryID is the CQL library identifier
	LibraryID string `json:"library_id"`

	// Expression is the CQL expression to evaluate
	Expression string `json:"expression"`

	// PeriodStart is the measurement period start (ISO 8601)
	PeriodStart string `json:"period_start"`

	// PeriodEnd is the measurement period end (ISO 8601)
	PeriodEnd string `json:"period_end"`

	// PopulationFilter optionally filters the population
	PopulationFilter *PopulationFilter `json:"population_filter,omitempty"`

	// Parameters are additional CQL parameters
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// PopulationFilter defines criteria for population selection.
type PopulationFilter struct {
	// AgeMin minimum age for inclusion
	AgeMin int `json:"age_min,omitempty"`

	// AgeMax maximum age for inclusion
	AgeMax int `json:"age_max,omitempty"`

	// Gender filter (M, F, or empty for all)
	Gender string `json:"gender,omitempty"`

	// ConditionCodes ICD/SNOMED codes for condition filtering
	ConditionCodes []string `json:"condition_codes,omitempty"`

	// ValueSetOIDs for condition filtering
	ValueSetOIDs []string `json:"value_set_oids,omitempty"`
}

// BatchEvaluationResponse represents the CQL evaluation result.
type BatchEvaluationResponse struct {
	// MeasureID echoes back the measure being evaluated
	MeasureID string `json:"measure_id"`

	// Expression echoes back the expression evaluated
	Expression string `json:"expression"`

	// TotalPopulation is the total patients evaluated
	TotalPopulation int `json:"total_population"`

	// MatchingCount is the count of patients matching the expression
	MatchingCount int `json:"matching_count"`

	// PatientIDs optionally contains the IDs of matching patients
	// (only populated if requested and population is small enough)
	PatientIDs []string `json:"patient_ids,omitempty"`

	// ExecutionTimeMs is the CQL engine execution time
	ExecutionTimeMs int64 `json:"execution_time_ms"`

	// EngineVersion is the CQL engine version for audit
	EngineVersion string `json:"engine_version"`

	// Errors contains any evaluation errors
	Errors []string `json:"errors,omitempty"`
}

// EvaluateBatch performs batch CQL evaluation.
// 🔴 This is the ONLY method for CQL evaluation in KB-13.
func (c *Client) EvaluateBatch(ctx context.Context, req *BatchEvaluationRequest) (*BatchEvaluationResponse, error) {
	c.logger.Debug("Starting batch CQL evaluation",
		zap.String("measure_id", req.MeasureID),
		zap.String("expression", req.Expression),
		zap.String("period_start", req.PeriodStart),
		zap.String("period_end", req.PeriodEnd),
	)

	startTime := time.Now()

	// Serialize request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build request
	url := c.baseURL + "/v1/cql/evaluate/batch"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("CQL batch evaluation failed",
			zap.String("measure_id", req.MeasureID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("CQL evaluation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("CQL evaluation returned error status",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)),
		)
		return nil, fmt.Errorf("CQL evaluation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result BatchEvaluationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Info("Batch CQL evaluation completed",
		zap.String("measure_id", req.MeasureID),
		zap.String("expression", req.Expression),
		zap.Int("total_population", result.TotalPopulation),
		zap.Int("matching_count", result.MatchingCount),
		zap.Int64("cql_time_ms", result.ExecutionTimeMs),
		zap.Duration("total_time", time.Since(startTime)),
	)

	return &result, nil
}

// MeasureEvaluationRequest requests evaluation of all measure populations.
type MeasureEvaluationRequest struct {
	MeasureID   string `json:"measure_id"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`

	// Populations to evaluate (initial, denominator, numerator, etc.)
	Populations []PopulationEvaluation `json:"populations"`
}

// PopulationEvaluation defines a single population to evaluate.
type PopulationEvaluation struct {
	PopulationType string `json:"population_type"`
	CQLExpression  string `json:"cql_expression"`
}

// MeasureEvaluationResponse contains results for all populations.
type MeasureEvaluationResponse struct {
	MeasureID       string                    `json:"measure_id"`
	PeriodStart     string                    `json:"period_start"`
	PeriodEnd       string                    `json:"period_end"`
	Populations     map[string]PopulationResult `json:"populations"`
	ExecutionTimeMs int64                     `json:"execution_time_ms"`
	EngineVersion   string                    `json:"engine_version"`
}

// PopulationResult contains the result for a single population.
type PopulationResult struct {
	Count      int      `json:"count"`
	PatientIDs []string `json:"patient_ids,omitempty"`
}

// EvaluateMeasure evaluates all populations for a measure in a single batch.
// This is more efficient than calling EvaluateBatch multiple times.
func (c *Client) EvaluateMeasure(ctx context.Context, req *MeasureEvaluationRequest) (*MeasureEvaluationResponse, error) {
	c.logger.Debug("Starting measure evaluation",
		zap.String("measure_id", req.MeasureID),
		zap.Int("population_count", len(req.Populations)),
	)

	startTime := time.Now()

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/v1/cql/evaluate/measure"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("measure evaluation request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("measure evaluation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result MeasureEvaluationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	c.logger.Info("Measure evaluation completed",
		zap.String("measure_id", req.MeasureID),
		zap.Int64("total_time_ms", time.Since(startTime).Milliseconds()),
	)

	return &result, nil
}

// Health checks connectivity to the CQL engine.
func (c *Client) Health(ctx context.Context) error {
	url := c.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CQL engine health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CQL engine unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
