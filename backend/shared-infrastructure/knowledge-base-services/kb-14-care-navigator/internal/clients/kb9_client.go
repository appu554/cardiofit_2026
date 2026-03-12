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

// KB9Client provides HTTP client for KB-9 Care Gaps service
// Used to fetch care gaps for task creation
type KB9Client struct {
	baseURL    string
	httpClient *http.Client
	config     config.KBClientConfig
	log        *logrus.Entry
}

// CareGap represents a care gap from KB-9
type CareGap struct {
	GapID          string           `json:"gap_id"`
	PatientID      string           `json:"patient_id"`
	MeasureID      string           `json:"measure_id"`
	MeasureName    string           `json:"measure_name"`
	Category       string           `json:"category"` // preventive, chronic, wellness
	GapType        string           `json:"gap_type"` // screening, immunization, follow_up, monitoring
	Status         string           `json:"status"`   // open, in_progress, closed, excluded
	Priority       string           `json:"priority"` // high, medium, low
	DueDate        *time.Time       `json:"due_date,omitempty"`
	DetectedDate   time.Time        `json:"detected_date"`
	Description    string           `json:"description"`
	Rationale      string           `json:"rationale"`
	Interventions  []Intervention   `json:"interventions,omitempty"`
	EvidenceSource string           `json:"evidence_source"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// Intervention represents a recommended intervention for a care gap
type Intervention struct {
	InterventionID   string   `json:"intervention_id"`
	Type             string   `json:"type"` // order, referral, education, outreach
	Description      string   `json:"description"`
	Code             string   `json:"code,omitempty"`
	CodeSystem       string   `json:"code_system,omitempty"`
	Priority         int      `json:"priority"`
	RequiredActions  []string `json:"required_actions,omitempty"`
	EstimatedMinutes int      `json:"estimated_minutes,omitempty"`
}

// CareGapRequest represents a request to fetch care gaps
type CareGapRequest struct {
	PatientID  string   `json:"patient_id,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Statuses   []string `json:"statuses,omitempty"`
	Priorities []string `json:"priorities,omitempty"`
	DueWithin  int      `json:"due_within_days,omitempty"`
	Limit      int      `json:"limit,omitempty"`
}

// CareGapResponse represents the response from KB-9 care gaps endpoint
type CareGapResponse struct {
	Success  bool      `json:"success"`
	CareGaps []CareGap `json:"care_gaps,omitempty"`
	Total    int       `json:"total"`
	Error    string    `json:"error,omitempty"`
}

// CareGapSummary represents a summary of care gaps for a patient or population
type CareGapSummary struct {
	TotalGaps      int            `json:"total_gaps"`
	OpenGaps       int            `json:"open_gaps"`
	HighPriority   int            `json:"high_priority"`
	MediumPriority int            `json:"medium_priority"`
	LowPriority    int            `json:"low_priority"`
	ByCategory     map[string]int `json:"by_category"`
	ByType         map[string]int `json:"by_type"`
	OverdueGaps    int            `json:"overdue_gaps"`
	DueSoon        int            `json:"due_soon"` // Within 30 days
}

// CareGapSummaryResponse represents the response from KB-9 summary endpoint
type CareGapSummaryResponse struct {
	Success bool            `json:"success"`
	Summary *CareGapSummary `json:"summary,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// NewKB9Client creates a new KB-9 Care Gaps HTTP client
func NewKB9Client(cfg config.KBClientConfig) *KB9Client {
	return &KB9Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
		log:    logrus.WithField("client", "kb9-care-gaps"),
	}
}

// IsEnabled returns whether the KB-9 client is enabled
func (c *KB9Client) IsEnabled() bool {
	return c.config.Enabled
}

// Health checks if KB-9 service is healthy
func (c *KB9Client) Health(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-9 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-9 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// GetPatientCareGaps retrieves care gaps for a specific patient
func (c *KB9Client) GetPatientCareGaps(ctx context.Context, patientID string) ([]CareGap, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-9 client disabled, returning empty care gaps")
		return []CareGap{}, nil
	}

	req := CareGapRequest{
		PatientID: patientID,
		Statuses:  []string{"open", "in_progress"},
	}

	return c.getCareGaps(ctx, req)
}

// GetOpenCareGaps retrieves all open care gaps
func (c *KB9Client) GetOpenCareGaps(ctx context.Context) ([]CareGap, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-9 client disabled, returning empty care gaps")
		return []CareGap{}, nil
	}

	req := CareGapRequest{
		Statuses: []string{"open"},
	}

	return c.getCareGaps(ctx, req)
}

// GetHighPriorityCareGaps retrieves high priority care gaps
func (c *KB9Client) GetHighPriorityCareGaps(ctx context.Context) ([]CareGap, error) {
	if !c.config.Enabled {
		return []CareGap{}, nil
	}

	req := CareGapRequest{
		Statuses:   []string{"open", "in_progress"},
		Priorities: []string{"high"},
	}

	return c.getCareGaps(ctx, req)
}

// GetCareGapsDueSoon retrieves care gaps due within specified days
func (c *KB9Client) GetCareGapsDueSoon(ctx context.Context, daysAhead int) ([]CareGap, error) {
	if !c.config.Enabled {
		return []CareGap{}, nil
	}

	req := CareGapRequest{
		Statuses:  []string{"open"},
		DueWithin: daysAhead,
	}

	return c.getCareGaps(ctx, req)
}

// GetCareGapsByCategory retrieves care gaps by category
func (c *KB9Client) GetCareGapsByCategory(ctx context.Context, category string) ([]CareGap, error) {
	if !c.config.Enabled {
		return []CareGap{}, nil
	}

	req := CareGapRequest{
		Categories: []string{category},
		Statuses:   []string{"open", "in_progress"},
	}

	return c.getCareGaps(ctx, req)
}

// getCareGaps performs the actual API call to fetch care gaps
func (c *KB9Client) getCareGaps(ctx context.Context, req CareGapRequest) ([]CareGap, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp CareGapResponse
	if err := c.doRequest(ctx, "POST", "/api/v1/care-gaps", body, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-9 returned error: %s", resp.Error)
	}

	return resp.CareGaps, nil
}

// GetCareGapSummary retrieves a summary of care gaps for a patient
func (c *KB9Client) GetCareGapSummary(ctx context.Context, patientID string) (*CareGapSummary, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-9 client disabled, returning empty summary")
		return &CareGapSummary{
			ByCategory: make(map[string]int),
			ByType:     make(map[string]int),
		}, nil
	}

	endpoint := fmt.Sprintf("/api/v1/care-gaps/summary?patient_id=%s", patientID)
	var resp CareGapSummaryResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-9 returned error: %s", resp.Error)
	}

	return resp.Summary, nil
}

// UpdateCareGapStatus updates the status of a care gap
func (c *KB9Client) UpdateCareGapStatus(ctx context.Context, gapID string, status string, closedBy string) error {
	if !c.config.Enabled {
		return nil
	}

	body := map[string]string{
		"status":    status,
		"closed_by": closedBy,
	}
	bodyBytes, _ := json.Marshal(body)

	endpoint := fmt.Sprintf("/api/v1/care-gaps/%s/status", gapID)
	return c.doRequest(ctx, "PATCH", endpoint, bodyBytes, nil)
}

// LinkCareGapToTask links a care gap to a KB-14 task
func (c *KB9Client) LinkCareGapToTask(ctx context.Context, gapID string, taskID string) error {
	if !c.config.Enabled {
		return nil
	}

	body := map[string]string{
		"task_id":     taskID,
		"task_source": "KB14_CARE_NAVIGATOR",
	}
	bodyBytes, _ := json.Marshal(body)

	endpoint := fmt.Sprintf("/api/v1/care-gaps/%s/link-task", gapID)
	return c.doRequest(ctx, "POST", endpoint, bodyBytes, nil)
}

// doRequest performs an HTTP request
func (c *KB9Client) doRequest(ctx context.Context, method, endpoint string, body []byte, result interface{}) error {
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
		return fmt.Errorf("KB-9 request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("KB-9 error: %d - %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
