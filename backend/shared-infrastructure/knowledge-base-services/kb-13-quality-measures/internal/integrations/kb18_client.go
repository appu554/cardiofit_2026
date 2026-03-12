// Package integrations provides HTTP clients for external KB service integrations.
package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB18Client provides integration with KB-18 Governance Engine.
//
// KB-18 is responsible for:
//   - Quality performance tracking for compliance reporting
//   - Governance rules and policy enforcement
//   - Audit trail management
//   - Regulatory compliance validation
type KB18Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB18Client creates a new KB-18 Governance client.
func NewKB18Client(baseURL string, logger *zap.Logger) *KB18Client {
	return &KB18Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// QualityPerformanceReport represents quality metrics sent to KB-18.
type QualityPerformanceReport struct {
	MeasureID       string    `json:"measure_id"`
	MeasureName     string    `json:"measure_name"`
	Program         string    `json:"program"`
	Domain          string    `json:"domain"`
	Score           float64   `json:"score"`
	PerformanceRate float64   `json:"performance_rate"`
	Target          float64   `json:"target"`
	MeetsTarget     bool      `json:"meets_target"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	CalculatedAt    time.Time `json:"calculated_at"`
	PopulationCounts struct {
		InitialPopulation    int `json:"initial_population"`
		Denominator          int `json:"denominator"`
		DenominatorExclusion int `json:"denominator_exclusion"`
		DenominatorException int `json:"denominator_exception"`
		Numerator            int `json:"numerator"`
		NumeratorExclusion   int `json:"numerator_exclusion"`
	} `json:"population_counts"`
}

// GovernanceRule represents a compliance rule from KB-18.
type GovernanceRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Severity    string   `json:"severity"`
	Conditions  []string `json:"conditions"`
	Actions     []string `json:"actions"`
	Active      bool     `json:"active"`
}

// ComplianceStatus represents compliance validation result.
type ComplianceStatus struct {
	Compliant       bool                `json:"compliant"`
	Violations      []ComplianceViolation `json:"violations,omitempty"`
	Warnings        []string            `json:"warnings,omitempty"`
	ValidatedAt     time.Time           `json:"validated_at"`
	NextReviewDate  *time.Time          `json:"next_review_date,omitempty"`
}

// ComplianceViolation represents a compliance rule violation.
type ComplianceViolation struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// SubmitPerformanceReport sends quality performance data to KB-18 for governance tracking.
func (c *KB18Client) SubmitPerformanceReport(ctx context.Context, report *QualityPerformanceReport) error {
	url := fmt.Sprintf("%s/v1/governance/quality-reports", c.baseURL)

	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-18 report submission failed",
			zap.String("measure_id", report.MeasureID),
			zap.Error(err),
		)
		return fmt.Errorf("KB-18 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("KB-18 returned status %d", resp.StatusCode)
	}

	c.logger.Debug("Submitted quality report to KB-18",
		zap.String("measure_id", report.MeasureID),
		zap.Float64("score", report.Score),
	)

	return nil
}

// GetGovernanceRules retrieves active governance rules from KB-18.
func (c *KB18Client) GetGovernanceRules(ctx context.Context, category string) ([]GovernanceRule, error) {
	url := fmt.Sprintf("%s/v1/governance/rules", c.baseURL)
	if category != "" {
		url = fmt.Sprintf("%s?category=%s", url, category)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KB-18 rules request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-18 returned status %d", resp.StatusCode)
	}

	var result struct {
		Rules []GovernanceRule `json:"rules"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Rules, nil
}

// ValidateCompliance checks compliance status for a measure.
func (c *KB18Client) ValidateCompliance(ctx context.Context, measureID string, score float64) (*ComplianceStatus, error) {
	url := fmt.Sprintf("%s/v1/governance/compliance/validate", c.baseURL)

	payload := map[string]interface{}{
		"measure_id": measureID,
		"score":      score,
		"source":     "kb-13-quality-measures",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KB-18 compliance request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-18 returned status %d", resp.StatusCode)
	}

	var status ComplianceStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}

// CreateAuditEntry creates an audit trail entry in KB-18.
func (c *KB18Client) CreateAuditEntry(ctx context.Context, action, resource, details string) error {
	url := fmt.Sprintf("%s/v1/governance/audit", c.baseURL)

	payload := map[string]interface{}{
		"source":    "kb-13-quality-measures",
		"action":    action,
		"resource":  resource,
		"details":   details,
		"timestamp": time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal audit entry: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-18 audit entry failed", zap.Error(err))
		return fmt.Errorf("KB-18 audit request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("KB-18 returned status %d", resp.StatusCode)
	}

	return nil
}

// HealthCheck verifies KB-18 is accessible.
func (c *KB18Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-18 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-18 health check returned status %d", resp.StatusCode)
	}

	return nil
}

// GetBaseURL returns the configured base URL.
func (c *KB18Client) GetBaseURL() string {
	return c.baseURL
}
