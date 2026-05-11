// Override substrate shapes — copy of kb-32 overrides.OverrideReason +
// the dual-vocabulary 20-code taxonomy (12 Wright/McCoy + 8 ACOP).
//
// SOURCE OF TRUTH: backend/shared-infrastructure/knowledge-base-services/
// kb-32-recommendation-craft/internal/overrides/taxonomy.go.
package substrate_types

import "time"

// OverrideReason mirrors kb-32 overrides.OverrideReason from Phase 2-
// completion Task 5 (dual-vocab). Note ID and RecommendationID are
// string-typed in kb-32; we preserve that exactly to keep the pin honest.
type OverrideReason struct {
	ID                  string
	RecommendationID    string
	ReasonCode          string // snake_case (application-primary)
	ReasonCodeShort     string // 3-letter (Guidelines Part 5)
	AppropriatenessFlag string // "appropriate_override" | "inappropriate_override" | "mixed"
	Reasoning           string
	CapturedAt          time.Time
	CapturedBy          string
}

// OverrideReasonCodePair is a single row of the dual-vocab mapping —
// kept as a struct rather than parallel slices so a single pin test
// asserts the entire mapping in lockstep.
type OverrideReasonCodePair struct {
	Snake string
	Short string
}

// CanonicalOverrideReasonCodes is the canonical ordered list of all 20
// dual-vocab pairs. Ordering matches kb-32 ValidReasonCodes /
// ValidShortCodes index-for-index: Wright/McCoy foundation (12) first,
// then ACOP extension (8).
var CanonicalOverrideReasonCodes = []OverrideReasonCodePair{
	// Wright/McCoy foundation (12)
	{"alert_fatigue", "ALF"},
	{"irrelevant_to_patient", "IRP"},
	{"patient_preference", "PPF"},
	{"clinical_judgment", "CJG"},
	{"alternative_pursued", "AAP"},
	{"monitoring_in_place", "MIP"},
	{"low_priority", "LPR"},
	{"documentation_concern", "DCN"},
	{"uncertain_evidence", "UNE"},
	{"system_error", "SYS"},
	{"workflow_constraint", "WFC"},
	{"duplicative_alert", "DPA"},
	// ACOP extension (8)
	{"goals_of_care_aligned", "GCA"},
	{"deprescribing_underway", "DUW"},
	{"frailty_consideration", "FRC"},
	{"family_consensus_pending", "FCP"},
	{"sdm_review_required", "SDR"},
	{"trial_period_active", "TPA"},
	{"audit_visit_imminent", "AVI"},
	{"cross_resident_pattern", "CRP"},
}
