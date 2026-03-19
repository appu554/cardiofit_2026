package services

import "fmt"

// EvaluateMAINTAINTransition applies M3-MAINTAIN lifecycle phase transition rules.
// Spec: Patient_Engagement_Loop_Specification Section 3 + Section 7 (M3-MAINTAIN JSON).
func EvaluateMAINTAINTransition(eval TransitionEvaluation) TransitionDecision {
	switch eval.CurrentPhase {
	case "CONSOLIDATION":
		if eval.DaysInPhase >= 90 && eval.MRIScore < 45 && eval.MRISustainedDays >= 28 &&
			eval.AdherencePct >= 0.50 && eval.ConsecutiveCheckins >= 4 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "INDEPENDENCE",
				Reason: fmt.Sprintf("MRI %.1f < 45 sustained %dd, adherence %.0f%% across %d check-ins",
					eval.MRIScore, eval.MRISustainedDays, eval.AdherencePct*100, eval.ConsecutiveCheckins)}
		}
		if eval.DaysInPhase >= 120 {
			return TransitionDecision{Action: "ESCALATE",
				Reason: "CONSOLIDATION exceeded 120 days without meeting Independence criteria — clinical review"}
		}
		return TransitionDecision{Action: "HOLD",
			Reason: fmt.Sprintf("CONSOLIDATION day %d — MRI %.1f, adherence %.0f%%",
				eval.DaysInPhase, eval.MRIScore, eval.AdherencePct*100)}

	case "INDEPENDENCE":
		if eval.DaysInPhase >= 90 && eval.MRIScore < 40 && eval.MRISustainedDays >= 56 &&
			eval.NoRelapseDays >= 60 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "STABILITY",
				Reason: fmt.Sprintf("MRI %.1f < 40 sustained %dd, no relapse for %dd",
					eval.MRIScore, eval.MRISustainedDays, eval.NoRelapseDays)}
		}
		if eval.DaysInPhase >= 120 {
			return TransitionDecision{Action: "ESCALATE",
				Reason: "INDEPENDENCE exceeded 120 days — clinical review"}
		}
		return TransitionDecision{Action: "HOLD",
			Reason: fmt.Sprintf("INDEPENDENCE day %d — MRI %.1f, relapse-free %dd",
				eval.DaysInPhase, eval.MRIScore, eval.NoRelapseDays)}

	case "STABILITY":
		if eval.YearReviewComplete && eval.HbA1cAtTarget && eval.HbA1cAtTargetReadings >= 2 &&
			eval.PhysicianGradApproval {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "PARTNERSHIP",
				Reason: "Year 1 review complete, HbA1c at target, physician approved graduation to Partnership"}
		}
		return TransitionDecision{Action: "HOLD",
			Reason: fmt.Sprintf("STABILITY day %d — awaiting year-review + physician approval",
				eval.DaysInPhase)}

	case "PARTNERSHIP":
		return TransitionDecision{Action: "HOLD",
			Reason: fmt.Sprintf("PARTNERSHIP day %d — lifelong maintenance", eval.DaysInPhase)}
	}

	return TransitionDecision{Action: "HOLD", Reason: "unknown M3-MAINTAIN phase"}
}

// EvaluateRECORRECTIONTransition applies the 45-day abbreviated re-correction cycle.
// Uses prior calibration from KB-26 (passed via MRIScore field).
func EvaluateRECORRECTIONTransition(eval TransitionEvaluation) TransitionDecision {
	switch eval.CurrentPhase {
	case "ASSESSMENT":
		if eval.DaysInPhase >= 3 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "CORRECTION",
				Reason: "assessment complete — entering 45-day re-correction with prior calibration"}
		}
		return TransitionDecision{Action: "HOLD", Reason: "ASSESSMENT in progress"}
	case "CORRECTION":
		if eval.MRIScore < 50 && eval.MRISustainedDays >= 14 {
			return TransitionDecision{Action: "ADVANCE", NextPhase: "GRADUATED",
				Reason: fmt.Sprintf("MRI %.1f < 50 sustained %dd — re-correction complete, return to MAINTAIN",
					eval.MRIScore, eval.MRISustainedDays)}
		}
		if eval.DaysInPhase >= 60 {
			return TransitionDecision{Action: "ESCALATE",
				Reason: "CORRECTION exceeded 60 days — clinical review required"}
		}
		return TransitionDecision{Action: "HOLD",
			Reason: fmt.Sprintf("CORRECTION day %d — MRI %.1f", eval.DaysInPhase, eval.MRIScore)}
	}
	return TransitionDecision{Action: "HOLD", Reason: "unknown M3-RECORRECTION phase"}
}
