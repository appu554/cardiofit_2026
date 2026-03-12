// Package trending provides multi-window trend analysis for lab results
package trending

import (
	"kb-16-lab-interpretation/pkg/types"
)

// =============================================================================
// CLINICAL CONTEXT FOR TREND INTERPRETATION
// =============================================================================

// TrendDirection indicates whether increasing values are clinically good or bad
type TrendDirection string

const (
	DirectionHigherBetter TrendDirection = "HIGHER_BETTER"  // e.g., Hemoglobin after anemia treatment
	DirectionLowerBetter  TrendDirection = "LOWER_BETTER"   // e.g., Creatinine, Glucose
	DirectionMidOptimal   TrendDirection = "MID_OPTIMAL"    // e.g., Potassium (both extremes bad)
	DirectionContextual   TrendDirection = "CONTEXTUAL"     // Depends on clinical scenario
)

// LabTrendContext provides clinical context for interpreting trends
type LabTrendContext struct {
	Code                  string         `json:"code"`
	Name                  string         `json:"name"`
	Direction             TrendDirection `json:"direction"`
	ClinicalSignificance  float64        `json:"clinical_significance"` // Per-unit change significance
	VolatilityThreshold   float64        `json:"volatility_threshold"`  // CV above this is concerning
	MinClinicalChange     float64        `json:"min_clinical_change"`   // Minimum change to be clinically relevant
	Unit                  string         `json:"unit"`
	InterpretationNotes   string         `json:"interpretation_notes,omitempty"`
}

// LabContextDatabase provides clinical context for trend interpretation
var LabContextDatabase = map[string]LabTrendContext{
	// CHEMISTRY - Renal/Metabolic
	"2160-0": { // Creatinine
		Code:                 "2160-0",
		Name:                 "Creatinine",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 0.3,      // 0.3 mg/dL change is significant
		VolatilityThreshold:  0.25,     // 25% CV is concerning
		MinClinicalChange:    0.2,      // <0.2 change may be lab variation
		Unit:                 "mg/dL",
		InterpretationNotes:  "Rising creatinine suggests declining kidney function. Acute rise >0.3 mg/dL in 48h may indicate AKI.",
	},
	"3094-0": { // BUN
		Code:                 "3094-0",
		Name:                 "BUN",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 5.0,
		VolatilityThreshold:  0.3,
		MinClinicalChange:    3.0,
		Unit:                 "mg/dL",
		InterpretationNotes:  "Rising BUN may indicate dehydration, kidney dysfunction, or GI bleeding.",
	},
	"2345-7": { // Glucose
		Code:                 "2345-7",
		Name:                 "Glucose",
		Direction:            DirectionLowerBetter, // For diabetics, generally want lower
		ClinicalSignificance: 30.0,
		VolatilityThreshold:  0.35,
		MinClinicalChange:    20.0,
		Unit:                 "mg/dL",
		InterpretationNotes:  "High volatility suggests poor glycemic control. Trend toward normal indicates improved management.",
	},
	"4548-4": { // HbA1c
		Code:                 "4548-4",
		Name:                 "HbA1c",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 0.5,
		VolatilityThreshold:  0.15,
		MinClinicalChange:    0.3,
		Unit:                 "%",
		InterpretationNotes:  "Reflects 2-3 month glucose average. Each 1% decrease reduces microvascular complication risk.",
	},

	// CHEMISTRY - Electrolytes (Mid-optimal)
	"2823-3": { // Potassium
		Code:                 "2823-3",
		Name:                 "Potassium",
		Direction:            DirectionMidOptimal,
		ClinicalSignificance: 0.5,
		VolatilityThreshold:  0.15,
		MinClinicalChange:    0.3,
		Unit:                 "mEq/L",
		InterpretationNotes:  "Both hypo/hyperkalemia are dangerous. Rapid changes require immediate attention.",
	},
	"2951-2": { // Sodium
		Code:                 "2951-2",
		Name:                 "Sodium",
		Direction:            DirectionMidOptimal,
		ClinicalSignificance: 3.0,
		VolatilityThreshold:  0.03,
		MinClinicalChange:    2.0,
		Unit:                 "mEq/L",
		InterpretationNotes:  "Rapid sodium changes (>8 mEq/L/24h) can cause neurological damage. Gradual correction preferred.",
	},
	"17861-6": { // Calcium
		Code:                 "17861-6",
		Name:                 "Calcium",
		Direction:            DirectionMidOptimal,
		ClinicalSignificance: 0.5,
		VolatilityThreshold:  0.1,
		MinClinicalChange:    0.3,
		Unit:                 "mg/dL",
		InterpretationNotes:  "Consider albumin-corrected calcium. Both extremes affect cardiac and neuromuscular function.",
	},

	// HEMATOLOGY - Generally higher better
	"718-7": { // Hemoglobin
		Code:                 "718-7",
		Name:                 "Hemoglobin",
		Direction:            DirectionHigherBetter,
		ClinicalSignificance: 1.0,
		VolatilityThreshold:  0.1,
		MinClinicalChange:    0.5,
		Unit:                 "g/dL",
		InterpretationNotes:  "Rising Hgb after anemia treatment indicates response. Rapid drop suggests bleeding or hemolysis.",
	},
	"789-8": { // RBC
		Code:                 "789-8",
		Name:                 "RBC",
		Direction:            DirectionHigherBetter,
		ClinicalSignificance: 0.5,
		VolatilityThreshold:  0.1,
		MinClinicalChange:    0.3,
		Unit:                 "million/uL",
		InterpretationNotes:  "Trends with hemoglobin. Consider polycythemia if persistently elevated.",
	},
	"4544-3": { // Hematocrit
		Code:                 "4544-3",
		Name:                 "Hematocrit",
		Direction:            DirectionHigherBetter,
		ClinicalSignificance: 3.0,
		VolatilityThreshold:  0.1,
		MinClinicalChange:    2.0,
		Unit:                 "%",
		InterpretationNotes:  "Approximately 3x hemoglobin value. Dehydration can falsely elevate.",
	},
	"777-3": { // Platelets
		Code:                 "777-3",
		Name:                 "Platelets",
		Direction:            DirectionMidOptimal,
		ClinicalSignificance: 50.0,
		VolatilityThreshold:  0.25,
		MinClinicalChange:    30.0,
		Unit:                 "K/uL",
		InterpretationNotes:  "Both extremes problematic. Rapid drops may indicate DIC, HIT, or bone marrow suppression.",
	},
	"6690-2": { // WBC
		Code:                 "6690-2",
		Name:                 "WBC",
		Direction:            DirectionContextual, // Depends on whether fighting infection or monitoring chemo
		ClinicalSignificance: 2.0,
		VolatilityThreshold:  0.3,
		MinClinicalChange:    1.5,
		Unit:                 "K/uL",
		InterpretationNotes:  "Rising may indicate infection response. Falling during chemotherapy may indicate neutropenia risk.",
	},

	// LIVER FUNCTION - Lower better for enzymes
	"1920-8": { // AST
		Code:                 "1920-8",
		Name:                 "AST",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 20.0,
		VolatilityThreshold:  0.4,
		MinClinicalChange:    10.0,
		Unit:                 "U/L",
		InterpretationNotes:  "Elevated in liver disease, MI, muscle damage. Ratio with ALT helps identify cause.",
	},
	"1742-6": { // ALT
		Code:                 "1742-6",
		Name:                 "ALT",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 20.0,
		VolatilityThreshold:  0.4,
		MinClinicalChange:    10.0,
		Unit:                 "U/L",
		InterpretationNotes:  "More specific for liver than AST. Dropping trend suggests resolving hepatic injury.",
	},
	"6768-6": { // ALP
		Code:                 "6768-6",
		Name:                 "ALP",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 30.0,
		VolatilityThreshold:  0.3,
		MinClinicalChange:    20.0,
		Unit:                 "U/L",
		InterpretationNotes:  "Elevated in cholestasis, bone disease. Consider GGT to differentiate.",
	},
	"1975-2": { // Bilirubin (Total)
		Code:                 "1975-2",
		Name:                 "Bilirubin, Total",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 0.5,
		VolatilityThreshold:  0.3,
		MinClinicalChange:    0.3,
		Unit:                 "mg/dL",
		InterpretationNotes:  "Rising bilirubin suggests worsening liver function or hemolysis.",
	},
	"1751-7": { // Albumin
		Code:                 "1751-7",
		Name:                 "Albumin",
		Direction:            DirectionHigherBetter,
		ClinicalSignificance: 0.3,
		VolatilityThreshold:  0.15,
		MinClinicalChange:    0.2,
		Unit:                 "g/dL",
		InterpretationNotes:  "Reflects synthetic liver function and nutritional status. Low levels affect drug protein binding.",
	},

	// CARDIAC MARKERS - Lower better (except in recovery)
	"6598-7": { // Troponin I
		Code:                 "6598-7",
		Name:                 "Troponin I",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 0.02,
		VolatilityThreshold:  0.5,
		MinClinicalChange:    0.01,
		Unit:                 "ng/mL",
		InterpretationNotes:  "Rising troponin pattern suggests acute MI. Peaks at 12-24h, normalizes in 7-10 days.",
	},
	"30934-4": { // BNP
		Code:                 "30934-4",
		Name:                 "BNP",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 100.0,
		VolatilityThreshold:  0.4,
		MinClinicalChange:    50.0,
		Unit:                 "pg/mL",
		InterpretationNotes:  "Elevated in heart failure. Dropping trend indicates response to treatment.",
	},

	// COAGULATION
	"34714-6": { // INR
		Code:                 "34714-6",
		Name:                 "INR",
		Direction:            DirectionContextual, // Target depends on indication
		ClinicalSignificance: 0.3,
		VolatilityThreshold:  0.2,
		MinClinicalChange:    0.2,
		Unit:                 "ratio",
		InterpretationNotes:  "Target varies by indication (2-3 for most, 2.5-3.5 for mechanical valves). High volatility increases bleeding/clotting risk.",
	},
	"5902-2": { // PT
		Code:                 "5902-2",
		Name:                 "PT",
		Direction:            DirectionLowerBetter, // Unless on anticoagulation
		ClinicalSignificance: 2.0,
		VolatilityThreshold:  0.15,
		MinClinicalChange:    1.0,
		Unit:                 "seconds",
		InterpretationNotes:  "Prolonged in liver disease, vitamin K deficiency, or warfarin therapy.",
	},

	// INFLAMMATORY MARKERS - Lower better
	"1988-5": { // CRP
		Code:                 "1988-5",
		Name:                 "CRP",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 10.0,
		VolatilityThreshold:  0.5,
		MinClinicalChange:    5.0,
		Unit:                 "mg/L",
		InterpretationNotes:  "Drops rapidly with infection resolution. Persistently elevated suggests ongoing inflammation.",
	},
	"75241-0": { // Procalcitonin
		Code:                 "75241-0",
		Name:                 "Procalcitonin",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 0.25,
		VolatilityThreshold:  0.5,
		MinClinicalChange:    0.1,
		Unit:                 "ng/mL",
		InterpretationNotes:  "More specific for bacterial infection than CRP. Used to guide antibiotic duration.",
	},
	"2524-7": { // Lactate
		Code:                 "2524-7",
		Name:                 "Lactate",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 1.0,
		VolatilityThreshold:  0.4,
		MinClinicalChange:    0.5,
		Unit:                 "mmol/L",
		InterpretationNotes:  "Elevated in sepsis, shock, ischemia. Clearance >10% per hour indicates adequate resuscitation.",
	},

	// LIPIDS - Generally lower better
	"2093-3": { // Cholesterol (Total)
		Code:                 "2093-3",
		Name:                 "Cholesterol, Total",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 20.0,
		VolatilityThreshold:  0.1,
		MinClinicalChange:    10.0,
		Unit:                 "mg/dL",
		InterpretationNotes:  "Target <200 mg/dL. Statins typically reduce 30-50%.",
	},
	"13457-7": { // LDL
		Code:                 "13457-7",
		Name:                 "LDL Cholesterol",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 15.0,
		VolatilityThreshold:  0.15,
		MinClinicalChange:    10.0,
		Unit:                 "mg/dL",
		InterpretationNotes:  "Primary target for CVD prevention. Goals vary by risk (70-100 mg/dL for high risk).",
	},
	"2085-9": { // HDL
		Code:                 "2085-9",
		Name:                 "HDL Cholesterol",
		Direction:            DirectionHigherBetter,
		ClinicalSignificance: 5.0,
		VolatilityThreshold:  0.15,
		MinClinicalChange:    3.0,
		Unit:                 "mg/dL",
		InterpretationNotes:  "Protective factor. Target >40 mg/dL (men), >50 mg/dL (women).",
	},
	"2571-8": { // Triglycerides
		Code:                 "2571-8",
		Name:                 "Triglycerides",
		Direction:            DirectionLowerBetter,
		ClinicalSignificance: 50.0,
		VolatilityThreshold:  0.3,
		MinClinicalChange:    25.0,
		Unit:                 "mg/dL",
		InterpretationNotes:  "Target <150 mg/dL. Very high levels (>500) risk pancreatitis.",
	},

	// THYROID
	"3016-3": { // TSH
		Code:                 "3016-3",
		Name:                 "TSH",
		Direction:            DirectionMidOptimal,
		ClinicalSignificance: 1.0,
		VolatilityThreshold:  0.3,
		MinClinicalChange:    0.5,
		Unit:                 "mIU/L",
		InterpretationNotes:  "Target 0.4-4.0 (or 0.5-2.5 for thyroid replacement). Inverse relationship with T4.",
	},
	"3024-7": { // T4 Free
		Code:                 "3024-7",
		Name:                 "T4, Free",
		Direction:            DirectionMidOptimal,
		ClinicalSignificance: 0.3,
		VolatilityThreshold:  0.15,
		MinClinicalChange:    0.2,
		Unit:                 "ng/dL",
		InterpretationNotes:  "Assess with TSH. Low T4 + high TSH = hypothyroid; High T4 + low TSH = hyperthyroid.",
	},
}

// GetLabContext retrieves clinical context for a lab test
func GetLabContext(code string) (*LabTrendContext, bool) {
	ctx, found := LabContextDatabase[code]
	return &ctx, found
}

// =============================================================================
// CONTEXT-AWARE TRAJECTORY INTERPRETATION
// =============================================================================

// InterpretTrajectory determines if a trajectory is clinically improving or worsening
// based on the lab-specific clinical context
func InterpretTrajectory(code string, trajectory types.Trajectory, slope float64) TrajectoryInterpretation {
	ctx, found := GetLabContext(code)

	// Default interpretation if no context available
	if !found {
		return defaultTrajectoryInterpretation(trajectory, slope)
	}

	return interpretWithContext(trajectory, slope, ctx)
}

// TrajectoryInterpretation provides clinical meaning to a trajectory
type TrajectoryInterpretation struct {
	ClinicalMeaning   string `json:"clinical_meaning"`   // IMPROVING, WORSENING, STABLE, CONCERNING
	Urgency           string `json:"urgency"`            // ROUTINE, MONITOR, ATTENTION, URGENT
	Explanation       string `json:"explanation"`
	RecommendedAction string `json:"recommended_action,omitempty"`
}

func interpretWithContext(trajectory types.Trajectory, slope float64, ctx *LabTrendContext) TrajectoryInterpretation {
	switch trajectory {
	case types.TrajectoryVolatile:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "CONCERNING",
			Urgency:           "ATTENTION",
			Explanation:       ctx.Name + " shows high variability, suggesting unstable clinical status or specimen issues.",
			RecommendedAction: "Review specimen collection, verify results, assess for rapid clinical changes.",
		}

	case types.TrajectoryStable:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "STABLE",
			Urgency:           "ROUTINE",
			Explanation:       ctx.Name + " is stable with no significant trend.",
			RecommendedAction: "Continue current monitoring frequency.",
		}

	case types.TrajectoryUnknown:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "INSUFFICIENT_DATA",
			Urgency:           "ROUTINE",
			Explanation:       "Not enough data points to determine trend for " + ctx.Name + ".",
			RecommendedAction: "Collect additional data points before trending assessment.",
		}
	}

	// Handle increasing/decreasing based on lab context
	isIncreasing := slope > 0

	switch ctx.Direction {
	case DirectionLowerBetter:
		if isIncreasing {
			return TrajectoryInterpretation{
				ClinicalMeaning:   "WORSENING",
				Urgency:           "ATTENTION",
				Explanation:       ctx.Name + " is rising, which may indicate worsening condition. " + ctx.InterpretationNotes,
				RecommendedAction: "Evaluate underlying cause and consider intervention.",
			}
		}
		return TrajectoryInterpretation{
			ClinicalMeaning:   "IMPROVING",
			Urgency:           "ROUTINE",
			Explanation:       ctx.Name + " is decreasing toward normal, suggesting improvement.",
			RecommendedAction: "Continue current treatment and monitoring.",
		}

	case DirectionHigherBetter:
		if isIncreasing {
			return TrajectoryInterpretation{
				ClinicalMeaning:   "IMPROVING",
				Urgency:           "ROUTINE",
				Explanation:       ctx.Name + " is rising, which typically indicates improvement. " + ctx.InterpretationNotes,
				RecommendedAction: "Continue current treatment approach.",
			}
		}
		return TrajectoryInterpretation{
			ClinicalMeaning:   "WORSENING",
			Urgency:           "ATTENTION",
			Explanation:       ctx.Name + " is falling, which may indicate deterioration. " + ctx.InterpretationNotes,
			RecommendedAction: "Evaluate for underlying cause (blood loss, production issues, etc.).",
		}

	case DirectionMidOptimal:
		// For mid-optimal labs, moving away from normal is worsening
		return TrajectoryInterpretation{
			ClinicalMeaning:   "MONITOR",
			Urgency:           "MONITOR",
			Explanation:       ctx.Name + " is trending - both high and low extremes are concerning. " + ctx.InterpretationNotes,
			RecommendedAction: "Assess if trending toward or away from therapeutic range.",
		}

	case DirectionContextual:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "CONTEXT_DEPENDENT",
			Urgency:           "MONITOR",
			Explanation:       ctx.Name + " interpretation depends on clinical context (e.g., on anticoagulation, fighting infection). " + ctx.InterpretationNotes,
			RecommendedAction: "Correlate with clinical scenario and treatment goals.",
		}
	}

	return defaultTrajectoryInterpretation(trajectory, slope)
}

func defaultTrajectoryInterpretation(trajectory types.Trajectory, slope float64) TrajectoryInterpretation {
	switch trajectory {
	case types.TrajectoryWorsening:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "WORSENING",
			Urgency:           "ATTENTION",
			Explanation:       "Values are trending in a concerning direction.",
			RecommendedAction: "Review and assess clinical significance.",
		}
	case types.TrajectoryImproving:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "IMPROVING",
			Urgency:           "ROUTINE",
			Explanation:       "Values are trending in a favorable direction.",
			RecommendedAction: "Continue current approach.",
		}
	case types.TrajectoryStable:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "STABLE",
			Urgency:           "ROUTINE",
			Explanation:       "Values are stable.",
			RecommendedAction: "Continue routine monitoring.",
		}
	case types.TrajectoryVolatile:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "CONCERNING",
			Urgency:           "ATTENTION",
			Explanation:       "High variability observed.",
			RecommendedAction: "Investigate cause of volatility.",
		}
	default:
		return TrajectoryInterpretation{
			ClinicalMeaning:   "UNKNOWN",
			Urgency:           "ROUTINE",
			Explanation:       "Insufficient data for trend analysis.",
			RecommendedAction: "Collect more data points.",
		}
	}
}

// IsClinicallySignificant determines if a change is clinically meaningful
func IsClinicallySignificant(code string, change float64) bool {
	ctx, found := GetLabContext(code)
	if !found {
		// Default: any change > 5% is considered significant
		return true
	}
	return change >= ctx.MinClinicalChange
}

// GetVolatilityThreshold returns the concerning CV threshold for a lab
func GetVolatilityThreshold(code string) float64 {
	ctx, found := GetLabContext(code)
	if !found {
		return 0.3 // Default 30%
	}
	return ctx.VolatilityThreshold
}
