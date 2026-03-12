package api

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// =============================================================================
// DAILYMED SPL FETCHER
// =============================================================================
// On-demand fetcher for FDA DailyMed SPL (Structured Product Labeling) XML.
// Used by handleGetSourceSection when raw_html is empty in the database.
//
// Flow:
//   1. Fetch full SPL XML from DailyMed API using set_id
//   2. Parse XML to find the section matching the requested LOINC code
//   3. Extract the <text> inner XML content
//   4. Return for transformation and caching
// =============================================================================

const (
	dailyMedBaseURL    = "https://dailymed.nlm.nih.gov/dailymed/services/v2/spls"
	dailyMedTimeoutSec = 30
)

var httpClient = &http.Client{
	Timeout: dailyMedTimeoutSec * time.Second,
}

// splDocument is the minimal SPL XML structure needed to extract section text.
type splDocument struct {
	XMLName   xml.Name       `xml:"document"`
	Component splDocComponent `xml:"component"`
}

type splDocComponent struct {
	StructuredBody splStructuredBody `xml:"structuredBody"`
}

type splStructuredBody struct {
	Components []splBodyComponent `xml:"component"`
}

type splBodyComponent struct {
	Section splSection `xml:"section"`
}

type splSection struct {
	ID         splID              `xml:"id"`
	Code       splCode            `xml:"code"`
	Title      string             `xml:"title"`
	Text       splText            `xml:"text"`
	Components []splBodyComponent `xml:"component"`
}

type splID struct {
	Root string `xml:"root,attr"`
}

type splCode struct {
	Code           string `xml:"code,attr"`
	CodeSystem     string `xml:"codeSystem,attr"`
	DisplayName    string `xml:"displayName,attr"`
}

type splText struct {
	InnerXML string `xml:",innerxml"`
}

// FetchSectionHTML fetches the SPL XML from DailyMed and extracts the
// inner XML of the <text> element for the section matching sectionCode.
// Returns the raw SPL inner XML (not yet transformed to HTML).
func FetchSectionHTML(ctx context.Context, setID, sectionCode string) (string, error) {
	url := fmt.Sprintf("%s/%s.xml", dailyMedBaseURL, setID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	// Note: Do NOT set Accept: application/xml — DailyMed returns 406 with it.
	// The .xml URL extension is sufficient to get XML content.

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SPL from DailyMed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("DailyMed returned status %d for set_id %s", resp.StatusCode, setID)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the XML
	var doc splDocument
	if err := xml.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("failed to parse SPL XML: %w", err)
	}

	// Search for the section matching the LOINC code
	innerXML := findSectionText(doc.Component.StructuredBody.Components, sectionCode)
	if innerXML == "" {
		return "", fmt.Errorf("section %s not found in SPL set_id %s", sectionCode, setID)
	}

	log.Printf("[DailyMed] Fetched section %s from set_id %s (%d bytes)", sectionCode, setID, len(innerXML))
	return innerXML, nil
}

// findSectionText recursively searches for a section with the given LOINC code
// and returns its <text> inner XML PLUS all subsection text concatenated.
// SPL nests content hierarchically — e.g., Adverse Reactions (34084-4) has
// subsections 6.1 (Clinical Trials) and 6.2 (Post-Marketing) with the actual
// AE tables. The pipeline maps all facts to the parent LOINC code, so we need
// to return the complete content tree.
func findSectionText(components []splBodyComponent, targetCode string) string {
	for _, comp := range components {
		section := comp.Section

		// Check if this section matches
		if section.Code.Code == targetCode {
			return collectAllText(section)
		}

		// Recurse into subsections to find the target
		if len(section.Components) > 0 {
			if found := findSectionText(section.Components, targetCode); found != "" {
				return found
			}
		}
	}
	return ""
}

// collectAllText gathers the <text> from a section and all its descendants,
// wrapping subsection titles as <h3> headers for navigation.
func collectAllText(section splSection) string {
	var parts []string

	// Add this section's own text
	text := strings.TrimSpace(section.Text.InnerXML)
	if text != "" {
		parts = append(parts, text)
	}

	// Add subsection text recursively
	for _, sub := range section.Components {
		subSection := sub.Section
		subText := strings.TrimSpace(subSection.Text.InnerXML)

		if subSection.Title != "" {
			parts = append(parts, "<title>"+subSection.Title+"</title>")
		}
		if subText != "" {
			parts = append(parts, subText)
		}

		// Recurse deeper (subsections can have sub-subsections)
		for _, deeper := range subSection.Components {
			deepText := collectAllText(deeper.Section)
			if deepText != "" {
				parts = append(parts, deepText)
			}
		}
	}

	return strings.Join(parts, "\n")
}
