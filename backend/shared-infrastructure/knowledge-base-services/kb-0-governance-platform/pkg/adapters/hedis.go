// Package adapters provides HEDIS (Healthcare Effectiveness Data and Information Set) adapter.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// HEDIS ADAPTER (NCQA QUALITY MEASURES)
// =============================================================================

// HEDISAdapter ingests HEDIS quality measures from NCQA.
// Used by KB-9 (Care Gaps), KB-13 (Quality Measures).
// HEDIS measures healthcare performance across key areas.
type HEDISAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
	apiKey     string
	year       int // HEDIS measurement year
}

// NewHEDISAdapter creates a new HEDIS adapter.
func NewHEDISAdapter(apiKey string, year int) *HEDISAdapter {
	return &HEDISAdapter{
		BaseAdapter: NewBaseAdapter(
			"HEDIS",
			models.AuthorityNCQA,
			[]models.KB{models.KB9, models.KB13},
		),
		baseURL: "https://www.ncqa.org/hedis",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		apiKey: apiKey,
		year:   year,
	}
}

// GetSupportedTypes returns the knowledge types this adapter can produce.
func (a *HEDISAdapter) GetSupportedTypes() []models.KnowledgeType {
	return []models.KnowledgeType{
		models.TypeQualityMeasure,
		models.TypeCareGap,
		models.TypeCQLLibrary,
	}
}

// FetchUpdates retrieves HEDIS measures updated since the given timestamp.
func (a *HEDISAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// HEDIS measures are typically released annually
	// Check for current year's specifications
	var items []RawItem

	// Fetch all active HEDIS measures
	measures, err := a.getAllMeasures(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HEDIS measures: %w", err)
	}

	for _, measure := range measures {
		// Only include measures updated after 'since'
		if measure.UpdatedAt.After(since) {
			data, err := json.Marshal(measure)
			if err != nil {
				continue
			}
			items = append(items, RawItem{
				ID:        measure.MeasureID,
				Authority: models.AuthorityNCQA,
				RawData:   data,
			})
		}
	}

	return items, nil
}

// Transform converts raw HEDIS measure to a KnowledgeItem.
func (a *HEDISAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	var measure NCQAHEDISMeasure
	if err := json.Unmarshal(raw.RawData, &measure); err != nil {
		return nil, fmt.Errorf("failed to parse HEDIS measure: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb13:hedis:%s:%d", measure.MeasureID, measure.Year),
		Type:    models.TypeQualityMeasure,
		KB:      models.KB13,
		Version: fmt.Sprintf("%d.%s", measure.Year, measure.Version),
		Name:    measure.Name,
		Description: measure.Description,
		Source: models.SourceAttribution{
			Authority:     models.AuthorityNCQA,
			Document:      fmt.Sprintf("HEDIS %d Technical Specifications", measure.Year),
			Section:       measure.Domain,
			Jurisdiction:  models.JurisdictionUS,
			URL:           measure.SpecificationURL,
			EffectiveDate: fmt.Sprintf("%d-01-01", measure.Year),
		},
		ContentRef:  fmt.Sprintf("hedis:%s", measure.MeasureID),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskMedium,
		WorkflowTemplate: models.TemplateQualityMed,
		RequiresDualReview: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs HEDIS-specific validation.
func (a *HEDISAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthorityNCQA {
		return fmt.Errorf("invalid authority: expected NCQA, got %s", item.Source.Authority)
	}
	return nil
}

// =============================================================================
// HEDIS MEASURE STRUCTURES
// =============================================================================

// NCQAHEDISMeasure represents a comprehensive HEDIS quality measure from NCQA.
// This is the full specification version, distinct from the simpler HEDISMeasure in cms.go.
type NCQAHEDISMeasure struct {
	MeasureID         string            `json:"measure_id"`
	Name              string            `json:"name"`
	Abbreviation      string            `json:"abbreviation"`
	Description       string            `json:"description"`
	Year              int               `json:"year"`
	Version           string            `json:"version"`
	Domain            string            `json:"domain"`          // e.g., "Effectiveness of Care"
	Subdomain         string            `json:"subdomain"`       // e.g., "Cardiovascular"
	MeasureType       HEDISMeasureType  `json:"measure_type"`
	AgeRange          *AgeRange         `json:"age_range,omitempty"`
	ProductLines      []HEDISProductLine `json:"product_lines"`
	DataSource        string            `json:"data_source"`     // Administrative, Hybrid, etc.
	NumeratorCriteria string            `json:"numerator"`
	DenominatorCriteria string          `json:"denominator"`
	Exclusions        []string          `json:"exclusions,omitempty"`
	ValueSets         []HEDISValueSet   `json:"value_sets,omitempty"`
	SpecificationURL  string            `json:"specification_url"`
	NQFNumber         string            `json:"nqf_number,omitempty"`
	CMSCoreSet        bool              `json:"cms_core_set"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// HEDISMeasureType represents the type of HEDIS measure.
type HEDISMeasureType string

const (
	HEDISMeasureProcess   HEDISMeasureType = "PROCESS"
	HEDISMeasureOutcome   HEDISMeasureType = "OUTCOME"
	HEDISMeasureStructure HEDISMeasureType = "STRUCTURE"
	HEDISMeasureAccess    HEDISMeasureType = "ACCESS"
)

// HEDISProductLine represents HEDIS product lines.
type HEDISProductLine string

const (
	HEDISCommercial HEDISProductLine = "COMMERCIAL"
	HEDISMedicare   HEDISProductLine = "MEDICARE"
	HEDISMedicaid   HEDISProductLine = "MEDICAID"
	HEDISExchange   HEDISProductLine = "EXCHANGE"
)

// AgeRange represents age criteria for a measure.
type AgeRange struct {
	MinAge   int    `json:"min_age"`
	MaxAge   int    `json:"max_age"`
	AgeUnit  string `json:"age_unit"` // years, months
	AsOfDate string `json:"as_of_date,omitempty"` // e.g., "end of measurement year"
}

// HEDISValueSet represents a value set used in a measure.
type HEDISValueSet struct {
	OID         string `json:"oid"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	CodeSystem  string `json:"code_system"` // ICD-10, CPT, SNOMED, etc.
}

// =============================================================================
// HEDIS DOMAINS
// =============================================================================

// HEDIS Domains (measurement categories)
const (
	HEDISDomainEffectiveness     = "Effectiveness of Care"
	HEDISDomainAccess            = "Access/Availability of Care"
	HEDISDomainExperience        = "Experience of Care"
	HEDISDomainUtilization       = "Utilization and Risk Adjusted Utilization"
	HEDISDomainHealthPlanDesc    = "Health Plan Descriptive Information"
	HEDISDomainCostOfCare        = "Cost of Care"
)

// HEDIS Subdomains
const (
	HEDISSubdomainPrevention     = "Prevention and Screening"
	HEDISSubdomainRespiratory    = "Respiratory Conditions"
	HEDISSubdomainCardiovascular = "Cardiovascular Conditions"
	HEDISSubdomainDiabetes       = "Diabetes"
	HEDISSubdomainMusculoskeletal = "Musculoskeletal Conditions"
	HEDISSubdomainBehavioral     = "Behavioral Health"
	HEDISSubdomainMedication     = "Medication Management"
	HEDISSubdomainOveruse        = "Overuse/Appropriateness"
)

// =============================================================================
// HEDIS MEASURE CATALOG (2024+)
// =============================================================================

// Common HEDIS Measures
var HEDISMeasureCatalog = map[string]HEDISMeasureInfo{
	// Prevention and Screening
	"BCS": {Name: "Breast Cancer Screening", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainPrevention},
	"CCS": {Name: "Cervical Cancer Screening", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainPrevention},
	"COL": {Name: "Colorectal Cancer Screening", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainPrevention},
	"CIS": {Name: "Childhood Immunization Status", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainPrevention},
	"IMA": {Name: "Immunizations for Adolescents", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainPrevention},
	"WCV": {Name: "Well-Child Visits", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainPrevention},

	// Cardiovascular
	"CBP": {Name: "Controlling High Blood Pressure", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainCardiovascular},
	"SPC": {Name: "Statin Therapy for Patients with Cardiovascular Disease", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainCardiovascular},
	"SPD": {Name: "Statin Therapy for Patients with Diabetes", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainDiabetes},

	// Diabetes
	"HBD": {Name: "Hemoglobin A1c Control for Patients with Diabetes", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainDiabetes},
	"EED": {Name: "Eye Exam for Patients with Diabetes", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainDiabetes},
	"KED": {Name: "Kidney Health Evaluation for Patients with Diabetes", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainDiabetes},

	// Respiratory
	"CWP": {Name: "Appropriate Testing for Pharyngitis", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainRespiratory},
	"AMR": {Name: "Asthma Medication Ratio", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainRespiratory},
	"PCE": {Name: "Pharmacotherapy Management of COPD Exacerbation", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainRespiratory},

	// Behavioral Health
	"ADD": {Name: "Follow-Up Care for Children Prescribed ADHD Medication", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainBehavioral},
	"AMM": {Name: "Antidepressant Medication Management", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainBehavioral},
	"FUH": {Name: "Follow-Up After Hospitalization for Mental Illness", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainBehavioral},
	"FUM": {Name: "Follow-Up After Emergency Department Visit for Mental Illness", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainBehavioral},
	"SAA": {Name: "Adherence to Antipsychotic Medications for Schizophrenia", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainBehavioral},

	// Medication Management
	"PBH": {Name: "Persistence of Beta-Blocker Treatment After Heart Attack", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainMedication},
	"MRP": {Name: "Medication Reconciliation Post-Discharge", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainMedication},

	// Access
	"AAP": {Name: "Adults' Access to Preventive/Ambulatory Health Services", Domain: HEDISDomainAccess, Subdomain: ""},
	"IET": {Name: "Initiation and Engagement of Substance Use Disorder Treatment", Domain: HEDISDomainAccess, Subdomain: HEDISSubdomainBehavioral},

	// Overuse
	"COU": {Name: "Risk of Continued Opioid Use", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainOveruse},
	"HDO": {Name: "Use of Opioids at High Dosage", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainOveruse},
	"UOP": {Name: "Use of Opioids from Multiple Providers", Domain: HEDISDomainEffectiveness, Subdomain: HEDISSubdomainOveruse},
}

// HEDISMeasureInfo contains basic measure metadata.
type HEDISMeasureInfo struct {
	Name      string
	Domain    string
	Subdomain string
}

// =============================================================================
// HEDIS OPERATIONS
// =============================================================================

// getAllMeasures retrieves all HEDIS measures for the configured year.
func (a *HEDISAdapter) getAllMeasures(ctx context.Context) ([]*NCQAHEDISMeasure, error) {
	url := fmt.Sprintf("%s/measures?year=%d", a.baseURL, a.year)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
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
		Measures []*NCQAHEDISMeasure `json:"measures"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Measures, nil
}

// GetMeasure retrieves a specific HEDIS measure by ID.
func (a *HEDISAdapter) GetMeasure(ctx context.Context, measureID string) (*NCQAHEDISMeasure, error) {
	url := fmt.Sprintf("%s/measures/%s?year=%d", a.baseURL, measureID, a.year)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if a.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.apiKey)
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

	var measure NCQAHEDISMeasure
	if err := json.NewDecoder(resp.Body).Decode(&measure); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &measure, nil
}

// GetMeasuresByDomain retrieves HEDIS measures filtered by domain.
func (a *HEDISAdapter) GetMeasuresByDomain(ctx context.Context, domain string) ([]*NCQAHEDISMeasure, error) {
	all, err := a.getAllMeasures(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*NCQAHEDISMeasure
	for _, m := range all {
		if m.Domain == domain {
			filtered = append(filtered, m)
		}
	}

	return filtered, nil
}

// GetMeasuresByProductLine retrieves measures applicable to a product line.
func (a *HEDISAdapter) GetMeasuresByProductLine(ctx context.Context, productLine HEDISProductLine) ([]*NCQAHEDISMeasure, error) {
	all, err := a.getAllMeasures(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*NCQAHEDISMeasure
	for _, m := range all {
		for _, pl := range m.ProductLines {
			if pl == productLine {
				filtered = append(filtered, m)
				break
			}
		}
	}

	return filtered, nil
}

// GetValueSetsForMeasure retrieves all value sets used by a measure.
func (a *HEDISAdapter) GetValueSetsForMeasure(ctx context.Context, measureID string) ([]HEDISValueSet, error) {
	measure, err := a.GetMeasure(ctx, measureID)
	if err != nil {
		return nil, err
	}
	if measure == nil {
		return nil, fmt.Errorf("measure not found: %s", measureID)
	}

	return measure.ValueSets, nil
}

// =============================================================================
// HEDIS TO CQL CONVERSION
// =============================================================================

// ToCQLLibrary converts a HEDIS measure specification to a CQL library stub.
func (a *HEDISAdapter) ToCQLLibrary(measure *NCQAHEDISMeasure) string {
	return fmt.Sprintf(`library %s version '%s'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers
include SupplementalDataElements version '3.0.000' called SDE
include MATGlobalCommonFunctions version '7.0.000' called Global

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'

/*
 * HEDIS Measure: %s
 * Domain: %s
 * Year: %d
 *
 * Denominator: %s
 * Numerator: %s
 */

context Patient

define "Initial Population":
  // Define initial population criteria
  true

define "Denominator":
  "Initial Population"
    and %s

define "Numerator":
  "Denominator"
    and %s

define "Denominator Exclusions":
  false
`,
		measure.Abbreviation,
		measure.Version,
		measure.Name,
		measure.Domain,
		measure.Year,
		measure.DenominatorCriteria,
		measure.NumeratorCriteria,
		"// Denominator criteria placeholder",
		"// Numerator criteria placeholder",
	)
}
