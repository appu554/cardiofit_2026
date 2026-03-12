// Package terminology provides tools to fetch real MedDRA data from OpenFDA FAERS API
//
// This fetcher retrieves REAL adverse event data with official MedDRA PT codes
// from the FDA Adverse Event Reporting System (FAERS).
//
// Source: https://open.fda.gov/apis/drug/event/
// No API key required for basic access (limited to 1000 requests/day without key)
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

// OpenFDAClient fetches real MedDRA data from FDA FAERS database
type OpenFDAClient struct {
	baseURL    string
	httpClient *http.Client
}

// FAERSResponse represents the OpenFDA API response structure
type FAERSResponse struct {
	Meta struct {
		Disclaimer string `json:"disclaimer"`
		Terms      string `json:"terms"`
		License    string `json:"license"`
		Results    struct {
			Skip  int `json:"skip"`
			Limit int `json:"limit"`
			Total int `json:"total"`
		} `json:"results"`
	} `json:"meta"`
	Results []FAERSResult `json:"results"`
}

// FAERSResult represents a single adverse event report
type FAERSResult struct {
	SafetyReportID string      `json:"safetyreportid"`
	ReceiveDate    string      `json:"receivedate"`
	Serious        interface{} `json:"serious"` // Can be int or string
	Patient        struct {
		Reaction []FAERSReaction `json:"reaction"`
		Drug     []FAERSDrug     `json:"drug"`
	} `json:"patient"`
}

// IsSerious returns whether this is a serious adverse event
func (r *FAERSResult) IsSerious() bool {
	switch v := r.Serious.(type) {
	case int:
		return v == 1
	case float64:
		return v == 1
	case string:
		return v == "1" || v == "true" || v == "yes"
	default:
		return false
	}
}

// FAERSReaction contains the MedDRA-coded adverse reaction
type FAERSReaction struct {
	// ReactionMedDRAPT is the official MedDRA Preferred Term
	// This is the REAL MedDRA data you need!
	ReactionMedDRAPT string `json:"reactionmeddrapt"`

	// ReactionMedDRAVersion is the MedDRA version used
	ReactionMedDRAVersion string `json:"reactionmeddraversionpt"`

	// ReactionOutcome is the outcome of the reaction (1-6)
	ReactionOutcome string `json:"reactionoutcome"`
}

// FAERSDrug contains drug information from the report
type FAERSDrug struct {
	MedicinalProduct string `json:"medicinalproduct"`
	DrugIndication   string `json:"drugindication"`
	OpenFDA          struct {
		BrandName    []string `json:"brand_name"`
		GenericName  []string `json:"generic_name"`
		RxCUI        []string `json:"rxcui"` // Real RxCUI codes!
		SPLSetID     []string `json:"spl_set_id"`
	} `json:"openfda"`
}

// MedDRASample represents extracted MedDRA term from FAERS
type MedDRASample struct {
	PTName         string `json:"pt_name"`          // MedDRA Preferred Term name
	MedDRAVersion  string `json:"meddra_version"`   // Version used
	DrugName       string `json:"drug_name"`        // Associated drug
	RxCUI          string `json:"rxcui"`            // RxNorm code if available
	ReportID       string `json:"report_id"`        // FAERS safety report ID
	Serious        bool   `json:"serious"`          // Was this a serious event?
}

// NewOpenFDAClient creates a client for fetching real MedDRA data
func NewOpenFDAClient() *OpenFDAClient {
	return &OpenFDAClient{
		baseURL: "https://api.fda.gov/drug/event.json",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchMedDRASamples retrieves real MedDRA PT terms from OpenFDA FAERS
// This gives you OFFICIAL MedDRA Preferred Terms as used in FDA reports
func (c *OpenFDAClient) FetchMedDRASamples(ctx context.Context, searchTerm string, limit int) ([]MedDRASample, error) {
	if limit > 100 {
		limit = 100 // Keep reasonable for sample purposes
	}

	// Build query URL
	// Search for adverse events containing the specified term
	query := url.Values{}
	query.Set("search", fmt.Sprintf("patient.reaction.reactionmeddrapt:%s", searchTerm))
	query.Set("limit", fmt.Sprintf("%d", limit))

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, query.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OpenFDA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenFDA returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var faersResp FAERSResponse
	if err := json.Unmarshal(body, &faersResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract MedDRA samples
	var samples []MedDRASample
	seen := make(map[string]bool)

	for _, result := range faersResp.Results {
		for _, reaction := range result.Patient.Reaction {
			// Skip duplicates
			if seen[reaction.ReactionMedDRAPT] {
				continue
			}
			seen[reaction.ReactionMedDRAPT] = true

			sample := MedDRASample{
				PTName:        reaction.ReactionMedDRAPT,
				MedDRAVersion: reaction.ReactionMedDRAVersion,
				ReportID:      result.SafetyReportID,
				Serious:       result.IsSerious(),
			}

			// Get associated drug info
			if len(result.Patient.Drug) > 0 {
				drug := result.Patient.Drug[0]
				sample.DrugName = drug.MedicinalProduct
				if len(drug.OpenFDA.RxCUI) > 0 {
					sample.RxCUI = drug.OpenFDA.RxCUI[0]
				}
			}

			samples = append(samples, sample)
		}
	}

	return samples, nil
}

// FetchCommonAdverseEvents retrieves common MedDRA terms for testing
// Returns real adverse events commonly reported to FDA
func (c *OpenFDAClient) FetchCommonAdverseEvents(ctx context.Context) ([]MedDRASample, error) {
	// Common adverse events to fetch - these are frequently reported to FDA
	commonTerms := []string{
		"Nausea",
		"Headache",
		"Diarrhoea",     // MedDRA uses British spelling
		"Fatigue",
		"Dizziness",
		"Vomiting",
		"Rash",
		"Pain",
		"Pyrexia",       // Fever in MedDRA terms
		"Dyspnoea",      // Shortness of breath
		"Arthralgia",    // Joint pain
		"Insomnia",
		"Anxiety",
		"Depression",
		"Hypertension",
	}

	var allSamples []MedDRASample
	seen := make(map[string]bool)

	for _, term := range commonTerms {
		samples, err := c.FetchMedDRASamples(ctx, term, 5)
		if err != nil {
			// Log error but continue with other terms
			fmt.Printf("Warning: Failed to fetch %s: %v\n", term, err)
			continue
		}

		for _, s := range samples {
			if !seen[s.PTName] {
				seen[s.PTName] = true
				allSamples = append(allSamples, s)
			}
		}

		// Rate limiting - be nice to the API
		time.Sleep(100 * time.Millisecond)
	}

	return allSamples, nil
}

// FetchDrugSpecificEvents retrieves MedDRA-coded events for a specific drug
func (c *OpenFDAClient) FetchDrugSpecificEvents(ctx context.Context, drugName string, limit int) ([]MedDRASample, error) {
	if limit > 100 {
		limit = 100
	}

	query := url.Values{}
	query.Set("search", fmt.Sprintf("patient.drug.medicinalproduct:%s", drugName))
	query.Set("limit", fmt.Sprintf("%d", limit))

	reqURL := fmt.Sprintf("%s?%s", c.baseURL, query.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OpenFDA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenFDA returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var faersResp FAERSResponse
	if err := json.Unmarshal(body, &faersResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var samples []MedDRASample
	seen := make(map[string]bool)

	for _, result := range faersResp.Results {
		for _, reaction := range result.Patient.Reaction {
			if seen[reaction.ReactionMedDRAPT] {
				continue
			}
			seen[reaction.ReactionMedDRAPT] = true

			sample := MedDRASample{
				PTName:        reaction.ReactionMedDRAPT,
				MedDRAVersion: reaction.ReactionMedDRAVersion,
				ReportID:      result.SafetyReportID,
				Serious:       result.IsSerious(),
			}

			for _, drug := range result.Patient.Drug {
				if drug.MedicinalProduct != "" {
					sample.DrugName = drug.MedicinalProduct
					if len(drug.OpenFDA.RxCUI) > 0 {
						sample.RxCUI = drug.OpenFDA.RxCUI[0]
					}
					break
				}
			}

			samples = append(samples, sample)
		}
	}

	return samples, nil
}
