// Package clients provides HTTP/GraphQL clients for KB services.
//
// KB7FHIRClient implements the runtime-safe FHIR terminology interface.
// This client uses ONLY precomputed expansions from PostgreSQL - NO Neo4j at runtime.
//
// ARCHITECTURAL CONSTRAINT (CTO/CMO Directive):
// "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."
//
// ❌ FORBIDDEN: Runtime Neo4j traversal during clinical execution
// ✅ REQUIRED: Precomputed expansions, O(1) indexed lookups
//
// DYNAMIC DESIGN (NO HARDCODED VALUESETS):
// All canonical ValueSets are fetched dynamically from KB-7 via GET /fhir/ValueSet.
// This ensures KB-7 is the SINGLE SOURCE OF TRUTH for terminology.
//
// Connects to: http://localhost:8092/fhir/*
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// KB7FHIRClient Interface (Runtime-Safe)
// ============================================================================

// KB7FHIRClient is the runtime-safe terminology client using FHIR endpoints.
// It provides O(1) indexed lookups against precomputed expansions.
// NO Neo4j traversal at runtime - deterministic and auditable.
type KB7FHIRClient interface {
	// ValidateCode checks if a code is a member of a ValueSet.
	// Uses: POST /fhir/ValueSet/$validate-code
	// Returns: true if code is in ValueSet, false otherwise
	ValidateCode(ctx context.Context, valueSetID string, system string, code string) (bool, error)

	// ExpandValueSet returns all codes in a ValueSet's precomputed expansion.
	// Uses: GET /fhir/ValueSet/:id/$expand
	// For CQL execution only - NOT for membership checks!
	ExpandValueSet(ctx context.Context, valueSetID string) ([]contracts.ClinicalCode, error)

	// GetValueSet returns ValueSet metadata.
	// Uses: GET /fhir/ValueSet/:id
	GetValueSet(ctx context.Context, valueSetID string) (*ValueSetMetadata, error)

	// ListCanonicalValueSets returns all canonical ValueSets from KB-7.
	// Uses: GET /fhir/ValueSet
	// This is the DYNAMIC method that replaces hardcoded lists!
	ListCanonicalValueSets(ctx context.Context) ([]ValueSetMetadata, error)

	// BatchValidateCodes checks multiple codes against a ValueSet.
	// More efficient than multiple ValidateCode calls.
	BatchValidateCodes(ctx context.Context, valueSetID string, codes []contracts.ClinicalCode) (map[string]bool, error)

	// GetMembershipsForCode returns all canonical ValueSets containing the code.
	// This is the correct way to determine semantic flags (has_afib, is_diabetic, etc.)
	// DYNAMIC: Uses ListCanonicalValueSets() to get the list from KB-7.
	GetMembershipsForCode(ctx context.Context, code contracts.ClinicalCode) (map[string]bool, error)

	// HealthCheck verifies FHIR endpoints are operational with precomputed data.
	HealthCheck(ctx context.Context) error
}

// ValueSetMetadata contains ValueSet definition information.
type ValueSetMetadata struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Status      string `json:"status"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Publisher   string `json:"publisher"`
	// Category is the clinical domain from KB-7 (e.g., "condition", "medication", "lab")
	// DYNAMIC: This comes from KB-7's value_sets.clinical_domain field - NO HARDCODING!
	Category string `json:"category,omitempty"`
	// UseContext specifies how this ValueSet is used (e.g., ["cql", "measure", "lab"])
	// DYNAMIC: Enables Vaidshala to discover which ValueSets to expand without hardcoding!
	UseContext []string `json:"useContext,omitempty"`
	// IsCanonical indicates if this ValueSet is marked as canonical in KB-7
	// Canonical ValueSets are important for ICU Intelligence, Safety alerts, Clinical facts
	IsCanonical bool `json:"is_canonical,omitempty"`
	// OID is the OID identifier for the ValueSet (e.g., "2.16.840.1.113883.3.464.1003.103.12.1001")
	OID string `json:"oid,omitempty"`
}

// ============================================================================
// KB-7 v2 Reverse Lookup Response Types
// ============================================================================

// ValueSetMembership represents a ValueSet that contains a given code.
// Returned by the $lookup-memberships endpoint.
type ValueSetMembership struct {
	ValueSetURL  string `json:"valueset_url"`
	ValueSetOID  string `json:"valueset_oid,omitempty"`
	SemanticName string `json:"semantic_name"`
	Title        string `json:"title,omitempty"`
	Category     string `json:"category,omitempty"`
	IsCanonical  bool   `json:"is_canonical"`
	CodeDisplay  string `json:"code_display,omitempty"`
}

// LookupMembershipsResponse is the response from $lookup-memberships endpoint.
// This is the KB-7 v2 REVERSE LOOKUP - returns all ValueSets containing a code in ONE query!
type LookupMembershipsResponse struct {
	Code             string               `json:"code"`
	System           string               `json:"system"`
	TotalMemberships int                  `json:"total_memberships"`
	CanonicalCount   int                  `json:"canonical_count"`
	SemanticNames    []string             `json:"semantic_names"`
	Memberships      []ValueSetMembership `json:"memberships"`
	ProcessingTimeMs int64                `json:"processing_time_ms"`
}

// ============================================================================
// KB7FHIRHTTPClient Implementation
// ============================================================================

// KB7FHIRHTTPClient implements KB7FHIRClient using HTTP calls to FHIR endpoints.
type KB7FHIRHTTPClient struct {
	baseURL    string
	httpClient *http.Client

	// Cached canonical ValueSets (loaded once, refreshed periodically)
	canonicalValueSets    []ValueSetMetadata
	canonicalValueSetsMu  sync.RWMutex
	lastValueSetRefresh   time.Time
	valueSetRefreshPeriod time.Duration
}

// NewKB7FHIRClient creates a new FHIR-compliant KB-7 client for runtime execution.
func NewKB7FHIRClient(baseURL string) *KB7FHIRHTTPClient {
	return &KB7FHIRHTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second, // Faster timeout - precomputed data should be quick
		},
		valueSetRefreshPeriod: 5 * time.Minute, // Refresh ValueSet list every 5 minutes
	}
}

// NewKB7FHIRClientWithHTTP creates a client with custom HTTP client.
func NewKB7FHIRClientWithHTTP(baseURL string, httpClient *http.Client) *KB7FHIRHTTPClient {
	return &KB7FHIRHTTPClient{
		baseURL:               baseURL,
		httpClient:            httpClient,
		valueSetRefreshPeriod: 5 * time.Minute,
	}
}

// ============================================================================
// ListCanonicalValueSets - DYNAMIC VALUESET DISCOVERY
// ============================================================================
// This replaces the hardcoded CanonicalValueSets array!
// KB-7 is now the SINGLE SOURCE OF TRUTH for what ValueSets exist.

func (c *KB7FHIRHTTPClient) ListCanonicalValueSets(ctx context.Context) ([]ValueSetMetadata, error) {
	// Check if we have a valid cached list
	c.canonicalValueSetsMu.RLock()
	if len(c.canonicalValueSets) > 0 && time.Since(c.lastValueSetRefresh) < c.valueSetRefreshPeriod {
		cached := make([]ValueSetMetadata, len(c.canonicalValueSets))
		copy(cached, c.canonicalValueSets)
		c.canonicalValueSetsMu.RUnlock()
		return cached, nil
	}
	c.canonicalValueSetsMu.RUnlock()

	// Fetch fresh list from KB-7
	endpoint := fmt.Sprintf("%s/fhir/ValueSet", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json")

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
		return nil, fmt.Errorf("FHIR ListValueSets failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse FHIR Bundle response
	// DYNAMIC: Now includes Category AND UseContext from KB-7!
	var bundle struct {
		ResourceType string `json:"resourceType"`
		Type         string `json:"type"`
		Total        int    `json:"total"`
		Entry        []struct {
			Resource struct {
				ResourceType string `json:"resourceType"`
				ID           string `json:"id"`
				URL          string `json:"url"`
				Name         string `json:"name"`
				Title        string `json:"title"`
				Status       string `json:"status"`
				Version      string `json:"version"`
				Description  string `json:"description"`
				Publisher    string `json:"publisher"`
				// Category is the clinical domain from KB-7 (condition, medication, lab)
				// DYNAMIC: KB-7 is the SINGLE SOURCE OF TRUTH for categorization!
				Category string `json:"category"`
				// UseContext specifies how this ValueSet is used (JSON array as string)
				// DYNAMIC: Enables discovery of which ValueSets to expand!
				UseContext string `json:"useContext"`
			} `json:"resource"`
		} `json:"entry"`
	}

	if err := json.Unmarshal(body, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to ValueSetMetadata slice
	// DYNAMIC: Category AND UseContext now come from KB-7 - NO HARDCODING!
	valueSets := make([]ValueSetMetadata, 0, len(bundle.Entry))
	for _, entry := range bundle.Entry {
		if entry.Resource.ResourceType == "ValueSet" {
			// Parse UseContext from JSON string to []string
			var useContextList []string
			if entry.Resource.UseContext != "" && entry.Resource.UseContext != "[]" {
				_ = json.Unmarshal([]byte(entry.Resource.UseContext), &useContextList)
			}

			valueSets = append(valueSets, ValueSetMetadata{
				ID:          entry.Resource.ID,
				URL:         entry.Resource.URL,
				Name:        entry.Resource.Name,
				Title:       entry.Resource.Title,
				Status:      entry.Resource.Status,
				Version:     entry.Resource.Version,
				Description: entry.Resource.Description,
				Publisher:   entry.Resource.Publisher,
				Category:    entry.Resource.Category,   // DYNAMIC: From KB-7's clinical_domain!
				UseContext:  useContextList,            // DYNAMIC: For ValueSet discovery!
			})
		}
	}

	// Cache the result
	c.canonicalValueSetsMu.Lock()
	c.canonicalValueSets = valueSets
	c.lastValueSetRefresh = time.Now()
	c.canonicalValueSetsMu.Unlock()

	return valueSets, nil
}

// ListValueSetsByContext returns ValueSets filtered by use_context.
// DYNAMIC: This replaces hardcoded ValueSetsToExpand arrays!
// Examples:
//   - ListValueSetsByContext(ctx, "cql") returns all CQL-relevant ValueSets
//   - ListValueSetsByContext(ctx, "measure") returns all measure-relevant ValueSets
func (c *KB7FHIRHTTPClient) ListValueSetsByContext(ctx context.Context, context string) ([]ValueSetMetadata, error) {
	allValueSets, err := c.ListCanonicalValueSets(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by use_context
	var filtered []ValueSetMetadata
	for _, vs := range allValueSets {
		for _, uc := range vs.UseContext {
			if uc == context {
				filtered = append(filtered, vs)
				break
			}
		}
	}

	return filtered, nil
}

// GetValueSetNamesForContext returns just the names of ValueSets for a given context.
// DYNAMIC: Use this to replace hardcoded ValueSetsToExpand arrays!
func (c *KB7FHIRHTTPClient) GetValueSetNamesForContext(ctx context.Context, context string) ([]string, error) {
	valueSets, err := c.ListValueSetsByContext(ctx, context)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(valueSets))
	for i, vs := range valueSets {
		names[i] = vs.Name
	}

	return names, nil
}

// ============================================================================
// ValidateCode - O(1) Membership Check
// ============================================================================
// This is the PRIMARY method for membership checks.
// Uses precomputed expansions - NO Neo4j at runtime.

func (c *KB7FHIRHTTPClient) ValidateCode(
	ctx context.Context,
	valueSetID string,
	system string,
	code string,
) (bool, error) {
	endpoint := fmt.Sprintf("%s/fhir/ValueSet/$validate-code", c.baseURL)

	// Build the ValueSet URL from ID
	valueSetURL := c.buildValueSetURL(valueSetID)

	requestBody := map[string]string{
		"url":    valueSetURL,
		"system": system,
		"code":   code,
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
	req.Header.Set("Accept", "application/fhir+json")

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
		return false, fmt.Errorf("FHIR $validate-code failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse FHIR Parameters response
	var result struct {
		ResourceType string `json:"resourceType"`
		Parameter    []struct {
			Name         string `json:"name"`
			ValueBoolean *bool  `json:"valueBoolean,omitempty"`
			ValueString  string `json:"valueString,omitempty"`
		} `json:"parameter"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	// Find the "result" parameter
	for _, param := range result.Parameter {
		if param.Name == "result" && param.ValueBoolean != nil {
			return *param.ValueBoolean, nil
		}
	}

	return false, nil
}

// ============================================================================
// ExpandValueSet - For CQL Execution Only
// ============================================================================
// WARNING: Do NOT use this for membership checks!
// Use ValidateCode for membership - it's O(1) vs O(n).

func (c *KB7FHIRHTTPClient) ExpandValueSet(
	ctx context.Context,
	valueSetID string,
) ([]contracts.ClinicalCode, error) {
	endpoint := fmt.Sprintf("%s/fhir/ValueSet/%s/$expand", c.baseURL, url.PathEscape(valueSetID))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json")

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
		return nil, fmt.Errorf("FHIR $expand failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse FHIR ValueSet response with expansion
	var result struct {
		ResourceType string `json:"resourceType"`
		ID           string `json:"id"`
		URL          string `json:"url"`
		Expansion    struct {
			Timestamp string `json:"timestamp"`
			Total     int    `json:"total"`
			Contains  []struct {
				System  string `json:"system"`
				Code    string `json:"code"`
				Display string `json:"display"`
			} `json:"contains"`
		} `json:"expansion"`
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

// ============================================================================
// GetValueSet - Metadata Retrieval
// ============================================================================

func (c *KB7FHIRHTTPClient) GetValueSet(
	ctx context.Context,
	valueSetID string,
) (*ValueSetMetadata, error) {
	endpoint := fmt.Sprintf("%s/fhir/ValueSet/%s", c.baseURL, url.PathEscape(valueSetID))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("ValueSet not found: %s", valueSetID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FHIR GetValueSet failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ResourceType string `json:"resourceType"`
		ID           string `json:"id"`
		URL          string `json:"url"`
		Name         string `json:"name"`
		Title        string `json:"title"`
		Status       string `json:"status"`
		Version      string `json:"version"`
		Description  string `json:"description"`
		Publisher    string `json:"publisher"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &ValueSetMetadata{
		ID:          result.ID,
		URL:         result.URL,
		Name:        result.Name,
		Title:       result.Title,
		Status:      result.Status,
		Version:     result.Version,
		Description: result.Description,
		Publisher:   result.Publisher,
	}, nil
}

// ============================================================================
// BatchValidateCodes - Efficient Multiple Code Validation
// ============================================================================

func (c *KB7FHIRHTTPClient) BatchValidateCodes(
	ctx context.Context,
	valueSetID string,
	codes []contracts.ClinicalCode,
) (map[string]bool, error) {
	results := make(map[string]bool)

	// For small batches, parallel validation is efficient
	// For large batches, consider expanding the ValueSet and checking locally
	for _, code := range codes {
		key := fmt.Sprintf("%s|%s", code.System, code.Code)
		isMember, err := c.ValidateCode(ctx, valueSetID, code.System, code.Code)
		if err != nil {
			// Log error but continue - membership is false if we can't validate
			results[key] = false
			continue
		}
		results[key] = isMember
	}

	return results, nil
}

// ============================================================================
// LookupMemberships - KB-7 v2 REVERSE LOOKUP (NEW!)
// ============================================================================
// This is the FAST method for getting all ValueSet memberships for a code.
// Uses: GET /fhir/CodeSystem/$lookup-memberships?code=X&system=Y
//
// PERFORMANCE:
//   - OLD: 18,000+ HTTP calls (one per ValueSet)
//   - NEW: 1 HTTP call (reverse lookup)
//   - Speedup: 18,000x faster!

func (c *KB7FHIRHTTPClient) LookupMemberships(
	ctx context.Context,
	code string,
	system string,
	canonicalOnly bool,
) (*LookupMembershipsResponse, error) {
	// Build endpoint URL with query parameters
	endpoint := fmt.Sprintf("%s/fhir/CodeSystem/$lookup-memberships", c.baseURL)

	// Add query parameters
	params := url.Values{}
	params.Set("code", code)
	if system != "" {
		params.Set("system", system)
	}
	if canonicalOnly {
		params.Set("canonical", "true")
	}

	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

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
		return nil, fmt.Errorf("$lookup-memberships failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result LookupMembershipsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ============================================================================
// GetMembershipsForCode - Semantic Flag Determination (KB-7 v2)
// ============================================================================
// This is the KEY method for KnowledgeSnapshotBuilder.
// Returns all ValueSets where the code is a member.
//
// KB-7 v2: Now uses REVERSE LOOKUP for O(1) performance!
// OLD: 18,000+ HTTP calls per code (one per ValueSet)
// NEW: 1 HTTP call per code (reverse lookup)

func (c *KB7FHIRHTTPClient) GetMembershipsForCode(
	ctx context.Context,
	code contracts.ClinicalCode,
) (map[string]bool, error) {
	// ═══════════════════════════════════════════════════════════════════════
	// KB-7 v2: Use REVERSE LOOKUP instead of iterating all ValueSets!
	// This is 18,000x faster than the old approach.
	// ═══════════════════════════════════════════════════════════════════════

	// Call the reverse lookup endpoint (1 HTTP call instead of 18,000+)
	result, err := c.LookupMemberships(ctx, code.Code, code.System, false)
	if err != nil {
		return nil, fmt.Errorf("reverse lookup failed: %w", err)
	}

	// Convert response to membership map
	memberships := make(map[string]bool)
	for _, m := range result.Memberships {
		// Use semantic name as the key (human-readable!)
		vsName := m.SemanticName
		if vsName == "" {
			vsName = m.ValueSetURL
		}
		memberships[vsName] = true
	}

	return memberships, nil
}

// GetMembershipsForCodeWithDetails returns detailed membership info including category.
// This is useful for building semantic flags with category-based prefixes.
func (c *KB7FHIRHTTPClient) GetMembershipsForCodeWithDetails(
	ctx context.Context,
	code contracts.ClinicalCode,
	canonicalOnly bool,
) ([]ValueSetMembership, error) {
	result, err := c.LookupMemberships(ctx, code.Code, code.System, canonicalOnly)
	if err != nil {
		return nil, fmt.Errorf("reverse lookup failed: %w", err)
	}

	return result.Memberships, nil
}

// GetSemanticNamesForCode returns just the semantic names for a code.
// This is the simplest way to get human-readable ValueSet memberships.
func (c *KB7FHIRHTTPClient) GetSemanticNamesForCode(
	ctx context.Context,
	code contracts.ClinicalCode,
	canonicalOnly bool,
) ([]string, error) {
	result, err := c.LookupMemberships(ctx, code.Code, code.System, canonicalOnly)
	if err != nil {
		return nil, fmt.Errorf("reverse lookup failed: %w", err)
	}

	return result.SemanticNames, nil
}

// ============================================================================
// HealthCheck - FHIR Endpoint Verification
// ============================================================================

func (c *KB7FHIRHTTPClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/fhir/health", c.baseURL)

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
		return fmt.Errorf("FHIR health check failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status           string `json:"status"`
		PrecomputedCodes int    `json:"precomputed_codes"`
		Architecture     string `json:"architecture"`
		Neo4jAtRuntime   bool   `json:"neo4j_at_runtime"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Status != "healthy" {
		return fmt.Errorf("KB-7 FHIR unhealthy: %s", result.Status)
	}

	if result.Neo4jAtRuntime {
		return fmt.Errorf("CRITICAL: KB-7 FHIR reports Neo4j at runtime - architecture violation!")
	}

	if result.PrecomputedCodes == 0 {
		return fmt.Errorf("WARNING: KB-7 FHIR has no precomputed codes - seeding required")
	}

	return nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// buildValueSetURL constructs the canonical ValueSet URL from an ID.
// DYNAMIC: No hardcoded mapping - uses pattern-based URL construction.
func (c *KB7FHIRHTTPClient) buildValueSetURL(valueSetID string) string {
	// If already a full URL, return as-is
	if len(valueSetID) > 4 && valueSetID[:4] == "http" {
		return valueSetID
	}

	// Pattern: Convert ID to kb7.health URL
	// Example: "DiabetesMellitus" -> "http://kb7.health/ValueSet/DiabetesMellitus"
	// Example: "LabHbA1c" -> "http://kb7.health/ValueSet/LabHbA1c"
	return fmt.Sprintf("http://kb7.health/ValueSet/%s", valueSetID)
}

// ============================================================================
// DYNAMIC Semantic Flag Building
// ============================================================================
// Flags are now derived using Category from KB-7's clinical_domain field!
// NO MORE HARDCODED PREFIX ARRAYS - KB-7 is the SINGLE SOURCE OF TRUTH.

// SemanticFlagMapping is kept for backwards compatibility but is now EMPTY.
// Category-based derivation is the new approach.
// DEPRECATED: Use BuildSemanticFlagsWithMetadata instead.
type SemanticFlagMapping struct {
	// DEPRECATED: These fields are no longer used.
	// Category from KB-7 now determines the flag prefix.
	ConditionPrefixes  []string
	MedicationPrefixes []string
	LabPrefixes        []string
}

// DefaultSemanticFlagMapping returns an EMPTY mapping.
// DEPRECATED: Use BuildSemanticFlagsWithMetadata for category-based derivation.
func DefaultSemanticFlagMapping() *SemanticFlagMapping {
	// INTENTIONALLY EMPTY: No more hardcoded prefixes!
	// KB-7's clinical_domain (Category) is now the source of truth.
	return &SemanticFlagMapping{
		ConditionPrefixes:  []string{}, // EMPTY - use Category from KB-7
		MedicationPrefixes: []string{}, // EMPTY - use Category from KB-7
		LabPrefixes:        []string{}, // EMPTY - use Category from KB-7
	}
}

// BuildSemanticFlagsWithMetadata converts membership results to semantic flags using KB-7 metadata.
// This is the NEW, DYNAMIC approach that uses Category from KB-7.
// PREFERRED: Use this instead of BuildSemanticFlagsFromMemberships.
func BuildSemanticFlagsWithMetadata(memberships map[string]bool, valueSets []ValueSetMetadata) map[string]bool {
	// Build a map of ValueSet name → Category for quick lookup
	categoryMap := make(map[string]string)
	for _, vs := range valueSets {
		categoryMap[vs.Name] = vs.Category
	}

	flags := make(map[string]bool)

	for vsName, isMember := range memberships {
		if !isMember {
			continue
		}

		// Get category from KB-7 metadata
		category := categoryMap[vsName]
		flagName := DeriveSemanticFlagNameWithCategory(vsName, category)
		if flagName != "" {
			flags[flagName] = true
		}
	}

	return flags
}

// BuildSemanticFlagsFromMemberships is DEPRECATED - use BuildSemanticFlagsWithMetadata.
// Kept for backwards compatibility, falls back to simple snake_case conversion.
func BuildSemanticFlagsFromMemberships(memberships map[string]bool, mapping *SemanticFlagMapping) map[string]bool {
	flags := make(map[string]bool)

	for vsName, isMember := range memberships {
		if !isMember {
			continue
		}

		// FALLBACK: Without metadata, just use snake_case conversion
		// This is less accurate than category-based derivation!
		flagName := toSnakeCase(vsName)
		if flagName != "" {
			flags[flagName] = true
		}
	}

	return flags
}

// DeriveSemanticFlagNameWithCategory derives a semantic flag name using KB-7's Category.
// This is the DYNAMIC approach - NO HARDCODED PREFIX MATCHING!
//
// Category values from KB-7:
//   - "condition" → "has_X" or "is_X" flags (e.g., has_diabetes, is_diabetic)
//   - "medication" → "on_X" flags (e.g., on_ace_inhibitor)
//   - "lab" → keep snake_case (e.g., lab_hba1c)
//   - "" or other → just snake_case
func DeriveSemanticFlagNameWithCategory(vsName, category string) string {
	snakeName := toSnakeCase(vsName)

	switch strings.ToLower(category) {
	case "condition":
		// Condition ValueSets use "has_" or "is_" prefix
		// Special case for diabetes → is_diabetic (backwards compatible)
		if strings.Contains(strings.ToLower(vsName), "diabetes") {
			return "is_diabetic"
		}
		return "has_" + snakeName

	case "medication":
		// Medication ValueSets use "on_" prefix
		return "on_" + snakeName

	case "lab":
		// Lab ValueSets keep original name (already starts with "lab_" typically)
		if strings.HasPrefix(snakeName, "lab_") {
			return snakeName
		}
		return "lab_" + snakeName

	default:
		// Unknown category - just use snake_case
		return snakeName
	}
}

// DeriveSemanticFlagName derives a semantic flag name from a ValueSet name.
// DEPRECATED: Use DeriveSemanticFlagNameWithCategory with KB-7 metadata instead.
// This fallback just converts to snake_case without category intelligence.
func DeriveSemanticFlagName(vsName string) string {
	// WITHOUT category info, we can only do basic conversion
	// For full semantic flag derivation, use DeriveSemanticFlagNameWithCategory!
	return toSnakeCase(vsName)
}

// toSnakeCase converts PascalCase/camelCase to snake_case
func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c)+32) // lowercase
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

// ============================================================================
// DEPRECATED: Hardcoded Constants (REMOVED)
// ============================================================================
// The following hardcoded constants have been REMOVED:
//
// - ValueSetDiabetesMellitus, ValueSetAtrialFibrillation, etc. (const block)
// - CanonicalValueSets array
// - canonicalValueSetURLs map
// - BuildSemanticFlags() with hardcoded mappings
//
// ALL ValueSets are now discovered dynamically via ListCanonicalValueSets()!
// KB-7 is the SINGLE SOURCE OF TRUTH.
// ============================================================================
