// Package interpretation provides clinical interpretation algorithms for lab results
// troponin_delta.go implements ESC 2023 NSTE-ACS Guidelines for hs-Troponin rapid protocols
package interpretation

import (
	"math"
	"time"
)

// =============================================================================
// hs-TROPONIN DELTA ALGORITHM (ESC 2023 NSTE-ACS Guidelines)
// =============================================================================

// HsTroponinDeltaResult contains the result of hs-Troponin rapid rule-out/rule-in evaluation
type HsTroponinDeltaResult struct {
	// Algorithm Used
	Algorithm string `json:"algorithm"` // "0/1h", "0/2h", "0/3h"

	// Input Values
	InitialValue   float64   `json:"initialValue"`   // ng/L
	DeltaValue     float64   `json:"deltaValue"`     // ng/L (second measurement)
	InitialTime    time.Time `json:"initialTime"`    // Time of first sample
	DeltaTime      time.Time `json:"deltaTime"`      // Time of second sample
	ActualInterval float64   `json:"actualInterval"` // Actual minutes between samples

	// Calculated Values
	AbsoluteChange float64 `json:"absoluteChange"` // Absolute change in ng/L
	PercentChange  float64 `json:"percentChange"`  // Percent change

	// Classification
	Classification string `json:"classification"` // RULE_OUT, RULE_IN, OBSERVE
	Confidence     string `json:"confidence"`     // HIGH, MODERATE, LOW

	// Assay Information
	AssayManufacturer string `json:"assayManufacturer"`
	AssayPlatform     string `json:"assayPlatform"`
	AssayName         string `json:"assayName"`

	// Clinical Interpretation
	Interpretation     string   `json:"interpretation"`     // Human-readable interpretation
	Recommendations    []string `json:"recommendations"`    // Clinical recommendations
	RequiresObservation bool     `json:"requiresObservation"` // Whether serial testing needed

	// Governance & Provenance
	Governance TroponinGovernance `json:"governance"`
}

// TroponinGovernance tracks clinical authority for troponin interpretation
type TroponinGovernance struct {
	Algorithm       string `json:"algorithm"`       // "ESC 0/1h Protocol", "ESC 0/2h Protocol"
	CutoffSource    string `json:"cutoffSource"`    // "ESC 2023 NSTE-ACS Guidelines Table 3"
	AssaySpecific   bool   `json:"assaySpecific"`   // Always true for hs-Troponin
	ManufacturerRef string `json:"manufacturerRef"` // Package insert reference
	FDAClearanceNum string `json:"fdaClearanceNum,omitempty"`
	EvidenceLevel   string `json:"evidenceLevel"`   // "HIGH" - Class I, Level A
}

// =============================================================================
// ASSAY-SPECIFIC CUTOFFS (ESC 2023 Table 3)
// =============================================================================

// AssayCutoffs contains the rule-out and rule-in thresholds for each assay
type AssayCutoffs struct {
	Manufacturer string
	Platform     string
	AssayName    string

	// 0/1h Protocol Cutoffs (ng/L)
	RuleOutBaseline_01h float64 // Baseline < this = Rule Out
	RuleInBaseline_01h  float64 // Baseline >= this = Rule In
	RuleInDelta_01h     float64 // Delta >= this = Rule In (if baseline doesn't meet rule-out)

	// 0/2h Protocol Cutoffs (ng/L) - for assays without validated 0/1h
	RuleOutBaseline_02h *float64
	RuleInBaseline_02h  *float64
	RuleInDelta_02h     *float64

	// Sex-specific 99th percentile URL (optional)
	URL99thPercentile       float64
	MaleURL99thPercentile   *float64
	FemaleURL99thPercentile *float64

	// Regulatory
	FDAClearanceNum  string
	PackageInsertRef string
}

// TroponinAssayCutoffs contains validated cutoffs from ESC 2023 Table 3
// CRITICAL: These values are assay-specific and cannot be generalized
var TroponinAssayCutoffs = map[string]AssayCutoffs{
	// Roche Elecsys hs-TnT (5th generation)
	"ROCHE_ELECSYS_HSTNT": {
		Manufacturer: "Roche",
		Platform:     "Cobas e801/e601/e411",
		AssayName:    "Elecsys Troponin T hs",

		// 0/1h Protocol (ESC 2023)
		RuleOutBaseline_01h: 5.0,  // <5 ng/L = Rule Out
		RuleInBaseline_01h:  52.0, // ≥52 ng/L = Rule In
		RuleInDelta_01h:     5.0,  // Delta ≥5 ng/L = Rule In

		URL99thPercentile: 14.0, // Combined sex 99th %ile

		FDAClearanceNum:  "K173327",
		PackageInsertRef: "Roche Elecsys Troponin T hs STAT Package Insert v9.0",
	},

	// Abbott Architect hs-TnI
	"ABBOTT_ARCHITECT_HSTNI": {
		Manufacturer: "Abbott",
		Platform:     "Architect i1000/i2000",
		AssayName:    "ARCHITECT STAT High Sensitive Troponin-I",

		// 0/1h Protocol (ESC 2023)
		RuleOutBaseline_01h: 4.0,  // <4 ng/L = Rule Out
		RuleInBaseline_01h:  64.0, // ≥64 ng/L = Rule In
		RuleInDelta_01h:     6.0,  // Delta ≥6 ng/L = Rule In

		URL99thPercentile:       26.0, // Combined sex 99th %ile
		MaleURL99thPercentile:   ptr(34.2),
		FemaleURL99thPercentile: ptr(15.6),

		FDAClearanceNum:  "K173384",
		PackageInsertRef: "Abbott ARCHITECT STAT High Sensitive Troponin-I Package Insert",
	},

	// Siemens Atellica hs-TnI
	"SIEMENS_ATELLICA_HSTNI": {
		Manufacturer: "Siemens",
		Platform:     "Atellica IM/ADVIA Centaur",
		AssayName:    "Atellica IM High-Sensitivity Troponin I",

		// 0/1h Protocol (ESC 2023)
		RuleOutBaseline_01h: 4.0,   // <4 ng/L = Rule Out
		RuleInBaseline_01h:  120.0, // ≥120 ng/L = Rule In
		RuleInDelta_01h:     6.0,   // Delta ≥6 ng/L = Rule In

		URL99thPercentile:       45.0, // Combined sex 99th %ile
		MaleURL99thPercentile:   ptr(53.0),
		FemaleURL99thPercentile: ptr(34.0),

		FDAClearanceNum:  "K181978",
		PackageInsertRef: "Siemens Atellica IM hs-TnI Package Insert",
	},

	// Beckman Coulter Access hs-TnI
	"BECKMAN_ACCESS_HSTNI": {
		Manufacturer: "Beckman Coulter",
		Platform:     "Access 2/DxI",
		AssayName:    "Access hsTnI",

		// 0/1h Protocol
		RuleOutBaseline_01h: 4.0,  // <4 ng/L = Rule Out
		RuleInBaseline_01h:  50.0, // ≥50 ng/L = Rule In
		RuleInDelta_01h:     5.0,  // Delta ≥5 ng/L = Rule In

		URL99thPercentile: 17.5, // Combined sex 99th %ile

		FDAClearanceNum:  "K180494",
		PackageInsertRef: "Beckman Coulter Access hsTnI Package Insert",
	},

	// Ortho VITROS hs-TnI
	"ORTHO_VITROS_HSTNI": {
		Manufacturer: "Ortho Clinical Diagnostics",
		Platform:     "VITROS 5600/XT 7600",
		AssayName:    "VITROS High Sensitivity Troponin I",

		// 0/2h Protocol (0/1h not validated)
		RuleOutBaseline_02h: ptr(4.0),
		RuleInBaseline_02h:  ptr(40.0),
		RuleInDelta_02h:     ptr(4.0),

		URL99thPercentile: 11.6,

		FDAClearanceNum:  "K190659",
		PackageInsertRef: "Ortho VITROS hs-TnI Package Insert",
	},
}

// =============================================================================
// EVALUATION FUNCTIONS
// =============================================================================

// EvaluateHsTroponin01h evaluates hs-Troponin using the ESC 0/1h rapid protocol
func EvaluateHsTroponin01h(baseline, hour1 float64, assay string) *HsTroponinDeltaResult {
	cutoffs, ok := TroponinAssayCutoffs[assay]
	if !ok {
		// Unknown assay - return observe with low confidence
		return &HsTroponinDeltaResult{
			Algorithm:      "0/1h",
			InitialValue:   baseline,
			DeltaValue:     hour1,
			AbsoluteChange: math.Abs(hour1 - baseline),
			Classification: "OBSERVE",
			Confidence:     "LOW",
			Interpretation: "Unknown assay - cannot apply validated cutoffs. Manual interpretation required.",
			Recommendations: []string{
				"Identify assay manufacturer and platform",
				"Use standard 3h protocol with 99th percentile URL",
				"Consider clinical context and ECG findings",
			},
			RequiresObservation: true,
			Governance: TroponinGovernance{
				Algorithm:     "ESC 0/1h Protocol",
				CutoffSource:  "Unknown - assay not validated",
				AssaySpecific: true,
				EvidenceLevel: "LOW",
			},
		}
	}

	result := &HsTroponinDeltaResult{
		Algorithm:         "0/1h",
		InitialValue:      baseline,
		DeltaValue:        hour1,
		AbsoluteChange:    math.Abs(hour1 - baseline),
		AssayManufacturer: cutoffs.Manufacturer,
		AssayPlatform:     cutoffs.Platform,
		AssayName:         cutoffs.AssayName,
	}

	if baseline > 0 {
		result.PercentChange = ((hour1 - baseline) / baseline) * 100
	}

	// Apply ESC 0/1h Algorithm (Figure 6, ESC 2023)
	// Step 1: Check baseline for immediate rule-out
	if baseline < cutoffs.RuleOutBaseline_01h {
		result.Classification = "RULE_OUT"
		result.Confidence = "HIGH"
		result.Interpretation = "Very low baseline troponin - AMI ruled out with high confidence."
		result.Recommendations = []string{
			"Consider alternative diagnoses for chest pain",
			"Assess for other causes of symptoms",
			"Safe for discharge if clinically stable",
		}
		result.RequiresObservation = false
	} else if baseline >= cutoffs.RuleInBaseline_01h {
		// Step 2: Check baseline for immediate rule-in
		result.Classification = "RULE_IN"
		result.Confidence = "HIGH"
		result.Interpretation = "Markedly elevated baseline troponin - NSTEMI likely."
		result.Recommendations = []string{
			"Initiate ACS protocol",
			"Cardiology consultation",
			"Consider early invasive strategy",
			"Start anticoagulation per guidelines",
		}
		result.RequiresObservation = false
	} else if result.AbsoluteChange >= cutoffs.RuleInDelta_01h {
		// Step 3: Check delta for rule-in
		result.Classification = "RULE_IN"
		result.Confidence = "HIGH"
		result.Interpretation = "Significant troponin rise/fall - NSTEMI likely."
		result.Recommendations = []string{
			"Initiate ACS protocol",
			"Cardiology consultation",
			"Serial ECGs",
			"Risk stratification (GRACE score)",
		}
		result.RequiresObservation = false
	} else {
		// Observation zone
		result.Classification = "OBSERVE"
		result.Confidence = "MODERATE"
		result.Interpretation = "Troponin in observation zone - serial testing required."
		result.Recommendations = []string{
			"Repeat troponin at 3 hours from initial presentation",
			"Serial ECG monitoring",
			"Clinical reassessment",
			"Consider admission for observation",
		}
		result.RequiresObservation = true
	}

	// Set governance
	result.Governance = TroponinGovernance{
		Algorithm:       "ESC 0/1h Protocol",
		CutoffSource:    "ESC 2023 NSTE-ACS Guidelines Table 3",
		AssaySpecific:   true,
		ManufacturerRef: cutoffs.PackageInsertRef,
		FDAClearanceNum: cutoffs.FDAClearanceNum,
		EvidenceLevel:   "HIGH", // Class I, Level A
	}

	return result
}

// EvaluateHsTroponin02h evaluates hs-Troponin using the ESC 0/2h rapid protocol
// Used when 0/1h protocol not validated for specific assay or 1h sample not available
func EvaluateHsTroponin02h(baseline, hour2 float64, assay string) *HsTroponinDeltaResult {
	cutoffs, ok := TroponinAssayCutoffs[assay]
	if !ok {
		return &HsTroponinDeltaResult{
			Algorithm:           "0/2h",
			InitialValue:        baseline,
			DeltaValue:          hour2,
			AbsoluteChange:      math.Abs(hour2 - baseline),
			Classification:      "OBSERVE",
			Confidence:          "LOW",
			Interpretation:      "Unknown assay - cannot apply validated cutoffs.",
			RequiresObservation: true,
			Governance: TroponinGovernance{
				Algorithm:     "ESC 0/2h Protocol",
				CutoffSource:  "Unknown - assay not validated",
				AssaySpecific: true,
				EvidenceLevel: "LOW",
			},
		}
	}

	result := &HsTroponinDeltaResult{
		Algorithm:         "0/2h",
		InitialValue:      baseline,
		DeltaValue:        hour2,
		AbsoluteChange:    math.Abs(hour2 - baseline),
		AssayManufacturer: cutoffs.Manufacturer,
		AssayPlatform:     cutoffs.Platform,
		AssayName:         cutoffs.AssayName,
	}

	if baseline > 0 {
		result.PercentChange = ((hour2 - baseline) / baseline) * 100
	}

	// Use 0/2h cutoffs if available, otherwise fall back to 0/1h with adjusted delta
	ruleOutBaseline := cutoffs.RuleOutBaseline_01h
	ruleInBaseline := cutoffs.RuleInBaseline_01h
	ruleInDelta := cutoffs.RuleInDelta_01h * 1.5 // Larger delta expected at 2h

	if cutoffs.RuleOutBaseline_02h != nil {
		ruleOutBaseline = *cutoffs.RuleOutBaseline_02h
	}
	if cutoffs.RuleInBaseline_02h != nil {
		ruleInBaseline = *cutoffs.RuleInBaseline_02h
	}
	if cutoffs.RuleInDelta_02h != nil {
		ruleInDelta = *cutoffs.RuleInDelta_02h
	}

	// Apply algorithm
	if baseline < ruleOutBaseline && result.AbsoluteChange < ruleInDelta/2 {
		result.Classification = "RULE_OUT"
		result.Confidence = "HIGH"
		result.Interpretation = "Low baseline and minimal change at 2h - AMI ruled out."
		result.Recommendations = []string{
			"Consider alternative diagnoses",
			"Safe for discharge if clinically stable",
		}
		result.RequiresObservation = false
	} else if baseline >= ruleInBaseline || result.AbsoluteChange >= ruleInDelta {
		result.Classification = "RULE_IN"
		result.Confidence = "HIGH"
		result.Interpretation = "Elevated troponin or significant delta at 2h - NSTEMI likely."
		result.Recommendations = []string{
			"Initiate ACS protocol",
			"Cardiology consultation",
		}
		result.RequiresObservation = false
	} else {
		result.Classification = "OBSERVE"
		result.Confidence = "MODERATE"
		result.Interpretation = "Indeterminate at 2h - further evaluation needed."
		result.Recommendations = []string{
			"Consider repeat troponin at 3-6 hours",
			"Clinical correlation required",
		}
		result.RequiresObservation = true
	}

	result.Governance = TroponinGovernance{
		Algorithm:       "ESC 0/2h Protocol",
		CutoffSource:    "ESC 2023 NSTE-ACS Guidelines",
		AssaySpecific:   true,
		ManufacturerRef: cutoffs.PackageInsertRef,
		FDAClearanceNum: cutoffs.FDAClearanceNum,
		EvidenceLevel:   "HIGH",
	}

	return result
}

// EvaluateHsTroponin03h evaluates using traditional 3h protocol with 99th percentile URL
func EvaluateHsTroponin03h(baseline, hour3 float64, assay string, sex string) *HsTroponinDeltaResult {
	cutoffs, ok := TroponinAssayCutoffs[assay]
	if !ok {
		return &HsTroponinDeltaResult{
			Algorithm:           "0/3h",
			InitialValue:        baseline,
			DeltaValue:          hour3,
			AbsoluteChange:      math.Abs(hour3 - baseline),
			Classification:      "OBSERVE",
			Confidence:          "LOW",
			Interpretation:      "Unknown assay - use clinical judgment.",
			RequiresObservation: true,
		}
	}

	result := &HsTroponinDeltaResult{
		Algorithm:         "0/3h",
		InitialValue:      baseline,
		DeltaValue:        hour3,
		AbsoluteChange:    math.Abs(hour3 - baseline),
		AssayManufacturer: cutoffs.Manufacturer,
		AssayPlatform:     cutoffs.Platform,
		AssayName:         cutoffs.AssayName,
	}

	// Use sex-specific 99th percentile if available
	url := cutoffs.URL99thPercentile
	if sex == "M" && cutoffs.MaleURL99thPercentile != nil {
		url = *cutoffs.MaleURL99thPercentile
	} else if sex == "F" && cutoffs.FemaleURL99thPercentile != nil {
		url = *cutoffs.FemaleURL99thPercentile
	}

	// Traditional algorithm: Compare to 99th percentile URL with >20% change
	rise := result.AbsoluteChange / baseline * 100 // percent change

	if baseline < url && hour3 < url {
		result.Classification = "RULE_OUT"
		result.Confidence = "HIGH"
		result.Interpretation = "Both values below 99th percentile URL - AMI unlikely."
		result.RequiresObservation = false
	} else if (baseline > url || hour3 > url) && math.Abs(rise) >= 20 {
		result.Classification = "RULE_IN"
		result.Confidence = "HIGH"
		result.Interpretation = "Troponin above URL with significant rise/fall - NSTEMI likely."
		result.RequiresObservation = false
	} else if baseline > url || hour3 > url {
		result.Classification = "OBSERVE"
		result.Confidence = "MODERATE"
		result.Interpretation = "Elevated troponin without significant change - may be chronic elevation."
		result.Recommendations = []string{
			"Consider causes of chronic troponin elevation",
			"Compare to prior baseline if available",
			"Clinical correlation required",
		}
		result.RequiresObservation = true
	} else {
		result.Classification = "RULE_OUT"
		result.Confidence = "MODERATE"
		result.Interpretation = "Values at or below URL - AMI less likely."
		result.RequiresObservation = false
	}

	result.Governance = TroponinGovernance{
		Algorithm:       "Traditional 0/3h Protocol",
		CutoffSource:    "99th Percentile URL per Package Insert",
		AssaySpecific:   true,
		ManufacturerRef: cutoffs.PackageInsertRef,
		EvidenceLevel:   "HIGH",
	}

	return result
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// GetSupportedAssays returns list of supported assay identifiers
func GetSupportedAssays() []string {
	assays := make([]string, 0, len(TroponinAssayCutoffs))
	for k := range TroponinAssayCutoffs {
		assays = append(assays, k)
	}
	return assays
}

// GetAssayCutoffs returns cutoffs for a specific assay
func GetAssayCutoffs(assay string) (*AssayCutoffs, bool) {
	cutoffs, ok := TroponinAssayCutoffs[assay]
	if !ok {
		return nil, false
	}
	return &cutoffs, true
}

// Get99thPercentileURL returns the sex-appropriate 99th percentile URL
func Get99thPercentileURL(assay string, sex string) (float64, bool) {
	cutoffs, ok := TroponinAssayCutoffs[assay]
	if !ok {
		return 0, false
	}

	if sex == "M" && cutoffs.MaleURL99thPercentile != nil {
		return *cutoffs.MaleURL99thPercentile, true
	}
	if sex == "F" && cutoffs.FemaleURL99thPercentile != nil {
		return *cutoffs.FemaleURL99thPercentile, true
	}
	return cutoffs.URL99thPercentile, true
}

// ValidateSampleTiming checks if sample timing is appropriate for the protocol
func ValidateSampleTiming(protocol string, actualMinutes float64) (bool, string) {
	switch protocol {
	case "0/1h":
		if actualMinutes < 50 {
			return false, "Sample taken too early - minimum 50 minutes for 0/1h protocol"
		}
		if actualMinutes > 90 {
			return false, "Sample taken too late for 0/1h protocol - consider 0/2h or 0/3h"
		}
		return true, ""
	case "0/2h":
		if actualMinutes < 110 {
			return false, "Sample taken too early - minimum 110 minutes for 0/2h protocol"
		}
		if actualMinutes > 150 {
			return false, "Sample taken too late for 0/2h protocol - consider 0/3h"
		}
		return true, ""
	case "0/3h":
		if actualMinutes < 170 {
			return false, "Sample taken too early - minimum 170 minutes for 0/3h protocol"
		}
		return true, ""
	default:
		return false, "Unknown protocol"
	}
}

// ptr is a helper function to create a pointer to a float64
func ptr(f float64) *float64 {
	return &f
}
