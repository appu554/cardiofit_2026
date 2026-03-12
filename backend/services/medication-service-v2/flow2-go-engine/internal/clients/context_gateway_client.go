package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/config"
	"flow2-go-engine/internal/models"
)

// ContextGatewayClient interface defines operations for Context Gateway snapshot management
type ContextGatewayClient interface {
	// Snapshot operations
	CreateSnapshot(ctx context.Context, request *models.SnapshotRequest) (*models.ClinicalSnapshot, error)
	GetSnapshot(ctx context.Context, snapshotID string) (*models.ClinicalSnapshot, error)
	ValidateSnapshot(ctx context.Context, snapshotID string) (*models.SnapshotValidationResult, error)
	DeleteSnapshot(ctx context.Context, snapshotID string) error
	ListSnapshots(ctx context.Context, filters *models.SnapshotFilters) ([]*models.SnapshotSummary, error)
	BatchCreateSnapshots(ctx context.Context, requests []*models.SnapshotRequest) (*models.BatchSnapshotResult, error)

	// Service operations
	GetSnapshotMetrics(ctx context.Context) (*models.SnapshotMetrics, error)
	GetServiceStatus(ctx context.Context) (*models.ServiceStatus, error)
	CleanupExpiredSnapshots(ctx context.Context) (*models.CleanupResult, error)

	// System methods
	HealthCheck(ctx context.Context) error
	Close() error
}

// contextGatewayClient implements the ContextGatewayClient interface
type contextGatewayClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
	config     config.ContextServiceConfig
}

// NewHTTPContextGatewayClient creates a new Context Gateway HTTP client
func NewHTTPContextGatewayClient(cfg config.ContextServiceConfig) (ContextGatewayClient, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	if cfg.URL == "" {
		return nil, fmt.Errorf("context gateway URL is required")
	}

	logger.WithField("url", cfg.URL).Info("Initializing Context Gateway client")

	// Configure HTTP client with timeouts and retry logic
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	client := &contextGatewayClient{
		baseURL:    cfg.URL,
		httpClient: httpClient,
		logger:     logger,
		config:     cfg,
	}

	// Test connection immediately
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("context gateway health check failed: %w", err)
	}

	logger.Info("Successfully connected to Context Gateway")
	return client, nil
}

// CreateSnapshot creates a new clinical snapshot
func (c *contextGatewayClient) CreateSnapshot(ctx context.Context, request *models.SnapshotRequest) (*models.ClinicalSnapshot, error) {
	c.logger.WithFields(logrus.Fields{
		"patient_id": request.PatientID,
		"recipe_id":  request.RecipeID,
		"ttl_hours":  request.TTLHours,
	}).Info("Creating clinical snapshot")

	url := fmt.Sprintf("%s/api/snapshots", c.baseURL)
	
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot request: %w", err)
	}

	resp, err := c.makeHTTPRequest(ctx, "POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("snapshot creation failed: %w", err)
	}

	var snapshot models.ClinicalSnapshot
	if err := json.Unmarshal(resp, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"snapshot_id":        snapshot.ID,
		"completeness_score": snapshot.CompletenessScore,
	}).Info("✅ Clinical snapshot created successfully")

	return &snapshot, nil
}

// GetSnapshot retrieves a clinical snapshot by ID
func (c *contextGatewayClient) GetSnapshot(ctx context.Context, snapshotID string) (*models.ClinicalSnapshot, error) {
	c.logger.WithField("snapshot_id", snapshotID).Info("Retrieving clinical snapshot")

	url := fmt.Sprintf("%s/api/snapshots/%s", c.baseURL, snapshotID)
	
	resp, err := c.makeHTTPRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("snapshot retrieval failed: %w", err)
	}

	var snapshot models.ClinicalSnapshot
	if err := json.Unmarshal(resp, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"snapshot_id":    snapshot.ID,
		"access_count":   snapshot.AccessedCount,
		"expires_at":     snapshot.ExpiresAt,
	}).Info("✅ Clinical snapshot retrieved successfully")

	return &snapshot, nil
}

// ValidateSnapshot validates snapshot integrity and status
func (c *contextGatewayClient) ValidateSnapshot(ctx context.Context, snapshotID string) (*models.SnapshotValidationResult, error) {
	c.logger.WithField("snapshot_id", snapshotID).Info("Validating clinical snapshot")

	url := fmt.Sprintf("%s/api/snapshots/%s/validate", c.baseURL, snapshotID)
	
	resp, err := c.makeHTTPRequest(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("snapshot validation failed: %w", err)
	}

	var validation models.SnapshotValidationResult
	if err := json.Unmarshal(resp, &validation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validation response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"snapshot_id":      validation.SnapshotID,
		"valid":           validation.Valid,
		"checksum_valid":  validation.ChecksumValid,
		"signature_valid": validation.SignatureValid,
	}).Info("✅ Snapshot validation completed")

	return &validation, nil
}

// DeleteSnapshot deletes a clinical snapshot
func (c *contextGatewayClient) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	c.logger.WithField("snapshot_id", snapshotID).Info("Deleting clinical snapshot")

	url := fmt.Sprintf("%s/api/snapshots/%s", c.baseURL, snapshotID)
	
	_, err := c.makeHTTPRequest(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("snapshot deletion failed: %w", err)
	}

	c.logger.WithField("snapshot_id", snapshotID).Info("✅ Clinical snapshot deleted successfully")
	return nil
}

// ListSnapshots lists clinical snapshots with optional filtering
func (c *contextGatewayClient) ListSnapshots(ctx context.Context, filters *models.SnapshotFilters) ([]*models.SnapshotSummary, error) {
	c.logger.Info("Listing clinical snapshots with filters")

	url := fmt.Sprintf("%s/api/snapshots", c.baseURL)
	
	// Add query parameters for filtering
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if filters != nil {
		q := req.URL.Query()
		if filters.PatientID != "" {
			q.Add("patient_id", filters.PatientID)
		}
		if filters.ProviderID != "" {
			q.Add("provider_id", filters.ProviderID)
		}
		if filters.RecipeID != "" {
			q.Add("recipe_id", filters.RecipeID)
		}
		if filters.Status != "" {
			q.Add("status", filters.Status)
		}
		if filters.Limit > 0 {
			q.Add("limit", fmt.Sprintf("%d", filters.Limit))
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("snapshot list request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("snapshot list failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var summaries []*models.SnapshotSummary
	if err := json.Unmarshal(body, &summaries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot summaries: %w", err)
	}

	c.logger.WithField("count", len(summaries)).Info("✅ Snapshot listing completed")
	return summaries, nil
}

// BatchCreateSnapshots creates multiple snapshots in batch
func (c *contextGatewayClient) BatchCreateSnapshots(ctx context.Context, requests []*models.SnapshotRequest) (*models.BatchSnapshotResult, error) {
	c.logger.WithField("count", len(requests)).Info("Batch creating clinical snapshots")

	url := fmt.Sprintf("%s/api/snapshots/batch-create", c.baseURL)
	
	payload, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	resp, err := c.makeHTTPRequest(ctx, "POST", url, payload)
	if err != nil {
		return nil, fmt.Errorf("batch snapshot creation failed: %w", err)
	}

	var result models.BatchSnapshotResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal batch result: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"total_requested": result.TotalRequested,
		"successful":      len(result.Successful),
		"failed":         len(result.Failed),
	}).Info("✅ Batch snapshot creation completed")

	return &result, nil
}

// GetSnapshotMetrics retrieves snapshot service metrics
func (c *contextGatewayClient) GetSnapshotMetrics(ctx context.Context) (*models.SnapshotMetrics, error) {
	c.logger.Info("Retrieving snapshot service metrics")

	url := fmt.Sprintf("%s/api/snapshots/metrics", c.baseURL)
	
	resp, err := c.makeHTTPRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("metrics retrieval failed: %w", err)
	}

	var metrics models.SnapshotMetrics
	if err := json.Unmarshal(resp, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"total_snapshots":  metrics.TotalSnapshots,
		"active_snapshots": metrics.ActiveSnapshots,
	}).Info("✅ Snapshot metrics retrieved")

	return &metrics, nil
}

// GetServiceStatus retrieves Context Gateway service status
func (c *contextGatewayClient) GetServiceStatus(ctx context.Context) (*models.ServiceStatus, error) {
	c.logger.Info("Retrieving Context Gateway service status")

	url := fmt.Sprintf("%s/api/snapshots/status", c.baseURL)
	
	resp, err := c.makeHTTPRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("service status retrieval failed: %w", err)
	}

	var status models.ServiceStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status response: %w", err)
	}

	c.logger.WithField("status", status.Status).Info("✅ Service status retrieved")
	return &status, nil
}

// CleanupExpiredSnapshots manually triggers cleanup of expired snapshots
func (c *contextGatewayClient) CleanupExpiredSnapshots(ctx context.Context) (*models.CleanupResult, error) {
	c.logger.Info("Triggering manual snapshot cleanup")

	url := fmt.Sprintf("%s/api/snapshots/cleanup", c.baseURL)
	
	resp, err := c.makeHTTPRequest(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("cleanup request failed: %w", err)
	}

	var result models.CleanupResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cleanup response: %w", err)
	}

	c.logger.WithField("deleted_count", result.DeletedCount).Info("✅ Snapshot cleanup completed")
	return &result, nil
}

// HealthCheck performs a health check against the Context Gateway
func (c *contextGatewayClient) HealthCheck(ctx context.Context) error {
	c.logger.Debug("Performing Context Gateway health check")

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
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	c.logger.Debug("✅ Context Gateway health check passed")
	return nil
}

// Close closes the HTTP client connections
func (c *contextGatewayClient) Close() error {
	c.logger.Info("Closing Context Gateway client")
	
	// Close idle connections
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	c.logger.Info("✅ Context Gateway client closed")
	return nil
}

// makeHTTPRequest is a helper method for making HTTP requests with error handling
func (c *contextGatewayClient) makeHTTPRequest(ctx context.Context, method, url string, payload []byte) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Flow2-Go-Engine/1.0")

	// Make request with retry logic
	var lastErr error
	maxRetries := 3
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				c.logger.WithFields(logrus.Fields{
					"attempt": attempt + 1,
					"error":   err.Error(),
				}).Warn("Request failed, retrying...")
				
				// Exponential backoff
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
				continue
			}
			break
		}
		defer resp.Body.Close()

		// Read response body
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				continue
			}
			break
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return responseBody, nil
		}

		// Handle error status codes
		var errorResp map[string]interface{}
		if json.Unmarshal(responseBody, &errorResp) == nil {
			if detail, ok := errorResp["detail"].(string); ok {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, detail)
			} else {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))
			}
		} else {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))
		}

		// Don't retry for client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}

		if attempt < maxRetries {
			c.logger.WithFields(logrus.Fields{
				"attempt":     attempt + 1,
				"status_code": resp.StatusCode,
			}).Warn("Request failed, retrying...")
			
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, lastErr)
}

// Helper functions for request/response logging
func (c *contextGatewayClient) logRequest(method, url string, payload []byte) {
	c.logger.WithFields(logrus.Fields{
		"method":      method,
		"url":         url,
		"payload_size": len(payload),
	}).Debug("Making Context Gateway request")
}

func (c *contextGatewayClient) logResponse(statusCode int, responseSize int, duration time.Duration) {
	c.logger.WithFields(logrus.Fields{
		"status_code":    statusCode,
		"response_size":  responseSize,
		"duration_ms":    duration.Milliseconds(),
	}).Debug("Received Context Gateway response")
}