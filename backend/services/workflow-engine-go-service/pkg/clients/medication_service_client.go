package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// MedicationServiceClient defines the interface for Medication Service communication
type MedicationServiceClient interface {
	Commit(ctx context.Context, request *MedicationCommitRequest) (*MedicationCommitResponse, error)
	CommitWithOverride(ctx context.Context, request *CommitWithOverrideRequest) (*MedicationCommitResponse, error)
	ValidateOrder(ctx context.Context, request *OrderValidationRequest) (*OrderValidationResponse, error)
	GetMedicationHistory(ctx context.Context, patientID string) (*MedicationHistoryResponse, error)
	HealthCheck(ctx context.Context) error
}

// MedicationCommitRequest represents a medication order commit request
type MedicationCommitRequest struct {
	ProposalSetID    string                 `json:"proposal_set_id"`
	ValidationID     string                 `json:"validation_id"`
	SelectedProposal map[string]interface{} `json:"selected_proposal"`
	ProviderDecision map[string]interface{} `json:"provider_decision"`
	CorrelationID    string                 `json:"correlation_id"`
	PatientID        string                 `json:"patient_id,omitempty"`
	ProviderID       string                 `json:"provider_id,omitempty"`
	EncounterID      string                 `json:"encounter_id,omitempty"`
	Priority         string                 `json:"priority,omitempty"`
	CommitMode       string                 `json:"commit_mode,omitempty"` // "immediate", "staged", "scheduled"
}

// MedicationCommitResponse represents the commit response
type MedicationCommitResponse struct {
	MedicationOrderID        string                 `json:"medication_order_id"`
	FHIRResourceID           string                 `json:"fhir_resource_id"`
	PersistenceStatus        string                 `json:"persistence_status"`
	EventPublicationStatus   string                 `json:"event_publication_status"`
	AuditTrailID             string                 `json:"audit_trail_id"`
	CommittedAt              time.Time              `json:"committed_at"`
	CommitMetrics            map[string]interface{} `json:"commit_metrics"`
	DownstreamNotifications  []string               `json:"downstream_notifications"`
	Status                   string                 `json:"status"`
	Message                  string                 `json:"message,omitempty"`
}

// OrderValidationRequest represents an order validation request
type OrderValidationRequest struct {
	MedicationOrder  map[string]interface{} `json:"medication_order"`
	PatientContext   map[string]interface{} `json:"patient_context"`
	ValidationLevel  string                 `json:"validation_level"` // "basic", "comprehensive"
	CorrelationID    string                 `json:"correlation_id"`
}

// OrderValidationResponse represents an order validation response
type OrderValidationResponse struct {
	ValidationID     string              `json:"validation_id"`
	Valid            bool                `json:"valid"`
	ValidationErrors []ValidationError   `json:"validation_errors,omitempty"`
	Warnings         []ValidationWarning `json:"warnings,omitempty"`
	FHIRCompliant    bool                `json:"fhir_compliant"`
	ProcessingTime   float64             `json:"processing_time_ms"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Code        string `json:"code"`
	Field       string `json:"field"`
	Message     string `json:"message"`
	Severity    string `json:"severity"`
	Correctable bool   `json:"correctable"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Code    string `json:"code"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// MedicationHistoryResponse represents medication history
type MedicationHistoryResponse struct {
	PatientID         string                   `json:"patient_id"`
	MedicationOrders  []map[string]interface{} `json:"medication_orders"`
	ActiveMedications []map[string]interface{} `json:"active_medications"`
	RecentChanges     []map[string]interface{} `json:"recent_changes"`
	LastUpdated       time.Time                `json:"last_updated"`
}

// CommitWithOverrideRequest represents a commit request with override details
type CommitWithOverrideRequest struct {
	ProposalID      string           `json:"proposal_id"`
	WorkflowID      string           `json:"workflow_id"`
	OverrideDetails *OverrideDetails `json:"override_details"`
}

// OverrideDetails contains information about the clinical override
type OverrideDetails struct {
	OverriddenBy      string `json:"overridden_by"`
	OverrideReason    string `json:"override_reason"`
	OverrideLevel     string `json:"override_level"`
	OriginalVerdict   string `json:"original_verdict"`
	CoSignature       *CoSignatureDetails `json:"co_signature,omitempty"`
	AlternativeAction string `json:"alternative_action,omitempty"`
}

// CoSignatureDetails represents co-signature information for high-level overrides
type CoSignatureDetails struct {
	CoSignedBy    string    `json:"co_signed_by"`
	CoSignedAt    time.Time `json:"co_signed_at"`
	CoSignerLevel string    `json:"co_signer_level"`
}

// medicationServiceClientImpl implements MedicationServiceClient
type medicationServiceClientImpl struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewMedicationServiceClient creates a new Medication Service client
func NewMedicationServiceClient(baseURL string, logger *zap.Logger) MedicationServiceClient {
	return &medicationServiceClientImpl{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     60 * time.Second,
			},
		},
		logger: logger,
	}
}

// Commit commits a medication order
func (c *medicationServiceClientImpl) Commit(ctx context.Context, request *MedicationCommitRequest) (*MedicationCommitResponse, error) {
	c.logger.Info("Committing medication order",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("proposal_set_id", request.ProposalSetID),
		zap.String("validation_id", request.ValidationID))

	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/medication/commit", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")
	httpReq.Header.Set("X-Correlation-ID", request.CorrelationID)
	httpReq.Header.Set("X-Request-Priority", "HIGH") // High priority for commit operations

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	c.logger.Info("Medication commit request completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.Duration("duration", duration),
		zap.Int("status_code", func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}()))

	if err != nil {
		c.logger.Error("Failed to commit medication order",
			zap.String("correlation_id", request.CorrelationID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute commit request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		c.logger.Error("Medication service returned error",
			zap.String("correlation_id", request.CorrelationID),
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("Medication Service returned status %d", resp.StatusCode)
	}

	// Parse response
	var response MedicationCommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Validate response
	if response.MedicationOrderID == "" {
		return nil, fmt.Errorf("invalid response: missing medication_order_id")
	}

	// Log commit success details
	c.logger.Info("Medication order committed successfully",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("medication_order_id", response.MedicationOrderID),
		zap.String("fhir_resource_id", response.FHIRResourceID),
		zap.String("persistence_status", response.PersistenceStatus),
		zap.String("event_status", response.EventPublicationStatus),
		zap.String("audit_trail_id", response.AuditTrailID))

	return &response, nil
}

// CommitWithOverride commits a medication order with override details
func (c *medicationServiceClientImpl) CommitWithOverride(ctx context.Context, request *CommitWithOverrideRequest) (*MedicationCommitResponse, error) {
	c.logger.Info("Committing medication order with override",
		zap.String("proposal_id", request.ProposalID),
		zap.String("workflow_id", request.WorkflowID),
		zap.String("overridden_by", request.OverrideDetails.OverriddenBy),
		zap.String("override_level", request.OverrideDetails.OverrideLevel))

	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/medication/commit/override", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")
	httpReq.Header.Set("X-Proposal-ID", request.ProposalID)
	httpReq.Header.Set("X-Override-Level", request.OverrideDetails.OverrideLevel)

	// Execute request
	startTime := time.Now()
	resp, err := c.httpClient.Do(httpReq)
	duration := time.Since(startTime)

	c.logger.Info("Medication override commit request completed",
		zap.String("proposal_id", request.ProposalID),
		zap.Duration("duration", duration),
		zap.Int("status_code", func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}()))

	if err != nil {
		c.logger.Error("Failed to commit medication order with override",
			zap.String("proposal_id", request.ProposalID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute override commit request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		c.logger.Error("Medication service returned error for override commit",
			zap.String("proposal_id", request.ProposalID),
			zap.Int("status_code", resp.StatusCode))
		return nil, fmt.Errorf("Medication Service returned status %d", resp.StatusCode)
	}

	// Parse response
	var response MedicationCommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Validate response
	if response.MedicationOrderID == "" {
		return nil, fmt.Errorf("invalid response: missing medication_order_id")
	}

	// Log override commit success details
	c.logger.Info("Medication order with override committed successfully",
		zap.String("proposal_id", request.ProposalID),
		zap.String("medication_order_id", response.MedicationOrderID),
		zap.String("fhir_resource_id", response.FHIRResourceID),
		zap.String("persistence_status", response.PersistenceStatus),
		zap.String("override_level", request.OverrideDetails.OverrideLevel),
		zap.String("overridden_by", request.OverrideDetails.OverriddenBy))

	return &response, nil
}

// ValidateOrder validates a medication order
func (c *medicationServiceClientImpl) ValidateOrder(ctx context.Context, request *OrderValidationRequest) (*OrderValidationResponse, error) {
	c.logger.Info("Validating medication order",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("validation_level", request.ValidationLevel))

	// Prepare request body
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/medication/validate", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")
	httpReq.Header.Set("X-Correlation-ID", request.CorrelationID)

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("Failed to validate medication order",
			zap.String("correlation_id", request.CorrelationID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute validation request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Medication Service returned status %d", resp.StatusCode)
	}

	// Parse response
	var response OrderValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Order validation completed",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("validation_id", response.ValidationID),
		zap.Bool("valid", response.Valid),
		zap.Bool("fhir_compliant", response.FHIRCompliant),
		zap.Int("error_count", len(response.ValidationErrors)),
		zap.Int("warning_count", len(response.Warnings)))

	return &response, nil
}

// GetMedicationHistory retrieves medication history for a patient
func (c *medicationServiceClientImpl) GetMedicationHistory(ctx context.Context, patientID string) (*MedicationHistoryResponse, error) {
	c.logger.Info("Retrieving medication history",
		zap.String("patient_id", patientID))

	// Create HTTP request
	url := fmt.Sprintf("%s/api/v1/medication/history/%s", c.baseURL, patientID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("User-Agent", "workflow-engine-service/1.0")

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.logger.Error("Failed to get medication history",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusNotFound {
		return &MedicationHistoryResponse{
			PatientID:         patientID,
			MedicationOrders:  []map[string]interface{}{},
			ActiveMedications: []map[string]interface{}{},
			RecentChanges:     []map[string]interface{}{},
			LastUpdated:       time.Now(),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Medication Service returned status %d", resp.StatusCode)
	}

	// Parse response
	var response MedicationHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Medication history retrieved",
		zap.String("patient_id", patientID),
		zap.Int("order_count", len(response.MedicationOrders)),
		zap.Int("active_count", len(response.ActiveMedications)),
		zap.Int("recent_changes", len(response.RecentChanges)))

	return &response, nil
}

// HealthCheck checks if Medication Service is healthy
func (c *medicationServiceClientImpl) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Medication Service unhealthy: status %d", resp.StatusCode)
	}

	// Parse health response to check service components
	var healthResponse struct {
		Status     string            `json:"status"`
		Components map[string]string `json:"components,omitempty"`
		Database   string            `json:"database,omitempty"`
		FHIR       string            `json:"fhir,omitempty"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err != nil {
		c.logger.Warn("Failed to decode health response", zap.Error(err))
		// Still return nil if basic health check passed
		return nil
	}

	// Log component status
	if healthResponse.Components != nil {
		for component, status := range healthResponse.Components {
			if status != "healthy" && status != "ok" {
				c.logger.Warn("Medication Service component unhealthy",
					zap.String("component", component),
					zap.String("status", status))
			}
		}
	}

	return nil
}