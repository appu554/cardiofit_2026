package clients

import (
	//"bytes"
	"context"
	//"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// ContextGatewayClient handles communication with the Context Gateway service
type ContextGatewayClient struct {
	client  *resty.Client
	baseURL string
	logger  *zap.Logger
}

// NewContextGatewayClient creates a new Context Gateway client
func NewContextGatewayClient(baseURL string, logger *zap.Logger) *ContextGatewayClient {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetRetryCount(3)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)
	
	// Add retry conditions
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		return r.StatusCode() >= 500 || err != nil
	})

	// Add request/response logging
	client.OnBeforeRequest(func(c *resty.Client, req *resty.Request) error {
		logger.Debug("Context Gateway request",
			zap.String("method", req.Method),
			zap.String("url", req.URL),
		)
		return nil
	})

	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		logger.Debug("Context Gateway response",
			zap.String("url", resp.Request.URL),
			zap.Int("status", resp.StatusCode()),
			zap.Duration("time", resp.Time()),
		)
		return nil
	})

	return &ContextGatewayClient{
		client:  client,
		baseURL: baseURL,
		logger:  logger,
	}
}

// CreateSnapshotRequest represents a request to create a clinical snapshot
type CreateSnapshotRequest struct {
	PatientID             string                    `json:"patient_id"`
	RecipeID              string                    `json:"recipe_id"`
	SnapshotType          string                    `json:"snapshot_type"`
	FreshnessRequirements map[string]time.Duration `json:"freshness_requirements"`
	RequiredFields        []string                  `json:"required_fields"`
	OptionalFields        []string                  `json:"optional_fields,omitempty"`
	Priority              string                    `json:"priority,omitempty"`
	ExpiryDuration        time.Duration             `json:"expiry_duration,omitempty"`
}

// CreateSnapshotResponse represents the response from creating a snapshot
type CreateSnapshotResponse struct {
	SnapshotID       string                 `json:"snapshot_id"`
	Status           string                 `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	ExpiresAt        time.Time              `json:"expires_at"`
	DataSources      map[string]interface{} `json:"data_sources"`
	FreshnessStatus  map[string]string      `json:"freshness_status"`
	QualityScore     float64                `json:"quality_score"`
	ProcessingTime   time.Duration          `json:"processing_time"`
	Warnings         []string               `json:"warnings,omitempty"`
}

// GetSnapshotResponse represents a clinical snapshot
type GetSnapshotResponse struct {
	SnapshotID        string                 `json:"snapshot_id"`
	PatientID         string                 `json:"patient_id"`
	RecipeID          string                 `json:"recipe_id"`
	Status            string                 `json:"status"`
	ClinicalData      map[string]interface{} `json:"clinical_data"`
	FreshnessMetadata map[string]interface{} `json:"freshness_metadata"`
	ValidationResults map[string]interface{} `json:"validation_results"`
	CreatedAt         time.Time              `json:"created_at"`
	ExpiresAt         time.Time              `json:"expires_at"`
	Hash              string                 `json:"hash"`
	Version           int                    `json:"version"`
}

// CreateSnapshot creates a new clinical snapshot through the Context Gateway
func (c *ContextGatewayClient) CreateSnapshot(ctx context.Context, req *CreateSnapshotRequest) (*CreateSnapshotResponse, error) {
	var response CreateSnapshotResponse
	
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&response).
		SetError(&APIError{}).
		Post(c.baseURL + "/api/v1/snapshots")

	if err != nil {
		c.logger.Error("Failed to create snapshot", zap.Error(err))
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	if resp.IsError() {
		apiErr := resp.Error().(*APIError)
		c.logger.Error("Context Gateway API error",
			zap.Int("status", resp.StatusCode()),
			zap.String("error", apiErr.Message),
		)
		return nil, fmt.Errorf("context gateway error: %s", apiErr.Message)
	}

	c.logger.Info("Snapshot created successfully",
		zap.String("snapshot_id", response.SnapshotID),
		zap.String("patient_id", req.PatientID),
		zap.Duration("processing_time", response.ProcessingTime),
	)

	return &response, nil
}

// GetSnapshot retrieves a clinical snapshot by ID
func (c *ContextGatewayClient) GetSnapshot(ctx context.Context, snapshotID string) (*GetSnapshotResponse, error) {
	var response GetSnapshotResponse
	
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetError(&APIError{}).
		Get(c.baseURL + "/api/v1/snapshots/" + snapshotID)

	if err != nil {
		c.logger.Error("Failed to get snapshot", zap.String("snapshot_id", snapshotID), zap.Error(err))
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return nil, ErrSnapshotNotFound
		}
		
		apiErr := resp.Error().(*APIError)
		return nil, fmt.Errorf("context gateway error: %s", apiErr.Message)
	}

	return &response, nil
}

// ValidateSnapshotRequest represents a request to validate a snapshot
type ValidateSnapshotRequest struct {
	SnapshotID      string            `json:"snapshot_id"`
	ValidationLevel string            `json:"validation_level"`
	CheckTypes      []string          `json:"check_types"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// ValidateSnapshotResponse represents snapshot validation results
type ValidateSnapshotResponse struct {
	SnapshotID        string                 `json:"snapshot_id"`
	IsValid           bool                   `json:"is_valid"`
	ValidationScore   float64                `json:"validation_score"`
	ValidationResults map[string]interface{} `json:"validation_results"`
	Errors            []string               `json:"errors,omitempty"`
	Warnings          []string               `json:"warnings,omitempty"`
	ValidatedAt       time.Time              `json:"validated_at"`
}

// ValidateSnapshot validates a clinical snapshot
func (c *ContextGatewayClient) ValidateSnapshot(ctx context.Context, req *ValidateSnapshotRequest) (*ValidateSnapshotResponse, error) {
	var response ValidateSnapshotResponse
	
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetResult(&response).
		SetError(&APIError{}).
		Post(c.baseURL + "/api/v1/snapshots/" + req.SnapshotID + "/validate")

	if err != nil {
		c.logger.Error("Failed to validate snapshot", zap.String("snapshot_id", req.SnapshotID), zap.Error(err))
		return nil, fmt.Errorf("failed to validate snapshot: %w", err)
	}

	if resp.IsError() {
		apiErr := resp.Error().(*APIError)
		return nil, fmt.Errorf("context gateway validation error: %s", apiErr.Message)
	}

	return &response, nil
}

// SupersedeSnapshotRequest represents a request to supersede a snapshot
type SupersedeSnapshotRequest struct {
	OldSnapshotID string `json:"old_snapshot_id"`
	NewSnapshotID string `json:"new_snapshot_id"`
	Reason        string `json:"reason"`
	SupersededBy  string `json:"superseded_by"`
}

// SupersedeSnapshot marks an old snapshot as superseded by a new one
func (c *ContextGatewayClient) SupersedeSnapshot(ctx context.Context, req *SupersedeSnapshotRequest) error {
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		SetError(&APIError{}).
		Post(c.baseURL + "/api/v1/snapshots/" + req.OldSnapshotID + "/supersede")

	if err != nil {
		c.logger.Error("Failed to supersede snapshot", zap.String("snapshot_id", req.OldSnapshotID), zap.Error(err))
		return fmt.Errorf("failed to supersede snapshot: %w", err)
	}

	if resp.IsError() {
		apiErr := resp.Error().(*APIError)
		return fmt.Errorf("context gateway supersede error: %s", apiErr.Message)
	}

	c.logger.Info("Snapshot superseded successfully",
		zap.String("old_snapshot_id", req.OldSnapshotID),
		zap.String("new_snapshot_id", req.NewSnapshotID),
		zap.String("reason", req.Reason),
	)

	return nil
}

// ListSnapshotsRequest represents a request to list snapshots
type ListSnapshotsRequest struct {
	PatientID    string    `json:"patient_id,omitempty"`
	Status       string    `json:"status,omitempty"`
	CreatedAfter time.Time `json:"created_after,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	Offset       int       `json:"offset,omitempty"`
}

// ListSnapshotsResponse represents a list of snapshots
type ListSnapshotsResponse struct {
	Snapshots  []GetSnapshotResponse `json:"snapshots"`
	TotalCount int                   `json:"total_count"`
	HasMore    bool                  `json:"has_more"`
}

// ListSnapshots retrieves a list of snapshots based on criteria
func (c *ContextGatewayClient) ListSnapshots(ctx context.Context, req *ListSnapshotsRequest) (*ListSnapshotsResponse, error) {
	var response ListSnapshotsResponse
	
	request := c.client.R().
		SetContext(ctx).
		SetResult(&response).
		SetError(&APIError{})

	// Add query parameters
	if req.PatientID != "" {
		request.SetQueryParam("patient_id", req.PatientID)
	}
	if req.Status != "" {
		request.SetQueryParam("status", req.Status)
	}
	if !req.CreatedAfter.IsZero() {
		request.SetQueryParam("created_after", req.CreatedAfter.Format(time.RFC3339))
	}
	if req.Limit > 0 {
		request.SetQueryParam("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Offset > 0 {
		request.SetQueryParam("offset", fmt.Sprintf("%d", req.Offset))
	}

	resp, err := request.Get(c.baseURL + "/api/v1/snapshots")

	if err != nil {
		c.logger.Error("Failed to list snapshots", zap.Error(err))
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	if resp.IsError() {
		apiErr := resp.Error().(*APIError)
		return nil, fmt.Errorf("context gateway list error: %s", apiErr.Message)
	}

	return &response, nil
}

// HealthCheck checks the health of the Context Gateway service
func (c *ContextGatewayClient) HealthCheck(ctx context.Context) (*ServiceHealth, error) {
	var health ServiceHealth
	
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&health).
		Get(c.baseURL + "/health/ready")

	if err != nil {
		return &ServiceHealth{
			Status: "unhealthy",
			Error:  err.Error(),
		}, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return &ServiceHealth{
			Status: "unhealthy",
			Error:  fmt.Sprintf("HTTP %d", resp.StatusCode()),
		}, nil
	}

	return &health, nil
}

// GetMetrics retrieves metrics from the Context Gateway service
func (c *ContextGatewayClient) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	var metrics map[string]interface{}
	
	resp, err := c.client.R().
		SetContext(ctx).
		SetResult(&metrics).
		Get(c.baseURL + "/metrics/json")

	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if resp.IsError() {
		return nil, fmt.Errorf("metrics endpoint error: HTTP %d", resp.StatusCode())
	}

	return metrics, nil
}

// SetTimeout sets the client timeout
func (c *ContextGatewayClient) SetTimeout(timeout time.Duration) {
	c.client.SetTimeout(timeout)
}

// SetRetryPolicy sets the retry policy
func (c *ContextGatewayClient) SetRetryPolicy(retryCount int, waitTime, maxWaitTime time.Duration) {
	c.client.SetRetryCount(retryCount)
	c.client.SetRetryWaitTime(waitTime)
	c.client.SetRetryMaxWaitTime(maxWaitTime)
}

// Common errors
var (
	ErrSnapshotNotFound = fmt.Errorf("snapshot not found")
	ErrInvalidSnapshot  = fmt.Errorf("invalid snapshot")
	ErrSnapshotExpired  = fmt.Errorf("snapshot expired")
)

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Status       string        `json:"status"`
	ResponseTime time.Duration `json:"response_time,omitempty"`
	Error        string        `json:"error,omitempty"`
	Version      string        `json:"version,omitempty"`
}

// APIError represents an API error response
type APIError struct {
	Message string            `json:"message"`
	Code    string            `json:"code,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}