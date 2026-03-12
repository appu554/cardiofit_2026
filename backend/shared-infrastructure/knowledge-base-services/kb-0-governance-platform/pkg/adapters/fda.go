// Package adapters provides FDA DailyMed adapter for drug data ingestion.
package adapters

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// FDA DAILYMED ADAPTER
// =============================================================================

// FDADailyMedAdapter ingests drug data from FDA DailyMed SPL documents.
// Used by KB-1 (Drug Dosing), KB-4 (Patient Safety), KB-5 (Drug Interactions).
type FDADailyMedAdapter struct {
	*BaseAdapter
	baseURL    string
	httpClient *http.Client
}

// NewFDADailyMedAdapter creates a new FDA DailyMed adapter.
func NewFDADailyMedAdapter() *FDADailyMedAdapter {
	return &FDADailyMedAdapter{
		BaseAdapter: NewBaseAdapter(
			"FDA_DAILYMED",
			models.AuthorityFDA,
			[]models.KB{models.KB1, models.KB4, models.KB5},
		),
		baseURL: "https://dailymed.nlm.nih.gov/dailymed/services/v2",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// FetchUpdates retrieves drug documents updated since the given timestamp.
func (a *FDADailyMedAdapter) FetchUpdates(ctx context.Context, since time.Time) ([]RawItem, error) {
	// Query DailyMed for updated SPL documents
	// This is a simplified implementation - real version would handle pagination
	url := fmt.Sprintf("%s/spls.json?pagesize=100", a.baseURL)

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

	// Parse response and filter by date
	// In production, this would parse the JSON response and fetch each SPL
	var items []RawItem

	// Placeholder - real implementation fetches actual SPL documents
	return items, nil
}

// Transform converts raw SPL XML to a KnowledgeItem.
func (a *FDADailyMedAdapter) Transform(ctx context.Context, raw RawItem) (*models.KnowledgeItem, error) {
	// Parse SPL XML
	var doc SPLDocument
	if err := xml.Unmarshal(raw.RawData, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse SPL: %w", err)
	}

	// Extract drug information
	item := &models.KnowledgeItem{
		ID:      fmt.Sprintf("kb1:fda:%s", doc.SetID.Extension),
		Type:    models.TypeDosingRule,
		KB:      models.KB1,
		Version: doc.VersionNum,
		Name:    doc.Title,
		Source: models.SourceAttribution{
			Authority:    models.AuthorityFDA,
			Document:     "DailyMed SPL",
			Jurisdiction: models.JurisdictionUS,
			URL:          fmt.Sprintf("https://dailymed.nlm.nih.gov/dailymed/drugInfo.cfm?setid=%s", doc.SetID.Extension),
		},
		ContentRef:  fmt.Sprintf("fda:spl:%s", doc.SetID.Extension),
		ContentHash: "", // Would compute SHA-256 hash of raw.RawData
		State:       models.StateDraft,
		RiskLevel:   models.RiskHigh, // Default for drugs, may be adjusted
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return item, nil
}

// Validate performs FDA-specific validation.
func (a *FDADailyMedAdapter) Validate(ctx context.Context, item *models.KnowledgeItem) error {
	// Validate required fields
	if item.Source.Authority != models.AuthorityFDA {
		return fmt.Errorf("invalid authority: expected FDA, got %s", item.Source.Authority)
	}

	if item.Source.Jurisdiction != models.JurisdictionUS {
		return fmt.Errorf("invalid jurisdiction: expected US, got %s", item.Source.Jurisdiction)
	}

	// Additional validation would check:
	// - Valid RxNorm code
	// - Required dosing sections present
	// - Black box warnings parsed correctly

	return nil
}

// FetchSPL retrieves a single SPL document by SetID.
func (a *FDADailyMedAdapter) FetchSPL(ctx context.Context, setID string) ([]byte, error) {
	url := fmt.Sprintf("%s/spls/%s.xml", a.baseURL, setID)

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
// SPL DOCUMENT STRUCTURES
// =============================================================================

// SPLDocument represents FDA Structured Product Labeling XML.
type SPLDocument struct {
	XMLName       xml.Name     `xml:"document"`
	ID            SPLID        `xml:"id"`
	SetID         SPLID        `xml:"setId"`
	VersionNum    string       `xml:"versionNumber>value,attr"`
	EffectiveTime string       `xml:"effectiveTime>value,attr"`
	Title         string       `xml:"title"`
	Components    []SPLComponent `xml:"component>structuredBody>component"`
}

// SPLID represents an SPL identifier.
type SPLID struct {
	Root      string `xml:"root,attr"`
	Extension string `xml:"extension,attr"`
}

// SPLComponent represents a section component.
type SPLComponent struct {
	Section SPLSection `xml:"section"`
}

// SPLSection represents a labeled section.
type SPLSection struct {
	ID    SPLID   `xml:"id"`
	Code  SPLCode `xml:"code"`
	Title string  `xml:"title"`
	Text  SPLText `xml:"text"`
}

// SPLCode represents section code.
type SPLCode struct {
	Code        string `xml:"code,attr"`
	CodeSystem  string `xml:"codeSystem,attr"`
	DisplayName string `xml:"displayName,attr"`
}

// SPLText represents section text content.
type SPLText struct {
	Paragraphs []string `xml:"paragraph"`
}

// SPL Section Codes for drug labeling.
const (
	SPLSectionDosageAdmin      = "34068-7" // DOSAGE AND ADMINISTRATION
	SPLSectionBlackBox         = "34084-4" // BOXED WARNING
	SPLSectionContraindications = "34070-3" // CONTRAINDICATIONS
	SPLSectionWarnings         = "43685-7" // WARNINGS AND PRECAUTIONS
	SPLSectionDrugInteractions = "34073-7" // DRUG INTERACTIONS
)
