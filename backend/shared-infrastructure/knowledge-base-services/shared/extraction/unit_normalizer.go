// Package extraction provides clinical data normalization utilities.
//
// Phase 3b.5.2: Unit Normalizer
// Key Principle: Clinical values must be standardized before comparison
// - CrCl → mL/min
// - eGFR → mL/min/1.73m²
// - Child-Pugh → A/B/C
// - Age categories → normalized ranges
package extraction

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// =============================================================================
// UNIT NORMALIZER
// =============================================================================

// UnitNormalizer standardizes clinical units to canonical forms
type UnitNormalizer struct {
	unitMappings       map[string]string
	variableMappings   map[string]string
	gfrPattern         *regexp.Regexp
	gfrRangePattern    *regexp.Regexp
	childPughPattern   *regexp.Regexp
	percentagePattern  *regexp.Regexp
	dosePattern        *regexp.Regexp
	frequencyPattern   *regexp.Regexp
}

// NewUnitNormalizer creates a normalizer with standard clinical mappings
func NewUnitNormalizer() *UnitNormalizer {
	return &UnitNormalizer{
		unitMappings: map[string]string{
			// Renal function units → canonical mL/min
			"ml/min":            "mL/min",
			"ml / min":          "mL/min",
			"ml per min":        "mL/min",
			"ml per minute":     "mL/min",
			"milliliters/min":   "mL/min",
			"ml/min/1.73m2":     "mL/min/1.73m²",
			"ml/min/1.73m²":     "mL/min/1.73m²",
			"ml/min/1.73 m2":    "mL/min/1.73m²",
			"ml/min/1.73 m²":    "mL/min/1.73m²",
			"ml/min per 1.73m2": "mL/min/1.73m²",

			// Dose units → canonical
			"mg":          "mg",
			"milligrams":  "mg",
			"milligram":   "mg",
			"mcg":         "mcg",
			"micrograms":  "mcg",
			"microgram":   "mcg",
			"μg":          "mcg",
			"µg":          "mcg",
			"ug":          "mcg",
			"g":           "g",
			"grams":       "g",
			"gram":        "g",
			"mg/kg":       "mg/kg",
			"mg per kg":   "mg/kg",
			"mcg/kg":      "mcg/kg",
			"mg/m2":       "mg/m²",
			"mg/m²":       "mg/m²",
			"mg per m2":   "mg/m²",

			// Volume units
			"ml":          "mL",
			"milliliters": "mL",
			"milliliter":  "mL",
			"l":           "L",
			"liters":      "L",
			"liter":       "L",

			// Time units → canonical
			"hr":      "hour",
			"hrs":     "hour",
			"hour":    "hour",
			"hours":   "hour",
			"h":       "hour",
			"day":     "day",
			"days":    "day",
			"d":       "day",
			"week":    "week",
			"weeks":   "week",
			"wk":      "week",
			"wks":     "week",
			"month":   "month",
			"months":  "month",
			"mo":      "month",
			"year":    "year",
			"years":   "year",
			"yr":      "year",
			"yrs":     "year",

			// Percentage
			"%":       "percent",
			"percent": "percent",
			"pct":     "percent",

			// Other clinical units
			"units":     "U",
			"unit":      "U",
			"u":         "U",
			"iu":        "IU",
			"mmol/l":    "mmol/L",
			"mmol/L":    "mmol/L",
			"mg/dl":     "mg/dL",
			"mg/dL":     "mg/dL",
			"ng/ml":     "ng/mL",
			"ng/mL":     "ng/mL",
		},
		variableMappings: map[string]string{
			// Renal function variables
			"crcl":                    "renal_function.crcl",
			"creatinine clearance":    "renal_function.crcl",
			"clcr":                    "renal_function.crcl",
			"creatinine-clearance":    "renal_function.crcl",
			"egfr":                    "renal_function.egfr",
			"estimated gfr":           "renal_function.egfr",
			"gfr":                     "renal_function.gfr",
			"glomerular filtration":   "renal_function.gfr",
			"creatinine":              "renal_function.creatinine",
			"serum creatinine":        "renal_function.creatinine",
			"renal function":          "renal_function.category",
			"renal impairment":        "renal_function.impairment",
			"kidney function":         "renal_function.category",

			// Hepatic function variables
			"child-pugh":              "hepatic.child_pugh",
			"child pugh":              "hepatic.child_pugh",
			"childpugh":               "hepatic.child_pugh",
			"child-pugh class":        "hepatic.child_pugh",
			"child-pugh score":        "hepatic.child_pugh",
			"hepatic impairment":      "hepatic.impairment_level",
			"hepatic function":        "hepatic.function",
			"liver function":          "hepatic.function",
			"liver impairment":        "hepatic.impairment_level",
			"cirrhosis":               "hepatic.cirrhosis",

			// Patient demographics
			"age":                     "patient.age",
			"patient age":             "patient.age",
			"pediatric":               "patient.age_category",
			"geriatric":               "patient.age_category",
			"elderly":                 "patient.age_category",
			"weight":                  "patient.weight",
			"body weight":             "patient.weight",
			"bw":                      "patient.weight",
			"bsa":                     "patient.bsa",
			"body surface area":       "patient.bsa",

			// Lab values
			"potassium":               "lab.potassium",
			"sodium":                  "lab.sodium",
			"glucose":                 "lab.glucose",
			"hemoglobin":              "lab.hemoglobin",
			"hgb":                     "lab.hemoglobin",
			"platelet":                "lab.platelet",
			"plt":                     "lab.platelet",
			"wbc":                     "lab.wbc",
			"alt":                     "lab.alt",
			"ast":                     "lab.ast",
			"bilirubin":               "lab.bilirubin",
			"albumin":                 "lab.albumin",
			"inr":                     "lab.inr",
		},
		// Patterns for extracting values
		gfrPattern:        regexp.MustCompile(`(?i)(CrCl|eGFR|GFR|creatinine\s+clearance|ClCr)\s*[:<]?\s*([<>≤≥]=?)\s*(\d+(?:\.\d+)?)`),
		gfrRangePattern:   regexp.MustCompile(`(?i)(CrCl|eGFR|GFR|creatinine\s+clearance|ClCr)\s*[:<]?\s*(\d+(?:\.\d+)?)\s*(?:-|to|–)\s*(\d+(?:\.\d+)?)`),
		childPughPattern:  regexp.MustCompile(`(?i)(Child[-\s]?Pugh\s*(?:class\s*)?([ABC]))|((mild|moderate|severe)\s+hepatic\s+impairment)`),
		percentagePattern: regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`),
		dosePattern:       regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(mg|mcg|g|μg|µg|ug|ml|mL)`),
		frequencyPattern:  regexp.MustCompile(`(?i)(once|twice|three times|four times|every\s+\d+\s+hours?|daily|bid|tid|qid|qd|q\d+h)`),
	}
}

// =============================================================================
// NORMALIZED VALUE TYPES
// =============================================================================

// NormalizedValue represents a standardized clinical value
type NormalizedValue struct {
	Variable     string   `json:"variable"`               // Canonical variable name
	NumericValue *float64 `json:"numeric_value,omitempty"`
	MinValue     *float64 `json:"min_value,omitempty"`    // For ranges
	MaxValue     *float64 `json:"max_value,omitempty"`    // For ranges
	StringValue  *string  `json:"string_value,omitempty"`
	Unit         string   `json:"unit"`                   // Canonical unit
	OriginalText string   `json:"original_text"`          // Source text for audit
	Confidence   float64  `json:"confidence"`             // How confident in the normalization
}

// IsRange returns true if this is a range value
func (nv *NormalizedValue) IsRange() bool {
	return nv.MinValue != nil && nv.MaxValue != nil
}

// =============================================================================
// UNIT NORMALIZATION METHODS
// =============================================================================

// NormalizeUnit converts a unit string to canonical form
func (n *UnitNormalizer) NormalizeUnit(unit string) string {
	normalized := strings.TrimSpace(strings.ToLower(unit))

	// Direct lookup
	if canonical, ok := n.unitMappings[normalized]; ok {
		return canonical
	}

	// Case-insensitive lookup
	for k, v := range n.unitMappings {
		if strings.EqualFold(k, normalized) {
			return v
		}
	}

	return unit // Return original if no mapping found
}

// NormalizeVariable converts a variable name to canonical form
func (n *UnitNormalizer) NormalizeVariable(variable string) string {
	normalized := strings.TrimSpace(strings.ToLower(variable))

	// Direct lookup
	if canonical, ok := n.variableMappings[normalized]; ok {
		return canonical
	}

	// Partial match lookup
	for k, v := range n.variableMappings {
		if strings.Contains(normalized, k) {
			return v
		}
	}

	return variable
}

// =============================================================================
// CHILD-PUGH NORMALIZATION
// =============================================================================

// NormalizeChildPugh converts hepatic impairment descriptions to A/B/C
func (n *UnitNormalizer) NormalizeChildPugh(text string) (string, float64) {
	lower := strings.ToLower(text)

	// Direct Child-Pugh class mentions
	if strings.Contains(lower, "child-pugh a") || strings.Contains(lower, "child pugh a") ||
		strings.Contains(lower, "class a") || strings.Contains(lower, "score 5-6") {
		return "A", 0.95
	}
	if strings.Contains(lower, "child-pugh b") || strings.Contains(lower, "child pugh b") ||
		strings.Contains(lower, "class b") || strings.Contains(lower, "score 7-9") {
		return "B", 0.95
	}
	if strings.Contains(lower, "child-pugh c") || strings.Contains(lower, "child pugh c") ||
		strings.Contains(lower, "class c") || strings.Contains(lower, "score 10-15") {
		return "C", 0.95
	}

	// Severity-based mapping
	if strings.Contains(lower, "mild hepatic") || strings.Contains(lower, "mild liver") {
		return "A", 0.85
	}
	if strings.Contains(lower, "moderate hepatic") || strings.Contains(lower, "moderate liver") {
		return "B", 0.85
	}
	if strings.Contains(lower, "severe hepatic") || strings.Contains(lower, "severe liver") ||
		strings.Contains(lower, "decompensated") {
		return "C", 0.85
	}

	return "", 0.0 // No match
}

// =============================================================================
// GFR/RENAL FUNCTION PARSING
// =============================================================================

// ParseGFRThreshold extracts GFR threshold from text
func (n *UnitNormalizer) ParseGFRThreshold(text string) (*NormalizedValue, error) {
	// Try range pattern first (e.g., "CrCl 30-60 mL/min")
	if matches := n.gfrRangePattern.FindStringSubmatch(text); len(matches) >= 4 {
		min, _ := strconv.ParseFloat(matches[2], 64)
		max, _ := strconv.ParseFloat(matches[3], 64)

		return &NormalizedValue{
			Variable:     n.NormalizeVariable(matches[1]),
			MinValue:     &min,
			MaxValue:     &max,
			Unit:         "mL/min",
			OriginalText: matches[0],
			Confidence:   0.9,
		}, nil
	}

	// Try single value with operator pattern (e.g., "CrCl < 30")
	if matches := n.gfrPattern.FindStringSubmatch(text); len(matches) >= 4 {
		value, _ := strconv.ParseFloat(matches[3], 64)

		return &NormalizedValue{
			Variable:     n.NormalizeVariable(matches[1]),
			NumericValue: &value,
			Unit:         "mL/min",
			OriginalText: matches[0],
			Confidence:   0.9,
		}, nil
	}

	return nil, fmt.Errorf("no GFR threshold found in: %s", text)
}

// ParseGFRRange extracts a GFR range from text like "30-60" or "30 to 60"
func (n *UnitNormalizer) ParseGFRRange(text string) (min, max float64, err error) {
	rangePattern := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:-|to|–)\s*(\d+(?:\.\d+)?)`)

	matches := rangePattern.FindStringSubmatch(text)
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("no range found in: %s", text)
	}

	min, _ = strconv.ParseFloat(matches[1], 64)
	max, _ = strconv.ParseFloat(matches[2], 64)

	return min, max, nil
}

// =============================================================================
// DOSE PARSING
// =============================================================================

// ParseDose extracts dose value and unit from text
func (n *UnitNormalizer) ParseDose(text string) (*NormalizedValue, error) {
	matches := n.dosePattern.FindStringSubmatch(text)
	if len(matches) < 3 {
		return nil, fmt.Errorf("no dose found in: %s", text)
	}

	value, _ := strconv.ParseFloat(matches[1], 64)
	unit := n.NormalizeUnit(matches[2])

	return &NormalizedValue{
		Variable:     "dose",
		NumericValue: &value,
		Unit:         unit,
		OriginalText: matches[0],
		Confidence:   0.9,
	}, nil
}

// ParsePercentage extracts percentage value from text
func (n *UnitNormalizer) ParsePercentage(text string) (*float64, error) {
	matches := n.percentagePattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no percentage found in: %s", text)
	}

	value, _ := strconv.ParseFloat(matches[1], 64)
	return &value, nil
}

// ParseFrequency extracts dosing frequency from text
func (n *UnitNormalizer) ParseFrequency(text string) string {
	matches := n.frequencyPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}

	freq := strings.ToLower(matches[1])

	// Normalize common frequencies
	switch {
	case freq == "once" || freq == "qd" || freq == "daily":
		return "daily"
	case freq == "twice" || freq == "bid":
		return "BID"
	case strings.Contains(freq, "three") || freq == "tid":
		return "TID"
	case strings.Contains(freq, "four") || freq == "qid":
		return "QID"
	case strings.HasPrefix(freq, "q"):
		return strings.ToUpper(freq)
	case strings.HasPrefix(freq, "every"):
		return freq
	}

	return freq
}

// =============================================================================
// RENAL IMPAIRMENT CATEGORY MAPPING
// =============================================================================

// RenalCategory represents standardized renal impairment categories
type RenalCategory string

const (
	RenalNormal       RenalCategory = "NORMAL"          // GFR >= 90
	RenalMild         RenalCategory = "MILD"            // GFR 60-89
	RenalModerate     RenalCategory = "MODERATE"        // GFR 30-59
	RenalSevere       RenalCategory = "SEVERE"          // GFR 15-29
	RenalKidneyFailure RenalCategory = "KIDNEY_FAILURE" // GFR < 15
	RenalDialysis     RenalCategory = "DIALYSIS"        // On dialysis
	RenalESRD         RenalCategory = "ESRD"            // End-stage renal disease
)

// GFRToCategory converts a GFR value to a renal impairment category
func (n *UnitNormalizer) GFRToCategory(gfr float64) RenalCategory {
	switch {
	case gfr >= 90:
		return RenalNormal
	case gfr >= 60:
		return RenalMild
	case gfr >= 30:
		return RenalModerate
	case gfr >= 15:
		return RenalSevere
	default:
		return RenalKidneyFailure
	}
}

// CategoryToGFRRange returns the GFR range for a category
func (n *UnitNormalizer) CategoryToGFRRange(category RenalCategory) (min, max float64) {
	switch category {
	case RenalNormal:
		return 90, 999 // No upper limit
	case RenalMild:
		return 60, 89
	case RenalModerate:
		return 30, 59
	case RenalSevere:
		return 15, 29
	case RenalKidneyFailure, RenalESRD:
		return 0, 14
	default:
		return 0, 999
	}
}

// ParseRenalCategory extracts renal category from descriptive text
func (n *UnitNormalizer) ParseRenalCategory(text string) (RenalCategory, float64) {
	lower := strings.ToLower(text)

	// Check for dialysis/ESRD first
	if strings.Contains(lower, "dialysis") || strings.Contains(lower, "hemodialysis") ||
		strings.Contains(lower, "peritoneal dialysis") {
		return RenalDialysis, 0.95
	}
	if strings.Contains(lower, "esrd") || strings.Contains(lower, "end-stage") ||
		strings.Contains(lower, "end stage") {
		return RenalESRD, 0.95
	}

	// Category-based
	if strings.Contains(lower, "normal renal") || strings.Contains(lower, "normal kidney") {
		return RenalNormal, 0.85
	}
	if strings.Contains(lower, "mild renal") || strings.Contains(lower, "mild kidney") ||
		strings.Contains(lower, "mildly impaired") {
		return RenalMild, 0.85
	}
	if strings.Contains(lower, "moderate renal") || strings.Contains(lower, "moderate kidney") ||
		strings.Contains(lower, "moderately impaired") {
		return RenalModerate, 0.85
	}
	if strings.Contains(lower, "severe renal") || strings.Contains(lower, "severe kidney") ||
		strings.Contains(lower, "severely impaired") {
		return RenalSevere, 0.85
	}

	return "", 0.0
}

// =============================================================================
// COMPREHENSIVE TEXT NORMALIZATION
// =============================================================================

// NormalizeConditionText attempts to extract a normalized value from any condition text
func (n *UnitNormalizer) NormalizeConditionText(text string) (*NormalizedValue, error) {
	// Try GFR/renal first
	if nv, err := n.ParseGFRThreshold(text); err == nil {
		return nv, nil
	}

	// Try Child-Pugh
	if childPugh, conf := n.NormalizeChildPugh(text); childPugh != "" {
		return &NormalizedValue{
			Variable:     "hepatic.child_pugh",
			StringValue:  &childPugh,
			OriginalText: text,
			Confidence:   conf,
		}, nil
	}

	// Try renal category
	if category, conf := n.ParseRenalCategory(text); category != "" {
		min, max := n.CategoryToGFRRange(category)
		return &NormalizedValue{
			Variable:     "renal_function.category",
			StringValue:  (*string)(&category),
			MinValue:     &min,
			MaxValue:     &max,
			Unit:         "mL/min",
			OriginalText: text,
			Confidence:   conf,
		}, nil
	}

	// Try generic dose
	if nv, err := n.ParseDose(text); err == nil {
		return nv, nil
	}

	return nil, fmt.Errorf("could not normalize: %s", text)
}
