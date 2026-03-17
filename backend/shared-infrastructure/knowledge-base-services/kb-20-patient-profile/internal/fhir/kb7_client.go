package fhir

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/config"
	"kb-patient-profile/internal/models"
)

// KB7Concept represents a LOINC concept returned by KB-7's /v1/concepts/:system/:code endpoint.
type KB7Concept struct {
	Code    string `json:"code"`
	Display string `json:"display"`
	System  string `json:"system"`
}

// kb7ConceptResponse is the JSON envelope from /v1/concepts/LOINC/:code.
type kb7ConceptResponse struct {
	Concept struct {
		Code    string `json:"code"`
		Display string `json:"display"`
	} `json:"concept"`
}

// kb7MembershipResponse is the JSON from /fhir/CodeSystem/$lookup-memberships.
type kb7MembershipResponse struct {
	Code             string `json:"code"`
	System           string `json:"system"`
	TotalMemberships int    `json:"total_memberships"`
	Memberships      []struct {
		ValueSetURL  string `json:"valueset_url"`
		SemanticName string `json:"semantic_name"`
		Category     string `json:"category"`
		CodeDisplay  string `json:"code_display"`
	} `json:"memberships"`
}

// KB7Client queries the KB-7 Terminology Service (port 8092) for LOINC operations.
// KB-7 is the sole source of truth for LOINC codes — no hardcoded fallback.
// Results are cached in-memory (TTL 30m) since LOINC codes rarely change.
type KB7Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger

	// In-memory cache keyed by LOINC code
	cache    map[string]*KB7Concept
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
	cacheTS  map[string]time.Time

	// Reverse cache: LOINC code → KB-20 lab type (built from KB-7 lookups)
	reverseCache   map[string]string
	reverseCacheMu sync.RWMutex
}

// NewKB7Client creates a client for KB-7 Terminology Service lookups.
func NewKB7Client(cfg config.KB7Config, logger *zap.Logger) *KB7Client {
	return &KB7Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:       logger,
		cache:        make(map[string]*KB7Concept),
		cacheTTL:     30 * time.Minute,
		cacheTS:      make(map[string]time.Time),
		reverseCache: make(map[string]string),
	}
}

// LookupConcept retrieves a LOINC concept by code from KB-7.
// Uses: GET /v1/concepts/LOINC/:code
func (c *KB7Client) LookupConcept(loincCode string) (*KB7Concept, error) {
	// Check cache
	c.cacheMu.RLock()
	if concept, ok := c.cache[loincCode]; ok {
		if time.Since(c.cacheTS[loincCode]) < c.cacheTTL {
			c.cacheMu.RUnlock()
			return concept, nil
		}
	}
	c.cacheMu.RUnlock()

	reqURL := fmt.Sprintf("%s/v1/concepts/LOINC/%s", c.baseURL, loincCode)

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("KB-7 concept lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-7 returned %d for LOINC %s: %s", resp.StatusCode, loincCode, string(body))
	}

	var envResp kb7ConceptResponse
	if err := json.NewDecoder(resp.Body).Decode(&envResp); err != nil {
		return nil, fmt.Errorf("failed to decode KB-7 concept response: %w", err)
	}

	concept := &KB7Concept{
		Code:    envResp.Concept.Code,
		Display: envResp.Concept.Display,
		System:  "http://loinc.org",
	}

	// Cache
	c.cacheMu.Lock()
	c.cache[loincCode] = concept
	c.cacheTS[loincCode] = time.Now()
	c.cacheMu.Unlock()

	c.logger.Debug("KB-7 concept resolved",
		zap.String("code", loincCode),
		zap.String("display", concept.Display))

	return concept, nil
}

// ResolveLabType maps a LOINC code to a KB-20 lab type using KB-7's membership lookup.
// Uses: GET /fhir/CodeSystem/$lookup-memberships?code=:code&system=http://loinc.org
// Returns the KB-20 lab type constant (e.g., "CREATININE") or the code itself if no match.
func (c *KB7Client) ResolveLabType(loincCode string) string {
	// Check reverse cache
	c.reverseCacheMu.RLock()
	if labType, ok := c.reverseCache[loincCode]; ok {
		c.reverseCacheMu.RUnlock()
		return labType
	}
	c.reverseCacheMu.RUnlock()

	// First try concept lookup for the display name
	concept, err := c.LookupConcept(loincCode)
	if err != nil {
		c.logger.Warn("KB-7 concept lookup failed for reverse resolution",
			zap.String("loinc_code", loincCode),
			zap.Error(err))
		return loincCode
	}

	// Match display text to KB-20 lab type using clinical keywords
	labType := displayToLabType(concept.Display)
	if labType == "" {
		labType = loincCode // fallback: use LOINC code itself if no keyword match
	}

	// Cache the reverse mapping
	c.reverseCacheMu.Lock()
	c.reverseCache[loincCode] = labType
	c.reverseCacheMu.Unlock()

	c.logger.Info("KB-7 reverse LOINC resolved",
		zap.String("loinc_code", loincCode),
		zap.String("display", concept.Display),
		zap.String("lab_type", labType))

	return labType
}

// displayToLabType maps KB-7 concept display text to KB-20 lab type constants.
// Uses keyword matching against the LOINC display name returned by KB-7.
func displayToLabType(display string) string {
	lower := strings.ToLower(display)

	switch {
	case strings.Contains(lower, "creatinine") && !strings.Contains(lower, "albumin"):
		return models.LabTypeCreatinine
	case strings.Contains(lower, "glomerular filtration") || strings.Contains(lower, "egfr"):
		return models.LabTypeEGFR
	case strings.Contains(lower, "glucose") && strings.Contains(lower, "fasting"):
		return models.LabTypeFBG
	case strings.Contains(lower, "glucose") && !strings.Contains(lower, "fasting"):
		return models.LabTypeFBG // default glucose to FBG
	case strings.Contains(lower, "hemoglobin a1c") || strings.Contains(lower, "glycated"):
		return models.LabTypeHbA1c
	case strings.Contains(lower, "systolic"):
		return models.LabTypeSBP
	case strings.Contains(lower, "diastolic"):
		return models.LabTypeDBP
	case strings.Contains(lower, "potassium"):
		return models.LabTypePotassium
	case strings.Contains(lower, "sodium"):
		return models.LabTypeSodium
	case strings.Contains(lower, "heart rate"):
		return "HEART_RATE"
	case strings.Contains(lower, "body weight") || (strings.Contains(lower, "weight") && !strings.Contains(lower, "birth")):
		return "WEIGHT"
	case strings.Contains(lower, "albumin") && strings.Contains(lower, "creatinine") && strings.Contains(lower, "ratio"):
		return models.LabTypeACR
	case strings.Contains(lower, "cholesterol") && strings.Contains(lower, "hdl"):
		return models.LabTypeHDL
	case strings.Contains(lower, "cholesterol") && strings.Contains(lower, "total"):
		return models.LabTypeTotalCholesterol
	case strings.Contains(lower, "cholesterol"):
		return models.LabTypeTotalCholesterol
	default:
		return ""
	}
}

// ResolveLOINC maps a KB-20 lab type to its LOINC code by looking up the concept in KB-7.
// This is the forward direction: lab type name → LOINC code.
// Uses concept lookup by code — the caller must provide a known LOINC code.
func (c *KB7Client) ResolveLOINC(loincCode string) (*KB7Concept, error) {
	return c.LookupConcept(loincCode)
}

// LookupMemberships returns all ValueSets a LOINC code belongs to.
// Uses: GET /fhir/CodeSystem/$lookup-memberships?code=:code&system=http://loinc.org
func (c *KB7Client) LookupMemberships(loincCode string) (*kb7MembershipResponse, error) {
	reqURL := fmt.Sprintf("%s/fhir/CodeSystem/$lookup-memberships?code=%s&system=%s",
		c.baseURL, url.QueryEscape(loincCode), url.QueryEscape("http://loinc.org"))

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("KB-7 membership lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-7 membership returned %d: %s", resp.StatusCode, string(body))
	}

	var result kb7MembershipResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode KB-7 membership response: %w", err)
	}

	return &result, nil
}

// ValidateLOINC checks if a LOINC code belongs to a KB-7 ValueSet.
// Uses: POST /v1/rules/valuesets/:identifier/validate
func (c *KB7Client) ValidateLOINC(code, valueSetID string) (bool, error) {
	reqURL := fmt.Sprintf("%s/v1/rules/valuesets/%s/validate", c.baseURL, valueSetID)

	payload := fmt.Sprintf(`{"code":"%s","system":"http://loinc.org"}`, code)

	resp, err := c.httpClient.Post(reqURL, "application/json",
		io.NopCloser(jsonReader(payload)))
	if err != nil {
		return false, fmt.Errorf("KB-7 validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode KB-7 validation response: %w", err)
	}

	return result.Valid, nil
}

// kb7ATCResponse is the JSON from KB-7's ATC class lookup.
type kb7ATCResponse struct {
	Code      string `json:"code"`
	ATCClass  string `json:"atc_class"`
	ATCCode   string `json:"atc_code"`
	ClassName string `json:"class_name"`
}

// LookupATCClass queries KB-7 (or RxNav-in-a-Box) to determine the ATC class for a drug.
// Uses: GET /v1/concepts/ATC/:drugName or rxnav endpoint.
// For glucocorticoid detection (Track 3), checks if ATC code starts with "H02AB"
// (systemic glucocorticoids). Returns the ATC code or empty string.
func (c *KB7Client) LookupATCClass(drugName string) (string, error) {
	reqURL := fmt.Sprintf("%s/v1/concepts/ATC/%s", c.baseURL, url.QueryEscape(drugName))

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return "", fmt.Errorf("KB-7 ATC lookup failed for %s: %w", drugName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("KB-7 ATC returned %d for %s", resp.StatusCode, drugName)
	}

	var result kb7ATCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode KB-7 ATC response: %w", err)
	}

	return result.ATCCode, nil
}

// IsSystemicGlucocorticoid checks if a drug name resolves to ATC class H02AB
// (systemic glucocorticoids) via KB-7. Falls back to a static list if KB-7 is unavailable.
func (c *KB7Client) IsSystemicGlucocorticoid(drugName string) bool {
	atcCode, err := c.LookupATCClass(drugName)
	if err == nil && strings.HasPrefix(atcCode, "H02AB") {
		return true
	}

	// Fallback: static list of 7 common systemic glucocorticoids (Indian clinical practice)
	// Only used if KB-7 is unreachable — KB-7 is the source of truth
	return isGlucocorticoidFallback(strings.ToLower(drugName))
}

// isGlucocorticoidFallback is the static fallback when KB-7 is unavailable.
func isGlucocorticoidFallback(drugNameLower string) bool {
	glucocorticoids := []string{
		"prednisolone", "prednisone", "methylprednisolone",
		"dexamethasone", "hydrocortisone", "betamethasone", "deflazacort",
	}
	for _, gc := range glucocorticoids {
		if strings.Contains(drugNameLower, gc) {
			return true
		}
	}
	return false
}

// HealthCheck verifies KB-7 is reachable.
func (c *KB7Client) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("KB-7 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-7 health check returned %d", resp.StatusCode)
	}
	return nil
}
