// Package interpretation provides clinical interpretation algorithms
// abg_panel.go implements arterial blood gas pattern recognition and acid-base analysis
package interpretation

import (
	"math"
)

// =============================================================================
// ABG PANEL INTERPRETATION
// =============================================================================

// ABGPanel represents an arterial blood gas with acid-base interpretation
type ABGPanel struct {
	// Input Values
	pH    float64 `json:"pH"`
	PaCO2 float64 `json:"paCO2"` // mmHg
	HCO3  float64 `json:"hco3"`  // mEq/L (bicarbonate)
	PaO2  float64 `json:"paO2"`  // mmHg
	FiO2  float64 `json:"fio2"`  // Fraction (0.21-1.0)

	// Additional inputs for calculations
	Sodium   float64 `json:"sodium,omitempty"`   // mEq/L
	Chloride float64 `json:"chloride,omitempty"` // mEq/L
	Albumin  float64 `json:"albumin,omitempty"`  // g/dL

	// Calculated Values
	AnionGap          float64 `json:"anionGap"`
	CorrectedAnionGap float64 `json:"correctedAnionGap"` // Albumin-corrected
	DeltaGap          float64 `json:"deltaGap"`          // ΔAG
	DeltaRatio        float64 `json:"deltaRatio"`        // ΔAG/ΔHCO3
	PFRatio           float64 `json:"pfRatio"`           // PaO2/FiO2
	AaGradient        float64 `json:"aaGradient"`        // A-a gradient
	ExpectedPaCO2     float64 `json:"expectedPaCO2"`     // Winter's formula
	ExpectedHCO3      float64 `json:"expectedHCO3"`      // For respiratory disorders

	// Primary Interpretation
	PrimaryDisorder    string `json:"primaryDisorder"`    // METABOLIC_ACIDOSIS, RESPIRATORY_ALKALOSIS, etc.
	SecondaryDisorder  string `json:"secondaryDisorder"`  // For mixed disorders
	CompensationStatus string `json:"compensationStatus"` // APPROPRIATE, INADEQUATE, EXCESSIVE
	OxygenationStatus  string `json:"oxygenationStatus"`  // NORMAL, HYPOXEMIA, SEVERE_HYPOXEMIA

	// Pattern Classification
	Patterns []string `json:"patterns"` // HIGH_ANION_GAP, NON_ANION_GAP, ARDS, etc.

	// Clinical Interpretation
	Interpretation  string   `json:"interpretation"`
	Recommendations []string `json:"recommendations"`

	// Governance
	Governance ABGGovernance `json:"governance"`
}

// ABGGovernance tracks clinical authority for ABG interpretation
type ABGGovernance struct {
	InterpretationMethod string   `json:"interpretationMethod"` // Boston Rules, Copenhagen, Stewart
	ReferenceText        string   `json:"referenceText"`
	FormulaeSources      []string `json:"formulaeSources"`
}

// Normal ranges and thresholds
const (
	// pH
	pHNormalLow  = 7.35
	pHNormalHigh = 7.45

	// PaCO2 (mmHg)
	PaCO2NormalLow  = 35.0
	PaCO2NormalHigh = 45.0

	// HCO3 (mEq/L)
	HCO3NormalLow  = 22.0
	HCO3NormalHigh = 26.0
	HCO3Normal     = 24.0

	// Anion Gap
	AnionGapNormalLow  = 8.0
	AnionGapNormalHigh = 12.0
	AnionGapNormal     = 12.0

	// PaO2/FiO2 Ratio (Berlin ARDS Definition)
	PFRatioMildARDS     = 300.0
	PFRatioModerateARDS = 200.0
	PFRatioSevereARDS   = 100.0
)

// Disorder types
const (
	DisorderMetabolicAcidosis    = "METABOLIC_ACIDOSIS"
	DisorderMetabolicAlkalosis   = "METABOLIC_ALKALOSIS"
	DisorderRespiratoryAcidosis  = "RESPIRATORY_ACIDOSIS"
	DisorderRespiratoryAlkalosis = "RESPIRATORY_ALKALOSIS"
	DisorderMixed                = "MIXED_DISORDER"
	DisorderNormal               = "NORMAL"
)

// Compensation status
const (
	CompensationAppropriate = "APPROPRIATE"
	CompensationInadequate  = "INADEQUATE"
	CompensationExcessive   = "EXCESSIVE"
)

// Oxygenation status
const (
	OxygenationNormal         = "NORMAL"
	OxygenationMildHypoxemia  = "MILD_HYPOXEMIA"
	OxygenationModerate       = "MODERATE_HYPOXEMIA"
	OxygenationSevere         = "SEVERE_HYPOXEMIA"
)

// Pattern types
const (
	PatternHighAnionGap  = "HIGH_ANION_GAP"
	PatternNonAnionGap   = "NON_ANION_GAP"
	PatternMildARDS      = "MILD_ARDS"
	PatternModerateARDS  = "MODERATE_ARDS"
	PatternSevereARDS    = "SEVERE_ARDS"
	PatternMixedHAGMANAG = "MIXED_HAGMA_NAGMA"
	PatternHAGMAMetAlk   = "HAGMA_WITH_METABOLIC_ALKALOSIS"
)

// =============================================================================
// INTERPRETATION FUNCTIONS
// =============================================================================

// InterpretABG performs comprehensive ABG interpretation
func InterpretABG(abg *ABGPanel) *ABGPanel {
	// Step 1: Calculate derived values
	abg.calculateAnionGap()
	abg.calculatePFRatio()
	abg.calculateAaGradient()

	// Step 2: Determine primary disorder
	abg.determinePrimaryDisorder()

	// Step 3: Check compensation
	abg.checkCompensation()

	// Step 4: Delta-Delta analysis for HAGMA
	if contains(abg.Patterns, PatternHighAnionGap) {
		abg.performDeltaDeltaAnalysis()
	}

	// Step 5: Assess oxygenation
	abg.assessOxygenation()

	// Step 6: Generate clinical interpretation
	abg.generateInterpretation()

	// Step 7: Set governance
	abg.Governance = ABGGovernance{
		InterpretationMethod: "Boston Rules + Winter's Formula",
		ReferenceText:        "Tietz Clinical Chemistry 7th ed",
		FormulaeSources: []string{
			"Winter's Formula",
			"Delta-Delta Ratio",
			"Berlin ARDS Definition",
			"Corrected Anion Gap (Figge)",
		},
	}

	return abg
}

// calculateAnionGap calculates anion gap and corrected AG
func (abg *ABGPanel) calculateAnionGap() {
	// Standard anion gap: Na - (Cl + HCO3)
	if abg.Sodium > 0 && abg.Chloride > 0 {
		abg.AnionGap = abg.Sodium - (abg.Chloride + abg.HCO3)
	}

	// Corrected anion gap for albumin (Figge formula)
	// Corrected AG = AG + 2.5 × (4.0 - albumin)
	if abg.Albumin > 0 {
		abg.CorrectedAnionGap = abg.AnionGap + 2.5*(4.0-abg.Albumin)
	} else {
		// Assume normal albumin if not provided
		abg.CorrectedAnionGap = abg.AnionGap
	}
}

// calculatePFRatio calculates PaO2/FiO2 ratio
func (abg *ABGPanel) calculatePFRatio() {
	if abg.FiO2 > 0 {
		abg.PFRatio = abg.PaO2 / abg.FiO2
	}
}

// calculateAaGradient calculates alveolar-arterial gradient
func (abg *ABGPanel) calculateAaGradient() {
	// A-a gradient = PAO2 - PaO2
	// PAO2 = (FiO2 × (Patm - PH2O)) - (PaCO2 / RQ)
	// Simplified at sea level: PAO2 = (FiO2 × 713) - (PaCO2 × 1.25)
	if abg.FiO2 > 0 {
		pao2 := (abg.FiO2 * 713) - (abg.PaCO2 * 1.25)
		abg.AaGradient = pao2 - abg.PaO2
	}
}

// determinePrimaryDisorder identifies the primary acid-base disorder
func (abg *ABGPanel) determinePrimaryDisorder() {
	// Step 1: Look at pH
	if abg.pH >= pHNormalLow && abg.pH <= pHNormalHigh {
		// pH normal - could be normal or compensated disorder
		if abg.PaCO2 >= PaCO2NormalLow && abg.PaCO2 <= PaCO2NormalHigh &&
			abg.HCO3 >= HCO3NormalLow && abg.HCO3 <= HCO3NormalHigh {
			abg.PrimaryDisorder = DisorderNormal
			return
		}
		// Could be fully compensated - analyze further
	}

	// Step 2: Acidemia (pH < 7.35)
	if abg.pH < pHNormalLow {
		if abg.PaCO2 > PaCO2NormalHigh {
			abg.PrimaryDisorder = DisorderRespiratoryAcidosis
		} else if abg.HCO3 < HCO3NormalLow {
			abg.PrimaryDisorder = DisorderMetabolicAcidosis
			abg.classifyMetabolicAcidosis()
		} else {
			// Both abnormal - likely mixed
			abg.PrimaryDisorder = DisorderMixed
		}
		return
	}

	// Step 3: Alkalemia (pH > 7.45)
	if abg.pH > pHNormalHigh {
		if abg.PaCO2 < PaCO2NormalLow {
			abg.PrimaryDisorder = DisorderRespiratoryAlkalosis
		} else if abg.HCO3 > HCO3NormalHigh {
			abg.PrimaryDisorder = DisorderMetabolicAlkalosis
		} else {
			abg.PrimaryDisorder = DisorderMixed
		}
		return
	}

	// Default to normal if criteria not met
	abg.PrimaryDisorder = DisorderNormal
}

// classifyMetabolicAcidosis determines if HAGMA or NAGMA
func (abg *ABGPanel) classifyMetabolicAcidosis() {
	ag := abg.CorrectedAnionGap
	if ag == 0 {
		ag = abg.AnionGap
	}

	if ag > AnionGapNormalHigh {
		abg.Patterns = append(abg.Patterns, PatternHighAnionGap)
	} else {
		abg.Patterns = append(abg.Patterns, PatternNonAnionGap)
	}
}

// checkCompensation evaluates if compensation is appropriate
func (abg *ABGPanel) checkCompensation() {
	switch abg.PrimaryDisorder {
	case DisorderMetabolicAcidosis:
		// Winter's Formula: Expected PaCO2 = 1.5 × HCO3 + 8 (±2)
		abg.ExpectedPaCO2 = 1.5*abg.HCO3 + 8

		if abg.PaCO2 < abg.ExpectedPaCO2-2 {
			abg.CompensationStatus = CompensationExcessive
			abg.SecondaryDisorder = DisorderRespiratoryAlkalosis
		} else if abg.PaCO2 > abg.ExpectedPaCO2+2 {
			abg.CompensationStatus = CompensationInadequate
			abg.SecondaryDisorder = DisorderRespiratoryAcidosis
		} else {
			abg.CompensationStatus = CompensationAppropriate
		}

	case DisorderMetabolicAlkalosis:
		// Expected PaCO2 = 0.7 × HCO3 + 21 (±2)
		abg.ExpectedPaCO2 = 0.7*abg.HCO3 + 21

		if abg.PaCO2 < abg.ExpectedPaCO2-2 {
			abg.CompensationStatus = CompensationExcessive
			abg.SecondaryDisorder = DisorderRespiratoryAlkalosis
		} else if abg.PaCO2 > abg.ExpectedPaCO2+2 {
			abg.CompensationStatus = CompensationInadequate
			abg.SecondaryDisorder = DisorderRespiratoryAcidosis
		} else {
			abg.CompensationStatus = CompensationAppropriate
		}

	case DisorderRespiratoryAcidosis:
		// Acute: Expected HCO3 = 24 + (PaCO2 - 40)/10
		// Chronic: Expected HCO3 = 24 + 3.5 × (PaCO2 - 40)/10
		acuteExpected := 24.0 + (abg.PaCO2-40)/10
		chronicExpected := 24.0 + 3.5*(abg.PaCO2-40)/10

		if abg.HCO3 < acuteExpected-2 {
			abg.CompensationStatus = CompensationInadequate
		} else if abg.HCO3 > chronicExpected+2 {
			abg.CompensationStatus = CompensationExcessive
			abg.SecondaryDisorder = DisorderMetabolicAlkalosis
		} else {
			abg.CompensationStatus = CompensationAppropriate
		}
		abg.ExpectedHCO3 = acuteExpected

	case DisorderRespiratoryAlkalosis:
		// Acute: Expected HCO3 = 24 - 2 × (40 - PaCO2)/10
		// Chronic: Expected HCO3 = 24 - 5 × (40 - PaCO2)/10
		acuteExpected := 24.0 - 2*(40-abg.PaCO2)/10
		chronicExpected := 24.0 - 5*(40-abg.PaCO2)/10

		if abg.HCO3 > acuteExpected+2 {
			abg.CompensationStatus = CompensationInadequate
		} else if abg.HCO3 < chronicExpected-2 {
			abg.CompensationStatus = CompensationExcessive
			abg.SecondaryDisorder = DisorderMetabolicAcidosis
		} else {
			abg.CompensationStatus = CompensationAppropriate
		}
		abg.ExpectedHCO3 = acuteExpected
	}
}

// performDeltaDeltaAnalysis checks for mixed HAGMA + another metabolic disorder
func (abg *ABGPanel) performDeltaDeltaAnalysis() {
	// Delta AG = Corrected AG - 12
	abg.DeltaGap = abg.CorrectedAnionGap - AnionGapNormal

	// Delta HCO3 = 24 - measured HCO3
	deltaHCO3 := HCO3Normal - abg.HCO3

	// Delta Ratio = ΔAG / ΔHCO3
	if deltaHCO3 > 0 {
		abg.DeltaRatio = abg.DeltaGap / deltaHCO3
	}

	// Interpretation:
	// Ratio 1-2: Pure HAGMA
	// Ratio < 1: HAGMA + Non-anion gap acidosis
	// Ratio > 2: HAGMA + Metabolic alkalosis
	if abg.DeltaRatio > 0 {
		if abg.DeltaRatio < 1 {
			abg.Patterns = append(abg.Patterns, PatternMixedHAGMANAG)
			abg.SecondaryDisorder = "NON_ANION_GAP_ACIDOSIS"
		} else if abg.DeltaRatio > 2 {
			abg.Patterns = append(abg.Patterns, PatternHAGMAMetAlk)
			abg.SecondaryDisorder = DisorderMetabolicAlkalosis
		}
	}
}

// assessOxygenation evaluates oxygenation status
func (abg *ABGPanel) assessOxygenation() {
	// Assess P/F ratio (Berlin Definition for ARDS)
	if abg.PFRatio > 0 {
		if abg.PFRatio < PFRatioSevereARDS {
			abg.OxygenationStatus = OxygenationSevere
			abg.Patterns = append(abg.Patterns, PatternSevereARDS)
		} else if abg.PFRatio < PFRatioModerateARDS {
			abg.OxygenationStatus = OxygenationModerate
			abg.Patterns = append(abg.Patterns, PatternModerateARDS)
		} else if abg.PFRatio < PFRatioMildARDS {
			abg.OxygenationStatus = OxygenationMildHypoxemia
			abg.Patterns = append(abg.Patterns, PatternMildARDS)
		} else {
			abg.OxygenationStatus = OxygenationNormal
		}
	} else {
		// Use absolute PaO2 if FiO2 not available
		if abg.PaO2 < 60 {
			abg.OxygenationStatus = OxygenationSevere
		} else if abg.PaO2 < 80 {
			abg.OxygenationStatus = OxygenationMildHypoxemia
		} else {
			abg.OxygenationStatus = OxygenationNormal
		}
	}
}

// generateInterpretation creates clinical interpretation text
func (abg *ABGPanel) generateInterpretation() {
	parts := []string{}

	// Primary disorder
	switch abg.PrimaryDisorder {
	case DisorderMetabolicAcidosis:
		if contains(abg.Patterns, PatternHighAnionGap) {
			parts = append(parts, "High anion gap metabolic acidosis")
			abg.Recommendations = append(abg.Recommendations,
				"Consider MUDPILES: Methanol, Uremia, DKA, Propylene glycol, INH/Iron, Lactic acidosis, Ethylene glycol, Salicylates")
		} else {
			parts = append(parts, "Non-anion gap (hyperchloremic) metabolic acidosis")
			abg.Recommendations = append(abg.Recommendations,
				"Consider HARDUPS: Hyperalimentation, Addison's, RTA, Diarrhea, Ureteral diversion, Pancreatic fistula, Saline infusion")
		}
	case DisorderMetabolicAlkalosis:
		parts = append(parts, "Metabolic alkalosis")
		abg.Recommendations = append(abg.Recommendations,
			"Assess volume status and check urine chloride")
	case DisorderRespiratoryAcidosis:
		parts = append(parts, "Respiratory acidosis")
		abg.Recommendations = append(abg.Recommendations,
			"Assess for hypoventilation - consider CNS, neuromuscular, or airway pathology")
	case DisorderRespiratoryAlkalosis:
		parts = append(parts, "Respiratory alkalosis")
		abg.Recommendations = append(abg.Recommendations,
			"Assess for hyperventilation - consider anxiety, pain, PE, sepsis, hepatic encephalopathy")
	case DisorderNormal:
		parts = append(parts, "Normal acid-base status")
	case DisorderMixed:
		parts = append(parts, "Mixed acid-base disorder")
	}

	// Compensation
	if abg.CompensationStatus != "" && abg.PrimaryDisorder != DisorderNormal {
		parts = append(parts, "with "+string(abg.CompensationStatus)+" compensation")
	}

	// Secondary disorder
	if abg.SecondaryDisorder != "" {
		parts = append(parts, "plus "+abg.SecondaryDisorder)
	}

	// Oxygenation
	if abg.OxygenationStatus != OxygenationNormal && abg.OxygenationStatus != "" {
		parts = append(parts, ". "+abg.OxygenationStatus)
		if contains(abg.Patterns, PatternSevereARDS) {
			abg.Recommendations = append(abg.Recommendations,
				"Severe ARDS - consider lung protective ventilation, prone positioning")
		}
	}

	abg.Interpretation = joinStrings(parts, " ")
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// joinStrings joins strings with a separator
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// NewABGPanel creates a new ABG panel with input values
func NewABGPanel(pH, paCO2, hco3, paO2, fio2 float64) *ABGPanel {
	return &ABGPanel{
		pH:    pH,
		PaCO2: paCO2,
		HCO3:  hco3,
		PaO2:  paO2,
		FiO2:  fio2,
	}
}

// NewABGPanelWithElectrolytes creates ABG panel with electrolytes for anion gap
func NewABGPanelWithElectrolytes(pH, paCO2, hco3, paO2, fio2, sodium, chloride, albumin float64) *ABGPanel {
	return &ABGPanel{
		pH:       pH,
		PaCO2:    paCO2,
		HCO3:     hco3,
		PaO2:     paO2,
		FiO2:     fio2,
		Sodium:   sodium,
		Chloride: chloride,
		Albumin:  albumin,
	}
}

// CalculateExpectedPaCO2 calculates expected PaCO2 using Winter's formula
func CalculateExpectedPaCO2(hco3 float64) (expected float64, rangeMin float64, rangeMax float64) {
	expected = 1.5*hco3 + 8
	rangeMin = expected - 2
	rangeMax = expected + 2
	return
}

// CalculateAnionGap calculates standard anion gap
func CalculateAnionGap(sodium, chloride, hco3 float64) float64 {
	return sodium - (chloride + hco3)
}

// CalculateCorrectedAnionGap calculates albumin-corrected anion gap
func CalculateCorrectedAnionGap(anionGap, albumin float64) float64 {
	// Figge formula: Corrected AG = AG + 2.5 × (4.0 - albumin)
	return anionGap + 2.5*(4.0-albumin)
}

// CalculateOsmolalGap calculates osmolar gap for toxic alcohol screening
func CalculateOsmolalGap(measuredOsm, sodium, bun, glucose, ethanol float64) float64 {
	// Calculated osmolality = 2×Na + BUN/2.8 + Glucose/18 + Ethanol/4.6
	calcOsm := 2*sodium + bun/2.8 + glucose/18
	if ethanol > 0 {
		calcOsm += ethanol / 4.6
	}
	return measuredOsm - calcOsm
}

// IsOsmolalGapElevated checks if osmolar gap suggests toxic alcohol
func IsOsmolalGapElevated(gap float64) bool {
	// Normal osmolar gap: -10 to +10 mOsm/kg
	// Gap > 10 suggests toxic alcohols (methanol, ethylene glycol)
	return math.Abs(gap) > 10
}
