// Package clients provides HTTP clients for KB service integrations
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/config"
)

// KB3Client provides HTTP client for KB-3 Temporal/Guidelines service
// Used to fetch overdue alerts and protocol deadlines for task creation
type KB3Client struct {
	baseURL    string
	httpClient *http.Client
	config     config.KBClientConfig
	log        *logrus.Entry
}

// TemporalAlert represents an alert from KB-3 about temporal constraint violations
type TemporalAlert struct {
	AlertID       string    `json:"alert_id"`
	PatientID     string    `json:"patient_id"`
	EncounterID   string    `json:"encounter_id,omitempty"`
	ProtocolID    string    `json:"protocol_id"`
	ProtocolName  string    `json:"protocol_name"`
	ConstraintID  string    `json:"constraint_id"`
	Action        string    `json:"action"`
	Severity      string    `json:"severity"` // critical, major, minor
	Status        string    `json:"status"`   // pending, acknowledged, resolved
	Deadline      time.Time `json:"deadline"`
	TimeOverdue   int       `json:"time_overdue_minutes"` // Negative if before deadline
	AlertTime     time.Time `json:"alert_time"`
	Description   string    `json:"description"`
	Reference     string    `json:"reference"`
	Acknowledged  bool      `json:"acknowledged"`
}

// ProtocolDeadline represents an upcoming protocol deadline from KB-3
type ProtocolDeadline struct {
	DeadlineID    string    `json:"deadline_id"`
	PatientID     string    `json:"patient_id"`
	EncounterID   string    `json:"encounter_id,omitempty"`
	ProtocolID    string    `json:"protocol_id"`
	ProtocolName  string    `json:"protocol_name"`
	StageID       string    `json:"stage_id"`
	StageName     string    `json:"stage_name"`
	ActionID      string    `json:"action_id"`
	ActionName    string    `json:"action_name"`
	Deadline      time.Time `json:"deadline"`
	SLAMinutes    int       `json:"sla_minutes"`
	Priority      string    `json:"priority"` // critical, high, medium, low
	CurrentStatus string    `json:"current_status"`
}

// MonitoringOverdue represents overdue monitoring from KB-3
type MonitoringOverdue struct {
	OverdueID     string    `json:"overdue_id"`
	PatientID     string    `json:"patient_id"`
	ProtocolID    string    `json:"protocol_id"`
	ProtocolName  string    `json:"protocol_name"`
	MonitoringType string   `json:"monitoring_type"` // lab, vital, assessment
	LastPerformed *time.Time `json:"last_performed,omitempty"`
	DueDate       time.Time `json:"due_date"`
	DaysOverdue   int       `json:"days_overdue"`
	Severity      string    `json:"severity"`
	Description   string    `json:"description"`
}

// AlertsResponse represents the response from KB-3 alerts endpoint
type AlertsResponse struct {
	Success bool            `json:"success"`
	Alerts  []TemporalAlert `json:"alerts,omitempty"`
	Total   int             `json:"total"`
	Error   string          `json:"error,omitempty"`
}

// DeadlinesResponse represents the response from KB-3 deadlines endpoint
type DeadlinesResponse struct {
	Success   bool               `json:"success"`
	Deadlines []ProtocolDeadline `json:"deadlines,omitempty"`
	Total     int                `json:"total"`
	Error     string             `json:"error,omitempty"`
}

// OverdueResponse represents the response from KB-3 overdue endpoint
type OverdueResponse struct {
	Success bool                `json:"success"`
	Overdue []MonitoringOverdue `json:"overdue,omitempty"`
	Total   int                 `json:"total"`
	Error   string              `json:"error,omitempty"`
}

// NewKB3Client creates a new KB-3 Temporal HTTP client
func NewKB3Client(cfg config.KBClientConfig) *KB3Client {
	return &KB3Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
		log:    logrus.WithField("client", "kb3-temporal"),
	}
}

// IsEnabled returns whether the KB-3 client is enabled
func (c *KB3Client) IsEnabled() bool {
	return c.config.Enabled
}

// Health checks if KB-3 service is healthy
func (c *KB3Client) Health(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-3 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-3 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// GetActiveAlerts retrieves all active temporal alerts
func (c *KB3Client) GetActiveAlerts(ctx context.Context) ([]TemporalAlert, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, returning empty alerts")
		return []TemporalAlert{}, nil
	}

	var resp AlertsResponse
	if err := c.doRequest(ctx, "GET", "/v1/alerts?status=pending", nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-3 returned error: %s", resp.Error)
	}

	return resp.Alerts, nil
}

// GetOverdueAlerts retrieves overdue temporal alerts
func (c *KB3Client) GetOverdueAlerts(ctx context.Context) ([]TemporalAlert, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, returning empty overdue alerts")
		return []TemporalAlert{}, nil
	}

	var resp AlertsResponse
	if err := c.doRequest(ctx, "GET", "/v1/alerts/overdue", nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-3 returned error: %s", resp.Error)
	}

	return resp.Alerts, nil
}

// GetAlertsByPatient retrieves alerts for a specific patient
func (c *KB3Client) GetAlertsByPatient(ctx context.Context, patientID string) ([]TemporalAlert, error) {
	if !c.config.Enabled {
		return []TemporalAlert{}, nil
	}

	endpoint := fmt.Sprintf("/v1/alerts?patient_id=%s&status=pending", patientID)
	var resp AlertsResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Alerts, nil
}

// GetUpcomingDeadlines retrieves upcoming protocol deadlines
func (c *KB3Client) GetUpcomingDeadlines(ctx context.Context, hoursAhead int) ([]ProtocolDeadline, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, returning empty deadlines")
		return []ProtocolDeadline{}, nil
	}

	endpoint := fmt.Sprintf("/v1/protocols/deadlines?hours_ahead=%d", hoursAhead)
	var resp DeadlinesResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-3 returned error: %s", resp.Error)
	}

	return resp.Deadlines, nil
}

// GetMonitoringOverdue retrieves overdue monitoring items
func (c *KB3Client) GetMonitoringOverdue(ctx context.Context) ([]MonitoringOverdue, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-3 client disabled, returning empty overdue monitoring")
		return []MonitoringOverdue{}, nil
	}

	var resp OverdueResponse
	if err := c.doRequest(ctx, "GET", "/v1/monitoring/overdue", nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-3 returned error: %s", resp.Error)
	}

	return resp.Overdue, nil
}

// AcknowledgeAlert acknowledges an alert in KB-3
func (c *KB3Client) AcknowledgeAlert(ctx context.Context, alertID string, acknowledgedBy string) error {
	if !c.config.Enabled {
		return nil
	}

	body := map[string]string{
		"acknowledged_by": acknowledgedBy,
	}
	bodyBytes, _ := json.Marshal(body)

	endpoint := fmt.Sprintf("/v1/alerts/%s/acknowledge", alertID)
	return c.doRequest(ctx, "POST", endpoint, bodyBytes, nil)
}

// doRequest performs an HTTP request
func (c *KB3Client) doRequest(ctx context.Context, method, endpoint string, body []byte, result interface{}) error {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, nil)
	}
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Client-Service", "kb-14-care-navigator")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-3 request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("KB-3 error: %d - %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
