// Package clients provides HTTP/GraphQL clients for KB services.
//
// KB7HTTPClient implements the KB7Client interface for KB-7 Terminology Service.
// It provides ValueSet expansion, code membership checks, and code resolution.
//
// Connects to: http://localhost:8087
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// KB7HTTPClient implements KB7Client by calling the KB-7 Terminology Service REST API.
type KB7HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB7HTTPClient creates a new KB-7 HTTP client.
func NewKB7HTTPClient(baseURL string) *KB7HTTPClient {
	return &KB7HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB7HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB7HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB7HTTPClient {
	return &KB7HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB7Client Interface Implementation
// ============================================================================

// ExpandValueSet returns all codes in a ValueSet.
// Calls: POST /v1/rules/valuesets/:identifier/expand
func (c *KB7HTTPClient) ExpandValueSet(
	ctx context.Context,
	valueSetName string,
) ([]contracts.ClinicalCode, error) {

	endpoint := fmt.Sprintf("%s/v1/rules/valuesets/%s/expand", c.baseURL, url.PathEscape(valueSetName))

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-7 expand failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse KB-7 expansion response
	var result struct {
		Expansion struct {
			Contains []struct {
				System  string `json:"system"`
				Code    string `json:"code"`
				Display string `json:"display"`
			} `json:"contains"`
			Total int `json:"total"`
		} `json:"expansion"`
		Identifier string `json:"identifier"`
		Name       string `json:"name"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to ClinicalCode slice
	codes := make([]contracts.ClinicalCode, 0, len(result.Expansion.Contains))
	for _, c := range result.Expansion.Contains {
		codes = append(codes, contracts.ClinicalCode{
			System:  c.System,
			Code:    c.Code,
			Display: c.Display,
		})
	}

	return codes, nil
}

// CheckMembership checks if a code is in specified ValueSets.
// Calls: POST /v1/rules/classify (reverse lookup)
// Returns the list of ValueSet names where the code is a member.
func (c *KB7HTTPClient) CheckMembership(
	ctx context.Context,
	code contracts.ClinicalCode,
	valueSetNames []string,
) ([]string, error) {

	// Use the classify endpoint which finds ALL value sets for a code
	endpoint := fmt.Sprintf("%s/v1/rules/classify", c.baseURL)

	requestBody := map[string]interface{}{
		"code":   code.Code,
		"system": code.System,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-7 classify failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse classify response
	var result struct {
		Code            string `json:"code"`
		System          string `json:"system"`
		MatchingValueSets []struct {
			Identifier string `json:"identifier"`
			Name       string `json:"name"`
			MatchType  string `json:"match_type"` // "exact" or "subsumption"
		} `json:"matching_valuesets"`
		TotalMatches int `json:"total_matches"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Filter to only requested ValueSets (if specified)
	requestedSet := make(map[string]bool)
	for _, vs := range valueSetNames {
		requestedSet[vs] = true
	}

	memberships := make([]string, 0)
	for _, vs := range result.MatchingValueSets {
		// If no filter specified, return all matches
		if len(valueSetNames) == 0 {
			memberships = append(memberships, vs.Name)
			continue
		}
		// Otherwise filter to requested ValueSets
		if requestedSet[vs.Name] || requestedSet[vs.Identifier] {
			memberships = append(memberships, vs.Name)
		}
	}

	return memberships, nil
}

// GetRelevantValueSets returns ValueSets relevant to patient conditions.
// Uses the classify endpoint to find ValueSets for each condition code.
func (c *KB7HTTPClient) GetRelevantValueSets(
	ctx context.Context,
	conditionCodes []contracts.ClinicalCode,
) ([]string, error) {

	// Collect all relevant ValueSets across all condition codes
	valueSetMap := make(map[string]bool)

	for _, code := range conditionCodes {
		memberships, err := c.CheckMembership(ctx, code, nil)
		if err != nil {
			// Log but continue with other codes
			continue
		}
		for _, vs := range memberships {
			valueSetMap[vs] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(valueSetMap))
	for vs := range valueSetMap {
		result = append(result, vs)
	}

	return result, nil
}

// ResolveCode gets display name for a code.
// Calls: GET /v1/concepts/:system/:code
func (c *KB7HTTPClient) ResolveCode(
	ctx context.Context,
	code contracts.ClinicalCode,
) (string, error) {

	// URL encode the system (e.g., "http://snomed.info/sct" -> encoded)
	encodedSystem := url.PathEscape(code.System)
	endpoint := fmt.Sprintf("%s/v1/concepts/%s/%s", c.baseURL, encodedSystem, code.Code)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		// Code not found - return empty display
		return "", nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("KB-7 concept lookup failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse concept lookup response
	var result struct {
		Code       string `json:"code"`
		Display    string `json:"display"`
		System     string `json:"system"`
		Definition string `json:"definition"`
		Active     bool   `json:"active"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Display, nil
}

// ============================================================================
// Additional Helper Methods
// ============================================================================

// ValidateCodeInValueSet checks if a specific code is valid in a ValueSet.
// Calls: POST /v1/rules/valuesets/:identifier/validate
func (c *KB7HTTPClient) ValidateCodeInValueSet(
	ctx context.Context,
	code contracts.ClinicalCode,
	valueSetIdentifier string,
) (bool, error) {

	endpoint := fmt.Sprintf("%s/v1/rules/valuesets/%s/validate", c.baseURL, url.PathEscape(valueSetIdentifier))

	requestBody := map[string]interface{}{
		"code":   code.Code,
		"system": code.System,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("KB-7 validate failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse validation response
	var result struct {
		Valid     bool   `json:"valid"`
		MatchType string `json:"match_type"`
		Message   string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Valid, nil
}

// HealthCheck verifies KB-7 service is healthy.
func (c *KB7HTTPClient) HealthCheck(ctx context.Context) error {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-7 unhealthy (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "healthy" {
		return fmt.Errorf("KB-7 unhealthy: %s", result.Status)
	}

	return nil
}

// ListValueSets retrieves available ValueSets from KB-7.
// Calls: GET /v1/rules/valuesets
func (c *KB7HTTPClient) ListValueSets(ctx context.Context) ([]string, error) {
	endpoint := fmt.Sprintf("%s/v1/rules/valuesets", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-7 list valuesets failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ValueSets []struct {
			Identifier string `json:"identifier"`
			Name       string `json:"name"`
			Status     string `json:"status"`
		} `json:"value_sets"`
		Count int `json:"count"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	names := make([]string, 0, len(result.ValueSets))
	for _, vs := range result.ValueSets {
		names = append(names, vs.Name)
	}

	return names, nil
}

// GetSubsumptionAncestors retrieves ancestors for a SNOMED code.
// Calls: POST /v1/subsumption/ancestors
func (c *KB7HTTPClient) GetSubsumptionAncestors(
	ctx context.Context,
	code contracts.ClinicalCode,
	maxDepth int,
) ([]contracts.ClinicalCode, error) {

	endpoint := fmt.Sprintf("%s/v1/subsumption/ancestors", c.baseURL)

	requestBody := map[string]interface{}{
		"code":      code.Code,
		"system":    code.System,
		"max_depth": maxDepth,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-7 ancestors failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Ancestors []struct {
			Code    string `json:"code"`
			Display string `json:"display"`
			Depth   int    `json:"depth"`
		} `json:"ancestors"`
		TotalCount int `json:"total_count"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	ancestors := make([]contracts.ClinicalCode, 0, len(result.Ancestors))
	for _, a := range result.Ancestors {
		ancestors = append(ancestors, contracts.ClinicalCode{
			System:  code.System, // Same system as input
			Code:    a.Code,
			Display: a.Display,
		})
	}

	return ancestors, nil
}
