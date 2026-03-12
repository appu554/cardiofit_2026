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

// KB8Client is a client for KB-8 Calculator Service
type KB8Client struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
	enabled    bool
}

// KB8Config holds KB-8 client configuration
type KB8Config struct {
	BaseURL string
	Timeout time.Duration
	Enabled bool
}

// NewKB8Client creates a new KB-8 Calculator Service client
func NewKB8Client(cfg KB8Config, log *logrus.Entry) *KB8Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &KB8Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log:     log.WithField("component", "kb8_client"),
		enabled: cfg.Enabled,
	}
}

// CalculateEGFRRequest is the request for eGFR calculation
type CalculateEGFRRequest struct {
	PatientID       string  `json:"patientId"`
	SerumCreatinine float64 `json:"serumCreatinine,omitempty"`
	Age             int     `json:"age,omitempty"`
	Sex             string  `json:"sex,omitempty"`
	Race            string  `json:"race,omitempty"`
}

// EGFRResult is the result from eGFR calculation
type EGFRResult struct {
	Value                        float64       `json:"value"`
	Unit                         string        `json:"unit"`
	CKDStage                     string        `json:"ckdStage"`
	CKDStageDisplay              string        `json:"ckdStageDisplay"`
	RequiresRenalDoseAdjustment  bool          `json:"requiresRenalDoseAdjustment"`
	DoseAdjustmentGuidance       string        `json:"doseAdjustmentGuidance"`
	Equation                     string        `json:"equation"`
	Inputs                       EGFRInputs    `json:"inputs"`
	Interpretation               string        `json:"interpretation"`
	Provenance                   *Provenance   `json:"provenance,omitempty"`
}

// EGFRInputs contains the input parameters used for eGFR calculation
type EGFRInputs struct {
	SerumCreatinine CreatinineInput `json:"serumCreatinine"`
	AgeYears        int             `json:"ageYears"`
	Sex             string          `json:"sex"`
}

// CreatinineInput represents creatinine input with value and unit
type CreatinineInput struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// CalculateAnionGapRequest is the request for anion gap calculation
type CalculateAnionGapRequest struct {
	PatientID  string  `json:"patientId"`
	Sodium     float64 `json:"sodium"`
	Chloride   float64 `json:"chloride"`
	Bicarbonate float64 `json:"bicarbonate"`
	Albumin    float64 `json:"albumin,omitempty"`
}

// AnionGapResult is the result from anion gap calculation
type AnionGapResult struct {
	Value             float64     `json:"value"`
	CorrectedValue    *float64    `json:"correctedValue,omitempty"`
	Unit              string      `json:"unit"`
	Interpretation    string      `json:"interpretation"`
	IsElevated        bool        `json:"isElevated"`
	IsCritical        bool        `json:"isCritical"`
	Provenance        *Provenance `json:"provenance,omitempty"`
}

// CalculateCorrectedCalciumRequest is the request for corrected calcium calculation
type CalculateCorrectedCalciumRequest struct {
	PatientID     string  `json:"patientId"`
	TotalCalcium  float64 `json:"totalCalcium"`
	Albumin       float64 `json:"albumin"`
}

// CorrectedCalciumResult is the result from corrected calcium calculation
type CorrectedCalciumResult struct {
	Value          float64     `json:"value"`
	Unit           string      `json:"unit"`
	Interpretation string      `json:"interpretation"`
	IsAbnormal     bool        `json:"isAbnormal"`
	Provenance     *Provenance `json:"provenance,omitempty"`
}

// BatchCalculateRequest is the request for batch calculation
type BatchCalculateRequest struct {
	PatientID              string           `json:"patientId"`
	Calculators            []CalculatorType `json:"calculators"`
	IncludeIndiaAdjustments bool            `json:"includeIndiaAdjustments,omitempty"`
	AsOf                   *time.Time       `json:"asOf,omitempty"`
}

// CalculatorType represents a type of calculator
type CalculatorType string

const (
	CalculatorEGFR          CalculatorType = "EGFR"
	CalculatorAnionGap      CalculatorType = "ANION_GAP"
	CalculatorCorrectedCa   CalculatorType = "CORRECTED_CALCIUM"
	CalculatorSOFA          CalculatorType = "SOFA"
	CalculatorQSOFA         CalculatorType = "QSOFA"
	CalculatorBMI           CalculatorType = "BMI"
	CalculatorCHA2DS2VASc   CalculatorType = "CHA2DS2_VASC"
)

// BatchCalculateResult is the result from batch calculation
type BatchCalculateResult struct {
	PatientID          string                `json:"patientId"`
	CalculatedAt       time.Time             `json:"calculatedAt"`
	OverallDataQuality string                `json:"overallDataQuality"`
	Summary            *CalculatorSummary    `json:"summary,omitempty"`
	Results            []CalculatorResult    `json:"results"`
	Failures           []CalculatorFailure   `json:"failures,omitempty"`
}

// CalculatorSummary provides a summary of key calculated values
type CalculatorSummary struct {
	EGFR                        *float64 `json:"egfr,omitempty"`
	CKDStage                    string   `json:"ckdStage,omitempty"`
	RequiresRenalDoseAdjustment bool     `json:"requiresRenalDoseAdjustment"`
	AnionGap                    *float64 `json:"anionGap,omitempty"`
	AnionGapElevated            bool     `json:"anionGapElevated"`
	SOFATotal                   *int     `json:"sofaTotal,omitempty"`
	SOFARiskLevel               string   `json:"sofaRiskLevel,omitempty"`
	QSOFAPositive               bool     `json:"qsofaPositive"`
	BMI                         *float64 `json:"bmi,omitempty"`
	BMICategory                 string   `json:"bmiCategory,omitempty"`
}

// CalculatorResult represents a single calculator result
type CalculatorResult struct {
	Type    CalculatorType `json:"type"`
	Success bool           `json:"success"`
	EGFR    *EGFRResult    `json:"egfr,omitempty"`
	AnionGap *AnionGapResult `json:"anionGap,omitempty"`
}

// CalculatorFailure represents a failed calculation
type CalculatorFailure struct {
	Type        CalculatorType `json:"type"`
	Error       string         `json:"error"`
	MissingData []string       `json:"missingData,omitempty"`
}

// Provenance contains CQL evaluation provenance for SaMD compliance
type Provenance struct {
	CQLLibrary    string            `json:"cqlLibrary"`
	CQLVersion    string            `json:"cqlVersion"`
	CQLExpression string            `json:"cqlExpression"`
	CalculatedAt  time.Time         `json:"calculatedAt"`
	DataSources   []DataSource      `json:"dataSources,omitempty"`
	DataQuality   string            `json:"dataQuality"`
	MissingData   []string          `json:"missingData,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
}

// DataSource represents a FHIR resource used in calculation
type DataSource struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Code         string `json:"code"`
	Display      string `json:"display"`
}

// IsEnabled returns whether KB-8 integration is enabled
func (c *KB8Client) IsEnabled() bool {
	return c.enabled && c.baseURL != ""
}

// CalculateEGFR calculates eGFR using KB-8's CKD-EPI 2021 formula
func (c *KB8Client) CalculateEGFR(ctx context.Context, req CalculateEGFRRequest) (*EGFRResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("KB-8 integration is not enabled")
	}

	endpoint := fmt.Sprintf("%s/api/v1/calculate/egfr", c.baseURL)

	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("eGFR calculation failed: %w", err)
	}

	var result EGFRResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse eGFR response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"patient_id": req.PatientID,
		"egfr":       result.Value,
		"ckd_stage":  result.CKDStage,
	}).Debug("KB-8 eGFR calculation completed")

	return &result, nil
}

// CalculateAnionGap calculates anion gap using KB-8
func (c *KB8Client) CalculateAnionGap(ctx context.Context, req CalculateAnionGapRequest) (*AnionGapResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("KB-8 integration is not enabled")
	}

	endpoint := fmt.Sprintf("%s/api/v1/calculate/anion-gap", c.baseURL)

	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("anion gap calculation failed: %w", err)
	}

	var result AnionGapResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse anion gap response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"patient_id":  req.PatientID,
		"anion_gap":   result.Value,
		"is_elevated": result.IsElevated,
	}).Debug("KB-8 anion gap calculation completed")

	return &result, nil
}

// CalculateCorrectedCalcium calculates albumin-corrected calcium using KB-8
func (c *KB8Client) CalculateCorrectedCalcium(ctx context.Context, req CalculateCorrectedCalciumRequest) (*CorrectedCalciumResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("KB-8 integration is not enabled")
	}

	endpoint := fmt.Sprintf("%s/api/v1/calculate/corrected-calcium", c.baseURL)

	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("corrected calcium calculation failed: %w", err)
	}

	var result CorrectedCalciumResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse corrected calcium response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"patient_id": req.PatientID,
		"corrected":  result.Value,
	}).Debug("KB-8 corrected calcium calculation completed")

	return &result, nil
}

// BatchCalculate performs multiple calculations in a single request
func (c *KB8Client) BatchCalculate(ctx context.Context, req BatchCalculateRequest) (*BatchCalculateResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("KB-8 integration is not enabled")
	}

	endpoint := fmt.Sprintf("%s/api/v1/calculate/batch", c.baseURL)

	resp, err := c.doRequest(ctx, "POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("batch calculation failed: %w", err)
	}

	var result BatchCalculateResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse batch response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"patient_id":    req.PatientID,
		"calculators":   len(req.Calculators),
		"results_count": len(result.Results),
		"failures":      len(result.Failures),
	}).Debug("KB-8 batch calculation completed")

	return &result, nil
}

// CheckHealth verifies KB-8 service availability
func (c *KB8Client) CheckHealth(ctx context.Context) error {
	if !c.IsEnabled() {
		return fmt.Errorf("KB-8 integration is not enabled")
	}

	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	_, err := c.doRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("KB-8 health check failed: %w", err)
	}

	return nil
}

// GetEGFRForPatient is a convenience method that fetches patient data and calculates eGFR
func (c *KB8Client) GetEGFRForPatient(ctx context.Context, patientID string) (*EGFRResult, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("KB-8 integration is not enabled")
	}

	// Use patientId-only request, KB-8 will fetch required data from FHIR
	req := CalculateEGFRRequest{
		PatientID: patientID,
	}

	return c.CalculateEGFR(ctx, req)
}

// doRequest performs an HTTP request to KB-8
func (c *KB8Client) doRequest(ctx context.Context, method, url string, body interface{}) ([]byte, error) {
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
		}).Warn("KB-8 request failed")
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// NOTE: All calculations are performed by KB-8 only. No local fallbacks are used.
// This ensures consistent CQL-based calculation with proper provenance for SaMD compliance.
