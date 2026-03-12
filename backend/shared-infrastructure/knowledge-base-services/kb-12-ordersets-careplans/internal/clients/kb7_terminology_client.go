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

	"kb-12-ordersets-careplans/internal/config"
)

// KB7TerminologyClient provides HTTP client for KB-7 Terminology service
type KB7TerminologyClient struct {
	baseURL    string
	httpClient *http.Client
	config     config.KBClientConfig
	log        *logrus.Entry
}

// CodeLookupRequest represents a request to lookup a clinical code
type CodeLookupRequest struct {
	Code       string `json:"code"`
	System     string `json:"system"` // snomed, icd10, loinc, rxnorm, cpt
}

// CodeLookupResponse represents code lookup result
type CodeLookupResponse struct {
	Success      bool        `json:"success"`
	Code         string      `json:"code"`
	System       string      `json:"system"`
	Display      string      `json:"display"`
	Definition   string      `json:"definition,omitempty"`
	ParentCodes  []CodeRef   `json:"parent_codes,omitempty"`
	ChildCodes   []CodeRef   `json:"child_codes,omitempty"`
	Synonyms     []string    `json:"synonyms,omitempty"`
	Active       bool        `json:"active"`
	EffectiveDate time.Time  `json:"effective_date,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
}

// CodeRef represents a reference to another code
type CodeRef struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
}

// CodeValidationRequest represents a request to validate codes
type CodeValidationRequest struct {
	Codes []CodeLookupRequest `json:"codes"`
}

// CodeValidationResponse represents validation results
type CodeValidationResponse struct {
	Success       bool               `json:"success"`
	TotalCodes    int                `json:"total_codes"`
	ValidCodes    int                `json:"valid_codes"`
	InvalidCodes  int                `json:"invalid_codes"`
	Results       []CodeValidation   `json:"results"`
	ErrorMessage  string             `json:"error_message,omitempty"`
}

// CodeValidation represents validation result for a single code
type CodeValidation struct {
	Code         string `json:"code"`
	System       string `json:"system"`
	Valid        bool   `json:"valid"`
	Display      string `json:"display,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ValueSetMembershipRequest represents a request to check value set membership
type ValueSetMembershipRequest struct {
	Code       string `json:"code"`
	System     string `json:"system"`
	ValueSetID string `json:"valueset_id"`
}

// ValueSetMembershipResponse represents value set membership result
type ValueSetMembershipResponse struct {
	Success      bool   `json:"success"`
	IsMember     bool   `json:"is_member"`
	ValueSetID   string `json:"valueset_id"`
	ValueSetName string `json:"valueset_name"`
	Code         string `json:"code"`
	Display      string `json:"display,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// CodeTranslationRequest represents a request to translate codes between systems
type CodeTranslationRequest struct {
	SourceCode   string `json:"source_code"`
	SourceSystem string `json:"source_system"`
	TargetSystem string `json:"target_system"`
}

// CodeTranslationResponse represents code translation result
type CodeTranslationResponse struct {
	Success      bool      `json:"success"`
	SourceCode   string    `json:"source_code"`
	SourceSystem string    `json:"source_system"`
	TargetSystem string    `json:"target_system"`
	Translations []CodeRef `json:"translations"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// ValueSet represents a value set definition
type ValueSet struct {
	ValueSetID   string    `json:"valueset_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Version      string    `json:"version"`
	Status       string    `json:"status"`
	Publisher    string    `json:"publisher"`
	Codes        []CodeRef `json:"codes"`
	ExpansionDate time.Time `json:"expansion_date"`
}

// NewKB7TerminologyClient creates a new KB-7 Terminology HTTP client
func NewKB7TerminologyClient(cfg config.KBClientConfig) *KB7TerminologyClient {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableKeepAlives:   false,
	}

	return &KB7TerminologyClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		config: cfg,
		log:    logrus.WithField("client", "kb7-terminology"),
	}
}

// IsEnabled returns whether the KB-7 client is enabled
func (c *KB7TerminologyClient) IsEnabled() bool {
	return c.config.Enabled
}

// Health checks if KB-7 service is healthy
func (c *KB7TerminologyClient) Health(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-7 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-7 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// LookupCode looks up a clinical code
func (c *KB7TerminologyClient) LookupCode(ctx context.Context, code string, system string) (*CodeLookupResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-7 client disabled, returning basic code response")
		return &CodeLookupResponse{
			Success: true,
			Code:    code,
			System:  system,
			Display: fmt.Sprintf("Code %s (%s)", code, system),
			Active:  true,
		}, nil
	}

	var resp *CodeLookupResponse
	// KB-7 uses /v1/concepts/:system/:code endpoint
	endpoint := fmt.Sprintf("/v1/concepts/%s/%s", system, code)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// LookupSNOMED looks up a SNOMED-CT code
func (c *KB7TerminologyClient) LookupSNOMED(ctx context.Context, code string) (*CodeLookupResponse, error) {
	return c.LookupCode(ctx, code, "snomed")
}

// LookupICD10 looks up an ICD-10 code
func (c *KB7TerminologyClient) LookupICD10(ctx context.Context, code string) (*CodeLookupResponse, error) {
	return c.LookupCode(ctx, code, "icd10")
}

// LookupLOINC looks up a LOINC code
func (c *KB7TerminologyClient) LookupLOINC(ctx context.Context, code string) (*CodeLookupResponse, error) {
	return c.LookupCode(ctx, code, "loinc")
}

// LookupRxNorm looks up an RxNorm code
func (c *KB7TerminologyClient) LookupRxNorm(ctx context.Context, code string) (*CodeLookupResponse, error) {
	return c.LookupCode(ctx, code, "rxnorm")
}

// LookupCPT looks up a CPT code
func (c *KB7TerminologyClient) LookupCPT(ctx context.Context, code string) (*CodeLookupResponse, error) {
	return c.LookupCode(ctx, code, "cpt")
}

// ValidateCodes validates multiple codes at once
func (c *KB7TerminologyClient) ValidateCodes(ctx context.Context, req *CodeValidationRequest) (*CodeValidationResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-7 client disabled, returning all codes as valid")
		results := make([]CodeValidation, len(req.Codes))
		for i, code := range req.Codes {
			results[i] = CodeValidation{
				Code:    code.Code,
				System:  code.System,
				Valid:   true,
				Display: fmt.Sprintf("Code %s", code.Code),
			}
		}
		return &CodeValidationResponse{
			Success:      true,
			TotalCodes:  len(req.Codes),
			ValidCodes:  len(req.Codes),
			InvalidCodes: 0,
			Results:     results,
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *CodeValidationResponse
	// KB-7 uses /v1/concepts/batch-validate for multiple code validation
	err = c.doRequest(ctx, "POST", "/v1/concepts/batch-validate", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CheckValueSetMembership checks if a code is in a value set
func (c *KB7TerminologyClient) CheckValueSetMembership(ctx context.Context, req *ValueSetMembershipRequest) (*ValueSetMembershipResponse, error) {
	if !c.config.Enabled {
		return &ValueSetMembershipResponse{
			Success:      true,
			IsMember:    true,
			ValueSetID:  req.ValueSetID,
			Code:        req.Code,
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *ValueSetMembershipResponse
	// KB-7 uses /v1/valuesets/:url/validate-code or /fhir/ValueSet/$validate-code
	err = c.doRequest(ctx, "POST", "/fhir/ValueSet/$validate-code", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// TranslateCode translates a code from one system to another
func (c *KB7TerminologyClient) TranslateCode(ctx context.Context, req *CodeTranslationRequest) (*CodeTranslationResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-7 client disabled, returning empty translation")
		return &CodeTranslationResponse{
			Success:      true,
			SourceCode:   req.SourceCode,
			SourceSystem: req.SourceSystem,
			TargetSystem: req.TargetSystem,
			Translations: []CodeRef{},
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *CodeTranslationResponse
	// KB-7 uses /v1/translate endpoint
	err = c.doRequest(ctx, "POST", "/v1/translate", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetValueSet retrieves a value set by ID
func (c *KB7TerminologyClient) GetValueSet(ctx context.Context, valueSetID string) (*ValueSet, error) {
	if !c.config.Enabled {
		return &ValueSet{
			ValueSetID: valueSetID,
			Name:       fmt.Sprintf("ValueSet %s", valueSetID),
			Status:     "active",
			Codes:      []CodeRef{},
		}, nil
	}

	var resp *ValueSet
	// KB-7 uses /v1/valuesets/:url endpoint (or /fhir/ValueSet/:id)
	endpoint := fmt.Sprintf("/v1/valuesets/%s", valueSetID)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// SearchCodes searches for codes by text
func (c *KB7TerminologyClient) SearchCodes(ctx context.Context, query string, system string, limit int) ([]CodeRef, error) {
	if !c.config.Enabled {
		return []CodeRef{}, nil
	}

	if limit <= 0 {
		limit = 10
	}

	var resp struct {
		Results []CodeRef `json:"results"`
	}
	// KB-7 uses /v1/concepts for search with query parameters
	endpoint := fmt.Sprintf("/v1/concepts?q=%s&system=%s&limit=%d", query, system, limit)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Results, nil
}

// GetCodeHierarchy retrieves the hierarchy for a code (parents and children)
func (c *KB7TerminologyClient) GetCodeHierarchy(ctx context.Context, code string, system string) (*CodeLookupResponse, error) {
	if !c.config.Enabled {
		return &CodeLookupResponse{
			Success:     true,
			Code:        code,
			System:      system,
			Display:     fmt.Sprintf("Code %s", code),
			ParentCodes: []CodeRef{},
			ChildCodes:  []CodeRef{},
			Active:      true,
		}, nil
	}

	var resp *CodeLookupResponse
	// KB-7 uses /v1/concepts/:system/:code endpoint - hierarchy returned with concept lookup
	endpoint := fmt.Sprintf("/v1/concepts/%s/%s", system, code)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// IsCodeActive checks if a code is currently active
func (c *KB7TerminologyClient) IsCodeActive(ctx context.Context, code string, system string) (bool, error) {
	resp, err := c.LookupCode(ctx, code, system)
	if err != nil {
		return false, err
	}
	return resp.Active, nil
}

// doRequest performs an HTTP request with retry logic
func (c *KB7TerminologyClient) doRequest(ctx context.Context, method, endpoint string, body []byte, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			waitTime := c.config.RetryWaitMin * time.Duration(1<<uint(attempt-1))
			if waitTime > c.config.RetryWaitMax {
				waitTime = c.config.RetryWaitMax
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

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
		req.Header.Set("X-Client-Service", "kb-12-ordersets")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.log.WithError(err).WithField("attempt", attempt+1).Warn("KB-7 request failed, retrying")
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("KB-7 server error: %d - %s", resp.StatusCode, string(respBody))
			c.log.WithField("status", resp.StatusCode).WithField("attempt", attempt+1).Warn("KB-7 server error, retrying")
			continue
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("KB-7 client error: %d - %s", resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("KB-7 request failed after %d retries: %w", c.config.MaxRetries+1, lastErr)
}
