package fda

import (
	"encoding/xml"
	"fmt"
	"strings"

	"kb-1-drug-rules/internal/models"
)

// =============================================================================
// SPL XML STRUCTURES
// =============================================================================

// SPLDocument represents FDA Structured Product Labeling XML document
type SPLDocument struct {
	XMLName       xml.Name       `xml:"document"`
	ID            SPLID          `xml:"id"`
	SetID         SPLID          `xml:"setId"`
	VersionNumber SPLVersionNum  `xml:"versionNumber"`
	EffectiveTime SPLTime        `xml:"effectiveTime"`
	Title         string         `xml:"title"`
	Components    []SPLComponent `xml:"component>structuredBody>component"`
}

// SPLID represents an SPL identifier
type SPLID struct {
	Root      string `xml:"root,attr"`
	Extension string `xml:"extension,attr"`
}

// SPLVersionNum represents version number
type SPLVersionNum struct {
	Value string `xml:"value,attr"`
}

// SPLTime represents a time element
type SPLTime struct {
	Value string `xml:"value,attr"`
}

// SPLComponent represents a section component in the SPL document
type SPLComponent struct {
	Section SPLSection `xml:"section"`
}

// SPLSection represents a labeled section
type SPLSection struct {
	ID         SPLID          `xml:"id"`
	Code       SPLCode        `xml:"code"`
	Title      string         `xml:"title"`
	Text       SPLText        `xml:"text"`
	Subjects   []SPLSubject   `xml:"subject"`
	Components []SPLComponent `xml:"component"`
}

// SPLCode represents section code (LOINC code)
type SPLCode struct {
	Code        string `xml:"code,attr"`
	CodeSystem  string `xml:"codeSystem,attr"`
	DisplayName string `xml:"displayName,attr"`
}

// SPLText represents section text content
type SPLText struct {
	Content    string     `xml:",chardata"`
	Paragraphs []string   `xml:"paragraph"`
	Lists      []SPLList  `xml:"list"`
	Tables     []SPLTable `xml:"table"`
}

// SPLList represents a list in text
type SPLList struct {
	Items []SPLListItem `xml:"item"`
}

// SPLListItem represents a list item
type SPLListItem struct {
	Content string `xml:",chardata"`
}

// SPLTable represents a table
type SPLTable struct {
	Header SPLTableHeader `xml:"thead"`
	Body   SPLTableBody   `xml:"tbody"`
}

// SPLTableHeader represents table header
type SPLTableHeader struct {
	Rows []SPLTableRow `xml:"tr"`
}

// SPLTableBody represents table body
type SPLTableBody struct {
	Rows []SPLTableRow `xml:"tr"`
}

// SPLTableRow represents a table row
type SPLTableRow struct {
	HeaderCells []SPLTableCell `xml:"th"`
	DataCells   []SPLTableCell `xml:"td"`
}

// SPLTableCell represents a table cell
type SPLTableCell struct {
	Content string `xml:",chardata"`
}

// SPLSubject represents a drug subject (manufactured product)
type SPLSubject struct {
	MedicinalProduct SPLMedicinalProduct `xml:"manufacturedProduct>manufacturedProduct"`
}

// SPLMedicinalProduct represents the manufactured product details
type SPLMedicinalProduct struct {
	Name     string  `xml:"name"`
	FormCode SPLCode `xml:"formCode"`
	Generic  struct {
		GenericMedicine struct {
			Name string `xml:"name"`
		} `xml:"genericMedicine"`
	} `xml:"asEntityWithGeneric"`
	Ingredients []SPLIngredient `xml:"ingredient"`
}

// SPLIngredient represents an ingredient
type SPLIngredient struct {
	ClassCode string `xml:"classCode,attr"`
	Quantity  struct {
		Numerator struct {
			Value string `xml:"value,attr"`
			Unit  string `xml:"unit,attr"`
		} `xml:"numerator"`
		Denominator struct {
			Value string `xml:"value,attr"`
			Unit  string `xml:"unit,attr"`
		} `xml:"denominator"`
	} `xml:"quantity"`
	Substance struct {
		Code        SPLCode `xml:"code"`
		Name        string  `xml:"name"`
		ActiveMoiety struct {
			ActiveMoiety struct {
				Code SPLCode `xml:"code"`
				Name string  `xml:"name"`
			} `xml:"activeMoiety"`
		} `xml:"activeMoiety"`
	} `xml:"ingredientSubstance"`
}

// =============================================================================
// SPL SECTION CODES (LOINC)
// =============================================================================

// SPL Section LOINC codes for drug label sections
const (
	SectionDosageAdmin       = "34068-7" // DOSAGE AND ADMINISTRATION
	SectionBlackBox          = "34084-4" // BOXED WARNING
	SectionContraindications = "34070-3" // CONTRAINDICATIONS
	SectionWarnings          = "43685-7" // WARNINGS AND PRECAUTIONS
	SectionAdverseReactions  = "34084-4" // ADVERSE REACTIONS
	SectionDrugInteractions  = "34073-7" // DRUG INTERACTIONS
	SectionUseSpecificPop    = "43684-0" // USE IN SPECIFIC POPULATIONS
	SectionPediatricUse      = "34081-0" // PEDIATRIC USE
	SectionGeriatricUse      = "34082-8" // GERIATRIC USE
	SectionRenalImpairment   = "42232-9" // USE IN PATIENTS WITH RENAL IMPAIRMENT
	SectionHepaticImpairment = "42229-5" // USE IN PATIENTS WITH HEPATIC IMPAIRMENT
	SectionClinPharm         = "34090-1" // CLINICAL PHARMACOLOGY
	SectionOverdosage        = "34088-5" // OVERDOSAGE
	SectionDescription       = "34089-3" // DESCRIPTION
	SectionHowSupplied       = "34069-5" // HOW SUPPLIED
	SectionIndicationsUsage  = "34067-9" // INDICATIONS AND USAGE
)

// =============================================================================
// PARSER
// =============================================================================

// Parser parses FDA SPL documents
type Parser struct{}

// NewParser creates a new SPL parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses SPL XML into structured data
func (p *Parser) Parse(xmlData []byte) (*SPLDocument, error) {
	var doc SPLDocument
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse SPL XML: %w", err)
	}
	return &doc, nil
}

// =============================================================================
// DRUG INFORMATION EXTRACTION
// =============================================================================

// ExtractDrugInfo extracts drug identification from SPL document
func (p *Parser) ExtractDrugInfo(doc *SPLDocument) (*models.DrugIdentification, error) {
	info := &models.DrugIdentification{}

	// Get drug name and generic name from subjects
	for _, comp := range doc.Components {
		for _, subject := range comp.Section.Subjects {
			if subject.MedicinalProduct.Name != "" && info.Name == "" {
				info.Name = cleanString(subject.MedicinalProduct.Name)
			}
			if subject.MedicinalProduct.Generic.GenericMedicine.Name != "" && info.GenericName == "" {
				info.GenericName = cleanString(subject.MedicinalProduct.Generic.GenericMedicine.Name)
			}
		}
		// Also check nested components
		for _, nested := range comp.Section.Components {
			for _, subject := range nested.Section.Subjects {
				if subject.MedicinalProduct.Name != "" && info.Name == "" {
					info.Name = cleanString(subject.MedicinalProduct.Name)
				}
				if subject.MedicinalProduct.Generic.GenericMedicine.Name != "" && info.GenericName == "" {
					info.GenericName = cleanString(subject.MedicinalProduct.Generic.GenericMedicine.Name)
				}
			}
		}
	}

	// Fallback to title if no name found
	if info.Name == "" {
		info.Name = p.extractDrugNameFromTitle(doc.Title)
	}

	// Extract drug class from description if available
	descSection := p.GetSection(doc, SectionDescription)
	if descSection != nil {
		text := p.GetSectionText(descSection)
		info.DrugClass = p.extractDrugClass(text)
	}

	return info, nil
}

// extractDrugNameFromTitle extracts drug name from SPL title
// Title format is typically: "DRUGNAME- active ingredient tablet"
func (p *Parser) extractDrugNameFromTitle(title string) string {
	// Remove common suffixes
	title = strings.TrimSpace(title)

	// Try splitting by dash
	parts := strings.Split(title, "-")
	if len(parts) > 0 {
		name := strings.TrimSpace(parts[0])
		// Remove trademark symbols
		name = strings.ReplaceAll(name, "®", "")
		name = strings.ReplaceAll(name, "™", "")
		return name
	}
	return title
}

// extractDrugClass attempts to extract drug class from description text
func (p *Parser) extractDrugClass(text string) string {
	text = strings.ToLower(text)

	// Common drug class patterns
	classPatterns := map[string][]string{
		"ACE Inhibitor":           {"angiotensin-converting enzyme inhibitor", "ace inhibitor"},
		"ARB":                     {"angiotensin receptor blocker", "angiotensin ii receptor"},
		"Beta Blocker":            {"beta-blocker", "beta blocker", "beta-adrenergic"},
		"Calcium Channel Blocker": {"calcium channel blocker", "calcium antagonist"},
		"Diuretic":                {"diuretic", "thiazide"},
		"Statin":                  {"hmg-coa reductase inhibitor", "statin"},
		"Anticoagulant":           {"anticoagulant", "blood thinner"},
		"Antiplatelet":            {"antiplatelet", "platelet aggregation inhibitor"},
		"Antidiabetic":            {"antidiabetic", "hypoglycemic", "glucose-lowering"},
		"NSAID":                   {"nonsteroidal anti-inflammatory", "nsaid"},
		"Opioid":                  {"opioid", "narcotic analgesic"},
		"Antibiotic":              {"antibiotic", "antibacterial", "antimicrobial"},
		"Antidepressant":          {"antidepressant", "ssri", "snri"},
		"Antipsychotic":           {"antipsychotic", "neuroleptic"},
		"Benzodiazepine":          {"benzodiazepine"},
		"PPI":                     {"proton pump inhibitor", "ppi"},
		"H2 Blocker":              {"h2 receptor antagonist", "h2 blocker"},
	}

	for class, patterns := range classPatterns {
		for _, pattern := range patterns {
			if strings.Contains(text, pattern) {
				return class
			}
		}
	}

	return ""
}

// =============================================================================
// SECTION ACCESS
// =============================================================================

// GetSection retrieves a specific section by LOINC code
func (p *Parser) GetSection(doc *SPLDocument, sectionCode string) *SPLSection {
	for _, comp := range doc.Components {
		if comp.Section.Code.Code == sectionCode {
			return &comp.Section
		}
		// Check nested components
		for _, nested := range comp.Section.Components {
			if nested.Section.Code.Code == sectionCode {
				return &nested.Section
			}
			// Check deeper nesting
			for _, deeper := range nested.Section.Components {
				if deeper.Section.Code.Code == sectionCode {
					return &deeper.Section
				}
			}
		}
	}
	return nil
}

// GetAllSections returns all sections from the document
func (p *Parser) GetAllSections(doc *SPLDocument) map[string]*SPLSection {
	sections := make(map[string]*SPLSection)

	var extractSections func(comps []SPLComponent)
	extractSections = func(comps []SPLComponent) {
		for _, comp := range comps {
			if comp.Section.Code.Code != "" {
				sections[comp.Section.Code.Code] = &comp.Section
			}
			extractSections(comp.Section.Components)
		}
	}

	extractSections(doc.Components)
	return sections
}

// GetSectionText extracts plain text from a section
func (p *Parser) GetSectionText(section *SPLSection) string {
	var text strings.Builder

	// Add title
	if section.Title != "" {
		text.WriteString(section.Title)
		text.WriteString("\n\n")
	}

	// Add paragraphs
	for _, para := range section.Text.Paragraphs {
		cleaned := cleanString(para)
		if cleaned != "" {
			text.WriteString(cleaned)
			text.WriteString("\n")
		}
	}

	// Add list items
	for _, list := range section.Text.Lists {
		for _, item := range list.Items {
			cleaned := cleanString(item.Content)
			if cleaned != "" {
				text.WriteString("• ")
				text.WriteString(cleaned)
				text.WriteString("\n")
			}
		}
	}

	// Add table content
	for _, table := range section.Text.Tables {
		for _, row := range table.Body.Rows {
			var cells []string
			for _, cell := range row.DataCells {
				cleaned := cleanString(cell.Content)
				if cleaned != "" {
					cells = append(cells, cleaned)
				}
			}
			if len(cells) > 0 {
				text.WriteString(strings.Join(cells, " | "))
				text.WriteString("\n")
			}
		}
	}

	// Also check nested content
	if section.Text.Content != "" {
		cleaned := cleanString(section.Text.Content)
		if cleaned != "" {
			text.WriteString(cleaned)
			text.WriteString("\n")
		}
	}

	return strings.TrimSpace(text.String())
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// cleanString removes extra whitespace and normalizes text
func cleanString(s string) string {
	// Remove XML entities and extra whitespace
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}

	return s
}

// GetSetID returns the SetID from the document
func (p *Parser) GetSetID(doc *SPLDocument) string {
	return doc.SetID.Root
}

// GetVersion returns the version number from the document
func (p *Parser) GetVersion(doc *SPLDocument) string {
	return doc.VersionNumber.Value
}

// GetEffectiveDate returns the effective date from the document
func (p *Parser) GetEffectiveDate(doc *SPLDocument) string {
	return doc.EffectiveTime.Value
}

// HasBlackBoxWarning checks if the document contains a black box warning
func (p *Parser) HasBlackBoxWarning(doc *SPLDocument) bool {
	return p.GetSection(doc, SectionBlackBox) != nil
}
