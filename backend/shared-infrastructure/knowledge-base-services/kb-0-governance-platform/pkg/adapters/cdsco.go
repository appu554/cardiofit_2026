// Package adapters provides CDSCO (Central Drugs Standard Control Organization) adapter for Indian drug data.
package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// CDSCO PACKAGE INSERT ADAPTER (INDIA)
// =============================================================================

// CDSCOPackageInsertAdapter ingests drug data from Indian CDSCO Package Inserts.
// Used by KB-1 (Drug Dosing), KB-4 (Patient Safety), KB-5 (Drug Interactions), KB-6 (Formulary).
type CDSCOPackageInsertAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
}

// NewCDSCOPackageInsertAdapter creates a new CDSCO Package Insert adapter.
func NewCDSCOPackageInsertAdapter() *CDSCOPackageInsertAdapter {
	return &CDSCOPackageInsertAdapter{
		BaseAdapter: NewBaseAdapter(
			"CDSCO_PACKAGE_INSERT",
			models.AuthorityCDSCO,
			[]models.KB{models.KB1, models.KB4, models.KB5, models.KB6},
		),
		baseURL: "https://cdsco.gov.in/opencms/export/sites/CDSCO_WEB",
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// FetchUpdates retrieves package inserts updated since the given timestamp.
func (a *CDSCOPackageInsertAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// CDSCO doesn't have a formal API - would need to scrape or use RSS
	// In production: implement web scraping with proper rate limiting
	url := fmt.Sprintf("%s/api/drugs?modified_after=%s", a.baseURL, since.Format("2006-01-02"))

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

// Transform converts raw CDSCO PI PDF content to a KnowledgeItem.
func (a *CDSCOPackageInsertAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	piDoc, err := a.parsePDF(raw.RawData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CDSCO PI: %w", err)
	}

	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb1:cdsco:%s", piDoc.RegistrationNumber),
		Type:    models.TypeDosingRule,
		KB:      models.KB1,
		Version: piDoc.Version,
		Name:    piDoc.BrandName,
		Source: models.SourceAttribution{
			Authority:    models.AuthorityCDSCO,
			Document:     "CDSCO Package Insert",
			Section:      "Dosage and Administration",
			Jurisdiction: models.JurisdictionIN,
			URL:          fmt.Sprintf("https://cdsco.gov.in/drugs/%s", piDoc.RegistrationNumber),
		},
		ContentRef:  fmt.Sprintf("cdsco:pi:%s", piDoc.RegistrationNumber),
		ContentHash: "",
		State:       models.StateDraft,
		RiskLevel:   models.RiskHigh,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs CDSCO-specific validation.
func (a *CDSCOPackageInsertAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	if item.Source.Authority != models.AuthorityCDSCO {
		return fmt.Errorf("invalid authority: expected CDSCO, got %s", item.Source.Authority)
	}

	if item.Source.Jurisdiction != models.JurisdictionIN {
		return fmt.Errorf("invalid jurisdiction: expected IN, got %s", item.Source.Jurisdiction)
	}

	// Additional CDSCO-specific validation:
	// - Valid CDSCO registration number
	// - NLEM (National List of Essential Medicines) status
	// - Schedule H/H1/X classification

	return nil
}

// FetchPackageInsert retrieves a single CDSCO Package Insert by registration number.
func (a *CDSCOPackageInsertAdapter) FetchPackageInsert(ctx context.Context, regNumber string) ([]byte, error) {
	url := fmt.Sprintf("%s/drugs/%s/pi.pdf", a.baseURL, regNumber)

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

// =============================================================================
// CDSCO DOCUMENT STRUCTURES
// =============================================================================

// CDSCOPackageInsert represents parsed CDSCO Package Insert.
type CDSCOPackageInsert struct {
	RegistrationNumber string
	BrandName          string
	GenericName        string
	Manufacturer       string
	Schedule           string // H, H1, X, G
	NLEMStatus         bool   // National List of Essential Medicines
	Version            string
	ApprovalDate       string
	Sections           []CDSCOSection
}

// CDSCOSection represents a section of the PI document.
type CDSCOSection struct {
	Title   string
	Content string
}

// parsePDF parses CDSCO PI PDF content.
func (a *CDSCOPackageInsertAdapter) parsePDF(data []byte) (*CDSCOPackageInsert, error) {
	// Placeholder - would use PDF library in production
	return &CDSCOPackageInsert{
		RegistrationNumber: "REG-IN-12345",
		BrandName:          "Unknown",
		GenericName:        "Unknown",
		Schedule:           "H",
		Version:            "1.0",
	}, nil
}

// CDSCO Section Identifiers for Package Inserts.
const (
	CDSCOSectionComposition       = "1"   // Composition
	CDSCOSectionDosageForm        = "2"   // Dosage Form/s
	CDSCOSectionIndications       = "3"   // Indication/s
	CDSCOSectionDosageAdmin       = "4"   // Dosage & Administration
	CDSCOSectionContraindications = "5"   // Contraindication/s
	CDSCOSectionWarnings          = "6"   // Warnings & Precautions
	CDSCOSectionInteractions      = "7"   // Drug Interactions
	CDSCOSectionAdverseReactions  = "8"   // Undesirable Effects
	CDSCOSectionOverdose          = "9"   // Overdose
	CDSCOSectionStorage           = "10"  // Storage
)

// =============================================================================
// NLEM (NATIONAL LIST OF ESSENTIAL MEDICINES) SUPPORT
// =============================================================================

// NLEMListing represents NLEM listing information.
type NLEMListing struct {
	DrugName           string
	Category           string // Primary, Secondary
	TherapeuticClass   string
	FormStrength       string
	Remarks            string
}

// FetchNLEMStatus checks if a drug is on the NLEM.
func (a *CDSCOPackageInsertAdapter) FetchNLEMStatus(ctx context.Context, genericName string) (*NLEMListing, error) {
	// NLEM API endpoint (if available)
	// In production: query NLEM database or use local cache
	return nil, nil
}

// =============================================================================
// DPCO (DRUG PRICE CONTROL ORDER) SUPPORT
// =============================================================================

// DPCOPrice represents DPCO ceiling price.
type DPCOPrice struct {
	DrugName     string
	FormStrength string
	CeilingPrice float64
	Unit         string
	EffectiveDate string
}

// FetchDPCOPrice retrieves DPCO ceiling price for a drug.
func (a *CDSCOPackageInsertAdapter) FetchDPCOPrice(ctx context.Context, drugName string) (*DPCOPrice, error) {
	// NPPA (National Pharmaceutical Pricing Authority) API
	// In production: query NPPA database
	return nil, nil
}

// =============================================================================
// SCHEDULE CLASSIFICATION
// =============================================================================

// DrugSchedule represents Indian drug scheduling.
type DrugSchedule string

const (
	ScheduleG  DrugSchedule = "G"  // General sale
	ScheduleH  DrugSchedule = "H"  // Prescription only
	ScheduleH1 DrugSchedule = "H1" // Prescription with special restrictions
	ScheduleX  DrugSchedule = "X"  // Narcotic/psychotropic substances
)

// ScheduleInfo contains schedule-specific requirements.
type ScheduleInfo struct {
	Schedule           DrugSchedule
	PrescriptionNeeded bool
	RecordKeeping      bool
	SpecialWarnings    []string
}

// GetScheduleInfo returns requirements for a drug schedule.
func GetScheduleInfo(schedule DrugSchedule) *ScheduleInfo {
	switch schedule {
	case ScheduleG:
		return &ScheduleInfo{
			Schedule:           ScheduleG,
			PrescriptionNeeded: false,
			RecordKeeping:      false,
		}
	case ScheduleH:
		return &ScheduleInfo{
			Schedule:           ScheduleH,
			PrescriptionNeeded: true,
			RecordKeeping:      false,
			SpecialWarnings:    []string{"Rx - SCHEDULE H DRUG - Warning: To be sold by retail on the prescription of a Registered Medical Practitioner only"},
		}
	case ScheduleH1:
		return &ScheduleInfo{
			Schedule:           ScheduleH1,
			PrescriptionNeeded: true,
			RecordKeeping:      true,
			SpecialWarnings:    []string{"Rx - SCHEDULE H1 DRUG - Warning: It is dangerous to take this preparation except in accordance with medical advice. Not to be sold by retail without the prescription of a Registered Medical Practitioner."},
		}
	case ScheduleX:
		return &ScheduleInfo{
			Schedule:           ScheduleX,
			PrescriptionNeeded: true,
			RecordKeeping:      true,
			SpecialWarnings:    []string{"NRx - SCHEDULE X DRUG - Warning: To be sold by retail on the prescription of a Registered Medical Practitioner only"},
		}
	default:
		return nil
	}
}
