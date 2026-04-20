package services

import (
	"strings"
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

func TestPersona_ASHAWorker_SimplifiedLanguage(t *testing.T) {
	items := []models.WorklistItem{
		{
			PatientID:     "p1",
			PrimaryReason: "Fluid overload detected — weight gain 2.5kg",
			SuggestedAction: "Call patient to assess dyspnea and peripheral edema",
		},
		{
			PatientID:     "p2",
			PrimaryReason: "Cardiorenal syndrome suspected based on eGFR decline",
			SuggestedAction: "Urgent nephrology referral recommended",
		},
	}

	persona := PersonaConfig{
		MaxItems:      10,
		Scope:         "VILLAGE",
		Actions:       []string{"VISIT_TODAY"},
		PrimaryAction: "VISIT_TODAY",
		Language:      "hi-IN",
	}

	result := ApplyPersonaFilter(items, nil, persona)

	// Fluid overload should be translated to layperson language
	if result[0].PrimaryReason == "Fluid overload detected — weight gain 2.5kg" {
		t.Error("ASHA filter should have simplified 'Fluid overload' to layperson terms")
	}
	if !strings.Contains(result[0].PrimaryReason, "water") || !strings.Contains(result[0].PrimaryReason, "legs") {
		t.Errorf("Expected layperson fluid overload text, got: %s", result[0].PrimaryReason)
	}

	// Cardiorenal should be translated
	if result[1].PrimaryReason == "Cardiorenal syndrome suspected based on eGFR decline" {
		t.Error("ASHA filter should have simplified 'cardiorenal' to layperson terms")
	}
	if !strings.Contains(result[1].PrimaryReason, "heart") || !strings.Contains(result[1].PrimaryReason, "kidney") {
		t.Errorf("Expected layperson cardiorenal text, got: %s", result[1].PrimaryReason)
	}
}

// TestASHA_PreservesContext is a regression guard against the old
// simplifyForASHA which returned the replacement phrase entirely, discarding
// surrounding clinical context like eGFR values, weight deltas, and dates.
// An ASHA worker must still see the numeric detail after translation.
func TestASHA_PreservesContext(t *testing.T) {
	cases := []struct {
		name             string
		input            string
		mustContain      []string // substrings that must appear in the output
		mustNotBeEqualTo string   // guard against original untransformed text
	}{
		{
			name:  "cardiorenal keeps eGFR numbers",
			input: "Cardiorenal syndrome suspected based on eGFR decline 45 to 32",
			mustContain: []string{
				"heart and kidney problem", // clinical term translated
				"eGFR",                     // context preserved
				"45",
				"32",
			},
			mustNotBeEqualTo: "Cardiorenal syndrome suspected based on eGFR decline 45 to 32",
		},
		{
			name:  "fluid overload keeps weight delta",
			input: "Fluid overload detected — weight gain 2.5kg in 3 days",
			mustContain: []string{
				"body holding too much water", // translated
				"2.5kg",                       // numeric context preserved
				"3 days",                      // temporal context preserved
			},
		},
		{
			name:  "therapeutic inertia keeps drug name",
			input: "Therapeutic inertia on metformin 500mg BD for 6 months",
			mustContain: []string{
				"medicine may need to be changed",
				"metformin",
				"500mg",
				"6 months",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := simplifyForASHA(tc.input)
			for _, s := range tc.mustContain {
				if !strings.Contains(got, s) {
					t.Errorf("output missing %q (context destroyed): got %q", s, got)
				}
			}
			if tc.mustNotBeEqualTo != "" && got == tc.mustNotBeEqualTo {
				t.Errorf("clinical term was not translated at all: %q", got)
			}
		})
	}
}

// TestASHA_Deterministic is a regression guard against the old map-iteration
// implementation, which produced different outputs for the same input across
// different requests (Go map iteration is randomized per-process). Every call
// with identical input must produce identical output.
func TestASHA_Deterministic(t *testing.T) {
	input := "Fluid overload with deterioration; patient in therapeutic inertia and dyspnea"
	first := simplifyForASHA(input)
	for i := 0; i < 50; i++ {
		got := simplifyForASHA(input)
		if got != first {
			t.Fatalf("non-deterministic output on iteration %d:\n  first: %q\n  got:   %q", i, first, got)
		}
	}
}

// TestASHA_CompoundBeforeBase verifies that compound phrases like "concordant
// deterioration" are translated as a unit, not as the substring "deterioration"
// preceded by an orphan "concordant". The ordered-slice design enforces this.
func TestASHA_CompoundBeforeBase(t *testing.T) {
	input := "Concordant deterioration across cardiac and renal axes"
	got := simplifyForASHA(input)
	if !strings.Contains(got, "multiple health signs getting worse together") {
		t.Errorf("compound phrase not translated: %q", got)
	}
	// The compound must have consumed its substring — there should be no bare
	// "deterioration" left over, nor a stranded "concordant".
	lower := strings.ToLower(got)
	if strings.Contains(lower, "concordant deterioration") {
		t.Errorf("compound phrase left untouched: %q", got)
	}
	if strings.Contains(lower, "concordant ") {
		t.Errorf("stranded 'concordant' prefix — base term matched instead of compound: %q", got)
	}
}

// TestASHA_UnmatchedInputPassesThrough ensures inputs with no clinical terms
// are returned verbatim (not swallowed / emptied / mangled).
func TestASHA_UnmatchedInputPassesThrough(t *testing.T) {
	input := "Patient stable, no intervention needed today."
	got := simplifyForASHA(input)
	if got != input {
		t.Errorf("unmatched input should pass through unchanged: got %q, want %q", got, input)
	}
}
