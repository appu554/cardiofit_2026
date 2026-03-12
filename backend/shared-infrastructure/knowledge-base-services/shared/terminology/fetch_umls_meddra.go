// Package terminology provides tools to fetch real MedDRA data from UMLS API
//
// UMLS (Unified Medical Language System) includes MedDRA as source vocabulary "MDR".
// This fetcher retrieves REAL MedDRA terms with official PT codes from NLM's UMLS API.
//
// UMLS License: FREE for all users (requires registration at uts.nlm.nih.gov)
// MedDRA Source: MDR (Medical Dictionary for Regulatory Activities)
//
// API Documentation: https://documentation.uts.nlm.nih.gov/rest/home.html
package terminology

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// UMLSClient fetches real MedDRA data from NLM UMLS API
type UMLSClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// UMLSSearchResponse represents the UMLS search API response
type UMLSSearchResponse struct {
	Result struct {
		ClassType string       `json:"classType"`
		Results   []UMLSResult `json:"results"`
	} `json:"result"`
}

// UMLSResult represents a single UMLS search result
type UMLSResult struct {
	UI       string `json:"ui"`       // Concept Unique Identifier (CUI)
	Name     string `json:"name"`     // Preferred name
	RootSource string `json:"rootSource"` // Source vocabulary (MDR for MedDRA)
	URI      string `json:"uri"`      // Full URI to concept
}

// UMLSAtomResponse represents UMLS atoms (source-specific terms)
type UMLSAtomResponse struct {
	Result []UMLSAtom `json:"result"`
}

// UMLSAtom represents a source-specific term in UMLS
type UMLSAtom struct {
	UI           string `json:"ui"`           // Atom Unique Identifier (AUI)
	Name         string `json:"name"`         // Term name
	SourceConcept string `json:"sourceConcept"` // Source-specific code (MedDRA PT code!)
	RootSource   string `json:"rootSource"`   // Source vocabulary
	TermType     string `json:"termType"`     // Term type (PT, LLT, etc.)
}

// UMLSMedDRATerm represents extracted MedDRA term from UMLS
type UMLSMedDRATerm struct {
	CUI          string `json:"cui"`           // UMLS Concept Unique Identifier
	PTCode       string `json:"pt_code"`       // MedDRA Preferred Term code
	PTName       string `json:"pt_name"`       // MedDRA Preferred Term name
	TermType     string `json:"term_type"`     // PT, LLT, HLT, HLGT, SOC
	SOCCode      string `json:"soc_code"`      // System Organ Class code if available
	SOCName      string `json:"soc_name"`      // SOC name if available
	Source       string `json:"source"`        // Always "MDR" for MedDRA
}

// NewUMLSClient creates a client for fetching real MedDRA data from UMLS
func NewUMLSClient(apiKey string) *UMLSClient {
	return &UMLSClient{
		apiKey:  apiKey,
		baseURL: "https://uts-ws.nlm.nih.gov/rest",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SearchMedDRA searches UMLS for MedDRA terms matching the query
// Returns real MedDRA PT codes from the official UMLS Metathesaurus
func (c *UMLSClient) SearchMedDRA(ctx context.Context, searchTerm string) ([]UMLSMedDRATerm, error) {
	// Build search URL - restrict to MDR (MedDRA) source
	params := url.Values{}
	params.Set("apiKey", c.apiKey)
	params.Set("string", searchTerm)
	params.Set("sabs", "MDR")           // Restrict to MedDRA source
	params.Set("returnIdType", "code")  // Return source codes (MedDRA codes)
	params.Set("pageSize", "25")

	reqURL := fmt.Sprintf("%s/search/current?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from UMLS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("UMLS returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp UMLSSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract MedDRA terms from results
	var terms []UMLSMedDRATerm
	for _, result := range searchResp.Result.Results {
		if result.RootSource == "MDR" {
			term := UMLSMedDRATerm{
				CUI:      result.UI,
				PTName:   result.Name,
				Source:   "MDR",
			}

			// Get the MedDRA code from atoms
			atoms, err := c.getAtoms(ctx, result.UI)
			if err == nil {
				for _, atom := range atoms {
					if atom.RootSource == "MDR" {
						term.PTCode = atom.SourceConcept
						term.TermType = atom.TermType
						break
					}
				}
			}

			terms = append(terms, term)
		}
	}

	return terms, nil
}

// getAtoms retrieves source-specific atoms for a UMLS concept
func (c *UMLSClient) getAtoms(ctx context.Context, cui string) ([]UMLSAtom, error) {
	params := url.Values{}
	params.Set("apiKey", c.apiKey)
	params.Set("sabs", "MDR")
	params.Set("ttys", "PT,LLT") // Get Preferred Terms and Lowest Level Terms
	params.Set("pageSize", "25")

	reqURL := fmt.Sprintf("%s/content/current/CUI/%s/atoms?%s", c.baseURL, cui, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("UMLS atoms returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var atomResp UMLSAtomResponse
	if err := json.Unmarshal(body, &atomResp); err != nil {
		return nil, err
	}

	return atomResp.Result, nil
}

// FetchCommonMedDRATerms retrieves common adverse event MedDRA terms for testing
// Returns real MedDRA PT codes from UMLS
func (c *UMLSClient) FetchCommonMedDRATerms(ctx context.Context) ([]UMLSMedDRATerm, error) {
	// Common adverse events to fetch - these are frequently reported
	commonTerms := []string{
		"Nausea",
		"Headache",
		"Diarrhoea",
		"Fatigue",
		"Dizziness",
		"Vomiting",
		"Rash",
		"Pyrexia",
		"Arthralgia",
		"Insomnia",
		"Anxiety",
		"Depression",
		"Hypertension",
		"Tachycardia",
		"Dyspnoea",
	}

	var allTerms []UMLSMedDRATerm
	seen := make(map[string]bool)

	for _, term := range commonTerms {
		results, err := c.SearchMedDRA(ctx, term)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch %s: %v\n", term, err)
			continue
		}

		for _, t := range results {
			if !seen[t.PTCode] && t.PTCode != "" {
				seen[t.PTCode] = true
				allTerms = append(allTerms, t)
			}
		}

		// Rate limiting - be nice to the API
		time.Sleep(200 * time.Millisecond)
	}

	return allTerms, nil
}

// GetMedDRAByCode retrieves MedDRA term details by PT code
func (c *UMLSClient) GetMedDRAByCode(ctx context.Context, ptCode string) (*UMLSMedDRATerm, error) {
	// Search by source code
	params := url.Values{}
	params.Set("apiKey", c.apiKey)
	params.Set("string", ptCode)
	params.Set("sabs", "MDR")
	params.Set("searchType", "exact")
	params.Set("inputType", "sourceCode")

	reqURL := fmt.Sprintf("%s/search/current?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from UMLS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("UMLS returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var searchResp UMLSSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(searchResp.Result.Results) == 0 {
		return nil, fmt.Errorf("MedDRA code %s not found in UMLS", ptCode)
	}

	result := searchResp.Result.Results[0]
	return &UMLSMedDRATerm{
		CUI:      result.UI,
		PTCode:   ptCode,
		PTName:   result.Name,
		Source:   "MDR",
	}, nil
}
