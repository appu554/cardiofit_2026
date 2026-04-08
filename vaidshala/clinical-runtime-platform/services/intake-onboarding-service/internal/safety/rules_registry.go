package safety

// RuleType identifies whether a rule is a HARD_STOP or SOFT_FLAG.
type RuleType string

const (
	RuleTypeHardStop RuleType = "HARD_STOP"
	RuleTypeSoftFlag RuleType = "SOFT_FLAG"
)

// RuleResult is the output of a single rule evaluation.
type RuleResult struct {
	RuleID   string   `json:"rule_id"`
	RuleType RuleType `json:"rule_type"`
	Reason   string   `json:"reason"`
}

// SafetyResult holds the complete result of a safety evaluation.
type SafetyResult struct {
	HardStops []RuleResult `json:"hard_stops"`
	SoftFlags []RuleResult `json:"soft_flags"`
}

// HasHardStop returns true if any HARD_STOP rule was triggered.
func (sr SafetyResult) HasHardStop() bool {
	return len(sr.HardStops) > 0
}

// HasSoftFlag returns true if any SOFT_FLAG rule was triggered.
func (sr SafetyResult) HasSoftFlag() bool {
	return len(sr.SoftFlags) > 0
}

// AllRuleIDs returns all triggered rule IDs.
func (sr SafetyResult) AllRuleIDs() []string {
	ids := make([]string, 0, len(sr.HardStops)+len(sr.SoftFlags))
	for _, r := range sr.HardStops {
		ids = append(ids, r.RuleID)
	}
	for _, r := range sr.SoftFlags {
		ids = append(ids, r.RuleID)
	}
	return ids
}
