// Package metabolic implements the MetabolicPhysiologyEngine (KB-24 / Commitment 8).
//
// CRITICAL BUILD CONSTRAINT:
// This package is SEPARATE from channel_b. The build-time import constraint
// (channel_b/import_constraint_test.go) ensures Channel B never imports this package.
// Channel B operates on raw labs only; this package provides optimization intelligence.
//
// The MetabolicPhysiologyEngine computes:
//   - MetabolicState classification
//   - Insulin Sensitivity Factor (ISF) estimation
//   - HyperglycaemiaMechanism classification
//   - Dawn phenomenon detection
//   - Control gain modulation inputs
package metabolic

import "time"

// MetabolicState classifies the patient's current metabolic context.
type MetabolicState string

const (
	MetabolicFasting        MetabolicState = "FASTING"
	MetabolicPostprandial   MetabolicState = "POSTPRANDIAL"
	MetabolicStressFasting  MetabolicState = "STRESS_FASTING"
	MetabolicDawnPhenomenon MetabolicState = "DAWN_PHENOMENON"
	MetabolicExercise       MetabolicState = "EXERCISE"
	MetabolicUnknown        MetabolicState = "UNKNOWN"
)

// HyperglycaemiaMechanism classifies the root cause of elevated glucose.
//
// SPECIFICATION NOTE — 7 TYPES vs SPEC'S 4:
// The original specification (Supplementary Addendum B-02) defined 4 classes:
//   1. Fasting hyperglycaemia      → mapped to INSULIN_DEFICIENCY or HEPATIC_OUTPUT
//   2. Post-prandial hyperglycaemia → mapped to INSULIN_RESISTANCE
//   3. Dawn phenomenon              → mapped to DAWN_PHENOMENON
//   4. Stress hyperglycaemia        → mapped to STRESS_RESPONSE
//
// This implementation extends to 7 types for finer-grained ISF estimation:
//   - INSULIN_DEFICIENCY: Low C-peptide (<0.6 ng/mL) — beta-cell failure.
//     The spec's "fasting hyperglycaemia" when cause is insufficient insulin production.
//   - INSULIN_RESISTANCE: High HOMA-IR (>2.5) or high C-peptide + high BMI.
//     The spec's "post-prandial hyperglycaemia" — insulin is produced but ineffective.
//   - HEPATIC_OUTPUT: Excessive hepatic glucose production during fasting.
//     A refinement of the spec's "fasting hyperglycaemia" — the cause is liver, not beta-cell.
//   - STRESS_RESPONSE: Elevated glucose during illness/physiological stress.
//     Maps directly to spec's "stress hyperglycaemia".
//   - DAWN_PHENOMENON: Pre-breakfast glucose rise (03:00–08:00) driven by cortisol/GH.
//     Maps directly to spec's "dawn phenomenon".
//   - MIXED: Multiple mechanisms contributing simultaneously.
//     Not in original spec. Added because clinical reality often involves overlapping causes.
//   - UNDETERMINED: Insufficient data to classify.
//     Not in original spec. Added as a safe default when lab data is incomplete.
//
// BACKWARD COMPATIBILITY:
// ComputeDose() does NOT branch on mechanism type — it uses SuggestedGainAdj (float64).
// KB-23 template gate rules reference the 4 original categories. The 7-type classification
// is a superset: any downstream consumer expecting the original 4 can map INSULIN_DEFICIENCY
// and HEPATIC_OUTPUT → "fasting", INSULIN_RESISTANCE → "postprandial", and treat MIXED/
// UNDETERMINED as the most conservative gain adjustment (1.0 = no change).
type HyperglycaemiaMechanism string

const (
	MechanismInsulinDeficiency HyperglycaemiaMechanism = "INSULIN_DEFICIENCY"  // Spec: fasting hyperglycaemia (beta-cell)
	MechanismInsulinResistance HyperglycaemiaMechanism = "INSULIN_RESISTANCE"  // Spec: post-prandial hyperglycaemia
	MechanismHepatic           HyperglycaemiaMechanism = "HEPATIC_OUTPUT"      // Spec: fasting hyperglycaemia (hepatic)
	MechanismStressResponse    HyperglycaemiaMechanism = "STRESS_RESPONSE"     // Spec: stress hyperglycaemia
	MechanismDawnPhenomenon    HyperglycaemiaMechanism = "DAWN_PHENOMENON"     // Spec: dawn phenomenon
	MechanismMixed             HyperglycaemiaMechanism = "MIXED"               // Extension: overlapping mechanisms
	MechanismUndetermined      HyperglycaemiaMechanism = "UNDETERMINED"        // Extension: insufficient data
)

// MetabolicInput is the data required for metabolic state classification.
// Sourced from KB-20 (labs), KB-22 (derived metrics), and KB-23 (enrichment).
type MetabolicInput struct {
	// Current glucose context
	GlucoseCurrent    float64
	GlucoseTimestamp  time.Time
	FastingDuration   *float64 // hours since last meal, nil if unknown

	// Insulin and C-peptide (when available)
	InsulinLevel      *float64 // mU/L
	CPeptideLevel     *float64 // ng/mL

	// Derived metrics
	HOMAIR            *float64 // Homeostatic Model Assessment of Insulin Resistance
	BMI               float64

	// Temporal context
	TimeOfDay         time.Time
	IsPreBreakfast    bool
	LastMealTimestamp  *time.Time

	// Treatment context
	CurrentInsulinDose float64
	InsulinType        string // "basal", "rapid", "premix"
	DaysOnTherapy      int
}

// MetabolicOutput contains the engine's assessment.
type MetabolicOutput struct {
	State              MetabolicState          `json:"metabolic_state"`
	Mechanism          HyperglycaemiaMechanism `json:"hyperglycaemia_mechanism"`
	ISF                float64                 `json:"isf"`                 // Insulin Sensitivity Factor (mg/dL per unit)
	DawnPhenomenon     bool                    `json:"dawn_phenomenon"`
	SuggestedGainAdj   float64                 `json:"suggested_gain_adj"` // multiplier for control gain
	ConfidenceScore    float64                 `json:"confidence_score"`   // 0.0 - 1.0
}

// Engine computes metabolic state and mechanism classification.
// This is the optimisation module — it refines dose recommendations
// but does NOT override safety channel decisions.
type Engine struct {
	// ISF estimation parameters
	DefaultISF          float64 // default: 50 mg/dL per unit (type 2)
	DawnPhenomenonStart int     // hour of day (default: 3)
	DawnPhenomenonEnd   int     // hour of day (default: 8)
}

// NewEngine creates a MetabolicPhysiologyEngine with defaults.
func NewEngine() *Engine {
	return &Engine{
		DefaultISF:          50.0,
		DawnPhenomenonStart: 3,
		DawnPhenomenonEnd:   8,
	}
}

// Classify performs metabolic state classification and ISF estimation.
func (e *Engine) Classify(input MetabolicInput) MetabolicOutput {
	state := e.classifyState(input)
	mechanism := e.classifyMechanism(input, state)
	isf := e.estimateISF(input)
	dawn := e.detectDawnPhenomenon(input)

	gainAdj := 1.0
	if dawn {
		gainAdj = 0.7 // reduce gain during dawn phenomenon
	}
	if state == MetabolicStressFasting {
		gainAdj = 0.5 // conservative during stress
	}

	confidence := e.computeConfidence(input)

	return MetabolicOutput{
		State:            state,
		Mechanism:        mechanism,
		ISF:              isf,
		DawnPhenomenon:   dawn,
		SuggestedGainAdj: gainAdj,
		ConfidenceScore:  confidence,
	}
}

func (e *Engine) classifyState(input MetabolicInput) MetabolicState {
	hour := input.TimeOfDay.Hour()

	// Dawn phenomenon window check
	if input.IsPreBreakfast && hour >= e.DawnPhenomenonStart && hour < e.DawnPhenomenonEnd {
		if input.GlucoseCurrent > 7.0 { // elevated fasting glucose in dawn window
			return MetabolicDawnPhenomenon
		}
	}

	// Fasting state
	if input.FastingDuration != nil && *input.FastingDuration > 8.0 {
		return MetabolicFasting
	}

	// Postprandial state
	if input.LastMealTimestamp != nil {
		hoursSinceMeal := input.GlucoseTimestamp.Sub(*input.LastMealTimestamp).Hours()
		if hoursSinceMeal < 4.0 {
			return MetabolicPostprandial
		}
	}

	// Stress-fasting (high glucose despite fasting)
	if input.FastingDuration != nil && *input.FastingDuration > 4.0 && input.GlucoseCurrent > 11.0 {
		return MetabolicStressFasting
	}

	return MetabolicUnknown
}

func (e *Engine) classifyMechanism(input MetabolicInput, state MetabolicState) HyperglycaemiaMechanism {
	if state == MetabolicDawnPhenomenon {
		return MechanismDawnPhenomenon
	}

	// If HOMA-IR available, use insulin resistance assessment
	if input.HOMAIR != nil {
		if *input.HOMAIR > 2.5 {
			return MechanismInsulinResistance
		}
	}

	// C-peptide based classification
	if input.CPeptideLevel != nil {
		if *input.CPeptideLevel < 0.6 {
			return MechanismInsulinDeficiency
		}
		if *input.CPeptideLevel > 3.0 && input.BMI > 30 {
			return MechanismInsulinResistance
		}
	}

	if state == MetabolicStressFasting {
		return MechanismStressResponse
	}

	return MechanismUndetermined
}

func (e *Engine) estimateISF(input MetabolicInput) float64 {
	// Base ISF estimation using the 1800 rule for type 2
	if input.CurrentInsulinDose > 0 {
		isf := 1800.0 / input.CurrentInsulinDose
		// Clamp to clinically reasonable range
		if isf < 10 {
			isf = 10
		}
		if isf > 200 {
			isf = 200
		}
		return isf
	}
	return e.DefaultISF
}

func (e *Engine) detectDawnPhenomenon(input MetabolicInput) bool {
	hour := input.TimeOfDay.Hour()
	return input.IsPreBreakfast &&
		hour >= e.DawnPhenomenonStart &&
		hour < e.DawnPhenomenonEnd &&
		input.GlucoseCurrent > 7.0
}

func (e *Engine) computeConfidence(input MetabolicInput) float64 {
	score := 0.3 // base confidence from glucose alone

	if input.FastingDuration != nil {
		score += 0.2
	}
	if input.InsulinLevel != nil {
		score += 0.15
	}
	if input.CPeptideLevel != nil {
		score += 0.15
	}
	if input.HOMAIR != nil {
		score += 0.1
	}
	if input.LastMealTimestamp != nil {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}
