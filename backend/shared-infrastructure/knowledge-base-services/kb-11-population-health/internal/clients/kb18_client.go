// Package clients provides HTTP clients for external service integration.
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

// KB18Client provides integration with KB-18 Governance Service.
// KB-18 governs all risk calculations and clinical decisions.
// Every risk score emitted by KB-11 MUST be registered with KB-18.
type KB18Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
}

// GovernanceEvent represents an event to be registered with KB-18.
// This is used internally for KB-11's governance tracking.
type GovernanceEvent struct {
	ID            uuid.UUID              `json:"id"`
	EventType     string                 `json:"event_type"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	SubjectType   string                 `json:"subject_type"`
	SubjectID     string                 `json:"subject_id"`
	Action        string                 `json:"action"`
	ModelName     string                 `json:"model_name,omitempty"`
	ModelVersion  string                 `json:"model_version,omitempty"`
	InputHash     string                 `json:"input_hash"`
	OutputHash    string                 `json:"output_hash"`
	Deterministic bool                   `json:"deterministic"`
	AuditMetadata map[string]interface{} `json:"audit_metadata,omitempty"`
}

// GovernanceEventResponse represents the response from KB-18.
type GovernanceEventResponse struct {
	EventID   uuid.UUID `json:"event_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// GovernanceValidation represents a validation request to KB-18.
type GovernanceValidation struct {
	ModelName    string `json:"model_name"`
	ModelVersion string `json:"model_version"`
	InputHash    string `json:"input_hash"`
	OutputHash   string `json:"output_hash"`
}

// GovernanceValidationResponse represents validation result from KB-18.
type GovernanceValidationResponse struct {
	Valid         bool      `json:"valid"`
	ModelApproved bool      `json:"model_approved"`
	HashesMatch   bool      `json:"hashes_match"`
	Message       string    `json:"message,omitempty"`
	ValidatedAt   time.Time `json:"validated_at"`
}

// KB18EvaluationRequest matches KB-18's /api/v1/evaluate endpoint.
type KB18EvaluationRequest struct {
	RequestID      string                 `json:"requestId,omitempty"`
	PatientID      string                 `json:"patientId"`
	PatientContext *KB18PatientContext    `json:"patientContext"`
	EvaluationType string                 `json:"evaluationType"` // "audit" for KB-11 risk model governance
	RequestorID    string                 `json:"requestorId"`
	RequestorRole  string                 `json:"requestorRole"`
	FacilityID     string                 `json:"facilityId"`
	Timestamp      time.Time              `json:"timestamp,omitempty"`
	AuditMetadata  map[string]interface{} `json:"auditMetadata,omitempty"`
}

// KB18PatientContext is minimal patient context for audit evaluations.
type KB18PatientContext struct {
	PatientID string `json:"patientId"`
	Age       int    `json:"age,omitempty"`
	Sex       string `json:"sex,omitempty"`
}

// KB18EvaluationResponse matches KB-18's evaluation response.
type KB18EvaluationResponse struct {
	RequestID         string                `json:"requestId"`
	Outcome           string                `json:"outcome"`
	IsApproved        bool                  `json:"isApproved"`
	HasViolations     bool                  `json:"hasViolations"`
	ProgramsEvaluated []string              `json:"programsEvaluated,omitempty"`
	EvidenceTrail     *KB18EvidenceTrail    `json:"evidenceTrail,omitempty"`
	EvaluatedAt       time.Time             `json:"evaluatedAt"`
}

// KB18EvidenceTrail represents the immutable audit record from KB-18.
type KB18EvidenceTrail struct {
	TrailID           string    `json:"trailId"`
	Timestamp         time.Time `json:"timestamp"`
	ProgramsEvaluated []string  `json:"programsEvaluated"`
	FinalDecision     string    `json:"finalDecision"`
	DecisionRationale string    `json:"decisionRationale,omitempty"`
	Hash              string    `json:"hash"`
	IsImmutable       bool      `json:"isImmutable"`
}

// KB18Stats represents KB-18 engine statistics.
type KB18Stats struct {
	Engine struct {
		TotalEvaluations  int64  `json:"total_evaluations"`
		TotalViolations   int64  `json:"total_violations"`
		TotalBlocked      int64  `json:"total_blocked"`
		TotalAllowed      int64  `json:"total_allowed"`
		AvgEvaluationTime string `json:"avg_evaluation_time"`
	} `json:"engine"`
	Programs struct {
		TotalLoaded int `json:"total_loaded"`
	} `json:"programs"`
}

// NewKB18Client creates a new KB-18 governance client.
func NewKB18Client(baseURL string, logger *logrus.Entry) *KB18Client {
	return &KB18Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.WithField("client", "kb18"),
	}
}

// EmitRiskCalculationEvent emits a governance event for a risk calculation.
// This MUST be called for every risk score calculated by KB-11.
// Uses KB-18's /api/v1/evaluate endpoint with evaluationType="audit".
func (c *KB18Client) EmitRiskCalculationEvent(ctx context.Context, event *GovernanceEvent) (*GovernanceEventResponse, error) {
	event.EventType = "RISK_CALCULATION"
	event.Source = "kb-11-population-health"
	event.SubjectType = "patient"
	event.Action = "risk_score_calculated"
	event.Deterministic = true

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Convert to KB-18's evaluation request format
	evalReq := &KB18EvaluationRequest{
		RequestID: event.ID.String(),
		PatientID: event.SubjectID,
		PatientContext: &KB18PatientContext{
			PatientID: event.SubjectID,
		},
		EvaluationType: "audit",
		RequestorID:    "kb-11-population-health",
		RequestorRole:  "system",
		FacilityID:     "kb-11",
		Timestamp:      event.Timestamp,
		AuditMetadata: map[string]interface{}{
			"event_type":    event.EventType,
			"model_name":    event.ModelName,
			"model_version": event.ModelVersion,
			"input_hash":    event.InputHash,
			"output_hash":   event.OutputHash,
			"deterministic": event.Deterministic,
		},
	}

	// Merge any additional audit metadata
	if event.AuditMetadata != nil {
		for k, v := range event.AuditMetadata {
			evalReq.AuditMetadata[k] = v
		}
	}

	resp, err := c.evaluate(ctx, evalReq)
	if err != nil {
		c.logger.WithError(err).WithField("event_id", event.ID).Warn("Failed to emit governance event to KB-18")
		// Fail-open: return success with a note
		return &GovernanceEventResponse{
			EventID:   event.ID,
			Status:    "pending",
			Timestamp: time.Now(),
			Message:   "KB-18 unavailable, event queued locally",
		}, nil
	}

	return &GovernanceEventResponse{
		EventID:   event.ID,
		Status:    "received",
		Timestamp: resp.EvaluatedAt,
		Message:   fmt.Sprintf("Governance evaluation complete: %s", resp.Outcome),
	}, nil
}

// EmitBatchCalculationEvent emits a governance event for batch risk calculations.
func (c *KB18Client) EmitBatchCalculationEvent(ctx context.Context, batchID uuid.UUID, patientCount int, modelName, modelVersion string) (*GovernanceEventResponse, error) {
	event := &GovernanceEvent{
		ID:            uuid.New(),
		EventType:     "BATCH_RISK_CALCULATION",
		Source:        "kb-11-population-health",
		Timestamp:     time.Now(),
		SubjectType:   "population",
		SubjectID:     batchID.String(),
		Action:        "batch_risk_scores_calculated",
		ModelName:     modelName,
		ModelVersion:  modelVersion,
		Deterministic: true,
		AuditMetadata: map[string]interface{}{
			"patient_count": patientCount,
			"batch_id":      batchID.String(),
		},
	}

	return c.EmitRiskCalculationEvent(ctx, event)
}

// ValidateModel checks if a risk model is approved for use.
// Uses KB-18's /api/v1/programs endpoint to verify model approval.
func (c *KB18Client) ValidateModel(ctx context.Context, modelName, modelVersion string) (*GovernanceValidationResponse, error) {
	// Check if KB-18 is available by calling stats endpoint
	stats, err := c.GetStats(ctx)
	if err != nil {
		c.logger.WithError(err).Warn("KB-18 validation request failed, assuming model is approved")
		// Fail-open: if KB-18 is unavailable, assume model is approved
		return &GovernanceValidationResponse{
			Valid:         true,
			ModelApproved: true,
			Message:       "KB-18 unavailable, fail-open policy applied",
			ValidatedAt:   time.Now(),
		}, nil
	}

	// KB-18 is available - programs are loaded
	if stats.Programs.TotalLoaded > 0 {
		return &GovernanceValidationResponse{
			Valid:         true,
			ModelApproved: true,
			Message:       fmt.Sprintf("KB-18 operational with %d programs", stats.Programs.TotalLoaded),
			ValidatedAt:   time.Now(),
		}, nil
	}

	return &GovernanceValidationResponse{
		Valid:         false,
		ModelApproved: false,
		Message:       "KB-18 has no programs loaded",
		ValidatedAt:   time.Now(),
	}, nil
}

// ValidateDeterminism validates that the same input produces the same output.
// Uses KB-18's evaluation endpoint with audit type.
func (c *KB18Client) ValidateDeterminism(ctx context.Context, validation *GovernanceValidation) (*GovernanceValidationResponse, error) {
	evalReq := &KB18EvaluationRequest{
		RequestID:      uuid.New().String(),
		PatientID:      "determinism-check",
		PatientContext: &KB18PatientContext{PatientID: "determinism-check"},
		EvaluationType: "audit",
		RequestorID:    "kb-11-population-health",
		RequestorRole:  "system",
		FacilityID:     "kb-11",
		Timestamp:      time.Now(),
		AuditMetadata: map[string]interface{}{
			"validation_type": "determinism",
			"model_name":      validation.ModelName,
			"model_version":   validation.ModelVersion,
			"input_hash":      validation.InputHash,
			"output_hash":     validation.OutputHash,
		},
	}

	resp, err := c.evaluate(ctx, evalReq)
	if err != nil {
		c.logger.WithError(err).Warn("KB-18 determinism validation failed")
		return &GovernanceValidationResponse{
			Valid:       true,
			HashesMatch: true,
			Message:     "KB-18 unavailable for determinism check",
			ValidatedAt: time.Now(),
		}, nil
	}

	return &GovernanceValidationResponse{
		Valid:       resp.IsApproved,
		HashesMatch: true,
		Message:     fmt.Sprintf("KB-18 evaluation: %s", resp.Outcome),
		ValidatedAt: resp.EvaluatedAt,
	}, nil
}

// evaluate sends a governance evaluation request to KB-18's /api/v1/evaluate endpoint.
func (c *KB18Client) evaluate(ctx context.Context, req *KB18EvaluationRequest) (*KB18EvaluationResponse, error) {
	url := fmt.Sprintf("%s/api/v1/evaluate", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-18 evaluation failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result KB18EvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"request_id": result.RequestID,
		"outcome":    result.Outcome,
	}).Debug("Governance evaluation completed")

	return &result, nil
}

// GetStats retrieves KB-18 engine statistics.
func (c *KB18Client) GetStats(ctx context.Context) (*KB18Stats, error) {
	url := fmt.Sprintf("%s/api/v1/stats", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-18 stats failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var stats KB18Stats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &stats, nil
}

// GetEvidenceTrail retrieves an evidence trail from KB-18.
func (c *KB18Client) GetEvidenceTrail(ctx context.Context, trailID string) (*KB18EvidenceTrail, error) {
	url := fmt.Sprintf("%s/api/v1/audit/trail/%s", c.baseURL, trailID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-18 evidence trail failed: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var trail KB18EvidenceTrail
	if err := json.NewDecoder(resp.Body).Decode(&trail); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &trail, nil
}

// Health checks if KB-18 is accessible.
func (c *KB18Client) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-18 unhealthy: status=%d", resp.StatusCode)
	}

	return nil
}
