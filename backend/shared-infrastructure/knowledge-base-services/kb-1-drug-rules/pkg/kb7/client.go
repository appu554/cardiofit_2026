package kb7

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/sirupsen/logrus"
)

// Client provides access to KB-7 Terminology Service for RxNorm validation
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// Config holds KB-7 client configuration
type Config struct {
	BaseURL     string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
}

// DefaultConfig returns default KB-7 client configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:    "http://localhost:8092",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 500 * time.Millisecond,
	}
}

// NewClient creates a new KB-7 terminology client
func NewClient(baseURL string, log *logrus.Entry) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log: log.WithField("component", "kb7-client"),
	}
}

// NewClientWithConfig creates a client with custom configuration
func NewClientWithConfig(cfg Config, log *logrus.Entry) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		log: log.WithField("component", "kb7-client"),
	}
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// Health checks KB-7 service health
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-7 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// =============================================================================
// RXNORM RESOLUTION
// =============================================================================

// RxNormConcept represents an RxNorm concept from KB-7
type RxNormConcept struct {
	RxNormCode   string   `json:"rxnorm_code"`
	Name         string   `json:"name"`
	TTY          string   `json:"tty"`           // Term Type (IN, BN, SCD, etc.)
	GenericName  string   `json:"generic_name"`
	BrandNames   []string `json:"brand_names"`
	DrugClass    string   `json:"drug_class"`
	ATCCodes     []string `json:"atc_codes"`
	NDCs         []string `json:"ndcs"`
	Ingredients  []string `json:"ingredients"`
	DoseForms    []string `json:"dose_forms"`
	Strengths    []string `json:"strengths"`
}

// ResolveNameToRxNorm resolves a drug name to RxNorm code
func (c *Client) ResolveNameToRxNorm(drugName string) (string, error) {
	return c.ResolveNameToRxNormWithContext(context.Background(), drugName)
}

// ResolveNameToRxNormWithContext resolves a drug name to RxNorm code with context
func (c *Client) ResolveNameToRxNormWithContext(ctx context.Context, drugName string) (string, error) {
	if drugName == "" {
		return "", fmt.Errorf("drug name cannot be empty")
	}

	endpoint := fmt.Sprintf("%s/v1/terminology/rxnorm/search?q=%s", c.baseURL, url.QueryEscape(drugName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // Drug not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Results []RxNormConcept `json:"results"`
		Count   int             `json:"count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Results) == 0 {
		return "", nil // No results
	}

	// Return the first (most relevant) result
	return result.Results[0].RxNormCode, nil
}

// GetRxNormConcept retrieves full concept details by RxNorm code
func (c *Client) GetRxNormConcept(ctx context.Context, rxnormCode string) (*RxNormConcept, error) {
	endpoint := fmt.Sprintf("%s/v1/terminology/rxnorm/%s", c.baseURL, rxnormCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var concept RxNormConcept
	if err := json.NewDecoder(resp.Body).Decode(&concept); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &concept, nil
}

// ValidateRxNormCode checks if an RxNorm code is valid
func (c *Client) ValidateRxNormCode(ctx context.Context, rxnormCode string) (bool, error) {
	concept, err := c.GetRxNormConcept(ctx, rxnormCode)
	if err != nil {
		return false, err
	}
	return concept != nil, nil
}

// =============================================================================
// ATC CODE MAPPING
// =============================================================================

// GetATCForRxNorm retrieves ATC codes for an RxNorm code
func (c *Client) GetATCForRxNorm(ctx context.Context, rxnormCode string) ([]string, error) {
	concept, err := c.GetRxNormConcept(ctx, rxnormCode)
	if err != nil {
		return nil, err
	}
	if concept == nil {
		return nil, nil
	}
	return concept.ATCCodes, nil
}

// =============================================================================
// DRUG CLASS LOOKUP
// =============================================================================

// GetDrugClass retrieves the drug class for an RxNorm code
func (c *Client) GetDrugClass(ctx context.Context, rxnormCode string) (string, error) {
	endpoint := fmt.Sprintf("%s/v1/terminology/rxnorm/%s/class", c.baseURL, rxnormCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		DrugClass string `json:"drug_class"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.DrugClass, nil
}

// =============================================================================
// NDC LOOKUP
// =============================================================================

// GetNDCsForRxNorm retrieves NDC codes for an RxNorm code
func (c *Client) GetNDCsForRxNorm(ctx context.Context, rxnormCode string) ([]string, error) {
	concept, err := c.GetRxNormConcept(ctx, rxnormCode)
	if err != nil {
		return nil, err
	}
	if concept == nil {
		return nil, nil
	}
	return concept.NDCs, nil
}

// ResolveNDCToRxNorm converts an NDC code to RxNorm code
func (c *Client) ResolveNDCToRxNorm(ctx context.Context, ndc string) (string, error) {
	endpoint := fmt.Sprintf("%s/v1/terminology/ndc/%s/rxnorm", c.baseURL, ndc)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		RxNormCode string `json:"rxnorm_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.RxNormCode, nil
}

// =============================================================================
// RELATED DRUGS
// =============================================================================

// RelatedDrug represents a related drug from KB-7
type RelatedDrug struct {
	RxNormCode   string `json:"rxnorm_code"`
	Name         string `json:"name"`
	Relationship string `json:"relationship"` // INGREDIENT_OF, TRADENAME_OF, etc.
}

// GetRelatedDrugs retrieves related drugs (ingredients, brand names, etc.)
func (c *Client) GetRelatedDrugs(ctx context.Context, rxnormCode string) ([]RelatedDrug, error) {
	endpoint := fmt.Sprintf("%s/v1/terminology/rxnorm/%s/related", c.baseURL, rxnormCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Related []RelatedDrug `json:"related"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Related, nil
}

// =============================================================================
// BATCH OPERATIONS
// =============================================================================

// BatchResolveNames resolves multiple drug names to RxNorm codes
func (c *Client) BatchResolveNames(ctx context.Context, drugNames []string) (map[string]string, error) {
	results := make(map[string]string)

	for _, name := range drugNames {
		rxnormCode, err := c.ResolveNameToRxNormWithContext(ctx, name)
		if err != nil {
			c.log.WithError(err).WithField("drug_name", name).Warn("Failed to resolve drug name")
			continue
		}
		if rxnormCode != "" {
			results[name] = rxnormCode
		}
	}

	return results, nil
}

// BatchValidateCodes validates multiple RxNorm codes
func (c *Client) BatchValidateCodes(ctx context.Context, codes []string) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, code := range codes {
		valid, err := c.ValidateRxNormCode(ctx, code)
		if err != nil {
			c.log.WithError(err).WithField("rxnorm_code", code).Warn("Failed to validate code")
			results[code] = false
			continue
		}
		results[code] = valid
	}

	return results, nil
}

// =============================================================================
// SNOMED MAPPING (if KB-7 supports it)
// =============================================================================

// GetSNOMEDForRxNorm retrieves SNOMED-CT code for an RxNorm code
func (c *Client) GetSNOMEDForRxNorm(ctx context.Context, rxnormCode string) (string, error) {
	endpoint := fmt.Sprintf("%s/v1/terminology/rxnorm/%s/snomed", c.baseURL, rxnormCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil // No SNOMED mapping
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		SNOMEDCode string `json:"snomed_code"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.SNOMEDCode, nil
}
