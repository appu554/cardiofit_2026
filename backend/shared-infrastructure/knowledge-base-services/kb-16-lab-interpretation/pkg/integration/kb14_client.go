// Package integration provides clients for other KB services
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"kb-16-lab-interpretation/pkg/types"
)

// KB14Client is a client for KB-14 Care Navigator service
type KB14Client struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB14Client creates a new KB-14 client
func NewKB14Client(baseURL string, log *logrus.Entry) *KB14Client {
	return &KB14Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log.WithField("component", "kb14_client"),
	}
}

// CreateTaskRequest is the request to create a KB-14 task
type CreateTaskRequest struct {
	Type        string                 `json:"type"`
	Priority    string                 `json:"priority"`
	Source      string                 `json:"source"`
	SourceID    string                 `json:"source_id"`
	PatientID   string                 `json:"patient_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	SLAMinutes  int                    `json:"sla_minutes"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// TaskResponse is the response from KB-14
type TaskResponse struct {
	Success bool   `json:"success"`
	TaskID  string `json:"task_id"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// CreateCriticalLabTask creates a task for a critical lab result
func (c *KB14Client) CreateCriticalLabTask(ctx context.Context, result *types.InterpretedResult) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/tasks", c.baseURL)

	// Generate unique source ID to prevent duplicates
	sourceID := fmt.Sprintf("kb16-%s-%s-%d",
		result.Result.PatientID,
		result.Result.Code,
		result.Result.CollectedAt.Unix()/3600) // Hour-level dedup

	// Determine priority based on panic vs critical
	priority := "HIGH"
	taskType := "CRITICAL_LAB_REVIEW"
	slaMinutes := 60

	if result.Interpretation.IsPanic {
		priority = "CRITICAL"
		taskType = "PANIC_LAB_VALUE"
		slaMinutes = 30
	}

	// Format title
	valueStr := "N/A"
	if result.Result.ValueNumeric != nil {
		valueStr = fmt.Sprintf("%.2f", *result.Result.ValueNumeric)
	}
	title := fmt.Sprintf("[%s LAB] %s: %s %s",
		priority,
		result.Result.Name,
		valueStr,
		result.Result.Unit)

	req := CreateTaskRequest{
		Type:        taskType,
		Priority:    priority,
		Source:      "KB16_LAB_VALUES",
		SourceID:    sourceID,
		PatientID:   result.Result.PatientID,
		Title:       title,
		Description: result.Interpretation.ClinicalComment,
		SLAMinutes:  slaMinutes,
		Metadata: map[string]interface{}{
			"lab_code":       result.Result.Code,
			"lab_name":       result.Result.Name,
			"value":          result.Result.ValueNumeric,
			"unit":           result.Result.Unit,
			"flag":           result.Interpretation.Flag,
			"severity":       result.Interpretation.Severity,
			"is_panic":       result.Interpretation.IsPanic,
			"is_critical":    result.Interpretation.IsCritical,
			"kb16_result_id": result.Result.ID.String(),
			"collected_at":   result.Result.CollectedAt.Format(time.RFC3339),
			"recommendations": result.Interpretation.Recommendations,
		},
	}

	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return "", fmt.Errorf("failed to create task: %w", err)
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(resp, &taskResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !taskResp.Success {
		return "", fmt.Errorf("KB-14 error: %s", taskResp.Error)
	}

	c.log.WithFields(logrus.Fields{
		"task_id":    taskResp.TaskID,
		"patient_id": result.Result.PatientID,
		"lab_code":   result.Result.Code,
		"priority":   priority,
	}).Info("Created KB-14 task for critical lab")

	return taskResp.TaskID, nil
}

// CreateDeltaAlertTask creates a task for a significant delta change
func (c *KB14Client) CreateDeltaAlertTask(ctx context.Context, result *types.InterpretedResult) (string, error) {
	if result.Interpretation.DeltaCheck == nil || !result.Interpretation.DeltaCheck.IsSignificant {
		return "", nil // No significant delta
	}

	endpoint := fmt.Sprintf("%s/api/v1/tasks", c.baseURL)

	// Generate unique source ID
	sourceID := fmt.Sprintf("kb16-delta-%s-%s-%d",
		result.Result.PatientID,
		result.Result.Code,
		result.Result.CollectedAt.Unix()/3600)

	delta := result.Interpretation.DeltaCheck
	direction := "increased"
	if delta.Change < 0 {
		direction = "decreased"
	}

	title := fmt.Sprintf("[DELTA ALERT] %s %s by %.1f%%",
		result.Result.Name,
		direction,
		delta.PercentChange)

	description := fmt.Sprintf(
		"%s has %s from %.2f to %.2f %s (%.1f%% change in %d hours). %s",
		result.Result.Name,
		direction,
		delta.PreviousValue,
		*result.Result.ValueNumeric,
		result.Result.Unit,
		delta.PercentChange,
		delta.WindowHours,
		result.Interpretation.ClinicalComment,
	)

	req := CreateTaskRequest{
		Type:        "DELTA_LAB_ALERT",
		Priority:    "MEDIUM",
		Source:      "KB16_LAB_VALUES",
		SourceID:    sourceID,
		PatientID:   result.Result.PatientID,
		Title:       title,
		Description: description,
		SLAMinutes:  120, // 2 hour SLA for delta alerts
		Metadata: map[string]interface{}{
			"lab_code":        result.Result.Code,
			"lab_name":        result.Result.Name,
			"current_value":   result.Result.ValueNumeric,
			"previous_value":  delta.PreviousValue,
			"change":          delta.Change,
			"percent_change":  delta.PercentChange,
			"window_hours":    delta.WindowHours,
			"unit":            result.Result.Unit,
			"kb16_result_id":  result.Result.ID.String(),
		},
	}

	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return "", fmt.Errorf("failed to create delta task: %w", err)
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(resp, &taskResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !taskResp.Success {
		return "", fmt.Errorf("KB-14 error: %s", taskResp.Error)
	}

	return taskResp.TaskID, nil
}

// CheckHealth checks if KB-14 is available
func (c *KB14Client) CheckHealth(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	_, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("KB-14 health check failed: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID
func (c *KB14Client) GetTask(ctx context.Context, taskID string) (*TaskResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/tasks/%s", c.baseURL, taskID)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var taskResp TaskResponse
	if err := json.Unmarshal(resp, &taskResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &taskResp, nil
}

// doRequest performs an HTTP request
func (c *KB14Client) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Client-Service", "kb-16-lab-interpretation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		c.log.WithFields(logrus.Fields{
			"status":   resp.StatusCode,
			"url":      url,
			"response": string(respBody),
		}).Warn("KB-14 request failed")
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
