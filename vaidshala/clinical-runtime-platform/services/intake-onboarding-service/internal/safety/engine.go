package safety

import "github.com/cardiofit/intake-onboarding-service/internal/slots"

// Engine evaluates all safety rules against current slot values.
// Deterministic, <5ms, zero external dependencies.
type Engine struct {
	hardStopRules []RuleFunc
	softFlagRules []RuleFunc
}

// NewEngine creates a safety engine with all registered rules.
func NewEngine() *Engine {
	return &Engine{
		hardStopRules: []RuleFunc{
			CheckH1TypeOneDM,
			CheckH2Pregnancy,
			CheckH3Dialysis,
			CheckH4ActiveCancer,
			CheckH5EGFRCritical,
			CheckH6RecentMIStroke,
			CheckH7HeartFailureSevere,
			CheckH8Child,
			CheckH9BariatricSurgery,
			CheckH10OrganTransplant,
			CheckH11SubstanceAbuse,
		},
		softFlagRules: []RuleFunc{
			CheckSF01Elderly,
			CheckSF02CKDModerate,
			CheckSF03Polypharmacy,
			CheckSF04LowBMI,
			CheckSF05InsulinUse,
			CheckSF06FallsRisk,
			CheckSF07CognitiveImpairment,
			CheckSF08NonAdherent,
		},
	}
}

// Evaluate runs all safety rules against the given slot snapshot.
// Returns collected HARD_STOPs and SOFT_FLAGs. Never returns an error --
// missing slot values simply cause the rule to not trigger (safe default).
func (e *Engine) Evaluate(snap slots.SlotSnapshot) SafetyResult {
	result := SafetyResult{
		HardStops: make([]RuleResult, 0),
		SoftFlags: make([]RuleResult, 0),
	}

	for _, rule := range e.hardStopRules {
		if triggered, ruleID, reason := rule(snap); triggered {
			result.HardStops = append(result.HardStops, RuleResult{
				RuleID:   ruleID,
				RuleType: RuleTypeHardStop,
				Reason:   reason,
			})
		}
	}

	for _, rule := range e.softFlagRules {
		if triggered, ruleID, reason := rule(snap); triggered {
			result.SoftFlags = append(result.SoftFlags, RuleResult{
				RuleID:   ruleID,
				RuleType: RuleTypeSoftFlag,
				Reason:   reason,
			})
		}
	}

	return result
}

// EvaluateForSlot runs the safety engine specifically after a slot fill.
// Returns the SafetyResult and whether the enrollment should be HARD_STOPPED.
func (e *Engine) EvaluateForSlot(snap slots.SlotSnapshot, slotName string) (SafetyResult, bool) {
	result := e.Evaluate(snap)
	return result, result.HasHardStop()
}
