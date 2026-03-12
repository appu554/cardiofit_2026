// Package adapters provides CMS eCQM (Electronic Clinical Quality Measures) adapter.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// CMS eCQM ADAPTER (USA)
// =============================================================================

// CMSeCQMAdapter ingests quality measures from CMS eCQM specifications.
// Used by KB-9 (Care Gaps), KB-13 (Quality Measures).
type CMSeCQMAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
}

// NewCMSeCQMAdapter creates a new CMS eCQM adapter.
func NewCMSeCQMAdapter() *CMSeCQMAdapter {
	return &CMSeCQMAdapter{
		BaseAdapter: NewBaseAdapter(
			"CMS_ECQM",
			models.AuthorityCMS,
			[]models.KB{models.KB9, models.KB13},
		),
		baseURL: "https://ecqi.healthit.gov/api",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// FetchUpdates retrieves eCQM measures updated since the given timestamp.
func (a *CMSeCQMAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// eCQI Resource Center API
	url := fmt.Sprintf("%s/measures?reporting_year=%d", a.baseURL, time.Now().Year())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	var items []RawItem
	// Parse response
	return items, nil
}

// Transform converts raw CQL/ELM content to a KnowledgeItem.
func (a *CMSeCQMAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	measure, err := a.parseMeasure(raw.RawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse eCQM: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb13:cms:%s:v%s", measure.CMSID, measure.Version),
		Type:    models.TypeQualityMeasure,
		KB:      models.KB13,
		Version: measure.Version,
		Name:    measure.Title,
		Description: measure.Description,
		Source: models.SourceAttribution{
			Authority:    models.AuthorityCMS,
			Document:     "CMS eCQM Specification",
			Section:      measure.CMSID,
			Jurisdiction: models.JurisdictionUS,
			URL:          fmt.Sprintf("https://ecqi.healthit.gov/ecqm/ec/%s", measure.CMSID),
		},
		ContentRef:  fmt.Sprintf("cms:ecqm:%s", measure.CMSID),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskMedium,
		WorkflowTemplate: models.TemplateQualityMed,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs CMS-specific validation.
func (a *CMSeCQMAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthorityCMS {
		return fmt.Errorf("invalid authority: expected CMS, got %s", item.Source.Authority)
	}

	// Additional CMS-specific validation:
	// - Valid CMS measure ID format (CMSxxxx)
	// - Valid reporting year
	// - CQL library compiles successfully
	// - Value sets resolve

	return nil
}

// FetchMeasure retrieves a single eCQM measure by CMS ID.
func (a *CMSeCQMAdapter) FetchMeasure(ctx context.Context, cmsID string) ([]byte, error) {
	url := fmt.Sprintf("%s/measures/%s", a.baseURL, cmsID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	return io.ReadAll(resp.Body)
}

// FetchCQL retrieves the CQL library for a measure.
func (a *CMSeCQMAdapter) FetchCQL(ctx context.Context, cmsID string, version string) ([]byte, error) {
	url := fmt.Sprintf("%s/measures/%s/cql?version=%s", a.baseURL, cmsID, version)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

	return io.ReadAll(resp.Body)
}

// =============================================================================
// eCQM STRUCTURES
// =============================================================================

// eCQMMeasure represents a CMS electronic Clinical Quality Measure.
type eCQMMeasure struct {
	CMSID               string   `json:"cms_id"`       // e.g., "CMS165"
	Title               string   `json:"title"`
	Description         string   `json:"description"`
	Version             string   `json:"version"`
	ReportingYear       int      `json:"reporting_year"`
	MeasureType         string   `json:"measure_type"` // Process, Outcome, Structure
	MeasureDomain       string   `json:"measure_domain"`
	MeasureSteward      string   `json:"measure_steward"`
	Programs            []string `json:"programs"`     // MIPS, Hospital, etc.
	CQLLibraries        []CQLLibrary `json:"cql_libraries"`
	ValueSets           []ValueSetRef `json:"value_sets"`
	Population          MeasurePopulation `json:"population"`
}

// CQLLibrary represents a CQL library reference.
type CQLLibrary struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Identifier string `json:"identifier"`
	CQLContent string `json:"cql_content,omitempty"`
	ELMContent string `json:"elm_content,omitempty"`
}

// ValueSetRef represents a value set reference.
type ValueSetRef struct {
	OID      string `json:"oid"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	CodeSystem string `json:"code_system"`
}

// MeasurePopulation represents measure population criteria.
type MeasurePopulation struct {
	InitialPopulation       string `json:"initial_population"`
	Denominator             string `json:"denominator"`
	DenominatorExclusions   string `json:"denominator_exclusions,omitempty"`
	DenominatorExceptions   string `json:"denominator_exceptions,omitempty"`
	Numerator               string `json:"numerator"`
	NumeratorExclusions     string `json:"numerator_exclusions,omitempty"`
	MeasureObservation      string `json:"measure_observation,omitempty"`
}

// parseMeasure parses eCQM JSON content.
func (a *CMSeCQMAdapter) parseMeasure(data []byte) (*eCQMMeasure, error) {
	var measure eCQMMeasure
	if err := json.Unmarshal(data, &measure); err != nil {
		return nil, fmt.Errorf("failed to parse measure: %w", err)
	}
	return &measure, nil
}

// =============================================================================
// HEDIS SUPPORT (NCQA)
// =============================================================================

// HEDISMeasure represents an NCQA HEDIS measure.
type HEDISMeasure struct {
	MeasureID     string `json:"measure_id"`
	MeasureName   string `json:"measure_name"`
	Domain        string `json:"domain"`
	Subdomain     string `json:"subdomain"`
	MeasureYear   int    `json:"measure_year"`
	Description   string `json:"description"`
	Specifications string `json:"specifications"`
}

// FetchHEDISMeasure retrieves HEDIS measure specifications.
func (a *CMSeCQMAdapter) FetchHEDISMeasure(ctx context.Context, measureID string, year int) (*HEDISMeasure, error) {
	// NCQA HEDIS measures require subscription
	// In production: authenticate and fetch from NCQA API
	return nil, nil
}

// =============================================================================
// QUALITY PROGRAM MAPPINGS
// =============================================================================

// QualityProgram represents a CMS quality reporting program.
type QualityProgram string

const (
	ProgramMIPS              QualityProgram = "MIPS"               // Merit-based Incentive Payment System
	ProgramACO               QualityProgram = "ACO"                // Accountable Care Organizations
	ProgramMedicarePart_C_D  QualityProgram = "MEDICARE_PART_C_D"  // Medicare Advantage/Part D
	ProgramMedicaid          QualityProgram = "MEDICAID"           // Medicaid
	ProgramHospitalIQR       QualityProgram = "HOSPITAL_IQR"       // Hospital Inpatient Quality Reporting
	ProgramHospitalOQR       QualityProgram = "HOSPITAL_OQR"       // Hospital Outpatient Quality Reporting
	ProgramMSF               QualityProgram = "MSF"                // Merit-based Incentive Payment System
)

// ProgramMeasureMapping maps measures to quality programs.
type ProgramMeasureMapping struct {
	CMSID    string
	Program  QualityProgram
	Weight   float64
	Required bool
}

// GetMeasuresForProgram returns measures applicable to a quality program.
func (a *CMSeCQMAdapter) GetMeasuresForProgram(ctx context.Context, program QualityProgram) ([]string, error) {
	// Query API for program-specific measures
	return nil, nil
}

// =============================================================================
// MEASURE CALCULATION SUPPORT
// =============================================================================

// MeasureCalculationResult represents the result of measure calculation.
type MeasureCalculationResult struct {
	CMSID                string
	ReportingPeriod      string
	InitialPopulation    int
	Denominator          int
	DenominatorExclusions int
	DenominatorExceptions int
	Numerator            int
	PerformanceRate      float64
	PatientList          []string
}

// CalculateMeasure calculates a measure for a patient population.
// In production: this would interface with the CQL engine.
func (a *CMSeCQMAdapter) CalculateMeasure(ctx context.Context, cmsID string, patientBundle []byte) (*MeasureCalculationResult, error) {
	// This would:
	// 1. Load the CQL library
	// 2. Load the ELM (executable) content
	// 3. Execute against patient data bundle
	// 4. Return population counts
	return nil, nil
}
