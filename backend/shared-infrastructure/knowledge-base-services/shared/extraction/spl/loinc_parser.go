// Package spl provides parsers for FDA Structured Product Label (SPL) sections.
// Uses LOINC codes to identify and route clinical information to appropriate KBs.
//
// Phase 3a.4: LOINC Section Parser
// Key Feature: Extract structured data from SPL sections using LOINC codes
package spl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cardiofit/shared/datasources/dailymed"
)

// =============================================================================
// LOINC SECTION PARSER
// =============================================================================

// LOINCSectionParser extracts clinical facts from SPL sections
type LOINCSectionParser struct {
	tableExtractor   *TabularHarvester
	gfrPattern       *regexp.Regexp
	childPughPattern *regexp.Regexp
	dosePattern      *regexp.Regexp
	percentPattern   *regexp.Regexp
}

// NewLOINCSectionParser creates a parser with clinical regex patterns
func NewLOINCSectionParser() *LOINCSectionParser {
	return &LOINCSectionParser{
		tableExtractor: NewTabularHarvester(),

		// GFR patterns: "CrCl < 30", "eGFR 30-60", "GFR less than 15", "creatinine clearance ≤ 30"
		gfrPattern: regexp.MustCompile(`(?i)(CrCl|eGFR|GFR|creatinine\s+clearance)\s*([<>≤≥]|less\s+than|greater\s+than|at\s+least)?\s*(\d+)(?:\s*[-–to]+\s*(\d+))?\s*(mL/min)?`),

		// Child-Pugh patterns: "Child-Pugh A", "moderate hepatic impairment", "Child-Pugh score 7-9"
		childPughPattern: regexp.MustCompile(`(?i)(Child[-\s]?Pugh\s*([ABC]|class\s*[ABC]|score\s*\d+[-–]\d+))|((mild|moderate|severe)\s+hepatic\s+impairment)`),

		// Dose patterns: "500 mg", "250-500mg", "10 mg/kg"
		dosePattern: regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*[-–to]*\s*(\d+(?:\.\d+)?)?\s*(mg|g|mcg|µg|mL|units?|IU)(?:/\s*(kg|m2|day|dose))?`),

		// Percentage patterns: "50%", "reduce by 50%", "25-50%"
		percentPattern: regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*[-–to]*\s*(\d+(?:\.\d+)?)?\s*%`),
	}
}

// =============================================================================
// PARSED SECTION TYPES
// =============================================================================

// ParsedSection contains extracted clinical data from an SPL section
type ParsedSection struct {
	LOINCCode          string
	LOINCName          string
	HasStructuredTable bool
	Tables             []HarvestedTable
	GFRThresholds      []GFRThreshold
	ChildPughClasses   []ChildPughClassification
	DoseValues         []DoseValue
	RawText            string
	ExtractionType     ExtractionType
	Confidence         float64
	TargetKBs          []string
}

// ExtractionType indicates how the data was extracted
type ExtractionType string

const (
	ExtractionTable   ExtractionType = "TABLE_PARSE"   // High confidence - structured table
	ExtractionRegex   ExtractionType = "REGEX_PARSE"   // Medium confidence - regex patterns
	ExtractionNeedsLLM ExtractionType = "NEEDS_LLM"    // Low confidence - needs LLM gap-filling
	ExtractionNone    ExtractionType = "NO_DATA"       // Section has no relevant data
)

// GFRThreshold represents a renal function threshold
type GFRThreshold struct {
	Operator   string  `json:"operator"`    // "<", ">", "<=", ">=", "range"
	LowerBound float64 `json:"lower_bound"` // Lower GFR value
	UpperBound float64 `json:"upper_bound"` // Upper GFR value (for range)
	Unit       string  `json:"unit"`        // "mL/min" typically
	Action     string  `json:"action"`      // "reduce dose", "avoid", "contraindicated", "no adjustment"
	RawMatch   string  `json:"raw_match"`   // Original matched text
}

// ChildPughClassification represents hepatic impairment severity
type ChildPughClassification struct {
	Class       string `json:"class"`        // "A", "B", "C"
	Severity    string `json:"severity"`     // "mild", "moderate", "severe"
	ScoreRange  string `json:"score_range"`  // "5-6", "7-9", "10-15"
	Action      string `json:"action"`       // "no adjustment", "reduce dose", "avoid"
	RawMatch    string `json:"raw_match"`
}

// DoseValue represents an extracted dose
type DoseValue struct {
	Amount    float64 `json:"amount"`
	MaxAmount float64 `json:"max_amount,omitempty"` // For ranges
	Unit      string  `json:"unit"`
	PerUnit   string  `json:"per_unit,omitempty"` // kg, m2, day, dose
	RawMatch  string  `json:"raw_match"`
}

// =============================================================================
// PARSING METHODS
// =============================================================================

// ParseSection parses any SPL section and routes to appropriate extraction
func (p *LOINCSectionParser) ParseSection(section *dailymed.SPLSection) (*ParsedSection, error) {
	// Route to appropriate parser based on LOINC code
	switch section.Code.Code {
	case dailymed.LOINCDosageAdministration:
		return p.ParseDosageSection(section)
	case dailymed.LOINCContraindications:
		return p.ParseContraindicationsSection(section)
	case dailymed.LOINCDrugInteractions:
		return p.ParseDrugInteractionsSection(section)
	case dailymed.LOINCBoxedWarning:
		return p.ParseBoxedWarningSection(section)
	default:
		return p.ParseGenericSection(section)
	}
}

// ParseDosageSection extracts renal/hepatic dosing from LOINC 34068-7
func (p *LOINCSectionParser) ParseDosageSection(section *dailymed.SPLSection) (*ParsedSection, error) {
	result := &ParsedSection{
		LOINCCode: section.Code.Code,
		LOINCName: "Dosage and Administration",
		RawText:   section.GetRawText(),
		TargetKBs: []string{"KB-1", "KB-6", "KB-16"},
	}

	// Priority 1: Extract structured tables
	if section.HasTables() {
		result.HasStructuredTable = true
		result.ExtractionType = ExtractionTable
		result.Confidence = 0.95

		for _, table := range section.GetTables() {
			harvested := p.tableExtractor.HarvestTable(table, section.Code.Code)
			result.Tables = append(result.Tables, harvested)

			// Extract GFR thresholds from GFR_DOSE tables
			if harvested.Type == TableTypeGFRDose {
				thresholds := p.extractGFRFromTable(harvested)
				result.GFRThresholds = append(result.GFRThresholds, thresholds...)
			}

			// Extract hepatic adjustments from HEPATIC_DOSE tables
			if harvested.Type == TableTypeHepaticDose {
				classifications := p.extractChildPughFromTable(harvested)
				result.ChildPughClasses = append(result.ChildPughClasses, classifications...)
			}
		}

		return result, nil
	}

	// Priority 2: Extract from prose using regex
	result.ExtractionType = ExtractionNeedsLLM
	result.Confidence = 0.5

	// Try GFR regex extraction
	gfrMatches := p.gfrPattern.FindAllStringSubmatch(result.RawText, -1)
	for _, match := range gfrMatches {
		threshold := p.parseGFRMatch(match)
		if threshold != nil {
			result.GFRThresholds = append(result.GFRThresholds, *threshold)
		}
	}

	// Try Child-Pugh regex extraction
	cpMatches := p.childPughPattern.FindAllStringSubmatch(result.RawText, -1)
	for _, match := range cpMatches {
		classification := p.parseChildPughMatch(match)
		if classification != nil {
			result.ChildPughClasses = append(result.ChildPughClasses, *classification)
		}
	}

	// If we found thresholds via regex, upgrade extraction type
	if len(result.GFRThresholds) > 0 || len(result.ChildPughClasses) > 0 {
		result.ExtractionType = ExtractionRegex
		result.Confidence = 0.75
	}

	return result, nil
}

// ParseContraindicationsSection extracts contraindication data from LOINC 34070-3
func (p *LOINCSectionParser) ParseContraindicationsSection(section *dailymed.SPLSection) (*ParsedSection, error) {
	result := &ParsedSection{
		LOINCCode: section.Code.Code,
		LOINCName: "Contraindications",
		RawText:   section.GetRawText(),
		TargetKBs: []string{"KB-4", "KB-5"},
	}

	if section.HasTables() {
		result.HasStructuredTable = true
		result.ExtractionType = ExtractionTable
		result.Confidence = 0.90

		for _, table := range section.GetTables() {
			harvested := p.tableExtractor.HarvestTable(table, section.Code.Code)
			result.Tables = append(result.Tables, harvested)
		}
	} else {
		result.ExtractionType = ExtractionNeedsLLM
		result.Confidence = 0.5
	}

	return result, nil
}

// ParseDrugInteractionsSection extracts DDI data from LOINC 34073-7
func (p *LOINCSectionParser) ParseDrugInteractionsSection(section *dailymed.SPLSection) (*ParsedSection, error) {
	result := &ParsedSection{
		LOINCCode: section.Code.Code,
		LOINCName: "Drug Interactions",
		RawText:   section.GetRawText(),
		TargetKBs: []string{"KB-5"},
	}

	if section.HasTables() {
		result.HasStructuredTable = true
		result.ExtractionType = ExtractionTable
		result.Confidence = 0.90

		for _, table := range section.GetTables() {
			harvested := p.tableExtractor.HarvestTable(table, section.Code.Code)
			result.Tables = append(result.Tables, harvested)
		}
	} else {
		result.ExtractionType = ExtractionNeedsLLM
		result.Confidence = 0.5
	}

	return result, nil
}

// ParseBoxedWarningSection extracts black box warning data from LOINC 34066-1
func (p *LOINCSectionParser) ParseBoxedWarningSection(section *dailymed.SPLSection) (*ParsedSection, error) {
	result := &ParsedSection{
		LOINCCode: section.Code.Code,
		LOINCName: "Boxed Warning",
		RawText:   section.GetRawText(),
		TargetKBs: []string{"KB-4"},
	}

	// Boxed warnings are often prose, but very important
	// Check for tables first
	if section.HasTables() {
		result.HasStructuredTable = true
		result.ExtractionType = ExtractionTable
		result.Confidence = 0.95

		for _, table := range section.GetTables() {
			harvested := p.tableExtractor.HarvestTable(table, section.Code.Code)
			result.Tables = append(result.Tables, harvested)
		}
	} else {
		// Boxed warnings often need LLM for semantic extraction
		result.ExtractionType = ExtractionNeedsLLM
		result.Confidence = 0.6
	}

	return result, nil
}

// ParseGenericSection handles sections without specific parsers
func (p *LOINCSectionParser) ParseGenericSection(section *dailymed.SPLSection) (*ParsedSection, error) {
	result := &ParsedSection{
		LOINCCode: section.Code.Code,
		LOINCName: section.Code.DisplayName,
		RawText:   section.GetRawText(),
		TargetKBs: p.RouteToKBs(section.Code.Code),
	}

	if section.HasTables() {
		result.HasStructuredTable = true
		result.ExtractionType = ExtractionTable
		result.Confidence = 0.80

		for _, table := range section.GetTables() {
			harvested := p.tableExtractor.HarvestTable(table, section.Code.Code)
			result.Tables = append(result.Tables, harvested)
		}
	} else {
		result.ExtractionType = ExtractionNeedsLLM
		result.Confidence = 0.4
	}

	return result, nil
}

// =============================================================================
// ROUTING
// =============================================================================

// RouteToKBs determines which KBs should receive facts from this section
func (p *LOINCSectionParser) RouteToKBs(loincCode string) []string {
	routing := map[string][]string{
		"34066-1": {"KB-4"},                   // Boxed Warning → Safety
		"34068-7": {"KB-1", "KB-6", "KB-16"},  // Dosage → Dosing, Formulary, Lab
		"34070-3": {"KB-4", "KB-5"},           // Contraindications → Safety, DDI
		"34073-7": {"KB-5"},                   // Drug Interactions → DDI
		"34077-8": {"KB-4"},                   // Pregnancy → Safety (also LactMed)
		"34080-2": {"KB-4"},                   // Nursing Mothers → Safety
		"34081-0": {"KB-4", "KB-1"},           // Pediatric Use → Safety, Dosing
		"34082-8": {"KB-4", "KB-1"},           // Geriatric Use → Safety, Dosing
		"34090-1": {"KB-1"},                   // Clinical Pharm → PK params
		"43685-7": {"KB-4"},                   // Warnings → Safety
		"34084-4": {"KB-4"},                   // Adverse Reactions → Safety
	}

	if targets, ok := routing[loincCode]; ok {
		return targets
	}
	return []string{} // Unknown section, don't route
}

// CanExtractWithoutLLM returns true if we have enough structured data
func (p *ParsedSection) CanExtractWithoutLLM() bool {
	return p.ExtractionType == ExtractionTable || p.ExtractionType == ExtractionRegex
}

// =============================================================================
// HELPER METHODS
// =============================================================================

func (p *LOINCSectionParser) parseGFRMatch(match []string) *GFRThreshold {
	if len(match) < 4 {
		return nil
	}

	threshold := &GFRThreshold{
		RawMatch: match[0],
		Unit:     "mL/min",
	}

	// Parse operator
	operator := strings.ToLower(match[2])
	switch {
	case strings.Contains(operator, "<") || strings.Contains(operator, "less"):
		threshold.Operator = "<"
	case strings.Contains(operator, ">") || strings.Contains(operator, "greater") || strings.Contains(operator, "at least"):
		threshold.Operator = ">"
	case strings.Contains(operator, "≤"):
		threshold.Operator = "<="
	case strings.Contains(operator, "≥"):
		threshold.Operator = ">="
	default:
		if match[4] != "" {
			threshold.Operator = "range"
		} else {
			threshold.Operator = "="
		}
	}

	// Parse values
	if val, err := parseFloat(match[3]); err == nil {
		threshold.LowerBound = val
	}

	if match[4] != "" {
		if val, err := parseFloat(match[4]); err == nil {
			threshold.UpperBound = val
			threshold.Operator = "range"
		}
	}

	return threshold
}

func (p *LOINCSectionParser) parseChildPughMatch(match []string) *ChildPughClassification {
	if len(match) < 2 {
		return nil
	}

	classification := &ChildPughClassification{
		RawMatch: match[0],
	}

	text := strings.ToLower(match[0])

	// Determine class/severity
	switch {
	case strings.Contains(text, "child-pugh a") || strings.Contains(text, "class a") || strings.Contains(text, "mild"):
		classification.Class = "A"
		classification.Severity = "mild"
		classification.ScoreRange = "5-6"
	case strings.Contains(text, "child-pugh b") || strings.Contains(text, "class b") || strings.Contains(text, "moderate"):
		classification.Class = "B"
		classification.Severity = "moderate"
		classification.ScoreRange = "7-9"
	case strings.Contains(text, "child-pugh c") || strings.Contains(text, "class c") || strings.Contains(text, "severe"):
		classification.Class = "C"
		classification.Severity = "severe"
		classification.ScoreRange = "10-15"
	}

	return classification
}

func (p *LOINCSectionParser) extractGFRFromTable(table HarvestedTable) []GFRThreshold {
	var thresholds []GFRThreshold

	for _, row := range table.Rows {
		// Look for GFR-related values in the condition column
		matches := p.gfrPattern.FindAllStringSubmatch(row.Condition, -1)
		for _, match := range matches {
			if threshold := p.parseGFRMatch(match); threshold != nil {
				// Try to extract action from the row values
				for _, value := range row.Values {
					threshold.Action = inferAction(value)
					if threshold.Action != "" {
						break
					}
				}
				thresholds = append(thresholds, *threshold)
			}
		}
	}

	return thresholds
}

func (p *LOINCSectionParser) extractChildPughFromTable(table HarvestedTable) []ChildPughClassification {
	var classifications []ChildPughClassification

	for _, row := range table.Rows {
		matches := p.childPughPattern.FindAllStringSubmatch(row.Condition, -1)
		for _, match := range matches {
			if classification := p.parseChildPughMatch(match); classification != nil {
				for _, value := range row.Values {
					classification.Action = inferAction(value)
					if classification.Action != "" {
						break
					}
				}
				classifications = append(classifications, *classification)
			}
		}
	}

	return classifications
}

func inferAction(text string) string {
	lower := strings.ToLower(text)

	switch {
	case strings.Contains(lower, "contraindicated") || strings.Contains(lower, "avoid"):
		return "avoid"
	case strings.Contains(lower, "reduce") || strings.Contains(lower, "decrease"):
		return "reduce"
	case strings.Contains(lower, "no adjustment") || strings.Contains(lower, "no change"):
		return "no_adjustment"
	case strings.Contains(lower, "not recommended"):
		return "not_recommended"
	default:
		return ""
	}
}

func parseFloat(s string) (float64, error) {
	// Simple float parsing - in production use strconv.ParseFloat
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
