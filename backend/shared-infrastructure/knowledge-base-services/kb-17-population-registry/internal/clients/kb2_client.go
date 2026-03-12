// Package clients provides HTTP clients for KB service integration
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"kb-17-population-registry/internal/config"
	"kb-17-population-registry/internal/models"
)

// KB2Client provides access to KB-2 Clinical Context service
type KB2Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
	enabled    bool
}

// NewKB2Client creates a new KB-2 client
func NewKB2Client(cfg *config.KBClientConfig, logger *logrus.Entry) *KB2Client {
	return &KB2Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger:  logger.WithField("client", "kb2"),
		enabled: cfg.Enabled,
	}
}

// GetPatientContext retrieves clinical context for a patient from KB-2
func (c *KB2Client) GetPatientContext(ctx context.Context, patientID string) (*models.PatientClinicalData, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/patients/%s/context", c.baseURL, patientID)

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
		Success bool                       `json:"success"`
		Data    *models.PatientClinicalData `json:"data"`
		Error   string                     `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("KB-2 error: %s", result.Error)
	}

	return result.Data, nil
}

// GetDiagnoses retrieves diagnoses for a patient
func (c *KB2Client) GetDiagnoses(ctx context.Context, patientID string, activeOnly bool) ([]models.Diagnosis, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/patients/%s/diagnoses", c.baseURL, patientID)
	if activeOnly {
		url += "?status=active"
	}

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
		Success bool              `json:"success"`
		Data    []models.Diagnosis `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// GetLabResults retrieves lab results for a patient
func (c *KB2Client) GetLabResults(ctx context.Context, patientID string, since time.Time) ([]models.LabResult, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/patients/%s/labs", c.baseURL, patientID)
	if !since.IsZero() {
		url += fmt.Sprintf("?since=%s", since.Format(time.RFC3339))
	}

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
		Success bool              `json:"success"`
		Data    []models.LabResult `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// GetMedications retrieves medications for a patient
func (c *KB2Client) GetMedications(ctx context.Context, patientID string, activeOnly bool) ([]models.Medication, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/patients/%s/medications", c.baseURL, patientID)
	if activeOnly {
		url += "?status=active"
	}

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
		Success bool               `json:"success"`
		Data    []models.Medication `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, nil
}

// Health checks KB-2 health
func (c *KB2Client) Health(ctx context.Context) error {
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
		return fmt.Errorf("KB-2 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}
