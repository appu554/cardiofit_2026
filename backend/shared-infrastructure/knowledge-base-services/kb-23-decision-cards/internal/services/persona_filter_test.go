package services

import (
	"testing"

	"kb-23-decision-cards/internal/models"
)

func makeItems(n int, prefix string) []models.WorklistItem {
	items := make([]models.WorklistItem, n)
	for i := range items {
		items[i] = models.WorklistItem{
			PatientID: prefix + string(rune('A'+i)),
			PAIScore:  float64(50 + i),
		}
	}
	return items
}

func TestPersona_PanelScope_FiltersToAssigned(t *testing.T) {
	items := makeItems(20, "p")
	// Assign 8 patient IDs (first 8).
	assigned := make([]string, 8)
	for i := 0; i < 8; i++ {
		assigned[i] = items[i].PatientID
	}

	persona := PersonaConfig{
		MaxItems:      50,
		Scope:         "ASSIGNED_PANEL",
		Actions:       []string{"CALL_PATIENT"},
		PrimaryAction: "CALL_PATIENT",
	}

	result := ApplyPersonaFilter(items, assigned, persona)

	if len(result) != 8 {
		t.Errorf("expected 8 items for assigned panel, got %d", len(result))
	}
	// Verify all returned items are in assigned set.
	assignedSet := make(map[string]struct{})
	for _, id := range assigned {
		assignedSet[id] = struct{}{}
	}
	for _, item := range result {
		if _, ok := assignedSet[item.PatientID]; !ok {
			t.Errorf("unexpected patient %s in filtered result", item.PatientID)
		}
	}
}

func TestPersona_MaxItemsEnforced(t *testing.T) {
	items := makeItems(30, "q")

	persona := PersonaConfig{
		MaxItems:      15,
		Scope:         "FACILITY",
		Actions:       []string{"ACKNOWLEDGE"},
		PrimaryAction: "ACKNOWLEDGE",
	}

	result := ApplyPersonaFilter(items, nil, persona)

	if len(result) != 15 {
		t.Errorf("expected 15 items after max truncation, got %d", len(result))
	}
}

func TestPersona_ActionButtonsAssigned(t *testing.T) {
	items := makeItems(3, "r")

	persona := PersonaConfig{
		MaxItems:      50,
		Scope:         "FACILITY",
		Actions:       []string{"CALL_PATIENT", "SCHEDULE_CLINIC", "TELECONSULT"},
		PrimaryAction: "CALL_PATIENT",
	}

	result := ApplyPersonaFilter(items, nil, persona)

	for _, item := range result {
		if len(item.ActionButtons) != 3 {
			t.Fatalf("expected 3 action buttons, got %d", len(item.ActionButtons))
		}
		// First button should be primary.
		foundPrimary := false
		for _, btn := range item.ActionButtons {
			if btn.ActionCode == "CALL_PATIENT" {
				if !btn.Primary {
					t.Error("CALL_PATIENT should be marked primary")
				}
				foundPrimary = true
			} else {
				if btn.Primary {
					t.Errorf("%s should not be marked primary", btn.ActionCode)
				}
			}
		}
		if !foundPrimary {
			t.Error("CALL_PATIENT button not found")
		}
	}
}

func TestPersona_ASHAWorker_PrimaryAction(t *testing.T) {
	items := makeItems(2, "s")

	persona := PersonaConfig{
		MaxItems:      20,
		Scope:         "VILLAGE",
		Actions:       []string{"VISIT_TODAY", "RECHECK_VITALS", "ESCALATE_TO_GP"},
		PrimaryAction: "VISIT_TODAY",
	}

	result := ApplyPersonaFilter(items, nil, persona)

	for _, item := range result {
		foundVisit := false
		for _, btn := range item.ActionButtons {
			if btn.ActionCode == "VISIT_TODAY" {
				if !btn.Primary {
					t.Error("VISIT_TODAY should be primary for ASHA persona")
				}
				foundVisit = true
			}
		}
		if !foundVisit {
			t.Error("VISIT_TODAY button not found")
		}
	}
}
