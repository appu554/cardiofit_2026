// Package clients provides HTTP clients for external service integration.
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// KB7Client provides read-only access to KB-7 Terminology Service.
// KB-7 provides SNOMED CT, ICD-10, LOINC, and RxNorm code lookups.
// IMPORTANT: This client is READ-ONLY - terminology data flows only inward.
type KB7Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
}

// ──────────────────────────────────────────────────────────────────────────────
// Response Types
// ──────────────────────────────────────────────────────────────────────────────

// CodeValidationResponse represents a code validation result from KB-7.
type CodeValidationResponse struct {
	Valid       bool   `json:"valid"`
	Code        string `json:"code"`
	System      string `json:"system"`
	Display     string `json:"display"`
	MatchType   string `json:"match_type,omitempty"`
	MatchedCode string `json:"matched_code,omitempty"`
	Message     string `json:"message,omitempty"`
}

// ConceptLookupResponse represents a concept lookup result.
type ConceptLookupResponse struct {
	Code           string            `json:"code"`
	System         string            `json:"system"`
	Display        string            `json:"display"`
	Definition     string            `json:"definition,omitempty"`
	Synonyms       []string          `json:"synonyms,omitempty"`
	ParentCodes    []string          `json:"parent_codes,omitempty"`
	ChildCodes     []string          `json:"child_codes,omitempty"`
	Properties     map[string]string `json:"properties,omitempty"`
	Active         bool              `json:"active"`
	EffectiveTime  string            `json:"effective_time,omitempty"`
}

// ValueSetMember represents a member in a value set.
type ValueSetMember struct {
	Code    string `json:"code"`
	System  string `json:"system"`
	Display string `json:"display"`
}

// ValueSetResponse represents a value set lookup result.
type ValueSetResponse struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Version     string           `json:"version"`
	Status      string           `json:"status"`
	Description string           `json:"description,omitempty"`
	Members     []ValueSetMember `json:"members"`
	MemberCount int              `json:"member_count"`
}

// HierarchyResponse represents a terminology hierarchy query result.
type HierarchyResponse struct {
	Code         string               `json:"code"`
	System       string               `json:"system"`
	Display      string               `json:"display"`
	Ancestors    []HierarchyNode      `json:"ancestors,omitempty"`
	Descendants  []HierarchyNode      `json:"descendants,omitempty"`
}

// HierarchyNode represents a node in a terminology hierarchy.
type HierarchyNode struct {
	Code    string `json:"code"`
	Display string `json:"display"`
	Level   int    `json:"level"`
}

// ──────────────────────────────────────────────────────────────────────────────
// Client Implementation
// ──────────────────────────────────────────────────────────────────────────────

// NewKB7Client creates a new KB-7 terminology client.
func NewKB7Client(baseURL string, logger *logrus.Entry) *KB7Client {
	return &KB7Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.WithField("client", "kb7"),
	}
}

// ValidateCode validates a single code against KB-7 terminology.
// Supports SNOMED CT, ICD-10, LOINC, RxNorm systems.
func (c *KB7Client) ValidateCode(ctx context.Context, code, system string) (*CodeValidationResponse, error) {
	// Map system URIs to KB-7 endpoints
	endpoint := c.getValidationEndpoint(system)
	if endpoint == "" {
		return &CodeValidationResponse{
			Valid:   false,
			Code:    code,
			System:  system,
			Message: "unsupported terminology system",
		}, nil
	}

	reqURL := fmt.Sprintf("%s/v1/%s/validate/%s", c.baseURL, endpoint, url.PathEscape(code))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &CodeValidationResponse{
			Valid:   false,
			Code:    code,
			System:  system,
			Message: "code not found",
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-7 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result CodeValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ValidateCodes validates multiple codes in a batch (READ-ONLY).
func (c *KB7Client) ValidateCodes(ctx context.Context, codes []string, system string) (map[string]*CodeValidationResponse, error) {
	results := make(map[string]*CodeValidationResponse)

	// For small batches, validate sequentially
	// TODO: Implement batch endpoint when KB-7 supports it
	for _, code := range codes {
		result, err := c.ValidateCode(ctx, code, system)
		if err != nil {
			c.logger.WithError(err).WithField("code", code).Warn("Failed to validate code")
			results[code] = &CodeValidationResponse{
				Valid:   false,
				Code:    code,
				System:  system,
				Message: err.Error(),
			}
			continue
		}
		results[code] = result
	}

	return results, nil
}

// LookupConcept retrieves full concept details from KB-7 (READ-ONLY).
func (c *KB7Client) LookupConcept(ctx context.Context, code, system string) (*ConceptLookupResponse, error) {
	endpoint := c.getValidationEndpoint(system)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported terminology system: %s", system)
	}

	reqURL := fmt.Sprintf("%s/v1/%s/concepts/%s", c.baseURL, endpoint, url.PathEscape(code))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-7 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var concept ConceptLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&concept); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &concept, nil
}

// GetValueSet retrieves a value set from KB-7 (READ-ONLY).
func (c *KB7Client) GetValueSet(ctx context.Context, valueSetID string) (*ValueSetResponse, error) {
	reqURL := fmt.Sprintf("%s/v1/valuesets/%s", c.baseURL, url.PathEscape(valueSetID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-7 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var valueSet ValueSetResponse
	if err := json.NewDecoder(resp.Body).Decode(&valueSet); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &valueSet, nil
}

// CheckValueSetMembership checks if a code is a member of a value set (READ-ONLY).
func (c *KB7Client) CheckValueSetMembership(ctx context.Context, code, valueSetID string) (bool, error) {
	reqURL := fmt.Sprintf("%s/v1/valuesets/%s/contains/%s", c.baseURL, url.PathEscape(valueSetID), url.PathEscape(code))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("KB-7 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		IsMember bool `json:"is_member"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.IsMember, nil
}

// GetHierarchy retrieves the hierarchy for a code (READ-ONLY).
func (c *KB7Client) GetHierarchy(ctx context.Context, code, system string, ancestorLevels, descendantLevels int) (*HierarchyResponse, error) {
	endpoint := c.getValidationEndpoint(system)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported terminology system: %s", system)
	}

	reqURL := fmt.Sprintf("%s/v1/%s/hierarchy/%s?ancestors=%d&descendants=%d",
		c.baseURL, endpoint, url.PathEscape(code), ancestorLevels, descendantLevels)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-7 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var hierarchy HierarchyResponse
	if err := json.NewDecoder(resp.Body).Decode(&hierarchy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &hierarchy, nil
}

// IsDescendantOf checks if a code is a descendant of another code (READ-ONLY).
// Useful for checking if a specific code falls under a general category.
func (c *KB7Client) IsDescendantOf(ctx context.Context, code, ancestorCode, system string) (bool, error) {
	endpoint := c.getValidationEndpoint(system)
	if endpoint == "" {
		return false, fmt.Errorf("unsupported terminology system: %s", system)
	}

	reqURL := fmt.Sprintf("%s/v1/%s/subsumes?child=%s&parent=%s",
		c.baseURL, endpoint, url.PathEscape(code), url.PathEscape(ancestorCode))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("KB-7 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		IsDescendant bool `json:"is_descendant"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.IsDescendant, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper Methods
// ──────────────────────────────────────────────────────────────────────────────

// getValidationEndpoint maps a terminology system URI to a KB-7 endpoint.
func (c *KB7Client) getValidationEndpoint(system string) string {
	systemLower := strings.ToLower(system)

	// Handle standard FHIR system URIs
	switch {
	case strings.Contains(systemLower, "snomed"):
		return "snomed"
	case strings.Contains(systemLower, "icd-10") || strings.Contains(systemLower, "icd10"):
		return "icd10"
	case strings.Contains(systemLower, "loinc"):
		return "loinc"
	case strings.Contains(systemLower, "rxnorm"):
		return "rxnorm"
	case systemLower == "snomed":
		return "snomed"
	case systemLower == "icd10" || systemLower == "icd-10":
		return "icd10"
	case systemLower == "loinc":
		return "loinc"
	case systemLower == "rxnorm":
		return "rxnorm"
	default:
		return ""
	}
}

// Health checks if KB-7 is accessible.
func (c *KB7Client) Health(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-7 unhealthy: status=%d", resp.StatusCode)
	}

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Risk Model Integration Helpers
// ──────────────────────────────────────────────────────────────────────────────

// ValidateRiskModelConditions validates all condition codes in a risk model.
// Used during model loading to ensure all ICD-10/SNOMED codes are valid.
func (c *KB7Client) ValidateRiskModelConditions(ctx context.Context, conditions []struct {
	Code   string
	System string
}) ([]string, error) {
	var invalidCodes []string

	for _, cond := range conditions {
		// Skip wildcard patterns (e.g., "I50.*")
		if strings.HasSuffix(cond.Code, "*") {
			continue
		}

		result, err := c.ValidateCode(ctx, cond.Code, cond.System)
		if err != nil {
			c.logger.WithError(err).WithField("code", cond.Code).Warn("Failed to validate condition code")
			continue
		}

		if !result.Valid {
			invalidCodes = append(invalidCodes, cond.Code)
		}
	}

	return invalidCodes, nil
}

// ExpandWildcardCode expands a wildcard pattern (e.g., "I50.*") to matching codes.
// Useful for risk model condition matching.
func (c *KB7Client) ExpandWildcardCode(ctx context.Context, pattern, system string) ([]string, error) {
	if !strings.HasSuffix(pattern, "*") {
		return []string{pattern}, nil
	}

	prefix := strings.TrimSuffix(pattern, "*")
	endpoint := c.getValidationEndpoint(system)
	if endpoint == "" {
		return nil, fmt.Errorf("unsupported terminology system: %s", system)
	}

	reqURL := fmt.Sprintf("%s/v1/%s/search?prefix=%s&limit=100", c.baseURL, endpoint, url.QueryEscape(prefix))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-7 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var result struct {
		Codes []string `json:"codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Codes, nil
}
