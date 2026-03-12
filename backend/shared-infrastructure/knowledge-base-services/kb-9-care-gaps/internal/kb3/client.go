// Package kb3 provides integration with KB-3 Temporal/Guidelines service.
package kb3

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

// ClientConfig holds configuration for the KB-3 client.
type ClientConfig struct {
	// BaseURL is the KB-3 service URL (e.g., http://kb-3-guidelines:8083)
	BaseURL string

	// Timeout for HTTP requests
	Timeout time.Duration

	// Enabled determines if KB-3 integration is active
	Enabled bool
}

// Client provides HTTP communication with KB-3 Temporal/Guidelines service.
type Client struct {
	config     ClientConfig
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new KB-3 client.
func NewClient(config ClientConfig, logger *zap.Logger) *Client {
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
	}
}

// IsEnabled returns true if KB-3 integration is enabled.
func (c *Client) IsEnabled() bool {
	return c.config.Enabled
}

// GetPatientSchedule retrieves all scheduled items for a patient.
// Endpoint: GET /v1/patients/{patientID}/schedule
func (c *Client) GetPatientSchedule(ctx context.Context, patientID string) (*ScheduleResponse, error) {
	if !c.config.Enabled {
		c.logger.Debug("KB-3 integration disabled, returning empty schedule")
		return &ScheduleResponse{Items: []ScheduledItem{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/v1/patients/%s/schedule", c.config.BaseURL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Fetching patient schedule from KB-3",
		zap.String("patient_id", patientID),
		zap.String("url", url),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("KB-3 request failed",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("KB-3 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("KB-3 returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("KB-3 returned status %d: %s", resp.StatusCode, string(body))
	}

	// KB-3 returns a plain array, not wrapped in a struct
	var items []ScheduledItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode KB-3 response: %w", err)
	}

	result := &ScheduleResponse{
		Items: items,
		Total: len(items),
	}

	c.logger.Debug("Retrieved patient schedule",
		zap.String("patient_id", patientID),
		zap.Int("items", result.Total),
	)

	return result, nil
}

// CreateScheduledItem creates a new scheduled care item.
// Endpoint: POST /v1/schedule/{patientID}/add
func (c *Client) CreateScheduledItem(ctx context.Context, patientID string, request *ScheduleRequest) (*ScheduledItem, error) {
	if !c.config.Enabled {
		c.logger.Debug("KB-3 integration disabled, skipping schedule creation")
		return nil, nil
	}

	url := fmt.Sprintf("%s/v1/schedule/%s/add", c.config.BaseURL, patientID)

	// Ensure request has patientID
	request.PatientID = patientID

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Creating scheduled item in KB-3",
		zap.String("patient_id", patientID),
		zap.String("type", string(request.Type)),
		zap.String("name", request.Name),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("KB-3 create schedule request failed", zap.Error(err))
		return nil, fmt.Errorf("KB-3 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("KB-3 create schedule returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("KB-3 returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ScheduledItem
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode KB-3 response: %w", err)
	}

	c.logger.Info("Created scheduled item in KB-3",
		zap.String("item_id", result.ItemID),
		zap.String("patient_id", patientID),
		zap.String("type", string(result.Type)),
	)

	return &result, nil
}

// GetOverdueAlerts retrieves system-wide overdue items.
// Endpoint: GET /v1/alerts/overdue
// KB-3 returns: {"pathway_overdue": [...], "scheduling_overdue": [...], "total_count": N}
func (c *Client) GetOverdueAlerts(ctx context.Context) (*OverdueAlertsResponse, error) {
	if !c.config.Enabled {
		c.logger.Debug("KB-3 integration disabled, returning empty alerts")
		return &OverdueAlertsResponse{Alerts: []OverdueAlert{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/v1/alerts/overdue", c.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Fetching overdue alerts from KB-3", zap.String("url", url))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("KB-3 overdue alerts request failed", zap.Error(err))
		return nil, fmt.Errorf("KB-3 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-3 returned status %d: %s", resp.StatusCode, string(body))
	}

	// KB-3 returns a different structure for system-wide alerts
	var kb3Response struct {
		PathwayOverdue    []PathwayOverdueItem `json:"pathway_overdue"`
		SchedulingOverdue []ScheduledItem      `json:"scheduling_overdue"`
		TotalCount        int                  `json:"total_count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&kb3Response); err != nil {
		return nil, fmt.Errorf("failed to decode KB-3 response: %w", err)
	}

	// Convert to our OverdueAlert format
	var alerts []OverdueAlert
	for _, item := range kb3Response.PathwayOverdue {
		// Convert nanoseconds overdue to days
		daysOverdue := int(item.OverdueBy / (24 * 60 * 60 * 1e9))
		alerts = append(alerts, OverdueAlert{
			ItemID:      item.ActionID,
			PatientID:   item.PatientID,
			Name:        item.ActionName,
			DueDate:     item.Deadline,
			DaysOverdue: daysOverdue,
			Severity:    item.Severity,
		})
	}
	for _, item := range kb3Response.SchedulingOverdue {
		alerts = append(alerts, OverdueAlert{
			ItemID:        item.ItemID,
			PatientID:     item.PatientID,
			Type:          item.Type,
			Name:          item.Name,
			DueDate:       item.DueDate,
			Priority:      item.Priority,
			SourceMeasure: item.SourceMeasureID,
		})
	}

	result := &OverdueAlertsResponse{
		Alerts: alerts,
		Total:  kb3Response.TotalCount,
	}

	c.logger.Debug("Retrieved overdue alerts", zap.Int("count", result.Total))

	return result, nil
}

// GetPatientOverdueAlerts retrieves overdue items for a specific patient.
// Endpoint: GET /v1/patients/{patientID}/overdue
// KB-3 returns a plain array of scheduled items
func (c *Client) GetPatientOverdueAlerts(ctx context.Context, patientID string) (*OverdueAlertsResponse, error) {
	if !c.config.Enabled {
		return &OverdueAlertsResponse{Alerts: []OverdueAlert{}, Total: 0}, nil
	}

	url := fmt.Sprintf("%s/v1/patients/%s/overdue", c.config.BaseURL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("KB-3 patient overdue alerts request failed",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("KB-3 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-3 returned status %d: %s", resp.StatusCode, string(body))
	}

	// KB-3 returns a plain array of scheduled items for patient overdue
	var items []ScheduledItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode KB-3 response: %w", err)
	}

	// Convert to OverdueAlert format
	var alerts []OverdueAlert
	now := time.Now()
	for _, item := range items {
		daysOverdue := int(now.Sub(item.DueDate).Hours() / 24)
		alerts = append(alerts, OverdueAlert{
			ItemID:        item.ItemID,
			PatientID:     item.PatientID,
			Type:          item.Type,
			Name:          item.Name,
			DueDate:       item.DueDate,
			DaysOverdue:   daysOverdue,
			Priority:      item.Priority,
			SourceMeasure: item.SourceMeasureID,
		})
	}

	return &OverdueAlertsResponse{
		Alerts: alerts,
		Total:  len(alerts),
	}, nil
}

// GetScheduleItemsByMeasure retrieves scheduled items for a specific measure.
// This filters items by their source_measure_id field.
func (c *Client) GetScheduleItemsByMeasure(ctx context.Context, patientID string, measureID string) ([]ScheduledItem, error) {
	schedule, err := c.GetPatientSchedule(ctx, patientID)
	if err != nil {
		return nil, err
	}

	// Filter by measure ID
	var items []ScheduledItem
	for _, item := range schedule.Items {
		if item.SourceMeasureID == measureID {
			items = append(items, item)
		}
	}

	return items, nil
}

// CompleteScheduledItem marks a scheduled item as complete.
// Endpoint: POST /v1/schedule/{patientID}/complete
func (c *Client) CompleteScheduledItem(ctx context.Context, patientID string, itemID string) error {
	if !c.config.Enabled {
		return nil
	}

	url := fmt.Sprintf("%s/v1/schedule/%s/complete", c.config.BaseURL, patientID)

	// KB-3 expects itemID in the request body
	body, err := json.Marshal(map[string]string{"item_id": itemID})
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	c.logger.Debug("Completing scheduled item in KB-3",
		zap.String("patient_id", patientID),
		zap.String("item_id", itemID),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-3 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("KB-3 returned status %d: %s", resp.StatusCode, string(body))
	}

	c.logger.Info("Completed scheduled item in KB-3",
		zap.String("patient_id", patientID),
		zap.String("item_id", itemID),
	)

	return nil
}

// HealthCheck verifies KB-3 service is reachable.
// Endpoint: GET /health
func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	url := fmt.Sprintf("%s/health", c.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-3 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-3 health check returned status %d", resp.StatusCode)
	}

	return nil
}
