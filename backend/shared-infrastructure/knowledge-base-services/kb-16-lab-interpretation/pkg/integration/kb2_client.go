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
)

// KB2Client is a client for KB-2 Clinical Context service
type KB2Client struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB2Client creates a new KB-2 client
func NewKB2Client(baseURL string, log *logrus.Entry) *KB2Client {
	return &KB2Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log.WithField("component", "kb2_client"),
	}
}

// PatientContext represents patient context from KB-2
type PatientContext struct {
	PatientID   string                 `json:"patient_id"`
	Age         int                    `json:"age"`
	Sex         string                 `json:"sex"`
	Conditions  []Condition            `json:"conditions"`
	Medications []Medication           `json:"medications"`
	Phenotypes  []string               `json:"phenotypes"`
	RiskFactors map[string]interface{} `json:"risk_factors"`
}

// Condition represents a patient condition
type Condition struct {
	Code        string `json:"code"`
	Display     string `json:"display"`
	Status      string `json:"status"`
	OnsetDate   string `json:"onset_date,omitempty"`
}

// Medication represents a patient medication
type Medication struct {
	Code       string `json:"code"`
	Display    string `json:"display"`
	Status     string `json:"status"`
	DosageText string `json:"dosage_text,omitempty"`
}

// BuildContextRequest is the request to KB-2 to build patient context
type BuildContextRequest struct {
	PatientID   string                 `json:"patient_id"`
	Patient     map[string]interface{} `json:"patient,omitempty"`
	Conditions  []map[string]interface{} `json:"conditions,omitempty"`
	Medications []map[string]interface{} `json:"medications,omitempty"`
	Observations []map[string]interface{} `json:"observations,omitempty"`
}

// BuildContextResponse is the response from KB-2
type BuildContextResponse struct {
	Success   bool                   `json:"success"`
	Context   ContextData            `json:"context"`
	Phenotypes []string              `json:"phenotypes"`
	Error     string                 `json:"error,omitempty"`
}

// ContextData represents the context data from KB-2
type ContextData struct {
	Demographics    Demographics    `json:"demographics"`
	ActiveConditions []Condition   `json:"active_conditions"`
	CurrentMeds      []Medication  `json:"current_medications"`
	RiskFactors      map[string]interface{} `json:"risk_factors"`
}

// Demographics represents patient demographics
type Demographics struct {
	AgeYears int    `json:"age_years"`
	Sex      string `json:"sex"`
}

// GetPatientContext retrieves patient context from KB-2
func (c *KB2Client) GetPatientContext(ctx context.Context, patientID string) (*PatientContext, error) {
	endpoint := fmt.Sprintf("%s/api/v1/context/build", c.baseURL)

	req := BuildContextRequest{
		PatientID: patientID,
		Patient:   map[string]interface{}{},
	}

	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient context: %w", err)
	}

	var buildResp BuildContextResponse
	if err := json.Unmarshal(resp, &buildResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !buildResp.Success && buildResp.Error != "" {
		return nil, fmt.Errorf("KB-2 error: %s", buildResp.Error)
	}

	return &PatientContext{
		PatientID:   patientID,
		Age:         buildResp.Context.Demographics.AgeYears,
		Sex:         buildResp.Context.Demographics.Sex,
		Conditions:  buildResp.Context.ActiveConditions,
		Medications: buildResp.Context.CurrentMeds,
		Phenotypes:  buildResp.Phenotypes,
		RiskFactors: buildResp.Context.RiskFactors,
	}, nil
}

// GetPatientContextSimple retrieves basic patient context by ID
func (c *KB2Client) GetPatientContextSimple(ctx context.Context, patientID string) (*PatientContext, error) {
	endpoint := fmt.Sprintf("%s/api/v1/patients/%s/context", c.baseURL, patientID)

	resp, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient context: %w", err)
	}

	var patientCtx PatientContext
	if err := json.Unmarshal(resp, &patientCtx); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &patientCtx, nil
}

// CheckHealth checks if KB-2 is available
func (c *KB2Client) CheckHealth(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	_, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("KB-2 health check failed: %w", err)
	}

	return nil
}

// doRequest performs an HTTP request
func (c *KB2Client) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
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
		}).Warn("KB-2 request failed")
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
