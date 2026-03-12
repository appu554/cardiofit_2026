// Package adapters provides EMA (European Medicines Agency) SmPC adapter.
package adapters

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// EMA SMPC ADAPTER (EUROPEAN UNION)
// =============================================================================

// EMASmPCAdapter ingests drug information from EMA's Summary of Product Characteristics.
// Used by KB-1 (Drug Dosing), KB-4 (Patient Safety), KB-5 (Drug Interactions).
// SmPC is the EU equivalent of FDA's drug labeling.
type EMASmPCAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
}

// NewEMASmPCAdapter creates a new EMA SmPC adapter.
func NewEMASmPCAdapter() *EMASmPCAdapter {
	return &EMASmPCAdapter{
		BaseAdapter: NewBaseAdapter(
			"EMA_SMPC",
			models.AuthorityEMA,
			[]models.KB{models.KB1, models.KB4, models.KB5},
		),
		baseURL: "https://www.ema.europa.eu/en/medicines",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetSupportedTypes returns the knowledge types this adapter can produce.
func (a *EMASmPCAdapter) GetSupportedTypes() []models.KnowledgeType {
	return []models.KnowledgeType{
		models.TypeDosingRule,
		models.TypeSafetyAlert,
		models.TypeInteraction,
	}
}

// FetchUpdates retrieves SmPCs updated since the given timestamp.
func (a *EMASmPCAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// EMA provides a medicines database with search functionality
	// In production, this would use the EMA medicines API or scrape the database
	searchURL := fmt.Sprintf("%s/api/v1/medicines?updated_after=%s",
		a.baseURL, since.Format("2006-01-02"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
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
	// Parse response and return items
	return items, nil
}

// Transform converts raw EMA SmPC content to a KnowledgeItem.
func (a *EMASmPCAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	smpc, err := a.parseSmPC(raw.RawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SmPC: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb1:ema:%s:%s", smpc.ProductNumber, smpc.Version),
		Type:    models.TypeDosingRule,
		KB:      models.KB1,
		Version: smpc.Version,
		Name:    smpc.ProductName,
		Description: fmt.Sprintf("SmPC for %s (%s)", smpc.ProductName, smpc.ActiveSubstance),
		Source: models.SourceAttribution{
			Authority:     models.AuthorityEMA,
			Document:      "Summary of Product Characteristics",
			Section:       "4.2 Posology and method of administration",
			Jurisdiction:  models.JurisdictionEU,
			URL:           smpc.URL,
			EffectiveDate: smpc.AuthorisationDate,
		},
		ContentRef:  fmt.Sprintf("ema:smpc:%s", smpc.ProductNumber),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskHigh,
		WorkflowTemplate: models.TemplateClinicalHigh,
		RequiresDualReview: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Set risk flags based on SmPC content
	if smpc.HasBlackTriangle {
		item.RiskFlags.HighAlertDrug = true
	}
	if smpc.IsControlled {
		item.RiskFlags.ControlledSubstance = true
	}

	return item, nil
}

// Validate performs EMA-specific validation.
func (a *EMASmPCAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthorityEMA {
		return fmt.Errorf("invalid authority: expected EMA, got %s", item.Source.Authority)
	}

	if item.Source.Jurisdiction != models.JurisdictionEU {
		return fmt.Errorf("invalid jurisdiction for EMA: expected EU, got %s", item.Source.Jurisdiction)
	}

	return nil
}

// =============================================================================
// EMA SMPC DATA STRUCTURES
// =============================================================================

// EMASmPC represents a parsed Summary of Product Characteristics.
type EMASmPC struct {
	ProductNumber     string `json:"product_number"`
	ProductName       string `json:"product_name"`
	ActiveSubstance   string `json:"active_substance"`
	ATCCode           string `json:"atc_code"`
	TherapeuticArea   string `json:"therapeutic_area"`
	AuthorisationDate string `json:"authorisation_date"`
	Version           string `json:"version"`
	URL               string `json:"url"`
	HasBlackTriangle  bool   `json:"has_black_triangle"` // Additional monitoring required
	IsControlled      bool   `json:"is_controlled"`
	IsOrphan          bool   `json:"is_orphan"`
	IsBiosimilar      bool   `json:"is_biosimilar"`

	// SmPC Sections (EU standard numbering)
	Sections          map[string]SmPCSection `json:"sections"`
}

// SmPCSection represents a section of the SmPC.
type SmPCSection struct {
	Number  string `json:"number"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// SmPC Section Numbers (EU Standard)
const (
	SmPCSectionName                     = "1"
	SmPCSectionComposition              = "2"
	SmPCSectionForm                     = "3"
	SmPCSectionClinicalParticulars      = "4"
	SmPCSectionTherapeuticIndications   = "4.1"
	SmPCSectionPosology                 = "4.2"  // Dosing information
	SmPCSectionContraindications        = "4.3"
	SmPCSectionWarnings                 = "4.4"
	SmPCSectionInteractions             = "4.5"  // Drug interactions
	SmPCSectionPregnancy                = "4.6"
	SmPCSectionDriving                  = "4.7"
	SmPCSectionUndesirableEffects       = "4.8"
	SmPCSectionOverdose                 = "4.9"
	SmPCSectionPharmacologicalProperties = "5"
	SmPCSectionPharmaceuticalParticulars = "6"
)

// parseSmPC parses SmPC content from various formats.
func (a *EMASmPCAdapter) parseSmPC(data []byte) (*EMASmPC, error) {
	// Try JSON first
	var smpc EMASmPC
	if err := json.Unmarshal(data, &smpc); err == nil {
		return &smpc, nil
	}

	// Try XML format (EMA ePI format)
	var xmlSmPC emaXMLSmPC
	if err := xml.Unmarshal(data, &xmlSmPC); err == nil {
		return a.convertXMLToSmPC(&xmlSmPC), nil
	}

	return nil, fmt.Errorf("unable to parse SmPC data")
}

// emaXMLSmPC represents XML-formatted SmPC from EMA.
type emaXMLSmPC struct {
	XMLName xml.Name `xml:"SmPC"`
	Product struct {
		Name            string `xml:"name"`
		Number          string `xml:"product_number"`
		ActiveSubstance string `xml:"active_substance"`
		ATCCode         string `xml:"atc_code"`
	} `xml:"product"`
	Sections []struct {
		Number  string `xml:"number,attr"`
		Title   string `xml:"title"`
		Content string `xml:",innerxml"`
	} `xml:"section"`
}

func (a *EMASmPCAdapter) convertXMLToSmPC(xmlSmPC *emaXMLSmPC) *EMASmPC {
	smpc := &EMASmPC{
		ProductNumber:   xmlSmPC.Product.Number,
		ProductName:     xmlSmPC.Product.Name,
		ActiveSubstance: xmlSmPC.Product.ActiveSubstance,
		ATCCode:         xmlSmPC.Product.ATCCode,
		Sections:        make(map[string]SmPCSection),
	}

	for _, sec := range xmlSmPC.Sections {
		smpc.Sections[sec.Number] = SmPCSection{
			Number:  sec.Number,
			Title:   sec.Title,
			Content: sec.Content,
		}
	}

	return smpc
}

// =============================================================================
// EMA MEDICINES DATABASE SEARCH
// =============================================================================

// SearchMedicines searches EMA medicines database.
func (a *EMASmPCAdapter) SearchMedicines(ctx context.Context, query string, limit int) ([]*EMAMedicineEntry, error) {
	searchURL := fmt.Sprintf("%s/api/v1/search?q=%s&limit=%d",
		a.baseURL, url.QueryEscape(query), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
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

	var result EMASearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Medicines, nil
}

// EMASearchResult represents search results from EMA.
type EMASearchResult struct {
	Total     int                  `json:"total"`
	Medicines []*EMAMedicineEntry  `json:"medicines"`
}

// EMAMedicineEntry represents a medicine in the EMA database.
type EMAMedicineEntry struct {
	ProductNumber     string   `json:"product_number"`
	ProductName       string   `json:"product_name"`
	ActiveSubstance   string   `json:"active_substance"`
	ATCCode           string   `json:"atc_code"`
	TherapeuticArea   string   `json:"therapeutic_area"`
	AuthorisationStatus string `json:"authorisation_status"` // Authorised, Withdrawn, etc.
	AuthorisationType string   `json:"authorisation_type"`   // Centralised, Decentralised
	AuthorisationDate string   `json:"authorisation_date"`
	MarketingAuthorisationHolder string `json:"mah"`
	HasBlackTriangle  bool     `json:"additional_monitoring"`
	IsOrphan          bool     `json:"orphan"`
	IsBiosimilar      bool     `json:"biosimilar"`
	IsGeneric         bool     `json:"generic"`
	SmPCURL           string   `json:"smpc_url"`
	PILUrl            string   `json:"pil_url"` // Patient Information Leaflet
	EPARUrl           string   `json:"epar_url"` // European Public Assessment Report
}

// GetSmPCDocument fetches the SmPC document for a medicine.
func (a *EMASmPCAdapter) GetSmPCDocument(ctx context.Context, productNumber string) ([]byte, error) {
	docURL := fmt.Sprintf("%s/%s/smpc", a.baseURL, productNumber)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, docURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/pdf, application/xml, application/json")

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

	return io.ReadAll(resp.Body)
}

// =============================================================================
// EMA POSOLOGY EXTRACTION
// =============================================================================

// ExtractPosology extracts dosing information from SmPC Section 4.2.
func (a *EMASmPCAdapter) ExtractPosology(smpc *EMASmPC) (*EMAPosology, error) {
	section, ok := smpc.Sections[SmPCSectionPosology]
	if !ok {
		return nil, fmt.Errorf("posology section (4.2) not found")
	}

	posology := &EMAPosology{
		RawText: section.Content,
	}

	// Parse structured dosing information
	posology.Adults = a.extractAdultDosing(section.Content)
	posology.Paediatric = a.extractPaediatricDosing(section.Content)
	posology.Elderly = a.extractElderlyDosing(section.Content)
	posology.RenalImpairment = a.extractRenalDosing(section.Content)
	posology.HepaticImpairment = a.extractHepaticDosing(section.Content)

	return posology, nil
}

// EMAPosology represents extracted posology information.
type EMAPosology struct {
	RawText           string              `json:"raw_text"`
	Adults            *EMADosingRegimen   `json:"adults,omitempty"`
	Paediatric        *EMADosingRegimen   `json:"paediatric,omitempty"`
	Elderly           *EMADosingRegimen   `json:"elderly,omitempty"`
	RenalImpairment   []EMADoseAdjustment `json:"renal_impairment,omitempty"`
	HepaticImpairment []EMADoseAdjustment `json:"hepatic_impairment,omitempty"`
}

// EMADosingRegimen represents a dosing regimen.
type EMADosingRegimen struct {
	InitialDose      string `json:"initial_dose,omitempty"`
	MaintenanceDose  string `json:"maintenance_dose,omitempty"`
	MaximumDose      string `json:"maximum_dose,omitempty"`
	Frequency        string `json:"frequency,omitempty"`
	Route            string `json:"route,omitempty"`
	Duration         string `json:"duration,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

// EMADoseAdjustment represents a dose adjustment for special populations.
type EMADoseAdjustment struct {
	Condition     string `json:"condition"`
	Severity      string `json:"severity,omitempty"` // Mild, Moderate, Severe
	Recommendation string `json:"recommendation"`
	Notes         string `json:"notes,omitempty"`
}

func (a *EMASmPCAdapter) extractAdultDosing(content string) *EMADosingRegimen {
	// Extract adult dosing from content using pattern matching
	// In production, would use NLP or structured parsing
	if strings.Contains(strings.ToLower(content), "adult") {
		return &EMADosingRegimen{
			Notes: "Adult dosing information present",
		}
	}
	return nil
}

func (a *EMASmPCAdapter) extractPaediatricDosing(content string) *EMADosingRegimen {
	if strings.Contains(strings.ToLower(content), "paediatric") ||
	   strings.Contains(strings.ToLower(content), "children") {
		return &EMADosingRegimen{
			Notes: "Paediatric dosing information present",
		}
	}
	return nil
}

func (a *EMASmPCAdapter) extractElderlyDosing(content string) *EMADosingRegimen {
	if strings.Contains(strings.ToLower(content), "elderly") {
		return &EMADosingRegimen{
			Notes: "Elderly dosing information present",
		}
	}
	return nil
}

func (a *EMASmPCAdapter) extractRenalDosing(content string) []EMADoseAdjustment {
	var adjustments []EMADoseAdjustment
	if strings.Contains(strings.ToLower(content), "renal") {
		adjustments = append(adjustments, EMADoseAdjustment{
			Condition: "Renal impairment",
			Notes:     "Renal adjustment information present",
		})
	}
	return adjustments
}

func (a *EMASmPCAdapter) extractHepaticDosing(content string) []EMADoseAdjustment {
	var adjustments []EMADoseAdjustment
	if strings.Contains(strings.ToLower(content), "hepatic") {
		adjustments = append(adjustments, EMADoseAdjustment{
			Condition: "Hepatic impairment",
			Notes:     "Hepatic adjustment information present",
		})
	}
	return adjustments
}

// =============================================================================
// EMA INTERACTIONS EXTRACTION
// =============================================================================

// ExtractInteractions extracts drug interaction information from SmPC Section 4.5.
func (a *EMASmPCAdapter) ExtractInteractions(smpc *EMASmPC) (*EMAInteractions, error) {
	section, ok := smpc.Sections[SmPCSectionInteractions]
	if !ok {
		return nil, fmt.Errorf("interactions section (4.5) not found")
	}

	interactions := &EMAInteractions{
		RawText:          section.Content,
		Contraindicated:  a.extractContraindicatedInteractions(section.Content),
		NotRecommended:   a.extractNotRecommendedInteractions(section.Content),
		Caution:          a.extractCautionInteractions(section.Content),
	}

	return interactions, nil
}

// EMAInteractions represents extracted drug interactions.
type EMAInteractions struct {
	RawText         string            `json:"raw_text"`
	Contraindicated []EMAInteraction  `json:"contraindicated,omitempty"`
	NotRecommended  []EMAInteraction  `json:"not_recommended,omitempty"`
	Caution         []EMAInteraction  `json:"caution,omitempty"`
}

// EMAInteraction represents a single drug interaction.
type EMAInteraction struct {
	InteractingDrug string `json:"interacting_drug"`
	DrugClass       string `json:"drug_class,omitempty"`
	Mechanism       string `json:"mechanism,omitempty"`
	Effect          string `json:"effect"`
	Severity        string `json:"severity"` // Contraindicated, Not Recommended, Caution
	Management      string `json:"management,omitempty"`
}

func (a *EMASmPCAdapter) extractContraindicatedInteractions(content string) []EMAInteraction {
	var interactions []EMAInteraction
	if strings.Contains(strings.ToLower(content), "contraindicated") {
		interactions = append(interactions, EMAInteraction{
			Severity: "Contraindicated",
			Effect:   "Concomitant use is contraindicated",
		})
	}
	return interactions
}

func (a *EMASmPCAdapter) extractNotRecommendedInteractions(content string) []EMAInteraction {
	var interactions []EMAInteraction
	if strings.Contains(strings.ToLower(content), "not recommended") {
		interactions = append(interactions, EMAInteraction{
			Severity: "Not Recommended",
			Effect:   "Concomitant use is not recommended",
		})
	}
	return interactions
}

func (a *EMASmPCAdapter) extractCautionInteractions(content string) []EMAInteraction {
	var interactions []EMAInteraction
	if strings.Contains(strings.ToLower(content), "caution") {
		interactions = append(interactions, EMAInteraction{
			Severity: "Caution",
			Effect:   "Use with caution",
		})
	}
	return interactions
}

// =============================================================================
// EMA ATC CODE HELPERS
// =============================================================================

// ATCLevel represents ATC classification levels.
type ATCLevel int

const (
	ATCLevelAnatomical      ATCLevel = 1 // 1st level: Anatomical main group
	ATCLevelTherapeutic     ATCLevel = 2 // 2nd level: Therapeutic subgroup
	ATCLevelPharmacological ATCLevel = 3 // 3rd level: Pharmacological subgroup
	ATCLevelChemical        ATCLevel = 4 // 4th level: Chemical subgroup
	ATCLevelSubstance       ATCLevel = 5 // 5th level: Chemical substance
)

// GetATCLevel returns the ATC code at the specified level.
func GetATCLevel(atcCode string, level ATCLevel) string {
	switch level {
	case ATCLevelAnatomical:
		if len(atcCode) >= 1 {
			return atcCode[:1]
		}
	case ATCLevelTherapeutic:
		if len(atcCode) >= 3 {
			return atcCode[:3]
		}
	case ATCLevelPharmacological:
		if len(atcCode) >= 4 {
			return atcCode[:4]
		}
	case ATCLevelChemical:
		if len(atcCode) >= 5 {
			return atcCode[:5]
		}
	case ATCLevelSubstance:
		return atcCode
	}
	return atcCode
}

// ATCAnatomicalGroups maps ATC first-level codes to anatomical groups.
var ATCAnatomicalGroups = map[string]string{
	"A": "Alimentary tract and metabolism",
	"B": "Blood and blood forming organs",
	"C": "Cardiovascular system",
	"D": "Dermatologicals",
	"G": "Genito-urinary system and sex hormones",
	"H": "Systemic hormonal preparations",
	"J": "Antiinfectives for systemic use",
	"L": "Antineoplastic and immunomodulating agents",
	"M": "Musculo-skeletal system",
	"N": "Nervous system",
	"P": "Antiparasitic products",
	"R": "Respiratory system",
	"S": "Sensory organs",
	"V": "Various",
}
