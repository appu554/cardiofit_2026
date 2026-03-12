// Package factstore provides the Content Parser for transforming raw SPL table data
// into structured clinical content that KB views expect.
//
// This module addresses the gap between:
//   - Raw extracted tables: {Rows, Headers, TableType}
//   - KB view requirements: {organ, impairmentLevel, severity, etc.}
package factstore

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cardiofit/shared/datasources/dailymed"
	"github.com/cardiofit/shared/terminology"
)

// ContentParser transforms raw table data into structured fact content
type ContentParser struct {
	// Precompiled regex patterns for efficiency
	gfrPattern        *regexp.Regexp
	severityPattern   *regexp.Regexp
	mechanismPatterns []*regexp.Regexp
	ndcPattern        *regexp.Regexp
	dosePattern       *regexp.Regexp
	percentPattern    *regexp.Regexp

	// Noise filtering patterns (added for quality improvement)
	noisePatterns *NoiseFilter

	// Phase 3 Issue 2+3 Fix: MedDRA dictionary-based normalization.
	// Replaces regex-based noise filtering with deterministic dictionary lookup.
	// If term is in MedDRA (80,000+ official terms) → valid clinical term
	// If term is NOT in MedDRA → noise (filtered out)
	// Also provides FAERS-compatible MedDRA PT codes for pharmacovigilance.
	aeNormalizer terminology.AdverseEventNormalizer
}

// =============================================================================
// NOISE FILTER - Filters out table artifacts, headers, and non-clinical content
// =============================================================================

// NoiseFilter contains patterns for identifying and filtering noise from SPL tables
type NoiseFilter struct {
	// Header patterns - common column headers that should be skipped
	headerPatterns *regexp.Regexp
	// Statistical patterns - CI, percentages, study metrics
	statisticalPatterns *regexp.Regexp
	// Section label patterns - "Clinical Impact:", "Intervention:", etc.
	sectionLabelPatterns *regexp.Regexp
	// Pure numeric patterns - just numbers without clinical context
	pureNumericPattern *regexp.Regexp
	// Table artifact patterns - formatting artifacts
	tableArtifactPatterns *regexp.Regexp
	// Valid clinical condition patterns - positive match for real conditions
	clinicalConditionPatterns *regexp.Regexp
	// Valid drug name patterns - positive match for real drug names
	validDrugPatterns *regexp.Regexp
}

// NewNoiseFilter creates a new noise filter with precompiled patterns
func NewNoiseFilter() *NoiseFilter {
	return &NoiseFilter{
		// Common table headers that indicate a header row (expanded patterns)
		headerPatterns: regexp.MustCompile(`(?i)^(number\s*(of)?\s*patients?|n\s*=|total|baseline|endpoint|parameter|variable|characteristic|outcome|event|category|treatment|placebo|subjects?|arm|group|patients?|week|day|month|year|visit|time|dose|study|trial|analysis|change\s+(at|from|in)|final\s+visit|mean|median|sd|se|range|subjects?\s+with|major|minor|primary|secondary|efficacy|safety|incidence|frequency|count|rate|gender|race|ethnicit|asian|caucasian|black|white|hispanic|male|female|age\s*(\(|range)?|bmi|body\s*weight|hemoglobin|hba1c|egfr|region|country|smoking|diabetes\s+duration|renal\s+function|demographic|population|disposition|screening|randomized|completed|discontinued|due\s+to|hazard\s+ratio|p[\s\-]?value|confidence\s+interval|risk\s+reduction|treatment\s+(arm|group|difference)).*$`),

		// Statistical notation that indicates study data, not clinical conditions
		statisticalPatterns: regexp.MustCompile(`(?i)(^\d+\s*\([0-9.]+%?\)|confidence\s+interval|95%\s*CI|\(CI\)|n\s*=\s*\d+|p\s*[<>=]\s*0\.\d+|HR\s*=|OR\s*=|RR\s*=|\[[\d.,\s-]+\]|\([\d.,\s-]+,\s*[\d.,\s-]+\)|number\s*\(%\)|%\s*subjects)`),

		// Section labels that end with colon - these are headers not data
		sectionLabelPatterns: regexp.MustCompile(`(?i)^(clinical\s+impact|intervention|examples?|precaution|avoid|recommendations?|management|monitoring|contraindication|warning|note|see\s+also|refer\s+to|other\s+symptoms|medication\s+guide):?\s*$`),

		// Pure numeric or percentage values without context
		pureNumericPattern: regexp.MustCompile(`^[\d.,\s%<>≤≥=()+-]+$`),

		// Table formatting artifacts (including single words and footnotes)
		tableArtifactPatterns: regexp.MustCompile(`(?i)^(--|—|†|‡|\*+|#|a|b|c|d|e|f|NA|N/A|NR|ND|ns|CRNM\*?|DVT[†‡]?|VTE[†‡]?)$`),

		// Valid clinical conditions (medical terms)
		clinicalConditionPatterns: regexp.MustCompile(`(?i)(ache|emia|itis|osis|opathy|algia|penia|cytosis|trophy|plasia|ectomy|plasty|scopy|gram|rrhea|rrhage|phasia|phagia|plegia|paresis|kinesia|esthesia|cardia|tension|glycemia|lipidemia|kalemia|natremia|calcemia|uria|toxicity|failure|dysfunction|syndrome|disorder|disease|infection|infarct|stroke|death|bleeding|hemorrhage|thrombosis|embolism|arrhythmia|tachycardia|bradycardia|hypotension|hypertension|edema|nausea|vomiting|diarrhea|constipation|headache|dizziness|fatigue|insomnia|rash|pruritus|pain)`),

		// Valid drug name patterns (proper nouns, chemical suffixes)
		validDrugPatterns: regexp.MustCompile(`(?i)([A-Z][a-z]+(mab|nib|vir|pril|sartan|statin|olol|pine|zole|prazole|tidine|cycline|mycin|cillin|floxacin|sulfa|methasone|olone|asone)|[A-Z][a-z]{4,}(®|™)?|CYP[0-9][A-Z]\d*\s+(inhibitor|inducer|substrate)s?|NSAID|ACE\s+inhibitor|ARB|diuretic|anticoagulant|antiplatelet)`),
	}
}

// IsHeaderRow checks if a row looks like a table header.
// Checks ALL rows (not just row 0) since FDA AE tables often have multi-row headers:
// Row 0: "System Organ Class / Adverse Reaction"
// Row 1: "Drug N=2000 (%) / Placebo N=1999 (%)"
func (nf *NoiseFilter) IsHeaderRow(row []string, rowIndex int) bool {
	if len(row) == 0 {
		return false
	}

	headerCount := 0
	nEqualsCount := 0
	nEqualsPattern := regexp.MustCompile(`(?i)n\s*=\s*\d+`)

	for _, cell := range row {
		cell = strings.TrimSpace(cell)
		if nf.headerPatterns.MatchString(cell) {
			headerCount++
		}
		// Detect "N=2000" or "Drug N=1999 (%)" column headers
		if nEqualsPattern.MatchString(cell) {
			nEqualsCount++
		}
	}

	ratio := float64(headerCount) / float64(len(row))

	// Row 0: lower threshold (30%) — first row is very likely a header
	if rowIndex == 0 && ratio > 0.3 {
		return true
	}

	// Other rows: higher threshold (50%) to avoid false positives on data rows
	// where a single cell might match (e.g., "Depression" won't trigger if 1/4 cells)
	if rowIndex > 0 && ratio > 0.5 {
		return true
	}

	// Any row with 2+ "N=<number>" cells is a header row (e.g., "Drug N=2000 (%) | Placebo N=1999 (%)")
	if nEqualsCount >= 2 {
		return true
	}

	return false
}

// meddraSOCNames contains all 27 MedDRA System Organ Class names (lowercase).
// SOC headers appear as row labels in FDA AE tables but are CATEGORY headers, not
// adverse events. Both the official comma-containing forms and the comma-stripped
// forms seen in SPL tables are included.
var meddraSOCNames = map[string]bool{
	// Official MedDRA v27.1 SOC names (lowercase, with commas)
	"blood and lymphatic system disorders":                          true,
	"cardiac disorders":                                             true,
	"congenital, familial and genetic disorders":                    true,
	"ear and labyrinth disorders":                                   true,
	"endocrine disorders":                                           true,
	"eye disorders":                                                 true,
	"gastrointestinal disorders":                                    true,
	"general disorders and administration site conditions":          true,
	"hepatobiliary disorders":                                       true,
	"immune system disorders":                                       true,
	"infections and infestations":                                   true,
	"injury, poisoning and procedural complications":                true,
	"investigations":                                                true,
	"metabolism and nutrition disorders":                             true,
	"musculoskeletal and connective tissue disorders":               true,
	"neoplasms benign, malignant and unspecified (incl cysts and polyps)": true,
	"nervous system disorders":                                      true,
	"pregnancy, puerperium and perinatal conditions":                true,
	"psychiatric disorders":                                         true,
	"renal and urinary disorders":                                   true,
	"reproductive system and breast disorders":                      true,
	"respiratory, thoracic and mediastinal disorders":               true,
	"skin and subcutaneous tissue disorders":                        true,
	"social circumstances":                                          true,
	"surgical and medical procedures":                               true,
	"vascular disorders":                                            true,
	"product issues":                                                true,
	// Comma-stripped variants (as they often appear in SPL tables)
	"congenital familial and genetic disorders":                     true,
	"injury poisoning and procedural complications":                 true,
	"neoplasms benign malignant and unspecified (incl cysts and polyps)": true,
	"neoplasms benign malignant and unspecified":                    true,
	"pregnancy puerperium and perinatal conditions":                 true,
	"respiratory thoracic and mediastinal disorders":                true,
}

// IsNoiseCondition checks if a condition name is noise (not a real clinical condition)
func (nf *NoiseFilter) IsNoiseCondition(condition string) bool {
	condition = strings.TrimSpace(condition)

	// Empty or whitespace only
	if condition == "" {
		return true
	}

	// Too short or too long
	if len(condition) < 3 || len(condition) > 100 {
		return true
	}

	// Statistical pattern
	if nf.statisticalPatterns.MatchString(condition) {
		return true
	}

	// Section label
	if nf.sectionLabelPatterns.MatchString(condition) {
		return true
	}

	// Pure numeric
	if nf.pureNumericPattern.MatchString(condition) {
		return true
	}

	// Table artifact
	if nf.tableArtifactPatterns.MatchString(condition) {
		return true
	}

	// Header pattern
	if nf.headerPatterns.MatchString(condition) {
		return true
	}

	// === NEW: Explicit noise words for SAFETY_SIGNAL conditions ===
	lower := strings.ToLower(condition)

	// Single-word severity/category labels (not conditions themselves)
	singleWordNoise := map[string]bool{
		"major": true, "minor": true, "analysis": true, "total": true,
		"primary": true, "secondary": true, "endpoint": true, "outcome": true,
		"event": true, "events": true, "treatment": true, "placebo": true,
		"hemorrhagic": true, "intracranial": true, // These need context
	}
	if singleWordNoise[lower] {
		return true
	}

	// Fix 2: MedDRA SOC names as exact-match rejections.
	// SOC headers appear as row labels in FDA AE tables (e.g., "Gastrointestinal disorders").
	// These are CATEGORY headers, not adverse events. Without this filter, MedDRA fuzzy
	// matching can match them to real LLTs (e.g., "Gastrointestinal disorder" is a valid LLT).
	// All 27 MedDRA System Organ Classes:
	if meddraSOCNames[lower] {
		return true
	}

	// Table header phrases and study arm labels
	headerPhrases := []string{
		"number of", "subjects with", "patients with", "change at",
		"change from", "final visit", "at week", "at day",
		"n =", "n=", "% subjects", "% patients",
		"add-on to", "interference with", "coadministered",
		// Fix 6: Manufacturer/packaging noise from How Supplied section leaking into AE tables
		"manufactured by", "manufactured for", "distributed by", "marketed by",
		"packaged by", "revised:", "revised date", "initial u.s. approval",
	}
	for _, phrase := range headerPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}

	// Fix 6: Corporate entity names (e.g., "Apotex Inc", "Teva Pharmaceuticals")
	// These appear in How Supplied/manufacturer sections and should never be AEs.
	corporateSuffixes := []string{
		" inc", " inc.", " corp", " corp.", " ltd", " ltd.",
		" llc", " plc", " pharmaceuticals", " pharma",
		" laboratories", " labs", " therapeutics",
	}
	for _, suffix := range corporateSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}

	// Contains footnote markers at end of text (like "DVT†", "CRNM*")
	footnoteMarkers := []string{"†", "‡", "*", "§", "¶"}
	for _, marker := range footnoteMarkers {
		if strings.HasSuffix(condition, marker) {
			return true
		}
	}

	return false
}

// IsValidClinicalCondition checks if text is a valid clinical condition
func (nf *NoiseFilter) IsValidClinicalCondition(condition string) bool {
	condition = strings.TrimSpace(condition)

	// Must not be noise first
	if nf.IsNoiseCondition(condition) {
		return false
	}

	// Check for clinical term patterns
	if nf.clinicalConditionPatterns.MatchString(condition) {
		return true
	}

	// Check for proper noun format (starts with capital, reasonable length)
	if len(condition) >= 4 && len(condition) <= 60 {
		// Starts with capital letter
		if condition[0] >= 'A' && condition[0] <= 'Z' {
			// Contains mostly letters
			letterCount := 0
			for _, c := range condition {
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == ' ' || c == '-' {
					letterCount++
				}
			}
			if float64(letterCount)/float64(len(condition)) > 0.7 {
				return true
			}
		}
	}

	return false
}

// IsNoiseInteractant checks if an interactant name is noise
func (nf *NoiseFilter) IsNoiseInteractant(interactant string) bool {
	interactant = strings.TrimSpace(interactant)

	// Too short or too long
	if len(interactant) < 3 || len(interactant) > 100 {
		return true
	}

	// Section label (ends with colon or matches section patterns)
	if strings.HasSuffix(interactant, ":") {
		return true
	}
	if nf.sectionLabelPatterns.MatchString(interactant) {
		return true
	}

	// Pure numeric
	if nf.pureNumericPattern.MatchString(interactant) {
		return true
	}

	// Table artifact
	if nf.tableArtifactPatterns.MatchString(interactant) {
		return true
	}

	// === NEW: Full sentence detection (interactants should not be sentences) ===
	lower := strings.ToLower(interactant)

	// Contains question mark - it's a question, not a drug
	if strings.Contains(interactant, "?") {
		return true
	}

	// Starts with common sentence starters - not a drug name
	sentenceStarters := []string{
		"the ", "this ", "that ", "these ", "those ",
		"what ", "how ", "when ", "where ", "why ", "which ",
		"if ", "for ", "with ", "your ", "you ", "it ",
		"other ", "some ", "any ", "all ", "most ",
	}
	for _, starter := range sentenceStarters {
		if strings.HasPrefix(lower, starter) {
			return true
		}
	}

	// Contains period followed by space (multi-sentence)
	if strings.Contains(interactant, ". ") {
		return true
	}

	// Too many words (drug names rarely exceed 5 words)
	wordCount := len(strings.Fields(interactant))
	if wordCount > 6 {
		return true
	}

	// === NEW: Side effect detection (side effects are not interactants) ===
	sideEffectPatterns := []string{
		"nausea", "dizziness", "headache", "vomiting", "diarrhea",
		"constipation", "fatigue", "drowsiness", "trembling", "twitching",
		"blurred vision", "dry mouth", "loss of appetite", "excessive urination",
		"muscle", "weakness", "discomfort", "agitation", "appetite",
		"problems", "symptoms", "side effects",
	}
	for _, effect := range sideEffectPatterns {
		if strings.Contains(lower, effect) {
			return true
		}
	}

	// Collapse whitespace/newlines (catches multi-line table headers like "Drug\n   Class")
	lower = strings.Join(strings.Fields(lower), " ")

	// Common noise words and table column headers
	noiseWords := []string{
		"clinical impact", "intervention", "examples", "precautions",
		"avoid", "see", "refer", "note", "table", "figure",
		"medication guide", "food and drug", "fda", "approved",
		// Table column headers that slip through as interactant names
		"drug class", "specific drug", "drug name",
		"concomitant drug", "concomitant medication",
		"interacting drug", "object drug", "precipitant drug",
		"positive urine", "urine glucose", "laboratory test",
		"coadministered drug", "pharmacodynamic interaction",
		"pharmacokinetic interaction", "interference with",
	}

	// Exact-match noise: single generic words that are column headers, not drugs
	exactNoiseWords := map[string]bool{
		"enzyme": true, "enzymes": true,
		"inhibitors": true, "inducers": true, // bare column headers; "CYP3A4 Inhibitors" is fine
		"drug": true, "drugs": true, "class": true,
		"effect": true, "recommendation": true, "management": true,
		"result": true, "comment": true, "comments": true,
	}
	if exactNoiseWords[lower] {
		return true
	}
	for _, noise := range noiseWords {
		if strings.HasPrefix(lower, noise) {
			return true
		}
	}

	return false
}

// IsValidDrugName checks if text looks like a valid drug/substance name
func (nf *NoiseFilter) IsValidDrugName(name string) bool {
	name = strings.TrimSpace(name)

	// Must not be noise first
	if nf.IsNoiseInteractant(name) {
		return false
	}

	// Check for drug name patterns
	if nf.validDrugPatterns.MatchString(name) {
		return true
	}

	// Check for proper noun format (drug names typically start with capital)
	if len(name) >= 4 && len(name) <= 80 {
		if name[0] >= 'A' && name[0] <= 'Z' {
			// Contains mostly letters
			letterCount := 0
			for _, c := range name {
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == ' ' || c == '-' || c == '/' || c == ',' {
					letterCount++
				}
			}
			if float64(letterCount)/float64(len(name)) > 0.7 {
				return true
			}
		}
	}

	return false
}

// numberPrefixPattern matches "1. ", "2. ", "10. " etc. at start of condition names
// (furosemide's "1. aplastic anemia" → "aplastic anemia")
var numberPrefixPattern = regexp.MustCompile(`^\d+\.\s*`)

// camelCaseBoundary detects lowercase→uppercase transitions mid-word
// that indicate concatenation artifacts (e.g., "AmblyopiaAmblyopia was often described...")
// Splits at the boundary and takes only the first segment.
var camelCaseBoundary = regexp.MustCompile(`([a-z])([A-Z])`)

// FilterConditionName cleans and validates a condition name, returns empty if noise
func (nf *NoiseFilter) FilterConditionName(condition string) string {
	condition = strings.TrimSpace(condition)

	// Early exit for empty/short strings
	if condition == "" || len(condition) < 3 {
		return ""
	}

	// P0.5: Reject entries >100 characters (multi-cell concatenation artifacts)
	if len(condition) > 100 {
		return ""
	}

	// P0.5: Handle multi-line entries (carvedilol's "Gastrointestinal\nDiarrhea\nNausea")
	// Take first non-SOC-header line as the actual condition
	if strings.Contains(condition, "\n") {
		lines := strings.Split(condition, "\n")
		condition = "" // reset, find first valid line
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Skip SOC header lines (e.g., "Gastrointestinal", "Nervous system")
			if nf.IsNoiseCondition(line) {
				continue
			}
			condition = line
			break
		}
		if condition == "" {
			return ""
		}
	}

	// Fix 5: CamelCase concatenation artifact detection.
	// Gabapentin has "AmblyopiaAmblyopia was often described as blurred vision" and
	// "AmblyopiaReported as blurred vision" — these are cell concatenation artifacts.
	// If the term contains a lowercase→uppercase transition mid-word, split at that
	// boundary and take only the first segment (the actual condition name).
	if camelCaseBoundary.MatchString(condition) {
		// Split at the first camelCase boundary
		parts := camelCaseBoundary.Split(condition, 2)
		if len(parts) > 0 {
			// Re-append the captured lowercase letter (it was in group 1)
			loc := camelCaseBoundary.FindStringIndex(condition)
			if loc != nil {
				condition = condition[:loc[0]+1] // Include the lowercase char before boundary
			}
		}
		condition = strings.TrimSpace(condition)
		if condition == "" || len(condition) < 3 {
			return ""
		}
	}

	// P0.5: Strip number prefix (furosemide's "1. aplastic anemia" → "aplastic anemia")
	condition = numberPrefixPattern.ReplaceAllString(condition, "")
	condition = strings.TrimSpace(condition)

	// Remove trailing punctuation
	condition = strings.TrimRight(condition, ":;,.")

	// Remove footnote markers from end
	for _, marker := range []string{"†", "‡", "*", "§", "¶"} {
		condition = strings.TrimSuffix(condition, marker)
	}
	condition = strings.TrimSpace(condition)

	// Check again after trimming
	if condition == "" || len(condition) < 3 {
		return ""
	}

	// Check if it's noise
	if nf.IsNoiseCondition(condition) {
		return ""
	}

	// Check if it's a valid clinical condition
	if nf.IsValidClinicalCondition(condition) {
		return condition
	}

	// Return empty if uncertain (conservative filtering)
	return ""
}

// FilterInteractantName cleans and validates an interactant name, returns empty if noise
func (nf *NoiseFilter) FilterInteractantName(interactant string) string {
	interactant = strings.TrimSpace(interactant)

	// Remove trailing punctuation
	interactant = strings.TrimRight(interactant, ":;,.")

	// Check if it's noise
	if nf.IsNoiseInteractant(interactant) {
		return ""
	}

	// Check if it's a valid drug name
	if nf.IsValidDrugName(interactant) {
		return interactant
	}

	// Return empty if uncertain
	return ""
}

// NewContentParser creates a new parser with precompiled patterns
func NewContentParser() *ContentParser {
	return &ContentParser{
		// GFR/CrCl patterns: "CrCl ≥60", "eGFR 30-59", "GFR <30 mL/min"
		gfrPattern: regexp.MustCompile(`(?i)(e?gfr|crcl|creatinine\s+clearance)\s*([<>≤≥]=?|)\s*(\d+)\s*(?:[-–to]+\s*(\d+))?\s*(ml/min|ml/min/1\.73)?`),

		// Severity patterns for safety signals
		severityPattern: regexp.MustCompile(`(?i)(fatal|death|life[- ]?threatening|serious|severe|moderate|mild|minor)`),

		// Drug interaction mechanism patterns
		mechanismPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(cyp[0-9][a-z]\d*)\s*(inhibit|induc|substrate)`),
			regexp.MustCompile(`(?i)reduce[sd]?\s+(tubular\s+)?secretion`),
			regexp.MustCompile(`(?i)increase[sd]?\s+(plasma\s+)?(concentration|level|exposure)`),
			regexp.MustCompile(`(?i)decrease[sd]?\s+(plasma\s+)?(concentration|level|exposure)`),
			regexp.MustCompile(`(?i)(additive|synergistic|antagonistic)\s+effect`),
			regexp.MustCompile(`(?i)compete[sd]?\s+(for\s+)?(binding|transport)`),
		},

		// NDC code pattern: "12345-678-90" or "12345-6789-01"
		ndcPattern: regexp.MustCompile(`(\d{4,5}-\d{3,4}-\d{1,2})`),

		// Dose patterns: "500 mg", "1000mg", "50%"
		dosePattern:    regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(mg|g|mcg|units?|ml|mg/kg)`),
		percentPattern: regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`),

		// Initialize noise filter for quality improvement
		noisePatterns: NewNoiseFilter(),
	}
}

// SetAdverseEventNormalizer configures the MedDRA-based adverse event normalizer.
// When set, ParseSafetySignal will use MedDRA dictionary lookup instead of regex
// for deterministic term validation and FAERS-compatible PT codes.
//
// Phase 3 Issues 2+3 Fix:
// - Issue 2: Returns official MedDRA PT codes for FDA FAERS compatibility
// - Issue 3: Replaces 47% regex ceiling with 80,000+ term dictionary lookup
func (cp *ContentParser) SetAdverseEventNormalizer(normalizer terminology.AdverseEventNormalizer) {
	cp.aeNormalizer = normalizer
}

// HasAdverseEventNormalizer returns true if MedDRA normalizer is configured and loaded.
func (cp *ContentParser) HasAdverseEventNormalizer() bool {
	return cp.aeNormalizer != nil && cp.aeNormalizer.IsLoaded()
}

// ParseResult contains parsed content and metadata
type ParseResult struct {
	Content    interface{} // Structured content (KB-specific structs with correct JSON tags)
	Confidence float64     // Parsing confidence (0.0-1.0)
	ParsedRows int         // Number of rows successfully parsed
	TotalRows  int         // Total rows in source table
	Warnings   []string    // Non-fatal parsing issues
}

// =============================================================================
// KB-SPECIFIC OUTPUT STRUCTS (JSON field names match KB view expectations)
// These structs ensure content->>'fieldName' queries in KB views work correctly
// =============================================================================

// KBOrganImpairmentContent matches kb1_renal_dosing view schema
// Fields: organ, impairmentLevel, egfrRangeLow, egfrRangeHigh, ckdStage, action
type KBOrganImpairmentContent struct {
	Organ           string  `json:"organ"`            // RENAL, HEPATIC
	ImpairmentLevel string  `json:"impairmentLevel"`  // MILD, MODERATE, SEVERE, ESRD
	EGFRRangeLow    float64 `json:"egfrRangeLow"`     // Lower GFR bound
	EGFRRangeHigh   float64 `json:"egfrRangeHigh"`    // Upper GFR bound
	CKDStage        string  `json:"ckdStage"`         // CKD stage (1, 2, 3a, 3b, 4, 5)
	Action          string  `json:"action"`           // REDUCE_DOSE, AVOID, CONTRAINDICATED, MONITOR
	DoseAdjustment  string  `json:"doseAdjustment"`   // e.g., "50% reduction"
	MaxDose         string  `json:"maxDose"`          // Maximum allowed dose
	MaxDoseUnit     string  `json:"maxDoseUnit"`      // Unit for max dose
	Rationale       string  `json:"rationale"`        // Clinical rationale
	RawText         string  `json:"rawText"`          // Original extracted text
}

// KBSafetySignalContent matches kb4_safety_signals view schema
// Fields: signalType, severity, conditionCode, conditionName, description, recommendation, requiresMonitor
//
// Phase 3 Enhancement: Added MedDRA fields for FAERS compatibility (Issue 2)
// MedDRA PT codes enable FDA pharmacovigilance integration and cross-drug safety analysis.
type KBSafetySignalContent struct {
	SignalType      string `json:"signalType"`      // BOXED_WARNING, CONTRAINDICATION, WARNING, PRECAUTION
	Severity        string `json:"severity"`        // CRITICAL, HIGH, MEDIUM, LOW
	ConditionCode   string `json:"conditionCode"`   // SNOMED code if available
	ConditionName   string `json:"conditionName"`   // Clinical condition name (normalized if MedDRA available)
	Description     string `json:"description"`     // Full warning description
	Recommendation  string `json:"recommendation"`  // Clinical recommendation
	RequiresMonitor bool   `json:"requiresMonitor"` // Whether monitoring is required
	Frequency       string `json:"frequency"`       // Occurrence frequency if known
	FrequencyBand   string `json:"frequencyBand,omitempty"` // Standardized MedDRA frequency band: VERY_COMMON, COMMON, UNCOMMON, RARE, VERY_RARE

	// Phase 3: MedDRA fields for FAERS compatibility
	MedDRAPT        string  `json:"meddraPT,omitempty"`        // MedDRA Preferred Term code (e.g., "10028813" for Nausea)
	MedDRAName      string  `json:"meddraName,omitempty"`      // MedDRA PT name (official terminology)
	MedDRALLT       string  `json:"meddraLLT,omitempty"`       // MedDRA Lowest Level Term code (if matched at LLT)
	MedDRASOC       string  `json:"meddraSOC,omitempty"`       // MedDRA System Organ Class code
	MedDRASOCName   string  `json:"meddraSOCName,omitempty"`   // MedDRA SOC name (e.g., "Gastrointestinal disorders")
	SNOMEDCode      string  `json:"snomedCode,omitempty"`      // SNOMED CT code from official MedDRA-SNOMED mapping
	TermConfidence  float64 `json:"termConfidence,omitempty"`  // Confidence in term normalization (1.0 = exact match)
}

// KBInteractionContent matches kb5_interactions view schema
type KBInteractionContent struct {
	InteractionType string `json:"interactionType"`           // DRUG_DRUG, DRUG_FOOD, DRUG_ALCOHOL, DRUG_LAB
	PrecipitantDrug string `json:"precipitantDrug,omitempty"` // Index drug (the drug whose label we're reading)
	ObjectDrug      string `json:"objectDrug,omitempty"`      // Interacting drug (same as interactantName for DDI)
	InteractantName string `json:"interactantName"`           // Name of interacting substance
	Severity        string `json:"severity"`                  // SEVERE, MODERATE, MILD
	Mechanism       string `json:"mechanism"`                 // Pharmacological mechanism
	ClinicalEffect  string `json:"clinicalEffect"`            // Clinical consequence
	Management      string `json:"management"`                // Management recommendation
	SourcePhrase    string `json:"sourcePhrase,omitempty"`    // Grounding text from table row for reviewer
}

// KBReproductiveSafetyContent for pregnancy/lactation facts
type KBReproductiveSafetyContent struct {
	Category          string   `json:"category"`          // PREGNANCY, LACTATION, FERTILITY
	RiskLevel         string   `json:"riskLevel"`         // CONTRAINDICATED, HIGH_RISK, CAUTION, COMPATIBLE
	FDACategory       string   `json:"fdaCategory"`       // Legacy category (A, B, C, D, X)
	PLLRSummary       string   `json:"pllrSummary"`       // PLLR narrative summary
	ExcretionInMilk   *bool    `json:"excretionInMilk"`   // Whether excreted in breast milk
	RelativeInfantDose *float64 `json:"relativeInfantDose"` // RID percentage
}

// KBFormularyContent for NDC/packaging facts
type KBFormularyContent struct {
	NDCCode      string `json:"ndcCode"`      // National Drug Code
	PackageForm  string `json:"packageForm"`  // tablet, capsule, etc.
	Strength     string `json:"strength"`     // e.g., "500 mg"
	PackageSize  string `json:"packageSize"`  // e.g., "100 tablets"
	Manufacturer string `json:"manufacturer"` // Manufacturer name
}

// KBLabReferenceContent for monitoring facts
type KBLabReferenceContent struct {
	LOINCCode           string   `json:"loincCode"`           // LOINC code
	LabName             string   `json:"labName"`             // Lab test name
	ReferenceRangeLow   *float64 `json:"referenceRangeLow"`   // Normal range low
	ReferenceRangeHigh  *float64 `json:"referenceRangeHigh"`  // Normal range high
	Unit                string   `json:"unit"`                // Unit of measure
	MonitoringFrequency string   `json:"monitoringFrequency"` // How often to monitor
	BaselineRequired    bool     `json:"baselineRequired"`    // If baseline needed
}

// Parse dispatches to the appropriate parser based on fact type
func (cp *ContentParser) Parse(table *dailymed.ExtractedTable, loincCode, factType, indexDrugName string) (*ParseResult, error) {
	switch factType {
	case "ORGAN_IMPAIRMENT":
		return cp.ParseOrganImpairment(table)
	case "SAFETY_SIGNAL":
		return cp.ParseSafetySignal(table, loincCode)
	case "INTERACTION":
		return cp.ParseInteraction(table, indexDrugName)
	case "REPRODUCTIVE_SAFETY":
		return cp.ParseReproductiveSafety(table, loincCode)
	case "FORMULARY":
		return cp.ParseFormulary(table)
	case "LAB_REFERENCE":
		return cp.ParseLabReference(table)
	default:
		// Unknown fact type - return raw table with low confidence
		return &ParseResult{
			Content:    table,
			Confidence: 0.3,
			TotalRows:  len(table.Rows),
			Warnings:   []string{"Unknown fact type, returning raw table"},
		}, nil
	}
}

// ParseOrganImpairment extracts renal/hepatic dosing adjustments
// Expected output matches kb1_renal_dosing view schema (organ, impairmentLevel, egfrRangeLow, egfrRangeHigh)
func (cp *ContentParser) ParseOrganImpairment(table *dailymed.ExtractedTable) (*ParseResult, error) {
	var contents []KBOrganImpairmentContent
	var warnings []string
	parsedRows := 0

	// Determine organ type from table context
	organSystem := cp.detectOrganSystem(table)

	// Find relevant column indices
	colMap := cp.mapColumns(table.Headers, []string{
		"gfr", "crcl", "creatinine", "clearance", "egfr", // threshold columns
		"dose", "dosage", "adjustment", "recommendation", // action columns
		"stage", "severity", "impairment", "level",       // level columns
	})

	for _, row := range table.Rows {
		if len(row) == 0 {
			continue
		}

		// KB-specific output struct with exact field names for KB views
		content := KBOrganImpairmentContent{
			Organ: organSystem, // Matches content->>'organ' in kb1_renal_dosing
		}

		// Search all cells for GFR values
		for i, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}

			// Check for GFR/CrCl values
			if matches := cp.gfrPattern.FindStringSubmatch(cell); matches != nil {
				// Parse operator and values
				op := matches[2]
				val1, _ := strconv.ParseFloat(matches[3], 64)
				val2Str := matches[4]

				if val2Str != "" {
					// Range: "30-59" → egfrRangeLow=30, egfrRangeHigh=59
					val2, _ := strconv.ParseFloat(val2Str, 64)
					content.EGFRRangeLow = val1
					content.EGFRRangeHigh = val2
					content.ImpairmentLevel = cp.classifyImpairmentLevel(val1, val2)
					content.CKDStage = cp.ckdStageFromGFR(val1, val2)
				} else {
					// Single value with operator: "<30" → egfrRangeHigh=30, egfrRangeLow=0
					switch cp.normalizeOperator(op) {
					case "LT", "LTE":
						content.EGFRRangeLow = 0
						content.EGFRRangeHigh = val1
					case "GT", "GTE":
						content.EGFRRangeLow = val1
						content.EGFRRangeHigh = 999 // High bound
					default:
						content.EGFRRangeLow = val1
						content.EGFRRangeHigh = val1
					}
					content.ImpairmentLevel = cp.classifyImpairmentBySingle(op, val1)
					content.CKDStage = cp.ckdStageFromGFR(content.EGFRRangeLow, content.EGFRRangeHigh)
				}
				content.RawText = cell
			}

			// Check for dose adjustments in other columns
			if i != colMap["gfr"] && i != colMap["crcl"] {
				if doseMatch := cp.dosePattern.FindStringSubmatch(cell); doseMatch != nil {
					if content.MaxDose == "" {
						content.MaxDose = doseMatch[1] // Numeric value
						content.MaxDoseUnit = doseMatch[2] // Unit
					}
				}
				if percentMatch := cp.percentPattern.FindStringSubmatch(cell); percentMatch != nil {
					content.DoseAdjustment = percentMatch[0] + " reduction"
				}
				// Capture recommendation text as action
				if cp.isRecommendationText(cell) {
					content.Action = cp.extractAction(cell)
					if content.Rationale == "" {
						content.Rationale = cell
					}
				}
			}
		}

		// Only add if we found threshold information
		if content.EGFRRangeLow > 0 || content.EGFRRangeHigh > 0 || content.ImpairmentLevel != "" {
			contents = append(contents, content)
			parsedRows++
		}
	}

	// Calculate confidence based on parsing success
	confidence := 0.5
	if len(table.Rows) > 0 {
		confidence = 0.5 + (float64(parsedRows)/float64(len(table.Rows)))*0.4
	}
	if parsedRows == 0 {
		warnings = append(warnings, "No GFR/CrCl thresholds found in table")
		confidence = 0.3
	}

	return &ParseResult{
		Content:    contents,
		Confidence: confidence,
		ParsedRows: parsedRows,
		TotalRows:  len(table.Rows),
		Warnings:   warnings,
	}, nil
}

// ParseSafetySignal extracts warnings, contraindications, and adverse reactions
// Expected output matches kb4_safety_signals view schema (signalType, severity, conditionName, etc.)
//
// Phase 3 Enhancement: Uses MedDRA dictionary lookup instead of regex (Issues 2+3).
// - Issue 2 Fix: Returns official MedDRA PT codes for FAERS compatibility
// - Issue 3 Fix: Uses MedDRA dictionary (80,000+ terms) instead of regex ceiling (47%)
//   If term is in MedDRA → valid clinical term with PT code
//   If term is NOT in MedDRA → noise (filtered out deterministically)
func (cp *ContentParser) ParseSafetySignal(table *dailymed.ExtractedTable, loincCode string) (*ParseResult, error) {
	return cp.ParseSafetySignalWithContext(context.Background(), table, loincCode)
}

// ParseSafetySignalWithContext is the context-aware version of ParseSafetySignal.
// Use this when you need to pass a context for timeout/cancellation.
func (cp *ContentParser) ParseSafetySignalWithContext(ctx context.Context, table *dailymed.ExtractedTable, loincCode string) (*ParseResult, error) {
	var contents []KBSafetySignalContent
	var warnings []string
	parsedRows := 0
	skippedNoise := 0
	meddraMatches := 0

	// Determine signal type from LOINC code
	signalType := cp.loincToSignalType(loincCode)

	for rowIdx, row := range table.Rows {
		if len(row) == 0 {
			continue
		}

		// NOISE FILTER: Skip header rows
		if cp.noisePatterns.IsHeaderRow(row, rowIdx) {
			skippedNoise++
			continue
		}

		// Combine row into text for analysis
		rowText := strings.Join(row, " ")

		// KB-specific struct with exact field names for kb4_safety_signals view
		content := KBSafetySignalContent{
			SignalType: signalType,                   // Matches content->>'signalType'
			Severity:   cp.extractSeverity(rowText),  // Matches content->>'severity'
		}

		// Extract condition/reaction name (usually first significant column)
		// Phase 3: Use MedDRA dictionary lookup instead of regex
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "" || len(cell) < 3 {
				continue
			}

			// NOISE FILTER: Skip numeric/percent values
			if cp.isNumericOrPercent(cell) {
				continue
			}

			// Fix 1: Noise filter MUST run BEFORE MedDRA normalization.
			// Without this ordering, garbage terms like "Female condom", "Whiteheads",
			// "Gender identity disorder NOS" reach MedDRA fuzzy matching, match real
			// MedDRA LLTs, and get auto-approved. The noise filter is the gatekeeper.
			if content.ConditionName == "" {
				// Step 1: Noise filter first — reject garbage before it reaches MedDRA
				filteredCondition := cp.noisePatterns.FilterConditionName(cell)
				if filteredCondition == "" {
					// Noise filter rejected — do NOT pass to MedDRA
					continue
				}

				// Step 2: Term survived noise filter — try MedDRA for validation + enrichment
				if cp.aeNormalizer != nil && cp.aeNormalizer.IsLoaded() {
					normalized, err := cp.aeNormalizer.Normalize(ctx, filteredCondition)
					if err == nil && normalized.IsValidTerm {
						// MedDRA confirmed — high confidence, enriched with codes
						content.ConditionName = normalized.CanonicalName
						content.MedDRAPT = normalized.MedDRAPT
						content.MedDRAName = normalized.MedDRAName
						content.MedDRALLT = normalized.MedDRALLT
						content.MedDRASOC = normalized.MedDRASOC
						content.MedDRASOCName = normalized.MedDRASOCName
						content.SNOMEDCode = normalized.SNOMEDCode
						content.TermConfidence = normalized.Confidence
						meddraMatches++
					} else {
						// P1.4: Passed noise filter but MedDRA miss — keep with lower confidence
						content.ConditionName = filteredCondition
						content.TermConfidence = 0.55
					}
				} else {
					// No MedDRA available — noise-filtered term accepted at base confidence
					content.ConditionName = filteredCondition
				}
			}

			// Look for frequency/incidence data (but don't use as condition)
			if content.ConditionName != "" {
				if strings.Contains(strings.ToLower(cell), "common") ||
					strings.Contains(strings.ToLower(cell), "rare") ||
					strings.Contains(strings.ToLower(cell), "frequent") {
					if content.Frequency == "" {
						content.Frequency = cell
					}
				}
			}
		}

		// P0.3: Percentage-based frequency extraction from adjacent cells
		// If keyword check didn't find frequency, scan row for percentage values
		// FDA AE tables put frequency in adjacent column: ["Headache", "12.3%", "8.1%"]
		if content.Frequency == "" && content.ConditionName != "" {
			for _, adjacentCell := range row {
				trimmed := strings.TrimSpace(adjacentCell)
				// Skip the cell that became conditionName and empty cells
				if trimmed == "" || trimmed == content.ConditionName {
					continue
				}
				if matches := cp.percentPattern.FindStringSubmatch(trimmed); len(matches) > 0 {
					content.Frequency = trimmed
					break
				}
			}
		}

		// P1.5c: Normalize frequency to MedDRA-standard band
		// This standardizes both percentage values ("12.3%" → VERY_COMMON)
		// and qualitative keywords ("rare" → RARE with 0.01-0.1% range)
		if content.Frequency != "" {
			content.FrequencyBand, content.Frequency = normalizeFrequencyBand(content.Frequency)
		}

		// NOISE FILTER: Skip if no valid condition was found
		if content.ConditionName == "" {
			skippedNoise++
			continue
		}

		// Fix: Populate conditionCode from MedDRA PT when available
		// This ensures downstream systems (KB-19, FAERS aggregation) have a canonical code
		if content.ConditionCode == "" && content.MedDRAPT != "" {
			content.ConditionCode = content.MedDRAPT
		}

		// Misclassification filter: biomarker MedDRA PTs (SOC = "Investigations")
		// are lab values, not adverse events — skip them as SAFETY_SIGNAL
		if isBiomarkerSOC(content.MedDRASOCName) {
			skippedNoise++
			continue
		}

		// Determine if monitoring is required based on severity
		// Matches content->>'requiresMonitor'
		content.RequiresMonitor = content.Severity == "CRITICAL" || content.Severity == "HIGH"

		// Extract recommendation based on signal type
		content.Recommendation = cp.extractRecommendation(rowText, signalType)

		// Build a clinical description from structured fields instead of raw table row.
		// Raw rowText contains concatenated cells like "Glucose <54 mg/dL [n (%)] 1 (0.4) – 1 (0.4)"
		// which is not useful for clinical display. Use the MedDRA-normalized condition name
		// with frequency and severity context.
		content.Description = buildSafetyDescription(content)
		contents = append(contents, content)
		parsedRows++
	}

	// Adjust confidence based on noise filtering and MedDRA matching
	confidence := 0.6
	totalProcessed := len(table.Rows) - skippedNoise
	if totalProcessed > 0 {
		// Higher confidence when we successfully filtered noise
		confidence = 0.7 + (float64(parsedRows)/float64(totalProcessed))*0.2
	}
	// Boost confidence when MedDRA dictionary was used (deterministic matching)
	if meddraMatches > 0 && parsedRows > 0 {
		meddraRatio := float64(meddraMatches) / float64(parsedRows)
		confidence = confidence + (meddraRatio * 0.1) // Up to 0.1 boost for full MedDRA coverage
		if confidence > 1.0 {
			confidence = 1.0
		}
	}
	if parsedRows == 0 {
		warnings = append(warnings, "No valid safety signals extracted after noise filtering")
		confidence = 0.3
	}
	if skippedNoise > 0 {
		warnings = append(warnings, "Filtered "+strconv.Itoa(skippedNoise)+" noise rows (headers, statistics, artifacts)")
	}
	if meddraMatches > 0 {
		warnings = append(warnings, "MedDRA normalized "+strconv.Itoa(meddraMatches)+" terms with official PT codes (FAERS compatible)")
	}

	return &ParseResult{
		Content:    contents,
		Confidence: confidence,
		ParsedRows: parsedRows,
		TotalRows:  len(table.Rows),
		Warnings:   warnings,
	}, nil
}

// ParseInteraction extracts drug-drug interaction data
// Expected output matches kb5_interactions view schema (interactionType, interactantName, severity, etc.)
// IMPROVED: Now includes noise filtering, PK table detection, and source phrase generation
func (cp *ContentParser) ParseInteraction(table *dailymed.ExtractedTable, indexDrugName string) (*ParseResult, error) {
	var contents []KBInteractionContent
	var warnings []string
	parsedRows := 0
	skippedNoise := 0

	// Find column mapping
	colMap := cp.mapColumns(table.Headers, []string{
		"drug", "interactant", "concomitant", "medication", // interactant columns
		"effect", "result", "consequence", "clinical",      // effect columns
		"recommendation", "management", "action",           // management columns
	})

	// Detect PK-style tables: columns with numeric % change data (e.g., digoxin DDI table)
	// These have headers like ["Drug", "Increase in digoxin Cmax", "Increase in digoxin AUC"]
	pkColIndices := detectPKColumns(table.Headers)

	// Track the last "section header row" for context grouping
	// Tables like digoxin's have sub-headers: "Digoxin concentrations increased >50%"
	var currentSectionHeader string

	for rowIdx, row := range table.Rows {
		if len(row) == 0 {
			continue
		}

		// NOISE FILTER: Skip header rows
		if cp.noisePatterns.IsHeaderRow(row, rowIdx) {
			skippedNoise++
			continue
		}

		rowText := strings.Join(row, " ")

		// Detect in-table section headers (e.g., "Digoxin concentrations increased >50%")
		// These are rows where column 0 has descriptive text and other columns are empty/NA
		if isInteractionSectionHeader(row, indexDrugName) {
			currentSectionHeader = strings.TrimSpace(row[0])
			skippedNoise++
			continue
		}

		// KB-specific struct with exact field names for kb5_interactions view
		content := KBInteractionContent{
			InteractionType: "DRUG_DRUG", // Default type; matches content->>'interactionType'
			PrecipitantDrug: indexDrugName,
		}

		// Extract interactant name with noise filtering
		for i, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}

			// First column often contains drug name
			if i == 0 || colMap["drug"] == i || colMap["interactant"] == i {
				if content.InteractantName == "" {
					// NOISE FILTER: Use filter to validate interactant name
					filteredName := cp.noisePatterns.FilterInteractantName(cell)
					if filteredName != "" {
						content.InteractantName = filteredName // Matches content->>'interactantName'
					}
				}
			}

			// Look for clinical effect descriptions (skip if it's noise)
			// FIX: check map key existence to avoid 0-index false match
			if effectIdx, ok := colMap["effect"]; ok && effectIdx == i && content.ClinicalEffect == "" {
				if !cp.noisePatterns.IsNoiseCondition(cell) && len(cell) > 5 {
					content.ClinicalEffect = cell
				}
			}
			if clinIdx, ok := colMap["clinical"]; ok && clinIdx == i && content.ClinicalEffect == "" {
				if !cp.noisePatterns.IsNoiseCondition(cell) && len(cell) > 5 {
					content.ClinicalEffect = cell
				}
			}

			// Look for management recommendations
			// FIX: same map key existence check
			if recIdx, ok := colMap["recommendation"]; ok && recIdx == i && content.Management == "" {
				if !cp.noisePatterns.IsNoiseCondition(cell) && len(cell) > 5 {
					content.Management = cell
				}
			}
			if mgmtIdx, ok := colMap["management"]; ok && mgmtIdx == i && content.Management == "" {
				if !cp.noisePatterns.IsNoiseCondition(cell) && len(cell) > 5 {
					content.Management = cell
				}
			}
		}

		// NOISE FILTER: Skip if no valid interactant name found
		if content.InteractantName == "" {
			skippedNoise++
			continue
		}

		// FIX 1: For PK-style tables, build clinicalEffect from numeric columns
		if content.ClinicalEffect == "" && len(pkColIndices) > 0 {
			content.ClinicalEffect = buildPKClinicalEffect(row, table.Headers, pkColIndices, indexDrugName, currentSectionHeader)
		}

		// FIX 1: Use section header as context for severity when no other signals
		if currentSectionHeader != "" && content.ClinicalEffect == "" {
			content.ClinicalEffect = currentSectionHeader
		}

		// Set objectDrug = interactant for drug-drug interactions
		content.ObjectDrug = content.InteractantName

		// Extract mechanism from row text
		content.Mechanism = cp.extractMechanism(rowText) // Matches content->>'mechanism'

		// Determine severity from text (include section header context)
		severityText := rowText
		if currentSectionHeader != "" {
			severityText = currentSectionHeader + " " + rowText
		}
		content.Severity = cp.extractInteractionSeverity(severityText) // Matches content->>'severity'

		// Check for specific interaction types
		lowerRow := strings.ToLower(rowText)
		if strings.Contains(lowerRow, "food") || strings.Contains(lowerRow, "grapefruit") {
			content.InteractionType = "DRUG_FOOD"
		} else if strings.Contains(lowerRow, "alcohol") {
			content.InteractionType = "DRUG_ALCOHOL"
		} else if strings.Contains(lowerRow, "lab") ||
			strings.Contains(lowerRow, "test") {
			content.InteractionType = "DRUG_LAB"
		}

		// FIX 3: Build sourcePhrase from table headers + row for grounding
		content.SourcePhrase = buildInteractionSourcePhrase(table.Headers, row, currentSectionHeader)

		// Enrich very short sourcePhrases (e.g., just "Lithium") with drug context
		if len(content.SourcePhrase) < 30 && content.InteractantName != "" {
			content.SourcePhrase = fmt.Sprintf("%s interaction with %s (from DDI table). %s",
				indexDrugName, content.InteractantName, content.SourcePhrase)
		}

		// FIX 3b: Back-fill clinicalEffect/management from sourcePhrase when column-based
		// extraction missed them (e.g., "Clinical Comment: Increased digoxin concentration")
		if content.ClinicalEffect == "" && content.SourcePhrase != "" {
			content.ClinicalEffect = extractFieldFromSourcePhrase(content.SourcePhrase, "clinical")
		}
		if content.Management == "" && content.SourcePhrase != "" {
			content.Management = extractFieldFromSourcePhrase(content.SourcePhrase, "management")
		}

		contents = append(contents, content)
		parsedRows++
	}

	// Adjust confidence based on noise filtering
	confidence := 0.6
	totalProcessed := len(table.Rows) - skippedNoise
	if totalProcessed > 0 {
		confidence = 0.7 + (float64(parsedRows)/float64(totalProcessed))*0.2
	}
	if parsedRows == 0 {
		warnings = append(warnings, "No valid drug interactions extracted after noise filtering")
		confidence = 0.3
	}
	if skippedNoise > 0 {
		warnings = append(warnings, "Filtered "+strconv.Itoa(skippedNoise)+" noise rows (headers, section labels)")
	}

	return &ParseResult{
		Content:    contents,
		Confidence: confidence,
		ParsedRows: parsedRows,
		TotalRows:  len(table.Rows),
		Warnings:   warnings,
	}, nil
}

// detectPKColumns identifies columns that contain pharmacokinetic % change data.
// These are common in DDI tables (e.g., "Increase in digoxin Cmax", "AUC change (%)").
func detectPKColumns(headers []string) []int {
	var pkCols []int
	for i, h := range headers {
		lower := strings.ToLower(h)
		if i == 0 {
			continue // Skip first column (always drug name)
		}
		// Match PK-related header keywords
		if strings.Contains(lower, "cmax") || strings.Contains(lower, "auc") ||
			strings.Contains(lower, "increase") || strings.Contains(lower, "decrease") ||
			strings.Contains(lower, "change") || strings.Contains(lower, "%") ||
			strings.Contains(lower, "ratio") || strings.Contains(lower, "fold") ||
			strings.Contains(lower, "concentration") || strings.Contains(lower, "exposure") {
			pkCols = append(pkCols, i)
		}
	}
	return pkCols
}

// isInteractionSectionHeader detects in-table sub-headers like
// "Digoxin concentrations increased greater than 50%" that span multiple columns.
func isInteractionSectionHeader(row []string, indexDrugName string) bool {
	if len(row) == 0 {
		return false
	}
	firstCell := strings.TrimSpace(row[0])
	if firstCell == "" {
		return false
	}

	// Check if non-first cells are all empty or "NA"
	nonEmptyOthers := 0
	for i := 1; i < len(row); i++ {
		cell := strings.TrimSpace(row[i])
		if cell != "" && !strings.EqualFold(cell, "na") && cell != "-" && cell != "–" {
			nonEmptyOthers++
		}
	}
	if nonEmptyOthers > 0 {
		return false
	}

	// Must reference the index drug or concentration/effect keywords
	lower := strings.ToLower(firstCell)
	drugLower := strings.ToLower(indexDrugName)
	if strings.Contains(lower, drugLower) ||
		strings.Contains(lower, "concentration") ||
		strings.Contains(lower, "increased") ||
		strings.Contains(lower, "decreased") ||
		strings.Contains(lower, "no effect") ||
		strings.Contains(lower, "reduced") {
		return true
	}

	return false
}

// buildPKClinicalEffect constructs a clinical effect description from PK table numeric columns.
// E.g., headers=["Drug","Cmax %","AUC %"], row=["Amiodarone","70%","NA"] → "Cmax increased 70%"
func buildPKClinicalEffect(row, headers []string, pkColIndices []int, indexDrugName, sectionHeader string) string {
	var parts []string
	for _, colIdx := range pkColIndices {
		if colIdx >= len(row) || colIdx >= len(headers) {
			continue
		}
		val := strings.TrimSpace(row[colIdx])
		if val == "" || strings.EqualFold(val, "na") || val == "-" || val == "–" {
			continue
		}
		header := strings.TrimSpace(headers[colIdx])
		parts = append(parts, header+": "+val)
	}

	if len(parts) == 0 {
		// Fall back to section header context
		if sectionHeader != "" {
			return sectionHeader
		}
		return ""
	}

	effect := strings.Join(parts, "; ")
	if sectionHeader != "" {
		effect = sectionHeader + " — " + effect
	}
	return effect
}

// extractFieldFromSourcePhrase extracts clinicalEffect or management from a pipe-delimited
// sourcePhrase when column-based extraction missed them.
// E.g., "Drug: Digoxin | Clinical Comment: Increased digoxin concentration" → "Increased digoxin concentration"
func extractFieldFromSourcePhrase(sourcePhrase, fieldType string) string {
	// Split by pipe delimiter used in buildInteractionSourcePhrase
	parts := strings.Split(sourcePhrase, " | ")

	var clinicalKeywords []string
	var mgmtKeywords []string

	clinicalKeywords = []string{"clinical comment", "clinical effect", "effect", "result", "consequence", "impact"}
	mgmtKeywords = []string{"recommendation", "management", "action", "precaution", "monitoring", "avoid"}

	keywords := clinicalKeywords
	if fieldType == "management" {
		keywords = mgmtKeywords
	}

	for _, part := range parts {
		// Check "Header: Value" format
		colonIdx := strings.Index(part, ": ")
		if colonIdx < 0 {
			continue
		}
		header := strings.ToLower(strings.TrimSpace(part[:colonIdx]))
		value := strings.TrimSpace(part[colonIdx+2:])
		if len(value) < 5 {
			continue
		}
		for _, kw := range keywords {
			if strings.Contains(header, kw) {
				return value
			}
		}
	}
	return ""
}

// buildInteractionSourcePhrase creates a human-readable source phrase for table-parsed interactions.
// Concatenates header + cell values into a readable format for reviewer grounding.
func buildInteractionSourcePhrase(headers, row []string, sectionHeader string) string {
	var parts []string
	if sectionHeader != "" {
		parts = append(parts, "["+sectionHeader+"]")
	}
	for i, cell := range row {
		cell = strings.TrimSpace(cell)
		if cell == "" {
			continue
		}
		if i < len(headers) && headers[i] != "" {
			parts = append(parts, headers[i]+": "+cell)
		} else {
			parts = append(parts, cell)
		}
	}
	phrase := strings.Join(parts, " | ")
	if len(phrase) > 400 {
		phrase = phrase[:397] + "..."
	}
	return phrase
}

// ParseReproductiveSafety extracts pregnancy/lactation safety data
func (cp *ContentParser) ParseReproductiveSafety(table *dailymed.ExtractedTable, loincCode string) (*ParseResult, error) {
	var contents []KBReproductiveSafetyContent
	var warnings []string
	parsedRows := 0

	// Determine category from LOINC code
	category := "PREGNANCY"
	if loincCode == "34080-2" { // Nursing Mothers
		category = "LACTATION"
	}

	for _, row := range table.Rows {
		if len(row) == 0 {
			continue
		}

		rowText := strings.Join(row, " ")
		// KB-specific struct with exact field names for reproductive safety
		content := KBReproductiveSafetyContent{
			Category: category, // Matches content->>'category'
		}

		// Extract FDA pregnancy category if present (legacy)
		if match := regexp.MustCompile(`(?i)category\s*([A-DX])`).FindStringSubmatch(rowText); match != nil {
			content.FDACategory = strings.ToUpper(match[1]) // Matches content->>'fdaCategory'
		}

		// Determine risk level
		content.RiskLevel = cp.extractReproductiveRisk(rowText) // Matches content->>'riskLevel'

		// Check for lactation-specific data
		if category == "LACTATION" {
			if strings.Contains(strings.ToLower(rowText), "excreted") ||
			   strings.Contains(strings.ToLower(rowText), "breast milk") {
				excretion := true
				content.ExcretionInMilk = &excretion // Matches content->>'excretionInMilk'
			}
			// Look for relative infant dose
			if match := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%\s*(of\s+)?maternal`).FindStringSubmatch(rowText); match != nil {
				rid, _ := strconv.ParseFloat(match[1], 64)
				content.RelativeInfantDose = &rid // Matches content->>'relativeInfantDose'
			}
		}

		// Extract summary text
		content.PLLRSummary = cp.extractSummaryText(rowText, 500) // Matches content->>'pllrSummary'

		if content.RiskLevel != "" || content.FDACategory != "" || content.PLLRSummary != "" {
			contents = append(contents, content)
			parsedRows++
		}
	}

	confidence := 0.5
	if parsedRows > 0 {
		confidence = 0.7
	}
	if parsedRows == 0 {
		warnings = append(warnings, "No reproductive safety data extracted")
		confidence = 0.3
	}

	return &ParseResult{
		Content:    contents,
		Confidence: confidence,
		ParsedRows: parsedRows,
		TotalRows:  len(table.Rows),
		Warnings:   warnings,
	}, nil
}

// ParseReproductiveSafetyFromProse extracts pregnancy/lactation facts from narrative text.
// P5.2: Most pregnancy/lactation sections are prose, not tables. This function
// applies the same extraction logic as ParseReproductiveSafety but on PlainText.
func (cp *ContentParser) ParseReproductiveSafetyFromProse(proseText string, loincCode string) *KBReproductiveSafetyContent {
	if len(proseText) < 30 {
		return nil
	}

	category := "PREGNANCY"
	if loincCode == "34080-2" {
		category = "LACTATION"
	}

	content := KBReproductiveSafetyContent{
		Category: category,
	}

	// Extract FDA pregnancy category
	if match := regexp.MustCompile(`(?i)(?:pregnancy\s+)?category\s*([A-DX])\b`).FindStringSubmatch(proseText); match != nil {
		content.FDACategory = strings.ToUpper(match[1])
	}

	// Determine risk level
	content.RiskLevel = cp.extractReproductiveRisk(proseText)

	// Lactation-specific data
	if category == "LACTATION" {
		lower := strings.ToLower(proseText)
		if strings.Contains(lower, "excreted") || strings.Contains(lower, "breast milk") ||
			strings.Contains(lower, "human milk") {
			excretion := true
			content.ExcretionInMilk = &excretion
		}
		// Relative infant dose
		if match := regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%\s*(?:of\s+)?maternal`).FindStringSubmatch(proseText); match != nil {
			rid, _ := strconv.ParseFloat(match[1], 64)
			content.RelativeInfantDose = &rid
		}
	}

	// PLLR summary — first 500 chars of the section
	content.PLLRSummary = cp.extractSummaryText(proseText, 500)

	// Only return if we extracted something meaningful
	if content.RiskLevel == "" && content.FDACategory == "" && content.PLLRSummary == "" {
		return nil
	}

	return &content
}

// ParseFormulary extracts NDC codes and packaging information
func (cp *ContentParser) ParseFormulary(table *dailymed.ExtractedTable) (*ParseResult, error) {
	var contents []KBFormularyContent
	var warnings []string
	parsedRows := 0

	for _, row := range table.Rows {
		if len(row) == 0 {
			continue
		}

		rowText := strings.Join(row, " ")
		// KB-specific struct with exact field names for formulary
		content := KBFormularyContent{}

		// Extract NDC code
		if match := cp.ndcPattern.FindStringSubmatch(rowText); match != nil {
			content.NDCCode = match[1]
		}

		// Extract strength
		if match := cp.dosePattern.FindStringSubmatch(rowText); match != nil {
			content.Strength = match[0]
		}

		// Extract package form
		content.PackageForm = cp.extractPackageForm(rowText)

		// Extract package size
		if match := regexp.MustCompile(`(\d+)\s*(tablets?|capsules?|vials?|bottles?|packets?|count)`).FindStringSubmatch(strings.ToLower(rowText)); match != nil {
			content.PackageSize = match[0]
		}

		// Extract manufacturer if present
		for _, cell := range row {
			if cp.looksLikeManufacturer(cell) {
				content.Manufacturer = strings.TrimSpace(cell)
				break
			}
		}

		if content.NDCCode != "" {
			contents = append(contents, content)
			parsedRows++
		}
	}

	confidence := 0.6
	if len(table.Rows) > 0 && parsedRows > 0 {
		confidence = 0.7 + (float64(parsedRows)/float64(len(table.Rows)))*0.2
	}
	if parsedRows == 0 {
		warnings = append(warnings, "No NDC codes found in table")
		confidence = 0.3
	}

	return &ParseResult{
		Content:    contents,
		Confidence: confidence,
		ParsedRows: parsedRows,
		TotalRows:  len(table.Rows),
		Warnings:   warnings,
	}, nil
}

// ParseLabReference extracts laboratory monitoring requirements
func (cp *ContentParser) ParseLabReference(table *dailymed.ExtractedTable) (*ParseResult, error) {
	var contents []KBLabReferenceContent
	var warnings []string
	parsedRows := 0

	for _, row := range table.Rows {
		if len(row) == 0 {
			continue
		}

		rowText := strings.Join(row, " ")
		// KB-specific struct with exact field names for lab reference
		content := KBLabReferenceContent{}

		// Extract lab test name
		for _, cell := range row {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}
			if content.LabName == "" && cp.looksLikeLabTest(cell) {
				content.LabName = cell
				break
			}
		}

		// Extract reference ranges
		if match := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*[-–to]+\s*(\d+(?:\.\d+)?)\s*(mg/dL|mmol/L|mEq/L|g/dL|%|IU/L|U/L)?`).FindStringSubmatch(rowText); match != nil {
			low, _ := strconv.ParseFloat(match[1], 64)
			high, _ := strconv.ParseFloat(match[2], 64)
			content.ReferenceRangeLow = &low
			content.ReferenceRangeHigh = &high
			if match[3] != "" {
				content.Unit = match[3]
			}
		}

		// Extract monitoring frequency
		content.MonitoringFrequency = cp.extractMonitoringFrequency(rowText)

		// Check for baseline requirement
		content.BaselineRequired = strings.Contains(strings.ToLower(rowText), "baseline") ||
			strings.Contains(strings.ToLower(rowText), "before") ||
			strings.Contains(strings.ToLower(rowText), "prior to")

		// Quality gate: require LOINC code OR numeric reference range
		// Rejects table headers and free-text rows that lack structured lab data
		if content.LabName != "" && (content.LOINCCode != "" || content.ReferenceRangeLow != nil || content.ReferenceRangeHigh != nil) {
			contents = append(contents, content)
			parsedRows++
		}
	}

	confidence := 0.5
	if parsedRows > 0 {
		confidence = 0.7
	}
	if parsedRows == 0 {
		warnings = append(warnings, "No lab references extracted from table")
		confidence = 0.3
	}

	return &ParseResult{
		Content:    contents,
		Confidence: confidence,
		ParsedRows: parsedRows,
		TotalRows:  len(table.Rows),
		Warnings:   warnings,
	}, nil
}

// =============================================================================
// P5.3: LAB_REFERENCE CROSS-SECTION PROSE GRAMMAR
// Extracts lab monitoring instructions from narrative text in Warnings, Dosage,
// and Warnings/Precautions sections. Most monitoring instructions are prose:
//   "Monitor serum potassium periodically during treatment."
//   "Check liver function tests before and during therapy."
// This is ADDITIVE — runs alongside SAFETY_SIGNAL extraction on the same section.
// =============================================================================

// labMonitoringPattern matches monitoring verb + lab test combinations in prose.
// Each pattern captures the lab test name and optionally the monitoring context.
var labMonitoringPatterns = []*regexp.Regexp{
	// "Monitor [lab] [frequency/context]"
	regexp.MustCompile(`(?i)\b(?:monitor|monitoring)\s+([\w\s/]+?)\s*(?:levels?|function|concentrations?)?\s*(?:before|during|periodically|every|at\s+baseline|prior\s+to|weekly|monthly|daily|quarterly|annually|regularly|routinely|as\s+clinically\s+indicated)`),
	// "Check [lab] [levels/function] [frequency]"
	regexp.MustCompile(`(?i)\b(?:check|measure|assess|evaluate|determine|obtain)\s+([\w\s/]+?)\s*(?:levels?|function|concentrations?)?\s*(?:before|during|periodically|every|at\s+baseline|prior\s+to|weekly|monthly|daily|quarterly|annually|regularly|routinely)`),
	// "[Lab] function should/must be monitored"
	regexp.MustCompile(`(?i)\b(renal|hepatic|liver|kidney|thyroid|cardiac)\s+function\s+(?:should|must|needs?\s+to)\s+(?:be\s+)?(?:monitored|assessed|evaluated|checked|measured)`),
	// "[Lab test] should/must be monitored/checked"
	regexp.MustCompile(`(?i)\b(LFTs?|liver\s+enzymes?|serum\s+creatinine|potassium|electrolytes?|blood\s+glucose|hemoglobin|hematocrit|platelet\s+count|INR|PT|aPTT|magnesium|calcium|sodium|chloride|BUN|TSH|blood\s+counts?|CBC)\s+(?:should|must|needs?\s+to)\s+(?:be\s+)?(?:monitored|checked|measured|obtained)`),
	// "Periodic monitoring of [lab]"
	regexp.MustCompile(`(?i)\b(?:periodic|routine|regular|baseline)\s+(?:monitoring|measurement|assessment|evaluation)\s+of\s+([\w\s/,]+?)(?:\s+is\s+|\s+should\s+|\s+during\s+|\s*\.|\s*,)`),
}

// extractLabMonitoringFromProse scans narrative text for lab monitoring instructions
// and returns structured LAB_REFERENCE facts. Cross-validates extracted lab names
// against the known lab test list to prevent false positives.
func (cp *ContentParser) extractLabMonitoringFromProse(proseText string) []KBLabReferenceContent {
	if len(proseText) < 50 {
		return nil
	}

	var results []KBLabReferenceContent
	seen := make(map[string]bool) // Deduplicate by normalized lab name

	// Split into sentences for context-bounded extraction
	sentences := splitIntoSentences(proseText)

	for _, sentence := range sentences {
		for _, pattern := range labMonitoringPatterns {
			matches := pattern.FindAllStringSubmatch(sentence, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}
				rawLabName := strings.TrimSpace(match[1])

				// Normalize common abbreviations to full lab names
				labName := normalizeLabName(rawLabName)
				if labName == "" {
					continue
				}

				// Cross-validate against known lab tests
				if !cp.looksLikeLabTest(labName) && !isKnownLabAbbreviation(rawLabName) {
					continue
				}

				// Deduplicate
				key := strings.ToLower(labName)
				if seen[key] {
					continue
				}
				seen[key] = true

				content := KBLabReferenceContent{
					LabName:             labName,
					MonitoringFrequency: cp.extractMonitoringFrequency(sentence),
					BaselineRequired: strings.Contains(strings.ToLower(sentence), "baseline") ||
						strings.Contains(strings.ToLower(sentence), "before") ||
						strings.Contains(strings.ToLower(sentence), "prior to"),
				}

				results = append(results, content)
			}
		}
	}

	return results
}

// normalizeLabName maps common abbreviations and organ-function terms to
// canonical lab test names suitable for LAB_REFERENCE facts.
func normalizeLabName(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))

	// Direct abbreviation mappings
	abbrevMap := map[string]string{
		"lfts":             "Liver function tests",
		"lft":              "Liver function tests",
		"liver enzymes":    "Liver function tests",
		"serum creatinine": "Serum creatinine",
		"inr":              "INR",
		"pt":               "Prothrombin time",
		"aptt":             "Activated partial thromboplastin time",
		"tsh":              "TSH",
		"cbc":              "Complete blood count",
		"blood counts":     "Complete blood count",
		"blood count":      "Complete blood count",
		"bun":              "BUN",
	}

	if mapped, ok := abbrevMap[lower]; ok {
		return mapped
	}

	// Organ-function terms → expanded names
	organMap := map[string]string{
		"renal":   "Renal function",
		"hepatic": "Hepatic function",
		"liver":   "Liver function tests",
		"kidney":  "Renal function",
		"thyroid": "Thyroid function",
		"cardiac": "Cardiac function",
	}

	if mapped, ok := organMap[lower]; ok {
		return mapped
	}

	// If it's a recognized lab test already, title-case it
	if len(raw) > 2 && len(raw) < 60 {
		return strings.TrimSpace(raw)
	}

	return ""
}

// isKnownLabAbbreviation checks common lab abbreviations that looksLikeLabTest
// might miss because they don't contain the full test name.
func isKnownLabAbbreviation(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	abbreviations := map[string]bool{
		"lfts": true, "lft": true, "liver enzymes": true,
		"inr": true, "pt": true, "aptt": true, "tsh": true,
		"cbc": true, "blood counts": true, "blood count": true,
		"renal": true, "hepatic": true, "liver": true, "kidney": true,
		"thyroid": true, "cardiac": true, "electrolytes": true,
		"serum creatinine": true, "platelet count": true,
		"blood glucose": true,
	}
	return abbreviations[lower]
}

// isLabMonitoringSection returns true if the LOINC code corresponds to a section
// that commonly contains lab monitoring instructions in SPL labels.
func isLabMonitoringSection(loincCode string) bool {
	return loincCode == "34068-7" || // Dosage and Administration
		loincCode == "34067-9" || // Warnings and Precautions
		loincCode == "43685-7" || // Warnings and Precautions (alternate)
		loincCode == "34084-4" || // Adverse Reactions (monitoring instructions sometimes appear here)
		loincCode == "34066-1" // Boxed Warning (critical monitoring requirements)
}

// isBiomarkerSOC returns true if the MedDRA SOC name indicates a lab/biomarker
// measurement rather than a true adverse event. These should be classified as
// LAB_REFERENCE, not SAFETY_SIGNAL.
//
// SAFETY: Only "Investigations" SOC (10022891) is safe to blanket-block.
// DO NOT block "Metabolism and nutrition disorders" — it contains critical
// safety signals: Lactic Acidosis (Black Box), Hypoglycaemia, DKA, Hyperkalaemia.
func isBiomarkerSOC(socName string) bool {
	if socName == "" {
		return false
	}
	return strings.ToLower(socName) == "investigations"
}

// Helper functions

func (cp *ContentParser) detectOrganSystem(table *dailymed.ExtractedTable) string {
	text := strings.ToLower(table.Caption + " " + strings.Join(table.Headers, " "))
	if strings.Contains(text, "renal") || strings.Contains(text, "kidney") ||
	   strings.Contains(text, "gfr") || strings.Contains(text, "crcl") ||
	   strings.Contains(text, "creatinine") {
		return "RENAL"
	}
	if strings.Contains(text, "hepatic") || strings.Contains(text, "liver") ||
	   strings.Contains(text, "child-pugh") || strings.Contains(text, "cirrhosis") {
		return "HEPATIC"
	}
	return "RENAL" // Default for organ impairment tables
}

func (cp *ContentParser) mapColumns(headers []string, keywords []string) map[string]int {
	result := make(map[string]int)
	for i, header := range headers {
		headerLower := strings.ToLower(header)
		for _, kw := range keywords {
			if strings.Contains(headerLower, kw) {
				result[kw] = i
			}
		}
	}
	return result
}

func (cp *ContentParser) normalizeThresholdType(raw string) string {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "egfr") {
		return "eGFR"
	}
	if strings.Contains(lower, "crcl") || strings.Contains(lower, "creatinine clearance") {
		return "CrCl"
	}
	if strings.Contains(lower, "gfr") {
		return "GFR"
	}
	return "GFR"
}

func (cp *ContentParser) normalizeOperator(op string) string {
	switch op {
	case "<", "":
		return "LT"
	case "<=", "≤":
		return "LTE"
	case ">":
		return "GT"
	case ">=", "≥":
		return "GTE"
	default:
		return "EQ"
	}
}

func (cp *ContentParser) classifyImpairmentLevel(low, high float64) string {
	// Standard CKD staging based on eGFR
	midpoint := (low + high) / 2
	switch {
	case midpoint >= 60:
		return "MILD"
	case midpoint >= 30:
		return "MODERATE"
	case midpoint >= 15:
		return "SEVERE"
	default:
		return "ESRD"
	}
}

func (cp *ContentParser) classifyImpairmentBySingle(op string, val float64) string {
	switch op {
	case ">=", "≥", ">":
		if val >= 60 {
			return "NORMAL"
		} else if val >= 30 {
			return "MILD"
		}
		return "MODERATE"
	case "<=", "≤", "<":
		if val <= 15 {
			return "ESRD"
		} else if val <= 30 {
			return "SEVERE"
		} else if val <= 60 {
			return "MODERATE"
		}
		return "MILD"
	default:
		return "UNKNOWN"
	}
}

// ckdStageFromGFR derives CKD stage from GFR range values
func (cp *ContentParser) ckdStageFromGFR(low, high float64) string {
	// Use midpoint for staging when range provided
	midpoint := (low + high) / 2
	if high > 100 {
		midpoint = low // Use lower bound if upper is unbounded
	}

	switch {
	case midpoint >= 90:
		return "1"
	case midpoint >= 60:
		return "2"
	case midpoint >= 45:
		return "3a"
	case midpoint >= 30:
		return "3b"
	case midpoint >= 15:
		return "4"
	default:
		return "5" // ESRD
	}
}

// extractAction derives a standardized action keyword from recommendation text
func (cp *ContentParser) extractAction(text string) string {
	lower := strings.ToLower(text)

	// Check for specific action keywords in priority order
	actions := []struct {
		keywords []string
		action   string
	}{
		{[]string{"contraindicated", "do not use", "should not"}, "CONTRAINDICATED"},
		{[]string{"avoid"}, "AVOID"},
		{[]string{"reduce", "decrease", "lower", "50%", "half"}, "REDUCE_DOSE"},
		{[]string{"discontinue", "stop"}, "DISCONTINUE"},
		{[]string{"monitor", "check", "evaluate"}, "MONITOR"},
		{[]string{"caution", "careful"}, "USE_CAUTION"},
		{[]string{"adjust"}, "ADJUST_DOSE"},
	}

	for _, a := range actions {
		for _, kw := range a.keywords {
			if strings.Contains(lower, kw) {
				return a.action
			}
		}
	}
	return "MONITOR" // Default conservative action
}

// buildSafetyDescription creates a meaningful clinical description from structured safety signal fields.
// Replaces raw table row text (e.g. "Glucose <54 mg/dL [n (%)] 1 (0.4) – 1 (0.4)") with a
// readable clinical summary using MedDRA-normalized condition name and extracted metadata.
func buildSafetyDescription(c KBSafetySignalContent) string {
	name := c.ConditionName
	if c.MedDRAName != "" {
		name = c.MedDRAName
	}
	if name == "" {
		return ""
	}

	var parts []string
	parts = append(parts, name)

	if c.Frequency != "" {
		parts = append(parts, "(frequency: "+c.Frequency+")")
	}
	if c.Severity != "" && c.Severity != "MEDIUM" {
		parts = append(parts, "["+c.Severity+"]")
	}

	return strings.Join(parts, " ")
}

// extractRecommendation generates clinical recommendation based on signal type and text
func (cp *ContentParser) extractRecommendation(text, signalType string) string {
	lower := strings.ToLower(text)

	// Check for explicit recommendations in text
	if strings.Contains(lower, "discontinue") {
		return "Discontinue use immediately"
	}
	if strings.Contains(lower, "contraindicated") {
		return "Use is contraindicated"
	}
	if strings.Contains(lower, "monitor") {
		return "Monitor patient closely"
	}
	if strings.Contains(lower, "avoid") {
		return "Avoid use in this condition"
	}

	// Generate based on signal type
	switch signalType {
	case "BOXED_WARNING":
		return "Review boxed warning before prescribing"
	case "CONTRAINDICATION":
		return "Do not use in patients with this condition"
	case "WARNING":
		return "Use with caution; monitor for adverse effects"
	case "ADVERSE_REACTION":
		return "Monitor for this adverse reaction"
	case "PRECAUTION":
		return "Exercise caution and monitor appropriately"
	default:
		return "Review clinical significance"
	}
}

func (cp *ContentParser) isRecommendationText(text string) bool {
	lower := strings.ToLower(text)
	keywords := []string{"reduce", "avoid", "contraindicated", "monitor", "adjust",
		"discontinue", "do not", "use with caution", "not recommended", "decrease"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func (cp *ContentParser) loincToSignalType(loincCode string) string {
	signalTypes := map[string]string{
		"34066-1": "BOXED_WARNING",
		"34070-3": "CONTRAINDICATION",
		"43685-7": "WARNING",
		"34084-4": "ADVERSE_REACTION",
		"43684-0": "PRECAUTION",
	}
	if st, ok := signalTypes[loincCode]; ok {
		return st
	}
	return "WARNING" // Default
}

func (cp *ContentParser) extractSeverity(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "fatal") || strings.Contains(lower, "death") ||
	   strings.Contains(lower, "life-threatening") || strings.Contains(lower, "life threatening") {
		return "CRITICAL"
	}
	if strings.Contains(lower, "serious") || strings.Contains(lower, "severe") {
		return "HIGH"
	}
	if strings.Contains(lower, "moderate") {
		return "MEDIUM"
	}
	if strings.Contains(lower, "mild") || strings.Contains(lower, "minor") {
		return "LOW"
	}
	return "MEDIUM" // Default
}

func (cp *ContentParser) isNumericOrPercent(text string) bool {
	text = strings.TrimSpace(text)
	if _, err := strconv.ParseFloat(text, 64); err == nil {
		return true
	}
	if strings.HasSuffix(text, "%") {
		return true
	}
	return false
}

func (cp *ContentParser) looksLikeDrugName(text string) bool {
	text = strings.TrimSpace(text)
	if len(text) < 3 || len(text) > 100 {
		return false
	}
	// Drug names typically start with capital letter and don't contain numbers
	if strings.ToUpper(string(text[0])) != string(text[0]) {
		return false
	}
	// Avoid common non-drug words
	skipWords := []string{"table", "figure", "note", "see", "refer", "the", "a", "an"}
	for _, w := range skipWords {
		if strings.EqualFold(text, w) {
			return false
		}
	}
	return true
}

func (cp *ContentParser) extractMechanism(text string) string {
	for _, pattern := range cp.mechanismPatterns {
		if match := pattern.FindStringSubmatch(text); match != nil {
			return strings.TrimSpace(match[0])
		}
	}
	return ""
}

func (cp *ContentParser) extractInteractionSeverity(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "contraindicated") || strings.Contains(lower, "avoid") ||
	   strings.Contains(lower, "do not use") {
		return "SEVERE"
	}
	if strings.Contains(lower, "caution") || strings.Contains(lower, "monitor") {
		return "MODERATE"
	}
	if strings.Contains(lower, "may") || strings.Contains(lower, "possible") {
		return "MILD"
	}
	return "MODERATE" // Default
}

func (cp *ContentParser) extractReproductiveRisk(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "contraindicated") || strings.Contains(lower, "do not use") {
		return "CONTRAINDICATED"
	}
	if strings.Contains(lower, "fetal harm") || strings.Contains(lower, "teratogenic") ||
	   strings.Contains(lower, "embryo") {
		return "HIGH_RISK"
	}
	if strings.Contains(lower, "caution") || strings.Contains(lower, "weigh") ||
	   strings.Contains(lower, "benefit") {
		return "CAUTION"
	}
	if strings.Contains(lower, "compatible") || strings.Contains(lower, "safe") {
		return "COMPATIBLE"
	}
	return "CAUTION" // Default conservative
}

func (cp *ContentParser) extractSummaryText(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) > maxLen {
		// Find last sentence boundary before maxLen
		cutoff := maxLen
		if idx := strings.LastIndex(text[:maxLen], "."); idx > maxLen/2 {
			cutoff = idx + 1
		}
		return strings.TrimSpace(text[:cutoff])
	}
	return text
}

func (cp *ContentParser) extractPackageForm(text string) string {
	lower := strings.ToLower(text)
	forms := []string{"tablet", "capsule", "injection", "solution", "suspension",
		"syrup", "cream", "ointment", "gel", "patch", "inhaler", "spray", "drops"}
	for _, form := range forms {
		if strings.Contains(lower, form) {
			return strings.Title(form)
		}
	}
	return ""
}

func (cp *ContentParser) looksLikeManufacturer(text string) bool {
	lower := strings.ToLower(text)
	// Manufacturer indicators
	indicators := []string{"inc", "llc", "corp", "pharmaceutical", "labs", "laboratories"}
	for _, ind := range indicators {
		if strings.Contains(lower, ind) {
			return true
		}
	}
	return false
}

func (cp *ContentParser) looksLikeLabTest(text string) bool {
	lower := strings.ToLower(text)
	labTests := []string{"creatinine", "bun", "alt", "ast", "hemoglobin", "hematocrit",
		"glucose", "potassium", "sodium", "chloride", "calcium", "magnesium",
		"lactate", "bicarbonate", "albumin", "bilirubin", "platelet", "wbc", "rbc"}
	for _, test := range labTests {
		if strings.Contains(lower, test) {
			return true
		}
	}
	return false
}

func (cp *ContentParser) extractMonitoringFrequency(text string) string {
	lower := strings.ToLower(text)
	patterns := []struct {
		keyword string
		freq    string
	}{
		{"daily", "daily"},
		{"weekly", "weekly"},
		{"monthly", "monthly"},
		{"annually", "annually"},
		{"every 3 months", "quarterly"},
		{"quarterly", "quarterly"},
		{"baseline", "baseline"},
		{"before treatment", "baseline"},
		{"periodically", "periodically"},
	}
	for _, p := range patterns {
		if strings.Contains(lower, p.keyword) {
			return p.freq
		}
	}
	return ""
}

// =============================================================================
// P1.5c: QUALITATIVE FREQUENCY STANDARDIZATION
// Maps qualitative descriptors and percentages to MedDRA-standard frequency bands.
// =============================================================================

// FrequencyBand constants aligned with MedDRA/CIOMS frequency classification
const (
	FrequencyBandVeryCommon = "VERY_COMMON" // ≥10%
	FrequencyBandCommon     = "COMMON"      // ≥1%, <10%
	FrequencyBandUncommon   = "UNCOMMON"    // ≥0.1%, <1%
	FrequencyBandRare       = "RARE"        // ≥0.01%, <0.1%
	FrequencyBandVeryRare   = "VERY_RARE"   // <0.01%
)

// qualitativeFreqRule maps a keyword pattern to a MedDRA frequency band
type qualitativeFreqRule struct {
	keyword string
	band    string
	rate    string // Human-readable rate range
}

// qualitativeFreqRules ordered from most specific to least specific
// to prevent "common" from matching before "very common" or "uncommon"
var qualitativeFreqRules = []qualitativeFreqRule{
	// Very common (≥10%)
	{"very common", FrequencyBandVeryCommon, ">=10%"},
	{"very commonly", FrequencyBandVeryCommon, ">=10%"},

	// Very rare (<0.01%)
	{"very rare", FrequencyBandVeryRare, "<0.01%"},
	{"very rarely", FrequencyBandVeryRare, "<0.01%"},
	{"isolated cases", FrequencyBandVeryRare, "<0.01%"},
	{"isolated reports", FrequencyBandVeryRare, "<0.01%"},

	// Uncommon (≥0.1%, <1%) — must come before "common"
	{"uncommon", FrequencyBandUncommon, "0.1-1%"},
	{"uncommonly", FrequencyBandUncommon, "0.1-1%"},
	{"infrequent", FrequencyBandUncommon, "0.1-1%"},
	{"infrequently", FrequencyBandUncommon, "0.1-1%"},

	// Common (≥1%, <10%)
	{"common", FrequencyBandCommon, "1-10%"},
	{"commonly", FrequencyBandCommon, "1-10%"},
	{"frequent", FrequencyBandCommon, "1-10%"},
	{"frequently", FrequencyBandCommon, "1-10%"},

	// Rare (≥0.01%, <0.1%)
	{"rare", FrequencyBandRare, "0.01-0.1%"},
	{"rarely", FrequencyBandRare, "0.01-0.1%"},
}

// normalizeFrequencyBand converts a raw frequency string into a standardized MedDRA band.
// Handles three formats:
//  1. Percentage values: "12.3%" → VERY_COMMON
//  2. Qualitative keywords: "common", "rare" → mapped band
//  3. Combined: "rare (0.01-0.1%)" → already normalized, extract band
//
// Returns (band, enrichedFrequency) where enrichedFrequency may be augmented
// with rate range for qualitative terms (e.g., "rare" → "rare (0.01-0.1%)").
func normalizeFrequencyBand(frequency string) (string, string) {
	if frequency == "" {
		return "", frequency
	}

	lower := strings.ToLower(strings.TrimSpace(frequency))

	// First try percentage extraction → numeric band classification
	percentRe := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	if matches := percentRe.FindStringSubmatch(lower); len(matches) > 1 {
		val := 0.0
		fmt.Sscanf(matches[1], "%f", &val)
		switch {
		case val >= 10.0:
			return FrequencyBandVeryCommon, frequency
		case val >= 1.0:
			return FrequencyBandCommon, frequency
		case val >= 0.1:
			return FrequencyBandUncommon, frequency
		case val >= 0.01:
			return FrequencyBandRare, frequency
		default:
			return FrequencyBandVeryRare, frequency
		}
	}

	// Then try qualitative keyword matching
	for _, rule := range qualitativeFreqRules {
		if strings.Contains(lower, rule.keyword) {
			// Enrich the frequency string with rate range for downstream consumers
			enriched := strings.TrimSpace(frequency) + " (" + rule.rate + ")"
			return rule.band, enriched
		}
	}

	return "", frequency
}

// =============================================================================
// P1.5b: PROSE FREQUENCY ANNOTATION
// Scans section PlainText for percentage patterns near identified AE terms.
// Two-pass design: MedDRA channel finds terms → this annotator finds nearby frequencies.
// =============================================================================

// annotateProseFrequency enriches safety signal facts with frequency data
// extracted from the section's narrative text. For each fact that lacks frequency,
// it searches a ±100 character window around the condition name in the prose
// for percentage patterns like "12.3%" or qualitative descriptors.
//
// This captures frequency data that lives in prose rather than tables, e.g.:
//
//	"Headache was reported in approximately 15% of patients receiving the drug"
//	"Nausea occurred commonly (≥1/100) in clinical trials"
func (cp *ContentParser) annotateProseFrequency(sectionText string, contents []KBSafetySignalContent) []KBSafetySignalContent {
	if sectionText == "" || len(contents) == 0 {
		return contents
	}

	lowerText := strings.ToLower(sectionText)
	percentRe := cp.percentPattern // Reuse the precompiled (\d+(?:\.\d+)?)\s*% pattern

	for i := range contents {
		// Skip facts that already have frequency from table extraction
		if contents[i].Frequency != "" {
			// Even if we have frequency, ensure FrequencyBand is populated
			if contents[i].FrequencyBand == "" {
				contents[i].FrequencyBand, contents[i].Frequency = normalizeFrequencyBand(contents[i].Frequency)
			}
			continue
		}

		condName := strings.ToLower(contents[i].ConditionName)
		if condName == "" {
			continue
		}

		// Find all occurrences of the condition name in the prose
		searchStart := 0
		for {
			idx := strings.Index(lowerText[searchStart:], condName)
			if idx < 0 {
				break
			}
			absIdx := searchStart + idx

			// Define ±100 character window around the term
			windowStart := absIdx - 100
			if windowStart < 0 {
				windowStart = 0
			}
			windowEnd := absIdx + len(condName) + 100
			if windowEnd > len(sectionText) {
				windowEnd = len(sectionText)
			}
			window := sectionText[windowStart:windowEnd]

			// Look for percentage in the window
			if match := percentRe.FindStringSubmatch(window); len(match) > 0 {
				contents[i].Frequency = match[0]
				contents[i].FrequencyBand, contents[i].Frequency = normalizeFrequencyBand(contents[i].Frequency)
				break
			}

			// Look for qualitative frequency keywords in the window
			lowerWindow := strings.ToLower(window)
			for _, rule := range qualitativeFreqRules {
				if strings.Contains(lowerWindow, rule.keyword) {
					contents[i].Frequency = rule.keyword + " (" + rule.rate + ")"
					contents[i].FrequencyBand = rule.band
					break
				}
			}

			// Move past this occurrence
			if contents[i].Frequency != "" {
				break
			}
			searchStart = absIdx + len(condName)
		}
	}

	return contents
}
