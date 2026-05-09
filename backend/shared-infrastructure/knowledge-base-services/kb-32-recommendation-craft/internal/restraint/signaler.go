// Package restraint implements the nine substrate-signal detectors described
// in Guidelines Part 10.  Each detector examines a ClinicalSnapshot and emits
// a Signal when its trigger condition is met.
//
// VisibilityClass: AD — restraint signals per Guidelines §10
//
// Restraint surfaces "stop the line" context inline with action, fulfilling
// Principle 4 (Restraint as a clinical answer).  This file is the data layer;
// UI rendering is deferred to Phase 4 Layer 4 surfaces.
package restraint

import kb32ctx "github.com/cardiofit/kb32/internal/context"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// Severity classifies the urgency of a restraint signal.
type Severity = string

const (
	// SeverityRed indicates an immediate clinical stop-the-line condition.
	SeverityRed Severity = "red"
	// SeverityAmber indicates an elevated-risk condition warranting review.
	SeverityAmber Severity = "amber"
)

// Signal is the output of a single detector: a typed, severity-rated
// annotation that can be surfaced inline with a clinical recommendation.
type Signal struct {
	// Type is the canonical detector identifier (e.g. "recent_fall_72h").
	Type string
	// Severity is either SeverityRed or SeverityAmber.
	Severity Severity
	// Reasoning is a short human-readable explanation of why the signal fired.
	Reasoning string
	// SuggestedPause is an optional, clinician-facing prompt for what to
	// consider before proceeding.
	SuggestedPause string
}

// Detector is a function that inspects a ClinicalSnapshot and returns a
// Signal if the trigger condition is met, or nil otherwise.
type Detector func(snap kb32ctx.ClinicalSnapshot) *Signal

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

// detectorsByName registers the 9 canonical restraint detectors.  Each key is
// the authoritative signal type string; the structural test
// TestExpectedSignalTypesAllRegistered asserts that all 9 are present.
var detectorsByName = map[string]Detector{
	"recent_fall_72h":             detectRecentFall,
	"acb_increase":                detectACBIncrease,
	"family_distress":             detectFamilyDistress,
	"end_of_life_proximity":       detectEndOfLifeProximity,
	"capacity_lapse":              detectCapacityLapse,
	"polypharmacy_threshold":      detectPolypharmacy,
	"frailty_step_change":         detectFrailtyStepChange,
	"recent_admission_72h":        detectRecentAdmission,
	"restrictive_practice_active": detectRestrictivePractice,
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// DetectAll runs every registered detector against snap and returns all
// Signals that fired.  The returned slice is empty (not nil) when no signals
// are triggered.
func DetectAll(snap kb32ctx.ClinicalSnapshot) []Signal {
	out := []Signal{}
	for _, d := range detectorsByName {
		if s := d(snap); s != nil {
			out = append(out, *s)
		}
	}
	return out
}

// DetectorRegistered reports whether name appears as a key in detectorsByName.
// Used by the structural test to assert all 9 canonical names are present.
func DetectorRegistered(name string) bool {
	_, ok := detectorsByName[name]
	return ok
}

// IsValidSeverity reports whether s is one of the two accepted severity
// strings ("red" or "amber").  The check is case-sensitive.
func IsValidSeverity(s string) bool {
	return s == SeverityRed || s == SeverityAmber
}

// ---------------------------------------------------------------------------
// Detector #1 — recent_fall_72h
// Trigger: RecentFall72h == true
// Severity: Red
// ---------------------------------------------------------------------------

func detectRecentFall(snap kb32ctx.ClinicalSnapshot) *Signal {
	if !snap.RecentFall72h {
		return nil
	}
	return &Signal{
		Type:           "recent_fall_72h",
		Severity:       SeverityRed,
		Reasoning:      "Resident fell within 72h",
		SuggestedPause: "Review fall mechanism and contributing medications before changing any prescriptions.",
	}
}

// ---------------------------------------------------------------------------
// Detector #2 — acb_increase
// Trigger: ACB ≥ 3
// Severity: Amber
// ---------------------------------------------------------------------------

func detectACBIncrease(snap kb32ctx.ClinicalSnapshot) *Signal {
	if snap.ACB < 3 {
		return nil
	}
	return &Signal{
		Type:           "acb_increase",
		Severity:       SeverityAmber,
		Reasoning:      "Anticholinergic burden elevated (ACB ≥ 3)",
		SuggestedPause: "Consider anticholinergic load before adding further CNS-active agents.",
	}
}

// ---------------------------------------------------------------------------
// Detector #3 — family_distress
// Trigger: FamilyDistress == true (substrate signal via SDM/family complaint patterns)
// Severity: Red
// ---------------------------------------------------------------------------

func detectFamilyDistress(snap kb32ctx.ClinicalSnapshot) *Signal {
	if !snap.FamilyDistress {
		return nil
	}
	return &Signal{
		Type:           "family_distress",
		Severity:       SeverityRed,
		Reasoning:      "Family or SDM distress signal active",
		SuggestedPause: "Consult with family or substitute decision-maker before proceeding.",
	}
}

// ---------------------------------------------------------------------------
// Detector #4 — end_of_life_proximity
// Trigger: CareIntensity == "end_of_life"
// Severity: Red
// ---------------------------------------------------------------------------

func detectEndOfLifeProximity(snap kb32ctx.ClinicalSnapshot) *Signal {
	if snap.CareIntensity != "end_of_life" {
		return nil
	}
	return &Signal{
		Type:           "end_of_life_proximity",
		Severity:       SeverityRed,
		Reasoning:      "Resident in end-of-life care",
		SuggestedPause: "Align all recommendations with comfort-focused goals of care.",
	}
}

// ---------------------------------------------------------------------------
// Detector #5 — capacity_lapse
// Trigger: CapacityLapse == true (substrate signal from capacity assessment)
// Severity: Red
// ---------------------------------------------------------------------------

func detectCapacityLapse(snap kb32ctx.ClinicalSnapshot) *Signal {
	if !snap.CapacityLapse {
		return nil
	}
	return &Signal{
		Type:           "capacity_lapse",
		Severity:       SeverityRed,
		Reasoning:      "Recent capacity assessment indicates cognitive decline",
		SuggestedPause: "Confirm substitute decision-maker involvement before prescribing changes.",
	}
}

// ---------------------------------------------------------------------------
// Detector #6 — polypharmacy_threshold
// Trigger: DBI ≥ 1.0
// Severity: Amber
// ---------------------------------------------------------------------------

func detectPolypharmacy(snap kb32ctx.ClinicalSnapshot) *Signal {
	if snap.DBI < 1.0 {
		return nil
	}
	return &Signal{
		Type:           "polypharmacy_threshold",
		Severity:       SeverityAmber,
		Reasoning:      "Drug Burden Index ≥ 1.0 (polypharmacy)",
		SuggestedPause: "Review total drug burden before adding or adjusting sedative/anticholinergic agents.",
	}
}

// ---------------------------------------------------------------------------
// Detector #7 — frailty_step_change
// Trigger: FrailtyStepIncrease30d == true (CFS increased ≥ 2 steps in 30 days)
// Severity: Amber
// ---------------------------------------------------------------------------

func detectFrailtyStepChange(snap kb32ctx.ClinicalSnapshot) *Signal {
	if !snap.FrailtyStepIncrease30d {
		return nil
	}
	return &Signal{
		Type:           "frailty_step_change",
		Severity:       SeverityAmber,
		Reasoning:      "Frailty score increased ≥ 2 steps in past 30 days",
		SuggestedPause: "Reassess medication appropriateness in light of rapid functional decline.",
	}
}

// ---------------------------------------------------------------------------
// Detector #8 — recent_admission_72h
// Trigger: RecentAdmission72h == true
// Severity: Red
// ---------------------------------------------------------------------------

func detectRecentAdmission(snap kb32ctx.ClinicalSnapshot) *Signal {
	if !snap.RecentAdmission72h {
		return nil
	}
	return &Signal{
		Type:           "recent_admission_72h",
		Severity:       SeverityRed,
		Reasoning:      "Resident admitted within 72h",
		SuggestedPause: "Ensure hospital discharge medications are reconciled before any new prescribing.",
	}
}

// ---------------------------------------------------------------------------
// Detector #9 — restrictive_practice_active
// Trigger: RestrictivePracticeActive == true (active restrictive practice consent)
// Severity: Red
// ---------------------------------------------------------------------------

func detectRestrictivePractice(snap kb32ctx.ClinicalSnapshot) *Signal {
	if !snap.RestrictivePracticeActive {
		return nil
	}
	return &Signal{
		Type:           "restrictive_practice_active",
		Severity:       SeverityRed,
		Reasoning:      "Active restrictive practice consent recorded",
		SuggestedPause: "Review restrictive practice documentation and SDM consent before proceeding.",
	}
}
