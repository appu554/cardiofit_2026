package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/config"
)

// KB25Client communicates with the KB-25 Lifestyle Knowledge Graph service (port 8136).
// It is used by ProtocolService to:
//   - Validate that no hard-stop lifestyle safety rules block a protocol before activation.
//   - Obtain projected outcomes for the new phase after a successful phase transition.
//
// All calls are best-effort on the projection path — errors are logged but do not
// prevent the mechanical phase transition from completing.
type KB25Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB25Client creates a client for the KB-25 Lifestyle Knowledge Graph service.
func NewKB25Client(cfg config.KB25Config, logger *zap.Logger) *KB25Client {
	return &KB25Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// SafetyCheckRequest is the payload sent to KB-25 POST /v1/safety/check.
type SafetyCheckRequest struct {
	PatientID  string             `json:"patient_id"`
	ProtocolID string             `json:"protocol_id"`
	Conditions map[string]float64 `json:"conditions"`
}

// SafetyCheckResponse is returned by KB-25 /v1/safety/check.
type SafetyCheckResponse struct {
	Safe     bool   `json:"safe"`
	RuleCode string `json:"rule_code,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// CheckSafety calls KB-25 to verify that no hard-stop lifestyle safety rules block
// the given protocol for the patient. Returns an error if the HTTP call fails or
// if KB-25 returns a non-200 status; returns the response (Safe=false) if KB-25
// identifies a blocking rule.
func (c *KB25Client) CheckSafety(req SafetyCheckRequest) (*SafetyCheckResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal KB-25 safety check request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/v1/safety/check", c.baseURL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("KB-25 safety check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-25 safety check returned status %d", resp.StatusCode)
	}

	var result SafetyCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode KB-25 safety check response: %w", err)
	}

	if !result.Safe {
		c.logger.Warn("KB-25 safety check blocked protocol activation",
			zap.String("patient_id", req.PatientID),
			zap.String("protocol_id", req.ProtocolID),
			zap.String("rule_code", result.RuleCode),
			zap.String("reason", result.Reason),
		)
	}

	return &result, nil
}

// ProjectionRequest is the payload sent to KB-25 POST /v1/project-combined.
type ProjectionRequest struct {
	PatientID   string   `json:"patient_id"`
	ProtocolIDs []string `json:"protocol_ids"`
	HorizonDays int      `json:"horizon_days"`
	Age         int      `json:"age,omitempty"`
	EGFR        float64  `json:"egfr,omitempty"`
	BMI         float64  `json:"bmi,omitempty"`
	Adherence   float64  `json:"adherence,omitempty"`
}

// ProjectionResponse is returned by KB-25 /v1/project-combined.
type ProjectionResponse struct {
	Projections []ProjectedOutcome `json:"projections"`
	Synergy     float64            `json:"synergy_multiplier,omitempty"`
}

// ProjectedOutcome is a single projected metric within a ProjectionResponse.
type ProjectedOutcome struct {
	Metric        string  `json:"metric"`
	BaselineDelta float64 `json:"baseline_delta"`
	Confidence    float64 `json:"confidence"`
}

// ProjectCombined calls KB-25 to obtain projected outcomes for the given protocols
// and patient context over the specified horizon. Errors are returned so callers
// on the phase-transition path can decide whether to treat them as warnings.
func (c *KB25Client) ProjectCombined(req ProjectionRequest) (*ProjectionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal KB-25 projection request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/v1/project-combined", c.baseURL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("KB-25 projection request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-25 projection returned status %d", resp.StatusCode)
	}

	var result ProjectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode KB-25 projection response: %w", err)
	}

	c.logger.Debug("KB-25 projection received",
		zap.String("patient_id", req.PatientID),
		zap.Int("projection_count", len(result.Projections)),
		zap.Float64("synergy_multiplier", result.Synergy),
	)

	return &result, nil
}

// HealthCheck verifies KB-25 is reachable.
func (c *KB25Client) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("KB-25 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-25 health check returned %d", resp.StatusCode)
	}
	return nil
}
