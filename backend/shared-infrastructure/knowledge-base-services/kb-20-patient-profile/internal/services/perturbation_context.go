package services

import "time"

// Suppression modes
const (
	SuppressionFull     = "FULL"
	SuppressionDampened = "DAMPENED"
	SuppressionTagged   = "TAGGED"
	SuppressionNone     = "NONE"
)

// Perturbation types — 6 types evaluated in EvaluatePerturbations().
// P1: glucocorticoid, P2: SGLT2i init, P3: insulin dose change,
// P4: festival fasting, P5: acute illness, P6: metformin hold.
const (
	PerturbationGlucocorticoid  = "GLUCOCORTICOID"     // P1
	PerturbationAcuteIllness    = "ACUTE_ILLNESS"       // P5
	PerturbationFestivalFasting = "FESTIVAL_FASTING"    // P4
	PerturbationSGLT2iInit      = "SGLT2I_INITIATION"   // P2
	PerturbationMetforminHold   = "METFORMIN_HOLD"      // P6
	PerturbationInsulinChange   = "INSULIN_DOSE_CHANGE" // P3
)

// PerturbationEvalInput gathers all data needed to evaluate active perturbations.
type PerturbationEvalInput struct {
	// Glucocorticoid (P1)
	ActiveSteroid    bool
	SteroidStartDate time.Time
	SteroidStopDate  *time.Time

	// Festival fasting (P4)
	FestivalActive  bool
	FestivalEndDate *time.Time
	FastingType     string // COMPLETE_FAST | FRUIT_ONLY | ONE_MEAL | DIETARY_RESTRICTION

	// Acute illness (P5) — manual physician flag
	AcuteIllnessFlag bool

	// Medication changes (P2, P3, P6)
	SGLT2iStartedWithin14d     bool
	InsulinDoseChangedWithin5d bool
	MetforminOnHold            bool

	// Current trajectory (for context-dependent suppression)
	TrajectoryClass string
}

// PerturbationContext is the result of perturbation evaluation.
type PerturbationContext struct {
	Suppressed           bool    `json:"perturbation_suppressed"`
	Mode                 string  `json:"suppression_mode"`        // FULL | DAMPENED | TAGGED | NONE
	DominantPerturbation string  `json:"dominant_perturbation"`
	GainFactorMultiplier float64 `json:"gain_factor_multiplier"`  // 0.0 (FULL), 0.5 (DAMPENED), 1.0 (TAGGED/NONE)
	CDIMultiplier        float64 `json:"cdi_multiplier"`          // same scale
}

// EvaluatePerturbations checks all 6 perturbation types and returns the
// highest-priority active perturbation's suppression context.
// Priority: glucocorticoid > acute illness > festival > metformin hold > insulin change > SGLT2i
func EvaluatePerturbations(input PerturbationEvalInput) PerturbationContext {
	// P1: Glucocorticoid (highest priority)
	if p := evalGlucocorticoid(input); p.Mode != SuppressionNone {
		return p
	}

	// P5: Acute illness (TAGGED, not suppressed)
	if input.AcuteIllnessFlag {
		return PerturbationContext{
			Suppressed:           false, // TAGGED means NOT suppressed
			Mode:                 SuppressionTagged,
			DominantPerturbation: PerturbationAcuteIllness,
			GainFactorMultiplier: 1.0,
			CDIMultiplier:        1.0,
		}
	}

	// P4: Festival fasting
	if p := evalFestivalFasting(input); p.Mode != SuppressionNone {
		return p
	}

	// P6: Metformin hold
	if input.MetforminOnHold {
		return PerturbationContext{
			Suppressed:           true,
			Mode:                 SuppressionDampened,
			DominantPerturbation: PerturbationMetforminHold,
			GainFactorMultiplier: 0.5,
			CDIMultiplier:        0.5,
		}
	}

	// P3: Insulin dose change (V-MCU's own action)
	if input.InsulinDoseChangedWithin5d {
		return PerturbationContext{
			Suppressed:           true,
			Mode:                 SuppressionDampened,
			DominantPerturbation: PerturbationInsulinChange,
			GainFactorMultiplier: 0.5,
			CDIMultiplier:        0.5,
		}
	}

	// P2: SGLT2i initiation
	if input.SGLT2iStartedWithin14d {
		return PerturbationContext{
			Suppressed:           true,
			Mode:                 SuppressionDampened,
			DominantPerturbation: PerturbationSGLT2iInit,
			GainFactorMultiplier: 0.5,
			CDIMultiplier:        0.5,
		}
	}

	// No perturbation
	return PerturbationContext{
		Mode:                 SuppressionNone,
		GainFactorMultiplier: 1.0,
		CDIMultiplier:        1.0,
	}
}

func evalGlucocorticoid(input PerturbationEvalInput) PerturbationContext {
	none := PerturbationContext{Mode: SuppressionNone, GainFactorMultiplier: 1.0, CDIMultiplier: 1.0}
	now := time.Now().UTC()

	if input.ActiveSteroid {
		// Active steroid course → FULL suppression
		return PerturbationContext{
			Suppressed:           true,
			Mode:                 SuppressionFull,
			DominantPerturbation: PerturbationGlucocorticoid,
			GainFactorMultiplier: 0.0,
			CDIMultiplier:        0.0,
		}
	}

	if input.SteroidStopDate != nil {
		daysSinceStop := int(now.Sub(*input.SteroidStopDate).Hours() / 24)

		// Active phase extension: steroid stop + 7 days → FULL
		if daysSinceStop <= 7 {
			return PerturbationContext{
				Suppressed:           true,
				Mode:                 SuppressionFull,
				DominantPerturbation: PerturbationGlucocorticoid,
				GainFactorMultiplier: 0.0,
				CDIMultiplier:        0.0,
			}
		}

		// Resolution phase: steroid stop + 7 to +28 days → DAMPENED
		if daysSinceStop <= 28 {
			return PerturbationContext{
				Suppressed:           true,
				Mode:                 SuppressionDampened,
				DominantPerturbation: PerturbationGlucocorticoid,
				GainFactorMultiplier: 0.5,
				CDIMultiplier:        0.5,
			}
		}
	}

	return none
}

func evalFestivalFasting(input PerturbationEvalInput) PerturbationContext {
	none := PerturbationContext{Mode: SuppressionNone, GainFactorMultiplier: 1.0, CDIMultiplier: 1.0}

	if input.FestivalActive {
		// During fasting: FULL for COMPLETE_FAST/FRUIT_ONLY, DAMPENED for ONE_MEAL
		mode := SuppressionFull
		gain := 0.0
		if input.FastingType == "ONE_MEAL" {
			mode = SuppressionDampened
			gain = 0.5
		}
		if input.FastingType == "DIETARY_RESTRICTION" {
			return none // minimal glucose impact — no suppression
		}

		return PerturbationContext{
			Suppressed:           true,
			Mode:                 mode,
			DominantPerturbation: PerturbationFestivalFasting,
			GainFactorMultiplier: gain,
			CDIMultiplier:        gain,
		}
	}

	// Post-fasting rebound window
	if input.FestivalEndDate != nil {
		daysSinceEnd := int(time.Now().UTC().Sub(*input.FestivalEndDate).Hours() / 24)
		reboundDays := postFastingReboundDays(input.FastingType)

		if daysSinceEnd <= reboundDays && reboundDays > 0 {
			return PerturbationContext{
				Suppressed:           true,
				Mode:                 SuppressionDampened,
				DominantPerturbation: PerturbationFestivalFasting,
				GainFactorMultiplier: 0.5,
				CDIMultiplier:        0.5,
			}
		}
	}

	return none
}

func postFastingReboundDays(fastingType string) int {
	switch fastingType {
	case "COMPLETE_FAST":
		return 5
	case "FRUIT_ONLY":
		return 3
	case "ONE_MEAL":
		return 2
	default:
		return 0
	}
}
