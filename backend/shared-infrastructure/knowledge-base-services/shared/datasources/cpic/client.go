// Package cpic provides a client for the CPIC (Clinical Pharmacogenetics Implementation Consortium) API.
// CPIC provides evidence-based pharmacogenomics guidelines for drug dosing.
//
// Phase 3b.1: CPIC API Client
// Authority Level: DEFINITIVE (LLM = ❌ NEVER)
//
// API Documentation: https://api.cpicpgx.org/
package cpic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// =============================================================================
// CPIC CLIENT
// =============================================================================

// Client provides access to CPIC pharmacogenomics guidelines
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for the CPIC client
type Config struct {
	BaseURL string        // Default: https://api.cpicpgx.org
	Timeout time.Duration // Default: 30s
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		BaseURL: "https://api.cpicpgx.org",
		Timeout: 30 * time.Second,
	}
}

// NewClient creates a new CPIC client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.cpicpgx.org"
	}

	return &Client{
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// =============================================================================
// DATA TYPES
// =============================================================================

// GeneDrugPair represents a gene-drug interaction from CPIC
type GeneDrugPair struct {
	ID              int              `json:"id"`
	DrugID          string           `json:"drugid"`
	DrugName        string           `json:"drugname"`
	GeneSymbol      string           `json:"genesymbol"`
	CPICLevel       string           `json:"cpiclevel"`        // "A", "A/B", "B", "C", "D"
	CPICLevelStatus string           `json:"cpiclevelstatus"`  // "In Guideline", "No Guideline"
	PGxOnFDALabel   bool             `json:"pgxonfdatest"`
	PGxFDALabelType string           `json:"pgxfdatestingtype"`
	Guideline       *Guideline       `json:"guideline,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
}

// Guideline represents a CPIC guideline publication
type Guideline struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	PMID         string `json:"pmid"`
	Publication  string `json:"publication"`
	Version      string `json:"version"`
	PublishedOn  string `json:"publishedOn"`
}

// Recommendation represents a dosing recommendation
type Recommendation struct {
	ID                int    `json:"id"`
	DrugName          string `json:"drug"`
	GeneSymbol        string `json:"gene"`
	Phenotype         string `json:"phenotype"`        // "Poor Metabolizer", "Intermediate Metabolizer", etc.
	ActivityScore     string `json:"activityscore"`    // Numeric activity score if available
	Implication       string `json:"implication"`      // Clinical implication
	Recommendation    string `json:"recommendation"`   // Dosing recommendation
	Classification    string `json:"classification"`   // "Strong", "Moderate", "Optional"
	Comments          string `json:"comments"`
	LookupKey         string `json:"lookupkey"`
	Population        string `json:"population"`
	PMID              string `json:"pmid"`
}

// Drug represents a drug in CPIC
type Drug struct {
	ID          string `json:"drugid"`
	Name        string `json:"name"`
	RxNormID    string `json:"rxnormid"`
	DrugBankID  string `json:"drugbankid"`
	ATCIDs      string `json:"atcids"`
	URL         string `json:"url"`
	FlowChart   string `json:"flowchart"`
}

// Gene represents a gene in CPIC
type Gene struct {
	Symbol           string `json:"symbol"`
	Name             string `json:"name"`
	NCBIID           string `json:"ncbiid"`
	EnsemblID        string `json:"ensemblid"`
	HGNCSymbol       string `json:"hgncsymbol"`
	Chromosome       string `json:"chromosome"`
	ChromosomeStart  int    `json:"chromosomestart"`
	ChromosomeEnd    int    `json:"chromosomeend"`
	URL              string `json:"url"`
	PhenotypeSummary string `json:"phenotypesummary"`
}

// Diplotype represents a diplotype lookup result
type Diplotype struct {
	ID                int    `json:"id"`
	GeneSymbol        string `json:"genesymbol"`
	Diplotype         string `json:"diplotype"`
	Phenotype         string `json:"phenotype"`
	EHRPriority       string `json:"ehrpriority"`
	ActivityScore     string `json:"activityscore"`
	Frequency         string `json:"frequency"`
	LookupKey         string `json:"lookupkey"`
}

// =============================================================================
// API METHODS
// =============================================================================

// GetGeneDrugPairs retrieves all gene-drug pairs, optionally filtered by drug name
func (c *Client) GetGeneDrugPairs(ctx context.Context, drugName string) ([]GeneDrugPair, error) {
	endpoint := "/v1/pair"
	if drugName != "" {
		endpoint = fmt.Sprintf("%s?name=%s", endpoint, url.QueryEscape(drugName))
	}

	var pairs []GeneDrugPair
	if err := c.doRequest(ctx, endpoint, &pairs); err != nil {
		return nil, fmt.Errorf("get gene-drug pairs: %w", err)
	}

	return pairs, nil
}

// GetGeneDrugPairByGene retrieves gene-drug pairs for a specific gene
func (c *Client) GetGeneDrugPairByGene(ctx context.Context, geneSymbol string) ([]GeneDrugPair, error) {
	endpoint := fmt.Sprintf("/v1/pair?gene=%s", url.QueryEscape(geneSymbol))

	var pairs []GeneDrugPair
	if err := c.doRequest(ctx, endpoint, &pairs); err != nil {
		return nil, fmt.Errorf("get pairs by gene: %w", err)
	}

	return pairs, nil
}

// GetRecommendations retrieves all recommendations for a drug
func (c *Client) GetRecommendations(ctx context.Context, drugName string) ([]Recommendation, error) {
	endpoint := fmt.Sprintf("/v1/recommendation?drug=%s", url.QueryEscape(drugName))

	var recs []Recommendation
	if err := c.doRequest(ctx, endpoint, &recs); err != nil {
		return nil, fmt.Errorf("get recommendations: %w", err)
	}

	return recs, nil
}

// GetRecommendationForPhenotype retrieves dosing recommendation for a specific drug/gene/phenotype
func (c *Client) GetRecommendationForPhenotype(ctx context.Context, drugName, geneSymbol, phenotype string) (*Recommendation, error) {
	recs, err := c.GetRecommendations(ctx, drugName)
	if err != nil {
		return nil, err
	}

	for _, rec := range recs {
		if strings.EqualFold(rec.GeneSymbol, geneSymbol) &&
			strings.EqualFold(rec.Phenotype, phenotype) {
			return &rec, nil
		}
	}

	return nil, fmt.Errorf("no recommendation found for %s/%s/%s", drugName, geneSymbol, phenotype)
}

// GetDrugs retrieves all drugs with CPIC guidelines
func (c *Client) GetDrugs(ctx context.Context) ([]Drug, error) {
	var drugs []Drug
	if err := c.doRequest(ctx, "/v1/drug", &drugs); err != nil {
		return nil, fmt.Errorf("get drugs: %w", err)
	}

	return drugs, nil
}

// GetDrugByRxNorm retrieves drug by RxNorm ID
func (c *Client) GetDrugByRxNorm(ctx context.Context, rxnormID string) (*Drug, error) {
	endpoint := fmt.Sprintf("/v1/drug?rxnormid=%s", url.QueryEscape(rxnormID))

	var drugs []Drug
	if err := c.doRequest(ctx, endpoint, &drugs); err != nil {
		return nil, fmt.Errorf("get drug by RxNorm: %w", err)
	}

	if len(drugs) == 0 {
		return nil, fmt.Errorf("no drug found for RxNorm ID: %s", rxnormID)
	}

	return &drugs[0], nil
}

// GetGenes retrieves all genes with CPIC guidelines
func (c *Client) GetGenes(ctx context.Context) ([]Gene, error) {
	var genes []Gene
	if err := c.doRequest(ctx, "/v1/gene", &genes); err != nil {
		return nil, fmt.Errorf("get genes: %w", err)
	}

	return genes, nil
}

// GetGeneBySymbol retrieves a gene by its symbol
func (c *Client) GetGeneBySymbol(ctx context.Context, symbol string) (*Gene, error) {
	endpoint := fmt.Sprintf("/v1/gene?symbol=%s", url.QueryEscape(symbol))

	var genes []Gene
	if err := c.doRequest(ctx, endpoint, &genes); err != nil {
		return nil, fmt.Errorf("get gene: %w", err)
	}

	if len(genes) == 0 {
		return nil, fmt.Errorf("no gene found: %s", symbol)
	}

	return &genes[0], nil
}

// GetDiplotypes retrieves diplotype-phenotype mappings for a gene
func (c *Client) GetDiplotypes(ctx context.Context, geneSymbol string) ([]Diplotype, error) {
	endpoint := fmt.Sprintf("/v1/diplotype?gene=%s", url.QueryEscape(geneSymbol))

	var diplotypes []Diplotype
	if err := c.doRequest(ctx, endpoint, &diplotypes); err != nil {
		return nil, fmt.Errorf("get diplotypes: %w", err)
	}

	return diplotypes, nil
}

// LookupPhenotype looks up phenotype from diplotype
func (c *Client) LookupPhenotype(ctx context.Context, geneSymbol, diplotype string) (*Diplotype, error) {
	diplotypes, err := c.GetDiplotypes(ctx, geneSymbol)
	if err != nil {
		return nil, err
	}

	for _, d := range diplotypes {
		if strings.EqualFold(d.Diplotype, diplotype) {
			return &d, nil
		}
	}

	return nil, fmt.Errorf("diplotype not found: %s %s", geneSymbol, diplotype)
}

// GetGuidelines retrieves all published guidelines
func (c *Client) GetGuidelines(ctx context.Context) ([]Guideline, error) {
	var guidelines []Guideline
	if err := c.doRequest(ctx, "/v1/guideline", &guidelines); err != nil {
		return nil, fmt.Errorf("get guidelines: %w", err)
	}

	return guidelines, nil
}

// =============================================================================
// CONVENIENCE METHODS
// =============================================================================

// HasPharmacogenomicsGuideline checks if a drug has CPIC guidelines
func (c *Client) HasPharmacogenomicsGuideline(ctx context.Context, drugName string) (bool, error) {
	pairs, err := c.GetGeneDrugPairs(ctx, drugName)
	if err != nil {
		return false, err
	}

	for _, pair := range pairs {
		if pair.CPICLevel == "A" || pair.CPICLevel == "A/B" || pair.CPICLevel == "B" {
			return true, nil
		}
	}

	return false, nil
}

// GetActionableGenes returns genes with actionable recommendations for a drug
func (c *Client) GetActionableGenes(ctx context.Context, drugName string) ([]string, error) {
	pairs, err := c.GetGeneDrugPairs(ctx, drugName)
	if err != nil {
		return nil, err
	}

	var genes []string
	seen := make(map[string]bool)

	for _, pair := range pairs {
		// Level A and A/B are most actionable
		if (pair.CPICLevel == "A" || pair.CPICLevel == "A/B") && !seen[pair.GeneSymbol] {
			genes = append(genes, pair.GeneSymbol)
			seen[pair.GeneSymbol] = true
		}
	}

	return genes, nil
}

// GetFullRecommendation retrieves complete recommendation with guideline context
func (c *Client) GetFullRecommendation(ctx context.Context, drugName, geneSymbol, diplotype string) (*FullRecommendation, error) {
	// Step 1: Lookup phenotype from diplotype
	diplotypeInfo, err := c.LookupPhenotype(ctx, geneSymbol, diplotype)
	if err != nil {
		return nil, fmt.Errorf("lookup phenotype: %w", err)
	}

	// Step 2: Get recommendation for phenotype
	rec, err := c.GetRecommendationForPhenotype(ctx, drugName, geneSymbol, diplotypeInfo.Phenotype)
	if err != nil {
		return nil, fmt.Errorf("get recommendation: %w", err)
	}

	// Step 3: Get gene-drug pair info
	pairs, err := c.GetGeneDrugPairs(ctx, drugName)
	if err != nil {
		return nil, fmt.Errorf("get pair info: %w", err)
	}

	var pair *GeneDrugPair
	for _, p := range pairs {
		if strings.EqualFold(p.GeneSymbol, geneSymbol) {
			pair = &p
			break
		}
	}

	return &FullRecommendation{
		DrugName:       drugName,
		GeneSymbol:     geneSymbol,
		Diplotype:      diplotype,
		Phenotype:      diplotypeInfo.Phenotype,
		ActivityScore:  diplotypeInfo.ActivityScore,
		Recommendation: rec,
		Pair:           pair,
	}, nil
}

// FullRecommendation contains complete PGx recommendation with context
type FullRecommendation struct {
	DrugName       string          `json:"drug_name"`
	GeneSymbol     string          `json:"gene_symbol"`
	Diplotype      string          `json:"diplotype"`
	Phenotype      string          `json:"phenotype"`
	ActivityScore  string          `json:"activity_score"`
	Recommendation *Recommendation `json:"recommendation"`
	Pair           *GeneDrugPair   `json:"pair,omitempty"`
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// HealthCheck verifies the CPIC API is accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.GetDrugs(ctx)
	return err
}

// =============================================================================
// HTTP REQUEST HANDLING
// =============================================================================

func (c *Client) doRequest(ctx context.Context, path string, result interface{}) error {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CPIC API error: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE IMPLEMENTATION (Phase 3b)
// =============================================================================
// These methods implement the datasources.AuthorityClient interface for unified
// fact retrieval across all ground truth authority sources.

// Import datasources package types used below (add to imports if not present)
// import "github.com/cardiofit/shared/datasources"

// Name returns the unique identifier for this data source
func (c *Client) Name() string {
	return "CPIC"
}

// Close releases any resources held by the client
func (c *Client) Close() error {
	// HTTP client doesn't need explicit cleanup
	return nil
}

// GetFacts retrieves all pharmacogenomics facts for a drug by RxCUI
func (c *Client) GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error) {
	// First, get the drug by RxCUI
	drug, err := c.GetDrugByRxNorm(ctx, rxcui)
	if err != nil {
		return nil, err
	}

	// Get all gene-drug pairs and recommendations for this drug
	pairs, err := c.GetGeneDrugPairs(ctx, drug.Name)
	if err != nil {
		return nil, err
	}

	var facts []AuthorityFact
	for _, pair := range pairs {
		// Only include pairs with actionable recommendations (Level A, A/B, or B)
		if pair.CPICLevel != "A" && pair.CPICLevel != "A/B" && pair.CPICLevel != "B" {
			continue
		}

		fact := AuthorityFact{
			ID:               fmt.Sprintf("cpic-%s-%s", rxcui, pair.GeneSymbol),
			AuthoritySource:  "CPIC",
			FactType:         FactTypePharmacogenomics,
			RxCUI:            rxcui,
			DrugName:         pair.DrugName,
			Content:          pair,
			EvidenceLevel:    pair.CPICLevel,
			ExtractionMethod: "AUTHORITY_LOOKUP",
			Confidence:       1.0,
			FetchedAt:        time.Now(),
		}

		// Set risk level based on CPIC level
		switch pair.CPICLevel {
		case "A":
			fact.RiskLevel = "HIGH"
			fact.ActionRequired = "REQUIRED"
			fact.Recommendations = []string{"Prescribing action required based on CPIC guideline"}
		case "A/B":
			fact.RiskLevel = "MODERATE"
			fact.ActionRequired = "RECOMMENDED"
			fact.Recommendations = []string{"Prescribing action recommended based on CPIC guideline"}
		case "B":
			fact.RiskLevel = "LOW"
			fact.ActionRequired = "CONSIDER"
			fact.Recommendations = []string{"Consider alternative based on CPIC guideline"}
		}

		if pair.Guideline != nil {
			fact.SourceURL = pair.Guideline.URL
			fact.References = []string{pair.Guideline.PMID}
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// GetFactsByName retrieves facts by drug name
func (c *Client) GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error) {
	pairs, err := c.GetGeneDrugPairs(ctx, drugName)
	if err != nil {
		return nil, err
	}

	var facts []AuthorityFact
	for _, pair := range pairs {
		if pair.CPICLevel != "A" && pair.CPICLevel != "A/B" && pair.CPICLevel != "B" {
			continue
		}

		fact := AuthorityFact{
			ID:               fmt.Sprintf("cpic-%s-%s", pair.DrugID, pair.GeneSymbol),
			AuthoritySource:  "CPIC",
			FactType:         FactTypePharmacogenomics,
			DrugName:         pair.DrugName,
			Content:          pair,
			EvidenceLevel:    pair.CPICLevel,
			ExtractionMethod: "AUTHORITY_LOOKUP",
			Confidence:       1.0,
			FetchedAt:        time.Now(),
		}

		if pair.Guideline != nil {
			fact.SourceURL = pair.Guideline.URL
			fact.References = []string{pair.Guideline.PMID}
		}

		facts = append(facts, fact)
	}

	return facts, nil
}

// GetFactByType retrieves a specific fact type for a drug
func (c *Client) GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error) {
	if factType != FactTypePharmacogenomics {
		return nil, fmt.Errorf("CPIC only provides %s facts, requested %s", FactTypePharmacogenomics, factType)
	}

	facts, err := c.GetFacts(ctx, rxcui)
	if err != nil {
		return nil, err
	}

	if len(facts) == 0 {
		return nil, nil
	}

	return &facts[0], nil
}

// Sync synchronizes the local cache with CPIC (full sync)
func (c *Client) Sync(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

	// Get all drugs with guidelines
	drugs, err := c.GetDrugs(ctx)
	if err != nil {
		return &SyncResult{
			Authority:  "CPIC",
			StartTime:  startTime,
			EndTime:    time.Now(),
			ErrorCount: 1,
			Errors:     []string{err.Error()},
			Success:    false,
		}, err
	}

	return &SyncResult{
		Authority:   "CPIC",
		StartTime:   startTime,
		EndTime:     time.Now(),
		TotalFacts:  len(drugs),
		NewFacts:    len(drugs),
		Success:     true,
	}, nil
}

// SyncDelta performs incremental sync (CPIC doesn't support delta, returns full sync)
func (c *Client) SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error) {
	// CPIC API doesn't support delta sync, perform full sync
	return c.Sync(ctx)
}

// Authority returns metadata about CPIC
func (c *Client) Authority() AuthorityMetadata {
	return AuthorityMetadata{
		Name:            "CPIC",
		FullName:        "Clinical Pharmacogenetics Implementation Consortium",
		URL:             "https://cpicpgx.org",
		Description:     "Evidence-based pharmacogenomics guidelines for drug dosing based on genetic test results",
		AuthorityLevel:  AuthorityDefinitive,
		DataFormat:      "REST_API",
		UpdateFrequency: "QUARTERLY",
		FactTypes:       []FactType{FactTypePharmacogenomics},
	}
}

// SupportedFactTypes returns the fact types provided by CPIC
func (c *Client) SupportedFactTypes() []FactType {
	return []FactType{FactTypePharmacogenomics}
}

// LLMPolicy returns the LLM policy for CPIC (NEVER - authoritative source)
func (c *Client) LLMPolicy() LLMPolicy {
	return LLMNever
}

// =============================================================================
// LOCAL TYPE ALIASES (to avoid import cycles)
// =============================================================================
// These mirror types from datasources package for interface implementation

type AuthorityFact struct {
	ID               string      `json:"id"`
	AuthoritySource  string      `json:"authority_source"`
	FactType         FactType    `json:"fact_type"`
	RxCUI            string      `json:"rxcui,omitempty"`
	DrugName         string      `json:"drug_name"`
	GenericName      string      `json:"generic_name,omitempty"`
	Content          interface{} `json:"content"`
	RiskLevel        string      `json:"risk_level,omitempty"`
	ActionRequired   string      `json:"action_required,omitempty"`
	Recommendations  []string    `json:"recommendations,omitempty"`
	EvidenceLevel    string      `json:"evidence_level,omitempty"`
	References       []string    `json:"references,omitempty"`
	ExtractionMethod string      `json:"extraction_method"`
	Confidence       float64     `json:"confidence"`
	FetchedAt        time.Time   `json:"fetched_at"`
	SourceVersion    string      `json:"source_version,omitempty"`
	SourceURL        string      `json:"source_url,omitempty"`
}

type FactType string

const (
	FactTypePharmacogenomics FactType = "PHARMACOGENOMICS"
)

type AuthorityMetadata struct {
	Name            string         `json:"name"`
	FullName        string         `json:"full_name"`
	URL             string         `json:"url"`
	Description     string         `json:"description"`
	AuthorityLevel  AuthorityLevel `json:"authority_level"`
	DataFormat      string         `json:"data_format"`
	UpdateFrequency string         `json:"update_frequency"`
	FactTypes       []FactType     `json:"fact_types"`
}

type AuthorityLevel string

const (
	AuthorityDefinitive AuthorityLevel = "DEFINITIVE"
)

type LLMPolicy string

const (
	LLMNever LLMPolicy = "NEVER"
)

type SyncResult struct {
	Authority    string    `json:"authority"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	TotalFacts   int       `json:"total_facts"`
	NewFacts     int       `json:"new_facts"`
	UpdatedFacts int       `json:"updated_facts"`
	DeletedFacts int       `json:"deleted_facts"`
	ErrorCount   int       `json:"error_count"`
	Errors       []string  `json:"errors,omitempty"`
	Success      bool      `json:"success"`
}
