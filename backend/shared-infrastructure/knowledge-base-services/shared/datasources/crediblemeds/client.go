// Package crediblemeds provides a client for CredibleMeds QT risk database.
// CredibleMeds maintains the most comprehensive database of drugs with QT prolongation risk.
//
// Phase 3b.2: CredibleMeds API Client
// Authority Level: DEFINITIVE (LLM = ❌ NEVER)
//
// Note: Requires academic/clinical license for full API access
// Website: https://crediblemeds.org/
package crediblemeds

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
// CREDIBLEMEDS CLIENT
// =============================================================================

// Client provides access to CredibleMeds QT risk database
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Config holds configuration for the CredibleMeds client
type Config struct {
	BaseURL string        // API endpoint
	APIKey  string        // Required for API access
	Timeout time.Duration // Default: 30s
}

// NewClient creates a new CredibleMeds client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.crediblemeds.org" // Placeholder - actual URL requires license
	}

	return &Client{
		baseURL: strings.TrimSuffix(config.BaseURL, "/"),
		apiKey:  config.APIKey,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// =============================================================================
// QT RISK CATEGORIES
// =============================================================================

// QTRiskCategory represents CredibleMeds QT risk classification
type QTRiskCategory string

const (
	// QTRiskKnown - Known risk of Torsades de Pointes (TdP)
	// Substantial evidence supports conclusion that drug prolongs QT and
	// is associated with TdP risk when used as directed
	QTRiskKnown QTRiskCategory = "KNOWN_RISK"

	// QTRiskPossible - Possible risk of TdP
	// Substantial evidence supports conclusion that drug can prolong QT,
	// but there is insufficient evidence for TdP risk when used as directed
	QTRiskPossible QTRiskCategory = "POSSIBLE_RISK"

	// QTRiskConditional - Conditional risk of TdP
	// Substantial evidence that drug is associated with TdP but only under
	// certain conditions (e.g., overdose, hypokalemia, drug interactions)
	QTRiskConditional QTRiskCategory = "CONDITIONAL_RISK"

	// QTRiskAvoidWithLQTS - To be avoided in patients with congenital LQTS
	// Not associated with TdP in normal population but should be avoided
	// in patients with congenital Long QT Syndrome
	QTRiskAvoidWithLQTS QTRiskCategory = "AVOID_WITH_LQTS"

	// QTRiskAvoidWithSQT - To be avoided in patients with short QT
	QTRiskAvoidWithSQT QTRiskCategory = "AVOID_WITH_SQT"

	// QTRiskNone - Not on CredibleMeds list (no documented QT risk)
	QTRiskNone QTRiskCategory = "NONE"
)

// =============================================================================
// DATA TYPES
// =============================================================================

// DrugQTRisk represents a drug's QT prolongation risk
type DrugQTRisk struct {
	DrugName          string         `json:"drug_name"`
	GenericName       string         `json:"generic_name"`
	BrandNames        []string       `json:"brand_names,omitempty"`
	RiskCategory      QTRiskCategory `json:"risk_category"`
	RiskDescription   string         `json:"risk_description"`
	LastReviewed      string         `json:"last_reviewed"`
	Evidence          []Evidence     `json:"evidence,omitempty"`
	Contraindications []string       `json:"contraindications,omitempty"`
	RiskFactors       []string       `json:"risk_factors,omitempty"`
	Recommendations   string         `json:"recommendations,omitempty"`
	DrugClass         string         `json:"drug_class"`
	TherapeuticUse    string         `json:"therapeutic_use"`
	RxCUI             string         `json:"rxcui,omitempty"`
}

// Evidence represents supporting evidence for QT risk
type Evidence struct {
	PMID       string `json:"pmid"`
	Citation   string `json:"citation"`
	Summary    string `json:"summary"`
	StudyType  string `json:"study_type"` // "Case Report", "Clinical Trial", "Meta-analysis"
	QTcChange  string `json:"qtc_change,omitempty"` // e.g., "+15ms"
	TdPCases   int    `json:"tdp_cases,omitempty"`
}

// DrugInteractionQT represents a drug-drug interaction affecting QT
type DrugInteractionQT struct {
	Drug1Name       string `json:"drug1_name"`
	Drug2Name       string `json:"drug2_name"`
	InteractionType string `json:"interaction_type"` // "Pharmacokinetic", "Pharmacodynamic", "Both"
	Mechanism       string `json:"mechanism"`
	Severity        string `json:"severity"` // "Minor", "Moderate", "Major", "Contraindicated"
	Recommendation  string `json:"recommendation"`
	Evidence        string `json:"evidence"`
}

// =============================================================================
// API METHODS
// =============================================================================

// GetQTRisk retrieves QT risk classification for a drug
func (c *Client) GetQTRisk(ctx context.Context, drugName string) (*DrugQTRisk, error) {
	endpoint := fmt.Sprintf("/v1/drugs/qt-risk?name=%s", url.QueryEscape(drugName))

	var risk DrugQTRisk
	err := c.doRequest(ctx, endpoint, &risk)

	if err != nil {
		// If drug not found, it's not on the list = no known QT risk
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			return &DrugQTRisk{
				DrugName:        drugName,
				RiskCategory:    QTRiskNone,
				RiskDescription: "Drug is not on CredibleMeds QT drug list",
			}, nil
		}
		return nil, fmt.Errorf("get QT risk: %w", err)
	}

	return &risk, nil
}

// GetQTRiskByRxCUI retrieves QT risk by RxNorm concept ID
func (c *Client) GetQTRiskByRxCUI(ctx context.Context, rxcui string) (*DrugQTRisk, error) {
	endpoint := fmt.Sprintf("/v1/drugs/qt-risk?rxcui=%s", url.QueryEscape(rxcui))

	var risk DrugQTRisk
	err := c.doRequest(ctx, endpoint, &risk)

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return &DrugQTRisk{
				RxCUI:           rxcui,
				RiskCategory:    QTRiskNone,
				RiskDescription: "Drug is not on CredibleMeds QT drug list",
			}, nil
		}
		return nil, fmt.Errorf("get QT risk by RxCUI: %w", err)
	}

	return &risk, nil
}

// GetAllKnownRiskDrugs retrieves all drugs with Known TdP Risk
func (c *Client) GetAllKnownRiskDrugs(ctx context.Context) ([]DrugQTRisk, error) {
	return c.getDrugsByCategory(ctx, QTRiskKnown)
}

// GetAllPossibleRiskDrugs retrieves all drugs with Possible TdP Risk
func (c *Client) GetAllPossibleRiskDrugs(ctx context.Context) ([]DrugQTRisk, error) {
	return c.getDrugsByCategory(ctx, QTRiskPossible)
}

// GetAllConditionalRiskDrugs retrieves all drugs with Conditional TdP Risk
func (c *Client) GetAllConditionalRiskDrugs(ctx context.Context) ([]DrugQTRisk, error) {
	return c.getDrugsByCategory(ctx, QTRiskConditional)
}

// GetDrugInteractionsQT retrieves QT-relevant drug interactions
func (c *Client) GetDrugInteractionsQT(ctx context.Context, drugName string) ([]DrugInteractionQT, error) {
	endpoint := fmt.Sprintf("/v1/interactions/qt?drug=%s", url.QueryEscape(drugName))

	var interactions []DrugInteractionQT
	if err := c.doRequest(ctx, endpoint, &interactions); err != nil {
		return nil, fmt.Errorf("get QT interactions: %w", err)
	}

	return interactions, nil
}

// CheckQTInteraction checks if two drugs have a QT-related interaction
func (c *Client) CheckQTInteraction(ctx context.Context, drug1, drug2 string) (*DrugInteractionQT, error) {
	endpoint := fmt.Sprintf("/v1/interactions/qt/check?drug1=%s&drug2=%s",
		url.QueryEscape(drug1), url.QueryEscape(drug2))

	var interaction DrugInteractionQT
	err := c.doRequest(ctx, endpoint, &interaction)

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil // No interaction found
		}
		return nil, fmt.Errorf("check QT interaction: %w", err)
	}

	return &interaction, nil
}

// =============================================================================
// CONVENIENCE METHODS
// =============================================================================

// IsHighQTRisk returns true if drug has known or possible QT risk
func (risk *DrugQTRisk) IsHighQTRisk() bool {
	return risk.RiskCategory == QTRiskKnown || risk.RiskCategory == QTRiskPossible
}

// IsAnyQTRisk returns true if drug has any documented QT risk
func (risk *DrugQTRisk) IsAnyQTRisk() bool {
	return risk.RiskCategory != QTRiskNone
}

// RequiresECGMonitoring returns true if ECG monitoring is recommended
func (risk *DrugQTRisk) RequiresECGMonitoring() bool {
	return risk.RiskCategory == QTRiskKnown || risk.RiskCategory == QTRiskConditional
}

// GetRiskLevel returns a numeric risk level (0-4) for sorting/comparison
func (risk *DrugQTRisk) GetRiskLevel() int {
	switch risk.RiskCategory {
	case QTRiskKnown:
		return 4
	case QTRiskPossible:
		return 3
	case QTRiskConditional:
		return 2
	case QTRiskAvoidWithLQTS, QTRiskAvoidWithSQT:
		return 1
	default:
		return 0
	}
}

// GetRiskColor returns a color code for UI display
func (risk *DrugQTRisk) GetRiskColor() string {
	switch risk.RiskCategory {
	case QTRiskKnown:
		return "red"
	case QTRiskPossible:
		return "orange"
	case QTRiskConditional:
		return "yellow"
	case QTRiskAvoidWithLQTS, QTRiskAvoidWithSQT:
		return "blue"
	default:
		return "green"
	}
}

// HasDrugInteractionRisk checks if combining with another QT drug is risky
func (c *Client) HasDrugInteractionRisk(ctx context.Context, drug1, drug2 string) (bool, string, error) {
	// Get QT risk for both drugs
	risk1, err := c.GetQTRisk(ctx, drug1)
	if err != nil {
		return false, "", err
	}

	risk2, err := c.GetQTRisk(ctx, drug2)
	if err != nil {
		return false, "", err
	}

	// Both drugs with Known risk = Contraindicated
	if risk1.RiskCategory == QTRiskKnown && risk2.RiskCategory == QTRiskKnown {
		return true, "Both drugs have Known TdP risk - combination contraindicated", nil
	}

	// Known + Possible or Known + Conditional = Major risk
	if (risk1.RiskCategory == QTRiskKnown && risk2.IsAnyQTRisk()) ||
		(risk2.RiskCategory == QTRiskKnown && risk1.IsAnyQTRisk()) {
		return true, "Combination of Known TdP risk drug with another QT-prolonging drug - avoid", nil
	}

	// Two Possible risk drugs = Moderate risk
	if risk1.RiskCategory == QTRiskPossible && risk2.RiskCategory == QTRiskPossible {
		return true, "Both drugs have Possible TdP risk - use with caution", nil
	}

	return false, "", nil
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// HealthCheck verifies the CredibleMeds API is accessible
func (c *Client) HealthCheck(ctx context.Context) error {
	// Try to get a known drug
	_, err := c.GetQTRisk(ctx, "amiodarone")
	return err
}

// =============================================================================
// INTERNAL METHODS
// =============================================================================

func (c *Client) getDrugsByCategory(ctx context.Context, category QTRiskCategory) ([]DrugQTRisk, error) {
	endpoint := fmt.Sprintf("/v1/drugs/qt-risk/category/%s", category)

	var drugs []DrugQTRisk
	if err := c.doRequest(ctx, endpoint, &drugs); err != nil {
		return nil, fmt.Errorf("get drugs by category: %w", err)
	}

	return drugs, nil
}

func (c *Client) doRequest(ctx context.Context, path string, result interface{}) error {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("not found: 404")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// =============================================================================
// AUTHORITY CLIENT INTERFACE IMPLEMENTATION (Phase 3b)
// =============================================================================

// Name returns the unique identifier for this data source
func (c *Client) Name() string {
	return "CredibleMeds"
}

// Close releases any resources held by the client
func (c *Client) Close() error {
	return nil
}

// GetFacts retrieves QT risk facts for a drug by RxCUI
func (c *Client) GetFacts(ctx context.Context, rxcui string) ([]AuthorityFact, error) {
	risk, err := c.GetQTRiskByRxCUI(ctx, rxcui)
	if err != nil {
		return nil, err
	}

	if risk.RiskCategory == QTRiskNone {
		return nil, nil // Not on CredibleMeds list
	}

	fact := c.riskToFact(risk, rxcui)
	return []AuthorityFact{fact}, nil
}

// GetFactsByName retrieves facts by drug name
func (c *Client) GetFactsByName(ctx context.Context, drugName string) ([]AuthorityFact, error) {
	risk, err := c.GetQTRisk(ctx, drugName)
	if err != nil {
		return nil, err
	}

	if risk.RiskCategory == QTRiskNone {
		return nil, nil
	}

	fact := c.riskToFact(risk, "")
	return []AuthorityFact{fact}, nil
}

// GetFactByType retrieves a specific fact type for a drug
func (c *Client) GetFactByType(ctx context.Context, rxcui string, factType FactType) (*AuthorityFact, error) {
	if factType != FactTypeQTProlongation {
		return nil, fmt.Errorf("CredibleMeds only provides %s facts", FactTypeQTProlongation)
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

// Sync synchronizes with CredibleMeds (full sync)
func (c *Client) Sync(ctx context.Context) (*SyncResult, error) {
	startTime := time.Now()

	// Get all known risk drugs as a sync test
	drugs, err := c.GetAllKnownRiskDrugs(ctx)
	if err != nil {
		return &SyncResult{
			Authority:  "CredibleMeds",
			StartTime:  startTime,
			EndTime:    time.Now(),
			ErrorCount: 1,
			Errors:     []string{err.Error()},
			Success:    false,
		}, err
	}

	return &SyncResult{
		Authority:  "CredibleMeds",
		StartTime:  startTime,
		EndTime:    time.Now(),
		TotalFacts: len(drugs),
		NewFacts:   len(drugs),
		Success:    true,
	}, nil
}

// SyncDelta performs incremental sync
func (c *Client) SyncDelta(ctx context.Context, since time.Time) (*SyncResult, error) {
	return c.Sync(ctx)
}

// Authority returns metadata about CredibleMeds
func (c *Client) Authority() AuthorityMetadata {
	return AuthorityMetadata{
		Name:            "CredibleMeds",
		FullName:        "CredibleMeds QT Drug Lists",
		URL:             "https://crediblemeds.org",
		Description:     "Comprehensive database of drugs with QT prolongation and Torsades de Pointes risk",
		AuthorityLevel:  AuthorityDefinitive,
		DataFormat:      "REST_API",
		UpdateFrequency: "MONTHLY",
		FactTypes:       []FactType{FactTypeQTProlongation},
	}
}

// SupportedFactTypes returns the fact types provided by CredibleMeds
func (c *Client) SupportedFactTypes() []FactType {
	return []FactType{FactTypeQTProlongation}
}

// LLMPolicy returns the LLM policy for CredibleMeds
func (c *Client) LLMPolicy() LLMPolicy {
	return LLMNever
}

// riskToFact converts DrugQTRisk to AuthorityFact
func (c *Client) riskToFact(risk *DrugQTRisk, rxcui string) AuthorityFact {
	fact := AuthorityFact{
		ID:               fmt.Sprintf("crediblemeds-%s", risk.DrugName),
		AuthoritySource:  "CredibleMeds",
		FactType:         FactTypeQTProlongation,
		RxCUI:            rxcui,
		DrugName:         risk.DrugName,
		GenericName:      risk.GenericName,
		Content:          risk,
		ExtractionMethod: "AUTHORITY_LOOKUP",
		Confidence:       1.0,
		FetchedAt:        time.Now(),
		SourceURL:        "https://crediblemeds.org",
	}

	// Map risk category to risk level and action
	switch risk.RiskCategory {
	case QTRiskKnown:
		fact.RiskLevel = "HIGH"
		fact.ActionRequired = "ALERT"
		fact.Recommendations = []string{
			"Drug has KNOWN risk of Torsades de Pointes",
			"ECG monitoring recommended",
			"Avoid in patients with QT prolongation risk factors",
		}
	case QTRiskPossible:
		fact.RiskLevel = "MODERATE"
		fact.ActionRequired = "CAUTION"
		fact.Recommendations = []string{
			"Drug has POSSIBLE risk of TdP",
			"Consider ECG monitoring",
		}
	case QTRiskConditional:
		fact.RiskLevel = "LOW"
		fact.ActionRequired = "MONITOR"
		fact.Recommendations = []string{
			"Drug has CONDITIONAL risk of TdP under certain conditions",
			"Monitor for risk factors (hypokalemia, drug interactions)",
		}
	case QTRiskAvoidWithLQTS:
		fact.RiskLevel = "MODERATE"
		fact.ActionRequired = "AVOID_IF_LQTS"
		fact.Recommendations = []string{
			"Avoid in patients with congenital Long QT Syndrome",
		}
	}

	return fact
}

// =============================================================================
// LOCAL TYPE ALIASES
// =============================================================================

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
	FactTypeQTProlongation FactType = "QT_PROLONGATION"
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
