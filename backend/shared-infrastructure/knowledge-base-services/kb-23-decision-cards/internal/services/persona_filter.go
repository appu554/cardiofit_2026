package services

import (
	"strings"

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

// clinicalToLayperson maps clinical terms to layperson Hindi-friendly
// equivalents for ASHA workers. The ASHA worker sees actionable
// instructions, not diagnostic labels.
var clinicalToLayperson = map[string]string{
	"cardiorenal":              "heart and kidney problem",
	"heart-kidney strain":      "heart and kidney problem — check for swollen legs",
	"fluid overload":           "body holding too much water — check for swollen legs and breathing difficulty",
	"hypertensive emergency":   "very high blood pressure — needs immediate doctor attention",
	"acute kidney injury":      "kidney problem — patient may need more water or medicine change",
	"severe hypoglycaemia":     "very low sugar — give sweet drink or food immediately",
	"medication reaction":      "new medicine may be causing problems",
	"post-hospital":            "recently came home from hospital — needs extra attention",
	"therapeutic inertia":      "medicine may need to be changed — discuss with doctor",
	"concordant deterioration": "multiple health signs getting worse together",
	"phenotype transition":     "health pattern has changed — discuss with doctor",
	"engagement":               "patient stopped measuring — visit to check on them",
	"measurement gap":          "no readings received — visit to check on them",
	"deterioration":            "health getting worse — visit today",
	"declining":                "health getting worse — visit soon",
}

// simplifyForASHA replaces clinical terms in text with layperson equivalents.
func simplifyForASHA(text string) string {
	lower := strings.ToLower(text)
	for clinical, simple := range clinicalToLayperson {
		if strings.Contains(lower, clinical) {
			return simple
		}
	}
	return text
}
