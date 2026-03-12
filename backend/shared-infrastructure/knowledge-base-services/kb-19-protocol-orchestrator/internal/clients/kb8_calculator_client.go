// Package clients provides HTTP clients for KB-19 to communicate with upstream services.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// KB8CalculatorClient is the HTTP client for KB-8 Clinical Calculator service.
// KB-8 provides clinical risk score calculations like CHA2DS2-VASc, SOFA, eGFR, etc.
type KB8CalculatorClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB8CalculatorClient creates a new KB8CalculatorClient.
func NewKB8CalculatorClient(baseURL string, timeout time.Duration, log *logrus.Entry) *KB8CalculatorClient {
	return &KB8CalculatorClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.WithField("client", "kb8-calculator"),
	}
}

// CalculatorRequest is the request to calculate a clinical score.
type CalculatorRequest struct {
	PatientID     uuid.UUID              `json:"patient_id"`
	CalculatorID  string                 `json:"calculator_id"`
	Inputs        map[string]interface{} `json:"inputs"`
	UseLatestLabs bool                   `json:"use_latest_labs"`
}

// CalculatorResult is the result of a clinical calculation.
type CalculatorResult struct {
	CalculatorID   string                 `json:"calculator_id"`
	CalculatorName string                 `json:"calculator_name"`
	Score          float64                `json:"score"`
	ScoreLabel     string                 `json:"score_label,omitempty"`    // e.g., "High Risk"
	Interpretation string                 `json:"interpretation,omitempty"`
	RiskCategory   string                 `json:"risk_category,omitempty"` // LOW, MODERATE, HIGH, VERY_HIGH
	Percentile     float64                `json:"percentile,omitempty"`
	Components     map[string]float64     `json:"components,omitempty"`    // Individual component scores
	InputsUsed     map[string]interface{} `json:"inputs_used"`
	CalculatedAt   time.Time              `json:"calculated_at"`
	Formula        string                 `json:"formula,omitempty"`
	Citation       string                 `json:"citation,omitempty"`
}

// BatchCalculatorRequest is the request to calculate multiple scores.
type BatchCalculatorRequest struct {
	PatientID      uuid.UUID `json:"patient_id"`
	CalculatorIDs  []string  `json:"calculator_ids"`
	UseLatestLabs  bool      `json:"use_latest_labs"`
}

// BatchCalculatorResult is the result of batch calculations.
type BatchCalculatorResult struct {
	PatientID   uuid.UUID                   `json:"patient_id"`
	Results     map[string]CalculatorResult `json:"results"`
	Errors      map[string]string           `json:"errors,omitempty"`
	CalculatedAt time.Time                  `json:"calculated_at"`
}

// AvailableCalculator describes an available calculator.
type AvailableCalculator struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`     // CARDIAC, RENAL, SEPSIS, BLEEDING, etc.
	RequiredInputs []string `json:"required_inputs"`
	OptionalInputs []string `json:"optional_inputs"`
	Description  string   `json:"description"`
	Citation     string   `json:"citation"`
}

// Common calculator IDs
const (
	CalcCHA2DS2VASc        = "CHA2DS2-VASc"
	CalcHASBLED            = "HAS-BLED"
	CalcSOFA               = "SOFA"
	CalcQSOFA              = "qSOFA"
	CalcEGFR               = "eGFR"
	CalcCrCl               = "CrCl"
	CalcMELD               = "MELD"
	CalcChildPugh          = "Child-Pugh"
	CalcAPACHEII           = "APACHE-II"
	CalcASCVD              = "ASCVD"
	CalcFramingham         = "Framingham"
	CalcWells              = "Wells-DVT"
	CalcWellsPE            = "Wells-PE"
	CalcGenevaScore        = "Geneva"
	CalcCURB65             = "CURB-65"
	CalcPESI               = "PESI"
	CalcTIMI               = "TIMI"
	CalcGRACE              = "GRACE"
	CalcCRUSADE            = "CRUSADE"
)

// Calculate calculates a single clinical score.
func (c *KB8CalculatorClient) Calculate(ctx context.Context, req CalculatorRequest) (*CalculatorResult, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":    req.PatientID,
		"calculator_id": req.CalculatorID,
	}).Debug("Calculating clinical score")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/calculate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("calculation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result CalculatorResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"calculator_id": req.CalculatorID,
		"score":         result.Score,
		"risk_category": result.RiskCategory,
	}).Debug("Calculation complete")

	return &result, nil
}

// CalculateBatch calculates multiple clinical scores for a patient.
func (c *KB8CalculatorClient) CalculateBatch(ctx context.Context, req BatchCalculatorRequest) (*BatchCalculatorResult, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":     req.PatientID,
		"calculator_count": len(req.CalculatorIDs),
	}).Debug("Calculating batch clinical scores")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/calculate/batch", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch calculation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result BatchCalculatorResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"patient_id":      req.PatientID,
		"results_count":   len(result.Results),
		"errors_count":    len(result.Errors),
	}).Debug("Batch calculation complete")

	return &result, nil
}

// GetCalculatorScores retrieves all relevant calculator scores for a patient.
// This is a convenience method that calculates commonly used scores.
func (c *KB8CalculatorClient) GetCalculatorScores(ctx context.Context, patientID uuid.UUID) (map[string]float64, error) {
	commonCalculators := []string{
		CalcEGFR,
		CalcCrCl,
		CalcCHA2DS2VASc,
		CalcHASBLED,
		CalcSOFA,
		CalcQSOFA,
	}

	result, err := c.CalculateBatch(ctx, BatchCalculatorRequest{
		PatientID:     patientID,
		CalculatorIDs: commonCalculators,
		UseLatestLabs: true,
	})
	if err != nil {
		return nil, err
	}

	scores := make(map[string]float64)
	for calcID, calcResult := range result.Results {
		scores[calcID] = calcResult.Score
	}

	return scores, nil
}

// ListCalculators lists all available calculators.
func (c *KB8CalculatorClient) ListCalculators(ctx context.Context) ([]AvailableCalculator, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/calculators", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list calculators failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result []AvailableCalculator
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// Health checks if KB-8 is healthy.
func (c *KB8CalculatorClient) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
