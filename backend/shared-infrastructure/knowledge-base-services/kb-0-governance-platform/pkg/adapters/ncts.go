// Package adapters provides NCTS (National Clinical Terminology Service) adapter for Australian SNOMED CT.
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
// NCTS ADAPTER (AUSTRALIAN SNOMED CT)
// =============================================================================

// NCTSAdapter ingests Australian clinical terminology from the National Clinical Terminology Service.
// Used by KB-7 (Terminology) for Australian region.
// NCTS provides SNOMED CT-AU, AMT (Australian Medicines Terminology), and other Australian extensions.
type NCTSAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
	apiKey     string
	edition    string // SNOMED CT-AU edition
}

// NewNCTSAdapter creates a new NCTS adapter.
func NewNCTSAdapter(apiKey string) *NCTSAdapter {
	return &NCTSAdapter{
		BaseAdapter: NewBaseAdapter(
			"NCTS",
			models.AuthoritySNOMED,
			[]models.KB{models.KB7},
		),
		baseURL: "https://api.healthterminologies.gov.au",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey:  apiKey,
		edition: "SNOMED_CT_AU", // Australian edition
	}
}

// GetSupportedTypes returns the knowledge types this adapter can produce.
func (a *NCTSAdapter) GetSupportedTypes() []models.KnowledgeType {
	return []models.KnowledgeType{
		models.TypeTerminology,
		models.TypeValueSet,
	}
}

// FetchUpdates retrieves SNOMED CT-AU concepts updated since the given timestamp.
func (a *NCTSAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// NCTS provides a FHIR-based API for terminology
	searchURL := fmt.Sprintf("%s/fhir/CodeSystem/$lookup?system=http://snomed.info/sct&version=http://snomed.info/sct/32506021000036107",
		a.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	a.setHeaders(req)

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

// Transform converts raw NCTS content to a KnowledgeItem.
func (a *NCTSAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	var concept NCTSConcept
	if err := json.Unmarshal(raw.RawData, &concept); err != nil {
		return nil, fmt.Errorf("failed to parse NCTS concept: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb7:ncts:%s", concept.ConceptID),
		Type:    models.TypeTerminology,
		KB:      models.KB7,
		Version: concept.EffectiveTime,
		Name:    concept.FSN,
		Description: concept.PreferredTerm,
		Source: models.SourceAttribution{
			Authority:    models.AuthoritySNOMED,
			Document:     "SNOMED CT-AU",
			Section:      concept.Hierarchy,
			Jurisdiction: models.JurisdictionAU,
			URL:          fmt.Sprintf("https://healthterminologies.gov.au/fhir/CodeSystem/australian-snomed-ct-%s", concept.ConceptID),
		},
		ContentRef:  fmt.Sprintf("ncts:concept:%s", concept.ConceptID),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskLow,
		WorkflowTemplate: models.TemplateInfraLow,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs NCTS-specific validation.
func (a *NCTSAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthoritySNOMED {
		return fmt.Errorf("invalid authority: expected SNOMED, got %s", item.Source.Authority)
	}
	if item.Source.Jurisdiction != models.JurisdictionAU {
		return fmt.Errorf("NCTS content must be Australian jurisdiction")
	}
	return nil
}

// =============================================================================
// NCTS DATA STRUCTURES
// =============================================================================

// NCTSConcept represents an Australian SNOMED CT concept.
type NCTSConcept struct {
	ConceptID       string                `json:"conceptId"`
	Active          bool                  `json:"active"`
	EffectiveTime   string                `json:"effectiveTime"`
	ModuleID        string                `json:"moduleId"`
	ModuleName      string                `json:"moduleName,omitempty"`
	FSN             string                `json:"fsn"`              // Fully Specified Name
	PreferredTerm   string                `json:"pt"`               // Preferred Term (AU)
	Hierarchy       string                `json:"hierarchy,omitempty"`
	Descriptions    []NCTSDescription     `json:"descriptions,omitempty"`
	Relationships   []NCTSRelationship    `json:"relationships,omitempty"`
	AUExtension     *AUExtensionInfo      `json:"auExtension,omitempty"`
}

// NCTSDescription represents a concept description.
type NCTSDescription struct {
	DescriptionID  string            `json:"descriptionId"`
	Active         bool              `json:"active"`
	Term           string            `json:"term"`
	TypeID         string            `json:"typeId"`
	LanguageCode   string            `json:"languageCode"`
	AcceptabilityMap map[string]string `json:"acceptabilityMap"`
}

// NCTSRelationship represents a concept relationship.
type NCTSRelationship struct {
	RelationshipID string       `json:"relationshipId"`
	Active         bool         `json:"active"`
	TypeID         string       `json:"typeId"`
	TypeName       string       `json:"typeName,omitempty"`
	DestinationID  string       `json:"destinationId"`
	DestinationName string      `json:"destinationName,omitempty"`
}

// AUExtensionInfo contains Australian-specific extension information.
type AUExtensionInfo struct {
	IsAUExtension   bool     `json:"isAUExtension"`
	AMTConcept      bool     `json:"amtConcept,omitempty"` // Australian Medicines Terminology
	CTYPEExtension  bool     `json:"ctypeExtension,omitempty"`
	PBSItem         bool     `json:"pbsItem,omitempty"` // Pharmaceutical Benefits Scheme
	AUSpecificTerms []string `json:"auSpecificTerms,omitempty"`
}

// =============================================================================
// NCTS TERMINOLOGY OPERATIONS
// =============================================================================

// LookupConcept retrieves a SNOMED CT-AU concept by ID.
func (a *NCTSAdapter) LookupConcept(ctx context.Context, conceptID string) (*NCTSConcept, error) {
	url := fmt.Sprintf("%s/fhir/CodeSystem/$lookup?system=http://snomed.info/sct&code=%s&version=http://snomed.info/sct/32506021000036107",
		a.baseURL, conceptID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	a.setHeaders(req)

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

	// Parse FHIR Parameters response
	return a.parseFHIRLookupResponse(body)
}

// SearchConcepts searches for SNOMED CT-AU concepts by term.
func (a *NCTSAdapter) SearchConcepts(ctx context.Context, term string, limit int) ([]*NCTSConcept, error) {
	searchURL := fmt.Sprintf("%s/fhir/ValueSet/$expand?url=http://snomed.info/sct?fhir_vs&filter=%s&count=%d",
		a.baseURL, url.QueryEscape(term), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return a.parseFHIRExpandResponse(body)
}

// GetAUExtensions retrieves Australian-specific extensions for a concept.
func (a *NCTSAdapter) GetAUExtensions(ctx context.Context, conceptID string) (*AUExtensionInfo, error) {
	// Australian module IDs
	auModuleID := "32506021000036107" // SNOMED CT-AU

	concept, err := a.LookupConcept(ctx, conceptID)
	if err != nil {
		return nil, err
	}
	if concept == nil {
		return nil, nil
	}

	ext := &AUExtensionInfo{
		IsAUExtension: concept.ModuleID == auModuleID,
	}

	// Check for AMT (Australian Medicines Terminology)
	if isAMTConcept(concept) {
		ext.AMTConcept = true
	}

	return ext, nil
}

// =============================================================================
// AUSTRALIAN MEDICINES TERMINOLOGY (AMT)
// =============================================================================

// AMT Module ID
const AMTModuleID = "900062011000036108"

// AMTConcept represents an Australian Medicines Terminology concept.
type AMTConcept struct {
	NCTSConcept
	AMTType    AMTConceptType `json:"amtType"`
	TradeProduct *AMTTradeProduct `json:"tradeProduct,omitempty"`
	MPP        *AMTMPPInfo    `json:"mpp,omitempty"` // Medicinal Product Pack
	MPUU       *AMTMPUUInfo   `json:"mpuu,omitempty"` // Medicinal Product Unit of Use
	TPUU       *AMTTPUUInfo   `json:"tpuu,omitempty"` // Trade Product Unit of Use
}

// AMTConceptType represents AMT concept types.
type AMTConceptType string

const (
	AMTMP      AMTConceptType = "MP"      // Medicinal Product
	AMTMPUU    AMTConceptType = "MPUU"    // Medicinal Product Unit of Use
	AMTMPP     AMTConceptType = "MPP"     // Medicinal Product Pack
	AMTTP      AMTConceptType = "TP"      // Trade Product
	AMTTPUU    AMTConceptType = "TPUU"    // Trade Product Unit of Use
	AMTTPP     AMTConceptType = "TPP"     // Trade Product Pack
	AMTCTPP    AMTConceptType = "CTPP"    // Containered Trade Product Pack
)

// AMTTradeProduct contains trade product information.
type AMTTradeProduct struct {
	BrandName    string `json:"brandName"`
	Manufacturer string `json:"manufacturer,omitempty"`
	TGA_PI_ID    string `json:"tgaPiId,omitempty"` // TGA Product Information ID
}

// AMTMPPInfo contains Medicinal Product Pack information.
type AMTMPPInfo struct {
	PackSize       string `json:"packSize"`
	UnitOfMeasure  string `json:"unitOfMeasure"`
	ContainerType  string `json:"containerType,omitempty"`
}

// AMTMPUUInfo contains Medicinal Product Unit of Use information.
type AMTMPUUInfo struct {
	Strength       string `json:"strength"`
	UnitOfMeasure  string `json:"unitOfMeasure"`
	DoseForm       string `json:"doseForm"`
}

// AMTTPUUInfo contains Trade Product Unit of Use information.
type AMTTPUUInfo struct {
	BrandName      string `json:"brandName"`
	Strength       string `json:"strength"`
	UnitOfMeasure  string `json:"unitOfMeasure"`
	DoseForm       string `json:"doseForm"`
}

// isAMTConcept checks if a concept is an AMT concept.
func isAMTConcept(concept *NCTSConcept) bool {
	return concept.ModuleID == AMTModuleID ||
		(concept.AUExtension != nil && concept.AUExtension.AMTConcept)
}

// GetAMTConcept retrieves an AMT concept with full details.
func (a *NCTSAdapter) GetAMTConcept(ctx context.Context, conceptID string) (*AMTConcept, error) {
	// Fetch base concept
	concept, err := a.LookupConcept(ctx, conceptID)
	if err != nil {
		return nil, err
	}
	if concept == nil {
		return nil, nil
	}

	if !isAMTConcept(concept) {
		return nil, fmt.Errorf("concept %s is not an AMT concept", conceptID)
	}

	amtConcept := &AMTConcept{
		NCTSConcept: *concept,
	}

	// Determine AMT type from hierarchy
	amtConcept.AMTType = a.determineAMTType(concept)

	return amtConcept, nil
}

// determineAMTType determines the AMT concept type.
func (a *NCTSAdapter) determineAMTType(concept *NCTSConcept) AMTConceptType {
	fsn := concept.FSN

	switch {
	case contains(fsn, "trade product pack"):
		if contains(fsn, "containered") {
			return AMTCTPP
		}
		return AMTTPP
	case contains(fsn, "trade product unit of use"):
		return AMTTPUU
	case contains(fsn, "trade product"):
		return AMTTP
	case contains(fsn, "medicinal product pack"):
		return AMTMPP
	case contains(fsn, "medicinal product unit of use"):
		return AMTMPUU
	case contains(fsn, "medicinal product"):
		return AMTMP
	default:
		return AMTMP
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	// Simple case-insensitive contains
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldSlice(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFoldSlice(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := 0; i < len(s); i++ {
		if toLower(s[i]) != toLower(t[i]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// =============================================================================
// PBS (PHARMACEUTICAL BENEFITS SCHEME) INTEGRATION
// =============================================================================

// PBSItem represents a PBS listed item.
type PBSItem struct {
	PBSCode        string  `json:"pbsCode"`
	ItemCode       string  `json:"itemCode"`
	DrugName       string  `json:"drugName"`
	FormStrength   string  `json:"formStrength"`
	PackSize       int     `json:"packSize"`
	Manufacturer   string  `json:"manufacturer"`
	MaxQuantity    int     `json:"maxQuantity"`
	NumRepeats     int     `json:"numRepeats"`
	RestrictionFlag string `json:"restrictionFlag,omitempty"` // R = Restricted, U = Unrestricted
	Schedule       string  `json:"schedule"` // General, CTG, RPBS
	AMTConceptID   string  `json:"amtConceptId,omitempty"` // Linked AMT concept
}

// GetPBSMappedConcepts retrieves AMT concepts mapped to PBS items.
func (a *NCTSAdapter) GetPBSMappedConcepts(ctx context.Context, pbsCode string) ([]*AMTConcept, error) {
	// NCTS provides PBS to AMT mappings
	url := fmt.Sprintf("%s/fhir/ConceptMap/$translate?system=http://pbs.gov.au&code=%s&target=http://snomed.info/sct",
		a.baseURL, pbsCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse FHIR ConceptMap translate response
	// In production, would parse the FHIR response properly
	return nil, nil
}

// =============================================================================
// FHIR RESPONSE PARSING
// =============================================================================

func (a *NCTSAdapter) parseFHIRLookupResponse(data []byte) (*NCTSConcept, error) {
	var params struct {
		ResourceType string `json:"resourceType"`
		Parameter    []struct {
			Name       string `json:"name"`
			ValueString string `json:"valueString,omitempty"`
			ValueCode  string `json:"valueCode,omitempty"`
		} `json:"parameter"`
	}

	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("failed to parse FHIR response: %w", err)
	}

	concept := &NCTSConcept{
		Active: true,
	}

	for _, p := range params.Parameter {
		switch p.Name {
		case "code":
			concept.ConceptID = p.ValueCode
		case "display":
			concept.PreferredTerm = p.ValueString
		case "name":
			concept.FSN = p.ValueString
		case "version":
			concept.EffectiveTime = p.ValueString
		}
	}

	return concept, nil
}

func (a *NCTSAdapter) parseFHIRExpandResponse(data []byte) ([]*NCTSConcept, error) {
	var valueSet struct {
		ResourceType string `json:"resourceType"`
		Expansion    struct {
			Contains []struct {
				System  string `json:"system"`
				Code    string `json:"code"`
				Display string `json:"display"`
			} `json:"contains"`
		} `json:"expansion"`
	}

	if err := json.Unmarshal(data, &valueSet); err != nil {
		return nil, fmt.Errorf("failed to parse FHIR response: %w", err)
	}

	var concepts []*NCTSConcept
	for _, c := range valueSet.Expansion.Contains {
		concepts = append(concepts, &NCTSConcept{
			ConceptID:     c.Code,
			PreferredTerm: c.Display,
			Active:        true,
		})
	}

	return concepts, nil
}

// setHeaders sets required headers for NCTS API requests.
func (a *NCTSAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/fhir+json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
	}
}

// =============================================================================
// NCTS VALUE SETS
// =============================================================================

// NCTSValueSet represents an Australian value set.
type NCTSValueSet struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	URL          string   `json:"url"`
	Version      string   `json:"version"`
	Status       string   `json:"status"` // active, draft, retired
	Publisher    string   `json:"publisher"`
	Concepts     []string `json:"concepts"` // SNOMED CT concept IDs
}

// Common Australian Value Sets
var NCTSValueSets = map[string]NCTSValueSet{
	"australian-indigenous-status": {
		ID:    "australian-indigenous-status",
		Name:  "AustralianIndigenousStatus",
		Title: "Australian Indigenous Status",
		URL:   "https://healthterminologies.gov.au/fhir/ValueSet/australian-indigenous-status-1",
	},
	"australian-states-territories": {
		ID:    "australian-states-territories",
		Name:  "AustralianStatesAndTerritories",
		Title: "Australian States and Territories",
		URL:   "https://healthterminologies.gov.au/fhir/ValueSet/australian-states-territories-1",
	},
	"amt-mp": {
		ID:    "amt-mp",
		Name:  "AMTMedicinalProduct",
		Title: "AMT Medicinal Products",
		URL:   "https://healthterminologies.gov.au/fhir/ValueSet/amt-mp",
	},
	"amt-tpp": {
		ID:    "amt-tpp",
		Name:  "AMTTradeProductPack",
		Title: "AMT Trade Product Packs",
		URL:   "https://healthterminologies.gov.au/fhir/ValueSet/amt-tpp",
	},
}

// GetValueSet retrieves an NCTS value set by ID.
func (a *NCTSAdapter) GetValueSet(ctx context.Context, valueSetID string) (*NCTSValueSet, error) {
	url := fmt.Sprintf("%s/fhir/ValueSet/%s", a.baseURL, valueSetID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var vs NCTSValueSet
	if err := json.NewDecoder(resp.Body).Decode(&vs); err != nil {
		return nil, err
	}

	return &vs, nil
}
