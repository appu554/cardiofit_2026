package services

import (
	"regexp"

	"kb-23-decision-cards/internal/models"
)

// PersonaConfig defines worklist behaviour for a clinician persona.
type PersonaConfig struct {
	MaxItems      int
	Scope         string // ASSIGNED_PANEL, FACILITY, VILLAGE
	Actions       []string
	PrimaryAction string
	Language      string // en-AU, en-IN, hi-IN
}

// ApplyPersonaFilter narrows a worklist to the items relevant for a persona,
// assigns action buttons, and enforces the persona's max item limit.
func ApplyPersonaFilter(items []models.WorklistItem, assignedPatientIDs []string, persona PersonaConfig) []models.WorklistItem {
	// Build lookup set for assigned patients.
	assigned := make(map[string]struct{}, len(assignedPatientIDs))
	for _, pid := range assignedPatientIDs {
		assigned[pid] = struct{}{}
	}

	// Filter by scope.
	var filtered []models.WorklistItem
	for _, item := range items {
		switch persona.Scope {
		case "ASSIGNED_PANEL":
			if _, ok := assigned[item.PatientID]; !ok {
				continue
			}
		}
		// FACILITY and VILLAGE scopes pass all items through.
		filtered = append(filtered, item)
	}

	// Truncate to MaxItems.
	if persona.MaxItems > 0 && len(filtered) > persona.MaxItems {
		filtered = filtered[:persona.MaxItems]
	}

	// Assign action buttons from persona config.
	for i := range filtered {
		var buttons []models.ActionButton
		for _, action := range persona.Actions {
			buttons = append(buttons, models.ActionButton{
				ActionCode:   action,
				DisplayLabel: actionLabel(action),
				Primary:      action == persona.PrimaryAction,
			})
		}
		filtered[i].ActionButtons = buttons
	}

	// ASHA worker language simplification: translate clinical terms
	// to layperson Hindi vocabulary so ASHA workers can act on items
	// without parsing clinical terminology.
	if persona.Language == "hi-IN" {
		for i := range filtered {
			filtered[i].PrimaryReason = simplifyForASHA(filtered[i].PrimaryReason)
			filtered[i].SuggestedAction = simplifyForASHA(filtered[i].SuggestedAction)
		}
	}

	return filtered
}

// actionLabel returns a human-readable label for an action code.
func actionLabel(code string) string {
	labels := map[string]string{
		"CALL_PATIENT":        "Call Patient",
		"SCHEDULE_CLINIC":     "Schedule Clinic",
		"TELECONSULT":         "Teleconsult",
		"MEDICATION_REVIEW":   "Medication Review",
		"VISIT_TODAY":         "Visit Today",
		"VISIT_TOMORROW":      "Visit Tomorrow",
		"RECHECK_VITALS":      "Recheck Vitals",
		"ESCALATE_TO_GP":      "Escalate to GP",
		"CALL_GP":             "Call GP",
		"CALL_ANM":            "Call ANM",
		"RECORD_VITALS":       "Record Vitals",
		"MEDICATION_HOLD":     "Hold Medication",
		"HANDOVER_NOTE":       "Handover Note",
		"ASHA_OUTREACH":       "ASHA Outreach",
		"PRESCRIPTION_REVIEW": "Prescription Review",
		"SCHEDULE_APPOINTMENT": "Schedule Appointment",
		"TELEHEALTH":          "Telehealth",
		"REFERRAL":            "Referral",
		"ACKNOWLEDGE":         "Acknowledge",
		"DEFER":               "Defer",
		"DISMISS":             "Dismiss",
	}
	if l, ok := labels[code]; ok {
		return l
	}
	return code
}

// ashaReplacement pairs a compiled case-insensitive pattern for a clinical
// term with the layperson phrase that should substitute for it.
type ashaReplacement struct {
	pattern   *regexp.Regexp
	layperson string
}

// ashaSimplifications maps clinical terminology to layperson equivalents for
// ASHA worker consumption. Ordering matters: compound / longer-phrase entries
// come before their substrings so the compound phrase wins (e.g. "concordant
// deterioration" is translated as a single unit before the bare word
// "deterioration" gets its own substitution). This is a stable, deterministic
// ordering — do NOT convert to a map.
//
// The replacement uses regexp.ReplaceAllString rather than returning the
// layperson text directly, so surrounding context (numeric values like eGFR,
// weight deltas, dates, patient names) is preserved verbatim.
var ashaSimplifications = buildAshaSimplifications([]struct{ clinical, layperson string }{
	// Compound / multi-word phrases first (longest-match wins).
	{"concordant deterioration", "multiple health signs getting worse together"},
	{"severe hypoglycaemia", "very low sugar — give sweet drink or food immediately"},
	{"lower extremity edema", "swollen legs"},
	{"hypertensive emergency", "very high blood pressure — needs immediate doctor attention"},
	{"acute kidney injury", "kidney problem — patient may need more water or medicine change"},
	{"therapeutic inertia", "medicine may need to be changed — discuss with doctor"},
	{"phenotype transition", "health pattern has changed — discuss with doctor"},
	{"cardiorenal syndrome", "heart and kidney problem"},
	{"heart-kidney strain", "heart and kidney problem — check for swollen legs"},
	{"medication reaction", "new medicine may be causing problems"},
	{"peripheral edema", "swollen legs"},
	{"measurement gap", "no readings received — visit to check on them"},
	{"fluid overload", "body holding too much water — check for swollen legs and breathing difficulty"},
	{"post-hospital", "recently came home from hospital — needs extra attention"},
	// Shorter / single-word terms last.
	{"deterioration", "health getting worse — visit today"},
	{"cardiorenal", "heart and kidney problem"},
	{"engagement", "patient stopped measuring — visit to check on them"},
	{"declining", "health getting worse — visit soon"},
	{"dyspnea", "breathing problem"},
})

// buildAshaSimplifications precompiles each clinical term into a case-insensitive
// regex pattern. regexp.QuoteMeta escapes regex metacharacters so entries like
// "heart-kidney strain" match literally.
func buildAshaSimplifications(pairs []struct{ clinical, layperson string }) []ashaReplacement {
	out := make([]ashaReplacement, len(pairs))
	for i, p := range pairs {
		out[i] = ashaReplacement{
			pattern:   regexp.MustCompile("(?i)" + regexp.QuoteMeta(p.clinical)),
			layperson: p.layperson,
		}
	}
	return out
}

// simplifyForASHA rewrites clinical terminology in text with ASHA-friendly
// phrasing, preserving all surrounding context (numbers, patient identifiers,
// units). Substitutions are applied in the deterministic order defined by
// ashaSimplifications; compound phrases are handled before their substrings.
//
// Known limitation: the output is English ASHA-friendly vocabulary, not Hindi
// Devanagari. Full multi-language rendering is deferred — PersonaConfig's
// Language field ("hi-IN") currently labels intent; a follow-up should either
// emit Devanagari replacements or the label should be narrowed to reflect
// what's actually produced.
func simplifyForASHA(text string) string {
	for _, r := range ashaSimplifications {
		text = r.pattern.ReplaceAllString(text, r.layperson)
	}
	return text
}
