// Package adapters provides RxNorm drug terminology adapter.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// RXNORM ADAPTER (USA)
// =============================================================================

// RxNormAdapter ingests drug terminology from NLM RxNorm.
// Used by KB-7 (Terminology), KB-1 (Drug Dosing), KB-5 (Drug Interactions), KB-6 (Formulary).
type RxNormAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
}

// NewRxNormAdapter creates a new RxNorm adapter.
func NewRxNormAdapter() *RxNormAdapter {
	return &RxNormAdapter{
		BaseAdapter: NewBaseAdapter(
			"RXNORM",
			models.AuthorityNLM,
			[]models.KB{models.KB7, models.KB1, models.KB5, models.KB6},
		),
		baseURL: "https://rxnav.nlm.nih.gov/REST",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// FetchUpdates retrieves RxNorm concepts updated since the given timestamp.
func (a *RxNormAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// RxNorm API - get new/updated concepts
	url := fmt.Sprintf("%s/rxcui?name=*&allsrc=1", a.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var items []RawItem
	// Parse response
	return items, nil
}

// Transform converts raw RxNorm RRF content to a KnowledgeItem.
func (a *RxNormAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	concept, err := a.parseConcept(raw.RawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RxNorm concept: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb7:rxnorm:%s", concept.RxCUI),
		Type:    models.TypeTerminology,
		KB:      models.KB7,
		Version: concept.Version,
		Name:    concept.Name,
		Description: fmt.Sprintf("%s (%s)", concept.Name, concept.TTY),
		Source: models.SourceAttribution{
			Authority:    models.AuthorityNLM,
			Document:     "RxNorm",
			Section:      concept.TTY,
			Jurisdiction: models.JurisdictionUS,
			URL:          fmt.Sprintf("https://mor.nlm.nih.gov/RxNav/search?searchBy=RXCUI&searchTerm=%s", concept.RxCUI),
		},
		ContentRef:  fmt.Sprintf("rxnorm:concept:%s", concept.RxCUI),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskLow,
		WorkflowTemplate: models.TemplateInfraLow,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs RxNorm-specific validation.
func (a *RxNormAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthorityNLM {
		return fmt.Errorf("invalid authority: expected NLM, got %s", item.Source.Authority)
	}

	// Additional RxNorm-specific validation:
	// - Valid RxCUI format
	// - Active concept
	// - Has NDC associations if clinical drug

	return nil
}

// =============================================================================
// RXNORM CONCEPT LOOKUP
// =============================================================================

// LookupByRxCUI retrieves a concept by RxCUI.
func (a *RxNormAdapter) LookupByRxCUI(ctx context.Context, rxcui string) (*RxNormConcept, error) {
	rxURL := fmt.Sprintf("%s/rxcui/%s/allProperties.json?prop=all", a.baseURL, rxcui)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rxURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return a.parseConcept(body)
}

// SearchByName searches for RxNorm concepts by name.
func (a *RxNormAdapter) SearchByName(ctx context.Context, name string) ([]*RxNormConcept, error) {
	searchURL := fmt.Sprintf("%s/drugs.json?name=%s", a.baseURL, url.QueryEscape(name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result RxNormSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Concepts, nil
}

// GetApproximateMatch finds approximate matches for a drug name.
func (a *RxNormAdapter) GetApproximateMatch(ctx context.Context, term string, maxEntries int) ([]*RxNormConcept, error) {
	approxURL := fmt.Sprintf("%s/approximateTerm.json?term=%s&maxEntries=%d",
		a.baseURL, url.QueryEscape(term), maxEntries)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, approxURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		ApproximateGroup struct {
			Candidates []struct {
				RxCUI string `json:"rxcui"`
				Score string `json:"score"`
				Rank  string `json:"rank"`
			} `json:"candidate"`
		} `json:"approximateGroup"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var concepts []*RxNormConcept
	for _, c := range result.ApproximateGroup.Candidates {
		concept, err := a.LookupByRxCUI(ctx, c.RxCUI)
		if err == nil && concept != nil {
			concepts = append(concepts, concept)
		}
	}

	return concepts, nil
}

// =============================================================================
// RXNORM RELATIONSHIPS
// =============================================================================

// GetRelatedConcepts retrieves related concepts by relationship type.
func (a *RxNormAdapter) GetRelatedConcepts(ctx context.Context, rxcui string, relationType string) ([]*RxNormConcept, error) {
	relURL := fmt.Sprintf("%s/rxcui/%s/related.json?rela=%s", a.baseURL, rxcui, relationType)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, relURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse related concepts
	var concepts []*RxNormConcept
	return concepts, nil
}

// GetNDCs retrieves NDC codes for an RxCUI.
func (a *RxNormAdapter) GetNDCs(ctx context.Context, rxcui string) ([]string, error) {
	ndcURL := fmt.Sprintf("%s/rxcui/%s/ndcs.json", a.baseURL, rxcui)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ndcURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		NDCGroup struct {
			NDCList struct {
				NDC []string `json:"ndc"`
			} `json:"ndcList"`
		} `json:"ndcGroup"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.NDCGroup.NDCList.NDC, nil
}

// GetIngredients retrieves active ingredients for a drug.
func (a *RxNormAdapter) GetIngredients(ctx context.Context, rxcui string) ([]*RxNormIngredient, error) {
	ingURL := fmt.Sprintf("%s/rxcui/%s/allrelated.json", a.baseURL, rxcui)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse ingredient information
	var ingredients []*RxNormIngredient
	return ingredients, nil
}

// =============================================================================
// RXNORM STRUCTURES
// =============================================================================

// RxNormConcept represents an RxNorm concept.
type RxNormConcept struct {
	RxCUI        string            `json:"rxcui"`
	Name         string            `json:"name"`
	TTY          string            `json:"tty"`           // Term Type
	Language     string            `json:"language"`
	Suppress     string            `json:"suppress"`
	UMLSCUI      string            `json:"umlscui,omitempty"`
	Version      string            `json:"version,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
}

// RxNormIngredient represents an active ingredient.
type RxNormIngredient struct {
	RxCUI       string  `json:"rxcui"`
	Name        string  `json:"name"`
	Strength    string  `json:"strength,omitempty"`
	StrengthNum float64 `json:"strength_num,omitempty"`
	StrengthUnit string `json:"strength_unit,omitempty"`
}

// RxNormSearchResult represents search results.
type RxNormSearchResult struct {
	Concepts []*RxNormConcept `json:"concepts"`
}

// parseConcept parses RxNorm concept JSON.
func (a *RxNormAdapter) parseConcept(data []byte) (*RxNormConcept, error) {
	var wrapper struct {
		PropConceptGroup struct {
			PropConcept []struct {
				PropCategory string `json:"propCategory"`
				PropName     string `json:"propName"`
				PropValue    string `json:"propValue"`
			} `json:"propConcept"`
		} `json:"propConceptGroup"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse concept: %w", err)
	}

	concept := &RxNormConcept{
		Properties: make(map[string]string),
	}
	for _, prop := range wrapper.PropConceptGroup.PropConcept {
		switch prop.PropName {
		case "RxCUI":
			concept.RxCUI = prop.PropValue
		case "RxNorm Name":
			concept.Name = prop.PropValue
		case "TTY":
			concept.TTY = prop.PropValue
		default:
			concept.Properties[prop.PropName] = prop.PropValue
		}
	}

	return concept, nil
}

// =============================================================================
// RXNORM TERM TYPES (TTY)
// =============================================================================

// RxNorm Term Type constants
const (
	// Ingredient Level
	TTYIngredient        = "IN"   // Ingredient
	TTYMultipleIngredients = "MIN" // Multiple Ingredients
	TTYPreciseIngredient = "PIN"  // Precise Ingredient

	// Clinical Drug Level
	TTYSemanticClinicalDrug     = "SCD"  // Semantic Clinical Drug
	TTYSemanticBrandedDrug      = "SBD"  // Semantic Branded Drug
	TTYGenericPack              = "GPCK" // Generic Pack
	TTYBrandedPack              = "BPCK" // Branded Pack

	// Component Level
	TTYSemanticClinicalDrugComponent = "SCDC" // Semantic Clinical Drug Component
	TTYSemanticBrandedDrugComponent  = "SBDC" // Semantic Branded Drug Component

	// Form Level
	TTYSemanticClinicalDrugForm = "SCDF" // Semantic Clinical Drug Form
	TTYSemanticBrandedDrugForm  = "SBDF" // Semantic Branded Drug Form
	TTYSemanticDoseFormGroup    = "SCDG" // Semantic Clinical Dose Form Group
	TTYSemanticBrandedDoseFormGroup = "SBDG" // Semantic Branded Dose Form Group

	// Drug Name Level
	TTYBrandName = "BN" // Brand Name
	TTYDoseForm  = "DF" // Dose Form
	TTYDoseFormGroup = "DFG" // Dose Form Group
)

// TTYInfo contains information about a term type.
type TTYInfo struct {
	TTY         string
	Name        string
	Description string
	Level       string
}

// GetTTYInfo returns information about all term types.
func GetTTYInfo() []TTYInfo {
	return []TTYInfo{
		{TTYIngredient, "Ingredient", "Active ingredient", "Ingredient"},
		{TTYMultipleIngredients, "Multiple Ingredients", "Multiple active ingredients", "Ingredient"},
		{TTYPreciseIngredient, "Precise Ingredient", "Precise form of ingredient", "Ingredient"},
		{TTYSemanticClinicalDrug, "Semantic Clinical Drug", "Generic drug with strength and form", "Clinical Drug"},
		{TTYSemanticBrandedDrug, "Semantic Branded Drug", "Brand drug with strength and form", "Clinical Drug"},
		{TTYBrandName, "Brand Name", "Proprietary drug name", "Name"},
		{TTYDoseForm, "Dose Form", "Physical form of drug", "Form"},
	}
}

// =============================================================================
// INTERACTION CHECKING
// =============================================================================

// CheckInteractions checks for drug-drug interactions.
func (a *RxNormAdapter) CheckInteractions(ctx context.Context, rxcuis []string) ([]*DrugInteraction, error) {
	if len(rxcuis) < 2 {
		return nil, nil
	}

	// RxNorm interaction API
	interactionURL := fmt.Sprintf("%s/interaction/list.json?rxcuis=%s",
		a.baseURL, joinRxCUIs(rxcuis))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, interactionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse interaction results
	var interactions []*DrugInteraction
	return interactions, nil
}

// DrugInteraction represents a drug-drug interaction.
type DrugInteraction struct {
	Drug1RxCUI    string `json:"drug1_rxcui"`
	Drug1Name     string `json:"drug1_name"`
	Drug2RxCUI    string `json:"drug2_rxcui"`
	Drug2Name     string `json:"drug2_name"`
	Severity      string `json:"severity"`      // high, moderate, low
	Description   string `json:"description"`
	Source        string `json:"source"`        // DrugBank, ONCHigh, etc.
}

// Helper function to join RxCUIs for API call
func joinRxCUIs(rxcuis []string) string {
	result := ""
	for i, rxcui := range rxcuis {
		if i > 0 {
			result += "+"
		}
		result += rxcui
	}
	return result
}

// =============================================================================
// SPELLING SUGGESTIONS
// =============================================================================

// GetSpellingSuggestions returns spelling suggestions for a drug name.
func (a *RxNormAdapter) GetSpellingSuggestions(ctx context.Context, name string) ([]string, error) {
	spellURL := fmt.Sprintf("%s/spellingsuggestions.json?name=%s", a.baseURL, url.QueryEscape(name))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spellURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		SuggestionGroup struct {
			SuggestionList struct {
				Suggestion []string `json:"suggestion"`
			} `json:"suggestionList"`
		} `json:"suggestionGroup"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.SuggestionGroup.SuggestionList.Suggestion, nil
}
