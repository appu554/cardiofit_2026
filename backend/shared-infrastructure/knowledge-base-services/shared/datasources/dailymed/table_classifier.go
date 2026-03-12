// Package dailymed provides table classification for SPL documents.
//
// Phase 3a.3: Table Classifier for DailyMed SPL
// Key Feature: Classify tables by type for targeted extraction routing
//
// Table Types:
// - GFR_DOSING: Renal dose adjustment tables (CrCl, eGFR)
// - HEPATIC_DOSING: Child-Pugh based dosing tables
// - DDI: Drug-drug interaction tables
// - PK_PARAMETERS: Pharmacokinetic parameter tables
// - ADVERSE_EVENTS: Adverse event incidence tables
// - UNKNOWN: Unclassified tables
package dailymed

import (
	"regexp"
	"strings"
)

// =============================================================================
// TABLE TYPE DEFINITIONS
// =============================================================================

// TableType represents the classification of an SPL table
type TableType string

const (
	// TableTypeGFRDosing indicates renal dose adjustment tables
	TableTypeGFRDosing TableType = "GFR_DOSING"

	// TableTypeHepaticDosing indicates Child-Pugh based dosing tables
	TableTypeHepaticDosing TableType = "HEPATIC_DOSING"

	// TableTypeDDI indicates drug-drug interaction tables
	TableTypeDDI TableType = "DDI"

	// TableTypePK indicates pharmacokinetic parameter tables
	TableTypePK TableType = "PK_PARAMETERS"

	// TableTypeAdverseEvents indicates adverse event incidence tables
	TableTypeAdverseEvents TableType = "ADVERSE_EVENTS"

	// TableTypeContraindications indicates contraindication tables
	TableTypeContraindications TableType = "CONTRAINDICATIONS"

	// TableTypeDosing indicates general dosing tables
	TableTypeDosing TableType = "DOSING"

	// TableTypeEfficacy indicates clinical trial efficacy/outcome tables
	// These contain endpoint results (hazard ratios, p-values, MACE, NNT)
	// and should NOT produce facts — they contaminate AE extraction
	TableTypeEfficacy TableType = "EFFICACY"

	// TableTypeUnknown indicates unclassified tables
	TableTypeUnknown TableType = "UNKNOWN"
)

// =============================================================================
// TABLE CLASSIFIER
// =============================================================================

// TableClassifier analyzes SPL tables to determine their type
type TableClassifier struct {
	patterns map[TableType]*classificationPattern
}

// classificationPattern contains regex patterns and keywords for classification
type classificationPattern struct {
	headerPatterns   []*regexp.Regexp
	captionPatterns  []*regexp.Regexp
	contentKeywords  []string
	requiredColumns  []string
	confidenceWeight float64
}

// NewTableClassifier creates a new table classifier
func NewTableClassifier() *TableClassifier {
	tc := &TableClassifier{
		patterns: make(map[TableType]*classificationPattern),
	}
	tc.initPatterns()
	return tc
}

// initPatterns initializes classification patterns for each table type
func (tc *TableClassifier) initPatterns() {
	// GFR/Renal Dosing patterns
	tc.patterns[TableTypeGFRDosing] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)creatinine\s+clearance`),
			regexp.MustCompile(`(?i)crcl`),
			regexp.MustCompile(`(?i)e?gfr`),
			regexp.MustCompile(`(?i)renal\s+(impairment|function)`),
			regexp.MustCompile(`(?i)ml/min`),
			regexp.MustCompile(`(?i)ml\s*/\s*min`),
			regexp.MustCompile(`(?i)kidney\s+function`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)renal`),
			regexp.MustCompile(`(?i)kidney`),
			regexp.MustCompile(`(?i)creatinine`),
		},
		contentKeywords: []string{
			"creatinine", "crcl", "egfr", "gfr", "ml/min",
			"renal impairment", "renal function", "kidney",
			"hemodialysis", "dialysis", "esrd", "ckd",
		},
		requiredColumns: []string{"dose", "dosage", "mg", "recommended"},
		confidenceWeight: 1.0,
	}

	// Hepatic Dosing patterns
	tc.patterns[TableTypeHepaticDosing] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)child[-\s]pugh`),
			regexp.MustCompile(`(?i)hepatic\s+(impairment|function)`),
			regexp.MustCompile(`(?i)liver\s+(function|impairment)`),
			regexp.MustCompile(`(?i)cirrhosis`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)hepatic`),
			regexp.MustCompile(`(?i)liver`),
			regexp.MustCompile(`(?i)child[-\s]pugh`),
		},
		contentKeywords: []string{
			"child-pugh", "child pugh", "hepatic impairment",
			"liver function", "cirrhosis", "class a", "class b", "class c",
			"mild hepatic", "moderate hepatic", "severe hepatic",
		},
		requiredColumns: []string{"dose", "dosage", "mg", "recommended"},
		confidenceWeight: 1.0,
	}

	// Drug-Drug Interaction patterns
	tc.patterns[TableTypeDDI] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)interacting\s+drug`),
			regexp.MustCompile(`(?i)concomitant`),
			regexp.MustCompile(`(?i)co[-\s]?administered`),
			regexp.MustCompile(`(?i)clinical\s+impact`),
			regexp.MustCompile(`(?i)effect\s+on`),
			regexp.MustCompile(`(?i)cyp[0-9]`),
			regexp.MustCompile(`(?i)inhibitor|inducer`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)interaction`),
			regexp.MustCompile(`(?i)concomitant`),
		},
		contentKeywords: []string{
			"interacting drug", "concomitant", "co-administered",
			"clinical impact", "cyp", "inhibitor", "inducer",
			"p-glycoprotein", "p-gp", "oatp", "ugt",
			"contraindicated", "avoid", "caution",
		},
		requiredColumns: []string{"drug", "effect", "recommendation"},
		confidenceWeight: 1.0,
	}

	// Pharmacokinetic patterns
	tc.patterns[TableTypePK] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)half[-\s]?life|t1/2`),
			regexp.MustCompile(`(?i)clearance`),
			regexp.MustCompile(`(?i)volume\s+of\s+distribution`),
			regexp.MustCompile(`(?i)auc`),
			regexp.MustCompile(`(?i)cmax`),
			regexp.MustCompile(`(?i)tmax`),
			regexp.MustCompile(`(?i)bioavailability`),
			regexp.MustCompile(`(?i)protein\s+binding`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)pharmacokinetic`),
			regexp.MustCompile(`(?i)pk\s+parameter`),
		},
		contentKeywords: []string{
			"half-life", "t1/2", "clearance", "volume of distribution",
			"auc", "cmax", "tmax", "bioavailability", "protein binding",
			"vd", "cl", "absorption", "distribution", "metabolism", "elimination",
		},
		requiredColumns: []string{"parameter", "value", "mean"},
		confidenceWeight: 0.9,
	}

	// Adverse Events patterns
	tc.patterns[TableTypeAdverseEvents] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)adverse\s+(reaction|event)`),
			regexp.MustCompile(`(?i)incidence`),
			regexp.MustCompile(`(?i)placebo`),
			regexp.MustCompile(`(?i)treatment\s+group`),
			regexp.MustCompile(`(?i)percentage|%`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)adverse`),
			regexp.MustCompile(`(?i)side\s+effect`),
		},
		contentKeywords: []string{
			"adverse reaction", "adverse event", "incidence",
			"placebo", "treatment group", "percentage", "%",
			"nausea", "headache", "dizziness", "fatigue",
		},
		requiredColumns: []string{"event", "adverse", "%"},
		confidenceWeight: 0.8,
	}

	// Contraindications patterns
	tc.patterns[TableTypeContraindications] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)contraindication`),
			regexp.MustCompile(`(?i)do\s+not\s+use`),
			regexp.MustCompile(`(?i)prohibited`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)contraindication`),
		},
		contentKeywords: []string{
			"contraindicated", "do not use", "prohibited",
			"hypersensitivity", "allergy", "anaphylaxis",
		},
		requiredColumns: []string{"condition", "reason"},
		confidenceWeight: 0.9,
	}

	// Efficacy / Clinical Trial Outcome patterns
	// These tables contain trial results (LIFE, COPERNICUS, CAPRIE, etc.)
	// and must be excluded from AE extraction to prevent garbage facts
	tc.patterns[TableTypeEfficacy] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)hazard\s+ratio`),
			regexp.MustCompile(`(?i)p[\s\-]?value`),
			regexp.MustCompile(`(?i)relative\s+risk`),
			regexp.MustCompile(`(?i)odds\s+ratio`),
			regexp.MustCompile(`(?i)primary\s+(endpoint|outcome)`),
			regexp.MustCompile(`(?i)confidence\s+interval|95%\s*CI`),
			regexp.MustCompile(`(?i)risk\s+reduction`),
			regexp.MustCompile(`(?i)number\s+needed\s+to\s+treat|NNT`),
			regexp.MustCompile(`(?i)MACE|major\s+adverse\s+cardiovascular`),
			regexp.MustCompile(`(?i)treatment\s+difference`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)efficacy`),
			regexp.MustCompile(`(?i)clinical\s+trial\s+results?`),
			regexp.MustCompile(`(?i)primary\s+analysis`),
			regexp.MustCompile(`(?i)(LIFE|COPERNICUS|CAPRIE|RALES|MERIT|CHARM)\s+(study|trial)`),
			regexp.MustCompile(`(?i)outcome`),
		},
		contentKeywords: []string{
			"hazard ratio", "p-value", "p value", "relative risk",
			"odds ratio", "confidence interval", "risk reduction",
			"primary endpoint", "primary outcome", "secondary endpoint",
			"superiority", "non-inferiority", "treatment difference",
			"event rate", "kaplan-meier", "intention to treat",
			"number needed to treat", "nnt", "mace",
		},
		requiredColumns: []string{"endpoint", "hazard", "p-value", "ratio"},
		confidenceWeight: 1.0, // High weight to outscored AE for shared-feature tables
	}

	// General Dosing patterns
	tc.patterns[TableTypeDosing] = &classificationPattern{
		headerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)dose|dosage`),
			regexp.MustCompile(`(?i)indication`),
			regexp.MustCompile(`(?i)recommended`),
			regexp.MustCompile(`(?i)mg|mcg|g`),
		},
		captionPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)dosing`),
			regexp.MustCompile(`(?i)dose`),
		},
		contentKeywords: []string{
			"dose", "dosage", "mg", "mcg", "once daily", "twice daily",
			"initial dose", "maintenance dose", "maximum dose",
		},
		requiredColumns: []string{"dose", "indication"},
		confidenceWeight: 0.7,
	}
}

// =============================================================================
// CLASSIFICATION METHODS
// =============================================================================

// ClassificationResult contains the classification outcome
type ClassificationResult struct {
	TableType       TableType
	Confidence      float64
	MatchedPatterns []string
	AlternativeType TableType
}

// ClassifyTable analyzes a table and returns its classification
func (tc *TableClassifier) ClassifyTable(table *SPLTable) *ClassificationResult {
	result := &ClassificationResult{
		TableType:       TableTypeUnknown,
		Confidence:      0.0,
		MatchedPatterns: make([]string, 0),
	}

	// Combine headers for analysis
	headerText := strings.ToLower(strings.Join(table.GetHeaders(), " "))
	// Also include any td cells in the header row (some tables use td instead of th)
	if table.THead.Row.Cells != nil {
		for _, cell := range table.THead.Row.Cells {
			headerText += " " + strings.ToLower(cell.Content)
		}
	}

	// Include caption in analysis
	captionText := strings.ToLower(table.Caption)

	// Build content text from rows
	var contentBuilder strings.Builder
	for _, row := range table.Rows {
		for _, cell := range row.Cells {
			contentBuilder.WriteString(strings.ToLower(cell.Content))
			contentBuilder.WriteString(" ")
		}
	}
	contentText := contentBuilder.String()

	// Score each table type
	bestScore := 0.0
	var secondBestType TableType
	secondBestScore := 0.0

	for tableType, pattern := range tc.patterns {
		score := tc.calculateScore(headerText, captionText, contentText, pattern)

		if score > bestScore {
			secondBestScore = bestScore
			secondBestType = result.TableType
			bestScore = score
			result.TableType = tableType
		} else if score > secondBestScore {
			secondBestScore = score
			secondBestType = tableType
		}
	}

	result.Confidence = bestScore
	if secondBestScore > 0.3 {
		result.AlternativeType = secondBestType
	}

	return result
}

// calculateScore computes a confidence score for a table type
func (tc *TableClassifier) calculateScore(headerText, captionText, contentText string, pattern *classificationPattern) float64 {
	score := 0.0

	// Check header patterns (highest weight)
	for _, p := range pattern.headerPatterns {
		if p.MatchString(headerText) {
			score += 0.4
		}
	}

	// Check caption patterns
	for _, p := range pattern.captionPatterns {
		if p.MatchString(captionText) {
			score += 0.2
		}
	}

	// Check content keywords
	keywordMatches := 0
	for _, keyword := range pattern.contentKeywords {
		if strings.Contains(contentText, keyword) {
			keywordMatches++
		}
	}
	if keywordMatches > 0 {
		// Normalize keyword score
		keywordScore := float64(keywordMatches) / float64(len(pattern.contentKeywords))
		if keywordScore > 0.3 {
			score += 0.3
		} else {
			score += keywordScore
		}
	}

	// Check required columns
	columnMatches := 0
	for _, col := range pattern.requiredColumns {
		if strings.Contains(headerText, col) || strings.Contains(contentText, col) {
			columnMatches++
		}
	}
	if columnMatches >= 2 {
		score += 0.1
	}

	// Apply confidence weight
	score *= pattern.confidenceWeight

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// =============================================================================
// BATCH CLASSIFICATION
// =============================================================================

// ClassifyAllTables classifies all tables in a document section
func (tc *TableClassifier) ClassifyAllTables(section *SPLSection) map[string]*ClassificationResult {
	results := make(map[string]*ClassificationResult)

	for i := range section.Text.Tables {
		table := &section.Text.Tables[i]
		tableID := table.ID
		if tableID == "" {
			tableID = section.Code.Code + "_table_" + string(rune('0'+i))
		}
		results[tableID] = tc.ClassifyTable(table)
	}

	return results
}

// =============================================================================
// ROUTING HELPERS
// =============================================================================

// GetTargetKBsForTableType returns the KBs that should receive data from this table type
func GetTargetKBsForTableType(tableType TableType) []string {
	switch tableType {
	case TableTypeGFRDosing:
		return []string{"KB-1"} // Drug Rules (Renal dosing)
	case TableTypeHepaticDosing:
		return []string{"KB-1"} // Drug Rules (Hepatic dosing)
	case TableTypeDDI:
		return []string{"KB-5"} // Drug Interactions
	case TableTypePK:
		return []string{"KB-1"} // Drug Rules (PK parameters)
	case TableTypeAdverseEvents:
		return []string{"KB-4"} // Patient Safety
	case TableTypeContraindications:
		return []string{"KB-4"} // Patient Safety
	case TableTypeDosing:
		return []string{"KB-1"} // Drug Rules
	case TableTypeEfficacy:
		return []string{} // Efficacy tables produce no facts — trial outcomes, not clinical rules
	default:
		return []string{} // Unknown tables need manual review
	}
}

// GetExtractionPriority returns the extraction priority for a table type
func GetExtractionPriority(tableType TableType) string {
	switch tableType {
	case TableTypeGFRDosing, TableTypeHepaticDosing:
		return "P0_CRITICAL" // Renal/hepatic dosing is safety-critical
	case TableTypeDDI, TableTypeContraindications:
		return "P0_CRITICAL" // Interactions and contraindications are safety-critical
	case TableTypeAdverseEvents:
		return "P1_HIGH"
	case TableTypeDosing:
		return "P1_HIGH"
	case TableTypePK:
		return "P2_MEDIUM"
	case TableTypeEfficacy:
		return "P3_LOW" // Efficacy tables are skipped — low priority classification only
	default:
		return "P3_LOW"
	}
}

// =============================================================================
// TABLE EXTRACTION
// =============================================================================

// ExtractedTable represents a fully extracted and classified table
type ExtractedTable struct {
	TableID        string
	TableType      TableType
	Confidence     float64
	Headers        []string
	Rows           [][]string
	Caption        string
	TargetKBs      []string
	Priority       string
	SourceSection  string
	SourceLOINC    string
	RawHTML        string
}

// ExtractAndClassifyTables extracts all tables from a section with classification
func (tc *TableClassifier) ExtractAndClassifyTables(section *SPLSection) []*ExtractedTable {
	var extracted []*ExtractedTable

	for i := range section.Text.Tables {
		table := &section.Text.Tables[i]
		classification := tc.ClassifyTable(table)

		// Convert table cells to string rows
		var rows [][]string
		for _, row := range table.Rows {
			var cells []string
			for _, cell := range row.Cells {
				cells = append(cells, stripHTMLTags(cell.Content))
			}
			rows = append(rows, cells)
		}

		// Get headers using the GetHeaders method
		headers := table.GetHeaders()
		// Fallback: check if there are td cells in the thead (some tables use td instead of th)
		if len(headers) == 0 && table.THead.Row.Cells != nil {
			for _, cell := range table.THead.Row.Cells {
				headers = append(headers, stripHTMLTags(cell.Content))
			}
		}

		tableID := table.ID
		if tableID == "" {
			tableID = section.Code.Code + "_table_" + string(rune('0'+i))
		}

		extracted = append(extracted, &ExtractedTable{
			TableID:       tableID,
			TableType:     classification.TableType,
			Confidence:    classification.Confidence,
			Headers:       headers,
			Rows:          rows,
			Caption:       table.Caption,
			TargetKBs:     GetTargetKBsForTableType(classification.TableType),
			Priority:      GetExtractionPriority(classification.TableType),
			SourceSection: section.Title,
			SourceLOINC:   section.Code.Code,
		})
	}

	return extracted
}
