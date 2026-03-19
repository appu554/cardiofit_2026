package services

import (
	"fmt"
	"strings"
)

// TrajectoryColor represents protocol progress classification.
type TrajectoryColor string

const (
	TrajectoryGreen  TrajectoryColor = "GREEN"
	TrajectoryYellow TrajectoryColor = "YELLOW"
	TrajectoryRed    TrajectoryColor = "RED"
)

// TrajectoryInput holds the data needed to compute trajectory for one protocol track.
type TrajectoryInput struct {
	ProtocolID     string
	CurrentPhase   string
	DaysInPhase    int
	DaysSinceStart int

	// Adherence metrics (0-100 scale)
	ProteinAdherence  float64 // PRP-specific
	ExerciseAdherence float64 // shared
	MealQualityScore  float64 // PRP / VFRP shared

	// Lab trends (deltas since last measurement)
	EGFRDelta    float64 // negative = declining (bad for PRP)
	FBGDelta     float64 // positive = worsening
	HbA1cDelta   float64 // positive = worsening
	TrigDelta    float64 // positive = worsening (bad for VFRP)
	WaistDeltaCm float64 // negative = improving (good for VFRP)
	WeightDeltaKg float64 // for VFRP safety

	// Safety context
	BMI float64

	// MRI forcing (optional — zero values = no MRI data available)
	// Spec §7: MRI >75 = force RED. MRI worsening >10 points in 14 days = force YELLOW.
	MRIScore    float64 // current MRI score 0-100 (0 = not available)
	MRIDelta14d float64 // MRI score change over last 14 days (positive = worsening)
}

// ProtocolTrajectoryResult is the classification output for one protocol.
type ProtocolTrajectoryResult struct {
	ProtocolID    string          `json:"protocol_id"`
	Color         TrajectoryColor `json:"color"`
	Score         float64         `json:"score"`          // 0-100, higher = better
	Reasons       []string        `json:"reasons"`        // human-readable factors
	EscalationDue bool            `json:"escalation_due"` // true if RED at day 63+
}

// CompositeTrajectory is the patient-level summary across all active protocols.
type CompositeTrajectory struct {
	PatientColor  TrajectoryColor           `json:"patient_color"`
	Protocols     []ProtocolTrajectoryResult `json:"protocols"`
	AnyEscalation bool             `json:"any_escalation"`
}

// TrajectoryEngine computes trajectory classifications.
// It is a pure computation engine with no database or network dependencies.
type TrajectoryEngine struct{}

// NewTrajectoryEngine creates a new engine.
func NewTrajectoryEngine() *TrajectoryEngine {
	return &TrajectoryEngine{}
}

// ---------------------------------------------------------------------------
// Thresholds
// ---------------------------------------------------------------------------

const (
	greenThreshold  = 70.0
	yellowThreshold = 40.0

	// Adherence / lab weighting by phase maturity.
	earlyAdherenceWeight = 0.70 // phases 1-2
	earlyLabWeight       = 0.30
	lateAdherenceWeight  = 0.50 // phase 3+
	lateLabWeight        = 0.50

	// Safety hard-stops
	egfrCriticalDelta      = -5.0 // eGFR decline of 5+ ml/min → RED
	vfrpWeightLossLimit    = 3.0  // kg
	vfrpBMISafetyThreshold = 25.0
)

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// Classify computes trajectory for a single protocol track.
func (e *TrajectoryEngine) Classify(input TrajectoryInput) ProtocolTrajectoryResult {
	var reasons []string

	// Step 1: adherence score
	adherence := computeAdherenceScore(input)
	reasons = append(reasons, fmt.Sprintf("adherence_score=%.0f", adherence))

	// Step 2: lab trend score
	labScore, labReasons, criticalLab := computeLabTrendScore(input)
	reasons = append(reasons, labReasons...)

	// Step 3: weighted combination (phase-aware)
	adhW, labW := phaseWeights(input.CurrentPhase)
	overall := adhW*adherence + labW*labScore
	reasons = append(reasons, fmt.Sprintf("overall_score=%.1f (adh_w=%.2f, lab_w=%.2f)", overall, adhW, labW))

	// Step 4: map to color
	color := scoreToColor(overall)

	// Step 5: safety overrides
	if criticalLab {
		color = TrajectoryRed
		reasons = append(reasons, "safety_override: critical lab factor detected")
	}
	if input.EGFRDelta <= egfrCriticalDelta {
		color = TrajectoryRed
		reasons = append(reasons, fmt.Sprintf("safety_override: eGFR_delta=%.1f <= %.1f", input.EGFRDelta, egfrCriticalDelta))
	}
	if isVFRP(input.ProtocolID) && input.WeightDeltaKg > vfrpWeightLossLimit && input.BMI > 0 && input.BMI < vfrpBMISafetyThreshold {
		color = TrajectoryRed
		reasons = append(reasons, fmt.Sprintf("safety_override: weight_loss=%.1fkg with BMI=%.1f < %.1f", input.WeightDeltaKg, input.BMI, vfrpBMISafetyThreshold))
	}

	// Step 6: MRI forcing (Spec §7)
	if input.MRIScore > 0 {
		baseColor := color
		color = applyMRIForcing(color, input.MRIScore, input.MRIDelta14d)
		if color == TrajectoryRed && baseColor != TrajectoryRed {
			reasons = append(reasons, fmt.Sprintf("MRI forcing: score %.0f", input.MRIScore))
		} else if color == TrajectoryYellow && baseColor != TrajectoryYellow {
			reasons = append(reasons, fmt.Sprintf("MRI forcing: worsening %.0f pts in 14d", input.MRIDelta14d))
		}
	}

	// Step 7: escalation check
	escalation := color == TrajectoryRed && input.DaysSinceStart >= 63

	return ProtocolTrajectoryResult{
		ProtocolID:    input.ProtocolID,
		Color:         color,
		Score:         overall,
		Reasons:       reasons,
		EscalationDue: escalation,
	}
}

// ClassifyAll computes trajectories for multiple protocols and produces a composite.
func (e *TrajectoryEngine) ClassifyAll(inputs []TrajectoryInput) CompositeTrajectory {
	if len(inputs) == 0 {
		return CompositeTrajectory{PatientColor: TrajectoryGreen}
	}

	results := make([]ProtocolTrajectoryResult, 0, len(inputs))
	worst := TrajectoryGreen
	anyEscalation := false

	for _, input := range inputs {
		r := e.Classify(input)
		results = append(results, r)
		if colorSeverity(r.Color) > colorSeverity(worst) {
			worst = r.Color
		}
		if r.EscalationDue {
			anyEscalation = true
		}
	}

	return CompositeTrajectory{
		PatientColor:  worst,
		Protocols:     results,
		AnyEscalation: anyEscalation,
	}
}

// ---------------------------------------------------------------------------
// Adherence scoring
// ---------------------------------------------------------------------------

func computeAdherenceScore(input TrajectoryInput) float64 {
	switch {
	case isPRP(input.ProtocolID):
		return averageNonZero(input.ProteinAdherence, input.ExerciseAdherence, input.MealQualityScore)
	case isVFRP(input.ProtocolID):
		return averageNonZero(input.ExerciseAdherence, input.MealQualityScore)
	default:
		if input.ExerciseAdherence > 0 {
			return input.ExerciseAdherence
		}
		return 0
	}
}

func averageNonZero(vals ...float64) float64 {
	var sum float64
	var count int
	for _, v := range vals {
		if v > 0 {
			sum += v
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// ---------------------------------------------------------------------------
// Lab trend scoring
// ---------------------------------------------------------------------------

// computeLabTrendScore returns (score 0-100, reason strings, critical flag).
func computeLabTrendScore(input TrajectoryInput) (float64, []string, bool) {
	var factors []float64
	var reasons []string
	critical := false

	if isPRP(input.ProtocolID) {
		// eGFR: declining is bad
		egfrScore := labScore_eGFR(input.EGFRDelta)
		factors = append(factors, egfrScore)
		reasons = append(reasons, fmt.Sprintf("lab_egfr=%.0f (delta=%.1f)", egfrScore, input.EGFRDelta))
		if egfrScore == 0 {
			critical = true
		}

		// FBG: rising is bad
		fbgScore := labScore_FBG(input.FBGDelta)
		factors = append(factors, fbgScore)
		reasons = append(reasons, fmt.Sprintf("lab_fbg=%.0f (delta=%.1f)", fbgScore, input.FBGDelta))
		if fbgScore == 0 {
			critical = true
		}

		// HbA1c: rising is bad
		a1cScore := labScore_HbA1c(input.HbA1cDelta)
		factors = append(factors, a1cScore)
		reasons = append(reasons, fmt.Sprintf("lab_hba1c=%.0f (delta=%.2f)", a1cScore, input.HbA1cDelta))
		if a1cScore == 0 {
			critical = true
		}
	} else if isVFRP(input.ProtocolID) {
		// Triglycerides: rising is bad
		trigScore := labScore_Trig(input.TrigDelta)
		factors = append(factors, trigScore)
		reasons = append(reasons, fmt.Sprintf("lab_trig=%.0f (delta=%.1f)", trigScore, input.TrigDelta))
		if trigScore == 0 {
			critical = true
		}

		// Waist: not decreasing is bad
		waistScore := labScore_Waist(input.WaistDeltaCm)
		factors = append(factors, waistScore)
		reasons = append(reasons, fmt.Sprintf("lab_waist=%.0f (delta=%.1f)", waistScore, input.WaistDeltaCm))
		if waistScore == 0 {
			critical = true
		}
	}

	if len(factors) == 0 {
		return 50, []string{"lab_trend=50 (no protocol-specific labs)"}, false
	}

	var sum float64
	for _, f := range factors {
		sum += f
	}
	avg := sum / float64(len(factors))
	return avg, reasons, critical
}

// Individual lab factor normalizers (0-100 scale).

func labScore_eGFR(delta float64) float64 {
	switch {
	case delta >= 0:
		return 100
	case delta > -5:
		return 50
	default:
		return 0
	}
}

func labScore_FBG(delta float64) float64 {
	switch {
	case delta <= 0:
		return 100
	case delta < 10:
		return 50
	default:
		return 0
	}
}

func labScore_HbA1c(delta float64) float64 {
	switch {
	case delta <= 0:
		return 100
	case delta < 0.5:
		return 50
	default:
		return 0
	}
}

func labScore_Trig(delta float64) float64 {
	switch {
	case delta <= 0:
		return 100
	case delta < 20:
		return 50
	default:
		return 0
	}
}

func labScore_Waist(deltaCm float64) float64 {
	switch {
	case deltaCm <= -2:
		return 100
	case deltaCm <= 0:
		return 50
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// Phase weighting
// ---------------------------------------------------------------------------

func phaseWeights(phase string) (adherenceW, labW float64) {
	// Late phases (phase 3+) weight labs more heavily.
	switch phase {
	case "OPTIMIZATION", "SUSTAINED_REDUCTION", "GRADUATED":
		return lateAdherenceWeight, lateLabWeight
	default:
		return earlyAdherenceWeight, earlyLabWeight
	}
}

// ---------------------------------------------------------------------------
// Color mapping
// ---------------------------------------------------------------------------

func scoreToColor(score float64) TrajectoryColor {
	switch {
	case score >= greenThreshold:
		return TrajectoryGreen
	case score >= yellowThreshold:
		return TrajectoryYellow
	default:
		return TrajectoryRed
	}
}

func colorSeverity(c TrajectoryColor) int {
	switch c {
	case TrajectoryRed:
		return 2
	case TrajectoryYellow:
		return 1
	default:
		return 0
	}
}

// ---------------------------------------------------------------------------
// MRI forcing
// ---------------------------------------------------------------------------

// applyMRIForcing applies MRI-based trajectory forcing rules.
// Only escalates — never downgrades an existing worse classification.
// Spec §7: MRI >75 = force RED. MRI worsening >10 points in 14 days = force YELLOW.
func applyMRIForcing(currentColor TrajectoryColor, mriScore float64, mriDelta14d float64) TrajectoryColor {
	if mriScore > 75 {
		return TrajectoryRed
	}
	if mriDelta14d > 10 && currentColor != TrajectoryRed {
		return TrajectoryYellow
	}
	return currentColor
}

// ---------------------------------------------------------------------------
// Protocol identification helpers
// ---------------------------------------------------------------------------

func isPRP(protocolID string) bool {
	return strings.Contains(strings.ToUpper(protocolID), "PRP")
}

func isVFRP(protocolID string) bool {
	return strings.Contains(strings.ToUpper(protocolID), "VFRP")
}
