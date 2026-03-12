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

// KB12Client provides HTTP client for KB-12 Order Sets & Care Plans service
// Used to fetch care plan activities and order set tasks
type KB12Client struct {
	baseURL    string
	httpClient *http.Client
	config     config.KBClientConfig
	log        *logrus.Entry
}

// CarePlan represents a care plan from KB-12
type CarePlan struct {
	PlanID        string         `json:"plan_id"`
	PatientID     string         `json:"patient_id"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Status        string         `json:"status"` // draft, active, completed, cancelled
	Category      string         `json:"category"` // chronic, acute, preventive, wellness
	StartDate     time.Time      `json:"start_date"`
	EndDate       *time.Time     `json:"end_date,omitempty"`
	Goals         []CarePlanGoal `json:"goals,omitempty"`
	Activities    []Activity     `json:"activities,omitempty"`
	Team          []TeamMember   `json:"team,omitempty"`
	CreatedBy     string         `json:"created_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// CarePlanGoal represents a goal within a care plan
type CarePlanGoal struct {
	GoalID       string     `json:"goal_id"`
	Description  string     `json:"description"`
	TargetDate   *time.Time `json:"target_date,omitempty"`
	Status       string     `json:"status"` // proposed, planned, accepted, in_progress, achieved, cancelled
	Priority     string     `json:"priority"`
	Measures     []string   `json:"measures,omitempty"`
}

// Activity represents an activity within a care plan
type Activity struct {
	ActivityID    string     `json:"activity_id"`
	CarePlanID    string     `json:"care_plan_id,omitempty"`
	PatientID     string     `json:"patient_id,omitempty"`
	Type          string     `json:"type"` // medication, lab, procedure, education, referral, follow_up
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Status        string     `json:"status"` // scheduled, in_progress, completed, cancelled, not_started
	Priority      string     `json:"priority"`
	ScheduledDate *time.Time `json:"scheduled_date,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	CompletedDate *time.Time `json:"completed_date,omitempty"`
	AssignedTo    string     `json:"assigned_to,omitempty"`
	AssignedRole  string     `json:"assigned_role,omitempty"`
	Frequency     string     `json:"frequency,omitempty"` // once, daily, weekly, monthly
	Notes         string     `json:"notes,omitempty"`
	OrderSetID    string     `json:"order_set_id,omitempty"`
}

// TeamMember represents a care team member in KB-12
type TeamMember struct {
	MemberID string `json:"member_id"`
	Role     string `json:"role"`
	Name     string `json:"name"`
}

// OrderSet represents an order set from KB-12
type OrderSet struct {
	OrderSetID  string       `json:"order_set_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Category    string       `json:"category"`
	Condition   string       `json:"condition,omitempty"`
	Version     string       `json:"version"`
	Status      string       `json:"status"` // active, draft, retired
	Orders      []OrderItem  `json:"orders,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// OrderItem represents an individual order within an order set
type OrderItem struct {
	OrderID     string   `json:"order_id"`
	Type        string   `json:"type"` // medication, lab, imaging, procedure, referral
	Name        string   `json:"name"`
	Code        string   `json:"code,omitempty"`
	CodeSystem  string   `json:"code_system,omitempty"`
	Required    bool     `json:"required"`
	Instructions string  `json:"instructions,omitempty"`
	Priority    string   `json:"priority"`
	SLAMinutes  int      `json:"sla_minutes,omitempty"`
}

// CarePlanResponse represents the response from KB-12 care plans endpoint
type CarePlanResponse struct {
	Success   bool       `json:"success"`
	CarePlans []CarePlan `json:"care_plans,omitempty"`
	Total     int        `json:"total"`
	Error     string     `json:"error,omitempty"`
}

// SingleCarePlanResponse represents a single care plan response
type SingleCarePlanResponse struct {
	Success  bool      `json:"success"`
	CarePlan *CarePlan `json:"care_plan,omitempty"`
	Error    string    `json:"error,omitempty"`
}

// ActivitiesResponse represents the response from KB-12 activities endpoint
type ActivitiesResponse struct {
	Success    bool       `json:"success"`
	Activities []Activity `json:"activities,omitempty"`
	Total      int        `json:"total"`
	Error      string     `json:"error,omitempty"`
}

// NewKB12Client creates a new KB-12 Order Sets HTTP client
func NewKB12Client(cfg config.KBClientConfig) *KB12Client {
	return &KB12Client{
		baseURL: cfg.URL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		config: cfg,
		log:    logrus.WithField("client", "kb12-order-sets"),
	}
}

// IsEnabled returns whether the KB-12 client is enabled
func (c *KB12Client) IsEnabled() bool {
	return c.config.Enabled
}

// Health checks if KB-12 service is healthy
func (c *KB12Client) Health(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-12 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-12 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// GetPatientCarePlans retrieves active care plans for a patient
func (c *KB12Client) GetPatientCarePlans(ctx context.Context, patientID string) ([]CarePlan, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-12 client disabled, returning empty care plans")
		return []CarePlan{}, nil
	}

	endpoint := fmt.Sprintf("/api/v1/careplans/patient/%s?status=active", patientID)
	var resp CarePlanResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-12 returned error: %s", resp.Error)
	}

	return resp.CarePlans, nil
}

// GetCarePlan retrieves a specific care plan by ID
func (c *KB12Client) GetCarePlan(ctx context.Context, planID string) (*CarePlan, error) {
	if !c.config.Enabled {
		return nil, nil
	}

	endpoint := fmt.Sprintf("/api/v1/careplans/%s", planID)
	var resp SingleCarePlanResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-12 returned error: %s", resp.Error)
	}

	return resp.CarePlan, nil
}

// GetPendingActivities retrieves pending activities for a patient
func (c *KB12Client) GetPendingActivities(ctx context.Context, patientID string) ([]Activity, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-12 client disabled, returning empty activities")
		return []Activity{}, nil
	}

	endpoint := fmt.Sprintf("/api/v1/activities/patient/%s?status=scheduled,not_started", patientID)
	var resp ActivitiesResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-12 returned error: %s", resp.Error)
	}

	return resp.Activities, nil
}

// GetOverdueActivities retrieves overdue activities
func (c *KB12Client) GetOverdueActivities(ctx context.Context) ([]Activity, error) {
	if !c.config.Enabled {
		return []Activity{}, nil
	}

	var resp ActivitiesResponse
	if err := c.doRequest(ctx, "GET", "/api/v1/activities/overdue", nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-12 returned error: %s", resp.Error)
	}

	return resp.Activities, nil
}

// GetActivitiesDueSoon retrieves activities due within specified days
func (c *KB12Client) GetActivitiesDueSoon(ctx context.Context, daysAhead int) ([]Activity, error) {
	if !c.config.Enabled {
		return []Activity{}, nil
	}

	endpoint := fmt.Sprintf("/api/v1/activities/due-soon?days=%d", daysAhead)
	var resp ActivitiesResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-12 returned error: %s", resp.Error)
	}

	return resp.Activities, nil
}

// GetActivitiesByType retrieves activities by type
func (c *KB12Client) GetActivitiesByType(ctx context.Context, activityType string) ([]Activity, error) {
	if !c.config.Enabled {
		return []Activity{}, nil
	}

	endpoint := fmt.Sprintf("/api/v1/activities?type=%s&status=scheduled,not_started", activityType)
	var resp ActivitiesResponse
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Activities, nil
}

// UpdateActivityStatus updates the status of an activity
func (c *KB12Client) UpdateActivityStatus(ctx context.Context, activityID string, status string, completedBy string) error {
	if !c.config.Enabled {
		return nil
	}

	body := map[string]string{
		"status":       status,
		"completed_by": completedBy,
	}
	bodyBytes, _ := json.Marshal(body)

	endpoint := fmt.Sprintf("/api/v1/activities/%s/status", activityID)
	return c.doRequest(ctx, "PATCH", endpoint, bodyBytes, nil)
}

// LinkActivityToTask links a care plan activity to a KB-14 task
func (c *KB12Client) LinkActivityToTask(ctx context.Context, activityID string, taskID string) error {
	if !c.config.Enabled {
		return nil
	}

	body := map[string]string{
		"task_id":     taskID,
		"task_source": "KB14_CARE_NAVIGATOR",
	}
	bodyBytes, _ := json.Marshal(body)

	endpoint := fmt.Sprintf("/api/v1/activities/%s/link-task", activityID)
	return c.doRequest(ctx, "POST", endpoint, bodyBytes, nil)
}

// GetOrderSet retrieves an order set by ID
func (c *KB12Client) GetOrderSet(ctx context.Context, orderSetID string) (*OrderSet, error) {
	if !c.config.Enabled {
		return nil, nil
	}

	endpoint := fmt.Sprintf("/api/v1/ordersets/%s", orderSetID)
	var resp struct {
		Success  bool      `json:"success"`
		OrderSet *OrderSet `json:"order_set,omitempty"`
		Error    string    `json:"error,omitempty"`
	}
	if err := c.doRequest(ctx, "GET", endpoint, nil, &resp); err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-12 returned error: %s", resp.Error)
	}

	return resp.OrderSet, nil
}

// doRequest performs an HTTP request
func (c *KB12Client) doRequest(ctx context.Context, method, endpoint string, body []byte, result interface{}) error {
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
		return fmt.Errorf("KB-12 request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("KB-12 error: %d - %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
