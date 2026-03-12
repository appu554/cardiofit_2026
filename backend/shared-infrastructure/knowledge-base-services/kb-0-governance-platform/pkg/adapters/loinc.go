// Package adapters provides LOINC (Logical Observation Identifiers Names and Codes) adapter.
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
// LOINC ADAPTER (GLOBAL)
// =============================================================================

// LOINCAdapter ingests laboratory and clinical observation codes from LOINC.
// Used by KB-7 (Terminology), KB-16 (Lab Reference Ranges).
type LOINCAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewLOINCAdapter creates a new LOINC adapter.
func NewLOINCAdapter(apiKey string) *LOINCAdapter {
	return &LOINCAdapter{
		BaseAdapter: NewBaseAdapter(
			"LOINC",
			models.AuthorityRegenstrief,
			[]models.KB{models.KB7, models.KB16},
		),
		baseURL: "https://fhir.loinc.org",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey: apiKey,
	}
}

// FetchUpdates retrieves LOINC codes updated since the given timestamp.
func (a *LOINCAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// LOINC FHIR API - fetch updated codes
	loincURL := fmt.Sprintf("%s/CodeSystem/$lookup?system=http://loinc.org&version=2.77",
		a.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loincURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Basic "+a.apiKey)
	}

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

// Transform converts raw LOINC content to a KnowledgeItem.
func (a *LOINCAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	code, err := a.parseCode(raw.RawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LOINC code: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb7:loinc:%s", code.Code),
		Type:    models.TypeTerminology,
		KB:      models.KB7,
		Version: code.Version,
		Name:    code.LongCommonName,
		Description: code.ShortName,
		Source: models.SourceAttribution{
			Authority:    models.AuthorityRegenstrief,
			Document:     "LOINC",
			Section:      code.Class,
			Jurisdiction: models.JurisdictionGlobal,
			URL:          fmt.Sprintf("https://loinc.org/%s", code.Code),
		},
		ContentRef:  fmt.Sprintf("loinc:code:%s", code.Code),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskLow,
		WorkflowTemplate: models.TemplateInfraLow,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs LOINC-specific validation.
func (a *LOINCAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthorityRegenstrief {
		return fmt.Errorf("invalid authority: expected Regenstrief, got %s", item.Source.Authority)
	}

	// Additional LOINC-specific validation:
	// - Valid LOINC code format (numeric with check digit)
	// - Active code status
	// - Has component and property

	return nil
}

// =============================================================================
// LOINC CODE LOOKUP
// =============================================================================

// LookupCode retrieves a single LOINC code.
func (a *LOINCAdapter) LookupCode(ctx context.Context, loincCode string) (*LOINCCode, error) {
	lookupURL := fmt.Sprintf("%s/CodeSystem/$lookup?system=http://loinc.org&code=%s",
		a.baseURL, loincCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lookupURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Basic "+a.apiKey)
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

	return a.parseCode(body)
}

// SearchCodes searches for LOINC codes by term.
func (a *LOINCAdapter) SearchCodes(ctx context.Context, term string, limit int) ([]*LOINCCode, error) {
	searchURL := fmt.Sprintf("%s/CodeSystem/$lookup?system=http://loinc.org&displayLanguage=en&_count=%d&filter=%s",
		a.baseURL, limit, url.QueryEscape(term))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Basic "+a.apiKey)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse search results
	var codes []*LOINCCode
	return codes, nil
}

// GetCodesByClass retrieves LOINC codes by class.
func (a *LOINCAdapter) GetCodesByClass(ctx context.Context, class string, limit int) ([]*LOINCCode, error) {
	classURL := fmt.Sprintf("%s/ValueSet/$expand?url=http://loinc.org/vs/LL%s&count=%d",
		a.baseURL, class, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, classURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Basic "+a.apiKey)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var codes []*LOINCCode
	return codes, nil
}

// =============================================================================
// LOINC STRUCTURES
// =============================================================================

// LOINCCode represents a LOINC code.
type LOINCCode struct {
	Code            string `json:"code"`
	LongCommonName  string `json:"long_common_name"`
	ShortName       string `json:"short_name"`
	Component       string `json:"component"`
	Property        string `json:"property"`
	TimeAspect      string `json:"time_aspect"`
	System          string `json:"system"`
	Scale           string `json:"scale"`
	Method          string `json:"method,omitempty"`
	Class           string `json:"class"`
	ClassType       string `json:"class_type"`
	Status          string `json:"status"`
	Version         string `json:"version"`
	UnitsRequired   string `json:"units_required,omitempty"`
	SubmittedUnits  string `json:"submitted_units,omitempty"`
	ExampleUnits    string `json:"example_units,omitempty"`
	OrderObs        string `json:"order_obs"`         // Order, Observation, Both
	RelatedNames    []string `json:"related_names,omitempty"`
	Panels          []string `json:"panels,omitempty"`
}

// LOINCPanel represents a LOINC panel (group of related codes).
type LOINCPanel struct {
	PanelCode     string       `json:"panel_code"`
	PanelName     string       `json:"panel_name"`
	PanelType     string       `json:"panel_type"`
	Members       []PanelMember `json:"members"`
	SequenceNum   int          `json:"sequence_num,omitempty"`
}

// PanelMember represents a member of a LOINC panel.
type PanelMember struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Cardinality  string `json:"cardinality"`  // R (required), O (optional), C (conditional)
	SequenceNum  int    `json:"sequence_num"`
	Observation  bool   `json:"observation"`
	Nested       bool   `json:"nested"`
}

// parseCode parses LOINC code from FHIR response.
func (a *LOINCAdapter) parseCode(data []byte) (*LOINCCode, error) {
	var fhirResult struct {
		Parameter []struct {
			Name      string `json:"name"`
			ValueString string `json:"valueString,omitempty"`
			ValueCode   string `json:"valueCode,omitempty"`
		} `json:"parameter"`
	}
	if err := json.Unmarshal(data, &fhirResult); err != nil {
		return nil, fmt.Errorf("failed to parse code: %w", err)
	}

	code := &LOINCCode{}
	for _, param := range fhirResult.Parameter {
		switch param.Name {
		case "code":
			code.Code = param.ValueCode
		case "display":
			code.LongCommonName = param.ValueString
		case "SHORTNAME":
			code.ShortName = param.ValueString
		case "COMPONENT":
			code.Component = param.ValueString
		case "PROPERTY":
			code.Property = param.ValueString
		case "TIME_ASPCT":
			code.TimeAspect = param.ValueString
		case "SYSTEM":
			code.System = param.ValueString
		case "SCALE_TYP":
			code.Scale = param.ValueString
		case "METHOD_TYP":
			code.Method = param.ValueString
		case "CLASS":
			code.Class = param.ValueString
		case "CLASSTYPE":
			code.ClassType = param.ValueString
		case "STATUS":
			code.Status = param.ValueString
		case "ORDER_OBS":
			code.OrderObs = param.ValueString
		}
	}

	return code, nil
}

// =============================================================================
// LOINC CLASSES
// =============================================================================

// LOINC Class constants
const (
	// Laboratory Classes
	LOINCClassChem        = "CHEM"      // Chemistry
	LOINCClassHem         = "HEM/BC"    // Hematology/Blood Bank
	LOINCClassMicro       = "MICRO"     // Microbiology
	LOINCClassUA          = "UA"        // Urinalysis
	LOINCClassCoag        = "COAG"      // Coagulation
	LOINCClassSero        = "SERO"      // Serology
	LOINCClassTox         = "TOX"       // Toxicology
	LOINCClassAllergy     = "ALLERGY"   // Allergy
	LOINCClassAbxBact     = "ABXBACT"   // Antibiotic susceptibility

	// Clinical Classes
	LOINCClassVitals      = "BDYCRC.ATOM" // Vital signs
	LOINCClassClinical    = "CLIN"       // Clinical
	LOINCClassSurvey      = "SURVEY"     // Survey instruments
	LOINCClassDocument    = "DOC"        // Document sections
	LOINCClassAttachment  = "ATTACH"     // Attachments

	// Imaging Classes
	LOINCClassRad         = "RAD"        // Radiology
	LOINCClassPath        = "PATH"       // Pathology
	LOINCClassCardiac     = "CARD"       // Cardiac

	// Drug Classes
	LOINCClassDrug        = "DRUG/TOX"   // Drug levels/Toxicology
)

// LOINCClassInfo contains information about a LOINC class.
type LOINCClassInfo struct {
	Class       string
	Name        string
	Description string
	Category    string
}

// GetLOINCClasses returns information about LOINC classes.
func GetLOINCClasses() []LOINCClassInfo {
	return []LOINCClassInfo{
		{LOINCClassChem, "Chemistry", "Chemistry laboratory tests", "Laboratory"},
		{LOINCClassHem, "Hematology/Blood Bank", "Blood tests and transfusion medicine", "Laboratory"},
		{LOINCClassMicro, "Microbiology", "Microbiology culture and sensitivity", "Laboratory"},
		{LOINCClassUA, "Urinalysis", "Urine analysis tests", "Laboratory"},
		{LOINCClassCoag, "Coagulation", "Coagulation studies", "Laboratory"},
		{LOINCClassSero, "Serology", "Serological tests", "Laboratory"},
		{LOINCClassTox, "Toxicology", "Toxicology screens", "Laboratory"},
		{LOINCClassVitals, "Vital Signs", "Physiological measurements", "Clinical"},
		{LOINCClassClinical, "Clinical", "Clinical observations", "Clinical"},
		{LOINCClassSurvey, "Survey", "Patient-reported instruments", "Clinical"},
		{LOINCClassRad, "Radiology", "Imaging procedures", "Imaging"},
		{LOINCClassPath, "Pathology", "Anatomic pathology", "Imaging"},
	}
}

// =============================================================================
// LOINC PROPERTIES (AXES)
// =============================================================================

// LOINC Property (2nd axis)
const (
	PropertyMassConcentration = "MCnc"  // Mass Concentration
	PropertySubstanceConcentration = "SCnc" // Substance Concentration
	PropertyNumberConcentration = "NCnc" // Number Concentration
	PropertyCatalyticConcentration = "CCnc" // Catalytic Concentration
	PropertyMassRate        = "MRat"  // Mass Rate
	PropertySubstanceRate   = "SRat"  // Substance Rate
	PropertyNumberRate      = "NRat"  // Number Rate
	PropertyCatalyticRate   = "CRat"  // Catalytic Rate
	PropertyMassContent     = "MCnt"  // Mass Content
	PropertyNumber          = "Num"   // Number/Count
	PropertyPresenceAbsence = "Pres"  // Presence/Absence
	PropertyArbitrary       = "ACnc"  // Arbitrary Concentration
	PropertyThreshold       = "Thresh" // Threshold
	PropertyIdentifier      = "ID"    // Identifier
	PropertyDate            = "Date"  // Date
	PropertyType            = "Type"  // Type
	PropertyFinding         = "Find"  // Finding
	PropertyImpression      = "Imp"   // Impression
)

// =============================================================================
// LOINC SCALE TYPES (6th axis)
// =============================================================================

// LOINC Scale Type constants
const (
	ScaleQuantitative  = "Qn"   // Quantitative (numeric result)
	ScaleOrdinal       = "Ord"  // Ordinal (ordered categorical)
	ScaleNominal       = "Nom"  // Nominal (unordered categorical)
	ScaleNarrative     = "Nar"  // Narrative (text)
	ScaleDocument      = "Doc"  // Document
	ScaleMulti         = "Multi" // Multiple answer list
	ScaleSet           = "Set"  // Set (unordered set)
)

// ScaleInfo contains information about a LOINC scale type.
type ScaleInfo struct {
	Scale       string
	Name        string
	Description string
	ResultType  string
}

// GetScaleTypes returns information about LOINC scale types.
func GetScaleTypes() []ScaleInfo {
	return []ScaleInfo{
		{ScaleQuantitative, "Quantitative", "Numeric value with units", "number"},
		{ScaleOrdinal, "Ordinal", "Ordered categorical value", "coded"},
		{ScaleNominal, "Nominal", "Unordered categorical value", "coded"},
		{ScaleNarrative, "Narrative", "Free text description", "string"},
		{ScaleDocument, "Document", "Document reference", "attachment"},
		{ScaleMulti, "Multiple", "Multiple select answers", "coded[]"},
		{ScaleSet, "Set", "Unordered set of values", "coded[]"},
	}
}

// =============================================================================
// COMMON LAB PANELS
// =============================================================================

// Common LOINC panel codes
const (
	// Chemistry Panels
	PanelBMP       = "24320-4"  // Basic Metabolic Panel
	PanelCMP       = "24323-8"  // Comprehensive Metabolic Panel
	PanelLipid     = "24331-1"  // Lipid Panel
	PanelLiver     = "24325-3"  // Hepatic Function Panel
	PanelRenal     = "24362-6"  // Renal Function Panel
	PanelThyroid   = "24348-5"  // Thyroid Panel

	// Hematology Panels
	PanelCBC       = "58410-2"  // Complete Blood Count
	PanelCBCDiff   = "57021-8"  // CBC with Differential
	PanelCoag      = "34528-5"  // Coagulation Panel

	// Urinalysis
	PanelUA        = "24356-8"  // Urinalysis Panel

	// Cardiac
	PanelCardiac   = "24325-3"  // Cardiac Panel
)

// CommonPanel represents a commonly ordered laboratory panel.
type CommonPanel struct {
	Code        string
	Name        string
	Description string
	Category    string
}

// GetCommonPanels returns commonly ordered laboratory panels.
func GetCommonPanels() []CommonPanel {
	return []CommonPanel{
		{PanelBMP, "Basic Metabolic Panel", "Glucose, electrolytes, BUN, creatinine", "Chemistry"},
		{PanelCMP, "Comprehensive Metabolic Panel", "BMP plus liver enzymes and protein", "Chemistry"},
		{PanelLipid, "Lipid Panel", "Total cholesterol, HDL, LDL, triglycerides", "Chemistry"},
		{PanelLiver, "Hepatic Function Panel", "Liver enzymes, bilirubin, albumin", "Chemistry"},
		{PanelRenal, "Renal Function Panel", "BUN, creatinine, eGFR", "Chemistry"},
		{PanelThyroid, "Thyroid Panel", "TSH, T3, T4", "Chemistry"},
		{PanelCBC, "Complete Blood Count", "RBC, WBC, hemoglobin, hematocrit, platelets", "Hematology"},
		{PanelCBCDiff, "CBC with Differential", "CBC plus WBC differential", "Hematology"},
		{PanelCoag, "Coagulation Panel", "PT, INR, PTT", "Hematology"},
		{PanelUA, "Urinalysis", "Physical, chemical, microscopic urine exam", "Urinalysis"},
	}
}

// =============================================================================
// REFERENCE RANGE SUPPORT
// =============================================================================

// ReferenceRange represents a laboratory reference range.
type ReferenceRange struct {
	LOINCCode       string   `json:"loinc_code"`
	LowValue        *float64 `json:"low_value,omitempty"`
	HighValue       *float64 `json:"high_value,omitempty"`
	Unit            string   `json:"unit"`
	AgeMin          int      `json:"age_min,omitempty"`
	AgeMax          int      `json:"age_max,omitempty"`
	Gender          string   `json:"gender,omitempty"`      // M, F, or empty for both
	CriticalLow     *float64 `json:"critical_low,omitempty"`
	CriticalHigh    *float64 `json:"critical_high,omitempty"`
	InterpretationNotes string `json:"interpretation_notes,omitempty"`
}

// GetReferenceRanges retrieves reference ranges for a LOINC code.
func (a *LOINCAdapter) GetReferenceRanges(ctx context.Context, loincCode string) ([]*ReferenceRange, error) {
	// Reference ranges are typically sourced from:
	// 1. Laboratory instrument manuals
	// 2. Clinical laboratory standards
	// 3. Published medical literature
	// This would integrate with KB-16 (Lab Reference Ranges)
	return nil, nil
}
