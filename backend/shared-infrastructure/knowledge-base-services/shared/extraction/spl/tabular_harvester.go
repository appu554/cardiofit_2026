// Package spl provides the Tabular Harvester for extracting structured data from SPL tables.
//
// Phase 3a.5: Tabular Harvester
// Key Principle: Parse <table> blocks from SPL XML and output structured JSON
// Output: JSON, not prose - Structured, verifiable, LLM-safe
// This becomes the KB-1 renal dosing spine
package spl

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/cardiofit/shared/datasources/dailymed"
)

// =============================================================================
// TABULAR HARVESTER
// =============================================================================

// TabularHarvester extracts structured data from SPL HTML tables
// Output: JSON, not prose. Preserves headers, units, footnotes.
type TabularHarvester struct {
	unitNormalizer *UnitNormalizer
	gfrPattern     *regexp.Regexp
	cpPattern      *regexp.Regexp
}

// NewTabularHarvester creates a new table harvester
func NewTabularHarvester() *TabularHarvester {
	return &TabularHarvester{
		unitNormalizer: NewUnitNormalizer(),
		gfrPattern:     regexp.MustCompile(`(?i)(CrCl|eGFR|GFR|creatinine\s+clearance)`),
		cpPattern:      regexp.MustCompile(`(?i)(Child[-\s]?Pugh|hepatic)`),
	}
}

// =============================================================================
// HARVESTED TABLE TYPES
// =============================================================================

// HarvestedTable represents a fully parsed table in JSON format
type HarvestedTable struct {
	ID          string         `json:"id"`
	Type        TableType      `json:"type"`          // GFR_DOSE, HEPATIC_DOSE, INTERACTION, etc.
	Headers     []ColumnHeader `json:"headers"`
	Rows        []TableRow     `json:"rows"`
	Footnotes   []string       `json:"footnotes,omitempty"`
	SourceLOINC string         `json:"source_loinc"` // Which section it came from
	Confidence  float64        `json:"confidence"`
}

// ColumnHeader represents a table column with optional unit
type ColumnHeader struct {
	Name     string `json:"name"`
	Unit     string `json:"unit,omitempty"`      // e.g., "mL/min", "mg", "%"
	Position int    `json:"position"`
}

// TableRow represents a single row in a harvested table
type TableRow struct {
	Condition  string            `json:"condition"`            // e.g., "CrCl 30-60 mL/min"
	Values     map[string]string `json:"values"`               // Column name → value
	Parsed     *ParsedDoseRule   `json:"parsed,omitempty"`     // Structured interpretation
	RowIndex   int               `json:"row_index"`
}

// ParsedDoseRule represents a structured dosing rule
type ParsedDoseRule struct {
	GFRMin       *float64 `json:"gfr_min,omitempty"`
	GFRMax       *float64 `json:"gfr_max,omitempty"`
	ChildPugh    string   `json:"child_pugh,omitempty"`     // "A", "B", "C"
	Action       string   `json:"action"`                   // "REDUCE", "AVOID", "CONTRAINDICATED", "NO_CHANGE"
	DoseAdjust   string   `json:"dose_adjust,omitempty"`    // "50%", "250mg BID"
	MaxDose      string   `json:"max_dose,omitempty"`
	Frequency    string   `json:"frequency,omitempty"`      // "BID", "TID", "daily"
	Notes        string   `json:"notes,omitempty"`
}

// TableType classifies the type of dosing table
type TableType string

const (
	TableTypeGFRDose      TableType = "GFR_DOSE"       // Renal dosing table
	TableTypeHepaticDose  TableType = "HEPATIC_DOSE"   // Hepatic dosing table
	TableTypeInteraction  TableType = "INTERACTION"    // Drug interaction table
	TableTypeAdverseEvent TableType = "ADVERSE_EVENT"  // Adverse events table
	TableTypePKParams     TableType = "PK_PARAMETERS"  // Pharmacokinetic parameters
	TableTypeAgeWeight    TableType = "AGE_WEIGHT"     // Age/weight based dosing
	TableTypeUnknown      TableType = "UNKNOWN"
)

// =============================================================================
// HARVESTING METHODS
// =============================================================================

// HarvestTable extracts structured data from an SPL table
func (h *TabularHarvester) HarvestTable(table dailymed.SPLTable, loincCode string) HarvestedTable {
	harvested := HarvestedTable{
		ID:          table.ID,
		SourceLOINC: loincCode,
		Confidence:  0.9,
	}

	// Extract and normalize headers
	harvested.Headers = h.extractHeaders(table)

	// Classify table type
	harvested.Type = h.ClassifyTable(harvested.Headers)

	// Extract rows with parsing
	harvested.Rows = h.extractRows(table, harvested.Headers, harvested.Type)

	// Extract footnotes if present
	if table.Caption != "" {
		harvested.Footnotes = append(harvested.Footnotes, table.Caption)
	}

	return harvested
}

// HarvestAllTables harvests all tables from an SPL section
func (h *TabularHarvester) HarvestAllTables(section *dailymed.SPLSection) []HarvestedTable {
	var tables []HarvestedTable

	for _, table := range section.GetTables() {
		harvested := h.HarvestTable(table, section.Code.Code)
		tables = append(tables, harvested)
	}

	return tables
}

// =============================================================================
// TABLE CLASSIFICATION
// =============================================================================

// ClassifyTable determines the table type based on headers and content
func (h *TabularHarvester) ClassifyTable(headers []ColumnHeader) TableType {
	headerStr := strings.ToLower(h.headersToString(headers))

	switch {
	case h.gfrPattern.MatchString(headerStr) ||
		strings.Contains(headerStr, "renal") ||
		strings.Contains(headerStr, "kidney"):
		return TableTypeGFRDose

	case h.cpPattern.MatchString(headerStr) ||
		strings.Contains(headerStr, "liver"):
		return TableTypeHepaticDose

	case strings.Contains(headerStr, "interaction") ||
		strings.Contains(headerStr, "concomitant") ||
		strings.Contains(headerStr, "co-administration"):
		return TableTypeInteraction

	case strings.Contains(headerStr, "adverse") ||
		strings.Contains(headerStr, "side effect"):
		return TableTypeAdverseEvent

	case strings.Contains(headerStr, "half-life") ||
		strings.Contains(headerStr, "clearance") ||
		strings.Contains(headerStr, "bioavailability") ||
		strings.Contains(headerStr, "auc") ||
		strings.Contains(headerStr, "cmax"):
		return TableTypePKParams

	case strings.Contains(headerStr, "age") ||
		strings.Contains(headerStr, "weight") ||
		strings.Contains(headerStr, "bsa"):
		return TableTypeAgeWeight

	default:
		return TableTypeUnknown
	}
}

// =============================================================================
// HEADER EXTRACTION
// =============================================================================

func (h *TabularHarvester) extractHeaders(table dailymed.SPLTable) []ColumnHeader {
	var headers []ColumnHeader

	// First try explicit headers
	if len(table.Headers) > 0 {
		for i, hdr := range table.Headers {
			name, unit := h.parseHeaderWithUnit(hdr)
			headers = append(headers, ColumnHeader{
				Name:     name,
				Unit:     unit,
				Position: i,
			})
		}
		return headers
	}

	// Try header row cells from THead
	if len(table.THead.Row.Cells) > 0 {
		for i, cell := range table.THead.Row.Cells {
			name, unit := h.parseHeaderWithUnit(cell.Content)
			headers = append(headers, ColumnHeader{
				Name:     name,
				Unit:     unit,
				Position: i,
			})
		}
	} else if len(table.THead.Row.HeaderCells) > 0 {
		// Also check HeaderCells (<th> elements)
		for i, cell := range table.THead.Row.HeaderCells {
			name, unit := h.parseHeaderWithUnit(cell.Content)
			headers = append(headers, ColumnHeader{
				Name:     name,
				Unit:     unit,
				Position: i,
			})
		}
	}

	return headers
}

// parseHeaderWithUnit splits "Dose (mg)" into name="Dose" and unit="mg"
func (h *TabularHarvester) parseHeaderWithUnit(header string) (string, string) {
	// Clean HTML
	header = stripHTMLTags(header)
	header = strings.TrimSpace(header)

	// Look for unit in parentheses
	unitPattern := regexp.MustCompile(`\(([^)]+)\)\s*$`)
	matches := unitPattern.FindStringSubmatch(header)

	if len(matches) > 1 {
		name := strings.TrimSpace(unitPattern.ReplaceAllString(header, ""))
		unit := h.unitNormalizer.Normalize(matches[1])
		return name, unit
	}

	return header, ""
}

// =============================================================================
// ROW EXTRACTION
// =============================================================================

func (h *TabularHarvester) extractRows(table dailymed.SPLTable, headers []ColumnHeader, tableType TableType) []TableRow {
	var rows []TableRow

	for rowIdx, row := range table.Rows {
		if len(row.Cells) == 0 {
			continue
		}

		tableRow := TableRow{
			Values:   make(map[string]string),
			RowIndex: rowIdx,
		}

		// First cell is typically the condition
		if len(row.Cells) > 0 {
			tableRow.Condition = stripHTMLTags(row.Cells[0].Content)
		}

		// Map remaining cells to headers
		for i, cell := range row.Cells {
			if i < len(headers) {
				tableRow.Values[headers[i].Name] = stripHTMLTags(cell.Content)
			}
		}

		// Try to parse structured rule based on table type
		tableRow.Parsed = h.parseRowRule(tableRow, tableType)

		rows = append(rows, tableRow)
	}

	return rows
}

// parseRowRule attempts to extract structured dosing rules from a row
func (h *TabularHarvester) parseRowRule(row TableRow, tableType TableType) *ParsedDoseRule {
	rule := &ParsedDoseRule{}
	hasData := false

	switch tableType {
	case TableTypeGFRDose:
		// Extract GFR bounds from condition
		gfrBounds := h.extractGFRBounds(row.Condition)
		if gfrBounds != nil {
			rule.GFRMin = gfrBounds.Min
			rule.GFRMax = gfrBounds.Max
			hasData = true
		}

	case TableTypeHepaticDose:
		// Extract Child-Pugh class from condition
		cpClass := h.extractChildPughClass(row.Condition)
		if cpClass != "" {
			rule.ChildPugh = cpClass
			hasData = true
		}
	}

	// Extract action/dose adjustment from values
	for _, value := range row.Values {
		lower := strings.ToLower(value)

		// Detect action
		if strings.Contains(lower, "contraindicated") {
			rule.Action = "CONTRAINDICATED"
			hasData = true
		} else if strings.Contains(lower, "avoid") {
			rule.Action = "AVOID"
			hasData = true
		} else if strings.Contains(lower, "reduce") {
			rule.Action = "REDUCE"
			hasData = true
		} else if strings.Contains(lower, "no adjustment") || strings.Contains(lower, "no change") {
			rule.Action = "NO_CHANGE"
			hasData = true
		}

		// Extract dose value
		dosePattern := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|µg|mL|%|units?)`)
		if matches := dosePattern.FindStringSubmatch(value); len(matches) > 0 {
			rule.DoseAdjust = matches[0]
			hasData = true
		}

		// Extract frequency
		freqPattern := regexp.MustCompile(`(?i)(once|twice|BID|TID|QID|daily|weekly|q\d+h)`)
		if matches := freqPattern.FindStringSubmatch(value); len(matches) > 0 {
			rule.Frequency = matches[0]
			hasData = true
		}

		// Extract max dose
		maxPattern := regexp.MustCompile(`(?i)max(?:imum)?\s*(?:dose)?\s*[:=]?\s*(\d+(?:\.\d+)?\s*(?:mg|g|mcg)?)`)
		if matches := maxPattern.FindStringSubmatch(value); len(matches) > 1 {
			rule.MaxDose = matches[1]
			hasData = true
		}
	}

	if hasData {
		return rule
	}
	return nil
}

// GFRBounds represents min/max GFR values
type GFRBounds struct {
	Min *float64
	Max *float64
}

func (h *TabularHarvester) extractGFRBounds(text string) *GFRBounds {
	// Pattern for ranges like "30-60", "≥60", "<15"
	rangePattern := regexp.MustCompile(`(\d+)\s*[-–to]+\s*(\d+)`)
	singlePattern := regexp.MustCompile(`([<>≤≥])\s*(\d+)`)

	bounds := &GFRBounds{}

	// Try range first
	if matches := rangePattern.FindStringSubmatch(text); len(matches) > 2 {
		if min, err := parseFloat(matches[1]); err == nil {
			bounds.Min = &min
		}
		if max, err := parseFloat(matches[2]); err == nil {
			bounds.Max = &max
		}
		return bounds
	}

	// Try single value with operator
	if matches := singlePattern.FindStringSubmatch(text); len(matches) > 2 {
		val, err := parseFloat(matches[2])
		if err != nil {
			return nil
		}

		switch matches[1] {
		case "<", "≤":
			bounds.Max = &val
		case ">", "≥":
			bounds.Min = &val
		}
		return bounds
	}

	return nil
}

func (h *TabularHarvester) extractChildPughClass(text string) string {
	lower := strings.ToLower(text)

	switch {
	case strings.Contains(lower, "class a") || strings.Contains(lower, "child-pugh a") || strings.Contains(lower, "mild"):
		return "A"
	case strings.Contains(lower, "class b") || strings.Contains(lower, "child-pugh b") || strings.Contains(lower, "moderate"):
		return "B"
	case strings.Contains(lower, "class c") || strings.Contains(lower, "child-pugh c") || strings.Contains(lower, "severe"):
		return "C"
	}

	return ""
}

// =============================================================================
// SERIALIZATION
// =============================================================================

// ToJSON serializes a harvested table for storage in source_sections.parsed_tables
func (t *HarvestedTable) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// ToJSONPretty returns formatted JSON for debugging
func (t *HarvestedTable) ToJSONPretty() ([]byte, error) {
	return json.MarshalIndent(t, "", "  ")
}

// FromJSON deserializes a harvested table
func FromJSON(data []byte) (*HarvestedTable, error) {
	var t HarvestedTable
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// =============================================================================
// UNIT NORMALIZER
// =============================================================================

// UnitNormalizer standardizes unit representations
type UnitNormalizer struct {
	mappings map[string]string
}

// NewUnitNormalizer creates a new unit normalizer
func NewUnitNormalizer() *UnitNormalizer {
	return &UnitNormalizer{
		mappings: map[string]string{
			"ml/min":       "mL/min",
			"ml/min/1.73":  "mL/min/1.73m²",
			"ml/min/1.73m": "mL/min/1.73m²",
			"milligrams":   "mg",
			"grams":        "g",
			"micrograms":   "mcg",
			"μg":           "mcg",
			"µg":           "mcg",
			"milliliters":  "mL",
			"liters":       "L",
			"units":        "U",
			"iu":           "IU",
			"percent":      "%",
		},
	}
}

// Normalize standardizes a unit string
func (u *UnitNormalizer) Normalize(unit string) string {
	lower := strings.ToLower(strings.TrimSpace(unit))

	if normalized, ok := u.mappings[lower]; ok {
		return normalized
	}

	return unit
}

// =============================================================================
// HELPERS
// =============================================================================

func (h *TabularHarvester) headersToString(headers []ColumnHeader) string {
	var names []string
	for _, hdr := range headers {
		names = append(names, hdr.Name)
	}
	return strings.Join(names, " ")
}

func stripHTMLTags(s string) string {
	// Simple HTML tag stripping
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	result := tagPattern.ReplaceAllString(s, " ")
	result = strings.Join(strings.Fields(result), " ") // Normalize whitespace
	return strings.TrimSpace(result)
}
