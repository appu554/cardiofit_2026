// Package clients provides HTTP clients for external services
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"flow2-go-engine/internal/models"

	"github.com/sirupsen/logrus"
)

// JITSafetyClient interface for JIT Safety Engine communication
type JITSafetyClient interface {
	RunJITSafetyCheck(ctx context.Context, request *models.JitSafetyContext) (*models.JitSafetyOutcome, error)
	RunEnhancedSafetyCheck(ctx context.Context, request *models.EnhancedSafetyRequest) (*models.EnhancedSafetyResponse, error)
	HealthCheck(ctx context.Context) error
}

// jitSafetyClient implements JITSafetyClient
type jitSafetyClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
	config     JITSafetyConfig
}

// JITSafetyConfig holds configuration for JIT Safety Client
type JITSafetyConfig struct {
	BaseURL        string        `json:"base_url" yaml:"base_url"`
	TimeoutSeconds int           `json:"timeout_seconds" yaml:"timeout_seconds"`
	RetryAttempts  int           `json:"retry_attempts" yaml:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay" yaml:"retry_delay"`
	EnableCircuitBreaker bool    `json:"enable_circuit_breaker" yaml:"enable_circuit_breaker"`
}

// NewJITSafetyClient creates a new JIT Safety Client
func NewJITSafetyClient(config JITSafetyConfig, logger *logrus.Logger) JITSafetyClient {
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 30
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 * time.Millisecond
	}

	httpClient := &http.Client{
		Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			MaxIdleConnsPerHost: 5,
		},
	}

	return &jitSafetyClient{
		baseURL:    config.BaseURL,
		httpClient: httpClient,
		logger:     logger,
		config:     config,
	}
}

// RunJITSafetyCheck performs JIT safety evaluation
func (j *jitSafetyClient) RunJITSafetyCheck(ctx context.Context, request *models.JitSafetyContext) (*models.JitSafetyOutcome, error) {
	startTime := time.Now()
	
	j.logger.WithFields(logrus.Fields{
		"request_id": request.RequestID,
		"drug_id":    request.Proposal.DrugID,
		"dose_mg":    request.Proposal.DoseMg,
	}).Debug("Starting JIT Safety check")

	// Validate request
	if err := j.validateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Perform request with retry logic
	outcome, err := j.performRequestWithRetry(ctx, request)
	if err != nil {
		j.logger.WithError(err).WithField("request_id", request.RequestID).Error("JIT Safety check failed")
		return nil, err
	}

	duration := time.Since(startTime)
	j.logger.WithFields(logrus.Fields{
		"request_id": request.RequestID,
		"decision":   outcome.Decision,
		"duration_ms": duration.Milliseconds(),
		"reasons":    len(outcome.Reasons),
		"ddis":       len(outcome.DDIs),
	}).Info("JIT Safety check completed")

	return outcome, nil
}

// performRequestWithRetry performs the HTTP request with retry logic
func (j *jitSafetyClient) performRequestWithRetry(ctx context.Context, request *models.JitSafetyContext) (*models.JitSafetyOutcome, error) {
	var lastErr error
	
	for attempt := 1; attempt <= j.config.RetryAttempts; attempt++ {
		outcome, err := j.performRequest(ctx, request)
		if err == nil {
			return outcome, nil
		}

		lastErr = err
		
		// Don't retry on client errors (4xx)
		if httpErr, ok := err.(*HTTPError); ok && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
			break
		}

		if attempt < j.config.RetryAttempts {
			j.logger.WithFields(logrus.Fields{
				"attempt":    attempt,
				"max_attempts": j.config.RetryAttempts,
				"request_id": request.RequestID,
				"error":      err.Error(),
			}).Warn("JIT Safety request failed, retrying")
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(j.config.RetryDelay * time.Duration(attempt)):
				// Continue to next attempt
			}
		}
	}

	return nil, fmt.Errorf("JIT Safety request failed after %d attempts: %w", j.config.RetryAttempts, lastErr)
}

// performRequest performs a single HTTP request
func (j *jitSafetyClient) performRequest(ctx context.Context, request *models.JitSafetyContext) (*models.JitSafetyOutcome, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/v1/jit-safety-check", j.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("X-Request-ID", request.RequestID)

	// Perform request
	resp, err := j.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(responseBody),
			URL:        url,
		}
	}

	// Unmarshal response
	var outcome models.JitSafetyOutcome
	if err := json.Unmarshal(responseBody, &outcome); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &outcome, nil
}

// RunEnhancedSafetyCheck performs enhanced JIT safety evaluation (placeholder)
func (j *jitSafetyClient) RunEnhancedSafetyCheck(ctx context.Context, request *models.EnhancedSafetyRequest) (*models.EnhancedSafetyResponse, error) {
	// This is a placeholder for the enhanced safety check
	// In a real implementation, this would call the enhanced endpoint
	j.logger.WithField("request_id", request.RequestID).Info("Enhanced safety check requested (not yet implemented)")

	return &models.EnhancedSafetyResponse{
		RequestID: request.RequestID,
		Status:    "not_implemented",
		Message:   "Enhanced safety check not yet implemented",
	}, nil
}

// validateRequest validates the JIT Safety request
func (j *jitSafetyClient) validateRequest(request *models.JitSafetyContext) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}
	
	if request.RequestID == "" {
		return fmt.Errorf("request_id is required")
	}
	
	if request.Proposal.DrugID == "" {
		return fmt.Errorf("proposal.drug_id is required")
	}
	
	if request.Proposal.DoseMg <= 0 {
		return fmt.Errorf("proposal.dose_mg must be positive")
	}
	
	if request.Proposal.IntervalH == 0 {
		return fmt.Errorf("proposal.interval_h must be positive")
	}
	
	if request.Proposal.Route == "" {
		return fmt.Errorf("proposal.route is required")
	}
	
	if request.Patient.AgeYears == 0 {
		return fmt.Errorf("patient.age_years is required")
	}
	
	if request.Patient.WeightKg <= 0 {
		return fmt.Errorf("patient.weight_kg must be positive")
	}

	return nil
}

// HealthCheck performs a health check on the JIT Safety Engine
func (j *jitSafetyClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", j.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := j.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed with status %d: %s", resp.StatusCode, string(body))
	}

	j.logger.Debug("JIT Safety Engine health check passed")
	return nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Message    string
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d error from %s: %s", e.StatusCode, e.URL, e.Message)
}

// IsRetryable returns true if the error is retryable
func (e *HTTPError) IsRetryable() bool {
	// Retry on 5xx server errors and some 4xx errors
	return e.StatusCode >= 500 || e.StatusCode == 408 || e.StatusCode == 429
}
