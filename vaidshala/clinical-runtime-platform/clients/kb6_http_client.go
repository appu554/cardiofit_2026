// Package clients provides HTTP clients for KB services.
//
// KB6HTTPClient implements the KB6Client interface for KB-6 Formulary Service.
// It provides formulary status, prior auth, generic alternatives, and regional
// availability (NLEM for India, PBS for Australia).
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client is used by KnowledgeSnapshotBuilder to populate FormularySnapshot.
// All formulary checks are pre-computed at snapshot build time - engines NEVER
// call formulary services directly at execution time.
//
// Connects to: http://localhost:8086 (Docker: kb6-formulary)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// KB6HTTPClient implements KB6Client by calling the KB-6 Formulary Service REST API.
type KB6HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB6HTTPClient creates a new KB-6 HTTP client.
func NewKB6HTTPClient(baseURL string) *KB6HTTPClient {
	return &KB6HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB6HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB6HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB6HTTPClient {
	return &KB6HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB6Client Interface Implementation
// ============================================================================

// GetFormularyStatus returns formulary status for medications.
// Calls KB-6 /api/v1/formulary/status endpoint.
func (c *KB6HTTPClient) GetFormularyStatus(
	ctx context.Context,
	medications []contracts.ClinicalCode,
	region string,
) (map[string]contracts.FormularyStatus, error) {

	statuses := make(map[string]contracts.FormularyStatus)

	if len(medications) == 0 {
		return statuses, nil
	}

	// Build medication code list
	medCodes := make([]string, 0, len(medications))
	for _, med := range medications {
		medCodes = append(medCodes, med.Code)
	}

	req := kb6StatusRequest{
		MedicationCodes: medCodes,
		Region:          region,
	}

	resp, err := c.callKB6(ctx, "/api/v1/formulary/status", req)
	if err != nil {
		return statuses, nil // Return empty on error
	}

	var result kb6StatusResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return statuses, nil
	}

	for _, s := range result.Statuses {
		key := fmt.Sprintf("http://www.nlm.nih.gov/research/umls/rxnorm|%s", s.MedicationCode)

		// Map KB-6 response to contract FormularyStatus
		// Determine status based on formulary and tier
		status := "not-listed"
		if s.IsOnFormulary {
			if s.Tier <= 1 {
				status = "preferred"
			} else {
				status = "non-preferred"
			}
		}

		// Build restrictions list from flags
		var restrictions []string
		if s.RequiresPriorAuth {
			restrictions = append(restrictions, "prior-authorization-required")
		}
		if s.StepTherapy {
			restrictions = append(restrictions, "step-therapy-required")
		}
		if s.QuantityLimit {
			restrictions = append(restrictions, "quantity-limit")
		}

		statuses[key] = contracts.FormularyStatus{
			Code: contracts.ClinicalCode{
				System: "http://www.nlm.nih.gov/research/umls/rxnorm",
				Code:   s.MedicationCode,
			},
			Status:        status,
			FormularyName: s.CoverageType,
			Tier:          s.Tier,
			Restrictions:  restrictions,
			CopayAmount:   s.Copay,
		}
	}

	return statuses, nil
}

// GetPriorAuthRequired returns medications needing prior authorization.
// Calls KB-6 /api/v1/formulary/prior-auth endpoint.
func (c *KB6HTTPClient) GetPriorAuthRequired(
	ctx context.Context,
	medications []contracts.ClinicalCode,
) ([]contracts.ClinicalCode, error) {

	if len(medications) == 0 {
		return []contracts.ClinicalCode{}, nil
	}

	medCodes := make([]string, 0, len(medications))
	for _, med := range medications {
		medCodes = append(medCodes, med.Code)
	}

	req := kb6PriorAuthRequest{
		MedicationCodes: medCodes,
	}

	resp, err := c.callKB6(ctx, "/api/v1/formulary/prior-auth", req)
	if err != nil {
		return []contracts.ClinicalCode{}, nil
	}

	var result kb6PriorAuthResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return []contracts.ClinicalCode{}, nil
	}

	priorAuthMeds := make([]contracts.ClinicalCode, 0, len(result.RequiresPriorAuth))
	for _, pa := range result.RequiresPriorAuth {
		priorAuthMeds = append(priorAuthMeds, contracts.ClinicalCode{
			System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
			Code:    pa.MedicationCode,
			Display: pa.MedicationName,
		})
	}

	return priorAuthMeds, nil
}

// GetGenericAlternatives returns generic alternatives for brand drugs.
// Calls KB-6 /api/v1/formulary/alternatives endpoint.
func (c *KB6HTTPClient) GetGenericAlternatives(
	ctx context.Context,
	medication contracts.ClinicalCode,
) ([]contracts.ClinicalCode, error) {

	req := kb6AlternativesRequest{
		MedicationCode: medication.Code,
		IncludeGenerics: true,
	}

	resp, err := c.callKB6(ctx, "/api/v1/formulary/alternatives", req)
	if err != nil {
		return []contracts.ClinicalCode{}, nil
	}

	var result kb6AlternativesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return []contracts.ClinicalCode{}, nil
	}

	alternatives := make([]contracts.ClinicalCode, 0, len(result.Alternatives))
	for _, alt := range result.Alternatives {
		alternatives = append(alternatives, contracts.ClinicalCode{
			System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
			Code:    alt.Code,
			Display: alt.Name,
		})
	}

	return alternatives, nil
}

// GetNLEMAvailability checks if drugs are on India National List of Essential Medicines.
// Calls KB-6 /api/v1/formulary/nlem endpoint.
func (c *KB6HTTPClient) GetNLEMAvailability(
	ctx context.Context,
	medications []contracts.ClinicalCode,
) (map[string]bool, error) {

	availability := make(map[string]bool)

	if len(medications) == 0 {
		return availability, nil
	}

	medCodes := make([]string, 0, len(medications))
	for _, med := range medications {
		medCodes = append(medCodes, med.Code)
	}

	req := kb6NLEMRequest{
		MedicationCodes: medCodes,
	}

	resp, err := c.callKB6(ctx, "/api/v1/formulary/nlem", req)
	if err != nil {
		return availability, nil
	}

	var result kb6NLEMResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return availability, nil
	}

	for code, onNLEM := range result.Availability {
		key := fmt.Sprintf("http://www.nlm.nih.gov/research/umls/rxnorm|%s", code)
		availability[key] = onNLEM
	}

	return availability, nil
}

// GetPBSAvailability checks if drugs are on Australia Pharmaceutical Benefits Scheme.
// Calls KB-6 /api/v1/formulary/pbs endpoint.
func (c *KB6HTTPClient) GetPBSAvailability(
	ctx context.Context,
	medications []contracts.ClinicalCode,
) (map[string]bool, error) {

	availability := make(map[string]bool)

	if len(medications) == 0 {
		return availability, nil
	}

	medCodes := make([]string, 0, len(medications))
	for _, med := range medications {
		medCodes = append(medCodes, med.Code)
	}

	req := kb6PBSRequest{
		MedicationCodes: medCodes,
	}

	resp, err := c.callKB6(ctx, "/api/v1/formulary/pbs", req)
	if err != nil {
		return availability, nil
	}

	var result kb6PBSResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return availability, nil
	}

	for code, onPBS := range result.Availability {
		key := fmt.Sprintf("http://www.nlm.nih.gov/research/umls/rxnorm|%s", code)
		availability[key] = onPBS
	}

	return availability, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB6HTTPClient) callKB6(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-6 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-6 service is healthy.
func (c *KB6HTTPClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-6 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// KB-6 Request/Response Types (internal)
// ============================================================================

type kb6StatusRequest struct {
	MedicationCodes []string `json:"medication_codes"`
	Region          string   `json:"region"`
}

type kb6StatusResult struct {
	Statuses []kb6Status `json:"statuses"`
}

type kb6Status struct {
	MedicationCode    string  `json:"medication_code"`
	IsOnFormulary     bool    `json:"is_on_formulary"`
	Tier              int     `json:"tier"`
	RequiresPriorAuth bool    `json:"requires_prior_auth"`
	StepTherapy       bool    `json:"step_therapy"`
	QuantityLimit     bool    `json:"quantity_limit"`
	CoverageType      string  `json:"coverage_type"`
	Copay             float64 `json:"copay"`
	CopayTier         string  `json:"copay_tier"`
}

type kb6PriorAuthRequest struct {
	MedicationCodes []string `json:"medication_codes"`
}

type kb6PriorAuthResult struct {
	RequiresPriorAuth []kb6PriorAuth `json:"requires_prior_auth"`
}

type kb6PriorAuth struct {
	MedicationCode string `json:"medication_code"`
	MedicationName string `json:"medication_name"`
	Reason         string `json:"reason"`
}

type kb6AlternativesRequest struct {
	MedicationCode  string `json:"medication_code"`
	IncludeGenerics bool   `json:"include_generics"`
}

type kb6AlternativesResult struct {
	Alternatives []kb6Alternative `json:"alternatives"`
}

type kb6Alternative struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Type       string  `json:"type"` // "generic", "therapeutic", "biosimilar"
	CostSaving float64 `json:"cost_saving"`
}

type kb6NLEMRequest struct {
	MedicationCodes []string `json:"medication_codes"`
}

type kb6NLEMResult struct {
	Availability map[string]bool `json:"availability"`
}

type kb6PBSRequest struct {
	MedicationCodes []string `json:"medication_codes"`
}

type kb6PBSResult struct {
	Availability map[string]bool `json:"availability"`
}
