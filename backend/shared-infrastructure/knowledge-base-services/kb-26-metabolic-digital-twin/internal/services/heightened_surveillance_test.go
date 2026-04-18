package services

import "testing"

func TestSurveillance_DeviationThreshold_Tightened(t *testing.T) {
	hsm := NewHeightenedSurveillanceMode()

	got := hsm.GetDeviationMultiplier(true)
	if got != 0.75 {
		t.Errorf("expected deviation multiplier 0.75 during transition, got %.2f", got)
	}

	got = hsm.GetDeviationMultiplier(false)
	if got != 1.0 {
		t.Errorf("expected deviation multiplier 1.0 outside transition, got %.2f", got)
	}
}

func TestSurveillance_PAIContextBoost(t *testing.T) {
	hsm := NewHeightenedSurveillanceMode()

	got := hsm.GetPAIContextBoost(true)
	if got != 15.0 {
		t.Errorf("expected PAI boost 15.0 during transition, got %.2f", got)
	}

	got = hsm.GetPAIContextBoost(false)
	if got != 0.0 {
		t.Errorf("expected PAI boost 0.0 outside transition, got %.2f", got)
	}
}

func TestSurveillance_EngagementGap72h(t *testing.T) {
	hsm := NewHeightenedSurveillanceMode()

	got := hsm.GetEngagementGapHours(true)
	if got != 72 {
		t.Errorf("expected engagement gap 72h during transition, got %d", got)
	}

	got = hsm.GetEngagementGapHours(false)
	if got != 168 {
		t.Errorf("expected engagement gap 168h (7 days) outside transition, got %d", got)
	}
}

func TestSurveillance_EscalationAmplification(t *testing.T) {
	hsm := NewHeightenedSurveillanceMode()

	tests := []struct {
		tier     string
		active   bool
		expected string
	}{
		{"ROUTINE", true, "URGENT"},
		{"URGENT", true, "IMMEDIATE"},
		{"IMMEDIATE", true, "IMMEDIATE"},
		{"ROUTINE", false, "ROUTINE"},
		{"URGENT", false, "URGENT"},
		{"IMMEDIATE", false, "IMMEDIATE"},
	}

	for _, tc := range tests {
		got := hsm.AmplifyEscalationTier(tc.tier, tc.active)
		if got != tc.expected {
			t.Errorf("AmplifyEscalationTier(%q, %v) = %q, want %q",
				tc.tier, tc.active, got, tc.expected)
		}
	}
}

func TestSurveillance_ExitRestoresNormal(t *testing.T) {
	hsm := NewHeightenedSurveillanceMode()

	// All functions with isActiveTransition=false return standard values
	if hsm.GetDeviationMultiplier(false) != 1.0 {
		t.Error("deviation multiplier should be 1.0 outside transition")
	}
	if hsm.GetPAIContextBoost(false) != 0.0 {
		t.Error("PAI boost should be 0.0 outside transition")
	}
	if hsm.GetEngagementGapHours(false) != 168 {
		t.Error("engagement gap should be 168h outside transition")
	}
	if hsm.AmplifyEscalationTier("ROUTINE", false) != "ROUTINE" {
		t.Error("escalation should not amplify outside transition")
	}
	if hsm.AmplifyEscalationTier("URGENT", false) != "URGENT" {
		t.Error("escalation should not amplify outside transition")
	}
}
