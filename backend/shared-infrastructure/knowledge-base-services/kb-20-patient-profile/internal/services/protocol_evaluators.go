package services

import "fmt"

// EvaluateGLYC1Transition applies GLYC-1 phase transition rules.
// Medication protocols use metric-based escalation rather than adherence-based.
func EvaluateGLYC1Transition(eval TransitionEvaluation) TransitionDecision {
	if eval.SafetyFlags {
		return TransitionDecision{Action: "ABORT", Reason: "safety flag triggered (hypoglycaemia or adverse event)"}
	}

	switch eval.CurrentPhase {
	case "MONOTHERAPY":
		// Advance to COMBINATION if HbA1c still above target after 12 weeks on max metformin
		if eval.DaysInPhase >= 84 && eval.HbA1cAboveTarget {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "COMBINATION", Reason: "HbA1c above target after 12 weeks on max metformin — add SGLT2i/GLP1RA"}
		}
		if eval.DaysInPhase >= 112 {
			return TransitionDecision{Action: "ESCALATE", Reason: "MONOTHERAPY phase exceeded 16 weeks — clinical review required"}
		}
		return TransitionDecision{Action: "HOLD", Reason: fmt.Sprintf("MONOTHERAPY day %d — titrating metformin", eval.DaysInPhase)}

	case "COMBINATION":
		// Advance to OPTIMIZATION if HbA1c at target for 24 weeks
		if eval.DaysInPhase >= 168 && !eval.HbA1cAboveTarget {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "OPTIMIZATION", Reason: "HbA1c at target for 24 weeks — entering maintenance"}
		}
		if eval.DaysInPhase >= 252 && eval.HbA1cAboveTarget {
			return TransitionDecision{Action: "ESCALATE", Reason: "HbA1c above target after 36 weeks on combination therapy — consider insulin"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "COMBINATION therapy in progress"}

	case "OPTIMIZATION":
		// Lifelong phase — no advancement, only escalation on deterioration
		if eval.HbA1cAboveTarget && eval.DaysInPhase >= 84 {
			return TransitionDecision{Action: "ESCALATE", Reason: "HbA1c rising above target during maintenance — medication review"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "OPTIMIZATION — maintenance phase"}
	}

	return TransitionDecision{Action: "HOLD", Reason: "unknown phase"}
}

// EvaluateHTN1Transition applies HTN-1 phase transition rules.
func EvaluateHTN1Transition(eval TransitionEvaluation) TransitionDecision {
	if eval.SafetyFlags {
		return TransitionDecision{Action: "ABORT", Reason: "safety flag triggered (hypotension or electrolyte emergency)"}
	}

	switch eval.CurrentPhase {
	case "MONOTHERAPY":
		if eval.DaysInPhase >= 28 && eval.SBPAboveTarget {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "DUAL_THERAPY", Reason: "SBP above target after 4 weeks on max ACEi/ARB — add CCB"}
		}
		if eval.DaysInPhase >= 42 {
			return TransitionDecision{Action: "ESCALATE", Reason: "MONOTHERAPY phase exceeded 6 weeks"}
		}
		return TransitionDecision{Action: "HOLD", Reason: fmt.Sprintf("MONOTHERAPY day %d — titrating ACEi/ARB", eval.DaysInPhase)}

	case "DUAL_THERAPY":
		if eval.DaysInPhase >= 28 && eval.SBPAboveTarget {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "TRIPLE_THERAPY", Reason: "SBP above target on dual therapy — add thiazide"}
		}
		if eval.DaysInPhase >= 42 {
			return TransitionDecision{Action: "ESCALATE", Reason: "DUAL_THERAPY phase exceeded 6 weeks"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "DUAL_THERAPY in progress"}

	case "TRIPLE_THERAPY":
		if eval.DaysInPhase >= 28 && eval.SBPAboveTarget {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "RESISTANT_HTN", Reason: "SBP above target on triple therapy — resistant hypertension confirmed, add spironolactone"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "TRIPLE_THERAPY in progress"}

	case "RESISTANT_HTN":
		// Terminal phase — only escalation
		if eval.SBPAboveTarget && eval.DaysInPhase >= 28 {
			return TransitionDecision{Action: "ESCALATE", Reason: "SBP above target on 4 agents — specialist referral required"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "RESISTANT_HTN — monitoring on spironolactone"}
	}

	return TransitionDecision{Action: "HOLD", Reason: "unknown phase"}
}

// EvaluateRENAL1Transition applies RENAL-1 phase transition rules.
func EvaluateRENAL1Transition(eval TransitionEvaluation) TransitionDecision {
	if eval.SafetyFlags {
		return TransitionDecision{Action: "ABORT", Reason: "safety flag triggered (AKI or severe hyperkalaemia)"}
	}

	// Safety-first: check eGFR decline in any phase
	if eval.EGFRDelta > 5 {
		return TransitionDecision{Action: "ESCALATE", Reason: "eGFR declined >5 mL/min — hold current step, nephrology review"}
	}

	switch eval.CurrentPhase {
	case "RAAS_OPTIMISATION":
		if eval.DaysInPhase >= 28 && eval.ACRNotImproving {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "SGLT2I_ADDITION", Reason: "ACR not improving after 4 weeks on max ACEi/ARB — add SGLT2i"}
		}
		if eval.DaysInPhase >= 56 {
			return TransitionDecision{Action: "ESCALATE", Reason: "RAAS_OPTIMISATION exceeded 8 weeks"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "RAAS_OPTIMISATION in progress"}

	case "SGLT2I_ADDITION":
		if eval.DaysInPhase >= 28 && eval.ACRNotImproving {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "FINERENONE_ADDITION", Reason: "ACR persistently elevated on RAAS + SGLT2i — add finerenone"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "SGLT2I_ADDITION in progress"}

	case "FINERENONE_ADDITION":
		if eval.DaysInPhase >= 56 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "MONITORING", Reason: "finerenone titration complete — entering maintenance monitoring"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "FINERENONE_ADDITION — titrating finerenone"}

	case "MONITORING":
		if eval.EGFRDelta > 3 {
			return TransitionDecision{Action: "ESCALATE", Reason: "eGFR declining >3 mL/min/yr during monitoring — nephrology review"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "MONITORING — stable"}
	}

	return TransitionDecision{Action: "HOLD", Reason: "unknown phase"}
}

// EvaluateDEPRESC1Transition applies DEPRESC-1 phase transition rules.
func EvaluateDEPRESC1Transition(eval TransitionEvaluation) TransitionDecision {
	if eval.SafetyFlags {
		return TransitionDecision{Action: "ABORT", Reason: "safety flag triggered during deprescribing (hypoglycaemia)"}
	}

	switch eval.CurrentPhase {
	case "ASSESSMENT":
		if eval.DaysInPhase >= 7 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "STEPDOWN", Reason: "assessment complete — begin medication stepdown"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "ASSESSMENT in progress"}

	case "STEPDOWN":
		// Abort if HbA1c rises too high during stepdown
		if eval.HbA1cAboveTarget {
			return TransitionDecision{Action: "ESCALATE", Reason: "HbA1c rising above 8.5% during stepdown — pause deprescribing, stabilise"}
		}
		if eval.DaysInPhase >= 56 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "MONITORING", Reason: "stepdown complete — entering post-deprescribing monitoring"}
		}
		return TransitionDecision{Action: "HOLD", Reason: fmt.Sprintf("STEPDOWN day %d — reducing medications", eval.DaysInPhase)}

	case "MONITORING":
		if eval.HbA1cAboveTarget {
			return TransitionDecision{Action: "ESCALATE", Reason: "HbA1c rising during monitoring — may need to restart agents"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "MONITORING — post-deprescribing stable"}
	}

	return TransitionDecision{Action: "HOLD", Reason: "unknown phase"}
}
