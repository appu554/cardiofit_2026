package services

import (
	"math"
	"strings"

	"kb-26-metabolic-digital-twin/internal/models"
)

// DeviationContext carries patient-specific context for deviation scoring.
type DeviationContext struct {
	CKMStage              string
	ActiveConfounderName  string  // empty if no confounder active
	HoursSinceLastReading float64
	IsPostDischarge       bool
	IsRuralConnectivity   bool
}

// AcuteDetectionConfig holds threshold configs loaded from YAML.
type AcuteDetectionConfig struct {
	// eGFR thresholds (percentage deviation)
	EGFRCriticalPct  float64
	EGFRHighPct      float64
	EGFRModeratePct  float64
	EGFRAbsoluteCrit float64 // absolute eGFR value below which is critical

	// SBP thresholds (mmHg deviation from baseline)
	SBPCriticalMmHg     float64
	SBPHighMmHg         float64
	SBPModerateMmHg     float64
	SBPAbsoluteCritHigh float64
	SBPAbsoluteCritLow  float64

	// Weight thresholds (kg gain in 72h)
	WeightCriticalKg       float64
	WeightHighKg           float64
	WeightModerateKg       float64
	WeightNonHFMaxSeverity string

	// Weight thresholds in HF context (CKM stage 4+) — more sensitive
	WeightHFCriticalKg float64
	WeightHFHighKg     float64
	WeightHFModerateKg float64

	// Potassium thresholds (absolute)
	PotassiumCritical float64
	PotassiumHigh     float64

	// Glucose FBG thresholds
	GlucoseHighAbsolute float64
	GlucoseModeratePct  float64

	// Gap amplification
	GapThresholdHours      float64
	GapRuralThresholdHours float64
	GapAmplifyLevels       int

	// Confounder dampening
	ConfounderDampenLevels int
	ConfounderMinFloor     string

	// Low confidence threshold multiplier (applied to HIGH and CRITICAL only)
	LowConfidenceMultiplier float64
}

// DefaultAcuteDetectionConfig returns production defaults.
func DefaultAcuteDetectionConfig() *AcuteDetectionConfig {
	return &AcuteDetectionConfig{
		EGFRCriticalPct:  30,
		EGFRHighPct:      25,
		EGFRModeratePct:  20,
		EGFRAbsoluteCrit: 20,

		SBPCriticalMmHg:     40,
		SBPHighMmHg:         30,
		SBPModerateMmHg:     25,
		SBPAbsoluteCritHigh: 180,
		SBPAbsoluteCritLow:  90,

		WeightCriticalKg:       3.0,
		WeightHighKg:           2.0,
		WeightModerateKg:       1.5,
		WeightNonHFMaxSeverity: "MODERATE",

		WeightHFCriticalKg: 2.0,
		WeightHFHighKg:     1.5,
		WeightHFModerateKg: 1.0,

		PotassiumCritical: 6.0,
		PotassiumHigh:     5.5,

		GlucoseHighAbsolute: 300,
		GlucoseModeratePct:  40,

		GapThresholdHours:      48,
		GapRuralThresholdHours: 72,
		GapAmplifyLevels:       1,

		ConfounderDampenLevels: 1,
		ConfounderMinFloor:     "MODERATE",

		LowConfidenceMultiplier: 1.5,
	}
}

// severityRank maps severity strings to numeric ranks for escalation/de-escalation.
var severityRank = map[string]int{
	"":         0,
	"MODERATE": 1,
	"HIGH":     2,
	"CRITICAL": 3,
}

// rankToSeverity is the inverse of severityRank.
var rankToSeverity = map[int]string{
	0: "",
	1: "MODERATE",
	2: "HIGH",
	3: "CRITICAL",
}

// escalateSeverity raises severity by the given number of levels, capping at CRITICAL.
func escalateSeverity(current string, levels int) string {
	rank := severityRank[current] + levels
	if rank > 3 {
		rank = 3
	}
	return rankToSeverity[rank]
}

// deescalateSeverity lowers severity by the given number of levels, never below floor.
func deescalateSeverity(current string, levels int, floor string) string {
	rank := severityRank[current] - levels
	floorRank := severityRank[floor]
	if rank < floorRank {
		rank = floorRank
	}
	if rank < 0 {
		rank = 0
	}
	return rankToSeverity[rank]
}

// ComputeDeviation evaluates a single vital sign reading against the patient baseline
// and returns a DeviationResult with clinical significance, gap amplification, and
// confounder dampening applied.
func ComputeDeviation(
	currentValue float64,
	baseline models.PatientBaselineSnapshot,
	vitalType string,
	config *AcuteDetectionConfig,
	context DeviationContext,
) models.DeviationResult {
	result := models.DeviationResult{
		VitalSignType:  vitalType,
		CurrentValue:   currentValue,
		BaselineMedian: baseline.BaselineMedian,
		BaselineMAD:    baseline.BaselineMAD,
	}

	if baseline.BaselineMedian == 0 {
		return result
	}

	// Step 1-2: Compute deviations.
	deviation := currentValue - baseline.BaselineMedian
	deviationAbs := math.Abs(deviation)
	deviationPct := deviationAbs / baseline.BaselineMedian * 100.0

	result.DeviationAbsolute = math.Round(deviationAbs*100) / 100
	result.DeviationPercent = math.Round(deviationPct*100) / 100

	// Step 3: Determine direction.
	if deviation >= 0 {
		result.Direction = "ABOVE_BASELINE"
	} else {
		result.Direction = "BELOW_BASELINE"
	}

	// Step 4: Directional rules — check if this direction triggers alerts for this vital type.
	if !isAlertDirection(vitalType, result.Direction) {
		// Benign direction (e.g., eGFR rise). No alert.
		result.ClinicalSignificance = ""
		return result
	}

	// Step 5-6: Map deviation to severity using config thresholds.
	// Low confidence widens HIGH and CRITICAL thresholds (not MODERATE).
	lowConfidence := baseline.Confidence == "LOW"
	severity := mapToSeverity(vitalType, currentValue, deviationAbs, deviationPct, config, lowConfidence, context)

	// Step 7: Context gating — weight in non-HF patients.
	if vitalType == "WEIGHT" && !isHFContext(context.CKMStage) {
		maxRank := severityRank[config.WeightNonHFMaxSeverity]
		if severityRank[severity] > maxRank {
			severity = config.WeightNonHFMaxSeverity
		}
	}

	// Step 8: Gap amplification.
	gapAmplified := false
	gapThreshold := config.GapThresholdHours
	if context.IsRuralConnectivity {
		gapThreshold = config.GapRuralThresholdHours
	}
	if context.HoursSinceLastReading > gapThreshold && severity != "" {
		severity = escalateSeverity(severity, config.GapAmplifyLevels)
		gapAmplified = true
	}

	// Step 9: Confounder dampening.
	confounderDampened := false
	if context.ActiveConfounderName != "" && severity != "" {
		severity = deescalateSeverity(severity, config.ConfounderDampenLevels, config.ConfounderMinFloor)
		confounderDampened = true
	}

	result.ClinicalSignificance = severity
	result.GapAmplified = gapAmplified
	result.ConfounderDampened = confounderDampened

	return result
}

// isAlertDirection returns true if the given direction triggers clinical alerts for the vital type.
func isAlertDirection(vitalType, direction string) bool {
	switch vitalType {
	case "EGFR":
		return direction == "BELOW_BASELINE" // only drops trigger
	case "SBP":
		return true // both directions trigger
	case "WEIGHT":
		return direction == "ABOVE_BASELINE" // only gains trigger
	case "POTASSIUM":
		return direction == "ABOVE_BASELINE" // only rises trigger
	case "GLUCOSE":
		return direction == "ABOVE_BASELINE" // only rises trigger
	default:
		return true
	}
}

// isHFContext returns true if the CKM stage indicates heart failure context (stage 4+).
func isHFContext(ckmStage string) bool {
	return strings.HasPrefix(ckmStage, "4")
}

// mapToSeverity determines the raw severity before gap/confounder adjustments.
func mapToSeverity(
	vitalType string,
	currentValue float64,
	deviationAbs float64,
	deviationPct float64,
	config *AcuteDetectionConfig,
	lowConfidence bool,
	context DeviationContext,
) string {
	switch vitalType {
	case "EGFR":
		return mapEGFRSeverity(currentValue, deviationPct, config, lowConfidence)
	case "SBP":
		return mapSBPSeverity(currentValue, deviationAbs, config, lowConfidence)
	case "WEIGHT":
		return mapWeightSeverity(deviationAbs, config, lowConfidence, context)
	case "POTASSIUM":
		return mapPotassiumSeverity(currentValue, config)
	case "GLUCOSE":
		return mapGlucoseSeverity(currentValue, deviationPct, config, lowConfidence)
	default:
		return ""
	}
}

// mapEGFRSeverity uses percentage-based thresholds for eGFR drops.
func mapEGFRSeverity(currentValue, deviationPct float64, config *AcuteDetectionConfig, lowConfidence bool) string {
	// Absolute critical: eGFR below absolute floor.
	if currentValue <= config.EGFRAbsoluteCrit {
		return "CRITICAL"
	}

	critPct := config.EGFRCriticalPct
	highPct := config.EGFRHighPct
	modPct := config.EGFRModeratePct

	// Low confidence widens HIGH and CRITICAL thresholds only.
	if lowConfidence {
		critPct *= config.LowConfidenceMultiplier
		highPct *= config.LowConfidenceMultiplier
	}

	switch {
	case deviationPct >= critPct:
		return "CRITICAL"
	case deviationPct >= highPct:
		return "HIGH"
	case deviationPct >= modPct:
		return "MODERATE"
	default:
		return ""
	}
}

// mapSBPSeverity uses absolute mmHg thresholds for SBP deviations.
func mapSBPSeverity(currentValue, deviationAbs float64, config *AcuteDetectionConfig, lowConfidence bool) string {
	// Absolute critical: SBP above or below absolute thresholds.
	if currentValue >= config.SBPAbsoluteCritHigh || currentValue <= config.SBPAbsoluteCritLow {
		return "CRITICAL"
	}

	critMmHg := config.SBPCriticalMmHg
	highMmHg := config.SBPHighMmHg
	modMmHg := config.SBPModerateMmHg

	if lowConfidence {
		critMmHg *= config.LowConfidenceMultiplier
		highMmHg *= config.LowConfidenceMultiplier
	}

	switch {
	case deviationAbs >= critMmHg:
		return "CRITICAL"
	case deviationAbs >= highMmHg:
		return "HIGH"
	case deviationAbs >= modMmHg:
		return "MODERATE"
	default:
		return ""
	}
}

// mapWeightSeverity uses absolute kg thresholds for weight gain.
// In HF context (CKM 4+), uses tighter thresholds.
func mapWeightSeverity(deviationAbs float64, config *AcuteDetectionConfig, lowConfidence bool, context DeviationContext) string {
	critKg := config.WeightCriticalKg
	highKg := config.WeightHighKg
	modKg := config.WeightModerateKg

	if isHFContext(context.CKMStage) {
		critKg = config.WeightHFCriticalKg
		highKg = config.WeightHFHighKg
		modKg = config.WeightHFModerateKg
	}

	if lowConfidence {
		critKg *= config.LowConfidenceMultiplier
		highKg *= config.LowConfidenceMultiplier
	}

	switch {
	case deviationAbs >= critKg:
		return "CRITICAL"
	case deviationAbs >= highKg:
		return "HIGH"
	case deviationAbs >= modKg:
		return "MODERATE"
	default:
		return ""
	}
}

// mapPotassiumSeverity uses absolute value thresholds.
func mapPotassiumSeverity(currentValue float64, config *AcuteDetectionConfig) string {
	switch {
	case currentValue >= config.PotassiumCritical:
		return "CRITICAL"
	case currentValue >= config.PotassiumHigh:
		return "HIGH"
	default:
		return ""
	}
}

// mapGlucoseSeverity uses both absolute and percentage thresholds.
func mapGlucoseSeverity(currentValue, deviationPct float64, config *AcuteDetectionConfig, lowConfidence bool) string {
	if currentValue >= config.GlucoseHighAbsolute {
		return "HIGH"
	}

	modPct := config.GlucoseModeratePct
	if lowConfidence {
		modPct *= config.LowConfidenceMultiplier
	}

	if deviationPct >= modPct {
		return "MODERATE"
	}
	return ""
}
