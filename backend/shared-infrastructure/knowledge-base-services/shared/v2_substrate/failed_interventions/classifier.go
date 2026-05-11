package failed_interventions

import "strings"

// ClassifyInterventionType maps a kb-32 ApplicableRule.RuleID into the
// canonical InterventionType vocabulary used by FailedInterventionRecord.
// Unknown / non-deprescribing rule IDs return ("", false), meaning the
// rule is not failed-intervention-eligible and the kb-32 override-capture
// hook will skip the FIR write.
//
// The mapping is intentionally small — only the highest-volume Phase 1
// rule families. Extending is non-breaking: callers must always check
// the boolean return.
//
// Prefix conventions (clinical informatics should review for Phase 2):
//
//	STOP_PSYCH_*      → "antipsychotic_deprescribing"
//	                    (psychotropic stop recommendations)
//	STOP_BENZO_*      → "benzodiazepine_deprescribing"
//	STOP_ANTICH_*     → "anticholinergic_deprescribing"
//	DOSE_REDUCE_*     → "dose_reduction"
//	MONITOR_*         → "" (monitor recommendations don't generate veto records)
//	ADD_*             → "" (additive recommendations aren't failed-intervention-eligible)
//
// Unknown prefixes return ("", false). Match is case-insensitive on the
// prefix.
func ClassifyInterventionType(ruleID string) (interventionType string, classified bool) {
	if ruleID == "" {
		return "", false
	}
	upper := strings.ToUpper(ruleID)
	switch {
	case strings.HasPrefix(upper, "STOP_PSYCH_"):
		return "antipsychotic_deprescribing", true
	case strings.HasPrefix(upper, "STOP_BENZO_"):
		return "benzodiazepine_deprescribing", true
	case strings.HasPrefix(upper, "STOP_ANTICH_"):
		return "anticholinergic_deprescribing", true
	case strings.HasPrefix(upper, "DOSE_REDUCE_"):
		return "dose_reduction", true
	case strings.HasPrefix(upper, "MONITOR_"),
		strings.HasPrefix(upper, "ADD_"):
		return "", false
	}
	return "", false
}
