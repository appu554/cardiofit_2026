// Package integrations provides HTTP clients for external KB service integrations.
//
// All KB services are assumed to be running in Docker containers and accessible
// via their configured URLs.
package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB7Client provides integration with KB-7 Terminology Service.
//
// KB-7 is responsible for:
//   - Value set resolution for diagnosis/procedure codes
//   - SNOMED CT, ICD-10, CPT code lookups
//   - Code system expansion
type KB7Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB7Client creates a new KB-7 Terminology client.
func NewKB7Client(baseURL string, logger *zap.Logger) *KB7Client {
	return &KB7Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ValueSet represents a value set from KB-7.
type ValueSet struct {
	OID         string          `json:"oid"`
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Status      string          `json:"status"`
	CodeCount   int             `json:"code_count"`
	CodeSystem  string          `json:"code_system"`
	Codes       []ValueSetCode  `json:"codes,omitempty"`
}

// ValueSetCode represents a single code in a value set.
type ValueSetCode struct {
	Code        string `json:"code"`
	Display     string `json:"display"`
	System      string `json:"system"`
}

// CodeLookupResult represents a terminology code lookup result.
type CodeLookupResult struct {
	Code        string   `json:"code"`
	Display     string   `json:"display"`
	System      string   `json:"system"`
	Found       bool     `json:"found"`
	Synonyms    []string `json:"synonyms,omitempty"`
	ParentCodes []string `json:"parent_codes,omitempty"`
}

// GetValueSet retrieves a value set by OID from KB-7.
func (c *KB7Client) GetValueSet(ctx context.Context, oid string) (*ValueSet, error) {
	url := fmt.Sprintf("%s/v1/valuesets/%s", c.baseURL, oid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-7 request failed",
			zap.String("url", url),
			zap.Error(err),
		)
		return nil, fmt.Errorf("KB-7 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("value set not found: %s", oid)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-7 returned status %d", resp.StatusCode)
	}

	var vs ValueSet
	if err := json.NewDecoder(resp.Body).Decode(&vs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Debug("Retrieved value set from KB-7",
		zap.String("oid", oid),
		zap.String("name", vs.Name),
		zap.Int("code_count", vs.CodeCount),
	)

	return &vs, nil
}

// ExpandValueSet retrieves all codes in a value set.
func (c *KB7Client) ExpandValueSet(ctx context.Context, oid string) ([]ValueSetCode, error) {
	url := fmt.Sprintf("%s/v1/valuesets/%s/expand", c.baseURL, oid)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KB-7 expand request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-7 returned status %d", resp.StatusCode)
	}

	var result struct {
		Codes []ValueSetCode `json:"codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Codes, nil
}

// LookupCode looks up a code in KB-7 terminology service.
func (c *KB7Client) LookupCode(ctx context.Context, system, code string) (*CodeLookupResult, error) {
	url := fmt.Sprintf("%s/v1/codes/lookup?system=%s&code=%s", c.baseURL, system, code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("KB-7 lookup request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &CodeLookupResult{
			Code:   code,
			System: system,
			Found:  false,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-7 returned status %d", resp.StatusCode)
	}

	var result CodeLookupResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result.Found = true
	return &result, nil
}

// ValidateMemberInValueSet checks if a code is a member of a value set.
func (c *KB7Client) ValidateMemberInValueSet(ctx context.Context, oid, system, code string) (bool, error) {
	url := fmt.Sprintf("%s/v1/valuesets/%s/validate-code?system=%s&code=%s",
		c.baseURL, oid, system, code)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("KB-7 validate request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("KB-7 returned status %d", resp.StatusCode)
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Valid, nil
}

// HealthCheck verifies KB-7 is accessible.
func (c *KB7Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-7 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-7 health check returned status %d", resp.StatusCode)
	}

	return nil
}

// GetBaseURL returns the configured base URL.
func (c *KB7Client) GetBaseURL() string {
	return c.baseURL
}
