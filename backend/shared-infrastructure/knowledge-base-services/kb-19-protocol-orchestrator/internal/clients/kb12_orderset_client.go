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

// KB12OrderSetClient is the HTTP client for KB-12 OrderSets/CarePlans service.
// KB-12 provides order set activation - converting abstract clinical decisions
// into concrete FHIR orders (MedicationRequest, ServiceRequest, etc.).
type KB12OrderSetClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB12OrderSetClient creates a new KB12OrderSetClient.
func NewKB12OrderSetClient(baseURL string, timeout time.Duration, log *logrus.Entry) *KB12OrderSetClient {
	return &KB12OrderSetClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.WithField("client", "kb12-orderset"),
	}
}

// OrderSetActivationRequest is the request to activate an order set.
// Note: KB-19's request format - will be converted to KB-12 format in ActivateOrderSet.
type OrderSetActivationRequest struct {
	PatientID      uuid.UUID              `json:"patient_id"`
	EncounterID    uuid.UUID              `json:"encounter_id"`
	OrderSetID     string                 `json:"order_set_id"`     // Maps to template_id in KB-12
	DecisionID     uuid.UUID              `json:"decision_id"`
	Target         string                 `json:"target"`           // Medication or procedure name
	TargetRxNorm   string                 `json:"target_rxnorm,omitempty"`
	TargetSNOMED   string                 `json:"target_snomed,omitempty"`
	Customizations map[string]interface{} `json:"customizations,omitempty"`
	Urgency        string                 `json:"urgency"`
	SourceProtocol string                 `json:"source_protocol"`
	Rationale      string                 `json:"rationale"`
}

// KB12ActivationRequest is the request format expected by KB-12's /api/v1/activate endpoint.
type KB12ActivationRequest struct {
	TemplateID     string                 `json:"template_id"`
	PatientID      string                 `json:"patient_id"`
	EncounterID    string                 `json:"encounter_id"`
	ActivatedBy    string                 `json:"activated_by"`
	SelectedOrders []string               `json:"selected_orders,omitempty"`
	Customizations map[string]interface{} `json:"customizations,omitempty"`
}

// OrderSetActivation is the response from activating an order set.
type OrderSetActivation struct {
	ID             uuid.UUID        `json:"id"`
	OrderSetID     string           `json:"order_set_id"`
	OrderSetName   string           `json:"order_set_name"`
	Status         string           `json:"status"`         // DRAFT, PENDING_SIGNATURE, ACTIVE, COMPLETED
	GeneratedOrders []GeneratedOrder `json:"generated_orders"`
	CarePlanRef    string           `json:"care_plan_ref,omitempty"`
	ActivatedAt    time.Time        `json:"activated_at"`
	ActivatedBy    string           `json:"activated_by"`
}

// GeneratedOrder represents a single order generated from an order set.
type GeneratedOrder struct {
	ID           uuid.UUID `json:"id"`
	OrderType    string    `json:"order_type"`    // MEDICATION, LAB, IMAGING, PROCEDURE, REFERRAL
	FHIRResource string    `json:"fhir_resource"` // MedicationRequest, ServiceRequest, etc.
	FHIRRef      string    `json:"fhir_ref"`      // Reference to the FHIR resource
	Display      string    `json:"display"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	Details      OrderDetails `json:"details"`
}

// OrderDetails contains order-specific details.
type OrderDetails struct {
	// For medications
	Medication   string `json:"medication,omitempty"`
	Dose         string `json:"dose,omitempty"`
	Route        string `json:"route,omitempty"`
	Frequency    string `json:"frequency,omitempty"`
	Duration     string `json:"duration,omitempty"`
	Instructions string `json:"instructions,omitempty"`

	// For labs
	TestName     string `json:"test_name,omitempty"`
	Specimen     string `json:"specimen,omitempty"`

	// For imaging
	Modality     string `json:"modality,omitempty"`
	BodySite     string `json:"body_site,omitempty"`

	// For referrals
	Specialty    string `json:"specialty,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// AvailableOrderSet describes an available order set.
type AvailableOrderSet struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Conditions   []string `json:"conditions"`      // Applicable conditions
	OrderCount   int      `json:"order_count"`
	Version      string   `json:"version"`
	IsActive     bool     `json:"is_active"`
}

// CarePlanRequest is the request to create a care plan.
type CarePlanRequest struct {
	PatientID       uuid.UUID            `json:"patient_id"`
	EncounterID     uuid.UUID            `json:"encounter_id"`
	Title           string               `json:"title"`
	Description     string               `json:"description"`
	Category        string               `json:"category"`
	Period          CarePlanPeriod       `json:"period"`
	Goals           []CarePlanGoal       `json:"goals"`
	Activities      []CarePlanActivity   `json:"activities"`
	SourceDecisions []uuid.UUID          `json:"source_decisions"`
}

// CarePlanPeriod defines the care plan duration.
type CarePlanPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end,omitempty"`
}

// CarePlanGoal defines a care plan goal.
type CarePlanGoal struct {
	Description   string `json:"description"`
	TargetValue   string `json:"target_value,omitempty"`
	TargetDate    time.Time `json:"target_date,omitempty"`
	Priority      string `json:"priority"`
}

// CarePlanActivity defines a care plan activity.
type CarePlanActivity struct {
	Kind        string `json:"kind"`        // MEDICATION, LAB, IMAGING, PROCEDURE
	Description string `json:"description"`
	Timing      string `json:"timing"`
	Status      string `json:"status"`
}

// CarePlan is the response from creating a care plan.
type CarePlan struct {
	ID          uuid.UUID          `json:"id"`
	FHIRRef     string             `json:"fhir_ref"`
	Title       string             `json:"title"`
	Status      string             `json:"status"`
	Period      CarePlanPeriod     `json:"period"`
	Goals       []CarePlanGoal     `json:"goals"`
	Activities  []CarePlanActivity `json:"activities"`
	CreatedAt   time.Time          `json:"created_at"`
}

// ActivateOrderSet activates an order set for a patient.
// Converts KB-19's request format to KB-12's expected format.
func (c *KB12OrderSetClient) ActivateOrderSet(ctx context.Context, req OrderSetActivationRequest) (*OrderSetActivation, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":   req.PatientID,
		"order_set_id": req.OrderSetID,
		"decision_id":  req.DecisionID,
	}).Debug("Activating order set")

	// Convert to KB-12's expected format
	kb12Req := KB12ActivationRequest{
		TemplateID:     req.OrderSetID,
		PatientID:      req.PatientID.String(),
		EncounterID:    req.EncounterID.String(),
		ActivatedBy:    "KB-19-Orchestrator", // System activation
		Customizations: req.Customizations,
	}

	body, err := json.Marshal(kb12Req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// KB-12 uses /api/v1/activate for order set activation
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/activate", bytes.NewReader(body))
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
		return nil, fmt.Errorf("order set activation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result OrderSetActivation
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"activation_id":  result.ID,
		"order_set_id":   req.OrderSetID,
		"orders_generated": len(result.GeneratedOrders),
	}).Debug("Order set activated")

	return &result, nil
}

// CreateCarePlan creates a care plan for a patient.
func (c *KB12OrderSetClient) CreateCarePlan(ctx context.Context, req CarePlanRequest) (*CarePlan, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id": req.PatientID,
		"title":      req.Title,
	}).Debug("Creating care plan")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/careplans", bytes.NewReader(body))
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
		return nil, fmt.Errorf("care plan creation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result CarePlan
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListOrderSets lists available order sets (templates in KB-12).
// KB-12 uses /api/v1/templates for listing order set templates.
func (c *KB12OrderSetClient) ListOrderSets(ctx context.Context, category string) ([]AvailableOrderSet, error) {
	url := c.baseURL + "/api/v1/templates"
	if category != "" {
		url += "?category=" + category
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list order sets failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result []AvailableOrderSet
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// FindOrderSetForTarget finds the appropriate order set (template) for a given target.
// KB-12 uses /api/v1/templates/search for searching order set templates.
// Query parameter is "q" (not "query").
func (c *KB12OrderSetClient) FindOrderSetForTarget(ctx context.Context, target string, category string) (*AvailableOrderSet, error) {
	url := fmt.Sprintf("%s/api/v1/templates/search?q=%s", c.baseURL, target)
	if category != "" {
		url += "&category=" + category
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No order set found for target
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search order set failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result AvailableOrderSet
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Health checks if KB-12 is healthy.
func (c *KB12OrderSetClient) Health(ctx context.Context) error {
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
