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

// KB19Client provides integration with KB-19 Protocol Orchestrator.
//
// KB-19 is responsible for:
//   - Receiving quality alerts for protocol adjustments
//   - Care pathway optimization based on quality metrics
//   - Protocol recommendation management
//   - Clinical workflow triggering
type KB19Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB19Client creates a new KB-19 Protocol client.
func NewKB19Client(baseURL string, logger *zap.Logger) *KB19Client {
	return &KB19Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// QualityAlert represents an alert sent to KB-19 based on quality metrics.
type QualityAlert struct {
	ID            string    `json:"id,omitempty"`
	MeasureID     string    `json:"measure_id"`
	MeasureName   string    `json:"measure_name"`
	AlertType     string    `json:"alert_type"` // threshold_breach, trend_decline, gap_identified
	Severity      string    `json:"severity"`   // critical, high, medium, low
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	CurrentScore  float64   `json:"current_score"`
	TargetScore   float64   `json:"target_score"`
	Threshold     float64   `json:"threshold,omitempty"`
	PatientIDs    []string  `json:"patient_ids,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	Source        string    `json:"source"`
}

// ProtocolRecommendation represents a protocol adjustment from KB-19.
type ProtocolRecommendation struct {
	ID           string   `json:"id"`
	MeasureID    string   `json:"measure_id"`
	ProtocolID   string   `json:"protocol_id"`
	ProtocolName string   `json:"protocol_name"`
	Type         string   `json:"type"` // add_step, modify_frequency, trigger_intervention
	Priority     string   `json:"priority"`
	Description  string   `json:"description"`
	Actions      []string `json:"actions"`
	ExpectedImpact string `json:"expected_impact"`
}

// CareGapNotification notifies KB-19 about identified care gaps.
type CareGapNotification struct {
	MeasureID     string   `json:"measure_id"`
	MeasureName   string   `json:"measure_name"`
	GapType       string   `json:"gap_type"` // process_gap, screening_gap, follow_up_gap
	PatientCount  int      `json:"patient_count"`
	PatientIDs    []string `json:"patient_ids,omitempty"`
	Priority      string   `json:"priority"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	Interventions []string `json:"suggested_interventions"`
}

// SendQualityAlert sends a quality alert to KB-19 for protocol consideration.
func (c *KB19Client) SendQualityAlert(ctx context.Context, alert *QualityAlert) error {
	url := fmt.Sprintf("%s/v1/protocols/quality-alerts", c.baseURL)

	alert.Source = "kb-13-quality-measures"
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}

	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-19 alert submission failed",
			zap.String("measure_id", alert.MeasureID),
			zap.String("alert_type", alert.AlertType),
			zap.Error(err),
		)
		return fmt.Errorf("KB-19 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("KB-19 returned status %d", resp.StatusCode)
	}

	c.logger.Info("Sent quality alert to KB-19",
		zap.String("measure_id", alert.MeasureID),
		zap.String("alert_type", alert.AlertType),
		zap.String("severity", alert.Severity),
	)

	return nil
}

// NotifyCareGaps sends care gap information to KB-19 for intervention planning.
func (c *KB19Client) NotifyCareGaps(ctx context.Context, notification *CareGapNotification) error {
	url := fmt.Sprintf("%s/v1/protocols/care-gap-notifications", c.baseURL)

	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-19 care gap notification failed",
			zap.String("measure_id", notification.MeasureID),
			zap.Error(err),
		)
		return fmt.Errorf("KB-19 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("KB-19 returned status %d", resp.StatusCode)
	}

	c.logger.Info("Sent care gap notification to KB-19",
		zap.String("measure_id", notification.MeasureID),
		zap.Int("patient_count", notification.PatientCount),
	)

	return nil
}

// GetProtocolRecommendations retrieves protocol recommendations for a measure.
func (c *KB19Client) GetProtocolRecommendations(ctx context.Context, measureID string) ([]ProtocolRecommendation, error) {
	url := fmt.Sprintf("%s/v1/protocols/recommendations?measure_id=%s&source=quality-measure", c.baseURL, measureID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KB-19 recommendations request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-19 returned status %d", resp.StatusCode)
	}

	var result struct {
		Recommendations []ProtocolRecommendation `json:"recommendations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Recommendations, nil
}

// TriggerIntervention triggers a protocol intervention for specific patients.
func (c *KB19Client) TriggerIntervention(ctx context.Context, measureID string, interventionType string, patientIDs []string) error {
	url := fmt.Sprintf("%s/v1/protocols/interventions/trigger", c.baseURL)

	payload := map[string]interface{}{
		"measure_id":        measureID,
		"intervention_type": interventionType,
		"patient_ids":       patientIDs,
		"source":            "kb-13-quality-measures",
		"triggered_at":      time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal intervention: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-19 intervention request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("KB-19 returned status %d", resp.StatusCode)
	}

	c.logger.Info("Triggered intervention via KB-19",
		zap.String("measure_id", measureID),
		zap.String("intervention_type", interventionType),
		zap.Int("patient_count", len(patientIDs)),
	)

	return nil
}

// SendThresholdAlert sends an alert when a measure falls below threshold.
func (c *KB19Client) SendThresholdAlert(ctx context.Context, measureID, measureName string, currentScore, threshold float64) error {
	alert := &QualityAlert{
		MeasureID:    measureID,
		MeasureName:  measureName,
		AlertType:    "threshold_breach",
		Severity:     c.determineSeverity(currentScore, threshold),
		Title:        fmt.Sprintf("%s Below Target Threshold", measureName),
		Description:  fmt.Sprintf("Quality measure %s is at %.1f%%, below the target threshold of %.1f%%", measureName, currentScore*100, threshold*100),
		CurrentScore: currentScore,
		TargetScore:  threshold,
		Threshold:    threshold,
		Recommendations: []string{
			"Review patient population for care gaps",
			"Consider protocol adjustments",
			"Evaluate intervention effectiveness",
		},
	}

	return c.SendQualityAlert(ctx, alert)
}

// determineSeverity calculates severity based on distance from threshold.
func (c *KB19Client) determineSeverity(score, threshold float64) string {
	gap := threshold - score
	if gap > 0.20 { // More than 20% below threshold
		return "critical"
	} else if gap > 0.10 { // 10-20% below
		return "high"
	} else if gap > 0.05 { // 5-10% below
		return "medium"
	}
	return "low"
}

// HealthCheck verifies KB-19 is accessible.
func (c *KB19Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-19 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-19 health check returned status %d", resp.StatusCode)
	}

	return nil
}

// GetBaseURL returns the configured base URL.
func (c *KB19Client) GetBaseURL() string {
	return c.baseURL
}
