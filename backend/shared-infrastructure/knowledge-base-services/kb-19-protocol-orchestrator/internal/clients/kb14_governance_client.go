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

// KB14GovernanceClient is the HTTP client for KB-14 Care Navigator/Governance service.
// KB-14 provides governance task creation, audit logging, and escalation workflows.
type KB14GovernanceClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB14GovernanceClient creates a new KB14GovernanceClient.
func NewKB14GovernanceClient(baseURL string, timeout time.Duration, log *logrus.Entry) *KB14GovernanceClient {
	return &KB14GovernanceClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.WithField("client", "kb14-governance"),
	}
}

// GovernanceTaskRequest is the request to create a governance task.
type GovernanceTaskRequest struct {
	PatientID       uuid.UUID   `json:"patient_id"`
	EncounterID     uuid.UUID   `json:"encounter_id"`
	TaskType        string      `json:"task_type"`        // REVIEW, OVERRIDE, ESCALATION, AUDIT
	Priority        string      `json:"priority"`         // LOW, MEDIUM, HIGH, CRITICAL
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	AssignedRole    string      `json:"assigned_role"`    // PHYSICIAN, NURSE, PHARMACIST, etc.
	AssignedUserID  string      `json:"assigned_user_id,omitempty"`
	DueAt           time.Time   `json:"due_at,omitempty"`
	SourceDecisionID uuid.UUID  `json:"source_decision_id"`
	RequiresSignoff bool        `json:"requires_signoff"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// GovernanceTask is the response from creating a governance task.
type GovernanceTask struct {
	ID              uuid.UUID              `json:"id"`
	PatientID       uuid.UUID              `json:"patient_id"`
	TaskType        string                 `json:"task_type"`
	Priority        string                 `json:"priority"`
	Title           string                 `json:"title"`
	Status          string                 `json:"status"`  // PENDING, IN_PROGRESS, COMPLETED, CANCELLED
	AssignedRole    string                 `json:"assigned_role"`
	AssignedUserID  string                 `json:"assigned_user_id,omitempty"`
	DueAt           time.Time              `json:"due_at,omitempty"`
	CompletedAt     time.Time              `json:"completed_at,omitempty"`
	CompletedBy     string                 `json:"completed_by,omitempty"`
	SignoffRequired bool                   `json:"signoff_required"`
	SignedOffAt     time.Time              `json:"signed_off_at,omitempty"`
	SignedOffBy     string                 `json:"signed_off_by,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
}

// OverrideRequest is the request to log a clinical override.
type OverrideRequest struct {
	PatientID       uuid.UUID `json:"patient_id"`
	EncounterID     uuid.UUID `json:"encounter_id"`
	DecisionID      uuid.UUID `json:"decision_id"`
	OverrideType    string    `json:"override_type"`     // SAFETY_BLOCK, ALERT, RECOMMENDATION
	OriginalAdvice  string    `json:"original_advice"`
	OverrideReason  string    `json:"override_reason"`
	ClinicalJustification string `json:"clinical_justification"`
	OverriddenBy    string    `json:"overridden_by"`
	RequiresReview  bool      `json:"requires_review"`
}

// OverrideRecord is the logged override.
type OverrideRecord struct {
	ID              uuid.UUID `json:"id"`
	PatientID       uuid.UUID `json:"patient_id"`
	DecisionID      uuid.UUID `json:"decision_id"`
	OverrideType    string    `json:"override_type"`
	OriginalAdvice  string    `json:"original_advice"`
	OverrideReason  string    `json:"override_reason"`
	ClinicalJustification string `json:"clinical_justification"`
	OverriddenBy    string    `json:"overridden_by"`
	OverriddenAt    time.Time `json:"overridden_at"`
	ReviewStatus    string    `json:"review_status"` // PENDING_REVIEW, REVIEWED, ACCEPTED, FLAGGED
	ReviewedBy      string    `json:"reviewed_by,omitempty"`
	ReviewedAt      time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes     string    `json:"review_notes,omitempty"`
}

// EscalationRequest is the request to trigger an escalation.
type EscalationRequest struct {
	PatientID       uuid.UUID `json:"patient_id"`
	EncounterID     uuid.UUID `json:"encounter_id"`
	EscalationType  string    `json:"escalation_type"`  // CLINICAL, SAFETY, COMPLIANCE
	Severity        string    `json:"severity"`         // WARNING, URGENT, CRITICAL
	Reason          string    `json:"reason"`
	SourceDecisionID uuid.UUID `json:"source_decision_id,omitempty"`
	EscalationPath  []string  `json:"escalation_path"`  // Role sequence
	NotifyImmediately bool    `json:"notify_immediately"`
}

// Escalation is the escalation record.
type Escalation struct {
	ID              uuid.UUID `json:"id"`
	PatientID       uuid.UUID `json:"patient_id"`
	EscalationType  string    `json:"escalation_type"`
	Severity        string    `json:"severity"`
	Status          string    `json:"status"`   // OPEN, ACKNOWLEDGED, RESOLVED, CLOSED
	CurrentLevel    int       `json:"current_level"`
	EscalationPath  []string  `json:"escalation_path"`
	AcknowledgedBy  string    `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  time.Time `json:"acknowledged_at,omitempty"`
	ResolvedBy      string    `json:"resolved_by,omitempty"`
	ResolvedAt      time.Time `json:"resolved_at,omitempty"`
	Resolution      string    `json:"resolution,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// AuditLogEntry is an audit log entry for compliance.
type AuditLogEntry struct {
	ID              uuid.UUID              `json:"id"`
	EventType       string                 `json:"event_type"`
	EntityType      string                 `json:"entity_type"`
	EntityID        uuid.UUID              `json:"entity_id"`
	PatientID       uuid.UUID              `json:"patient_id,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	Action          string                 `json:"action"`
	Details         map[string]interface{} `json:"details,omitempty"`
	IPAddress       string                 `json:"ip_address,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
}

// AuditLogRequest is the request to write an audit log entry.
type AuditLogRequest struct {
	EventType       string                 `json:"event_type"`
	EntityType      string                 `json:"entity_type"`
	EntityID        uuid.UUID              `json:"entity_id"`
	PatientID       uuid.UUID              `json:"patient_id,omitempty"`
	UserID          string                 `json:"user_id,omitempty"`
	Action          string                 `json:"action"`
	Details         map[string]interface{} `json:"details,omitempty"`
}

// CreateTask creates a governance task.
func (c *KB14GovernanceClient) CreateTask(ctx context.Context, req GovernanceTaskRequest) (*GovernanceTask, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id": req.PatientID,
		"task_type":  req.TaskType,
		"priority":   req.Priority,
	}).Debug("Creating governance task")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/tasks", bytes.NewReader(body))
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
		return nil, fmt.Errorf("task creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result GovernanceTask
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"task_id":    result.ID,
		"patient_id": req.PatientID,
		"task_type":  req.TaskType,
	}).Debug("Governance task created")

	return &result, nil
}

// LogOverride logs a clinical override for audit and review.
func (c *KB14GovernanceClient) LogOverride(ctx context.Context, req OverrideRequest) (*OverrideRecord, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":    req.PatientID,
		"decision_id":   req.DecisionID,
		"override_type": req.OverrideType,
	}).Debug("Logging clinical override")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/overrides", bytes.NewReader(body))
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
		return nil, fmt.Errorf("override logging failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result OverrideRecord
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// TriggerEscalation triggers a clinical escalation.
func (c *KB14GovernanceClient) TriggerEscalation(ctx context.Context, req EscalationRequest) (*Escalation, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":      req.PatientID,
		"escalation_type": req.EscalationType,
		"severity":        req.Severity,
	}).Debug("Triggering escalation")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/escalations", bytes.NewReader(body))
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
		return nil, fmt.Errorf("escalation trigger failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result Escalation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// WriteAuditLog writes an audit log entry.
func (c *KB14GovernanceClient) WriteAuditLog(ctx context.Context, req AuditLogRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/audit", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("audit log write failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Health checks if KB-14 is healthy.
func (c *KB14GovernanceClient) Health(ctx context.Context) error {
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
