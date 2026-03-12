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

// KB3TemporalClient is the HTTP client for KB-3 Guidelines/Temporal service.
// KB-3 provides temporal binding - scheduling follow-ups, setting deadlines,
// and creating time-based clinical alerts.
type KB3TemporalClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB3TemporalClient creates a new KB3TemporalClient.
func NewKB3TemporalClient(baseURL string, timeout time.Duration, log *logrus.Entry) *KB3TemporalClient {
	return &KB3TemporalClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.WithField("client", "kb3-temporal"),
	}
}

// TemporalBindingRequest is the request to create a temporal binding.
type TemporalBindingRequest struct {
	PatientID     uuid.UUID `json:"patient_id"`
	EncounterID   uuid.UUID `json:"encounter_id"`
	DecisionID    uuid.UUID `json:"decision_id"`
	DecisionType  string    `json:"decision_type"`
	Target        string    `json:"target"`
	SourceProtocol string   `json:"source_protocol"`
	Urgency       string    `json:"urgency"`
	Timing        *Timing   `json:"timing,omitempty"`
}

// Timing specifies when an action should occur.
type Timing struct {
	DueWithin     string    `json:"due_within,omitempty"`     // e.g., "1h", "24h", "7d"
	ScheduledAt   time.Time `json:"scheduled_at,omitempty"`
	RecurringCron string    `json:"recurring_cron,omitempty"` // For recurring tasks
	Deadline      time.Time `json:"deadline,omitempty"`
}

// TemporalBinding is the response from creating a temporal binding.
type TemporalBinding struct {
	ID            uuid.UUID `json:"id"`
	DecisionID    uuid.UUID `json:"decision_id"`
	BindingType   string    `json:"binding_type"`   // DEADLINE, FOLLOWUP, RECURRING, ALERT
	Status        string    `json:"status"`         // PENDING, ACTIVE, COMPLETED, EXPIRED
	DueAt         time.Time `json:"due_at"`
	AlertSettings *AlertSettings `json:"alert_settings,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// AlertSettings configures how alerts are generated.
type AlertSettings struct {
	AlertType      string   `json:"alert_type"`       // REMINDER, WARNING, ESCALATION
	NotifyRoles    []string `json:"notify_roles"`     // PHYSICIAN, NURSE, PHARMACIST
	EscalationPath []string `json:"escalation_path"`
	WarningBefore  string   `json:"warning_before"`   // e.g., "15m", "1h"
}

// FollowUpRequest is the request to schedule a follow-up.
type FollowUpRequest struct {
	PatientID       uuid.UUID `json:"patient_id"`
	EncounterID     uuid.UUID `json:"encounter_id"`
	FollowUpType    string    `json:"follow_up_type"`   // LAB, IMAGING, CONSULT, REASSESS
	Reason          string    `json:"reason"`
	ScheduleWithin  string    `json:"schedule_within"`  // e.g., "24h", "7d"
	Priority        string    `json:"priority"`         // STAT, URGENT, ROUTINE
	SourceDecisionID uuid.UUID `json:"source_decision_id"`
}

// FollowUp is the response from scheduling a follow-up.
type FollowUp struct {
	ID             uuid.UUID `json:"id"`
	PatientID      uuid.UUID `json:"patient_id"`
	FollowUpType   string    `json:"follow_up_type"`
	ScheduledDate  time.Time `json:"scheduled_date"`
	Status         string    `json:"status"`
	RemindersSent  int       `json:"reminders_sent"`
	CreatedAt      time.Time `json:"created_at"`
}

// DeadlineRequest is the request to set a clinical deadline.
type DeadlineRequest struct {
	PatientID        uuid.UUID `json:"patient_id"`
	EncounterID      uuid.UUID `json:"encounter_id"`
	DeadlineType     string    `json:"deadline_type"`     // ACTION_REQUIRED, REVIEW, RECERT
	Description      string    `json:"description"`
	DueAt            time.Time `json:"due_at"`
	EscalateAfter    string    `json:"escalate_after"`    // e.g., "30m", "1h"
	SourceDecisionID uuid.UUID `json:"source_decision_id"`
}

// Deadline is the response from setting a deadline.
type Deadline struct {
	ID           uuid.UUID `json:"id"`
	PatientID    uuid.UUID `json:"patient_id"`
	DeadlineType string    `json:"deadline_type"`
	DueAt        time.Time `json:"due_at"`
	Status       string    `json:"status"`    // PENDING, MET, MISSED, ESCALATED
	MetAt        time.Time `json:"met_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// PathwayStartRequest is the request format expected by KB-3's /v1/pathways/start endpoint.
// KB-3 expects pathway_id (not protocol_id) and patient_id as strings.
type PathwayStartRequest struct {
	PathwayID string                 `json:"pathway_id"`
	PatientID string                 `json:"patient_id"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// PathwayResponse is the response from KB-3's pathway start endpoint.
type PathwayResponse struct {
	ID         uuid.UUID `json:"id"`
	PatientID  uuid.UUID `json:"patient_id"`
	ProtocolID string    `json:"protocol_id"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
}

// BindTiming creates a temporal binding for a decision by starting a pathway in KB-3.
// KB-3 uses /v1/pathways/start to initiate clinical pathways with temporal constraints.
func (c *KB3TemporalClient) BindTiming(ctx context.Context, req TemporalBindingRequest) (*TemporalBinding, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":  req.PatientID,
		"decision_id": req.DecisionID,
	}).Debug("Creating temporal binding via pathway start")

	// Convert to KB-3's pathway start request format
	// KB-3 expects pathway_id (maps to our SourceProtocol) and patient_id as strings
	pathwayReq := PathwayStartRequest{
		PathwayID: req.SourceProtocol,
		PatientID: req.PatientID.String(),
		Context: map[string]interface{}{
			"decision_type": req.DecisionType,
			"target":        req.Target,
			"encounter_id":  req.EncounterID.String(),
			"decision_id":   req.DecisionID.String(),
		},
	}

	body, err := json.Marshal(pathwayReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// KB-3 uses /v1/ prefix (not /api/v1/)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/pathways/start", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("temporal binding failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var pathwayResp PathwayResponse
	if err := json.NewDecoder(resp.Body).Decode(&pathwayResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert pathway response to temporal binding format
	dueAt := time.Now().Add(24 * time.Hour) // Default due time
	if req.Timing != nil && !req.Timing.Deadline.IsZero() {
		dueAt = req.Timing.Deadline
	}

	result := &TemporalBinding{
		ID:          pathwayResp.ID,
		DecisionID:  req.DecisionID,
		BindingType: "PATHWAY",
		Status:      pathwayResp.Status,
		DueAt:       dueAt,
		CreatedAt:   pathwayResp.StartedAt,
	}

	c.log.WithFields(logrus.Fields{
		"binding_id":  result.ID,
		"decision_id": req.DecisionID,
		"due_at":      result.DueAt,
	}).Debug("Temporal binding created via pathway")

	return result, nil
}

// ScheduleItemRequest is the request format for KB-3's /v1/schedule/:patientId/add endpoint.
type ScheduleItemRequest struct {
	ItemType    string    `json:"item_type"`    // LAB, IMAGING, CONSULT, REASSESS, MEDICATION
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	Priority    string    `json:"priority"`     // STAT, URGENT, ROUTINE
	ProtocolRef string    `json:"protocol_ref,omitempty"`
}

// ScheduleItemResponse is the response from KB-3's schedule add endpoint.
type ScheduleItemResponse struct {
	ID          uuid.UUID `json:"id"`
	PatientID   uuid.UUID `json:"patient_id"`
	ItemType    string    `json:"item_type"`
	Description string    `json:"description"`
	DueDate     time.Time `json:"due_date"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// ScheduleFollowUp schedules a follow-up action using KB-3's schedule endpoint.
// KB-3 uses /v1/schedule/:patientId/add for scheduling items.
func (c *KB3TemporalClient) ScheduleFollowUp(ctx context.Context, req FollowUpRequest) (*FollowUp, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":     req.PatientID,
		"follow_up_type": req.FollowUpType,
	}).Debug("Scheduling follow-up via KB-3 schedule endpoint")

	// Parse schedule_within duration
	scheduleDuration, err := time.ParseDuration(req.ScheduleWithin)
	if err != nil {
		scheduleDuration = 24 * time.Hour // Default to 24 hours
	}

	// Convert to KB-3's schedule item request format
	scheduleReq := ScheduleItemRequest{
		ItemType:    req.FollowUpType,
		Description: req.Reason,
		DueDate:     time.Now().Add(scheduleDuration),
		Priority:    req.Priority,
		ProtocolRef: req.SourceDecisionID.String(),
	}

	body, err := json.Marshal(scheduleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// KB-3 uses /v1/schedule/:patientId/add
	url := fmt.Sprintf("%s/v1/schedule/%s/add", c.baseURL, req.PatientID.String())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("schedule follow-up failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var scheduleResp ScheduleItemResponse
	if err := json.NewDecoder(resp.Body).Decode(&scheduleResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to FollowUp response format
	result := &FollowUp{
		ID:            scheduleResp.ID,
		PatientID:     scheduleResp.PatientID,
		FollowUpType:  scheduleResp.ItemType,
		ScheduledDate: scheduleResp.DueDate,
		Status:        scheduleResp.Status,
		RemindersSent: 0,
		CreatedAt:     scheduleResp.CreatedAt,
	}

	return result, nil
}

// SetDeadline sets a clinical deadline using KB-3's schedule endpoint.
// KB-3 uses /v1/schedule/:patientId/add for scheduling deadline items.
func (c *KB3TemporalClient) SetDeadline(ctx context.Context, req DeadlineRequest) (*Deadline, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":    req.PatientID,
		"deadline_type": req.DeadlineType,
		"due_at":        req.DueAt,
	}).Debug("Setting deadline via KB-3 schedule endpoint")

	// Convert to KB-3's schedule item request format with DEADLINE type
	scheduleReq := ScheduleItemRequest{
		ItemType:    "DEADLINE_" + req.DeadlineType,
		Description: req.Description,
		DueDate:     req.DueAt,
		Priority:    "URGENT", // Deadlines are urgent by default
		ProtocolRef: req.SourceDecisionID.String(),
	}

	body, err := json.Marshal(scheduleReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// KB-3 uses /v1/schedule/:patientId/add
	url := fmt.Sprintf("%s/v1/schedule/%s/add", c.baseURL, req.PatientID.String())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("set deadline failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var scheduleResp ScheduleItemResponse
	if err := json.NewDecoder(resp.Body).Decode(&scheduleResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to Deadline response format
	result := &Deadline{
		ID:           scheduleResp.ID,
		PatientID:    scheduleResp.PatientID,
		DeadlineType: req.DeadlineType,
		DueAt:        scheduleResp.DueDate,
		Status:       scheduleResp.Status,
		CreatedAt:    scheduleResp.CreatedAt,
	}

	return result, nil
}

// Health checks if KB-3 is healthy.
func (c *KB3TemporalClient) Health(ctx context.Context) error {
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
