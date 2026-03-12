// Package clients provides HTTP clients for KB services.
//
// KB5HTTPClient implements the KB5Client interface for KB-5 Drug Interactions Service.
// It provides drug-drug interaction checking for current and potential medications.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client is used by KnowledgeSnapshotBuilder to populate InteractionSnapshot.
// All interaction checks are pre-computed at snapshot build time - engines NEVER
// call interaction services directly at execution time.
//
// Connects to: http://localhost:8095 (Docker: kb5-drug-interactions, maps 8095->8085)
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

// KB5HTTPClient implements KB5Client by calling the KB-5 Drug Interactions Service REST API.
type KB5HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB5HTTPClient creates a new KB-5 HTTP client.
func NewKB5HTTPClient(baseURL string) *KB5HTTPClient {
	return &KB5HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB5HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB5HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB5HTTPClient {
	return &KB5HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB5Client Interface Implementation
// ============================================================================

// GetCurrentInteractions checks interactions between patient's active medications.
// Calls KB-5 /api/v1/interactions/check endpoint.
func (c *KB5HTTPClient) GetCurrentInteractions(
	ctx context.Context,
	medications []contracts.ClinicalCode,
) ([]contracts.DrugInteraction, error) {

	if len(medications) < 2 {
		// Need at least 2 drugs for interaction check
		return []contracts.DrugInteraction{}, nil
	}

	// Build drug codes array (KB-5 expects array of RxNorm code strings)
	drugCodes := make([]string, 0, len(medications))
	for _, med := range medications {
		drugCodes = append(drugCodes, med.Code)
	}

	req := kb5InteractionRequest{
		DrugCodes:           drugCodes,
		CheckType:           "comprehensive",
		IncludeAlternatives: true,
		IncludeMonitoring:   true,
	}

	resp, err := c.callKB5(ctx, "/api/v1/interactions/check", req)
	if err != nil {
		return nil, fmt.Errorf("failed to check interactions: %w", err)
	}

	var result kb5InteractionResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse interaction response: %w", err)
	}

	interactions := make([]contracts.DrugInteraction, 0, len(result.InteractionsFound))
	for _, i := range result.InteractionsFound {
		// Build comprehensive description including clinical effect
		description := i.Description
		if i.ClinicalEffect != "" {
			description = fmt.Sprintf("%s Clinical effect: %s", description, i.ClinicalEffect)
		}

		interactions = append(interactions, contracts.DrugInteraction{
			Drug1: contracts.ClinicalCode{
				System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
				Code:    i.Drug1Code,
				Display: i.Drug1Name,
			},
			Drug2: contracts.ClinicalCode{
				System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
				Code:    i.Drug2Code,
				Display: i.Drug2Name,
			},
			Severity:       i.Severity,
			Description:    description,
			Recommendation: i.Management,
			Evidence:       fmt.Sprintf("%s (%s)", i.EvidenceLevel, i.Source),
		})
	}

	return interactions, nil
}

// GetPotentialInteractions checks interactions if adding common drugs.
// Calls KB-5 /api/v1/interactions/check endpoint with current drugs.
// Note: KB-5 uses the same endpoint for potential checks with different check_type.
func (c *KB5HTTPClient) GetPotentialInteractions(
	ctx context.Context,
	medications []contracts.ClinicalCode,
) ([]contracts.DrugInteraction, error) {

	if len(medications) == 0 {
		return []contracts.DrugInteraction{}, nil
	}

	// Build drug codes array (KB-5 expects array of RxNorm code strings)
	drugCodes := make([]string, 0, len(medications))
	for _, med := range medications {
		drugCodes = append(drugCodes, med.Code)
	}

	// Use the same interaction check endpoint with a "potential" check type
	req := kb5InteractionRequest{
		DrugCodes:           drugCodes,
		CheckType:           "quick", // Quick check for potential interactions
		IncludeAlternatives: true,
		IncludeMonitoring:   false,
	}

	resp, err := c.callKB5(ctx, "/api/v1/interactions/check", req)
	if err != nil {
		// Log but return empty - potential interactions are advisory
		return []contracts.DrugInteraction{}, nil
	}

	// Use the same response format as GetCurrentInteractions since same endpoint
	var result kb5InteractionResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return []contracts.DrugInteraction{}, nil
	}

	interactions := make([]contracts.DrugInteraction, 0, len(result.InteractionsFound))
	for _, i := range result.InteractionsFound {
		// Build comprehensive description including clinical effect
		description := fmt.Sprintf("[Potential] %s", i.Description)
		if i.ClinicalEffect != "" {
			description = fmt.Sprintf("%s Clinical effect: %s", description, i.ClinicalEffect)
		}

		interactions = append(interactions, contracts.DrugInteraction{
			Drug1: contracts.ClinicalCode{
				System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
				Code:    i.Drug1Code,
				Display: i.Drug1Name,
			},
			Drug2: contracts.ClinicalCode{
				System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
				Code:    i.Drug2Code,
				Display: i.Drug2Name,
			},
			Severity:       i.Severity,
			Description:    description,
			Recommendation: i.Management,
			Evidence:       fmt.Sprintf("%s (%s)", i.EvidenceLevel, i.Source),
		})
	}

	return interactions, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB5HTTPClient) callKB5(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
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
		return nil, fmt.Errorf("KB-5 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-5 service is healthy.
func (c *KB5HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-5 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// KB-5 Request/Response Types (internal)
// ============================================================================

// kb5InteractionRequest matches the actual KB-5 API contract.
// KB-5 expects drug_codes as an array of RxNorm code strings, NOT objects.
type kb5InteractionRequest struct {
	DrugCodes           []string               `json:"drug_codes"`
	PatientID           string                 `json:"patient_id,omitempty"`
	CheckType           string                 `json:"check_type,omitempty"`
	ClinicalContext     map[string]interface{} `json:"clinical_context,omitempty"`
	SeverityFilter      []string               `json:"severity_filter,omitempty"`
	IncludeAlternatives bool                   `json:"include_alternatives"`
	IncludeMonitoring   bool                   `json:"include_monitoring"`
}

type kb5InteractionResult struct {
	InteractionsFound []kb5Interaction `json:"interactions_found"`
	Summary           kb5Summary       `json:"summary"`
}

type kb5Interaction struct {
	Drug1Code       string `json:"drug1_code"`
	Drug1Name       string `json:"drug1_name"`
	Drug2Code       string `json:"drug2_code"`
	Drug2Name       string `json:"drug2_name"`
	Severity        string `json:"severity"`
	InteractionType string `json:"interaction_type"`
	Description     string `json:"description"`
	ClinicalEffect  string `json:"clinical_effect"`
	Management      string `json:"management"`
	EvidenceLevel   string `json:"evidence_level"`
	Source          string `json:"source"`
}

type kb5Summary struct {
	TotalInteractions   int  `json:"total_interactions"`
	CriticalCount       int  `json:"critical_count"`
	SevereCount         int  `json:"severe_count"`
	ModerateCount       int  `json:"moderate_count"`
	HasContraindication bool `json:"has_contraindication"`
}
