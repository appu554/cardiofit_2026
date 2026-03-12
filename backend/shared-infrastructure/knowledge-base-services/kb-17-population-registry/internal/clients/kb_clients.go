// Package clients provides HTTP clients for KB service integration
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/config"
	"kb-17-population-registry/internal/models"
)

// KB8Client provides access to KB-8 Calculator service
type KB8Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
	enabled    bool
}

// NewKB8Client creates a new KB-8 client
func NewKB8Client(cfg *config.KBClientConfig, logger *logrus.Entry) *KB8Client {
	return &KB8Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger:  logger.WithField("client", "kb8"),
		enabled: cfg.Enabled,
	}
}

// GetRiskScore retrieves a risk score for a patient
func (c *KB8Client) GetRiskScore(ctx context.Context, patientID, scoreType string) (*models.RiskScoreData, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/patients/%s/scores/%s", c.baseURL, patientID, scoreType)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Success bool                  `json:"success"`
		Data    *models.RiskScoreData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// CalculateScore calculates a risk score for a patient
func (c *KB8Client) CalculateScore(ctx context.Context, patientID, scoreType string, params map[string]interface{}) (*models.RiskScoreData, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/calculate/%s", c.baseURL, scoreType)

	body := map[string]interface{}{
		"patient_id": patientID,
		"parameters": params,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Success bool                  `json:"success"`
		Data    *models.RiskScoreData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// Health checks KB-8 health
func (c *KB8Client) Health(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-8 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// KB9Client provides access to KB-9 Care Gaps service
type KB9Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
	enabled    bool
}

// NewKB9Client creates a new KB-9 client
func NewKB9Client(cfg *config.KBClientConfig, logger *logrus.Entry) *KB9Client {
	return &KB9Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger:  logger.WithField("client", "kb9"),
		enabled: cfg.Enabled,
	}
}

// GetPatientCareGaps retrieves care gaps for a patient
func (c *KB9Client) GetPatientCareGaps(ctx context.Context, patientID string) ([]string, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/patients/%s/care-gaps", c.baseURL, patientID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Success bool     `json:"success"`
		Data    []string `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// UpdateCareGapStatus updates a care gap status
func (c *KB9Client) UpdateCareGapStatus(ctx context.Context, patientID, gapID, status string) error {
	if !c.enabled {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/patients/%s/care-gaps/%s", c.baseURL, patientID, gapID)

	body := map[string]string{
		"status": status,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Health checks KB-9 health
func (c *KB9Client) Health(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-9 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// KB14Client provides access to KB-14 Task Engine service
type KB14Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
	enabled    bool
}

// NewKB14Client creates a new KB-14 client
func NewKB14Client(cfg *config.KBClientConfig, logger *logrus.Entry) *KB14Client {
	return &KB14Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger:  logger.WithField("client", "kb14"),
		enabled: cfg.Enabled,
	}
}

// CreateTaskRequest represents a task creation request
type CreateTaskRequest struct {
	Type        string                 `json:"type"`
	Priority    string                 `json:"priority,omitempty"`
	Source      string                 `json:"source"`
	SourceID    string                 `json:"source_id,omitempty"`
	PatientID   string                 `json:"patient_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateTask creates a task in KB-14
func (c *KB14Client) CreateTask(ctx context.Context, req *CreateTaskRequest) (uuid.UUID, error) {
	if !c.enabled {
		return uuid.Nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/tasks", c.baseURL)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return uuid.Nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.ID, nil
}

// CreateEnrollmentTask creates a task for a new registry enrollment
func (c *KB14Client) CreateEnrollmentTask(ctx context.Context, enrollment *models.RegistryPatient) (uuid.UUID, error) {
	req := &CreateTaskRequest{
		Type:      "CARE_PLAN_REVIEW",
		Priority:  "MEDIUM",
		Source:    "KB17_REGISTRY",
		SourceID:  enrollment.ID.String(),
		PatientID: enrollment.PatientID,
		Title:     fmt.Sprintf("Initial %s Assessment", enrollment.RegistryCode),
		Description: fmt.Sprintf(
			"Patient enrolled in %s registry with %s risk tier. Initial assessment required.",
			enrollment.RegistryCode,
			enrollment.RiskTier,
		),
		Metadata: map[string]interface{}{
			"registry_code": enrollment.RegistryCode,
			"risk_tier":     enrollment.RiskTier,
			"enrolled_at":   enrollment.EnrolledAt,
		},
	}

	// Adjust priority based on risk tier
	if enrollment.RiskTier == models.RiskTierCritical {
		req.Priority = "CRITICAL"
		req.Type = "CRITICAL_LAB_REVIEW"
	} else if enrollment.RiskTier == models.RiskTierHigh {
		req.Priority = "HIGH"
	}

	return c.CreateTask(ctx, req)
}

// Health checks KB-14 health
func (c *KB14Client) Health(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-14 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
