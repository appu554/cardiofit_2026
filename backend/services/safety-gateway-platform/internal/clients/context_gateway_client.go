package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// ContextGatewayClient handles HTTP communication with the Context Service (Python FastAPI)
type ContextGatewayClient struct {
	httpClient     *http.Client
	baseURL        string
	config         *config.ContextGatewayConfig
	logger         *logger.Logger
	circuitBreaker *CircuitBreaker
}

// CircuitBreakerConfig defines circuit breaker parameters
type CircuitBreakerConfig struct {
	FailureThreshold int           `yaml:"failure_threshold"`
	ResetTimeout     time.Duration `yaml:"reset_timeout"`
	MaxRequests      int           `yaml:"max_requests"`
}

// CircuitBreaker implements basic circuit breaker pattern
type CircuitBreaker struct {
	failureCount    int
	lastFailureTime time.Time
	state           string // "closed", "open", "half-open"
	config          CircuitBreakerConfig
	logger          *logger.Logger
}

// NewContextGatewayClient creates a new Context Gateway HTTP client
func NewContextGatewayClient(cfg *config.ContextGatewayConfig, logger *logger.Logger) (*ContextGatewayClient, error) {
	// Set up HTTP client with timeout
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	// Initialize circuit breaker with default config
	circuitBreakerCfg := CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
		MaxRequests:      10,
	}
	circuitBreaker := &CircuitBreaker{
		state:  "closed",
		config: circuitBreakerCfg,
		logger: logger,
	}

	// Build base URL using endpoint from config
	baseURL := strings.TrimSuffix(cfg.Endpoint, "/")
	// If endpoint doesn't have protocol, add http://
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		if cfg.EnableTLS {
			baseURL = "https://" + baseURL
		} else {
			baseURL = "http://" + baseURL
		}
	}

	cgClient := &ContextGatewayClient{
		httpClient:     httpClient,
		baseURL:        baseURL,
		config:         cfg,
		logger:         logger,
		circuitBreaker: circuitBreaker,
	}

	// Perform health check if enabled
	if cfg.HealthCheck {
		if err := cgClient.healthCheck(); err != nil {
			return nil, fmt.Errorf("Context Service health check failed: %w", err)
		}
	}

	logger.Info("Context Gateway HTTP client initialized",
		zap.String("base_url", baseURL),
		zap.Duration("timeout", cfg.Timeout),
		zap.Int("max_retries", cfg.MaxRetries),
		zap.String("service_name", cfg.ServiceName),
	)

	return cgClient, nil
}

// GetSnapshot retrieves a clinical snapshot by ID using HTTP REST API
func (c *ContextGatewayClient) GetSnapshot(ctx context.Context, snapshotID string) (*types.ClinicalSnapshot, error) {
	if snapshotID == "" {
		return nil, fmt.Errorf("snapshot ID cannot be empty")
	}

	c.logger.Debug("Requesting snapshot from Context Service",
		zap.String("snapshot_id", snapshotID),
	)

	// Check circuit breaker
	if !c.circuitBreaker.allowRequest() {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	// Build URL: GET /api/snapshots/{snapshot_id}
	url := fmt.Sprintf("%s/api/snapshots/%s", c.baseURL, snapshotID)

	// Execute with retry logic
	snapshot, err := c.executeWithRetry(ctx, "GET", url, nil)
	if err != nil {
		c.circuitBreaker.recordFailure()
		c.logger.Error("Failed to get snapshot from Context Service",
			zap.String("snapshot_id", snapshotID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("Context Service request failed: %w", err)
	}

	c.circuitBreaker.recordSuccess()
	c.logger.Debug("Successfully retrieved snapshot",
		zap.String("snapshot_id", snapshotID),
		zap.String("patient_id", snapshot.PatientID),
		zap.Float64("data_completeness", snapshot.DataCompleteness),
	)

	return snapshot, nil
}

// CreateSnapshot creates a new clinical snapshot via HTTP REST API
func (c *ContextGatewayClient) CreateSnapshot(ctx context.Context, request *types.SnapshotRequest) (*types.ClinicalSnapshot, error) {
	c.logger.Debug("Creating snapshot via Context Service",
		zap.String("patient_id", request.PatientID),
		zap.String("recipe_id", request.RecipeID),
	)

	// Check circuit breaker
	if !c.circuitBreaker.allowRequest() {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	// Build URL: POST /api/snapshots
	url := fmt.Sprintf("%s/api/snapshots", c.baseURL)

	// Convert request to JSON
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Execute request
	snapshot, err := c.executeWithRetry(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		c.circuitBreaker.recordFailure()
		c.logger.Error("Failed to create snapshot",
			zap.String("patient_id", request.PatientID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("Context Service request failed: %w", err)
	}

	c.circuitBreaker.recordSuccess()
	c.logger.Info("Successfully created snapshot",
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("patient_id", snapshot.PatientID),
	)

	return snapshot, nil
}

// ValidateSnapshot validates a snapshot using HTTP REST API
func (c *ContextGatewayClient) ValidateSnapshot(ctx context.Context, snapshotID string) (*types.SnapshotValidationResult, error) {
	if snapshotID == "" {
		return nil, fmt.Errorf("snapshot ID cannot be empty")
	}

	c.logger.Debug("Validating snapshot via Context Service",
		zap.String("snapshot_id", snapshotID),
	)

	// Check circuit breaker
	if !c.circuitBreaker.allowRequest() {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	// Build URL: POST /api/snapshots/{snapshot_id}/validate
	url := fmt.Sprintf("%s/api/snapshots/%s/validate", c.baseURL, snapshotID)

	// Execute request
	resp, err := c.httpClient.Post(url, "application/json", nil)
	if err != nil {
		c.circuitBreaker.recordFailure()
		c.logger.Error("Failed to validate snapshot",
			zap.String("snapshot_id", snapshotID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("Context Service request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("validation request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var validationResult types.SnapshotValidationResult
	if err := json.NewDecoder(resp.Body).Decode(&validationResult); err != nil {
		return nil, fmt.Errorf("failed to decode validation response: %w", err)
	}

	c.circuitBreaker.recordSuccess()
	c.logger.Debug("Snapshot validation completed",
		zap.String("snapshot_id", snapshotID),
		zap.Bool("valid", validationResult.Valid),
		zap.Bool("checksum_valid", validationResult.ChecksumValid),
		zap.Bool("signature_valid", validationResult.SignatureValid),
	)

	return &validationResult, nil
}

// ListSnapshots retrieves a list of snapshots with optional filtering
func (c *ContextGatewayClient) ListSnapshots(ctx context.Context, patientID string, limit int) ([]*types.SnapshotReference, error) {
	c.logger.Debug("Listing snapshots from Context Service",
		zap.String("patient_id", patientID),
		zap.Int("limit", limit),
	)

	// Check circuit breaker
	if !c.circuitBreaker.allowRequest() {
		return nil, fmt.Errorf("circuit breaker is open")
	}

	// Build URL: GET /api/snapshots with optional query parameters
	url := fmt.Sprintf("%s/api/snapshots", c.baseURL)
	if patientID != "" {
		url += fmt.Sprintf("?patient_id=%s", patientID)
	}
	if limit > 0 {
		separator := "?"
		if patientID != "" {
			separator = "&"
		}
		url += fmt.Sprintf("%slimit=%d", separator, limit)
	}

	// Execute request
	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.circuitBreaker.recordFailure()
		return nil, fmt.Errorf("list request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var snapshots []*types.SnapshotReference
	if err := json.NewDecoder(resp.Body).Decode(&snapshots); err != nil {
		return nil, fmt.Errorf("failed to decode snapshots list: %w", err)
	}

	c.circuitBreaker.recordSuccess()
	c.logger.Debug("Successfully listed snapshots",
		zap.Int("count", len(snapshots)),
	)

	return snapshots, nil
}

// Close cleans up the HTTP client resources
func (c *ContextGatewayClient) Close() error {
	// HTTP client doesn't need explicit cleanup
	c.logger.Info("Context Gateway client closed")
	return nil
}

// executeWithRetry executes HTTP request with retry logic and returns parsed snapshot
func (c *ContextGatewayClient) executeWithRetry(ctx context.Context, method, url string, body io.Reader) (*types.ClinicalSnapshot, error) {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Warn("Retrying HTTP request",
				zap.String("method", method),
				zap.String("url", url),
				zap.Int("attempt", attempt),
			)
			
			// Exponential backoff
			backoff := time.Duration(attempt) * 200 * time.Millisecond
			time.Sleep(backoff)
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Safety-Gateway-Platform/1.0")
		req.Header.Set("X-Requesting-Service", c.config.ServiceName)

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			if c.isRetryableError(err) {
				continue
			}
			break
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: status %d", resp.StatusCode)
			continue // Retry on server errors
		}

		if resp.StatusCode >= 400 {
			// Client errors are not retryable
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("client error: status %d, body: %s", resp.StatusCode, string(body))
			break
		}

		// Parse successful response
		var snapshot types.ClinicalSnapshot
		if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
			lastErr = fmt.Errorf("failed to decode response: %w", err)
			break
		}

		return &snapshot, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.config.MaxRetries+1, lastErr)
}

// Circuit breaker implementation methods

// allowRequest checks if the circuit breaker allows the request
func (cb *CircuitBreaker) allowRequest() bool {
	now := time.Now()

	switch cb.state {
	case "closed":
		return true
	case "open":
		// Check if reset timeout has elapsed
		if now.Sub(cb.lastFailureTime) > cb.config.ResetTimeout {
			cb.logger.Info("Circuit breaker transitioning to half-open")
			cb.state = "half-open"
			cb.failureCount = 0
			return true
		}
		return false
	case "half-open":
		return true
	default:
		return true
	}
}

// recordSuccess records a successful request
func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == "half-open" {
		cb.logger.Info("Circuit breaker transitioning to closed")
		cb.state = "closed"
	}
	cb.failureCount = 0
}

// recordFailure records a failed request
func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == "closed" && cb.failureCount >= cb.config.FailureThreshold {
		cb.logger.Warn("Circuit breaker transitioning to open",
			zap.Int("failure_count", cb.failureCount),
			zap.Int("threshold", cb.config.FailureThreshold),
		)
		cb.state = "open"
	} else if cb.state == "half-open" {
		cb.logger.Warn("Circuit breaker transitioning back to open")
		cb.state = "open"
	}
}

// healthCheck performs a health check against Context Service
func (c *ContextGatewayClient) healthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Build URL: GET /health
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Context Service is not healthy: status %d, body: %s", resp.StatusCode, string(body))
	}

	c.logger.Debug("Context Service health check passed")
	return nil
}

// isRetryableError determines if an HTTP error is retryable
func (c *ContextGatewayClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors that are typically retryable
	errStr := err.Error()
	retryableErrors := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"temporary failure",
		"no such host",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(strings.ToLower(errStr), retryable) {
			return true
		}
	}

	return false
}

// GetStats returns client statistics for monitoring
func (c *ContextGatewayClient) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"base_url":              c.baseURL,
		"timeout":               c.config.Timeout.String(),
		"max_retries":           c.config.MaxRetries,
		"circuit_breaker_state": c.circuitBreaker.state,
		"failure_count":         c.circuitBreaker.failureCount,
		"last_failure_time":     c.circuitBreaker.lastFailureTime,
	}
}