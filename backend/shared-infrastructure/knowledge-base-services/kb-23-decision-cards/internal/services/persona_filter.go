package services

import "kb-23-decision-cards/internal/models"

// PersonaConfig defines worklist behaviour for a clinician persona.
type PersonaConfig struct {
	MaxItems      int
	Scope         string // ASSIGNED_PANEL, FACILITY, VILLAGE
	Actions       []string
	PrimaryAction string
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

	return filtered
}

// actionLabel returns a human-readable label for an action code.
func actionLabel(code string) string {
	labels := map[string]string{
		"CALL_PATIENT":      "Call Patient",
		"SCHEDULE_CLINIC":   "Schedule Clinic",
		"TELECONSULT":       "Teleconsult",
		"MEDICATION_REVIEW": "Medication Review",
		"VISIT_TODAY":       "Visit Today",
		"RECHECK_VITALS":    "Recheck Vitals",
		"ESCALATE_TO_GP":    "Escalate to GP",
		"REFERRAL":          "Referral",
		"ACKNOWLEDGE":       "Acknowledge",
		"DEFER":             "Defer",
		"DISMISS":           "Dismiss",
	}
	if l, ok := labels[code]; ok {
		return l
	}
	return code
}
