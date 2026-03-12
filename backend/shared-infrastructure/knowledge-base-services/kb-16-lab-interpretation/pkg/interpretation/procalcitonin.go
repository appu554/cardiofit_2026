// Package interpretation provides clinical interpretation algorithms
// procalcitonin.go implements IDSA 2023 procalcitonin guidance for antibiotic stewardship
package interpretation

import (
	"time"
)

// =============================================================================
// PROCALCITONIN INTERPRETATION (IDSA 2023)
// =============================================================================

// ProcalcitoninInterpretation contains PCT interpretation and antibiotic guidance
type ProcalcitoninInterpretation struct {
	// Input Values
	Value         float64    `json:"value"`                   // ng/mL
	PreviousValue *float64   `json:"previousValue,omitempty"` // Prior PCT for de-escalation
	HoursElapsed  float64    `json:"hoursElapsed,omitempty"`  // Hours since previous
	PreviousTime  *time.Time `json:"previousTime,omitempty"`

	// Clinical Context
	ClinicalContext string `json:"clinicalContext"` // CAP, HAP, SEPSIS, LRTI, OTHER

	// Interpretation
	BacterialLikelihood string `json:"bacterialLikelihood"` // LOW, MODERATE, HIGH, VERY_HIGH
	AntibioticGuidance  string `json:"antibioticGuidance"`  // DISCOURAGE, CONSIDER, RECOMMEND, CONTINUE
	DeEscalationSafe    *bool  `json:"deEscalationSafe,omitempty"`

	// Calculated (for serial monitoring)
	PercentChange float64 `json:"percentChange,omitempty"`

	// Clinical Output
	Interpretation  string   `json:"interpretation"`
	Recommendations []string `json:"recommendations"`
	Caveats         []string `json:"caveats,omitempty"`

	// Governance
	Governance PCTGovernance `json:"governance"`
}

// PCTGovernance tracks clinical authority for PCT interpretation
type PCTGovernance struct {
	GuidelineSource    string `json:"guidelineSource"`
	ClinicalContext    string `json:"clinicalContext"`
	EvidenceLevel      string `json:"evidenceLevel"`
	RecommendationGrade string `json:"recommendationGrade,omitempty"`
}

// PCT Thresholds by clinical context (IDSA 2023)
// Different thresholds apply depending on clinical scenario
const (
	// Community-Acquired Pneumonia (CAP)
	PCTCAPRuleOut        = 0.1  // <0.1 ng/mL - bacterial infection unlikely
	PCTCAPIntermediate   = 0.25 // 0.1-0.25 ng/mL - consider clinical picture
	PCTCAPLikely         = 0.25 // ≥0.25 ng/mL - bacterial infection likely

	// Sepsis / ICU
	PCTSepsisLow         = 0.5  // <0.5 ng/mL - low probability
	PCTSepsisIntermediate = 2.0 // 0.5-2.0 ng/mL - moderate probability
	PCTSepsisHigh        = 2.0  // ≥2.0 ng/mL - high probability
	PCTSepsisVeryHigh    = 10.0 // ≥10.0 ng/mL - very high, severe sepsis/shock

	// De-escalation thresholds
	PCTDeEscalationDrop  = 80.0  // ≥80% drop from peak = safe to stop
	PCTDeEscalationLevel = 0.5   // OR absolute <0.5 ng/mL
	PCTMinDropForStop    = 0.25  // OR <0.25 ng/mL regardless of % change
)

// Clinical context types
const (
	ContextCAP     = "CAP"     // Community-acquired pneumonia
	ContextHAP     = "HAP"     // Hospital-acquired pneumonia
	ContextSepsis  = "SEPSIS"  // Sepsis/septic shock
	ContextLRTI    = "LRTI"    // Lower respiratory tract infection
	ContextUTI     = "UTI"     // Urinary tract infection
	ContextOther   = "OTHER"
)

// Bacterial likelihood levels
const (
	LikelihoodVeryLow  = "VERY_LOW"
	LikelihoodLow      = "LOW"
	LikelihoodModerate = "MODERATE"
	LikelihoodHigh     = "HIGH"
	LikelihoodVeryHigh = "VERY_HIGH"
)

// Antibiotic guidance levels
const (
	GuidanceDiscourage = "DISCOURAGE"  // Strongly discourage antibiotics
	GuidanceConsider   = "CONSIDER"    // Consider clinical context
	GuidanceRecommend  = "RECOMMEND"   // Antibiotics recommended
	GuidanceContinue   = "CONTINUE"    // Continue current antibiotics
	GuidanceDeEscalate = "DE_ESCALATE" // Safe to de-escalate/stop
)

// =============================================================================
// INTERPRETATION FUNCTIONS
// =============================================================================

// InterpretProCalcitonin interprets PCT value based on clinical context
func InterpretProCalcitonin(value float64, previous *float64, hours float64, context string) *ProcalcitoninInterpretation {
	result := &ProcalcitoninInterpretation{
		Value:           value,
		PreviousValue:   previous,
		HoursElapsed:    hours,
		ClinicalContext: context,
	}

	// Calculate percent change if previous value available
	if previous != nil && *previous > 0 {
		result.PercentChange = ((*previous - value) / *previous) * 100
	}

	// Interpret based on clinical context
	switch context {
	case ContextCAP, ContextLRTI:
		result.interpretCAP()
	case ContextSepsis:
		result.interpretSepsis()
	case ContextHAP:
		result.interpretHAP()
	default:
		result.interpretGeneral()
	}

	// Check for de-escalation opportunity
	result.checkDeEscalation()

	// Add caveats
	result.addCaveats()

	// Set governance
	result.Governance = PCTGovernance{
		GuidelineSource:    "IDSA 2023 Procalcitonin Guidance",
		ClinicalContext:    context,
		EvidenceLevel:      "MODERATE (2B)",
		RecommendationGrade: "B",
	}

	return result
}

// interpretCAP interprets PCT for community-acquired pneumonia
func (pct *ProcalcitoninInterpretation) interpretCAP() {
	if pct.Value < PCTCAPRuleOut {
		pct.BacterialLikelihood = LikelihoodVeryLow
		pct.AntibioticGuidance = GuidanceDiscourage
		pct.Interpretation = "Very low PCT (<0.1 ng/mL) - bacterial pneumonia unlikely."
		pct.Recommendations = []string{
			"Antibiotics strongly discouraged based on PCT",
			"Consider viral etiology or alternative diagnosis",
			"Retest in 6-12 hours if clinical concern persists",
			"Do not use PCT alone to exclude pneumonia in critically ill",
		}
	} else if pct.Value < PCTCAPIntermediate {
		pct.BacterialLikelihood = LikelihoodLow
		pct.AntibioticGuidance = GuidanceConsider
		pct.Interpretation = "Low PCT (0.1-0.25 ng/mL) - bacterial infection possible but unlikely."
		pct.Recommendations = []string{
			"Antibiotics discouraged unless high clinical suspicion",
			"Consider clinical picture, imaging, and other markers",
			"Repeat PCT in 6-12 hours if clinically uncertain",
			"Atypical pathogens may have lower PCT elevation",
		}
	} else if pct.Value < 0.5 {
		pct.BacterialLikelihood = LikelihoodModerate
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Moderately elevated PCT (0.25-0.5 ng/mL) - bacterial infection likely."
		pct.Recommendations = []string{
			"Antibiotics recommended",
			"Follow serial PCT to guide duration",
			"De-escalate when PCT drops ≥80% or <0.25",
		}
	} else {
		pct.BacterialLikelihood = LikelihoodHigh
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Elevated PCT (≥0.5 ng/mL) - bacterial infection highly likely."
		pct.Recommendations = []string{
			"Antibiotics strongly recommended",
			"Consider blood cultures before antibiotics",
			"Serial PCT monitoring for de-escalation",
		}
	}
}

// interpretSepsis interprets PCT for sepsis/ICU context
func (pct *ProcalcitoninInterpretation) interpretSepsis() {
	if pct.Value < PCTSepsisLow {
		pct.BacterialLikelihood = LikelihoodLow
		pct.AntibioticGuidance = GuidanceConsider
		pct.Interpretation = "Low PCT (<0.5 ng/mL) in sepsis workup - bacterial etiology less likely."
		pct.Recommendations = []string{
			"Bacterial sepsis less likely but not excluded",
			"Consider viral, fungal, or localized infection",
			"Clinical context and other markers essential",
			"Repeat in 6-12 hours if clinical suspicion persists",
		}
	} else if pct.Value < PCTSepsisHigh {
		pct.BacterialLikelihood = LikelihoodModerate
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Moderately elevated PCT (0.5-2.0 ng/mL) - bacterial sepsis possible."
		pct.Recommendations = []string{
			"Bacterial infection moderately likely",
			"Continue antibiotics pending cultures",
			"Serial PCT for de-escalation guidance",
		}
	} else if pct.Value < PCTSepsisVeryHigh {
		pct.BacterialLikelihood = LikelihoodHigh
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Significantly elevated PCT (2.0-10.0 ng/mL) - bacterial sepsis highly likely."
		pct.Recommendations = []string{
			"Bacterial sepsis highly likely",
			"Continue broad-spectrum antibiotics",
			"Source control assessment",
			"Daily PCT for response monitoring",
		}
	} else {
		pct.BacterialLikelihood = LikelihoodVeryHigh
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Very high PCT (≥10 ng/mL) - severe bacterial sepsis/septic shock."
		pct.Recommendations = []string{
			"Severe bacterial sepsis highly likely",
			"Aggressive broad-spectrum coverage",
			"Urgent source control",
			"Consider sepsis bundle completion",
			"Poor prognosis if PCT fails to clear",
		}
	}
}

// interpretHAP interprets PCT for hospital-acquired pneumonia
func (pct *ProcalcitoninInterpretation) interpretHAP() {
	// HAP thresholds similar to CAP but with higher caution
	if pct.Value < 0.1 {
		pct.BacterialLikelihood = LikelihoodLow
		pct.AntibioticGuidance = GuidanceDiscourage
		pct.Interpretation = "Very low PCT in HAP evaluation - bacterial VAP/HAP unlikely."
		pct.Recommendations = []string{
			"Bacterial HAP/VAP unlikely with very low PCT",
			"Consider non-infectious causes of infiltrate",
			"Clinical context remains important",
			"Do not delay antibiotics in shock",
		}
	} else if pct.Value < 0.5 {
		pct.BacterialLikelihood = LikelihoodModerate
		pct.AntibioticGuidance = GuidanceConsider
		pct.Interpretation = "Low-moderate PCT in HAP - clinical judgment required."
		pct.Recommendations = []string{
			"Bacterial etiology possible",
			"Consider clinical trajectory and imaging",
			"Narrow spectrum if started empirically",
		}
	} else {
		pct.BacterialLikelihood = LikelihoodHigh
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Elevated PCT in HAP - bacterial infection likely."
		pct.Recommendations = []string{
			"Bacterial HAP/VAP likely",
			"Treat per HAP/VAP guidelines",
			"Serial PCT for de-escalation",
		}
	}
}

// interpretGeneral provides general PCT interpretation
func (pct *ProcalcitoninInterpretation) interpretGeneral() {
	if pct.Value < 0.1 {
		pct.BacterialLikelihood = LikelihoodVeryLow
		pct.AntibioticGuidance = GuidanceDiscourage
		pct.Interpretation = "Very low PCT - systemic bacterial infection unlikely."
		pct.Recommendations = []string{
			"Systemic bacterial infection unlikely",
			"Consider alternative diagnoses",
			"Local infection possible despite low PCT",
		}
	} else if pct.Value < 0.25 {
		pct.BacterialLikelihood = LikelihoodLow
		pct.AntibioticGuidance = GuidanceConsider
		pct.Interpretation = "Low PCT - bacterial infection possible but unlikely."
		pct.Recommendations = []string{
			"Bacterial infection possible",
			"Clinical correlation required",
		}
	} else if pct.Value < 0.5 {
		pct.BacterialLikelihood = LikelihoodModerate
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Moderately elevated PCT - bacterial infection likely."
		pct.Recommendations = []string{
			"Bacterial infection likely",
			"Antibiotics generally recommended",
		}
	} else {
		pct.BacterialLikelihood = LikelihoodHigh
		pct.AntibioticGuidance = GuidanceRecommend
		pct.Interpretation = "Elevated PCT - bacterial infection highly likely."
		pct.Recommendations = []string{
			"Bacterial infection highly likely",
			"Antibiotics recommended",
		}
	}
}

// checkDeEscalation assesses if de-escalation is safe
func (pct *ProcalcitoninInterpretation) checkDeEscalation() {
	if pct.PreviousValue == nil {
		return // Cannot assess without prior value
	}

	prev := *pct.PreviousValue

	// De-escalation criteria (IDSA 2023):
	// 1. ≥80% drop from peak, OR
	// 2. Absolute value <0.5 ng/mL, OR
	// 3. Absolute value <0.25 ng/mL (strong)

	if pct.Value < PCTMinDropForStop {
		safe := true
		pct.DeEscalationSafe = &safe
		pct.AntibioticGuidance = GuidanceDeEscalate
		pct.Recommendations = append(pct.Recommendations,
			"PCT <0.25 ng/mL - safe to discontinue antibiotics")
	} else if pct.PercentChange >= PCTDeEscalationDrop {
		safe := true
		pct.DeEscalationSafe = &safe
		pct.AntibioticGuidance = GuidanceDeEscalate
		pct.Recommendations = append(pct.Recommendations,
			"PCT dropped ≥80% - consider stopping antibiotics")
	} else if pct.Value < PCTDeEscalationLevel && prev >= PCTDeEscalationLevel {
		safe := true
		pct.DeEscalationSafe = &safe
		pct.AntibioticGuidance = GuidanceDeEscalate
		pct.Recommendations = append(pct.Recommendations,
			"PCT <0.5 ng/mL (down from elevated) - consider de-escalation")
	} else if pct.PercentChange < 20 {
		safe := false
		pct.DeEscalationSafe = &safe
		pct.Recommendations = append(pct.Recommendations,
			"PCT not adequately declining (<20% drop) - continue antibiotics")
	}
}

// addCaveats adds important caveats to PCT interpretation
func (pct *ProcalcitoninInterpretation) addCaveats() {
	// Common caveats regardless of result
	pct.Caveats = append(pct.Caveats,
		"PCT should not be used in isolation - integrate with clinical picture",
		"PCT may be elevated in non-infectious conditions (trauma, surgery, burns, cardiogenic shock)",
	)

	// Context-specific caveats
	switch pct.ClinicalContext {
	case ContextCAP:
		pct.Caveats = append(pct.Caveats,
			"Atypical pathogens (Mycoplasma, Chlamydia) may not elevate PCT significantly",
			"Localized infections may have low PCT despite bacterial etiology",
		)
	case ContextSepsis:
		pct.Caveats = append(pct.Caveats,
			"Do not delay antibiotics in septic shock pending PCT",
			"Fungal sepsis may have lower PCT elevation",
			"Chronic renal failure can elevate baseline PCT",
		)
	case ContextHAP:
		pct.Caveats = append(pct.Caveats,
			"Immunocompromised patients may have blunted PCT response",
			"Recent surgery may cause PCT elevation",
		)
	}

	// Low value caveats
	if pct.Value < 0.1 {
		pct.Caveats = append(pct.Caveats,
			"Very low PCT does not exclude localized infection",
			"Early infection may not yet have elevated PCT - retest if concern persists",
		)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// InterpretPCTForCAP is a convenience function for CAP interpretation
func InterpretPCTForCAP(value float64) *ProcalcitoninInterpretation {
	return InterpretProCalcitonin(value, nil, 0, ContextCAP)
}

// InterpretPCTForSepsis is a convenience function for sepsis interpretation
func InterpretPCTForSepsis(value float64) *ProcalcitoninInterpretation {
	return InterpretProCalcitonin(value, nil, 0, ContextSepsis)
}

// AssessPCTTrend evaluates a series of PCT measurements
func AssessPCTTrend(values []float64, times []time.Time) *ProcalcitoninInterpretation {
	if len(values) < 2 {
		return nil
	}

	latest := values[len(values)-1]
	previous := values[len(values)-2]
	hours := times[len(times)-1].Sub(times[len(times)-2]).Hours()

	return InterpretProCalcitonin(latest, &previous, hours, ContextSepsis)
}

// IsPCTElevated checks if PCT is elevated above threshold
func IsPCTElevated(value float64, context string) bool {
	switch context {
	case ContextCAP, ContextLRTI:
		return value >= PCTCAPLikely
	case ContextSepsis:
		return value >= PCTSepsisLow
	default:
		return value >= 0.25
	}
}

// GetPCTThreshold returns the threshold for a given context
func GetPCTThreshold(context string, level string) float64 {
	switch context {
	case ContextCAP, ContextLRTI:
		switch level {
		case "rule_out":
			return PCTCAPRuleOut
		case "likely":
			return PCTCAPLikely
		default:
			return PCTCAPIntermediate
		}
	case ContextSepsis:
		switch level {
		case "low":
			return PCTSepsisLow
		case "high":
			return PCTSepsisHigh
		case "very_high":
			return PCTSepsisVeryHigh
		default:
			return PCTSepsisIntermediate
		}
	default:
		return 0.25
	}
}
